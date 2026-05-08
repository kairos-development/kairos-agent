package runtime

import (
	"testing"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStateMachine_NilLogger(t *testing.T) {
	sm := NewStateMachine(nil)
	assert.NotNil(t, sm)
	assert.NotNil(t, sm.logger)
	assert.Equal(t, entity.RunModeIdle, sm.current)
	assert.Equal(t, entity.RunModeIdle, sm.previous)
	assert.Empty(t, sm.listeners)
}

func TestNewStateMachine_WithLogger(t *testing.T) {
	logger := logrus.New()
	sm := NewStateMachine(logger)
	assert.Equal(t, logger, sm.logger)
}

func TestCurrent(t *testing.T) {
	sm := NewStateMachine(nil)
	assert.Equal(t, entity.RunModeIdle, sm.Current())

	sm.current = entity.RunModePaperTrading
	assert.Equal(t, entity.RunModePaperTrading, sm.Current())
}

func TestSubscribe(t *testing.T) {
	sm := NewStateMachine(nil)
	ch := make(chan StateTransition, 1)

	id := sm.Subscribe(ch)
	assert.Equal(t, uint64(1), id)

	id2 := sm.Subscribe(ch)
	assert.Equal(t, uint64(2), id2)

	assert.Len(t, sm.listeners, 2)
}

func TestUnsubscribe(t *testing.T) {
	sm := NewStateMachine(nil)
	ch := make(chan StateTransition, 1)

	id1 := sm.Subscribe(ch)
	id2 := sm.Subscribe(ch)
	assert.Len(t, sm.listeners, 2)

	sm.Unsubscribe(id1)
	assert.Len(t, sm.listeners, 1)
	_, exists := sm.listeners[id1]
	assert.False(t, exists)
	_, exists = sm.listeners[id2]
	assert.True(t, exists)
}

func TestUnsubscribe_NonExistent(t *testing.T) {
	sm := NewStateMachine(nil)
	// Should not panic
	sm.Unsubscribe(999)
	assert.Empty(t, sm.listeners)
}

func TestSubscribe_Unsubscribe_TransitionNotifiesOnlyActive(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	sm := NewStateMachine(logger)

	ch1 := make(chan StateTransition, 10)
	ch2 := make(chan StateTransition, 10)

	id1 := sm.Subscribe(ch1)
	sm.Subscribe(ch2)

	// Unsubscribe ch1
	sm.Unsubscribe(id1)

	err := sm.Transition(entity.RunModePaperTrading, "test")
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	// ch1 should NOT receive, ch2 should
	assert.Len(t, ch1, 0)
	assert.Len(t, ch2, 1)
}

func TestTransition_IdleToPaper(t *testing.T) {
	sm := NewStateMachine(nil)
	err := sm.Transition(entity.RunModePaperTrading, "test")
	require.NoError(t, err)
	assert.Equal(t, entity.RunModePaperTrading, sm.current)
	assert.Equal(t, entity.RunModeIdle, sm.previous)
}

func TestTransition_PaperToLive(t *testing.T) {
	sm := NewStateMachine(nil)
	err := sm.Transition(entity.RunModePaperTrading, "step1")
	require.NoError(t, err)

	err = sm.Transition(entity.RunModeLiveTrading, "step2")
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeLiveTrading, sm.current)
	assert.Equal(t, entity.RunModePaperTrading, sm.previous)
}

func TestTransition_IdleToHalted(t *testing.T) {
	sm := NewStateMachine(nil)
	err := sm.Transition(entity.RunModeHalted, "integrity failure")
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeHalted, sm.current)
	assert.Equal(t, entity.RunModeIdle, sm.previous)
}

func TestTransition_HaltedToPaperBlocked(t *testing.T) {
	sm := NewStateMachine(nil)
	err := sm.Transition(entity.RunModeHalted, "failure")
	require.NoError(t, err)

	err = sm.Transition(entity.RunModePaperTrading, "illegal")
	assert.Error(t, err)

	var invalidErr *InvalidTransitionError
	require.ErrorAs(t, err, &invalidErr)
	assert.Equal(t, entity.RunModeHalted, invalidErr.From)
	assert.Equal(t, entity.RunModePaperTrading, invalidErr.To)
}

func TestSMTransition_HaltedToLiveBlocked(t *testing.T) {
	sm := NewStateMachine(nil)
	err := sm.Transition(entity.RunModeHalted, "failure")
	require.NoError(t, err)

	err = sm.Transition(entity.RunModeLiveTrading, "illegal")
	assert.Error(t, err)
}

func TestSMTransition_HaltedToIdleAllowed(t *testing.T) {
	sm := NewStateMachine(nil)
	err := sm.Transition(entity.RunModeHalted, "failure")
	require.NoError(t, err)

	err = sm.Transition(entity.RunModeIdle, "restart")
	require.NoError(t, err)
	assert.Equal(t, entity.RunModeIdle, sm.current)
}

func TestSMTransition_NoopSameState(t *testing.T) {
	sm := NewStateMachine(nil)
	err := sm.Transition(entity.RunModeIdle, "no-op")
	assert.NoError(t, err)
	assert.Equal(t, entity.RunModeIdle, sm.current)
}

func TestTransition_Chain(t *testing.T) {
	sm := NewStateMachine(nil)

	// Idle -> Paper -> Live -> Idle -> Halted -> Idle
	transitions := []struct {
		to     entity.RunMode
		reason string
	}{
		{entity.RunModePaperTrading, "start paper"},
		{entity.RunModeLiveTrading, "go live"},
		{entity.RunModeIdle, "stop"},
		{entity.RunModeHalted, "failure"},
		{entity.RunModeIdle, "restart"},
	}

	for _, tr := range transitions {
		err := sm.Transition(tr.to, tr.reason)
		require.NoError(t, err)
	}

	assert.Equal(t, entity.RunModeIdle, sm.current)
}

func TestInvalidTransitionError_Error(t *testing.T) {
	err := &InvalidTransitionError{
		From: entity.RunModeHalted,
		To:   entity.RunModeLiveTrading,
	}
	expected := "invalid transition from halted to live_trading"
	assert.Equal(t, expected, err.Error())
}

func TestInvalidTransitionError_ErrorString(t *testing.T) {
	err := InvalidTransitionError{
		From: entity.RunModeHalted,
		To:   entity.RunModePaperTrading,
	}
	assert.Contains(t, err.Error(), "invalid transition")
	assert.Contains(t, err.Error(), "halted")
	assert.Contains(t, err.Error(), "paper_trading")
}

func TestNotify_NoListeners(t *testing.T) {
	sm := NewStateMachine(nil)
	// Should not panic with no listeners
	sm.notify(StateTransition{
		From:      entity.RunModeIdle,
		To:        entity.RunModePaperTrading,
		Timestamp: time.Now(),
		Reason:    "test",
	})
}

func TestNotify_ListenerFull(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	sm := NewStateMachine(logger)

	// Create a channel with buffer 0 (unbuffered) and don't read from it
	ch := make(chan StateTransition)
	sm.Subscribe(ch)

	// Transition should not block even if listener can't receive
	err := sm.Transition(entity.RunModePaperTrading, "test")
	require.NoError(t, err)

	// Give it a moment to attempt notification
	time.Sleep(50 * time.Millisecond)
}

func TestSubscribe_TransitionPreservesTimestamp(t *testing.T) {
	sm := NewStateMachine(nil)
	ch := make(chan StateTransition, 1)
	sm.Subscribe(ch)

	before := time.Now().UTC()
	err := sm.Transition(entity.RunModePaperTrading, "timestamp test")
	require.NoError(t, err)

	select {
	case tr := <-ch:
		assert.False(t, tr.Timestamp.IsZero())
		assert.True(t, tr.Timestamp.After(before) || tr.Timestamp.Equal(before))
		assert.Equal(t, "timestamp test", tr.Reason)
	case <-time.After(time.Second):
		t.Fatal("expected transition notification")
	}
}

func TestTransition_AllModesFromIdle(t *testing.T) {
	modes := []entity.RunMode{
		entity.RunModePaperTrading,
		entity.RunModeLiveTrading,
		entity.RunModeHalted,
		entity.RunModeIdle,
	}

	for _, mode := range modes {
		sm := NewStateMachine(nil)
		err := sm.Transition(mode, "test")
		require.NoError(t, err)
		assert.Equal(t, mode, sm.current)
	}
}
