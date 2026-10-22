package orderdatarealization

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tei/internal/database/persistence"
	"tei/internal/database/providers/postgres"
)

// boundarySubmissions are the boundary values registered through the gate
// before the round trip.
func (c *Chain) boundarySubmissions() []Submission {
	t := c.template()
	mk := func(ext *string, note *string, at time.Time, total string) Submission {
		s := t
		s.ExternalID, s.DeliveryNote, s.OrderedAt, s.Total = ext, note, at, total
		s.Lines = []OrderLine{{LineNumber: 1, ItemCode: "SKU-0001", Quantity: "0.001", UnitPrice: "99999999999999999999.99"}}
		return s
	}
	ny := time.FixedZone("UTC-5", -5*3600)
	jst := time.FixedZone("JST", 9*3600)
	return []Submission{
		mk(strp("BND-EMPTY"), strp(""), time.Date(2026, 10, 22, 10, 0, 0, 0, jst), "1.00"),
		mk(nil, nil, time.Date(2026, 10, 22, 10, 0, 1, 0, jst), "2.00"),
		mk(strp("0000000123"), strp("先頭ゼロの外部番号"), time.Date(2026, 10, 22, 10, 0, 2, 0, jst), "3.00"),
		mk(strp("BND-BIG"), strp("桁の大きい金額"), time.Date(2026, 10, 22, 10, 0, 3, 0, jst), "123456789012345678901234.5678"),
		mk(strp("BND-NEG"), strp("負の金額と末尾のゼロ"), time.Date(2026, 10, 22, 10, 0, 4, 0, jst), "-0.0050"),
		mk(strp("BND-TZ-NY"), strp("時差 -05:00"), time.Date(2026, 10, 22, 23, 30, 0, 0, ny), "4.00"),
		mk(strp("BND-TZ-JST"), strp("時差 +09:00(同じ瞬間)"), time.Date(2026, 10, 23, 13, 30, 0, 0, jst), "5.00"),
		mk(strp("BND-USEC"), strp("マイクロ秒"), time.Date(2026, 10, 23, 4, 30, 0, 123456000, time.UTC), "6.00"),
		mk(strp("BND-JA"), strp("髙橋様宛 〜 ①番窓口 𠮷野家前 ﾃｽﾄ ‐－"), time.Date(2026, 10, 22, 10, 0, 5, 0, jst), "7.00"),
	}
}

// Step10SQLite realizes the final version in SQLite (through the sample's
// declared dialect) and round trips the registered orders and boundary
// values. A naive mapping is run as a negative control.
func (c *Chain) Step10SQLite(ctx context.Context) error {
	if err := c.require(10, c.WorkDir); err != nil {
		return err
	}
	if _, err := pythonCommand(); err != nil {
		return err
	}
	st := c.db
	pool, err := newPool(st.dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	a := c.newAdapter(pool)
	num := numbering{db: st.db, revision: c.final}
	var boundary []Registration
	for i, s := range c.boundarySubmissions() {
		id, err := num.allocate(ctx)
		if err != nil {
			return err
		}
		sub := withIDs(s, id, s.ExternalID)
		r := c.submit(ctx, fmt.Sprintf("boundary-%d", i+1), a, sub)
		if r.Outcome != OutcomeCommitted {
			return fmt.Errorf("step 10: boundary registration %d: %+v", i+1, r)
		}
		num.event(ctx, id, "committed", "", sub.ExternalID)
		boundary = append(boundary, r)
	}
	schema, err := sqliteSchema(c.worldFinal, c.Decl.SQLite)
	if err != nil {
		return err
	}
	pg := map[string]pgTable{}
	types := map[string]string{}
	for name, t := range schema {
		var qcols []string
		for _, col := range t.Columns {
			qcols = append(qcols, quoteIdent(col))
		}
		raw, err := postgres.ExecuteRawQuery(ctx, st.db, fmt.Sprintf("SELECT %s FROM public.%s ORDER BY 1", strings.Join(qcols, ", "), quoteIdent(name)), nil)
		if err != nil {
			return err
		}
		pg[name] = pgTable{columns: t.Columns, storage: t.Storage, rows: raw.Rows}
		for i, col := range t.Columns {
			for _, r := range raw.Rows {
				if r[i] != nil {
					types[string(t.Storage[i])] = fmt.Sprintf("%T", r[i])
					break
				}
			}
			_ = col
		}
	}
	list, err := c.listOrders(ctx, st.db)
	if err != nil {
		return err
	}
	order := listOrderOf(list)
	declared, err := roundTrip(ctx, c.Dir, "declared", c.worldFinal, c.Decl.SQLite, pg, order)
	if err != nil {
		return fmt.Errorf("step 10 declared dialect: %w", err)
	}
	naive, err := roundTrip(ctx, c.Dir, "naive", c.worldFinal, c.Decl.SQLiteNaive, pg, order)
	if err != nil {
		return fmt.Errorf("step 10 naive dialect: %w", err)
	}
	pgNullExternal, err := countRows(ctx, st.db, `SELECT count(*) FROM public.orders WHERE external_id IS NULL`)
	if err != nil {
		return err
	}
	probesOK := strings.HasPrefix(declared.UniqueProbes["duplicate-external-id"], "integrity-error") && declared.UniqueProbes["second-null-external-id"] == "accepted"
	passed := declared.Undeclared == 0 && declared.ValuesCompared > 0 && declared.ListOrderSame && probesOK && naive.Undeclared > 0 && declared.FileRemoved && naive.FileRemoved
	c.fault(Check{ID: "P-DB-03-SQLiteでの往復", Step: "10",
		Injected: "同じ最終版の保存写像をSQLiteへ実現し、工程9で登録した受注と境界値(空文字とNULL、先頭ゼロ、桁の大きい金額、時差のある時刻、マイクロ秒、日本語)を書いて読み戻す",
		Expect:   "宣言した方言の対応では値の差が0件(宣言した既知の差分は値の差として現れない)。素朴な対応(数値の型の宣言)では差が検出される(陰性対照)",
		Observed: map[string]any{
			"declared": map[string]any{"valuesCompared": declared.ValuesCompared, "diffs": len(declared.Diffs), "listOrderSame": declared.ListOrderSame, "uniqueProbes": declared.UniqueProbes, "sqliteVersion": declared.SQLiteVersion},
			"naive":    map[string]any{"valuesCompared": naive.ValuesCompared, "diffs": naive.Diffs},
			"postgresGoTypes":  types,
			"postgresNullExternalIds": pgNullExternal,
		},
		Passed: passed,
	})
	c.addStep(StepRecord{
		Step: 10, Name: "第二の製品(SQLite)での実現と値の往復", ModelRevision: c.final,
		Assets: []AssetUse{{Asset: "(SQLiteの提供元)", Called: false, Reason: "TEIにSQLiteの提供元がない。go.modにSQLiteのドライバーを足さず(依存の追加は禁止)、Python標準のsqlite3を別プロセスで使う。方言の対応はサンプルの宣言(declarations.jsonのsqliteDialect)に置く"}},
		Output: map[string]any{
			"boundaryRegistrations": boundary, "declared": declared, "naive": naive,
			"dialectPlacement": c.Decl.SQLite.Placement, "knownDifferences": c.Decl.SQLite.KnownDifferences,
			"heterogeneousVerified": []string{
				"同じ最終版の保存写像(列、主キー、一意制約、索引)を、PostgreSQL(TEIの提供元の描画とサンプルの一意制約の描画)とSQLite(サンプルの宣言した型の対応)の二つの製品に実現した",
				"PostgreSQLで読んだ値(pgxを通した生の値)を、宣言した表現でSQLiteのファイルへ書き、SQLite自身が読み返した値と、意味の値(整数、文字列、十進の文字列、UTCの瞬間)として全件比べた",
				"SQLiteの一意制約が重複を拒否し、NULLの重複を許すこと(PostgreSQLの既定のNULLS DISTINCTと同じ)",
				"UTCの固定幅の文字列で並べたSQLiteの一覧の順序が、PostgreSQLのtimestamptzの順序(時刻が同じなら受注番号の昇順に揃えた)と一致すること",
			},
			"heterogeneousNotVerified": []string{
				"TEIのSQLiteの提供元(カタログの観測、schemadiffでの比較、移行の描画と適用)はない。SQLite側の構造は宣言から作っただけで、観測して期待状態と比べていない",
				"TEIのクエリ実行器でSQLiteへ一覧と明細の問合せを実行していない(一覧の順序はサンプルが書いたSQLiteのSQLで確かめた)",
				"Goのドライバーを通したSQLiteへの書込み、SQLiteでの同時実行と書込みの待ち、照合順序の差(P-DB-04)、SQLiteでの算術と十進の並べ替え",
				"PostgreSQLとSQLiteの間の移行、二つの製品の同時運用や同期",
			},
		},
	})
	_ = persistence.StorageInt64
	c.gap("プラグイン", "共通層と方言層の分離が設計案にとどまり、SQLiteの提供元と方言の宣言の場所(物理型の対応の面)がない", "サンプルの宣言(sqliteDialect)とPython標準のsqlite3")
	return nil
}
