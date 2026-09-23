package scaffold

import (
	"github.com/google/uuid"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
)

// XoscalNamespace is the DNS-based namespace UUID for xoscal entities.
var XoscalNamespace = uuid.MustParse("d3b07384-d113-494a-81a2-2b6389710f27")

// UUIDFromURN creates a deterministic RFC-4122 UUIDv5 from a semantic URN.
func UUIDFromURN(urn string) *commonv1.UUID {
	u := uuid.NewSHA1(XoscalNamespace, []byte(urn))
	return &commonv1.UUID{Value: u.String()}
}

// StringUUIDFromURN returns a string UUIDv5 from a semantic URN.
func StringUUIDFromURN(urn string) string {
	return uuid.NewSHA1(XoscalNamespace, []byte(urn)).String()
}
