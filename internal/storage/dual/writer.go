package dual

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

type StreamType string

const (
	StreamCritical  StreamType = "critical"
	StreamAnalytics StreamType = "analytics"
)

type Config struct {
	CriticalPath     string
	AnalyticsPath    string
	MaxOpenConns     int
	MaxIdleConns     int
	AnalyticsBufSize int
}

func DefaultConfig() Config {
	return Config{
		CriticalPath:     "/tmp/kairos-critical.db",
		AnalyticsPath:    "/tmp/kairos-analytics.db",
		MaxOpenConns:     1,
		MaxIdleConns:     1,
		AnalyticsBufSize: 10000,
	}
}

type Writer struct {
	criticalDB  *sqlx.DB
	analyticsDB *sqlx.DB
	logger      *logrus.Logger

	analyticsCh chan analyticsEvent
	stopCh      chan struct{}
	wg          sync.WaitGroup

	metricsMu        sync.RWMutex
	criticalWrites   int64
	analyticsWrites  int64
	analyticsDropped int64
	closed           bool
}

type analyticsEvent struct {
	eventType string
	data      interface{}
	timestamp time.Time
}

func NewWriter(cfg Config, logger *logrus.Logger) (*Writer, error) {
	ctx := context.Background()
	if logger == nil {
		logger = logrus.New()
	}

	criticalDB, err := openCriticalDB(cfg.CriticalPath)
	if err != nil {
		return nil, err
	}

	if err := RunMigrations(ctx, criticalDB); err != nil {
		criticalDB.Close()
		return nil, fmt.Errorf("critical migrations: %w", err)
	}

	analyticsDB, err := openAnalyticsDB(cfg.AnalyticsPath)
	if err != nil {
		criticalDB.Close()
		return nil, err
	}

	if err := RunMigrations(ctx, analyticsDB); err != nil {
		criticalDB.Close()
		analyticsDB.Close()
		return nil, fmt.Errorf("analytics migrations: %w", err)
	}

	w := &Writer{
		criticalDB:  criticalDB,
		analyticsDB: analyticsDB,
		logger:      logger,
		analyticsCh: make(chan analyticsEvent, cfg.AnalyticsBufSize),
		stopCh:      make(chan struct{}),
	}

	w.wg.Add(1)
	go w.analyticsWorker()

	logger.WithFields(logrus.Fields{
		"critical_path":  cfg.CriticalPath,
		"analytics_path": cfg.AnalyticsPath,
	}).Info("dual stream writer initialized")

	return w, nil
}

func openCriticalDB(path string) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite3", path+"?_busy_timeout=5000&_journal_mode=WAL&_synchronous=FULL")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db, nil
}

func openAnalyticsDB(path string) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite3", path+"?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	return db, nil
}

func (w *Writer) WriteCritical(ctx context.Context, query string, args ...interface{}) error {
	_, err := w.criticalDB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	w.metricsMu.Lock()
	w.criticalWrites++
	w.metricsMu.Unlock()
	return nil
}

func (w *Writer) WriteAnalytics(eventType string, data interface{}) {
	event := analyticsEvent{
		eventType: eventType,
		data:      data,
		timestamp: time.Now().UTC(),
	}

	select {
	case w.analyticsCh <- event:
		w.metricsMu.Lock()
		w.analyticsWrites++
		w.metricsMu.Unlock()
	default:
		w.metricsMu.Lock()
		w.analyticsDropped++
		w.metricsMu.Unlock()
	}
}

func (w *Writer) analyticsWorker() {
	defer w.wg.Done()

	for {
		select {
		case event := <-w.analyticsCh:
			w.writeAnalyticsEvent(event)
		case <-w.stopCh:
			return
		}
	}
}

func (w *Writer) writeAnalyticsEvent(event analyticsEvent) {
	query := `INSERT INTO analytics_events (event_type, data, timestamp) VALUES (?, ?, ?)`
	dataJSON, err := json.Marshal(event.data)
	if err != nil {
		w.logger.WithError(err).WithField("event_type", event.eventType).Error("failed to marshal analytics data")
		return
	}
	_, err = w.analyticsDB.Exec(query, event.eventType, string(dataJSON), event.timestamp)
	if err != nil {
		w.logger.WithError(err).WithField("event_type", event.eventType).Error("failed to write analytics event")
	}
}

func (w *Writer) Close() error {
	w.metricsMu.Lock()
	if w.closed {
		w.metricsMu.Unlock()
		return nil
	}
	w.closed = true
	w.metricsMu.Unlock()

	close(w.stopCh)
	w.wg.Wait()

	// Drain remaining analytics events from the channel after worker has stopped.
	// Events that were in-flight when stopCh was closed but not yet processed
	// by the worker are lost. This is a best-effort drain to minimize data loss.
	for {
		select {
		case event := <-w.analyticsCh:
			w.writeAnalyticsEvent(event)
		default:
			goto done
		}
	}

done:
	if err := w.criticalDB.Close(); err != nil {
		return err
	}
	return w.analyticsDB.Close()
}

func (w *Writer) Metrics() (critical, analytics, dropped int64) {
	w.metricsMu.RLock()
	defer w.metricsMu.RUnlock()
	return w.criticalWrites, w.analyticsWrites, w.analyticsDropped
}
