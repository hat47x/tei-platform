package applicationview

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// SchemaVersionは、この投影のJSON契約の版を表す。
const SchemaVersion = "tei.applicationview/v0"

// Viewは、Application Patternの対話契約を、表示側が使える形に解決した投影である。
// 意味を複製せず、PatternとSemanticのidentityを型付きで参照する。
type View struct {
	SchemaVersion string                  `json:"schemaVersion"`
	Application   core.CanonicalID        `json:"application"`
	Pattern       core.CanonicalID        `json:"pattern"`
	Kind          application.PatternKind `json:"kind"`
	Data          core.CanonicalID        `json:"data,omitempty"`
	Items         []Item                  `json:"items,omitempty"`
	Navigations   []Navigation            `json:"navigations,omitempty"`
}

// Itemは、対話に現れる項目と、その型・表示名・入力部品・意味の役割・参照先を表す。
type Item struct {
	Attribute core.CanonicalID `json:"attribute"`
	Type      core.CanonicalID `json:"type,omitempty"`
	Label     string           `json:"label,omitempty"`
	Input     string           `json:"input,omitempty"`
	Required  bool             `json:"required,omitempty"`
	Immutable bool             `json:"immutable,omitempty"`
	Values    []string         `json:"values,omitempty"`
	Lookup    *Lookup          `json:"lookup,omitempty"`
}

// Lookupは、項目の値を参照Patternから得る束縛を解決した結果である。
type Lookup struct {
	Pattern         core.CanonicalID        `json:"pattern"`
	Kind            application.PatternKind `json:"kind"`
	TargetAttribute core.CanonicalID        `json:"targetAttribute"`
	TargetType      core.CanonicalID        `json:"targetType,omitempty"`
}

// Navigationは、このPatternへ入る遷移元を解決した結果である。
type Navigation struct {
	From core.CanonicalID        `json:"from"`
	Kind application.PatternKind `json:"kind"`
	Data core.CanonicalID        `json:"data,omitempty"`
}

// Buildは、Application Patternのinteractionを、Semanticの型とPatternの種別で解決する。
// 参照が欠けている場合は推測で補わず、エラーを返す。
func Build(world ir.World, applicationID, patternID core.CanonicalID) (View, error) {
	var app *application.Application
	for i := range world.Applications {
		if world.Applications[i].ID == applicationID {
			app = &world.Applications[i]
			break
		}
	}
	if app == nil {
		return View{}, fmt.Errorf("application %sが見つかりません", applicationID)
	}

	patterns := map[core.CanonicalID]application.Pattern{}
	dataAttributes := map[core.CanonicalID]map[core.CanonicalID]semantic.Attribute{}
	relations := map[core.CanonicalID]semantic.Relation{}
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
			attributes := map[core.CanonicalID]semantic.Attribute{}
			for _, attribute := range data.Attributes {
				attributes[attribute.ID] = attribute
			}
			dataAttributes[data.ID] = attributes
			for _, relation := range data.Relations {
				if _, exists := relations[relation.ID]; !exists {
					relations[relation.ID] = relation
				}
			}
		}
	}

	pattern, ok := patterns[patternID]
	if !ok {
		return View{}, fmt.Errorf("application pattern %sが見つかりません", patternID)
	}
	if pattern.Interaction == nil {
		return View{}, fmt.Errorf("application pattern %sはinteractionを持ちません", patternID)
	}
	if pattern.Data == "" {
		return View{}, fmt.Errorf("application pattern %sはdataを持ちません", patternID)
	}
	attributes, ok := dataAttributes[pattern.Data]
	if !ok {
		return View{}, fmt.Errorf("semantic data %sが見つかりません", pattern.Data)
	}

	view := View{
		SchemaVersion: SchemaVersion,
		Application:   app.ID,
		Pattern:       pattern.ID,
		Kind:          pattern.Kind,
		Data:          pattern.Data,
	}
	for _, item := range pattern.Interaction.Items {
		attribute, ok := attributes[item.Attribute]
		if !ok {
			return View{}, fmt.Errorf("interaction item %sは%sのattributeではありません", item.Attribute, pattern.Data)
		}
		resolved := Item{
			Attribute: item.Attribute,
			Type:      attribute.TypeRef,
			Label:     labelOf(attribute),
			Input:     inputKind(attribute.TypeRef),
			Required:  attribute.Required,
			Immutable: attribute.Immutable,
			Values:    typeValues[attribute.TypeRef],
		}
		if item.Lookup != nil {
			target, ok := patterns[item.Lookup.Pattern]
			if !ok {
				return View{}, fmt.Errorf("lookup pattern %sが見つかりません", item.Lookup.Pattern)
			}
			if target.Kind != application.PatternReference {
				return View{}, fmt.Errorf("lookup pattern %sはreferenceではありません: %s", target.ID, target.Kind)
			}
			relation, ok := relations[target.Relation]
			if !ok {
				return View{}, fmt.Errorf("reference pattern %sのrelation %sが見つかりません", target.ID, target.Relation)
			}
			targetAttributes, ok := dataAttributes[relation.Target]
			if !ok {
				return View{}, fmt.Errorf("relation target %sが見つかりません", relation.Target)
			}
			targetAttribute, ok := targetAttributes[item.Lookup.TargetAttribute]
			if !ok {
				return View{}, fmt.Errorf("lookup target %sは%sのattributeではありません", item.Lookup.TargetAttribute, relation.Target)
			}
			resolved.Lookup = &Lookup{
				Pattern:         target.ID,
				Kind:            target.Kind,
				TargetAttribute: item.Lookup.TargetAttribute,
				TargetType:      targetAttribute.TypeRef,
			}
		}
		view.Items = append(view.Items, resolved)
	}
	for _, navigation := range pattern.Interaction.Navigations {
		source, ok := patterns[navigation.From]
		if !ok {
			return View{}, fmt.Errorf("navigation source %sが見つかりません", navigation.From)
		}
		if source.Kind != application.PatternBrowse {
			return View{}, fmt.Errorf("navigation source %sはbrowseではありません: %s", source.ID, source.Kind)
		}
		view.Navigations = append(view.Navigations, Navigation{
			From: source.ID,
			Kind: source.Kind,
			Data: source.Data,
		})
	}
	return view, nil
}

func labelOf(attribute semantic.Attribute) string {
	if attribute.Display != "" {
		return attribute.Display
	}
	if attribute.Symbol != "" {
		return attribute.Symbol
	}
	return string(attribute.ID)
}

func inputKind(typeRef core.CanonicalID) string {
	switch typeRef {
	case "std.integer", "std.decimal", "std.money":
		return "number"
	case "std.date":
		return "date"
	case "std.datetime":
		return "datetime-local"
	case "std.time":
		return "time"
	case "std.boolean":
		return "checkbox"
	default:
		return "text"
	}
}
