package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/kairos-development/kairos-agent/internal/service/agent"
	"github.com/kairos-development/kairos-agent/internal/tui/actions"
)

// Command represents a parsed TUI command.
type Command struct {
	Name string
	Args []string
}

// Handler routes parsed commands to ActionService.
type Handler struct {
	actions actions.ActionService
}

// NewHandler creates a new command handler.
func NewHandler(actions actions.ActionService) *Handler {
	return &Handler{
		actions: actions,
	}
}

// Execute executes a command.
func (h *Handler) Execute(ctx context.Context, cmd Command) error {
	switch cmd.Name {
	case "start":
		return h.handleStart(ctx, cmd.Args)
	case "stop", "stopall":
		return h.actions.StopAll(ctx)
	case "backtest":
		return h.actions.QueueBacktest(ctx)
	case "set":
		return h.handleSet(ctx, cmd.Args)
	case "export":
		return h.handleExport(ctx, cmd.Args)
	default:
		return fmt.Errorf("unknown command: %s", cmd.Name)
	}
}

// handleStart handles start commands.
func (h *Handler) handleStart(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("start requires mode argument: paper, scan, or live")
	}

	mode := args[0]
	switch mode {
	case "paper":
		return h.actions.StartPaper(ctx)
	case "scan":
		return h.actions.StartScan(ctx)
	case "live":
		return h.actions.StartLive(ctx)
	default:
		return fmt.Errorf("unknown mode: %s (use paper, scan, or live)", mode)
	}
}

// handleSet handles configuration updates.
func (h *Handler) handleSet(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("set requires key and value arguments")
	}

	key := args[0]
	value := strings.Join(args[1:], " ")

	return h.actions.UpdateConfig(ctx, key, value)
}

// handleExport handles CSV export.
func (h *Handler) handleExport(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("export requires destination path argument")
	}

	destPath := args[0]
	input := agent.ExportCSVInput{
		DestinationPath: destPath,
	}

	output, err := h.actions.ExportCSV(ctx, input)
	if err != nil {
		return err
	}

	// Output will be handled by the TUI to display the file path
	_ = output
	return nil
}

// Parse parses a command string into a Command.
func Parse(input string) (Command, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Command{}, fmt.Errorf("empty command")
	}

	// Remove leading slash if present
	if strings.HasPrefix(input, "/") {
		input = input[1:]
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return Command{}, fmt.Errorf("empty command")
	}

	return Command{
		Name: parts[0],
		Args: parts[1:],
	}, nil
}
