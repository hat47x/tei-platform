package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/plugin/inputprofile"
	"tei/internal/plugin/transform"
)

// sb1GenerationTestHost models ONE Host-owned admission source. A single
// mutex protects CURRENT selection, revocation generation, and physical Start.
// This is a test fixture, not Production Host Authority or a grant store.
type sb1GenerationTestHost struct {
	mu          sync.RWMutex
	application core.CanonicalID
	environment core.CanonicalID
	selection   inputprofile.BuildInputSelection
	generation  uint64
	allowed     bool
	starts      int
}

func (h *sb1GenerationTestHost) prepare(
	ctx context.Context, source transform.Source, plan plugin.BuildPlan, raw []byte,
) (inputprofile.PreparedPackageInputs, uint64, error) {
	var generation uint64
	prepared, err := inputprofile.PrepareCurrentHostPackageInputs(
		ctx, h.application, h.environment,
		func(_ context.Context, app, env core.CanonicalID) (inputprofile.CurrentPackageBuildBinding, error) {
			h.mu.RLock()
			defer h.mu.RUnlock()
			if !h.allowed {
				return inputprofile.CurrentPackageBuildBinding{}, errors.New("Host revoked this build")
			}
			generation = h.generation
			return inputprofile.CurrentPackageBuildBinding{
				Application: h.application,
				Environment: h.environment,
				Selection:   h.selection,
			}, nil
		},
		source, plan, raw,
	)
	return prepared, generation, err
}

func (h *sb1GenerationTestHost) revoke() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.allowed = false
	h.generation++
}

func (h *sb1GenerationTestHost) restore() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.allowed = true
	h.generation++
}

func (h *sb1GenerationTestHost) startGate(
	generation uint64, selection inputprofile.BuildInputSelection,
	entered, release chan struct{},
) func(context.Context, func() error) error {
	return func(ctx context.Context, start func() error) error {
		h.mu.RLock()
		defer h.mu.RUnlock()
		if !h.allowed || h.generation != generation || h.selection != selection {
			return errors.New("current Host Start generation or exact SB-1 selection was revoked")
		}
		if entered != nil {
			close(entered)
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		// This critical section ends when cmd.Start returns, not when the
		// child exits. Host revocation cannot complete before the OS Start.
		h.starts++
		return start()
	}
}

func (h *sb1GenerationTestHost) startCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.starts
}

func waitSB1HostStage(t *testing.T, label string, ready <-chan struct{}) {
	t.Helper()
	select {
	case <-ready:
	case <-time.After(15 * time.Second):
		t.Fatalf("timeout waiting for SB-1 Host %s", label)
	}
}

// TestSB1HostGenerationAdmissionSerializesPhysicalStart contrasts the known
// read/start gap with a host-serialized reference OS Start boundary. It
// exercises a separately built v1 process, not an in-process evaluator.
//
// Passing demonstrates this fixture's linearization order only. A trusted
// Production Host must supply the actual scoped Authority, revocation lock,
// exact Source/Plan/Engine selection and Artifact acceptance independently.
func TestSB1HostGenerationAdmissionSerializesPhysicalStart(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(repo, filepath.FromSlash(transformPackageSourcePath))
	sourceRaw, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	source, err := transform.ParseSourceJSON(sourceRaw)
	if err != nil {
		t.Fatal(err)
	}
	sourceFP, err := transform.SourceFingerprint(source)
	if err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(t.TempDir(), "sb1-exact-v1-process")
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	build := exec.Command(goPath, "build", "-trimpath", "-buildvcs=false", "-mod=readonly", "-o", executable, "./tools/reference-transform-runtime-engine")
	build.Dir = repo
	build.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOPROXY=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build physical SB-1 Engine: %v: %s", err, output)
	}
	engineDigest, err := hashFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	definition := plugin.Definition{
		ID:      core.MustCanonicalID("tei.inputprofile.process"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
	}
	plan := plugin.BuildPlan{
		FormatVersion: plugin.BuildPlanFormatVersion,
		Plugin: plugin.Identity{
			ID: definition.ID, Version: definition.Version,
		},
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(definition),
		Sources: []plugin.BuildSource{{
			Kind: "plugin-definition", ID: "definition", Locator: "definition.tei",
			Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		}},
		Builder: plugin.BuilderIdentity{
			ID: core.MustCanonicalID("tei.builder.plugin_package"), Version: "1.0.0",
			ContractVersion: "tei.plugin-transform/v1",
			ArtifactDigest:  "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		Recipe:     plugin.BuildRecipe{Kind: "tei-plugin-package", Entrypoint: "definition.tei"},
		OutputKind: plugin.BuildOutputPluginPackage, Target: "plugin-package", Profile: "reference",
	}
	planFP, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}
	oracle, err := plugin.MaterializePluginPackage(definition, plan)
	if err != nil {
		t.Fatal(err)
	}
	const raw = `{"externalRequirements":[],"requires":[],"provides":[],"version":"1.0.0","runtime":{"args":[],"kind":"builtin"},"id":"tei.inputprofile.process"}`
	host := &sb1GenerationTestHost{
		application: core.MustCanonicalID("order_entry"),
		environment: core.MustCanonicalID("development"),
		selection: inputprofile.BuildInputSelection{
			Profile:                    inputprofile.PluginDefinitionJSONV2,
			LockedSourceFingerprint:    sourceFP,
			LockedBuildPlanFingerprint: planFP,
		},
		generation: 1, allowed: true,
	}
	first, firstGeneration, err := host.prepare(context.Background(), source, plan, []byte(raw))
	if err != nil || firstGeneration != 1 {
		t.Fatalf("initial exact Host selection failed: generation=%d err=%v", firstGeneration, err)
	}
	start := func(p inputprofile.PreparedPackageInputs, generation uint64, entered, release chan struct{}) (transform.EvaluationResult, error) {
		return evaluateWithExactEngineAtStart(
			executable, engineDigest, sourcePath, source, p.Inputs,
			host.startGate(generation, host.selection, entered, release),
		)
	}
	// If revocation completes before Start, the OLD prepared bytes cannot
	// start even though their content digest and canonical package are valid.
	host.revoke()
	if _, _, err := host.prepare(context.Background(), source, plan, []byte(raw)); !errors.Is(err, inputprofile.ErrCurrentPackageSelectionUnavailable) {
		t.Fatalf("current selection failed to notice revocation: %v", err)
	}
	if _, err := start(first, firstGeneration, nil, nil); err == nil {
		t.Fatal("revoked Host admitted physical SB-1 Start")
	}
	// Restoring the SAME Profile/Lock is not resurrection of the old grant.
	// The revoked generation must fail despite identical Source and Plan.
	host.restore()
	if _, err := start(first, firstGeneration, nil, nil); err == nil {
		t.Fatal("ABA restoration admitted stale Host selection generation")
	}
	ready, newGeneration, err := host.prepare(context.Background(), source, plan, []byte(raw))
	if err != nil || newGeneration != 3 || ready.Evidence != first.Evidence {
		t.Fatalf("restored Host selection changed canonical input or generation: %d %v", newGeneration, err)
	}

	// The physical Start holds the Host read-side gate; an overlapping revoke
	// cannot complete until OS Start has returned. No wait-for-child lock.
	entered := make(chan struct{})
	release := make(chan struct{})
	type processResult struct {
		output transform.EvaluationResult
		err    error
	}
	run := make(chan processResult, 1)
	go func() {
		out, err := start(ready, newGeneration, entered, release)
		run <- processResult{output: out, err: err}
	}()
	waitSB1HostStage(t, "Start gate entry", entered)
	attempted := make(chan struct{})
	revokedDone := make(chan struct{})
	go func() {
		close(attempted)
		host.revoke()
		close(revokedDone)
	}()
	waitSB1HostStage(t, "revocation attempt", attempted)
	select {
	case <-revokedDone:
		t.Fatal("Host revocation completed while physical Start gate was held")
	default:
	}
	close(release)
	var got processResult
	select {
	case got = <-run:
	case <-time.After(15 * time.Second):
		t.Fatal("admitted SB-1 process failed to finish")
	}
	if got.err != nil || !bytes.Equal(got.output.Output.Bytes, oracle) {
		t.Fatalf("Host-admitted physical Engine output mismatch: %v", got.err)
	}
	if _, err := inputprofile.ObservePackageOutput(ready.Evidence, got.output); err != nil {
		t.Fatalf("Host-admitted output observation: %v", err)
	}
	waitSB1HostStage(t, "revocation completion", revokedDone)
	if host.startCount() != 1 {
		t.Fatalf("physical Start count %d, want one", host.startCount())
	}
	if _, err := start(ready, newGeneration, nil, nil); err == nil {
		t.Fatal("completed revocation allowed old generation to start again")
	}
	// A fresh Host snapshot remains refused, but already returned canonical
	// bytes do not imply post-revocation acceptance or a signed Build Attempt.
	if _, _, err := host.prepare(context.Background(), source, plan, []byte(raw)); !errors.Is(err, inputprofile.ErrCurrentPackageSelectionUnavailable) {
		t.Fatal(fmt.Errorf("revoked Host returned new admission: %w", err))
	}
}
