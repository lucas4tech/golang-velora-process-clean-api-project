//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	outboxentity "rankmyapp/internal/domain/outbox/entity"
	"rankmyapp/internal/infra/messaging/rabbitmq"
	"rankmyapp/internal/infra/outboxworker"
	"rankmyapp/test/integration/helpers"
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

func TestWorker_OrderCreated_PublishedToRabbitMQ(t *testing.T) {
	conn, ch, err := helpers.ConnectRabbitMQ(rabbitURL())
	require.NoError(t, err)
	defer conn.Close()
	defer ch.Close()

	const exchange = "orders.events"
	qName, err := helpers.BindTestQueue(ch, exchange, "order.created")
	require.NoError(t, err)

	pub, err := rabbitmq.NewPublisher(conn, exchange)
	require.NoError(t, err)
	defer pub.Close()

	msg := outboxentity.NewOutboxMessage("msg-1", "order-1", "order.created", []byte(`{"id":"order-1"}`))

	repo := &mockOutboxRepo{}
	repo.On("FindPending", mock.Anything, mock.Anything).
		Return([]*outboxentity.OutboxMessage{msg}, nil).Once()
	repo.On("FindPending", mock.Anything, mock.Anything).
		Return([]*outboxentity.OutboxMessage{}, nil)
	repo.On("UpdateStatus", mock.Anything, mock.Anything).Return(nil)

	w := outboxworker.New(repo, pub)
	w.ProcessOnce(context.Background())

	msgs, err := ch.Consume(qName, "", true, true, false, false, nil)
	require.NoError(t, err)

	select {
	case received := <-msgs:
		assert.Equal(t, "order.created", received.RoutingKey)
		assert.JSONEq(t, `{"id":"order-1"}`, string(received.Body))
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for published message")
	}

	repo.AssertExpectations(t)
}

func TestWorker_StatusChanged_PublishedToRabbitMQ(t *testing.T) {
	conn, ch, err := helpers.ConnectRabbitMQ(rabbitURL())
	require.NoError(t, err)
	defer conn.Close()
	defer ch.Close()

	const exchange = "orders.events"
	qName, err := helpers.BindTestQueue(ch, exchange, "order.status_changed")
	require.NoError(t, err)

	pub, err := rabbitmq.NewPublisher(conn, exchange)
	require.NoError(t, err)
	defer pub.Close()

	msg := outboxentity.NewOutboxMessage("msg-2", "order-2", "order.status_changed", []byte(`{"id":"order-2","status":"processing"}`))

	repo := &mockOutboxRepo{}
	repo.On("FindPending", mock.Anything, mock.Anything).
		Return([]*outboxentity.OutboxMessage{msg}, nil).Once()
	repo.On("UpdateStatus", mock.Anything, mock.Anything).Return(nil)

	w := outboxworker.New(repo, pub)
	w.ProcessOnce(context.Background())

	msgs, err := ch.Consume(qName, "", true, true, false, false, nil)
	require.NoError(t, err)

	select {
	case received := <-msgs:
		assert.Equal(t, "order.status_changed", received.RoutingKey)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for published message")
	}

	repo.AssertExpectations(t)
}

func TestWorker_PublishFailure_IncrementsAttempt(t *testing.T) {
	failPub := &failPublisher{err: errors.New("broker unavailable")}

	msg := outboxentity.NewOutboxMessage("msg-3", "order-3", "order.created", []byte(`{}`))
	initialAttempts := msg.Attempts

	repo := &mockOutboxRepo{}
	repo.On("FindPending", mock.Anything, mock.Anything).
		Return([]*outboxentity.OutboxMessage{msg}, nil).Once()
	repo.On("UpdateStatus", mock.Anything, mock.Anything).Return(nil)

	w := outboxworker.New(repo, failPub)
	w.ProcessOnce(context.Background())

	assert.Equal(t, initialAttempts+1, msg.Attempts)
	repo.AssertExpectations(t)
}

func TestWorker_MaxAttempts_MarksAsFailed(t *testing.T) {
	msg := outboxentity.NewOutboxMessage("msg-4", "order-4", "order.created", []byte(`{}`))
	for range 5 {
		msg.IncrementAttempt()
	}

	repo := &mockOutboxRepo{}
	repo.On("FindPending", mock.Anything, mock.Anything).
		Return([]*outboxentity.OutboxMessage{msg}, nil).Once()
	repo.On("UpdateStatus", mock.Anything, mock.Anything).Return(nil)

	called := false
	pub := &callbackPublisher{fn: func() { called = true }}

	w := outboxworker.New(repo, pub)
	w.ProcessOnce(context.Background())

	assert.False(t, called, "publish should not be called when max attempts reached")
	assert.Equal(t, outboxentity.OutboxStatusFailed, msg.Status)
	repo.AssertExpectations(t)
}

type failPublisher struct{ err error }

func (p *failPublisher) Publish(_ context.Context, _ string, _ []byte) error { return p.err }

type callbackPublisher struct{ fn func() }

func (p *callbackPublisher) Publish(_ context.Context, _ string, _ []byte) error {
	p.fn()
	return nil
}
