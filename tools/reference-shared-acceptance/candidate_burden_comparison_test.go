package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

// 独立したWorkspace複製に対して代表的な変更を行い、読み取り専用候補が
// 意図した境界だけを報告するか（過剰報告・過少報告）を測る。
// 候補は受理条件ではないため、ここでの件数は判断の負担を比較する材料であり、
// 意味の正しさや実行時の安全性を示さない。
func TestReferenceCandidateViewBurdenComparison(t *testing.T) {
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
	run := func(bin string, args ...string) ([]byte, int) {
		t.Helper()
		cmd := exec.Command(bin, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		if err == nil {
			return stdout.Bytes(), 0
		}
		if exit, ok := err.(*exec.ExitError); ok {
			return stdout.Bytes(), exit.ExitCode()
		}
		t.Fatalf("%v: %v\nstderr=%s", args, err, stderr.String())
		return nil, -1
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
	type dbCandidate struct {
		Snapshot string `json:"snapshot"`
		Version  string `json:"version"`
		Contract string `json:"contract"`
	}
	type appCandidate struct {
		Snapshot    string `json:"snapshot"`
		Application string `json:"application"`
		Version     string `json:"version"`
		Contract    string `json:"contract"`
	}
	type gateCandidate struct {
		Snapshot string `json:"snapshot"`
		Step     string `json:"step"`
	}
	type report struct {
		Contracts      []json.RawMessage `json:"contractCandidates"`
		Gates          []gateCandidate   `json:"unreferencedUnsatisfiableGateCandidates"`
		AppReleases    []appCandidate    `json:"applicationReleaseContractCandidates"`
		DBReleases     []dbCandidate     `json:"databaseReleaseContractCandidates"`
		AuthorityLimit []string          `json:"authorityLimits"`
		AfterErrors    []struct {
			Severity string `json:"severity"`
		} `json:"afterDiagnostics"`
	}
	type counts struct{ contract, gate, appRelease, dbRelease int }
	decode := func(out []byte) (report, counts) {
		t.Helper()
		var r report
		if err := json.Unmarshal(out, &r); err != nil {
			t.Fatalf("比較結果を読めません: %v\n%s", err, out)
		}
		return r, counts{len(r.Contracts), len(r.Gates), len(r.AppReleases), len(r.DBReleases)}
	}
	compareWorkspace := func(before, after string, extra ...string) ([]byte, int, report, counts) {
		t.Helper()
		args := append([]string{"compare-application-membership", "-before", before, "-after", after}, extra...)
		out, exit := run(compare, args...)
		r, c := decode(out)
		return out, exit, r, c
	}
	versions := func(r report, snapshot string) []string {
		var out []string
		for _, c := range r.DBReleases {
			if c.Snapshot == snapshot {
				out = append(out, c.Version)
			}
		}
		sort.Strings(out)
		return out
	}
	equal := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}
	errorCount := func(r report) int {
		n := 0
		for _, d := range r.AfterErrors {
			if d.Severity == "error" {
				n++
			}
		}
		return n
	}

	baseDir := scratchWorkspace(t)
	if out, exit := run(tei, "validate", "-workspace", baseDir); exit != 0 {
		t.Fatalf("baseline validate が失敗しました: %s", out)
	}
	_, exit, baselineReport, baseline := compareWorkspace(baseDir, baseDir)
	if exit != 0 {
		t.Fatalf("baseline 比較の終了コード=%d", exit)
	}
	t.Logf("baseline: contract=%d gate=%d appRelease=%d dbRelease=%d", baseline.contract, baseline.gate, baseline.appRelease, baseline.dbRelease)
	if baseline.contract != 0 || baseline.dbRelease != 0 || baseline.gate != 0 || baseline.appRelease != 0 {
		t.Fatalf("現行Referenceの新しい候補は0件を期待: %+v", baseline)
	}
	if !contains(baselineReport.AuthorityLimit, "database-release-contract-candidates-only-major-behind-not-complete") {
		t.Fatalf("databaseRelease候補のauthorityLimitsがありません: %v", baselineReport.AuthorityLimit)
	}

	t.Run("unrelated-contract-minor-changes-no-new-candidate", func(t *testing.T) {
		before, after := scratchWorkspace(t), scratchWorkspace(t)
		replaceOnce(after, "databases/sales-core/contracts/order-line.tei", "contract sales_core.order_line v1.0", "contract sales_core.order_line v1.1")
		if out, exit := run(tei, "validate", "-workspace", after); exit != 0 {
			t.Fatalf("validate が失敗しました: %s", out)
		}
		_, exit, _, got := compareWorkspace(before, after)
		if exit != 0 {
			t.Fatalf("比較の終了コード=%d", exit)
		}
		if got.contract == 0 {
			t.Fatalf("無関係なContract変更が候補に出ていません: %+v", got)
		}
		if got.gate != baseline.gate || got.appRelease != baseline.appRelease || got.dbRelease != baseline.dbRelease {
			t.Fatalf("無関係な変更で新しい候補が変化しました: got=%+v baseline=%+v", got, baseline)
		}
		t.Logf("unrelated-contract-minor: contract=%d gate=%d appRelease=%d dbRelease=%d", got.contract, got.gate, got.appRelease, got.dbRelease)
	})

	t.Run("contract-minor-update-changes-no-new-candidate", func(t *testing.T) {
		before, after := scratchWorkspace(t), scratchWorkspace(t)
		replaceOnce(after, "databases/sales-core/contracts/order.tei", "contract sales_core.order v4.2", "contract sales_core.order v4.3")
		if out, exit := run(tei, "validate", "-workspace", after); exit != 0 {
			t.Fatalf("validate が失敗しました: %s", out)
		}
		_, exit, _, got := compareWorkspace(before, after)
		if exit != 0 {
			t.Fatalf("比較の終了コード=%d", exit)
		}
		if got.contract == 0 {
			t.Fatalf("Contract変更が候補に出ていません: %+v", got)
		}
		if got.gate != baseline.gate || got.appRelease != baseline.appRelease || got.dbRelease != baseline.dbRelease {
			t.Fatalf("minor更新で新しい候補が変化しました: got=%+v baseline=%+v", got, baseline)
		}
		t.Logf("contract-minor: contract=%d gate=%d appRelease=%d dbRelease=%d", got.contract, got.gate, got.appRelease, got.dbRelease)
	})

	t.Run("coordinated-major-update-reports-only-latest-database-release", func(t *testing.T) {
		before, after := scratchWorkspace(t), scratchWorkspace(t)
		replaceOnce(after, "databases/sales-core/contracts/order.tei", "contract sales_core.order v4.2", "contract sales_core.order v5.0")
		replaceOnce(after, "applications/order-entry/application.tei", "contract sales_core.order ^4", "contract sales_core.order ^5")
		if out, exit := run(tei, "validate", "-workspace", after); exit != 0 {
			t.Fatalf("validate が失敗しました: %s", out)
		}
		_, exit, r, got := compareWorkspace(before, after)
		if exit != 0 {
			t.Fatalf("比較の終了コード=%d", exit)
		}
		if got.dbRelease != baseline.dbRelease+1 || got.appRelease != baseline.appRelease+1 || got.gate != baseline.gate {
			t.Fatalf("期待した増分と違います: got=%+v baseline=%+v", got, baseline)
		}
		if want := []string{"4.2.0"}; !equal(versions(r, "after"), want) || len(versions(r, "before")) != 0 {
			t.Fatalf("behindの版が正しくありません: %+v", r.DBReleases)
		}
		found := false
		for _, c := range r.AppReleases {
			if c.Snapshot == "after" && c.Application == "order_entry" && c.Version == "3.8.0" && c.Contract == "sales_core.order" {
				found = true
			}
		}
		if !found {
			t.Fatalf("Application Releaseのmajor不一致が出ていません: %+v", r.AppReleases)
		}
		t.Logf("coordinated-major: contract=%d gate=%d appRelease=%d dbRelease=%d", got.contract, got.gate, got.appRelease, got.dbRelease)
	})

	t.Run("unused-gate-adds-only-one-gate-candidate", func(t *testing.T) {
		before, after := scratchWorkspace(t), scratchWorkspace(t)
		replaceOnce(after, "releases/solutions/order-rollout/2026.8.27.tei", "    after order_app sales_expand", `    step stale_gate {
        kind database-gate
        phase transition
        database master_data
        contract master_data.nonexistent ^7
    }

    after order_app sales_expand`)
		if out, exit := run(tei, "validate", "-workspace", after); exit != 0 {
			t.Fatalf("validate が失敗しました: %s", out)
		}
		_, exit, r, got := compareWorkspace(before, after)
		if exit != 0 {
			t.Fatalf("比較の終了コード=%d", exit)
		}
		if got.gate != baseline.gate+1 || got.appRelease != baseline.appRelease || got.dbRelease != baseline.dbRelease {
			t.Fatalf("期待した増分と違います: got=%+v baseline=%+v", got, baseline)
		}
		found := false
		for _, c := range r.Gates {
			if c.Snapshot == "after" && c.Step == "stale_gate" {
				found = true
			}
		}
		if !found {
			t.Fatalf("未使用gateが候補に出ていません: %+v", r.Gates)
		}
		t.Logf("unused-gate: contract=%d gate=%d appRelease=%d dbRelease=%d", got.contract, got.gate, got.appRelease, got.dbRelease)
	})

	t.Run("invalid-major-change-hides-candidates-unless-explicit", func(t *testing.T) {
		before, after := scratchWorkspace(t), scratchWorkspace(t)
		replaceOnce(after, "databases/sales-core/contracts/order.tei", "contract sales_core.order v4.2", "contract sales_core.order v5.0")
		if out, exit := run(tei, "validate", "-workspace", after); exit == 0 {
			t.Fatalf("要求を更新しないmajor変更はvalidate失敗を期待: %s", out)
		}
		out, exit, r, got := compareWorkspace(before, after)
		if exit != 1 || errorCount(r) == 0 {
			t.Fatalf("既定は終了1とエラー診断を期待: exit=%d errors=%d\n%s", exit, errorCount(r), out)
		}
		if got.contract != 0 || got.gate != 0 || got.appRelease != 0 || got.dbRelease != 0 {
			t.Fatalf("既定では未検証ワークスペースの候補を出しません: %+v", got)
		}
		out, exit, r, got = compareWorkspace(before, after, "-include-unvalidated-candidates")
		if exit != 1 || errorCount(r) == 0 {
			t.Fatalf("明示モードでも終了1とエラー診断を維持する必要があります: exit=%d\n%s", exit, out)
		}
		if got.dbRelease != baseline.dbRelease+1 || got.appRelease != baseline.appRelease+1 {
			t.Fatalf("明示モードの候補増分が違います: got=%+v baseline=%+v", got, baseline)
		}
		if want := []string{"4.2.0"}; !equal(versions(r, "after"), want) {
			t.Fatalf("明示モードのbehind版が正しくありません: %+v", r.DBReleases)
		}
		t.Logf("invalid-major(explicit): contract=%d gate=%d appRelease=%d dbRelease=%d", got.contract, got.gate, got.appRelease, got.dbRelease)
	})
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
