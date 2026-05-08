package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/tui/actions"
	"github.com/kairos-development/kairos-agent/internal/tui/commands"
	"github.com/kairos-development/kairos-agent/internal/tui/components"
	"github.com/kairos-development/kairos-agent/internal/tui/query"
	"github.com/kairos-development/kairos-agent/internal/tui/styles"
	"github.com/kairos-development/kairos-agent/internal/tui/viewmodel"
	"github.com/shopspring/decimal"
)

// ViewType identifies the active view.
type ViewType int

const (
	ViewDashboard ViewType = iota
	ViewMarket
	ViewPositions
	ViewBalance
	ViewLogs
	ViewSettings
)

// Model is the main TUI model.
type Model struct {
	queries    query.QueryService
	actions    actions.ActionService
	cmdHandler *commands.Handler

	statusBar      *components.StatusBar
	logViewer      *components.LogViewer
	commandInput   *components.CommandInput
	tabBar         *components.TabBar
	dashboard      *components.Dashboard
	positionsTable *components.PositionsTable
	balanceTable   *components.BalanceTable
	marketView     *components.MarketView
	settingsView   *components.SettingsView

	currentView  ViewType
	activeSymbol string

	width  int
	height int
	ready  bool

	startTime time.Time
	ping      time.Duration
	ntpDrift  time.Duration
	pnl       decimal.Decimal

	ctx context.Context
}

// New creates a new TUI model.
func New(ctx context.Context, queries query.QueryService, actions actions.ActionService) *Model {
	cmdHandler := commands.NewHandler(actions)

	return &Model{
		queries:        queries,
		actions:        actions,
		cmdHandler:     cmdHandler,
		statusBar:      components.NewStatusBar(),
		logViewer:      components.NewLogViewer(1000),
		commandInput:   components.NewCommandInput(),
		tabBar:         components.NewTabBar(),
		dashboard:      components.NewDashboard(),
		positionsTable: components.NewPositionsTable(),
		balanceTable:   components.NewBalanceTable(),
		marketView:     components.NewMarketView(),
		settingsView:   components.NewSettingsView(),
		currentView:    ViewDashboard,
		activeSymbol:   "BTCUSDT",
		startTime:      time.Now().UTC(),
		ctx:            ctx,
	}
}

// Init initializes the TUI.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.commandInput.Focus(),
		tickCmd(),
		m.subscribeToEvents(),
		loadDashboardCmd(m.ctx, m.queries),
		loadPositionsCmd(m.ctx, m.queries),
		loadBalanceCmd(m.ctx, m.queries),
		loadMarketCmd(m.ctx, m.queries, m.activeSymbol),
		loadSettingsCmd(m.ctx, m.queries),
	)
}

// Update handles messages and updates the model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.updateComponentSizes()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+l":
			// Clear logs
			m.logViewer = components.NewLogViewer(1000)
			m.updateComponentSizes()
		case "pgup":
			if m.currentView == ViewLogs || m.currentView == ViewPositions {
				if m.currentView == ViewLogs {
					m.logViewer.ScrollUp()
				} else {
					m.positionsTable.ScrollUp()
				}
			}
		case "pgdown":
			if m.currentView == ViewLogs || m.currentView == ViewPositions {
				if m.currentView == ViewLogs {
					m.logViewer.ScrollDown()
				} else {
					m.positionsTable.ScrollDown()
				}
			}

		// Log filtering (only in Logs view)
		case "i":
			if m.currentView == ViewLogs {
				level := components.LogLevelInfo
				m.logViewer.SetFilterLevel(&level)
			}
		case "w":
			if m.currentView == ViewLogs {
				level := components.LogLevelWarning
				m.logViewer.SetFilterLevel(&level)
			}
		case "e":
			if m.currentView == ViewLogs {
				level := components.LogLevelError
				m.logViewer.SetFilterLevel(&level)
			}
		case "a":
			if m.currentView == ViewLogs {
				m.logViewer.ClearFilters()
			}
		case "t":
			if m.currentView == ViewLogs {
				m.logViewer.ToggleTimestamp()
			}

		// View switching
		case "f1":
			m.currentView = ViewDashboard
			m.tabBar.SetActiveTab(0)
		case "f2":
			m.currentView = ViewMarket
			m.tabBar.SetActiveTab(1)
		case "f3":
			m.currentView = ViewPositions
			m.tabBar.SetActiveTab(2)
		case "f4":
			m.currentView = ViewBalance
			m.tabBar.SetActiveTab(3)
		case "f5":
			m.currentView = ViewLogs
			m.tabBar.SetActiveTab(4)
		case "f6":
			m.currentView = ViewSettings
			m.tabBar.SetActiveTab(5)
		case "ctrl+right":
			m.cycleViewForward()
		case "ctrl+left":
			m.cycleViewBackward()
		case "left":
			if m.currentView == ViewMarket {
				m.cyclePreviousSymbol()
				cmds = append(cmds, loadMarketCmd(m.ctx, m.queries, m.activeSymbol))
			}
		case "right":
			if m.currentView == ViewMarket {
				m.cycleNextSymbol()
				cmds = append(cmds, loadMarketCmd(m.ctx, m.queries, m.activeSymbol))
			}
		}

	case components.CommandSubmittedMsg:
		// Parse and execute command
		cmd, err := normalizeSubmittedCommand(msg.Command)
		if err != nil {
			m.logViewer.AddLog(components.LogLevelError, "Parse error: "+err.Error())
		} else {
			m.logViewer.AddLog(components.LogLevelInfo, "Executing: "+cmd.Name)
			// Execute command asynchronously
			cmds = append(cmds, executeCommandCmd(m.ctx, m.cmdHandler, cmd))
		}

	case commandResultMsg:
		if msg.err != nil {
			m.logViewer.AddLog(components.LogLevelError, "Error: "+msg.err.Error())
		} else {
			m.logViewer.AddLog(components.LogLevelSuccess, "Success")
		}

	case dashboardLoadedMsg:
		m.dashboard.Update(msg.vm)

	case positionsLoadedMsg:
		m.positionsTable.Update(msg.vm)

	case balanceLoadedMsg:
		m.balanceTable.Update(msg.vm)

	case marketLoadedMsg:
		m.marketView.Update(msg.vm)

	case settingsLoadedMsg:
		m.settingsView.Update(msg.vm)

	// Domain event messages
	case orderEventMsg:
		m.logViewer.AddLog(msg.level, msg.message)
		// Trigger refresh for dashboard (active orders)
		cmds = append(cmds, loadDashboardCmd(m.ctx, m.queries))

	case positionEventMsg:
		m.logViewer.AddLog(msg.level, msg.message)
		// Trigger refresh for positions and dashboard
		cmds = append(cmds, loadPositionsCmd(m.ctx, m.queries), loadDashboardCmd(m.ctx, m.queries))

	case balanceEventMsg:
		m.logViewer.AddLog(msg.level, msg.message)
		// Trigger refresh for balance
		cmds = append(cmds, loadBalanceCmd(m.ctx, m.queries))

	case riskEventMsg:
		m.logViewer.AddLog(msg.level, msg.message)

	case strategyEventMsg:
		m.logViewer.AddLog(msg.level, msg.message)
		// Trigger refresh for dashboard (active strategies)
		cmds = append(cmds, loadDashboardCmd(m.ctx, m.queries))

	case tickMsg:
		// Update metrics
		m.updateMetrics()
		// Reload dashboard, positions, and balance data periodically
		cmds = append(cmds, tickCmd(), loadDashboardCmd(m.ctx, m.queries), loadPositionsCmd(m.ctx, m.queries), loadBalanceCmd(m.ctx, m.queries))

	case engineEventMsg:
		m.handleEngineEvent(msg)
	}

	// Update command input
	var cmd tea.Cmd
	cmd = m.commandInput.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the TUI.
func (m *Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Render status bar
	statusBar := m.statusBar.Render()

	// Render tab bar
	tabBar := m.tabBar.Render()

	// Render main content area based on current view
	var mainContent string
	contentHeight := m.height - 8 // Reserve space for status bar, tab bar, command input, borders

	switch m.currentView {
	case ViewDashboard:
		mainContent = m.renderDashboard(contentHeight)
	case ViewMarket:
		mainContent = m.renderMarket(contentHeight)
	case ViewPositions:
		mainContent = m.renderPositions(contentHeight)
	case ViewBalance:
		mainContent = m.renderBalance(contentHeight)
	case ViewLogs:
		m.logViewer.SetSize(m.width, contentHeight)
		mainContent = m.logViewer.Render()
	case ViewSettings:
		mainContent = m.renderSettings(contentHeight)
	default:
		mainContent = "Unknown view"
	}

	// Render command input
	commandInput := m.commandInput.View()
	commandBar := styles.BottomBarStyle.Width(m.width).Render(commandInput)

	// Combine all sections
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		statusBar,
		tabBar,
		mainContent,
		commandBar,
	)

	return styles.BaseStyle.Render(content)
}

func normalizeSubmittedCommand(input components.Command) (commands.Command, error) {
	name := strings.TrimSpace(strings.TrimPrefix(input.Name, "/"))
	if name == "" {
		return commands.Command{}, fmt.Errorf("empty command")
	}
	return commands.Command{Name: name, Args: append([]string(nil), input.Args...)}, nil
}

// updateComponentSizes updates component sizes based on terminal size.
func (m *Model) updateComponentSizes() {
	m.statusBar.SetWidth(m.width)
	m.commandInput.SetWidth(m.width)
	m.tabBar.SetWidth(m.width)
}

// cycleViewForward cycles to the next view.
func (m *Model) cycleViewForward() {
	m.currentView = (m.currentView + 1) % 6
	m.tabBar.SetActiveTab(int(m.currentView))
}

// cycleViewBackward cycles to the previous view.
func (m *Model) cycleViewBackward() {
	if m.currentView == 0 {
		m.currentView = 5
	} else {
		m.currentView--
	}
	m.tabBar.SetActiveTab(int(m.currentView))
}

// cyclePreviousSymbol cycles to the previous symbol.
func (m *Model) cyclePreviousSymbol() {
	symbols := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT", "BNBUSDT"}
	for i, sym := range symbols {
		if sym == m.activeSymbol {
			if i == 0 {
				m.activeSymbol = symbols[len(symbols)-1]
			} else {
				m.activeSymbol = symbols[i-1]
			}
			return
		}
	}
}

// cycleNextSymbol cycles to the next symbol.
func (m *Model) cycleNextSymbol() {
	symbols := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT", "BNBUSDT"}
	for i, sym := range symbols {
		if sym == m.activeSymbol {
			if i == len(symbols)-1 {
				m.activeSymbol = symbols[0]
			} else {
				m.activeSymbol = symbols[i+1]
			}
			return
		}
	}
}

// renderDashboard renders the dashboard view.
func (m *Model) renderDashboard(height int) string {
	m.dashboard.SetSize(m.width, height)
	return m.dashboard.Render()
}

// renderMarket renders the market view.
func (m *Model) renderMarket(height int) string {
	m.marketView.SetSize(m.width, height)
	return m.marketView.Render()
}

// renderPositions renders the positions view.
func (m *Model) renderPositions(height int) string {
	m.positionsTable.SetSize(m.width, height)
	return m.positionsTable.Render()
}

// renderBalance renders the balance view.
func (m *Model) renderBalance(height int) string {
	m.balanceTable.SetSize(m.width, height)
	return m.balanceTable.Render()
}

// renderSettings renders the settings view.
func (m *Model) renderSettings(height int) string {
	m.settingsView.SetSize(m.width, height)
	return m.settingsView.Render()
}

// commandResultMsg is sent when a command completes.
type commandResultMsg struct {
	err error
}

// dashboardLoadedMsg is sent when dashboard data is loaded.
type dashboardLoadedMsg struct {
	vm *viewmodel.Dashboard
}

// positionsLoadedMsg is sent when positions data is loaded.
type positionsLoadedMsg struct {
	vm *viewmodel.Positions
}

// balanceLoadedMsg is sent when balance data is loaded.
type balanceLoadedMsg struct {
	vm *viewmodel.Balance
}

// marketLoadedMsg is sent when market data is loaded.
type marketLoadedMsg struct {
	vm *viewmodel.Market
}

// settingsLoadedMsg is sent when settings data is loaded.
type settingsLoadedMsg struct {
	vm *viewmodel.Settings
}

// executeCommandCmd executes a command asynchronously.
func executeCommandCmd(ctx context.Context, handler *commands.Handler, cmd commands.Command) tea.Cmd {
	return func() tea.Msg {
		err := handler.Execute(ctx, cmd)
		return commandResultMsg{err: err}
	}
}

// loadDashboardCmd loads dashboard data asynchronously.
func loadDashboardCmd(ctx context.Context, queries query.QueryService) tea.Cmd {
	return func() tea.Msg {
		vm, err := queries.GetDashboard(ctx)
		if err != nil {
			// Log error but don't crash
			return commandResultMsg{err: err}
		}
		return dashboardLoadedMsg{vm: vm}
	}
}

// loadPositionsCmd loads positions data asynchronously.
func loadPositionsCmd(ctx context.Context, queries query.QueryService) tea.Cmd {
	return func() tea.Msg {
		vm, err := queries.GetPositions(ctx)
		if err != nil {
			return commandResultMsg{err: err}
		}
		return positionsLoadedMsg{vm: vm}
	}
}

// loadBalanceCmd loads balance data asynchronously.
func loadBalanceCmd(ctx context.Context, queries query.QueryService) tea.Cmd {
	return func() tea.Msg {
		vm, err := queries.GetBalance(ctx)
		if err != nil {
			return commandResultMsg{err: err}
		}
		return balanceLoadedMsg{vm: vm}
	}
}

// loadMarketCmd loads market data asynchronously.
func loadMarketCmd(ctx context.Context, queries query.QueryService, symbol string) tea.Cmd {
	return func() tea.Msg {
		vm, err := queries.GetMarket(ctx, symbol)
		if err != nil {
			return commandResultMsg{err: err}
		}
		return marketLoadedMsg{vm: vm}
	}
}

// loadSettingsCmd loads settings data asynchronously.
func loadSettingsCmd(ctx context.Context, queries query.QueryService) tea.Cmd {
	return func() tea.Msg {
		vm, err := queries.GetSettings(ctx)
		if err != nil {
			return commandResultMsg{err: err}
		}
		return settingsLoadedMsg{vm: vm}
	}
}

// updateMetrics updates system metrics.
func (m *Model) updateMetrics() {
	uptime := time.Since(m.startTime)

	mode := entity.RunModeIdle
	m.statusBar.Update(mode, m.ping, m.ntpDrift, m.pnl, uptime)

	m.ping = 50 * time.Millisecond
	m.ntpDrift = 10 * time.Millisecond
}

func (m *Model) handleEngineEvent(msg engineEventMsg) {
	m.logViewer.AddLog(msg.level, msg.message)
}

func (m *Model) subscribeToEvents() tea.Cmd {
	return nil
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// engineEventMsg wraps an engine event for the TUI.
type engineEventMsg struct {
	level   components.LogLevel
	message string
}
