# Runtimeへの経路は `servicehost` だけではない（2026-11-08）

- 対象: [`internal/plugin/runtime`](../../../internal/plugin/runtime/service_carrier.go)と[`internal/servicehost`](../../../internal/servicehost/service_carrier.go)、[構成Kernel移行Gate](../../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)
- 位置づけ: **静的走査の記録**。どちらの経路を使うべきかは示しません。
- 計測した版: `raw/head.txt`（`origin/main` `070834044` の作業ツリー）。

前の記録（[`servicehost` に非試験の呼出し元が無いのはなぜか](../SERVICEHOST_CONNECTION_PATH_SURVEY_2026-11-08/README.md)）は、Invoker を組み立てる段が試験に留まっていることを示しました。この記録は、**Runtime へ到達する経路が `servicehost` 以外にもあるか**を見ます。

## 1. 観測

```text
== counts ==
non-test importers of plugin/runtime: 30
among those, also importing servicehost: 0
```

- **`plugin/runtime` を import する非試験ファイルは30件**あります（`examples/` を含む）。
- **そのうち `servicehost` も import するものは0件**です。30件すべてが `direct`（`servicehost` を介さない）です。

### 1.1 Runtime への2つの入口

`plugin/runtime` には、外部から呼ばれる入口が2つあります。

| 入口 | 定義 | 非試験の参照 |
| --- | --- | --- |
| `InvokeSelectedServiceCarrier` | `service_carrier.go:107` | **1件のみ**（`internal/servicehost/service_carrier.go:112`） |
| `InvokeWithProvenance` | `model.go:634` | **3件**（`adapters.go`、`reference-order-rollout-host-continuity/host.go:337`、`reference-plugin-capability-contract/runtime_authority_bridge.go:171`）ほか内部呼出し |

```
== non-test references to InvokeSelectedServiceCarrier ==
internal/plugin/runtime/service_carrier.go:98   (doc comment)
internal/plugin/runtime/service_carrier.go:107  (definition)
internal/servicehost/service_carrier.go:112     (the only caller)

== non-test references to InvokeWithProvenance ==
internal/plugin/runtime/adapters.go:35
internal/plugin/runtime/model.go:595
internal/plugin/runtime/model.go:634            (definition)
internal/plugin/runtime/service_carrier.go:193
tools/reference-order-rollout-host-continuity/host.go:337
tools/reference-plugin-capability-contract/runtime_authority_bridge.go:171
```

## 2. この観測が示すこと

- **Host世代の固定（`InvokeSelectedServiceCarrier`）を通る非試験の経路は、`servicehost` の1つだけ**です。そしてその `servicehost` を組み立てる非試験の呼出し元は0件です（前の記録）。
- **一方、`InvokeWithProvenance` を直接呼ぶ非試験の経路は3件あります**。これらは `servicehost` を経由しないため、**Host世代のピン留めを受けません**。
- つまり「Runtime に到達する経路が無い」のではなく、**ピン留めされた経路だけが未接続**で、ピン留めしない経路は参照実装で動いています。Gateの「Cの試験成功を実HostからのService Carrier呼出しと読み替えない」は、この区別と対応します。

## 3. この記録が示さないこと

- **どちらの経路を使うべきか**。`InvokeWithProvenance` を直接使う3件が、ピン留めを必要としない用途なのかは確かめていません。
- **`servicehost` を接続すべきか**。Gateの判断です。
- **30件の import の内訳**。`servicehost` を import しないことは確かめましたが、それぞれが Runtime の何を使うかは整理していません。
- **実行時の挙動**。静的な import と関数参照の走査のみです。

## 4. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（import と入口の参照を分けて数える） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0行） |
| `raw/importers.txt` | `plugin/runtime`・`servicehost` の非試験 importer 一覧 |
| `raw/paths.txt` | 30件それぞれが `servicehost` 経由か直接か |
| `raw/entrypoints.txt` | 2つの入口の非試験参照 |
| `raw/summary.txt` | 上の表を作る元の数 |

## 5. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `26a32c0ffe48616e2a58d72fb6ffd31ef2c04633f349546645260a08060ce672` | 3274 |
| `raw/head.txt` | `c5b746fae96e2cf7d4ed61a392055eb0ad3ba1e7da0aff0fa322c9d9b128b1f8` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |
| `raw/importers.txt` | `d137bbd7a9c9cc9aa9a656ae2f4fb59d5b76b1d2a805ab9471d5a208243a0b1b` | 1981 |
| `raw/paths.txt` | `b666c60f803dc13ae5b7668914df9498ebca1042f8b46b94c07237a3d7c9d49f` | 1906 |
| `raw/entrypoints.txt` | `f0f118c9bc8979f3c3f83f03a8c57b4e8c2a56bf4daaa165f23b613a0e506f1f` | 1538 |
| `raw/summary.txt` | `ab15364da71aae5aa6ac07283288ebcdb9830a8b553054feb56b2d4b425d7ade` | 97 |

`raw/paths.txt` の30行は、`plugin/runtime` を import する非試験ファイルの一覧です。`examples/` を含むため、製品の経路と参照実装が混ざっています。
