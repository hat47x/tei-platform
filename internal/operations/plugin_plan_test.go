package operations

import (
	"bytes"
	"encoding/json"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
)

func pcid(t *testing.T, value string) core.CanonicalID {
	t.Helper()
	return core.MustCanonicalID(value)
}

func pbuiltin(t *testing.T, id string, requires ...plugin.Requirement) plugin.Definition {
	t.Helper()
	return plugin.Definition{
		ID:       pcid(t, id),
		Version:  "1.0.0",
		Runtime:  plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Requires: requires,
	}
}

func preq(t *testing.T, id string, major int) plugin.Requirement {
	t.Helper()
	return plugin.Requirement{Plugin: pcid(t, id), CompatibleMajor: major}
}

func penv(t *testing.T, id string, plugins ...string) environment.Environment {
	t.Helper()
	env := environment.Environment{ID: pcid(t, id)}
	for _, p := range plugins {
		env.Plugins = append(env.Plugins, environment.ExtensionRequirement{ID: pcid(t, p), CompatibleMajor: 1})
	}
	return env
}

func pdbProvider(t *testing.T, id string, provider environment.DatabaseProvider) plugin.Capability {
	t.Helper()
	return plugin.Capability{
		Kind:              plugin.CapabilityDatabaseProvider,
		ID:                pcid(t, id),
		DatabaseProviders: []environment.DatabaseProvider{provider},
	}
}

func codesOf(plan Plan) map[string]bool {
	got := map[string]bool{}
	for _, r := range plan.Reasons {
		got[r.Code] = true
	}
	for _, c := range plan.Conflicts {
		got[c.Code] = true
	}
	for _, code := range VerifyPlan(plan) {
		got[code] = true
	}
	return got
}

// PM2-01: A requires B, B requires C. Disabling B stops A with it; C and any
// unrelated plugin stay unchanged; confirmation is required; nothing is applied.
func TestPluginPlanPM201RequiredDependencyClosure(t *testing.T) {
	a := pbuiltin(t, "a", preq(t, "b", 1))
	b := pbuiltin(t, "b", preq(t, "c", 1))
	c := pbuiltin(t, "c")
	env := penv(t, "development", "a", "b", "c")
	plan, err := PluginPlan(PluginPlanInput{
		Plugins:      []plugin.Definition{a, b, c},
		Environments: []environment.Environment{env},
		Environment:  pcid(t, "development"),
		Action:       PlanActionDisable,
		Plugin:       pcid(t, "b"),
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	if got := plan.Candidates; len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("candidates=%v, want [a b]", got)
	}
	if got := plan.Additional; len(got) != 1 || got[0] != "a" {
		t.Fatalf("additional=%v, want [a]", got)
	}
	if got := plan.Unchanged; len(got) != 1 || got[0] != "c" {
		t.Fatalf("unchanged=%v, want [c]", got)
	}
	if !plan.ConfirmationRequired {
		t.Fatal("confirmationRequired must be true when a dependent is added")
	}
	if !plan.SourcesPreserved || !plan.DataPreserved {
		t.Fatalf("disable must preserve sources and data: sources=%t data=%t", plan.SourcesPreserved, plan.DataPreserved)
	}
	if plan.SelectionChanged {
		t.Fatal("selectionChanged must always be false")
	}
	if !hasPlanReason(plan, PlanReasonDependentMustBeStopped) {
		t.Fatalf("a dependent must be named as a reason: %+v", plan.Reasons)
	}
	// Nothing was applied: the candidates are a proposal, not a state change.
	if plan.Digest == "" {
		t.Fatal("a plan must carry a digest")
	}
}

// PM2-02: TEI has no optional-dependency declaration, so the plan flags it as
// not modelled, never adds an optional dependent to the candidates, and never
// claims an optional feature was lost.
func TestPluginPlanPM202OptionalDependencyNotModelled(t *testing.T) {
	b := pbuiltin(t, "b")
	// x is active and shares no required edge with b: it is the shape an
	// optional dependent would take, and it must not become a candidate.
	x := plugin.Definition{ID: pcid(t, "x"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Provides: []plugin.Capability{pdbProvider(t, "feature.x", environment.ProviderPostgres)}}
	env := penv(t, "development", "b", "x")
	plan, err := PluginPlan(PluginPlanInput{
		Plugins:      []plugin.Definition{b, x},
		Environments: []environment.Environment{env},
		Environment:  pcid(t, "development"),
		Action:       PlanActionDisable,
		Plugin:       pcid(t, "b"),
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	if !plan.OptionalDependencyNotModelled {
		t.Fatal("optionalDependencyNotModelled must be true")
	}
	if len(plan.OptionalFeaturesLost) != 0 {
		t.Fatalf("optionalFeaturesLost must be empty, got %v", plan.OptionalFeaturesLost)
	}
	if got := plan.Candidates; len(got) != 1 || got[0] != "b" {
		t.Fatalf("candidates=%v, want [b] only (no optional dependent)", got)
	}
	if len(plan.Additional) != 0 {
		t.Fatalf("additional must be empty, got %v", plan.Additional)
	}
	if got := plan.Unchanged; len(got) != 1 || got[0] != "x" {
		t.Fatalf("unchanged=%v, want [x]", got)
	}
	if plan.ConfirmationRequired {
		t.Fatal("no dependent means no confirmation is required")
	}
}

// A plugin the Workspace does not declare is a missing target, not a refusal to
// touch the others.
func TestPluginPlanMissingTargetPlugin(t *testing.T) {
	b := pbuiltin(t, "b")
	env := penv(t, "development", "b")
	plan, err := PluginPlan(PluginPlanInput{
		Plugins:      []plugin.Definition{b},
		Environments: []environment.Environment{env},
		Environment:  pcid(t, "development"),
		Action:       PlanActionDisable,
		Plugin:       pcid(t, "absent"),
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	if len(plan.Candidates) != 0 {
		t.Fatalf("an undeclared target must produce no candidates, got %v", plan.Candidates)
	}
	if !hasPlanReason(plan, PlanReasonPluginNotDeclared) {
		t.Fatalf("want plugin-not-declared, got %+v", plan.Reasons)
	}
}

// PM2-04: a required dependency that is not declared at all is unresolved, and
// the plan refuses without adding other plugins to the candidate set.
func TestPluginPlanUnresolvedDependency(t *testing.T) {
	b := pbuiltin(t, "b", preq(t, "c", 1)) // c is never declared
	env := penv(t, "development", "b")
	plan, err := PluginPlan(PluginPlanInput{
		Plugins:      []plugin.Definition{b},
		Environments: []environment.Environment{env},
		Environment:  pcid(t, "development"),
		Action:       PlanActionRemove,
		Plugin:       pcid(t, "b"),
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	if len(plan.Candidates) != 0 {
		t.Fatalf("an unresolved dependency must refuse, got candidates %v", plan.Candidates)
	}
	if len(plan.Unresolved) != 1 || plan.Unresolved[0] != "b:c" {
		t.Fatalf("unresolved=%v, want [b:c]", plan.Unresolved)
	}
	if !hasPlanReason(plan, PlanReasonDependencyUnresolved) {
		t.Fatalf("want dependency-unresolved, got %+v", plan.Reasons)
	}
}

// A version mismatch is a version reason carrying its PLUGIN_RESOLVE_* code,
// never a conflict or an unresolved-declaration reason.
func TestPluginPlanVersionMismatchIsAVersionReason(t *testing.T) {
	a := pbuiltin(t, "a", preq(t, "b", 2)) // demands major 2
	b := pbuiltin(t, "b")                  // only 1.0.0 exists
	env := penv(t, "development", "a", "b")
	plan, err := PluginPlan(PluginPlanInput{
		Plugins:      []plugin.Definition{a, b},
		Environments: []environment.Environment{env},
		Environment:  pcid(t, "development"),
		Action:       PlanActionDisable,
		Plugin:       pcid(t, "a"),
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	reason, ok := findPlanReason(plan, PlanReasonVersionUnsupported)
	if !ok {
		t.Fatalf("want version-unsupported, got %+v", plan.Reasons)
	}
	if reason.ResolverCode != plugin.DiagnosticNoCompatiblePlugin {
		t.Fatalf("version-unsupported must map to %q, got %q", plugin.DiagnosticNoCompatiblePlugin, reason.ResolverCode)
	}
	if hasPlanReason(plan, PlanReasonContractIncompatible) || hasPlanReason(plan, PlanReasonDependencyUnresolved) {
		t.Fatalf("a version mismatch must not be a conflict or unresolved reason: %+v", plan.Reasons)
	}
	if len(plan.Candidates) != 0 {
		t.Fatalf("a version mismatch must refuse, got candidates %v", plan.Candidates)
	}
}

// PM2-09: a cycle is reported in cycles, mapped to PLUGIN_RESOLVE_DEPENDENCY_CYCLE,
// and produces no candidates.
func TestPluginPlanDependencyCycle(t *testing.T) {
	a := pbuiltin(t, "a", preq(t, "b", 1))
	b := pbuiltin(t, "b", preq(t, "a", 1))
	env := penv(t, "development", "a", "b")
	plan, err := PluginPlan(PluginPlanInput{
		Plugins:      []plugin.Definition{a, b},
		Environments: []environment.Environment{env},
		Environment:  pcid(t, "development"),
		Action:       PlanActionDisable,
		Plugin:       pcid(t, "a"),
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	if len(plan.Cycles) == 0 {
		t.Fatalf("a cycle must be reported, got %+v", plan.Cycles)
	}
	reason, ok := findPlanReason(plan, PlanReasonDependencyCycle)
	if !ok || reason.ResolverCode != plugin.DiagnosticDependencyCycle {
		t.Fatalf("want dependency-cycle mapped to %q, got %+v", plugin.DiagnosticDependencyCycle, plan.Reasons)
	}
	if len(plan.Candidates) != 0 {
		t.Fatalf("a cycle must refuse, got candidates %v", plan.Candidates)
	}
}

// A declared plugin the environment does not select cannot be disabled.
func TestPluginPlanEnvNotSelected(t *testing.T) {
	b := pbuiltin(t, "b")
	unrelated := pbuiltin(t, "unrelated")
	env := penv(t, "development", "unrelated")
	plan, err := PluginPlan(PluginPlanInput{
		Plugins:      []plugin.Definition{b, unrelated},
		Environments: []environment.Environment{env},
		Environment:  pcid(t, "development"),
		Action:       PlanActionDisable,
		Plugin:       pcid(t, "b"),
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	if !hasPlanReason(plan, PlanReasonEnvNotSelected) {
		t.Fatalf("want env-not-selected, got %+v", plan.Reasons)
	}
	if len(plan.Candidates) != 0 {
		t.Fatalf("an inactive target must produce no candidates, got %v", plan.Candidates)
	}
}

// delete-data and convert-data are irreversible: data is not preserved and
// confirmation is required.
func TestPluginPlanDataChangeIsIrreversible(t *testing.T) {
	b := pbuiltin(t, "b")
	env := penv(t, "development", "b")
	for _, action := range []string{PlanActionDeleteData, PlanActionConvertData} {
		plan, err := PluginPlan(PluginPlanInput{
			Plugins:      []plugin.Definition{b},
			Environments: []environment.Environment{env},
			Environment:  pcid(t, "development"),
			Action:       action,
			Plugin:       pcid(t, "b"),
		})
		if err != nil {
			t.Fatalf("%s: PluginPlan: %v", action, err)
		}
		if plan.DataPreserved {
			t.Fatalf("%s must not report data preserved", action)
		}
		if !plan.SourcesPreserved {
			t.Fatalf("%s must still preserve sources", action)
		}
		if !plan.ConfirmationRequired {
			t.Fatalf("%s must require its own confirmation", action)
		}
		if !hasPlanReason(plan, PlanReasonIrreversibleOperation) {
			t.Fatalf("%s must name the irreversibility, got %+v", action, plan.Reasons)
		}
	}
}

func updateLock(t *testing.T, env string, entries ...plugin.LockedPlugin) plugin.Lock {
	t.Helper()
	return plugin.Lock{
		FormatVersion: plugin.LockFormatVersion,
		Environments:  []plugin.EnvironmentLock{{Environment: pcid(t, env), Plugins: entries}},
	}
}

// update separates every change kind, requires a restart for a static Go binary
// and regeneration only when build inputs moved.
func TestPluginPlanUpdateDiff(t *testing.T) {
	p1 := pbuiltin(t, "p", preq(t, "q", 1))
	p1.Provides = []plugin.Capability{pdbProvider(t, "database.x", environment.ProviderPostgres)}
	p1.Artifact = plugin.Artifact{Digest: "sha256:" + repeat("a", 64)}
	p2 := plugin.Definition{ID: pcid(t, "p"), Version: "2.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Requires: []plugin.Requirement{preq(t, "q", 1), preq(t, "r", 1)}}
	p2.Provides = []plugin.Capability{pdbProvider(t, "database.x", environment.ProviderPostgres), pdbProvider(t, "database.y", environment.ProviderOracle)}
	p2.Artifact = plugin.Artifact{Digest: "sha256:" + repeat("b", 64)}
	q := pbuiltin(t, "q")
	r := pbuiltin(t, "r")
	env := penv(t, "development", "p", "q", "r")

	curLock := updateLock(t, "development", plugin.LockedPlugin{
		ID: pcid(t, "p"), Version: "1.0.0", Fingerprint: plugin.DefinitionFingerprint(p1),
		ArtifactDigest: "sha256:" + repeat("a", 64), BuildPlanFingerprint: "bp1",
	})
	tgtLock := updateLock(t, "development", plugin.LockedPlugin{
		ID: pcid(t, "p"), Version: "2.0.0", Fingerprint: plugin.DefinitionFingerprint(p2),
		ArtifactDigest: "sha256:" + repeat("b", 64), BuildPlanFingerprint: "bp2",
	})
	plan, err := PluginPlan(PluginPlanInput{
		Plugins:      []plugin.Definition{p1, p2, q, r},
		Environments: []environment.Environment{env},
		Environment:  pcid(t, "development"),
		Action:       PlanActionUpdate,
		Plugin:       pcid(t, "p"),
		Lock:         &curLock,
		TargetLock:   &tgtLock,
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	if plan.Diff == nil {
		t.Fatal("an update plan must carry a diff")
	}
	d := plan.Diff
	if !d.VersionChanged || !d.ContractChanged || !d.ProviderChanged || !d.ArtifactChanged || !d.BuildPlanChanged || !d.DependencyChanged {
		t.Fatalf("every change kind must be separated: %+v", d)
	}
	if len(plan.RestartRequired) != 1 || plan.RestartRequired[0] != "p" {
		t.Fatalf("a builtin plugin update must require a restart, got %v", plan.RestartRequired)
	}
	if len(plan.RegenerationRequired) != 1 || plan.RegenerationRequired[0] != "p" {
		t.Fatalf("changed build inputs must require regeneration, got %v", plan.RegenerationRequired)
	}
	if hasPlanReason(plan, PlanReasonContractMismatch) {
		t.Fatalf("matching fingerprints must not be a contract mismatch: %+v", plan.Reasons)
	}
}

// A build-input-only change (same version, same contract) requires regeneration
// but is the narrowest possible update.
func TestPluginPlanUpdateRegenerationOnly(t *testing.T) {
	p1 := pbuiltin(t, "p")
	env := penv(t, "development", "p")
	fp := plugin.DefinitionFingerprint(p1)
	curLock := updateLock(t, "development", plugin.LockedPlugin{ID: pcid(t, "p"), Version: "1.0.0", Fingerprint: fp, ArtifactDigest: "sha256:" + repeat("a", 64), BuildPlanFingerprint: "bp1"})
	tgtLock := updateLock(t, "development", plugin.LockedPlugin{ID: pcid(t, "p"), Version: "1.0.0", Fingerprint: fp, ArtifactDigest: "sha256:" + repeat("a", 64), BuildPlanFingerprint: "bp2"})
	plan, err := PluginPlan(PluginPlanInput{
		Plugins:      []plugin.Definition{p1},
		Environments: []environment.Environment{env},
		Environment:  pcid(t, "development"),
		Action:       PlanActionUpdate,
		Plugin:       pcid(t, "p"),
		Lock:         &curLock,
		TargetLock:   &tgtLock,
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	if plan.Diff == nil || !plan.Diff.BuildPlanChanged || plan.Diff.VersionChanged || plan.Diff.ContractChanged {
		t.Fatalf("only the build plan changed: %+v", plan.Diff)
	}
	if len(plan.RegenerationRequired) != 1 {
		t.Fatalf("regeneration required, got %v", plan.RegenerationRequired)
	}
}

// PM2-12: a lock whose contract fingerprint no longer matches the declaration it
// pins is a contract mismatch, and the update is not shown as compatible.
func TestPluginPlanUpdateContractMismatch(t *testing.T) {
	p1 := pbuiltin(t, "p")
	env := penv(t, "development", "p")
	stale := updateLock(t, "development", plugin.LockedPlugin{ID: pcid(t, "p"), Version: "1.0.0", Fingerprint: "diverged", ArtifactDigest: "sha256:" + repeat("a", 64), BuildPlanFingerprint: "bp1"})
	tgt := updateLock(t, "development", plugin.LockedPlugin{ID: pcid(t, "p"), Version: "1.0.0", Fingerprint: "diverged2", ArtifactDigest: "sha256:" + repeat("a", 64), BuildPlanFingerprint: "bp2"})
	plan, err := PluginPlan(PluginPlanInput{
		Plugins:      []plugin.Definition{p1},
		Environments: []environment.Environment{env},
		Environment:  pcid(t, "development"),
		Action:       PlanActionUpdate,
		Plugin:       pcid(t, "p"),
		Lock:         &stale,
		TargetLock:   &tgt,
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	if !hasPlanReason(plan, PlanReasonContractMismatch) {
		t.Fatalf("want contract-mismatch, got %+v", plan.Reasons)
	}
}

// update without a target lock names the missing input rather than inventing a
// diff.
func TestPluginPlanUpdateWithoutTargetLock(t *testing.T) {
	p1 := pbuiltin(t, "p")
	env := penv(t, "development", "p")
	cur := updateLock(t, "development", plugin.LockedPlugin{ID: pcid(t, "p"), Version: "1.0.0", Fingerprint: plugin.DefinitionFingerprint(p1)})
	plan, err := PluginPlan(PluginPlanInput{
		Plugins:      []plugin.Definition{p1},
		Environments: []environment.Environment{env},
		Environment:  pcid(t, "development"),
		Action:       PlanActionUpdate,
		Plugin:       pcid(t, "p"),
		Lock:         &cur,
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	if plan.Diff != nil {
		t.Fatalf("no target lock must yield no diff, got %+v", plan.Diff)
	}
	if !hasPlanReason(plan, PlanReasonLockNotProvided) {
		t.Fatalf("want lock-not-provided, got %+v", plan.Reasons)
	}
}

// A lock that describes another environment is a source-fingerprint mismatch.
func TestPluginPlanUpdateLockForAnotherEnvironment(t *testing.T) {
	p1 := pbuiltin(t, "p")
	env := penv(t, "development", "p")
	other := updateLock(t, "production", plugin.LockedPlugin{ID: pcid(t, "p"), Version: "1.0.0", Fingerprint: plugin.DefinitionFingerprint(p1)})
	tgt := updateLock(t, "development", plugin.LockedPlugin{ID: pcid(t, "p"), Version: "1.0.0", Fingerprint: plugin.DefinitionFingerprint(p1)})
	plan, err := PluginPlan(PluginPlanInput{
		Plugins:      []plugin.Definition{p1},
		Environments: []environment.Environment{env},
		Environment:  pcid(t, "development"),
		Action:       PlanActionUpdate,
		Plugin:       pcid(t, "p"),
		Lock:         &other,
		TargetLock:   &tgt,
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	if !hasPlanReason(plan, PlanReasonSourceFingerprintMismatch) {
		t.Fatalf("want source-fingerprint-mismatch, got %+v", plan.Reasons)
	}
}

// Two providers of one contract are a conflict, named distinctly from a
// duplicate declaration.
func TestPluginPlanMultipleProvidersConflict(t *testing.T) {
	one := plugin.Definition{ID: pcid(t, "one"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Provides: []plugin.Capability{pdbProvider(t, "database.x", environment.ProviderPostgres)}}
	two := plugin.Definition{ID: pcid(t, "two"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Provides: []plugin.Capability{pdbProvider(t, "database.x", environment.ProviderPostgres)}}
	env := penv(t, "development", "one", "two")
	plan, err := PluginPlan(PluginPlanInput{
		Plugins:      []plugin.Definition{one, two},
		Environments: []environment.Environment{env},
		Environment:  pcid(t, "development"),
		Action:       PlanActionDisable,
		Plugin:       pcid(t, "one"),
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	found := false
	for _, c := range plan.Conflicts {
		if c.Kind == ConflictMultipleProviders && c.Code == PlanReasonContractIncompatible {
			found = true
		}
	}
	if !found {
		t.Fatalf("want a multiple-providers conflict, got %+v", plan.Conflicts)
	}
}

// The digest covers the plan content: changing any candidate changes it.
func TestPluginPlanDigestChangesWithCandidates(t *testing.T) {
	a := pbuiltin(t, "a", preq(t, "b", 1))
	b := pbuiltin(t, "b")
	env := penv(t, "development", "a", "b")
	base, err := PluginPlan(PluginPlanInput{
		Plugins: []plugin.Definition{a, b}, Environments: []environment.Environment{env},
		Environment: pcid(t, "development"), Action: PlanActionDisable, Plugin: pcid(t, "b"),
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	// Same input yields a byte-identical digest.
	again, err := PluginPlan(PluginPlanInput{
		Plugins: []plugin.Definition{a, b}, Environments: []environment.Environment{env},
		Environment: pcid(t, "development"), Action: PlanActionDisable, Plugin: pcid(t, "b"),
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	if base.Digest != again.Digest {
		t.Fatalf("digest is not stable: %q vs %q", base.Digest, again.Digest)
	}
	// A plan whose candidate set differs must have a different digest.
	tampered := base
	tampered.Candidates = append([]string{"b"}, "a")
	if planDigest(tampered) == base.Digest {
		t.Fatal("changing a candidate must change the digest")
	}
}

// A plan whose content no longer matches its digest is rejected as tampered.
func TestPluginPlanTamperedIsRejected(t *testing.T) {
	b := pbuiltin(t, "b")
	env := penv(t, "development", "b")
	plan, err := PluginPlan(PluginPlanInput{
		Plugins: []plugin.Definition{b}, Environments: []environment.Environment{env},
		Environment: pcid(t, "development"), Action: PlanActionDisable, Plugin: pcid(t, "b"),
	})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	if got := VerifyPlan(plan); len(got) != 0 {
		t.Fatalf("a fresh plan must verify, got %v", got)
	}
	// Round-trip through JSON, then tamper a candidate while keeping the digest.
	raw, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	var loaded Plan
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatal(err)
	}
	if got := VerifyPlan(loaded); len(got) != 0 {
		t.Fatalf("a round-tripped plan must verify, got %v", got)
	}
	loaded.Candidates = []string{"b", "smuggled"}
	if got := VerifyPlan(loaded); len(got) != 1 || got[0] != PlanReasonPlanTampered {
		t.Fatalf("a tampered plan must be rejected with plan-tampered, got %v", got)
	}
}

// The plan never mutates its input: normalization inside
// plugin.DefinitionFingerprint sorts capability slices in place, so the planner
// must work on copies. Requires are declared out of order to catch a leak.
func TestPluginPlanDoesNotMutateInput(t *testing.T) {
	a := plugin.Definition{ID: pcid(t, "a"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Requires: []plugin.Requirement{preq(t, "z", 1), preq(t, "b", 1)}}
	z := pbuiltin(t, "z")
	b := pbuiltin(t, "b")
	plugins := []plugin.Definition{a, z, b}
	env := penv(t, "development", "a", "z", "b")
	before, err := json.Marshal(plugins)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PluginPlan(PluginPlanInput{
		Plugins: plugins, Environments: []environment.Environment{env},
		Environment: pcid(t, "development"), Action: PlanActionDisable, Plugin: pcid(t, "b"),
	}); err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	after, err := json.Marshal(plugins)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("input mutated:\nbefore=%s\nafter=%s", before, after)
	}
}

func TestPluginPlanRejectsUnknownActionAndEnvironment(t *testing.T) {
	b := pbuiltin(t, "b")
	env := penv(t, "development", "b")
	if _, err := PluginPlan(PluginPlanInput{
		Plugins: []plugin.Definition{b}, Environments: []environment.Environment{env},
		Environment: pcid(t, "development"), Action: "apply", Plugin: pcid(t, "b"),
	}); err == nil {
		t.Fatal("an action outside the closed set must be refused")
	}
	if _, err := PluginPlan(PluginPlanInput{
		Plugins: []plugin.Definition{b}, Environments: []environment.Environment{env},
		Environment: pcid(t, "nowhere"), Action: PlanActionDisable, Plugin: pcid(t, "b"),
	}); err == nil {
		t.Fatal("an unknown environment must be refused")
	}
}

// Every reason code the planner can emit is produced by one of the inputs
// above, so no declared word is dead.
func TestPluginPlanEveryReasonCodeIsProduced(t *testing.T) {
	got := map[string]bool{}
	collect := func(t *testing.T, in PluginPlanInput) {
		t.Helper()
		plan, err := PluginPlan(in)
		if err != nil {
			t.Fatalf("PluginPlan: %v", err)
		}
		for code := range codesOf(plan) {
			got[code] = true
		}
	}

	// dependent-must-be-stopped, plugin-not-declared, dependency-cycle,
	// dependency-unresolved, version-unsupported, env-not-selected,
	// irreversible-operation, contract-incompatible (multiple providers).
	abc := []plugin.Definition{pbuiltin(t, "a", preq(t, "b", 1)), pbuiltin(t, "b", preq(t, "c", 1)), pbuiltin(t, "c")}
	collect(t, PluginPlanInput{Plugins: abc, Environments: []environment.Environment{penv(t, "development", "a", "b", "c")}, Environment: pcid(t, "development"), Action: PlanActionDisable, Plugin: pcid(t, "b")})
	collect(t, PluginPlanInput{Plugins: abc, Environments: []environment.Environment{penv(t, "development", "a", "b", "c")}, Environment: pcid(t, "development"), Action: PlanActionDisable, Plugin: pcid(t, "absent")})
	// env-not-selected: a declared plugin the environment does not select.
	collect(t, PluginPlanInput{Plugins: []plugin.Definition{pbuiltin(t, "b"), pbuiltin(t, "unrelated")}, Environments: []environment.Environment{penv(t, "development", "unrelated")}, Environment: pcid(t, "development"), Action: PlanActionDisable, Plugin: pcid(t, "b")})
	collect(t, PluginPlanInput{Plugins: []plugin.Definition{pbuiltin(t, "b")}, Environments: []environment.Environment{penv(t, "development", "b")}, Environment: pcid(t, "development"), Action: PlanActionDisable, Plugin: pcid(t, "b")})
	cycle := []plugin.Definition{pbuiltin(t, "a", preq(t, "b", 1)), pbuiltin(t, "b", preq(t, "a", 1))}
	collect(t, PluginPlanInput{Plugins: cycle, Environments: []environment.Environment{penv(t, "development", "a", "b")}, Environment: pcid(t, "development"), Action: PlanActionDisable, Plugin: pcid(t, "a")})
	collect(t, PluginPlanInput{Plugins: []plugin.Definition{pbuiltin(t, "b", preq(t, "c", 1))}, Environments: []environment.Environment{penv(t, "development", "b")}, Environment: pcid(t, "development"), Action: PlanActionRemove, Plugin: pcid(t, "b")})
	collect(t, PluginPlanInput{Plugins: []plugin.Definition{pbuiltin(t, "a", preq(t, "b", 2)), pbuiltin(t, "b")}, Environments: []environment.Environment{penv(t, "development", "a", "b")}, Environment: pcid(t, "development"), Action: PlanActionDisable, Plugin: pcid(t, "a")})
	collect(t, PluginPlanInput{Plugins: []plugin.Definition{pbuiltin(t, "b")}, Environments: []environment.Environment{penv(t, "development", "b")}, Environment: pcid(t, "development"), Action: PlanActionDeleteData, Plugin: pcid(t, "b")})
	mp := []plugin.Definition{
		{ID: pcid(t, "one"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Provides: []plugin.Capability{pdbProvider(t, "database.x", environment.ProviderPostgres)}},
		{ID: pcid(t, "two"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Provides: []plugin.Capability{pdbProvider(t, "database.x", environment.ProviderPostgres)}},
	}
	collect(t, PluginPlanInput{Plugins: mp, Environments: []environment.Environment{penv(t, "development", "one", "two")}, Environment: pcid(t, "development"), Action: PlanActionDisable, Plugin: pcid(t, "one")})

	// contract-mismatch, lock-not-provided, source-fingerprint-mismatch.
	p1 := pbuiltin(t, "p")
	stale := updateLock(t, "development", plugin.LockedPlugin{ID: pcid(t, "p"), Version: "1.0.0", Fingerprint: "diverged"})
	staleTarget := updateLock(t, "development", plugin.LockedPlugin{ID: pcid(t, "p"), Version: "1.0.0", Fingerprint: "diverged2"})
	collect(t, PluginPlanInput{Plugins: []plugin.Definition{p1}, Environments: []environment.Environment{penv(t, "development", "p")}, Environment: pcid(t, "development"), Action: PlanActionUpdate, Plugin: pcid(t, "p"), Lock: &stale, TargetLock: &staleTarget})
	collect(t, PluginPlanInput{Plugins: []plugin.Definition{p1}, Environments: []environment.Environment{penv(t, "development", "p")}, Environment: pcid(t, "development"), Action: PlanActionUpdate, Plugin: pcid(t, "p")})
	other := updateLock(t, "production", plugin.LockedPlugin{ID: pcid(t, "p"), Version: "1.0.0", Fingerprint: plugin.DefinitionFingerprint(p1)})
	tgt := updateLock(t, "development", plugin.LockedPlugin{ID: pcid(t, "p"), Version: "1.0.0", Fingerprint: plugin.DefinitionFingerprint(p1)})
	collect(t, PluginPlanInput{Plugins: []plugin.Definition{p1}, Environments: []environment.Environment{penv(t, "development", "p")}, Environment: pcid(t, "development"), Action: PlanActionUpdate, Plugin: pcid(t, "p"), Lock: &other, TargetLock: &tgt})

	// plan-tampered: a fresh plan verifies, a tampered one does not.
	fresh, err := PluginPlan(PluginPlanInput{Plugins: []plugin.Definition{pbuiltin(t, "b")}, Environments: []environment.Environment{penv(t, "development", "b")}, Environment: pcid(t, "development"), Action: PlanActionDisable, Plugin: pcid(t, "b")})
	if err != nil {
		t.Fatalf("PluginPlan: %v", err)
	}
	fresh.Candidates = append(fresh.Candidates, "smuggled")
	for _, code := range VerifyPlan(fresh) {
		got[code] = true
	}

	for _, code := range []string{
		PlanReasonPluginNotDeclared, PlanReasonEnvNotSelected, PlanReasonVersionUnsupported,
		PlanReasonDependencyCycle, PlanReasonDependencyUnresolved, PlanReasonContractIncompatible,
		PlanReasonContractMismatch, PlanReasonLockNotProvided, PlanReasonSourceFingerprintMismatch,
		PlanReasonDependentMustBeStopped, PlanReasonIrreversibleOperation, PlanReasonPlanTampered,
	} {
		if !got[code] {
			t.Errorf("reason code %q is not produced by any input above", code)
		}
	}
}

func hasPlanReason(plan Plan, code string) bool {
	_, ok := findPlanReason(plan, code)
	return ok
}

func findPlanReason(plan Plan, code string) (PlanReason, bool) {
	for _, r := range plan.Reasons {
		if r.Code == code {
			return r, true
		}
	}
	return PlanReason{}, false
}

func repeat(s string, n int) string {
	var b bytes.Buffer
	for i := 0; i < n; i++ {
		b.WriteString(s)
	}
	return b.String()
}
