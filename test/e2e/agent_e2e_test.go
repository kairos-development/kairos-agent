package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/config"
	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	configprovider "github.com/kairos-development/kairos-agent/internal/providers/config"
	runtimeprovider "github.com/kairos-development/kairos-agent/internal/providers/runtime"
	"github.com/kairos-development/kairos-agent/internal/runtime"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_DemoMode tests the complete demo mode workflow.
func TestE2E_DemoMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	// Setup state directory
	stateDir := t.TempDir()

	// Initialize configuration
	configManager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	// Initialize runtime
	runtimeManager := runtime.NewManager("test-version", logrus.New())

	// Create repositories
	configRepo := configprovider.New(configManager)
	runtimeRepo := runtimeprovider.New(runtimeManager)

	// Create service
	svc := serviceagent.New(
		runtimeRepo,
		configRepo,
		&mockDisclaimerRepo{accepted: false},
		&mockExportRepo{},
		&mockPluginTrustRepo{},
	)

	// Test 1: Verify initial state is demo mode
	status, err := svc.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.LicenseStateDemo, status.License)
	assert.Equal(t, entity.RunModeIdle, status.Mode)

	// Test 2: Start scanning in demo mode
	err = svc.StartScan(ctx)
	require.NoError(t, err)

	status, err = svc.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeScanning, status.Mode)
	assert.Equal(t, entity.LicenseStateDemo, status.License)

	// Test 3: Verify demo mode restrictions
	checkOutput, err := svc.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.LicenseStateDemo, checkOutput.Status.License)
	assert.Contains(t, checkOutput.AllowedOutbound, "exchange-public")
	assert.Contains(t, checkOutput.AllowedOutbound, "exchange-private")
	assert.NotContains(t, checkOutput.AllowedOutbound, "cloud-license")

	// Test 4: Attempt to start live trading without disclaimer (should fail)
	err = svc.StartLive(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disclaimer")

	// Test 5: Stop all and return to idle
	err = svc.StopAll(ctx)
	require.NoError(t, err)

	status, err = svc.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeIdle, status.Mode)
}

// TestE2E_PaperTrading tests the complete paper trading workflow.
func TestE2E_PaperTrading(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	// Setup state directory
	stateDir := t.TempDir()

	// Initialize configuration
	configManager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	// Initialize runtime
	runtimeManager := runtime.NewManager("test-version", logrus.New())

	// Create repositories
	configRepo := configprovider.New(configManager)
	runtimeRepo := runtimeprovider.New(runtimeManager)

	// Create service
	svc := serviceagent.New(
		runtimeRepo,
		configRepo,
		&mockDisclaimerRepo{accepted: true},
		&mockExportRepo{},
		&mockPluginTrustRepo{},
	)

	// Test 1: Start paper trading
	err = svc.StartPaper(ctx)
	require.NoError(t, err)

	status, err := svc.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RunModePaperTrading, status.Mode)

	// Test 2: Verify paper trading state persists
	time.Sleep(500 * time.Millisecond)

	status, err = svc.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RunModePaperTrading, status.Mode)

	// Test 3: Update configuration while paper trading
	err = svc.UpdateConfig(ctx, serviceagent.UpdateConfigInput{
		Key:   "risk.max_position",
		Value: "2000",
	})
	require.NoError(t, err)

	cfg, err := svc.Config(ctx)
	require.NoError(t, err)
	assert.Equal(t, "2000", cfg.Risk.MaxPosition)

	// Test 4: Stop paper trading
	err = svc.StopAll(ctx)
	require.NoError(t, err)

	status, err = svc.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeIdle, status.Mode)
}

// TestE2E_Backtest tests the complete backtest workflow.
func TestE2E_Backtest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	// Setup state directory
	stateDir := t.TempDir()

	// Initialize configuration
	configManager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	// Initialize runtime
	runtimeManager := runtime.NewManager("test-version", logrus.New())

	// Create repositories
	configRepo := configprovider.New(configManager)
	runtimeRepo := runtimeprovider.New(runtimeManager)

	// Create service
	svc := serviceagent.New(
		runtimeRepo,
		configRepo,
		&mockDisclaimerRepo{accepted: true},
		&mockExportRepo{},
		&mockPluginTrustRepo{},
	)

	// Test 1: Queue backtest
	err = svc.QueueBacktest(ctx)
	require.NoError(t, err)

	status, err := svc.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeBacktesting, status.Mode)

	// Test 2: Verify backtest mode
	time.Sleep(500 * time.Millisecond)

	status, err = svc.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeBacktesting, status.Mode)

	// Test 3: Stop backtest
	err = svc.StopAll(ctx)
	require.NoError(t, err)

	status, err = svc.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeIdle, status.Mode)
}

// TestE2E_LiveTrading tests the complete live trading workflow.
func TestE2E_LiveTrading(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	// Setup state directory
	stateDir := t.TempDir()

	// Initialize configuration
	configManager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	// Initialize runtime
	runtimeManager := runtime.NewManager("test-version", logrus.New())

	// Create repositories
	configRepo := configprovider.New(configManager)
	runtimeRepo := runtimeprovider.New(runtimeManager)

	// Create service with accepted disclaimer
	svc := serviceagent.New(
		runtimeRepo,
		configRepo,
		&mockDisclaimerRepo{accepted: true},
		&mockExportRepo{},
		&mockPluginTrustRepo{},
	)

	// Test 1: Start live trading (requires disclaimer)
	err = svc.StartLive(ctx)
	require.NoError(t, err)

	status, err := svc.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeLiveTrading, status.Mode)
	assert.Equal(t, entity.LicenseStateLicensed, status.License)

	// Test 2: Verify live trading state
	time.Sleep(500 * time.Millisecond)

	status, err = svc.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeLiveTrading, status.Mode)

	// Test 3: Emergency stop
	err = svc.EmergencyStop(ctx)
	require.NoError(t, err)

	status, err = svc.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeHalted, status.Mode)
	assert.Equal(t, entity.ConnectivityStateNetworkWait, status.Connectivity)
}

// TestE2E_ModeTransitions tests transitions between different modes.
func TestE2E_ModeTransitions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	// Setup state directory
	stateDir := t.TempDir()

	// Initialize configuration
	configManager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	// Initialize runtime
	runtimeManager := runtime.NewManager("test-version", logrus.New())

	// Create repositories
	configRepo := configprovider.New(configManager)
	runtimeRepo := runtimeprovider.New(runtimeManager)

	// Create service
	svc := serviceagent.New(
		runtimeRepo,
		configRepo,
		&mockDisclaimerRepo{accepted: true},
		&mockExportRepo{},
		&mockPluginTrustRepo{},
	)

	// Test transition sequence: Idle -> Scan -> Paper -> Backtest -> Idle
	transitions := []struct {
		name     string
		action   func() error
		expected entity.RunMode
	}{
		{
			name:     "Idle to Scanning",
			action:   func() error { return svc.StartScan(ctx) },
			expected: entity.RunModeScanning,
		},
		{
			name:     "Scanning to Paper Trading",
			action:   func() error { return svc.StartPaper(ctx) },
			expected: entity.RunModePaperTrading,
		},
		{
			name:     "Paper Trading to Backtest",
			action:   func() error { return svc.QueueBacktest(ctx) },
			expected: entity.RunModeBacktesting,
		},
		{
			name:     "Backtest to Idle",
			action:   func() error { return svc.StopAll(ctx) },
			expected: entity.RunModeIdle,
		},
	}

	for _, tt := range transitions {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action()
			require.NoError(t, err)

			status, err := svc.Status(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, status.Mode)
		})
	}
}

// TestE2E_ConfigurationPersistence tests configuration persistence across operations.
func TestE2E_ConfigurationPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()

	// Setup state directory
	stateDir := t.TempDir()

	// Initialize configuration
	configManager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	// Initialize runtime
	runtimeManager := runtime.NewManager("test-version", logrus.New())

	// Create repositories
	configRepo := configprovider.New(configManager)
	runtimeRepo := runtimeprovider.New(runtimeManager)

	// Create service
	svc := serviceagent.New(
		runtimeRepo,
		configRepo,
		&mockDisclaimerRepo{accepted: true},
		&mockExportRepo{},
		&mockPluginTrustRepo{},
	)

	// Test 1: Update configuration
	err = svc.UpdateConfig(ctx, serviceagent.UpdateConfigInput{
		Key:   "risk.max_position",
		Value: "5000",
	})
	require.NoError(t, err)

	// Test 2: Verify configuration persisted
	cfg, err := svc.Config(ctx)
	require.NoError(t, err)
	assert.Equal(t, "5000", cfg.Risk.MaxPosition)

	// Test 3: Create new service instance with same state directory
	newConfigManager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	newConfigRepo := configprovider.New(newConfigManager)
	newRuntimeRepo := runtimeprovider.New(runtime.NewManager("test-version", logrus.New()))

	newSvc := serviceagent.New(
		newRuntimeRepo,
		newConfigRepo,
		&mockDisclaimerRepo{accepted: true},
		&mockExportRepo{},
		&mockPluginTrustRepo{},
	)

	// Test 4: Verify configuration loaded from disk
	cfg, err = newSvc.Config(ctx)
	require.NoError(t, err)
	assert.Equal(t, "5000", cfg.Risk.MaxPosition)
}

// TestE2E_ExportCSV tests CSV export functionality.
func TestE2E_ExportCSV(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()

	// Setup state directory
	stateDir := t.TempDir()

	// Initialize configuration
	configManager, err := config.NewManager(stateDir)
	require.NoError(t, err)

	// Initialize runtime
	runtimeManager := runtime.NewManager("test-version", logrus.New())

	// Create repositories
	configRepo := configprovider.New(configManager)
	runtimeRepo := runtimeprovider.New(runtimeManager)

	// Create service with mock exporter
	mockExporter := &mockExportRepo{
		output: serviceagent.ExportCSVOutput{Path: stateDir + "/export.csv"},
	}

	svc := serviceagent.New(
		runtimeRepo,
		configRepo,
		&mockDisclaimerRepo{accepted: true},
		mockExporter,
		&mockPluginTrustRepo{},
	)

	// Test export
	output, err := svc.ExportCSV(ctx, serviceagent.ExportCSVInput{
		DestinationPath: stateDir + "/export.csv",
	})
	require.NoError(t, err)
	assert.Equal(t, stateDir+"/export.csv", output.Path)
}

// Mock repositories for E2E tests

type mockDisclaimerRepo struct {
	accepted bool
	sources  []string
}

func (m *mockDisclaimerRepo) DisclaimerAccepted(context.Context) (bool, error) {
	return m.accepted, nil
}

func (m *mockDisclaimerRepo) AcceptDisclaimer(_ context.Context, source string) error {
	m.sources = append(m.sources, source)
	m.accepted = true
	return nil
}

type mockExportRepo struct {
	output serviceagent.ExportCSVOutput
}

func (m *mockExportRepo) ExportCSV(context.Context, serviceagent.ExportCSVInput) (serviceagent.ExportCSVOutput, error) {
	return m.output, nil
}

type mockPluginTrustRepo struct {
	trusted []string
}

func (m *mockPluginTrustRepo) TrustPluginKey(_ context.Context, input serviceagent.TrustPluginKeyInput) error {
	m.trusted = append(m.trusted, input.Fingerprint)
	return nil
}
