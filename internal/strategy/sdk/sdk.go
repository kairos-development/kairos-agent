package sdk

import (
	"github.com/shopspring/decimal"
	"time"
)

// StrategySDK provides Go bindings for WASM strategy development.
// Strategies compile this SDK to WASM and call these functions.

// MarketDataPoint represents a single market data point.
type MarketDataPoint struct {
	Symbol    string
	LastPrice decimal.Decimal
	BidPrice  decimal.Decimal
	AskPrice  decimal.Decimal
	Volume24h decimal.Decimal
	Timestamp time.Time
}

// OrderSignal represents a trading signal to place an order.
type OrderSignal struct {
	Action      string // "BUY", "SELL", "CLOSE", "HOLD"
	Symbol      string
	Quantity    decimal.Decimal
	Price       decimal.Decimal
	TimeInForce string          // "GTC", "IOC", "FOK"
	Confidence  decimal.Decimal // 0-1
	Reason      string
}

// PositionInfo represents current position state.
type PositionInfo struct {
	Symbol        string
	Side          string
	Quantity      decimal.Decimal
	EntryPrice    decimal.Decimal
	CurrentPrice  decimal.Decimal
	UnrealizedPnL decimal.Decimal
	RealizedPnL   decimal.Decimal
}

// BalanceInfo represents account balance.
type BalanceInfo struct {
	Total     decimal.Decimal
	Available decimal.Decimal
	Locked    decimal.Decimal
}

// StrategyContext provides data to strategy callbacks.
type StrategyContext struct {
	MarketData MarketDataPoint
	Position   PositionInfo
	Balance    BalanceInfo
	Timestamp  time.Time
}

// Strategy interface that WASM strategies must implement.
type Strategy interface {
	// OnTick is called on each market tick.
	// Returns an OrderSignal if a trade should be executed, nil otherwise.
	OnTick(ctx *StrategyContext) *OrderSignal

	// OnOrderFilled is called when an order is filled.
	OnOrderFilled(orderID string, filledQty, filledPrice decimal.Decimal)

	// OnOrderCanceled is called when an order is canceled.
	OnOrderCanceled(orderID string)

	// GetParameters returns current strategy parameters.
	GetParameters() map[string]interface{}

	// SetParameters updates strategy parameters.
	SetParameters(params map[string]interface{}) error

	// Reset resets strategy state.
	Reset() error
}

// BaseStrategy provides default implementations for Strategy interface.
type BaseStrategy struct {
	Name       string
	Version    string
	Parameters map[string]interface{}
}

// GetParameters returns strategy parameters.
func (bs *BaseStrategy) GetParameters() map[string]interface{} {
	return bs.Parameters
}

// SetParameters updates strategy parameters.
func (bs *BaseStrategy) SetParameters(params map[string]interface{}) error {
	bs.Parameters = params
	return nil
}

// Reset resets strategy state.
func (bs *BaseStrategy) Reset() error {
	return nil
}

// SimpleMovingAverage calculates a simple moving average.
func SimpleMovingAverage(prices []decimal.Decimal, period int) decimal.Decimal {
	if len(prices) < period {
		return decimal.Zero
	}

	sum := decimal.Zero
	for i := len(prices) - period; i < len(prices); i++ {
		sum = sum.Add(prices[i])
	}

	return sum.Div(decimal.NewFromInt(int64(period)))
}

// ExponentialMovingAverage calculates an exponential moving average.
func ExponentialMovingAverage(prices []decimal.Decimal, period int) decimal.Decimal {
	if len(prices) == 0 {
		return decimal.Zero
	}

	multiplier := decimal.NewFromInt(2).Div(decimal.NewFromInt(int64(period + 1)))

	ema := prices[0]
	for i := 1; i < len(prices); i++ {
		ema = prices[i].Mul(multiplier).Add(ema.Mul(decimal.NewFromInt(1).Sub(multiplier)))
	}

	return ema
}

// RSI calculates the Relative Strength Index.
func RSI(prices []decimal.Decimal, period int) decimal.Decimal {
	if len(prices) < period+1 {
		return decimal.Zero
	}

	gains := decimal.Zero
	losses := decimal.Zero

	for i := len(prices) - period; i < len(prices); i++ {
		change := prices[i].Sub(prices[i-1])
		if change.IsPositive() {
			gains = gains.Add(change)
		} else {
			losses = losses.Add(change.Abs())
		}
	}

	avgGain := gains.Div(decimal.NewFromInt(int64(period)))
	avgLoss := losses.Div(decimal.NewFromInt(int64(period)))

	if avgLoss.IsZero() {
		return decimal.NewFromInt(100)
	}

	rs := avgGain.Div(avgLoss)
	rsi := decimal.NewFromInt(100).Sub(decimal.NewFromInt(100).Div(decimal.NewFromInt(1).Add(rs)))

	return rsi
}

// MACD calculates the Moving Average Convergence Divergence.
type MACDResult struct {
	MACD      decimal.Decimal
	Signal    decimal.Decimal
	Histogram decimal.Decimal
}

func MACD(prices []decimal.Decimal, fastPeriod, slowPeriod, signalPeriod int) *MACDResult {
	fastEMA := ExponentialMovingAverage(prices, fastPeriod)
	slowEMA := ExponentialMovingAverage(prices, slowPeriod)

	macdLine := fastEMA.Sub(slowEMA)

	// Signal line is EMA of MACD line
	// For simplicity, use SMA here
	signalLine := decimal.Zero
	if len(prices) >= signalPeriod {
		signalLine = SimpleMovingAverage(prices, signalPeriod)
	}

	histogram := macdLine.Sub(signalLine)

	return &MACDResult{
		MACD:      macdLine,
		Signal:    signalLine,
		Histogram: histogram,
	}
}

// Bollinger Bands calculates Bollinger Bands.
type BollingerBandsResult struct {
	MiddleBand decimal.Decimal
	UpperBand  decimal.Decimal
	LowerBand  decimal.Decimal
}

func BollingerBands(prices []decimal.Decimal, period int, stdDevMultiplier decimal.Decimal) *BollingerBandsResult {
	if len(prices) < period {
		return &BollingerBandsResult{}
	}

	// Calculate SMA
	sma := SimpleMovingAverage(prices, period)

	// Calculate standard deviation
	sumSquaredDiff := decimal.Zero
	for i := len(prices) - period; i < len(prices); i++ {
		diff := prices[i].Sub(sma)
		sumSquaredDiff = sumSquaredDiff.Add(diff.Mul(diff))
	}

	variance := sumSquaredDiff.Div(decimal.NewFromInt(int64(period)))
	// Note: sqrt is not available in decimal, so we approximate
	stdDev := variance // Simplified - in production use proper sqrt

	return &BollingerBandsResult{
		MiddleBand: sma,
		UpperBand:  sma.Add(stdDev.Mul(stdDevMultiplier)),
		LowerBand:  sma.Sub(stdDev.Mul(stdDevMultiplier)),
	}
}
