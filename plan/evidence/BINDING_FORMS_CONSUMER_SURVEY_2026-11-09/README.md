# 6つの宣言のうち、実行まで届いているのはどれか（2026-11-09）

- 対象: [C-3の参照](../EXECUTOR_ID_RESOLUTION_REFERENCE_2026-11-09/README.md)、[判断材料](../../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)、[配備実行のBinding調査](../DEPLOYMENT_EXECUTOR_BINDING_SURVEY_2026-11-08/README.md)
- 位置づけ: **読み取り専用の調査**。何も変更しません。
- 計測した版: `raw/head.txt`（`origin/main` `0837a6f4`）。

C-3 の参照は、**配備実行の能力IDを宣言から解決できる**ことを示しました。この調査は、**同じ形の宣言が他にも5つあり、そのうち2つは既に製品コードで解決されている**ことを記録します。

## 1. Environment が持つ6つの宣言

`internal/environment/model.go` の `Environment` は、実装を指す宣言を6種類持ちます。

| 宣言 | 何を指すか | 解決する関数 | 非試験で解決しているか |
| --- | --- | --- | --- |
| `DeploymentExecutors` | 動作を実行する能力 | `routing.RouteCommand` | **する**（`Router.Dispatch` 経由） |
| `ObservationProviders` | 状態を観測する能力 | `observation.Binding` | **する**（`cmd/tei/main.go:1043`） |
| `EvidenceProviders` | 証拠を集める能力 | `evidence.Binding` | **する**（`cmd/tei/main.go:1030`） |
| `Instances` | データベース実体 | （参照のみ） | 参照・検証のみ |
| `Bindings` | Application と実体の対応 | （参照のみ） | 参照・検証のみ |
| `ResourceBindings` | 資源の対応 | `resource.MatchBindings` | **する**（`plugin/runtime`） |

## 2. 観測

### 2.1 すべての宣言に非試験の読み手がいる

```text
== deployment executor ==
  internal/deployment/routing/executor.go:26   ← 解決
  internal/index/index.go:452                  ← 索引
  internal/syntax/parser.go:526                ← 解析
  internal/validation/validation.go:997        ← 検証
  tools/reference-plugin-capability-coverage/coverage.go:206
  tools/reference-plugin-selection-matrix/matrix.go:117

== observation provider ==
  internal/deployment/observation/provider.go:39   ← 解決
  internal/index/index.go:455 / syntax / validation
  tools/... 2件

== evidence provider ==
  internal/deployment/evidence/provider.go:32      ← 解決
  internal/index/index.go:458 / syntax / validation
  tools/... 2件
```

**6つとも、索引・解析・検証の3経路で読まれています。** これは「宣言が読まれていない」のではなく、**「解決の後に実行へ届くか」が宣言ごとに違う**ことを意味します。

### 2.2 observation と evidence は、C-3 と同じ形を製品コードで既に持つ

```go
func Binding(env environment.Environment, kind deployment.ObservationKind) (core.CanonicalID, error) {
    // 宣言から ID を選び、無ければ error、重複しても error
}
```

これは **C-3 の参照が `RouteCommand` で行ったことと同じ**です。しかも**製品の `cmd/tei/main.go` が実際に呼んでいます**（`-plan` の出力に provider の経路を載せる）。

### 2.3 しかし Router は、試験の中でしか組み立てられていない

```text
observation.Router{ / evidence.Router{ の構築:
  非試験: 0件
  試験:   internal/deployment/observation/provider_test.go
          internal/deployment/evidence/provider_test.go
```

**`Binding` は解決するが、`Router` を製品が組み立てないため、`Observe`／`Collect` は製品から呼ばれません。** これは C-1 で見た断絶と**同じ形**です — 解決は宣言から、実行は別の扉から。

## 3. この観測が示すこと

- **断絶は配備実行だけの話ではありません。** 6つの宣言のうち、**解決まで届いているのは3つ、実行まで届いているのは1つ**（`DeploymentExecutors`、しかも C-1 の参照経由）です。
- **observation と evidence は、C-3 が示した解決の形を既に製品に持っています。** C-3 は新しい仕組みではなく、**既にある形を配備実行にも適用できるか**を試したものでした。
- **「宣言がある」と「実行に届く」は別**です。`DeploymentExecutors` は索引・検証・解決の3経路すべてに現れますが、実行は参照実装からしか入りません。

## 4. この記録が示さないこと

- **どの宣言を実行へ繋ぐべきか**。判断材料の C-1〜C-4 は未決です。
- **observation/evidence の実行が正しいこと**。`Router` が試験でしか組み立てられていない事実を示しただけで、組み立てるべきかは評価していません。
- **`Instances` と `Bindings` に解決関数が要るか**。参照のみである事実を示しただけで、必要性は評価していません。
- 実装の変更（この記録は調査のみです）。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（6つの宣言ごとに非試験の読み手を列挙） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域 |
| `raw/deploymentexecutors.txt` ほか5件 | 宣言ごとの読み手の一覧 |

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `1a3e38625d9208531392c093e27febf852703542a0c67d4124f3d8a651534595` | 1935 |
| `raw/head.txt` | `40138ce13d755ddc70fb8373739ecb54de5a10c53c944fdd23c83ba58cbcb4e7` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |
| `raw/deploymentexecutors.txt` | `70ca1a7070315920124780d687fb47183f8bff2948cbd274ce699eeebb808b74` | 627 |
| `raw/observationproviders.txt` | `7853e4ccdf595d026ba8bc753bd3894717e55b1bdf3a3eda0959f5354550f805` | 739 |
| `raw/evidenceproviders.txt` | `b45b7e5c67576827a07ffa6ffc0371ed3499b30399f25d86a1bfb0d4d3c0eac3` | 703 |
| `raw/instances.txt` | `8b7be251ff50c98c0f0da48c40a97b2e7cde131485462d72894abdccf611ab4d` | 831 |
| `raw/bindings.txt` | `7db3ae2c7e8fed175d721947ff74a47dac747472ebd8a16feb19210d331fab0d` | 1484 |
| `raw/resourcebindings.txt` | `8bb95ec01052a0ccc72fdb38a23b0a2845fcb2fec6c8bd349b8fcf11fd509487` | 562 |

`raw/pre-status.txt` が0バイトなのは、計測前に作業領域が**クリーンだった**ことを示します（新規の記録ディレクトリを除く）。
