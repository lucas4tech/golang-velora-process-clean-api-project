package outboxworker

import (
	"context"
	"time"

	"go.elastic.co/apm/v2"
	"go.uber.org/zap"

	outboxrepo "rankmyapp/internal/domain/outbox/repository"
	"rankmyapp/pkg/logger"
)

const (
	defaultBatchSize = 50
	defaultInterval  = 5 * time.Second
	maxAttempts      = 5
)

type MessagePublisher interface {
	Publish(ctx context.Context, eventName string, payload []byte) error
}

type Worker struct {
	repo      outboxrepo.OutboxRepository
	publisher MessagePublisher
	interval  time.Duration
	batchSize int
}

func New(repo outboxrepo.OutboxRepository, publisher MessagePublisher) *Worker {
	return &Worker{
		repo:      repo,
		publisher: publisher,
		interval:  defaultInterval,
		batchSize: defaultBatchSize,
	}
}

func (w *Worker) Start(ctx context.Context) {
	log := logger.Get()
	log.Info("outbox worker: starting", zap.Duration("interval", w.interval))

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("outbox worker: shutting down")
			return
		case <-ticker.C:
			w.ProcessOnce(ctx)
		}
	}
}

func (w *Worker) ProcessOnce(ctx context.Context) {
	tx := apm.DefaultTracer().StartTransaction("outbox.process", "background")
	defer tx.End()
	ctx = apm.ContextWithTransaction(ctx, tx)

	log := logger.Get()

	msgs, err := w.repo.FindPending(ctx, w.batchSize)
	if err != nil {
		log.Error("outbox worker: failed to fetch messages", zap.Error(err))
		return
	}

	for _, msg := range msgs {
		if msg.Attempts >= maxAttempts {
			msg.MarkFailed()
			_ = w.repo.UpdateStatus(ctx, msg)
			log.Warn("outbox worker: message marked as failed (max attempts reached)",
				zap.String("id", msg.ID),
				zap.String("event", msg.EventName),
			)
			continue
		}

		if err := w.publisher.Publish(ctx, msg.EventName, msg.Payload); err != nil {
			msg.IncrementAttempt()
			_ = w.repo.UpdateStatus(ctx, msg)
			log.Error("outbox worker: publish failed",
				zap.String("id", msg.ID),
				zap.String("event", msg.EventName),
				zap.Error(err),
			)
			continue
		}

		msg.MarkPublished()
		if err := w.repo.UpdateStatus(ctx, msg); err != nil {
			log.Error("outbox worker: failed to mark as published",
				zap.String("id", msg.ID),
				zap.Error(err),
			)
		}
	}
}
