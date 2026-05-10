package connector

import (
	"context"
	"errors"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	"github.com/shopspring/decimal"
)

// ErrStreamEventsUnsupported indicates that a connector does not expose stream lifecycle events.
var ErrStreamEventsUnsupported = errors.New("stream events unsupported")

// ErrPositionSnapshotsUnsupported indicates that a connector cannot return full position snapshots.
var ErrPositionSnapshotsUnsupported = errors.New("position snapshots unsupported")

// Connector defines the interface for exchange connectivity.
// Implementations must handle REST execution, WebSocket streams, reconnection,
// and exchange-specific quirks while presenting a unified domain interface.
type Connector interface {
	// Name returns the exchange identifier (e.g., "bybit", "binance").
	Name() string

	// Connect establishes exchange connections and authenticates.
	// Returns an error if authentication fails or permissions are insufficient.
	Connect(ctx context.Context) error

	// Disconnect gracefully closes all connections.
	Disconnect(ctx context.Context) error

	// IsConnected returns true if the connector is currently connected.
	IsConnected() bool

	// SubmitOrder sends an order to the exchange.
	// Returns the exchange order ID or an error.
	SubmitOrder(ctx context.Context, order *entity.Order) (exchangeOrderID string, err error)

	// CancelOrder cancels an active order on the exchange.
	CancelOrder(ctx context.Context, orderID string) error

	// GetOpenOrders retrieves all open orders for the account.
	// Returns a slice of orders that are currently active (not filled, canceled, or rejected).
	GetOpenOrders(ctx context.Context) ([]*entity.Order, error)

	// QueryOrder retrieves the current status of an order from the exchange.
	QueryOrder(ctx context.Context, orderID string) (*entity.Order, error)

	// GetPosition retrieves the current position for a symbol.
	GetPosition(ctx context.Context, symbol string) (*entity.Position, error)

	// GetBalance retrieves the current account balance.
	GetBalance(ctx context.Context) (*entity.AccountBalance, error)

	// GetSymbol retrieves symbol metadata and trading constraints.
	GetSymbol(ctx context.Context, symbol string) (*entity.Symbol, error)

	// RefreshSymbols updates all symbol metadata from the exchange.
	// This should be called periodically to detect maintenance/suspend/delist events.
	RefreshSymbols(ctx context.Context) error

	// CheckPermissions verifies API key permissions.
	// Returns an error if the key lacks required permissions or has dangerous permissions.
	CheckPermissions(ctx context.Context) (*Permissions, error)

	// SubscribeOrders subscribes to order execution updates via WebSocket.
	// The channel is closed when the subscription ends.
	SubscribeOrders(ctx context.Context) (<-chan *OrderUpdate, error)

	// SubscribePositions subscribes to position updates via WebSocket.
	SubscribePositions(ctx context.Context) (<-chan *PositionUpdate, error)

	// SubscribeBalance subscribes to balance updates via WebSocket.
	SubscribeBalance(ctx context.Context) (<-chan *BalanceUpdate, error)

	// SubscribeTicker subscribes to market ticker updates via WebSocket.
	SubscribeTicker(ctx context.Context, symbol string) (<-chan *TickerUpdate, error)
}

// StreamEventSubscriber is implemented by connectors that expose transport-level
// stream lifecycle events such as disconnects, reconnects, and detected gaps.
type StreamEventSubscriber interface {
	// SubscribeStreamEvents subscribes to WebSocket lifecycle events.
	SubscribeStreamEvents(ctx context.Context) (<-chan *StreamEvent, error)
}

// PositionSnapshotReader is implemented by connectors that can return a full
// REST snapshot of account positions for reconciliation.
type PositionSnapshotReader interface {
	// GetPositions retrieves all non-flat positions for the account.
	GetPositions(ctx context.Context) ([]*entity.Position, error)
}

// StreamEventType identifies a connector stream lifecycle event.
type StreamEventType string

const (
	// StreamEventDisconnected indicates that a streaming connection was lost.
	StreamEventDisconnected StreamEventType = "disconnected"

	// StreamEventReconnected indicates that a streaming connection recovered.
	StreamEventReconnected StreamEventType = "reconnected"

	// StreamEventGap indicates that the connector detected a potential stream gap.
	StreamEventGap StreamEventType = "gap"
)

// StreamEvent describes a transport-level stream lifecycle event.
type StreamEvent struct {
	Type          StreamEventType
	Source        string
	Reason        string
	OccurredAtUTC time.Time
}

// Permissions describes API key capabilities.
type Permissions struct {
	CanRead     bool
	CanTrade    bool
	HasWithdraw bool // Dangerous: should trigger warning
	HasTransfer bool // Dangerous: should trigger warning
}

// IsOverPrivileged returns true if the key has dangerous permissions.
func (p *Permissions) IsOverPrivileged() bool {
	return p.HasWithdraw || p.HasTransfer
}

// IsSufficientForTrading returns true if the key can execute trades.
func (p *Permissions) IsSufficientForTrading() bool {
	return p.CanRead && p.CanTrade
}

// OrderUpdate represents a real-time order execution update from the exchange.
type OrderUpdate struct {
	OrderID         string
	ClientOrderID   string
	ExchangeOrderID string
	Status          entity.OrderStatus
	FilledQty       decimal.Decimal
	RemainingQty    decimal.Decimal
	AvgFillPrice    decimal.Decimal
	UpdatedAtUTC    time.Time
}

// PositionUpdate represents a real-time position update from the exchange.
type PositionUpdate struct {
	Symbol        string
	Side          entity.PositionSide
	Quantity      decimal.Decimal
	EntryPrice    decimal.Decimal
	UnrealizedPnL decimal.Decimal
	UpdatedAtUTC  time.Time
}

// BalanceUpdate represents a real-time balance update from the exchange.
type BalanceUpdate struct {
	Asset        string
	Total        decimal.Decimal
	Available    decimal.Decimal
	Locked       decimal.Decimal
	UpdatedAtUTC time.Time
}

// TickerUpdate represents a real-time market ticker update.
type TickerUpdate struct {
	Symbol       string
	LastPrice    decimal.Decimal
	BidPrice     decimal.Decimal
	AskPrice     decimal.Decimal
	Volume24h    decimal.Decimal
	UpdatedAtUTC time.Time
}

// ConnectorConfig contains connector initialization parameters.
type ConnectorConfig struct {
	APIKey      string
	APISecret   string
	Testnet     bool
	ProxyURL    string // SOCKS5 or HTTP proxy URL
	DNSCacheTTL time.Duration
	Timeout     time.Duration
}

// Factory creates connector instances for different exchanges.
type Factory interface {
	// Create instantiates a connector for the specified exchange.
	Create(exchange string, config ConnectorConfig) (Connector, error)

	// SupportedExchanges returns the list of supported exchange identifiers.
	SupportedExchanges() []string
}
