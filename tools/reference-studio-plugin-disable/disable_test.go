package studioplugindisable

import (
	"reflect"
	"strings"
	"testing"
)

func fixture() State {
	return State{
		SettingsVersion: "s3", GitHead: "head-abc", Permission: "allowed",
		StandardEditorAvailable: true,
		Plugins: []Plugin{
			{ID: "a", Installed: true, Enabled: true, RequiredDeps: []string{"b"}},
			{ID: "b", Installed: true, Enabled: true, RequiredDeps: []string{"c"}},
			{ID: "c", Installed: true, Enabled: true},
			{ID: "d", Installed: true, Enabled: true, OptionalDeps: []string{"b"}},
			{ID: "e", Installed: true, Enabled: true},
			{ID: "f", Installed: false, Enabled: false, RequiredDeps: []string{"b"}},
		},
		Sources: []Source{
			{Path: "semantic/order.tei", Fingerprint: "sha256:aaa", UnknownAttributes: true, UnknownElements: true},
			{Path: "semantic/customer.tei", Fingerprint: "sha256:bbb"},
		},
	}
}

func enabledMap(state State) map[string]bool {
	out := map[string]bool{}
	for _, p := range state.Plugins {
		out[p.ID] = p.Enabled
	}
	return out
}

func hasPrefix(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func TestDisableBExpandsToDependentsAndLeavesC(t *testing.T) {
	plan, err := PlanAction(fixture(), Request{Action: ActionDisable, PluginID: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Refused {
		t.Fatalf("plan refused: %v", plan.Reasons)
	}
	if !reflect.DeepEqual(plan.Candidates, []string{"a", "b"}) {
		t.Fatalf("candidates=%v want [a b]", plan.Candidates)
	}
	if !reflect.DeepEqual(plan.Additional, []string{"a"}) {
		t.Fatalf("additional=%v want [a]", plan.Additional)
	}
	if !reflect.DeepEqual(plan.Unchanged, []string{"c", "d", "e", "f"}) {
		t.Fatalf("unchanged=%v want [c d e f]", plan.Unchanged)
	}
	if !contains(plan.OptionalFeaturesLost, "d:b") {
		t.Fatalf("optional loss not recorded: %v", plan.OptionalFeaturesLost)
	}
	if !plan.ConfirmationRequired {
		t.Fatal("expanded candidates did not require confirmation")
	}
	if plan.SelectionChanged {
		t.Fatal("the plan changed a row selection")
	}
	if !plan.SourcesPreserved {
		t.Fatal("the plan claimed to change sources")
	}
}

func TestApplyWithoutConfirmationChangesNothing(t *testing.T) {
	state := fixture()
	plan, err := PlanAction(state, Request{Action: ActionDisable, PluginID: "b"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Apply(state, plan, ApplyRequest{}, func([]string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if result.SettingsApplied {
		t.Fatal("settings applied without confirmation")
	}
	if !contains(result.Reasons, "confirmation-required") {
		t.Fatalf("reasons=%v", result.Reasons)
	}
	if result.EnabledAfter != nil {
		t.Fatalf("a refused apply produced settings: %v", result.EnabledAfter)
	}
}

func TestApplyDisablesCandidatesAndPreservesSources(t *testing.T) {
	state := fixture()
	before := enabledMap(state)
	plan, err := PlanAction(state, Request{Action: ActionDisable, PluginID: "b"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Apply(state, plan, ApplyRequest{Confirm: true}, func([]string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if !result.SettingsApplied || !result.ExtensionsStarted || !result.AllSucceeded {
		t.Fatalf("result=%+v", result)
	}
	want := map[string]bool{"a": false, "b": false, "c": true, "d": true, "e": true, "f": false}
	if !reflect.DeepEqual(result.EnabledAfter, want) {
		t.Fatalf("enabledAfter=%v want %v", result.EnabledAfter, want)
	}
	for _, id := range []string{"c", "d", "e"} {
		if result.EnabledAfter[id] != before[id] {
			t.Errorf("unrelated plugin %q changed", id)
		}
	}
	if !result.SourcePreserved || !reflect.DeepEqual(result.SourcesAfter, state.Sources) {
		t.Fatalf("sources changed: %+v", result.SourcesAfter)
	}
	if !result.StandardEditorAvailable {
		t.Fatal("the standard editor entry was lost")
	}
	if result.PluginSpecificFeature {
		t.Fatal("the plugin-specific feature was reported as available")
	}
	if result.AutoInstalled || result.AutoStarted {
		t.Fatal("the apply installed or started a plugin by itself")
	}
	if !result.SettingsAtomic || result.ExternalEffectAtomic {
		t.Fatalf("atomicity flags=%v/%v", result.SettingsAtomic, result.ExternalEffectAtomic)
	}
	if reflect.DeepEqual(enabledMap(state), before) == false {
		t.Fatal("the input state was mutated")
	}
}

func TestRemoveKeepsSourcesAndMarksRemoved(t *testing.T) {
	state := fixture()
	plan, err := PlanAction(state, Request{Action: ActionRemove, PluginID: "b"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Apply(state, plan, ApplyRequest{Confirm: true}, func([]string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.RemovedAfter, []string{"a", "b"}) {
		t.Fatalf("removed=%v", result.RemovedAfter)
	}
	if !result.SourcePreserved || !reflect.DeepEqual(result.SourcesAfter, state.Sources) {
		t.Fatal("remove changed the design sources")
	}
	if !result.StandardEditorAvailable {
		t.Fatal("remove lost the standard editor")
	}
}

func TestCycleIsDistinctAndRefused(t *testing.T) {
	state := State{SettingsVersion: "s1", Permission: "allowed", StandardEditorAvailable: true,
		Plugins: []Plugin{
			{ID: "x", Installed: true, Enabled: true, RequiredDeps: []string{"y"}},
			{ID: "y", Installed: true, Enabled: true, RequiredDeps: []string{"x"}},
		}}
	plan, err := PlanAction(state, Request{Action: ActionDisable, PluginID: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Refused || !contains(plan.Reasons, "dependency-cycle") {
		t.Fatalf("plan=%+v", plan)
	}
	if len(plan.Cycles) == 0 {
		t.Fatal("cycle was not named")
	}
}

func TestMissingDependencyIsUnresolved(t *testing.T) {
	state := State{SettingsVersion: "s1", Permission: "allowed", StandardEditorAvailable: true,
		Plugins: []Plugin{{ID: "m", Installed: true, Enabled: true, RequiredDeps: []string{"ghost"}}}}
	plan, err := PlanAction(state, Request{Action: ActionDisable, PluginID: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Refused || !contains(plan.Reasons, "unresolved-dependency") {
		t.Fatalf("plan=%+v", plan)
	}
	if !contains(plan.Unresolved, "m:ghost") {
		t.Fatalf("unresolved=%v", plan.Unresolved)
	}
}

func TestStaleSettingsIsRefused(t *testing.T) {
	state := fixture()
	plan, err := PlanAction(state, Request{Action: ActionDisable, PluginID: "b"})
	if err != nil {
		t.Fatal(err)
	}
	changed := fixture()
	changed.SettingsVersion = "s4"
	result, err := Apply(changed, plan, ApplyRequest{Confirm: true}, func([]string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if result.SettingsApplied {
		t.Fatal("a stale settings version was applied")
	}
	if !hasPrefix(result.Reasons, "stale-settings:") {
		t.Fatalf("reasons=%v", result.Reasons)
	}
}

func TestUncommittedSourceEditIsDetectedWithSameHead(t *testing.T) {
	state := fixture()
	plan, err := PlanAction(state, Request{Action: ActionDisable, PluginID: "b"})
	if err != nil {
		t.Fatal(err)
	}
	changed := fixture()
	changed.Sources[0].Fingerprint = "sha256:edited-but-not-committed"
	// The Git HEAD, settings version, declarations and permission are equal.
	result, err := Apply(changed, plan, ApplyRequest{Confirm: true}, func([]string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if result.SettingsApplied {
		t.Fatal("an uncommitted source edit was not detected")
	}
	if !contains(result.Reasons, "stale-source:semantic/order.tei") {
		t.Fatalf("reasons=%v", result.Reasons)
	}
}

func TestPermissionRevocationIsRefused(t *testing.T) {
	state := fixture()
	plan, err := PlanAction(state, Request{Action: ActionDisable, PluginID: "b"})
	if err != nil {
		t.Fatal(err)
	}
	changed := fixture()
	changed.Permission = "denied"
	result, err := Apply(changed, plan, ApplyRequest{Confirm: true}, func([]string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if result.SettingsApplied {
		t.Fatal("a revoked permission was applied")
	}
	if !hasPrefix(result.Reasons, "permission-revoked:") {
		t.Fatalf("reasons=%v", result.Reasons)
	}
}

func TestExtensionFailureIsNotAllSuccess(t *testing.T) {
	state := fixture()
	plan, err := PlanAction(state, Request{Action: ActionDisable, PluginID: "b"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Apply(state, plan, ApplyRequest{Confirm: true}, func([]string) error {
		return &syntheticError{"start failed"}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.SettingsApplied {
		t.Fatal("settings were not applied before the extension start")
	}
	if result.ExtensionsStarted || result.AllSucceeded {
		t.Fatalf("an extension failure was reported as success: %+v", result)
	}
	if len(result.ExtensionErrors) == 0 {
		t.Fatal("the extension error was not recorded")
	}
	if !result.SettingsAtomic || result.ExternalEffectAtomic {
		t.Fatal("settings atomicity and external effects were conflated")
	}
}

type syntheticError struct{ message string }

func (e *syntheticError) Error() string { return e.message }

func TestEnableRequiresExplicitDependencies(t *testing.T) {
	state := State{SettingsVersion: "s1", Permission: "allowed", StandardEditorAvailable: true,
		Plugins: []Plugin{
			{ID: "a", Installed: true, Enabled: false, RequiredDeps: []string{"b"}},
			{ID: "b", Installed: true, Enabled: false, RequiredDeps: []string{"c"}},
			{ID: "c", Installed: true, Enabled: true},
		}}
	refused, err := PlanAction(state, Request{Action: ActionEnable, PluginID: "a"})
	if err != nil {
		t.Fatal(err)
	}
	if !refused.Refused || !contains(refused.Reasons, "dependency-not-enabled:b") {
		t.Fatalf("plan=%+v", refused)
	}
	enableB, err := PlanAction(state, Request{Action: ActionEnable, PluginID: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if enableB.Refused {
		t.Fatalf("enable b refused: %v", enableB.Reasons)
	}
	bApplied, err := Apply(state, enableB, ApplyRequest{Confirm: true}, func([]string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if !bApplied.SettingsApplied || !bApplied.EnabledAfter["b"] {
		t.Fatalf("enable b=%+v", bApplied)
	}
	enabledState := state
	enabledState.Plugins = append([]Plugin(nil), state.Plugins...)
	enabledState.Plugins[1].Enabled = true
	enableA, err := PlanAction(enabledState, Request{Action: ActionEnable, PluginID: "a"})
	if err != nil {
		t.Fatal(err)
	}
	aApplied, err := Apply(enabledState, enableA, ApplyRequest{Confirm: true}, func([]string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if !aApplied.SettingsApplied || !aApplied.EnabledAfter["a"] {
		t.Fatalf("enable a=%+v", aApplied)
	}
	if aApplied.AutoInstalled || aApplied.AutoStarted {
		t.Fatal("enable installed or started a plugin by itself")
	}
	if !aApplied.PluginSpecificFeature {
		t.Fatal("enable did not restore the plugin-specific feature")
	}
}

func TestPlanAndApplyAreDeterministicAndDoNotMutateTheState(t *testing.T) {
	state := fixture()
	before := enabledMap(state)
	first, err := PlanAction(state, Request{Action: ActionDisable, PluginID: "b"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := PlanAction(state, Request{Action: ActionDisable, PluginID: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("the same input produced different plans")
	}
	if !reflect.DeepEqual(enabledMap(state), before) {
		t.Fatal("PlanAction mutated the state")
	}
	apply1, err := Apply(state, first, ApplyRequest{Confirm: true}, func([]string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	apply2, err := Apply(state, first, ApplyRequest{Confirm: true}, func([]string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(apply1, apply2) {
		t.Fatal("the same plan applied differently")
	}
	if !reflect.DeepEqual(enabledMap(state), before) {
		t.Fatal("Apply mutated the state")
	}
}

func TestTamperedPlanIsRejected(t *testing.T) {
	state := fixture()
	plan, err := PlanAction(state, Request{Action: ActionDisable, PluginID: "b"})
	if err != nil {
		t.Fatal(err)
	}
	plan.Candidates = []string{"a", "b", "e"}
	result, err := Apply(state, plan, ApplyRequest{Confirm: true}, func([]string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if result.SettingsApplied || !contains(result.Reasons, "plan-tampered") {
		t.Fatalf("result=%+v", result)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
