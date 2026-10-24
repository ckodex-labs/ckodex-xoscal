package webstudio

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
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

	"github.com/mchorfa/xoscal/server/internal/frameworks"
	"github.com/mchorfa/xoscal/server/internal/schemavalidate"
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
	mux.HandleFunc("/api/v1/frameworks", s.handleFrameworks)
	mux.HandleFunc("/api/v1/frameworks/", s.handleFrameworkSubpath)
	mux.HandleFunc("/api/v1/compare", s.handleCompare)
	mux.HandleFunc("/api/v1/validate", s.handleValidate)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return s, nil
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
		dir := r.URL.Query().Get("dir")
		if dir == "" {
			dir = "data/frameworks"
		}
		rawLayer := strings.ToLower(r.URL.Query().Get("layer"))
		layer := "catalog.json"
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
			if strings.HasSuffix(rawLayer, ".json") {
				layer = rawLayer
			} else {
				layer = rawLayer + ".json"
			}
		}

		artPath := filepath.Join(dir, fw.ID(), layer)
		var rep *frameworks.ParityReport
		var err error
		// #nosec G703,G304 -- artPath is resolved within the configured framework artifact directory.
		if _, statErr := os.Stat(artPath); statErr == nil {
			rep, err = frameworks.VerifyArtifactParity(fw, artPath)
		} else {
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
			default:
				http.Error(w, fmt.Sprintf("unknown layer %s", layer), http.StatusBadRequest)
				return
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
			var data []byte
			switch layerName {
			case "catalog", "catalog.json":
				data = layers.Catalog
			case "profile", "profile.json":
				data = layers.Profile
			case "component-definition", "component-definition.json", "components":
				data = layers.ComponentDefinition
			case "system-security-plan", "system-security-plan.json", "ssp":
				data = layers.SystemSecurityPlan
			case "assessment-plan", "assessment-plan.json", "ap":
				data = layers.AssessmentPlan
			case "assessment-results", "assessment-results.json", "ar":
				data = layers.AssessmentResults
			case "plan-of-action-and-milestones", "plan-of-action-and-milestones.json", "poam":
				data = layers.POAM
			default:
				http.Error(w, fmt.Sprintf("unknown layer %s", layerName), http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
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

// OpenBrowser attempts to open the specified URL in the system's default browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}
