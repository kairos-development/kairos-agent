package conv

import (
	"strings"

	"github.com/kairos-development/kairos-agent/internal/controllers/tui/dto"
)

// ParsedCommand is the normalized TUI command model used by the controller.
type ParsedCommand struct {
	Name string
	Args []string
}

// ParseCommand normalizes a raw TUI command into a controller command model.
func ParseCommand(input dto.CommandInput) ParsedCommand {
	fields := strings.Fields(strings.TrimSpace(input.Raw))
	if len(fields) == 0 {
		return ParsedCommand{}
	}
	return ParsedCommand{Name: strings.ToLower(fields[0]), Args: fields[1:]}
}
