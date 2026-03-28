package http

import (
	"errors"
	"net/http"

	"github.com/Altusha4/microservice/payment-service/internal/domain"
	"github.com/Altusha4/microservice/payment-service/internal/usecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	paymentUC *usecase.PaymentUseCase
}

func NewHandler(paymentUC *usecase.PaymentUseCase) *Handler {
	return &Handler{paymentUC: paymentUC}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/payments", h.ProcessPayment)
	r.GET("/payments/:order_id", h.GetPayment)
}

type processPaymentRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Amount  int64  `json:"amount"   binding:"required,gt=0"`
}

type paymentResponse struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id,omitempty"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
}

func toPaymentResponse(p *domain.Payment) paymentResponse {
	return paymentResponse{
		ID:            p.ID,
		OrderID:       p.OrderID,
		TransactionID: p.TransactionID,
		Amount:        p.Amount,
		Status:        p.Status,
	}
}

func (h *Handler) ProcessPayment(c *gin.Context) {
	var req processPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, err := h.paymentUC.ProcessPayment(c.Request.Context(), req.OrderID, req.Amount)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidAmount) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toPaymentResponse(payment))
}

func (h *Handler) GetPayment(c *gin.Context) {
	orderID := c.Param("order_id")
	payment, err := h.paymentUC.GetPaymentByOrderID(c.Request.Context(), orderID)
	if err != nil {
		if errors.Is(err, usecase.ErrPaymentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPaymentResponse(payment))
}
