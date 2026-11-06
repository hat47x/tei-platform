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

// どのApplication Releaseからも前提として使われず、現在のContract定義も
// Database Releaseも満たせないdatabase-gateを、読み取り専用比較が列挙するか
// を実CLIで確認する。受理条件は変更しない。
func TestReferenceUnreferencedUnsatisfiableGateCandidates(t *testing.T) {
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
	const solutionRel = "releases/solutions/order-rollout/2026.8.27.tei"
	addStaleGate := func(dir, step, database, contract, major string) {
		t.Helper()
		replaceOnce(dir, solutionRel, "    after order_app sales_expand",
			"    step "+step+" {\n        kind database-gate\n        phase transition\n        database "+database+"\n        contract "+contract+" ^"+major+"\n    }\n\n    after order_app sales_expand")
	}
	type candidates struct {
		GateCandidates []struct {
			Snapshot        string `json:"snapshot"`
			Solution        string `json:"solution"`
			Version         string `json:"version"`
			Step            string `json:"step"`
			Database        string `json:"database"`
			Contract        string `json:"contract"`
			CompatibleMajor int    `json:"compatibleMajor"`
		} `json:"unreferencedUnsatisfiableGateCandidates"`
	}
	report := func(before, after string) candidates {
		t.Helper()
		var got candidates
		if err := json.Unmarshal(run(compare, true, "compare-application-membership", "-before", before, "-after", after), &got); err != nil {
			t.Fatal(err)
		}
		return got
	}

	t.Run("consistent-reference-has-no-candidates", func(t *testing.T) {
		dir := scratchWorkspace(t)
		run(tei, true, "validate", "-workspace", dir)
		if got := report(dir, dir); len(got.GateCandidates) != 0 {
			t.Fatalf("整合したgateが候補になっています: %+v", got.GateCandidates)
		}
	})

	t.Run("unreferenced-unsatisfiable-gate-reported", func(t *testing.T) {
		before, after := scratchWorkspace(t), scratchWorkspace(t)
		addStaleGate(after, "stale_gate", "master_data", "master_data.nonexistent", "7")
		run(tei, true, "validate", "-workspace", after)
		got := report(before, after)
		if len(got.GateCandidates) != 1 {
			t.Fatalf("未使用・不成立のgateが1件ではありません: %+v", got.GateCandidates)
		}
		c := got.GateCandidates[0]
		if c.Snapshot != "after" || c.Solution != "order_rollout" || c.Version != "2026.8.27" || c.Step != "stale_gate" ||
			c.Database != "master_data" || c.Contract != "master_data.nonexistent" || c.CompatibleMajor != 7 {
			t.Fatalf("候補のidentityが欠けています: %+v", c)
		}
		t.Log("unused gate for a nonexistent contract: validate ok; comparison reports it read-only")
	})

	t.Run("unreferenced-but-current-definition-satisfies", func(t *testing.T) {
		before, after := scratchWorkspace(t), scratchWorkspace(t)
		addStaleGate(after, "stale_gate", "sales_core", "sales_core.order_line", "1")
		run(tei, true, "validate", "-workspace", after)
		if got := report(before, after); len(got.GateCandidates) != 0 {
			t.Fatalf("現在定義で満たせるgateが候補になっています: %+v", got.GateCandidates)
		}
	})

	t.Run("removed-gate-kept-in-before-snapshot", func(t *testing.T) {
		before, after := scratchWorkspace(t), scratchWorkspace(t)
		addStaleGate(before, "stale_gate", "master_data", "master_data.nonexistent", "7")
		run(tei, true, "validate", "-workspace", before)
		got := report(before, after)
		if len(got.GateCandidates) != 1 || got.GateCandidates[0].Snapshot != "before" || got.GateCandidates[0].Step != "stale_gate" {
			t.Fatalf("削除前のgateが残っていません: %+v", got.GateCandidates)
		}
	})
}
