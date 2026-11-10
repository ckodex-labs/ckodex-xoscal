package utcp

import (
	"encoding/json"
	"strings"
	"testing"

	"google.golang.org/grpc"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
)

func TestGenerateAndValidateStandardManual(t *testing.T) {
	manual := BuildCanonicalManual(ManualOptions{
		BaseHTTPURL: "http://127.0.0.1:8080",
		GRPCHost:    "127.0.0.1:50051",
		BinaryPath:  "xoscal",
		Version:     "1.0.0",
	})

	if len(manual.AllowedCommunicationProtocols) == 0 {
		t.Fatalf("expected AllowedCommunicationProtocols populated on Manual")
	}

	for _, tool := range manual.Tools {
		if len(tool.ToolCallTemplate.AllowedCommunicationProtocols) == 0 {
			t.Errorf("tool %q missing AllowedCommunicationProtocols", tool.Name)
		}
		if tool.ToolCallTemplate.CallTemplateType == "cli" {
			if strings.TrimSpace(tool.ToolCallTemplate.Command) == "" {
				t.Errorf("cli tool %q missing Command", tool.Name)
			}
			if len(tool.ToolCallTemplate.Commands) == 0 || strings.TrimSpace(tool.ToolCallTemplate.Commands[0].Command) == "" {
				t.Errorf("cli tool %q missing Commands[] entry", tool.Name)
			} else {
				cmdEntry := tool.ToolCallTemplate.Commands[0]
				if !cmdEntry.AppendToFinalOutput {
					t.Errorf("cli tool %q Commands[0].AppendToFinalOutput = false, want true", tool.Name)
				}
				if strings.Contains(cmdEntry.Command, "${") {
					t.Errorf("cli tool %q Commands[0].Command still contains unexpanded ${...} placeholder: %q", tool.Name, cmdEntry.Command)
				}
			}
		}
	}

	data, err := json.MarshalIndent(manual, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}

	rep, err := ValidateManual(data)
	if err != nil {
		t.Fatalf("ValidateManual: %v", err)
	}
	if !rep.Valid || rep.Verdict != "CONFORMANT" {
		t.Fatalf("expected CONFORMANT manual, got valid=%t verdict=%s findings=%+v", rep.Valid, rep.Verdict, rep.Findings)
	}
	if rep.UTCPVersion != SpecVersion {
		t.Errorf("UTCPVersion = %q, want %q", rep.UTCPVersion, SpecVersion)
	}
	if rep.ToolCount < 10 {
		t.Errorf("expected at least 10 tools in xOSCAL UTCP manual, got %d", rep.ToolCount)
	}
	for _, transport := range []string{"cli", "http", "grpc"} {
		if rep.Transports[transport] == 0 {
			t.Errorf("expected at least 1 %q tool in manual, got 0", transport)
		}
	}
}

func TestGRPCToolsReferenceProtoDefinitions(t *testing.T) {
	serviceDescs := []grpc.ServiceDesc{
		servicesv1.GovernanceService_ServiceDesc,
		servicesv1.OscalService_ServiceDesc,
		servicesv1.TransparencyExchangeService_ServiceDesc,
		servicesv1.TransparencyGraphService_ServiceDesc,
	}

	validMethods := make(map[string]map[string]bool)
	for _, sd := range serviceDescs {
		methods := make(map[string]bool, len(sd.Methods))
		for _, m := range sd.Methods {
			methods[m.MethodName] = true
		}
		validMethods[sd.ServiceName] = methods
	}

	manual := BuildCanonicalManual(ManualOptions{})
	grpcCount := 0
	for _, tool := range manual.Tools {
		if tool.ToolCallTemplate.CallTemplateType != "grpc" {
			continue
		}
		grpcCount++
		svc := tool.ToolCallTemplate.GRPCService
		method := tool.ToolCallTemplate.GRPCMethod
		methods, ok := validMethods[svc]
		if !ok {
			t.Errorf("gRPC tool %q references unknown protobuf service %q", tool.Name, svc)
			continue
		}
		if !methods[method] {
			t.Errorf("gRPC tool %q references unknown RPC method %q on service %q", tool.Name, method, svc)
		}
	}
	if grpcCount == 0 {
		t.Fatalf("expected at least 1 gRPC tool in canonical manual")
	}
}

func TestCLITemplateRequiredInputs(t *testing.T) {
	manual := BuildCanonicalManual(ManualOptions{})
	toolMap := make(map[string]Tool)
	for _, tool := range manual.Tools {
		toolMap[tool.Name] = tool
	}

	hasReq := func(reqs []string, want string) bool {
		for _, r := range reqs {
			if r == want {
				return true
			}
		}
		return false
	}

	vp, ok := toolMap["xoscal_verify_parity"]
	if !ok {
		t.Fatalf("missing xoscal_verify_parity tool")
	}
	if !hasReq(vp.Inputs.Required, "dir") {
		t.Errorf("xoscal_verify_parity Required = %v, want 'dir' included", vp.Inputs.Required)
	}

	seed, ok := toolMap["xoscal_seed_knowledge_graph"]
	if !ok {
		t.Fatalf("missing xoscal_seed_knowledge_graph tool")
	}
	for _, req := range []string{"dsn", "snapshot"} {
		if !hasReq(seed.Inputs.Required, req) {
			t.Errorf("xoscal_seed_knowledge_graph Required = %v, want %q included", seed.Inputs.Required, req)
		}
	}

	cw, ok := toolMap["xoscal_graph_crosswalk"]
	if !ok {
		t.Fatalf("missing xoscal_graph_crosswalk tool")
	}
	if !hasReq(cw.Inputs.Required, "control") {
		t.Errorf("xoscal_graph_crosswalk Required = %v, want 'control' included", cw.Inputs.Required)
	}

	br, ok := toolMap["xoscal_graph_blast_radius"]
	if !ok {
		t.Fatalf("missing xoscal_graph_blast_radius tool")
	}
	if !hasReq(br.Inputs.Required, "target") {
		t.Errorf("xoscal_graph_blast_radius Required = %v, want 'target' included", br.Inputs.Required)
	}

	if _, ok := toolMap["xoscal_feeds_sync"]; !ok {
		t.Fatalf("missing xoscal_feeds_sync tool")
	}
}

func TestValidateManualAcceptsCommandsOnlyCLI(t *testing.T) {
	commandsOnly := Manual{
		ManualVersion: "1.0.0",
		UTCPVersion:   SpecVersion,
		Info:          ManualInfo{Title: "Commands Only CLI", Version: "1.0.0"},
		Tools: []Tool{
			{
				Name:        "cli_commands_tool",
				Description: "CLI tool using upstream python/ts commands[] array",
				Inputs:      JSONSchema{Type: "object"},
				Outputs:     JSONSchema{Type: "object"},
				ToolCallTemplate: CallTemplate{
					CallTemplateType: "cli",
					Commands: []CLICommand{
						{Command: "xoscal doctor -json", AppendToFinalOutput: true},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(commandsOnly)
	rep, err := ValidateManual(data)
	if err != nil {
		t.Fatalf("ValidateManual: %v", err)
	}
	if !rep.Valid || rep.Verdict != "CONFORMANT" {
		t.Fatalf("expected CONFORMANT for commands[] CLI tool, got valid=%t findings=%+v", rep.Valid, rep.Findings)
	}
}

func TestSupportedRepositoriesCoverage(t *testing.T) {
	expectedURLs := []string{
		"https://github.com/universal-tool-calling-protocol",
		"https://github.com/universal-tool-calling-protocol/utcp-specification",
		"https://github.com/universal-tool-calling-protocol/utcp-mcp",
		"https://github.com/universal-tool-calling-protocol/rs-utcp",
		"https://github.com/universal-tool-calling-protocol/typescript-utcp",
		"https://github.com/universal-tool-calling-protocol/go-utcp",
		"https://github.com/universal-tool-calling-protocol/python-utcp",
	}

	urlSet := make(map[string]bool)
	for _, r := range SupportedRepositories {
		urlSet[r.URL] = true
	}

	for _, want := range expectedURLs {
		if !urlSet[want] {
			t.Errorf("SupportedRepositories missing required repository URL %q", want)
		}
	}
}

func TestValidateManualRejectsWebSocketPerRule10(t *testing.T) {
	badManual := Manual{
		ManualVersion: "1.0.0",
		UTCPVersion:   SpecVersion,
		Info:          ManualInfo{Title: "Bad WS Manual", Version: "1.0.0"},
		Tools: []Tool{
			{
				Name:        "ws_stream_tool",
				Description: "Forbidden websocket tool",
				Inputs:      JSONSchema{Type: "object"},
				Outputs:     JSONSchema{Type: "object"},
				ToolCallTemplate: CallTemplate{
					CallTemplateType: "websocket",
					URL:              "wss://example.com/stream",
				},
			},
		},
	}

	data, _ := json.Marshal(badManual)
	rep, err := ValidateManual(data)
	if err != nil {
		t.Fatalf("ValidateManual: %v", err)
	}
	if rep.Valid || rep.Verdict != "NON_CONFORMANT" {
		t.Fatalf("expected NON_CONFORMANT for websocket tool, got valid=%t verdict=%s", rep.Valid, rep.Verdict)
	}
	foundAntiWS := false
	for _, f := range rep.Findings {
		if f.Code == "ANTI_WEBSOCKET_FORBIDDEN" {
			foundAntiWS = true
			break
		}
	}
	if !foundAntiWS {
		t.Errorf("expected ANTI_WEBSOCKET_FORBIDDEN finding, got %+v", rep.Findings)
	}
}

func TestGenerateSDKHarnessAndMCPBridge(t *testing.T) {
	targets := []struct {
		target     string
		expect     string
		namespaced bool
	}{
		{"go-utcp", "github.com/universal-tool-calling-protocol/go-utcp", true},
		{"python-utcp", "from utcp.utcp_client import UtcpClient", true},
		{"typescript-utcp", `from "@utcp/sdk"`, true},
		{"rs-utcp", "use rs_utcp::{UtcpClient, UtcpClientConfig, CallTemplate};", true},
		{"utcp-mcp", "https://github.com/universal-tool-calling-protocol/utcp-mcp", false},
	}

	for _, tc := range targets {
		code, err := GenerateSDKHarness(tc.target, "http://127.0.0.1:8080/.well-known/utcp", "./utcp.json")
		if err != nil {
			t.Fatalf("GenerateSDKHarness(%q): %v", tc.target, err)
		}
		if !strings.Contains(code, tc.expect) {
			t.Errorf("GenerateSDKHarness(%q) missing %q in output:\n%s", tc.target, tc.expect, code)
		}
		if tc.namespaced && !strings.Contains(code, "xoscal_http_manual.xoscal_controls_search") {
			t.Errorf("GenerateSDKHarness(%q) missing namespaced tool 'xoscal_http_manual.xoscal_controls_search' in output:\n%s", tc.target, code)
		}
	}

	discCfg, err := BuildDiscoveryConfig("http://127.0.0.1:8080/.well-known/utcp", "./utcp.json")
	if err != nil {
		t.Fatalf("BuildDiscoveryConfig: %v", err)
	}
	if !strings.Contains(string(discCfg), `"allowed_communication_protocols"`) {
		t.Errorf("BuildDiscoveryConfig missing allowed_communication_protocols: %s", string(discCfg))
	}
}
