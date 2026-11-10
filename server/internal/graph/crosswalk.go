package graph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	_ "github.com/mchorfa/xoscal/server/internal/canadianframeworks"
	"github.com/mchorfa/xoscal/server/internal/feeds"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
	"github.com/mchorfa/xoscal/server/internal/ontology"
	"github.com/mchorfa/xoscal/server/internal/vectorstate"
)

// CrosswalkMatch represents an equivalent or harmonized control across frameworks and jurisdictions.
type CrosswalkMatch struct {
	URN            string  `json:"urn"`
	FrameworkID    string  `json:"framework_id"`
	Jurisdiction   string  `json:"jurisdiction"`
	ControlID      string  `json:"control_id"`
	Title          string  `json:"title"`
	ThematicDomain string  `json:"thematic_domain"`
	Relation       string  `json:"relation"` // EQUIVALENT_TO or HARMONIZES_WITH
	Confidence     float64 `json:"confidence"`
}

// ThreatLink represents an active threat or weakness linked to a control in a crosswalk or blast-radius query.
type ThreatLink struct {
	URN         string   `json:"urn"`
	Class       string   `json:"class"` // threat:KEVAdvisory, threat:CVE, threat:CWE, threat:AttackTechnique
	ID          string   `json:"id"`    // e.g. CVE-2024-3400, CWE-77, T1190
	Title       string   `json:"title"`
	Relation    string   `json:"relation"` // ATTACKS_CONTROL, MITIGATES_CWE, MITIGATES_TECHNIQUE
	IsAnti      bool     `json:"is_anti"`
	Vendor      string   `json:"vendor,omitempty"`
	Product     string   `json:"product,omitempty"`
	RelatedCWE  string   `json:"related_cwe,omitempty"`
	Techniques  []string `json:"techniques,omitempty"`
	Description string   `json:"description,omitempty"`
}

// CrosswalkReport is the result of a multi-jurisdiction control crosswalk and threat posture query.
type CrosswalkReport struct {
	Query                string                  `json:"query"`
	CanonicalControlID   string                  `json:"canonical_control_id"`
	ThematicDomain       string                  `json:"thematic_domain"`
	PrimaryControl       *CrosswalkMatch         `json:"primary_control,omitempty"`
	EquivalentControls   []CrosswalkMatch        `json:"equivalent_controls"`
	HarmonizedControls   []CrosswalkMatch        `json:"harmonized_controls"`
	JurisdictionCoverage map[string]int          `json:"jurisdiction_coverage"`
	ActiveThreats        []ThreatLink            `json:"active_threats"`
	VectorPosture        vectorstate.VectorState `json:"vector_posture"`
}

const maxBlastRadiusNodes = 5000

// BlastRadiusReport captures the multi-hop impact subgraph and 6D Vector State for a target node.
type BlastRadiusReport struct {
	TargetQuery         string                  `json:"target_query"`
	RootNode            *Node                   `json:"root_node"`
	MaxDepth            int                     `json:"max_depth"`
	Truncated           bool                    `json:"truncated,omitempty"`
	ImpactedNodes       []*Node                 `json:"impacted_nodes"`
	ImpactedEdges       []*Edge                 `json:"impacted_edges"`
	ImpactedFrameworks  []string                `json:"impacted_frameworks"`
	ImpactedControls    []string                `json:"impacted_controls"`
	MitigatingControls  []string                `json:"mitigating_controls"`
	JurisdictionImpact  map[string]int          `json:"jurisdiction_impact"`
	ClassBreakdown      map[string]int          `json:"class_breakdown"`
	RelationBreakdown   map[string]int          `json:"relation_breakdown"`
	AntiEdgeCount       int                     `json:"anti_edge_count"`
	MitigationEdgeCount int                     `json:"mitigation_edge_count"`
	VectorPosture       vectorstate.VectorState `json:"vector_posture"`
}

// GraphReconcileReport captures the Day-2 invariant and coherence verification verdict for UnifiedGraph (CKODEX Rule 34).
type GraphReconcileReport struct {
	Coherent      bool                    `json:"coherent"`
	CheckedNodes  int                     `json:"checked_nodes"`
	CheckedEdges  int                     `json:"checked_edges"`
	Violations    []string                `json:"violations,omitempty"`
	GraphSHA256   string                  `json:"graph_sha256"`
	VectorPosture vectorstate.VectorState `json:"vector_posture"`
}

// GraphStats summarizes the node, edge, jurisdiction, and ontology metrics of the Unified Knowledge Graph.
type GraphStats struct {
	OntologyVersion    string         `json:"ontology_version"`
	OntologyNamespace  string         `json:"ontology_namespace"`
	OntologySHA256     string         `json:"ontology_sha256"`
	TotalNodes         int            `json:"total_nodes"`
	TotalEdges         int            `json:"total_edges"`
	AntiEdgeCount      int            `json:"anti_edge_count"`
	FrameworksCount    int            `json:"frameworks_count"`
	FeedsCount         int            `json:"feeds_count"`
	NodesByClass       map[string]int `json:"nodes_by_class"`
	EdgesByPredicate   map[string]int `json:"edges_by_predicate"`
	JurisdictionCounts map[string]int `json:"jurisdiction_counts"`
	GraphSHA256        string         `json:"graph_sha256"`
}

// UnifiedGraph holds the in-memory indexed property graph of all 36 frameworks and 4 intelligence feeds.
type UnifiedGraph struct {
	Nodes         []*Node
	Edges         []*Edge
	nodeByURN     map[string]*Node
	edgesFrom     map[string][]*Edge
	edgesTo       map[string][]*Edge
	ctrlByCanon   map[string][]*Node
	canonAlias    map[string]string
	knownFw       map[string]string
	domainCtrls   map[string][]*Node
	expectedFwIDs []string
	loadErrors    []string
	stats         *GraphStats
}

var (
	defaultUnifiedMu   sync.RWMutex
	defaultUnifiedInst *UnifiedGraph
)

// DefaultUnifiedGraph returns the process-wide cached UnifiedGraph built from DefaultRegistry and the 4 built-in feeds.
// Transient build errors are never permanently cached, allowing subsequent calls to self-heal.
func DefaultUnifiedGraph() (*UnifiedGraph, error) {
	defaultUnifiedMu.RLock()
	inst := defaultUnifiedInst
	defaultUnifiedMu.RUnlock()
	if inst != nil {
		return inst, nil
	}

	defaultUnifiedMu.Lock()
	defer defaultUnifiedMu.Unlock()
	if defaultUnifiedInst != nil {
		return defaultUnifiedInst, nil
	}
	ug, err := BuildUnifiedKnowledgeGraph(context.Background(), nil, frameworks.DefaultRegistry())
	if err != nil {
		return nil, err
	}
	defaultUnifiedInst = ug
	return ug, nil
}

// ReloadDefaultUnifiedGraph rebuilds the UnifiedGraph from DefaultRegistry and atomically swaps the cached instance.
func ReloadDefaultUnifiedGraph(ctx context.Context) (*UnifiedGraph, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ug, err := BuildUnifiedKnowledgeGraph(ctx, nil, frameworks.DefaultRegistry())
	if err != nil {
		return nil, err
	}
	defaultUnifiedMu.Lock()
	defaultUnifiedInst = ug
	defaultUnifiedMu.Unlock()
	return ug, nil
}

func normalizeJurISO(j string) string {
	switch strings.ToLower(strings.TrimSpace(j)) {
	case "ca", "can", "canada":
		return "ca"
	case "us", "usa", "united-states", "united states":
		return "us"
	case "uk", "gb", "gbr", "united-kingdom", "united kingdom":
		return "gb"
	case "fr", "fra", "france":
		return "fr"
	case "de", "deu", "germany":
		return "de"
	case "eu", "european-union", "european union":
		return "eu"
	default:
		return strings.ToLower(strings.TrimSpace(j))
	}
}

func normalizeAuthoritySlug(auth string) string {
	low := strings.ToLower(strings.TrimSpace(auth))
	switch {
	case strings.Contains(low, "canadian centre") || strings.Contains(low, "cccs") || strings.Contains(low, "cse"):
		return "cccs-cse"
	case strings.Contains(low, "ised"):
		return "ised-ca"
	case strings.Contains(low, "nist"):
		return "nist"
	case strings.Contains(low, "cisa"):
		return "cisa"
	case strings.Contains(low, "dod"):
		return "us-dod"
	case strings.Contains(low, "general services") || strings.Contains(low, "gsa"):
		return "us-gsa"
	case strings.Contains(low, "cjis"):
		return "us-cjis"
	case strings.Contains(low, "new york"):
		return "ny-dfs"
	case strings.Contains(low, "ncsc"):
		return "ncsc-uk"
	case strings.Contains(low, "anssi"):
		return "anssi"
	case strings.Contains(low, "cnil"):
		return "cnil"
	case strings.Contains(low, "aviation"):
		return "dsac-fr"
	case strings.Contains(low, "bsi"):
		return "bsi"
	case strings.Contains(low, "ria"):
		return "ria"
	case strings.Contains(low, "enisa") || strings.Contains(low, "european commission"):
		return "enisa"
	case strings.Contains(low, "iso"):
		return "iso"
	case low == "eu":
		return "eu-commission"
	default:
		s := strings.ReplaceAll(low, " ", "-")
		s = strings.ReplaceAll(s, "/", "-")
		if s == "" {
			return "authority-default"
		}
		return s
	}
}

// normalizeSP800171ID converts padded 800-171 control IDs like "03.01.01" into short form "3.1.1".
func normalizeSP800171ID(id string) string {
	parts := strings.Split(strings.TrimSpace(id), ".")
	if len(parts) != 3 {
		return strings.ToLower(strings.TrimSpace(id))
	}
	for i, p := range parts {
		trimmed := strings.TrimLeft(p, "0")
		if trimmed == "" {
			trimmed = "0"
		}
		parts[i] = trimmed
	}
	return strings.ToLower(strings.Join(parts, "."))
}

// BuildUnifiedKnowledgeGraph ingests all registered frameworks in reg (or DefaultRegistry if nil)
// together with the 4 authoritative feeds into a property graph, optionally persisting to store via BulkUpsertGraph.
func BuildUnifiedKnowledgeGraph(ctx context.Context, store Store, reg *frameworks.Registry) (*UnifiedGraph, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	if reg == nil {
		reg = frameworks.DefaultRegistry()
	}

	now := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	ug := &UnifiedGraph{
		nodeByURN:   make(map[string]*Node, 20000),
		edgesFrom:   make(map[string][]*Edge, 20000),
		edgesTo:     make(map[string][]*Edge, 20000),
		ctrlByCanon: make(map[string][]*Node, 2048),
		canonAlias:  make(map[string]string, 1024),
		knownFw:     make(map[string]string, 128),
		domainCtrls: make(map[string][]*Node, 16),
	}
	edgeSeen := make(map[string]bool, 60000)

	addNode := func(urn, kind string, labels map[string]any) *Node {
		if existing, ok := ug.nodeByURN[urn]; ok {
			if len(labels) > 0 {
				var cur map[string]any
				_ = json.Unmarshal([]byte(existing.LabelsJSON), &cur)
				if cur == nil {
					cur = make(map[string]any, len(labels))
				}
				changed := false
				for k, v := range labels {
					if curVal, exists := cur[k]; !exists || curVal == "" || curVal == nil {
						cur[k] = v
						changed = true
					}
				}
				if _, hasTitle := labels["title"]; hasTitle && existing.Kind == ontology.ClassControl && kind == ontology.ClassControlEnhancement {
					existing.Kind = kind
					changed = true
				}
				if changed {
					if lb, err := json.Marshal(cur); err == nil {
						existing.LabelsJSON = string(lb)
					}
				}
			}
			return existing
		}
		lb, _ := json.Marshal(labels)
		n := &Node{
			ID:         urn,
			Kind:       kind,
			URN:        urn,
			LabelsJSON: string(lb),
			CreatedAt:  now,
		}
		ug.nodeByURN[urn] = n
		ug.Nodes = append(ug.Nodes, n)
		return n
	}

	addEdge := func(fromURN, pred, toURN, qualifier string, weight float64) {
		if fromURN == "" || toURN == "" || fromURN == toURN {
			return
		}
		if _, ok := ug.nodeByURN[fromURN]; !ok {
			return
		}
		if _, ok := ug.nodeByURN[toURN]; !ok {
			return
		}
		edgeID := fmt.Sprintf("edge:%s:%s:%s", fromURN, pred, toURN)
		if edgeSeen[edgeID] {
			return
		}
		edgeSeen[edgeID] = true
		if weight <= 0 {
			weight = 1.0
		}
		ext := "{}"
		if ontology.IsAntiPredicate(pred) {
			ext = `{"is_anti":true,"rule":"CKODEX-Rule-17"}`
		}
		e := &Edge{
			ID:             edgeID,
			FromNode:       fromURN,
			ToNode:         toURN,
			Relation:       pred,
			Qualifier:      qualifier,
			ClaimID:        "claim:" + edgeID,
			EvidenceDigest: "sha256:ontology-v1",
			ProofStateJSON: `{"status":"verified"}`,
			ValidFrom:      now,
			TrustState:     "verified",
			Weight:         weight,
			ExtensionsJSON: ext,
		}
		ug.Edges = append(ug.Edges, e)
		ug.edgesFrom[fromURN] = append(ug.edgesFrom[fromURN], e)
		ug.edgesTo[toURN] = append(ug.edgesTo[toURN], e)
	}

	// 1. Seed canonical Jurisdictions
	jurisdictionNames := map[string]string{
		"ca": "Canada",
		"us": "United States",
		"eu": "European Union",
		"de": "Germany",
		"fr": "France",
		"gb": "United Kingdom",
	}
	for code, name := range jurisdictionNames {
		addNode("urn:xoscal:jurisdiction:"+code, ontology.ClassJurisdiction, map[string]any{
			"code":  code,
			"title": name,
		})
	}

	// 2. Seed canonical 12 Thematic Domains + 20 NIST Control Families
	for _, dom := range ontology.ThematicDomains {
		addNode("urn:xoscal:family:"+dom, ontology.ClassControlFamily, map[string]any{
			"id":    dom,
			"title": strings.ReplaceAll(dom, "_", " "),
			"kind":  "thematic_domain",
		})
	}
	nistFamilies := map[string]string{
		"AC": "Access Control",
		"AT": "Awareness and Training",
		"AU": "Audit and Accountability",
		"CA": "Assessment, Authorization, and Monitoring",
		"CM": "Configuration Management",
		"CP": "Contingency Planning",
		"IA": "Identification and Authentication",
		"IR": "Incident Response",
		"MA": "Maintenance",
		"MP": "Media Protection",
		"PE": "Physical and Environmental Protection",
		"PL": "Planning",
		"PM": "Program Management",
		"PS": "Personnel Security",
		"PT": "PII Processing and Transparency",
		"RA": "Risk Assessment",
		"SA": "System and Services Acquisition",
		"SC": "System and Communications Protection",
		"SI": "System and Information Integrity",
		"SR": "Supply Chain Risk Management",
	}
	for fid, ftitle := range nistFamilies {
		addNode("urn:xoscal:family:"+fid, ontology.ClassControlFamily, map[string]any{
			"id":    fid,
			"title": ftitle,
			"kind":  "nist_family",
		})
	}

	// Track representative controls per (framework, thematic_domain) for cross-jurisdiction harmonization
	repByFwDomain := make(map[string]map[string]*Node)
	// Index CMMC 2.0 controls by their trailing SP 800-171 section number (e.g. "3.1.1" -> ac.l1-3.1.1)
	cmmcBy171 := make(map[string][]*Node)

	// 3. Ingest all registered frameworks and their controls, enhancements, profiles, and ODPs
	fwList := reg.List()
	for _, fw := range fwList {
		fwID := fw.ID()
		ug.knownFw[strings.ToLower(fwID)] = fwID
		for _, al := range fw.Aliases() {
			ug.knownFw[strings.ToLower(al)] = fwID
		}
		jurISO := normalizeJurISO(fw.Jurisdiction())
		jurURN := "urn:xoscal:jurisdiction:" + jurISO
		addNode(jurURN, ontology.ClassJurisdiction, map[string]any{"code": jurISO})

		authSlug := normalizeAuthoritySlug(fw.Authority())
		authURN := "urn:xoscal:authority:" + authSlug
		addNode(authURN, ontology.ClassAuthority, map[string]any{
			"slug":  authSlug,
			"title": fw.Authority(),
		})

		fwURN := "urn:xoscal:framework:" + fwID
		addNode(fwURN, ontology.ClassFramework, map[string]any{
			"id":            fwID,
			"title":         fw.Title(),
			"version":       fw.Version(),
			"jurisdiction":  jurISO,
			"authority":     fw.Authority(),
			"control_count": fw.ControlCount(),
		})
		addEdge(fwURN, ontology.PredInJurisdiction, jurURN, "sovereign", 1.0)
		addEdge(fwURN, ontology.PredGovernedByAuthority, authURN, "mandated", 1.0)

		var profileURN string
		if fw.HasProfile() {
			profileURN = "urn:xoscal:profile:" + fwID
			addNode(profileURN, ontology.ClassProfile, map[string]any{
				"id":           fwID,
				"title":        fw.Title() + " Baseline Profile",
				"framework_id": fwID,
				"jurisdiction": jurISO,
			})
			addEdge(profileURN, ontology.PredBelongsToFramework, fwURN, "profile", 1.0)
			addEdge(profileURN, ontology.PredInJurisdiction, jurURN, "sovereign", 1.0)
		}

		repByFwDomain[fwID] = make(map[string]*Node)
		ug.expectedFwIDs = append(ug.expectedFwIDs, fwID)

		ctrls, err := frameworks.LoadControls(fw)
		if err != nil {
			return nil, fmt.Errorf("load controls for framework %s (fail-closed): %w", fwID, err)
		}
		if len(ctrls) == 0 {
			return nil, fmt.Errorf("framework %s returned 0 controls (fail-closed)", fwID)
		}

		for _, c := range ctrls {
			cid := strings.ToLower(strings.TrimSpace(c.ID))
			if cid == "" {
				continue
			}
			ctrlURN := fmt.Sprintf("urn:xoscal:requirement:%s:%s", fwID, cid)
			nodeClass := ontology.ClassControl
			if c.IsEnhancement {
				nodeClass = ontology.ClassControlEnhancement
			}
			stmtText := strings.Join(c.Statements, " ")
			domain := ontology.ClassifyThematicDomain(cid, c.FamilyID, c.Title, stmtText)

			ctrlNode := addNode(ctrlURN, nodeClass, map[string]any{
				"id":              cid,
				"framework_id":    fwID,
				"jurisdiction":    jurISO,
				"title":           c.Title,
				"family_id":       c.FamilyID,
				"thematic_domain": domain,
				"is_canadian":     c.IsCanadian,
				"is_enhancement":  c.IsEnhancement,
			})

			addEdge(ctrlURN, ontology.PredBelongsToFramework, fwURN, "member", 1.0)

			domURN := "urn:xoscal:family:" + domain
			addEdge(ctrlURN, ontology.PredInControlFamily, domURN, "thematic", 1.0)

			if c.FamilyID != "" {
				famCode := strings.ToUpper(strings.TrimSpace(c.FamilyID))
				famURN := "urn:xoscal:family:" + famCode
				addNode(famURN, ontology.ClassControlFamily, map[string]any{
					"id":    famCode,
					"title": c.FamilyTitle,
				})
				addEdge(ctrlURN, ontology.PredInControlFamily, famURN, "family", 1.0)
			}

			if c.IsEnhancement && c.ParentControlID != "" {
				parentURN := fmt.Sprintf("urn:xoscal:requirement:%s:%s", fwID, strings.ToLower(c.ParentControlID))
				addNode(parentURN, ontology.ClassControl, map[string]any{
					"id":           strings.ToLower(c.ParentControlID),
					"framework_id": fwID,
					"jurisdiction": jurISO,
				})
				addEdge(ctrlURN, ontology.PredEnhancesControl, parentURN, "enhancement", 1.0)
			}

			if profileURN != "" {
				addEdge(ctrlURN, ontology.PredSelectedByProfile, profileURN, "selected", 1.0)
			}

			for paramID, paramVal := range c.Parameters {
				if paramID == "" {
					continue
				}
				odpURN := fmt.Sprintf("urn:xoscal:odp:%s:%s", fwID, strings.ToLower(paramID))
				addNode(odpURN, ontology.ClassParameterODP, map[string]any{
					"id":           paramID,
					"value":        paramVal,
					"framework_id": fwID,
					"control_id":   cid,
				})
				addEdge(ctrlURN, ontology.PredDefinesODP, odpURN, "odp", 1.0)
			}

			ug.ctrlByCanon[cid] = append(ug.ctrlByCanon[cid], ctrlNode)
			if fwID == "cmmc-2.0" {
				if dashIdx := strings.LastIndex(cid, "-"); dashIdx != -1 && dashIdx+1 < len(cid) {
					sec171 := normalizeSP800171ID(cid[dashIdx+1:])
					cmmcBy171[sec171] = append(cmmcBy171[sec171], ctrlNode)
				}
			}
			if !c.IsEnhancement {
				if _, exists := repByFwDomain[fwID][domain]; !exists {
					repByFwDomain[fwID][domain] = ctrlNode
					ug.domainCtrls[domain] = append(ug.domainCtrls[domain], ctrlNode)
				}
			}
		}
	}

	// 4. Synthesize EQUIVALENT_TO edges across frameworks sharing canonical control IDs (e.g. ac-2 across ITSP.10.033, ITSG-33, PBMM, FedRAMP)
	for cid, nodes := range ug.ctrlByCanon {
		if len(nodes) < 2 {
			continue
		}
		// Pick primary node (prefer cccs-itsp-10-033 or first node) and connect all peers bidirectionally
		primaryIdx := 0
		for i, n := range nodes {
			if strings.Contains(n.URN, ":cccs-itsp-10-033:") {
				primaryIdx = i
				break
			}
		}
		primary := nodes[primaryIdx]
		for i, peer := range nodes {
			if i == primaryIdx {
				continue
			}
			addEdge(primary.URN, ontology.PredEquivalentTo, peer.URN, "exact-control-id:"+cid, 1.0)
			addEdge(peer.URN, ontology.PredEquivalentTo, primary.URN, "exact-control-id:"+cid, 1.0)
		}
	}

	// Ensure crosswalk reference frameworks have framework nodes and aliases
	crosswalkFwDefs := []struct {
		id      string
		jur     string
		auth    string
		title   string
		aliases []string
	}{
		{"nist-sp-800-53-rev5", "us", "nist", "NIST SP 800-53 Rev 5", []string{"nist-800-53", "800-53"}},
		{"nist-sp-800-171-rev2", "us", "nist", "NIST SP 800-171 Rev 2/3", []string{"nist-800-171", "800-171", "sp-800-171"}},
		{"iso-27001-2022", "eu", "iso", "ISO/IEC 27001:2022", []string{"iso-27001", "iso27001"}},
		{"bsi-it-gs-2023", "de", "bsi", "BSI IT-Grundschutz Kompendium 2023", []string{"bsi-it-gs"}},
		{"anssi-hygiene", "fr", "anssi", "ANSSI Guide d'hygiène informatique", []string{"anssi"}},
		{"uk-cyber-essentials", "gb", "ncsc-uk", "UK NCSC Cyber Essentials", []string{"cyber-essentials"}},
	}
	for _, cfd := range crosswalkFwDefs {
		ug.knownFw[cfd.id] = cfd.id
		for _, al := range cfd.aliases {
			ug.knownFw[al] = cfd.id
		}
		fwURN := "urn:xoscal:framework:" + cfd.id
		addNode(fwURN, ontology.ClassFramework, map[string]any{
			"id":           cfd.id,
			"title":        cfd.title,
			"jurisdiction": cfd.jur,
		})
		addEdge(fwURN, ontology.PredInJurisdiction, "urn:xoscal:jurisdiction:"+cfd.jur, "sovereign", 1.0)
		addNode("urn:xoscal:authority:"+cfd.auth, ontology.ClassAuthority, map[string]any{"slug": cfd.auth})
		addEdge(fwURN, ontology.PredGovernedByAuthority, "urn:xoscal:authority:"+cfd.auth, "mandated", 1.0)
	}

	// 5. Ingest Feed 1: cccs-nist-eu-crosswalk (EQUIVALENT_TO and HARMONIZES_WITH across CA, US, EU, DE, FR, GB)
	for _, cw := range feeds.BuiltinCrosswalks {
		canonID := strings.ToLower(cw.CanonicalControl)
		ug.canonAlias[canonID] = canonID
		primaryURN := fmt.Sprintf("urn:xoscal:requirement:cccs-itsp-10-033:%s", canonID)
		addNode(primaryURN, ontology.ClassControl, map[string]any{
			"id":              canonID,
			"framework_id":    "cccs-itsp-10-033",
			"jurisdiction":    "ca",
			"title":           cw.Title,
			"thematic_domain": cw.ThematicDomain,
		})

		// Connect NIST SP 800-53 Rev 5 control via EQUIVALENT_TO
		nist53URN := fmt.Sprintf("urn:xoscal:requirement:nist-sp-800-53-rev5:%s", canonID)
		n53 := addNode(nist53URN, ontology.ClassControl, map[string]any{
			"id":              canonID,
			"framework_id":    "nist-sp-800-53-rev5",
			"jurisdiction":    "us",
			"title":           cw.Title + " (NIST SP 800-53 " + strings.ToUpper(canonID) + ")",
			"thematic_domain": cw.ThematicDomain,
		})
		addEdge(nist53URN, ontology.PredBelongsToFramework, "urn:xoscal:framework:nist-sp-800-53-rev5", "member", 1.0)
		ug.ctrlByCanon[canonID] = appendUniqueNode(ug.ctrlByCanon[canonID], n53)
		addEdge(primaryURN, ontology.PredEquivalentTo, nist53URN, "nist-800-53-crosswalk", 1.0)
		addEdge(nist53URN, ontology.PredEquivalentTo, primaryURN, "nist-800-53-crosswalk", 1.0)

		// Connect NIST SP 800-171 and CMMC 2.0 controls via EQUIVALENT_TO
		for _, s171 := range cw.SP800171Controls {
			raw171 := strings.ToLower(strings.TrimSpace(s171))
			short171 := normalizeSP800171ID(raw171)
			ug.canonAlias[raw171] = canonID
			ug.canonAlias[short171] = canonID

			for _, idForm := range []string{short171, raw171} {
				u171 := fmt.Sprintf("urn:xoscal:requirement:nist-sp-800-171-rev2:%s", idForm)
				n171 := addNode(u171, ontology.ClassControl, map[string]any{
					"id":              idForm,
					"framework_id":    "nist-sp-800-171-rev2",
					"jurisdiction":    "us",
					"title":           cw.Title + " (NIST SP 800-171 " + strings.ToUpper(idForm) + ")",
					"thematic_domain": cw.ThematicDomain,
				})
				addEdge(u171, ontology.PredBelongsToFramework, "urn:xoscal:framework:nist-sp-800-171-rev2", "member", 1.0)
				ug.ctrlByCanon[canonID] = appendUniqueNode(ug.ctrlByCanon[canonID], n171)
				ug.ctrlByCanon[idForm] = appendUniqueNode(ug.ctrlByCanon[idForm], n171)
				addEdge(primaryURN, ontology.PredEquivalentTo, u171, "nist-800-171-crosswalk", 1.0)
				addEdge(u171, ontology.PredEquivalentTo, primaryURN, "nist-800-171-crosswalk", 1.0)
			}

			for _, cmmcNode := range cmmcBy171[short171] {
				ug.ctrlByCanon[canonID] = appendUniqueNode(ug.ctrlByCanon[canonID], cmmcNode)
				var lbl map[string]any
				_ = json.Unmarshal([]byte(cmmcNode.LabelsJSON), &lbl)
				if cid, ok := lbl["id"].(string); ok && cid != "" {
					ug.canonAlias[cid] = canonID
				}
				addEdge(primaryURN, ontology.PredEquivalentTo, cmmcNode.URN, "cmmc-800-171-crosswalk", 1.0)
				addEdge(cmmcNode.URN, ontology.PredEquivalentTo, primaryURN, "cmmc-800-171-crosswalk", 1.0)
			}
		}

		// Connect ITSP.10.171 controls via EQUIVALENT_TO
		for _, c171 := range cw.ITSP10171Controls {
			low171 := strings.ToLower(c171)
			ug.canonAlias[low171] = canonID
			ug.canonAlias[strings.TrimPrefix(low171, "_")] = canonID
			u := fmt.Sprintf("urn:xoscal:requirement:cccs-itsp-10-171:%s", low171)
			n := addNode(u, ontology.ClassControl, map[string]any{
				"id":              low171,
				"framework_id":    "cccs-itsp-10-171",
				"jurisdiction":    "ca",
				"title":           cw.Title + " (ITSP.10.171 " + strings.TrimPrefix(strings.ToUpper(c171), "_") + ")",
				"thematic_domain": cw.ThematicDomain,
			})
			addEdge(u, ontology.PredBelongsToFramework, "urn:xoscal:framework:cccs-itsp-10-171", "member", 1.0)
			ug.ctrlByCanon[canonID] = appendUniqueNode(ug.ctrlByCanon[canonID], n)
			addEdge(primaryURN, ontology.PredEquivalentTo, u, "nist-800-171-crosswalk", 1.0)
			addEdge(u, ontology.PredEquivalentTo, primaryURN, "nist-800-171-crosswalk", 1.0)
		}

		// Connect CyberSecure Canada controls via HARMONIZES_WITH
		for _, csc := range cw.CyberSecureCanada {
			u := fmt.Sprintf("urn:xoscal:requirement:cybersecure-canada:%s", strings.ToLower(csc))
			addNode(u, ontology.ClassControl, map[string]any{
				"id":              strings.ToLower(csc),
				"framework_id":    "cybersecure-canada",
				"jurisdiction":    "ca",
				"title":           cw.Title + " (CyberSecure Canada " + strings.ToUpper(csc) + ")",
				"thematic_domain": cw.ThematicDomain,
			})
			addEdge(u, ontology.PredBelongsToFramework, "urn:xoscal:framework:cybersecure-canada", "member", 1.0)
			addEdge(primaryURN, ontology.PredHarmonizesWith, u, "ca-baseline-harmonization", 0.9)
			addEdge(u, ontology.PredHarmonizesWith, primaryURN, "ca-baseline-harmonization", 0.9)
		}

		// Connect Multi-Jurisdiction Harmonization targets (US, EU, DE, FR, GB)
		type intlTarget struct {
			fwID  string
			jur   string
			codes []string
		}
		intlTargets := []intlTarget{
			{"nist-csf-2.0", "us", cw.NISTCSF2Subcats},
			{"iso-27001-2022", "eu", cw.ISO27001Controls},
			{"iso-iec-42001", "eu", cw.ISO27001Controls},
			{"nis2", "eu", cw.NIS2Articles},
			{"dora", "eu", cw.DORAPillars},
			{"bsi-it-gs-2023", "de", cw.BSIModules},
			{"bs-it-gs-2023-ops", "de", cw.BSIModules},
			{"anssi-hygiene", "fr", cw.ANSSIRules},
			{"anssi-guide-hygiene", "fr", cw.ANSSIRules},
			{"uk-cyber-essentials", "gb", cw.UKCyberEssentials},
			{"cyber_essentials", "gb", cw.UKCyberEssentials},
		}
		for _, tgt := range intlTargets {
			for _, code := range tgt.codes {
				normCode := strings.ToLower(code)
				tURN := fmt.Sprintf("urn:xoscal:requirement:%s:%s", tgt.fwID, normCode)
				addNode(tURN, ontology.ClassControl, map[string]any{
					"id":              normCode,
					"framework_id":    tgt.fwID,
					"jurisdiction":    tgt.jur,
					"title":           fmt.Sprintf("%s (%s %s)", cw.Title, strings.ToUpper(tgt.fwID), code),
					"thematic_domain": cw.ThematicDomain,
				})
				addEdge(tURN, ontology.PredBelongsToFramework, "urn:xoscal:framework:"+tgt.fwID, "member", 1.0)
				addEdge(primaryURN, ontology.PredHarmonizesWith, tURN, "multi-jurisdiction-crosswalk", 0.88)
				addEdge(tURN, ontology.PredHarmonizesWith, primaryURN, "multi-jurisdiction-crosswalk", 0.88)
			}
			// Also link to real loaded representative control in that framework & thematic domain if present
			if domMap, ok := repByFwDomain[tgt.fwID]; ok {
				if repNode, ok := domMap[cw.ThematicDomain]; ok {
					addEdge(primaryURN, ontology.PredHarmonizesWith, repNode.URN, "thematic-domain:"+cw.ThematicDomain, 0.85)
					addEdge(repNode.URN, ontology.PredHarmonizesWith, primaryURN, "thematic-domain:"+cw.ThematicDomain, 0.85)
				}
			}
		}
	}

	// Also connect thematic domain representatives across all 36 frameworks
	for dom, domNodes := range ug.domainCtrls {
		if len(domNodes) < 2 {
			continue
		}
		hub := domNodes[0]
		for _, n := range domNodes {
			if strings.Contains(n.URN, ":cccs-itsp-10-033:") {
				hub = n
				break
			}
		}
		for _, peer := range domNodes {
			if peer.URN == hub.URN {
				continue
			}
			addEdge(hub.URN, ontology.PredHarmonizesWith, peer.URN, "domain:"+dom, 0.80)
			addEdge(peer.URN, ontology.PredHarmonizesWith, hub.URN, "domain:"+dom, 0.80)
		}
	}

	// 6. Ingest Feed 2: cisa-kev-active-threats (threat:KEVAdvisory, threat:CVE, threat:CWE, ATTACKS_CONTROL, MITIGATES_CWE)
	for _, kev := range feeds.ActiveThreatAdvisories() {
		kevURN := "urn:xoscal:kev:" + kev.CVEID
		cveURN := "urn:xoscal:cve:" + kev.CVEID
		cweURN := "urn:xoscal:cwe:" + kev.CWEID

		addNode(kevURN, ontology.ClassKEVAdvisory, map[string]any{
			"id":                kev.CVEID,
			"title":             kev.VulnerabilityName,
			"vendor":            kev.Vendor,
			"product":           kev.Product,
			"date_added":        kev.DateAdded,
			"due_date":          kev.DueDate,
			"ransomware_use":    kev.RansomwareUse,
			"cwe_id":            kev.CWEID,
			"attack_techniques": kev.AttackTechniques,
			"short_description": kev.ShortDescription,
		})
		addNode(cveURN, ontology.ClassCVE, map[string]any{
			"id":          kev.CVEID,
			"title":       kev.VulnerabilityName,
			"vendor":      kev.Vendor,
			"product":     kev.Product,
			"cwe_id":      kev.CWEID,
			"description": kev.ShortDescription,
		})
		addNode(cweURN, ontology.ClassCWE, map[string]any{
			"id":    kev.CWEID,
			"title": kev.CWEName,
		})

		addEdge(kevURN, ontology.PredExploitsCWE, cweURN, "cisa-kev", 1.0)
		addEdge(cveURN, ontology.PredExploitsCWE, cweURN, "nvd-cwe", 1.0)

		for _, techID := range kev.AttackTechniques {
			techURN := "urn:xoscal:attack:" + techID
			addNode(techURN, ontology.ClassAttackTechnique, map[string]any{
				"id":    techID,
				"title": "MITRE ATT&CK " + techID,
			})
			addEdge(kevURN, ontology.PredMapsToTechnique, techURN, "kev-attack", 1.0)
			addEdge(cveURN, ontology.PredMapsToTechnique, techURN, "cve-attack", 1.0)
			addEdge(cweURN, ontology.PredMapsToTechnique, techURN, "cwe-attack", 1.0)
		}

		for _, attackedCtrl := range kev.AttackedControls {
			cid := strings.ToLower(attackedCtrl)
			for _, fwTarget := range []string{"cccs-itsp-10-033", "cccs-itsg-33", "cccs-medium-cloud-pbmm", "fedramp-rev5"} {
				ctrlURN := fmt.Sprintf("urn:xoscal:requirement:%s:%s", fwTarget, cid)
				if _, exists := ug.nodeByURN[ctrlURN]; exists {
					addEdge(kevURN, ontology.PredAttacksControl, ctrlURN, "active-exploitation", 1.0)
					addEdge(cveURN, ontology.PredAttacksControl, ctrlURN, "cve-impact", 1.0)
				}
			}
		}

		for _, mitCtrl := range kev.MitigatingControls {
			cid := strings.ToLower(mitCtrl)
			for _, fwTarget := range []string{"cccs-itsp-10-033", "cccs-itsg-33", "cccs-medium-cloud-pbmm", "fedramp-rev5"} {
				ctrlURN := fmt.Sprintf("urn:xoscal:requirement:%s:%s", fwTarget, cid)
				if _, exists := ug.nodeByURN[ctrlURN]; exists {
					addEdge(ctrlURN, ontology.PredMitigatesCWE, cweURN, "cwe-mitigation", 1.0)
				}
			}
		}
	}

	// 7. Ingest Feed 3: mitre-attack-800-53 (threat:AttackTechnique, MITIGATES_TECHNIQUE, ATTACKS_CONTROL)
	for _, tech := range feeds.ActiveAttackMappings() {
		techURN := "urn:xoscal:attack:" + tech.TechniqueID
		// Overwrite/enrich labels on technique node
		lb, _ := json.Marshal(map[string]any{
			"id":              tech.TechniqueID,
			"title":           tech.Name,
			"tactic":          tech.Tactic,
			"thematic_domain": tech.ThematicDomain,
			"description":     tech.Description,
		})
		if existing, ok := ug.nodeByURN[techURN]; ok {
			existing.LabelsJSON = string(lb)
		} else {
			addNode(techURN, ontology.ClassAttackTechnique, map[string]any{
				"id":              tech.TechniqueID,
				"title":           tech.Name,
				"tactic":          tech.Tactic,
				"thematic_domain": tech.ThematicDomain,
				"description":     tech.Description,
			})
		}

		for _, cweID := range tech.RelatedCWEs {
			cweURN := "urn:xoscal:cwe:" + cweID
			addNode(cweURN, ontology.ClassCWE, map[string]any{"id": cweID, "title": cweID})
			addEdge(cweURN, ontology.PredMapsToTechnique, techURN, "cwe-technique", 1.0)
		}

		for idx, mitCtrl := range tech.MitigatingControls {
			cid := strings.ToLower(mitCtrl)
			for _, fwTarget := range []string{"cccs-itsp-10-033", "cccs-itsg-33", "cccs-medium-cloud-pbmm", "fedramp-rev5"} {
				ctrlURN := fmt.Sprintf("urn:xoscal:requirement:%s:%s", fwTarget, cid)
				if _, exists := ug.nodeByURN[ctrlURN]; exists {
					addEdge(ctrlURN, ontology.PredMitigatesTechnique, techURN, "ctid-800-53-r5", 1.0)
					if idx == 0 {
						addEdge(techURN, ontology.PredAttacksControl, ctrlURN, "adversary-technique-target", 1.0)
					}
				}
			}
		}
	}

	// 8. Ingest Feed 4: sovereign-regulatory-lifecycle (SUPERSEDES_FRAMEWORK, DERIVED_FROM)
	for _, lc := range feeds.BuiltinLifecycleEdges {
		fromID := lc.FromFramework
		if fw, ok := reg.Get(fromID); ok {
			fromID = fw.ID()
		}
		toID := lc.ToFramework
		if fw, ok := reg.Get(toID); ok {
			toID = fw.ID()
		}
		fromURN := "urn:xoscal:framework:" + fromID
		toURN := "urn:xoscal:framework:" + toID
		addNode(fromURN, ontology.ClassFramework, map[string]any{"id": fromID, "jurisdiction": lc.Jurisdiction})
		addNode(toURN, ontology.ClassFramework, map[string]any{"id": toID, "jurisdiction": lc.Jurisdiction})
		addEdge(fromURN, lc.Predicate, toURN, lc.EffectiveDate, 1.0)
	}

	// 9. Sort nodes, edges, and adjacency lists deterministically (eliminating Go map iteration non-determinism)
	sort.Slice(ug.Nodes, func(i, j int) bool { return ug.Nodes[i].URN < ug.Nodes[j].URN })
	sort.Slice(ug.Edges, func(i, j int) bool { return ug.Edges[i].ID < ug.Edges[j].ID })
	for k := range ug.edgesFrom {
		s := ug.edgesFrom[k]
		sort.Slice(s, func(i, j int) bool { return s[i].ID < s[j].ID })
	}
	for k := range ug.edgesTo {
		s := ug.edgesTo[k]
		sort.Slice(s, func(i, j int) bool { return s[i].ID < s[j].ID })
	}

	// 10. Compute summary statistics & full content-addressed GraphSHA256
	nodesByClass := make(map[string]int)
	jurCounts := make(map[string]int)
	for _, n := range ug.Nodes {
		nodesByClass[n.Kind]++
		var lbl map[string]any
		if err := json.Unmarshal([]byte(n.LabelsJSON), &lbl); err == nil {
			if j, ok := lbl["jurisdiction"].(string); ok && j != "" {
				jurCounts[j]++
			}
		}
	}

	edgesByPred := make(map[string]int)
	antiEdges := 0
	for _, e := range ug.Edges {
		edgesByPred[e.Relation]++
		if ontology.IsAntiPredicate(e.Relation) {
			antiEdges++
		}
	}

	ontSpec := ontology.CanonicalSpec()
	graphHex := computeGraphContentSHA256(ug.Nodes, ug.Edges, ontSpec.SHA256)

	ug.stats = &GraphStats{
		OntologyVersion:    ontology.OntologyVersion,
		OntologyNamespace:  ontology.OntologyNamespace,
		OntologySHA256:     ontSpec.SHA256,
		TotalNodes:         len(ug.Nodes),
		TotalEdges:         len(ug.Edges),
		AntiEdgeCount:      antiEdges,
		FrameworksCount:    len(fwList),
		FeedsCount:         len(feeds.ListFeeds()),
		NodesByClass:       nodesByClass,
		EdgesByPredicate:   edgesByPred,
		JurisdictionCounts: jurCounts,
		GraphSHA256:        graphHex,
	}

	if store != nil {
		if err := store.BulkUpsertGraph(ctx, ug.Nodes, ug.Edges); err != nil {
			return nil, fmt.Errorf("persist unified graph to store: %w", err)
		}
	}

	return ug, nil
}

func computeGraphContentSHA256(nodes []*Node, edges []*Edge, ontologySHA string) string {
	h := sha256.New()
	_, _ = h.Write([]byte("ontology:" + ontologySHA + "\n"))

	sortedNodes := make([]*Node, len(nodes))
	copy(sortedNodes, nodes)
	sort.Slice(sortedNodes, func(i, j int) bool {
		if sortedNodes[i] == nil {
			return true
		}
		if sortedNodes[j] == nil {
			return false
		}
		return sortedNodes[i].URN < sortedNodes[j].URN
	})
	for _, n := range sortedNodes {
		if n == nil {
			_, _ = h.Write([]byte("node:<nil>\n"))
			continue
		}
		_, _ = fmt.Fprintf(h, "N|%s|%s|%s|%s\n", n.ID, n.URN, n.Kind, n.LabelsJSON)
	}

	sortedEdges := make([]*Edge, len(edges))
	copy(sortedEdges, edges)
	sort.Slice(sortedEdges, func(i, j int) bool {
		if sortedEdges[i] == nil {
			return true
		}
		if sortedEdges[j] == nil {
			return false
		}
		return sortedEdges[i].ID < sortedEdges[j].ID
	})
	for _, e := range sortedEdges {
		if e == nil {
			_, _ = h.Write([]byte("edge:<nil>\n"))
			continue
		}
		_, _ = fmt.Fprintf(h, "E|%s|%s|%s|%s|%s|%.4f|%s|%s|%s\n",
			e.ID, e.FromNode, e.Relation, e.ToNode, e.Qualifier, e.Weight,
			e.TrustState, e.EvidenceDigest, e.ExtensionsJSON)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func cloneNode(n *Node) *Node {
	if n == nil {
		return nil
	}
	cp := *n
	return &cp
}

func cloneEdge(e *Edge) *Edge {
	if e == nil {
		return nil
	}
	cp := *e
	if e.ValidTo != nil {
		vt := *e.ValidTo
		cp.ValidTo = &vt
	}
	return &cp
}

func appendUniqueNode(slice []*Node, n *Node) []*Node {
	for _, item := range slice {
		if item.URN == n.URN {
			return slice
		}
	}
	return append(slice, n)
}

// Stats returns a defensive deep copy of the precomputed GraphStats for the UnifiedGraph.
func (ug *UnifiedGraph) Stats() *GraphStats {
	if ug == nil || ug.stats == nil {
		return nil
	}
	cp := *ug.stats
	cp.NodesByClass = make(map[string]int, len(ug.stats.NodesByClass))
	for k, v := range ug.stats.NodesByClass {
		cp.NodesByClass[k] = v
	}
	cp.EdgesByPredicate = make(map[string]int, len(ug.stats.EdgesByPredicate))
	for k, v := range ug.stats.EdgesByPredicate {
		cp.EdgesByPredicate[k] = v
	}
	cp.JurisdictionCounts = make(map[string]int, len(ug.stats.JurisdictionCounts))
	for k, v := range ug.stats.JurisdictionCounts {
		cp.JurisdictionCounts[k] = v
	}
	return &cp
}

// ReconcileDefaultGraph runs Day-2 invariant and coherence verification on the cached DefaultUnifiedGraph.
func ReconcileDefaultGraph() (*GraphReconcileReport, error) {
	ug, err := DefaultUnifiedGraph()
	if err != nil {
		return nil, err
	}
	return ug.ReconcileInvariants()
}

// ReconcileInvariants verifies structural referential integrity, OWL/JSON-LD triple conformance,
// uniqueness, 6-jurisdiction sovereign coverage, and cryptographic SHA-256 bitrot protection (CKODEX Rule 34).
func (ug *UnifiedGraph) ReconcileInvariants() (*GraphReconcileReport, error) {
	if ug == nil {
		return nil, fmt.Errorf("unified graph is nil")
	}

	var violations []string
	seenNodes := make(map[string]*Node, len(ug.Nodes))
	jurCounts := map[string]int{
		"ca": 0,
		"us": 0,
		"eu": 0,
		"de": 0,
		"fr": 0,
		"gb": 0,
	}

	for idx, n := range ug.Nodes {
		if n == nil || strings.TrimSpace(n.URN) == "" {
			violations = append(violations, fmt.Sprintf("nil or empty URN node at index %d", idx))
			continue
		}
		if _, dup := seenNodes[n.URN]; dup {
			violations = append(violations, fmt.Sprintf("duplicate node URN %q", n.URN))
		}
		seenNodes[n.URN] = n

		if indexed, ok := ug.nodeByURN[n.URN]; !ok || indexed != n {
			violations = append(violations, fmt.Sprintf("node %q missing or decoherent in nodeByURN index", n.URN))
		}

		var lbl map[string]any
		if err := json.Unmarshal([]byte(n.LabelsJSON), &lbl); err == nil {
			if j, ok := lbl["jurisdiction"].(string); ok && j != "" {
				jurCounts[j]++
			}
		}
	}

	if len(ug.nodeByURN) != len(seenNodes) {
		violations = append(violations, fmt.Sprintf("nodeByURN index size mismatch: index=%d slice=%d", len(ug.nodeByURN), len(seenNodes)))
	}

	if len(ug.loadErrors) > 0 {
		violations = append(violations, ug.loadErrors...)
	}

	seenEdges := make(map[string]bool, len(ug.Edges))
	fwCtrlCounts := make(map[string]int, len(ug.expectedFwIDs))
	for idx, e := range ug.Edges {
		if e == nil || strings.TrimSpace(e.ID) == "" {
			violations = append(violations, fmt.Sprintf("nil or empty ID edge at index %d", idx))
			continue
		}
		if seenEdges[e.ID] {
			violations = append(violations, fmt.Sprintf("duplicate edge ID %q", e.ID))
		}
		seenEdges[e.ID] = true

		fromNode, okFrom := ug.nodeByURN[e.FromNode]
		if !okFrom || fromNode == nil || seenNodes[e.FromNode] == nil {
			okFrom = false
			violations = append(violations, fmt.Sprintf("dangling edge %q: missing FromNode %q", e.ID, e.FromNode))
		}
		toNode, okTo := ug.nodeByURN[e.ToNode]
		if !okTo || toNode == nil || seenNodes[e.ToNode] == nil {
			okTo = false
			violations = append(violations, fmt.Sprintf("dangling edge %q: missing ToNode %q", e.ID, e.ToNode))
		}
		if okFrom && okTo {
			if err := ontology.ValidateTriple(fromNode.Kind, e.Relation, toNode.Kind); err != nil {
				violations = append(violations, fmt.Sprintf("ontology triple violation on edge %q: %v", e.ID, err))
			}
			if e.Relation == ontology.PredBelongsToFramework &&
				(fromNode.Kind == ontology.ClassControl || fromNode.Kind == ontology.ClassControlEnhancement) {
				fwCtrlCounts[e.ToNode]++
			}
		}
	}

	for _, fwID := range ug.expectedFwIDs {
		fwURN := "urn:xoscal:framework:" + fwID
		if seenNodes[fwURN] == nil {
			violations = append(violations, fmt.Sprintf("missing registered framework node %q", fwURN))
		} else if fwCtrlCounts[fwURN] == 0 {
			violations = append(violations, fmt.Sprintf("registered framework %q has 0 loaded control nodes", fwID))
		}
	}
	if len(ug.expectedFwIDs) > 0 && ug.stats != nil && ug.stats.FrameworksCount < len(ug.expectedFwIDs) {
		violations = append(violations, fmt.Sprintf("frameworks_count (%d) is below registered framework count (%d)", ug.stats.FrameworksCount, len(ug.expectedFwIDs)))
	}

	for _, jur := range []string{"ca", "us", "eu", "de", "fr", "gb"} {
		if jurCounts[jur] <= 0 {
			violations = append(violations, fmt.Sprintf("missing sovereign jurisdiction coverage for %q (0 nodes)", jur))
		}
	}

	ontSHA := ontology.CanonicalSpec().SHA256
	if ug.stats != nil && ug.stats.OntologySHA256 != "" {
		ontSHA = ug.stats.OntologySHA256
	}
	recomputedSHA := computeGraphContentSHA256(ug.Nodes, ug.Edges, ontSHA)
	tampered := false
	if ug.stats == nil || recomputedSHA != ug.stats.GraphSHA256 {
		expected := "<nil>"
		if ug.stats != nil {
			expected = ug.stats.GraphSHA256
		}
		violations = append(violations, fmt.Sprintf("graph SHA-256 bitrot/mutation detected: recomputed=%s stored=%s", recomputedSHA, expected))
		tampered = true
	}

	coherent := len(violations) == 0
	posture := vectorstate.VectorState{
		ArtifactName:  "unified-property-graph",
		ArtifactKind:  "owl:OntologyGraph",
		Presence:      vectorstate.PresencePresent,
		Valence:       vectorstate.ValencePositive,
		AntiCount:     len(violations),
		Coherence:     vectorstate.Coherent,
		Evidence:      vectorstate.EvidenceVerified,
		Lifecycle:     vectorstate.ModeNormal,
		Epoch:         time.Now().UTC(),
		TotalControls: len(ug.Nodes),
		PassingCount:  len(ug.Nodes),
	}
	if !coherent {
		posture.Valence = vectorstate.ValenceNegative
		posture.Coherence = vectorstate.Decoherent
		posture.Lifecycle = vectorstate.ModeQuarantined
		posture.PassingCount = 0
		posture.DegradedCount = len(violations)
		if tampered {
			posture.Evidence = vectorstate.EvidenceTampered
		} else {
			posture.Evidence = vectorstate.EvidenceUnverified
		}
	}

	return &GraphReconcileReport{
		Coherent:      coherent,
		CheckedNodes:  len(ug.Nodes),
		CheckedEdges:  len(ug.Edges),
		Violations:    violations,
		GraphSHA256:   recomputedSHA,
		VectorPosture: posture,
	}, nil
}

// ComputeGraphStats returns the GraphStats for the default cached UnifiedGraph.
func ComputeGraphStats() (*GraphStats, error) {
	ug, err := DefaultUnifiedGraph()
	if err != nil {
		return nil, err
	}
	return ug.Stats(), nil
}

// QueryCrosswalk executes a multi-jurisdiction crosswalk and threat query on the default UnifiedGraph.
func QueryCrosswalk(controlQuery string) (*CrosswalkReport, error) {
	ug, err := DefaultUnifiedGraph()
	if err != nil {
		return nil, err
	}
	return ug.QueryCrosswalk(controlQuery)
}

// QueryCrosswalk resolves equivalent controls, multi-jurisdiction harmonized controls,
// active KEV/CVE/CWE/ATT&CK threats, and 6D Vector State posture for the given control query.
func (ug *UnifiedGraph) QueryCrosswalk(controlQuery string) (*CrosswalkReport, error) {
	q := strings.TrimSpace(controlQuery)
	if q == "" {
		return nil, fmt.Errorf("control query cannot be empty")
	}

	qLower := strings.ToLower(q)
	qLower = strings.TrimPrefix(qLower, "urn:xoscal:requirement:")

	var fwFilter string
	rawCtrlID := qLower
	if parts := strings.SplitN(qLower, ":", 2); len(parts) == 2 {
		fwFilter = strings.TrimSpace(parts[0])
		rawCtrlID = strings.TrimSpace(parts[1])
	}

	// Resolve canonical control ID alias (e.g. "3.1.1" / "03.01.01" / "_03.01.01.a" / "ac.l1-3.1.1" -> "ac-2")
	canonID := rawCtrlID
	if mapped, ok := ug.canonAlias[rawCtrlID]; ok && mapped != "" {
		canonID = mapped
	} else if mapped, ok := ug.canonAlias[normalizeSP800171ID(rawCtrlID)]; ok && mapped != "" {
		canonID = mapped
	}

	// Resolve primary control node
	var primaryNode *Node
	if fwFilter != "" {
		resolvedFw, ok := ug.knownFw[fwFilter]
		if !ok {
			if fw, exists := frameworks.DefaultRegistry().Get(fwFilter); exists {
				resolvedFw = fw.ID()
				ok = true
			}
		}
		if !ok {
			return nil, fmt.Errorf("framework %q not found in unified knowledge graph", fwFilter)
		}
		for _, candidateCtrl := range []string{rawCtrlID, "_" + rawCtrlID, normalizeSP800171ID(rawCtrlID), canonID} {
			targetURN := fmt.Sprintf("urn:xoscal:requirement:%s:%s", resolvedFw, candidateCtrl)
			if n := ug.nodeByURN[targetURN]; n != nil {
				primaryNode = n
				break
			}
		}
		if primaryNode == nil {
			return nil, fmt.Errorf("control %q not found in framework %q", rawCtrlID, resolvedFw)
		}
	}
	if primaryNode == nil {
		candidates := ug.ctrlByCanon[canonID]
		if len(candidates) == 0 && canonID != rawCtrlID {
			candidates = ug.ctrlByCanon[rawCtrlID]
		}
		for _, c := range candidates {
			if strings.Contains(c.URN, ":cccs-itsp-10-033:") {
				primaryNode = c
				break
			}
		}
		if primaryNode == nil && len(candidates) > 0 {
			primaryNode = candidates[0]
		}
	}
	if primaryNode == nil {
		return nil, fmt.Errorf("control %q not found in unified knowledge graph", controlQuery)
	}

	primaryMatch := nodeToCrosswalkMatch(primaryNode, "PRIMARY", 1.0)
	if primaryMatch.ThematicDomain == "" {
		primaryMatch.ThematicDomain = ontology.ClassifyThematicDomain(primaryMatch.ControlID, "", primaryMatch.Title, "")
	}

	jurCoverage := map[string]int{
		"ca": 0,
		"us": 0,
		"eu": 0,
		"de": 0,
		"fr": 0,
		"gb": 0,
	}
	if primaryMatch.Jurisdiction != "" {
		jurCoverage[primaryMatch.Jurisdiction]++
	}

	seenEquiv := map[string]bool{primaryNode.URN: true}
	seenHarm := map[string]bool{primaryNode.URN: true}
	var equivMatches []CrosswalkMatch
	var harmMatches []CrosswalkMatch
	var threats []ThreatLink
	seenThreats := make(map[string]bool)

	// Collect all nodes for this canonical control ID across frameworks
	seedNodes := []*Node{primaryNode}
	for _, n := range ug.ctrlByCanon[canonID] {
		if n.URN != primaryNode.URN {
			seedNodes = append(seedNodes, n)
			if !seenEquiv[n.URN] {
				seenEquiv[n.URN] = true
				m := nodeToCrosswalkMatch(n, ontology.PredEquivalentTo, 1.0)
				equivMatches = append(equivMatches, m)
				jurCoverage[m.Jurisdiction]++
			}
		}
	}

	// Traverse outgoing and incoming edges on seedNodes
	for _, sn := range seedNodes {
		for _, e := range ug.edgesFrom[sn.URN] {
			target := ug.nodeByURN[e.ToNode]
			if target == nil {
				continue
			}
			switch e.Relation {
			case ontology.PredEquivalentTo:
				if !seenEquiv[target.URN] {
					seenEquiv[target.URN] = true
					m := nodeToCrosswalkMatch(target, ontology.PredEquivalentTo, 1.0)
					equivMatches = append(equivMatches, m)
					jurCoverage[m.Jurisdiction]++
				}
			case ontology.PredHarmonizesWith:
				if !seenHarm[target.URN] && !seenEquiv[target.URN] {
					seenHarm[target.URN] = true
					m := nodeToCrosswalkMatch(target, ontology.PredHarmonizesWith, e.Weight)
					harmMatches = append(harmMatches, m)
					jurCoverage[m.Jurisdiction]++
				}
			case ontology.PredMitigatesCWE, ontology.PredMitigatesTechnique:
				key := e.Relation + ":" + target.URN
				if !seenThreats[key] {
					seenThreats[key] = true
					threats = append(threats, nodeToThreatLink(target, e.Relation))
				}
			}
		}

		for _, e := range ug.edgesTo[sn.URN] {
			source := ug.nodeByURN[e.FromNode]
			if source == nil {
				continue
			}
			switch e.Relation {
			case ontology.PredAttacksControl:
				key := e.Relation + ":" + source.URN
				if !seenThreats[key] {
					seenThreats[key] = true
					threats = append(threats, nodeToThreatLink(source, e.Relation))
				}
			case ontology.PredEquivalentTo:
				if !seenEquiv[source.URN] {
					seenEquiv[source.URN] = true
					m := nodeToCrosswalkMatch(source, ontology.PredEquivalentTo, 1.0)
					equivMatches = append(equivMatches, m)
					jurCoverage[m.Jurisdiction]++
				}
			case ontology.PredHarmonizesWith:
				if !seenHarm[source.URN] && !seenEquiv[source.URN] {
					seenHarm[source.URN] = true
					m := nodeToCrosswalkMatch(source, ontology.PredHarmonizesWith, e.Weight)
					harmMatches = append(harmMatches, m)
					jurCoverage[m.Jurisdiction]++
				}
			}
		}
	}

	// Ensure thematic domain harmonization includes representatives across all 6 jurisdictions
	for _, dn := range ug.domainCtrls[primaryMatch.ThematicDomain] {
		if !seenHarm[dn.URN] && !seenEquiv[dn.URN] {
			seenHarm[dn.URN] = true
			m := nodeToCrosswalkMatch(dn, ontology.PredHarmonizesWith, 0.80)
			harmMatches = append(harmMatches, m)
			jurCoverage[m.Jurisdiction]++
		}
	}

	sort.Slice(equivMatches, func(i, j int) bool {
		return equivMatches[i].URN < equivMatches[j].URN
	})
	sort.Slice(harmMatches, func(i, j int) bool {
		return harmMatches[i].URN < harmMatches[j].URN
	})
	sort.Slice(threats, func(i, j int) bool {
		if threats[i].IsAnti != threats[j].IsAnti {
			return threats[i].IsAnti
		}
		return threats[i].URN < threats[j].URN
	})

	antiCount := 0
	mitCount := 0
	for _, t := range threats {
		if t.IsAnti {
			antiCount++
		} else {
			mitCount++
		}
	}

	posture := computeSubgraphVectorState("crosswalk:"+canonID, "reg:Control", len(equivMatches)+len(harmMatches)+1, antiCount, mitCount)

	return &CrosswalkReport{
		Query:                controlQuery,
		CanonicalControlID:   canonID,
		ThematicDomain:       primaryMatch.ThematicDomain,
		PrimaryControl:       &primaryMatch,
		EquivalentControls:   equivMatches,
		HarmonizedControls:   harmMatches,
		JurisdictionCoverage: jurCoverage,
		ActiveThreats:        threats,
		VectorPosture:        posture,
	}, nil
}

// ComputeBlastRadius executes a multi-hop blast-radius and 6D Vector State impact traversal on the default UnifiedGraph.
func ComputeBlastRadius(targetURNOrID string, maxDepth int) (*BlastRadiusReport, error) {
	ug, err := DefaultUnifiedGraph()
	if err != nil {
		return nil, err
	}
	return ug.ComputeBlastRadius(targetURNOrID, maxDepth)
}

// ComputeBlastRadius traverses up to maxDepth hops from targetURNOrID, collecting all impacted
// nodes, edges, frameworks, jurisdictions, ANTI relations, and computing the 6D Vector State.
func (ug *UnifiedGraph) ComputeBlastRadius(targetURNOrID string, maxDepth int) (*BlastRadiusReport, error) {
	if maxDepth <= 0 {
		maxDepth = 2
	}
	if maxDepth > 5 {
		maxDepth = 5
	}

	root := ug.resolveNode(targetURNOrID)
	if root == nil {
		return nil, fmt.Errorf("target %q not found in unified knowledge graph", targetURNOrID)
	}

	visitedNodes := map[string]*Node{root.URN: root}
	visitedEdges := make(map[string]*Edge)
	frontier := []*Node{root}
	truncated := false

	for depth := 0; depth < maxDepth && len(frontier) > 0 && !truncated; depth++ {
		sort.Slice(frontier, func(i, j int) bool { return frontier[i].URN < frontier[j].URN })
		var nextFrontier []*Node

		for _, curr := range frontier {
			if depth > 0 && (curr.Kind == ontology.ClassJurisdiction || curr.Kind == ontology.ClassAuthority || curr.Kind == ontology.ClassControlFamily) {
				continue
			}

			// Outgoing edges (deterministic order)
			outEdges := ug.edgesFrom[curr.URN]
			if len(outEdges) > 1 {
				sortedOut := make([]*Edge, len(outEdges))
				copy(sortedOut, outEdges)
				sort.Slice(sortedOut, func(i, j int) bool { return sortedOut[i].ID < sortedOut[j].ID })
				outEdges = sortedOut
			}
			for _, e := range outEdges {
				neighbor := ug.nodeByURN[e.ToNode]
				if neighbor == nil {
					continue
				}
				if _, seen := visitedNodes[neighbor.URN]; !seen {
					if len(visitedNodes) >= maxBlastRadiusNodes {
						truncated = true
						break
					}
					visitedNodes[neighbor.URN] = neighbor
					nextFrontier = append(nextFrontier, neighbor)
				}
				visitedEdges[e.ID] = e
			}
			if truncated {
				break
			}

			// Incoming edges (deterministic order)
			inEdges := ug.edgesTo[curr.URN]
			if len(inEdges) > 1 {
				sortedIn := make([]*Edge, len(inEdges))
				copy(sortedIn, inEdges)
				sort.Slice(sortedIn, func(i, j int) bool { return sortedIn[i].ID < sortedIn[j].ID })
				inEdges = sortedIn
			}
			for _, e := range inEdges {
				// Do not fan out to all unrelated sibling controls in a framework/profile when traversing through a parent framework/profile at depth > 0
				if depth > 0 && (curr.Kind == ontology.ClassFramework || curr.Kind == ontology.ClassProfile) &&
					(e.Relation == ontology.PredBelongsToFramework || e.Relation == ontology.PredSelectedByProfile) {
					continue
				}
				neighbor := ug.nodeByURN[e.FromNode]
				if neighbor == nil {
					continue
				}
				if _, seen := visitedNodes[neighbor.URN]; !seen {
					if len(visitedNodes) >= maxBlastRadiusNodes {
						truncated = true
						break
					}
					visitedNodes[neighbor.URN] = neighbor
					nextFrontier = append(nextFrontier, neighbor)
				}
				visitedEdges[e.ID] = e
			}
			if truncated {
				break
			}
		}

		sort.Slice(nextFrontier, func(i, j int) bool { return nextFrontier[i].URN < nextFrontier[j].URN })
		frontier = nextFrontier
	}

	nodes := make([]*Node, 0, len(visitedNodes))
	fwSet := make(map[string]bool)
	ctrlSet := make(map[string]bool)
	mitCtrlSet := make(map[string]bool)
	jurImpact := make(map[string]int)
	classBreakdown := make(map[string]int)

	for _, n := range visitedNodes {
		nodes = append(nodes, cloneNode(n))
		classBreakdown[n.Kind]++
		var lbl map[string]any
		_ = json.Unmarshal([]byte(n.LabelsJSON), &lbl)
		if j, ok := lbl["jurisdiction"].(string); ok && j != "" {
			jurImpact[j]++
		}
		if n.Kind == ontology.ClassFramework {
			if id, ok := lbl["id"].(string); ok && id != "" {
				fwSet[id] = true
			}
		}
		if n.Kind == ontology.ClassControl || n.Kind == ontology.ClassControlEnhancement {
			if fwID, ok := lbl["framework_id"].(string); ok && fwID != "" {
				fwSet[fwID] = true
			}
			ctrlSet[n.URN] = true
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].URN < nodes[j].URN })

	edges := make([]*Edge, 0, len(visitedEdges))
	relBreakdown := make(map[string]int)
	antiCount := 0
	mitCount := 0
	for _, e := range visitedEdges {
		edges = append(edges, cloneEdge(e))
		relBreakdown[e.Relation]++
		if ontology.IsAntiPredicate(e.Relation) {
			antiCount++
		}
		if e.Relation == ontology.PredMitigatesCWE || e.Relation == ontology.PredMitigatesTechnique {
			mitCount++
			mitCtrlSet[e.FromNode] = true
		}
	}
	sort.Slice(edges, func(i, j int) bool { return edges[i].ID < edges[j].ID })

	var fws []string
	for fw := range fwSet {
		fws = append(fws, fw)
	}
	sort.Strings(fws)

	var ctrls []string
	for c := range ctrlSet {
		ctrls = append(ctrls, c)
	}
	sort.Strings(ctrls)

	var mitCtrls []string
	for mc := range mitCtrlSet {
		mitCtrls = append(mitCtrls, mc)
	}
	sort.Strings(mitCtrls)

	posture := computeSubgraphVectorState(root.URN, root.Kind, len(ctrls), antiCount, mitCount)

	return &BlastRadiusReport{
		TargetQuery:         targetURNOrID,
		RootNode:            cloneNode(root),
		MaxDepth:            maxDepth,
		Truncated:           truncated,
		ImpactedNodes:       nodes,
		ImpactedEdges:       edges,
		ImpactedFrameworks:  fws,
		ImpactedControls:    ctrls,
		MitigatingControls:  mitCtrls,
		JurisdictionImpact:  jurImpact,
		ClassBreakdown:      classBreakdown,
		RelationBreakdown:   relBreakdown,
		AntiEdgeCount:       antiCount,
		MitigationEdgeCount: mitCount,
		VectorPosture:       posture,
	}, nil
}

// ExportSubgraph exports the blast-radius subgraph around targetURNOrID in "jsonld", "cytoscape", or "cypher" format.
func ExportSubgraph(targetURNOrID string, maxDepth int, format string) (string, error) {
	ug, err := DefaultUnifiedGraph()
	if err != nil {
		return "", err
	}
	return ug.ExportSubgraph(targetURNOrID, maxDepth, format)
}

// ExportSubgraph serializes the subgraph around targetURNOrID into "jsonld", "cytoscape", or "cypher".
func (ug *UnifiedGraph) ExportSubgraph(targetURNOrID string, maxDepth int, format string) (string, error) {
	rep, err := ug.ComputeBlastRadius(targetURNOrID, maxDepth)
	if err != nil {
		return "", err
	}

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "jsonld", "json-ld":
		graphItems := make([]map[string]any, 0, len(rep.ImpactedNodes))
		edgesFromNode := make(map[string][]*Edge)
		for _, e := range rep.ImpactedEdges {
			edgesFromNode[e.FromNode] = append(edgesFromNode[e.FromNode], e)
		}
		for _, n := range rep.ImpactedNodes {
			item := map[string]any{
				"@id":   n.URN,
				"@type": n.Kind,
			}
			var lbl map[string]any
			if err := json.Unmarshal([]byte(n.LabelsJSON), &lbl); err == nil {
				for k, v := range lbl {
					item[k] = v
				}
			}
			if outEdges := edgesFromNode[n.URN]; len(outEdges) > 0 {
				relMap := make(map[string][]string)
				for _, e := range outEdges {
					relMap[e.Relation] = append(relMap[e.Relation], e.ToNode)
				}
				for rel, targets := range relMap {
					item[rel] = targets
				}
			}
			graphItems = append(graphItems, item)
		}
		doc := map[string]any{
			"@context":       ontology.CanonicalSpec().Context,
			"@id":            "urn:xoscal:subgraph:" + rep.RootNode.URN,
			"root":           rep.RootNode.URN,
			"vector_posture": rep.VectorPosture,
			"@graph":         graphItems,
		}
		b, err := json.MarshalIndent(doc, "", "  ")
		return string(b), err

	case "cytoscape":
		var elements []map[string]any
		for _, n := range rep.ImpactedNodes {
			var lbl map[string]any
			_ = json.Unmarshal([]byte(n.LabelsJSON), &lbl)
			data := map[string]any{
				"id":    n.URN,
				"kind":  n.Kind,
				"label": n.URN,
			}
			if t, ok := lbl["title"].(string); ok && t != "" {
				data["label"] = t
			}
			for k, v := range lbl {
				data[k] = v
			}
			elements = append(elements, map[string]any{
				"group": "nodes",
				"data":  data,
			})
		}
		for _, e := range rep.ImpactedEdges {
			elements = append(elements, map[string]any{
				"group": "edges",
				"data": map[string]any{
					"id":      e.ID,
					"source":  e.FromNode,
					"target":  e.ToNode,
					"label":   e.Relation,
					"is_anti": ontology.IsAntiPredicate(e.Relation),
					"weight":  e.Weight,
				},
			})
		}
		doc := map[string]any{
			"root":           rep.RootNode.URN,
			"vector_posture": rep.VectorPosture,
			"elements":       elements,
		}
		b, err := json.MarshalIndent(doc, "", "  ")
		return string(b), err

	case "cypher":
		var sb strings.Builder
		sb.WriteString("// xOSCAL Property Graph Cypher Export\n")
		for _, n := range rep.ImpactedNodes {
			label := sanitizeCypherIdentifier(n.Kind)
			var lbl map[string]any
			_ = json.Unmarshal([]byte(n.LabelsJSON), &lbl)
			title, _ := lbl["title"].(string)
			_, _ = fmt.Fprintf(&sb, "MERGE (n:%s {urn: %s, title: %s});\n",
				label, escapeCypherString(n.URN), escapeCypherString(title))
		}
		for _, e := range rep.ImpactedEdges {
			relLabel := sanitizeCypherIdentifier(e.Relation)
			_, _ = fmt.Fprintf(&sb, "MATCH (a {urn: %s}), (b {urn: %s}) MERGE (a)-[:%s {is_anti: %t, weight: %.2f}]->(b);\n",
				escapeCypherString(e.FromNode), escapeCypherString(e.ToNode), relLabel, ontology.IsAntiPredicate(e.Relation), e.Weight)
		}
		return sb.String(), nil

	default:
		return "", fmt.Errorf("unsupported subgraph export format %q (expected: jsonld, cytoscape, cypher)", format)
	}
}

func escapeCypherString(s string) string {
	var sb strings.Builder
	sb.Grow(len(s) + 8)
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			sb.WriteString(`\\`)
		case '"':
			sb.WriteString(`\"`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			if r < 0x20 {
				_, _ = fmt.Fprintf(&sb, `\u%04x`, r)
			} else {
				sb.WriteRune(r)
			}
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

func sanitizeCypherIdentifier(raw string) string {
	var sb strings.Builder
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			sb.WriteRune(r)
		} else {
			sb.WriteByte('_')
		}
	}
	out := sb.String()
	if out == "" {
		return "_Node"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "_" + out
	}
	return out
}

// LoadUnifiedGraphFromSQLite reconstructs an operational UnifiedGraph directly from SQLite
// kg_graph_nodes and kg_graph_edges tables without rebuilding from static embedded catalogs.
func LoadUnifiedGraphFromSQLite(ctx context.Context, dsn string) (*UnifiedGraph, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("sqlite dsn is required")
	}
	st, err := NewSQLiteStore(dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite graph store %q: %w", dsn, err)
	}
	defer func() { _ = st.Close() }()

	sqlStore, ok := st.(*SQLiteStore)
	if !ok {
		return nil, fmt.Errorf("unexpected graph store implementation %T", st)
	}
	return LoadUnifiedGraphFromStore(ctx, sqlStore)
}

// LoadUnifiedGraphFromStore reconstructs an operational UnifiedGraph from an open *SQLiteStore.
func LoadUnifiedGraphFromStore(ctx context.Context, sqlStore *SQLiteStore) (*UnifiedGraph, error) {
	if sqlStore == nil {
		return nil, fmt.Errorf("sqlite graph store is nil")
	}
	nodes, edges, err := sqlStore.LoadAllGraph(ctx)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("sqlite graph store contains 0 nodes (run 'xoscal seed' first)")
	}

	ug := &UnifiedGraph{
		Nodes:       nodes,
		Edges:       edges,
		nodeByURN:   make(map[string]*Node, len(nodes)),
		edgesFrom:   make(map[string][]*Edge, len(nodes)),
		edgesTo:     make(map[string][]*Edge, len(nodes)),
		ctrlByCanon: make(map[string][]*Node, 2048),
		canonAlias:  make(map[string]string, 1024),
		knownFw:     make(map[string]string, 128),
		domainCtrls: make(map[string][]*Node, 16),
	}

	for _, fw := range frameworks.DefaultRegistry().List() {
		fwID := fw.ID()
		ug.knownFw[strings.ToLower(fwID)] = fwID
		for _, al := range fw.Aliases() {
			ug.knownFw[strings.ToLower(al)] = fwID
		}
	}
	crosswalkAliases := map[string]string{
		"nist-sp-800-53-rev5":  "nist-sp-800-53-rev5",
		"nist-800-53":          "nist-sp-800-53-rev5",
		"800-53":               "nist-sp-800-53-rev5",
		"nist-sp-800-171-rev2": "nist-sp-800-171-rev2",
		"nist-800-171":         "nist-sp-800-171-rev2",
		"800-171":              "nist-sp-800-171-rev2",
		"sp-800-171":           "nist-sp-800-171-rev2",
		"iso-27001-2022":       "iso-27001-2022",
		"iso-27001":            "iso-27001-2022",
		"iso27001":             "iso-27001-2022",
		"bsi-it-gs-2023":       "bsi-it-gs-2023",
		"bsi-it-gs":            "bsi-it-gs-2023",
		"anssi-hygiene":        "anssi-hygiene",
		"anssi":                "anssi-hygiene",
		"uk-cyber-essentials":  "uk-cyber-essentials",
		"cyber-essentials":     "uk-cyber-essentials",
	}
	for k, v := range crosswalkAliases {
		ug.knownFw[k] = v
	}

	sort.Slice(ug.Nodes, func(i, j int) bool { return ug.Nodes[i].URN < ug.Nodes[j].URN })
	sort.Slice(ug.Edges, func(i, j int) bool { return ug.Edges[i].ID < ug.Edges[j].ID })

	repByFwDomain := make(map[string]map[string]*Node)
	cmmcBy171 := make(map[string][]*Node)
	nodesByClass := make(map[string]int)
	jurCounts := make(map[string]int)

	for _, n := range ug.Nodes {
		ug.nodeByURN[n.URN] = n
		nodesByClass[n.Kind]++
		var lbl map[string]any
		_ = json.Unmarshal([]byte(n.LabelsJSON), &lbl)
		if j, ok := lbl["jurisdiction"].(string); ok && j != "" {
			jurCounts[j]++
		}
		if n.Kind == ontology.ClassFramework {
			fwID, _ := lbl["id"].(string)
			if fwID != "" {
				ug.knownFw[strings.ToLower(fwID)] = fwID
				if ver, _ := lbl["version"].(string); ver != "" {
					ug.expectedFwIDs = append(ug.expectedFwIDs, fwID)
				}
			}
		}
		if n.Kind == ontology.ClassControl || n.Kind == ontology.ClassControlEnhancement {
			cid, _ := lbl["id"].(string)
			cid = strings.ToLower(strings.TrimSpace(cid))
			fwID, _ := lbl["framework_id"].(string)
			dom, _ := lbl["thematic_domain"].(string)
			isEnh, _ := lbl["is_enhancement"].(bool)
			if cid != "" {
				ug.ctrlByCanon[cid] = appendUniqueNode(ug.ctrlByCanon[cid], n)
				if fwID == "cmmc-2.0" {
					if dashIdx := strings.LastIndex(cid, "-"); dashIdx != -1 && dashIdx+1 < len(cid) {
						sec171 := normalizeSP800171ID(cid[dashIdx+1:])
						cmmcBy171[sec171] = append(cmmcBy171[sec171], n)
					}
				}
			}
			if !isEnh && n.Kind == ontology.ClassControl && fwID != "" && dom != "" {
				if repByFwDomain[fwID] == nil {
					repByFwDomain[fwID] = make(map[string]*Node)
				}
				if _, exists := repByFwDomain[fwID][dom]; !exists {
					repByFwDomain[fwID][dom] = n
					ug.domainCtrls[dom] = append(ug.domainCtrls[dom], n)
				}
			}
		}
	}

	for _, cw := range feeds.BuiltinCrosswalks {
		canonID := strings.ToLower(cw.CanonicalControl)
		ug.canonAlias[canonID] = canonID
		if n53 := ug.nodeByURN["urn:xoscal:requirement:nist-sp-800-53-rev5:"+canonID]; n53 != nil {
			ug.ctrlByCanon[canonID] = appendUniqueNode(ug.ctrlByCanon[canonID], n53)
		}
		for _, s171 := range cw.SP800171Controls {
			raw171 := strings.ToLower(strings.TrimSpace(s171))
			short171 := normalizeSP800171ID(raw171)
			ug.canonAlias[raw171] = canonID
			ug.canonAlias[short171] = canonID
			for _, idForm := range []string{short171, raw171} {
				if n171 := ug.nodeByURN["urn:xoscal:requirement:nist-sp-800-171-rev2:"+idForm]; n171 != nil {
					ug.ctrlByCanon[canonID] = appendUniqueNode(ug.ctrlByCanon[canonID], n171)
				}
			}
			for _, cmmcNode := range cmmcBy171[short171] {
				ug.ctrlByCanon[canonID] = appendUniqueNode(ug.ctrlByCanon[canonID], cmmcNode)
				var lbl map[string]any
				_ = json.Unmarshal([]byte(cmmcNode.LabelsJSON), &lbl)
				if cid, _ := lbl["id"].(string); cid != "" {
					ug.canonAlias[cid] = canonID
				}
			}
		}
		for _, c171 := range cw.ITSP10171Controls {
			low171 := strings.ToLower(c171)
			ug.canonAlias[low171] = canonID
			ug.canonAlias[strings.TrimPrefix(low171, "_")] = canonID
			if n171 := ug.nodeByURN["urn:xoscal:requirement:cccs-itsp-10-171:"+low171]; n171 != nil {
				ug.ctrlByCanon[canonID] = appendUniqueNode(ug.ctrlByCanon[canonID], n171)
			}
		}
	}

	edgesByPred := make(map[string]int)
	antiEdges := 0
	for _, e := range ug.Edges {
		ug.edgesFrom[e.FromNode] = append(ug.edgesFrom[e.FromNode], e)
		ug.edgesTo[e.ToNode] = append(ug.edgesTo[e.ToNode], e)
		edgesByPred[e.Relation]++
		if ontology.IsAntiPredicate(e.Relation) {
			antiEdges++
		}
	}

	fwCount := len(ug.expectedFwIDs)
	if fwCount == 0 {
		fwCount = nodesByClass[ontology.ClassFramework]
	}
	ontSpec := ontology.CanonicalSpec()
	graphHex := computeGraphContentSHA256(ug.Nodes, ug.Edges, ontSpec.SHA256)

	ug.stats = &GraphStats{
		OntologyVersion:    ontology.OntologyVersion,
		OntologyNamespace:  ontology.OntologyNamespace,
		OntologySHA256:     ontSpec.SHA256,
		TotalNodes:         len(ug.Nodes),
		TotalEdges:         len(ug.Edges),
		AntiEdgeCount:      antiEdges,
		FrameworksCount:    fwCount,
		FeedsCount:         len(feeds.ListFeeds()),
		NodesByClass:       nodesByClass,
		EdgesByPredicate:   edgesByPred,
		JurisdictionCounts: jurCounts,
		GraphSHA256:        graphHex,
	}

	return ug, nil
}

func (ug *UnifiedGraph) resolveNode(query string) *Node {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil
	}
	if n, ok := ug.nodeByURN[q]; ok {
		return n
	}
	qLower := strings.ToLower(q)
	qUpper := strings.ToUpper(q)

	if resolvedFw, ok := ug.knownFw[qLower]; ok {
		if n, exists := ug.nodeByURN["urn:xoscal:framework:"+resolvedFw]; exists {
			return n
		}
	}
	if parts := strings.SplitN(qLower, ":", 2); len(parts) == 2 {
		if resolvedFw, ok := ug.knownFw[parts[0]]; ok {
			if n, exists := ug.nodeByURN["urn:xoscal:requirement:"+resolvedFw+":"+parts[1]]; exists {
				return n
			}
		}
	}

	// Try direct URN prefixes
	candidates := []string{
		"urn:xoscal:kev:" + qUpper,
		"urn:xoscal:cve:" + qUpper,
		"urn:xoscal:cwe:" + qUpper,
		"urn:xoscal:attack:" + qUpper,
		"urn:xoscal:framework:" + qLower,
		"urn:xoscal:profile:" + qLower,
		"urn:xoscal:jurisdiction:" + qLower,
		"urn:xoscal:family:" + qUpper,
		"urn:xoscal:requirement:" + qLower,
		"urn:xoscal:requirement:cccs-itsp-10-033:" + qLower,
	}
	for _, cand := range candidates {
		if n, ok := ug.nodeByURN[cand]; ok {
			return n
		}
	}
	lookupKey := qLower
	if mapped, ok := ug.canonAlias[qLower]; ok && mapped != "" {
		lookupKey = mapped
	}
	if ctrls := ug.ctrlByCanon[lookupKey]; len(ctrls) > 0 {
		for _, c := range ctrls {
			if strings.Contains(c.URN, ":cccs-itsp-10-033:") {
				return c
			}
		}
		return ctrls[0]
	}
	return nil
}

func nodeToCrosswalkMatch(n *Node, rel string, conf float64) CrosswalkMatch {
	var lbl map[string]any
	_ = json.Unmarshal([]byte(n.LabelsJSON), &lbl)
	fwID, _ := lbl["framework_id"].(string)
	jur, _ := lbl["jurisdiction"].(string)
	cid, _ := lbl["id"].(string)
	title, _ := lbl["title"].(string)
	dom, _ := lbl["thematic_domain"].(string)
	return CrosswalkMatch{
		URN:            n.URN,
		FrameworkID:    fwID,
		Jurisdiction:   jur,
		ControlID:      cid,
		Title:          title,
		ThematicDomain: dom,
		Relation:       rel,
		Confidence:     conf,
	}
}

func nodeToThreatLink(n *Node, rel string) ThreatLink {
	var lbl map[string]any
	_ = json.Unmarshal([]byte(n.LabelsJSON), &lbl)
	id, _ := lbl["id"].(string)
	title, _ := lbl["title"].(string)
	vendor, _ := lbl["vendor"].(string)
	product, _ := lbl["product"].(string)
	cweID, _ := lbl["cwe_id"].(string)
	desc, _ := lbl["short_description"].(string)
	if desc == "" {
		desc, _ = lbl["description"].(string)
	}
	var techs []string
	if rawTechs, ok := lbl["attack_techniques"].([]any); ok {
		for _, rt := range rawTechs {
			if s, ok := rt.(string); ok {
				techs = append(techs, s)
			}
		}
	}
	return ThreatLink{
		URN:         n.URN,
		Class:       n.Kind,
		ID:          id,
		Title:       title,
		Relation:    rel,
		IsAnti:      ontology.IsAntiPredicate(rel),
		Vendor:      vendor,
		Product:     product,
		RelatedCWE:  cweID,
		Techniques:  techs,
		Description: desc,
	}
}

func computeSubgraphVectorState(name, kind string, totalControls, antiCount, mitigationCount int) vectorstate.VectorState {
	if totalControls <= 0 {
		totalControls = 1
	}
	vs := vectorstate.VectorState{
		ArtifactName:  name,
		ArtifactKind:  kind,
		Presence:      vectorstate.PresencePresent,
		Valence:       vectorstate.ValencePositive,
		AntiCount:     antiCount,
		Coherence:     vectorstate.Coherent,
		Evidence:      vectorstate.EvidenceVerified,
		Lifecycle:     vectorstate.ModeNormal,
		Epoch:         time.Now().UTC(),
		TotalControls: totalControls,
		PassingCount:  totalControls,
	}

	if antiCount > 0 {
		if mitigationCount > 0 {
			vs.Coherence = vectorstate.PartiallyCoherent
			vs.Valence = vectorstate.ValenceMixed
			vs.Lifecycle = vectorstate.ModeDegraded
			vs.DegradedCount = antiCount
		} else {
			vs.Coherence = vectorstate.Decoherent
			vs.Valence = vectorstate.ValenceNegative
			vs.Lifecycle = vectorstate.ModeQuarantined
			vs.DegradedCount = antiCount
			vs.PassingCount = 0
		}
	}

	return vs
}
