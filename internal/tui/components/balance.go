package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/kairos-development/kairos-agent/internal/tui/styles"
	"github.com/kairos-development/kairos-agent/internal/tui/viewmodel"
)

// BalanceTable displays account balances in a table format.
type BalanceTable struct {
	vm     *viewmodel.Balance
	width  int
	height int
}

// NewBalanceTable creates a new balance table.
func NewBalanceTable() *BalanceTable {
	return &BalanceTable{}
}

// Update updates the balance table with new view model data.
func (bt *BalanceTable) Update(vm *viewmodel.Balance) {
	bt.vm = vm
}

// SetSize sets the dimensions of the table.
func (bt *BalanceTable) SetSize(width, height int) {
	bt.width = width
	bt.height = height
}

// Render renders the balance table.
func (bt *BalanceTable) Render() string {
	if bt.vm == nil {
		return lipgloss.NewStyle().
			Width(bt.width).
			Height(bt.height).
			Render("Loading balances...")
	}

	if len(bt.vm.Balances) == 0 {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.ColorMuted).
			Padding(1, 2).
			Width(bt.width - 4).
			Height(bt.height - 2).
			Render(styles.HelpStyle.Render("No balance data available"))
	}

	// Title
	title := styles.TitleStyle.Render("Account Balance")

	// Header
	header := bt.renderHeader()

	// Separator
	separator := strings.Repeat("─", bt.width-8)

	// Rows
	var rows []string
	for _, balance := range bt.vm.Balances {
		rows = append(rows, bt.renderRow(balance))
	}

	// Footer with total equity
	footer := bt.renderFooter()

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
		Width(bt.width - 4).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, title, box)
}

// renderHeader renders the table header.
func (bt *BalanceTable) renderHeader() string {
	cols := []string{
		padRight("Asset", 10),
		padRight("Total", 16),
		padRight("Available", 16),
		padRight("Locked", 16),
	}

	header := strings.Join(cols, " ")
	return styles.TableHeaderStyle.Render(header)
}

// renderRow renders a single balance row.
func (bt *BalanceTable) renderRow(balance viewmodel.BalanceItem) string {
	cols := []string{
		padRight(balance.Asset, 10),
		padRight(balance.Total, 16),
		padRight(balance.Available, 16),
		padRight(balance.Locked, 16),
	}

	return strings.Join(cols, " ")
}

// renderFooter renders the footer with total equity.
func (bt *BalanceTable) renderFooter() string {
	totalEquity := bt.vm.TotalEquity

	equityStyle := lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true)

	return fmt.Sprintf("Total Equity (USD): %s", equityStyle.Render(totalEquity))
}
