package service

import (
	"context"
	"database/sql"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"

	"github.com/mchorfa/xoscal/server/internal/store"
)

// OscalServer implements servicesv1.OscalServiceServer.
type OscalServer struct {
	servicesv1.UnimplementedOscalServiceServer
	store store.Store
}

// NewOscalServer creates a new gRPC server backed by the given store.
func NewOscalServer(s store.Store) *OscalServer {
	return &OscalServer{store: s}
}

func notFoundErr(name string, err error) error {
	if err == sql.ErrNoRows {
		return status.Errorf(codes.NotFound, "%s not found", name)
	}
	return status.Errorf(codes.Internal, "%s: %v", name, err)
}

// ---- Catalog ----

func (s *OscalServer) GetCatalog(ctx context.Context, req *servicesv1.GetCatalogRequest) (*servicesv1.GetCatalogResponse, error) {
	c, err := s.store.GetCatalog(ctx, req.Uuid.Value)
	if err != nil {
		return nil, notFoundErr("catalog", err)
	}
	return &servicesv1.GetCatalogResponse{Catalog: c}, nil
}

func (s *OscalServer) ListCatalogs(ctx context.Context, req *servicesv1.ListCatalogsRequest) (*servicesv1.ListCatalogsResponse, error) {
	catalogs, nextToken, err := s.store.ListCatalogs(ctx, req.Filter, int(req.PageSize), req.PageToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list catalogs: %v", err)
	}
	return &servicesv1.ListCatalogsResponse{Catalogs: catalogs, NextPageToken: nextToken}, nil
}

func (s *OscalServer) CreateCatalog(ctx context.Context, req *servicesv1.CreateCatalogRequest) (*servicesv1.CreateCatalogResponse, error) {
	if req.Catalog == nil {
		return nil, status.Error(codes.InvalidArgument, "catalog is required")
	}
	if err := s.store.CreateCatalog(ctx, req.Catalog); err != nil {
		return nil, status.Errorf(codes.Internal, "create catalog: %v", err)
	}
	return &servicesv1.CreateCatalogResponse{Catalog: req.Catalog}, nil
}

func (s *OscalServer) UpdateCatalog(ctx context.Context, req *servicesv1.UpdateCatalogRequest) (*servicesv1.UpdateCatalogResponse, error) {
	if req.Catalog == nil {
		return nil, status.Error(codes.InvalidArgument, "catalog is required")
	}
	if err := s.store.UpdateCatalog(ctx, req.Uuid.Value, req.Catalog); err != nil {
		return nil, notFoundErr("catalog", err)
	}
	return &servicesv1.UpdateCatalogResponse{Catalog: req.Catalog}, nil
}

func (s *OscalServer) DeleteCatalog(ctx context.Context, req *servicesv1.DeleteCatalogRequest) (*servicesv1.DeleteCatalogResponse, error) {
	if err := s.store.DeleteCatalog(ctx, req.Uuid.Value); err != nil {
		return nil, notFoundErr("catalog", err)
	}
	return &servicesv1.DeleteCatalogResponse{Success: true}, nil
}

// ---- Profile ----

func (s *OscalServer) GetProfile(ctx context.Context, req *servicesv1.GetProfileRequest) (*servicesv1.GetProfileResponse, error) {
	p, err := s.store.GetProfile(ctx, req.Uuid.Value)
	if err != nil {
		return nil, notFoundErr("profile", err)
	}
	return &servicesv1.GetProfileResponse{Profile: p}, nil
}

func (s *OscalServer) ListProfiles(ctx context.Context, req *servicesv1.ListProfilesRequest) (*servicesv1.ListProfilesResponse, error) {
	profiles, nextToken, err := s.store.ListProfiles(ctx, req.Filter, int(req.PageSize), req.PageToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list profiles: %v", err)
	}
	return &servicesv1.ListProfilesResponse{Profiles: profiles, NextPageToken: nextToken}, nil
}

func (s *OscalServer) CreateProfile(ctx context.Context, req *servicesv1.CreateProfileRequest) (*servicesv1.CreateProfileResponse, error) {
	if req.Profile == nil {
		return nil, status.Error(codes.InvalidArgument, "profile is required")
	}
	if err := s.store.CreateProfile(ctx, req.Profile); err != nil {
		return nil, status.Errorf(codes.Internal, "create profile: %v", err)
	}
	return &servicesv1.CreateProfileResponse{Profile: req.Profile}, nil
}

func (s *OscalServer) UpdateProfile(ctx context.Context, req *servicesv1.UpdateProfileRequest) (*servicesv1.UpdateProfileResponse, error) {
	if req.Profile == nil {
		return nil, status.Error(codes.InvalidArgument, "profile is required")
	}
	if err := s.store.UpdateProfile(ctx, req.Uuid.Value, req.Profile); err != nil {
		return nil, notFoundErr("profile", err)
	}
	return &servicesv1.UpdateProfileResponse{Profile: req.Profile}, nil
}

func (s *OscalServer) DeleteProfile(ctx context.Context, req *servicesv1.DeleteProfileRequest) (*servicesv1.DeleteProfileResponse, error) {
	if err := s.store.DeleteProfile(ctx, req.Uuid.Value); err != nil {
		return nil, notFoundErr("profile", err)
	}
	return &servicesv1.DeleteProfileResponse{Success: true}, nil
}

// ---- Component Definition ----

func (s *OscalServer) GetComponentDefinition(ctx context.Context, req *servicesv1.GetComponentDefinitionRequest) (*servicesv1.GetComponentDefinitionResponse, error) {
	cd, err := s.store.GetComponentDefinition(ctx, req.Uuid.Value)
	if err != nil {
		return nil, notFoundErr("component definition", err)
	}
	return &servicesv1.GetComponentDefinitionResponse{ComponentDefinition: cd}, nil
}

func (s *OscalServer) ListComponentDefinitions(ctx context.Context, req *servicesv1.ListComponentDefinitionsRequest) (*servicesv1.ListComponentDefinitionsResponse, error) {
	cds, nextToken, err := s.store.ListComponentDefinitions(ctx, req.Filter, int(req.PageSize), req.PageToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list component definitions: %v", err)
	}
	return &servicesv1.ListComponentDefinitionsResponse{ComponentDefinitions: cds, NextPageToken: nextToken}, nil
}

func (s *OscalServer) CreateComponentDefinition(ctx context.Context, req *servicesv1.CreateComponentDefinitionRequest) (*servicesv1.CreateComponentDefinitionResponse, error) {
	if req.ComponentDefinition == nil {
		return nil, status.Error(codes.InvalidArgument, "component_definition is required")
	}
	if err := s.store.CreateComponentDefinition(ctx, req.ComponentDefinition); err != nil {
		return nil, status.Errorf(codes.Internal, "create component definition: %v", err)
	}
	return &servicesv1.CreateComponentDefinitionResponse{ComponentDefinition: req.ComponentDefinition}, nil
}

func (s *OscalServer) UpdateComponentDefinition(ctx context.Context, req *servicesv1.UpdateComponentDefinitionRequest) (*servicesv1.UpdateComponentDefinitionResponse, error) {
	if req.ComponentDefinition == nil {
		return nil, status.Error(codes.InvalidArgument, "component_definition is required")
	}
	if err := s.store.UpdateComponentDefinition(ctx, req.Uuid.Value, req.ComponentDefinition); err != nil {
		return nil, notFoundErr("component definition", err)
	}
	return &servicesv1.UpdateComponentDefinitionResponse{ComponentDefinition: req.ComponentDefinition}, nil
}

func (s *OscalServer) DeleteComponentDefinition(ctx context.Context, req *servicesv1.DeleteComponentDefinitionRequest) (*servicesv1.DeleteComponentDefinitionResponse, error) {
	if err := s.store.DeleteComponentDefinition(ctx, req.Uuid.Value); err != nil {
		return nil, notFoundErr("component definition", err)
	}
	return &servicesv1.DeleteComponentDefinitionResponse{Success: true}, nil
}

// ---- SSP ----

func (s *OscalServer) GetSsp(ctx context.Context, req *servicesv1.GetSspRequest) (*servicesv1.GetSspResponse, error) {
	ssp, err := s.store.GetSsp(ctx, req.Uuid.Value)
	if err != nil {
		return nil, notFoundErr("ssp", err)
	}
	return &servicesv1.GetSspResponse{Ssp: ssp}, nil
}

func (s *OscalServer) ListSsps(ctx context.Context, req *servicesv1.ListSspsRequest) (*servicesv1.ListSspsResponse, error) {
	ssps, nextToken, err := s.store.ListSsps(ctx, req.Filter, int(req.PageSize), req.PageToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list ssps: %v", err)
	}
	return &servicesv1.ListSspsResponse{Ssps: ssps, NextPageToken: nextToken}, nil
}

func (s *OscalServer) CreateSsp(ctx context.Context, req *servicesv1.CreateSspRequest) (*servicesv1.CreateSspResponse, error) {
	if req.Ssp == nil {
		return nil, status.Error(codes.InvalidArgument, "ssp is required")
	}
	if err := s.store.CreateSsp(ctx, req.Ssp); err != nil {
		return nil, status.Errorf(codes.Internal, "create ssp: %v", err)
	}
	return &servicesv1.CreateSspResponse{Ssp: req.Ssp}, nil
}

func (s *OscalServer) UpdateSsp(ctx context.Context, req *servicesv1.UpdateSspRequest) (*servicesv1.UpdateSspResponse, error) {
	if req.Ssp == nil {
		return nil, status.Error(codes.InvalidArgument, "ssp is required")
	}
	if err := s.store.UpdateSsp(ctx, req.Uuid.Value, req.Ssp); err != nil {
		return nil, notFoundErr("ssp", err)
	}
	return &servicesv1.UpdateSspResponse{Ssp: req.Ssp}, nil
}

func (s *OscalServer) DeleteSsp(ctx context.Context, req *servicesv1.DeleteSspRequest) (*servicesv1.DeleteSspResponse, error) {
	if err := s.store.DeleteSsp(ctx, req.Uuid.Value); err != nil {
		return nil, notFoundErr("ssp", err)
	}
	return &servicesv1.DeleteSspResponse{Success: true}, nil
}

// ---- Assessment Plan ----

func (s *OscalServer) GetAssessmentPlan(ctx context.Context, req *servicesv1.GetAssessmentPlanRequest) (*servicesv1.GetAssessmentPlanResponse, error) {
	ap, err := s.store.GetAssessmentPlan(ctx, req.Uuid.Value)
	if err != nil {
		return nil, notFoundErr("assessment plan", err)
	}
	return &servicesv1.GetAssessmentPlanResponse{AssessmentPlan: ap}, nil
}

func (s *OscalServer) ListAssessmentPlans(ctx context.Context, req *servicesv1.ListAssessmentPlansRequest) (*servicesv1.ListAssessmentPlansResponse, error) {
	aps, nextToken, err := s.store.ListAssessmentPlans(ctx, req.Filter, int(req.PageSize), req.PageToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list assessment plans: %v", err)
	}
	return &servicesv1.ListAssessmentPlansResponse{AssessmentPlans: aps, NextPageToken: nextToken}, nil
}

func (s *OscalServer) CreateAssessmentPlan(ctx context.Context, req *servicesv1.CreateAssessmentPlanRequest) (*servicesv1.CreateAssessmentPlanResponse, error) {
	if req.AssessmentPlan == nil {
		return nil, status.Error(codes.InvalidArgument, "assessment_plan is required")
	}
	if err := s.store.CreateAssessmentPlan(ctx, req.AssessmentPlan); err != nil {
		return nil, status.Errorf(codes.Internal, "create assessment plan: %v", err)
	}
	return &servicesv1.CreateAssessmentPlanResponse{AssessmentPlan: req.AssessmentPlan}, nil
}

func (s *OscalServer) UpdateAssessmentPlan(ctx context.Context, req *servicesv1.UpdateAssessmentPlanRequest) (*servicesv1.UpdateAssessmentPlanResponse, error) {
	if req.AssessmentPlan == nil {
		return nil, status.Error(codes.InvalidArgument, "assessment_plan is required")
	}
	if err := s.store.UpdateAssessmentPlan(ctx, req.Uuid.Value, req.AssessmentPlan); err != nil {
		return nil, notFoundErr("assessment plan", err)
	}
	return &servicesv1.UpdateAssessmentPlanResponse{AssessmentPlan: req.AssessmentPlan}, nil
}

func (s *OscalServer) DeleteAssessmentPlan(ctx context.Context, req *servicesv1.DeleteAssessmentPlanRequest) (*servicesv1.DeleteAssessmentPlanResponse, error) {
	if err := s.store.DeleteAssessmentPlan(ctx, req.Uuid.Value); err != nil {
		return nil, notFoundErr("assessment plan", err)
	}
	return &servicesv1.DeleteAssessmentPlanResponse{Success: true}, nil
}

// ---- Assessment Results ----

func (s *OscalServer) GetAssessmentResults(ctx context.Context, req *servicesv1.GetAssessmentResultsRequest) (*servicesv1.GetAssessmentResultsResponse, error) {
	ar, err := s.store.GetAssessmentResults(ctx, req.Uuid.Value)
	if err != nil {
		return nil, notFoundErr("assessment results", err)
	}
	return &servicesv1.GetAssessmentResultsResponse{AssessmentResults: ar}, nil
}

func (s *OscalServer) ListAssessmentResults(ctx context.Context, req *servicesv1.ListAssessmentResultsRequest) (*servicesv1.ListAssessmentResultsResponse, error) {
	ars, nextToken, err := s.store.ListAssessmentResults(ctx, req.Filter, int(req.PageSize), req.PageToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list assessment results: %v", err)
	}
	return &servicesv1.ListAssessmentResultsResponse{AssessmentResultsList: ars, NextPageToken: nextToken}, nil
}

func (s *OscalServer) CreateAssessmentResults(ctx context.Context, req *servicesv1.CreateAssessmentResultsRequest) (*servicesv1.CreateAssessmentResultsResponse, error) {
	if req.AssessmentResults == nil {
		return nil, status.Error(codes.InvalidArgument, "assessment_results is required")
	}
	if err := s.store.CreateAssessmentResults(ctx, req.AssessmentResults); err != nil {
		return nil, status.Errorf(codes.Internal, "create assessment results: %v", err)
	}
	return &servicesv1.CreateAssessmentResultsResponse{AssessmentResults: req.AssessmentResults}, nil
}

func (s *OscalServer) UpdateAssessmentResults(ctx context.Context, req *servicesv1.UpdateAssessmentResultsRequest) (*servicesv1.UpdateAssessmentResultsResponse, error) {
	if req.AssessmentResults == nil {
		return nil, status.Error(codes.InvalidArgument, "assessment_results is required")
	}
	if err := s.store.UpdateAssessmentResults(ctx, req.Uuid.Value, req.AssessmentResults); err != nil {
		return nil, notFoundErr("assessment results", err)
	}
	return &servicesv1.UpdateAssessmentResultsResponse{AssessmentResults: req.AssessmentResults}, nil
}

func (s *OscalServer) DeleteAssessmentResults(ctx context.Context, req *servicesv1.DeleteAssessmentResultsRequest) (*servicesv1.DeleteAssessmentResultsResponse, error) {
	if err := s.store.DeleteAssessmentResults(ctx, req.Uuid.Value); err != nil {
		return nil, notFoundErr("assessment results", err)
	}
	return &servicesv1.DeleteAssessmentResultsResponse{Success: true}, nil
}

// ---- POAM ----

func (s *OscalServer) GetPoam(ctx context.Context, req *servicesv1.GetPoamRequest) (*servicesv1.GetPoamResponse, error) {
	p, err := s.store.GetPoam(ctx, req.Uuid.Value)
	if err != nil {
		return nil, notFoundErr("poam", err)
	}
	return &servicesv1.GetPoamResponse{Poam: p}, nil
}

func (s *OscalServer) ListPoams(ctx context.Context, req *servicesv1.ListPoamsRequest) (*servicesv1.ListPoamsResponse, error) {
	poams, nextToken, err := s.store.ListPoams(ctx, req.Filter, int(req.PageSize), req.PageToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list poams: %v", err)
	}
	return &servicesv1.ListPoamsResponse{Poams: poams, NextPageToken: nextToken}, nil
}

func (s *OscalServer) CreatePoam(ctx context.Context, req *servicesv1.CreatePoamRequest) (*servicesv1.CreatePoamResponse, error) {
	if req.Poam == nil {
		return nil, status.Error(codes.InvalidArgument, "poam is required")
	}
	if err := s.store.CreatePoam(ctx, req.Poam); err != nil {
		return nil, status.Errorf(codes.Internal, "create poam: %v", err)
	}
	return &servicesv1.CreatePoamResponse{Poam: req.Poam}, nil
}

func (s *OscalServer) UpdatePoam(ctx context.Context, req *servicesv1.UpdatePoamRequest) (*servicesv1.UpdatePoamResponse, error) {
	if req.Poam == nil {
		return nil, status.Error(codes.InvalidArgument, "poam is required")
	}
	if err := s.store.UpdatePoam(ctx, req.Uuid.Value, req.Poam); err != nil {
		return nil, notFoundErr("poam", err)
	}
	return &servicesv1.UpdatePoamResponse{Poam: req.Poam}, nil
}

func (s *OscalServer) DeletePoam(ctx context.Context, req *servicesv1.DeletePoamRequest) (*servicesv1.DeletePoamResponse, error) {
	if err := s.store.DeletePoam(ctx, req.Uuid.Value); err != nil {
		return nil, notFoundErr("poam", err)
	}
	return &servicesv1.DeletePoamResponse{Success: true}, nil
}

// ---- Mapping ----

func (s *OscalServer) GetMapping(ctx context.Context, req *servicesv1.GetMappingRequest) (*servicesv1.GetMappingResponse, error) {
	m, err := s.store.GetMapping(ctx, req.Uuid.Value)
	if err != nil {
		return nil, notFoundErr("mapping", err)
	}
	return &servicesv1.GetMappingResponse{Mapping: m}, nil
}

func (s *OscalServer) ListMappings(ctx context.Context, req *servicesv1.ListMappingsRequest) (*servicesv1.ListMappingsResponse, error) {
	mappings, nextToken, err := s.store.ListMappings(ctx, req.Filter, int(req.PageSize), req.PageToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list mappings: %v", err)
	}
	return &servicesv1.ListMappingsResponse{Mappings: mappings, NextPageToken: nextToken}, nil
}

func (s *OscalServer) CreateMapping(ctx context.Context, req *servicesv1.CreateMappingRequest) (*servicesv1.CreateMappingResponse, error) {
	if req.Mapping == nil {
		return nil, status.Error(codes.InvalidArgument, "mapping is required")
	}
	if err := s.store.CreateMapping(ctx, req.Mapping); err != nil {
		return nil, status.Errorf(codes.Internal, "create mapping: %v", err)
	}
	return &servicesv1.CreateMappingResponse{Mapping: req.Mapping}, nil
}

func (s *OscalServer) UpdateMapping(ctx context.Context, req *servicesv1.UpdateMappingRequest) (*servicesv1.UpdateMappingResponse, error) {
	if req.Mapping == nil {
		return nil, status.Error(codes.InvalidArgument, "mapping is required")
	}
	if err := s.store.UpdateMapping(ctx, req.Uuid.Value, req.Mapping); err != nil {
		return nil, notFoundErr("mapping", err)
	}
	return &servicesv1.UpdateMappingResponse{Mapping: req.Mapping}, nil
}

func (s *OscalServer) DeleteMapping(ctx context.Context, req *servicesv1.DeleteMappingRequest) (*servicesv1.DeleteMappingResponse, error) {
	if err := s.store.DeleteMapping(ctx, req.Uuid.Value); err != nil {
		return nil, notFoundErr("mapping", err)
	}
	return &servicesv1.DeleteMappingResponse{Success: true}, nil
}

// ---- Search ----

func (s *OscalServer) Search(ctx context.Context, req *servicesv1.SearchRequest) (*servicesv1.SearchResponse, error) {
	results, nextToken, err := s.store.Search(ctx, req.Query, req.ModelTypes, int(req.PageSize), req.PageToken)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search: %v", err)
	}
	var out []*servicesv1.SearchResult
	for _, r := range results {
		out = append(out, &servicesv1.SearchResult{
			ModelType: r.ModelType,
			Uuid:      &commonv1.UUID{Value: r.UUID},
			Title:     r.Title,
			Score:     r.Score,
		})
	}
	return &servicesv1.SearchResponse{Results: out, NextPageToken: nextToken}, nil
}
