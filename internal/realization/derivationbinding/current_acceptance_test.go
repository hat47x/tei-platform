package derivationbinding

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"tei/internal/semantic/derivationref"
)

type d4CurrentReader struct {
	record       CurrentAcceptedBindings
	err          error
	calls        int
	meaning      map[string]derivationref.Definition
	meaningErr   error
	meaningCalls int
}

func (r *d4CurrentReader) ReadCurrentAcceptedBindings(_ context.Context, _ string) (CurrentAcceptedBindings, error) {
	r.calls++
	return r.record, r.err
}

// This synthetic store only models the trusted Host comparison. The real
// Accepted meaning and its fingerprint remain with #1121/#267, never D4.
func (r *d4CurrentReader) VerifyCurrentAcceptedRuleMeaning(_ context.Context, _ CurrentAcceptedBindings, rule RuleContract, actual derivationref.Definition) (bool, error) {
	r.meaningCalls++
	if r.meaningErr != nil {
		return false, r.meaningErr
	}
	accepted, exists := r.meaning[rule.DefinitionID]
	return exists && reflect.DeepEqual(accepted, actual), nil
}

func TestD4CurrentHostAcceptanceGatesRulesBindingsAndChoices(t *testing.T) {
	const app, revision = "sales.app", "sha256:current-model"
	newRequest := func(t *testing.T) (BindingRequest, *d4CurrentReader) {
		t.Helper()
		request := d4Fixture(t)
		acceptedMeaning := request.Definitions[0]
		acceptedMeaning.Inputs = append([]string(nil), acceptedMeaning.Inputs...) // independent Host test store
		reader := &d4CurrentReader{record: CurrentAcceptedBindings{
			ApplicationID: app, ModelRevision: revision,
			DecisionRef: "decision.current", AuthorityRevision: "authority.2",
			Snapshot: request.Accepted,
		}, meaning: map[string]derivationref.Definition{acceptedMeaning.ID: acceptedMeaning}}
		request.Accepted = AcceptedBindingSnapshot{} // never client-supplied
		return request, reader
	}
	expectDenied := func(t *testing.T, plan BindingPlan, code string) {
		t.Helper()
		if !d4HasCode(plan, code) || len(plan.Bindings) != 0 {
			t.Fatalf("expected fail-closed %q without partial plan: %+v", code, plan)
		}
	}

	t.Run("current-host-acceptance-allowed", func(t *testing.T) {
		req, reader := newRequest(t)
		got := ResolveCurrentBindings(context.Background(), reader, app, revision, req)
		if len(got.Diagnostics) != 0 || len(got.Bindings) != 2 || reader.calls != 1 {
			t.Fatalf("current accepted Rule/Binding/Choice must resolve both targets: %+v calls=%d", got, reader.calls)
		}
	})

	t.Run("revoked-binding-no-partial-result", func(t *testing.T) {
		req, reader := newRequest(t)
		// The current Host decision no longer includes the Query-local
		// Binding. A stale caller request cannot restore that Binding.
		reader.record.Snapshot.Bindings = reader.record.Snapshot.Bindings[:1]
		expectDenied(t, ResolveCurrentBindings(context.Background(), reader, app, revision, req), "current-acceptance-incomplete")
	})

	t.Run("ambiguous-after-choice-revoked", func(t *testing.T) {
		req := d4Fixture(t)
		d4SecondRule(&req)
		reader := &d4CurrentReader{record: CurrentAcceptedBindings{
			ApplicationID: app, ModelRevision: revision, DecisionRef: "decision.current",
			AuthorityRevision: "authority.3", Snapshot: req.Accepted,
		}, meaning: map[string]derivationref.Definition{
			req.Definitions[0].ID: req.Definitions[0],
			req.Definitions[1].ID: req.Definitions[1],
		}}
		req.Accepted = AcceptedBindingSnapshot{}
		// Two accepted Rules reach one target, but the current Host
		// decision does NOT include a Choice. No order-based fallback.
		expectDenied(t, ResolveCurrentBindings(context.Background(), reader, app, revision, req), "ambiguous-target")
		reader.record.Snapshot.Choices = []AcceptedChoice{{
			Target:    reader.record.Snapshot.Bindings[0].Target,
			BindingID: "sales.alternative_attribute",
		}}
		got := ResolveCurrentBindings(context.Background(), reader, app, revision, req)
		if len(got.Diagnostics) != 0 || len(got.Bindings) != 2 {
			t.Fatalf("current accepted Choice should resolve ambiguity: %+v", got)
		}
	})

	t.Run("wrong-model-revision", func(t *testing.T) {
		req, reader := newRequest(t)
		reader.record.ModelRevision = "sha256:stale-model"
		expectDenied(t, ResolveCurrentBindings(context.Background(), reader, app, revision, req), "current-acceptance-mismatch")
	})

	t.Run("wrong-application-or-missing-decision", func(t *testing.T) {
		req, reader := newRequest(t)
		reader.record.ApplicationID = "other.app"
		expectDenied(t, ResolveCurrentBindings(context.Background(), reader, app, revision, req), "current-acceptance-mismatch")
		reader.record.ApplicationID = app
		reader.record.DecisionRef = ""
		expectDenied(t, ResolveCurrentBindings(context.Background(), reader, app, revision, req), "current-acceptance-mismatch")
	})

	t.Run("untrusted-caller-cannot-self-approve", func(t *testing.T) {
		req, reader := newRequest(t)
		// The caller injects an otherwise valid but unauthenticated
		// snapshot; it must not be treated as the Host's decision.
		req.Accepted = reader.record.Snapshot
		expectDenied(t, ResolveCurrentBindings(context.Background(), reader, app, revision, req), "caller-acceptance-not-allowed")
		if reader.calls != 0 {
			t.Fatalf("caller claims must be rejected before invoking Host reader; calls=%d", reader.calls)
		}
	})

	t.Run("reader-failure-does-not-leak-secret", func(t *testing.T) {
		req, reader := newRequest(t)
		reader.err = errors.New("secret accepted Rule: private-cost")
		got := ResolveCurrentBindings(context.Background(), reader, app, revision, req)
		expectDenied(t, got, "current-acceptance-unavailable")
		for _, diag := range got.Diagnostics {
			if diag.Code == reader.err.Error() || diag.At == reader.err.Error() {
				t.Fatalf("private Host error leaked: %+v", got)
			}
		}
	})

	t.Run("same-rule-id-version-changed-operator-denied", func(t *testing.T) {
		req, reader := newRequest(t)
		req.Definitions[0].Operator = "add"
		got := ResolveCurrentBindings(context.Background(), reader, app, revision, req)
		expectDenied(t, got, "rule-meaning-change-reaccept")
		if reader.meaningCalls != 1 {
			t.Fatalf("Host must compare accepted meaning before planning: calls=%d", reader.meaningCalls)
		}
	})

	t.Run("same-rule-id-version-changed-input-denied", func(t *testing.T) {
		req, reader := newRequest(t)
		req.Definitions[0].Inputs[1] = "sales.order_line.discount_price"
		got := ResolveCurrentBindings(context.Background(), reader, app, revision, req)
		expectDenied(t, got, "rule-meaning-change-reaccept")
	})

	t.Run("unchanged-meaning-new-realization-allowed", func(t *testing.T) {
		req, reader := newRequest(t)
		reader.record.Snapshot.Realizations[0].PluginID = "example.wasm"
		reader.record.Snapshot.Realizations[0].Implementation = "wasm:line-amount"
		got := ResolveCurrentBindings(context.Background(), reader, app, revision, req)
		if len(got.Diagnostics) != 0 || len(got.Bindings) != 2 || reader.meaningCalls != 1 {
			t.Fatalf("same canonical Rule must allow new locked realization for both targets: %+v calls=%d", got, reader.meaningCalls)
		}
		for _, b := range got.Bindings {
			if b.PluginID != "example.wasm" {
				t.Fatalf("unexpected implementation selected: %+v", b)
			}
		}
	})

	t.Run("meaning-reader-error-not-disclosed", func(t *testing.T) {
		req, reader := newRequest(t)
		reader.meaningErr = errors.New("private accepted meaning fingerprint")
		got := ResolveCurrentBindings(context.Background(), reader, app, revision, req)
		expectDenied(t, got, "current-acceptance-unavailable")
		for _, diag := range got.Diagnostics {
			if diag.Code == reader.meaningErr.Error() || diag.At == reader.meaningErr.Error() {
				t.Fatalf("Host meaning details leaked: %+v", got)
			}
		}
	})

	t.Run("no-reader", func(t *testing.T) {
		req, _ := newRequest(t)
		expectDenied(t, ResolveCurrentBindings(context.Background(), nil, app, revision, req), "current-acceptance-reader-required")
	})
}
