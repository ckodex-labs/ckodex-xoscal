package portal

import (
	"encoding/json"
	"testing"
)

func TestBuildManifest_UnsignedHasNoProof(t *testing.T) {
	out, err := BuildManifest([]Artifact{{Name: "go-sdk", Digest: "sha256:abc"}})
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got[0]["signed"] != false {
		t.Errorf("signed = %v, want false", got[0]["signed"])
	}
	if _, ok := got[0]["cosign_bundle"]; ok {
		t.Errorf("cosign_bundle present on unsigned artifact")
	}
}

func TestBuildManifest_SignedWhenBundleAndRekor(t *testing.T) {
	out, _ := BuildManifest([]Artifact{{
		Name: "proto-set", Digest: "sha256:def",
		CosignBundle: "cosign.bundle", RekorID: "1234567",
	}})
	var got []map[string]any
	_ = json.Unmarshal(out, &got)
	if got[0]["signed"] != true {
		t.Errorf("signed = %v, want true", got[0]["signed"])
	}
}
