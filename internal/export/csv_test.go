package export

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteCSV_Success(t *testing.T) {
	var buf bytes.Buffer
	rows := []TradeRecord{
		{
			Date:        time.Date(2024, 10, 25, 11, 30, 0, 0, time.UTC),
			Pair:        "BTCUSDT",
			Side:        "buy",
			Size:        decimal.RequireFromString("1"),
			EntryPrice:  decimal.RequireFromString("10"),
			ExitPrice:   decimal.RequireFromString("11"),
			Fee:         decimal.RequireFromString("0.1"),
			RealizedPNL: decimal.RequireFromString("0.9"),
		},
	}

	err := WriteCSV(&buf, "UTC", rows)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "date,pair,side,size,entry_price,exit_price,fee,realized_pnl")
	assert.Contains(t, output, "BTCUSDT")
	assert.Contains(t, output, "buy")
}

func TestWriteCSV_UsesConfiguredTimezone(t *testing.T) {
	var buf bytes.Buffer
	rows := []TradeRecord{{
		Date:        time.Date(2024, 10, 25, 11, 30, 0, 0, time.UTC),
		Pair:        "BTCUSDT",
		Side:        "buy",
		Size:        decimal.RequireFromString("1"),
		EntryPrice:  decimal.RequireFromString("10"),
		ExitPrice:   decimal.RequireFromString("11"),
		Fee:         decimal.RequireFromString("0.1"),
		RealizedPNL: decimal.RequireFromString("0.9"),
	}}

	err := WriteCSV(&buf, "Europe/Moscow", rows)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "2024-10-25T14:30:00+03:00")
}

func TestWriteCSV_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	rows := []TradeRecord{}

	err := WriteCSV(&buf, "UTC", rows)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "date,pair,side,size,entry_price,exit_price,fee,realized_pnl")

	// Only header line
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 1)
}

func TestWriteCSV_MultipleRows(t *testing.T) {
	var buf bytes.Buffer
	rows := []TradeRecord{
		{
			Date:        time.Date(2024, 10, 25, 11, 30, 0, 0, time.UTC),
			Pair:        "BTCUSDT",
			Side:        "buy",
			Size:        decimal.RequireFromString("1"),
			EntryPrice:  decimal.RequireFromString("50000"),
			ExitPrice:   decimal.RequireFromString("51000"),
			Fee:         decimal.RequireFromString("10"),
			RealizedPNL: decimal.RequireFromString("990"),
		},
		{
			Date:        time.Date(2024, 10, 26, 14, 45, 0, 0, time.UTC),
			Pair:        "ETHUSDT",
			Side:        "sell",
			Size:        decimal.RequireFromString("10"),
			EntryPrice:  decimal.RequireFromString("3000"),
			ExitPrice:   decimal.RequireFromString("2900"),
			Fee:         decimal.RequireFromString("5"),
			RealizedPNL: decimal.RequireFromString("995"),
		},
	}

	err := WriteCSV(&buf, "UTC", rows)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "BTCUSDT")
	assert.Contains(t, output, "ETHUSDT")
	assert.Contains(t, output, "buy")
	assert.Contains(t, output, "sell")

	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 3) // Header + 2 data rows
}

func TestWriteCSV_InvalidTimezone(t *testing.T) {
	var buf bytes.Buffer
	rows := []TradeRecord{
		{
			Date:        time.Date(2024, 10, 25, 11, 30, 0, 0, time.UTC),
			Pair:        "BTCUSDT",
			Side:        "buy",
			Size:        decimal.RequireFromString("1"),
			EntryPrice:  decimal.RequireFromString("10"),
			ExitPrice:   decimal.RequireFromString("11"),
			Fee:         decimal.RequireFromString("0.1"),
			RealizedPNL: decimal.RequireFromString("0.9"),
		},
	}

	err := WriteCSV(&buf, "Invalid/Timezone", rows)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load export timezone")
}

func TestWriteCSV_DifferentTimezones(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		expected string
	}{
		{"UTC", "UTC", "2024-10-25T11:30:00Z"},
		{"America/New_York", "America/New_York", "2024-10-25T07:30:00-04:00"},
		{"Asia/Tokyo", "Asia/Tokyo", "2024-10-25T20:30:00+09:00"},
		{"Europe/London", "Europe/London", "2024-10-25T12:30:00+01:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			rows := []TradeRecord{{
				Date:        time.Date(2024, 10, 25, 11, 30, 0, 0, time.UTC),
				Pair:        "BTCUSDT",
				Side:        "buy",
				Size:        decimal.RequireFromString("1"),
				EntryPrice:  decimal.RequireFromString("10"),
				ExitPrice:   decimal.RequireFromString("11"),
				Fee:         decimal.RequireFromString("0.1"),
				RealizedPNL: decimal.RequireFromString("0.9"),
			}}

			err := WriteCSV(&buf, tt.timezone, rows)
			require.NoError(t, err)
			assert.Contains(t, buf.String(), tt.expected)
		})
	}
}

func TestWriteCSV_DecimalPrecision(t *testing.T) {
	var buf bytes.Buffer
	rows := []TradeRecord{
		{
			Date:        time.Date(2024, 10, 25, 11, 30, 0, 0, time.UTC),
			Pair:        "BTCUSDT",
			Side:        "buy",
			Size:        decimal.RequireFromString("0.12345678"),
			EntryPrice:  decimal.RequireFromString("50000.12345678"),
			ExitPrice:   decimal.RequireFromString("51000.87654321"),
			Fee:         decimal.RequireFromString("10.5"),
			RealizedPNL: decimal.RequireFromString("990.75321"),
		},
	}

	err := WriteCSV(&buf, "UTC", rows)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "0.12345678")
	assert.Contains(t, output, "50000.12345678")
	assert.Contains(t, output, "51000.87654321")
	assert.Contains(t, output, "10.5")
	assert.Contains(t, output, "990.75321")
}

func TestWriteCSV_NegativePnL(t *testing.T) {
	var buf bytes.Buffer
	rows := []TradeRecord{
		{
			Date:        time.Date(2024, 10, 25, 11, 30, 0, 0, time.UTC),
			Pair:        "BTCUSDT",
			Side:        "buy",
			Size:        decimal.RequireFromString("1"),
			EntryPrice:  decimal.RequireFromString("51000"),
			ExitPrice:   decimal.RequireFromString("50000"),
			Fee:         decimal.RequireFromString("10"),
			RealizedPNL: decimal.RequireFromString("-1010"),
		},
	}

	err := WriteCSV(&buf, "UTC", rows)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "-1010")
}

func TestWriteCSV_ZeroValues(t *testing.T) {
	var buf bytes.Buffer
	rows := []TradeRecord{
		{
			Date:        time.Date(2024, 10, 25, 11, 30, 0, 0, time.UTC),
			Pair:        "BTCUSDT",
			Side:        "buy",
			Size:        decimal.Zero,
			EntryPrice:  decimal.Zero,
			ExitPrice:   decimal.Zero,
			Fee:         decimal.Zero,
			RealizedPNL: decimal.Zero,
		},
	}

	err := WriteCSV(&buf, "UTC", rows)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "0")
}

func TestWriteCSV_SpecialCharactersInPair(t *testing.T) {
	var buf bytes.Buffer
	rows := []TradeRecord{
		{
			Date:        time.Date(2024, 10, 25, 11, 30, 0, 0, time.UTC),
			Pair:        "BTC-USDT",
			Side:        "buy",
			Size:        decimal.RequireFromString("1"),
			EntryPrice:  decimal.RequireFromString("10"),
			ExitPrice:   decimal.RequireFromString("11"),
			Fee:         decimal.RequireFromString("0.1"),
			RealizedPNL: decimal.RequireFromString("0.9"),
		},
	}

	err := WriteCSV(&buf, "UTC", rows)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "BTC-USDT")
}

func TestWriteCSV_HeaderFormat(t *testing.T) {
	var buf bytes.Buffer
	rows := []TradeRecord{}

	err := WriteCSV(&buf, "UTC", rows)
	require.NoError(t, err)

	output := buf.String()
	expectedHeader := "date,pair,side,size,entry_price,exit_price,fee,realized_pnl"
	assert.True(t, strings.HasPrefix(output, expectedHeader))
}

func TestWriteCSV_LargeDataset(t *testing.T) {
	var buf bytes.Buffer
	rows := make([]TradeRecord, 1000)

	for i := 0; i < 1000; i++ {
		rows[i] = TradeRecord{
			Date:        time.Date(2024, 10, 25, 11, 30, 0, 0, time.UTC).Add(time.Duration(i) * time.Hour),
			Pair:        "BTCUSDT",
			Side:        "buy",
			Size:        decimal.RequireFromString("1"),
			EntryPrice:  decimal.RequireFromString("50000"),
			ExitPrice:   decimal.RequireFromString("51000"),
			Fee:         decimal.RequireFromString("10"),
			RealizedPNL: decimal.RequireFromString("990"),
		}
	}

	err := WriteCSV(&buf, "UTC", rows)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 1001) // Header + 1000 data rows
}
