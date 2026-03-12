package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rankmyapp/internal/domain/order/valueobject"
)

func validItems(t *testing.T) []valueobject.OrderItem {
	t.Helper()
	item, err := valueobject.NewOrderItem("prod-1", "T-Shirt", 2, 49.90)
	require.NoError(t, err)
	return []valueobject.OrderItem{item}
}

func TestNewOrder_Success(t *testing.T) {
	o, err := NewOrder("ord-1", "cust-1", validItems(t))
	require.NoError(t, err)
	assert.Equal(t, "ord-1", o.ID())
	assert.Equal(t, valueobject.StatusCreated, o.Status())
	assert.InDelta(t, 99.80, o.TotalPrice(), 0.01)

	evts := o.DomainEvents()
	require.Len(t, evts, 1)
	assert.Equal(t, "order.created", evts[0].EventName())
}

func TestNewOrder_MissingID(t *testing.T) {
	_, err := NewOrder("", "cust-1", validItems(t))
	assert.Error(t, err)
}

func TestNewOrder_MissingCustomer(t *testing.T) {
	_, err := NewOrder("ord-1", "", validItems(t))
	assert.Error(t, err)
}

func TestNewOrder_NoItems(t *testing.T) {
	_, err := NewOrder("ord-1", "cust-1", nil)
	assert.Error(t, err)
}

func TestOrder_UpdateStatus_ValidTransition(t *testing.T) {
	o, _ := NewOrder("ord-1", "cust-1", validItems(t))
	o.ClearDomainEvents()

	err := o.UpdateStatus(valueobject.StatusProcessing)
	require.NoError(t, err)
	assert.Equal(t, valueobject.StatusProcessing, o.Status())

	evts := o.DomainEvents()
	require.Len(t, evts, 1)
	assert.Equal(t, "order.status_changed", evts[0].EventName())
}

func TestOrder_UpdateStatus_AnyEnumAllowed(t *testing.T) {
	o, _ := NewOrder("ord-1", "cust-1", validItems(t))
	o.ClearDomainEvents()

	err := o.UpdateStatus(valueobject.StatusDelivered)
	require.NoError(t, err)
	assert.Equal(t, valueobject.StatusDelivered, o.Status())
}

func TestOrder_UpdateStatus_Cancelled(t *testing.T) {
	o, _ := NewOrder("ord-1", "cust-1", validItems(t))
	o.ClearDomainEvents()

	err := o.UpdateStatus(valueobject.StatusCancelled)
	require.NoError(t, err)
	assert.Equal(t, valueobject.StatusCancelled, o.Status())
	assert.True(t, o.Status().IsTerminal())
}

func TestOrder_UpdateStatus_SameStatusNoOp(t *testing.T) {
	o, _ := NewOrder("ord-1", "cust-1", validItems(t))
	o.ClearDomainEvents()

	err := o.UpdateStatus(valueobject.StatusCreated)
	require.NoError(t, err)
	assert.Equal(t, valueobject.StatusCreated, o.Status())
	assert.Len(t, o.DomainEvents(), 0)
}

func TestReconstitute_Getters(t *testing.T) {
	items := validItems(t)
	created := time.Now().UTC().Add(-time.Hour)
	updated := time.Now().UTC()
	o := Reconstitute("ord-1", "cust-1", items, valueobject.StatusProcessing, 99.80, created, updated)

	assert.Equal(t, "ord-1", o.ID())
	assert.Equal(t, "cust-1", o.CustomerID())
	assert.Equal(t, valueobject.StatusProcessing, o.Status())
	assert.Equal(t, items, o.Items())
	assert.InDelta(t, 99.80, o.TotalPrice(), 0.01)
	assert.Equal(t, created, o.CreatedAt())
	assert.Equal(t, updated, o.UpdatedAt())
	assert.Nil(t, o.DomainEvents())
}
