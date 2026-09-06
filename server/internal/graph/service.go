package graph

import (
	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"github.com/mchorfa/xoscal/server/internal/transparency"
)

type GraphServer struct {
	servicesv1.UnimplementedTransparencyGraphServiceServer
	store    Store
	exchange transparency.Store
}

// NewGraphServer creates a new GraphServer.
func NewGraphServer(store Store, exchange ...transparency.Store) *GraphServer {
	server := &GraphServer{store: store}
	if len(exchange) > 0 {
		server.exchange = exchange[0]
	}
	return server
}

// ProjectEdge creates an edge from an existing claim.
