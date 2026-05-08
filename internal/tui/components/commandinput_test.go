package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewCommandInput(t *testing.T) {
	ci := NewCommandInput()
	assert.NotNil(t, ci)
	assert.Empty(t, ci.history)
	assert.Equal(t, -1, ci.historyIndex)
	assert.False(t, ci.showSuggestions)
}

func TestCommandInput_SetWidth(t *testing.T) {
	ci := NewCommandInput()
	ci.SetWidth(100)
	assert.Equal(t, 100, ci.width)
	assert.Equal(t, 96, ci.textInput.Width) // 100 - 4
}

func TestCommandInput_Value(t *testing.T) {
	ci := NewCommandInput()
	ci.SetValue("test command")
	assert.Equal(t, "test command", ci.Value())
}

func TestCommandInput_SetValue(t *testing.T) {
	ci := NewCommandInput()
	ci.SetValue("/help")
	assert.Equal(t, "/help", ci.Value())
}

func TestCommandInput_Reset(t *testing.T) {
	ci := NewCommandInput()
	ci.SetValue("test")
	ci.historyIndex = 5
	ci.showSuggestions = true

	ci.Reset()

	assert.Empty(t, ci.Value())
	assert.Equal(t, -1, ci.historyIndex)
	assert.False(t, ci.showSuggestions)
}

func TestCommandInput_ParseCommand_Simple(t *testing.T) {
	ci := NewCommandInput()
	cmd := ci.parseCommand("/help")

	assert.Equal(t, "/help", cmd.Name)
	assert.Empty(t, cmd.Args)
}

func TestCommandInput_ParseCommand_WithArgs(t *testing.T) {
	ci := NewCommandInput()
	cmd := ci.parseCommand("/start live BTCUSDT")

	assert.Equal(t, "/start", cmd.Name)
	assert.Equal(t, []string{"live", "BTCUSDT"}, cmd.Args)
}

func TestCommandInput_ParseCommand_Empty(t *testing.T) {
	ci := NewCommandInput()
	cmd := ci.parseCommand("")

	assert.Empty(t, cmd.Name)
	assert.Empty(t, cmd.Args)
}

func TestCommandInput_ParseCommand_MultipleSpaces(t *testing.T) {
	ci := NewCommandInput()
	cmd := ci.parseCommand("/set   max_position   1000")

	assert.Equal(t, "/set", cmd.Name)
	assert.Equal(t, []string{"max_position", "1000"}, cmd.Args)
}

func TestCommandInput_HistoryUp_Empty(t *testing.T) {
	ci := NewCommandInput()
	ci.historyUp()

	assert.Equal(t, -1, ci.historyIndex)
	assert.Empty(t, ci.Value())
}

func TestCommandInput_HistoryUp_WithHistory(t *testing.T) {
	ci := NewCommandInput()
	ci.history = []string{"/help", "/status", "/exit"}

	ci.historyUp()
	assert.Equal(t, 2, ci.historyIndex)
	assert.Equal(t, "/exit", ci.Value())

	ci.historyUp()
	assert.Equal(t, 1, ci.historyIndex)
	assert.Equal(t, "/status", ci.Value())

	ci.historyUp()
	assert.Equal(t, 0, ci.historyIndex)
	assert.Equal(t, "/help", ci.Value())

	// Can't go further up
	ci.historyUp()
	assert.Equal(t, 0, ci.historyIndex)
}

func TestCommandInput_HistoryDown_NoHistory(t *testing.T) {
	ci := NewCommandInput()
	ci.historyDown()

	assert.Equal(t, -1, ci.historyIndex)
}

func TestCommandInput_HistoryDown_WithHistory(t *testing.T) {
	ci := NewCommandInput()
	ci.history = []string{"/help", "/status", "/exit"}
	ci.historyIndex = 0

	ci.historyDown()
	assert.Equal(t, 1, ci.historyIndex)
	assert.Equal(t, "/status", ci.Value())

	ci.historyDown()
	assert.Equal(t, 2, ci.historyIndex)
	assert.Equal(t, "/exit", ci.Value())

	// Go past end
	ci.historyDown()
	assert.Equal(t, -1, ci.historyIndex)
	assert.Empty(t, ci.Value())
}

func TestCommandInput_GetSuggestions_Empty(t *testing.T) {
	ci := NewCommandInput()
	suggestions := ci.getSuggestions("")

	// Empty input returns all suggestions (no prefix filter)
	assert.NotEmpty(t, suggestions)
}

func TestCommandInput_GetSuggestions_Matching(t *testing.T) {
	ci := NewCommandInput()
	suggestions := ci.getSuggestions("/st")

	assert.NotEmpty(t, suggestions)
	assert.Contains(t, suggestions, "/start live BTCUSDT")
	assert.Contains(t, suggestions, "/start paper BTCUSDT")
	assert.Contains(t, suggestions, "/status")
	assert.Contains(t, suggestions, "/stop")
	assert.Contains(t, suggestions, "/stopall")
}

func TestCommandInput_GetSuggestions_ExactMatch(t *testing.T) {
	ci := NewCommandInput()
	suggestions := ci.getSuggestions("/help")

	assert.Len(t, suggestions, 1)
	assert.Equal(t, "/help", suggestions[0])
}

func TestCommandInput_GetSuggestions_NoMatch(t *testing.T) {
	ci := NewCommandInput()
	suggestions := ci.getSuggestions("/xyz")

	assert.Empty(t, suggestions)
}

func TestCommandInput_UpdateSuggestions_Empty(t *testing.T) {
	ci := NewCommandInput()
	ci.SetValue("")
	ci.updateSuggestions()

	assert.False(t, ci.showSuggestions)
	assert.Empty(t, ci.suggestions)
}

func TestCommandInput_UpdateSuggestions_WithInput(t *testing.T) {
	ci := NewCommandInput()
	ci.SetValue("/st")
	ci.updateSuggestions()

	assert.True(t, ci.showSuggestions)
	assert.NotEmpty(t, ci.suggestions)
	assert.Equal(t, 0, ci.selectedSugg)
}

func TestCommandInput_NextSuggestion(t *testing.T) {
	ci := NewCommandInput()
	ci.suggestions = []string{"/help", "/status", "/stop"}
	ci.showSuggestions = true
	ci.selectedSugg = 0

	ci.nextSuggestion()
	assert.Equal(t, 1, ci.selectedSugg)
	assert.Equal(t, "/status", ci.Value())

	ci.nextSuggestion()
	assert.Equal(t, 2, ci.selectedSugg)
	assert.Equal(t, "/stop", ci.Value())

	// Wrap around
	ci.nextSuggestion()
	assert.Equal(t, 0, ci.selectedSugg)
	assert.Equal(t, "/help", ci.Value())
}

func TestCommandInput_NextSuggestion_NoSuggestions(t *testing.T) {
	ci := NewCommandInput()
	ci.showSuggestions = false

	ci.nextSuggestion()
	assert.Equal(t, 0, ci.selectedSugg)
}

func TestCommandInput_PrevSuggestion(t *testing.T) {
	ci := NewCommandInput()
	ci.suggestions = []string{"/help", "/status", "/stop"}
	ci.showSuggestions = true
	ci.selectedSugg = 2

	ci.prevSuggestion()
	assert.Equal(t, 1, ci.selectedSugg)
	assert.Equal(t, "/status", ci.Value())

	ci.prevSuggestion()
	assert.Equal(t, 0, ci.selectedSugg)
	assert.Equal(t, "/help", ci.Value())

	// Wrap around
	ci.prevSuggestion()
	assert.Equal(t, 2, ci.selectedSugg)
	assert.Equal(t, "/stop", ci.Value())
}

func TestCommandInput_PrevSuggestion_NoSuggestions(t *testing.T) {
	ci := NewCommandInput()
	ci.showSuggestions = false

	ci.prevSuggestion()
	assert.Equal(t, 0, ci.selectedSugg)
}

func TestCommandInput_View_NoSuggestions(t *testing.T) {
	ci := NewCommandInput()
	ci.SetValue("/help")

	view := ci.View()
	assert.NotEmpty(t, view)
}

func TestCommandInput_View_WithSuggestions(t *testing.T) {
	ci := NewCommandInput()
	ci.suggestions = []string{"/help", "/status"}
	ci.showSuggestions = true
	ci.selectedSugg = 0

	view := ci.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "/help")
	assert.Contains(t, view, "/status")
}

func TestCommandInput_Update_KeyUp(t *testing.T) {
	ci := NewCommandInput()
	ci.history = []string{"/help"}

	msg := tea.KeyMsg{Type: tea.KeyUp}
	ci.Update(msg)

	assert.Equal(t, 0, ci.historyIndex)
	assert.Equal(t, "/help", ci.Value())
}

func TestCommandInput_Update_KeyDown(t *testing.T) {
	ci := NewCommandInput()
	ci.history = []string{"/help"}
	ci.historyIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyDown}
	ci.Update(msg)

	assert.Equal(t, -1, ci.historyIndex)
}

func TestCommandInput_Update_KeyTab(t *testing.T) {
	ci := NewCommandInput()
	ci.suggestions = []string{"/help", "/status"}
	ci.showSuggestions = true
	ci.selectedSugg = 0

	msg := tea.KeyMsg{Type: tea.KeyTab}
	ci.Update(msg)

	assert.Equal(t, 1, ci.selectedSugg)
}

func TestCommandInput_Update_KeyShiftTab(t *testing.T) {
	ci := NewCommandInput()
	ci.suggestions = []string{"/help", "/status"}
	ci.showSuggestions = true
	ci.selectedSugg = 1

	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	ci.Update(msg)

	assert.Equal(t, 0, ci.selectedSugg)
}

func TestCommandInput_Submit_Empty(t *testing.T) {
	ci := NewCommandInput()
	ci.SetValue("")

	cmd := ci.submit()
	assert.Nil(t, cmd)
}

func TestCommandInput_Submit_WithValue(t *testing.T) {
	ci := NewCommandInput()
	ci.SetValue("/help")

	cmd := ci.submit()
	assert.NotNil(t, cmd)

	// Check history was updated
	assert.Len(t, ci.history, 1)
	assert.Equal(t, "/help", ci.history[0])

	// Check input was reset
	assert.Empty(t, ci.Value())
}

func TestCommandInput_Submit_HistoryLimit(t *testing.T) {
	ci := NewCommandInput()

	// Add 101 commands
	for i := 0; i < 101; i++ {
		ci.SetValue("/help")
		ci.submit()
	}

	// Should be limited to 100
	assert.Len(t, ci.history, 100)
}

func TestExecuteCommand_Help(t *testing.T) {
	cmd := Command{Name: "/help"}
	cmdFunc := ExecuteCommand(cmd)

	msg := cmdFunc()
	result, ok := msg.(CommandResultMsg)
	assert.True(t, ok)
	assert.True(t, result.Success)
	assert.Contains(t, result.Message, "Available commands")
}

func TestExecuteCommand_Exit(t *testing.T) {
	cmd := Command{Name: "/exit"}
	cmdFunc := ExecuteCommand(cmd)

	msg := cmdFunc()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestExecuteCommand_Unknown(t *testing.T) {
	cmd := Command{Name: "/unknown"}
	cmdFunc := ExecuteCommand(cmd)

	msg := cmdFunc()
	result, ok := msg.(CommandResultMsg)
	assert.True(t, ok)
	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "Unknown command")
}

func TestCommandInput_Focus(t *testing.T) {
	ci := NewCommandInput()
	cmd := ci.Focus()
	assert.NotNil(t, cmd)
}

func TestCommandInput_Blur(t *testing.T) {
	ci := NewCommandInput()
	ci.Blur()
	// Just verify it doesn't panic
}
