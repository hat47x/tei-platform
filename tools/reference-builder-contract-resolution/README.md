# Builder Contract Resolution Reference

GitHub #1101 のread-only Referenceです。

TEIのBuild modelはすでにexact `BuilderIdentity`をBuildPlanへ記録できます。しかし、BuildPlan確定前に「どのBuilder Provider / physical artifactを使うか」を言語中立に解決する境界はまだありません。

このReferenceは、Go / Rust / JavaScript / Wasm等の特定toolchainを実行せず、**resolver semanticsだけ**を検証します。

## 責務分担

### #951

拡張可能Capability ContractのReferenceです。

本Referenceはその設計圧力をconsumer側から利用しますが、#951のReference codeをproduction primitiveとして扱いません。

固定`CapabilityKind`を追加せず、Builder Providerは`build-tool` Authority scopeであることを要求します。

### #1096

Go builder adapter内部を扱います。

対象:

- Go executable / toolchain identity
- Source closure
- hermetic environment
- isolated cache/state
- actual Go build
- reproducibility Evidence

本Referenceはそれらを実装しません。

### #1101

上位resolverを扱います。

```text
Build Intent
  -> Builder Contract
  -> compatible Provider
  -> Conformance Evidence
  -> explicit / unambiguous resolution
  -> exact BuilderIdentity
  -> BuildPlan
```

## Identity

次を分けます。

```text
Builder Contract
  = build serviceの意味

Builder Provider
  = logical implementation provider

ImplementationID
  = Provider内のimplementation identity

ArtifactDigest
  = exact physical implementation bytes

BuilderIdentity
  = BuildPlanへ渡すexact builder identity

Conformance Evidence
  = ProviderがContractへ適合したEvidence

Resolution
  = Intent + Contract + Provider + Evidenceのexact組合せ

Resolution Lock
  = 一度確定したResolutionのreplay identity
```

同じものへ潰しません。

## Build Intent

Reference Intentは次を明示します。

- Contract ID / version
- Recipe kind
- Target
- Profile

上位ContractへGo package path、Cargo manifest、npm script、WIT path等は入れません。

それらは各adapterが解釈するSource / Recipe / Parameterへ閉じます。

## Fresh resolution

fresh resolutionでは、compatibleかつconformantなProviderを列挙します。

条件:

- Authority scope = `build-tool`
- Contract ID / version一致
- Contract fingerprint一致
- Conformance profile一致
- Recipe kind対応
- Target対応
- Profile対応
- exact Provider artifactに対応するConformance Evidenceがちょうど一つ

### Providerが一つ

wiring無しでも選択できます。

### Providerが複数

列挙順、install順、PATH順では選びません。

曖昧としてfail-closedです。

### explicit wiring

logical Provider ID / version / ImplementationIDを指定できます。

同じlogical Provider / implementationに複数physical artifactがある場合、ArtifactDigestを省略したwiringは曖昧としてrejectします。

ArtifactDigestまで指定すればexact physical realizationへ固定できます。

## Wrong Authority / drift

次はProviderとして受理しません。

- `build-tool`以外のscope
- Contract fingerprint drift
- Conformance Evidence欠落
- Conformance Evidenceが複数
- Recipe / Target / Profile非対応
- Provider artifactとEvidence artifact不一致

resolution成功だけでProvider TrustやBuild Acceptanceを付与しません。

## exact BuilderIdentity

選択したProviderからBuildPlan向けのexact BuilderIdentityを構成します。

```text
BuilderIdentity.ID
  = ProviderID

BuilderIdentity.Version
  = ProviderVersion

BuilderIdentity.ContractVersion
  = ContractID@ContractVersion

BuilderIdentity.ArtifactDigest
  = exact Provider ArtifactDigest
```

Providerの`ImplementationID`はResolution / Lockに残します。

current BuilderIdentityにImplementationID fieldが無いことを理由に、implementation identityを捨てません。

## Fresh resolutionとLocked replay

最重要の分離です。

### fresh resolution

現在のcatalogを見てProviderを選び直します。

後からcompatible Providerが追加されれば、以前は一意だったintentが曖昧になることがあります。

### locked replay

一度確定した:

- Intent fingerprint
- Contract fingerprint
- Provider ID / version
- ImplementationID
- exact ArtifactDigest
- Conformance Evidence fingerprint
- exact BuilderIdentity
- Resolution fingerprint

を再検証します。

catalogへ新Providerが追加されても、locked replayは別Providerへ暗黙切替しません。

```text
fresh catalog:
  Provider A
  Provider B
  -> ambiguous

existing lock:
  exact Provider A / artifact X
  -> replay A
```

Lock対象ProviderのartifactやConformance Evidenceがdriftした場合はfail-closedです。

## bootstrap

このReferenceはstage0 seed resolverを実装しません。

ただしself-hostingでは:

```text
stage0:
  explicit exact seed BuilderIdentity

stage1+:
  resolved / locked Builder Provider
```

のように、seedを自動defaultとして隠さないことを前提にします。

#1103のBuild graphと接続する場合も、stage0 seedはexplicit external seedとして扱います。

## Authority limit

Evidence runnerは次を明示します。

- builder resolution != build execution
- builder resolution != build acceptance
- builder resolution != Runtime Authority
- builder resolution != Provider Trust
- locked replay != fresh resolution

Resolverは「何を使うか」をexactに決める境界であり、「それを信頼してよい」「Buildが成功した」「Runtimeで使ってよい」を決める境界ではありません。

## read-only Evidence runner

```bash
go run ./tools/reference-builder-contract-resolution
```

固定fixtureを使います。

1. Provider Aだけのcatalogをexplicit wiringでfresh resolve
2. Resolution Lockを作成
3. compatible Provider Bをcatalogへ追加
4. fresh resolutionがambiguousになることを確認
5. lock replayはexact Provider Aを維持
6. JSON Evidenceをstdoutへ出力

filesystem / network mutationは行いません。

## 非目標

- production Builder resolver
- actual Go / Rust / JS / Wasm execution
- package manager
- remote build service
- arbitrary shell command
- #951 production primitiveの先回り抽出
- #1096 Go adapterの共通化
- Build scheduler
- Acceptance / Runtime Authority


## 非選択supportはLock identityではない

Providerがselected Intentとは無関係なRecipe / Target / Profileへの対応を後から追加しても、既存locked resolutionをstaleにしません。

```text
selected Intent
  recipe = deterministic-copy
  target = host/reference
  profile = reference
```

に対して、同じexact Provider artifactが`host/other`を追加対応しただけなら:

- fresh catalog metadataは増える
- selected Intentへのeligibilityは維持
- Resolution fingerprintは不変
- locked replayは成功

とします。

選択したRecipe / Target / Profile自体はIntent fingerprintへ入っているため、selected semanticsは固定されています。

## Lock前のcross-field再検証

Resolutionは「resolve関数が返した構造だから正しい」と仮定しません。

`lockBuilderResolution`の入口で次を再計算・照合します。

- Contract内容 -> Contract fingerprint
- Provider Contract ID / version / fingerprint / Conformance profile
- Provider scope = build-tool
- Provider identity -> exact BuilderIdentity
- Resolution fields -> Resolution fingerprint

形式上validな文字列でも、これらの組合せが一致しないResolutionはLock化しません。

Lock側でもProvider ID / version / artifactとBuilderIdentityをcross-checkします。

これにより、resolve後にfieldを差し替えた偽Resolutionや、Providerと無関係なBuilderIdentityをlocked replayへ持ち込むことをfail-closedにします。