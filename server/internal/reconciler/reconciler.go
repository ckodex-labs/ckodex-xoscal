package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mchorfa/xoscal/server/internal/kg"
)

// ConflictType classifies the nature of a detected conflict.
type ConflictType string

const (
	ConflictTypeDuplicate       ConflictType = "duplicate"
	ConflictTypeVersionMismatch ConflictType = "version_mismatch"
	ConflictTypeSemanticDrift   ConflictType = "semantic_drift"
	ConflictTypeMappingGap      ConflictType = "mapping_gap"
)

// Conflict represents a discrepancy between two or more KG entities.
type Conflict struct {
	ID          string          `json:"id"`
	Type        ConflictType    `json:"type"`
	Description string          `json:"description"`
	Sources     []string        `json:"sources"`
	DetectedAt  time.Time       `json:"detected_at"`
	Resolved    bool            `json:"resolved"`
	Resolution  json.RawMessage `json:"resolution,omitempty"`
}

// Reconciler detects and records conflicts from proposed KG entities.
type Reconciler struct {
	store kg.Store
}

// NewReconciler creates a reconciler backed by a KG store.
func NewReconciler(store kg.Store) *Reconciler {
	return &Reconciler{store: store}
}

// Propose evaluates a batch of proposed entities, writes them to the KG, and
// returns any conflicts detected.
func (r *Reconciler) Propose(ctx context.Context, proposals []*kg.Entity) ([]*Conflict, error) {
	var conflicts []*Conflict
	for _, p := range proposals {
		c, err := r.evaluate(ctx, p)
		if err != nil {
			return nil, fmt.Errorf("evaluate %s: %w", p.URN, err)
		}
		if c != nil {
			if err := r.persistConflict(ctx, c); err != nil {
				return nil, fmt.Errorf("persist conflict: %w", err)
			}
			conflicts = append(conflicts, c)
			// Existing entity with different payload → update to create new version.
			if err := r.store.UpdateEntity(ctx, p); err != nil {
				return nil, fmt.Errorf("update entity %s: %w", p.URN, err)
			}
			continue
		}
		// No conflict: entity either doesn't exist or is identical.
		existing, err := r.store.GetEntity(ctx, p.URN)
		if err != nil {
			// Not found → create.
			if err := r.store.CreateEntity(ctx, p); err != nil {
				return nil, fmt.Errorf("create entity %s: %w", p.URN, err)
			}
		} else if !jsonEqual(existing.Payload, p.Payload) {
			// Different but no specific conflict rule matched → update anyway.
			if err := r.store.UpdateEntity(ctx, p); err != nil {
				return nil, fmt.Errorf("update entity %s: %w", p.URN, err)
			}
		}
		// If identical → no-op.
	}
	return conflicts, nil
}

func (r *Reconciler) evaluate(ctx context.Context, proposal *kg.Entity) (*Conflict, error) {
	existing, err := r.store.GetEntity(ctx, proposal.URN)
	if err != nil {
		// Not found → no conflict.
		return nil, nil
	}

	// Identical → no conflict.
	if jsonEqual(existing.Payload, proposal.Payload) {
		return nil, nil
	}

	// 1. VersionMismatch: same URN, different version in payload.
	if diff := fieldDiff(existing.Payload, proposal.Payload, "version"); diff != "" {
		return &Conflict{
			ID:          fmt.Sprintf("conflict-version-%s-%d", proposal.URN, time.Now().Unix()),
			Type:        ConflictTypeVersionMismatch,
			Description: fmt.Sprintf("Version mismatch for %s: %s", proposal.URN, diff),
			Sources:     []string{existing.URN, proposal.URN},
			DetectedAt:  time.Now().UTC(),
			Resolved:    false,
		}, nil
	}

	// 2. SemanticDrift: same text but different risk_level or role.
	if drift := detectSemanticDrift(existing.Payload, proposal.Payload); drift != "" {
		return &Conflict{
			ID:          fmt.Sprintf("conflict-drift-%s-%d", proposal.URN, time.Now().Unix()),
			Type:        ConflictTypeSemanticDrift,
			Description: fmt.Sprintf("Semantic drift for %s: %s", proposal.URN, drift),
			Sources:     []string{existing.URN, proposal.URN},
			DetectedAt:  time.Now().UTC(),
			Resolved:    false,
		}, nil
	}

	// 3. Duplicate: same URN but different payload (generic catch-all).
	return &Conflict{
		ID:          fmt.Sprintf("conflict-%s-%d", proposal.URN, time.Now().Unix()),
		Type:        ConflictTypeDuplicate,
		Description: fmt.Sprintf("URN %s already exists with different payload", proposal.URN),
		Sources:     []string{existing.URN, proposal.URN},
		DetectedAt:  time.Now().UTC(),
		Resolved:    false,
	}, nil
}

// EvaluateBatch detects MappingGap conflicts by scanning all mapping entities.
func (r *Reconciler) EvaluateBatch(ctx context.Context, proposals []*kg.Entity) ([]*Conflict, error) {
	// Collect all mappings (existing + proposed).
	existingMappings, err := r.store.ListEntities(ctx, "reg:Mapping", kg.EntityStatusActive)
	if err != nil {
		return nil, fmt.Errorf("list mappings: %w", err)
	}
	var allMappings []*kg.Entity
	allMappings = append(allMappings, existingMappings...)
	for _, p := range proposals {
		if p.Type == "reg:Mapping" {
			allMappings = append(allMappings, p)
		}
	}

	// Build reverse index: control URN → requirement URNs.
	controlToReqs := make(map[string][]string)
	for _, m := range allMappings {
		var mp kg.Mapping
		if err := json.Unmarshal(m.Payload, &mp); err != nil {
			continue
		}
		controlToReqs[mp.To] = append(controlToReqs[mp.To], mp.From)
	}

	var conflicts []*Conflict
	for controlURN, reqURNs := range controlToReqs {
		if len(reqURNs) > 1 {
			c := &Conflict{
				ID:          fmt.Sprintf("conflict-mapping-%s-%d", controlURN, time.Now().Unix()),
				Type:        ConflictTypeMappingGap,
				Description: fmt.Sprintf("Control %s mapped from multiple requirements: %v", controlURN, reqURNs),
				Sources:     reqURNs,
				DetectedAt:  time.Now().UTC(),
				Resolved:    false,
			}
			if err := r.persistConflict(ctx, c); err != nil {
				return nil, fmt.Errorf("persist mapping gap conflict: %w", err)
			}
			conflicts = append(conflicts, c)
		}
	}
	return conflicts, nil
}

func jsonEqual(a, b json.RawMessage) bool {
	var va, vb interface{}
	if err := json.Unmarshal(a, &va); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &vb); err != nil {
		return false
	}
	ab, _ := json.Marshal(va)
	bb, _ := json.Marshal(vb)
	return string(ab) == string(bb)
}

func fieldDiff(a, b json.RawMessage, field string) string {
	var ma, mb map[string]interface{}
	if err := json.Unmarshal(a, &ma); err != nil {
		return ""
	}
	if err := json.Unmarshal(b, &mb); err != nil {
		return ""
	}
	va, okA := ma[field]
	vb, okB := mb[field]
	if okA && okB && fmt.Sprint(va) != fmt.Sprint(vb) {
		return fmt.Sprintf("%s changed from %v to %v", field, va, vb)
	}
	return ""
}

func detectSemanticDrift(existingPayload, proposalPayload json.RawMessage) string {
	var existing, proposal map[string]interface{}
	if err := json.Unmarshal(existingPayload, &existing); err != nil {
		return ""
	}
	if err := json.Unmarshal(proposalPayload, &proposal); err != nil {
		return ""
	}

	// Compare text field equality.
	existingText, _ := existing["text"].(string)
	proposalText, _ := proposal["text"].(string)
	if existingText == "" || proposalText == "" || existingText != proposalText {
		return "" // different or missing text → not a drift of the same requirement
	}

	var diffs []string
	for _, field := range []string{"risk_level", "role", "lifecycle"} {
		va, okA := existing[field]
		vb, okB := proposal[field]
		if okA && okB && fmt.Sprint(va) != fmt.Sprint(vb) {
			diffs = append(diffs, fmt.Sprintf("%s: %v → %v", field, va, vb))
		}
	}
	return strings.Join(diffs, "; ")
}

func (r *Reconciler) persistConflict(ctx context.Context, c *Conflict) error {
	raw, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal conflict: %w", err)
	}
	entity := &kg.Entity{
		URN:     c.ID,
		Type:    "reg:Conflict",
		Payload: raw,
	}
	return r.store.CreateEntity(ctx, entity)
}

// ResolveConflict marks a conflict as resolved and appends resolution metadata.
func (r *Reconciler) ResolveConflict(ctx context.Context, conflictID string, resolution []byte) error {
	e, err := r.store.GetEntity(ctx, conflictID)
	if err != nil {
		return fmt.Errorf("get conflict %s: %w", conflictID, err)
	}
	var c Conflict
	if err := json.Unmarshal(e.Payload, &c); err != nil {
		return fmt.Errorf("unmarshal conflict: %w", err)
	}
	c.Resolved = true
	c.Resolution = resolution
	raw, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal resolved conflict: %w", err)
	}
	e.Payload = raw
	return r.store.UpdateEntity(ctx, e)
}

// ListConflicts returns all unresolved conflicts from the KG.
func (r *Reconciler) ListConflicts(ctx context.Context) ([]*Conflict, error) {
	entities, err := r.store.ListEntities(ctx, "reg:Conflict", kg.EntityStatusActive)
	if err != nil {
		return nil, fmt.Errorf("list conflicts: %w", err)
	}
	var out []*Conflict
	for _, e := range entities {
		var c Conflict
		if err := json.Unmarshal(e.Payload, &c); err != nil {
			continue
		}
		if !c.Resolved {
			out = append(out, &c)
		}
	}
	return out, nil
}
