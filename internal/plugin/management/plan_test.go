package management

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
)

// chainDefinitions builds A -> B -> C, where each plugin requires the next.
func chainDefinitions(t *testing.T) []plugin.Definition {
	t.Helper()
	return []plugin.Definition{
		{
			ID:       cid(t, "a"),
			Version:  "1.0.0",
			Runtime:  plugin.Runtime{Kind: plugin.RuntimeBuiltin},
			Requires: []plugin.Requirement{{Plugin: cid(t, "b"), CompatibleMajor: 1}},
		},
		{
			ID:       cid(t, "b"),
			Version:  "1.0.0",
			Runtime:  plugin.Runtime{Kind: plugin.RuntimeBuiltin},
			Requires: []plugin.Requirement{{Plugin: cid(t, "c"), CompatibleMajor: 1}},
		},
		builtinPlugin(t, "c"),
		builtinPlugin(t, "d"),
	}
}

func chainEnvironment(t *testing.T) environment.Environment {
	t.Helper()
	env := environment.Environment{ID: cid(t, "development")}
	env.Plugins = []environment.ExtensionRequirement{
		{ID: cid(t, "a"), CompatibleMajor: 1},
		{ID: cid(t, "d"), CompatibleMajor: 1},
	}
	return env
}

// TestDisableBExpandsToDependentsAndLeavesDUnchanged is the central negative:
// disabling B must propose A+B and must not touch C or D, and must demand an
// explicit confirmation because the candidate set grew.
func TestDisableBExpandsToDependentsAndLeavesDUnchanged(t *testing.T) {
	defs := chainDefinitions(t)
	env := chainEnvironment(t)
	plan, err := DerivePlan(PlanInput{
		World:       world(defs, env),
		Environment: env.ID,
		Request:     PlanRequest{Action: ActionDisable, PluginID: "b"},
	})
	if err != nil {
		t.Fatalf("DerivePlan: %v", err)
	}
	if plan.Refused {
		t.Fatalf("the plan must be proposed, not refused: %+v", plan.Reasons)
	}
	if got := strings.Join(plan.Candidates, ","); got != "a,b" {
		t.Fatalf("candidates = %q, want a,b", got)
	}
	if got := strings.Join(plan.Additional, ","); got != "a" {
		t.Fatalf("additional = %q, want a", got)
	}
	if got := strings.Join(plan.Unchanged, ","); got != "c,d" {
		t.Fatalf("unchanged = %q, want c,d", got)
	}
	if !plan.ConfirmationRequired {
		t.Fatalf("a plan that stops a dependent must require an explicit confirmation")
	}
	if !plan.SourcesPreserved || !plan.DataPreserved {
		t.Fatalf("disable must preserve Source and data: %+v", plan)
	}
	if plan.SelectionChanged {
		t.Fatalf("a plan must not change the row selection")
	}
	if len(plan.RestartRequired) == 0 {
		t.Fatalf("a static target needs a restart, and the plan must say so")
	}
	if !plan.OptionalDependencyNotModelled {
		t.Fatalf("TEI has no optional dependency, and the plan must say so")
	}
	if plan.Digest == "" {
		t.Fatalf("a plan must carry a digest so an applying operation can refuse a stale one")
	}

	// Nothing in the World was mutated: the derivation is read-only.
	if len(defs) != 4 || len(env.Plugins) != 2 {
		t.Fatalf("the input must not be mutated")
	}
}

// TestDisableTransitiveDependentIsIncluded pins the transitive rule: stopping
// C stops B, and stopping B stops A, so all three are candidates.
func TestDisableTransitiveDependentIsIncluded(t *testing.T) {
	defs := chainDefinitions(t)
	env := chainEnvironment(t)
	plan, err := DerivePlan(PlanInput{
		World:       world(defs, env),
		Environment: env.ID,
		Request:     PlanRequest{Action: ActionDisable, PluginID: "c"},
	})
	if err != nil {
		t.Fatalf("DerivePlan: %v", err)
	}
	if got := strings.Join(plan.Candidates, ","); got != "a,b,c" {
		t.Fatalf("candidates = %q, want a,b,c (the transitive dependents)", got)
	}
	if got := strings.Join(plan.Unchanged, ","); got != "d" {
		t.Fatalf("unchanged = %q, want d", got)
	}
	if !plan.ConfirmationRequired {
		t.Fatalf("stopping a transitive dependent needs an explicit confirmation")
	}
}

// TestRemoveIsRefusedWhileSelectedDependentsRemain pins the rule that a
// removal never silently strands a dependent: the plan refuses instead of
// proposing a candidate set.
func TestRemoveIsRefusedWhileSelectedDependentsRemain(t *testing.T) {
	defs := chainDefinitions(t)
	env := chainEnvironment(t)
	plan, err := DerivePlan(PlanInput{
		World:       world(defs, env),
		Environment: env.ID,
		Request:     PlanRequest{Action: ActionRemove, PluginID: "b"},
	})
	if err != nil {
		t.Fatalf("DerivePlan: %v", err)
	}
	if !plan.Refused {
		t.Fatalf("removing a plugin other selected plugins require must be refused")
	}
	if !containsString(plan.Reasons, "still-required") {
		t.Fatalf("reasons = %+v, want still-required", plan.Reasons)
	}
	if len(plan.Candidates) != 0 {
		t.Fatalf("a refused plan must not propose candidates: %+v", plan.Candidates)
	}
	if len(plan.Conflicts) == 0 {
		t.Fatalf("the refused plan must name the dependents: %+v", plan)
	}
}

// TestRemoveDataPreserved checks that removing a plugin that nothing requires
// keeps data and names regeneration.
func TestRemoveDataPreserved(t *testing.T) {
	env := environment.Environment{ID: cid(t, "development")}
	env.Plugins = []environment.ExtensionRequirement{{ID: cid(t, "alone"), CompatibleMajor: 1}}
	defs := []plugin.Definition{builtinPlugin(t, "alone")}
	plan, err := DerivePlan(PlanInput{
		World:       world(defs, env),
		Environment: env.ID,
		Request:     PlanRequest{Action: ActionRemove, PluginID: "alone"},
	})
	if err != nil {
		t.Fatalf("DerivePlan: %v", err)
	}
	if plan.Refused {
		t.Fatalf("an unrequired plugin may be removed: %+v", plan.Reasons)
	}
	if !plan.DataPreserved {
		t.Fatalf("removal must preserve data")
	}
	if len(plan.RegenerationRequired) == 0 {
		t.Fatalf("removal changes the build input, so regeneration must be named")
	}
}

// TestNotSelectedPluginIsRefused checks that a plan only applies to a plugin
// the environment actually selects.
func TestNotSelectedPluginIsRefused(t *testing.T) {
	defs := chainDefinitions(t)
	env := chainEnvironment(t)
	plan, err := DerivePlan(PlanInput{
		World:       world(defs, env),
		Environment: env.ID,
		Request:     PlanRequest{Action: ActionDisable, PluginID: "not-declared-here"},
	})
	if err != nil {
		t.Fatalf("DerivePlan: %v", err)
	}
	if !plan.Refused {
		t.Fatalf("a plugin the environment does not select cannot be planned")
	}
	if !containsString(plan.Reasons, "not-selected") {
		t.Fatalf("reasons = %+v, want not-selected", plan.Reasons)
	}
}

// TestUnknownActionIsRefused keeps the action vocabulary closed.
func TestUnknownActionIsRefused(t *testing.T) {
	env := chainEnvironment(t)
	_, err := DerivePlan(PlanInput{
		World:       world(chainDefinitions(t), env),
		Environment: env.ID,
		Request:     PlanRequest{Action: "delete-everything", PluginID: "a"},
	})
	if err == nil {
		t.Fatalf("an unknown action must be refused")
	}
	var unsupported *UnsupportedPlanError
	if !asUnsupported(err, &unsupported) {
		t.Fatalf("wrong error type: %T", err)
	}
}

func asUnsupported(err error, target **UnsupportedPlanError) bool {
	value, ok := err.(*UnsupportedPlanError)
	if ok {
		*target = value
	}
	return ok
}

// TestUnknownEnvironmentIsRefused checks the environment is read from the
// World, not assumed.
func TestUnknownEnvironmentIsRefused(t *testing.T) {
	env := chainEnvironment(t)
	_, err := DerivePlan(PlanInput{
		World:       world(chainDefinitions(t), env),
		Environment: cid(t, "production"),
		Request:     PlanRequest{Action: ActionDisable, PluginID: "a"},
	})
	if err == nil {
		t.Fatalf("an unknown environment must be refused")
	}
	var unknown *UnknownEnvironmentError
	if !asUnknownEnvironment(err, &unknown) {
		t.Fatalf("wrong error type: %T", err)
	}
}

func asUnknownEnvironment(err error, target **UnknownEnvironmentError) bool {
	value, ok := err.(*UnknownEnvironmentError)
	if ok {
		*target = value
	}
	return ok
}

// TestDeleteDataIsNeverRunnableUnattended pins the A4 boundary: the plan for a
// destructive operation names the approval and the missing recovery path, and
// lists the action as unsafe to run unattended.
func TestDeleteDataIsNeverRunnableUnattended(t *testing.T) {
	env := chainEnvironment(t)
	plan, err := DerivePlan(PlanInput{
		World:       world(chainDefinitions(t), env),
		Environment: env.ID,
		Request:     PlanRequest{Action: ActionDeleteData, PluginID: "c"},
	})
	if err != nil {
		t.Fatalf("DerivePlan: %v", err)
	}
	if !plan.Refused {
		t.Fatalf("data deletion must be refused until the recovery path exists")
	}
	if !containsString(plan.Reasons, "requires-owner-approval") {
		t.Fatalf("reasons = %+v, want requires-owner-approval", plan.Reasons)
	}
	if !containsString(plan.Reasons, "recovery-path-not-established") {
		t.Fatalf("reasons = %+v, want recovery-path-not-established", plan.Reasons)
	}
	if !containsString(plan.UnattendedUnsafe, ActionDeleteData) {
		t.Fatalf("data deletion must be listed as unsafe unattended: %+v", plan.UnattendedUnsafe)
	}
	if plan.DataPreserved {
		t.Fatalf("a data deletion plan must not claim data is preserved")
	}
}

// TestConvertDataIsNeverRunnableUnattended is the same boundary for conversion.
func TestConvertDataIsNeverRunnableUnattended(t *testing.T) {
	env := chainEnvironment(t)
	plan, err := DerivePlan(PlanInput{
		World:       world(chainDefinitions(t), env),
		Environment: env.ID,
		Request:     PlanRequest{Action: ActionConvertData, PluginID: "c"},
	})
	if err != nil {
		t.Fatalf("DerivePlan: %v", err)
	}
	if !plan.Refused {
		t.Fatalf("conversion must be refused until the recovery path exists")
	}
	if !containsString(plan.UnattendedUnsafe, ActionConvertData) {
		t.Fatalf("conversion must be listed as unsafe unattended: %+v", plan.UnattendedUnsafe)
	}
}

// TestUpdateWithoutTargetLockIsRefused checks the update plan does not invent
// a target.
func TestUpdateWithoutTargetLockIsRefused(t *testing.T) {
	env := chainEnvironment(t)
	plan, err := DerivePlan(PlanInput{
		World:       world(chainDefinitions(t), env),
		Environment: env.ID,
		Request:     PlanRequest{Action: ActionUpdate, PluginID: "b"},
	})
	if err != nil {
		t.Fatalf("DerivePlan: %v", err)
	}
	if !plan.Refused {
		t.Fatalf("an update needs the target lock and must refuse without it")
	}
	if !containsString(plan.Reasons, "target-lock-required") {
		t.Fatalf("reasons = %+v, want target-lock-required", plan.Reasons)
	}
}

// TestUpdateDiffsTwoLocks checks the update plan reports version/contract/
// artifact/build changes separately.
func TestUpdateDiffsTwoLocks(t *testing.T) {
	defs := chainDefinitions(t)
	env := chainEnvironment(t)
	catalog := plugin.Catalog{Plugins: defs}
	current, err := plugin.BuildLock(catalog, []environment.Environment{env})
	if err != nil {
		t.Fatalf("BuildLock: %v", err)
	}
	next := *cloneLock(current)
	for i := range next.Environments[0].Plugins {
		if next.Environments[0].Plugins[i].ID == cid(t, "b") {
			next.Environments[0].Plugins[i].Fingerprint = "changed-contract"
			next.Environments[0].Plugins[i].ArtifactDigest = "sha256:" + strings.Repeat("a", 64)
		}
	}
	plan, err := DerivePlan(PlanInput{
		World:       world(defs, env),
		Environment: env.ID,
		Lock:        &current,
		TargetLock:  &next,
		Request:     PlanRequest{Action: ActionUpdate, PluginID: "b"},
	})
	if err != nil {
		t.Fatalf("DerivePlan: %v", err)
	}
	if plan.Refused {
		t.Fatalf("a lock diff must be proposed: %+v", plan.Reasons)
	}
	kinds := map[string]bool{}
	for _, change := range plan.UpdateChanges {
		kinds[change.Kind] = true
		if change.PluginID != "b" {
			t.Fatalf("only b changed in the target lock: %+v", plan.UpdateChanges)
		}
	}
	if !kinds["contract"] || !kinds["artifact"] {
		t.Fatalf("contract and artifact changes must be separate: %+v", plan.UpdateChanges)
	}
	// BuildPlanFingerprint is unchanged in this target lock, so the build
	// change must NOT be claimed: an artifact change alone does not prove the
	// build inputs changed.
	if kinds["build"] {
		t.Fatalf("an artifact change alone must not claim a build change: %+v", plan.UpdateChanges)
	}
	if len(plan.RegenerationRequired) == 0 {
		t.Fatalf("an update changes build inputs, so regeneration must be named")
	}
}

// TestUpdateWithNoChangeIsRefused checks an identical lock is not an update.
func TestUpdateWithNoChangeIsRefused(t *testing.T) {
	defs := chainDefinitions(t)
	env := chainEnvironment(t)
	lock, err := plugin.BuildLock(plugin.Catalog{Plugins: defs}, []environment.Environment{env})
	if err != nil {
		t.Fatalf("BuildLock: %v", err)
	}
	identical := *cloneLock(lock)
	plan, err := DerivePlan(PlanInput{
		World:       world(defs, env),
		Environment: env.ID,
		Lock:        &lock,
		TargetLock:  &identical,
		Request:     PlanRequest{Action: ActionUpdate, PluginID: "b"},
	})
	if err != nil {
		t.Fatalf("DerivePlan: %v", err)
	}
	if !plan.Refused || !containsString(plan.Reasons, "no-change") {
		t.Fatalf("an identical lock must be refused as no-change: %+v", plan)
	}
}

// TestPlanIsDeterministic checks the same input yields the same plan and digest,
// which is what lets an applying operation reject a tampered plan.
func TestPlanIsDeterministic(t *testing.T) {
	defs := chainDefinitions(t)
	env := chainEnvironment(t)
	first, err := DerivePlan(PlanInput{World: world(defs, env), Environment: env.ID,
		Request: PlanRequest{Action: ActionDisable, PluginID: "b"}})
	if err != nil {
		t.Fatalf("DerivePlan: %v", err)
	}
	second, err := DerivePlan(PlanInput{World: world(defs, env), Environment: env.ID,
		Request: PlanRequest{Action: ActionDisable, PluginID: "b"}})
	if err != nil {
		t.Fatalf("DerivePlan: %v", err)
	}
	if first.Digest != second.Digest {
		t.Fatalf("the same request must yield the same digest: %s vs %s", first.Digest, second.Digest)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) {
		t.Fatalf("the same request must yield the same plan:\n%s\n%s", a, b)
	}
}

// TestPlanDigestChangesWithContent checks a tampered plan has a different
// digest, so the applying operation can refuse it.
func TestPlanDigestChangesWithContent(t *testing.T) {
	plan, err := DerivePlan(PlanInput{World: world(chainDefinitions(t), chainEnvironment(t)),
		Environment: cid(t, "development"),
		Request:     PlanRequest{Action: ActionDisable, PluginID: "b"}})
	if err != nil {
		t.Fatalf("DerivePlan: %v", err)
	}
	tampered := plan
	tampered.Candidates = append(tampered.Candidates, "c")
	if got := planDigest(tampered); got == plan.Digest {
		t.Fatalf("a changed candidate set must change the digest")
	}
}

// TestPlanCarriesTheSnapshot checks the plan records what it was made from.
func TestPlanCarriesTheSnapshot(t *testing.T) {
	env := chainEnvironment(t)
	plan, err := DerivePlan(PlanInput{World: world(chainDefinitions(t), env), Environment: env.ID,
		Request: PlanRequest{Action: ActionDisable, PluginID: "b"}})
	if err != nil {
		t.Fatalf("DerivePlan: %v", err)
	}
	if plan.Snapshot.DeclarationFingerprint == "" || plan.Snapshot.EnvironmentFingerprint == "" {
		t.Fatalf("the snapshot must fingerprint the declaration and the environment: %+v", plan.Snapshot)
	}
	if plan.Snapshot.SettingsVersion != nil {
		t.Fatalf("settingsVersion must stay nil while no settings source exists")
	}
	if plan.Snapshot.ScanState != ScanComplete {
		t.Fatalf("scan state = %q", plan.Snapshot.ScanState)
	}
	if plan.Snapshot.LockFingerprint != "" {
		t.Fatalf("no lock was supplied, so no lock fingerprint must be claimed")
	}
}

// TestPlanDoesNotGrantPermission keeps the authority limit visible.
func TestPlanDoesNotGrantPermission(t *testing.T) {
	plan, err := DerivePlan(PlanInput{World: world(chainDefinitions(t), chainEnvironment(t)),
		Environment: cid(t, "development"),
		Request:     PlanRequest{Action: ActionDisable, PluginID: "b"}})
	if err != nil {
		t.Fatalf("DerivePlan: %v", err)
	}
	if len(plan.AuthorityLimits) == 0 {
		t.Fatalf("a plan must name what it does not guarantee")
	}
	joined := strings.Join(plan.AuthorityLimits, " ")
	if !strings.Contains(joined, "grants no permission") {
		t.Fatalf("the plan must state that it grants no permission: %q", joined)
	}
}

// TestOptionalDependencyIsNotModelled pins PM2-02: TEI has no optional
// dependency, so the plan must not pretend one was preserved.
func TestOptionalDependencyIsNotModelled(t *testing.T) {
	plan, err := DerivePlan(PlanInput{World: world(chainDefinitions(t), chainEnvironment(t)),
		Environment: cid(t, "development"),
		Request:     PlanRequest{Action: ActionDisable, PluginID: "b"}})
	if err != nil {
		t.Fatalf("DerivePlan: %v", err)
	}
	if !plan.OptionalDependencyNotModelled {
		t.Fatalf("the plan must say optional dependencies are not modelled")
	}
	if len(plan.OptionalFeaturesLost) != 0 {
		t.Fatalf("no optional feature can be lost when none is declared: %+v", plan.OptionalFeaturesLost)
	}
}

// TestSimplestPlanActionsAreClosed keeps the action list honest.
func TestSimplestPlanActionsAreClosed(t *testing.T) {
	if len(PlanActions) != 5 {
		t.Fatalf("PlanActions = %+v", PlanActions)
	}
	for _, action := range PlanActions {
		env := chainEnvironment(t)
		if _, err := DerivePlan(PlanInput{World: world(chainDefinitions(t), env), Environment: env.ID,
			Request: PlanRequest{Action: action, PluginID: "a"}}); err != nil {
			var unsupported *UnsupportedPlanError
			if asUnsupported(err, &unsupported) {
				t.Errorf("action %q must be supported", action)
			}
			t.Fatalf("DerivePlan(%s): %v", action, err)
		}
	}
}

// cloneLock copies a lock deep enough to modify one entry.
func cloneLock(lock plugin.Lock) *plugin.Lock {
	out := plugin.Lock{FormatVersion: lock.FormatVersion}
	for _, el := range lock.Environments {
		plugins := append([]plugin.LockedPlugin(nil), el.Plugins...)
		out.Environments = append(out.Environments, plugin.EnvironmentLock{Environment: el.Environment, Plugins: plugins})
	}
	return &out
}

var _ = core.CanonicalID("")
