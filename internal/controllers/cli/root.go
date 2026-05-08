package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kairos-development/kairos-agent/internal/backup"
	"github.com/kairos-development/kairos-agent/internal/bootstrap"
	cliconv "github.com/kairos-development/kairos-agent/internal/controllers/cli/conv"
	"github.com/kairos-development/kairos-agent/internal/controllers/tui"
	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/spf13/cobra"
)

// NewRootCommand builds the Kairos CLI presentation layer.
func NewRootCommand() *cobra.Command {
	var stateDir string
	var vaultPassword string
	command := &cobra.Command{
		Use:   "kairos",
		Short: "Kairos agent command center",
		RunE: func(cmd *cobra.Command, _ []string) error {
			factory := bootstrap.Factory{StateDir: stateDir, VaultPassword: vaultPassword}
			container, err := factory.Open(cmd.Context())
			if err != nil {
				return err
			}
			defer container.Close()
			program := tea.NewProgram(tui.New(container.AgentService))
			_, err = program.Run()
			return err
		},
	}
	command.PersistentFlags().StringVar(&stateDir, "state-dir", cliconv.DefaultStateDir(), "Kairos state directory")
	command.PersistentFlags().StringVar(&vaultPassword, "vault-password", "secret", "Vault password")
	command.AddCommand(newInitCommand(&stateDir, &vaultPassword))
	command.AddCommand(newCheckCommand(&stateDir, &vaultPassword))
	command.AddCommand(newBackupCommand(&stateDir))
	command.AddCommand(newRestoreCommand(&stateDir))
	command.AddCommand(newExportCommand(&stateDir, &vaultPassword))
	command.AddCommand(newEmergencyStopCommand(&stateDir, &vaultPassword))
	command.AddCommand(newPluginTrustCommand(&stateDir, &vaultPassword))
	command.AddCommand(newHeadlessCommand(&stateDir, &vaultPassword))
	return command
}

func newInitCommand(stateDir *string, vaultPassword *string) *cobra.Command {
	var accept bool
	command := &cobra.Command{
		Use:   "init",
		Short: "Initialize Kairos state",
		RunE: func(cmd *cobra.Command, _ []string) error {
			factory := bootstrap.Factory{StateDir: *stateDir, VaultPassword: *vaultPassword}
			container, err := factory.Open(cmd.Context())
			if err != nil {
				return err
			}
			defer container.Close()
			fmt.Println(entity.FinancialRiskDisclaimer)
			if !accept {
				fmt.Print("Type YES to accept the financial risk disclaimer for live trading: ")
				reader := bufio.NewReader(os.Stdin)
				line, _ := reader.ReadString('\n')
				accept = strings.TrimSpace(line) == "YES"
			}
			if accept {
				if err := container.AgentService.AcceptDisclaimer(cmd.Context(), "kairos init"); err != nil {
					return err
				}
			}
			fmt.Printf("initialized %s\n", *stateDir)
			return nil
		},
	}
	command.Flags().BoolVar(&accept, "accept-financial-risk", false, "Accept the financial risk disclaimer during initialization")
	return command
}

func newCheckCommand(stateDir *string, vaultPassword *string) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Print runtime, telemetry, and integrity status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			factory := bootstrap.Factory{StateDir: *stateDir, VaultPassword: *vaultPassword}
			container, err := factory.Open(cmd.Context())
			if err != nil {
				return err
			}
			defer container.Close()
			check, err := container.AgentService.Check(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Print(cliconv.FormatCheckOutput(cliconv.CheckOutputToDTO(check)))
			return nil
		},
	}
}

func newBackupCommand(stateDir *string) *cobra.Command {
	var password string
	command := &cobra.Command{
		Use:   "backup",
		Short: "Create an encrypted profile backup",
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := backup.Create(*stateDir, password)
			if err != nil {
				return err
			}
			fmt.Printf("archive=%s\nsha256=%s\n", result.ArchivePath, result.Checksum)
			return nil
		},
	}
	command.Flags().StringVar(&password, "password", "secret", "Backup encryption password")
	command.Flags().Bool("full", true, "Create a full profile backup")
	return command
}

func newRestoreCommand(stateDir *string) *cobra.Command {
	var filePath string
	var password string
	var confirm bool
	command := &cobra.Command{
		Use:   "restore",
		Short: "Restore an encrypted profile backup",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !confirm {
				return errors.New("restore requires --confirm")
			}
			return backup.Restore(*stateDir, filePath, password)
		},
	}
	command.Flags().StringVar(&filePath, "file", "", "Backup archive path")
	command.Flags().StringVar(&password, "password", "secret", "Backup decryption password")
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm destructive restore")
	command.MarkFlagRequired("file")
	return command
}

func newExportCommand(stateDir *string, vaultPassword *string) *cobra.Command {
	var format string
	command := &cobra.Command{
		Use:   "export",
		Short: "Export trade history",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if format != "csv" {
				return fmt.Errorf("unsupported export format %q", format)
			}
			factory := bootstrap.Factory{StateDir: *stateDir, VaultPassword: *vaultPassword}
			container, err := factory.Open(cmd.Context())
			if err != nil {
				return err
			}
			defer container.Close()
			path := filepath.Join(*stateDir, cliconv.DefaultExportFilename(container.Application.Runtime.Snapshot().LastUpdatedAtUTC))
			result, err := container.AgentService.ExportCSV(cmd.Context(), serviceagent.ExportCSVInput{DestinationPath: path})
			if err != nil {
				return err
			}
			fmt.Println(result.Path)
			return nil
		},
	}
	command.Flags().StringVar(&format, "format", "csv", "Export format")
	return command
}

func newEmergencyStopCommand(stateDir *string, vaultPassword *string) *cobra.Command {
	var confirm bool
	command := &cobra.Command{
		Use:   "emergency-stop",
		Short: "Bypass queues and halt the runtime immediately",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !confirm {
				return errors.New("emergency-stop requires --confirm")
			}
			factory := bootstrap.Factory{StateDir: *stateDir, VaultPassword: *vaultPassword}
			container, err := factory.Open(cmd.Context())
			if err != nil {
				return err
			}
			defer container.Close()
			if err := container.AgentService.EmergencyStop(cmd.Context()); err != nil {
				return err
			}
			fmt.Println("runtime halted")
			return nil
		},
	}
	command.Flags().BoolVar(&confirm, "confirm", false, "Confirm emergency stop")
	return command
}

func newPluginTrustCommand(stateDir *string, vaultPassword *string) *cobra.Command {
	var pubkeyPath string
	command := &cobra.Command{Use: "plugin", Short: "Manage plugin trust relationships"}
	subcommand := &cobra.Command{
		Use:   "trust <fingerprint>",
		Short: "Trust a plugin signing key already listed in config.yaml",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			factory := bootstrap.Factory{StateDir: *stateDir, VaultPassword: *vaultPassword}
			container, err := factory.Open(cmd.Context())
			if err != nil {
				return err
			}
			defer container.Close()
			publicKey, err := os.ReadFile(pubkeyPath)
			if err != nil {
				return err
			}
			return container.AgentService.TrustPluginKey(cmd.Context(), serviceagent.TrustPluginKeyInput{Fingerprint: args[0], PublicKey: publicKey})
		},
	}
	subcommand.Flags().StringVar(&pubkeyPath, "pubkey-file", "", "Public key file to trust")
	subcommand.MarkFlagRequired("pubkey-file")
	command.AddCommand(subcommand)
	return command
}

func newHeadlessCommand(stateDir *string, vaultPassword *string) *cobra.Command {
	var live bool
	var scan bool
	command := &cobra.Command{
		Use:   "headless",
		Short: "Run the agent without TUI output",
		RunE: func(cmd *cobra.Command, _ []string) error {
			factory := bootstrap.Factory{StateDir: *stateDir, VaultPassword: *vaultPassword}
			container, err := factory.Open(cmd.Context())
			if err != nil {
				return err
			}
			defer container.Close()
			switch {
			case live:
				if err := container.AgentService.StartLive(cmd.Context()); err != nil {
					return err
				}
			case scan:
				if err := container.AgentService.StartScan(cmd.Context()); err != nil {
					return err
				}
			default:
				if err := container.AgentService.StopAll(cmd.Context()); err != nil {
					return err
				}
			}
			check, err := container.AgentService.Check(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Print(cliconv.FormatCheckOutput(cliconv.CheckOutputToDTO(check)))
			return nil
		},
	}
	command.Flags().BoolVar(&live, "live", false, "Start in live trading mode")
	command.Flags().BoolVar(&scan, "scan", false, "Start in scanning mode")
	return command
}
