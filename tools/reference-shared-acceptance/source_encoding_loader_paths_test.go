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

// 先頭BOMの許容が意味定義だけの偶然ではなく、各domain loaderで同じかを実CLIで確かめる。
// 対象は現在のreference/workspaceの複製であり、他Workspaceや実Providerの動作は示さない。
func TestReferenceBOMAcrossLoaderPaths(t *testing.T) {
	original := scratchWorkspace(t)
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

	for _, tc := range []struct{ loader, path string }{
		{"application", "applications/order-entry/application.tei"},
		{"database", "databases/sales-core/database.tei"},
		{"contract", "databases/sales-core/contracts/order-line.tei"},
		{"environment", "environments/development/environment.tei"},
		{"plugin", "plugins/tei-database-postgres/1.0.0/plugin.tei"},
		{"release", "releases/databases/sales-core/4.2.0.tei"},
		{"header-detail-binding", "applications/order-entry/realizations/header-detail/order-lines.tei"},
		{"executor-binding", "environments/development/contract-query-executors/order-line-find-by-order.tei"},
		{"sql-realization", "databases/sales-core/query-realizations/order-line-find-by-order.tei"},
		{"bundle", "bundles/tei-reference-providers/1.0.0/bundle.tei"},
		{"migration", "databases/sales-core/migrations/20260826_001/migration.tei"},
		{"persistence", "databases/sales-core/persistence/order-line.tei"},
	} {
		t.Run(tc.loader, func(t *testing.T) {
			dir := scratchWorkspace(t)
			path := filepath.Join(dir, filepath.FromSlash(tc.path))
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, path, "\ufeff"+string(content))
			run(tei, true, "validate", "-workspace", dir)
			if withBOM := lock(dir); !bytes.Equal(withBOM, baseline) {
				t.Fatalf("BOMの付与でLockが変わりました:\nbefore=%s\nafter=%s", baseline, withBOM)
			}
			var comparison struct {
				ChangedCollections []string          `json:"changedCollections"`
				StrictChanges      []json.RawMessage `json:"strictChanges"`
				SemanticChanges    []json.RawMessage `json:"semanticChanges"`
				ContractCandidates []json.RawMessage `json:"contractCandidates"`
				ComparisonError    string            `json:"comparisonError"`
			}
			if err := json.Unmarshal(run(compare, true, "compare-application-membership", "-before", original, "-after", dir), &comparison); err != nil {
				t.Fatal(err)
			}
			if comparison.ComparisonError != "" || len(comparison.ChangedCollections) != 0 || len(comparison.StrictChanges) != 0 || len(comparison.SemanticChanges) != 0 || len(comparison.ContractCandidates) != 0 {
				t.Fatalf("BOMの付与でWorldが変わりました: %+v", comparison)
			}

			// BOMを除いた不正byteは、そのloaderでも引き続き拒否する。
			writeFile(t, path, string(content)+"\x00")
			run(tei, false, "validate", "-workspace", dir)
		})
	}
}
