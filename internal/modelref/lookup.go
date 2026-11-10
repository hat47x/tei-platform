package modelref

import "strings"

// lookup is a read-only index for one immutable graph snapshot. It keeps
// validation linear in the number of references for ordinary unique IDs.
// Scope remains part of each candidate's logical key.
// Resolver resolves many references against one immutable snapshot. Building
// it indexes the nodes once, so resolving R references costs O(nodes + R)
// instead of the O(nodes * R) that calling Resolve per reference costs: Resolve
// re-indexes every node on every call.
type Resolver struct {
	lookup lookup
}

// NewResolver indexes nodes once for reuse across references.
func NewResolver(nodes []Node) Resolver { return Resolver{lookup: newLookup(nodes)} }

// Resolve answers one reference. The result is identical to Resolve(nodes, ref)
// for the nodes the resolver was built from.
func (r Resolver) Resolve(ref Ref) (Key, string) { return r.lookup.resolve(ref) }

type lookup struct {
	byID map[string][]Node
}

func newLookup(nodes []Node) lookup {
	indexed := lookup{byID: make(map[string][]Node, len(nodes))}
	for _, node := range nodes {
		indexed.byID[node.Key.Subject.ID] = append(indexed.byID[node.Key.Subject.ID], node)
	}
	return indexed
}

func (l lookup) resolve(ref Ref) (Key, string) {
	if strings.TrimSpace(ref.Subject.Kind) == "" || strings.TrimSpace(ref.Subject.ID) == "" ||
		(ref.Scope != "" && strings.TrimSpace(ref.Scope) != ref.Scope) {
		return Key{}, "invalid-reference"
	}
	candidates := make([]Node, 0, 1)
	wrongKind := false
	for _, node := range l.byID[ref.Subject.ID] {
		if ref.Scope != "" && node.Key.Scope != ref.Scope {
			continue
		}
		if node.Key.Subject.Kind != ref.Subject.Kind {
			wrongKind = true
			continue
		}
		candidates = append(candidates, node)
	}
	if len(candidates) == 0 {
		if wrongKind {
			return Key{}, "wrong-kind"
		}
		return Key{}, "missing-reference"
	}
	if len(candidates) > 1 {
		return Key{}, "ambiguous-reference"
	}
	if ref.Version != "" && candidates[0].Version != ref.Version {
		return Key{}, "version-mismatch"
	}
	return candidates[0].Key, ""
}
