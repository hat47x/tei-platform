package webapicontract

import (
	"sort"
	"strconv"
	"strings"

	"tei/internal/core"
)

// FixtureScope is a trusted test-only authorization snapshot. Its tenant list
// is not a replacement for the real Policy/Row Scope Provider.
type FixtureScope struct {
	PrincipalID    string
	RowScopeID     string
	PolicyRevision string
	Tenants        map[string]bool
}

type FixtureRow struct {
	TenantID string
	Values   map[core.CanonicalID]any
}

// FixtureReadResult is deliberately a public-only projection: no raw row,
// unselected attribute or internal provider evidence can be serialized from it.
type FixtureReadResult struct {
	Columns []string
	Rows    [][]any
}

// ExecuteScopedFixtureRead is a small IN-MEMORY, non-SQL example of actual
// scope -> filter -> ordering -> limit -> projection enforcement. It is NOT a
// general Query Executor or an OData implementation. Its backing rows, tenant
// identities and scope are supplied by trusted test fixtures, not the browser.
func ExecuteScopedFixtureRead(
	publication PublishedBrowse,
	plan ReadPlan,
	support ProviderReadSupport,
	scope FixtureScope,
	rows []FixtureRow,
) (FixtureReadResult, error) {
	if err := CheckReadDispatch(publication, plan, support); err != nil {
		return FixtureReadResult{}, err
	}
	if scope.PrincipalID == "" || scope.PrincipalID != plan.PrincipalID ||
		scope.RowScopeID == "" || scope.RowScopeID != plan.ScopeID ||
		scope.PolicyRevision == "" || scope.PolicyRevision != plan.PolicyRevision ||
		len(scope.Tenants) == 0 {
		return FixtureReadResult{}, ErrRejected
	}
	allowed := false
	for tenant, enabled := range scope.Tenants {
		if tenant == "" {
			return FixtureReadResult{}, ErrRejected
		}
		allowed = allowed || enabled
	}
	if !allowed {
		return FixtureReadResult{}, ErrRejected
	}
	attributes := make(map[string]core.CanonicalID, len(publication.Fields))
	for _, field := range publication.Fields {
		if field.Name == "" || field.AttributeID == "" {
			return FixtureReadResult{}, ErrRejected
		}
		attributes[field.Name] = field.AttributeID
	}
	// First restrict by trusted tenant membership. Filtering, sort and Top
	// MUST NOT be applied to the full, unscoped backing rows.
	scoped := make([]FixtureRow, 0, len(rows))
	for _, row := range rows {
		if row.TenantID == "" || row.Values == nil {
			return FixtureReadResult{}, ErrRejected
		}
		if !scope.Tenants[row.TenantID] {
			continue
		}
		scoped = append(scoped, row)
	}
	if plan.Filter != nil {
		attribute := attributes[plan.Filter.FieldID]
		filtered := make([]FixtureRow, 0, len(scoped))
		for _, row := range scoped {
			value, ok := row.Values[attribute]
			if !ok {
				return FixtureReadResult{}, ErrRejected
			}
			cmp, err := compareFixtureLiteral(value, plan.Filter.Value)
			if err != nil {
				return FixtureReadResult{}, ErrRejected
			}
			if fixtureComparisonMatches(cmp, plan.Filter.Operator) {
				filtered = append(filtered, row)
			} else if !validFixtureOperator(plan.Filter.Operator) {
				return FixtureReadResult{}, ErrRejected
			}
		}
		scoped = filtered
	}
	if plan.Order != nil {
		attribute := attributes[plan.Order.FieldID]
		var sortFailure bool
		sort.SliceStable(scoped, func(i, j int) bool {
			a, okA := scoped[i].Values[attribute]
			b, okB := scoped[j].Values[attribute]
			if !okA || !okB {
				sortFailure = true
				return false
			}
			cmp, err := compareFixtureValues(a, b)
			if err != nil {
				sortFailure = true
				return false
			}
			if plan.Order.Descending {
				return cmp > 0
			}
			return cmp < 0
		})
		if sortFailure {
			return FixtureReadResult{}, ErrRejected
		}
	}
	if len(scoped) > plan.Limit {
		scoped = scoped[:plan.Limit]
	}
	result := FixtureReadResult{Columns: append([]string(nil), plan.Selected...)}
	for _, row := range scoped {
		projected := make([]any, len(plan.Selected))
		for i, name := range plan.Selected {
			value, ok := row.Values[attributes[name]]
			if !ok {
				return FixtureReadResult{}, ErrRejected
			}
			projected[i] = value
		}
		result.Rows = append(result.Rows, projected)
	}
	return result, nil
}

func validFixtureOperator(operator string) bool {
	switch operator {
	case "eq", "ne", "gt", "ge", "lt", "le":
		return true
	}
	return false
}

func fixtureComparisonMatches(cmp int, operator string) bool {
	switch operator {
	case "eq":
		return cmp == 0
	case "ne":
		return cmp != 0
	case "gt":
		return cmp > 0
	case "ge":
		return cmp >= 0
	case "lt":
		return cmp < 0
	case "le":
		return cmp <= 0
	}
	return false
}

func compareFixtureLiteral(value any, raw string) (int, error) {
	switch actual := value.(type) {
	case string:
		expected, ok := parseFixtureString(raw)
		if !ok {
			return 0, ErrRejected
		}
		return strings.Compare(actual, expected), nil
	case int64:
		expected, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return 0, ErrRejected
		}
		return compareFixtureInt(actual, expected), nil
	case bool:
		if raw != "true" && raw != "false" {
			return 0, ErrRejected
		}
		expected := raw == "true"
		return compareFixtureBool(actual, expected), nil
	default:
		return 0, ErrRejected
	}
}

func compareFixtureValues(a, b any) (int, error) {
	switch left := a.(type) {
	case string:
		right, ok := b.(string)
		if ok {
			return strings.Compare(left, right), nil
		}
	case int64:
		right, ok := b.(int64)
		if ok {
			return compareFixtureInt(left, right), nil
		}
	case bool:
		right, ok := b.(bool)
		if ok {
			return compareFixtureBool(left, right), nil
		}
	}
	return 0, ErrRejected
}

func compareFixtureInt(a, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareFixtureBool(a, b bool) int {
	if a == b {
		return 0
	}
	if !a {
		return -1
	}
	return 1
}

// OData-style single-quoted strings escape a quote by doubling it.
// Reject malformed raw literals even if a caller forges a ReadPlan.
func parseFixtureString(raw string) (string, bool) {
	if len(raw) < 2 || raw[0] != '\'' || raw[len(raw)-1] != '\'' {
		return "", false
	}
	inner := raw[1 : len(raw)-1]
	var out strings.Builder
	for i := 0; i < len(inner); i++ {
		if inner[i] != '\'' {
			out.WriteByte(inner[i])
			continue
		}
		if i+1 >= len(inner) || inner[i+1] != '\'' {
			return "", false
		}
		out.WriteByte('\'')
		i++
	}
	return out.String(), true
}
