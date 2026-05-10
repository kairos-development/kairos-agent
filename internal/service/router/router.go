package router

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/domain/events"
	"github.com/sirupsen/logrus"
)

// ExchangeConnector defines the interface for live exchange operations.
type ExchangeConnector interface {
	SubmitOrder(ctx context.Context, order *entity.Order) (string, error)
	CancelOrder(ctx context.Context, exchangeOrderID string) error
	QueryOrderStatus(ctx context.Context, exchangeOrderID string) (*entity.Order, error)
}

// PaperSimulator defines the interface for paper trading simulation.
type PaperSimulator interface {
	SubmitOrder(ctx context.Context, order *entity.Order) (string, error)
	CancelOrder(ctx context.Context, orderID string) error
	GetOrderStatus(ctx context.Context, orderID string) (*entity.Order, error)
}

// TradingGate validates whether new order entries may be submitted.
type TradingGate interface {
	// CheckNewEntry returns nil only when a new trade entry is allowed.
	CheckNewEntry(ctx context.Context) error
}

// OrderRouter routes orders to either live exchange or paper simulator.
// It maintains an in-flight journal for idempotency and timeout handling.
type OrderRouter struct {
	mu sync.RWMutex

	config    *entity.RouterConfig
	connector ExchangeConnector
	simulator PaperSimulator
	publisher *events.Publisher
	logger    *logrus.Logger
	gate      TradingGate

	// In-flight order tracking
	inFlight map[string]*InFlightOrder
}

// InFlightOrder tracks an order that has been submitted but not yet confirmed.
type InFlightOrder struct {
	Order         *entity.Order
	SubmittedAt   time.Time
	LastQueryAt   *time.Time
	QueryAttempts int
}

// NewOrderRouter creates a new order router.
func NewOrderRouter(
	config *entity.RouterConfig,
	connector ExchangeConnector,
	simulator PaperSimulator,
	publisher *events.Publisher,
	logger *logrus.Logger,
) *OrderRouter {
	if config == nil {
		config = entity.DefaultRouterConfig()
	}

	if logger == nil {
		logger = logrus.New()
	}

	return &OrderRouter{
		config:    config,
		connector: connector,
		simulator: simulator,
		publisher: publisher,
		logger:    logger,
		inFlight:  make(map[string]*InFlightOrder),
	}
}

// SubmitOrder submits an order through the appropriate routing destination.
func (r *OrderRouter) SubmitOrder(ctx context.Context, order *entity.Order) (string, error) {
	if err := r.checkNewEntry(ctx); err != nil {
		return "", err
	}

	r.mu.Lock()

	// Check in-flight limit
	if len(r.inFlight) >= r.config.MaxInFlightOrders {
		r.mu.Unlock()
		return "", fmt.Errorf("max in-flight orders reached: %d", r.config.MaxInFlightOrders)
	}

	// Generate deterministic ClientOrderID if not set
	if order.ClientOrderID == "" {
		order.ClientOrderID = r.generateClientOrderID(order)
	}

	// Check idempotency
	if r.config.EnableIdempotency {
		if existing, exists := r.inFlight[order.ClientOrderID]; exists {
			r.mu.Unlock()
			r.logger.WithFields(logrus.Fields{
				"client_order_id": order.ClientOrderID,
				"order_id":        existing.Order.ID,
			}).Warn("Duplicate order submission detected, returning existing")
			return existing.Order.ExchangeOrderID, nil
		}
	}

	// Add to in-flight tracking
	inFlightOrder := &InFlightOrder{
		Order:       order,
		SubmittedAt: time.Now().UTC(),
	}
	r.inFlight[order.ClientOrderID] = inFlightOrder

	r.mu.Unlock()

	// Route based on mode
	var exchangeOrderID string
	var err error

	switch r.config.Mode {
	case entity.RoutingModeLive:
		exchangeOrderID, err = r.submitLive(ctx, order)
	case entity.RoutingModePaper:
		exchangeOrderID, err = r.submitPaper(ctx, order)
	default:
		err = fmt.Errorf("unknown routing mode: %s", r.config.Mode)
	}

	if err != nil {
		// Remove from in-flight on error
		r.mu.Lock()
		delete(r.inFlight, order.ClientOrderID)
		r.mu.Unlock()
		return "", err
	}

	// Update in-flight with exchange order ID
	r.mu.Lock()
	if inFlightOrder, exists := r.inFlight[order.ClientOrderID]; exists {
		inFlightOrder.Order.ExchangeOrderID = exchangeOrderID
	}
	r.mu.Unlock()

	r.logger.WithFields(logrus.Fields{
		"client_order_id":   order.ClientOrderID,
		"exchange_order_id": exchangeOrderID,
		"mode":              r.config.Mode,
	}).Info("Order submitted")

	return exchangeOrderID, nil
}

func (r *OrderRouter) checkNewEntry(ctx context.Context) error {
	r.mu.RLock()
	gate := r.gate
	r.mu.RUnlock()

	if gate == nil {
		return nil
	}
	if err := gate.CheckNewEntry(ctx); err != nil {
		return fmt.Errorf("trading gate: %w", err)
	}
	return nil
}

// CancelOrder cancels an order through the appropriate routing destination.
func (r *OrderRouter) CancelOrder(ctx context.Context, exchangeOrderID string) error {
	switch r.config.Mode {
	case entity.RoutingModeLive:
		return r.connector.CancelOrder(ctx, exchangeOrderID)
	case entity.RoutingModePaper:
		return r.simulator.CancelOrder(ctx, exchangeOrderID)
	default:
		return fmt.Errorf("unknown routing mode: %s", r.config.Mode)
	}
}

// QueryOrderStatus queries order status with timeout fallback.
func (r *OrderRouter) QueryOrderStatus(ctx context.Context, exchangeOrderID string) (*entity.Order, error) {
	// Create timeout context
	queryCtx, cancel := context.WithTimeout(ctx, time.Duration(r.config.TimeoutSeconds)*time.Second)
	defer cancel()

	var order *entity.Order
	var err error

	switch r.config.Mode {
	case entity.RoutingModeLive:
		order, err = r.connector.QueryOrderStatus(queryCtx, exchangeOrderID)
	case entity.RoutingModePaper:
		order, err = r.simulator.GetOrderStatus(queryCtx, exchangeOrderID)
	default:
		return nil, fmt.Errorf("unknown routing mode: %s", r.config.Mode)
	}

	if err != nil && r.config.EnableStatusFallback {
		r.logger.WithError(err).WithField("exchange_order_id", exchangeOrderID).Warn("Order status query failed, checking in-flight cache")

		// Fallback to in-flight cache
		r.mu.RLock()
		for _, inFlight := range r.inFlight {
			if inFlight.Order.ExchangeOrderID == exchangeOrderID {
				order = inFlight.Order
				err = nil
				break
			}
		}
		r.mu.RUnlock()
	}

	return order, err
}

// ConfirmOrder removes an order from in-flight tracking after terminal state.
func (r *OrderRouter) ConfirmOrder(clientOrderID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.inFlight, clientOrderID)

	r.logger.WithField("client_order_id", clientOrderID).Debug("Order confirmed and removed from in-flight")
}

// CleanupTimedOutOrders removes orders that have exceeded timeout without confirmation.
func (r *OrderRouter) CleanupTimedOutOrders(ctx context.Context) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	timeout := time.Duration(r.config.TimeoutSeconds) * time.Second
	now := time.Now().UTC()
	cleaned := 0

	for clientOrderID, inFlight := range r.inFlight {
		if now.Sub(inFlight.SubmittedAt) > timeout {
			r.logger.WithFields(logrus.Fields{
				"client_order_id": clientOrderID,
				"submitted_at":    inFlight.SubmittedAt,
				"timeout":         timeout,
			}).Warn("Order timed out, removing from in-flight")

			delete(r.inFlight, clientOrderID)
			cleaned++

			// Publish timeout event
			r.publishOrderTimeout(ctx, inFlight.Order)
		}
	}

	return cleaned
}

// GetInFlightCount returns the current number of in-flight orders.
func (r *OrderRouter) GetInFlightCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.inFlight)
}

// GetMode returns the current routing mode.
func (r *OrderRouter) GetMode() entity.RoutingMode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config.Mode
}

// SetMode changes the routing mode.
func (r *OrderRouter) SetMode(mode entity.RoutingMode) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.config.Mode = mode

	r.logger.WithField("mode", mode).Info("Routing mode changed")
}

// SetTradingGate configures the runtime safety gate for new entries.
func (r *OrderRouter) SetTradingGate(gate TradingGate) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.gate = gate
}

func (r *OrderRouter) submitLive(ctx context.Context, order *entity.Order) (string, error) {
	if r.connector == nil {
		return "", fmt.Errorf("live connector not configured")
	}

	return r.connector.SubmitOrder(ctx, order)
}

func (r *OrderRouter) submitPaper(ctx context.Context, order *entity.Order) (string, error) {
	if r.simulator == nil {
		return "", fmt.Errorf("paper simulator not configured")
	}

	return r.simulator.SubmitOrder(ctx, order)
}

// generateClientOrderID creates a deterministic SHA256-based client order ID.
// Format: SHA256(strategy_id + symbol + side + timestamp_bucket + nonce)
func (r *OrderRouter) generateClientOrderID(order *entity.Order) string {
	// Bucket timestamp to 1-second intervals for determinism
	timestampBucket := order.CreatedAtUTC.Truncate(time.Second).Unix()

	// Create deterministic input
	input := fmt.Sprintf("%s:%s:%s:%d:%s",
		order.StrategyID,
		order.Symbol,
		order.Side,
		timestampBucket,
		order.ID, // Use order ID as nonce
	)

	// Compute SHA256
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

func (r *OrderRouter) publishOrderTimeout(ctx context.Context, order *entity.Order) {
	if r.publisher == nil {
		return
	}

	event := &events.OrderTimeoutEvent{
		BaseEvent: events.BaseEvent{
			EventType:  events.EventTypeOrderTimeout,
			OccurredAt: time.Now().UTC(),
		},
		OrderID:       order.ID,
		ClientOrderID: order.ClientOrderID,
		TimeoutAt:     time.Now().UTC(),
	}

	r.publisher.PublishAsync(ctx, event)
}
