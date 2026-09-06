package transparency

import (
	"context"
	"strings"
	"time"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewExchangeServer creates a new ExchangeServer.
func NewExchangeServer(store Store) *ExchangeServer {
	return &ExchangeServer{store: store}
}

// CreateClaim stores a new claim and computes its initial trust state.

// CreateClaim stores a new claim and computes its initial trust state.
func (s *ExchangeServer) CreateClaim(ctx context.Context, req *servicesv1.CreateClaimRequest) (*servicesv1.CreateClaimResponse, error) {
	c := req.GetClaim()
	if c == nil || c.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "claim id is required")
	}
	if c.GetPredicate() == nil || c.GetPredicate().GetRelation() == "" {
		return nil, status.Error(codes.InvalidArgument, "claim predicate relation is required")
	}
	if c.GetSubject() == nil || (c.GetSubject().GetId() == "" && c.GetSubject().GetDigest() == "") {
		return nil, status.Error(codes.InvalidArgument, "claim subject id or digest is required")
	}

	claim := claimFromProto(c)
	claim.TrustState = "candidate"
	claim.ProofStateJSON = defaultProofStateJSON()
	claim.CreatedAt = time.Now().UTC()
	if claim.ObservedTime.IsZero() {
		claim.ObservedTime = claim.CreatedAt
	}

	if err := s.store.CreateClaim(ctx, claim); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			if existing, lookupErr := s.store.GetClaim(ctx, claim.ID); lookupErr == nil && claimsEquivalent(existing, claim) {
				return &servicesv1.CreateClaimResponse{
					Claim:      claimToProto(existing),
					TrustState: existing.TrustState,
				}, nil
			}
			return nil, status.Errorf(codes.AlreadyExists, "claim %q already exists", claim.ID)
		}
		return nil, status.Errorf(codes.Internal, "create claim: %v", err)
	}

	return &servicesv1.CreateClaimResponse{
		Claim:      claimToProto(claim),
		TrustState: claim.TrustState,
	}, nil
}

// GetClaim retrieves a claim by ID.

// GetClaim retrieves a claim by ID.
func (s *ExchangeServer) GetClaim(ctx context.Context, req *servicesv1.GetClaimRequest) (*servicesv1.GetClaimResponse, error) {
	claim, err := s.store.GetClaim(ctx, req.GetClaimId())
	if err != nil {
		return nil, mapStoreError("get claim", req.GetClaimId(), err)
	}
	return &servicesv1.GetClaimResponse{Claim: claimToProto(claim)}, nil
}

// ListClaims returns claims matching optional filters.

// ListClaims returns claims matching optional filters.
func (s *ExchangeServer) ListClaims(ctx context.Context, req *servicesv1.ListClaimsRequest) (*servicesv1.ListClaimsResponse, error) {
	var validAfter time.Time
	if req.GetValidAfter() != nil {
		validAfter = req.GetValidAfter().AsTime()
	}
	pageSize := int(req.GetPageSize())
	if pageSize < 0 || pageSize > maxListPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "page_size must be between 0 and %d", maxListPageSize)
	}
	claims, nextToken, err := s.store.ListClaims(ctx, req.GetSubjectDigest(), req.GetBomKind(),
		req.GetRelation(), req.GetTrustState(), validAfter, pageSize, req.GetPageToken())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "invalid page token") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "list claims: %v", err)
	}
	var out []*servicesv1.Claim
	for _, c := range claims {
		out = append(out, claimToProto(c))
	}
	return &servicesv1.ListClaimsResponse{Claims: out, NextPageToken: nextToken}, nil
}

// ListVerificationEvents returns the append-only audit history for a claim.
// The global chain is validated before any events are exposed so an operator
// never mistakes a partial or tampered history for a trustworthy receipt.

// ListVerificationEvents returns the append-only audit history for a claim.
// The global chain is validated before any events are exposed so an operator
// never mistakes a partial or tampered history for a trustworthy receipt.
func (s *ExchangeServer) ListVerificationEvents(ctx context.Context, req *servicesv1.ListVerificationEventsRequest) (*servicesv1.ListVerificationEventsResponse, error) {
	claimID := req.GetClaimId()
	if claimID == "" {
		return nil, status.Error(codes.InvalidArgument, "claim_id is required")
	}
	if _, err := s.store.GetClaim(ctx, claimID); err != nil {
		return nil, mapStoreError("get claim", claimID, err)
	}
	if err := s.store.VerifyVerificationChain(ctx); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "verification audit chain is invalid: %v", err)
	}
	events, err := s.store.ListVerificationEvents(ctx, claimID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list verification events: %v", err)
	}
	out := make([]*servicesv1.VerificationEvent, 0, len(events))
	for _, event := range events {
		converted, err := verificationEventToProto(event)
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "verification audit event %q is invalid: %v", event.EventID, err)
		}
		out = append(out, converted)
	}
	return &servicesv1.ListVerificationEventsResponse{Events: out, ChainValid: true}, nil
}

// ExportClaimReceipt emits a deterministic metadata receipt only after the
// latest persisted verification event proves the beta profile. Evidence bytes
// are deliberately excluded; the receipt carries content digests so the
// operator can retrieve and verify them independently.
