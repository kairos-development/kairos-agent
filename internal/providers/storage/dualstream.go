package storageprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

// DualStreamWriter manages two separate SQLite streams: Critical and Analytics.
// Critical stream: synchronous writes with PRAGMA synchronous=FULL for orders, balances, risk events.
// Analytics stream: asynchronous writes with PRAGMA synchronous=NORMAL for logs, ticks, metrics.
type DualStreamWriter struct {
	criticalDB  *sqlx.DB
	analyticsDB *sqlx.DB
	logger      *logrus.Logger

	// Channels for async writes
	analyticsCh chan analyticsEvent
	stopCh      chan struct{}
	wg          sync.WaitGroup

	// Metrics
	criticalWrites   int64
	analyticsWrites  int64
	analyticsDropped int64
	mu               sync.RWMutex
	closed           bool
}

// analyticsEvent represents an event to be written to analytics stream.
type analyticsEvent struct {
	eventType string
	data      map[string]interface{}
	timestamp time.Time
}

// NewDualStreamWriter creates a new dual stream writer.
func NewDualStreamWriter(criticalPath, analyticsPath string, logger *logrus.Logger) (*DualStreamWriter, error) {
	if logger == nil {
		logger = logrus.New()
	}

	// Open critical database with strict synchronization
	criticalDB, err := sqlx.Open("sqlite", criticalPath)
	if err != nil {
		logger.WithError(err).Error("Failed to open critical database")
		return nil, fmt.Errorf("open critical db: %w", err)
	}
	cleanup := func() {
		_ = criticalDB.Close()
	}

	// Configure critical database
	if _, err := criticalDB.Exec("PRAGMA synchronous=FULL"); err != nil {
		logger.WithError(err).Error("Failed to set critical pragma synchronous")
		cleanup()
		return nil, fmt.Errorf("set critical pragma: %w", err)
	}
	if _, err := criticalDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		logger.WithError(err).Error("Failed to set critical journal mode")
		cleanup()
		return nil, fmt.Errorf("set critical journal mode: %w", err)
	}
	if _, err := criticalDB.Exec("PRAGMA fsync=NORMAL"); err != nil {
		logger.WithError(err).Error("Failed to set critical fsync")
		cleanup()
		return nil, fmt.Errorf("set critical fsync: %w", err)
	}

	// Set connection pool for critical (single writer)
	criticalDB.SetMaxOpenConns(1)
	criticalDB.SetMaxIdleConns(1)

	logger.WithField("path", criticalPath).Info("Critical database initialized")

	// Open analytics database with relaxed synchronization
	analyticsDB, err := sqlx.Open("sqlite", analyticsPath)
	if err != nil {
		logger.WithError(err).Error("Failed to open analytics database")
		cleanup()
		return nil, fmt.Errorf("open analytics db: %w", err)
	}
	cleanup = func() {
		_ = criticalDB.Close()
		_ = analyticsDB.Close()
	}

	// Configure analytics database
	if _, err := analyticsDB.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		logger.WithError(err).Error("Failed to set analytics pragma synchronous")
		cleanup()
		return nil, fmt.Errorf("set analytics pragma: %w", err)
	}
	if _, err := analyticsDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		logger.WithError(err).Error("Failed to set analytics journal mode")
		cleanup()
		return nil, fmt.Errorf("set analytics journal mode: %w", err)
	}

	// Allow multiple readers for analytics
	analyticsDB.SetMaxOpenConns(10)
	analyticsDB.SetMaxIdleConns(5)

	logger.WithField("path", analyticsPath).Info("Analytics database initialized")

	dsw := &DualStreamWriter{
		criticalDB:  criticalDB,
		analyticsDB: analyticsDB,
		logger:      logger,
		analyticsCh: make(chan analyticsEvent, 10000), // Large buffer for analytics
		stopCh:      make(chan struct{}),
	}

	// Start analytics writer goroutine
	dsw.wg.Add(1)
	go dsw.analyticsWriter()

	return dsw, nil
}

// WriteCritical writes to the critical stream (synchronous, guaranteed).
func (dsw *DualStreamWriter) WriteCritical(ctx context.Context, query string, args ...interface{}) error {
	// Use context for timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := dsw.criticalDB.ExecContext(ctx, query, args...)
	if err != nil {
		dsw.logger.WithError(err).WithField("query", query).Error("Critical write failed")
		return fmt.Errorf("critical write failed: %w", err)
	}

	dsw.mu.Lock()
	dsw.criticalWrites++
	dsw.mu.Unlock()

	_ = result
	return nil
}

// WriteAnalytics queues an event for asynchronous writing to analytics stream.
// If the queue is full, the event is dropped (QoS 0).
func (dsw *DualStreamWriter) WriteAnalytics(eventType string, data map[string]interface{}) {
	event := analyticsEvent{
		eventType: eventType,
		data:      data,
		timestamp: time.Now().UTC(),
	}

	dsw.mu.Lock()
	if dsw.closed {
		dsw.analyticsDropped++
		dsw.mu.Unlock()
		return
	}

	select {
	case dsw.analyticsCh <- event:
		// Event queued successfully
	default:
		// Queue full, drop event (QoS 0)
		dsw.analyticsDropped++

		dsw.logger.WithField("event_type", eventType).Warn("Analytics event dropped - queue full")
	}
	dsw.mu.Unlock()
}

// analyticsWriter runs in a goroutine and writes queued analytics events.
func (dsw *DualStreamWriter) analyticsWriter() {
	defer dsw.wg.Done()

	for {
		select {
		case event := <-dsw.analyticsCh:
			dsw.writeAnalyticsEvent(event)

		case <-dsw.stopCh:
			dsw.drainAnalytics()
			return
		}
	}
}

func (dsw *DualStreamWriter) drainAnalytics() {
	for {
		select {
		case event := <-dsw.analyticsCh:
			dsw.writeAnalyticsEvent(event)
		default:
			return
		}
	}
}

func (dsw *DualStreamWriter) writeAnalyticsEvent(event analyticsEvent) {
	// Serialize data to JSON to prevent SQL injection.
	dataJSON, err := json.Marshal(event.data)
	if err != nil {
		dsw.logger.WithError(err).WithField("event_type", event.eventType).Error("Failed to marshal analytics data")
		return
	}

	// Write to analytics database with parameterized query.
	query := `INSERT INTO analytics (event_type, data, timestamp) VALUES (?, ?, ?)`
	_, err = dsw.analyticsDB.Exec(query, event.eventType, string(dataJSON), event.timestamp)
	if err != nil {
		dsw.logger.WithError(err).WithField("event_type", event.eventType).Error("Analytics write error")
		return
	}

	dsw.mu.Lock()
	dsw.analyticsWrites++
	dsw.mu.Unlock()
}

// Close closes both databases.
func (dsw *DualStreamWriter) Close() error {
	dsw.mu.Lock()
	if dsw.closed {
		dsw.mu.Unlock()
		return nil
	}
	dsw.closed = true
	dsw.mu.Unlock()

	// Stop analytics writer
	close(dsw.stopCh)
	dsw.wg.Wait()

	dsw.logger.Info("Closing databases")

	// Close databases
	var errs []string
	if err := dsw.criticalDB.Close(); err != nil {
		dsw.logger.WithError(err).Error("Failed to close critical database")
		errs = append(errs, fmt.Sprintf("close critical db: %v", err))
	}
	if err := dsw.analyticsDB.Close(); err != nil {
		dsw.logger.WithError(err).Error("Failed to close analytics database")
		errs = append(errs, fmt.Sprintf("close analytics db: %v", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("close dual stream writer: %s", strings.Join(errs, "; "))
	}

	dsw.logger.Info("Databases closed successfully")
	return nil
}

// Stats returns write statistics.
func (dsw *DualStreamWriter) Stats() (critical, analytics, dropped int64) {
	dsw.mu.RLock()
	defer dsw.mu.RUnlock()
	return dsw.criticalWrites, dsw.analyticsWrites, dsw.analyticsDropped
}

// CleanupOldAnalytics deletes analytics data older than the specified duration.
func (dsw *DualStreamWriter) CleanupOldAnalytics(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().UTC().Add(-olderThan)

	query := "DELETE FROM analytics WHERE timestamp < ?"
	result, err := dsw.analyticsDB.ExecContext(ctx, query, cutoff)
	if err != nil {
		dsw.logger.WithError(err).WithField("cutoff", cutoff).Error("Failed to cleanup analytics")
		return fmt.Errorf("cleanup analytics: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	dsw.logger.WithField("rows_deleted", rowsAffected).Info("Analytics cleanup completed")

	// Vacuum to reclaim space
	_, err = dsw.analyticsDB.ExecContext(ctx, "VACUUM")
	if err != nil {
		dsw.logger.WithError(err).Error("Failed to vacuum analytics database")
		return fmt.Errorf("vacuum analytics: %w", err)
	}

	return nil
}
