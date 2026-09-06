package oscal

import (
	"strings"

	"github.com/google/uuid"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

// Generator converts KG snapshots into OSCAL protobuf artifacts.
type Generator struct {
	store kg.Store
}

func NewGenerator(store kg.Store) *Generator {
	return &Generator{store: store}
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
			b.WriteRune(r)
		default:
			// Replace spaces, parentheses, plus signs, etc. with underscore.
			b.WriteByte('_')
		}
	}
	result := b.String()
	// Ensure it doesn't start with a digit (already handled above, but
	// double-check in case the first char was replaced).
	if len(result) > 0 {
		first := result[0]
		if first >= '0' && first <= '9' {
			result = "_" + result
		}
	}
	return result
}

var uuidNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // DNS namespace

// GenerateCatalog builds an OSCAL Catalog from a snapshot of requirements.
// Supports hierarchical groups derived from parent_urn / depth relationships.
