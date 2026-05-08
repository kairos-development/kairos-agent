package bybit

import (
	"context"

	"github.com/kairos-development/kairos-agent/internal/domain/connector"
	"github.com/kairos-development/kairos-agent/internal/domain/entity"
	connectorpkg "github.com/kairos-development/kairos-contracts/connector"
)

// Adapter adapts the Bybit connector to the agent's domain connector interface.
type Adapter struct {
	conn connectorpkg.Connector
}

// NewAdapter creates a new Bybit connector adapter.
func NewAdapter(conn connectorpkg.Connector) *Adapter {
	return &Adapter{conn: conn}
}

// Name returns the exchange identifier.
func (a *Adapter) Name() string {
	return a.conn.Name()
}

// Connect establishes exchange connections and authenticates.
func (a *Adapter) Connect(ctx context.Context) error {
	return a.conn.Connect(ctx)
}

// Disconnect gracefully closes all connections.
func (a *Adapter) Disconnect(ctx context.Context) error {
	return a.conn.Disconnect(ctx)
}

// IsConnected returns true if the connector is currently connected.
func (a *Adapter) IsConnected() bool {
	return a.conn.IsConnected()
}

// SubmitOrder sends an order to the exchange.
func (a *Adapter) SubmitOrder(ctx context.Context, order *entity.Order) (string, error) {
	connOrder := &connectorpkg.Order{
		ID:              order.ID,
		ClientOrderID:   order.ClientOrderID,
		ExchangeOrderID: order.ExchangeOrderID,
		StrategyID:      order.StrategyID,
		Symbol:          order.Symbol,
		Side:            mapOrderSide(order.Side),
		Type:            mapOrderType(order.Type),
		Status:          mapOrderStatus(order.Status),
		TimeInForce:     mapTimeInForce(order.TimeInForce),
		Quantity:        order.Quantity,
		Price:           order.Price,
		FilledQty:       order.FilledQty,
		RemainingQty:    order.RemainingQty,
		AvgFillPrice:    order.AvgFillPrice,
		CreatedAtUTC:    order.CreatedAtUTC,
		UpdatedAtUTC:    order.UpdatedAtUTC,
		SubmittedAtUTC:  order.SubmittedAtUTC,
		FilledAtUTC:     order.FilledAtUTC,
	}

	return a.conn.SubmitOrder(ctx, connOrder)
}

// CancelOrder cancels an active order on the exchange.
func (a *Adapter) CancelOrder(ctx context.Context, orderID string) error {
	return a.conn.CancelOrder(ctx, orderID)
}

// GetOpenOrders retrieves all open orders for the account.
func (a *Adapter) GetOpenOrders(ctx context.Context) ([]*entity.Order, error) {
	connOrders, err := a.conn.GetOpenOrders(ctx)
	if err != nil {
		return nil, err
	}

	orders := make([]*entity.Order, 0, len(connOrders))
	for _, co := range connOrders {
		orders = append(orders, unmapOrder(co))
	}

	return orders, nil
}

// QueryOrder retrieves the current status of an order from the exchange.
func (a *Adapter) QueryOrder(ctx context.Context, orderID string) (*entity.Order, error) {
	connOrder, err := a.conn.QueryOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	return unmapOrder(connOrder), nil
}

// GetPosition retrieves the current position for a symbol.
func (a *Adapter) GetPosition(ctx context.Context, symbol string) (*entity.Position, error) {
	connPos, err := a.conn.GetPosition(ctx, symbol)
	if err != nil {
		return nil, err
	}

	return unmapPosition(connPos), nil
}

// GetBalance retrieves the current account balance.
func (a *Adapter) GetBalance(ctx context.Context) (*entity.AccountBalance, error) {
	connBal, err := a.conn.GetBalance(ctx)
	if err != nil {
		return nil, err
	}

	balances := make([]entity.Balance, 0, len(connBal.Balances))
	for _, cb := range connBal.Balances {
		balances = append(balances, entity.Balance{
			Asset:        cb.Asset,
			Total:        cb.Total,
			Available:    cb.Available,
			Locked:       cb.Locked,
			UpdatedAtUTC: cb.UpdatedAtUTC,
		})
	}

	return &entity.AccountBalance{
		Balances:     balances,
		UpdatedAtUTC: connBal.UpdatedAtUTC,
	}, nil
}

// GetSymbol retrieves symbol metadata and trading constraints.
func (a *Adapter) GetSymbol(ctx context.Context, symbol string) (*entity.Symbol, error) {
	connSym, err := a.conn.GetSymbol(ctx, symbol)
	if err != nil {
		return nil, err
	}

	return &entity.Symbol{
		Name:          connSym.Name,
		BaseCurrency:  connSym.BaseCurrency,
		QuoteCurrency: connSym.QuoteCurrency,
		Status:        mapSymbolStatus(connSym.Status),
		MinOrderQty:   connSym.MinOrderQty,
		MaxOrderQty:   connSym.MaxOrderQty,
		StepSize:      connSym.StepSize,
		MinPrice:      connSym.MinPrice,
		MaxPrice:      connSym.MaxPrice,
		TickSize:      connSym.TickSize,
		MinNotional:   connSym.MinNotional,
		MakerFee:      connSym.MakerFee,
		TakerFee:      connSym.TakerFee,
		UpdatedAtUTC:  connSym.UpdatedAtUTC,
	}, nil
}

// RefreshSymbols updates all symbol metadata from the exchange.
func (a *Adapter) RefreshSymbols(ctx context.Context) error {
	return a.conn.RefreshSymbols(ctx)
}

// CheckPermissions verifies API key permissions.
func (a *Adapter) CheckPermissions(ctx context.Context) (*connector.Permissions, error) {
	connPerms, err := a.conn.CheckPermissions(ctx)
	if err != nil {
		return nil, err
	}

	return &connector.Permissions{
		CanRead:     connPerms.CanRead,
		CanTrade:    connPerms.CanTrade,
		HasWithdraw: connPerms.HasWithdraw,
		HasTransfer: connPerms.HasTransfer,
	}, nil
}

// SubscribeOrders subscribes to order execution updates via WebSocket.
func (a *Adapter) SubscribeOrders(ctx context.Context) (<-chan *connector.OrderUpdate, error) {
	connCh, err := a.conn.SubscribeOrders(ctx)
	if err != nil {
		return nil, err
	}

	ch := make(chan *connector.OrderUpdate, 100)

	go func() {
		defer close(ch)
		for update := range connCh {
			ch <- &connector.OrderUpdate{
				OrderID:         update.OrderID,
				ClientOrderID:   update.ClientOrderID,
				ExchangeOrderID: update.ExchangeOrderID,
				Status:          unmapOrderStatusFromConnector(update.Status),
				FilledQty:       update.FilledQty,
				RemainingQty:    update.RemainingQty,
				AvgFillPrice:    update.AvgFillPrice,
				UpdatedAtUTC:    update.UpdatedAtUTC,
			}
		}
	}()

	return ch, nil
}

// SubscribePositions subscribes to position updates via WebSocket.
func (a *Adapter) SubscribePositions(ctx context.Context) (<-chan *connector.PositionUpdate, error) {
	connCh, err := a.conn.SubscribePositions(ctx)
	if err != nil {
		return nil, err
	}

	ch := make(chan *connector.PositionUpdate, 100)

	go func() {
		defer close(ch)
		for update := range connCh {
			ch <- &connector.PositionUpdate{
				Symbol:        update.Symbol,
				Side:          unmapPositionSideFromConnector(update.Side),
				Quantity:      update.Quantity,
				EntryPrice:    update.EntryPrice,
				UnrealizedPnL: update.UnrealizedPnL,
				UpdatedAtUTC:  update.UpdatedAtUTC,
			}
		}
	}()

	return ch, nil
}

// SubscribeBalance subscribes to balance updates via WebSocket.
func (a *Adapter) SubscribeBalance(ctx context.Context) (<-chan *connector.BalanceUpdate, error) {
	connCh, err := a.conn.SubscribeBalance(ctx)
	if err != nil {
		return nil, err
	}

	ch := make(chan *connector.BalanceUpdate, 100)

	go func() {
		defer close(ch)
		for update := range connCh {
			ch <- &connector.BalanceUpdate{
				Asset:        update.Asset,
				Total:        update.Total,
				Available:    update.Available,
				Locked:       update.Locked,
				UpdatedAtUTC: update.UpdatedAtUTC,
			}
		}
	}()

	return ch, nil
}

// SubscribeTicker subscribes to market ticker updates via WebSocket.
func (a *Adapter) SubscribeTicker(ctx context.Context, symbol string) (<-chan *connector.TickerUpdate, error) {
	connCh, err := a.conn.SubscribeTicker(ctx, symbol)
	if err != nil {
		return nil, err
	}

	ch := make(chan *connector.TickerUpdate, 100)

	go func() {
		defer close(ch)
		for update := range connCh {
			ch <- &connector.TickerUpdate{
				Symbol:       update.Symbol,
				LastPrice:    update.LastPrice,
				BidPrice:     update.BidPrice,
				AskPrice:     update.AskPrice,
				Volume24h:    update.Volume24h,
				UpdatedAtUTC: update.UpdatedAtUTC,
			}
		}
	}()

	return ch, nil
}

// Mapping functions

func mapOrderSide(side entity.OrderSide) connectorpkg.OrderSide {
	if side == entity.OrderSideBuy {
		return connectorpkg.OrderSideBuy
	}
	return connectorpkg.OrderSideSell
}

func mapOrderType(orderType entity.OrderType) connectorpkg.OrderType {
	if orderType == entity.OrderTypeLimit {
		return connectorpkg.OrderTypeLimit
	}
	return connectorpkg.OrderTypeMarket
}

func mapOrderStatus(status entity.OrderStatus) connectorpkg.OrderStatus {
	switch status {
	case entity.OrderStatusPending:
		return connectorpkg.OrderStatusPending
	case entity.OrderStatusSubmitted:
		return connectorpkg.OrderStatusSubmitted
	case entity.OrderStatusPartial:
		return connectorpkg.OrderStatusPartial
	case entity.OrderStatusFilled:
		return connectorpkg.OrderStatusFilled
	case entity.OrderStatusCanceled:
		return connectorpkg.OrderStatusCanceled
	case entity.OrderStatusRejected:
		return connectorpkg.OrderStatusRejected
	default:
		return connectorpkg.OrderStatusPending
	}
}

func mapTimeInForce(tif entity.TimeInForce) connectorpkg.TimeInForce {
	switch tif {
	case entity.TimeInForceGTC:
		return connectorpkg.TimeInForceGTC
	case entity.TimeInForceIOC:
		return connectorpkg.TimeInForceIOC
	case entity.TimeInForceFOK:
		return connectorpkg.TimeInForceFOK
	default:
		return connectorpkg.TimeInForceGTC
	}
}

func mapSymbolStatus(status connectorpkg.SymbolStatus) entity.SymbolStatus {
	switch status {
	case connectorpkg.SymbolStatusTrading:
		return entity.SymbolStatusTrading
	case connectorpkg.SymbolStatusSuspended:
		return entity.SymbolStatusSuspended
	case connectorpkg.SymbolStatusMaintenance:
		return entity.SymbolStatusMaintenance
	default:
		return entity.SymbolStatusSuspended
	}
}

func unmapOrder(co *connectorpkg.Order) *entity.Order {
	return &entity.Order{
		ID:              co.ID,
		ClientOrderID:   co.ClientOrderID,
		ExchangeOrderID: co.ExchangeOrderID,
		StrategyID:      co.StrategyID,
		Symbol:          co.Symbol,
		Side:            unmapOrderSideFromConnector(co.Side),
		Type:            unmapOrderTypeFromConnector(co.Type),
		Status:          unmapOrderStatusFromConnector(co.Status),
		TimeInForce:     unmapTimeInForceFromConnector(co.TimeInForce),
		Quantity:        co.Quantity,
		Price:           co.Price,
		FilledQty:       co.FilledQty,
		RemainingQty:    co.RemainingQty,
		AvgFillPrice:    co.AvgFillPrice,
		CreatedAtUTC:    co.CreatedAtUTC,
		UpdatedAtUTC:    co.UpdatedAtUTC,
		SubmittedAtUTC:  co.SubmittedAtUTC,
		FilledAtUTC:     co.FilledAtUTC,
	}
}

func unmapPosition(cp *connectorpkg.Position) *entity.Position {
	return &entity.Position{
		Symbol:        cp.Symbol,
		Side:          unmapPositionSideFromConnector(cp.Side),
		Quantity:      cp.Quantity,
		EntryPrice:    cp.EntryPrice,
		CurrentPrice:  cp.CurrentPrice,
		UnrealizedPnL: cp.UnrealizedPnL,
		RealizedPnL:   cp.RealizedPnL,
	}
}

func unmapOrderSideFromConnector(side connectorpkg.OrderSide) entity.OrderSide {
	if side == connectorpkg.OrderSideBuy {
		return entity.OrderSideBuy
	}
	return entity.OrderSideSell
}

func unmapOrderTypeFromConnector(orderType connectorpkg.OrderType) entity.OrderType {
	if orderType == connectorpkg.OrderTypeLimit {
		return entity.OrderTypeLimit
	}
	return entity.OrderTypeMarket
}

func unmapOrderStatusFromConnector(status connectorpkg.OrderStatus) entity.OrderStatus {
	switch status {
	case connectorpkg.OrderStatusPending:
		return entity.OrderStatusPending
	case connectorpkg.OrderStatusSubmitted:
		return entity.OrderStatusSubmitted
	case connectorpkg.OrderStatusPartial:
		return entity.OrderStatusPartial
	case connectorpkg.OrderStatusFilled:
		return entity.OrderStatusFilled
	case connectorpkg.OrderStatusCanceled:
		return entity.OrderStatusCanceled
	case connectorpkg.OrderStatusRejected:
		return entity.OrderStatusRejected
	default:
		return entity.OrderStatusPending
	}
}

func unmapTimeInForceFromConnector(tif connectorpkg.TimeInForce) entity.TimeInForce {
	switch tif {
	case connectorpkg.TimeInForceGTC:
		return entity.TimeInForceGTC
	case connectorpkg.TimeInForceIOC:
		return entity.TimeInForceIOC
	case connectorpkg.TimeInForceFOK:
		return entity.TimeInForceFOK
	default:
		return entity.TimeInForceGTC
	}
}

func unmapPositionSideFromConnector(side connectorpkg.PositionSide) entity.PositionSide {
	switch side {
	case connectorpkg.PositionSideLong:
		return entity.PositionSideLong
	case connectorpkg.PositionSideShort:
		return entity.PositionSideShort
	default:
		return entity.PositionSideFlat
	}
}
