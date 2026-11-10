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
	// The data layer declares subjects of its own: a database, the contracts
	// it publishes, and each contract query. Declared edges name them, and an
	// edge whose endpoint is not a node is dropped, so `targets-database`,
	// `requires-contract`, `contract-query`, `reads` and `writes` all vanished
	// and the data layer never appeared in `trace` (#4068).
	//
	// These nodes are addressable as edges but are not added to the advertised
	// subject-kind list: that list is the public query grammar, and widening it
	// is a contract change (option B of #4068).
	for _, e := range d.DatabaseElements {
		if e.Kind == "" || e.ID == "" {
			continue
		}
		add(e.Kind, string(e.ID))
	}
	ruleSet := map[modelref.Rule]bool{}
	// The same edge can be declared by more than one collection — `realizes` is
	// built both from DeclaredDependencies and from CustomGoBindings — so the
	// projection must be a set here. Appending unconditionally reported one
	// incoming edge as two, which read as "two applications implement this
	// capability" (#4024).
	//
	// Unresolved drops repeats for the same reason: the message names the
	// reference and nothing else, so a second copy adds no information and
	// inflates every count taken from it.
	edgeSeen := map[string]bool{}
	unresolvedSeen := map[string]bool{}
	unresolve := func(message string) {
		if unresolvedSeen[message] {
			return
		}
		unresolvedSeen[message] = true
		p.Unresolved = append(p.Unresolved, message)
	}
	addEdge := func(fromKind, fromID, relation, toKind, toID string) {
		from := modelref.Key{Scope: scope, Subject: subject.Subject{Kind: fromKind, ID: fromID}}
		if !nodes[from] {
			unresolve("unmaterialized-from:" + fromKind + ":" + fromID)
			return
		}
		if toKind == "semantic-element" {
			// Keep resolution linear in the number of definitions and edges.
			// Two definitions with the same ID but different kinds are not
			// interchangeable; they must remain explicitly ambiguous.
			kinds := semanticKindsByID[toID]
			if len(kinds) != 1 {
				unresolve("unresolved-semantic-element:" + toID)
				return
			}
			toKind = kinds[0]
		}
		to := modelref.Key{Scope: scope, Subject: subject.Subject{Kind: toKind, ID: toID}}
		if !nodes[to] {
			unresolve("unmaterialized-to:" + toKind + ":" + toID)
			return
		}
		// The identity of an edge is what a caller can see: the endpoints and
		// the relation. A repeat is the same statement, not a second one.
		identity := fromKind + "\x00" + fromID + "\x00" + relation + "\x00" + toKind + "\x00" + toID
		if edgeSeen[identity] {
			return
		}
		edgeSeen[identity] = true
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
	// `realize X with custom go`, declared under an application, states that
	// the application implements the capability with a generated Go adapter.
	// The index already carried the binding in CustomGoBindings, but the graph
	// never made it an edge, so `trace` could not answer "what implements this
	// capability" (#3983). The edge runs application -> capability named
	// `realizes`: a reader asking about a capability reads -direction incoming.
	// The adapter filename and hook name stay index metadata; they name a
	// generated file, not a model subject, so they are not graph nodes.
	for _, b := range d.CustomGoBindings {
		if b.Application == "" || b.Capability == "" {
			p.Unresolved = append(p.Unresolved, "incomplete-realization:"+string(b.Capability))
			continue
		}
		addEdge("application", string(b.Application), "realizes", "semantic-capability", string(b.Capability))
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

// SupportsSubjectKind reports whether FromIndex materializes this exact subject
// kind. A kind that is absent here may still be readable through the Canonical
// World (inspect); the projection would not place it in the candidate graph at
// all. Callers that must tell "no declared references" apart from "this kind is
// not projected" ask this question before they trace, so the answer cannot
// drift from what FromIndex actually builds.
func SupportsSubjectKind(kind string) bool {
	for _, supported := range SupportedSubjectKinds {
		if kind == supported {
			return true
		}
	}
	return false
}

// SupportedSubjectKinds lists the kind set SupportsSubjectKind answers from.
// It is exported so a caller can advertise the same set it will accept, without
// keeping a second copy that could drift from FromIndex (#3242, #3180).
var SupportedSubjectKinds = []string{
	"semantic", "semantic-type", "semantic-data", "semantic-attribute",
	"semantic-relation", "semantic-validation", "semantic-capability",
	"application", "application-pattern", "query",
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
