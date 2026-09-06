package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"github.com/mchorfa/xoscal/server/internal/embedding"
	"github.com/mchorfa/xoscal/server/internal/ingestion"
)

func (s *GovernanceServer) IngestRequirements(ctx context.Context, req *servicesv1.IngestRequirementsRequest) (*servicesv1.IngestRequirementsResponse, error) {
	if len(req.RawData) == 0 {
		return nil, status.Error(codes.InvalidArgument, "raw_data is required")
	}
	parser := selectParser(req.Format)
	if parser == nil {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported format: %s", req.Format)
	}
	norm := ingestion.NewNormalizer(req.Framework)
	pipe := ingestion.NewPipeline(parser, norm, s.rec)
	res, err := pipe.Run(ctx, req.RawData)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ingestion pipeline: %v", err)
	}

	// Index ingested entities for semantic search.
	for _, e := range res.Entities {
		if err := s.vectorStore.Index(ctx, embedding.Document{
			UUID:      e.URN,
			ModelType: e.Type,
			Framework: req.Framework,
			Content:   string(e.Payload),
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "index ingested entity %s: %v", e.URN, err)
		}
	}

	return &servicesv1.IngestRequirementsResponse{
		Created:   boundedInt32(len(res.Entities)),
		Conflicts: boundedInt32(len(res.Conflicts)),
	}, nil
}

func selectParser(format string) ingestion.Parser {
	switch format {
	case "", "eu-ai-act-json", "json":
		return &ingestion.EUAIActParser{}
	case "ciso-assistant-yaml", "yaml":
		return &ingestion.CISOAssistantParser{}
	default:
		return nil
	}
}

// --- Semantic Search ---

func (s *GovernanceServer) SemanticSearch(ctx context.Context, req *servicesv1.SemanticSearchRequest) (*servicesv1.SemanticSearchResponse, error) {
	results, err := s.vectorStore.Search(ctx, req.Query, req.Framework, int(req.TopK))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "semantic search: %v", err)
	}
	var out []*servicesv1.SemanticSearchResult
	for _, r := range results {
		out = append(out, &servicesv1.SemanticSearchResult{
			EntityUrn:  r.UUID,
			EntityType: r.ModelType,
			Score:      float64(r.Score),
		})
	}
	return &servicesv1.SemanticSearchResponse{Results: out}, nil
}

// --- OSCAL Generation ---
