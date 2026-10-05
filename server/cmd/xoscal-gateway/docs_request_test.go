package main

import (
	"html"
	"os"
	"regexp"
	"strings"
	"testing"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
)

// Exercise the actual published example through the gateway's request decoder.
// OSCAL export JSON and protobuf request JSON differ in roots and scalar types.
func TestDocumentedAssessmentPlanRequest(t *testing.T) {
	document, err := os.ReadFile("../../../site/docs.html")
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`(?s)<pre id="assessment-plan-request"[^>]*><code>(.*?)</code></pre>`).FindSubmatch(document)
	if len(match) != 2 {
		t.Fatal("documented assessment-plan request example is missing")
	}
	var request servicesv1.CreateAssessmentPlanRequest
	if err := newOSCALMarshaler().NewDecoder(strings.NewReader(html.UnescapeString(string(match[1])))).Decode(&request); err != nil {
		t.Fatalf("published request body cannot be decoded: %v", err)
	}
	plan := request.GetAssessmentPlan()
	if plan.GetUuid().GetValue() != testUUID || plan.GetMetadata().GetOscalVersion() != "1.2.3" || plan.GetImportSsp().GetHref().GetValue() != "https://example.gov/ssp.json" {
		t.Fatal("published request body lost protobuf model fields")
	}
	if len(plan.GetReviewedControls().GetControlSelections()) != 1 || plan.GetReviewedControls().GetControlSelections()[0].GetIncludeAll() == nil {
		t.Fatal("published request body lost its reviewed-control selection")
	}
}

func TestRawOSCALDocumentIsNotGatewayRequest(t *testing.T) {
	var request servicesv1.CreateAssessmentPlanRequest
	// Unknown roots may be discarded by the gateway's configured decoder;
	// successful JSON decoding must not be mistaken for a usable request.
	err := newOSCALMarshaler().NewDecoder(strings.NewReader(`{"assessment-plan":{"uuid":"123e4567-e89b-42d3-a456-426614174000","metadata":{"title":"Raw OSCAL","version":"1.0","oscal-version":"1.2.3"}}}`)).Decode(&request)
	if err == nil && request.GetAssessmentPlan() != nil {
		t.Fatal("raw OSCAL document unexpectedly populated the request envelope")
	}
}
