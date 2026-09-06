package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
)

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
