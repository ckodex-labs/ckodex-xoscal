package service

import (
	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"github.com/mchorfa/xoscal/server/internal/embedding"
	"github.com/mchorfa/xoscal/server/internal/gitops"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/reconciler"
)

type GovernanceServer struct {
	servicesv1.UnimplementedGovernanceServiceServer
	kgStore     kg.Store
	rec         *reconciler.Reconciler
	vectorStore embedding.VectorStore
	generator   *oscal.Generator
	gitops      *gitops.Client
}

// NewGovernanceServer creates a new governance gRPC server backed by the
// given advanced components.
func NewGovernanceServer(
	kgStore kg.Store,
	rec *reconciler.Reconciler,
	vectorStore embedding.VectorStore,
) *GovernanceServer {
	return &GovernanceServer{
		kgStore:     kgStore,
		rec:         rec,
		vectorStore: vectorStore,
		generator:   oscal.NewGenerator(kgStore),
		gitops:      gitops.NewClient(),
	}
}

// --- Entity CRUD ---
