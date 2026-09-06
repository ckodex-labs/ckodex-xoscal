package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGatewayTokenUsesFirstNonCommentLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("# managed\n\n beta-token \nsecond-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	token, err := loadGatewayToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if token != "beta-token" {
		t.Fatalf("token = %q, want beta-token", token)
	}
}

func TestLoadGatewayTokenRejectsEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("# no token\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := loadGatewayToken(path); err == nil {
		t.Fatal("loadGatewayToken accepted a file without a token")
	}
}

func TestGRPCDialOptionsFailClosedForTokenWithoutTLS(t *testing.T) {
	if _, err := grpcDialOptions("", "", "", "", "beta-token"); err == nil {
		t.Fatal("grpcDialOptions allowed a token over an unconfigured transport")
	}
}

func TestGRPCDialOptionsAllowLocalUnauthenticatedMode(t *testing.T) {
	opts, err := grpcDialOptions("", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) != 1 {
		t.Fatalf("dial options = %d, want one local transport option", len(opts))
	}
}

func TestGRPCDialOptionsBindTokenToTLS(t *testing.T) {
	opts, err := grpcDialOptions("", "", "", "localhost", "beta-token")
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) != 2 {
		t.Fatalf("dial options = %d, want TLS plus token credentials", len(opts))
	}

	metadata, err := (staticTokenCredentials{token: "beta-token"}).GetRequestMetadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if metadata["authorization"] != "Bearer beta-token" {
		t.Fatalf("authorization metadata = %q", metadata["authorization"])
	}
	if !(staticTokenCredentials{token: "beta-token"}).RequireTransportSecurity() {
		t.Fatal("static token credentials do not require transport security")
	}
}

func TestWithTokenAuth(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := withTokenAuth(next, "beta-token")

	for _, tc := range []struct {
		name       string
		authorize  string
		wantStatus int
	}{
		{name: "missing", wantStatus: http.StatusUnauthorized},
		{name: "invalid", authorize: "Bearer wrong", wantStatus: http.StatusUnauthorized},
		{name: "valid bearer", authorize: "Bearer beta-token", wantStatus: http.StatusNoContent},
		{name: "valid raw token", authorize: "beta-token", wantStatus: http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.authorize != "" {
				req.Header.Set("Authorization", tc.authorize)
			}
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)
			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.Code, tc.wantStatus)
			}
		})
	}
}
