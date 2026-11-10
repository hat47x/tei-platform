# 行範囲の宣言を通す修正の確認（2026-11）

## 対象

- 実装: `internal/validation/hook_scope.go`の`scopeFitsPattern`
- 試験: `internal/validation/hook_scope_test.go`
- 対象版: `main` の `092202f36` から分岐した`fix/hook-row-scope-accepted-20261110`
- 環境: Windows、Go `1.27.0`

## 見つけた矛盾

`row`範囲を3ラウンドかけて実装しました（宣言・検査、実行時の拒否、`RunRows`、`RunReachable`への配線）。しかし**宣言の段階で通りませんでした**。

- `interaction`（フックを含む）を持てるのは**record Patternだけ**です（`APPLICATION_PATTERN_INTERACTION_NOT_ALLOWED`）。
- record Patternの範囲は`record`だけを許していました（`scopeFitsPattern`）。
- したがって`row`範囲は**どこにも宣言できず**、実装全体が到達不能な死んだコードでした。

試験で確かめました。

| 範囲 | record Patternでの診断 |
| --- | --- |
| （なし） | なし |
| `row` | `SCOPE_MISMATCH` |
| `record` | なし |
| `item` | `SCOPE_MISMATCH` |
| `header` | `SCOPE_MISMATCH` |

**`row`だけを実装して、`browse`・`header-detail`は`interaction`を持てない**という組み合わせを見落としていました。3ラウンドの作業が宣言段階で止まっていたことになります。

## 変更

`scopeFitsPattern`の`record`で`row`も許すようにしました。

- record Patternは1レコードを扱い、**関連する行の集合について行ごとに判定もできる**（`RunRows`が適用する）。
- ここで`row`を拒むと、`interaction`がrecord限定である以上、`row`はどこにも宣言できません。

## 観測

- `TestHookScopeFitsPattern` 成功（`record`+`row`を`true`へ更新）。
- `TestHookScopeAcceptsAMatchingUnit` 成功（`record`と`row`の両方で診断なし）。
- 例のPatternで`scope row`が`validate`を通ることを手元で確認（確認後は`scope record`へ戻しました。下記参照）。
- `internal/validation`・`internal/hook`・`internal/syntax`・`internal/applicationview` は`ok`。

## 例を`row`へ変えなかった理由

例のフックは`uses`も宣言しています。`row`+`uses`は`HOOK_SCOPE_ROW_USES_UNSUPPORTED`で実行時に拒否される（行ごとに能力を呼ぶ経路が未実装）ため、例を`row`にすると**参照実行が止まります**。実際に試して確認しました。

そのため例は`scope record`のままにしました。`row`を例で動かすには、`row`+`uses`の実行（行ごとの能力呼出し）が先に必要です。

## 読み取れないこと

- **`row`範囲の実運用**。宣言は通りますが、CLI/Studioは`Runner.Rows`を埋めておらず、`row`+`uses`は拒否されるため、現状の参照実行では`row`範囲は動きません。
- `browse`・`header-detail`の`interaction`（`row`を本来使いたい種別）。
- `item`・`header`の適用。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`row`範囲が宣言できることは確認済み、実行経路の配線と`browse`系の`interaction`は未実装です。

## 後継（2026-11-10）

この記録にある「`uses`を伴う行単位フックを`HOOK_SCOPE_ROW_USES_UNSUPPORTED`で拒否する」「二つの経路で`uses`の扱いが違う」という状態は、その後解消しました。

- 行単位の`uses`を実行する経路（[確認記録](XTEA_LOWCODE_HOOK_ROW_USES_CALLS_2026-11.md)）を入れ、`RunRows`も`applyReachableRows`も`hook.RunRowCalls`を通します。
- 宣言した能力は**1行ごとにその行の値で呼ばれます**。
- `refuseRowScopeWithUses`は削除し、`HOOK_SCOPE_ROW_USES_UNSUPPORTED`は現在のコードに存在しません。

本記録の観察（当時は経路ごとに扱いが違ったこと）は履歴として有効ですが、**現在の宣言規則として読まないでください**。
