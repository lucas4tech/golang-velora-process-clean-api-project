package repository

import (
	"context"

	"rankmyapp/internal/domain/outbox/entity"
)

type OutboxRepository interface {
	Save(ctx context.Context, msg *entity.OutboxMessage) error
	FindPending(ctx context.Context, limit int) ([]*entity.OutboxMessage, error)
	UpdateStatus(ctx context.Context, msg *entity.OutboxMessage) error
}
