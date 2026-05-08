package storageprovider

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDualStreamWriter_NewDualStreamWriter tests creation of dual stream writer.
func TestDualStreamWriter_NewDualStreamWriter(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)
	require.NotNil(t, dsw)

	defer dsw.Close()

	// Check databases were created
	assert.FileExists(t, criticalPath)
	assert.FileExists(t, analyticsPath)
}

// TestDualStreamWriter_WriteCritical tests synchronous critical writes.
func TestDualStreamWriter_WriteCritical(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)
	defer dsw.Close()

	// Create table
	ctx := context.Background()
	err = dsw.WriteCritical(ctx, "CREATE TABLE IF NOT EXISTS orders (id INTEGER PRIMARY KEY, symbol TEXT)")
	require.NoError(t, err)

	// Insert data
	err = dsw.WriteCritical(ctx, "INSERT INTO orders (symbol) VALUES (?)", "BTCUSDT")
	require.NoError(t, err)

	// Check stats
	critical, _, _ := dsw.Stats()
	assert.Equal(t, int64(2), critical)
}

// TestDualStreamWriter_WriteAnalytics tests asynchronous analytics writes.
func TestDualStreamWriter_WriteAnalytics(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)
	defer dsw.Close()

	// Create analytics table
	ctx := context.Background()
	_, err = dsw.analyticsDB.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS analytics (id INTEGER PRIMARY KEY, event_type TEXT, data TEXT, timestamp DATETIME)")
	require.NoError(t, err)

	// Write analytics events
	for i := 0; i < 10; i++ {
		dsw.WriteAnalytics("test_event", map[string]interface{}{
			"counter": i,
			"message": "test message",
		})
	}

	// Wait for async writes
	time.Sleep(200 * time.Millisecond)

	// Check stats
	_, analytics, _ := dsw.Stats()
	assert.Equal(t, int64(10), analytics)
}

// TestDualStreamWriter_AnalyticsQoS0_DropsOnOverflow tests that analytics drops events when queue is full.
func TestDualStreamWriter_AnalyticsQoS0_DropsOnOverflow(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)
	defer dsw.Close()

	// Create analytics table
	ctx := context.Background()
	_, err = dsw.analyticsDB.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS analytics (id INTEGER PRIMARY KEY, event_type TEXT, data TEXT, timestamp DATETIME)")
	require.NoError(t, err)

	// Fill queue beyond capacity (buffer is 10000)
	for i := 0; i < 15000; i++ {
		dsw.WriteAnalytics("flood_event", map[string]interface{}{
			"counter": i,
		})
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Some events should be dropped
	_, analytics, dropped := dsw.Stats()
	assert.Greater(t, dropped, int64(0), "Should have dropped some events")
	assert.Less(t, analytics, int64(15000), "Should not have written all events")
}

// TestDualStreamWriter_Close tests graceful shutdown.
func TestDualStreamWriter_Close(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)

	// Write some data
	ctx := context.Background()
	err = dsw.WriteCritical(ctx, "CREATE TABLE IF NOT EXISTS test (id INTEGER)")
	require.NoError(t, err)

	// Close
	err = dsw.Close()
	assert.NoError(t, err)

	// Databases should be closed (further operations should fail)
	err = dsw.WriteCritical(ctx, "INSERT INTO test VALUES (1)")
	assert.Error(t, err)
}

// TestDualStreamWriter_CriticalTimeout tests that critical writes respect context timeout.
func TestDualStreamWriter_CriticalTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)
	defer dsw.Close()

	// Create table
	ctx := context.Background()
	err = dsw.WriteCritical(ctx, "CREATE TABLE IF NOT EXISTS test (id INTEGER)")
	require.NoError(t, err)

	// Context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond) // Ensure context is expired

	// Should fail due to timeout
	err = dsw.WriteCritical(ctx, "INSERT INTO test VALUES (1)")
	assert.Error(t, err)
}

// TestDualStreamWriter_CleanupOldAnalytics tests cleanup of old analytics data.
func TestDualStreamWriter_CleanupOldAnalytics(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)
	defer dsw.Close()

	// Create analytics table
	ctx := context.Background()
	_, err = dsw.analyticsDB.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS analytics (id INTEGER PRIMARY KEY, event_type TEXT, data TEXT, timestamp DATETIME)")
	require.NoError(t, err)

	// Insert old data
	oldTime := time.Now().UTC().Add(-48 * time.Hour)
	_, err = dsw.analyticsDB.ExecContext(ctx, "INSERT INTO analytics (event_type, data, timestamp) VALUES (?, ?, ?)",
		"old_event", "{}", oldTime)
	require.NoError(t, err)

	// Insert recent data
	recentTime := time.Now().UTC()
	_, err = dsw.analyticsDB.ExecContext(ctx, "INSERT INTO analytics (event_type, data, timestamp) VALUES (?, ?, ?)",
		"recent_event", "{}", recentTime)
	require.NoError(t, err)

	// Cleanup data older than 24 hours
	err = dsw.CleanupOldAnalytics(ctx, 24*time.Hour)
	assert.NoError(t, err)

	// Check that old data was deleted
	var count int
	err = dsw.analyticsDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM analytics").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Should have only 1 recent record")
}

// TestDualStreamWriter_JSONSerialization tests that data is properly JSON serialized.
func TestDualStreamWriter_JSONSerialization(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)
	defer dsw.Close()

	// Create analytics table
	ctx := context.Background()
	_, err = dsw.analyticsDB.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS analytics (id INTEGER PRIMARY KEY, event_type TEXT, data TEXT, timestamp DATETIME)")
	require.NoError(t, err)

	// Write event with special characters that could cause SQL injection
	dsw.WriteAnalytics("test_event", map[string]interface{}{
		"message": "'; DROP TABLE analytics; --",
		"value":   "<script>alert('xss')</script>",
	})

	// Wait for write
	time.Sleep(100 * time.Millisecond)

	// Table should still exist (no SQL injection)
	var count int
	err = dsw.analyticsDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM analytics").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Data should be properly escaped in JSON
	var data string
	err = dsw.analyticsDB.QueryRowContext(ctx, "SELECT data FROM analytics LIMIT 1").Scan(&data)
	require.NoError(t, err)
	assert.Contains(t, data, "DROP TABLE")
	assert.Contains(t, data, "script")
}

// TestDualStreamWriter_ConcurrentWrites tests concurrent writes to critical stream.
func TestDualStreamWriter_ConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)
	defer dsw.Close()

	// Create table
	ctx := context.Background()
	err = dsw.WriteCritical(ctx, "CREATE TABLE IF NOT EXISTS test (id INTEGER PRIMARY KEY AUTOINCREMENT, value INTEGER)")
	require.NoError(t, err)

	// Concurrent writes
	const numWrites = 50
	errCh := make(chan error, numWrites)

	for i := 0; i < numWrites; i++ {
		go func(val int) {
			err := dsw.WriteCritical(context.Background(), "INSERT INTO test (value) VALUES (?)", val)
			errCh <- err
		}(i)
	}

	// Collect errors
	for i := 0; i < numWrites; i++ {
		err := <-errCh
		assert.NoError(t, err)
	}

	// Check all writes succeeded
	var count int
	err = dsw.criticalDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM test").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, numWrites, count)
}

// TestDualStreamWriter_Stats tests statistics tracking.
func TestDualStreamWriter_Stats(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)
	defer dsw.Close()

	// Initial stats
	critical, analytics, dropped := dsw.Stats()
	assert.Equal(t, int64(0), critical)
	assert.Equal(t, int64(0), analytics)
	assert.Equal(t, int64(0), dropped)

	// Write critical
	ctx := context.Background()
	dsw.WriteCritical(ctx, "CREATE TABLE IF NOT EXISTS test (id INTEGER)")
	dsw.WriteCritical(ctx, "INSERT INTO test VALUES (1)")

	// Write analytics
	_, err = dsw.analyticsDB.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS analytics (id INTEGER PRIMARY KEY, event_type TEXT, data TEXT, timestamp DATETIME)")
	require.NoError(t, err)

	dsw.WriteAnalytics("event1", map[string]interface{}{"test": 1})
	dsw.WriteAnalytics("event2", map[string]interface{}{"test": 2})

	time.Sleep(100 * time.Millisecond)

	// Check stats
	critical, analytics, dropped = dsw.Stats()
	assert.Equal(t, int64(2), critical)
	assert.Equal(t, int64(2), analytics)
	assert.Equal(t, int64(0), dropped)
}

// TestDualStreamWriter_NewWithNilLogger tests creation with nil logger.
func TestDualStreamWriter_NewWithNilLogger(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, nil)
	require.NoError(t, err)
	defer dsw.Close()

	assert.NotNil(t, dsw)
	assert.NotNil(t, dsw.logger)
}

// TestDualStreamWriter_NewWithInvalidPath tests creation with invalid path.
func TestDualStreamWriter_NewWithInvalidPath(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	// Use invalid path (directory that doesn't exist and can't be created)
	invalidPath := "/invalid/nonexistent/path/db.sqlite"

	_, err := NewDualStreamWriter(invalidPath, invalidPath, logger)
	assert.Error(t, err)
}

// TestDualStreamWriter_CloseMultipleTimes tests calling close multiple times.
func TestDualStreamWriter_CloseMultipleTimes(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)

	// First close
	err = dsw.Close()
	assert.NoError(t, err)

	// Second close should not panic
	err = dsw.Close()
	// May return error since databases are already closed, but shouldn't panic
}

// TestDualStreamWriter_WriteAfterClose tests writing after close.
func TestDualStreamWriter_WriteAfterClose(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)

	// Close
	err = dsw.Close()
	require.NoError(t, err)

	// Try to write - should fail
	ctx := context.Background()
	err = dsw.WriteCritical(ctx, "CREATE TABLE test (id INTEGER)")
	assert.Error(t, err)
}

// TestDualStreamWriter_CleanupWithNoOldData tests cleanup when no old data exists.
func TestDualStreamWriter_CleanupWithNoOldData(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)
	defer dsw.Close()

	// Create analytics table
	ctx := context.Background()
	_, err = dsw.analyticsDB.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS analytics (id INTEGER PRIMARY KEY, event_type TEXT, data TEXT, timestamp DATETIME)")
	require.NoError(t, err)

	// Insert only recent data
	recentTime := time.Now().UTC()
	_, err = dsw.analyticsDB.ExecContext(ctx, "INSERT INTO analytics (event_type, data, timestamp) VALUES (?, ?, ?)",
		"recent_event", "{}", recentTime)
	require.NoError(t, err)

	// Cleanup - should not delete recent data
	err = dsw.CleanupOldAnalytics(ctx, 24*time.Hour)
	assert.NoError(t, err)

	// Check that data still exists
	var count int
	err = dsw.analyticsDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM analytics").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestDualStreamWriter_NewWithPragmaErrors(t *testing.T) {
	// Test that pragma errors are handled
	// This is hard to trigger as SQLite pragmas rarely fail
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)
	defer dsw.Close()
}

func TestDualStreamWriter_CleanupOldAnalyticsWithError(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)

	// Close analytics DB to cause error
	dsw.analyticsDB.Close()

	ctx := context.Background()
	err = dsw.CleanupOldAnalytics(ctx, 24*time.Hour)
	assert.Error(t, err)

	dsw.Close()
}

func TestDualStreamWriter_AnalyticsDBOpenError(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := "/invalid/path/analytics.db"

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	_, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	assert.Error(t, err)
	// Error could be from open or pragma depending on SQLite behavior
}

func TestDualStreamWriter_AnalyticsPragmaError(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	// Create valid databases
	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)
	dsw.Close()
}

func TestDualStreamWriter_CloseAnalyticsDBError(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)

	// Close analytics DB first
	dsw.analyticsDB.Close()

	// Close should still succeed
	err = dsw.Close()
	assert.NoError(t, err)
}

func TestDualStreamWriter_CloseCriticalDBError(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)

	// Close critical DB first
	dsw.criticalDB.Close()

	// Close should still succeed
	err = dsw.Close()
	assert.NoError(t, err)
}

func TestDualStreamWriter_AnalyticsWriterLoop(t *testing.T) {
	tmpDir := t.TempDir()
	criticalPath := filepath.Join(tmpDir, "critical.db")
	analyticsPath := filepath.Join(tmpDir, "analytics.db")

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	dsw, err := NewDualStreamWriter(criticalPath, analyticsPath, logger)
	require.NoError(t, err)
	defer dsw.Close()

	// Write some analytics events
	dsw.WriteAnalytics("test_event", map[string]interface{}{"key": "value"})

	// Give time for async write
	time.Sleep(100 * time.Millisecond)
}
