# 環境の配備実行 Binding は、どこまで使われるか（2026-11-08）

- 対象: [`internal/deployment/routing/executor.go`](../../../internal/deployment/routing/executor.go)と [`cmd/tei/main.go`](../../../cmd/tei/main.go)、[配備実行アダプタは、誰がどう組み立てているか](../DEPLOYMENT_EXECUTOR_BINDING_SURVEY_2026-11-08/README.md)
- 位置づけ: **静的な走査の記録**。どこで捨てられるべきかは示しません。
- 計測した版: `raw/head.txt`（`origin/main` `0c73b7f5` の作業ツリー）。

前の記録は、参照実装が `DeploymentExecutor` を**直書きの CapabilityID** で組み立てることを示しました。この記録は、**環境の `deployment-executor` Binding が、どこで読まれ、どこで実行に使われるか**を追います。

## 1. 観測

### 1.1 Binding を読む箇所

```text
internal/deployment/routing/executor.go:26   環境の Binding を実際に照合する（RouteCommand）
internal/environment/model.go:110            型の宣言
internal/index/index.go:452                  索引への登録
internal/syntax/parser.go:526                構文からの取り込み
internal/validation/validation.go:996        構成の検証
tools/reference-plugin-capability-coverage   参照実装（能力の選択）
tools/reference-plugin-selection-matrix      参照実装（選択の対応表）
```

**実行の経路として Binding を読むのは `routing.RouteCommand` だけ**です。ほかは宣言・索引・検証・参照実装です。

### 1.2 Binding から実行までの2つの経路

`routing.Router.Dispatch` は Binding を解決して `executor.Execute` まで呼びます（`executor.go:105`）。しかし **`routing.Router` を組み立てる非試験の箇所は0件**です（`executor_test.go:34` のみ）。

一方、CLI は `RouteCommands` を呼びますが（`cmd/tei/main.go:984`）、その結果は `deploymentProviderPlan{Executors: routed}` に入り、**JSON として書き出すだけ**です。CLI は `executor.Execute` を呼びません。

```text
== CLI deployment subcommands that mention routing or executor ==
31:  deploymentrouting "tei/internal/deployment/routing"
52:  Executors     []deploymentrouting.RoutedCommand `json:"executors,omitempty"`
883: var result executor.Result                     ← 外部から受け取る（実行はしない）
903: deploymentrouting.ValidateResult(env, cmd, result)  ← 受け取った結果を検証
984: deploymentrouting.RouteCommands(env, ...)      ← 経路を計画へ載せる
```

### 1.3 実行する箇所

`executor.Execute` を呼ぶ非試験の箇所のうち、配備実行に関係するのは3件です。

| 箇所 | 起動方法 |
| --- | --- |
| `tools/reference-deployment-attempt/main.go:132` | `adapter.Execute`（直書きの `CapabilityID`） |
| `tools/reference-plugin-build-supply/stage4.go:131` | `adapter.Execute`（固定の能力） |
| `tools/reference-plugin-self-build/main.go:769` | `nativeAdapter.Execute`（固定の能力） |

**3件とも参照実装で、環境の Binding から解決していません。**

## 2. この観測が示すこと

- **環境の `deployment-executor` Binding は、検証と計画（JSON）まで使われ、実行には使われていません。** `RouteCommands` の結果は「どの executor が担当するか」の計画で、実際に動かすのは別の経路です。
- **Binding を実行へ繋ぐ `routing.Router` は、非試験では組み立てられていません。** 試験では組み立てられています（`executor_test.go`）。
- **実行する3件は、Binding を通らず固定の能力を直叩き**します。前の記録の「CapabilityID が直書き」と一致します。
- つまり **`servicehost` と同じ形の断絶**が、配備実行の側にもあります。検証・計画は宣言から作られ、実行は別の入口から入ります。

## 3. この記録が示さないこと

- **この断絶が問題であること**。検証と実行が分かれているのは、PoCの段階で意図された形かもしれません。
- **`routing.Router` を組み立てるべきか**。どこで組み立てるかは設計判断です。
- **CLI が実行しないことの是非**。CLI が `deploy` を実行しないのは、外部の実行主体に委ねる設計かもしれません。
- **3件の参照実装が同じ安全性であること**。
- **実行時の挙動**。静的な走査のみです。

## 4. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（Binding の読取り・経路の呼出し・実行箇所・CLI の言及） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0行） |
| `raw/readers.txt` | `DeploymentExecutors` を読む非試験の箇所 |
| `raw/routing-callers.txt` | `RouteCommand(s)` の非試験の呼出し元 |
| `raw/execute-sites.txt` | `Execute` の非試験の呼出し元 |
| `raw/cli.txt` | CLI の該当行 |

## 5. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `9e41c897b9694a0dd4c19f0a770bc2df2d736d534a10848bf827668ffaf4260f` | 2318 |
| `raw/head.txt` | `85a39827a5a9c562890ecf2a410fb4db1591f8501bbd4c7b9d8e8129f5504011` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |
| `raw/readers.txt` | `09e0cd944b98987d2017b7b12c06d4788347cf105ad19c9e01160f767dca9f94` | 737 |
| `raw/routing-callers.txt` | `213561e57c63be60e70fb5853a7badc299b93ecdea3a714dd5881bf47e5fea69` | 685 |
| `raw/execute-sites.txt` | `85d030eaa17ff25ffcf5c641e870283651dac16d957ed2527c092a5f058d4121` | 2224 |
| `raw/cli.txt` | `a202b29c11697be43821067f9fd46ccb71298305364114040814f731a7d74834` | 413 |

`raw/execute-sites.txt` は `.Execute(` を広く拾うため、HTMLテンプレートの `Execute` など**配備と無関係な行も含みます**。上の表の3件は、その中から配備実行に関係するものを選んだものです。この選別は機械的ではなく、私が読み取って行いました。
