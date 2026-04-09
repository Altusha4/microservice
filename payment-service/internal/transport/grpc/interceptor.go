package grpc

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

func LoggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	log.Printf("[gRPC] method: %s | started", info.FullMethod)

	resp, err := handler(ctx, req)

	duration := time.Since(start)
	if err != nil {
		log.Printf("[gRPC] method: %s | duration: %v | error: %v", info.FullMethod, duration, err)
	} else {
		log.Printf("[gRPC] method: %s | duration: %v | success", info.FullMethod, duration)
	}

	return resp, err
}
