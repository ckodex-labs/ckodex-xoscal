package ingestion

import (
	"context"
	"testing"
)

var cmmcTestYAML = []byte(`
urn: urn:intuitem:risk:library:cmmc-2.0
locale: en
ref_id: CMMC-2.0
name: CMMC version 2.0
description: Cybersecurity Maturity Model Certification
version: 6
publication_date: 2026-03-05
provider: DoD
packager: intuitem
objects:
  framework:
    urn: urn:intuitem:risk:framework:cmmc-2.0
    ref_id: CMMC-2.0
    name: CMMC version 2.0
    description: Cybersecurity Maturity Model Certification
    implementation_groups_definition:
      - ref_id: L1
        name: Level 1
      - ref_id: L2
        name: Level 2
    requirement_nodes:
      - urn: urn:intuitem:risk:req_node:cmmc-2.0:ac
        assessable: false
        depth: 1
        ref_id: AC
        name: ACCESS CONTROL
      - urn: urn:intuitem:risk:req_node:cmmc-2.0:ac.l1-3.1.1
        assessable: true
        depth: 2
        parent_urn: urn:intuitem:risk:req_node:cmmc-2.0:ac
        ref_id: AC.L1-3.1.1
        name: Authorized Access Control
        description: Limit information system access to authorized users.
        implementation_groups:
          - L1
`)

func TestCISOAssistantParser_Parse(t *testing.T) {
	p := &CISOAssistantParser{}
	reqs, err := p.Parse(context.Background(), cmmcTestYAML)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(reqs) != 2 {
		t.Fatalf("expected 2 requirements, got %d", len(reqs))
	}

	// Check root node
	if reqs[0].Assessable {
		t.Errorf("root node AC should not be assessable")
	}
	if reqs[0].Depth != 1 {
		t.Errorf("root depth expected 1, got %d", reqs[0].Depth)
	}

	// Check leaf node
	if !reqs[1].Assessable {
		t.Errorf("leaf node AC.L1-3.1.1 should be assessable")
	}
	if reqs[1].Depth != 2 {
		t.Errorf("leaf depth expected 2, got %d", reqs[1].Depth)
	}
	if reqs[1].ParentURN == "" {
		t.Errorf("leaf node should have parent_urn")
	}
	if len(reqs[1].ImplementationGroups) != 1 || reqs[1].ImplementationGroups[0] != "L1" {
		t.Errorf("expected implementation group L1, got %v", reqs[1].ImplementationGroups)
	}
}

func TestCISOAssistantParser_ParseEmpty(t *testing.T) {
	p := &CISOAssistantParser{}
	reqs, err := p.Parse(context.Background(), []byte(""))
	if err != nil {
		t.Fatalf("unexpected error for empty input: %v", err)
	}
	if len(reqs) != 0 {
		t.Fatalf("expected 0 requirements for empty input, got %d", len(reqs))
	}
}
