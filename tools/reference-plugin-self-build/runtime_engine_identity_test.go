package main

import (
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

func TestReferenceRuntimeEngineLogicalVersionAndProvenance(t *testing.T) {
	cases := []struct {
		name     string
		id       core.CanonicalID
		version  string
		contract string
	}{
		{
			name:     "deterministic transform",
			id:       core.MustCanonicalID("tei.runtime.engine.deterministic_transform.reference"),
			version:  transformRuntimeEngineVersion,
			contract: "tei.runtime-engine/deterministic-transform-v1",
		},
		{
			name:     "static response",
			id:       core.MustCanonicalID("tei.runtime.engine.static_response.reference"),
			version:  nativeRuntimeEngineVersion,
			contract: "tei.runtime-engine/static-response-v1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const source = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			const toolchain = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
			const target = "host/linux/amd64"
			engine, err := referenceRuntimeEngineIdentity(tc.id, tc.version, tc.contract, source, toolchain, target)
			if err != nil {
				t.Fatal(err)
			}
			if engine.Version != "1.0.0" || engine.ID != tc.id || engine.ContractVersion != tc.contract {
				t.Fatalf("unexpected logical Runtime Engine identity: %+v", engine)
			}
			if want := sha256Digest([]byte(source + "\n" + toolchain + "\n")); engine.ImplementationDigest != want {
				t.Fatalf("engine implementation digest = %q, want %q", engine.ImplementationDigest, want)
			}
			fingerprint, err := pruntime.RuntimeEngineIdentityFingerprint(engine)
			if err != nil {
				t.Fatal(err)
			}

			// Repository revision is retained as build provenance, not inserted
			// into RuntimeEngineIdentity or its fingerprint.
			first := evidence{RepositoryRevision: "revision-a", NativeRuntimeEngine: engine}
			second := evidence{RepositoryRevision: "revision-b", NativeRuntimeEngine: engine}
			if first.RepositoryRevision == second.RepositoryRevision {
				t.Fatal("test must use distinct repository revisions")
			}
			if got, err := pruntime.RuntimeEngineIdentityFingerprint(second.NativeRuntimeEngine); err != nil || got != fingerprint {
				t.Fatalf("repository revision changed engine fingerprint: %q, %v", got, err)
			}

			changedSource, err := referenceRuntimeEngineIdentity(tc.id, tc.version, tc.contract, "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", toolchain, target)
			if err != nil {
				t.Fatal(err)
			}
			changedToolchain, err := referenceRuntimeEngineIdentity(tc.id, tc.version, tc.contract, source, "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", target)
			if err != nil {
				t.Fatal(err)
			}
			for name, changed := range map[string]pruntime.RuntimeEngineIdentity{
				"source":    changedSource,
				"toolchain": changedToolchain,
				"release": func() pruntime.RuntimeEngineIdentity {
					copy := engine
					copy.Version = "1.0.1"
					return copy
				}(),
				"contract": func() pruntime.RuntimeEngineIdentity {
					copy := engine
					copy.ContractVersion = tc.contract + ".next"
					return copy
				}(),
				"target": func() pruntime.RuntimeEngineIdentity {
					copy := engine
					copy.Target = "host/windows/amd64"
					return copy
				}(),
			} {
				got, err := pruntime.RuntimeEngineIdentityFingerprint(changed)
				if err != nil {
					t.Fatalf("%s: %v", name, err)
				}
				if got == fingerprint {
					t.Errorf("%s change did not change engine fingerprint", name)
				}
			}
		})
	}
}

func TestReferenceRuntimeEngineIdentityRejectsInvalidTarget(t *testing.T) {
	_, err := referenceRuntimeEngineIdentity(
		core.MustCanonicalID("tei.runtime.engine.deterministic_transform.reference"),
		transformRuntimeEngineVersion,
		"tei.runtime-engine/deterministic-transform-v1",
		"source", "toolchain", "",
	)
	if err == nil {
		t.Fatal("missing target must be rejected")
	}
}

// The Reference lock must not become stale from a docs-only repository commit.
// A real implementation change still must invalidate the lock.
func TestReferenceRuntimeEngineLockSeparatesRevisionFromImplementation(t *testing.T) {
	build := func(revision, sourceDigest string) (evidence, pruntime.RuntimeEngineLock) {
		engine, err := referenceRuntimeEngineIdentity(
			core.MustCanonicalID("tei.runtime.engine.static_response.reference"),
			nativeRuntimeEngineVersion,
			"tei.runtime-engine/static-response-v1",
			sourceDigest,
			"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"host/linux/amd64",
		)
		if err != nil {
			t.Fatal(err)
		}
		return evidence{RepositoryRevision: revision, NativeRuntimeEngine: engine}, pruntime.RuntimeEngineLock{
			FormatVersion: pruntime.RuntimeEngineLockFormatVersion,
			Bindings: []pruntime.RuntimeEngineBinding{{
				Plugin: plugin.Identity{
					ID:      core.MustCanonicalID("tei.native.runtime.reference"),
					Version: "1.0.0",
				},
				Engine: engine,
			}},
		}
	}
	const source = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	firstEvidence, firstLock := build("revision-before-docs", source)
	secondEvidence, secondLock := build("revision-after-docs", source)
	if firstEvidence.RepositoryRevision == secondEvidence.RepositoryRevision {
		t.Fatal("repository provenance must retain the different revisions")
	}
	if firstEvidence.NativeRuntimeEngine.ImplementationDigest != firstLock.Bindings[0].Engine.ImplementationDigest {
		t.Fatal("runtime engine lock lost its implementation digest")
	}
	firstFP, err := pruntime.RuntimeEngineLockFingerprint(firstLock)
	if err != nil {
		t.Fatal(err)
	}
	secondFP, err := pruntime.RuntimeEngineLockFingerprint(secondLock)
	if err != nil {
		t.Fatal(err)
	}
	if firstFP != secondFP {
		t.Fatal("repository-only revision changed the Runtime Engine Lock")
	}
	_, changedLock := build("revision-after-source-change", "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	changedFP, err := pruntime.RuntimeEngineLockFingerprint(changedLock)
	if err != nil {
		t.Fatal(err)
	}
	if changedFP == firstFP {
		t.Fatal("implementation source change did not invalidate the Runtime Engine Lock")
	}
}
