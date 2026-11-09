package main

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
)

// ResolvedPairwisePlan covers pairs observed in supplied exact-lock candidates.
// Neither lock authenticity nor a complete feasible candidate universe is established.
type ResolvedPairwisePlan struct {
	Strategy                      string             `json:"strategy"`
	ClaimCeiling                  string             `json:"claimCeiling"`
	Qualified                     bool               `json:"qualified"`
	CandidateUniverseCompleteness string             `json:"candidateUniverseCompleteness"`
	FullCartesianSize             string             `json:"fullCartesianSize"`
	CandidateCount                int                `json:"candidateCount"`
	DeclaredPairs                 int                `json:"declaredPairs"`
	CandidatePairs                int                `json:"candidatePairs"`
	CoveredCandidatePairs         int                `json:"coveredCandidatePairs"`
	UnrepresentedDeclaredPairs    int                `json:"unrepresentedDeclaredPairs"`
	Cases                         []ResolvedPlanCase `json:"cases"`
}

type ResolvedPlanCase struct {
	Selections      []Selection `json:"selections"`
	LockFingerprint string      `json:"lockFingerprint"`
}

type resolvedCandidate struct {
	selection ResolvedPlanCase
	pairSet   map[pairKey]bool
}

// PlanResolvedPairwise selects a deterministic greedy cover of pairs present in
// *supplied* exact locks. It never solves constraints or proves those locks
// describe all possible configurations. No test execution happens here.
func PlanResolvedPairwise(input Input) (ResolvedPairwisePlan, error) {
	out := ResolvedPairwisePlan{
		Strategy: "supplied-locks-pairwise", ClaimCeiling: "planning-only",
		Qualified: false, CandidateUniverseCompleteness: "unknown",
		Cases: []ResolvedPlanCase{},
	}
	if input.MaxCases < 1 || len(input.Axes) == 0 || len(input.Locks) == 0 {
		return out, errors.New("positive maxCases, axes and supplied exact locks are required")
	}
	axes := make([]Axis, len(input.Axes))
	for i, a := range input.Axes {
		axes[i] = Axis{Name: a.Name, Options: append([]string(nil), a.Options...)}
	}
	sort.Slice(axes, func(i, j int) bool { return axes[i].Name < axes[j].Name })
	options := make([]map[string]int, len(axes))
	axisIndexes := map[string]int{}
	full := big.NewInt(1)
	for i := range axes {
		a := &axes[i]
		if !validToken(a.Name) || len(a.Options) == 0 {
			return out, fmt.Errorf("invalid axis %q", a.Name)
		}
		if _, ok := axisIndexes[a.Name]; ok {
			return out, fmt.Errorf("duplicate axis %q", a.Name)
		}
		axisIndexes[a.Name] = i
		sort.Strings(a.Options)
		options[i] = map[string]int{}
		for j, v := range a.Options {
			if !validToken(v) {
				return out, fmt.Errorf("invalid option %q", v)
			}
			if _, ok := options[i][v]; ok {
				return out, fmt.Errorf("duplicate option %q", v)
			}
			options[i][v] = j
		}
		full.Mul(full, big.NewInt(int64(len(a.Options))))
	}
	out.FullCartesianSize = full.String()
	for i := range axes {
		for j := i + 1; j < len(axes); j++ {
			out.DeclaredPairs += len(axes[i].Options) * len(axes[j].Options)
		}
	}
	candidates := []resolvedCandidate{}
	keys, fingerprints, targets := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, lock := range input.Locks {
		if len(lock.Selections) != len(axes) || !validToken(lock.Fingerprint) || lock.SourceFingerprint == "" || lock.ArtifactDigest == "" || lock.Target == "" {
			return out, errors.New("lock requires exact selections and source, artifact, target, fingerprint")
		}
		if fingerprints[lock.Fingerprint] {
			return out, fmt.Errorf("duplicate lock fingerprint %q", lock.Fingerprint)
		}
		fingerprints[lock.Fingerprint] = true
		targets[lock.Target] = true
		values := make([]int, len(axes))
		seen := map[string]bool{}
		selections := make([]Selection, len(axes))
		for _, sel := range lock.Selections {
			idx, ok := axisIndexes[sel.Axis]
			if !ok || seen[sel.Axis] {
				return out, errors.New("unknown or duplicate selection axis")
			}
			seen[sel.Axis] = true
			val, exists := options[idx][sel.Value]
			if !exists {
				return out, errors.New("lock selection is outside declared axis values")
			}
			values[idx] = val
			selections[idx] = sel
			if sel.Axis == "target" && sel.Value != lock.Target {
				return out, errors.New("target selection disagrees with lock's actual target")
			}
		}
		k := key(selections)
		if keys[k] {
			return out, fmt.Errorf("duplicate resolved case %q", k)
		}
		keys[k] = true
		pairSet := map[pairKey]bool{}
		for i := range values {
			for j := i + 1; j < len(values); j++ {
				pairSet[pairKey{i, values[i], j, values[j]}] = true
			}
		}
		candidates = append(candidates, resolvedCandidate{
			selection: ResolvedPlanCase{Selections: selections, LockFingerprint: lock.Fingerprint},
			pairSet:   pairSet,
		})
	}
	// A target is part of the applicability identity: never mix multiple real
	// targets unless the target dimension is explicitly represented in the plan.
	if len(targets) > 1 {
		if _, ok := axisIndexes["target"]; !ok {
			return out, errors.New("multi-target locks require an explicit target axis")
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return key(candidates[i].selection.Selections) < key(candidates[j].selection.Selections)
	})
	out.CandidateCount = len(candidates)
	universe := map[pairKey]bool{}
	for _, c := range candidates {
		for pair := range c.pairSet {
			universe[pair] = true
		}
	}
	out.CandidatePairs = len(universe)
	out.UnrepresentedDeclaredPairs = out.DeclaredPairs - out.CandidatePairs
	// For a single axis the pair universe is empty; selecting every supplied
	// value is necessary to avoid a vacuous "all pairs covered" impression.
	if len(axes) == 1 {
		if len(candidates) > input.MaxCases {
			return out, errors.New("selected cases exceed maxCases")
		}
		for _, c := range candidates {
			out.Cases = append(out.Cases, c.selection)
		}
		return out, nil
	}
	for len(universe) > 0 {
		if len(out.Cases) >= input.MaxCases {
			return out, errors.New("selected cases exceed maxCases")
		}
		best, bestCount := -1, 0
		for i, c := range candidates {
			count := 0
			for p := range c.pairSet {
				if universe[p] {
					count++
				}
			}
			if count > bestCount {
				best, bestCount = i, count
			}
		}
		if best < 0 {
			return out, errors.New("failed to cover supplied pairs")
		}
		out.Cases = append(out.Cases, candidates[best].selection)
		for p := range candidates[best].pairSet {
			delete(universe, p)
		}
	}
	out.CoveredCandidatePairs = out.CandidatePairs
	return out, nil
}
