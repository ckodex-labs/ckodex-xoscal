package service

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"github.com/mchorfa/xoscal/server/internal/fetcher"
	"github.com/mchorfa/xoscal/server/internal/ingestion"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

func (s *GovernanceServer) BulkIngestFrameworks(ctx context.Context, req *servicesv1.BulkIngestFrameworksRequest) (*servicesv1.BulkIngestFrameworksResponse, error) {
	owner := req.GithubOwner
	if owner == "" {
		owner = "intuitem"
	}
	repo := req.GithubRepo
	if repo == "" {
		repo = "ciso-assistant-community"
	}
	path := req.GithubPath
	if path == "" {
		path = "backend/library/libraries"
	}

	fetcher := fetcher.NewGitHubFetcher(owner, repo, path)
	bulk := ingestion.NewBulkIngestor(fetcher, s.rec, s.kgStore)
	res, err := bulk.Run(ctx, req.Filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "bulk ingest: %v", err)
	}

	var summaries []*servicesv1.FrameworkSummary
	var totalConflicts int
	for _, fr := range res.Frameworks {
		summaries = append(summaries, &servicesv1.FrameworkSummary{
			RefId:           fr.RefID,
			Name:            fr.RefID,
			NodeCount:       boundedInt32(fr.NodesCreated),
			AssessableCount: boundedInt32(fr.AssessableCount),
		})
		totalConflicts += fr.Conflicts
	}

	return &servicesv1.BulkIngestFrameworksResponse{
		Frameworks:     summaries,
		TotalNodes:     boundedInt32(res.TotalNodes),
		TotalConflicts: boundedInt32(totalConflicts),
	}, nil
}

func (s *GovernanceServer) ListFrameworks(ctx context.Context, req *servicesv1.ListFrameworksRequest) (*servicesv1.ListFrameworksResponse, error) {
	entities, err := s.kgStore.ListEntities(ctx, "reg:Framework", kg.EntityStatusActive)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list frameworks: %v", err)
	}
	var out []*servicesv1.FrameworkSummary
	for _, e := range entities {
		var fw kg.Framework
		if err := json.Unmarshal(e.Payload, &fw); err != nil {
			continue
		}
		if req.JurisdictionFilter != "" && fw.Locale != req.JurisdictionFilter {
			continue
		}
		out = append(out, &servicesv1.FrameworkSummary{
			RefId:   fw.RefID,
			Name:    fw.Name,
			Version: fw.Version,
		})
	}
	return &servicesv1.ListFrameworksResponse{Frameworks: out}, nil
}

func (s *GovernanceServer) GetFramework(ctx context.Context, req *servicesv1.GetFrameworkRequest) (*servicesv1.GetFrameworkResponse, error) {
	entities, err := s.kgStore.ListEntities(ctx, "reg:Framework", kg.EntityStatusActive)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get framework: %v", err)
	}
	for _, e := range entities {
		var fw kg.Framework
		if err := json.Unmarshal(e.Payload, &fw); err != nil {
			continue
		}
		if fw.RefID == req.RefId {
			return &servicesv1.GetFrameworkResponse{
				Framework: &servicesv1.FrameworkSummary{
					RefId:   fw.RefID,
					Name:    fw.Name,
					Version: fw.Version,
				},
			}, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "framework %s not found", req.RefId)
}

// --- Cross-Framework Mappings ---

func (s *GovernanceServer) GenerateCrossFrameworkMappings(ctx context.Context, req *servicesv1.GenerateCrossFrameworkMappingsRequest) (*servicesv1.GenerateCrossFrameworkMappingsResponse, error) {
	maps, err := s.generator.GenerateCrossFrameworkMappings(ctx, req.SnapshotName, req.SourceFramework, req.TargetFramework)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate cross-framework mappings: %v", err)
	}
	return &servicesv1.GenerateCrossFrameworkMappingsResponse{Maps: maps}, nil
}

// --- GitOps ---

func (s *GovernanceServer) PublishRelease(ctx context.Context, req *servicesv1.PublishReleaseRequest) (*servicesv1.PublishReleaseResponse, error) {
	if s.gitops == nil {
		return nil, status.Error(codes.Internal, "gitops client not initialized")
	}
	releaseURL, uploadURL, err := s.gitops.PublishRelease(
		ctx,
		req.RepoOwner,
		req.RepoName,
		req.TagName,
		req.TargetCommitish,
		req.ReleaseName,
		req.Body,
		req.AssetPaths,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "publish release: %v", err)
	}
	return &servicesv1.PublishReleaseResponse{
		ReleaseUrl: releaseURL,
		UploadUrl:  uploadURL,
	}, nil
}
