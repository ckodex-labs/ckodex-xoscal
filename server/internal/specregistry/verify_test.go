package specregistry

import "testing"

func TestCompareModel_MatchAndDrift(t *testing.T) {
	data := []byte("schema-bytes")
	expected := Hash(data)
	if s := CompareModel("catalog", expected, data); s.Status != "match" {
		t.Errorf("match case Status = %q", s.Status)
	}
	s := CompareModel("ssp", expected, []byte("different"))
	if s.Status != "drift" {
		t.Errorf("drift case Status = %q", s.Status)
	}
	if s.Expected != expected || s.Actual != Hash([]byte("different")) {
		t.Errorf("status hashes wrong: %+v", s)
	}
}

func TestAggregate_AndExitCode(t *testing.T) {
	clean := Aggregate("1.2.2", []ModelStatus{{Status: "match"}, {Status: "match"}})
	if clean.Drift || clean.ExitCode() != 0 {
		t.Errorf("clean: drift=%v exit=%d", clean.Drift, clean.ExitCode())
	}
	dirty := Aggregate("1.2.2", []ModelStatus{{Status: "match"}, {Status: "drift"}})
	if !dirty.Drift || dirty.ExitCode() != 3 {
		t.Errorf("dirty: drift=%v exit=%d", dirty.Drift, dirty.ExitCode())
	}
}
