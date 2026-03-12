package dto

import "time"

type CreateOrderItemInput struct {
	ProductID   string  `json:"product_id" binding:"required" example:"prod-abc-123"`
	ProductName string  `json:"product_name" binding:"required" example:"Wireless Headphones"`
	Quantity    int     `json:"quantity" binding:"required,min=1" example:"2"`
	UnitPrice   float64 `json:"unit_price" binding:"required,gt=0" example:"149.99"`
}

type CreateOrderInput struct {
	CustomerID string                 `json:"customer_id" binding:"required" example:"customer-789"`
	Items      []CreateOrderItemInput `json:"items" binding:"required,min=1,dive"`
}

type UpdateOrderStatusInput struct {
	// Status - Options: created, processing, shipped, delivered, cancelled
	Status string `json:"status" binding:"required" example:"processing" enums:"created,processing,shipped,delivered,cancelled"`
}

type OrderItemResponse struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Subtotal    float64 `json:"subtotal"`
}

type OrderResponse struct {
	ID         string              `json:"id"`
	CustomerID string              `json:"customer_id"`
	Items      []OrderItemResponse `json:"items"`
	Status     string              `json:"status" example:"processing" enums:"created,processing,shipped,delivered,cancelled"`
	TotalPrice float64             `json:"total_price"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

type PaginatedOrdersResponse struct {
	Data   []*OrderResponse `json:"data"`
	Total  int              `json:"total"`
	Limit  int64            `json:"limit"`
	Offset int64            `json:"offset"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
