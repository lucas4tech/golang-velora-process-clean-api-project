package rabbitmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"rankmyapp/pkg/apmutil"
	"rankmyapp/pkg/logger"
)

type Publisher struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
}

func NewPublisher(conn *amqp.Connection, exchange string) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("rabbitmq: open channel: %w", err)
	}

	if err = ch.ExchangeDeclare(
		exchange,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		return nil, fmt.Errorf("rabbitmq: declare exchange: %w", err)
	}

	return &Publisher{conn: conn, channel: ch, exchange: exchange}, nil
}

func (p *Publisher) Publish(ctx context.Context, eventName string, payload []byte) (err error) {
	log := logger.Get()

	span, sctx := apmutil.MessagingPublishSpan(ctx, eventName)
	defer func() { apmutil.EndSpan(span, err) }()

	err = p.channel.PublishWithContext(sctx,
		p.exchange,
		eventName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         payload,
		},
	)
	if err != nil {
		log.Error("rabbitmq: publish failed", zap.String("event", eventName), zap.Error(err))
		return fmt.Errorf("rabbitmq: publish: %w", err)
	}

	log.Info("rabbitmq: published", zap.String("event", eventName))
	return nil
}

func (p *Publisher) Close() {
	if p.channel != nil {
		_ = p.channel.Close()
	}
}
