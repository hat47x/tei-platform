package iomapping

import (
	"errors"
	"reflect"
	"testing"

	"tei/internal/composition"
)

var sourceContract = composition.Contract{ID: "source", Version: "1", Shape: "source-shape"}
var targetContract = composition.Contract{ID: "target", Version: "1", Shape: "target-shape"}

func field(id string) Field {
	return Field{ID: id, Type: Type{Name: "text", Unit: "none"}, Multiplicity: One, Classification: "public"}
}
func fixture(t *testing.T) (*composition.Registry, Plan) {
	t.Helper()
	r := composition.NewRegistry()
	_, err := r.Apply(0, []composition.Provider{{ID: "source-p", Scope: "app", Offers: sourceContract}, {ID: "target-p", Scope: "app", Offers: targetContract}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := Plan{Scope: "app", Slots: []Slot{
		{ID: "producer", Contract: sourceContract, Multiplicity: One, Outputs: []Field{field("value"), field("unused")}},
		{ID: "consumer", Contract: targetContract, Multiplicity: One, Inputs: []Field{field("value")}},
	}, Mappings: []Mapping{{From: Ref{"producer", sourceContract, "value"}, To: Ref{"consumer", targetContract, "value"}}}}
	return r, p
}
func has(report Report, code string) bool {
	for _, d := range report.Diagnostics {
		if d.Code == code {
			return true
		}
	}
	return false
}
func TestDataOrderUsesExistingRegistry(t *testing.T) {
	r, p := fixture(t)
	before := r.Snapshot()
	got := Inspect(r, p)
	if got.Verdict != composition.ProposalCandidate || !reflect.DeepEqual(got.Order, []string{"producer", "consumer"}) {
		t.Fatalf("D03: %+v", got)
	}
	if !reflect.DeepEqual(before, r.Snapshot()) {
		t.Fatal("inspection changed registry")
	}
	if got.Wiring.Findings[0].Selected == nil || len(got.Wiring.Findings[0].VerificationTargets) != 1 {
		t.Fatal("external verification subjects lost")
	}
	_, err := r.Apply(before.Revision, nil, []string{"source-p"})
	if err != nil {
		t.Fatal(err)
	}
	got = Inspect(r, p)
	if got.Verdict != composition.ProposalUnknown || len(got.Order) != 0 || !has(got, "no-known-provider") {
		t.Fatalf("D04: %+v", got)
	}
}
func TestRegistrationOrderAndLocalNames(t *testing.T) {
	r, p := fixture(t)
	same := composition.NewRegistry()
	_, err := same.Apply(0, []composition.Provider{{ID: "target-p", Scope: "app", Offers: targetContract}, {ID: "source-p", Scope: "app", Offers: sourceContract}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(Inspect(r, p), Inspect(same, p)) {
		t.Fatal("D02: registration order changed report")
	}
	p.Slots = append(p.Slots, Slot{ID: "second-consumer", Contract: targetContract, Multiplicity: One, Inputs: []Field{field("value")}})
	p.Mappings = append(p.Mappings, Mapping{From: p.Mappings[0].From, To: Ref{"second-consumer", targetContract, "value"}})
	if got := Inspect(r, p); got.Verdict != composition.ProposalCandidate || len(got.Wiring.Findings) != 3 {
		t.Fatalf("independent local slots: %+v", got)
	}
	_, err = r.Apply(r.Snapshot().Revision, []composition.Provider{{ID: "source-p2", Scope: "app", Offers: sourceContract}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := Inspect(r, p); !has(got, "ambiguous-providers") || got.Verdict != composition.ProposalConflict {
		t.Fatalf("D01: %+v", got)
	}
	p.Slots[0].ProviderID = "source-p2"
	if got := Inspect(r, p); got.Verdict != composition.ProposalCandidate {
		t.Fatalf("explicit choice: %+v", got)
	}
}
func TestMappingNegativeCases(t *testing.T) {
	cases := []struct {
		name, code string
		edit       func(*Plan)
	}{
		{"D05", "invalid-ordering", func(p *Plan) { p.After = []Ordering{{Before: "producer", After: "consumer", Reason: " "}} }},
		{"cycle", "data-order-cycle", func(p *Plan) { p.After = []Ordering{{Before: "consumer", After: "producer", Reason: "cleanup"}} }},
		{"M01", "explicit-conversion-or-missing-policy-required", func(p *Plan) { p.Slots[0].Outputs[0].Type.Name = "decimal"; p.Slots[1].Inputs[0].Type.Name = "integer" }},
		{"M02", "explicit-conversion-or-missing-policy-required", func(p *Plan) { p.Slots[0].Outputs[0].Multiplicity = Optional }},
		{"M04", "type-incompatible", func(p *Plan) { p.Slots[0].Outputs[0].Classification = "personal" }},
		{"M05", "context-override", func(p *Plan) { p.Slots[1].Context = []Field{field("principal")}; p.Mappings[0].To.Field = "principal" }},
		{"M06", "type-incompatible", func(p *Plan) {
			p.Slots[0].Outputs[0].Type = Type{Name: "decimal", Unit: "JPY"}
			p.Slots[1].Inputs[0].Type = Type{Name: "decimal", Unit: "USD"}
		}},
		{"M07", "type-undetermined", func(p *Plan) { p.Slots[0].Outputs[0].Type.Unit = "" }},
		{"M08", "unknown-field-reference", func(p *Plan) { p.Mappings[0].From.Contract = targetContract }},
		{"M09", "external-transform-forbidden", func(p *Plan) { p.Mappings[0].ReadsExternal = true }},
		{"unexamined-operator", "operator-not-evaluated", func(p *Plan) { p.Mappings[0].Operator = "round@1" }},
		{"duplicate", "duplicate-destination", func(p *Plan) { p.Mappings = append(p.Mappings, p.Mappings[0]) }},
		{"new-required-input", "required-input-unmapped", func(p *Plan) { p.Slots[1].Inputs = append(p.Slots[1].Inputs, field("new-field")) }},
		{"shape", "unknown-field-reference", func(p *Plan) { p.Mappings[0].From.Contract.Shape = "different" }},
		{"outcome", "undeclared-missing-outcome", func(p *Plan) { p.Mappings[0].MissingOutcome = "invented" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, p := fixture(t)
			tc.edit(&p)
			got := Inspect(r, p)
			if got.Verdict == composition.ProposalCandidate || got.Order != nil || !has(got, tc.code) {
				t.Fatalf("%s: %+v", tc.code, got)
			}
		})
	}
}
func TestCompatibility(t *testing.T) {
	one := field("x")
	optional := one
	optional.Multiplicity = Optional
	integer := one
	integer.Type.Name = "integer"
	decimal := one
	decimal.Type.Name = "decimal"
	list := one
	list.Multiplicity = List
	set := one
	set.Multiplicity = Set
	identity := one
	identity.Type = Type{Name: "identity", Unit: "none", IdentityDomain: "tenant-a"}
	other := identity
	other.Type.IdentityDomain = "tenant-b"
	cases := []struct {
		from, to Field
		want     Compatibility
	}{{one, one, Identical}, {one, optional, Widening}, {optional, one, Conditional}, {integer, decimal, Widening}, {decimal, integer, Conditional}, {list, set, Incompatible}, {list, one, Incompatible}, {identity, other, Incompatible}}
	for _, tc := range cases {
		if got := Compare(tc.from, tc.to); got != tc.want {
			t.Fatalf("%+v -> %+v = %s", tc.from, tc.to, got)
		}
	}
}
func TestOptionalPolicyAndIsolation(t *testing.T) {
	r, p := fixture(t)
	p.Slots = append(p.Slots, Slot{ID: "extra", Contract: composition.Contract{ID: "extra", Version: "1", Shape: "s"}, Multiplicity: Optional, WhenAbsent: "record-and-continue"})
	if got := Inspect(r, p); got.Verdict != composition.ProposalCandidate || got.Wiring.Verdict != composition.ProposalUnknown {
		t.Fatalf("declared absence: %+v", got)
	}
	for _, critical := range []bool{true, false} {
		p.Slots[2].Critical = critical
		if !critical {
			p.Slots[2].WhenAbsent = ""
		}
		if got := Inspect(r, p); !has(got, "invalid-optional-policy") {
			t.Fatalf("C03/C04: %+v", got)
		}
	}
	p.Scope = "other-app"
	if got := Inspect(r, p); got.Verdict == composition.ProposalCandidate {
		t.Fatal("cross-scope providers reused")
	}
	if got := Inspect(nil, p); got.Verdict == composition.ProposalCandidate {
		t.Fatal("nil registry accepted")
	}
}
func TestContributionsUseDeclaredKeys(t *testing.T) {
	r, p := fixture(t)
	p.Slots = append(p.Slots, Slot{ID: "menu", Contract: sourceContract, Multiplicity: Set})
	_, err := r.Apply(r.Snapshot().Revision, []composition.Provider{{ID: "source-p2", Scope: "app", Offers: sourceContract}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	p.Slots[0].ProviderID = "source-p"
	p.Contributions = []Contribution{{"menu", "source-p2", "02"}, {"menu", "source-p", "01"}}
	a := Inspect(r, p)
	p.Contributions[0], p.Contributions[1] = p.Contributions[1], p.Contributions[0]
	b := Inspect(r, p)
	if a.Verdict != composition.ProposalCandidate || !reflect.DeepEqual(a, b) || !reflect.DeepEqual(a.ContributionOrder["menu"], []string{"source-p", "source-p2"}) {
		t.Fatalf("C01: %+v / %+v", a, b)
	}
	p.Contributions[1].OrderKey = "01"
	if got := Inspect(r, p); !has(got, "contribution-collision") || got.ContributionOrder != nil {
		t.Fatalf("C02: %+v", got)
	}
}
func TestSamplesPreserveNullEmptyAndOnlyMappedFields(t *testing.T) {
	r, p := fixture(t)
	p.Slots[0].Outputs[0].Nullable = true
	p.Slots[1].Inputs[0].Nullable = true
	for _, v := range []SampleValue{{State: Present, Scalar: ""}, {State: Null}, {State: Present, Scalar: "value"}} {
		samples := map[Ref]SampleValue{p.Mappings[0].From: v, {Slot: "producer", Contract: sourceContract, Field: "unused"}: {State: Present, Scalar: "private-unmapped"}}
		got, err := ProjectSamples(r, p, samples)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Values) != 1 || got.Values[p.Mappings[0].To] != v {
			t.Fatalf("M03/M10: %+v", got)
		}
	}
	for _, state := range []ValueState{Absent, Unavailable, Pending} {
		_, err := ProjectSamples(r, p, map[Ref]SampleValue{p.Mappings[0].From: {State: state}})
		var missing *MissingError
		if !errors.As(err, &missing) || missing.State != state {
			t.Fatalf("M11 state %s lost: %v", state, err)
		}
	}
	p.Slots[0].Outputs[0].Multiplicity = Optional
	p.Slots[1].Outcomes = []string{"not-notifiable"}
	p.Mappings[0].MissingOutcome = "not-notifiable"
	got, err := ProjectSamples(r, p, nil)
	if err != nil || got.Outcome != "not-notifiable" || got.Values != nil {
		t.Fatalf("M02 declared outcome: %+v %v", got, err)
	}
	_, err = ProjectSamples(r, p, map[Ref]SampleValue{p.Mappings[0].From: {State: Pending}})
	if err == nil {
		t.Fatal("pending became business absence")
	}
}
