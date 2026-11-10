// Package bindingguardedexecutor is a read-only reference that closes the gap
// recorded in ADAPTER_BINDING_BLINDSPOT_2026-11-09: the DeploymentExecutor
// adapter holds a hard-coded CapabilityID and never consults the Environment's
// binding, so it can run an executor the binding did not select.
//
// The reference does not change the adapter. It shows the caller-side fix: look
// up the executor through routing.RouteCommand first, and refuse to hand the
// adapter an executor the binding did not select. The adapter then keeps working
// unchanged, but its hard-coded id can no longer disagree with the binding.
package bindingguardedexecutor

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"tei/internal/core"
	"tei/internal/deployment/executor"
	"tei/internal/deployment/routing"
	"tei/internal/environment"
)

const (
	Schema    = "tei.reference.binding-guarded-executor/v0"
	Authority = "derived-reference"

	RuleID      = "tei.reference.binding-guarded-executor"
	RuleVersion = "v0"
)

const rulePolicy = `binding-guarded-executor/v0
resolve: routing.RouteCommand over the Environment's deployment-executor binding
compare: the id the caller wants against the id the binding selects
refuse: a request for an id that is not the bound one, before any executor runs
counted: served (the requested id is the bound one and the Host registered it),
         refused (no binding, another id, or not registered)
not verified: that the executor works, that its artifact is trusted, or that an
              acknowledgement reflects external state
not issued: authentication, authority, permission, or a durable receipt
`

// Status names why a request did not produce an executor.
const (
	StatusServed     = "SERVED"
	StatusNotBound   = "NOT_BOUND"
	StatusNotMatched = "NOT_MATCHED"
	StatusUnknown    = "UNKNOWN_EXECUTOR"
)

// Outcome is one request.
type Outcome struct {
	ActionID string `json:"actionId"`
	// RequestedID is the id the caller wants to run. For the DeploymentExecutor
	// adapter this is its hard-coded CapabilityID.
	RequestedID string `json:"requestedId"`
	// BoundID is what routing.RouteCommand selected. Empty when no binding
	// matched.
	BoundID string `json:"boundId,omitempty"`
	Status  string `json:"status"`
	// Served reports that the requested id is the bound one and registered.
	Served bool   `json:"served"`
	Detail string `json:"detail,omitempty"`
}

// Report is the whole run.
type Report struct {
	Schema          string    `json:"schema"`
	Authority       string    `json:"authority"`
	RuleID          string    `json:"ruleId"`
	RuleVersion     string    `json:"ruleVersion"`
	RuleFingerprint string    `json:"ruleFingerprint"`
	Environment     string    `json:"environment"`
	Outcomes        []Outcome `json:"outcomes"`
	// Served counts requests that matched the binding. Refused counts every
	// other status.
	Served  int      `json:"servedCount"`
	Refused int      `json:"refusedCount"`
	Notes   []string `json:"notes"`
}

// Request is one lookup: a command plus the id the caller wants to run.
type Request struct {
	Command     executor.Command
	RequestedID core.CanonicalID
}

// Input is what the Host supplies. The reference builds no registry.
type Input struct {
	Environment environment.Environment
	// Registered is the set of executor ids the Host actually holds. The
	// reference never constructs an executor.
	Registered []core.CanonicalID
	Requests   []Request
}

// Run performs the lookups. It does not execute anything: the point is whether
// the guard would have handed the adapter an executor the binding did not select.
func Run(in Input) Report {
	registered := make(map[core.CanonicalID]bool, len(in.Registered))
	for _, id := range in.Registered {
		registered[id] = true
	}

	report := Report{
		Schema: Schema, Authority: Authority,
		RuleID: RuleID, RuleVersion: RuleVersion,
		RuleFingerprint: fingerprint(),
		Environment:     string(in.Environment.ID),
		Outcomes:        []Outcome{},
		Notes:           notes(),
	}

	for _, req := range in.Requests {
		outcome := Resolve(in.Environment, req, registered)
		if outcome.Served {
			report.Served++
		} else {
			report.Refused++
		}
		report.Outcomes = append(report.Outcomes, outcome)
	}
	return report
}

// Resolve is the guard itself: given a command and the id the caller wants, it
// answers whether that id is the one the Environment's binding selects.
func Resolve(
	env environment.Environment, req Request, registered map[core.CanonicalID]bool,
) Outcome {
	outcome := Outcome{ActionID: req.Command.ActionID, RequestedID: string(req.RequestedID)}

	routed, err := routing.RouteCommand(env, req.Command)
	if err != nil {
		outcome.Status = StatusNotBound
		outcome.Detail = err.Error()
		return outcome
	}
	outcome.BoundID = string(routed.ExecutorID)

	if req.RequestedID != routed.ExecutorID {
		outcome.Status = StatusNotMatched
		outcome.Detail = "the environment binds a different executor for this action"
		return outcome
	}
	if !registered[req.RequestedID] {
		outcome.Status = StatusUnknown
		outcome.Detail = "the binding names an executor the Host did not register"
		return outcome
	}
	outcome.Status = StatusServed
	outcome.Served = true
	return outcome
}

func notes() []string {
	return []string{
		"the guard compares the requested id with routing.RouteCommand's result",
		"a request for another id is refused before any executor runs",
		"the adapter itself is unchanged; the guard sits in the caller's lookup",
		"the guard does not prove the executor works or that an effect occurred",
		"no authentication, authority, permission or durable receipt is issued here",
	}
}

func fingerprint() string { return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(rulePolicy))) }

// Marshal renders the report as stable, indented JSON.
func (r Report) Marshal() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
