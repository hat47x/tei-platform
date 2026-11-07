# reference-plugin-capability-coverage

プラグインが `provides` で宣言した能力のうち、**どの環境の Binding からも選ばれていないもの**を一覧する、読み取り専用の参照です。選ばれていないことは欠陥の証明ではありません。

~~~text
go run ./tools/reference-plugin-capability-coverage/cmd/tei-plugin-capability-coverage \
  -workspace reference/workspace
~~~

Workspaceは変更しません。Provider解決・実装選択・権限発行はしません。

## 数える選択

| Binding | 数える | 理由 |
| --- | --- | --- |
| `deployment-executor` | する | 能力IDをそのまま指す |
| `observation-provider` | する | 同上 |
| `evidence-provider` | する | 同上 |
| `database-instance` | **しない** | 指すのは提供技術名（`postgres`）で、能力ID（`database.postgres`）ではない |
| `lease-policy`・`grant`・`external` | しない | 選択ではなく、資源・権限・外部要件 |

`database-provider` の能力が常に `unselected` に見えるのは、この数え方の結果です。レポートの `notes` にも明記しています。

## 出力

- `capabilities[]`: 宣言ごとの `status`（`selected`／`unselected`）と、選んだ `environments[]`。
- `unselectedCount`: 選ばれていない宣言の数。
- `notes[]`: この一覧が示さないこと。

## 扱っている範囲

- 宣言であり、実装の存在・適合・信頼・権限ではありません。
- `selected` は「環境がその能力IDを名指しした」ことだけを示します。
- 読み取り専用です。`-workspace` の外へは書きません。

## 未実装

- `database-provider` の選択を、instance と能力IDの対応を宣言から解決して数えること（現行モデルにその対応がありません）。
- Bundle経由の間接選択を、選んだ主体ごとに分けること。
- 実Provider・実Host・権限との照合。

実行記録は [`plan/evidence/PLUGIN_CAPABILITY_COVERAGE_REFERENCE_2026-11-08`](../../plan/evidence/PLUGIN_CAPABILITY_COVERAGE_REFERENCE_2026-11-08/README.md) にあります。
