package runtime

import (
	"fmt"
	"sync"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/sirupsen/logrus"
)

// StateTransition represents a state change event.
type StateTransition struct {
	From      entity.RunMode
	To        entity.RunMode
	Timestamp time.Time
	Reason    string
}

// StateMachine manages agent state transitions per instruction 8.3.
type StateMachine struct {
	mu           sync.RWMutex
	current      entity.RunMode
	previous     entity.RunMode
	transitionAt time.Time
	listeners    map[uint64]chan<- StateTransition
	nextID       uint64
	logger       *logrus.Logger
}

// NewStateMachine creates a state machine starting in Idle.
func NewStateMachine(logger *logrus.Logger) *StateMachine {
	if logger == nil {
		logger = logrus.New()
	}
	return &StateMachine{
		current:      entity.RunModeIdle,
		previous:     entity.RunModeIdle,
		transitionAt: time.Now().UTC(),
		listeners:    make(map[uint64]chan<- StateTransition),
		nextID:       1,
		logger:       logger,
	}
}

// Current returns the current state.
func (sm *StateMachine) Current() entity.RunMode {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current
}

// Transition attempts a state change per instruction 8.3 rules.
func (sm *StateMachine) Transition(to entity.RunMode, reason string) error {
	sm.mu.Lock()
	from := sm.current

	if from == to {
		sm.mu.Unlock()
		return nil
	}

	if from == entity.RunModeHalted && to != entity.RunModeIdle {
		sm.mu.Unlock()
		return &InvalidTransitionError{From: from, To: to}
	}

	if to == entity.RunModeHalted {
		sm.previous = from
		sm.current = to
		sm.transitionAt = time.Now().UTC()
		sm.mu.Unlock()
		sm.notify(StateTransition{From: from, To: to, Timestamp: sm.transitionAt, Reason: reason})
		return nil
	}

	sm.previous = from
	sm.current = to
	sm.transitionAt = time.Now().UTC()
	sm.mu.Unlock()

	sm.notify(StateTransition{From: from, To: to, Timestamp: sm.transitionAt, Reason: reason})
	return nil
}

// Subscribe registers a listener for state transitions.
func (sm *StateMachine) Subscribe(ch chan<- StateTransition) uint64 {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	id := sm.nextID
	sm.nextID++
	sm.listeners[id] = ch
	return id
}

// Unsubscribe removes a listener.
func (sm *StateMachine) Unsubscribe(id uint64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.listeners, id)
}

func (sm *StateMachine) notify(t StateTransition) {
	sm.mu.RLock()
	listeners := make([]chan<- StateTransition, 0, len(sm.listeners))
	for _, ch := range sm.listeners {
		listeners = append(listeners, ch)
	}
	sm.mu.RUnlock()

	for _, ch := range listeners {
		select {
		case ch <- t:
		default:
		}
	}

	sm.logger.WithFields(logrus.Fields{
		"from":   t.From,
		"to":     t.To,
		"reason": t.Reason,
	}).Info("State transition")
}

// InvalidTransitionError is returned for illegal transitions.
type InvalidTransitionError struct {
	From entity.RunMode
	To   entity.RunMode
}

func (e *InvalidTransitionError) Error() string {
	return fmt.Sprintf("invalid transition from %s to %s", e.From, e.To)
}
