package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Altusha4/microservice/notification-service/internal/infrastructure/rabbitmq"
	"github.com/Altusha4/microservice/notification-service/internal/repository/postgres"
	"github.com/Altusha4/microservice/notification-service/internal/usecase"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	_ = godotenv.Load()

	dsn := getEnv("NOTIFICATION_DB_DSN", "postgres://postgres:postgres@localhost:5435/notification_db?sslmode=disable")
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	// ##############################
	// DB
	// ##############################
	db, err := openDB(dsn)
	if err != nil {
		log.Fatalf("connect notification_db: %v", err)
	}
	defer db.Close()

	idempotencyRepo := postgres.NewIdempotencyRepo(db)
	notificationUC := usecase.NewNotificationUseCase(idempotencyRepo)

	// ##############################
	// RabbitMQ
	// ##############################
	rabbitConn, err := dialRabbit(rabbitURL, 30, 2*time.Second)
	if err != nil {
		log.Fatalf("connect rabbitmq: %v", err)
	}
	consumer, err := rabbitmq.NewConsumer(rabbitConn)
	if err != nil {
		log.Fatalf("create consumer: %v", err)
	}
	log.Println("[notification] connected to rabbitmq, consumer ready")

	// ##############################
	// Graceful shutdown wiring
	// ##############################
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Run consumer in current goroutine — it blocks until ctx is done.
	go func() {
		<-ctx.Done()
		log.Println("[notification] shutdown signal received")
	}()

	if err := consumer.Run(ctx, notificationUC.HandlePaymentCompleted); err != nil {
		log.Printf("[notification] consumer exited: %v", err)
	}

	if err := consumer.Close(); err != nil {
		log.Printf("[notification] consumer close error: %v", err)
	}
	log.Println("[notification] shutdown complete")
}

// ##############################
// Helpers
// ##############################

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

func dialRabbit(url string, attempts int, backoff time.Duration) (*amqp.Connection, error) {
	var lastErr error
	for i := 1; i <= attempts; i++ {
		conn, err := amqp.Dial(url)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		log.Printf("[notification] rabbitmq dial attempt %d/%d failed: %v", i, attempts, err)
		time.Sleep(backoff)
	}
	return nil, lastErr
}
