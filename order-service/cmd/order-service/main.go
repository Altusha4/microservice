package main

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	pborder "github.com/Altusha4/ap2-generated/order"
	"github.com/Altusha4/microservice/order-service/internal/app"
	cacheredis "github.com/Altusha4/microservice/order-service/internal/infrastructure/redis"
	"github.com/Altusha4/microservice/order-service/internal/repository/postgres"
	transportgrpc "github.com/Altusha4/microservice/order-service/internal/transport/grpc"
	transporthttp "github.com/Altusha4/microservice/order-service/internal/transport/http"
	"github.com/Altusha4/microservice/order-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	goredis "github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func main() {
	_ = godotenv.Load()

	dsn := getEnv("ORDER_DB_DSN", "postgres://postgres:postgres@localhost:5433/order_db?sslmode=disable")
	httpPort := getEnv("HTTP_PORT", "8080")
	grpcPort := getEnv("GRPC_PORT", "50052")
	paymentGRPCAddr := getEnv("PAYMENT_GRPC_ADDR", "localhost:50051")

	// Assignment 4 — Redis cache config
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnvInt("REDIS_DB", 0)
	cacheTTL := time.Duration(getEnvInt("ORDER_CACHE_TTL_SECONDS", 300)) * time.Second

	// Assignment 4 bonus — rate limiter config
	rateLimit := getEnvInt("RATE_LIMIT_PER_MIN", 10)

	// ##############################
	// DB
	// ##############################
	db, err := app.OpenDB(dsn)
	if err != nil {
		log.Fatalf("connect to order_db: %v", err)
	}
	defer db.Close()

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
	log.Printf("[order] connected to redis at %s (cache TTL %s)", redisAddr, cacheTTL)

	orderCache := cacheredis.NewOrderCache(redisClient, cacheTTL)

	// ##############################
	// Wiring
	// ##############################
	orderRepo := postgres.NewOrderRepo(db)
	paymentClient, err := app.NewGRPCPaymentClient(paymentGRPCAddr)
	if err != nil {
		log.Fatalf("connect to payment gRPC: %v", err)
	}

	orderUC := usecase.NewOrderUseCase(orderRepo, paymentClient, orderCache)

	// ##############################
	// gRPC server (status streaming)
	// ##############################
	go func() {
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatalf("failed to listen grpc: %v", err)
		}
		grpcServer := grpc.NewServer()
		pborder.RegisterOrderServiceServer(grpcServer, transportgrpc.NewOrderGRPCHandler(orderRepo))
		log.Printf("order-service gRPC listening on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc serve error: %v", err)
		}
	}()

	// ##############################
	// HTTP server with rate limiter (Assignment 4 bonus)
	// ##############################
	rateLimiter := transporthttp.NewRateLimiter(redisClient, rateLimit, time.Minute)
	log.Printf("[order] rate limiter: %d requests per minute", rateLimit)

	handler := transporthttp.NewHandler(orderUC)
	r := gin.Default()
	r.Use(rateLimiter.Middleware())
	handler.RegisterRoutes(r)

	log.Printf("order-service HTTP listening on :%s", httpPort)
	if err := r.Run(":" + httpPort); err != nil {
		log.Fatalf("http server error: %v", err)
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
