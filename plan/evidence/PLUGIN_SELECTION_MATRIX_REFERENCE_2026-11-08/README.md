# 環境のBindingを、能力の対応表で一括して解決する（2026-11-08）

- 対象: [`docs/plugins.md` の「利用する機能の解決」](../../../docs/plugins.md)と、[`internal/validation`](../../../internal/validation/validation.go)の選択の診断
- 位置づけ: **参照実装と実行記録**。診断の権限は `internal/validation` が持ち、この参照は一覧を出すだけです。
- 計測した版: `raw/head.txt`（`origin/main` `39d4debae` の作業ツリー）。

現行の構成検証は、Bindingごとに「対応する能力があるか」を診断します。ただし**診断は1件ずつ**で、環境の選択面を一望する経路はありません。この記録は、6種類のBindingを同じ規則で一括して解決する参照実装を作り、参照Workspaceで観測したものです。

## 1. 何を作ったか

`tools/reference-plugin-selection-matrix`（新規・読み取り専用）。

| Binding | 突き合わせる能力 |
| --- | --- |
| 配備実行（action） | `CapabilityDeploymentExecutor` + `SupportsAction` |
| 状態観測（observation） | `CapabilityObservationProvider` + `SupportsObservation` |
| 検証根拠（evidence） | `CapabilityEvidenceProvider` + `SupportsEvidence` |
| データベース instance（technology） | `CapabilityDatabaseProvider` + `SupportsDatabaseProvider` |
| 環境の機能（feature） | `CapabilityEditorIntelligence` + `SupportsFeature` |
| 移行（tool） | `CapabilityMigrationProvider` + `SupportsMigrationTool` |

**既存の `Supports*` をそのまま使い、新しい対応表を作りません。** 判定は `selected`／`missing`／`ambiguous` の3値で、`missing` と `ambiguous` は既存の診断と同じ語彙です。`ambiguous` は**解消せず、両方を列挙**します。

## 2. 観測

`reference/workspace` の結果です。

```text
environments: ['development', 'production']
selections: 24  missing: 2  ambiguous: 0
```

- **24件のBindingのうち22件が `selected`**、**2件が `missing`**、`ambiguous` は0件。
- `missing` の2件は、どちらも `editor.database_workbench`（development と production の1件ずつ）です。
- `editor.sql_analysis` は `tei.editor.intelligence.sql` の `editor.sqlglot` で `selected` です。**同じプラグインの能力が、機能によって対応する／しない**ことが見えます。

## 3. 重要な限界（この記録で観測したこと）

**`editor.database_workbench` の `missing` は、製品の診断ではありません。**

`internal/validation/validation.go` は、環境の機能のうち **`editor.sql_analysis` だけ**を能力と突き合わせます（`if ... feature == environment.FeatureSQLAnalysis` の分岐）。`editor.database_workbench` は `Known()` の検査だけを受けており、対応する能力が無くても診断を出しません。

つまりこの `missing` は、**現行の検証がまだ見ていない差**を先に見つけたものです。参照実装は意図的に `SupportsFeature` を全機能に当てているため、検証より広く見ています。レポートの `notes` と規則の指紋にこの非対称を明記しました。

## 4. この観測が示すこと

- **選択の面は、既存の `Supports*` だけで一括して解決できる。** 新しい対応表も、Bindingごとの特別扱いも要りませんでした。
- **検証が見ている範囲と、能力が宣言している範囲には差がある。** `editor.database_workbench` がその例で、能力側の対応宣言が無いまま環境が機能を有効にしています。
- **`ambiguous` は0件**でした。参照Workspaceでは、同じBindingを複数の能力が支える状態は起きていません。

## 5. この記録が示さないこと

- **`editor.database_workbench` が欠陥であること。** 検証が意図的に見ていない可能性があり、この記録は差の存在だけを示します。
- **検証の規則を変えるべきこと。** 診断の権限は `internal/validation` が持ちます。
- **Provider の適合・Artifact の信頼・Host の権限。** 選択は宣言の突き合わせであり、実装の存在や適合の証明ではありません。
- **実DB・実Provider。** 合成の Workspace と読み取りのみです。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順 |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（新規ツールと`aw/`のみ） |
| `raw/matrix.json` | 24件のBindingと判定 |
| `raw/summary.txt` | 上の表を作る元の一覧 |

## 7. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `23d717c5c31b6a81a7c38d7322a1e403bca77b89aaeca3956d9eb58122da1c15` | 1420 |
| `raw/head.txt` | `96da2902eaedc0df147e482c2b467a2d63591302f1da32d5a23cf96b2b46960e` | 73 |
| `raw/pre-status.txt` | `bb097bf23fb73bfcd21d49394550b8c64bec63ff8038efa69f8e078e80a4b26b` | 239 |
| `raw/matrix.json` | `ace01a1bf21c22578f4c69aa4f512da7aa6a8c004d25535caf3228e3d5cd727a` | 8214 |
| `raw/summary.txt` | `2a5c0e9e136e121333d230e4a1ad5c5fc9f57652ef1db7da1da8af5218b5c97a` | 2856 |

規則の指紋は `sha256:02fcc5ab9d8745b471cdc0b3601c134439069c7b5c01baaccfc9b9e1283e8550` です。
