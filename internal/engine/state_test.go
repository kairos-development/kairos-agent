package engine

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestStateMachine_NewStateMachineWithNilLogger tests creation with nil logger.
func TestStateMachine_NewStateMachineWithNilLogger(t *testing.T) {
	sm := NewStateMachine(nil)

	assert.NotNil(t, sm)
	assert.NotNil(t, sm.logger)
	assert.Equal(t, StateIdle, sm.Current())
}

// TestStateMachine_InitialState tests initial state is Idle.
func TestStateMachine_InitialState(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	sm := NewStateMachine(logger)

	assert.Equal(t, StateIdle, sm.Current())
	assert.Equal(t, StateIdle, sm.Previous())
}

// TestStateMachine_ValidTransition tests valid state transitions.
func TestStateMachine_ValidTransition(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	sm := NewStateMachine(logger)

	// Idle -> LiveTrading
	err := sm.Transition(StateLiveTrading, "start trading")
	assert.NoError(t, err)
	assert.Equal(t, StateLiveTrading, sm.Current())
	assert.Equal(t, StateIdle, sm.Previous())

	// LiveTrading -> Halted
	err = sm.Transition(StateHalted, "emergency stop")
	assert.NoError(t, err)
	assert.Equal(t, StateHalted, sm.Current())
	assert.Equal(t, StateLiveTrading, sm.Previous())

	// Halted -> Idle (only valid transition from Halted)
	err = sm.Transition(StateIdle, "resume")
	assert.NoError(t, err)
	assert.Equal(t, StateIdle, sm.Current())
	assert.Equal(t, StateHalted, sm.Previous())
}

// TestStateMachine_InvalidTransition tests invalid state transitions.
func TestStateMachine_InvalidTransition(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	sm := NewStateMachine(logger)

	// Transition to same state should fail
	err := sm.Transition(StateIdle, "no-op")
	assert.Error(t, err)
	assert.IsType(t, &InvalidTransitionError{}, err)

	// Transition to LiveTrading
	err = sm.Transition(StateLiveTrading, "start")
	assert.NoError(t, err)

	// Halt
	err = sm.Transition(StateHalted, "halt")
	assert.NoError(t, err)

	// Halted -> LiveTrading (invalid, must go to Idle first)
	err = sm.Transition(StateLiveTrading, "try to resume")
	assert.Error(t, err)
	assert.IsType(t, &InvalidTransitionError{}, err)
}

// TestStateMachine_HaltFromAnyState tests halting from any state.
func TestStateMachine_HaltFromAnyState(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	states := []State{StateIdle, StateScanning, StateLiveTrading, StatePaperTrading, StateBacktesting, StateOptimizing}

	for _, state := range states {
		sm := NewStateMachine(logger)

		// Transition to test state
		if state != StateIdle {
			err := sm.Transition(state, "setup")
			assert.NoError(t, err)
		}

		// Halt should always work
		err := sm.Transition(StateHalted, "emergency")
		assert.NoError(t, err, "Should be able to halt from %s", state.String())
		assert.Equal(t, StateHalted, sm.Current())
	}
}

// TestStateMachine_ToIdleFromAnyState tests transitioning to Idle from any state.
func TestStateMachine_ToIdleFromAnyState(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	states := []State{StateScanning, StateLiveTrading, StatePaperTrading, StateBacktesting, StateOptimizing}

	for _, state := range states {
		sm := NewStateMachine(logger)

		// Transition to test state
		err := sm.Transition(state, "setup")
		assert.NoError(t, err)

		// To Idle should always work (except from Halted)
		err = sm.Transition(StateIdle, "stop")
		assert.NoError(t, err, "Should be able to go to Idle from %s", state.String())
		assert.Equal(t, StateIdle, sm.Current())
	}
}

// TestStateMachine_SubscribeAndNotify tests listener notification.
func TestStateMachine_SubscribeAndNotify(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	sm := NewStateMachine(logger)

	// Subscribe
	ch := make(chan StateTransition, 10)
	id := sm.Subscribe(ch)
	assert.NotZero(t, id)

	// Transition
	err := sm.Transition(StateLiveTrading, "start trading")
	assert.NoError(t, err)

	// Wait for notification
	select {
	case transition := <-ch:
		assert.Equal(t, StateIdle, transition.From)
		assert.Equal(t, StateLiveTrading, transition.To)
		assert.Equal(t, "start trading", transition.Reason)
		assert.False(t, transition.Timestamp.IsZero())
	case <-time.After(1 * time.Second):
		t.Fatal("Transition notification not received")
	}
}

// TestStateMachine_Unsubscribe tests unsubscribe functionality.
func TestStateMachine_Unsubscribe(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	sm := NewStateMachine(logger)

	// Subscribe
	ch := make(chan StateTransition, 10)
	id := sm.Subscribe(ch)

	// Unsubscribe
	sm.Unsubscribe(id)

	// Transition
	err := sm.Transition(StateLiveTrading, "start")
	assert.NoError(t, err)

	// Should not receive notification
	select {
	case <-ch:
		t.Fatal("Received notification after unsubscribe")
	case <-time.After(100 * time.Millisecond):
		// Expected - no notification
	}
}

// TestStateMachine_MultipleListeners tests multiple listeners.
func TestStateMachine_MultipleListeners(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	sm := NewStateMachine(logger)

	// Subscribe multiple listeners
	ch1 := make(chan StateTransition, 10)
	ch2 := make(chan StateTransition, 10)
	ch3 := make(chan StateTransition, 10)

	sm.Subscribe(ch1)
	sm.Subscribe(ch2)
	sm.Subscribe(ch3)

	// Transition
	err := sm.Transition(StateLiveTrading, "start")
	assert.NoError(t, err)

	// All listeners should receive
	for i, ch := range []chan StateTransition{ch1, ch2, ch3} {
		select {
		case transition := <-ch:
			assert.Equal(t, StateIdle, transition.From)
			assert.Equal(t, StateLiveTrading, transition.To)
		case <-time.After(1 * time.Second):
			t.Fatalf("Listener %d did not receive notification", i+1)
		}
	}
}

// TestStateMachine_ConcurrentTransitions tests concurrent transition attempts.
func TestStateMachine_ConcurrentTransitions(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	sm := NewStateMachine(logger)

	var wg sync.WaitGroup
	var successCount int32
	var errorCount int32

	// Try concurrent transitions to different states
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			state := StateLiveTrading
			if n%2 == 0 {
				state = StatePaperTrading
			}

			err := sm.Transition(state, "concurrent")
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&errorCount, 1)
			}
		}(i)
	}

	wg.Wait()

	// At least one should succeed
	assert.Greater(t, atomic.LoadInt32(&successCount), int32(0), "At least one transition should succeed")
	// State should have changed from Idle
	assert.NotEqual(t, StateIdle, sm.Current(), "State should have changed")
	// Some may fail due to race conditions or invalid transitions
	total := atomic.LoadInt32(&successCount) + atomic.LoadInt32(&errorCount)
	assert.Equal(t, int32(10), total, "All goroutines should complete")
}

// TestStateMachine_TransitionedAt tests timestamp tracking.
func TestStateMachine_TransitionedAt(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	sm := NewStateMachine(logger)

	before := time.Now().UTC()
	time.Sleep(10 * time.Millisecond)

	err := sm.Transition(StateLiveTrading, "test")
	assert.NoError(t, err)

	time.Sleep(10 * time.Millisecond)
	after := time.Now().UTC()

	transitionTime := sm.TransitionedAt()
	assert.True(t, transitionTime.After(before), "Transition time should be after 'before'")
	assert.True(t, transitionTime.Before(after), "Transition time should be before 'after'")
}

// TestStateMachine_StateString tests state string representation.
func TestStateMachine_StateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateIdle, "Idle"},
		{StateScanning, "Scanning"},
		{StateLiveTrading, "Live Trading"},
		{StatePaperTrading, "Paper Trading"},
		{StateBacktesting, "Backtesting"},
		{StateOptimizing, "Optimizing"},
		{StateHalted, "Halted"},
		{State(999), "Unknown"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.state.String())
	}
}

// TestStateMachine_SlowListener tests that slow listeners don't block transitions.
func TestStateMachine_SlowListener(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	sm := NewStateMachine(logger)

	// Subscribe slow listener with small buffer
	ch := make(chan StateTransition, 1)
	sm.Subscribe(ch)

	// First transition - should be received
	err := sm.Transition(StateLiveTrading, "first")
	assert.NoError(t, err)

	select {
	case <-ch:
		// Received
	case <-time.After(100 * time.Millisecond):
		t.Fatal("First transition not received")
	}

	// Don't drain channel - make it full

	// Second transition - should not block even if listener is slow
	start := time.Now()
	err = sm.Transition(StateIdle, "second")
	assert.NoError(t, err)
	duration := time.Since(start)

	// Should complete quickly (not block on slow listener)
	assert.Less(t, duration, 100*time.Millisecond, "Transition should not block on slow listener")
}

// TestStateMachine_InvalidTransitionError tests error type.
func TestStateMachine_InvalidTransitionError(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	sm := NewStateMachine(logger)

	// Try invalid transition (same state)
	err := sm.Transition(StateIdle, "same state")
	assert.Error(t, err)

	// Check error type and message
	var invalidErr *InvalidTransitionError
	assert.ErrorAs(t, err, &invalidErr)
	assert.Equal(t, StateIdle, invalidErr.From)
	assert.Equal(t, StateIdle, invalidErr.To)
	assert.Contains(t, err.Error(), "invalid state transition")
}
