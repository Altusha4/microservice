package grpc

import (
	"log"
	"time"

	pb "github.com/Altusha4/ap2-generated/order"
	"github.com/Altusha4/microservice/order-service/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderGRPCHandler struct {
	pb.UnimplementedOrderServiceServer
	repo repository.OrderRepository
}

func NewOrderGRPCHandler(repo repository.OrderRepository) *OrderGRPCHandler {
	return &OrderGRPCHandler{repo: repo}
}

func (h *OrderGRPCHandler) SubscribeToOrderUpdates(req *pb.OrderRequest, stream pb.OrderService_SubscribeToOrderUpdatesServer) error {
	if req.OrderId == "" {
		return status.Error(codes.InvalidArgument, "order_id is required")
	}

	var lastStatus string

	for {
		select {
		case <-stream.Context().Done():
			log.Printf("[streaming] client disconnected for order %s", req.OrderId)
			return nil
		default:
		}

		order, err := h.repo.GetByID(stream.Context(), req.OrderId)
		if err != nil {
			return status.Errorf(codes.Internal, "get order: %v", err)
		}
		if order == nil {
			return status.Error(codes.NotFound, "order not found")
		}

		if order.Status != lastStatus {
			lastStatus = order.Status
			update := &pb.OrderStatusUpdate{
				OrderId:   order.ID,
				Status:    order.Status,
				UpdatedAt: timestamppb.New(time.Now()),
			}
			if err := stream.Send(update); err != nil {
				return status.Errorf(codes.Internal, "send stream: %v", err)
			}
			log.Printf("[streaming] sent status '%s' for order %s", order.Status, order.ID)

			if order.Status == "Paid" || order.Status == "Failed" || order.Status == "Cancelled" {
				return nil
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}
