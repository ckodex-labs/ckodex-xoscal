package frameworks

var defaultRegistry = NewRegistry()

// DefaultRegistry returns the global default Framework registry.
func DefaultRegistry() *Registry {
	return defaultRegistry
}

// Register registers a framework with the default registry.
func Register(f Framework) error {
	return defaultRegistry.Register(f)
}

// Get retrieves a framework from the default registry.
func Get(idOrAlias string) (Framework, bool) {
	return defaultRegistry.Get(idOrAlias)
}

// Has reports whether a framework with idOrAlias exists in the default registry.
func Has(idOrAlias string) bool {
	return defaultRegistry.Has(idOrAlias)
}

// List returns all frameworks in the default registry.
func List() []Framework {
	return defaultRegistry.List()
}

// ListInfo returns lightweight Info metadata for all frameworks in the default registry.
func ListInfo() []Info {
	return defaultRegistry.ListInfo()
}

// FindByJurisdiction returns frameworks from the default registry matching the jurisdiction.
func FindByJurisdiction(jurisdiction string) []Framework {
	return defaultRegistry.FindByJurisdiction(jurisdiction)
}

// ExportFramework exports a specific framework from the default registry.
func ExportFramework(outDir, idOrAlias string) error {
	return defaultRegistry.Export(outDir, idOrAlias)
}

// ExportAll exports all frameworks from the default registry.
func ExportAll(outDir string) error {
	return defaultRegistry.ExportAll(outDir)
}

// ExportJurisdiction exports all frameworks for a jurisdiction from the default registry.
func ExportJurisdiction(outDir, jurisdiction string) error {
	return defaultRegistry.ExportJurisdiction(outDir, jurisdiction)
}
