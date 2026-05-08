package dual

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWriter(t *testing.T) {
	logger := logrus.New()
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-critical.db"
	cfg.AnalyticsPath = "/tmp/test-analytics.db"

	w, err := NewWriter(cfg, logger)
	require.NoError(t, err)
	defer w.Close()

	assert.NotNil(t, w.criticalDB)
	assert.NotNil(t, w.analyticsDB)
}

func TestWriteCritical(t *testing.T) {
	logger := logrus.New()
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-critical2.db"
	cfg.AnalyticsPath = "/tmp/test-analytics2.db"

	w, err := NewWriter(cfg, logger)
	require.NoError(t, err)
	defer w.Close()

	ctx := context.Background()
	err = w.WriteCritical(ctx, "CREATE TABLE IF NOT EXISTS test_critical (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)

	err = w.WriteCritical(ctx, "DELETE FROM test_critical")
	require.NoError(t, err)

	err = w.WriteCritical(ctx, "INSERT INTO test_critical (id) VALUES (?)", 1)
	require.NoError(t, err)
}

func TestWriteAnalytics(t *testing.T) {
	logger := logrus.New()
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-critical3.db"
	cfg.AnalyticsPath = "/tmp/test-analytics3.db"
	cfg.AnalyticsBufSize = 10

	w, err := NewWriter(cfg, logger)
	require.NoError(t, err)
	defer w.Close()

	w.WriteAnalytics("test_event", map[string]interface{}{"key": "value"})

	time.Sleep(200 * time.Millisecond)

	critical, analytics, dropped := w.Metrics()
	assert.Equal(t, int64(0), critical)
	assert.Equal(t, int64(1), analytics)
	assert.Equal(t, int64(0), dropped)
}

func TestWriteAnalytics_DropOnFull(t *testing.T) {
	logger := logrus.New()
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-critical4.db"
	cfg.AnalyticsPath = "/tmp/test-analytics4.db"
	cfg.AnalyticsBufSize = 2

	w, err := NewWriter(cfg, logger)
	require.NoError(t, err)
	defer w.Close()

	for i := 0; i < 100; i++ {
		w.WriteAnalytics("event", nil)
	}

	time.Sleep(200 * time.Millisecond)

	_, _, dropped := w.Metrics()
	assert.Greater(t, dropped, int64(0))
}

func TestClose(t *testing.T) {
	logger := logrus.New()
	cfg := DefaultConfig()
	cfg.CriticalPath = "/tmp/test-critical5.db"
	cfg.AnalyticsPath = "/tmp/test-analytics5.db"

	w, err := NewWriter(cfg, logger)
	require.NoError(t, err)

	err = w.Close()
	assert.NoError(t, err)

	_, _, dropped := w.Metrics()
	assert.Equal(t, int64(0), dropped)
}
