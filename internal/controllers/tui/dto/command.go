package dto

// CommandInput is the raw presentation command entered in the TUI prompt.
type CommandInput struct {
	Raw string
}

// ViewModel is the presentation model rendered by the TUI layer.
type ViewModel struct {
	Header string
	Body   []string
	Prompt string
}
