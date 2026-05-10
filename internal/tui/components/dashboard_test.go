package components

import (
	"strings"
	"testing"

	"github.com/kairos-development/kairos-agent/internal/tui/viewmodel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDashboard(t *testing.T) {
	d := NewDashboard()
	require.NotNil(t, d)
	assert.Nil(t, d.vm)
	assert.Equal(t, 0, d.width)
	assert.Equal(t, 0, d.height)
}

func TestDashboard_Update(t *testing.T) {
	d := NewDashboard()
	vm := &viewmodel.Dashboard{
		SystemStatus: viewmodel.SystemStatus{
			Mode:         "paper_trading",
			Connectivity: "connected",
			License:      "demo",
			Uptime:       "1h 30m",
			NTPDrift:     "50ms",
			HaltReason:   "exchange stream gap",
			HaltedAt:     "14:32:15",
		},
	}

	d.Update(vm)
	assert.Equal(t, vm, d.vm)
}

func TestDashboard_SetSize(t *testing.T) {
	d := NewDashboard()
	d.SetSize(120, 40)
	assert.Equal(t, 120, d.width)
	assert.Equal(t, 40, d.height)
}

func TestDashboard_Render_NoData(t *testing.T) {
	d := NewDashboard()
	d.SetSize(80, 24)

	rendered := d.Render()
	assert.Contains(t, rendered, "Loading dashboard...")
}

func TestDashboard_Render_WithData(t *testing.T) {
	d := NewDashboard()
	d.SetSize(80, 24)

	vm := &viewmodel.Dashboard{
		SystemStatus: viewmodel.SystemStatus{
			Mode:         "paper_trading",
			Connectivity: "connected",
			License:      "demo",
			Uptime:       "1h 30m",
			NTPDrift:     "50ms",
			HaltReason:   "exchange stream gap",
			HaltedAt:     "14:32:15",
		},
		Performance: viewmodel.Performance{
			TotalPnL:      "+$1234.56",
			DailyPnL:      "+$234.56",
			OpenPositions: 3,
			ActiveOrders:  5,
			WinRate:       "65.2%",
			TotalTrades:   42,
		},
		ActiveStrategies: []viewmodel.Strategy{
			{
				Name:   "SMA-Cross-BTC",
				Status: "active",
				PnL:    "+$456.78",
			},
		},
		ActiveOrders: []viewmodel.Order{
			{
				Symbol: "BTCUSDT",
				Side:   "BUY",
				Type:   "LIMIT",
				Qty:    "0.1000",
				Price:  "50000",
				Status: "PENDING",
			},
		},
		RecentActivity: []viewmodel.Activity{
			{
				Timestamp: "14:32:15",
				Message:   "Order filled",
			},
		},
	}

	d.Update(vm)
	rendered := d.Render()

	require.NotEmpty(t, rendered)

	// Verify system status section
	assert.Contains(t, rendered, "System Status")
	assert.Contains(t, rendered, "paper_trading")
	assert.Contains(t, rendered, "connected")
	assert.Contains(t, rendered, "demo")
	assert.Contains(t, rendered, "1h 30m")
	assert.Contains(t, rendered, "exchange stream gap")
	assert.Contains(t, rendered, "14:32:15")

	// Verify performance section
	assert.Contains(t, rendered, "Performance")
	assert.Contains(t, rendered, "+$1234.56")
	assert.Contains(t, rendered, "+$234.56")
	assert.Contains(t, rendered, "65.2%")

	// Verify strategies section
	assert.Contains(t, rendered, "Active Strategies")
	assert.Contains(t, rendered, "SMA-Cross-BTC")
	assert.Contains(t, rendered, "+$456.78")

	// Verify orders section
	assert.Contains(t, rendered, "Active Orders")
	assert.Contains(t, rendered, "BTCUSDT")
	assert.Contains(t, rendered, "BUY")
	assert.Contains(t, rendered, "LIMIT")

	// Verify activity section
	assert.Contains(t, rendered, "Recent Activity")
	assert.Contains(t, rendered, "14:32:15")
	assert.Contains(t, rendered, "Order filled")
}

func TestDashboard_Render_MinimalData(t *testing.T) {
	d := NewDashboard()
	d.SetSize(80, 24)

	vm := &viewmodel.Dashboard{
		SystemStatus: viewmodel.SystemStatus{
			Mode:         "idle",
			Connectivity: "disconnected",
			License:      "unlicensed",
			Uptime:       "0s",
			NTPDrift:     "0ms",
		},
		Performance: viewmodel.Performance{
			TotalPnL:      "+$0.00",
			DailyPnL:      "+$0.00",
			OpenPositions: 0,
			ActiveOrders:  0,
			WinRate:       "0.0%",
			TotalTrades:   0,
		},
		ActiveStrategies: []viewmodel.Strategy{},
		ActiveOrders:     []viewmodel.Order{},
		RecentActivity:   []viewmodel.Activity{},
	}

	d.Update(vm)
	rendered := d.Render()

	require.NotEmpty(t, rendered)

	// Should have system status and performance
	assert.Contains(t, rendered, "System Status")
	assert.Contains(t, rendered, "Performance")

	// Should NOT have sections for empty data
	assert.NotContains(t, rendered, "Active Strategies")
	assert.NotContains(t, rendered, "Active Orders (Recent 5)")
	assert.NotContains(t, rendered, "Recent Activity")
}

func TestDashboard_ColorizeMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{"live trading", "live_trading"},
		{"paper trading", "paper_trading"},
		{"scanning", "scanning"},
		{"idle", "idle"},
		{"unknown", "unknown_mode"},
	}

	d := NewDashboard()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.colorizeMode(tt.mode)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, tt.mode)
		})
	}
}

func TestDashboard_ColorizeConnectivity(t *testing.T) {
	tests := []struct {
		name         string
		connectivity string
	}{
		{"connected", "connected"},
		{"reconnecting", "reconnecting"},
		{"network waiting", "network_waiting"},
		{"disconnected", "disconnected"},
	}

	d := NewDashboard()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.colorizeConnectivity(tt.connectivity)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, tt.connectivity)
		})
	}
}

func TestDashboard_ColorizeLicense(t *testing.T) {
	tests := []struct {
		name    string
		license string
	}{
		{"licensed", "licensed"},
		{"demo", "demo"},
		{"grace", "grace"},
		{"unlicensed", "unlicensed"},
	}

	d := NewDashboard()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.colorizeLicense(tt.license)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, tt.license)
		})
	}
}

func TestDashboard_ColorizePnL(t *testing.T) {
	tests := []struct {
		name string
		pnl  string
	}{
		{"positive", "+$1234.56"},
		{"negative", "-$1234.56"},
		{"zero", "+$0.00"},
		{"no sign", "$1234.56"},
	}

	d := NewDashboard()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.colorizePnL(tt.pnl)
			assert.NotEmpty(t, result)
			// Should contain the PnL value (may be wrapped in ANSI codes)
			assert.True(t, strings.Contains(result, tt.pnl) || strings.Contains(result, strings.TrimPrefix(tt.pnl, "+")))
		})
	}
}

func TestDashboard_Render_MultipleStrategies(t *testing.T) {
	d := NewDashboard()
	d.SetSize(80, 24)

	vm := &viewmodel.Dashboard{
		SystemStatus: viewmodel.SystemStatus{
			Mode:         "paper_trading",
			Connectivity: "connected",
			License:      "demo",
		},
		Performance: viewmodel.Performance{
			TotalPnL: "+$1000.00",
			DailyPnL: "+$500.00",
		},
		ActiveStrategies: []viewmodel.Strategy{
			{Name: "Strategy1", Status: "active", PnL: "+$100.00"},
			{Name: "Strategy2", Status: "active", PnL: "+$200.00"},
			{Name: "Strategy3", Status: "paused", PnL: "-$50.00"},
		},
	}

	d.Update(vm)
	rendered := d.Render()

	assert.Contains(t, rendered, "Strategy1")
	assert.Contains(t, rendered, "Strategy2")
	assert.Contains(t, rendered, "Strategy3")
	assert.Contains(t, rendered, "+$100.00")
	assert.Contains(t, rendered, "+$200.00")
	assert.Contains(t, rendered, "-$50.00")
}

func TestDashboard_Render_MultipleOrders(t *testing.T) {
	d := NewDashboard()
	d.SetSize(80, 24)

	vm := &viewmodel.Dashboard{
		SystemStatus: viewmodel.SystemStatus{
			Mode:         "paper_trading",
			Connectivity: "connected",
			License:      "demo",
		},
		Performance: viewmodel.Performance{
			TotalPnL: "+$1000.00",
		},
		ActiveOrders: []viewmodel.Order{
			{Symbol: "BTCUSDT", Side: "BUY", Type: "LIMIT", Qty: "0.1", Price: "50000", Status: "PENDING"},
			{Symbol: "ETHUSDT", Side: "SELL", Type: "MARKET", Qty: "2.5", Price: "0", Status: "SUBMITTED"},
		},
	}

	d.Update(vm)
	rendered := d.Render()

	assert.Contains(t, rendered, "BTCUSDT")
	assert.Contains(t, rendered, "ETHUSDT")
	assert.Contains(t, rendered, "BUY")
	assert.Contains(t, rendered, "SELL")
	assert.Contains(t, rendered, "LIMIT")
	assert.Contains(t, rendered, "MARKET")
}

func TestDashboard_Render_NegativePnL(t *testing.T) {
	d := NewDashboard()
	d.SetSize(80, 24)

	vm := &viewmodel.Dashboard{
		SystemStatus: viewmodel.SystemStatus{
			Mode:         "paper_trading",
			Connectivity: "connected",
			License:      "demo",
		},
		Performance: viewmodel.Performance{
			TotalPnL: "-$500.00",
			DailyPnL: "-$100.00",
		},
	}

	d.Update(vm)
	rendered := d.Render()

	assert.Contains(t, rendered, "-$500.00")
	assert.Contains(t, rendered, "-$100.00")
}
