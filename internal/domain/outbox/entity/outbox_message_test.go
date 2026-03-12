package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewOutboxMessage(t *testing.T) {
	msg := NewOutboxMessage("id-1", "agg-1", "order.created", []byte(`{}`))
	assert.Equal(t, "id-1", msg.ID)
	assert.Equal(t, "agg-1", msg.AggregateID)
	assert.Equal(t, "order.created", msg.EventName)
	assert.Equal(t, []byte(`{}`), msg.Payload)
	assert.Equal(t, OutboxStatusPending, msg.Status)
	assert.Equal(t, 0, msg.Attempts)
	assert.False(t, msg.CreatedAt.IsZero())
	assert.False(t, msg.UpdatedAt.IsZero())
}

func TestOutboxMessage_MarkPublished(t *testing.T) {
	msg := NewOutboxMessage("id-1", "agg-1", "evt", nil)
	msg.MarkPublished()
	assert.Equal(t, OutboxStatusPublished, msg.Status)
}

func TestOutboxMessage_MarkFailed(t *testing.T) {
	msg := NewOutboxMessage("id-1", "agg-1", "evt", nil)
	msg.MarkFailed()
	assert.Equal(t, OutboxStatusFailed, msg.Status)
}

func TestOutboxMessage_IncrementAttempt(t *testing.T) {
	msg := NewOutboxMessage("id-1", "agg-1", "evt", nil)
	assert.Equal(t, 0, msg.Attempts)
	msg.IncrementAttempt()
	assert.Equal(t, 1, msg.Attempts)
	msg.IncrementAttempt()
	assert.Equal(t, 2, msg.Attempts)
}
