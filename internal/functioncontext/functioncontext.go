// Package functioncontext describes what one executed browse Pattern actually
// observed: which Data it is about, under which retrieval conditions, and how much
// of the target set the observation covers.
//
// It owns no Source syntax, Provider, Registry or second canonical. The type
// vocabulary belongs to #2506; this package only produces the first concrete
// instance from an existing Driver execution so #2586 can observe it end to end.
// Claims, Pattern application and Intent candidates are out of scope.
package functioncontext

import (
	"fmt"
	"sort"
	"time"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/environment"
	"tei/internal/semantic"
)

// Grain says what one result row stands for.
type Grain string

const (
	// GrainEntity: a declared candidate-key of the Data is fully present in the mapped columns.
	GrainEntity Grain = "entity"
	// GrainQueryRow: no declared candidate-key is covered; rows are not claimed to be entities.
	GrainQueryRow Grain = "query-row"
)

// SliceState is how the rows were obtained. A failed read never produces a Slice.
type SliceState string

const (
	SliceStatePage  SliceState = "page"
	SliceStateEmpty SliceState = "empty"
)

// CoverageBasisDeclaredPageContract records that coverage is inferred only from the
// Pattern's declared page-size, never from the SQL text.
const CoverageBasisDeclaredPageContract = "declared-page-contract"

// SourceRevisionUnknown is used until #2387 supplies a Source revision.
const SourceRevisionUnknown = "unknown"

type Definition struct {
	ContextID      core.CanonicalID   `json:"contextId"`
	Application    core.CanonicalID   `json:"application"`
	Subject        core.CanonicalID   `json:"subject"`
	RelationPath   []core.CanonicalID `json:"relationPath"`
	Query          core.CanonicalID   `json:"query"`
	Grain          Grain              `json:"grain"`
	IdentityKey    core.CanonicalID   `json:"identityKey,omitempty"`
	IdentityFields []core.CanonicalID `json:"identityFields,omitempty"`
}

type Binding struct {
	Environment     core.CanonicalID             `json:"environment"`
	Instance        core.CanonicalID             `json:"instance"`
	Definition      core.CanonicalID             `json:"definition"`
	Provider        environment.DatabaseProvider `json:"provider"`
	DatasourceAlias core.LocalName               `json:"datasourceAlias"`
	ParameterNames  []string                     `json:"parameterNames,omitempty"`
}

type FilterScope struct {
	Attribute core.CanonicalID `json:"attribute"`
	Operator  string           `json:"operator"`
	Value     any              `json:"value"`
}

type SortScope struct {
	Attribute core.CanonicalID `json:"attribute"`
	Direction string           `json:"direction"`
}

type PageScope struct {
	Size   int   `json:"size"`
	Offset int64 `json:"offset"`
}

// Scope holds only conditions the Pattern declared, not proof that native SQL
// or the provider applied them. Other inputs appear by name.
type Scope struct {
	Filters     []FilterScope `json:"filters,omitempty"`
	Sorts       []SortScope   `json:"sorts,omitempty"`
	Page        *PageScope    `json:"page,omitempty"`
	OtherInputs []string      `json:"otherInputs,omitempty"`
}

// Row carries the business identity of one row. Its position in the page is not
// an operation target.
type Row struct {
	Identity map[core.CanonicalID]any `json:"identity,omitempty"`
}

type Slice struct {
	State          SliceState         `json:"state"`
	RowCount       int                `json:"rowCount"`
	Columns        []core.CanonicalID `json:"columns"`
	Rows           []Row              `json:"rows"`
	Coverage       semantic.Coverage  `json:"coverage"`
	CoverageBasis  string             `json:"coverageBasis"`
	EmptyJustified bool               `json:"emptyJustified"`
	ObservedAt     time.Time          `json:"observedAt"`
	SourceRevision string             `json:"sourceRevision"`
}

type Context struct {
	Definition Definition `json:"definition"`
	Binding    Binding    `json:"binding"`
	Scope      Scope      `json:"scope"`
	Slice      Slice      `json:"slice"`
}

// FromBrowseExecution derives the Context from one executed browse Pattern.
// requested is the caller's input as given (before the Driver injected page-size).
// parameterNames are the executor-reported bound parameter names (names only).
func FromBrowseExecution(
	exec driver.BrowseQueryEnvironmentExecution,
	requested map[string]any,
	parameterNames []string,
	observedAt time.Time,
) (Context, error) {
	res := exec.Execution.Resolution
	if res.Data == nil || res.Query == nil || res.Datasource == nil {
		return Context{}, fmt.Errorf("browse Pattern %sのData・Query・Datasourceが解決されていません", res.Pattern.ID)
	}
	pattern := res.Pattern
	mapped := exec.Execution.Result.Mapped

	columns := make([]core.CanonicalID, len(mapped.Columns))
	columnIndex := map[core.CanonicalID]int{}
	for i, column := range mapped.Columns {
		columns[i] = column.Attribute
		columnIndex[column.Attribute] = i
	}

	definition := Definition{
		ContextID:    pattern.ID,
		Application:  res.Application.ID,
		Subject:      res.Data.ID,
		RelationPath: []core.CanonicalID{},
		Query:        res.Query.ID,
		Grain:        GrainQueryRow,
	}
	var identityIdx []int
	for _, key := range res.Data.CandidateKeys {
		idx, ok := indexesOf(key.Attributes, columnIndex)
		if len(key.Attributes) > 0 && ok {
			definition.Grain = GrainEntity
			definition.IdentityKey = key.ID
			definition.IdentityFields = append([]core.CanonicalID(nil), key.Attributes...)
			identityIdx = idx
			break // 宣言順で最初に完全に含まれるkeyを使う。推測で選ばない。
		}
	}

	scope := Scope{}
	declared := map[string]bool{}
	for _, filter := range pattern.Filters {
		if filter.Parameter == "" || filter.OperatorParameter == "" {
			continue
		}
		operator, _ := requested[filter.OperatorParameter].(string)
		scope.Filters = append(scope.Filters, FilterScope{Attribute: filter.Attribute, Operator: operator, Value: requested[filter.Parameter]})
		declared[filter.Parameter] = true
		declared[filter.OperatorParameter] = true
	}
	for _, s := range pattern.Sorts {
		scope.Sorts = append(scope.Sorts, SortScope{Attribute: s.Attribute, Direction: s.Direction})
	}
	if pattern.PageSize > 0 {
		var offset int64
		if pattern.PageOffsetParameter != "" {
			switch v := requested[pattern.PageOffsetParameter].(type) {
			case int:
				offset = int64(v)
			case int32:
				offset = int64(v)
			case int64:
				offset = v
			}
			declared[pattern.PageOffsetParameter] = true
		}
		if pattern.PageSizeParameter != "" {
			declared[pattern.PageSizeParameter] = true
		}
		scope.Page = &PageScope{Size: pattern.PageSize, Offset: offset}
	}
	names := make([]string, 0, len(requested))
	for name := range requested {
		if !declared[name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	scope.OtherInputs = names

	rows := make([]Row, 0, len(mapped.Rows))
	for _, values := range mapped.Rows {
		row := Row{}
		if definition.Grain == GrainEntity {
			row.Identity = map[core.CanonicalID]any{}
			for i, idx := range identityIdx {
				row.Identity[definition.IdentityFields[i]] = textual(values[idx])
			}
		}
		rows = append(rows, row)
	}

	slice := Slice{
		State:          SliceStatePage,
		RowCount:       len(rows),
		Columns:        columns,
		Rows:           rows,
		Coverage:       semantic.CoverageUnestablished,
		CoverageBasis:  CoverageBasisDeclaredPageContract,
		ObservedAt:     observedAt.UTC(),
		SourceRevision: SourceRevisionUnknown,
	}
	if len(rows) == 0 {
		slice.State = SliceStateEmpty
	}
	// A declared LIMIT and an exhausted result do not prove that native SQL or
	// the provider covered the declared subject (RLS, extra predicates, hidden
	// LIMITs, and ignored parameters can all yield a short or empty page).
	// Keep coverage unestablished until the execution supplies that evidence.

	return Context{
		Definition: definition,
		Binding: Binding{
			Environment:     exec.Environment,
			Instance:        exec.Instance.ID,
			Definition:      exec.Instance.Definition,
			Provider:        exec.Instance.Provider,
			DatasourceAlias: res.Datasource.Alias,
			ParameterNames:  append([]string(nil), parameterNames...),
		},
		Scope: scope,
		Slice: slice,
	}, nil
}

func indexesOf(attributes []core.CanonicalID, columnIndex map[core.CanonicalID]int) ([]int, bool) {
	out := make([]int, 0, len(attributes))
	for _, attribute := range attributes {
		i, ok := columnIndex[attribute]
		if !ok {
			return nil, false
		}
		out = append(out, i)
	}
	return out, true
}

func textual(v any) any {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return v
}
