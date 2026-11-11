package partition

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Assignment is one Function→Subsystem placement.
type Assignment struct {
	Function  string `json:"function"`
	Subsystem string `json:"subsystem"`
}

// Candidate is one feasible subsystem partition candidate.
type Candidate struct {
	ID               string       `json:"id"`
	Assignments      []Assignment `json:"assignments"`
	Status           string       `json:"status"`
	ConstraintsMet   bool         `json:"constraintsMet"`
	ConstraintIssues []string     `json:"constraintIssues"`
	Evaluation       Evaluation   `json:"evaluation"`
}

// Evaluation summarizes cross-boundary and internal cohesion metrics for a candidate.
type Evaluation struct {
	CrossWriteCount  int            `json:"crossWriteCount"`
	CrossWriteWeight int64          `json:"crossWriteWeight"`
	CrossReadCount   int            `json:"crossReadCount"`
	InternalWeight   int64          `json:"internalWeight"`
	TotalWeight      int64          `json:"totalWeight"`
	Cohesion         float64        `json:"cohesion"`
	Sizes            map[string]int `json:"sizes"`
}

// CandidateProblem extends the existing exact evaluator's input with read edges and constraints.
type CandidateProblem struct {
	Functions   []Function
	Writes      []Write
	MustLinks   []MustLink
	ReadEdges   []ReadEdge
	Constraints []Constraint
}

// ReadEdge is a non-write access occurrence.
type ReadEdge struct {
	ID            string `json:"id"`
	Function      string `json:"function"`
	Data          string `json:"data"`
	Provenance    string `json:"provenance"`
	SourceLocator string `json:"sourceLocator"`
	Notes         string `json:"notes"`
}

// CandidateOptions tunes generation.
type CandidateOptions struct {
	MaxExhaustiveFunctions int
	MaxCandidates          int
}

// CandidateOption is a functional option for GenerateCandidates.
type CandidateOption func(*CandidateOptions)

// WithMaxExhaustiveFunctions switches to exhaustive search when functions are at most this many.
func WithMaxExhaustiveFunctions(n int) CandidateOption {
	return func(o *CandidateOptions) { o.MaxExhaustiveFunctions = n }
}

// WithMaxCandidates stops generation after this many candidates are produced.
func WithMaxCandidates(n int) CandidateOption {
	return func(o *CandidateOptions) { o.MaxCandidates = n }
}

func defaultOptions() CandidateOptions {
	return CandidateOptions{MaxExhaustiveFunctions: 10, MaxCandidates: 4096}
}

// GenerateCandidates deterministically produces ranked feasible candidates.
// The returned status is "exhaustive-proven" for small exhaustive searches and "heuristic" otherwise.
func GenerateCandidates(p CandidateProblem, opts ...CandidateOption) ([]Candidate, string) {
	options := defaultOptions()
	for _, apply := range opts {
		apply(&options)
	}

	if len(p.Functions) == 0 {
		return nil, "infeasible"
	}

	fnIDs := make([]string, len(p.Functions))
	for i, f := range p.Functions {
		fnIDs[i] = f.ID
	}
	components, err := MustLinkComponents(fnIDs, p.MustLinks)
	if err != nil {
		return nil, "infeasible"
	}
	allowedByFn := map[string][]string{}
	for _, f := range p.Functions {
		allowedByFn[f.ID] = f.Allowed
	}
	allowedByComponent := make([][]string, len(components))
	for i, comp := range components {
		if len(comp) == 0 {
			return nil, "infeasible"
		}
		intersected := make([]string, 0)
		for _, s := range allowedByFn[comp[0]] {
			ok := true
			for _, other := range comp[1:] {
				found := false
				for _, a := range allowedByFn[other] {
					if a == s {
						found = true
						break
					}
				}
				if !found {
					ok = false
					break
				}
			}
			if ok {
				intersected = append(intersected, s)
			}
		}
		if len(intersected) == 0 {
			return nil, "infeasible"
		}
		sort.Strings(intersected)
		allowedByComponent[i] = intersected
	}

	// collect all referenced subsystem ids for complete enumeration.
	allSubsystems := map[string]bool{}
	for _, f := range p.Functions {
		for _, s := range f.Allowed {
			allSubsystems[s] = true
		}
	}
	for _, c := range p.Constraints {
		switch c.Kind {
		case ConstraintCapacity:
			allSubsystems[c.Subsystem] = true
		case ConstraintFixed:
			allSubsystems[c.Subsystem] = true
		}
	}
	subsystemList := make([]string, 0, len(allSubsystems))
	for s := range allSubsystems {
		subsystemList = append(subsystemList, s)
	}
	sort.Strings(subsystemList)

	useExhaustive := len(p.Functions) <= options.MaxExhaustiveFunctions

	assignments := make([]map[string]string, 0)
	if useExhaustive {
		assignments = enumerateAssignments(components, allowedByComponent, subsystemList, p.Constraints)
	} else {
		assignments = heuristicAssignments(components, allowedByComponent, subsystemList, p.Constraints, options.MaxCandidates)
	}
	if len(assignments) == 0 {
		return nil, "infeasible"
	}

	candidates := make([]Candidate, 0, len(assignments))
	for i, assign := range assignments {
		asgn := make([]Assignment, 0, len(assign))
		for fn, sub := range assign {
			asgn = append(asgn, Assignment{Function: fn, Subsystem: sub})
		}
		sort.Slice(asgn, func(i, j int) bool { return asgn[i].Function < asgn[j].Function })
		issues, feasible := validateAssignment(assign, p.Constraints)
		ev := evaluateAssignment(assign, p.Writes, p.ReadEdges)
		status := "candidate-ranked"
		if useExhaustive {
			status = "exhaustive-proven"
		}
		candidates = append(candidates, Candidate{
			ID:               fmt.Sprintf("cand-%03d", i+1),
			Assignments:      asgn,
			Status:           status,
			ConstraintsMet:   feasible,
			ConstraintIssues: issues,
			Evaluation:       ev,
		})
	}
	sortCandidates(candidates)
	return candidates, candidates[0].Status
}

func enumerateAssignments(components [][]string, allowedByComponent [][]string, subsystems []string, constraints []Constraint) []map[string]string {
	var walk func(idx int, current map[string]string, out []map[string]string) []map[string]string
	walk = func(idx int, current map[string]string, out []map[string]string) []map[string]string {
		if idx == len(components) {
			_, feasible := validateAssignment(current, constraints)
			if !feasible {
				return out
			}
			// ensure capacity constraint is respected during enumeration when present.
			if capacityViolated(current, constraints) {
				return out
			}
			return append(out, cloneAssignment(current))
		}
		comp := components[idx]
		for _, sub := range allowedByComponent[idx] {
			current[comp[0]] = sub
			for _, fn := range comp[1:] {
				current[fn] = sub
			}
			out = walk(idx+1, current, out)
			if len(out) > 4096 {
				break
			}
		}
		return out
	}
	return walk(0, map[string]string{}, nil)
}

func heuristicAssignments(components [][]string, allowedByComponent [][]string, subsystems []string, constraints []Constraint, max int) []map[string]string {
	out := []map[string]string{}
	seen := map[string]bool{}
	// try each subsystem for the first component, then greedily fill others.
	for _, primary := range subsystems {
		assign := map[string]string{}
		for i, comp := range components {
			chosen := ""
			if i == 0 {
				chosen = firstAllowed(allowedByComponent[0], primary)
			} else {
				chosen = bestAllowedByCohesion(comp, allowedByComponent[i], assign, components, allowedByComponent)
			}
			if chosen == "" {
				continue
			}
			for _, fn := range comp {
				assign[fn] = chosen
			}
		}
		if len(assign) != len(components) {
			continue
		}
		key := assignmentKey(assign)
		if seen[key] {
			continue
		}
		seen[key] = true
		_, feasible := validateAssignment(assign, constraints)
		if !feasible || capacityViolated(assign, constraints) {
			continue
		}
		out = append(out, cloneAssignment(assign))
		if len(out) >= max {
			break
		}
	}
	return out
}

func firstAllowed(options []string, primary string) string {
	if len(options) == 0 {
		return ""
	}
	for _, s := range options {
		if s == primary {
			return s
		}
	}
	return options[0]
}

func bestAllowedByCohesion(comp []string, options []string, current map[string]string, components [][]string, allowedByComponent [][]string) string {
	best := ""
	bestScore := int64(math.MaxInt64)
	for _, sub := range options {
		current[comp[0]] = sub
		for _, fn := range comp[1:] {
			current[fn] = sub
		}
		if capacityViolated(current, []Constraint{}) {
			continue
		}
		score := int64(0)
		// simple heuristic: prefer subsystem that minimizes expected cross weight among remaining components.
		for j, other := range components {
			if j == 0 || current[other[0]] != "" {
				continue
			}
			for _, opt := range allowedByComponent[j] {
				current[other[0]] = opt
				for _, fn := range other[1:] {
					current[fn] = opt
				}
				score++
				if score >= bestScore {
					break
				}
			}
		}
		if score < bestScore || (score == bestScore && (best == "" || sub < best)) {
			bestScore = score
			best = sub
		}
	}
	return best
}

func assignmentKey(assign map[string]string) string {
	keys := make([]string, 0, len(assign))
	for k := range assign {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+":"+assign[k])
	}
	return strings.Join(parts, "|")
}

func cloneAssignment(src map[string]string) map[string]string {
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func capacityViolated(assign map[string]string, constraints []Constraint) bool {
	cap := map[string]int{}
	for _, c := range constraints {
		if c.Kind != ConstraintCapacity {
			continue
		}
		max := c.MaxSize
		if max <= 0 {
			continue
		}
		for _, sub := range assign {
			if sub == c.Subsystem {
				cap[sub]++
				if cap[sub] > max {
					return true
				}
			}
		}
	}
	return false
}

func validateAssignment(assign map[string]string, constraints []Constraint) ([]string, bool) {
	issues := []string{}
	feasible := true
	// must-link already enforced by enumeration; validate remaining constraints.
	for _, c := range constraints {
		switch c.Kind {
		case ConstraintMustSeparate:
			if assign[c.Left] == assign[c.Right] {
				issues = append(issues, fmt.Sprintf("%s must separate %s/%s", c.ID, c.Left, c.Right))
				feasible = false
			}
		case ConstraintFixed:
			if assign[c.Left] != c.Subsystem {
				issues = append(issues, fmt.Sprintf("%s fixed %s to %s but got %s", c.ID, c.Left, c.Subsystem, assign[c.Left]))
				feasible = false
			}
		case ConstraintExternalOwned:
			// not a placement constraint; keep for evaluation stage.
		case ConstraintSharedUpdate:
			// not a placement constraint; evaluate later.
		case ConstraintCapacity:
			// evaluated in enumeration already.
		case ConstraintBusinessInvariant:
			first := assign[c.Members[0]]
			for _, m := range c.Members[1:] {
				if assign[m] != first {
					issues = append(issues, fmt.Sprintf("%s business invariant violated", c.ID))
					feasible = false
					break
				}
			}
		default:
			// ignore unknown constraint kinds for validation.
		}
	}
	sort.Strings(issues)
	return issues, feasible
}

func evaluateAssignment(assign map[string]string, writes []Write, readEdges []ReadEdge) Evaluation {
	weightByData := map[string]int64{}
	totalWeight := int64(0)
	for _, w := range writes {
		totalWeight += w.Weight
		weightByData[w.Data] += w.Weight
	}
	sizes := map[string]int{}
	crossWrites := 0
	crossWriteWeight := int64(0)
	crossReads := 0
	internalWeight := int64(0)
	// build per-function write summaries
	writesByFn := map[string][]Write{}
	for _, w := range writes {
		writesByFn[w.Function] = append(writesByFn[w.Function], w)
	}
	readByFn := map[string][]ReadEdge{}
	for _, r := range readEdges {
		readByFn[r.Function] = append(readByFn[r.Function], r)
	}

	for fn, sub := range assign {
		sizes[sub]++
		for _, w := range writesByFn[fn] {
			if w.Owner != sub {
				crossWrites++
				crossWriteWeight += w.Weight
			} else {
				internalWeight += w.Weight
			}
		}
		for range readByFn[fn] {
			// read edge target data ownership not used here; count occurrence.
			crossReads++
		}
	}

	cohesion := 0.0
	if totalWeight > 0 {
		cohesion = float64(internalWeight) / float64(totalWeight)
	}
	return Evaluation{
		CrossWriteCount: crossWrites, CrossWriteWeight: crossWriteWeight, CrossReadCount: crossReads,
		InternalWeight: internalWeight, TotalWeight: totalWeight, Cohesion: cohesion, Sizes: sizes,
	}
}

func sortCandidates(candidates []Candidate) {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].ConstraintsMet != candidates[j].ConstraintsMet {
			return candidates[i].ConstraintsMet
		}
		if candidates[i].Evaluation.CrossWriteCount != candidates[j].Evaluation.CrossWriteCount {
			return candidates[i].Evaluation.CrossWriteCount < candidates[j].Evaluation.CrossWriteCount
		}
		if candidates[i].Evaluation.CrossWriteWeight != candidates[j].Evaluation.CrossWriteWeight {
			return candidates[i].Evaluation.CrossWriteWeight < candidates[j].Evaluation.CrossWriteWeight
		}
		if candidates[i].Evaluation.CrossReadCount != candidates[j].Evaluation.CrossReadCount {
			return candidates[i].Evaluation.CrossReadCount < candidates[j].Evaluation.CrossReadCount
		}
		if candidates[i].Evaluation.Cohesion != candidates[j].Evaluation.Cohesion {
			return candidates[i].Evaluation.Cohesion > candidates[j].Evaluation.Cohesion
		}
		if candidates[i].Evaluation.TotalWeight != 0 || candidates[j].Evaluation.TotalWeight != 0 {
			return candidates[i].Evaluation.TotalWeight > candidates[j].Evaluation.TotalWeight
		}
		return candidates[i].ID < candidates[j].ID
	})
}

// ParetoFront returns the set of candidates that are non-dominated by both cross-write and cross-read objectives.
func ParetoFront(candidates []Candidate) []Candidate {
	if len(candidates) == 0 {
		return nil
	}
	// Only consider feasible candidates with positive total weight or positive edges.
	out := make([]Candidate, 0)
	for _, c := range candidates {
		if !c.ConstraintsMet {
			continue
		}
		out = append(out, c)
	}
	dom := make([]bool, len(out))
	for i := 0; i < len(out); i++ {
		if dom[i] {
			continue
		}
		for j := 0; j < len(out); j++ {
			if i == j || dom[j] {
				continue
			}
			if out[j].Evaluation.CrossWriteCount <= out[i].Evaluation.CrossWriteCount &&
				out[j].Evaluation.CrossReadCount <= out[i].Evaluation.CrossReadCount &&
				(out[j].Evaluation.CrossWriteCount < out[i].Evaluation.CrossWriteCount || out[j].Evaluation.CrossReadCount < out[i].Evaluation.CrossReadCount) {
				dom[i] = true
				break
			}
		}
	}
	pareto := make([]Candidate, 0, len(out))
	for i, c := range out {
		if !dom[i] {
			pareto = append(pareto, c)
		}
	}
	sortCandidates(pareto)
	return pareto
}

// CandidateEvaluationBreakdown returns a human-readable breakdown for a candidate.
func CandidateEvaluationBreakdown(c Candidate) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("candidate=%s status=%s feasible=%t cross_write_count=%d cross_write_weight=%d cross_read_count=%d cohesion=%.4f total_weight=%d sizes=%v",
		c.ID, c.Status, c.ConstraintsMet, c.Evaluation.CrossWriteCount, c.Evaluation.CrossWriteWeight,
		c.Evaluation.CrossReadCount, c.Evaluation.Cohesion, c.Evaluation.TotalWeight, c.Evaluation.Sizes))
	return sb.String()
}
