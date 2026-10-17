// Package derivedbridge projects existing Derived Index entries into typed,
// read-only candidate references. It neither imports source nor mutates World.
package derivedbridge

import (
	"sort"
	"strings"
	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/subject"
)

// Projection is a limited view of the existing Derived Index. The caller
// supplies the scope; neither logical version nor source revision is inferred.
type Projection struct {
	Graph           modelref.Graph  `json:"graph"`
	Rules           []modelref.Rule `json:"rules"`
	Unresolved      []string        `json:"unresolved,omitempty"`
	AuthorityLimits []string        `json:"authorityLimits"`
}

// FromIndex transports only already-declared references. Unknown targets are
// reported, never fabricated. Business Procedures are not in ir.World and are
// deliberately not inferred from Application patterns or Function-level CRUD.
func FromIndex(d index.Derived, scope string) Projection {
	// Capacities only avoid repeated growth; an empty projection still carries
	// empty, non-nil Nodes and Edges.
	nodeHint := len(d.SemanticElements) + 2*len(d.ApplicationPatterns) + len(d.Queries)
	p := Projection{Graph: modelref.Graph{
		Schema: "tei.modelref.v0",
		Nodes:  make([]modelref.Node, 0, nodeHint),
		Edges:  make([]modelref.Edge, 0, len(d.SemanticDependencies)+len(d.DeclaredDependencies)),
	}, AuthorityLimits: []string{
		"derived-index-is-not-authoritative-world", "business-procedure-not-in-current-world", "source-evidence-not-bound",
	}}
	if scope == "" {
		p.Unresolved = append(p.Unresolved, "scope-required")
		return p
	}
	nodes := make(map[modelref.Key]bool, nodeHint)
	semanticKindsByID := make(map[string][]string)
	add := func(kind, id string) {
		if id == "" {
			return
		}
		k := modelref.Key{Scope: scope, Subject: subject.Subject{Kind: kind, ID: id}}
		if nodes[k] {
			return // repeated use of the same Application is not a duplicate definition
		}
		nodes[k] = true
		if strings.HasPrefix(kind, "semantic-") {
			semanticKindsByID[id] = append(semanticKindsByID[id], kind)
		}
		p.Graph.Nodes = append(p.Graph.Nodes, modelref.Node{Key: k})
	}
	for _, e := range d.SemanticElements {
		kind, ok := semanticKind(e.Kind)
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
			// Keep resolution linear in the number of definitions and edges.
			// Two definitions with the same ID but different kinds are not
			// interchangeable; they must remain explicitly ambiguous.
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
	sortByKey(p.Graph.Nodes, func(n modelref.Node) string {
		return n.Key.Subject.Kind + "/" + n.Key.Subject.ID
	})
	sortByKey(p.Graph.Edges, func(e modelref.Edge) string {
		return e.From.Subject.Kind + "/" + e.From.Subject.ID + "/" + e.Relation + "/" + e.To.Subject.Kind + "/" + e.To.Subject.ID
	})
	sortByKey(p.Rules, func(r modelref.Rule) string {
		return r.FromKind + "/" + r.Relation + "/" + r.ToKind
	})
	sort.Strings(p.Unresolved)
	return p
}

// sortByKey orders items by a string key that is built once per item. Building
// the key inside the comparison costs O(n log n) string concatenations, each of
// which allocates. The key text, and therefore the resulting order including
// the placement of items whose keys are equal, is exactly what a comparison of
// the same keys through sort.Slice produces: the decorated slice goes through
// the same sort.Slice with the same outcomes for every comparison.
func sortByKey[T any](items []T, key func(T) string) {
	if len(items) < 2 {
		return
	}
	type keyed struct {
		key  string
		item T
	}
	decorated := make([]keyed, len(items))
	for i, item := range items {
		decorated[i] = keyed{key: key(item), item: item}
	}
	sort.Slice(decorated, func(i, j int) bool { return decorated[i].key < decorated[j].key })
	for i := range decorated {
		items[i] = decorated[i].item
	}
}

func semanticKind(kind string) (string, bool) {
	switch kind {
	case "namespace":
		return "semantic", true
	case "type":
		return "semantic-type", true
	case "data":
		return "semantic-data", true
	case "attribute":
		return "semantic-attribute", true
	case "relation":
		return "semantic-relation", true
	case "validation":
		return "semantic-validation", true
	case "capability":
		return "semantic-capability", true
	default:
		return "", false
	}
}
