package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"github.com/kairos-development/kairos-agent/internal/strategy"
)

type Engine struct {
	registry *strategy.StrategyRegistry
	contexts map[string]*RunContext
	mu       sync.RWMutex
}

func NewEngine(registry *strategy.StrategyRegistry) *Engine {
	return &Engine{
		registry: registry,
		contexts: make(map[string]*RunContext),
	}
}

type RunContext struct {
	StrategyID string
	InstanceID string
	Symbol     string
	Position   strategy.PositionInfo
	Balance    strategy.BalanceInfo
	LastSignal *strategy.Signal
	LastTick   time.Time
	Params     map[string]interface{}
}

func (e *Engine) Tick(ctx context.Context, input TickInput) (*strategy.Signal, error) {
	e.mu.Lock()
	rc, ok := e.contexts[input.InstanceID]
	if !ok {
		rc = &RunContext{
			StrategyID: input.StrategyID,
			InstanceID: input.InstanceID,
			Symbol:     input.Symbol,
			Params:     input.Parameters,
		}
		e.contexts[input.InstanceID] = rc
	}
	e.mu.Unlock()

	strat, err := e.registry.Get(input.StrategyID)
	if err != nil {
		return nil, fmt.Errorf("get strategy: %w", err)
	}

	marketData := strategy.MarketData{
		Symbol:    input.Symbol,
		LastPrice: input.LastPrice,
		BidPrice:  input.BidPrice,
		AskPrice:  input.AskPrice,
		Volume24h: input.Volume24h,
		High24h:   input.High24h,
		Low24h:    input.Low24h,
		Timestamp: input.Timestamp,
	}

	ctxData := &strategy.StrategyContext{
		MarketData:   marketData,
		CurrentPrice: input.LastPrice,
		Position:     rc.Position,
		Balance:      rc.Balance,
		Timestamp:    input.Timestamp,
	}

	signal, err := strat.OnTick(ctx, ctxData)
	if err != nil {
		return nil, fmt.Errorf("strategy tick: %w", err)
	}

	if signal != nil {
		rc.LastSignal = signal
		rc.LastTick = input.Timestamp
	}
	return signal, nil
}

func (e *Engine) OnOrderFilled(ctx context.Context, input OrderFilledInput) error {
	e.mu.RLock()
	rc, ok := e.contexts[input.InstanceID]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("instance %s not found", input.InstanceID)
	}

	strat, err := e.registry.Get(rc.StrategyID)
	if err != nil {
		return fmt.Errorf("get strategy: %w", err)
	}

	if err := strat.OnOrderFilled(ctx, input.OrderID, input.FilledQty, input.FillPrice); err != nil {
		return fmt.Errorf("on order filled: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	rc.Position = input.NewPosition
	rc.Balance = input.NewBalance
	return nil
}

func (e *Engine) OnOrderCanceled(ctx context.Context, instanceID, orderID string) error {
	e.mu.RLock()
	rc, ok := e.contexts[instanceID]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	strat, err := e.registry.Get(rc.StrategyID)
	if err != nil {
		return fmt.Errorf("get strategy: %w", err)
	}

	return strat.OnOrderCanceled(ctx, orderID)
}

func (e *Engine) UpdatePosition(ctx context.Context, instanceID string, pos strategy.PositionInfo) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if rc, ok := e.contexts[instanceID]; ok {
		rc.Position = pos
	}
}

func (e *Engine) UpdateBalance(ctx context.Context, instanceID string, bal strategy.BalanceInfo) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if rc, ok := e.contexts[instanceID]; ok {
		rc.Balance = bal
	}
}

func (e *Engine) Reset(ctx context.Context, instanceID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	rc, ok := e.contexts[instanceID]
	if !ok {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	strat, err := e.registry.Get(rc.StrategyID)
	if err != nil {
		return fmt.Errorf("get strategy: %w", err)
	}

	if err := strat.Reset(); err != nil {
		return fmt.Errorf("reset strategy: %w", err)
	}

	delete(e.contexts, instanceID)
	return nil
}

type TickInput struct {
	InstanceID string
	StrategyID string
	Symbol     string
	LastPrice  decimal.Decimal
	BidPrice   decimal.Decimal
	AskPrice   decimal.Decimal
	Volume24h  decimal.Decimal
	High24h    decimal.Decimal
	Low24h     decimal.Decimal
	Timestamp  time.Time
	Parameters map[string]interface{}
}

type OrderFilledInput struct {
	InstanceID  string
	OrderID     string
	FilledQty   decimal.Decimal
	FillPrice   decimal.Decimal
	NewPosition strategy.PositionInfo
	NewBalance  strategy.BalanceInfo
}
