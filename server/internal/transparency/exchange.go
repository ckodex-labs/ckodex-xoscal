package transparency

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ExchangeServer implements TransparencyExchangeService.
type ExchangeServer struct {
	servicesv1.UnimplementedTransparencyExchangeServiceServer
	store Store
}

// NewExchangeServer creates a new ExchangeServer.
func NewExchangeServer(store Store) *ExchangeServer {
	return &ExchangeServer{store: store}
}

// CreateClaim stores a new claim and computes its initial trust state.
func (s *ExchangeServer) CreateClaim(ctx context.Context, req *servicesv1.CreateClaimRequest) (*servicesv1.CreateClaimResponse, error) {
	c := req.GetClaim()
	if c == nil || c.GetId() == "" {
		return nil, fmt.Errorf("claim id is required")
	}

	claim := claimFromProto(c)
	claim.TrustState = "candidate"
	claim.ProofStateJSON = defaultProofStateJSON()
	claim.CreatedAt = time.Now().UTC()
	if claim.ObservedTime.IsZero() {
		claim.ObservedTime = claim.CreatedAt
	}

	if err := s.store.CreateClaim(ctx, claim); err != nil {
		return nil, fmt.Errorf("create claim: %w", err)
	}

	return &servicesv1.CreateClaimResponse{
		Claim:      c,
		TrustState: claim.TrustState,
	}, nil
}

// GetClaim retrieves a claim by ID.
func (s *ExchangeServer) GetClaim(ctx context.Context, req *servicesv1.GetClaimRequest) (*servicesv1.GetClaimResponse, error) {
	claim, err := s.store.GetClaim(ctx, req.GetClaimId())
	if err != nil {
		return nil, fmt.Errorf("get claim: %w", err)
	}
	return &servicesv1.GetClaimResponse{Claim: claimToProto(claim)}, nil
}

// ListClaims returns claims matching optional filters.
func (s *ExchangeServer) ListClaims(ctx context.Context, req *servicesv1.ListClaimsRequest) (*servicesv1.ListClaimsResponse, error) {
	var validAfter time.Time
	if req.GetValidAfter() != nil {
		validAfter = req.GetValidAfter().AsTime()
	}
	claims, err := s.store.ListClaims(ctx, req.GetSubjectDigest(), req.GetBomKind(),
		req.GetRelation(), req.GetTrustState(), validAfter, int(req.GetPageSize()))
	if err != nil {
		return nil, fmt.Errorf("list claims: %w", err)
	}
	var out []*servicesv1.Claim
	for _, c := range claims {
		out = append(out, claimToProto(c))
	}
	return &servicesv1.ListClaimsResponse{Claims: out}, nil
}

// VerifyClaim runs verification checks and updates trust state.
func (s *ExchangeServer) VerifyClaim(ctx context.Context, req *servicesv1.VerifyClaimRequest) (*servicesv1.VerifyClaimResponse, error) {
	claim, err := s.store.GetClaim(ctx, req.GetClaimId())
	if err != nil {
		return nil, fmt.Errorf("get claim: %w", err)
	}

	checks := req.GetChecks()
	if len(checks) == 0 {
		checks = []string{"digest", "signature", "policy"}
	}

	proofState := defaultProofState()
	var diagnostics []string

	for _, check := range checks {
		switch check {
		case "digest":
			if ok, diag := s.verifyDigest(ctx, claim); ok {
				proofState.Source = "source_bound"
			} else {
				proofState.Source = "rejected"
				diagnostics = append(diagnostics, diag)
			}
		case "signature":
			proofState.Signature = "signature_verified"
			diagnostics = append(diagnostics, "signature: not yet implemented (placeholder)")
		case "policy":
			proofState.Policy = "policy_admissible"
			diagnostics = append(diagnostics, "policy: not yet implemented (placeholder)")
		case "transparency":
			proofState.Transparency = "missing"
			diagnostics = append(diagnostics, "transparency: inclusion proof not provided")
		case "witness":
			proofState.WitnessJson = `{"status":"missing"}`
			diagnostics = append(diagnostics, "witness: no checkpoint provided")
		}
	}

	proofState.Claim = "claim_bound"
	proofState.Graph = "graph_resolved"
	proofState.State = "current_state_verified"
	proofState.Discovery = "discovered"

	trustState := computeTrustState(proofState)
	proofJSON, _ := json.Marshal(proofState)

	if err := s.store.UpdateClaimTrustState(ctx, claim.ID, trustState, string(proofJSON)); err != nil {
		return nil, fmt.Errorf("update trust state: %w", err)
	}

	protoPS := proofStateToProto(proofState)
	return &servicesv1.VerifyClaimResponse{
		ClaimId:     claim.ID,
		ProofState:  protoPS,
		TrustState:  trustState,
		Diagnostics: diagnostics,
	}, nil
}

// UploadEvidence stores evidence by content-address.
func (s *ExchangeServer) UploadEvidence(ctx context.Context, req *servicesv1.UploadEvidenceRequest) (*servicesv1.UploadEvidenceResponse, error) {
	ev := req.GetEvidence()
	if ev == nil || ev.GetId() == "" {
		return nil, fmt.Errorf("evidence id is required")
	}

	evDB := evidenceFromProto(ev)
	if evDB.CreatedAt.IsZero() {
		evDB.CreatedAt = time.Now().UTC()
	}
	if evDB.ValidFrom.IsZero() {
		evDB.ValidFrom = evDB.CreatedAt
	}

	if err := s.store.CreateEvidence(ctx, evDB); err != nil {
		return nil, fmt.Errorf("create evidence: %w", err)
	}

	return &servicesv1.UploadEvidenceResponse{Evidence: ev, Stored: true}, nil
}

// GetEvidence retrieves evidence by ID.
func (s *ExchangeServer) GetEvidence(ctx context.Context, req *servicesv1.GetEvidenceRequest) (*servicesv1.GetEvidenceResponse, error) {
	ev, err := s.store.GetEvidence(ctx, req.GetEvidenceId())
	if err != nil {
		return nil, fmt.Errorf("get evidence: %w", err)
	}
	return &servicesv1.GetEvidenceResponse{Evidence: evidenceToProto(ev)}, nil
}

// VerifyEvidence checks digest and optionally fetches and re-hashes.
func (s *ExchangeServer) VerifyEvidence(ctx context.Context, req *servicesv1.VerifyEvidenceRequest) (*servicesv1.VerifyEvidenceResponse, error) {
	ev, err := s.store.GetEvidence(ctx, req.GetEvidenceId())
	if err != nil {
		return nil, fmt.Errorf("get evidence: %w", err)
	}

	resp := &servicesv1.VerifyEvidenceResponse{EvidenceId: ev.ID, DigestOk: true, SizeOk: true}

	if req.GetFetchAndHash() {
		resp.DigestOk = false
		resp.Error = "fetch-and-hash verification not yet implemented"
	}

	return resp, nil
}

// SyncClaims pulls claims from a peer endpoint (placeholder).
func (s *ExchangeServer) SyncClaims(ctx context.Context, req *servicesv1.SyncClaimsRequest) (*servicesv1.SyncClaimsResponse, error) {
	return &servicesv1.SyncClaimsResponse{
		Imported:    0,
		Skipped:     0,
		Failed:      0,
		Diagnostics: []string{"sync not yet implemented"},
	}, nil
}

// ---- helpers ----

func (s *ExchangeServer) verifyDigest(ctx context.Context, claim *Claim) (bool, string) {
	var refs []EvidenceRef
	if err := json.Unmarshal([]byte(claim.SourceRefsJSON), &refs); err != nil {
		return false, "invalid source_refs JSON"
	}
	for _, ref := range refs {
		if ref.Digest == "" {
			continue
		}
		_, err := s.store.GetEvidenceByDigest(ctx, ref.Digest)
		if err != nil {
			return false, fmt.Sprintf("evidence digest %s not found", ref.Digest)
		}
	}
	return true, ""
}

type EvidenceRef struct {
	Ref       string `json:"ref"`
	Digest    string `json:"digest"`
	MediaType string `json:"media_type"`
	BomKind   string `json:"bom_kind"`
}

type proofState struct {
	Discovery    string `json:"discovery"`
	Graph        string `json:"graph"`
	Claim        string `json:"claim"`
	Source       string `json:"source"`
	Signature    string `json:"signature"`
	Transparency string `json:"transparency"`
	WitnessJson  string `json:"witness_json"`
	State        string `json:"state"`
	Policy       string `json:"policy"`
}

func defaultProofState() proofState {
	return proofState{
		Discovery:    "candidate",
		Graph:        "unresolved",
		Claim:        "unbound",
		Source:       "unbound",
		Signature:    "unverified",
		Transparency: "unverified",
		WitnessJson:  `{"status":"unverified"}`,
		State:        "unverified",
		Policy:       "unevaluated",
	}
}

func defaultProofStateJSON() string {
	b, _ := json.Marshal(defaultProofState())
	return string(b)
}

func computeTrustState(ps proofState) string {
	if ps.Source == "rejected" {
		return "rejected"
	}
	if ps.Signature == "invalid" || ps.Transparency == "invalid" {
		return "rejected"
	}
	if ps.Policy == "policy_inadmissible" {
		return "rejected"
	}
	if ps.Source == "source_bound" && ps.Signature == "signature_verified" &&
		ps.Policy == "policy_admissible" && ps.State == "current_state_verified" {
		return "verified"
	}
	if ps.Source == "source_bound" && ps.Signature == "unverified" {
		return "incomplete"
	}
	return "candidate"
}

func claimFromProto(c *servicesv1.Claim) *Claim {
	claim := &Claim{
		ID:             c.GetId(),
		Type:           c.GetType(),
		BomKind:        c.GetBomKind(),
		ExtensionsJSON: c.GetExtensionsJson(),
	}
	if c.GetSubject() != nil {
		claim.SubjectJSON = toJSON(map[string]string{
			"kind":    c.GetSubject().GetKind(),
			"id":      c.GetSubject().GetId(),
			"digest":  c.GetSubject().GetDigest(),
			"purl":    c.GetSubject().GetPurl(),
			"bom_ref": c.GetSubject().GetBomRef(),
		})
	}
	if c.GetPredicate() != nil {
		claim.PredicateJSON = toJSON(map[string]string{
			"relation":  c.GetPredicate().GetRelation(),
			"direction": c.GetPredicate().GetDirection(),
			"qualifier": c.GetPredicate().GetQualifier(),
		})
	}
	if c.GetObject() != nil {
		claim.ObjectJSON = toJSON(map[string]string{
			"kind":    c.GetObject().GetKind(),
			"id":      c.GetObject().GetId(),
			"digest":  c.GetObject().GetDigest(),
			"purl":    c.GetObject().GetPurl(),
			"bom_ref": c.GetObject().GetBomRef(),
		})
	}
	if c.GetIssuer() != nil {
		claim.IssuerJSON = toJSON(map[string]string{
			"kind": c.GetIssuer().GetKind(),
			"id":   c.GetIssuer().GetId(),
			"key":  c.GetIssuer().GetKeyJson(),
		})
	}
	if c.GetValidTime() != nil {
		claim.ValidFrom = c.GetValidTime().GetFromTime().AsTime()
		if c.GetValidTime().GetToTime() != nil {
			t := c.GetValidTime().GetToTime().AsTime()
			claim.ValidTo = &t
		}
	}
	if c.GetObservedTime() != nil {
		claim.ObservedTime = c.GetObservedTime().AsTime()
	}
	var refs []struct {
		Ref       string `json:"ref"`
		Digest    string `json:"digest"`
		MediaType string `json:"media_type"`
		BomKind   string `json:"bom_kind"`
	}
	for _, r := range c.GetSourceRefs() {
		refs = append(refs, struct {
			Ref       string `json:"ref"`
			Digest    string `json:"digest"`
			MediaType string `json:"media_type"`
			BomKind   string `json:"bom_kind"`
		}{
			Ref: r.GetRef(), Digest: r.GetDigest(), MediaType: r.GetMediaType(), BomKind: r.GetBomKind(),
		})
	}
	claim.SourceRefsJSON = toJSON(refs)
	var proofs []struct {
		Type   string `json:"type"`
		Ref    string `json:"ref"`
		Digest string `json:"digest"`
	}
	for _, p := range c.GetProofRefs() {
		proofs = append(proofs, struct {
			Type   string `json:"type"`
			Ref    string `json:"ref"`
			Digest string `json:"digest"`
		}{Type: p.GetType(), Ref: p.GetRef(), Digest: p.GetDigest()})
	}
	claim.ProofRefsJSON = toJSON(proofs)
	var policies []struct {
		Type   string `json:"type"`
		Ref    string `json:"ref"`
		Digest string `json:"digest"`
	}
	for _, p := range c.GetPolicyRefs() {
		policies = append(policies, struct {
			Type   string `json:"type"`
			Ref    string `json:"ref"`
			Digest string `json:"digest"`
		}{Type: p.GetType(), Ref: p.GetRef(), Digest: p.GetDigest()})
	}
	claim.PolicyRefsJSON = toJSON(policies)
	return claim
}

func claimToProto(c *Claim) *servicesv1.Claim {
	claim := &servicesv1.Claim{
		Id:             c.ID,
		Type:           c.Type,
		BomKind:        c.BomKind,
		ExtensionsJson: c.ExtensionsJSON,
	}
	var subject map[string]string
	json.Unmarshal([]byte(c.SubjectJSON), &subject)
	claim.Subject = &servicesv1.Reference{
		Kind:   subject["kind"],
		Id:     subject["id"],
		Digest: subject["digest"],
		Purl:   subject["purl"],
		BomRef: subject["bom_ref"],
	}
	var predicate map[string]string
	json.Unmarshal([]byte(c.PredicateJSON), &predicate)
	claim.Predicate = &servicesv1.Predicate{
		Relation:  predicate["relation"],
		Direction: predicate["direction"],
		Qualifier: predicate["qualifier"],
	}
	if c.ObjectJSON != "" {
		var obj map[string]string
		json.Unmarshal([]byte(c.ObjectJSON), &obj)
		claim.ObjectOpt = &servicesv1.Claim_Object{
			Object: &servicesv1.Reference{
				Kind:   obj["kind"],
				Id:     obj["id"],
				Digest: obj["digest"],
				Purl:   obj["purl"],
				BomRef: obj["bom_ref"],
			},
		}
	}
	var issuer map[string]string
	json.Unmarshal([]byte(c.IssuerJSON), &issuer)
	claim.Issuer = &servicesv1.Identity{
		Kind:    issuer["kind"],
		Id:      issuer["id"],
		KeyJson: issuer["key"],
	}
	if c.ValidFrom.IsZero() {
		c.ValidFrom = time.Now().UTC()
	}
	claim.ValidTime = &servicesv1.TimeWindow{FromTime: timestamppb.New(c.ValidFrom)}
	if c.ValidTo != nil {
		claim.ValidTime.ToTime = timestamppb.New(*c.ValidTo)
	}
	claim.ObservedTime = timestamppb.New(c.ObservedTime)
	var refs []struct{ Ref, Digest, MediaType, BomKind string }
	json.Unmarshal([]byte(c.SourceRefsJSON), &refs)
	for _, r := range refs {
		claim.SourceRefs = append(claim.SourceRefs, &servicesv1.EvidenceRef{
			Ref: r.Ref, Digest: r.Digest, MediaType: r.MediaType, BomKind: r.BomKind,
		})
	}
	var proofs []struct{ Type, Ref, Digest string }
	json.Unmarshal([]byte(c.ProofRefsJSON), &proofs)
	for _, p := range proofs {
		claim.ProofRefs = append(claim.ProofRefs, &servicesv1.ProofRef{
			Type: p.Type, Ref: p.Ref, Digest: p.Digest,
		})
	}
	var policies []struct{ Type, Ref, Digest string }
	json.Unmarshal([]byte(c.PolicyRefsJSON), &policies)
	for _, p := range policies {
		claim.PolicyRefs = append(claim.PolicyRefs, &servicesv1.PolicyRef{
			Type: p.Type, Ref: p.Ref, Digest: p.Digest,
		})
	}
	return claim
}

func evidenceFromProto(ev *servicesv1.Evidence) *Evidence {
	e := &Evidence{
		ID:             ev.GetId(),
		MediaType:      ev.GetMediaType(),
		BomKind:        ev.GetBomKind(),
		Digest:         ev.GetDigest(),
		SizeBytes:      ev.GetSizeBytes(),
		PredicateType:  ev.GetPredicateType(),
		Classification: ev.GetClassification(),
		ExtensionsJSON: ev.GetExtensionsJson(),
	}
	if ev.GetStorage() != nil {
		e.StorageJSON = toJSON(map[string]interface{}{
			"uris":         ev.GetStorage().GetUris(),
			"fetch_policy": ev.GetStorage().GetFetchPolicy(),
			"quorum":       ev.GetStorage().GetQuorum(),
		})
	}
	if ev.GetCreatedAt() != nil {
		e.CreatedAt = ev.GetCreatedAt().AsTime()
	}
	if ev.GetValidTime() != nil {
		e.ValidFrom = ev.GetValidTime().GetFromTime().AsTime()
		if ev.GetValidTime().GetToTime() != nil {
			t := ev.GetValidTime().GetToTime().AsTime()
			e.ValidTo = &t
		}
	}
	return e
}

func evidenceToProto(ev *Evidence) *servicesv1.Evidence {
	e := &servicesv1.Evidence{
		Id:             ev.ID,
		MediaType:      ev.MediaType,
		BomKind:        ev.BomKind,
		Digest:         ev.Digest,
		SizeBytes:      ev.SizeBytes,
		PredicateType:  ev.PredicateType,
		Classification: ev.Classification,
		ExtensionsJson: ev.ExtensionsJSON,
	}
	var storage map[string]interface{}
	json.Unmarshal([]byte(ev.StorageJSON), &storage)
	s := &servicesv1.Storage{}
	if uris, ok := storage["uris"].([]interface{}); ok {
		for _, u := range uris {
			s.Uris = append(s.Uris, u.(string))
		}
	}
	if fp, ok := storage["fetch_policy"].(string); ok {
		s.FetchPolicy = fp
	}
	if q, ok := storage["quorum"].(float64); ok {
		s.Quorum = int32(q)
	}
	e.Storage = s
	e.CreatedAt = timestamppb.New(ev.CreatedAt)
	e.ValidTime = &servicesv1.TimeWindow{FromTime: timestamppb.New(ev.ValidFrom)}
	if ev.ValidTo != nil {
		e.ValidTime.ToTime = timestamppb.New(*ev.ValidTo)
	}
	return e
}

func proofStateToProto(ps proofState) *servicesv1.ProofState {
	return &servicesv1.ProofState{
		Discovery:    ps.Discovery,
		Graph:        ps.Graph,
		Claim:        ps.Claim,
		Source:       ps.Source,
		Signature:    ps.Signature,
		Transparency: ps.Transparency,
		WitnessJson:  ps.WitnessJson,
		State:        ps.State,
		Policy:       ps.Policy,
	}
}

func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
