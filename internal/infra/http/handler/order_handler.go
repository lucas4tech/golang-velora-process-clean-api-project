package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"rankmyapp/internal/app/command"
	"rankmyapp/internal/app/dto"
	"rankmyapp/internal/app/query"
	"rankmyapp/internal/app/usecase"
	orderrepo "rankmyapp/internal/domain/order/repository"
	"rankmyapp/internal/domain/order/valueobject"
	apperrors "rankmyapp/pkg/errors"
)

type OrderHandler struct {
	createHandler       *usecase.CreateOrderHandler
	updateStatusHandler *usecase.UpdateOrderStatusHandler
	getByIDHandler      *usecase.GetOrderByIDHandler
	listHandler         *usecase.ListOrdersHandler
}

func NewOrderHandler(
	create *usecase.CreateOrderHandler,
	updateStatus *usecase.UpdateOrderStatusHandler,
	getByID *usecase.GetOrderByIDHandler,
	list *usecase.ListOrdersHandler,
) *OrderHandler {
	return &OrderHandler{
		createHandler:       create,
		updateStatusHandler: updateStatus,
		getByIDHandler:      getByID,
		listHandler:         list,
	}
}

// CreateOrder godoc
// @Summary      Cria um novo pedido
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateOrderInput  true  "Dados do pedido"
// @Success      201   {object}  dto.OrderResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      500   {object}  dto.ErrorResponse
// @Router       /api/v1/orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var input dto.CreateOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(wrapBindError(err))
		return
	}

	result, err := h.createHandler.Handle(c.Request.Context(), command.CreateOrderCommand{Input: input})
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

// GetOrderByID godoc
// @Summary      Consulta pedido por ID
// @Tags         orders
// @Produce      json
// @Param        id   path      string  true  "Order ID"
// @Success      200  {object}  dto.OrderResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Router       /api/v1/orders/{id} [get]
func (h *OrderHandler) GetOrderByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		_ = c.Error(errInvalidID())
		return
	}

	result, err := h.getByIDHandler.Handle(c.Request.Context(), query.GetOrderByIDQuery{OrderID: id})
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListOrders godoc
// @Summary      Lista pedidos com filtros
// @Tags         orders
// @Produce      json
// @Param        customer_id  query     string  false  "ID do cliente"
// @Param        status       query     string  false  "Status do pedido"
// @Param        limit        query     int     false  "Limite de registros"
// @Param        offset       query     int     false  "Offset de paginação"
// @Success      200          {object}  dto.PaginatedOrdersResponse
// @Router       /api/v1/orders [get]
func (h *OrderHandler) ListOrders(c *gin.Context) {
	filter := orderrepo.OrderFilter{
		CustomerID: c.Query("customer_id"),
		Limit:      parseIntQuery(c, "limit", 20),
		Offset:     parseIntQuery(c, "offset", 0),
	}

	if s := c.Query("status"); s != "" {
		status, err := valueobject.NewOrderStatus(s)
		if err != nil {
			_ = c.Error(apperrors.Wrap(apperrors.ErrInvalidStatus.Code, err.Error(), apperrors.ErrInvalidStatus.StatusCode, err))
			return
		}
		filter.Status = &status
	}

	result, err := h.listHandler.Handle(c.Request.Context(), query.ListOrdersQuery{Filter: filter})
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateOrderStatus godoc
// @Summary      Atualiza o status do pedido
// @Description  Aceita qualquer status do enum: created, processing, shipped, delivered, cancelled (sem restrição de transição).
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        id    path      string                       true  "Order ID"
// @Param        body  body      dto.UpdateOrderStatusInput   true  "Novo status. Opções: created, processing, shipped, delivered, cancelled"
// @Success      200   {object}  dto.OrderResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Failure      422   {object}  dto.ErrorResponse
// @Router       /api/v1/orders/{id}/status [patch]
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		_ = c.Error(errInvalidID())
		return
	}

	var input dto.UpdateOrderStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(wrapBindError(err))
		return
	}
	if _, err := valueobject.NewOrderStatus(input.Status); err != nil {
		_ = c.Error(apperrors.Wrap(apperrors.ErrInvalidStatus.Code, err.Error(), apperrors.ErrInvalidStatus.StatusCode, err))
		return
	}

	result, err := h.updateStatusHandler.Handle(c.Request.Context(), command.UpdateOrderStatusCommand{
		OrderID: id,
		Status:  input.Status,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// HealthCheck godoc
// @Summary      Healthcheck do serviço
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func parseIntQuery(c *gin.Context, key string, defaultVal int64) int64 {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil || n < 0 {
		return defaultVal
	}
	return n
}
