package webstudio

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"runtime"
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
func (s *Server) Start() (string, error) {
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
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
