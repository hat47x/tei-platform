package generatorselection

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// #2595 EB-01: 宣言したSource(manifest.Source)と、go buildへ実際に渡るファイル
// (goSourceFiles)が別でも、現行のSelectは候補を拒否しない。
//
// このテストは現状を記録する特性化試験である。main.goがあれば他の.goファイルを
// ビルドしない選択規則(generator.go混入の防止が意図)は変えていない。
// 修正(#2595 EB-04)を入れたら、この試験の期待を拒否へ反転させる。
// 実行はTEI外の実Provider・Hostを使わず、合成した一時ディレクトリだけで行う。
func TestGoDeclaredSourceDiffersFromBuiltFileIsCurrentlyAccepted(t *testing.T) {
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
	if control.BuildEvidence == nil || mismatch.BuildEvidence == nil {
		t.Fatalf("build evidence is missing: control=%+v mismatch=%+v", control.BuildEvidence, mismatch.BuildEvidence)
	}

	// 現状: 宣言Source D1(audit.go)の候補が、実際にはD2(main.go)から作られた
	// 同一binaryで評価され、合格している。
	if !mismatch.Passed {
		t.Fatalf("current behavior changed: the declared/built mismatch is now rejected (%q); invert this test and record it on #2595", mismatch.RejectReason)
	}
	if mismatch.BinaryDigest != control.BinaryDigest {
		t.Fatalf("mismatch binary %s differs from the control binary %s", mismatch.BinaryDigest, control.BinaryDigest)
	}
	if mismatch.SourceDigest != digest(declared) || mismatch.SourceDigest == control.SourceDigest {
		t.Fatalf("receipt SourceDigest does not carry the declared digest: %+v", mismatch)
	}
	if mismatch.BuildEvidence.SourceDigest != digest(declared) {
		t.Fatalf("build evidence SourceDigest=%s want the declared %s", mismatch.BuildEvidence.SourceDigest, digest(declared))
	}
	if mismatch.BuildEvidence.BuildInputDigest == control.BuildEvidence.BuildInputDigest {
		t.Fatal("BuildInputDigest should differ because it is derived from the declared source, not the built file")
	}
}
