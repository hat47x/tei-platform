package derivedbridge

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/subject"
)

// semanticKindReferenceとfromIndexReferenceは、整列と確保を減らす書き換え前の定義そのもの。
// 実装が変わっても、Projectionの内容と並びがこの定義と一致し続けることを固定する。
func semanticKindReference(kind string) (string, bool) {
	switch kind {
	case "namespace":
		return "semantic", true
	case "type", "data", "attribute", "relation", "validation", "capability":
		return "semantic-" + kind, true
	default:
		return "", false
	}
}

func fromIndexReference(d index.Derived, scope string) Projection {
	p := Projection{Graph: modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{}, Edges: []modelref.Edge{}}, AuthorityLimits: []string{
		"derived-index-is-not-authoritative-world", "business-procedure-not-in-current-world", "source-evidence-not-bound",
	}}
	if scope == "" {
		p.Unresolved = append(p.Unresolved, "scope-required")
		return p
	}
	nodes := make(map[modelref.Key]bool)
	semanticKindsByID := make(map[string][]string)
	add := func(kind, id string) {
		if id == "" {
			return
		}
		k := modelref.Key{Scope: scope, Subject: subject.Subject{Kind: kind, ID: id}}
		if nodes[k] {
			return
		}
		nodes[k] = true
		if strings.HasPrefix(kind, "semantic-") {
			semanticKindsByID[id] = append(semanticKindsByID[id], kind)
		}
		p.Graph.Nodes = append(p.Graph.Nodes, modelref.Node{Key: k})
	}
	for _, e := range d.SemanticElements {
		kind, ok := semanticKindReference(e.Kind)
		if !ok {
			p.Unresolved = append(p.Unresolved, "unknown-semantic-kind:"+e.Kind)
			continue
		}
		key := modelref.Key{Scope: scope, Subject: subject.Subject{Kind: kind, ID: string(e.ID)}}
		if nodes[key] {
			p.Unresolved = append(p.Unresolved, "duplicate-semantic-definition:"+kind+":"+string(e.ID))
			continue
		}
		add(kind, string(e.ID))
	}
	for _, e := range d.ApplicationPatterns {
		add("application", string(e.Application))
		add("application-pattern", string(e.Pattern))
	}
	for _, e := range d.Queries {
		add("query", string(e.Query))
	}
	ruleSet := map[modelref.Rule]bool{}
	addEdge := func(fromKind, fromID, relation, toKind, toID string) {
		from := modelref.Key{Scope: scope, Subject: subject.Subject{Kind: fromKind, ID: fromID}}
		if !nodes[from] {
			p.Unresolved = append(p.Unresolved, "unmaterialized-from:"+fromKind+":"+fromID)
			return
		}
		if toKind == "semantic-element" {
			kinds := semanticKindsByID[toID]
			if len(kinds) != 1 {
				p.Unresolved = append(p.Unresolved, "unresolved-semantic-element:"+toID)
				return
			}
			toKind = kinds[0]
		}
		to := modelref.Key{Scope: scope, Subject: subject.Subject{Kind: toKind, ID: toID}}
		if !nodes[to] {
			p.Unresolved = append(p.Unresolved, "unmaterialized-to:"+toKind+":"+toID)
			return
		}
		rule := modelref.Rule{Relation: relation, FromKind: fromKind, ToKind: toKind}
		ruleSet[rule] = true
		p.Graph.Edges = append(p.Graph.Edges, modelref.Edge{From: from, To: modelref.Ref{Scope: scope, Subject: to.Subject}, Relation: relation})
	}
	for _, e := range d.SemanticDependencies {
		addEdge(e.FromKind, string(e.From), e.Relation, e.ToKind, string(e.To))
	}
	for _, e := range d.DeclaredDependencies {
		addEdge(e.FromKind, string(e.From), e.Relation, e.ToKind, string(e.To))
	}
	for rule := range ruleSet {
		p.Rules = append(p.Rules, rule)
	}
	sort.Slice(p.Graph.Nodes, func(i, j int) bool {
		a, b := p.Graph.Nodes[i].Key, p.Graph.Nodes[j].Key
		return a.Subject.Kind+"/"+a.Subject.ID < b.Subject.Kind+"/"+b.Subject.ID
	})
	sort.Slice(p.Graph.Edges, func(i, j int) bool {
		a, b := p.Graph.Edges[i], p.Graph.Edges[j]
		return a.From.Subject.Kind+"/"+a.From.Subject.ID+"/"+a.Relation+"/"+a.To.Subject.Kind+"/"+a.To.Subject.ID < b.From.Subject.Kind+"/"+b.From.Subject.ID+"/"+b.Relation+"/"+b.To.Subject.Kind+"/"+b.To.Subject.ID
	})
	sort.Slice(p.Rules, func(i, j int) bool {
		a, b := p.Rules[i], p.Rules[j]
		return a.FromKind+"/"+a.Relation+"/"+a.ToKind < b.FromKind+"/"+b.Relation+"/"+b.ToKind
	})
	sort.Strings(p.Unresolved)
	return p
}

// randomDerivedは、区切り文字を含むIDと関係名、同一IDの別種別、重複した定義と依存、
// 存在しない参照を意図して混ぜたDerived Indexを決定的に作る。
func randomDerived(seed int64, elements, edges int) index.Derived {
	random := rand.New(rand.NewSource(seed))
	ids := []string{"a", "b", "c", "a/b", "b/c", "x", "x/y", "y", "名前", "p.q"}
	pickID := func() core.CanonicalID { return core.CanonicalID(ids[random.Intn(len(ids))]) }
	semanticKinds := []string{"namespace", "type", "data", "attribute", "relation", "validation", "capability", "bogus"}
	relations := []string{"r", "s", "r/s", "t/u", "targets", ""}
	fromKinds := []string{"semantic", "semantic-type", "semantic-data", "semantic-attribute", "semantic-relation", "application", "application-pattern", "query", "unknown-kind"}
	toKinds := append(append([]string(nil), fromKinds...), "semantic-element", "semantic-element")

	var d index.Derived
	for i := 0; i < elements; i++ {
		d.SemanticElements = append(d.SemanticElements, index.SemanticElementEntry{
			Kind: semanticKinds[random.Intn(len(semanticKinds))], ID: pickID(),
		})
	}
	for i := 0; i < elements/4; i++ {
		d.ApplicationPatterns = append(d.ApplicationPatterns, index.ApplicationPatternIndexEntry{Application: pickID(), Pattern: pickID()})
		d.Queries = append(d.Queries, index.QueryIndexEntry{Query: pickID()})
	}
	for i := 0; i < edges; i++ {
		semantic := index.SemanticDependency{
			FromKind: fromKinds[random.Intn(len(fromKinds))], From: pickID(),
			Relation: relations[random.Intn(len(relations))],
			ToKind:   toKinds[random.Intn(len(toKinds))], To: pickID(),
		}
		d.SemanticDependencies = append(d.SemanticDependencies, semantic)
		if i%3 == 0 {
			d.DeclaredDependencies = append(d.DeclaredDependencies, index.DeclaredDependency{
				FromKind: semantic.FromKind, From: semantic.From, Relation: semantic.Relation,
				ToKind: semantic.ToKind, To: semantic.To,
			})
		}
	}
	return d
}

func TestFromIndexMatchesPerComparisonKeyDefinition(t *testing.T) {
	for seed := int64(1); seed <= 60; seed++ {
		d := randomDerived(seed, 12+int(seed%7)*9, 40+int(seed%5)*60)
		got := FromIndex(d, "scope")
		want := fromIndexReference(d, "scope")
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("seed %d: Projection differs from the per-comparison definition\n got nodes=%d edges=%d rules=%d unresolved=%d\nwant nodes=%d edges=%d rules=%d unresolved=%d",
				seed, len(got.Graph.Nodes), len(got.Graph.Edges), len(got.Rules), len(got.Unresolved),
				len(want.Graph.Nodes), len(want.Graph.Edges), len(want.Rules), len(want.Unresolved))
		}
	}
}

// 構造が異なるのに整列キーの文字列が同じになる辺（関係名や参照元IDに"/"を含む）でも、
// 同じキーの辺の並びまで書き換え前と一致する。
func TestFromIndexKeepsOrderOfEdgesWhoseSortKeysCollide(t *testing.T) {
	d := index.Derived{
		SemanticElements: []index.SemanticElementEntry{
			{Kind: "data", ID: "a"}, {Kind: "data", ID: "a/b"}, {Kind: "data", ID: "x"},
		},
		SemanticDependencies: []index.SemanticDependency{
			{FromKind: "semantic-data", From: "a/b", Relation: "c", ToKind: "semantic-data", To: "x"},
			{FromKind: "semantic-data", From: "a", Relation: "b/c", ToKind: "semantic-data", To: "x"},
			{FromKind: "semantic-data", From: "a/b", Relation: "c", ToKind: "semantic-data", To: "x"},
			{FromKind: "semantic-data", From: "a", Relation: "b/c", ToKind: "semantic-data", To: "x"},
		},
	}
	got := FromIndex(d, "scope")
	want := fromIndexReference(d, "scope")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order of colliding sort keys changed:\n got %+v\nwant %+v", got.Graph.Edges, want.Graph.Edges)
	}
	if len(got.Graph.Edges) != 4 {
		t.Fatalf("expected the four edges to be kept, got %d", len(got.Graph.Edges))
	}
}

func TestFromIndexEmptyAndScopeRequiredKeepShapes(t *testing.T) {
	for _, scope := range []string{"", "scope"} {
		got := FromIndex(index.Derived{}, scope)
		want := fromIndexReference(index.Derived{}, scope)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("scope %q: got %+v want %+v", scope, got, want)
		}
		if scope != "" && (got.Graph.Nodes == nil || got.Graph.Edges == nil) {
			t.Fatal("an empty projection must carry empty, non-nil Nodes and Edges")
		}
	}
}

func TestSemanticKindMatchesDefinitionForEveryKind(t *testing.T) {
	kinds := []string{"namespace", "type", "data", "attribute", "relation", "validation", "capability", "", "Type", "semantic", "semantic-data", "bogus"}
	for _, kind := range kinds {
		gotKind, gotOK := semanticKind(kind)
		wantKind, wantOK := semanticKindReference(kind)
		if gotKind != wantKind || gotOK != wantOK {
			t.Fatalf("%q: got (%q,%v) want (%q,%v)", kind, gotKind, gotOK, wantKind, wantOK)
		}
	}
}

func TestSortByKeyMatchesComparatorThatBuildsKeysPerComparison(t *testing.T) {
	random := rand.New(rand.NewSource(20261017))
	pieces := []string{"", "a", "b", "a/b", "/", "名", "1", "10"}
	for _, count := range []int{0, 1, 2, 3, 12, 13, 100, 2500} {
		t.Run(fmt.Sprintf("n=%d", count), func(t *testing.T) {
			type item struct{ Name, Tag string }
			items := make([]item, count)
			for i := range items {
				items[i] = item{Name: pieces[random.Intn(len(pieces))] + pieces[random.Intn(len(pieces))], Tag: fmt.Sprint(i)}
			}
			key := func(v item) string { return v.Name }
			want := append([]item(nil), items...)
			sort.Slice(want, func(i, j int) bool { return key(want[i]) < key(want[j]) })
			got := append([]item(nil), items...)
			sortByKey(got, key)
			if !reflect.DeepEqual(got, want) {
				t.Fatal("order differs from the per-comparison definition, including the placement of equal keys")
			}
		})
	}
}
