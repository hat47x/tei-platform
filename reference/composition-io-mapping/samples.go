package iomapping

import (
	"fmt"
	"sort"
	"tei/internal/composition"
)

type ValueState string

const (
	Present     ValueState = "present"
	Null        ValueState = "null"
	Absent      ValueState = "absent"
	Unavailable ValueState = "unavailable"
	Pending     ValueState = "pending"
)

// SampleValue is an opaque scalar fixture, not a production typed value. E2
// copies identity samples only; it does not implement numeric conversions.
type SampleValue struct {
	State  ValueState
	Scalar string
}
type Projection struct {
	Values  map[Ref]SampleValue
	Outcome string
}
type MissingError struct {
	Ref   Ref
	State ValueState
}

func (e *MissingError) Error() string {
	return fmt.Sprintf("sample %s/%s is %s", e.Ref.Slot, e.Ref.Field, e.State)
}

// ProjectSamples freshly inspects declarations and projects only mapped fields.
// The result is illustrative and cannot be used as an invocation authorization.
// Pending/unavailable values never become an absent-value business outcome.
func ProjectSamples(registry *composition.Registry, plan Plan, samples map[Ref]SampleValue) (Projection, error) {
	if report := Inspect(registry, plan); report.Verdict != composition.ProposalCandidate {
		return Projection{}, fmt.Errorf("mapping declaration is %s", report.Verdict)
	}
	out := Projection{Values: map[Ref]SampleValue{}}
	slots := map[string]Slot{}
	for _, s := range plan.Slots {
		slots[s.ID] = s
	}
	mappings := append([]Mapping(nil), plan.Mappings...)
	sort.Slice(mappings, func(i, j int) bool {
		a, b := mappings[i].To, mappings[j].To
		if a.Slot != b.Slot {
			return a.Slot < b.Slot
		}
		return a.Field < b.Field
	})
	for _, m := range mappings {
		source, _ := lookup(slots, m.From, false)
		target, _ := lookup(slots, m.To, true)
		if source.Multiplicity == List || source.Multiplicity == Set || target.Multiplicity == List || target.Multiplicity == Set {
			return Projection{}, fmt.Errorf("collection sample projection is not implemented")
		}
		v, ok := samples[m.From]
		if !ok {
			v = SampleValue{State: Absent}
		}
		if v.State != Present && v.Scalar != "" {
			return Projection{}, fmt.Errorf("non-present sample cannot contain a scalar")
		}
		switch v.State {
		case Present:
		case Null:
			if !source.Nullable || !target.Nullable {
				return Projection{}, &MissingError{m.From, Null}
			}
		case Absent:
			if m.MissingOutcome == "" {
				return Projection{}, &MissingError{m.From, Absent}
			}
			if out.Outcome != "" && out.Outcome != m.MissingOutcome {
				return Projection{}, fmt.Errorf("multiple missing outcomes require explicit control flow")
			}
			out.Outcome = m.MissingOutcome
			continue
		case Pending, Unavailable:
			return Projection{}, &MissingError{m.From, v.State}
		default:
			return Projection{}, fmt.Errorf("invalid sample state for %s/%s", m.From.Slot, m.From.Field)
		}
		out.Values[m.To] = v
	}
	if out.Outcome != "" {
		out.Values = nil
	}
	return out, nil
}
