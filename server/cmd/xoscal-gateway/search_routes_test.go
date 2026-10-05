package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	services "github.com/mchorfa/xoscal/proto/oscal/services/v1"
)

type oscalSearchStub struct {
	services.UnimplementedOscalServiceServer
}

func (oscalSearchStub) Search(context.Context, *services.SearchRequest) (*services.SearchResponse, error) {
	return &services.SearchResponse{NextPageToken: "oscal-search"}, nil
}

type semanticSearchStub struct {
	services.UnimplementedGovernanceServiceServer
}

func (semanticSearchStub) SemanticSearch(context.Context, *services.SemanticSearchRequest) (*services.SemanticSearchResponse, error) {
	return &services.SemanticSearchResponse{Results: []*services.SemanticSearchResult{{EntityUrn: "semantic-search"}}}, nil
}

// Exercise the actual generated gateway handlers in the application's registration
// order. A docs-only route change would fail to make both RPCs reachable here.
func TestSearchRoutesReachDistinctServices(t *testing.T) {
	mux := runtime.NewServeMux()
	if err := services.RegisterGovernanceServiceHandlerServer(context.Background(), mux, semanticSearchStub{}); err != nil {
		t.Fatal(err)
	}
	if err := services.RegisterOscalServiceHandlerServer(context.Background(), mux, oscalSearchStub{}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ path, marker string }{
		{"/v1/search?query=test", "oscal-search"},
		{"/v1/search/semantic?query=test", "semantic-search"},
	} {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), tc.marker) {
			t.Errorf("%s = status %d, body %s; want %s", tc.path, recorder.Code, recorder.Body.String(), tc.marker)
		}
	}
}
