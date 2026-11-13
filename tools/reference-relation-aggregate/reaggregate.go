package relationaggregate

import (
	"tei/internal/semantic/derivationref"
	relationsets "tei/reference/derivation-relationsets"
)

// This file is a JSON adapter only (#2593). The eligibility decision is made
// by reference/derivation-relationsets.Reaggregate; nothing is decided here.

// ReaggregationAxis is the collapsed axis: kind is "dimension" or "time".
type ReaggregationAxis struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// ReaggregationTime declares "period-flow" or "point-stock" and the shared
// as-of point of a stock.
type ReaggregationTime struct {
	Kind string `json:"kind"`
	AsOf string `json:"asOf,omitempty"`
}

// ReaggregationComponent is one intermediate-state component. Groups reuse
// Element: id is the group identity, not a fact identity.
type ReaggregationComponent struct {
	Anchor   string   `json:"anchor"`
	Relation string   `json:"relation"`
	Key      []string `json:"key"`
	Unit     string   `json:"unit,omitempty"`
	// Complete is a pointer so that an omitted field stays unknown (never read)
	// instead of being read as a definite partial retrieval.
	Complete *bool     `json:"complete,omitempty"`
	Groups   []Element `json:"groups"`
}

// ReaggregationInput is one declared reaggregation claim. An omitted field
// stays undeclared; no default is filled in.
type ReaggregationInput struct {
	MeasureRef     string                            `json:"measureRef,omitempty"`
	Law            string                            `json:"law,omitempty"`
	BaseGrain      string                            `json:"baseGrain,omitempty"`
	Scope          string                            `json:"scope,omitempty"`
	Axis           ReaggregationAxis                 `json:"axis"`
	Partition      string                            `json:"partition,omitempty"`
	Time           ReaggregationTime                 `json:"time"`
	EmptySetPolicy string                            `json:"emptySetPolicy,omitempty"`
	NullPolicy     string                            `json:"nullPolicy,omitempty"`
	Components     map[string]ReaggregationComponent `json:"components"`
}

// ReaggregationFinding is one reason the claim is not eligible.
type ReaggregationFinding struct {
	Verdict string `json:"verdict"`
	Code    string `json:"code"`
	Gap     string `json:"gap"`
	Message string `json:"message"`
}

// ReaggregationReport is the read-only result. Result and Exact are present
// only for an eligible verdict.
type ReaggregationReport struct {
	Verdict    string                 `json:"verdict"`
	Code       string                 `json:"code"`
	Gap        string                 `json:"gap,omitempty"`
	Requires   []string               `json:"requires,omitempty"`
	Findings   []ReaggregationFinding `json:"findings,omitempty"`
	Result     string                 `json:"result,omitempty"`
	ResultNull bool                   `json:"resultNull,omitempty"`
	Exact      string                 `json:"exact,omitempty"`
}

// EvaluateReaggregation converts the request and delegates the decision.
func EvaluateReaggregation(in ReaggregationInput) ReaggregationReport {
	claim := relationsets.Claim{
		MeasureRef: in.MeasureRef,
		Law:        relationsets.Law(in.Law),
		BaseGrain:  in.BaseGrain,
		Scope:      in.Scope,
		Axis:       relationsets.Axis{Name: in.Axis.Name, Kind: relationsets.AxisKind(in.Axis.Kind)},
		Partition:  relationsets.Partition(in.Partition),
		Time:       relationsets.Temporal{Kind: relationsets.TemporalKind(in.Time.Kind), AsOf: in.Time.AsOf},
		Policy:     relationsets.AggregatePolicy{EmptySet: relationsets.EmptySetPolicy(in.EmptySetPolicy), Nulls: nullPolicy(in.NullPolicy)},
		Components: make(map[relationsets.Component]relationsets.Part, len(in.Components)),
	}
	for name, component := range in.Components {
		set := relationsets.SetRef{
			Anchor:       component.Anchor,
			RelationID:   component.Relation,
			Key:          component.Key,
			Shape:        relationsets.Aggregate,
			Completeness: relationsets.Unfetched,
		}
		if component.Complete != nil {
			set.Completeness = relationsets.Partial
			if *component.Complete {
				set.Completeness = relationsets.Complete
			}
		}
		for _, group := range component.Groups {
			set.Rows = append(set.Rows, relationsets.Row{Identity: group.ID, Amount: groupAmount(group)})
		}
		claim.Components[relationsets.Component(name)] = relationsets.Part{Set: set, Unit: component.Unit}
	}

	got := relationsets.Reaggregate(claim)
	report := ReaggregationReport{
		Verdict:    string(got.Verdict),
		Code:       got.Code,
		Gap:        got.Gap,
		Exact:      got.Exact,
		ResultNull: got.Result.State == derivationref.Null,
	}
	if got.Result.State == derivationref.Present {
		report.Result = got.Result.Decimal
	}
	for _, name := range got.Requires {
		report.Requires = append(report.Requires, string(name))
	}
	for _, f := range got.Findings {
		report.Findings = append(report.Findings, ReaggregationFinding{Verdict: string(f.Verdict), Code: f.Code, Gap: f.Gap, Message: f.Message})
	}
	return report
}

// nullPolicy maps this tool's wording ("ignore") to the reference ("skip").
// Any other value is passed through so that an unknown policy is refused there.
func nullPolicy(v string) relationsets.NullPolicy {
	if v == NullIgnore {
		return relationsets.NullsSkip
	}
	return relationsets.NullPolicy(v)
}

// groupAmount maps an element state. An unavailable or failed group is a
// missing input, never NULL or zero; an unrecognized state is passed through
// and refused by the reference.
func groupAmount(e Element) derivationref.Input {
	switch e.State {
	case ElementPresent:
		return derivationref.Input{State: derivationref.Present, Decimal: e.Amount}
	case ElementNull:
		return derivationref.Input{State: derivationref.Null}
	case ElementInputUnavailable, ElementExecutionFailed:
		return derivationref.Input{State: derivationref.Missing}
	default:
		return derivationref.Input{State: derivationref.InputState(e.State)}
	}
}
