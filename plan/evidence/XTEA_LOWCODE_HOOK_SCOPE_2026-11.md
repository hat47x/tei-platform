# フック宣言の範囲（scope）の確認（2026-11）

## 対象

- 実装: `internal/application/model.go`の`Hook.Scope`、`internal/syntax/application_pattern.go`の`scope`、`internal/validation/hook_scope.go`と`application_pattern.go`の検査、`internal/applicationview`とCLI/Studioの投影
- 試験: `internal/syntax/application_pattern_hook_test.go`、`internal/validation/hook_scope_test.go`、`cmd/tei/xtea_example_test.go`
- 対象版: `main` の `a9d98e5fe` から分岐した`feat/hook-scope-declaration-20261110`
- 環境: Windows、Go `1.27.0`

## 見つけた不足

RQ-C2の要求文は、フック宣言が**契機・範囲（行、レコード、項目）**・読み取るデータ・使用する能力・生成する結果・失敗の種類を持つと定めています。実装には**範囲がありませんでした**（`phase`・`reads`・`uses`・`produces`・`failure`のみ）。

X-TEAのイベントスクリプトにも範囲の宣言が無く、行単位で判定するフックとレコード単位で判定するフックは、スクリプトを読まないと区別できませんでした。

## 変更

- `application.Hook.Scope`を追加（`row`・`record`・`item`・`header`）。空は「Pattern自身の単位」。
- `.tei`の`hook`ブロックに`scope <word>`を追加。語彙外は構文で拒否（`scope must be row, record, item, or header`）。
- 検査を追加。
  - `APPLICATION_PATTERN_INTERACTION_HOOK_SCOPE_UNSUPPORTED`:語彙外。
  - `APPLICATION_PATTERN_INTERACTION_HOOK_SCOPE_MISMATCH`:Pattern種別が見ない単位（例:record Patternに`scope row`）。
- 投影:`view`のテキストに`scope`、Studioの「フック」表に「範囲」列、JSONの`hooks[].scope`。
- 例の`order-record.tei`へ`scope record`を追加。
- 試験を追加（構文1件、検査4件、例1件更新）。

## 種別と範囲の対応

| Pattern種別 | 許す範囲 |
| --- | --- |
| `browse` | `row`・`record` |
| `record` | `record` |
| `header-detail` | `header`・`row` |
| `command` | `item`・`record` |
| `reference` | `record` |

知らない種別は絞りません（新しい種別を、誰も決めていない段階で使えなくしないため）。

## 観測

- `TestParseRecordApplicationPatternHookScope` 成功。`scope record`が読まれ、`scope column`は拒否される。
- `TestHookScopeFitsPattern` 成功。上表の全組合せ。
- `TestHookScopeAcceptsAMatchingUnit` 成功。`record`と範囲なしは診断なし。
- `TestHookScopeRefusesAMismatch` 成功。record Patternの`scope header`は`SCOPE_MISMATCH`。
- `TestHookScopeRefusesAnUnsupportedWord` 成功。`column`は`SCOPE_UNSUPPORTED`。
- 例の`view`は`- before-update scope record reads ...`を出す。
- `validate`は`examples/xtea-lowcode`・`reference/workspace`とも`ok`（既存警告のみ）。

## 試験を書く過程で分かったこと

最初は**browse Pattern**へフックを付けて検査しようとしましたが、診断が出ませんでした。フィクスチャをrecord Patternへ変えて確認しました。

**その理由を最初は「record Patternの経路でしか検査が走らない」と書きましたが、誤りでした。** 実際には、browse などの record 以外の Pattern は `interaction` 自体を持てません（`APPLICATION_PATTERN_INTERACTION_NOT_ALLOWED`:「the first interaction contract is limited to record patterns」）。フックは `interaction` の中にあるので、record 以外では**宣言そのものが拒否**されます。

つまり検査の非対称ではなく、**対話の契約が今は record Pattern に限られている**という明示的な設計です。フックが検査されない record 以外の Pattern は存在しません。

## 読み取れないこと

- **browse・header-detail・command Pattern は `interaction` を持てません**（`APPLICATION_PATTERN_INTERACTION_NOT_ALLOWED`）。フックは`interaction`の中にあるため、これらでは宣言そのものが拒否されます。つまりフックが検査をすり抜ける経路はありません。**範囲を他のPattern種別でも使えるようにするには、まず`interaction`の契約を広げる判断が要ります。**
- 範囲が実行時に効くこと（行ごとに判定を繰り返す等）。今は宣言と検査だけです。
- `interaction` の契約を record 以外へ広げること（範囲をそれらの種別で使う前提）。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。範囲の宣言・検査・投影は参照実装で確認済み、実行時の適用は`Run`（レコード単位）と`RunRows`（行単位）に在りますが、`RunRows`をホストから呼ぶ経路は未実装です（2026-11-10の訂正を参照）。

## 追記（2026-11-10）: 実行時の適用範囲と移行計画への反映

上の記録は宣言・検証・投影を扱っています。その後の実測で次を確かめ、移行計画のフック表へ1行足しました。

### 実行時の拒否

`internal/hook/run.go`は、実行境界が適用できない範囲を**宣言があるのに何もしない**状態にせず拒否します。

```text
HOOK_SCOPE_UNAPPLIED: hook <phase> declares scope <scope> but this runner applies one hook per record
```

X-TEAの欠陥（知識ノートD-9「定義は在るが実行時に何もしない」）を避けるための検査です。

### まだ適用されない範囲

現行の実行境界は「1レコードに1フック」です。したがって:

| 宣言 | 実行 |
| --- | --- |
| 空（Pattern自身の単位） | 適用される |
| `record` | 適用される |
| `row` | 適用される。ただし `Run` ではなく `RunRows` を使う（`Run` は `HOOK_SCOPE_NEEDS_ROWS` で `RunRows` を指す） |
| `item`・`header` | `HOOK_SCOPE_UNAPPLIED` で拒否される（この実行境界は適用しない） |

設計のX-TEA対応表は「一覧では行ごと、単票では1件」と述べており、`row`の適用は設計上求められています。参照実装はそこまで届いていません。

### 移行計画への反映

`plan/workstreams/XTEA_LOWCODE_TRANSFER_PLAN_2026-11.md`のフック表に、この宣言面の行が無かったため追加しました（`Scope`の記述はWP4の散文と危険表にしか無く、実装対応表には無かった）。

### 読み取れないこと

- `item`（項目単位）と`header`（見出し明細の明細行）の適用。`HOOK_SCOPE_UNAPPLIED`で拒否されます。
- **`RunRows` をホストから呼ぶ経路**。`internal/hook/rows.go` に実装と試験（5件）はありますが、`cmd`・`internal`のどちらからも呼ばれていません。CLI・Studioの参照実行は `RunReachable` を使い、`row` は理由つきで skip されます。
- 実Pluginが範囲をどう解釈するか。

### 訂正（2026-11-10）

この追記の初版は「行ごとの適用は未実装」と書きましたが、**誤りでした**。`internal/hook/rows.go`の`RunRows`が行ごとの適用を実装しており、5件の試験で固定されています。

- 各行の値を、宣言した`reads`だけに絞る（`TestRunRowsNarrowsToDeclaredReads`）
- 拒否した行を名指し、それ以降を訪問しない（`TestRunRowsStopsAtTheRefusedRow`）
- `row`以外の範囲を拒否する（`TestRunRowsRefusesANonRowScope`）
- 空の結果を失敗にしない（`TestRunRowsAcceptsAnEmptyResult`）

正しい未実装は「**`RunRows`をホストから呼ぶ経路**」です。実装は在りますが、CLI・Studioの参照実行は`RunReachable`を使うため、`row`のフックは`HOOK_SCOPE_NEEDS_ROWS`で`RunRows`を指され、`RunReachable`では理由つきでskipされます。
