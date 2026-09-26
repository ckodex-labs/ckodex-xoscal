package canadianframeworks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

// SeedKG seeds all authoritative Canadian cybersecurity frameworks and their
// individual control requirements directly into a Knowledge Graph store.
// The operation is idempotent: existing entities are updated in place.
func SeedKG(ctx context.Context, store kg.Store) error {
	now := time.Now().UTC()

	frameworks := []struct {
		info    FrameworkInfo
		catalog *catalogv1.Catalog
	}{
		{
			info: FrameworkInfo{
				RefID:        "cccs-itsg-33",
				Name:         "CCCS ITSG-33 IT Security Risk Management: A Lifecycle Approach (Annex 3)",
				Version:      "1.2.3",
				Locale:       "ca",
				Provider:     "Canadian Centre for Cyber Security / Communications Security Establishment",
				ControlCount: 18,
				HasProfile:   false,
			},
			catalog: BuildCCCSITSG33Catalog(),
		},
		{
			info: FrameworkInfo{
				RefID:        "cybersecure-canada",
				Name:         "CyberSecure Canada (CAN/DGSI 104)",
				Version:      "CAN/DGSI 104",
				Locale:       "ca",
				Provider:     "Innovation, Science and Economic Development Canada (ISED) / CCCS",
				ControlCount: 46,
				HasProfile:   false,
			},
			catalog: BuildCyberSecureCanadaCatalog(),
		},
		{
			info: FrameworkInfo{
				RefID:        "cccs-itsp-10-171",
				Name:         "CCCS ITSP.10.171 Security Requirements for Contracting with the Government of Canada",
				Version:      "1.2.3",
				Locale:       "ca",
				Provider:     "Canadian Centre for Cyber Security / Communications Security Establishment",
				ControlCount: 267,
				HasProfile:   false,
			},
			catalog: BuildCCCSITSP10171Catalog(),
		},
		{
			info: FrameworkInfo{
				RefID:        "cccs-medium-cloud-pbmm",
				Name:         "CCCS Medium Cloud PBMM Baseline (ITSP.50.103 / ITSG-33 Annex 4A Profile 1)",
				Version:      "ITSP.50.103",
				Locale:       "ca",
				Provider:     "Canadian Centre for Cyber Security / Communications Security Establishment",
				ControlCount: 30,
				HasProfile:   true,
			},
			catalog: BuildCCCSMediumCloudPBMMCatalog(),
		},
	}

	for _, fwEntry := range frameworks {
		fw := kg.Framework{
			URN:      fmt.Sprintf("urn:xoscal:framework:%s", fwEntry.info.RefID),
			Type:     "Framework",
			RefID:    fwEntry.info.RefID,
			Name:     fwEntry.info.Name,
			Version:  fwEntry.info.Version,
			Locale:   fwEntry.info.Locale,
			Provider: fwEntry.info.Provider,
		}
		fwPayload, err := json.Marshal(fw)
		if err != nil {
			return fmt.Errorf("marshal framework %s: %w", fwEntry.info.RefID, err)
		}

		fwEntity := &kg.Entity{
			URN:       fw.URN,
			Type:      "reg:Framework",
			Version:   1,
			Status:    kg.EntityStatusActive,
			ValidFrom: now,
			Payload:   fwPayload,
		}

		if err := upsertEntity(ctx, store, fwEntity); err != nil {
			return fmt.Errorf("upsert framework %s: %w", fwEntry.info.RefID, err)
		}

		// Track seeded groups to create group hierarchy
		seededGroups := make(map[string]bool)

		// Traverse controls from catalog root and groups
		controls := extractAllControls(fwEntry.catalog)
		for _, item := range controls {
			ctrl := item.ctrl
			ctrlID := ""
			if ctrl.Id != nil {
				ctrlID = ctrl.Id.Value
			}
			if ctrlID == "" {
				continue
			}

			var parentURN string
			if item.section != "" {
				groupURN := fmt.Sprintf("urn:xoscal:requirement:%s:group:%s", fwEntry.info.RefID, item.section)
				parentURN = groupURN

				if !seededGroups[item.section] {
					groupReq := kg.Requirement{
						URN:        groupURN,
						Type:       "Requirement",
						Framework:  fwEntry.info.RefID,
						NodeRefID:  item.section,
						NodeName:   strings.ToUpper(item.section),
						Citation:   item.section,
						Assessable: false,
					}
					gPayload, err := json.Marshal(groupReq)
					if err != nil {
						return fmt.Errorf("marshal group %s: %w", groupURN, err)
					}
					gEntity := &kg.Entity{
						URN:       groupURN,
						Type:      "reg:Requirement",
						Version:   1,
						Status:    kg.EntityStatusActive,
						ValidFrom: now,
						Payload:   gPayload,
					}
					if err := upsertEntity(ctx, store, gEntity); err != nil {
						return fmt.Errorf("upsert group %s: %w", groupURN, err)
					}
					seededGroups[item.section] = true
				}
			}

			title := ""
			if ctrl.Title != nil {
				title = ctrl.Title.Value
			}

			statement := extractProse(ctrl)

			req := kg.Requirement{
				URN:        fmt.Sprintf("urn:xoscal:requirement:%s:%s", fwEntry.info.RefID, ctrlID),
				Type:       "Requirement",
				Framework:  fwEntry.info.RefID,
				Citation:   ctrlID,
				Title:      title,
				Text:       statement,
				Section:    item.section,
				ParentURN:  parentURN,
				Assessable: true,
			}

			reqPayload, err := json.Marshal(req)
			if err != nil {
				return fmt.Errorf("marshal requirement %s: %w", req.URN, err)
			}

			reqEntity := &kg.Entity{
				URN:       req.URN,
				Type:      "reg:Requirement",
				Version:   1,
				Status:    kg.EntityStatusActive,
				ValidFrom: now,
				Payload:   reqPayload,
			}

			if err := upsertEntity(ctx, store, reqEntity); err != nil {
				return fmt.Errorf("upsert requirement %s: %w", req.URN, err)
			}
		}
	}

	return nil
}

type controlWithContext struct {
	ctrl    *catalogv1.Control
	section string
}

func extractAllControls(cat *catalogv1.Catalog) []controlWithContext {
	var results []controlWithContext
	if cat == nil {
		return results
	}

	for _, c := range cat.Controls {
		results = append(results, controlWithContext{ctrl: c, section: ""})
	}

	var traverseGroup func(g *catalogv1.Group, parentSection string)
	traverseGroup = func(g *catalogv1.Group, parentSection string) {
		if g == nil {
			return
		}
		sec := ""
		if g.Id != nil {
			sec = g.Id.Value
		}
		if parentSection != "" && sec != "" {
			sec = parentSection + "/" + sec
		} else if parentSection != "" {
			sec = parentSection
		}

		for _, c := range g.Controls {
			results = append(results, controlWithContext{ctrl: c, section: sec})
		}
		for _, sub := range g.Groups {
			traverseGroup(sub, sec)
		}
	}

	for _, g := range cat.Groups {
		traverseGroup(g, "")
	}

	return results
}

func extractProse(ctrl *catalogv1.Control) string {
	if ctrl == nil || len(ctrl.Parts) == 0 {
		return ""
	}
	var stmts []string
	for _, part := range ctrl.Parts {
		for _, prose := range part.Prose {
			if prose != nil && prose.Value != "" {
				stmts = append(stmts, prose.Value)
			}
		}
	}
	return strings.Join(stmts, "\n\n")
}

func upsertEntity(ctx context.Context, store kg.Store, e *kg.Entity) error {
	existing, err := store.GetEntity(ctx, e.URN)
	if err == nil && existing != nil {
		return store.UpdateEntity(ctx, e)
	}
	return store.CreateEntity(ctx, e)
}
