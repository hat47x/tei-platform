package pluginbuildsupply

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

// RunnerEvidence is the part of the self-build runner's evidence JSON that
// the sample binds to stages 1, 2 and 4. Field names are the runner's.
type RunnerEvidence struct {
	Schema                             string                         `json:"schema"`
	RepositoryRevision                 string                         `json:"repositoryRevision"`
	RepositoryTreeDigest               string                         `json:"repositoryTreeDigest"`
	GoVersion                          string                         `json:"goVersion"`
	GoBinaryDigest                     string                         `json:"goBinaryDigest"`
	GoToolchainDigest                  string                         `json:"goToolchainDigest"`
	ProcessSourceClosureDigest         string                         `json:"processSourceClosureDigest"`
	TransformBootstrap                 RunnerTransformBootstrap       `json:"transformBootstrap"`
	PackageReproducible                bool                           `json:"packageReproducible"`
	NativeReproducible                 bool                           `json:"nativeReproducible"`
	NativeFirstArtifactDigest          string                         `json:"nativeFirstArtifactDigest"`
	NativeRuntimeEngine                pruntime.RuntimeEngineIdentity `json:"nativeRuntimeEngine"`
	NativeRuntimeEngineFingerprint     string                         `json:"nativeRuntimeEngineFingerprint"`
	NativeRuntimeEngineLock            pruntime.RuntimeEngineLock     `json:"nativeRuntimeEngineLock"`
	NativeRuntimeEngineLockFingerprint string                         `json:"nativeRuntimeEngineLockFingerprint"`
	BuildPlan                          plugin.BuildPlan               `json:"buildPlan"`
	BuildPlanFingerprint               string                         `json:"buildPlanFingerprint"`
	FirstArtifactDigest                string                         `json:"firstArtifactDigest"`
	SecondArtifactDigest               string                         `json:"secondArtifactDigest"`
	Reproducible                       bool                           `json:"reproducible"`
	LockedBuild                        plugin.LockedBuild             `json:"lockedBuild"`
	BuildArtifactLockFingerprint       string                         `json:"buildArtifactLockFingerprint"`
	RuntimePluginLockFingerprint       string                         `json:"runtimePluginLockFingerprint"`
	RuntimePluginLock                  plugin.Lock                    `json:"runtimePluginLock"`
}

type RunnerTransformBootstrap struct {
	RuntimeEngine                pruntime.RuntimeEngineIdentity    `json:"runtimeEngine"`
	RuntimeEngineFingerprint     string                            `json:"runtimeEngineFingerprint"`
	SeedBuilderReproducible      bool                              `json:"seedBuilderReproducible"`
	Stage1ArtifactDigest         string                            `json:"stage1ArtifactDigest"`
	Stage2ArtifactDigest         string                            `json:"stage2ArtifactDigest"`
	ConvergenceResult            plugin.BootstrapConvergenceResult `json:"convergenceResult"`
	PackageBuilderArtifactDigest string                            `json:"packageBuilderArtifactDigest"`
	PackageBuildPlan             plugin.BuildPlan                  `json:"packageBuildPlan"`
	PackageBuildPlanFingerprint  string                            `json:"packageBuildPlanFingerprint"`
	PackageReproducible          bool                              `json:"packageReproducible"`
	PackageFirstArtifactDigest   string                            `json:"packageFirstArtifactDigest"`
	PackageOracleDigest          string                            `json:"packageOracleDigest"`
}

// RunnerResult is one run of tools/reference-plugin-self-build.
type RunnerResult struct {
	ExitCode int
	Stdout   []byte
	Stderr   string
	Evidence *RunnerEvidence
}

// RunSelfBuild runs the existing runner binary unchanged. env is the full
// environment of the runner process.
func RunSelfBuild(ctx context.Context, runner, repo, goPath string, env []string) (RunnerResult, error) {
	cmd := exec.CommandContext(ctx, runner, "-repo", repo, "-go", goPath)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	result := RunnerResult{Stdout: stdout.Bytes(), Stderr: stderr.String()}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	default:
		return result, err
	}
	var ev RunnerEvidence
	dec := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	if err := dec.Decode(&ev); err != nil {
		return result, fmt.Errorf("decode runner evidence: %w", err)
	}
	result.Evidence = &ev
	return result, nil
}

// Stage3 is the binding of the runner's evidence to the confirmed plan and
// the stored artifact.
type Stage3 struct {
	PlanFingerprint                 string `json:"planFingerprint"`
	FirstArtifactDigest             string `json:"firstArtifactDigest"`
	SecondArtifactDigest            string `json:"secondArtifactDigest"`
	RegisteredArtifactDigest        string `json:"registeredArtifactDigest"`
	RegisteredPlanFingerprint       string `json:"registeredPlanFingerprint"`
	RuntimePluginLockFingerprint    string `json:"runtimePluginLockFingerprint"`
	TransformEngineFingerprint      string `json:"transformEngineFingerprint"`
	StaticResponseEngineFingerprint string `json:"staticResponseEngineFingerprint"`
	BootstrapConverged              bool   `json:"bootstrapConverged"`
	SeedBuilderTrustRetained        bool   `json:"seedBuilderTrustRetained"`
}

// ErrBindingMismatch marks a stage 3 binding failure.
var ErrBindingMismatch = errors.New("stage 3 evidence does not match the confirmed plan")

// BindStage3 accepts the runner's evidence only when its own plan is the
// plan stage 1 confirmed, the source closure and toolchain are the ones
// stage 1 observed, both builds equal the artifact stage 2 stored, and the
// host registration (runtime plugin lock) names the same plan and digest.
// The runner assembles its plan by itself; this is the re-check of that plan
// against the derived one.
func BindStage3(s1 Stage1, rep Reproducibility, ev RunnerEvidence, trust TrustStore) (Stage3, error) {
	fail := func(format string, args ...any) (Stage3, error) {
		return Stage3{}, fmt.Errorf("%w: "+format, append([]any{ErrBindingMismatch}, args...)...)
	}
	if err := trust.Admit(s1.Lock); err != nil {
		return Stage3{}, err
	}
	if ev.GoToolchainDigest != s1.Toolchain.ToolchainDigest || ev.BuildPlan.Builder != s1.Plan.Builder {
		return fail("toolchain %s, stage 1 resolved builder %s", ev.GoToolchainDigest, s1.Plan.Builder.ArtifactDigest)
	}
	var closureDigest string
	for _, src := range s1.Plan.Sources {
		if src.Kind == sourceKindGoClosure {
			closureDigest = src.Digest
		}
	}
	if ev.ProcessSourceClosureDigest != closureDigest {
		return fail("source closure %s, stage 1 observed %s", ev.ProcessSourceClosureDigest, closureDigest)
	}
	if ev.RepositoryRevision != s1.Revision {
		return fail("revision %s, stage 1 derived at %s", ev.RepositoryRevision, s1.Revision)
	}
	runnerPlanFP, err := plugin.BuildPlanFingerprint(ev.BuildPlan)
	if err != nil {
		return fail("runner plan is invalid: %v", err)
	}
	if runnerPlanFP != ev.BuildPlanFingerprint || runnerPlanFP != s1.PlanFingerprint {
		return fail("runner plan %s, stage 1 plan %s", runnerPlanFP, s1.PlanFingerprint)
	}
	if !ev.Reproducible || ev.FirstArtifactDigest != ev.SecondArtifactDigest {
		return fail("runner builds are not reproducible (%s, %s)", ev.FirstArtifactDigest, ev.SecondArtifactDigest)
	}
	if !rep.Reproducible || rep.PlanFingerprint != s1.PlanFingerprint || ev.FirstArtifactDigest != rep.ArtifactDigest {
		return fail("runner artifact %s, stage 2 stored %s", ev.FirstArtifactDigest, rep.ArtifactDigest)
	}
	wantLocked, err := plugin.LockBuildOutput(s1.Plan, rep.ArtifactDigest)
	if err != nil {
		return Stage3{}, err
	}
	if ev.LockedBuild != wantLocked {
		return fail("build artifact lock differs from the stage 1 plan and stage 2 artifact")
	}
	registered, err := lockedPlugin(ev.RuntimePluginLock, s1.Plan.Plugin)
	if err != nil {
		return fail("%v", err)
	}
	if registered.ArtifactDigest != rep.ArtifactDigest || registered.BuildPlanFingerprint != s1.PlanFingerprint {
		return fail("host registration %s / %s", registered.ArtifactDigest, registered.BuildPlanFingerprint)
	}
	tb := ev.TransformBootstrap
	if !tb.ConvergenceResult.Converged || tb.Stage1ArtifactDigest != tb.Stage2ArtifactDigest || !tb.SeedBuilderReproducible {
		return fail("transform builder stages did not converge")
	}
	if err := engineBoundToPackagePlan(tb); err != nil {
		return fail("%v", err)
	}
	if !tb.PackageReproducible || tb.PackageFirstArtifactDigest != tb.PackageOracleDigest || !ev.NativeReproducible || !ev.PackageReproducible {
		return fail("package or native realization is not reproducible")
	}
	return Stage3{
		PlanFingerprint: runnerPlanFP, FirstArtifactDigest: ev.FirstArtifactDigest, SecondArtifactDigest: ev.SecondArtifactDigest,
		RegisteredArtifactDigest: registered.ArtifactDigest, RegisteredPlanFingerprint: registered.BuildPlanFingerprint,
		RuntimePluginLockFingerprint: ev.RuntimePluginLockFingerprint,
		TransformEngineFingerprint:   tb.RuntimeEngineFingerprint, StaticResponseEngineFingerprint: ev.NativeRuntimeEngineFingerprint,
		BootstrapConverged: true,
		// The Go seed stays the builder of record; convergence does not erase it.
		SeedBuilderTrustRetained: ev.BuildPlan.Builder.ID == core.MustCanonicalID("tei.builder.go.seed"),
	}, nil
}

// engineBoundToPackagePlan checks that the package plan names the engine the
// bootstrap ran with and the builder digest it realized.
func engineBoundToPackagePlan(tb RunnerTransformBootstrap) error {
	plan := tb.PackageBuildPlan
	fp, err := plugin.BuildPlanFingerprint(plan)
	if err != nil || fp != tb.PackageBuildPlanFingerprint {
		return fmt.Errorf("package plan fingerprint does not recompute")
	}
	if plan.Builder.ArtifactDigest != tb.PackageBuilderArtifactDigest {
		return fmt.Errorf("package plan builder %s, bootstrap realized %s", plan.Builder.ArtifactDigest, tb.PackageBuilderArtifactDigest)
	}
	dependency := false
	for _, d := range plan.Dependencies {
		if d.Kind == "runtime-engine" && d.ID == string(tb.RuntimeEngine.ID) && d.ArtifactDigest == tb.RuntimeEngine.ImplementationDigest {
			dependency = true
		}
	}
	parameter := false
	for _, p := range plan.Parameters {
		if p.Name == "runtime-engine-fingerprint" && p.Value == tb.RuntimeEngineFingerprint {
			parameter = true
		}
	}
	if !dependency || !parameter {
		return fmt.Errorf("package plan does not bind the transform engine %s", tb.RuntimeEngineFingerprint)
	}
	return nil
}

func lockedPlugin(lock plugin.Lock, id plugin.Identity) (plugin.LockedPlugin, error) {
	var found []plugin.LockedPlugin
	for _, env := range lock.Environments {
		for _, p := range env.Plugins {
			if p.ID == id.ID && p.Version == id.Version {
				found = append(found, p)
			}
		}
	}
	if len(found) != 1 {
		return plugin.LockedPlugin{}, fmt.Errorf("runtime plugin lock has %d entries for %s@%s", len(found), id.ID, id.Version)
	}
	return found[0], nil
}
