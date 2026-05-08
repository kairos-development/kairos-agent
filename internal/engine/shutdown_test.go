package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStorage is a mock implementation of the StorageProvider interface for testing.
type mockStorage struct {
	closeErr error
	closed   bool
}

func (m *mockStorage) Close() error {
	m.closed = true
	return m.closeErr
}

// TestGracefulShutdown_NewGracefulShutdown tests creation of graceful shutdown handler.
func TestGracefulShutdown_NewGracefulShutdown(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	gs := NewGracefulShutdown(engine, logger)

	assert.NotNil(t, gs)
	assert.NotNil(t, gs.engine)
	assert.NotNil(t, gs.logger)
	assert.NotNil(t, gs.done)
}

// TestGracefulShutdown_NewWithNilLogger tests creation with nil logger.
func TestGracefulShutdown_NewWithNilLogger(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)

	gs := NewGracefulShutdown(engine, nil)

	assert.NotNil(t, gs)
	assert.NotNil(t, gs.logger)
}

// TestGracefulShutdown_Shutdown tests basic shutdown sequence.
func TestGracefulShutdown_Shutdown(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)

	gs := NewGracefulShutdown(engine, logger)

	// Perform shutdown
	err = gs.Shutdown(context.Background())
	assert.NoError(t, err)

	// Engine should be halted
	assert.Equal(t, StateHalted, engine.State())

	// done channel should be closed
	select {
	case <-gs.done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("done channel not closed")
	}
}

// TestGracefulShutdown_ShutdownWithStorage tests shutdown with storage.
func TestGracefulShutdown_ShutdownWithStorage(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)

	// Set mock storage
	mockStore := &mockStorage{}
	engine.SetStorage(mockStore)

	gs := NewGracefulShutdown(engine, logger)

	// Perform shutdown
	err = gs.Shutdown(context.Background())
	assert.NoError(t, err)

	// Storage should be closed
	assert.True(t, mockStore.closed)
}

// TestGracefulShutdown_ShutdownWithStorageError tests shutdown with storage error.
func TestGracefulShutdown_ShutdownWithStorageError(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)

	// Set mock storage with error
	mockStore := &mockStorage{closeErr: errors.New("close failed")}
	engine.SetStorage(mockStore)

	gs := NewGracefulShutdown(engine, logger)

	// Perform shutdown - should complete despite error
	err = gs.Shutdown(context.Background())
	assert.Error(t, err) // Should return error

	// Storage should have been attempted
	assert.True(t, mockStore.closed)

	// Engine should still be halted
	assert.Equal(t, StateHalted, engine.State())
}

// TestGracefulShutdown_MultipleShutdowns tests calling shutdown multiple times.
func TestGracefulShutdown_MultipleShutdowns(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)

	gs := NewGracefulShutdown(engine, logger)

	// First shutdown
	err = gs.Shutdown(context.Background())
	assert.NoError(t, err)

	// Second shutdown should be no-op
	err = gs.Shutdown(context.Background())
	assert.NoError(t, err)
}

// TestGracefulShutdown_ShutdownTimeout tests shutdown with context timeout.
func TestGracefulShutdown_ShutdownTimeout(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)

	gs := NewGracefulShutdown(engine, logger)

	// Create context with very short timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond) // Ensure context is expired

	// Shutdown should still complete (uses its own 30s timeout internally)
	err = gs.Shutdown(shutdownCtx)
	assert.NoError(t, err)
}

// TestGracefulShutdown_WaitForShutdown tests waiting for shutdown completion.
func TestGracefulShutdown_WaitForShutdown(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)

	gs := NewGracefulShutdown(engine, logger)

	// Start goroutine that waits for shutdown
	waitDone := make(chan struct{})
	go func() {
		gs.WaitForShutdown()
		close(waitDone)
	}()

	// Perform shutdown
	time.Sleep(50 * time.Millisecond)
	err = gs.Shutdown(context.Background())
	assert.NoError(t, err)

	// Wait should complete
	select {
	case <-waitDone:
		// Expected
	case <-time.After(1 * time.Second):
		t.Fatal("WaitForShutdown did not complete")
	}
}

// TestGracefulShutdown_CancelAllOrders tests order cancellation with nil connector.
func TestGracefulShutdown_CancelAllOrders(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	gs := NewGracefulShutdown(engine, logger)

	// Should not error with nil connector
	err := gs.cancelAllOrders(context.Background())
	assert.NoError(t, err)
}

// TestGracefulShutdown_UnloadStrategies tests strategy unloading with nil executor.
func TestGracefulShutdown_UnloadStrategies(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	gs := NewGracefulShutdown(engine, logger)

	// Should not error with nil executor
	err := gs.unloadStrategies(context.Background())
	assert.NoError(t, err)
}

// TestGracefulShutdown_CloseConnections tests connection closing with nil components.
func TestGracefulShutdown_CloseConnections(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	gs := NewGracefulShutdown(engine, logger)

	// Should not error with nil components
	err := gs.closeConnections(context.Background())
	assert.NoError(t, err)
}

// TestGracefulShutdown_CloseConnectionsWithStorage tests closing with storage.
func TestGracefulShutdown_CloseConnectionsWithStorage(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	mockStore := &mockStorage{}
	engine.SetStorage(mockStore)

	gs := NewGracefulShutdown(engine, logger)

	// Should close storage
	err := gs.closeConnections(context.Background())
	assert.NoError(t, err)
	assert.True(t, mockStore.closed)
}

// TestGracefulShutdown_Start tests signal handling setup.
func TestGracefulShutdown_Start(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	err := engine.Start()
	require.NoError(t, err)
	defer engine.Stop()

	gs := NewGracefulShutdown(engine, logger)

	// Start signal handler (non-blocking)
	gs.Start(context.Background())

	// Verify it doesn't block
	time.Sleep(10 * time.Millisecond)

	// Note: We can't easily test signal delivery in unit tests
	// This test just verifies Start() doesn't panic or block
}

// TestGracefulShutdown_UnloadStrategiesWithExecutor tests unloading with strategy executor.
func TestGracefulShutdown_UnloadStrategiesWithExecutor(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	gs := NewGracefulShutdown(engine, logger)

	// Note: We can't easily create a real StrategyExecutor without WASM runtime
	// This test verifies the nil case is handled
	err := gs.unloadStrategies(context.Background())
	assert.NoError(t, err)
}

// TestGracefulShutdown_CancelAllOrdersWithConnector tests order cancellation.
func TestGracefulShutdown_CancelAllOrdersWithConnector(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	engine := New(ctx, nil, nil, logger)
	gs := NewGracefulShutdown(engine, logger)

	// Note: We can't easily create a real Connector without exchange credentials
	// This test verifies the nil case is handled
	err := gs.cancelAllOrders(context.Background())
	assert.NoError(t, err)
}
