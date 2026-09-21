package changeanalysis

import (
	"sort"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// ImpactEdgeは、変更されたsubjectと、exact identityで結ばれたsubjectとの依存を一本示す。
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

// Impactは、追加・変更されたsubjectから、宣言されたexact identityの参照を辿った辺を返す。
//
//   - validation: 制約対象のAttribute、所有Data、そのDataを指すRelation、そのRelationを使うPattern、namespaceを利用するApplication
//   - relation: target Data、key Attribute、そのRelationを使うPattern
//   - attribute: 所有Data
//   - namespace: それを利用するApplication
//   - application: 利用するsemantic namespace、要求するDatabase・Contract
//   - application-pattern: owner Application、Relation、Contract Query、入力Attribute
//
// 解析できなかった範囲はUnresolvedとして返し、実行系（Driver / Target / Environment）への影響は主張しない。
func Impact(after ir.World, changes []SubjectChange) ([]ImpactEdge, []Unresolved) {
	var edges []ImpactEdge
	undeclaredValidation := map[string]bool{}
	applicationAdded := map[string]bool{}

	for _, change := range changes {
		if change.Change == "removed" {
			continue
		}
		id := core.CanonicalID(change.ID)
		switch change.Kind {
		case "validation":
			es, apps := validationEdges(after, id)
			edges = append(edges, es...)
			for _, app := range apps {
				undeclaredValidation[app] = true
			}
		case "relation":
			edges = append(edges, relationEdges(after, id)...)
		case "attribute":
			if owner, ok := ownerData(after, id); ok {
				edges = append(edges, ImpactEdge{
					From: subjectKey("attribute", id), To: subjectKey("data", owner), Via: "semantic.data.attribute",
					Reason: "AttributeはこのDataが宣言している",
				})
			}
		case "namespace":
			for _, app := range after.Applications {
				if usesNamespace(app, id) {
					edges = append(edges, ImpactEdge{
						From: subjectKey("namespace", id), To: subjectKey("application", app.ID), Via: "application.uses-semantic-namespace",
						Reason: "ApplicationがこのsemanticをNamespaceとして利用宣言している",
					})
				}
			}
		case "application":
			if app, ok := findApplication(after, id); ok {
				edges = append(edges, applicationEdges(app)...)
				if change.Change == "added" {
					applicationAdded[string(id)] = true
				}
			}
		case "application-pattern":
			if app, pattern, ok := findPattern(after, id); ok {
				edges = append(edges, patternEdges(app, pattern)...)
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
	for _, app := range sortedKeys(undeclaredValidation) {
		unresolved = append(unresolved, Unresolved{
			Code: "APPLICATION_VALIDATION_SELECTION_NOT_CANONICAL",
			Message: "application " + app + " はこのValidationを`uses validation`で宣言していない。現行のDriver / Canonical modelは、" +
				"Applicationが評価するValidationの選択を宣言から導出しておらず（Reference scenarioはrule IDを呼び出し側で選ぶ）、" +
				"この変更だけでは実行時に評価されるとは言えない",
		})
	}
	for _, app := range sortedKeys(applicationAdded) {
		unresolved = append(unresolved, Unresolved{
			Code: "ENVIRONMENT_BINDING_NOT_AUTHORED",
			Message: "application " + app + " はこの変更で追加されたが、Environment instanceへのbinding・Provider・Permission grantは含まれない。" +
				"実行可能であるとは言えず、Environment / Runtime authorityはこの変更では変わらない",
		})
	}
	unresolved = append(unresolved,
		Unresolved{Code: "DRIVER_SCENARIO_NOT_RUN", Message: "解析はDriver scenarioを実行しない。実行結果への影響はこの解析の根拠にしていない"},
		Unresolved{Code: "TARGET_PROJECTION_NOT_EVALUATED", Message: "解析はWeb Target Projectionを評価しない。World直下の他collectionが不変であること以上は主張しない"},
	)
	return edges, unresolved
}

// validationEdgesは、Validationから辿れる辺と、それを`uses validation`で宣言していないApplicationを返す。
func validationEdges(after ir.World, id core.CanonicalID) ([]ImpactEdge, []string) {
	rule, namespace, ok := findValidation(after, id)
	if !ok {
		return nil, nil
	}
	var edges []ImpactEdge
	var undeclared []string
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
			patternKey := subjectKey("application-pattern", p.ID)
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
		if !usesNamespace(app, namespace) {
			continue
		}
		edges = append(edges, ImpactEdge{
			From: validationKey, To: subjectKey("application", app.ID), Via: "application.uses-semantic-namespace",
			Reason: "ApplicationがValidationの属するsemantic namespaceを利用宣言している",
		})
		declared := false
		for _, declaredID := range app.Validations {
			if declaredID == rule.ID {
				declared = true
			}
		}
		if !declared {
			undeclared = append(undeclared, string(app.ID))
		}
	}
	return edges, undeclared
}

func relationEdges(after ir.World, id core.CanonicalID) []ImpactEdge {
	var edges []ImpactEdge
	for _, ns := range after.Semantics {
		for _, d := range ns.Data {
			for _, r := range d.Relations {
				if r.ID != id {
					continue
				}
				relationKey := subjectKey("relation", r.ID)
				edges = append(edges, ImpactEdge{
					From: relationKey, To: subjectKey("data", r.Target), Via: "semantic.relation.target",
					Reason: "Relationが指すtarget Dataである",
				})
				for _, k := range r.Keys {
					edges = append(edges,
						ImpactEdge{From: relationKey, To: subjectKey("attribute", k.SourceAttribute), Via: "semantic.relation.key.source", Reason: "Relation keyのsource側Attributeである"},
						ImpactEdge{From: relationKey, To: subjectKey("attribute", k.TargetAttribute), Via: "semantic.relation.key.target", Reason: "Relation keyのtarget側Attributeである"},
					)
				}
			}
		}
	}
	for _, app := range after.Applications {
		for _, p := range app.Patterns {
			if p.Relation == id {
				edges = append(edges, ImpactEdge{
					From: subjectKey("relation", id), To: subjectKey("application-pattern", p.ID), Via: "application.pattern.relation",
					Reason: "Application PatternがこのRelationを明示的に参照している",
				})
			}
		}
	}
	return edges
}

func applicationEdges(app application.Application) []ImpactEdge {
	var edges []ImpactEdge
	key := subjectKey("application", app.ID)
	for _, ns := range app.SemanticUses {
		edges = append(edges, ImpactEdge{
			From: key, To: subjectKey("namespace", ns), Via: "application.uses-semantic-namespace",
			Reason: "Applicationが利用宣言しているsemantic namespaceである",
		})
	}
	for _, db := range app.Databases {
		edges = append(edges, ImpactEdge{
			From: key, To: "database:" + string(db.TargetDatabase), Via: "application.database.target",
			Reason: "Applicationがalias " + string(db.Alias) + " で要求しているDatabaseである",
		})
		for _, c := range db.RequiredContracts {
			edges = append(edges, ImpactEdge{
				From: key, To: subjectKey("contract", c.ContractID), Via: "application.database.requires-contract",
				Reason: "Applicationがalias " + string(db.Alias) + " で要求しているDatabase Contractである",
			})
		}
	}
	return edges
}

func patternEdges(app application.Application, p application.Pattern) []ImpactEdge {
	var edges []ImpactEdge
	key := subjectKey("application-pattern", p.ID)
	edges = append(edges, ImpactEdge{
		From: key, To: subjectKey("application", app.ID), Via: "application.pattern.owner",
		Reason: "PatternはこのApplicationが所有している",
	})
	if p.Relation != "" {
		edges = append(edges, ImpactEdge{
			From: key, To: subjectKey("relation", p.Relation), Via: "application.pattern.relation",
			Reason: "Patternが明示的に参照しているRelationである",
		})
	}
	if p.Data != "" {
		edges = append(edges, ImpactEdge{
			From: key, To: subjectKey("data", p.Data), Via: "application.pattern.data",
			Reason: "Patternが明示的に参照しているDataである",
		})
	}
	if cq := p.ContractQuery; cq != nil {
		edges = append(edges, ImpactEdge{
			From: key, To: "contract-query:" + string(cq.Contract) + "/" + string(cq.Query), Via: "application.pattern.contract-query",
			Reason: "Patternが明示的に選んだDatabase Contract Queryである",
		})
		for _, in := range cq.Inputs {
			edges = append(edges, ImpactEdge{
				From: key, To: subjectKey("attribute", in.TargetAttribute), Via: "application.pattern.contract-query.input",
				Reason: "Contract Queryのinput " + string(in.Input) + " へ渡す値のAttributeである",
			})
		}
	}
	return edges
}

func usesNamespace(app application.Application, namespace core.CanonicalID) bool {
	for _, used := range app.SemanticUses {
		if used == namespace {
			return true
		}
	}
	return false
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

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
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

func findApplication(w ir.World, id core.CanonicalID) (application.Application, bool) {
	for _, app := range w.Applications {
		if app.ID == id {
			return app, true
		}
	}
	return application.Application{}, false
}

func findPattern(w ir.World, id core.CanonicalID) (application.Application, application.Pattern, bool) {
	for _, app := range w.Applications {
		for _, p := range app.Patterns {
			if p.ID == id {
				return app, p, true
			}
		}
	}
	return application.Application{}, application.Pattern{}, false
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
