package unitofwork

import (
	"context"

	orderrepo "rankmyapp/internal/domain/order/repository"
	outboxrepo "rankmyapp/internal/domain/outbox/repository"
)

type UnitOfWork interface {
	Execute(ctx context.Context, fn func(ctx context.Context, or orderrepo.OrderRepository, ob outboxrepo.OutboxRepository) error) error
}
