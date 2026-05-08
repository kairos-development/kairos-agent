package agent

import (
	"context"
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
)

type runtimeMock struct {
	status      entity.RuntimeStatus
	transitions []struct {
		mode   entity.RunMode
		reason string
	}
	license      []entity.LicenseState
	connectivity []entity.ConnectivityState
}

func (m *runtimeMock) Status(context.Context) (entity.RuntimeStatus, error) { return m.status, nil }
func (m *runtimeMock) Transition(_ context.Context, mode entity.RunMode, reason string) error {
	m.transitions = append(m.transitions, struct {
		mode   entity.RunMode
		reason string
	}{mode, reason})
	return nil
}
func (m *runtimeMock) SetLicense(_ context.Context, state entity.LicenseState) error {
	m.license = append(m.license, state)
	return nil
}
func (m *runtimeMock) SetConnectivity(_ context.Context, state entity.ConnectivityState) error {
	m.connectivity = append(m.connectivity, state)
	return nil
}

type configMock struct {
	cfg     entity.AgentConfig
	setCfg  []entity.AgentConfig
	updates []map[string]interface{}
}

func (m *configMock) Config(context.Context) (entity.AgentConfig, error) { return m.cfg, nil }
func (m *configMock) SetConfig(_ context.Context, cfg entity.AgentConfig) error {
	m.setCfg = append(m.setCfg, cfg)
	m.cfg = cfg
	return nil
}
func (m *configMock) UpdateConfig(_ context.Context, updates map[string]interface{}) error {
	m.updates = append(m.updates, updates)
	return nil
}

type disclaimerMock struct {
	accepted bool
	sources  []string
}

func (m *disclaimerMock) DisclaimerAccepted(context.Context) (bool, error) { return m.accepted, nil }
func (m *disclaimerMock) AcceptDisclaimer(_ context.Context, source string) error {
	m.sources = append(m.sources, source)
	return nil
}

type exporterMock struct{ output ExportCSVOutput }

func (m exporterMock) ExportCSV(context.Context, ExportCSVInput) (ExportCSVOutput, error) {
	return m.output, nil
}

type pluginTrustMock struct{ trusted []string }

func (m *pluginTrustMock) TrustPluginKey(_ context.Context, input TrustPluginKeyInput) error {
	m.trusted = append(m.trusted, input.Fingerprint)
	return nil
}

func TestStartLiveRequiresDisclaimer(t *testing.T) {
	svc := New(&runtimeMock{}, &configMock{}, &disclaimerMock{accepted: false}, exporterMock{}, &pluginTrustMock{})
	if err := svc.StartLive(context.Background()); err == nil {
		t.Fatal("expected disclaimer requirement error")
	}
}

func TestCheckBuildsAllowedOutbound(t *testing.T) {
	runtimeRepo := &runtimeMock{status: entity.RuntimeStatus{Mode: entity.RunModeIdle, Connectivity: entity.ConnectivityStateConnected, License: entity.LicenseStateLicensed, Integrity: entity.IntegrityStateTrusted, LastUpdatedAtUTC: time.Now().UTC()}}
	cfg := entity.AgentConfig{SchemaVersion: 1, Telemetry: entity.TelemetryConsent{Enabled: true, Profile: entity.TelemetryProfileMinimal, ConsentVersion: "v1"}}
	svc := New(runtimeRepo, &configMock{cfg: cfg}, &disclaimerMock{accepted: true}, exporterMock{}, &pluginTrustMock{})
	out, err := svc.Check(context.Background())
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if len(out.AllowedOutbound) != 4 {
		t.Fatalf("expected 4 allowed outbound channels, got %d", len(out.AllowedOutbound))
	}
}

func TestSetConfig(t *testing.T) {
	configRepo := &configMock{}
	svc := New(&runtimeMock{}, configRepo, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})
	ctx := context.Background()

	newConfig := entity.AgentConfig{
		SchemaVersion:  1,
		ExportTimezone: "America/New_York",
	}

	err := svc.SetConfig(ctx, SetConfigInput{Config: newConfig})
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	if len(configRepo.setCfg) != 1 {
		t.Fatalf("expected 1 SetConfig call, got %d", len(configRepo.setCfg))
	}
	if configRepo.setCfg[0].ExportTimezone != "America/New_York" {
		t.Fatalf("expected timezone America/New_York, got %s", configRepo.setCfg[0].ExportTimezone)
	}
}

func TestSetConfig_InvalidSchema(t *testing.T) {
	configRepo := &configMock{}
	svc := New(&runtimeMock{}, configRepo, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})
	ctx := context.Background()

	invalidConfig := entity.AgentConfig{
		SchemaVersion:  999,
		ExportTimezone: "UTC",
	}

	err := svc.SetConfig(ctx, SetConfigInput{Config: invalidConfig})
	if err == nil {
		t.Fatal("expected error for invalid schema version")
	}
	if len(configRepo.setCfg) != 0 {
		t.Fatalf("expected 0 SetConfig calls, got %d", len(configRepo.setCfg))
	}
}

func TestUpdateConfig(t *testing.T) {
	configRepo := &configMock{}
	svc := New(&runtimeMock{}, configRepo, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})
	ctx := context.Background()

	err := svc.UpdateConfig(ctx, UpdateConfigInput{
		Key:   "export_timezone",
		Value: "Europe/London",
	})
	if err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}

	if len(configRepo.updates) != 1 {
		t.Fatalf("expected 1 UpdateConfig call, got %d", len(configRepo.updates))
	}
	if configRepo.updates[0]["export_timezone"] != "Europe/London" {
		t.Fatalf("expected timezone Europe/London, got %v", configRepo.updates[0]["export_timezone"])
	}
}

func TestUpdateConfig_MultipleFields(t *testing.T) {
	configRepo := &configMock{}
	svc := New(&runtimeMock{}, configRepo, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})
	ctx := context.Background()

	err := svc.UpdateConfig(ctx, UpdateConfigInput{
		Key:   "risk.max_position",
		Value: "5000",
	})
	if err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}

	err = svc.UpdateConfig(ctx, UpdateConfigInput{
		Key:   "journal_max_size_bytes",
		Value: int64(64 << 20),
	})
	if err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}

	if len(configRepo.updates) != 2 {
		t.Fatalf("expected 2 UpdateConfig calls, got %d", len(configRepo.updates))
	}
	if configRepo.updates[0]["risk.max_position"] != "5000" {
		t.Fatalf("expected max_position 5000, got %v", configRepo.updates[0]["risk.max_position"])
	}
	if configRepo.updates[1]["journal_max_size_bytes"] != int64(64<<20) {
		t.Fatalf("expected journal_max_size_bytes %d, got %v", int64(64<<20), configRepo.updates[1]["journal_max_size_bytes"])
	}
}

func TestStatus(t *testing.T) {
	expectedStatus := entity.RuntimeStatus{
		Mode:         entity.RunModeScanning,
		Connectivity: entity.ConnectivityStateConnected,
		License:      entity.LicenseStateDemo,
	}
	runtimeRepo := &runtimeMock{status: expectedStatus}
	svc := New(runtimeRepo, &configMock{}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	status, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if status.Mode != entity.RunModeScanning {
		t.Fatalf("expected mode scanning, got %s", status.Mode)
	}
}

func TestConfig(t *testing.T) {
	expectedConfig := entity.AgentConfig{
		SchemaVersion:  1,
		ExportTimezone: "Asia/Tokyo",
	}
	configRepo := &configMock{cfg: expectedConfig}
	svc := New(&runtimeMock{}, configRepo, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	cfg, err := svc.Config(context.Background())
	if err != nil {
		t.Fatalf("Config failed: %v", err)
	}
	if cfg.ExportTimezone != "Asia/Tokyo" {
		t.Fatalf("expected timezone Asia/Tokyo, got %s", cfg.ExportTimezone)
	}
}

func TestStartScan(t *testing.T) {
	runtimeRepo := &runtimeMock{}
	svc := New(runtimeRepo, &configMock{}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	err := svc.StartScan(context.Background())
	if err != nil {
		t.Fatalf("StartScan failed: %v", err)
	}
	if len(runtimeRepo.transitions) != 1 || runtimeRepo.transitions[0].mode != entity.RunModeScanning {
		t.Fatalf("expected RunModeScanning, got %v", runtimeRepo.transitions)
	}
}

func TestStartPaper(t *testing.T) {
	runtimeRepo := &runtimeMock{}
	svc := New(runtimeRepo, &configMock{}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	err := svc.StartPaper(context.Background())
	if err != nil {
		t.Fatalf("StartPaper failed: %v", err)
	}
	if len(runtimeRepo.transitions) != 1 || runtimeRepo.transitions[0].mode != entity.RunModePaperTrading {
		t.Fatalf("expected RunModePaperTrading, got %v", runtimeRepo.transitions)
	}
}

func TestStartLive_Success(t *testing.T) {
	runtimeRepo := &runtimeMock{}
	svc := New(runtimeRepo, &configMock{}, &disclaimerMock{accepted: true}, exporterMock{}, &pluginTrustMock{})

	err := svc.StartLive(context.Background())
	if err != nil {
		t.Fatalf("StartLive failed: %v", err)
	}
	if len(runtimeRepo.license) != 1 || runtimeRepo.license[0] != entity.LicenseStateLicensed {
		t.Fatalf("expected LicenseStateLicensed, got %v", runtimeRepo.license)
	}
	if len(runtimeRepo.transitions) != 1 || runtimeRepo.transitions[0].mode != entity.RunModeLiveTrading {
		t.Fatalf("expected RunModeLiveTrading, got %v", runtimeRepo.transitions)
	}
}

func TestQueueBacktest(t *testing.T) {
	runtimeRepo := &runtimeMock{}
	svc := New(runtimeRepo, &configMock{}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	err := svc.QueueBacktest(context.Background())
	if err != nil {
		t.Fatalf("QueueBacktest failed: %v", err)
	}
	if len(runtimeRepo.transitions) != 1 || runtimeRepo.transitions[0].mode != entity.RunModeBacktesting {
		t.Fatalf("expected RunModeBacktesting, got %v", runtimeRepo.transitions)
	}
}

func TestStopAll(t *testing.T) {
	runtimeRepo := &runtimeMock{}
	svc := New(runtimeRepo, &configMock{}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	err := svc.StopAll(context.Background())
	if err != nil {
		t.Fatalf("StopAll failed: %v", err)
	}
	if len(runtimeRepo.transitions) != 1 || runtimeRepo.transitions[0].mode != entity.RunModeIdle {
		t.Fatalf("expected RunModeIdle, got %v", runtimeRepo.transitions)
	}
}

func TestEmergencyStop(t *testing.T) {
	runtimeRepo := &runtimeMock{}
	svc := New(runtimeRepo, &configMock{}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	err := svc.EmergencyStop(context.Background())
	if err != nil {
		t.Fatalf("EmergencyStop failed: %v", err)
	}
	if len(runtimeRepo.transitions) != 1 || runtimeRepo.transitions[0].mode != entity.RunModeHalted {
		t.Fatalf("expected RunModeHalted, got %v", runtimeRepo.transitions)
	}
	if len(runtimeRepo.connectivity) != 1 || runtimeRepo.connectivity[0] != entity.ConnectivityStateNetworkWait {
		t.Fatalf("expected ConnectivityStateNetworkWait, got %v", runtimeRepo.connectivity)
	}
}

func TestExportCSV(t *testing.T) {
	expectedOutput := ExportCSVOutput{Path: "/tmp/export.csv"}
	svc := New(&runtimeMock{}, &configMock{}, &disclaimerMock{}, exporterMock{output: expectedOutput}, &pluginTrustMock{})

	output, err := svc.ExportCSV(context.Background(), ExportCSVInput{DestinationPath: "/tmp/export.csv"})
	if err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}
	if output.Path != "/tmp/export.csv" {
		t.Fatalf("expected path /tmp/export.csv, got %s", output.Path)
	}
}

func TestAcceptDisclaimer(t *testing.T) {
	disclaimerRepo := &disclaimerMock{}
	svc := New(&runtimeMock{}, &configMock{}, disclaimerRepo, exporterMock{}, &pluginTrustMock{})

	err := svc.AcceptDisclaimer(context.Background(), "tui")
	if err != nil {
		t.Fatalf("AcceptDisclaimer failed: %v", err)
	}
	if len(disclaimerRepo.sources) != 1 || disclaimerRepo.sources[0] != "tui" {
		t.Fatalf("expected source 'tui', got %v", disclaimerRepo.sources)
	}
}

func TestTrustPluginKey_Allowed(t *testing.T) {
	cfg := entity.AgentConfig{
		SchemaVersion:             1,
		TrustedPluginFingerprints: []string{"test-fingerprint"},
	}
	pluginRepo := &pluginTrustMock{}
	svc := New(&runtimeMock{}, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, pluginRepo)

	err := svc.TrustPluginKey(context.Background(), TrustPluginKeyInput{
		Fingerprint: "test-fingerprint",
		PublicKey:   []byte("test-key"),
	})
	if err != nil {
		t.Fatalf("TrustPluginKey failed: %v", err)
	}
	if len(pluginRepo.trusted) != 1 || pluginRepo.trusted[0] != "test-fingerprint" {
		t.Fatalf("expected fingerprint 'test-fingerprint', got %v", pluginRepo.trusted)
	}
}

func TestTrustPluginKey_NotAllowed(t *testing.T) {
	cfg := entity.AgentConfig{
		SchemaVersion:             1,
		TrustedPluginFingerprints: []string{"allowed-fingerprint"},
	}
	pluginRepo := &pluginTrustMock{}
	svc := New(&runtimeMock{}, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, pluginRepo)

	err := svc.TrustPluginKey(context.Background(), TrustPluginKeyInput{
		Fingerprint: "not-allowed-fingerprint",
		PublicKey:   []byte("test-key"),
	})
	if err == nil {
		t.Fatal("expected error for non-allowed fingerprint")
	}
	if len(pluginRepo.trusted) != 0 {
		t.Fatalf("expected 0 trusted keys, got %d", len(pluginRepo.trusted))
	}
}

func TestCheck_DemoMode(t *testing.T) {
	runtimeRepo := &runtimeMock{status: entity.RuntimeStatus{
		Mode:             entity.RunModeIdle,
		Connectivity:     entity.ConnectivityStateConnected,
		License:          entity.LicenseStateDemo,
		Integrity:        entity.IntegrityStateTrusted,
		LastUpdatedAtUTC: time.Now().UTC(),
	}}
	cfg := entity.AgentConfig{SchemaVersion: 1, Telemetry: entity.TelemetryConsent{Enabled: false, Profile: entity.TelemetryProfileOff}}
	svc := New(runtimeRepo, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	out, err := svc.Check(context.Background())
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(out.AllowedOutbound) != 2 {
		t.Fatalf("expected 2 allowed outbound channels in demo mode, got %d", len(out.AllowedOutbound))
	}
}

func TestCheck_MinimalProfile(t *testing.T) {
	runtimeRepo := &runtimeMock{status: entity.RuntimeStatus{
		Mode:             entity.RunModeIdle,
		Connectivity:     entity.ConnectivityStateConnected,
		License:          entity.LicenseStateLicensed,
		Integrity:        entity.IntegrityStateTrusted,
		LastUpdatedAtUTC: time.Now().UTC(),
	}}
	cfg := entity.AgentConfig{SchemaVersion: 1, Telemetry: entity.TelemetryConsent{Enabled: true, Profile: entity.TelemetryProfileSync}}
	svc := New(runtimeRepo, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	out, err := svc.Check(context.Background())
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(out.AllowedOutbound) != 4 {
		t.Fatalf("expected 4 allowed outbound channels in full profile, got %d", len(out.AllowedOutbound))
	}
}

func TestStartLive_WithoutDisclaimer(t *testing.T) {
	runtimeRepo := &runtimeMock{}
	svc := New(runtimeRepo, &configMock{}, &disclaimerMock{accepted: false}, exporterMock{}, &pluginTrustMock{})

	err := svc.StartLive(context.Background())
	if err == nil {
		t.Fatal("expected error when starting live without disclaimer")
	}
	if len(runtimeRepo.license) != 0 {
		t.Fatalf("expected no license changes, got %d", len(runtimeRepo.license))
	}
}

func TestEmergencyStop_SetsNetworkWait(t *testing.T) {
	runtimeRepo := &runtimeMock{}
	svc := New(runtimeRepo, &configMock{}, &disclaimerMock{}, exporterMock{}, &pluginTrustMock{})

	err := svc.EmergencyStop(context.Background())
	if err != nil {
		t.Fatalf("EmergencyStop failed: %v", err)
	}
	if len(runtimeRepo.connectivity) != 1 || runtimeRepo.connectivity[0] != entity.ConnectivityStateNetworkWait {
		t.Fatalf("expected ConnectivityStateNetworkWait, got %v", runtimeRepo.connectivity)
	}
	if len(runtimeRepo.transitions) != 1 || runtimeRepo.transitions[0].mode != entity.RunModeHalted {
		t.Fatalf("expected RunModeHalted, got %v", runtimeRepo.transitions)
	}
}

func TestTrustPluginKey_EmptyFingerprint(t *testing.T) {
	cfg := entity.AgentConfig{
		SchemaVersion:             1,
		TrustedPluginFingerprints: []string{"allowed-fingerprint"},
	}
	pluginRepo := &pluginTrustMock{}
	svc := New(&runtimeMock{}, &configMock{cfg: cfg}, &disclaimerMock{}, exporterMock{}, pluginRepo)

	err := svc.TrustPluginKey(context.Background(), TrustPluginKeyInput{
		Fingerprint: "",
		PublicKey:   []byte("test-key"),
	})
	if err == nil {
		t.Fatal("expected error for empty fingerprint")
	}
	if len(pluginRepo.trusted) != 0 {
		t.Fatalf("expected 0 trusted keys, got %d", len(pluginRepo.trusted))
	}
}

func TestNew(t *testing.T) {
	runtimeRepo := &runtimeMock{}
	configRepo := &configMock{}
	disclaimerRepo := &disclaimerMock{}
	exporter := exporterMock{}
	pluginRepo := &pluginTrustMock{}

	svc := New(runtimeRepo, configRepo, disclaimerRepo, exporter, pluginRepo)

	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}
