package frameworks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	catalogv1 "github.com/mchorfa/xoscal/proto/oscal/catalog/v1"
	commonv1 "github.com/mchorfa/xoscal/proto/oscal/common/v1"
	"github.com/mchorfa/xoscal/server/internal/kg"
	"github.com/mchorfa/xoscal/server/internal/ontology"
)

// SeedKnowledgeGraph seeds all frameworks in the provided registry (or DefaultRegistry if nil)
// into both kg_entities and the property graph tables (kg_graph_nodes, kg_graph_edges).
func SeedKnowledgeGraph(ctx context.Context, store kg.Store, regs ...*Registry) error {
	var reg *Registry
	if len(regs) > 0 && regs[0] != nil {
		reg = regs[0]
	} else {
		reg = DefaultRegistry()
	}
	return reg.SeedKG(ctx, store)
}

type seedBatch struct {
	entities       []*kg.Entity
	frameworks     []kg.Framework
	assessableReqs []kg.Requirement
}

func (b *seedBatch) appendFrom(other *seedBatch) {
	if other == nil {
		return
	}
	b.entities = append(b.entities, other.entities...)
	b.frameworks = append(b.frameworks, other.frameworks...)
	b.assessableReqs = append(b.assessableReqs, other.assessableReqs...)
}

func seedFrameworksBatch(ctx context.Context, store kg.Store, fws []Framework) error {
	if len(fws) == 0 {
		return nil
	}
	standard := make([]Framework, 0, len(fws))
	for _, f := range fws {
		if seeder, ok := f.(KGSeeder); ok {
			err := seeder.SeedKG(ctx, store)
			if err == nil {
				continue
			}
			if !errors.Is(err, ErrNoCustomSeeder) {
				return fmt.Errorf("seed framework %s: %w", f.ID(), err)
			}
		}
		standard = append(standard, f)
	}
	if len(standard) == 0 {
		return nil
	}

	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}
	if workers > 8 {
		workers = 8
	}
	if workers > len(standard) {
		workers = len(standard)
	}

	batches := make([]*seedBatch, len(standard))
	errs := make([]error, len(standard))
	jobs := make(chan int, len(standard))
	for i := range standard {
		jobs <- i
	}
	close(jobs)

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for idx := range jobs {
				sb, err := collectFrameworkSeedBatch(standard[idx])
				if err != nil {
					errs[idx] = fmt.Errorf("seed framework %s: %w", standard[idx].ID(), err)
					continue
				}
				batches[idx] = sb
			}
		}()
	}
	wg.Wait()

	totalEntities := 0
	totalFws := 0
	totalReqs := 0
	for i, err := range errs {
		if err != nil {
			return err
		}
		if batches[i] != nil {
			totalEntities += len(batches[i].entities)
			totalFws += len(batches[i].frameworks)
			totalReqs += len(batches[i].assessableReqs)
		}
	}

	combined := &seedBatch{
		entities:       make([]*kg.Entity, 0, totalEntities),
		frameworks:     make([]kg.Framework, 0, totalFws),
		assessableReqs: make([]kg.Requirement, 0, totalReqs),
	}
	for _, sb := range batches {
		combined.appendFrom(sb)
	}
	return commitSeedBatch(ctx, store, combined)
}

func collectFrameworkSeedBatch(f Framework) (*seedBatch, error) {
	if protoProv, ok := f.(ProtoCatalogProvider); ok {
		catProto, err := protoProv.CatalogProto()
		if err == nil && catProto != nil {
			return buildProtoCatalogBatch(f, catProto)
		}
	}
	return buildJSONCatalogBatch(f)
}

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

	sb, err := collectFrameworkSeedBatch(f)
	if err != nil {
		return err
	}
	return commitSeedBatch(ctx, store, sb)
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

func buildProtoCatalogBatch(f Framework, cat *catalogv1.Catalog) (*seedBatch, error) {
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
		return nil, fmt.Errorf("marshal framework %s: %w", f.ID(), err)
	}

	fwEntity := &kg.Entity{
		URN:       fw.URN,
		Type:      "reg:Framework",
		Version:   1,
		Status:    kg.EntityStatusActive,
		ValidFrom: now,
		Payload:   fwPayload,
	}

	sb := &seedBatch{
		entities:   []*kg.Entity{fwEntity},
		frameworks: []kg.Framework{fw},
	}

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
					return nil, fmt.Errorf("marshal group %s: %w", groupURN, err)
				}
				sb.entities = append(sb.entities, &kg.Entity{
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
			return nil, fmt.Errorf("marshal requirement %s: %w", req.URN, err)
		}

		sb.entities = append(sb.entities, &kg.Entity{
			URN:       req.URN,
			Type:      "reg:Requirement",
			Version:   1,
			Status:    kg.EntityStatusActive,
			ValidFrom: now,
			Payload:   reqPayload,
		})
		sb.assessableReqs = append(sb.assessableReqs, req)
	}

	return sb, nil
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
		switch pName {
		case "gc-guidance", "canadian-guidance":
			mode = "canadian"
		case "guidance":
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

func buildJSONCatalogBatch(f Framework) (*seedBatch, error) {
	data, err := f.CatalogJSON()
	if err != nil {
		return nil, err
	}

	var doc oscalCatalogDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal oscal catalog json: %w", err)
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
		return nil, fmt.Errorf("marshal framework %s: %w", f.ID(), err)
	}

	sb := &seedBatch{
		entities: []*kg.Entity{{
			URN:       fw.URN,
			Type:      "reg:Framework",
			Version:   1,
			Status:    kg.EntityStatusActive,
			ValidFrom: now,
			Payload:   fwPayload,
		}},
		frameworks: []kg.Framework{fw},
	}

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
				sb.entities = append(sb.entities, &kg.Entity{
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
			switch pName {
			case "gc-guidance", "canadian-guidance":
				mode = "canadian"
			case "guidance":
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

		sb.entities = append(sb.entities, &kg.Entity{
			URN:       req.URN,
			Type:      "reg:Requirement",
			Version:   1,
			Status:    kg.EntityStatusActive,
			ValidFrom: now,
			Payload:   reqPayload,
		})
		sb.assessableReqs = append(sb.assessableReqs, req)

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
			sb.entities = append(sb.entities, &kg.Entity{
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
			return nil, err
		}
	}
	for _, c := range doc.Catalog.Controls {
		if err := seedControl(c, "", ""); err != nil {
			return nil, err
		}
	}

	return sb, nil
}

type bulkUpsertStore interface {
	BulkUpsertEntities(ctx context.Context, entities []*kg.Entity) error
}

type sqlDBProvider interface {
	DB() *sql.DB
}

func commitSeedBatch(ctx context.Context, store kg.Store, sb *seedBatch) error {
	if sb == nil || len(sb.entities) == 0 {
		return nil
	}
	if dbProv, ok := store.(sqlDBProvider); ok && dbProv.DB() != nil {
		return commitSeedBatchSQLite(ctx, dbProv.DB(), sb)
	}
	if bs, ok := store.(bulkUpsertStore); ok {
		return bs.BulkUpsertEntities(ctx, sb.entities)
	}
	for _, e := range sb.entities {
		if err := upsertEntity(ctx, store, e); err != nil {
			return err
		}
	}
	return nil
}

type graphNodeRow struct {
	urn        string
	kind       string
	labelsJSON string
}

type graphEdgeRow struct {
	id      string
	fromURN string
	toURN   string
	rel     string
	claimID string
	weight  float64
}

type jurLabel struct {
	Code string `json:"code"`
}

type authLabel struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type fwLabel struct {
	ID           string `json:"id"`
	Jurisdiction string `json:"jurisdiction"`
	Title        string `json:"title"`
	Version      string `json:"version"`
}

type fwStubLabel struct {
	ID string `json:"id"`
}

type domStubLabel struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

type parentStubLabel struct {
	Framework string `json:"framework"`
}

func formatCtrlLabelJSON(family, framework, id, domain, title string) string {
	b := make([]byte, 0, len(family)+len(framework)+len(id)+len(domain)+len(title)+76)
	b = append(b, `{"family":`...)
	b = strconv.AppendQuote(b, family)
	b = append(b, `,"framework":`...)
	b = strconv.AppendQuote(b, framework)
	b = append(b, `,"id":`...)
	b = strconv.AppendQuote(b, id)
	b = append(b, `,"thematic_domain":`...)
	b = strconv.AppendQuote(b, domain)
	b = append(b, `,"title":`...)
	b = strconv.AppendQuote(b, title)
	b = append(b, '}')
	return string(b)
}

func formatODPLabelJSON(framework, id, label string) string {
	b := make([]byte, 0, len(framework)+len(id)+len(label)+36)
	b = append(b, `{"framework":`...)
	b = strconv.AppendQuote(b, framework)
	b = append(b, `,"id":`...)
	b = strconv.AppendQuote(b, id)
	b = append(b, `,"label":`...)
	b = strconv.AppendQuote(b, label)
	b = append(b, '}')
	return string(b)
}

const sqliteHexDigits = "0123456789abcdef"

func appendSQLiteBlob(b *strings.Builder, data []byte) {
	b.Grow(len(data)*2 + 3)
	b.WriteString("X'")
	for _, v := range data {
		b.WriteByte(sqliteHexDigits[v>>4])
		b.WriteByte(sqliteHexDigits[v&0x0f])
	}
	b.WriteByte('\'')
}

func appendSQLiteText(b *strings.Builder, s string) {
	b.WriteByte('\'')
	if strings.IndexByte(s, '\'') < 0 && strings.IndexByte(s, 0) < 0 {
		b.WriteString(s)
	} else {
		for i := 0; i < len(s); i++ {
			ch := s[i]
			if ch == '\'' {
				b.WriteString("''")
			} else if ch != 0 {
				b.WriteByte(ch)
			}
		}
	}
	b.WriteByte('\'')
}

func buildValuesPlaceholders(rows, cols int) string {
	var b strings.Builder
	b.Grow(rows * (cols*2 + 3))
	for r := 0; r < rows; r++ {
		if r > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('(')
		for c := 0; c < cols; c++ {
			if c > 0 {
				b.WriteByte(',')
			}
			b.WriteByte('?')
		}
		b.WriteByte(')')
	}
	return b.String()
}

func commitSeedBatchSQLite(ctx context.Context, db *sql.DB, sb *seedBatch) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire sqlite conn: %w", err)
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), `PRAGMA foreign_keys = ON; PRAGMA synchronous = NORMAL;`)
		_ = conn.Close()
	}()

	// PRAGMA foreign_keys MUST be set outside an active transaction in SQLite.
	// Referential integrity is guaranteed below because every edge endpoint is
	// explicitly inserted into kg_graph_nodes before kg_graph_edges in the same tx.
	_, _ = conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF; PRAGMA synchronous = OFF; PRAGMA temp_store = MEMORY; PRAGMA cache_size = -64000;`)

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin seed tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 1. Load existing max versions in one query so fresh DBs do zero per-row SELECTs.
	existingMaxVer := make(map[string]int)
	rows, err := tx.QueryContext(ctx, `SELECT urn, MAX(version) FROM kg_entities GROUP BY urn`)
	if err != nil {
		return fmt.Errorf("query existing entity versions: %w", err)
	}
	for rows.Next() {
		var urn string
		var mv int
		if err := rows.Scan(&urn, &mv); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan entity version: %w", err)
		}
		existingMaxVer[urn] = mv
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	_ = rows.Close()

	var existingNodes, existingEdges int
	if len(existingMaxVer) == 0 {
		_ = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM kg_graph_nodes`).Scan(&existingNodes)
		_ = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM kg_graph_edges`).Scan(&existingEdges)
	}

	now := time.Now().UTC()
	var expireStmt *sql.Stmt
	defer func() {
		if expireStmt != nil {
			_ = expireStmt.Close()
		}
	}()

	dedupEntities := make([]*kg.Entity, 0, len(sb.entities))
	for _, entity := range sb.entities {
		if entity == nil {
			continue
		}
		if entity.Status == "" {
			entity.Status = kg.EntityStatusActive
		}
		if entity.ValidFrom.IsZero() {
			entity.ValidFrom = now
		}
		mv := existingMaxVer[entity.URN]
		if mv > 0 {
			if expireStmt == nil {
				expireStmt, err = tx.PrepareContext(ctx,
					`UPDATE kg_entities SET status = ?, valid_to = ? WHERE urn = ? AND status = ?`)
				if err != nil {
					return fmt.Errorf("prepare expire entity stmt: %w", err)
				}
			}
			if _, err := expireStmt.ExecContext(ctx, string(kg.EntityStatusSuperseded), now, entity.URN, string(kg.EntityStatusActive)); err != nil {
				return fmt.Errorf("expire old version %s: %w", entity.URN, err)
			}
		}
		entity.Version = mv + 1
		existingMaxVer[entity.URN] = entity.Version
		dedupEntities = append(dedupEntities, entity)
	}

	isFreshBulk := len(existingMaxVer) == len(dedupEntities) && existingNodes == 0 && existingEdges == 0 && expireStmt == nil

	if isFreshBulk {
		dropIdxSQL := `
DROP INDEX IF EXISTS idx_kg_type;
DROP INDEX IF EXISTS idx_kg_status;
DROP INDEX IF EXISTS idx_kg_valid;
DROP INDEX IF EXISTS idx_nodes_kind;
DROP INDEX IF EXISTS idx_nodes_urn;
DROP INDEX IF EXISTS idx_edges_from;
DROP INDEX IF EXISTS idx_edges_to;
DROP INDEX IF EXISTS idx_edges_relation;
DROP INDEX IF EXISTS idx_edges_trust;
DROP INDEX IF EXISTS idx_edges_valid;
`
		if _, err := tx.ExecContext(ctx, dropIdxSQL); err != nil {
			return fmt.Errorf("drop secondary indexes for bulk seed: %w", err)
		}
		sort.Slice(dedupEntities, func(i, j int) bool {
			return dedupEntities[i].URN < dedupEntities[j].URN
		})
	}

	// 2. Multi-row bulk INSERT into kg_entities.
	// Using SQL-escaped text literals with a single bound ?1 (now) avoids hundreds of thousands
	// of per-parameter modernc.org/libc Xmalloc/Xfree C-heap allocations under -race.
	const entityPrefix = `INSERT INTO kg_entities(urn, entity_type, version, status, valid_from, valid_to, payload) VALUES `
	var sqlBuf strings.Builder
	if isFreshBulk {
		const freshEntityBatch = 200
		sqlBuf.Grow(freshEntityBatch * 512)
		for start := 0; start < len(dedupEntities); start += freshEntityBatch {
			end := start + freshEntityBatch
			if end > len(dedupEntities) {
				end = len(dedupEntities)
			}
			chunk := dedupEntities[start:end]
			sqlBuf.Reset()
			sqlBuf.WriteString(entityPrefix)
			for i, ent := range chunk {
				if i > 0 {
					sqlBuf.WriteByte(',')
				}
				sqlBuf.WriteByte('(')
				appendSQLiteText(&sqlBuf, ent.URN)
				sqlBuf.WriteByte(',')
				appendSQLiteText(&sqlBuf, ent.Type)
				sqlBuf.WriteString(",1,'active',?1,NULL,")
				appendSQLiteBlob(&sqlBuf, ent.Payload)
				sqlBuf.WriteByte(')')
			}
			if _, err := tx.ExecContext(ctx, sqlBuf.String(), now); err != nil {
				return fmt.Errorf("bulk insert kg_entities: %w", err)
			}
		}
	} else {
		const entityChunkSize = 25
		var entityStmt *sql.Stmt
		if len(dedupEntities) >= entityChunkSize {
			entityStmt, err = tx.PrepareContext(ctx, entityPrefix+buildValuesPlaceholders(entityChunkSize, 7))
			if err != nil {
				return fmt.Errorf("prepare bulk entity stmt: %w", err)
			}
			defer func() { _ = entityStmt.Close() }()
		}
		entityArgs := make([]any, 0, entityChunkSize*7)
		for start := 0; start < len(dedupEntities); start += entityChunkSize {
			end := start + entityChunkSize
			if end > len(dedupEntities) {
				end = len(dedupEntities)
			}
			chunk := dedupEntities[start:end]
			entityArgs = entityArgs[:0]
			for _, ent := range chunk {
				entityArgs = append(entityArgs, ent.URN, ent.Type, ent.Version, string(ent.Status), ent.ValidFrom, ent.ValidTo, []byte(ent.Payload))
			}
			if len(chunk) == entityChunkSize && entityStmt != nil {
				if _, err := entityStmt.ExecContext(ctx, entityArgs...); err != nil {
					return fmt.Errorf("bulk insert kg_entities: %w", err)
				}
			} else {
				q := entityPrefix + buildValuesPlaceholders(len(chunk), 7)
				if _, err := tx.ExecContext(ctx, q, entityArgs...); err != nil {
					return fmt.Errorf("bulk insert kg_entities remainder: %w", err)
				}
			}
		}
	}

	// 3. Build property graph nodes and edges in memory.
	stubMap := make(map[string]graphNodeRow, 128)
	var stubOrder []string
	addStubJSON := func(urn, kind string, labels any) {
		if _, exists := stubMap[urn]; exists {
			return
		}
		lb, _ := json.Marshal(labels)
		stubMap[urn] = graphNodeRow{urn: urn, kind: kind, labelsJSON: string(lb)}
		stubOrder = append(stubOrder, urn)
	}

	nodeMap := make(map[string]graphNodeRow, len(sb.frameworks)+len(sb.assessableReqs)+128)
	nodeOrder := make([]string, 0, len(sb.frameworks)+len(sb.assessableReqs)+128)
	addNodeRaw := func(urn, kind, labelsJSON string) {
		if _, exists := nodeMap[urn]; !exists {
			nodeOrder = append(nodeOrder, urn)
		}
		nodeMap[urn] = graphNodeRow{urn: urn, kind: kind, labelsJSON: labelsJSON}
	}
	addNodeJSON := func(urn, kind string, labels any) {
		lb, _ := json.Marshal(labels)
		addNodeRaw(urn, kind, string(lb))
	}

	edgeMap := make(map[string]graphEdgeRow, len(sb.assessableReqs)*3)
	edgeOrder := make([]string, 0, len(sb.assessableReqs)*3)
	addEdge := func(fromURN, rel, toURN string, weight float64) {
		edgeID := "edge:" + fromURN + ":" + rel + ":" + toURN
		if _, exists := edgeMap[edgeID]; !exists {
			edgeOrder = append(edgeOrder, edgeID)
		}
		edgeMap[edgeID] = graphEdgeRow{
			id:      edgeID,
			fromURN: fromURN,
			toURN:   toURN,
			rel:     rel,
			claimID: "claim:" + edgeID,
			weight:  weight,
		}
	}

	for _, fw := range sb.frameworks {
		jurCode := normalizeISOJurisdiction(fw.Locale)
		jurURN := "urn:xoscal:jurisdiction:" + jurCode
		authSlug := authoritySlug(fw.Provider)
		authURN := "urn:xoscal:authority:" + authSlug

		addStubJSON(jurURN, ontology.ClassJurisdiction, jurLabel{Code: jurCode})
		addStubJSON(authURN, ontology.ClassAuthority, authLabel{Name: fw.Provider, Slug: authSlug})
		addNodeJSON(fw.URN, ontology.ClassFramework, fwLabel{
			ID:           fw.RefID,
			Jurisdiction: jurCode,
			Title:        fw.Name,
			Version:      fw.Version,
		})
		addEdge(fw.URN, ontology.PredInJurisdiction, jurURN, 1.0)
		addEdge(fw.URN, ontology.PredGovernedByAuthority, authURN, 1.0)
	}

	for _, req := range sb.assessableReqs {
		fwURN := "urn:xoscal:framework:" + req.Framework
		addStubJSON(fwURN, ontology.ClassFramework, fwStubLabel{ID: req.Framework})

		isEnh := strings.HasPrefix(req.ParentURN, "urn:xoscal:requirement:") && !strings.Contains(req.ParentURN, ":group:")
		nodeClass := ontology.ClassControl
		if isEnh {
			nodeClass = ontology.ClassControlEnhancement
		}
		domain := ontology.ClassifyThematicDomain(req.NodeRefID, req.Section, req.Title, req.Text)
		addNodeRaw(req.URN, nodeClass, formatCtrlLabelJSON(req.Section, req.Framework, req.NodeRefID, domain, req.Title))
		addEdge(req.URN, ontology.PredBelongsToFramework, fwURN, 1.0)

		domURN := "urn:xoscal:family:" + domain
		addStubJSON(domURN, ontology.ClassControlFamily, domStubLabel{ID: domain, Kind: "thematic_domain"})
		addEdge(req.URN, ontology.PredInControlFamily, domURN, 1.0)

		if isEnh && req.ParentURN != "" {
			addStubJSON(req.ParentURN, ontology.ClassControl, parentStubLabel{Framework: req.Framework})
			addEdge(req.URN, ontology.PredEnhancesControl, req.ParentURN, 1.0)
		}
		for _, p := range req.Params {
			if p.ID == "" {
				continue
			}
			odpURN := "urn:xoscal:odp:" + req.Framework + ":" + p.ID
			addNodeRaw(odpURN, ontology.ClassParameterODP, formatODPLabelJSON(req.Framework, p.ID, p.Label))
			addEdge(req.URN, ontology.PredDefinesODP, odpURN, 1.0)
		}
	}

	var stubsToInsert []graphNodeRow
	for _, urn := range stubOrder {
		if _, inFull := nodeMap[urn]; !inFull {
			if isFreshBulk {
				nodeMap[urn] = stubMap[urn]
				nodeOrder = append(nodeOrder, urn)
			} else {
				stubsToInsert = append(stubsToInsert, stubMap[urn])
			}
		}
	}
	sort.Strings(nodeOrder)
	sort.Strings(edgeOrder)

	const graphBatchSize = 300
	for start := 0; start < len(stubsToInsert); start += graphBatchSize {
		end := start + graphBatchSize
		if end > len(stubsToInsert) {
			end = len(stubsToInsert)
		}
		chunk := stubsToInsert[start:end]
		sqlBuf.Reset()
		sqlBuf.WriteString(`INSERT OR IGNORE INTO kg_graph_nodes(id, kind, urn, labels_json, created_at) VALUES `)
		for i, n := range chunk {
			if i > 0 {
				sqlBuf.WriteByte(',')
			}
			sqlBuf.WriteByte('(')
			appendSQLiteText(&sqlBuf, n.urn)
			sqlBuf.WriteByte(',')
			appendSQLiteText(&sqlBuf, n.kind)
			sqlBuf.WriteByte(',')
			appendSQLiteText(&sqlBuf, n.urn)
			sqlBuf.WriteByte(',')
			appendSQLiteText(&sqlBuf, n.labelsJSON)
			sqlBuf.WriteString(",?1)")
		}
		if _, err := tx.ExecContext(ctx, sqlBuf.String(), now); err != nil {
			return fmt.Errorf("bulk insert stub graph nodes: %w", err)
		}
	}

	const nodePrefix = `INSERT INTO kg_graph_nodes(id, kind, urn, labels_json, created_at) VALUES `
	nodeSuffix := ` ON CONFLICT(id) DO UPDATE SET kind=excluded.kind, urn=excluded.urn, labels_json=excluded.labels_json`
	if isFreshBulk {
		nodeSuffix = ""
	}
	for start := 0; start < len(nodeOrder); start += graphBatchSize {
		end := start + graphBatchSize
		if end > len(nodeOrder) {
			end = len(nodeOrder)
		}
		chunkURNs := nodeOrder[start:end]
		sqlBuf.Reset()
		sqlBuf.Grow(len(chunkURNs)*220 + 256)
		sqlBuf.WriteString(nodePrefix)
		for i, urn := range chunkURNs {
			n := nodeMap[urn]
			if i > 0 {
				sqlBuf.WriteByte(',')
			}
			sqlBuf.WriteByte('(')
			appendSQLiteText(&sqlBuf, n.urn)
			sqlBuf.WriteByte(',')
			appendSQLiteText(&sqlBuf, n.kind)
			sqlBuf.WriteByte(',')
			appendSQLiteText(&sqlBuf, n.urn)
			sqlBuf.WriteByte(',')
			appendSQLiteText(&sqlBuf, n.labelsJSON)
			sqlBuf.WriteString(",?1)")
		}
		sqlBuf.WriteString(nodeSuffix)
		if _, err := tx.ExecContext(ctx, sqlBuf.String(), now); err != nil {
			return fmt.Errorf("bulk upsert graph nodes: %w", err)
		}
	}

	const edgePrefix = `INSERT INTO kg_graph_edges(id, from_node, to_node, relation, qualifier, claim_id, evidence_digest, proof_state_json, valid_from, valid_to, trust_state, weight, extensions_json) VALUES `
	edgeSuffix := ` ON CONFLICT(id) DO UPDATE SET from_node=excluded.from_node, to_node=excluded.to_node, relation=excluded.relation, qualifier=excluded.qualifier, claim_id=excluded.claim_id, evidence_digest=excluded.evidence_digest, proof_state_json=excluded.proof_state_json, valid_from=excluded.valid_from, valid_to=excluded.valid_to, trust_state=excluded.trust_state, weight=excluded.weight, extensions_json=excluded.extensions_json`
	if isFreshBulk {
		edgeSuffix = ""
	}
	for start := 0; start < len(edgeOrder); start += graphBatchSize {
		end := start + graphBatchSize
		if end > len(edgeOrder) {
			end = len(edgeOrder)
		}
		chunkIDs := edgeOrder[start:end]
		sqlBuf.Reset()
		sqlBuf.Grow(len(chunkIDs)*360 + 512)
		sqlBuf.WriteString(edgePrefix)
		for i, id := range chunkIDs {
			e := edgeMap[id]
			if i > 0 {
				sqlBuf.WriteByte(',')
			}
			sqlBuf.WriteByte('(')
			appendSQLiteText(&sqlBuf, e.id)
			sqlBuf.WriteByte(',')
			appendSQLiteText(&sqlBuf, e.fromURN)
			sqlBuf.WriteByte(',')
			appendSQLiteText(&sqlBuf, e.toURN)
			sqlBuf.WriteByte(',')
			appendSQLiteText(&sqlBuf, e.rel)
			sqlBuf.WriteString(",'',")
			appendSQLiteText(&sqlBuf, e.claimID)
			sqlBuf.WriteString(",'sha256:seeded','{}',?1,NULL,'verified',1.0,'{}')")
		}
		sqlBuf.WriteString(edgeSuffix)
		if _, err := tx.ExecContext(ctx, sqlBuf.String(), now); err != nil {
			return fmt.Errorf("bulk upsert graph edges: %w", err)
		}
	}

	if isFreshBulk {
		createIdxSQL := `
CREATE INDEX IF NOT EXISTS idx_kg_type ON kg_entities(entity_type);
CREATE INDEX IF NOT EXISTS idx_kg_status ON kg_entities(status);
CREATE INDEX IF NOT EXISTS idx_kg_valid ON kg_entities(valid_from, valid_to);
CREATE INDEX IF NOT EXISTS idx_nodes_kind ON kg_graph_nodes(kind);
CREATE INDEX IF NOT EXISTS idx_nodes_urn ON kg_graph_nodes(urn);
CREATE INDEX IF NOT EXISTS idx_edges_from ON kg_graph_edges(from_node);
CREATE INDEX IF NOT EXISTS idx_edges_to ON kg_graph_edges(to_node);
CREATE INDEX IF NOT EXISTS idx_edges_relation ON kg_graph_edges(relation);
CREATE INDEX IF NOT EXISTS idx_edges_trust ON kg_graph_edges(trust_state);
CREATE INDEX IF NOT EXISTS idx_edges_valid ON kg_graph_edges(valid_from, valid_to);
`
		if _, err := tx.ExecContext(ctx, createIdxSQL); err != nil {
			return fmt.Errorf("recreate secondary indexes after bulk seed: %w", err)
		}
	}

	return tx.Commit()
}

func normalizeISOJurisdiction(j string) string {
	switch strings.ToLower(strings.TrimSpace(j)) {
	case "ca", "can", "canada":
		return "ca"
	case "us", "usa", "united-states", "united states":
		return "us"
	case "uk", "gb", "gbr", "united-kingdom", "united kingdom":
		return "gb"
	case "fr", "fra", "france":
		return "fr"
	case "de", "deu", "germany":
		return "de"
	case "ee", "est", "estonia":
		return "ee"
	case "eu", "european-union", "european union":
		return "eu"
	case "int", "intl", "international", "global", "iso":
		return "international"
	default:
		return strings.ToLower(strings.TrimSpace(j))
	}
}

func authoritySlug(auth string) string {
	low := strings.ToLower(strings.TrimSpace(auth))
	switch {
	case strings.Contains(low, "canadian centre") || strings.Contains(low, "cccs") || strings.Contains(low, "cse"):
		return "cccs-cse"
	case strings.Contains(low, "ised"):
		return "ised-ca"
	case strings.Contains(low, "nist"):
		return "nist"
	case strings.Contains(low, "cisa"):
		return "cisa"
	case strings.Contains(low, "dod"):
		return "us-dod"
	case strings.Contains(low, "general services") || strings.Contains(low, "gsa"):
		return "us-gsa"
	case strings.Contains(low, "cjis"):
		return "us-cjis"
	case strings.Contains(low, "new york"):
		return "ny-dfs"
	case strings.Contains(low, "ncsc"):
		return "ncsc-uk"
	case strings.Contains(low, "anssi"):
		return "anssi"
	case strings.Contains(low, "cnil"):
		return "cnil"
	case strings.Contains(low, "aviation"):
		return "dsac-fr"
	case strings.Contains(low, "bsi"):
		return "bsi"
	case strings.Contains(low, "ria"):
		return "ria"
	case strings.Contains(low, "enisa") || strings.Contains(low, "european commission"):
		return "enisa"
	case strings.Contains(low, "iso"):
		return "iso"
	case low == "eu":
		return "eu-commission"
	default:
		s := strings.ReplaceAll(low, " ", "-")
		s = strings.ReplaceAll(s, "/", "-")
		if s == "" {
			return "authority-default"
		}
		return s
	}
}

func upsertEntity(ctx context.Context, store kg.Store, e *kg.Entity) error {
	existing, err := store.GetEntity(ctx, e.URN)
	if err == nil && existing != nil {
		return store.UpdateEntity(ctx, e)
	}
	return store.CreateEntity(ctx, e)
}
