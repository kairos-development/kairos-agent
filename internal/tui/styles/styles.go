package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	ColorPrimary    = lipgloss.Color("#00D9FF")
	ColorSuccess    = lipgloss.Color("#00FF88")
	ColorWarning    = lipgloss.Color("#FFB800")
	ColorDanger     = lipgloss.Color("#FF4444")
	ColorMuted      = lipgloss.Color("#666666")
	ColorBackground = lipgloss.Color("#1a1a1a")
	ColorForeground = lipgloss.Color("#ffffff")

	// Base styles
	BaseStyle = lipgloss.NewStyle().
			Foreground(ColorForeground).
			Background(ColorBackground)

	// Top bar styles
	TopBarStyle = lipgloss.NewStyle().
			Foreground(ColorForeground).
			Background(lipgloss.Color("#2a2a2a")).
			Padding(0, 1).
			Bold(true)

	StatusIdleStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Bold(true)

	StatusActiveStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	StatusWarningStyle = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true)

	StatusDangerStyle = lipgloss.NewStyle().
				Foreground(ColorDanger).
				Bold(true)

	// Bottom bar styles
	BottomBarStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Background(lipgloss.Color("#2a2a2a")).
			Padding(0, 1)

	// Command input styles
	CommandInputStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)

	CommandPromptStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	// Log styles
	LogInfoStyle = lipgloss.NewStyle().
			Foreground(ColorForeground)

	LogWarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	LogErrorStyle = lipgloss.NewStyle().
			Foreground(ColorDanger)

	LogSuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(ColorMuted)

	TableCellStyle = lipgloss.NewStyle().
			Foreground(ColorForeground).
			Padding(0, 1)

	// Border styles
	BorderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(1, 2)

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Padding(0, 1)

	// Help styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)
)
