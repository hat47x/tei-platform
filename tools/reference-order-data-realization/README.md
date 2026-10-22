# reference-order-data-realization

技術検証用サンプル[`order-data-realization`](../../reference/order-data-realization/README.md)(サンプル3)の工程1〜10を、一つの通しの記録としてつなぐ実装です。設計の正本は[技術検証用サンプルの統合設計 4.3節](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md)です。

この文書は作業中です。調査の結論として、工程ごとに既存資産を呼べるかを先に記録します。

## 工程ごとの既存資産の呼出し(調査の結論)

| 工程 | 既存資産 | 呼べるか | 呼び方、または呼べない理由 |
| --- | --- | --- | --- |
| 1 | なし(足場なし) | - | サンプル側で、作業領域の複製へ属性を直接編集する。共通受理(`tools/reference-shared-acceptance`)の型付き操作に既存のDataへの属性追加がないため、受理は未接続 |
| 2 | `tools/reference-example-slide` | 呼べる | 取込み可能なパッケージ`exampleslide`。`CatalogFromWorld`、`Review`、`BindAnswers`、`ProbeBaseCandidateKeys`を変更せずに呼ぶ |
| 2 | `tools/reference-shared-acceptance` | 呼べる(別プロセス) | `package main`のため取り込めない。`go build`した実行ファイルの`revision`、`preview`、`review`、`accept`で`add-candidate-key`を複製へ受理する |
| 3 | `tools/reference-er-table-crossview` | 呼べる | 取込み可能なパッケージ`ercrossview`の`Project`。一意制約の宣言は共通受理に操作がないため、保存写像への直接編集で行い、受理は未接続 |
| 4 | `tools/reference-standard-function-candidates` | 呼べる(別プロセス) | `package main`のため取り込めない。`go build`した実行ファイルを`-workspace`と`-application order_entry`で呼ぶ |
| 5 | `tools/reference-postgres-integration`、`tools/reference-postgres-migration-rendering` | 呼べない | どちらも`package main`で、表を手組みし、無条件の`DROP SCHEMA ... CASCADE`を持つ。作業領域を受け取らない。同じ内部部品(`realization.RealizeDesired`、`postgres.Provider`、`schemadiff.Compare`、`migration.Plan`)を直接使う |
| 5 | `tools/reference-ddl-entry` | 呼べる | 取込み可能なパッケージ`ddlentry`の`Render`で、変更前の版R0から旧構造の表を作る |
| 6・9 | `tools/reference-data-first-persistence` | 呼べない | `package main`で、受付依頼(`examples/first-workspace`)と手書きの宣言に固定されている。同じ設計(駆動部の`ResolvePattern`、`ResolveCommandBinding`、`InvokeCapability`、`ExecuteObservationRead`)を受注へ当てる最小の実装をサンプル側に置く |
| 7 | `tools/reference-scenario-state-diff` | 呼べる | 取込み可能なパッケージ`scenariodiff`の`Plan`で、移行の前後の行を比べる |
| 8 | `tools/reference-postgres-header-detail-runtime` | 呼べない | `package main`で、手組みの表と固定の行を作り、判定段階Bが既定の動作と出力を固定する。同じ内部の入口(`ExecuteHeaderDetailContractQueryInEnvironment`)を、一覧の結果を受けて呼ぶ |
| 10 | なし(足場なし) | - | SQLiteの提供元がない。Python標準の`sqlite3`を別プロセスで使い、方言の対応はサンプルの宣言に置く |
