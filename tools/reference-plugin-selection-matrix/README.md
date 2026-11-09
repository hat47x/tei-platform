# reference-plugin-selection-matrix

環境が宣言した**Binding**ごとに、どのプラグイン能力が対応するか、対応が無いか、複数あるかを一覧する、読み取り専用の参照です。`internal/validation` が持つ診断の権限はそのままに、選択面を一望するためだけに使います。

~~~text
go run ./tools/reference-plugin-selection-matrix/cmd/tei-plugin-selection-matrix \
  -workspace reference/workspace
~~~

## 突き合わせ

既存の `plugin.Capability` の `Supports*` をそのまま使います。新しい対応表は作りません。

| Binding | 能力 | 判定に使う関数 |
| --- | --- | --- |
| 配備実行 | `deployment-executor` | `SupportsAction` |
| 状態観測 | `observation-provider` | `SupportsObservation` |
| 検証根拠 | `evidence-provider` | `SupportsEvidence` |
| データベース instance | `database-provider` | `SupportsDatabaseProvider` |
| 環境の機能 | `editor-intelligence` | `SupportsFeature` |
| 移行 | `migration-provider` | `SupportsMigrationTool` |

## 判定

| 判定 | 意味 |
| --- | --- |
| `selected` | 対応する宣言が1件 |
| `missing` | 対応する宣言が0件 |
| `ambiguous` | 対応する宣言が2件以上。**解消せず、両方を列挙**します |

## この参照が示さないこと

- 診断の権限。拒否するのは `internal/validation` です。
- Provider の適合・Artifact の信頼・Host の権限。選択は宣言の突き合わせです。
- **`editor.database_workbench` の `missing` は製品の診断ではありません。** 現行の検証は環境の機能のうち `editor.sql_analysis` だけを能力と突き合わせるため、この差は参照実装が検証より広く見ている結果です（レポートの `notes` と規則の指紋に明記）。

実行記録は [`plan/evidence/PLUGIN_SELECTION_MATRIX_REFERENCE_2026-11-08`](../../plan/evidence/PLUGIN_SELECTION_MATRIX_REFERENCE_2026-11-08/README.md) にあります。
