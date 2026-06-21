package specregistry

import (
	"strings"
	"testing"
)

func TestSchemaURL(t *testing.T) {
	got := SchemaURL("1.2.2", "oscal_catalog_schema.json")
	want := "https://github.com/usnistgov/OSCAL/releases/download/v1.2.2/oscal_catalog_schema.json"
	if got != want {
		t.Errorf("SchemaURL = %q, want %q", got, want)
	}
}

func TestHash_Format(t *testing.T) {
	h := Hash([]byte("hello"))
	if !strings.HasPrefix(h, "sha256:") {
		t.Errorf("Hash missing sha256: prefix: %q", h)
	}
	// sha256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	if h != "sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Errorf("Hash = %q", h)
	}
}

func TestIsPopulated(t *testing.T) {
	empty := &Registry{Models: []Model{{Model: "catalog", SchemaSHA256: ""}}}
	if empty.IsPopulated() {
		t.Error("registry with empty hash should be unpopulated")
	}
	full := &Registry{Models: []Model{{Model: "catalog", SchemaSHA256: "sha256:abc"}}}
	if !full.IsPopulated() {
		t.Error("registry with all hashes should be populated")
	}
	if (&Registry{}).IsPopulated() {
		t.Error("registry with no models should be unpopulated")
	}
}

func TestParseSerialize_Roundtrip(t *testing.T) {
	in := []byte("version: \"1.0\"\noscal_version: \"1.2.2\"\ngenerated_at: \"\"\nmodels:\n  - model: catalog\n    schema_asset: oscal_catalog_schema.json\n    schema_sha256: \"\"\n    proto: proto/oscal/catalog/v1/catalog.proto\n")
	r, err := Parse(in)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.OSCALVersion != "1.2.2" || len(r.Models) != 1 || r.Models[0].Model != "catalog" {
		t.Fatalf("parsed wrong: %+v", r)
	}
	out, err := r.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	if r2, err := Parse(out); err != nil || r2.Models[0].SchemaAsset != "oscal_catalog_schema.json" {
		t.Fatalf("roundtrip failed: %v %+v", err, r2)
	}
}
