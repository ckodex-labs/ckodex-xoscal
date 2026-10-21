package frameworks

import (
	"context"
	"errors"
	"fmt"

	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	profilev1 "github.com/mchorfa/xoscal/proto/oscal/profile/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

// ErrProfileNotSupported is returned when requesting a profile from a framework without one.
var ErrProfileNotSupported = errors.New("framework does not define a companion profile")

// ErrCatalogNotSupported is returned when a framework has no catalog definition.
var ErrCatalogNotSupported = errors.New("framework does not define a catalog")

// ErrNoCustomSeeder is returned when a framework does not implement custom seeding.
var ErrNoCustomSeeder = errors.New("framework does not define a custom seeder")

// ErrProtoNotSupported is returned when a framework does not provide native protobuf models.
var ErrProtoNotSupported = errors.New("framework does not provide native protobuf models")

// Config holds configuration parameters for instantiating a BaseFramework.
type Config struct {
	ID             string
	Aliases        []string
	Title          string
	Version        string
	Jurisdiction   string
	Authority      string
	ControlCount   int
	HasProfile     bool
	CatalogFn      func() ([]byte, error)
	ProfileFn      func() ([]byte, error)
	CatalogProtoFn func() (*catalogv1.Catalog, error)
	ProfileProtoFn func() (*profilev1.Profile, error)
	SeedKGFn       func(ctx context.Context, store kg.Store) error
}

// BaseFramework is a flexible implementation of Framework.
type BaseFramework struct {
	cfg Config
}

// New creates a BaseFramework from the provided configuration.
func New(cfg Config) *BaseFramework {
	return &BaseFramework{cfg: cfg}
}

func (f *BaseFramework) ID() string {
	return f.cfg.ID
}

func (f *BaseFramework) Aliases() []string {
	return f.cfg.Aliases
}

func (f *BaseFramework) Title() string {
	return f.cfg.Title
}

func (f *BaseFramework) Version() string {
	return f.cfg.Version
}

func (f *BaseFramework) Jurisdiction() string {
	return f.cfg.Jurisdiction
}

func (f *BaseFramework) Authority() string {
	return f.cfg.Authority
}

func (f *BaseFramework) ControlCount() int {
	return f.cfg.ControlCount
}

func (f *BaseFramework) HasProfile() bool {
	return f.cfg.HasProfile
}

func (f *BaseFramework) CatalogJSON() ([]byte, error) {
	if f.cfg.CatalogFn != nil {
		return f.cfg.CatalogFn()
	}
	return nil, fmt.Errorf("%w: %s", ErrCatalogNotSupported, f.cfg.ID)
}

func (f *BaseFramework) ProfileJSON() ([]byte, error) {
	if !f.cfg.HasProfile {
		return nil, fmt.Errorf("%w: %s", ErrProfileNotSupported, f.cfg.ID)
	}
	if f.cfg.ProfileFn != nil {
		return f.cfg.ProfileFn()
	}
	return nil, fmt.Errorf("%w: %s", ErrProfileNotSupported, f.cfg.ID)
}

func (f *BaseFramework) CatalogProto() (*catalogv1.Catalog, error) {
	if f.cfg.CatalogProtoFn != nil {
		return f.cfg.CatalogProtoFn()
	}
	return nil, ErrProtoNotSupported
}

func (f *BaseFramework) ProfileProto() (*profilev1.Profile, error) {
	if f.cfg.ProfileProtoFn != nil {
		return f.cfg.ProfileProtoFn()
	}
	return nil, ErrProtoNotSupported
}

func (f *BaseFramework) SeedKG(ctx context.Context, store kg.Store) error {
	if f.cfg.SeedKGFn != nil {
		return f.cfg.SeedKGFn(ctx, store)
	}
	return ErrNoCustomSeeder
}
