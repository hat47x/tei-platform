# `internal/servicehost` に非試験の呼出し元が無いのはなぜか（2026-11-08）

- 対象: [構成Kernel移行Gate](../../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)の「`internal/servicehost` は非試験のファイルからどこも import されていない」という記録
- 位置づけ: **静的走査の記録**。接続の是非や実装の変更提案ではありません。
- 計測した版: `raw/head.txt`（`origin/main` `045346a3` の作業ツリー）。

前の記録（[構成Kernel移行Gateの依存の形を、いまのmainで確かめる](../COMPOSITION_GATE_DEPENDENCY_RECHECK_2026-11-08/README.md)）は、`internal/servicehost` の非試験 importer が0件であることを確かめました。この記録は、**なぜ0件なのか**を、必要な部品の組み立て箇所から見ます。

## 1. `servicehost` が要求する3つの入力

`NewSelectedServiceCarrierInvoker` は3つを要求します（[`service_carrier.go`](../../../internal/servicehost/service_carrier.go)）。

```go
func NewSelectedServiceCarrierInvoker(
	coordinator *composition.Coordinator,
	runtime     pruntime.Registry,
	routes      ServiceCarrierRouteSource,
) (*SelectedServiceCarrierInvoker, error)
```

そのうち `routes` は `BindHostApplicationEnvironmentRoute`（[`application_route_source.go`](../../../internal/servicehost/application_route_source.go)）で作ります。**`servicehost` 自身が、Host所有の2つの入力（Coordinator・Route）を組み立てる関数を提供しています。**

## 2. 観測

各コンストラクタの呼出し箇所を、試験と非試験に分けました（`constructors.txt`）。

| コンストラクタ | 非試験の呼出し元 | 試験の呼出し元 |
| --- | --- | --- |
| `NewSelectedServiceCarrierInvoker` | **0**（定義ファイルを除く） | **5** |
| `BindHostApplicationEnvironmentRoute` | 0（定義ファイルのみ） | 3 |
| `composition.NewSelectedCoordinator` | **1**（`tools/reference-project-advice-lit/contract_bridge.go`） | 21 |
| `composition.NewRegistry()` | **3** | 27 |

`NewSelectedServiceCarrierInvoker` を呼ぶ試験は次の5件です。

```text
internal/driver/reference_current_host_service_route_test.go
internal/servicehost/application_route_source_test.go
internal/servicehost/service_carrier_test.go
reference/web-composition/go-host/canonical_generation_service_handoff_test.go
reference/web-composition/go-host/two_application_host_service_carrier_test.go
```

## 3. この観測が示すこと

- **部品は揃っています。** Coordinator を作る非試験の呼出し元が1件（`contract_bridge.go`）あり、Registry を作る非試験の呼出し元が3件あります。
- **しかし、そのうち `servicehost` の Invoker を組み立てるものは0件**です。`contract_bridge.go` は Coordinator を `Acquire()` まで使いますが、`servicehost` は通りません。
- したがって「非試験の呼出し元が0件」は、**部品が無いからではなく、`servicehost` を組み立てる段が参照実装の試験に留まっている**ためです。試験では Host所有の入力を作って Invoker を組み立てられますが、製品の経路ではまだ誰も組み立てていません。
- Gateの「Cの試験成功を実HostからのService Carrier呼出しと読み替えない」という留保は、この組み立て段の不在と対応します。

## 4. この記録が示さないこと

- **`servicehost` を接続すべきか**。Gateの判断です。
- **`contract_bridge.go` が `servicehost` を使うべきか**。この参照は Coordinator の `Acquire()` だけを使う設計で、Service Carrier を通す必要があるかは別の問いです。
- **試験5件が同じ内容であること**。件数だけを数えており、各試験が何を確かめているかは整理していません。
- **実行時の挙動**。静的な呼出し箇所の走査のみです。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（コンストラクタの呼出し箇所を試験／非試験に分ける） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0行） |
| `raw/constructors.txt` | 呼出し箇所の一覧（試験／非試験の別つき） |
| `raw/summary.txt` | 上の表を作る元の数 |

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `33bfce19f6f84da3bb8c72edfb92d013ff887e65dd88e086cb46d5fa4ac4aa71` | 3438 |
| `raw/head.txt` | `3ebd84a2478983c55f721dc12c028e7ff451d6bda5c69596c087cc474888b214` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |
| `raw/constructors.txt` | `cf1ceb4be2644f06cb3077ed9b389f65969038f7ec2015b5a35950d63de603a8` | 5762 |
| `raw/summary.txt` | `37ae29d6d003591b058a1b0924a64eb22c61837511016f07e36a2c334567124d` | 226 |

`raw/constructors.txt` は**呼出し箇所のファイル一覧**で、行数や出現回数ではありません。`NewSelectedCoordinator` を名指すファイルには定義ファイル（`internal/composition/bootstrap.go`）も含まれますが、非試験の「呼出し元」としては数えていません。

`raw/summary.txt` の `non-test call sites of NewSelectedServiceCarrierInvoker` の行は、初回の実行で `grep -c` と `wc -l` の併用により `0` が二重に出ていたため、`wc -l` に統一して再実行しました。最終的な値は0で変わりません。
