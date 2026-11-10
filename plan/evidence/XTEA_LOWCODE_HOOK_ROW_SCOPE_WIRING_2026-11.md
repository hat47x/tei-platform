# 行範囲を到達可能な実行へ配線した確認（2026-11）

## 対象

- 実装: `internal/hook/run.go`の`Runner.Rows`、`internal/hook/reach.go`の行範囲の分岐、`internal/hook/reach_rows.go`（`applyReachableRows`）
- 試験: `internal/hook/reach_rows_test.go`
- 対象版: `main` の `e63d4c8f4` から分岐した`feat/hook-row-scope-wiring-20261110`
- 環境: Windows、Go `1.27.0`

## 前ラウンドの残し

前ラウンドは`Runner.RunRows`を足しましたが、CLI/Studioが使う`RunReachable`は`row`範囲を**`Skipped`に入れるだけ**でした。つまり境界の実装はあっても、実際の参照実行では行範囲が動きませんでした。

## 変更

- `Runner.Rows`を追加（行の集合）。空なら従来どおり「行集合が必要」と報告。
- `RunReachable`が行集合を持っている場合、`row`範囲を**行ごとに適用**する（`applyReachableRows`）。
  - 各行は宣言の`reads`だけを見る。行に無い値は**パターン水準の値で代用しない**（その値はどの行のものでもない）。
  - 全ての行が通ったときだけ`Executed`に入れる。
  - 拒否はその行で止め、`Refused`と`Reasons`に**行番号**を書く（`row 1: ...`）。部分的に適用されたページが「合格」と読まれない。
  - `on-failure warn`の行は`Warned`に記録し、ページは続ける。
- `Run`が`Runner.Rows`を持っている場合、`row`範囲を`RunRows`へ委譲する。
- **`row`範囲で`uses`も宣言するフックは拒否**（`HOOK_SCOPE_ROW_USES_UNSUPPORTED`）。宣言した呼出しを行ごとに1回ずつ行う経路がまだ無いため、ページ全体で1回だけ呼ぶと「1行分の約束を果たし、他の行の約束を破る」ことになります。fail-closedを選びました。

## 観測

- `TestRunReachableAppliesRowScopeToEveryRow` 成功。3行で3回呼ばれ、`Executed`に入り、`Skipped`は空。
- `TestRunReachableStopsAtTheRefusedRow` 成功。2行目が拒否すると呼出しは2回で止まり（3行目は実行されない）、`Refused`が立ち、`Outcome`が`value-rejected`、`Reasons`に`row 1`が入る。**拒否された位相は`Executed`に入らない**。
- `TestRunReachableWithoutRowsSkipsARowScope` 成功。行集合が無ければ呼出しは0回で、`Skipped`と「row set」の理由が残る。
- `TestRunReachableRefusesRowScopeWithUses` 成功。`uses`付きの行範囲は拒否され、呼出しは0回。
- `TestRunRoutesARowScopeToRunRows` 成功。`Run`が`Rows`を持つとき行ごとに委譲する。
- `internal/hook`・`internal/studiohost`・`internal/validation`は`ok`。

## 読み取れること

- 行範囲が**参照実行の経路でも**行ごとに適用される（`RunReachable`は CLI と Studio が共有する）。
- 部分適用が「合格」と読まれない。拒否は行番号つきで残る。
- 行集合を渡さない呼出しは、行範囲を1回だけ実行せず、理由つきで飛ばす。

## 読み取れないこと

- **CLI/Studio が実際に`Runner.Rows`を埋めていません。** 参照実行（`command.Result`）は行集合を返さないため、渡す行がありません。したがって**例のWorkspaceで`row`範囲を宣言すると、参照実行では引き続き「row set が必要」で飛びます**。埋めるには、参照実行が行集合を返す設計（一覧の結果をフック境界へ渡す）が必要です。
- `row`範囲と`uses`の組合せ（行ごとの能力呼出し）。
- `item`・`header`の適用。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`RunReachable`の行範囲適用は参照実装で確認済み、行集合を供給する呼出し側の配線と`uses`の組合せは未実装です。

## 後継（2026-11-10）

この記録にある「`uses`を伴う行単位フックを`HOOK_SCOPE_ROW_USES_UNSUPPORTED`で拒否する」「二つの経路で`uses`の扱いが違う」という状態は、その後解消しました。

- 行単位の`uses`を実行する経路（[確認記録](XTEA_LOWCODE_HOOK_ROW_USES_CALLS_2026-11.md)）を入れ、`RunRows`も`applyReachableRows`も`hook.RunRowCalls`を通します。
- 宣言した能力は**1行ごとにその行の値で呼ばれます**。
- `refuseRowScopeWithUses`は削除し、`HOOK_SCOPE_ROW_USES_UNSUPPORTED`は現在のコードに存在しません。

本記録の観察（当時は経路ごとに扱いが違ったこと）は履歴として有効ですが、**現在の宣言規則として読まないでください**。
