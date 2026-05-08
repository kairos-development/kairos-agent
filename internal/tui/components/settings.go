package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kairos-development/kairos-agent/internal/tui/styles"
	"github.com/kairos-development/kairos-agent/internal/tui/viewmodel"
)

// SettingsView displays configuration settings (read-only).
type SettingsView struct {
	vm     *viewmodel.Settings
	width  int
	height int
}

// NewSettingsView creates a new settings view.
func NewSettingsView() *SettingsView {
	return &SettingsView{}
}

// Update updates the settings view with new view model data.
func (sv *SettingsView) Update(vm *viewmodel.Settings) {
	sv.vm = vm
}

// SetSize sets the dimensions of the view.
func (sv *SettingsView) SetSize(width, height int) {
	sv.width = width
	sv.height = height
}

// Render renders the settings view.
func (sv *SettingsView) Render() string {
	if sv.vm == nil {
		return lipgloss.NewStyle().
			Width(sv.width).
			Height(sv.height).
			Render("Loading settings...")
	}

	// Title
	title := styles.TitleStyle.Render("Configuration (Read-Only)")

	// Content
	content := sv.renderContent()

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(1, 2).
		Width(sv.width - 4).
		Height(sv.height - 4).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, title, box)
}

// renderContent renders the settings content.
func (sv *SettingsView) renderContent() string {
	var sections []string

	// Risk Management section
	if len(sv.vm.Risk) > 0 {
		sections = append(sections, sv.renderSection("Risk Management", sv.vm.Risk))
	}

	// Trading section
	if len(sv.vm.Trading) > 0 {
		sections = append(sections, sv.renderSection("Trading", sv.vm.Trading))
	}

	// System section
	if len(sv.vm.System) > 0 {
		sections = append(sections, sv.renderSection("System", sv.vm.System))
	}

	// Footer help text
	helpStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted).Italic(true)
	footer := helpStyle.Render("\nUse /set <key> <value> to change settings")

	return strings.Join(sections, "\n\n") + footer
}

// renderSection renders a configuration section.
func (sv *SettingsView) renderSection(title string, items []viewmodel.ConfigItem) string {
	sectionStyle := lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	valueStyle := lipgloss.NewStyle().Foreground(styles.ColorSuccess)

	var lines []string
	lines = append(lines, sectionStyle.Render(title))

	for _, item := range items {
		line := fmt.Sprintf("  %s: %s",
			keyStyle.Render(padRight(item.Key, 25)),
			valueStyle.Render(item.Value),
		)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
