// Package servicecarrierconnect is a reference that builds the C-2 connect
// point (servicehost.SelectedServiceCarrierInvoker) outside a test, so the two
// connect points named in the decision material can be compared on the same
// footing.
//
// C-2 carries a Service request through a Host-pinned generation. Unlike C-1
// (deployment execution), it requires a Coordinator, a Runtime registry and a
// Host-owned route source. This reference assembles the invoker from explicit
// Host-supplied inputs and reports what it actually did.
//
// It performs no external effect: the process launcher and the acceptance
// reader are supplied by the caller.
package servicecarrierconnect

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"tei/internal/composition"
	"tei/internal/core"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/servicehost"
)

const (
	// Schema identifies the report. It is a reference output, not a product
	// contract.
	Schema = "tei.reference.service-carrier-connect/v1"
	// Authority states what the report is allowed to claim.
	Authority = "derived-reference"

	RuleID      = "tei.reference.service-carrier-connect"
	RuleVersion = "v1"
)

// rulePolicy is covered by the fingerprint.
const rulePolicy = `service-carrier-connect/v1
selection: the Host supplies the pinned Binding and the route for it
dispatch: servicehost.SelectedServiceCarrierInvoker.Invoke
assembled by this reference: BindHostApplicationEnvironmentRoute + NewSelectedServiceCarrierInvoker
refused when: the coordinator or route source is missing, the route is
              unavailable, or the route is for a different pinned Binding
after invocation: ranCount is actual carrier entry, observed at the call; a
              refusal by the Host guard happens before entry and is counted as
              refusedCount; unverifiedCount is a subset of ranCount where the
              carrier was entered and then returned an error or an unusable
              answer
not verified: that the Provider's answer reflects external state
not issued: authentication, authority, permission, or a durable receipt
`

// Refusal codes name why the reference did not run.
const (
	RefusedInvalidInput     = "CONNECT_INPUT_INVALID"
	RefusedRouteUnavailable = "ROUTE_UNAVAILABLE"
	RefusedBindingMismatch  = "ROUTE_BINDING_MISMATCH"
	// RefusedGuard names a refusal by the Host guard. It happens before the
	// carrier is entered, so it is a refusal and not an invocation.
	RefusedGuard         = "CARRIER_GENERATION_UNAVAILABLE"
	UnverifiedInvocation = "SERVICE_INVOCATION_UNVERIFIED"
	UnverifiedGeneration = "SERVICE_RESULT_GENERATION_MISMATCH"
)

// Outcome is one invocation's result.
type Outcome struct {
	RequestID  string `json:"requestId"`
	Generation uint64 `json:"generation"`
	Pinned     bool   `json:"pinnedGeneration"`
	// CarrierInvoked records entry into the pinned Service carrier only. It
	// does not establish an external effect or a durable receipt.
	CarrierInvoked bool `json:"carrierInvoked,omitempty"`
	// Payload is what the Provider returned, verbatim.
	Payload string `json:"payload,omitempty"`
	// Error records a refusal or a carrier error as text. It is never a success.
	Error string `json:"error,omitempty"`
}

// Report is the whole run.
type Report struct {
	Schema          string    `json:"schema"`
	Authority       string    `json:"authority"`
	RuleID          string    `json:"ruleId"`
	RuleVersion     string    `json:"ruleVersion"`
	RuleFingerprint string    `json:"ruleFingerprint"`
	Application     string    `json:"application"`
	Environment     string    `json:"environment"`
	Outcomes        []Outcome `json:"outcomes"`
	// Ran counts entries into the pinned carrier, regardless of error or
	// answer usability. Refused is strictly before that call. Unverified is a
	// subset of Ran: the carrier ran but returned an error or an unusable
	// answer, so an external effect may already exist. None of these counters
	// establishes a business effect.
	Ran        int      `json:"ranCount"`
	Refused    int      `json:"refusedCount"`
	Unverified int      `json:"unverifiedCount"`
	Notes      []string `json:"notes"`
}

// Inputs is what a Host must supply. The reference invents none of them.
type Inputs struct {
	Application core.CanonicalID
	Environment core.CanonicalID
	// Current reads the Host's authoritative route for the pinned Binding. It
	// is the same callback BindHostApplicationEnvironmentRoute takes.
	Current servicehost.CurrentApplicationServiceRoute
	// Binding is the Host's pinned generation.
	Binding composition.Binding
	// Selected is the Host's composition selection for the Binding.
	Selected composition.Selection
	// Registry is the Host's composition registry, already holding the Provider.
	Registry *composition.Registry
	// Guard checks the pinned Binding before each invocation.
	Guard func(composition.Binding) error
	// Runtime is the Host's registry. The reference never builds one.
	Runtime pruntime.Registry
}

// Request is one Service request.
type Request struct {
	RequestID string
	Payload   json.RawMessage
}

// Run assembles the C-2 connect point and invokes each request in RequestID
// order. Any refusal is reported as a value, never as a panic or an abort.
func Run(in Inputs, requests []Request) Report {
	report := Report{
		Schema: Schema, Authority: Authority,
		RuleID: RuleID, RuleVersion: RuleVersion,
		RuleFingerprint: fingerprint(),
		Application:     string(in.Application),
		Environment:     string(in.Environment),
		Outcomes:        []Outcome{},
		Notes:           notes(),
	}
	refuseAll := func(code, detail string) Report {
		report.Refused = len(requests)
		for _, req := range requests {
			report.Outcomes = append(report.Outcomes, Outcome{
				RequestID: req.RequestID, Generation: in.Binding.Generation,
				Error: code + ": " + detail,
			})
		}
		return report
	}
	if in.Current == nil || in.Registry == nil || in.Guard == nil {
		return refuseAll(RefusedInvalidInput, "a current route source, a registry and a guard are required")
	}

	// Assemble the same two pieces a Host would: the scoped route source and
	// the invoker that pins the generation.
	routes, err := servicehost.BindHostApplicationEnvironmentRoute(in.Application, in.Environment, in.Current)
	if err != nil {
		return refuseAll(RefusedInvalidInput, "bind the Host route: "+err.Error())
	}
	coordinator, err := composition.NewSelectedCoordinator(
		in.Registry, in.Selected, in.Binding, composition.NewScope(), in.Guard,
	)
	if err != nil {
		return refuseAll(RefusedInvalidInput, "build the Host coordinator: "+err.Error())
	}
	invoker, err := servicehost.NewSelectedServiceCarrierInvoker(coordinator, in.Runtime, routes)
	if err != nil {
		return refuseAll(RefusedInvalidInput, "build the service carrier invoker: "+err.Error())
	}

	// Ask for the route first, so a missing route is a refusal rather than an
	// invocation that fails for an unrelated reason.
	route, err := routes(context.Background(), in.Binding)
	if err != nil {
		return refuseAll(RefusedRouteUnavailable, err.Error())
	}
	if route.Binding != in.Binding {
		return refuseAll(RefusedBindingMismatch, "the route is for a different pinned Binding")
	}

	ordered := append([]Request(nil), requests...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].RequestID < ordered[j].RequestID })
	for _, req := range ordered {
		// The carrier is entered inside Invoke, after the Host guard. "Ran" must
		// therefore be observed, not assumed: a guard rejection happens before
		// entry and is a refusal, not an invocation.
		invocation, invokeErr := invoker.Invoke(context.Background(), req.RequestID, req.Payload)
		if invokeErr != nil {
			if isPreEntryRefusal(invokeErr) {
				// The Host guard refused before the carrier ran. Nothing was
				// invoked, so this is a refusal.
				report.Refused++
				report.Outcomes = append(report.Outcomes, Outcome{
					RequestID: req.RequestID, Generation: in.Binding.Generation,
					CarrierInvoked: false,
					Error:          RefusedGuard + ": " + invokeErr.Error(),
				})
				continue
			}
			// The carrier was entered and then failed.
			report.Ran++
			report.Unverified++
			report.Outcomes = append(report.Outcomes, Outcome{
				RequestID: req.RequestID, Generation: in.Binding.Generation,
				CarrierInvoked: true,
				Error:          UnverifiedInvocation + ": " + invokeErr.Error(),
			})
			continue
		}
		report.Ran++
		if invocation.Generation != in.Binding {
			report.Unverified++
			report.Outcomes = append(report.Outcomes, Outcome{
				RequestID: req.RequestID, Generation: invocation.Generation.Generation,
				CarrierInvoked: true,
				Error:          UnverifiedGeneration + ": the answer is not for the pinned generation",
			})
			continue
		}
		report.Outcomes = append(report.Outcomes, Outcome{
			RequestID:      req.RequestID,
			Generation:     invocation.Generation.Generation,
			Pinned:         true,
			CarrierInvoked: true,
			Payload:        string(invocation.Service.Runtime.Response.Payload),
		})
	}
	return report
}

func notes() []string {
	return []string{
		"ranCount counts carrier entry, refusedCount only pre-invocation and unverifiedCount post-invocation errors or an answer for another generation",
		"an answer is the Provider responding, not proof that external state changed",
		"the coordinator, runtime registry and route source are supplied by the Host",
		"the Host's pinned Binding is compared with the route before anything runs",
		"no authentication, authority, permission or durable receipt is issued here",
	}
}

func fingerprint() string { return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(rulePolicy))) }

// Marshal renders the report as stable, indented JSON.
func (r Report) Marshal() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }

// isPreEntryRefusal reports whether the invoker refused before entering the
// carrier. The invoker maps both a Host-guard rejection and a route failure to
// ErrServiceCarrierGenerationUnavailable, and neither of those runs the carrier.
func isPreEntryRefusal(err error) bool {
	return errors.Is(err, servicehost.ErrServiceCarrierGenerationUnavailable)
}
