package main

import (
	"io"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/encoding/protojson"

	assessment_planv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_plan/v1"
	assessment_resultsv1 "github.com/mchorfa/xoscal/proto/oscal/assessment_results/v1"
	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	componentv1 "github.com/mchorfa/xoscal/proto/oscal/component_definition/v1"
	poamv1 "github.com/mchorfa/xoscal/proto/oscal/poam/v1"
	profilev1 "github.com/mchorfa/xoscal/proto/oscal/profile/v1"
	sspv1 "github.com/mchorfa/xoscal/proto/oscal/ssp/v1"
	"github.com/mchorfa/xoscal/server/internal/oscal"
)

// oscalMarshaler wraps the default grpc-gateway JSONPb marshaler and
// overrides Marshal for OSCAL artifact types, producing OSCAL-compliant
// JSON (kebab-case, root model wrapper, unwrapped values) instead of
// protobuf JSON (camelCase, {"value":"..."} wrappers, no root wrapper).
//
// Unmarshal, NewDecoder, NewEncoder, ContentType and Name are delegated
// to the embedded JSONPb so that request parsing continues to accept
// protobuf JSON (the wire format clients send).
type oscalMarshaler struct {
	*gwruntime.JSONPb
}

// newOSCALMarshaler builds a marshaler that emits OSCAL-compliant JSON for
// OSCAL artifact types and falls back to protojson for everything else.
func newOSCALMarshaler() *oscalMarshaler {
	return &oscalMarshaler{
		JSONPb: &gwruntime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				Multiline:       true,
				EmitUnpopulated: false,
				Indent:          "  ",
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		},
	}
}

// Marshal serializes v to JSON. For OSCAL artifact types it uses the
// model-specific OSCAL exporters; for all other types it delegates to
// the default JSONPb (protojson).
func (m *oscalMarshaler) Marshal(v interface{}) ([]byte, error) {
	switch msg := v.(type) {
	case *catalogv1.Catalog:
		return oscal.ExportCatalogJSON(msg)
	case *profilev1.Profile:
		return oscal.ExportProfileJSON(msg)
	case *sspv1.SystemSecurityPlan:
		return oscal.ExportSSPJSON(msg)
	case *componentv1.ComponentDefinition:
		return oscal.ExportComponentDefinitionJSON(msg)
	case *assessment_planv1.AssessmentPlan:
		return oscal.ExportAssessmentPlanJSON(msg)
	case *assessment_resultsv1.AssessmentResults:
		return oscal.ExportAssessmentResultsJSON(msg)
	case *poamv1.PlanOfActionAndMilestones:
		return oscal.ExportPOAMJSON(msg)
	default:
		return m.JSONPb.Marshal(v)
	}
}

// Name satisfies the runtime.Marshaler interface.
func (m *oscalMarshaler) Name() string {
	return "oscal-json"
}

// NewDecoder delegates to the embedded JSONPb.
func (m *oscalMarshaler) NewDecoder(r io.Reader) gwruntime.Decoder {
	return m.JSONPb.NewDecoder(r)
}

// NewEncoder delegates to the embedded JSONPb.
func (m *oscalMarshaler) NewEncoder(w io.Writer) gwruntime.Encoder {
	return m.JSONPb.NewEncoder(w)
}

// ContentType delegates to the embedded JSONPb.
func (m *oscalMarshaler) ContentType(v interface{}) string {
	return m.JSONPb.ContentType(v)
}
