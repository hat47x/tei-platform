package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

func writeReferenceStage2Acceptance(t *testing.T, root string, engine pruntime.RuntimeEngineIdentity) exactEngineOptions {
	t.Helper()
	options := exactEngineOptions{
		Stage2Lock:       filepath.Join(root, "stage2-lock.json"),
		HostRegistration: filepath.Join(root, "host-registration.json"),
	}
	write := func(path string, value any) {
		t.Helper()
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(options.Stage2Lock, referenceStage2EngineLock{
		FormatVersion: referenceStage2LockFormatVersion,
		Stage:         referenceStage2Stage,
		RuntimeEngine: engine,
	})
	write(options.HostRegistration, referenceStage2HostRegistration{
		FormatVersion: referenceStage2LockFormatVersion,
		Stage:         referenceStage2Stage,
		HostID:        core.MustCanonicalID("tei.reference.host"),
		RuntimeEngine: engine,
	})
	return options
}

func TestReferenceStage2AcceptanceRequiresLockAndIndependentHostRegistration(t *testing.T) {
	engine := bootstrapTestEngine()
	options := writeReferenceStage2Acceptance(t, t.TempDir(), engine)
	result, err := acceptReferenceStage2Engine(options, engine)
	if err != nil {
		t.Fatal(err)
	}
	if result.Engine != engine || result.LockFingerprint == "" || result.HostRegistrationFingerprint == "" {
		t.Fatalf("accepted engine identity or fingerprints missing: %#v", result)
	}
	for _, tt := range []struct {
		name string
		edit func(*exactEngineOptions)
	}{
		{"no lock", func(o *exactEngineOptions) { o.Stage2Lock = "" }},
		{"no registration", func(o *exactEngineOptions) { o.HostRegistration = "" }},
		{"no authority", func(o *exactEngineOptions) {
			o.Stage2Lock = ""
			o.HostRegistration = ""
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			incomplete := options
			tt.edit(&incomplete)
			if _, err := acceptReferenceStage2Engine(incomplete, engine); err == nil {
				t.Fatal("build evidence alone accidentally granted stage2 invocation authority")
			}
		})
	}
}

func TestReferenceStage2AcceptanceRejectsHostAndLockDriftBeforeInvocation(t *testing.T) {
	engine := bootstrapTestEngine()
	root := t.TempDir()
	options := writeReferenceStage2Acceptance(t, root, engine)
	if _, err := acceptReferenceStage2Engine(options, engine); err != nil {
		t.Fatal(err)
	}
	changed := engine
	changed.ImplementationDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err := acceptReferenceStage2Engine(options, changed); err == nil || !strings.Contains(err.Error(), "lock") {
		t.Fatalf("rotated executable accepted under stale lock: %v", err)
	}
	writeReferenceStage2Acceptance(t, root, changed)
	if _, err := acceptReferenceStage2Engine(options, engine); err == nil {
		t.Fatal("stale executable accepted under rotated lock and Host registration")
	}
	writeReferenceStage2Acceptance(t, root, engine)
	raw, err := os.ReadFile(options.HostRegistration)
	if err != nil {
		t.Fatal(err)
	}
	var registration referenceStage2HostRegistration
	if err := json.Unmarshal(raw, &registration); err != nil {
		t.Fatal(err)
	}
	registration.RuntimeEngine.ImplementationDigest = changed.ImplementationDigest
	raw, err = json.Marshal(registration)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(options.HostRegistration, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := acceptReferenceStage2Engine(options, engine); err == nil || !strings.Contains(err.Error(), "Host registration") {
		t.Fatalf("mismatched Host registration accepted: %v", err)
	}
}

func TestReferenceStage2LockFingerprintFollowsArtifactNotRepositoryRevision(t *testing.T) {
	engine := bootstrapTestEngine()
	dir := t.TempDir()
	options := writeReferenceStage2Acceptance(t, dir, engine)
	first, err := acceptReferenceStage2Engine(options, engine)
	if err != nil {
		t.Fatal(err)
	}
	// Neither record carries a repository revision: docs-only repository
	// changes do not invalidate a byte-identical exact engine artifact.
	second, err := acceptReferenceStage2Engine(options, engine)
	if err != nil {
		t.Fatal(err)
	}
	if first.LockFingerprint != second.LockFingerprint || first.HostRegistrationFingerprint != second.HostRegistrationFingerprint {
		t.Fatal("unchanged engine or Host authority changed acceptance fingerprints")
	}
	changed := engine
	changed.ImplementationDigest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	writeReferenceStage2Acceptance(t, dir, changed)
	rotated, err := acceptReferenceStage2Engine(options, changed)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.LockFingerprint == first.LockFingerprint || rotated.HostRegistrationFingerprint == first.HostRegistrationFingerprint {
		t.Fatal("runtime executable rotation did not invalidate lock or registration")
	}
}

func TestReferenceStage2RuntimePolicyBindsAcceptedEngineAndPlan(t *testing.T) {
	engine := bootstrapTestEngine()
	options := writeReferenceStage2Acceptance(t, t.TempDir(), engine)
	accepted, err := acceptReferenceStage2Engine(options, engine)
	if err != nil {
		t.Fatal(err)
	}
	source := bootstrapTestMetaSource(t)
	plan := transformBootstrapStagePlan{
		Stage: plugin.BootstrapStage2,
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.stage2_test"),
			Version:         "1.0.0",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		},
		SourceClosureDigest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		SourceFingerprint:   "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		Execution: transformBootstrapStageExecution{
			Kind:          transformBootstrapExecutionInterpreted,
			Target:        engine.Target,
			RuntimeEngine: &engine,
		},
		Output: source.Identity,
	}
	policy, err := accepted.PolicyFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(policy) != 64 {
		t.Fatalf("unexpected policy fingerprint: %s", policy)
	}
	changedPlan := plan
	changedPlan.Builder.ArtifactDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	next, err := accepted.PolicyFingerprint(changedPlan)
	if err != nil {
		t.Fatal(err)
	}
	if policy == next {
		t.Fatal("stage2 plan change failed to stale the Runtime Policy fingerprint")
	}
	other := engine
	other.ImplementationDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	changedPlan.Execution.RuntimeEngine = &other
	if _, err := accepted.PolicyFingerprint(changedPlan); err == nil {
		t.Fatal("accepted engine does not match execution plan, but policy was issued")
	}
}
