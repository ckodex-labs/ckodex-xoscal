package transparency

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store persists claims and evidence for the transparency exchange.
type Store interface {
	CreateClaim(ctx context.Context, claim *Claim) error
	GetClaim(ctx context.Context, id string) (*Claim, error)
	ListClaims(ctx context.Context, subjectDigest, bomKind, relation, trustState string, validAfter time.Time, limit int) ([]*Claim, error)
	UpdateClaimTrustState(ctx context.Context, id, trustState string, proofStateJSON string) error

	CreateEvidence(ctx context.Context, ev *Evidence) error
	GetEvidence(ctx context.Context, id string) (*Evidence, error)
	GetEvidenceByDigest(ctx context.Context, digest string) (*Evidence, error)

	Close() error
}

// Claim is the database representation of a transparency claim.
type Claim struct {
	ID             string
	Type           string
	SubjectJSON    string
	PredicateJSON  string
	ObjectJSON     string
	IssuerJSON     string
	BomKind        string
	ValidFrom      time.Time
	ValidTo        *time.Time
	ObservedTime   time.Time
	SourceRefsJSON string
	ProofRefsJSON  string
	PolicyRefsJSON string
	ExtensionsJSON string
	TrustState     string
	ProofStateJSON string
	CreatedAt      time.Time
}

// Evidence is the database representation of a transparency evidence object.
type Evidence struct {
	ID                   string
	MediaType            string
	BomKind              string
	Digest               string
	SizeBytes            int64
	StorageJSON          string
	ProducedByJSON       string
	SubjectRefsJSON      string
	PredicateType        string
	SpecJSON             string
	CreatedAt            time.Time
	ValidFrom            time.Time
	ValidTo              *time.Time
	SupersedesJSON       string
	IntegrityMethodsJSON string
	Classification       string
	ExtensionsJSON       string
}

// SQLiteStore implements Store with SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens and migrates the transparency tables.
func NewSQLiteStore(dsn string) (Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS kg_claims (
	id TEXT PRIMARY KEY,
	claim_type TEXT NOT NULL,
	subject_json TEXT NOT NULL,
	predicate_json TEXT NOT NULL,
	object_json TEXT,
	issuer_json TEXT NOT NULL,
	bom_kind TEXT,
	valid_from DATETIME NOT NULL,
	valid_to DATETIME,
	observed_time DATETIME NOT NULL,
	source_refs_json TEXT NOT NULL,
	proof_refs_json TEXT,
	policy_refs_json TEXT,
	extensions_json TEXT,
	trust_state TEXT NOT NULL DEFAULT 'candidate',
	proof_state_json TEXT,
	created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_claims_type ON kg_claims(claim_type);
CREATE INDEX IF NOT EXISTS idx_claims_bomkind ON kg_claims(bom_kind);
CREATE INDEX IF NOT EXISTS idx_claims_trust ON kg_claims(trust_state);
CREATE INDEX IF NOT EXISTS idx_claims_valid ON kg_claims(valid_from, valid_to);

CREATE TABLE IF NOT EXISTS kg_evidence (
	id TEXT PRIMARY KEY,
	media_type TEXT NOT NULL,
	bom_kind TEXT NOT NULL,
	digest TEXT NOT NULL UNIQUE,
	size_bytes INTEGER,
	storage_json TEXT NOT NULL,
	produced_by_json TEXT,
	subject_refs_json TEXT,
	predicate_type TEXT,
	spec_json TEXT,
	created_at DATETIME,
	valid_from DATETIME NOT NULL,
	valid_to DATETIME,
	supersedes_json TEXT,
	integrity_methods_json TEXT,
	classification TEXT,
	extensions_json TEXT
);
CREATE INDEX IF NOT EXISTS idx_evidence_digest ON kg_evidence(digest);
CREATE INDEX IF NOT EXISTS idx_evidence_bomkind ON kg_evidence(bom_kind);

CREATE TABLE IF NOT EXISTS kg_claim_evidence (
	claim_id TEXT NOT NULL REFERENCES kg_claims(id) ON DELETE CASCADE,
	evidence_id TEXT NOT NULL REFERENCES kg_evidence(id) ON DELETE CASCADE,
	PRIMARY KEY (claim_id, evidence_id)
);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}
	return nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

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

func (s *SQLiteStore) GetClaim(ctx context.Context, id string) (*Claim, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, claim_type, subject_json, predicate_json, object_json, issuer_json, bom_kind,
		valid_from, valid_to, observed_time, source_refs_json, proof_refs_json, policy_refs_json, extensions_json,
		trust_state, proof_state_json, created_at FROM kg_claims WHERE id = ?`, id)
	return scanClaim(row)
}

func (s *SQLiteStore) ListClaims(ctx context.Context, subjectDigest, bomKind, relation, trustState string, validAfter time.Time, limit int) ([]*Claim, error) {
	query := `SELECT id, claim_type, subject_json, predicate_json, object_json, issuer_json, bom_kind,
		valid_from, valid_to, observed_time, source_refs_json, proof_refs_json, policy_refs_json, extensions_json,
		trust_state, proof_state_json, created_at FROM kg_claims WHERE 1=1`
	args := []interface{}{}
	if bomKind != "" {
		query += ` AND bom_kind = ?`
		args = append(args, bomKind)
	}
	if trustState != "" {
		query += ` AND trust_state = ?`
		args = append(args, trustState)
	}
	if !validAfter.IsZero() {
		query += ` AND valid_from >= ?`
		args = append(args, validAfter)
	}
	query += ` ORDER BY created_at DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list claims: %w", err)
	}
	defer rows.Close()
	var out []*Claim
	for rows.Next() {
		c, err := scanClaim(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) UpdateClaimTrustState(ctx context.Context, id, trustState string, proofStateJSON string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE kg_claims SET trust_state = ?, proof_state_json = ? WHERE id = ?`,
		trustState, proofStateJSON, id)
	return err
}

func (s *SQLiteStore) CreateEvidence(ctx context.Context, ev *Evidence) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO kg_evidence(id, media_type, bom_kind, digest, size_bytes, storage_json, produced_by_json,
		subject_refs_json, predicate_type, spec_json, created_at, valid_from, valid_to, supersedes_json,
		integrity_methods_json, classification, extensions_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ev.ID, ev.MediaType, ev.BomKind, ev.Digest, ev.SizeBytes, ev.StorageJSON, ev.ProducedByJSON,
		ev.SubjectRefsJSON, ev.PredicateType, ev.SpecJSON, ev.CreatedAt, ev.ValidFrom, ev.ValidTo,
		ev.SupersedesJSON, ev.IntegrityMethodsJSON, ev.Classification, ev.ExtensionsJSON,
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
		integrity_methods_json, classification, extensions_json FROM kg_evidence WHERE id = ?`, id)
	return scanEvidence(row)
}

func (s *SQLiteStore) GetEvidenceByDigest(ctx context.Context, digest string) (*Evidence, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, media_type, bom_kind, digest, size_bytes, storage_json, produced_by_json,
		subject_refs_json, predicate_type, spec_json, created_at, valid_from, valid_to, supersedes_json,
		integrity_methods_json, classification, extensions_json FROM kg_evidence WHERE digest = ?`, digest)
	return scanEvidence(row)
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
	err := row.Scan(&ev.ID, &ev.MediaType, &ev.BomKind, &ev.Digest, &sizeBytes, &ev.StorageJSON, &producedBy,
		&subjectRefs, &ev.PredicateType, &spec, &createdAt, &ev.ValidFrom, &validTo, &supersedes,
		&integrity, &ev.Classification, &extensions)
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
	return &ev, nil
}
