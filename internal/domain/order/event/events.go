package event

import "time"

type DomainEvent interface {
	EventName() string
	AggregateID() string
}

type OrderCreatedEvent struct {
	ID         string
	CustomerID string
	OccurredAt time.Time
}

func (e OrderCreatedEvent) EventName() string   { return "order.created" }
func (e OrderCreatedEvent) AggregateID() string { return e.ID }

type OrderStatusChangedEvent struct {
	ID         string
	OldStatus  string
	NewStatus  string
	OccurredAt time.Time
}

func (e OrderStatusChangedEvent) EventName() string   { return "order.status_changed" }
func (e OrderStatusChangedEvent) AggregateID() string { return e.ID }
