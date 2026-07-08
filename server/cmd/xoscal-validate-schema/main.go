// Command xoscal-validate-schema validates OSCAL JSON artifacts against
// the official NIST OSCAL 1.1.2 JSON schemas.
//
// Usage:
//
//	xoscal-validate-schema -file <path> [-kind <kind>]
//
// If -kind is not specified, the root key in the JSON is auto-detected.
// Exit codes: 0 = valid, 1 = error, 3 = validation failure.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
)

func main() {
	filePath := flag.String("file", "", "path to OSCAL JSON artifact to validate")
	kindFlag := flag.String("kind", "", "artifact kind (catalog, profile, ssp, component-definition, assessment-plan, assessment-results, poam)")
	flag.Parse()

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "error: -file is required")
		flag.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
		os.Exit(1)
	}

	v, err := schemavalidate.NewValidator()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading schemas: %v\n", err)
		os.Exit(1)
	}

	kind, err := resolveKind(*kindFlag, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := v.Validate(data, kind); err != nil {
		fmt.Fprintf(os.Stderr, "INVALID: %s failed OSCAL %s schema validation\n", kind.Name, schemavalidate.SchemaVersion)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}

	fmt.Printf("VALID: %s passes OSCAL %s schema validation\n", kind.Name, schemavalidate.SchemaVersion)
}

// resolveKind determines the artifact kind from the flag or by auto-detecting
// the root key in the JSON.
func resolveKind(kindFlag string, data []byte) (schemavalidate.ArtifactKind, error) {
	if kindFlag != "" {
		return kindFromName(kindFlag)
	}

	// Auto-detect from root key
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return schemavalidate.ArtifactKind{}, fmt.Errorf("unmarshal JSON: %w", err)
	}

	for _, k := range []schemavalidate.ArtifactKind{
		schemavalidate.KindCatalog,
		schemavalidate.KindProfile,
		schemavalidate.KindSSP,
		schemavalidate.KindComponentDefinition,
		schemavalidate.KindAssessmentPlan,
		schemavalidate.KindAssessmentResults,
		schemavalidate.KindPOAM,
	} {
		if _, ok := doc[k.RootKey]; ok {
			return k, nil
		}
	}

	return schemavalidate.ArtifactKind{}, fmt.Errorf("could not auto-detect artifact kind; specify -kind explicitly")
}

func kindFromName(name string) (schemavalidate.ArtifactKind, error) {
	switch strings.ToLower(name) {
	case "catalog":
		return schemavalidate.KindCatalog, nil
	case "profile":
		return schemavalidate.KindProfile, nil
	case "ssp", "system-security-plan":
		return schemavalidate.KindSSP, nil
	case "component-definition", "component":
		return schemavalidate.KindComponentDefinition, nil
	case "assessment-plan", "ap":
		return schemavalidate.KindAssessmentPlan, nil
	case "assessment-results", "ar":
		return schemavalidate.KindAssessmentResults, nil
	case "poam", "plan-of-action-and-milestones":
		return schemavalidate.KindPOAM, nil
	default:
		return schemavalidate.ArtifactKind{}, fmt.Errorf("unknown kind: %s", name)
	}
}
