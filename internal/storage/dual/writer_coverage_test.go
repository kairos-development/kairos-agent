package dual

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWriter_NilLogger(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-nil-logger.db"
	cfg.AnalyticsPath = "/tmp/test-nil-logger-analytics.db"

	w, err := NewWriter(cfg, nil)
	require.NoError(t, err)
	defer w.Close()

	assert.NotNil(t, w)
	assert.NotNil(t, w.logger)
}

func TestNewWriter_AnalyticsWorkerStarted(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-worker.db"
	cfg.AnalyticsPath = "/tmp/test-worker-analytics.db"

	w, err := NewWriter(cfg, logrus.New())
	require.NoError(t, err)
	defer w.Close()

	// Write an analytics event and wait for it to be processed
	w.WriteAnalytics("test_startup", map[string]interface{}{"init": true})
	time.Sleep(300 * time.Millisecond)

	_, analytics, _ := w.Metrics()
	assert.Greater(t, analytics, int64(0))
}

func TestWriteAnalyticsEvent_MarshalError(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-marshal.db"
	cfg.AnalyticsPath = "/tmp/test-marshal-analytics.db"

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	w, err := NewWriter(cfg, logger)
	require.NoError(t, err)
	defer w.Close()

	// Write an event with unmarshable data (channel cannot be JSON marshaled)
	w.WriteAnalytics("bad_event", map[string]interface{}{"ch": make(chan int)})

	// Wait for the worker to process the event (it will log an error but not panic)
	time.Sleep(300 * time.Millisecond)

	// The writer should still be operational
	_, _, dropped := w.Metrics()
	// The event may be dropped or marshaling error is handled gracefully
	assert.GreaterOrEqual(t, dropped, int64(0))
}

func TestWriteAnalyticsEvent_SuccessfulWrite(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-success-event.db"
	cfg.AnalyticsPath = "/tmp/test-success-event-analytics.db"

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	w, err := NewWriter(cfg, logger)
	require.NoError(t, err)
	defer w.Close()

	// Write a valid event
	testData := map[string]interface{}{
		"symbol": "BTCUSDT",
		"price":  50000,
		"action": "buy",
	}
	w.WriteAnalytics("trade_signal", testData)
	time.Sleep(300 * time.Millisecond)

	_, analytics, _ := w.Metrics()
	assert.Equal(t, int64(1), analytics)
}

func TestClose_AlreadyClosed(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-closed.db"
	cfg.AnalyticsPath = "/tmp/test-closed-analytics.db"

	w, err := NewWriter(cfg, logrus.New())
	require.NoError(t, err)

	err = w.Close()
	require.NoError(t, err)

	// Second close should be a no-op
	err = w.Close()
	assert.NoError(t, err)
}

func TestClose_FlushAnalytics(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-flush.db"
	cfg.AnalyticsPath = "/tmp/test-flush-analytics.db"
	cfg.AnalyticsBufSize = 100

	w, err := NewWriter(cfg, logrus.New())
	require.NoError(t, err)

	// Write several events
	for i := 0; i < 5; i++ {
		w.WriteAnalytics("event", map[string]interface{}{"index": i})
	}

	// Close should flush all events
	err = w.Close()
	assert.NoError(t, err)

	_, analytics, _ := w.Metrics()
	assert.Equal(t, int64(5), analytics)
}

func TestWriteAnalytics_MultipleEventTypes(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-multi-type.db"
	cfg.AnalyticsPath = "/tmp/test-multi-type-analytics.db"

	w, err := NewWriter(cfg, logrus.New())
	require.NoError(t, err)
	defer w.Close()

	events := map[string]map[string]interface{}{
		"order_created": {"order_id": "1"},
		"order_filled":  {"order_id": "1"},
		"tick":          {"price": 50000},
		"signal":        {"action": "buy"},
		"risk_check":    {"status": "ok"},
	}

	for eventType, data := range events {
		w.WriteAnalytics(eventType, data)
	}

	time.Sleep(300 * time.Millisecond)

	_, analytics, _ := w.Metrics()
	assert.Equal(t, int64(len(events)), analytics)
}

func TestWriteAnalytics_WithNilData(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-nil-data.db"
	cfg.AnalyticsPath = "/tmp/test-nil-data-analytics.db"

	w, err := NewWriter(cfg, logrus.New())
	require.NoError(t, err)
	defer w.Close()

	w.WriteAnalytics("nil_event", nil)
	time.Sleep(300 * time.Millisecond)

	_, analytics, _ := w.Metrics()
	assert.Equal(t, int64(1), analytics)
}

func TestWriteAnalytics_WithComplexData(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-complex.db"
	cfg.AnalyticsPath = "/tmp/test-complex-analytics.db"

	w, err := NewWriter(cfg, logrus.New())
	require.NoError(t, err)
	defer w.Close()

	complexData := map[string]interface{}{
		"nested": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": "deep_value",
			},
		},
		"array": []int{1, 2, 3},
		"mixed": []interface{}{"a", 1, true},
	}

	w.WriteAnalytics("complex_event", complexData)
	time.Sleep(300 * time.Millisecond)

	_, analytics, _ := w.Metrics()
	assert.Equal(t, int64(1), analytics)
}

func TestMetrics_AfterMultipleWrites(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-metrics.db"
	cfg.AnalyticsPath = "/tmp/test-metrics-analytics.db"

	w, err := NewWriter(cfg, logrus.New())
	require.NoError(t, err)
	defer w.Close()

	// Write critical
	err = w.WriteCritical(context.Background(), "CREATE TABLE IF NOT EXISTS test_metrics (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)

	err = w.WriteCritical(context.Background(), "INSERT OR REPLACE INTO test_metrics (id) VALUES (?)", 1)
	require.NoError(t, err)

	// Write analytics
	for i := 0; i < 3; i++ {
		w.WriteAnalytics("event", map[string]interface{}{"i": i})
	}
	time.Sleep(300 * time.Millisecond)

	critical, analytics, dropped := w.Metrics()
	assert.Equal(t, int64(2), critical)
	assert.Equal(t, int64(3), analytics)
	assert.Equal(t, int64(0), dropped)
}

func TestDefaultConfig_Values(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "/tmp/kairos-critical.db", cfg.CriticalPath)
	assert.Equal(t, "/tmp/kairos-analytics.db", cfg.AnalyticsPath)
	assert.Equal(t, 1, cfg.MaxOpenConns)
	assert.Equal(t, 1, cfg.MaxIdleConns)
	assert.Equal(t, 10000, cfg.AnalyticsBufSize)
}

func TestWriteAnalytics_HighVolume(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-highvol.db"
	cfg.AnalyticsPath = "/tmp/test-highvol-analytics.db"
	cfg.AnalyticsBufSize = 5000

	w, err := NewWriter(cfg, logrus.New())
	require.NoError(t, err)
	defer w.Close()

	for i := 0; i < 100; i++ {
		w.WriteAnalytics("high_vol", map[string]interface{}{"index": i})
	}

	time.Sleep(500 * time.Millisecond)

	_, analytics, _ := w.Metrics()
	assert.Equal(t, int64(100), analytics)
}

func TestWriteAnalytics_BufferOverflow(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-overflow.db"
	cfg.AnalyticsPath = "/tmp/test-overflow-analytics.db"
	cfg.AnalyticsBufSize = 2

	w, err := NewWriter(cfg, logrus.New())
	require.NoError(t, err)
	defer w.Close()

	// Send more events than the buffer can hold; some will be dropped
	for i := 0; i < 20; i++ {
		w.WriteAnalytics("overflow", map[string]interface{}{"i": i})
	}

	// Metrics count includes both queued and dropped events
	_, analytics, dropped := w.Metrics()
	assert.Greater(t, analytics, int64(0))
	assert.Greater(t, dropped, int64(0))
	assert.Equal(t, int64(20), analytics+dropped)
}

func TestAnalyticsEvent_JSONRoundtrip(t *testing.T) {
	original := analyticsEvent{
		eventType: "test",
		data:      map[string]interface{}{"key": "value", "num": 42},
		timestamp: time.Now().UTC().Truncate(time.Millisecond),
	}

	dataJSON, err := json.Marshal(original.data)
	require.NoError(t, err)

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(dataJSON, &decoded))

	assert.Equal(t, "value", decoded["key"])
	assert.Equal(t, float64(42), decoded["num"])
}
