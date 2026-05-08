package jobs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	config := DefaultConfig()
	ctx := context.Background()

	manager := NewManager(ctx, config, nil)

	if manager == nil {
		t.Fatal("expected manager to be created")
	}

	if manager.criticalWorkers != config.CriticalWorkers {
		t.Errorf("expected %d critical workers, got %d", config.CriticalWorkers, manager.criticalWorkers)
	}

	if manager.liveWorkers != config.LiveWorkers {
		t.Errorf("expected %d live workers, got %d", config.LiveWorkers, manager.liveWorkers)
	}

	if manager.backgroundWorkers != config.BackgroundWorkers {
		t.Errorf("expected %d background workers, got %d", config.BackgroundWorkers, manager.backgroundWorkers)
	}

	if manager.analyticsWorkers != config.AnalyticsWorkers {
		t.Errorf("expected %d analytics workers, got %d", config.AnalyticsWorkers, manager.analyticsWorkers)
	}
}

func TestManager_SubmitAndExecute(t *testing.T) {
	config := DefaultConfig()
	ctx := context.Background()

	manager := NewManager(ctx, config, nil)
	manager.Start()
	defer manager.Stop()

	var mu sync.Mutex
	executed := false
	job := &Job{
		Name:     "test_job",
		Priority: PriorityCritical,
		Fn: func(ctx context.Context) error {
			mu.Lock()
			executed = true
			mu.Unlock()
			return nil
		},
	}

	err := manager.Submit(job)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Wait for job to execute
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !executed {
		t.Error("expected job to be executed")
	}

	stats := manager.GetStats()
	if stats.TotalProcessed != 1 {
		t.Errorf("expected 1 job processed, got %d", stats.TotalProcessed)
	}
}

func TestManager_SubmitFunc(t *testing.T) {
	config := DefaultConfig()
	ctx := context.Background()

	manager := NewManager(ctx, config, nil)
	manager.Start()
	defer manager.Stop()

	var mu sync.Mutex
	executed := false
	err := manager.SubmitFunc("test_func", PriorityLive, func(ctx context.Context) error {
		mu.Lock()
		executed = true
		mu.Unlock()
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !executed {
		t.Error("expected function to be executed")
	}
}

func TestManager_JobWithError(t *testing.T) {
	config := DefaultConfig()
	ctx := context.Background()

	manager := NewManager(ctx, config, nil)
	manager.Start()
	defer manager.Stop()

	expectedError := errors.New("job failed")
	job := &Job{
		Name:     "failing_job",
		Priority: PriorityCritical,
		Fn: func(ctx context.Context) error {
			return expectedError
		},
	}

	err := manager.Submit(job)
	if err != nil {
		t.Fatalf("expected no error on submit, got %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	stats := manager.GetStats()
	if stats.TotalFailed != 1 {
		t.Errorf("expected 1 failed job, got %d", stats.TotalFailed)
	}
}

func TestManager_PriorityQueues(t *testing.T) {
	config := DefaultConfig()
	ctx := context.Background()

	manager := NewManager(ctx, config, nil)
	manager.Start()
	defer manager.Stop()

	priorities := []Priority{
		PriorityCritical,
		PriorityLive,
		PriorityBackground,
		PriorityAnalytics,
	}

	for _, priority := range priorities {
		job := &Job{
			Name:     "test_job_" + priority.String(),
			Priority: priority,
			Fn: func(ctx context.Context) error {
				return nil
			},
		}

		err := manager.Submit(job)
		if err != nil {
			t.Fatalf("expected no error for priority %s, got %v", priority.String(), err)
		}
	}

	time.Sleep(200 * time.Millisecond)

	stats := manager.GetStats()
	if stats.TotalProcessed != 4 {
		t.Errorf("expected 4 jobs processed, got %d", stats.TotalProcessed)
	}
}

func TestManager_QueueFull(t *testing.T) {
	config := &Config{
		CriticalWorkers:   1,
		LiveWorkers:       1,
		BackgroundWorkers: 1,
		AnalyticsWorkers:  1,
		QueueSize:         2,
	}
	ctx := context.Background()

	manager := NewManager(ctx, config, nil)
	// Don't start workers to keep queue full

	// Fill queue
	for i := 0; i < 2; i++ {
		job := &Job{
			Name:     "job",
			Priority: PriorityCritical,
			Fn: func(ctx context.Context) error {
				return nil
			},
		}
		err := manager.Submit(job)
		if err != nil {
			t.Fatalf("expected no error filling queue, got %v", err)
		}
	}

	// Try to submit when queue is full
	job := &Job{
		Name:     "overflow_job",
		Priority: PriorityCritical,
		Fn: func(ctx context.Context) error {
			return nil
		},
	}

	err := manager.Submit(job)
	if err == nil {
		t.Fatal("expected error when queue is full")
	}
}

func TestManager_Stop(t *testing.T) {
	config := DefaultConfig()
	ctx := context.Background()

	manager := NewManager(ctx, config, nil)
	manager.Start()

	// Submit some jobs
	for i := 0; i < 5; i++ {
		_ = manager.SubmitFunc("job", PriorityBackground, func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	}

	// Stop should wait for jobs to complete
	manager.Stop()

	stats := manager.GetStats()
	if stats.TotalProcessed == 0 {
		t.Error("expected some jobs to be processed before stop")
	}
}

func TestManager_ContextCancellation(t *testing.T) {
	config := DefaultConfig()
	ctx, cancel := context.WithCancel(context.Background())

	manager := NewManager(ctx, config, nil)
	manager.Start()

	jobStarted := make(chan bool)
	var mu sync.Mutex
	jobCanceled := false

	job := &Job{
		Name:     "long_job",
		Priority: PriorityCritical,
		Fn: func(ctx context.Context) error {
			jobStarted <- true
			select {
			case <-ctx.Done():
				mu.Lock()
				jobCanceled = true
				mu.Unlock()
				return ctx.Err()
			case <-time.After(5 * time.Second):
				return nil
			}
		},
	}

	err := manager.Submit(job)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Wait for job to start
	<-jobStarted

	// Cancel context
	cancel()

	// Wait for cancellation to propagate
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !jobCanceled {
		t.Error("expected job to be canceled")
	}
}

func TestManager_GetStats(t *testing.T) {
	config := DefaultConfig()
	ctx := context.Background()

	manager := NewManager(ctx, config, nil)
	manager.Start()
	defer manager.Stop()

	// Submit multiple jobs
	for i := 0; i < 10; i++ {
		_ = manager.SubmitFunc("job", PriorityCritical, func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	}

	time.Sleep(200 * time.Millisecond)

	stats := manager.GetStats()

	if stats.TotalProcessed == 0 {
		t.Error("expected some jobs to be processed")
	}

	if stats.AvgProcessingTime == 0 {
		t.Error("expected non-zero average processing time")
	}
}

func TestManager_JobIDGeneration(t *testing.T) {
	config := DefaultConfig()
	ctx := context.Background()

	manager := NewManager(ctx, config, nil)

	job := &Job{
		Name:     "test_job",
		Priority: PriorityCritical,
		Fn: func(ctx context.Context) error {
			return nil
		},
	}

	err := manager.Submit(job)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if job.ID == "" {
		t.Error("expected job ID to be generated")
	}

	if job.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestManager_UnknownPriority(t *testing.T) {
	config := DefaultConfig()
	ctx := context.Background()

	manager := NewManager(ctx, config, nil)

	job := &Job{
		Name:     "test_job",
		Priority: Priority(999), // Invalid priority
		Fn: func(ctx context.Context) error {
			return nil
		},
	}

	err := manager.Submit(job)
	if err == nil {
		t.Fatal("expected error for unknown priority")
	}
}

func TestPriority_String(t *testing.T) {
	tests := []struct {
		priority Priority
		expected string
	}{
		{PriorityCritical, "critical"},
		{PriorityLive, "live"},
		{PriorityBackground, "background"},
		{PriorityAnalytics, "analytics"},
		{Priority(999), "unknown"},
	}

	for _, tt := range tests {
		result := tt.priority.String()
		if result != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, result)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.CriticalWorkers != 4 {
		t.Errorf("expected 4 critical workers, got %d", config.CriticalWorkers)
	}

	if config.LiveWorkers != 2 {
		t.Errorf("expected 2 live workers, got %d", config.LiveWorkers)
	}

	if config.BackgroundWorkers != 2 {
		t.Errorf("expected 2 background workers, got %d", config.BackgroundWorkers)
	}

	if config.AnalyticsWorkers != 1 {
		t.Errorf("expected 1 analytics worker, got %d", config.AnalyticsWorkers)
	}

	if config.QueueSize != 100 {
		t.Errorf("expected queue size 100, got %d", config.QueueSize)
	}
}

func TestManager_MultipleJobsSequential(t *testing.T) {
	config := DefaultConfig()
	ctx := context.Background()

	manager := NewManager(ctx, config, nil)
	manager.Start()
	defer manager.Stop()

	executionOrder := make([]int, 0)
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		index := i
		job := &Job{
			Name:     "sequential_job",
			Priority: PriorityCritical,
			Fn: func(ctx context.Context) error {
				mu.Lock()
				executionOrder = append(executionOrder, index)
				mu.Unlock()
				return nil
			},
		}
		err := manager.Submit(job)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(executionOrder) != 5 {
		t.Errorf("expected 5 jobs executed, got %d", len(executionOrder))
	}
}

func TestManager_NilConfig(t *testing.T) {
	ctx := context.Background()
	manager := NewManager(ctx, nil, nil)

	if manager == nil {
		t.Fatal("expected manager to be created with nil config")
	}

	// Should use default config
	if manager.criticalWorkers != 4 {
		t.Errorf("expected default 4 critical workers, got %d", manager.criticalWorkers)
	}
}
