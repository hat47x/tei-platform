# ISSUE-000079: Plugin self-buildをSourceからArtifactまで再現可能にする

- 状態: Active
- 起票日: 2026-10-06
- GitHub Issue: #943
- 関連: `docs/plugins.md` / `docs/conformance/plugin-runtime-m0.22.1.md` / ISSUE-000077 / ADR-0022 / ADR-0028

## 背景

TEIはセルフビルドを課題としています。Pluginについても、出来上がった実装を解決して実行するだけでなく、PluginのSourceからartifactを再構築し、その結果をPlugin LockとEvidenceへ結び付けられる必要があります。

現在のPlugin基盤は、Plugin / Capability / Bundleのidentity、dependency解決、Plugin Lock、artifact digest、builtin / process Runtime、Permission / Resource / Sandbox / Lease / fencingを持っています。これらは「どのPlugin実装を使い、どう実行するか」というRuntime側の境界として機能しています。

不足しているのはBuild側です。現行の`artifact` blockはartifactのidentityを記述できますが、そのartifactがどのSource closureから、どのbuilderとbuild dependencyを使い、どのTarget向けに生成されたかを再現できません。したがって、artifact digestが一致しても、Repositoryだけからbuildをやり直せるとは限りません。

## Product contract

Plugin self-buildでは、次の因果を追跡できることを要求します。

```text
authoritative Plugin / implementation Source
  -> resolved Build inputs
  -> exact Builder
  -> Build execution
  -> Build Evidence / provenance
  -> Artifact digest
  -> Plugin Lock
  -> Runtime resolution
```

Build可能であることは、artifactをAccepted Plugin Lockへ採用できることやRuntimeでinvokeできることを意味しません。Authoring / Acceptance / Build / RuntimeのAuthorityは分離します。

また、Plugin package identity、Capability identity、physical artifact identityを一対一には固定しません。複数Pluginを一つのartifactへまとめるTarget最適化や、一つのlogical PluginをTargetごとに異なるartifactへrealizeする構成を許容します。

## Self-buildの到達段階

「PluginをTEIでbuildできる」という表現だけでは、どこまでself-hostingしたかが曖昧になります。Referenceでは到達段階を分けます。

### SB-1 Package self-build

TEIがPlugin descriptor、TEI authoring source、必要なmetadataから配布packageを決定論的にmaterializeできる段階です。

この段階では、Capability implementationそのものがHost組み込みであっても構いません。Packageを作れることと、実装コードをTEIで生成できることを混同しません。

### SB-2 Implementation self-build

Pluginの実行artifactをBuild contractから生成できる段階です。Go、Rust、JavaScript、WebAssembly等の実装もここへ含めます。

言語ごとに別のPlugin modelを作らず、builder adapterの差として扱います。

### SB-3 Builder self-hosting

TEI-native builderまたはTEI自身のbuild機能を、既存seedからstage1として構築し、stage1からstage2を再構築できる段階です。

```text
stage0 trusted seed
  -> stage1 current builder / standard plugins
  -> stage2 rebuilt by stage1
  -> convergence Evidence
```

stage0を隠しません。seedのidentity、version、artifact digest、trust boundaryをEvidenceに残します。

## 不足している設計要素

### Build Source closure

単一のSource pathだけでは不十分です。Build結果へ影響するSourceをclosureとして扱います。

対象には、Plugin implementation source、TEI source、言語固有manifest / lockfile、code generation input、build parameter、support artifactを含めます。physical pathはlocatorとして利用できますが、Pluginのlogical identityにはしません。

### Build graph

現行の`requires plugin`はRuntime/package dependencyです。compiler、generator、SDK、builder Plugin等を同じgraphへ混ぜません。

Build graphとRuntime graphを分離し、Build-only dependencyがRuntime artifactへ残らないことを確認します。通常のdependency cycleとbootstrap stageも区別します。

### Builder identity

builderは「実行したcommand文字列」ではなく、identityとcontractを持つ対象として扱います。最低限、builder identity / version、builder artifact identity、build contract version、Target / profile、parametersを再現できる必要があります。

将来はbuilder自体をPlugin / Capabilityとして提供できる構造を検討します。ただし、最初のsyntaxとして任意shell scriptを採用しません。

### Build Lock

Plugin LockはRuntimeで利用するexact Plugin / descriptor / artifactを固定します。Build側では、Source closure、builder、build dependency、Target、parameterを固定するBuild Lock相当が必要です。

両者は同じdigestを共有する場合があっても役割が異なります。Runtime LockへBuild情報を無理に詰め込みません。

### Build Evidence / provenance

Build結果から、少なくとも次へ戻れるようにします。

- input Source closure
- resolved build dependency
- builder
- build definition / parameter
- Target
- produced artifact
- execution result
- reproducibility / hermeticityの観測結果

SLSA Build ProvenanceのBuildDefinition / resolvedDependencies / builder / subject分離は参考にします。ただしSLSA schemaそのものをTEIのCanonical modelにはしません。

### Reproducibility

self-buildの成功条件をprocess exit codeだけにしません。同一inputからartifact digestを再現できるかをEvidence化します。

未宣言network access、Host環境依存、timestamp埋め込み等でbit-for-bit再現できない場合は、その差を隠さずUnresolvedとして残します。必要なら将来、bit reproducibilityとsemantic equivalenceを別のConformanceとして定義しますが、最初から両者を同一視しません。

## TEI-native Pluginと他言語Plugin

TEI-native Pluginを実現するために、TEI DSLを一般目的言語へ拡張することを前提にはしません。まずは宣言的に表せる実装、generatorでrealizeする実装、Runtime adapterへ接続する実装を分けます。

他言語Pluginも同じSource → Build → Artifact契約を使います。Go実装だから特別なPlugin kindになる、Wasm実装だから別のCapability identityになる、という設計にはしません。

Web Targetではnative processを前提にしません。WebAssembly Component Modelのようにinterfaceと実装言語を分けられる仕組みは候補ですが、TEI Capabilityの正本をWITへ移すことはしません。

## 最初のReference

最初のReferenceは二段階で進めます。

1. 既存のReference process Pluginを使い、Go implementationからprocess artifactを作るBuild contractを検証する。
2. TEI-nativeにmaterializeできる標準Pluginを選び、SB-1からSB-2へ進める。

この順序により、TEI-native syntaxが未確定のままBuild graph / provenance / Lockまで同時に設計することを避けます。

## 実装順序

### E0 — Contract

Repository正本へ次を固定します。

- SB-1 / SB-2 / SB-3の区別
- Build graphとRuntime graphの分離
- Source / builder / artifact / Lock / Evidenceのidentity関係
- stage0 seedの扱い
- Build Authorityの停止線

### E1 — Build Plan / Build Lock model

最小のread-only modelを追加します。最初はexecutorを含めず、同じ入力から同じBuild Plan fingerprintを作れることを確認します。

Build PlanはPlugin descriptor fingerprintだけでなく、Source closure、builder、Target、build dependencyを含めます。

### E2 — Foreign-language Reference

既存Reference process PluginをGo builder adapterでbuildし、artifact digestとBuild Evidenceを得ます。

この段階ではGoを標準Plugin実装言語に固定しません。

### E3 — TEI-native Reference

標準Pluginを一つ選び、TEI sourceからpackage / implementation artifactをmaterializeします。どこまでをTEI-native implementationと呼べるかをEvidenceで明示します。

### E4 — Bootstrap convergence

builderのstage0 / stage1 / stage2を実行し、同一Sourceからのconvergenceを確認します。stage0 seedをRepository外に置く場合も、そのidentityをBuild Evidenceへ残します。

## Acceptance

- Plugin artifactからSource closure / builder / Target / build definitionへ戻れる。
- Build-only dependencyがRuntime dependencyへ混入しない。
- Build graphの通常cycleとbootstrap stageを区別できる。
- stage0 seedを明示し、隠れたHost依存として扱わない。
- current Sourceからstage1をbuildし、stage1からstage2をbuildできる。
- stage1 / stage2のconvergence判定をEvidenceとして残せる。
- generated artifact digestをPlugin Lockへ結び付けられる。
- Build Evidenceを削除しても業務上のCanonical meaningは変わらない。
- Build Evidenceが存在するだけでAcceptance / Runtime Authorityを得ない。
- TEI-native Pluginを少なくとも一つReferenceでself-buildできる。
- 他言語Pluginもbuilder差し替えで同じBuild contractへ載せられる。
- Web / Desktop等のTarget差でもlogical Plugin identityを維持できる。
- build失敗、unresolved dependency、digest不一致をfail-closedで扱う。
- clean checkoutから再実行できるRepository Evidenceを残す。

## 非目標

全Pluginを直ちにTEI-nativeへ書き直しません。Go等の既存実装も禁止しません。任意shell scriptを標準Build contractにはしません。

Runtime PluginとSemantic Extensionも再統合しません。Build成功だけでartifactをAccepted状態へ昇格させず、stage0 seedが不要になったとも主張しません。

## 外部設計から採るもの

Rust compiler bootstrapからは、stage0 seedを隠さずstage1 / stage2を分ける考え方を採ります。

SLSA Build Provenanceからは、build definition、resolved dependency、builder、artifact subjectを分ける考え方を採ります。

Nix derivationからは、build結果へ影響するinputを明示し、同じinputから同じ結果を再現する考え方を採ります。

WebAssembly Component Modelからは、interface contractと実装言語を分離する考え方を採ります。

これらのschemaやtoolchainをTEI Coreへコピーするのではなく、既存のidentity / Evidence / Authority境界へ翻訳して使います。

## 参考

- Rust Compiler Development Guide: https://rustc-dev-guide.rust-lang.org/building/bootstrapping/intro.html
- SLSA Build Provenance: https://slsa.dev/spec/v1.2/build-provenance
- Nix derivations: https://wiki.nixos.org/wiki/Derivations
- WebAssembly Component Model: https://component-model.bytecodealliance.org/
