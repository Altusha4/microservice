package app

import (
	"context"

	pb "github.com/Altusha4/ap2-generated/payment"
	"github.com/Altusha4/microservice/order-service/internal/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type GRPCPaymentClient struct {
	client pb.PaymentServiceClient
}

func NewGRPCPaymentClient(addr string) (*GRPCPaymentClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &GRPCPaymentClient{client: pb.NewPaymentServiceClient(conn)}, nil
}

func (c *GRPCPaymentClient) ProcessPayment(ctx context.Context, req usecase.PaymentRequest) (*usecase.PaymentResponse, error) {
	resp, err := c.client.ProcessPayment(ctx, &pb.PaymentRequest{
		OrderId: req.OrderID,
		Amount:  req.Amount,
	})
	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() == codes.InvalidArgument {
			return nil, usecase.ErrInvalidAmount
		}
		return nil, usecase.ErrPaymentServiceUnavailable
	}
	return &usecase.PaymentResponse{
		Status:        resp.Status,
		TransactionID: resp.TransactionId,
	}, nil
}
