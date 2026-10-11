// Package modelref is a candidate, source-independent contract for typed
// cross-view references. It is not an Acceptance or Runtime authority.
package modelref

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"tei/internal/subject"
)

// Key is a durable logical identity. The ID is author-defined (hyphens are
// permitted); neither a source offset nor a generated runtime key belongs here.
type Key struct {
	Scope   string          `json:"scope"`
	Subject subject.Subject `json:"subject"`
}

type Ref struct {
	Scope   string          `json:"scope,omitempty"`
	Subject subject.Subject `json:"subject"`
	Version string          `json:"version,omitempty"`
}

// Evidence records where a proposed or observed meaning came from. Tokens
// and fingerprints are evidence only; they never create a canonical identity.
type Evidence struct {
	Authority           string `json:"authority"` // accepted, proposal, source-observation, derived
	Context             string `json:"context,omitempty"`
	SourceToken         string `json:"sourceToken,omitempty"`
	ArtifactFingerprint string `json:"artifactFingerprint,omitempty"`
	RuleFingerprint     string `json:"ruleFingerprint,omitempty"`
}

type Node struct {
	Key          Key        `json:"key"`
	Version      string     `json:"version,omitempty"`
	DisplayName  string     `json:"displayName,omitempty"`
	PhysicalName string     `json:"physicalName,omitempty"`
	Evidence     []Evidence `json:"evidence,omitempty"`
}

type Edge struct {
	From     Key        `json:"from"`
	To       Ref        `json:"to"`
	Relation string     `json:"relation"`
	Evidence []Evidence `json:"evidence,omitempty"`
}

type Graph struct {
	Schema string `json:"schema"`
	Nodes  []Node `json:"nodes"`
	Edges  []Edge `json:"edges"`
}

// Rule is provided by a semantic extension. An empty AcyclicGroup does not
// prohibit cycles. Distinct groups are validated independently.
type Rule struct {
	Relation        string `json:"relation"`
	FromKind        string `json:"fromKind"`
	ToKind          string `json:"toKind"`
	AcyclicGroup    string `json:"acyclicGroup,omitempty"`
	DisplayAffects  bool   `json:"displayAffects,omitempty"`
	PhysicalAffects bool   `json:"physicalAffects,omitempty"`
}

type Diagnostic struct {
	Code string `json:"code"`
	At   string `json:"at"`
}

func validKey(k Key) bool {
	return strings.TrimSpace(k.Scope) != "" && strings.TrimSpace(k.Subject.Kind) != "" &&
		strings.TrimSpace(k.Subject.ID) != "" && !strings.ContainsAny(k.Scope, "\n\r\t")
}

func evidenceDiagnostics(prefix string, values []Evidence) []Diagnostic {
	var out []Diagnostic
	for i, v := range values {
		at := fmt.Sprintf("%s/evidence/%d", prefix, i)
		switch v.Authority {
		case "accepted", "proposal":
		case "source-observation":
			if v.SourceToken == "" || v.ArtifactFingerprint == "" {
				out = append(out, Diagnostic{Code: "unbound-source-evidence", At: at})
			}
		case "derived":
			if v.RuleFingerprint == "" || v.ArtifactFingerprint == "" {
				out = append(out, Diagnostic{Code: "unbound-derivation-evidence", At: at})
			}
		default:
			out = append(out, Diagnostic{Code: "unknown-evidence-authority", At: at})
		}
	}
	return out
}

// Resolve is intentionally strict: an omitted scope must identify exactly one
// subject across all scopes, and a version constraint must match exactly.
func Resolve(nodes []Node, ref Ref) (Key, string) {
	return newLookup(nodes).resolve(ref)
}

// Validate validates references without elevating any candidate to Accepted
// meaning. This graph has no capability to execute a referenced action.
func Validate(g Graph, rules []Rule) []Diagnostic {
	var out []Diagnostic
	if g.Schema != "tei.modelref.v0" {
		out = append(out, Diagnostic{Code: "unsupported-schema", At: "schema"})
	}
	counts := make(map[Key]int)
	for i, n := range g.Nodes {
		at := fmt.Sprintf("nodes/%d", i)
		if !validKey(n.Key) {
			out = append(out, Diagnostic{Code: "invalid-node-key", At: at})
		}
		counts[n.Key]++
		if counts[n.Key] > 1 {
			out = append(out, Diagnostic{Code: "duplicate-node-key", At: at})
		}
		out = append(out, evidenceDiagnostics(at, n.Evidence)...)
	}
	resolution := newLookup(g.Nodes)
	// Conflicting plugin rules cannot be resolved by registration order.
	type ruleKey struct{ relation, fromKind, toKind string }
	rulesByKey := make(map[ruleKey]Rule, len(rules))
	for i, rule := range rules {
		at := fmt.Sprintf("rules/%d", i)
		if strings.TrimSpace(rule.Relation) == "" || strings.TrimSpace(rule.FromKind) == "" || strings.TrimSpace(rule.ToKind) == "" {
			out = append(out, Diagnostic{Code: "invalid-rule", At: at})
			continue
		}
		k := ruleKey{rule.Relation, rule.FromKind, rule.ToKind}
		if previous, ok := rulesByKey[k]; ok && previous != rule {
			out = append(out, Diagnostic{Code: "conflicting-rule", At: at})
			continue
		}
		rulesByKey[k] = rule
	}
	groupEdges := make(map[string]map[Key][]Key)
	for i, e := range g.Edges {
		at := fmt.Sprintf("edges/%d", i)
		out = append(out, evidenceDiagnostics(at, e.Evidence)...)
		if counts[e.From] != 1 {
			out = append(out, Diagnostic{Code: "invalid-source", At: at})
			continue
		}
		to, issue := resolution.resolve(e.To)
		if issue != "" {
			out = append(out, Diagnostic{Code: issue, At: at})
			continue
		}
		rule, found := rulesByKey[ruleKey{e.Relation, e.From.Subject.Kind, to.Subject.Kind}]
		if !found {
			out = append(out, Diagnostic{Code: "invalid-edge-type", At: at})
		} else if rule.AcyclicGroup != "" {
			if groupEdges[rule.AcyclicGroup] == nil {
				groupEdges[rule.AcyclicGroup] = make(map[Key][]Key)
			}
			groupEdges[rule.AcyclicGroup][e.From] = append(groupEdges[rule.AcyclicGroup][e.From], to)
		}
	}
	for group, adjacency := range groupEdges {
		color := make(map[Key]uint8)
		var visit func(Key)
		visit = func(k Key) {
			color[k] = 1
			for _, dest := range adjacency[k] {
				if color[dest] == 1 {
					out = append(out, Diagnostic{Code: "cycle", At: group})
					continue
				}
				if color[dest] == 0 {
					visit(dest)
				}
			}
			color[k] = 2
		}
		keys := make([]Key, 0, len(adjacency))
		for k := range adjacency {
			keys = append(keys, k)
		}
		sortKeys(keys)
		for _, k := range keys {
			if color[k] == 0 {
				visit(k)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].At != out[j].At {
			return out[i].At < out[j].At
		}
		return out[i].Code < out[j].Code
	})
	return out
}

func sortKeys(ks []Key) {
	sort.Slice(ks, func(i, j int) bool { return keyString(ks[i]) < keyString(ks[j]) })
}
func keyString(k Key) string { return k.Scope + "\x00" + k.Subject.Kind + "\x00" + k.Subject.ID }

// A transition across logical identities must be explicit and reviewed by
// the existing Shared Acceptance flow. No automatic rewrites or redirects.
type Transition struct {
	Kind string `json:"kind"` // rekey, split, merge
	From []Key  `json:"from"`
	To   []Key  `json:"to"`
}

func ValidateTransition(before, after Graph, t Transition) string {
	switch t.Kind {
	case "rekey":
		if len(t.From) != 1 || len(t.To) != 1 {
			return "invalid-cardinality"
		}
	case "split":
		if len(t.From) != 1 || len(t.To) < 2 {
			return "invalid-cardinality"
		}
	case "merge":
		if len(t.From) < 2 || len(t.To) != 1 {
			return "invalid-cardinality"
		}
	default:
		return "unknown-transition"
	}
	has := func(nodes []Node, key Key) bool {
		for _, n := range nodes {
			if n.Key == key {
				return true
			}
		}
		return false
	}
	seen := make(map[Key]bool)
	for _, key := range t.From {
		if !validKey(key) || seen[key] || !has(before.Nodes, key) || has(after.Nodes, key) {
			return "invalid-source-transition"
		}
		seen[key] = true
	}
	for _, key := range t.To {
		if !validKey(key) || seen[key] || has(before.Nodes, key) || !has(after.Nodes, key) {
			return "invalid-target-transition"
		}
		seen[key] = true
	}
	return ""
}

type Change struct {
	Key  Key    `json:"key"`
	Kind string `json:"kind"` // added, removed, display-renamed, physical-renamed, version-changed
}

func Diff(before, after Graph) []Change {
	old, next := make(map[Key]Node), make(map[Key]Node)
	for _, n := range before.Nodes {
		old[n.Key] = n
	}
	for _, n := range after.Nodes {
		next[n.Key] = n
	}
	var out []Change
	for key, n := range old {
		other, ok := next[key]
		if !ok {
			out = append(out, Change{Key: key, Kind: "removed"})
			continue
		}
		if n.DisplayName != other.DisplayName {
			out = append(out, Change{Key: key, Kind: "display-renamed"})
		}
		if n.PhysicalName != other.PhysicalName {
			out = append(out, Change{Key: key, Kind: "physical-renamed"})
		}
		if n.Version != other.Version {
			out = append(out, Change{Key: key, Kind: "version-changed"})
		}
		if evidenceKey(n.Evidence) != evidenceKey(other.Evidence) {
			out = append(out, Change{Key: key, Kind: "evidence-changed"})
		}
	}
	for key := range next {
		if _, ok := old[key]; !ok {
			out = append(out, Change{Key: key, Kind: "added"})
		}
	}
	// Edge changes are themselves semantic changes even if both endpoint
	// nodes keep their version. Do not silently preserve stale traces.
	type signature struct {
		From     Key
		To       Ref
		Relation string
	}

	// Evidence is an unordered set of observations for this edge. A change
	// in source / derivation evidence must invalidate the reference even
	// when from, to, relation and node versions remain unchanged.
	oldEdges, newEdges := make(map[signature][]string), make(map[signature][]string)
	for _, edge := range before.Edges {
		sig := signature{edge.From, edge.To, edge.Relation}
		oldEdges[sig] = append(oldEdges[sig], evidenceKey(edge.Evidence))
	}
	for _, edge := range after.Edges {
		sig := signature{edge.From, edge.To, edge.Relation}
		newEdges[sig] = append(newEdges[sig], evidenceKey(edge.Evidence))
	}
	for _, signatures := range []map[signature][]string{oldEdges, newEdges} {
		for sig := range signatures {
			sort.Strings(signatures[sig])
		}
	}
	structural, provenance := make(map[Key]bool), make(map[Key]bool)
	for sig, oldEvidence := range oldEdges {
		newEvidence := newEdges[sig]
		if len(oldEvidence) != len(newEvidence) {
			structural[sig.From] = true
		} else if !reflect.DeepEqual(oldEvidence, newEvidence) {
			provenance[sig.From] = true
		}
	}
	for sig, newEvidence := range newEdges {
		if len(oldEdges[sig]) != len(newEvidence) {
			structural[sig.From] = true
		}
	}
	for key := range structural {
		out = append(out, Change{Key: key, Kind: "reference-changed"})
	}
	for key := range provenance {
		if !structural[key] {
			out = append(out, Change{Key: key, Kind: "reference-evidence-changed"})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Key != out[j].Key {
			return keyString(out[i].Key) < keyString(out[j].Key)
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

// evidenceKey represents provenance as a set, independent of input order.
// JSON strings are canonical for the fixed Evidence struct of string fields.
func evidenceKey(values []Evidence) string {
	parts := make([]string, 0, len(values))
	for _, item := range values {
		encoded, _ := json.Marshal(item)
		parts = append(parts, string(encoded))
	}
	sort.Strings(parts)
	return strings.Join(parts, "\x00")
}

type Affected struct {
	Changed  Key    `json:"changed"`
	Consumer Key    `json:"consumer"`
	Reason   string `json:"reason"`
}

// Impact traverses incoming references of the *before* graph, including
// consumers of removed nodes. This is a candidate review list, not evidence
// of actual execution, authority, or business effects. References that cannot
// be uniquely resolved or whose rule is unknown are not silently accepted.
func Impact(before Graph, changes []Change, rules []Rule) []Affected {
	type incomingEdge struct {
		from Key
		rule Rule
	}
	nodes := make(map[Key]Node, len(before.Nodes))
	bySubject := make(map[string][]Key, len(before.Nodes))
	for _, node := range before.Nodes {
		nodes[node.Key] = node
		refKey := node.Key.Subject.Kind + "\x00" + node.Key.Subject.ID
		bySubject[refKey] = append(bySubject[refKey], node.Key)
	}
	incoming := make(map[Key][]incomingEdge)
	for _, edge := range before.Edges {
		if _, ok := nodes[edge.From]; !ok {
			continue
		}
		var target Key
		if edge.To.Scope != "" {
			target = Key{Scope: edge.To.Scope, Subject: edge.To.Subject}
			if _, ok := nodes[target]; !ok {
				continue
			}
		} else {
			keys := bySubject[edge.To.Subject.Kind+"\x00"+edge.To.Subject.ID]
			if len(keys) != 1 {
				continue
			}
			target = keys[0]
		}
		if edge.To.Version != "" && nodes[target].Version != edge.To.Version {
			continue
		}
		for _, rule := range rules {
			if rule.Relation == edge.Relation && rule.FromKind == edge.From.Subject.Kind && rule.ToKind == target.Subject.Kind {
				incoming[target] = append(incoming[target], incomingEdge{from: edge.From, rule: rule})
			}
		}
	}
	var out []Affected
	for _, change := range changes {
		visited := map[Key]bool{change.Key: true}
		queue := []Key{change.Key}
		for len(queue) != 0 {
			current := queue[0]
			queue = queue[1:]
			for _, edge := range incoming[current] {
				// A label or physical-name dependency must be explicit at
				// every hop, not just the first consumer.
				if change.Kind == "display-renamed" && !edge.rule.DisplayAffects ||
					change.Kind == "physical-renamed" && !edge.rule.PhysicalAffects {
					continue
				}
				if visited[edge.from] {
					continue
				}
				visited[edge.from] = true
				queue = append(queue, edge.from)
				out = append(out, Affected{Changed: change.Key, Consumer: edge.from, Reason: change.Kind})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Changed != b.Changed {
			return keyString(a.Changed) < keyString(b.Changed)
		}
		if a.Consumer != b.Consumer {
			return keyString(a.Consumer) < keyString(b.Consumer)
		}
		return a.Reason < b.Reason
	})
	return out
}
