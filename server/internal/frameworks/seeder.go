package frameworks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
)

// SeedFramework seeds a single framework and its controls into the Knowledge Graph store.
func SeedFramework(ctx context.Context, store kg.Store, f Framework) error {
	// If custom seeder is provided and active, invoke it
	if seeder, ok := f.(KGSeeder); ok {
		err := seeder.SeedKG(ctx, store)
		if err == nil {
			return nil
		}
		if !errors.Is(err, ErrNoCustomSeeder) {
			return err
		}
	}

	// If native proto is provided, seed via proto
	if protoProv, ok := f.(ProtoCatalogProvider); ok {
		catProto, err := protoProv.CatalogProto()
		if err == nil && catProto != nil {
			return seedProtoCatalog(ctx, store, f, catProto)
		}
	}

	// Fallback to seeding from OSCAL Catalog JSON
	return seedJSONCatalog(ctx, store, f)
}

func seedProtoCatalog(ctx context.Context, store kg.Store, f Framework, cat *catalogv1.Catalog) error {
	now := time.Now().UTC()

	fw := kg.Framework{
		URN:      fmt.Sprintf("urn:xoscal:framework:%s", f.ID()),
		Type:     "Framework",
		RefID:    f.ID(),
		Name:     f.Title(),
		Version:  f.Version(),
		Locale:   f.Jurisdiction(),
		Provider: f.Authority(),
	}

	fwPayload, err := json.Marshal(fw)
	if err != nil {
		return fmt.Errorf("marshal framework %s: %w", f.ID(), err)
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
		return fmt.Errorf("upsert framework %s: %w", f.ID(), err)
	}

	seededGroups := make(map[string]bool)
	controls := extractAllProtoControls(cat)

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
			groupURN := fmt.Sprintf("urn:xoscal:requirement:%s:group:%s", f.ID(), item.section)
			parentURN = groupURN

			if !seededGroups[item.section] {
				groupReq := kg.Requirement{
					URN:        groupURN,
					Type:       "Requirement",
					Framework:  f.ID(),
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

		statement := extractProtoProse(ctrl)

		req := kg.Requirement{
			URN:        fmt.Sprintf("urn:xoscal:requirement:%s:%s", f.ID(), ctrlID),
			Type:       "Requirement",
			Framework:  f.ID(),
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

	return nil
}

type protoControlWithContext struct {
	ctrl    *catalogv1.Control
	section string
}

func extractAllProtoControls(cat *catalogv1.Catalog) []protoControlWithContext {
	var results []protoControlWithContext
	if cat == nil {
		return results
	}

	for _, c := range cat.Controls {
		results = append(results, protoControlWithContext{ctrl: c, section: ""})
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
			results = append(results, protoControlWithContext{ctrl: c, section: sec})
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

func extractProtoProse(ctrl *catalogv1.Control) string {
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

// Minimal OSCAL JSON structs for seeding from raw catalog JSON
type oscalCatalogDoc struct {
	Catalog struct {
		Groups   []oscalJSONGroup   `json:"groups"`
		Controls []oscalJSONControl `json:"controls"`
	} `json:"catalog"`
}

type oscalJSONGroup struct {
	ID       string             `json:"id"`
	Title    string             `json:"title"`
	Groups   []oscalJSONGroup   `json:"groups"`
	Controls []oscalJSONControl `json:"controls"`
}

type oscalJSONControl struct {
	ID       string             `json:"id"`
	Title    string             `json:"title"`
	Parts    []oscalJSONPart    `json:"parts"`
	Controls []oscalJSONControl `json:"controls"`
}

type oscalJSONPart struct {
	Name  string `json:"name"`
	Prose string `json:"prose"`
}

func seedJSONCatalog(ctx context.Context, store kg.Store, f Framework) error {
	data, err := f.CatalogJSON()
	if err != nil {
		return err
	}

	var doc oscalCatalogDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("unmarshal oscal catalog json: %w", err)
	}

	now := time.Now().UTC()
	fw := kg.Framework{
		URN:      fmt.Sprintf("urn:xoscal:framework:%s", f.ID()),
		Type:     "Framework",
		RefID:    f.ID(),
		Name:     f.Title(),
		Version:  f.Version(),
		Locale:   f.Jurisdiction(),
		Provider: f.Authority(),
	}

	fwPayload, err := json.Marshal(fw)
	if err != nil {
		return fmt.Errorf("marshal framework %s: %w", f.ID(), err)
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
		return fmt.Errorf("upsert framework %s: %w", f.ID(), err)
	}

	seededGroups := make(map[string]bool)

	var seedControl func(c oscalJSONControl, sec, parentURN string) error
	seedControl = func(c oscalJSONControl, sec, parentURN string) error {
		if c.ID == "" {
			return nil
		}

		var stmts []string
		for _, p := range c.Parts {
			if p.Prose != "" {
				stmts = append(stmts, p.Prose)
			}
		}

		req := kg.Requirement{
			URN:        fmt.Sprintf("urn:xoscal:requirement:%s:%s", f.ID(), c.ID),
			Type:       "Requirement",
			Framework:  f.ID(),
			Citation:   c.ID,
			Title:      c.Title,
			Text:       strings.Join(stmts, "\n\n"),
			Section:    sec,
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

		ctrlURN := req.URN
		for _, sc := range c.Controls {
			if err := seedControl(sc, sec, ctrlURN); err != nil {
				return err
			}
		}

		return nil
	}

	var seedGroup func(g oscalJSONGroup, parentSec string) error
	seedGroup = func(g oscalJSONGroup, parentSec string) error {
		sec := g.ID
		if parentSec != "" && sec != "" {
			sec = parentSec + "/" + sec
		} else if parentSec != "" {
			sec = parentSec
		}

		groupURN := fmt.Sprintf("urn:xoscal:requirement:%s:group:%s", f.ID(), sec)
		if !seededGroups[sec] {
			groupReq := kg.Requirement{
				URN:        groupURN,
				Type:       "Requirement",
				Framework:  f.ID(),
				NodeRefID:  sec,
				NodeName:   strings.ToUpper(g.Title),
				Citation:   sec,
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
			seededGroups[sec] = true
		}

		for _, c := range g.Controls {
			if err := seedControl(c, sec, groupURN); err != nil {
				return err
			}
		}

		for _, sub := range g.Groups {
			if err := seedGroup(sub, sec); err != nil {
				return err
			}
		}

		return nil
	}

	for _, g := range doc.Catalog.Groups {
		if err := seedGroup(g, ""); err != nil {
			return err
		}
	}
	for _, c := range doc.Catalog.Controls {
		if err := seedControl(c, "", ""); err != nil {
			return err
		}
	}

	return nil
}

func upsertEntity(ctx context.Context, store kg.Store, e *kg.Entity) error {
	existing, err := store.GetEntity(ctx, e.URN)
	if err == nil && existing != nil {
		return store.UpdateEntity(ctx, e)
	}
	return store.CreateEntity(ctx, e)
}
