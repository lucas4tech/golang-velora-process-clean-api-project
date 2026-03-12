package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"rankmyapp/internal/domain/order/entity"
	orderrepo "rankmyapp/internal/domain/order/repository"
)

type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) Save(ctx context.Context, order *entity.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderRepository) FindByID(ctx context.Context, id string) (*entity.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Order), args.Error(1)
}

func (m *MockOrderRepository) FindAll(ctx context.Context, filter orderrepo.OrderFilter) ([]*entity.Order, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Order), args.Error(1)
}

func (m *MockOrderRepository) Update(ctx context.Context, order *entity.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}
