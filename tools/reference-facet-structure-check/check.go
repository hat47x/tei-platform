package facetstructurecheck

import (
	"fmt"
	"sort"
	"strings"
)

var destinationSet = map[string]bool{
	DestBasic:          true,
	DestComposite:      true,
	DestFacet:          true,
	DestPlugin:         true,
	DestRepresentation: true,
}

var relationSet = map[string]bool{
	RelIsA:         true,
	RelRefines:     true,
	RelHasFacet:    true,
	RelComposedOf:  true,
	RelRealizedBy:  true,
	RelProjectedAs: true,
	RelConvertedTo: true,
}

// gapForFacet names the facet a descriptor could declare and the gap it leaves
// when it does not. It is used by the contract self-description so a consumer
// reads the same mapping the checks use.
var gapForFacet = map[string]string{
	"unit":      GapUnitFacet,
	"scale":     GapScaleFacet,
	"namespace": GapNamespaceFacet,
	"crs":       GapCRSFacet,
	"dimension": GapDimensionFacet,
	"timeAxes":  GapTimeAxisFacet,
	"key":       GapKeyEqualityFacet,
	"arity":     GapArityFacet,
}

// Check judges every case.
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
	gaps        []string
	diagnostics []Diagnostic
	evidence    []string
	reject      bool
	unkn        bool
	destination string
}

func (e *evaluation) rejectWith(code, detail string) {
	e.reject = true
	e.diagnostics = append(e.diagnostics, Diagnostic{Code: code, Severity: SeverityReject, Detail: detail})
}

func (e *evaluation) gapWith(code, detail string) {
	e.gaps = append(e.gaps, code)
	e.diagnostics = append(e.diagnostics, Diagnostic{Code: code, Severity: SeverityGap, Detail: detail})
}

func (e *evaluation) unknownWith(code, detail string) {
	e.unkn = true
	e.diagnostics = append(e.diagnostics, Diagnostic{Code: code, Severity: SeverityUnknown, Detail: detail})
}

func (e *evaluation) info(code, detail string) {
	e.diagnostics = append(e.diagnostics, Diagnostic{Code: code, Severity: SeverityInfo, Detail: detail})
}

func checkCase(c Case, opts Options) Result {
	e := &evaluation{id: c.ID, title: c.Title, destination: c.Judgment.Destination}

	checkDestination(e, c, opts)
	checkRelations(e, c)
	checkDeclaredFacets(e, c)
	checkUnsupported(e, c)
	checkInferences(e, c, opts)
	checkCoreCounterexample(e, c, opts)
	checkEvidence(e, c)

	return Result{
		ID:          e.id,
		Title:       e.title,
		Verdict:     decideVerdict(e),
		Destination: c.Judgment.Destination,
		Gaps:        sortedCopy(e.gaps),
		Diagnostics: sortDiagnostics(e.diagnostics),
		Evidence:    sortedCopy(e.evidence),
	}
}

func decideVerdict(e *evaluation) string {
	switch {
	case e.reject:
		return VerdictRejected
	case e.unkn && len(e.gaps) == 0:
		return VerdictUnknown
	case len(e.gaps) > 0:
		return VerdictGap
	}
	return VerdictFits
}

func checkDestination(e *evaluation, c Case, opts Options) {
	d := c.Judgment.Destination
	if d == "" {
		e.unknownWith(DiagDestinationMissing, "no destination is declared, so nothing is judged")
		return
	}
	if opts.AllowCoreWithoutCounterexample && d == "core-type" {
		// Mutant: the closed five destinations are widened to admit core-type.
		return
	}
	if !destinationSet[d] {
		e.rejectWith(DiagUnknownDestination, fmt.Sprintf("destination %q is not one of the five", d))
	}
}

func checkRelations(e *evaluation, c Case) {
	for _, rel := range c.Judgment.Relations {
		if !relationSet[rel] {
			e.rejectWith(DiagRelationKindUnknown, fmt.Sprintf("relation kind %q is not one of the seven", rel))
		}
	}
	// `is-a` claims substitution. A structure that is merely compatible with
	// another's facets must use has-facet; using is-a there overstates it.
	if c.Structure.Facets.Composition != "" &&
		contains(c.Judgment.Relations, RelIsA) &&
		!contains(c.Judgment.Relations, RelHasFacet) &&
		c.Judgment.Destination == DestFacet {
		e.rejectWith(DiagRelationMisuse, "a facet destination must use has-facet, not is-a")
	}
}

func checkDeclaredFacets(e *evaluation, c Case) {
	f := c.Structure.Facets
	if f.Unit == "" && c.Structure.BaseType != "" {
		// A unit is never read from the base type. If the family needs a unit
		// and none is declared, that is a gap, not an inferred unit.
		if needsUnit(c.Structure.Family) {
			e.gapWith(GapUnitFacet, "the family needs a unit and the descriptor declares none")
		}
	}
	if needsScale(c.Structure.Family) && f.Scale == "" {
		e.gapWith(GapScaleFacet, "the family needs a quantity scale and the descriptor declares none")
	}
	if needsCRS(c.Structure.Family) && f.CRS == "" {
		e.gapWith(GapCRSFacet, "the family is spatially referenced and no CRS is declared")
	}
	if needsDimension(c.Structure.Family) && f.Dimension == "" {
		e.gapWith(GapDimensionFacet, "the family is indexed and no dimension is declared")
	}
	if needsTimeAxes(c.Structure.Family) && len(f.TimeAxes) == 0 {
		e.gapWith(GapTimeAxisFacet, "the family carries time and no axis role is declared")
	}
	if needsKey(c.Structure.Family) && f.Key == "" {
		e.gapWith(GapKeyEqualityFacet, "the family keys values and no key equality is declared")
	}
	if needsArity(c.Structure.Family) && f.Arity == "" && !contains(c.Structure.Unsupported, GapArityFacet) {
		e.gapWith(GapArityFacet, "the family has relations and no arity is declared")
	}
	if needsStateSpace(c.Structure.Family) && f.StateSpace == "" {
		e.gapWith(GapStateMachineFacet, "the family carries state and no state space is declared")
	}
	if needsPurity(c.Structure.Family) && f.Purity == "" {
		e.gapWith(GapEffectFacet, "the family carries calls and no purity or effect is declared")
	}
}

func needsUnit(family string) bool {
	switch family {
	case "quantity":
		return true
	}
	return false
}

func needsScale(family string) bool {
	return family == "quantity"
}

func needsCRS(family string) bool {
	return family == "space"
}

func needsDimension(family string) bool {
	switch family {
	case "ndarray", "math":
		return true
	}
	return false
}

func needsTimeAxes(family string) bool {
	return family == "timeseries"
}

func needsKey(family string) bool {
	return family == "map"
}

func needsArity(family string) bool {
	return family == "relation" || family == "graph"
}

func needsStateSpace(family string) bool {
	return family == "state"
}

func needsPurity(family string) bool {
	return family == "function"
}

func checkUnsupported(e *evaluation, c Case) {
	for _, u := range c.Structure.Unsupported {
		if u == "" {
			continue
		}
		// An unsupported meaning is a gap by definition: the current Source has
		// no place for it.
		e.gaps = append(e.gaps, u)
		e.diagnostics = append(e.diagnostics, Diagnostic{
			Code: DiagGapUndeclared, Severity: SeverityGap,
			Detail: "the descriptor records an unsupported meaning: " + u,
		})
	}
}

func checkInferences(e *evaluation, c Case, opts Options) {
	f := c.Structure.Facets
	if opts.InferMultiplicityFromComposition {
		// Mutant: `many` is read from the composition and multiplicity is
		// treated as settled.
		return
	}
	if opts.TreatSameBaseAsSameUnit {
		// Mutant: the base type stands in for a declared unit.
		if f.Unit == "" && c.Structure.BaseType != "" {
			e.diagnostics = append(e.diagnostics, Diagnostic{
				Code: DiagUnitInference, Severity: SeverityInfo,
				Detail: "the base type " + c.Structure.BaseType + " was read as a unit",
			})
		}
		return
	}
	// Declaring a sequence says nothing about order, uniqueness or
	// completeness. The rule applies to the families where those facets exist;
	// a function or a state machine is not a collection.
	if !collectionLike(c.Structure.Family) {
		return
	}
	if f.Composition != "" && f.Multiplicity == "" && e.destination == DestFacet {
		e.rejectWith(DiagMultiplicityInference,
			"composition "+f.Composition+" is declared while multiplicity, order and completeness are not")
	}
	if f.Order == "" && f.Composition != "" && e.destination == DestFacet {
		e.rejectWith(DiagMultiplicityInference,
			"order is not declared although the structure is a "+f.Composition)
	}
}

func collectionLike(family string) bool {
	switch family {
	case "collection", "map", "relation", "graph":
		return true
	}
	return false
}

func checkCoreCounterexample(e *evaluation, c Case, opts Options) {
	// The core destination does not exist in this closed vocabulary, so a case
	// that asks for it is a design question for the Owner, not a pass. What is
	// enforced here is the rule that the Facet description cannot be skipped
	// without a concrete counterexample.
	if c.Judgment.Destination == "core-type" {
		if opts.AllowCoreWithoutCounterexample && strings.TrimSpace(c.Judgment.CoreCounterexample) != "" {
			// Mutant: a recorded counterexample is taken as licence to add the
			// Core type, which this reference must not grant.
			return
		}
		e.rejectWith(DiagUnknownDestination,
			"core-type is not an available destination; return the request to the Owner")
		return
	}
	if strings.TrimSpace(c.Judgment.CoreCounterexample) != "" && c.Judgment.Destination == DestFacet {
		// A counterexample recorded while the destination stays a facet is
		// consistent: the counterexample is what kept the type out of Core.
		e.info(DiagCoreWithoutCounter, "a counterexample is recorded and the destination stays a facet")
	}
}

func checkEvidence(e *evaluation, c Case) {
	if len(c.Judgment.Evidence) == 0 {
		e.unknownWith(DiagEvidenceMissing, "the judgment carries no evidence")
		return
	}
	for _, ev := range c.Judgment.Evidence {
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
	case SeverityGap:
		return 1
	case SeverityUnknown:
		return 2
	case SeverityInfo:
		return 3
	}
	return 4
}

// Compare matches the checked results against the hand-written expectations.
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
		if got.Destination != want.Destination {
			matched = false
			cmp.Mismatches = append(cmp.Mismatches, Mismatch{ID: want.ID, Field: "destination", Expected: want.Destination, Observed: got.Destination})
		}
		if !equalSet(got.Gaps, sortedCopy(want.Gaps)) {
			matched = false
			cmp.Mismatches = append(cmp.Mismatches, Mismatch{ID: want.ID, Field: "gaps", Expected: joinList(want.Gaps), Observed: joinList(got.Gaps)})
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
