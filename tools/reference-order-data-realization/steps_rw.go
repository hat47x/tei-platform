package orderdatarealization

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"tei/internal/core"
	"tei/internal/driver"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	sqlquery "tei/internal/driver/queryruntime/sql"
	"tei/internal/environment"
	"tei/tools/internal/postgresdb"
)

// newPool opens a separate pool with one connection.
func newPool(dsn string) (*sql.DB, error) {
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func (c *Chain) newAdapter(db *sql.DB) *orderAdapter {
	return &orderAdapter{db: db, w: c.db.wFinal, attr: c.Decl.Attribute.Attribute}
}

// template is the submission tried in step 6 and registered unchanged in
// step 9 once the database has converged.
func (c *Chain) template() Submission {
	return Submission{
		ModelRevision: c.final, CustomerID: 7, OrderedAt: baseTime.Add(26 * time.Hour), Status: "received", Total: "4800.00",
		DeliveryNote: strp("午前中に配達(置き配可)"),
		Lines: []OrderLine{
			{LineNumber: 1, ItemCode: "SKU-101", Quantity: "2", UnitPrice: "1200.00"},
			{LineNumber: 2, ItemCode: "SKU-102", Quantity: "3", UnitPrice: "800.00"},
		},
	}
}

func withIDs(s Submission, orderID int64, ext *string) Submission {
	s.OrderID, s.ExternalID = orderID, ext
	s.Lines = append([]OrderLine(nil), s.Lines...)
	for i := range s.Lines {
		s.Lines[i].ID = orderID*10 + int64(i+1)
	}
	return s
}

// Step6RefuseBeforeMigration tries the registration against the old structure.
func (c *Chain) Step6RefuseBeforeMigration(ctx context.Context) error {
	if err := c.require(6, c.WorkDir); err != nil {
		return err
	}
	st := c.db
	pool, err := newPool(st.dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	a := c.newAdapter(pool)
	before, _, err := rowsDigest(ctx, st.db, orderR0Columns, st.seeded)
	if err != nil {
		return err
	}
	countBefore, _ := countRows(ctx, st.db, `SELECT count(*) FROM public.orders`)
	reg := c.submit(ctx, "step6-before-migration", a, withIDs(c.template(), 0, strp("pending")))
	after, _, err := rowsDigest(ctx, st.db, orderR0Columns, st.seeded)
	if err != nil {
		return err
	}
	countAfter, _ := countRows(ctx, st.db, `SELECT count(*) FROM public.orders`)
	var physical string
	for _, s := range reg.Preflight {
		if s.Stage == "physical-schema" {
			physical = s.Status + ": " + s.Detail
		}
	}
	passed := reg.Outcome == OutcomeRefused && reg.AdapterCall == 0 && before == after && countBefore == countAfter
	c.fault(Check{ID: "P-MIG-03-移行前の登録", Step: "6",
		Injected: "属性を追加した定義が検証を通った直後に、列も一意制約もない旧構造のDBへ、変更後の定義のまま登録する",
		Expect:   "実行前確認で差分が未解決のため、登録用アダプターを呼ばずに拒否し、既存の行が変わらない",
		Observed: map[string]any{"outcome": reg.Outcome, "adapterCalls": reg.AdapterCall, "physicalSchemaStage": physical, "rowsUnchanged": before == after, "rowCount": countAfter},
		Passed:   passed,
	})
	if !passed {
		return fmt.Errorf("step 6: registration was not refused: %+v", reg)
	}
	c.addStep(StepRecord{
		Step: 6, Name: "移行前の登録の拒否", ModelRevision: c.final,
		Assets: []AssetUse{{Asset: "tools/reference-data-first-persistence", Called: false, Reason: "package mainで、受付依頼(examples/first-workspace)と手書きの宣言・表に固定されている。同じ段階(browse-pattern、command-pattern-binding、environment-database-binding、physical-schema)の実行前確認を受注へ当て、駆動部のInvokeCapabilityで手書きのアダプターを呼ぶ"}},
		Output: map[string]any{
			"registration": reg, "missingItemsFromStep4": c.missing["create"],
			"note": "工程4の不足項目(capability、binding、persistence、provider、authority、independent-read)のうち、手書きのアダプター、別プールの読取りを補った。authority(認可)は補っていない",
		},
	})
	c.gap("実行時", "保存の実行前確認が受注に使えない(受付依頼に固定)。受注の契約sales_core.order v4.2は書込みを状態の列に限るが、手書きのアダプターはその範囲を検査されずに全列を書く", "サンプル側の実行前確認。契約の書込み範囲の検査は未実装のまま記録")
	return nil
}

// Step8ListAndDetail lists the orders of the converged database and passes the
// identifier of the selected row to the header-detail query unchanged.
func (c *Chain) Step8ListAndDetail(ctx context.Context) error {
	// In the same run, a copy changed after step 3 is refused first.
	tampered := filepath.Join(c.Dir, "workspace-tampered")
	if err := CopyWorkspace(c.WorkDir, tampered); err != nil {
		return err
	}
	if _, err := applyLineEdit(tampered, LineEdit{File: "semantic/sales/order.tei", Mode: "after", Anchor: "    attribute " + c.Decl.Attribute.Symbol + " type " + c.Decl.Attribute.Type, Lines: []string{"    attribute deliveryWindow type std.text"}}); err != nil {
		return err
	}
	tamperErr := c.require(8, tampered)
	var mm *RevisionMismatch
	c.Rec.Completion = append(c.Rec.Completion, Check{ID: "版の変更で停止", Step: "8",
		Injected: "工程7の後、最終版の複製をさらに変更した複製(属性をもう一つ追加)を工程8へ渡す",
		Expect:   "版の照合値が最終版と一致せず、DBへ触れる前に停止する",
		Observed: map[string]any{"error": errString(tamperErr)},
		Passed:   errors.As(tamperErr, &mm) && mm.Want == c.final,
	})
	if err := c.require(8, c.WorkDir); err != nil {
		return err
	}
	st := c.db
	list, err := c.listOrders(ctx, st.db)
	if err != nil {
		return fmt.Errorf("step 8 list: %w", err)
	}
	var attrs []string
	for _, col := range list.Columns {
		attrs = append(attrs, string(col.Attribute))
	}
	hasNew := false
	for _, a := range attrs {
		if a == c.Decl.Attribute.Attribute {
			hasNew = true
		}
	}
	if !hasNew {
		return fmt.Errorf("step 8: list lacks %s: %v", c.Decl.Attribute.Attribute, attrs)
	}
	resolver := driver.NewResolver(c.worldFinal)
	registry := driver.ContractQueryExecutorMap{
		{Environment: environmentID, Instance: "sales_core.dev", Capability: "database.query.postgres"}: sqlquery.Executor{
			Provider: environment.ProviderPostgres, Realizations: c.worldFinal.ContractQuerySQLRealizations, Runner: postgresquery.Runner{DB: st.db},
		},
	}
	detail := func(orderID int64) (int, []any, []string, error) {
		row, ok := findRow(list, "sales.order.order_id", orderID)
		if !ok {
			return 0, nil, nil, fmt.Errorf("order %d is not in the list", orderID)
		}
		exec, err := resolver.ExecuteHeaderDetailContractQueryInEnvironment(ctx, environmentID, applicationID, detailPatternID, list, row, registry)
		if err != nil {
			return 0, nil, nil, err
		}
		var ids []any
		for _, r := range exec.Execution.Result.Rows {
			ids = append(ids, r.Attributes["sales.order_line.order_line_id"])
		}
		var params []string
		if p, ok := exec.Execution.Result.Payload.(postgresquery.Result); ok {
			params = p.ParameterNames
		}
		return len(exec.Execution.Result.Rows), ids, params, nil
	}
	withLines, lineIDs, params, err := detail(1001)
	if err != nil {
		return fmt.Errorf("step 8 detail: %w", err)
	}
	none, _, _, err := detail(1003)
	if err != nil {
		return fmt.Errorf("step 8 detail (no lines): %w", err)
	}
	// An execution failure is not an empty collection.
	if _, err := st.db.ExecContext(ctx, `ALTER TABLE public.order_lines RENAME TO order_lines_hidden`); err != nil {
		return err
	}
	_, _, _, failErr := detail(1001)
	if _, err := st.db.ExecContext(ctx, `ALTER TABLE public.order_lines_hidden RENAME TO order_lines`); err != nil {
		return err
	}
	if withLines != 2 || none != 0 || failErr == nil || len(params) != 1 || params[0] != "order_id" {
		return fmt.Errorf("step 8: lines %d, none %d, failure %v, params %v", withLines, none, failErr, params)
	}
	c.Rec.Completion = append(c.Rec.Completion, Check{ID: "一覧の識別子を明細へ渡す", Step: "8",
		Expect:   "一覧で選んだ受注の行(MappedResultと行番号)をそのまま親子明細の取得へ渡す。明細のない受注は0件、表の欠落は実行失敗として区別する",
		Observed: map[string]any{"listRows": len(list.Rows), "selectedOrder": 1001, "lines": withLines, "lineIds": lineIDs, "parameterNames": params, "orderWithoutLines": none, "executionFailure": errString(failErr)},
		Passed:   true,
	})
	c.addStep(StepRecord{
		Step: 8, Name: "一覧と親子明細の取得", ModelRevision: c.final,
		Assets: []AssetUse{
			{Asset: "tools/reference-postgres-header-detail-runtime", Called: false, Reason: "package mainで固定の表と行を作り、判定段階Bが既定の動作と出力を固定している。同じ内部の入口ExecuteHeaderDetailContractQueryInEnvironmentを、一覧の結果を受けて呼ぶ"},
			{Asset: "(一覧の入口)", Called: false, Reason: "受注の一覧を取得するtools側の入口は#1197で削除済み。駆動部のExecuteBrowseQueryとpostgresquery.Executorで新設"},
		},
		Output: map[string]any{"listAttributes": attrs, "listRows": len(list.Rows), "detail": map[string]any{"order": 1001, "lines": withLines, "lineIds": lineIDs}, "noLines": none, "failureDistinguished": failErr != nil},
	})
	return nil
}

// Step9Register registers through the gate, re-reads on another pool, and
// injects duplicate, concurrent, lost-connection, re-read loss and issuer stop
// faults.
func (c *Chain) Step9Register(ctx context.Context) error {
	if err := c.require(9, c.WorkDir); err != nil {
		return err
	}
	st := c.db
	num := numbering{db: st.db, revision: c.final}
	iss := newIssuer(70001)
	defer func() { iss.srv.Close() }()
	policy, prefix := c.Decl.Numbering.ExternalNumber.OnUnavailable, c.Decl.Numbering.ExternalNumber.ProvisionalPrefix
	poolA, err := newPool(st.dsn)
	if err != nil {
		return err
	}
	defer poolA.Close()
	poolB, err := newPool(st.dsn)
	if err != nil {
		return err
	}
	adapterA := c.newAdapter(poolA)
	pre, err := c.listOrders(ctx, poolB)
	if err != nil {
		return err
	}
	n0 := len(pre.Rows)
	out := map[string]any{"listBefore": n0}
	var regs []Registration
	committed := 0

	register := func(label string, a *orderAdapter, sub Submission) Registration {
		r := c.submit(ctx, label, a, sub)
		regs = append(regs, r)
		if r.Outcome == OutcomeCommitted {
			committed++
			num.event(ctx, sub.OrderID, "committed", "", sub.ExternalID)
		}
		return r
	}
	newOrder := func() (Submission, string, error) {
		id, err := num.allocate(ctx)
		if err != nil {
			return Submission{}, "", err
		}
		ext, how, err := externalNumber(ctx, iss, policy, prefix, id)
		if err != nil {
			return Submission{}, "", err
		}
		if how == "provisional" {
			num.event(ctx, id, "provisional-external-number", "issuer-unavailable", &ext)
		}
		return withIDs(c.template(), id, &ext), how, nil
	}

	// A: the same registration that step 6 refused now passes.
	subA, _, err := newOrder()
	if err != nil {
		return err
	}
	regA := register("A-after-convergence", adapterA, subA)
	match, listCount, readErr := c.reread(ctx, poolB, subA)
	var pidB int64
	poolB.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pidB)
	obsA := classifyObservation(regA.Outcome == OutcomeCommitted, readErr, match == "match")
	out["A"] = map[string]any{"registration": regA, "observation": obsA, "listAfter": listCount, "commandPid": regA.CommandPID, "readPid": pidB}
	if regA.Outcome != OutcomeCommitted || obsA != ObsObserved || listCount != n0+1 || regA.CommandPID == pidB {
		return fmt.Errorf("step 9: registration A: %+v obs %s (%s) list %d/%d", regA, obsA, match, listCount, n0+1)
	}

	// Duplicate external number: a confirmed failure.
	dupID, err := num.allocate(ctx)
	if err != nil {
		return err
	}
	regDup := register("duplicate-external-number", adapterA, withIDs(c.template(), dupID, subA.ExternalID))
	if regDup.Outcome == OutcomeConfirmedFailure {
		num.event(ctx, dupID, "cancelled", "duplicate-external-number-rejected", subA.ExternalID)
	}

	// Cancellation before commit leaves a gap.
	subC, _, err := newOrder()
	if err != nil {
		return err
	}
	cancelAdapter := c.newAdapter(poolA)
	cancelAdapter.hooks.beforeCommit = func(context.Context, *sql.Tx, int64) error { return errCancelled }
	regC := register("cancelled-by-user", cancelAdapter, subC)
	if regC.Outcome == OutcomeCancelled {
		num.event(ctx, subC.OrderID, "cancelled", "cancelled-by-user-before-commit", subC.ExternalID)
	}

	// Two connections register the same external number almost at once.
	concurrent, err := c.concurrentSameExternal(ctx, num, iss, &regs, &committed)
	if err != nil {
		return err
	}

	// The connection is lost before commit: the result is unknown until a
	// separate query settles it.
	subU, _, err := newOrder()
	if err != nil {
		return err
	}
	poolU, err := newPool(st.dsn)
	if err != nil {
		return err
	}
	lostAdapter := c.newAdapter(poolU)
	lostAdapter.hooks.beforeCommit = func(ctx context.Context, _ *sql.Tx, pid int64) error {
		_, err := st.db.ExecContext(ctx, `SELECT pg_terminate_backend($1)`, pid)
		return err
	}
	regU := register("connection-lost", lostAdapter, subU)
	poolU.Close()
	present, err := countRows(ctx, st.db, `SELECT count(*) FROM public.orders WHERE order_id = $1`, subU.OrderID)
	if err != nil {
		return err
	}
	settled := "not-committed"
	if present == 1 {
		settled = "committed"
		committed++
	} else {
		num.event(ctx, subU.OrderID, "cancelled", "connection-lost-settled-not-committed", subU.ExternalID)
	}
	c.Rec.Completion = append(c.Rec.Completion, Check{ID: "確定した失敗と結果不明の区別", Step: "9",
		Expect:   "重複の拒否は確定した失敗(SQLSTATE 23505)、接続の喪失は結果不明とし、結果不明は別の接続の照会で決着させる",
		Observed: map[string]any{"duplicate": regDup.Outcome + " " + regDup.SQLState, "connectionLost": regU.Outcome + " " + regU.SQLState, "settledByQuery": settled},
		Passed:   regDup.Outcome == OutcomeConfirmedFailure && regDup.SQLState == "23505" && regU.Outcome == OutcomeUnknown,
	})

	// P-TX-02: the command commits, then the re-read cannot be made.
	subD, _, err := newOrder()
	if err != nil {
		return err
	}
	regD := register("D-read-pool-closed", adapterA, subD)
	poolB.Close()
	_, _, errD := c.reread(ctx, poolB, subD)
	obsD := classifyObservation(regD.Outcome == OutcomeCommitted, errD, false)
	poolC, err := newPool(st.dsn)
	if err != nil {
		return err
	}
	defer poolC.Close()
	matchD, _, errD2 := c.reread(ctx, poolC, subD)
	obsD2 := classifyObservation(regD.Outcome == OutcomeCommitted, errD2, matchD == "match")

	subE, _, err := newOrder()
	if err != nil {
		return err
	}
	regE := register("E-table-lost", adapterA, subE)
	hidden := "orders_hidden_" + c.Rec.RunID
	if _, err := st.db.ExecContext(ctx, `ALTER TABLE public.orders RENAME TO `+quoteIdent(hidden)); err != nil {
		return err
	}
	_, _, errE := c.reread(ctx, poolC, subE)
	obsE := classifyObservation(regE.Outcome == OutcomeCommitted, errE, false)
	if _, err := st.db.ExecContext(ctx, `ALTER TABLE public.`+quoteIdent(hidden)+` RENAME TO orders`); err != nil {
		return err
	}
	matchE, _, errE2 := c.reread(ctx, poolC, subE)
	obsE2 := classifyObservation(regE.Outcome == OutcomeCommitted, errE2, matchE == "match")
	c.fault(Check{ID: "P-TX-02-登録後の再読込の喪失", Step: "9",
		Injected: "登録のコミット後に、(a)読取り用の接続を閉じる、(b)表を失わせる(名前を変える)",
		Expect:   "登録の結果(コミット済み)と別接続の観測(read-failed)を別の根拠として記録し、観測できない間はobservedにしない。読取りが戻った後の再読込でobservedになる",
		Observed: map[string]any{
			"a": map[string]any{"command": regD.Outcome, "observation": obsD, "readError": errString(errD), "laterObservation": obsD2},
			"b": map[string]any{"command": regE.Outcome, "observation": obsE, "readError": errString(errE), "laterObservation": obsE2},
		},
		Passed: regD.Outcome == OutcomeCommitted && obsD == ObsReadFailed && obsD2 == ObsObserved &&
			regE.Outcome == OutcomeCommitted && obsE == ObsReadFailed && obsE2 == ObsObserved,
	})

	// P-INT-03: the issuer stops; the declared policy decides.
	iss.stop()
	subF, howF, errF := newOrder()
	var regF Registration
	var reassigned string
	var obsF string
	if errF == nil {
		regF = register("F-issuer-stopped", adapterA, subF)
		iss.start()
		n, err := iss.issue(ctx)
		if err != nil {
			return err
		}
		res, err := st.db.ExecContext(ctx, `UPDATE public.orders SET external_id = $1 WHERE order_id = $2 AND external_id = $3`, n, subF.OrderID, *subF.ExternalID)
		if err != nil {
			return err
		}
		if k, _ := res.RowsAffected(); k == 1 {
			reassigned = n
			num.event(ctx, subF.OrderID, "reassigned", "issuer-recovered", &n)
		}
		subF.ExternalID = &reassigned
		m, _, err := c.reread(ctx, poolC, subF)
		obsF = classifyObservation(regF.Outcome == OutcomeCommitted, err, m == "match")
	}
	gaps, err := num.gaps(ctx)
	if err != nil {
		return err
	}
	final, err := c.listOrders(ctx, poolC)
	if err != nil {
		return err
	}
	policyOK := false
	switch policy {
	case PolicyProvisional:
		policyOK = errF == nil && howF == "provisional" && regF.Outcome == OutcomeCommitted && reassigned != "" && obsF == ObsObserved
	case PolicyStop:
		policyOK = errors.Is(errF, ErrIssuerUnavailable)
	}
	c.fault(Check{ID: "P-INT-03-欠番と採番元の停止", Step: "9",
		Injected: "重複の拒否、利用者の取消し、同時登録の敗者、接続の喪失で受注番号の欠番を作り、外部番号の採番元(HTTP)を止める",
		Expect:   "欠番がすべて採番の記録(取消しの理由)で説明でき、採番元の停止時に宣言した方針どおりに振る舞う",
		Observed: map[string]any{
			"gaps": gaps, "declaredPolicy": policy, "issuerStopped": map[string]any{"how": howF, "error": errString(errF), "registration": regF.Outcome, "provisional": ptrValue(subF.ExternalID), "reassignedTo": reassigned, "observation": obsF},
		},
		Passed: len(gaps.Gaps) >= 3 && len(gaps.Unexplained) == 0 && len(gaps.UnrecordedUsed) == 0 && policyOK,
	})
	out["registrations"] = regs
	out["concurrentSameExternalNumber"] = concurrent
	out["listAfter"] = len(final.Rows)
	out["committed"] = committed
	if len(final.Rows) != n0+committed {
		return fmt.Errorf("step 9: list has %d rows, want %d", len(final.Rows), n0+committed)
	}
	c.addStep(StepRecord{
		Step: 9, Name: "登録、別接続での再読込、重複拒否", ModelRevision: c.final,
		Assets: []AssetUse{{Asset: "tools/reference-data-first-persistence", Called: false, Reason: "工程6と同じ(受付依頼に固定)。駆動部のInvokeCapability、ExecuteObservationRead、ExecuteBrowseQueryを受注へ当てる"}},
		Output: out,
	})
	c.gap("データ", "業務キーの採番(採番元、欠番の許容、欠番の理由、採番元の停止時の方針)を宣言できない", "サンプルの宣言(numbering)と台帳(odr_meta.numbering_event)")
	return nil
}

// concurrentSameExternal makes two connections register the same external
// number: the first holds its transaction open until the second is observed
// waiting on the unique index, then commits.
func (c *Chain) concurrentSameExternal(ctx context.Context, num numbering, iss *issuer, regs *[]Registration, committed *int) (map[string]any, error) {
	st := c.db
	idX, err := num.allocate(ctx)
	if err != nil {
		return nil, err
	}
	idY, err := num.allocate(ctx)
	if err != nil {
		return nil, err
	}
	ext, err := iss.issue(ctx)
	if err != nil {
		return nil, err
	}
	p1, err := newPool(st.dsn)
	if err != nil {
		return nil, err
	}
	defer p1.Close()
	p2, err := newPool(st.dsn)
	if err != nil {
		return nil, err
	}
	defer p2.Close()
	a1, a2 := c.newAdapter(p1), c.newAdapter(p2)
	inserted := make(chan struct{})
	release := make(chan struct{})
	a1.hooks.beforeCommit = func(ctx context.Context, _ *sql.Tx, _ int64) error {
		close(inserted)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	pid2 := make(chan int64, 1)
	a2.hooks.onBegin = func(pid int64) { pid2 <- pid }
	res1 := make(chan Registration, 1)
	res2 := make(chan Registration, 1)
	go func() { res1 <- c.submit(ctx, "X-first", a1, withIDs(c.template(), idX, &ext)) }()
	select {
	case <-inserted:
	case r := <-res1:
		return nil, fmt.Errorf("first registration ended early: %+v", r)
	}
	go func() { res2 <- c.submit(ctx, "Y-second", a2, withIDs(c.template(), idY, &ext)) }()
	var pid int64
	select {
	case pid = <-pid2:
	case r := <-res2:
		close(release)
		return nil, fmt.Errorf("second registration ended early: %+v", r)
	}
	waited := false
	for i := 0; i < 2000 && !waited; i++ {
		n, err := countRows(ctx, st.db, `SELECT count(*) FROM pg_catalog.pg_stat_activity WHERE pid = $1 AND wait_event_type = 'Lock'`, pid)
		if err != nil {
			close(release)
			return nil, err
		}
		waited = n == 1
		if !waited {
			time.Sleep(2 * time.Millisecond)
		}
	}
	close(release)
	r1, r2 := <-res1, <-res2
	*regs = append(*regs, r1, r2)
	for _, r := range []Registration{r1, r2} {
		switch r.Outcome {
		case OutcomeCommitted:
			*committed++
			num.event(ctx, r.OrderID, "committed", "", &ext)
		case OutcomeConfirmedFailure:
			num.event(ctx, r.OrderID, "cancelled", "concurrent-duplicate-external-number-rejected", &ext)
		}
	}
	rows, err := countRows(ctx, st.db, `SELECT count(*) FROM public.orders WHERE external_id = $1`, ext)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"externalId": ext, "secondWasWaitingOnLock": waited, "first": r1.Outcome, "second": r2.Outcome + " " + r2.SQLState, "rowsWithExternalId": rows}
	c.fault(Check{ID: "固有-同じ外部番号の同時登録", Step: "9",
		Injected: "二つの接続が同じ外部番号をほぼ同時に登録する(先の接続はコミットの直前で待ち、後の接続が一意索引の待ちに入ったのを確かめてからコミットする)",
		Expect:   "一件だけ成立し、後の登録は確定した失敗(23505)になる。先にコミットされた登録は巻き戻らない",
		Observed: out,
		Passed:   waited && r1.Outcome == OutcomeCommitted && r2.Outcome == OutcomeConfirmedFailure && r2.SQLState == "23505" && rows == 1,
	})
	return out, nil
}

var _ = core.CanonicalID("")
