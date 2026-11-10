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

// Manual represents a UTCP v1.0.1 tool manual document published at /.well-known/utcp or utcp.json.
type Manual struct {
	ManualVersion                 string            `json:"manual_version"`
	UTCPVersion                   string            `json:"utcp_version"`
	AllowedCommunicationProtocols []string          `json:"allowed_communication_protocols,omitempty"`
	Info                          ManualInfo        `json:"info"`
	Variables                     map[string]string `json:"variables,omitempty"`
	Auth                          *AuthSpec         `json:"auth,omitempty"`
	Tools                         []Tool            `json:"tools"`
}

// ManualInfo describes the provider metadata in a UTCP manual.
type ManualInfo struct {
	Title       string            `json:"title"`
	Version     string            `json:"version"`
	Description string            `json:"description,omitempty"`
	SpecURL     string            `json:"spec_url,omitempty"`
	Ecosystem   map[string]string `json:"x-utcp-ecosystem,omitempty"`
}

// AuthSpec describes authentication configuration for UTCP call templates.
type AuthSpec struct {
	AuthType string `json:"auth_type"` // "api_key", "bearer", "basic", "oauth2", "none"
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
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Default     any      `json:"default,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// GovernanceMetadata attaches CKODEX constitutional authority & state classification to a tool template.
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

// CallTemplate defines the native protocol execution instructions for a UTCP tool.
type CallTemplate struct {
	Name                          string              `json:"name,omitempty"`
	CallTemplateType              string              `json:"call_template_type"` // "cli", "http", "grpc", "sse", "text", "mcp"
	AllowedCommunicationProtocols []string            `json:"allowed_communication_protocols,omitempty"`
	Command                       string              `json:"command,omitempty"`
	Args                          []string            `json:"args,omitempty"`
	Commands                      []CLICommand        `json:"commands,omitempty"`
	WorkingDir                    string              `json:"working_dir,omitempty"`
	EnvVars                       map[string]string   `json:"env_vars,omitempty"`
	URL                           string              `json:"url,omitempty"`
	HTTPMethod                    string              `json:"http_method,omitempty"`
	ContentType                   string              `json:"content_type,omitempty"`
	Headers                       map[string]string   `json:"headers,omitempty"`
	BodyField                     string              `json:"body_field,omitempty"`
	GRPCService                   string              `json:"grpc_service,omitempty"`
	GRPCMethod                    string              `json:"grpc_method,omitempty"`
	GRPCHost                      string              `json:"grpc_host,omitempty"`
	FilePath                      string              `json:"file_path,omitempty"`
	Auth                          *AuthSpec           `json:"auth,omitempty"`
	Governance                    *GovernanceMetadata `json:"x-ckodex-governance,omitempty"`
}

// Tool represents an individual callable tool entry in a UTCP manual.
type Tool struct {
	Name             string       `json:"name"`
	Description      string       `json:"description"`
	Tags             []string     `json:"tags,omitempty"`
	Inputs           JSONSchema   `json:"inputs"`
	Outputs          JSONSchema   `json:"outputs"`
	ToolCallTemplate CallTemplate `json:"tool_call_template"`
}

// ManualOptions configures generation of the xOSCAL UTCP Manual.
type ManualOptions struct {
	BaseHTTPURL string
	GRPCHost    string
	BinaryPath  string
	Version     string
}

var utcpTemplatePlaceholderRe = regexp.MustCompile(`\$\{([a-zA-Z0-9_]+)\}`)

func buildCLICommands(bin string, args []string) []CLICommand {
	parts := make([]string, 0, 1+len(args))
	if strings.TrimSpace(bin) != "" {
		parts = append(parts, strings.TrimSpace(bin))
	}
	for _, a := range args {
		converted := utcpTemplatePlaceholderRe.ReplaceAllString(a, "UTCP_ARG_${1}_UTCP_END")
		parts = append(parts, converted)
	}
	return []CLICommand{
		{
			Command:             strings.Join(parts, " "),
			AppendToFinalOutput: true,
		},
	}
}

// BuildCanonicalManual is an alias for GenerateManual returning the canonical xOSCAL UTCP v1.0.1 Manual.
func BuildCanonicalManual(opts ManualOptions) *Manual {
	return GenerateManual(opts)
}

// GenerateManual synthesizes the complete xOSCAL UTCP v1.0.1 Manual covering
// native CLI, HTTP (Web Studio + REST Gateway), and gRPC transports.
func GenerateManual(opts ManualOptions) *Manual {
	if opts.BaseHTTPURL == "" {
		opts.BaseHTTPURL = "http://127.0.0.1:8080"
	}
	opts.BaseHTTPURL = strings.TrimRight(opts.BaseHTTPURL, "/")
	if opts.GRPCHost == "" {
		opts.GRPCHost = "127.0.0.1:50051"
	}
	if opts.BinaryPath == "" {
		opts.BinaryPath = "xoscal"
	}
	if opts.Version == "" {
		opts.Version = DefaultManualVersion
	}

	ecoMap := make(map[string]string, len(SupportedRepositories))
	for _, r := range SupportedRepositories {
		ecoMap[r.Name] = r.URL
	}

	return &Manual{
		ManualVersion:                 opts.Version,
		UTCPVersion:                   SpecVersion,
		AllowedCommunicationProtocols: DefaultAllowedCommunicationProtocols(),
		Info: ManualInfo{
			Title:       "xOSCAL High-Assurance NIST OSCAL 1.2.3 & Sovereign Compliance Engine",
			Version:     opts.Version,
			Description: "Universal Tool Calling Protocol (UTCP) manual for direct native invocation of xOSCAL CLI, HTTP REST, and gRPC compliance tools across 36 sovereign frameworks (16,237 controls).",
			SpecURL:     SpecRepositoryURL,
			Ecosystem:   ecoMap,
		},
		Variables: map[string]string{
			"XOSCAL_BASE_URL":  opts.BaseHTTPURL,
			"XOSCAL_GRPC_HOST": opts.GRPCHost,
			"XOSCAL_BIN":       opts.BinaryPath,
		},
		Tools: buildStandardTools(opts),
	}
}

func buildStandardTools(opts ManualOptions) []Tool {
	stdOutSchema := JSONSchema{
		Type: "object",
		Properties: map[string]JSONSchemaProperty{
			"status":   {Type: "string", Description: "Execution status or verdict"},
			"evidence": {Type: "string", Description: "Machine-verifiable JSON or structured report output"},
		},
	}

	tools := []Tool{
		{
			Name:        "xoscal_doctor",
			Description: "Run Rule 36 deep diagnostics across runtime substrate, embedded OSCAL 1.2.3 schemas, 36 registered frameworks, SQLite KG, and Tier-2 oscal-cli.",
			Tags:        []string{"diagnostics", "doctor", "oscal", "cli", "read-only"},
			Inputs: JSONSchema{
				Type: "object",
				Properties: map[string]JSONSchemaProperty{
					"json": {Type: "boolean", Description: "Emit machine-verifiable JSON decision/evidence trace", Default: true},
				},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_cli_doctor",
				CallTemplateType: "cli",
				Command:          opts.BinaryPath,
				Args:             []string{"doctor", "-json"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceDiscover,
					RequiredAuthority: "urn:ckodex:authority:observer",
					EvidenceChannel:   "decision-trace",
					AirGapCapable:     true,
				},
			},
		},
		{
			Name:        "xoscal_frameworks_list",
			Description: "List all 36 registered sovereign and international compliance frameworks (Canada, USA, UK, France, Germany, EU) with control counts.",
			Tags:        []string{"frameworks", "registry", "sovereign", "cli", "read-only"},
			Inputs: JSONSchema{
				Type: "object",
				Properties: map[string]JSONSchemaProperty{
					"jurisdiction": {Type: "string", Description: "Optional jurisdiction code filter (ca, us, uk, fr, de, eu)"},
				},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_cli_frameworks_list",
				CallTemplateType: "cli",
				Command:          opts.BinaryPath,
				Args:             []string{"frameworks", "list", "-jurisdiction", "${jurisdiction}", "-json"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceDiscover,
					RequiredAuthority: "urn:ckodex:authority:observer",
					EvidenceChannel:   "telemetry-trace",
					AirGapCapable:     true,
				},
			},
		},
		{
			Name:        "xoscal_controls_search",
			Description: "Full-text search across 16,237 controls, statements, ODP parameters, and sovereign guidance in any registered framework.",
			Tags:        []string{"controls", "search", "oscal", "itsp-10-033", "cli"},
			Inputs: JSONSchema{
				Type:     "object",
				Required: []string{"query"},
				Properties: map[string]JSONSchemaProperty{
					"query":     {Type: "string", Description: "Search keyword or phrase"},
					"framework": {Type: "string", Description: "Framework ID (e.g., cccs-itsp-10-033, nist-csf-2.0)", Default: "cccs-itsp-10-033"},
				},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_cli_controls_search",
				CallTemplateType: "cli",
				Command:          opts.BinaryPath,
				Args:             []string{"controls", "search", "-framework", "${framework}", "-json", "${query}"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceDiscover,
					RequiredAuthority: "urn:ckodex:authority:observer",
					EvidenceChannel:   "decision-trace",
					AirGapCapable:     true,
				},
			},
		},
		{
			Name:        "xoscal_controls_get",
			Description: "Inspect a specific control's verbatim title, hierarchical statement AST, ODP parameters, guidance, GC discussion, and enhancements.",
			Tags:        []string{"controls", "inspect", "ast", "odp", "cli"},
			Inputs: JSONSchema{
				Type:     "object",
				Required: []string{"control_id"},
				Properties: map[string]JSONSchemaProperty{
					"control_id": {Type: "string", Description: "Control identifier (e.g., ac-2, sc-7)"},
					"framework":  {Type: "string", Description: "Framework ID", Default: "cccs-itsp-10-033"},
				},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_cli_controls_get",
				CallTemplateType: "cli",
				Command:          opts.BinaryPath,
				Args:             []string{"controls", "get", "-framework", "${framework}", "-json", "${control_id}"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceDiscover,
					RequiredAuthority: "urn:ckodex:authority:observer",
					EvidenceChannel:   "evidence-trace",
					AirGapCapable:     true,
				},
			},
		},
		{
			Name:        "xoscal_compare_frameworks",
			Description: "Compare two compliance frameworks and compute transition deltas (added, modified, retained, removed controls and family breakdown).",
			Tags:        []string{"compare", "transition", "delta", "itsg-33", "itsp-10-033", "cli"},
			Inputs: JSONSchema{
				Type:     "object",
				Required: []string{"source_framework", "target_framework"},
				Properties: map[string]JSONSchemaProperty{
					"source_framework": {Type: "string", Description: "Source baseline framework ID (e.g., cccs-itsg-33)"},
					"target_framework": {Type: "string", Description: "Target framework ID (e.g., cccs-itsp-10-033)"},
				},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_cli_compare",
				CallTemplateType: "cli",
				Command:          opts.BinaryPath,
				Args:             []string{"compare", "-json", "${source_framework}", "${target_framework}"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceDiscover,
					RequiredAuthority: "urn:ckodex:authority:auditor",
					EvidenceChannel:   "decision-trace",
					AirGapCapable:     true,
				},
			},
		},
		{
			Name:        "xoscal_verify_parity",
			Description: "Execute forensic verbatim parity verification across all 7 OSCAL 1.2.3 layers, SHA-256 sidecars, independent baseline inventory, and Knowledge Graph round-trip.",
			Tags:        []string{"verify-parity", "conformance", "evidence", "oscal", "cli"},
			Inputs: JSONSchema{
				Type:     "object",
				Required: []string{"dir"},
				Properties: map[string]JSONSchemaProperty{
					"dir":       {Type: "string", Description: "Directory containing exported framework artifacts", Default: "data/frameworks"},
					"framework": {Type: "string", Description: "Optional specific framework ID"},
				},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_cli_verify_parity",
				CallTemplateType: "cli",
				Command:          opts.BinaryPath,
				Args:             []string{"verify-parity", "-dir", "${dir}", "-framework", "${framework}", "-strict", "-json"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceTrust,
					RequiredAuthority: "urn:ckodex:authority:verifier",
					EvidenceChannel:   "evidence-trace",
					AirGapCapable:     true,
				},
			},
		},
		{
			Name:        "xoscal_verify_vector_state",
			Description: "Evaluate an OSCAL artifact against embedded NIST 1.2.3 schemas and compute the 6D CKODEX Vector State S(e,t) = <P,V,A,C,E,L,tau>.",
			Tags:        []string{"verify", "vector-state", "schema", "posture", "cli"},
			Inputs: JSONSchema{
				Type:     "object",
				Required: []string{"file"},
				Properties: map[string]JSONSchemaProperty{
					"file": {Type: "string", Description: "Path to OSCAL JSON artifact"},
				},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_cli_verify",
				CallTemplateType: "cli",
				Command:          opts.BinaryPath,
				Args:             []string{"verify", "-file", "${file}", "-json"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceTrust,
					RequiredAuthority: "urn:ckodex:authority:verifier",
					EvidenceChannel:   "evidence-trace",
					AirGapCapable:     true,
				},
			},
		},
		{
			Name:        "xoscal_seed_knowledge_graph",
			Description: "Seed the SQLite Knowledge Graph with authoritative requirements from registered compliance frameworks.",
			Tags:        []string{"seed", "knowledge-graph", "sqlite", "mutate", "cli"},
			Inputs: JSONSchema{
				Type:     "object",
				Required: []string{"dsn", "snapshot"},
				Properties: map[string]JSONSchemaProperty{
					"dsn":       {Type: "string", Description: "SQLite database path", Default: "oscal.db"},
					"framework": {Type: "string", Description: "Framework ID to seed (empty for all 36 frameworks)"},
					"snapshot":  {Type: "string", Description: "Snapshot name to create", Default: "v1"},
				},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_cli_seed",
				CallTemplateType: "cli",
				Command:          opts.BinaryPath,
				Args:             []string{"seed", "-dsn", "${dsn}", "-framework", "${framework}", "-snapshot", "${snapshot}"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceMutate,
					RequiredAuthority: "urn:ckodex:authority:operator",
					EvidenceChannel:   "evidence-trace",
					AirGapCapable:     true,
				},
			},
		},
		{
			Name:        "xoscal_generate_artifacts",
			Description: "Generate schema-validated 7-layer OSCAL 1.2.3 artifacts and SHA-256 sidecars from a Knowledge Graph snapshot.",
			Tags:        []string{"generate", "oscal", "knowledge-graph", "mutate", "cli"},
			Inputs: JSONSchema{
				Type:     "object",
				Required: []string{"dsn", "snapshot", "framework", "out"},
				Properties: map[string]JSONSchemaProperty{
					"dsn":       {Type: "string", Description: "SQLite database path"},
					"snapshot":  {Type: "string", Description: "Snapshot name"},
					"framework": {Type: "string", Description: "Framework ID"},
					"out":       {Type: "string", Description: "Output directory for generated OSCAL JSON and .sha256 sidecars"},
				},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_cli_generate",
				CallTemplateType: "cli",
				Command:          opts.BinaryPath,
				Args:             []string{"generate", "-dsn", "${dsn}", "-snapshot", "${snapshot}", "-framework", "${framework}", "-out", "${out}", "-validate"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceMutate,
					RequiredAuthority: "urn:ckodex:authority:operator",
					EvidenceChannel:   "evidence-trace",
					AirGapCapable:     true,
				},
			},
		},
		{
			Name:        "xoscal_graph_crosswalk",
			Description: "Query the 36-framework property graph for multi-jurisdiction control equivalency (EQUIVALENT_TO), thematic harmonization (HARMONIZES_WITH), active KEV/ATT&CK threats, and 6D Vector State posture.",
			Tags:        []string{"graph", "crosswalk", "ontology", "sovereign", "cli", "read-only"},
			Inputs: JSONSchema{
				Type:     "object",
				Required: []string{"control"},
				Properties: map[string]JSONSchemaProperty{
					"control": {Type: "string", Description: "Control ID or <framework>:<control-id> (e.g., ac-2, si-2, sc-7, cccs-itsp-10-033:ac-2)", Default: "ac-2"},
				},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_cli_graph_crosswalk",
				CallTemplateType: "cli",
				Command:          opts.BinaryPath,
				Args:             []string{"graph", "crosswalk", "-control", "${control}", "-json"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceDiscover,
					RequiredAuthority: "urn:ckodex:authority:auditor",
					EvidenceChannel:   "decision-trace",
					AirGapCapable:     true,
				},
			},
		},
		{
			Name:        "xoscal_graph_blast_radius",
			Description: "Compute the multi-hop blast radius and 6D CKODEX Vector State S(e,t) = <P,V,A,C,E,L,tau> across 36 frameworks for a CVE/KEV advisory, ATT&CK technique, framework, or control.",
			Tags:        []string{"graph", "blast-radius", "kev", "attack", "vector-state", "cli", "read-only"},
			Inputs: JSONSchema{
				Type:     "object",
				Required: []string{"target"},
				Properties: map[string]JSONSchemaProperty{
					"target": {Type: "string", Description: "Target CVE/KEV ID, ATT&CK technique, framework, or control (e.g., CVE-2024-3400, T1190, cccs-itsg-33)", Default: "CVE-2024-3400"},
				},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_cli_graph_blast_radius",
				CallTemplateType: "cli",
				Command:          opts.BinaryPath,
				Args:             []string{"graph", "blast-radius", "-target", "${target}", "-json"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceTrust,
					RequiredAuthority: "urn:ckodex:authority:verifier",
					EvidenceChannel:   "evidence-trace",
					AirGapCapable:     true,
				},
			},
		},
		{
			Name:        "xoscal_feeds_sync",
			Description: "Synchronize the 4 authoritative sovereign regulatory and threat intelligence feeds (CCCS/NIST/EU Crosswalk, CISA KEV, MITRE ATT&CK v16, Sovereign Lifecycle) in air-gap offline mode and emit SHA-256 execution receipts.",
			Tags:        []string{"feeds", "sync", "kev", "attack", "air-gap", "receipt", "cli"},
			Inputs: JSONSchema{
				Type: "object",
				Properties: map[string]JSONSchemaProperty{
					"offline": {Type: "boolean", Description: "Air-gap offline deterministic sync mode (Rule 40)", Default: true},
				},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_cli_feeds_sync",
				CallTemplateType: "cli",
				Command:          opts.BinaryPath,
				Args:             []string{"feeds", "sync", "-offline", "-json"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceMutate,
					RequiredAuthority: "urn:ckodex:authority:operator",
					EvidenceChannel:   "evidence-trace",
					AirGapCapable:     true,
				},
			},
		},
		{
			Name:        "xoscal_http_list_frameworks",
			Description: "Query the xOSCAL HTTP API for all registered compliance frameworks and control summaries.",
			Tags:        []string{"http", "rest", "frameworks", "read-only"},
			Inputs: JSONSchema{
				Type:       "object",
				Properties: map[string]JSONSchemaProperty{},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_http_frameworks",
				CallTemplateType: "http",
				URL:              opts.BaseHTTPURL + "/api/v1/frameworks",
				HTTPMethod:       "GET",
				Headers:          map[string]string{"Accept": "application/json"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceDiscover,
					RequiredAuthority: "urn:ckodex:authority:observer",
					EvidenceChannel:   "telemetry-trace",
					AirGapCapable:     true,
				},
			},
		},
		{
			Name:        "xoscal_http_compare_frameworks",
			Description: "Compute a live framework transition matrix over HTTP between two registered frameworks.",
			Tags:        []string{"http", "rest", "compare", "transition"},
			Inputs: JSONSchema{
				Type:     "object",
				Required: []string{"from", "to"},
				Properties: map[string]JSONSchemaProperty{
					"from": {Type: "string", Description: "Source framework ID"},
					"to":   {Type: "string", Description: "Target framework ID"},
				},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_http_compare",
				CallTemplateType: "http",
				URL:              opts.BaseHTTPURL + "/api/v1/compare?from=${from}&to=${to}",
				HTTPMethod:       "GET",
				Headers:          map[string]string{"Accept": "application/json"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceDiscover,
					RequiredAuthority: "urn:ckodex:authority:auditor",
					EvidenceChannel:   "decision-trace",
					AirGapCapable:     true,
				},
			},
		},
		{
			Name:        "xoscal_http_validate_artifact",
			Description: "Validate an OSCAL 1.2.3 JSON artifact and evaluate its 6D Vector State posture over HTTP.",
			Tags:        []string{"http", "rest", "validate", "vector-state"},
			Inputs: JSONSchema{
				Type:     "object",
				Required: []string{"artifact"},
				Properties: map[string]JSONSchemaProperty{
					"artifact": {Type: "object", Description: "OSCAL 1.2.3 JSON document"},
				},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_http_validate",
				CallTemplateType: "http",
				URL:              opts.BaseHTTPURL + "/api/v1/validate",
				HTTPMethod:       "POST",
				ContentType:      "application/json",
				BodyField:        "artifact",
				Headers:          map[string]string{"Accept": "application/json"},
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceTrust,
					RequiredAuthority: "urn:ckodex:authority:verifier",
					EvidenceChannel:   "evidence-trace",
					AirGapCapable:     true,
				},
			},
		},
		{
			Name:        "xoscal_grpc_generate_catalog",
			Description: "Invoke oscal.services.v1.GovernanceService/GenerateCatalog directly over native gRPC + Protobuf.",
			Tags:        []string{"grpc", "protobuf", "governance-service", "catalog"},
			Inputs: JSONSchema{
				Type:     "object",
				Required: []string{"snapshot_name", "framework"},
				Properties: map[string]JSONSchemaProperty{
					"snapshot_name": {Type: "string", Description: "Knowledge Graph snapshot identifier"},
					"framework":     {Type: "string", Description: "Target compliance framework ID"},
				},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_grpc_governance_generate_catalog",
				CallTemplateType: "grpc",
				GRPCHost:         opts.GRPCHost,
				GRPCService:      "oscal.services.v1.GovernanceService",
				GRPCMethod:       "GenerateCatalog",
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
			Tags:        []string{"grpc", "protobuf", "governance-service"},
			Inputs: JSONSchema{
				Type:     "object",
				Required: []string{"snapshot_name", "framework"},
				Properties: map[string]JSONSchemaProperty{
					"snapshot_name": {Type: "string", Description: "Knowledge Graph snapshot identifier"},
					"framework":     {Type: "string", Description: "Target compliance framework ID"},
				},
			},
			Outputs: stdOutSchema,
			ToolCallTemplate: CallTemplate{
				Name:             "xoscal_grpc_governance_generate_assessment_results",
				CallTemplateType: "grpc",
				GRPCHost:         opts.GRPCHost,
				GRPCService:      "oscal.services.v1.GovernanceService",
				GRPCMethod:       "GenerateAssessmentResults",
				Governance: &GovernanceMetadata{
					OperationClass:    GovernanceDiscover,
					RequiredAuthority: "urn:ckodex:authority:auditor",
					EvidenceChannel:   "decision-trace",
					AirGapCapable:     true,
				},
			},
		},
	}

	for i := range tools {
		tools[i].ToolCallTemplate.AllowedCommunicationProtocols = DefaultAllowedCommunicationProtocols()
		if tools[i].ToolCallTemplate.CallTemplateType == "cli" {
			tools[i].ToolCallTemplate.Commands = buildCLICommands(tools[i].ToolCallTemplate.Command, tools[i].ToolCallTemplate.Args)
		}
	}

	return tools
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

	seenNames := make(map[string]bool, len(manual.Tools))
	for _, t := range manual.Tools {
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

		// CKODEX Constitutional Rule 10 Hard Anti-Invariant: Do not introduce WebSockets.
		if ct == "websocket" || ct == "ws" || strings.HasPrefix(strings.ToLower(t.ToolCallTemplate.URL), "ws://") || strings.HasPrefix(strings.ToLower(t.ToolCallTemplate.URL), "wss://") {
			rep.Valid = false
			rep.Verdict = "NON_CONFORMANT"
			rep.Findings = append(rep.Findings, ValidationFinding{
				Code:     "ANTI_WEBSOCKET_FORBIDDEN",
				Severity: "ANTI_INVARIANT",
				ToolName: name,
				Message:  "CKODEX Rule 10 violation: WebSocket transport is forbidden; use gRPC streaming, HTTP, CLI, or SSE instead",
			})
			continue
		}

		if !allowedCallTemplateTypes[ct] {
			rep.addError("UNSUPPORTED_CALL_TEMPLATE_TYPE", name, fmt.Sprintf("unsupported call_template_type %q", ct))
			continue
		}

		switch ct {
		case "cli":
			hasCmd := strings.TrimSpace(t.ToolCallTemplate.Command) != ""
			hasCmds := len(t.ToolCallTemplate.Commands) > 0 && strings.TrimSpace(t.ToolCallTemplate.Commands[0].Command) != ""
			if !hasCmd && !hasCmds {
				rep.addError("MISSING_CLI_COMMAND", name, "cli call_template_type requires non-empty 'command' or 'commands'")
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
		}

		// CKODEX Rule 6: DISCOVER != MUTATE check
		if t.ToolCallTemplate.Governance != nil {
			method := strings.ToUpper(strings.TrimSpace(t.ToolCallTemplate.HTTPMethod))
			if (method == "DELETE" || method == "PUT") && t.ToolCallTemplate.Governance.OperationClass == GovernanceDiscover {
				rep.addError("ANTI_GOVERNANCE_COLLAPSE", name, fmt.Sprintf("CKODEX Rule 6 violation: HTTP %s tool cannot be classified as DISCOVER", method))
			}
		}
	}

	return rep, nil
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
