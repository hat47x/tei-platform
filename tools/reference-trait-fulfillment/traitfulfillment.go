// Package traitfulfillment is a read-only reference for #3279 (P0-F): it asks
// whether an implementation obligation stated by one atomic Trait is met, using
// only what the existing Contract/Provider composition and the existing
// Inspector observation already say.
//
// An atomic Trait is NOT a new schema here. It is one existing
// composition.Contract (one Operation whose Shape fixes its request, result and
// invariant) attached to one Subject in one Scope. The first instance is the
// authoring Inspector Contract `tei.operations.inspect`, whose invariant is
// operations.ValidateInspectionProjection.
//
// The reference owns no Registry, no resolver and no runner. Provider presence
// is whatever composition.Registry.InspectWiringSet answers; an observed call is
// whatever inspectoradapter.LocalInspectorHost reported. It only combines them
// with one extra input the Registry cannot know: how completely the caller
// loaded the Source, the Required set and the Provider catalog.
//
// The state words below are explanatory strings of this reference. They are not
// a Core enum, an accepted API, or a grant. A result here never qualifies the
// formal Provider invocation (#1251), a verified BOM (#3267), or a current
// Grant/Effect (#1174).
package traitfulfillment

import (
	"fmt"
	"sort"

	"tei/internal/composition"
	"tei/internal/inspectoradapter"
)

// Obligation levels. An advisory Trait never produces an unmet obligation.
const (
	LevelRequired = "required"
	LevelAdvisory = "advisory"
)

// Implementation methods. `manual-required` means the target itself must supply
// the implementation named by ImplementedBy; another Provider offering the same
// Contract is not chosen in its place. `provided` accepts the one unambiguous
// existing Provider.
const (
	MethodManualRequired = "manual-required"
	MethodProvided       = "provided"
)

// Finding states, weakest evidence first.
const (
	// StateConflict: the requirements or the catalog contradict each other.
	StateConflict = "conflict"
	// StateUnfulfilled: Source and catalog were confirmed complete and the
	// required implementation is not there.
	StateUnfulfilled = "unfulfilled"
	// StateUnknown: the implementation was not found, but what was loaded is not
	// enough to say it is absent.
	StateUnknown = "unknown"
	// StateNotProvided: an advisory Trait has no implementation. Not a failure.
	StateNotProvided = "not-provided"
	// StateDeclaredUnverified: a Provider declares the Contract. A matching
	// signature or a self-declaration is all that is known.
	StateDeclaredUnverified = "declared-unverified"
	// StateInvariantObserved: the selected implementation's own code was entered
	// and its answer passed the Contract invariant, for the observed call only.
	StateInvariantObserved = "invariant-observed"
)

// Report summaries add one word: the required findings are all at least
// declared, but the Required set itself was not fully obtained.
const SummaryPartial = "partial"

// Obligation is one atomic Trait requirement on one Subject.
type Obligation struct {
	Trait   composition.Contract
	Scope   string
	Subject string
	Level   string
	Method  string
	// ImplementedBy is the Provider ID the target must itself supply. It is
	// required for manual-required and must be empty for provided.
	ImplementedBy string
}

// Closure is the caller's statement of how completely it loaded its inputs.
// The reference does not verify it; it only refuses to conclude absence
// without it. An omitted Closure therefore means "nothing confirmed".
type Closure struct {
	// SourceComplete: the Subject's Source was read to the end (for example a
	// validation report without Incomplete and without unread-source findings).
	SourceComplete bool
	// RequiredComplete: every Trait required of the Subjects was obtained. When
	// false, known gaps are still reported, but no overall fulfilment is.
	RequiredComplete bool
	// CatalogComplete: every Provider declaration in scope was loaded into the
	// Registry, as of CatalogRevision. A claim about another revision is stale
	// and counts as not complete.
	CatalogComplete bool
	CatalogRevision uint64
}

// Observation is one call seen through the existing Inspector route.
type Observation struct {
	// Invocation is what the Host reported about the entry it called.
	Invocation inspectoradapter.Invocation
	// ExpectedEntry is the independently written symbol of the implementation
	// that should have been entered. It is not read from the inventory.
	ExpectedEntry string
	// ConsumerChecked is true when the consumer received the answer through
	// inspectoradapter.Selected.Inspect, which applies the Contract invariant
	// after the entry returns. ConsumerErr is the error that call returned.
	ConsumerChecked bool
	ConsumerErr     error
}

// Finding is the answer for one merged Obligation.
type Finding struct {
	Trait   composition.Contract
	Scope   string
	Subject string
	Level   string
	Method  string
	State   string
	// Reason names the first missing edge in machine-readable form.
	Reason string
	// ProviderID is the declared implementation, when one was selected.
	ProviderID string
	// NotChosen lists Providers that offer the same Contract but were not
	// accepted in place of a manual-required implementation.
	NotChosen []string
	// Observed lists the separate facts that were actually seen. A state above
	// declared-unverified claims exactly these and nothing else.
	Observed []string
}

// Report is the answer for a set of Obligations.
type Report struct {
	// Summary is the weakest state among the required findings, or
	// SummaryPartial when they are all at least declared but RequiredComplete is
	// false. Advisory findings never change it. It is empty when nothing is
	// required.
	Summary  string
	Findings []Finding
	// NotQualified names what this static reference never establishes.
	NotQualified []string
}

var notQualified = []string{
	"formal Provider artifact entry invoked by a product Consumer (#1251)",
	"verified BOM for an exact Provider/Consumer/Target lock (#3267)",
	"current Host Grant, disclosure and Effect (#1174)",
	"Owner acceptance of meaning",
}

type key struct {
	trait   composition.Contract
	scope   string
	subject string
}

func (k key) slot() string {
	return fmt.Sprintf("%q/%q/%q/%q/%q", k.scope, k.subject, k.trait.ID, k.trait.Version, k.trait.Shape)
}

func validObligation(o Obligation) bool {
	if o.Trait.ID == "" || o.Trait.Version == "" || o.Trait.Shape == "" || o.Scope == "" || o.Subject == "" {
		return false
	}
	if o.Level != LevelRequired && o.Level != LevelAdvisory {
		return false
	}
	switch o.Method {
	case MethodManualRequired:
		return o.ImplementedBy != ""
	case MethodProvided:
		return o.ImplementedBy == ""
	}
	return false
}

// merged is the order-independent union of the Obligations on one key.
type merged struct {
	key      key
	level    string
	method   string
	by       string
	invalid  bool
	conflict bool
}

// merge treats Obligations as a set. A repeated Obligation adds nothing; a
// required one is not weakened by an advisory one on the same key; two
// different implementation methods for the same key contradict each other.
// Every rule is symmetric, so the input order cannot change the result.
func merge(obligations []Obligation) []merged {
	byKey := map[key]*merged{}
	for _, o := range obligations {
		k := key{trait: o.Trait, scope: o.Scope, subject: o.Subject}
		m, seen := byKey[k]
		if !seen {
			m = &merged{key: k, level: o.Level, method: o.Method, by: o.ImplementedBy}
			byKey[k] = m
			m.invalid = !validObligation(o)
			continue
		}
		if !validObligation(o) {
			m.invalid = true
			continue
		}
		if o.Level == LevelRequired {
			m.level = LevelRequired
		}
		if o.Method != m.method || o.ImplementedBy != m.by {
			m.conflict = true
		}
	}
	out := make([]merged, 0, len(byKey))
	for _, m := range byKey {
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key.slot() < out[j].key.slot() })
	return out
}

// Assess answers each Obligation from one Registry snapshot per Scope. It is
// read-only: it selects, calls and grants nothing.
func Assess(registry *composition.Registry, obligations []Obligation, closure Closure, observations []Observation) Report {
	report := Report{NotQualified: append([]string(nil), notQualified...)}
	items := merge(obligations)
	report.Findings = make([]Finding, len(items))
	byScope := map[string][]int{}
	for i, m := range items {
		report.Findings[i] = Finding{
			Trait: m.key.trait, Scope: m.key.scope, Subject: m.key.subject,
			Level: m.level, Method: m.method,
		}
		switch {
		case m.invalid:
			report.Findings[i].State, report.Findings[i].Reason = StateUnknown, "invalid-obligation"
		case m.conflict:
			report.Findings[i].State, report.Findings[i].Reason = StateConflict, "contradictory-implementation-method"
		default:
			byScope[m.key.scope] = append(byScope[m.key.scope], i)
		}
	}
	for _, indices := range byScope {
		reqs := make([]composition.WiringRequirement, len(indices))
		for n, i := range indices {
			reqs[n] = composition.WiringRequirement{
				Contract: items[i].key.trait, Scope: items[i].key.scope,
				ProviderID: items[i].by, SlotID: items[i].key.slot(),
			}
		}
		set := registry.InspectWiringSet(reqs)
		// Absence can be concluded only from a complete Source and a complete
		// catalog at the very revision this answer was read from.
		absenceConfirmed := closure.SourceComplete && closure.CatalogComplete &&
			registry != nil && closure.CatalogRevision == set.Revision
		for n, i := range indices {
			applyWiring(&report.Findings[i], set.Findings[n], absenceConfirmed, observations)
		}
	}
	report.Summary = summarize(report.Findings, closure.RequiredComplete)
	return report
}

func applyWiring(f *Finding, wiring composition.WiringFinding, absenceConfirmed bool, observations []Observation) {
	f.Reason = string(wiring.Reason)
	switch wiring.Verdict {
	case composition.ProposalConflict:
		f.State = StateConflict
		return
	case composition.ProposalCandidate:
		if wiring.Selected == nil {
			f.State = StateUnknown
			return
		}
		f.ProviderID = wiring.Selected.ProviderID
		f.State, f.Reason = StateDeclaredUnverified, "declaration-only"
		applyObservations(f, observations)
		return
	}
	if f.Method == MethodManualRequired {
		// Another Provider of the same Contract is reported, never substituted.
		f.NotChosen = append([]string(nil), wiring.Candidates...)
	}
	switch {
	case f.Level == LevelAdvisory:
		f.State = StateNotProvided
	case absenceConfirmed:
		f.State = StateUnfulfilled
	default:
		f.State = StateUnknown
	}
}

// applyObservations raises a declared implementation to invariant-observed only
// when one observation shows, as separate facts, that the selected Provider's
// independently expected code was entered on the Host's own Workspace and that
// the consumer's invariant check accepted the answer.
func applyObservations(f *Finding, observations []Observation) {
	// Among failed observations the most basic missing edge is reported, so
	// their order does not change the Reason.
	rank := 0
	fail := func(r int, reason string) {
		if rank == 0 || r < rank {
			rank, f.Reason = r, reason
		}
	}
	for _, o := range observations {
		if o.Invocation.SelectedProviderID != f.ProviderID {
			continue
		}
		switch {
		case o.Invocation.RegisteredProviderID != f.ProviderID:
			fail(1, "registered-implementation-differs")
		case o.ExpectedEntry == "" || o.Invocation.ObservedEntry != o.ExpectedEntry:
			fail(2, "entered-code-is-not-the-expected-implementation")
		case o.Invocation.Err != nil || !o.Invocation.OriginVerified:
			fail(3, "call-reached-the-entry-but-failed")
		case !o.ConsumerChecked:
			fail(4, "invariant-not-checked")
		case o.ConsumerErr != nil:
			fail(5, "invariant-rejected-the-answer")
		default:
			f.State, f.Reason = StateInvariantObserved, "one-observed-call"
			f.Observed = []string{
				"selected:" + f.ProviderID,
				"entered:" + o.Invocation.ObservedEntry,
				"origin:host-manifest",
				"invariant:" + f.Trait.ID,
			}
			return
		}
	}
}

var weakness = map[string]int{
	StateConflict: 0, StateUnfulfilled: 1, StateUnknown: 2,
	StateDeclaredUnverified: 3, StateInvariantObserved: 4,
}

func summarize(findings []Finding, requiredComplete bool) string {
	summary := ""
	for _, f := range findings {
		if f.Level != LevelRequired {
			continue
		}
		if summary == "" || weakness[f.State] < weakness[summary] {
			summary = f.State
		}
	}
	if !requiredComplete && (summary == StateDeclaredUnverified || summary == StateInvariantObserved) {
		return SummaryPartial
	}
	if !requiredComplete && summary == "" {
		return StateUnknown
	}
	return summary
}
