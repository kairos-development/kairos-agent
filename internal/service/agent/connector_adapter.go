package agent

import (
	"context"

	domainconnector "github.com/kairos-development/kairos-agent/internal/domain/connector"
	"github.com/kairos-development/kairos-agent/internal/domain/entity"
)

// ConnectorAdapter adapts between domain entities and connector types.
type ConnectorAdapter struct {
	conn domainconnector.Connector
}

// NewConnectorAdapter creates a new connector adapter.
func NewConnectorAdapter(conn domainconnector.Connector) *ConnectorAdapter {
	return &ConnectorAdapter{conn: conn}
}

// SubmitOrder submits an order through the domain connector.
func (a *ConnectorAdapter) SubmitOrder(ctx context.Context, order *entity.Order) (string, error) {
	return a.conn.SubmitOrder(ctx, order)
}

// CancelOrder cancels an order on the exchange.
func (a *ConnectorAdapter) CancelOrder(ctx context.Context, orderID string) error {
	return a.conn.CancelOrder(ctx, orderID)
}

// GetOpenOrders retrieves all active orders from the exchange.
func (a *ConnectorAdapter) GetOpenOrders(ctx context.Context) ([]*entity.Order, error) {
	return a.conn.GetOpenOrders(ctx)
}

// QueryOrder retrieves order status from the exchange.
func (a *ConnectorAdapter) QueryOrder(ctx context.Context, orderID string) (*entity.Order, error) {
	return a.conn.QueryOrder(ctx, orderID)
}

// QueryOrderStatus retrieves order status from the exchange.
func (a *ConnectorAdapter) QueryOrderStatus(ctx context.Context, exchangeOrderID string) (*entity.Order, error) {
	return a.conn.QueryOrder(ctx, exchangeOrderID)
}

// GetPosition retrieves a position from the exchange.
func (a *ConnectorAdapter) GetPosition(ctx context.Context, symbol string) (*entity.Position, error) {
	return a.conn.GetPosition(ctx, symbol)
}

// GetPositions retrieves all non-flat positions from the exchange.
func (a *ConnectorAdapter) GetPositions(ctx context.Context) ([]*entity.Position, error) {
	reader, ok := a.conn.(domainconnector.PositionSnapshotReader)
	if !ok {
		return nil, domainconnector.ErrPositionSnapshotsUnsupported
	}
	return reader.GetPositions(ctx)
}

// GetBalance retrieves account balance from the exchange.
func (a *ConnectorAdapter) GetBalance(ctx context.Context) (*entity.AccountBalance, error) {
	return a.conn.GetBalance(ctx)
}

// GetSymbol retrieves symbol metadata from the exchange.
func (a *ConnectorAdapter) GetSymbol(ctx context.Context, symbol string) (*entity.Symbol, error) {
	return a.conn.GetSymbol(ctx, symbol)
}
