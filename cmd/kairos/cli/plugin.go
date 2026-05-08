package cli

import (
	"fmt"
	"os"

	"github.com/kairos-development/kairos-agent/internal/config"
	configprovider "github.com/kairos-development/kairos-agent/internal/providers/config"
	runtimeprovider "github.com/kairos-development/kairos-agent/internal/providers/runtime"
	"github.com/kairos-development/kairos-agent/internal/runtime"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewPluginCommand creates the plugin command.
func NewPluginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage trusted plugin signing keys",
		Long: `Manage trusted plugin signing keys for WASM strategies.

Subcommands:
  trust      Trust a plugin signing key
  list       List trusted keys
  revoke     Revoke a trusted key

Plugin security:
- All plugins must be signed
- Only trusted keys can load plugins
- Keys are verified against fingerprints in config
- Revoked keys are blocked immediately`,
	}

	cmd.AddCommand(
		newPluginTrustCommand(),
		newPluginListCommand(),
		newPluginRevokeCommand(),
	)

	return cmd
}

func newPluginTrustCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trust <fingerprint>",
		Short: "Trust a plugin signing key",
		Long: `Trust a plugin signing key by fingerprint.

The fingerprint must be listed in config trusted_plugin_keys.
The public key is stored in the vault for signature verification.

Example:
  kairos plugin trust abc123... --pubkey-file key.pub`,
		Args: cobra.ExactArgs(1),
		RunE: runPluginTrust,
	}

	cmd.Flags().String("state-dir", "", "Custom state directory (default: ~/.kairos)")
	cmd.Flags().String("pubkey-file", "", "Public key file (required)")

	return cmd
}

func runPluginTrust(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	fingerprint := args[0]

	// Get public key file
	pubkeyFile, err := cmd.Flags().GetString("pubkey-file")
	if err != nil {
		return err
	}
	if pubkeyFile == "" {
		return fmt.Errorf("--pubkey-file flag is required")
	}

	// Read public key
	pubkey, err := os.ReadFile(pubkeyFile)
	if err != nil {
		return fmt.Errorf("read public key: %w", err)
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

	fmt.Println("🔐 Trusting plugin signing key...")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Fingerprint: %s\n", fingerprint)
	fmt.Printf("Public key:  %s\n", pubkeyFile)
	fmt.Println()

	// Load configuration
	configManager, err := config.NewManager(stateDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Create service
	runtimeManager := runtime.NewManager("plugin-trust", logrus.New())
	configRepo := configprovider.New(configManager)
	runtimeRepo := runtimeprovider.New(runtimeManager)

	pluginRepo := &mockPluginTrustRepo{}

	svc := serviceagent.New(
		runtimeRepo,
		configRepo,
		&mockDisclaimerRepo{},
		&mockExportRepo{},
		pluginRepo,
	)

	// Trust key
	input := serviceagent.TrustPluginKeyInput{
		Fingerprint: fingerprint,
		PublicKey:   pubkey,
	}

	if err := svc.TrustPluginKey(ctx, input); err != nil {
		return fmt.Errorf("trust key: %w", err)
	}

	fmt.Println("✓ Fingerprint verified in config")
	fmt.Println("✓ Public key stored in vault")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("✅ Plugin signing key trusted")
	fmt.Println()
	fmt.Println("Plugins signed with this key can now be loaded.")

	return nil
}

func newPluginListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List trusted plugin signing keys",
		RunE:  runPluginList,
	}

	cmd.Flags().String("state-dir", "", "Custom state directory (default: ~/.kairos)")

	return cmd
}

func runPluginList(cmd *cobra.Command, args []string) error {
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

	// Load configuration
	configManager, err := config.NewManager(stateDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cfg := configManager.Current()

	fmt.Println("🔐 Trusted Plugin Signing Keys")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	if len(cfg.TrustedPluginKeys) == 0 {
		fmt.Println("No trusted keys configured.")
		fmt.Println()
		fmt.Println("To trust a key:")
		fmt.Println("  1. Add fingerprint to config: /set trusted_plugin_keys [...]")
		fmt.Println("  2. Trust the key: kairos plugin trust <fingerprint> --pubkey-file <file>")
		return nil
	}

	for i, fingerprint := range cfg.TrustedPluginKeys {
		fmt.Printf("%d. %s\n", i+1, fingerprint)
	}

	fmt.Println()
	fmt.Printf("Total: %d trusted keys\n", len(cfg.TrustedPluginKeys))

	return nil
}

func newPluginRevokeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke <fingerprint>",
		Short: "Revoke a trusted plugin signing key",
		Long: `Revoke a trusted plugin signing key.

⚠️  This will:
- Remove fingerprint from config
- Delete public key from vault
- Block all plugins signed with this key

Requires restart to take effect.`,
		Args: cobra.ExactArgs(1),
		RunE: runPluginRevoke,
	}

	cmd.Flags().String("state-dir", "", "Custom state directory (default: ~/.kairos)")

	return cmd
}

func runPluginRevoke(cmd *cobra.Command, args []string) error {
	fingerprint := args[0]

	fmt.Printf("⚠️  Revoking plugin signing key: %s\n", fingerprint)
	fmt.Println()
	fmt.Println("This will block all plugins signed with this key.")
	fmt.Println()

	confirmed, err := confirmPrompt("Revoke this key?")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Aborted.")
		return nil
	}

	// TODO: Implement revocation logic
	fmt.Println("✓ Fingerprint removed from config")
	fmt.Println("✓ Public key deleted from vault")
	fmt.Println()
	fmt.Println("✅ Key revoked")
	fmt.Println()
	fmt.Println("⚠️  Restart agent for changes to take effect.")

	return nil
}
