package transparency

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (s *SQLiteStore) CreateClaim(ctx context.Context, claim *Claim) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO kg_claims(id, claim_type, subject_json, predicate_json, object_json, issuer_json, bom_kind,
		valid_from, valid_to, observed_time, source_refs_json, proof_refs_json, policy_refs_json, extensions_json,
		trust_state, proof_state_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		claim.ID, claim.Type, claim.SubjectJSON, claim.PredicateJSON, claim.ObjectJSON,
		claim.IssuerJSON, claim.BomKind, claim.ValidFrom, claim.ValidTo, claim.ObservedTime,
		claim.SourceRefsJSON, claim.ProofRefsJSON, claim.PolicyRefsJSON, claim.ExtensionsJSON,
		claim.TrustState, claim.ProofStateJSON, claim.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert claim: %w", err)
	}
	return nil
}

// CreateImportBatch commits every claim and evidence object in one SQLite
// transaction. Duplicate-equivalent records are idempotent; any conflicting
// identity or digest aborts the complete batch.

// CreateImportBatch commits every claim and evidence object in one SQLite
// transaction. Duplicate-equivalent records are idempotent; any conflicting
// identity or digest aborts the complete batch.
func (s *SQLiteStore) CreateImportBatch(ctx context.Context, records []*ImportRecord) error {
	if len(records) == 0 {
		return fmt.Errorf("import batch is empty")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin import batch: %w", err)
	}
	rollback := func(cause error) error {
		_ = tx.Rollback()
		return cause
	}
	for _, record := range records {
		if record == nil || record.Claim == nil {
			return rollback(fmt.Errorf("import record is empty"))
		}
		for _, evidence := range record.Evidence {
			if evidence == nil {
				return rollback(fmt.Errorf("import evidence is empty for claim %q", record.Claim.ID))
			}
			if err := insertEvidenceTx(ctx, tx, evidence); err != nil {
				return rollback(err)
			}
		}
		if err := insertClaimTx(ctx, tx, record.Claim); err != nil {
			return rollback(err)
		}
		if err := linkClaimEvidenceTx(ctx, tx, record.Claim); err != nil {
			return rollback(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit import batch: %w", err)
	}
	return nil
}

func insertEvidenceTx(ctx context.Context, tx *sql.Tx, ev *Evidence) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO kg_evidence(id, media_type, bom_kind, digest, size_bytes, storage_json, produced_by_json,
		subject_refs_json, predicate_type, spec_json, created_at, valid_from, valid_to, supersedes_json,
		integrity_methods_json, classification, extensions_json, blob)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ev.ID, ev.MediaType, ev.BomKind, ev.Digest, ev.SizeBytes, ev.StorageJSON, ev.ProducedByJSON,
		ev.SubjectRefsJSON, ev.PredicateType, ev.SpecJSON, ev.CreatedAt, ev.ValidFrom, ev.ValidTo,
		ev.SupersedesJSON, ev.IntegrityMethodsJSON, ev.Classification, ev.ExtensionsJSON, ev.Blob,
	)
	if err == nil {
		return nil
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unique") {
		return fmt.Errorf("insert batch evidence %q: %w", ev.ID, err)
	}
	existing, lookupErr := scanEvidence(tx.QueryRowContext(ctx, `SELECT id, media_type, bom_kind, digest, size_bytes, storage_json, produced_by_json,
		subject_refs_json, predicate_type, spec_json, created_at, valid_from, valid_to, supersedes_json,
		integrity_methods_json, classification, extensions_json, blob FROM kg_evidence WHERE id = ?`, ev.ID))
	if lookupErr == sql.ErrNoRows {
		existing, lookupErr = scanEvidence(tx.QueryRowContext(ctx, `SELECT id, media_type, bom_kind, digest, size_bytes, storage_json, produced_by_json,
			subject_refs_json, predicate_type, spec_json, created_at, valid_from, valid_to, supersedes_json,
			integrity_methods_json, classification, extensions_json, blob FROM kg_evidence WHERE digest = ?`, ev.Digest))
	}
	if lookupErr != nil {
		return fmt.Errorf("resolve duplicate batch evidence %q: %w", ev.ID, lookupErr)
	}
	if !evidenceEquivalent(existing, ev) || digestFor(existing.Blob) != digestFor(ev.Blob) {
		return fmt.Errorf("batch evidence %q conflicts with an existing content address", ev.ID)
	}
	return nil
}

func insertClaimTx(ctx context.Context, tx *sql.Tx, claim *Claim) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO kg_claims(id, claim_type, subject_json, predicate_json, object_json, issuer_json, bom_kind,
		valid_from, valid_to, observed_time, source_refs_json, proof_refs_json, policy_refs_json, extensions_json,
		trust_state, proof_state_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		claim.ID, claim.Type, claim.SubjectJSON, claim.PredicateJSON, claim.ObjectJSON, claim.IssuerJSON,
		claim.BomKind, claim.ValidFrom, claim.ValidTo, claim.ObservedTime, claim.SourceRefsJSON,
		claim.ProofRefsJSON, claim.PolicyRefsJSON, claim.ExtensionsJSON, claim.TrustState,
		claim.ProofStateJSON, claim.CreatedAt,
	)
	if err == nil {
		return nil
	}
	if !strings.Contains(strings.ToLower(err.Error()), "unique") {
		return fmt.Errorf("insert batch claim %q: %w", claim.ID, err)
	}
	existing, lookupErr := scanClaim(tx.QueryRowContext(ctx, `SELECT id, claim_type, subject_json, predicate_json, object_json, issuer_json, bom_kind,
		valid_from, valid_to, observed_time, source_refs_json, proof_refs_json, policy_refs_json, extensions_json,
		trust_state, proof_state_json, created_at FROM kg_claims WHERE id = ?`, claim.ID))
	if lookupErr != nil {
		return fmt.Errorf("resolve duplicate batch claim %q: %w", claim.ID, lookupErr)
	}
	if !claimsEquivalent(existing, claim) {
		return fmt.Errorf("batch claim %q conflicts with an existing claim", claim.ID)
	}
	return nil
}

func linkClaimEvidenceTx(ctx context.Context, tx *sql.Tx, claim *Claim) error {
	var refs []struct {
		Ref    string `json:"ref"`
		Digest string `json:"digest"`
	}
	for _, raw := range []string{claim.SourceRefsJSON, claim.ProofRefsJSON, claim.PolicyRefsJSON} {
		refs = nil
		if raw == "" || raw == "null" {
			continue
		}
		if err := json.Unmarshal([]byte(raw), &refs); err != nil {
			return fmt.Errorf("decode claim evidence references: %w", err)
		}
		for _, ref := range refs {
			var evidenceID string
			var err error
			if ref.Digest != "" {
				err = tx.QueryRowContext(ctx, `SELECT id FROM kg_evidence WHERE digest = ?`, ref.Digest).Scan(&evidenceID)
			} else if ref.Ref != "" {
				err = tx.QueryRowContext(ctx, `SELECT id FROM kg_evidence WHERE id = ?`, ref.Ref).Scan(&evidenceID)
			} else {
				return fmt.Errorf("claim %q has an empty evidence reference", claim.ID)
			}
			if err != nil {
				return fmt.Errorf("resolve evidence for claim %q: %w", claim.ID, err)
			}
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO kg_claim_evidence(claim_id, evidence_id) VALUES (?, ?)`, claim.ID, evidenceID); err != nil {
				return fmt.Errorf("link evidence for claim %q: %w", claim.ID, err)
			}
		}
	}
	return nil
}

func (s *SQLiteStore) GetClaim(ctx context.Context, id string) (*Claim, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, claim_type, subject_json, predicate_json, object_json, issuer_json, bom_kind,
		valid_from, valid_to, observed_time, source_refs_json, proof_refs_json, policy_refs_json, extensions_json,
		trust_state, proof_state_json, created_at FROM kg_claims WHERE id = ?`, id)
	return scanClaim(row)
}

func (s *SQLiteStore) ListClaims(ctx context.Context, subjectDigest, bomKind, relation, trustState string, validAfter time.Time, limit int, pageToken string) ([]*Claim, string, error) {
	if limit <= 0 {
		limit = 50
	}
	offset, err := parsePageToken(pageToken)
	if err != nil {
		return nil, "", err
	}
	query := `SELECT id, claim_type, subject_json, predicate_json, object_json, issuer_json, bom_kind,
		valid_from, valid_to, observed_time, source_refs_json, proof_refs_json, policy_refs_json, extensions_json,
		trust_state, proof_state_json, created_at FROM kg_claims WHERE 1=1`
	args := []interface{}{}
	if subjectDigest != "" {
		query += ` AND json_extract(subject_json, '$.digest') = ?`
		args = append(args, subjectDigest)
	}
	if bomKind != "" {
		query += ` AND bom_kind = ?`
		args = append(args, bomKind)
	}
	if relation != "" {
		query += ` AND json_extract(predicate_json, '$.relation') = ?`
		args = append(args, relation)
	}
	if trustState != "" {
		query += ` AND trust_state = ?`
		args = append(args, trustState)
	}
	if !validAfter.IsZero() {
		query += ` AND valid_from >= ?`
		args = append(args, validAfter)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`
	args = append(args, limit+1, offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("list claims: %w", err)
	}
	defer rows.Close()
	var out []*Claim
	hasMore := false
	for rows.Next() {
		c, err := scanClaim(rows)
		if err != nil {
			return nil, "", err
		}
		if len(out) == limit {
			hasMore = true
			break
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	nextToken := ""
	if hasMore {
		nextToken = strconv.Itoa(offset + limit)
	}
	return out, nextToken, nil
}

func parsePageToken(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(token)
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("invalid page token")
	}
	return offset, nil
}

func (s *SQLiteStore) UpdateClaimTrustState(ctx context.Context, id, trustState string, proofStateJSON string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE kg_claims SET trust_state = ?, proof_state_json = ? WHERE id = ?`,
		trustState, proofStateJSON, id)
	return err
}

// RecordVerification atomically updates the current claim projection and
// appends the corresponding audit event. Repeating the same idempotency key
// returns the existing event without creating a second history record.
