package frameworks

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type manifestFrameworkSpec struct {
	ID           string
	Aliases      []string
	Title        string
	Version      string
	Jurisdiction string
	Authority    string
	ControlCount int
}

var manifestFrameworkSpecs = []manifestFrameworkSpec{
	{ID: "nist-csf-2.0", Aliases: []string{"csf-2.0", "nist-csf"}, Title: "NIST CSF version 2.0", Version: "6", Jurisdiction: "usa", Authority: "NIST", ControlCount: 106},
	{ID: "cmmc-2.0", Aliases: []string{"cmmc"}, Title: "CMMC version 2.0", Version: "6", Jurisdiction: "usa", Authority: "DoD", ControlCount: 110},
	{ID: "fedramp-rev5", Aliases: []string{"fedramp"}, Title: "GSA FedRAMP Rev5", Version: "1", Jurisdiction: "usa", Authority: "US General Services Administration", ControlCount: 410},
	{ID: "cisa-cpg-2.0", Aliases: []string{"cisa-cpg"}, Title: "CISA Cybersecurity Performance Goals v2.0", Version: "1", Jurisdiction: "usa", Authority: "CISA", ControlCount: 34},
	{ID: "cisa-vendor-scrm", Title: "VENDOR SUPPLY CHAIN RISK MANAGEMENT (SCRM) TEMPLATE", Version: "2", Jurisdiction: "usa", Authority: "CISA", ControlCount: 256},
	{ID: "cjis-version-5.9.5", Aliases: []string{"cjis"}, Title: "Criminal Justice Information Services (CJIS) Security Policy 5.9.5", Version: "1", Jurisdiction: "usa", Authority: "US CJIS", ControlCount: 1567},
	{ID: "dfs-500-2023-11", Aliases: []string{"ny-dfs-500"}, Title: "NY DFS 500 with 2023-11 amendments", Version: "1", Jurisdiction: "usa", Authority: "NEW YORK STATE", ControlCount: 165},
	{ID: "cyber_essentials", Aliases: []string{"cyber-essentials"}, Title: "Cyber Essentials: Requirements for IT infrastructure v3.1", Version: "2", Jurisdiction: "uk", Authority: "NCSC", ControlCount: 100},
	{ID: "anssi-guide-hygiene", Title: "ANSSI - Guide d'hygiène informatique", Version: "3", Jurisdiction: "france", Authority: "ANSSI", ControlCount: 61},
	{ID: "anssi-guide-hygiene-detail", Title: "ANSSI - Guide d'hygiène informatique - version détaillée", Version: "4", Jurisdiction: "france", Authority: "ANSSI", ControlCount: 200},
	{ID: "anssi-guide-admin-securisee-si", Title: "Recommandations relatives à l'administration sécurisée des systèmes d'information", Version: "1", Jurisdiction: "france", Authority: "ANSSI", ControlCount: 77},
	{ID: "anssi-referentiel-general-de-securite-annexe-b2", Aliases: []string{"rgs-b2"}, Title: "Référentiel Général de Sécurité 2.0 - Annexe B2", Version: "1", Jurisdiction: "france", Authority: "ANSSI", ControlCount: 48},
	{ID: "cnil-guide-securite", Title: "CNIL : guide de sécurité des données", Version: "1", Jurisdiction: "france", Authority: "CNIL", ControlCount: 72},
	{ID: "3cf-v2", Aliases: []string{"3cf"}, Title: "Cadre de Conformité Cyber France (3CF) pour l'aviation civile - v2", Version: "2", Jurisdiction: "france", Authority: "Direction de la sécurité de l'aviation civile", ControlCount: 457},
	{ID: "bs-it-gs-2023-isms", Aliases: []string{"bsi-it-gs-2023-isms"}, Title: "IT-Grundschutz-Kompendium – Werkzeug für Informationssicherheit - ISMS: Sicherheitsmanagement", Version: "1", Jurisdiction: "germany", Authority: "BSI", ControlCount: 84},
	{ID: "bs-it-gs-2023-orp", Aliases: []string{"bsi-it-gs-2023-orp"}, Title: "IT-Grundschutz-Kompendium – Werkzeug für Informationssicherheit - ORP: Organisation und Personal", Version: "1", Jurisdiction: "germany", Authority: "BSI", ControlCount: 229},
	{ID: "bs-it-gs-2023-con", Aliases: []string{"bsi-it-gs-2023-con"}, Title: "IT-Grundschutz-Kompendium – Werkzeug für Informationssicherheit - CON: Konzeption und Vorgehensweisen", Version: "1", Jurisdiction: "germany", Authority: "BSI", ControlCount: 558},
	{ID: "bs-it-gs-2023-sys", Aliases: []string{"bsi-it-gs-2023-sys"}, Title: "IT-Grundschutz-Kompendium – Werkzeug für Informationssicherheit - SYS: IT-Systeme", Version: "1", Jurisdiction: "germany", Authority: "BSI", ControlCount: 1638},
	{ID: "bs-it-gs-2023-app", Aliases: []string{"bsi-it-gs-2023-app"}, Title: "IT-Grundschutz-Kompendium – Werkzeug für Informationssicherheit - APP: Anwendungen", Version: "1", Jurisdiction: "germany", Authority: "BSI", ControlCount: 1222},
	{ID: "bs-it-gs-2023-inf", Aliases: []string{"bsi-it-gs-2023-inf"}, Title: "IT-Grundschutz-Kompendium – Werkzeug für Informationssicherheit - INF: Infrastruktur", Version: "1", Jurisdiction: "germany", Authority: "BSI", ControlCount: 927},
	{ID: "bs-it-gs-2023-net", Aliases: []string{"bsi-it-gs-2023-net"}, Title: "IT-Grundschutz-Kompendium – Werkzeug für Informationssicherheit - NET: Netze und Kommunikation", Version: "1", Jurisdiction: "germany", Authority: "BSI", ControlCount: 892},
	{ID: "bs-it-gs-2023-ops", Aliases: []string{"bsi-it-gs-2023-ops"}, Title: "IT-Grundschutz-Kompendium – Werkzeug für Informationssicherheit - OPS: Betrieb", Version: "1", Jurisdiction: "germany", Authority: "BSI", ControlCount: 1036},
	{ID: "bs-it-gs-2023-ind", Aliases: []string{"bsi-it-gs-2023-ind"}, Title: "IT-Grundschutz-Kompendium – Werkzeug für Informationssicherheit - IND: Industrielle IT", Version: "1", Jurisdiction: "germany", Authority: "BSI", ControlCount: 227},
	{ID: "bs-it-gs-2023-der", Aliases: []string{"bsi-it-gs-2023-der"}, Title: "IT-Grundschutz-Kompendium – Werkzeug für Informationssicherheit - DER: Detektion und Reaktion", Version: "1", Jurisdiction: "germany", Authority: "BSI", ControlCount: 546},
	{ID: "bsi-c5-2020", Aliases: []string{"c5-2020"}, Title: "BSI C5 Library", Version: "1", Jurisdiction: "germany", Authority: "BSI", ControlCount: 121},
	{ID: "bsi-externer-cloud-dienste", Title: "Mindeststandard-des-BSI-zur-Nutzung-externer-Cloud-Dienste (Version 2.1)", Version: "1", Jurisdiction: "germany", Authority: "BSI", ControlCount: 88},
	{ID: "e-its-2024", Title: "E-ITS 2024", Version: "1", Jurisdiction: "germany", Authority: "RIA", ControlCount: 1802},
	{ID: "nis2", Title: "NIS2 - Annex to the Commission Implementing Regulation 2024/2690 with Technical Implementation Guidance by ENISA", Version: "13", Jurisdiction: "eu", Authority: "EUROPEAN COMMISSION + ENISA", ControlCount: 351},
	{ID: "dora", Title: "Digital Operational Resilience Act (DORA)", Version: "7", Jurisdiction: "eu", Authority: "EU", ControlCount: 260},
	{ID: "eu-ai-act", Aliases: []string{"ai-act"}, Title: "EU Artificial Intelligence Act (AI Act)", Version: "2", Jurisdiction: "eu", Authority: "EU", ControlCount: 347},
	{ID: "iso-iec-42001", Aliases: []string{"iso42001-2023", "iso-42001"}, Title: "ISO 42001-2023 — Artificial intelligence  — Requirements and Controls outline", Version: "2", Jurisdiction: "eu", Authority: "ISO", ControlCount: 67},
}

func init() {
	RegisterManifestFrameworks(DefaultRegistry())
}

// RegisterManifestFrameworks registers all 31 non-Canadian manifest frameworks into the provided registry.
func RegisterManifestFrameworks(r *Registry) {
	_, callerFile, _, _ := runtime.Caller(0)
	repoRoot := ""
	if callerFile != "" {
		repoRoot = filepath.Clean(filepath.Join(filepath.Dir(callerFile), "..", "..", ".."))
	}

	for _, spec := range manifestFrameworkSpecs {
		s := spec
		var mu sync.Mutex
		var cached []byte

		loadCatalog := func() ([]byte, error) {
			mu.Lock()
			defer mu.Unlock()
			if len(cached) > 0 {
				return cached, nil
			}

			var candidates []string
			if envDir := os.Getenv("XOSCAL_DATA_DIR"); envDir != "" {
				candidates = append(candidates,
					filepath.Join(envDir, s.ID, "catalog.json"),
					filepath.Join(envDir, "frameworks", s.ID, "catalog.json"),
				)
			}
			candidates = append(candidates,
				filepath.Join("data", "frameworks", s.ID, "catalog.json"),
				filepath.Join("/data", "frameworks", s.ID, "catalog.json"),
				filepath.Join("/usr/share/xoscal/data/frameworks", s.ID, "catalog.json"),
			)
			if exePath, err := os.Executable(); err == nil && exePath != "" {
				exeDir := filepath.Dir(exePath)
				candidates = append(candidates,
					filepath.Join(exeDir, "data", "frameworks", s.ID, "catalog.json"),
					filepath.Join(exeDir, "..", "data", "frameworks", s.ID, "catalog.json"),
					filepath.Join(exeDir, "..", "share", "xoscal", "data", "frameworks", s.ID, "catalog.json"),
				)
			}
			if repoRoot != "" {
				candidates = append(candidates, filepath.Join(repoRoot, "data", "frameworks", s.ID, "catalog.json"))
			}

			for _, p := range candidates {
				// #nosec G304 -- path is derived from the static compiled manifestFrameworkSpecs table.
				if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
					cached = data
					return cached, nil
				}
			}
			return nil, fmt.Errorf("catalog.json not found for framework %s", s.ID)
		}

		_ = r.Register(New(Config{
			ID:           s.ID,
			Aliases:      s.Aliases,
			Title:        s.Title,
			Version:      s.Version,
			Jurisdiction: s.Jurisdiction,
			Authority:    s.Authority,
			ControlCount: s.ControlCount,
			HasProfile:   false,
			CatalogFn:    loadCatalog,
		}))
	}
}
