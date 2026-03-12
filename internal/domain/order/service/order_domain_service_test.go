package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rankmyapp/internal/domain/order/entity"
	orderrepo "rankmyapp/internal/domain/order/repository"
	repomocks "rankmyapp/internal/domain/order/repository/mocks"
	"rankmyapp/internal/domain/order/valueobject"
)

func makeOrder(t *testing.T, customerID string, status valueobject.OrderStatus) *entity.Order {
	t.Helper()
	item, _ := valueobject.NewOrderItem("p1", "Item", 1, 10.0)
	o, err := entity.NewOrder("ord-"+customerID, customerID, []valueobject.OrderItem{item})
	require.NoError(t, err)
	o.ClearDomainEvents()
	if status != valueobject.StatusCreated {
		_ = o.UpdateStatus(valueobject.StatusProcessing)
		o.ClearDomainEvents()
		if status == valueobject.StatusCancelled {
			_ = o.UpdateStatus(valueobject.StatusCancelled)
			o.ClearDomainEvents()
		}
	}
	return o
}

func TestCanCustomerCreateOrder_BelowLimit(t *testing.T) {
	repo := new(repomocks.MockOrderRepository)
	filter := orderrepo.OrderFilter{CustomerID: "cust-1", Limit: 100}
	repo.On("FindAll", context.Background(), filter).
		Return([]*entity.Order{makeOrder(t, "cust-1", valueobject.StatusCreated)}, nil)

	svc := New(repo)
	err := svc.CanCustomerCreateOrder(context.Background(), "cust-1")
	assert.NoError(t, err)
}

func TestCanCustomerCreateOrder_AtLimit(t *testing.T) {
	orders := make([]*entity.Order, maxActiveOrders)
	for i := range orders {
		orders[i] = makeOrder(t, "cust-1", valueobject.StatusCreated)
	}

	repo := new(repomocks.MockOrderRepository)
	filter := orderrepo.OrderFilter{CustomerID: "cust-1", Limit: 100}
	repo.On("FindAll", context.Background(), filter).Return(orders, nil)

	svc := New(repo)
	err := svc.CanCustomerCreateOrder(context.Background(), "cust-1")
	assert.Error(t, err)
}

func TestCanCustomerCreateOrder_RepoError(t *testing.T) {
	repo := new(repomocks.MockOrderRepository)
	filter := orderrepo.OrderFilter{CustomerID: "cust-1", Limit: 100}
	repo.On("FindAll", context.Background(), filter).Return(nil, errors.New("db error"))

	svc := New(repo)
	err := svc.CanCustomerCreateOrder(context.Background(), "cust-1")
	assert.Error(t, err)
}

func TestIsOrderEligibleForCancellation_Eligible(t *testing.T) {
	o := makeOrder(t, "cust-1", valueobject.StatusCreated)

	repo := new(repomocks.MockOrderRepository)
	repo.On("FindByID", context.Background(), o.ID()).Return(o, nil)

	svc := New(repo)
	ok, err := svc.IsOrderEligibleForCancellation(context.Background(), o.ID())
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestIsOrderEligibleForCancellation_NotEligible(t *testing.T) {
	o := makeOrder(t, "cust-1", valueobject.StatusCreated)
	_ = o.UpdateStatus(valueobject.StatusProcessing)
	_ = o.UpdateStatus(valueobject.StatusShipped)
	_ = o.UpdateStatus(valueobject.StatusDelivered)
	o.ClearDomainEvents()

	repo := new(repomocks.MockOrderRepository)
	repo.On("FindByID", context.Background(), o.ID()).Return(o, nil)

	svc := New(repo)
	ok, err := svc.IsOrderEligibleForCancellation(context.Background(), o.ID())
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestIsOrderEligibleForCancellation_RepoError(t *testing.T) {
	repo := new(repomocks.MockOrderRepository)
	repo.On("FindByID", context.Background(), "ord-x").Return(nil, errors.New("db error"))

	svc := New(repo)
	_, err := svc.IsOrderEligibleForCancellation(context.Background(), "ord-x")
	assert.Error(t, err)
}
