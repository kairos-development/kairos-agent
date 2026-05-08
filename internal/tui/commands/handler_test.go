package commands

import (
	"context"
	"testing"

	"github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock action service

type mockActionService struct {
	startPaperCalled    bool
	startLiveCalled     bool
	startScanCalled     bool
	stopAllCalled       bool
	queueBacktestCalled bool
	updateConfigCalled  bool
	exportCSVCalled     bool
	updateConfigKey     string
	updateConfigValue   interface{}
	exportCSVInput      agent.ExportCSVInput
	err                 error
}

func (m *mockActionService) StartPaper(ctx context.Context) error {
	m.startPaperCalled = true
	return m.err
}

func (m *mockActionService) StartLive(ctx context.Context) error {
	m.startLiveCalled = true
	return m.err
}

func (m *mockActionService) StartScan(ctx context.Context) error {
	m.startScanCalled = true
	return m.err
}

func (m *mockActionService) StopAll(ctx context.Context) error {
	m.stopAllCalled = true
	return m.err
}

func (m *mockActionService) QueueBacktest(ctx context.Context) error {
	m.queueBacktestCalled = true
	return m.err
}

func (m *mockActionService) UpdateConfig(ctx context.Context, key string, value interface{}) error {
	m.updateConfigCalled = true
	m.updateConfigKey = key
	m.updateConfigValue = value
	return m.err
}

func (m *mockActionService) ExportCSV(ctx context.Context, input agent.ExportCSVInput) (agent.ExportCSVOutput, error) {
	m.exportCSVCalled = true
	m.exportCSVInput = input
	if m.err != nil {
		return agent.ExportCSVOutput{}, m.err
	}
	return agent.ExportCSVOutput{Path: "/tmp/export.csv"}, nil
}

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCmd  string
		wantArgs []string
		wantErr  bool
	}{
		{
			name:     "simple command",
			input:    "start",
			wantCmd:  "start",
			wantArgs: []string{},
			wantErr:  false,
		},
		{
			name:     "command with slash",
			input:    "/start",
			wantCmd:  "start",
			wantArgs: []string{},
			wantErr:  false,
		},
		{
			name:     "command with args",
			input:    "start paper",
			wantCmd:  "start",
			wantArgs: []string{"paper"},
			wantErr:  false,
		},
		{
			name:     "command with multiple args",
			input:    "set max_position_size 1000.00",
			wantCmd:  "set",
			wantArgs: []string{"max_position_size", "1000.00"},
			wantErr:  false,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := Parse(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantCmd, cmd.Name)
			assert.Equal(t, tt.wantArgs, cmd.Args)
		})
	}
}

func TestHandler_Execute_Start(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantPaper bool
		wantScan  bool
		wantLive  bool
		wantErr   bool
	}{
		{
			name:      "start paper",
			args:      []string{"paper"},
			wantPaper: true,
			wantErr:   false,
		},
		{
			name:     "start scan",
			args:     []string{"scan"},
			wantScan: true,
			wantErr:  false,
		},
		{
			name:     "start live",
			args:     []string{"live"},
			wantLive: true,
			wantErr:  false,
		},
		{
			name:    "start no args",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "start invalid mode",
			args:    []string{"invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mock := &mockActionService{}
			handler := NewHandler(mock)

			cmd := Command{Name: "start", Args: tt.args}
			err := handler.Execute(ctx, cmd)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantPaper, mock.startPaperCalled)
			assert.Equal(t, tt.wantScan, mock.startScanCalled)
			assert.Equal(t, tt.wantLive, mock.startLiveCalled)
		})
	}
}

func TestHandler_Execute_Stop(t *testing.T) {
	ctx := context.Background()
	mock := &mockActionService{}
	handler := NewHandler(mock)

	cmd := Command{Name: "stop", Args: []string{}}
	err := handler.Execute(ctx, cmd)

	require.NoError(t, err)
	assert.True(t, mock.stopAllCalled)
}

func TestHandler_Execute_StopAll(t *testing.T) {
	ctx := context.Background()
	mock := &mockActionService{}
	handler := NewHandler(mock)

	cmd := Command{Name: "stopall", Args: []string{}}
	err := handler.Execute(ctx, cmd)

	require.NoError(t, err)
	assert.True(t, mock.stopAllCalled)
}

func TestHandler_Execute_Backtest(t *testing.T) {
	ctx := context.Background()
	mock := &mockActionService{}
	handler := NewHandler(mock)

	cmd := Command{Name: "backtest", Args: []string{}}
	err := handler.Execute(ctx, cmd)

	require.NoError(t, err)
	assert.True(t, mock.queueBacktestCalled)
}

func TestHandler_Execute_Set(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantKey   string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "set with key and value",
			args:      []string{"max_position_size", "1000.00"},
			wantKey:   "max_position_size",
			wantValue: "1000.00",
			wantErr:   false,
		},
		{
			name:      "set with multiple value words",
			args:      []string{"description", "Test", "Strategy"},
			wantKey:   "description",
			wantValue: "Test Strategy",
			wantErr:   false,
		},
		{
			name:    "set with no args",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "set with only key",
			args:    []string{"max_position_size"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mock := &mockActionService{}
			handler := NewHandler(mock)

			cmd := Command{Name: "set", Args: tt.args}
			err := handler.Execute(ctx, cmd)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.True(t, mock.updateConfigCalled)
			assert.Equal(t, tt.wantKey, mock.updateConfigKey)
			assert.Equal(t, tt.wantValue, mock.updateConfigValue)
		})
	}
}

func TestHandler_Execute_Export(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
		wantErr  bool
	}{
		{
			name:     "export with path",
			args:     []string{"/tmp/export.csv"},
			wantPath: "/tmp/export.csv",
			wantErr:  false,
		},
		{
			name:    "export without path",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mock := &mockActionService{}
			handler := NewHandler(mock)

			cmd := Command{Name: "export", Args: tt.args}
			err := handler.Execute(ctx, cmd)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.True(t, mock.exportCSVCalled)
			assert.Equal(t, tt.wantPath, mock.exportCSVInput.DestinationPath)
		})
	}
}

func TestHandler_Execute_UnknownCommand(t *testing.T) {
	ctx := context.Background()
	mock := &mockActionService{}
	handler := NewHandler(mock)

	cmd := Command{Name: "unknown", Args: []string{}}
	err := handler.Execute(ctx, cmd)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestHandler_Execute_Error(t *testing.T) {
	ctx := context.Background()
	mock := &mockActionService{err: assert.AnError}
	handler := NewHandler(mock)

	cmd := Command{Name: "stop", Args: []string{}}
	err := handler.Execute(ctx, cmd)

	assert.Error(t, err)
	assert.True(t, mock.stopAllCalled)
}
