package plan

import (
	"math"
	"sort"

	"tei/internal/core"
	"tei/internal/environment"
)

type CaptureMode string

const (
	ModeExplain        CaptureMode = "explain"
	ModeExplainAnalyze CaptureMode = "explain-analyze"
)

type NodeKind string

const (
	NodeScan         NodeKind = "scan"
	NodeIndexScan    NodeKind = "index-scan"
	NodeJoin         NodeKind = "join"
	NodeAggregate    NodeKind = "aggregate"
	NodeSort         NodeKind = "sort"
	NodeMaterialize  NodeKind = "materialize"
	NodeLimit        NodeKind = "limit"
	NodeModify       NodeKind = "modify"
	NodeSetOperation NodeKind = "set-operation"
	NodeGather       NodeKind = "gather"
	NodeOther        NodeKind = "other"
)

type Estimate struct {
	StartupCost *float64 `json:"startupCost,omitempty"`
	TotalCost   *float64 `json:"totalCost,omitempty"`
	Rows        *float64 `json:"rows,omitempty"`
	Width       *int64   `json:"width,omitempty"`
}

// Actual values follow PostgreSQL EXPLAIN semantics: rows and node times are
// per-loop averages, while Loops records how often the node ran. Keeping the
// values separate avoids silently multiplying measurements into a different
// metric than the provider reports.
type Actual struct {
	StartupTimeMs *float64 `json:"startupTimeMs,omitempty"`
	TotalTimeMs   *float64 `json:"totalTimeMs,omitempty"`
	Rows          *float64 `json:"rowsPerLoop,omitempty"`
	Loops         *float64 `json:"loops,omitempty"`
}

type Node struct {
	ID              string         `json:"id"`
	Kind            NodeKind       `json:"kind"`
	ProviderKind    string         `json:"providerKind"`
	Children        []string       `json:"children,omitempty"`
	ParentRelation  string         `json:"parentRelation,omitempty"`
	PhysicalObject  string         `json:"physicalObject,omitempty"`
	Alias           string         `json:"alias,omitempty"`
	IndexName       string         `json:"indexName,omitempty"`
	Estimate        Estimate       `json:"estimate"`
	Actual          *Actual        `json:"actual,omitempty"`
	ProviderDetails map[string]any `json:"providerDetails,omitempty"`
}

type IR struct {
	Query           core.CanonicalID             `json:"query"`
	Database        core.CanonicalID             `json:"database"`
	Provider        environment.DatabaseProvider `json:"provider"`
	Mode            CaptureMode                  `json:"mode"`
	Root            string                       `json:"root"`
	Nodes           []Node                       `json:"nodes"`
	PlanningTimeMs  *float64                     `json:"planningTimeMs,omitempty"`
	ExecutionTimeMs *float64                     `json:"executionTimeMs,omitempty"`
	ProviderDetails map[string]any               `json:"providerDetails,omitempty"`
}

func (p IR) Node(id string) (Node, bool) {
	for _, n := range p.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return Node{}, false
}

// RowEstimateRatio compares the provider's per-loop actual row count with the
// estimated rows for the same node. A value of 1 means the estimate matched;
// values above/below 1 show direction without claiming a diagnosis.
func RowEstimateRatio(n Node) (float64, bool) {
	if n.Estimate.Rows == nil || n.Actual == nil || n.Actual.Rows == nil || *n.Estimate.Rows == 0 {
		return 0, false
	}
	v := *n.Actual.Rows / *n.Estimate.Rows
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

func (p IR) Normalize() IR {
	out := p
	out.Nodes = append([]Node(nil), p.Nodes...)
	sort.Slice(out.Nodes, func(i, j int) bool { return out.Nodes[i].ID < out.Nodes[j].ID })
	for i := range out.Nodes {
		out.Nodes[i].Children = append([]string(nil), out.Nodes[i].Children...)
	}
	return out
}
