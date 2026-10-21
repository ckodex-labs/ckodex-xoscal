package canadianframeworks

import (
	"context"

	"github.com/mchorfa/xoscal/server/internal/frameworks"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

// SeedKG seeds all authoritative Canadian cybersecurity frameworks and their
// individual control requirements directly into a Knowledge Graph store.
// The operation is idempotent: existing entities are updated in place.
func SeedKG(ctx context.Context, store kg.Store) error {
	for _, f := range frameworks.DefaultRegistry().FindByJurisdiction("ca") {
		if err := frameworks.SeedFramework(ctx, store, f); err != nil {
			return err
		}
	}
	return nil
}
