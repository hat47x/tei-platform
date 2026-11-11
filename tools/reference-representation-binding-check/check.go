package representationbindingcheck

import (
	"fmt"
	"sort"
	"strings"
)

var representationSet = map[string]bool{
	RepTable: true, RepForm: true, RepTree: true, RepGraph: true, RepChart: true,
	RepGantt: true, RepMap: true, RepMatrix: true, RepMarkdown: true,
	RepFormula: true, RepDrawing: true, RepImage: true, RepAudio: true,
	RepVideo: true, RepComposite: true,
}

var operationSet = map[string]bool{
	OpReadOnly: true, OpInverseClaim: true, OpExplicitAction: true,
}

var axisRoleSet = map[string]bool{
	AxisDomain: true, AxisObservation: true, AxisPresentation: true, AxisStructural: true,
}

func Check(file CaseFile, opts Options) Output {
	desc := ContractDescription()
	out := Output{SchemaVersion: ReportSchema, Checked: len(file.Cases), Contract: &desc}
	for _, c := range file.Cases {
		out.Results = append(out.Results, checkCase(c, opts))
	}
	sort.SliceStable(out.Results, func(i, j int) bool { return out.Results[i].ID < out.Results[j].ID })
	return out
}

type evaluation struct {
	id          string
	title       string
	diagnostics []Diagnostic
	evidence    []string
	unqualified []string
	reject      bool
	unqual      bool
	unkn        bool
}

func (e *evaluation) rejectWith(code, detail string) {
	e.reject = true
	e.diagnostics = append(e.diagnostics, Diagnostic{Code: code, Severity: SeverityReject, Detail: detail})
}

func (e *evaluation) unqualifyWith(code, column, detail string) {
	e.unqual = true
	e.unqualified = append(e.unqualified, column)
	e.diagnostics = append(e.diagnostics, Diagnostic{Code: code, Severity: SeverityUnqualified, Detail: detail})
}

func (e *evaluation) unknownWith(code, detail string) {
	e.unkn = true
	e.diagnostics = append(e.diagnostics, Diagnostic{Code: code, Severity: SeverityUnknown, Detail: detail})
}

func (e *evaluation) info(code, detail string) {
	e.diagnostics = append(e.diagnostics, Diagnostic{Code: code, Severity: SeverityInfo, Detail: detail})
}

func checkCase(c Case, opts Options) Result {
	e := &evaluation{id: c.ID, title: c.Title}

	checkRepresentation(e, c, opts)
	checkContext(e, c, opts)
	checkScopeReuse(e, c)
	checkAxes(e, c, opts)
	checkGrain(e, c)
	checkSelection(e, c, opts)
	checkNodeIdentity(e, c)
	checkAggregate(e, c)
	checkDisplayEquality(e, c)
	checkCRS(e, c)
	checkOperation(e, c)
	checkEvidence(e, c)

	verdict := decideVerdict(e)
	return Result{
		ID:          e.id,
		Title:       e.title,
		Verdict:     verdict,
		Diagnostics: sortDiagnostics(e.diagnostics),
		Evidence:    sortedCopy(e.evidence),
		Unqualified: unqualifiedColumns(e, verdict),
	}
}

// unqualifiedColumns reports the columns that block the binding. A rejected
// result is blocked by the contradiction itself, not by a missing column, so the
// list is empty there: the two verdicts answer different questions.
func unqualifiedColumns(e *evaluation, verdict string) []string {
	if verdict != VerdictUnqualified {
		return []string{}
	}
	return sortedCopy(e.unqualified)
}

func decideVerdict(e *evaluation) string {
	switch {
	case e.reject:
		return VerdictRejected
	case e.unqual:
		return VerdictUnqualified
	case e.unkn:
		return VerdictUnknown
	}
	return VerdictBound
}

func checkRepresentation(e *evaluation, c Case, opts Options) {
	if !representationSet[c.Representation.Kind] {
		e.unknownWith(DiagUnknownRepresentation, fmt.Sprintf(
			"representation kind %q is not one of the fifteen", c.Representation.Kind))
		return
	}
	if opts.TreatTopologyAsMeaning {
		// Mutant: the layout is adopted as the data meaning.
		return
	}
	if c.Representation.TopologyIsDataMeaning {
		e.rejectWith(DiagTopologyPromoted,
			"the presentation topology is declared to be the data meaning")
	}
	if len(c.Representation.Topology) == 0 && c.Representation.Kind != RepComposite {
		e.unknownWith(DiagEvidenceMissing,
			"the representation type declares no internal topology")
	}
}

func checkContext(e *evaluation, c Case, opts Options) {
	if c.Binding.ContextID == "" {
		if opts.AcceptContextless {
			// Mutant: an instance carrying meaning is accepted outside a context.
			return
		}
		e.rejectWith(DiagContextMissing,
			"a meaning-bearing instance is placed outside any Function Data Context")
		return
	}
	for _, other := range c.OtherBindings {
		if other.ContextID == "" || other.ContextID != c.Binding.ContextID {
			continue
		}
		if other.Epoch != "" && c.Binding.Epoch != "" && other.Epoch != c.Binding.Epoch {
			e.rejectWith(DiagContextDuplicate, fmt.Sprintf(
				"context %q is bound at two epochs, %q and %q", c.Binding.ContextID, other.Epoch, c.Binding.Epoch))
		}
	}
}

// checkScopeReuse refuses a scope that two different contexts share. The same
// subset name does not establish the same subset.
func checkScopeReuse(e *evaluation, c Case) {
	if c.Binding.Scope == "" {
		return
	}
	for _, other := range c.OtherBindings {
		if other.ContextID == "" || other.ContextID == c.Binding.ContextID {
			continue
		}
		if other.Scope != "" && other.Scope == c.Binding.Scope {
			e.rejectWith(DiagScopeReused, fmt.Sprintf(
				"scope %q is reused by contexts %q and %q", c.Binding.Scope, c.Binding.ContextID, other.ContextID))
		}
	}
}

func checkAxes(e *evaluation, c Case, opts Options) {
	b := c.Binding
	for _, a := range b.Axes {
		if !axisRoleSet[a.Role] {
			e.rejectWith(DiagAxisUndeclared, fmt.Sprintf(
				"axis %q has role %q, which is not one of the four", a.Name, a.Role))
			continue
		}
		if a.Generated {
			// A generated axis is not a source axis: pagination, reflow and
			// sonification produce positions the source never had.
			e.info(DiagGeneratedAxis,
				fmt.Sprintf("axis %q (%s) is generated by the representation", a.Name, a.Role))
		}
	}

	// A source axis reported under a different role is a conflation.
	for _, src := range b.SourceAxes {
		found := false
		for _, a := range b.Axes {
			if a.Role == src {
				found = true
				break
			}
		}
		if !found {
			e.rejectWith(DiagAxisConflated, fmt.Sprintf(
				"source axis %q is reported under a role the representation does not carry", src))
		}
	}

	if needsTimeRole(c.Representation.Kind) {
		if len(b.Axes) == 0 {
			e.unqualifyWith(DiagAxisUndeclared, "axes",
				"the representation carries time and declares no axis role")
		}
		clockUnknown := b.Clock == "" || b.Clock == "unknown"
		if clockUnknown && !opts.AcceptUnknownClock {
			e.unqualifyWith(DiagClockUnknown, "clock",
				"the clock or timebase is unknown; synchronization cannot be claimed")
		}
		if b.SyncClaimed {
			if b.SyncError == "" {
				e.unqualifyWith(DiagSyncClaimUnbound, "syncError",
					"synchronization is claimed without an error bound")
			}
			clockUnknown := b.Clock == "" || b.Clock == "unknown"
			if clockUnknown && opts.AcceptUnknownClock {
				// Mutant: an unknown clock is taken as a synchronized pair.
				e.info(DiagClockUnknown, "an unknown clock was accepted as synchronized")
			}
		}
	}
	// A domain source axis rendered only as a presentation axis is the exact
	// pair the issue names as confusable. When the representation carries both
	// a domain axis and a playback axis, the mapping between them is legitimate
	// and the clock decides whether it is qualified.
	if contains(b.SourceAxes, AxisDomain) && !hasAxisRole(b.Axes, AxisDomain) {
		e.rejectWith(DiagAxisConflated,
			"domain time is reported only as playback time")
	}
}

func hasAxisRole(axes []Axis, role string) bool {
	for _, a := range axes {
		if a.Role == role {
			return true
		}
	}
	return false
}

func needsTimeRole(kind string) bool {
	switch kind {
	case RepGantt, RepAudio, RepVideo:
		return true
	}
	return false
}

func checkGrain(e *evaluation, c Case) {
	if c.Binding.Grain == "" {
		e.unqualifyWith(DiagGrainUndeclared, "grain",
			"no grain is declared, so the rendered unit is not the source unit")
	}
	if c.Binding.Scope == "" {
		e.unqualifyWith(DiagGrainUndeclared, "scope",
			"no scope is declared, so the visible subset is not the target set")
	}
}

func checkSelection(e *evaluation, c Case, opts Options) {
	if opts.TreatSelectionAsIdentity {
		// Mutant: the selection is read as the data identity.
		return
	}
	if c.Binding.Selection != "" && c.Binding.SelectionIsIdentity {
		e.rejectWith(DiagSelectionNotIdentity,
			"the presentation selection is declared to be the data identity")
	}
}

func checkNodeIdentity(e *evaluation, c Case) {
	// A tree node is a position in the rendering, not the identity of the
	// relation it renders. Hidden edges make a unique edit target impossible.
	if c.Representation.Kind != RepTree {
		return
	}
	if c.Binding.NodeIsUniqueTarget {
		e.rejectWith(DiagNodeNotIdentity,
			"a tree node is declared to be the unique edit target")
	}
	if len(c.Binding.DeclaredLosses) > 0 && contains(c.Binding.DeclaredLosses, "hidden-edge") {
		e.rejectWith(DiagNodeNotIdentity,
			"a tree with hidden edges cannot offer a unique edit target")
	}
}

func checkAggregate(e *evaluation, c Case) {
	if c.Binding.SourceGrain == "" || c.Binding.Grain == "" {
		return
	}
	if c.Binding.SourceGrain == c.Binding.Grain {
		return
	}
	// A differing rendered grain is only an aggregation when the binding itself
	// declares that grain or fact identity is lost. A finer rendering unit (a
	// frame over samples, for example) is not an aggregate of the source rows.
	aggregated := contains(c.Binding.DeclaredLosses, "grain") ||
		contains(c.Binding.DeclaredLosses, "fact-identity")
	if aggregated {
		e.rejectWith(DiagAggregateNotEntity, fmt.Sprintf(
			"rendered grain %q is an aggregate of source grain %q", c.Binding.Grain, c.Binding.SourceGrain))
		return
	}
	e.info(DiagAggregateNotEntity, fmt.Sprintf(
		"rendered grain %q differs from source grain %q and no grain loss is declared",
		c.Binding.Grain, c.Binding.SourceGrain))
}

func checkDisplayEquality(e *evaluation, c Case) {
	if c.Binding.DisplayEqualityImpliesMeaning {
		e.rejectWith(DiagDisplayNotMeaning,
			"equal rendered bytes are declared to establish equal computation")
	}
}

func checkCRS(e *evaluation, c Case) {
	if c.Representation.Kind != RepMap && c.Representation.Kind != RepDrawing {
		return
	}
	if c.Binding.CRS == "" {
		e.unqualifyWith(DiagCRSUndeclared, "crs",
			"a spatial representation declares no coordinate reference system")
	}
}

func checkOperation(e *evaluation, c Case) {
	if c.Binding.Operation == "" {
		e.unqualifyWith(DiagOperationUnqualified, "operation",
			"no operation qualification is declared")
		return
	}
	if !operationSet[c.Binding.Operation] {
		e.rejectWith(DiagOperationUnqualified, fmt.Sprintf(
			"operation %q is not one of the three", c.Binding.Operation))
		return
	}
	// An inverse claim is its own qualification. A read-only instance that also
	// claims an inverse is two claims at once.
	if c.Binding.Operation == OpReadOnly && c.Binding.DeclaredLosses != nil &&
		contains(c.Binding.DeclaredLosses, "aggregation-inverse") {
		e.rejectWith(DiagInverseUnqualified,
			"an aggregated instance cannot qualify an inverse")
	}
}

func checkEvidence(e *evaluation, c Case) {
	if len(c.Binding.Evidence) == 0 {
		e.unknownWith(DiagEvidenceMissing, "the binding carries no evidence")
		return
	}
	for _, ev := range c.Binding.Evidence {
		if strings.TrimSpace(ev) == "" {
			continue
		}
		e.evidence = append(e.evidence, ev)
	}
}

func contains(in []string, want string) bool {
	for _, s := range in {
		if s == want {
			return true
		}
	}
	return false
}

func sortedCopy(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if strings.TrimSpace(s) == "" {
			continue
		}
		out = append(out, s)
	}
	sort.Strings(out)
	unique := out[:0]
	for i, s := range out {
		if i > 0 && s == out[i-1] {
			continue
		}
		unique = append(unique, s)
	}
	return unique
}

func sortDiagnostics(in []Diagnostic) []Diagnostic {
	sort.SliceStable(in, func(i, j int) bool {
		if severityRank(in[i].Severity) != severityRank(in[j].Severity) {
			return severityRank(in[i].Severity) < severityRank(in[j].Severity)
		}
		if in[i].Code != in[j].Code {
			return in[i].Code < in[j].Code
		}
		return in[i].Detail < in[j].Detail
	})
	out := in[:0]
	for i, d := range in {
		if i > 0 && d == in[i-1] {
			continue
		}
		out = append(out, d)
	}
	if out == nil {
		return []Diagnostic{}
	}
	return out
}

func severityRank(s string) int {
	switch s {
	case SeverityReject:
		return 0
	case SeverityUnqualified:
		return 1
	case SeverityUnknown:
		return 2
	case SeverityInfo:
		return 3
	}
	return 4
}

func Compare(report Output, expected ExpectedFile) Comparison {
	byID := make(map[string]Result, len(report.Results))
	for _, r := range report.Results {
		byID[r.ID] = r
	}
	cmp := Comparison{SchemaVersion: ReportSchema, Total: len(expected.Expectations)}
	for _, want := range expected.Expectations {
		got, ok := byID[want.ID]
		if !ok {
			cmp.Mismatches = append(cmp.Mismatches, Mismatch{ID: want.ID, Field: "missing", Expected: "checked", Observed: "absent"})
			continue
		}
		matched := true
		if got.Verdict != want.Verdict {
			matched = false
			cmp.Mismatches = append(cmp.Mismatches, Mismatch{ID: want.ID, Field: "verdict", Expected: want.Verdict, Observed: got.Verdict})
		}
		if !equalSet(got.Unqualified, sortedCopy(want.Unqualified)) {
			matched = false
			cmp.Mismatches = append(cmp.Mismatches, Mismatch{ID: want.ID, Field: "unqualified", Expected: joinList(want.Unqualified), Observed: joinList(got.Unqualified)})
		}
		if matched {
			cmp.Matched++
		}
	}
	if len(cmp.Mismatches) == 0 {
		cmp.Status = "ok"
	} else {
		cmp.Status = "mismatch"
	}
	return cmp
}

func equalSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func joinList(in []string) string {
	if len(in) == 0 {
		return "(none)"
	}
	return strings.Join(in, ",")
}
