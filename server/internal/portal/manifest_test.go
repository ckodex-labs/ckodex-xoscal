package portal

import "testing"

const sampleManifest = `version: "1.0"
source_base_url: "https://raw.githubusercontent.com/intuitem/ciso-assistant-community/main/backend/library/libraries"
frameworks:
  - ref_id: nist-csf-2.0
    jurisdiction: usa
    filename: nist-csf-2.0.yaml
  - ref_id: dora
    jurisdiction: eu
    filename: dora.yaml
`

func TestParseManifest_Source(t *testing.T) {
	m, err := ParseManifest([]byte(sampleManifest))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	owner, repo, path, err := m.Source()
	if err != nil {
		t.Fatalf("Source: %v", err)
	}
	if owner != "intuitem" || repo != "ciso-assistant-community" || path != "backend/library/libraries" {
		t.Errorf("got %s/%s/%s", owner, repo, path)
	}
	if ids := m.RefIDs(); len(ids) != 2 || ids[0] != "nist-csf-2.0" {
		t.Errorf("RefIDs = %v", ids)
	}
}
