package http

import (
	"errors"
	"net/http"

	"github.com/Altusha4/microservice/order-service/internal/domain"
	"github.com/Altusha4/microservice/order-service/internal/usecase"
	"github.com/gin-gonic/gin"
)

// Handler holds references to use cases for thin Gin handlers.
type Handler struct {
	orderUC *usecase.OrderUseCase
}

// NewHandler creates a new HTTP handler.
func NewHandler(orderUC *usecase.OrderUseCase) *Handler {
	return &Handler{orderUC: orderUC}
}

// RegisterRoutes wires up all order-service routes on the given engine.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/orders", h.CreateOrder)
	r.GET("/orders/:id", h.GetOrder)
	r.PATCH("/orders/:id/cancel", h.CancelOrder)
}

// createOrderRequest is the parsed JSON body for POST /orders.
type createOrderRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	ItemName   string `json:"item_name"   binding:"required"`
	Amount     int64  `json:"amount"      binding:"required,gt=0"`
}

// orderResponse is the JSON response for order operations.
type orderResponse struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	ItemName   string `json:"item_name"`
	Amount     int64  `json:"amount"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

func toOrderResponse(o *domain.Order) orderResponse {
	return orderResponse{
		ID:         o.ID,
		CustomerID: o.CustomerID,
		ItemName:   o.ItemName,
		Amount:     o.Amount,
		Status:     o.Status,
		CreatedAt:  o.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// CreateOrder handles POST /orders.
func (h *Handler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")

	order, err := h.orderUC.CreateOrder(c.Request.Context(), req.CustomerID, req.ItemName, req.Amount, idempotencyKey)
	if err != nil {
		if errors.Is(err, usecase.ErrPaymentServiceUnavailable) {
			resp := gin.H{"error": "payment service unavailable"}
			if order != nil {
				resp["order"] = toOrderResponse(order)
			}
			c.JSON(http.StatusServiceUnavailable, resp)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toOrderResponse(order))
}

// GetOrder handles GET /orders/:id.
func (h *Handler) GetOrder(c *gin.Context) {
	id := c.Param("id")
	order, err := h.orderUC.GetOrder(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, usecase.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toOrderResponse(order))
}

// CancelOrder handles PATCH /orders/:id/cancel.
func (h *Handler) CancelOrder(c *gin.Context) {
	id := c.Param("id")
	err := h.orderUC.CancelOrder(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrOrderNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		case errors.Is(err, usecase.ErrCannotCancelPaidOrder):
			c.JSON(http.StatusConflict, gin.H{"error": "paid orders cannot be cancelled"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "order cancelled"})
}
