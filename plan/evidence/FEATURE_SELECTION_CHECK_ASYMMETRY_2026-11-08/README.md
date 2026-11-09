# 環境の機能は、すべて選択を検査されるか（2026-11-08）

- 対象: [`docs/plugins.md` の「利用する機能の解決」](../../../docs/plugins.md)と、[`internal/validation/validation.go`](../../../internal/validation/validation.go)の `ENVIRONMENT_FEATURE_PLUGIN_*`
- 位置づけ: **実行で確かめた記録**。検査を足すべきかは示しません。
- 計測した版: `raw/head.txt`（`origin/main` `fc49a70f` の作業ツリー）。

前の記録（[構成検証の選択規則は、Bindingの形ごとに揃っているか](../PLUGIN_SELECTION_RULE_SURVEY_2026-11-08/README.md)）は、ソースの走査で「環境の機能だけ検査対象が `editor.sql_analysis` に限られている」ことを示しました。この記録は、**それを実行で確かめます**。

## 1. 何をしたか

受理済みの `reference/workspace` を3つ複製し、**どの機能を有効にするかだけ**を変えました。どの複製でも、対応する能力を持つプラグイン（`tei.editor.intelligence.sql`）は**選択から外してあります**。

| 複製 | 有効な機能 | 能力の有無 |
| --- | --- | --- |
| `sql-only` | `editor.sql_analysis` のみ | 無し |
| `workbench-only` | `editor.database_workbench` のみ | 無し |
| `accepted` | 受理済みのまま（両方＋プラグイン有り） | 有り |

## 2. 観測

| 複製 | 終了コード | `ENVIRONMENT_FEATURE_PLUGIN` の数 |
| --- | --- | --- |
| `sql-only` | **1** | **1** |
| `workbench-only` | **0** | **0** |
| `accepted` | 0 | 0 |

`sql-only` の診断（全文）:

```text
error	ENVIRONMENT_FEATURE_PLUGIN_MISSING	environment development feature editor.sql_analysis has no resolved editor-intelligence plugin capability
```

**同じ条件（プラグインが無く、能力も無い）で、機能の名前だけを変えると、診断が出る場合と出ない場合に分かれます。**

## 3. この観測が示すこと

- 走査で見た非対称は、**実行でも同じ**でした。`editor.sql_analysis` は検査され、`editor.database_workbench` は検査されません。
- `accepted` の複製が 0 で終わることは、**受理済みの Workspace がこの差を隠している**ことを意味します。受理済みでは `editor.database_workbench` に対応する能力が無いまま、診断が出ていません。
- 前の記録（[環境のBindingを、能力の対応表で一括して解決する](../PLUGIN_SELECTION_MATRIX_REFERENCE_2026-11-08/README.md)）で参照実装が出した `missing` 2件は、**この差を先に見つけたもの**でした。

## 4. この記録が示さないこと

- **検査を足すべきか**。`editor.database_workbench` が既知であるだけで、能力の対応を要求しない設計なのかもしれません。
- **`editor.database_workbench` が実際に使えないこと**。環境が機能を有効にしても、能力が無ければ動かないという推測は、この記録では確かめていません。
- **`docs/plugins.md` の記述の訂正**。記述は「明示的に選ぶ」としており、機能ごとの差には触れていません。誤りとは言えません。
- **実DB・実Provider**。合成の Workspace と `tei validate` の読み取りのみです。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（3つの複製と `tei validate`） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/sql-only.out.txt`・`raw/workbench-only.out.txt`・`raw/accepted.out.txt` | 各複製の出力 |
| `raw/exit-codes.txt` | 終了コード |
| `raw/summary.txt` | 上の表を作る元の一覧 |

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `f3f9e26381e5b0d76a4bc0ca2e1f133b53406db8c1c6e8a54f7fb834aab05fa5` | 3113 |
| `raw/head.txt` | `b75d3c80430dbc6f7d196d5dbd35c2c5c50443cda807efaeff653cd1d33b4ab0` | 73 |
| `raw/pre-status.txt` | `ebc523c3321ce4aa09a4e31fb87406df39d3dc96e956bcd8699cda0710158868` | 31 |
| `raw/sql-only.out.txt` | `59bf59ccc88049e2c6f215a03639f53ead27905be5085e11559f1b5d796a00ba` | 649 |
| `raw/workbench-only.out.txt` | `2f495c06bc0281ebba5161dd931e25dea0966dc6f704d9a3c52cde7e7e9f22fa` | 505 |
| `raw/accepted.out.txt` | `2f495c06bc0281ebba5161dd931e25dea0966dc6f704d9a3c52cde7e7e9f22fa` | 505 |
| `raw/exit-codes.txt` | `342ce0ae3dcc352c30f8fa99d3e5c748fc189a0671b93df121f2d47bec27b564` | 54 |
| `raw/summary.txt` | `119ee9d12c97465e13cbc28b0a7faa39b3629b46f5887f93c2075894704fa353` | 397 |

`raw/workbench-only.out.txt` と `raw/accepted.out.txt` は**同じSHA-256**です。違うのは環境の宣言だけで、`tei validate` の出力は同一でした。`raw/sql-only.err.txt`（14バイト）は `exit status 1` の行だけを含みます。
