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

// Solution Release の database-gate が、現在のContract定義とどう照合されるかを
// 実CLIで観測する。gateは履歴Application Releaseの前提としてrelease graphで使われるが、
// validateが現在定義との存在・所有者・互換majorを検査するかを分けて確かめる。
// 受理条件は変更せず、実Deploymentのgate実行結果は示さない。
func TestReferenceSolutionReleaseGateAgainstCurrentDefinition(t *testing.T) {
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
	type result struct {
		stdout, stderr []byte
		ok             bool
	}
	run := func(bin string, wantOK bool, args ...string) result {
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
		return result{stdout: stdout.Bytes(), stderr: stderr.Bytes(), ok: ok}
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
	type compiled struct {
		Executable bool     `json:"executable"`
		Findings   []string `json:"findings"`
		Edges      []struct {
			From     string `json:"from"`
			To       string `json:"to"`
			Relation string `json:"relation"`
			Contract string `json:"contract"`
		} `json:"edges"`
	}
	graph := func(dir string) compiled {
		t.Helper()
		out := run(tei, true, "solution-release-graph", "-workspace", dir, "-solution", "order_rollout", "-version", "2026.8.27")
		var got compiled
		if err := json.Unmarshal(out.stdout, &got); err != nil {
			t.Fatalf("solution-release-graph JSON: %v\nstdout=%s\nstderr=%s", err, out.stdout, out.stderr)
		}
		return got
	}
	requireEdge := func(got compiled, want bool, from, to, contract string) {
		t.Helper()
		found := false
		for _, edge := range got.Edges {
			if edge.From == from && edge.To == to && edge.Contract == contract {
				found = true
				break
			}
		}
		if found != want {
			t.Fatalf("edge %s->%s (%s) の有無=%v、期待=%v: %+v", from, to, contract, found, want, got.Edges)
		}
	}

	t.Run("baseline-gate-consistent", func(t *testing.T) {
		dir := scratchWorkspace(t)
		validate := run(tei, true, "validate", "-workspace", dir)
		if bytes.Contains(validate.stderr, []byte("master_ready")) || bytes.Contains(validate.stdout, []byte("master_ready")) {
			t.Fatalf("整合したgateに診断が出ています: %s%s", validate.stdout, validate.stderr)
		}
		got := graph(dir)
		if !got.Executable || len(got.Findings) != 0 {
			t.Fatalf("gateが前提として解決していません: %+v", got)
		}
		requireEdge(got, true, "master_ready", "order_app", "master_data.customer")
	})

	t.Run("current-definition-ahead-of-gate-silent", func(t *testing.T) {
		dir := scratchWorkspace(t)
		replaceOnce(dir, "databases/master-data/contracts/customer.tei", "contract master_data.customer v2.4", "contract master_data.customer v3.0")
		replaceOnce(dir, "applications/order-entry/application.tei", "contract master_data.customer ^2", "contract master_data.customer ^3")
		replaceOnce(dir, "applications/repair-intake/application.tei", "contract master_data.customer ^2", "contract master_data.customer ^3")
		run(tei, true, "validate", "-workspace", dir)
		got := graph(dir)
		// gate ^2 は履歴Application Release 3.8.0の ^2 と一致するため解決する。
		// 現在定義はv3.0だが、validateもrelease graphもその不一致を診断しない。
		if !got.Executable || len(got.Findings) != 0 {
			t.Fatalf("gate ^2 が現在v3.0でも解決しています: %+v", got)
		}
		requireEdge(got, true, "master_ready", "order_app", "master_data.customer")
		t.Log("current contract v3.0 vs gate ^2: validate ok and release graph executable; mismatch stays silent")
	})

	t.Run("missing-prerequisite-gate-refused", func(t *testing.T) {
		dir := scratchWorkspace(t)
		replaceOnce(dir, "releases/solutions/order-rollout/2026.8.27.tei", "contract master_data.customer ^2", "contract master_data.customer_missing ^2")
		out := run(tei, false, "validate", "-workspace", dir)
		joined := append(out.stdout, out.stderr...)
		if !bytes.Contains(joined, []byte("SOLUTION_RELEASE_NOT_EXECUTABLE")) || !bytes.Contains(joined, []byte("no prerequisite step provides master_data.customer compatible major = 2")) {
			t.Fatalf("利用中のgate Contract欠落が拒否されていません: %s", joined)
		}
		t.Log("gate contract needed by an Application Release: validate refuses through release compilation")
	})

	t.Run("unused-unknown-gate-silent", func(t *testing.T) {
		dir := scratchWorkspace(t)
		replaceOnce(dir, "releases/solutions/order-rollout/2026.8.27.tei",
			"    after order_app sales_expand",
			"    step stale_gate {\n        kind database-gate\n        phase transition\n        database sales_core\n        contract sales_core.order_missing ^1\n    }\n\n    after order_app sales_expand")
		out := run(tei, true, "validate", "-workspace", dir)
		joined := append(out.stdout, out.stderr...)
		if bytes.Contains(joined, []byte("stale_gate")) || bytes.Contains(joined, []byte("order_missing")) {
			t.Fatalf("未使用gateの存在しないContractが診断されています: %s", joined)
		}
		t.Log("unused gate for a nonexistent contract: validate ok; declaration is not checked")
	})

	t.Run("gate-zero-major-rejected", func(t *testing.T) {
		dir := scratchWorkspace(t)
		replaceOnce(dir, "releases/solutions/order-rollout/2026.8.27.tei", "contract master_data.customer ^2", "contract master_data.customer ^0")
		out := run(tei, false, "validate", "-workspace", dir)
		joined := append(out.stdout, out.stderr...)
		if !bytes.Contains(joined, []byte("INCOMPLETE_DATABASE_GATE_STEP")) {
			t.Fatalf("gateの互換major 0が拒否されていません: %s", joined)
		}
		t.Log("gate compatible major 0: validate refuses with INCOMPLETE_DATABASE_GATE_STEP")
	})

	t.Run("gate-appears-in-contract-candidates", func(t *testing.T) {
		before, after := scratchWorkspace(t), scratchWorkspace(t)
		replaceOnce(after, "databases/master-data/contracts/customer.tei", "contract master_data.customer v2.4", "contract master_data.customer v2.5")
		run(tei, true, "validate", "-workspace", after)
		out := run(compare, true, "compare-application-membership", "-before", before, "-after", after)
		var report applicationMembershipComparison
		if err := json.Unmarshal(out.stdout, &report); err != nil {
			t.Fatalf("comparison JSON: %v\n%s", err, out.stdout)
		}
		found := map[string]int{}
		for _, candidate := range report.ContractCandidates {
			for _, consumer := range candidate.Consumers {
				if consumer.Kind == "solution-release-contract-gate" {
					found[consumer.Snapshot+"/"+consumer.ID]++
				}
			}
		}
		for _, snapshot := range []string{"before", "after"} {
			if found[snapshot+"/order_rollout@2026.8.27/master_ready"] != 1 {
				t.Fatalf("Solution Release gateの候補が%sにありません: %+v", snapshot, found)
			}
		}
		t.Log("master_data.customer version edit reports order_rollout@2026.8.27/master_ready in both snapshots")
	})
}
