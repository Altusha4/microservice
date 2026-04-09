package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	pborder "github.com/Altusha4/ap2-generated/order"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <order-id>")
	}
	orderID := os.Args[1]

	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	client := pborder.NewOrderServiceClient(conn)
	stream, err := client.SubscribeToOrderUpdates(context.Background(), &pborder.OrderRequest{
		OrderId: orderID,
	})
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	fmt.Printf("Subscribed to order %s updates...\n", orderID)
	for {
		update, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("Stream closed")
			break
		}
		if err != nil {
			log.Fatalf("recv error: %v", err)
		}
		fmt.Printf("[UPDATE] order_id=%s | status=%s | time=%s\n",
			update.OrderId, update.Status, update.UpdatedAt.AsTime().Format("15:04:05"))
	}
}
