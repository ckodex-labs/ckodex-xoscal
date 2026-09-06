package graph

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GraphServer) ProjectEdge(ctx context.Context, req *servicesv1.ProjectEdgeRequest) (*servicesv1.ProjectEdgeResponse, error) {
	if req.GetClaimId() == "" {
		return nil, status.Error(codes.InvalidArgument, "claim_id is required")
	}
	if s.exchange == nil {
		return nil, status.Error(codes.FailedPrecondition, "claim exchange is not configured")
	}
	claim, err := s.exchange.GetClaim(ctx, req.GetClaimId())
	if err != nil {
		return nil, mapStoreError("get claim", req.GetClaimId(), err)
	}
	var subject, object, predicate map[string]string
	if err := json.Unmarshal([]byte(claim.SubjectJSON), &subject); err != nil {
		return nil, status.Error(codes.FailedPrecondition, "claim subject is invalid JSON")
	}
	if err := json.Unmarshal([]byte(claim.PredicateJSON), &predicate); err != nil || predicate["relation"] == "" {
		return nil, status.Error(codes.FailedPrecondition, "claim predicate is invalid or has no relation")
	}
	if claim.ObjectJSON == "" || json.Unmarshal([]byte(claim.ObjectJSON), &object) != nil {
		return nil, status.Error(codes.FailedPrecondition, "claim object is required for graph projection")
	}
	fromNode := nodeID(subject)
	toNode := nodeID(object)
	if fromNode == "" || toNode == "" {
		return nil, status.Error(codes.FailedPrecondition, "claim subject and object need an id or digest")
	}
	edgeID := req.GetEdgeId()
	if edgeID == "" {
		edgeID = defaultProjectionEdgeID(req.GetClaimId(), fromNode, toNode, predicate["relation"])
	}
	var sourceRefs []struct {
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal([]byte(claim.SourceRefsJSON), &sourceRefs); err != nil {
		return nil, status.Error(codes.FailedPrecondition, "claim source references are invalid JSON")
	}
	if len(sourceRefs) == 0 || sourceRefs[0].Digest == "" {
		return nil, status.Error(codes.FailedPrecondition, "claim needs a source evidence digest for graph projection")
	}
	from := &Node{ID: fromNode, Kind: subject["kind"], URN: nodeURN(subject), CreatedAt: claim.CreatedAt}
	to := &Node{ID: toNode, Kind: object["kind"], URN: nodeURN(object), CreatedAt: claim.CreatedAt}
	e := &Edge{
		ID:             edgeID,
		FromNode:       fromNode,
		ToNode:         toNode,
		Relation:       predicate["relation"],
		ClaimID:        req.GetClaimId(),
		ProofStateJSON: claim.ProofStateJSON,
		TrustState:     claim.TrustState,
		Weight:         1.0,
		ValidFrom:      claim.ValidFrom,
	}
	if e.ValidFrom.IsZero() {
		e.ValidFrom = claim.CreatedAt
		if e.ValidFrom.IsZero() {
			e.ValidFrom = time.Now().UTC()
		}
	}
	e.EvidenceDigest = sourceRefs[0].Digest
	event, err := s.store.CreateProjection(ctx, from, to, e)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, status.Errorf(codes.AlreadyExists, "edge %q already exists", edgeID)
		}
		return nil, status.Errorf(codes.Internal, "create edge: %v", err)
	}

	return &servicesv1.ProjectEdgeResponse{
		Edge:            edgeToProto(e),
		ProjectionEvent: projectionEventToProto(event),
	}, nil
}

// ListProjectionEvents returns the append-only graph projection audit history
// for a claim or edge after validating the global event chain.
func (s *GraphServer) ListProjectionEvents(ctx context.Context, req *servicesv1.ListProjectionEventsRequest) (*servicesv1.ListProjectionEventsResponse, error) {
	if req.GetClaimId() == "" && req.GetEdgeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "claim_id or edge_id is required")
	}
	if err := s.store.VerifyProjectionChain(ctx); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "graph projection audit chain is invalid: %v", err)
	}
	events, err := s.store.ListProjectionEvents(ctx, req.GetClaimId(), req.GetEdgeId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list projection events: %v", err)
	}
	out := make([]*servicesv1.GraphProjectionEvent, 0, len(events))
	for _, event := range events {
		out = append(out, projectionEventToProto(event))
	}
	return &servicesv1.ListProjectionEventsResponse{Events: out, ChainValid: true}, nil
}

// GetEdge retrieves a graph edge by ID.
