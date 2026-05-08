package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/stretchr/testify/assert"
)

type mockAgentService struct {
	status            entity.RuntimeStatus
	statusErr         error
	startScanErr      error
	startPaperErr     error
	startLiveErr      error
	queueBacktestErr  error
	stopAllErr        error
	exportCSVOutput   serviceagent.ExportCSVOutput
	exportCSVErr      error
	setConfigErr      error
	updateConfigErr   error
	updateConfigCalls []serviceagent.UpdateConfigInput
}

func (m *mockAgentService) Status(context.Context) (entity.RuntimeStatus, error) {
	return m.status, m.statusErr
}

func (m *mockAgentService) StartScan(context.Context) error {
	return m.startScanErr
}

func (m *mockAgentService) StartPaper(context.Context) error {
	return m.startPaperErr
}

func (m *mockAgentService) StartLive(context.Context) error {
	return m.startLiveErr
}

func (m *mockAgentService) QueueBacktest(context.Context) error {
	return m.queueBacktestErr
}

func (m *mockAgentService) StopAll(context.Context) error {
	return m.stopAllErr
}

func (m *mockAgentService) ExportCSV(context.Context, serviceagent.ExportCSVInput) (serviceagent.ExportCSVOutput, error) {
	return m.exportCSVOutput, m.exportCSVErr
}

func (m *mockAgentService) SetConfig(context.Context, serviceagent.SetConfigInput) error {
	return m.setConfigErr
}

func (m *mockAgentService) UpdateConfig(_ context.Context, input serviceagent.UpdateConfigInput) error {
	m.updateConfigCalls = append(m.updateConfigCalls, input)
	return m.updateConfigErr
}

func TestModel_HandleCommand_Help(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/help")

	assert.Contains(t, model.lines[len(model.lines)-1], "commands:")
}

func TestModel_HandleCommand_Scan(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/scan")

	assert.Contains(t, model.lines[len(model.lines)-1], "scanner enabled")
}

func TestModel_HandleCommand_StartPaper(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/start paper")

	assert.Contains(t, model.lines[len(model.lines)-1], "paper trading started")
}

func TestModel_HandleCommand_StartLive(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/start live")

	assert.Contains(t, model.lines[len(model.lines)-1], "live trading started")
}

func TestModel_HandleCommand_StartInvalid(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/start invalid")

	assert.Contains(t, model.lines[len(model.lines)-1], "usage: /start paper|live")
}

func TestModel_HandleCommand_StartNoArgs(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/start")

	assert.Contains(t, model.lines[len(model.lines)-1], "usage: /start paper|live")
}

func TestModel_HandleCommand_Backtest(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/backtest")

	assert.Contains(t, model.lines[len(model.lines)-1], "backtest job queued")
}

func TestModel_HandleCommand_StopAll(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/stopall")

	assert.Contains(t, model.lines[len(model.lines)-1], "runtime returned to idle")
}

func TestModel_HandleCommand_ExportCSV(t *testing.T) {
	svc := &mockAgentService{
		exportCSVOutput: serviceagent.ExportCSVOutput{Path: "/tmp/export.csv"},
	}
	model := New(svc)

	model.handleCommand("/export csv")

	assert.Contains(t, model.lines[len(model.lines)-1], "csv export written to /tmp/export.csv")
}

func TestModel_HandleCommand_ExportInvalid(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/export invalid")

	assert.Contains(t, model.lines[len(model.lines)-1], "usage: /export csv")
}

func TestModel_HandleCommand_Set(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/set export_timezone UTC")

	assert.Len(t, svc.updateConfigCalls, 1)
	assert.Equal(t, "export_timezone", svc.updateConfigCalls[0].Key)
	assert.Equal(t, "UTC", svc.updateConfigCalls[0].Value)
	assert.Contains(t, model.lines[len(model.lines)-1], "config updated: export_timezone = UTC")
}

func TestModel_HandleCommand_SetMultipleWords(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/set risk.max_position 5000 USD")

	assert.Len(t, svc.updateConfigCalls, 1)
	assert.Equal(t, "risk.max_position", svc.updateConfigCalls[0].Key)
	assert.Equal(t, "5000 USD", svc.updateConfigCalls[0].Value)
}

func TestModel_HandleCommand_SetNoArgs(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/set")

	assert.Contains(t, model.lines[len(model.lines)-1], "usage: /set <key> <value>")
}

func TestModel_HandleCommand_SetOneArg(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/set export_timezone")

	assert.Contains(t, model.lines[len(model.lines)-1], "usage: /set <key> <value>")
}

func TestModel_HandleCommand_Update(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/update journal_max_size_bytes 67108864")

	assert.Len(t, svc.updateConfigCalls, 1)
	assert.Equal(t, "journal_max_size_bytes", svc.updateConfigCalls[0].Key)
	assert.Equal(t, "67108864", svc.updateConfigCalls[0].Value)
	assert.Contains(t, model.lines[len(model.lines)-1], "config updated: journal_max_size_bytes = 67108864")
}

func TestModel_HandleCommand_UpdateNoArgs(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/update")

	assert.Contains(t, model.lines[len(model.lines)-1], "usage: /update <key> <value>")
}

func TestModel_HandleCommand_UnknownCommand(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	model.handleCommand("/unknown")

	assert.Contains(t, model.lines[len(model.lines)-1], "unknown command")
}

func TestModel_HandleCommand_ErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		service  *mockAgentService
		command  string
		expected string
	}{
		{
			name:     "scan error",
			service:  &mockAgentService{startScanErr: assert.AnError},
			command:  "/scan",
			expected: "assert.AnError",
		},
		{
			name:     "start paper error",
			service:  &mockAgentService{startPaperErr: assert.AnError},
			command:  "/start paper",
			expected: "assert.AnError",
		},
		{
			name:     "start live error",
			service:  &mockAgentService{startLiveErr: assert.AnError},
			command:  "/start live",
			expected: "assert.AnError",
		},
		{
			name:     "backtest error",
			service:  &mockAgentService{queueBacktestErr: assert.AnError},
			command:  "/backtest",
			expected: "assert.AnError",
		},
		{
			name:     "stopall error",
			service:  &mockAgentService{stopAllErr: assert.AnError},
			command:  "/stopall",
			expected: "assert.AnError",
		},
		{
			name:     "export error",
			service:  &mockAgentService{exportCSVErr: assert.AnError},
			command:  "/export csv",
			expected: "export failed",
		},
		{
			name:     "set error",
			service:  &mockAgentService{updateConfigErr: assert.AnError},
			command:  "/set key value",
			expected: "set failed",
		},
		{
			name:     "update error",
			service:  &mockAgentService{updateConfigErr: assert.AnError},
			command:  "/update key value",
			expected: "update failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := New(tt.service)
			model.handleCommand(tt.command)
			assert.Contains(t, model.lines[len(model.lines)-1], tt.expected)
		})
	}
}

func TestModel_Update_EnterKey(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)
	model.input = "/help"

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := updatedModel.(Model)

	assert.Empty(t, m.input)
	assert.Contains(t, m.history, "/help")
	assert.Greater(t, len(m.lines), 1)
}

func TestModel_Update_Backspace(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)
	model.input = "test"

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m := updatedModel.(Model)

	assert.Equal(t, "tes", m.input)
}

func TestModel_Update_BackspaceEmpty(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)
	model.input = ""

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m := updatedModel.(Model)

	assert.Equal(t, "", m.input)
}

func TestModel_Update_CharacterInput(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m := updatedModel.(Model)

	assert.Equal(t, "a", m.input)
}

func TestModel_AppendLine_MaxLines(t *testing.T) {
	svc := &mockAgentService{}
	model := New(svc)

	for i := 0; i < 20; i++ {
		model.appendLine("line")
	}

	assert.Len(t, model.lines, 12)
}

func TestModel_View(t *testing.T) {
	svc := &mockAgentService{
		status: entity.RuntimeStatus{
			Mode:         entity.RunModeIdle,
			Connectivity: entity.ConnectivityStateConnected,
			License:      entity.LicenseStateDemo,
			NTPDrift:     0,
		},
	}
	model := New(svc)
	model.input = "test input"

	view := model.View()

	assert.Contains(t, view, "Mode: idle")
	assert.Contains(t, view, "test input")
}
