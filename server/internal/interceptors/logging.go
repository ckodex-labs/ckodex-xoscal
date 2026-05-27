package interceptors

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func UnaryLogging(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		dur := time.Since(start)
		st, _ := status.FromError(err)

		attrs := []slog.Attr{
			slog.String("method", info.FullMethod),
			slog.Duration("duration", dur),
			slog.String("code", st.Code().String()),
		}
		if rid := RequestIDFromContext(ctx); rid != "" {
			attrs = append(attrs, slog.String("request_id", rid))
		}
		if err != nil {
			logger.LogAttrs(ctx, slog.LevelError, "gRPC request failed", append(attrs, slog.String("error", err.Error()))...)
		} else {
			logger.LogAttrs(ctx, slog.LevelInfo, "gRPC request", attrs...)
		}
		return resp, err
	}
}
