//go:build integration

package helpers

import (
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func ConnectRabbitMQ(url string) (*amqp.Connection, *amqp.Channel, error) {
	const maxRetries = 10
	var (
		conn *amqp.Connection
		err  error
	)

	for i := range maxRetries {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("rabbitmq: could not connect after retries: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("rabbitmq: could not open channel: %w", err)
	}

	return conn, ch, nil
}

func BindTestQueue(ch *amqp.Channel, exchange, routingKey string) (string, error) {
	if err := ch.ExchangeDeclare(
		exchange, "topic", true, false, false, false, nil,
	); err != nil {
		return "", fmt.Errorf("rabbitmq: declare exchange: %w", err)
	}

	q, err := ch.QueueDeclare(
		"",    // auto-generated name
		false, // non-durable
		true,  // auto-delete
		true,  // exclusive
		false,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("rabbitmq: declare queue: %w", err)
	}

	if err := ch.QueueBind(q.Name, routingKey, exchange, false, nil); err != nil {
		return "", fmt.Errorf("rabbitmq: bind queue: %w", err)
	}

	return q.Name, nil
}
