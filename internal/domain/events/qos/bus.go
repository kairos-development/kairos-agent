package qos

import (
	"context"
	"sync"
	"time"

	"github.com/kairos-development/kairos-agent/internal/domain/events"
	"github.com/sirupsen/logrus"
)

type QoSLevel string

const (
	QoS0_AtMostOnce  QoSLevel = "0" // Fire and forget, may drop
	QoS1_AtLeastOnce QoSLevel = "1" // Guaranteed delivery, may duplicate
	QoS2_ExactlyOnce QoSLevel = "2" // Guaranteed exactly once (not implemented, falls back to QoS1)
)

type Config struct {
	QoSLevel      QoSLevel
	MaxRetries    int
	RetryInterval time.Duration
	BufferSize    int
}

func DefaultConfig() Config {
	return Config{
		QoSLevel:      QoS1_AtLeastOnce,
		MaxRetries:    3,
		RetryInterval: 1 * time.Second,
		BufferSize:    1000,
	}
}

type Bus struct {
	publisher *events.Publisher
	logger    *logrus.Logger
	cfg       Config
	ctx       context.Context
	cancel    context.CancelFunc

	asyncCh chan events.Event
	stopCh  chan struct{}
	wg      sync.WaitGroup

	mu           sync.RWMutex
	handlers     map[events.EventType][]qosHandler
	publishedCnt int64
	droppedCnt   int64
	retriedCnt   int64
	closed       bool
}

type qosHandler struct {
	handler events.Handler
	qos     QoSLevel
}

func NewBus(cfg Config, logger *logrus.Logger) *Bus {
	if logger == nil {
		logger = logrus.New()
	}

	ctx, cancel := context.WithCancel(context.Background())
	bus := &Bus{
		publisher: events.NewPublisher(),
		logger:    logger,
		cfg:       cfg,
		ctx:       ctx,
		cancel:    cancel,
		asyncCh:   make(chan events.Event, cfg.BufferSize),
		stopCh:    make(chan struct{}),
		handlers:  make(map[events.EventType][]qosHandler),
	}

	bus.wg.Add(1)
	go bus.asyncWorker()

	logger.WithField("qos_level", cfg.QoSLevel).Info("QoS event bus initialized")
	return bus
}

func (b *Bus) Subscribe(eventType events.EventType, handler events.Handler, qos QoSLevel) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], qosHandler{handler: handler, qos: qos})
}

func (b *Bus) SubscribeFunc(eventType events.EventType, fn func(context.Context, events.Event) error, qos QoSLevel) {
	b.Subscribe(eventType, events.HandlerFunc(fn), qos)
}

func (b *Bus) Publish(ctx context.Context, event events.Event) error {
	b.mu.RLock()
	handlers := b.handlers[event.Type()]
	b.mu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}

	var errs []error
	for _, h := range handlers {
		if err := b.deliver(ctx, event, h); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (b *Bus) deliver(ctx context.Context, event events.Event, h qosHandler) error {
	switch h.qos {
	case QoS0_AtMostOnce:
		return b.deliverQoS0(ctx, event, h.handler)
	case QoS1_AtLeastOnce:
		return b.deliverQoS1(ctx, event, h.handler)
	default:
		return b.deliverQoS1(ctx, event, h.handler)
	}
}

func (b *Bus) deliverQoS0(ctx context.Context, event events.Event, handler events.Handler) error {
	b.mu.Lock()
	b.publishedCnt++
	b.mu.Unlock()

	go func() {
		if err := handler.Handle(ctx, event); err != nil {
			b.logger.WithError(err).WithField("event", event.Type()).Warn("QoS0 handler error (ignored)")
		}
	}()
	return nil
}

func (b *Bus) deliverQoS1(ctx context.Context, event events.Event, handler events.Handler) error {
	b.mu.Lock()
	b.publishedCnt++
	b.mu.Unlock()

	var err error
	for i := 0; i <= b.cfg.MaxRetries; i++ {
		err = handler.Handle(ctx, event)
		if err == nil {
			return nil
		}
		b.mu.Lock()
		b.retriedCnt++
		b.mu.Unlock()
		b.logger.WithError(err).WithFields(logrus.Fields{
			"event":   event.Type(),
			"attempt": i + 1,
			"max":     b.cfg.MaxRetries + 1,
		}).Warn("QoS1 retry")

		if i < b.cfg.MaxRetries {
			time.Sleep(b.cfg.RetryInterval)
		}
	}
	return err
}

func (b *Bus) PublishAsync(ctx context.Context, event events.Event) {
	select {
	case b.asyncCh <- event:
		b.mu.Lock()
		b.publishedCnt++
		b.mu.Unlock()
	default:
		b.mu.Lock()
		b.droppedCnt++
		b.mu.Unlock()
		b.logger.WithField("event", event.Type()).Warn("QoS event dropped (buffer full)")
	}
}

func (b *Bus) asyncWorker() {
	defer b.wg.Done()

	for {
		select {
		case event := <-b.asyncCh:
			if err := b.Publish(b.ctx, event); err != nil {
				b.logger.WithError(err).WithField("event", event.Type()).Error("Async publish error")
			}
		case <-b.stopCh:
			return
		}
	}
}

func (b *Bus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()

	b.cancel()
	close(b.stopCh)
	b.wg.Wait()
	return nil
}

func (b *Bus) Stats() (published, dropped, retried int64) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.publishedCnt, b.droppedCnt, b.retriedCnt
}
