package query

import "rankmyapp/internal/domain/order/repository"

type GetOrderByIDQuery struct {
	OrderID string
}

type ListOrdersQuery struct {
	Filter repository.OrderFilter
}
