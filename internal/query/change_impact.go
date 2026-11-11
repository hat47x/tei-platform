package query

import "sort"

// Read-only change impact for one derived claim (#2582 RC-01/02/04).
//
// This is a pure explanation over an existing DerivedIR. It is not a reactive
// runtime, a notification broker, a cache or a planner: it reads no data,
// applies no delta and stores nothing. Every statement is relative to the
// dependency information the producer supplied (analyzer lowering plus the
// caller's explicit source bindings). Where that information is missing the
// verdict is unknown; a dependency graph is never completed by inference.

type ImpactVerdict string

const (
	// ImpactRetain: no declared read dependency on any changed context. The
	// claim keeps the qualification it already had; it is not promoted.
	ImpactRetain ImpactVerdict = "retain"
	// ImpactRecheck: a declared dependency changed. The shown value must not
	// be treated as current until it is read again.
	ImpactRecheck ImpactVerdict = "recheck"
	// ImpactUnknown: the dependency closure is unresolved, so neither "not
	// affected" nor a bounded affected set can be stated.
	ImpactUnknown ImpactVerdict = "unknown"
)

const (
	RefreshNone = "none"
	// The only refresh offered. No provider delta guarantee is in evidence
	// here, so insert-only or differential application is never proposed.
	RefreshReadOnlyRefetch = "read-only-refetch"

	// Display reuse of a claim says nothing about whether an operation may be
	// executed against it, and an observed change is not a confirmed commit.
	OperationEligibilityNotEvaluated = "not-evaluated"
)

type InputSliceState string

const (
	SliceNotFetched InputSliceState = "not-fetched"
	SlicePartial    InputSliceState = "partial"
	SliceFailed     InputSliceState = "failed"
	// SliceComplete with ObservedRows == 0 is a verified empty result within
	// the retrieved scope. It is a different fact from not-fetched or failed.
	SliceComplete InputSliceState = "complete"
	// SliceStateUnknown is reported when the caller supplied no state.
	SliceStateUnknown InputSliceState = "unknown"
)

// InputSlice is the caller-declared retrieval state of one Data Context. It
// is echoed as evidence; this package does not verify provider completeness.
type InputSlice struct {
	State        InputSliceState `json:"state"`
	ObservedRows int             `json:"observedRows"`
}

type ChangeImpactRequest struct {
	// Claim is the RelNode ID whose output the claim or display item reads.
	Claim string `json:"claim"`
	// Bindings maps a source RelNode ID to the Data Context ID it reads. It
	// is an explicit producer declaration. Source names, aliases and SQL
	// text are never used to guess a context.
	Bindings map[string]string `json:"bindings,omitempty"`
	// Slices is keyed by Data Context ID.
	Slices map[string]InputSlice `json:"slices,omitempty"`
	// Changed lists Data Context IDs with an observed change.
	Changed []string `json:"changed,omitempty"`
}

type ChangeImpact struct {
	Claim   string        `json:"claim"`
	Verdict ImpactVerdict `json:"verdict"`
	Refresh string        `json:"refresh"`
	// DependsOn is the declared context closure found so far. It is complete
	// only when Unresolved is empty.
	DependsOn []string `json:"dependsOn,omitempty"`
	// Unresolved lists why the closure cannot be trusted.
	Unresolved []string `json:"unresolved,omitempty"`
	// NonMonotonic lists, per changed context, operators on the path to the
	// claim under which new input may retract or replace shown output.
	NonMonotonic []string `json:"nonMonotonic,omitempty"`
	// Provisional lists outer-join null-supplying contexts that are not
	// complete: an unmatched (NULL-extended) row is then not proof of absence.
	Provisional []string              `json:"provisional,omitempty"`
	Inputs      map[string]InputSlice `json:"inputs,omitempty"`
	// DisplayReusable is true only for ImpactRetain.
	DisplayReusable      bool   `json:"displayReusable"`
	OperationEligibility string `json:"operationEligibility"`
}

const changeImpactVisitBudget = 10000

type impactWalk struct {
	nodes       map[string]RelNode
	byScope     map[string][]string
	bindings    map[string]string
	onStack     map[string]bool
	visited     map[string]bool
	budget      int
	unresolved  map[string]bool
	depends     map[string]bool
	hazards     map[string]map[string]bool // context -> operator hazards on a path
	nullSide    map[string]bool            // contexts on an outer-join null-supplying side
	scopesInUse map[string]bool
}

func nodeScope(n RelNode) string {
	if n.ScopeID == "" {
		return "scope:root"
	}
	return n.ScopeID
}

func (w *impactWalk) walk(id string, carried []string, nullSupplying bool) {
	if w.budget <= 0 {
		w.unresolved["graph-too-large"] = true
		return
	}
	w.budget--
	n, ok := w.nodes[id]
	if !ok {
		w.unresolved["dangling-input:"+id] = true
		return
	}
	if w.onStack[id] {
		w.unresolved["cycle:"+id] = true
		return
	}
	w.onStack[id] = true
	defer delete(w.onStack, id)
	w.visited[id] = true
	w.scopesInUse[nodeScope(n)] = true

	switch n.Kind {
	case RelSource:
		if n.SourceKind == "cte" || n.SourceKind == "subquery" {
			members := w.byScope[n.Binding]
			if n.Binding == "" || n.Resolution != "logical" || len(members) == 0 {
				w.unresolved["binding-unresolved:"+id] = true
				return
			}
			// Scope granularity only: which operator of the bound scope
			// feeds this reference is not recorded, so every member counts.
			next := append(append([]string(nil), carried...), "scope-binding")
			for _, m := range members {
				w.walk(m, next, nullSupplying)
			}
			return
		}
		ctx, bound := w.bindings[id]
		if !bound || ctx == "" {
			w.unresolved["source-unbound:"+id] = true
			return
		}
		w.depends[ctx] = true
		if w.hazards[ctx] == nil {
			w.hazards[ctx] = map[string]bool{}
		}
		for _, h := range carried {
			w.hazards[ctx][h] = true
		}
		if nullSupplying {
			w.nullSide[ctx] = true
		}
		return
	case RelOpaque:
		w.unresolved["opaque:"+id] = true
		return
	case RelJoin:
		if len(n.Inputs) != 2 {
			w.unresolved["join-arity:"+id] = true
			return
		}
		leftNull, rightNull := false, false
		extra := ""
		switch n.JoinKind {
		case "inner", "cross":
		case "left":
			rightNull = true
		case "right":
			leftNull = true
		case "full":
			leftNull, rightNull = true, true
		default:
			// semi/anti/unknown kinds are not proven monotone.
			extra = "join-kind-unverified"
		}
		for i, in := range n.Inputs {
			next := append([]string(nil), carried...)
			if extra != "" {
				next = append(next, extra)
			}
			sideNull := (i == 0 && leftNull) || (i == 1 && rightNull)
			if sideNull {
				next = append(next, "outer-join-null-side")
			}
			w.walk(in, next, nullSupplying || sideNull)
		}
		return
	}

	hazard := ""
	switch n.Kind {
	case RelAggregate:
		hazard = "aggregate"
	case RelDistinct:
		hazard = "distinct"
	case RelLimit:
		hazard = "limit"
	case RelFilter:
		// Predicate text is not interpreted. It may hold NOT EXISTS, IS NULL
		// over an outer join or other negation, so monotonicity is unproven.
		hazard = "filter-unparsed"
	case RelProject, RelSort:
	default:
		w.unresolved["unknown-operator:"+id] = true
		return
	}
	if len(n.Inputs) == 0 {
		w.unresolved["missing-input:"+id] = true
		return
	}
	next := append([]string(nil), carried...)
	if hazard != "" {
		next = append(next, hazard)
	}
	for _, in := range n.Inputs {
		w.walk(in, next, nullSupplying)
	}
}

// AssessChangeImpact explains, for one claim, whether an observed change to
// some Data Contexts leaves the claim reusable for display. It never reports
// retain unless the whole dependency closure is resolved from producer data.
func AssessChangeImpact(ir DerivedIR, req ChangeImpactRequest) ChangeImpact {
	out := ChangeImpact{
		Claim:                req.Claim,
		Verdict:              ImpactUnknown,
		Refresh:              RefreshReadOnlyRefetch,
		OperationEligibility: OperationEligibilityNotEvaluated,
	}
	w := &impactWalk{
		nodes: map[string]RelNode{}, byScope: map[string][]string{}, bindings: req.Bindings,
		onStack: map[string]bool{}, visited: map[string]bool{}, budget: changeImpactVisitBudget,
		unresolved: map[string]bool{}, depends: map[string]bool{}, hazards: map[string]map[string]bool{},
		nullSide: map[string]bool{}, scopesInUse: map[string]bool{},
	}
	for _, n := range ir.Relational.Nodes {
		if _, dup := w.nodes[n.ID]; dup {
			w.unresolved["duplicate-node:"+n.ID] = true
		}
		w.nodes[n.ID] = n
		w.byScope[nodeScope(n)] = append(w.byScope[nodeScope(n)], n.ID)
	}
	// Partial or absent lowering does not say where the unlowered part is,
	// so no claim of this query can be shown to be unaffected.
	if ir.Coverage.Parse != CoverageFull || ir.Coverage.Lowering != CoverageFull {
		w.unresolved["lowering-not-full"] = true
	}
	if _, ok := w.nodes[req.Claim]; !ok || req.Claim == "" {
		w.unresolved["claim-not-found"] = true
	} else {
		w.walk(req.Claim, nil, false)
	}

	changed := map[string]bool{}
	for _, c := range req.Changed {
		changed[c] = true
	}

	// A nested scope that no operator links to (for example a correlated or
	// scalar subquery inside a predicate) can still decide the claim's rows.
	parents := map[string]string{}
	duplicateScope := false
	for _, s := range ir.Scopes {
		if _, dup := parents[s.ID]; dup {
			duplicateScope = true
		}
		parents[s.ID] = s.ParentID
	}
	if duplicateScope {
		w.unresolved["scope-hierarchy-ambiguous"] = true
	}
	for scope := range w.scopesInUse {
		if _, ok := parents[scope]; !ok {
			w.unresolved["scope-unknown:"+scope] = true
		}
	}
	for _, n := range ir.Relational.Nodes {
		if w.visited[n.ID] || !w.underUsedScope(nodeScope(n), parents) {
			continue
		}
		switch n.Kind {
		case RelOpaque:
			w.unresolved["unlinked-opaque:"+n.ID] = true
		case RelSource:
			if n.SourceKind == "cte" || n.SourceKind == "subquery" {
				w.unresolved["unlinked-binding:"+n.ID] = true
			} else if ctx, ok := w.bindings[n.ID]; !ok || ctx == "" {
				w.unresolved["unlinked-source-unbound:"+n.ID] = true
			} else if changed[ctx] {
				w.unresolved["unlinked-nested-dependency:"+ctx] = true
			}
		}
	}

	out.DependsOn = sortedKeys(w.depends)
	out.Unresolved = sortedKeys(w.unresolved)
	if len(out.DependsOn) > 0 {
		out.Inputs = map[string]InputSlice{}
	}
	for _, ctx := range out.DependsOn {
		slice, ok := req.Slices[ctx]
		if !ok || slice.State == "" {
			slice = InputSlice{State: SliceStateUnknown}
		}
		out.Inputs[ctx] = slice
		if w.nullSide[ctx] && slice.State != SliceComplete {
			out.Provisional = append(out.Provisional, "unmatched-rows-not-absence-proof:"+ctx+":"+string(slice.State))
		}
	}

	affected := false
	for _, ctx := range out.DependsOn {
		if !changed[ctx] {
			continue
		}
		affected = true
		for _, h := range sortedKeys(w.hazards[ctx]) {
			out.NonMonotonic = append(out.NonMonotonic, h+":"+ctx)
		}
	}
	switch {
	case affected:
		// A known dependency changed: the claim must be read again whether
		// or not other paths are unresolved. Unresolved stays listed.
		out.Verdict = ImpactRecheck
	case len(out.Unresolved) > 0:
		out.Verdict = ImpactUnknown
	default:
		out.Verdict = ImpactRetain
		out.Refresh = RefreshNone
		out.DisplayReusable = true
	}
	return out
}

// underUsedScope reports whether scope is a strict descendant of a scope the
// claim's closure already uses.
func (w *impactWalk) underUsedScope(scope string, parents map[string]string) bool {
	seen := map[string]bool{}
	for cur := parents[scope]; cur != "" && !seen[cur]; cur = parents[cur] {
		seen[cur] = true
		if w.scopesInUse[cur] {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
