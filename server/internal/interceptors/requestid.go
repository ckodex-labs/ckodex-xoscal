package interceptors

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const requestIDHeader = "x-request-id"

type requestIDKey struct{}

// RequestIDFromContext returns the request ID stored in ctx, or empty string.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// UnaryRequestID injects or propagates a request-id through gRPC metadata and context.
func UnaryRequestID() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get(requestIDHeader); len(vals) > 0 && vals[0] != "" {
				ctx = context.WithValue(ctx, requestIDKey{}, vals[0])
				return handler(ctx, req)
			}
		}
		// If no incoming request-id, generate a short one (caller may overwrite).
		id := generateID()
		ctx = context.WithValue(ctx, requestIDKey{}, id)
		return handler(ctx, req)
	}
}

func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
