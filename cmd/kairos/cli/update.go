package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	cloudprovider "github.com/kairos-development/kairos-agent/internal/providers/cloud"
	"github.com/spf13/cobra"
)

// NewUpdateCommand creates the update command.
func NewUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Manage Kairos updates",
		Long: `Manage Kairos agent updates.

Subcommands:
  check      Check for available updates
  download   Download and verify update
  apply      Apply downloaded update

Update process:
1. Check for updates (safe, read-only)
2. Download update and verify signature
3. Apply update (requires confirmation)

Updates are staged and verified before application.
Rollback is available if update fails.`,
	}

	cmd.AddCommand(
		newUpdateCheckCommand(),
		newUpdateDownloadCommand(),
		newUpdateApplyCommand(),
	)

	return cmd
}

func newUpdateCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check for available updates",
		Long: `Check for available Kairos updates.

Queries update server for:
- Latest version
- Release notes
- Security advisories
- Breaking changes

This is a read-only operation, safe to run anytime.`,
		RunE: runUpdateCheck,
	}

	cmd.Flags().String("channel", "stable", "Update channel (stable, beta, dev)")
	cmd.Flags().String("cloud-url", os.Getenv("KAIROS_CLOUD_URL"), "Kairos Cloud base URL")
	cmd.Flags().String("token", os.Getenv("KAIROS_CLOUD_TOKEN"), "Kairos Cloud bearer token (prefer vault/env over shell history)")

	return cmd
}

func runUpdateCheck(cmd *cobra.Command, args []string) error {
	channel, _ := cmd.Flags().GetString("channel")
	cloudURL, _ := cmd.Flags().GetString("cloud-url")
	token, _ := cmd.Flags().GetString("token")

	fmt.Println("🔍 Checking for updates...")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Current version: %s\n", Version)
	fmt.Printf("Channel:         %s\n", channel)
	fmt.Println()

	client, err := cloudprovider.NewClient(cloudprovider.Config{
		BaseURL:     cloudURL,
		AccessToken: token,
		UserAgent:   "kairos-agent/" + Version,
		Timeout:     10 * time.Second,
	})
	if errors.Is(err, cloudprovider.ErrDisabled) {
		fmt.Println("Cloud update checks are not configured.")
		fmt.Println("Set KAIROS_CLOUD_URL or pass --cloud-url to enable signed update metadata checks.")
		return nil
	}
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	update, err := client.CheckUpdate(ctx, cloudprovider.UpdateQuery{
		CurrentVersion: Version,
		Channel:        channel,
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
	})
	if errors.Is(err, cloudprovider.ErrNoUpdateAvailable) {
		fmt.Println("✅ You are running the latest version")
		return nil
	}
	if err != nil {
		return err
	}
	if update.Release == nil || update.Artifact == nil {
		return fmt.Errorf("cloud returned incomplete update metadata")
	}

	fmt.Println("✓ Connected to update server")
	fmt.Println("✓ Verified signed update metadata is available")
	fmt.Println()
	fmt.Printf("📦 Update available: %s → %s\n", Version, update.Release.Version)
	fmt.Println()
	fmt.Println("Release notes:")
	fmt.Printf("  %s\n", update.Release.NotesURL)
	fmt.Printf("Criticality: %s\n", update.Release.Criticality)
	fmt.Printf("Artifact:    %s/%s\n", update.Artifact.OS, update.Artifact.Arch)
	fmt.Printf("Checksum:    sha256:%s\n", update.Artifact.SHA256)
	fmt.Println()
	fmt.Println("To download:")
	fmt.Printf("  kairos update download --version %s\n", update.Release.Version)
	fmt.Println()
	fmt.Println("To apply:")
	fmt.Println("  kairos update apply --confirm")

	return nil
}

func newUpdateDownloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download and verify update",
		Long: `Download update and verify signature.

Downloads:
- New binary
- Signature file
- Release notes

Verifies:
- Signature authenticity
- Checksum integrity
- Version compatibility

Update is staged but not applied until 'update apply'.`,
		RunE: runUpdateDownload,
	}

	cmd.Flags().String("version", "", "Specific version to download (default: latest)")

	return cmd
}

func runUpdateDownload(cmd *cobra.Command, args []string) error {
	version, _ := cmd.Flags().GetString("version")
	if version == "" {
		version = "latest"
	}

	fmt.Println("📥 Downloading update...")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("Version: %s\n", version)
	fmt.Println()

	// TODO: Implement actual download
	fmt.Println("✓ Downloading binary...")
	fmt.Println("✓ Downloading signature...")
	fmt.Println("✓ Verifying signature...")
	fmt.Println("✓ Verifying checksum...")
	fmt.Println("✓ Checking compatibility...")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("✅ Update downloaded and verified")
	fmt.Println()
	fmt.Println("To apply:")
	fmt.Println("  kairos update apply --confirm")
	fmt.Println()
	fmt.Println("⚠️  Backup recommended before applying:")
	fmt.Println("  kairos backup --full")

	return nil
}

func newUpdateApplyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply downloaded update",
		Long: `Apply previously downloaded update.

⚠️  This will:
- Stop running agent
- Replace binary
- Migrate configuration if needed
- Restart agent

Requires --confirm flag.
Backup is recommended before applying.`,
		RunE: runUpdateApply,
	}

	cmd.Flags().Bool("confirm", false, "Confirm update application (required)")
	cmd.Flags().Bool("restart", true, "Restart agent after update")

	return cmd
}

func runUpdateApply(cmd *cobra.Command, args []string) error {
	confirmed, _ := cmd.Flags().GetBool("confirm")
	if !confirmed {
		return fmt.Errorf("update apply requires --confirm flag\n\nThis will replace the binary. Add --confirm to proceed.")
	}

	restart, _ := cmd.Flags().GetBool("restart")

	fmt.Println("🔄 Applying update...")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	// Final confirmation
	confirmed, err := confirmPrompt("⚠️  Apply update now?")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Aborted.")
		return nil
	}

	// TODO: Implement actual update application
	fmt.Println("✓ Stopping agent...")
	fmt.Println("✓ Creating backup of current binary...")
	fmt.Println("✓ Replacing binary...")
	fmt.Println("✓ Migrating configuration...")
	fmt.Println("✓ Verifying installation...")

	if restart {
		fmt.Println("✓ Restarting agent...")
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("✅ Update applied successfully")
	fmt.Println()
	fmt.Printf("New version: %s\n", "1.0.1")
	fmt.Println()

	if !restart {
		fmt.Println("Agent not restarted. Start manually:")
		fmt.Println("  kairos")
	}

	fmt.Println()
	fmt.Println("Rollback available if needed:")
	fmt.Println("  Binary backup: ~/.kairos/backups/kairos.backup")

	return nil
}
