package circuitbreaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	cb := New("test", Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
	})

	assert.NotNil(t, cb)
	assert.Equal(t, "test", cb.name)
	assert.Equal(t, uint32(3), cb.maxRequests)
	assert.Equal(t, 10*time.Second, cb.interval)
	assert.Equal(t, 5*time.Second, cb.timeout)
	assert.Equal(t, StateClosed, cb.State())
}

func TestNew_DefaultValues(t *testing.T) {
	cb := New("test", Config{})

	assert.Equal(t, uint32(1), cb.maxRequests)
	assert.Equal(t, 60*time.Second, cb.interval)
	assert.Equal(t, 60*time.Second, cb.timeout)
	assert.NotNil(t, cb.readyToTrip)
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	cb := New("test", Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
	})

	ctx := context.Background()
	called := false

	err := cb.Execute(ctx, func() error {
		called = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, StateClosed, cb.State())

	counts := cb.Counts()
	assert.Equal(t, uint32(1), counts.Requests)
	assert.Equal(t, uint32(1), counts.TotalSuccesses)
	assert.Equal(t, uint32(1), counts.ConsecutiveSuccesses)
	assert.Equal(t, uint32(0), counts.ConsecutiveFailures)
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	cb := New("test", Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	ctx := context.Background()
	testErr := errors.New("test error")

	// First failure
	err := cb.Execute(ctx, func() error {
		return testErr
	})

	assert.Equal(t, testErr, err)
	assert.Equal(t, StateClosed, cb.State())

	counts := cb.Counts()
	assert.Equal(t, uint32(1), counts.ConsecutiveFailures)

	// Second failure
	err = cb.Execute(ctx, func() error {
		return testErr
	})

	assert.Equal(t, testErr, err)
	assert.Equal(t, StateClosed, cb.State())

	counts = cb.Counts()
	assert.Equal(t, uint32(2), counts.ConsecutiveFailures)

	// Third failure - should trip
	err = cb.Execute(ctx, func() error {
		return testErr
	})

	assert.Equal(t, testErr, err)
	assert.Equal(t, StateOpen, cb.State())

	// Counts are reset when transitioning to Open state
	counts = cb.Counts()
	assert.Equal(t, uint32(0), counts.ConsecutiveFailures)
}

func TestCircuitBreaker_Execute_Open(t *testing.T) {
	cb := New("test", Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     100 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
	})

	ctx := context.Background()
	testErr := errors.New("test error")

	// Trip the circuit breaker
	err := cb.Execute(ctx, func() error {
		return testErr
	})
	assert.Equal(t, testErr, err)
	assert.Equal(t, StateOpen, cb.State())

	// Try to execute while open
	err = cb.Execute(ctx, func() error {
		return nil
	})

	assert.Equal(t, ErrCircuitOpen, err)
	assert.Equal(t, StateOpen, cb.State())
}

func TestCircuitBreaker_Execute_HalfOpen(t *testing.T) {
	cb := New("test", Config{
		MaxRequests: 2,
		Interval:    10 * time.Second,
		Timeout:     100 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
	})

	ctx := context.Background()
	testErr := errors.New("test error")

	// Trip the circuit breaker
	err := cb.Execute(ctx, func() error {
		return testErr
	})
	assert.Equal(t, testErr, err)
	assert.Equal(t, StateOpen, cb.State())

	// Wait for timeout to transition to half-open
	time.Sleep(150 * time.Millisecond)

	// First request in half-open should succeed
	err = cb.Execute(ctx, func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.State())

	// Second successful request should close the circuit
	err = cb.Execute(ctx, func() error {
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.State())
}

func TestCircuitBreaker_Execute_HalfOpenFailure(t *testing.T) {
	cb := New("test", Config{
		MaxRequests: 2,
		Interval:    10 * time.Second,
		Timeout:     100 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
	})

	ctx := context.Background()
	testErr := errors.New("test error")

	// Trip the circuit breaker
	err := cb.Execute(ctx, func() error {
		return testErr
	})
	assert.Equal(t, testErr, err)
	assert.Equal(t, StateOpen, cb.State())

	// Wait for timeout to transition to half-open
	time.Sleep(150 * time.Millisecond)

	// First request in half-open fails - should reopen
	err = cb.Execute(ctx, func() error {
		return testErr
	})
	assert.Equal(t, testErr, err)
	assert.Equal(t, StateOpen, cb.State())
}

func TestCircuitBreaker_Execute_HalfOpenTooManyRequests(t *testing.T) {
	cb := New("test", Config{
		MaxRequests: 1,
		Interval:    10 * time.Second,
		Timeout:     100 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
	})

	ctx := context.Background()
	testErr := errors.New("test error")

	// Trip the circuit breaker
	err := cb.Execute(ctx, func() error {
		return testErr
	})
	assert.Equal(t, testErr, err)
	assert.Equal(t, StateOpen, cb.State())

	// Wait for timeout to transition to half-open
	time.Sleep(150 * time.Millisecond)

	// Verify we're in half-open by checking state before any requests
	assert.Equal(t, StateHalfOpen, cb.State())

	// First request in half-open - starts but doesn't complete yet
	done := make(chan bool)
	go func() {
		_ = cb.Execute(ctx, func() error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})
		done <- true
	}()

	// Give first request time to start
	time.Sleep(10 * time.Millisecond)

	// Second request should be rejected while first is still running
	err = cb.Execute(ctx, func() error {
		return nil
	})
	assert.Equal(t, ErrTooManyRequests, err)

	// Wait for first request to complete
	<-done
}

func TestCircuitBreaker_Execute_ContextCanceled(t *testing.T) {
	cb := New("test", Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := cb.Execute(ctx, func() error {
		return nil
	})

	assert.Equal(t, context.Canceled, err)
}

func TestCircuitBreaker_Call(t *testing.T) {
	cb := New("test", Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
	})

	called := false
	err := cb.Call(func() error {
		called = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestCircuitBreaker_OnStateChange(t *testing.T) {
	var stateChanges []struct {
		name string
		from State
		to   State
	}

	cb := New("test", Config{
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     100 * time.Millisecond,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 1
		},
		OnStateChange: func(name string, from State, to State) {
			stateChanges = append(stateChanges, struct {
				name string
				from State
				to   State
			}{name, from, to})
		},
	})

	ctx := context.Background()
	testErr := errors.New("test error")

	// Trip the circuit breaker
	err := cb.Execute(ctx, func() error {
		return testErr
	})
	require.Equal(t, testErr, err)

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Trigger half-open
	_ = cb.Execute(ctx, func() error {
		return nil
	})

	// Should have state changes
	assert.GreaterOrEqual(t, len(stateChanges), 1)
	assert.Equal(t, "test", stateChanges[0].name)
	assert.Equal(t, StateClosed, stateChanges[0].from)
	assert.Equal(t, StateOpen, stateChanges[0].to)
}

func TestState_String(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "Closed"},
		{StateOpen, "Open"},
		{StateHalfOpen, "HalfOpen"},
		{State(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestCircuitBreaker_IntervalReset(t *testing.T) {
	cb := New("test", Config{
		MaxRequests: 3,
		Interval:    100 * time.Millisecond,
		Timeout:     5 * time.Second,
	})

	ctx := context.Background()

	// Make a successful request
	err := cb.Execute(ctx, func() error {
		return nil
	})
	assert.NoError(t, err)

	counts := cb.Counts()
	assert.Equal(t, uint32(1), counts.Requests)

	// Wait for interval to expire
	time.Sleep(150 * time.Millisecond)

	// Make another request - counts should be reset
	err = cb.Execute(ctx, func() error {
		return nil
	})
	assert.NoError(t, err)

	counts = cb.Counts()
	assert.Equal(t, uint32(1), counts.Requests)
}

func TestCircuitBreaker_Concurrent(t *testing.T) {
	cb := New("test", Config{
		MaxRequests: 10,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
	})

	ctx := context.Background()
	done := make(chan bool, 100)

	// Execute 100 concurrent requests
	for i := 0; i < 100; i++ {
		go func() {
			_ = cb.Execute(ctx, func() error {
				time.Sleep(1 * time.Millisecond)
				return nil
			})
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 100; i++ {
		<-done
	}

	counts := cb.Counts()
	assert.Equal(t, uint32(100), counts.Requests)
	assert.Equal(t, uint32(100), counts.TotalSuccesses)
}
