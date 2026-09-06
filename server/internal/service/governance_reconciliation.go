package service

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func (s *GovernanceServer) ProposeMappingUpdate(ctx context.Context, req *servicesv1.ProposeMappingUpdateRequest) (*servicesv1.ProposeMappingUpdateResponse, error) {
	if s.gitops == nil {
		return nil, status.Error(codes.Internal, "gitops client not initialized")
	}
	prURL, prNumber, err := s.gitops.ProposeMappingUpdate(
		ctx,
		req.RepoOwner,
		req.RepoName,
		req.BaseBranch,
		req.HeadBranch,
		req.Title,
		req.Body,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "propose mapping update: %v", err)
	}
	return &servicesv1.ProposeMappingUpdateResponse{
		PullRequestUrl:    prURL,
		PullRequestNumber: boundedInt32(prNumber),
	}, nil
}

// --- Reconciler ---

func (s *GovernanceServer) Propose(ctx context.Context, req *servicesv1.ProposeRequest) (*servicesv1.ProposeResponse, error) {
	var proposals []*kg.Entity
	for _, pe := range req.Proposals {
		proposals = append(proposals, toKGEntity(pe))
	}
	conflicts, err := s.rec.Propose(ctx, proposals)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "propose: %v", err)
	}
	// Also detect batch-level conflicts (e.g. MappingGap).
	batchConflicts, err := s.rec.EvaluateBatch(ctx, proposals)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "evaluate batch: %v", err)
	}
	conflicts = append(conflicts, batchConflicts...)
	return &servicesv1.ProposeResponse{
		Accepted:  boundedInt32(len(proposals) - len(conflicts)),
		Conflicts: boundedInt32(len(conflicts)),
	}, nil
}

func (s *GovernanceServer) ListConflicts(ctx context.Context, req *servicesv1.ListConflictsRequest) (*servicesv1.ListConflictsResponse, error) {
	conflicts, err := s.rec.ListConflicts(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list conflicts: %v", err)
	}
	var out []*servicesv1.Entity
	for _, c := range conflicts {
		raw, err := json.Marshal(c)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "marshal conflict %s: %v", c.ID, err)
		}
		out = append(out, &servicesv1.Entity{
			Urn:     c.ID,
			Type:    string(c.Type),
			Payload: string(raw),
		})
	}
	_ = req
	return &servicesv1.ListConflictsResponse{Conflicts: out}, nil
}

func (s *GovernanceServer) ResolveConflict(ctx context.Context, req *servicesv1.ResolveConflictRequest) (*servicesv1.ResolveConflictResponse, error) {
	if err := s.rec.ResolveConflict(ctx, req.ConflictUrn, []byte(req.Resolution)); err != nil {
		return nil, status.Errorf(codes.Internal, "resolve conflict: %v", err)
	}
	return &servicesv1.ResolveConflictResponse{Success: true}, nil
}

// --- Helpers ---
