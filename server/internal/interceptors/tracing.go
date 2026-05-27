package interceptors

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

// UnaryTracing creates a gRPC unary interceptor that starts an OpenTelemetry span
// for each RPC and records the method name and outcome.
func UnaryTracing() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("xoscal")
		ctx, span := tracer.Start(ctx, info.FullMethod)
		defer span.End()

		span.SetAttributes(attribute.String("rpc.method", info.FullMethod))

		resp, err := handler(ctx, req)
		if err != nil {
			span.RecordError(err)
			span.SetAttributes(attribute.String("rpc.status", "error"))
		} else {
			span.SetAttributes(attribute.String("rpc.status", "ok"))
		}
		return resp, err
	}
}
