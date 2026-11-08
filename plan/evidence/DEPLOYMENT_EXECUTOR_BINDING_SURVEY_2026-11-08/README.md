# 配備実行アダプタは、誰がどう組み立てているか（2026-11-08）

- 対象: [`internal/plugin/runtime/adapters.go`](../../../internal/plugin/runtime/adapters.go)の `DeploymentExecutor` と、[ピン留めしない3経路は、何のためにRuntimeを直接呼ぶのか](../RUNTIME_UNPINNED_PATH_PURPOSES_2026-11-08/README.md)
- 位置づけ: **静的な走査の記録**。世代を固定すべきかは示しません。
- 計測した版: `raw/head.txt`（`origin/main` `e6450c26` の作業ツリー）。

前の記録は、`InvokeWithProvenance` を直接呼ぶ3経路のうち `adapters.go` だけが**製品パッケージの中**にあることを示しました。この記録は、**その `DeploymentExecutor` を誰がどう組み立てているか**を見ます。

## 1. 観測

```text
== counts ==
sites naming DeploymentExecutor: 19
  non-test: 4
  test: 15
non-test sites also using Coordinator: 0
```

`DeploymentExecutor` を名指す19箇所のうち、**非試験は4件**です。

| 非試験の組み立て箇所 | `Plan` | `CapabilityID` |
| --- | --- | --- |
| `examples/process-plugin/main.go` | ローカルで構築 | 変数 `capabilityID` |
| `tools/reference-deployment-attempt/main.go` | ローカルで構築 | `migration.atlas`（直書き） |
| `tools/reference-plugin-build-supply/stage4.go` | ローカルで構築 | `process.reference`（定数） |
| `tools/reference-plugin-self-build/main.go` | `runtimePlan` | `runtime.native_reference`（直書き） |

**4件とも `composition.Coordinator` を使っていません。** また、CapabilityID は**4件とも直書きかローカル変数**で、`tei validate` が解決した能力から来ていません。

## 2. この観測が示すこと

- **`DeploymentExecutor` を組み立てる非試験の箇所は4件あり、いずれも参照実装（`tools/`・`examples/`）です。** 製品の CLI（`cmd/tei`）はこの型を組み立てていません。
- **4件とも Coordinator を使わない**ため、`servicehost` と同じ Host世代のピン留めを受けません。
- **CapabilityID が直書き**である点が共通しています。`Environment` の `deployment-executor` Binding（`supportsAction` で解決される）を通っていません。
- つまり `adapters.go` は製品パッケージの中にありますが、**それを組み立てるのは参照実装だけ**で、宣言から解決した能力と結びつける経路はまだありません。

## 3. この記録が示さないこと

- **世代を固定すべきか**。配備実行が Host世代のピン留めを必要とするかは、この記録では判断していません。
- **4件が同じ安全性であること**。`Registry` の中身（Sandbox・Verifier・Resources）は箇所ごとに違い、この記録では整理していません。
- **直書きの CapabilityID が誤りであること**。参照実装が固定の能力を試すために直書きするのは自然です。
- **実行時の挙動**。静的な走査のみです。

## 4. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（型を名指す箇所を試験／非試験に分け、組み立て内容と Coordinator の有無を見る） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0行） |
| `raw/sites.txt` | 19箇所の一覧（試験／非試験の別つき） |
| `raw/construction.txt` | 非試験4件の組み立て内容 |
| `raw/coordinator.txt` | 各組み立て箇所が `Coordinator` を使うか |
| `raw/summary.txt` | 上の表を作る元の数 |

## 5. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `032cf6530d91c0e9bfadd0dfd876411f406dd3d3fd72a306890b4dc3167986cc` | 2783 |
| `raw/head.txt` | `cf7298a2e0473263d8acd7703a0267431767520b4119059fb41ef62b9f47b833` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |
| `raw/sites.txt` | `b51d1fb9db7f33b3146d57d924b83d7029de46fb00d6bd492ff1208a70335f2c` | 1226 |
| `raw/construction.txt` | `129016a5eed04f8b44ba42a4921914edf1f48d16445c96d68ef171641987e87e` | 1979 |
| `raw/coordinator.txt` | `59715af082e21ff62301ca388348165d7c02125a901ef5bc05c579d6f6c5e5db` | 254 |
| `raw/summary.txt` | `275697d8b411eabc883110c18fa3f6ed662b1bb1fb4e48989acbfa4ab4b438e7` | 115 |

`raw/sites.txt` は `pruntime.DeploymentExecutor{` または `runtime.DeploymentExecutor{` を名指すファイルの一覧です。**型の使用箇所ではなく、複合リテラルの記述箇所**を数えているため、メソッド定義や型宣言は含みません。
