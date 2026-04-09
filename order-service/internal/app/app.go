package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Altusha4/microservice/order-service/internal/usecase"
)

// #######################################
// PAYMENT CLIENT
// #######################################

type HTTPPaymentClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPPaymentClient(baseURL string) *HTTPPaymentClient {
	return &HTTPPaymentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

type paymentRequestBody struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

type paymentResponseBody struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

func (c *HTTPPaymentClient) ProcessPayment(ctx context.Context, req usecase.PaymentRequest) (*usecase.PaymentResponse, error) {
	body, err := json.Marshal(paymentRequestBody{OrderID: req.OrderID, Amount: req.Amount})
	if err != nil {
		return nil, fmt.Errorf("marshal payment request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/payments", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build payment http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call payment service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("payment service returned status %d", resp.StatusCode)
	}

	var respBody paymentResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, fmt.Errorf("decode payment response: %w", err)
	}

	return &usecase.PaymentResponse{
		Status:        respBody.Status,
		TransactionID: respBody.TransactionID,
	}, nil
}

// #######################################
// DATABASE UTILS
// #######################################

func OpenDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}
