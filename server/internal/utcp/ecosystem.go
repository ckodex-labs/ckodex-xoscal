package utcp

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RepositoryDescriptor describes one of the 7 official Universal Tool Calling Protocol repositories.
type RepositoryDescriptor struct {
	Name        string `json:"name"`
	Language    string `json:"language"`
	URL         string `json:"url"`
	Package     string `json:"package"`
	Role        string `json:"role"`
	Description string `json:"description"`
}

// SupportedRepositories lists the 7 official UTCP ecosystem repositories supported by xOSCAL.
var SupportedRepositories = []RepositoryDescriptor{
	{
		Name:        "universal-tool-calling-protocol",
		Language:    "Multi-Language",
		URL:         "https://github.com/universal-tool-calling-protocol",
		Package:     "utcp",
		Role:        "Organization & Governance Root",
		Description: "Official Universal Tool Calling Protocol organization for direct, zero-wrapper tool discovery and invocation.",
	},
	{
		Name:        "utcp-specification",
		Language:    "JSON Schema / OpenAPI",
		URL:         "https://github.com/universal-tool-calling-protocol/utcp-specification",
		Package:     "utcp-spec@1.0.1",
		Role:        "Protocol Specification",
		Description: "Normative UTCP v1.0.1 Manual, Tool, Auth, and CallTemplate schema definitions.",
	},
	{
		Name:        "utcp-mcp",
		Language:    "TypeScript / MCP Bridge",
		URL:         "https://github.com/universal-tool-calling-protocol/utcp-mcp",
		Package:     "@utcp/mcp-bridge",
		Role:        "UTCP-to-MCP Bridge Server",
		Description: "Universal MCP bridge exposing UTCP manuals and native call templates directly to MCP clients.",
	},
	{
		Name:        "rs-utcp",
		Language:    "Rust",
		URL:         "https://github.com/universal-tool-calling-protocol/rs-utcp",
		Package:     "rs-utcp",
		Role:        "Rust Async Tokio SDK",
		Description: "High-assurance Rust UTCP client & provider runtime with Tokio async transports and semantic tag search.",
	},
	{
		Name:        "typescript-utcp",
		Language:    "TypeScript / Node.js",
		URL:         "https://github.com/universal-tool-calling-protocol/typescript-utcp",
		Package:     "@utcp/sdk",
		Role:        "TypeScript Monorepo SDK",
		Description: "Official TypeScript/JavaScript UTCP client and pluggable protocol modules backed by Zod validation.",
	},
	{
		Name:        "go-utcp",
		Language:    "Go",
		URL:         "https://github.com/universal-tool-calling-protocol/go-utcp",
		Package:     "github.com/universal-tool-calling-protocol/go-utcp",
		Role:        "Go Native SDK",
		Description: "Official Go UTCP client and provider framework for heterogeneous backend tool discovery and invocation.",
	},
	{
		Name:        "python-utcp",
		Language:    "Python",
		URL:         "https://github.com/universal-tool-calling-protocol/python-utcp",
		Package:     "utcp",
		Role:        "Python Pydantic SDK",
		Description: "Official Python UTCP SDK with Pydantic v2 models (UtcpClient, UtcpManual, Tool, CallTemplate).",
	},
}

// UTCPClientConfig represents a strict UTCP v1.0.1 .utcp_config.json file consumed by
// @utcp/mcp-bridge, @utcp/sdk, python-utcp, go-utcp, and rs-utcp.
type UTCPClientConfig struct {
	ManualCallTemplates []CallTemplate `json:"manual_call_templates"`
}

// MCPBridgeConfig represents a standard MCP host configuration invoking @utcp/mcp-bridge.
type MCPBridgeConfig struct {
	MCPServers map[string]MCPServer `json:"mcpServers"`
}

// MCPServer represents a standard MCP server entry invoking @utcp/mcp-bridge.
type MCPServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

func resolveManualDiscoveryURL(baseOrManualURL string) string {
	u := strings.TrimSpace(baseOrManualURL)
	if u == "" {
		return "http://127.0.0.1:8080/.well-known/utcp"
	}
	u = strings.TrimRight(u, "/")
	if strings.HasSuffix(u, "/.well-known/utcp") ||
		strings.HasSuffix(u, "/api/v1/utcp/manual") ||
		strings.HasSuffix(u, "/api/v1/utcp") ||
		strings.HasSuffix(u, ".json") {
		return u
	}
	return u + "/.well-known/utcp"
}

// GenerateUTCPClientConfig produces a pure UTCP v1.0.1 .utcp_config.json document
// registering the xOSCAL HTTP manual discovery endpoint under manual_call_templates.
func GenerateUTCPClientConfig(baseURL string) ([]byte, error) {
	manualURL := resolveManualDiscoveryURL(baseURL)
	cfg := UTCPClientConfig{
		ManualCallTemplates: []CallTemplate{
			{
				Name:                          "xoscal",
				CallTemplateType:              "http",
				URL:                           manualURL,
				HTTPMethod:                    "GET",
				AllowedCommunicationProtocols: []string{"http"},
			},
		},
	}
	return json.MarshalIndent(cfg, "", "  ")
}

// BuildDiscoveryConfig produces the UTCP discovery .utcp_config.json configuration.
func BuildDiscoveryConfig(manualURL, _ string) ([]byte, error) {
	return GenerateUTCPClientConfig(manualURL)
}

// GenerateMCPBridgeConfig produces an MCP host configuration invoking @utcp/mcp-bridge
// pointed at utcpConfigPath (defaulting to ".utcp_config.json" when empty or set to the
// legacy "./utcp.json" manual flag default).
func GenerateMCPBridgeConfig(baseURL, utcpConfigPath string) ([]byte, error) {
	_ = resolveManualDiscoveryURL(baseURL)
	cfgPath := strings.TrimSpace(utcpConfigPath)
	if cfgPath == "" || cfgPath == "./utcp.json" {
		cfgPath = ".utcp_config.json"
	}

	cfg := MCPBridgeConfig{
		MCPServers: map[string]MCPServer{
			"utcp-xoscal": {
				Command: "npx",
				Args:    []string{"-y", "@utcp/mcp-bridge"},
				Env: map[string]string{
					"UTCP_CONFIG_FILE": cfgPath,
				},
			},
		},
	}

	return json.MarshalIndent(cfg, "", "  ")
}

// GenerateSDKHarness generates an idiomatic integration harness for one of the official
// UTCP language SDKs: "go" (go-utcp), "python" (python-utcp), "typescript" (typescript-utcp),
// "rust" (rs-utcp), or "mcp" (utcp-mcp).
func GenerateSDKHarness(target, manualURL, manualFilePath string) (string, error) {
	resolvedManualURL := resolveManualDiscoveryURL(manualURL)

	switch strings.ToLower(strings.TrimSpace(target)) {
	case "go", "go-utcp":
		return fmt.Sprintf(`// Generated by xOSCAL for https://github.com/universal-tool-calling-protocol/go-utcp
// Expects .utcp_config.json pointing to %s
package main

import (
	"context"
	"fmt"
	"log"

	utcp "github.com/universal-tool-calling-protocol/go-utcp"
)

func main() {
	ctx := context.Background()
	cfg := &utcp.UtcpClientConfig{ProvidersFilePath: ".utcp_config.json"}

	client, err := utcp.NewUTCPClient(ctx, cfg, nil, nil)
	if err != nil {
		log.Fatalf("init go-utcp client: %%v", err)
	}

	out, err := client.CallTool(ctx, "xoscal.xoscal_controls_search", map[string]any{
		"q":         "encryption",
		"framework": "cccs-itsp-10-033",
	})
	if err != nil {
		log.Fatalf("call xoscal.xoscal_controls_search: %%v", err)
	}
	fmt.Printf("xOSCAL UTCP result: %%v\n", out)
}
`, resolvedManualURL), nil

	case "python", "py", "python-utcp":
		return fmt.Sprintf(`# Generated by xOSCAL for https://github.com/universal-tool-calling-protocol/python-utcp
import asyncio
import utcp_http  # registers HTTP call template plugin
from utcp.utcp_client import UtcpClient

async def main() -> None:
    client = await UtcpClient.create(
        config={
            "manual_call_templates": [
                {
                    "name": "xoscal",
                    "call_template_type": "http",
                    "url": %q,
                    "http_method": "GET",
                    "allowed_communication_protocols": ["http"],
                }
            ]
        }
    )
    try:
        result = await client.call_tool(
            "xoscal.xoscal_controls_search",
            {"q": "encryption", "framework": "cccs-itsp-10-033"},
        )
        print(result)
    finally:
        await client.close()

if __name__ == "__main__":
    asyncio.run(main())
`, resolvedManualURL), nil

	case "ts", "typescript", "javascript", "js", "typescript-utcp":
		return fmt.Sprintf(`// Generated by xOSCAL for https://github.com/universal-tool-calling-protocol/typescript-utcp
import "@utcp/http";
import { UtcpClient } from "@utcp/sdk";

async function main() {
  const client = await UtcpClient.create(process.cwd(), {
    manual_call_templates: [
      {
        name: "xoscal",
        call_template_type: "http",
        url: %q,
        http_method: "GET",
        allowed_communication_protocols: ["http"],
      },
    ],
  });
  try {
    const result = await client.callTool("xoscal.xoscal_controls_search", {
      q: "encryption",
      framework: "cccs-itsp-10-033",
    });
    console.log("xOSCAL UTCP result:", result);
  } finally {
    await client.close();
  }
}

main().catch(console.error);
`, resolvedManualURL), nil

	case "rust", "rs", "rs-utcp":
		return fmt.Sprintf(`// Generated by xOSCAL for https://github.com/universal-tool-calling-protocol/rs-utcp
// Expects .utcp_config.json pointing to %s
use std::sync::Arc;
use serde_json::json;
use rs_utcp::{
    UtcpClient, UtcpClientInterface,
    config::UtcpClientConfig,
    repository::in_memory::InMemoryToolRepository,
    tag::tag_search::TagSearchStrategy,
};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let config = UtcpClientConfig::new().with_providers_file_path(".utcp_config.json".into());
    let repo = Arc::new(InMemoryToolRepository::new());
    let search = Arc::new(TagSearchStrategy::new(repo.clone(), 1.0));
    let client = UtcpClient::create(config, repo, search).await?;

    let result = client
        .call_tool(
            "xoscal.xoscal_controls_search",
            json!({
                "q": "encryption",
                "framework": "cccs-itsp-10-033"
            }),
        )
        .await?;

    println!("xOSCAL UTCP result: {}", result);
    Ok(())
}
`, resolvedManualURL), nil

	case "mcp", "utcp-mcp":
		b, err := GenerateMCPBridgeConfig(resolvedManualURL, manualFilePath)
		if err != nil {
			return "", err
		}
		return string(b), nil

	default:
		return "", fmt.Errorf("unsupported UTCP SDK target %q; supported targets: go-utcp, python-utcp, typescript-utcp, rs-utcp, utcp-mcp", target)
	}
}
