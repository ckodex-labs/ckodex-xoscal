package bundle

import (
	"path/filepath"
	"testing"
)

func TestMemberRegistrationUnicodeFilenameCompatibility(t *testing.T) {
	for _, pair := range [][2]string{{"s.json", "ſ.json"}, {"strasse.json", "straße.json"}, {"σ.json", "ς.json"}, {"Résumé.json", "résumé.json"}} {
		t.Run(pair[0], func(t *testing.T) {
			members := make(map[string]bool)
			if err := registerMember(members, "oscal/"+pair[0]); err != nil {
				t.Fatal(err)
			}
			if err := registerMember(members, "oscal/"+pair[1]); err == nil {
				t.Fatal("equivalent filename registered twice")
			}
		})
	}
	members := make(map[string]bool)
	for _, name := range []string{"oscal/résumé.json", "oscal/日本語.json", "oscal/Ελληνικά.json"} {
		if err := registerMember(members, name); err != nil {
			t.Fatalf("unique Unicode filename rejected: %v", err)
		}
	}
}

func TestPortableReservedFilenameCompatibility(t *testing.T) {
	for _, name := range []string{"CON", "con.txt", "PRN .log", "AUX", "nul.json", "COM1", "LPT9.txt", "COM¹.log", "LPT².txt", "com³", "CONIN$", "conout$.txt"} {
		if !reservedPortableBase(name) {
			t.Errorf("reserved device basename accepted: %s", name)
		}
		if err := canonicalMember("evidence/" + name); err == nil {
			t.Errorf("reserved portable member accepted: %s", name)
		}
	}
	for _, name := range []string{"CONSOLE", "PRINTER.log", "COM0", "COM10", "LPT0.json", "LPT¹0", "COM⁴.log", "report.txt"} {
		if reservedPortableBase(name) {
			t.Errorf("ordinary filename marked reserved: %s", name)
		}
		if err := canonicalMember("evidence/" + name); err != nil {
			t.Errorf("ordinary portable member rejected: %s: %v", name, err)
		}
	}
}

func TestProducerPreservesUniqueUnicodeFilenames(t *testing.T) {
	parent := t.TempDir()
	artifacts := []string{filepath.Join(parent, "résumé.json"), filepath.Join(parent, "日本語.json")}
	for _, artifact := range artifacts {
		mustBundleWrite(t, artifact, []byte(`{"title":"Unicode filename"}`))
	}
	output := filepath.Join(parent, "bundle.tar.gz")
	if err := CreateAuditBundle(artifacts, "", output); err != nil {
		t.Fatal(err)
	}
	manifest, err := VerifyBundle(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Artifacts) != len(artifacts) {
		t.Fatal("Unicode artifact filename omitted")
	}
	for i, artifact := range artifacts {
		if manifest.Artifacts[i].Path != "oscal/"+filepath.Base(artifact) {
			t.Fatalf("Unicode filename changed: %s", manifest.Artifacts[i].Path)
		}
	}
}
