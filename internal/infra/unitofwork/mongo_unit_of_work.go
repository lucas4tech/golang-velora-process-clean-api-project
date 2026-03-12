package unitofwork

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"

	orderrepo "rankmyapp/internal/domain/order/repository"
	outboxrepo "rankmyapp/internal/domain/outbox/repository"
)

type mongoUnitOfWork struct {
	client            *mongo.Client
	orderRepoFactory  func(ctx context.Context) orderrepo.OrderRepository
	outboxRepoFactory func(ctx context.Context) outboxrepo.OutboxRepository
}

func NewMongoUnitOfWork(
	client *mongo.Client,
	orderRepoFactory func(ctx context.Context) orderrepo.OrderRepository,
	outboxRepoFactory func(ctx context.Context) outboxrepo.OutboxRepository,
) UnitOfWork {
	return &mongoUnitOfWork{
		client:            client,
		orderRepoFactory:  orderRepoFactory,
		outboxRepoFactory: outboxRepoFactory,
	}
}

func (u *mongoUnitOfWork) Execute(
	ctx context.Context,
	fn func(ctx context.Context, or orderrepo.OrderRepository, ob outboxrepo.OutboxRepository) error,
) error {
	session, err := u.client.StartSession()
	if err != nil {
		return fmt.Errorf("uow: start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, txErr := session.WithTransaction(ctx, func(sessCtx context.Context) (interface{}, error) {
		or := u.orderRepoFactory(sessCtx)
		ob := u.outboxRepoFactory(sessCtx)
		return nil, fn(sessCtx, or, ob)
	})
	return txErr
}
