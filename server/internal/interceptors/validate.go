package interceptors

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Validator is implemented by request messages that can validate themselves.
type Validator interface {
	Validate() error
}

// UnaryValidate returns a gRPC interceptor that calls Validate() on requests
// implementing the Validator interface.
func UnaryValidate() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if v, ok := req.(Validator); ok {
			if err := v.Validate(); err != nil {
				return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("validation failed: %v", err))
			}
		}
		return handler(ctx, req)
	}
}
