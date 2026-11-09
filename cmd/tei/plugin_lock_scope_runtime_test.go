package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/plugin"
	"tei/internal/plugin/implementation"
	pluginbuiltin "tei/internal/plugin/implementation/builtin"
	pluginruntime "tei/internal/plugin/runtime"
	"tei/internal/workspace"
)

// Verifies only the reference realization bytes; not signatures, publisher
// trust, an OS sandbox, or a production artifact store.
type scopedRealizationVerifier struct{ bytes []byte }

func (v scopedRealizationVerifier) Verify(_ context.Context, _ plugin.Identity, artifact plugin.Artifact) (pluginruntime.ArtifactVerification, error) {
	if plugin.BuildOutputDigest(v.bytes) != artifact.Digest {
		return pluginruntime.ArtifactVerification{}, fmt.Errorf("realization digest mismatch")
	}
	return pluginruntime.ArtifactVerification{VerifierID: "reference-realization-digest", DigestVerified: true}, nil
}

func TestScopedCLILockRunsReferenceBuiltinWithoutBusinessModel(t *testing.T) {
	repo := testRepositoryRoot(t)
	root := filepath.Join(repo, "reference", "plugin-self-build", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.World.Semantics) != 1 || loaded.World.Semantics[0].ID != "std" || len(loaded.World.Semantics[0].Data) != 0 || len(loaded.PluginImplementations) != 1 || len(loaded.World.Environments) != 1 {
		t.Fatalf("unexpected reference workspace: %+v", loaded.World)
	}
	source := loaded.PluginImplementations[0]
	materialized, digest, err := implementation.Materialize(source)
	if err != nil {
		t.Fatal(err)
	}
	var realized plugin.Definition
	for i := range loaded.World.Plugins {
		def := loaded.World.Plugins[i]
		if def.ID == source.Plugin.ID && def.Version == source.Plugin.Version {
			realized, err = plugin.RealizeBuiltDefinition(def, digest)
			if err != nil {
				t.Fatal(err)
			}
			loaded.World.Plugins[i] = realized
		}
	}
	if realized.ID == "" {
		t.Fatal("reference implementation descriptor missing")
	}
	dir := t.TempDir()
	worldPath := filepath.Join(dir, "world.json")
	w, err := json.Marshal(loaded.World)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(worldPath, w, 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "tei")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, "./cmd/tei")
	build.Dir = repo
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	command := exec.Command(bin, "plugin-lock", "-world", worldPath, "-environment", "reference")
	var out, stderr bytes.Buffer
	command.Stdout, command.Stderr = &out, &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("CLI lock: %v\n%s\n%s", err, out.String(), stderr.String())
	}
	var envelope struct {
		OK     bool        `json:"ok"`
		Mode   string      `json:"mode"`
		Result plugin.Lock `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil || !envelope.OK || envelope.Mode != "environment" || len(envelope.Result.Environments) != 1 {
		t.Fatalf("lock envelope: %+v, %v", envelope, err)
	}
	catalog := plugin.Catalog{Plugins: loaded.World.Plugins, Bundles: loaded.World.Bundles}
	env := loaded.World.Environments[0]
	plan, err := pluginruntime.BuildPlan(catalog, env, envelope.Result)
	if err != nil {
		t.Fatal(err)
	}
	builtin, err := pluginbuiltin.New(materialized, realized)
	if err != nil {
		t.Fatal(err)
	}
	adapter := pluginruntime.DeploymentExecutor{
		Plan: plan, CapabilityID: core.MustCanonicalID("runtime.native_reference"),
		Registry: pluginruntime.Registry{Builtins: map[string]pluginruntime.Builtin{string(realized.ID) + "@" + realized.Version: builtin}, Verifier: scopedRealizationVerifier{bytes: materialized}},
	}
	commandInput := executor.Command{ActionID: "application/start", IdempotencyKey: "scope-test", Kind: string(deployment.ActionApplicationStart)}
	result, err := adapter.Execute(context.Background(), commandInput)
	if err != nil || result.Outcome != executor.OutcomeAccepted || result.AttemptID != "tei-native-reference" {
		t.Fatalf("reference builtin invocation: %+v, %v", result, err)
	}
	other := env
	other.ID = core.MustCanonicalID("not_pinned")
	if _, err := pluginruntime.BuildPlan(catalog, other, envelope.Result); err == nil || !strings.Contains(err.Error(), "no environment") {
		t.Fatalf("partial lock applied elsewhere: %v", err)
	}
	modified := append([]plugin.Definition(nil), catalog.Plugins...)
	for i := range modified {
		if modified[i].ID == realized.ID {
			modified[i].Runtime.Command = "modified"
		}
	}
	if _, err := pluginruntime.BuildPlan(plugin.Catalog{Plugins: modified, Bundles: catalog.Bundles}, env, envelope.Result); err == nil || !strings.Contains(err.Error(), "fingerprint diverged") {
		t.Fatalf("changed descriptor accepted: %v", err)
	}
	badRegistry := adapter.Registry
	badRegistry.Verifier = scopedRealizationVerifier{bytes: []byte("not the selected realization")}
	badAdapter := adapter
	badAdapter.Registry = badRegistry
	if _, err := badAdapter.Execute(context.Background(), commandInput); err == nil || !strings.Contains(err.Error(), "verification failed") {
		t.Fatalf("unverified realization executed: %v", err)
	}
	t.Log("CLI partial lock -> reference runtime plan -> actual native builtin response without business semantic definitions; wrong environment, changed descriptor and wrong realization rejected")
}
