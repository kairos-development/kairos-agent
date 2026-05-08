package metrics

import (
	"runtime"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	once     sync.Once
	instance *Metrics
)

// Metrics holds all Prometheus metrics for the trading engine.
type Metrics struct {
	// Event bus metrics
	EventsPublished   *prometheus.CounterVec
	EventsDropped     prometheus.Counter
	EventSubscribers  prometheus.Gauge
	EventDeliveryTime prometheus.Histogram

	// Engine metrics
	StateTransitions *prometheus.CounterVec
	CurrentState     prometheus.Gauge
	EngineUptime     prometheus.Gauge

	// NTP metrics
	NTPDrift      prometheus.Gauge
	NTPSyncErrors prometheus.Counter
	NTPLastSync   prometheus.Gauge

	// Order metrics
	OrdersCreated   *prometheus.CounterVec
	OrdersSubmitted *prometheus.CounterVec
	OrdersFilled    *prometheus.CounterVec
	OrdersCanceled  *prometheus.CounterVec
	OrdersRejected  *prometheus.CounterVec

	// System metrics
	GoroutineCount prometheus.Gauge
	MemoryUsage    prometheus.Gauge
	CPUUsage       prometheus.Gauge

	// Storage metrics
	CriticalWrites  prometheus.Counter
	AnalyticsWrites prometheus.Counter
	AnalyticsDrops  prometheus.Counter

	// Connector metrics
	ConnectorErrors   *prometheus.CounterVec
	ConnectorLatency  *prometheus.HistogramVec
	WebSocketMessages *prometheus.CounterVec
}

// New creates a new Metrics instance with all Prometheus metrics registered.
// Uses singleton pattern to avoid duplicate metric registration.
func New(namespace string) *Metrics {
	once.Do(func() {
		if namespace == "" {
			namespace = "kairos"
		}

		instance = &Metrics{
			// Event bus metrics
			EventsPublished: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "eventbus",
					Name:      "events_published_total",
					Help:      "Total number of events published by type",
				},
				[]string{"event_type"},
			),
			EventsDropped: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "eventbus",
					Name:      "events_dropped_total",
					Help:      "Total number of events dropped due to slow subscribers",
				},
			),
			EventSubscribers: promauto.NewGauge(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: "eventbus",
					Name:      "subscribers_count",
					Help:      "Current number of event subscribers",
				},
			),
			EventDeliveryTime: promauto.NewHistogram(
				prometheus.HistogramOpts{
					Namespace: namespace,
					Subsystem: "eventbus",
					Name:      "event_delivery_seconds",
					Help:      "Time taken to deliver events to subscribers",
					Buckets:   prometheus.DefBuckets,
				},
			),

			// Engine metrics
			StateTransitions: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "engine",
					Name:      "state_transitions_total",
					Help:      "Total number of state transitions by from/to state",
				},
				[]string{"from", "to"},
			),
			CurrentState: promauto.NewGauge(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: "engine",
					Name:      "current_state",
					Help:      "Current engine state (0=Idle, 1=Scanning, 2=LiveTrading, 3=PaperTrading, 4=Backtesting, 5=Optimizing, 6=Halted)",
				},
			),
			EngineUptime: promauto.NewGauge(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: "engine",
					Name:      "uptime_seconds",
					Help:      "Engine uptime in seconds",
				},
			),

			// NTP metrics
			NTPDrift: promauto.NewGauge(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: "ntp",
					Name:      "drift_milliseconds",
					Help:      "Current NTP time drift in milliseconds",
				},
			),
			NTPSyncErrors: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "ntp",
					Name:      "sync_errors_total",
					Help:      "Total number of NTP sync errors",
				},
			),
			NTPLastSync: promauto.NewGauge(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: "ntp",
					Name:      "last_sync_timestamp",
					Help:      "Unix timestamp of last successful NTP sync",
				},
			),

			// Order metrics
			OrdersCreated: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "orders",
					Name:      "created_total",
					Help:      "Total number of orders created by symbol",
				},
				[]string{"symbol"},
			),
			OrdersSubmitted: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "orders",
					Name:      "submitted_total",
					Help:      "Total number of orders submitted to exchange by symbol",
				},
				[]string{"symbol"},
			),
			OrdersFilled: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "orders",
					Name:      "filled_total",
					Help:      "Total number of orders filled by symbol",
				},
				[]string{"symbol"},
			),
			OrdersCanceled: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "orders",
					Name:      "canceled_total",
					Help:      "Total number of orders canceled by symbol",
				},
				[]string{"symbol"},
			),
			OrdersRejected: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "orders",
					Name:      "rejected_total",
					Help:      "Total number of orders rejected by symbol",
				},
				[]string{"symbol"},
			),

			// System metrics
			GoroutineCount: promauto.NewGauge(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: "system",
					Name:      "goroutines_count",
					Help:      "Current number of goroutines",
				},
			),
			MemoryUsage: promauto.NewGauge(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: "system",
					Name:      "memory_bytes",
					Help:      "Current memory usage in bytes",
				},
			),
			CPUUsage: promauto.NewGauge(
				prometheus.GaugeOpts{
					Namespace: namespace,
					Subsystem: "system",
					Name:      "cpu_usage_percent",
					Help:      "Current CPU usage percentage",
				},
			),

			// Storage metrics
			CriticalWrites: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "storage",
					Name:      "critical_writes_total",
					Help:      "Total number of critical database writes",
				},
			),
			AnalyticsWrites: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "storage",
					Name:      "analytics_writes_total",
					Help:      "Total number of analytics database writes",
				},
			),
			AnalyticsDrops: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "storage",
					Name:      "analytics_drops_total",
					Help:      "Total number of analytics events dropped due to queue overflow",
				},
			),

			// Connector metrics
			ConnectorErrors: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "connector",
					Name:      "errors_total",
					Help:      "Total number of connector errors by type",
				},
				[]string{"error_type"},
			),
			ConnectorLatency: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: namespace,
					Subsystem: "connector",
					Name:      "request_duration_seconds",
					Help:      "Connector request latency by operation",
					Buckets:   prometheus.DefBuckets,
				},
				[]string{"operation"},
			),
			WebSocketMessages: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "connector",
					Name:      "websocket_messages_total",
					Help:      "Total number of WebSocket messages by type",
				},
				[]string{"message_type"},
			),
		}
	})

	return instance
}

// UpdateSystemMetrics updates system-level metrics (goroutines, memory).
func (m *Metrics) UpdateSystemMetrics() {
	m.GoroutineCount.Set(float64(runtime.NumGoroutine()))

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	m.MemoryUsage.Set(float64(memStats.Alloc))
}

// RecordEventDropped increments the dropped events counter.
func (m *Metrics) RecordEventDropped(count uint64) {
	m.EventsDropped.Add(float64(count))
}

// RecordStorageStats updates storage metrics.
func (m *Metrics) RecordStorageStats(criticalWrites, analyticsWrites, analyticsDrops int64) {
	m.CriticalWrites.Add(float64(criticalWrites))
	m.AnalyticsWrites.Add(float64(analyticsWrites))
	m.AnalyticsDrops.Add(float64(analyticsDrops))
}
