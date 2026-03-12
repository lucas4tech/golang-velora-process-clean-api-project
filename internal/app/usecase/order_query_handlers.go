package usecase

import (
	"context"

	"rankmyapp/internal/app/dto"
	"rankmyapp/internal/app/query"
	orderrepo "rankmyapp/internal/domain/order/repository"
)

type GetOrderByIDHandler struct {
	repo orderrepo.OrderRepository
}

func NewGetOrderByIDHandler(repo orderrepo.OrderRepository) *GetOrderByIDHandler {
	return &GetOrderByIDHandler{repo: repo}
}

func (h *GetOrderByIDHandler) Handle(ctx context.Context, q query.GetOrderByIDQuery) (*dto.OrderResponse, error) {
	order, err := h.repo.FindByID(ctx, q.OrderID)
	if err != nil {
		return nil, err
	}
	return toOrderResponse(order), nil
}

type ListOrdersHandler struct {
	repo orderrepo.OrderRepository
}

func NewListOrdersHandler(repo orderrepo.OrderRepository) *ListOrdersHandler {
	return &ListOrdersHandler{repo: repo}
}

func (h *ListOrdersHandler) Handle(ctx context.Context, q query.ListOrdersQuery) (*dto.PaginatedOrdersResponse, error) {
	orders, err := h.repo.FindAll(ctx, q.Filter)
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.OrderResponse, len(orders))
	for i, o := range orders {
		responses[i] = toOrderResponse(o)
	}

	return &dto.PaginatedOrdersResponse{
		Data:   responses,
		Total:  len(responses),
		Limit:  q.Filter.Limit,
		Offset: q.Filter.Offset,
	}, nil
}
