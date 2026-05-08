package strategy

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type registryStrategy struct{}

func (registryStrategy) Name() string    { return "test" }
func (registryStrategy) Version() string { return "1.0.0" }
func (registryStrategy) OnTick(context.Context, *StrategyContext) (*Signal, error) {
	return &Signal{Action: SignalActionHold}, nil
}
func (registryStrategy) OnOrderFilled(context.Context, string, decimal.Decimal, decimal.Decimal) error {
	return nil
}
func (registryStrategy) OnOrderCanceled(context.Context, string) error { return nil }
func (registryStrategy) GetParameters() map[string]interface{} {
	return map[string]interface{}{"period": 14}
}
func (registryStrategy) SetParameters(map[string]interface{}) error { return nil }
func (registryStrategy) Reset() error                               { return nil }

func TestStrategyRegistryLifecycle(t *testing.T) {
	registry := NewStrategyRegistry()
	strat := registryStrategy{}
	require.NoError(t, registry.Register("s1", strat))
	assert.Error(t, registry.Register("s1", strat))
	got, err := registry.Get("s1")
	require.NoError(t, err)
	assert.Equal(t, "test", got.Name())
	assert.Len(t, registry.List(), 1)
	require.NoError(t, registry.Unregister("s1"))
	assert.Error(t, registry.Unregister("s1"))
	_, err = registry.Get("s1")
	assert.Error(t, err)
}
