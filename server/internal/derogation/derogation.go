// Package derogation implements Constitutional Rule 23:
// "Accepted risk does not rewrite history. A derogation binds: failed requirement,
// scope, authority, justification, compensating controls, evidence, expiry,
// re-evaluation trigger. The underlying vector remains truthful."
package derogation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DerogationEntry represents a single formal time-bounded risk acceptance.
type DerogationEntry struct {
	ID                   string    `json:"id"`
	ControlID            string    `json:"control_id"`
	Scope                string    `json:"scope"`
	Authority            string    `json:"authority"` // URN of authorizing risk owner
	Justification        string    `json:"justification"`
	CompensatingControls []string  `json:"compensating_controls"`
	EvidenceDigest       string    `json:"evidence_digest,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	ExpiresAt            time.Time `json:"expires_at"`
	ReevaluationTrigger  string    `json:"reevaluation_trigger"`
	Revoked              bool      `json:"revoked"`
}

// IsActive returns true if the derogation is not revoked and not expired.
func (d *DerogationEntry) IsActive(t time.Time) bool {
	if d.Revoked {
		return false
	}
	return t.Before(d.ExpiresAt)
}

// IsExpired returns true if the derogation is not revoked and its TTL has lapsed.
func (d *DerogationEntry) IsExpired(t time.Time) bool {
	if d.Revoked {
		return false
	}
	return !t.Before(d.ExpiresAt)
}

// TimeRemaining returns duration until expiry (or 0 if already expired).
func (d *DerogationEntry) TimeRemaining(t time.Time) time.Duration {
	if d.IsExpired(t) {
		return 0
	}
	return d.ExpiresAt.Sub(t)
}

// Store manages the persistence and retrieval of derogation entries.
type Store struct {
	Entries []DerogationEntry `json:"derogations"`
}

// Load reads derogations from disk. If the file does not exist, an empty store is returned.
func Load(path string) (*Store, error) {
	s := &Store{Entries: make([]DerogationEntry, 0)}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read derogations file: %w", err)
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("unmarshal derogations: %w", err)
	}
	return s, nil
}

// Save writes the derogations store to disk.
func (s *Store) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("create derogation dir: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal derogations: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// Add appends a new derogation entry.
func (s *Store) Add(entry DerogationEntry) error {
	if strings.TrimSpace(entry.ControlID) == "" {
		return fmt.Errorf("control_id is required")
	}
	if strings.TrimSpace(entry.Authority) == "" {
		return fmt.Errorf("authority URN is required")
	}
	if strings.TrimSpace(entry.Justification) == "" {
		return fmt.Errorf("justification is required")
	}
	if entry.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("expires_at must be in the future")
	}
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("derog-%s-%d", entry.ControlID, time.Now().Unix())
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}

	s.Entries = append(s.Entries, entry)
	return nil
}

// GetActiveForControl returns the active derogation for a control, if one exists.
func (s *Store) GetActiveForControl(controlID string, t time.Time) *DerogationEntry {
	for i := range s.Entries {
		e := &s.Entries[i]
		if strings.EqualFold(e.ControlID, controlID) && e.IsActive(t) {
			return e
		}
	}
	return nil
}

// GetExpiredForControl returns the expired derogation for a control, if one exists.
func (s *Store) GetExpiredForControl(controlID string, t time.Time) *DerogationEntry {
	for i := range s.Entries {
		e := &s.Entries[i]
		if strings.EqualFold(e.ControlID, controlID) && e.IsExpired(t) {
			return e
		}
	}
	return nil
}

// Revoke marks a derogation ID as revoked.
func (s *Store) Revoke(id string) error {
	for i := range s.Entries {
		if s.Entries[i].ID == id {
			s.Entries[i].Revoked = true
			return nil
		}
	}
	return fmt.Errorf("derogation with ID %s not found", id)
}
