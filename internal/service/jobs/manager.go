package jobs

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Priority defines job priority classes.
type Priority int

const (
	// PriorityCritical for risk management, order execution, position updates.
	PriorityCritical Priority = iota
	// PriorityLive for live trading operations.
	PriorityLive
	// PriorityBackground for analytics, logging, metrics.
	PriorityBackground
	// PriorityAnalytics for historical data processing.
	PriorityAnalytics
)

func (p Priority) String() string {
	switch p {
	case PriorityCritical:
		return "critical"
	case PriorityLive:
		return "live"
	case PriorityBackground:
		return "background"
	case PriorityAnalytics:
		return "analytics"
	default:
		return "unknown"
	}
}

// Job represents a unit of work to be executed.
type Job struct {
	ID          string
	Name        string
	Priority    Priority
	Fn          func(context.Context) error
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	Error       error
}

// Manager manages job execution with priority classes and resource limits.
type Manager struct {
	mu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Job queues by priority
	criticalQueue   chan *Job
	liveQueue       chan *Job
	backgroundQueue chan *Job
	analyticsQueue  chan *Job

	// Active jobs tracking
	activeJobs map[string]*Job

	// Worker pools
	criticalWorkers   int
	liveWorkers       int
	backgroundWorkers int
	analyticsWorkers  int

	// Metrics
	totalJobsProcessed   int
	totalJobsFailed      int
	jobProcessingTimeSum time.Duration
	stopped              bool

	logger *logrus.Logger
}

// Config contains job manager configuration.
type Config struct {
	CriticalWorkers   int
	LiveWorkers       int
	BackgroundWorkers int
	AnalyticsWorkers  int
	QueueSize         int
}

// DefaultConfig returns default job manager configuration.
func DefaultConfig() *Config {
	return &Config{
		CriticalWorkers:   4, // High priority for risk/execution
		LiveWorkers:       2, // Live trading operations
		BackgroundWorkers: 2, // Background tasks
		AnalyticsWorkers:  1, // Analytics can be throttled
		QueueSize:         100,
	}
}

// NewManager creates a new job manager.
func NewManager(ctx context.Context, config *Config, logger *logrus.Logger) *Manager {
	if config == nil {
		config = DefaultConfig()
	}

	if logger == nil {
		logger = logrus.New()
	}

	ctx, cancel := context.WithCancel(ctx)

	m := &Manager{
		ctx:               ctx,
		cancel:            cancel,
		criticalQueue:     make(chan *Job, config.QueueSize),
		liveQueue:         make(chan *Job, config.QueueSize),
		backgroundQueue:   make(chan *Job, config.QueueSize),
		analyticsQueue:    make(chan *Job, config.QueueSize),
		activeJobs:        make(map[string]*Job),
		criticalWorkers:   config.CriticalWorkers,
		liveWorkers:       config.LiveWorkers,
		backgroundWorkers: config.BackgroundWorkers,
		analyticsWorkers:  config.AnalyticsWorkers,
		logger:            logger,
	}

	return m
}

// Start starts the job manager and worker pools.
func (m *Manager) Start() {
	m.logger.WithFields(logrus.Fields{
		"critical_workers":   m.criticalWorkers,
		"live_workers":       m.liveWorkers,
		"background_workers": m.backgroundWorkers,
		"analytics_workers":  m.analyticsWorkers,
	}).Info("Starting job manager")

	// Start critical workers
	for i := 0; i < m.criticalWorkers; i++ {
		m.wg.Add(1)
		go m.worker(PriorityCritical, m.criticalQueue)
	}

	// Start live workers
	for i := 0; i < m.liveWorkers; i++ {
		m.wg.Add(1)
		go m.worker(PriorityLive, m.liveQueue)
	}

	// Start background workers
	for i := 0; i < m.backgroundWorkers; i++ {
		m.wg.Add(1)
		go m.worker(PriorityBackground, m.backgroundQueue)
	}

	// Start analytics workers
	for i := 0; i < m.analyticsWorkers; i++ {
		m.wg.Add(1)
		go m.worker(PriorityAnalytics, m.analyticsQueue)
	}

	m.logger.Info("Job manager started")
}

// Submit submits a job for execution.
func (m *Manager) Submit(job *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return fmt.Errorf("job manager stopped")
	}

	if job == nil {
		return fmt.Errorf("job is nil")
	}

	if job.ID == "" {
		job.ID = fmt.Sprintf("job_%d", time.Now().UnixNano())
	}

	job.CreatedAt = time.Now().UTC()

	// Select queue based on priority
	var queue chan *Job
	switch job.Priority {
	case PriorityCritical:
		queue = m.criticalQueue
	case PriorityLive:
		queue = m.liveQueue
	case PriorityBackground:
		queue = m.backgroundQueue
	case PriorityAnalytics:
		queue = m.analyticsQueue
	default:
		return fmt.Errorf("unknown priority: %d", job.Priority)
	}

	// Non-blocking send
	select {
	case queue <- job:
		m.activeJobs[job.ID] = job
		m.logger.WithFields(logrus.Fields{
			"job_id":   job.ID,
			"name":     job.Name,
			"priority": job.Priority.String(),
		}).Debug("Job submitted")
		return nil
	default:
		return fmt.Errorf("queue full for priority %s", job.Priority.String())
	}
}

// SubmitFunc is a convenience method to submit a function as a job.
func (m *Manager) SubmitFunc(name string, priority Priority, fn func(context.Context) error) error {
	job := &Job{
		Name:     name,
		Priority: priority,
		Fn:       fn,
	}
	return m.Submit(job)
}

// Stop stops the job manager and waits for all workers to finish.
func (m *Manager) Stop() {
	m.logger.Info("Stopping job manager")

	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return
	}
	m.stopped = true

	m.cancel()

	// Close queues
	close(m.criticalQueue)
	close(m.liveQueue)
	close(m.backgroundQueue)
	close(m.analyticsQueue)
	m.mu.Unlock()

	// Wait for workers
	m.wg.Wait()

	m.logger.WithFields(logrus.Fields{
		"total_processed": m.totalJobsProcessed,
		"total_failed":    m.totalJobsFailed,
	}).Info("Job manager stopped")
}

// GetStats returns job manager statistics.
func (m *Manager) GetStats() *Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgProcessingTime := time.Duration(0)
	if m.totalJobsProcessed > 0 {
		avgProcessingTime = m.jobProcessingTimeSum / time.Duration(m.totalJobsProcessed)
	}

	return &Stats{
		TotalProcessed:      m.totalJobsProcessed,
		TotalFailed:         m.totalJobsFailed,
		ActiveJobs:          len(m.activeJobs),
		CriticalQueueSize:   len(m.criticalQueue),
		LiveQueueSize:       len(m.liveQueue),
		BackgroundQueueSize: len(m.backgroundQueue),
		AnalyticsQueueSize:  len(m.analyticsQueue),
		AvgProcessingTime:   avgProcessingTime,
	}
}

func (m *Manager) worker(priority Priority, queue chan *Job) {
	defer m.wg.Done()

	m.logger.WithField("priority", priority.String()).Debug("Worker started")

	for {
		select {
		case <-m.ctx.Done():
			m.logger.WithField("priority", priority.String()).Debug("Worker stopped")
			return
		case job, ok := <-queue:
			if !ok {
				m.logger.WithField("priority", priority.String()).Debug("Worker queue closed")
				return
			}

			m.executeJob(job)
		}
	}
}

func (m *Manager) executeJob(job *Job) {
	startTime := time.Now().UTC()
	job.StartedAt = &startTime

	m.logger.WithFields(logrus.Fields{
		"job_id":   job.ID,
		"name":     job.Name,
		"priority": job.Priority.String(),
	}).Debug("Executing job")

	// Execute job function. A single bad job must not crash the worker pool.
	err := m.runJob(job)

	completedTime := time.Now().UTC()
	job.CompletedAt = &completedTime
	job.Error = err

	duration := completedTime.Sub(startTime)

	// Update metrics
	m.mu.Lock()
	m.totalJobsProcessed++
	m.jobProcessingTimeSum += duration
	if err != nil {
		m.totalJobsFailed++
	}
	delete(m.activeJobs, job.ID)
	m.mu.Unlock()

	if err != nil {
		m.logger.WithFields(logrus.Fields{
			"job_id":      job.ID,
			"name":        job.Name,
			"priority":    job.Priority.String(),
			"duration_ms": duration.Milliseconds(),
		}).WithError(err).Error("Job failed")
	} else {
		m.logger.WithFields(logrus.Fields{
			"job_id":      job.ID,
			"name":        job.Name,
			"priority":    job.Priority.String(),
			"duration_ms": duration.Milliseconds(),
		}).Debug("Job completed")
	}
}

func (m *Manager) runJob(job *Job) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("job panic: %v", r)
			m.logger.WithFields(logrus.Fields{
				"job_id": job.ID,
				"name":   job.Name,
				"panic":  r,
				"stack":  string(debug.Stack()),
			}).Error("Job panic recovered")
		}
	}()

	if job.Fn == nil {
		return fmt.Errorf("job function is nil")
	}

	return job.Fn(m.ctx)
}

// Stats contains job manager statistics.
type Stats struct {
	TotalProcessed      int
	TotalFailed         int
	ActiveJobs          int
	CriticalQueueSize   int
	LiveQueueSize       int
	BackgroundQueueSize int
	AnalyticsQueueSize  int
	AvgProcessingTime   time.Duration
}
