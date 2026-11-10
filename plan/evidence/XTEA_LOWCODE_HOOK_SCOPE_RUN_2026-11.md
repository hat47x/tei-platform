# フック範囲の実行時適用の確認（2026-11）

## 対象

- 実装: `internal/hook/run.go`・`internal/hook/sequence.go`・`internal/hook/reach.go`の範囲の扱い
- 試験: `internal/hook/scope_run_test.go`
- 対象版: `main` の `177024840` から分岐した`feat/hook-scope-run-20261110`
- 環境: Windows、Go `1.27.0`

## 見つけた不足

前ラウンドで`hook`宣言へ`scope`を足し、構文・検査・投影まで通しました。しかし**Runnerは`scope`をまったく読んでいませんでした**（`internal/hook`全体で`Scope`の参照が0件）。宣言は検査を通るのに、実行時には何の効果もない状態です。

これはX-TEAの**D-9**と同じ形の欠陥です。D-9は「定義に残るが実行されない属性」（`XF110`の`RangeKey`がコメントアウトされ実行時に効かない）で、知識目録は「定義にある値と実行時の意味が一致しない。利用者は設定した効果を期待する」と評価しています。範囲を宣言できるのに効かないなら、同じ問題を作り直すことになります。

## 変更

3つの入口すべてで、**適用できない範囲は拒否または理由つきの保留**にしました。

| 入口 | 扱い |
| --- | --- |
| `Runner.Run` | `HOOK_SCOPE_UNAPPLIED`で拒否。フックは実行しない |
| `Runner.RunSequence` | 同じコードで拒否。後続の位相へ進まない |
| `Runner.RunReachable` | その位相を`Skipped`に入れ、理由に範囲名を書く（実行済みと報告しない） |

適用できる範囲は`record`と空（Pattern自身の単位）だけです。これらの入口は**レコード1件につきフック1回**しか実行しないため、`row`・`item`・`header`を適用したふりをしません。

## 観測

- `TestRunRefusesAnUnappliedScope` 成功。`row`・`item`・`header`のいずれも拒否され、実装は**呼ばれない**（`ran`が空）。
- `TestRunAppliesTheRecordScope` 成功。`record`と範囲なしは実行される（拒否が狭いことの確認）。
- `TestRunSequenceRefusesAnUnappliedScope` 成功。`RunSequence`も同じ規則で拒否し、フックを実行しない。
- `TestRunReachableSkipsAnUnappliedScopeWithAReason` 成功。`before-update`が`Skipped`に入り、理由に`scope row`が含まれ、実行されない。
- `internal/hook`・`internal/validation`・`internal/syntax`・`internal/applicationview` は`ok`。
- 例の`validate`は`ok`（既存警告のみ）。例は`scope record`なので実行経路でも通る。

## 読み取れること

- 宣言された範囲が実行時に無視される経路が塞がれた。3つの入口が同じ規則を使う。
- 適用できない範囲は「実行した」と報告されない。`RunReachable`は`Skipped`と理由を残す。
- 範囲を足したことが、**宣言と実行の一致**という形で検査できる。

## 読み取れないこと

- **`row`・`item`・`header`を実際に適用すること**（行ごとに繰り返す、項目ごとに判定する等）。今は拒否します。適用するには、範囲ごとの繰り返しをRunnerへ入れる設計が要ります。
- `interaction`の契約をrecord以外のPatternへ広げること（前ラウンドの訂正を参照）。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。範囲が実行時に無視されないことは参照実装で確認済み、`record`以外の範囲の適用は未実装です。
