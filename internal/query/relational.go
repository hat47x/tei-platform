package query

type RelNodeKind string

const (
	RelSource    RelNodeKind = "source"
	RelJoin      RelNodeKind = "join"
	RelFilter    RelNodeKind = "filter"
	RelAggregate RelNodeKind = "aggregate"
	RelProject   RelNodeKind = "project"
	RelDistinct  RelNodeKind = "distinct"
	RelSort      RelNodeKind = "sort"
	RelLimit     RelNodeKind = "limit"
	RelOpaque    RelNodeKind = "opaque"
)

type RelNode struct {
	ID          string      `json:"id"`
	Kind        RelNodeKind `json:"kind"`
	ScopeID     string      `json:"scopeId,omitempty"`
	Inputs      []string    `json:"inputs,omitempty"`
	Source      string      `json:"source,omitempty"`
	SourceKind  string      `json:"sourceKind,omitempty"`
	Binding     string      `json:"binding,omitempty"`
	Alias       string      `json:"alias,omitempty"`
	Resolution  string      `json:"resolution,omitempty"`
	JoinKind    string      `json:"joinKind,omitempty"`
	Expressions []string    `json:"expressions,omitempty"`
}

type RelationalIR struct {
	Nodes []RelNode `json:"nodes,omitempty"`
	Root  string    `json:"root,omitempty"`
}
