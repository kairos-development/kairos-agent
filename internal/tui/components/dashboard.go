package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kairos-development/kairos-agent/internal/tui/styles"
	"github.com/kairos-development/kairos-agent/internal/tui/viewmodel"
)

// Dashboard displays the dashboard view with system status and metrics.
type Dashboard struct {
	vm     *viewmodel.Dashboard
	width  int
	height int
}

// NewDashboard creates a new dashboard component.
func NewDashboard() *Dashboard {
	return &Dashboard{}
}

// Update updates the dashboard with new view model data.
func (d *Dashboard) Update(vm *viewmodel.Dashboard) {
	d.vm = vm
}

// SetSize sets the dashboard size.
func (d *Dashboard) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// Render renders the dashboard.
func (d *Dashboard) Render() string {
	if d.vm == nil {
		return lipgloss.NewStyle().
			Width(d.width).
			Height(d.height).
			Render("Loading dashboard...")
	}

	var sections []string

	// System Status section
	sections = append(sections, d.renderSystemStatus())

	// Performance section
	sections = append(sections, d.renderPerformance())

	// Active Strategies section (if any)
	if len(d.vm.ActiveStrategies) > 0 {
		sections = append(sections, d.renderActiveStrategies())
	}

	// Active Orders section (if any)
	if len(d.vm.ActiveOrders) > 0 {
		sections = append(sections, d.renderActiveOrders())
	}

	// Recent Activity section (if any)
	if len(d.vm.RecentActivity) > 0 {
		sections = append(sections, d.renderRecentActivity())
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderSystemStatus renders the system status section.
func (d *Dashboard) renderSystemStatus() string {
	title := styles.TitleStyle.Render("System Status")

	content := fmt.Sprintf(
		"Mode: %s    Connectivity: %s\nLicense: %s     Uptime: %s",
		d.colorizeMode(d.vm.SystemStatus.Mode),
		d.colorizeConnectivity(d.vm.SystemStatus.Connectivity),
		d.colorizeLicense(d.vm.SystemStatus.License),
		d.vm.SystemStatus.Uptime,
	)
	if d.vm.SystemStatus.HaltReason != "" {
		haltLine := fmt.Sprintf("Halted: %s", d.vm.SystemStatus.HaltReason)
		if d.vm.SystemStatus.HaltedAt != "" {
			haltLine = fmt.Sprintf("%s at %s", haltLine, d.vm.SystemStatus.HaltedAt)
		}
		content += "\n" + styles.LogErrorStyle.Render(haltLine)
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(1, 2).
		Width(d.width - 4).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, title, box)
}

// renderPerformance renders the performance metrics section.
func (d *Dashboard) renderPerformance() string {
	title := styles.TitleStyle.Render("Performance")

	leftCol := fmt.Sprintf(
		"Total PnL:      %s\nDaily PnL:      %s\nOpen Positions: %d",
		d.colorizePnL(d.vm.Performance.TotalPnL),
		d.colorizePnL(d.vm.Performance.DailyPnL),
		d.vm.Performance.OpenPositions,
	)

	rightCol := fmt.Sprintf(
		"Win Rate:     %s\nTotal Trades: %d\nActive Orders: %d",
		d.vm.Performance.WinRate,
		d.vm.Performance.TotalTrades,
		d.vm.Performance.ActiveOrders,
	)

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width((d.width-8)/2).Render(leftCol),
		lipgloss.NewStyle().Width(4).Render("│"),
		lipgloss.NewStyle().Width((d.width-8)/2).Render(rightCol),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(1, 2).
		Width(d.width - 4).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, title, box)
}

// renderActiveStrategies renders the active strategies section.
func (d *Dashboard) renderActiveStrategies() string {
	title := styles.TitleStyle.Render("Active Strategies")

	var lines []string
	for _, strategy := range d.vm.ActiveStrategies {
		line := fmt.Sprintf(
			"%-20s %-10s PnL: %s",
			strategy.Name,
			strategy.Status,
			d.colorizePnL(strategy.PnL),
		)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(1, 2).
		Width(d.width - 4).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, title, box)
}

// renderActiveOrders renders the active orders section.
func (d *Dashboard) renderActiveOrders() string {
	title := styles.TitleStyle.Render("Active Orders (Recent 5)")

	var lines []string
	for _, order := range d.vm.ActiveOrders {
		line := fmt.Sprintf(
			"%-10s %-4s %-5s %8s %10s %-10s",
			order.Symbol,
			order.Side,
			order.Type,
			order.Qty,
			order.Price,
			order.Status,
		)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(1, 2).
		Width(d.width - 4).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, title, box)
}

// renderRecentActivity renders the recent activity section.
func (d *Dashboard) renderRecentActivity() string {
	title := styles.TitleStyle.Render("Recent Activity")

	var lines []string
	for _, activity := range d.vm.RecentActivity {
		line := fmt.Sprintf("%s %s", activity.Timestamp, activity.Message)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(1, 2).
		Width(d.width - 4).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, title, box)
}

// colorizeMode colorizes the mode string.
func (d *Dashboard) colorizeMode(mode string) string {
	switch mode {
	case "live_trading":
		return lipgloss.NewStyle().Foreground(styles.ColorDanger).Bold(true).Render(mode)
	case "paper_trading":
		return lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render(mode)
	case "scanning":
		return lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(mode)
	default:
		return lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(mode)
	}
}

// colorizeConnectivity colorizes the connectivity string.
func (d *Dashboard) colorizeConnectivity(connectivity string) string {
	switch connectivity {
	case "connected":
		return lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render(connectivity)
	case "reconnecting":
		return lipgloss.NewStyle().Foreground(styles.ColorWarning).Render(connectivity)
	case "network_waiting":
		return lipgloss.NewStyle().Foreground(styles.ColorDanger).Render(connectivity)
	default:
		return lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(connectivity)
	}
}

// colorizeLicense colorizes the license string.
func (d *Dashboard) colorizeLicense(license string) string {
	switch license {
	case "licensed":
		return lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render(license)
	case "demo":
		return lipgloss.NewStyle().Foreground(styles.ColorPrimary).Render(license)
	case "grace":
		return lipgloss.NewStyle().Foreground(styles.ColorWarning).Render(license)
	default:
		return lipgloss.NewStyle().Foreground(styles.ColorDanger).Render(license)
	}
}

// colorizePnL colorizes the PnL string.
func (d *Dashboard) colorizePnL(pnl string) string {
	if strings.HasPrefix(pnl, "+") {
		return lipgloss.NewStyle().Foreground(styles.ColorSuccess).Bold(true).Render(pnl)
	} else if strings.HasPrefix(pnl, "-") {
		return lipgloss.NewStyle().Foreground(styles.ColorDanger).Bold(true).Render(pnl)
	}
	return pnl
}
