package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/core"
	pruntime "tei/internal/plugin/runtime"
	"tei/internal/plugin/transform"
)

func makeExactEngineFixture(t *testing.T) (root string, options exactEngineOptions) {
	t.Helper()
	base := t.TempDir()
	root = filepath.Join(base, "repo")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	options.Executable = filepath.Join(base, "verified-engine")
	options.BuildEvidence = filepath.Join(base, "build-evidence.json")
	if err := os.WriteFile(options.Executable, []byte("test engine artifact"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := hashFile(options.Executable)
	if err != nil {
		t.Fatal(err)
	}
	engine := pruntime.RuntimeEngineIdentity{
		ID:                   core.MustCanonicalID("tei.runtime.engine.deterministic_transform.reference"),
		Version:              transformRuntimeEngineVersion,
		ContractVersion:      "tei.runtime-engine/deterministic-transform-v1",
		ImplementationDigest: digest,
		Target:               "host/" + runtime.GOOS + "/" + runtime.GOARCH,
	}
	e := exactEngineBuildEvidence{
		Schema:               "tei.reference.transform-engine-build-evidence/v1",
		RepositoryRevision:   "revision-a",
		EngineID:             string(engine.ID),
		EngineVersion:        engine.Version,
		EngineContract:       engine.ContractVersion,
		Target:               engine.Target,
		RuntimeEngine:        &engine,
		Entrypoint:           "./tools/reference-transform-runtime-engine",
		SourceClosure:        []exactEngineSourceMember{{Path: "go.mod", Digest: digest}},
		SourceClosureDigest:  digest,
		Builder:              exactEngineGoBuilder{Version: "go1.23.2", ToolchainDigest: digest},
		Recipe:               []string{"go", "build"},
		FirstArtifactDigest:  digest,
		SecondArtifactDigest: digest,
		Reproducible:         true,
		ArtifactDigest:       digest,
	}
	fingerprint, err := exactEngineFingerprint(e)
	if err != nil {
		t.Fatal(err)
	}
	e.EvidenceFingerprint = fingerprint
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(options.BuildEvidence, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return root, options
}

func TestExactEngineRequiresReproducibleArtifactAndExactBuildReceipt(t *testing.T) {
	root, options := makeExactEngineFixture(t)
	engine, fingerprint, err := verifyExactEngine(root, "revision-a", options)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprint == "" || engine.ImplementationDigest == "" {
		t.Fatal("exact runtime binding lost build evidence or artifact digest")
	}
	if _, _, err := verifyExactEngine(root, "revision-b", options); err == nil {
		t.Fatal("build Evidence from a different checkout was accepted")
	}
	if err := os.WriteFile(options.Executable, []byte("tampered binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := verifyExactEngine(root, "revision-a", options); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("changed executable was accepted: %v", err)
	}
}

func TestExactEngineRejectsAlteredReceipt(t *testing.T) {
	root, options := makeExactEngineFixture(t)
	raw, err := os.ReadFile(options.BuildEvidence)
	if err != nil {
		t.Fatal(err)
	}
	var e exactEngineBuildEvidence
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatal(err)
	}
	e.RuntimeEngine.Version = "1.0.1"
	raw, err = json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(options.BuildEvidence, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := verifyExactEngine(root, "revision-a", options); err == nil {
		t.Fatal("altered engine identity was accepted")
	}
}

func TestExactEngineRejectsMissingOrRepositoryLocalArtifact(t *testing.T) {
	root, options := makeExactEngineFixture(t)
	if _, _, err := verifyExactEngine(root, "revision-a", exactEngineOptions{Executable: options.Executable}); err == nil {
		t.Fatal("unpaired executable was accepted")
	}
	inside := filepath.Join(root, "inside-engine")
	if err := os.WriteFile(inside, []byte("test engine artifact"), 0o755); err != nil {
		t.Fatal(err)
	}
	options.Executable = inside
	if _, _, err := verifyExactEngine(root, "revision-a", options); err == nil {
		t.Fatal("repository-local engine executable was accepted")
	}
}

func TestExactEngineWirePreservesOrderedObjectsAndBytes(t *testing.T) {
	value, err := transform.Object(
		transform.Field{Name: "second", Value: transform.Bytes([]byte{0, 0xff})},
		transform.Field{Name: "first", Value: transform.Array(transform.String("ok"))},
	)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := exactEngineWire(value, 0)
	if err != nil {
		t.Fatal(err)
	}
	if wire.Object == nil || len(*wire.Object) != 2 {
		t.Fatalf("missing ordered fields: %#v", wire)
	}
	if (*wire.Object)[0].Name != "second" || (*wire.Object)[1].Name != "first" {
		t.Fatalf("field order changed: %#v", *wire.Object)
	}
	if (*wire.Object)[0].Value.BytesBase64 == nil || *(*wire.Object)[0].Value.BytesBase64 != "AP8=" {
		t.Fatalf("byte value changed: %#v", *wire.Object)
	}
}
