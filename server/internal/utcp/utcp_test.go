package utcp

import (
	"encoding/json"
	"strings"
	"testing"

	"google.golang.org/grpc"

	servicesv1 "github.com/mchorfa/xoscal/proto/oscal/services/v1"
)

func assertOnlyAllowedKeys(t *testing.T, contextLabel string, obj map[string]any, allowed map[string]bool) {
	t.Helper()
	for k := range obj {
		if !allowed[k] {
			t.Errorf("%s contains non-standard/unknown JSON key %q (allowed: %v)", contextLabel, k, allowed)
		}
	}
}

func TestStrictZodWireSchema_CanonicalHTTPManual(t *testing.T) {
	manual := BuildCanonicalManual("http://127.0.0.1:8080", "127.0.0.1:50051")

	data, err := json.MarshalIndent(manual, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}

	// Verify UtcpManualSchema.strict() exact root keys: ONLY utcp_version, manual_version, tools
	var rawRoot map[string]any
	if err := json.Unmarshal(data, &rawRoot); err != nil {
		t.Fatalf("Unmarshal rawRoot: %v", err)
	}
	allowedManualKeys := map[string]bool{
		"utcp_version":   true,
		"manual_version": true,
		"tools":          true,
	}
	assertOnlyAllowedKeys(t, "Manual root", rawRoot, allowedManualKeys)
	if len(rawRoot) != 3 {
		t.Fatalf("Manual root key count = %d, want exact 3 keys (utcp_version, manual_version, tools): %v", len(rawRoot), rawRoot)
	}

	allowedToolKeys := map[string]bool{
		"name":               true,
		"description":        true,
		"tags":               true,
		"inputs":             true,
		"outputs":            true,
		"tool_call_template": true,
	}
	allowedHTTPCallTemplateKeys := map[string]bool{
		"name":                            true,
		"call_template_type":              true,
		"http_method":                     true,
		"url":                             true,
		"content_type":                    true,
		"auth":                            true,
		"headers":                         true,
		"body_field":                      true,
		"header_fields":                   true,
		"allowed_communication_protocols": true,
	}

	rawTools, ok := rawRoot["tools"].([]any)
	if !ok {
		t.Fatalf("tools is not an array")
	}
	if len(rawTools) != 18 {
		t.Fatalf("expected 18 HTTP tools in canonical manual, got %d", len(rawTools))
	}

	expectedHTTPTools := []string{
		"xoscal_health_check",
		"xoscal_doctor",
		"xoscal_frameworks_list",
		"xoscal_frameworks_get",
		"xoscal_frameworks_layers",
		"xoscal_frameworks_parity",
		"xoscal_controls_list",
		"xoscal_controls_search",
		"xoscal_controls_get",
		"xoscal_ontology_get",
		"xoscal_graph_stats",
		"xoscal_graph_crosswalk",
		"xoscal_graph_blast_radius",
		"xoscal_graph_reconcile",
		"xoscal_graph_export",
		"xoscal_feeds_list",
		"xoscal_feeds_sync",
		"xoscal_schema_validate",
	}
	seenTools := make(map[string]bool, len(rawTools))

	for i, item := range rawTools {
		tm, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("tool[%d] is not an object", i)
		}
		name, _ := tm["name"].(string)
		seenTools[name] = true
		assertOnlyAllowedKeys(t, "Tool "+name, tm, allowedToolKeys)

		ct, ok := tm["tool_call_template"].(map[string]any)
		if !ok {
			t.Fatalf("tool %q missing tool_call_template object", name)
		}
		if ct["call_template_type"] != "http" {
			t.Errorf("canonical manual tool %q has call_template_type=%v, want \"http\"", name, ct["call_template_type"])
		}
		assertOnlyAllowedKeys(t, "HttpCallTemplate "+name, ct, allowedHTTPCallTemplateKeys)
	}

	for _, wantName := range expectedHTTPTools {
		if !seenTools[wantName] {
			t.Errorf("canonical HTTP manual missing required tool %q", wantName)
		}
	}

	rep, err := ValidateManual(data)
	if err != nil {
		t.Fatalf("ValidateManual: %v", err)
	}
	if !rep.Valid || rep.Verdict != "CONFORMANT" {
		t.Fatalf("expected CONFORMANT manual, got valid=%t verdict=%s findings=%+v", rep.Valid, rep.Verdict, rep.Findings)
	}
	if rep.Transports["http"] != 18 || rep.Transports["cli"] != 0 || rep.Transports["grpc"] != 0 {
		t.Errorf("unexpected transport counts for canonical HTTP-only manual: %+v", rep.Transports)
	}
}

func TestStrictZodWireSchema_CLICallTemplateAndQuoting(t *testing.T) {
	manual := BuildManualWithOptions(ManualOptions{
		BaseURL:    "http://127.0.0.1:8080",
		BinaryPath: "xoscal",
		IncludeCLI: true,
	})

	data, err := json.Marshal(manual)
	if err != nil {
		t.Fatalf("Marshal CLI manual: %v", err)
	}

	var rawRoot map[string]any
	if err := json.Unmarshal(data, &rawRoot); err != nil {
		t.Fatalf("Unmarshal CLI manual: %v", err)
	}

	allowedCLICallTemplateKeys := map[string]bool{
		"name":                            true,
		"call_template_type":              true,
		"commands":                        true,
		"working_dir":                     true,
		"env_vars":                        true,
		"allowed_communication_protocols": true,
	}
	allowedCLICommandKeys := map[string]bool{
		"command":                true,
		"append_to_final_output": true,
	}

	rawTools := rawRoot["tools"].([]any)
	foundQuotedJurisdiction := false
	for _, item := range rawTools {
		tm := item.(map[string]any)
		name, _ := tm["name"].(string)
		ct := tm["tool_call_template"].(map[string]any)
		if ct["call_template_type"] != "cli" {
			t.Fatalf("expected cli call_template_type for %q, got %v", name, ct["call_template_type"])
		}
		// Strictly reject command, args, x_ckodex_governance, x-ckodex-governance
		assertOnlyAllowedKeys(t, "CliCallTemplate "+name, ct, allowedCLICallTemplateKeys)

		cmds, ok := ct["commands"].([]any)
		if !ok || len(cmds) == 0 {
			t.Fatalf("cli tool %q missing commands[]", name)
		}
		cmd0 := cmds[0].(map[string]any)
		assertOnlyAllowedKeys(t, "CLICommand "+name, cmd0, allowedCLICommandKeys)
		cmdStr, _ := cmd0["command"].(string)
		if strings.Contains(cmdStr, "${") {
			t.Errorf("cli tool %q command still contains unexpanded ${...}: %q", name, cmdStr)
		}
		if name == "xoscal_frameworks_list" && strings.Contains(cmdStr, `"UTCP_ARG_jurisdiction_UTCP_END" -json`) {
			foundQuotedJurisdiction = true
		}
	}
	if !foundQuotedJurisdiction {
		t.Errorf("expected quoted \"UTCP_ARG_jurisdiction_UTCP_END\" placeholder before -json in xoscal_frameworks_list")
	}

	rep, err := ValidateManual(data)
	if err != nil {
		t.Fatalf("ValidateManual CLI: %v", err)
	}
	if !rep.Valid || rep.Verdict != "CONFORMANT" {
		t.Fatalf("CLI manual failed validation: %+v", rep.Findings)
	}
}

func TestBearerTokenEnvVarPopulatesApiKeyAuth(t *testing.T) {
	manual := BuildManualWithOptions(ManualOptions{
		BaseURL:           "http://127.0.0.1:8080",
		IncludeHTTP:       true,
		BearerTokenEnvVar: "XOSCAL_API_TOKEN",
	})

	data, err := json.Marshal(manual)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var rawRoot map[string]any
	if err := json.Unmarshal(data, &rawRoot); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	allowedAuthKeys := map[string]bool{
		"auth_type": true,
		"api_key":   true,
		"var_name":  true,
		"location":  true,
	}

	for _, item := range rawRoot["tools"].([]any) {
		tm := item.(map[string]any)
		name, _ := tm["name"].(string)
		ct := tm["tool_call_template"].(map[string]any)
		if name == "xoscal_health_check" {
			if _, hasAuth := ct["auth"]; hasAuth {
				t.Errorf("public probe tool xoscal_health_check should not require auth")
			}
			continue
		}
		authMap, ok := ct["auth"].(map[string]any)
		if !ok {
			t.Fatalf("protected HTTP tool %q missing auth object", name)
		}
		assertOnlyAllowedKeys(t, "ApiKeyAuth "+name, authMap, allowedAuthKeys)
		if authMap["auth_type"] != "api_key" ||
			authMap["api_key"] != "Bearer ${XOSCAL_API_TOKEN}" ||
			authMap["var_name"] != "Authorization" ||
			authMap["location"] != "header" {
			t.Errorf("unexpected auth block on %q: %+v", name, authMap)
		}
	}
}

func TestFullManualAndGRPCToolsReferenceProtoDefinitions(t *testing.T) {
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

	manual := BuildFullManual("http://127.0.0.1:8080", "127.0.0.1:50051")
	data, err := json.Marshal(manual)
	if err != nil {
		t.Fatalf("Marshal full manual: %v", err)
	}
	rep, err := ValidateManual(data)
	if err != nil {
		t.Fatalf("ValidateManual full manual: %v", err)
	}
	if !rep.Valid || rep.Verdict != "CONFORMANT" {
		t.Fatalf("expected CONFORMANT full manual, got valid=%t findings=%+v", rep.Valid, rep.Findings)
	}

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
		t.Fatalf("expected at least 1 gRPC tool in full manual")
	}

	ext := manual.ExtendedEnvelope()
	extBytes, err := json.Marshal(ext)
	if err != nil {
		t.Fatalf("Marshal ExtendedEnvelope: %v", err)
	}
	if !strings.Contains(string(extBytes), `"metadata"`) || !strings.Contains(string(extBytes), `"allowed_communication_protocols"`) {
		t.Errorf("ExtendedEnvelope missing expected metadata fields: %s", string(extBytes))
	}
}

func TestValidateManual_NegativeGovernanceWebSocketAndMCP(t *testing.T) {
	cases := []struct {
		name     string
		manual   Manual
		wantCode string
	}{
		{
			name: "POST with DISCOVER standing rejected",
			manual: Manual{
				UTCPVersion:   SpecVersion,
				ManualVersion: "1.0.0",
				Tools: []Tool{{
					Name:        "bad_post_discover",
					Description: "Mutating POST falsely tagged as DISCOVER",
					Tags:        []string{"governance:DISCOVER"},
					Inputs:      JSONSchema{Type: "object"},
					Outputs:     JSONSchema{Type: "object"},
					ToolCallTemplate: CallTemplate{
						CallTemplateType: "http",
						HTTPMethod:       "POST",
						URL:              "http://127.0.0.1:8080/api/v1/feeds/sync",
					},
				}},
			},
			wantCode: "ANTI_GOVERNANCE_COLLAPSE",
		},
		{
			name: "PATCH with DISCOVER standing rejected",
			manual: Manual{
				UTCPVersion:   SpecVersion,
				ManualVersion: "1.0.0",
				Tools: []Tool{{
					Name:        "bad_patch_discover",
					Description: "Mutating PATCH falsely tagged as DISCOVER",
					Governance:  GovernanceDiscover,
					Inputs:      JSONSchema{Type: "object"},
					Outputs:     JSONSchema{Type: "object"},
					ToolCallTemplate: CallTemplate{
						CallTemplateType: "http",
						HTTPMethod:       "PATCH",
						URL:              "http://127.0.0.1:8080/api/v1/resource",
					},
				}},
			},
			wantCode: "ANTI_GOVERNANCE_COLLAPSE",
		},
		{
			name: "DELETE with TRUST standing rejected",
			manual: Manual{
				UTCPVersion:   SpecVersion,
				ManualVersion: "1.0.0",
				Tools: []Tool{{
					Name:        "bad_delete_trust",
					Description: "Mutating DELETE falsely tagged as TRUST",
					Tags:        []string{"governance:TRUST"},
					Inputs:      JSONSchema{Type: "object"},
					Outputs:     JSONSchema{Type: "object"},
					ToolCallTemplate: CallTemplate{
						CallTemplateType: "http",
						HTTPMethod:       "DELETE",
						URL:              "http://127.0.0.1:8080/api/v1/resource",
					},
				}},
			},
			wantCode: "ANTI_GOVERNANCE_COLLAPSE",
		},
		{
			name: "AllowedCommunicationProtocols containing ws rejected",
			manual: Manual{
				UTCPVersion:   SpecVersion,
				ManualVersion: "1.0.0",
				Tools: []Tool{{
					Name:        "ws_proto_leak",
					Description: "HTTP tool advertising ws in allowed_communication_protocols",
					Tags:        []string{"governance:DISCOVER"},
					Inputs:      JSONSchema{Type: "object"},
					Outputs:     JSONSchema{Type: "object"},
					ToolCallTemplate: CallTemplate{
						CallTemplateType:              "http",
						HTTPMethod:                    "GET",
						URL:                           "http://127.0.0.1:8080/healthz",
						AllowedCommunicationProtocols: []string{"http", "ws"},
					},
				}},
			},
			wantCode: "ANTI_WEBSOCKET_FORBIDDEN",
		},
		{
			name: "WebSocket call_template_type and wss URL rejected",
			manual: Manual{
				UTCPVersion:   SpecVersion,
				ManualVersion: "1.0.0",
				Tools: []Tool{{
					Name:        "ws_stream_tool",
					Description: "Forbidden websocket tool",
					Inputs:      JSONSchema{Type: "object"},
					Outputs:     JSONSchema{Type: "object"},
					ToolCallTemplate: CallTemplate{
						CallTemplateType: "websocket",
						URL:              "wss://example.com/stream",
					},
				}},
			},
			wantCode: "ANTI_WEBSOCKET_FORBIDDEN",
		},
		{
			name: "MCP call template with nil config rejected",
			manual: Manual{
				UTCPVersion:   SpecVersion,
				ManualVersion: "1.0.0",
				Tools: []Tool{{
					Name:        "empty_mcp_tool",
					Description: "MCP tool missing config.mcpServers",
					Tags:        []string{"governance:INVOKE"},
					Inputs:      JSONSchema{Type: "object"},
					Outputs:     JSONSchema{Type: "object"},
					ToolCallTemplate: CallTemplate{
						CallTemplateType: "mcp",
					},
				}},
			},
			wantCode: "MISSING_MCP_SERVERS",
		},
		{
			name: "MCP call template with empty mcpServers map rejected",
			manual: Manual{
				UTCPVersion:   SpecVersion,
				ManualVersion: "1.0.0",
				Tools: []Tool{{
					Name:        "empty_mcp_servers_map",
					Description: "MCP tool with empty mcpServers map",
					Tags:        []string{"governance:INVOKE"},
					Inputs:      JSONSchema{Type: "object"},
					Outputs:     JSONSchema{Type: "object"},
					ToolCallTemplate: CallTemplate{
						CallTemplateType: "mcp",
						Config:           &MCPTemplateConfig{MCPServers: map[string]MCPServer{}},
					},
				}},
			},
			wantCode: "MISSING_MCP_SERVERS",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.manual)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			rep, err := ValidateManual(data)
			if err != nil {
				t.Fatalf("ValidateManual: %v", err)
			}
			if rep.Valid || rep.Verdict != "NON_CONFORMANT" {
				t.Fatalf("expected NON_CONFORMANT, got valid=%t verdict=%s findings=%+v", rep.Valid, rep.Verdict, rep.Findings)
			}
			found := false
			for _, f := range rep.Findings {
				if f.Code == tc.wantCode {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected finding %q, got %+v", tc.wantCode, rep.Findings)
			}
		})
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

func TestGenerateSDKHarnessAndMCPBridge_7RepoConformance(t *testing.T) {
	// 1. Pure .utcp_config.json generator
	utcpCfgBytes, err := GenerateUTCPClientConfig("http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("GenerateUTCPClientConfig: %v", err)
	}
	var rawClientCfg map[string]any
	if err := json.Unmarshal(utcpCfgBytes, &rawClientCfg); err != nil {
		t.Fatalf("Unmarshal .utcp_config.json: %v", err)
	}
	assertOnlyAllowedKeys(t, ".utcp_config.json", rawClientCfg, map[string]bool{"manual_call_templates": true})
	if !strings.Contains(string(utcpCfgBytes), `"name": "xoscal"`) ||
		!strings.Contains(string(utcpCfgBytes), `"url": "http://127.0.0.1:8080/.well-known/utcp"`) ||
		!strings.Contains(string(utcpCfgBytes), `"allowed_communication_protocols"`) {
		t.Errorf("unexpected .utcp_config.json output:\n%s", string(utcpCfgBytes))
	}

	// 2. MCP host config for @utcp/mcp-bridge
	mcpCfgBytes, err := GenerateMCPBridgeConfig("http://127.0.0.1:8080", "/etc/xoscal/.utcp_config.json")
	if err != nil {
		t.Fatalf("GenerateMCPBridgeConfig: %v", err)
	}
	var rawMCPCfg map[string]any
	if err := json.Unmarshal(mcpCfgBytes, &rawMCPCfg); err != nil {
		t.Fatalf("Unmarshal MCPBridgeConfig: %v", err)
	}
	assertOnlyAllowedKeys(t, "MCPBridgeConfig", rawMCPCfg, map[string]bool{"mcpServers": true})
	for _, want := range []string{`"utcp-xoscal"`, `"@utcp/mcp-bridge"`, `"UTCP_CONFIG_FILE": "/etc/xoscal/.utcp_config.json"`} {
		if !strings.Contains(string(mcpCfgBytes), want) {
			t.Errorf("GenerateMCPBridgeConfig missing %s in:\n%s", want, string(mcpCfgBytes))
		}
	}

	// 3. SDK Harnesses across all 5 SDK repositories
	sdkChecks := []struct {
		target string
		wants  []string
	}{
		{
			target: "python-utcp",
			wants: []string{
				"import utcp_http",
				"from utcp.utcp_client import UtcpClient",
				"await UtcpClient.create(\n        config={",
				`await client.call_tool(`,
				`"xoscal.xoscal_controls_search"`,
				`"q": "encryption"`,
				"await client.close()",
			},
		},
		{
			target: "typescript-utcp",
			wants: []string{
				`import "@utcp/http";`,
				`import { UtcpClient } from "@utcp/sdk";`,
				"await UtcpClient.create(process.cwd(), {",
				`await client.callTool("xoscal.xoscal_controls_search"`,
				`q: "encryption"`,
				"await client.close()",
			},
		},
		{
			target: "go-utcp",
			wants: []string{
				`utcp "github.com/universal-tool-calling-protocol/go-utcp"`,
				`&utcp.UtcpClientConfig{ProvidersFilePath: ".utcp_config.json"}`,
				`utcp.NewUTCPClient(ctx, cfg, nil, nil)`,
				`client.CallTool(ctx, "xoscal.xoscal_controls_search"`,
			},
		},
		{
			target: "rs-utcp",
			wants: []string{
				"use std::sync::Arc;",
				"InMemoryToolRepository",
				"TagSearchStrategy",
				`UtcpClientConfig::new().with_providers_file_path(".utcp_config.json".into())`,
				"UtcpClient::create(config, repo, search).await?",
				`"xoscal.xoscal_controls_search"`,
			},
		},
		{
			target: "utcp-mcp",
			wants: []string{
				`"@utcp/mcp-bridge"`,
				`"UTCP_CONFIG_FILE": ".utcp_config.json"`,
			},
		},
	}

	for _, tc := range sdkChecks {
		code, err := GenerateSDKHarness(tc.target, "http://127.0.0.1:8080", ".utcp_config.json")
		if err != nil {
			t.Fatalf("GenerateSDKHarness(%q): %v", tc.target, err)
		}
		for _, w := range tc.wants {
			if !strings.Contains(code, w) {
				t.Errorf("GenerateSDKHarness(%q) missing %q in:\n%s", tc.target, w, code)
			}
		}
	}
}
