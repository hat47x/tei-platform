# 行範囲（scope row）の適用の確認（2026-11）

## 対象

- 実装: `internal/hook/rows.go`（`Runner.RunRows`・`RowRefusal`・`narrowTo`）、`internal/hook/run.go`・`sequence.go`・`reach.go`の`row`の扱い
- 試験: `internal/hook/rows_test.go`、`internal/hook/scope_run_test.go`
- 対象版: `main` の `a9b549435` から分岐した`feat/hook-row-scope-20261110`
- 環境: Windows、Go `1.27.0`

## 前ラウンドの残し

前ラウンドは「適用できない範囲は拒否する」で止めました。`row`・`item`・`header`はいずれも拒否され、**宣言はできるが実行できない**状態でした。D-9（定義にあるが効かない）を再生産しないための暫定でしたが、要求は範囲（行、レコード、項目）を宣言として持つことを求めており、`row`を実行できることが要ります。

## 変更

`Runner.RunRows`を追加し、`row`範囲を**行ごとに1回**適用します。

| 項目 | 内容 |
| --- | --- |
| 呼出し | `RunRows(ctx, invocation, rows)`。行の数だけフックを呼ぶ |
| 値の絞り込み | 各行を宣言の`reads`へ絞る（宣言外の属性は実装へ渡さない） |
| 停止 | 最初の拒否で止め、**行番号を名指す**（`RowRefusal`）。以降の行は訪れない |
| 業務結末 | `RowRefusal`は`OutcomeOf`で結末名を返す（`errors.As`で取り出せる） |
| 空の結果 | 拒否しない（0件のページはエラーではない） |
| `row`以外 | `RowRows`は`HOOK_SCOPE_NOT_ROW`で拒否 |

`Run`と`RunSequence`は`row`を`HOOK_SCOPE_NEEDS_ROWS`で**`RunRows`へ案内**します。`RunReachable`は`row`を`Skipped`に入れ、理由に「row setが必要、`RunRows`を使う」と書きます。`item`・`header`は引き続き`HOOK_SCOPE_UNAPPLIED`です。

## 観測

- `TestRunRowsAppliesTheHookToEachRow` 成功。3行で3回呼ばれ、各行が**自分の値**（10・20・30）を見る。
- `TestRunRowsNarrowsToDeclaredReads` 成功。宣言外の`secret`は実装へ渡らない。
- `TestRunRowsStopsAtTheRefusedRow` 成功。2行目で拒否すると呼出しは2回で止まり、`RowRefusal.Row`が`1`（0起点）、結末名`value-rejected`が残る。
- `TestRunRowsRefusesANonRowScope` 成功。範囲なし・`record`・`header`は`RunRows`で拒否され、実装は呼ばれない。
- `TestRunRowsAcceptsAnEmptyResult` 成功。0件はエラーにならない。
- `internal/hook`・`internal/validation`・`internal/syntax`は`ok`。

## 読み取れること

- `row`範囲が**宣言どおりに繰り返される**。X-TEAのスクリプトは1回の呼出しの中で利用者がループを書いていたため、宣言は繰り返しを見ませんでした。
- 拒否が**行を名指す**ので、ページ全体を再実行せずに該当行を特定できる。
- 値の絞り込みが行ごとにも効く（宣言外の属性は渡らない）。
- `row`を扱えない入口は、黙って1回実行せず`RunRows`へ案内する。

## 読み取れないこと

- **`item`・`header`の適用**は未実装です（引き続き拒否）。
- CLI/Studioの参照実行が`RunRows`を呼ぶ配線。今は境界の実装だけで、呼出し側は`Run`/`RunSequence`のままです。したがって**例のWorkspaceで`row`範囲を宣言すると、参照実行では`HOOK_SCOPE_NEEDS_ROWS`で止まります**。
- 行の識別（どの行か）を属性で示すこと。今は0起点の位置だけです。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`row`範囲の繰り返し適用は参照実装の境界で確認済み、呼出し側の配線と`item`・`header`の適用は未実装です。
