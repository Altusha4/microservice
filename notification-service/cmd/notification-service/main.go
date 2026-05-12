package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Altusha4/microservice/notification-service/internal/email"
	"github.com/Altusha4/microservice/notification-service/internal/infrastructure/rabbitmq"
	cacheredis "github.com/Altusha4/microservice/notification-service/internal/infrastructure/redis"
	"github.com/Altusha4/microservice/notification-service/internal/repository/postgres"
	"github.com/Altusha4/microservice/notification-service/internal/usecase"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	goredis "github.com/redis/go-redis/v9"
)

func main() {
	_ = godotenv.Load()

	dbDSN := getEnv("NOTIFICATION_DB_DSN", "postgres://postgres:postgres@localhost:5434/notification_db?sslmode=disable")
	rabbitmqURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnvInt("REDIS_DB", 0)
	maxRetries := getEnvInt("EMAIL_MAX_RETRIES", 3)
	backoffBaseMs := getEnvInt("EMAIL_BACKOFF_BASE_MS", 2000)
	idempotencyTTL := 24 * time.Hour

	// ##############################
	// Postgres
	// ##############################
	db, err := sql.Open("postgres", dbDSN)
	if err != nil {
		log.Fatalf("open notification_db: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("connect to notification_db: %v", err)
	}

	// ##############################
	// Redis
	// ##############################
	redisClient := goredis.NewClient(&goredis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("connect to redis: %v", err)
	}
	log.Printf("[notification] connected to redis at %s (idempotency TTL %s)", redisAddr, idempotencyTTL)

	// ##############################
	// Email sender
	// ##############################
	sender, err := email.Build()
	if err != nil {
		log.Fatalf("build email sender: %v", err)
	}

	// ##############################
	// Wiring
	// ##############################
	idempotencyRepo := postgres.NewIdempotencyRepo(db)
	idempotencyCache := cacheredis.NewIdempotencyCache(redisClient, idempotencyTTL)

	notifUC := usecase.NewNotificationUseCase(
		idempotencyRepo,
		idempotencyCache,
		sender,
		usecase.RetryConfig{
			MaxAttempts: maxRetries,
			BaseDelay:   time.Duration(backoffBaseMs) * time.Millisecond,
		},
	)

	// ##############################
	// RabbitMQ
	// ##############################
	amqpConn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		log.Fatalf("connect to rabbitmq: %v", err)
	}
	defer amqpConn.Close()

	consumer, err := rabbitmq.NewConsumer(amqpConn)
	if err != nil {
		log.Fatalf("create consumer: %v", err)
	}
	defer consumer.Close()

	log.Println("[notification] connected to rabbitmq, consumer ready")

	// ##############################
	// Graceful shutdown
	// ##############################
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := consumer.Run(ctx, notifUC.HandlePaymentCompleted); err != nil {
		log.Printf("[notification] consumer stopped: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
