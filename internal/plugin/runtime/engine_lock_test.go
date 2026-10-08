package runtime_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

const engineLockArtifactDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func engineLockIdentity(id string) pruntime.RuntimeEngineIdentity {
	return pruntime.RuntimeEngineIdentity{
		ID:                   core.MustCanonicalID(id),
		Version:              "1.0.0",
		ContractVersion:      "tei.runtime-engine/reference-v1",
		ImplementationDigest: engineLockArtifactDigest,
		Target:               "host/reference",
	}
}

func engineLockPlan(boundary pruntime.Boundary) pruntime.EnvironmentPlan {
	plan := pruntime.EnvironmentPlan{
		Environment:           core.MustCanonicalID("reference"),
		PluginLockFingerprint: "plugin-lock",
		Plugins: []pruntime.PluginRuntime{{
			Plugin: plugin.Identity{
				ID:      core.MustCanonicalID("tei.native.reference"),
				Version: "1.0.0",
			},
			DescriptorFingerprint: "descriptor",
			BuildPlanFingerprint:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Boundary:              boundary,
			Artifact: plugin.Artifact{
				Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			},
		}},
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	return plan
}

func TestRuntimeEngineLockFingerprintIgnoresBindingOrder(t *testing.T) {
	a := pruntime.RuntimeEngineLock{
		FormatVersion: pruntime.RuntimeEngineLockFormatVersion,
		Bindings: []pruntime.RuntimeEngineBinding{
			{
				Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.z"), Version: "1.0.0"},
				Engine: engineLockIdentity("tei.runtime.engine.z"),
			},
			{
				Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.a"), Version: "1.0.0"},
				Engine: engineLockIdentity("tei.runtime.engine.a"),
			},
		},
	}
	b := a
	b.Bindings = append([]pruntime.RuntimeEngineBinding(nil), a.Bindings...)
	b.Bindings[0], b.Bindings[1] = b.Bindings[1], b.Bindings[0]

	afp, err := pruntime.RuntimeEngineLockFingerprint(a)
	if err != nil {
		t.Fatal(err)
	}
	bfp, err := pruntime.RuntimeEngineLockFingerprint(b)
	if err != nil {
		t.Fatal(err)
	}
	if afp != bfp {
		t.Fatalf("binding order changed runtime engine lock fingerprint: %s != %s", afp, bfp)
	}
}

func TestRuntimeEngineLockRejectsDuplicatePluginBinding(t *testing.T) {
	lock := pruntime.RuntimeEngineLock{
		FormatVersion: pruntime.RuntimeEngineLockFormatVersion,
		Bindings: []pruntime.RuntimeEngineBinding{
			{
				Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.native.reference"), Version: "1.0.0"},
				Engine: engineLockIdentity("tei.runtime.engine.a"),
			},
			{
				Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.native.reference"), Version: "1.0.0"},
				Engine: engineLockIdentity("tei.runtime.engine.b"),
			},
		},
	}
	if _, err := pruntime.RuntimeEngineLockFingerprint(lock); err == nil {
		t.Fatal("expected duplicate runtime engine binding to fail")
	}
}

func TestBindRuntimeEngineLockRequiresEveryBuildBoundBuiltin(t *testing.T) {
	plan := engineLockPlan(pruntime.BoundaryInProcess)
	lock := pruntime.RuntimeEngineLock{FormatVersion: pruntime.RuntimeEngineLockFormatVersion}
	if _, err := pruntime.BindRuntimeEngineLock(plan, lock); err == nil {
		t.Fatal("expected missing build-bound builtin engine binding to fail")
	}
}

func TestBindRuntimeEngineLockRejectsProcessPlugin(t *testing.T) {
	plan := engineLockPlan(pruntime.BoundaryProcess)
	lock := pruntime.RuntimeEngineLock{
		FormatVersion: pruntime.RuntimeEngineLockFormatVersion,
		Bindings: []pruntime.RuntimeEngineBinding{{
			Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.native.reference"), Version: "1.0.0"},
			Engine: engineLockIdentity("tei.runtime.engine.a"),
		}},
	}
	if _, err := pruntime.BindRuntimeEngineLock(plan, lock); err == nil {
		t.Fatal("expected process plugin runtime engine binding to fail")
	}
}

func TestRuntimeEngineIdentityChangesLockAndPolicyFingerprint(t *testing.T) {
	plan := engineLockPlan(pruntime.BoundaryInProcess)
	baseEngine := engineLockIdentity("tei.runtime.engine.a")
	baseLock := pruntime.RuntimeEngineLock{
		FormatVersion: pruntime.RuntimeEngineLockFormatVersion,
		Bindings: []pruntime.RuntimeEngineBinding{{
			Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.native.reference"), Version: "1.0.0"},
			Engine: baseEngine,
		}},
	}
	baseFP, err := pruntime.RuntimeEngineLockFingerprint(baseLock)
	if err != nil {
		t.Fatal(err)
	}
	basePlan, err := pruntime.BindRuntimeEngineLock(plan, baseLock)
	if err != nil {
		t.Fatal(err)
	}

	changedEngine := baseEngine
	changedEngine.ImplementationDigest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	changedLock := baseLock
	changedLock.Bindings = append([]pruntime.RuntimeEngineBinding(nil), baseLock.Bindings...)
	changedLock.Bindings[0].Engine = changedEngine
	changedFP, err := pruntime.RuntimeEngineLockFingerprint(changedLock)
	if err != nil {
		t.Fatal(err)
	}
	changedPlan, err := pruntime.BindRuntimeEngineLock(plan, changedLock)
	if err != nil {
		t.Fatal(err)
	}

	if baseFP == changedFP {
		t.Fatal("runtime engine implementation change did not change lock fingerprint")
	}
	if basePlan.RuntimePolicyFingerprint == changedPlan.RuntimePolicyFingerprint {
		t.Fatal("runtime engine implementation change did not change runtime policy fingerprint")
	}
}

func TestRuntimeEngineLockRejectsMalformedInputs(t *testing.T) {
	base := pruntime.RuntimeEngineLock{
		FormatVersion: pruntime.RuntimeEngineLockFormatVersion,
		Bindings: []pruntime.RuntimeEngineBinding{{
			Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.native.reference"), Version: "1.0.0"},
			Engine: engineLockIdentity("tei.runtime.engine.a"),
		}},
	}
	tests := []struct {
		name string
		edit func(*pruntime.RuntimeEngineLock)
	}{
		{"format", func(lock *pruntime.RuntimeEngineLock) {
			lock.FormatVersion = 99
		}},
		{"empty", func(lock *pruntime.RuntimeEngineLock) {
			lock.Bindings = nil
		}},
		{"plugin id", func(lock *pruntime.RuntimeEngineLock) {
			lock.Bindings[0].Plugin.ID = core.CanonicalID("Invalid")
		}},
		{"plugin version", func(lock *pruntime.RuntimeEngineLock) {
			lock.Bindings[0].Plugin.Version = "v1"
		}},
		{"engine id", func(lock *pruntime.RuntimeEngineLock) {
			lock.Bindings[0].Engine.ID = core.CanonicalID("Invalid")
		}},
		{"engine digest", func(lock *pruntime.RuntimeEngineLock) {
			lock.Bindings[0].Engine.ImplementationDigest = "sha256:bad"
		}},
		{"engine contract", func(lock *pruntime.RuntimeEngineLock) {
			lock.Bindings[0].Engine.ContractVersion = ""
		}},
		{"engine target", func(lock *pruntime.RuntimeEngineLock) {
			lock.Bindings[0].Engine.Target = ""
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lock := base
			lock.Bindings = append([]pruntime.RuntimeEngineBinding(nil), base.Bindings...)
			tt.edit(&lock)
			if _, err := pruntime.RuntimeEngineLockFingerprint(lock); err == nil {
				t.Fatal("expected malformed runtime engine lock to fail")
			}
		})
	}
}
