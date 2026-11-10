package oscal

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultOSCALTimestamp = "2026-01-01T00:00:00Z"

func deterministicProtoTimestamp() *timestamppb.Timestamp {
	return timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
}

// Generator converts KG snapshots into OSCAL protobuf artifacts.
type Generator struct {
	store kg.Store
}

func NewGenerator(store kg.Store) *Generator {
	return &Generator{store: store}
}

// parsedSnapshot holds a single-pass decoded representation of a KG snapshot
// filtered for a target framework, avoiding repeated GetSnapshot calls and
// redundant JSON unmarshals across multi-artifact generation.
type parsedSnapshot struct {
	fwMeta         *kg.Framework
	requirements   []kg.Requirement
	assessableReqs []kg.Requirement
	mappings       []kg.Mapping
}

// payloadMayMatchFramework performs an allocation-free ASCII case-insensitive
// substring check on raw JSON bytes to skip unmarshaling requirements belonging
// to unrelated frameworks when scanning large multi-framework snapshots.
func payloadMayMatchFramework(payload []byte, lowerFramework string) bool {
	if lowerFramework == "" {
		return true
	}
	for i := 0; i < len(lowerFramework); i++ {
		c := lowerFramework[i]
		if c >= 0x80 || c == '\\' || c == '"' {
			return true
		}
	}
	if bytes.Contains(payload, []byte(`\u`)) || bytes.Contains(payload, []byte(`\U`)) {
		return true
	}
	n := len(lowerFramework)
	if len(payload) < n {
		return false
	}
	first := lowerFramework[0]
	limit := len(payload) - n
	for i := 0; i <= limit; i++ {
		b := payload[i]
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		if b != first {
			continue
		}
		match := true
		for j := 1; j < n; j++ {
			bj := payload[i+j]
			if bj >= 'A' && bj <= 'Z' {
				bj += 'a' - 'A'
			}
			if bj != lowerFramework[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func canonicalFrameworkAlias(framework string) string {
	switch strings.ToLower(strings.TrimSpace(framework)) {
	case "nist_csf_2", "nist-csf-2", "csf-2.0", "nist-csf":
		return "nist-csf-2.0"
	case "iso42001", "iso_42001", "iso-42001":
		return "iso-iec-42001"
	case "sp800-53", "nist-800-53", "800-53":
		return "nist-sp-800-53-rev5"
	default:
		return ""
	}
}

func (g *Generator) loadParsedSnapshot(ctx context.Context, snapshotName, framework string, includeMappings bool) (*parsedSnapshot, error) {
	lowerFw := strings.ToLower(framework)
	altFw := canonicalFrameworkAlias(framework)
	snap := &parsedSnapshot{}
	matchesPayload := func(payload []byte) bool {
		if payloadMayMatchFramework(payload, lowerFw) {
			return true
		}
		return altFw != "" && payloadMayMatchFramework(payload, altFw)
	}
	matchesID := func(id string) bool {
		return strings.EqualFold(id, framework) || (altFw != "" && strings.EqualFold(id, altFw))
	}
	consumeEntity := func(entityType string, payload []byte) {
		switch entityType {
		case "reg:Framework":
			if !matchesPayload(payload) {
				return
			}
			var fw kg.Framework
			if err := json.Unmarshal(payload, &fw); err == nil && matchesID(fw.RefID) {
				fwCopy := fw
				snap.fwMeta = &fwCopy
			}
		case "reg:Requirement":
			if !matchesPayload(payload) {
				return
			}
			var req kg.Requirement
			if err := json.Unmarshal(payload, &req); err != nil {
				return
			}
			if !matchesID(req.Framework) {
				return
			}
			snap.requirements = append(snap.requirements, req)
			if req.Assessable {
				snap.assessableReqs = append(snap.assessableReqs, req)
			}
		case "reg:Mapping":
			if !includeMappings {
				return
			}
			var m kg.Mapping
			if err := json.Unmarshal(payload, &m); err != nil {
				return
			}
			snap.mappings = append(snap.mappings, m)
		}
	}

	if dbProv, ok := g.store.(interface{ DB() *sql.DB }); ok && dbProv.DB() != nil {
		var (
			rows *sql.Rows
			err  error
		)
		if lowerFw != "" {
			pat1 := "urn:xoscal:requirement:" + lowerFw + ":%"
			pat2 := pat1
			if altFw != "" {
				pat2 = "urn:xoscal:requirement:" + altFw + ":%"
			}
			rows, err = dbProv.DB().QueryContext(ctx,
				`SELECT e.entity_type, e.payload
				 FROM kg_snapshot_entities se
				 JOIN kg_entities e ON se.entity_urn = e.urn AND se.version = e.version
				 WHERE se.snapshot_name = ?
				   AND (se.entity_urn NOT LIKE 'urn:xoscal:requirement:%' OR se.entity_urn LIKE ? OR se.entity_urn LIKE ?)
				 ORDER BY se.rowid`,
				snapshotName, pat1, pat2,
			)
		} else {
			rows, err = dbProv.DB().QueryContext(ctx,
				`SELECT e.entity_type, e.payload
				 FROM kg_snapshot_entities se
				 JOIN kg_entities e ON se.entity_urn = e.urn AND se.version = e.version
				 WHERE se.snapshot_name = ?
				 ORDER BY se.rowid`,
				snapshotName,
			)
		}
		if err != nil {
			return nil, fmt.Errorf("get snapshot: %w", err)
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var entityType string
			var payload []byte
			if err := rows.Scan(&entityType, &payload); err != nil {
				return nil, fmt.Errorf("scan snapshot entity: %w", err)
			}
			consumeEntity(entityType, payload)
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("iterate snapshot entities: %w", err)
		}
		return snap, nil
	}

	entities, err := g.store.GetSnapshot(ctx, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}
	for _, e := range entities {
		if e == nil {
			continue
		}
		consumeEntity(e.Type, e.Payload)
	}
	return snap, nil
}

// newUUID generates a deterministic v5 UUID from the given namespace and name.
func newUUID(ns uuid.UUID, name string) *commonv1.UUID {
	u := uuid.NewSHA1(ns, []byte(name))
	return &commonv1.UUID{Value: u.String()}
}

// sanitizeToken ensures a value conforms to the OSCAL Token datatype
// (NCName: must start with a letter or underscore, followed by letters,
// digits, dots, hyphens, or underscores). Invalid characters are replaced
// with underscores; values starting with a digit are prefixed with "_".
func sanitizeToken(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	for i, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			if i == 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
		case r == '.' || r == '-' || r == '_':
			if i == 0 && (r == '.' || r == '-') {
				b.WriteByte('_')
			}
			b.WriteRune(r)
		default:
			// Replace spaces, parentheses, plus signs, etc. with underscore.
			b.WriteByte('_')
		}
	}
	result := b.String()
	// Ensure it doesn't start with a digit, dot, or hyphen.
	if len(result) > 0 {
		first := result[0]
		if (first >= '0' && first <= '9') || first == '.' || first == '-' {
			result = "_" + result
		}
	}
	return result
}

var uuidNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // DNS namespace

// GenerateCatalog builds an OSCAL Catalog from a snapshot of requirements.
// Supports hierarchical groups derived from parent_urn / depth relationships.
