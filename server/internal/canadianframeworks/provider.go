package canadianframeworks

import (
	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	profilev1 "github.com/mchorfa/xoscal/proto/oscal/profile/v1"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
	"github.com/mchorfa/xoscal/server/internal/oscal"
)

func init() {
	RegisterAll(frameworks.DefaultRegistry())
}

// RegisterAll registers all authoritative Canadian frameworks into the provided registry.
func RegisterAll(r *frameworks.Registry) {
	// 1. CCCS ITSG-33
	_ = r.Register(frameworks.New(frameworks.Config{
		ID:           "cccs-itsg-33",
		Aliases:      []string{"itsg-33", "itsg33"},
		Title:        "CCCS ITSG-33 IT Security Risk Management: A Lifecycle Approach (Annex 3)",
		Version:      "1.2.3",
		Jurisdiction: "ca",
		Authority:    "Canadian Centre for Cyber Security / Communications Security Establishment",
		ControlCount: 42,
		HasProfile:   false,
		CatalogFn: func() ([]byte, error) {
			return oscal.ExportCatalogJSON(BuildCCCSITSG33Catalog())
		},
		CatalogProtoFn: func() (*catalogv1.Catalog, error) {
			return BuildCCCSITSG33Catalog(), nil
		},
	}))

	// 2. CyberSecure Canada
	_ = r.Register(frameworks.New(frameworks.Config{
		ID:           "cybersecure-canada",
		Aliases:      []string{"can-dgsi-104", "csc"},
		Title:        "CyberSecure Canada (CAN/DGSI 104)",
		Version:      "CAN/DGSI 104",
		Jurisdiction: "ca",
		Authority:    "Innovation, Science and Economic Development Canada (ISED) / CCCS",
		ControlCount: 46,
		HasProfile:   false,
		CatalogFn: func() ([]byte, error) {
			return oscal.ExportCatalogJSON(BuildCyberSecureCanadaCatalog())
		},
		CatalogProtoFn: func() (*catalogv1.Catalog, error) {
			return BuildCyberSecureCanadaCatalog(), nil
		},
	}))

	// 3. CCCS ITSP.10.171
	_ = r.Register(frameworks.New(frameworks.Config{
		ID:           "cccs-itsp-10-171",
		Aliases:      []string{"itsp.10.171", "itsp10171"},
		Title:        "CCCS ITSP.10.171 Security Requirements for Contracting with the Government of Canada",
		Version:      "1.2.3",
		Jurisdiction: "ca",
		Authority:    "Canadian Centre for Cyber Security / Communications Security Establishment",
		ControlCount: 267,
		HasProfile:   false,
		CatalogFn: func() ([]byte, error) {
			return oscal.ExportCatalogJSON(BuildCCCSITSP10171Catalog())
		},
		CatalogProtoFn: func() (*catalogv1.Catalog, error) {
			return BuildCCCSITSP10171Catalog(), nil
		},
	}))

	// 4. CCCS Medium Cloud PBMM Baseline
	_ = r.Register(frameworks.New(frameworks.Config{
		ID:           "cccs-medium-cloud-pbmm",
		Aliases:      []string{"pbmm", "medium-cloud-pbmm", "itsp.50.103"},
		Title:        "CCCS Medium Cloud PBMM Baseline (ITSP.50.103 / ITSG-33 Annex 4A Profile 1)",
		Version:      "ITSP.50.103",
		Jurisdiction: "ca",
		Authority:    "Canadian Centre for Cyber Security / Communications Security Establishment",
		ControlCount: 42,
		HasProfile:   true,
		CatalogFn: func() ([]byte, error) {
			return oscal.ExportCatalogJSON(BuildCCCSMediumCloudPBMMCatalog())
		},
		CatalogProtoFn: func() (*catalogv1.Catalog, error) {
			return BuildCCCSMediumCloudPBMMCatalog(), nil
		},
		ProfileFn: func() ([]byte, error) {
			return oscal.ExportProfileJSON(BuildCCCSMediumCloudPBMMProfile())
		},
		ProfileProtoFn: func() (*profilev1.Profile, error) {
			return BuildCCCSMediumCloudPBMMProfile(), nil
		},
	}))

	// 5. CCCS ITSP.10.033 & ITSP.10.033-01
	_ = r.Register(frameworks.New(frameworks.Config{
		ID:           "cccs-itsp-10-033",
		Aliases:      []string{"itsp.10.033", "itsp10033", "itsp-10-033"},
		Title:        "CCCS ITSP.10.033 Security and Privacy Controls and Assurance Activities Catalogue",
		Version:      "1.0 (2026-03-31)",
		Jurisdiction: "ca",
		Authority:    "Canadian Centre for Cyber Security / Communications Security Establishment",
		ControlCount: 330,
		HasProfile:   true,
		CatalogFn: func() ([]byte, error) {
			return itsp10033CatalogJSON, nil
		},
		ProfileFn: func() ([]byte, error) {
			return itsp10033ProfileJSON, nil
		},
	}))
}
