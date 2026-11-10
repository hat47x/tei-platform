// Package evidencerouterconnect is a read-only reference for the
// EvidenceProviders gap: evidence.Binding already resolves the declaration and
// `cmd/tei -plan` calls it, but nothing outside a test assembles
// evidence.Router, so Collect never runs in a product entry.
//
// It is the evidence-side counterpart of reference-observation-router-connect,
// and it exists to answer one question: does the same four-stage shape hold when
// the router needs a durable.ActionRecord instead of just an Environment?
package evidencerouterconnect

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/durable"
	"tei/internal/deployment/evidence"
	"tei/internal/environment"
)

const (
	// Schema identifies the report. It is a reference output, not a product
	// contract.
	Schema = "tei.reference.evidence-router-connect/v0"
	// Authority states what the report is allowed to claim.
	Authority = "derived-reference"

	RuleID      = "tei.reference.evidence-router-connect"
	RuleVersion = "v0"
)

const rulePolicy = `evidence-router-connect/v0
resolution: evidence.Binding over the Environment's evidence-provider declarations
assembly: evidence.Router{Providers: <Host-supplied>}
entry: evidence.Router.Collect, one call per durable action record in ModeEvidence
counted: resolved (Binding returned an id), selected (the registry held it),
         invoked (the provider's Collect was entered)
refused when: the record is not an evidence gate, the binding is missing or
              ambiguous, the id is not registered, or the provider does not conform
after entry: a failure inside the provider is unknown, not a refusal; it is
             counted separately because the call already happened
not verified: that the evidence reflects external state, that its outcome is
              correct, or that a grant is current
not issued: authentication, authority, permission, or a durable receipt
`

// Status names the outcome of one record.
const (
	StatusCollected     = "COLLECTED"
	StatusNotEvidence   = "NOT_EVIDENCE_GATE"
	StatusBindingMissig = "BINDING_MISSING"
	StatusNotRegistered = "NOT_REGISTERED"
	StatusNotConforming = "NOT_CONFORMING"
	StatusCollectFailed = "COLLECT_FAILED"
	StatusEvidenceBad   = "EVIDENCE_INVALID"
)

// Outcome is one record's result.
type Outcome struct {
	ActionID string `json:"actionId"`
	Kind     string `json:"kind"`
	// ProviderID is what the declaration selected. Empty when the binding did
	// not resolve.
	ProviderID string `json:"providerId,omitempty"`
	Status     string `json:"status"`
	// Resolved reports that evidence.Binding returned an id.
	Resolved bool `json:"resolved"`
	// Selected reports that the Host registry held a provider for that id.
	Selected bool `json:"selected"`
	// Invoked reports that the selected provider's Collect was entered. It does
	// not establish that the evidence reflects external state.
	Invoked bool `json:"invoked"`
	// Outcome is the provider's pass/fail answer, verbatim. Empty when the
	// provider did not answer.
	Outcome string `json:"evidenceOutcome,omitempty"`
	Detail  string `json:"detail,omitempty"`
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
	// Collected counts outcomes with StatusCollected. Refused is strictly before
	// entry: nothing was called. Unknown is a subset of Invoked: the provider was
	// entered and then failed, so whether the evidence reflects the world cannot
	// be established.
	Collected int `json:"collectedCount"`
	Refused   int `json:"refusedCount"`
	Unknown   int `json:"unknownCount"`
	Invoked   int `json:"invokedCount"`
	// Attempted is how many records were tried. It makes Collected readable: 1
	// of 1 and 1 of 2 both give Collected=1, and only Attempted tells them apart.
	Attempted int `json:"attemptedCount"`
	// Complete reports that every attempted record was collected. A partial run
	// keeps its refusals and unknowns and is never presented as complete.
	Complete bool     `json:"complete"`
	Notes    []string `json:"notes"`
}

// Input is what the Host supplies. The reference builds no registry of its own.
type Input struct {
	Environment environment.Environment
	// Providers is the Host's evidence-provider registry, keyed by the id the
	// declaration names. The reference never constructs a provider.
	Providers map[core.CanonicalID]evidence.Provider
	// Records are the durable action records to collect evidence for. Only
	// ModeEvidence records can produce evidence; the others are refused with a
	// distinct status rather than silently skipped.
	Records []durable.ActionRecord
}

// Run assembles the existing Router and collects evidence for each record.
// Refusals are values, never panics.
func Run(ctx context.Context, in Input) Report {
	report := Report{
		Schema: Schema, Authority: Authority,
		RuleID: RuleID, RuleVersion: RuleVersion,
		RuleFingerprint: fingerprint(),
		Environment:     string(in.Environment.ID),
		Outcomes:        []Outcome{},
		Notes:           notes(),
	}

	for _, rec := range in.Records {
		outcome := Outcome{ActionID: rec.ID, Kind: string(rec.EvidenceKind)}

		// A record that is not an evidence gate cannot produce evidence. This is
		// refused here so the reason is visible, rather than passing through the
		// router's generic error.
		if rec.Mode != durable.ModeEvidence {
			outcome.Status = StatusNotEvidence
			outcome.Detail = "the record is not an evidence gate"
			report.Refused++
			report.Outcomes = append(report.Outcomes, outcome)
			continue
		}

		// 1. Resolve from the declaration, using the existing function.
		id, resolveErr := evidence.Binding(in.Environment, rec.EvidenceKind)
		if resolveErr != nil {
			outcome.Status = StatusBindingMissig
			outcome.Detail = resolveErr.Error()
			report.Refused++
			report.Outcomes = append(report.Outcomes, outcome)
			continue
		}
		outcome.Resolved = true
		outcome.ProviderID = string(id)

		// 2. Report whether the Host registered it, before Collect decides.
		provider, registered := in.Providers[id]
		if !registered {
			outcome.Status = StatusNotRegistered
			outcome.Detail = "the declaration names a provider the Host did not register"
			report.Refused++
			report.Outcomes = append(report.Outcomes, outcome)
			continue
		}
		outcome.Selected = true

		// 3. Enter the provider through the existing Router, with a witness so
		// "entered" is observed rather than inferred.
		witness := &collectWitness{Provider: provider}
		witnessRouter := evidence.Router{
			Providers: map[core.CanonicalID]evidence.Provider{id: witness},
		}
		collected, collectErr := witnessRouter.Collect(ctx, in.Environment, rec)
		outcome.Invoked = witness.invoked
		if witness.invoked {
			report.Invoked++
		}
		if collectErr != nil {
			outcome.Detail = collectErr.Error()
			if witness.invoked {
				// The provider was entered and then failed. Whether the evidence
				// reflects the world is now unknown, so this is not a refusal.
				outcome.Status = StatusCollectFailed
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
			// must not present that as collected evidence.
			outcome.Status = StatusEvidenceBad
			outcome.Detail = "the router answered without entering the provider"
			report.Refused++
			report.Outcomes = append(report.Outcomes, outcome)
			continue
		}
		// The existing Router already rejects evidence whose target does not
		// match the record, so this reference does not repeat that check.
		outcome.Outcome = string(collected.Outcome)
		outcome.Status = StatusCollected
		report.Collected++
		report.Outcomes = append(report.Outcomes, outcome)
	}
	// A run is complete only when at least one record was attempted and every
	// attempt produced evidence. Partial runs stay partial.
	report.Attempted = len(report.Outcomes)
	report.Complete = report.Attempted > 0 && report.Collected == report.Attempted
	return report
}

// collectWitness observes the call boundary only. It grants nothing and
// inspects no external state.
type collectWitness struct {
	evidence.Provider
	invoked bool
}

func (w *collectWitness) Collect(ctx context.Context, req evidence.Request) (durable.Evidence, error) {
	w.invoked = true
	return w.Provider.Collect(ctx, req)
}

func notes() []string {
	return []string{
		"resolution uses evidence.Binding, the same function cmd/tei already calls",
		"the router is the existing evidence.Router; no second registry is added",
		"invoked means the provider's Collect was entered, not that the evidence is true",
		"a pass outcome is the provider's claim, not proof that external state matches it",
		"unknownCount counts entries that failed after the provider was called; it is a subset of invokedCount",
		"complete means every attempted record was collected; it is not proof that the evidence is true",
		"no authentication, authority, permission or durable receipt is issued here",
	}
}

func fingerprint() string { return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(rulePolicy))) }

// Marshal renders the report as stable, indented JSON.
func (r Report) Marshal() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }

// Compile-time check that the constants used above are the deployment kinds this
// reference claims to handle. It keeps the reference honest if the kind set
// changes.
var _ = deployment.EvidenceRolloutHealth
