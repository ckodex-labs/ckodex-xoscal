package feeds

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/mchorfa/xoscal/server/internal/ontology"
	"github.com/mchorfa/xoscal/server/internal/resilience"
)

// Synced checkpoint file names persisted inside CheckpointDir on live feed sync.
const (
	SyncedCISAKEVFile     = "synced_cisa_kev.json"
	SyncedMITREAttackFile = "synced_mitre_attack.json"
)

// Feed IDs for the 4 authoritative built-in intelligence feeds.
const (
	FeedCrosswalk          = "cccs-nist-eu-crosswalk"
	FeedCISAKEV            = "cisa-kev-active-threats"
	FeedMITREAttack        = "mitre-attack-800-53"
	FeedSovereignLifecycle = "sovereign-regulatory-lifecycle"

	// DefaultMaxPayloadBytes bounds upstream live HTTP feed responses to 32 MiB
	// (accommodating ~9.5 MB NIST SP 800-53 Rev 5 OSCAL JSON while rejecting unbounded streams).
	DefaultMaxPayloadBytes int64 = 32 << 20
)

var (
	// ErrIntegrityTampered aliases resilience.ErrIntegrityTampered for checkpoint callers.
	ErrIntegrityTampered = resilience.ErrIntegrityTampered

	errPayloadTooLarge = errors.New("upstream feed payload exceeded maximum byte limit (truncated)")

	defaultCircuitRegistry = resilience.NewCircuitRegistry(resilience.CircuitConfig{
		FailureThreshold: 3,
		Cooldown:         30 * time.Second,
	})

	syncedCacheMu              sync.RWMutex
	latestSyncedAdvisories     []ThreatAdvisory
	latestSyncedAttackMappings []AttackMapping
)

// ResetSyncedCache clears the in-memory live-synced threat advisories and ATT&CK mappings cache.
func ResetSyncedCache() {
	syncedCacheMu.Lock()
	latestSyncedAdvisories = nil
	latestSyncedAttackMappings = nil
	syncedCacheMu.Unlock()
}

// FeedDescriptor describes an authoritative regulatory or threat intelligence feed.
type FeedDescriptor struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Category      string   `json:"category"` // "crosswalk", "threat-kev", "threat-attack", "lifecycle"
	Authority     string   `json:"authority"`
	Jurisdictions []string `json:"jurisdictions"`
	UpstreamURL   string   `json:"upstream_url"`
	AirGapReady   bool     `json:"air_gap_ready"`
	RecordCount   int      `json:"record_count"`
	SHA256        string   `json:"sha256"`
	Description   string   `json:"description"`
}

// FeedDigestEntry records the per-feed digest, resilience telemetry, and degraded-mode contract in a sync receipt.
type FeedDigestEntry struct {
	FeedID            string `json:"feed_id"`
	Mode              string `json:"mode"`
	RecordCount       int    `json:"record_count"`
	NodesProjected    int    `json:"nodes_projected"`
	EdgesProjected    int    `json:"edges_projected"`
	SHA256            string `json:"sha256"`
	CircuitState      string `json:"circuit_state"`
	Attempts          int    `json:"attempts"`
	LatencyMs         int64  `json:"latency_ms"`
	LiveRecordsParsed int    `json:"live_records_parsed,omitempty"`
	DegradedReason    string `json:"degraded_reason,omitempty"`
}

// FeedSyncReceipt is the immutable cryptographic receipt emitted after syncing feeds (CKODEX Rules 12 & 28-31).
type FeedSyncReceipt struct {
	ReceiptID       string            `json:"receipt_id"`
	Timestamp       time.Time         `json:"timestamp"`
	Mode            string            `json:"mode"`             // "airgap-embedded" or "live-http"
	OperationalMode string            `json:"operational_mode"` // "NORMAL" or "DEGRADED"
	FallbackCount   int               `json:"fallback_count"`
	Feeds           []FeedDigestEntry `json:"feeds"`
	TotalNodes      int               `json:"total_nodes_projected"`
	TotalEdges      int               `json:"total_edges_projected"`
	BundleSHA256    string            `json:"bundle_sha256"`
	OntologySHA256  string            `json:"ontology_sha256"`
}

// SyncOptions configures feed synchronization, resilience bounds, and atomic checkpointing.
type SyncOptions struct {
	FeedID          string
	Offline         bool
	HTTPClient      *http.Client
	BaseURL         string
	MaxPayloadBytes int64
	CircuitRegistry *resilience.CircuitRegistry
	RetryConfig     *resilience.RetryConfig
	CheckpointDir   string
}

// FlightRecorderEntry is an append-only hash-chained ledger record written to sync_flight_recorder.jsonl.
type FlightRecorderEntry struct {
	Sequence        int       `json:"sequence"`
	Timestamp       time.Time `json:"timestamp"`
	ReceiptID       string    `json:"receipt_id"`
	BundleSHA256    string    `json:"bundle_sha256"`
	OperationalMode string    `json:"operational_mode"`
	FallbackCount   int       `json:"fallback_count"`
	ReceiptSHA256   string    `json:"receipt_sha256"`
	PrevReceiptHash string    `json:"prev_receipt_sha256"`
	EntrySHA256     string    `json:"entry_sha256"`
}

// FeedInspection provides detailed records and digest metadata for a single feed.
type FeedInspection struct {
	Descriptor       FeedDescriptor         `json:"descriptor"`
	Crosswalks       []CrosswalkEntry       `json:"crosswalks,omitempty"`
	KEVAdvisories    []KEVAdvisoryEntry     `json:"kev_advisories,omitempty"`
	AttackTechniques []AttackTechniqueEntry `json:"attack_techniques,omitempty"`
	LifecycleEdges   []LifecycleEdgeEntry   `json:"lifecycle_edges,omitempty"`
}

func computeJSONDigest(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ListFeeds returns descriptors for all 4 authoritative built-in feeds with deterministic SHA-256 digests.
func ListFeeds() []FeedDescriptor {
	return []FeedDescriptor{
		{
			ID:            FeedCrosswalk,
			Title:         "CCCS ITSP.10.033 / NIST SP 800-53 / 800-171 / EU Multi-Jurisdiction Crosswalk",
			Category:      "crosswalk",
			Authority:     "CCCS / NIST / ENISA / BSI / ANSSI / NCSC",
			Jurisdictions: []string{"ca", "us", "eu", "de", "fr", "gb"},
			UpstreamURL:   "https://raw.githubusercontent.com/usnistgov/oscal-content/main/nist.gov/SP800-53/rev5/json/NIST_SP-800-53_rev5_catalog.json",
			AirGapReady:   true,
			RecordCount:   len(BuiltinCrosswalks),
			SHA256:        computeJSONDigest(BuiltinCrosswalks),
			Description:   "Multi-jurisdiction control equivalency and harmonization mappings across CA, US, EU, DE, FR, and GB standards.",
		},
		{
			ID:            FeedCISAKEV,
			Title:         "CISA Known Exploited Vulnerabilities (KEV) Active Threat Catalog",
			Category:      "threat-kev",
			Authority:     "CISA / CCCS Cyber Centre",
			Jurisdictions: []string{"us", "ca", "eu", "gb"},
			UpstreamURL:   "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json",
			AirGapReady:   true,
			RecordCount:   len(BuiltinKEVAdvisories),
			SHA256:        computeJSONDigest(BuiltinKEVAdvisories),
			Description:   "Active in-the-wild exploited CVEs with CWE weaknesses, ATT&CK techniques, and ATTACKS_CONTROL / MITIGATES_CWE control bindings.",
		},
		{
			ID:            FeedMITREAttack,
			Title:         "MITRE ATT&CK v16 Enterprise to NIST 800-53 Rev 5 / ITSP.10.033 Mappings",
			Category:      "threat-attack",
			Authority:     "MITRE Center for Threat-Informed Defense / NIST",
			Jurisdictions: []string{"us", "ca", "eu", "de", "fr", "gb"},
			UpstreamURL:   "https://raw.githubusercontent.com/center-for-threat-informed-defense/attack-control-framework-mappings/main/frameworks/attack_14_1/nist800_53_r5/stix/nist800-53-r5-mappings.json",
			AirGapReady:   true,
			RecordCount:   len(BuiltinAttackTechniques),
			SHA256:        computeJSONDigest(BuiltinAttackTechniques),
			Description:   "Enterprise adversary techniques mapped to mitigating NIST 800-53 Rev 5 and CCCS ITSP.10.033 controls.",
		},
		{
			ID:            FeedSovereignLifecycle,
			Title:         "Sovereign Regulatory Supersession & Baseline Derivation Feed",
			Category:      "lifecycle",
			Authority:     "CCCS / TBS / NIST / ANSSI / BSI / ENISA",
			Jurisdictions: []string{"ca", "us", "eu", "de", "fr", "gb"},
			UpstreamURL:   "https://www.cyber.gc.ca/en/guidance/itsp10033-security-and-privacy-controls-and-assurance-activities-catalogue",
			AirGapReady:   true,
			RecordCount:   len(BuiltinLifecycleEdges),
			SHA256:        computeJSONDigest(BuiltinLifecycleEdges),
			Description:   "Regulatory lifecycle transitions including ITSP.10.033 superseding ITSG-33 (effective 2026-03-31) and PBMM / ITSP.10.171 derivations.",
		},
	}
}

// InspectFeed returns the descriptor and active dataset (merging verified live-synced records
// over the air-gap baseline when present) for a specific feed ID.
func InspectFeed(feedID string, checkpointDirs ...string) (*FeedInspection, error) {
	id := strings.ToLower(strings.TrimSpace(feedID))
	for _, desc := range ListFeeds() {
		if desc.ID == id {
			out := &FeedInspection{Descriptor: desc}
			switch id {
			case FeedCrosswalk:
				out.Crosswalks = append([]CrosswalkEntry(nil), BuiltinCrosswalks...)
			case FeedCISAKEV:
				out.KEVAdvisories = ActiveKEVAdvisories(checkpointDirs...)
				out.Descriptor.RecordCount = len(out.KEVAdvisories)
			case FeedMITREAttack:
				out.AttackTechniques = ActiveAttackTechniques(checkpointDirs...)
				out.Descriptor.RecordCount = len(out.AttackTechniques)
			case FeedSovereignLifecycle:
				out.LifecycleEdges = append([]LifecycleEdgeEntry(nil), BuiltinLifecycleEdges...)
			}
			return out, nil
		}
	}
	return nil, fmt.Errorf("unknown feed %q (expected one of: %s, %s, %s, %s)",
		feedID, FeedCrosswalk, FeedCISAKEV, FeedMITREAttack, FeedSovereignLifecycle)
}

func resolveFeedCacheDir(checkpointDirs ...string) string {
	for _, d := range checkpointDirs {
		if trimmed := strings.TrimSpace(d); trimmed != "" {
			return filepath.Clean(trimmed)
		}
	}
	if envDir := strings.TrimSpace(os.Getenv("XOSCAL_FEED_CACHE_DIR")); envDir != "" {
		return filepath.Clean(envDir)
	}
	return ""
}

func mergeKEVAdvisories(base, synced []KEVAdvisoryEntry) []KEVAdvisoryEntry {
	out := make([]KEVAdvisoryEntry, len(base))
	idxByCVE := make(map[string]int, len(base)+len(synced))
	for i, b := range base {
		out[i] = b
		idxByCVE[strings.ToUpper(strings.TrimSpace(b.CVEID))] = i
	}
	for _, s := range synced {
		key := strings.ToUpper(strings.TrimSpace(s.CVEID))
		if key == "" {
			continue
		}
		if pos, exists := idxByCVE[key]; exists {
			out[pos] = s
		} else {
			idxByCVE[key] = len(out)
			out = append(out, s)
		}
	}
	return out
}

func mergeAttackTechniques(base, synced []AttackTechniqueEntry) []AttackTechniqueEntry {
	out := make([]AttackTechniqueEntry, len(base))
	idxByTech := make(map[string]int, len(base)+len(synced))
	for i, b := range base {
		out[i] = b
		idxByTech[strings.ToUpper(strings.TrimSpace(b.TechniqueID))] = i
	}
	for _, s := range synced {
		key := strings.ToUpper(strings.TrimSpace(s.TechniqueID))
		if key == "" {
			continue
		}
		if pos, exists := idxByTech[key]; exists {
			out[pos] = s
		} else {
			idxByTech[key] = len(out)
			out = append(out, s)
		}
	}
	return out
}

// ActiveThreatAdvisories returns the active CISA KEV threat advisories, preferring
// cryptographically verified synced_cisa_kev.json in checkpointDirs / XOSCAL_FEED_CACHE_DIR,
// then in-memory live-synced advisories, merged onto the air-gap BuiltinKEVAdvisories baseline.
func ActiveThreatAdvisories(checkpointDirs ...string) []ThreatAdvisory {
	if dir := resolveFeedCacheDir(checkpointDirs...); dir != "" {
		var synced []KEVAdvisoryEntry
		err := resilience.WithDirLock(dir, false, func() error {
			data, _, readErr := resilience.ReadVerifiedFile(filepath.Join(dir, SyncedCISAKEVFile), DefaultMaxPayloadBytes)
			if readErr != nil {
				return readErr
			}
			return json.Unmarshal(data, &synced)
		})
		if err == nil && len(synced) > 0 {
			return mergeKEVAdvisories(BuiltinKEVAdvisories, synced)
		}
	}
	syncedCacheMu.RLock()
	mem := append([]ThreatAdvisory(nil), latestSyncedAdvisories...)
	syncedCacheMu.RUnlock()
	if len(mem) > 0 {
		return mergeKEVAdvisories(BuiltinKEVAdvisories, mem)
	}
	return append([]KEVAdvisoryEntry(nil), BuiltinKEVAdvisories...)
}

// ActiveKEVAdvisories aliases ActiveThreatAdvisories.
func ActiveKEVAdvisories(checkpointDirs ...string) []KEVAdvisoryEntry {
	return ActiveThreatAdvisories(checkpointDirs...)
}

// ActiveAttackMappings returns the active MITRE ATT&CK technique mappings, preferring
// cryptographically verified synced_mitre_attack.json in checkpointDirs / XOSCAL_FEED_CACHE_DIR,
// then in-memory live-synced mappings, merged onto the air-gap BuiltinAttackTechniques baseline.
func ActiveAttackMappings(checkpointDirs ...string) []AttackMapping {
	if dir := resolveFeedCacheDir(checkpointDirs...); dir != "" {
		var synced []AttackTechniqueEntry
		err := resilience.WithDirLock(dir, false, func() error {
			data, _, readErr := resilience.ReadVerifiedFile(filepath.Join(dir, SyncedMITREAttackFile), DefaultMaxPayloadBytes)
			if readErr != nil {
				return readErr
			}
			return json.Unmarshal(data, &synced)
		})
		if err == nil && len(synced) > 0 {
			return mergeAttackTechniques(BuiltinAttackTechniques, synced)
		}
	}
	syncedCacheMu.RLock()
	mem := append([]AttackMapping(nil), latestSyncedAttackMappings...)
	syncedCacheMu.RUnlock()
	if len(mem) > 0 {
		return mergeAttackTechniques(BuiltinAttackTechniques, mem)
	}
	return append([]AttackTechniqueEntry(nil), BuiltinAttackTechniques...)
}

// ActiveAttackTechniques aliases ActiveAttackMappings.
func ActiveAttackTechniques(checkpointDirs ...string) []AttackTechniqueEntry {
	return ActiveAttackMappings(checkpointDirs...)
}

type upstreamKEVCatalog struct {
	CatalogVersion  string             `json:"catalogVersion"`
	Vulnerabilities []upstreamKEVEntry `json:"vulnerabilities"`
}

type upstreamKEVEntry struct {
	CVEID                      string   `json:"cveID"`
	VendorProject              string   `json:"vendorProject"`
	Product                    string   `json:"product"`
	VulnerabilityName          string   `json:"vulnerabilityName"`
	DateAdded                  string   `json:"dateAdded"`
	DueDate                    string   `json:"dueDate"`
	KnownRansomwareCampaignUse string   `json:"knownRansomwareCampaignUse"`
	ShortDescription           string   `json:"shortDescription"`
	CWEs                       []string `json:"cwes"`
}

func isValidCVEID(raw string) bool {
	s := strings.ToUpper(strings.TrimSpace(raw))
	if !strings.HasPrefix(s, "CVE-") {
		return false
	}
	parts := strings.Split(s, "-")
	if len(parts) != 3 || len(parts[1]) != 4 || len(parts[2]) < 4 {
		return false
	}
	for _, p := range parts[1:] {
		for i := 0; i < len(p); i++ {
			if p[i] < '0' || p[i] > '9' {
				return false
			}
		}
	}
	return true
}

func cweToControls(cweID string) (cweName string, techniques []string, attacked []string, mitigating []string) {
	norm := strings.ToUpper(strings.TrimSpace(cweID))
	switch norm {
	case "CWE-77", "CWE-78", "CWE-94":
		return "Command / Code Injection", []string{"T1190", "T1059"}, []string{"si-2", "si-10", "sc-7"}, []string{"si-2", "si-10", "sc-7", "cm-6", "ra-5"}
	case "CWE-89":
		return "SQL Injection", []string{"T1190", "T1059"}, []string{"si-10", "si-2", "sc-8"}, []string{"si-2", "si-10", "ra-5", "ir-4"}
	case "CWE-287", "CWE-306", "CWE-798":
		return "Improper or Missing Authentication", []string{"T1190", "T1078"}, []string{"ia-2", "ac-2", "si-2"}, []string{"si-2", "ia-2", "ac-2", "ac-6", "au-6"}
	case "CWE-22":
		return "Path Traversal", []string{"T1190", "T1552"}, []string{"si-10", "ac-3", "si-2"}, []string{"si-2", "si-10", "ac-3", "cm-6"}
	case "CWE-362":
		return "Race Condition", []string{"T1190", "T1068"}, []string{"si-2", "cm-6", "ac-6"}, []string{"si-2", "cm-6", "sc-7", "ra-5"}
	case "CWE-400":
		return "Uncontrolled Resource Consumption", []string{"T1190"}, []string{"sc-7", "si-2"}, []string{"sc-7", "si-2", "cm-6", "ca-7"}
	case "CWE-502":
		return "Deserialization of Untrusted Data", []string{"T1190", "T1059"}, []string{"si-10", "si-2", "cm-6"}, []string{"si-2", "si-10", "cm-6", "sc-7"}
	case "CWE-119", "CWE-120", "CWE-122", "CWE-787", "CWE-416":
		return "Memory Corruption / Buffer Overflow", []string{"T1190", "T1068"}, []string{"si-2", "sc-7", "cm-6"}, []string{"si-2", "sc-7", "cm-6", "ra-5"}
	case "CWE-79", "CWE-918":
		return "Web Input / SSRF Weakness", []string{"T1190"}, []string{"si-10", "sc-7", "si-2"}, []string{"si-2", "si-10", "sc-7", "cm-6"}
	default:
		if norm == "" {
			norm = "CWE-NVD-OTHER"
		}
		return norm, []string{"T1190"}, []string{"si-2", "cm-6"}, []string{"si-2", "cm-6", "ra-5"}
	}
}

// ParseUpstreamCISAKEV validates and parses an upstream CISA Known Exploited Vulnerabilities JSON catalog,
// rejecting captive-portal HTML or unrelated JSON and deterministically mapping CWEs to security controls.
func ParseUpstreamCISAKEV(data []byte) ([]KEVAdvisoryEntry, string, error) {
	if !json.Valid(data) {
		return nil, "", errors.New("invalid JSON in upstream CISA KEV response")
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return nil, "", fmt.Errorf("upstream CISA KEV must be a top-level JSON object: %w", err)
	}
	if _, hasVulns := top["vulnerabilities"]; !hasVulns {
		return nil, "", errors.New("upstream CISA KEV JSON missing required 'vulnerabilities' array")
	}

	var cat upstreamKEVCatalog
	if err := json.Unmarshal(data, &cat); err != nil {
		return nil, "", fmt.Errorf("decode CISA KEV catalog: %w", err)
	}
	if len(cat.Vulnerabilities) == 0 {
		return nil, "", errors.New("upstream CISA KEV catalog contains 0 vulnerabilities")
	}

	entries := make([]KEVAdvisoryEntry, 0, len(cat.Vulnerabilities))
	for i, v := range cat.Vulnerabilities {
		if !isValidCVEID(v.CVEID) {
			return nil, "", fmt.Errorf("invalid CVE ID %q at index %d", v.CVEID, i)
		}
		cveNorm := strings.ToUpper(strings.TrimSpace(v.CVEID))
		primaryCWE := "CWE-NVD-OTHER"
		for _, c := range v.CWEs {
			trimmed := strings.ToUpper(strings.TrimSpace(c))
			if strings.HasPrefix(trimmed, "CWE-") {
				primaryCWE = trimmed
				break
			}
		}
		cweName, techs, attacked, mitigating := cweToControls(primaryCWE)
		entries = append(entries, KEVAdvisoryEntry{
			CVEID:              cveNorm,
			Vendor:             strings.TrimSpace(v.VendorProject),
			Product:            strings.TrimSpace(v.Product),
			VulnerabilityName:  strings.TrimSpace(v.VulnerabilityName),
			DateAdded:          strings.TrimSpace(v.DateAdded),
			DueDate:            strings.TrimSpace(v.DueDate),
			RansomwareUse:      strings.TrimSpace(v.KnownRansomwareCampaignUse),
			CWEID:              primaryCWE,
			CWEName:            cweName,
			AttackTechniques:   techs,
			AttackedControls:   attacked,
			MitigatingControls: mitigating,
			ShortDescription:   strings.TrimSpace(v.ShortDescription),
		})
	}
	return entries, strings.TrimSpace(cat.CatalogVersion), nil
}

// ParseUpstreamMITREAttackEntries validates a MITRE CTID STIX 2.1 bundle or structured mapping object
// and returns extracted AttackTechniqueEntry records alongside the total count of valid upstream objects.
func ParseUpstreamMITREAttackEntries(data []byte) ([]AttackTechniqueEntry, int, error) {
	if !json.Valid(data) {
		return nil, 0, errors.New("invalid JSON in upstream MITRE ATT&CK response")
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return nil, 0, fmt.Errorf("upstream MITRE ATT&CK payload must be a top-level JSON object: %w", err)
	}

	if rawType, ok := top["type"]; ok {
		var bundleType string
		if err := json.Unmarshal(rawType, &bundleType); err == nil && bundleType == "bundle" {
			rawObjs, hasObjs := top["objects"]
			if !hasObjs {
				return nil, 0, errors.New("STIX bundle missing 'objects' array")
			}
			var objs []map[string]any
			if err := json.Unmarshal(rawObjs, &objs); err != nil {
				return nil, 0, fmt.Errorf("invalid STIX bundle 'objects': %w", err)
			}
			validCount := 0
			coaControls := make(map[string]string)
			mitigatesMap := make(map[string][]string)

			for _, o := range objs {
				t, _ := o["type"].(string)
				id, _ := o["id"].(string)
				if strings.TrimSpace(t) != "" || strings.TrimSpace(id) != "" {
					validCount++
				}
				switch t {
				case "course-of-action":
					ctrlID := extractSTIXExternalID(o, "")
					if ctrlID == "" {
						if name, _ := o["name"].(string); strings.Contains(name, "-") {
							ctrlID = strings.Fields(name)[0]
						}
					}
					if ctrlID != "" && id != "" {
						coaControls[id] = strings.ToLower(strings.TrimSpace(ctrlID))
					}
				case "relationship":
					relType, _ := o["relationship_type"].(string)
					srcRef, _ := o["source_ref"].(string)
					tgtRef, _ := o["target_ref"].(string)
					if relType == "mitigates" && srcRef != "" && tgtRef != "" {
						mitigatesMap[tgtRef] = append(mitigatesMap[tgtRef], srcRef)
					}
				}
			}
			if validCount == 0 {
				return nil, 0, errors.New("STIX bundle contains 0 valid objects")
			}

			var parsed []AttackTechniqueEntry
			for _, o := range objs {
				t, _ := o["type"].(string)
				if t != "attack-pattern" {
					continue
				}
				stixID, _ := o["id"].(string)
				techID := extractSTIXExternalID(o, "T")
				if techID == "" {
					if xid, _ := o["x_mitre_id"].(string); strings.HasPrefix(strings.ToUpper(strings.TrimSpace(xid)), "T") {
						techID = strings.ToUpper(strings.TrimSpace(xid))
					}
				}
				if techID == "" {
					continue
				}
				name, _ := o["name"].(string)
				if strings.TrimSpace(name) == "" {
					name = "MITRE ATT&CK " + techID
				}
				desc, _ := o["description"].(string)
				tactic := extractSTIXTactic(o)
				var mitCtrls []string
				for _, coaID := range mitigatesMap[stixID] {
					if cid := coaControls[coaID]; cid != "" {
						mitCtrls = append(mitCtrls, cid)
					}
				}
				if len(mitCtrls) == 0 {
					mitCtrls = []string{"si-2", "cm-6", "sc-7"}
				}
				dom := ontology.ClassifyThematicDomain(mitCtrls[0], "", name, desc)
				parsed = append(parsed, AttackTechniqueEntry{
					TechniqueID:        techID,
					Name:               strings.TrimSpace(name),
					Tactic:             tactic,
					ThematicDomain:     dom,
					MitigatingControls: mitCtrls,
					Description:        strings.TrimSpace(desc),
				})
			}
			return parsed, validCount, nil
		}
	}

	for _, key := range []string{"attack_techniques", "mappings"} {
		if rawArr, ok := top[key]; ok {
			var items []map[string]any
			if err := json.Unmarshal(rawArr, &items); err != nil {
				return nil, 0, fmt.Errorf("invalid %q array in MITRE ATT&CK payload: %w", key, err)
			}
			if len(items) == 0 {
				return nil, 0, fmt.Errorf("upstream MITRE ATT&CK %q array is empty", key)
			}
			var typed []AttackTechniqueEntry
			_ = json.Unmarshal(rawArr, &typed)
			var parsed []AttackTechniqueEntry
			for _, entry := range typed {
				tid := strings.ToUpper(strings.TrimSpace(entry.TechniqueID))
				if !strings.HasPrefix(tid, "T") {
					continue
				}
				entry.TechniqueID = tid
				if entry.Name == "" {
					entry.Name = "MITRE ATT&CK " + tid
				}
				if len(entry.MitigatingControls) == 0 {
					entry.MitigatingControls = []string{"si-2", "cm-6"}
				}
				if entry.ThematicDomain == "" {
					entry.ThematicDomain = ontology.ClassifyThematicDomain(entry.MitigatingControls[0], "", entry.Name, entry.Description)
				}
				parsed = append(parsed, entry)
			}
			return parsed, len(items), nil
		}
	}

	return nil, 0, errors.New("upstream MITRE ATT&CK JSON is neither a STIX bundle nor a mapping collection")
}

func extractSTIXExternalID(obj map[string]any, requiredPrefix string) string {
	refs, ok := obj["external_references"].([]any)
	if !ok {
		return ""
	}
	for _, r := range refs {
		rm, ok := r.(map[string]any)
		if !ok {
			continue
		}
		extID, _ := rm["external_id"].(string)
		extID = strings.ToUpper(strings.TrimSpace(extID))
		if extID == "" {
			continue
		}
		if requiredPrefix == "" || strings.HasPrefix(extID, requiredPrefix) {
			return extID
		}
	}
	return ""
}

func extractSTIXTactic(obj map[string]any) string {
	phases, ok := obj["kill_chain_phases"].([]any)
	if !ok {
		return "Enterprise"
	}
	for _, p := range phases {
		pm, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if pn, _ := pm["phase_name"].(string); strings.TrimSpace(pn) != "" {
			return strings.TrimSpace(pn)
		}
	}
	return "Enterprise"
}

// ParseUpstreamMITREAttackMappings validates a MITRE CTID STIX 2.1 bundle or structured mapping object
// and returns the number of valid objects parsed.
func ParseUpstreamMITREAttackMappings(data []byte) (int, error) {
	_, cnt, err := ParseUpstreamMITREAttackEntries(data)
	return cnt, err
}

func parseUpstreamCrosswalkJSON(data []byte) (int, error) {
	if !json.Valid(data) {
		return 0, errors.New("invalid JSON in upstream crosswalk/catalog response")
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return 0, fmt.Errorf("upstream crosswalk response must be a top-level JSON object: %w", err)
	}

	if rawCat, ok := top["catalog"]; ok {
		var cat struct {
			Controls []json.RawMessage `json:"controls"`
			Groups   []struct {
				Controls []json.RawMessage `json:"controls"`
			} `json:"groups"`
		}
		if err := json.Unmarshal(rawCat, &cat); err != nil {
			return 0, fmt.Errorf("invalid OSCAL catalog structure: %w", err)
		}
		total := len(cat.Controls)
		for _, g := range cat.Groups {
			total += len(g.Controls)
		}
		if total == 0 {
			return 0, errors.New("upstream OSCAL catalog contains 0 controls")
		}
		return total, nil
	}

	if rawCW, ok := top["crosswalks"]; ok {
		var cws []json.RawMessage
		if err := json.Unmarshal(rawCW, &cws); err != nil || len(cws) == 0 {
			return 0, errors.New("upstream crosswalks array is invalid or empty")
		}
		return len(cws), nil
	}

	return 0, errors.New("upstream JSON missing 'catalog' or 'crosswalks' root key")
}

func validateUpstreamLifecyclePayload(data []byte) (int, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) < 32 {
		return 0, fmt.Errorf("upstream lifecycle response too short (%d bytes < 32)", len(trimmed))
	}
	if bytes.IndexByte(trimmed, 0) >= 0 || !utf8.Valid(trimmed) {
		return 0, errors.New("upstream lifecycle response contains invalid UTF-8 or binary NUL bytes")
	}
	if trimmed[0] == '[' {
		return 0, errors.New("upstream lifecycle JSON payload must be a top-level object")
	}
	if trimmed[0] == '{' {
		if !json.Valid(trimmed) {
			return 0, errors.New("upstream lifecycle JSON payload is malformed")
		}
		var top map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &top); err != nil {
			return 0, fmt.Errorf("upstream lifecycle JSON must be an object: %w", err)
		}
		hasMarkerKey := false
		for _, k := range []string{"lifecycle_edges", "edges", "frameworks", "controls", "catalog", "version", "title"} {
			if _, ok := top[k]; ok {
				hasMarkerKey = true
				break
			}
		}
		if !hasMarkerKey {
			return 0, errors.New("upstream lifecycle JSON missing expected regulatory lifecycle keys")
		}
		return len(BuiltinLifecycleEdges), nil
	}

	lower := strings.ToLower(string(trimmed))
	for _, portalMarker := range []string{"captive portal", "airport wifi", "wi-fi login", "sign in to access the internet", "network authentication required"} {
		if strings.Contains(lower, portalMarker) {
			return 0, fmt.Errorf("upstream lifecycle response rejected: captive portal detected (%s)", portalMarker)
		}
	}
	hasRegMarker := false
	for _, regMarker := range []string{"itsp", "itsg", "cccs", "cyber.gc.ca", "nist", "oscal", "sovereign", "security", "control", "anssi", "bsi", "enisa"} {
		if strings.Contains(lower, regMarker) {
			hasRegMarker = true
			break
		}
	}
	if !hasRegMarker {
		return 0, errors.New("upstream lifecycle response missing sovereign regulatory content markers")
	}
	return len(BuiltinLifecycleEdges), nil
}

func validateUpstreamFeedPayload(feedID string, body []byte) (int, error) {
	switch feedID {
	case FeedCISAKEV:
		entries, _, err := ParseUpstreamCISAKEV(body)
		if err != nil {
			return 0, err
		}
		return len(entries), nil
	case FeedMITREAttack:
		return ParseUpstreamMITREAttackMappings(body)
	case FeedCrosswalk:
		return parseUpstreamCrosswalkJSON(body)
	case FeedSovereignLifecycle:
		return validateUpstreamLifecyclePayload(body)
	default:
		return 0, fmt.Errorf("unsupported feed %q", feedID)
	}
}

func isTransientStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// SyncFeeds synchronizes all (or a selected) regulatory and threat intelligence feed,
// enforcing payload ceilings, upstream structural validation, bounded retries, per-feed circuit breakers,
// explicit degraded-mode fallback, and optional atomic SHA-256 flight-recorder checkpointing.
func SyncFeeds(ctx context.Context, opts SyncOptions) (*FeedSyncReceipt, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	allFeeds := ListFeeds()
	var selected []FeedDescriptor
	targetID := strings.ToLower(strings.TrimSpace(opts.FeedID))
	if targetID != "" {
		for _, f := range allFeeds {
			if f.ID == targetID {
				selected = append(selected, f)
				break
			}
		}
		if len(selected) == 0 {
			return nil, fmt.Errorf("unknown feed %q", opts.FeedID)
		}
	} else {
		selected = allFeeds
	}

	httpClient := opts.HTTPClient
	if !opts.Offline && httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	maxBytes := opts.MaxPayloadBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxPayloadBytes
	}

	circuitReg := opts.CircuitRegistry
	if circuitReg == nil {
		circuitReg = defaultCircuitRegistry
	}

	retryCfg := resilience.RetryConfig{
		MaxRetries: 2,
		BaseDelay:  50 * time.Millisecond,
		MaxDelay:   500 * time.Millisecond,
	}
	if opts.RetryConfig != nil {
		retryCfg = *opts.RetryConfig
	}

	mode := "airgap-embedded"
	if !opts.Offline {
		mode = "live-http"
	}

	var entries []FeedDigestEntry
	var syncedKEV []KEVAdvisoryEntry
	var syncedAttack []AttackTechniqueEntry
	totalNodes := 0
	totalEdges := 0
	fallbackCount := 0

	for _, f := range selected {
		cb := circuitReg.Get(f.ID)
		feedMode := "airgap-embedded"
		digest := f.SHA256
		circuitState := cb.Snapshot().State
		attempts := 0
		var latencyMs int64
		liveRecords := 0
		effectiveRecords := f.RecordCount
		degradedReason := ""

		targetURL := f.UpstreamURL
		if opts.BaseURL != "" {
			targetURL = strings.TrimRight(opts.BaseURL, "/") + "/" + f.ID
		}

		if !opts.Offline && httpClient != nil && targetURL != "" {
			if ctxErr := ctx.Err(); ctxErr != nil {
				fallbackCount++
				degradedReason = fmt.Sprintf("caller context cancelled: %v", ctxErr)
			} else {
				allowed, effState := cb.Allow()
				circuitState = effState
				if !allowed {
					fallbackCount++
					degradedReason = fmt.Sprintf("circuit breaker %s for feed %s", effState, f.ID)
				} else {
					start := time.Now()
					var fetchedBody []byte
					var parsedCount int

					att, fetchErr := resilience.RetryWithBackoff(ctx, retryCfg, func(attemptCtx context.Context, _ int) error {
						req, reqBuildErr := http.NewRequestWithContext(attemptCtx, http.MethodGet, targetURL, nil)
						if reqBuildErr != nil {
							return reqBuildErr
						}
						resp, doErr := httpClient.Do(req)
						if doErr != nil {
							if attemptCtx.Err() != nil {
								return attemptCtx.Err()
							}
							return resilience.MarkRetryable(fmt.Errorf("upstream request failed: %w", doErr))
						}
						body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
						_ = resp.Body.Close()
						if readErr != nil {
							if attemptCtx.Err() != nil {
								return attemptCtx.Err()
							}
							return resilience.MarkRetryable(fmt.Errorf("read upstream body: %w", readErr))
						}
						if resp.StatusCode != http.StatusOK {
							statusErr := fmt.Errorf("upstream returned HTTP %d", resp.StatusCode)
							if isTransientStatus(resp.StatusCode) {
								return resilience.MarkRetryable(statusErr)
							}
							return statusErr
						}
						if int64(len(body)) > maxBytes {
							return fmt.Errorf("%w: exceeded %d bytes", errPayloadTooLarge, maxBytes)
						}
						if len(body) == 0 {
							return errors.New("upstream returned empty 200 OK body")
						}
						cnt, valErr := validateUpstreamFeedPayload(f.ID, body)
						if valErr != nil {
							return fmt.Errorf("upstream payload validation failed: %w", valErr)
						}
						fetchedBody = body
						parsedCount = cnt
						return nil
					})

					attempts = att
					latencyMs = time.Since(start).Milliseconds()

					if fetchErr == nil && len(fetchedBody) > 0 {
						cb.RecordSuccess()
						circuitState = cb.Snapshot().State
						sum := sha256.Sum256(fetchedBody)
						digest = hex.EncodeToString(sum[:])
						feedMode = "live-http"
						liveRecords = parsedCount
						switch f.ID {
						case FeedCISAKEV:
							if kevParsed, _, kErr := ParseUpstreamCISAKEV(fetchedBody); kErr == nil && len(kevParsed) > 0 {
								syncedKEV = kevParsed
								syncedCacheMu.Lock()
								latestSyncedAdvisories = append([]ThreatAdvisory(nil), kevParsed...)
								syncedCacheMu.Unlock()
								effectiveRecords = len(mergeKEVAdvisories(BuiltinKEVAdvisories, kevParsed))
							}
						case FeedMITREAttack:
							if attParsed, _, aErr := ParseUpstreamMITREAttackEntries(fetchedBody); aErr == nil && len(attParsed) > 0 {
								syncedAttack = attParsed
								syncedCacheMu.Lock()
								latestSyncedAttackMappings = append([]AttackMapping(nil), attParsed...)
								syncedCacheMu.Unlock()
								effectiveRecords = len(mergeAttackTechniques(BuiltinAttackTechniques, attParsed))
							}
						}
					} else if ctx.Err() != nil {
						cb.CancelProbe()
						circuitState = cb.Snapshot().State
						fallbackCount++
						degradedReason = fmt.Sprintf("caller context cancelled: %v", ctx.Err())
					} else {
						cb.RecordFailure()
						circuitState = cb.Snapshot().State
						fallbackCount++
						if fetchErr != nil {
							degradedReason = fetchErr.Error()
						} else {
							degradedReason = "upstream sync fell back to embedded air-gap dataset"
						}
					}
				}
			}
		}

		nodesProj, edgesProj := estimateFeedGraphContributionCount(f.ID, effectiveRecords)
		totalNodes += nodesProj
		totalEdges += edgesProj

		entries = append(entries, FeedDigestEntry{
			FeedID:            f.ID,
			Mode:              feedMode,
			RecordCount:       effectiveRecords,
			NodesProjected:    nodesProj,
			EdgesProjected:    edgesProj,
			SHA256:            digest,
			CircuitState:      circuitState,
			Attempts:          attempts,
			LatencyMs:         latencyMs,
			LiveRecordsParsed: liveRecords,
			DegradedReason:    degradedReason,
		})
	}

	opMode := "NORMAL"
	if !opts.Offline && fallbackCount > 0 {
		opMode = "DEGRADED"
	}

	type canonicalFeedHashItem struct {
		FeedID         string `json:"feed_id"`
		Mode           string `json:"mode"`
		RecordCount    int    `json:"record_count"`
		NodesProjected int    `json:"nodes_projected"`
		EdgesProjected int    `json:"edges_projected"`
		SHA256         string `json:"sha256"`
	}
	canonItems := make([]canonicalFeedHashItem, len(entries))
	for i, e := range entries {
		canonItems[i] = canonicalFeedHashItem{
			FeedID:         e.FeedID,
			Mode:           e.Mode,
			RecordCount:    e.RecordCount,
			NodesProjected: e.NodesProjected,
			EdgesProjected: e.EdgesProjected,
			SHA256:         e.SHA256,
		}
	}

	ontSpec := ontologySpecDigest()
	bundleInput, _ := json.Marshal(struct {
		Mode            string                  `json:"mode"`
		OperationalMode string                  `json:"operational_mode"`
		Feeds           []canonicalFeedHashItem `json:"feeds"`
		OntologySHA256  string                  `json:"ontology_sha256"`
	}{
		Mode:            mode,
		OperationalMode: opMode,
		Feeds:           canonItems,
		OntologySHA256:  ontSpec,
	})
	bundleSum := sha256.Sum256(bundleInput)
	bundleHex := hex.EncodeToString(bundleSum[:])

	rcpt := &FeedSyncReceipt{
		ReceiptID:       "rcpt_feeds_" + bundleHex[:16],
		Timestamp:       time.Now().UTC(),
		Mode:            mode,
		OperationalMode: opMode,
		FallbackCount:   fallbackCount,
		Feeds:           entries,
		TotalNodes:      totalNodes,
		TotalEdges:      totalEdges,
		BundleSHA256:    bundleHex,
		OntologySHA256:  ontSpec,
	}

	if strings.TrimSpace(opts.CheckpointDir) != "" {
		if err := saveSyncCheckpointWithPayloads(opts.CheckpointDir, rcpt, syncedKEV, syncedAttack); err != nil {
			return nil, fmt.Errorf("persist feed sync checkpoint: %w", err)
		}
	}

	return rcpt, nil
}

func computeLedgerEntryHash(e FlightRecorderEntry) string {
	payload, _ := json.Marshal(struct {
		Sequence        int    `json:"sequence"`
		Timestamp       string `json:"timestamp"`
		ReceiptID       string `json:"receipt_id"`
		BundleSHA256    string `json:"bundle_sha256"`
		OperationalMode string `json:"operational_mode"`
		FallbackCount   int    `json:"fallback_count"`
		ReceiptSHA256   string `json:"receipt_sha256"`
		PrevReceiptHash string `json:"prev_receipt_sha256"`
	}{
		Sequence:        e.Sequence,
		Timestamp:       e.Timestamp.UTC().Format(time.RFC3339Nano),
		ReceiptID:       e.ReceiptID,
		BundleSHA256:    e.BundleSHA256,
		OperationalMode: e.OperationalMode,
		FallbackCount:   e.FallbackCount,
		ReceiptSHA256:   e.ReceiptSHA256,
		PrevReceiptHash: e.PrevReceiptHash,
	})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// SaveSyncCheckpoint atomically persists latest_sync_receipt.json + .sha256 and appends a
// hash-chained FlightRecorderEntry to sync_flight_recorder.jsonl inside dir.
func SaveSyncCheckpoint(dir string, receipt *FeedSyncReceipt) error {
	return saveSyncCheckpointWithPayloads(dir, receipt, nil, nil)
}

func saveSyncCheckpointWithPayloads(dir string, receipt *FeedSyncReceipt, syncedKEV []KEVAdvisoryEntry, syncedAttack []AttackTechniqueEntry) error {
	if strings.TrimSpace(dir) == "" {
		return errors.New("checkpoint directory is required")
	}
	if receipt == nil {
		return errors.New("feed sync receipt is nil")
	}

	cleanDir := filepath.Clean(dir)
	return resilience.WithDirLock(cleanDir, true, func() error {
		if err := os.MkdirAll(cleanDir, 0o750); err != nil {
			return fmt.Errorf("create checkpoint dir %s: %w", cleanDir, err)
		}

		if len(syncedKEV) > 0 {
			kevBytes, err := json.MarshalIndent(syncedKEV, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal synced_cisa_kev.json: %w", err)
			}
			kevBytes = append(kevBytes, '\n')
			if _, err := resilience.WriteFileAtomicWithSHA256(filepath.Join(cleanDir, SyncedCISAKEVFile), kevBytes); err != nil {
				return fmt.Errorf("write %s: %w", SyncedCISAKEVFile, err)
			}
		}

		if len(syncedAttack) > 0 {
			attBytes, err := json.MarshalIndent(syncedAttack, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal synced_mitre_attack.json: %w", err)
			}
			attBytes = append(attBytes, '\n')
			if _, err := resilience.WriteFileAtomicWithSHA256(filepath.Join(cleanDir, SyncedMITREAttackFile), attBytes); err != nil {
				return fmt.Errorf("write %s: %w", SyncedMITREAttackFile, err)
			}
		}

		ledgerPath := filepath.Join(cleanDir, "sync_flight_recorder.jsonl")
		prevHash := strings.Repeat("0", 64)
		nextSeq := 1

		if _, statErr := os.Stat(ledgerPath); statErr == nil {
			entries, verifyErr := readAndVerifyLedger(ledgerPath)
			if verifyErr != nil {
				return verifyErr
			}
			if len(entries) > 0 {
				last := entries[len(entries)-1]
				prevHash = last.EntrySHA256
				nextSeq = last.Sequence + 1
			}
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return fmt.Errorf("stat flight recorder ledger: %w", statErr)
		}

		rcptBytes, err := json.MarshalIndent(receipt, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal receipt: %w", err)
		}
		rcptBytes = append(rcptBytes, '\n')

		latestPath := filepath.Join(cleanDir, "latest_sync_receipt.json")
		rcptHex, err := resilience.WriteFileAtomicWithSHA256(latestPath, rcptBytes)
		if err != nil {
			return fmt.Errorf("write latest_sync_receipt.json: %w", err)
		}

		entry := FlightRecorderEntry{
			Sequence:        nextSeq,
			Timestamp:       receipt.Timestamp.UTC(),
			ReceiptID:       receipt.ReceiptID,
			BundleSHA256:    receipt.BundleSHA256,
			OperationalMode: receipt.OperationalMode,
			FallbackCount:   receipt.FallbackCount,
			ReceiptSHA256:   rcptHex,
			PrevReceiptHash: prevHash,
		}
		entry.EntrySHA256 = computeLedgerEntryHash(entry)

		lineBytes, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("marshal flight recorder entry: %w", err)
		}
		lineBytes = append(lineBytes, '\n')

		// #nosec G304 -- ledgerPath is confined inside operator-supplied cleanDir.
		f, err := os.OpenFile(ledgerPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return fmt.Errorf("open flight recorder ledger: %w", err)
		}
		if _, err := f.Write(lineBytes); err != nil {
			_ = f.Close()
			return fmt.Errorf("append flight recorder ledger: %w", err)
		}
		if err := f.Sync(); err != nil {
			_ = f.Close()
			return fmt.Errorf("fsync flight recorder ledger: %w", err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("close flight recorder ledger: %w", err)
		}

		return nil
	})
}

// LoadVerifiedCheckpoint reads and cryptographically verifies latest_sync_receipt.json against its .sha256 sidecar.
func LoadVerifiedCheckpoint(dir string) (*FeedSyncReceipt, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("checkpoint directory is required")
	}
	cleanDir := filepath.Clean(dir)
	var rcpt *FeedSyncReceipt
	err := resilience.WithDirLock(cleanDir, false, func() error {
		var loadErr error
		rcpt, _, loadErr = loadVerifiedCheckpointWithDigest(cleanDir)
		return loadErr
	})
	if err != nil {
		return nil, err
	}
	return rcpt, nil
}

func loadVerifiedCheckpointWithDigest(dir string) (*FeedSyncReceipt, string, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, "", errors.New("checkpoint directory is required")
	}
	latestPath := filepath.Join(filepath.Clean(dir), "latest_sync_receipt.json")
	data, digestHex, err := resilience.ReadVerifiedFile(latestPath, 16<<20)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", fmt.Errorf("%w: missing latest_sync_receipt.json in %s", ErrIntegrityTampered, dir)
		}
		return nil, "", err
	}
	var rcpt FeedSyncReceipt
	if err := json.Unmarshal(data, &rcpt); err != nil {
		return nil, "", fmt.Errorf("%w: malformed latest_sync_receipt.json: %v", ErrIntegrityTampered, err)
	}
	return &rcpt, digestHex, nil
}

func readAndVerifyLedger(ledgerPath string) ([]FlightRecorderEntry, error) {
	const maxLedgerBytes int64 = 64 << 20

	// #nosec G304 -- ledgerPath is derived from clean checkpoint directory.
	f, err := os.Open(ledgerPath)
	if err != nil {
		return nil, fmt.Errorf("%w: open ledger %s: %v", ErrIntegrityTampered, ledgerPath, err)
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("%w: stat ledger %s: %v", ErrIntegrityTampered, ledgerPath, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: ledger %s is not a regular file", ErrIntegrityTampered, ledgerPath)
	}
	if info.Size() > maxLedgerBytes {
		return nil, fmt.Errorf("%w: ledger %s exceeds %d bytes", resilience.ErrPayloadTooLarge, ledgerPath, maxLedgerBytes)
	}

	scanner := bufio.NewScanner(io.LimitReader(f, maxLedgerBytes+1))
	scanner.Buffer(make([]byte, 64*1024), 4<<20)

	var entries []FlightRecorderEntry
	expectedPrev := strings.Repeat("0", 64)
	seq := 0

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		seq++
		var e FlightRecorderEntry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("%w: invalid JSON on flight recorder line %d: %v", ErrIntegrityTampered, seq, err)
		}
		if e.Sequence != seq {
			return nil, fmt.Errorf("%w: flight recorder sequence discontinuity at line %d (got %d)", ErrIntegrityTampered, seq, e.Sequence)
		}
		if subtle.ConstantTimeCompare([]byte(strings.ToLower(e.PrevReceiptHash)), []byte(expectedPrev)) != 1 {
			return nil, fmt.Errorf("%w: broken hash chain at sequence %d (prev_receipt_sha256=%s, want %s)", ErrIntegrityTampered, seq, e.PrevReceiptHash, expectedPrev)
		}
		expectedHash := computeLedgerEntryHash(e)
		if subtle.ConstantTimeCompare([]byte(strings.ToLower(e.EntrySHA256)), []byte(expectedHash)) != 1 {
			return nil, fmt.Errorf("%w: entry_sha256 mismatch at sequence %d (got %s, want %s)", ErrIntegrityTampered, seq, e.EntrySHA256, expectedHash)
		}
		expectedPrev = expectedHash
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%w: scan flight recorder ledger: %v", ErrIntegrityTampered, err)
	}
	return entries, nil
}

// VerifyCheckpointDir verifies both latest_sync_receipt.json.sha256 and every hash-chained line
// in sync_flight_recorder.jsonl, returning the verified latest receipt, total ledger entry count, or ErrIntegrityTampered.
func VerifyCheckpointDir(dir string) (*FeedSyncReceipt, int, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, 0, errors.New("checkpoint directory is required")
	}
	cleanDir := filepath.Clean(dir)

	var rcpt *FeedSyncReceipt
	var count int
	err := resilience.WithDirLock(cleanDir, false, func() error {
		loadedRcpt, rcptHex, loadErr := loadVerifiedCheckpointWithDigest(cleanDir)
		if loadErr != nil {
			return loadErr
		}

		ledgerPath := filepath.Join(cleanDir, "sync_flight_recorder.jsonl")
		entries, verifyErr := readAndVerifyLedger(ledgerPath)
		if verifyErr != nil {
			return verifyErr
		}
		if len(entries) == 0 {
			return fmt.Errorf("%w: sync_flight_recorder.jsonl is empty", ErrIntegrityTampered)
		}

		tip := entries[len(entries)-1]
		if subtle.ConstantTimeCompare([]byte(strings.ToLower(tip.ReceiptSHA256)), []byte(rcptHex)) != 1 || tip.ReceiptID != loadedRcpt.ReceiptID {
			return fmt.Errorf("%w: latest_sync_receipt.json (%s) does not match flight recorder tip (%s)", ErrIntegrityTampered, loadedRcpt.ReceiptID, tip.ReceiptID)
		}

		rcpt = loadedRcpt
		count = len(entries)
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return rcpt, count, nil
}

func ontologySpecDigest() string {
	return ontology.CanonicalSpec().SHA256
}

func estimateFeedGraphContributionCount(feedID string, recordCount int) (nodes int, edges int) {
	switch feedID {
	case FeedCrosswalk:
		if recordCount <= 0 {
			recordCount = len(BuiltinCrosswalks)
		}
		return recordCount * 6, recordCount * 14
	case FeedCISAKEV:
		if recordCount <= 0 {
			recordCount = len(BuiltinKEVAdvisories)
		}
		return recordCount*2 + 9, recordCount * 8
	case FeedMITREAttack:
		if recordCount <= 0 {
			recordCount = len(BuiltinAttackTechniques)
		}
		return recordCount, recordCount * 6
	case FeedSovereignLifecycle:
		return 6, len(BuiltinLifecycleEdges) * 2
	default:
		return 0, 0
	}
}
