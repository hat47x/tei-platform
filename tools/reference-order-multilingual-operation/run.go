package ordermultilingualoperation

import (
	"context"
	"fmt"
	"path/filepath"
)

// Environment variables of the through test.
const (
	DSNEnv       = "TEI_OMO_DSN"
	ConfirmEnv   = "TEI_OMO_CONFIRM_ISOLATED"
	ConfirmValue = "tei_omo"
	OutDirEnv    = "TEI_OMO_OUT_DIR"
	PythonEnv    = "TEI_OMO_PYTHON"
	NodeEnv      = "TEI_OMO_NODE"
)

// Config is what the through run needs besides the repository.
type Config struct {
	DSN    string
	Python string
	Node   string
}

// Run executes stages 1-6, the fault injections and the completion checks,
// then verifies the chain by reading the record only.
func (c *Chain) Run(ctx context.Context, cfg Config) error {
	steps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"stage 1", c.Stage1}, {"stage 2", c.Stage2}, {"stage 3", c.Stage3},
		{"identity", c.IdentityChecks}, {"bundle tampering", c.FaultBundleTampering}, {"P-DB-02", c.FaultPDB02},
		{"stage 4", c.Stage4},
		{"stage 5", func(ctx context.Context) error {
			return c.Stage5(ctx, PGStore{DSN: cfg.DSN}, SQLiteStore{Path: filepath.Join(c.Work, "omo.sqlite3"), Python: cfg.Python})
		}},
		{"stage 6", func(ctx context.Context) error { return c.Stage6(ctx, cfg.Node) }},
	}
	for _, s := range steps {
		if err := s.fn(ctx); err != nil {
			c.Finish()
			return fmt.Errorf("%s: %w", s.name, err)
		}
	}
	c.Finish()
	return nil
}

func (c *Chain) faultPassed(prefix string) (bool, []string) {
	ok, n := true, 0
	var ids []string
	for _, f := range c.Rec.Faults {
		if len(f.ID) >= len(prefix) && f.ID[:len(prefix)] == prefix {
			n++
			ids = append(ids, f.ID)
			ok = ok && f.Passed
		}
	}
	return ok && n > 0, ids
}

// Finish computes the completion conditions over the same target list,
// records gaps and limits, and verifies the chain.
func (c *Chain) Finish() {
	if len(c.Matrix) > 0 {
		agree := true
		categories := map[string]bool{}
		for _, r := range c.Matrix {
			agree = agree && r.Agrees
			if r.Agrees {
				categories[r.Kind+"/"+r.Category+"/"+r.After.Status] = true
			}
		}
		need := []string{"message/dependent/expired", "display-name/dependent/expired", "function-name/dependent/expired",
			"message/dependent-other-language/expired", "message/independent/valid", "message/other-provider/valid",
			"message/other-language/valid", "display-name/other-language/valid", "display-name/same-provider-unrevised/valid"}
		var missing []string
		for _, k := range need {
			if !categories[k] {
				missing = append(missing, k)
			}
		}
		var s2, s4 StageRecord
		for _, s := range c.Rec.Stages {
			switch s.Stage {
			case 2:
				s2 = s
			case 4:
				s4 = s
			}
		}
		history, _ := s4.Outputs["historyKeptR1English"].(float64)
		c.Rec.Completion = append(c.Rec.Completion, Check{ID: "one-revision-expires-dependent-translations-only", Stage: "4",
			Passed: agree && len(missing) == 0 && s2.LockFP == s4.LockFP && history == 3,
			Observed: map[string]any{"rows": len(c.Matrix), "allRowsAgree": agree, "missingCategories": missing,
				"sameLock": s2.LockFP == s4.LockFP, "historyKeptR1English": history}})
		old, _ := s4.Outputs["oldPinnedAdmission"].(map[string]any)
		neu, _ := s4.Outputs["newPinnedAdmission"].(map[string]any)
		reconf, _ := s4.Outputs["afterReconfirm"].(map[string]any)
		aligned := true
		for _, k := range []string{"message order.batch_confirmed en-US std.lang_en", "display-name data:sales.order en-US std.lang_en",
			"function-name capability:sales.order.submit en-US std.lang_en"} {
			cell, _ := reconf[k].(map[string]any)
			aligned = aligned && cell["status"] == "valid"
		}
		c.Rec.Completion = append(c.Rec.Completion, Check{ID: "regenerated-bundle-differs-and-old-pin-refused", Stage: "4",
			Passed: c.B2.Digest != "" && c.B2.Digest != c.B1.Digest && old["admitted"] == false && neu["admitted"] == true && aligned,
			Observed: map[string]any{"before": c.B1.Digest, "after": c.B2.Digest, "oldPinnedReason": old["reason"],
				"newPinnedAdmitted": neu["admitted"], "messageDisplayFunctionAlignedEnUS": aligned}})
	}
	pdb, ids := c.faultPassed("pdb02-")
	mix, _ := c.faultPassed("pmix03-")
	c.Rec.Completion = append(c.Rec.Completion, Check{ID: "japanese-continues-without-silent-old-translation", Stage: "2-4",
		Passed: pdb && mix, Observed: map[string]any{"cases": ids, "pdb02": pdb, "pmix03": mix}})
	var s5 StageRecord
	for _, s := range c.Rec.Stages {
		if s.Stage == 5 {
			s5 = s
		}
	}
	if s5.Stage == 5 {
		pre, _ := s5.Outputs["precondition"].(map[string]any)
		c.Rec.Conditional = append(c.Rec.Conditional, Check{ID: "stage5-precondition", Stage: "5",
			Passed: true, Observed: map[string]any{"established": pre["established"],
				"countedAsCompletion": false, "reason": "no product description attribute in the workspace; the item code mapping is the sample's"}})
	}
	c.Rec.Gaps = Gaps()
	c.Rec.Limits = Limits()
	c.Rec.Through = VerifyThrough(c.Rec)
	passed := c.Rec.Through.Complete
	for _, list := range [][]Check{c.Rec.Faults, c.Rec.Completion, c.Rec.Conditional} {
		for _, ch := range list {
			passed = passed && ch.Passed
		}
	}
	c.Rec.Passed = passed && len(c.Rec.Faults) > 0 && len(c.Rec.Completion) >= 4
}

// Gaps are the platform gaps the run observed.
func Gaps() []Gap {
	return []Gap{
		{"固有", "データ", "reference/workspaceに、受注の業務定義を所有するプラグイン、std.message、言語プラグイン、日本語の表示名がない。構文の識別子はASCIIで、Symbolは識別子。dataとattributeのdisplay文字列は構文にあるが使われておらず、capabilityにはない",
			"複製に所有プラグインsales.order_entry、std.message、言語プラグインを加え、日本語の表示名とメッセージは別の原文宣言(testdata/source-r*.json)に置き、派生索引の型つきの識別子へ結ぶ"},
		{"固有", "コア", "原文版の導出規則が資産ごとに異なる(メッセージは定義全体のSHA-256、表示名は表示文字列のSHA-256、業務データは文章版)",
			"資産ごとの版をそのまま使い、通しの記録に原文宣言のダイジェスト(改訂一回)と各対象の資産固有の版の前後を並べる"},
		{"固有", "プラグイン", "翻訳の許可と受理の確認口が、メッセージ(ReviewGate)と表示名(Grantのみ。配布候補の作成に確認口がない)で別",
			"サンプルのHostが同じ確認(sourceacceptance.VerifyCurrent、原文宣言、固定情報)を両方の前に行う。製品の確認口ではない"},
		{"固有", "コア", "reference/workspaceの複製では、databases/sales-core/README.mdが解析済みの読込み写しにないため、workspace.IndexedLoadedSnapshotが失敗し、overlaysのBuildAcceptedWorkspaceDisplayとBuildWorkspaceDisplayが使えない",
			"Hostの照合にはworkspace.BuildSourceIndexを使う(読込み後にファイルを読む)。BuildAcceptedWorkspaceDisplayは一回呼んで失敗を記録する"},
		{"P-MIX-03", "実行時", "配布候補の整合性とLockの照合だけでは、原文改訂と許可の取消しを検出できない。ダイジェストは訳文と同時に再計算できる",
			"サンプル側で、配布候補の各表示の原文版を現在の原文と照合し、現在の条件で再生成した候補のダイジェストと比べる。署名と認証された配布経路はない"},
		{"固有", "実行時", "メッセージの引数展開と件数書式の接続がなく、named-v0に複数形の選択がない",
			"言語プラグインの訳に複数形の別形をサンプルの形で持たせ、ValidateTextで検査し、Probeの分類で選んで{count}を置換する"},
		{"固有", "データ", "業務データの多言語属性が意味モデルにない。受注明細のitemCodeから商品説明への関係がない",
			"product:<itemCode>への対応と属性catalog.product.descriptionはサンプルの値。工程5は完了条件に数えない"},
		{"P-TX-02", "ツール", "翻訳依頼と応答の追跡、受理トランザクション内での再照合がない。結果不明は資産ではtranslation-failedと区別されない",
			"模擬の翻訳器の誤りの型でサンプルが結果不明を分類し、自動で再試行しない。再照合はPrepareTranslationの読み直しだけ"},
		{"P-DB-04", "データ", "並びと比較の意味を宣言する手段がない",
			"サンプルが並びの鍵(body、record_id、locale、UTF-8の符号位置順)を宣言し、PostgreSQLはCOLLATE \"C\"、SQLiteはBINARYへ写す"},
	}
}

// Limits states what the run is not evidence for.
func Limits() []string {
	return []string{
		"acceptance, grants and review are a synthetic host in the test process, not product acceptance or authorization",
		"the translator is a mock in the test process; nothing is sent to any external service",
		"one disposable PostgreSQL instance and one SQLite file on the same machine; connections are separate but in one process (SQLite: a Python process per call)",
		"PostgreSQL and SQLite agree on the declared binary order of these rows; no linguistic equivalence (normalization of voiced marks or half-width kana) is claimed",
		"stage 5 and 6 are conditional and not counted as completion",
		"the bundle digest detects accidental changes; it does not authenticate a publisher",
	}
}
