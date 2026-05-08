package entity

// RoutingMode identifies the order routing destination.
type RoutingMode string

const (
	// RoutingModeLive routes orders to the live exchange.
	RoutingModeLive RoutingMode = "live"
	// RoutingModePaper routes orders to the paper trading simulator.
	RoutingModePaper RoutingMode = "paper"
)

// RouterConfig contains order router configuration.
type RouterConfig struct {
	Mode                 RoutingMode
	EnableIdempotency    bool
	TimeoutSeconds       int
	MaxInFlightOrders    int
	EnableStatusFallback bool
}

// DefaultRouterConfig returns default router configuration.
func DefaultRouterConfig() *RouterConfig {
	return &RouterConfig{
		Mode:                 RoutingModePaper,
		EnableIdempotency:    true,
		TimeoutSeconds:       30,
		MaxInFlightOrders:    100,
		EnableStatusFallback: true,
	}
}
