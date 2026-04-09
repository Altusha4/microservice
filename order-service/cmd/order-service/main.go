package main

import (
	"log"
	"net"
	"os"

	pborder "github.com/Altusha4/ap2-generated/order"
	"github.com/Altusha4/microservice/order-service/internal/app"
	"github.com/Altusha4/microservice/order-service/internal/repository/postgres"
	transportgrpc "github.com/Altusha4/microservice/order-service/internal/transport/grpc"
	transporthttp "github.com/Altusha4/microservice/order-service/internal/transport/http"
	"github.com/Altusha4/microservice/order-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

func main() {
	_ = godotenv.Load()

	dsn := getEnv("ORDER_DB_DSN", "postgres://postgres:postgres@localhost:5433/order_db?sslmode=disable")
	httpPort := getEnv("HTTP_PORT", "8080")
	grpcPort := getEnv("GRPC_PORT", "50052")
	paymentGRPCAddr := getEnv("PAYMENT_GRPC_ADDR", "localhost:50051")

	db, err := app.OpenDB(dsn)
	if err != nil {
		log.Fatalf("connect to order_db: %v", err)
	}
	defer db.Close()

	orderRepo := postgres.NewOrderRepo(db)
	paymentClient, err := app.NewGRPCPaymentClient(paymentGRPCAddr)
	if err != nil {
		log.Fatalf("connect to payment gRPC: %v", err)
	}

	orderUC := usecase.NewOrderUseCase(orderRepo, paymentClient)

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

	handler := transporthttp.NewHandler(orderUC)
	r := gin.Default()
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
