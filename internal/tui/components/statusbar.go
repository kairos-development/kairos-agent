package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/tui/styles"
	"github.com/shopspring/decimal"
)

// StatusBar renders the top status bar with system information.
type StatusBar struct {
	mode     entity.RunMode
	ping     time.Duration
	ntpDrift time.Duration
	pnl      decimal.Decimal
	uptime   time.Duration
	width    int
}

// NewStatusBar creates a new status bar.
func NewStatusBar() *StatusBar {
	return &StatusBar{}
}

// Update updates the status bar data.
func (s *StatusBar) Update(mode entity.RunMode, ping, ntpDrift time.Duration, pnl decimal.Decimal, uptime time.Duration) {
	s.mode = mode
	s.ping = ping
	s.ntpDrift = ntpDrift
	s.pnl = pnl
	s.uptime = uptime
}

// SetWidth sets the width of the status bar.
func (s *StatusBar) SetWidth(width int) {
	s.width = width
}

// Render renders the status bar.
func (s *StatusBar) Render() string {
	var statusStyle lipgloss.Style
	switch s.mode {
	case entity.RunModeIdle:
		statusStyle = styles.StatusIdleStyle
	case entity.RunModeLiveTrading:
		statusStyle = styles.StatusActiveStyle
	case entity.RunModePaperTrading, entity.RunModeBacktesting, entity.RunModeOptimizing, entity.RunModeScanning:
		statusStyle = styles.StatusWarningStyle
	default:
		statusStyle = styles.StatusDangerStyle
	}

	status := statusStyle.Render(fmt.Sprintf("● %s", s.mode.Display()))

	// Ping indicator
	pingColor := styles.ColorSuccess
	if s.ping > 100*time.Millisecond {
		pingColor = styles.ColorWarning
	}
	if s.ping > 500*time.Millisecond {
		pingColor = styles.ColorDanger
	}
	ping := lipgloss.NewStyle().Foreground(pingColor).Render(fmt.Sprintf("Ping: %dms", s.ping.Milliseconds()))

	// NTP drift indicator
	driftColor := styles.ColorSuccess
	if s.ntpDrift > 100*time.Millisecond {
		driftColor = styles.ColorWarning
	}
	if s.ntpDrift > 500*time.Millisecond {
		driftColor = styles.ColorDanger
	}
	drift := lipgloss.NewStyle().Foreground(driftColor).Render(fmt.Sprintf("Drift: %dms", s.ntpDrift.Milliseconds()))

	// PnL indicator
	pnlColor := styles.ColorSuccess
	pnlSign := "+"
	if s.pnl.IsNegative() {
		pnlColor = styles.ColorDanger
		pnlSign = ""
	}
	pnl := lipgloss.NewStyle().Foreground(pnlColor).Bold(true).Render(fmt.Sprintf("PnL: %s$%s", pnlSign, s.pnl.StringFixed(2)))

	// Uptime
	uptime := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(fmt.Sprintf("Uptime: %s", formatDuration(s.uptime)))

	// Combine all elements
	left := lipgloss.JoinHorizontal(lipgloss.Left, status, "  ", ping, "  ", drift)
	right := lipgloss.JoinHorizontal(lipgloss.Left, pnl, "  ", uptime)

	// Calculate spacing
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	spacing := s.width - leftWidth - rightWidth - 4 // 4 for padding

	if spacing < 0 {
		spacing = 0
	}

	bar := lipgloss.JoinHorizontal(lipgloss.Left, left, strings.Repeat(" ", spacing), right)

	return styles.TopBarStyle.Width(s.width).Render(bar)
}

// formatDuration formats a duration in human-readable format.
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
