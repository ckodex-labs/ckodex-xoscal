package webstudio

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mchorfa/xoscal/server/internal/feeds"
	"github.com/mchorfa/xoscal/server/internal/frameworks"
	"github.com/mchorfa/xoscal/server/internal/graph"
	"github.com/mchorfa/xoscal/server/internal/ontology"
	"github.com/mchorfa/xoscal/server/internal/resilience"
	"github.com/mchorfa/xoscal/server/internal/rootfs"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
	"github.com/mchorfa/xoscal/server/internal/utcp"
	"github.com/mchorfa/xoscal/server/internal/vectorstate"
)

//go:embed index.html
var indexHTML []byte

// Server provides the local zero-dependency xOSCAL Web Studio.
type Server struct {
	httpServer  *http.Server
	registry    *frameworks.Registry
	validator   *schemavalidate.Validator
	listener    net.Listener
	mu          sync.RWMutex
	cachedCtrls map[string][]frameworks.ControlInfo
}

// Config configures the Web Studio server.
type Config struct {
	Host     string
	Port     int
	Registry *frameworks.Registry
}

// NewServer initializes a new Web Studio server instance.
func NewServer(cfg Config) (*Server, error) {
	reg := cfg.Registry
	if reg == nil {
		reg = frameworks.DefaultRegistry()
	}

	val, err := schemavalidate.NewValidator()
	if err != nil {
		return nil, fmt.Errorf("init schema validator: %w", err)
	}

	s := &Server{
		registry:    reg,
		validator:   val,
		cachedCtrls: make(map[string][]frameworks.ControlInfo),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleHealthz)
	s.RegisterFrameworkAndValidationRoutes(mux)
	mux.HandleFunc("/api/v1/ontology", s.handleOntology)
	mux.HandleFunc("/api/v1/graph/stats", s.handleGraphStats)
	mux.HandleFunc("/api/v1/graph/crosswalk", s.handleGraphCrosswalk)
	mux.HandleFunc("/api/v1/graph/blast-radius", s.handleGraphBlastRadius)
	mux.HandleFunc("/api/v1/graph/export", s.handleGraphExport)
	mux.HandleFunc("/api/v1/feeds", s.handleFeeds)
	mux.HandleFunc("/api/v1/feeds/sync", s.handleFeedsSync)
	mux.HandleFunc("/.well-known/utcp", s.handleUTCPManual)
	mux.HandleFunc("/api/v1/utcp", s.handleUTCPManual)
	mux.HandleFunc("/api/v1/utcp/validate", s.handleUTCPValidate)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           resilience.WrapHTTPHandler(mux, 128),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	return s, nil
}

// RegisterFrameworkAndValidationRoutes mounts the shared framework registry, control explorer,
// parity verifier, framework comparison, and OSCAL schema/vector-state validation endpoints on mux.
func (s *Server) RegisterFrameworkAndValidationRoutes(mux *http.ServeMux) {
	if s == nil || mux == nil {
		return
	}
	mux.HandleFunc("/api/v1/frameworks", s.handleFrameworks)
	mux.HandleFunc("/api/v1/frameworks/", s.handleFrameworkSubpath)
	mux.HandleFunc("/api/v1/compare", s.handleCompare)
	mux.HandleFunc("/api/v1/validate", s.handleValidate)
}

// Start begins listening on the configured address.
// If the configured port is already in use, it attempts up to 9 consecutive fallback ports.
func (s *Server) Start() (string, error) {
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		// Attempt port fallback if specific port was busy
		host, portStr, splitErr := net.SplitHostPort(s.httpServer.Addr)
		if splitErr == nil {
			if basePort, numErr := strconv.Atoi(portStr); numErr == nil && basePort > 0 {
				for p := basePort + 1; p <= basePort+9; p++ {
					altAddr := net.JoinHostPort(host, strconv.Itoa(p))
					if altLn, altErr := net.Listen("tcp", altAddr); altErr == nil {
						s.listener = altLn
						s.httpServer.Addr = altAddr
						go func() {
							_ = s.httpServer.Serve(altLn)
						}()
						return altLn.Addr().String(), nil
					}
				}
			}
		}
		return "", fmt.Errorf("listen on %s: %w", s.httpServer.Addr, err)
	}
	s.listener = ln

	go func() {
		_ = s.httpServer.Serve(ln)
	}()

	return ln.Addr().String(), nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Handler returns the underlying http.Handler for testing or embedding.
func (s *Server) Handler() http.Handler {
	return s.httpServer.Handler
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":     "healthy",
		"time":       time.Now().UTC().Format(time.RFC3339),
		"frameworks": len(s.registry.ListInfo()),
	})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(indexHTML)
}

func (s *Server) handleFrameworks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	infos := s.registry.ListInfo()
	writeJSON(w, http.StatusOK, infos)
}

func (s *Server) handleFrameworkSubpath(w http.ResponseWriter, r *http.Request) {
	// Pattern: /api/v1/frameworks/{id} or /api/v1/frameworks/{id}/controls
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/frameworks/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	fwID := parts[0]
	fw, exists := s.registry.Get(fwID)
	if !exists {
		http.Error(w, fmt.Sprintf("framework %s not found", fwID), http.StatusNotFound)
		return
	}

	if len(parts) == 1 {
		// Details of framework
		writeJSON(w, http.StatusOK, frameworks.InfoOf(fw))
		return
	}

	if parts[1] == "parity" {
		dir := strings.TrimSpace(r.URL.Query().Get("dir"))
		if dir == "" {
			dir = "data/frameworks"
		}
		if strings.Contains(dir, "..") || strings.ContainsRune(dir, '\x00') || strings.Contains(dir, "\\") {
			http.Error(w, "invalid dir parameter: path traversal is forbidden", http.StatusBadRequest)
			return
		}
		cwd, cwdErr := os.Getwd()
		absDir, absErr := filepath.Abs(dir)
		relDir, relErr := filepath.Rel(cwd, absDir)
		if cwdErr != nil || absErr != nil || relErr != nil || relDir == ".." || strings.HasPrefix(relDir, ".."+string(filepath.Separator)) {
			http.Error(w, "invalid dir parameter: directory must reside within workspace", http.StatusBadRequest)
			return
		}

		rawLayer := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("layer")))
		var layer string
		switch rawLayer {
		case "", "catalog", "catalog.json":
			layer = "catalog.json"
		case "profile", "profile.json":
			layer = "profile.json"
		case "component-definition", "component-definition.json", "components", "component":
			layer = "component-definition.json"
		case "system-security-plan", "system-security-plan.json", "ssp":
			layer = "system-security-plan.json"
		case "assessment-plan", "assessment-plan.json", "ap":
			layer = "assessment-plan.json"
		case "assessment-results", "assessment-results.json", "ar":
			layer = "assessment-results.json"
		case "plan-of-action-and-milestones", "plan-of-action-and-milestones.json", "poam":
			layer = "plan-of-action-and-milestones.json"
		default:
			http.Error(w, fmt.Sprintf("invalid layer %q: must be one of the 7 canonical OSCAL layers", rawLayer), http.StatusBadRequest)
			return
		}

		var rep *frameworks.ParityReport
		var err error
		loadedFromDisk := false

		if wsRoot, rootErr := os.OpenRoot(cwd); rootErr == nil {
			canReadDisk := true
			if relDir != "." {
				dirInfo, statErr := rootfs.OptionalStat(wsRoot, relDir)
				if statErr != nil {
					_ = wsRoot.Close()
					http.Error(w, "invalid dir parameter: symlink or path escapes workspace", http.StatusBadRequest)
					return
				}
				if dirInfo == nil {
					canReadDisk = false
				} else if !dirInfo.IsDir() {
					_ = wsRoot.Close()
					http.Error(w, "invalid dir parameter: target is not a directory", http.StatusBadRequest)
					return
				}
			}

			if canReadDisk {
				relArt := filepath.Join(fw.ID(), layer)
				if relDir != "." {
					relArt = filepath.Join(relDir, relArt)
				}
				artBytes, readErr := rootfs.ReadFile(wsRoot, relArt)
				if readErr == nil {
					loadedFromDisk = true
					rep, err = frameworks.VerifyBytesParity(fw, artBytes, "memory://"+fw.ID()+"/"+layer)
					if err == nil && rep != nil {
						rep.ArtifactPath = relArt
						if scBytes, scErr := rootfs.OptionalReadFile(wsRoot, relArt+".sha256"); scErr == nil && scBytes != nil {
							scContent := strings.TrimSpace(string(scBytes))
							rep.SidecarPath = rep.ArtifactPath + ".sha256"
							rep.SidecarSHA256 = scContent
							if strings.Contains(strings.ToLower(scContent), strings.ToLower(rep.ArtifactSHA256)) {
								rep.SidecarMatch = true
							} else {
								rep.SidecarMatch = false
								rep.Verdict = frameworks.VerdictDeviationDetected
								rep.Discrepancies = append(rep.Discrepancies, frameworks.Discrepancy{
									ControlID: "*",
									Aspect:    "SHA256Sidecar",
									Expected:  rep.ArtifactSHA256,
									Actual:    scContent,
									Message:   "SHA-256 sidecar digest does not match computed artifact hash",
								})
							}
						} else {
							rep.SidecarMatch = false
						}
					}
				} else if !errors.Is(readErr, os.ErrNotExist) {
					_ = wsRoot.Close()
					http.Error(w, "invalid artifact path inside dir parameter", http.StatusBadRequest)
					return
				}
			}
			_ = wsRoot.Close()
		}

		if !loadedFromDisk {
			layers, pErr := frameworks.ProjectAllLayers(fw)
			if pErr != nil {
				http.Error(w, fmt.Sprintf("project layers: %v", pErr), http.StatusInternalServerError)
				return
			}
			var b []byte
			switch layer {
			case "catalog.json":
				b = layers.Catalog
			case "profile.json":
				b = layers.Profile
			case "component-definition.json":
				b = layers.ComponentDefinition
			case "system-security-plan.json":
				b = layers.SystemSecurityPlan
			case "assessment-plan.json":
				b = layers.AssessmentPlan
			case "assessment-results.json":
				b = layers.AssessmentResults
			case "plan-of-action-and-milestones.json":
				b = layers.POAM
			}
			rep, err = frameworks.VerifyBytesParity(fw, b, "memory://"+fw.ID()+"/"+layer)
		}
		if err != nil {
			http.Error(w, fmt.Sprintf("parity check error: %v", err), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, rep)
		return
	}

	if parts[1] == "layers" {
		layers, err := frameworks.ProjectAllLayers(fw)
		if err != nil {
			http.Error(w, fmt.Sprintf("project layers error: %v", err), http.StatusInternalServerError)
			return
		}

		if len(parts) == 3 && parts[2] != "" {
			layerName := strings.ToLower(parts[2])
			if layerName == "checksums.sha256" || layerName == "checksums" {
				hashOf := func(b []byte) string {
					h := sha256.Sum256(b)
					return hex.EncodeToString(h[:])
				}
				lines := []string{
					fmt.Sprintf("%s  catalog.json", hashOf(layers.Catalog)),
					fmt.Sprintf("%s  profile.json", hashOf(layers.Profile)),
					fmt.Sprintf("%s  component-definition.json", hashOf(layers.ComponentDefinition)),
					fmt.Sprintf("%s  system-security-plan.json", hashOf(layers.SystemSecurityPlan)),
					fmt.Sprintf("%s  assessment-plan.json", hashOf(layers.AssessmentPlan)),
					fmt.Sprintf("%s  assessment-results.json", hashOf(layers.AssessmentResults)),
					fmt.Sprintf("%s  plan-of-action-and-milestones.json", hashOf(layers.POAM)),
				}
				data := []byte(strings.Join(lines, "\n") + "\n")
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				if r.URL.Query().Get("download") == "1" {
					w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fw.ID()+"-checksums.sha256"))
				}
				w.WriteHeader(http.StatusOK)
				// #nosec G705 -- response is strictly text/plain SHA-256 hex digests with X-Content-Type-Options: nosniff.
				_, _ = w.Write(data)
				return
			}

			var data []byte
			var fileName string
			switch layerName {
			case "catalog", "catalog.json":
				data = layers.Catalog
				fileName = "catalog.json"
			case "profile", "profile.json":
				data = layers.Profile
				fileName = "profile.json"
			case "component-definition", "component-definition.json", "components":
				data = layers.ComponentDefinition
				fileName = "component-definition.json"
			case "system-security-plan", "system-security-plan.json", "ssp":
				data = layers.SystemSecurityPlan
				fileName = "system-security-plan.json"
			case "assessment-plan", "assessment-plan.json", "ap":
				data = layers.AssessmentPlan
				fileName = "assessment-plan.json"
			case "assessment-results", "assessment-results.json", "ar":
				data = layers.AssessmentResults
				fileName = "assessment-results.json"
			case "plan-of-action-and-milestones", "plan-of-action-and-milestones.json", "poam":
				data = layers.POAM
				fileName = "plan-of-action-and-milestones.json"
			default:
				http.Error(w, fmt.Sprintf("unknown layer %s", layerName), http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			if r.URL.Query().Get("download") == "1" {
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fw.ID()+"-"+fileName))
			}
			w.WriteHeader(http.StatusOK)
			// #nosec G705 -- response is strictly application/json OSCAL document with X-Content-Type-Options: nosniff.
			_, _ = w.Write(data)
			return
		}

		type LayerSummary struct {
			Name      string `json:"name"`
			Kind      string `json:"kind"`
			SizeBytes int    `json:"size_bytes"`
			SHA256    string `json:"sha256"`
		}

		hashOf := func(b []byte) string {
			h := sha256.Sum256(b)
			return hex.EncodeToString(h[:])
		}

		list := []LayerSummary{
			{"catalog.json", "catalog", len(layers.Catalog), hashOf(layers.Catalog)},
			{"profile.json", "profile", len(layers.Profile), hashOf(layers.Profile)},
			{"component-definition.json", "component-definition", len(layers.ComponentDefinition), hashOf(layers.ComponentDefinition)},
			{"system-security-plan.json", "system-security-plan", len(layers.SystemSecurityPlan), hashOf(layers.SystemSecurityPlan)},
			{"assessment-plan.json", "assessment-plan", len(layers.AssessmentPlan), hashOf(layers.AssessmentPlan)},
			{"assessment-results.json", "assessment-results", len(layers.AssessmentResults), hashOf(layers.AssessmentResults)},
			{"plan-of-action-and-milestones.json", "plan-of-action-and-milestones", len(layers.POAM), hashOf(layers.POAM)},
		}
		writeJSON(w, http.StatusOK, list)
		return
	}

	if parts[1] == "controls" {
		controls, err := s.getOrLoadControls(fw)
		if err != nil {
			http.Error(w, fmt.Sprintf("load controls: %v", err), http.StatusInternalServerError)
			return
		}

		if len(parts) == 3 && parts[2] != "" {
			ctrlID := parts[2]
			ctrl, found := frameworks.GetControl(controls, ctrlID)
			if !found {
				http.Error(w, fmt.Sprintf("control %s not found in %s", ctrlID, fwID), http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, ctrl)
			return
		}

		// Query params: q, family, limit
		query := r.URL.Query().Get("q")
		fam := r.URL.Query().Get("family")

		matched := controls
		if fam != "" {
			var famFiltered []frameworks.ControlInfo
			famUpper := strings.ToUpper(fam)
			for _, c := range matched {
				if strings.ToUpper(c.FamilyID) == famUpper {
					famFiltered = append(famFiltered, c)
				}
			}
			matched = famFiltered
		}

		if query != "" {
			matched = frameworks.SearchControls(matched, query, 0)
		}

		writeJSON(w, http.StatusOK, matched)
		return
	}

	http.NotFound(w, r)
}

func (s *Server) handleCompare(w http.ResponseWriter, r *http.Request) {
	fromID := r.URL.Query().Get("from")
	toID := r.URL.Query().Get("to")

	if fromID == "" || toID == "" {
		http.Error(w, "missing required query parameters 'from' and 'to'", http.StatusBadRequest)
		return
	}

	fromFw, ok := s.registry.Get(fromID)
	if !ok {
		http.Error(w, fmt.Sprintf("framework '%s' not found", fromID), http.StatusNotFound)
		return
	}

	toFw, ok := s.registry.Get(toID)
	if !ok {
		http.Error(w, fmt.Sprintf("framework '%s' not found", toID), http.StatusNotFound)
		return
	}

	comp, err := frameworks.CompareFrameworks(fromFw, toFw)
	if err != nil {
		http.Error(w, fmt.Sprintf("comparison failed: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, comp)
}

type validateResponse struct {
	Valid         bool                     `json:"valid"`
	Kind          string                   `json:"kind,omitempty"`
	SchemaVersion string                   `json:"schema_version"`
	SchemaError   string                   `json:"schema_error,omitempty"`
	VectorState   *vectorstate.VectorState `json:"vector_state,omitempty"`
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10*1024*1024))
	if err != nil {
		http.Error(w, fmt.Sprintf("read request body: %v", err), http.StatusBadRequest)
		return
	}

	resp := validateResponse{
		SchemaVersion: schemavalidate.SchemaVersion,
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		resp.Valid = false
		resp.SchemaError = fmt.Sprintf("invalid JSON payload: %v", err)
		writeJSON(w, http.StatusOK, resp)
		return
	}

	var detectedKind schemavalidate.ArtifactKind
	for _, k := range []schemavalidate.ArtifactKind{
		schemavalidate.KindComponentDefinition,
		schemavalidate.KindSSP,
		schemavalidate.KindCatalog,
		schemavalidate.KindProfile,
		schemavalidate.KindAssessmentPlan,
		schemavalidate.KindAssessmentResults,
		schemavalidate.KindPOAM,
		schemavalidate.KindMapping,
	} {
		if _, ok := raw[k.RootKey]; ok {
			detectedKind = k
			break
		}
	}

	if detectedKind.Name == "" {
		resp.Valid = false
		resp.SchemaError = "unknown OSCAL artifact kind (could not match root key)"
		writeJSON(w, http.StatusOK, resp)
		return
	}

	resp.Kind = detectedKind.Name
	if err := s.validator.Validate(body, detectedKind); err != nil {
		resp.Valid = false
		resp.SchemaError = err.Error()
	} else {
		resp.Valid = true
	}

	// Compute Vector State
	st, _ := vectorstate.EvaluateBytes(body, "")
	resp.VectorState = st

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) getOrLoadControls(fw frameworks.Framework) ([]frameworks.ControlInfo, error) {
	s.mu.RLock()
	cached, ok := s.cachedCtrls[fw.ID()]
	s.mu.RUnlock()
	if ok {
		return cached, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if cached, ok := s.cachedCtrls[fw.ID()]; ok {
		return cached, nil
	}

	ctrls, err := frameworks.LoadControls(fw)
	if err != nil {
		return nil, err
	}

	s.cachedCtrls[fw.ID()] = ctrls
	return ctrls, nil
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func sanitizeHTTPHost(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" || len(s) > 255 {
		return ""
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '.' || c == '-' || c == ':' || c == '[' || c == ']') {
			return ""
		}
	}
	return s
}

func (s *Server) handleUTCPManual(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	scheme := "http://"
	if r.TLS != nil {
		scheme = "https://"
	}
	baseURL := "http://127.0.0.1:8080"
	if host := sanitizeHTTPHost(r.Host); host != "" {
		baseURL = scheme + host
	}
	manual := utcp.GenerateManual(utcp.ManualOptions{
		BaseHTTPURL: baseURL,
	})
	writeJSON(w, http.StatusOK, manual)
}

func (s *Server) handleUTCPValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	rep, err := utcp.ValidateManual(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (s *Server) handleOntology(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, ontology.CanonicalSpec())
}

func (s *Server) handleGraphStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ug, err := graph.DefaultUnifiedGraph()
	if err != nil {
		http.Error(w, fmt.Sprintf("unified graph error: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, ug.Stats())
}

func (s *Server) handleGraphCrosswalk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctrl := strings.TrimSpace(r.URL.Query().Get("control"))
	if ctrl == "" {
		ctrl = "ac-2"
	}
	ug, err := graph.DefaultUnifiedGraph()
	if err != nil {
		http.Error(w, fmt.Sprintf("unified graph error: %v", err), http.StatusInternalServerError)
		return
	}
	rep, err := ug.QueryCrosswalk(ctrl)
	if err != nil {
		http.Error(w, fmt.Sprintf("crosswalk error: %v", err), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (s *Server) handleGraphBlastRadius(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	target := strings.TrimSpace(r.URL.Query().Get("target"))
	if target == "" {
		target = "CVE-2024-3400"
	}
	depth := 2
	if dStr := strings.TrimSpace(r.URL.Query().Get("depth")); dStr != "" {
		if dVal, err := strconv.Atoi(dStr); err == nil && dVal > 0 {
			depth = dVal
		}
	}
	ug, err := graph.DefaultUnifiedGraph()
	if err != nil {
		http.Error(w, fmt.Sprintf("unified graph error: %v", err), http.StatusInternalServerError)
		return
	}
	rep, err := ug.ComputeBlastRadius(target, depth)
	if err != nil {
		http.Error(w, fmt.Sprintf("blast radius error: %v", err), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (s *Server) handleGraphExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	target := strings.TrimSpace(r.URL.Query().Get("target"))
	if target == "" {
		target = "CVE-2024-3400"
	}
	format := strings.TrimSpace(r.URL.Query().Get("format"))
	if format == "" {
		format = "jsonld"
	}
	depth := 2
	if dStr := strings.TrimSpace(r.URL.Query().Get("depth")); dStr != "" {
		if dVal, err := strconv.Atoi(dStr); err == nil && dVal > 0 {
			depth = dVal
		}
	}
	ug, err := graph.DefaultUnifiedGraph()
	if err != nil {
		http.Error(w, fmt.Sprintf("unified graph error: %v", err), http.StatusInternalServerError)
		return
	}
	out, err := ug.ExportSubgraph(target, depth, format)
	if err != nil {
		http.Error(w, fmt.Sprintf("export error: %v", err), http.StatusBadRequest)
		return
	}
	if strings.EqualFold(format, "cypher") {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	// #nosec G705 -- response is strictly text/plain or application/json with X-Content-Type-Options: nosniff.
	_, _ = w.Write([]byte(out))
}

func (s *Server) handleFeeds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	feedID := strings.TrimSpace(r.URL.Query().Get("feed"))
	if feedID != "" {
		b, err := feeds.InspectFeed(feedID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, b)
		return
	}
	writeJSON(w, http.StatusOK, feeds.ListFeeds())
}

func (s *Server) handleFeedsSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	feedID := strings.TrimSpace(r.URL.Query().Get("feed"))
	receipt, err := feeds.SyncFeeds(r.Context(), feeds.SyncOptions{
		FeedID:  feedID,
		Offline: true,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}

// OpenBrowser attempts to open the specified URL in the system's default browser.
func OpenBrowser(url string) error {
	if !strings.HasPrefix(url, "http://127.0.0.1:") && !strings.HasPrefix(url, "http://localhost:") {
		return fmt.Errorf("refusing to open non-local browser URL: %s", url)
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		// #nosec G204 -- url is verified above to be a local loopback HTTP listener URL.
		cmd = exec.Command("open", url)
	case "linux":
		// #nosec G204 -- url is verified above to be a local loopback HTTP listener URL.
		cmd = exec.Command("xdg-open", url)
	case "windows":
		// #nosec G204 -- url is verified above to be a local loopback HTTP listener URL.
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}
