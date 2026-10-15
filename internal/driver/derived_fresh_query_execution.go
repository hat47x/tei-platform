package driver

import (
	"context"
	"fmt"
	"time"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver/queryresult"
	"tei/internal/query"
	"tei/internal/semantic/derivationref"
)

// ExecuteFreshDerivedBrowseQuery is an opt-in D2-to-D3 adapter. It leaves the
// normal Browse Query path untouched, and neither constructs current Authority
// nor infers SQL evidence, source observations or the evaluation clock.
// All of those inputs must come from the trusted Host/Provider boundary.
// A cache hit carries only the derived result, not a fabricated raw query result.
func (r Resolver) ExecuteFreshDerivedBrowseQuery(
	ctx context.Context,
	applicationID, patternID core.CanonicalID,
	executor CompleteBrowseQueryExecutor,
	parameters map[string]any,
	plan query.DerivedSelect,
	sqlEvidence query.CompleteSourceEvidence,
	authority FieldReadAuthorization,
	memo *query.DerivedMemo,
	witness derivationref.CacheWitness,
	policy derivationref.CachePolicy,
	now time.Time,
	verifier derivationref.CurrentWitnessVerifier,
) (query.DerivedSelectResult, bool, error) {
	resolved, err := r.ResolvePattern(applicationID, patternID)
	if err != nil {
		return query.DerivedSelectResult{}, false, err
	}
	if resolved.Pattern.Kind != application.PatternBrowse || resolved.Query == nil || resolved.Datasource == nil {
		return query.DerivedSelectResult{}, false, fmt.Errorf("%w: unresolved browse Query or Datasource", query.ErrDerivedUnsupported)
	}
	if executor == nil || authority == nil {
		return query.DerivedSelectResult{}, false, fmt.Errorf("%w: executor/authority missing", query.ErrDerivedUnsupported)
	}
	canRead := func(attribute core.CanonicalID) bool {
		return authority.CanReadAttribute(ctx, attribute)
	}
	load := query.CompleteDerivedSource(func(loadCtx context.Context, maxRows int) ([]query.DerivedRow, bool, error) {
		if maxRows < 1 || maxRows != plan.MaxRows {
			return nil, false, query.ErrDerivedIncomplete
		}
		observed, complete, err := executor.ExecuteCompleteBrowseQuery(
			loadCtx, *resolved.Query, *resolved.Datasource, parameters, maxRows)
		if err != nil {
			return nil, false, fmt.Errorf("complete browse Query %s failed: %w", resolved.Query.ID, err)
		}
		if !complete || len(observed.Raw.Rows) > maxRows {
			return nil, false, query.ErrDerivedIncomplete
		}
		mapped, err := queryresult.Map(*resolved.Query, observed.Raw)
		if err != nil {
			return nil, false, fmt.Errorf("derived Query %s result mapping failed: %w", resolved.Query.ID, err)
		}
		rows, err := derivedRowsFromMappedResult(mapped)
		if err != nil {
			return nil, false, err
		}
		return rows, true, nil
	})
	// Resolve owns SQL/FieldRead preflight and post-load validation, witness
	// requalification, SourceAsOf age policy and per-hit current verification.
	return memo.Resolve(ctx, *resolved.Query, plan, sqlEvidence, witness,
		policy, now, verifier, canRead, load)
}
