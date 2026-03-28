package main

import (
	"log"
	"os"

	"github.com/Altusha4/microservice/order-service/internal/app"
	"github.com/Altusha4/microservice/order-service/internal/repository/postgres"
	transporthttp "github.com/Altusha4/microservice/order-service/internal/transport/http"
	"github.com/Altusha4/microservice/order-service/internal/usecase"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// ── Configuration from environment ──────────────────────────────────────
	dsn := getEnv("ORDER_DB_DSN", "postgres://postgres:postgres@localhost:5433/order_db?sslmode=disable")
	paymentServiceURL := getEnv("PAYMENT_SERVICE_URL", "http://localhost:8081")
	port := getEnv("PORT", "8080")

	// ── Database ─────────────────────────────────────────────────────────────
	db, err := app.OpenDB(dsn)
	if err != nil {
		log.Fatalf("connect to order_db: %v", err)
	}
	defer db.Close()

	// ── Dependency Injection (manual composition root) ────────────────────────
	orderRepo := postgres.NewOrderRepo(db)
	paymentClient := app.NewHTTPPaymentClient(paymentServiceURL)
	orderUC := usecase.NewOrderUseCase(orderRepo, paymentClient)
	handler := transporthttp.NewHandler(orderUC)

	// ── HTTP Server ───────────────────────────────────────────────────────────
	r := gin.Default()
	handler.RegisterRoutes(r)

	log.Printf("order-service listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("order-service server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
