// Package utcp implements the Universal Tool Calling Protocol (UTCP) v1.0.1
// specification (https://github.com/universal-tool-calling-protocol/utcp-specification)
// and multi-language ecosystem integrations (utcp-mcp, rs-utcp, typescript-utcp,
// go-utcp, python-utcp) under CKODEX constitutional governance invariants.
package utcp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	// SpecVersion is the target Universal Tool Calling Protocol specification version.
	SpecVersion = "1.0.1"
	// DefaultManualVersion is the default xOSCAL UTCP manual version.
	DefaultManualVersion = "1.0.0"
	// SpecRepositoryURL is the canonical UTCP specification repository.
	SpecRepositoryURL = "https://github.com/universal-tool-calling-protocol/utcp-specification"
	// OrganizationURL is the canonical UTCP GitHub organization.
	OrganizationURL = "https://github.com/universal-tool-calling-protocol"
)

// GovernanceClass models CKODEX Rule 6 distinct governance states (DISCOVER != TRUST != INVOKE != MUTATE != PROMOTE).
type GovernanceClass string

// GovernanceStanding is an alias for GovernanceClass representing CKODEX Rule 6 operational standing.
type GovernanceStanding = GovernanceClass

const (
	GovernanceDiscover GovernanceClass = "DISCOVER"
	GovernanceTrust    GovernanceClass = "TRUST"
	GovernanceInvoke   GovernanceClass = "INVOKE"
	GovernanceMutate   GovernanceClass = "MUTATE"
	GovernancePromote  GovernanceClass = "PROMOTE"
)

// DefaultAllowedCommunicationProtocols returns the UTCP v1.0.1 communication
// protocols authorized for xOSCAL manuals and call templates (excluding WebSockets per Rule 10).
func DefaultAllowedCommunicationProtocols() []string {
	return []string{"cli", "http", "grpc", "sse", "text", "mcp"}
}

// Manual represents a strict UTCP v1.0.1 wire manual document published at /.well-known/utcp.
// Upstream typescript-utcp (UtcpManualSchema.strict()) and @utcp/mcp-bridge reject any extra
// root keys beyond utcp_version, manual_version, and tools. Extended metadata is kept out of
// default JSON wire serialization and exposed via ExtendedEnvelope() when ?profile=extended is requested.
type Manual struct {
	UTCPVersion                   string            `json:"utcp_version"`
	ManualVersion                 string            `json:"manual_version"`
	Tools                         []Tool            `json:"tools"`
	Name                          string            `json:"-"`
	Description                   string            `json:"-"`
	AllowedCommunicationProtocols []string          `json:"-"`
	Metadata                      *ManualInfo       `json:"-"`
	Info                          ManualInfo        `json:"-"`
	Variables                     map[string]string `json:"-"`
	Auth                          *AuthSpec         `json:"-"`
}

// ExtendedManualEnvelope carries extended CKODEX/xOSCAL provider metadata when
// ?profile=extended is explicitly requested by diagnostic clients.
type ExtendedManualEnvelope struct {
	UTCPVersion                   string            `json:"utcp_version"`
	ManualVersion                 string            `json:"manual_version"`
	Name                          string            `json:"name,omitempty"`
	Description                   string            `json:"description,omitempty"`
	AllowedCommunicationProtocols []string          `json:"allowed_communication_protocols,omitempty"`
	Metadata                      *ManualInfo       `json:"metadata,omitempty"`
	Variables                     map[string]string `json:"variables,omitempty"`
	Tools                         []Tool            `json:"tools"`
}

// ExtendedEnvelope returns an ExtendedManualEnvelope projection of m for ?profile=extended responses.
func (m *Manual) ExtendedEnvelope() *ExtendedManualEnvelope {
	if m == nil {
		return nil
	}
	meta := m.Metadata
	if meta == nil && m.Info.Title != "" {
		cp := m.Info
		meta = &cp
	}
	return &ExtendedManualEnvelope{
		UTCPVersion:                   m.UTCPVersion,
		ManualVersion:                 m.ManualVersion,
		Name:                          m.Name,
		Description:                   m.Description,
		AllowedCommunicationProtocols: m.AllowedCommunicationProtocols,
		Metadata:                      meta,
		Variables:                     m.Variables,
		Tools:                         m.Tools,
	}
}

// ManualInfo describes the provider metadata in an extended UTCP manual envelope.
type ManualInfo struct {
	Title       string            `json:"title"`
	Version     string            `json:"version"`
	Description string            `json:"description,omitempty"`
	SpecURL     string            `json:"spec_url,omitempty"`
	Ecosystem   map[string]string `json:"x-utcp-ecosystem,omitempty"`
}

// AuthSpec describes UTCP v1.0.1 authentication configuration for call templates
// (e.g. ApiKeyAuth: {"auth_type": "api_key", "api_key": "Bearer ${XOSCAL_API_TOKEN}", "var_name": "Authorization", "location": "header"}).
type AuthSpec struct {
	AuthType string `json:"auth_type"` // "api_key", "basic", "oauth2"
	APIKey   string `json:"api_key,omitempty"`
	VarName  string `json:"var_name,omitempty"`
	Location string `json:"location,omitempty"` // "header", "query", "cookie"
	Prefix   string `json:"prefix,omitempty"`
}

// JSONSchema represents a JSON Schema object for tool inputs and outputs.
type JSONSchema struct {
	Type        string                        `json:"type"`
	Description string                        `json:"description,omitempty"`
	Properties  map[string]JSONSchemaProperty `json:"properties,omitempty"`
	Required    []string                      `json:"required,omitempty"`
}

// JSONSchemaProperty describes a single property in a tool's input or output JSON Schema.
type JSONSchemaProperty struct {
	Type        string              `json:"type"`
	Description string              `json:"description,omitempty"`
	Default     any                 `json:"default,omitempty"`
	Enum        []string            `json:"enum,omitempty"`
	Items       *JSONSchemaProperty `json:"items,omitempty"`
}

// GovernanceMetadata attaches CKODEX constitutional authority & state classification internally.
type GovernanceMetadata struct {
	OperationClass    GovernanceClass `json:"operation_class"`
	RequiredAuthority string          `json:"required_authority"`
	EvidenceChannel   string          `json:"evidence_channel"`
	AirGapCapable     bool            `json:"air_gap_capable"`
}

// CLICommand represents a single command entry in a UTCP v1.0.1 CLI call template
// as consumed by python-utcp (utcp-cli) and typescript-utcp (@utcp/cli).
type CLICommand struct {
	Command             string `json:"command"`
	AppendToFinalOutput bool   `json:"append_to_final_output,omitempty"`
}

// MCPTemplateConfig models the config object required on UTCP "mcp" call templates.
type MCPTemplateConfig struct {
	MCPServers map[string]MCPServer `json:"mcpServers,omitempty"`
}

// CallTemplate defines the native protocol execution instructions for a UTCP tool.
// Strict UTCP v1.0.1 Zod schemas (@utcp/http, @utcp/cli) reject unknown keys such as
// "command", "args", or "x_ckodex_governance", so internal-only fields use json:"-".
type CallTemplate struct {
	Name                          string              `json:"name,omitempty"`
	CallTemplateType              string              `json:"call_template_type"` // "cli", "http", "grpc", "sse", "text", "mcp"
	AllowedCommunicationProtocols []string            `json:"allowed_communication_protocols,omitempty"`
	Commands                      []CLICommand        `json:"commands,omitempty"`
	WorkingDir                    string              `json:"working_dir,omitempty"`
	EnvVars                       map[string]string   `json:"env_vars,omitempty"`
	URL                           string              `json:"url,omitempty"`
	HTTPMethod                    string              `json:"http_method,omitempty"`
	ContentType                   string              `json:"content_type,omitempty"`
	Headers                       map[string]string   `json:"headers,omitempty"`
	BodyField                     string              `json:"body_field,omitempty"`
	HeaderFields                  []string            `json:"header_fields,omitempty"`
	GRPCService                   string              `json:"grpc_service,omitempty"`
	GRPCMethod                    string              `json:"grpc_method,omitempty"`
	GRPCHost                      string              `json:"grpc_host,omitempty"`
	FilePath                      string              `json:"file_path,omitempty"`
	Config                        *MCPTemplateConfig  `json:"config,omitempty"`
	Auth                          *AuthSpec           `json:"auth,omitempty"`
	Command                       string              `json:"-"`
	Args                          []string            `json:"-"`
	Governance                    *GovernanceMetadata `json:"-"`
}

// Tool represents an individual callable tool entry in a UTCP manual.
// Governance classification is encoded on the wire via standard Tool.Tags ("governance:<STANDING>")
// and mirrored onto the internal Governance field (json:"-").
type Tool struct {
	Name             string             `json:"name"`
	Description      string             `json:"description"`
	Tags             []string           `json:"tags,omitempty"`
	Inputs           JSONSchema         `json:"inputs"`
	Outputs          JSONSchema         `json:"outputs"`
	ToolCallTemplate CallTemplate       `json:"tool_call_template"`
	Governance       GovernanceStanding `json:"-"`
}

type wireTool struct {
	Name             string       `json:"name"`
	Description      string       `json:"description"`
	Tags             []string     `json:"tags,omitempty"`
	Inputs           JSONSchema   `json:"inputs"`
	Outputs          JSONSchema   `json:"outputs"`
	ToolCallTemplate CallTemplate `json:"tool_call_template"`
}

func (t Tool) effectiveGovernance() GovernanceStanding {
	if strings.TrimSpace(string(t.Governance)) != "" {
		return GovernanceStanding(strings.ToUpper(strings.TrimSpace(string(t.Governance))))
	}
	if t.ToolCallTemplate.Governance != nil && strings.TrimSpace(string(t.ToolCallTemplate.Governance.OperationClass)) != "" {
		return GovernanceStanding(strings.ToUpper(strings.TrimSpace(string(t.ToolCallTemplate.Governance.OperationClass))))
	}
	return parseGovernanceFromTags(t.Tags)
}

func parseGovernanceFromTags(tags []string) GovernanceStanding {
	for _, raw := range tags {
		trimmed := strings.TrimSpace(raw)
		if len(trimmed) > len("governance:") && strings.EqualFold(trimmed[:len("governance:")], "governance:") {
			val := strings.ToUpper(strings.TrimSpace(trimmed[len("governance:"):]))
			if val != "" {
				return GovernanceStanding(val)
			}
		}
	}
	return ""
}

func ensureGovernanceTag(tags []string, standing GovernanceStanding) []string {
	st := strings.ToUpper(strings.TrimSpace(string(standing)))
	if st == "" {
		return tags
	}
	wantTag := "governance:" + st
	for _, existing := range tags {
		if strings.EqualFold(strings.TrimSpace(existing), wantTag) {
			return tags
		}
		if len(existing) > len("governance:") && strings.EqualFold(strings.TrimSpace(existing)[:len("governance:")], "governance:") {
			return tags
		}
	}
	out := make([]string, 0, len(tags)+1)
	out = append(out, tags...)
	out = append(out, wantTag)
	return out
}

// MarshalJSON serializes Tool using strictly UTCP v1.0.1 ToolSchema fields while ensuring
// Governance standing is represented as a standard "governance:<STANDING>" entry in tags.
func (t Tool) MarshalJSON() ([]byte, error) {
	wt := wireTool{
		Name:             t.Name,
		Description:      t.Description,
		Tags:             ensureGovernanceTag(t.Tags, t.effectiveGovernance()),
		Inputs:           t.Inputs,
		Outputs:          t.Outputs,
		ToolCallTemplate: t.ToolCallTemplate,
	}
	return json.Marshal(wt)
}

// UnmarshalJSON deserializes Tool and populates the internal Governance field from tags.
func (t *Tool) UnmarshalJSON(data []byte) error {
	var wt wireTool
	if err := json.Unmarshal(data, &wt); err != nil {
		return err
	}
	t.Name = wt.Name
	t.Description = wt.Description
	t.Tags = wt.Tags
	t.Inputs = wt.Inputs
	t.Outputs = wt.Outputs
	t.ToolCallTemplate = wt.ToolCallTemplate
	t.Governance = parseGovernanceFromTags(wt.Tags)
	return nil
}

// ManualOptions configures generation of the xOSCAL UTCP Manual.
type ManualOptions struct {
	BaseURL           string
	BaseHTTPURL       string // Backward-compatible alias for BaseURL
	GRPCAddr          string
	GRPCHost          string // Backward-compatible alias for GRPCAddr
	BinaryPath        string
	Version           string
	IncludeHTTP       bool
	IncludeCLI        bool
	IncludeGRPC       bool
	BearerTokenEnvVar string
}

func normalizeManualOptions(opts ManualOptions) ManualOptions {
	if strings.TrimSpace(opts.BaseURL) == "" {
		opts.BaseURL = strings.TrimSpace(opts.BaseHTTPURL)
	}
	if opts.BaseURL == "" {
		opts.BaseURL = "http://127.0.0.1:8080"
	}
	opts.BaseURL = strings.TrimRight(opts.BaseURL, "/")
	opts.BaseHTTPURL = opts.BaseURL

	if strings.TrimSpace(opts.GRPCAddr) == "" {
		opts.GRPCAddr = strings.TrimSpace(opts.GRPCHost)
	}
	if opts.GRPCAddr == "" {
		opts.GRPCAddr = "127.0.0.1:50051"
	}
	opts.GRPCHost = opts.GRPCAddr

	if strings.TrimSpace(opts.BinaryPath) == "" {
		opts.BinaryPath = "xoscal"
	}
	if strings.TrimSpace(opts.Version) == "" {
		opts.Version = DefaultManualVersion
	}
	if !opts.IncludeHTTP && !opts.IncludeCLI && !opts.IncludeGRPC {
		opts.IncludeHTTP = true
	}
	return opts
}

var utcpTemplatePlaceholderRe = regexp.MustCompile(`\$\{([a-zA-Z0-9_]+)\}`)

func buildCLICommands(bin string, args []string) []CLICommand {
	parts := make([]string, 0, 1+len(args))
	if strings.TrimSpace(bin) != "" {
		parts = append(parts, strings.TrimSpace(bin))
	}
	for _, a := range args {
		if utcpTemplatePlaceholderRe.MatchString(a) {
			converted := utcpTemplatePlaceholderRe.ReplaceAllStringFunc(a, func(m string) string {
				sub := utcpTemplatePlaceholderRe.FindStringSubmatch(m)
				if len(sub) == 2 {
					return `"UTCP_ARG_` + sub[1] + `_UTCP_END"`
				}
				return m
			})
			parts = append(parts, converted)
		} else {
			parts = append(parts, a)
		}
	}
	return []CLICommand{
		{
			Command:             strings.Join(parts, " "),
			AppendToFinalOutput: true,
		},
	}
}

// BuildCanonicalManual returns the wire-safe HTTP-only UTCP v1.0.1 Manual for /.well-known/utcp discovery.
func BuildCanonicalManual(baseURL, grpcAddr string) *Manual {
	return BuildManualWithOptions(ManualOptions{
		BaseURL:     baseURL,
		GRPCAddr:    grpcAddr,
		IncludeHTTP: true,
		IncludeCLI:  false,
		IncludeGRPC: false,
	})
}

// BuildFullManual returns a UTCP Manual containing HTTP, CLI, and gRPC tool templates.
func BuildFullManual(baseURL, grpcAddr string) *Manual {
	return BuildManualWithOptions(ManualOptions{
		BaseURL:     baseURL,
		GRPCAddr:    grpcAddr,
		IncludeHTTP: true,
		IncludeCLI:  true,
		IncludeGRPC: true,
	})
}

// GenerateManual is a backward-compatible alias for BuildManualWithOptions.
func GenerateManual(opts ManualOptions) *Manual {
	return BuildManualWithOptions(opts)
}

// BuildManualWithOptions synthesizes the xOSCAL UTCP v1.0.1 Manual according to opts.
// When none of IncludeHTTP, IncludeCLI, or IncludeGRPC are set, it defaults to HTTP-only
// so upstream python-utcp, typescript-utcp, and @utcp/mcp-bridge clients deserialize cleanly.
func BuildManualWithOptions(opts ManualOptions) *Manual {
	opts = normalizeManualOptions(opts)

	ecoMap := make(map[string]string, len(SupportedRepositories))
	for _, r := range SupportedRepositories {
		ecoMap[r.Name] = r.URL
	}

	var allowedProtos []string
	if opts.IncludeHTTP {
		allowedProtos = append(allowedProtos, "http")
	}
	if opts.IncludeCLI {
		allowedProtos = append(allowedProtos, "cli")
	}
	if opts.IncludeGRPC {
		allowedProtos = append(allowedProtos, "grpc")
	}

	info := ManualInfo{
		Title:       "xOSCAL High-Assurance NIST OSCAL 1.2.3 & Sovereign Compliance Engine",
		Version:     opts.Version,
		Description: "Universal Tool Calling Protocol (UTCP) v1.0.1 manual for xOSCAL compliance, ontology, graph, and feed operations across 36 sovereign frameworks (16,237 controls).",
		SpecURL:     SpecRepositoryURL,
		Ecosystem:   ecoMap,
	}

	var tools []Tool
	if opts.IncludeHTTP {
		tools = append(tools, buildHTTPTools(opts)...)
	}
	if opts.IncludeCLI {
		tools = append(tools, buildCLITools(opts)...)
	}
	if opts.IncludeGRPC {
		tools = append(tools, buildGRPCTools(opts)...)
	}

	return &Manual{
		UTCPVersion:                   SpecVersion,
		ManualVersion:                 opts.Version,
		Tools:                         tools,
		Name:                          "xoscal",
		Description:                   info.Description,
		AllowedCommunicationProtocols: allowedProtos,
		Metadata:                      &info,
		Info:                          info,
		Variables: map[string]string{
			"XOSCAL_BASE_URL":  opts.BaseURL,
			"XOSCAL_GRPC_HOST": opts.GRPCAddr,
			"XOSCAL_BIN":       opts.BinaryPath,
		},
	}
}

func stdToolOutputSchema() JSONSchema {
	return JSONSchema{
		Type: "object",
		Properties: map[string]JSONSchemaProperty{
			"status":   {Type: "string", Description: "Execution status or verdict"},
			"evidence": {Type: "string", Description: "Machine-verifiable JSON or structured report output"},
		},
	}
}

func protectedHTTPAuth(opts ManualOptions) *AuthSpec {
	envVar := strings.TrimSpace(opts.BearerTokenEnvVar)
	if envVar == "" {
		return nil
	}
	return &AuthSpec{
		AuthType: "api_key",
		APIKey:   fmt.Sprintf("Bearer ${%s}", envVar),
		VarName:  "Authorization",
		Location: "header",
	}
}

func buildHTTPTools(opts ManualOptions) []Tool {
	stdOut := stdToolOutputSchema()
	auth := protectedHTTPAuth(opts)

	mkHTTPTool := func(name, desc string, standing GovernanceStanding, tags []string, method, path string, protected bool, inputs JSONSchema) Tool {
		var toolAuth *AuthSpec
		if protected && auth != nil {
			cp := *auth
			toolAuth = &cp
		}
		ct := CallTemplate{
			Name:                          name,
			CallTemplateType:              "http",
			HTTPMethod:                    method,
			URL:                           opts.BaseURL + path,
			Headers:                       map[string]string{"Accept": "application/json"},
			AllowedCommunicationProtocols: []string{"http"},
			Auth:                          toolAuth,
			Governance: &GovernanceMetadata{
				OperationClass:  standing,
				AirGapCapable:   true,
				EvidenceChannel: "evidence-trace",
			},
		}
		if method == "POST" || method == "PUT" || method == "PATCH" {
			ct.ContentType = "application/json"
		}
		fullTags := ensureGovernanceTag(tags, standing)
		return Tool{
			Name:             name,
			Description:      desc,
			Tags:             fullTags,
			Inputs:           inputs,
			Outputs:          stdOut,
			ToolCallTemplate: ct,
			Governance:       standing,
		}
	}

	return []Tool{
		mkHTTPTool(
			"xoscal_health_check",
			"Probe xOSCAL gateway/server liveness and readiness status over HTTP.",
			GovernanceDiscover,
			[]string{"health", "probe", "http", "read-only"},
			"GET",
			"/healthz",
			false,
			JSONSchema{Type: "object"},
		),
		mkHTTPTool(
			"xoscal_doctor",
			"Run Rule 36 deep diagnostics across runtime substrate, embedded OSCAL 1.2.3 schemas, 36 registered frameworks, and unified property graph.",
			GovernanceDiscover,
			[]string{"diagnostics", "doctor", "oscal", "http", "read-only"},
			"GET",
			"/api/v1/doctor",
			true,
			JSONSchema{Type: "object"},
		),
		mkHTTPTool(
			"xoscal_frameworks_list",
			"List all 36 registered sovereign and international compliance frameworks with optional jurisdiction filtering.",
			GovernanceDiscover,
			[]string{"frameworks", "registry", "sovereign", "http", "read-only"},
			"GET",
			"/api/v1/frameworks",
			true,
			JSONSchema{
				Type: "object",
				Properties: map[string]JSONSchemaProperty{
					"jurisdiction": {Type: "string", Description: "Optional jurisdiction code filter (ca, us, uk, fr, de, eu)"},
				},
			},
		),
		mkHTTPTool(
			"xoscal_frameworks_get",
			"Retrieve metadata and control count summary for a registered compliance framework by ID or alias.",
			GovernanceDiscover,
			[]string{"frameworks", "inspect", "sovereign", "http", "read-only"},
			"GET",
			"/api/v1/frameworks/{id}",
			true,
			JSONSchema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]JSONSchemaProperty{
					"id": {Type: "string", Description: "Framework identifier (e.g., cccs-itsp-10-033, nist-sp-800-53-rev5)"},
				},
			},
		),
		mkHTTPTool(
			"xoscal_frameworks_layers",
			"List projected 7-layer NIST OSCAL 1.2.3 artifacts and SHA-256 digests for a compliance framework.",
			GovernanceDiscover,
			[]string{"frameworks", "layers", "oscal", "http", "read-only"},
			"GET",
			"/api/v1/frameworks/{id}/layers",
			true,
			JSONSchema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]JSONSchemaProperty{
					"id": {Type: "string", Description: "Framework identifier (e.g., cccs-itsp-10-033, cybersecure-canada)"},
				},
			},
		),
		mkHTTPTool(
			"xoscal_frameworks_parity",
			"Verify forensic verbatim parity and SHA-256 integrity for a compliance framework's OSCAL 1.2.3 projection.",
			GovernanceTrust,
			[]string{"frameworks", "parity", "conformance", "evidence", "http", "read-only"},
			"GET",
			"/api/v1/frameworks/{id}/parity",
			true,
			JSONSchema{
				Type:     "object",
				Required: []string{"id"},
				Properties: map[string]JSONSchemaProperty{
					"id": {Type: "string", Description: "Framework identifier (e.g., cybersecure-canada, cccs-itsp-10-033)"},
				},
			},
		),
		mkHTTPTool(
			"xoscal_controls_list",
			"List controls, families, ODP parameters, and sovereign guidance for a compliance framework.",
			GovernanceDiscover,
			[]string{"controls", "list", "oscal", "http", "read-only"},
			"GET",
			"/api/v1/controls",
			true,
			JSONSchema{
				Type: "object",
				Properties: map[string]JSONSchemaProperty{
					"framework": {Type: "string", Description: "Framework identifier (defaults to cccs-itsp-10-033)", Default: "cccs-itsp-10-033"},
				},
			},
		),
		mkHTTPTool(
			"xoscal_controls_search",
			"Full-text search across 16,237 controls, statements, ODP parameters, and sovereign guidance over HTTP.",
			GovernanceDiscover,
			[]string{"controls", "search", "oscal", "itsp-10-033", "http", "read-only"},
			"GET",
			"/api/v1/controls/search",
			true,
			JSONSchema{
				Type:     "object",
				Required: []string{"q"},
				Properties: map[string]JSONSchemaProperty{
					"q":         {Type: "string", Description: "Search keyword or phrase"},
					"framework": {Type: "string", Description: "Framework ID (e.g., cccs-itsp-10-033, nist-csf-2.0)", Default: "cccs-itsp-10-033"},
					"limit":     {Type: "integer", Description: "Optional maximum number of matching controls to return", Default: 25},
				},
			},
		),
		mkHTTPTool(
			"xoscal_controls_get",
			"Retrieve a specific control's verbatim title, hierarchical statement AST, ODP parameters, and sovereign guidance.",
			GovernanceDiscover,
			[]string{"controls", "inspect", "ast", "odp", "http", "read-only"},
			"GET",
			"/api/v1/controls/{framework}/{id}",
			true,
			JSONSchema{
				Type:     "object",
				Required: []string{"framework", "id"},
				Properties: map[string]JSONSchemaProperty{
					"framework": {Type: "string", Description: "Framework identifier (e.g., cccs-itsp-10-033)"},
					"id":        {Type: "string", Description: "Control identifier (e.g., ac-2, sc-7)"},
				},
			},
		),
		mkHTTPTool(
			"xoscal_ontology_get",
			"Retrieve the formal OWL/JSON-LD v1.0.0 regulatory and threat ontology specification (12 classes, 16 predicates).",
			GovernanceDiscover,
			[]string{"ontology", "jsonld", "owl", "http", "read-only"},
			"GET",
			"/api/v1/ontology",
			true,
			JSONSchema{Type: "object"},
		),
		mkHTTPTool(
			"xoscal_graph_stats",
			"Query node, edge, framework, feed, and anti-invariant statistics of the 36-framework unified property graph.",
			GovernanceDiscover,
			[]string{"graph", "stats", "telemetry", "http", "read-only"},
			"GET",
			"/api/v1/graph/stats",
			true,
			JSONSchema{Type: "object"},
		),
		mkHTTPTool(
			"xoscal_graph_crosswalk",
			"Query the 36-framework property graph for multi-jurisdiction control equivalency, harmonization, active threats, and 6D Vector State.",
			GovernanceDiscover,
			[]string{"graph", "crosswalk", "sovereign", "http", "read-only"},
			"GET",
			"/api/v1/graph/crosswalk",
			true,
			JSONSchema{
				Type:     "object",
				Required: []string{"control"},
				Properties: map[string]JSONSchemaProperty{
					"framework": {Type: "string", Description: "Optional source framework ID (e.g., cccs-itsp-10-033)"},
					"control":   {Type: "string", Description: "Control ID (e.g., ac-2, si-2, sc-7)", Default: "ac-2"},
				},
			},
		),
		mkHTTPTool(
			"xoscal_graph_blast_radius",
			"Compute multi-hop blast radius and 6D CKODEX Vector State across 36 frameworks for a CVE/KEV advisory, ATT&CK technique, framework, or control.",
			GovernanceDiscover,
			[]string{"graph", "blast-radius", "kev", "attack", "vector-state", "http", "read-only"},
			"GET",
			"/api/v1/graph/blast-radius",
			true,
			JSONSchema{
				Type:     "object",
				Required: []string{"node"},
				Properties: map[string]JSONSchemaProperty{
					"node":  {Type: "string", Description: "Target CVE/KEV ID, ATT&CK technique, framework, or control (e.g., CVE-2024-3400)", Default: "CVE-2024-3400"},
					"depth": {Type: "integer", Description: "Traversal hop depth (1..5)", Default: 2},
				},
			},
		),
		mkHTTPTool(
			"xoscal_graph_reconcile",
			"Verify Day-2 ontological invariants, structural coherence, and SHA-256 digest of the unified property graph.",
			GovernanceTrust,
			[]string{"graph", "reconcile", "coherence", "evidence", "http", "read-only"},
			"GET",
			"/api/v1/graph/reconcile",
			true,
			JSONSchema{Type: "object"},
		),
		mkHTTPTool(
			"xoscal_graph_export",
			"Export a framework or threat subgraph in JSON-LD, Cytoscape, or Cypher format.",
			GovernanceDiscover,
			[]string{"graph", "export", "jsonld", "cypher", "http", "read-only"},
			"GET",
			"/api/v1/graph/export",
			true,
			JSONSchema{
				Type: "object",
				Properties: map[string]JSONSchemaProperty{
					"framework": {Type: "string", Description: "Target framework, CVE, or control node identifier (e.g., cccs-itsp-10-033, CVE-2024-3400)", Default: "cccs-itsp-10-033"},
					"format":    {Type: "string", Description: "Export format (jsonld, cytoscape, cypher)", Default: "jsonld", Enum: []string{"jsonld", "cytoscape", "cypher"}},
				},
			},
		),
		mkHTTPTool(
			"xoscal_feeds_list",
			"List authoritative sovereign regulatory and threat intelligence feeds with SHA-256 digests.",
			GovernanceDiscover,
			[]string{"feeds", "kev", "attack", "http", "read-only"},
			"GET",
			"/api/v1/feeds",
			true,
			JSONSchema{Type: "object"},
		),
		mkHTTPTool(
			"xoscal_feeds_sync",
			"Synchronize sovereign regulatory and threat intelligence feeds, persist hash-chained flight-recorder receipts, and reload the unified graph.",
			GovernanceMutate,
			[]string{"feeds", "sync", "receipt", "mutate", "http"},
			"POST",
			"/api/v1/feeds/sync",
			true,
			JSONSchema{
				Type: "object",
				Properties: map[string]JSONSchemaProperty{
					"offline":  {Type: "boolean", Description: "Air-gap offline deterministic sync mode (defaults to true)", Default: true},
					"feed_ids": {Type: "array", Description: "Optional subset of feed IDs to synchronize", Items: &JSONSchemaProperty{Type: "string"}},
				},
			},
		),
		// HTTP POST invokes active schema & 6D Vector State evaluation on an arbitrary document payload
		// (classified as INVOKE per CKODEX Rule 6: TRUST != INVOKE, since HTTP POST is prohibited under DISCOVER/TRUST).
		mkHTTPTool(
			"xoscal_schema_validate",
			"Validate an OSCAL 1.2.3 JSON document against embedded NIST schemas and compute its 6D CKODEX Vector State posture.",
			GovernanceInvoke,
			[]string{"validate", "schema", "vector-state", "trust-verification", "http"},
			"POST",
			"/api/v1/validate",
			true,
			JSONSchema{
				Type:     "object",
				Required: []string{"document"},
				Properties: map[string]JSONSchemaProperty{
					"kind":     {Type: "string", Description: "Optional OSCAL model kind (catalog, profile, component-definition, ssp, assessment-plan, assessment-results, poam, mapping)"},
					"document": {Type: "object", Description: "OSCAL 1.2.3 JSON document to validate"},
				},
			},
		),
	}
}

func buildCLITools(opts ManualOptions) []Tool {
	stdOut := stdToolOutputSchema()
	prefix := ""
	if opts.IncludeHTTP {
		prefix = "cli_"
	}

	mkCLITool := func(baseName string, hasHTTPEquivalent bool, desc string, standing GovernanceStanding, tags []string, args []string, inputs JSONSchema) Tool {
		name := baseName
		if hasHTTPEquivalent && prefix != "" {
			name = "xoscal_" + prefix + strings.TrimPrefix(baseName, "xoscal_")
		}
		fullTags := ensureGovernanceTag(tags, standing)
		return Tool{
			Name:        name,
			Description: desc,
			Tags:        fullTags,
			Inputs:      inputs,
			Outputs:     stdOut,
			Governance:  standing,
			ToolCallTemplate: CallTemplate{
				Name:                          "xoscal_cli_" + strings.TrimPrefix(baseName, "xoscal_"),
				CallTemplateType:              "cli",
				Commands:                      buildCLICommands(opts.BinaryPath, args),
				AllowedCommunicationProtocols: []string{"cli"},
				Command:                       opts.BinaryPath,
				Args:                          args,
				Governance: &GovernanceMetadata{
					OperationClass:  standing,
					AirGapCapable:   true,
					EvidenceChannel: "evidence-trace",
				},
			},
		}
	}

	return []Tool{
		mkCLITool(
			"xoscal_doctor",
			true,
			"Run Rule 36 deep diagnostics via local xOSCAL CLI.",
			GovernanceDiscover,
			[]string{"diagnostics", "doctor", "oscal", "cli", "read-only"},
			[]string{"doctor", "-json"},
			JSONSchema{Type: "object"},
		),
		mkCLITool(
			"xoscal_frameworks_list",
			true,
			"List all 36 registered sovereign and international compliance frameworks via CLI.",
			GovernanceDiscover,
			[]string{"frameworks", "registry", "sovereign", "cli", "read-only"},
			[]string{"frameworks", "list", "-jurisdiction", "${jurisdiction}", "-json"},
			JSONSchema{
				Type: "object",
				Properties: map[string]JSONSchemaProperty{
					"jurisdiction": {Type: "string", Description: "Optional jurisdiction code filter (ca, us, uk, fr, de, eu)"},
				},
			},
		),
		mkCLITool(
			"xoscal_controls_search",
			true,
			"Full-text search across 16,237 controls via local xOSCAL CLI.",
			GovernanceDiscover,
			[]string{"controls", "search", "oscal", "cli", "read-only"},
			[]string{"controls", "search", "-framework", "${framework}", "-json", "${query}"},
			JSONSchema{
				Type:     "object",
				Required: []string{"query"},
				Properties: map[string]JSONSchemaProperty{
					"query":     {Type: "string", Description: "Search keyword or phrase"},
					"framework": {Type: "string", Description: "Framework ID", Default: "cccs-itsp-10-033"},
				},
			},
		),
		mkCLITool(
			"xoscal_controls_get",
			true,
			"Inspect a specific control's verbatim title, statement AST, ODP parameters, and sovereign guidance via CLI.",
			GovernanceDiscover,
			[]string{"controls", "inspect", "ast", "odp", "cli", "read-only"},
			[]string{"controls", "get", "-framework", "${framework}", "-json", "${control_id}"},
			JSONSchema{
				Type:     "object",
				Required: []string{"control_id"},
				Properties: map[string]JSONSchemaProperty{
					"control_id": {Type: "string", Description: "Control identifier (e.g., ac-2, sc-7)"},
					"framework":  {Type: "string", Description: "Framework ID", Default: "cccs-itsp-10-033"},
				},
			},
		),
		mkCLITool(
			"xoscal_compare_frameworks",
			false,
			"Compare two compliance frameworks and compute transition deltas.",
			GovernanceDiscover,
			[]string{"compare", "transition", "delta", "cli", "read-only"},
			[]string{"compare", "-json", "${source_framework}", "${target_framework}"},
			JSONSchema{
				Type:     "object",
				Required: []string{"source_framework", "target_framework"},
				Properties: map[string]JSONSchemaProperty{
					"source_framework": {Type: "string", Description: "Source baseline framework ID (e.g., cccs-itsg-33)"},
					"target_framework": {Type: "string", Description: "Target framework ID (e.g., cccs-itsp-10-033)"},
				},
			},
		),
		mkCLITool(
			"xoscal_verify_parity",
			false,
			"Execute forensic verbatim parity verification across all 7 OSCAL 1.2.3 layers and SHA-256 sidecars.",
			GovernanceTrust,
			[]string{"verify-parity", "conformance", "evidence", "oscal", "cli"},
			[]string{"verify-parity", "-dir", "${dir}", "-framework", "${framework}", "-strict", "-json"},
			JSONSchema{
				Type:     "object",
				Required: []string{"dir"},
				Properties: map[string]JSONSchemaProperty{
					"dir":       {Type: "string", Description: "Directory containing exported framework artifacts", Default: "data/frameworks"},
					"framework": {Type: "string", Description: "Optional specific framework ID"},
				},
			},
		),
		mkCLITool(
			"xoscal_verify_vector_state",
			false,
			"Evaluate an OSCAL artifact against embedded NIST 1.2.3 schemas and compute the 6D CKODEX Vector State.",
			GovernanceTrust,
			[]string{"verify", "vector-state", "schema", "posture", "cli"},
			[]string{"verify", "-file", "${file}", "-json"},
			JSONSchema{
				Type:     "object",
				Required: []string{"file"},
				Properties: map[string]JSONSchemaProperty{
					"file": {Type: "string", Description: "Path to OSCAL JSON artifact"},
				},
			},
		),
		mkCLITool(
			"xoscal_seed_knowledge_graph",
			false,
			"Seed the SQLite Knowledge Graph with authoritative requirements from registered compliance frameworks.",
			GovernanceMutate,
			[]string{"seed", "knowledge-graph", "sqlite", "mutate", "cli"},
			[]string{"seed", "-dsn", "${dsn}", "-framework", "${framework}", "-snapshot", "${snapshot}"},
			JSONSchema{
				Type:     "object",
				Required: []string{"dsn", "snapshot"},
				Properties: map[string]JSONSchemaProperty{
					"dsn":       {Type: "string", Description: "SQLite database path", Default: "oscal.db"},
					"framework": {Type: "string", Description: "Framework ID to seed (empty for all 36 frameworks)"},
					"snapshot":  {Type: "string", Description: "Snapshot name to create", Default: "v1"},
				},
			},
		),
		mkCLITool(
			"xoscal_generate_artifacts",
			false,
			"Generate schema-validated 7-layer OSCAL 1.2.3 artifacts and SHA-256 sidecars from a Knowledge Graph snapshot.",
			GovernanceMutate,
			[]string{"generate", "oscal", "knowledge-graph", "mutate", "cli"},
			[]string{"generate", "-dsn", "${dsn}", "-snapshot", "${snapshot}", "-framework", "${framework}", "-out", "${out}", "-validate"},
			JSONSchema{
				Type:     "object",
				Required: []string{"dsn", "snapshot", "framework", "out"},
				Properties: map[string]JSONSchemaProperty{
					"dsn":       {Type: "string", Description: "SQLite database path"},
					"snapshot":  {Type: "string", Description: "Snapshot name"},
					"framework": {Type: "string", Description: "Framework ID"},
					"out":       {Type: "string", Description: "Output directory for generated OSCAL JSON and .sha256 sidecars"},
				},
			},
		),
		mkCLITool(
			"xoscal_graph_crosswalk",
			true,
			"Query the 36-framework property graph for multi-jurisdiction control equivalency via CLI.",
			GovernanceDiscover,
			[]string{"graph", "crosswalk", "ontology", "sovereign", "cli", "read-only"},
			[]string{"graph", "crosswalk", "-control", "${control}", "-json"},
			JSONSchema{
				Type:     "object",
				Required: []string{"control"},
				Properties: map[string]JSONSchemaProperty{
					"control": {Type: "string", Description: "Control ID or <framework>:<control-id>", Default: "ac-2"},
				},
			},
		),
		mkCLITool(
			"xoscal_graph_blast_radius",
			true,
			"Compute the multi-hop blast radius and 6D CKODEX Vector State via CLI.",
			GovernanceTrust,
			[]string{"graph", "blast-radius", "kev", "attack", "vector-state", "cli", "read-only"},
			[]string{"graph", "blast-radius", "-target", "${target}", "-json"},
			JSONSchema{
				Type:     "object",
				Required: []string{"target"},
				Properties: map[string]JSONSchemaProperty{
					"target": {Type: "string", Description: "Target CVE/KEV ID, ATT&CK technique, framework, or control", Default: "CVE-2024-3400"},
				},
			},
		),
		mkCLITool(
			"xoscal_feeds_sync",
			true,
			"Synchronize the 4 authoritative sovereign regulatory and threat intelligence feeds via CLI.",
			GovernanceMutate,
			[]string{"feeds", "sync", "kev", "attack", "air-gap", "receipt", "cli"},
			[]string{"feeds", "sync", "-offline", "-json"},
			JSONSchema{
				Type: "object",
				Properties: map[string]JSONSchemaProperty{
					"offline": {Type: "boolean", Description: "Air-gap offline deterministic sync mode (Rule 40)", Default: true},
				},
			},
		),
	}
}

func buildGRPCTools(opts ManualOptions) []Tool {
	stdOut := stdToolOutputSchema()
	return []Tool{
		{
			Name:        "xoscal_grpc_generate_catalog",
			Description: "Invoke oscal.services.v1.GovernanceService/GenerateCatalog directly over native gRPC + Protobuf.",
			Tags:        ensureGovernanceTag([]string{"grpc", "protobuf", "governance-service", "catalog"}, GovernanceInvoke),
			Governance:  GovernanceInvoke,
			Inputs: JSONSchema{
				Type:     "object",
				Required: []string{"snapshot_name", "framework"},
				Properties: map[string]JSONSchemaProperty{
					"snapshot_name": {Type: "string", Description: "Knowledge Graph snapshot identifier"},
					"framework":     {Type: "string", Description: "Target compliance framework ID"},
				},
			},
			Outputs: stdOut,
			ToolCallTemplate: CallTemplate{
				Name:                          "xoscal_grpc_governance_generate_catalog",
				CallTemplateType:              "grpc",
				GRPCHost:                      opts.GRPCAddr,
				GRPCService:                   "oscal.services.v1.GovernanceService",
				GRPCMethod:                    "GenerateCatalog",
				AllowedCommunicationProtocols: []string{"grpc"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceInvoke,
					RequiredAuthority: "urn:ckodex:authority:operator",
					EvidenceChannel:   "execution-trace",
					AirGapCapable:     true,
				},
			},
		},
		{
			Name:        "xoscal_grpc_evaluate_governance",
			Description: "Invoke oscal.services.v1.GovernanceService/GenerateAssessmentResults for governance assessment evaluation over native gRPC.",
			Tags:        ensureGovernanceTag([]string{"grpc", "protobuf", "governance-service"}, GovernanceDiscover),
			Governance:  GovernanceDiscover,
			Inputs: JSONSchema{
				Type:     "object",
				Required: []string{"snapshot_name", "framework"},
				Properties: map[string]JSONSchemaProperty{
					"snapshot_name": {Type: "string", Description: "Knowledge Graph snapshot identifier"},
					"framework":     {Type: "string", Description: "Target compliance framework ID"},
				},
			},
			Outputs: stdOut,
			ToolCallTemplate: CallTemplate{
				Name:                          "xoscal_grpc_governance_generate_assessment_results",
				CallTemplateType:              "grpc",
				GRPCHost:                      opts.GRPCAddr,
				GRPCService:                   "oscal.services.v1.GovernanceService",
				GRPCMethod:                    "GenerateAssessmentResults",
				AllowedCommunicationProtocols: []string{"grpc"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceDiscover,
					RequiredAuthority: "urn:ckodex:authority:auditor",
					EvidenceChannel:   "decision-trace",
					AirGapCapable:     true,
				},
			},
		},
	}
}

// ValidationFinding represents a structural or constitutional finding on a UTCP manual.
type ValidationFinding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"` // "ERROR", "ANTI_INVARIANT", "WARNING"
	ToolName string `json:"tool_name,omitempty"`
	Message  string `json:"message"`
}

// ValidationReport summarizes the validation and CKODEX conformance of a UTCP manual.
type ValidationReport struct {
	Valid         bool                `json:"valid"`
	Verdict       string              `json:"verdict"` // "CONFORMANT" or "NON_CONFORMANT"
	UTCPVersion   string              `json:"utcp_version"`
	ManualVersion string              `json:"manual_version"`
	ToolCount     int                 `json:"tool_count"`
	ManualSHA256  string              `json:"manual_sha256"`
	Transports    map[string]int      `json:"transports"`
	Findings      []ValidationFinding `json:"findings,omitempty"`
}

var allowedCallTemplateTypes = map[string]bool{
	"cli":  true,
	"http": true,
	"grpc": true,
	"sse":  true,
	"text": true,
	"mcp":  true,
}

func isForbiddenWebSocketProto(p string) bool {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "websocket", "ws", "wss":
		return true
	default:
		return false
	}
}

func hasForbiddenWebSocketScheme(rawURL string) bool {
	u := strings.ToLower(strings.TrimSpace(rawURL))
	return strings.HasPrefix(u, "ws://") || strings.HasPrefix(u, "wss://")
}

// ValidateManual validates raw UTCP Manual JSON bytes against the UTCP v1.0.1
// specification and CKODEX constitutional invariants (including Rule 10 No WebSockets
// and Rule 6 Governance State Separation).
func ValidateManual(data []byte) (*ValidationReport, error) {
	sum := sha256.Sum256(data)
	rep := &ValidationReport{
		Valid:        true,
		Verdict:      "CONFORMANT",
		ManualSHA256: hex.EncodeToString(sum[:]),
		Transports:   make(map[string]int),
	}

	var manual Manual
	if err := json.Unmarshal(data, &manual); err != nil {
		return nil, fmt.Errorf("invalid UTCP manual JSON: %w", err)
	}

	var rawRoot map[string]any
	_ = json.Unmarshal(data, &rawRoot)

	rep.UTCPVersion = manual.UTCPVersion
	rep.ManualVersion = manual.ManualVersion
	rep.ToolCount = len(manual.Tools)

	if strings.TrimSpace(manual.UTCPVersion) == "" {
		rep.addError("MISSING_UTCP_VERSION", "", "utcp_version is required by utcp-specification")
	}
	if strings.TrimSpace(manual.ManualVersion) == "" {
		rep.addError("MISSING_MANUAL_VERSION", "", "manual_version is required by utcp-specification")
	}
	if len(manual.Tools) == 0 {
		rep.addError("EMPTY_TOOLS", "", "UTCP manual must declare at least one tool in tools[]")
	}

	// Inspect any root-level allowed_communication_protocols present in raw JSON
	if rawProtos, ok := rawRoot["allowed_communication_protocols"].([]any); ok {
		for _, p := range rawProtos {
			if ps, ok := p.(string); ok && isForbiddenWebSocketProto(ps) {
				rep.addAntiInvariant("ANTI_WEBSOCKET_FORBIDDEN", "", fmt.Sprintf("CKODEX Rule 10 violation: forbidden protocol %q in manual allowed_communication_protocols", ps))
			}
		}
	}

	var rawTools []any
	if rt, ok := rawRoot["tools"].([]any); ok {
		rawTools = rt
	}

	seenNames := make(map[string]bool, len(manual.Tools))
	for idx, t := range manual.Tools {
		name := strings.TrimSpace(t.Name)
		if name == "" {
			rep.addError("MISSING_TOOL_NAME", "", "tool entry is missing required 'name'")
			continue
		}
		if seenNames[name] {
			rep.addError("DUPLICATE_TOOL_NAME", name, fmt.Sprintf("duplicate tool name %q violates UTCP uniqueness constraint", name))
		}
		seenNames[name] = true

		if strings.TrimSpace(t.Description) == "" {
			rep.addError("MISSING_TOOL_DESCRIPTION", name, "tool entry is missing required 'description'")
		}
		if t.Inputs.Type != "object" {
			rep.addError("INVALID_INPUTS_SCHEMA", name, fmt.Sprintf("tool %q inputs.type must be 'object', got %q", name, t.Inputs.Type))
		}
		if t.Outputs.Type != "object" {
			rep.addError("INVALID_OUTPUTS_SCHEMA", name, fmt.Sprintf("tool %q outputs.type must be 'object', got %q", name, t.Outputs.Type))
		}

		ct := strings.ToLower(strings.TrimSpace(t.ToolCallTemplate.CallTemplateType))
		rep.Transports[ct]++

		// CKODEX Constitutional Rule 10 Hard Anti-Invariant: Reject websocket/ws/wss in call_template_type,
		// allowed_communication_protocols, or URL schemes.
		wsViolation := isForbiddenWebSocketProto(ct) || hasForbiddenWebSocketScheme(t.ToolCallTemplate.URL)
		for _, proto := range t.ToolCallTemplate.AllowedCommunicationProtocols {
			if isForbiddenWebSocketProto(proto) {
				wsViolation = true
				break
			}
		}
		if idx < len(rawTools) {
			if rawToolMap, ok := rawTools[idx].(map[string]any); ok {
				if rawProts, ok := rawToolMap["allowed_communication_protocols"].([]any); ok {
					for _, p := range rawProts {
						if ps, ok := p.(string); ok && isForbiddenWebSocketProto(ps) {
							wsViolation = true
							break
						}
					}
				}
			}
		}
		if wsViolation {
			rep.addAntiInvariant(
				"ANTI_WEBSOCKET_FORBIDDEN",
				name,
				"CKODEX Rule 10 violation: WebSocket transport (websocket/ws/wss) is forbidden; use gRPC streaming, HTTP, CLI, or SSE instead",
			)
			continue
		}

		if !allowedCallTemplateTypes[ct] {
			rep.addError("UNSUPPORTED_CALL_TEMPLATE_TYPE", name, fmt.Sprintf("unsupported call_template_type %q", ct))
			continue
		}

		switch ct {
		case "cli":
			hasCmds := len(t.ToolCallTemplate.Commands) > 0 && strings.TrimSpace(t.ToolCallTemplate.Commands[0].Command) != ""
			if !hasCmds {
				rep.addError("MISSING_CLI_COMMAND", name, "cli call_template_type requires non-empty 'commands' array")
			}
		case "http", "sse":
			if strings.TrimSpace(t.ToolCallTemplate.URL) == "" {
				rep.addError("MISSING_HTTP_URL", name, fmt.Sprintf("%s call_template_type requires non-empty 'url'", ct))
			}
			if ct == "http" && strings.TrimSpace(t.ToolCallTemplate.HTTPMethod) == "" {
				rep.addError("MISSING_HTTP_METHOD", name, "http call_template_type requires 'http_method'")
			}
		case "grpc":
			if strings.TrimSpace(t.ToolCallTemplate.GRPCService) == "" || strings.TrimSpace(t.ToolCallTemplate.GRPCMethod) == "" {
				rep.addError("MISSING_GRPC_TARGET", name, "grpc call_template_type requires 'grpc_service' and 'grpc_method'")
			}
		case "text":
			if strings.TrimSpace(t.ToolCallTemplate.FilePath) == "" && strings.TrimSpace(t.ToolCallTemplate.URL) == "" {
				rep.addError("MISSING_TEXT_SOURCE", name, "text call_template_type requires 'file_path' or 'url'")
			}
		case "mcp":
			if t.ToolCallTemplate.Config == nil || len(t.ToolCallTemplate.Config.MCPServers) == 0 {
				rep.addError("MISSING_MCP_SERVERS", name, "mcp call_template_type requires non-empty 'config.mcpServers' map")
			}
		}

		// CKODEX Rule 6: Reject HTTP POST, PUT, PATCH, or DELETE if governance standing is DISCOVER or TRUST.
		standing := t.effectiveGovernance()
		if standing == "" && idx < len(rawTools) {
			standing = extractRawGovernanceStanding(rawTools[idx])
		}
		if ct == "http" {
			method := strings.ToUpper(strings.TrimSpace(t.ToolCallTemplate.HTTPMethod))
			if method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE" {
				if standing == GovernanceDiscover || standing == GovernanceTrust {
					rep.addAntiInvariant(
						"ANTI_GOVERNANCE_COLLAPSE",
						name,
						fmt.Sprintf("CKODEX Rule 6 violation: HTTP %s tool cannot be classified as %s", method, standing),
					)
				}
			}
		}
	}

	return rep, nil
}

func extractRawGovernanceStanding(rawTool any) GovernanceStanding {
	m, ok := rawTool.(map[string]any)
	if !ok {
		return ""
	}
	if gs, ok := m["governance"].(string); ok && strings.TrimSpace(gs) != "" {
		return GovernanceStanding(strings.ToUpper(strings.TrimSpace(gs)))
	}
	if ctMap, ok := m["tool_call_template"].(map[string]any); ok {
		for _, k := range []string{"x-ckodex-governance", "x_ckodex_governance"} {
			if govObj, ok := ctMap[k].(map[string]any); ok {
				if oc, ok := govObj["operation_class"].(string); ok && strings.TrimSpace(oc) != "" {
					return GovernanceStanding(strings.ToUpper(strings.TrimSpace(oc)))
				}
			}
		}
	}
	return ""
}

func (r *ValidationReport) addError(code, tool, msg string) {
	r.Valid = false
	r.Verdict = "NON_CONFORMANT"
	r.Findings = append(r.Findings, ValidationFinding{
		Code:     code,
		Severity: "ERROR",
		ToolName: tool,
		Message:  msg,
	})
}

func (r *ValidationReport) addAntiInvariant(code, tool, msg string) {
	r.Valid = false
	r.Verdict = "NON_CONFORMANT"
	r.Findings = append(r.Findings, ValidationFinding{
		Code:     code,
		Severity: "ANTI_INVARIANT",
		ToolName: tool,
		Message:  msg,
	})
}
