package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

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
	flag.Parse()

	if err := run(*workspacePath, *worldPath, *pluginLockPath, *write); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(workspacePath, worldPath, pluginLockPath string, write bool) error {
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
		fmt.Println("Reference goldenをCanonical Workspaceから更新しました。git diffで意味上の差分を確認してください。")
		return nil
	}
	if err := checkArtifacts(artifacts); err != nil {
		return err
	}
	fmt.Println("Reference goldenはCanonical Workspaceと一致しています。")
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

func writeArtifacts(artifacts []artifact) error {
	for _, item := range artifacts {
		if err := os.WriteFile(item.Path, item.Data, 0o644); err != nil {
			return fmt.Errorf("Reference golden %sを更新できません: %w", item.Path, err)
		}
	}
	return nil
}
