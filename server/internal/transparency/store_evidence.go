package transparency

import (
	"context"
	"database/sql"
	"fmt"
)

func (s *SQLiteStore) CreateEvidence(ctx context.Context, ev *Evidence) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO kg_evidence(id, media_type, bom_kind, digest, size_bytes, storage_json, produced_by_json,
		subject_refs_json, predicate_type, spec_json, created_at, valid_from, valid_to, supersedes_json,
		integrity_methods_json, classification, extensions_json, blob)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ev.ID, ev.MediaType, ev.BomKind, ev.Digest, ev.SizeBytes, ev.StorageJSON, ev.ProducedByJSON,
		ev.SubjectRefsJSON, ev.PredicateType, ev.SpecJSON, ev.CreatedAt, ev.ValidFrom, ev.ValidTo,
		ev.SupersedesJSON, ev.IntegrityMethodsJSON, ev.Classification, ev.ExtensionsJSON, ev.Blob,
	)
	if err != nil {
		return fmt.Errorf("insert evidence: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetEvidence(ctx context.Context, id string) (*Evidence, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, media_type, bom_kind, digest, size_bytes, storage_json, produced_by_json,
		subject_refs_json, predicate_type, spec_json, created_at, valid_from, valid_to, supersedes_json,
		integrity_methods_json, classification, extensions_json, blob FROM kg_evidence WHERE id = ?`, id)
	return scanEvidence(row)
}

func (s *SQLiteStore) GetEvidenceByDigest(ctx context.Context, digest string) (*Evidence, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, media_type, bom_kind, digest, size_bytes, storage_json, produced_by_json,
		subject_refs_json, predicate_type, spec_json, created_at, valid_from, valid_to, supersedes_json,
		integrity_methods_json, classification, extensions_json, blob FROM kg_evidence WHERE digest = ?`, digest)
	return scanEvidence(row)
}

func (s *SQLiteStore) GetEvidenceBlob(ctx context.Context, id string) ([]byte, error) {
	var blob []byte
	if err := s.db.QueryRowContext(ctx, `SELECT blob FROM kg_evidence WHERE id = ?`, id).Scan(&blob); err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("get evidence blob: %w", err)
	}
	return blob, nil
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanClaim(row scanner) (*Claim, error) {
	var c Claim
	var validTo sql.NullTime
	var proofState, extensions, objectJSON, proofRefs, policyRefs sql.NullString
	err := row.Scan(&c.ID, &c.Type, &c.SubjectJSON, &c.PredicateJSON, &objectJSON,
		&c.IssuerJSON, &c.BomKind, &c.ValidFrom, &validTo, &c.ObservedTime,
		&c.SourceRefsJSON, &proofRefs, &policyRefs, &extensions,
		&c.TrustState, &proofState, &c.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scan claim: %w", err)
	}
	if validTo.Valid {
		c.ValidTo = &validTo.Time
	}
	if proofState.Valid {
		c.ProofStateJSON = proofState.String
	}
	if extensions.Valid {
		c.ExtensionsJSON = extensions.String
	}
	if objectJSON.Valid {
		c.ObjectJSON = objectJSON.String
	}
	if proofRefs.Valid {
		c.ProofRefsJSON = proofRefs.String
	}
	if policyRefs.Valid {
		c.PolicyRefsJSON = policyRefs.String
	}
	return &c, nil
}

func scanEvidence(row scanner) (*Evidence, error) {
	var ev Evidence
	var validTo sql.NullTime
	var createdAt sql.NullTime
	var sizeBytes sql.NullInt64
	var producedBy, subjectRefs, spec, supersedes, integrity, extensions sql.NullString
	var blob []byte
	err := row.Scan(&ev.ID, &ev.MediaType, &ev.BomKind, &ev.Digest, &sizeBytes, &ev.StorageJSON, &producedBy,
		&subjectRefs, &ev.PredicateType, &spec, &createdAt, &ev.ValidFrom, &validTo, &supersedes,
		&integrity, &ev.Classification, &extensions, &blob)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scan evidence: %w", err)
	}
	if sizeBytes.Valid {
		ev.SizeBytes = sizeBytes.Int64
	}
	if createdAt.Valid {
		ev.CreatedAt = createdAt.Time
	}
	if validTo.Valid {
		ev.ValidTo = &validTo.Time
	}
	if producedBy.Valid {
		ev.ProducedByJSON = producedBy.String
	}
	if subjectRefs.Valid {
		ev.SubjectRefsJSON = subjectRefs.String
	}
	if spec.Valid {
		ev.SpecJSON = spec.String
	}
	if supersedes.Valid {
		ev.SupersedesJSON = supersedes.String
	}
	if integrity.Valid {
		ev.IntegrityMethodsJSON = integrity.String
	}
	if extensions.Valid {
		ev.ExtensionsJSON = extensions.String
	}
	ev.Blob = blob
	return &ev, nil
}
