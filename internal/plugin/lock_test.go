package plugin_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/plugin"
)

func TestLockPinsVersionEvenWhenNewerCompatibleAppears(t *testing.T) {
	base := plugin.Definition{ID: core.MustCanonicalID("tei.x"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}}
	env := environment.Environment{ID: core.MustCanonicalID("prod"), Plugins: []environment.ExtensionRequirement{{ID: base.ID, CompatibleMajor: 1}}}
	catalog := plugin.Catalog{Plugins: []plugin.Definition{base}}
	lock, err := plugin.BuildLock(catalog, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	catalog.Plugins = append(catalog.Plugins, plugin.Definition{ID: base.ID, Version: "1.9.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}})
	r, err := catalog.ResolveEnvironmentWithLock(env, lock)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Plugins) != 1 || r.Plugins[0].Version != "1.0.0" {
		t.Fatalf("locked resolution: %#v", r.Plugins)
	}
}

// Two projects may select the same logical Plugin while adopting its releases
// at different times. A new catalog release must not modify either pinned lock.
func TestIndependentEnvironmentsAdoptSharedPluginOnSeparateLocks(t *testing.T) {
	id := core.MustCanonicalID("tei.reference.shared-derivation")
	v1 := plugin.Definition{
		ID:      id,
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Provides: []plugin.Capability{{
			Kind: plugin.CapabilityEditorIntelligence,
			ID:   core.MustCanonicalID("reference.derive"),
		}},
	}
	projectA := environment.Environment{
		ID:      core.MustCanonicalID("project.alpha"),
		Plugins: []environment.ExtensionRequirement{{ID: id, CompatibleMajor: 1}},
	}
	projectB := environment.Environment{
		ID:      core.MustCanonicalID("project.beta"),
		Plugins: []environment.ExtensionRequirement{{ID: id, CompatibleMajor: 1}},
	}
	catalog := plugin.Catalog{Plugins: []plugin.Definition{v1}}
	pinnedA, err := plugin.BuildLock(catalog, []environment.Environment{projectA})
	if err != nil {
		t.Fatal(err)
	}

	v11 := v1
	v11.Version = "1.1.0"
	catalog.Plugins = append(catalog.Plugins, v11)
	pinnedB, err := plugin.BuildLock(catalog, []environment.Environment{projectB})
	if err != nil {
		t.Fatal(err)
	}

	// Another compatible release must not silently upgrade either project.
	v12 := v1
	v12.Version = "1.2.0"
	catalog.Plugins = append(catalog.Plugins, v12)
	assertVersion := func(env environment.Environment, lock plugin.Lock, want string) {
		t.Helper()
		resolved, err := catalog.ResolveEnvironmentWithLock(env, lock)
		if err != nil {
			t.Fatal(err)
		}
		if len(resolved.Plugins) != 1 || resolved.Plugins[0].Version != want {
			t.Fatalf("%s: resolved %#v, want %s", env.ID, resolved.Plugins, want)
		}
		if _, ok := resolved.Capability(plugin.CapabilityEditorIntelligence, core.MustCanonicalID("reference.derive")); !ok {
			t.Fatalf("%s: shared capability was not resolved", env.ID)
		}
	}
	assertVersion(projectA, pinnedA, "1.0.0")
	assertVersion(projectB, pinnedB, "1.1.0")

	// Project A may adopt the new version and then explicitly roll back.
	upgradedA, err := plugin.BuildLock(catalog, []environment.Environment{projectA})
	if err != nil {
		t.Fatal(err)
	}
	assertVersion(projectA, upgradedA, "1.2.0")
	assertVersion(projectA, pinnedA, "1.0.0")
	assertVersion(projectB, pinnedB, "1.1.0")

	// Each lock is scoped to its own environment, not a repository-wide default.
	if _, err := catalog.ResolveEnvironmentWithLock(projectA, pinnedB); err == nil {
		t.Fatal("another project's lock must not authorize this environment")
	}
	if _, err := catalog.ResolveEnvironmentWithLock(projectB, pinnedA); err == nil {
		t.Fatal("another project's lock must not authorize this environment")
	}

	// Mutation of a pinned definition invalidates only the affected project.
	mutated := plugin.Catalog{Plugins: append([]plugin.Definition(nil), catalog.Plugins...)}
	mutated.Plugins[0].Runtime = plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "changed"}
	if _, err := mutated.ResolveEnvironmentWithLock(projectA, pinnedA); err == nil {
		t.Fatal("changed pinned definition must be rejected")
	}
	if _, err := mutated.ResolveEnvironmentWithLock(projectB, pinnedB); err != nil {
		t.Fatalf("unrelated project's pinned release must remain usable: %v", err)
	}
}

func TestLockDetectsDescriptorMutation(t *testing.T) {
	base := plugin.Definition{ID: core.MustCanonicalID("tei.x"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}}
	env := environment.Environment{ID: core.MustCanonicalID("prod"), Plugins: []environment.ExtensionRequirement{{ID: base.ID, CompatibleMajor: 1}}}
	catalog := plugin.Catalog{Plugins: []plugin.Definition{base}}
	lock, err := plugin.BuildLock(catalog, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	catalog.Plugins[0].Runtime = plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "changed"}
	if _, err := catalog.ResolveEnvironmentWithLock(env, lock); err == nil {
		t.Fatal("expected fingerprint divergence")
	}
}

func TestDefinitionFingerprintIgnoresCapabilityOrdering(t *testing.T) {
	a := plugin.Definition{ID: core.MustCanonicalID("tei.x"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Provides: []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: core.MustCanonicalID("runtime.x"), Actions: []deployment.ActionKind{deployment.ActionTrafficShift, deployment.ActionApplicationStart}}}}
	b := a
	b.Provides = []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: core.MustCanonicalID("runtime.x"), Actions: []deployment.ActionKind{deployment.ActionApplicationStart, deployment.ActionTrafficShift}}}
	if plugin.DefinitionFingerprint(a) != plugin.DefinitionFingerprint(b) {
		t.Fatal("semantic reordering changed fingerprint")
	}
}

func TestEnvironmentLockFingerprintIgnoresPluginOrdering(t *testing.T) {
	l1 := plugin.Lock{FormatVersion: plugin.LockFormatVersion, Environments: []plugin.EnvironmentLock{{Environment: "prod", Plugins: []plugin.LockedPlugin{{ID: "tei.a", Version: "1.0.0", Fingerprint: "a"}, {ID: "tei.b", Version: "1.0.0", Fingerprint: "b"}}}}}
	l2 := plugin.Lock{FormatVersion: plugin.LockFormatVersion, Environments: []plugin.EnvironmentLock{{Environment: "prod", Plugins: []plugin.LockedPlugin{{ID: "tei.b", Version: "1.0.0", Fingerprint: "b"}, {ID: "tei.a", Version: "1.0.0", Fingerprint: "a"}}}}}
	a, err := plugin.EnvironmentLockFingerprint(l1, "prod")
	if err != nil {
		t.Fatal(err)
	}
	b, err := plugin.EnvironmentLockFingerprint(l2, "prod")
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("ordering changed environment lock fingerprint: %s != %s", a, b)
	}
}

func TestLegacyFormat2PluginLockRemainsReadable(t *testing.T) {
	def := plugin.Definition{
		ID:      core.MustCanonicalID("tei.legacy"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
	}
	env := environment.Environment{
		ID: core.MustCanonicalID("legacy"),
		Plugins: []environment.ExtensionRequirement{{
			ID:              def.ID,
			CompatibleMajor: 1,
		}},
	}
	legacy := plugin.Lock{
		FormatVersion: 2,
		Environments: []plugin.EnvironmentLock{{
			Environment: env.ID,
			Plugins: []plugin.LockedPlugin{{
				ID:          def.ID,
				Version:     def.Version,
				Fingerprint: plugin.DefinitionFingerprint(def),
			}},
		}},
	}
	if _, err := plugin.EnvironmentLockFingerprint(legacy, env.ID); err != nil {
		t.Fatal(err)
	}
	resolved, err := (plugin.Catalog{Plugins: []plugin.Definition{def}}).ResolveEnvironmentWithLock(env, legacy)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Plugins) != 1 || resolved.Plugins[0].ID != def.ID {
		t.Fatalf("resolved legacy lock: %#v", resolved.Plugins)
	}
}

func TestBuildTraceFormat3PluginLockIsReadable(t *testing.T) {
	def := plugin.Definition{
		ID:      core.MustCanonicalID("tei.traced"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "traced"},
		Artifact: plugin.Artifact{
			Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	}
	env := environment.Environment{
		ID: core.MustCanonicalID("traced"),
		Plugins: []environment.ExtensionRequirement{{
			ID:              def.ID,
			CompatibleMajor: 1,
		}},
	}
	lock := plugin.Lock{
		FormatVersion: plugin.BuildTraceLockFormatVersion,
		Environments: []plugin.EnvironmentLock{{
			Environment: env.ID,
			Plugins: []plugin.LockedPlugin{{
				ID:                   def.ID,
				Version:              def.Version,
				Fingerprint:          plugin.DefinitionFingerprint(def),
				ArtifactDigest:       def.Artifact.Digest,
				BuildPlanFingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			}},
		}},
	}
	if _, err := plugin.EnvironmentLockFingerprint(lock, env.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := (plugin.Catalog{Plugins: []plugin.Definition{def}}).ResolveEnvironmentWithLock(env, lock); err != nil {
		t.Fatal(err)
	}
}

func TestUnsupportedPluginLockFormatFails(t *testing.T) {
	lock := plugin.Lock{FormatVersion: 99}
	if _, err := plugin.EnvironmentLockFingerprint(lock, core.MustCanonicalID("x")); err == nil {
		t.Fatal("expected unsupported lock format to fail")
	}
	env := environment.Environment{ID: core.MustCanonicalID("x")}
	if _, err := (plugin.Catalog{}).ResolveEnvironmentWithLock(env, lock); err == nil {
		t.Fatal("expected unsupported lock format to fail")
	}
}

func TestBuildTraceFormat3RejectsInvalidBuildPlanFingerprint(t *testing.T) {
	def := plugin.Definition{
		ID:      core.MustCanonicalID("tei.invalid_trace"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "invalid-trace"},
		Artifact: plugin.Artifact{
			Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	}
	env := environment.Environment{
		ID: core.MustCanonicalID("invalid_trace"),
		Plugins: []environment.ExtensionRequirement{{
			ID:              def.ID,
			CompatibleMajor: 1,
		}},
	}
	lock := plugin.Lock{
		FormatVersion: plugin.BuildTraceLockFormatVersion,
		Environments: []plugin.EnvironmentLock{{
			Environment: env.ID,
			Plugins: []plugin.LockedPlugin{{
				ID:                   def.ID,
				Version:              def.Version,
				Fingerprint:          plugin.DefinitionFingerprint(def),
				ArtifactDigest:       def.Artifact.Digest,
				BuildPlanFingerprint: "not-a-fingerprint",
			}},
		}},
	}
	if _, err := plugin.EnvironmentLockFingerprint(lock, env.ID); err == nil {
		t.Fatal("expected invalid build plan fingerprint to fail")
	}
	if _, err := (plugin.Catalog{Plugins: []plugin.Definition{def}}).ResolveEnvironmentWithLock(env, lock); err == nil {
		t.Fatal("expected invalid build plan fingerprint to fail")
	}
}

func TestLegacyFormat2RejectsBuildTraceField(t *testing.T) {
	def := plugin.Definition{
		ID:      core.MustCanonicalID("tei.legacy_trace"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "legacy-trace"},
		Artifact: plugin.Artifact{
			Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	}
	env := environment.Environment{
		ID: core.MustCanonicalID("legacy_trace"),
		Plugins: []environment.ExtensionRequirement{{
			ID:              def.ID,
			CompatibleMajor: 1,
		}},
	}
	lock := plugin.Lock{
		FormatVersion: 2,
		Environments: []plugin.EnvironmentLock{{
			Environment: env.ID,
			Plugins: []plugin.LockedPlugin{{
				ID:                   def.ID,
				Version:              def.Version,
				Fingerprint:          plugin.DefinitionFingerprint(def),
				ArtifactDigest:       def.Artifact.Digest,
				BuildPlanFingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			}},
		}},
	}
	if _, err := plugin.EnvironmentLockFingerprint(lock, env.ID); err == nil {
		t.Fatal("expected format 2 build trace field to fail")
	}
	if _, err := (plugin.Catalog{Plugins: []plugin.Definition{def}}).ResolveEnvironmentWithLock(env, lock); err == nil {
		t.Fatal("expected format 2 build trace field to fail")
	}
}

func TestBuildTraceFormat3RequiresArtifactDigest(t *testing.T) {
	def := plugin.Definition{
		ID:      core.MustCanonicalID("tei.trace_without_artifact"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
	}
	env := environment.Environment{
		ID: core.MustCanonicalID("trace_without_artifact"),
		Plugins: []environment.ExtensionRequirement{{
			ID:              def.ID,
			CompatibleMajor: 1,
		}},
	}
	lock := plugin.Lock{
		FormatVersion: plugin.BuildTraceLockFormatVersion,
		Environments: []plugin.EnvironmentLock{{
			Environment: env.ID,
			Plugins: []plugin.LockedPlugin{{
				ID:                   def.ID,
				Version:              def.Version,
				Fingerprint:          plugin.DefinitionFingerprint(def),
				BuildPlanFingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			}},
		}},
	}
	if _, err := plugin.EnvironmentLockFingerprint(lock, env.ID); err == nil {
		t.Fatal("expected build trace without artifact digest to fail")
	}
	if _, err := (plugin.Catalog{Plugins: []plugin.Definition{def}}).ResolveEnvironmentWithLock(env, lock); err == nil {
		t.Fatal("expected build trace without artifact digest to fail")
	}
}
