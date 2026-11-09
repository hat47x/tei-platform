# CON-12c は、いまの観測・証拠の宣言では問えない（2026-11-09）

- 対象: [判断材料の第12節 CON-12c](../../workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)、[観測の参照](../OBSERVATION_ROUTER_CONNECT_REFERENCE_2026-11-09/README.md)、[証拠の参照](../EVIDENCE_ROUTER_CONNECT_REFERENCE_2026-11-09/README.md)
- 位置づけ: **読み取り専用の調査**。何も変更しません。
- 計測した版: `raw/head.txt`（`origin/main` `6fef2539`）。

判断材料の CON-12c は「同じ Provider ID でも Artifact digest・Host/Grant epoch・Source Scope が変わったら、旧 Binding を再利用しない」を求めます。この記録は、**その問いが前提にしている項が、観測・証拠の宣言に存在するか**を確かめます。

## 1. 4つの層の宣言を並べる

```text
== generation.Binding（C-1/C-2が土台にする）==
   ContractID, ContractVersion, ProviderID, ArtifactDigest, Generation

== environment.ObservationProviderBinding（観測の参照が使う）==
   Kind, Provider

== environment.EvidenceProviderBinding（証拠の参照が使う）==
   Kind, Provider

== runtime.ServiceProviderSelection（Service carrier が使う）==
   ContractID, ContractVersion, ContractFingerprint, Scope,
   Provider, ImplementationID, ArtifactDigest
```

### 識別子の数を数える

```text
== identity fields per binding ==
   generation-binding:  2     (ArtifactDigest, Generation)
   observation-binding: 0
   evidence-binding:    0
   service-selection:   2     (Scope, ArtifactDigest)
```

## 2. この観測が示すこと

- **観測と証拠の宣言は、`Kind` と `Provider` しか持ちません。** digest も epoch も Scope もありません。
- **したがって CON-12c は、この2つの層では「まだ実行していない」のではなく、問い自体が立ちません。** 比較する項が宣言に無いためです。
- **識別子を持つのは `generation.Binding` と `ServiceProviderSelection` の2層だけ**です。CON-12c を実際に検証できるのは、この2層を使う経路（C-1・C-2）に限られます。
- **これは「観測・証拠が識別子を持つべき」という主張ではありません。** 観測と証拠は `Kind` で選ぶ設計であり、Artifact の同一性は別の層（carrier）が持つ、という現在の分担を示しただけです。

## 3. この記録が示さないこと

- **観測・証拠の宣言に digest/epoch/Scope を足すべきか**。判断材料の C-1〜C-4 は未決で、これは設計判断です。
- **C-1/C-2 で CON-12c が満たされていること**。識別子を持つ層を示しただけで、変更時の非再利用を試してはいません。
- 実 Provider の Artifact 信頼、現在の Grant、開示 Scope。
- CON-12d（部分成功）の評価。

## 4. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（4つの層の宣言を抜き出し、識別子の数を数える） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0バイト＝クリーン） |
| `raw/generation-binding.txt` ほか3件 | 層ごとの宣言 |
| `raw/identity-count.txt` | 層ごとの識別子の数 |

## 5. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `c053a02b22c28a8a9ef8ee9f5a333d2b800b9e394f9e8793e2287490157b7ea0` | 2462 |
| `raw/head.txt` | `bd6e6c872fb99d0ca5ca15c6ed772a854a6813084c2a87ffeec286435cb0b295` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |
| `raw/generation-binding.txt` | `bdb6975d28d5cf2eba0bef2661a9f6e17295ecfd0eaa966443dc1176e14e2ba1` | 243 |
| `raw/observation-binding.txt` | `6d4896c0254fa0384b2ee92d7847c9becdc1e6510e4643106de35622311af6aa` | 281 |
| `raw/evidence-binding.txt` | `4d3856ef01ee38c04bf03744cb428c6512e5b65af4c774bce32070065eb838f6` | 266 |
| `raw/service-selection.txt` | `aa3a0e285884495c7ad8f2e002bea0bbac1133e56c5c5657834967b4b02f08f8` | 605 |
| `raw/identity-count.txt` | `e35043bd3de3f6641f7f0faba22a9b61d14a118b15ec5f1e39379f7d2f121553` | 132 |
