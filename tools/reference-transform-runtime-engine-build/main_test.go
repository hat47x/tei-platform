package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExactRuntimeEngineIdentitySeparatesLogicalVersionFromRepositoryRevision(t *testing.T) {
	digest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	engine, err := exactRuntimeEngineIdentity(digest, "host/test/test")
	if err != nil {
		t.Fatal(err)
	}
	if string(engine.ID) != engineID {
		t.Fatalf("engine id: got %s want %s", engine.ID, engineID)
	}
	if engine.Version != engineVersion {
		t.Fatalf("engine version: got %s want %s", engine.Version, engineVersion)
	}
	if engine.ContractVersion != engineContract {
		t.Fatalf("engine contract: got %s want %s", engine.ContractVersion, engineContract)
	}
	if engine.ImplementationDigest != digest {
		t.Fatalf("engine implementation digest: got %s want %s", engine.ImplementationDigest, digest)
	}
	if engine.Target != "host/test/test" {
		t.Fatalf("engine target: got %s", engine.Target)
	}

	evidenceA := buildEvidence{
		Schema:             evidenceSchema,
		RepositoryRevision: "revision-a",
		EngineID:           engineID,
		EngineVersion:      engineVersion,
		EngineContract:     engineContract,
		Target:             engine.Target,
		RuntimeEngine:      &engine,
	}
	evidenceB := evidenceA
	evidenceB.RepositoryRevision = "revision-b"

	if evidenceA.RuntimeEngine.Version != evidenceB.RuntimeEngine.Version {
		t.Fatal("repository revision changed logical Runtime Engine version")
	}
	if evidenceA.RuntimeEngine.ImplementationDigest != evidenceB.RuntimeEngine.ImplementationDigest {
		t.Fatal("repository revision changed exact Runtime Engine artifact identity")
	}
}

func TestExactRuntimeEngineIdentityRejectsUnresolvedArtifact(t *testing.T) {
	if _, err := exactRuntimeEngineIdentity("", "host/test/test"); err == nil {
		t.Fatal("expected unresolved artifact digest to fail")
	}
}

func TestResolveReproducibilityKeepsDivergentArtifactUnresolved(t *testing.T) {
	ok, artifact := resolveReproducibility(
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	)
	if ok {
		t.Fatal("different artifact digests were reported as reproducible")
	}
	if artifact != "" {
		t.Fatalf("non-reproducible artifact was assigned a canonical digest: %s", artifact)
	}

	ok, artifact = resolveReproducibility(
		"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	)
	if !ok {
		t.Fatal("matching artifact digests were not reported as reproducible")
	}
	if artifact != "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" {
		t.Fatalf("reproducible artifact digest: %s", artifact)
	}
}

func TestSourceClosureFingerprintIgnoresMemberOrder(t *testing.T) {
	a := []sourceMember{
		{Path: "b.go", Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		{Path: "a.go", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
	b := []sourceMember{a[1], a[0]}
	if sourceClosureFingerprint(a) != sourceClosureFingerprint(b) {
		t.Fatal("source closure fingerprint depends on declaration order")
	}
}

func TestSourceClosureFingerprintChangesWithSourceDigest(t *testing.T) {
	base := []sourceMember{{
		Path:   "a.go",
		Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}}
	changed := []sourceMember{{
		Path:   "a.go",
		Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}}
	if sourceClosureFingerprint(base) == sourceClosureFingerprint(changed) {
		t.Fatal("source closure fingerprint ignored member digest")
	}
}

func TestEvidenceFingerprintBindsBuildInputsAndArtifact(t *testing.T) {
	base := buildEvidence{
		Schema:              evidenceSchema,
		RepositoryRevision:  "revision-a",
		EngineID:            engineID,
		EngineContract:      engineContract,
		Target:              "host/test/test",
		Entrypoint:          enginePackage,
		SourceClosure:       []sourceMember{{Path: "go.mod", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},
		SourceClosureDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Builder: goBuilderEvidence{
			Version:         "go version go1.test",
			BinaryDigest:    "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			ToolchainDigest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		},
		Recipe:               []string{"go", "build", "-trimpath", enginePackage},
		BuildSettings:        []string{"CGO_ENABLED=0", "GOTOOLCHAIN=local"},
		FirstArtifactDigest:  "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		SecondArtifactDigest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
		Reproducible:         true,
		ArtifactDigest:       "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
	}
	baseFP, err := evidenceFingerprint(base)
	if err != nil {
		t.Fatal(err)
	}

	changed := base
	changed.BuildSettings = append([]string(nil), base.BuildSettings...)
	changed.BuildSettings[0] = "CGO_ENABLED=1"
	changedFP, err := evidenceFingerprint(changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedFP == baseFP {
		t.Fatal("evidence fingerprint ignored build settings")
	}

	changed = base
	changed.ArtifactDigest = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	changedFP, err = evidenceFingerprint(changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedFP == baseFP {
		t.Fatal("evidence fingerprint ignored artifact digest")
	}

	changed = base
	changed.EvidenceFingerprint = strings.Repeat("f", 64)
	changedFP, err = evidenceFingerprint(changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedFP != baseFP {
		t.Fatal("evidence fingerprint recursively depended on itself")
	}
}

func TestFixedGoBuildSettingsPinTargetAndAmbientGoControls(t *testing.T) {
	settings := fixedGoBuildSettings()
	required := []string{
		"CGO_ENABLED=0",
		"GOARCH=" + runtime.GOARCH,
		"GOENV=off",
		"GOOS=" + runtime.GOOS,
		"GOPROXY=off",
		"GOTOOLCHAIN=local",
		"GOWORK=off",
	}
	for _, want := range required {
		found := false
		for _, got := range settings {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing fixed Go build setting %q in %#v", want, settings)
		}
	}
}


func TestCopyVerifiedArtifactRequiresDestinationOutsideRepository(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(t.TempDir(), "engine")
	if err := os.WriteFile(source, []byte("engine-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := hashFile(source)
	if err != nil {
		t.Fatal(err)
	}

	inside := filepath.Join(root, "engine-copy")
	if err := copyVerifiedArtifactOutsideRepository(root, source, inside, digest); err == nil {
		t.Fatal("expected artifact export inside repository to fail")
	}
	if _, err := os.Stat(inside); !os.IsNotExist(err) {
		t.Fatalf("inside-repository artifact was created: %v", err)
	}

	outsideDir := t.TempDir()
	outside := filepath.Join(outsideDir, "engine-copy")
	if err := copyVerifiedArtifactOutsideRepository(root, source, outside, digest); err != nil {
		t.Fatal(err)
	}
	got, err := hashFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if got != digest {
		t.Fatalf("copied artifact digest changed: got %s want %s", got, digest)
	}
}

func TestCopyVerifiedArtifactRejectsUnexpectedDigest(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(t.TempDir(), "engine")
	if err := os.WriteFile(source, []byte("engine-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "engine-copy")
	err := copyVerifiedArtifactOutsideRepository(
		root,
		source,
		outside,
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	)
	if err == nil {
		t.Fatal("expected unexpected artifact digest to fail")
	}
	if _, statErr := os.Stat(outside); !os.IsNotExist(statErr) {
		t.Fatalf("artifact-out remained after digest failure: %v", statErr)
	}
}


func TestCopyVerifiedArtifactRejectsExistingDestination(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(t.TempDir(), "engine")
	if err := os.WriteFile(source, []byte("engine-bytes"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := hashFile(source)
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "engine-copy")
	if err := os.WriteFile(outside, []byte("existing"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyVerifiedArtifactOutsideRepository(root, source, outside, digest); err == nil {
		t.Fatal("expected existing artifact-out to fail")
	}
	raw, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "existing" {
		t.Fatal("existing artifact-out was overwritten")
	}
}
