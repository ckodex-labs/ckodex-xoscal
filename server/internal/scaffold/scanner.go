package scaffold

import (
	"encoding/json"
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
	Components  []ComponentCandidate
}

// ScanRepository inspects the given directory and identifies languages, frameworks,
// infrastructure definitions, and components.
func ScanRepository(dir string) (*ProjectScan, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	scan := &ProjectScan{
		Dir:     absDir,
		Name:    filepath.Base(absDir),
		Version: "1.0.0",
	}

	// 1. Check for Go
	if data, err := os.ReadFile(filepath.Join(absDir, "go.mod")); err == nil {
		scan.Languages = append(scan.Languages, "Go")
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "module ") {
				modName := strings.TrimSpace(strings.TrimPrefix(line, "module"))
				parts := strings.Split(modName, "/")
				scan.Name = parts[len(parts)-1]
				break
			}
		}
	}

	// 2. Check for Node / TS
	if data, err := os.ReadFile(filepath.Join(absDir, "package.json")); err == nil {
		scan.Languages = append(scan.Languages, "TypeScript/JavaScript")
		var pkg struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description"`
		}
		if json.Unmarshal(data, &pkg) == nil {
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
	}

	// 3. Check for Python
	if fileExists(filepath.Join(absDir, "requirements.txt")) ||
		fileExists(filepath.Join(absDir, "pyproject.toml")) ||
		fileExists(filepath.Join(absDir, "Pipfile")) {
		scan.Languages = append(scan.Languages, "Python")
	}

	// 4. Check for Rust
	if fileExists(filepath.Join(absDir, "Cargo.toml")) {
		scan.Languages = append(scan.Languages, "Rust")
	}

	// 5. Check for Containerization
	if fileExists(filepath.Join(absDir, "Dockerfile")) ||
		fileExists(filepath.Join(absDir, "Containerfile")) ||
		fileExists(filepath.Join(absDir, "docker-compose.yml")) {
		scan.HasDocker = true
	}

	// 6. Check for Kubernetes
	if dirExists(filepath.Join(absDir, "k8s")) ||
		dirExists(filepath.Join(absDir, "manifests")) ||
		dirExists(filepath.Join(absDir, "charts")) {
		scan.HasK8s = true
	}

	// 7. Check for Terraform / IaC
	if dirExists(filepath.Join(absDir, "terraform")) {
		scan.HasTF = true
	}
	_ = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".tf") {
			scan.HasTF = true
		}
		return nil
	})

	if scan.Description == "" {
		scan.Description = scan.Name + " primary service and operational substrate"
	}

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

	return scan, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
