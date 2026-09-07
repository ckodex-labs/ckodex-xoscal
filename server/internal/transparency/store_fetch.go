package transparency

import (
	"context"
	"database/sql"
	"time"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FetchEvent is one append-only record of an external evidence fetch attempt.
// It records policy outcomes as well as successes so denials are auditable.
type FetchEvent struct {
	URL        string
	Digest     string
	SizeBytes  int64
	Outcome    string // fetched | policy_denied | fetch_failed
	Detail     string
	FetchedAt  time.Time
	DurationMS int64
}

func (e FetchEvent) toProto() *servicesv1.EvidenceFetchAudit {
	return &servicesv1.EvidenceFetchAudit{
		Url:        e.URL,
		Digest:     e.Digest,
		SizeBytes:  e.SizeBytes,
		Outcome:    e.Outcome,
		Detail:     e.Detail,
		FetchedAt:  timestamppb.New(e.FetchedAt.UTC()),
		DurationMs: e.DurationMS,
	}
}

const fetchEventsSchema = `
CREATE TABLE IF NOT EXISTS kg_fetch_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	url TEXT NOT NULL,
	digest TEXT NOT NULL DEFAULT '',
	size_bytes INTEGER NOT NULL DEFAULT 0,
	outcome TEXT NOT NULL,
	detail TEXT NOT NULL DEFAULT '',
	fetched_at DATETIME NOT NULL DEFAULT (datetime('now')),
	duration_ms INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_fetch_events_outcome ON kg_fetch_events(outcome);
CREATE INDEX IF NOT EXISTS idx_fetch_events_time ON kg_fetch_events(fetched_at);
`

func (s *SQLiteStore) migrateFetchEvents() error {
	if _, err := s.db.Exec(fetchEventsSchema); err != nil {
		return err
	}
	return nil
}

// RecordFetchEvent appends one fetch audit record. Fetch events are
// append-only: there is no update or delete path.
func (s *SQLiteStore) RecordFetchEvent(ctx context.Context, ev FetchEvent) error {
	if ev.FetchedAt.IsZero() {
		ev.FetchedAt = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
INSERT INTO kg_fetch_events (url, digest, size_bytes, outcome, detail, fetched_at, duration_ms)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ev.URL, ev.Digest, ev.SizeBytes, ev.Outcome, ev.Detail, ev.FetchedAt.UTC(), ev.DurationMS)
	if err != nil {
		return err
	}
	if rows, err := res.RowsAffected(); err == nil && rows != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// ListFetchEvents returns the most recent fetch events, newest first.
func (s *SQLiteStore) ListFetchEvents(ctx context.Context, limit int) ([]FetchEvent, error) {
	if limit <= 0 || limit > maxListPageSize {
		limit = maxListPageSize
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT url, digest, size_bytes, outcome, detail, fetched_at, duration_ms
FROM kg_fetch_events ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]FetchEvent, 0, limit)
	for rows.Next() {
		var ev FetchEvent
		var fetchedAt time.Time
		if err := rows.Scan(&ev.URL, &ev.Digest, &ev.SizeBytes, &ev.Outcome, &ev.Detail, &fetchedAt, &ev.DurationMS); err != nil {
			return nil, err
		}
		ev.FetchedAt = fetchedAt.UTC()
		events = append(events, ev)
	}
	return events, rows.Err()
}
