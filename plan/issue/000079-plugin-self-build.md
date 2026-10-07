# ISSUE-000079: Plugin self-buildをSourceからArtifactまで再現可能にする

- 状態: In Progress
- 起票日: 2026-10-06
- GitHub Issue: #943
- 関連: `docs/plugins.md` / `docs/conformance/plugin-runtime-m0.22.1.md` / ISSUE-000077 / ISSUE-000080 / GitHub #957 / ADR-0022 / ADR-0028
- 関連RQ: RQ-021 / RQ-023 / RQ-034

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

Self-buildの実行にTEI Studioや特定のAgent Adapterを必須にしません。clean checkoutから通常のCLI / Repository Toolで再実行できることを基準にし、ChatGPT等のチャット実行環境、Codex、Claude Code、CI等も同じRepository contractを利用できる構造にします。Agent Sessionや一般的なTest状態をBuild Lockへ持ち込まず、Build再現性に必要なSource / builder / dependency / artifactの情報だけを扱います。

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

将来はbuilder自体をPlugin / Capabilityとして提供できる構造を検討します。ただし、最初のsyntaxとして任意shell scriptを採用しません。PluginがCore固定enumを増やさず新しいCapability Contractを提供する境界はISSUE-000080で扱い、本Issueから独自のbuilder専用Capability機構を作りません。

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

TEI-native implementation SourceとBuiltin realizationの不足はGitHub #957で扱います。

現行の`runtime builtin`は、Plugin descriptorから実装を生成する方式ではなく、Hostが`runtime.Registry.Builtins`へGo implementationを登録する方式です。したがって、SB-1 package materializeだけではTEI-native implementation self-buildは成立しません。

#957では既存`runtime.Builtin` ABIを維持したまま、TEI-native implementation Source → realization → provider registrationを最小Referenceで実証します。Semantic ValidationはAccepted meaningを評価する別Authorityなので、Plugin implementation Sourceへ転用しません。

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
- TEI StudioやAgent Adapterなしに、通常のCLI / Repository Toolから同じself-build経路を実行できる。
- Agent Session、Workspace、一般的な検証状態をBuild Plan / Build Lockのidentityへ混ぜない。

## 非目標

全Pluginを直ちにTEI-nativeへ書き直しません。Go等の既存実装も禁止しません。任意shell scriptを標準Build contractにはしません。

Runtime PluginとSemantic Extensionも再統合しません。Build成功だけでartifactをAccepted状態へ昇格させず、stage0 seedが不要になったとも主張しません。

また、self-buildのために専用Agent Runtimeや常駐Adapterを必須にしません。開発ToolやCIが所有するSession / Job状態をTEIのBuild modelへ複製することも非目標です。

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


## 2026-10-06 E1 candidate — Build Plan / Build Lock

E1の最初の候補実装として、`internal/plugin/build.go`へread-onlyなBuild Plan / Build Lock modelを追加しました。

Build Planは、Plugin identity、descriptor fingerprint、Source closure、builder identity、build-only dependency、Target / profile、非secret parameterを保持します。Build dependencyは既存の`Definition.Requires`へ統合せず、Runtime / package dependencyと分離しました。

Build Planのfingerprintは、Source、dependency、parameterの宣言順に依存しません。一方、Source digest / locator、builder、dependency、Target、parameterのいずれかが変わればfingerprintも変わります。Sourceやbuilderのdigestが未固定の場合、重複したSource / dependency / parameterがある場合はfail-closedで停止します。

Build Lockは、Plugin / TargetごとにBuild Plan fingerprintと生成artifact digestを結び付けます。既存Plugin Lockを置き換えるものではありません。Build Lockは「どのbuild inputsから何が生成されたか」を固定し、Plugin Lockは「RuntimeでどのPlugin実装を使うか」を固定します。

このincrementでは、次を意図的に実装していません。

- `.tei`のBuild syntax
- Workspace loader
- build executor
- Go固有builder adapter
- Build Evidence / provenance artifact
- Plugin Lockとのcross-check
- stage0 / stage1 / stage2 runner

そのため、E1全体が完了したとは扱いません。追加したunit testはfingerprintの順序不変性、入力変更によるfingerprint変更、未固定inputと重複inputの拒否、Build Lockのartifact bindingを対象としています。

この実行環境ではRepositoryのfull checkoutを使った`go test`と`tools/repository-health-check.sh`を実行できません。コードはcandidateとしてDraft PRに置き、exact-headの検証Evidenceが得られるまでmainへ統合しません。


## 2026-10-06 E1再監査とE2候補

E1候補を実Referenceへ接続する過程で、二つの境界を修正しました。

第一に、既存packageにはRuntime Plugin Lockを生成する`BuildLock(...)`関数があるため、候補modelの`BuildLock`型はGo上で名前が衝突します。候補modelは`BuildArtifactLock`へ改名し、Runtime Plugin Lockとの役割差も型名で明示しました。

第二に、既存`DefinitionFingerprint`は`artifact.digest`を含みます。このfingerprintをBuild Planへそのまま使うと、生成後に決まるartifact digestを生成前に要求する循環が生じます。このため、Build Planではartifact identityを除いた`BuildDefinitionFingerprint`を使います。Build後は`RealizeBuiltDefinition`でSource Definitionを変更せずderived Runtime Definitionへartifact digestを結び付け、従来の`DefinitionFingerprint`とPlugin Lockへ渡します。

これにより、次の経路を候補実装で表せるようになりました。

```text
artifact-free Source Definition
  -> BuildDefinitionFingerprint
  -> Build Plan
  -> produced artifact
  -> BuildArtifactLock
  -> derived Runtime Definition
  -> existing Runtime Plugin Lock
```

`VerifyRuntimePluginBuild`は、Plugin identity、build definition fingerprint、Build Plan fingerprint、Target / profile、Runtime descriptor fingerprint、artifact digestを照合します。Build側の記録だけでRuntime Authorityを得ることはありません。

また、`BuildDependency`のidentityをTEIの`CanonicalID`に限定する案を撤回しました。外部module、package、SDK等をTEI固有IDへ変換せず記録できるよう、kind / string identity / version / artifact digestの組にしています。Plugin runtime dependencyとは引き続き別です。

### E2 Reference source

既存`tools/reference-process-plugin/main.go`には実process実装がある一方、通常Reference Workspaceには対応するartifact-freeなSource Plugin Definitionがありませんでした。mainの通常Reference Worldを変更せず、`reference/plugin-self-build/workspace`を独立Referenceとして追加しました。

このWorkspaceでは`tei.process.reference@1.0.0`をprocess Pluginとして宣言し、artifact digestはSourceへ記述しません。実装Sourceは既存`tools/reference-process-plugin/main.go`を再利用し、Reference配下へ複製しません。

### E2 runner candidate

`tools/reference-plugin-self-build`を候補実装として追加しました。trustedなclean checkoutで実行した場合、次を一続きに検証する構成です。

1. tracked source全体をSHA-256の保守的なSource closureとして固定する。
2. Source Plugin DefinitionをWorkspace loaderで読む。
3. 実際に使用するGo toolchain closureをfingerprintする。
4. 対象packageが外部Go moduleへ依存していないことを確認する。
5. network取得を禁止し、`GOWORK=off`、`GOENV=off`、`GOTOOLCHAIN=local`、`CGO_ENABLED=0`等を固定する。
6. 別々のbuild cacheで同じPluginを2回buildする。
7. 二つのartifact digestが一致しなければ失敗する。
8. BuildArtifactLockを作る。
9. Source Definitionからderived Runtime Definitionを作る。
10. 既存Plugin Lockを生成し、Build Plan / artifact / Runtime Lockを相互照合する。
11. build後もRepositoryがcleanであることを確認する。

初期Referenceでは、Source closureまたはGo toolchain closureにsymlinkがある場合は推測して追跡せず停止します。

Source closureは現時点ではRepository全体です。これは過剰包含ですが入力の取りこぼしを避けるための意図的な初期条件です。E2 Evidence成立後に、必要であれば`go list`等から正確なbuild dependency closureへ狭めます。

このrunnerはまだexact-headで実行していません。したがってactual artifact digest、reproducibility成立、SB-2達成は未主張です。Draft PR #949の検証Gateを維持します。


## 2026-10-06 Build recipe・出力種別・SB-1 package self-build

E1の静的監査を続けた結果、Build Planにはbuild対象そのものを表す情報が不足していました。Source、builder、Targetが同じでも、別のentrypointをbuildすれば結果は変わります。runner内の定数だけに依存すると、その差をBuild Planから説明できません。

このため、Build Planに`BuildRecipe`を追加しました。初期形は`kind`と`entrypoint`だけを持ちます。任意のshell commandはCanonicalなBuild contractへ入れません。Go Referenceでは`go-package`と`./tools/reference-process-plugin`を記録します。将来、TEI-native builderや別言語のbuilderを使う場合も、builder adapterがrecipeを解釈する構造を維持します。

また、Buildの生成物を一種類として扱うと、SB-1で作ったPlugin packageをRuntime artifactとして誤ってPlugin Lockへ接続できます。そこで`BuildOutputKind`を追加し、少なくとも次を区別します。

- `plugin-package`
- `runtime-artifact`
- `builder-artifact`

値自体は拡張可能とし、Runtimeとの照合だけが`runtime-artifact`を要求します。BuildArtifactLockの一意性にもoutput kindを含めるため、同じPlugin / TargetについてpackageとRuntime artifactを別々に記録できます。

Build Definitionのfingerprint計算では、正規化前にdeep cloneするようにしました。fingerprintを計算しただけでauthoritative Sourceのslice順が変わらないことをtestで固定しています。

### SB-1 package materializer

`internal/plugin/package.go`に、最初のSB-1 package profileを追加しました。artifact-freeなPlugin Definitionと`plugin-package`向けBuild Planから、正規化済みDefinitionを含む決定論的なJSON packageを生成します。

packageにはBuild Plan fingerprintとBuild Definition fingerprintを含めます。同じ意味のDefinitionでCapability内の宣言順だけが違う場合は、同じpackage bytesとdigestになります。一方、Build Planのprofile等が変わればpackage bytesも変わります。

この初期profileではRuntime artifactを含むSource Definitionを受け付けません。SB-1のpackage outputとSB-2のRuntime outputを別経路に保つためです。

通常のReference Workspaceにある`tei.runtime.reference@1.0.0`を使い、実descriptorからSB-1 packageをmaterializeするReference testも追加しました。これは標準Pluginのpackage self-build候補であり、Capability実装そのものをTEIで生成できたという主張ではありません。

### runnerへのSB-1統合

`tools/reference-plugin-self-build`は、process PluginのSB-2候補だけでなく、標準`tei.runtime.reference`のSB-1 packageも同じEvidenceに含めるよう更新しました。

runnerがactualに成功した場合、少なくとも次の二つを別のBuild Plan / BuildArtifactLockとして確認できます。

```text
tei.runtime.reference
  -> plugin-package

tei.process.reference
  -> runtime-artifact
```

前者はSB-1、後者はForeign-language ReferenceとしてのSB-2候補です。両者を同じ「self-build済み」という一語へまとめません。

actual executionは引き続き#956で確認します。SB-1 packageのmaterializerとReference testを追加したことだけでは、Repository Healthやactual Evidenceの成立は主張しません。


### Builder contract versionとGo実行環境の固定

builderの実体versionと、TEIがそのbuilderへ渡すBuild contractのversionは別の事実です。同じGo toolchainを使っていても、Build Planの解釈規則が変われば同じbuildとは扱えません。

そのため、`BuilderIdentity`に`contractVersion`を追加しました。現在のReferenceは`tei.plugin-build/v1`を使います。toolchain version、builder artifact digest、contract versionのいずれかが変わればBuild Plan fingerprintも変わります。

Go Reference runnerでは、既存の利用者環境を暗黙に参照しないよう、`GOCACHE`、`GOMODCACHE`、`GOPATH`を一時領域へ分離します。依存確認と二回のbuildは、それぞれ隔離したstateを使います。

さらに、`go list`にも`-mod=readonly`を指定し、`GOPROXY=off`に加えて`GOSUMDB=off`、`GOVCS=*:off`を固定しました。外部moduleやVCSを暗黙に取得して成功したように見せないためです。

SB-1 packageも同じBuild Planから二回materializeし、package digestが一致する場合だけEvidenceを成立させます。これにより、SB-1とSB-2の双方で「同じ入力から同じ出力」を直接確認します。


### 既存Plugin Validationの再利用

SB-1/SB-2 runnerは、Workspaceを読み込めたことだけではbuildへ進みません。通常Referenceとself-build専用Referenceの双方について、既存の`internal/validation.Validate`を実行し、Error diagnosticが一つでもあれば停止します。

Build層へPlugin Validation規則を複製しないためです。Runtime kind、Capability kind、Action、Permission、Artifact、Environment binding等の検証は既存のValidationを正本とし、Build Planとpackage materializerはbuild固有のidentity・reproducibility・authority境界だけを受け持ちます。


### Runtime Plugin LockからBuild Planへの逆追跡

E1の再監査で、artifact digestだけではRuntime Lockからexact Build Planへ一意に戻れないことが分かりました。

異なるBuild Planが同じbytesを生成する場合や、複数のlogical Pluginを一つのphysical artifactへまとめるTarget最適化では、artifact digestだけを逆引きkeyにできません。

このため、既存format 2 Runtime Plugin Lockは互換性のため維持し、self-buildされたRuntime implementationを明示的に採用するときだけ`buildPlanFingerprint`をPlugin entryへbindしてformat 3へ昇格する候補実装を追加しました。

```text
Build Plan
  -> plan fingerprint
  -> BuildArtifactLock
       + artifact digest
  -> Runtime Plugin Lock v3
       + buildPlanFingerprint
       + artifactDigest
  -> EnvironmentLockFingerprint
  -> Runtime / Durable Invocation
```

Runtime Plugin LockへSource closureやbuilderの詳細を複製しません。exact Build Planへの参照だけを保持します。

format 2は引き続き読み取り可能です。format 3では`buildPlanFingerprint`が存在する場合、その値が64桁hexでなければ拒否します。package outputをRuntime Lockへbindすること、artifact digestが一致しないbuildをbindすること、既に別Build Planへbind済みのentryを書き換えることもfail-closedにします。

これにより、Plugin Lock fingerprint自体がBuild Plan bindingを含むため、Durable Invocationが保持するPlugin Lock fingerprintからself-build provenanceへ戻れる経路を維持できます。


## 2026-10-07 E3 candidate — TEI-native interpreted Plugin realization

E3では、Plugin descriptorだけでなく実装Source自体をTEI-nativeに保持し、Runtimeで利用できるrealizationへ変換する最初のReferenceを#957 / Draft PR #969で進めます。

最初の`static-response-v1`は汎用言語ではありません。一つのPlugin Definition、一つの`deployment-executor` Capability、一つのvalidなdeployment Action、一つの`execute` operationだけを許します。Permission、Plugin runtime dependency、external runtime dependencyは許可しません。

responseからHost-ownedな`executorId`、`actionId`、`idempotencyKey`、`pluginInvocation`を設定できません。`externalRef`や`observedAt`で外部世界を観測したように主張することもできません。

implementation SourceはCanonicalな`ir.World`へ追加せず、Plugin専用の`plugin-implementations` source rootから読み込み、Source Indexでは`RoleImplementation`として追跡します。

### Source → artifact bytes → Runtime

TEI-native Sourceはdeterministic realization manifestへmaterializeします。invalid UTF-8、duplicate JSON key、複数top-level value、過剰nesting、unsupported fieldをfail-closedで拒否します。

Runtime adapterはauthoring時のSource objectを直接実行しません。Lock対象のexact realization bytesを再parseし、format version、Source fingerprint、canonical bytes、Runtime Definition artifact digestを照合した後だけ既存`runtime.Builtin` ABIへ変換します。

```text
TEI-native Source
  -> canonical realization bytes
  -> BuildArtifactLock
  -> Runtime Plugin Lock v3
  -> exact realization bytesを再parse
  -> existing runtime.Builtin ABI
```

authoritative Plugin Definitionはartifact-freeのまま保持し、build後にだけderived Runtime Definitionへartifact digestを結び付けます。

native realizationはrunner内部のhelperで直接生成せず、`tools/reference-native-plugin-builder`を隔離Go stateで2回buildします。builder binary digest一致を要求し、そのexact digestをBuild Planの`BuilderIdentity.ArtifactDigest`へ入れます。Go toolchainはnative PluginのBuilder artifactではなく、このReference builderを生成するstage0 seed側のEvidenceとして残します。

builder processは空のenvironmentで実行しますが、OS sandboxやnetwork namespaceで隔離したとは主張しません。hermetic build executionは#943の既存Reproducibility / hermeticity要件として残します。

build-bound BuiltinではHost-owned `BuiltinRegistration`がBuild Plan fingerprintとrealization digestを保持し、Lockと不一致ならInvoke前に停止します。preflightで受理したBuiltin instanceをInvokeまで保持し、registration差替えによるTOCTOUも避けます。

### interpreted realizationという限界

E3の`runtime-artifact`はstandalone executableではなく、Host内のgeneric interpreter / Builtin adapterが解釈するartifactです。完全なexecution identityはrealization digestだけでは決まりません。

```text
execution identity
  = Plugin realization artifact
  + Runtime Engine / interpreter
  + Host acceptance
```

現在のReferenceではRepository全体を保守的なBuild Plan Source closureに含めるためengine Source変更もBuild Plan fingerprintへ波及します。ただしproductionのexact Runtime Engine identityを独立に固定できたことは意味しません。この不足は#973で追跡します。

#973が解決するまでは、standalone executable相当のSB-2達成、realization digest単独による完全なruntime behavior固定、engine差替えへのreplay safety、Web / Wasm engine bindingを主張しません。

#971でfocused test、self-build Evidence v1、既存SB-1 / Go process SB-2 regression、TEI-native actual invocation、full Repository Health、clean pre/post stateを確認します。actual Evidenceが得られるまでは#969をReady化・mergeしません。

### 2026-10-07 SB-1 builder identity再監査

E3でexact native builder identityを導入した後、SB-1 package materializationにも同じ問題が残っていることを確認しました。runner内のGo helperがpackageを生成しているにもかかわらず、Build PlanではGo toolchain digestをBuilder artifactとしていました。

この非対称性を解消するため、`tools/reference-plugin-package-builder`を追加します。runnerはこのbuilderをisolated Go stateで2回buildし、binary digest一致を確認します。そのexact builder digestをSB-1 Build Planへ入れ、first / second builderからpackageを各1回生成してpackage bytes / digest一致を要求します。

Go toolchainはSB-1 packageの直接Builderではなく、Reference builderを生成するstage0 seed側のEvidenceとして扱います。これによってSB-1とTEI-native E3でBuilder identityの意味を揃えます。

ただし、builder executable自体のSB-3 stage convergenceやOS-level hermetic executionが成立したという主張はしません。
