package transparency

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// RecordVerification atomically updates the current claim projection and
// appends the corresponding audit event. Repeating the same idempotency key
// returns the existing event without creating a second history record.
func (s *SQLiteStore) RecordVerification(ctx context.Context, claimID, trustState, proofStateJSON string, event *VerificationEvent) (*VerificationEvent, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin verification transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	existing, err := scanVerificationEvent(tx.QueryRowContext(ctx,
		`SELECT sequence, event_id, claim_id, request_id, idempotency_key, checks_json,
		proof_state_json, provider_state_json, trust_state, diagnostics_json,
		previous_hash, event_hash, created_at
		FROM verification_events WHERE idempotency_key = ?`, event.IdempotencyKey))
	if err == nil {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit existing verification: %w", err)
		}
		return existing, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("lookup verification idempotency key: %w", err)
	}

	result, err := tx.ExecContext(ctx,
		`UPDATE kg_claims SET trust_state = ?, proof_state_json = ? WHERE id = ?`,
		trustState, proofStateJSON, claimID)
	if err != nil {
		return nil, fmt.Errorf("update claim verification state: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil || rows != 1 {
		if err != nil {
			return nil, fmt.Errorf("verify claim update count: %w", err)
		}
		return nil, sql.ErrNoRows
	}

	var previousHash string
	if err := tx.QueryRowContext(ctx,
		`SELECT event_hash FROM verification_events ORDER BY sequence DESC LIMIT 1`).Scan(&previousHash); err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("read previous verification hash: %w", err)
	}
	event.Sequence = 0
	event.ClaimID = claimID
	event.TrustState = trustState
	event.ProofStateJSON = proofStateJSON
	event.PreviousHash = previousHash
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	event.EventHash = hashVerificationEvent(event)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO verification_events(event_id, claim_id, request_id, idempotency_key,
		checks_json, proof_state_json, provider_state_json, trust_state, diagnostics_json,
		previous_hash, event_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.EventID, event.ClaimID, event.RequestID, event.IdempotencyKey,
		event.ChecksJSON, event.ProofStateJSON, event.ProviderStateJSON, event.TrustState,
		event.DiagnosticsJSON, event.PreviousHash, event.EventHash, event.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("append verification event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit verification: %w", err)
	}
	return event, nil
}

func (s *SQLiteStore) ListVerificationEvents(ctx context.Context, claimID string) ([]*VerificationEvent, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT sequence, event_id, claim_id, request_id, idempotency_key, checks_json,
		proof_state_json, provider_state_json, trust_state, diagnostics_json,
		previous_hash, event_hash, created_at
		FROM verification_events WHERE claim_id = ? ORDER BY sequence ASC`, claimID)
	if err != nil {
		return nil, fmt.Errorf("list verification events: %w", err)
	}
	defer rows.Close()
	var events []*VerificationEvent
	for rows.Next() {
		event, err := scanVerificationEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list verification events: %w", err)
	}
	return events, nil
}

// VerifyVerificationChain validates the global append-only hash chain.

// VerifyVerificationChain validates the global append-only hash chain.
func (s *SQLiteStore) VerifyVerificationChain(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx,
		`SELECT sequence, event_id, claim_id, request_id, idempotency_key, checks_json,
		proof_state_json, provider_state_json, trust_state, diagnostics_json,
		previous_hash, event_hash, created_at
		FROM verification_events ORDER BY sequence ASC`)
	if err != nil {
		return fmt.Errorf("read verification chain: %w", err)
	}
	defer rows.Close()
	previousHash := ""
	for rows.Next() {
		event, err := scanVerificationEvent(rows)
		if err != nil {
			return err
		}
		if event.PreviousHash != previousHash {
			return fmt.Errorf("verification chain link %d does not match", event.Sequence)
		}
		if event.EventHash != hashVerificationEvent(event) {
			return fmt.Errorf("verification event %q hash does not match", event.EventID)
		}
		previousHash = event.EventHash
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read verification chain: %w", err)
	}
	return nil
}

func scanVerificationEvent(row scanner) (*VerificationEvent, error) {
	var event VerificationEvent
	if err := row.Scan(&event.Sequence, &event.EventID, &event.ClaimID, &event.RequestID,
		&event.IdempotencyKey, &event.ChecksJSON, &event.ProofStateJSON,
		&event.ProviderStateJSON, &event.TrustState, &event.DiagnosticsJSON,
		&event.PreviousHash, &event.EventHash, &event.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scan verification event: %w", err)
	}
	return &event, nil
}

func hashVerificationEvent(event *VerificationEvent) string {
	canonical := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s",
		event.EventID, event.ClaimID, event.RequestID, event.IdempotencyKey,
		event.ChecksJSON, event.ProofStateJSON, event.ProviderStateJSON,
		event.TrustState, event.DiagnosticsJSON, event.PreviousHash,
		event.CreatedAt.UTC().Format(time.RFC3339Nano), "verification-event-v1")
	sum := sha256.Sum256([]byte(canonical))
	return "sha256:" + hex.EncodeToString(sum[:])
}
