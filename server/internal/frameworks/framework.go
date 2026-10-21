package frameworks

import (
	"context"

	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	profilev1 "github.com/mchorfa/xoscal/proto/oscal/profile/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

// Framework is the canonical interface for any compliance framework or baseline.
type Framework interface {
	ID() string
	Aliases() []string
	Title() string
	Version() string
	Jurisdiction() string
	Authority() string
	ControlCount() int
	HasProfile() bool
	CatalogJSON() ([]byte, error)
	ProfileJSON() ([]byte, error)
}

// ProtoCatalogProvider is an optional interface for frameworks that provide
// a native protobuf OSCAL Catalog model.
type ProtoCatalogProvider interface {
	CatalogProto() (*catalogv1.Catalog, error)
}

// ProtoProfileProvider is an optional interface for frameworks that provide
// a native protobuf OSCAL Profile model.
type ProtoProfileProvider interface {
	ProfileProto() (*profilev1.Profile, error)
}

// KGSeeder is an optional interface for frameworks with custom knowledge
// graph seeding implementations.
type KGSeeder interface {
	SeedKG(ctx context.Context, store kg.Store) error
}

// Info provides lightweight metadata about a framework.
type Info struct {
	ID           string   `json:"id"`
	Aliases      []string `json:"aliases,omitempty"`
	Title        string   `json:"title"`
	Version      string   `json:"version"`
	Jurisdiction string   `json:"jurisdiction"`
	Authority    string   `json:"authority"`
	ControlCount int      `json:"control_count"`
	HasProfile   bool     `json:"has_profile"`
}

// InfoOf extracts lightweight Info metadata from any Framework.
func InfoOf(f Framework) Info {
	return Info{
		ID:           f.ID(),
		Aliases:      f.Aliases(),
		Title:        f.Title(),
		Version:      f.Version(),
		Jurisdiction: f.Jurisdiction(),
		Authority:    f.Authority(),
		ControlCount: f.ControlCount(),
		HasProfile:   f.HasProfile(),
	}
}
