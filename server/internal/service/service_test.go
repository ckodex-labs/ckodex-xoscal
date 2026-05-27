package service

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/store"
)

func newTestServer(t *testing.T) (servicesv1.OscalServiceClient, func()) {
	s, err := store.NewSQLiteStore(":memory:", dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	servicesv1.RegisterOscalServiceServer(grpcServer, NewOscalServer(s))

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("server stopped: %v", err)
		}
	}()

	conn, err := grpc.Dial("bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	cleanup := func() {
		conn.Close()
		grpcServer.GracefulStop()
		s.Close()
	}

	return servicesv1.NewOscalServiceClient(conn), cleanup
}

func TestCreateAndGetCatalog(t *testing.T) {
	ctx := context.Background()
	client, cleanup := newTestServer(t)
	defer cleanup()

	c := &catalogv1.Catalog{
		Uuid:     &commonv1.UUID{Value: "cat-001"},
		Metadata: &commonv1.Metadata{Title: "Test Catalog", Version: "1.0.0"},
	}

	_, err := client.CreateCatalog(ctx, &servicesv1.CreateCatalogRequest{Catalog: c})
	if err != nil {
		t.Fatalf("create catalog: %v", err)
	}

	resp, err := client.GetCatalog(ctx, &servicesv1.GetCatalogRequest{Uuid: &commonv1.UUID{Value: "cat-001"}})
	if err != nil {
		t.Fatalf("get catalog: %v", err)
	}
	if resp.Catalog.Metadata.Title != "Test Catalog" {
		t.Errorf("title = %q, want %q", resp.Catalog.Metadata.Title, "Test Catalog")
	}
}

func TestDeleteCatalogNotFound(t *testing.T) {
	ctx := context.Background()
	client, cleanup := newTestServer(t)
	defer cleanup()

	_, err := client.DeleteCatalog(ctx, &servicesv1.DeleteCatalogRequest{Uuid: &commonv1.UUID{Value: "missing"}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Errorf("code = %v, want %v", st.Code(), codes.NotFound)
	}
}

func TestSearch(t *testing.T) {
	ctx := context.Background()
	client, cleanup := newTestServer(t)
	defer cleanup()

	_, _ = client.CreateCatalog(ctx, &servicesv1.CreateCatalogRequest{Catalog: &catalogv1.Catalog{
		Uuid:     &commonv1.UUID{Value: "c1"},
		Metadata: &commonv1.Metadata{Title: "ISO 27001 Catalog", Version: "2023"},
	}})

	resp, err := client.Search(ctx, &servicesv1.SearchRequest{Query: "ISO"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Errorf("results = %d, want 1", len(resp.Results))
	}
	if resp.Results[0].Title != "ISO 27001 Catalog" {
		t.Errorf("title = %q, want %q", resp.Results[0].Title, "ISO 27001 Catalog")
	}
}
