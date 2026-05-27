package ingestion

import (
	"context"
	"fmt"

	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/reconciler"
)

// Pipeline orchestrates parse → normalize → reconcile → store.
type Pipeline struct {
	parser     Parser
	normalizer *Normalizer
	reconciler *reconciler.Reconciler
}

// NewPipeline builds an ingestion pipeline.
func NewPipeline(parser Parser, normalizer *Normalizer, rec *reconciler.Reconciler) *Pipeline {
	return &Pipeline{
		parser:     parser,
		normalizer: normalizer,
		reconciler: rec,
	}
}

// Run executes the full pipeline: parse raw → normalize → propose to reconciler.
func (p *Pipeline) Run(ctx context.Context, raw []byte) (*PipelineResult, error) {
	reqs, err := p.parser.Parse(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	entities, err := p.normalizer.Normalize(reqs)
	if err != nil {
		return nil, fmt.Errorf("normalize: %w", err)
	}
	conflicts, err := p.reconciler.Propose(ctx, entities)
	if err != nil {
		return nil, fmt.Errorf("propose: %w", err)
	}
	return &PipelineResult{
		Requirements: reqs,
		Entities:     entities,
		Conflicts:    conflicts,
	}, nil
}

// PipelineResult carries the output of a pipeline run.
type PipelineResult struct {
	Requirements []Requirement
	Entities     []*kg.Entity
	Conflicts    []*reconciler.Conflict
}
