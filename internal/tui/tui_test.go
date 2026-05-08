package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/kairos-development/kairos-agent/internal/tui/commands"
	"github.com/kairos-development/kairos-agent/internal/tui/components"
	"github.com/kairos-development/kairos-agent/internal/tui/viewmodel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tuiQueryMock struct {
	dashboard *viewmodel.Dashboard
	positions *viewmodel.Positions
	balance   *viewmodel.Balance
	market    *viewmodel.Market
	settings  *viewmodel.Settings
	err       error
}

func (m *tuiQueryMock) GetDashboard(context.Context) (*viewmodel.Dashboard, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.dashboard, nil
}
func (m *tuiQueryMock) GetPositions(context.Context) (*viewmodel.Positions, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.positions, nil
}
func (m *tuiQueryMock) GetBalance(context.Context) (*viewmodel.Balance, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.balance, nil
}
func (m *tuiQueryMock) GetMarket(context.Context, string) (*viewmodel.Market, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.market, nil
}
func (m *tuiQueryMock) GetSettings(context.Context) (*viewmodel.Settings, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.settings, nil
}

type tuiActionMock struct {
	paper        bool
	live         bool
	scan         bool
	stopAll      bool
	backtest     bool
	updatedKey   string
	updatedValue interface{}
	exportInput  serviceagent.ExportCSVInput
	err          error
}

func (m *tuiActionMock) StartPaper(context.Context) error    { m.paper = true; return m.err }
func (m *tuiActionMock) StartLive(context.Context) error     { m.live = true; return m.err }
func (m *tuiActionMock) StartScan(context.Context) error     { m.scan = true; return m.err }
func (m *tuiActionMock) StopAll(context.Context) error       { m.stopAll = true; return m.err }
func (m *tuiActionMock) QueueBacktest(context.Context) error { m.backtest = true; return m.err }
func (m *tuiActionMock) UpdateConfig(_ context.Context, key string, value interface{}) error {
	m.updatedKey = key
	m.updatedValue = value
	return m.err
}
func (m *tuiActionMock) ExportCSV(_ context.Context, input serviceagent.ExportCSVInput) (serviceagent.ExportCSVOutput, error) {
	m.exportInput = input
	if m.err != nil {
		return serviceagent.ExportCSVOutput{}, m.err
	}
	return serviceagent.ExportCSVOutput{Path: input.DestinationPath}, nil
}

func newTestModel(t *testing.T) (*Model, *tuiQueryMock, *tuiActionMock) {
	t.Helper()
	queries := &tuiQueryMock{
		dashboard: &viewmodel.Dashboard{},
		positions: &viewmodel.Positions{},
		balance:   &viewmodel.Balance{},
		market:    &viewmodel.Market{Symbol: "BTCUSDT"},
		settings:  &viewmodel.Settings{},
	}
	actions := &tuiActionMock{}
	return New(context.Background(), queries, actions), queries, actions
}

func TestNormalizeSubmittedCommandPreservesArgs(t *testing.T) {
	cmd, err := normalizeSubmittedCommand(components.Command{Name: "/start", Args: []string{"paper", "BTCUSDT"}})
	require.NoError(t, err)
	assert.Equal(t, commands.Command{Name: "start", Args: []string{"paper", "BTCUSDT"}}, cmd)

	_, err = normalizeSubmittedCommand(components.Command{})
	assert.Error(t, err)
}

func TestModelViewSwitchingAndSizing(t *testing.T) {
	model, _, _ := newTestModel(t)
	assert.Equal(t, "Initializing...", model.View())

	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	model = updated.(*Model)
	if cmd != nil {
		assert.Nil(t, cmd())
	}
	assert.True(t, model.ready)
	assert.NotEmpty(t, model.View())

	keys := []struct {
		key  string
		view ViewType
	}{
		{"f2", ViewMarket},
		{"f3", ViewPositions},
		{"f4", ViewBalance},
		{"f5", ViewLogs},
		{"f6", ViewSettings},
		{"f1", ViewDashboard},
	}
	for _, tt := range keys {
		updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})
		model = updated.(*Model)
		assert.Equal(t, tt.view, model.currentView)
	}

	model.cycleViewBackward()
	assert.Equal(t, ViewSettings, model.currentView)
	model.cycleViewForward()
	assert.Equal(t, ViewDashboard, model.currentView)
}

func TestModelMarketSymbolCycling(t *testing.T) {
	model, _, _ := newTestModel(t)
	assert.Equal(t, "BTCUSDT", model.activeSymbol)
	model.cyclePreviousSymbol()
	assert.Equal(t, "BNBUSDT", model.activeSymbol)
	model.cycleNextSymbol()
	assert.Equal(t, "BTCUSDT", model.activeSymbol)
}

func runTeaCmd(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, child := range batch {
			out = append(out, runTeaCmd(child)...)
		}
		return out
	}
	return []tea.Msg{msg}
}

func TestModelCommandSubmittedUsesActionServiceArgs(t *testing.T) {
	model, _, actions := newTestModel(t)
	updated, cmd := model.Update(components.CommandSubmittedMsg{Command: components.Command{Name: "/start", Args: []string{"paper"}}})
	model = updated.(*Model)
	_ = runTeaCmd(cmd)
	assert.True(t, actions.paper)
	assert.False(t, actions.live)
	assert.NotNil(t, model)
}

func TestLoadCommandsReturnMessages(t *testing.T) {
	_, queries, _ := newTestModel(t)
	assert.IsType(t, dashboardLoadedMsg{}, loadDashboardCmd(context.Background(), queries)())
	assert.IsType(t, positionsLoadedMsg{}, loadPositionsCmd(context.Background(), queries)())
	assert.IsType(t, balanceLoadedMsg{}, loadBalanceCmd(context.Background(), queries)())
	assert.IsType(t, marketLoadedMsg{}, loadMarketCmd(context.Background(), queries, "BTCUSDT")())
	assert.IsType(t, settingsLoadedMsg{}, loadSettingsCmd(context.Background(), queries)())

	queries.err = assert.AnError
	assert.IsType(t, commandResultMsg{}, loadDashboardCmd(context.Background(), queries)())
}

func TestExecuteCommandCmd(t *testing.T) {
	_, _, actions := newTestModel(t)
	handler := commands.NewHandler(actions)
	msg := executeCommandCmd(context.Background(), handler, commands.Command{Name: "set", Args: []string{"max_position", "1000"}})()
	result, ok := msg.(commandResultMsg)
	require.True(t, ok)
	require.NoError(t, result.err)
	assert.Equal(t, "max_position", actions.updatedKey)
}

func TestUpdateDataMessagesAndTick(t *testing.T) {
	model, _, _ := newTestModel(t)
	now := time.Now().UTC()
	model.Update(dashboardLoadedMsg{vm: &viewmodel.Dashboard{SystemStatus: viewmodel.SystemStatus{Mode: string(entity.RunModePaperTrading)}}})
	model.Update(positionsLoadedMsg{vm: &viewmodel.Positions{TotalPnL: "+$0.00"}})
	model.Update(balanceLoadedMsg{vm: &viewmodel.Balance{TotalEquity: "$0.00"}})
	model.Update(marketLoadedMsg{vm: &viewmodel.Market{Symbol: "BTCUSDT"}})
	model.Update(settingsLoadedMsg{vm: &viewmodel.Settings{}})
	_, cmd := model.Update(tickMsg(now))
	assert.NotNil(t, cmd)
}

func TestModelInit(t *testing.T) {
	model, _, _ := newTestModel(t)
	cmd := model.Init()
	assert.NotNil(t, cmd)

	// Run the init command
	msgs := runTeaCmd(cmd)
	assert.NotEmpty(t, msgs)
}

func TestModelUpdateWithQuitMsg(t *testing.T) {
	model, _, _ := newTestModel(t)
	updated, cmd := model.Update(tea.QuitMsg{})
	assert.NotNil(t, updated)
	assert.Nil(t, cmd)
}

func TestModelUpdateWithKeyMsg(t *testing.T) {
	model, _, _ := newTestModel(t)
	model.ready = true

	// Test Ctrl+C
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.NotNil(t, updated)
	assert.NotNil(t, cmd)

	// Test Ctrl+Left
	model.currentView = ViewPositions
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlLeft})
	model = updated.(*Model)
	assert.Equal(t, ViewMarket, model.currentView)

	// Test Ctrl+Right
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlRight})
	model = updated.(*Model)
	assert.Equal(t, ViewPositions, model.currentView)
}

func TestModelUpdateWithEngineEventMsg(t *testing.T) {
	model, _, _ := newTestModel(t)
	model.ready = true
	model.width = 100
	model.height = 40
	model.logViewer.SetSize(100, 10)

	updated, _ := model.Update(engineEventMsg{level: components.LogLevelInfo, message: "test event"})
	model = updated.(*Model)
	assert.Contains(t, model.logViewer.Render(), "test event")
}

func TestModelUpdateWithCommandResultMsg(t *testing.T) {
	model, _, _ := newTestModel(t)
	model.ready = true
	model.width = 100
	model.height = 40
	model.logViewer.SetSize(100, 10)

	updated, _ := model.Update(commandResultMsg{err: nil})
	model = updated.(*Model)
	assert.Contains(t, model.logViewer.Render(), "Success")

	updated, _ = model.Update(commandResultMsg{err: assert.AnError})
	model = updated.(*Model)
	assert.Contains(t, model.logViewer.Render(), "Error")
}

func TestModelUpdateWithLoadedMessages(t *testing.T) {
	model, _, _ := newTestModel(t)
	model.ready = true

	updated, _ := model.Update(dashboardLoadedMsg{vm: &viewmodel.Dashboard{}})
	assert.NotNil(t, updated)

	updated, _ = model.Update(positionsLoadedMsg{vm: &viewmodel.Positions{}})
	assert.NotNil(t, updated)

	updated, _ = model.Update(balanceLoadedMsg{vm: &viewmodel.Balance{}})
	assert.NotNil(t, updated)

	updated, _ = model.Update(marketLoadedMsg{vm: &viewmodel.Market{Symbol: "BTCUSDT"}})
	assert.NotNil(t, updated)

	updated, _ = model.Update(settingsLoadedMsg{vm: &viewmodel.Settings{}})
	assert.NotNil(t, updated)
}

func TestModelViewRendering(t *testing.T) {
	model, _, _ := newTestModel(t)
	model.ready = true
	model.width = 100
	model.height = 40

	// Test each view renders without panic
	views := []ViewType{
		ViewDashboard,
		ViewMarket,
		ViewPositions,
		ViewBalance,
		ViewLogs,
		ViewSettings,
	}

	for _, view := range views {
		model.currentView = view
		output := model.View()
		assert.NotEmpty(t, output)
	}
}

func TestModelMarketViewSymbolNavigation(t *testing.T) {
	model, _, _ := newTestModel(t)
	model.ready = true
	model.currentView = ViewMarket

	// Test left arrow (previous symbol)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = updated.(*Model)
	assert.Equal(t, "BNBUSDT", model.activeSymbol)

	// Test right arrow (next symbol)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = updated.(*Model)
	assert.Equal(t, "BTCUSDT", model.activeSymbol)
}

func TestModelCommandInputFocus(t *testing.T) {
	model, _, _ := newTestModel(t)
	model.ready = true

	// Command input should handle key messages
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	assert.NotNil(t, updated)
}

func TestNormalizeSubmittedCommand_EmptyCommand(t *testing.T) {
	_, err := normalizeSubmittedCommand(components.Command{Name: ""})
	assert.Error(t, err)
}

func TestNormalizeSubmittedCommand_WithoutSlash(t *testing.T) {
	cmd, err := normalizeSubmittedCommand(components.Command{Name: "start", Args: []string{"paper"}})
	require.NoError(t, err)
	assert.Equal(t, "start", cmd.Name)
}

func TestModelCycleViewEdgeCases(t *testing.T) {
	model, _, _ := newTestModel(t)

	// Cycle forward from last view
	model.currentView = ViewSettings
	model.cycleViewForward()
	assert.Equal(t, ViewDashboard, model.currentView)

	// Cycle backward from first view
	model.currentView = ViewDashboard
	model.cycleViewBackward()
	assert.Equal(t, ViewSettings, model.currentView)
}

func TestModelCycleSymbolEdgeCases(t *testing.T) {
	model, _, _ := newTestModel(t)

	// Cycle to last symbol
	model.activeSymbol = "BTCUSDT"
	model.cyclePreviousSymbol()
	assert.Equal(t, "BNBUSDT", model.activeSymbol)

	// Cycle from last to first
	model.activeSymbol = "BNBUSDT"
	model.cycleNextSymbol()
	assert.Equal(t, "BTCUSDT", model.activeSymbol)
}

func TestModelUpdateBeforeReady(t *testing.T) {
	model, _, _ := newTestModel(t)
	model.ready = false

	// Key messages before ready should be ignored
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f1")})
	model = updated.(*Model)
	assert.Equal(t, ViewDashboard, model.currentView) // Should stay at default
}

func TestModelTickMsgGeneratesRefresh(t *testing.T) {
	model, _, _ := newTestModel(t)
	model.ready = true

	now := time.Now().UTC()
	updated, cmd := model.Update(tickMsg(now))
	assert.NotNil(t, updated)
	assert.NotNil(t, cmd)

	// Should generate batch of load commands
	msgs := runTeaCmd(cmd)
	assert.NotEmpty(t, msgs)
}

func TestModelWithNilQueryService(t *testing.T) {
	actions := &tuiActionMock{}

	model := New(context.Background(), nil, actions)
	assert.NotNil(t, model)
}

func TestModelWithNilActionService(t *testing.T) {
	queries := &tuiQueryMock{
		dashboard: &viewmodel.Dashboard{},
		positions: &viewmodel.Positions{},
		balance:   &viewmodel.Balance{},
		market:    &viewmodel.Market{Symbol: "BTCUSDT"},
		settings:  &viewmodel.Settings{},
	}

	model := New(context.Background(), queries, nil)
	assert.NotNil(t, model)
}

func TestModelUpdateWithLogFiltering(t *testing.T) {
	model, _, _ := newTestModel(t)
	model.ready = true
	model.currentView = ViewLogs

	// Test info filter
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	assert.NotNil(t, updated)

	// Test warning filter
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("w")})
	assert.NotNil(t, updated)

	// Test error filter
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	assert.NotNil(t, updated)

	// Test clear filters
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	assert.NotNil(t, updated)

	// Test toggle timestamp
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	assert.NotNil(t, updated)
}

func TestModelUpdateWithScrolling(t *testing.T) {
	model, _, _ := newTestModel(t)
	model.ready = true
	model.currentView = ViewLogs

	// Test page up
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	assert.NotNil(t, updated)

	// Test page down
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.NotNil(t, updated)

	// Test scrolling in positions view
	model.currentView = ViewPositions
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	assert.NotNil(t, updated)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.NotNil(t, updated)
}

func TestModelUpdateWithClearLogs(t *testing.T) {
	model, _, _ := newTestModel(t)
	model.ready = true

	// Test ctrl+l to clear logs
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	assert.NotNil(t, updated)
}
