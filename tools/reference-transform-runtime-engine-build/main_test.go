package main

import (
	"runtime"
	"strings"
	"testing"
)

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
