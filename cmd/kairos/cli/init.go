package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kairos-development/kairos-agent/internal/config"
	"github.com/spf13/cobra"
)

// NewInitCommand creates the init command.
func NewInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize Kairos profile and accept disclaimer",
		Long: `Initialize a new Kairos trading profile.

This command:
- Creates the state directory (~/.kairos)
- Generates default configuration
- Prompts for financial risk disclaimer acceptance
- Creates encrypted vault for API keys
- Sets up audit logging

This is a one-time setup command. Run before first use.`,
		RunE: runInit,
	}

	cmd.Flags().String("state-dir", "", "Custom state directory (default: ~/.kairos)")
	cmd.Flags().Bool("accept-disclaimer", false, "Accept financial risk disclaimer non-interactively")
	cmd.Flags().Bool("force", false, "Reinitialize existing profile (destructive)")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
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

	// Check if already initialized
	configPath := filepath.Join(stateDir, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			return fmt.Errorf("profile already initialized at %s\nUse --force to reinitialize (destructive)", stateDir)
		}

		confirmed, err := confirmPrompt("⚠️  Reinitializing will DELETE existing profile. Continue?")
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Aborted.")
			return nil
		}

		// Remove existing state
		if err := os.RemoveAll(stateDir); err != nil {
			return fmt.Errorf("remove existing state: %w", err)
		}
	}

	// Create state directory
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	fmt.Printf("Initializing Kairos profile at: %s\n\n", stateDir)

	// Display financial risk disclaimer
	if err := displayDisclaimer(cmd); err != nil {
		return err
	}

	// Create default configuration
	fmt.Println("\n📝 Creating default configuration...")
	configManager, err := config.NewManager(stateDir)
	if err != nil {
		return fmt.Errorf("create config manager: %w", err)
	}

	cfg := configManager.Current()
	fmt.Printf("   ✓ Config created: %s\n", cfg.Paths.ConfigPath)
	fmt.Printf("   ✓ Trades DB: %s\n", cfg.Paths.TradesDB)
	fmt.Printf("   ✓ Vault DB: %s\n", cfg.Paths.VaultDB)
	fmt.Printf("   ✓ Journal: %s\n", cfg.Paths.JournalPath)
	fmt.Printf("   ✓ Audit log: %s\n", cfg.Paths.AuditPath)

	// Create disclaimer acceptance record
	disclaimerPath := filepath.Join(stateDir, "disclaimer_accepted")
	if err := os.WriteFile(disclaimerPath, []byte("cli-init\n"), 0o600); err != nil {
		return fmt.Errorf("write disclaimer record: %w", err)
	}

	fmt.Println("\n✅ Initialization complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Set API keys: kairos (TUI) → /config ui")
	fmt.Println("  2. Configure risk limits: /set risk.max_position <amount>")
	fmt.Println("  3. Start paper trading: /start paper <symbol>")
	fmt.Println("\nRun 'kairos' to launch the TUI Command Center.")

	return nil
}

func displayDisclaimer(cmd *cobra.Command) error {
	acceptFlag, _ := cmd.Flags().GetBool("accept-disclaimer")

	disclaimer := `
╔════════════════════════════════════════════════════════════════════════════╗
║                     FINANCIAL RISK DISCLAIMER                              ║
╚════════════════════════════════════════════════════════════════════════════╝

⚠️  TRADING INVOLVES SUBSTANTIAL RISK OF LOSS

By using Kairos Trading Agent, you acknowledge and accept that:

1. FINANCIAL RISK: Trading cryptocurrencies and derivatives involves substantial
   risk of loss. You may lose some or all of your invested capital.

2. NO GUARANTEES: Past performance does not guarantee future results. No trading
   system can guarantee profits.

3. OPERATOR RESPONSIBILITY: You are solely responsible for:
   - All trading decisions and their outcomes
   - Monitoring positions and risk exposure
   - Ensuring compliance with applicable laws
   - Securing API keys and access credentials

4. SOFTWARE PROVIDED "AS IS": This software is provided without warranties of
   any kind. The developers are not liable for trading losses, system failures,
   or any damages arising from use of this software.

5. DEMO MODE RECOMMENDED: Start with demo or paper trading to understand the
   system before risking real capital.

6. LIVE TRADING REQUIRES: Explicit confirmation, accepted disclaimer, and
   operator supervision. Never run live trading unattended.

═══════════════════════════════════════════════════════════════════════════════

By proceeding, you confirm that you:
- Understand and accept these risks
- Are legally permitted to trade in your jurisdiction
- Will not hold the developers liable for any losses
- Have read and understood the documentation

`

	fmt.Print(disclaimer)

	if acceptFlag {
		fmt.Println("✓ Disclaimer accepted via --accept-disclaimer flag")
		return nil
	}

	confirmed, err := confirmPrompt("Do you accept the financial risk disclaimer?")
	if err != nil {
		return err
	}

	if !confirmed {
		return fmt.Errorf("disclaimer not accepted - initialization aborted")
	}

	fmt.Println("✓ Disclaimer accepted")
	return nil
}
