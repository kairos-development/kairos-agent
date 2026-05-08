package components

import (
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestNewStatusBar(t *testing.T) {
	sb := NewStatusBar()
	assert.NotNil(t, sb)
	assert.Equal(t, 0, sb.width)
}

func TestStatusBar_Update(t *testing.T) {
	sb := NewStatusBar()

	mode := entity.RunModeLiveTrading
	ping := 50 * time.Millisecond
	ntpDrift := 10 * time.Millisecond
	pnl := decimal.NewFromFloat(123.45)
	uptime := 2 * time.Hour

	sb.Update(mode, ping, ntpDrift, pnl, uptime)

	assert.Equal(t, mode, sb.mode)
	assert.Equal(t, ping, sb.ping)
	assert.Equal(t, ntpDrift, sb.ntpDrift)
	assert.True(t, pnl.Equal(sb.pnl))
	assert.Equal(t, uptime, sb.uptime)
}

func TestStatusBar_SetWidth(t *testing.T) {
	sb := NewStatusBar()
	sb.SetWidth(100)
	assert.Equal(t, 100, sb.width)
}

func TestStatusBar_Render_Idle(t *testing.T) {
	sb := NewStatusBar()
	sb.SetWidth(80)
	sb.Update(entity.RunModeIdle, 50*time.Millisecond, 10*time.Millisecond, decimal.Zero, time.Hour)

	result := sb.Render()
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Idle")
}

func TestStatusBar_Render_LiveTrading(t *testing.T) {
	sb := NewStatusBar()
	sb.SetWidth(80)
	sb.Update(entity.RunModeLiveTrading, 50*time.Millisecond, 10*time.Millisecond, decimal.NewFromFloat(100.50), time.Hour)

	result := sb.Render()
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Live")
}

func TestStatusBar_Render_Halted(t *testing.T) {
	sb := NewStatusBar()
	sb.SetWidth(80)
	sb.Update(entity.RunModeIdle, 50*time.Millisecond, 10*time.Millisecond, decimal.Zero, time.Hour)

	result := sb.Render()
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Idle")
}

func TestStatusBar_Render_PingColors(t *testing.T) {
	sb := NewStatusBar()
	sb.SetWidth(80)

	sb.Update(entity.RunModeIdle, 50*time.Millisecond, 10*time.Millisecond, decimal.Zero, time.Hour)
	result := sb.Render()
	assert.NotEmpty(t, result)

	sb.Update(entity.RunModeIdle, 200*time.Millisecond, 10*time.Millisecond, decimal.Zero, time.Hour)
	result = sb.Render()
	assert.NotEmpty(t, result)

	sb.Update(entity.RunModeIdle, 600*time.Millisecond, 10*time.Millisecond, decimal.Zero, time.Hour)
	result = sb.Render()
	assert.NotEmpty(t, result)
}

func TestStatusBar_Render_DriftColors(t *testing.T) {
	sb := NewStatusBar()
	sb.SetWidth(80)

	sb.Update(entity.RunModeIdle, 50*time.Millisecond, 50*time.Millisecond, decimal.Zero, time.Hour)
	result := sb.Render()
	assert.NotEmpty(t, result)

	// Medium drift (yellow)
	sb.Update(entity.RunModeIdle, 50*time.Millisecond, 200*time.Millisecond, decimal.Zero, time.Hour)
	result = sb.Render()
	assert.NotEmpty(t, result)

	// High drift (red)
	sb.Update(entity.RunModeIdle, 50*time.Millisecond, 600*time.Millisecond, decimal.Zero, time.Hour)
	result = sb.Render()
	assert.NotEmpty(t, result)
}

func TestStatusBar_Render_PositivePnL(t *testing.T) {
	sb := NewStatusBar()
	sb.SetWidth(80)
	sb.Update(entity.RunModeIdle, 50*time.Millisecond, 10*time.Millisecond, decimal.NewFromFloat(123.45), time.Hour)

	result := sb.Render()
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "+")
}

func TestStatusBar_Render_NegativePnL(t *testing.T) {
	sb := NewStatusBar()
	sb.SetWidth(80)
	sb.Update(entity.RunModeIdle, 50*time.Millisecond, 10*time.Millisecond, decimal.NewFromFloat(-123.45), time.Hour)

	result := sb.Render()
	assert.NotEmpty(t, result)
}

func TestStatusBar_Render_NarrowWidth(t *testing.T) {
	sb := NewStatusBar()
	sb.SetWidth(20)
	sb.Update(entity.RunModeIdle, 50*time.Millisecond, 10*time.Millisecond, decimal.Zero, time.Hour)

	result := sb.Render()
	assert.NotEmpty(t, result)
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"seconds only", 45 * time.Second, "45s"},
		{"minutes and seconds", 2*time.Minute + 30*time.Second, "2m 30s"},
		{"hours and minutes", 2*time.Hour + 15*time.Minute, "2h 15m"},
		{"hours only", 3 * time.Hour, "3h 0m"},
		{"zero", 0, "0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.want, result)
		})
	}
}
