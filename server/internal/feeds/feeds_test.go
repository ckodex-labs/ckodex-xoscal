package feeds_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mchorfa/xoscal/server/internal/feeds"
	"github.com/mchorfa/xoscal/server/internal/resilience"
)

func TestListAndInspectBuiltinFeeds(t *testing.T) {
	list := feeds.ListFeeds()
	if len(list) != 4 {
		t.Fatalf("expected 4 built-in authoritative feeds, got %d", len(list))
	}

	expectedIDs := []string{
		feeds.FeedCrosswalk,
		feeds.FeedCISAKEV,
		feeds.FeedMITREAttack,
		feeds.FeedSovereignLifecycle,
	}

	for _, id := range expectedIDs {
		insp, err := feeds.InspectFeed(id)
		if err != nil {
			t.Fatalf("InspectFeed(%q) error: %v", id, err)
		}
		if !insp.Descriptor.AirGapReady {
			t.Errorf("expected feed %q to be AirGapReady", id)
		}
		if len(insp.Descriptor.SHA256) != 64 {
			t.Errorf("expected 64-hex SHA256 on feed %q, got %q", id, insp.Descriptor.SHA256)
		}
	}

	kevInsp, _ := feeds.InspectFeed(feeds.FeedCISAKEV)
	if len(kevInsp.KEVAdvisories) < 15 {
		t.Errorf("expected at least 15 curated KEV advisories, got %d", len(kevInsp.KEVAdvisories))
	}

	attInsp, _ := feeds.InspectFeed(feeds.FeedMITREAttack)
	if len(attInsp.AttackTechniques) < 15 {
		t.Errorf("expected at least 15 curated MITRE ATT&CK techniques, got %d", len(attInsp.AttackTechniques))
	}

	if _, err := feeds.InspectFeed("non-existent-feed"); err == nil {
		t.Errorf("expected error for non-existent feed")
	}
}

func TestSyncFeedsAirGapAndLiveHTTP(t *testing.T) {
	ctx := context.Background()

	// 1. Air-gap offline sync
	rcpt, err := feeds.SyncFeeds(ctx, feeds.SyncOptions{Offline: true})
	if err != nil {
		t.Fatalf("SyncFeeds(Offline: true) error: %v", err)
	}
	if rcpt.Mode != "airgap-embedded" {
		t.Errorf("Mode = %q, want 'airgap-embedded'", rcpt.Mode)
	}
	if rcpt.OperationalMode != "NORMAL" || rcpt.FallbackCount != 0 {
		t.Errorf("expected OperationalMode=NORMAL FallbackCount=0, got %s / %d", rcpt.OperationalMode, rcpt.FallbackCount)
	}
	if len(rcpt.Feeds) != 4 {
		t.Errorf("expected 4 feed entries in receipt, got %d", len(rcpt.Feeds))
	}
	if len(rcpt.BundleSHA256) != 64 || len(rcpt.OntologySHA256) != 64 {
		t.Errorf("invalid receipt digests: bundle=%q ontology=%q", rcpt.BundleSHA256, rcpt.OntologySHA256)
	}
	if rcpt.TotalNodes == 0 || rcpt.TotalEdges == 0 {
		t.Errorf("expected non-zero projected nodes/edges, got nodes=%d edges=%d", rcpt.TotalNodes, rcpt.TotalEdges)
	}

	// 2. Single feed filter
	singleRcpt, err := feeds.SyncFeeds(ctx, feeds.SyncOptions{FeedID: feeds.FeedCISAKEV, Offline: true})
	if err != nil {
		t.Fatalf("SyncFeeds single feed error: %v", err)
	}
	if len(singleRcpt.Feeds) != 1 || singleRcpt.Feeds[0].FeedID != feeds.FeedCISAKEV {
		t.Errorf("unexpected single feed receipt: %+v", singleRcpt.Feeds)
	}

	// 3. Simulated Live HTTP sync hitting local httptest.Server with valid upstream schemas
	var hits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		switch {
		case strings.HasSuffix(r.URL.Path, "/"+feeds.FeedCrosswalk):
			_, _ = w.Write([]byte(`{"catalog":{"uuid":"11111111-2222-3333-4444-555555555555","controls":[{"id":"ac-2"}]}}`))
		case strings.HasSuffix(r.URL.Path, "/"+feeds.FeedCISAKEV):
			_, _ = w.Write([]byte(`{"catalogVersion":"2026.11.09","vulnerabilities":[{"cveID":"CVE-2024-3400","vendorProject":"Palo Alto","product":"PAN-OS","vulnerabilityName":"PAN-OS Command Injection","shortDescription":"Active exploit","cwes":["CWE-77"]}]}`))
		case strings.HasSuffix(r.URL.Path, "/"+feeds.FeedMITREAttack):
			_, _ = w.Write([]byte(`{"type":"bundle","id":"bundle--1","objects":[{"type":"attack-pattern","id":"attack-pattern--1"},{"type":"course-of-action","id":"course-of-action--1"}]}`))
		default:
			_, _ = w.Write([]byte(`<html><body>CCCS ITSP.10.033 Sovereign Security Control Catalogue Effective 2026-03-31</body></html>`))
		}
	}))
	defer ts.Close()

	liveRcpt, err := feeds.SyncFeeds(ctx, feeds.SyncOptions{
		Offline:         false,
		HTTPClient:      ts.Client(),
		BaseURL:         ts.URL,
		CircuitRegistry: resilience.NewCircuitRegistry(resilience.CircuitConfig{FailureThreshold: 3}),
	})
	if err != nil {
		t.Fatalf("SyncFeeds live mode error: %v", err)
	}
	if liveRcpt.Mode != "live-http" {
		t.Errorf("expected Mode 'live-http', got %q", liveRcpt.Mode)
	}
	if liveRcpt.OperationalMode != "NORMAL" || liveRcpt.FallbackCount != 0 {
		t.Errorf("expected OperationalMode=NORMAL FallbackCount=0, got %s / %d", liveRcpt.OperationalMode, liveRcpt.FallbackCount)
	}
	if got := hits.Load(); got != 4 {
		t.Errorf("expected httptest.Server to receive 4 feed requests, got %d", got)
	}
	for _, entry := range liveRcpt.Feeds {
		if entry.Mode != "live-http" {
			t.Errorf("expected feed %s entry Mode 'live-http', got %q (reason: %s)", entry.FeedID, entry.Mode, entry.DegradedReason)
		}
		if entry.LiveRecordsParsed == 0 {
			t.Errorf("expected LiveRecordsParsed > 0 for feed %s", entry.FeedID)
		}
	}

	// 4. Fallback to airgap-embedded digest when upstream HTTP fails
	failTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer failTS.Close()

	fallbackRcpt, err := feeds.SyncFeeds(ctx, feeds.SyncOptions{
		FeedID:          feeds.FeedCISAKEV,
		Offline:         false,
		BaseURL:         failTS.URL,
		CircuitRegistry: resilience.NewCircuitRegistry(resilience.CircuitConfig{FailureThreshold: 3}),
		RetryConfig:     &resilience.RetryConfig{MaxRetries: 2, BaseDelay: 2 * time.Millisecond, MaxDelay: 10 * time.Millisecond},
	})
	if err != nil {
		t.Fatalf("SyncFeeds fallback error: %v", err)
	}
	if len(fallbackRcpt.Feeds) != 1 || fallbackRcpt.Feeds[0].Mode != "airgap-embedded" {
		t.Errorf("expected graceful fallback to airgap-embedded on HTTP 503, got %+v", fallbackRcpt.Feeds)
	}
	if fallbackRcpt.OperationalMode != "DEGRADED" || fallbackRcpt.FallbackCount != 1 {
		t.Errorf("expected DEGRADED with FallbackCount=1, got %s / %d", fallbackRcpt.OperationalMode, fallbackRcpt.FallbackCount)
	}
}

func TestParseUpstreamCISAKEVAndMITREValidation(t *testing.T) {
	kevRaw := []byte(`{
		"catalogVersion": "2026.11.10",
		"vulnerabilities": [
			{
				"cveID": "CVE-2024-3400",
				"vendorProject": "Palo Alto Networks",
				"product": "PAN-OS",
				"vulnerabilityName": "PAN-OS Command Injection",
				"dateAdded": "2024-04-12",
				"dueDate": "2024-04-19",
				"knownRansomwareCampaignUse": "Known",
				"shortDescription": "Root command injection",
				"cwes": ["CWE-77"]
			},
			{
				"cveID": "CVE-2024-1709",
				"vendorProject": "ConnectWise",
				"product": "ScreenConnect",
				"vulnerabilityName": "Authentication Bypass",
				"cwes": ["CWE-287"]
			}
		]
	}`)

	entries, ver, err := feeds.ParseUpstreamCISAKEV(kevRaw)
	if err != nil {
		t.Fatalf("ParseUpstreamCISAKEV error: %v", err)
	}
	if ver != "2026.11.10" || len(entries) != 2 {
		t.Fatalf("unexpected KEV parse result: ver=%q count=%d", ver, len(entries))
	}
	if entries[0].CWEID != "CWE-77" || len(entries[0].AttackedControls) == 0 || len(entries[0].MitigatingControls) == 0 {
		t.Errorf("expected deterministic CWE->control mapping on CWE-77, got %+v", entries[0])
	}

	// Reject malformed CVE IDs, captive portals, and valid-but-wrong JSON
	for _, bad := range [][]byte{
		[]byte(`<html><body>Captive Portal Login</body></html>`),
		[]byte(`{"status":"ok","portal":true}`),
		[]byte(`{"catalogVersion":"1","vulnerabilities":[{"cveID":"NOT-A-CVE"}]}`),
		[]byte(`["array-not-object"]`),
	} {
		if _, _, err := feeds.ParseUpstreamCISAKEV(bad); err == nil {
			t.Errorf("expected ParseUpstreamCISAKEV to reject %q", string(bad))
		}
	}

	// MITRE STIX bundle validation
	stixRaw := []byte(`{"type":"bundle","objects":[{"type":"attack-pattern","id":"attack-pattern--1"},{"type":"relationship","id":"relationship--1"}]}`)
	cnt, err := feeds.ParseUpstreamMITREAttackMappings(stixRaw)
	if err != nil || cnt != 2 {
		t.Fatalf("expected 2 STIX objects, got cnt=%d err=%v", cnt, err)
	}
	if _, err := feeds.ParseUpstreamMITREAttackMappings([]byte(`{"captive_portal":true}`)); err == nil {
		t.Errorf("expected ParseUpstreamMITREAttackMappings to reject unrelated JSON object")
	}
}

func TestSyncFeedsRejectsCaptivePortalAndTruncatedPayloads(t *testing.T) {
	ctx := context.Background()

	// 1. Captive portal returning 200 OK HTML for JSON feeds
	portalTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><head><title>Airport WiFi</title></head><body>Please sign in to access the internet.</body></html>`))
	}))
	defer portalTS.Close()

	for _, feedID := range []string{feeds.FeedCrosswalk, feeds.FeedCISAKEV, feeds.FeedMITREAttack, feeds.FeedSovereignLifecycle} {
		rcpt, err := feeds.SyncFeeds(ctx, feeds.SyncOptions{
			FeedID:          feedID,
			Offline:         false,
			BaseURL:         portalTS.URL,
			CircuitRegistry: resilience.NewCircuitRegistry(resilience.CircuitConfig{FailureThreshold: 3}),
		})
		if err != nil {
			t.Fatalf("SyncFeeds(%s) unexpected error: %v", feedID, err)
		}
		e := rcpt.Feeds[0]
		if e.Mode != "airgap-embedded" || rcpt.OperationalMode != "DEGRADED" || e.DegradedReason == "" {
			t.Fatalf("feed %s accepted captive portal HTML instead of falling back: %+v", feedID, e)
		}
	}

	// 2. Oversized payload (> MaxPayloadBytes) must be rejected as truncated, NEVER hashed
	oversizedTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Emit 2 KiB valid JSON when MaxPayloadBytes is 512 bytes
		pad := strings.Repeat("a", 2048)
		_, _ = w.Write([]byte(`{"catalogVersion":"1.0","vulnerabilities":[{"cveID":"CVE-2024-3400","shortDescription":"` + pad + `"}]}`))
	}))
	defer oversizedTS.Close()

	kevDesc, _ := feeds.InspectFeed(feeds.FeedCISAKEV)
	truncRcpt, err := feeds.SyncFeeds(ctx, feeds.SyncOptions{
		FeedID:          feeds.FeedCISAKEV,
		Offline:         false,
		BaseURL:         oversizedTS.URL,
		MaxPayloadBytes: 512,
		CircuitRegistry: resilience.NewCircuitRegistry(resilience.CircuitConfig{FailureThreshold: 3}),
	})
	if err != nil {
		t.Fatalf("SyncFeeds truncation check error: %v", err)
	}
	fe := truncRcpt.Feeds[0]
	if fe.Mode != "airgap-embedded" {
		t.Fatalf("expected truncated feed to fall back to airgap-embedded, got %q", fe.Mode)
	}
	if fe.SHA256 != kevDesc.Descriptor.SHA256 {
		t.Fatalf("truncated bytes were hashed! got %s, want embedded %s", fe.SHA256, kevDesc.Descriptor.SHA256)
	}
	if !strings.Contains(fe.DegradedReason, "truncated") && !strings.Contains(fe.DegradedReason, "exceeded") {
		t.Errorf("expected truncation reason in DegradedReason, got %q", fe.DegradedReason)
	}
}

func TestSyncFeedsTransientRetryAndCircuitBreakerLifecycle(t *testing.T) {
	ctx := context.Background()

	// 1. Transient HTTP 503 x2 -> 200 OK on 3rd attempt recovers cleanly
	var transientCalls atomic.Int32
	transientTS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := transientCalls.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"catalogVersion":"2026.11.10","vulnerabilities":[{"cveID":"CVE-2024-3400","cwes":["CWE-77"]}]}`))
	}))
	defer transientTS.Close()

	recoveredRcpt, err := feeds.SyncFeeds(ctx, feeds.SyncOptions{
		FeedID:          feeds.FeedCISAKEV,
		Offline:         false,
		BaseURL:         transientTS.URL,
		CircuitRegistry: resilience.NewCircuitRegistry(resilience.CircuitConfig{FailureThreshold: 3}),
		RetryConfig:     &resilience.RetryConfig{MaxRetries: 2, BaseDelay: 2 * time.Millisecond, MaxDelay: 10 * time.Millisecond},
	})
	if err != nil {
		t.Fatalf("SyncFeeds transient recovery error: %v", err)
	}
	if recoveredRcpt.OperationalMode != "NORMAL" || recoveredRcpt.Feeds[0].Mode != "live-http" || recoveredRcpt.Feeds[0].Attempts != 3 {
		t.Fatalf("expected live-http recovery on attempt 3, got %+v", recoveredRcpt.Feeds[0])
	}

	// 2. Permanent HTTP 404 does NOT retry (Attempts == 1), trips circuit after FailureThreshold=2, then recovers after cooldown
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	var healthy atomic.Bool
	var upstreamHits atomic.Int32

	cbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		upstreamHits.Add(1)
		if !healthy.Load() {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"catalogVersion":"2026.11.10","vulnerabilities":[{"cveID":"CVE-2024-3400","cwes":["CWE-77"]}]}`))
	}))
	defer cbServer.Close()

	reg := resilience.NewCircuitRegistry(resilience.CircuitConfig{
		FailureThreshold: 2,
		Cooldown:         30 * time.Second,
		Now: func() time.Time {
			mu.Lock()
			defer mu.Unlock()
			return now
		},
	})

	// Failure 1 -> CLOSED, Attempts=1 (no retry on 404)
	r1, _ := feeds.SyncFeeds(ctx, feeds.SyncOptions{FeedID: feeds.FeedCISAKEV, Offline: false, BaseURL: cbServer.URL, CircuitRegistry: reg})
	if r1.Feeds[0].Attempts != 1 || r1.Feeds[0].CircuitState != resilience.StateClosed {
		t.Fatalf("expected Attempts=1 CircuitState=CLOSED on first 404, got %+v", r1.Feeds[0])
	}

	// Failure 2 -> trips to OPEN
	r2, _ := feeds.SyncFeeds(ctx, feeds.SyncOptions{FeedID: feeds.FeedCISAKEV, Offline: false, BaseURL: cbServer.URL, CircuitRegistry: reg})
	if r2.Feeds[0].CircuitState != resilience.StateOpen {
		t.Fatalf("expected CircuitState=OPEN after 2nd failure, got %+v", r2.Feeds[0])
	}

	// Sync 3 while OPEN -> fast-fails without hitting upstream server (Attempts=0)
	hitsBefore := upstreamHits.Load()
	r3, _ := feeds.SyncFeeds(ctx, feeds.SyncOptions{FeedID: feeds.FeedCISAKEV, Offline: false, BaseURL: cbServer.URL, CircuitRegistry: reg})
	if r3.Feeds[0].Attempts != 0 || upstreamHits.Load() != hitsBefore {
		t.Fatalf("expected fast-fail with 0 upstream hits while OPEN, got attempts=%d hitsDelta=%d", r3.Feeds[0].Attempts, upstreamHits.Load()-hitsBefore)
	}

	// Advance clock past cooldown + mark upstream healthy -> HALF_OPEN probe succeeds -> CLOSED
	mu.Lock()
	now = now.Add(35 * time.Second)
	mu.Unlock()
	healthy.Store(true)

	r4, _ := feeds.SyncFeeds(ctx, feeds.SyncOptions{FeedID: feeds.FeedCISAKEV, Offline: false, BaseURL: cbServer.URL, CircuitRegistry: reg})
	if r4.OperationalMode != "NORMAL" || r4.Feeds[0].Mode != "live-http" || r4.Feeds[0].CircuitState != resilience.StateClosed {
		t.Fatalf("expected HALF_OPEN -> CLOSED recovery, got %+v", r4.Feeds[0])
	}
}

func TestSyncCheckpointAndFlightRecorderTamperDetection(t *testing.T) {
	ctx := context.Background()
	cpDir := filepath.Join(t.TempDir(), "checkpoints")

	// Execute 3 sync cycles writing to the same checkpoint directory
	var lastRcpt *feeds.FeedSyncReceipt
	for i := 0; i < 3; i++ {
		rcpt, err := feeds.SyncFeeds(ctx, feeds.SyncOptions{
			Offline:       true,
			CheckpointDir: cpDir,
		})
		if err != nil {
			t.Fatalf("SyncFeeds cycle %d error: %v", i+1, err)
		}
		lastRcpt = rcpt
	}

	verifiedRcpt, count, err := feeds.VerifyCheckpointDir(cpDir)
	if err != nil {
		t.Fatalf("VerifyCheckpointDir error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 hash-chained ledger entries, got %d", count)
	}
	if verifiedRcpt.ReceiptID != lastRcpt.ReceiptID {
		t.Fatalf("ReceiptID mismatch: got %s, want %s", verifiedRcpt.ReceiptID, lastRcpt.ReceiptID)
	}

	// 1. Tamper 1 byte in latest_sync_receipt.json -> must fail with ErrIntegrityTampered
	latestPath := filepath.Join(cpDir, "latest_sync_receipt.json")
	origLatest, err := os.ReadFile(latestPath)
	if err != nil {
		t.Fatal(err)
	}
	tamperedLatest := append([]byte(nil), origLatest...)
	idx := bytes.Index(tamperedLatest, []byte("rcpt_feeds_"))
	if idx < 0 {
		t.Fatal("missing rcpt_feeds_ in latest_sync_receipt.json")
	}
	tamperedLatest[idx] = 'X'
	if err := os.WriteFile(latestPath, tamperedLatest, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := feeds.VerifyCheckpointDir(cpDir); !errors.Is(err, feeds.ErrIntegrityTampered) {
		t.Fatalf("expected ErrIntegrityTampered when latest_sync_receipt.json modified, got %v", err)
	}
	// Restore latest_sync_receipt.json
	if err := os.WriteFile(latestPath, origLatest, 0o600); err != nil {
		t.Fatal(err)
	}

	// 2. Tamper 1 byte in sync_flight_recorder.jsonl -> must fail with ErrIntegrityTampered
	ledgerPath := filepath.Join(cpDir, "sync_flight_recorder.jsonl")
	origLedger, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	tamperedLedger := bytes.Replace(origLedger, []byte(`"NORMAL"`), []byte(`"HACKED"`), 1)
	if err := os.WriteFile(ledgerPath, tamperedLedger, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := feeds.VerifyCheckpointDir(cpDir); !errors.Is(err, feeds.ErrIntegrityTampered) {
		t.Fatalf("expected ErrIntegrityTampered when flight recorder ledger modified, got %v", err)
	}
}

func TestSyncFeedsCallerContextCancelDoesNotTripBreaker(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer ts.Close()

	reg := resilience.NewCircuitRegistry(resilience.CircuitConfig{
		FailureThreshold: 2,
		Cooldown:         30 * time.Second,
	})

	// 5 consecutive caller-cancelled requests must NOT trip the shared CircuitBreaker to OPEN
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		rcpt, err := feeds.SyncFeeds(ctx, feeds.SyncOptions{
			FeedID:          feeds.FeedCISAKEV,
			Offline:         false,
			BaseURL:         ts.URL,
			CircuitRegistry: reg,
			RetryConfig:     &resilience.RetryConfig{MaxRetries: 1, BaseDelay: 5 * time.Millisecond},
		})
		cancel()
		if err != nil {
			t.Fatalf("SyncFeeds cancelled ctx iter %d error: %v", i, err)
		}
		if rcpt.Feeds[0].CircuitState != resilience.StateClosed {
			t.Fatalf("caller context cancellation falsely tripped circuit breaker to %s on iter %d", rcpt.Feeds[0].CircuitState, i)
		}
	}
}

func TestHelperProcessCheckpointSync(t *testing.T) {
	cpDir := os.Getenv("XOSCAL_TEST_CHECKPOINT_DIR")
	if cpDir == "" {
		return
	}
	for i := 0; i < 5; i++ {
		if _, err := feeds.SyncFeeds(context.Background(), feeds.SyncOptions{
			Offline:       true,
			CheckpointDir: cpDir,
		}); err != nil {
			t.Fatalf("child process SyncFeeds iter %d failed: %v", i, err)
		}
	}
}

func TestSyncCheckpointMultiProcessAndConcurrentReaderContention(t *testing.T) {
	cpDir := filepath.Join(t.TempDir(), "multiprocess_cp")

	// 1. Concurrent in-process writers + concurrent LoadVerifiedCheckpoint / VerifyCheckpointDir readers
	if _, err := feeds.SyncFeeds(context.Background(), feeds.SyncOptions{Offline: true, CheckpointDir: cpDir}); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	stopReaders := make(chan struct{})
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopReaders:
					return
				default:
					if _, err := feeds.LoadVerifiedCheckpoint(cpDir); err != nil {
						t.Errorf("concurrent LoadVerifiedCheckpoint torn read error: %v", err)
						return
					}
					if _, _, err := feeds.VerifyCheckpointDir(cpDir); err != nil {
						t.Errorf("concurrent VerifyCheckpointDir torn read error: %v", err)
						return
					}
				}
			}
		}()
	}

	var writerWG sync.WaitGroup
	for w := 0; w < 4; w++ {
		writerWG.Add(1)
		go func() {
			defer writerWG.Done()
			for i := 0; i < 5; i++ {
				if _, err := feeds.SyncFeeds(context.Background(), feeds.SyncOptions{Offline: true, CheckpointDir: cpDir}); err != nil {
					t.Errorf("concurrent in-process SyncFeeds error: %v", err)
					return
				}
			}
		}()
	}
	writerWG.Wait()
	close(stopReaders)
	wg.Wait()

	// 2. Spawn 4 parallel OS child processes writing 5 checkpoints each (20 more entries -> 1 + 20 + 20 = 41 total)
	const numProcs = 4
	var procWG sync.WaitGroup
	errCh := make(chan error, numProcs)
	for p := 0; p < numProcs; p++ {
		procWG.Add(1)
		go func() {
			defer procWG.Done()
			// #nosec G204 -- invokes current test binary for multi-process flock verification.
			cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcessCheckpointSync$")
			cmd.Env = append(os.Environ(), "XOSCAL_TEST_CHECKPOINT_DIR="+cpDir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				errCh <- errors.New(string(out) + ": " + err.Error())
			}
		}()
	}
	procWG.Wait()
	close(errCh)
	for procErr := range errCh {
		t.Fatalf("multi-process checkpoint child failed: %v", procErr)
	}

	_, totalCount, err := feeds.VerifyCheckpointDir(cpDir)
	if err != nil {
		t.Fatalf("VerifyCheckpointDir after multi-process contention failed: %v", err)
	}
	if totalCount != 41 {
		t.Fatalf("expected 41 serialized hash-chained ledger entries (1 + 20 goroutines + 20 OS child writes), got %d", totalCount)
	}
}

func TestSyncFeedsLivePersistenceAndActiveAdvisories(t *testing.T) {
	t.Cleanup(feeds.ResetSyncedCache)
	feeds.ResetSyncedCache()

	cpDir := filepath.Join(t.TempDir(), "live_synced_cp")
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
						"vendorProject":"SyntheticEdge",
						"product":"GatewayOS",
						"vulnerabilityName":"Synthetic Zero-Day Command Injection",
						"dateAdded":"2026-11-10",
						"dueDate":"2026-11-17",
						"knownRansomwareCampaignUse":"Known",
						"shortDescription":"Novel live-synced CVE for persistence verification",
						"cwes":["CWE-78"]
					}
				]
			}`))
		case strings.HasSuffix(r.URL.Path, "/"+feeds.FeedMITREAttack):
			_, _ = w.Write([]byte(`{
				"type":"bundle",
				"id":"bundle--live-test",
				"objects":[
					{
						"type":"attack-pattern",
						"id":"attack-pattern--t1999",
						"name":"Synthetic Container Escape",
						"description":"Novel adversary technique synced live",
						"external_references":[{"source_name":"mitre-attack","external_id":"T1999"}],
						"kill_chain_phases":[{"kill_chain_name":"mitre-attack","phase_name":"privilege-escalation"}]
					},
					{
						"type":"course-of-action",
						"id":"course-of-action--si2",
						"name":"SI-2 Flaw Remediation",
						"external_references":[{"source_name":"NIST 800-53","external_id":"SI-2"}]
					},
					{
						"type":"relationship",
						"id":"relationship--mit1",
						"relationship_type":"mitigates",
						"source_ref":"course-of-action--si2",
						"target_ref":"attack-pattern--t1999"
					}
				]
			}`))
		case strings.HasSuffix(r.URL.Path, "/"+feeds.FeedCrosswalk):
			_, _ = w.Write([]byte(`{"catalog":{"uuid":"11111111-2222-3333-4444-555555555555","controls":[{"id":"ac-2"}]}}`))
		default:
			_, _ = w.Write([]byte(`<html><body>CCCS ITSP.10.033 Sovereign Security Control Catalogue Effective 2026-03-31</body></html>`))
		}
	}))
	defer ts.Close()

	rcpt, err := feeds.SyncFeeds(context.Background(), feeds.SyncOptions{
		Offline:         false,
		HTTPClient:      ts.Client(),
		BaseURL:         ts.URL,
		CheckpointDir:   cpDir,
		CircuitRegistry: resilience.NewCircuitRegistry(resilience.CircuitConfig{FailureThreshold: 3}),
	})
	if err != nil {
		t.Fatalf("SyncFeeds live persistence error: %v", err)
	}
	if rcpt.OperationalMode != "NORMAL" {
		t.Fatalf("expected OperationalMode NORMAL, got %s", rcpt.OperationalMode)
	}

	// Verify synced_cisa_kev.json and synced_mitre_attack.json + .sha256 sidecars exist and pass ReadVerifiedFile
	for _, fn := range []string{feeds.SyncedCISAKEVFile, feeds.SyncedMITREAttackFile} {
		if _, _, err := resilience.ReadVerifiedFile(filepath.Join(cpDir, fn), feeds.DefaultMaxPayloadBytes); err != nil {
			t.Fatalf("ReadVerifiedFile(%s) failed: %v", fn, err)
		}
	}

	// Clear in-memory cache so ActiveThreatAdvisories / ActiveAttackMappings must read from verified checkpoint files
	feeds.ResetSyncedCache()

	kevActive := feeds.ActiveThreatAdvisories(cpDir)
	foundCVE := false
	for _, adv := range kevActive {
		if adv.CVEID == "CVE-2099-9999" {
			foundCVE = true
			if adv.CWEID != "CWE-78" || len(adv.MitigatingControls) == 0 {
				t.Errorf("unexpected CVE-2099-9999 mapping: %+v", adv)
			}
			break
		}
	}
	if !foundCVE {
		t.Fatalf("expected novel CVE-2099-9999 in ActiveThreatAdvisories(%s)", cpDir)
	}

	attActive := feeds.ActiveAttackMappings(cpDir)
	foundTech := false
	for _, tech := range attActive {
		if tech.TechniqueID == "T1999" {
			foundTech = true
			if len(tech.MitigatingControls) == 0 || tech.MitigatingControls[0] != "si-2" {
				t.Errorf("unexpected T1999 mitigating controls: %+v", tech)
			}
			break
		}
	}
	if !foundTech {
		t.Fatalf("expected novel T1999 in ActiveAttackMappings(%s)", cpDir)
	}
}
