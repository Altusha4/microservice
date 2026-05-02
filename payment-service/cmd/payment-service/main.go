package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/Altusha4/ap2-generated/payment"
	"github.com/Altusha4/microservice/payment-service/internal/app"
	"github.com/Altusha4/microservice/payment-service/internal/infrastructure/rabbitmq"
	"github.com/Altusha4/microservice/payment-service/internal/repository/postgres"
	transportgrpc "github.com/Altusha4/microservice/payment-service/internal/transport/grpc"
	transporthttp "github.com/Altusha4/microservice/payment-service/internal/transport/http"
	"github.com/Altusha4/microservice/payment-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

func main() {
	_ = godotenv.Load()

	dsn := getEnv("PAYMENT_DB_DSN", "postgres://postgres:postgres@localhost:5434/payment_db?sslmode=disable")
	grpcPort := getEnv("GRPC_PORT", "50051")
	httpPort := getEnv("HTTP_PORT", "8081")
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	// ##############################
	// DB
	// ##############################
	db, err := app.OpenDB(dsn)
	if err != nil {
		log.Fatalf("connect to payment_db: %v", err)
	}
	defer db.Close()

	// ##############################
	// RabbitMQ — wait for broker
	// ##############################
	rabbitConn, err := dialRabbit(rabbitURL, 30, 2*time.Second)
	if err != nil {
		log.Fatalf("connect to rabbitmq: %v", err)
	}
	publisher, err := rabbitmq.NewPublisher(rabbitConn)
	if err != nil {
		log.Fatalf("create publisher: %v", err)
	}
	log.Println("[payment] connected to rabbitmq, publisher ready")

	// ##############################
	// Wiring
	// ##############################
	paymentRepo := postgres.NewPaymentRepo(db)
	paymentUC := usecase.NewPaymentUseCase(paymentRepo, publisher)

	// ##############################
	// gRPC server
	// ##############################
	grpcLis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("listen grpc: %v", err)
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(transportgrpc.LoggingInterceptor),
	)
	pb.RegisterPaymentServiceServer(grpcServer, transportgrpc.NewPaymentGRPCHandler(paymentUC))

	go func() {
		log.Printf("payment-service gRPC listening on :%s", grpcPort)
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Printf("grpc serve error: %v", err)
		}
	}()

	// ##############################
	// HTTP server
	// ##############################
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	transporthttp.NewHandler(paymentUC).RegisterRoutes(r)

	httpServer := &http.Server{
		Addr:    ":" + httpPort,
		Handler: r,
	}
	go func() {
		log.Printf("payment-service HTTP listening on :%s", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http serve error: %v", err)
		}
	}()

	// ##############################
	// Graceful shutdown
	// ##############################
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	log.Println("[payment] shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}
	grpcServer.GracefulStop()
	if err := publisher.Close(); err != nil {
		log.Printf("publisher close error: %v", err)
	}
	log.Println("[payment] shutdown complete")
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

// dialRabbit retries the AMQP dial — RabbitMQ in docker-compose
// usually needs a few seconds to be ready even with healthcheck.
func dialRabbit(url string, attempts int, backoff time.Duration) (*amqp.Connection, error) {
	var lastErr error
	for i := 1; i <= attempts; i++ {
		conn, err := amqp.Dial(url)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		log.Printf("[payment] rabbitmq dial attempt %d/%d failed: %v", i, attempts, err)
		time.Sleep(backoff)
	}
	return nil, lastErr
}
