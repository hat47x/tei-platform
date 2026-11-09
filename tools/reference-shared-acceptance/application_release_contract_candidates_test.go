package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Application ReleaseのContract要求の互換majorが現在定義とずれる場合を、
// 読み取り専用比較が列挙するか実CLIで確認する。現在定義にない要求は既存の
// APPLICATION_RELEASE_CONTRACT_NOT_CURRENT が扱うため候補にしない。
// 受理条件は変更しない。
func TestReferenceApplicationReleaseContractCandidates(t *testing.T) {
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
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		ok := err == nil
		if err != nil {
			if _, isExit := err.(*exec.ExitError); !isExit {
				t.Fatal(err)
			}
		}
		if ok != wantOK {
			t.Fatalf("%v の成功=%v、期待=%v\nstdout=%s\nstderr=%s", args, ok, wantOK, stdout.String(), stderr.String())
		}
		return stdout.Bytes()
	}
	replaceOnce := func(root, rel, old, next string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Count(content, []byte(old)) != 1 {
			t.Fatalf("%sの置換対象が1件ではありません: %q", rel, old)
		}
		writeFile(t, path, string(bytes.Replace(content, []byte(old), []byte(next), 1)))
	}
	type mismatch struct {
		Snapshot        string `json:"snapshot"`
		Application     string `json:"application"`
		Version         string `json:"version"`
		Alias           string `json:"alias"`
		Database        string `json:"database"`
		Contract        string `json:"contract"`
		CompatibleMajor int    `json:"compatibleMajor"`
		CurrentVersion  string `json:"currentContractVersion"`
	}
	type report struct {
		Candidates []mismatch `json:"applicationReleaseContractCandidates"`
	}
	compareReport := func(before, after string) report {
		t.Helper()
		var got report
		if err := json.Unmarshal(run(compare, true, "compare-application-membership", "-before", before, "-after", after), &got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	byKey := func(got report) map[string]mismatch {
		out := map[string]mismatch{}
		for _, c := range got.Candidates {
			out[c.Snapshot+"/"+c.Application+"@"+c.Version+"/"+c.Contract] = c
		}
		return out
	}

	t.Run("baseline-defers-missing-contract-to-existing-warning", func(t *testing.T) {
		dir := scratchWorkspace(t)
		output := string(run(tei, true, "validate", "-workspace", dir))
		if !strings.Contains(output, "APPLICATION_RELEASE_CONTRACT_NOT_CURRENT") || !strings.Contains(output, "sales_core.order_legacy") {
			t.Fatalf("既存の履歴Contract警告が出ていません: %s", output)
		}
		got := compareReport(dir, dir)
		if len(got.Candidates) != 0 {
			t.Fatalf("既存警告と重複する候補を出さないでください: %+v", got.Candidates)
		}
	})

	t.Run("major-update-reports-mismatch", func(t *testing.T) {
		before, after := scratchWorkspace(t), scratchWorkspace(t)
		replaceOnce(after, "databases/sales-core/contracts/order.tei", "contract sales_core.order v4.2", "contract sales_core.order v5.0")
		replaceOnce(after, "applications/order-entry/application.tei", "contract sales_core.order ^4", "contract sales_core.order ^5")
		run(tei, true, "validate", "-workspace", after)
		got := compareReport(before, after)
		found := byKey(got)
		c, ok := found["after/order_entry@3.8.0/sales_core.order"]
		if !ok || c.CompatibleMajor != 4 || c.CurrentVersion != "5.0" {
			t.Fatalf("major不一致が正しく出ていません: %+v", got.Candidates)
		}
		if _, ok := found["before/order_entry@3.8.0/sales_core.order"]; ok {
			t.Fatalf("変更前の一致が候補になっています: %+v", got.Candidates)
		}
		if len(got.Candidates) != 1 {
			t.Fatalf("afterのmajor不一致1件だけを期待: %+v", got.Candidates)
		}
		t.Log("current v5.0 vs Application Release ^4: validate ok; comparison reports the mismatch read-only")
	})

	t.Run("minor-update-adds-no-major-mismatch", func(t *testing.T) {
		before, after := scratchWorkspace(t), scratchWorkspace(t)
		replaceOnce(after, "databases/sales-core/contracts/order.tei", "contract sales_core.order v4.2", "contract sales_core.order v4.3")
		run(tei, true, "validate", "-workspace", after)
		got := compareReport(before, after)
		if len(got.Candidates) != 0 {
			t.Fatalf("minor更新でmajor不一致が増えています: %+v", got.Candidates)
		}
		if _, ok := byKey(got)["after/order_entry@3.8.0/sales_core.order"]; ok {
			t.Fatalf("minor更新が候補になっています: %+v", got.Candidates)
		}
	})
}
