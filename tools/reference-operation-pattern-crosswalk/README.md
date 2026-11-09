# reference-operation-pattern-crosswalk

受理済みの意味から、**宣言されたPattern**と**導出された標準操作**を突き合わせる、読み取り専用の参照です。[操作面の設計候補](../../plan/design/OPERATION_SURFACE_DESIGN_2026-11.md)の「追記（2026-11-07）」が書いたW3の対応表を、動く形で試すためのものです。

W3は**未着手**のままです。この参照は案を試すものであり、案を採用済み・実装済みとは扱いません。

~~~text
go run ./tools/reference-operation-pattern-crosswalk/cmd/tei-operation-pattern-crosswalk \
  -workspace reference/workspace
~~~

Workspaceは変更しません。識別子・操作・受理を書き戻しません。

## 突き合わせる三つの宣言

名前の類似・prefix・ロード順は使いません。次の三つだけです。

| Pattern Kind | 対応する操作 | 突き合わせる欄 |
| --- | --- | --- |
| `browse` | `list` | `data` = `target` |
| `record` | `read` | `data` = `target` |
| `command` | `create` / `update` | `capability` の効果 = `binding`（効果を宣言している場合だけ） |
| `reference` | `read` | `relation` のtarget Data = `target` |
| `header-detail` | `list` と `read` | `relation` のtarget Data = `target` |

導出された操作は、既存の`operations.DeriveOperationSurface`をそのまま使います。二つ目の導出規則は作りません。

## 出力

- `links`: 一致した組。`via`が`data`・`capability`・`relation`のどれで決まったかを持ちます。
- `unlinked`: 一致しなかった宣言。`operation-surface`と同じgap語彙（`missing-input`・`missing-semantics`・`conflict`・`out-of-scope`）で理由を返します。既定値で埋めません。
- `conflicts`: 二つの宣言が同じ操作を取ったとき。どちらかを選びません。
- `ambiguous`: `header-detail`のように複数の操作種別へ対応するKind。全一致を並べ、選ばないことを明示します。

## 扱っている範囲

- 宣言であり、実行権限ではありません。`links`は「この意味からはそう読める」ことだけを示します。
- 効果を宣言しない`command`は`out-of-scope`です。一致を作るために効果を補いません。
- 読み取り専用です。`-workspace`の外へは書きません。

## 未実装

- 実DB・実Provider・権限・実Consumerとの照合。
- `header-detail`を`list`と`read`のどちらか一方に決める規則（#1127/#1124の担当）。
- 実運用のWorkspaceでの網羅。現在は参照Workspaceとその候補でだけ確認しています。
