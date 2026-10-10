package modelref

import (
	"fmt"
	"sort"
)

// TraceOptions selects only direct declared references. There is no inferred
// transitive business impact or implicit wildcard scope.
type TraceOptions struct {
	Direction string `json:"direction"` // outgoing, incoming, both
	Mode      string `json:"mode"`      // summary, full
	Offset    int    `json:"offset,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type TraceError struct {
	Code    string
	Message string
}

func (e *TraceError) Error() string { return e.Message }

// TraceRow preserves the original edge and its evidence, even if the target
// cannot be resolved. Only rows with Status "resolved" have a ResolvedTo key.
// A resolved row is still a graph candidate, not Accepted meaning.
type TraceRow struct {
	Status     string `json:"status"` // resolved, unresolved
	Edge       Edge   `json:"edge"`
	ResolvedTo *Key   `json:"resolvedTo,omitempty"`
	Issue      string `json:"issue,omitempty"`
}

// TraceResult is an indexed, stable read of a candidate graph. "summary" only
// returns counts. "full" has no implicit truncation; a supplied positive Limit
// enables explicit paging through NextOffset.
type TraceResult struct {
	SchemaVersion string      `json:"schemaVersion"`
	Subject       Key         `json:"subject"`
	Direction     string      `json:"direction"`
	Mode          string      `json:"mode"`
	Authority     string      `json:"authority"`
	Total         int         `json:"total"`
	Unresolved    int         `json:"unresolved"`
	Offset        int         `json:"offset,omitempty"`
	Rows          []TraceRow  `json:"rows,omitempty"`
	NextOffset    *int        `json:"nextOffset,omitempty"`
}

// TraceDirect selects and validates one-hop edges from the existing typed
// Graph. It never fabricates a relation from matching names or Source paths.
// Rules must be supplied by the caller's semantic extension, not the CLI.
func TraceDirect(graph Graph, rules []Rule, subject Key, options TraceOptions) (TraceResult, error) {
	if graph.Schema != "tei.modelref.v0" {
		return TraceResult{}, &TraceError{Code: "UNSUPPORTED_SCHEMA", Message: "unsupported model reference graph schema"}
	}
	if !validKey(subject) {
		return TraceResult{}, &TraceError{Code: "INVALID_SUBJECT", Message: "scope, kind, and id are required"}
	}
	if options.Direction == "" {
		options.Direction = "both"
	}
	if options.Mode == "" {
		options.Mode = "full"
	}
	switch options.Direction {
	case "outgoing", "incoming", "both":
	default:
		return TraceResult{}, &TraceError{Code: "INVALID_ARGUMENT", Message: "direction must be outgoing, incoming, or both"}
	}
	if options.Mode != "summary" && options.Mode != "full" {
		return TraceResult{}, &TraceError{Code: "INVALID_ARGUMENT", Message: "mode must be summary or full"}
	}
	if options.Offset < 0 || options.Limit < 0 || options.Mode == "summary" && (options.Offset != 0 || options.Limit != 0) {
		return TraceResult{}, &TraceError{Code: "INVALID_ARGUMENT", Message: "offset/limit require nonnegative values and full mode"}
	}
	counts := make(map[Key]int, len(graph.Nodes))
	for _, node := range graph.Nodes {
		counts[node.Key]++
	}
	if counts[subject] == 0 {
		return TraceResult{}, &TraceError{Code: "SUBJECT_NOT_FOUND", Message: "subject does not exist in typed graph"}
	}
	if counts[subject] != 1 {
		return TraceResult{}, &TraceError{Code: "SUBJECT_AMBIGUOUS", Message: "subject is declared more than once in typed graph"}
	}

	type ruleKey struct{ relation, fromKind, toKind string }
	type ruleValue struct {
		rule     Rule
		conflict bool
	}
	rulesByKey := make(map[ruleKey]ruleValue, len(rules))
	for _, rule := range rules {
		key := ruleKey{rule.Relation, rule.FromKind, rule.ToKind}
		previous, exists := rulesByKey[key]
		if exists && previous.rule != rule {
			previous.conflict = true
			rulesByKey[key] = previous
		} else if !exists {
			rulesByKey[key] = ruleValue{rule: rule}
		}
	}
	lookup := newLookup(graph.Nodes)
	rows := make([]TraceRow, 0)
	unresolved := 0
	for _, edge := range graph.Edges {
		outgoing := edge.From == subject
		// A scope-less ref may or may not resolve to the requested subject.
		// Keep it visible as unresolved evidence when resolution is ambiguous.
		incoming := edge.To.Subject == subject.Subject &&
			(edge.To.Scope == "" || edge.To.Scope == subject.Scope)
		if options.Direction == "outgoing" && !outgoing ||
			options.Direction == "incoming" && !incoming ||
			options.Direction == "both" && !outgoing && !incoming {
			continue
		}
		row := TraceRow{Status: "unresolved", Edge: edge}
		switch {
		case counts[edge.From] == 0:
			row.Issue = "invalid-source"
		case counts[edge.From] > 1:
			row.Issue = "ambiguous-source"
		default:
			dest, issue := lookup.resolve(edge.To)
			if issue != "" {
				row.Issue = issue
			} else {
				rule, exists := rulesByKey[ruleKey{edge.Relation, edge.From.Subject.Kind, dest.Subject.Kind}]
				switch {
				case !exists:
					row.Issue = "invalid-edge-type"
				case rule.conflict:
					row.Issue = "conflicting-rule"
				default:
					findings := evidenceDiagnostics("edge", edge.Evidence)
					if len(findings) != 0 {
						row.Issue = findings[0].Code
					} else {
						row.Status = "resolved"
						row.ResolvedTo = &dest
					}
				}
			}
		}
		if row.Status == "unresolved" {
			unresolved++
		}
		rows = append(rows, row)
	}
	// Deterministic traversal is independent of insertion order, physical file
	// location, and iteration order of any maps used for rule resolution.
	sort.Slice(rows, func(i, j int) bool {
		return traceRowKey(rows[i]) < traceRowKey(rows[j])
	})
	result := TraceResult{
		SchemaVersion: "tei.modelref.trace/v0", Subject: subject,
		Direction: options.Direction, Mode: options.Mode, Authority: "candidate-read-only",
		Total: len(rows), Unresolved: unresolved,
	}
	if options.Mode == "summary" {
		return result, nil
	}
	start := options.Offset
	if start > len(rows) {
		start = len(rows)
	}
	end := len(rows)
	if options.Limit > 0 && options.Limit < end-start {
		end = start + options.Limit
	}
	result.Offset = start
	result.Rows = rows[start:end]
	if end < len(rows) {
		next := end
		result.NextOffset = &next
	}
	return result, nil
}

func traceRowKey(row TraceRow) string {
	to := row.Edge.To
	// Length-prefixed fields prevent collisions when user-defined IDs contain
	// separators (including hyphens, colons, and slashes).
	fields := []string{
		keyString(row.Edge.From), to.Scope, to.Subject.Kind, to.Subject.ID,
		to.Version, row.Edge.Relation, row.Status, row.Issue, evidenceKey(row.Edge.Evidence),
	}
	out := ""
	for _, field := range fields {
		out += fmt.Sprintf("%d:%s", len(field), field)
	}
	return out
}
