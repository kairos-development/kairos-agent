package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultRouterConfig(t *testing.T) {
	cfg := DefaultRouterConfig()
	assert.Equal(t, RoutingModePaper, cfg.Mode)
	assert.True(t, cfg.EnableIdempotency)
	assert.Equal(t, 30, cfg.TimeoutSeconds)
	assert.Equal(t, 100, cfg.MaxInFlightOrders)
	assert.True(t, cfg.EnableStatusFallback)
}
