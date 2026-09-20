package changeanalysis

import (
	"sort"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// ImpactEdgeは、変更されたsubjectから影響を受けるsubjectへの、exact identityに基づく依存を一本示す。
// 名前の類似や物理列の一致からは辺を作らない。
type ImpactEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Via    string `json:"via"`
	Reason string `json:"reason"`
}

// Unresolvedは、解析が判断できなかった事項を成功や「影響なし」へ読み替えずに残す。
type Unresolved struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func subjectKey(kind string, id core.CanonicalID) string { return kind + ":" + string(id) }

// Impactは、追加・変更されたValidationから、それがconstrainするAttribute、所有Data、
// そのDataを指すRelation、Application Pattern、namespaceを利用するApplicationへの辺を返す。
// 解析できなかった範囲はUnresolvedとして返し、実行系（Driver / Target）への影響は主張しない。
func Impact(after ir.World, changes []SubjectChange) ([]ImpactEdge, []Unresolved) {
	var edges []ImpactEdge
	unresolvedApplications := map[string]bool{}

	for _, change := range changes {
		if change.Kind != "validation" {
			continue
		}
		rule, namespace, ok := findValidation(after, core.CanonicalID(change.ID))
		if !ok {
			continue
		}
		validationKey := subjectKey("validation", rule.ID)
		attributeKey := subjectKey("attribute", rule.Attribute)
		edges = append(edges, ImpactEdge{
			From: validationKey, To: attributeKey, Via: "semantic.validation.attribute",
			Reason: "Validationが制約対象として宣言したAttributeである",
		})

		if rule.Relation != "" {
			edges = append(edges, ImpactEdge{
				From: validationKey, To: subjectKey("relation", rule.Relation), Via: "semantic.validation.relation",
				Reason: "relation-unique ValidationがRelation scopeとして宣言したRelationである",
			})
		}

		owner, hasOwner := ownerData(after, rule.Attribute)
		relationIDs := map[core.CanonicalID]bool{}
		if rule.Relation != "" {
			relationIDs[rule.Relation] = true
		}
		if hasOwner {
			edges = append(edges, ImpactEdge{
				From: attributeKey, To: subjectKey("data", owner), Via: "semantic.data.attribute",
				Reason: "AttributeはこのDataが宣言している",
			})
			for _, ns := range after.Semantics {
				for _, d := range ns.Data {
					for _, r := range d.Relations {
						if r.Target != owner {
							continue
						}
						relationIDs[r.ID] = true
						edges = append(edges, ImpactEdge{
							From: subjectKey("data", owner), To: subjectKey("relation", r.ID), Via: "semantic.relation.target",
							Reason: "Relationのtargetがこのデータである",
						})
					}
				}
			}
		}

		for _, app := range after.Applications {
			for _, p := range app.Patterns {
				patternKey := subjectKey("pattern", p.ID)
				if hasOwner && p.Data == owner {
					edges = append(edges, ImpactEdge{
						From: subjectKey("data", owner), To: patternKey, Via: "application.pattern.data",
						Reason: "Application PatternがこのDataを明示的に参照している",
					})
				}
				if p.Relation != "" && relationIDs[p.Relation] {
					edges = append(edges, ImpactEdge{
						From: subjectKey("relation", p.Relation), To: patternKey, Via: "application.pattern.relation",
						Reason: "Application PatternがこのRelationを明示的に参照している",
					})
				}
			}
			usesNamespace := false
			for _, used := range app.SemanticUses {
				if used == namespace {
					usesNamespace = true
				}
			}
			if !usesNamespace {
				continue
			}
			edges = append(edges, ImpactEdge{
				From: validationKey, To: subjectKey("application", app.ID), Via: "application.uses-semantic-namespace",
				Reason: "ApplicationがValidationの属するsemantic namespaceを利用宣言している",
			})
			declared := false
			for _, id := range app.Validations {
				if id == rule.ID {
					declared = true
				}
			}
			if !declared {
				unresolvedApplications[string(app.ID)] = true
			}
		}
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		return edges[i].Via < edges[j].Via
	})
	edges = dedupeEdges(edges)

	var unresolved []Unresolved
	apps := make([]string, 0, len(unresolvedApplications))
	for app := range unresolvedApplications {
		apps = append(apps, app)
	}
	sort.Strings(apps)
	for _, app := range apps {
		unresolved = append(unresolved, Unresolved{
			Code: "APPLICATION_VALIDATION_SELECTION_NOT_CANONICAL",
			Message: "application " + app + " はこのValidationを`uses validation`で宣言していない。現行のDriver / Canonical modelは、" +
				"Applicationが評価するValidationの選択を宣言から導出しておらず（Reference scenarioはrule IDを呼び出し側で選ぶ）、" +
				"この変更だけでは実行時に評価されるとは言えない",
		})
	}
	unresolved = append(unresolved,
		Unresolved{Code: "DRIVER_SCENARIO_NOT_RUN", Message: "解析はDriver scenarioを実行しない。実行結果への影響はこの解析の根拠にしていない"},
		Unresolved{Code: "TARGET_PROJECTION_NOT_EVALUATED", Message: "解析はWeb Target Projectionを評価しない。World直下の他collectionが不変であること以上は主張しない"},
	)
	return edges, unresolved
}

func dedupeEdges(edges []ImpactEdge) []ImpactEdge {
	var out []ImpactEdge
	for i, e := range edges {
		if i > 0 && e == edges[i-1] {
			continue
		}
		out = append(out, e)
	}
	return out
}

func findValidation(w ir.World, id core.CanonicalID) (semantic.Validation, core.CanonicalID, bool) {
	for _, ns := range w.Semantics {
		for _, v := range ns.Validations {
			if v.ID == id {
				return v, ns.ID, true
			}
		}
	}
	return semantic.Validation{}, "", false
}

func ownerData(w ir.World, attribute core.CanonicalID) (core.CanonicalID, bool) {
	for _, ns := range w.Semantics {
		for _, d := range ns.Data {
			for _, a := range d.Attributes {
				if a.ID == attribute {
					return d.ID, true
				}
			}
		}
	}
	return "", false
}
