package transparency

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

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
	if ps.Source == "source_bound" && (ps.Signature != "signature_verified" ||
		ps.Policy != "policy_admissible" || ps.State != "current_state_verified") {
		return "incomplete"
	}
	return "candidate"
}

func digestFor(blob []byte) string {
	sum := sha256.Sum256(blob)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func claimsEquivalent(left, right *Claim) bool {
	return left.Type == right.Type && left.SubjectJSON == right.SubjectJSON &&
		left.PredicateJSON == right.PredicateJSON && left.ObjectJSON == right.ObjectJSON &&
		left.IssuerJSON == right.IssuerJSON && left.BomKind == right.BomKind &&
		left.ValidFrom.Equal(right.ValidFrom) && sameOptionalTime(left.ValidTo, right.ValidTo) &&
		left.SourceRefsJSON == right.SourceRefsJSON &&
		left.ProofRefsJSON == right.ProofRefsJSON && left.PolicyRefsJSON == right.PolicyRefsJSON &&
		left.ExtensionsJSON == right.ExtensionsJSON
}

func sameOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func evidenceEquivalent(left, right *Evidence) bool {
	return left.Digest == right.Digest && left.SizeBytes == right.SizeBytes
}

func sameJSON(left, right string) bool {
	var leftValue, rightValue interface{}
	if json.Unmarshal([]byte(left), &leftValue) != nil || json.Unmarshal([]byte(right), &rightValue) != nil {
		return left == right
	}
	leftCanonical, leftErr := json.Marshal(leftValue)
	rightCanonical, rightErr := json.Marshal(rightValue)
	return leftErr == nil && rightErr == nil && string(leftCanonical) == string(rightCanonical)
}
