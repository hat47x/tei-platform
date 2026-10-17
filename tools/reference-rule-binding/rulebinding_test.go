package rulebinding

import "testing"

// The fixtures follow Data Derivation D4 (#1692): rule `sales.line-amount`
// (quantity × unit_price) is reused by several targets.

func baseRules() []Rule {
	return []Rule{
		{ID: "sales.line-amount", Version: "1", Accepted: true, Type: "amount", NullPolicy: "propagate"},
		{ID: "sales.line-gross", Version: "1", Accepted: true, Type: "amount", NullPolicy: "propagate"},
		{ID: "sales.line-label", Version: "1", Accepted: true, Type: "text"},
	}
}

func baseTargets() []Target {
	return []Target{
		{ID: "sales.order-line.amount", Type: "amount"},
		{ID: "query.order-list.line-total", Type: "amount"},
		{ID: "sales.order-line.label", Type: "text"},
	}
}

func TestOneRuleResolvesToSeveralTargets(t *testing.T) {
	in := Input{
		Rules:   baseRules(),
		Targets: baseTargets(),
		Bindings: []Binding{
			{RuleID: "sales.line-amount", TargetID: "sales.order-line.amount"},
			{RuleID: "sales.line-amount", TargetID: "query.order-list.line-total"},
		},
	}
	for _, target := range []string{"sales.order-line.amount", "query.order-list.line-total"} {
		in.ResolveTarget = target
		report := Resolve(in)
		if !report.Accepted || report.Code != CodeResolved || report.RuleID != "sales.line-amount" || report.RuleVersion != "1" {
			t.Fatalf("target=%s report=%+v want RESOLVED via sales.line-amount", target, report)
		}
	}
}

func TestConflictingBindingsAreAmbiguous(t *testing.T) {
	in := Input{
		Rules:   baseRules(),
		Targets: baseTargets(),
		Bindings: []Binding{
			{RuleID: "sales.line-amount", TargetID: "sales.order-line.amount"},
			{RuleID: "sales.line-gross", TargetID: "sales.order-line.amount"},
		},
		ResolveTarget: "sales.order-line.amount",
	}
	report := Resolve(in)
	if report.Accepted || report.Code != CodeAmbiguous {
		t.Fatalf("report=%+v want AMBIGUOUS", report)
	}
}

func TestUniqueExplicitPriorityResolvesConflict(t *testing.T) {
	in := Input{
		Rules:   baseRules(),
		Targets: baseTargets(),
		Bindings: []Binding{
			{RuleID: "sales.line-amount", TargetID: "sales.order-line.amount", Priority: 10},
			{RuleID: "sales.line-gross", TargetID: "sales.order-line.amount"},
		},
		ResolveTarget: "sales.order-line.amount",
	}
	report := Resolve(in)
	if !report.Accepted || report.Code != CodeResolved || report.RuleID != "sales.line-amount" {
		t.Fatalf("report=%+v want RESOLVED via priority", report)
	}
}

func TestImplementationReplacementKeepsMeaning(t *testing.T) {
	in := Input{
		Rules:   baseRules(),
		Targets: baseTargets(),
		Bindings: []Binding{
			{RuleID: "sales.line-amount", TargetID: "sales.order-line.amount"},
		},
		Overrides: []Override{
			{TargetID: "sales.order-line.amount", RuleID: "sales.line-amount", Kind: OverrideImplementation, FromImpl: "go", ToImpl: "wasm"},
		},
		ResolveTarget: "sales.order-line.amount",
	}
	report := Resolve(in)
	if !report.Accepted || report.Code != CodeImplementationReplaced || report.RuleID != "sales.line-amount" {
		t.Fatalf("report=%+v want IMPLEMENTATION_REPLACED", report)
	}
	if len(report.Implementations) != 2 || report.Implementations[0] != "go" || report.Implementations[1] != "wasm" {
		t.Fatalf("implementations=%v want [go wasm]", report.Implementations)
	}
}

func TestContractChangeOverrideRequiresReacceptance(t *testing.T) {
	in := Input{
		Rules:   baseRules(),
		Targets: baseTargets(),
		Bindings: []Binding{
			{RuleID: "sales.line-amount", TargetID: "sales.order-line.amount"},
		},
		Overrides: []Override{
			{TargetID: "sales.order-line.amount", RuleID: "sales.line-amount", Kind: OverrideContractChange, FromImpl: "propagate", ToImpl: "zero"},
		},
		ResolveTarget: "sales.order-line.amount",
	}
	report := Resolve(in)
	if report.Accepted || report.Code != CodeContractChangeRequiresAccept {
		t.Fatalf("report=%+v want CONTRACT_CHANGE_REQUIRES_ACCEPTANCE", report)
	}
}

func TestSameNameDifferentNamespaceIsRejected(t *testing.T) {
	in := Input{
		Rules:   baseRules(),
		Targets: baseTargets(),
		Bindings: []Binding{
			{RuleID: "other.line-amount", TargetID: "sales.order-line.amount"},
		},
		ResolveTarget: "sales.order-line.amount",
	}
	report := Resolve(in)
	if report.Accepted || report.Code != CodeSameNameDifferentID {
		t.Fatalf("report=%+v want SAME_NAME_DIFFERENT_ID", report)
	}
}

func TestTypeMismatchIsRejected(t *testing.T) {
	in := Input{
		Rules:   baseRules(),
		Targets: baseTargets(),
		Bindings: []Binding{
			{RuleID: "sales.line-label", TargetID: "sales.order-line.amount"},
		},
		ResolveTarget: "sales.order-line.amount",
	}
	report := Resolve(in)
	if report.Accepted || report.Code != CodeTypeMismatch {
		t.Fatalf("report=%+v want TYPE_MISMATCH", report)
	}
}

func TestUnacceptedRuleIsRejected(t *testing.T) {
	rules := baseRules()
	rules = append(rules, Rule{ID: "sales.line-tax", Version: "1", Accepted: false, Type: "amount"})
	in := Input{
		Rules:   rules,
		Targets: baseTargets(),
		Bindings: []Binding{
			{RuleID: "sales.line-tax", TargetID: "sales.order-line.amount"},
		},
		ResolveTarget: "sales.order-line.amount",
	}
	report := Resolve(in)
	if report.Accepted || report.Code != CodeRuleNotAccepted {
		t.Fatalf("report=%+v want RULE_NOT_ACCEPTED", report)
	}
}

func TestUnregisteredPluginIsRejected(t *testing.T) {
	in := Input{
		Rules:   baseRules(),
		Targets: baseTargets(),
		Bindings: []Binding{
			{RuleID: "sales.line-amount", TargetID: "sales.order-line.amount", Plugin: "case-acme"},
		},
		RegisteredPlugins: []string{"case-other"},
		ResolveTarget:     "sales.order-line.amount",
	}
	report := Resolve(in)
	if report.Accepted || report.Code != CodePluginNotRegistered {
		t.Fatalf("report=%+v want PLUGIN_NOT_REGISTERED", report)
	}
}

func TestUnknownTargetAndUnboundTarget(t *testing.T) {
	in := Input{
		Rules:         baseRules(),
		Targets:       baseTargets(),
		ResolveTarget: "sales.unknown",
	}
	if report := Resolve(in); report.Code != CodeTargetUnknown {
		t.Fatalf("report=%+v want TARGET_UNKNOWN", report)
	}
	in.ResolveTarget = "sales.order-line.label"
	if report := Resolve(in); report.Code != CodeUnbound {
		t.Fatalf("report=%+v want UNBOUND", report)
	}
}
