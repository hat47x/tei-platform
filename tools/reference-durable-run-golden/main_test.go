package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/releaseplan"
)

func TestReplaceFileAtomicallyReplacesWholeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "durable-run.json")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	want := []byte("{\n  \"run\": \"new\"\n}\n")
	if err := replaceFileAtomically(path, want); err != nil {
		t.Fatalf("安全な置換に失敗しました: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("置換後の内容が不完全です: got=%q want=%q", got, want)
	}

	temps, err := filepath.Glob(filepath.Join(dir, ".durable-run.json.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temps) != 0 {
		t.Fatalf("一時fileが残っています: %#v", temps)
	}
}

func TestFindSolutionReleaseRequiresExactIDAndVersion(t *testing.T) {
	definitions := []releaseplan.Definition{
		{ID: core.MustCanonicalID("order_rollout"), Version: "2026.8.26"},
		{ID: core.MustCanonicalID("other_rollout"), Version: "2026.8.27"},
		{ID: core.MustCanonicalID("order_rollout"), Version: "2026.8.27"},
	}

	got, ok := findSolutionRelease(definitions, core.MustCanonicalID("order_rollout"), "2026.8.27")
	if !ok {
		t.Fatal("完全一致するSolution Releaseを解決できません")
	}
	if got.ID != core.MustCanonicalID("order_rollout") || got.Version != "2026.8.27" {
		t.Fatalf("別Solution Releaseを解決しています: %#v", got)
	}
}
