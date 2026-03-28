package main

import (
	"log"
	"os"

	"github.com/Altusha4/microservice/payment-service/internal/app"
	"github.com/Altusha4/microservice/payment-service/internal/repository/postgres"
	transporthttp "github.com/Altusha4/microservice/payment-service/internal/transport/http"
	"github.com/Altusha4/microservice/payment-service/internal/usecase"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// ── Configuration from environment ──────────────────────────────────────
	dsn := getEnv("PAYMENT_DB_DSN", "postgres://postgres:postgres@localhost:5434/payment_db?sslmode=disable")
	port := getEnv("PORT", "8081")

	// ── Database ─────────────────────────────────────────────────────────────
	db, err := app.OpenDB(dsn)
	if err != nil {
		log.Fatalf("connect to payment_db: %v", err)
	}
	defer db.Close()

	// ── Dependency Injection (manual composition root) ────────────────────────
	paymentRepo := postgres.NewPaymentRepo(db)
	paymentUC := usecase.NewPaymentUseCase(paymentRepo)
	handler := transporthttp.NewHandler(paymentUC)

	// ── HTTP Server ───────────────────────────────────────────────────────────
	r := gin.Default()
	handler.RegisterRoutes(r)

	log.Printf("payment-service listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("payment-service server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
