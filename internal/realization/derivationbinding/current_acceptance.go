package derivationbinding

import (
	"context"
	"strings"

	"tei/internal/modelref"
)

// CurrentAcceptedBindings is a read-only Host-owned projection of a CURRENT
// Rule/Binding/Choice acceptance decision. It is not a new decision or policy
// authority. A serialized request or an old receipt cannot implement this
// witness. Actual decision resolution, revocation and provenance belong to
// the existing #1121/#267 Acceptance owner.
type CurrentAcceptedBindings struct {
	ApplicationID     string
	ModelRevision     string
	DecisionRef       string
	AuthorityRevision string
	Snapshot          AcceptedBindingSnapshot
}

// CurrentBindingReader is a narrow consumer port, not an Acceptance store.
// Its trusted Host implementation MUST read current acceptance/revocation
// each time; no implementation of that Host-owned reader is supplied here.
type CurrentBindingReader interface {
	ReadCurrentAcceptedBindings(context.Context, string) (CurrentAcceptedBindings, error)
}

// ResolveCurrentBindings supplies the D4 planner ONLY with the Host reader's
// current accepted snapshot. request must be the Host's already-validated
// model/target projection and expectedModelRevision must come from the same
// trusted model read. This function does not establish their authenticity,
// perform runtime authorization, or give atomicity across later execution.
//
// Unlike raw ResolveBindings, this boundary requires every requested target
// to resolve: revoking one Binding cannot silently return a partial plan.
func ResolveCurrentBindings(ctx context.Context, reader CurrentBindingReader, applicationID, expectedModelRevision string, request BindingRequest) BindingPlan {
	deny := func(code string) BindingPlan {
		return BindingPlan{Diagnostics: []BindingDiagnostic{{Code: code, At: "current-acceptance"}}}
	}
	if ctx == nil || reader == nil || strings.TrimSpace(applicationID) == "" ||
		strings.TrimSpace(expectedModelRevision) == "" {
		return deny("current-acceptance-reader-required")
	}
	// Never let callers smuggle self-declared acceptance into this path.
	if len(request.Accepted.Rules) != 0 || len(request.Accepted.Bindings) != 0 ||
		len(request.Accepted.Choices) != 0 || len(request.Accepted.Realizations) != 0 {
		return deny("caller-acceptance-not-allowed")
	}
	if ctx.Err() != nil {
		return deny("current-acceptance-unavailable")
	}
	current, err := reader.ReadCurrentAcceptedBindings(ctx, applicationID)
	// Do not disclose Host reader diagnostics, which may contain private
	// decision identifiers or other application data.
	if err != nil || ctx.Err() != nil {
		return deny("current-acceptance-unavailable")
	}
	if current.ApplicationID != applicationID ||
		current.ModelRevision != expectedModelRevision ||
		strings.TrimSpace(current.DecisionRef) == "" ||
		strings.TrimSpace(current.AuthorityRevision) == "" {
		return deny("current-acceptance-mismatch")
	}
	request.Accepted = current.Snapshot
	plan := ResolveBindings(request)
	if len(plan.Diagnostics) != 0 {
		return plan // ResolveBindings itself strips every partial Binding.
	}
	if len(request.Targets) == 0 {
		return deny("current-acceptance-incomplete")
	}
	expected := make(map[modelref.Key]bool, len(request.Targets))
	for _, target := range request.Targets {
		expected[modelref.Key{Scope: target.Ref.Scope, Subject: target.Ref.Subject}] = true
	}
	for _, bound := range plan.Bindings {
		if !expected[bound.Target] {
			return deny("current-acceptance-incomplete")
		}
		delete(expected, bound.Target)
	}
	if len(expected) != 0 {
		return deny("current-acceptance-incomplete")
	}
	if ctx.Err() != nil {
		return deny("current-acceptance-unavailable")
	}
	return plan
}
