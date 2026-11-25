package validationeval_test

import (
	"math/big"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A rule may write `or <name>` after an input and the declaration may state
// `absent as <name>`. The evaluator asks the SELECTED implementation whether it
// can read that mapping — but only the implementation the composition resolved
// carries that answer.
//
// OperatorImplementation has had an Absent field since the mapping was added,
// while NamedImplementation (what a binary carries and a Workspace selects by
// name) did not: ResolveImplementations built the resolved implementation
// without it. Every shipped mapping was therefore unreachable through the
// product path, and a Workspace could select an implementation that declares it
// reads `open` and still get undecided (#4359).
func TestTheAbsentMappingTravelsWithTheResolvedImplementation(t *testing.T) {
	// A program that carries an implementation reading `open` as zero.
	available := []validationeval.NamedImplementation{
		{
			ID:    "test.number.at-least-with-open",
			Input: semantic.OperatorInputNumber,
			Number: func(value *big.Rat) (bool, error) {
				return value.Sign() >= 0, nil
			},
			Absent: func(mapping string) (*big.Rat, bool) {
				if mapping == "open" {
					return new(big.Rat), true
				}
				return nil, false
			},
		},
	}

	resolved, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{
			{Operator: "stock.at_least", Implementation: "test.number.at-least-with-open"},
		}, available, nil)
	if err != nil {
		t.Fatalf("cannot resolve the selection: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("resolved=%d, want one", len(resolved))
	}
	implementation := resolved[0]

	if implementation.Absent == nil {
		t.Fatal("the resolved implementation cannot read a mapping, although the one the program carries can")
	}
	mapped, ok := implementation.Absent("open")
	if !ok {
		t.Fatalf("the mapping `open` was not readable although the implementation states it")
	}
	if mapped == nil || mapped.Sign() != 0 {
		t.Fatalf("the mapping `open` reads as %v, want zero", mapped)
	}
	// A mapping the implementation never stated stays unreadable, so the
	// decision is undecided rather than read as zero.
	if _, ok := implementation.Absent("closed"); ok {
		t.Fatal("a mapping the implementation never stated was read")
	}
}

// An implementation that states no mapping resolves to one that states none,
// so its rules stay undecided rather than being handed a value nobody stated.
func TestAnImplementationWithoutMappingsResolvesWithoutThem(t *testing.T) {
	available := []validationeval.NamedImplementation{
		{
			ID:    "test.number.at-least",
			Input: semantic.OperatorInputNumber,
			Number: func(value *big.Rat) (bool, error) {
				return value.Sign() >= 0, nil
			},
		},
	}
	resolved, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{
			{Operator: "stock.at_least", Implementation: "test.number.at-least"},
		}, available, nil)
	if err != nil {
		t.Fatalf("cannot resolve the selection: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("resolved=%d, want one", len(resolved))
	}
	if resolved[0].Absent != nil {
		t.Fatal("an implementation that states no mapping resolved to one that does")
	}
}
