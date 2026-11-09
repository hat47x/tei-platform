# 構成検証の選択規則は、Bindingの形ごとに揃っているか（2026-11-08）

- 対象: [`docs/plugins.md` の「利用する機能の解決」](../../../docs/plugins.md)と、[`internal/validation/validation.go`](../../../internal/validation/validation.go)の選択の診断
- 位置づけ: **静的走査の記録**。規則の良し悪しや変更の提案ではありません。
- 計測した版: `raw/head.txt`（`origin/main` `141c377ed` の作業ツリー）。

`docs/plugins.md` は「機能の提供実装を明示的に選び、候補が見つからない場合や複数ある場合は推測で選ばずエラーとする」と書いています。この記録は、**その規則が Binding の形ごとに同じ強さで実装されているか**を、実ソースの走査で確かめます。

## 1. 観測

### 1.1 6つの Binding 形式すべてに、選択の診断がある

| Binding | 対応なし | 複数 | 能力が非対応 |
| --- | --- | --- | --- |
| 配備実行 | `DEPLOYMENT_EXECUTOR_PLUGIN_MISSING_OR_AMBIGUOUS` | 同左（1つの診断で両方） | `DEPLOYMENT_EXECUTOR_CAPABILITY_UNSUPPORTED` |
| 状態観測 | `OBSERVATION_PROVIDER_PLUGIN_MISSING_OR_AMBIGUOUS` | 同左 | `OBSERVATION_PROVIDER_CAPABILITY_UNSUPPORTED` |
| 検証根拠 | `EVIDENCE_PROVIDER_PLUGIN_MISSING_OR_AMBIGUOUS` | 同左 | `EVIDENCE_PROVIDER_CAPABILITY_UNSUPPORTED` |
| データベース instance | `DATABASE_PROVIDER_PLUGIN_MISSING` | `DATABASE_PROVIDER_PLUGIN_AMBIGUOUS` | （能力側で判定） |
| 移行 | `MIGRATION_PROVIDER_PLUGIN_MISSING` | `MIGRATION_PROVIDER_PLUGIN_AMBIGUOUS` | （能力側で判定） |
| 環境の機能 | `ENVIRONMENT_FEATURE_PLUGIN_MISSING` | `ENVIRONMENT_FEATURE_PLUGIN_AMBIGUOUS` | （能力側で判定） |

**6形式すべてに、対応なしと複数の診断があります。** `docs/plugins.md` の記述と一致します。

### 1.2 環境の機能だけ、対象が1つに限られている

```go
// internal/validation/validation.go:908
if resolveErr == nil && feature == environment.FeatureSQLAnalysis {
```

機能の選択を検査する分岐は、**`editor.sql_analysis` のときだけ**通ります。`internal/environment/model.go` の `Known()` は `editor.sql_analysis` と `editor.database_workbench` の2つを既知として受け付けますが、**`editor.database_workbench` は `Known()` の検査だけを受け、能力との突き合わせを受けません**。

他の5形式（配備実行・状態観測・検証根拠・データベース instance・移行）に、このような「対象を1つに限る」分岐はありません。

## 2. この観測が示すこと

- **`docs/plugins.md` の規則は、6形式のどれにも共通の形で存在します。** 記述と実装が食い違っているわけではありません。
- ただし**環境の機能だけ、規則の適用範囲が1つの機能に限られています**。`editor.database_workbench` を有効にした環境は、対応する能力が無くても診断を受けません。これは「推測で選ばない」のではなく「**選ばれていないことを検査しない**」状態です。
- 前の記録（[環境のBindingを、能力の対応表で一括して解決する](../PLUGIN_SELECTION_MATRIX_REFERENCE_2026-11-08/README.md)）で観測した `missing` 2件は、この走査と一致します。

## 3. この記録が示さないこと

- **`editor.database_workbench` の検査を足すべきか**。今は機能が既知であるだけで、能力の対応を要求しない設計なのかもしれません。この記録は差の存在だけを示します。
- **規則を変えること**。診断の権限は `internal/validation` が持ちます。
- **実行時の挙動**。走査はソースの文字列と分岐の位置によるもので、実際に診断が出るかは試していません。
- **`docs/plugins.md` の記述の訂正**。記述は6形式すべてに当てはまり、誤っていません。

## 4. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（診断名の抽出・形式ごとの検査の有無・機能の分岐・既知の機能） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0行） |
| `raw/rule-scan.txt` | 走査結果 |

## 5. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `5c5515ecba339a033da531f1d4949cc4cb801454ebb5c15b7445af0bda885cfc` | 1410 |
| `raw/head.txt` | `3b3ed2d9478214effa83b7206772bde83506ee746d3d60aef03d9c9f04b437cf` | 73 |
| `raw/pre-status.txt` | `7c30c4e2993484aa097396c96bcfc24b7d8bc7fea003d9cd07becd8b3b793a52` | 12 |
| `raw/rule-scan.txt` | `1308946ed41d772647a58461fb81047e1e74145825f7378b7b18938bf2595ebc` | 2341 |

`raw/pre-status.txt` は12バイトで、`aw/` の行だけを含みます（走査中にスクリプトを一時ディレクトリへ置いたため）。走査は grep による文字列探索で、Goの型解決をしていません。`feature == environment.FeatureSQLAnalysis` 以外の経路で機能が検査されていないことは、この走査だけでは否定できません。
