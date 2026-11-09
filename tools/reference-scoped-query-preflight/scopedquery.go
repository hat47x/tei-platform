// Package scopedquery decides whether a scoped-query realization may issue SQL against a
// datasource. It takes the scope predicate from the authorization basis, never from the
// client, refuses when the scope column, query/data binding, provider, field or operation
// is not explicitly realized, and refuses to remove unscoped rows after execution. It is
// read-only: it never issues SQL and never writes a Source.
//
// This reference does not prove row-scope enforcement against a real database. It checks
// the pre-execution contract only.
package scopedquery

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
)

const Schema = "tei.reference.scoped-query-preflight/v0"

// Authorization is the trusted host-side basis for the scope predicate.
type Authorization struct {
	Principal      string   `json:"principal"`
	Tenant         string   `json:"tenant"`
	Subject        string   `json:"subject"`
	Purpose        string   `json:"purpose"`
	PolicyRevision string   `json:"policyRevision"`
	Expired        bool     `json:"expired"`
	Scopes         []string `json:"scopes"`
}

// Realization is the explicit scoped query realization: the canonical identities and the
// physical SQL mapping the host declares for this query.
type Realization struct {
	Application string   `json:"application"`
	Pattern     string   `json:"pattern"`
	Query       string   `json:"query"`
	Data        string   `json:"data"`
	Datasource  string   `json:"datasource"`
	Provider    string   `json:"provider"`
	ScopeColumn string   `json:"scopeColumn"`
	Fields      []string `json:"fields"`
	Operations  []string `json:"operations"`
}

// Request is the client request. Its tenant, scope and role are never authoritative.
type Request struct {
	Tenant     string   `json:"tenant"`
	Scope      string   `json:"scope"`
	Role       string   `json:"role"`
	Query      string   `json:"query"`
	Data       string   `json:"data"`
	Filter     string   `json:"filter"`
	Sort       string   `json:"sort"`
	Limit      int      `json:"limit"`
	Fields     []string `json:"fields"`
	Operations []string `json:"operations"`
	// PostFilterRows is true when the caller wants to delete unscoped rows after
	// execution. That is forbidden.
	PostFilterRows bool `json:"postFilterRows"`
}

// Known is the trusted explicit mapping the host allows.
type Known struct {
	Fields     []string `json:"fields"`
	Providers  []string `json:"providers"`
	Operations []string `json:"operations"`
}

// Input is one read-only preflight request.
type Input struct {
	Authorization Authorization `json:"authorization"`
	Realization   Realization   `json:"realization"`
	Request       Request       `json:"request"`
	Known         Known         `json:"known"`
}

// Item is a refusal or finding.
type Item struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
}

// Clause is one clause of the executed plan, in execution order.
type Clause struct {
	Kind   string `json:"kind"` // scope | filter | order | limit
	Detail string `json:"detail"`
}

// Decision is the pre-execution decision.
type Decision struct {
	Execute          bool     `json:"execute"`
	Clauses          []Clause `json:"clauses"`
	BoundParams      []string `json:"boundParams"`
	Identifiers      []string `json:"identifiers"`
	PostFilter       bool     `json:"postFilter"`
	UnscopedFallback bool     `json:"unscopedFallback"`
	Refusals         []Item   `json:"refusals"`
}

// Report is the read-only result.
type Report struct {
	Schema      string   `json:"schema"`
	Decision    Decision `json:"decision"`
	Fingerprint string   `json:"fingerprint"`
}

func set(list []string) map[string]bool {
	m := make(map[string]bool, len(list))
	for _, s := range list {
		m[s] = true
	}
	return m
}

// hardRefusedOperations are not permitted until their behavior is demonstrated.
var hardRefusedOperations = map[string]bool{"count": true, "aggregate": true, "expand": true}

// Plan decides whether the query may issue SQL.
func Plan(in Input) Report {
	refusals := []Item{}
	add := func(code, subject string) {
		refusals = append(refusals, Item{Code: code, Subject: subject})
	}

	if in.Authorization.Expired {
		add("authorization-expired", in.Authorization.PolicyRevision)
	}
	if len(in.Authorization.Scopes) == 0 {
		add("authorization-scope-missing", in.Authorization.Subject)
	}
	// Client-specified tenant, scope and role are never authoritative.
	if (in.Request.Tenant != "" && in.Request.Tenant != in.Authorization.Tenant) ||
		(in.Request.Scope != "" && !set(in.Authorization.Scopes)[in.Request.Scope]) ||
		(in.Request.Role != "" && in.Request.Role != in.Authorization.Subject) {
		add("client-scope-not-authoritative", in.Authorization.Principal)
	}
	if in.Realization.ScopeColumn == "" {
		add("scope-column-missing", in.Realization.Data)
	}
	if in.Realization.Query == "" || in.Realization.Data == "" ||
		(in.Request.Query != "" && in.Request.Query != in.Realization.Query) ||
		(in.Request.Data != "" && in.Request.Data != in.Realization.Data) {
		add("query-data-mismatch", in.Realization.Query)
	}
	if !set(in.Known.Providers)[in.Realization.Provider] {
		add("provider-not-supported", in.Realization.Provider)
	}
	declaredOps := set(in.Realization.Operations)
	knownOps := set(in.Known.Operations)
	for _, op := range in.Request.Operations {
		if hardRefusedOperations[op] {
			add("operation-not-supported", op)
			continue
		}
		if !declaredOps[op] || !knownOps[op] {
			add("operation-not-supported", op)
		}
	}
	declaredFields := set(in.Realization.Fields)
	knownFields := set(in.Known.Fields)
	for _, f := range in.Request.Fields {
		if !declaredFields[f] || !knownFields[f] {
			add("unknown-field", f)
		}
	}
	if in.Request.PostFilterRows {
		add("post-filter-forbidden", in.Authorization.Subject)
	}

	sort.Slice(refusals, func(i, j int) bool {
		if refusals[i].Code != refusals[j].Code {
			return refusals[i].Code < refusals[j].Code
		}
		return refusals[i].Subject < refusals[j].Subject
	})

	decision := Decision{
		Execute:          len(refusals) == 0,
		Clauses:          []Clause{},
		BoundParams:      []string{},
		Identifiers:      []string{},
		PostFilter:       false,
		UnscopedFallback: false,
		Refusals:         refusals,
	}
	if decision.Execute {
		// Scope is applied before the client filter, order and limit.
		decision.Clauses = []Clause{
			{Kind: "scope", Detail: in.Realization.ScopeColumn + " in authorization scopes"},
			{Kind: "filter", Detail: in.Request.Filter},
			{Kind: "order", Detail: in.Request.Sort},
			{Kind: "limit", Detail: fmt.Sprint(in.Request.Limit)},
		}
		decision.BoundParams = []string{in.Authorization.Tenant, in.Request.Filter, in.Request.Sort, fmt.Sprint(in.Request.Limit)}
		ids := append([]string{in.Realization.ScopeColumn}, in.Request.Fields...)
		sort.Strings(ids)
		decision.Identifiers = ids
	}

	return Report{
		Schema:      Schema,
		Decision:    decision,
		Fingerprint: fingerprint(decision),
	}
}

func fingerprint(d Decision) string {
	raw, err := json.Marshal(d)
	if err != nil {
		raw = []byte(fmt.Sprintf("%v", d))
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("sha256:%x", sum)
}
