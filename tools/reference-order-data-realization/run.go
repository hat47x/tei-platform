package orderdatarealization

import (
	"context"
	"fmt"
	"strings"
)

// Run executes the whole chain against the disposable database and always
// tries to clean up what it created. The returned record is complete as far as
// the chain got.
func (c *Chain) Run(ctx context.Context, dsn string) (err error) {
	defer func() {
		if c.db != nil {
			cleanupErr := cleanupOwned(context.WithoutCancel(ctx), c.db.db, &c.db.safety)
			c.db.db.Close()
			if err == nil && cleanupErr != nil {
				err = cleanupErr
			}
		}
		c.evaluate(err)
	}()
	if err := c.RunDesign(ctx); err != nil {
		return err
	}
	steps := []func(context.Context) error{
		func(ctx context.Context) error { return c.Step5BuildPlan(ctx, dsn) },
		c.Step6RefuseBeforeMigration,
		c.Step7ApplyAndConverge,
		c.Step8ListAndDetail,
		c.Step9Register,
		c.Step10SQLite,
	}
	for _, s := range steps {
		if err := s(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *Chain) findFault(id string) *Check {
	for i := range c.Rec.Faults {
		if c.Rec.Faults[i].ID == id {
			return &c.Rec.Faults[i]
		}
	}
	return nil
}

func (c *Chain) findCompletion(id string) *Check {
	for i := range c.Rec.Completion {
		if c.Rec.Completion[i].ID == id {
			return &c.Rec.Completion[i]
		}
	}
	return nil
}

func (c *Chain) step(n int) *StepRecord {
	for i := range c.Rec.Steps {
		if c.Rec.Steps[i].Step == n {
			return &c.Rec.Steps[i]
		}
	}
	return nil
}

func passed(ch *Check) bool { return ch != nil && ch.Passed }

// evaluate writes the completion conditions of the README and the overall
// verdict. A condition whose evidence is missing fails.
func (c *Chain) evaluate(runErr error) {
	r := c.Rec.Revisions
	bound := len(c.Rec.Steps) == 10
	var revs []string
	for _, s := range c.Rec.Steps {
		revs = append(revs, fmt.Sprintf("%d:%s", s.Step, short(s.ModelRevision)))
		if s.Step >= 4 && s.ModelRevision != r.R3 {
			bound = false
		}
	}
	chain := []string{short(r.R0), short(r.R1), short(r.R2), short(r.R3)}
	planBound := c.db != nil && c.db.plan != nil && c.db.plan.ModelRevision == r.R3
	add := func(ch Check) {
		if existing := c.findCompletion(ch.ID); existing != nil {
			*existing = ch
			return
		}
		c.Rec.Completion = append(c.Rec.Completion, ch)
	}
	tamper := c.findCompletion("版の変更で停止")
	add(Check{ID: "C1-同じ版での引継ぎ", Step: "1-10",
		Expect:   "属性追加と二つの判断が最終版R3に結び付いたまま、期待状態、移行の差分、一覧、登録に使われる。別の版や複製を読めば停止する",
		Observed: map[string]any{"revisionChain": chain, "stepRevisions": revs, "planBoundToR3": planBound, "tamperedCopyStopped": passed(tamper)},
		Passed:   bound && planBound && passed(tamper),
	})
	var variantAccepted any
	var diffsDiffer any
	if s := c.step(5); s != nil {
		out := s.Output.(map[string]any)
		diffsDiffer = out["diffsDiffer"]
		if p, ok := out["variantDuplicateProbe"].(map[string]any); ok {
			variantAccepted = p["sameExternalNumberAccepted"]
		}
	}
	dup := c.findCompletion("確定した失敗と結果不明の区別")
	add(Check{ID: "C2-判断の有無で差分と結果が変わる", Step: "5・9",
		Expect:   "判断を行った版と行わなかった版で、工程5の差分(一意制約の追加の有無)と工程9の結果(重複の拒否の有無)が異なる",
		Observed: map[string]any{"diffsDiffer": diffsDiffer, "variantSameExternalNumberAccepted": variantAccepted, "finalDuplicate": observed(dup, "duplicate")},
		Passed:   diffsDiffer == true && variantAccepted == 2 && passed(dup),
	})
	mig6 := c.findFault("P-MIG-03-移行前の登録")
	var step7 map[string]any
	if s := c.step(7); s != nil {
		step7 = s.Output.(map[string]any)
	}
	regAPassed := false
	if s := c.step(9); s != nil {
		if a, ok := s.Output.(map[string]any)["A"].(map[string]any); ok {
			regAPassed = a["observation"] == ObsObserved
		}
	}
	add(Check{ID: "C3-移行前の拒否と収束後の登録", Step: "6・7・9",
		Expect:   "変更前の構造へ登録すると実行前に拒否され、移行と再観測での収束の後に同じ登録が通る。既存の行は失われず、新しい属性は既存の行でNULL",
		Observed: map[string]any{"refused": passed(mig6), "seededRowsUnchanged": step7["seededRowsUnchanged"], "seededRowsNewAttributeNull": step7["seededRowsNewAttributeNull"], "sameRegistrationAfterConvergence": regAPassed},
		Passed:   passed(mig6) && step7["seededRowsUnchanged"] == true && regAPassed && c.db != nil && step7["seededRowsNewAttributeNull"] == len(c.db.seeded),
	})
	if ch := c.findCompletion("一覧の識別子を明細へ渡す"); ch == nil {
		add(Check{ID: "一覧の識別子を明細へ渡す", Step: "8", Expect: "一覧で選んだ受注の識別子をそのまま明細へ渡す", Observed: "not run"})
	}
	tx := c.findFault("P-TX-02-登録後の再読込の喪失")
	add(Check{ID: "C5-登録の結果と別接続の観測", Step: "9",
		Expect:   "登録の結果と別接続での再読込を別々の根拠として記録し、再読込に失敗した登録は観測なしのまま残す。確定した失敗と結果不明を区別する",
		Observed: map[string]any{"P-TX-02": passed(tx), "confirmedVsUnknown": passed(dup)},
		Passed:   passed(tx) && passed(dup),
	})
	var safety any
	safe := false
	if c.db != nil {
		s := c.db.safety
		safety = s
		safe = s.UserRelationsBefore == 0 && s.ExtraSchemasBefore == 0 && len(s.LeftUnowned) == 0 && s.UserRelationsAfter == 0 && s.ExtraSchemasAfter == 0 && len(s.Dropped) > 0
	}
	add(Check{ID: "C6-使い捨てDBの隔離と破棄", Step: "5-10",
		Expect:   "工程をまたぐ使い捨てのDBを一つだけ使い、開始前に空であることを確かめ、終了時に所有印のあるものだけを削除する",
		Observed: safety, Passed: safe,
	})
	add(Check{ID: "C7-拡張(旧版と共存する移行、SQLiteでの往復)", Step: "7・10",
		Expect:   "旧版が書いた行でも新しい属性が宣言どおりで、旧版が残る間は縮約が拒否される。SQLiteで読み戻した値がPostgreSQLの値と一致するか、宣言した既知の差分だけ",
		Observed: map[string]any{"P-MIG-03-old-writer": passed(c.findFault("P-MIG-03-旧版の書込みと共存する移行")), "P-DB-03": passed(c.findFault("P-DB-03-SQLiteでの往復"))},
		Passed:   passed(c.findFault("P-MIG-03-旧版の書込みと共存する移行")) && passed(c.findFault("P-DB-03-SQLiteでの往復")),
	})
	ok := runErr == nil && len(c.Rec.Steps) == 10
	for _, f := range c.Rec.Faults {
		ok = ok && f.Passed
	}
	for _, ch := range c.Rec.Completion {
		ok = ok && ch.Passed
	}
	wantFaults := []string{"P-MIG-03-移行前の登録", "固有-計画後の構造変更", "P-MIG-03-旧版の書込みと共存する移行", "P-TX-02-登録後の再読込の喪失", "固有-同じ外部番号の同時登録", "P-INT-03-欠番と採番元の停止", "P-DB-03-SQLiteでの往復"}
	for _, id := range wantFaults {
		ok = ok && passed(c.findFault(id))
	}
	if runErr != nil {
		c.Rec.Environment["error"] = runErr.Error()
	}
	c.Rec.Limits = []string{
		"登録のアダプターは手書き(custom go)で、受注の契約(書込みは状態の列だけ)の範囲を検査していない",
		"認可(authority)はない。回答者の身元も認証していない(claimed human)",
		"属性の追加と一意制約の宣言は複製への直接編集で、共通受理を通していない。候補キーだけが共通受理を通った",
		"コードの生成はない。登録の確定と再読込は同じプロセスの別の接続(プール)で、別のサーバではない",
		"DBは検証器が起動した使い捨てのPostgreSQL一つで、実環境ではない。外部番号の採番元は同じプロセスのHTTPの模擬",
		"スキーマの比較は保存写像が使うスキーマ(public)に限る。サンプルの管理表(odr_meta)は比較の外",
		"SQLiteはPython標準のsqlite3を別プロセスで使った。TEIのSQLiteの提供元ではない",
	}
	c.Rec.Passed = ok
}

func observed(ch *Check, key string) any {
	if ch == nil {
		return nil
	}
	if m, ok := ch.Observed.(map[string]any); ok {
		return m[key]
	}
	return nil
}

func short(rev string) string {
	rev = strings.TrimPrefix(rev, "sha256:")
	if len(rev) > 12 {
		return rev[:12]
	}
	return rev
}
