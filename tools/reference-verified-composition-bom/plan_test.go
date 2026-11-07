package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestPairwise3AxisReducesCases(t *testing.T) {
	input := Input{MaxCases: 6, Axes: []Axis{
		{Name: "C", Options: []string{"c1", "c0"}},
		{Name: "A", Options: []string{"a1", "a0"}},
		{Name: "B", Options: []string{"b1", "b0"}},
	}}
	first, err := PlanPairwise(input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := PlanPairwise(input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Qualified || first.ClaimCeiling != "planning-only" || first.CoveredPairs != 12 || first.TotalPairs != 12 || len(first.Cases) != 4 || first.FullCartesianSize != "8" {
		t.Fatalf("unexpected plan %+v", first)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("non-deterministic pairwise plan")
	}
}

func TestPairwise4Axis3OptionsAndLargeCartesian(t *testing.T) {
	axes := []Axis{}
	for i := 0; i < 4; i++ {
		axes = append(axes, Axis{Name: fmt.Sprintf("axis%d", i), Options: []string{"v0", "v1", "v2"}})
	}
	plan, err := PlanPairwise(Input{Axes: axes, MaxCases: 40})
	if err != nil {
		t.Fatal(err)
	}
	if plan.FullCartesianSize != "81" || plan.TotalPairs != 54 || plan.CoveredPairs != 54 || len(plan.Cases) >= 81 {
		t.Fatalf("unexpected %+v", plan)
	}
}

func TestPairwiseRejectsInvalidAndOverBudget(t *testing.T) {
	base := Input{Axes: []Axis{{Name: "A", Options: []string{"0", "1"}}, {Name: "B", Options: []string{"0", "1"}}}, MaxCases: 3}
	if _, err := PlanPairwise(base); err == nil {
		t.Fatal("accepted insufficient maxCases")
	}
	base.MaxCases = 4
	base.Axes = append(base.Axes, Axis{Name: "A", Options: []string{"0", "1"}})
	if _, err := PlanPairwise(base); err == nil {
		t.Fatal("accepted duplicate axis")
	}
	base.Axes[2].Name = "C"
	base.Axes[2].Options = []string{"0", "0"}
	if _, err := PlanPairwise(base); err == nil {
		t.Fatal("accepted duplicate option")
	}
}

func TestPairwiseSingleAxisStillCoversAllValues(t *testing.T) {
	x := Input{MaxCases: 3, Axes: []Axis{{Name: "version", Options: []string{"3", "1", "2"}}}}
	p, e := PlanPairwise(x)
	if e != nil {
		t.Fatal(e)
	}
	if p.TotalPairs != 0 || p.CoveredPairs != 0 || len(p.Cases) != 3 {
		t.Fatalf("got %+v", p)
	}
}

func TestPairwiseDoesNotMutateInput(t *testing.T) {
	x := Input{MaxCases: 10, Axes: []Axis{
		{Name: "z", Options: []string{"3", "1", "2"}},
		{Name: "a", Options: []string{"b", "a"}},
	}}
	want := []Axis{
		{Name: "z", Options: []string{"3", "1", "2"}},
		{Name: "a", Options: []string{"b", "a"}},
	}
	if _, err := PlanPairwise(x); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(x.Axes, want) {
		t.Fatalf("input mutated: %+v", x.Axes)
	}
}

func TestPairwiseUnevenAxisSizes(t *testing.T) {
	x := Input{MaxCases: 200, Axes: []Axis{
		{Name: "a", Options: []string{"0", "1", "2", "3"}},
		{Name: "b", Options: []string{"0", "1", "2"}},
		{Name: "c", Options: []string{"0", "1"}},
		{Name: "d", Options: []string{"0", "1", "2"}},
		{Name: "e", Options: []string{"0", "1"}},
	}}
	p, err := PlanPairwise(x)
	if err != nil {
		t.Fatal(err)
	}
	if p.CoveredPairs != p.TotalPairs || len(p.Cases) >= 144 || p.FullCartesianSize != "144" {
		t.Fatalf("unexpected pairwise coverage: %+v", p)
	}
}
