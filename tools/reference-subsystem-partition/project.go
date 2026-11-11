package partition

import (
	"sort"
)

// ProjectedFunction is a function assigned to a subsystem with its operations.
type ProjectedFunction struct {
	Function string   `json:"function"`
	Writes   []string `json:"writes,omitempty"`
	Reads    []string `json:"reads,omitempty"`
}

// ProjectedDependency is one delegation edge visible from a subsystem.
// An invocation is not a direct write; it is recorded as a projection only.
type ProjectedDependency struct {
	Caller          string `json:"caller"`
	Callee          string `json:"callee"`
	CalleeSubsystem string `json:"calleeSubsystem,omitempty"`
	Internal        bool   `json:"internal"`
	Pattern         string `json:"pattern,omitempty"`
	SourceLocator   string `json:"sourceLocator,omitempty"`
}

// ProjectedSubsystem is the ER/CRUD/dependency projection for one subsystem.
type ProjectedSubsystem struct {
	Subsystem    string                `json:"subsystem"`
	Functions    []ProjectedFunction   `json:"functions"`
	DataIDs      []string              `json:"dataIds,omitempty"`
	Dependencies []ProjectedDependency `json:"dependencies,omitempty"`
}

// ProjectionReport is a deterministic per-subsystem projection of write/read/dependency edges.
type ProjectionReport struct {
	SchemaVersion  string               `json:"schemaVersion"`
	GeneratedAt    string               `json:"generatedAt"`
	BaseCommit     string               `json:"baseCommit"`
	AuthorityLimit string               `json:"authorityLimit"`
	Subsystems     []ProjectedSubsystem `json:"subsystems"`
}

// BuildProjectionReport deterministically constructs per-subsystem projections from an assignment.
//
// The projection is derived-only and read-only:
//   - DataIDs lists the caller-supplied data identities touched by this subsystem's functions.
//   - Dependencies lists invocation edges whose caller belongs to this subsystem. It is a
//     projection of delegation, not evidence of a direct write and not accepted ownership.
//
// Callers must still apply a disclosure-safe Public Slice before returning this to any consumer.
func BuildProjectionReport(assign map[string]string, writes []Write, readEdges []ReadEdge, invocations []InvocationEdge, functions []string, dataIDs []string) ProjectionReport {
	subsystemMap := map[string]*ProjectedSubsystem{}
	for _, fn := range functions {
		sub, ok := assign[fn]
		if !ok {
			continue
		}
		if _, ok := subsystemMap[sub]; !ok {
			subsystemMap[sub] = &ProjectedSubsystem{Subsystem: sub}
		}
		subsystemMap[sub].Functions = append(subsystemMap[sub].Functions, ProjectedFunction{Function: fn})
	}
	// touched collects data identities per subsystem from write and read occurrences.
	touched := map[string]map[string]bool{}
	for _, w := range writes {
		sub, ok := assign[w.Function]
		if !ok || subsystemMap[sub] == nil {
			continue
		}
		if touched[sub] == nil {
			touched[sub] = map[string]bool{}
		}
		touched[sub][w.Data] = true
		fnIdx := indexOfFunction(subsystemMap[sub].Functions, w.Function)
		if fnIdx < 0 {
			continue
		}
		subsystemMap[sub].Functions[fnIdx].Writes = append(subsystemMap[sub].Functions[fnIdx].Writes, w.Data)
	}
	for _, r := range readEdges {
		sub, ok := assign[r.Function]
		if !ok || subsystemMap[sub] == nil {
			continue
		}
		if touched[sub] == nil {
			touched[sub] = map[string]bool{}
		}
		touched[sub][r.Data] = true
		fnIdx := indexOfFunction(subsystemMap[sub].Functions, r.Function)
		if fnIdx < 0 {
			continue
		}
		subsystemMap[sub].Functions[fnIdx].Reads = append(subsystemMap[sub].Functions[fnIdx].Reads, r.Data)
	}
	for _, inv := range invocations {
		sub, ok := assign[inv.Caller]
		if !ok || subsystemMap[sub] == nil {
			continue
		}
		calleeSub, assigned := assign[inv.Callee]
		if !assigned {
			calleeSub = ""
		}
		subsystemMap[sub].Dependencies = append(subsystemMap[sub].Dependencies, ProjectedDependency{
			Caller: inv.Caller, Callee: inv.Callee, CalleeSubsystem: calleeSub,
			Internal: assigned && calleeSub == sub, Pattern: inv.Pattern, SourceLocator: inv.SourceLocator,
		})
	}
	subsystems := make([]ProjectedSubsystem, 0, len(subsystemMap))
	for _, sub := range subsystemMap {
		sort.Slice(sub.Functions, func(i, j int) bool { return sub.Functions[i].Function < sub.Functions[j].Function })
		for _, fn := range sub.Functions {
			sort.Strings(fn.Writes)
			sort.Strings(fn.Reads)
		}
		// only report data identities the caller supplied, so unlisted ids cannot leak in.
		dataSet := touched[sub.Subsystem]
		if dataSet != nil && len(dataIDs) > 0 {
			ids := make([]string, 0, len(dataSet))
			for _, d := range dataIDs {
				if dataSet[d] {
					ids = append(ids, d)
				}
			}
			sub.DataIDs = ids
		}
		sort.Strings(sub.DataIDs)
		sort.Slice(sub.Dependencies, func(i, j int) bool {
			if sub.Dependencies[i].Caller != sub.Dependencies[j].Caller {
				return sub.Dependencies[i].Caller < sub.Dependencies[j].Caller
			}
			return sub.Dependencies[i].Callee < sub.Dependencies[j].Callee
		})
		subsystems = append(subsystems, *sub)
	}
	sort.Slice(subsystems, func(i, j int) bool { return subsystems[i].Subsystem < subsystems[j].Subsystem })
	return ProjectionReport{
		SchemaVersion:  "tei.subsystem-partition.projection/v0",
		GeneratedAt:    "",
		BaseCommit:     "",
		AuthorityLimit: AuthorityLimit,
		Subsystems:     subsystems,
	}
}

func indexOfFunction(fns []ProjectedFunction, id string) int {
	for i, f := range fns {
		if f.Function == id {
			return i
		}
	}
	return -1
}
