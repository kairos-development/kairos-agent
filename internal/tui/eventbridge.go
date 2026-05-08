package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kairos-development/kairos-agent/internal/domain/events"
	"github.com/kairos-development/kairos-agent/internal/tui/components"
)

// EventBridge converts domain events into TUI messages.
// This provides a clean separation between domain events and UI updates.
type EventBridge struct {
	program *tea.Program
}

// NewEventBridge creates a new event bridge.
func NewEventBridge(program *tea.Program) *EventBridge {
	return &EventBridge{
		program: program,
	}
}

// OnEvent handles domain events and converts them to TUI messages.
func (eb *EventBridge) OnEvent(ctx context.Context, event events.Event) {
	if eb.program == nil {
		return
	}

	// Convert domain event to TUI message
	msg := eb.convertEvent(event)
	if msg != nil {
		eb.program.Send(msg)
	}
}

// convertEvent converts a domain event to a TUI message.
func (eb *EventBridge) convertEvent(event events.Event) tea.Msg {
	switch e := event.(type) {
	case *events.OrderCreatedEvent:
		return orderEventMsg{
			level:   components.LogLevelInfo,
			message: fmt.Sprintf("Order created: %s %s %s @ %s", e.Side, e.Quantity.String(), e.Symbol, e.Price.String()),
		}

	case *events.OrderSubmittedEvent:
		return orderEventMsg{
			level:   components.LogLevelSuccess,
			message: fmt.Sprintf("Order submitted: %s", e.ExchangeOrderID),
		}

	case *events.OrderFilledEvent:
		return orderEventMsg{
			level:   components.LogLevelSuccess,
			message: fmt.Sprintf("Order filled: %s @ %s", e.FilledQty.String(), e.AvgFillPrice.String()),
		}

	case *events.OrderPartialEvent:
		return orderEventMsg{
			level:   components.LogLevelInfo,
			message: fmt.Sprintf("Order partial fill: %s filled, %s remaining", e.FilledQty.String(), e.RemainingQty.String()),
		}

	case *events.OrderCanceledEvent:
		return orderEventMsg{
			level:   components.LogLevelWarning,
			message: fmt.Sprintf("Order canceled: %s", e.Reason),
		}

	case *events.OrderRejectedEvent:
		return orderEventMsg{
			level:   components.LogLevelError,
			message: fmt.Sprintf("Order rejected: %s", e.Reason),
		}

	case *events.PositionOpenedEvent:
		return positionEventMsg{
			level:   components.LogLevelSuccess,
			message: fmt.Sprintf("Position opened: %s %s %s @ %s", e.Side, e.Quantity.String(), e.Symbol, e.EntryPrice.String()),
		}

	case *events.PositionUpdatedEvent:
		return positionEventMsg{
			level:   components.LogLevelInfo,
			message: fmt.Sprintf("Position updated: PnL %s", e.UnrealizedPnL.String()),
		}

	case *events.PositionClosedEvent:
		return positionEventMsg{
			level:   components.LogLevelSuccess,
			message: fmt.Sprintf("Position closed: %s PnL %s", e.Symbol, e.RealizedPnL.String()),
		}

	case *events.BalanceUpdatedEvent:
		return balanceEventMsg{
			level:   components.LogLevelInfo,
			message: fmt.Sprintf("Balance updated: %s = %s", e.Asset, e.Total.String()),
		}

	case *events.RiskViolationEvent:
		return riskEventMsg{
			level:   components.LogLevelError,
			message: fmt.Sprintf("Risk violation: %s - %s (current: %s, limit: %s)", e.ViolationType, e.Action, e.CurrentValue.String(), e.LimitValue.String()),
		}

	case *events.RiskWarningEvent:
		return riskEventMsg{
			level:   components.LogLevelWarning,
			message: fmt.Sprintf("Risk warning: %s at %s%% of limit", e.WarningType, e.ThresholdPct.String()),
		}

	case *events.StrategyStartedEvent:
		return strategyEventMsg{
			level:   components.LogLevelSuccess,
			message: fmt.Sprintf("Strategy started: %s (%s)", e.StrategyName, e.StrategyType),
		}

	case *events.StrategyPausedEvent:
		return strategyEventMsg{
			level:   components.LogLevelWarning,
			message: fmt.Sprintf("Strategy paused: %s", e.Reason),
		}

	case *events.StrategyStoppedEvent:
		return strategyEventMsg{
			level:   components.LogLevelInfo,
			message: fmt.Sprintf("Strategy stopped: %s (Final PnL: %s)", e.Reason, e.FinalPnL.String()),
		}

	default:
		// Unknown event type - ignore
		return nil
	}
}

// TUI event messages

type orderEventMsg struct {
	level   components.LogLevel
	message string
}

type positionEventMsg struct {
	level   components.LogLevel
	message string
}

type balanceEventMsg struct {
	level   components.LogLevel
	message string
}

type riskEventMsg struct {
	level   components.LogLevel
	message string
}

type strategyEventMsg struct {
	level   components.LogLevel
	message string
}
