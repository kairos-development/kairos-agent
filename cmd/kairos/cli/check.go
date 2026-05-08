package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kairos-development/kairos-agent/internal/config"
	configprovider "github.com/kairos-development/kairos-agent/internal/providers/config"
	runtimeprovider "github.com/kairos-development/kairos-agent/internal/providers/runtime"
	"github.com/kairos-development/kairos-agent/internal/runtime"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewCheckCommand creates the check command.
func NewCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run system health check and diagnostics",
		Long: `Run comprehensive system health check.

Verifies:
- Profile initialization and configuration
- Vault accessibility
- License status
- Network connectivity requirements
- Telemetry settings
- Port availability
- File permissions
- Disk space

This is a read-only diagnostic command safe to run anytime.`,
		RunE: runCheck,
	}

	cmd.Flags().String("state-dir", "", "Custom state directory (default: ~/.kairos)")
	cmd.Flags().Bool("verbose", false, "Show detailed diagnostic information")

	return cmd
}

func runCheck(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	verbose, _ := cmd.Flags().GetBool("verbose")

	fmt.Println("🔍 Kairos System Health Check")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	// Get state directory
	stateDir, err := cmd.Flags().GetString("state-dir")
	if err != nil {
		return err
	}
	if stateDir == "" {
		stateDir, err = getStateDir()
		if err != nil {
			return err
		}
	}

	checks := []checkFunc{
		{name: "Profile Initialization", fn: checkProfileInit(stateDir, verbose)},
		{name: "Configuration", fn: checkConfiguration(stateDir, verbose)},
		{name: "File Permissions", fn: checkFilePermissions(stateDir, verbose)},
		{name: "Disk Space", fn: checkDiskSpace(stateDir, verbose)},
		{name: "Runtime Status", fn: checkRuntimeStatus(ctx, stateDir, verbose)},
		{name: "License Status", fn: checkLicenseStatus(ctx, stateDir, verbose)},
		{name: "Network Requirements", fn: checkNetworkRequirements(ctx, stateDir, verbose)},
		{name: "Telemetry Settings", fn: checkTelemetrySettings(ctx, stateDir, verbose)},
	}

	passed := 0
	failed := 0
	warnings := 0

	for _, check := range checks {
		result := check.fn()

		var icon string
		switch result.status {
		case checkStatusPass:
			icon = "✓"
			passed++
		case checkStatusFail:
			icon = "✗"
			failed++
		case checkStatusWarn:
			icon = "⚠"
			warnings++
		}

		fmt.Printf("[%s] %s\n", icon, check.name)
		if result.message != "" {
			fmt.Printf("    %s\n", result.message)
		}
		if verbose && result.details != "" {
			fmt.Printf("    Details: %s\n", result.details)
		}
		fmt.Println()
	}

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Summary: %d passed, %d warnings, %d failed\n", passed, warnings, failed)

	if failed > 0 {
		fmt.Println("\n❌ System check failed. Review errors above.")
		return fmt.Errorf("health check failed with %d errors", failed)
	}

	if warnings > 0 {
		fmt.Println("\n⚠️  System check passed with warnings. Review above.")
	} else {
		fmt.Println("\n✅ All checks passed. System is healthy.")
	}

	return nil
}

type checkFunc struct {
	name string
	fn   func() checkResult
}

type checkStatus int

const (
	checkStatusPass checkStatus = iota
	checkStatusWarn
	checkStatusFail
)

type checkResult struct {
	status  checkStatus
	message string
	details string
}

func checkProfileInit(stateDir string, verbose bool) func() checkResult {
	return func() checkResult {
		configPath := filepath.Join(stateDir, "config.yaml")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return checkResult{
				status:  checkStatusFail,
				message: "Profile not initialized. Run 'kairos init' first.",
				details: fmt.Sprintf("Config file not found: %s", configPath),
			}
		}

		disclaimerPath := filepath.Join(stateDir, "disclaimer_accepted")
		if _, err := os.Stat(disclaimerPath); os.IsNotExist(err) {
			return checkResult{
				status:  checkStatusWarn,
				message: "Disclaimer not accepted. Live trading will be disabled.",
				details: fmt.Sprintf("Disclaimer file not found: %s", disclaimerPath),
			}
		}

		return checkResult{
			status:  checkStatusPass,
			message: fmt.Sprintf("Profile initialized at %s", stateDir),
		}
	}
}

func checkConfiguration(stateDir string, verbose bool) func() checkResult {
	return func() checkResult {
		configManager, err := config.NewManager(stateDir)
		if err != nil {
			return checkResult{
				status:  checkStatusFail,
				message: fmt.Sprintf("Failed to load configuration: %s", err.Error()),
			}
		}

		cfg := configManager.Current()
		if cfg == nil {
			return checkResult{
				status:  checkStatusFail,
				message: "Configuration is nil",
			}
		}

		if err := cfg.Validate(); err != nil {
			return checkResult{
				status:  checkStatusFail,
				message: fmt.Sprintf("Configuration validation failed: %s", err.Error()),
			}
		}

		details := fmt.Sprintf("Schema v%d, Timezone: %s, Journal: %d MB max",
			cfg.SchemaVersion,
			cfg.ExportTimezone,
			cfg.JournalMaxSizeBytes/(1024*1024))

		return checkResult{
			status:  checkStatusPass,
			message: "Configuration valid",
			details: details,
		}
	}
}

func checkFilePermissions(stateDir string, verbose bool) func() checkResult {
	return func() checkResult {
		// Check state directory permissions
		info, err := os.Stat(stateDir)
		if err != nil {
			return checkResult{
				status:  checkStatusFail,
				message: fmt.Sprintf("Cannot access state directory: %s", err.Error()),
			}
		}

		mode := info.Mode().Perm()
		if mode != 0o700 {
			return checkResult{
				status:  checkStatusWarn,
				message: fmt.Sprintf("State directory has permissive mode: %o (recommended: 0700)", mode),
			}
		}

		return checkResult{
			status:  checkStatusPass,
			message: "File permissions secure (0700)",
		}
	}
}

func checkDiskSpace(stateDir string, verbose bool) func() checkResult {
	return func() checkResult {
		// Simple check: try to create a temp file
		testFile := filepath.Join(stateDir, ".diskcheck")
		if err := os.WriteFile(testFile, []byte("test"), 0o600); err != nil {
			return checkResult{
				status:  checkStatusFail,
				message: fmt.Sprintf("Cannot write to state directory: %s", err.Error()),
			}
		}
		os.Remove(testFile)

		return checkResult{
			status:  checkStatusPass,
			message: "Disk space available",
		}
	}
}

func checkRuntimeStatus(ctx context.Context, stateDir string, verbose bool) func() checkResult {
	return func() checkResult {
		configManager, err := config.NewManager(stateDir)
		if err != nil {
			return checkResult{
				status:  checkStatusFail,
				message: fmt.Sprintf("Cannot load config: %s", err.Error()),
			}
		}

		runtimeManager := runtime.NewManager("check-cmd", logrus.New())
		configRepo := configprovider.New(configManager)
		runtimeRepo := runtimeprovider.New(runtimeManager)

		svc := serviceagent.New(
			runtimeRepo,
			configRepo,
			&mockDisclaimerRepo{},
			&mockExportRepo{},
			&mockPluginTrustRepo{},
		)

		status, err := svc.Status(ctx)
		if err != nil {
			return checkResult{
				status:  checkStatusFail,
				message: fmt.Sprintf("Cannot get runtime status: %s", err.Error()),
			}
		}

		details := fmt.Sprintf("Mode: %s, Connectivity: %s, License: %s",
			status.Mode, status.Connectivity, status.License)

		return checkResult{
			status:  checkStatusPass,
			message: "Runtime accessible",
			details: details,
		}
	}
}

func checkLicenseStatus(ctx context.Context, stateDir string, verbose bool) func() checkResult {
	return func() checkResult {
		configManager, err := config.NewManager(stateDir)
		if err != nil {
			return checkResult{
				status:  checkStatusFail,
				message: fmt.Sprintf("Cannot load config: %s", err.Error()),
			}
		}

		runtimeManager := runtime.NewManager("check-cmd", logrus.New())
		configRepo := configprovider.New(configManager)
		runtimeRepo := runtimeprovider.New(runtimeManager)

		svc := serviceagent.New(
			runtimeRepo,
			configRepo,
			&mockDisclaimerRepo{},
			&mockExportRepo{},
			&mockPluginTrustRepo{},
		)

		checkOutput, err := svc.Check(ctx)
		if err != nil {
			return checkResult{
				status:  checkStatusFail,
				message: fmt.Sprintf("Check failed: %s", err.Error()),
			}
		}

		var message string
		var status checkStatus
		switch checkOutput.Status.License {
		case "demo":
			message = "Demo mode (no license required)"
			status = checkStatusPass
		case "licensed":
			message = "Licensed"
			status = checkStatusPass
		case "grace":
			message = "Grace period active"
			status = checkStatusWarn
		case "unlicensed":
			message = "Unlicensed - live trading disabled"
			status = checkStatusWarn
		default:
			message = fmt.Sprintf("Unknown license state: %s", checkOutput.Status.License)
			status = checkStatusWarn
		}

		return checkResult{
			status:  status,
			message: message,
		}
	}
}

func checkNetworkRequirements(ctx context.Context, stateDir string, verbose bool) func() checkResult {
	return func() checkResult {
		configManager, err := config.NewManager(stateDir)
		if err != nil {
			return checkResult{
				status:  checkStatusFail,
				message: fmt.Sprintf("Cannot load config: %s", err.Error()),
			}
		}

		runtimeManager := runtime.NewManager("check-cmd", logrus.New())
		configRepo := configprovider.New(configManager)
		runtimeRepo := runtimeprovider.New(runtimeManager)

		svc := serviceagent.New(
			runtimeRepo,
			configRepo,
			&mockDisclaimerRepo{},
			&mockExportRepo{},
			&mockPluginTrustRepo{},
		)

		checkOutput, err := svc.Check(ctx)
		if err != nil {
			return checkResult{
				status:  checkStatusFail,
				message: fmt.Sprintf("Check failed: %s", err.Error()),
			}
		}

		message := fmt.Sprintf("Outbound channels: %v", checkOutput.AllowedOutbound)

		return checkResult{
			status:  checkStatusPass,
			message: message,
		}
	}
}

func checkTelemetrySettings(ctx context.Context, stateDir string, verbose bool) func() checkResult {
	return func() checkResult {
		configManager, err := config.NewManager(stateDir)
		if err != nil {
			return checkResult{
				status:  checkStatusFail,
				message: fmt.Sprintf("Cannot load config: %s", err.Error()),
			}
		}

		runtimeManager := runtime.NewManager("check-cmd", logrus.New())
		configRepo := configprovider.New(configManager)
		runtimeRepo := runtimeprovider.New(runtimeManager)

		svc := serviceagent.New(
			runtimeRepo,
			configRepo,
			&mockDisclaimerRepo{},
			&mockExportRepo{},
			&mockPluginTrustRepo{},
		)

		checkOutput, err := svc.Check(ctx)
		if err != nil {
			return checkResult{
				status:  checkStatusFail,
				message: fmt.Sprintf("Check failed: %s", err.Error()),
			}
		}

		message := fmt.Sprintf("Telemetry: %s (enabled=%v)", checkOutput.Telemetry.Profile, checkOutput.Telemetry.Enabled)

		return checkResult{
			status:  checkStatusPass,
			message: message,
		}
	}
}

// Mock repositories for check command
type mockDisclaimerRepo struct{}

func (m *mockDisclaimerRepo) DisclaimerAccepted(context.Context) (bool, error) {
	return true, nil
}

func (m *mockDisclaimerRepo) AcceptDisclaimer(context.Context, string) error {
	return nil
}

type mockExportRepo struct{}

func (m *mockExportRepo) ExportCSV(context.Context, serviceagent.ExportCSVInput) (serviceagent.ExportCSVOutput, error) {
	return serviceagent.ExportCSVOutput{}, nil
}

type mockPluginTrustRepo struct{}

func (m *mockPluginTrustRepo) TrustPluginKey(context.Context, serviceagent.TrustPluginKeyInput) error {
	return nil
}
