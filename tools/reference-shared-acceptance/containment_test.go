package main

import (
	"os"
	"path/filepath"
	"testing"

	"tei/tools/internal/changeanalysis"
)

func TestExpectedDeltaCheckRejectsUnexpectedEffects(t *testing.T) {
	expected := []changeanalysis.SubjectChange{{Kind: "validation", ID: "sales.a", Change: "added"}}

	if got := checkExpectedDelta(expected, expected, []string{"semantics"}, []string{"semantics"}); len(got) != 0 {
		t.Fatalf("宣言どおりの差分が拒否されました: %+v", got)
	}

	extra := append([]changeanalysis.SubjectChange{{Kind: "attribute", ID: "sales.order.total", Change: "changed"}}, expected...)
	if got := checkExpectedDelta(extra, expected, []string{"semantics"}, []string{"semantics"}); len(got) != 1 || got[0].Code != "SEMANTIC_DIFF_MISMATCH" {
		t.Fatalf("宣言外のsemantic変更が検出されていません: %+v", got)
	}
	if got := checkExpectedDelta(nil, expected, []string{"semantics"}, []string{"semantics"}); len(got) != 1 || got[0].Code != "SEMANTIC_DIFF_MISMATCH" {
		t.Fatalf("宣言した変更が起きなかったことが検出されていません: %+v", got)
	}
	wrongChange := []changeanalysis.SubjectChange{{Kind: "validation", ID: "sales.a", Change: "changed"}}
	if got := checkExpectedDelta(wrongChange, expected, []string{"semantics"}, []string{"semantics"}); len(got) != 1 {
		t.Fatalf("変更の種類の違い(added / changed)が検出されていません: %+v", got)
	}

	for _, changed := range [][]string{{"applications", "semantics"}, {"persistenceMappings"}, nil} {
		got := checkExpectedDelta(expected, expected, changed, []string{"semantics"})
		if len(got) != 1 || got[0].Code != "UNEXPECTED_COLLECTION_CHANGE" {
			t.Fatalf("collection変更 %v が検出されていません: %+v", changed, got)
		}
	}
}

func TestContainmentAllowsOnlyTheSourceFilesTheDeltaEdits(t *testing.T) {
	baseDir := scratchWorkspace(t)
	before, err := takeSnapshot(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	edit := []SourceEdit{{Path: "semantic/sales/validation.tei"}}

	touch := func(dir, relative string) {
		t.Helper()
		path := filepath.Join(dir, filepath.FromSlash(relative))
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(content, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	snapshotAfter := func(relatives ...string) snapshot {
		t.Helper()
		dir := scratchWorkspace(t)
		for _, r := range relatives {
			touch(dir, r)
		}
		s, err := takeSnapshot(dir)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}

	if got := checkContainment(before, snapshotAfter("semantic/sales/validation.tei"), edit); len(got) != 0 {
		t.Fatalf("deltaどおりの変更が拒否されました: %+v", got)
	}

	codes := func(diags []Diagnostic) []string {
		var out []string
		for _, d := range diags {
			out = append(out, d.Code)
		}
		return out
	}
	stray := checkContainment(before, snapshotAfter("semantic/sales/validation.tei", "semantic/sales/order.tei"), edit)
	if !hasCode(codes(stray), "UNEXPECTED_SOURCE_CHANGE") {
		t.Fatalf("deltaに対応しないsource変更が検出されていません: %+v", stray)
	}
	otherLayer := checkContainment(before, snapshotAfter("semantic/sales/validation.tei", "applications/order-entry/application.tei"), edit)
	if !hasCode(codes(otherLayer), "LAYER_VIOLATION") || !hasCode(codes(otherLayer), "UNEXPECTED_SOURCE_CHANGE") {
		t.Fatalf("semantic層の外の変更が検出されていません: %+v", otherLayer)
	}
	missing := checkContainment(before, snapshotAfter(), edit)
	if !hasCode(codes(missing), "EXPECTED_SOURCE_CHANGE_MISSING") {
		t.Fatalf("deltaが変更するはずのfileが変わっていないことが検出されていません: %+v", missing)
	}
}
