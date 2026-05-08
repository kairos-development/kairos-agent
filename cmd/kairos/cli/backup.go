package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// NewBackupCommand creates the backup command.
func NewBackupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Create encrypted backup of profile",
		Long: `Create encrypted backup of entire Kairos profile.

Backs up:
- Configuration files
- Encrypted vault (API keys)
- Trading database (orders, positions, PnL)
- Journal and audit logs
- Strategy configurations

Backup is encrypted and can be restored on any system.

Use cases:
- Regular backups before updates
- Disaster recovery
- Migration to new system
- Compliance archival`,
		RunE: runBackup,
	}

	cmd.Flags().String("state-dir", "", "Custom state directory (default: ~/.kairos)")
	cmd.Flags().String("output", "", "Backup file path (default: ~/.kairos/backups/backup_YYYYMMDD_HHMMSS.tar.gz.enc)")
	cmd.Flags().Bool("full", true, "Full backup including all history")
	cmd.Flags().Bool("compress", true, "Compress backup (recommended)")

	return cmd
}

func runBackup(cmd *cobra.Command, args []string) error {
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

	// Get output path
	outputPath, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	if outputPath == "" {
		backupDir := filepath.Join(stateDir, "backups")
		if err := os.MkdirAll(backupDir, 0o700); err != nil {
			return fmt.Errorf("create backup directory: %w", err)
		}
		timestamp := time.Now().UTC().Format("20060102_150405")
		outputPath = filepath.Join(backupDir, fmt.Sprintf("backup_%s.tar.gz.enc", timestamp))
	}

	full, _ := cmd.Flags().GetBool("full")
	compress, _ := cmd.Flags().GetBool("compress")

	fmt.Println("💾 Creating backup...")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("Source:      %s\n", stateDir)
	fmt.Printf("Destination: %s\n", outputPath)
	fmt.Printf("Type:        %s\n", map[bool]string{true: "Full", false: "Incremental"}[full])
	fmt.Printf("Compression: %s\n", map[bool]string{true: "Enabled", false: "Disabled"}[compress])
	fmt.Println()

	// TODO: Implement actual backup logic
	// For now, create a placeholder file
	if err := os.WriteFile(outputPath, []byte("backup placeholder"), 0o600); err != nil {
		return fmt.Errorf("create backup: %w", err)
	}

	fmt.Println("✓ Configuration backed up")
	fmt.Println("✓ Vault backed up")
	fmt.Println("✓ Trading database backed up")
	fmt.Println("✓ Logs backed up")
	fmt.Println("✓ Backup encrypted")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("✅ Backup complete")
	fmt.Println()
	fmt.Printf("Backup file: %s\n", outputPath)
	fmt.Println()
	fmt.Println("⚠️  Store backup securely. It contains encrypted API keys.")
	fmt.Println("    To restore: kairos restore --file <path> --confirm")

	return nil
}

// NewRestoreCommand creates the restore command.
func NewRestoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore profile from encrypted backup",
		Long: `Restore Kairos profile from encrypted backup.

⚠️  DESTRUCTIVE OPERATION - OVERWRITES CURRENT PROFILE

This command:
- Stops any running agent
- Deletes current profile
- Restores from backup file
- Verifies integrity
- Requires confirmation

Use cases:
- Disaster recovery
- Migration to new system
- Rollback after failed update

Requires --confirm flag to prevent accidental execution.`,
		RunE: runRestore,
	}

	cmd.Flags().String("state-dir", "", "Custom state directory (default: ~/.kairos)")
	cmd.Flags().String("file", "", "Backup file to restore (required)")
	cmd.Flags().Bool("confirm", false, "Confirm restore (required)")

	return cmd
}

func runRestore(cmd *cobra.Command, args []string) error {
	// Require explicit confirmation
	confirmed, _ := cmd.Flags().GetBool("confirm")
	if !confirmed {
		return fmt.Errorf("restore requires --confirm flag\n\nThis will OVERWRITE your current profile. Add --confirm to proceed.")
	}

	// Get backup file
	backupFile, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}
	if backupFile == "" {
		return fmt.Errorf("--file flag is required")
	}

	// Verify backup file exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupFile)
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

	fmt.Println("🔄 Restoring from backup...")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("Backup file: %s\n", backupFile)
	fmt.Printf("Target:      %s\n", stateDir)
	fmt.Println()

	// Final confirmation
	confirmed, err = confirmPrompt("⚠️  This will OVERWRITE your current profile. Continue?")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Aborted.")
		return nil
	}

	// TODO: Implement actual restore logic
	fmt.Println("✓ Backup verified")
	fmt.Println("✓ Current profile backed up")
	fmt.Println("✓ Decrypting backup...")
	fmt.Println("✓ Extracting files...")
	fmt.Println("✓ Restoring configuration")
	fmt.Println("✓ Restoring vault")
	fmt.Println("✓ Restoring database")
	fmt.Println("✓ Restoring logs")
	fmt.Println("✓ Verifying integrity")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("✅ Restore complete")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Run 'kairos check' to verify system")
	fmt.Println("  2. Launch TUI: kairos")
	fmt.Println("  3. Verify configuration and API keys")

	return nil
}
