package derivationrelationsets

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"tei/internal/semantic/derivationref"
)

// This file is the #2593 read-only eligibility check for reducing already
// reduced rows again. It decides from the declared base grain, group identity,
// collapsed axis, law and surviving intermediate state whether the reduction
// keeps its meaning. It never decides from the numeric type or from a matching
// display value.
//
// It is not a measure master, a metric DSL, a query optimizer or a BI engine.
// Rounding and unit conversion stay with D8 #1705, SQL lowering with D2 #1640,
// provider results with #2622, authorization with D5 #1693 and the business
// acceptance of a measure with its owner. "eligible" here means only that the
// declared inputs are sufficient for the declared law on synthetic input.

// Law is the declared composition law of one reaggregation.
type Law string

const (
	LawSum   Law = "sum"   // partial sums of one disjoint partition
	LawMean  Law = "mean"  // (sum, count) pairs, divided once at the end
	LawRatio Law = "ratio" // (numerator, denominator) pairs, divided once at the end
)

// Component names the intermediate state one group row carries.
type Component string

const (
	ComponentSum         Component = "sum"
	ComponentCount       Component = "count"
	ComponentNumerator   Component = "numerator"
	ComponentDenominator Component = "denominator"
	// ComponentValue is the finalized local result (a local mean or a local
	// ratio). It is accepted as input only so that its insufficiency can be
	// reported; no law composes it.
	ComponentValue Component = "value"
)

// AxisKind separates collapsing a business dimension from collapsing time.
type AxisKind string

const (
	AxisDimension AxisKind = "dimension"
	AxisTime      AxisKind = "time"
)

// Axis is the axis the reaggregation collapses.
type Axis struct {
	Name string
	Kind AxisKind
}

// Partition declares whether the groups split the base facts without overlap
// along the collapsed axis. It is evidence supplied by the caller; it is never
// inferred from row values.
type Partition string

const (
	PartitionUndeclared  Partition = ""
	PartitionDisjoint    Partition = "disjoint"
	PartitionOverlapping Partition = "overlapping"
)

// TemporalKind separates a flow over a period from a stock at a point in time.
type TemporalKind string

const (
	TemporalUndeclared TemporalKind = ""
	PeriodFlow         TemporalKind = "period-flow"
	PointStock         TemporalKind = "point-stock"
)

// Temporal is the declared time meaning of the measure. AsOf is the shared
// point in time of a stock when a dimension is collapsed.
type Temporal struct {
	Kind TemporalKind
	AsOf string
}

// Part is one component of the intermediate state: an existing SetRef whose
// rows are the groups (Row.Identity is the group identity, not a fact
// identity) and the single declared unit of its amounts.
type Part struct {
	Set  SetRef
	Unit string
}

// Claim is one declared reaggregation. Every field is a declaration by the
// caller; an empty field stays unknown and is never filled with a default.
type Claim struct {
	MeasureRef string
	Law        Law
	BaseGrain  string // identity unit of one base fact, e.g. one order line
	Scope      string // population the result is claimed for
	Axis       Axis
	Partition  Partition
	Time       Temporal
	Policy     AggregatePolicy
	Components map[Component]Part
}

// Verdict is the three-valued outcome. Unknown is not a weaker "eligible".
type Verdict string

const (
	Eligible   Verdict = "eligible"
	Ineligible Verdict = "ineligible"
	Unknown    Verdict = "unknown"
)

// Gap reasons reuse the wording of the operation-surface gap vocabulary
// (internal/operations: missing-input / missing-semantics / conflict /
// out-of-scope) instead of introducing a second vocabulary.
const (
	GapMissingInput     = "missing-input"
	GapMissingSemantics = "missing-semantics"
	GapConflict         = "conflict"
	GapOutOfScope       = "out-of-scope"
)

// Finding codes.
const (
	CodeOK                      = "OK"
	CodeLawUndeclared           = "LAW_UNDECLARED"
	CodeLawNotOwned             = "LAW_NOT_OWNED"
	CodeIntermediateStateLost   = "INTERMEDIATE_STATE_LOST"
	CodeFactOccurrenceRepeated  = "FACT_OCCURRENCE_REPEATED"
	CodeIncompleteSet           = "INCOMPLETE_SET"
	CodeInputUnavailable        = "INPUT_UNAVAILABLE"
	CodePolicyUndeclared        = "POLICY_UNDECLARED"
	CodeBindingInvalid          = "BINDING_INVALID"
	CodeNullPairingUndefined    = "NULL_PAIRING_UNDEFINED"
	CodeUnitUndeclared          = "UNIT_UNDECLARED"
	CodeScopeMismatch           = "SCOPE_MISMATCH"
	CodeComponentGroupMismatch  = "COMPONENT_GROUPS_MISMATCH"
	CodeComponentNotDistinct    = "COMPONENT_NOT_DISTINCT"
	CodeGroupStateContradictory = "GROUP_STATE_CONTRADICTORY"
	CodeBaseGrainUndeclared     = "BASE_GRAIN_UNDECLARED"
	CodeScopeUndeclared         = "SCOPE_UNDECLARED"
	CodeAxisUndeclared          = "AXIS_UNDECLARED"
	CodePartitionUndeclared     = "PARTITION_UNDECLARED"
	CodeGroupsOverlap           = "GROUPS_OVERLAP"
	CodeTimeUndeclared          = "TIME_SEMANTICS_UNDECLARED"
	CodeAsOfUndeclared          = "AS_OF_UNDECLARED"
	CodeStockSummedAcrossTime   = "STOCK_SUMMED_ACROSS_TIME"
	CodeStockOverTimeNotOwned   = "STOCK_OVER_TIME_NOT_OWNED"
	CodeCountBasisUndeclared    = "COUNT_BASIS_UNDECLARED"
	CodeZeroDenominator         = "ZERO_DENOMINATOR_UNDEFINED"
)

// Finding is one reason the claim is not eligible.
type Finding struct {
	Verdict Verdict
	Code    string
	Gap     string
	Message string
}

// Report is the read-only diagnostic. Result and Exact are set only when the
// verdict is eligible. Exact is the unrounded rational ("3/4", "18"); Result
// carries the same value as a decimal only when it terminates, because rounding
// is not owned here.
type Report struct {
	Verdict  Verdict
	Code     string
	Gap      string
	Requires []Component
	Findings []Finding
	Result   derivationref.Input
	Exact    string
	// Unit is the unit of an eligible result, composed from the declared
	// component units without cancellation or conversion ("point/observation",
	// "USD/EUR"). Whether that unit is meaningful stays with D8 #1705.
	Unit string
}

func resultUnit(c Claim, required []Component) string {
	unit := c.Components[required[0]].Unit
	if len(required) > 1 {
		unit += "/" + c.Components[required[1]].Unit
	}
	return unit
}

func requiredComponents(law Law) []Component {
	switch law {
	case LawSum:
		return []Component{ComponentSum}
	case LawMean:
		return []Component{ComponentSum, ComponentCount}
	case LawRatio:
		return []Component{ComponentNumerator, ComponentDenominator}
	default:
		return nil
	}
}

// Reaggregate checks one declared claim and, only when every required
// declaration and intermediate state is present and consistent, composes the
// result. A definite violation yields Ineligible; an absent declaration yields
// Unknown. Neither returns a value, so a naive result cannot leak through.
func Reaggregate(c Claim) Report {
	if strings.TrimSpace(string(c.Law)) == "" {
		return failed(nil, Finding{Unknown, CodeLawUndeclared, GapMissingSemantics, "no composition law is declared; an operator name or a numeric type does not imply one"})
	}
	required := requiredComponents(c.Law)
	if required == nil {
		return failed(nil, Finding{Unknown, CodeLawNotOwned, GapOutOfScope, fmt.Sprintf("law %q is not composed here; it needs the original facts or a qualified intermediate representation", c.Law)})
	}
	var missing []string
	for _, name := range required {
		if _, ok := c.Components[name]; !ok {
			missing = append(missing, string(name))
		}
	}
	if len(missing) > 0 {
		message := fmt.Sprintf("law %q needs %s; the intermediate state is not available", c.Law, strings.Join(missing, ", "))
		if _, ok := c.Components[ComponentValue]; ok {
			message += "; finalized local values cannot be composed and are not averaged or summed instead"
		}
		return failed(required, Finding{Ineligible, CodeIntermediateStateLost, GapMissingInput, message})
	}

	var findings []Finding
	add := func(v Verdict, code, gap, format string, args ...any) {
		findings = append(findings, Finding{v, code, gap, fmt.Sprintf(format, args...)})
	}
	totals := make(map[Component]derivationref.Input, len(required))
	paired := len(required) > 1
	for _, name := range required {
		part := c.Components[name]
		if strings.TrimSpace(part.Unit) == "" {
			add(Unknown, CodeUnitUndeclared, GapMissingSemantics, "component %s has no declared unit", name)
		}
		if paired && c.Policy.Nulls == NullsSkip && hasNull(part.Set) {
			add(Unknown, CodeNullPairingUndefined, GapMissingSemantics, "component %s has a NULL group; skipping it in one component of a pair is not defined here", name)
			continue
		}
		total, err := sumIdentified(part.Set, c.Policy)
		switch {
		case err == nil && name == ComponentCount && !countsAreNaturals(part.Set):
			add(Unknown, CodeBindingInvalid, GapMissingSemantics, "component %s must hold non-negative whole numbers of base facts", name)
		case err == nil:
			totals[name] = total
		case errors.Is(err, ErrFanOut):
			add(Ineligible, CodeFactOccurrenceRepeated, GapConflict, "component %s: %v; a repeated occurrence is neither counted again nor removed by value", name, err)
		case errors.Is(err, ErrIncomplete) && part.Set.Completeness == Partial:
			add(Ineligible, CodeIncompleteSet, GapMissingInput, "component %s: %v", name, err)
		case errors.Is(err, ErrIncomplete):
			add(Unknown, CodeInputUnavailable, GapMissingInput, "component %s: %v", name, err)
		case errors.Is(err, ErrNoPolicy):
			add(Unknown, CodePolicyUndeclared, GapMissingSemantics, "component %s: %v", name, err)
		default:
			add(Unknown, CodeBindingInvalid, GapMissingSemantics, "component %s: %v", name, err)
		}
	}
	if paired {
		first := c.Components[required[0]].Set
		for _, name := range required[1:] {
			other := c.Components[name].Set
			if first.Anchor != other.Anchor {
				add(Ineligible, CodeScopeMismatch, GapConflict, "components %s and %s are anchored to different populations", required[0], name)
			}
			// Group pairing is only comparable when both sets are identified.
			if Bind(first) == nil && Bind(other) == nil {
				if !sameGroups(first, other) || !sameKey(first, other) {
					add(Ineligible, CodeComponentGroupMismatch, GapConflict, "components %s and %s do not cover the same groups under the same key, so their pairing is lost", required[0], name)
				}
				if first.RelationID == other.RelationID {
					add(Ineligible, CodeComponentNotDistinct, GapConflict, "components %s and %s are the same relation %s; one set cannot stand for two intermediate states", required[0], name, first.RelationID)
				}
			}
		}
		if c.Law == LawMean {
			if group, ok := contradictoryMeanGroup(c.Components[ComponentSum].Set, c.Components[ComponentCount].Set); ok {
				add(Ineligible, CodeGroupStateContradictory, GapConflict, "group %s has a non-zero sum over zero facts", group)
			}
		}
	}

	if strings.TrimSpace(c.BaseGrain) == "" {
		add(Unknown, CodeBaseGrainUndeclared, GapMissingSemantics, "the identity unit of one base fact is not declared")
	}
	if strings.TrimSpace(c.Scope) == "" {
		add(Unknown, CodeScopeUndeclared, GapMissingSemantics, "the population the result is claimed for is not declared")
	}
	axisKnown := strings.TrimSpace(c.Axis.Name) != "" && (c.Axis.Kind == AxisDimension || c.Axis.Kind == AxisTime)
	if !axisKnown {
		add(Unknown, CodeAxisUndeclared, GapMissingSemantics, "the collapsed axis and its kind are not declared")
	}
	switch c.Partition {
	case PartitionDisjoint:
	case PartitionOverlapping:
		add(Ineligible, CodeGroupsOverlap, GapConflict, "the groups share base facts along %q; group results cannot be combined without the fact identities", c.Axis.Name)
	default:
		add(Unknown, CodePartitionUndeclared, GapMissingSemantics, "it is not declared whether the groups split the base facts without overlap")
	}
	switch c.Time.Kind {
	case PeriodFlow:
	case PointStock:
		switch {
		case !axisKnown:
		case c.Axis.Kind == AxisTime && c.Law == LawSum:
			add(Ineligible, CodeStockSummedAcrossTime, GapConflict, "a point-in-time stock is not summable across %q", c.Axis.Name)
		case c.Axis.Kind == AxisTime:
			add(Unknown, CodeStockOverTimeNotOwned, GapOutOfScope, "reducing a stock over time needs a declared temporal operator, which is not owned here")
		case strings.TrimSpace(c.Time.AsOf) == "":
			add(Unknown, CodeAsOfUndeclared, GapMissingSemantics, "a stock needs one shared as-of point to be combined across %q", c.Axis.Name)
		}
	default:
		add(Unknown, CodeTimeUndeclared, GapMissingSemantics, "it is not declared whether the measure is a period flow or a point-in-time stock")
	}
	if c.Law == LawMean && c.Policy.Nulls == NullsUndeclared {
		add(Unknown, CodeCountBasisUndeclared, GapMissingSemantics, "the NULL basis shared by sum and count is not declared")
	}
	if len(findings) > 0 {
		return failed(required, findings...)
	}

	for _, name := range required {
		if totals[name].State == derivationref.Null {
			return Report{Verdict: Eligible, Code: CodeOK, Requires: required, Result: derivationref.Input{State: derivationref.Null}, Unit: resultUnit(c, required)}
		}
	}
	value, _ := new(big.Rat).SetString(totals[required[0]].Decimal)
	if paired {
		divisor, _ := new(big.Rat).SetString(totals[required[1]].Decimal)
		if divisor.Sign() == 0 {
			return failed(required, Finding{Unknown, CodeZeroDenominator, GapMissingSemantics, fmt.Sprintf("total %s is zero; the result is not defined here", required[1])})
		}
		value.Quo(value, divisor)
	}
	report := Report{Verdict: Eligible, Code: CodeOK, Requires: required, Exact: value.RatString(), Unit: resultUnit(c, required)}
	if decimal, ok := terminatingDecimal(value); ok {
		report.Result = derivationref.Input{State: derivationref.Present, Decimal: decimal}
	}
	return report
}

// failed builds a report without a value. A definite violation outranks a
// missing declaration.
func failed(required []Component, findings ...Finding) Report {
	head := findings[0]
	for _, f := range findings {
		if f.Verdict == Ineligible {
			head = f
			break
		}
	}
	return Report{Verdict: head.Verdict, Code: head.Code, Gap: head.Gap, Requires: required, Findings: findings}
}

func hasNull(set SetRef) bool {
	for _, row := range set.Rows {
		if row.Amount.State == derivationref.Null {
			return true
		}
	}
	return false
}

func sameKey(a, b SetRef) bool {
	if len(a.Key) != len(b.Key) {
		return false
	}
	for i := range a.Key {
		if a.Key[i] != b.Key[i] {
			return false
		}
	}
	return true
}

// contradictoryMeanGroup reports a group whose sum is non-zero although its
// count is zero. Such a pair cannot come from any set of facts, so composing
// it would hide a broken intermediate state behind a plausible total.
func contradictoryMeanGroup(sum, count SetRef) (string, bool) {
	counts := make(map[string]*big.Rat, len(count.Rows))
	for _, row := range count.Rows {
		if row.Amount.State == derivationref.Present {
			if v, ok := parsePlainDecimal(row.Amount.Decimal); ok {
				counts[row.Identity] = v
			}
		}
	}
	for _, row := range sum.Rows {
		if row.Amount.State != derivationref.Present {
			continue
		}
		s, ok := parsePlainDecimal(row.Amount.Decimal)
		if n, found := counts[row.Identity]; ok && found && n.Sign() == 0 && s.Sign() != 0 {
			return row.Identity, true
		}
	}
	return "", false
}

func countsAreNaturals(set SetRef) bool {
	for _, row := range set.Rows {
		if row.Amount.State != derivationref.Present {
			continue
		}
		v, ok := parsePlainDecimal(row.Amount.Decimal)
		if !ok || !v.IsInt() || v.Sign() < 0 {
			return false
		}
	}
	return true
}

func sameGroups(a, b SetRef) bool {
	ids := func(set SetRef) []string {
		out := make([]string, 0, len(set.Rows))
		for _, row := range set.Rows {
			out = append(out, row.Identity)
		}
		sort.Strings(out)
		return out
	}
	left, right := ids(a), ids(b)
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// terminatingDecimal renders v exactly when its reduced denominator has no
// prime factor other than 2 and 5. Otherwise the caller keeps the rational.
func terminatingDecimal(v *big.Rat) (string, bool) {
	den := new(big.Int).Set(v.Denom())
	scale := 0
	for _, prime := range []int64{2, 5} {
		p, rem := big.NewInt(prime), new(big.Int)
		count := 0
		for {
			q, r := new(big.Int).QuoRem(den, p, rem)
			if r.Sign() != 0 {
				break
			}
			den, count = q, count+1
		}
		if count > scale {
			scale = count
		}
	}
	if den.Cmp(big.NewInt(1)) != 0 {
		return "", false
	}
	return v.FloatString(scale), true
}
