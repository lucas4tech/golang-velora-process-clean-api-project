package configs

import "os"

type AppConfig struct {
	Port string
}

type MongoConfig struct {
	URI      string
	Database string
}

type RabbitMQConfig struct {
	URL      string
	Exchange string
}

func Load() (AppConfig, MongoConfig, RabbitMQConfig) {
	app := AppConfig{
		Port: getEnv("PORT", "8080"),
	}
	mongo := MongoConfig{
		URI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		Database: getEnv("MONGO_DATABASE", "rankmyapp"),
	}
	rabbit := RabbitMQConfig{
		URL:      getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		Exchange: getEnv("RABBITMQ_EXCHANGE", "orders.events"),
	}
	return app, mongo, rabbit
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
