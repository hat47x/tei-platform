package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// WindowsのEditorが付け得るUTF-8 BOMを、Sourceの読込みから定義の解釈までで
// 一度だけ無視できるかを実CLIで確かめる。バイト位置を使う編集経路は変えず、
// BOM以外の不正byteは従来どおり拒否する。実Providerの実行や他Toolの編集結果は示さない。
func TestReferenceBOMSourceLoadsAndInvalidByteStillRefused(t *testing.T) {
	original, dir := scratchWorkspace(t), scratchWorkspace(t)
	binDir := t.TempDir()
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	tei, compare := filepath.Join(binDir, "tei"+ext), filepath.Join(binDir, "compare"+ext)
	for _, build := range []struct{ output, pkg string }{{tei, "../../cmd/tei"}, {compare, "."}} {
		if output, err := exec.Command("go", "build", "-o", build.output, build.pkg).CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", build.pkg, err, output)
		}
	}
	run := func(bin string, wantOK bool, args ...string) []byte {
		t.Helper()
		cmd := exec.Command(bin, args...)
		var out, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &stderr
		err := cmd.Run()
		ok := err == nil
		if err != nil {
			if _, isExit := err.(*exec.ExitError); !isExit {
				t.Fatal(err)
			}
		}
		if ok != wantOK {
			t.Fatalf("%v の成功=%v、期待=%v\nstdout=%s\nstderr=%s", args, ok, wantOK, out.String(), stderr.String())
		}
		return out.Bytes()
	}
	lock := func(workspace string) []byte {
		t.Helper()
		var envelope struct {
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(run(tei, true, "plugin-lock", "-workspace", workspace), &envelope); err != nil || len(envelope.Result) == 0 {
			t.Fatalf("Lockを取得できません: %v", err)
		}
		return envelope.Result
	}
	baseline := lock(original)

	path := filepath.Join(dir, "semantic", "sales", "order.tei")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, "\ufeff"+string(content))
	run(tei, true, "validate", "-workspace", dir)
	run(tei, true, "plugin-resolve", "-workspace", dir, "-environment", "development")
	if withBOM := lock(dir); !bytes.Equal(withBOM, baseline) {
		t.Fatalf("BOMの付与でLockが変わりました:\nbefore=%s\nafter=%s", baseline, withBOM)
	}

	// BOMはSourceの前後関係を変えない。Worldの集合差分が空であることを確かめる。
	var comparison struct {
		ChangedCollections []string          `json:"changedCollections"`
		StrictChanges      []json.RawMessage `json:"strictChanges"`
		SemanticChanges    []json.RawMessage `json:"semanticChanges"`
		ContractCandidates []json.RawMessage `json:"contractCandidates"`
		ComparisonError    string            `json:"comparisonError"`
	}
	raw := run(compare, true, "compare-application-membership", "-before", original, "-after", dir)
	if err := json.Unmarshal(raw, &comparison); err != nil {
		t.Fatal(err)
	}
	if comparison.ComparisonError != "" || len(comparison.ChangedCollections) != 0 || len(comparison.StrictChanges) != 0 || len(comparison.SemanticChanges) != 0 || len(comparison.ContractCandidates) != 0 {
		t.Fatalf("BOMの付与でWorldが変わりました: %+v", comparison)
	}

	// BOMを除いた不正byteは、BOMの許容に巻き込まず拒否する。
	writeFile(t, path, string(content)+"\x00")
	run(tei, false, "validate", "-workspace", dir)
}
