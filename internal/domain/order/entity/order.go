package entity

import (
	"errors"
	"time"

	"rankmyapp/internal/domain/order/event"
	"rankmyapp/internal/domain/order/valueobject"
)

// Order is the aggregate root
type Order struct {
	id         string
	customerID string
	status     valueobject.OrderStatus
	items      []valueobject.OrderItem
	totalPrice float64
	createdAt  time.Time
	updatedAt  time.Time

	domainEvents []event.DomainEvent
}

func NewOrder(id, customerID string, items []valueobject.OrderItem) (*Order, error) {
	if id == "" {
		return nil, errors.New("order ID is required")
	}
	if customerID == "" {
		return nil, errors.New("customer ID is required")
	}
	if len(items) == 0 {
		return nil, errors.New("order must have at least one item")
	}

	now := time.Now().UTC()
	o := &Order{
		id:         id,
		customerID: customerID,
		status:     valueobject.StatusCreated,
		items:      items,
		createdAt:  now,
		updatedAt:  now,
	}
	for _, item := range items {
		o.totalPrice += item.Total()
	}

	o.domainEvents = append(o.domainEvents, event.OrderCreatedEvent{
		ID:         id,
		CustomerID: customerID,
		OccurredAt: now,
	})

	return o, nil
}

func Reconstitute(
	id, customerID string,
	items []valueobject.OrderItem,
	status valueobject.OrderStatus,
	totalPrice float64,
	createdAt, updatedAt time.Time,
) *Order {
	return &Order{
		id:         id,
		customerID: customerID,
		status:     status,
		items:      items,
		totalPrice: totalPrice,
		createdAt:  createdAt,
		updatedAt:  updatedAt,
	}
}

func (o *Order) UpdateStatus(newStatus valueobject.OrderStatus) error {
	if o.status == newStatus {
		return nil
	}
	old := o.status
	o.status = newStatus
	o.updatedAt = time.Now().UTC()

	o.domainEvents = append(o.domainEvents, event.OrderStatusChangedEvent{
		ID:         o.id,
		OldStatus:  old.String(),
		NewStatus:  newStatus.String(),
		OccurredAt: o.updatedAt,
	})
	return nil
}

func (o *Order) DomainEvents() []event.DomainEvent { return o.domainEvents }

func (o *Order) ClearDomainEvents() { o.domainEvents = nil }

func (o *Order) ID() string                      { return o.id }
func (o *Order) CustomerID() string              { return o.customerID }
func (o *Order) Status() valueobject.OrderStatus { return o.status }
func (o *Order) Items() []valueobject.OrderItem  { return o.items }
func (o *Order) TotalPrice() float64             { return o.totalPrice }
func (o *Order) CreatedAt() time.Time            { return o.createdAt }
func (o *Order) UpdatedAt() time.Time            { return o.updatedAt }
