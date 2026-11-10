// Package ontology defines the formal multi-domain OWL/JSON-LD Regulatory & Threat
// Ontology for xOSCAL, spanning sovereign compliance frameworks, controls, ODP
// parameters, CISA KEV advisories, CVEs, CWEs, MITRE ATT&CK techniques, and
// CKODEX Rule 17 ANTI relations.
package ontology

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// OntologyVersion is the semantic version of the canonical xOSCAL ontology.
	OntologyVersion = "1.0.0"
	// OntologyNamespace is the canonical JSON-LD IRI namespace for the xOSCAL ontology.
	OntologyNamespace = "https://xoscal.dev/ns/ontology/v1#"
	// OntologySecNamespace is the canonical JSON-LD IRI prefix for security governance classes.
	OntologySecNamespace = "https://xoscal.dev/ns/ontology/v1/sec#"
	// OntologyRegNamespace is the canonical JSON-LD IRI prefix for regulatory compliance classes.
	OntologyRegNamespace = "https://xoscal.dev/ns/ontology/v1/reg#"
	// OntologyThreatNamespace is the canonical JSON-LD IRI prefix for threat intelligence classes.
	OntologyThreatNamespace = "https://xoscal.dev/ns/ontology/v1/threat#"
)

// Canonical Node Class constants (12 classes).
const (
	ClassJurisdiction       = "sec:Jurisdiction"
	ClassAuthority          = "sec:Authority"
	ClassFramework          = "reg:Framework"
	ClassProfile            = "reg:Profile"
	ClassControlFamily      = "reg:ControlFamily"
	ClassControl            = "reg:Control"
	ClassControlEnhancement = "reg:ControlEnhancement"
	ClassParameterODP       = "reg:ParameterODP"
	ClassKEVAdvisory        = "threat:KEVAdvisory"
	ClassThreatAdvisory     = ClassKEVAdvisory
	ClassCVE                = "threat:CVE"
	ClassCWE                = "threat:CWE"
	ClassAttackTechnique    = "threat:AttackTechnique"
)

// Canonical Predicate constants (16 predicates).
const (
	PredInJurisdiction      = "IN_JURISDICTION"
	PredGovernedByAuthority = "GOVERNED_BY_AUTHORITY"
	PredBelongsToFramework  = "BELONGS_TO_FRAMEWORK"
	PredInControlFamily     = "IN_CONTROL_FAMILY"
	PredEnhancesControl     = "ENHANCES_CONTROL"
	PredDefinesODP          = "DEFINES_ODP"
	PredSelectedByProfile   = "SELECTED_BY_PROFILE"
	PredEquivalentTo        = "EQUIVALENT_TO"
	PredHarmonizesWith      = "HARMONIZES_WITH"
	PredDerivedFrom         = "DERIVED_FROM"
	PredExploitsCWE         = "EXPLOITS_CWE"
	PredMapsToTechnique     = "MAPS_TO_TECHNIQUE"
	PredMitigatesCWE        = "MITIGATES_CWE"
	PredMitigatesTechnique  = "MITIGATES_TECHNIQUE"
	PredAttacksControl      = "ATTACKS_CONTROL"
	PredSupersedesFramework = "SUPERSEDES_FRAMEWORK"
)

// Canonical 12 Cross-Jurisdiction Thematic Domains.
const (
	DomainAccessControl           = "ACCESS_CONTROL"
	DomainCryptography            = "CRYPTOGRAPHY"
	DomainIncidentResponse        = "INCIDENT_RESPONSE"
	DomainSupplyChain             = "SUPPLY_CHAIN"
	DomainVulnerabilityManagement = "VULNERABILITY_MANAGEMENT"
	DomainAuditAccountability     = "AUDIT_ACCOUNTABILITY"
	DomainGovernanceRisk          = "GOVERNANCE_RISK"
	DomainSystemIntegrity         = "SYSTEM_INTEGRITY"
	DomainIdentityAuth            = "IDENTITY_AUTH"
	DomainPhysicalEnvironmental   = "PHYSICAL_ENVIRONMENTAL"
	DomainContinuityRecovery      = "CONTINUITY_RECOVERY"
	DomainDataPrivacy             = "DATA_PRIVACY"
)

// ThematicDomains lists all 12 canonical cross-jurisdiction security/privacy domains.
var ThematicDomains = []string{
	DomainAccessControl,
	DomainCryptography,
	DomainIncidentResponse,
	DomainSupplyChain,
	DomainVulnerabilityManagement,
	DomainAuditAccountability,
	DomainGovernanceRisk,
	DomainSystemIntegrity,
	DomainIdentityAuth,
	DomainPhysicalEnvironmental,
	DomainContinuityRecovery,
	DomainDataPrivacy,
}

// NodeClass describes a formal OWL/JSON-LD class in the xOSCAL ontology.
type NodeClass struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Domain      string   `json:"domain"` // "security", "regulatory", "threat"
	URNPattern  string   `json:"urn_pattern"`
	Description string   `json:"description"`
	Examples    []string `json:"examples,omitempty"`
}

// PredicateSpec describes a typed semantic predicate (object property) in the xOSCAL ontology.
type PredicateSpec struct {
	Name          string   `json:"name"`
	Category      string   `json:"category"` // "structural", "crosswalk", "threat", "anti"
	Domain        []string `json:"domain"`
	Range         []string `json:"range"`
	Inverse       string   `json:"inverse"`
	IsAnti        bool     `json:"is_anti"`
	DefaultWeight float64  `json:"default_weight"`
	Description   string   `json:"description"`
}

// Spec represents the complete exported JSON-LD ontology specification with cryptographic digest.
type Spec struct {
	Context         map[string]any  `json:"@context"`
	ID              string          `json:"@id"`
	Type            string          `json:"@type"`
	Version         string          `json:"version"`
	Namespace       string          `json:"namespace"`
	Classes         []NodeClass     `json:"classes"`
	Predicates      []PredicateSpec `json:"predicates"`
	ThematicDomains []string        `json:"thematic_domains"`
	SHA256          string          `json:"sha256"`
}

var canonicalClasses = []NodeClass{
	{
		ID:          ClassJurisdiction,
		Label:       "Sovereign Jurisdiction",
		Domain:      "security",
		URNPattern:  "urn:xoscal:jurisdiction:<code-iso>",
		Description: "Sovereign nation-state or supranational regulatory jurisdiction.",
		Examples:    []string{"urn:xoscal:jurisdiction:ca", "urn:xoscal:jurisdiction:us", "urn:xoscal:jurisdiction:eu", "urn:xoscal:jurisdiction:de", "urn:xoscal:jurisdiction:fr", "urn:xoscal:jurisdiction:gb"},
	},
	{
		ID:          ClassAuthority,
		Label:       "Governing Authority",
		Domain:      "security",
		URNPattern:  "urn:xoscal:authority:<slug>",
		Description: "Authoritative cybersecurity agency, standards body, or regulator issuing compliance mandates.",
		Examples:    []string{"urn:xoscal:authority:cccs-cse", "urn:xoscal:authority:nist", "urn:xoscal:authority:anssi", "urn:xoscal:authority:bsi", "urn:xoscal:authority:enisa", "urn:xoscal:authority:ncsc-uk"},
	},
	{
		ID:          ClassFramework,
		Label:       "Regulatory or Compliance Framework",
		Domain:      "regulatory",
		URNPattern:  "urn:xoscal:framework:<id>",
		Description: "Authoritative security, privacy, or AI governance catalogue or regulatory standard.",
		Examples:    []string{"urn:xoscal:framework:cccs-itsp-10-033", "urn:xoscal:framework:cccs-itsg-33", "urn:xoscal:framework:nist-csf-2.0", "urn:xoscal:framework:nis2"},
	},
	{
		ID:          ClassProfile,
		Label:       "Tailored Baseline Profile",
		Domain:      "regulatory",
		URNPattern:  "urn:xoscal:profile:<id>",
		Description: "Tailored control baseline selecting and parameterizing controls from an upstream catalogue.",
		Examples:    []string{"urn:xoscal:profile:cccs-medium-cloud-pbmm", "urn:xoscal:profile:cccs-itsp-10-033-profile"},
	},
	{
		ID:          ClassControlFamily,
		Label:       "Control Family / Security Domain",
		Domain:      "regulatory",
		URNPattern:  "urn:xoscal:family:<id>",
		Description: "Security control family (e.g. AC, AU, CM, IA, IR, RA, SC, SI, SR) or cross-jurisdiction thematic domain.",
		Examples:    []string{"urn:xoscal:family:AC", "urn:xoscal:family:SC", "urn:xoscal:family:ACCESS_CONTROL", "urn:xoscal:family:CRYPTOGRAPHY"},
	},
	{
		ID:          ClassControl,
		Label:       "Security or Privacy Control",
		Domain:      "regulatory",
		URNPattern:  "urn:xoscal:requirement:<fw>:<id>",
		Description: "Assessable base security, privacy, or operational resilience requirement within a framework.",
		Examples:    []string{"urn:xoscal:requirement:cccs-itsp-10-033:ac-2", "urn:xoscal:requirement:cccs-itsp-10-033:si-2"},
	},
	{
		ID:          ClassControlEnhancement,
		Label:       "Control Enhancement",
		Domain:      "regulatory",
		URNPattern:  "urn:xoscal:requirement:<fw>:<id>",
		Description: "Control enhancement augmenting a base security control with higher-assurance requirements.",
		Examples:    []string{"urn:xoscal:requirement:cccs-itsp-10-033:ac-2.1", "urn:xoscal:requirement:cccs-itsp-10-033:ia-2.1"},
	},
	{
		ID:          ClassParameterODP,
		Label:       "Organization-Defined Parameter (ODP)",
		Domain:      "regulatory",
		URNPattern:  "urn:xoscal:odp:<fw>:<param-id>",
		Description: "Organization-Defined Parameter (ODP) slot or assignment value within a control statement.",
		Examples:    []string{"urn:xoscal:odp:cccs-itsp-10-033:ac-02_odp.01"},
	},
	{
		ID:          ClassKEVAdvisory,
		Label:       "CISA Known Exploited Vulnerability Advisory",
		Domain:      "threat",
		URNPattern:  "urn:xoscal:kev:<cve-id>",
		Description: "Active exploitation advisory from the CISA Known Exploited Vulnerabilities (KEV) catalog.",
		Examples:    []string{"urn:xoscal:kev:CVE-2024-3400", "urn:xoscal:kev:CVE-2024-21887"},
	},
	{
		ID:          ClassCVE,
		Label:       "Common Vulnerabilities and Exposures Record",
		Domain:      "threat",
		URNPattern:  "urn:xoscal:cve:<cve-id>",
		Description: "Publicly disclosed cybersecurity vulnerability record.",
		Examples:    []string{"urn:xoscal:cve:CVE-2024-3400", "urn:xoscal:cve:CVE-2023-4966"},
	},
	{
		ID:          ClassCWE,
		Label:       "Common Weakness Enumeration Category",
		Domain:      "threat",
		URNPattern:  "urn:xoscal:cwe:<cwe-id>",
		Description: "Software or hardware weakness category exploited by vulnerabilities.",
		Examples:    []string{"urn:xoscal:cwe:CWE-77", "urn:xoscal:cwe:CWE-287", "urn:xoscal:cwe:CWE-89"},
	},
	{
		ID:          ClassAttackTechnique,
		Label:       "MITRE ATT&CK Enterprise Technique",
		Domain:      "threat",
		URNPattern:  "urn:xoscal:attack:<technique-id>",
		Description: "Adversary tactic and technique from the MITRE ATT&CK Enterprise matrix.",
		Examples:    []string{"urn:xoscal:attack:T1190", "urn:xoscal:attack:T1078", "urn:xoscal:attack:T1059"},
	},
}

var canonicalPredicates = []PredicateSpec{
	{
		Name:          PredInJurisdiction,
		Category:      "structural",
		Domain:        []string{ClassFramework, ClassProfile},
		Range:         []string{ClassJurisdiction},
		Inverse:       "HAS_FRAMEWORK_IN_JURISDICTION",
		IsAnti:        false,
		DefaultWeight: 1.0,
		Description:   "Binds a regulatory framework or profile to its sovereign jurisdiction.",
	},
	{
		Name:          PredGovernedByAuthority,
		Category:      "structural",
		Domain:        []string{ClassFramework, ClassProfile},
		Range:         []string{ClassAuthority},
		Inverse:       "GOVERNS_FRAMEWORK",
		IsAnti:        false,
		DefaultWeight: 1.0,
		Description:   "Associates a framework with its issuing national or international authority.",
	},
	{
		Name:          PredBelongsToFramework,
		Category:      "structural",
		Domain:        []string{ClassControl, ClassControlEnhancement, ClassProfile},
		Range:         []string{ClassFramework},
		Inverse:       "CONTAINS_REQUIREMENT",
		IsAnti:        false,
		DefaultWeight: 1.0,
		Description:   "Links a control, control enhancement, or profile to its parent compliance framework.",
	},
	{
		Name:          PredInControlFamily,
		Category:      "structural",
		Domain:        []string{ClassControl, ClassControlEnhancement},
		Range:         []string{ClassControlFamily},
		Inverse:       "FAMILY_INCLUDES_CONTROL",
		IsAnti:        false,
		DefaultWeight: 1.0,
		Description:   "Classifies a control or enhancement under a control family or canonical thematic domain.",
	},
	{
		Name:          PredEnhancesControl,
		Category:      "structural",
		Domain:        []string{ClassControlEnhancement},
		Range:         []string{ClassControl, ClassControlEnhancement},
		Inverse:       "HAS_ENHANCEMENT",
		IsAnti:        false,
		DefaultWeight: 1.0,
		Description:   "Connects a control enhancement to the base control or parent enhancement it augments.",
	},
	{
		Name:          PredDefinesODP,
		Category:      "structural",
		Domain:        []string{ClassControl, ClassControlEnhancement},
		Range:         []string{ClassParameterODP},
		Inverse:       "DEFINED_BY_CONTROL",
		IsAnti:        false,
		DefaultWeight: 1.0,
		Description:   "Declares an Organization-Defined Parameter (ODP) within a control or enhancement.",
	},
	{
		Name:          PredSelectedByProfile,
		Category:      "structural",
		Domain:        []string{ClassControl, ClassControlEnhancement},
		Range:         []string{ClassProfile},
		Inverse:       "PROFILE_SELECTS_CONTROL",
		IsAnti:        false,
		DefaultWeight: 1.0,
		Description:   "Indicates that a control or enhancement is selected in a baseline profile.",
	},
	{
		Name:          PredEquivalentTo,
		Category:      "crosswalk",
		Domain:        []string{ClassControl, ClassControlEnhancement},
		Range:         []string{ClassControl, ClassControlEnhancement},
		Inverse:       PredEquivalentTo,
		IsAnti:        false,
		DefaultWeight: 1.0,
		Description:   "Symmetric cross-framework control equivalency (1:1 normative correspondence).",
	},
	{
		Name:          PredHarmonizesWith,
		Category:      "crosswalk",
		Domain:        []string{ClassControl, ClassControlEnhancement, ClassControlFamily},
		Range:         []string{ClassControl, ClassControlEnhancement, ClassControlFamily},
		Inverse:       PredHarmonizesWith,
		IsAnti:        false,
		DefaultWeight: 0.85,
		Description:   "Multi-jurisdiction thematic or functional control harmonization across international standards.",
	},
	{
		Name:          PredDerivedFrom,
		Category:      "crosswalk",
		Domain:        []string{ClassFramework, ClassProfile, ClassControl, ClassControlEnhancement},
		Range:         []string{ClassFramework, ClassProfile, ClassControl, ClassControlEnhancement},
		Inverse:       "BASIS_FOR_DERIVATION",
		IsAnti:        false,
		DefaultWeight: 0.95,
		Description:   "Lineage relation showing that a framework, profile, or control is derived from an upstream baseline.",
	},
	{
		Name:          PredExploitsCWE,
		Category:      "threat",
		Domain:        []string{ClassCVE, ClassKEVAdvisory},
		Range:         []string{ClassCWE},
		Inverse:       "EXPLOITED_BY_VULNERABILITY",
		IsAnti:        false,
		DefaultWeight: 1.0,
		Description:   "Links a CVE or CISA KEV advisory to the underlying CWE weakness it exploits.",
	},
	{
		Name:          PredMapsToTechnique,
		Category:      "threat",
		Domain:        []string{ClassCVE, ClassKEVAdvisory, ClassCWE},
		Range:         []string{ClassAttackTechnique},
		Inverse:       "TECHNIQUE_USED_IN",
		IsAnti:        false,
		DefaultWeight: 1.0,
		Description:   "Associates a vulnerability, KEV advisory, or weakness with a MITRE ATT&CK Enterprise technique.",
	},
	{
		Name:          PredMitigatesCWE,
		Category:      "threat",
		Domain:        []string{ClassControl, ClassControlEnhancement},
		Range:         []string{ClassCWE},
		Inverse:       "MITIGATED_BY_CONTROL",
		IsAnti:        false,
		DefaultWeight: 1.0,
		Description:   "Specifies that a security control or enhancement mitigates a CWE weakness class.",
	},
	{
		Name:          PredMitigatesTechnique,
		Category:      "threat",
		Domain:        []string{ClassControl, ClassControlEnhancement},
		Range:         []string{ClassAttackTechnique},
		Inverse:       "TECHNIQUE_MITIGATED_BY",
		IsAnti:        false,
		DefaultWeight: 1.0,
		Description:   "Specifies that a security control or enhancement mitigates a MITRE ATT&CK technique.",
	},
	{
		Name:          PredAttacksControl,
		Category:      "anti",
		Domain:        []string{ClassKEVAdvisory, ClassCVE, ClassAttackTechnique},
		Range:         []string{ClassControl, ClassControlEnhancement},
		Inverse:       "ATTACKED_BY_THREAT",
		IsAnti:        true,
		DefaultWeight: 1.0,
		Description:   "CKODEX Rule 17 ANTI relation: active exploitation or adversary technique attacks/invalidates unpatched control posture.",
	},
	{
		Name:          PredSupersedesFramework,
		Category:      "anti",
		Domain:        []string{ClassFramework},
		Range:         []string{ClassFramework},
		Inverse:       "SUPERSEDED_BY_FRAMEWORK",
		IsAnti:        true,
		DefaultWeight: 1.0,
		Description:   "CKODEX Rule 17 ANTI relation: newer sovereign standard supersedes and deprecates a legacy framework.",
	},
}

// Classes returns a copy of the 12 canonical ontology node classes.
func Classes() []NodeClass {
	out := make([]NodeClass, len(canonicalClasses))
	copy(out, canonicalClasses)
	return out
}

// Predicates returns a copy of the 16 canonical ontology predicates.
func Predicates() []PredicateSpec {
	out := make([]PredicateSpec, len(canonicalPredicates))
	copy(out, canonicalPredicates)
	return out
}

// LookupClass returns the NodeClass definition for a given class ID.
func LookupClass(classID string) (NodeClass, bool) {
	for _, c := range canonicalClasses {
		if c.ID == classID {
			return c, true
		}
	}
	return NodeClass{}, false
}

// LookupPredicate returns the PredicateSpec for a given predicate name.
func LookupPredicate(predicate string) (PredicateSpec, bool) {
	for _, p := range canonicalPredicates {
		if p.Name == predicate {
			return p, true
		}
	}
	return PredicateSpec{}, false
}

// IsAntiPredicate returns true if the given predicate carries CKODEX Rule 17 ANTI semantics.
func IsAntiPredicate(predicate string) bool {
	if p, ok := LookupPredicate(predicate); ok {
		return p.IsAnti
	}
	return false
}

// ValidateTriple checks that subject class, predicate, and object class conform to the formal ontology domain and range constraints.
func ValidateTriple(fromClass, predicate, toClass string) error {
	if _, ok := LookupClass(fromClass); !ok {
		return fmt.Errorf("unknown ontology subject class %q", fromClass)
	}
	if _, ok := LookupClass(toClass); !ok {
		return fmt.Errorf("unknown ontology object class %q", toClass)
	}
	pred, ok := LookupPredicate(predicate)
	if !ok {
		return fmt.Errorf("unknown ontology predicate %q", predicate)
	}

	domainValid := false
	for _, d := range pred.Domain {
		if d == fromClass {
			domainValid = true
			break
		}
	}
	if !domainValid {
		return fmt.Errorf("subject class %q is not in domain %v of predicate %q", fromClass, pred.Domain, predicate)
	}

	rangeValid := false
	for _, r := range pred.Range {
		if r == toClass {
			rangeValid = true
			break
		}
	}
	if !rangeValid {
		return fmt.Errorf("object class %q is not in range %v of predicate %q", toClass, pred.Range, predicate)
	}

	return nil
}

// ComputeSpecSHA256 computes the deterministic SHA-256 hex digest over the complete
// canonical JSON-LD ontology specification (including @context, @id, @type, version,
// namespace, classes, predicates, and thematic_domains) with SHA256 zeroed prior to hashing.
func ComputeSpecSHA256(spec *Spec) string {
	if spec == nil {
		return ""
	}
	cp := *spec
	cp.SHA256 = ""
	raw, _ := json.Marshal(cp)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// CanonicalSpec builds the complete JSON-LD ontology specification with deterministic SHA-256 digest.
func CanonicalSpec() *Spec {
	ctxMap := map[string]any{
		"@vocab": OntologyNamespace,
		"owl":    "http://www.w3.org/2002/07/owl#",
		"rdfs":   "http://www.w3.org/2000/01/rdf-schema#",
		"xsd":    "http://www.w3.org/2001/XMLSchema#",
		"oscal":  "http://csrc.nist.gov/ns/oscal/1.0/",
		"sec":    OntologySecNamespace,
		"reg":    OntologyRegNamespace,
		"threat": OntologyThreatNamespace,
	}

	spec := &Spec{
		Context:         ctxMap,
		ID:              OntologyNamespace,
		Type:            "owl:Ontology",
		Version:         OntologyVersion,
		Namespace:       OntologyNamespace,
		Classes:         Classes(),
		Predicates:      Predicates(),
		ThematicDomains: append([]string(nil), ThematicDomains...),
	}
	spec.SHA256 = ComputeSpecSHA256(spec)

	return spec
}

// ClassifyThematicDomain maps a control across any of the 36 registered frameworks
// (NIST, CCCS ITSP.10.033, ITSG-33, PBMM, ITSP.10.171, CyberSecure Canada, CMMC,
// NIST CSF 2.0, BSI IT-Grundschutz, ANSSI, NIS2, DORA, EU AI Act, ISO 42001, UK Cyber Essentials)
// into one of the 12 canonical cross-jurisdiction thematic domains.
func ClassifyThematicDomain(ctrlID, familyID, title, statement string) string {
	cid := strings.ToLower(strings.TrimSpace(ctrlID))
	fid := strings.ToUpper(strings.TrimSpace(familyID))
	combined := strings.ToLower(title + " " + statement + " " + cid + " " + fid)

	// 1. Direct NIST / CCCS 2-letter family code or control ID prefix
	prefix := ""
	if len(cid) >= 2 && (strings.HasPrefix(cid[2:], "-") || strings.HasPrefix(cid[2:], ".")) {
		prefix = strings.ToUpper(cid[:2])
	}
	for _, code := range []string{fid, prefix} {
		switch code {
		case "AC":
			return DomainAccessControl
		case "IA":
			return DomainIdentityAuth
		case "SC":
			if strings.Contains(combined, "crypt") || strings.Contains(combined, "tls") || strings.Contains(combined, "cipher") || strings.Contains(combined, "key") || strings.Contains(combined, "confidentiality") || cid == "sc-8" || cid == "sc-12" || cid == "sc-13" || cid == "sc-28" {
				return DomainCryptography
			}
			return DomainSystemIntegrity
		case "AU":
			return DomainAuditAccountability
		case "IR":
			return DomainIncidentResponse
		case "RA":
			if strings.Contains(combined, "vulnerab") || strings.Contains(combined, "scan") || cid == "ra-5" {
				return DomainVulnerabilityManagement
			}
			return DomainGovernanceRisk
		case "SI":
			if strings.Contains(combined, "flaw") || strings.Contains(combined, "patch") || strings.Contains(combined, "vulnerab") || strings.Contains(combined, "malicious") || cid == "si-2" || cid == "si-3" {
				return DomainVulnerabilityManagement
			}
			return DomainSystemIntegrity
		case "CM", "SA", "MA":
			return DomainSystemIntegrity
		case "SR":
			return DomainSupplyChain
		case "CP":
			return DomainContinuityRecovery
		case "PE":
			return DomainPhysicalEnvironmental
		case "PT", "IP", "AR", "SE", "TR", "UL", "DM", "DI", "AP":
			return DomainDataPrivacy
		case "CA", "PM", "PL", "PS", "AT":
			return DomainGovernanceRisk
		}
	}

	// 2. NIST SP 800-171 / CCCS ITSP.10.171 / CMMC 3.x family codes
	cleanCID := strings.TrimPrefix(cid, "_")
	switch {
	case strings.HasPrefix(cleanCID, "03.01") || strings.HasPrefix(cleanCID, "3.1.") || strings.HasPrefix(fid, "03.01") || strings.HasPrefix(fid, "AC."):
		return DomainAccessControl
	case strings.HasPrefix(cleanCID, "03.05") || strings.HasPrefix(cleanCID, "3.5.") || strings.HasPrefix(fid, "03.05") || strings.HasPrefix(fid, "IA."):
		return DomainIdentityAuth
	case strings.HasPrefix(cleanCID, "03.03") || strings.HasPrefix(cleanCID, "3.3.") || strings.HasPrefix(fid, "03.03") || strings.HasPrefix(fid, "AU."):
		return DomainAuditAccountability
	case strings.HasPrefix(cleanCID, "03.06") || strings.HasPrefix(cleanCID, "3.6.") || strings.HasPrefix(fid, "03.06") || strings.HasPrefix(fid, "IR."):
		return DomainIncidentResponse
	case strings.HasPrefix(cleanCID, "03.10") || strings.HasPrefix(cleanCID, "3.10.") || strings.HasPrefix(fid, "03.10") || strings.HasPrefix(fid, "PE."):
		return DomainPhysicalEnvironmental
	case strings.HasPrefix(cleanCID, "03.11") || strings.HasPrefix(cleanCID, "3.11.") || strings.HasPrefix(cleanCID, "03.14") || strings.HasPrefix(cleanCID, "3.14."):
		return DomainVulnerabilityManagement
	case strings.HasPrefix(cleanCID, "03.13") || strings.HasPrefix(cleanCID, "3.13."):
		return DomainCryptography
	case strings.HasPrefix(cleanCID, "03.17") || strings.HasPrefix(cleanCID, "3.17."):
		return DomainSupplyChain
	case strings.HasPrefix(cleanCID, "03.04") || strings.HasPrefix(cleanCID, "3.4."):
		return DomainSystemIntegrity
	}

	// 3. CyberSecure Canada baseline groups (bc-1 .. bc-13, oc)
	switch {
	case strings.HasPrefix(cid, "bc-1"):
		return DomainIncidentResponse
	case strings.HasPrefix(cid, "bc-2") || strings.HasPrefix(cid, "bc-4"):
		return DomainVulnerabilityManagement
	case strings.HasPrefix(cid, "bc-5"):
		return DomainIdentityAuth
	case strings.HasPrefix(cid, "bc-6") || strings.HasPrefix(cid, "bc-8"):
		return DomainCryptography
	case strings.HasPrefix(cid, "bc-7"):
		return DomainContinuityRecovery
	case strings.HasPrefix(cid, "bc-3") || strings.HasPrefix(cid, "bc-9") || strings.HasPrefix(cid, "bc-10") || strings.HasPrefix(cid, "bc-11"):
		return DomainSystemIntegrity
	case strings.HasPrefix(cid, "bc-12"):
		return DomainAccessControl
	case strings.HasPrefix(cid, "oc"):
		return DomainGovernanceRisk
	}

	// 4. Keyword / semantic classification across EU/DE/FR/UK/US/CA standards
	switch {
	case strings.Contains(combined, "multi-factor") || strings.Contains(combined, "mfa") || strings.Contains(combined, "authenticat") || strings.Contains(combined, "password") || strings.Contains(combined, "credential") || strings.Contains(combined, "authentification"):
		return DomainIdentityAuth
	case strings.Contains(combined, "access control") || strings.Contains(combined, "least privilege") || strings.Contains(combined, "account management") || strings.Contains(combined, "contrôle d'accès") || strings.Contains(combined, "zugriff") || strings.Contains(combined, "berechtigung"):
		return DomainAccessControl
	case strings.Contains(combined, "crypt") || strings.Contains(combined, "encrypt") || strings.Contains(combined, "chiffrement") || strings.Contains(combined, "krypt") || strings.Contains(combined, "verschlüssel") || strings.Contains(combined, "tls"):
		return DomainCryptography
	case strings.Contains(combined, "incident") || strings.Contains(combined, "forensic") || strings.Contains(combined, "vorfall") || strings.Contains(combined, "detektion und reaktion") || strings.HasPrefix(fid, "DER"):
		return DomainIncidentResponse
	case strings.Contains(combined, "supply chain") || strings.Contains(combined, "supplier") || strings.Contains(combined, "third-party") || strings.Contains(combined, "vendor") || strings.Contains(combined, "fournisseur") || strings.Contains(combined, "lieferkette"):
		return DomainSupplyChain
	case strings.Contains(combined, "vulnerab") || strings.Contains(combined, "patch") || strings.Contains(combined, "flaw") || strings.Contains(combined, "malware") || strings.Contains(combined, "correctif") || strings.Contains(combined, "schwachstell"):
		return DomainVulnerabilityManagement
	case strings.Contains(combined, "audit") || strings.Contains(combined, "logging") || strings.Contains(combined, "journal") || strings.Contains(combined, "protokollierung") || strings.Contains(combined, "siem"):
		return DomainAuditAccountability
	case strings.Contains(combined, "backup") || strings.Contains(combined, "continuity") || strings.Contains(combined, "recovery") || strings.Contains(combined, "resilience") || strings.Contains(combined, "sauvegarde") || strings.Contains(combined, "notfall") || strings.Contains(combined, "datensicherung"):
		return DomainContinuityRecovery
	case strings.Contains(combined, "physical") || strings.Contains(combined, "facility") || strings.Contains(combined, "physique") || strings.Contains(combined, "infrastruktur") || strings.HasPrefix(fid, "INF"):
		return DomainPhysicalEnvironmental
	case strings.Contains(combined, "privacy") || strings.Contains(combined, "personal data") || strings.Contains(combined, "pii") || strings.Contains(combined, "gdpr") || strings.Contains(combined, "cnil") || strings.Contains(combined, "données personnelles") || strings.Contains(combined, "datenschutz"):
		return DomainDataPrivacy
	case strings.Contains(combined, "configuration") || strings.Contains(combined, "firewall") || strings.Contains(combined, "boundary") || strings.Contains(combined, "network") || strings.Contains(combined, "integrity") || strings.HasPrefix(fid, "SYS") || strings.HasPrefix(fid, "APP") || strings.HasPrefix(fid, "NET") || strings.HasPrefix(fid, "OPS"):
		return DomainSystemIntegrity
	default:
		return DomainGovernanceRisk
	}
}
