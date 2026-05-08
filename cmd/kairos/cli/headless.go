package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kairos-development/kairos-agent/internal/config"
	configprovider "github.com/kairos-development/kairos-agent/internal/providers/config"
	runtimeprovider "github.com/kairos-development/kairos-agent/internal/providers/runtime"
	"github.com/kairos-development/kairos-agent/internal/runtime"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewHeadlessCommand creates the headless command.
func NewHeadlessCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "headless",
		Short: "Run agent in headless mode (no TUI)",
		Long: `Run Kairos agent in headless mode for systemd/Docker deployment.

Modes:
  --paper    Paper trading mode (safe for testing)
  --scan     Market scanning mode (read-only)
  --live     Live trading mode (requires --confirm)

Headless mode:
- No interactive TUI
- Logs to stdout/stderr
- Graceful shutdown on SIGTERM/SIGINT
- Suitable for systemd units
- Suitable for Docker containers

Example systemd unit:
  [Service]
  ExecStart=/usr/local/bin/kairos headless --paper
  Restart=always
  User=kairos

⚠️  Live trading in headless mode requires:
  - Accepted disclaimer
  - Valid license
  - --confirm flag
  - Monitoring/alerting setup`,
		RunE: runHeadless,
	}

	cmd.Flags().String("state-dir", "", "Custom state directory (default: ~/.kairos)")
	cmd.Flags().Bool("paper", false, "Paper trading mode")
	cmd.Flags().Bool("scan", false, "Scanning mode")
	cmd.Flags().Bool("live", false, "Live trading mode (requires --confirm)")
	cmd.Flags().Bool("confirm", false, "Confirm live trading in headless mode")

	return cmd
}

func runHeadless(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Get flags
	paper, _ := cmd.Flags().GetBool("paper")
	scan, _ := cmd.Flags().GetBool("scan")
	live, _ := cmd.Flags().GetBool("live")
	confirm, _ := cmd.Flags().GetBool("confirm")

	// Validate mode flags
	modeCount := 0
	if paper {
		modeCount++
	}
	if scan {
		modeCount++
	}
	if live {
		modeCount++
	}

	if modeCount == 0 {
		return fmt.Errorf("mode flag required: --paper, --scan, or --live")
	}
	if modeCount > 1 {
		return fmt.Errorf("only one mode flag allowed")
	}

	// Live mode requires confirmation
	if live && !confirm {
		return fmt.Errorf("live trading in headless mode requires --confirm flag\n\n⚠️  This is dangerous. Ensure monitoring is in place.")
	}

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

	// Determine mode
	var mode string
	if paper {
		mode = "paper"
	} else if scan {
		mode = "scan"
	} else if live {
		mode = "live"
	}

	fmt.Printf("🤖 Kairos Headless Mode\n")
	fmt.Printf("═══════════════════════════════════════════════════════════\n")
	fmt.Printf("Mode:       %s\n", mode)
	fmt.Printf("State dir:  %s\n", stateDir)
	fmt.Printf("═══════════════════════════════════════════════════════════\n")
	fmt.Println()

	// Load configuration
	configManager, err := config.NewManager(stateDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Create service
	runtimeManager := runtime.NewManager("headless", logrus.New())
	configRepo := configprovider.New(configManager)
	runtimeRepo := runtimeprovider.New(runtimeManager)

	svc := serviceagent.New(
		runtimeRepo,
		configRepo,
		&mockDisclaimerRepo{},
		&mockExportRepo{},
		&mockPluginTrustRepo{},
	)

	// Start appropriate mode
	fmt.Printf("Starting %s mode...\n", mode)

	switch mode {
	case "paper":
		if err := svc.StartPaper(ctx); err != nil {
			return fmt.Errorf("start paper trading: %w", err)
		}
		fmt.Println("✓ Paper trading started")

	case "scan":
		if err := svc.StartScan(ctx); err != nil {
			return fmt.Errorf("start scanning: %w", err)
		}
		fmt.Println("✓ Scanning started")

	case "live":
		if err := svc.StartLive(ctx); err != nil {
			return fmt.Errorf("start live trading: %w", err)
		}
		fmt.Println("✓ Live trading started")
		fmt.Println("⚠️  LIVE TRADING ACTIVE - Monitor closely")
	}

	fmt.Println()
	fmt.Println("Agent running. Press Ctrl+C to stop gracefully.")
	fmt.Println()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigChan
	fmt.Printf("\n\nReceived signal: %s\n", sig)
	fmt.Println("Shutting down gracefully...")

	// Stop agent
	if err := svc.StopAll(ctx); err != nil {
		return fmt.Errorf("stop agent: %w", err)
	}

	fmt.Println("✓ Agent stopped")
	fmt.Println("✅ Shutdown complete")

	return nil
}
