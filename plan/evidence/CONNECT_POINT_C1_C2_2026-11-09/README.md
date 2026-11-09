# C-1 と C-2 は、同じ問いの2案か（2026-11-09）

- 対象: [判断材料: 実行の側をどこで宣言へ繋ぐか](../../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)の **C-1**・**C-2**、および[宣言のBindingから実行までを、参照実装で繋いでみる](../BINDING_TO_EXECUTION_REFERENCE_2026-11-09/README.md)
- 位置づけ: **静的な読み取りの記録**。どちらを選ぶかは示しません。
- 計測した版: `raw/head.txt`（`origin/main` `0b20495b` の作業ツリー）。

判断材料は C-1（`routing.Router` を組み立てる）と C-2（`servicehost` を入れる）を「実行の入口を製品へ引き込む」2案として並べ、**両方を同時に選ぶと2つの経路ができる**と書きました。この記録は、**その2つが本当に同じ問いの選択肢なのか**を、入口の形から確かめます。

## 1. 観測

### 1.1 2つの入口が受け取るもの

| | C-1 `Router.Dispatch` | C-2 `SelectedServiceCarrierInvoker.Invoke` |
| --- | --- | --- |
| 入力 | `executor.Command`（action id・kind・database/application） | `requestID` + `payload`（`json.RawMessage`） |
| 環境 | `environment.Environment`（Binding を読む） | なし（Application/Environment は構築時に固定） |
| 実行先 | `executor.Executor`（`ID`・`Supports`・`Execute`） | `pruntime.EnvironmentPlan` + `ServiceProviderSelection` |
| 世代 | **なし**（Binding が executor ID を決めるだけ） | **あり**（`Coordinator.Acquire` で Host世代をピン留め） |

**C-1 は「配備の動作」を実行し、C-2 は「Service の要求」を運びます。** 入力の型が `executor.Command` と `json.RawMessage` で異なり、C-1 には世代の概念がありません。

### 1.2 どちらが既に到達可能か

```text
== non-test sites naming each connect point ==
C-1 routing.Router: 1
C-2 NewSelectedServiceCarrierInvoker: 1
```

- C-1 の1件は **`tools/reference-binding-to-execution`**（前のラウンドで作った参照）。
- C-2 の1件は **`internal/servicehost/service_carrier.go`**（定義そのもの）。

つまり **C-1 は非試験の呼出し元が1件できましたが、C-2 はまだ0件**（定義ファイルのみ）です。

## 2. この観測が示すこと

- **C-1 と C-2 は、同じ問いの代替ではありません。** 実行する対象が違います（配備の動作 vs Service の要求）。判断材料が「両方を同時に選ぶと2つの経路ができる」と書いたのは、**同じ対象を取り合う場合の話**で、この2つには当てはまりません。
- **両方を選ぶことは、経路の重複ではありません。** 配備の実行と Service の要求は、別の仕事です。
- **C-2 はまだ誰も組み立てていません**（C-1 は前のラウンドで組み立て済み）。C-2 を試すなら、C-1 と同じ形の参照が必要です。
- C-2 は世代のピン留めを伴うため、**C-1 より要求が多い**（Coordinator・Runtime・RouteSource・Plan・Selection）。

## 3. この記録が示さないこと

- **どちらを先に製品へ入れるべきか**。判断材料の順序（C-4 → C-1/C-2 → C-3）は、この記録では変えていません。
- **C-2 が組み立て可能であること**。定義はありますが、非試験の呼出し元が0件なので、C-1 と同じ形で動くかは確かめていません。
- **判断材料の誤り**。判断材料は「同じ対象を取り合う2案」として書いており、この記録は対象が違うことを補足するものです。
- **実行時の挙動**。静的な読み取りのみです。

## 4. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（2つの入口の形と、非試験の呼出し元を数える） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0行） |
| `raw/signatures.txt` | 2つの入口のシグネチャ |
| `raw/hands-to.txt` | それぞれが渡す型（Executor SPI と Runtime carrier） |
| `raw/inputs.txt` | それぞれが必要とする入力の一覧 |
| `raw/c1-sites.txt`・`raw/c2-sites.txt` | 非試験で名指すファイル |
| `raw/summary.txt` | 上の表を作る元の数 |

## 5. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `fd12162c4e6a2b021887f5067fe5ac99d626fcf48e1fa189a498a94447f07e26` | 3796 |
| `raw/head.txt` | `dc04f84e9df09e7fc0cc11a3306ed0d0358e6d0f065b799dc9a770fcc00e77a6` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |
| `raw/signatures.txt` | `c4d2d643b6f2be34194ee2e896ace52eafcdea1f5226b561d789f0bca9d19c3f` | 1231 |
| `raw/hands-to.txt` | `d68115193f70bda271ff694342edcedf139228829911e0e9e96d252b92eb998a` | 549 |
| `raw/inputs.txt` | `42da4a6c61399692b4039b2c632253c31ecf8d4a378aea22157b6bb454f1bc30` | 588 |
| `raw/c1-sites.txt` | `6d7c69c2df9e96252079e3ddfeea922b10d8646d4f4df665f5c63306ac84142a` | 57 |
| `raw/c2-sites.txt` | `79e0d7ecc281a7484ce7cb34f5fcdf84bfc80f41f4b24b139cb9f7daa2e4ba19` | 40 |
| `raw/summary.txt` | `5edc9d95abf1e27c9e9051445b6fdb5d007c907c91a0f6fa4ec1a463c5c77b11` | 109 |

`raw/inputs.txt` の C-2 の入力一覧は、**`NewSelectedServiceCarrierInvoker` の引数と `InvokeSelectedServiceCarrier` の引数から私が書き起こしたもの**で、コードから機械的に抽出したものではありません。
