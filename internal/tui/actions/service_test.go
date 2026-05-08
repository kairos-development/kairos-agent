package actions

import (
	"context"
	"testing"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock agent service

type mockAgentService struct {
	startPaperCalled    bool
	startLiveCalled     bool
	startScanCalled     bool
	stopAllCalled       bool
	queueBacktestCalled bool
	updateConfigCalled  bool
	exportCSVCalled     bool
	updateConfigInput   agent.UpdateConfigInput
	exportCSVInput      agent.ExportCSVInput
	err                 error
}

func (m *mockAgentService) StartPaper(ctx context.Context) error {
	m.startPaperCalled = true
	return m.err
}

func (m *mockAgentService) StartLive(ctx context.Context) error {
	m.startLiveCalled = true
	return m.err
}

func (m *mockAgentService) StartScan(ctx context.Context) error {
	m.startScanCalled = true
	return m.err
}

func (m *mockAgentService) StopAll(ctx context.Context) error {
	m.stopAllCalled = true
	return m.err
}

func (m *mockAgentService) QueueBacktest(ctx context.Context) error {
	m.queueBacktestCalled = true
	return m.err
}

func (m *mockAgentService) UpdateConfig(ctx context.Context, input agent.UpdateConfigInput) error {
	m.updateConfigCalled = true
	m.updateConfigInput = input
	return m.err
}

func (m *mockAgentService) ExportCSV(ctx context.Context, input agent.ExportCSVInput) (agent.ExportCSVOutput, error) {
	m.exportCSVCalled = true
	m.exportCSVInput = input
	if m.err != nil {
		return agent.ExportCSVOutput{}, m.err
	}
	return agent.ExportCSVOutput{Path: "/tmp/export.csv"}, nil
}

// Implement other required methods as no-ops
func (m *mockAgentService) AcceptDisclaimer(ctx context.Context, source string) error {
	return nil
}

func (m *mockAgentService) Check(ctx context.Context) (agent.CheckOutput, error) {
	return agent.CheckOutput{}, nil
}

func (m *mockAgentService) Status(ctx context.Context) (entity.RuntimeStatus, error) {
	return entity.RuntimeStatus{}, nil
}

func (m *mockAgentService) Config(ctx context.Context) (entity.AgentConfig, error) {
	return entity.AgentConfig{}, nil
}

func (m *mockAgentService) EmergencyStop(ctx context.Context) error {
	return nil
}

func (m *mockAgentService) SetConfig(ctx context.Context, input agent.SetConfigInput) error {
	return nil
}

func (m *mockAgentService) TrustPluginKey(ctx context.Context, input agent.TrustPluginKeyInput) error {
	return nil
}

func (m *mockAgentService) Entitlement(ctx context.Context) (agent.EntitlementOutput, error) {
	return agent.EntitlementOutput{}, nil
}

func (m *mockAgentService) CanUseFeature(ctx context.Context, feature string) (bool, error) {
	return true, nil
}

func TestActionService_StartPaper(t *testing.T) {
	ctx := context.Background()
	mock := &mockAgentService{}
	svc := NewActionService(mock)

	err := svc.StartPaper(ctx)

	require.NoError(t, err)
	assert.True(t, mock.startPaperCalled)
}

func TestActionService_StartPaper_Error(t *testing.T) {
	ctx := context.Background()
	mock := &mockAgentService{err: assert.AnError}
	svc := NewActionService(mock)

	err := svc.StartPaper(ctx)

	assert.Error(t, err)
	assert.True(t, mock.startPaperCalled)
}

func TestActionService_StartLive(t *testing.T) {
	ctx := context.Background()
	mock := &mockAgentService{}
	svc := NewActionService(mock)

	err := svc.StartLive(ctx)

	require.NoError(t, err)
	assert.True(t, mock.startLiveCalled)
}

func TestActionService_StartLive_Error(t *testing.T) {
	ctx := context.Background()
	mock := &mockAgentService{err: assert.AnError}
	svc := NewActionService(mock)

	err := svc.StartLive(ctx)

	assert.Error(t, err)
	assert.True(t, mock.startLiveCalled)
}

func TestActionService_StartScan(t *testing.T) {
	ctx := context.Background()
	mock := &mockAgentService{}
	svc := NewActionService(mock)

	err := svc.StartScan(ctx)

	require.NoError(t, err)
	assert.True(t, mock.startScanCalled)
}

func TestActionService_StopAll(t *testing.T) {
	ctx := context.Background()
	mock := &mockAgentService{}
	svc := NewActionService(mock)

	err := svc.StopAll(ctx)

	require.NoError(t, err)
	assert.True(t, mock.stopAllCalled)
}

func TestActionService_QueueBacktest(t *testing.T) {
	ctx := context.Background()
	mock := &mockAgentService{}
	svc := NewActionService(mock)

	err := svc.QueueBacktest(ctx)

	require.NoError(t, err)
	assert.True(t, mock.queueBacktestCalled)
}

func TestActionService_UpdateConfig(t *testing.T) {
	ctx := context.Background()
	mock := &mockAgentService{}
	svc := NewActionService(mock)

	err := svc.UpdateConfig(ctx, "max_position_size", "1000.00")

	require.NoError(t, err)
	assert.True(t, mock.updateConfigCalled)
	assert.Equal(t, "max_position_size", mock.updateConfigInput.Key)
	assert.Equal(t, "1000.00", mock.updateConfigInput.Value)
}

func TestActionService_UpdateConfig_Error(t *testing.T) {
	ctx := context.Background()
	mock := &mockAgentService{err: assert.AnError}
	svc := NewActionService(mock)

	err := svc.UpdateConfig(ctx, "max_position_size", "1000.00")

	assert.Error(t, err)
	assert.True(t, mock.updateConfigCalled)
}

func TestActionService_ExportCSV(t *testing.T) {
	ctx := context.Background()
	mock := &mockAgentService{}
	svc := NewActionService(mock)

	input := agent.ExportCSVInput{
		DestinationPath: "/tmp/test.csv",
	}

	output, err := svc.ExportCSV(ctx, input)

	require.NoError(t, err)
	assert.True(t, mock.exportCSVCalled)
	assert.Equal(t, "/tmp/test.csv", mock.exportCSVInput.DestinationPath)
	assert.Equal(t, "/tmp/export.csv", output.Path)
}

func TestActionService_ExportCSV_Error(t *testing.T) {
	ctx := context.Background()
	mock := &mockAgentService{err: assert.AnError}
	svc := NewActionService(mock)

	input := agent.ExportCSVInput{
		DestinationPath: "/tmp/test.csv",
	}

	output, err := svc.ExportCSV(ctx, input)

	assert.Error(t, err)
	assert.True(t, mock.exportCSVCalled)
	assert.Empty(t, output.Path)
}
