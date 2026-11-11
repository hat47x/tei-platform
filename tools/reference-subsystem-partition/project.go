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

// ProjectedSubsystem is the ER/CRUD/dependency projection for one subsystem.
type ProjectedSubsystem struct {
	Subsystem string              `json:"subsystem"`
	Functions []ProjectedFunction `json:"functions"`
	DataIDs   []string            `json:"dataIds,omitempty"`
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
	for _, w := range writes {
		sub, ok := assign[w.Function]
		if !ok {
			continue
		}
		if subsystemMap[sub] == nil {
			continue
		}
		fnIdx := -1
		for i, f := range subsystemMap[sub].Functions {
			if f.Function == w.Function {
				fnIdx = i
				break
			}
		}
		if fnIdx < 0 {
			continue
		}
		subsystemMap[sub].Functions[fnIdx].Writes = append(subsystemMap[sub].Functions[fnIdx].Writes, w.Data)
	}
	for _, r := range readEdges {
		sub, ok := assign[r.Function]
		if !ok {
			continue
		}
		if subsystemMap[sub] == nil {
			continue
		}
		fnIdx := -1
		for i, f := range subsystemMap[sub].Functions {
			if f.Function == r.Function {
				fnIdx = i
				break
			}
		}
		if fnIdx < 0 {
			continue
		}
		subsystemMap[sub].Functions[fnIdx].Reads = append(subsystemMap[sub].Functions[fnIdx].Reads, r.Data)
	}
	subsystems := make([]ProjectedSubsystem, 0, len(subsystemMap))
	for _, sub := range subsystemMap {
		sort.Slice(sub.Functions, func(i, j int) bool { return sub.Functions[i].Function < sub.Functions[j].Function })
		for _, fn := range sub.Functions {
			sort.Strings(fn.Writes)
			sort.Strings(fn.Reads)
		}
		sort.Strings(sub.DataIDs)
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
