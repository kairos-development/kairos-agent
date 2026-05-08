package reconciliation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/domain/events"
	"github.com/kairos-development/kairos-agent/internal/domain/storage"
	"github.com/sirupsen/logrus"
)

// ExchangeConnector defines the interface for querying exchange state.
type ExchangeConnector interface {
	GetOpenOrders(ctx context.Context) ([]*entity.Order, error)
	GetPositions(ctx context.Context) ([]*entity.Position, error)
	QueryOrderStatus(ctx context.Context, exchangeOrderID string) (*entity.Order, error)
}

// Reconciler reconciles local state with exchange state.
// This is critical after crashes, reconnects, or WS gaps.
type Reconciler struct {
	mu sync.Mutex

	connector    ExchangeConnector
	orderRepo    storage.OrderRepository
	positionRepo storage.PositionRepository
	publisher    *events.Publisher
	logger       *logrus.Logger

	lastReconcileAt time.Time
}

// NewReconciler creates a new reconciler.
func NewReconciler(
	connector ExchangeConnector,
	orderRepo storage.OrderRepository,
	positionRepo storage.PositionRepository,
	publisher *events.Publisher,
	logger *logrus.Logger,
) *Reconciler {
	if logger == nil {
		logger = logrus.New()
	}

	return &Reconciler{
		connector:    connector,
		orderRepo:    orderRepo,
		positionRepo: positionRepo,
		publisher:    publisher,
		logger:       logger,
	}
}

// ReconcileOrders reconciles local order state with exchange state.
// Returns the number of orders updated and any zombie orders detected.
func (r *Reconciler) ReconcileOrders(ctx context.Context) (*ReconcileResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	startTime := time.Now()
	r.logger.Info("Starting order reconciliation")

	result := &ReconcileResult{
		StartedAt: startTime,
	}

	// Get all active local orders
	localOrders, err := r.orderRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active local orders: %w", err)
	}

	// Get all open orders from exchange
	exchangeOrders, err := r.connector.GetOpenOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("get open orders from exchange: %w", err)
	}

	// Build exchange order map by ExchangeOrderID
	exchangeOrderMap := make(map[string]*entity.Order)
	for _, order := range exchangeOrders {
		exchangeOrderMap[order.ExchangeOrderID] = order
	}

	// Reconcile each local order
	for _, localOrder := range localOrders {
		if localOrder.ExchangeOrderID == "" {
			// Order not yet submitted to exchange, skip
			continue
		}

		exchangeOrder, existsOnExchange := exchangeOrderMap[localOrder.ExchangeOrderID]

		if !existsOnExchange {
			// Zombie order: exists locally but not on exchange
			r.logger.WithFields(logrus.Fields{
				"order_id":          localOrder.ID,
				"exchange_order_id": localOrder.ExchangeOrderID,
			}).Warn("Zombie order detected: exists locally but not on exchange")

			// Mark as canceled locally
			localOrder.Status = entity.OrderStatusCanceled
			localOrder.UpdatedAtUTC = time.Now().UTC()

			if err := r.orderRepo.Update(ctx, localOrder); err != nil {
				r.logger.WithError(err).Error("Failed to update zombie order")
				continue
			}

			result.ZombieOrders = append(result.ZombieOrders, localOrder.ID)
			result.OrdersUpdated++
			continue
		}

		// Check if state differs
		if r.orderStatesDiffer(localOrder, exchangeOrder) {
			r.logger.WithFields(logrus.Fields{
				"order_id":            localOrder.ID,
				"local_status":        localOrder.Status,
				"exchange_status":     exchangeOrder.Status,
				"local_filled_qty":    localOrder.FilledQty.String(),
				"exchange_filled_qty": exchangeOrder.FilledQty.String(),
			}).Info("Order state mismatch, updating local state")

			// Update local state to match exchange
			localOrder.Status = exchangeOrder.Status
			localOrder.FilledQty = exchangeOrder.FilledQty
			localOrder.RemainingQty = exchangeOrder.RemainingQty
			localOrder.AvgFillPrice = exchangeOrder.AvgFillPrice
			localOrder.UpdatedAtUTC = time.Now().UTC()

			if exchangeOrder.FilledAtUTC != nil {
				localOrder.FilledAtUTC = exchangeOrder.FilledAtUTC
			}

			if err := r.orderRepo.Update(ctx, localOrder); err != nil {
				r.logger.WithError(err).Error("Failed to update order during reconciliation")
				continue
			}

			result.OrdersUpdated++
		}
	}

	// Detect partial fills that need to be restored
	for _, localOrder := range localOrders {
		if localOrder.IsPartiallyFilled() {
			result.PartialFills = append(result.PartialFills, localOrder.ID)
		}
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	r.lastReconcileAt = result.CompletedAt

	r.logger.WithFields(logrus.Fields{
		"orders_updated": result.OrdersUpdated,
		"zombie_orders":  len(result.ZombieOrders),
		"partial_fills":  len(result.PartialFills),
		"duration_ms":    result.Duration.Milliseconds(),
	}).Info("Order reconciliation completed")

	return result, nil
}

// ReconcilePositions reconciles local position state with exchange state.
func (r *Reconciler) ReconcilePositions(ctx context.Context) (*ReconcileResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	startTime := time.Now()
	r.logger.Info("Starting position reconciliation")

	result := &ReconcileResult{
		StartedAt: startTime,
	}

	// Get all local positions
	localPositions, err := r.positionRepo.ListOpen(ctx)
	if err != nil {
		return nil, fmt.Errorf("list local positions: %w", err)
	}

	// Get all positions from exchange
	exchangePositions, err := r.connector.GetPositions(ctx)
	if err != nil {
		return nil, fmt.Errorf("get positions from exchange: %w", err)
	}

	// Build exchange position map by symbol
	exchangePositionMap := make(map[string]*entity.Position)
	for _, pos := range exchangePositions {
		exchangePositionMap[pos.Symbol] = pos
	}

	// Reconcile each local position
	for _, localPos := range localPositions {
		exchangePos, existsOnExchange := exchangePositionMap[localPos.Symbol]

		if !existsOnExchange || exchangePos.IsFlat() {
			// Position closed on exchange but still open locally
			if !localPos.IsFlat() {
				r.logger.WithFields(logrus.Fields{
					"position_id": localPos.ID,
					"symbol":      localPos.Symbol,
				}).Warn("Position closed on exchange but open locally")

				// Mark as closed locally
				localPos.Quantity = exchangePos.Quantity
				localPos.UpdatedAtUTC = time.Now().UTC()

				if err := r.positionRepo.Update(ctx, localPos); err != nil {
					r.logger.WithError(err).Error("Failed to update position")
					continue
				}

				result.PositionsUpdated++
			}
			continue
		}

		// Check if state differs
		if r.positionStatesDiffer(localPos, exchangePos) {
			r.logger.WithFields(logrus.Fields{
				"position_id":       localPos.ID,
				"symbol":            localPos.Symbol,
				"local_quantity":    localPos.Quantity.String(),
				"exchange_quantity": exchangePos.Quantity.String(),
			}).Info("Position state mismatch, updating local state")

			// Update local state to match exchange
			localPos.Quantity = exchangePos.Quantity
			localPos.CurrentPrice = exchangePos.CurrentPrice
			localPos.UnrealizedPnL = exchangePos.UnrealizedPnL
			localPos.UpdatedAtUTC = time.Now().UTC()

			if err := r.positionRepo.Update(ctx, localPos); err != nil {
				r.logger.WithError(err).Error("Failed to update position during reconciliation")
				continue
			}

			result.PositionsUpdated++
		}
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	r.logger.WithFields(logrus.Fields{
		"positions_updated": result.PositionsUpdated,
		"duration_ms":       result.Duration.Milliseconds(),
	}).Info("Position reconciliation completed")

	return result, nil
}

// ReconcileAll performs full reconciliation of orders and positions.
func (r *Reconciler) ReconcileAll(ctx context.Context) error {
	orderResult, err := r.ReconcileOrders(ctx)
	if err != nil {
		return fmt.Errorf("reconcile orders: %w", err)
	}

	positionResult, err := r.ReconcilePositions(ctx)
	if err != nil {
		return fmt.Errorf("reconcile positions: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"orders_updated":    orderResult.OrdersUpdated,
		"positions_updated": positionResult.PositionsUpdated,
		"zombie_orders":     len(orderResult.ZombieOrders),
		"partial_fills":     len(orderResult.PartialFills),
	}).Info("Full reconciliation completed")

	return nil
}

// LastReconcileAt returns the timestamp of the last reconciliation.
func (r *Reconciler) LastReconcileAt() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastReconcileAt
}

func (r *Reconciler) orderStatesDiffer(local, exchange *entity.Order) bool {
	return local.Status != exchange.Status ||
		!local.FilledQty.Equal(exchange.FilledQty) ||
		!local.RemainingQty.Equal(exchange.RemainingQty) ||
		!local.AvgFillPrice.Equal(exchange.AvgFillPrice)
}

func (r *Reconciler) positionStatesDiffer(local, exchange *entity.Position) bool {
	return !local.Quantity.Equal(exchange.Quantity) ||
		!local.CurrentPrice.Equal(exchange.CurrentPrice)
}

// ReconcileResult contains the results of a reconciliation operation.
type ReconcileResult struct {
	StartedAt        time.Time
	CompletedAt      time.Time
	Duration         time.Duration
	OrdersUpdated    int
	PositionsUpdated int
	ZombieOrders     []string
	PartialFills     []string
}
