# 配備実行の能力IDを、宣言から解決する（C-3）（2026-11-09）

- 対象: [判断材料: 実行の側をどこで宣言へ繋ぐか](../../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)の **C-3**、[配備実行アダプタは、誰がどう組み立てているか](../DEPLOYMENT_EXECUTOR_BINDING_SURVEY_2026-11-08/README.md)
- 位置づけ: **参照実装と実行記録**。参照実装の直叩きを製品へ変えるべきかは示しません。
- 計測した版: `raw/head.txt`（`origin/main` `0950509e` の作業ツリー＋このPRの追加）。

前の記録は、`DeploymentExecutor` を組み立てる非試験の4件が**すべて参照実装**で、**`CapabilityID` が直書き**であることを示しました。この記録は、**その ID を環境の Binding から解決できるか**を試します。

## 1. 何を作ったか

`tools/reference-executor-id-resolution`（新規・読み取り専用）。既存の関数だけで3つの問いに答えます。

| 問い | 使う既存の関数 |
| --- | --- |
| この action で、Binding はどの executor を選ぶか | `routing.RouteCommand` |
| その ID は、lock から作った plan に存在するか | `pruntime.BuildPlan` → `EnvironmentPlan.Capability` |
| その能力は action を支えるか | `Capability.SupportsAction` |

**`hardCodedID` は比較のために並べるだけで、解決には使いません。**

## 2. 観測

```text
== resolved ==
  status: RESOLVED  admitted: True
  resolvedId: migration.atlas    hardCodedId: migration.atlas

== no-lock ==
  status: LOCK_REJECTED  admitted: False
  resolvedId: migration.atlas    hardCodedId: migration.atlas
  detail: unsupported plugin lock format 0

== other-bound-executor ==
  status: RESOLVED  admitted: True
  resolvedId: runtime.reference  hardCodedId: migration.atlas

== wrong-database ==
  status: BINDING_MISSING  admitted: False
  resolvedId: -                  hardCodedId: migration.atlas
  detail: environment development has no executor binding for action database-release ...
```

- **`resolved`**: Binding と実 lock が揃えば `RESOLVED`。解決した ID は直書きの値と一致します。
- **`no-lock`**: lock が無いと `LOCK_REJECTED`。**ID は解決できても、plan に載るかは別**だということが分かれます。
- **`other-bound-executor`**: **これが C-3 の要点**です。`application-retire` は `runtime.reference` に束縛されており、**直書きの `migration.atlas` ではなく `runtime.reference` が解決されます**。参照実装の直書きが Binding と食い違いうることを、実行で示しています。
- **`wrong-database`**: Binding が無い組み合わせは `BINDING_MISSING` で、ID を推測しません。

## 3. この観測が示すこと

- **C-3 は成立します。** 直書きの `CapabilityID` は、環境の Binding から解決できます。既存の3関数だけで足ります。
- **解決と admission は別の段**です。`routing.RouteCommand` は Binding から ID を決めますが、**その ID が lock の plan に載るかは `BuildPlan` が決めます**。`no-lock` がその分離を示します。
- **直書きは Binding と食い違いうる**ことが、`other-bound-executor` で具体化しました。同じ Workspace でも、action が違えば別の executor が正解です。

## 4. この記録が示さないこと

- **参照実装の直書きを製品へ変えるべきか**。判断材料の C-1〜C-4 は未決です。
- **`environment` の Binding が正しいこと**。`tei validate` の検証は別の経路で、この参照は検証を通しません。
- **実配備の成立**。この参照は実行しません（`executor.Execute` を呼びません）。
- **4件の参照実装すべての置き換え**。試したのは「解決できるか」だけで、各参照実装の `Registry` 構成は触っていません。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（4つの場合） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/resolved.json`・`raw/no-lock.json`・`raw/other-bound-executor.json`・`raw/wrong-database.json` | 4つの場合の出力 |
| `raw/summary.txt` | 上の表を作る元の一覧 |

`*.err.txt` はすべて0バイトです。

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `b5c480a5fb39e8aced0fc0becebf6869d213e33fc2d78125f4764f9d62cef535` | 3524 |
| `raw/head.txt` | `0374dc6f78ccd2e88ae515f2c53715c831e85ecff33a3eddd8a1c006366ad543` | 73 |
| `raw/pre-status.txt` | `5969de588175e1a047c4301f3ba3e5db6a5a9c0eb927c5e373549e801f52c5c4` | 304 |
| `raw/resolved.json` | `93fb43fb80f93a06a3b94ba2e8a19ddedf08509a97a0e294b4d4c058a6f20ba4` | 878 |
| `raw/no-lock.json` | `7e3e280afd6fba7bafc9adc73fb93d64011cb32dfff540945c3ef047e918cdbd` | 932 |
| `raw/other-bound-executor.json` | `d6f0627763113c31a5b348f179369ea2e50d262132c9da189a01205905231a86` | 882 |
| `raw/wrong-database.json` | `ed111ae37bc60a7188a839dcd5bb37f0698815ce8fbb230d9aa5ddbd084bdd19` | 975 |
| `raw/summary.txt` | `96bbde4e426ff9ef992bf7f64e429acfa032110010b7a57051b1522e6ea4acc7` | 690 |

規則の指紋は `sha256:513efb0c943afe65acd769270e648d76c00c7e3790cd3ad8363167c1178adec7` です。

`no-lock` の `detail` は「unsupported plugin lock format 0」で、**lock を渡さなかった結果**です。参照は lock を発明しません。
