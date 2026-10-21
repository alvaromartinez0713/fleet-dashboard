package messaging

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/fleet-platform/backend/internal/worker/adapters/inbound/parser"
	"github.com/fleet-platform/backend/internal/worker/domain"
	"github.com/fleet-platform/backend/internal/worker/ports"
)

// Runner orchestrates Kafka message consumption.
//
// Responsibilities (focused):
// - Consume messages from Kafka (loop + fetch)
// - Parse + execute processor
// - Delegate error handling to ErrorStrategy
// - Commit offsets based on strategy decision
//
// Clean Architecture:
// - Depends on ports.PositionProcessor (interface)
// - Delegates error decisions to ErrorStrategy (SRP)
type Runner struct {
	queue          MessageQueue
	parser         *parser.PositionEventParser
	processor      ports.PositionProcessor
	circuitBreaker ports.CircuitBreaker
	errorStrategy  *ErrorStrategy
	clock          ports.Clock
	logger         ports.Logger
	shutdown       chan struct{}
	wg             sync.WaitGroup
	running        bool
	mu             sync.RWMutex
	shutdownTimeout time.Duration
	reconnectDelay time.Duration
}

// NewRunner creates a runner.
func NewRunner(
	queue MessageQueue,
	parser *parser.PositionEventParser,
	processor ports.PositionProcessor,
	circuitBreaker ports.CircuitBreaker,
	errorStrategy *ErrorStrategy,
	clock ports.Clock,
	logger ports.Logger,
	shutdownTimeout time.Duration,
	reconnectDelay time.Duration,
) *Runner {
	if reconnectDelay == 0 {
		reconnectDelay = 5 * time.Second
	}
	return &Runner{
		queue:           queue,
		parser:          parser,
		processor:       processor,
		circuitBreaker:  circuitBreaker,
		errorStrategy:   errorStrategy,
		clock:           clock,
		logger:          logger,
		shutdown:        make(chan struct{}),
		shutdownTimeout: shutdownTimeout,
		reconnectDelay:  reconnectDelay,
	}
}

func (r *Runner) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		return errors.New("already running")
	}
	r.running = true
	r.wg.Add(1)
	go r.runLoop(ctx)
	return nil
}

func (r *Runner) runLoop(ctx context.Context) {
	defer r.wg.Done()
	r.logger.Info("consumption loop started", ports.BuildFields()...)
	for {
		select {
		case <-r.shutdown:
			r.logger.Info("shutdown signal received", ports.BuildFields()...)
			return
		case <-ctx.Done():
			r.logger.Info("context cancelled", ports.BuildFields()...)
			return
		default:
			r.processMessage(ctx)
		}
	}
}

func (r *Runner) processMessage(ctx context.Context) {
	// Check circuit breaker before fetching
	if r.circuitBreaker.State() == ports.StateOpen {
		r.logger.Warn("circuit breaker is open, pausing consumption", ports.BuildFields()...)
		time.Sleep(5 * time.Second)
		return
	}

	// Fetch message from Kafka
	msg, err := r.queue.FetchMessage(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		r.logger.Warn("fetch failed", ports.BuildFields("error", err.Error())...)
		time.Sleep(r.reconnectDelay)
		return
	}

	// Generate trace ID and get current time for parsing and persistence
	traceID, _ := domain.NewTraceID(nil)
	now := r.clock.Now()

	// Parse + validate with both validation time and persistence time
	position, err := r.parser.ParseAndValidate(msg.Value, now, now)
	if err != nil {
		// Delegate error handling to strategy
		action := r.errorStrategy.HandleParseError(ctx, err, traceID, msg.Value)
		r.applyAction(ctx, action, msg)
		return
	}

	// Execute processor
	result, err := r.processor.Execute(ctx, position)
	if err != nil {
		// Delegate error handling to strategy
		action := r.errorStrategy.HandleProcessorError(ctx, err, traceID, position)
		r.applyAction(ctx, action, msg)
		return
	}

	// Success - commit offset
	r.logger.Info("position processed successfully",
		ports.Field{Key: "event_id", Value: result.EventID().String()},
		ports.Field{Key: "trace_id", Value: traceID.String()},
	)
	r.commitOffset(ctx, msg)
}

// applyAction executes the action decided by ErrorStrategy.
func (r *Runner) applyAction(ctx context.Context, action ErrorAction, msg Message) {
	if action.ShouldPause {
		time.Sleep(action.PauseDuration)
	}
	if action.ShouldCommit {
		r.commitOffset(ctx, msg)
	}
}

func (r *Runner) commitOffset(ctx context.Context, msg Message) {
	commitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := r.queue.CommitOffset(commitCtx, msg); err != nil {
		r.logger.Warn("commit failed", ports.BuildFields("error", err.Error())...)
	}
}

func (r *Runner) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.running {
		return
	}
	close(r.shutdown)
	r.wg.Wait()
	r.running = false
}
