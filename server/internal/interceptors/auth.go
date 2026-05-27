package interceptors

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthConfig holds authentication settings.
type AuthConfig struct {
	Mode          string // "none", "token", "spire"
	SPIREAddr     string // Unix socket for SPIRE agent
	AllowedTokens map[string]bool
}

// UnaryAuth returns an interceptor that enforces authentication based on mode.
func UnaryAuth(cfg AuthConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		switch cfg.Mode {
		case "none", "":
			return handler(ctx, req)
		case "token":
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				return nil, status.Error(codes.Unauthenticated, "missing metadata")
			}
			tokens := md.Get("authorization")
			if len(tokens) == 0 {
				return nil, status.Error(codes.Unauthenticated, "missing authorization token")
			}
			token := tokens[0]
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}
			if !cfg.AllowedTokens[token] {
				return nil, status.Error(codes.PermissionDenied, "invalid token")
			}
			return handler(ctx, req)
		case "spire":
			// SPIRE authentication would verify SVID here.
			// This is a stub; full implementation needs SPIRE workload API.
			return nil, status.Errorf(codes.Unimplemented, "spire auth not yet implemented (socket: %s)", cfg.SPIREAddr)
		default:
			return nil, status.Errorf(codes.Internal, "unknown auth mode: %s", cfg.Mode)
		}
	}
}
