package cli

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kairos-development/kairos-agent/internal/config"
	"github.com/kairos-development/kairos-agent/internal/domain/events"
	configprovider "github.com/kairos-development/kairos-agent/internal/providers/config"
	runtimeprovider "github.com/kairos-development/kairos-agent/internal/providers/runtime"
	"github.com/kairos-development/kairos-agent/internal/providers/storage/sqlite"
	"github.com/kairos-development/kairos-agent/internal/runtime"
	serviceagent "github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/kairos-development/kairos-agent/internal/tui"
	"github.com/kairos-development/kairos-agent/internal/tui/actions"
	"github.com/kairos-development/kairos-agent/internal/tui/query"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Version information (set by build flags)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// Execute runs the root command.
func Execute() error {
	rootCmd := NewRootCommand()
	return rootCmd.Execute()
}

// NewRootCommand creates the root kairos command.
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kairos",
		Short: "Kairos Trading Agent - Autonomous trading system",
		Long: `Kairos Trading Agent v1

A production-grade autonomous trading system with:
- Demo, paper, and live trading modes
- Strategy builder with no-code UI
- Risk management and position tracking
- Encrypted vault for API keys
- Audit logging and compliance`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default: launch TUI Command Center
			return runTUI(cmd.Context())
		},
	}

	// Add subcommands
	cmd.AddCommand(
		NewInitCommand(),
		NewCheckCommand(),
		NewHeadlessCommand(),
		NewBackupCommand(),
		NewRestoreCommand(),
		NewExportCommand(),
		NewEmergencyStopCommand(),
		NewPluginCommand(),
		NewUpdateCommand(),
		NewVersionCommand(),
	)

	return cmd
}

// runTUI launches the TUI Command Center.
func runTUI(ctx context.Context) error {
	// Get state directory
	stateDir, err := getStateDir()
	if err != nil {
		return fmt.Errorf("get state directory: %w", err)
	}

	// Load configuration
	configManager, err := config.NewManager(stateDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Initialize storage
	storageDB, err := sqlite.Open(stateDir + "/trades.db")
	if err != nil {
		return fmt.Errorf("initialize storage: %w", err)
	}
	defer storageDB.Close()

	// Create repositories
	configRepo := configprovider.New(configManager)
	runtimeManager := runtime.NewManager("tui", logrus.New())
	runtimeRepo := runtimeprovider.New(runtimeManager)

	// Create storage repositories
	positionRepo := sqlite.NewPositionRepository(storageDB)
	orderRepo := sqlite.NewOrderRepository(storageDB)
	balanceRepo := sqlite.NewBalanceRepository(storageDB)
	strategyRepo := sqlite.NewStrategyRepository(storageDB)

	// Create agent service
	agentService := serviceagent.New(
		runtimeRepo,
		configRepo,
		&mockDisclaimerRepo{},
		&mockExportRepo{},
		&mockPluginTrustRepo{},
	)

	// Create query service
	queryService := query.NewQueryService(
		runtimeRepo,
		positionRepo,
		balanceRepo,
		orderRepo,
		configRepo,
		strategyRepo,
	)

	// Create action service
	actionService := actions.NewActionService(agentService)

	model := tui.New(ctx, queryService, actionService)

	// Run TUI
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Create event bridge to connect domain events to TUI
	eventBridge := tui.NewEventBridge(p)

	// Create event publisher and subscribe the bridge
	eventPublisher := tui.NewEventPublisher()
	eventPublisher.Subscribe(func(event events.Event) {
		eventBridge.OnEvent(ctx, event)
	})

	// TODO: Wire up event publisher to domain event sources
	// For now, the event bridge is ready but not connected to actual events
	// This will be completed when integrating with the engine/connector

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run TUI: %w", err)
	}

	return nil
}

// getStateDir returns the state directory path.
func getStateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return home + "/.kairos", nil
}

// confirmPrompt asks for user confirmation.
func confirmPrompt(message string) (bool, error) {
	fmt.Printf("%s [y/N]: ", message)
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		return false, err
	}
	return response == "y" || response == "Y" || response == "yes" || response == "YES", nil
}
