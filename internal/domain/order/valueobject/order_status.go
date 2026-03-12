package valueobject

import "errors"

type OrderStatus string

const (
	StatusCreated    OrderStatus = "created"
	StatusProcessing OrderStatus = "processing"
	StatusShipped    OrderStatus = "shipped"
	StatusDelivered  OrderStatus = "delivered"
	StatusCancelled  OrderStatus = "cancelled"
)

var validTransitions = map[OrderStatus][]OrderStatus{
	StatusCreated:    {StatusProcessing, StatusCancelled},
	StatusProcessing: {StatusShipped, StatusCancelled},
	StatusShipped:    {StatusDelivered, StatusCancelled},
}

func NewOrderStatus(s string) (OrderStatus, error) {
	st := OrderStatus(s)
	switch st {
	case StatusCreated, StatusProcessing, StatusShipped, StatusDelivered, StatusCancelled:
		return st, nil
	}
	return "", errors.New("invalid order status: " + s)
}

func (s OrderStatus) CanTransitionTo(next OrderStatus) bool {
	for _, allowed := range validTransitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}

func (s OrderStatus) IsTerminal() bool {
	return s == StatusDelivered || s == StatusCancelled
}

func (s OrderStatus) String() string { return string(s) }
