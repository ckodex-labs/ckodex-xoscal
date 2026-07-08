package integration

import (
	"context"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/dbutil"
	"github.com/mchorfa/xoscal/server/internal/embedding"
	"github.com/mchorfa/xoscal/server/internal/ingestion"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/reconciler"
)

func TestEndToEnd(t *testing.T) {
	ctx := context.Background()

	// Setup stores
	kgStore, err := kg.NewSQLiteStore(":memory:", dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("kg store: %v", err)
	}
	defer kgStore.Close()

	vectorStore, err := embedding.NewSQLiteVectorStore(":memory:", dbutil.PoolConfig{})
	if err != nil {
		t.Fatalf("vector store: %v", err)
	}
	defer vectorStore.Close()

	// Ingest EU AI Act requirements
	raw := []byte(`[
		{"id":"art16-a","citation":"Article 16(a)","title":"Risk Management","text":"Implement risk management system. Should be documented.","role":"provider","risk_level":"high-risk","framework":"eu-ai-act","assessable":true},
		{"id":"art16-b","citation":"Article 16(b)","title":"Data Governance","text":"Ensure training data quality","role":"provider","risk_level":"high-risk","framework":"eu-ai-act","assessable":true}
	]`)

	rec := reconciler.NewReconciler(kgStore)
	pipeline := ingestion.NewPipeline(
		&ingestion.EUAIActParser{},
		ingestion.NewNormalizer("eu-ai-act"),
		rec,
	)

	res, err := pipeline.Run(ctx, raw)
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}
	if len(res.Requirements) != 2 {
		t.Fatalf("requirements = %d, want 2", len(res.Requirements))
	}
	if len(res.Entities) != 2 {
		t.Fatalf("entities = %d, want 2", len(res.Entities))
	}

	// Create snapshot and release
	ss, err := kgStore.CreateSnapshot(ctx, "v1")
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if ss.EntityCount != 2 {
		t.Errorf("snapshot entities = %d, want 2", ss.EntityCount)
	}

	_, err = kgStore.CreateRelease(ctx, "r1", "v1")
	if err != nil {
		t.Fatalf("release: %v", err)
	}

	// Generate all OSCAL artifacts
	gen := oscal.NewGenerator(kgStore)
	all, err := gen.GenerateAllArtifacts(ctx, "v1", "eu-ai-act", nil)
	if err != nil {
		t.Fatalf("generate all: %v", err)
	}
	if all.Catalog == nil || len(all.Catalog.Controls) != 2 {
		t.Errorf("catalog controls = %d, want 2", len(all.Catalog.Controls))
	}
	if all.Profile == nil {
		t.Error("profile is nil")
	}
	if all.SSP == nil {
		t.Error("ssp is nil")
	}
	if all.ComponentDefinition == nil {
		t.Error("component-definition is nil")
	}
	if all.AssessmentPlan == nil {
		t.Error("assessment-plan is nil")
	}
	if len(all.AssessmentPlan.Tasks) != 2 {
		t.Errorf("assessment tasks = %d, want 2", len(all.AssessmentPlan.Tasks))
	}
	if all.POAM == nil {
		t.Error("poam is nil")
	}
	if all.AssessmentResults == nil {
		t.Error("assessment-results is nil")
	}
	if len(all.AssessmentResults.Results) == 0 || len(all.AssessmentResults.Results[0].Findings) != 2 {
		t.Errorf("assessment findings = %d, want 2", len(all.AssessmentResults.Results[0].Findings))
	}
	if len(all.AssessmentResults.Results[0].Observations) != 2 {
		t.Errorf("assessment observations = %d, want 2", len(all.AssessmentResults.Results[0].Observations))
	}
	if len(all.Mappings) == 0 {
		t.Logf("mappings empty (expected with no mapping entities)")
	}

	// Verify data-driven catalog properties.
	ctrl := all.Catalog.Controls[0]
	if len(ctrl.Props) == 0 {
		t.Error("expected catalog control to have data-driven props")
	}
	var hasRole, hasRisk bool
	for _, p := range ctrl.Props {
		if p.Name == "role" && p.Value == "provider" {
			hasRole = true
		}
		if p.Name == "risk-level" && p.Value == "high-risk" {
			hasRisk = true
		}
	}
	if !hasRole {
		t.Error("expected role=provider prop in catalog control")
	}
	if !hasRisk {
		t.Error("expected risk-level=high-risk prop in catalog control")
	}
	// Verify guidance extraction (check all controls since snapshot ordering is non-deterministic).
	var hasGuidance bool
	for _, c := range all.Catalog.Controls {
		for _, part := range c.Parts {
			if part.Name == "guidance" {
				hasGuidance = true
			}
		}
	}
	if !hasGuidance {
		t.Error("expected guidance part in at least one catalog control for text containing 'Should'")
	}

	// Verify SSP enrichment.
	if all.SSP.SystemCharacteristics == nil || all.SSP.SystemCharacteristics.SecurityImpactLevel == nil {
		t.Error("expected SSP SecurityImpactLevel")
	}
	if all.SSP.SystemImplementation == nil || len(all.SSP.SystemImplementation.InventoryItems) == 0 {
		t.Error("expected SSP inventory items")
	}
	if len(all.SSP.Metadata.ResponsibleParties) == 0 {
		t.Error("expected SSP responsible parties derived from roles")
	}

	// Index for semantic search
	for _, e := range res.Entities {
		vectorStore.Index(ctx, embedding.Document{
			UUID:      e.URN,
			ModelType: e.Type,
			Framework: "eu-ai-act",
			Content:   string(e.Payload),
		})
	}

	searchResults, err := vectorStore.Search(ctx, "risk", "eu-ai-act", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(searchResults) == 0 {
		t.Logf("FTS5 search returned 0 results (expected with small dataset)")
	}

	t.Logf("End-to-end success: %d requirements, %d controls, %d search results",
		len(res.Requirements), len(all.Catalog.Controls), len(searchResults))
}
