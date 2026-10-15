package derivationref

import (
	"reflect"
	"testing"

	"tei/internal/modelref"
	"tei/internal/semantic"
	"tei/internal/subject"
)

const (
	d4RuleID   = "sales.line-amount"
	d4AmountID = "sales.order-line.amount"
	d4QueryID  = "query.order-list.line-total"
)

func d4Ref(scope, kind, id, version string) modelref.Ref {
	return modelref.Ref{Scope: scope, Subject: subject.Subject{Kind: kind, ID: id}, Version: version}
}

func d4Node(ref modelref.Ref, display string) modelref.Node {
	return modelref.Node{
		Key: modelref.Key{Scope: ref.Scope, Subject: ref.Subject},
		Version: ref.Version, DisplayName: display,
	}
}

func d4Fixture(t *testing.T) BindingRequest {
	t.Helper()
	ruleRef := d4Ref("sales", "derivation-rule", d4RuleID, "1")
	amountRef := d4Ref("sales", "attribute", d4AmountID, "1")
	queryRef := d4Ref("order-list", "query-local-result", d4QueryID, "1")
	ns := semantic.Namespace{Data: []semantic.Data{{
		ID: "sales.order-line",
		Attributes: []semantic.Attribute{
			{ID: d4AmountID, Symbol: "Amount", TypeRef: "std.decimal"},
			{ID: "sales.order-line.quantity", Symbol: "Quantity", TypeRef: "std.decimal"},
			{ID: "sales.order-line.unit-price", Symbol: "UnitPrice", TypeRef: "std.decimal"},
		},
	}}}
	realAttribute, err := SemanticAttributeTarget("sales", ns, "sales.order-line", d4AmountID, "1")
	if err != nil {
		t.Fatal(err)
	}
	if realAttribute.Ref != amountRef || realAttribute.OutputType != "std.decimal" {
		t.Fatalf("real Semantic attribute not projected exactly: %+v", realAttribute)
	}
	return BindingRequest{
		Graph: modelref.Graph{Schema: "tei.modelref.v0", Nodes: []modelref.Node{
			d4Node(ruleRef, "line-amount"), d4Node(amountRef, "Amount"),
			d4Node(queryRef, "Amount"),
		}},
		Definitions: []Definition{{ID: d4RuleID, Inputs: []string{
			"sales.order-line.quantity", "sales.order-line.unit-price",
		}, Operator: "multiply", Pure: true}},
		Targets: []TypedTarget{
			realAttribute,
			// Query-local results are a synthetic typed fixture until #1640
			// supplies their accepted result contracts to this read-only port.
			{Ref: queryRef, OutputType: "std.decimal"},
		},
		Accepted: AcceptedBindingSnapshot{
			Rules: []RuleContract{{Ref: ruleRef, DefinitionID: d4RuleID,
				OutputType: "std.decimal", NullPolicy: "propagate"}},
			Bindings: []AcceptedBinding{
				{ID: "line-attribute", Rule: ruleRef, Target: amountRef, Scope: SemanticAttributeScope},
				{ID: "query-result", Rule: ruleRef, Target: queryRef, Scope: QueryLocalScope},
			},
			Realizations: []SelectedRealization{{
				Rule: ruleRef, PluginID: "example-go", Implementation: "go:line-amount",
				OutputType: "std.decimal", NullPolicy: "propagate",
			}},
		},
	}
}

func d4HasCode(plan BindingPlan, code string) bool {
	for _, diag := range plan.Diagnostics {
		if diag.Code == code {
			return true
		}
	}
	return false
}

func TestD4TwoTargetsReuseOneAcceptedRuleAndRealSemanticAttribute(t *testing.T) {
	req := d4Fixture(t)
	got := ResolveBindings(req)
	if len(got.Diagnostics) != 0 || len(got.Bindings) != 2 {
		t.Fatalf("expected two resolved targets: %+v", got)
	}
	if got.Bindings[0].Target == got.Bindings[1].Target ||
		got.Bindings[0].Rule != got.Bindings[1].Rule {
		t.Fatalf("Rule body copied or targets conflated: %+v", got.Bindings)
	}
	for _, b := range got.Bindings {
		if b.OutputType != "std.decimal" || b.PluginID != "example-go" {
			t.Fatalf("typed plan differs: %+v", b)
		}
	}
	// Ordering is by typed target identity, not registration order.
	req.Accepted.Bindings[0], req.Accepted.Bindings[1] = req.Accepted.Bindings[1], req.Accepted.Bindings[0]
	reverse := ResolveBindings(req)
	if !reflect.DeepEqual(got, reverse) {
		t.Fatalf("registration order changed plan: normal=%+v reversed=%+v", got, reverse)
	}
}

func d4SecondRule(req *BindingRequest) {
	second := d4Ref("sales", "derivation-rule", "sales.alternative-amount", "1")
	req.Graph.Nodes = append(req.Graph.Nodes, d4Node(second, "Amount"))
	req.Definitions = append(req.Definitions, Definition{ID: second.Subject.ID, Operator: "multiply", Pure: true})
	req.Accepted.Rules = append(req.Accepted.Rules, RuleContract{
		Ref: second, DefinitionID: second.Subject.ID, OutputType: "std.decimal", NullPolicy: "propagate",
	})
	req.Accepted.Bindings = append(req.Accepted.Bindings, AcceptedBinding{
		ID: "alternative-attribute", Rule: second, Target: req.Accepted.Bindings[0].Target,
		Scope: SemanticAttributeScope,
	})
}

func TestD4SameTargetConflictRejectsUnlessAcceptedChoice(t *testing.T) {
	req := d4Fixture(t)
	d4SecondRule(&req)
	conflict := ResolveBindings(req)
	if !d4HasCode(conflict, "ambiguous-target") || len(conflict.Bindings) != 0 {
		t.Fatalf("collision must fail closed: %+v", conflict)
	}
	req.Accepted.Choices = []AcceptedChoice{{
		Target: req.Accepted.Bindings[0].Target, BindingID: "alternative-attribute",
	}}
	selected := ResolveBindings(req)
	if len(selected.Diagnostics) != 0 || len(selected.Bindings) != 2 {
		t.Fatalf("accepted choice not applied: %+v", selected)
	}
	if selected.Bindings[1].BindingID != "alternative-attribute" &&
		selected.Bindings[0].BindingID != "alternative-attribute" {
		t.Fatalf("chosen binding missing: %+v", selected)
	}
	req.Accepted.Choices = append(req.Accepted.Choices, AcceptedChoice{
		Target: req.Accepted.Bindings[0].Target, BindingID: "line-attribute",
	})
	if !d4HasCode(ResolveBindings(req), "conflicting-choice") {
		t.Fatal("two choices for one target accepted")
	}
}

func TestD4FailuresAreFailClosed(t *testing.T) {
	cases := []struct {
		name   string
		edit   func(*BindingRequest)
		code   string
	}{
		{"unaccepted-binding", func(r *BindingRequest) {
			r.ProposedBindings = []AcceptedBinding{r.Accepted.Bindings[0]}
		}, "unaccepted-binding"},
		{"unknown-target-id-same-name", func(r *BindingRequest) {
			r.Accepted.Bindings[0].Target.Subject.ID = "sales.other-amount"
			r.Graph.Nodes = append(r.Graph.Nodes, d4Node(d4Ref("sales", "attribute", "sales.other-amount", "1"), "Amount"))
		}, "target-contract-missing"},
		{"missing-target", func(r *BindingRequest) {
			r.Accepted.Bindings[0].Target.Subject.ID = "does-not-exist"
		}, "missing-reference"},
		{"wrong-kind", func(r *BindingRequest) {
			r.Accepted.Bindings[0].Target.Subject.Kind = "data"
		}, "wrong-kind"},
		{"scope-mismatch", func(r *BindingRequest) {
			r.Accepted.Bindings[0].Scope = QueryLocalScope
		}, "binding-scope-mismatch"},
		{"unqualified-scope", func(r *BindingRequest) {
			r.Accepted.Bindings[0].Target.Scope = ""
		}, "unqualified-reference"},
		{"type-mismatch", func(r *BindingRequest) {
			r.Targets[0].OutputType = "std.string"
		}, "target-type-mismatch"},
		{"rule-version-mismatch", func(r *BindingRequest) {
			r.Accepted.Bindings[0].Rule.Version = "2"
		}, "version-mismatch"},
		{"target-version-mismatch", func(r *BindingRequest) {
			r.Accepted.Bindings[0].Target.Version = "2"
		}, "version-mismatch"},
		{"unaccepted-rule", func(r *BindingRequest) {
			r.Accepted.Rules = nil
		}, "unaccepted-rule"},
		{"duplicate-binding-id", func(r *BindingRequest) {
			r.Accepted.Bindings[1].ID = r.Accepted.Bindings[0].ID
		}, "duplicate-or-empty-binding-id"},
		{"duplicate-target-type", func(r *BindingRequest) {
			r.Targets = append(r.Targets, r.Targets[0])
		}, "duplicate-target-contract"},
		{"duplicated-rule-catalog", func(r *BindingRequest) {
			r.Accepted.Rules = append(r.Accepted.Rules, r.Accepted.Rules[0])
		}, "duplicate-rule-contract"},
		{"invalid-selection", func(r *BindingRequest) {
			r.Accepted.Choices = []AcceptedChoice{{Target: r.Accepted.Bindings[0].Target, BindingID: "no-binding"}}
		}, "choice-not-applicable"},
		{"rule-identity-not-symbol", func(r *BindingRequest) {
			r.Accepted.Bindings[0].Rule.Subject.ID = "sales.other-rule"
			r.Graph.Nodes = append(r.Graph.Nodes, d4Node(d4Ref("sales", "derivation-rule", "sales.other-rule", "1"), d4RuleID))
		}, "unaccepted-rule"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := d4Fixture(t)
			tc.edit(&r)
			got := ResolveBindings(r)
			if !d4HasCode(got, tc.code) || len(got.Bindings) != 0 {
				t.Fatalf("expected %q, no partial plan: %+v", tc.code, got)
			}
		})
	}
}

func TestD4RealizationReplacementOnlyWithoutMeaningChange(t *testing.T) {
	r := d4Fixture(t)
	before := ResolveBindings(r)
	if len(before.Diagnostics) != 0 {
		t.Fatal(before.Diagnostics)
	}
	r.Accepted.Realizations[0].PluginID = "example-wasm"
	r.Accepted.Realizations[0].Implementation = "wasm:line-amount"
	after := ResolveBindings(r)
	if len(after.Diagnostics) != 0 || len(after.Bindings) != 2 {
		t.Fatalf("replacement failed: %+v", after)
	}
	for i := range before.Bindings {
		old, next := before.Bindings[i], after.Bindings[i]
		if old.Rule != next.Rule || old.Target != next.Target || old.OutputType != next.OutputType ||
			next.PluginID != "example-wasm" {
			t.Fatalf("realization incorrectly changed meaning: old=%+v new=%+v", old, next)
		}
	}
	r.Accepted.Realizations[0].NullPolicy = "null-as-zero"
	if got := ResolveBindings(r); !d4HasCode(got, "realization-contract-change-reaccept") || len(got.Bindings) != 0 {
		t.Fatalf("null semantics changed without acceptance: %+v", got)
	}
	r = d4Fixture(t)
	r.Accepted.Realizations[0].OutputType = "std.string"
	if got := ResolveBindings(r); !d4HasCode(got, "realization-contract-change-reaccept") {
		t.Fatalf("type change disguised as implementation: %+v", got)
	}
}

func TestD4ChangeImpactUsesExistingTypedModelref(t *testing.T) {
	r := d4Fixture(t)
	before := r.Graph
	after := modelref.Graph{Schema: before.Schema, Nodes: append([]modelref.Node(nil), before.Nodes...)}
	after.Nodes[0].Version = "2"
	changes := modelref.Diff(before, after)
	if len(changes) != 1 || changes[0].Kind != "version-changed" {
		t.Fatalf("rule version change not detected: %+v", changes)
	}
	r.Graph = after
	if got := ResolveBindings(r); !d4HasCode(got, "version-mismatch") {
		t.Fatalf("stale accepted rule/bindings silently retained: %+v", got)
	}
	after.Nodes = append([]modelref.Node(nil), before.Nodes[1:]...)
	changes = modelref.Diff(before, after)
	if len(changes) != 1 || changes[0].Kind != "removed" {
		t.Fatalf("deletion impact not exposed: %+v", changes)
	}
	r.Graph = after
	if got := ResolveBindings(r); !d4HasCode(got, "missing-reference") {
		t.Fatalf("deleted rule silently resolved: %+v", got)
	}
}

func TestD4SemanticAttributeProjectionRejectsMissingAndAmbiguity(t *testing.T) {
	ns := semantic.Namespace{Data: []semantic.Data{{
		ID: "orders", Attributes: []semantic.Attribute{
			{ID: "orders.total", Symbol: "Total", TypeRef: "std.decimal"},
		},
	}}}
	if _, err := SemanticAttributeTarget("sales", ns, "orders", "no-id", "1"); err == nil {
		t.Fatal("missing ID was inferred from display symbol")
	}
	ns.Data[0].Attributes = append(ns.Data[0].Attributes, ns.Data[0].Attributes[0])
	if _, err := SemanticAttributeTarget("sales", ns, "orders", "orders.total", "1"); err == nil {
		t.Fatal("duplicate Semantic Attribute was silently selected")
	}
}
