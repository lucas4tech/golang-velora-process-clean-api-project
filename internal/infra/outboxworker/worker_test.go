package outboxworker_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	outboxentity "rankmyapp/internal/domain/outbox/entity"
	"rankmyapp/internal/infra/outboxworker"
)

type mockOutboxRepo struct{ mock.Mock }

func (m *mockOutboxRepo) Save(ctx context.Context, msg *outboxentity.OutboxMessage) error {
	return m.Called(ctx, msg).Error(0)
}
func (m *mockOutboxRepo) FindPending(ctx context.Context, limit int) ([]*outboxentity.OutboxMessage, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*outboxentity.OutboxMessage), args.Error(1)
}
func (m *mockOutboxRepo) UpdateStatus(ctx context.Context, msg *outboxentity.OutboxMessage) error {
	return m.Called(ctx, msg).Error(0)
}

type mockPublisher struct {
	publishFn func(ctx context.Context, eventName string, payload []byte) error
}

func (m *mockPublisher) Publish(ctx context.Context, eventName string, payload []byte) error {
	return m.publishFn(ctx, eventName, payload)
}

func TestWorker_ProcessOnce_Success(t *testing.T) {
	repo := &mockOutboxRepo{}
	msg := outboxentity.NewOutboxMessage("msg-1", "order-1", "order.created", []byte(`{}`))

	repo.On("FindPending", mock.Anything, mock.Anything).Return([]*outboxentity.OutboxMessage{msg}, nil).Once()
	repo.On("UpdateStatus", mock.Anything, mock.Anything).Return(nil)

	pub := &mockPublisher{publishFn: func(_ context.Context, _ string, _ []byte) error { return nil }}

	w := outboxworker.New(repo, pub)
	w.ProcessOnce(context.Background())

	repo.AssertExpectations(t)
}

func TestWorker_MaxAttempts_MarksAsFailed(t *testing.T) {
	repo := &mockOutboxRepo{}
	msg := outboxentity.NewOutboxMessage("msg-2", "order-2", "order.created", []byte(`{}`))
	for range 5 {
		msg.IncrementAttempt()
	}

	repo.On("FindPending", mock.Anything, mock.Anything).Return([]*outboxentity.OutboxMessage{msg}, nil).Once()
	repo.On("UpdateStatus", mock.Anything, mock.Anything).Return(nil)

	pub := &mockPublisher{publishFn: func(_ context.Context, _ string, _ []byte) error {
		t.Fatal("should not publish when max attempts reached")
		return nil
	}}

	w := outboxworker.New(repo, pub)
	w.ProcessOnce(context.Background())

	repo.AssertExpectations(t)
}

func TestWorker_PublishFailure_IncrementsAttempt(t *testing.T) {
	repo := &mockOutboxRepo{}
	msg := outboxentity.NewOutboxMessage("msg-3", "order-3", "order.created", []byte(`{}`))

	repo.On("FindPending", mock.Anything, mock.Anything).Return([]*outboxentity.OutboxMessage{msg}, nil).Once()
	repo.On("UpdateStatus", mock.Anything, mock.Anything).Return(nil)

	pub := &mockPublisher{publishFn: func(_ context.Context, _ string, _ []byte) error {
		return errors.New("broker unavailable")
	}}

	w := outboxworker.New(repo, pub)
	w.ProcessOnce(context.Background())

	repo.AssertExpectations(t)
}
