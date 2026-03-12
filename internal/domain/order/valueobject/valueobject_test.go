package valueobject

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOrderStatus_Valid(t *testing.T) {
	for _, s := range []string{"created", "processing", "shipped", "delivered", "cancelled"} {
		st, err := NewOrderStatus(s)
		require.NoError(t, err)
		assert.Equal(t, OrderStatus(s), st)
	}
}

func TestNewOrderStatus_Invalid(t *testing.T) {
	_, err := NewOrderStatus("unknown")
	assert.Error(t, err)
}

func TestOrderStatus_CanTransitionTo(t *testing.T) {
	cases := []struct {
		from OrderStatus
		to   OrderStatus
		ok   bool
	}{
		{StatusCreated, StatusProcessing, true},
		{StatusCreated, StatusCancelled, true},
		{StatusCreated, StatusDelivered, false},
		{StatusProcessing, StatusShipped, true},
		{StatusProcessing, StatusCancelled, true},
		{StatusShipped, StatusDelivered, true},
		{StatusShipped, StatusCancelled, true},
		{StatusDelivered, StatusCancelled, false},
	}
	for _, c := range cases {
		assert.Equal(t, c.ok, c.from.CanTransitionTo(c.to), "%s -> %s", c.from, c.to)
	}
}

func TestOrderStatus_IsTerminal(t *testing.T) {
	assert.True(t, StatusDelivered.IsTerminal())
	assert.True(t, StatusCancelled.IsTerminal())
	assert.False(t, StatusCreated.IsTerminal())
	assert.False(t, StatusProcessing.IsTerminal())
	assert.False(t, StatusShipped.IsTerminal())
}

func TestNewOrderItem_Valid(t *testing.T) {
	item, err := NewOrderItem("prod-1", "Sneakers", 2, 199.90)
	require.NoError(t, err)
	assert.InDelta(t, 399.80, item.Total(), 0.01)
}

func TestNewOrderItem_MissingProductID(t *testing.T) {
	_, err := NewOrderItem("", "Product", 1, 10.0)
	assert.Error(t, err)
}

func TestNewOrderItem_InvalidQuantity(t *testing.T) {
	_, err := NewOrderItem("prod-1", "Product", 0, 10.0)
	assert.Error(t, err)
}

func TestNewOrderItem_InvalidPrice(t *testing.T) {
	_, err := NewOrderItem("prod-1", "Product", 1, -5.0)
	assert.Error(t, err)
}

func TestNewOrderItem_MissingProductName(t *testing.T) {
	_, err := NewOrderItem("prod-1", "", 1, 10.0)
	assert.Error(t, err)
}

func TestOrderItem_Total_Subtotal(t *testing.T) {
	item, err := NewOrderItem("p1", "Item", 3, 10.5)
	require.NoError(t, err)
	assert.InDelta(t, 31.5, item.Total(), 0.01)
	assert.InDelta(t, 31.5, item.Subtotal(), 0.01)
}

func TestOrderStatus_String(t *testing.T) {
	assert.Equal(t, "created", StatusCreated.String())
	assert.Equal(t, "processing", StatusProcessing.String())
}
