package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/scaffold"
)

// The path-looking input is untrusted artifact/CSV data. Only the operator's
// --out flag may select a local write destination, including an absolute path.
func TestTabularCommandsDoNotUseContentAsOutputPaths(t *testing.T) {
	workspace := t.TempDir()
	_, artifact, err := scaffold.ScaffoldWorkspace(workspace, "nist-sp-800-53-rev5")
	if err != nil {
		t.Fatal(err)
	}
	protected := filepath.Join(t.TempDir(), "must-not-change")
	marker := []byte("protected local file")
	if err := os.WriteFile(protected, marker, 0600); err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(artifact, &document); err != nil {
		t.Fatal(err)
	}
	component := document["component-definition"].(map[string]any)["components"].([]any)[0].(map[string]any)
	component["title"] = protected
	maliciousArtifact, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(workspace, "input.json")
	if err := os.WriteFile(input, maliciousArtifact, 0600); err != nil {
		t.Fatal(err)
	}
	outputCSV := filepath.Join(t.TempDir(), "requested.csv")
	runTabular([]string{"export", "-in", input, "-out", outputCSV})
	data, err := os.ReadFile(outputCSV)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 2 {
		t.Fatal("export has no control rows")
	}
	if rows[1][1] != protected {
		t.Fatal("crafted component path did not reach the actual export data")
	}
	rows[0] = append(rows[0], "Output Path")
	for i := 1; i < len(rows); i++ {
		rows[i][5] = "../../" + filepath.Base(protected)
		rows[i] = append(rows[i], protected)
	}
	var edited bytes.Buffer
	writer := csv.NewWriter(&edited)
	if err := writer.WriteAll(rows); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outputCSV, edited.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	outputJSON := filepath.Join(t.TempDir(), "requested.json")
	runTabular([]string{"import", "-in", outputCSV, "-base", input, "-out", outputJSON})
	updated, err := os.ReadFile(outputJSON)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(updated, []byte("../../"+filepath.Base(protected))) {
		t.Fatal("crafted path did not reach actual imported output data")
	}
	unchanged, err := os.ReadFile(protected)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(unchanged, marker) {
		t.Fatal("input content selected a write destination")
	}
	for _, path := range []string{outputCSV, outputJSON} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("output %s permissions = %o, want 0600", path, info.Mode().Perm())
		}
	}
}
