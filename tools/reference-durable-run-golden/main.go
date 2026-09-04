package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"tei/internal/core"
	"tei/internal/deployment"
	deploymentdurable "tei/internal/deployment/durable"
	"tei/internal/plugin"
	pluginruntime "tei/internal/plugin/runtime"
	"tei/internal/releaseplan"
	"tei/internal/validation"
	"tei/internal/workspace"
)

const (
	defaultWorkspacePath  = "reference/workspace"
	defaultDeploymentPath = "reference/observed/production.deployment-state.json"
	defaultPluginLockPath = "reference/workspace/.tei/plugin-lock.json"
	defaultDurableRunPath = "reference/order_rollout.durable-run.m0.21.start.json"
	defaultSolutionID     = "order_rollout"
	defaultVersion        = "2026.8.27"
	defaultRunID          = "order-rollout-prod-m0.21"
)

func main() {
	write := flag.Bool("write", false, "正規生成したDurable Runで保存goldenを安全に更新する")
	workspacePath := flag.String("workspace", defaultWorkspacePath, "Reference Workspace directory")
	deploymentPath := flag.String("deployment", defaultDeploymentPath, "fixed observed deployment state path")
	pluginLockPath := flag.String("plugin-lock", defaultPluginLockPath, "Reference Plugin Lock path")
	durableRunPath := flag.String("durable-run", defaultDurableRunPath, "Reference Durable Run golden path")
	solutionID := flag.String("solution", defaultSolutionID, "Reference Solution Release ID")
	version := flag.String("version", defaultVersion, "Reference Solution Release version")
	runID := flag.String("run-id", defaultRunID, "Reference Durable Run ID")
	flag.Parse()

	if err := run(
		*workspacePath,
		*deploymentPath,
		*pluginLockPath,
		*durableRunPath,
		*solutionID,
		*version,
		*runID,
		*write,
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(workspacePath, deploymentPath, pluginLockPath, durableRunPath, solutionID, version, runID string, write bool) error {
	generated, err := generateDurableRun(workspacePath, deploymentPath, pluginLockPath, solutionID, version, runID)
	if err != nil {
		return err
	}

	if !write {
		current, err := os.ReadFile(durableRunPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("Reference Durable Run %sがありません。更新する場合は-writeを指定してください", durableRunPath)
			}
			return fmt.Errorf("Reference Durable Run %sを読み込めません: %w", durableRunPath, err)
		}
		if !bytes.Equal(current, generated) {
			return fmt.Errorf("Reference Durable Run %sが正規deployment-start経路の生成結果と一致しません。fingerprintやJSONを手編集せず、差分を確認したうえで-writeを実行してください", durableRunPath)
		}
		fmt.Println("Reference Durable RunはWorkspace / fixed observed deployment state / Plugin Lockからの正規生成結果と一致しています。")
		return nil
	}

	if err := replaceFileAtomically(durableRunPath, generated); err != nil {
		return err
	}
	fmt.Printf("Reference Durable Run %sを正規生成結果から更新しました。git diffで意味上の差分を確認してください。\n", durableRunPath)
	return nil
}

func generateDurableRun(workspacePath, deploymentPath, pluginLockPath, solutionID, version, runID string) ([]byte, error) {
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("Reference Workspace %sを読み込めません: %w", workspacePath, err)
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity != validation.Error {
			continue
		}
		return nil, fmt.Errorf("Reference WorkspaceにValidation Errorがあります: %s: %s", diagnostic.Code, diagnostic.Message)
	}

	solution := core.MustCanonicalID(solutionID)
	sr, ok := findSolutionRelease(loaded.World.SolutionReleases, solution, version)
	if !ok {
		return nil, fmt.Errorf("Reference Solution Release %s@%sがありません", solutionID, version)
	}

	var state deployment.Snapshot
	if err := decodeJSONFile(deploymentPath, &state); err != nil {
		return nil, fmt.Errorf("observed deployment state %sを読み込めません: %w", deploymentPath, err)
	}
	for _, diagnostic := range validation.ValidateDeployment(loaded.World, state) {
		if diagnostic.Severity != validation.Error {
			continue
		}
		return nil, fmt.Errorf("observed deployment stateにValidation Errorがあります: %s: %s", diagnostic.Code, diagnostic.Message)
	}

	compiled := releaseplan.CompileWithDeployment(sr, loaded.World.ApplicationReleases, loaded.World.DatabaseReleases, state)

	var lock plugin.Lock
	if err := decodeJSONFile(pluginLockPath, &lock); err != nil {
		return nil, fmt.Errorf("Reference Plugin Lock %sを読み込めません: %w", pluginLockPath, err)
	}
	env, ok := findEnvironment(loaded.World.Environments, state.Environment)
	if !ok {
		return nil, fmt.Errorf("observed deployment stateのEnvironment %sがCanonical Worldにありません", state.Environment)
	}
	runtimePlan, err := pluginruntime.BuildPlan(
		plugin.Catalog{Plugins: loaded.World.Plugins, Bundles: loaded.World.Bundles},
		env,
		lock,
	)
	if err != nil {
		return nil, fmt.Errorf("Plugin Lock / Runtime PolicyをEnvironment %sへ解決できません: %w", state.Environment, err)
	}

	run := deploymentdurable.StartWithRuntimePolicy(
		runID,
		sr,
		compiled,
		state,
		runtimePlan.PluginLockFingerprint,
		runtimePlan.RuntimePolicyFingerprint,
	)
	return encodeJSON(run)
}

func decodeJSONFile(path string, dst any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(dst)
}

func encodeJSON(value any) ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func replaceFileAtomically(path string, data []byte) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	temp, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return fmt.Errorf("Reference Durable Runの一時fileを%sへ作成できません: %w", dir, err)
	}
	tempPath := temp.Name()
	closed := false
	defer func() {
		if !closed {
			_ = temp.Close()
		}
		_ = os.Remove(tempPath)
	}()

	if err := temp.Chmod(0o644); err != nil {
		return fmt.Errorf("Reference Durable Run一時fileのmodeを設定できません: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		return fmt.Errorf("Reference Durable Run一時fileへ書き込めません: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("Reference Durable Run一時fileをsyncできません: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("Reference Durable Run一時fileをcloseできません: %w", err)
	}
	closed = true
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("Reference Durable Runを安全に置換できません: %w", err)
	}
	return nil
}

func findSolutionRelease(definitions []releaseplan.Definition, id core.CanonicalID, version string) (releaseplan.Definition, bool) {
	for _, definition := range definitions {
		if definition.ID == id && definition.Version == version {
			return definition, true
		}
	}
	return releaseplan.Definition{}, false
}

func findEnvironment(environments []pluginEnvironment, id core.CanonicalID) (pluginEnvironment, bool) {
	for _, environment := range environments {
		if environment.ID == id {
			return environment, true
		}
	}
	return pluginEnvironment{}, false
}

// pluginEnvironmentはEnvironmentの必要部分だけを再定義しないためのaliasである。
// 実体はenvironment.Environmentをそのまま利用する。
type pluginEnvironment = environment.Environment
