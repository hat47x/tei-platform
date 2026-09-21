package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// S7は、S6で受理したApplicationのdatasource aliasをEnvironmentのdatabase instanceへ結ぶ。
// Application意味の変更とは別authorityだが、同じ受入境界を通す（`ISSUE-000040` S7）。
func environmentBindingCandidate(revision string) Candidate {
	return Candidate{
		Schema:            candidateSchema,
		ID:                "cand-repair-intake-environment-binding",
		BaseModelRevision: revision,
		Origin:            Origin{Kind: "cli", Actor: "test"},
		Intent:            "repair_intakeのcustomer_master aliasを、各Environmentの既存master_data instanceへ結ぶ",
		Evidence: []Evidence{
			{Source: "applications/repair-intake/application.tei", Note: "aliasを宣言しているApplication source"},
		},
		Delta: []Operation{
			{
				Op: opAddEnvironmentBinding, Placement: "environments/development/environment.tei",
				EnvironmentBinding: &EnvironmentBindingSpec{
					Environment: "development", Application: "repair_intake",
					Alias: "customer_master", Instance: "master_data.dev",
				},
			},
			{
				Op: opAddEnvironmentBinding, Placement: "environments/production/environment.tei",
				EnvironmentBinding: &EnvironmentBindingSpec{
					Environment: "production", Application: "repair_intake",
					Alias: "customer_master", Instance: "master_data.prod",
				},
			},
		},
	}
}

func previewEnvironmentBinding(t *testing.T, mutate func(c *Candidate)) (string, evaluation) {
	t.Helper()
	dir := scratchWorkspaceBeforeEnvironmentBinding(t)
	c := environmentBindingCandidate(revisionOfDir(t, dir))
	if mutate != nil {
		mutate(&c)
	}
	ev, err := evaluate(dir, mustJSON(t, c))
	if err != nil {
		t.Fatal(err)
	}
	return dir, ev
}

func TestEnvironmentBindingPreviewReportsTheDecisionSeparatelyFromMeaning(t *testing.T) {
	_, ev := previewEnvironmentBinding(t, nil)
	if !ev.preview.Acceptable {
		t.Fatalf("Previewが受理可能になりません: %v", allDiagnostics(ev.preview))
	}

	// Environment bindingはApplicationの意味を変えない。semantic / application差分は空のままである。
	if len(ev.preview.SemanticDiff) != 0 || len(ev.preview.ApplicationDiff) != 0 {
		t.Fatalf("意味の差分が出ています: semantic=%v application=%v", ev.preview.SemanticDiff, ev.preview.ApplicationDiff)
	}
	if got := keysOf(ev.preview.EnvironmentDiff); len(got) != 2 ||
		got[0] != "added environment-binding:development/repair_intake.customer_master" ||
		got[1] != "added environment-binding:production/repair_intake.customer_master" {
		t.Fatalf("environment差分=%v", got)
	}
	if got := ev.preview.ChangedCollections; len(got) != 1 || got[0] != "environments" {
		t.Fatalf("changed collections=%v want [environments]", got)
	}

	// impactはexact identityから辿る。名前の一致で結ばない。
	edges := map[string]bool{}
	for _, e := range ev.preview.Impact {
		if e.Reason == "" {
			t.Fatalf("辺に導出理由がありません: %+v", e)
		}
		edges[e.From+" -> "+e.To+" via "+e.Via] = true
	}
	for _, want := range []string{
		"environment-binding:development/repair_intake.customer_master -> application:repair_intake via environment.binding.application",
		"environment-binding:development/repair_intake.customer_master -> database-instance:master_data.dev via environment.binding.instance",
		"environment-binding:development/repair_intake.customer_master -> environment:development via environment.binding.owner",
		"environment-binding:production/repair_intake.customer_master -> database-instance:master_data.prod via environment.binding.instance",
		"database-instance:master_data.dev -> database:master_data via environment.instance.definition",
	} {
		if !edges[want] {
			t.Fatalf("辺 %q がありません: %v", want, ev.preview.Impact)
		}
	}

	// bindingができても、接続・実行の権限はこの変更に含まれない。unresolvedとして残す。
	var codes []string
	for _, u := range ev.preview.Unresolved {
		codes = append(codes, u.Code)
	}
	if !hasCode(codes, "RUNTIME_AUTHORITY_NOT_GRANTED") {
		t.Fatalf("実行権限が未付与であることを残していません: %v", codes)
	}
}

func TestEnvironmentBindingAcceptWritesOnlyTheTwoBindingLines(t *testing.T) {
	dir := scratchWorkspaceBeforeEnvironmentBinding(t)
	before := treeDigest(t, dir)
	data := mustJSON(t, environmentBindingCandidate(revisionOfDir(t, dir)))

	decision, err := accept(dir, data, acceptOptions{})
	if err != nil || decision.Decision != "accepted" || !decision.SourceChanged {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}

	after := treeDigest(t, dir)
	if len(after) != len(before) {
		t.Fatalf("file数が変わりました: before=%d after=%d", len(before), len(after))
	}
	changed := []string{}
	for path, digest := range after {
		if before[path] != digest {
			changed = append(changed, path)
		}
	}
	want := map[string]string{
		"environments/development/environment.tei": "    bind repair_intake.customer_master -> master_data.dev",
		"environments/production/environment.tei":  "    bind repair_intake.customer_master -> master_data.prod",
	}
	if len(changed) != len(want) {
		t.Fatalf("変更されたfile=%v", changed)
	}
	for _, path := range changed {
		line, ok := want[path]
		if !ok {
			t.Fatalf("宣言していないfileが変わりました: %s", path)
		}
		content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(string(content), line+"\n") != 1 {
			t.Fatalf("%sへbinding一行が入っていません", path)
		}
		// 挿入はEnvironment blockの内側で終わる。resource / grant / lease等の既存宣言は動かさない。
		for _, keep := range []string{"database-instance master_data.", "grant deployment-executor", "lease-policy deployment-executor"} {
			if !strings.Contains(string(content), keep) {
				t.Fatalf("%sの既存宣言 %q が失われています", path, keep)
			}
		}
	}
	for _, c := range decision.Conformance {
		if !c.Passed {
			t.Fatalf("受理後のConformance %s が通っていません", c.Name)
		}
	}
	if len(decision.EnvironmentDiff) != 2 {
		t.Fatalf("受理結果のenvironment差分=%v", keysOf(decision.EnvironmentDiff))
	}
}

// 層をまたぐ正しさは共有Validationが見る。境界側で再実装していないことを、
// 共有validatorのdiagnosticがそのままvalidate stageへ現れることで確認する。
func TestEnvironmentBindingLeavesCrossLayerCorrectnessToTheSharedValidator(t *testing.T) {
	cases := map[string]struct {
		mutate func(c *Candidate)
		code   string
	}{
		"unknown alias": {
			mutate: func(c *Candidate) { c.Delta[0].EnvironmentBinding.Alias = "unknown_alias" },
			code:   "UNKNOWN_BINDING_DATASOURCE_ALIAS",
		},
		"unknown instance": {
			mutate: func(c *Candidate) { c.Delta[0].EnvironmentBinding.Instance = "master_data.missing" },
			code:   "UNKNOWN_DATABASE_INSTANCE",
		},
		"target mismatch": {
			mutate: func(c *Candidate) { c.Delta[0].EnvironmentBinding.Instance = "sales_core.dev" },
			code:   "DATABASE_BINDING_TARGET_MISMATCH",
		},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			dir, ev := previewEnvironmentBinding(t, testCase.mutate)
			if ev.preview.Acceptable {
				t.Fatal("受理可能になっています")
			}
			if stage := stageStatus(t, ev.preview, "validate"); stage.Status != "fail" {
				t.Fatalf("validate stage=%s。共有Validationへ到達していません", stage.Status)
			}
			if !hasCode(diagnosticCodes(ev.preview), testCase.code) {
				t.Fatalf("%s が出ていません: %v", testCase.code, diagnosticCodes(ev.preview))
			}
			assertTreeUnchanged(t, treeDigest(t, dir), treeDigest(t, dir))
		})
	}
}

// 受入境界側が持つのは、存在・一意性・配置の確認だけである。
func TestEnvironmentBindingRefusesUnknownEnvironmentDuplicateAndWrongPlacement(t *testing.T) {
	cases := map[string]struct {
		mutate func(c *Candidate)
		code   string
	}{
		"unknown environment": {
			mutate: func(c *Candidate) { c.Delta[0].EnvironmentBinding.Environment = "staging" },
			code:   "ENVIRONMENT_NOT_FOUND",
		},
		"binding already exists": {
			mutate: func(c *Candidate) {
				c.Delta[0].EnvironmentBinding.Application = "order_entry"
				c.Delta[0].EnvironmentBinding.Alias = "master"
			},
			code: "ENVIRONMENT_BINDING_ALREADY_EXISTS",
		},
		"placement is another environment": {
			mutate: func(c *Candidate) { c.Delta[0].Placement = "environments/production/environment.tei" },
			code:   "PLACEMENT_NOT_OWNED",
		},
		"placement is not managed source": {
			mutate: func(c *Candidate) { c.Delta[0].Placement = "environments/development/missing.tei" },
			code:   "PLACEMENT_NOT_MANAGED_SOURCE",
		},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			dir, ev := previewEnvironmentBinding(t, testCase.mutate)
			if ev.preview.Acceptable {
				t.Fatal("受理可能になっています")
			}
			if !hasCode(diagnosticCodes(ev.preview), testCase.code) {
				t.Fatalf("%s が出ていません: %v", testCase.code, diagnosticCodes(ev.preview))
			}
			if got := treeDigest(t, dir); len(got) == 0 {
				t.Fatal("workspaceが読めません")
			}
		})
	}
}

func TestEnvironmentBindingOperationIsRefusedWhenMalformed(t *testing.T) {
	cases := map[string]func(c *Candidate){
		"no placement":       func(c *Candidate) { c.Delta[0].Placement = "" },
		"no binding":         func(c *Candidate) { c.Delta[0].EnvironmentBinding = nil },
		"empty alias":        func(c *Candidate) { c.Delta[0].EnvironmentBinding.Alias = "" },
		"alias is qualified": func(c *Candidate) { c.Delta[0].EnvironmentBinding.Alias = "repair_intake.customer_master" },
		"semantic field":     func(c *Candidate) { c.Delta[0].Namespace = "sales" },
		"duplicate in delta": func(c *Candidate) { c.Delta = append(c.Delta, c.Delta[0]) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			_, ev := previewEnvironmentBinding(t, mutate)
			if ev.preview.Acceptable {
				t.Fatal("壊れたoperationが受理可能になっています")
			}
			if stage := stageStatus(t, ev.preview, "normalize"); stage.Status == "pass" {
				if resolve := stageStatus(t, ev.preview, "resolve"); resolve.Status != "fail" {
					t.Fatalf("normalizeもresolveも拒否していません: %v", diagnosticCodes(ev.preview))
				}
			}
		})
	}
}
