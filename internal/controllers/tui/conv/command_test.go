package conv

import (
	"testing"

	"github.com/kairos-development/kairos-agent/internal/controllers/tui/dto"
	"github.com/stretchr/testify/assert"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want ParsedCommand
	}{
		{name: "empty", raw: "   ", want: ParsedCommand{}},
		{name: "lowercases command", raw: " /START paper BTCUSDT ", want: ParsedCommand{Name: "/start", Args: []string{"paper", "BTCUSDT"}}},
		{name: "keeps args", raw: "/set max_position 1000", want: ParsedCommand{Name: "/set", Args: []string{"max_position", "1000"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseCommand(dto.CommandInput{Raw: tt.raw}))
		})
	}
}
