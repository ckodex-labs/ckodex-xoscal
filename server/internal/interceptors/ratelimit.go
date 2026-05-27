package interceptors

import (
	"context"
	"net"
	"sync"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// perClientLimiter tracks rate limiters per client address.
type perClientLimiter struct {
	rps      float64
	burst    int
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
}

func newPerClientLimiter(rps float64, burst int) *perClientLimiter {
	return &perClientLimiter{
		rps:      rps,
		burst:    burst,
		limiters: make(map[string]*rate.Limiter),
	}
}

func (pcl *perClientLimiter) allow(addr string) bool {
	pcl.mu.RLock()
	lim, ok := pcl.limiters[addr]
	pcl.mu.RUnlock()
	if ok {
		return lim.Allow()
	}

	pcl.mu.Lock()
	defer pcl.mu.Unlock()
	if lim, ok := pcl.limiters[addr]; ok {
		return lim.Allow()
	}
	lim = rate.NewLimiter(rate.Limit(pcl.rps), pcl.burst)
	pcl.limiters[addr] = lim
	return lim.Allow()
}

func clientAddr(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return p.Addr.String()
	}
	return host
}

// UnaryRateLimit returns an interceptor that throttles requests per client with a token bucket.
func UnaryRateLimit(rps float64, burst int) grpc.UnaryServerInterceptor {
	pcl := newPerClientLimiter(rps, burst)
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !pcl.allow(clientAddr(ctx)) {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}
