package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"rankmyapp/internal/domain/outbox/entity"
)

type MockOutboxRepository struct {
	mock.Mock
}

func (m *MockOutboxRepository) Save(ctx context.Context, msg *entity.OutboxMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockOutboxRepository) FindPending(ctx context.Context, limit int) ([]*entity.OutboxMessage, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.OutboxMessage), args.Error(1)
}

func (m *MockOutboxRepository) UpdateStatus(ctx context.Context, msg *entity.OutboxMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}
