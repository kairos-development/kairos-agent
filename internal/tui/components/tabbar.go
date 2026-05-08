package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/kairos-development/kairos-agent/internal/tui/styles"
)

// TabBar displays navigation tabs for different views.
type TabBar struct {
	activeTab int
	tabs      []Tab
	width     int
}

// Tab represents a single tab.
type Tab struct {
	Key   string
	Label string
}

// NewTabBar creates a new tab bar.
func NewTabBar() *TabBar {
	return &TabBar{
		activeTab: 0,
		tabs: []Tab{
			{Key: "F1", Label: "Dashboard"},
			{Key: "F2", Label: "Market"},
			{Key: "F3", Label: "Positions"},
			{Key: "F4", Label: "Balance"},
			{Key: "F5", Label: "Logs"},
			{Key: "F6", Label: "Settings"},
		},
		width: 80,
	}
}

// SetActiveTab sets the active tab index.
func (t *TabBar) SetActiveTab(index int) {
	if index >= 0 && index < len(t.tabs) {
		t.activeTab = index
	}
}

// SetWidth sets the tab bar width.
func (t *TabBar) SetWidth(width int) {
	t.width = width
}

// Render renders the tab bar.
func (t *TabBar) Render() string {
	var tabs []string

	for i, tab := range t.tabs {
		var style lipgloss.Style
		if i == t.activeTab {
			style = tabActiveStyle
		} else {
			style = tabInactiveStyle
		}

		label := tab.Key + " " + tab.Label
		tabs = append(tabs, style.Render(label))
	}

	tabRow := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Add help text on the right (context-sensitive)
	var helpText string
	switch t.activeTab {
	case 4: // Logs view
		helpText = tabHelpStyle.Render("I/W/E Filter | A All | T Time | PgUp/Dn Scroll")
	default:
		helpText = tabHelpStyle.Render("? Help | Ctrl+←/→ Switch | /help Commands")
	}

	// Calculate spacing
	tabRowWidth := lipgloss.Width(tabRow)
	helpWidth := lipgloss.Width(helpText)
	spacing := t.width - tabRowWidth - helpWidth - 2
	if spacing < 0 {
		spacing = 0
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		tabRow,
		lipgloss.NewStyle().Width(spacing).Render(""),
		helpText,
	)
}

// Tab styles
var (
	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(styles.ColorPrimary).
			Background(lipgloss.Color("#1a1a1a")).
			Padding(0, 2).
			MarginRight(1)

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(styles.ColorMuted).
				Background(lipgloss.Color("#0a0a0a")).
				Padding(0, 2).
				MarginRight(1)

	tabHelpStyle = lipgloss.NewStyle().
			Foreground(styles.ColorMuted).
			Italic(true)
)
