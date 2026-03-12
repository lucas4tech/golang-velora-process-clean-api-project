package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rankmyapp/internal/app/query"
	"rankmyapp/internal/app/usecase"
	"rankmyapp/internal/domain/order/entity"
	orderrepo "rankmyapp/internal/domain/order/repository"
)

func TestGetOrderByIDHandler_Success(t *testing.T) {
	repo := &mockOrderRepository{}
	order := buildTestOrder(t)

	repo.On("FindByID", context.Background(), "order-1").Return(order, nil)

	handler := usecase.NewGetOrderByIDHandler(repo)
	result, err := handler.Handle(context.Background(), query.GetOrderByIDQuery{OrderID: "order-1"})

	require.NoError(t, err)
	assert.Equal(t, "order-1", result.ID)
	assert.Equal(t, "created", result.Status)
	repo.AssertExpectations(t)
}

func TestGetOrderByIDHandler_NotFound(t *testing.T) {
	repo := &mockOrderRepository{}
	repo.On("FindByID", context.Background(), "missing").Return((*entity.Order)(nil), assert.AnError)

	handler := usecase.NewGetOrderByIDHandler(repo)
	_, err := handler.Handle(context.Background(), query.GetOrderByIDQuery{OrderID: "missing"})

	assert.Error(t, err)
}

func TestListOrdersHandler_Success(t *testing.T) {
	repo := &mockOrderRepository{}
	orders := []*entity.Order{buildTestOrder(t), buildTestOrder(t)}

	filter := orderrepo.OrderFilter{Limit: 10, Offset: 0}
	repo.On("FindAll", context.Background(), filter).Return(orders, nil)

	handler := usecase.NewListOrdersHandler(repo)
	result, err := handler.Handle(context.Background(), query.ListOrdersQuery{Filter: filter})

	require.NoError(t, err)
	assert.Len(t, result.Data, 2)
	assert.Equal(t, 2, result.Total)
	repo.AssertExpectations(t)
}

func TestListOrdersHandler_FindAllError(t *testing.T) {
	repo := &mockOrderRepository{}
	repo.On("FindAll", context.Background(), orderrepo.OrderFilter{Limit: 20, Offset: 0}).Return(([]*entity.Order)(nil), assert.AnError)

	handler := usecase.NewListOrdersHandler(repo)
	_, err := handler.Handle(context.Background(), query.ListOrdersQuery{Filter: orderrepo.OrderFilter{Limit: 20, Offset: 0}})
	assert.Error(t, err)
}

func TestListOrdersHandler_Empty(t *testing.T) {
	repo := &mockOrderRepository{}
	filter := orderrepo.OrderFilter{Limit: 10}
	repo.On("FindAll", context.Background(), filter).Return([]*entity.Order{}, nil)

	handler := usecase.NewListOrdersHandler(repo)
	result, err := handler.Handle(context.Background(), query.ListOrdersQuery{Filter: filter})

	require.NoError(t, err)
	assert.Empty(t, result.Data)
}
