package entity

import "time"

type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "pending"
	OutboxStatusPublished OutboxStatus = "published"
	OutboxStatusFailed    OutboxStatus = "failed"
)

type OutboxMessage struct {
	ID          string
	AggregateID string
	EventName   string
	Payload     []byte
	Status      OutboxStatus
	Attempts    int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewOutboxMessage(id, aggregateID, eventName string, payload []byte) *OutboxMessage {
	now := time.Now().UTC()
	return &OutboxMessage{
		ID:          id,
		AggregateID: aggregateID,
		EventName:   eventName,
		Payload:     payload,
		Status:      OutboxStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (m *OutboxMessage) MarkPublished() {
	m.Status = OutboxStatusPublished
	m.UpdatedAt = time.Now().UTC()
}

func (m *OutboxMessage) MarkFailed() {
	m.Status = OutboxStatusFailed
	m.UpdatedAt = time.Now().UTC()
}

func (m *OutboxMessage) IncrementAttempt() {
	m.Attempts++
	m.UpdatedAt = time.Now().UTC()
}
