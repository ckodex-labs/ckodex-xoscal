package scaffold

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mchorfa/xoscal/server/internal/rootfs"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ComponentCandidate represents a discovered or inferred software/infra component.
type ComponentCandidate struct {
	Name        string
	Type        string // "software", "service", "hardware", "interconnection"
	Title       string
	Description string
	Controls    []SuggestedControl
}

// SuggestedControl represents a baseline compliance control to seed for this component.
type SuggestedControl struct {
	ControlID   string
	Description string
}

// ProjectScan contains the results of inspecting a project repository.
type ProjectScan struct {
	Dir         string
	Name        string
	Version     string
	Description string
	Languages   []string
	HasDocker   bool
	HasK8s      bool
	HasTF       bool
	HasUTCP     bool
	UTCPSDKs    []string
	Components  []ComponentCandidate
}

// ScanRepository inspects the given directory and identifies languages, frameworks,
// infrastructure definitions, and components.
func ScanRepository(dir string) (result *ProjectScan, resultErr error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	return scanRepository(root, dir)
}

func scanRepository(root *os.Root, dir string) (*ProjectScan, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	scan := &ProjectScan{Dir: absDir, Name: filepath.Base(absDir), Version: "1.0.0"}
	if err := scanMetadata(root, scan); err != nil {
		return nil, err
	}
	if err := scanMarkers(root, scan); err != nil {
		return nil, err
	}
	if err := fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := root.Stat(name)
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() && strings.HasSuffix(name, ".tf") {
			scan.HasTF = true
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("inspect repository tree: %w", err)
	}
	if scan.Description == "" {
		scan.Description = scan.Name + " primary service and operational substrate"
	}
	addDefaultComponents(scan)
	return scan, nil
}

func scanMetadata(root *os.Root, scan *ProjectScan) error {
	data, err := rootfs.OptionalReadFile(root, "go.mod")
	if err != nil {
		return fmt.Errorf("read go.mod: %w", err)
	}
	if data != nil {
		scan.Languages = append(scan.Languages, "Go")
		raw := string(data)
		for _, line := range strings.Split(raw, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "module ") {
				parts := strings.Split(strings.TrimSpace(strings.TrimPrefix(line, "module")), "/")
				scan.Name = parts[len(parts)-1]
				break
			}
		}
		if strings.Contains(raw, "universal-tool-calling-protocol/go-utcp") || strings.Contains(raw, "go-utcp") {
			scan.HasUTCP = true
			scan.UTCPSDKs = append(scan.UTCPSDKs, "go-utcp")
		}
	}
	data, err = rootfs.OptionalReadFile(root, "package.json")
	if err != nil {
		return fmt.Errorf("read package.json: %w", err)
	}
	if data != nil {
		scan.Languages = append(scan.Languages, "TypeScript/JavaScript")
		applyPackageMetadata(data, scan)
		raw := string(data)
		if strings.Contains(raw, "@utcp/sdk") || strings.Contains(raw, "typescript-utcp") {
			scan.HasUTCP = true
			scan.UTCPSDKs = append(scan.UTCPSDKs, "typescript-utcp")
		}
		if strings.Contains(raw, "@utcp/mcp") || strings.Contains(raw, "utcp-mcp") {
			scan.HasUTCP = true
			scan.UTCPSDKs = append(scan.UTCPSDKs, "utcp-mcp")
		}
	}
	for _, pyFile := range []string{"requirements.txt", "pyproject.toml", "Pipfile"} {
		if pyData, rErr := rootfs.OptionalReadFile(root, pyFile); rErr == nil && pyData != nil {
			raw := string(pyData)
			if strings.Contains(raw, "python-utcp") || strings.Contains(raw, "utcp-http") || strings.Contains(raw, "utcp-cli") || strings.Contains(raw, "\"utcp") {
				scan.HasUTCP = true
				scan.UTCPSDKs = append(scan.UTCPSDKs, "python-utcp")
				break
			}
		}
	}
	if cargoData, rErr := rootfs.OptionalReadFile(root, "Cargo.toml"); rErr == nil && cargoData != nil {
		raw := string(cargoData)
		if strings.Contains(raw, "rs-utcp") || strings.Contains(raw, "universal-tool-calling-protocol/rs-utcp") {
			scan.HasUTCP = true
			scan.UTCPSDKs = append(scan.UTCPSDKs, "rs-utcp")
		}
	}
	return nil
}

func applyPackageMetadata(data []byte, scan *ProjectScan) {
	var pkg struct{ Name, Version, Description string }
	if json.Unmarshal(data, &pkg) != nil {
		return
	}
	if pkg.Name != "" {
		scan.Name = pkg.Name
	}
	if pkg.Version != "" {
		scan.Version = pkg.Version
	}
	if pkg.Description != "" {
		scan.Description = pkg.Description
	}
}

func scanMarkers(root *os.Root, scan *ProjectScan) error {
	markers := []struct {
		names     []string
		directory bool
		language  string
		detected  *bool
	}{
		{names: []string{"requirements.txt", "pyproject.toml", "Pipfile"}, language: "Python"},
		{names: []string{"Cargo.toml"}, language: "Rust"},
		{names: []string{"Dockerfile", "Containerfile", "docker-compose.yml"}, detected: &scan.HasDocker},
		{names: []string{"k8s", "manifests", "charts"}, directory: true, detected: &scan.HasK8s},
		{names: []string{"terraform"}, directory: true, detected: &scan.HasTF},
		{names: []string{"utcp.json", ".utcp.json", ".utcp_config.json"}, detected: &scan.HasUTCP},
	}
	for _, marker := range markers {
		found := false
		for _, name := range marker.names {
			info, err := rootfs.OptionalStat(root, name)
			if err != nil {
				return fmt.Errorf("inspect %s: %w", name, err)
			}
			if info != nil && info.IsDir() == marker.directory {
				found = true
			}
		}
		if found && marker.language != "" {
			scan.Languages = append(scan.Languages, marker.language)
		}
		if marker.detected != nil && found {
			*marker.detected = true
		}
	}
	return nil
}

func addDefaultComponents(scan *ProjectScan) {
	// Build default components
	mainComp := ComponentCandidate{
		Name:        scan.Name,
		Type:        "software",
		Title:       scan.Name + " Core Engine",
		Description: scan.Description,
		Controls: []SuggestedControl{
			{ControlID: "ac-2", Description: "Account management implemented via service authentication and credential checks."},
			{ControlID: "ac-3", Description: "Access enforcement enforced through least-privilege capability boundaries."},
			{ControlID: "sc-8", Description: "Transmission confidentiality and integrity enforced via TLS/mTLS on external endpoints."},
			{ControlID: "sc-13", Description: "Cryptographic protection implemented using standard SHA-256 and Ed25519 primitives."},
			{ControlID: "si-2", Description: "Flaw remediation verified via continuous dependency audits and static code analysis."},
		},
	}
	scan.Components = append(scan.Components, mainComp)

	if scan.HasDocker || scan.HasK8s {
		infraComp := ComponentCandidate{
			Name:        scan.Name + "-deployment",
			Type:        "service",
			Title:       scan.Name + " Deployment Workload",
			Description: "Containerized runtime environment and orchestrator manifests",
			Controls: []SuggestedControl{
				{ControlID: "cm-8", Description: "Information system component inventory tracked via container image digests."},
				{ControlID: "sc-28", Description: "Protection of information at rest within mounted volumes and ephemeral storage."},
			},
		}
		scan.Components = append(scan.Components, infraComp)
	}

	if scan.HasUTCP {
		sdkNote := "utcp-specification"
		if len(scan.UTCPSDKs) > 0 {
			sdkNote = strings.Join(scan.UTCPSDKs, ", ")
		}
		utcpComp := ComponentCandidate{
			Name:        scan.Name + "-utcp-gateway",
			Type:        "interconnection",
			Title:       scan.Name + " Universal Tool Calling Protocol (UTCP) Surface",
			Description: fmt.Sprintf("Direct native UTCP v1.0.1 tool manual and call templates (%s)", sdkNote),
			Controls: []SuggestedControl{
				{ControlID: "ac-3", Description: "Least-authority capability enforcement across UTCP tool call templates."},
				{ControlID: "ca-3", Description: "System interconnections governed via explicit UTCP v1.0.1 manual schemas."},
				{ControlID: "sc-7", Description: "Boundary protection enforced across HTTP, gRPC, and CLI UTCP transports (WebSockets forbidden per Rule 10)."},
				{ControlID: "si-10", Description: "Information input validation enforced via strict JSON Schema inputs/outputs on every UTCP tool."},
			},
		}
		scan.Components = append(scan.Components, utcpComp)
	}
}
