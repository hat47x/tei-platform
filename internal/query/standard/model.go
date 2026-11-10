package standard

// SourceSpan marks the byte range of a parsed construct in the original SQL.
type SourceSpan struct {
	Start int `json:"start,omitempty"`
	End   int `json:"end,omitempty"`
}

type Severity string

const (
	SeverityError       Severity = "error"
	SeverityWarning     Severity = "warning"
	SeverityUnsupported Severity = "unsupported"
)

// Diagnostic records a parse/lower/generate finding with optional source span.
type Diagnostic struct {
	Code     string     `json:"code"`
	Message  string     `json:"message"`
	Span     SourceSpan `json:"span,omitempty"`
	Severity Severity   `json:"severity,omitempty"`
}

// Expr is a dialect-neutral query expression.
type Expr interface{}

type Column struct {
	Qualifier string `json:"qualifier,omitempty"`
	Name      string `json:"name"`
}

type Literal struct {
	Kind  string `json:"kind"` // number | string | null | bool | star
	Value string `json:"value"`
}

type Parameter struct {
	Name string `json:"name"`
}

type BinaryOp struct {
	Op    string `json:"op"`
	Left  Expr   `json:"left"`
	Right Expr   `json:"right"`
}

type UnaryOp struct {
	Op      string `json:"op"`
	Operand Expr   `json:"operand"`
}

type FunctionCall struct {
	Name     string `json:"name"`
	Distinct bool   `json:"distinct,omitempty"`
	Args     []Expr `json:"args,omitempty"`
}

type IsNull struct {
	Negated bool `json:"negated,omitempty"`
	Expr    Expr `json:"expr"`
}

type InList struct {
	Negated  bool           `json:"negated,omitempty"`
	Expr     Expr           `json:"expr"`
	List     []Expr         `json:"list,omitempty"`
	Subquery *StandardQuery `json:"subquery,omitempty"`
}

type Between struct {
	Expr Expr `json:"expr"`
	Low  Expr `json:"low"`
	High Expr `json:"high"`
}

type Exists struct {
	Negated bool           `json:"negated,omitempty"`
	Query   *StandardQuery `json:"query"`
}

type Cast struct {
	Expr Expr   `json:"expr"`
	Type string `json:"type"`
}

type Case struct {
	Value Expr   `json:"value,omitempty"`
	Whens []When `json:"whens"`
	Else  Expr   `json:"else,omitempty"`
}

type When struct {
	Cond Expr `json:"cond"`
	Then Expr `json:"then"`
}

type SubqueryExpr struct {
	Query   *StandardQuery `json:"query"`
	Lateral bool           `json:"lateral,omitempty"`
}

// Node is a relational algebra operator.
type Node interface{}

type Scan struct {
	ID         string `json:"id"`
	Source     string `json:"source"`
	SourceKind string `json:"sourceKind,omitempty"`
	Alias      string `json:"alias,omitempty"`
	SubqueryID string `json:"subqueryId,omitempty"`
}

type Filter struct {
	NodeID    string `json:"nodeId"`
	Condition Expr   `json:"condition"`
	Having    bool   `json:"having,omitempty"`
}

type Project struct {
	NodeID string        `json:"nodeId"`
	Items  []ProjectItem `json:"items"`
}

type ProjectItem struct {
	Expr  Expr   `json:"expr"`
	Alias string `json:"alias,omitempty"`
}

type Join struct {
	NodeID    string   `json:"nodeId"`
	Kind      string   `json:"kind"`
	Left      string   `json:"left"`
	Right     string   `json:"right"`
	Condition Expr     `json:"condition,omitempty"`
	Using     []string `json:"using,omitempty"`
}

type Aggregate struct {
	NodeID   string        `json:"nodeId"`
	Grouping []Expr        `json:"grouping,omitempty"`
	Items    []ProjectItem `json:"items"`
}

type Sort struct {
	NodeID string    `json:"nodeId"`
	Keys   []SortKey `json:"keys"`
}

type SortKey struct {
	Expr       Expr  `json:"expr"`
	Desc       bool  `json:"desc"`
	NullsFirst *bool `json:"nullsFirst,omitempty"`
}

type Limit struct {
	NodeID string `json:"nodeId"`
	Limit  string `json:"limit"`
	Offset string `json:"offset,omitempty"`
}

type Distinct struct {
	NodeID string `json:"nodeId"`
}

type Union struct {
	NodeID string `json:"nodeId"`
	All    bool   `json:"all"`
	Left   string `json:"left"`
	Right  string `json:"right"`
}

type Values struct {
	NodeID string   `json:"nodeId"`
	Rows   [][]Expr `json:"rows"`
}

type Subquery struct {
	ID    string         `json:"id"`
	Query *StandardQuery `json:"query"`
	Alias string         `json:"alias,omitempty"`
}

type SubqueryScan struct {
	NodeID     string `json:"nodeId"`
	SubqueryID string `json:"subqueryId"`
	Alias      string `json:"alias,omitempty"`
}

type Opaque struct {
	NodeID string `json:"nodeId"`
	SQL    string `json:"sql"`
	Reason string `json:"reason"`
}

type CTE struct {
	ID    string         `json:"id"`
	Query *StandardQuery `json:"query"`
}

// StandardQuery is a dialect-neutral, serializable query representation.
type StandardQuery struct {
	Dialect     string       `json:"dialect"`
	CTEs        []CTE        `json:"ctes,omitempty"`
	Nodes       []Node       `json:"nodes,omitempty"`
	Root        string       `json:"root"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}
