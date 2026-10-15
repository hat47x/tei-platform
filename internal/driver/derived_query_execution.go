package driver

import (
	"context"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver/queryresult"
	"tei/internal/query"
	"tei/internal/semantic/derivationref"
)

// CompleteBrowseQueryExecutor is opt-in. It must fetch an unpaginated, complete
// source of at most maxRows from the unchanged native SQL and attest completion.
// A normal BrowseQueryExecutor MUST NOT be silently treated as complete.
type CompleteBrowseQueryExecutor interface {
	ExecuteCompleteBrowseQuery(context.Context, query.EmbeddedSQL, application.DatabaseRequirement, map[string]any, int) (BrowseQueryResult, bool, error)
}

// FieldReadAuthorization is supplied by the current Host/Provider authority.
// A D1 pure operator is not itself a grant to read protected inputs.
type FieldReadAuthorization interface {
	CanReadAttribute(context.Context, core.CanonicalID) bool
}

type DerivedBrowseQueryExecution struct {
	Browse  BrowseQueryExecution
	Derived query.DerivedSelectResult
}

// ExecuteDerivedBrowseQuery connects D2 to the existing Pattern resolution,
// native Query execution and explicit raw-to-semantic mapping. Authorization
// and source-shape preflight happen before any Provider invocation.
func (r Resolver) ExecuteDerivedBrowseQuery(
	ctx context.Context,
	applicationID, patternID core.CanonicalID,
	executor CompleteBrowseQueryExecutor,
	parameters map[string]any,
	plan query.DerivedSelect,
	source query.CompleteSourceEvidence,
	authority FieldReadAuthorization,
) (DerivedBrowseQueryExecution, error) {
	resolved, err := r.ResolvePattern(applicationID, patternID)
	if err != nil {
		return DerivedBrowseQueryExecution{}, err
	}
	if resolved.Pattern.Kind != application.PatternBrowse || resolved.Query == nil || resolved.Datasource == nil {
		return DerivedBrowseQueryExecution{}, fmt.Errorf("%w: unresolved browse Query or Datasource", query.ErrDerivedUnsupported)
	}
	if executor == nil || authority == nil {
		return DerivedBrowseQueryExecution{}, fmt.Errorf("%w: executor/authority missing", query.ErrDerivedUnsupported)
	}
	if err := query.ValidateDerivedSelect(*resolved.Query, plan, source, func(attribute core.CanonicalID) bool {
		return authority.CanReadAttribute(ctx, attribute)
	}); err != nil {
		return DerivedBrowseQueryExecution{}, err
	}

	observed, complete, err := executor.ExecuteCompleteBrowseQuery(ctx, *resolved.Query, *resolved.Datasource, parameters, plan.MaxRows)
	if err != nil {
		return DerivedBrowseQueryExecution{}, fmt.Errorf("complete browse Query %s failed: %w", resolved.Query.ID, err)
	}
	if !complete || len(observed.Raw.Rows) > plan.MaxRows {
		return DerivedBrowseQueryExecution{}, query.ErrDerivedIncomplete
	}
	mapped, err := queryresult.Map(*resolved.Query, observed.Raw)
	if err != nil {
		return DerivedBrowseQueryExecution{}, fmt.Errorf("derived Query %s result mapping failed: %w", resolved.Query.ID, err)
	}
	observed.Mapped = mapped
	rows, err := derivedRowsFromMappedResult(mapped)
	if err != nil {
		return DerivedBrowseQueryExecution{}, err
	}
	selected, err := query.EvaluateDerivedSelect(plan, rows, complete)
	if err != nil {
		return DerivedBrowseQueryExecution{}, err
	}
	return DerivedBrowseQueryExecution{
		Browse:  BrowseQueryExecution{Resolution: resolved, Result: observed},
		Derived: selected,
	}, nil
}

// derivedRowsFromMappedResult preserves D2's explicit Query result mapping and
// lossless decimal conversion for both direct D2 and opt-in D3 execution.
func derivedRowsFromMappedResult(mapped queryresult.MappedResult) ([]query.DerivedRow, error) {
	rows := make([]query.DerivedRow, 0, len(mapped.Rows))
	for _, rawRow := range mapped.Rows {
		values := make(map[core.CanonicalID]derivationref.Input, len(rawRow))
		for i, raw := range rawRow {
			val, err := exactDerivedInput(raw)
			if err != nil {
				return nil, fmt.Errorf("%w: unsupported raw value for %s: %v", query.ErrDerivedUnsupported, mapped.Columns[i].Attribute, err)
			}
			values[mapped.Columns[i].Attribute] = val
		}
		rows = append(rows, query.DerivedRow{Values: values})
	}
	return rows, nil
}

func exactDerivedInput(raw any) (derivationref.Input, error) {
	if raw == nil {
		return derivationref.Input{State: derivationref.Null}, nil
	}
	switch v := raw.(type) {
	case string:
		return derivationref.Input{State: derivationref.Present, Decimal: v}, nil
	case []byte:
		return derivationref.Input{State: derivationref.Present, Decimal: string(v)}, nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return derivationref.Input{State: derivationref.Present, Decimal: fmt.Sprint(v)}, nil
	default:
		// Float values have already lost decimal scale/precision. No implicit
		// conversion to a supposedly exact monetary value.
		return derivationref.Input{}, fmt.Errorf("value type %T has no lossless D2 conversion", raw)
	}
}
