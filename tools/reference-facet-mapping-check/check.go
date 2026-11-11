package facetmappingcheck

import (
	"fmt"
	"sort"
	"strings"
)

// nonInvertibleLosses are the losses that make an edit inverse unusable no
// matter how the operation column is declared. A read-only projection that
// drops one of these cannot be written back row by row.
var nonInvertibleLosses = map[string]bool{
	LossOrder:              true,
	LossOccurrenceIdentity: true,
	LossEdgeMultiplicity:   true,
	LossParentArity:        true,
	LossHiddenEdge:         true,
	LossGrain:              true,
	LossFactIdentity:       true,
	LossAggregationInverse: true,
	LossPresenceStates:     true,
	LossCoverage:           true,
	LossSelectionIdentity:  true,
	LossSourceRevision:     true,
	LossInjectivity:        true,
	LossQuantityScale:      true,
	LossTimeAxisRole:       true,
	LossFanout:             true,
}

// Check evaluates every case and returns the report. It reads no external
// state: every observation comes from the case file.
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
	losses      []string
	diagnostics []Diagnostic
	evidence    []string
	unknowns    []string
	claims      int
	evidenced   int
	reject      bool
	unqual      bool
	unkn        bool
}

func (e *evaluation) rejectWith(code, detail string) {
	e.reject = true
	e.diagnostics = append(e.diagnostics, Diagnostic{Code: code, Severity: SeverityReject, Detail: detail})
}

func (e *evaluation) unqualifyWith(code, detail string) {
	e.unqual = true
	e.diagnostics = append(e.diagnostics, Diagnostic{Code: code, Severity: SeverityUnqualified, Detail: detail})
}

func (e *evaluation) unknownWith(code, detail string) {
	e.unkn = true
	e.diagnostics = append(e.diagnostics, Diagnostic{Code: code, Severity: SeverityUnknown, Detail: detail})
}

// contradict records a preservation claim that the observed loss refutes. It
// stays informational: the loss itself already decides the verdict, and a
// contradiction must not be read as a permission decision.
func (e *evaluation) contradict(detail string) {
	e.diagnostics = append(e.diagnostics, Diagnostic{
		Code: DiagClaimContradicted, Severity: SeverityInfo, Detail: detail,
	})
}

func (e *evaluation) info(code, detail string) {
	e.diagnostics = append(e.diagnostics, Diagnostic{Code: code, Severity: SeverityInfo, Detail: detail})
}

func (e *evaluation) lose(code string) {
	e.losses = append(e.losses, code)
}

// claim records one preservation claim. The number of claims and the number of
// claims that carry evidence decide whether an empty verdict is preserved or
// unknown: a claim without evidence is never promoted to preserved.
func (e *evaluation) claim(c Claim, name string) {
	e.claims++
	if !c.ClaimedPreserved {
		return
	}
	if !evidenceUsable(c.Evidence) {
		e.unknownWith(DiagEvidenceMissing,
			fmt.Sprintf("%s claims preservation without evidence", name))
		e.unknowns = append(e.unknowns, name+":unEvidenced")
		return
	}
	e.evidenced++
	for _, ev := range c.Evidence {
		if strings.TrimSpace(ev) == "" {
			continue
		}
		e.evidence = append(e.evidence, name+":"+ev)
	}
}

func evidenceUsable(evidence []string) bool {
	for _, ev := range evidence {
		if strings.TrimSpace(ev) != "" {
			return true
		}
	}
	return false
}

func checkCase(c Case, opts Options) Result {
	e := &evaluation{id: c.ID, title: c.Title}

	checkOrder(e, c, opts)
	checkIdentity(e, c)
	checkMultiplicity(e, c, opts)
	checkGrain(e, c, opts)
	checkPresence(e, c, opts)
	checkQuantity(e, c, opts)
	checkTime(e, c)
	checkCoverage(e, c)
	checkRevision(e, c)
	checkComposite(e, c)

	verdict := decideVerdict(e)
	inverse := decideInverse(e, c, verdict)

	losses := sortedCopy(e.losses)
	checkLossDeclaration(e, c, losses)

	return Result{
		ID:          e.id,
		Title:       e.title,
		Verdict:     verdict,
		Losses:      losses,
		Diagnostics: sortDiagnostics(e.diagnostics),
		Inverse:     inverse,
		Evidence:    sortedCopy(e.evidence),
		Unknowns:    sortedCopy(e.unknowns),
	}
}

// checkLossDeclaration compares the losses the checker observed with the ones
// the contract declares. A loss that the author did not declare is reported
// because an understated contract is the failure mode the column exists for;
// it never changes the verdict, which is about what the mapping actually does.
func checkLossDeclaration(e *evaluation, c Case, observed []string) {
	declared := map[string]bool{}
	for _, d := range c.Mapping.Loss.Declared {
		if strings.TrimSpace(d) == "" {
			continue
		}
		declared[d] = true
		if !containsObserved(observed, d) {
			e.info(DiagLossUndeclared, "declared loss "+d+" was not observed by the check")
		}
	}
	for _, loss := range observed {
		if !declared[loss] {
			e.info(DiagLossUndeclared, "observed loss "+loss+" is not declared in the contract")
		}
	}
}

func containsObserved(observed []string, want string) bool {
	for _, l := range observed {
		if l == want {
			return true
		}
	}
	return false
}

func decideVerdict(e *evaluation) string {
	switch {
	case e.reject:
		return VerdictRejected
	case e.unqual:
		return VerdictUnqualified
	case e.unkn && len(e.losses) == 0:
		return VerdictUnknown
	case len(e.losses) > 0:
		return VerdictLossy
	case e.claims == 0:
		// Nothing was claimed at all: the check observed no obligation to keep.
		return VerdictUnknown
	case e.evidenced == 0:
		return VerdictUnknown
	}
	return VerdictPreserved
}

// decideInverse reports the edit-inverse qualification separately from the
// read-only verdict. A successful projection never implies a usable inverse, so
// the answer here is the inverse column itself, not a mapping verdict.
func decideInverse(e *evaluation, c Case, verdict string) string {
	if !c.Mapping.Operation.InverseEligible {
		return InverseNotClaimed
	}
	if verdict != VerdictPreserved && verdict != VerdictLossy {
		e.info(DiagInverseUnqualified,
			"inverse is declared eligible while the mapping verdict is "+verdict)
		return InverseIneligible
	}
	for _, loss := range e.losses {
		if nonInvertibleLosses[loss] {
			e.info(DiagInverseUnqualified,
				"inverse is declared eligible while "+loss+" is lost")
			return InverseIneligible
		}
	}
	return InverseEligible
}

func checkOrder(e *evaluation, c Case, opts Options) {
	src := c.Structure.Facets.Order
	dst := c.Representation.Facets.Order
	if src == "" {
		return
	}
	if opts.IgnoreOrder {
		// Mutant: pretend an ordered source survives an unordered target.
		e.claim(c.Mapping.Order, "order")
		return
	}
	if src == "ordered" && dst == "unordered" {
		e.lose(LossOrder)
		if c.Mapping.Order.ClaimedPreserved {
			e.contradict("an unordered target cannot carry the source order")
		}
	}
	if src == "ordered" && dst == "ordered" &&
		c.Structure.Facets.OrderOrigin != "" && c.Representation.Facets.OrderOrigin != "" &&
		c.Structure.Facets.OrderOrigin != c.Representation.Facets.OrderOrigin {
		// An order the representation fabricates is not the source order.
		e.lose(LossOrder)
	}
	e.claim(c.Mapping.Order, "order")
}

func checkIdentity(e *evaluation, c Case) {
	kind := c.Mapping.Identity.Kind
	lossyIdentity := false
	switch kind {
	case "1:1", "":
	case "1:N", "N:1", "N:M", "partial":
		lossyIdentity = true
		e.lose(LossOccurrenceIdentity)
	default:
		lossyIdentity = true
		e.lose(LossOccurrenceIdentity)
		e.unknownWith(DiagEvidenceMissing,
			fmt.Sprintf("identity kind %q is not a known correspondence", kind))
	}
	if lossyIdentity && c.Mapping.Identity.ClaimedPreserved {
		e.rejectWith(DiagClaimContradicted,
			"correspondence "+kind+" cannot preserve individual identity")
	}
	e.claim(Claim{ClaimedPreserved: c.Mapping.Identity.ClaimedPreserved, Evidence: c.Mapping.Identity.Evidence}, "identity")

	// A presentation selection is not a data identity. Reject the reading that
	// turns a selected row into an editable identity.
	if kind == "" && c.Mapping.Target.Selection != "" {
		e.rejectWith(DiagSelectionIdentity,
			"presentation selection is used where a data identity is required")
		e.lose(LossSelectionIdentity)
	}
}

func checkMultiplicity(e *evaluation, c Case, opts Options) {
	m := c.Mapping.Multiset
	if m.Fanout > 1 {
		// A fanout means the rendered row set is not the source row set.
		e.lose(LossFanout)
		if m.ClaimedPreserved {
			e.rejectWith(DiagClaimContradicted, fmt.Sprintf(
				"fanout %d multiplies rows while multiplicity is claimed preserved", m.Fanout))
		} else {
			e.unqualifyWith(DiagFanoutUndeclared, fmt.Sprintf(
				"fanout %d multiplies rows and no fanout policy is declared", m.Fanout))
		}
	} else if m.ClaimedPreserved && !evidenceUsable(m.Evidence) {
		e.unknownWith(DiagEvidenceMissing, "multiset claims preservation without evidence")
	}
	// A relation the mapping does not consume is not part of its fanout: a
	// second relation whose rows are excluded by an explicit predicate is not
	// a joined relation. Only an observed multiplication is reported here.
	if len(c.Structure.Relations) > 1 && m.Fanout > 1 {
		e.lose(LossRelationArity)
	}
	if m.ClaimedPreserved || m.Fanout > 0 {
		e.claim(Claim{ClaimedPreserved: m.ClaimedPreserved, Evidence: m.Evidence}, "multiset")
	}

	// A bag rendered as a set drops the occurrences that made it a bag.
	srcMulti := c.Structure.Facets.Multiplicity
	dstMulti := c.Representation.Facets.Multiplicity
	dedup := (srcMulti == "multi" || srcMulti == "bag" || srcMulti == "multiset") &&
		(dstMulti == "unique" || dstMulti == "set" || dstMulti == "single")
	if dedup {
		e.lose(LossOccurrenceIdentity)
	}

	// A multi-edge relation rendered as a tree loses edge occurrences, parent
	// arity, and the edges the tree cannot show.
	multigraph := strings.HasPrefix(c.Structure.Family, "graph") &&
		(c.Structure.Facets.Composition == "multigraph" || c.Structure.Facets.Composition == "multi")
	if multigraph && c.Representation.Kind == "tree" {
		e.lose(LossEdgeMultiplicity)
		e.lose(LossParentArity)
		e.lose(LossHiddenEdge)
	}
}

func checkGrain(e *evaluation, c Case, opts Options) {
	g := c.Mapping.Grain
	changed := g.SourceGrain != "" && g.TargetGrain != "" && g.SourceGrain != g.TargetGrain
	aggregated := g.Steps > 0 || changed
	if aggregated {
		e.lose(LossGrain)
		e.lose(LossFactIdentity)
		if g.ClaimedPreserved {
			e.contradict("aggregation changes grain; it cannot claim grain preservation")
		}
		if c.Mapping.Operation.InverseEligible && !opts.AllowAggregateInverse {
			// An aggregate row has no unique source row to write back to.
			e.lose(LossAggregationInverse)
			e.rejectWith(DiagGrainWriteback,
				"an aggregated value cannot be written back to the source rows")
		}
	}
	if aggregated || g.ClaimedPreserved {
		e.claim(Claim{ClaimedPreserved: g.ClaimedPreserved, Evidence: g.Evidence}, "grain")
	}
}

func checkPresence(e *evaluation, c Case, opts Options) {
	srcStates := c.Structure.Facets.Presence
	dstStates := c.Representation.Facets.Presence
	if len(srcStates) > 0 && len(dstStates) > 0 {
		if opts.BinarizePresence {
			// Mutant: pretend the four presence states collapse without loss.
			e.claim(c.Mapping.Presence, "presence")
			return
		}
		if !equalStateSet(srcStates, dstStates) {
			e.lose(LossPresenceStates)
			if c.Mapping.Presence.ClaimedPreserved {
				e.contradict("presence states are collapsed but preservation is claimed")
			}
		}
	}
	e.claim(c.Mapping.Presence, "presence")
}

func equalStateSet(a, b []string) bool {
	return equalSet(sortedCopy(a), sortedCopy(b))
}

func checkQuantity(e *evaluation, c Case, opts Options) {
	q := c.Mapping.Quantity

	// An approximate conversion never reports preservation on its own. The
	// planner asked for this explicitly: a rounding or an approximation is
	// reported as a loss, and claiming preservation through it is refused.
	if q.Approximation != "" && q.Approximation != "none" {
		e.lose(LossPrecision)
		if q.ClaimedPreserved {
			e.rejectWith(DiagApproximationClaimed, fmt.Sprintf(
				"approximation %q is reported as a preserved conversion", q.Approximation))
		}
	}

	scaleConflict := q.SourceScale != "" && q.TargetScale != "" && q.SourceScale != q.TargetScale
	if scaleConflict {
		if opts.IgnoreQuantityScale {
			// Mutant: pretend a point and a delta share an arithmetic contract.
		} else {
			e.lose(LossQuantityScale)
			e.rejectWith(DiagScaleConflated, fmt.Sprintf(
				"quantity scale %q cannot be treated as %q", q.SourceScale, q.TargetScale))
		}
	}

	unitChanged := q.SourceUnit != "" && q.TargetUnit != "" && q.SourceUnit != q.TargetUnit
	if unitChanged {
		e.lose(LossUnitConversion)
		switch {
		case q.ConversionPolicy == "":
			e.unqualifyWith(DiagUnitUndeclared,
				fmt.Sprintf("unit %q to %q needs a conversion policy", q.SourceUnit, q.TargetUnit))
		case q.ConversionPolicy == "currency" && q.ConversionInstant == "" && q.RateRevision == "":
			e.unqualifyWith(DiagUnitUndeclared,
				"currency conversion needs an instant or a rate revision")
		}
	}

	crsChanged := q.SourceCRS != "" && q.TargetCRS != "" && q.SourceCRS != q.TargetCRS
	if crsChanged {
		e.lose(LossCRS)
		if q.CRSTransform == "" {
			e.unqualifyWith(DiagCRSUndeclared,
				fmt.Sprintf("crs %q to %q needs a declared transform and axis order", q.SourceCRS, q.TargetCRS))
		}
	}

	if q.Precision != "" && q.TargetPrecision != "" && q.Precision != q.TargetPrecision {
		e.lose(LossPrecision)
	}

	quantified := scaleConflict || unitChanged || crsChanged ||
		q.Precision != "" || q.TargetPrecision != "" ||
		q.SourceUnit != "" || q.SourceScale != "" || q.SourceCRS != ""
	if quantified {
		e.claim(Claim{ClaimedPreserved: q.ClaimedPreserved, Evidence: q.Evidence}, "quantity")
	}
}

func checkTime(e *evaluation, c Case) {
	t := c.Mapping.Time
	if len(t.SourceAxes) == 0 && len(t.TargetAxes) == 0 {
		return
	}
	axisMismatch := len(t.SourceAxes) > 0 && len(t.TargetAxes) > 0 && !equalStateSet(t.SourceAxes, t.TargetAxes)
	if axisMismatch {
		e.lose(LossTimeAxisRole)
		e.rejectWith(DiagAxisConflated, fmt.Sprintf(
			"time axes %v cannot be reported as %v", t.SourceAxes, t.TargetAxes))
	}
	clockUnknown := (t.SourceClock == "" || t.SourceClock == "unknown") &&
		(t.TargetClock == "" || t.TargetClock == "unknown")
	if clockUnknown {
		e.unqualifyWith(DiagSyncUnknown, "clock or timebase is unknown; synchronization cannot be claimed")
	}
	if t.ClaimedPreserved && t.SyncError == "" {
		e.unqualifyWith(DiagSyncUnknown, "preservation is claimed without a synchronization error bound")
	}
	e.claim(Claim{ClaimedPreserved: t.ClaimedPreserved, Evidence: t.Evidence}, "time")
}

func checkCoverage(e *evaluation, c Case) {
	cv := c.Mapping.Coverage
	if cv.Claimed == "" {
		return
	}
	if cv.Claimed == "complete" && cv.Observed != "complete" {
		e.lose(LossCoverage)
		e.rejectWith(DiagCoverageOverclaim, fmt.Sprintf(
			"coverage is claimed complete while only %q was observed", cv.Observed))
		return
	}
	if cv.Observed == "partial" || cv.Observed == "unknown" {
		e.lose(LossCoverage)
		if len(cv.NegativeDependency) == 0 {
			e.unknownWith(DiagCoverageOverclaim, "a partial read has no recorded negative dependency")
		}
	}
}

func checkRevision(e *evaluation, c Case) {
	src := c.Mapping.Source.Revision
	dst := c.Mapping.Target.Revision
	if src != "" && dst != "" && src != dst {
		e.lose(LossSourceRevision)
		e.rejectWith(DiagRevisionStale, fmt.Sprintf(
			"source revision %q does not match the rendered revision %q", src, dst))
	}
	// A representation instance belongs to exactly one Function Data Context.
	// Moving the same type into another Context keeps neither the Scope nor the
	// Grant that the old Context held.
	repCtx := c.Representation.ContextID
	tgtCtx := c.Mapping.Target.ContextID
	if repCtx != "" && tgtCtx != "" && repCtx != tgtCtx {
		e.lose(LossContextIdentity)
		e.rejectWith(DiagContextMismatch, fmt.Sprintf(
			"representation is owned by context %q but the mapping targets %q", repCtx, tgtCtx))
	}
	repEpoch := c.Representation.Epoch
	tgtEpoch := c.Mapping.Target.Epoch
	if repEpoch != "" && tgtEpoch != "" && repEpoch != tgtEpoch {
		e.lose(LossContextIdentity)
		e.rejectWith(DiagContextMismatch, fmt.Sprintf(
			"representation epoch %q does not match the mapping epoch %q", repEpoch, tgtEpoch))
	}
}

func checkComposite(e *evaluation, c Case) {
	if c.Composite == nil {
		return
	}
	cp := *c.Composite
	if cp.RepresentationEqual && !cp.SourceEqual {
		e.lose(LossInjectivity)
		if cp.InverseFromRepresentation {
			e.rejectWith(DiagInjectivityUnproven,
				"equal representation bytes from different sources do not restore the source")
			e.lose(LossSelectionIdentity)
		}
	}
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
