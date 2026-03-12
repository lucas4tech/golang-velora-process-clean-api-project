package event

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOrderCreatedEvent_EventName(t *testing.T) {
	e := OrderCreatedEvent{ID: "ord-1", CustomerID: "c1", OccurredAt: time.Now()}
	assert.Equal(t, "order.created", e.EventName())
}

func TestOrderCreatedEvent_AggregateID(t *testing.T) {
	e := OrderCreatedEvent{ID: "ord-1", CustomerID: "c1", OccurredAt: time.Now()}
	assert.Equal(t, "ord-1", e.AggregateID())
}

func TestOrderStatusChangedEvent_EventName(t *testing.T) {
	e := OrderStatusChangedEvent{ID: "ord-1", OldStatus: "created", NewStatus: "processing", OccurredAt: time.Now()}
	assert.Equal(t, "order.status_changed", e.EventName())
}

func TestOrderStatusChangedEvent_AggregateID(t *testing.T) {
	e := OrderStatusChangedEvent{ID: "ord-1", OldStatus: "created", NewStatus: "processing", OccurredAt: time.Now()}
	assert.Equal(t, "ord-1", e.AggregateID())
}
