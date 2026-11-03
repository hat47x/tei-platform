package applicationview

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
)

// BrowseViewは、browse PatternのQuery結果を、一覧の列として解決した投影である。
type BrowseView struct {
	SchemaVersion       string                  `json:"schemaVersion"`
	Application         core.CanonicalID        `json:"application"`
	Pattern             core.CanonicalID        `json:"pattern"`
	Kind                application.PatternKind `json:"kind"`
	Data                core.CanonicalID        `json:"data,omitempty"`
	Query               core.CanonicalID        `json:"query,omitempty"`
	Columns             []Item                  `json:"columns,omitempty"`
	Filters             []FilterRef             `json:"filters,omitempty"`
	Sorts               []SortRef               `json:"sorts,omitempty"`
	PageSize            int                     `json:"pageSize,omitempty"`
	PrintTemplate       string                  `json:"printTemplate,omitempty"`
	Packages            []PackageRef            `json:"packages,omitempty"`
	SemanticFingerprint string                  `json:"semanticFingerprint,omitempty"`
}

// SortRefは、一覧の並び順を解決した結果である。
type SortRef struct {
	Attribute core.CanonicalID `json:"attribute"`
	Label     string           `json:"label,omitempty"`
	Direction string           `json:"direction"`
}

// FilterRefは、一覧の絞り込みに使う項目と演算子を解決した結果である。
type FilterRef struct {
	Attribute core.CanonicalID `json:"attribute"`
	Label     string           `json:"label,omitempty"`
	Type      core.CanonicalID `json:"type,omitempty"`
	Input     string           `json:"input,omitempty"`
	Parameter string           `json:"parameter,omitempty"`
	Operators []string         `json:"operators,omitempty"`
}

// BuildBrowseは、browse PatternのQuery結果の列を、Semanticの型と表示名で解決する。
// Query結果の対応が欠けている場合は推測で補わない。
func BuildBrowse(world ir.World, applicationID, patternID core.CanonicalID) (BrowseView, error) {
	var app *application.Application
	for i := range world.Applications {
		if world.Applications[i].ID == applicationID {
			app = &world.Applications[i]
			break
		}
	}
	if app == nil {
		return BrowseView{}, fmt.Errorf("application %sが見つかりません", applicationID)
	}
	var pattern *application.Pattern
	for i := range app.Patterns {
		if app.Patterns[i].ID == patternID {
			pattern = &app.Patterns[i]
			break
		}
	}
	if pattern == nil {
		return BrowseView{}, fmt.Errorf("application pattern %sが見つかりません", patternID)
	}
	if pattern.Kind != application.PatternBrowse {
		return BrowseView{}, fmt.Errorf("application pattern %sはbrowseではありません: %s", pattern.ID, pattern.Kind)
	}
	if pattern.Query == "" {
		return BrowseView{}, fmt.Errorf("browse pattern %sはqueryを持ちません", pattern.ID)
	}

	var embedded *query.EmbeddedSQL
	for i := range world.Queries {
		if world.Queries[i].ID == pattern.Query {
			embedded = &world.Queries[i]
			break
		}
	}
	if embedded == nil {
		return BrowseView{}, fmt.Errorf("query %sが見つかりません", pattern.Query)
	}
	if embedded.Application != app.ID {
		return BrowseView{}, fmt.Errorf("query %sはapplication %sのものではありません", pattern.Query, app.ID)
	}
	if len(embedded.ResultColumns) == 0 {
		return BrowseView{}, fmt.Errorf("query %sは結果列を持ちません", pattern.Query)
	}

	attributes := map[core.CanonicalID]semantic.Attribute{}
	typeValues := map[core.CanonicalID][]string{}
	for _, namespace := range world.Semantics {
		for _, typ := range namespace.Types {
			if len(typ.Values) > 0 {
				typeValues[typ.ID] = typ.Values
			}
		}
		for _, data := range namespace.Data {
			for _, attribute := range data.Attributes {
				attributes[attribute.ID] = attribute
			}
		}
	}

	view := BrowseView{
		SchemaVersion: SchemaVersion,
		Application:   app.ID,
		Pattern:       pattern.ID,
		Kind:          pattern.Kind,
		Data:          pattern.Data,
		Query:         pattern.Query,
		PageSize:      pattern.PageSize,
		PrintTemplate: pattern.PrintTemplate,
	}
	fingerprint, err := fingerprintForApp(world, *app)
	if err != nil {
		return BrowseView{}, err
	}
	view.SemanticFingerprint = fingerprint
	for _, pkg := range app.Packages {
		for _, member := range pkg.Members {
			if member.Pattern == pattern.ID {
				view.Packages = append(view.Packages, PackageRef{ID: pkg.ID, Version: pkg.Version})
				break
			}
		}
	}
	for _, column := range embedded.ResultColumns {
		attribute, ok := attributes[column.Attribute]
		if !ok {
			return BrowseView{}, fmt.Errorf("query %s result column %s refers to unknown attribute %s", pattern.Query, column.Column, column.Attribute)
		}
		item := itemFor(attribute, typeValues)
		view.Columns = append(view.Columns, item)
	}
	for _, sort := range pattern.Sorts {
		attribute, ok := attributes[sort.Attribute]
		if !ok {
			return BrowseView{}, fmt.Errorf("browse pattern %s sort refers to unknown attribute %s", pattern.ID, sort.Attribute)
		}
		view.Sorts = append(view.Sorts, SortRef{Attribute: sort.Attribute, Label: labelOf(attribute), Direction: sort.Direction})
	}
	for _, filter := range pattern.Filters {
		attribute, ok := attributes[filter.Attribute]
		if !ok {
			return BrowseView{}, fmt.Errorf("browse pattern %s filter refers to unknown attribute %s", pattern.ID, filter.Attribute)
		}
		view.Filters = append(view.Filters, FilterRef{
			Attribute: filter.Attribute,
			Label:     labelOf(attribute),
			Type:      attribute.TypeRef,
			Input:     inputKind(attribute.TypeRef),
			Parameter: filter.Parameter,
			Operators: filter.Operators,
		})
	}
	return view, nil
}
