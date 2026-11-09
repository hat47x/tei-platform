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

// Contract版の変更をRelease記録と突き合わせ、現行のvalidateが何を拒否し、
// 何を黙って通すかを実CLIで観測する。Release記録は比較の候補にだけ現れ、
// 受理条件は変更しない。実ProviderやDeploymentの実行結果は示さない。
func TestReferenceContractChangeAgainstReleaseRecords(t *testing.T) {
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
		return append(out.Bytes(), stderr.Bytes()...)
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
	requireLine := func(out []byte, want bool, code, subject string) {
		t.Helper()
		found := false
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, code) && strings.Contains(line, subject) {
				found = true
				break
			}
		}
		if found != want {
			t.Fatalf("%s に %s が含まれる=%v、期待=%v\n%s", code, subject, found, want, out)
		}
	}

	t.Run("coordinated-major-update", func(t *testing.T) {
		before, after := scratchWorkspace(t), scratchWorkspace(t)
		const contractRel = "databases/sales-core/contracts/order.tei"
		replaceOnce(after, contractRel, "contract sales_core.order v4.2", "contract sales_core.order v5.0")
		replaceOnce(after, "applications/order-entry/application.tei", "contract sales_core.order ^4", "contract sales_core.order ^5")

		out := run(tei, true, "validate", "-workspace", after)
		// 現行のApplication要求は満たすため通る。一方、Release記録側に残る ^4 と
		// 提供版 4.x は診断されない。ここではその不在を明示的に観測する。
		requireLine(out, false, "APPLICATION_RELEASE", "order_entry@3.8.0")
		requireLine(out, false, "DATABASE_RELEASE", "sales_core release 4.2.0")
		requireLine(out, true, "DATABASE_RELEASE_CONTRACT_NOT_CURRENT", "sales_core release 4.0.0")

		var report applicationMembershipComparison
		raw := run(compare, true, "compare-application-membership", "-before", before, "-after", after)
		if err := json.Unmarshal(raw, &report); err != nil {
			t.Fatalf("comparison JSON: %v\n%s", err, raw)
		}
		if len(report.ContractCandidates) != 1 || report.ContractCandidates[0].Contract != "sales_core.order" {
			t.Fatalf("unexpected contract candidates: %+v", report.ContractCandidates)
		}
		found := map[string]int{}
		for _, consumer := range report.ContractCandidates[0].Consumers {
			found[consumer.Snapshot+"/"+consumer.Kind]++
		}
		for _, kind := range []string{"application-release-contract-requirement", "database-release-contract-provided"} {
			if found["before/"+kind] == 0 || found["after/"+kind] == 0 {
				t.Fatalf("Release記録の候補 %s が前後にありません: %+v", kind, found)
			}
		}
		if !bytes.Contains(raw, []byte("order_entry@3.8.0/sales")) || !bytes.Contains(raw, []byte("sales_core@4.2.0")) {
			t.Fatalf("Release記録のidentityが候補にありません: %s", raw)
		}
		t.Log("coordinated major update: validate ok; Release records stay silent; comparison lists them as candidates")
	})

	t.Run("uncoordinated-major-update-refused", func(t *testing.T) {
		dir := scratchWorkspace(t)
		replaceOnce(dir, "databases/sales-core/contracts/order.tei", "contract sales_core.order v4.2", "contract sales_core.order v5.0")
		out := run(tei, false, "validate", "-workspace", dir)
		requireLine(out, true, "DATABASE_CONTRACT_VERSION_UNSATISFIED", "application order_entry requires sales_core.order")
		t.Log("uncoordinated major update: validate refuses with DATABASE_CONTRACT_VERSION_UNSATISFIED")
	})

	t.Run("release-ahead-of-current-definition-warns", func(t *testing.T) {
		dir := scratchWorkspace(t)
		replaceOnce(dir, "databases/sales-core/contracts/order.tei", "contract sales_core.order v4.2", "contract sales_core.order v3.0")
		replaceOnce(dir, "applications/order-entry/application.tei", "contract sales_core.order ^4", "contract sales_core.order ^3")
		out := run(tei, true, "validate", "-workspace", dir)
		// 逆方向（Release記録が現行定義より新しい）は既存の警告がある。
		// この非対称性が「現行定義が先へ進んだ場合だけ無言」であることの対照になる。
		requireLine(out, true, "DATABASE_RELEASE_CONTRACT_AHEAD_OF_CURRENT_DEFINITION", "sales_core release 4.2.0")
		// Application Releaseの ^4 は現行 ^3 と衝突するが、こちらは診断されない。
		requireLine(out, false, "APPLICATION_RELEASE", "order_entry@3.8.0")
		t.Log("release ahead of current definition: database warning exists; application release mismatch stays silent")
	})

	t.Run("minor-update-keeps-release-candidates", func(t *testing.T) {
		before, after := scratchWorkspace(t), scratchWorkspace(t)
		replaceOnce(after, "databases/sales-core/contracts/order.tei", "contract sales_core.order v4.2", "contract sales_core.order v4.3")
		run(tei, true, "validate", "-workspace", after)
		var report applicationMembershipComparison
		if err := json.Unmarshal(run(compare, true, "compare-application-membership", "-before", before, "-after", after), &report); err != nil {
			t.Fatal(err)
		}
		var kinds []string
		for _, candidate := range report.ContractCandidates {
			for _, consumer := range candidate.Consumers {
				kinds = append(kinds, consumer.Snapshot+"/"+consumer.Kind)
			}
		}
		joined := strings.Join(kinds, ",")
		if !strings.Contains(joined, "after/database-release-contract-provided") || !strings.Contains(joined, "after/application-release-contract-requirement") {
			t.Fatalf("minor更新でRelease候補が出ていません: %s", joined)
		}
	})
}
