package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rankmyapp/internal/app/dto"
	"rankmyapp/internal/app/usecase"
	"rankmyapp/internal/domain/order/entity"
	orderrepo "rankmyapp/internal/domain/order/repository"
	"rankmyapp/internal/domain/order/valueobject"
	outboxentity "rankmyapp/internal/domain/outbox/entity"
	outboxrepo "rankmyapp/internal/domain/outbox/repository"
	"rankmyapp/internal/infra/http/handler"
	"rankmyapp/internal/infra/http/middleware"
	apperrors "rankmyapp/pkg/errors"

	"github.com/stretchr/testify/mock"
)

func init() { gin.SetMode(gin.TestMode) }

type mockOrderRepo struct{ mock.Mock }

func (m *mockOrderRepo) Save(ctx context.Context, o *entity.Order) error {
	return m.Called(ctx, o).Error(0)
}
func (m *mockOrderRepo) FindByID(ctx context.Context, id string) (*entity.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Order), args.Error(1)
}
func (m *mockOrderRepo) FindAll(ctx context.Context, f orderrepo.OrderFilter) ([]*entity.Order, error) {
	args := m.Called(ctx, f)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Order), args.Error(1)
}
func (m *mockOrderRepo) Update(ctx context.Context, o *entity.Order) error {
	return m.Called(ctx, o).Error(0)
}

type mockOutboxRepo struct{ mock.Mock }

func (m *mockOutboxRepo) Save(ctx context.Context, msg *outboxentity.OutboxMessage) error {
	return m.Called(ctx, msg).Error(0)
}
func (m *mockOutboxRepo) FindPending(ctx context.Context, limit int) ([]*outboxentity.OutboxMessage, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*outboxentity.OutboxMessage), args.Error(1)
}
func (m *mockOutboxRepo) UpdateStatus(ctx context.Context, msg *outboxentity.OutboxMessage) error {
	return m.Called(ctx, msg).Error(0)
}

type mockUoW struct {
	orderRepo  orderrepo.OrderRepository
	outboxRepo outboxrepo.OutboxRepository
}

func (m *mockUoW) Execute(ctx context.Context, fn func(context.Context, orderrepo.OrderRepository, outboxrepo.OutboxRepository) error) error {
	return fn(ctx, m.orderRepo, m.outboxRepo)
}

func buildOrder(t *testing.T) *entity.Order {
	t.Helper()
	item, err := valueobject.NewOrderItem("prod-1", "Sneakers", 2, 100.0)
	require.NoError(t, err)
	o, err := entity.NewOrder("order-1", "cust-1", []valueobject.OrderItem{item})
	require.NoError(t, err)
	return o
}

func setupRouter(h *handler.OrderHandler) *gin.Engine {
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.POST("/orders", h.CreateOrder)
	r.GET("/orders/:id", h.GetOrderByID)
	r.GET("/orders", h.ListOrders)
	r.PATCH("/orders/:id/status", h.UpdateOrderStatus)
	return r
}

func TestCreateOrder_HTTP_Success(t *testing.T) {
	orderRepo := &mockOrderRepo{}
	outboxRepo := &mockOutboxRepo{}
	uow := &mockUoW{orderRepo: orderRepo, outboxRepo: outboxRepo}
	readRepo := &mockOrderRepo{}

	orderRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
	outboxRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	h := handler.NewOrderHandler(
		usecase.NewCreateOrderHandler(uow),
		usecase.NewUpdateOrderStatusHandler(uow, readRepo),
		usecase.NewGetOrderByIDHandler(readRepo),
		usecase.NewListOrdersHandler(readRepo),
	)

	body := dto.CreateOrderInput{
		CustomerID: "cust-1",
		Items:      []dto.CreateOrderItemInput{{ProductID: "p1", ProductName: "Sneakers", Quantity: 1, UnitPrice: 150.0}},
	}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp dto.OrderResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "cust-1", resp.CustomerID)
	assert.Equal(t, "created", resp.Status)
}

func TestCreateOrder_HTTP_InvalidBody(t *testing.T) {
	h := handler.NewOrderHandler(nil, nil, nil, nil)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(h).ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateOrder_HTTP_CreateFails(t *testing.T) {
	orderRepo := &mockOrderRepo{}
	outboxRepo := &mockOutboxRepo{}
	uow := &mockUoW{orderRepo: orderRepo, outboxRepo: outboxRepo}
	orderRepo.On("Save", mock.Anything, mock.Anything).Return(assert.AnError)

	h := handler.NewOrderHandler(
		usecase.NewCreateOrderHandler(uow),
		usecase.NewUpdateOrderStatusHandler(uow, &mockOrderRepo{}),
		usecase.NewGetOrderByIDHandler(&mockOrderRepo{}),
		usecase.NewListOrdersHandler(&mockOrderRepo{}),
	)
	body := dto.CreateOrderInput{
		CustomerID: "cust-1",
		Items:      []dto.CreateOrderItemInput{{ProductID: "p1", ProductName: "X", Quantity: 1, UnitPrice: 10.0}},
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(h).ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListOrders_HTTP_RepoError(t *testing.T) {
	readRepo := &mockOrderRepo{}
	readRepo.On("FindAll", mock.Anything, mock.Anything).Return(([]*entity.Order)(nil), apperrors.ErrOrderNotFound)

	uow := &mockUoW{orderRepo: &mockOrderRepo{}, outboxRepo: &mockOutboxRepo{}}
	h := handler.NewOrderHandler(
		usecase.NewCreateOrderHandler(uow),
		usecase.NewUpdateOrderStatusHandler(uow, readRepo),
		usecase.NewGetOrderByIDHandler(readRepo),
		usecase.NewListOrdersHandler(readRepo),
	)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/orders", nil)
	setupRouter(h).ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateOrderStatus_HTTP_NotFound(t *testing.T) {
	readRepo := &mockOrderRepo{}
	readRepo.On("FindByID", mock.Anything, "missing").Return((*entity.Order)(nil), apperrors.ErrOrderNotFound)

	uow := &mockUoW{orderRepo: &mockOrderRepo{}, outboxRepo: &mockOutboxRepo{}}
	h := handler.NewOrderHandler(
		usecase.NewCreateOrderHandler(uow),
		usecase.NewUpdateOrderStatusHandler(uow, readRepo),
		usecase.NewGetOrderByIDHandler(readRepo),
		usecase.NewListOrdersHandler(readRepo),
	)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, "/orders/missing/status", bytes.NewBufferString(`{"status":"processing"}`))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(h).ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetOrderByID_HTTP_Success(t *testing.T) {
	readRepo := &mockOrderRepo{}
	order := buildOrder(t)
	readRepo.On("FindByID", mock.Anything, "order-1").Return(order, nil)

	uow := &mockUoW{orderRepo: &mockOrderRepo{}, outboxRepo: &mockOutboxRepo{}}
	h := handler.NewOrderHandler(
		usecase.NewCreateOrderHandler(uow),
		usecase.NewUpdateOrderStatusHandler(uow, readRepo),
		usecase.NewGetOrderByIDHandler(readRepo),
		usecase.NewListOrdersHandler(readRepo),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/orders/order-1", nil)
	setupRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp dto.OrderResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "order-1", resp.ID)
}

func TestGetOrderByID_HTTP_NotFound(t *testing.T) {
	readRepo := &mockOrderRepo{}
	readRepo.On("FindByID", mock.Anything, "unknown").Return((*entity.Order)(nil), apperrors.ErrOrderNotFound)

	uow := &mockUoW{orderRepo: &mockOrderRepo{}, outboxRepo: &mockOutboxRepo{}}
	h := handler.NewOrderHandler(
		usecase.NewCreateOrderHandler(uow),
		usecase.NewUpdateOrderStatusHandler(uow, readRepo),
		usecase.NewGetOrderByIDHandler(readRepo),
		usecase.NewListOrdersHandler(readRepo),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/orders/unknown", nil)
	setupRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateOrderStatus_HTTP_Success(t *testing.T) {
	orderRepo := &mockOrderRepo{}
	outboxRepo := &mockOutboxRepo{}
	uow := &mockUoW{orderRepo: orderRepo, outboxRepo: outboxRepo}

	order := buildOrder(t)
	readRepo := &mockOrderRepo{}
	readRepo.On("FindByID", mock.Anything, "order-1").Return(order, nil)
	orderRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	outboxRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	h := handler.NewOrderHandler(
		usecase.NewCreateOrderHandler(uow),
		usecase.NewUpdateOrderStatusHandler(uow, readRepo),
		usecase.NewGetOrderByIDHandler(readRepo),
		usecase.NewListOrdersHandler(readRepo),
	)

	body := dto.UpdateOrderStatusInput{Status: "processing"}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, "/orders/order-1/status", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetOrderByID_HTTP_EmptyID(t *testing.T) {
	readRepo := &mockOrderRepo{}
	uow := &mockUoW{orderRepo: &mockOrderRepo{}, outboxRepo: &mockOutboxRepo{}}
	h := handler.NewOrderHandler(
		usecase.NewCreateOrderHandler(uow),
		usecase.NewUpdateOrderStatusHandler(uow, readRepo),
		usecase.NewGetOrderByIDHandler(readRepo),
		usecase.NewListOrdersHandler(readRepo),
	)

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/orders/x", nil)
	ctx.Params = gin.Params{{Key: "id", Value: ""}}
	h.GetOrderByID(ctx)
	middleware.ErrorHandler()(ctx)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListOrders_HTTP_Success(t *testing.T) {
	readRepo := &mockOrderRepo{}
	readRepo.On("FindAll", mock.Anything, mock.Anything).Return([]*entity.Order{}, nil)

	uow := &mockUoW{orderRepo: &mockOrderRepo{}, outboxRepo: &mockOutboxRepo{}}
	h := handler.NewOrderHandler(
		usecase.NewCreateOrderHandler(uow),
		usecase.NewUpdateOrderStatusHandler(uow, readRepo),
		usecase.NewGetOrderByIDHandler(readRepo),
		usecase.NewListOrdersHandler(readRepo),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/orders", nil)
	setupRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListOrders_HTTP_WithLimitOffset(t *testing.T) {
	readRepo := &mockOrderRepo{}
	readRepo.On("FindAll", mock.Anything, mock.Anything).Return([]*entity.Order{}, nil)

	uow := &mockUoW{orderRepo: &mockOrderRepo{}, outboxRepo: &mockOutboxRepo{}}
	h := handler.NewOrderHandler(
		usecase.NewCreateOrderHandler(uow),
		usecase.NewUpdateOrderStatusHandler(uow, readRepo),
		usecase.NewGetOrderByIDHandler(readRepo),
		usecase.NewListOrdersHandler(readRepo),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/orders?limit=10&offset=5", nil)
	setupRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListOrders_HTTP_InvalidLimitUsesDefault(t *testing.T) {
	readRepo := &mockOrderRepo{}
	readRepo.On("FindAll", mock.Anything, mock.Anything).Return([]*entity.Order{}, nil)

	uow := &mockUoW{orderRepo: &mockOrderRepo{}, outboxRepo: &mockOutboxRepo{}}
	h := handler.NewOrderHandler(
		usecase.NewCreateOrderHandler(uow),
		usecase.NewUpdateOrderStatusHandler(uow, readRepo),
		usecase.NewGetOrderByIDHandler(readRepo),
		usecase.NewListOrdersHandler(readRepo),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/orders?limit=abc&offset=-1", nil)
	setupRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListOrders_HTTP_WithStatusFilter(t *testing.T) {
	readRepo := &mockOrderRepo{}
	readRepo.On("FindAll", mock.Anything, mock.MatchedBy(func(f orderrepo.OrderFilter) bool {
		return f.Status != nil && *f.Status == valueobject.StatusProcessing
	})).Return([]*entity.Order{}, nil)

	uow := &mockUoW{orderRepo: &mockOrderRepo{}, outboxRepo: &mockOutboxRepo{}}
	h := handler.NewOrderHandler(
		usecase.NewCreateOrderHandler(uow),
		usecase.NewUpdateOrderStatusHandler(uow, readRepo),
		usecase.NewGetOrderByIDHandler(readRepo),
		usecase.NewListOrdersHandler(readRepo),
	)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/orders?status=processing", nil)
	setupRouter(h).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListOrders_HTTP_InvalidStatus(t *testing.T) {
	readRepo := &mockOrderRepo{}
	uow := &mockUoW{orderRepo: &mockOrderRepo{}, outboxRepo: &mockOutboxRepo{}}
	h := handler.NewOrderHandler(
		usecase.NewCreateOrderHandler(uow),
		usecase.NewUpdateOrderStatusHandler(uow, readRepo),
		usecase.NewGetOrderByIDHandler(readRepo),
		usecase.NewListOrdersHandler(readRepo),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/orders?status=invalid", nil)
	setupRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateOrderStatus_HTTP_EmptyID(t *testing.T) {
	readRepo := &mockOrderRepo{}
	uow := &mockUoW{orderRepo: &mockOrderRepo{}, outboxRepo: &mockOutboxRepo{}}
	h := handler.NewOrderHandler(
		usecase.NewCreateOrderHandler(uow),
		usecase.NewUpdateOrderStatusHandler(uow, readRepo),
		usecase.NewGetOrderByIDHandler(readRepo),
		usecase.NewListOrdersHandler(readRepo),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, "/orders//status", bytes.NewBufferString(`{"status":"processing"}`))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateOrderStatus_HTTP_InvalidStatus(t *testing.T) {
	readRepo := &mockOrderRepo{}
	uow := &mockUoW{orderRepo: &mockOrderRepo{}, outboxRepo: &mockOutboxRepo{}}
	h := handler.NewOrderHandler(
		usecase.NewCreateOrderHandler(uow),
		usecase.NewUpdateOrderStatusHandler(uow, readRepo),
		usecase.NewGetOrderByIDHandler(readRepo),
		usecase.NewListOrdersHandler(readRepo),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, "/orders/order-1/status", bytes.NewBufferString(`{"status":"invalid"}`))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateOrderStatus_HTTP_InvalidBody(t *testing.T) {
	readRepo := &mockOrderRepo{}
	uow := &mockUoW{orderRepo: &mockOrderRepo{}, outboxRepo: &mockOutboxRepo{}}
	h := handler.NewOrderHandler(
		usecase.NewCreateOrderHandler(uow),
		usecase.NewUpdateOrderStatusHandler(uow, readRepo),
		usecase.NewGetOrderByIDHandler(readRepo),
		usecase.NewListOrdersHandler(readRepo),
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPatch, "/orders/order-1/status", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	setupRouter(h).ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHealthCheck(t *testing.T) {
	r := gin.New()
	r.GET("/health", handler.HealthCheck)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
