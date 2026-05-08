package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kairos-development/kairos-agent/internal/strategy"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/tetratelabs/wazero/api"
)

// WasmStrategy wraps a WASM module as a Strategy implementation.
type WasmStrategy struct {
	id        string
	name      string
	version   string
	instance  api.Module
	runtime   *Runtime
	logger    *logrus.Logger
	memory    api.Memory
	allocator *memoryAllocator
}

// memoryAllocator manages WASM linear memory allocation.
type memoryAllocator struct {
	mu         sync.Mutex
	nextOffset uint32
	maxSize    uint32
}

// newMemoryAllocator creates a new memory allocator.
func newMemoryAllocator() *memoryAllocator {
	return &memoryAllocator{
		nextOffset: 65536,       // Start at 64KB to avoid stack/data sections
		maxSize:    1024 * 1024, // 1MB max per allocation
	}
}

// allocate reserves memory and returns the offset.
func (ma *memoryAllocator) allocate(size uint32) (uint32, error) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	if size > ma.maxSize {
		return 0, fmt.Errorf("allocation too large: %d bytes (max %d)", size, ma.maxSize)
	}

	offset := ma.nextOffset
	ma.nextOffset += size

	// Align to 8-byte boundary
	if ma.nextOffset%8 != 0 {
		ma.nextOffset += 8 - (ma.nextOffset % 8)
	}

	return offset, nil
}

// reset resets the allocator back to its initial state
func (ma *memoryAllocator) reset() {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	ma.nextOffset = 65536
}

// NewWasmStrategy creates a new WASM-based strategy.
func NewWasmStrategy(id, name, version string, instance api.Module, runtime *Runtime, logger *logrus.Logger) *WasmStrategy {
	if logger == nil {
		logger = logrus.New()
	}

	return &WasmStrategy{
		id:        id,
		name:      name,
		version:   version,
		instance:  instance,
		runtime:   runtime,
		logger:    logger,
		memory:    instance.Memory(),
		allocator: newMemoryAllocator(),
	}
}

// Name returns the strategy name.
func (ws *WasmStrategy) Name() string {
	return ws.name
}

// Version returns the strategy version.
func (ws *WasmStrategy) Version() string {
	return ws.version
}

// OnTick is called on each market tick.
func (ws *WasmStrategy) OnTick(ctx context.Context, data *strategy.StrategyContext) (*strategy.Signal, error) {
	defer ws.allocator.reset()

	// Serialize context to JSON
	contextJSON, err := json.Marshal(data)
	if err != nil {
		ws.logger.WithError(err).Error("Failed to marshal strategy context")
		return nil, fmt.Errorf("marshal context: %w", err)
	}

	// Write context to WASM memory
	contextPtr, err := ws.writeToMemory(contextJSON)
	if err != nil {
		ws.logger.WithError(err).Error("Failed to write context to memory")
		return nil, fmt.Errorf("write to memory: %w", err)
	}

	// Call OnTick function
	results, err := ws.runtime.CallFunction(ctx, ws.instance, "on_tick", uint64(contextPtr), uint64(len(contextJSON)))
	if err != nil {
		ws.logger.WithError(err).Error("Failed to call on_tick")
		return nil, fmt.Errorf("call on_tick: %w", err)
	}

	// Parse result
	if len(results) == 0 {
		return nil, nil
	}

	// Read signal from memory
	signalPtr := results[0]
	signalLen := results[1]

	signalJSON, ok := ws.memory.Read(uint32(signalPtr), uint32(signalLen))
	if !ok {
		return nil, fmt.Errorf("failed to read signal from memory")
	}

	// Unmarshal signal
	var signal strategy.Signal
	if err := json.Unmarshal(signalJSON, &signal); err != nil {
		ws.logger.WithError(err).Error("Failed to unmarshal signal")
		return nil, fmt.Errorf("unmarshal signal: %w", err)
	}

	ws.logger.WithFields(logrus.Fields{
		"action":     signal.Action,
		"symbol":     signal.Symbol,
		"quantity":   signal.Quantity.String(),
		"confidence": signal.Confidence.String(),
	}).Debug("Strategy generated signal")

	return &signal, nil
}

// OnOrderFilled is called when an order is filled.
func (ws *WasmStrategy) OnOrderFilled(ctx context.Context, orderID string, filledQty, filledPrice decimal.Decimal) error {
	defer ws.allocator.reset()

	// Serialize order fill data
	fillData := map[string]interface{}{
		"order_id":     orderID,
		"filled_qty":   filledQty.String(),
		"filled_price": filledPrice.String(),
	}

	fillJSON, err := json.Marshal(fillData)
	if err != nil {
		ws.logger.WithError(err).Error("Failed to marshal order fill data")
		return fmt.Errorf("marshal fill data: %w", err)
	}

	// Write to memory
	fillPtr, err := ws.writeToMemory(fillJSON)
	if err != nil {
		return fmt.Errorf("write to memory: %w", err)
	}

	// Call OnOrderFilled function
	_, err = ws.runtime.CallFunction(ctx, ws.instance, "on_order_filled", uint64(fillPtr), uint64(len(fillJSON)))
	if err != nil {
		ws.logger.WithError(err).WithField("order_id", orderID).Error("Failed to call on_order_filled")
		return fmt.Errorf("call on_order_filled: %w", err)
	}

	return nil
}

// OnOrderCanceled is called when an order is canceled.
func (ws *WasmStrategy) OnOrderCanceled(ctx context.Context, orderID string) error {
	defer ws.allocator.reset()

	// Serialize order cancel data
	cancelData := map[string]interface{}{
		"order_id": orderID,
	}

	cancelJSON, err := json.Marshal(cancelData)
	if err != nil {
		ws.logger.WithError(err).Error("Failed to marshal order cancel data")
		return fmt.Errorf("marshal cancel data: %w", err)
	}

	// Write to memory
	cancelPtr, err := ws.writeToMemory(cancelJSON)
	if err != nil {
		return fmt.Errorf("write to memory: %w", err)
	}

	// Call OnOrderCanceled function
	_, err = ws.runtime.CallFunction(ctx, ws.instance, "on_order_canceled", uint64(cancelPtr), uint64(len(cancelJSON)))
	if err != nil {
		ws.logger.WithError(err).WithField("order_id", orderID).Error("Failed to call on_order_canceled")
		return fmt.Errorf("call on_order_canceled: %w", err)
	}

	return nil
}

// GetParameters returns the strategy parameters.
func (ws *WasmStrategy) GetParameters() map[string]interface{} {
	// Use a timeout context since this is a synchronous call
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Call get_parameters function if exported
	results, err := ws.runtime.CallFunction(ctx, ws.instance, "get_parameters")
	if err != nil {
		ws.logger.WithError(err).Debug("get_parameters not implemented or failed")
		return make(map[string]interface{})
	}

	if len(results) < 2 {
		return make(map[string]interface{})
	}

	// Read JSON from memory
	ptr := uint32(results[0])
	length := uint32(results[1])

	data, ok := ws.memory.Read(ptr, length)
	if !ok {
		ws.logger.Error("Failed to read parameters from memory")
		return make(map[string]interface{})
	}

	var params map[string]interface{}
	if err := json.Unmarshal(data, &params); err != nil {
		ws.logger.WithError(err).Error("Failed to unmarshal parameters")
		return make(map[string]interface{})
	}

	return params
}

// SetParameters updates strategy parameters.
func (ws *WasmStrategy) SetParameters(params map[string]interface{}) error {
	defer ws.allocator.reset()

	// Use a timeout context since this is a synchronous call
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Serialize parameters
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal parameters: %w", err)
	}

	// Write to memory
	ptr, err := ws.writeToMemory(paramsJSON)
	if err != nil {
		return fmt.Errorf("write to memory: %w", err)
	}

	// Call set_parameters function if exported
	_, err = ws.runtime.CallFunction(ctx, ws.instance, "set_parameters", uint64(ptr), uint64(len(paramsJSON)))
	if err != nil {
		ws.logger.WithError(err).Debug("set_parameters not implemented or failed")
		return fmt.Errorf("call set_parameters: %w", err)
	}

	ws.logger.WithField("param_count", len(params)).Debug("Strategy parameters updated")
	return nil
}

// Reset resets the strategy state.
func (ws *WasmStrategy) Reset() error {
	// Use a timeout context since this is a synchronous call
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Call reset function
	_, err := ws.runtime.CallFunction(ctx, ws.instance, "reset")
	if err != nil {
		ws.logger.WithError(err).Error("Failed to call reset")
		return fmt.Errorf("call reset: %w", err)
	}

	// Reset allocator
	ws.allocator = newMemoryAllocator()

	ws.logger.Info("Strategy reset")
	return nil
}

// Close closes the WASM instance and releases resources.
func (ws *WasmStrategy) Close(ctx context.Context) error {
	if ws.instance != nil {
		if err := ws.instance.Close(ctx); err != nil {
			ws.logger.WithError(err).Error("Failed to close WASM instance")
			return fmt.Errorf("close instance: %w", err)
		}
		ws.logger.Info("WASM instance closed")
	}
	return nil
}

// writeToMemory writes data to WASM linear memory and returns the pointer.
func (ws *WasmStrategy) writeToMemory(data []byte) (uint32, error) {
	// Allocate memory
	offset, err := ws.allocator.allocate(uint32(len(data)))
	if err != nil {
		return 0, err
	}

	// Ensure memory is large enough
	memSize := ws.memory.Size()
	requiredSize := offset + uint32(len(data))
	if requiredSize > memSize {
		// Try to grow memory
		pagesToGrow := (requiredSize - memSize + 65535) / 65536
		_, ok := ws.memory.Grow(pagesToGrow)
		if !ok {
			return 0, fmt.Errorf("failed to grow memory from %d to %d bytes", memSize, requiredSize)
		}
	}

	// Write to memory
	ok := ws.memory.Write(offset, data)
	if !ok {
		return 0, fmt.Errorf("failed to write to memory at offset %d", offset)
	}

	return offset, nil
}

// StrategyExecutor manages strategy execution and lifecycle.
type StrategyExecutor struct {
	mu         sync.RWMutex
	runtime    *Runtime
	strategies map[string]*WasmStrategy
	logger     *logrus.Logger
}

// NewStrategyExecutor creates a new strategy executor.
func NewStrategyExecutor(runtime *Runtime, logger *logrus.Logger) *StrategyExecutor {
	if logger == nil {
		logger = logrus.New()
	}

	return &StrategyExecutor{
		runtime:    runtime,
		strategies: make(map[string]*WasmStrategy),
		logger:     logger,
	}
}

// LoadStrategy loads a WASM strategy from a module ID.
func (se *StrategyExecutor) LoadStrategy(ctx context.Context, strategyID, name, version string) (*WasmStrategy, error) {
	se.mu.Lock()
	defer se.mu.Unlock()

	// Check if already loaded
	if _, exists := se.strategies[strategyID]; exists {
		return nil, fmt.Errorf("strategy %s already loaded", strategyID)
	}

	// Instantiate module
	instance, err := se.runtime.InstantiateModule(ctx, strategyID)
	if err != nil {
		se.logger.WithError(err).WithField("strategy_id", strategyID).Error("Failed to instantiate strategy module")
		return nil, fmt.Errorf("instantiate module: %w", err)
	}

	// Create strategy wrapper
	wasmStrategy := NewWasmStrategy(strategyID, name, version, instance, se.runtime, se.logger)

	// Store in registry
	se.strategies[strategyID] = wasmStrategy

	se.logger.WithFields(logrus.Fields{
		"strategy_id": strategyID,
		"name":        name,
		"version":     version,
	}).Info("Strategy loaded")

	return wasmStrategy, nil
}

// GetStrategy retrieves a loaded strategy.
func (se *StrategyExecutor) GetStrategy(strategyID string) (*WasmStrategy, error) {
	se.mu.RLock()
	defer se.mu.RUnlock()

	strategy, exists := se.strategies[strategyID]
	if !exists {
		return nil, fmt.Errorf("strategy %s not loaded", strategyID)
	}
	return strategy, nil
}

// UnloadStrategy unloads a strategy and closes its WASM instance.
func (se *StrategyExecutor) UnloadStrategy(ctx context.Context, strategyID string) error {
	se.mu.Lock()
	defer se.mu.Unlock()

	strategy, exists := se.strategies[strategyID]
	if !exists {
		return fmt.Errorf("strategy %s not found", strategyID)
	}

	// Close WASM instance
	if err := strategy.Close(ctx); err != nil {
		se.logger.WithError(err).WithField("strategy_id", strategyID).Warn("Failed to close strategy cleanly")
	}

	delete(se.strategies, strategyID)
	se.logger.WithField("strategy_id", strategyID).Info("Strategy unloaded")

	return nil
}

// UnloadAll unloads all strategies.
func (se *StrategyExecutor) UnloadAll(ctx context.Context) error {
	se.mu.Lock()
	defer se.mu.Unlock()

	var lastErr error
	for id, strategy := range se.strategies {
		if err := strategy.Close(ctx); err != nil {
			se.logger.WithError(err).WithField("strategy_id", id).Warn("Failed to close strategy")
			lastErr = err
		}
		delete(se.strategies, id)
	}

	se.logger.Info("All strategies unloaded")
	return lastErr
}

// ListLoadedStrategies returns all loaded strategies.
func (se *StrategyExecutor) ListLoadedStrategies() map[string]*WasmStrategy {
	se.mu.RLock()
	defer se.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]*WasmStrategy, len(se.strategies))
	for k, v := range se.strategies {
		result[k] = v
	}
	return result
}
