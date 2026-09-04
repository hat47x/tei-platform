package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	deploymentdurable "tei/internal/deployment/durable"
	"tei/internal/plugin"
)

func TestEncodeJSONMatchesCLIFormat(t *testing.T) {
	encoded, err := encodeJSON(struct {
		Value string `json:"value"`
	}{Value: "reference"})
	if err != nil {
		t.Fatalf("JSONを生成できません: %v", err)
	}
	want := "{\n  \"value\": \"reference\"\n}\n"
	if string(encoded) != want {
		t.Fatalf("CLIと異なるJSON形式です:\n%s\nwant:\n%s", encoded, want)
	}
}

func TestCheckArtifactsReportsStaleAndMissingGolden(t *testing.T) {
	dir := t.TempDir()
	stalePath := filepath.Join(dir, "world.json")
	missingPath := filepath.Join(dir, "plugin-lock.json")
	if err := os.WriteFile(stalePath, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("fixtureを書き込めません: %v", err)
	}

	err := checkArtifacts([]artifact{
		{Path: stalePath, Data: []byte("new\n")},
		{Path: missingPath, Data: []byte("lock\n")},
	})
	if err == nil {
		t.Fatal("stale / missing goldenを一致扱いしています")
	}
	message := err.Error()
	if !strings.Contains(message, stalePath) || !strings.Contains(message, missingPath+"（未作成）") {
		t.Fatalf("不一致artifactを特定できません: %s", message)
	}
	if !strings.Contains(message, "go run ./tools/reference-golden -write") {
		t.Fatalf("正規更新手順を案内していません: %s", message)
	}
}

func TestWriteArtifactsMakesGoldenCheckPass(t *testing.T) {
	dir := t.TempDir()
	worldPath := filepath.Join(dir, "world.json")
	lockPath := filepath.Join(dir, "plugin-lock.json")
	artifacts := []artifact{
		{Path: worldPath, Data: []byte("world\n")},
		{Path: lockPath, Data: []byte("lock\n")},
	}

	if err := writeArtifacts(artifacts); err != nil {
		t.Fatalf("goldenを更新できません: %v", err)
	}
	if err := checkArtifacts(artifacts); err != nil {
		t.Fatalf("更新直後のgoldenが不一致です: %v", err)
	}

	world, err := os.ReadFile(worldPath)
	if err != nil {
		t.Fatalf("world goldenを読み込めません: %v", err)
	}
	lock, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("plugin lock goldenを読み込めません: %v", err)
	}
	if string(world) != "world\n" || string(lock) != "lock\n" {
		t.Fatalf("生成内容が変わっています: world=%q lock=%q", world, lock)
	}
}

func TestReplaceArtifactSafelyReplacesExistingGoldenWithoutTempResidue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "world.json")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("既存golden fixtureを書き込めません: %v", err)
	}

	if err := replaceArtifactSafely(path, []byte("new\n")); err != nil {
		t.Fatalf("既存goldenを安全に置換できません: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("置換後goldenを読み込めません: %v", err)
	}
	if string(got) != "new\n" {
		t.Fatalf("置換後golden=%q want=%q", got, "new\n")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("golden directoryを確認できません: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".world.json.tmp-") {
			t.Fatalf("safe replace後に一時fileが残っています: %s", entry.Name())
		}
	}
}

func TestCheckDurableRunPluginLockAcceptsMatchingFingerprint(t *testing.T) {
	lock := referencePluginLock()
	fingerprint, err := plugin.EnvironmentLockFingerprint(lock, "production")
	if err != nil {
		t.Fatalf("Plugin Lock fingerprintを生成できません: %v", err)
	}

	runPath := filepath.Join(t.TempDir(), "run.json")
	runJSON, err := encodeJSON(deploymentdurable.Run{
		Environment:           "production",
		PluginLockFingerprint: fingerprint,
	})
	if err != nil {
		t.Fatalf("Durable Run fixtureを生成できません: %v", err)
	}
	if err := os.WriteFile(runPath, runJSON, 0o644); err != nil {
		t.Fatalf("Durable Run fixtureを書き込めません: %v", err)
	}

	if err := checkDurableRunPluginLock(runPath, lock); err != nil {
		t.Fatalf("一致するPlugin Lock依存を不一致扱いしています: %v", err)
	}
}

func TestCheckDurableRunPluginLockRejectsStaleFingerprint(t *testing.T) {
	lock := referencePluginLock()
	runPath := filepath.Join(t.TempDir(), "run.json")
	runJSON, err := encodeJSON(deploymentdurable.Run{
		Environment:           "production",
		PluginLockFingerprint: "stale-fingerprint",
	})
	if err != nil {
		t.Fatalf("Durable Run fixtureを生成できません: %v", err)
	}
	if err := os.WriteFile(runPath, runJSON, 0o644); err != nil {
		t.Fatalf("Durable Run fixtureを書き込めません: %v", err)
	}

	err = checkDurableRunPluginLock(runPath, lock)
	if err == nil {
		t.Fatal("古いPlugin Lock fingerprintを一致扱いしています")
	}
	message := err.Error()
	if !strings.Contains(message, "stale-fingerprint") || !strings.Contains(message, "tei deployment-start") {
		t.Fatalf("stale依存と正規再生成経路を説明できていません: %s", message)
	}
}

func TestCheckDurableRunPluginLockCanBeDisabled(t *testing.T) {
	if err := checkDurableRunPluginLock("", referencePluginLock()); err != nil {
		t.Fatalf("空pathでの依存検査無効化に失敗しました: %v", err)
	}
}

func referencePluginLock() plugin.Lock {
	return plugin.Lock{
		FormatVersion: plugin.LockFormatVersion,
		Environments: []plugin.EnvironmentLock{
			{
				Environment: "production",
				Plugins: []plugin.LockedPlugin{
					{ID: "tei.database.postgres", Version: "1.0.0", Fingerprint: "postgres-definition"},
				},
			},
		},
	}
}
