package repository

import (
	"context"

	"rankmyapp/internal/domain/order/entity"
	"rankmyapp/internal/domain/order/valueobject"
)

type OrderFilter struct {
	CustomerID string
	Status     *valueobject.OrderStatus
	Limit      int64
	Offset     int64
}

type OrderRepository interface {
	Save(ctx context.Context, order *entity.Order) error
	FindByID(ctx context.Context, id string) (*entity.Order, error)
	FindAll(ctx context.Context, filter OrderFilter) ([]*entity.Order, error)
	Update(ctx context.Context, order *entity.Order) error
}
