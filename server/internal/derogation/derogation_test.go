package derogation

import (
	"path/filepath"
	"testing"
	"time"
)

func TestDerogation_Lifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, ".xoscal", "derogations.json")

	store, err := Load(storePath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	now := time.Now().UTC()
	future := now.Add(14 * 24 * time.Hour) // 14 days TTL

	entry := DerogationEntry{
		ControlID:            "sc-8",
		Scope:                "service:worker",
		Authority:            "urn:xoscal:authority:ciso-alice",
		Justification:        "Isolated internal VPC worker; no external egress permitted",
		CompensatingControls: []string{"sc-7 boundary protection", "au-2 audit logs"},
		ExpiresAt:            future,
		ReevaluationTrigger:  "Next quarterly network architecture review",
	}

	if err := store.Add(entry); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if err := store.Save(storePath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Reload from disk
	reloaded, err := Load(storePath)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	active := reloaded.GetActiveForControl("sc-8", now)
	if active == nil {
		t.Fatalf("expected active derogation for sc-8")
	}
	if active.TimeRemaining(now) <= 0 {
		t.Errorf("expected positive time remaining")
	}

	// Verify expiration check
	futureCheck := future.Add(1 * time.Hour)
	expired := reloaded.GetExpiredForControl("sc-8", futureCheck)
	if expired == nil {
		t.Fatalf("expected expired derogation at future time")
	}

	// Test Revocation
	if err := reloaded.Revoke(active.ID); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}
	if reloaded.GetActiveForControl("sc-8", now) != nil {
		t.Errorf("revoked derogation should not be active")
	}
}
