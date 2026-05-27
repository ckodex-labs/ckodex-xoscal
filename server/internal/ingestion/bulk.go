package ingestion

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mchorfa/xoscal/server/internal/fetcher"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/reconciler"
	"gopkg.in/yaml.v3"
)

// BulkIngestor orchestrates fetch → parse → normalize → reconcile for multiple frameworks.
type BulkIngestor struct {
	fetcher    *fetcher.GitHubFetcher
	normalizer *Normalizer
	reconciler *reconciler.Reconciler
	kgStore    kg.Store
}

// NewBulkIngestor builds a bulk ingestor with the given components.
func NewBulkIngestor(
	ghFetcher *fetcher.GitHubFetcher,
	rec *reconciler.Reconciler,
	kgStore kg.Store,
) *BulkIngestor {
	return &BulkIngestor{
		fetcher:    ghFetcher,
		reconciler: rec,
		kgStore:    kgStore,
	}
}

// FrameworkResult carries the outcome for a single framework ingestion.
type FrameworkResult struct {
	RefID           string
	Filename        string
	NodesCreated    int
	AssessableCount int
	GroupsCreated   int
	Conflicts       int
	Error           error
}

// BulkResult aggregates per-framework ingestion results.
type BulkResult struct {
	Frameworks []FrameworkResult
	TotalNodes int
	TotalIGs   int
}

// Run executes bulk ingestion for the given filter.
func (b *BulkIngestor) Run(ctx context.Context, filter string) (*BulkResult, error) {
	files, err := b.fetcher.Fetch(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("fetch frameworks: %w", err)
	}

	var result BulkResult
	for _, file := range files {
		fr, err := b.ingestOne(ctx, file)
		if err != nil {
			fr.Error = err
		}
		result.Frameworks = append(result.Frameworks, fr)
		result.TotalNodes += fr.NodesCreated
		result.TotalIGs += fr.GroupsCreated
	}
	return &result, nil
}

func (b *BulkIngestor) ingestOne(ctx context.Context, file fetcher.FrameworkFile) (FrameworkResult, error) {
	res := FrameworkResult{RefID: file.RefID, Filename: file.Filename}

	parser := &CISOAssistantParser{}
	reqs, err := parser.Parse(ctx, file.Content)
	if err != nil {
		return res, fmt.Errorf("parse %s: %w", file.Filename, err)
	}

	// Extract library metadata for framework entity.
	var lib CISOLibrary
	_ = yaml.Unmarshal(file.Content, &lib)

	// Derive framework ref_id from parsed data if empty.
	frameworkRefID := file.RefID
	if frameworkRefID == "" && len(reqs) > 0 {
		frameworkRefID = reqs[0].Framework
	}
	if frameworkRefID == "" {
		frameworkRefID = lib.RefID
	}
	if frameworkRefID == "" {
		frameworkRefID = lib.Objects.Framework.RefID
	}

	norm := NewNormalizer(frameworkRefID)
	entities, err := norm.Normalize(reqs)
	if err != nil {
		return res, fmt.Errorf("normalize %s: %w", file.Filename, err)
	}

	for _, e := range entities {
		var req kg.Requirement
		_ = json.Unmarshal(e.Payload, &req)
		if req.Assessable {
			res.AssessableCount++
		}
	}
	res.NodesCreated = len(entities)

	conflicts, err := b.reconciler.Propose(ctx, entities)
	if err != nil {
		return res, fmt.Errorf("reconcile %s: %w", file.Filename, err)
	}
	res.Conflicts = len(conflicts)

	// Persist framework metadata as a KG entity.
	if frameworkRefID != "" {
		fwPayload := kg.Framework{
			URN:             fmt.Sprintf("urn:ciso:framework:%s", frameworkRefID),
			Type:            "reg:Framework",
			RefID:           frameworkRefID,
			Name:            lib.Name,
			Description:     lib.Description,
			Provider:        lib.Provider,
			Version:         lib.Version,
			PublicationDate: lib.PublicationDate,
			Locale:          lib.Locale,
			Packager:        lib.Packager,
		}
		if fwPayload.Name == "" {
			fwPayload.Name = file.RefID
		}
		_ = b.persistFramework(ctx, fwPayload)
	}

	return res, nil
}

func (b *BulkIngestor) persistFramework(ctx context.Context, fw kg.Framework) error {
	raw, err := json.Marshal(fw)
	if err != nil {
		return fmt.Errorf("marshal framework: %w", err)
	}
	entity := &kg.Entity{
		URN:     fw.URN,
		Type:    fw.Type,
		Payload: raw,
	}
	if err := b.kgStore.CreateEntity(ctx, entity); err != nil {
		// Entity may already exist; attempt update.
		if err := b.kgStore.UpdateEntity(ctx, entity); err != nil {
			return fmt.Errorf("persist framework %s: %w", fw.URN, err)
		}
	}
	return nil
}
