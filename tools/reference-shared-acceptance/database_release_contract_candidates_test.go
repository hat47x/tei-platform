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

// Databaseごとの最新Releaseが提供するContractのmajorが現在定義より古い場合を、
// 既存のDATABASE_RELEASE_CONTRACT_AHEAD_OF_CURRENT_DEFINITIONの逆方向として
// 読み取り専用比較が列挙するか実CLIで確認する。履歴Releaseは繰り返さない。
// 受理条件は変更しない。
func TestReferenceDatabaseReleaseContractCandidates(t *testing.T) {
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
	type candidate struct {
		Snapshot        string `json:"snapshot"`
		Database        string `json:"database"`
		Version         string `json:"version"`
		Contract        string `json:"contract"`
		ProvidedVersion string `json:"providedContractVersion"`
		CurrentVersion  string `json:"currentContractVersion"`
	}
	type report struct {
		Candidates []candidate `json:"databaseReleaseContractCandidates"`
	}
	compareReport := func(before, after string) report {
		t.Helper()
		var got report
		if err := json.Unmarshal(run(compare, true, "compare-application-membership", "-before", before, "-after", after), &got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	byKey := func(got report) map[string]candidate {
		out := map[string]candidate{}
		for _, c := range got.Candidates {
			out[c.Snapshot+"/"+c.Version] = c
		}
		return out
	}

	t.Run("baseline-has-no-behind-candidate", func(t *testing.T) {
		dir := scratchWorkspace(t)
		output := string(run(tei, true, "validate", "-workspace", dir))
		if strings.Contains(output, "DATABASE_RELEASE_CONTRACT_AHEAD_OF_CURRENT_DEFINITION") {
			t.Fatalf("現行Referenceに先行警告が出ています: %s", output)
		}
		got := compareReport(dir, dir)
		if len(got.Candidates) != 0 {
			t.Fatalf("現行Referenceではmajorが古い提供がないため候補0件を期待: %+v", got.Candidates)
		}
	})

	t.Run("major-update-reports-behind-candidates", func(t *testing.T) {
		before, after := scratchWorkspace(t), scratchWorkspace(t)
		replaceOnce(after, "databases/sales-core/contracts/order.tei", "contract sales_core.order v4.2", "contract sales_core.order v5.0")
		replaceOnce(after, "applications/order-entry/application.tei", "contract sales_core.order ^4", "contract sales_core.order ^5")
		run(tei, true, "validate", "-workspace", after)
		got := compareReport(before, after)
		if len(got.Candidates) != 1 {
			t.Fatalf("最新Releaseの4.2.0だけを期待: %+v", got.Candidates)
		}
		found := byKey(got)
		c, ok := found["after/4.2.0"]
		if !ok || c.ProvidedVersion != "4.2" || c.CurrentVersion != "5.0" || c.Contract != "sales_core.order" || c.Database != "sales_core" {
			t.Fatalf("4.2.0のbehindが正しくありません: %+v", got.Candidates)
		}
		if _, ok := found["after/4.1.0"]; ok {
			t.Fatalf("履歴Releaseの4.1.0を繰り返しています: %+v", got.Candidates)
		}
		if _, ok := found["before/4.2.0"]; ok {
			t.Fatalf("変更前の一致が候補になっています: %+v", got.Candidates)
		}
		t.Log("current v5.0: validate ok; only the latest release 4.2.0 is reported read-only as behind")
	})

	t.Run("forward-update-keeps-existing-warning-only", func(t *testing.T) {
		before, after := scratchWorkspace(t), scratchWorkspace(t)
		replaceOnce(after, "databases/sales-core/contracts/order.tei", "contract sales_core.order v4.2", "contract sales_core.order v3.0")
		replaceOnce(after, "applications/order-entry/application.tei", "contract sales_core.order ^4", "contract sales_core.order ^3")
		output := string(run(tei, true, "validate", "-workspace", after))
		for _, version := range []string{"4.1.0", "4.2.0"} {
			if !strings.Contains(output, "DATABASE_RELEASE_CONTRACT_AHEAD_OF_CURRENT_DEFINITION") || !strings.Contains(output, version) {
				t.Fatalf("%sの先行警告が出ていません: %s", version, output)
			}
		}
		got := compareReport(before, after)
		if len(got.Candidates) != 0 {
			t.Fatalf("先行方向は既存警告が扱うため候補0件を期待: %+v", got.Candidates)
		}
	})

	t.Run("minor-behind-is-not-a-major-candidate", func(t *testing.T) {
		before, after := scratchWorkspace(t), scratchWorkspace(t)
		replaceOnce(after, "databases/sales-core/contracts/order.tei", "contract sales_core.order v4.2", "contract sales_core.order v4.3")
		run(tei, true, "validate", "-workspace", after)
		got := compareReport(before, after)
		if len(got.Candidates) != 0 {
			t.Fatalf("minorだけ古い提供はmajor候補にしません: %+v", got.Candidates)
		}
	})
}
