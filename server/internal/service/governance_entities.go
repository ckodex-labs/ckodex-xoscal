package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func (s *GovernanceServer) CreateEntity(ctx context.Context, req *servicesv1.CreateEntityRequest) (*servicesv1.CreateEntityResponse, error) {
	if req.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}
	e := toKGEntity(req.Entity)
	if err := s.kgStore.CreateEntity(ctx, e); err != nil {
		return nil, status.Errorf(codes.Internal, "create entity: %v", err)
	}
	return &servicesv1.CreateEntityResponse{Entity: toProtoEntity(e)}, nil
}

func (s *GovernanceServer) GetEntity(ctx context.Context, req *servicesv1.GetEntityRequest) (*servicesv1.GetEntityResponse, error) {
	e, err := s.kgStore.GetEntity(ctx, req.Urn)
	if err != nil {
		return nil, notFoundErr("entity", err)
	}
	return &servicesv1.GetEntityResponse{Entity: toProtoEntity(e)}, nil
}

func (s *GovernanceServer) UpdateEntity(ctx context.Context, req *servicesv1.UpdateEntityRequest) (*servicesv1.UpdateEntityResponse, error) {
	if req.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}
	e := toKGEntity(req.Entity)
	if err := s.kgStore.UpdateEntity(ctx, e); err != nil {
		return nil, status.Errorf(codes.Internal, "update entity: %v", err)
	}
	return &servicesv1.UpdateEntityResponse{Entity: toProtoEntity(e)}, nil
}

func (s *GovernanceServer) ListEntities(ctx context.Context, req *servicesv1.ListEntitiesRequest) (*servicesv1.ListEntitiesResponse, error) {
	var statusFilter kg.EntityStatus
	if req.StatusFilter != "" {
		statusFilter = kg.EntityStatus(req.StatusFilter)
	}
	entities, err := s.kgStore.ListEntities(ctx, req.TypeFilter, statusFilter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list entities: %v", err)
	}
	var out []*servicesv1.Entity
	for _, e := range entities {
		out = append(out, toProtoEntity(e))
	}
	return &servicesv1.ListEntitiesResponse{Entities: out}, nil
}

// --- Snapshots & Releases ---

func (s *GovernanceServer) CreateSnapshot(ctx context.Context, req *servicesv1.CreateSnapshotRequest) (*servicesv1.CreateSnapshotResponse, error) {
	ss, err := s.kgStore.CreateSnapshot(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create snapshot: %v", err)
	}
	return &servicesv1.CreateSnapshotResponse{
		Name:        ss.Name,
		EntityCount: boundedInt32(ss.EntityCount),
	}, nil
}

func (s *GovernanceServer) GetSnapshot(ctx context.Context, req *servicesv1.GetSnapshotRequest) (*servicesv1.GetSnapshotResponse, error) {
	entities, err := s.kgStore.GetSnapshot(ctx, req.Name)
	if err != nil {
		return nil, notFoundErr("snapshot", err)
	}
	var out []*servicesv1.Entity
	for _, e := range entities {
		out = append(out, toProtoEntity(e))
	}
	return &servicesv1.GetSnapshotResponse{Entities: out}, nil
}

func (s *GovernanceServer) ListSnapshots(ctx context.Context, req *servicesv1.ListSnapshotsRequest) (*servicesv1.ListSnapshotsResponse, error) {
	snapshots, err := s.kgStore.ListSnapshots(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list snapshots: %v", err)
	}
	var names []string
	for _, ss := range snapshots {
		names = append(names, ss.Name)
	}
	return &servicesv1.ListSnapshotsResponse{Names: names}, nil
}

func (s *GovernanceServer) CreateRelease(ctx context.Context, req *servicesv1.CreateReleaseRequest) (*servicesv1.CreateReleaseResponse, error) {
	r, err := s.kgStore.CreateRelease(ctx, req.Name, req.SnapshotName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create release: %v", err)
	}
	return &servicesv1.CreateReleaseResponse{
		Name:         r.Name,
		SnapshotName: r.Snapshot,
	}, nil
}

func (s *GovernanceServer) ListReleases(ctx context.Context, req *servicesv1.ListReleasesRequest) (*servicesv1.ListReleasesResponse, error) {
	releases, err := s.kgStore.ListReleases(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list releases: %v", err)
	}
	var names []string
	for _, r := range releases {
		names = append(names, r.Name)
	}
	return &servicesv1.ListReleasesResponse{Names: names}, nil
}

// --- Ingestion ---
