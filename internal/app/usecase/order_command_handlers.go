package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"rankmyapp/internal/app/command"
	"rankmyapp/internal/app/dto"
	"rankmyapp/internal/domain/order/entity"
	orderrepo "rankmyapp/internal/domain/order/repository"
	"rankmyapp/internal/domain/order/valueobject"
	outboxentity "rankmyapp/internal/domain/outbox/entity"
	outboxrepo "rankmyapp/internal/domain/outbox/repository"
	"rankmyapp/internal/infra/unitofwork"
	apperrors "rankmyapp/pkg/errors"
	"rankmyapp/pkg/logger"
)

type CreateOrderHandler struct {
	uow unitofwork.UnitOfWork
}

func NewCreateOrderHandler(uow unitofwork.UnitOfWork) *CreateOrderHandler {
	return &CreateOrderHandler{uow: uow}
}

func (h *CreateOrderHandler) Handle(ctx context.Context, cmd command.CreateOrderCommand) (*dto.OrderResponse, error) {
	log := logger.Get()

	items := make([]valueobject.OrderItem, 0, len(cmd.Input.Items))
	for _, i := range cmd.Input.Items {
		item, err := valueobject.NewOrderItem(i.ProductID, i.ProductName, i.Quantity, i.UnitPrice)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	orderID := uuid.NewString()
	order, err := entity.NewOrder(orderID, cmd.Input.CustomerID, items)
	if err != nil {
		return nil, err
	}

	var result *dto.OrderResponse

	err = h.uow.Execute(ctx, func(ctx context.Context, or orderrepo.OrderRepository, ob outboxrepo.OutboxRepository) error {
		if saveErr := or.Save(ctx, order); saveErr != nil {
			return saveErr
		}

		for _, domainEvent := range order.DomainEvents() {
			payload, jsonErr := json.Marshal(domainEvent)
			if jsonErr != nil {
				return fmt.Errorf("marshal event: %w", jsonErr)
			}
			msg := outboxentity.NewOutboxMessage(
				uuid.NewString(),
				domainEvent.AggregateID(),
				domainEvent.EventName(),
				payload,
			)
			if saveErr := ob.Save(ctx, msg); saveErr != nil {
				return saveErr
			}
		}

		order.ClearDomainEvents()
		result = toOrderResponse(order)
		return nil
	})

	if err != nil {
		log.Error("CreateOrderHandler: " + err.Error())
		return nil, err
	}

	return result, nil
}

type UpdateOrderStatusHandler struct {
	uow           unitofwork.UnitOfWork
	readOrderRepo orderrepo.OrderRepository
}

func NewUpdateOrderStatusHandler(uow unitofwork.UnitOfWork, readRepo orderrepo.OrderRepository) *UpdateOrderStatusHandler {
	return &UpdateOrderStatusHandler{uow: uow, readOrderRepo: readRepo}
}

func (h *UpdateOrderStatusHandler) Handle(ctx context.Context, cmd command.UpdateOrderStatusCommand) (*dto.OrderResponse, error) {
	log := logger.Get()

	newStatus, err := valueobject.NewOrderStatus(cmd.Status)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrInvalidStatus.Code, err.Error(), apperrors.ErrInvalidStatus.StatusCode, err)
	}

	order, err := h.readOrderRepo.FindByID(ctx, cmd.OrderID)
	if err != nil {
		return nil, err
	}

	if err = order.UpdateStatus(newStatus); err != nil {
		return nil, apperrors.Wrap(apperrors.ErrInvalidTransition.Code, err.Error(), apperrors.ErrInvalidTransition.StatusCode, err)
	}

	var result *dto.OrderResponse

	err = h.uow.Execute(ctx, func(ctx context.Context, or orderrepo.OrderRepository, ob outboxrepo.OutboxRepository) error {
		if updateErr := or.Update(ctx, order); updateErr != nil {
			return updateErr
		}

		for _, domainEvent := range order.DomainEvents() {
			payload, jsonErr := json.Marshal(domainEvent)
			if jsonErr != nil {
				return fmt.Errorf("marshal event: %w", jsonErr)
			}
			msg := outboxentity.NewOutboxMessage(
				uuid.NewString(),
				domainEvent.AggregateID(),
				domainEvent.EventName(),
				payload,
			)
			if saveErr := ob.Save(ctx, msg); saveErr != nil {
				return saveErr
			}
		}

		order.ClearDomainEvents()
		result = toOrderResponse(order)
		return nil
	})

	if err != nil {
		log.Error("UpdateOrderStatusHandler: " + err.Error())
		return nil, err
	}

	return result, nil
}

func toOrderResponse(o *entity.Order) *dto.OrderResponse {
	items := make([]dto.OrderItemResponse, len(o.Items()))
	for i, item := range o.Items() {
		items[i] = dto.OrderItemResponse{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Subtotal:    item.Subtotal(),
		}
	}
	return &dto.OrderResponse{
		ID:         o.ID(),
		CustomerID: o.CustomerID(),
		Items:      items,
		Status:     o.Status().String(),
		TotalPrice: o.TotalPrice(),
		CreatedAt:  o.CreatedAt(),
		UpdatedAt:  o.UpdatedAt(),
	}
}

var TimeNow = func() time.Time { return time.Now().UTC() }
