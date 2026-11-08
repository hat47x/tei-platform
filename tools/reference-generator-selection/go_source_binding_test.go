package generatorselection

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #2595 EB-01/EB-04: 宣言したSource(manifest.Source)と、go buildへ実際に渡るファイル
// (goSourceFiles)が別の候補。
//
// EB-04の修正後は、宣言SourceがBuild対象に含まれない候補を評価前に拒否する。
// main.goがあれば他の.goファイルをビルドしない選択規則(generator.go混入の防止が
// 意図)は変えていない。
// 実行はTEI外の実Provider・Hostを使わず、合成した一時ディレクトリだけで行う。
func TestGoDeclaredSourceDiffersFromBuiltFileIsRejected(t *testing.T) {
	workspaceDir, contract, fixtures, candidates := paths(t)
	built, err := os.ReadFile(filepath.Join(candidates, "template", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	// 宣言だけされ、main.goがあるためビルドされないファイル。
	declared := []byte("package main\n\n// declared by the manifest, never passed to go build\nfunc main() {}\n")

	writeGo := func(t *testing.T, root, id, source string, files map[string][]byte, sourceBytes []byte) {
		t.Helper()
		dir := filepath.Join(root, id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for name, content := range files {
			if err := os.WriteFile(filepath.Join(dir, name), content, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		raw, err := json.Marshal(CandidateManifest{
			Schema: CandidateSchema, ID: id, Origin: "handwritten", Language: LanguageGo,
			Source: source, SourceDigest: digest(sourceBytes),
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "candidate.json"), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	root := t.TempDir()
	writeGo(t, root, "control", "main.go", map[string][]byte{"main.go": built}, built)
	writeGo(t, root, "mismatch", "audit.go", map[string][]byte{"audit.go": declared, "main.go": built}, declared)

	receipt, err := Select(context.Background(), workspaceDir, contract, fixtures, root)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]CandidateReceipt{}
	for _, candidate := range receipt.Candidates {
		byID[candidate.ID] = candidate
	}
	control, mismatch := byID["control"], byID["mismatch"]
	if !control.Passed {
		t.Fatalf("control candidate (declared == built) was not qualified: %+v", control)
	}
	// 宣言Source D1(audit.go)と、go buildへ渡るD2(main.go)が別の候補は拒否する。
	if mismatch.Passed {
		t.Fatalf("declared/built source mismatch was qualified: %+v", mismatch)
	}
	if !strings.Contains(mismatch.RejectReason, "not among the files passed to go build") {
		t.Fatalf("unexpected reject reason: %q", mismatch.RejectReason)
	}
	if mismatch.BuildEvidence != nil || mismatch.BinaryDigest != "" {
		t.Fatalf("rejected candidate must not carry a built binary: %+v", mismatch)
	}
}
