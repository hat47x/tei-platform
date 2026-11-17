package applicationview

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// detailPlaceholderRowsは、印刷面の明細に置く合成のプレースホルダ行数である。
// 実際の明細行はこのビューへ来ていないため、行数はデータから決まらない。
// したがって固定する。以前はpage-breakのRowsへ合わせていたため、
// 明細は常にちょうど1ページ分になり、改ページも見出しの繰返しも投影へ
// 現れなかった。行数をRowsから切り離せば、Rowsは「どう分けるか」だけを決め、
// 分割しても行は落ちない。
const detailPlaceholderRows = 5

// HeaderDetailViewは、header-detail PatternのRelationを、見出し部と明細部に解決した投影である。
// 項目はPatternのinteractionではなく、Relationのsource Dataとtarget Dataの属性から取る。
type HeaderDetailView struct {
	SchemaVersion       string                  `json:"schemaVersion"`
	Application         core.CanonicalID        `json:"application"`
	Pattern             core.CanonicalID        `json:"pattern"`
	Kind                application.PatternKind `json:"kind"`
	Relation            core.CanonicalID        `json:"relation"`
	Header              core.CanonicalID        `json:"header"`
	Detail              core.CanonicalID        `json:"detail"`
	HeaderItems         []Item                  `json:"headerItems,omitempty"`
	DetailColumns       []Item                  `json:"detailColumns,omitempty"`
	Aggregates          []AggregateRef          `json:"aggregates,omitempty"`
	PageBreak           *PageBreakRef           `json:"pageBreak,omitempty"`
	PrintTemplate       string                  `json:"printTemplate,omitempty"`
	DetailRows          []int                   `json:"-"`
	Packages            []PackageRef            `json:"packages,omitempty"`
	SemanticFingerprint string                  `json:"semanticFingerprint,omitempty"`
}

// PageBreakRefは、header-detailの印刷の改ページを解決した結果である。
type PageBreakRef struct {
	Rows         int  `json:"rows"`
	RepeatHeader bool `json:"repeatHeader,omitempty"`
}

// DetailPageは、印刷面の1ページ分に置く明細行である。
// Numberは1から始まる面の順序、RowsはDetailRowsの部分列であり、
// RepeatHeaderはその面が見出し行を描くかを決める。
type DetailPage struct {
	Number       int
	Rows         []int
	RepeatHeader bool
}

// DetailPagesは、印刷面の明細行を、宣言した改ページの大きさで面へ分ける。
//
// 面の大きさはpage-breakのRowsであり、分割はその宣言がある場合に限る。
// 宣言が無い、または明細行が1面に収まる場合は、分割しない1面を返す。
// 見出しの繰返しはrepeat-headerが決める。1面目は常に見出しを描き、
// 2面目以降はrepeat-headerがある場合にだけ描く。宣言が無いのに
// 2面目へ見出しを描けば、繰り返さない宣言が繰り返して見える。
// 分割しても行は落とさない。最後の面は端数を持つ。
func (view HeaderDetailView) DetailPages() []DetailPage {
	capacity := 0
	if view.PageBreak != nil && view.PageBreak.Rows > 0 {
		capacity = view.PageBreak.Rows
	}
	if capacity <= 0 || len(view.DetailRows) <= capacity {
		return []DetailPage{{Number: 1, Rows: view.DetailRows, RepeatHeader: true}}
	}
	repeat := view.PageBreak != nil && view.PageBreak.RepeatHeader
	pages := make([]DetailPage, 0, (len(view.DetailRows)+capacity-1)/capacity)
	for start := 0; start < len(view.DetailRows); start += capacity {
		end := start + capacity
		if end > len(view.DetailRows) {
			end = len(view.DetailRows)
		}
		number := len(pages) + 1
		pages = append(pages, DetailPage{
			Number:       number,
			Rows:         view.DetailRows[start:end],
			RepeatHeader: number == 1 || repeat,
		})
	}
	return pages
}

// AggregateRefは、header-detailの明細に対する集計の宣言を解決した結果である。
type AggregateRef struct {
	Name      string           `json:"name"`
	Function  string           `json:"function"`
	Attribute core.CanonicalID `json:"attribute"`
	Label     string           `json:"label,omitempty"`
	Type      core.CanonicalID `json:"type,omitempty"`
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
	fingerprint, err := fingerprintForApp(world, *app)
	if err != nil {
		return HeaderDetailView{}, err
	}
	view.SemanticFingerprint = fingerprint
	view.PrintTemplate = pattern.PrintTemplate
	if pattern.PageBreak != nil {
		view.PageBreak = &PageBreakRef{Rows: pattern.PageBreak.Rows, RepeatHeader: pattern.PageBreak.RepeatHeader}
	}
	// 明細行は実データではなく合成のプレースホルダである。その個数は
	// 改ページの大きさとは独立に固定する。以前は個数をpage-breakのRowsへ
	// 合わせていたため、明細は常にちょうど1ページ分になり、分割も
	// 見出しの繰返しも投影に現れなかった。個数を固定すれば、Rowsは
	// 「どう分けるか」だけを決め、行を失わない。
	for i := 0; i < detailPlaceholderRows; i++ {
		view.DetailRows = append(view.DetailRows, i)
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
	detailByID := map[core.CanonicalID]semantic.Attribute{}
	for _, attribute := range detailAttributes {
		detailByID[attribute.ID] = attribute
	}
	for _, aggregate := range pattern.Aggregates {
		attribute, ok := detailByID[aggregate.Attribute]
		if !ok {
			return HeaderDetailView{}, fmt.Errorf("aggregate %s refers to %s that is not an attribute of %s", aggregate.Name, aggregate.Attribute, relation.Target)
		}
		view.Aggregates = append(view.Aggregates, AggregateRef{
			Name:      aggregate.Name,
			Function:  aggregate.Function,
			Attribute: aggregate.Attribute,
			Label:     labelOf(attribute),
			Type:      attribute.TypeRef,
		})
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
