package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	outboxentity "rankmyapp/internal/domain/outbox/entity"
	outboxrepo "rankmyapp/internal/domain/outbox/repository"
	apperrors "rankmyapp/pkg/errors"
)

const outboxCollection = "outbox_messages"

type outboxDocument struct {
	ID          string    `bson:"_id"`
	AggregateID string    `bson:"aggregate_id"`
	EventName   string    `bson:"event_name"`
	Payload     []byte    `bson:"payload"`
	Status      string    `bson:"status"`
	Attempts    int       `bson:"attempts"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
}

type MongoOutboxRepository struct {
	db *mongo.Database
}

func NewMongoOutboxRepository(db *mongo.Database) *MongoOutboxRepository {
	return &MongoOutboxRepository{db: db}
}

func NewMongoOutboxRepositoryFromContext(db *mongo.Database) outboxrepo.OutboxRepository {
	return &MongoOutboxRepository{db: db}
}

func (r *MongoOutboxRepository) coll() *mongo.Collection {
	return r.db.Collection(outboxCollection)
}

func (r *MongoOutboxRepository) Save(ctx context.Context, msg *outboxentity.OutboxMessage) error {
	_, err := r.coll().InsertOne(ctx, toOutboxDocument(msg))
	if err != nil {
		return apperrors.Wrap(apperrors.ErrInternal.Code, "error saving outbox", 500, err)
	}
	return nil
}

func (r *MongoOutboxRepository) FindPending(ctx context.Context, limit int) ([]*outboxentity.OutboxMessage, error) {
	return findPendingOutbox(ctx, r.coll(), limit)
}

func (r *MongoOutboxRepository) UpdateStatus(ctx context.Context, msg *outboxentity.OutboxMessage) error {
	return updateOutboxStatus(ctx, r.coll(), msg)
}

func findPendingOutbox(ctx context.Context, coll *mongo.Collection, limit int) ([]*outboxentity.OutboxMessage, error) {
	filter := bson.M{"status": string(outboxentity.OutboxStatusPending)}
	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "created_at", Value: 1}})

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrInternal.Code, "error fetching outbox", 500, err)
	}
	defer cursor.Close(ctx)

	var docs []outboxDocument
	if err = cursor.All(ctx, &docs); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrInternal.Code, "error decoding outbox", 500, err)
	}

	msgs := make([]*outboxentity.OutboxMessage, len(docs))
	for i, d := range docs {
		msgs[i] = fromOutboxDocument(d)
	}
	return msgs, nil
}

func updateOutboxStatus(ctx context.Context, coll *mongo.Collection, msg *outboxentity.OutboxMessage) error {
	filter := bson.M{"_id": msg.ID}
	update := bson.M{"$set": bson.M{
		"status":     string(msg.Status),
		"attempts":   msg.Attempts,
		"updated_at": msg.UpdatedAt,
	}}
	_, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrInternal.Code, "error updating outbox", 500, err)
	}
	return nil
}

func toOutboxDocument(m *outboxentity.OutboxMessage) outboxDocument {
	return outboxDocument{
		ID:          m.ID,
		AggregateID: m.AggregateID,
		EventName:   m.EventName,
		Payload:     m.Payload,
		Status:      string(m.Status),
		Attempts:    m.Attempts,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func fromOutboxDocument(d outboxDocument) *outboxentity.OutboxMessage {
	return &outboxentity.OutboxMessage{
		ID:          d.ID,
		AggregateID: d.AggregateID,
		EventName:   d.EventName,
		Payload:     d.Payload,
		Status:      outboxentity.OutboxStatus(d.Status),
		Attempts:    d.Attempts,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}
