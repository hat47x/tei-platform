package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	deploymentdurable "tei/internal/deployment/durable"
	"tei/internal/plugin"
	"tei/internal/validation"
	"tei/internal/workspace"
)

type artifact struct {
	Path string
	Data []byte
}

func main() {
	write := flag.Bool("write", false, "Canonical WorkspaceからReference goldenを更新する")
	workspacePath := flag.String("workspace", "reference/workspace", "Reference Workspace directory")
	worldPath := flag.String("world", "reference/world.json", "Reference World golden path")
	pluginLockPath := flag.String("plugin-lock", "reference/workspace/.tei/plugin-lock.json", "Reference Plugin Lock golden path")
	durableRunPath := flag.String("durable-run", "reference/order_rollout.durable-run.m0.21.start.json", "Plugin Lock依存を検査するReference Durable Run path。空文字で検査しない")
	flag.Parse()

	if err := run(*workspacePath, *worldPath, *pluginLockPath, *durableRunPath, *write); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(workspacePath, worldPath, pluginLockPath, durableRunPath string, write bool) error {
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		return fmt.Errorf("Reference Workspaceを読み込めません: %w", err)
	}

	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			return fmt.Errorf("Reference WorkspaceにValidation Errorがあります: %s: %s", diagnostic.Code, diagnostic.Message)
		}
	}

	worldJSON, err := encodeJSON(loaded.World)
	if err != nil {
		return fmt.Errorf("Reference WorldをJSONへ変換できません: %w", err)
	}

	lock, err := plugin.BuildLock(
		plugin.Catalog{Plugins: loaded.World.Plugins, Bundles: loaded.World.Bundles},
		loaded.World.Environments,
	)
	if err != nil {
		return fmt.Errorf("Reference Plugin Lockを生成できません: %w", err)
	}
	lockJSON, err := encodeJSON(lock)
	if err != nil {
		return fmt.Errorf("Reference Plugin LockをJSONへ変換できません: %w", err)
	}

	artifacts := []artifact{
		{Path: worldPath, Data: worldJSON},
		{Path: pluginLockPath, Data: lockJSON},
	}
	if write {
		if err := writeArtifacts(artifacts); err != nil {
			return err
		}
		if err := checkDurableRunPluginLock(durableRunPath, lock); err != nil {
			return fmt.Errorf(
				"Reference World / Plugin LockはCanonical Workspaceから更新しましたが、依存artifactが一致しません: %w",
				err,
			)
		}
		fmt.Println("Reference goldenをCanonical Workspaceから更新しました。git diffで意味上の差分を確認してください。")
		return nil
	}
	if err := checkArtifacts(artifacts); err != nil {
		return err
	}
	if err := checkDurableRunPluginLock(durableRunPath, lock); err != nil {
		return err
	}
	fmt.Println("Reference goldenと依存Durable RunはCanonical Workspaceと一致しています。")
	return nil
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

func checkArtifacts(artifacts []artifact) error {
	var stale []string
	for _, item := range artifacts {
		current, err := os.ReadFile(item.Path)
		if err != nil {
			if os.IsNotExist(err) {
				stale = append(stale, item.Path+"（未作成）")
				continue
			}
			return fmt.Errorf("Reference golden %sを読み込めません: %w", item.Path, err)
		}
		if !bytes.Equal(current, item.Data) {
			stale = append(stale, item.Path)
		}
	}
	if len(stale) == 0 {
		return nil
	}
	return fmt.Errorf(
		"Reference goldenがCanonical Workspaceと一致しません: %s\n更新する場合は `go run ./tools/reference-golden -write` を実行し、生成差分を確認してください",
		strings.Join(stale, ", "),
	)
}

func checkDurableRunPluginLock(path string, lock plugin.Lock) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Reference Durable Run %sがありません", path)
		}
		return fmt.Errorf("Reference Durable Run %sを読み込めません: %w", path, err)
	}
	var run deploymentdurable.Run
	if err := json.Unmarshal(raw, &run); err != nil {
		return fmt.Errorf("Reference Durable Run %sをJSONとして読み込めません: %w", path, err)
	}

	expected, err := plugin.EnvironmentLockFingerprint(lock, run.Environment)
	if err != nil {
		return fmt.Errorf("Reference Durable Run %sのEnvironment %sに対応するPlugin Lock fingerprintを生成できません: %w", path, run.Environment, err)
	}
	if run.PluginLockFingerprint == expected {
		return nil
	}
	return fmt.Errorf(
		"Reference Durable Run %sのPlugin Lock fingerprintが古い可能性があります: run=%s current=%s。fingerprintを手編集せず、`tei deployment-start`の正規経路からDurable Runを再生成してください",
		path,
		run.PluginLockFingerprint,
		expected,
	)
}

func writeArtifacts(artifacts []artifact) error {
	for _, item := range artifacts {
		if err := replaceArtifactSafely(item.Path, item.Data); err != nil {
			return err
		}
	}
	return nil
}

// replaceArtifactSafelyは既存goldenを直接truncateせず、同じdirectoryの一時fileへ
// 内容を完全に書き込み、Sync / Closeできた後にRenameする。
// 生成途中の失敗で既存artifactを空または途中までの内容にしないことが目的であり、
// OSをまたいだRenameのatomicityそのものは契約にしない。
func replaceArtifactSafely(path string, data []byte) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	temp, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return fmt.Errorf("Reference golden %sの一時fileを%sへ作成できません: %w", path, dir, err)
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
		return fmt.Errorf("Reference golden %sの一時file modeを設定できません: %w", path, err)
	}
	if _, err := temp.Write(data); err != nil {
		return fmt.Errorf("Reference golden %sの一時fileへ書き込めません: %w", path, err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("Reference golden %sの一時fileをsyncできません: %w", path, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("Reference golden %sの一時fileをcloseできません: %w", path, err)
	}
	closed = true
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("Reference golden %sを安全に置換できません: %w", path, err)
	}
	return nil
}
