package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockOrderDomainService struct {
	mock.Mock
}

func (m *MockOrderDomainService) CanCustomerCreateOrder(ctx context.Context, customerID string) error {
	args := m.Called(ctx, customerID)
	return args.Error(0)
}

func (m *MockOrderDomainService) IsOrderEligibleForCancellation(ctx context.Context, orderID string) (bool, error) {
	args := m.Called(ctx, orderID)
	return args.Bool(0), args.Error(1)
}
