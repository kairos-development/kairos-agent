package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kairos-development/kairos-agent/internal/tui/styles"
	"github.com/kairos-development/kairos-agent/internal/tui/viewmodel"
)

// PositionsTable displays open positions in a table format.
type PositionsTable struct {
	vm     *viewmodel.Positions
	width  int
	height int
	offset int
}

// NewPositionsTable creates a new positions table.
func NewPositionsTable() *PositionsTable {
	return &PositionsTable{
		offset: 0,
	}
}

// Update updates the positions table with new view model data.
func (pt *PositionsTable) Update(vm *viewmodel.Positions) {
	pt.vm = vm
	// Reset offset when data changes
	pt.offset = 0
}

// SetSize sets the dimensions of the table.
func (pt *PositionsTable) SetSize(width, height int) {
	pt.width = width
	pt.height = height
}

// ScrollUp scrolls the table up.
func (pt *PositionsTable) ScrollUp() {
	if pt.offset > 0 {
		pt.offset--
	}
}

// ScrollDown scrolls the table down.
func (pt *PositionsTable) ScrollDown() {
	if pt.vm == nil {
		return
	}
	maxOffset := len(pt.vm.Positions) - (pt.height - 4) // Account for header and footer
	if maxOffset < 0 {
		maxOffset = 0
	}
	if pt.offset < maxOffset {
		pt.offset++
	}
}

// Render renders the positions table.
func (pt *PositionsTable) Render() string {
	if pt.vm == nil {
		return lipgloss.NewStyle().
			Width(pt.width).
			Height(pt.height).
			Render("Loading positions...")
	}

	if len(pt.vm.Positions) == 0 {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.ColorMuted).
			Padding(1, 2).
			Width(pt.width - 4).
			Height(pt.height - 2).
			Render(styles.HelpStyle.Render("No open positions"))
	}

	// Title
	title := styles.TitleStyle.Render("Open Positions")

	// Header
	header := pt.renderHeader()

	// Separator
	separator := strings.Repeat("─", pt.width-8)

	// Rows
	var rows []string
	start := pt.offset
	end := pt.offset + (pt.height - 6) // Account for title, header, separator, footer
	if end > len(pt.vm.Positions) {
		end = len(pt.vm.Positions)
	}

	for i := start; i < end; i++ {
		rows = append(rows, pt.renderRow(pt.vm.Positions[i]))
	}

	// Footer with total PnL
	footer := pt.renderFooter()

	// Combine all parts
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		separator,
		strings.Join(rows, "\n"),
		separator,
		footer,
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(1, 2).
		Width(pt.width - 4).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, title, box)
}

// renderHeader renders the table header.
func (pt *PositionsTable) renderHeader() string {
	cols := []string{
		padRight("Symbol", 10),
		padRight("Side", 6),
		padRight("Qty", 10),
		padRight("Entry", 12),
		padRight("Current", 12),
		padRight("PnL", 12),
		padRight("%", 8),
		padRight("Time", 10),
	}

	header := strings.Join(cols, " ")
	return styles.TableHeaderStyle.Render(header)
}

// renderRow renders a single position row.
func (pt *PositionsTable) renderRow(pos viewmodel.Position) string {
	// Determine color based on PnL
	pnlColor := styles.ColorSuccess
	if strings.HasPrefix(pos.PnL, "-") {
		pnlColor = styles.ColorDanger
	}

	pnlStyle := lipgloss.NewStyle().Foreground(pnlColor).Bold(true)
	percentStyle := lipgloss.NewStyle().Foreground(pnlColor)

	// Color-code side
	sideColor := styles.ColorSuccess
	if pos.Side == "short" || pos.Side == "SHORT" {
		sideColor = styles.ColorDanger
	}
	sideStyle := lipgloss.NewStyle().Foreground(sideColor)

	cols := []string{
		padRight(pos.Symbol, 10),
		padRight(sideStyle.Render(pos.Side), 6),
		padRight(pos.Qty, 10),
		padRight(pos.EntryPrice, 12),
		padRight(pos.CurrentPrice, 12),
		padRight(pnlStyle.Render(pos.PnL), 12),
		padRight(percentStyle.Render(pos.PnLPercent), 8),
		padRight(pos.EntryTime, 10),
	}

	return strings.Join(cols, " ")
}

// renderFooter renders the footer with total PnL.
func (pt *PositionsTable) renderFooter() string {
	totalPnL := pt.vm.TotalPnL

	// Determine color
	pnlColor := styles.ColorSuccess
	if strings.HasPrefix(totalPnL, "-") {
		pnlColor = styles.ColorDanger
	}

	pnlStyle := lipgloss.NewStyle().Foreground(pnlColor).Bold(true)

	return fmt.Sprintf("Total PnL: %s", pnlStyle.Render(totalPnL))
}

// padRight pads a string to the right with spaces.
func padRight(s string, width int) string {
	// Account for ANSI color codes by using lipgloss.Width
	actualWidth := lipgloss.Width(s)
	if actualWidth >= width {
		return s
	}
	return s + strings.Repeat(" ", width-actualWidth)
}
