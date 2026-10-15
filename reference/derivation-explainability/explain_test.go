package derivationexplainability

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"tei/internal/semantic/derivationref"
)

type fakeAuthority struct {
	allowed map[string]bool
	errs    map[string]error
}

func newAuthority() *fakeAuthority {
	return &fakeAuthority{allowed: map[string]bool{}, errs: map[string]error{}}
}

func authKey(subject string, field Field, element string) string {
	return subject + "\x00" + string(field) + "\x00" + element
}

func (a *fakeAuthority) grant(subject string, field Field, element string) {
	a.allowed[authKey(subject, field, element)] = true
}

func (a *fakeAuthority) fail(subject string, field Field, element string, err error) {
	a.errs[authKey(subject, field, element)] = err
}

func (a *fakeAuthority) CanDisclose(_ context.Context, subject string, field Field, element string) error {
	key := authKey(subject, field, element)
	if err, ok := a.errs[key]; ok {
		return err
	}
	if a.allowed[key] {
		return nil
	}
	return errors.New("denied")
}

const (
	admin   = "staff.admin"
	public  = "staff.public"
	fixedAt = "2026-10-16T00:00:00Z"
)

func amountEvidence() Evidence {
	return Evidence{
		Rule:        RuleOrigin{RuleID: "sales.line_amount", Revision: "r1", SourcePath: "src/xf/line.go"},
		TargetID:    "sales.order_line.amount",
		Realization: Realization{Identity: "go/reference", Version: "p1", VersionKnown: true},
		Inputs: []InputEvidence{
			{InputID: "sales.order_line.cost", Version: "c1", VersionKnown: true, State: derivationref.Present, Value: "300"},
			{InputID: "sales.order_line.quantity", Version: "q1", VersionKnown: true, State: derivationref.Present, Value: "2"},
		},
		Context: EvaluationContext{Class: "online-request"},
		Result:  derivationref.Input{State: derivationref.Present, Decimal: "600"},
		Outcome: OutcomeValue,
	}
}

func grantAdmin(a *fakeAuthority, subject string, ev Evidence) {
	a.grant(subject, FieldResult, ev.TargetID)
	a.grant(subject, FieldRule, ev.Rule.RuleID)
	a.grant(subject, FieldRuleSource, ev.Rule.RuleID)
	a.grant(subject, FieldTarget, ev.TargetID)
	a.grant(subject, FieldRealization, ev.Realization.Identity)
	a.grant(subject, FieldContext, ev.TargetID)
	a.grant(subject, FieldReason, ev.TargetID)
	a.grant(subject, FieldConfirmed, ev.TargetID)
	for _, in := range ev.Inputs {
		a.grant(subject, FieldInputIdentity, in.InputID)
		a.grant(subject, FieldInputValue, in.InputID)
	}
}

func mustProject(t *testing.T, subject string, authority DisclosureAuthority, ex Explanation) View {
	t.Helper()
	view, err := Project(context.Background(), subject, authority, ex)
	if err != nil {
		t.Fatalf("unexpected projection error: %v", err)
	}
	return view
}

func marshal(t *testing.T, view View) string {
	t.Helper()
	payload, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(payload)
}

func TestD7IdentityTracksMeaningNotLocatorOrValue(t *testing.T) {
	ev := amountEvidence()
	base, err := ev.Identity()
	if err != nil {
		t.Fatal(err)
	}

	// Moving the source file or recomputing a different value does not change
	// what the explanation is about.
	same := ev
	same.Rule.SourcePath = "elsewhere/renamed.go"
	same.Context.AsOf = time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	same.Result = derivationref.Input{State: derivationref.Present, Decimal: "700"}
	moved, err := same.Identity()
	if err != nil {
		t.Fatal(err)
	}
	if moved != base {
		t.Fatalf("locator, time or value changed identity: %s != %s", moved, base)
	}

	// A rule revision, an input version or a realization version does change
	// the meaning the explanation is about.
	for name, mutate := range map[string]func(*Evidence){
		"rule revision": func(e *Evidence) { e.Rule.Revision = "r2" },
		"input version": func(e *Evidence) { e.Inputs[0].Version = "c2" },
		"realization":   func(e *Evidence) { e.Realization.Version = "p2" },
		"context class": func(e *Evidence) { e.Context.Class = "batch-run" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := ev
			changed.Inputs = append([]InputEvidence(nil), ev.Inputs...)
			mutate(&changed)
			got, err := changed.Identity()
			if err != nil {
				t.Fatal(err)
			}
			if got == base {
				t.Fatalf("%s did not change identity", name)
			}
		})
	}
}

func TestD7ClassifiesOutcomesWithoutCollapsing(t *testing.T) {
	present := derivationref.Input{State: derivationref.Present, Decimal: "600"}
	null := derivationref.Input{State: derivationref.Null}
	missing, missingErr := derivationref.Multiply(derivationref.Input{State: derivationref.Missing}, present)
	_, operatorErr := derivationref.Multiply(derivationref.Input{State: derivationref.Present, Decimal: "abc"}, present)

	cases := []struct {
		name    string
		value   derivationref.Input
		err     error
		outcome Outcome
		reason  Reason
	}{
		{"value", present, nil, OutcomeValue, ReasonNone},
		{"null", null, nil, OutcomeNull, ReasonNone},
		{"missing", missing, missingErr, OutcomeNoValue, ReasonInputMissing},
		{"not evaluable", derivationref.Input{}, operatorErr, OutcomeNoValue, ReasonNotEvaluable},
	}
	seen := map[Reason]bool{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outcome, reason := Classify(tc.value, tc.err)
			if outcome != tc.outcome || reason != tc.reason {
				t.Fatalf("got (%s, %s), want (%s, %s)", outcome, reason, tc.outcome, tc.reason)
			}
			seen[reason] = true
		})
	}
	if len(seen) != 3 {
		t.Fatalf("distinct causes were collapsed: %v", seen)
	}
}

func TestD7RelationReasonsComeFromStateNotErrorText(t *testing.T) {
	for state, want := range map[derivationref.InputState]Reason{
		derivationref.Missing:           ReasonInputMissing,
		derivationref.AbsentRelation:    ReasonAbsentRelation,
		derivationref.MultipleRelations: ReasonMultipleRelations,
	} {
		if got := ReasonForState(state); got != want {
			t.Fatalf("state %s: got %s, want %s", state, got, want)
		}
	}
}

func TestD7GeneralViewerDoesNotLeakHiddenInput(t *testing.T) {
	ev := amountEvidence()

	adminAuthority := newAuthority()
	grantAdmin(adminAuthority, admin, ev)
	adminView := mustProject(t, admin, adminAuthority, Explanation{Evidence: ev})
	if len(adminView.Inputs) != 2 || adminView.Rule.SourcePath == "" {
		t.Fatalf("admin lost evidence: %+v", adminView)
	}

	// The general viewer may see the amount target but has no read right for
	// the private cost. Neither the cost value nor even its existence may
	// appear in the projection.
	general := newAuthority()
	general.grant(public, FieldResult, ev.TargetID)
	general.grant(public, FieldTarget, ev.TargetID)
	general.grant(public, FieldRule, ev.Rule.RuleID)
	general.grant(public, FieldRealization, ev.Realization.Identity)
	general.grant(public, FieldInputIdentity, "sales.order_line.quantity")
	general.grant(public, FieldInputValue, "sales.order_line.quantity")

	view := mustProject(t, public, general, Explanation{Evidence: ev})
	if view.Result != "600" {
		t.Fatalf("general lost the disclosed amount: %q", view.Result)
	}
	if len(view.Inputs) != 1 || view.Inputs[0].InputID != "sales.order_line.quantity" || view.Inputs[0].Value != "2" {
		t.Fatalf("general input projection leaked or hid the wrong input: %+v", view.Inputs)
	}
	if view.Rule.SourcePath != "" {
		t.Fatalf("source path leaked without FieldRuleSource: %+v", view.Rule)
	}
	payload := marshal(t, view)
	for _, secret := range []string{"cost", "300", "line.go"} {
		if strings.Contains(payload, secret) {
			t.Fatalf("general projection leaked %q: %s", secret, payload)
		}
	}
}

func TestD7WithheldInputExistenceIsNotAsserted(t *testing.T) {
	ev := amountEvidence()
	authority := newAuthority()
	grantAdmin(authority, admin, ev)
	// Deny every disclosure about the private cost input.
	for _, field := range []Field{FieldInputIdentity, FieldInputValue} {
		authority.fail(admin, field, "sales.order_line.cost", errors.New("no"))
	}
	view := mustProject(t, admin, authority, Explanation{Evidence: ev})
	if len(view.Inputs) != 1 || view.Inputs[0].InputID == "sales.order_line.cost" {
		t.Fatalf("withheld input still present: %+v", view.Inputs)
	}
	if strings.Contains(marshal(t, view), "cost") {
		t.Fatal("withheld input identity leaked")
	}
}

func TestD7ConfirmedFactIsSeparateFromCurrentEvaluation(t *testing.T) {
	current := amountEvidence()
	current.Rule.Revision = "r2"
	current.Inputs[0].Value = "350"
	current.Result = derivationref.Input{State: derivationref.Present, Decimal: "700"}

	confirmedAt, err := time.Parse(time.RFC3339, fixedAt)
	if err != nil {
		t.Fatal(err)
	}
	confirmed := &derivationref.ConfirmedFact{
		FactID:             "fact.order_1",
		TargetID:           "sales.order_line.amount",
		Value:              derivationref.Input{State: derivationref.Present, Decimal: "600"},
		SourceRuleID:       "sales.line_amount",
		SourceRuleRevision: "r1",
		ConfirmedAt:        confirmedAt,
		CommandID:          "cmd.1",
		PersistedRevision:  "rev.1",
	}

	authority := newAuthority()
	grantAdmin(authority, admin, current)
	authority.grant(admin, FieldConfirmed, confirmed.TargetID)

	view := mustProject(t, admin, authority, Explanation{Evidence: current, Confirmed: confirmed})
	if view.Result != "700" || view.Confirmed == nil || view.Confirmed.Value != "600" {
		t.Fatalf("current evaluation overwrote the confirmed fact: %+v", view)
	}
	if view.Confirmed.Revision != "r1" || current.Rule.Revision != "r2" {
		t.Fatalf("confirmed and current revisions were conflated: %+v", view.Confirmed)
	}

	// A viewer without FieldConfirmed must not see the persisted fact at all.
	general := newAuthority()
	grantAdmin(general, public, current)
	general.fail(public, FieldConfirmed, current.TargetID, errors.New("no"))
	restricted := mustProject(t, public, general, Explanation{Evidence: current, Confirmed: confirmed})
	if restricted.Confirmed != nil || strings.Contains(marshal(t, restricted), "fact.order_1") {
		t.Fatal("confirmed fact leaked without FieldConfirmed")
	}
}

func TestD7UnknownVersionIsNotFabricated(t *testing.T) {
	ev := amountEvidence()
	ev.Realization.VersionKnown = false
	ev.Realization.Version = ""
	ev.Inputs[0].VersionKnown = false
	ev.Inputs[0].Version = ""
	if err := Validate(ev); err != nil {
		t.Fatalf("unknown versions must be valid evidence: %v", err)
	}
	authority := newAuthority()
	grantAdmin(authority, admin, ev)
	view := mustProject(t, admin, authority, Explanation{Evidence: ev})
	if view.Realization.VersionKnown || view.Inputs[0].VersionKnown {
		t.Fatalf("unknown version was marked known: %+v", view)
	}
	payload := marshal(t, view)
	if strings.Contains(payload, "\"version\":\"0\"") || strings.Contains(payload, "\"version\":\"\"") {
		t.Fatalf("unknown version was fabricated: %s", payload)
	}
}

func TestD7ValidateRejectsPathAsSemanticIdentity(t *testing.T) {
	cases := map[string]func(*Evidence){
		"rule path":     func(e *Evidence) { e.Rule.RuleID = "src/xf/XF100.java" },
		"target path":   func(e *Evidence) { e.TargetID = "src/target.go" },
		"input path":    func(e *Evidence) { e.Inputs[0].InputID = "data/input.go" },
		"absolute path": func(e *Evidence) { e.Rule.RuleID = "/abs/rule" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			ev := amountEvidence()
			mutate(&ev)
			if err := Validate(ev); !errors.Is(err, ErrInvalidEvidence) {
				t.Fatalf("path accepted as semantic identity: %v", err)
			}
		})
	}

	// A source locator is allowed only in the dedicated SourcePath field.
	ev := amountEvidence()
	ev.Rule.SourcePath = "src/xf/XF100.java"
	if err := Validate(ev); err != nil {
		t.Fatalf("source path must be allowed in SourcePath: %v", err)
	}
}

func TestD7ValidateRequiresCanonicalInputOrder(t *testing.T) {
	ev := amountEvidence()
	ev.Inputs[0], ev.Inputs[1] = ev.Inputs[1], ev.Inputs[0]
	if err := Validate(ev); !errors.Is(err, ErrInvalidEvidence) {
		t.Fatalf("non-canonical order accepted: %v", err)
	}
}

func TestD7NoValueOutcomeCarriesTheFineReason(t *testing.T) {
	for state, reason := range map[derivationref.InputState]Reason{
		derivationref.Missing:           ReasonInputMissing,
		derivationref.AbsentRelation:    ReasonAbsentRelation,
		derivationref.MultipleRelations: ReasonMultipleRelations,
	} {
		t.Run(string(state), func(t *testing.T) {
			ev := amountEvidence()
			ev.Result = derivationref.Input{}
			ev.Outcome = OutcomeNoValue
			ev.Reason = ReasonForState(state)
			if err := Validate(ev); err != nil {
				t.Fatal(err)
			}
			authority := newAuthority()
			grantAdmin(authority, admin, ev)
			view := mustProject(t, admin, authority, Explanation{Evidence: ev})
			if view.Outcome != OutcomeNoValue || view.Reason != reason || view.Result != "" {
				t.Fatalf("outcome not distinguished: %+v", view)
			}
		})
	}
}

func TestD7ReasonCanBeWithheldWhileOutcomeRemains(t *testing.T) {
	ev := amountEvidence()
	ev.Result = derivationref.Input{}
	ev.Outcome = OutcomeNoValue
	ev.Reason = ReasonMultipleRelations

	authority := newAuthority()
	grantAdmin(authority, admin, ev)
	authority.fail(admin, FieldReason, ev.TargetID, errors.New("no"))

	view := mustProject(t, admin, authority, Explanation{Evidence: ev})
	if view.Outcome != OutcomeNoValue || view.Reason != ReasonNone {
		t.Fatalf("fine reason was not withheld: %+v", view)
	}
}

func TestD7DeniedResultDisclosesNothingAndNoSecret(t *testing.T) {
	ev := amountEvidence()
	authority := newAuthority()
	grantAdmin(authority, public, ev)
	secret := errors.New("backend says cost=300 for private tenant")
	authority.fail(public, FieldResult, ev.TargetID, secret)

	view, err := Project(context.Background(), public, authority, Explanation{Evidence: ev})
	if !errors.Is(err, ErrNotDisclosable) {
		t.Fatalf("expected ErrNotDisclosable, got %v", err)
	}
	if !reflect.DeepEqual(view, View{}) {
		t.Fatalf("denied projection returned partial data: %+v", view)
	}
	for _, leak := range []string{"cost", "300", "private"} {
		if strings.Contains(err.Error(), leak) {
			t.Fatalf("denial leaked %q: %v", leak, err)
		}
	}
}

func TestD7NilAuthorityAndEmptySubjectFailClosed(t *testing.T) {
	ev := amountEvidence()
	if _, err := Project(context.Background(), admin, nil, Explanation{Evidence: ev}); !errors.Is(err, ErrNotDisclosable) {
		t.Fatalf("nil authority accepted: %v", err)
	}
	authority := newAuthority()
	grantAdmin(authority, admin, ev)
	if _, err := Project(context.Background(), "  ", authority, Explanation{Evidence: ev}); !errors.Is(err, ErrNotDisclosable) {
		t.Fatalf("blank subject accepted: %v", err)
	}
}
