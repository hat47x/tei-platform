package webapicontract

import (
	"context"
	"errors"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/query"
)

var (
	ErrScopedQueryUnavailable = errors.New("scoped query execution unavailable")
	ErrScopedQueryContract    = errors.New("scoped query result contract violation")
)

// ScopedCanonicalExecutor is intentionally NOT the ordinary BrowseQueryExecutor.
// This interface requires the trusted authorization plan as an execution input.
// An ordinary native SQL executor cannot be automatically substituted for it.
type ScopedCanonicalExecutor interface {
	ExecuteScopedBrowseQuery(
		context.Context,
		query.EmbeddedSQL,
		application.DatabaseRequirement,
		ReadPlan,
		map[string]any,
	) (driver.BrowseQueryResult, error)
}

type scopedDriverAdapter struct {
	queryID  core.CanonicalID
	plan     ReadPlan
	provider ScopedCanonicalExecutor
}

func (adapter scopedDriverAdapter) ExecuteBrowseQuery(
	ctx context.Context,
	q query.EmbeddedSQL,
	datasource application.DatabaseRequirement,
	parameters map[string]any,
) (driver.BrowseQueryResult, error) {
	if q.ID == "" || q.ID != adapter.queryID || adapter.provider == nil {
		return driver.BrowseQueryResult{}, ErrScopedQueryContract
	}
	return adapter.provider.ExecuteScopedBrowseQuery(ctx, q, datasource, adapter.plan, parameters)
}

// PublishedReadResult is the ONLY externally publishable output of this
// reference bridge. Raw/Mapped provider data and provider evidence are kept
// inside the call; they are never returned to the HTTP transport.
type PublishedReadResult struct {
	Columns []string
	Rows    [][]any
}

// ExecuteScopedCanonicalRead exercises the REAL Driver's Query execution and
// result-mapping pathway without changing SQL or inventing canonical meaning.
// A selected scoped Provider MUST enforce Scope -> Filter -> Order -> Limit
// before returning rows. This reference can verify plan propagation and safe
// result projection, but cannot independently prove a real Provider did so.
//
// This does not perform security.authorize: callers must bind the plan via
// AuthorizeBrowseRead using a trusted context. Do NOT expose this as an HTTP
// handler without runtime policy/Provider enforcement and fresh revalidation.
func ExecuteScopedCanonicalRead(
	ctx context.Context,
	resolver driver.Resolver,
	publication PublishedBrowse,
	plan ReadPlan,
	support ProviderReadSupport,
	provider ScopedCanonicalExecutor,
	parameters map[string]any,
) (PublishedReadResult, error) {
	if ctx == nil || provider == nil ||
		publication.ApplicationID == "" || publication.PatternID == "" ||
		publication.QueryID == "" || publication.DataID == "" {
		return PublishedReadResult{}, ErrScopedQueryUnavailable
	}
	if err := CheckReadDispatch(publication, plan, support); err != nil {
		return PublishedReadResult{}, err
	}
	// Freeze the authorized publication separately from the object delivered
	// to a Provider. A Provider must not be able to widen the selected fields
	// by mutating a shared slice or pointer after authorization.
	selected := append([]string(nil), plan.Selected...)
	forwarded := plan
	forwarded.Selected = append([]string(nil), plan.Selected...)
	if plan.Filter != nil {
		filter := *plan.Filter
		forwarded.Filter = &filter
	}
	if plan.Order != nil {
		order := *plan.Order
		forwarded.Order = &order
	}
	execution, err := resolver.ExecuteBrowseQuery(
		ctx, publication.ApplicationID, publication.PatternID,
		scopedDriverAdapter{queryID: publication.QueryID, plan: forwarded, provider: provider},
		parameters,
	)
	if err != nil {
		return PublishedReadResult{}, ErrScopedQueryUnavailable
	}
	if execution.Resolution.Query == nil || execution.Resolution.Data == nil ||
		execution.Resolution.Query.ID != publication.QueryID ||
		execution.Resolution.Data.ID != publication.DataID {
		return PublishedReadResult{}, ErrScopedQueryContract
	}
	mapped := execution.Result.Mapped
	indexes := make(map[core.CanonicalID]int, len(mapped.Columns))
	for i, field := range mapped.Columns {
		if field.Attribute == "" {
			return PublishedReadResult{}, ErrScopedQueryContract
		}
		if _, duplicate := indexes[field.Attribute]; duplicate {
			return PublishedReadResult{}, ErrScopedQueryContract
		}
		indexes[field.Attribute] = i
	}
	published := make(map[string]core.CanonicalID, len(publication.Fields))
	for _, field := range publication.Fields {
		if field.Name == "" || field.AttributeID == "" {
			return PublishedReadResult{}, ErrScopedQueryContract
		}
		if _, duplicate := published[field.Name]; duplicate {
			return PublishedReadResult{}, ErrScopedQueryContract
		}
		published[field.Name] = field.AttributeID
	}
	result := PublishedReadResult{Columns: selected}
	selectedIndex := make([]int, len(selected))
	for i, name := range selected {
		attribute, exists := published[name]
		if !exists {
			return PublishedReadResult{}, ErrScopedQueryContract
		}
		index, exists := indexes[attribute]
		if !exists {
			return PublishedReadResult{}, ErrScopedQueryContract
		}
		selectedIndex[i] = index
	}
	for _, row := range mapped.Rows {
		if len(row) != len(mapped.Columns) {
			return PublishedReadResult{}, ErrScopedQueryContract
		}
		projected := make([]any, len(selectedIndex))
		for i, index := range selectedIndex {
			projected[i] = row[index]
		}
		result.Rows = append(result.Rows, projected)
	}
	return result, nil
}
