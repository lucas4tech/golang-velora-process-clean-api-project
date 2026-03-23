// cmd/api — HTTP API service entry point.
//
// @title           RankMyApp Order API
// @version         1.0
// @description     E-commerce order management service following Clean Architecture + CQRS + DDD + Outbox Pattern.
// @host            localhost:8080
// @BasePath        /
// @tag.name        orders
// @tag.description Order management endpoints
// @tag.name        health
// @tag.description Health check
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.elastic.co/apm/v2"
	"go.mongodb.org/mongo-driver/v2/mongo"
	mongoopts "go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"

	"rankmyapp/configs"
	_ "rankmyapp/docs"
	"rankmyapp/internal/app/usecase"
	orderrepo "rankmyapp/internal/domain/order/repository"
	outboxrepo "rankmyapp/internal/domain/outbox/repository"
	"rankmyapp/internal/infra/http/handler"
	"rankmyapp/internal/infra/http/router"
	"rankmyapp/internal/infra/messaging/rabbitmq"
	orderdb "rankmyapp/internal/infra/persistence/mongodb/order"
	outboxdb "rankmyapp/internal/infra/persistence/mongodb/outbox"
	"rankmyapp/internal/infra/unitofwork"
	"rankmyapp/pkg/logger"
)

func main() {
	log := logger.Get()

	appCfg, mongoCfg, rabbitCfg := configs.Load()

	mongoClient, err := mongo.Connect(mongoopts.Client().ApplyURI(mongoCfg.URI))
	if err != nil {
		log.Fatal("failed to connect to MongoDB", zap.Error(err))
	}
	defer func() { _ = mongoClient.Disconnect(context.Background()) }()

	db := mongoClient.Database(mongoCfg.Database)

	amqpConn, err := amqp.Dial(rabbitCfg.URL)
	if err != nil {
		log.Fatal("failed to connect to RabbitMQ", zap.Error(err))
	}
	defer amqpConn.Close()

	publisher, err := rabbitmq.NewPublisher(amqpConn, rabbitCfg.Exchange)
	if err != nil {
		log.Fatal("failed to create RabbitMQ publisher", zap.Error(err))
	}
	defer publisher.Close()
	_ = publisher

	readOrderRepo := orderdb.NewMongoOrderRepository(db)

	uow := unitofwork.NewMongoUnitOfWork(
		mongoClient,
		func(ctx context.Context) orderrepo.OrderRepository {
			return orderdb.NewMongoOrderRepositoryFromContext(db)
		},
		func(ctx context.Context) outboxrepo.OutboxRepository {
			return outboxdb.NewMongoOutboxRepositoryFromContext(db)
		},
	)

	createHandler := usecase.NewCreateOrderHandler(uow)
	updateHandler := usecase.NewUpdateOrderStatusHandler(uow, readOrderRepo)
	getByIDHandler := usecase.NewGetOrderByIDHandler(readOrderRepo)
	listHandler := usecase.NewListOrdersHandler(readOrderRepo)

	orderHandler := handler.NewOrderHandler(createHandler, updateHandler, getByIDHandler, listHandler)
	r := router.Setup(orderHandler)

	srv := &http.Server{
		Addr:    ":" + appCfg.Port,
		Handler: r,
	}

	go func() {
		log.Info("api: starting server", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("api: listen error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("api: signal received, shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("api: forced shutdown", zap.Error(err))
	}
	apm.DefaultTracer().Flush(nil)
	log.Info("api: server stopped")
}
