package valueobject

import "errors"

type OrderItem struct {
	ProductID   string
	ProductName string
	Quantity    int
	UnitPrice   float64
}

func NewOrderItem(productID, productName string, quantity int, unitPrice float64) (OrderItem, error) {
	if productID == "" {
		return OrderItem{}, errors.New("product ID is required")
	}
	if productName == "" {
		return OrderItem{}, errors.New("product name is required")
	}
	if quantity <= 0 {
		return OrderItem{}, errors.New("quantity must be greater than zero")
	}
	if unitPrice <= 0 {
		return OrderItem{}, errors.New("unit price must be greater than zero")
	}
	return OrderItem{
		ProductID:   productID,
		ProductName: productName,
		Quantity:    quantity,
		UnitPrice:   unitPrice,
	}, nil
}

func (i OrderItem) Total() float64 {
	return float64(i.Quantity) * i.UnitPrice
}

func (i OrderItem) Subtotal() float64 {
	return i.Total()
}
