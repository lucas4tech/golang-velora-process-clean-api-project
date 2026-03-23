package apmutil

import (
	"context"

	"go.elastic.co/apm/v2"
)

func MongoSpan(ctx context.Context, operation, collection string) (*apm.Span, context.Context) {
	name := collection + "." + operation
	span, ctx := apm.StartSpan(ctx, name, "db")
	if span != nil && !span.Dropped() {
		span.Subtype = "mongodb"
		span.Action = "query"
		span.Context.SetDatabase(apm.DatabaseSpanContext{
			Type:      "mongodb",
			Statement: name,
		})
	}
	return span, ctx
}

func EndSpan(span *apm.Span, err error) {
	if span == nil {
		return
	}
	if err != nil {
		span.Outcome = "failure"
	} else {
		span.Outcome = "success"
	}
	span.End()
}

func MessagingPublishSpan(ctx context.Context, routingKey string) (*apm.Span, context.Context) {
	span, ctx := apm.StartSpan(ctx, "rabbitmq.publish", "messaging")
	if span != nil && !span.Dropped() {
		span.Subtype = "rabbitmq"
		span.Action = "send"
		span.Context.SetMessage(apm.MessageSpanContext{QueueName: routingKey})
	}
	return span, ctx
}
