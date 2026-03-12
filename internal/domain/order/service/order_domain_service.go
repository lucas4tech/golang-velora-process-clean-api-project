package service

import (
	"context"
	"errors"

	orderrepo "rankmyapp/internal/domain/order/repository"
	"rankmyapp/internal/domain/order/valueobject"
)

const maxActiveOrders = 5

type Port interface {
	CanCustomerCreateOrder(ctx context.Context, customerID string) error
	IsOrderEligibleForCancellation(ctx context.Context, orderID string) (bool, error)
}

type OrderDomainService struct {
	repo orderrepo.OrderRepository
}

func New(repo orderrepo.OrderRepository) *OrderDomainService {
	return &OrderDomainService{repo: repo}
}

func (s *OrderDomainService) CanCustomerCreateOrder(ctx context.Context, customerID string) error {
	filter := orderrepo.OrderFilter{CustomerID: customerID, Limit: 100}
	orders, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		return err
	}

	active := 0
	for _, o := range orders {
		if !o.Status().IsTerminal() {
			active++
		}
	}
	if active >= maxActiveOrders {
		return errors.New("customer has reached the maximum number of simultaneous active orders")
	}
	return nil
}

func (s *OrderDomainService) IsOrderEligibleForCancellation(ctx context.Context, orderID string) (bool, error) {
	o, err := s.repo.FindByID(ctx, orderID)
	if err != nil {
		return false, err
	}
	return o.Status().CanTransitionTo(valueobject.StatusCancelled), nil
}
