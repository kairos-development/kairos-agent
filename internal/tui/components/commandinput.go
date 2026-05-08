package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kairos-development/kairos-agent/internal/tui/styles"
)

// Command represents a parsed command.
type Command struct {
	Name string
	Args []string
}

// CommandInput handles command line input with autocomplete and history.
type CommandInput struct {
	textInput       textinput.Model
	history         []string
	historyIndex    int
	suggestions     []string
	selectedSugg    int
	showSuggestions bool
	width           int
}

// NewCommandInput creates a new command input.
func NewCommandInput() *CommandInput {
	ti := textinput.New()
	ti.Placeholder = "Type a command... (try /help)"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 80

	ti.PromptStyle = styles.CommandPromptStyle
	ti.TextStyle = styles.CommandInputStyle

	return &CommandInput{
		textInput:    ti,
		history:      make([]string, 0, 100),
		historyIndex: -1,
	}
}

// Update handles input events.
func (ci *CommandInput) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			ci.historyUp()
			return nil
		case tea.KeyDown:
			ci.historyDown()
			return nil
		case tea.KeyTab:
			ci.nextSuggestion()
			return nil
		case tea.KeyShiftTab:
			ci.prevSuggestion()
			return nil
		case tea.KeyEnter:
			return ci.submit()
		}
	}

	// Update suggestions based on input
	ci.updateSuggestions()

	var cmd tea.Cmd
	ci.textInput, cmd = ci.textInput.Update(msg)
	return cmd
}

// View renders the command input.
func (ci *CommandInput) View() string {
	input := ci.textInput.View()

	if ci.showSuggestions && len(ci.suggestions) > 0 {
		// Render suggestions above the input
		suggLines := make([]string, 0, len(ci.suggestions))
		for i, sugg := range ci.suggestions {
			if i == ci.selectedSugg {
				suggLines = append(suggLines, styles.CommandInputStyle.Render("> "+sugg))
			} else {
				suggLines = append(suggLines, styles.HelpStyle.Render("  "+sugg))
			}
		}
		suggestions := strings.Join(suggLines, "\n")
		return suggestions + "\n" + input
	}

	return input
}

// SetWidth sets the width of the command input.
func (ci *CommandInput) SetWidth(width int) {
	ci.width = width
	ci.textInput.Width = width - 4
}

// Value returns the current input value.
func (ci *CommandInput) Value() string {
	return ci.textInput.Value()
}

// SetValue sets the input value.
func (ci *CommandInput) SetValue(value string) {
	ci.textInput.SetValue(value)
}

// Reset clears the input.
func (ci *CommandInput) Reset() {
	ci.textInput.SetValue("")
	ci.historyIndex = -1
	ci.showSuggestions = false
}

// Focus focuses the input.
func (ci *CommandInput) Focus() tea.Cmd {
	return ci.textInput.Focus()
}

// Blur removes focus from the input.
func (ci *CommandInput) Blur() {
	ci.textInput.Blur()
}

// submit handles command submission.
func (ci *CommandInput) submit() tea.Cmd {
	value := ci.textInput.Value()
	if value == "" {
		return nil
	}

	// Add to history
	ci.history = append(ci.history, value)
	if len(ci.history) > 100 {
		ci.history = ci.history[1:]
	}

	// Parse command
	cmd := ci.parseCommand(value)

	// Reset input
	ci.Reset()

	// Return command message
	return func() tea.Msg {
		return CommandSubmittedMsg{Command: cmd}
	}
}

// parseCommand parses a command string into a Command struct.
func (ci *CommandInput) parseCommand(input string) Command {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return Command{}
	}

	return Command{
		Name: parts[0],
		Args: parts[1:],
	}
}

// historyUp navigates up in command history.
func (ci *CommandInput) historyUp() {
	if len(ci.history) == 0 {
		return
	}

	if ci.historyIndex == -1 {
		ci.historyIndex = len(ci.history) - 1
	} else if ci.historyIndex > 0 {
		ci.historyIndex--
	}

	ci.textInput.SetValue(ci.history[ci.historyIndex])
}

// historyDown navigates down in command history.
func (ci *CommandInput) historyDown() {
	if ci.historyIndex == -1 {
		return
	}

	ci.historyIndex++
	if ci.historyIndex >= len(ci.history) {
		ci.historyIndex = -1
		ci.textInput.SetValue("")
	} else {
		ci.textInput.SetValue(ci.history[ci.historyIndex])
	}
}

// updateSuggestions updates the autocomplete suggestions.
func (ci *CommandInput) updateSuggestions() {
	input := ci.textInput.Value()
	if input == "" {
		ci.showSuggestions = false
		return
	}

	// Get suggestions based on input
	ci.suggestions = ci.getSuggestions(input)
	ci.showSuggestions = len(ci.suggestions) > 0
	ci.selectedSugg = 0
}

// getSuggestions returns autocomplete suggestions for the input.
func (ci *CommandInput) getSuggestions(input string) []string {
	commands := []string{
		"/help",
		"/start live BTCUSDT",
		"/start paper BTCUSDT",
		"/stop",
		"/stopall",
		"/scan ALL --strategy=trend",
		"/backtest MA_Cross 30d",
		"/set max_position 1000",
		"/config ui",
		"/status",
		"/positions",
		"/orders",
		"/balance",
		"/update",
		"/exit",
	}

	var suggestions []string
	for _, cmd := range commands {
		if strings.HasPrefix(cmd, input) {
			suggestions = append(suggestions, cmd)
		}
	}

	return suggestions
}

// nextSuggestion selects the next suggestion.
func (ci *CommandInput) nextSuggestion() {
	if !ci.showSuggestions || len(ci.suggestions) == 0 {
		return
	}

	ci.selectedSugg++
	if ci.selectedSugg >= len(ci.suggestions) {
		ci.selectedSugg = 0
	}

	ci.textInput.SetValue(ci.suggestions[ci.selectedSugg])
}

// prevSuggestion selects the previous suggestion.
func (ci *CommandInput) prevSuggestion() {
	if !ci.showSuggestions || len(ci.suggestions) == 0 {
		return
	}

	ci.selectedSugg--
	if ci.selectedSugg < 0 {
		ci.selectedSugg = len(ci.suggestions) - 1
	}

	ci.textInput.SetValue(ci.suggestions[ci.selectedSugg])
}

// CommandSubmittedMsg is sent when a command is submitted.
type CommandSubmittedMsg struct {
	Command Command
}

// CommandResultMsg is sent when a command execution completes.
type CommandResultMsg struct {
	Success bool
	Message string
	Error   error
}

// ExecuteCommand executes a command and returns a result message.
func ExecuteCommand(cmd Command) tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement actual command execution
		switch cmd.Name {
		case "/help":
			return CommandResultMsg{
				Success: true,
				Message: "Available commands: /start, /stop, /scan, /backtest, /set, /config, /status, /exit",
			}
		case "/exit":
			return tea.Quit()
		default:
			return CommandResultMsg{
				Success: false,
				Message: fmt.Sprintf("Unknown command: %s", cmd.Name),
			}
		}
	}
}
