package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func resolvedTestInput() Input {
	axes := []Axis{
		{Name: "provider", Options: []string{"1.4", "1.5"}},
		{Name: "consumer", Options: []string{"2.0", "2.1"}},
		{Name: "renderer", Options: []string{"3.0", "3.1"}},
	}
	locks := []Lock{}
	// Only four of eight declared combinations were actually supplied as resolved locks.
	for i, choices := range [][3]string{
		{"1.4", "2.0", "3.0"},
		{"1.4", "2.0", "3.1"},
		{"1.4", "2.1", "3.0"},
		{"1.5", "2.0", "3.0"},
	} {
		locks = append(locks, Lock{
			Selections:        []Selection{{Axis: "renderer", Value: choices[2]}, {Axis: "provider", Value: choices[0]}, {Axis: "consumer", Value: choices[1]}},
			Fingerprint:       fmt.Sprintf("lock%d", i),
			SourceFingerprint: fmt.Sprintf("source%d", i),
			ArtifactDigest:    fmt.Sprintf("digest%d", i), Target: "linux-amd64",
		})
	}
	return Input{Axes: axes, Locks: locks, MaxCases: 8}
}

func TestResolvedPairwiseKnownOnlyNoVacuousCertification(t *testing.T) {
	in := resolvedTestInput()
	out, err := PlanResolvedPairwise(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Qualified || out.ClaimCeiling != "planning-only" || out.CandidateUniverseCompleteness != "unknown" {
		t.Fatalf("improper qualification %+v", out)
	}
	if out.FullCartesianSize != "8" || out.CandidateCount != 4 || out.DeclaredPairs != 12 || out.CandidatePairs != 9 || out.CoveredCandidatePairs != 9 || out.UnrepresentedDeclaredPairs != 3 {
		t.Fatalf("incorrect scope %+v", out)
	}
	if len(out.Cases) > 4 || len(out.Cases) < 1 {
		t.Fatalf("unexpected row count %d", len(out.Cases))
	}
	locks := map[string]bool{}
	for _, l := range in.Locks {
		locks[l.Fingerprint] = true
	}
	for _, c := range out.Cases {
		if !locks[c.LockFingerprint] {
			t.Fatalf("fabricated lock %+v", c)
		}
	}
	// The chosen plan's universe covers precisely pairs represented by supplied locks,
	// but makes no statement that the three absent pairs are impossible.
	again, err := PlanResolvedPairwise(in)
	if err != nil || !reflect.DeepEqual(out, again) {
		t.Fatalf("nondeterministic: %v", err)
	}
	in.Locks[0], in.Locks[3] = in.Locks[3], in.Locks[0]
	permuted, err := PlanResolvedPairwise(in)
	if err != nil || !reflect.DeepEqual(out, permuted) {
		t.Fatalf("ordering changed: %v", err)
	}
}

func TestResolvedPairwiseBudgetAndInvalidLocks(t *testing.T) {
	x := resolvedTestInput()
	x.MaxCases = 1
	if _, err := PlanResolvedPairwise(x); err == nil || !strings.Contains(err.Error(), "maxCases") {
		t.Fatalf("missing budget fail: %v", err)
	}
	x = resolvedTestInput()
	x.Locks = append(x.Locks, x.Locks[0])
	if _, err := PlanResolvedPairwise(x); err == nil {
		t.Fatal("duplicate accepted")
	}
	x = resolvedTestInput()
	x.Locks[0].Selections[0].Value = "unknown"
	if _, err := PlanResolvedPairwise(x); err == nil {
		t.Fatal("invalid selection accepted")
	}
	x = resolvedTestInput()
	x.Locks[0].ArtifactDigest = ""
	if _, err := PlanResolvedPairwise(x); err == nil {
		t.Fatal("missing artifact identity accepted")
	}
}

func TestResolvedPairwiseCannotMixTargetsWithoutTargetAxis(t *testing.T) {
	x := resolvedTestInput()
	x.Locks[0].Target = "windows-amd64"
	if _, err := PlanResolvedPairwise(x); err == nil {
		t.Fatal("mixed target qualified without dimension")
	}
	x = Input{
		Axes:     []Axis{{Name: "target", Options: []string{"linux-amd64", "windows-amd64"}}, {Name: "version", Options: []string{"1.4"}}},
		MaxCases: 2,
		Locks: []Lock{
			{Selections: []Selection{{Axis: "target", Value: "linux-amd64"}, {Axis: "version", Value: "1.4"}}, Target: "linux-amd64", Fingerprint: "a", ArtifactDigest: "digest-a", SourceFingerprint: "source-a"},
			{Selections: []Selection{{Axis: "target", Value: "windows-amd64"}, {Axis: "version", Value: "1.4"}}, Target: "windows-amd64", Fingerprint: "b", ArtifactDigest: "digest-b", SourceFingerprint: "source-b"},
		},
	}
	p, err := PlanResolvedPairwise(x)
	if err != nil || p.CandidatePairs != 2 || len(p.Cases) != 2 {
		t.Fatalf("unexpected %+v: %v", p, err)
	}
	x.Locks[0].Selections[0].Value = "windows-amd64"
	if _, err := PlanResolvedPairwise(x); err == nil {
		t.Fatal("target mismatch accepted")
	}
}

func TestResolvedPairwiseExhaustiveSubsets(t *testing.T) {
	// Enumerate 255 nonempty subsets of an 8-tuple universe as a regression
	// against erroneous assumptions that every declared pair is available.
	base := resolvedTestInput()
	locks := []Lock{}
	for a := 0; a < 2; a++ {
		for b := 0; b < 2; b++ {
			for c := 0; c < 2; c++ {
				locks = append(locks, Lock{
					Selections:  []Selection{{Axis: "consumer", Value: base.Axes[1].Options[b]}, {Axis: "renderer", Value: base.Axes[2].Options[c]}, {Axis: "provider", Value: base.Axes[0].Options[a]}},
					Fingerprint: fmt.Sprintf("l%d%d%d", a, b, c), SourceFingerprint: "source", ArtifactDigest: "artifact", Target: "linux-amd64",
				})
			}
		}
	}
	for mask := 1; mask < 1<<len(locks); mask++ {
		x := Input{Axes: base.Axes, MaxCases: 8}
		for i, l := range locks {
			if mask&(1<<i) != 0 {
				x.Locks = append(x.Locks, l)
			}
		}
		p, err := PlanResolvedPairwise(x)
		if err != nil {
			t.Fatalf("subset %d rejected: %v", mask, err)
		}
		if p.CoveredCandidatePairs != p.CandidatePairs || len(p.Cases) > len(x.Locks) || p.CandidatePairs > p.DeclaredPairs || p.Qualified {
			t.Fatalf("subset %d mismatch: %+v", mask, p)
		}
		// Every pair seen in any supplied exact lock appears in a chosen row.
		for _, l := range x.Locks {
			for i := range l.Selections {
				for j := i + 1; j < len(l.Selections); j++ {
					a, b := l.Selections[i], l.Selections[j]
					found := false
					for _, c := range p.Cases {
						m := map[string]string{}
						for _, s := range c.Selections {
							m[s.Axis] = s.Value
						}
						if m[a.Axis] == a.Value && m[b.Axis] == b.Value {
							found = true
							break
						}
					}
					if !found {
						t.Fatalf("subset %d lacks pair (%s,%s)", mask, a.Axis, b.Axis)
					}
				}
			}
		}
	}
}
