package ontology_test

import (
	"testing"

	"github.com/mchorfa/xoscal/server/internal/ontology"
)

func TestOntologyClassesAndPredicates(t *testing.T) {
	spec := ontology.CanonicalSpec()
	if spec.Version != ontology.OntologyVersion {
		t.Fatalf("Version = %q, want %q", spec.Version, ontology.OntologyVersion)
	}
	if spec.Namespace != ontology.OntologyNamespace {
		t.Fatalf("Namespace = %q, want %q", spec.Namespace, ontology.OntologyNamespace)
	}
	if len(spec.Classes) != 12 {
		t.Fatalf("expected 12 canonical node classes, got %d", len(spec.Classes))
	}
	if len(spec.Predicates) != 16 {
		t.Fatalf("expected 16 canonical predicates, got %d", len(spec.Predicates))
	}
	if len(spec.ThematicDomains) != 12 {
		t.Fatalf("expected 12 canonical thematic domains, got %d", len(spec.ThematicDomains))
	}
	if len(spec.SHA256) != 64 {
		t.Fatalf("expected 64-hex SHA256 digest, got %q", spec.SHA256)
	}

	// Verify deterministic SHA256
	spec2 := ontology.CanonicalSpec()
	if spec.SHA256 != spec2.SHA256 {
		t.Fatalf("expected deterministic ontology SHA256, got %s != %s", spec.SHA256, spec2.SHA256)
	}

	// Verify ANTI predicates per CKODEX Rule 17
	if !ontology.IsAntiPredicate(ontology.PredAttacksControl) {
		t.Errorf("expected %s to be an ANTI predicate", ontology.PredAttacksControl)
	}
	if !ontology.IsAntiPredicate(ontology.PredSupersedesFramework) {
		t.Errorf("expected %s to be an ANTI predicate", ontology.PredSupersedesFramework)
	}
	if ontology.IsAntiPredicate(ontology.PredEquivalentTo) {
		t.Errorf("expected %s NOT to be an ANTI predicate", ontology.PredEquivalentTo)
	}
}

func TestValidateTriple(t *testing.T) {
	validCases := []struct {
		from string
		pred string
		to   string
	}{
		{ontology.ClassFramework, ontology.PredInJurisdiction, ontology.ClassJurisdiction},
		{ontology.ClassFramework, ontology.PredGovernedByAuthority, ontology.ClassAuthority},
		{ontology.ClassControl, ontology.PredBelongsToFramework, ontology.ClassFramework},
		{ontology.ClassControlEnhancement, ontology.PredInControlFamily, ontology.ClassControlFamily},
		{ontology.ClassControlEnhancement, ontology.PredEnhancesControl, ontology.ClassControl},
		{ontology.ClassControl, ontology.PredDefinesODP, ontology.ClassParameterODP},
		{ontology.ClassControl, ontology.PredSelectedByProfile, ontology.ClassProfile},
		{ontology.ClassControl, ontology.PredEquivalentTo, ontology.ClassControl},
		{ontology.ClassControl, ontology.PredHarmonizesWith, ontology.ClassControl},
		{ontology.ClassFramework, ontology.PredDerivedFrom, ontology.ClassFramework},
		{ontology.ClassKEVAdvisory, ontology.PredExploitsCWE, ontology.ClassCWE},
		{ontology.ClassKEVAdvisory, ontology.PredMapsToTechnique, ontology.ClassAttackTechnique},
		{ontology.ClassControl, ontology.PredMitigatesCWE, ontology.ClassCWE},
		{ontology.ClassControl, ontology.PredMitigatesTechnique, ontology.ClassAttackTechnique},
		{ontology.ClassKEVAdvisory, ontology.PredAttacksControl, ontology.ClassControl},
		{ontology.ClassFramework, ontology.PredSupersedesFramework, ontology.ClassFramework},
	}

	for _, tc := range validCases {
		if err := ontology.ValidateTriple(tc.from, tc.pred, tc.to); err != nil {
			t.Errorf("ValidateTriple(%q, %q, %q) unexpected error: %v", tc.from, tc.pred, tc.to, err)
		}
	}

	invalidCases := []struct {
		from string
		pred string
		to   string
	}{
		{"unknown:Class", ontology.PredInJurisdiction, ontology.ClassJurisdiction},
		{ontology.ClassFramework, "UNKNOWN_PRED", ontology.ClassJurisdiction},
		{ontology.ClassFramework, ontology.PredInJurisdiction, "unknown:Class"},
		{ontology.ClassJurisdiction, ontology.PredInJurisdiction, ontology.ClassFramework},
		{ontology.ClassControl, ontology.PredSupersedesFramework, ontology.ClassControl},
	}

	for _, tc := range invalidCases {
		if err := ontology.ValidateTriple(tc.from, tc.pred, tc.to); err == nil {
			t.Errorf("ValidateTriple(%q, %q, %q) expected error, got nil", tc.from, tc.pred, tc.to)
		}
	}
}

func TestClassifyThematicDomain(t *testing.T) {
	cases := []struct {
		ctrlID string
		famID  string
		title  string
		stmt   string
		want   string
	}{
		{"ac-2", "AC", "Account Management", "Manage system accounts", ontology.DomainAccessControl},
		{"ia-2", "IA", "Identification and Authentication", "Multi-factor authentication", ontology.DomainIdentityAuth},
		{"sc-13", "SC", "Cryptographic Protection", "FIPS-validated cryptography", ontology.DomainCryptography},
		{"si-2", "SI", "Flaw Remediation", "Install security-relevant software updates", ontology.DomainVulnerabilityManagement},
		{"ir-4", "IR", "Incident Handling", "Incident response capability", ontology.DomainIncidentResponse},
		{"sr-3", "SR", "Supply Chain Controls and Processes", "Supply chain risk management", ontology.DomainSupplyChain},
		{"au-2", "AU", "Event Logging", "Audit events", ontology.DomainAuditAccountability},
		{"cp-9", "CP", "System Backup", "Backup user-level and system-level information", ontology.DomainContinuityRecovery},
		{"pe-3", "PE", "Physical Access Control", "Physical access authorizations", ontology.DomainPhysicalEnvironmental},
		{"pt-2", "PT", "Authority to Process Personally Identifiable Information", "Privacy authority", ontology.DomainDataPrivacy},
		{"_03.01.01.A", "03.01", "Account Management", "Authorized account types", ontology.DomainAccessControl},
		{"bc-1-1", "BC-1", "Incident Response Plan", "Respond to incidents", ontology.DomainIncidentResponse},
	}

	for _, tc := range cases {
		got := ontology.ClassifyThematicDomain(tc.ctrlID, tc.famID, tc.title, tc.stmt)
		if got != tc.want {
			t.Errorf("ClassifyThematicDomain(%q, %q) = %q, want %q", tc.ctrlID, tc.famID, got, tc.want)
		}
	}
}
