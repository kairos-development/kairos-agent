package engine

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// State represents the global state of the trading agent.
type State int

const (
	// StateIdle indicates the agent is running but not actively trading.
	StateIdle State = iota
	// StateScanning indicates the agent is scanning markets for opportunities.
	StateScanning
	// StateLiveTrading indicates the agent is executing live trades.
	StateLiveTrading
	// StatePaperTrading indicates the agent is simulating trades.
	StatePaperTrading
	// StateBacktesting indicates the agent is running historical backtests.
	StateBacktesting
	// StateOptimizing indicates the agent is optimizing strategy parameters.
	StateOptimizing
	// StateHalted indicates the agent has been emergency stopped.
	StateHalted
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateScanning:
		return "Scanning"
	case StateLiveTrading:
		return "Live Trading"
	case StatePaperTrading:
		return "Paper Trading"
	case StateBacktesting:
		return "Backtesting"
	case StateOptimizing:
		return "Optimizing"
	case StateHalted:
		return "Halted"
	default:
		return "Unknown"
	}
}

// StateMachine manages the global state of the trading agent.
// It ensures thread-safe state transitions and broadcasts state changes.
type StateMachine struct {
	mu            sync.RWMutex
	currentState  State
	previousState State
	transitionAt  time.Time
	listeners     map[uint64]chan<- StateTransition
	nextID        uint64
	logger        *logrus.Logger
}

// StateTransition represents a state change event.
type StateTransition struct {
	From      State
	To        State
	Timestamp time.Time
	Reason    string
}

// ListenerID uniquely identifies a state transition listener.
type ListenerID uint64

// NewStateMachine creates a new state machine starting in Idle state.
func NewStateMachine(logger *logrus.Logger) *StateMachine {
	if logger == nil {
		logger = logrus.New()
	}

	return &StateMachine{
		currentState:  StateIdle,
		previousState: StateIdle,
		transitionAt:  time.Now().UTC(),
		listeners:     make(map[uint64]chan<- StateTransition),
		nextID:        1,
		logger:        logger,
	}
}

// Current returns the current state.
func (sm *StateMachine) Current() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

// Previous returns the previous state.
func (sm *StateMachine) Previous() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.previousState
}

// TransitionedAt returns when the last transition occurred.
func (sm *StateMachine) TransitionedAt() time.Time {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.transitionAt
}

// Transition attempts to transition to a new state.
// Returns an error if the transition is not allowed.
func (sm *StateMachine) Transition(to State, reason string) error {
	sm.mu.Lock()

	from := sm.currentState

	// Validate transition
	if !sm.isValidTransition(from, to) {
		sm.mu.Unlock()
		return &InvalidTransitionError{From: from, To: to}
	}

	// Perform transition
	sm.previousState = from
	sm.currentState = to
	sm.transitionAt = time.Now().UTC()

	// Create transition event
	transition := StateTransition{
		From:      from,
		To:        to,
		Timestamp: sm.transitionAt,
		Reason:    reason,
	}

	// Copy listeners to avoid holding lock during notification
	listenersCopy := make([]chan<- StateTransition, 0, len(sm.listeners))
	for _, listener := range sm.listeners {
		listenersCopy = append(listenersCopy, listener)
	}

	sm.mu.Unlock()

	// Notify listeners outside the lock
	for _, listener := range listenersCopy {
		select {
		case listener <- transition:
		default:
			// Non-blocking send - if listener is slow, skip
			sm.logger.Warn("State transition listener channel full, dropping notification")
		}
	}

	sm.logger.WithFields(logrus.Fields{
		"from":   from.String(),
		"to":     to.String(),
		"reason": reason,
	}).Info("State transition")

	return nil
}

// Subscribe registers a listener for state transitions.
// The channel should be buffered to avoid blocking the state machine.
// Returns a ListenerID that can be used to unsubscribe.
func (sm *StateMachine) Subscribe(ch chan<- StateTransition) ListenerID {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := sm.nextID
	sm.nextID++

	sm.listeners[id] = ch

	sm.logger.WithField("listener_id", id).Debug("State transition listener subscribed")

	return ListenerID(id)
}

// Unsubscribe removes a listener for state transitions.
func (sm *StateMachine) Unsubscribe(id ListenerID) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.listeners, uint64(id))

	sm.logger.WithField("listener_id", id).Debug("State transition listener unsubscribed")
}

// isValidTransition checks if a state transition is allowed.
func (sm *StateMachine) isValidTransition(from, to State) bool {
	// Cannot transition to same state
	if from == to {
		return false
	}

	// Halted can only transition to Idle
	if from == StateHalted && to != StateIdle {
		return false
	}

	// Can always halt from any state
	if to == StateHalted {
		return true
	}

	// Can always go to Idle
	if to == StateIdle {
		return true
	}

	// All other transitions are valid
	return true
}

// InvalidTransitionError is returned when an invalid state transition is attempted.
type InvalidTransitionError struct {
	From State
	To   State
}

func (e *InvalidTransitionError) Error() string {
	return "invalid state transition from " + e.From.String() + " to " + e.To.String()
}
