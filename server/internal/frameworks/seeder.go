package frameworks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
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

func extractExplicitProfileSelection(f Framework) map[string]bool {
	profBytes, err := f.ProfileJSON()
	if err != nil || len(profBytes) == 0 {
		return nil
	}
	var root map[string]interface{}
	if err := json.Unmarshal(profBytes, &root); err != nil {
		return nil
	}
	prof, _ := root["profile"].(map[string]interface{})
	if prof == nil {
		return nil
	}
	imports, _ := prof["imports"].([]interface{})
	selected := make(map[string]bool)
	for _, impRaw := range imports {
		imp, _ := impRaw.(map[string]interface{})
		if _, hasAll := imp["include-all"]; hasAll {
			return nil
		}
		incList, _ := imp["include-controls"].([]interface{})
		for _, incRaw := range incList {
			inc, _ := incRaw.(map[string]interface{})
			withIDs, _ := inc["with-ids"].([]interface{})
			for _, wid := range withIDs {
				if s, ok := wid.(string); ok && s != "" {
					selected[strings.ToLower(s)] = true
				}
			}
		}
	}
	if len(selected) == 0 {
		return nil
	}
	return selected
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

	var batch []*kg.Entity
	batch = append(batch, fwEntity)

	seededGroups := make(map[string]bool)
	controls := extractAllProtoControls(cat)
	profSelected := extractExplicitProfileSelection(f)

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
		if item.parentCtrlID != "" {
			parentURN = fmt.Sprintf("urn:xoscal:requirement:%s:%s", f.ID(), item.parentCtrlID)
		} else if item.section != "" {
			groupURN := fmt.Sprintf("urn:xoscal:requirement:%s:group:%s", f.ID(), item.section)
			parentURN = groupURN

			if !seededGroups[item.section] {
				groupTitle := strings.ToUpper(item.section)
				if item.sectionTitle != "" {
					groupTitle = item.sectionTitle
				}
				groupReq := kg.Requirement{
					URN:        groupURN,
					Type:       "Requirement",
					Framework:  f.ID(),
					NodeRefID:  item.section,
					NodeName:   groupTitle,
					Citation:   item.section,
					Assessable: false,
				}
				gPayload, err := json.Marshal(groupReq)
				if err != nil {
					return fmt.Errorf("marshal group %s: %w", groupURN, err)
				}
				batch = append(batch, &kg.Entity{
					URN:       groupURN,
					Type:      "reg:Requirement",
					Version:   1,
					Status:    kg.EntityStatusActive,
					ValidFrom: now,
					Payload:   gPayload,
				})
				seededGroups[item.section] = true
			}
		}

		title := ""
		if ctrl.Title != nil {
			title = ctrl.Title.Value
		}

		statement, guidance, canadianGuidance := extractProtoStatementAndGuidance(ctrl)
		var kgParams []kg.ParamDef
		for _, p := range ctrl.Params {
			if p != nil && p.Id != nil {
				lbl := ""
				if p.Label != nil {
					lbl = p.Label.Value
				}
				var vals []string
				for _, v := range p.Values {
					if v != nil && v.Value != "" {
						vals = append(vals, v.Value)
					}
				}
				var choices []string
				howMany := ""
				if p.Select != nil {
					if p.Select.HowMany {
						howMany = "one-or-more"
					}
					for _, ch := range p.Select.Choices {
						if ch != nil && ch.Value != "" {
							choices = append(choices, ch.Value)
						}
					}
				}
				if lbl == "" && len(vals) > 0 {
					lbl = strings.Join(vals, ", ")
				} else if lbl == "" && len(choices) > 0 {
					lbl = strings.Join(choices, "; ")
				}
				kgParams = append(kgParams, kg.ParamDef{
					ID:      p.Id.Value,
					Label:   lbl,
					Values:  vals,
					Choices: choices,
					HowMany: howMany,
					Class:   p.Class,
				})
			}
		}

		var implGroups []string
		if profSelected != nil && profSelected[strings.ToLower(ctrlID)] {
			implGroups = []string{"profile-selected"}
		}

		req := kg.Requirement{
			URN:                  fmt.Sprintf("urn:xoscal:requirement:%s:%s", f.ID(), ctrlID),
			Type:                 "Requirement",
			Framework:            f.ID(),
			Citation:             ctrlID,
			NodeRefID:            ctrlID,
			Title:                title,
			Class:                ctrl.Class,
			Text:                 statement,
			Guidance:             guidance,
			CanadianGuidance:     canadianGuidance,
			Params:               kgParams,
			Props:                protoPropsToKG(ctrl.Props),
			Parts:                protoPartsToKG(ctrl.Parts),
			Section:              item.section,
			ParentURN:            parentURN,
			Assessable:           true,
			ImplementationGroups: implGroups,
		}

		reqPayload, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal requirement %s: %w", req.URN, err)
		}

		batch = append(batch, &kg.Entity{
			URN:       req.URN,
			Type:      "reg:Requirement",
			Version:   1,
			Status:    kg.EntityStatusActive,
			ValidFrom: now,
			Payload:   reqPayload,
		})
	}

	return upsertEntitiesBatch(ctx, store, batch)
}

type protoControlWithContext struct {
	ctrl         *catalogv1.Control
	section      string
	sectionTitle string
	parentCtrlID string
}

func extractAllProtoControls(cat *catalogv1.Catalog) []protoControlWithContext {
	var results []protoControlWithContext
	if cat == nil {
		return results
	}

	var traverseControl func(c *catalogv1.Control, sec, secTitle, parentCtrlID string)
	traverseControl = func(c *catalogv1.Control, sec, secTitle, parentCtrlID string) {
		if c == nil {
			return
		}
		effSec := sec
		effSecTitle := secTitle
		if effSec == "" {
			for _, prop := range c.Props {
				if prop == nil {
					continue
				}
				if strings.EqualFold(prop.Name, "family-id") && prop.Value != "" {
					effSec = prop.Value
				}
				if strings.EqualFold(prop.Name, "family-title") && prop.Value != "" {
					effSecTitle = prop.Value
				}
			}
		}
		results = append(results, protoControlWithContext{
			ctrl:         c,
			section:      effSec,
			sectionTitle: effSecTitle,
			parentCtrlID: parentCtrlID,
		})
		ctrlID := ""
		if c.Id != nil {
			ctrlID = c.Id.Value
		}
		for _, sub := range c.Controls {
			traverseControl(sub, effSec, effSecTitle, ctrlID)
		}
	}

	for _, c := range cat.Controls {
		traverseControl(c, "", "", "")
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
		secTitle := ""
		if g.Title != nil {
			secTitle = g.Title.Value
		}

		for _, c := range g.Controls {
			traverseControl(c, sec, secTitle, "")
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

func extractProtoStatementAndGuidance(ctrl *catalogv1.Control) (string, string, string) {
	if ctrl == nil || len(ctrl.Parts) == 0 {
		return "", "", ""
	}
	var stmts []string
	var guidance []string
	var canadianGuidance []string
	var visitPart func(part *catalogv1.Part, mode string)
	visitPart = func(part *catalogv1.Part, mode string) {
		if part == nil {
			return
		}
		pName := strings.ToLower(part.Name)
		if pName == "gc-guidance" || pName == "canadian-guidance" {
			mode = "canadian"
		} else if pName == "guidance" {
			mode = "guidance"
		}
		lbl := ""
		if part.Title != nil {
			lbl = part.Title.Value
		}
		for _, prop := range part.Props {
			if prop != nil && strings.EqualFold(prop.Name, "label") && prop.Value != "" {
				lbl = prop.Value
			}
		}
		for _, prose := range part.Prose {
			if prose != nil && prose.Value != "" {
				if mode == "canadian" {
					canadianGuidance = append(canadianGuidance, prose.Value)
					guidance = append(guidance, prose.Value)
				} else if mode == "guidance" {
					guidance = append(guidance, prose.Value)
				} else if strings.EqualFold(part.Name, "item") && lbl != "" {
					stmts = append(stmts, lbl+" "+prose.Value)
				} else {
					stmts = append(stmts, prose.Value)
				}
			}
		}
		for _, sub := range part.Parts {
			visitPart(sub, mode)
		}
	}
	for _, part := range ctrl.Parts {
		visitPart(part, "")
	}
	return strings.Join(stmts, "\n\n"), strings.Join(guidance, "\n\n"), strings.Join(canadianGuidance, "\n\n")
}

func protoPropsToKG(props []*commonv1.Property) []kg.PropDef {
	var out []kg.PropDef
	for _, p := range props {
		if p == nil || p.Name == "" {
			continue
		}
		out = append(out, kg.PropDef{
			Name:  p.Name,
			Value: p.Value,
			Ns:    p.Ns,
			Class: p.Class,
			ID:    p.Id,
		})
	}
	return out
}

func protoPartsToKG(parts []*catalogv1.Part) []kg.PartDef {
	var out []kg.PartDef
	for _, p := range parts {
		if p == nil {
			continue
		}
		id := ""
		if p.Id != nil {
			id = p.Id.Value
		}
		title := ""
		if p.Title != nil {
			title = p.Title.Value
		}
		var proseList []string
		for _, pr := range p.Prose {
			if pr != nil && pr.Value != "" {
				proseList = append(proseList, pr.Value)
			}
		}
		out = append(out, kg.PartDef{
			ID:    id,
			Name:  p.Name,
			Ns:    p.Ns,
			Class: p.Class,
			Title: title,
			Prose: strings.Join(proseList, "\n\n"),
			Props: protoPropsToKG(p.Props),
			Parts: protoPartsToKG(p.Parts),
		})
	}
	return out
}

func jsonPropsToKG(props []oscalJSONProp) []kg.PropDef {
	var out []kg.PropDef
	for _, p := range props {
		if p.Name == "" {
			continue
		}
		out = append(out, kg.PropDef{
			Name:  p.Name,
			Value: p.Value,
			Ns:    p.Ns,
			Class: p.Class,
			ID:    p.ID,
		})
	}
	return out
}

func jsonPartsToKG(parts []oscalJSONPart) []kg.PartDef {
	var out []kg.PartDef
	for _, p := range parts {
		out = append(out, kg.PartDef{
			ID:    p.ID,
			Name:  p.Name,
			Ns:    p.Ns,
			Class: p.Class,
			Title: p.Title,
			Prose: p.Prose,
			Props: jsonPropsToKG(p.Props),
			Parts: jsonPartsToKG(p.Parts),
		})
	}
	return out
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
	Class    string             `json:"class"`
	Title    string             `json:"title"`
	Params   []oscalJSONParam   `json:"params"`
	Props    []oscalJSONProp    `json:"props"`
	Parts    []oscalJSONPart    `json:"parts"`
	Controls []oscalJSONControl `json:"controls"`
}

type oscalJSONParam struct {
	ID     string   `json:"id"`
	Class  string   `json:"class"`
	Label  string   `json:"label"`
	Values []string `json:"values"`
	Select *struct {
		HowMany string   `json:"how-many"`
		Choice  []string `json:"choice"`
	} `json:"select"`
}

type oscalJSONProp struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
	Ns    string `json:"ns"`
	Class string `json:"class"`
}

type oscalJSONPart struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Ns    string          `json:"ns"`
	Class string          `json:"class"`
	Title string          `json:"title"`
	Prose string          `json:"prose"`
	Parts []oscalJSONPart `json:"parts"`
	Props []oscalJSONProp `json:"props"`
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

	var batch []*kg.Entity
	batch = append(batch, &kg.Entity{
		URN:       fw.URN,
		Type:      "reg:Framework",
		Version:   1,
		Status:    kg.EntityStatusActive,
		ValidFrom: now,
		Payload:   fwPayload,
	})

	seededGroups := make(map[string]bool)
	profSelected := extractExplicitProfileSelection(f)

	var seedControl func(c oscalJSONControl, sec, parentURN string) error
	seedControl = func(c oscalJSONControl, sec, parentURN string) error {
		if c.ID == "" {
			return nil
		}

		effSec := sec
		var effSecTitle string
		if effSec == "" {
			for _, prop := range c.Props {
				if strings.EqualFold(prop.Name, "family-id") && prop.Value != "" {
					effSec = prop.Value
				}
				if strings.EqualFold(prop.Name, "family-title") && prop.Value != "" {
					effSecTitle = prop.Value
				}
			}
		}

		if parentURN == "" && effSec != "" {
			groupURN := fmt.Sprintf("urn:xoscal:requirement:%s:group:%s", f.ID(), effSec)
			parentURN = groupURN
			if !seededGroups[effSec] {
				gTitle := strings.ToUpper(effSec)
				if effSecTitle != "" {
					gTitle = effSecTitle
				}
				groupReq := kg.Requirement{
					URN:        groupURN,
					Type:       "Requirement",
					Framework:  f.ID(),
					NodeRefID:  effSec,
					NodeName:   gTitle,
					Citation:   effSec,
					Assessable: false,
				}
				gPayload, err := json.Marshal(groupReq)
				if err != nil {
					return fmt.Errorf("marshal group %s: %w", groupURN, err)
				}
				batch = append(batch, &kg.Entity{
					URN:       groupURN,
					Type:      "reg:Requirement",
					Version:   1,
					Status:    kg.EntityStatusActive,
					ValidFrom: now,
					Payload:   gPayload,
				})
				seededGroups[effSec] = true
			}
		}

		var stmts []string
		var guidance []string
		var canadianGuidance []string
		var visitPart func(p oscalJSONPart, mode string)
		visitPart = func(p oscalJSONPart, mode string) {
			pName := strings.ToLower(p.Name)
			if pName == "gc-guidance" || pName == "canadian-guidance" {
				mode = "canadian"
			} else if pName == "guidance" {
				mode = "guidance"
			}
			lbl := p.Title
			for _, prop := range p.Props {
				if strings.EqualFold(prop.Name, "label") && prop.Value != "" {
					lbl = prop.Value
				}
			}
			if p.Prose != "" {
				if mode == "canadian" {
					canadianGuidance = append(canadianGuidance, p.Prose)
					guidance = append(guidance, p.Prose)
				} else if mode == "guidance" {
					guidance = append(guidance, p.Prose)
				} else if strings.EqualFold(p.Name, "item") && lbl != "" {
					stmts = append(stmts, lbl+" "+p.Prose)
				} else {
					stmts = append(stmts, p.Prose)
				}
			}
			for _, sub := range p.Parts {
				visitPart(sub, mode)
			}
		}
		for _, p := range c.Parts {
			visitPart(p, "")
		}

		var kgParams []kg.ParamDef
		for _, p := range c.Params {
			lbl := p.Label
			var choices []string
			howMany := ""
			if p.Select != nil {
				howMany = p.Select.HowMany
				choices = p.Select.Choice
			}
			if lbl == "" && len(p.Values) > 0 {
				lbl = strings.Join(p.Values, ", ")
			} else if lbl == "" && len(choices) > 0 {
				lbl = strings.Join(choices, "; ")
			}
			kgParams = append(kgParams, kg.ParamDef{
				ID:      p.ID,
				Label:   lbl,
				Values:  p.Values,
				Choices: choices,
				HowMany: howMany,
				Class:   p.Class,
			})
		}

		var implGroups []string
		if profSelected != nil && profSelected[strings.ToLower(c.ID)] {
			implGroups = []string{"profile-selected"}
		}

		req := kg.Requirement{
			URN:                  fmt.Sprintf("urn:xoscal:requirement:%s:%s", f.ID(), c.ID),
			Type:                 "Requirement",
			Framework:            f.ID(),
			Citation:             c.ID,
			NodeRefID:            c.ID,
			Title:                c.Title,
			Class:                c.Class,
			Text:                 strings.Join(stmts, "\n\n"),
			Guidance:             strings.Join(guidance, "\n\n"),
			CanadianGuidance:     strings.Join(canadianGuidance, "\n\n"),
			Params:               kgParams,
			Props:                jsonPropsToKG(c.Props),
			Parts:                jsonPartsToKG(c.Parts),
			Section:              effSec,
			ParentURN:            parentURN,
			Assessable:           true,
			ImplementationGroups: implGroups,
		}

		reqPayload, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal requirement %s: %w", req.URN, err)
		}

		batch = append(batch, &kg.Entity{
			URN:       req.URN,
			Type:      "reg:Requirement",
			Version:   1,
			Status:    kg.EntityStatusActive,
			ValidFrom: now,
			Payload:   reqPayload,
		})

		ctrlURN := req.URN
		for _, sc := range c.Controls {
			if err := seedControl(sc, effSec, ctrlURN); err != nil {
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
			gTitle := g.Title
			if gTitle == "" {
				gTitle = strings.ToUpper(sec)
			}
			groupReq := kg.Requirement{
				URN:        groupURN,
				Type:       "Requirement",
				Framework:  f.ID(),
				NodeRefID:  sec,
				NodeName:   gTitle,
				Citation:   sec,
				Assessable: false,
			}
			gPayload, err := json.Marshal(groupReq)
			if err != nil {
				return fmt.Errorf("marshal group %s: %w", groupURN, err)
			}
			batch = append(batch, &kg.Entity{
				URN:       groupURN,
				Type:      "reg:Requirement",
				Version:   1,
				Status:    kg.EntityStatusActive,
				ValidFrom: now,
				Payload:   gPayload,
			})
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

	return upsertEntitiesBatch(ctx, store, batch)
}

type bulkUpsertStore interface {
	BulkUpsertEntities(ctx context.Context, entities []*kg.Entity) error
}

func upsertEntitiesBatch(ctx context.Context, store kg.Store, batch []*kg.Entity) error {
	if bs, ok := store.(bulkUpsertStore); ok {
		return bs.BulkUpsertEntities(ctx, batch)
	}
	for _, e := range batch {
		if err := upsertEntity(ctx, store, e); err != nil {
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
