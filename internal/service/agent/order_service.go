package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/kairos-development/kairos-agent/internal/domain/events"
	"github.com/kairos-development/kairos-agent/internal/domain/risk"
	"github.com/kairos-development/kairos-agent/internal/domain/storage"
	"github.com/shopspring/decimal"
)

// OrderService orchestrates order lifecycle operations.
// It coordinates between domain entities, risk engine, storage, and exchange connector.
type OrderService interface {
	// CreateOrder validates and creates a new order.
	CreateOrder(ctx context.Context, req CreateOrderRequest) (*entity.Order, error)

	// SubmitOrder submits a pending order to the exchange.
	SubmitOrder(ctx context.Context, orderID string) error

	// CancelOrder cancels an active order.
	CancelOrder(ctx context.Context, orderID string) error

	// GetOrder retrieves an order by ID.
	GetOrder(ctx context.Context, orderID string) (*entity.Order, error)

	// ListOrders retrieves orders for a strategy.
	ListOrders(ctx context.Context, strategyID string, limit, offset int) ([]*entity.Order, error)

	// ReconcileOrders reconciles local order state with exchange state.
	// This is called after crash recovery or connectivity loss.
	ReconcileOrders(ctx context.Context) error
}

// CreateOrderRequest contains parameters for creating a new order.
type CreateOrderRequest struct {
	StrategyID  string
	Symbol      string
	Side        entity.OrderSide
	Type        entity.OrderType
	Quantity    decimal.Decimal
	Price       decimal.Decimal
	TimeInForce entity.TimeInForce
}

type orderService struct {
	orderRepo    storage.OrderRepository
	positionRepo storage.PositionRepository
	strategyRepo storage.StrategyRepository
	balanceRepo  storage.BalanceRepository
	symbolRepo   storage.SymbolRepository
	riskEngine   *risk.Engine
	connector    *ConnectorAdapter
	publisher    *events.Publisher
}

// NewOrderService creates a new order service.
func NewOrderService(
	orderRepo storage.OrderRepository,
	positionRepo storage.PositionRepository,
	strategyRepo storage.StrategyRepository,
	balanceRepo storage.BalanceRepository,
	symbolRepo storage.SymbolRepository,
	riskEngine *risk.Engine,
	conn *ConnectorAdapter,
	publisher *events.Publisher,
) OrderService {
	return &orderService{
		orderRepo:    orderRepo,
		positionRepo: positionRepo,
		strategyRepo: strategyRepo,
		balanceRepo:  balanceRepo,
		symbolRepo:   symbolRepo,
		riskEngine:   riskEngine,
		connector:    conn,
		publisher:    publisher,
	}
}

// CreateOrder validates and creates a new order.
func (s *orderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*entity.Order, error) {
	// Load strategy
	strategy, err := s.strategyRepo.GetByID(ctx, req.StrategyID)
	if err != nil {
		return nil, fmt.Errorf("get strategy: %w", err)
	}

	if !strategy.CanTrade() {
		return nil, fmt.Errorf("strategy cannot trade in current state: %s", strategy.Status)
	}

	// Load symbol metadata
	symbol, err := s.symbolRepo.GetByName(ctx, req.Symbol)
	if err != nil {
		return nil, fmt.Errorf("get symbol: %w", err)
	}

	// Load current position
	position, err := s.positionRepo.GetByStrategyAndSymbol(ctx, req.StrategyID, req.Symbol)
	if err != nil && err != storage.ErrNotFound {
		return nil, fmt.Errorf("get position: %w", err)
	}

	// Load balance
	balances, err := s.balanceRepo.GetLatest(ctx)
	if err != nil {
		return nil, fmt.Errorf("get balance: %w", err)
	}

	var balance *entity.Balance
	if balances != nil {
		balance = balances.GetBalance(symbol.QuoteCurrency)
	}

	// Create order entity
	order := &entity.Order{
		ID:            generateOrderID(),
		ClientOrderID: generateClientOrderID(),
		StrategyID:    req.StrategyID,
		Symbol:        req.Symbol,
		Side:          req.Side,
		Type:          req.Type,
		Status:        entity.OrderStatusPending,
		TimeInForce:   req.TimeInForce,
		Quantity:      req.Quantity,
		Price:         req.Price,
		FilledQty:     decimal.Zero,
		RemainingQty:  req.Quantity,
		AvgFillPrice:  decimal.Zero,
		CreatedAtUTC:  time.Now().UTC(),
		UpdatedAtUTC:  time.Now().UTC(),
	}

	// Risk validation
	if err := s.riskEngine.ValidateOrder(ctx, order, strategy, symbol, balance, position); err != nil {
		return nil, fmt.Errorf("risk validation failed: %w", err)
	}

	// Persist order
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// Publish event
	s.publishOrderCreated(ctx, order)

	return order, nil
}

// SubmitOrder submits a pending order to the exchange.
func (s *orderService) SubmitOrder(ctx context.Context, orderID string) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	if !order.IsPending() {
		return fmt.Errorf("order is not pending: %s", order.Status)
	}

	// Submit to exchange
	exchangeOrderID, err := s.connector.SubmitOrder(ctx, order)
	if err != nil {
		order.Status = entity.OrderStatusRejected
		order.UpdatedAtUTC = time.Now().UTC()
		_ = s.orderRepo.Update(ctx, order)
		s.publishOrderRejected(ctx, order, err.Error())
		return fmt.Errorf("submit order to exchange: %w", err)
	}

	// Update order state
	now := time.Now().UTC()
	order.ExchangeOrderID = exchangeOrderID
	order.Status = entity.OrderStatusSubmitted
	order.SubmittedAtUTC = &now
	order.UpdatedAtUTC = now

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	// Publish event
	s.publishOrderSubmitted(ctx, order)

	return nil
}

// CancelOrder cancels an active order.
func (s *orderService) CancelOrder(ctx context.Context, orderID string) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	if !order.IsActive() {
		return fmt.Errorf("order is not active: %s", order.Status)
	}

	// Cancel on exchange
	if err := s.connector.CancelOrder(ctx, order.ExchangeOrderID); err != nil {
		return fmt.Errorf("cancel order on exchange: %w", err)
	}

	// Update order state
	order.Status = entity.OrderStatusCanceled
	order.UpdatedAtUTC = time.Now().UTC()

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	// Publish event
	s.publishOrderCanceled(ctx, order, "user_requested")

	return nil
}

// GetOrder retrieves an order by ID.
func (s *orderService) GetOrder(ctx context.Context, orderID string) (*entity.Order, error) {
	return s.orderRepo.GetByID(ctx, orderID)
}

// ListOrders retrieves orders for a strategy.
func (s *orderService) ListOrders(ctx context.Context, strategyID string, limit, offset int) ([]*entity.Order, error) {
	return s.orderRepo.ListByStrategy(ctx, strategyID, limit, offset)
}

// ReconcileOrders reconciles local order state with exchange state.
func (s *orderService) ReconcileOrders(ctx context.Context) error {
	// Get all active orders from local storage
	localOrders, err := s.orderRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("list active orders: %w", err)
	}

	// Query each order from exchange
	for _, order := range localOrders {
		if order.ExchangeOrderID == "" {
			continue
		}

		exchangeOrder, err := s.connector.QueryOrder(ctx, order.ExchangeOrderID)
		if err != nil {
			// Log error but continue reconciliation
			continue
		}

		// Update local state to match exchange
		order.Status = exchangeOrder.Status
		order.FilledQty = exchangeOrder.FilledQty
		order.RemainingQty = exchangeOrder.RemainingQty
		order.AvgFillPrice = exchangeOrder.AvgFillPrice
		order.UpdatedAtUTC = time.Now().UTC()

		if err := s.orderRepo.Update(ctx, order); err != nil {
			return fmt.Errorf("update order during reconciliation: %w", err)
		}
	}

	return nil
}

func (s *orderService) publishOrderCreated(ctx context.Context, order *entity.Order) {
	if s.publisher == nil {
		return
	}

	event := &events.OrderCreatedEvent{
		BaseEvent: events.BaseEvent{
			EventType:  events.EventTypeOrderCreated,
			OccurredAt: time.Now().UTC(),
		},
		OrderID:       order.ID,
		ClientOrderID: order.ClientOrderID,
		StrategyID:    order.StrategyID,
		Symbol:        order.Symbol,
		Side:          string(order.Side),
		OrderType:     string(order.Type),
		Quantity:      order.Quantity,
		Price:         order.Price,
	}

	s.publisher.PublishAsync(ctx, event)
}

func (s *orderService) publishOrderSubmitted(ctx context.Context, order *entity.Order) {
	if s.publisher == nil {
		return
	}

	event := &events.OrderSubmittedEvent{
		BaseEvent: events.BaseEvent{
			EventType:  events.EventTypeOrderSubmitted,
			OccurredAt: time.Now().UTC(),
		},
		OrderID:         order.ID,
		ClientOrderID:   order.ClientOrderID,
		ExchangeOrderID: order.ExchangeOrderID,
		SubmittedAtUTC:  *order.SubmittedAtUTC,
	}

	s.publisher.PublishAsync(ctx, event)
}

func (s *orderService) publishOrderCanceled(ctx context.Context, order *entity.Order, reason string) {
	if s.publisher == nil {
		return
	}

	event := &events.OrderCanceledEvent{
		BaseEvent: events.BaseEvent{
			EventType:  events.EventTypeOrderCanceled,
			OccurredAt: time.Now().UTC(),
		},
		OrderID:       order.ID,
		CanceledAtUTC: time.Now().UTC(),
		Reason:        reason,
	}

	s.publisher.PublishAsync(ctx, event)
}

func (s *orderService) publishOrderRejected(ctx context.Context, order *entity.Order, reason string) {
	if s.publisher == nil {
		return
	}

	event := &events.OrderRejectedEvent{
		BaseEvent: events.BaseEvent{
			EventType:  events.EventTypeOrderRejected,
			OccurredAt: time.Now().UTC(),
		},
		OrderID:       order.ID,
		RejectedAtUTC: time.Now().UTC(),
		Reason:        reason,
	}

	s.publisher.PublishAsync(ctx, event)
}

func generateOrderID() string {
	return fmt.Sprintf("ord_%d", time.Now().UnixNano())
}

func generateClientOrderID() string {
	return fmt.Sprintf("cli_%d", time.Now().UnixNano())
}
