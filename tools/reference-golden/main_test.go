package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
