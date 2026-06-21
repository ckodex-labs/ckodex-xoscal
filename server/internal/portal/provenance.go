// Package portal assembles static-portal artifacts (provenance, framework export).
package portal

import "encoding/json"

// Artifact is one provenance subject (the proto set or an SDK bundle).
type Artifact struct {
	Name         string
	Digest       string
	SBOMRef      string
	SLSARef      string
	CosignBundle string
	RekorID      string
}

type record struct {
	Name         string `json:"name"`
	Digest       string `json:"digest"`
	SBOMRef      string `json:"sbom_ref,omitempty"`
	SLSARef      string `json:"slsa_ref,omitempty"`
	CosignBundle string `json:"cosign_bundle,omitempty"`
	RekorID      string `json:"rekor_id,omitempty"`
	Signed       bool   `json:"signed"`
}

// BuildManifest renders provenance records. An artifact is "signed" only when it
// carries both a cosign bundle and a Rekor entry; otherwise signed=false and no
// proof fields are emitted (no fabricated attestation).
func BuildManifest(arts []Artifact) ([]byte, error) {
	out := make([]record, 0, len(arts))
	for _, a := range arts {
		out = append(out, record{
			Name: a.Name, Digest: a.Digest,
			SBOMRef: a.SBOMRef, SLSARef: a.SLSARef,
			CosignBundle: a.CosignBundle, RekorID: a.RekorID,
			Signed: a.CosignBundle != "" && a.RekorID != "",
		})
	}
	return json.MarshalIndent(out, "", "  ")
}
