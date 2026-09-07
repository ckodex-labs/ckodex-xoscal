package transparency

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store persists claims and evidence for the transparency exchange.
type Store interface {
	CreateClaim(ctx context.Context, claim *Claim) error
	CreateImportBatch(ctx context.Context, records []*ImportRecord) error
	GetClaim(ctx context.Context, id string) (*Claim, error)
	ListClaims(ctx context.Context, subjectDigest, bomKind, relation, trustState string, validAfter time.Time, limit int, pageToken string) ([]*Claim, string, error)
	UpdateClaimTrustState(ctx context.Context, id, trustState string, proofStateJSON string) error
	RecordVerification(ctx context.Context, claimID, trustState, proofStateJSON string, event *VerificationEvent) (*VerificationEvent, error)
	ListVerificationEvents(ctx context.Context, claimID string) ([]*VerificationEvent, error)
	VerifyVerificationChain(ctx context.Context) error

	CreateEvidence(ctx context.Context, ev *Evidence) error
	GetEvidence(ctx context.Context, id string) (*Evidence, error)
	GetEvidenceByDigest(ctx context.Context, digest string) (*Evidence, error)
	GetEvidenceBlob(ctx context.Context, id string) ([]byte, error)

	RecordFetchEvent(ctx context.Context, ev FetchEvent) error
	ListFetchEvents(ctx context.Context, limit int) ([]FetchEvent, error)

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
	Blob                 []byte
}

// ImportRecord is the storage-level unit for an all-or-nothing intake batch.
type ImportRecord struct {
	Claim    *Claim
	Evidence []*Evidence
}

// VerificationEvent is an append-only audit record for a claim verification.
// EventHash chains records in insertion order so an operator can detect a
// modified or removed event when exporting the audit history.
type VerificationEvent struct {
	Sequence          int64
	EventID           string
	ClaimID           string
	RequestID         string
	IdempotencyKey    string
	ChecksJSON        string
	ProofStateJSON    string
	ProviderStateJSON string
	TrustState        string
	DiagnosticsJSON   string
	PreviousHash      string
	EventHash         string
	CreatedAt         time.Time
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
	extensions_json TEXT,
	blob BLOB
);
CREATE INDEX IF NOT EXISTS idx_evidence_digest ON kg_evidence(digest);
CREATE INDEX IF NOT EXISTS idx_evidence_bomkind ON kg_evidence(bom_kind);

CREATE TABLE IF NOT EXISTS kg_claim_evidence (
	claim_id TEXT NOT NULL REFERENCES kg_claims(id) ON DELETE CASCADE,
	evidence_id TEXT NOT NULL REFERENCES kg_evidence(id) ON DELETE CASCADE,
	PRIMARY KEY (claim_id, evidence_id)
);

CREATE TABLE IF NOT EXISTS verification_events (
	sequence INTEGER PRIMARY KEY AUTOINCREMENT,
	event_id TEXT NOT NULL UNIQUE,
	claim_id TEXT NOT NULL REFERENCES kg_claims(id) ON DELETE RESTRICT,
	request_id TEXT NOT NULL,
	idempotency_key TEXT NOT NULL UNIQUE,
	checks_json TEXT NOT NULL,
	proof_state_json TEXT NOT NULL,
	provider_state_json TEXT NOT NULL,
	trust_state TEXT NOT NULL,
	diagnostics_json TEXT NOT NULL,
	previous_hash TEXT NOT NULL,
	event_hash TEXT NOT NULL,
	created_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_verification_events_claim ON verification_events(claim_id, sequence);
CREATE TRIGGER IF NOT EXISTS verification_events_no_update
BEFORE UPDATE ON verification_events
BEGIN
	SELECT RAISE(ABORT, 'verification_events are append-only');
END;
CREATE TRIGGER IF NOT EXISTS verification_events_no_delete
BEFORE DELETE ON verification_events
BEGIN
	SELECT RAISE(ABORT, 'verification_events are append-only');
END;
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}
	// Existing beta databases predate inline evidence content. Keep the migration
	// explicit so digest verification cannot silently operate against a missing
	// content column after an upgrade.
	if _, err := s.db.Exec(`ALTER TABLE kg_evidence ADD COLUMN blob BLOB`); err != nil &&
		!strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return fmt.Errorf("add evidence blob column: %w", err)
	}
	if err := s.migrateFetchEvents(); err != nil {
		return fmt.Errorf("migrate fetch events: %w", err)
	}
	return nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }
