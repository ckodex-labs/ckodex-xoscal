package graph_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/mchorfa/xoscal/server/internal/feeds"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
	"github.com/mchorfa/xoscal/server/internal/graph"
	"github.com/mchorfa/xoscal/server/internal/ontology"
	"github.com/mchorfa/xoscal/server/internal/resilience"
	"github.com/mchorfa/xoscal/server/internal/vectorstate"
)

func smallSixJurisdictionRegistry(t *testing.T) *frameworks.Registry {
	t.Helper()
	def := frameworks.DefaultRegistry()
	reg := frameworks.NewRegistry()
	for _, id := range []string{
		"cccs-itsg-33",
		"cmmc-2.0",
		"nis2",
		"bs-it-gs-2023-ops",
		"anssi-guide-hygiene",
		"cyber_essentials",
	} {
		fw, ok := def.Get(id)
		if !ok {
			t.Fatalf("missing framework %s in DefaultRegistry", id)
		}
		if err := reg.Register(fw); err != nil {
			t.Fatalf("register %s: %v", id, err)
		}
	}
	return reg
}

func TestBuildUnifiedKnowledgeGraphAndStats(t *testing.T) {
	ctx := context.Background()
	store, err := graph.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	ug, err := graph.BuildUnifiedKnowledgeGraph(ctx, store, frameworks.DefaultRegistry())
	if err != nil {
		t.Fatalf("BuildUnifiedKnowledgeGraph: %v", err)
	}

	stats := ug.Stats()
	if stats.FrameworksCount < 36 {
		t.Errorf("expected at least 36 frameworks in unified graph, got %d", stats.FrameworksCount)
	}
	if stats.FeedsCount != 4 {
		t.Errorf("expected 4 feeds in unified graph, got %d", stats.FeedsCount)
	}
	if stats.TotalNodes < 10000 {
		t.Errorf("expected >10,000 nodes across 36 frameworks + 4 feeds, got %d", stats.TotalNodes)
	}
	if stats.TotalEdges < 20000 {
		t.Errorf("expected >20,000 edges across 36 frameworks + 4 feeds, got %d", stats.TotalEdges)
	}
	if stats.AntiEdgeCount == 0 {
		t.Errorf("expected non-zero ANTI edges in unified graph")
	}

	// Verify every edge in UnifiedGraph conforms to ontology domain/range constraints
	nodeKind := make(map[string]string, len(ug.Nodes))
	for _, n := range ug.Nodes {
		nodeKind[n.URN] = n.Kind
	}
	for _, e := range ug.Edges {
		fromClass := nodeKind[e.FromNode]
		toClass := nodeKind[e.ToNode]
		if err := ontology.ValidateTriple(fromClass, e.Relation, toClass); err != nil {
			t.Fatalf("invalid ontology triple on edge %s (%s -[%s]-> %s): %v", e.ID, fromClass, e.Relation, toClass, err)
		}
	}

	// Verify nodes and edges were persisted to SQLiteStore via BulkUpsertGraph without cascade wipeout
	fwNode, err := store.GetNode(ctx, "urn:xoscal:framework:cccs-itsp-10-033")
	if err != nil || fwNode == nil {
		t.Fatalf("expected cccs-itsp-10-033 node persisted in SQLiteStore: %v", err)
	}
	kevNode, err := store.GetNode(ctx, "urn:xoscal:kev:CVE-2024-3400")
	if err != nil || kevNode == nil {
		t.Fatalf("expected CVE-2024-3400 KEV node persisted in SQLiteStore: %v", err)
	}
}

func TestQueryCrosswalkMultiJurisdiction(t *testing.T) {
	rep, err := graph.QueryCrosswalk("ac-2")
	if err != nil {
		t.Fatalf("QueryCrosswalk('ac-2') error: %v", err)
	}
	if rep.CanonicalControlID != "ac-2" {
		t.Errorf("CanonicalControlID = %q, want 'ac-2'", rep.CanonicalControlID)
	}
	if len(rep.EquivalentControls) == 0 {
		t.Errorf("expected non-empty EquivalentControls for ac-2")
	}
	if len(rep.HarmonizedControls) == 0 {
		t.Errorf("expected non-empty HarmonizedControls for ac-2")
	}

	// Verify key equivalent frameworks (NIST 800-53, NIST 800-171, CCCS ITSP.10.171, CMMC 2.0, ITSG-33, PBMM, FedRAMP)
	equivFws := make(map[string]bool)
	for _, m := range rep.EquivalentControls {
		equivFws[m.FrameworkID] = true
	}
	for _, wantFw := range []string{"nist-sp-800-53-rev5", "nist-sp-800-171-rev2", "cccs-itsp-10-171", "cmmc-2.0", "cccs-itsg-33", "cccs-medium-cloud-pbmm", "fedramp-rev5"} {
		if !equivFws[wantFw] {
			t.Errorf("expected %q in ac-2 EquivalentControls, got %v", wantFw, equivFws)
		}
	}

	// Verify key harmonized frameworks (ISO 27001, BSI, ANSSI, NIS2, DORA, CyberSecure Canada, NIST CSF 2.0, UK Cyber Essentials)
	harmFws := make(map[string]bool)
	for _, m := range rep.HarmonizedControls {
		harmFws[m.FrameworkID] = true
	}
	for _, wantFw := range []string{"iso-27001-2022", "bsi-it-gs-2023", "anssi-hygiene", "uk-cyber-essentials", "nis2", "dora", "cybersecure-canada", "nist-csf-2.0"} {
		if !harmFws[wantFw] {
			t.Errorf("expected %q in ac-2 HarmonizedControls, got %v", wantFw, harmFws)
		}
	}

	// Verify all 6 jurisdictions are covered
	for _, jur := range []string{"ca", "us", "eu", "de", "fr", "gb"} {
		if rep.JurisdictionCoverage[jur] == 0 {
			t.Errorf("expected jurisdiction %q coverage > 0 for ac-2, got 0", jur)
		}
	}

	// Verify ActiveThreats and 6D VectorState
	if len(rep.ActiveThreats) == 0 {
		t.Errorf("expected ActiveThreats linked to ac-2")
	}
	if rep.VectorPosture.AntiCount == 0 {
		t.Errorf("expected AntiCount > 0 on ac-2 posture due to active KEV/ATT&CK threats")
	}

	// Verify SP 800-171, CCCS ITSP.10.171, CMMC 2.0, and framework alias crosswalk queries resolve to canonical ac-2 with full threats
	for _, q := range []string{"3.1.1", "03.01.01", "cccs-itsp-10-171:_03.01.01.a", "cmmc-2.0:ac.l1-3.1.1", "itsp.10.033:ac-2"} {
		r, err := graph.QueryCrosswalk(q)
		if err != nil {
			t.Fatalf("QueryCrosswalk(%q) error: %v", q, err)
		}
		if r.CanonicalControlID != "ac-2" {
			t.Errorf("QueryCrosswalk(%q) CanonicalControlID = %q, want 'ac-2'", q, r.CanonicalControlID)
		}
		if len(r.ActiveThreats) == 0 {
			t.Errorf("QueryCrosswalk(%q) expected ActiveThreats > 0", q)
		}
	}

	// Verify invalid framework prefix and nonexistent control return errors instead of silently falling back
	if _, err := graph.QueryCrosswalk("nonexistent-fw:ac-2"); err == nil {
		t.Errorf("expected error for unknown framework prefix 'nonexistent-fw:ac-2', got nil")
	}
	if _, err := graph.QueryCrosswalk("cccs-itsp-10-033:nonexistent-ctrl-999"); err == nil {
		t.Errorf("expected error for nonexistent control in cccs-itsp-10-033, got nil")
	}
}

func TestComputeBlastRadiusAndExportSubgraph(t *testing.T) {
	// 1. Blast radius for CVE-2024-3400 (active KEV with mitigating controls -> PARTIALLY_COHERENT)
	br, err := graph.ComputeBlastRadius("CVE-2024-3400", 2)
	if err != nil {
		t.Fatalf("ComputeBlastRadius('CVE-2024-3400') error: %v", err)
	}
	if br.AntiEdgeCount == 0 {
		t.Errorf("expected AntiEdgeCount > 0 for CVE-2024-3400")
	}
	if len(br.ImpactedControls) == 0 {
		t.Errorf("expected ImpactedControls > 0 for CVE-2024-3400")
	}
	if br.VectorPosture.Coherence != vectorstate.PartiallyCoherent && br.VectorPosture.Coherence != vectorstate.Decoherent {
		t.Errorf("expected PARTIALLY_COHERENT or DECOHERENT for CVE-2024-3400, got %s", br.VectorPosture.Coherence)
	}

	// 1b. Verify blast radius from a single control at depth 2 does not explode through parent framework into unrelated controls like pe-1
	brCtrl, err := graph.ComputeBlastRadius("cccs-itsp-10-033:si-2", 2)
	if err != nil {
		t.Fatalf("ComputeBlastRadius('cccs-itsp-10-033:si-2', 2) error: %v", err)
	}
	for _, cURN := range brCtrl.ImpactedControls {
		if strings.HasSuffix(cURN, ":pe-1") {
			t.Errorf("ComputeBlastRadius('cccs-itsp-10-033:si-2', 2) leaked unrelated physical control %s via framework hub explosion", cURN)
		}
	}
	if len(brCtrl.ImpactedNodes) > 500 {
		t.Errorf("expected bounded subgraph (<500 nodes) for cccs-itsp-10-033:si-2 at depth 2, got %d nodes", len(brCtrl.ImpactedNodes))
	}

	// 2. Blast radius for cccs-itsg-33 (superseded framework with 0 CWE mitigations at depth 1 -> DECOHERENT)
	brFw, err := graph.ComputeBlastRadius("cccs-itsg-33", 1)
	if err != nil {
		t.Fatalf("ComputeBlastRadius('cccs-itsg-33') error: %v", err)
	}
	if brFw.AntiEdgeCount == 0 {
		t.Errorf("expected SUPERSEDES_FRAMEWORK ANTI edge on cccs-itsg-33")
	}
	if brFw.VectorPosture.Coherence != vectorstate.Decoherent {
		t.Errorf("expected DECOHERENT posture for superseded framework cccs-itsg-33 at depth 1, got %s", brFw.VectorPosture.Coherence)
	}

	// 3. ExportSubgraph in jsonld, cytoscape, and cypher formats
	for _, format := range []string{"jsonld", "cytoscape", "cypher"} {
		out, err := graph.ExportSubgraph("CVE-2024-3400", 1, format)
		if err != nil {
			t.Fatalf("ExportSubgraph(%q) error: %v", format, err)
		}
		if !strings.Contains(out, "CVE-2024-3400") {
			t.Errorf("ExportSubgraph(%q) missing CVE-2024-3400 in output", format)
		}
	}
}

func TestGraphReconcileInvariantsAndTamperDetection(t *testing.T) {
	ug, err := graph.BuildUnifiedKnowledgeGraph(context.Background(), nil, smallSixJurisdictionRegistry(t))
	if err != nil {
		t.Fatalf("BuildUnifiedKnowledgeGraph: %v", err)
	}

	rep, err := ug.ReconcileInvariants()
	if err != nil {
		t.Fatalf("ReconcileInvariants error: %v", err)
	}
	if !rep.Coherent || len(rep.Violations) != 0 {
		t.Fatalf("expected Coherent=true with 0 violations, got violations=%v", rep.Violations)
	}
	if rep.VectorPosture.Coherence != vectorstate.Coherent || rep.VectorPosture.Evidence != vectorstate.EvidenceVerified {
		t.Errorf("unexpected VectorPosture on coherent graph: %+v", rep.VectorPosture)
	}

	// Inject in-memory bitrot into a node label -> ReconcileInvariants must flag SHA-256 mismatch & DECOHERENT
	origLabels := ug.Nodes[0].LabelsJSON
	ug.Nodes[0].LabelsJSON = `{"tampered":true}`
	badRep, err := ug.ReconcileInvariants()
	if err != nil {
		t.Fatalf("ReconcileInvariants on tampered graph error: %v", err)
	}
	if badRep.Coherent || len(badRep.Violations) == 0 {
		t.Fatalf("expected ReconcileInvariants to detect in-memory bitrot")
	}
	if badRep.VectorPosture.Coherence != vectorstate.Decoherent || badRep.VectorPosture.Evidence != vectorstate.EvidenceTampered {
		t.Errorf("expected DECOHERENT + TAMPERED posture, got %+v", badRep.VectorPosture)
	}
	ug.Nodes[0].LabelsJSON = origLabels

	// Inject in-memory bitrot into an edge TrustState -> ReconcileInvariants must flag SHA-256 mismatch
	origTrust := ug.Edges[0].TrustState
	ug.Edges[0].TrustState = "tampered-trust"
	edgeBadRep, err := ug.ReconcileInvariants()
	if err != nil {
		t.Fatalf("ReconcileInvariants on tampered edge error: %v", err)
	}
	if edgeBadRep.Coherent || edgeBadRep.VectorPosture.Evidence != vectorstate.EvidenceTampered {
		t.Fatalf("expected ReconcileInvariants to detect edge TrustState bitrot, got %+v", edgeBadRep)
	}
	ug.Edges[0].TrustState = origTrust

	// Inject a dangling + ontology-violating edge -> ReconcileInvariants must detect it
	ug.Edges = append(ug.Edges, &graph.Edge{
		ID:       "edge:dangling:bad",
		FromNode: "urn:xoscal:jurisdiction:ca",
		ToNode:   "urn:xoscal:missing:ghost",
		Relation: ontology.PredAttacksControl,
	})
	danglingRep, _ := ug.ReconcileInvariants()
	if danglingRep.Coherent {
		t.Fatalf("expected Coherent=false on dangling edge")
	}
}

func TestReloadDefaultUnifiedGraphSelfHealing(t *testing.T) {
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// Cancelled reload fails cleanly without poisoning DefaultUnifiedGraph
	if _, err := graph.ReloadDefaultUnifiedGraph(cancelledCtx); err == nil {
		t.Fatalf("expected error on cancelled context reload")
	}

	ug, err := graph.ReloadDefaultUnifiedGraph(context.Background())
	if err != nil || ug == nil {
		t.Fatalf("expected clean hot-reload on valid context, got err=%v", err)
	}
	rec, err := ug.ReconcileInvariants()
	if err != nil || !rec.Coherent {
		t.Fatalf("reloaded graph failed ReconcileInvariants: err=%v rep=%+v", err, rec)
	}
}

func TestConcurrentGraphAccessAndDefensiveDeepCopy(t *testing.T) {
	ug, err := graph.DefaultUnifiedGraph()
	if err != nil {
		t.Fatalf("DefaultUnifiedGraph: %v", err)
	}
	origSHA := ug.Stats().GraphSHA256

	const workers = 32
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		workerID := i
		wg.Add(1)
		go func() {
			defer wg.Done()

			// 1. Mutate Stats() maps aggressively
			st := ug.Stats()
			st.NodesByClass["CORRUPTED_CLASS"] = 999999
			st.EdgesByPredicate["CORRUPTED_PRED"] = 999999
			st.JurisdictionCounts["ca"] = -1

			// 2. QueryCrosswalk
			cw, cwErr := ug.QueryCrosswalk("ac-2")
			if cwErr != nil {
				t.Errorf("worker %d QueryCrosswalk error: %v", workerID, cwErr)
				return
			}
			if len(cw.EquivalentControls) > 0 {
				cw.EquivalentControls[0].Title = "MUTATED"
			}

			// 3. ComputeBlastRadius and mutate returned Node/Edge structs
			br, brErr := ug.ComputeBlastRadius("CVE-2024-3400", 2)
			if brErr != nil {
				t.Errorf("worker %d ComputeBlastRadius error: %v", workerID, brErr)
				return
			}
			br.RootNode.Kind = "MUTATED_KIND"
			br.RootNode.LabelsJSON = `{"corrupted":true}`
			if len(br.ImpactedNodes) > 0 {
				br.ImpactedNodes[0].URN = "urn:corrupted"
				br.ImpactedNodes[0].LabelsJSON = `{"corrupted":true}`
			}
			if len(br.ImpactedEdges) > 0 {
				br.ImpactedEdges[0].Relation = "CORRUPTED_REL"
				br.ImpactedEdges[0].FromNode = "urn:corrupted"
			}

			// 4. ExportSubgraph
			if _, expErr := ug.ExportSubgraph("CVE-2024-3400", 1, "jsonld"); expErr != nil {
				t.Errorf("worker %d ExportSubgraph error: %v", workerID, expErr)
			}

			// 5. ReconcileInvariants concurrently across subset of workers while mutators hammer reads
			if workerID%8 == 0 {
				rec, recErr := ug.ReconcileInvariants()
				if recErr != nil || !rec.Coherent {
					t.Errorf("worker %d ReconcileInvariants detected corruption: err=%v violations=%v", workerID, recErr, rec.Violations)
				}
			}
		}()
	}
	wg.Wait()

	finalRec, err := ug.ReconcileInvariants()
	if err != nil || !finalRec.Coherent || finalRec.GraphSHA256 != origSHA {
		t.Fatalf("singleton graph corrupted after 32-goroutine chaos run: err=%v coherent=%v sha=%s want=%s violations=%v",
			err, finalRec.Coherent, finalRec.GraphSHA256, origSHA, finalRec.Violations)
	}
}

func TestBlastRadiusHardTruncationDeterminism(t *testing.T) {
	// Build a synthetic framework with 5,300 controls to trigger maxBlastRadiusNodes (5,000)
	type minCtrl struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	ctrls := make([]minCtrl, 5300)
	for i := range ctrls {
		ctrls[i] = minCtrl{
			ID:    fmt.Sprintf("ctrl-%04d", i+1),
			Title: fmt.Sprintf("Synthetic Control %04d", i+1),
		}
	}
	catDoc, err := json.Marshal(map[string]any{
		"catalog": map[string]any{
			"uuid":     "11111111-2222-3333-4444-555555555555",
			"metadata": map[string]any{"title": "Mega Framework", "version": "1.0", "oscal-version": "1.2.3"},
			"controls": ctrls,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	reg := frameworks.NewRegistry()
	if err := reg.Register(frameworks.New(frameworks.Config{
		ID:           "mega-fw",
		Title:        "Mega Synthetic Framework",
		Version:      "1.0",
		Jurisdiction: "ca",
		Authority:    "CCCS",
		ControlCount: len(ctrls),
		CatalogFn:    func() ([]byte, error) { return catDoc, nil },
	})); err != nil {
		t.Fatal(err)
	}

	ug, err := graph.BuildUnifiedKnowledgeGraph(context.Background(), nil, reg)
	if err != nil {
		t.Fatalf("BuildUnifiedKnowledgeGraph mega-fw: %v", err)
	}

	br1, err := ug.ComputeBlastRadius("mega-fw", 1)
	if err != nil {
		t.Fatalf("ComputeBlastRadius mega-fw run 1: %v", err)
	}
	if !br1.Truncated {
		t.Fatalf("expected Truncated=true for >5000 neighbor fan-out, got false (nodes=%d)", len(br1.ImpactedNodes))
	}
	if len(br1.ImpactedNodes) != 5000 {
		t.Fatalf("expected hard bound of 5000 ImpactedNodes, got %d", len(br1.ImpactedNodes))
	}

	// Second run must produce 100% identical deterministic node URNs
	br2, err := ug.ComputeBlastRadius("mega-fw", 1)
	if err != nil {
		t.Fatalf("ComputeBlastRadius mega-fw run 2: %v", err)
	}
	if len(br2.ImpactedNodes) != len(br1.ImpactedNodes) {
		t.Fatalf("non-deterministic node count: %d vs %d", len(br1.ImpactedNodes), len(br2.ImpactedNodes))
	}
	for i := range br1.ImpactedNodes {
		if br1.ImpactedNodes[i].URN != br2.ImpactedNodes[i].URN {
			t.Fatalf("non-deterministic node at index %d: %s vs %s", i, br1.ImpactedNodes[i].URN, br2.ImpactedNodes[i].URN)
		}
	}
}

func TestBuildUnifiedKnowledgeGraphFailsClosedOnMissingCatalog(t *testing.T) {
	reg := frameworks.NewRegistry()
	if err := reg.Register(frameworks.New(frameworks.Config{
		ID:           "broken-fw",
		Title:        "Broken Framework",
		Version:      "1.0",
		Jurisdiction: "ca",
		Authority:    "CCCS",
		ControlCount: 10,
		CatalogFn: func() ([]byte, error) {
			return nil, errors.New("simulated missing catalog.json")
		},
	})); err != nil {
		t.Fatal(err)
	}

	if _, err := graph.BuildUnifiedKnowledgeGraph(context.Background(), nil, reg); err == nil {
		t.Fatalf("expected BuildUnifiedKnowledgeGraph to fail closed when framework catalog load fails, got nil")
	}

	emptyReg := frameworks.NewRegistry()
	emptyCat, _ := json.Marshal(map[string]any{
		"catalog": map[string]any{
			"uuid":     "11111111-2222-3333-4444-555555555555",
			"metadata": map[string]any{"title": "Empty", "version": "1.0", "oscal-version": "1.2.3"},
			"controls": []any{},
		},
	})
	if err := emptyReg.Register(frameworks.New(frameworks.Config{
		ID:           "empty-fw",
		Title:        "Empty Framework",
		Version:      "1.0",
		Jurisdiction: "ca",
		Authority:    "CCCS",
		ControlCount: 0,
		CatalogFn:    func() ([]byte, error) { return emptyCat, nil },
	})); err != nil {
		t.Fatal(err)
	}

	if _, err := graph.BuildUnifiedKnowledgeGraph(context.Background(), nil, emptyReg); err == nil {
		t.Fatalf("expected BuildUnifiedKnowledgeGraph to fail closed when framework has 0 controls, got nil")
	}
}

func TestLiveSyncedCVEProjectsIntoUnifiedGraphAndSQLiteRoundTrip(t *testing.T) {
	t.Cleanup(feeds.ResetSyncedCache)
	feeds.ResetSyncedCache()

	cpDir := filepath.Join(t.TempDir(), "feed_cache")
	t.Setenv("XOSCAL_FEED_CACHE_DIR", cpDir)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		switch {
		case strings.HasSuffix(r.URL.Path, "/"+feeds.FeedCISAKEV):
			_, _ = w.Write([]byte(`{
				"catalogVersion":"2026.11.10",
				"vulnerabilities":[
					{
						"cveID":"CVE-2099-9999",
						"vendorProject":"QuantumEdge",
						"product":"MeshFirewall",
						"vulnerabilityName":"MeshFirewall Auth Bypass",
						"shortDescription":"Title with \"quotes\" and \\backslash\\ and\nnewline",
						"cwes":["CWE-287"]
					}
				]
			}`))
		case strings.HasSuffix(r.URL.Path, "/"+feeds.FeedMITREAttack):
			_, _ = w.Write([]byte(`{"attack_techniques":[{"technique_id":"T1999","name":"Quote \"Test\" \\ Slash\tTab","tactic":"initial-access","mitigating_controls":["ia-2","ac-2"]}]}`))
		case strings.HasSuffix(r.URL.Path, "/"+feeds.FeedCrosswalk):
			_, _ = w.Write([]byte(`{"catalog":{"uuid":"11111111-2222-3333-4444-555555555555","controls":[{"id":"ac-2"}]}}`))
		default:
			_, _ = w.Write([]byte(`<html><body>CCCS ITSP.10.033 Sovereign Security Control Catalogue Effective 2026-03-31</body></html>`))
		}
	}))
	defer ts.Close()

	if _, err := feeds.SyncFeeds(context.Background(), feeds.SyncOptions{
		Offline:         false,
		HTTPClient:      ts.Client(),
		BaseURL:         ts.URL,
		CheckpointDir:   cpDir,
		CircuitRegistry: resilience.NewCircuitRegistry(resilience.CircuitConfig{FailureThreshold: 3}),
	}); err != nil {
		t.Fatalf("SyncFeeds: %v", err)
	}

	// Clear in-memory cache so BuildUnifiedKnowledgeGraph reads from verified XOSCAL_FEED_CACHE_DIR checkpoint
	feeds.ResetSyncedCache()

	smallReg := smallSixJurisdictionRegistry(t)
	dbPath := filepath.Join(t.TempDir(), "xoscal_graph.db")
	st, err := graph.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	memUG, err := graph.BuildUnifiedKnowledgeGraph(context.Background(), st, smallReg)
	_ = st.Close()
	if err != nil {
		t.Fatalf("BuildUnifiedKnowledgeGraph to SQLite: %v", err)
	}

	br, err := memUG.ComputeBlastRadius("CVE-2099-9999", 2)
	if err != nil {
		t.Fatalf("ComputeBlastRadius('CVE-2099-9999') failed: %v", err)
	}
	if br.AntiEdgeCount == 0 || len(br.ImpactedControls) == 0 {
		t.Fatalf("expected novel CVE-2099-9999 to project ANTI edges and impacted controls, got %+v", br)
	}

	// Verify Cypher export escapes quotes, backslashes, and tabs cleanly
	cypherOut, err := memUG.ExportSubgraph("T1999", 1, "cypher")
	if err != nil {
		t.Fatalf("ExportSubgraph('T1999', 'cypher') failed: %v", err)
	}
	if !strings.Contains(cypherOut, `\"Test\"`) || !strings.Contains(cypherOut, `\\`) || !strings.Contains(cypherOut, `\t`) {
		t.Fatalf("Cypher export did not escape quotes/backslashes/tabs properly:\n%s", cypherOut)
	}

	// Verify SQLite write -> LoadUnifiedGraphFromSQLite parity
	sqlUG, err := graph.LoadUnifiedGraphFromSQLite(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("LoadUnifiedGraphFromSQLite: %v", err)
	}
	if sqlUG.Stats().GraphSHA256 != memUG.Stats().GraphSHA256 {
		t.Fatalf("SQLite GraphSHA256 mismatch: got %s, want %s", sqlUG.Stats().GraphSHA256, memUG.Stats().GraphSHA256)
	}
	rec, err := sqlUG.ReconcileInvariants()
	if err != nil || !rec.Coherent {
		t.Fatalf("SQLite-loaded graph failed ReconcileInvariants: err=%v violations=%v", err, rec.Violations)
	}
	cw, err := sqlUG.QueryCrosswalk("3.1.1")
	if err != nil || cw.CanonicalControlID != "ac-2" {
		t.Fatalf("SQLite-loaded QueryCrosswalk('3.1.1') failed: err=%v cw=%+v", err, cw)
	}
	sqlBR, err := sqlUG.ComputeBlastRadius("CVE-2099-9999", 2)
	if err != nil || sqlBR.AntiEdgeCount != br.AntiEdgeCount {
		t.Fatalf("SQLite-loaded ComputeBlastRadius('CVE-2099-9999') mismatch: err=%v got=%d want=%d", err, sqlBR.AntiEdgeCount, br.AntiEdgeCount)
	}
}
