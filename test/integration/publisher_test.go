//go:build integration

package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rankmyapp/internal/infra/messaging/rabbitmq"
	"rankmyapp/test/integration/helpers"
)

func rabbitURL() string {
	if u := os.Getenv("RABBITMQ_URL"); u != "" {
		return u
	}
	return "amqp://guest:guest@localhost:5672/"
}

func TestPublisher_PublishAndConsume(t *testing.T) {
	conn, ch, err := helpers.ConnectRabbitMQ(rabbitURL())
	require.NoError(t, err)
	defer conn.Close()
	defer ch.Close()

	const (
		exchange   = "orders.events"
		routingKey = "order.created"
	)

	qName, err := helpers.BindTestQueue(ch, exchange, routingKey)
	require.NoError(t, err)

	pub, err := rabbitmq.NewPublisher(conn, exchange)
	require.NoError(t, err)
	defer pub.Close()

	ctx := context.Background()
	err = pub.Publish(ctx, routingKey, []byte(`{"event":"order.created","id":"test-1"}`))
	require.NoError(t, err)

	msgs, err := ch.Consume(qName, "", true, true, false, false, nil)
	require.NoError(t, err)

	select {
	case msg := <-msgs:
		assert.Equal(t, routingKey, msg.RoutingKey)
		assert.JSONEq(t, `{"event":"order.created","id":"test-1"}`, string(msg.Body))
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestPublisher_PublishMultipleRoutingKeys(t *testing.T) {
	conn, ch, err := helpers.ConnectRabbitMQ(rabbitURL())
	require.NoError(t, err)
	defer conn.Close()
	defer ch.Close()

	const exchange = "orders.events"

	keys := []string{"order.created", "order.status_changed"}
	queues := make(map[string]string)

	for _, key := range keys {
		qName, err := helpers.BindTestQueue(ch, exchange, key)
		require.NoError(t, err)
		queues[key] = qName
	}

	pub, err := rabbitmq.NewPublisher(conn, exchange)
	require.NoError(t, err)
	defer pub.Close()

	ctx := context.Background()
	for _, key := range keys {
		err := pub.Publish(ctx, key, []byte(`{"routing_key":"`+key+`"}`))
		require.NoError(t, err)
	}

	for _, key := range keys {
		msgs, err := ch.Consume(queues[key], "", true, true, false, false, nil)
		require.NoError(t, err)

		select {
		case msg := <-msgs:
			assert.Equal(t, key, msg.RoutingKey)
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for message with routing key %q", key)
		}
	}
}

func TestPublisher_ClosedConnection(t *testing.T) {
	conn, ch, err := helpers.ConnectRabbitMQ(rabbitURL())
	require.NoError(t, err)
	ch.Close()

	pub, err := rabbitmq.NewPublisher(conn, "orders.events")
	require.NoError(t, err)
	defer pub.Close()

	conn.Close()

	err = pub.Publish(context.Background(), "order.created", []byte(`{}`))

	var amqpErr *amqp.Error
	assert.ErrorAs(t, err, &amqpErr, "expected an amqp.Error after connection closed")
}
