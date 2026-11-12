// Package functioncontext projects ONE browse Query execution into a value that states
// what the execution observed and, equally, what it did not establish.
//
// The value is built from the return value of a single execution. It is not stored, not
// registered and not looked up later. Same-Attempt identity is the fact that every field
// comes from the same in-process return value.
//
// It deliberately never reports coverage as complete, never reports an empty result as
// justified, and never infers an identity: those claims need evidence that a native SQL
// Query execution cannot supply.
package functioncontext

import (
	"encoding/json"
	"fmt"
	"time"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/semantic"
)

const (
	StateValue = "value"
	StateNull  = "null"

	// BindingHostAsserted: the Instance is the declaration; that the live connection is that
	// Instance is a statement by the host. TEI does not verify it.
	BindingHostAsserted = "host-asserted"

	NotObserved   = "unobserved"
	NotEvaluated  = "not-evaluated"
	NotDeclared   = "not-declared"
	SourceUnknown = "unknown"

	IdentityUndeclared = "undeclared"
	IdentityDeclared   = "declared"

	// TimeRendering names how time.Time values are rendered, so a Consumer does not see the
	// host's local zone.
	TimeRendering = "utc-rfc3339nano"
)

// ConnectionObservation is what the host observed on the connection it handed to the
// Executor, in the same transaction as the read. It is an observation, not a verification
// of the Binding.
type ConnectionObservation struct {
	Database      string    `json:"database"`
	User          string    `json:"user"`
	ServerVersion string    `json:"serverVersion"`
	ReadOnly      bool      `json:"readOnly"`
	Snapshot      string    `json:"snapshot"`
	ObservedAt    time.Time `json:"observedAt"`
}

// ParameterShape names a runtime parameter and its Go type. The value is never recorded.
// DeclaredByQuery is false when the Query declares no such input.
type ParameterShape struct {
	Name            string `json:"name"`
	GoType          string `json:"goType"`
	DeclaredByQuery bool   `json:"declaredByQuery"`
}

type Input struct {
	Execution  driver.BrowseQueryEnvironmentExecution
	Connection ConnectionObservation
	Parameters []ParameterShape
}

type Cell struct {
	State string `json:"state"`
	Value any    `json:"value,omitempty"`
}

type Column struct {
	Raw       string           `json:"raw"`
	Attribute core.CanonicalID `json:"attribute"`
}

type Context struct {
	Function struct {
		Application core.CanonicalID `json:"application"`
		Pattern     core.CanonicalID `json:"pattern"`
		Query       core.CanonicalID `json:"query"`
		Data        core.CanonicalID `json:"data,omitempty"`
	} `json:"function"`
	Source struct {
		QuerySourceSHA256 string `json:"querySourceSha256"`
		Dialect           string `json:"dialect"`
		SourceRevision    string `json:"sourceRevision"`
	} `json:"source"`
	Binding struct {
		Environment core.CanonicalID `json:"environment"`
		Alias       string           `json:"alias"`
		Instance    core.CanonicalID `json:"instance"`
		Definition  core.CanonicalID `json:"definition"`
		Provider    string           `json:"provider"`
		Assertion   string           `json:"assertion"`
	} `json:"binding"`
	Connection ConnectionObservation `json:"connection"`
	Parameters []ParameterShape      `json:"parameters"`
	Rowset     struct {
		RawColumns []string `json:"rawColumns"`
		RowCount   int      `json:"rowCount"`
		// FullyRead: the cursor was read to its end without error. Not a claim about the
		// relation behind the Query.
		FullyRead bool `json:"fullyRead"`
	} `json:"rowset"`
	Columns []Column `json:"columns"`
	Rows    [][]Cell `json:"rows"`
	// NullOrigin: a null may be a stored NULL or a value hidden by the database (a masking
	// view returns NULL). The execution cannot tell them apart.
	NullOrigin string `json:"nullOrigin"`
	// MissingFields: a mapped column absent from the result, or an unmapped column present
	// in it, is refused by the mapping before this value exists.
	MissingFields string `json:"missingFields"`
	Grain         struct {
		Identity string `json:"identity"`
		// Key is the Owner-declared candidate key of the Data, if any. It is never inferred.
		Key               []core.CanonicalID `json:"key,omitempty"`
		ExactDuplicateRow int                `json:"exactDuplicateRows"`
	} `json:"grain"`
	Coverage struct {
		Status         semantic.Coverage `json:"status"`
		EmptyJustified bool              `json:"emptyJustified"`
		Pagination     string            `json:"pagination"`
		Visibility     string            `json:"visibility"`
	} `json:"coverage"`
	Currency   string `json:"currency"`
	TimeFormat string `json:"timeRendering"`
	Disclosure struct {
		Columns int    `json:"columns"`
		Rows    int    `json:"rows"`
		Omitted string `json:"omitted"`
	} `json:"disclosure"`
}

// From projects one execution. Every state it cannot establish is written as an explicit
// unobserved/undeclared/not-evaluated value rather than omitted or inferred.
func From(in Input) (Context, error) {
	exec := in.Execution
	res := exec.Execution.Resolution
	if res.Query == nil || res.Datasource == nil {
		return Context{}, fmt.Errorf("execution has no resolved Query or Datasource")
	}
	if exec.QuerySourceSHA256 == "" {
		return Context{}, fmt.Errorf("execution carries no Query Source digest")
	}
	raw, mapped := exec.Execution.Result.Raw, exec.Execution.Result.Mapped
	if len(mapped.Rows) != len(raw.Rows) {
		return Context{}, fmt.Errorf("mapped rows %d != raw rows %d", len(mapped.Rows), len(raw.Rows))
	}

	var c Context
	c.Function.Application = res.Application.ID
	c.Function.Pattern = res.Pattern.ID
	c.Function.Query = res.Query.ID
	c.Source.QuerySourceSHA256 = exec.QuerySourceSHA256
	c.Source.Dialect = string(res.Query.Dialect)
	c.Source.SourceRevision = SourceUnknown
	c.Binding.Environment = exec.Environment
	c.Binding.Alias = string(res.Datasource.Alias)
	c.Binding.Instance = exec.Instance.ID
	c.Binding.Definition = exec.Instance.Definition
	c.Binding.Provider = string(exec.Instance.Provider)
	c.Binding.Assertion = BindingHostAsserted
	c.Connection = in.Connection
	c.Connection.ObservedAt = in.Connection.ObservedAt.UTC()
	c.Parameters = append([]ParameterShape(nil), in.Parameters...)
	c.Rowset.RawColumns = append([]string(nil), raw.Columns...)
	c.Rowset.RowCount = len(raw.Rows)
	c.Rowset.FullyRead = true
	for _, column := range mapped.Columns {
		c.Columns = append(c.Columns, Column{Raw: column.Raw, Attribute: column.Attribute})
	}

	seen := map[string]int{}
	c.Rows = make([][]Cell, 0, len(mapped.Rows))
	for _, row := range mapped.Rows {
		if len(row) != len(mapped.Columns) {
			return Context{}, fmt.Errorf("mapped row width %d != columns %d", len(row), len(mapped.Columns))
		}
		cells := make([]Cell, len(row))
		for i, value := range row {
			cells[i] = cell(value)
		}
		key, err := json.Marshal(cells)
		if err != nil {
			return Context{}, err
		}
		seen[string(key)]++
		c.Rows = append(c.Rows, cells)
	}
	for _, n := range seen {
		if n > 1 {
			c.Grain.ExactDuplicateRow += n - 1
		}
	}

	c.NullOrigin = "unestablished"
	c.MissingFields = "refused-by-mapping"
	c.Grain.Identity = IdentityUndeclared
	if res.Data != nil {
		c.Function.Data = res.Data.ID
		if len(res.Data.CandidateKeys) > 0 {
			c.Grain.Identity = IdentityDeclared
			c.Grain.Key = append([]core.CanonicalID(nil), res.Data.CandidateKeys[0].Attributes...)
		}
	}
	c.Coverage.Status = semantic.CoverageUnestablished
	c.Coverage.EmptyJustified = false
	c.Coverage.Pagination = NotDeclared
	c.Coverage.Visibility = NotObserved
	c.Currency = NotEvaluated
	c.TimeFormat = TimeRendering
	c.Disclosure.Columns = len(c.Columns)
	c.Disclosure.Rows = len(c.Rows)
	c.Disclosure.Omitted = "none: every fetched row and every mapped column"
	return c, nil
}

func cell(value any) Cell {
	switch v := value.(type) {
	case nil:
		return Cell{State: StateNull}
	case time.Time:
		return Cell{State: StateValue, Value: v.UTC().Format(time.RFC3339Nano)}
	case []byte:
		return Cell{State: StateValue, Value: string(v)}
	default:
		return Cell{State: StateValue, Value: v}
	}
}
