package gitops

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientPublishReleaseAndProposeMappingUpdate(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token-123")
	if c := NewClient(); c == nil || c.gh == nil {
		t.Fatal("NewClient with GITHUB_TOKEN returned nil")
	}
	t.Setenv("GITHUB_TOKEN", "")
	if c := NewClient(); c == nil || c.gh == nil {
		t.Fatal("NewClient without GITHUB_TOKEN returned nil")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/ckodex-labs/ckodex-xoscal/releases", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Authorization") != "Bearer unit-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"html_url":   "https://github.com/ckodex-labs/ckodex-xoscal/releases/tag/v0.4.1",
			"upload_url": "https://uploads.github.com/repos/ckodex-labs/ckodex-xoscal/releases/1/assets{?name,label}",
		})
	})
	mux.HandleFunc("/repos/ckodex-labs/ckodex-xoscal/pulls", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"html_url": "https://github.com/ckodex-labs/ckodex-xoscal/pull/42",
			"number":   42,
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClientWithBaseURL(srv.URL, "unit-token")
	if err != nil {
		t.Fatalf("NewClientWithBaseURL: %v", err)
	}

	htmlURL, uploadURL, err := client.PublishRelease(
		context.Background(),
		"ckodex-labs",
		"ckodex-xoscal",
		"v0.4.1",
		"main",
		"xOSCAL v0.4.1",
		"Release notes",
		nil,
	)
	if err != nil {
		t.Fatalf("PublishRelease failed: %v", err)
	}
	if htmlURL != "https://github.com/ckodex-labs/ckodex-xoscal/releases/tag/v0.4.1" || uploadURL == "" {
		t.Fatalf("unexpected release URLs: html=%q upload=%q", htmlURL, uploadURL)
	}

	prURL, prNum, err := client.ProposeMappingUpdate(
		context.Background(),
		"ckodex-labs",
		"ckodex-xoscal",
		"main",
		"feat/crosswalk",
		"Update crosswalk mappings",
		"Automated crosswalk update",
	)
	if err != nil {
		t.Fatalf("ProposeMappingUpdate failed: %v", err)
	}
	if prNum != 42 || prURL != "https://github.com/ckodex-labs/ckodex-xoscal/pull/42" {
		t.Fatalf("unexpected PR response: num=%d url=%q", prNum, prURL)
	}

	unauthClient, err := NewClientWithBaseURL(srv.URL, "")
	if err != nil {
		t.Fatalf("NewClientWithBaseURL unauth: %v", err)
	}
	if _, _, err := unauthClient.PublishRelease(context.Background(), "ckodex-labs", "ckodex-xoscal", "v0.4.1", "main", "x", "y", nil); err == nil {
		t.Fatal("expected error when unauthorized")
	}
}
