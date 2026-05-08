package engine

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEngineLifecycle tests the complete engine lifecycle.
func TestEngineLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create engine with default config
	cfg := DefaultConfig()
	eng := New(ctx, nil, cfg, logger)

	// Start engine
	err := eng.Start()
	require.NoError(t, err, "Engine should start successfully")

	// Verify initial state
	assert.Equal(t, StateIdle, eng.State(), "Engine should start in Idle state")

	// Let engine run for a bit
	time.Sleep(2 * time.Second)

	// Transition to scanning
	err = eng.TransitionTo(StateScanning, "integration test")
	require.NoError(t, err, "Should transition to Scanning")
	assert.Equal(t, StateScanning, eng.State())

	// Transition to paper trading
	err = eng.TransitionTo(StatePaperTrading, "integration test")
	require.NoError(t, err, "Should transition to PaperTrading")
	assert.Equal(t, StatePaperTrading, eng.State())

	// Let it run in paper trading mode
	time.Sleep(2 * time.Second)

	// Transition back to idle
	err = eng.TransitionTo(StateIdle, "integration test")
	require.NoError(t, err, "Should transition back to Idle")
	assert.Equal(t, StateIdle, eng.State())

	// Stop engine
	err = eng.Stop()
	require.NoError(t, err, "Engine should stop successfully")
	assert.Equal(t, StateHalted, eng.State(), "Engine should be halted after stop")
}

// TestEngineStateTransitions tests all valid state transitions.
func TestEngineStateTransitions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	cfg := DefaultConfig()
	eng := New(ctx, nil, cfg, logger)

	err := eng.Start()
	require.NoError(t, err)
	defer eng.Stop()

	// Test all valid transitions from Idle
	states := []State{
		StateScanning,
		StateLiveTrading,
		StatePaperTrading,
		StateBacktesting,
		StateOptimizing,
	}

	for _, state := range states {
		t.Run("Idle->"+state.String(), func(t *testing.T) {
			// Ensure we're in Idle
			if eng.State() != StateIdle {
				err := eng.TransitionTo(StateIdle, "reset")
				require.NoError(t, err)
			}

			// Transition to target state
			err := eng.TransitionTo(state, "test")
			assert.NoError(t, err)
			assert.Equal(t, state, eng.State())

			// Transition back to Idle
			err = eng.TransitionTo(StateIdle, "reset")
			assert.NoError(t, err)
			assert.Equal(t, StateIdle, eng.State())
		})
	}
}

// TestEngineEventBusIntegration tests event bus integration.
func TestEngineEventBusIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	cfg := DefaultConfig()
	eng := New(ctx, nil, cfg, logger)

	err := eng.Start()
	require.NoError(t, err)
	defer eng.Stop()

	// Subscribe to state transition events
	eventBus := eng.EventBus()
	ch := make(chan Event, 10)
	subscriber := SubscriberFunc(func(ctx context.Context, event Event) {
		ch <- event
	})

	eventBus.Subscribe(EventTypeStateTransition, QoS1, subscriber)

	// Trigger a state transition
	err = eng.TransitionTo(StateLiveTrading, "test event")
	require.NoError(t, err)

	// Wait for event
	select {
	case event := <-ch:
		assert.Equal(t, EventTypeStateTransition, event.Type())
		stateEvent, ok := event.(*StateTransitionEvent)
		require.True(t, ok, "Event should be StateTransitionEvent")
		assert.Equal(t, StateIdle, stateEvent.From)
		assert.Equal(t, StateLiveTrading, stateEvent.To)
	case <-time.After(5 * time.Second):
		t.Fatal("State transition event not received within timeout")
	}
}

// TestEngineGracefulShutdown tests graceful shutdown.
func TestEngineGracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	cfg := DefaultConfig()
	eng := New(ctx, nil, cfg, logger)

	err := eng.Start()
	require.NoError(t, err)

	// Transition to live trading
	err = eng.TransitionTo(StateLiveTrading, "test")
	require.NoError(t, err)

	// Let it run
	time.Sleep(1 * time.Second)

	// Create graceful shutdown handler
	gs := NewGracefulShutdown(eng, logger)

	// Perform graceful shutdown
	err = gs.Shutdown(context.Background())
	assert.NoError(t, err, "Graceful shutdown should succeed")

	// Verify engine is halted
	assert.Equal(t, StateHalted, eng.State())
}

// TestEngineContextCancellation tests context cancellation.
func TestEngineContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithCancel(context.Background())
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	cfg := DefaultConfig()
	eng := New(ctx, nil, cfg, logger)

	err := eng.Start()
	require.NoError(t, err)

	// Let it run
	time.Sleep(1 * time.Second)

	// Cancel context
	cancel()

	// Give workers time to stop
	time.Sleep(500 * time.Millisecond)

	// Stop should still work
	err = eng.Stop()
	assert.NoError(t, err)
}

// TestEngineMultipleStartStop tests multiple start/stop cycles.
func TestEngineMultipleStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	cfg := DefaultConfig()
	eng := New(ctx, nil, cfg, logger)

	// First cycle
	err := eng.Start()
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)
	err = eng.Stop()
	require.NoError(t, err)

	// Reset to Idle
	err = eng.TransitionTo(StateIdle, "reset")
	require.NoError(t, err)

	// Second cycle
	err = eng.Start()
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)
	err = eng.Stop()
	assert.NoError(t, err)
}

// TestEngineMetricsIntegration tests metrics integration.
func TestEngineMetricsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	cfg := DefaultConfig()
	eng := New(ctx, nil, cfg, logger)

	err := eng.Start()
	require.NoError(t, err)
	defer eng.Stop()

	// Perform some state transitions
	err = eng.TransitionTo(StateLiveTrading, "test")
	require.NoError(t, err)

	err = eng.TransitionTo(StatePaperTrading, "test")
	require.NoError(t, err)

	err = eng.TransitionTo(StateIdle, "test")
	require.NoError(t, err)

	// Let metrics worker run
	time.Sleep(12 * time.Second)

	// Metrics should have been updated
	// Note: We can't easily verify Prometheus metrics in tests,
	// but we can verify the engine is still running
	assert.NotEqual(t, StateHalted, eng.State())
}
