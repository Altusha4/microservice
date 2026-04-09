package main

import (
	"log"
	"net"
	"os"

	pb "github.com/Altusha4/ap2-generated/payment"
	"github.com/Altusha4/microservice/payment-service/internal/app"
	"github.com/Altusha4/microservice/payment-service/internal/repository/postgres"
	transportgrpc "github.com/Altusha4/microservice/payment-service/internal/transport/grpc"
	transporthttp "github.com/Altusha4/microservice/payment-service/internal/transport/http"
	"github.com/Altusha4/microservice/payment-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

func main() {
	_ = godotenv.Load()

	dsn := getEnv("PAYMENT_DB_DSN", "postgres://postgres:postgres@localhost:5434/payment_db?sslmode=disable")
	grpcPort := getEnv("GRPC_PORT", "50051")
	httpPort := getEnv("HTTP_PORT", "8081")

	db, err := app.OpenDB(dsn)
	if err != nil {
		log.Fatalf("connect to payment_db: %v", err)
	}
	defer db.Close()

	paymentRepo := postgres.NewPaymentRepo(db)
	paymentUC := usecase.NewPaymentUseCase(paymentRepo)

	go func() {
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatalf("failed to listen grpc: %v", err)
		}
		grpcServer := grpc.NewServer(
			grpc.UnaryInterceptor(transportgrpc.LoggingInterceptor),
		)
		pb.RegisterPaymentServiceServer(grpcServer, transportgrpc.NewPaymentGRPCHandler(paymentUC))
		log.Printf("payment-service gRPC listening on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("grpc serve error: %v", err)
		}
	}()

	handler := transporthttp.NewHandler(paymentUC)
	r := gin.Default()
	handler.RegisterRoutes(r)

	log.Printf("payment-service HTTP listening on :%s", httpPort)
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
