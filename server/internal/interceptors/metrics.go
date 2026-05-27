package interceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// MetricsCollector is a minimal interface for gRPC metrics.
type MetricsCollector interface {
	RecordRPC(method string, code string, duration time.Duration)
}

// InMemoryMetrics is a basic in-memory metrics collector.
type InMemoryMetrics struct{}

func (m *InMemoryMetrics) RecordRPC(method string, code string, duration time.Duration) {}

func UnaryMetrics(mc MetricsCollector) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		st, _ := status.FromError(err)
		mc.RecordRPC(info.FullMethod, st.Code().String(), time.Since(start))
		return resp, err
	}
}
