// Package resilience provides bounded Day-2 operational primitives (CKODEX Rules 28-33):
// per-target 3-state circuit breakers, bounded exponential backoff with full jitter,
// crash-safe atomic file persistence with SHA-256 sidecars, and resilient HTTP
// middleware with panic recovery and concurrency backpressure.
package resilience

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Circuit breaker canonical states.
const (
	StateClosed   = "CLOSED"
	StateOpen     = "OPEN"
	StateHalfOpen = "HALF_OPEN"
)

// Default resilience thresholds and bounds.
const (
	DefaultFailureThreshold = 3
	DefaultCooldown         = 30 * time.Second
	DefaultMaxRetries       = 2
	DefaultBaseDelay        = 100 * time.Millisecond
	DefaultMaxDelay         = 2 * time.Second
	DefaultMaxInFlight      = 128
	DefaultMaxVerifiedBytes = 64 << 20 // 64 MiB
)

var (
	// ErrIntegrityTampered indicates a missing/corrupted SHA-256 sidecar or content digest mismatch.
	ErrIntegrityTampered = errors.New("integrity tampered: sha256 digest mismatch or missing sidecar")
	// ErrPayloadTooLarge indicates an input file or stream exceeded the configured byte ceiling.
	ErrPayloadTooLarge = errors.New("payload exceeds maximum allowed byte bound")
)

// CircuitConfig configures a per-target 3-state CircuitBreaker.
type CircuitConfig struct {
	Target           string
	FailureThreshold int
	Cooldown         time.Duration
	Now              func() time.Time
}

// CircuitSnapshot captures an immutable point-in-time view of a CircuitBreaker.
type CircuitSnapshot struct {
	Target              string    `json:"target"`
	State               string    `json:"state"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	OpenedAt            time.Time `json:"opened_at,omitempty"`
	LastFailureAt       time.Time `json:"last_failure_at,omitempty"`
	LastSuccessAt       time.Time `json:"last_success_at,omitempty"`
}

// CircuitBreaker implements a thread-safe 3-state circuit breaker (CLOSED -> OPEN -> HALF_OPEN).
type CircuitBreaker struct {
	mu                  sync.Mutex
	target              string
	state               string
	failures            int
	failureThreshold    int
	cooldown            time.Duration
	openedAt            time.Time
	probeStartedAt      time.Time
	lastFailureAt       time.Time
	lastSuccessAt       time.Time
	halfOpenProbeActive bool
	now                 func() time.Time
}

// NewCircuitBreaker constructs a CircuitBreaker with sensible defaults.
func NewCircuitBreaker(cfg CircuitConfig) *CircuitBreaker {
	threshold := cfg.FailureThreshold
	if threshold <= 0 {
		threshold = DefaultFailureThreshold
	}
	cooldown := cfg.Cooldown
	if cooldown <= 0 {
		cooldown = DefaultCooldown
	}
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	return &CircuitBreaker{
		target:           cfg.Target,
		state:            StateClosed,
		failureThreshold: threshold,
		cooldown:         cooldown,
		now:              nowFn,
	}
}

// Allow reports whether a request to the target is permitted and returns the effective circuit state.
// When OPEN and Cooldown has elapsed, Allow transitions the breaker to HALF_OPEN and permits a single probe.
// If a HALF_OPEN probe stalls longer than Cooldown without recording an outcome, its lease expires so the breaker cannot deadlock.
func (cb *CircuitBreaker) Allow() (bool, string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := cb.now()
	switch cb.state {
	case StateClosed:
		return true, StateClosed
	case StateOpen:
		if now.Sub(cb.openedAt) >= cb.cooldown {
			cb.state = StateHalfOpen
			cb.halfOpenProbeActive = true
			cb.probeStartedAt = now
			return true, StateHalfOpen
		}
		return false, StateOpen
	case StateHalfOpen:
		if !cb.halfOpenProbeActive || now.Sub(cb.probeStartedAt) >= cb.cooldown {
			cb.halfOpenProbeActive = true
			cb.probeStartedAt = now
			return true, StateHalfOpen
		}
		return false, StateHalfOpen
	default:
		return true, StateClosed
	}
}

// CancelProbe releases an active HALF_OPEN probe lease when the caller aborts locally
// (e.g., caller context cancelled) without testing the upstream target.
func (cb *CircuitBreaker) CancelProbe() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen && cb.halfOpenProbeActive {
		cb.halfOpenProbeActive = false
		cb.probeStartedAt = time.Time{}
	}
}

// RecordSuccess resets consecutive failures and closes the circuit.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = StateClosed
	cb.halfOpenProbeActive = false
	cb.probeStartedAt = time.Time{}
	cb.lastSuccessAt = cb.now()
}

// RecordFailure increments consecutive failures and opens the circuit on threshold or immediately from HALF_OPEN.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := cb.now()
	cb.failures++
	cb.lastFailureAt = now

	if cb.state == StateHalfOpen || cb.failures >= cb.failureThreshold {
		cb.state = StateOpen
		cb.openedAt = now
		cb.halfOpenProbeActive = false
		cb.probeStartedAt = time.Time{}
	}
}

// Snapshot returns a thread-safe snapshot of the circuit breaker's state.
func (cb *CircuitBreaker) Snapshot() CircuitSnapshot {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return CircuitSnapshot{
		Target:              cb.target,
		State:               cb.state,
		ConsecutiveFailures: cb.failures,
		OpenedAt:            cb.openedAt,
		LastFailureAt:       cb.lastFailureAt,
		LastSuccessAt:       cb.lastSuccessAt,
	}
}

// CircuitRegistry manages per-target CircuitBreakers safely across goroutines.
type CircuitRegistry struct {
	mu         sync.Mutex
	defaultCfg CircuitConfig
	breakers   map[string]*CircuitBreaker
}

// NewCircuitRegistry creates a new per-target CircuitBreaker registry.
func NewCircuitRegistry(defaultCfg CircuitConfig) *CircuitRegistry {
	return &CircuitRegistry{
		defaultCfg: defaultCfg,
		breakers:   make(map[string]*CircuitBreaker),
	}
}

// Get returns or initializes the CircuitBreaker for target.
func (r *CircuitRegistry) Get(target string) *CircuitBreaker {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cb, ok := r.breakers[target]; ok {
		return cb
	}
	cfg := r.defaultCfg
	cfg.Target = target
	cb := NewCircuitBreaker(cfg)
	r.breakers[target] = cb
	return cb
}

// RetryableError wraps a transient error so RetryWithBackoff knows it is safe to retry.
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// MarkRetryable wraps err as a transient retryable error.
func MarkRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err}
}

// IsRetryable reports whether err is marked as retryable.
func IsRetryable(err error) bool {
	var re *RetryableError
	return errors.As(err, &re)
}

// RetryConfig configures bounded exponential backoff with full jitter.
type RetryConfig struct {
	MaxRetries     int
	MaxAttempts    int
	BaseDelay      time.Duration
	MaxDelay       time.Duration
	DisableRetries bool
}

// DefaultRetryConfig returns the canonical bounded retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: DefaultMaxRetries,
		BaseDelay:  DefaultBaseDelay,
		MaxDelay:   DefaultMaxDelay,
	}
}

// NoRetryConfig returns a RetryConfig that executes a single attempt with 0 retries.
func NoRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:    1,
		DisableRetries: true,
	}
}

// RetryWithBackoff executes fn up to 1 + MaxRetries times when fn returns a RetryableError,
// sleeping with exponential backoff + full jitter between attempts and aborting immediately if ctx is cancelled.
func RetryWithBackoff(ctx context.Context, cfg RetryConfig, fn func(ctx context.Context, attempt int) error) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	var maxRetries int
	switch {
	case cfg.DisableRetries || cfg.MaxRetries < 0 || cfg.MaxAttempts == 1:
		maxRetries = 0
	case cfg.MaxAttempts > 1:
		maxRetries = cfg.MaxAttempts - 1
	case cfg == (RetryConfig{}):
		maxRetries = DefaultMaxRetries
	default:
		maxRetries = cfg.MaxRetries
	}

	baseDelay := cfg.BaseDelay
	if baseDelay <= 0 {
		baseDelay = DefaultBaseDelay
	}
	maxDelay := cfg.MaxDelay
	if maxDelay <= 0 {
		maxDelay = DefaultMaxDelay
	}
	if maxDelay < baseDelay {
		maxDelay = baseDelay
	}

	attempts := 0
	for attempt := 1; attempt <= maxRetries+1; attempt++ {
		if err := ctx.Err(); err != nil {
			return attempts, err
		}
		attempts++
		err := fn(ctx, attempt)
		if err == nil {
			return attempts, nil
		}
		if !IsRetryable(err) || attempt > maxRetries {
			return attempts, err
		}

		shift := attempt - 1
		if shift > 30 {
			shift = 30
		}
		capDelay := baseDelay << shift
		if capDelay <= 0 || capDelay > maxDelay {
			capDelay = maxDelay
		}

		wait := fullJitterDuration(capDelay)
		if wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return attempts, fmt.Errorf("%w: %v", ctx.Err(), err)
			case <-timer.C:
			}
		}
	}

	return attempts, nil
}

func fullJitterDuration(maxDur time.Duration) time.Duration {
	if maxDur <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxDur)+1))
	if err != nil {
		return maxDur / 2
	}
	return time.Duration(n.Int64())
}

func randomHex(numBytes int) string {
	buf := make([]byte, numBytes)
	if _, err := rand.Read(buf); err != nil {
		sum := sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
		return hex.EncodeToString(sum[:numBytes])
	}
	return hex.EncodeToString(buf)
}

func syncDir(dir string) {
	// #nosec G304 -- dir is the parent directory of an atomic file target.
	df, err := os.Open(dir)
	if err != nil {
		return
	}
	_ = df.Sync()
	_ = df.Close()
}

func writeFileAtomic(path string, data []byte) error {
	cleanPath := filepath.Clean(path)
	dir := filepath.Dir(cleanPath)
	base := filepath.Base(cleanPath)

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create parent directory %s: %w", dir, err)
	}

	tmpName := fmt.Sprintf(".tmp-%s-%d-%s", base, os.Getpid(), randomHex(6))
	tmpPath := filepath.Join(dir, tmpName)

	// #nosec G304 -- tmpPath is constructed in the target file's parent directory for atomic rename.
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create atomic temp file: %w", err)
	}

	cleanup := func() {
		_ = os.Remove(tmpPath)
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("write atomic temp file: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		cleanup()
		return fmt.Errorf("fsync atomic temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close atomic temp file: %w", err)
	}
	// #nosec G302 -- atomic artifacts, checkpoints, and .sha256 sidecars must be 0644 readable once published.
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		cleanup()
		return fmt.Errorf("chmod atomic temp file: %w", err)
	}
	if err := os.Rename(tmpPath, cleanPath); err != nil {
		cleanup()
		return fmt.Errorf("atomic rename to %s: %w", cleanPath, err)
	}
	syncDir(dir)
	return nil
}

// WriteFileAtomicWithSHA256 crash-safely writes data to path via fsync + chmod 0644 + atomic rename
// and atomically writes the hex SHA-256 digest to path + ".sha256" via its own temp file + fsync + chmod 0644 + rename.
func WriteFileAtomicWithSHA256(path string, data []byte) (string, error) {
	cleanPath := filepath.Clean(path)
	sum := sha256.Sum256(data)
	digestHex := hex.EncodeToString(sum[:])

	if err := writeFileAtomic(cleanPath, data); err != nil {
		return "", err
	}
	if err := writeFileAtomic(cleanPath+".sha256", []byte(digestHex+"\n")); err != nil {
		return "", fmt.Errorf("write atomic sha256 sidecar: %w", err)
	}
	return digestHex, nil
}

// ReadVerifiedFile reads path (bounded by maxBytes+1) and path + ".sha256", recomputes SHA-256,
// and returns ErrIntegrityTampered if the sidecar is missing, malformed, or does not match.
func ReadVerifiedFile(path string, maxBytes int64) ([]byte, string, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxVerifiedBytes
	}
	cleanPath := filepath.Clean(path)

	// #nosec G304 -- caller explicitly requests verified read of target path.
	f, err := os.Open(cleanPath)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return nil, "", err
	}
	if !info.Mode().IsRegular() {
		return nil, "", fmt.Errorf("%w: %s is not a regular file", ErrIntegrityTampered, cleanPath)
	}

	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > maxBytes {
		return nil, "", fmt.Errorf("%w: %s exceeds limit of %d bytes", ErrPayloadTooLarge, cleanPath, maxBytes)
	}

	sidecarPath := cleanPath + ".sha256"
	// #nosec G304 -- sidecar path is adjacent to cleanPath.
	scFile, err := os.Open(sidecarPath)
	if err != nil {
		return nil, "", fmt.Errorf("%w: missing or unreadable sidecar %s: %v", ErrIntegrityTampered, sidecarPath, err)
	}
	defer func() { _ = scFile.Close() }()

	scInfo, err := scFile.Stat()
	if err != nil || !scInfo.Mode().IsRegular() {
		return nil, "", fmt.Errorf("%w: sidecar %s is not a regular file", ErrIntegrityTampered, sidecarPath)
	}

	scRaw, err := io.ReadAll(io.LimitReader(scFile, 4096))
	if err != nil {
		return nil, "", fmt.Errorf("%w: read sidecar %s: %v", ErrIntegrityTampered, sidecarPath, err)
	}

	expectedHex, ok := parseSidecarDigest(string(scRaw))
	if !ok {
		return nil, "", fmt.Errorf("%w: malformed sha256 sidecar %s", ErrIntegrityTampered, sidecarPath)
	}

	actualSum := sha256.Sum256(data)
	actualHex := hex.EncodeToString(actualSum[:])

	if subtle.ConstantTimeCompare([]byte(expectedHex), []byte(actualHex)) != 1 {
		return nil, "", fmt.Errorf("%w: digest mismatch for %s (expected %s, got %s)", ErrIntegrityTampered, cleanPath, expectedHex, actualHex)
	}

	return data, actualHex, nil
}

func parseSidecarDigest(raw string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(raw))
	if len(fields) == 0 {
		return "", false
	}
	token := strings.ToLower(strings.TrimPrefix(strings.ToLower(fields[0]), "sha256:"))
	if len(token) != sha256.Size*2 {
		return "", false
	}
	decoded, err := hex.DecodeString(token)
	if err != nil || len(decoded) != sha256.Size {
		return "", false
	}
	return token, true
}

type trackingResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *trackingResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *trackingResponseWriter) Write(b []byte) (int, error) {
	w.wroteHeader = true
	return w.ResponseWriter.Write(b)
}

func (w *trackingResponseWriter) Flush() {
	if fl, ok := w.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}

func (w *trackingResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func sanitizeRequestID(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" || len(s) > 128 {
		return ""
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x21 || c > 0x7e {
			return ""
		}
	}
	return s
}

// WrapHTTPHandler wraps next with baseline security headers, request ID propagation/generation,
// bounded in-flight concurrency backpressure (503 + Retry-After: 1), and safe panic recovery (500 JSON).
func WrapHTTPHandler(next http.Handler, maxInFlight int) http.Handler {
	if maxInFlight <= 0 {
		maxInFlight = DefaultMaxInFlight
	}
	sem := make(chan struct{}, maxInFlight)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := sanitizeRequestID(r.Header.Get("X-Request-ID"))
		if reqID == "" {
			reqID = "req_" + randomHex(8)
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Request-ID", reqID)

		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
		default:
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":               "server busy: concurrency limit reached",
				"retry_after_seconds": 1,
			})
			return
		}

		tw := &trackingResponseWriter{ResponseWriter: w}
		defer func() {
			if rec := recover(); rec != nil {
				// #nosec G706 -- request attributes are structured and sanitized before logging.
				slog.Error("recovered panic in HTTP handler",
					"request_id", reqID,
					"method", sanitizeRequestID(r.Method),
					"path", sanitizeRequestID(r.URL.Path),
					"panic", fmt.Sprintf("%v", rec),
				)
				if !tw.wroteHeader {
					w.Header().Del("Content-Length")
					w.Header().Del("Content-Encoding")
					w.Header().Set("X-Content-Type-Options", "nosniff")
					w.Header().Set("X-Frame-Options", "DENY")
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.Header().Set("X-Request-ID", reqID)
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error":      "internal server error",
						"request_id": reqID,
					})
				}
			}
		}()

		next.ServeHTTP(tw, r)
	})
}
