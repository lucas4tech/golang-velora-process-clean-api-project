package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"rankmyapp/internal/app/command"
	"rankmyapp/internal/app/dto"
	"rankmyapp/internal/app/usecase"
	"rankmyapp/internal/domain/order/entity"
	"rankmyapp/internal/domain/order/valueobject"
)

func buildTestOrder(t *testing.T) *entity.Order {
	t.Helper()
	item, err := valueobject.NewOrderItem("prod-1", "Sneakers", 2, 100.0)
	require.NoError(t, err)
	o, err := entity.NewOrder("order-1", "cust-1", []valueobject.OrderItem{item})
	require.NoError(t, err)
	return o
}

func TestCreateOrderHandler_Success(t *testing.T) {
	orderRepo := &mockOrderRepository{}
	outboxRepo := &mockOutboxRepository{}
	uow := &mockUnitOfWork{orderRepo: orderRepo, outboxRepo: outboxRepo}

	orderRepo.On("Save", context.Background(), mock.Anything).Return(nil)
	outboxRepo.On("Save", context.Background(), mock.Anything).Return(nil)

	handler := usecase.NewCreateOrderHandler(uow)
	cmd := command.CreateOrderCommand{
		Input: dto.CreateOrderInput{
			CustomerID: "cust-1",
			Items: []dto.CreateOrderItemInput{
				{ProductID: "prod-1", ProductName: "Sneakers", Quantity: 2, UnitPrice: 100.0},
			},
		},
	}

	result, err := handler.Handle(context.Background(), cmd)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "cust-1", result.CustomerID)
	assert.Equal(t, "created", result.Status)
	assert.InDelta(t, 200.0, result.TotalPrice, 0.001)
}

func TestCreateOrderHandler_EmptyItems(t *testing.T) {
	uow := &mockUnitOfWork{}
	handler := usecase.NewCreateOrderHandler(uow)
	cmd := command.CreateOrderCommand{
		Input: dto.CreateOrderInput{
			CustomerID: "cust-1",
			Items:      []dto.CreateOrderItemInput{},
		},
	}
	_, err := handler.Handle(context.Background(), cmd)
	assert.Error(t, err)
}

func TestCreateOrderHandler_InvalidItem(t *testing.T) {
	uow := &mockUnitOfWork{}
	handler := usecase.NewCreateOrderHandler(uow)
	cmd := command.CreateOrderCommand{
		Input: dto.CreateOrderInput{
			CustomerID: "cust-1",
			Items: []dto.CreateOrderItemInput{
				{ProductID: "p1", ProductName: "X", Quantity: 0, UnitPrice: 10.0},
			},
		},
	}
	_, err := handler.Handle(context.Background(), cmd)
	assert.Error(t, err)
}

func TestCreateOrderHandler_SaveError(t *testing.T) {
	orderRepo := &mockOrderRepository{}
	outboxRepo := &mockOutboxRepository{}
	uow := &mockUnitOfWork{orderRepo: orderRepo, outboxRepo: outboxRepo}

	orderRepo.On("Save", context.Background(), mock.Anything).Return(assert.AnError)

	handler := usecase.NewCreateOrderHandler(uow)
	cmd := command.CreateOrderCommand{
		Input: dto.CreateOrderInput{
			CustomerID: "cust-1",
			Items: []dto.CreateOrderItemInput{
				{ProductID: "p1", ProductName: "Item", Quantity: 1, UnitPrice: 50.0},
			},
		},
	}
	_, err := handler.Handle(context.Background(), cmd)
	assert.Error(t, err)
}

func TestUpdateOrderStatusHandler_Success(t *testing.T) {
	orderRepo := &mockOrderRepository{}
	outboxRepo := &mockOutboxRepository{}
	uow := &mockUnitOfWork{orderRepo: orderRepo, outboxRepo: outboxRepo}

	order := buildTestOrder(t)
	readRepo := &mockOrderRepository{}
	readRepo.On("FindByID", context.Background(), "order-1").Return(order, nil)

	orderRepo.On("Update", context.Background(), mock.Anything).Return(nil)
	outboxRepo.On("Save", context.Background(), mock.Anything).Return(nil)

	handler := usecase.NewUpdateOrderStatusHandler(uow, readRepo)
	result, err := handler.Handle(context.Background(), command.UpdateOrderStatusCommand{
		OrderID: "order-1",
		Status:  "processing",
	})

	require.NoError(t, err)
	assert.Equal(t, "processing", result.Status)
	readRepo.AssertExpectations(t)
	orderRepo.AssertExpectations(t)
}

func TestUpdateOrderStatusHandler_AnyStatusInEnumAllowed(t *testing.T) {
	orderRepo := &mockOrderRepository{}
	outboxRepo := &mockOutboxRepository{}
	uow := &mockUnitOfWork{orderRepo: orderRepo, outboxRepo: outboxRepo}

	order := buildTestOrder(t)
	readRepo := &mockOrderRepository{}
	readRepo.On("FindByID", context.Background(), "order-1").Return(order, nil)
	orderRepo.On("Update", context.Background(), mock.Anything).Return(nil)
	outboxRepo.On("Save", context.Background(), mock.Anything).Return(nil)

	handler := usecase.NewUpdateOrderStatusHandler(uow, readRepo)
	result, err := handler.Handle(context.Background(), command.UpdateOrderStatusCommand{
		OrderID: "order-1",
		Status:  "delivered",
	})
	require.NoError(t, err)
	assert.Equal(t, "delivered", result.Status)
}

func TestUpdateOrderStatusHandler_NotFound(t *testing.T) {
	readRepo := &mockOrderRepository{}
	uow := &mockUnitOfWork{}

	readRepo.On("FindByID", context.Background(), "unknown-id").Return((*entity.Order)(nil), assert.AnError)

	handler := usecase.NewUpdateOrderStatusHandler(uow, readRepo)
	_, err := handler.Handle(context.Background(), command.UpdateOrderStatusCommand{
		OrderID: "unknown-id",
		Status:  "processing",
	})
	assert.Error(t, err)
}

func TestUpdateOrderStatusHandler_UpdateFails(t *testing.T) {
	order := buildTestOrder(t)
	readRepo := &mockOrderRepository{}
	orderRepo := &mockOrderRepository{}
	outboxRepo := &mockOutboxRepository{}
	uow := &mockUnitOfWork{orderRepo: orderRepo, outboxRepo: outboxRepo}

	readRepo.On("FindByID", context.Background(), "order-1").Return(order, nil)
	orderRepo.On("Update", context.Background(), mock.Anything).Return(assert.AnError)

	handler := usecase.NewUpdateOrderStatusHandler(uow, readRepo)
	_, err := handler.Handle(context.Background(), command.UpdateOrderStatusCommand{
		OrderID: "order-1",
		Status:  "processing",
	})
	assert.Error(t, err)
}

func TestUpdateOrderStatusHandler_InvalidStatus(t *testing.T) {
	readRepo := &mockOrderRepository{}
	uow := &mockUnitOfWork{}

	handler := usecase.NewUpdateOrderStatusHandler(uow, readRepo)
	_, err := handler.Handle(context.Background(), command.UpdateOrderStatusCommand{
		OrderID: "order-1",
		Status:  "invalid",
	})
	assert.Error(t, err)
}
