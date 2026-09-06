package graph

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"github.com/mchorfa/xoscal/server/internal/transparency"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GraphServer) ExplainClaim(ctx context.Context, req *servicesv1.ExplainClaimRequest) (*servicesv1.ExplainClaimResponse, error) {
	if s.exchange == nil {
		return nil, status.Error(codes.FailedPrecondition, "claim exchange is not configured")
	}
	claimID := req.GetClaimId()
	if claimID == "" && req.GetEdgeId() != "" {
		edge, err := s.store.GetEdge(ctx, req.GetEdgeId())
		if err != nil {
			return nil, mapStoreError("get edge", req.GetEdgeId(), err)
		}
		claimID = edge.ClaimID
	}
	if claimID == "" {
		return nil, status.Error(codes.InvalidArgument, "claim_id or edge_id is required")
	}
	claim, err := s.exchange.GetClaim(ctx, claimID)
	if err != nil {
		return nil, mapStoreError("get claim", claimID, err)
	}
	response := &servicesv1.ExplainClaimResponse{
		Claim:          transparency.ClaimToProto(claim),
		ProofStateJson: claim.ProofStateJSON,
		TrustState:     claim.TrustState,
	}
	var refs []struct {
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal([]byte(claim.SourceRefsJSON), &refs); err != nil {
		return nil, status.Error(codes.FailedPrecondition, "claim source references are invalid JSON")
	}
	for _, ref := range refs {
		if ref.Digest == "" {
			continue
		}
		evidence, err := s.exchange.GetEvidenceByDigest(ctx, ref.Digest)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, status.Errorf(codes.Internal, "get evidence %q: %v", ref.Digest, err)
		}
		response.Evidence = append(response.Evidence, transparency.EvidenceToProto(evidence))
	}
	return response, nil
}

// ComputeTrustState re-computes trust state for a claim or edge.
func (s *GraphServer) ComputeTrustState(ctx context.Context, req *servicesv1.ComputeTrustStateRequest) (*servicesv1.ComputeTrustStateResponse, error) {
	if req.GetEdgeId() != "" {
		e, err := s.store.GetEdge(ctx, req.GetEdgeId())
		if err != nil {
			return nil, mapStoreError("get edge", req.GetEdgeId(), err)
		}
		ps := parseProofState(e.ProofStateJSON)
		return &servicesv1.ComputeTrustStateResponse{
			EdgeId:     e.ID,
			ProofState: ps,
			TrustState: e.TrustState,
		}, nil
	}
	if req.GetClaimId() != "" {
		if s.exchange == nil {
			return nil, status.Error(codes.FailedPrecondition, "claim exchange is not configured")
		}
		claim, err := s.exchange.GetClaim(ctx, req.GetClaimId())
		if err != nil {
			return nil, mapStoreError("get claim", req.GetClaimId(), err)
		}
		return &servicesv1.ComputeTrustStateResponse{
			ClaimId:    claim.ID,
			ProofState: parseProofState(claim.ProofStateJSON),
			TrustState: claim.TrustState,
		}, nil
	}
	return nil, status.Error(codes.InvalidArgument, "claim_id or edge_id is required")
}

// VerifyClosure verifies all claims reachable from a subject for a given purpose.
func (s *GraphServer) VerifyClosure(ctx context.Context, req *servicesv1.VerifyClosureRequest) (*servicesv1.VerifyClosureResponse, error) {
	if req.GetSubjectNode() == "" {
		return nil, status.Error(codes.InvalidArgument, "subject_node is required")
	}
	if _, err := s.store.GetNode(ctx, req.GetSubjectNode()); err != nil {
		return nil, mapStoreError("get subject node", req.GetSubjectNode(), err)
	}
	// Inspect every reachable edge so an unverified edge cannot disappear simply
	// because it was filtered out of a traversal query.
	resp, err := s.ImpactRadius(ctx, &servicesv1.ImpactRadiusRequest{
		Node:          req.GetSubjectNode(),
		MaxDepth:      5,
		MinTrustState: "",
	})
	if err != nil {
		return nil, err
	}

	failed := 0
	for _, edge := range resp.Edges {
		if edge.GetTrustState() != "verified" {
			failed++
		}
	}
	verdict := len(resp.Edges) > 0 && failed == 0
	trustState := "incomplete"
	if verdict {
		trustState = "verified"
	}
	diagnostics := []string{}
	if len(resp.Edges) == 0 {
		diagnostics = append(diagnostics, "no projected edges found for subject")
	}
	if failed > 0 {
		diagnostics = append(diagnostics, fmt.Sprintf("%d reachable edge(s) are not verified", failed))
	}
	return &servicesv1.VerifyClosureResponse{
		SubjectNode:  req.GetSubjectNode(),
		Purpose:      req.GetPurpose(),
		Verdict:      verdict,
		TrustState:   trustState,
		Diagnostics:  diagnostics,
		EdgesChecked: boundedInt32(len(resp.Edges)),
		EdgesFailed:  boundedInt32(failed),
	}, nil
}

// ---- helpers ----
