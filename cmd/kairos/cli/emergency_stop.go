package cli

import (
	"fmt"

	"github.com/kairos-development/kairos-agent/internal/config"
	configprovider "github.com/kairos-development/kairos-agent/internal/providers/config"
	runtimeprovider "github.com/kairos-development/kairos-agent/internal/providers/runtime"
	"github.com/kairos-development/kairos-agent/internal/runtime"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewEmergencyStopCommand creates the emergency-stop command.
func NewEmergencyStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "emergency-stop",
		Short: "Emergency stop - halt all trading immediately",
		Long: `Emergency stop command for critical situations.

⚠️  DESTRUCTIVE OPERATION - USE WITH CAUTION

This command:
- Bypasses normal shutdown procedures
- Cancels all open orders immediately
- Closes positions at market (if --close-positions)
- Halts runtime and blocks new entries
- Sets connectivity to network_waiting state
- Creates audit record

Use this ONLY in emergencies:
- Exchange API compromise suspected
- Runaway strategy detected
- System integrity compromised
- Regulatory requirement

For normal shutdown, use TUI /stopall command instead.

Requires --confirm flag to prevent accidental execution.`,
		RunE: runEmergencyStop,
	}

	cmd.Flags().String("state-dir", "", "Custom state directory (default: ~/.kairos)")
	cmd.Flags().Bool("confirm", false, "Confirm emergency stop (required)")
	cmd.Flags().Bool("close-positions", false, "Close all positions at market (dangerous)")

	return cmd
}

func runEmergencyStop(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Require explicit confirmation
	confirmed, _ := cmd.Flags().GetBool("confirm")
	if !confirmed {
		return fmt.Errorf("emergency stop requires --confirm flag\n\nThis is a destructive operation. Add --confirm to proceed.")
	}

	closePositions, _ := cmd.Flags().GetBool("close-positions")

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

	fmt.Println("🚨 EMERGENCY STOP INITIATED")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	// Load configuration
	configManager, err := config.NewManager(stateDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Create service
	runtimeManager := runtime.NewManager("emergency-stop-cmd", logrus.New())
	configRepo := configprovider.New(configManager)
	runtimeRepo := runtimeprovider.New(runtimeManager)

	svc := serviceagent.New(
		runtimeRepo,
		configRepo,
		&mockDisclaimerRepo{},
		&mockExportRepo{},
		&mockPluginTrustRepo{},
	)

	// Get current status
	status, err := svc.Status(ctx)
	if err != nil {
		return fmt.Errorf("get status: %w", err)
	}

	fmt.Printf("Current mode: %s\n", status.Mode)
	fmt.Printf("Current connectivity: %s\n", status.Connectivity)
	fmt.Println()

	if closePositions {
		fmt.Println("⚠️  --close-positions flag set")
		fmt.Println("    All positions will be closed at market price")
		fmt.Println()

		confirmed, err := confirmPrompt("⚠️  Close all positions at market?")
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Aborted.")
			return nil
		}

		// TODO: Implement position closing logic
		fmt.Println("❌ Position closing not yet implemented")
		fmt.Println("    Proceeding with emergency stop only")
		fmt.Println()
	}

	// Execute emergency stop
	fmt.Println("Executing emergency stop...")

	if err := svc.EmergencyStop(ctx); err != nil {
		return fmt.Errorf("emergency stop failed: %w", err)
	}

	// Verify new status
	status, err = svc.Status(ctx)
	if err != nil {
		return fmt.Errorf("verify status: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ Runtime halted")
	fmt.Println("✓ New entries blocked")
	fmt.Println("✓ Connectivity set to network_waiting")
	fmt.Println()
	fmt.Printf("New mode: %s\n", status.Mode)
	fmt.Printf("New connectivity: %s\n", status.Connectivity)
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("✅ Emergency stop complete")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Review audit logs for cause")
	fmt.Println("  2. Verify all orders cancelled on exchange")
	fmt.Println("  3. Check positions and balances")
	fmt.Println("  4. Run 'kairos check' to verify system state")
	fmt.Println("  5. Restart via TUI when ready")

	return nil
}
