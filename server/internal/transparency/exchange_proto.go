package transparency

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func verificationEventToProto(event *VerificationEvent) (*servicesv1.VerificationEvent, error) {
	var diagnostics []string
	if event.DiagnosticsJSON == "" {
		diagnostics = []string{}
	} else if err := json.Unmarshal([]byte(event.DiagnosticsJSON), &diagnostics); err != nil {
		return nil, fmt.Errorf("diagnostics JSON: %w", err)
	}
	return &servicesv1.VerificationEvent{
		Sequence:          event.Sequence,
		EventId:           event.EventID,
		ClaimId:           event.ClaimID,
		RequestId:         event.RequestID,
		IdempotencyKey:    event.IdempotencyKey,
		ChecksJson:        event.ChecksJSON,
		ProofStateJson:    event.ProofStateJSON,
		ProviderStateJson: event.ProviderStateJSON,
		TrustState:        event.TrustState,
		Diagnostics:       diagnostics,
		PreviousHash:      event.PreviousHash,
		EventHash:         event.EventHash,
		CreatedAt:         timestamppb.New(event.CreatedAt),
	}, nil
}

func mapStoreError(operation, id string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return status.Errorf(codes.NotFound, "%s %q not found", operation, id)
	}
	return status.Errorf(codes.Internal, "%s: %v", operation, err)
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
		TrustState:     c.TrustState,
		ProofStateJson: c.ProofStateJSON,
	}
	var subject map[string]string
	decodeJSON(c.SubjectJSON, &subject)
	claim.Subject = &servicesv1.Reference{
		Kind:   subject["kind"],
		Id:     subject["id"],
		Digest: subject["digest"],
		Purl:   subject["purl"],
		BomRef: subject["bom_ref"],
	}
	var predicate map[string]string
	decodeJSON(c.PredicateJSON, &predicate)
	claim.Predicate = &servicesv1.Predicate{
		Relation:  predicate["relation"],
		Direction: predicate["direction"],
		Qualifier: predicate["qualifier"],
	}
	if c.ObjectJSON != "" {
		var obj map[string]string
		decodeJSON(c.ObjectJSON, &obj)
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
	decodeJSON(c.IssuerJSON, &issuer)
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
	decodeJSON(c.SourceRefsJSON, &refs)
	for _, r := range refs {
		claim.SourceRefs = append(claim.SourceRefs, &servicesv1.EvidenceRef{
			Ref: r.Ref, Digest: r.Digest, MediaType: r.MediaType, BomKind: r.BomKind,
		})
	}
	var proofs []struct{ Type, Ref, Digest string }
	decodeJSON(c.ProofRefsJSON, &proofs)
	for _, p := range proofs {
		claim.ProofRefs = append(claim.ProofRefs, &servicesv1.ProofRef{
			Type: p.Type, Ref: p.Ref, Digest: p.Digest,
		})
	}
	var policies []struct{ Type, Ref, Digest string }
	decodeJSON(c.PolicyRefsJSON, &policies)
	for _, p := range policies {
		claim.PolicyRefs = append(claim.PolicyRefs, &servicesv1.PolicyRef{
			Type: p.Type, Ref: p.Ref, Digest: p.Digest,
		})
	}
	return claim
}

// ClaimToProto exposes the canonical exchange mapping to the graph projection
// and other presentation adapters. Keeping one mapper prevents divergent
// serialization of proof-bearing records across services.

// ClaimToProto exposes the canonical exchange mapping to the graph projection
// and other presentation adapters. Keeping one mapper prevents divergent
// serialization of proof-bearing records across services.
func ClaimToProto(c *Claim) *servicesv1.Claim { return claimToProto(c) }

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
	decodeJSON(ev.StorageJSON, &storage)
	s := &servicesv1.Storage{}
	if uris, ok := storage["uris"].([]interface{}); ok {
		for _, u := range uris {
			if uri, ok := u.(string); ok {
				s.Uris = append(s.Uris, uri)
			}
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

// EvidenceToProto exposes the canonical exchange mapping to presentation
// adapters without exposing stored evidence bytes.

// EvidenceToProto exposes the canonical exchange mapping to presentation
// adapters without exposing stored evidence bytes.
func EvidenceToProto(ev *Evidence) *servicesv1.Evidence { return evidenceToProto(ev) }

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

// decodeJSON keeps malformed optional persistence fields from becoming a
// panic during response conversion; callers receive the target's zero value.

// decodeJSON keeps malformed optional persistence fields from becoming a
// panic during response conversion; callers receive the target's zero value.
func decodeJSON(data string, target any) {
	if data == "" {
		return
	}
	if err := json.Unmarshal([]byte(data), target); err != nil {
		return
	}
}

func toJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
