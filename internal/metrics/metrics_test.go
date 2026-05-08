package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	m := New("test")

	require.NotNil(t, m)
	assert.NotNil(t, m.EventsPublished)
	assert.NotNil(t, m.EventsDropped)
	assert.NotNil(t, m.EventSubscribers)
	assert.NotNil(t, m.EventDeliveryTime)
	assert.NotNil(t, m.StateTransitions)
	assert.NotNil(t, m.CurrentState)
	assert.NotNil(t, m.EngineUptime)
	assert.NotNil(t, m.NTPDrift)
	assert.NotNil(t, m.NTPSyncErrors)
	assert.NotNil(t, m.NTPLastSync)
	assert.NotNil(t, m.OrdersCreated)
	assert.NotNil(t, m.OrdersSubmitted)
	assert.NotNil(t, m.OrdersFilled)
	assert.NotNil(t, m.OrdersCanceled)
	assert.NotNil(t, m.OrdersRejected)
	assert.NotNil(t, m.GoroutineCount)
	assert.NotNil(t, m.MemoryUsage)
	assert.NotNil(t, m.CPUUsage)
	assert.NotNil(t, m.CriticalWrites)
	assert.NotNil(t, m.AnalyticsWrites)
	assert.NotNil(t, m.AnalyticsDrops)
	assert.NotNil(t, m.ConnectorErrors)
	assert.NotNil(t, m.ConnectorLatency)
	assert.NotNil(t, m.WebSocketMessages)
}

func TestNew_DefaultNamespace(t *testing.T) {
	m := New("")
	require.NotNil(t, m)
	assert.NotNil(t, m.EventsPublished)
}

func TestNew_Singleton(t *testing.T) {
	m1 := New("test1")
	m2 := New("test2")

	// Should return the same instance due to singleton pattern
	assert.Equal(t, m1, m2)
}

func TestUpdateSystemMetrics(t *testing.T) {
	m := New("test")

	// Should not panic
	m.UpdateSystemMetrics()

	// Verify metrics were updated (values should be > 0)
	// Note: We can't easily verify the exact values, but we can check they're set
}

func TestRecordEventDropped(t *testing.T) {
	m := New("test")

	// Should not panic
	m.RecordEventDropped(5)
	m.RecordEventDropped(10)
}

func TestRecordStorageStats(t *testing.T) {
	m := New("test")

	// Should not panic
	m.RecordStorageStats(100, 200, 5)
	m.RecordStorageStats(50, 75, 2)
}

func TestMetrics_EventsPublished(t *testing.T) {
	m := New("test")

	// Increment event counter
	m.EventsPublished.WithLabelValues("test_event").Inc()
	m.EventsPublished.WithLabelValues("test_event").Add(5)
}

func TestMetrics_StateTransitions(t *testing.T) {
	m := New("test")

	// Record state transitions
	m.StateTransitions.WithLabelValues("Idle", "Scanning").Inc()
	m.StateTransitions.WithLabelValues("Scanning", "LiveTrading").Inc()
}

func TestMetrics_OrderMetrics(t *testing.T) {
	m := New("test")

	// Record order metrics
	m.OrdersCreated.WithLabelValues("BTCUSDT").Inc()
	m.OrdersSubmitted.WithLabelValues("BTCUSDT").Inc()
	m.OrdersFilled.WithLabelValues("BTCUSDT").Inc()
	m.OrdersCanceled.WithLabelValues("BTCUSDT").Inc()
	m.OrdersRejected.WithLabelValues("BTCUSDT").Inc()
}

func TestMetrics_NTPMetrics(t *testing.T) {
	m := New("test")

	// Set NTP metrics
	m.NTPDrift.Set(50.5)
	m.NTPSyncErrors.Inc()
	m.NTPLastSync.Set(1234567890)
}

func TestMetrics_SystemMetrics(t *testing.T) {
	m := New("test")

	// Set system metrics
	m.GoroutineCount.Set(100)
	m.MemoryUsage.Set(1024 * 1024 * 100)
	m.CPUUsage.Set(45.5)
}

func TestMetrics_StorageMetrics(t *testing.T) {
	m := New("test")

	// Increment storage metrics
	m.CriticalWrites.Add(10)
	m.AnalyticsWrites.Add(100)
	m.AnalyticsDrops.Add(5)
}

func TestMetrics_ConnectorMetrics(t *testing.T) {
	m := New("test")

	// Record connector metrics
	m.ConnectorErrors.WithLabelValues("connection_error").Inc()
	m.ConnectorLatency.WithLabelValues("submit_order").Observe(0.150)
	m.WebSocketMessages.WithLabelValues("order_update").Inc()
}

func TestMetrics_EngineMetrics(t *testing.T) {
	m := New("test")

	// Set engine metrics
	m.CurrentState.Set(2) // LiveTrading
	m.EngineUptime.Set(3600)
}

func TestMetrics_EventBusMetrics(t *testing.T) {
	m := New("test")

	// Set event bus metrics
	m.EventSubscribers.Set(5)
	m.EventDeliveryTime.Observe(0.001)
	m.EventsDropped.Add(2)
}
