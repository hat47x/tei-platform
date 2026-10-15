package runtimechange

import "testing"

func sample() Input {
	return Input{
		Bindings: []Binding{
			{Element: "el-amount", Part: "generated", Resolved: true, Subjects: []string{"Data.Order.amount"}, Kinds: []string{"presentation", "data"}},
			{Element: "el-edit", Part: "generated", Resolved: true, Subjects: []string{"Action.Order.update"}, Kinds: []string{"action"}},
			{Element: "el-approve", Part: "native", Resolved: true, Subjects: []string{"Authorization.Order.approve"}, Kinds: []string{"authorization"}},
			{Element: "el-custom", Part: "custom", Resolved: true, Subjects: []string{"Data.Order.custom"}, Kinds: []string{"data"}},
			{Element: "el-unknown", Part: "generated", Resolved: false},
		},
		Instructions: []Instruction{
			{ID: "i1", Element: "el-amount", Target: "presentation"},
			{ID: "i2", Element: "el-edit", Target: "business-rule"},
			{ID: "i3", Element: "el-approve", Target: "authorization", ScreenOnly: true},
			{ID: "i4", Element: "el-unknown", Target: "presentation"},
			{ID: "i5", Element: "el-custom", Target: "business-rule"},
		},
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func hasFinding(findings []Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

func TestPlanSeparatesCategories(t *testing.T) {
	r := Plan(sample())
	if !contains(r.AppearanceOnly, "i1") {
		t.Errorf("i1 not appearance-only: %+v", r.AppearanceOnly)
	}
	if !contains(r.BusinessRule, "i2") || !contains(r.BusinessRule, "i5") {
		t.Errorf("business-rule = %+v", r.BusinessRule)
	}
	if !contains(r.Authorization, "i3") {
		t.Errorf("authorization = %+v", r.Authorization)
	}
	if !contains(r.Unresolved, "i4") {
		t.Errorf("unresolved = %+v", r.Unresolved)
	}
	if contains(r.AppearanceOnly, "i2") {
		t.Error("a business rule was reported as appearance-only")
	}
}

func TestPlanDoesNotGuessUnknownMapping(t *testing.T) {
	r := Plan(sample())
	for _, p := range r.Proposals {
		if p.Instruction == "i4" {
			if p.Category != "unresolved" || p.Resolved {
				t.Errorf("i4 = %+v, want unresolved and not resolved", p)
			}
		}
	}
	if !hasFinding(r.Findings, "unresolved-binding") {
		t.Errorf("missing unresolved-binding in %+v", r.Findings)
	}

	extra := sample()
	extra.Bindings = append(extra.Bindings,
		Binding{Element: "el-empty", Part: "generated", Resolved: true},
	)
	extra.Instructions = append(extra.Instructions,
		Instruction{ID: "i6", Element: "el-empty", Target: "presentation"},
		Instruction{ID: "i7", Element: "missing", Target: "presentation"},
	)
	r2 := Plan(extra)
	if !hasFinding(r2.Findings, "binding-without-subject") {
		t.Errorf("missing binding-without-subject in %+v", r2.Findings)
	}
	if !hasFinding(r2.Findings, "unknown-element") {
		t.Errorf("missing unknown-element in %+v", r2.Findings)
	}
	if !contains(r2.Unresolved, "i6") || !contains(r2.Unresolved, "i7") {
		t.Errorf("unresolved = %+v", r2.Unresolved)
	}
}

func TestScreenDisableIsNotAuthorization(t *testing.T) {
	r := Plan(sample())
	if !hasFinding(r.Findings, "screen-disable-does-not-grant-authorization") {
		t.Fatalf("missing screen-disable finding in %+v", r.Findings)
	}
	acc := Accept(sampleWithAccepted("i3"))
	if acc.SourceChanged {
		t.Fatal("a screen-only change moved the Source")
	}
	if !hasFinding(acc.Findings, "refused-accept:screen-only-not-authorization") {
		t.Errorf("screen-only authorization change was not refused: %+v", acc.Findings)
	}
}

func TestUnacceptedChangeLeavesSourceUnchanged(t *testing.T) {
	plan := Plan(sample())
	if plan.SourceChanged {
		t.Fatal("Plan must not change the Source")
	}
	acc := Accept(sample())
	if acc.SourceChanged {
		t.Fatal("an unaccepted change changed the Source")
	}
	if acc.MeaningFingerprint != plan.MeaningFingerprint {
		t.Fatalf("fingerprint moved without acceptance: %s vs %s", acc.MeaningFingerprint, plan.MeaningFingerprint)
	}
}

func TestAppearanceChangeDoesNotMoveMeaning(t *testing.T) {
	plan := Plan(sample())
	acc := Accept(sampleWithAccepted("i1"))
	if acc.SourceChanged {
		t.Fatal("an appearance-only change changed the Source")
	}
	if acc.MeaningFingerprint != plan.MeaningFingerprint {
		t.Fatalf("appearance-only change moved the meaning fingerprint: %s vs %s", acc.MeaningFingerprint, plan.MeaningFingerprint)
	}
	if len(acc.Applied) == 0 {
		t.Fatal("the appearance-only change was not applied")
	}
}

func TestAcceptedBusinessRuleMovesMeaning(t *testing.T) {
	plan := Plan(sample())
	acc := Accept(sampleWithAccepted("i2"))
	if !acc.SourceChanged {
		t.Fatal("an accepted business rule did not change the Source")
	}
	if acc.MeaningFingerprint == plan.MeaningFingerprint {
		t.Fatal("an accepted business rule did not move the meaning fingerprint")
	}
	found := false
	for _, a := range acc.Applied {
		if a.Instruction == "i2" && a.Subject == "Action.Order.update" {
			found = true
		}
	}
	if !found {
		t.Fatalf("accepted change not applied: %+v", acc.Applied)
	}
}

func TestSupportRangesAndRefusal(t *testing.T) {
	r := Plan(sample())
	support := map[string]string{}
	for _, p := range r.Proposals {
		support[p.Element] = p.Support
	}
	if support["el-amount"] != "supported" {
		t.Errorf("generated part support = %q", support["el-amount"])
	}
	if support["el-approve"] != "review-required" {
		t.Errorf("native part support = %q", support["el-approve"])
	}
	if support["el-custom"] != "unsupported" {
		t.Errorf("custom part support = %q", support["el-custom"])
	}
	if !hasFinding(r.Findings, "unsupported-part") {
		t.Errorf("missing unsupported-part in %+v", r.Findings)
	}
	acc := Accept(sampleWithAccepted("i5"))
	if hasFinding(acc.Findings, "refused-accept:unsupported-part") == false {
		t.Errorf("custom part was not refused: %+v", acc.Findings)
	}
	for _, a := range acc.Applied {
		if a.Instruction == "i5" {
			t.Fatalf("an unsupported-part change was applied: %+v", a)
		}
	}
}

func TestAcceptIsDeterministic(t *testing.T) {
	a := Accept(sampleWithAccepted("i2", "i1"))
	b := Accept(sampleWithAccepted("i1", "i2"))
	if Fingerprint(a) != Fingerprint(b) {
		t.Fatalf("acceptance is not order-independent: %s vs %s", Fingerprint(a), Fingerprint(b))
	}
}

func sampleWithAccepted(ids ...string) Input {
	in := sample()
	in.Accepted = ids
	return in
}
