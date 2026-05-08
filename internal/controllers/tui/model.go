package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tuiConv "github.com/kairos-development/kairos-agent/internal/controllers/tui/conv"
	"github.com/kairos-development/kairos-agent/internal/controllers/tui/dto"
	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
)

// Model is the TUI presentation controller.
type Model struct {
	service         agentService
	strategyBuilder strategyBuilderService
	input           string
	history         []string
	lines           []string
}

// New constructs a layered TUI controller.
func New(service agentService) Model {
	return Model{service: service, lines: []string{"Kairos command center ready. Type /help to see available commands."}}
}

// NewWithStrategyBuilder constructs a TUI controller with strategy builder support.
func NewWithStrategyBuilder(service agentService, strategyBuilder strategyBuilderService) Model {
	return Model{
		service:         service,
		strategyBuilder: strategyBuilder,
		lines:           []string{"Kairos command center ready. Type /help to see available commands."},
	}
}

// Init initializes the TUI.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles TUI messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			command := strings.TrimSpace(m.input)
			m.input = ""
			if command != "" {
				m.history = append(m.history, command)
				m.lines = append(m.lines, "> "+command)
				m.handleCommand(command)
			}
		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			m.input += msg.String()
		}
	}
	return m, nil
}

func (m *Model) handleCommand(raw string) {
	parsed := tuiConv.ParseCommand(dto.CommandInput{Raw: raw})
	ctx := context.Background()
	switch parsed.Name {
	case "/help":
		helpText := "commands: /help /scan /start paper /start live /backtest /stopall /export csv /set <key> <value> /update <key> <value>"
		if m.strategyBuilder != nil {
			helpText += " /strategy <list|templates|create|update|enable|disable|delete|show>"
		}
		m.appendLine(helpText)
	case "/strategy":
		if m.strategyBuilder == nil {
			m.appendLine("strategy builder not available")
			return
		}
		m.handleStrategyCommand(ctx, parsed.Args)
	case "/scan":
		if err := m.service.StartScan(ctx); err != nil {
			m.appendLine(err.Error())
			return
		}
		m.appendLine("scanner enabled")
	case "/start":
		if len(parsed.Args) == 0 {
			m.appendLine("usage: /start paper|live")
			return
		}
		switch parsed.Args[0] {
		case "paper":
			if err := m.service.StartPaper(ctx); err != nil {
				m.appendLine(err.Error())
				return
			}
			m.appendLine("paper trading started")
		case "live":
			if err := m.service.StartLive(ctx); err != nil {
				m.appendLine(err.Error())
				return
			}
			m.appendLine("live trading started")
		default:
			m.appendLine("usage: /start paper|live")
		}
	case "/backtest":
		if err := m.service.QueueBacktest(ctx); err != nil {
			m.appendLine(err.Error())
			return
		}
		m.appendLine("backtest job queued")
	case "/stopall":
		if err := m.service.StopAll(ctx); err != nil {
			m.appendLine(err.Error())
			return
		}
		m.appendLine("runtime returned to idle")
	case "/export":
		if len(parsed.Args) == 1 && parsed.Args[0] == "csv" {
			result, err := m.service.ExportCSV(ctx, serviceagent.ExportCSVInput{})
			if err != nil {
				m.appendLine("export failed: " + err.Error())
				return
			}
			m.appendLine("csv export written to " + result.Path)
			return
		}
		m.appendLine("usage: /export csv")
	case "/set":
		if len(parsed.Args) < 2 {
			m.appendLine("usage: /set <key> <value>")
			return
		}
		key := parsed.Args[0]
		value := strings.Join(parsed.Args[1:], " ")
		if err := m.service.UpdateConfig(ctx, serviceagent.UpdateConfigInput{Key: key, Value: value}); err != nil {
			m.appendLine("set failed: " + err.Error())
			return
		}
		m.appendLine(fmt.Sprintf("config updated: %s = %s", key, value))
	case "/update":
		if len(parsed.Args) < 2 {
			m.appendLine("usage: /update <key> <value>")
			return
		}
		key := parsed.Args[0]
		value := strings.Join(parsed.Args[1:], " ")
		if err := m.service.UpdateConfig(ctx, serviceagent.UpdateConfigInput{Key: key, Value: value}); err != nil {
			m.appendLine("update failed: " + err.Error())
			return
		}
		m.appendLine(fmt.Sprintf("config updated: %s = %s", key, value))
	default:
		m.appendLine("unknown command")
	}
}

func (m *Model) appendLine(line string) {
	m.lines = append(m.lines, line)
	if len(m.lines) > 12 {
		m.lines = m.lines[len(m.lines)-12:]
	}
}

// View renders the TUI presentation model.
func (m Model) View() string {
	status, err := m.service.Status(context.Background())
	if err != nil {
		return "Kairos command center unavailable: " + err.Error()
	}
	header := renderHeader(status)
	body := strings.Join(m.lines, "\n")
	footer := lipgloss.NewStyle().BorderTop(true).Render("/ " + m.input)
	return header + "\n\n" + body + "\n\n" + footer
}

func renderHeader(status entity.RuntimeStatus) string {
	header := lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("Mode: %s | Net: %s | License: %s | NTP: %s", status.Mode, status.Connectivity, status.License, status.NTPDrift),
	)
	if status.Banner != "" {
		header += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true).Render(status.Banner)
	}
	return header
}

var _ tea.Model = Model{}
