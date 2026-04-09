package grpc

import (
	"context"

	pb "github.com/Altusha4/ap2-generated/payment"
	"github.com/Altusha4/microservice/payment-service/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PaymentGRPCHandler struct {
	pb.UnimplementedPaymentServiceServer
	paymentUC *usecase.PaymentUseCase
}

func NewPaymentGRPCHandler(uc *usecase.PaymentUseCase) *PaymentGRPCHandler {
	return &PaymentGRPCHandler{paymentUC: uc}
}

func (h *PaymentGRPCHandler) ProcessPayment(ctx context.Context, req *pb.PaymentRequest) (*pb.PaymentResponse, error) {
	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be greater than 0")
	}

	payment, err := h.paymentUC.ProcessPayment(ctx, req.OrderId, req.Amount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "process payment: %v", err)
	}

	return &pb.PaymentResponse{
		TransactionId: payment.TransactionID,
		Status:        payment.Status,
		CreatedAt:     timestamppb.Now(),
	}, nil
}
