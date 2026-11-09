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

最初は**browse Pattern**へフックを付けて検査しようとしましたが、診断が出ませんでした。フック宣言の検査は`validateRecordPatternInteraction`にあり、**record Patternの経路でしか走らない**ためです。フィクスチャをrecord Patternへ変えて確認しました。

これは「範囲の検査がrecord Patternにしか無い」という実装の現状でもあります（下記）。

## 読み取れないこと

- **browse・header-detail・command Patternのフック宣言は検査されません。** フックの宣言と検査は今のところrecord Patternの経路にだけあります。範囲を足したことで、この非対称が表に出ました。
- 範囲が実行時に効くこと（行ごとに判定を繰り返す等）。今は宣言と検査だけです。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。範囲の宣言・検査・投影は参照実装で確認済み、実行時の適用とrecord以外のPattern種別の検査は未実装です。
