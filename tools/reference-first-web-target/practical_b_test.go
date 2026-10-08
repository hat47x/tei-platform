package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/workspace"
)

// 実証記録B（受理待ち）の機械的な証明。詳細はplan/issue/000075-practical-scenario-mvp-demonstration.md。
//
// reference/workspaceには書き込まない。候補 plan/issue/000075-practical-b/candidate.json は、
// workspaceのcopy（t.TempDir）に対してだけ、tools/reference-shared-acceptanceのCLIで受理する。
// そのpackageはmain packageなのでimportできず、CLIを実行して受理する。
//
// このtestが示すのは、scratch copyの中で候補がacceptedになり、order_entryの宣言と評価がその規則を含むことだけである。
// 実際のowner受理、Gate Evidence、browser上の表示は示さない。
const (
	practicalBCandidate = "plan/issue/000075-practical-b/candidate.json"
	practicalBRule      = "sales.order_line.line_number_positive"
)

func TestPracticalBLineNumberPositiveRuleAcceptedOnlyInScratchCopy(t *testing.T) {
	realWorkspace := referenceWorkspacePath(t)
	repo := filepath.Dir(filepath.Dir(realWorkspace))
	candidate := filepath.Join(repo, filepath.FromSlash(practicalBCandidate))
	tool := buildPracticalBAcceptanceTool(t, repo)

	realRevision := practicalBModelRevision(t, tool, realWorkspace)

	// copyは、末尾が reference/workspace にならない場所へ作る。acceptのbaseline保護を迂回しないため。
	scratch := filepath.Join(t.TempDir(), "scratch", "workspace")
	copyPracticalBWorkspace(t, realWorkspace, scratch)
	// 実workspaceは受理後の状態なので、scratch copyは受理前のbaselineへ戻してから候補を受理する。
	revertPracticalBAcceptance(t, tool, scratch)
	before := readPracticalBTree(t, scratch)

	var preview struct {
		Acceptable bool `json:"acceptable"`
	}
	runPracticalBTool(t, tool, &preview, "preview", "-workspace", scratch, "-candidate", candidate)
	if !preview.Acceptable {
		t.Fatal("scratch copyでのPreviewが受理不可です")
	}

	var review struct {
		PreviewAcceptable bool   `json:"previewAcceptable"`
		ReviewFingerprint string `json:"reviewFingerprint"`
	}
	runPracticalBTool(t, tool, &review, "review", "-workspace", scratch, "-candidate", candidate, "-format", "json")
	if !review.PreviewAcceptable || review.ReviewFingerprint == "" {
		t.Fatalf("Review Projectionが受理可能なfingerprintを返しません: %+v", review)
	}

	var decision struct {
		Decision      string `json:"decision"`
		SourceChanged bool   `json:"sourceChanged"`
		RecordPath    string `json:"recordPath"`
	}
	derivedOut := filepath.Join(t.TempDir(), "world.json")
	runPracticalBTool(t, tool, &decision,
		"accept", "-workspace", scratch, "-candidate", candidate,
		"-expected-review-fingerprint", review.ReviewFingerprint,
		"-derived-out", derivedOut,
	)
	if decision.Decision != "accepted" || !decision.SourceChanged {
		t.Fatalf("scratch copyでacceptedになりません: %+v", decision)
	}

	// scratch copyの変更は、宣言する2つのsource fileと、判断を残すDecision記録の1件だけに閉じる。
	after := readPracticalBTree(t, scratch)
	changed, added, removed := diffPracticalBTrees(before, after)
	wantChanged := []string{"applications/order-entry/application.tei", "semantic/sales/validation.tei"}
	if !reflect.DeepEqual(changed, wantChanged) {
		t.Fatalf("変更されたsource=%v want=%v", changed, wantChanged)
	}
	if len(removed) != 0 || len(added) != 1 || added[0] != decision.RecordPath || !strings.HasPrefix(added[0], ".tei/decisions/") {
		t.Fatalf("追加・削除されたfile: added=%v removed=%v recordPath=%s", added, removed, decision.RecordPath)
	}

	loaded, err := workspace.Load(scratch)
	if err != nil {
		t.Fatalf("受理後のscratch copyを読み直せません: %v", err)
	}
	world := loaded.World

	t.Run("a order_entry resolves three rules in declaration order", func(t *testing.T) {
		rules, err := driver.NewResolver(world).ResolveApplicationValidations(core.MustCanonicalID("order_entry"))
		if err != nil {
			t.Fatal(err)
		}
		got := make([]string, 0, len(rules))
		for _, rule := range rules {
			got = append(got, string(rule.ID))
		}
		want := []string{
			"sales.order_line.quantity_positive",
			"sales.order.lines.line_number_unique",
			practicalBRule,
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("解決された規則=%v want=%v", got, want)
		}
	})

	t.Run("b default browse row completes with three satisfied validations", func(t *testing.T) {
		runner := newPracticalBCommandRunner(t, world)
		// RowIndex 0と2026-09-18T08:30:00Zは、既存のCommand scenarioが既定の一覧として使う組み合わせである。
		result, err := runner.RunCommand(context.Background(), commandScenarioInput{
			LoginID:    "legacy-order-editor",
			Credential: "editor-secret",
			From:       "2026-09-18T08:30:00Z",
			RowIndex:   0,
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.Stage != "completed" {
			t.Fatalf("stage=%s result=%+v", result.Stage, result)
		}
		wantValidations := []string{
			"sales.order_line.quantity_positive",
			"sales.order.lines.line_number_unique",
			practicalBRule,
		}
		if len(result.Validations) != len(wantValidations) {
			t.Fatalf("validations=%+v", result.Validations)
		}
		for i, validation := range result.Validations {
			if validation.Validation != wantValidations[i] || validation.Result != "satisfied" {
				t.Fatalf("validation[%d]=%+v", i, validation)
			}
		}
	})

	t.Run("c first order line with line_number 0 fails only the new rule", func(t *testing.T) {
		runner := newPracticalBCommandRunner(t, world)
		readResult, err := newReferenceReadScenarioRunner(world).Run(
			context.Background(),
			readScenarioInput{From: "2026-09-18T08:30:00Z", RowIndex: 0},
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(readResult.OrderLineRows) == 0 {
			t.Fatal("test setup: order-line rowsがありません")
		}
		// 最初の受注明細の行番号だけを0にする。数量は既定の正の値のまま残すため、
		// 失敗するのは新しい規則（line_number_positive）だけになる。
		// line_number_uniqueは、0と2の二つの値を見て、重複がないため満たされる。
		if !setSemanticRowValue(&readResult.OrderLineRows[0], "sales.order_line.line_number", 0) {
			t.Fatal("test setup: line_number attribute not found")
		}
		runner.read = staticReadScenarioRunner{result: readResult}

		result, err := runner.RunCommand(context.Background(), commandScenarioInput{
			LoginID:    "legacy-order-editor",
			Credential: "editor-secret",
			From:       "2026-09-18T08:30:00Z",
			RowIndex:   0,
		})
		if err != nil {
			t.Fatalf("Validation not-satisfied became Execution Error: %v", err)
		}
		if result.Stage != "validation-not-satisfied" {
			t.Fatalf("stage=%s result=%+v", result.Stage, result)
		}
		if len(result.Validations) != 3 ||
			result.Validations[0] != (validationResultProjection{Validation: "sales.order_line.quantity_positive", Result: "satisfied"}) ||
			result.Validations[1] != (validationResultProjection{Validation: "sales.order.lines.line_number_unique", Result: "satisfied"}) ||
			result.Validations[2] != (validationResultProjection{Validation: practicalBRule, Result: "not-satisfied"}) {
			t.Fatalf("validations=%+v", result.Validations)
		}
		if result.Authorization != nil || result.Command != nil || result.Observation != nil {
			t.Fatalf("Validation not-satisfied continued downstream: %+v", result)
		}
	})

	t.Run("d real reference/workspace declares the three order_entry validations after acceptance", func(t *testing.T) {
		realLoaded, err := workspace.Load(realWorkspace)
		if err != nil {
			t.Fatal(err)
		}
		rules, err := driver.NewResolver(realLoaded.World).ResolveApplicationValidations(core.MustCanonicalID("order_entry"))
		if err != nil {
			t.Fatal(err)
		}
		got := make([]string, 0, len(rules))
		for _, rule := range rules {
			got = append(got, string(rule.ID))
		}
		want := []string{
			"sales.order_line.quantity_positive",
			"sales.order.lines.line_number_unique",
			practicalBRule,
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("実workspaceの解決結果=%v want=%v", got, want)
		}
		if revisionAfter := practicalBModelRevision(t, tool, realWorkspace); revisionAfter != realRevision {
			t.Fatalf("実workspaceのrevisionが変わりました: before=%s after=%s", realRevision, revisionAfter)
		}
	})
}

func newPracticalBCommandRunner(t *testing.T, world ir.World) *referenceCommandScenarioRunner {
	t.Helper()
	runner, err := newReferenceCommandScenarioRunner(
		world,
		legacyFixturePath(t, "users.json"),
		legacyFixturePath(t, "authorization.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return runner
}

// buildPracticalBAcceptanceToolは、受理CLIを一度だけbuildして返す。
func buildPracticalBAcceptanceTool(t *testing.T, repo string) string {
	t.Helper()
	tool := filepath.Join(t.TempDir(), "reference-shared-acceptance")
	if runtime.GOOS == "windows" {
		tool += ".exe"
	}
	var stderr bytes.Buffer
	cmd := exec.Command("go", "build", "-o", tool, "./tools/reference-shared-acceptance")
	cmd.Dir = repo
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("reference-shared-acceptanceをbuildできません: %v: %s", err, stderr.String())
	}
	return tool
}

func runPracticalBTool(t *testing.T, tool string, out any, args ...string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(tool, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s が失敗しました: %v\nstdout=%s\nstderr=%s", args[0], err, stdout.String(), stderr.String())
	}
	if out != nil {
		if err := json.Unmarshal(stdout.Bytes(), out); err != nil {
			t.Fatalf("%s の出力JSONを読めません: %v: %s", args[0], err, stdout.String())
		}
	}
}

func practicalBModelRevision(t *testing.T, tool, dir string) string {
	t.Helper()
	var out struct {
		ModelRevision string `json:"modelRevision"`
	}
	runPracticalBTool(t, tool, &out, "revision", "-workspace", dir)
	return out.ModelRevision
}

func copyPracticalBWorkspace(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("workspace copyを作れません: %v", err)
	}
}

// 受理後の正本から、scratch copyを受理前のbaselineへ戻す。
// 戻すのは、受理時のsourceEditが追加した行だけである。戻した各fileのsha256が受理前のfingerprintと一致し、
// revisionがbaselineへ戻ったことを確かめてから、候補を受理するscratch copyとして使う。
// 受理記録もscratch copyから外す。外さないと、同じ内容の記録が既にあるため「追加された記録」が空になる。
const (
	practicalBBaseModelRevision = "sha256:4234d86eba95e86a1bf0fd34df7c896f095cca1aa10dc5650d9ddd4dc2b08db4"
	practicalBApplicationSource = "applications/order-entry/application.tei"
	practicalBValidationSource  = "semantic/sales/validation.tei"
	practicalBApplicationBefore = "sha256:d9152277abbeb0fdc1ea23e5d54084b8c2ef1ebb5e45e3d32b2340fe04eff71a"
	practicalBValidationBefore  = "sha256:bd641910f993ef84cbab1ccf3616de8d98c715e7679268115a71a0327e126633"
	practicalBApplicationInsert = "    uses validation sales.order_line.line_number_positive\n"
	practicalBValidationInsert  = "\nvalidation sales.order_line.line_number_positive positive {\n    attribute sales.order_line.line_number\n}\n"
	practicalBAcceptedRecord    = ".tei/decisions/a043abc84702feb4a4558e2fbfd6fa17bd60867075f65d126d648998a11f2523.json"
)

func revertPracticalBAcceptance(t *testing.T, tool, scratch string) {
	t.Helper()
	revertPracticalBSource(t, scratch, practicalBApplicationSource, practicalBApplicationInsert, practicalBApplicationBefore)
	revertPracticalBSource(t, scratch, practicalBValidationSource, practicalBValidationInsert, practicalBValidationBefore)
	if err := os.Remove(filepath.Join(scratch, filepath.FromSlash(practicalBAcceptedRecord))); err != nil {
		t.Fatalf("受理記録をscratch copyから外せません: %v", err)
	}
	if revision := practicalBModelRevision(t, tool, scratch); revision != practicalBBaseModelRevision {
		t.Fatalf("scratch copyがbaselineへ戻りません: revision=%s want=%s", revision, practicalBBaseModelRevision)
	}
}

func revertPracticalBSource(t *testing.T, scratch, rel, insert, before string) {
	t.Helper()
	path := filepath.Join(scratch, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if strings.Count(content, insert) != 1 {
		t.Fatalf("%s に受理時の追加行が一つだけ見つかりません", rel)
	}
	restored := strings.Replace(content, insert, "", 1)
	if got := practicalBSHA256(restored); got != before {
		t.Fatalf("%s を戻した結果が受理前のfingerprintと一致しません: got=%s want=%s", rel, got, before)
	}
	if err := os.WriteFile(path, []byte(restored), 0o644); err != nil {
		t.Fatal(err)
	}
}

func practicalBSHA256(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// readPracticalBTreeは、workspace内の全fileの内容を、slash区切りの相対pathで返す。
func readPracticalBTree(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("workspaceを読めません: %v", err)
	}
	return files
}

func diffPracticalBTrees(before, after map[string]string) (changed, added, removed []string) {
	for path, content := range before {
		next, ok := after[path]
		switch {
		case !ok:
			removed = append(removed, path)
		case next != content:
			changed = append(changed, path)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			added = append(added, path)
		}
	}
	sort.Strings(changed)
	sort.Strings(added)
	sort.Strings(removed)
	return changed, added, removed
}
