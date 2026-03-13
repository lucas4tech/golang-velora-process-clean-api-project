package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/v2/mongo"
	mongoopts "go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"

	"rankmyapp/configs"
	"rankmyapp/internal/infra/messaging/rabbitmq"
	"rankmyapp/internal/infra/outboxworker"
	outboxdb "rankmyapp/internal/infra/persistence/mongodb/outbox"
	"rankmyapp/pkg/logger"
)

func main() {
	log := logger.Get()

	_, mongoCfg, rabbitCfg := configs.Load()

	mongoClient, err := mongo.Connect(mongoopts.Client().ApplyURI(mongoCfg.URI))
	if err != nil {
		log.Fatal("worker: failed to connect to MongoDB", zap.Error(err))
	}
	defer func() { _ = mongoClient.Disconnect(context.Background()) }()

	db := mongoClient.Database(mongoCfg.Database)

	amqpConn, err := amqp.Dial(rabbitCfg.URL)
	if err != nil {
		log.Fatal("worker: failed to connect to RabbitMQ", zap.Error(err))
	}
	defer amqpConn.Close()

	publisher, err := rabbitmq.NewPublisher(amqpConn, rabbitCfg.Exchange)
	if err != nil {
		log.Fatal("worker: failed to create RabbitMQ publisher", zap.Error(err))
	}
	defer publisher.Close()

	outboxRepo := outboxdb.NewMongoOutboxRepository(db)
	worker := outboxworker.New(outboxRepo, publisher)

	ctx, cancel := context.WithCancel(context.Background())
	go worker.Start(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("worker: signal received, shutting down...")
	cancel()
	time.Sleep(2 * time.Second)
	log.Info("worker: stopped")
}
