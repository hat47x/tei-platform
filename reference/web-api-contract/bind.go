// Package webapicontract is an isolated reference for binding external read
// requests and write commands to a typed, authorization-aware contract.
// It is not a production OData server or a TEI runtime authorization engine.
package webapicontract

import (
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var ErrRejected = errors.New("request rejected: unsupported or unauthorized expression")

// FieldPolicy is a synthetic per-invocation authorization decision supplied
// by a trusted server-side caller, never by a browser query parameter.
type FieldPolicy struct {
	Select bool
	Filter bool
	Sort   bool
	Write  bool
}

type Field struct {
	ID     string
	Policy FieldPolicy
}

// TrustedReadContext must be provided by the real Host/authorization provider.
// This reference only verifies nonempty identity; it cannot verify authenticity.
type TrustedReadContext struct {
	PrincipalID    string
	RowScopeID     string
	PolicyRevision string
}

type Filter struct {
	FieldID string
	Operator string
	Value string
}

type Ordering struct {
	FieldID string
	Descending bool
}

type ReadPlan struct {
	Selected []string
	Filter *Filter
	Order *Ordering
	Limit int
	// The executor MUST apply this authorized scope before client filter,
	// count/aggregation and pagination. Scope ID is NOT a SQL predicate.
	ScopeID string
	PolicyRevision string
}

var identifier = regexp.MustCompile("^[A-Za-z][A-Za-z0-9_]*$")
var simpleComparison = regexp.MustCompile("^([A-Za-z][A-Za-z0-9_]*) (eq|ne|gt|ge|lt|le) ('(?:[^']|'')*'|[0-9]+|true|false|null)$")

// BindReadSubset accepts a deliberately tiny URL query subset, suitable for
// testing semantic authorization. It does NOT implement OData 4.01: a real
// protocol adapter must validate the full OData grammar and wire contracts.
func BindReadSubset(values url.Values, fields []Field, ctx TrustedReadContext) (ReadPlan, error) {
	if ctx.PrincipalID == "" || ctx.RowScopeID == "" || ctx.PolicyRevision == "" {
		return ReadPlan{}, ErrRejected
	}
	known := make(map[string]FieldPolicy, len(fields))
	for _, f := range fields {
		if !identifier.MatchString(f.ID) {
			return ReadPlan{}, ErrRejected
		}
		if _, exists := known[f.ID]; exists {
			return ReadPlan{}, ErrRejected
		}
		known[f.ID] = f.Policy
	}
	params := map[string]string{}
	for name, vals := range values {
		key := strings.ToLower(strings.TrimPrefix(name, "$"))
		if (key != "select" && key != "filter" && key != "orderby" && key != "top") || len(vals) != 1 || vals[0] == "" {
			return ReadPlan{}, ErrRejected
		}
		if _, duplicate := params[key]; duplicate {
			return ReadPlan{}, ErrRejected
		}
		params[key] = vals[0]
	}
	plan := ReadPlan{ScopeID: ctx.RowScopeID, PolicyRevision: ctx.PolicyRevision, Limit: 50}
	if selected, ok := params["select"]; ok {
		seen := map[string]bool{}
		for _, name := range strings.Split(selected, ",") {
			name = strings.TrimSpace(name)
			policy, exists := known[name]
			if !exists || !policy.Select || seen[name] {
				return ReadPlan{}, ErrRejected
			}
			seen[name] = true
			plan.Selected = append(plan.Selected, name)
		}
	} else {
		// Default projection is limited to visible fields, never SELECT *.
		for _, field := range fields {
			if field.Policy.Select {
				plan.Selected = append(plan.Selected, field.ID)
			}
		}
	}
	if len(plan.Selected) == 0 {
		return ReadPlan{}, ErrRejected
	}
	if expression, ok := params["filter"]; ok {
		parts := simpleComparison.FindStringSubmatch(expression)
		if parts == nil || !known[parts[1]].Filter {
			return ReadPlan{}, ErrRejected
		}
		plan.Filter = &Filter{FieldID: parts[1], Operator: parts[2], Value: parts[3]}
	}
	if expression, ok := params["orderby"]; ok {
		parts := strings.Fields(expression)
		if len(parts) < 1 || len(parts) > 2 || !known[parts[0]].Sort {
			return ReadPlan{}, ErrRejected
		}
		if len(parts) == 2 && parts[1] != "asc" && parts[1] != "desc" {
			return ReadPlan{}, ErrRejected
		}
		plan.Order = &Ordering{FieldID: parts[0], Descending: len(parts) == 2 && parts[1] == "desc"}
	}
	if raw, ok := params["top"]; ok {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			return ReadPlan{}, ErrRejected
		}
		plan.Limit = n
	}
	return plan, nil
}

// FieldChange deliberately distinguishes clearing a field from setting a null
// value. The real Action contract owns their business semantics.
type FieldChange struct {
	Kind string // "set" or "clear"
	Value any
}

type CommandSpec struct {
	ActionID string
	Allowed bool
	WritableFields map[string]bool
	RequiresRevision bool
	RequiresIdempotency bool
}

type CommandRequest struct {
	ActionID string
	ExpectedRevision string
	IdempotencyKey string
	Changes map[string]FieldChange
}

type CommandIntent struct {
	ActionID string
	ExpectedRevision string
	IdempotencyKey string
	Changes map[string]FieldChange
}

// BindCommand checks a synthetic Action decision; it does not execute the
// command, commit a transaction, or claim exactly-once delivery.
func BindCommand(req CommandRequest, spec CommandSpec, trustedPrincipalID string) (CommandIntent, error) {
	if trustedPrincipalID == "" || req.ActionID == "" || req.ActionID != spec.ActionID || !spec.Allowed {
		return CommandIntent{}, ErrRejected
	}
	if spec.RequiresRevision && req.ExpectedRevision == "" {
		return CommandIntent{}, ErrRejected
	}
	if spec.RequiresIdempotency && req.IdempotencyKey == "" {
		return CommandIntent{}, ErrRejected
	}
	changes := make(map[string]FieldChange, len(req.Changes))
	for name, ch := range req.Changes {
		if !identifier.MatchString(name) || !spec.WritableFields[name] {
			return CommandIntent{}, ErrRejected
		}
		if ch.Kind != "set" && ch.Kind != "clear" {
			return CommandIntent{}, ErrRejected
		}
		if ch.Kind == "clear" && ch.Value != nil {
			return CommandIntent{}, ErrRejected
		}
		changes[name] = ch
	}
	return CommandIntent{ActionID: req.ActionID, ExpectedRevision: req.ExpectedRevision, IdempotencyKey: req.IdempotencyKey, Changes: changes}, nil
}
