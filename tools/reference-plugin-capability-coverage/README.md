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
| `database-instance` | する（技術名で解決） | 指すのは提供技術名（`postgres`）。能力IDとは文字列で比べず、`Capability.SupportsDatabaseProvider` で解決する（既存の validation と同じ問い） |
| `lease-policy`・`grant`・`external` | しない | 選択ではなく、資源・権限・外部要件 |

`database-provider` も、instance の提供技術に対応する能力が選ばれたものとして扱います。技術名と能力IDを文字列で比べないため、`postgres` と `database.postgres` の食い違いは起きません。

## 出力

- `capabilities[]`: 宣言ごとの `status`（`selected`／`unselected`）と、選んだ `environments[]`。
- `unselectedCount`: 選ばれていない宣言の数。
- `notes[]`: この一覧が示さないこと。

## 扱っている範囲

- 宣言であり、実装の存在・適合・信頼・権限ではありません。
- `selected` は「環境がその能力IDを名指しした」ことだけを示します。
- 読み取り専用です。`-workspace` の外へは書きません。

## 未実装

- `database-provider` の解決が、instance と能力の対応を**宣言**から引くこと（現行は能力側の `databaseProviders` を引いており、instance 側の宣言ではありません）。
- Bundle経由の間接選択を、選んだ主体ごとに分けること。
- 実Provider・実Host・権限との照合。

実行記録は [`plan/evidence/PLUGIN_CAPABILITY_COVERAGE_REFERENCE_2026-11-08`](../../plan/evidence/PLUGIN_CAPABILITY_COVERAGE_REFERENCE_2026-11-08/README.md) にあります。
