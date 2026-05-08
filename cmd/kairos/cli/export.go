package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/kairos-development/kairos-agent/internal/config"
	configprovider "github.com/kairos-development/kairos-agent/internal/providers/config"
	runtimeprovider "github.com/kairos-development/kairos-agent/internal/providers/runtime"
	"github.com/kairos-development/kairos-agent/internal/runtime"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewExportCommand creates the export command.
func NewExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export trading data to CSV",
		Long: `Export trading data for analysis, reporting, or tax purposes.

Exports:
- All executed trades
- Position history
- PnL calculations
- Timestamps in configured timezone

Output format: CSV (Excel-compatible)

Use cases:
- Tax reporting
- Performance analysis
- External backtesting
- Compliance audits`,
		RunE: runExport,
	}

	cmd.Flags().String("state-dir", "", "Custom state directory (default: ~/.kairos)")
	cmd.Flags().String("output", "", "Output file path (default: ~/.kairos/export_YYYYMMDD_HHMMSS.csv)")
	cmd.Flags().String("format", "csv", "Export format (currently only csv supported)")

	return cmd
}

func runExport(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

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
		// Default: state_dir/export_YYYYMMDD_HHMMSS.csv
		timestamp := time.Now().UTC().Format("20060102_150405")
		outputPath = filepath.Join(stateDir, fmt.Sprintf("export_%s.csv", timestamp))
	}

	// Get format
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return err
	}
	if format != "csv" {
		return fmt.Errorf("unsupported format: %s (only csv supported)", format)
	}

	fmt.Println("📊 Exporting trading data...")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	// Load configuration
	configManager, err := config.NewManager(stateDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Create service
	runtimeManager := runtime.NewManager("export-cmd", logrus.New())
	configRepo := configprovider.New(configManager)
	runtimeRepo := runtimeprovider.New(runtimeManager)

	svc := serviceagent.New(
		runtimeRepo,
		configRepo,
		&mockDisclaimerRepo{},
		&mockExportRepo{},
		&mockPluginTrustRepo{},
	)

	// Execute export
	input := serviceagent.ExportCSVInput{
		DestinationPath: outputPath,
	}

	result, err := svc.ExportCSV(ctx, input)
	if err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	fmt.Printf("✓ Export complete\n")
	fmt.Printf("  Output: %s\n", result.Path)
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("✅ Data exported successfully")

	return nil
}
