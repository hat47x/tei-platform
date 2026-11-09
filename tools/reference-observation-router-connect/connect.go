// Package observationrouterconnect is a read-only reference for the
// ObservationProviders gap: the binding is already resolved by product code
// (observation.Binding, called from `cmd/tei -plan`), but nothing outside a test
// assembles observation.Router, so Observe never runs in a product entry.
//
// This reference does the smallest thing that closes that gap: it takes the
// Environment and a Host-supplied provider registry, assembles the existing
// Router, and calls Observe once per declared kind. It reports which kind was
// resolved, which provider was selected, and whether that provider was actually
// entered.
//
// It does not add a resolver, a second registry, or a new trust source.
package observationrouterconnect

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/observation"
	"tei/internal/environment"
)

const (
	// Schema identifies the report. It is a reference output, not a product
	// contract.
	Schema = "tei.reference.observation-router-connect/v0"
	// Authority states what the report is allowed to claim.
	Authority = "derived-reference"

	RuleID      = "tei.reference.observation-router-connect"
	RuleVersion = "v0"
)

const rulePolicy = `observation-router-connect/v0
resolution: observation.Binding over the Environment's observation-provider declarations
assembly: observation.Router{Providers: <Host-supplied>}
entry: observation.Router.Observe, one call per requested kind
counted: resolved (Binding returned an id), selected (the registry held it),
         invoked (the provider's Observe was entered)
refused when: the binding is missing or ambiguous, the selected id is not in the
              Host registry, or the provider does not conform to the binding
after entry: a failure inside the provider is unknown, not a refusal; it is
             counted separately because the call already happened
not verified: that the fragment reflects external state, that the provider's
              artifact is trusted, or that a grant is current
not issued: authentication, authority, permission, or a durable receipt
`

// Status names why an observation did not complete.
const (
	StatusObserved       = "OBSERVED"
	StatusBindingMissing = "BINDING_MISSING"
	StatusNotRegistered  = "NOT_REGISTERED"
	StatusNotConforming  = "NOT_CONFORMING"
	StatusObserveFailed  = "OBSERVE_FAILED"
	// StatusFragmentInvalid is used when the router answers without entering
	// the provider. A fragment whose identity does not match the binding is
	// rejected by the existing Router and arrives as StatusObserveFailed.
	StatusFragmentInvalid = "FRAGMENT_INVALID"
)

// Outcome is one kind's result.
type Outcome struct {
	Kind string `json:"kind"`
	// ProviderID is what the declaration selected. Empty when the binding did
	// not resolve.
	ProviderID string `json:"providerId,omitempty"`
	Status     string `json:"status"`
	// Resolved reports that observation.Binding returned an id.
	Resolved bool `json:"resolved"`
	// Selected reports that the Host registry held a provider for that id.
	Selected bool `json:"selected"`
	// Invoked reports that the selected provider's Observe was entered. It does
	// not establish that the fragment reflects external state.
	Invoked bool `json:"invoked"`
	// FragmentFields counts the non-empty facets in the returned fragment, so a
	// reader can tell "entered and answered" from "entered and answered with
	// nothing".
	FragmentFields int    `json:"fragmentFields,omitempty"`
	Detail         string `json:"detail,omitempty"`
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
	// Observed counts outcomes with StatusObserved. Refused is strictly
	// before entry: nothing was called. Unknown is a subset of Invoked: the
	// provider was entered and then failed, so whether the world was read
	// cannot be established.
	Observed int `json:"observedCount"`
	Refused  int `json:"refusedCount"`
	Unknown  int `json:"unknownCount"`
	Invoked  int `json:"invokedCount"`
	// Attempted is how many kinds were tried. It makes Observed readable: 1 of
	// 1 and 1 of 2 both give Observed=1, and only Attempted tells them apart.
	Attempted int `json:"attemptedCount"`
	// Complete reports that every attempted kind was observed. A partial run
	// keeps its refusals and is never presented as complete.
	Complete bool     `json:"complete"`
	Notes    []string `json:"notes"`
}

// Input is what the Host supplies. The reference builds no registry of its own.
type Input struct {
	Environment environment.Environment
	// Providers is the Host's observation-provider registry, keyed by the id the
	// declaration names. The reference never constructs a provider.
	Providers map[core.CanonicalID]observation.Provider
	// Kinds is the set of observation kinds to attempt. Empty means every kind
	// the Environment declares.
	Kinds []deployment.ObservationKind
}

// Run assembles the existing Router and observes each kind. Refusals are values,
// never panics.
func Run(ctx context.Context, in Input) Report {
	report := Report{
		Schema: Schema, Authority: Authority,
		RuleID: RuleID, RuleVersion: RuleVersion,
		RuleFingerprint: fingerprint(),
		Environment:     string(in.Environment.ID),
		Outcomes:        []Outcome{},
		Notes:           notes(),
	}

	kinds := in.Kinds
	if len(kinds) == 0 {
		seen := map[deployment.ObservationKind]bool{}
		for _, b := range in.Environment.ObservationProviders {
			if !seen[b.Kind] {
				seen[b.Kind] = true
				kinds = append(kinds, b.Kind)
			}
		}
	}

	for _, kind := range kinds {
		outcome := Outcome{Kind: string(kind)}

		// 1. Resolve from the declaration, using the existing function.
		id, resolveErr := observation.Binding(in.Environment, kind)
		if resolveErr != nil {
			outcome.Status = StatusBindingMissing
			outcome.Detail = resolveErr.Error()
			report.Refused++
			report.Outcomes = append(report.Outcomes, outcome)
			continue
		}
		outcome.Resolved = true
		outcome.ProviderID = string(id)

		// 2. Report whether the Host registered it, before Observe decides.
		provider, registered := in.Providers[id]
		if !registered {
			outcome.Status = StatusNotRegistered
			outcome.Detail = "the declaration names a provider the Host did not register"
			report.Refused++
			report.Outcomes = append(report.Outcomes, outcome)
			continue
		}
		outcome.Selected = true

		// 3. Enter the provider. Bind a witness so "entered" is observable
		// rather than inferred from a non-empty fragment.
		witness := &observeWitness{Provider: provider}
		witnessRouter := observation.Router{
			Providers: map[core.CanonicalID]observation.Provider{id: witness},
		}
		fragment, observeErr := witnessRouter.Observe(ctx, in.Environment, kind)
		outcome.Invoked = witness.invoked
		if witness.invoked {
			report.Invoked++
		}
		if observeErr != nil {
			outcome.Detail = observeErr.Error()
			if witness.invoked {
				// The provider was entered and then failed. Whether it read the
				// world is now unknown, so this is not a refusal.
				outcome.Status = StatusObserveFailed
				report.Unknown++
			} else {
				// The router stopped before the provider was entered.
				outcome.Status = StatusNotConforming
				report.Refused++
			}
			report.Outcomes = append(report.Outcomes, outcome)
			continue
		}
		if !witness.invoked {
			// The Router answered without entering the provider: the reference
			// must not present that as an observation.
			outcome.Status = StatusFragmentInvalid
			outcome.Detail = "the router answered without entering the provider"
			report.Refused++
			report.Outcomes = append(report.Outcomes, outcome)
			continue
		}
		// The existing Router already rejects a fragment whose identity does not
		// match the binding, so this reference does not repeat that check.
		outcome.FragmentFields = fragmentFields(fragment)
		outcome.Status = StatusObserved
		report.Observed++
		report.Outcomes = append(report.Outcomes, outcome)
	}
	// A run is complete only when at least one kind was attempted and every
	// attempt produced an observation. Partial runs stay partial.
	report.Attempted = len(report.Outcomes)
	report.Complete = report.Attempted > 0 && report.Observed == report.Attempted
	return report
}

// observeWitness observes the call boundary only. It grants nothing and
// inspects no external state.
type observeWitness struct {
	observation.Provider
	invoked bool
}

func (w *observeWitness) Observe(ctx context.Context, req observation.Request) (observation.Fragment, error) {
	w.invoked = true
	return w.Provider.Observe(ctx, req)
}

func fragmentFields(f observation.Fragment) int {
	n := 0
	if f.ObservedAt != "" {
		n++
	}
	if len(f.ApplicationRuntime) > 0 {
		n++
	}
	if len(f.Databases) > 0 {
		n++
	}
	return n
}

func notes() []string {
	return []string{
		"resolution uses observation.Binding, the same function cmd/tei already calls",
		"the router is the existing observation.Router; no second registry is added",
		"invoked means the provider's Observe was entered, not that external state was read",
		"a non-empty fragment is not proof that the observation succeeded in the world",
		"unknownCount counts entries that failed after the provider was called; it is a subset of invokedCount",
		"complete means every attempted kind was observed; it is not proof that the world matches",
		"no authentication, authority, permission or durable receipt is issued here",
	}
}

func fingerprint() string { return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(rulePolicy))) }

// Marshal renders the report as stable, indented JSON.
func (r Report) Marshal() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
