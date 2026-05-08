package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/kairos-development/kairos-agent/internal/tui/styles"
	"github.com/kairos-development/kairos-agent/internal/tui/viewmodel"
)

// MarketView displays market ticker data.
type MarketView struct {
	vm     *viewmodel.Market
	width  int
	height int
}

// NewMarketView creates a new market view.
func NewMarketView() *MarketView {
	return &MarketView{}
}

// Update updates the market view with new view model data.
func (mv *MarketView) Update(vm *viewmodel.Market) {
	mv.vm = vm
}

// SetSize sets the dimensions of the view.
func (mv *MarketView) SetSize(width, height int) {
	mv.width = width
	mv.height = height
}

// Render renders the market view.
func (mv *MarketView) Render() string {
	if mv.vm == nil {
		return lipgloss.NewStyle().
			Width(mv.width).
			Height(mv.height).
			Render("Loading market data...")
	}

	// Title
	title := styles.TitleStyle.Render(fmt.Sprintf("Market: %s", mv.vm.Symbol))

	// Content
	content := mv.renderContent()

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorMuted).
		Padding(1, 2).
		Width(mv.width - 4).
		Height(mv.height - 4).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, title, box)
}

// renderContent renders the market data content.
func (mv *MarketView) renderContent() string {
	labelStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)
	valueStyle := lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true)
	priceStyle := lipgloss.NewStyle().Foreground(styles.ColorSuccess).Bold(true)

	lines := []string{
		"",
		fmt.Sprintf("%s  %s", labelStyle.Render("Last Price:"), priceStyle.Render(mv.vm.LastPrice)),
		fmt.Sprintf("%s       %s", labelStyle.Render("Bid:"), valueStyle.Render(mv.vm.BidPrice)),
		fmt.Sprintf("%s       %s", labelStyle.Render("Ask:"), valueStyle.Render(mv.vm.AskPrice)),
		fmt.Sprintf("%s    %s (%s)", labelStyle.Render("Spread:"), valueStyle.Render(mv.vm.Spread), mv.vm.SpreadPct),
		fmt.Sprintf("%s %s", labelStyle.Render("Volume 24h:"), valueStyle.Render(mv.vm.Volume24h)),
		"",
		"",
		labelStyle.Render("[←] Previous Symbol  [→] Next Symbol"),
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
