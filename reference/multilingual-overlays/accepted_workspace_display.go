package multilingualoverlays

// O2c1 connects the existing Source Acceptance verifier to the O2b
// Workspace display adapter. It does NOT implement Host authorization,
// an Acceptance Reader, or acceptance of localization Grants/Packs.

import (
	"context"
	"fmt"

	"tei/internal/plugin"
	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// AcceptedDisplayReview carries both the *current* Host Source witness and
// exact display/translation candidates to the existing Host policy gate.
// The Source decision alone cannot authorize a Pack, Grant, or Override.
type AcceptedDisplayReview struct {
	Acceptance sourceacceptance.Record
	Display    ReviewRequest
}

// AcceptedDisplayGate is a Host-supplied policy gate. It MUST verify that the
// selected translation definitions and target environment are authorized.
// An always-allow function is valid only as a synthetic test stub.
type AcceptedDisplayGate func(AcceptedDisplayReview) error

// BuildAcceptedWorkspaceDisplay loads the real Workspace exactly once, builds
// a typed Source Index from the *same parsed bytes*, and checks that index
// against the Host's currently accepted Source via sourceacceptance.VerifyCurrent.
// Only then can the caller's separate Host display-policy gate evaluate the
// Surface/Packs/Grants/Overrides, before normal Plugin Lock + E1 selection.
//
// Reader and gate MUST be wired to a trusted Host: accepting a caller-supplied
// fake Reader/always-allow gate does not prove source acceptance, rights,
// revocation or authentic delivery. This read-only preflight cannot make the
// multi-file Workspace Load atomic (#1017) or prevent revocation after return;
// Host must revalidate at actual display admission/dispatch.
func BuildAcceptedWorkspaceDisplay(
	ctx context.Context,
	reader sourceacceptance.Reader,
	applicationID string,
	root string,
	lock plugin.Lock,
	cfg WorkspaceDisplayConfig,
	gate AcceptedDisplayGate,
) (DisplayManifest, SourceWitness, error) {
	reject := func(message string) (DisplayManifest, SourceWitness, error) {
		return DisplayManifest{}, SourceWitness{}, fmt.Errorf("accepted-workspace-display: %s", message)
	}
	if ctx == nil {
		return reject("context-required")
	}
	if reader == nil {
		return reject("host-reader-required")
	}
	if gate == nil {
		return reject("host-display-gate-required")
	}
	if err := ctx.Err(); err != nil {
		return reject("context-cancelled")
	}
	if cfg.Scope == "" || cfg.Data == "" || cfg.SourceLocale == "" || cfg.Locale == "" {
		return reject("incomplete-binding")
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return reject("workspace-load-failed")
	}
	var accepted sourceacceptance.Record
	verify := func(index sourceindex.Index) error {
		record, err := sourceacceptance.VerifyCurrent(ctx, reader, applicationID, index)
		if err != nil {
			// Do not forward arbitrary Host Reader errors to UI consumers.
			return fmt.Errorf("current-acceptance-verification-failed")
		}
		accepted = record
		return nil
	}
	review := func(request ReviewRequest) error {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context-cancelled")
		}
		if accepted.DecisionRef == "" || accepted.ModelRevision == "" {
			return fmt.Errorf("current-acceptance-missing")
		}
		return gate(AcceptedDisplayReview{Acceptance: accepted, Display: request})
	}
	manifest, witness, err := buildLoadedWorkspaceDisplay(root, loaded, lock, cfg, review, verify)
	if err != nil {
		return DisplayManifest{}, SourceWitness{}, err
	}
	if err := ctx.Err(); err != nil {
		return reject("context-cancelled")
	}
	return manifest, witness, nil
}
