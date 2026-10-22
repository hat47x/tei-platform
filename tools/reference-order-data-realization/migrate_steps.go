package orderdatarealization

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
	"tei/tools/internal/postgresdb"
	ddlentry "tei/tools/reference-ddl-entry"
	scenariodiff "tei/tools/reference-scenario-state-diff"
)

type dbState struct {
	dsn      string
	db       *sql.DB
	safety   DatabaseSafety
	instance environment.DatabaseInstance

	desiredR0, desiredFinal, desiredVariant physical.SchemaSnapshot
	plan                                    *MigrationPlan
	wFinal, wR0                             rowWriter

	seeded      []int64
	seedDigest  string
	oldWriterID []int64
}

var orderR0Columns = []string{"order_id", "customer_id", "ordered_at", "status", "total_amount", "external_id"}

// baseTime is the fixed business time of the synthetic rows; no wall clock is
// used for any judgement.
var baseTime = time.Date(2026, 10, 1, 9, 0, 0, 0, time.FixedZone("JST", 9*3600))

func strp(s string) *string { return &s }

// Step5BuildPlan builds the old structure from R0, seeds rows, observes,
// derives the expected state from the final version (and from the variant
// without decisions) and plans the migration bound to the observation.
func (c *Chain) Step5BuildPlan(ctx context.Context, dsn string) error {
	if err := c.require(5, c.WorkDir); err != nil {
		return err
	}
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		return err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("step 5: connect: %w", err)
	}
	st := &dbState{dsn: dsn, db: db, instance: environment.DatabaseInstance{ID: "sales_core.dev", Definition: databaseID, Provider: environment.ProviderPostgres}}
	c.db = st
	marker := ownerPrefix + c.Rec.RunID
	if st.safety, err = checkDedicatedEmpty(ctx, db, marker); err != nil {
		return fmt.Errorf("step 5: %w", err)
	}
	ddl, err := ddlentry.Render(c.worldR0, databaseID, environment.ProviderPostgres)
	if err != nil {
		return fmt.Errorf("step 5: ddl-entry: %w", err)
	}
	if len(ddl.Unresolved) > 0 {
		return fmt.Errorf("step 5: ddl-entry left tables unresolved: %+v", ddl.Unresolved)
	}
	var stmts []string
	for _, t := range ddl.Tables {
		stmts = append(stmts, t.Statements...)
		q := `"public".` + quoteIdent(t.Name)
		stmts = append(stmts, markStatement("TABLE", q, marker))
		st.safety.Created = append(st.safety.Created, "public."+t.Name)
	}
	stmts = append(stmts,
		"CREATE SCHEMA "+metaSchema, markStatement("SCHEMA", metaSchema, marker),
		"CREATE TABLE "+metaSchema+".writer (writer_id text PRIMARY KEY, model_revision text NOT NULL, state text NOT NULL CHECK (state IN ('active','retired')))",
		markStatement("TABLE", metaSchema+".writer", marker),
		"CREATE SEQUENCE "+metaSchema+".order_no_seq START 5001 CACHE 1",
		markStatement("SEQUENCE", metaSchema+".order_no_seq", marker),
		"CREATE TABLE "+metaSchema+".numbering_event (seq bigserial PRIMARY KEY, order_id bigint, event text NOT NULL, reason text, external_id text, model_revision text NOT NULL)",
		markStatement("TABLE", metaSchema+".numbering_event", marker),
	)
	st.safety.Created = append(st.safety.Created, "schema "+metaSchema, metaSchema+".writer", metaSchema+".order_no_seq", metaSchema+".numbering_event")
	if _, err := execWithLockTimeout(ctx, db, c.Decl.Migration.LockTimeoutMillis, stmts); err != nil {
		return fmt.Errorf("step 5: create old structure: %w", err)
	}
	if st.wR0, err = rowWriterOf(c.worldR0); err != nil {
		return err
	}
	if st.wFinal, err = rowWriterOf(c.worldFinal); err != nil {
		return err
	}
	// Seed rows through the R0 mapping (the old structure has no new column).
	seed := []struct {
		id       int64
		ext      any
		lines    int
		minutes  int
		total    string
	}{
		{1001, "S-1001", 2, 0, "3200.00"},
		{1002, "000123", 1, 30, "800.00"},
		{1003, nil, 0, 60, "0.00"},
		{1004, "S-1004", 0, 90, "12000.50"},
	}
	lineID := int64(1100)
	for _, s := range seed {
		q, args, err := insertFor(st.wR0.order, map[string]any{
			"sales.order.order_id": s.id, "sales.order.customer_id": int64(1), "sales.order.ordered_at": baseTime.Add(time.Duration(s.minutes) * time.Minute),
			"sales.order.status": "received", "sales.order.total": s.total, "sales.order.external_id": s.ext,
		}, nil)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, q, args...); err != nil {
			return fmt.Errorf("step 5 seed: %w", err)
		}
		for l := 1; l <= s.lines; l++ {
			lineID++
			q, args, err := insertFor(st.wR0.line, map[string]any{
				"sales.order_line.order_line_id": lineID, "sales.order_line.line_number": int64(l), "sales.order_line.item_code": fmt.Sprintf("SKU-%03d", l),
				"sales.order_line.quantity": "2", "sales.order_line.unit_price": "800.00",
			}, map[string]any{"sales.order.order_id": s.id})
			if err != nil {
				return err
			}
			if _, err := db.ExecContext(ctx, q, args...); err != nil {
				return fmt.Errorf("step 5 seed line: %w", err)
			}
		}
		st.seeded = append(st.seeded, s.id)
	}
	if st.seedDigest, _, err = rowsDigest(ctx, db, orderR0Columns, st.seeded); err != nil {
		return err
	}
	obs, err := observe(ctx, db, st.instance)
	if err != nil {
		return fmt.Errorf("step 5 observe: %w", err)
	}
	if st.desiredR0, err = desiredOf(c.worldR0); err != nil {
		return err
	}
	if st.desiredFinal, err = desiredOf(c.worldFinal); err != nil {
		return err
	}
	if st.desiredVariant, err = desiredOf(c.worldVariant); err != nil {
		return err
	}
	r0Diff, err := schemadiff.Compare(st.desiredR0, obs.Snapshot)
	if err != nil {
		return err
	}
	if len(r0Diff.Changes) != 0 || len(r0Diff.Unresolved) != 0 {
		return fmt.Errorf("step 5: the old structure does not match R0: %+v", r0Diff)
	}
	variantDiff, err := schemadiff.Compare(st.desiredVariant, obs.Snapshot)
	if err != nil {
		return err
	}
	plan, err := planMigration("odr-"+c.Rec.RunID, c.final, c.Decl.Migration, st.desiredFinal, obs.Snapshot)
	if err != nil {
		return fmt.Errorf("step 5 plan: %w", err)
	}
	st.plan = plan
	variantProbe, err := c.variantDuplicateProbe(ctx, variantDiff)
	if err != nil {
		return fmt.Errorf("step 5 variant probe: %w", err)
	}
	after, err := observe(ctx, db, st.instance)
	if err != nil {
		return err
	}
	if observationFingerprint(after.Snapshot) != plan.BasedOnObservation {
		return fmt.Errorf("step 5: the rolled-back variant probe changed the database")
	}
	kinds := func(d schemadiff.Diff) []string {
		var k []string
		for _, ch := range d.Changes {
			k = append(k, string(ch.Kind)+" "+ch.Table)
		}
		return k
	}
	c.addStep(StepRecord{
		Step: 5, Name: "旧構造のDBの構築、差分の検出、移行の計画", ModelRevision: c.final,
		Assets: []AssetUse{
			{Asset: "tools/reference-ddl-entry", Called: true, How: "Goの取込み: ddlentry.Render(R0の世界, sales_core, postgres)で旧構造の表を作る"},
			{Asset: "tools/reference-postgres-integration", Called: false, Reason: "package mainで作業領域を受け取らず、表を手組みし、無条件のDROP SCHEMA ... CASCADEを持つ。同じ内部部品(RealizeDesired、postgres.Provider、schemadiff.Compare)を直接使う"},
			{Asset: "tools/reference-postgres-migration-rendering", Called: false, Reason: "package mainで固定の表と索引だけを扱い、無条件のDROPを持つ。同じ内部部品(migration.Plan、migration.Render)を直接使う"},
		},
		Output: map[string]any{
			"databaseSafety":            st.safety,
			"oldStructureMatchesR0":     true,
			"seededOrders":              st.seeded,
			"seedDigestR0Columns":       st.seedDigest,
			"observation":               plan.BasedOnObservation,
			"diffFinal":                 kinds(plan.Diff),
			"diffVariantWithoutDecisions": kinds(variantDiff),
			"diffsDiffer":               fmt.Sprint(kinds(plan.Diff)) != fmt.Sprint(kinds(variantDiff)),
			"plan":                      plan,
			"variantDuplicateProbe":     variantProbe,
			"comparisonProjection":      "観測は保存写像が使うスキーマ(public)に限る。サンプルの管理表はodr_metaに置く",
		},
	})
	c.gap("ツール", "移行の描画(PostgreSQLの提供元)が一意制約の追加を描画できない", "契約(contract)段階の文をサンプルが描画する。providerの拒否の文言を計画に記録")
	c.gap("コア", "移行計画の観測の照合値(BasedOnActualFingerprint)を適用の前に照合していない。構造の観測の照合値を作る公開の関数もない", "サンプルが観測の照合値を計算し、適用の直前に再観測して照合する")
	return nil
}

// variantDuplicateProbe applies the variant's migration (no decisions) in a
// transaction, registers two orders with the same external number, and rolls
// back.
func (c *Chain) variantDuplicateProbe(ctx context.Context, diff schemadiff.Diff) (map[string]any, error) {
	st := c.db
	stmts, err := migration.Render(environment.ProviderPostgres, diff)
	if err != nil {
		return nil, err
	}
	wv, err := rowWriterOf(c.worldVariant)
	if err != nil {
		return nil, err
	}
	tx, err := st.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL lock_timeout = '%dms'", c.Decl.Migration.LockTimeoutMillis)); err != nil {
		return nil, err
	}
	for _, s := range stmts {
		if _, err := tx.ExecContext(ctx, s); err != nil {
			return nil, err
		}
	}
	accepted := 0
	for i, id := range []int64{8801, 8802} {
		q, args, err := insertFor(wv.order, map[string]any{
			"sales.order.order_id": id, "sales.order.customer_id": int64(1), "sales.order.ordered_at": baseTime,
			"sales.order.status": "received", "sales.order.total": "1.00", "sales.order.external_id": "0079999",
			c.Decl.Attribute.Attribute: fmt.Sprintf("variant %d", i),
		}, nil)
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, q, args...); err == nil {
			accepted++
		}
	}
	if err := tx.Rollback(); err != nil {
		return nil, err
	}
	return map[string]any{"statements": stmts, "sameExternalNumberAccepted": accepted, "rolledBack": true}, nil
}

// oldWriter is the registration process of the previous version: it builds
// its INSERT from the R0 storage mapping and does not know the new attribute.
type oldWriter struct {
	db        *sql.DB
	w         rowWriter
	id        string
	next      int64
	mu        sync.Mutex
	committed []int64
	errs      []string
	n         atomic.Int64
	stop      chan struct{}
	done      chan struct{}
}

func startOldWriter(ctx context.Context, dsn string, w rowWriter, revision string, first int64) (*oldWriter, error) {
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	o := &oldWriter{db: db, w: w, id: "legacy-order-entry", next: first, stop: make(chan struct{}), done: make(chan struct{})}
	if _, err := db.ExecContext(ctx, `INSERT INTO `+metaSchema+`.writer VALUES ($1, $2, 'active')`, o.id, revision); err != nil {
		db.Close()
		return nil, err
	}
	go o.loop(ctx)
	return o, nil
}

func (o *oldWriter) loop(ctx context.Context) {
	defer close(o.done)
	for {
		select {
		case <-o.stop:
			return
		case <-ctx.Done():
			return
		default:
		}
		id := o.next
		o.next++
		var ext any
		if id%2 == 0 {
			ext = fmt.Sprintf("LEG-%05d", id)
		}
		q, args, err := insertFor(o.w.order, map[string]any{
			"sales.order.order_id": id, "sales.order.customer_id": int64(2), "sales.order.ordered_at": baseTime.Add(time.Duration(id-9000) * time.Second),
			"sales.order.status": "legacy-entry", "sales.order.total": "1000.00", "sales.order.external_id": ext,
		}, nil)
		if err == nil {
			err = func() error {
				tx, err := o.db.BeginTx(ctx, nil)
				if err != nil {
					return err
				}
				defer tx.Rollback()
				if _, err := tx.ExecContext(ctx, q, args...); err != nil {
					return err
				}
				// Hold the row lock briefly so that DDL has to wait for it.
				if _, err := tx.ExecContext(ctx, `SELECT pg_sleep(0.01)`); err != nil {
					return err
				}
				return tx.Commit()
			}()
		}
		if err != nil {
			o.mu.Lock()
			o.errs = append(o.errs, err.Error())
			o.mu.Unlock()
			continue
		}
		o.mu.Lock()
		o.committed = append(o.committed, id)
		o.mu.Unlock()
		o.n.Add(1)
	}
}

func (o *oldWriter) waitFor(ctx context.Context, n int64) error {
	for o.n.Load() < n {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-o.done:
			return fmt.Errorf("old writer stopped after %d rows: %v", o.n.Load(), o.errs)
		case <-time.After(2 * time.Millisecond):
		}
	}
	return nil
}

func (o *oldWriter) stopAndRetire(ctx context.Context) error {
	close(o.stop)
	<-o.done
	defer o.db.Close()
	_, err := o.db.ExecContext(ctx, `UPDATE `+metaSchema+`.writer SET state = 'retired' WHERE writer_id = $1`, o.id)
	return err
}

func columnExists(ctx context.Context, db *sql.DB, table, column string) (bool, error) {
	n, err := countRows(ctx, db, `SELECT count(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2`, table, column)
	return n > 0, err
}

func uniqueExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	n, err := countRows(ctx, db, `SELECT count(*) FROM pg_catalog.pg_constraint WHERE conname = $1 AND contype = 'u'`, name)
	return n > 0, err
}

// Step7ApplyAndConverge injects a concurrent structure change between plan
// and apply, then applies the expand stage while the old-version writer keeps
// writing, refuses the contract stage while it remains, and converges.
func (c *Chain) Step7ApplyAndConverge(ctx context.Context) error {
	if err := c.require(7, c.WorkDir); err != nil {
		return err
	}
	st := c.db
	lock := c.Decl.Migration.LockTimeoutMillis
	col := c.Decl.Attribute.Column
	out := map[string]any{}

	// Fault (固有 5・7): another operator changes the table after planning.
	other, err := postgresdb.OpenSQL(st.dsn)
	if err != nil {
		return err
	}
	defer other.Close()
	if _, err := other.ExecContext(ctx, `ALTER TABLE public.orders ADD COLUMN drift_note text`); err != nil {
		return err
	}
	_, applyErr := applyExpand(ctx, st.db, st.instance, st.plan, lock)
	stopped := errors.Is(applyErr, ErrObservationChanged)
	added, _ := columnExists(ctx, st.db, "orders", col)
	fresh, err := observe(ctx, st.db, st.instance)
	if err != nil {
		return err
	}
	replan, replanErr := planMigration("odr-"+c.Rec.RunID+"-replan", c.final, c.Decl.Migration, st.desiredFinal, fresh.Snapshot)
	var needs []string
	if replan != nil {
		for _, ch := range replan.NeedsDecision {
			name := ""
			if ch.BeforeColumn != nil {
				name = ch.BeforeColumn.Name
			} else if ch.Column != nil {
				name = ch.Column.Name
			}
			needs = append(needs, string(ch.Kind)+" "+ch.Table+"."+name)
		}
	}
	c.fault(Check{ID: "固有-計画後の構造変更", Step: "5・7",
		Injected: "計画の後、適用の前に、別の接続がpublic.ordersへ列drift_noteを追加する",
		Expect:   "適用直前の観測の照合値が計画と食い違い、文を一つも実行せずに停止する。物理だけにある列の削除を含む再計画は自動で適用しない",
		Observed: map[string]any{"applyError": errString(applyErr), "stoppedBeforeExecution": stopped && !added, "replanError": errString(replanErr), "needsDecision": needs},
		Passed:   stopped && !added && errors.Is(replanErr, ErrNeedsDecision) && len(needs) == 1,
	})
	if _, err := other.ExecContext(ctx, `ALTER TABLE public.orders DROP COLUMN drift_note`); err != nil {
		return err
	}
	fresh, err = observe(ctx, st.db, st.instance)
	if err != nil {
		return err
	}
	plan, err := planMigration("odr-"+c.Rec.RunID+"-2", c.final, c.Decl.Migration, st.desiredFinal, fresh.Snapshot)
	if err != nil {
		return fmt.Errorf("step 7 replan: %w", err)
	}
	out["replannedAfterRevert"] = map[string]any{"observation": plan.BasedOnObservation, "sameAsOriginalPlan": plan.BasedOnObservation == st.plan.BasedOnObservation}
	st.plan = plan

	// Fault (P-MIG-03 拡張): the old-version writer keeps writing.
	writer, err := startOldWriter(ctx, st.dsn, st.wR0, c.Rec.Revisions.R0, 9001)
	if err != nil {
		return err
	}
	retired := false
	defer func() {
		if !retired {
			writer.stopAndRetire(context.Background())
		}
	}()
	if err := writer.waitFor(ctx, 3); err != nil {
		return err
	}
	beforeExpand := writer.n.Load()
	expand, err := applyExpand(ctx, st.db, st.instance, plan, lock)
	if err != nil {
		return fmt.Errorf("step 7 expand: %w", err)
	}
	if err := writer.waitFor(ctx, writer.n.Load()+3); err != nil {
		return err
	}
	_, contractErr := applyContract(ctx, st.db, st.instance, plan, st.desiredFinal, c.final, lock)
	refused := errors.Is(contractErr, ErrOldWritersRemain)
	uniqueWhileOld, _ := uniqueExists(ctx, st.db, c.Decl.UniqueDecision.Name)
	// The expand stage can still be undone before contract.
	reversible := func() error {
		tx, err := st.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL lock_timeout = '%dms'", lock)); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `ALTER TABLE public.orders DROP COLUMN `+quoteIdent(col))
		return err
	}()
	if err := writer.stopAndRetire(ctx); err != nil {
		return err
	}
	retired = true
	writer.mu.Lock()
	st.oldWriterID = append([]int64(nil), writer.committed...)
	writerErrs := append([]string(nil), writer.errs...)
	writer.mu.Unlock()
	nonNull, err := countRows(ctx, st.db, fmt.Sprintf(`SELECT count(*) FROM public.orders WHERE order_id >= 9001 AND %s IS NOT NULL`, quoteIdent(col)))
	if err != nil {
		return err
	}
	writtenAfterExpand := int64(len(st.oldWriterID)) - beforeExpand
	c.fault(Check{ID: "P-MIG-03-旧版の書込みと共存する移行", Step: "7",
		Injected: "移行の適用中に、R0の保存写像で組んだ旧版の登録処理が書き込み続ける",
		Expect:   "ロック待ちが宣言の上限内で、旧版の行でも新しい属性が宣言どおり(NULL)になり、旧版が残る間は縮約(一意制約の追加)が拒否される。縮約の前なら拡張を戻せる",
		Observed: map[string]any{
			"oldWriterRows": len(st.oldWriterID), "rowsWrittenAfterExpand": writtenAfterExpand, "oldWriterErrors": writerErrs,
			"oldWriterRowsWithNonNullNewAttribute": nonNull, "expand": expand,
			"contractError": errString(contractErr), "uniqueConstraintWhileOldWriter": uniqueWhileOld,
			"expandReversibleBeforeContract": reversible == nil, "reversibleError": errString(reversible),
		},
		Passed: expand.WithinLockTimeout && len(writerErrs) == 0 && writtenAfterExpand > 0 && nonNull == 0 && refused && !uniqueWhileOld && reversible == nil,
	})

	contract, err := applyContract(ctx, st.db, st.instance, plan, st.desiredFinal, c.final, lock)
	if err != nil {
		return fmt.Errorf("step 7 contract: %w", err)
	}
	diff, err := converged(ctx, st.db, st.instance, st.desiredFinal)
	if err != nil {
		return err
	}
	if len(diff.Changes) != 0 || len(diff.Unresolved) != 0 {
		return fmt.Errorf("step 7: not converged: %+v", diff)
	}
	digest, n, err := rowsDigest(ctx, st.db, orderR0Columns, st.seeded)
	if err != nil {
		return err
	}
	seededNull, err := countRows(ctx, st.db, fmt.Sprintf(`SELECT count(*) FROM public.orders WHERE order_id < 2000 AND %s IS NULL`, quoteIdent(col)))
	if err != nil {
		return err
	}
	total, err := countRows(ctx, st.db, `SELECT count(*) FROM public.orders`)
	if err != nil {
		return err
	}
	stateDiff, err := c.scenarioDiff(ctx)
	if err != nil {
		return err
	}
	out["expand"], out["contract"] = expand, contract
	out["convergedChanges"], out["convergedUnresolved"] = len(diff.Changes), len(diff.Unresolved)
	out["seededRowsUnchanged"] = digest == st.seedDigest && n == len(st.seeded)
	out["seededRowsNewAttributeNull"] = seededNull
	out["rowsBefore"], out["rowsAfter"] = len(st.seeded), total
	out["scenarioStateDiff"] = stateDiff
	if digest != st.seedDigest || seededNull != len(st.seeded) || total != len(st.seeded)+len(st.oldWriterID) {
		return fmt.Errorf("step 7: existing rows changed (digest %v, null %d, total %d)", digest == st.seedDigest, seededNull, total)
	}
	c.addStep(StepRecord{
		Step: 7, Name: "移行の適用と再観測での収束", ModelRevision: c.final,
		Assets: []AssetUse{
			{Asset: "tools/reference-postgres-migration-rendering", Called: false, Reason: "package mainで固定の表を扱う。適用は内部の入口migration.ApplyPostgresWithFreshObservationを直接使う"},
			{Asset: "tools/reference-scenario-state-diff", Called: true, How: "Goの取込み: scenariodiff.Planで、移行前の既存行と収束後の行を比べる"},
		},
		Output: out,
	})
	c.gap("データ", "旧版の書込みと共存する移行手順(拡張・縮約の段階、旧版が残る間の縮約の拒否)を提供元が持たない", "段階と拒否条件をサンプルの宣言に置き、書き手の登録表(odr_meta.writer)で判定")
	return nil
}

// scenarioDiff compares the seeded rows before the migration with all rows
// after it through the scenario state diff.
func (c *Chain) scenarioDiff(ctx context.Context) (map[string]any, error) {
	st := c.db
	col := c.Decl.Attribute.Column
	model := scenariodiff.Model{Data: []scenariodiff.DataDef{{ID: "order", Fields: []scenariodiff.Field{
		{Name: "order_id", Type: "int", Key: true}, {Name: "external_id", Type: "string"}, {Name: "status", Type: "string"},
		{Name: "total_amount", Type: "string"}, {Name: col, Type: "string"},
	}}}}
	read := func(where string) ([]map[string]any, error) {
		rows, err := st.db.QueryContext(ctx, fmt.Sprintf(`SELECT order_id, external_id, status, total_amount::text, %s FROM public.orders WHERE %s ORDER BY order_id`, quoteIdent(col), where))
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []map[string]any
		for rows.Next() {
			var id int64
			var ext, note sql.NullString
			var status, total string
			if err := rows.Scan(&id, &ext, &status, &total, &note); err != nil {
				return nil, err
			}
			out = append(out, map[string]any{"order_id": float64(id), "external_id": nullable(ext), "status": status, "total_amount": total, col: nullable(note)})
		}
		return out, rows.Err()
	}
	before, err := read("order_id < 2000")
	if err != nil {
		return nil, err
	}
	after, err := read("true")
	if err != nil {
		return nil, err
	}
	report := scenariodiff.Plan(scenariodiff.Input{Model: model, Snapshots: []scenariodiff.Snapshot{
		{ID: "before-migration", Dataset: []scenariodiff.DataSet{{Data: "order", Rows: before}}},
		{ID: "converged", Dataset: []scenariodiff.DataSet{{Data: "order", Rows: after}}},
	}})
	if len(report.SnapshotDiffs) != 1 {
		return nil, fmt.Errorf("scenario diff: %d diffs", len(report.SnapshotDiffs))
	}
	d := report.SnapshotDiffs[0]
	addedRows := map[string]bool{}
	for _, cell := range d.Added {
		addedRows[cell.Key] = true
	}
	res := map[string]any{"schema": report.Schema, "addedRows": len(addedRows), "removedCells": len(d.Removed), "changedCells": len(d.Changed), "dataFindings": len(report.DataFindings), "fingerprint": report.Fingerprint}
	if len(d.Removed) != 0 || len(d.Changed) != 0 || len(addedRows) != len(st.oldWriterID) || len(report.DataFindings) != 0 {
		raw, _ := json.Marshal(d)
		return res, fmt.Errorf("scenario diff shows unexpected change: %s findings %v", raw, report.DataFindings)
	}
	return res, nil
}

func nullable(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	return s.String
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

var _ = core.CanonicalID("")
