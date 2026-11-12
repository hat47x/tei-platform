package partition

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// AccessOperation is one of C/R/U/D.
type AccessOperation string

const (
	OpCreate AccessOperation = "C"
	OpRead   AccessOperation = "R"
	OpUpdate AccessOperation = "U"
	OpDelete AccessOperation = "D"
)

// Directness distinguishes direct occurrences from indirect/side-effect ones.
type Directness string

const (
	DirectnessDirect    Directness = "direct"
	DirectnessIndirect  Directness = "indirect"
	DirectnessExternal  Directness = "external-action"
	DirectnessDelegated Directness = "delegated"
)

// AccessOccurrence is one edge in the Function-Data bipartite graph.
// Read edges and Write edges are stored separately.
type AccessOccurrence struct {
	ID            string          `json:"id"`
	Function      string          `json:"function"`
	Data          string          `json:"data"` // caller-qualified identity
	Operation     AccessOperation `json:"operation"`
	Provenance    ProvenanceKind  `json:"provenance"`
	Directness    Directness      `json:"directness"`
	SourceLocator string          `json:"sourceLocator"`
	Owner         string          `json:"owner"`
	Multiplicity  int             `json:"multiplicity"` // >=1 distinct evidences on the same relation
	Coverage      string          `json:"coverage"`
	Staleness     string          `json:"staleness"`
	Notes         string          `json:"notes"`
}

// AccessGraph separates Read edges and Write (C/U/D) edges.
type AccessGraph struct {
	Functions   []string           `json:"functions"`
	DataIDs     []string           `json:"dataIds"`
	ReadEdges   []AccessOccurrence `json:"readEdges"`
	WriteEdges  []AccessOccurrence `json:"writeEdges"`
	Invocations []InvocationEdge   `json:"invocations"`
}

// InvocationEdge is a Function→Function delegation (not a direct write).
type InvocationEdge struct {
	Caller        string         `json:"caller"`
	Callee        string         `json:"callee"`
	Pattern       string         `json:"pattern"`
	Provenance    ProvenanceKind `json:"provenance"`
	SourceLocator string         `json:"sourceLocator"`
	Notes         string         `json:"notes"`
}

// BuildAccessGraph deterministically constructs a graph from caller-supplied edges.
func BuildAccessGraph(functions []string, dataIDs []string, edges []AccessOccurrence, invocations []InvocationEdge) (AccessGraph, error) {
	fnSet := map[string]bool{}
	dataSet := map[string]bool{}
	for _, f := range functions {
		f = strings.TrimSpace(f)
		if f == "" {
			return AccessGraph{}, fmt.Errorf("invalid function identity")
		}
		fnSet[f] = true
	}
	for _, d := range dataIDs {
		d = strings.TrimSpace(d)
		if d == "" {
			return AccessGraph{}, fmt.Errorf("invalid data identity")
		}
		dataSet[d] = true
	}
	sortedFn := make([]string, 0, len(fnSet))
	for k := range fnSet {
		sortedFn = append(sortedFn, k)
	}
	sort.Strings(sortedFn)
	sortedData := make([]string, 0, len(dataSet))
	for k := range dataSet {
		sortedData = append(sortedData, k)
	}
	sort.Strings(sortedData)

	readEdges := make([]AccessOccurrence, 0)
	writeEdges := make([]AccessOccurrence, 0)
	seen := map[string]bool{}
	for _, e := range edges {
		e.ID = strings.TrimSpace(e.ID)
		e.Function = strings.TrimSpace(e.Function)
		e.Data = strings.TrimSpace(e.Data)
		if e.ID == "" || e.Function == "" || e.Data == "" {
			return AccessGraph{}, fmt.Errorf("edge missing identity")
		}
		if !fnSet[e.Function] || !dataSet[e.Data] {
			return AccessGraph{}, fmt.Errorf("edge refers to unknown function or data: %s -> %s", e.Function, e.Data)
		}
		if seen[e.ID] {
			return AccessGraph{}, fmt.Errorf("duplicate edge id %q", e.ID)
		}
		seen[e.ID] = true
		switch e.Operation {
		case OpRead:
			readEdges = append(readEdges, e)
		case OpCreate, OpUpdate, OpDelete:
			writeEdges = append(writeEdges, e)
		default:
			return AccessGraph{}, fmt.Errorf("edge operation must be C/R/U/D: %s", e.ID)
		}
	}
	sort.Slice(readEdges, func(i, j int) bool {
		if readEdges[i].ID != readEdges[j].ID {
			return readEdges[i].ID < readEdges[j].ID
		}
		return readEdges[i].Data < readEdges[j].Data
	})
	sort.Slice(writeEdges, func(i, j int) bool {
		if writeEdges[i].ID != writeEdges[j].ID {
			return writeEdges[i].ID < writeEdges[j].ID
		}
		return writeEdges[i].Data < writeEdges[j].Data
	})

	invSet := map[string]bool{}
	for _, inv := range invocations {
		inv.Caller = strings.TrimSpace(inv.Caller)
		inv.Callee = strings.TrimSpace(inv.Callee)
		if inv.Caller == "" || inv.Callee == "" {
			return AccessGraph{}, fmt.Errorf("invocation edge missing identity")
		}
		if !fnSet[inv.Caller] || !fnSet[inv.Callee] {
			return AccessGraph{}, fmt.Errorf("invocation refers to unknown function")
		}
		key := inv.Caller + "|" + inv.Callee
		if invSet[key] {
			return AccessGraph{}, fmt.Errorf("duplicate invocation %s", key)
		}
		invSet[key] = true
	}
	sort.Slice(invocations, func(i, j int) bool {
		if invocations[i].Caller != invocations[j].Caller {
			return invocations[i].Caller < invocations[j].Caller
		}
		return invocations[i].Callee < invocations[j].Callee
	})

	return AccessGraph{
		Functions: sortedFn, DataIDs: sortedData, ReadEdges: readEdges, WriteEdges: writeEdges, Invocations: invocations,
	}, nil
}

// MustMarshalAccessGraph deterministically serializes an AccessGraph.
func MustMarshalAccessGraph(g AccessGraph) []byte {
	b, _ := json.Marshal(g)
	return b
}
