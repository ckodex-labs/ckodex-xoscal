package frameworks

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/mchorfa/xoscal/server/internal/kg"
)

// Sentinel errors for the framework registry.
var (
	ErrFrameworkNotFound = errors.New("framework not found in registry")
	ErrDuplicateID       = errors.New("duplicate framework id")
)

// Registry manages a thread-safe catalog of compliance frameworks.
type Registry struct {
	mu         sync.RWMutex
	frameworks map[string]Framework
	aliasMap   map[string]string
}

// NewRegistry initializes an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		frameworks: make(map[string]Framework),
		aliasMap:   make(map[string]string),
	}
}

// Register registers a framework with its canonical ID and aliases.
func (r *Registry) Register(f Framework) error {
	if f == nil {
		return errors.New("cannot register nil framework")
	}

	id := strings.ToLower(strings.TrimSpace(f.ID()))
	if id == "" {
		return errors.New("cannot register framework with empty id")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.frameworks[id]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateID, id)
	}

	r.frameworks[id] = f
	r.aliasMap[id] = id

	for _, alias := range f.Aliases() {
		normAlias := strings.ToLower(strings.TrimSpace(alias))
		if normAlias != "" {
			r.aliasMap[normAlias] = id
		}
	}

	return nil
}

// Get retrieves a framework by its canonical ID or registered alias.
func (r *Registry) Get(idOrAlias string) (Framework, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	norm := strings.ToLower(strings.TrimSpace(idOrAlias))
	canonID, ok := r.aliasMap[norm]
	if !ok {
		return nil, false
	}

	f, ok := r.frameworks[canonID]
	return f, ok
}

// Has reports whether a framework with the given ID or alias is registered.
func (r *Registry) Has(idOrAlias string) bool {
	_, ok := r.Get(idOrAlias)
	return ok
}

// List returns all registered frameworks sorted deterministically.
func (r *Registry) List() []Framework {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]Framework, 0, len(r.frameworks))
	for _, f := range r.frameworks {
		list = append(list, f)
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].Jurisdiction() != list[j].Jurisdiction() {
			return list[i].Jurisdiction() < list[j].Jurisdiction()
		}
		return list[i].ID() < list[j].ID()
	})

	return list
}

// ListInfo returns lightweight Info metadata for all registered frameworks.
func (r *Registry) ListInfo() []Info {
	fws := r.List()
	infos := make([]Info, len(fws))
	for i, f := range fws {
		infos[i] = InfoOf(f)
	}
	return infos
}

// FindByJurisdiction returns all frameworks registered under a given jurisdiction code.
func (r *Registry) FindByJurisdiction(jurisdiction string) []Framework {
	target := strings.ToLower(strings.TrimSpace(jurisdiction))
	all := r.List()
	var matches []Framework
	for _, f := range all {
		if strings.ToLower(f.Jurisdiction()) == target {
			matches = append(matches, f)
		}
	}
	return matches
}

// Export exports a specific framework by canonical ID or alias to outDir.
// Export exports all 7 layers of OSCAL 1.2.3 artifacts for a registered framework to outDir.
func (r *Registry) Export(outDir, idOrAlias string) error {
	f, ok := r.Get(idOrAlias)
	if !ok {
		return fmt.Errorf("%w: %s", ErrFrameworkNotFound, idOrAlias)
	}
	return ExportAllLayers(outDir, f)
}

// ExportAuthoritative exports only the canonical primary OSCAL artifact (catalog or profile)
// for a registered framework to outDir, preserving exact 1:1 compatibility with portal manifests.
func (r *Registry) ExportAuthoritative(outDir, idOrAlias string) error {
	f, ok := r.Get(idOrAlias)
	if !ok {
		return fmt.Errorf("%w: %s", ErrFrameworkNotFound, idOrAlias)
	}
	return ExportAuthoritative(outDir, f)
}

// ExportAll exports all registered frameworks to outDir.
func (r *Registry) ExportAll(outDir string) error {
	for _, f := range r.List() {
		if err := Export(outDir, f); err != nil {
			return fmt.Errorf("export framework %s: %w", f.ID(), err)
		}
	}
	return nil
}

// ExportJurisdiction exports all frameworks for a specific jurisdiction.
func (r *Registry) ExportJurisdiction(outDir, jurisdiction string) error {
	targets := r.FindByJurisdiction(jurisdiction)
	if len(targets) == 0 {
		return fmt.Errorf("no frameworks registered for jurisdiction %s", jurisdiction)
	}
	for _, f := range targets {
		if err := Export(outDir, f); err != nil {
			return fmt.Errorf("export framework %s: %w", f.ID(), err)
		}
	}
	return nil
}

// SeedKG seeds all registered frameworks into the Knowledge Graph store.
func (r *Registry) SeedKG(ctx context.Context, store kg.Store) error {
	for _, f := range r.List() {
		if err := SeedFramework(ctx, store, f); err != nil {
			return fmt.Errorf("seed framework %s: %w", f.ID(), err)
		}
	}
	return nil
}
