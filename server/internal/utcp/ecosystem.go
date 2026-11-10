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
		Package:     "@utcp/mcp",
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

// MCPBridgeConfig represents the configuration bundle for universal-tool-calling-protocol/utcp-mcp.
type MCPBridgeConfig struct {
	Schema              string               `json:"$schema,omitempty"`
	BridgeRepository    string               `json:"bridge_repository"`
	SpecificationURL    string               `json:"specification_url"`
	ManualCallTemplates []CallTemplate       `json:"manual_call_templates"`
	MCPServers          map[string]MCPServer `json:"mcpServers"`
}

// MCPServer represents a standard MCP server entry invoking @utcp/mcp (utcp-mcp).
type MCPServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// BuildDiscoveryConfig is an alias for GenerateMCPBridgeConfig returning the UTCP discovery and MCP bridge configuration.
func BuildDiscoveryConfig(manualURL, manualFilePath string) ([]byte, error) {
	return GenerateMCPBridgeConfig(manualURL, manualFilePath)
}

// GenerateMCPBridgeConfig produces a ready-to-use configuration for
// https://github.com/universal-tool-calling-protocol/utcp-mcp so MCP clients
// can invoke xOSCAL's native UTCP manual without custom wrapper code.
func GenerateMCPBridgeConfig(manualURL, manualFilePath string) ([]byte, error) {
	if manualURL == "" {
		manualURL = "http://127.0.0.1:8080/.well-known/utcp"
	}
	if manualFilePath == "" {
		manualFilePath = "./utcp.json"
	}

	allowed := DefaultAllowedCommunicationProtocols()
	cfg := MCPBridgeConfig{
		BridgeRepository: "https://github.com/universal-tool-calling-protocol/utcp-mcp",
		SpecificationURL: SpecRepositoryURL,
		ManualCallTemplates: []CallTemplate{
			{
				Name:                          "xoscal_local_manual",
				CallTemplateType:              "text",
				FilePath:                      manualFilePath,
				AllowedCommunicationProtocols: allowed,
			},
			{
				Name:                          "xoscal_http_manual",
				CallTemplateType:              "http",
				URL:                           manualURL,
				HTTPMethod:                    "GET",
				Headers:                       map[string]string{"Accept": "application/json"},
				AllowedCommunicationProtocols: allowed,
			},
		},
		MCPServers: map[string]MCPServer{
			"xoscal-utcp": {
				Command: "npx",
				Args:    []string{"-y", "@utcp/mcp", "--config", ".utcp_config.json"},
				Env: map[string]string{
					"UTCP_MANUAL_PATH": manualFilePath,
					"UTCP_MANUAL_URL":  manualURL,
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
	if manualURL == "" {
		manualURL = "http://127.0.0.1:8080/.well-known/utcp"
	}
	if manualFilePath == "" {
		manualFilePath = "./utcp.json"
	}

	switch strings.ToLower(strings.TrimSpace(target)) {
	case "go", "go-utcp":
		return fmt.Sprintf(`// Generated by xOSCAL for https://github.com/universal-tool-calling-protocol/go-utcp
package main

import (
	"context"
	"fmt"
	"log"

	utcp "github.com/universal-tool-calling-protocol/go-utcp"
)

func main() {
	ctx := context.Background()
	cfg := &utcp.UtcpClientConfig{
		ManualCallTemplates: []utcp.CallTemplate{
			{
				Name:                          "xoscal_http_manual",
				CallTemplateType:              "http",
				URL:                           %q,
				HTTPMethod:                    "GET",
				AllowedCommunicationProtocols: []string{"cli", "http", "grpc", "sse", "text", "mcp"},
			},
			{
				Name:                          "xoscal_local_manual",
				CallTemplateType:              "text",
				FilePath:                      %q,
				AllowedCommunicationProtocols: []string{"cli", "http", "grpc", "sse", "text", "mcp"},
			},
		},
	}

	client, err := utcp.NewUtcpClient(ctx, cfg, nil, nil)
	if err != nil {
		log.Fatalf("init go-utcp client: %%v", err)
	}

	out, err := client.CallTool(ctx, "xoscal_http_manual.xoscal_controls_search", map[string]any{
		"framework": "cccs-itsp-10-033",
		"query":     "account management",
	})
	if err != nil {
		log.Fatalf("call xoscal_http_manual.xoscal_controls_search: %%v", err)
	}
	fmt.Printf("xOSCAL UTCP result: %%v\n", out)
}
`, manualURL, manualFilePath), nil

	case "python", "py", "python-utcp":
		return fmt.Sprintf(`# Generated by xOSCAL for https://github.com/universal-tool-calling-protocol/python-utcp
import asyncio
from utcp.utcp_client import UtcpClient
from utcp.data.utcp_client_config import UtcpClientConfig

async def main() -> None:
    config = UtcpClientConfig(
        manual_call_templates=[
            {
                "name": "xoscal_http_manual",
                "call_template_type": "http",
                "url": %q,
                "http_method": "GET",
                "allowed_communication_protocols": ["cli", "http", "grpc", "sse", "text", "mcp"],
            },
            {
                "name": "xoscal_local_manual",
                "call_template_type": "text",
                "file_path": %q,
                "allowed_communication_protocols": ["cli", "http", "grpc", "sse", "text", "mcp"],
            },
        ]
    )
    client = await UtcpClient.create(config=config)
    result = await client.call_tool(
        "xoscal_http_manual.xoscal_controls_search",
        {"framework": "cccs-itsp-10-033", "query": "account management"},
    )
    print(result)

if __name__ == "__main__":
    asyncio.run(main())
`, manualURL, manualFilePath), nil

	case "ts", "typescript", "javascript", "js", "typescript-utcp":
		return fmt.Sprintf(`// Generated by xOSCAL for https://github.com/universal-tool-calling-protocol/typescript-utcp
import { UtcpClient } from "@utcp/sdk";

async function main() {
  const client = await UtcpClient.create(process.cwd(), {
    manual_call_templates: [
      {
        name: "xoscal_http_manual",
        call_template_type: "http",
        url: %q,
        http_method: "GET",
        allowed_communication_protocols: ["cli", "http", "grpc", "sse", "text", "mcp"],
      },
      {
        name: "xoscal_local_manual",
        call_template_type: "text",
        file_path: %q,
        allowed_communication_protocols: ["cli", "http", "grpc", "sse", "text", "mcp"],
      },
    ],
  });

  const result = await client.callTool("xoscal_http_manual.xoscal_controls_search", {
    framework: "cccs-itsp-10-033",
    query: "account management",
  });
  console.log("xOSCAL UTCP result:", result);
}

main().catch(console.error);
`, manualURL, manualFilePath), nil

	case "rust", "rs", "rs-utcp":
		return fmt.Sprintf(`// Generated by xOSCAL for https://github.com/universal-tool-calling-protocol/rs-utcp
use rs_utcp::{UtcpClient, UtcpClientConfig, CallTemplate};
use serde_json::json;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let config = UtcpClientConfig::builder()
        .add_manual_template(CallTemplate::http("xoscal_http_manual", %q, "GET"))
        .add_manual_template(CallTemplate::text("xoscal_local_manual", %q))
        .build();

    let client = UtcpClient::new(config, None, None).await?;
    let result = client
        .call_tool(
            "xoscal_http_manual.xoscal_controls_search",
            json!({
                "framework": "cccs-itsp-10-033",
                "query": "account management"
            }),
        )
        .await?;

    println!("xOSCAL UTCP result: {}", result);
    Ok(())
}
`, manualURL, manualFilePath), nil

	case "mcp", "utcp-mcp":
		b, err := GenerateMCPBridgeConfig(manualURL, manualFilePath)
		if err != nil {
			return "", err
		}
		return string(b), nil

	default:
		return "", fmt.Errorf("unsupported UTCP SDK target %q; supported targets: go-utcp, python-utcp, typescript-utcp, rs-utcp, utcp-mcp", target)
	}
}
