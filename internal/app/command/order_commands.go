package command

import "rankmyapp/internal/app/dto"

type CreateOrderCommand struct {
	Input dto.CreateOrderInput
}

type UpdateOrderStatusCommand struct {
	OrderID string
	Status  string
}
