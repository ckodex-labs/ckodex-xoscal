package service

import (
	"context"
	"encoding/json"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
	"github.com/mchorfa/xoscal/server/internal/embedding"
	"github.com/mchorfa/xoscal/server/internal/fetcher"
	"github.com/mchorfa/xoscal/server/internal/gitops"
	"github.com/mchorfa/xoscal/server/internal/ingestion"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/oscal"
	"github.com/mchorfa/xoscal/server/internal/reconciler"
)

// GovernanceServer implements servicesv1.GovernanceServiceServer.
type GovernanceServer struct {
	servicesv1.UnimplementedGovernanceServiceServer
	kgStore     kg.Store
	rec         *reconciler.Reconciler
	vectorStore embedding.VectorStore
	generator   *oscal.Generator
	gitops      *gitops.Client
}

// NewGovernanceServer creates a new governance gRPC server backed by the
// given advanced components.
func NewGovernanceServer(
	kgStore kg.Store,
	rec *reconciler.Reconciler,
	vectorStore embedding.VectorStore,
) *GovernanceServer {
	return &GovernanceServer{
		kgStore:     kgStore,
		rec:         rec,
		vectorStore: vectorStore,
		generator:   oscal.NewGenerator(kgStore),
		gitops:      gitops.NewClient(),
	}
}

// --- Entity CRUD ---

func (s *GovernanceServer) CreateEntity(ctx context.Context, req *servicesv1.CreateEntityRequest) (*servicesv1.CreateEntityResponse, error) {
	if req.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}
	e := toKGEntity(req.Entity)
	if err := s.kgStore.CreateEntity(ctx, e); err != nil {
		return nil, status.Errorf(codes.Internal, "create entity: %v", err)
	}
	return &servicesv1.CreateEntityResponse{Entity: toProtoEntity(e)}, nil
}

func (s *GovernanceServer) GetEntity(ctx context.Context, req *servicesv1.GetEntityRequest) (*servicesv1.GetEntityResponse, error) {
	e, err := s.kgStore.GetEntity(ctx, req.Urn)
	if err != nil {
		return nil, notFoundErr("entity", err)
	}
	return &servicesv1.GetEntityResponse{Entity: toProtoEntity(e)}, nil
}

func (s *GovernanceServer) UpdateEntity(ctx context.Context, req *servicesv1.UpdateEntityRequest) (*servicesv1.UpdateEntityResponse, error) {
	if req.Entity == nil {
		return nil, status.Error(codes.InvalidArgument, "entity is required")
	}
	e := toKGEntity(req.Entity)
	if err := s.kgStore.UpdateEntity(ctx, e); err != nil {
		return nil, status.Errorf(codes.Internal, "update entity: %v", err)
	}
	return &servicesv1.UpdateEntityResponse{Entity: toProtoEntity(e)}, nil
}

func (s *GovernanceServer) ListEntities(ctx context.Context, req *servicesv1.ListEntitiesRequest) (*servicesv1.ListEntitiesResponse, error) {
	var statusFilter kg.EntityStatus
	if req.StatusFilter != "" {
		statusFilter = kg.EntityStatus(req.StatusFilter)
	}
	entities, err := s.kgStore.ListEntities(ctx, req.TypeFilter, statusFilter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list entities: %v", err)
	}
	var out []*servicesv1.Entity
	for _, e := range entities {
		out = append(out, toProtoEntity(e))
	}
	return &servicesv1.ListEntitiesResponse{Entities: out}, nil
}

// --- Snapshots & Releases ---

func (s *GovernanceServer) CreateSnapshot(ctx context.Context, req *servicesv1.CreateSnapshotRequest) (*servicesv1.CreateSnapshotResponse, error) {
	ss, err := s.kgStore.CreateSnapshot(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create snapshot: %v", err)
	}
	return &servicesv1.CreateSnapshotResponse{
		Name:        ss.Name,
		EntityCount: boundedInt32(ss.EntityCount),
	}, nil
}

func (s *GovernanceServer) GetSnapshot(ctx context.Context, req *servicesv1.GetSnapshotRequest) (*servicesv1.GetSnapshotResponse, error) {
	entities, err := s.kgStore.GetSnapshot(ctx, req.Name)
	if err != nil {
		return nil, notFoundErr("snapshot", err)
	}
	var out []*servicesv1.Entity
	for _, e := range entities {
		out = append(out, toProtoEntity(e))
	}
	return &servicesv1.GetSnapshotResponse{Entities: out}, nil
}

func (s *GovernanceServer) ListSnapshots(ctx context.Context, req *servicesv1.ListSnapshotsRequest) (*servicesv1.ListSnapshotsResponse, error) {
	snapshots, err := s.kgStore.ListSnapshots(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list snapshots: %v", err)
	}
	var names []string
	for _, ss := range snapshots {
		names = append(names, ss.Name)
	}
	return &servicesv1.ListSnapshotsResponse{Names: names}, nil
}

func (s *GovernanceServer) CreateRelease(ctx context.Context, req *servicesv1.CreateReleaseRequest) (*servicesv1.CreateReleaseResponse, error) {
	r, err := s.kgStore.CreateRelease(ctx, req.Name, req.SnapshotName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create release: %v", err)
	}
	return &servicesv1.CreateReleaseResponse{
		Name:         r.Name,
		SnapshotName: r.Snapshot,
	}, nil
}

func (s *GovernanceServer) ListReleases(ctx context.Context, req *servicesv1.ListReleasesRequest) (*servicesv1.ListReleasesResponse, error) {
	releases, err := s.kgStore.ListReleases(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list releases: %v", err)
	}
	var names []string
	for _, r := range releases {
		names = append(names, r.Name)
	}
	return &servicesv1.ListReleasesResponse{Names: names}, nil
}

// --- Ingestion ---

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

func (s *GovernanceServer) GenerateCatalog(ctx context.Context, req *servicesv1.GenerateCatalogRequest) (*servicesv1.GenerateCatalogResponse, error) {
	catalog, err := s.generator.GenerateCatalog(ctx, req.SnapshotName, req.Framework)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate catalog: %v", err)
	}
	return &servicesv1.GenerateCatalogResponse{Catalog: catalog}, nil
}

func (s *GovernanceServer) GenerateProfile(ctx context.Context, req *servicesv1.GenerateProfileRequest) (*servicesv1.GenerateProfileResponse, error) {
	profile, err := s.generator.GenerateProfile(ctx, req.SnapshotName, req.Framework, req.SelectedControls)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate profile: %v", err)
	}
	return &servicesv1.GenerateProfileResponse{Profile: profile}, nil
}

func (s *GovernanceServer) GenerateMappings(ctx context.Context, req *servicesv1.GenerateMappingsRequest) (*servicesv1.GenerateMappingsResponse, error) {
	maps, err := s.generator.GenerateMappings(ctx, req.SnapshotName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate mappings: %v", err)
	}
	return &servicesv1.GenerateMappingsResponse{Maps: maps}, nil
}

func (s *GovernanceServer) GenerateSSP(ctx context.Context, req *servicesv1.GenerateSSPRequest) (*servicesv1.GenerateSSPResponse, error) {
	ssp, err := s.generator.GenerateSSP(ctx, req.SnapshotName, req.Framework)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate ssp: %v", err)
	}
	return &servicesv1.GenerateSSPResponse{Ssp: ssp}, nil
}

func (s *GovernanceServer) GenerateComponentDefinition(ctx context.Context, req *servicesv1.GenerateComponentDefinitionRequest) (*servicesv1.GenerateComponentDefinitionResponse, error) {
	compDef, err := s.generator.GenerateComponentDefinition(ctx, req.SnapshotName, req.Framework)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate component definition: %v", err)
	}
	return &servicesv1.GenerateComponentDefinitionResponse{ComponentDefinition: compDef}, nil
}

func (s *GovernanceServer) GenerateAssessmentPlan(ctx context.Context, req *servicesv1.GenerateAssessmentPlanRequest) (*servicesv1.GenerateAssessmentPlanResponse, error) {
	ap, err := s.generator.GenerateAssessmentPlan(ctx, req.SnapshotName, req.Framework)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate assessment plan: %v", err)
	}
	return &servicesv1.GenerateAssessmentPlanResponse{AssessmentPlan: ap}, nil
}

func (s *GovernanceServer) GeneratePOAM(ctx context.Context, req *servicesv1.GeneratePOAMRequest) (*servicesv1.GeneratePOAMResponse, error) {
	poam, err := s.generator.GeneratePOAM(ctx, req.SnapshotName, req.Framework)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate poam: %v", err)
	}
	return &servicesv1.GeneratePOAMResponse{Poam: poam}, nil
}

func (s *GovernanceServer) GenerateAssessmentResults(ctx context.Context, req *servicesv1.GenerateAssessmentResultsRequest) (*servicesv1.GenerateAssessmentResultsResponse, error) {
	ar, err := s.generator.GenerateAssessmentResults(ctx, req.SnapshotName, req.Framework)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate assessment results: %v", err)
	}
	return &servicesv1.GenerateAssessmentResultsResponse{AssessmentResults: ar}, nil
}

// --- Framework Ingestion ---

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

func (s *GovernanceServer) ProposeMappingUpdate(ctx context.Context, req *servicesv1.ProposeMappingUpdateRequest) (*servicesv1.ProposeMappingUpdateResponse, error) {
	if s.gitops == nil {
		return nil, status.Error(codes.Internal, "gitops client not initialized")
	}
	prURL, prNumber, err := s.gitops.ProposeMappingUpdate(
		ctx,
		req.RepoOwner,
		req.RepoName,
		req.BaseBranch,
		req.HeadBranch,
		req.Title,
		req.Body,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "propose mapping update: %v", err)
	}
	return &servicesv1.ProposeMappingUpdateResponse{
		PullRequestUrl:    prURL,
		PullRequestNumber: boundedInt32(prNumber),
	}, nil
}

// --- Reconciler ---

func (s *GovernanceServer) Propose(ctx context.Context, req *servicesv1.ProposeRequest) (*servicesv1.ProposeResponse, error) {
	var proposals []*kg.Entity
	for _, pe := range req.Proposals {
		proposals = append(proposals, toKGEntity(pe))
	}
	conflicts, err := s.rec.Propose(ctx, proposals)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "propose: %v", err)
	}
	// Also detect batch-level conflicts (e.g. MappingGap).
	batchConflicts, err := s.rec.EvaluateBatch(ctx, proposals)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "evaluate batch: %v", err)
	}
	conflicts = append(conflicts, batchConflicts...)
	return &servicesv1.ProposeResponse{
		Accepted:  boundedInt32(len(proposals) - len(conflicts)),
		Conflicts: boundedInt32(len(conflicts)),
	}, nil
}

func (s *GovernanceServer) ListConflicts(ctx context.Context, req *servicesv1.ListConflictsRequest) (*servicesv1.ListConflictsResponse, error) {
	conflicts, err := s.rec.ListConflicts(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list conflicts: %v", err)
	}
	var out []*servicesv1.Entity
	for _, c := range conflicts {
		raw, err := json.Marshal(c)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "marshal conflict %s: %v", c.ID, err)
		}
		out = append(out, &servicesv1.Entity{
			Urn:     c.ID,
			Type:    string(c.Type),
			Payload: string(raw),
		})
	}
	_ = req
	return &servicesv1.ListConflictsResponse{Conflicts: out}, nil
}

func (s *GovernanceServer) ResolveConflict(ctx context.Context, req *servicesv1.ResolveConflictRequest) (*servicesv1.ResolveConflictResponse, error) {
	if err := s.rec.ResolveConflict(ctx, req.ConflictUrn, []byte(req.Resolution)); err != nil {
		return nil, status.Errorf(codes.Internal, "resolve conflict: %v", err)
	}
	return &servicesv1.ResolveConflictResponse{Success: true}, nil
}

// --- Helpers ---

func toKGEntity(p *servicesv1.Entity) *kg.Entity {
	version := 1
	if p.Version != "" {
		if v, err := strconv.Atoi(p.Version); err == nil {
			version = v
		}
	}
	return &kg.Entity{
		URN:     p.Urn,
		Type:    p.Type,
		Version: version,
		Status:  kg.EntityStatus(p.Status),
		Payload: []byte(p.Payload),
	}
}

func toProtoEntity(e *kg.Entity) *servicesv1.Entity {
	return &servicesv1.Entity{
		Urn:     e.URN,
		Type:    e.Type,
		Version: strconv.Itoa(e.Version),
		Status:  string(e.Status),
		Payload: string(e.Payload),
	}
}

const maxInt32Value = 1<<31 - 1

// boundedInt32 converts a count or identifier only after constraining it to
// the range representable by the protobuf int32 field.
func boundedInt32(value int) int32 {
	if value > maxInt32Value {
		return maxInt32Value
	}
	if value < -maxInt32Value-1 {
		return -maxInt32Value - 1
	}
	// #nosec G115 -- the bounds above prove that value fits in int32.
	return int32(value)
}
