package postgres

import (
	"encoding/json"
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/plan"
	"tei/internal/environment"
)

// NormalizeExplainJSON converts PostgreSQL's machine-readable EXPLAIN JSON to
// TEI's provider-neutral Execution Plan IR. Unknown provider attributes remain
// in ProviderDetails so normalization does not discard useful observations.
func NormalizeExplainJSON(queryID, databaseID core.CanonicalID, raw []byte, snapshot *physical.SchemaSnapshot) (plan.IR, error) {
	var docs []map[string]any
	if err := json.Unmarshal(raw, &docs); err != nil {
		return plan.IR{}, fmt.Errorf("postgres EXPLAIN JSON: %w", err)
	}
	if len(docs) != 1 {
		return plan.IR{}, fmt.Errorf("postgres EXPLAIN JSON: expected one statement, got %d", len(docs))
	}
	top := docs[0]
	rawPlan, ok := top["Plan"].(map[string]any)
	if !ok {
		return plan.IR{}, fmt.Errorf("postgres EXPLAIN JSON: missing Plan object")
	}
	out := plan.IR{Query: queryID, Database: databaseID, Provider: environment.ProviderPostgres, Mode: plan.ModeExplain}
	out.PlanningTimeMs = numberPtr(top["Planning Time"])
	out.ExecutionTimeMs = numberPtr(top["Execution Time"])
	out.ProviderDetails = copyExcept(top, "Plan", "Planning Time", "Execution Time")

	hasActual := false
	var walk func(map[string]any, string) string
	walk = func(src map[string]any, id string) string {
		providerKind, _ := src["Node Type"].(string)
		n := plan.Node{
			ID:             id,
			Kind:           normalizePlanNodeKind(providerKind),
			ProviderKind:   providerKind,
			ParentRelation: stringValue(src["Parent Relationship"]),
			Alias:          stringValue(src["Alias"]),
			IndexName:      stringValue(src["Index Name"]),
			Estimate: plan.Estimate{
				StartupCost: numberPtr(src["Startup Cost"]),
				TotalCost:   numberPtr(src["Total Cost"]),
				Rows:        numberPtr(src["Plan Rows"]),
				Width:       integerPtr(src["Plan Width"]),
			},
		}
		relation := stringValue(src["Relation Name"])
		schema := stringValue(src["Schema"])
		if relation != "" {
			if schema != "" {
				n.PhysicalObject = schema + "." + relation
			} else if snapshot != nil {
				if t, ok := snapshot.Table(relation); ok {
					n.PhysicalObject = t.QualifiedName()
				} else {
					n.PhysicalObject = relation
				}
			} else {
				n.PhysicalObject = relation
			}
		}
		actual := plan.Actual{
			StartupTimeMs: numberPtr(src["Actual Startup Time"]),
			TotalTimeMs:   numberPtr(src["Actual Total Time"]),
			Rows:          numberPtr(src["Actual Rows"]),
			Loops:         numberPtr(src["Actual Loops"]),
		}
		if actual.StartupTimeMs != nil || actual.TotalTimeMs != nil || actual.Rows != nil || actual.Loops != nil {
			n.Actual = &actual
			hasActual = true
		}
		n.ProviderDetails = copyExcept(src,
			"Plans", "Node Type", "Parent Relationship", "Relation Name", "Schema", "Alias", "Index Name",
			"Startup Cost", "Total Cost", "Plan Rows", "Plan Width",
			"Actual Startup Time", "Actual Total Time", "Actual Rows", "Actual Loops")
		// Append first so the pre-order IDs remain stable independently of
		// sorting in normalized output.
		out.Nodes = append(out.Nodes, n)
		nodeIndex := len(out.Nodes) - 1
		if children, ok := src["Plans"].([]any); ok {
			for i, child := range children {
				childMap, ok := child.(map[string]any)
				if !ok {
					continue
				}
				childID := fmt.Sprintf("%s.%d", id, i)
				walk(childMap, childID)
				out.Nodes[nodeIndex].Children = append(out.Nodes[nodeIndex].Children, childID)
			}
		}
		return id
	}
	out.Root = walk(rawPlan, "plan:0")
	if hasActual {
		out.Mode = plan.ModeExplainAnalyze
	}
	return out.Normalize(), nil
}

func normalizePlanNodeKind(v string) plan.NodeKind {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "seq scan", "bitmap heap scan", "tid scan", "tid range scan", "subquery scan", "function scan", "table function scan", "values scan", "cte scan", "named tuplestore scan", "worktable scan", "foreign scan", "custom scan", "sample scan":
		return plan.NodeScan
	case "index scan", "index only scan", "bitmap index scan":
		return plan.NodeIndexScan
	case "nested loop", "hash join", "merge join":
		return plan.NodeJoin
	case "aggregate", "group", "windowagg":
		return plan.NodeAggregate
	case "sort", "incremental sort":
		return plan.NodeSort
	case "materialize", "memoize":
		return plan.NodeMaterialize
	case "limit":
		return plan.NodeLimit
	case "modifytable", "insert", "update", "delete", "merge":
		return plan.NodeModify
	case "append", "merge append", "setop", "recursive union":
		return plan.NodeSetOperation
	case "gather", "gather merge":
		return plan.NodeGather
	default:
		return plan.NodeOther
	}
}

func numberPtr(v any) *float64 {
	switch n := v.(type) {
	case float64:
		return &n
	case float32:
		f := float64(n)
		return &f
	case int:
		f := float64(n)
		return &f
	case int64:
		f := float64(n)
		return &f
	case json.Number:
		f, err := n.Float64()
		if err == nil {
			return &f
		}
	}
	return nil
}

func integerPtr(v any) *int64 {
	p := numberPtr(v)
	if p == nil {
		return nil
	}
	n := int64(*p)
	return &n
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func copyExcept(src map[string]any, keys ...string) map[string]any {
	skip := map[string]bool{}
	for _, k := range keys {
		skip[k] = true
	}
	out := map[string]any{}
	for k, v := range src {
		if !skip[k] {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
