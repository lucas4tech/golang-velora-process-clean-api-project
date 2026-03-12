package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"rankmyapp/internal/infra/http/handler"
	"rankmyapp/internal/infra/http/middleware"
)

func Setup(orderHandler *handler.OrderHandler) *gin.Engine {
	r := gin.New()

	r.Use(middleware.RequestLogger())
	r.Use(gin.Recovery())
	r.Use(middleware.ErrorHandler())

	r.GET("/health", handler.HealthCheck)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{
		orders := v1.Group("/orders")
		{
			orders.POST("", orderHandler.CreateOrder)
			orders.GET("", orderHandler.ListOrders)
			orders.GET("/:id", orderHandler.GetOrderByID)
			orders.PATCH("/:id/status", orderHandler.UpdateOrderStatus)
		}
	}

	return r
}
