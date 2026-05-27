package interceptors

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestUnaryRateLimit_AllowsUnderLimit(t *testing.T) {
	interceptor := UnaryRateLimit(100, 1)
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUnaryRateLimit_BlocksOverLimit(t *testing.T) {
	interceptor := UnaryRateLimit(0, 0) // no capacity
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	st, _ := status.FromError(err)
	if st.Code() != codes.ResourceExhausted {
		t.Fatalf("expected ResourceExhausted, got %v", st.Code())
	}
}

func TestUnaryRecovery_CatchesPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	interceptor := UnaryRecovery(logger)
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("boom")
	}
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test/Panic"}, handler)
	st, _ := status.FromError(err)
	if st.Code() != codes.Internal {
		t.Fatalf("expected Internal, got %v", st.Code())
	}
}

func TestUnaryRecovery_PassesThroughNormal(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	interceptor := UnaryRecovery(logger)
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}
	resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp != "ok" {
		t.Fatalf("expected ok, got %v", resp)
	}
}

func TestUnaryRequestID_PropagatesFromMetadata(t *testing.T) {
	interceptor := UnaryRequestID()
	md := metadata.Pairs("x-request-id", "abc-123")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		id := RequestIDFromContext(ctx)
		if id != "abc-123" {
			t.Fatalf("expected request-id abc-123, got %s", id)
		}
		return id, nil
	}
	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "abc-123" {
		t.Fatalf("expected abc-123, got %v", resp)
	}
}

func TestUnaryRequestID_GeneratesWhenMissing(t *testing.T) {
	interceptor := UnaryRequestID()
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		id := RequestIDFromContext(ctx)
		if id == "" {
			t.Fatal("expected generated request-id, got empty")
		}
		return id, nil
	}
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnaryMetrics_RecordsRPC(t *testing.T) {
	mc := &recordingMetrics{}
	interceptor := UnaryMetrics(mc)
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test/Method"}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mc.method != "/test/Method" {
		t.Fatalf("expected method /test/Method, got %s", mc.method)
	}
	if mc.code != "OK" {
		t.Fatalf("expected code OK, got %s", mc.code)
	}
	if mc.duration <= 0 {
		t.Fatal("expected positive duration")
	}
}

func TestUnaryMetrics_RecordsErrorCode(t *testing.T) {
	mc := &recordingMetrics{}
	interceptor := UnaryMetrics(mc)
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, status.Error(codes.InvalidArgument, "bad")
	}
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test/Method"}, handler)
	if err == nil {
		t.Fatal("expected error")
	}
	if mc.code != "InvalidArgument" {
		t.Fatalf("expected code InvalidArgument, got %s", mc.code)
	}
}

type recordingMetrics struct {
	method   string
	code     string
	duration time.Duration
}

func (m *recordingMetrics) RecordRPC(method, code string, duration time.Duration) {
	m.method = method
	m.code = code
	m.duration = duration
}

func TestInMemoryMetrics_NoOp(t *testing.T) {
	var m InMemoryMetrics
	m.RecordRPC("/test/Method", "OK", time.Millisecond)
}

func TestRequestIDFromContext_NotPresent(t *testing.T) {
	if id := RequestIDFromContext(context.Background()); id != "" {
		t.Fatalf("expected empty, got %s", id)
	}
}

type fakeValidator struct{ err error }

func (v *fakeValidator) Validate() error { return v.err }

func TestUnaryValidate_PassesValid(t *testing.T) {
	interceptor := UnaryValidate()
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}
	_, err := interceptor(context.Background(), &fakeValidator{err: nil}, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUnaryValidate_BlocksInvalid(t *testing.T) {
	interceptor := UnaryValidate()
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}
	_, err := interceptor(context.Background(), &fakeValidator{err: errors.New("bad")}, &grpc.UnaryServerInfo{}, handler)
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestUnaryAuth_NoneMode(t *testing.T) {
	interceptor := UnaryAuth(AuthConfig{Mode: "none"})
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}
	resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp != "ok" {
		t.Fatalf("expected ok, got %v", resp)
	}
}

func TestUnaryAuth_TokenValid(t *testing.T) {
	interceptor := UnaryAuth(AuthConfig{
		Mode:          "token",
		AllowedTokens: map[string]bool{"valid-token": true},
	})
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}
	md := metadata.Pairs("authorization", "Bearer valid-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	resp, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp != "ok" {
		t.Fatalf("expected ok, got %v", resp)
	}
}

func TestUnaryAuth_TokenInvalid(t *testing.T) {
	interceptor := UnaryAuth(AuthConfig{
		Mode:          "token",
		AllowedTokens: map[string]bool{"valid-token": true},
	})
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}
	md := metadata.Pairs("authorization", "Bearer invalid-token")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
	st, _ := status.FromError(err)
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", st.Code())
	}
}

func TestUnaryAuth_MissingMetadata(t *testing.T) {
	interceptor := UnaryAuth(AuthConfig{
		Mode:          "token",
		AllowedTokens: map[string]bool{"valid-token": true},
	})
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	st, _ := status.FromError(err)
	if st.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", st.Code())
	}
}

func TestUnaryAuth_SpireNotImplemented(t *testing.T) {
	interceptor := UnaryAuth(AuthConfig{Mode: "spire", SPIREAddr: "/tmp/spire.sock"})
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	st, _ := status.FromError(err)
	if st.Code() != codes.Unimplemented {
		t.Fatalf("expected Unimplemented, got %v", st.Code())
	}
}

func TestUnaryAuth_UnknownMode(t *testing.T) {
	interceptor := UnaryAuth(AuthConfig{Mode: "weird"})
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	st, _ := status.FromError(err)
	if st.Code() != codes.Internal {
		t.Fatalf("expected Internal, got %v", st.Code())
	}
}
