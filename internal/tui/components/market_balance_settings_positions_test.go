package components

import (
	"strings"
	"testing"

	"github.com/kairos-development/kairos-agent/internal/tui/viewmodel"
	"github.com/stretchr/testify/assert"
)

func TestMarketViewRenderStates(t *testing.T) {
	view := NewMarketView()
	view.SetSize(80, 20)
	assert.Contains(t, view.Render(), "Loading market data")
	view.Update(&viewmodel.Market{Symbol: "BTCUSDT", LastPrice: "$50,000.00", BidPrice: "$49,999.00", AskPrice: "$50,001.00", Spread: "$2.00", SpreadPct: "0.004%", Volume24h: "100 BTC"})
	rendered := view.Render()
	assert.Contains(t, rendered, "Market: BTCUSDT")
	assert.Contains(t, rendered, "$50,000.00")
}

func TestBalanceTableRenderStates(t *testing.T) {
	table := NewBalanceTable()
	table.SetSize(90, 20)
	assert.Contains(t, table.Render(), "Loading balances")
	table.Update(&viewmodel.Balance{})
	assert.Contains(t, table.Render(), "No balance data")
	table.Update(&viewmodel.Balance{TotalEquity: "$10,000.00", Balances: []viewmodel.BalanceItem{{Asset: "USDT", Total: "10000", Available: "9000", Locked: "1000"}}})
	rendered := table.Render()
	assert.Contains(t, rendered, "Account Balance")
	assert.Contains(t, rendered, "USDT")
	assert.Contains(t, rendered, "$10,000.00")
}

func TestSettingsViewRenderStates(t *testing.T) {
	view := NewSettingsView()
	view.SetSize(90, 24)
	assert.Contains(t, view.Render(), "Loading settings")
	view.Update(&viewmodel.Settings{Risk: []viewmodel.ConfigItem{{Key: "max_position", Value: "1000"}}, System: []viewmodel.ConfigItem{{Key: "export_timezone", Value: "UTC"}}})
	rendered := view.Render()
	assert.Contains(t, rendered, "Configuration")
	assert.Contains(t, rendered, "Risk Management")
	assert.Contains(t, rendered, "export_timezone")
	assert.Contains(t, rendered, "/set")
}

func TestPositionsTableRenderAndScroll(t *testing.T) {
	table := NewPositionsTable()
	table.SetSize(100, 8)
	assert.Contains(t, table.Render(), "Loading positions")
	table.Update(&viewmodel.Positions{})
	assert.Contains(t, table.Render(), "No open positions")
	table.Update(&viewmodel.Positions{TotalPnL: "-$10.00", Positions: []viewmodel.Position{
		{Symbol: "BTCUSDT", Side: "LONG", Qty: "0.1", EntryPrice: "$50000", CurrentPrice: "$50100", PnL: "+$10.00", PnLPercent: "1%", EntryTime: "12:00"},
		{Symbol: "ETHUSDT", Side: "SHORT", Qty: "2", EntryPrice: "$3000", CurrentPrice: "$2990", PnL: "-$5.00", PnLPercent: "-1%", EntryTime: "12:01"},
		{Symbol: "SOLUSDT", Side: "LONG", Qty: "5", EntryPrice: "$100", CurrentPrice: "$101", PnL: "+$5.00", PnLPercent: "1%", EntryTime: "12:02"},
	}})
	rendered := table.Render()
	assert.Contains(t, rendered, "Open Positions")
	assert.Contains(t, rendered, "BTCUSDT")
	assert.Contains(t, rendered, "Total PnL")
	table.ScrollDown()
	assert.GreaterOrEqual(t, table.offset, 0)
	table.ScrollUp()
	assert.Equal(t, 0, table.offset)
	assert.True(t, strings.HasSuffix(padRight("x", 3), "  "))
}
