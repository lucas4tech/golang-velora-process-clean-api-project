package mongodb

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"rankmyapp/internal/domain/order/entity"
	orderrepo "rankmyapp/internal/domain/order/repository"
	"rankmyapp/internal/domain/order/valueobject"
	"rankmyapp/pkg/apmutil"
	apperrors "rankmyapp/pkg/errors"
)

const orderCollection = "orders"

type orderDocument struct {
	ID         string         `bson:"_id"`
	CustomerID string         `bson:"customer_id"`
	Items      []itemDocument `bson:"items"`
	Status     string         `bson:"status"`
	TotalPrice float64        `bson:"total_price"`
	CreatedAt  time.Time      `bson:"created_at"`
	UpdatedAt  time.Time      `bson:"updated_at"`
}

type itemDocument struct {
	ProductID   string  `bson:"product_id"`
	ProductName string  `bson:"product_name"`
	Quantity    int     `bson:"quantity"`
	UnitPrice   float64 `bson:"unit_price"`
}

type MongoOrderRepository struct {
	db *mongo.Database
}

func NewMongoOrderRepository(db *mongo.Database) *MongoOrderRepository {
	return &MongoOrderRepository{db: db}
}

func NewMongoOrderRepositoryFromContext(db *mongo.Database) orderrepo.OrderRepository {
	return &MongoOrderRepository{db: db}
}

func (r *MongoOrderRepository) coll() *mongo.Collection {
	return r.db.Collection(orderCollection)
}

func (r *MongoOrderRepository) Save(ctx context.Context, order *entity.Order) (err error) {
	span, sctx := apmutil.MongoSpan(ctx, "insertOne", orderCollection)
	defer apmutil.EndSpan(span, err)

	doc := toDocument(order)
	_, err = r.coll().InsertOne(sctx, doc)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrInternal.Code, "error saving order", 500, err)
	}
	return nil
}

func (r *MongoOrderRepository) FindByID(ctx context.Context, id string) (_ *entity.Order, err error) {
	span, sctx := apmutil.MongoSpan(ctx, "findOne", orderCollection)
	defer func() {
		if err != nil && errors.Is(err, apperrors.ErrOrderNotFound) {
			apmutil.EndSpan(span, nil)
			return
		}
		apmutil.EndSpan(span, err)
	}()

	var doc orderDocument
	err = r.coll().FindOne(sctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.ErrOrderNotFound
		}
		return nil, apperrors.Wrap(apperrors.ErrInternal.Code, "error fetching order", 500, err)
	}
	return fromDocument(doc)
}

func (r *MongoOrderRepository) FindAll(ctx context.Context, filter orderrepo.OrderFilter) ([]*entity.Order, error) {
	return findAll(ctx, r.coll(), filter, orderCollection)
}

func (r *MongoOrderRepository) Update(ctx context.Context, order *entity.Order) error {
	return updateOrder(ctx, r.coll(), order, orderCollection)
}

func findAll(ctx context.Context, coll *mongo.Collection, filter orderrepo.OrderFilter, collectionName string) (orders []*entity.Order, err error) {
	span, sctx := apmutil.MongoSpan(ctx, "find", collectionName)
	defer func() { apmutil.EndSpan(span, err) }()

	q := bson.M{}
	if filter.CustomerID != "" {
		q["customer_id"] = filter.CustomerID
	}
	if filter.Status != nil {
		q["status"] = filter.Status.String()
	}

	opts := options.Find()
	if filter.Limit > 0 {
		opts.SetLimit(filter.Limit)
	}
	if filter.Offset > 0 {
		opts.SetSkip(filter.Offset)
	}
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := coll.Find(sctx, q, opts)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrInternal.Code, "error listing orders", 500, err)
	}
	defer cursor.Close(sctx)

	var docs []orderDocument
	if err = cursor.All(sctx, &docs); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrInternal.Code, "error decoding orders", 500, err)
	}

	orders = make([]*entity.Order, 0, len(docs))
	for _, d := range docs {
		o, ferr := fromDocument(d)
		if ferr != nil {
			return nil, ferr
		}
		orders = append(orders, o)
	}
	return orders, nil
}

func updateOrder(ctx context.Context, coll *mongo.Collection, order *entity.Order, collectionName string) (err error) {
	span, sctx := apmutil.MongoSpan(ctx, "updateOne", collectionName)
	defer func() {
		if err != nil && errors.Is(err, apperrors.ErrOrderNotFound) {
			apmutil.EndSpan(span, nil)
			return
		}
		apmutil.EndSpan(span, err)
	}()

	filter := bson.M{"_id": order.ID()}
	update := bson.M{"$set": bson.M{
		"status":     order.Status().String(),
		"updated_at": order.UpdatedAt(),
	}}
	res, err := coll.UpdateOne(sctx, filter, update)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrInternal.Code, "error updating order", 500, err)
	}
	if res.MatchedCount == 0 {
		return apperrors.ErrOrderNotFound
	}
	return nil
}

func toDocument(o *entity.Order) orderDocument {
	items := make([]itemDocument, len(o.Items()))
	for i, item := range o.Items() {
		items[i] = itemDocument{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
		}
	}
	return orderDocument{
		ID:         o.ID(),
		CustomerID: o.CustomerID(),
		Items:      items,
		Status:     o.Status().String(),
		TotalPrice: o.TotalPrice(),
		CreatedAt:  o.CreatedAt(),
		UpdatedAt:  o.UpdatedAt(),
	}
}

func fromDocument(doc orderDocument) (*entity.Order, error) {
	items := make([]valueobject.OrderItem, len(doc.Items))
	for i, it := range doc.Items {
		item, err := valueobject.NewOrderItem(it.ProductID, it.ProductName, it.Quantity, it.UnitPrice)
		if err != nil {
			return nil, err
		}
		items[i] = item
	}
	status, err := valueobject.NewOrderStatus(doc.Status)
	if err != nil {
		return nil, err
	}
	return entity.Reconstitute(doc.ID, doc.CustomerID, items, status, doc.TotalPrice, doc.CreatedAt, doc.UpdatedAt), nil
}
