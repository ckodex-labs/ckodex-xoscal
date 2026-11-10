package resilience_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mchorfa/xoscal/server/internal/resilience"
)

func TestCircuitBreakerStateMachine(t *testing.T) {
	now := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	var clockMu sync.Mutex
	clock := func() time.Time {
		clockMu.Lock()
		defer clockMu.Unlock()
		return now
	}
	advance := func(d time.Duration) {
		clockMu.Lock()
		now = now.Add(d)
		clockMu.Unlock()
	}

	cb := resilience.NewCircuitBreaker(resilience.CircuitConfig{
		Target:           "cisa-kev",
		FailureThreshold: 3,
		Cooldown:         30 * time.Second,
		Now:              clock,
	})

	// 1. Starts CLOSED
	if ok, st := cb.Allow(); !ok || st != resilience.StateClosed {
		t.Fatalf("expected initial Allow=(true, CLOSED), got (%v, %s)", ok, st)
	}

	// 2. Two failures keep circuit CLOSED
	cb.RecordFailure()
	cb.RecordFailure()
	if snap := cb.Snapshot(); snap.State != resilience.StateClosed || snap.ConsecutiveFailures != 2 {
		t.Fatalf("expected CLOSED with 2 failures, got %+v", snap)
	}

	// 3. Third failure trips circuit to OPEN
	cb.RecordFailure()
	if snap := cb.Snapshot(); snap.State != resilience.StateOpen || snap.ConsecutiveFailures != 3 {
		t.Fatalf("expected OPEN with 3 failures, got %+v", snap)
	}

	// 4. Fast-fail while OPEN before cooldown expires
	advance(15 * time.Second)
	if ok, st := cb.Allow(); ok || st != resilience.StateOpen {
		t.Fatalf("expected fast-fail Allow=(false, OPEN) during cooldown, got (%v, %s)", ok, st)
	}

	// 5. Advance past cooldown -> transitions OPEN -> HALF_OPEN and allows single probe
	advance(16 * time.Second)
	if ok, st := cb.Allow(); !ok || st != resilience.StateHalfOpen {
		t.Fatalf("expected probe Allow=(true, HALF_OPEN) after cooldown, got (%v, %s)", ok, st)
	}
	// Second concurrent probe while HALF_OPEN probe is active must be blocked
	if ok, st := cb.Allow(); ok || st != resilience.StateHalfOpen {
		t.Fatalf("expected second concurrent probe blocked (false, HALF_OPEN), got (%v, %s)", ok, st)
	}

	// 6. Failure in HALF_OPEN immediately re-opens circuit
	cb.RecordFailure()
	if snap := cb.Snapshot(); snap.State != resilience.StateOpen {
		t.Fatalf("expected immediate transition back to OPEN on HALF_OPEN failure, got %+v", snap)
	}
	if ok, st := cb.Allow(); ok || st != resilience.StateOpen {
		t.Fatalf("expected Allow=(false, OPEN) after HALF_OPEN failure, got (%v, %s)", ok, st)
	}

	// 7. Advance past cooldown -> HALF_OPEN -> CancelProbe releases slot without tripping/closing -> stalled probe lease expires after Cooldown
	advance(31 * time.Second)
	if ok, st := cb.Allow(); !ok || st != resilience.StateHalfOpen {
		t.Fatalf("expected Allow=(true, HALF_OPEN), got (%v, %s)", ok, st)
	}
	cb.CancelProbe()
	if ok, st := cb.Allow(); !ok || st != resilience.StateHalfOpen {
		t.Fatalf("expected Allow=(true, HALF_OPEN) immediately after CancelProbe, got (%v, %s)", ok, st)
	}
	// Simulate hung/abandoned probe: blocked before Cooldown, then re-acquired after Cooldown so breaker never deadlocks
	if ok, st := cb.Allow(); ok || st != resilience.StateHalfOpen {
		t.Fatalf("expected concurrent probe blocked during active lease, got (%v, %s)", ok, st)
	}
	advance(31 * time.Second)
	if ok, st := cb.Allow(); !ok || st != resilience.StateHalfOpen {
		t.Fatalf("expected stalled HALF_OPEN probe lease to expire after Cooldown, got (%v, %s)", ok, st)
	}
	cb.RecordSuccess()
	snap := cb.Snapshot()
	if snap.State != resilience.StateClosed || snap.ConsecutiveFailures != 0 {
		t.Fatalf("expected CLOSED with 0 failures after RecordSuccess, got %+v", snap)
	}
	if ok, st := cb.Allow(); !ok || st != resilience.StateClosed {
		t.Fatalf("expected Allow=(true, CLOSED) after recovery, got (%v, %s)", ok, st)
	}
}

func TestRetryWithBackoff(t *testing.T) {
	// 1. Transient failure recovers on retry
	calls := 0
	attempts, err := resilience.RetryWithBackoff(context.Background(), resilience.RetryConfig{
		MaxRetries: 2,
		BaseDelay:  5 * time.Millisecond,
		MaxDelay:   20 * time.Millisecond,
	}, func(_ context.Context, attempt int) error {
		calls++
		if attempt < 3 {
			return resilience.MarkRetryable(errors.New("http 503 transient"))
		}
		return nil
	})
	if err != nil || attempts != 3 || calls != 3 {
		t.Fatalf("expected recovery on attempt 3, got attempts=%d calls=%d err=%v", attempts, calls, err)
	}

	// 2. Permanent (non-retryable) error stops immediately after 1 attempt
	permCalls := 0
	permAttempts, permErr := resilience.RetryWithBackoff(context.Background(), resilience.RetryConfig{
		MaxRetries: 3,
		BaseDelay:  5 * time.Millisecond,
	}, func(_ context.Context, _ int) error {
		permCalls++
		return errors.New("http 404 not found")
	})
	if permErr == nil || permAttempts != 1 || permCalls != 1 {
		t.Fatalf("expected 1 attempt on permanent error, got attempts=%d calls=%d err=%v", permAttempts, permCalls, permErr)
	}

	// 2b. Negative MaxRetries disables retries cleanly (1 attempt total)
	noRetryAttempts, noRetryErr := resilience.RetryWithBackoff(context.Background(), resilience.RetryConfig{
		MaxRetries: -1,
	}, func(_ context.Context, _ int) error {
		return resilience.MarkRetryable(errors.New("transient"))
	})
	if noRetryErr == nil || noRetryAttempts != 1 {
		t.Fatalf("expected 1 attempt when MaxRetries=-1, got attempts=%d err=%v", noRetryAttempts, noRetryErr)
	}

	// 3. Context cancellation aborts immediately during backoff wait
	ctx, cancel := context.WithCancel(context.Background())
	start := time.Now()
	cancelAttempts, cancelErr := resilience.RetryWithBackoff(ctx, resilience.RetryConfig{
		MaxRetries: 5,
		BaseDelay:  5 * time.Second,
		MaxDelay:   10 * time.Second,
	}, func(_ context.Context, _ int) error {
		go func() {
			time.Sleep(15 * time.Millisecond)
			cancel()
		}()
		return resilience.MarkRetryable(errors.New("upstream timeout"))
	})
	elapsed := time.Since(start)
	if !errors.Is(cancelErr, context.Canceled) {
		t.Fatalf("expected context.Canceled error, got %v", cancelErr)
	}
	if cancelAttempts != 1 {
		t.Fatalf("expected 1 attempt before ctx cancel, got %d", cancelAttempts)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("expected fast cancellation during backoff, took %v", elapsed)
	}
}

func TestAtomicFilePersistenceAndTamperDetection(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "checkpoint.json")
	payload := []byte(`{"receipt_id":"rcpt_123","status":"coherent"}`)

	digest, err := resilience.WriteFileAtomicWithSHA256(target, payload)
	if err != nil {
		t.Fatalf("WriteFileAtomicWithSHA256 error: %v", err)
	}
	if len(digest) != 64 {
		t.Fatalf("expected 64-hex sha256 digest, got %q", digest)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected 0600 permissions, got %o", info.Mode().Perm())
	}

	// Verified read succeeds
	gotData, gotDigest, err := resilience.ReadVerifiedFile(target, 1024)
	if err != nil {
		t.Fatalf("ReadVerifiedFile error: %v", err)
	}
	if string(gotData) != string(payload) || gotDigest != digest {
		t.Fatalf("ReadVerifiedFile mismatch: got (%q, %s)", string(gotData), gotDigest)
	}

	// Reject oversized file when maxBytes is smaller than file size
	if _, _, err := resilience.ReadVerifiedFile(target, 10); !errors.Is(err, resilience.ErrPayloadTooLarge) {
		t.Fatalf("expected ErrPayloadTooLarge, got %v", err)
	}

	// Flip 1 byte in file -> must return ErrIntegrityTampered
	tampered := append([]byte(nil), payload...)
	tampered[len(tampered)-2] ^= 0x01
	if err := os.WriteFile(target, tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := resilience.ReadVerifiedFile(target, 1024); !errors.Is(err, resilience.ErrIntegrityTampered) {
		t.Fatalf("expected ErrIntegrityTampered on 1-byte flip, got %v", err)
	}

	// Restore valid file, then delete .sha256 sidecar -> must return ErrIntegrityTampered
	if err := os.WriteFile(target, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(target + ".sha256"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := resilience.ReadVerifiedFile(target, 1024); !errors.Is(err, resilience.ErrIntegrityTampered) {
		t.Fatalf("expected ErrIntegrityTampered when sidecar is missing, got %v", err)
	}

	// Replace .sha256 sidecar with a directory -> must return ErrIntegrityTampered
	if err := os.Mkdir(target+".sha256", 0o750); err != nil {
		t.Fatal(err)
	}
	if _, _, err := resilience.ReadVerifiedFile(target, 1024); !errors.Is(err, resilience.ErrIntegrityTampered) {
		t.Fatalf("expected ErrIntegrityTampered when sidecar is a directory, got %v", err)
	}
}

func TestWrapHTTPHandler_PanicRecoveryAndConcurrencyBackpressure(t *testing.T) {
	blockCh := make(chan struct{})
	enteredCh := make(chan struct{}, 4)

	mux := http.NewServeMux()
	mux.HandleFunc("/panic", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "999999")
		w.Header().Set("Content-Encoding", "gzip")
		panic("simulated nil pointer dereference in secret internal module")
	})
	mux.HandleFunc("/slow", func(w http.ResponseWriter, _ *http.Request) {
		enteredCh <- struct{}{}
		<-blockCh
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/ok", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	wrapped := resilience.WrapHTTPHandler(mux, 2)

	// 1. Baseline headers + X-Request-ID propagation
	okReq := httptest.NewRequest(http.MethodGet, "/ok", nil)
	okReq.Header.Set("X-Request-ID", "client-trace-42")
	okRec := httptest.NewRecorder()
	wrapped.ServeHTTP(okRec, okReq)
	if okRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", okRec.Code)
	}
	if got := okRec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := okRec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", got)
	}
	if got := okRec.Header().Get("X-Request-ID"); got != "client-trace-42" {
		t.Errorf("X-Request-ID = %q, want client-trace-42", got)
	}

	// 2. Panic recovery returns clean 500 JSON without leaking panic details, bogus Content-Length, or semaphore slots
	for i := 0; i < 5; i++ {
		pReq := httptest.NewRequest(http.MethodGet, "/panic", nil)
		pRec := httptest.NewRecorder()
		wrapped.ServeHTTP(pRec, pReq)
		if pRec.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 on panic (iter %d), got %d", i, pRec.Code)
		}
		if got := pRec.Header().Get("Content-Encoding"); got != "" {
			t.Errorf("expected Content-Encoding cleared on panic recovery, got %q", got)
		}
		if got := pRec.Header().Get("Content-Length"); got == "999999" {
			t.Errorf("expected stale Content-Length cleared on panic recovery, got %q", got)
		}
		body := pRec.Body.String()
		if strings.Contains(body, "secret internal module") || strings.Contains(body, "goroutine") {
			t.Fatalf("panic handler leaked internal details: %s", body)
		}
		var errDoc map[string]string
		if err := json.Unmarshal(pRec.Body.Bytes(), &errDoc); err != nil {
			t.Fatalf("decode 500 json: %v", err)
		}
		if errDoc["error"] != "internal server error" || errDoc["request_id"] == "" {
			t.Errorf("unexpected 500 error body: %+v", errDoc)
		}
	}

	// 3. Concurrency saturation backpressure (maxInFlight = 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/slow", nil)
			wrapped.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("expected in-flight slow request to complete with 200, got %d", rec.Code)
			}
		}()
	}

	// Wait until both concurrency slots are occupied
	<-enteredCh
	<-enteredCh

	// Third concurrent request must immediately receive 503 + Retry-After: 1
	satRec := httptest.NewRecorder()
	satReq := httptest.NewRequest(http.MethodGet, "/ok", nil)
	wrapped.ServeHTTP(satRec, satReq)
	if satRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable when saturated, got %d", satRec.Code)
	}
	if got := satRec.Header().Get("Retry-After"); got != "1" {
		t.Errorf("Retry-After = %q, want '1'", got)
	}
	if !strings.Contains(satRec.Body.String(), `"retry_after_seconds":1`) {
		t.Errorf("expected retry_after_seconds in 503 JSON, got %s", satRec.Body.String())
	}

	// Release slow handlers and verify recovery
	close(blockCh)
	wg.Wait()

	afterRec := httptest.NewRecorder()
	afterReq := httptest.NewRequest(http.MethodGet, "/ok", nil)
	wrapped.ServeHTTP(afterRec, afterReq)
	if afterRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK after saturation cleared, got %d", afterRec.Code)
	}
}
