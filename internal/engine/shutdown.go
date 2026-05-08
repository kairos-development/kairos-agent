package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// GracefulShutdown manages the shutdown sequence for the engine.
// It ensures all open orders are canceled and connections are properly closed.
type GracefulShutdown struct {
	engine *Engine
	logger *logrus.Logger
	mu     sync.Mutex
	done   chan struct{}
}

// NewGracefulShutdown creates a new graceful shutdown handler.
func NewGracefulShutdown(eng *Engine, logger *logrus.Logger) *GracefulShutdown {
	if logger == nil {
		logger = logrus.New()
	}

	return &GracefulShutdown{
		engine: eng,
		logger: logger,
		done:   make(chan struct{}),
	}
}

// Start begins listening for shutdown signals.
func (gs *GracefulShutdown) Start(ctx context.Context) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-sigCh
		gs.logger.WithField("signal", sig).Info("Received shutdown signal")
		if err := gs.Shutdown(ctx); err != nil {
			gs.logger.WithError(err).Error("Shutdown completed with errors")
		}
	}()
}

// Shutdown performs the graceful shutdown sequence.
func (gs *GracefulShutdown) Shutdown(ctx context.Context) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	select {
	case <-gs.done:
		// Already shutting down
		return nil
	default:
	}

	gs.logger.Info("Starting graceful shutdown")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var allErrs []error

	// Step 1: Transition to Halted state (if not already halted)
	gs.logger.Info("Step 1/5: Halting trading")
	if gs.engine.State() != StateHalted {
		if err := gs.engine.TransitionTo(StateHalted, "graceful shutdown initiated"); err != nil {
			gs.logger.WithError(err).Warn("Failed to transition to halted")
			allErrs = append(allErrs, fmt.Errorf("transition to halted: %w", err))
		}
	}

	// Step 2: Cancel all open orders
	gs.logger.Info("Step 2/5: Canceling open orders")
	if err := gs.cancelAllOrders(shutdownCtx); err != nil {
		gs.logger.WithError(err).Warn("Failed to cancel all orders")
		allErrs = append(allErrs, fmt.Errorf("cancel orders: %w", err))
	}

	// Step 3: Unload all WASM strategies
	gs.logger.Info("Step 3/5: Unloading strategies")
	if err := gs.unloadStrategies(shutdownCtx); err != nil {
		gs.logger.WithError(err).Warn("Failed to unload strategies")
		allErrs = append(allErrs, fmt.Errorf("unload strategies: %w", err))
	}

	// Step 4: Close database connections
	gs.logger.Info("Step 4/5: Closing database connections")
	if err := gs.closeConnections(shutdownCtx); err != nil {
		gs.logger.WithError(err).Warn("Failed to close connections")
		allErrs = append(allErrs, fmt.Errorf("close connections: %w", err))
	}

	// Step 5: Stop the engine
	gs.logger.Info("Step 5/5: Stopping engine")
	if err := gs.engine.Stop(); err != nil {
		gs.logger.WithError(err).Warn("Failed to stop engine")
		allErrs = append(allErrs, fmt.Errorf("stop engine: %w", err))
	}

	close(gs.done)
	gs.logger.Info("Graceful shutdown complete")
	return errors.Join(allErrs...)
}

// cancelAllOrders cancels all open orders on the exchange.
func (gs *GracefulShutdown) cancelAllOrders(ctx context.Context) error {
	// Get connector from engine
	connector := gs.engine.GetConnector()
	if connector == nil {
		gs.logger.Debug("No connector available, skipping order cancellation")
		return nil
	}

	// Get all open orders
	orders, err := connector.GetOpenOrders(ctx)
	if err != nil {
		gs.logger.WithError(err).Error("Failed to get open orders")
		return fmt.Errorf("get open orders: %w", err)
	}

	if len(orders) == 0 {
		gs.logger.Info("No open orders to cancel")
		return nil
	}

	gs.logger.WithField("count", len(orders)).Info("Canceling open orders")

	// Cancel orders concurrently
	var wg sync.WaitGroup
	errCh := make(chan error, len(orders))

	for _, order := range orders {
		wg.Add(1)
		go func(orderID string) {
			defer wg.Done()

			if err := connector.CancelOrder(ctx, orderID); err != nil {
				gs.logger.WithError(err).WithField("order_id", orderID).Warn("Failed to cancel order")
				errCh <- err
			} else {
				gs.logger.WithField("order_id", orderID).Debug("Order canceled")
			}
		}(order.ID)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var allErrs []error
	for err := range errCh {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) > 0 {
		gs.logger.WithField("failed_count", len(allErrs)).Warn("Some orders failed to cancel")
		return fmt.Errorf("failed to cancel %d orders: %w", len(allErrs), errors.Join(allErrs...))
	}

	gs.logger.Info("All open orders canceled successfully")
	return nil
}

// unloadStrategies unloads all WASM strategies.
func (gs *GracefulShutdown) unloadStrategies(ctx context.Context) error {
	// Get strategy executor from engine
	executor := gs.engine.GetStrategyExecutor()
	if executor == nil {
		gs.logger.Debug("No strategy executor available")
		return nil
	}

	// Unload all strategies
	if err := executor.UnloadAll(ctx); err != nil {
		return err
	}

	gs.logger.Info("All strategies unloaded")
	return nil
}

// closeConnections closes all database and network connections.
func (gs *GracefulShutdown) closeConnections(ctx context.Context) error {
	var allErrs []error

	// Close connector
	connector := gs.engine.GetConnector()
	if connector != nil {
		gs.logger.Debug("Closing exchange connector")
		if err := connector.Disconnect(ctx); err != nil {
			gs.logger.WithError(err).Warn("Failed to disconnect connector")
			allErrs = append(allErrs, fmt.Errorf("disconnect connector: %w", err))
		}
	}

	// Close storage provider
	storage := gs.engine.GetStorage()
	if storage != nil {
		gs.logger.Debug("Closing storage connections")
		if err := storage.Close(); err != nil {
			gs.logger.WithError(err).Warn("Failed to close storage")
			allErrs = append(allErrs, fmt.Errorf("close storage: %w", err))
		}
	}

	// Close event bus
	eventBus := gs.engine.GetEventBus()
	if eventBus != nil {
		gs.logger.Debug("Shutting down event bus")
		eventBus.Shutdown()
	}

	gs.logger.Info("All connections closed")
	return errors.Join(allErrs...)
}

// WaitForShutdown blocks until shutdown is complete.
func (gs *GracefulShutdown) WaitForShutdown() {
	<-gs.done
}
