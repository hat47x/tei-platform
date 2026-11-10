package main

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
)

// MatrixPlan is a synthetic test selection, not a run or verified BOM.
type MatrixPlan struct {
	Strategy          string        `json:"strategy"`
	ClaimCeiling      string        `json:"claimCeiling"`
	Qualified         bool          `json:"qualified"`
	FullCartesianSize string        `json:"fullCartesianSize"`
	TotalPairs        int           `json:"totalPairs"`
	CoveredPairs      int           `json:"coveredPairs"`
	Cases             [][]Selection `json:"cases"`
}

type pairKey struct{ earlierAxis, earlierValue, laterAxis, laterValue int }

func PlanPairwise(input Input) (MatrixPlan, error) {
	plan := MatrixPlan{Strategy: "pairwise", ClaimCeiling: "planning-only", Qualified: false, Cases: [][]Selection{}}
	if input.MaxCases < 1 || len(input.Axes) < 1 {
		return plan, errors.New("axes and positive maxCases are required")
	}
	axes := make([]Axis, len(input.Axes))
	for i, axis := range input.Axes {
		axes[i] = Axis{Name: axis.Name, Options: append([]string(nil), axis.Options...)}
	}
	sort.Slice(axes, func(i, j int) bool { return axes[i].Name < axes[j].Name })
	seenAxis := map[string]bool{}
	full := big.NewInt(1)
	for i := range axes {
		a := &axes[i]
		if !validToken(a.Name) || seenAxis[a.Name] || len(a.Options) == 0 {
			return plan, fmt.Errorf("invalid or duplicate axis %q", a.Name)
		}
		seenAxis[a.Name] = true
		seenOption := map[string]bool{}
		for _, opt := range a.Options {
			if !validToken(opt) || seenOption[opt] {
				return plan, fmt.Errorf("invalid or duplicate option %q", opt)
			}
			seenOption[opt] = true
		}
		sort.Strings(a.Options)
		full.Mul(full, big.NewInt(int64(len(a.Options))))
	}
	plan.FullCartesianSize = full.String()
	// Each row is a list of option indexes in axis-name order.
	rows := [][]int{}
	if len(axes) == 1 {
		for option := range axes[0].Options {
			if len(rows) >= input.MaxCases {
				return plan, errors.New("pairwise plan exceeds maxCases")
			}
			rows = append(rows, []int{option})
		}
	} else {
		// Start with the full 2-axis combinations, the minimum needed to cover their pairs.
		for a := range axes[0].Options {
			for b := range axes[1].Options {
				if len(rows) >= input.MaxCases {
					return plan, errors.New("pairwise plan exceeds maxCases")
				}
				rows = append(rows, []int{a, b})
			}
		}
		for k := 2; k < len(axes); k++ {
			uncovered := map[pairKey]bool{}
			for prev := 0; prev < k; prev++ {
				for v := range axes[prev].Options {
					for newV := range axes[k].Options {
						uncovered[pairKey{prev, v, k, newV}] = true
					}
				}
			}
			// Horizontal growth: extend existing rows with the option that covers most new pairs.
			for i := range rows {
				best, score := -1, -1
				for newV := range axes[k].Options {
					benefit := 0
					for j, v := range rows[i] {
						if uncovered[pairKey{j, v, k, newV}] {
							benefit++
						}
					}
					if benefit > score {
						score, best = benefit, newV
					}
				}
				rows[i] = append(rows[i], best)
				for j, v := range rows[i][:k] {
					delete(uncovered, pairKey{j, v, k, best})
				}
			}
			// Vertical growth: cover each still-uncovered pair with a new full-length row.
			for len(uncovered) > 0 {
				if len(rows) >= input.MaxCases {
					return plan, errors.New("pairwise plan exceeds maxCases")
				}
				seed := pairKey{}
				found := false
				for j := 0; j < k && !found; j++ {
					for v := range axes[j].Options {
						for newV := range axes[k].Options {
							key := pairKey{j, v, k, newV}
							if uncovered[key] {
								seed = key
								found = true
								break
							}
						}
						if found {
							break
						}
					}
				}
				if !found {
					return plan, errors.New("unexpected pairwise uncovered invariant")
				}
				row := make([]int, k+1)
				row[k] = seed.laterValue
				row[seed.earlierAxis] = seed.earlierValue
				for j := 0; j < k; j++ {
					if j == seed.earlierAxis {
						continue
					}
					// Choose an uncovered pair where possible; otherwise the first value.
					for v := range axes[j].Options {
						if uncovered[pairKey{j, v, k, seed.laterValue}] {
							row[j] = v
							break
						}
					}
				}
				for j, v := range row[:k] {
					delete(uncovered, pairKey{j, v, k, row[k]})
				}
				rows = append(rows, row)
			}
		}
	}
	// Independently measure actual pair coverage in the selected rows, not inferred from the algorithm.
	covered := map[pairKey]bool{}
	totalPairs := 0
	for i := range axes {
		for j := i + 1; j < len(axes); j++ {
			totalPairs += len(axes[i].Options) * len(axes[j].Options)
		}
	}
	for _, r := range rows {
		selected := make([]Selection, len(axes))
		for a, v := range r {
			selected[a] = Selection{Axis: axes[a].Name, Value: axes[a].Options[v]}
		}
		plan.Cases = append(plan.Cases, selected)
		for a, v := range r {
			for b := a + 1; b < len(r); b++ {
				covered[pairKey{a, v, b, r[b]}] = true
			}
		}
	}
	sort.Slice(plan.Cases, func(i, j int) bool { return key(plan.Cases[i]) < key(plan.Cases[j]) })
	plan.TotalPairs = totalPairs
	plan.CoveredPairs = len(covered)
	if len(covered) != totalPairs {
		return plan, errors.New("pairwise plan does not cover every pair")
	}
	return plan, nil
}
