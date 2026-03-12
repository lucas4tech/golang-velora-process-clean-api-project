package usecase_test

import (
	"context"

	"github.com/stretchr/testify/mock"

	"rankmyapp/internal/domain/order/entity"
	orderrepo "rankmyapp/internal/domain/order/repository"
	outboxentity "rankmyapp/internal/domain/outbox/entity"
	outboxrepo "rankmyapp/internal/domain/outbox/repository"
)

type mockOrderRepository struct{ mock.Mock }

func (m *mockOrderRepository) Save(ctx context.Context, o *entity.Order) error {
	return m.Called(ctx, o).Error(0)
}
func (m *mockOrderRepository) FindByID(ctx context.Context, id string) (*entity.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Order), args.Error(1)
}
func (m *mockOrderRepository) FindAll(ctx context.Context, f orderrepo.OrderFilter) ([]*entity.Order, error) {
	args := m.Called(ctx, f)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Order), args.Error(1)
}
func (m *mockOrderRepository) Update(ctx context.Context, o *entity.Order) error {
	return m.Called(ctx, o).Error(0)
}

type mockOutboxRepository struct{ mock.Mock }

func (m *mockOutboxRepository) Save(ctx context.Context, msg *outboxentity.OutboxMessage) error {
	return m.Called(ctx, msg).Error(0)
}
func (m *mockOutboxRepository) FindPending(ctx context.Context, limit int) ([]*outboxentity.OutboxMessage, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*outboxentity.OutboxMessage), args.Error(1)
}
func (m *mockOutboxRepository) UpdateStatus(ctx context.Context, msg *outboxentity.OutboxMessage) error {
	return m.Called(ctx, msg).Error(0)
}

type mockUnitOfWork struct {
	mock.Mock
	orderRepo  orderrepo.OrderRepository
	outboxRepo outboxrepo.OutboxRepository
}

func (m *mockUnitOfWork) Execute(
	ctx context.Context,
	fn func(ctx context.Context, or orderrepo.OrderRepository, ob outboxrepo.OutboxRepository) error,
) error {
	if m.orderRepo != nil && m.outboxRepo != nil {
		return fn(ctx, m.orderRepo, m.outboxRepo)
	}
	args := m.Called(ctx, fn)
	return args.Error(0)
}
