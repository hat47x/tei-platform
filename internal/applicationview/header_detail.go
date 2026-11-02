package applicationview

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// HeaderDetailViewは、header-detail PatternのRelationを、見出し部と明細部に解決した投影である。
// 項目はPatternのinteractionではなく、Relationのsource Dataとtarget Dataの属性から取る。
type HeaderDetailView struct {
	SchemaVersion string                  `json:"schemaVersion"`
	Application   core.CanonicalID        `json:"application"`
	Pattern       core.CanonicalID        `json:"pattern"`
	Kind          application.PatternKind `json:"kind"`
	Relation      core.CanonicalID        `json:"relation"`
	Header        core.CanonicalID        `json:"header"`
	Detail        core.CanonicalID        `json:"detail"`
	HeaderItems   []Item                  `json:"headerItems,omitempty"`
	DetailColumns []Item                  `json:"detailColumns,omitempty"`
	Packages      []PackageRef            `json:"packages,omitempty"`
}

// PatternKindは、Application Patternの種別だけを返す。viewコマンドの分岐に使う。
func PatternKind(world ir.World, applicationID, patternID core.CanonicalID) (application.PatternKind, error) {
	var app *application.Application
	for i := range world.Applications {
		if world.Applications[i].ID == applicationID {
			app = &world.Applications[i]
			break
		}
	}
	if app == nil {
		return "", fmt.Errorf("application %sが見つかりません", applicationID)
	}
	for _, pattern := range app.Patterns {
		if pattern.ID == patternID {
			return pattern.Kind, nil
		}
	}
	return "", fmt.Errorf("application pattern %sが見つかりません", patternID)
}

// BuildHeaderDetailは、header-detail PatternのRelationから、見出し部と明細部の項目を解決する。
func BuildHeaderDetail(world ir.World, applicationID, patternID core.CanonicalID) (HeaderDetailView, error) {
	var app *application.Application
	for i := range world.Applications {
		if world.Applications[i].ID == applicationID {
			app = &world.Applications[i]
			break
		}
	}
	if app == nil {
		return HeaderDetailView{}, fmt.Errorf("application %sが見つかりません", applicationID)
	}

	patterns := map[core.CanonicalID]application.Pattern{}
	relations := map[core.CanonicalID]semantic.Relation{}
	relationSources := map[core.CanonicalID]core.CanonicalID{}
	dataAttributes := map[core.CanonicalID][]semantic.Attribute{}
	typeValues := map[core.CanonicalID][]string{}
	for _, candidate := range world.Applications {
		for _, pattern := range candidate.Patterns {
			if _, exists := patterns[pattern.ID]; !exists {
				patterns[pattern.ID] = pattern
			}
		}
	}
	for _, namespace := range world.Semantics {
		for _, typ := range namespace.Types {
			if len(typ.Values) > 0 {
				typeValues[typ.ID] = typ.Values
			}
		}
		for _, data := range namespace.Data {
			dataAttributes[data.ID] = data.Attributes
			for _, relation := range data.Relations {
				if _, exists := relations[relation.ID]; !exists {
					relations[relation.ID] = relation
					relationSources[relation.ID] = data.ID
				}
			}
		}
	}

	pattern, ok := patterns[patternID]
	if !ok {
		return HeaderDetailView{}, fmt.Errorf("application pattern %sが見つかりません", patternID)
	}
	if pattern.Kind != application.PatternHeaderDetail {
		return HeaderDetailView{}, fmt.Errorf("application pattern %sはheader-detailではありません: %s", pattern.ID, pattern.Kind)
	}
	relation, ok := relations[pattern.Relation]
	if !ok {
		return HeaderDetailView{}, fmt.Errorf("header-detail pattern %sのrelation %sが見つかりません", pattern.ID, pattern.Relation)
	}
	header := relationSources[relation.ID]
	headerAttributes, ok := dataAttributes[header]
	if !ok {
		return HeaderDetailView{}, fmt.Errorf("relation source for %sが見つかりません", relation.ID)
	}
	detailAttributes, ok := dataAttributes[relation.Target]
	if !ok {
		return HeaderDetailView{}, fmt.Errorf("relation target %sが見つかりません", relation.Target)
	}

	view := HeaderDetailView{
		SchemaVersion: SchemaVersion,
		Application:   app.ID,
		Pattern:       pattern.ID,
		Kind:          pattern.Kind,
		Relation:      relation.ID,
		Header:        header,
		Detail:        relation.Target,
	}
	for _, pkg := range app.Packages {
		for _, member := range pkg.Members {
			if member.Pattern != pattern.ID {
				continue
			}
			view.Packages = append(view.Packages, PackageRef{ID: pkg.ID, Version: pkg.Version})
			break
		}
	}
	for _, attribute := range headerAttributes {
		view.HeaderItems = append(view.HeaderItems, itemFor(attribute, typeValues))
	}
	for _, attribute := range detailAttributes {
		view.DetailColumns = append(view.DetailColumns, itemFor(attribute, typeValues))
	}
	return view, nil
}

func itemFor(attribute semantic.Attribute, typeValues map[core.CanonicalID][]string) Item {
	return Item{
		Attribute: attribute.ID,
		Type:      attribute.TypeRef,
		Label:     labelOf(attribute),
		Input:     inputKind(attribute.TypeRef),
		Required:  attribute.Required,
		Immutable: attribute.Immutable,
		Values:    typeValues[attribute.TypeRef],
	}
}
