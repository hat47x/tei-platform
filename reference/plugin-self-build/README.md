# Plugin self-build Reference

このReferenceは、ISSUE-000079でPlugin self-buildを検証するための独立Workspaceです。通常の`reference/workspace`へprocess Pluginを追加せず、self-build固有のSource → Build → Artifact → Plugin Lockを分離して検証します。

Plugin descriptorの正本は`workspace/plugins/tei-process-reference/1.0.0/plugin.tei`です。実装SourceはRepository既存の`tools/reference-process-plugin/main.go`を使います。実装SourceをこのReferenceへ複製しません。

Source descriptorでは`artifact`を宣言しません。artifact digestはBuildの出力であり、生成前のauthoritative sourceへ書き戻す前提にはしません。Build後はderived Runtime Definitionへdigestを束縛し、既存Plugin Lockへ渡します。

このReferenceが現時点で主張するのは、artifact未生成のSource Definitionを通常のWorkspace loaderで読み込めることまでです。Go toolchain identity、actual build execution、Build Evidence、bit-for-bit reproducibility、stage0 / stage1 / stage2 convergenceは後続Evidenceです。


## E2 runner候補

候補runnerはRepository rootから次のように実行します。

```bash
go run ./tools/reference-plugin-self-build -repo .
```

runnerは開始時と終了時にGit worktreeがcleanであることを要求します。Plugin artifactはRepository外の一時directoryへ出力し、Repositoryへ生成物を残しません。

最初のE2ではSource closureをtracked Repository全体として扱います。また、Go toolchainの実行ファイルだけでなく、compiler / linker /標準ライブラリSourceを含むtoolchain closureをfingerprintします。対象packageに外部Go module依存が見つかった場合は、module cacheを暗黙の入力にせず停止します。

同じSource / builder条件で、別々のbuild cacheを使って2回buildし、SHA-256 artifact digestが一致した場合だけ`reproducible=true`を出力します。その後、Build Plan、BuildArtifactLock、derived Runtime Definition、既存Plugin Lockを相互照合します。

この説明はrunnerの候補実装を示すものであり、actual Evidenceの成立を示すものではありません。exact-headのtrusted checkoutでrunnerとRepository Health Checkが成功するまでSB-2完了とは扱いません。


## SB-1 package self-build

同じrunnerで、通常のReference Workspaceにある`tei.runtime.reference@1.0.0`から決定論的なPlugin packageも生成します。

このpackageはdescriptorだけを対象とする最初のSB-1 profileです。Source DefinitionにRuntime artifact digestが含まれている場合は受け付けません。生成物の種類は`plugin-package`として記録し、`runtime-artifact`とは別のBuildArtifactLock entryにします。

Build Planにはbuilderだけでなく、`tei-plugin-package`というrecipe kindと対象descriptorのentrypointも記録します。そのため、同じSource closureとbuilderを使って別のPluginをpackageした場合でも、Build Plan fingerprintから区別できます。

SB-1 packageの生成に成功しても、Pluginの実行実装をTEI自身で生成できたことにはなりません。process Pluginのruntime artifactを2回buildしてdigest一致を確認するE2とは、別の到達段階として扱います。


## Builder contractと隔離したGo state

ReferenceではGo toolchainのversionと、TEI側のBuild contract versionを別に記録します。現在のcontract versionは`tei.plugin-build/v1`です。

依存確認とruntime artifactの二回buildでは、`GOCACHE`、`GOMODCACHE`、`GOPATH`を一時領域へ分離します。`GOPROXY=off`、`GOSUMDB=off`、`GOVCS=*:off`、`GOTOOLCHAIN=local`も固定し、既存cacheや外部取得を暗黙の入力にしません。

SB-1 packageも同じBuild Planから二回materializeし、二つのSHA-256 digestが一致することをEvidence条件にします。


## Build前のValidation

runnerは、通常Referenceとself-build専用Referenceの双方を既存の`internal/validation.Validate`へ通します。Error diagnosticが残るWorkspaceではpackage materializeやruntime artifact buildへ進みません。

Pluginの妥当性規則をself-build側へ複製せず、既存Validationをそのまま正本として使います。


## Runtime Plugin Lockのbuild trace

process Pluginのruntime artifactをBuildArtifactLockへ固定した後、既存Runtime Plugin Lockへexact Build Plan fingerprintをbindします。

既存のformat 2 lockは読み取り互換を維持します。self-buildされたRuntime implementationを採用するEnvironmentでは、lockをbuild-trace対応のformat 3へ昇格します。

format 3のPlugin entryは`artifactDigest`に加えて`buildPlanFingerprint`を保持します。Source closureやbuilderの詳細はRuntime Lockへ複製せず、Build Planへの参照だけを残します。

runnerはformat 3への昇格、Plugin entryのBuild Plan fingerprint一致、Environment Lock fingerprintの生成まで確認します。


## E3 TEI-native Builtin implementation Source

ISSUE #957では、Plugin descriptorそのものとは別に、TEI-nativeなimplementation SourceをReference化します。

最初のprofileは`static-response-v1`だけです。一般目的の式言語や任意コード実行は導入しません。

```text
plugin-implementation tei.native.runtime.reference v1.0.0 {
    profile static-response-v1
    capability deployment-executor runtime.native_reference

    operation execute json {
        {
            "attemptId": "tei-native-reference",
            "outcome": "accepted"
        }
    }
}
```

この最初のprofileは、一つのPluginについて既存`deployment-executor` Capabilityの`execute` operationを一つだけ実装します。filesystem、network、clock、random等のambient inputは持たず、同じSourceとRequestから同じResponseを返します。追加operation、別Capability kind、未定義operationへの呼び出しはerrorです。ほかのCapabilityへ一般化するのは、#951のcallable contract境界を踏まえた後続課題です。

implementation SourceはCanonicalな`ir.World`へ追加しません。WorkspaceではPlugin専用の`plugin-implementations` source rootに保持し、Source Indexでは`RoleImplementation`として追跡します。Plugin descriptorとimplementation Sourceは別のAuthorityです。

### realizationとRuntime登録

`static-response-v1` Sourceは正規化した決定論的realization manifestへmaterializeします。そのSHA-256 digestを`runtime-artifact`としてBuildArtifactLockとRuntime Plugin Lock v3へ結び付けます。

Runtime adapterは元のSource objectから直接Builtinを作りません。Lock対象のrealization bytesを再parseし、format version、Source fingerprint、canonical bytes、Runtime Definitionのartifact digestを照合した後だけBuiltinへ変換します。したがってReference上では、Lockしたartifactとは別のSource objectを実行する経路を許しません。

Builtin implementation自身にbuild provenanceを自己申告させません。Host側の`BuiltinRegistration`が次を保持します。

- accepted Builtin implementation
- exact Build Plan fingerprint
- realization digest

Runtime Plugin Lockがbuild traceを持つBuiltinでは、Invoke前にこのregistrationを照合します。Build Planまたはrealization digestが一致しない場合はfail-closedです。

legacyなbuild traceなしBuiltinは、既存`Registry.Builtins`経路を維持します。

build-bound Builtinのrealization digestはin-process realizationのidentityであり、外部process artifactと同じ意味ではありません。そのため外部`ArtifactVerifier`を成功したように装いません。Invocation Provenanceには`buildPlanFingerprint`と`realizationDigest`を別に残します。

signature / provenance metadataをbuild-bound Builtinへ付与する契約はまだありません。初期Referenceでは、そのmetadataが存在する場合は検証を省略せずfail-closedにします。

### Reference vertical slice

self-build専用Workspaceへ次を追加します。

```text
tei.native.runtime.reference@1.0.0
  -> TEI-native static-response-v1 Source
  -> deterministic realization
  -> BuildArtifactLock
  -> Runtime Plugin Lock v3
  -> Host BuiltinRegistration
  -> existing runtime.Builtin ABI
  -> existing DeploymentExecutor adapter
```

Referenceでは`application-start`を呼び、`attemptId=tei-native-reference` / `outcome=accepted`を返します。Action ID、Idempotency Key、Executor IDは既存DeploymentExecutor adapterが現在の契約どおり補完します。

既存`runtime.Builtin` ABI、RegistryのRequest / Response、DeploymentExecutor contractは置き換えません。

### runner Evidence

`tools/reference-plugin-self-build`は今後、次の三つを別々のBuild Plan / outputとして記録します。

```text
tei.runtime.reference
  -> plugin-package                 # SB-1

tei.process.reference
  -> runtime-artifact / process     # foreign-language SB-2

tei.native.runtime.reference
  -> runtime-artifact / builtin     # TEI-native E3
```

TEI-native realizationについても同じSourceから二回materializeし、digest一致を要求します。その後Runtime Lock v3へbindし、accepted Builtin registration経由で実際にDeploymentExecutorを呼び出します。

このE3はTEI-native implementation SourceからBuiltin realizationを再構築できる最初のReferenceです。builder自体は依然としてGo seed implementationを使うため、SB-3 self-hostingを達成したとは扱いません。


### interpreted realizationとRuntime Engine

E3の`runtime-artifact`はstandalone executableではありません。`static-response-v1`では、deterministicなrealization bytesをHost内のgeneric Builtin adapterが解釈します。

したがって、完全なexecution identityはrealization digestだけでは決まりません。

```text
execution identity
  = realization artifact
  + Runtime Engine / interpreter
  + Host acceptance
```

現在のReferenceではBuild PlanのSource closureにRepository全体を含めるため、generic adapterのSource変更もBuild Plan fingerprintへ反映されます。しかし、これはproductionのRuntime Engine identityを独立に固定したことを意味しません。

Runtime Engineのexact identity、engine drift検知、native Host / Web / Wasm間のengine bindingは#973で扱います。

そのためE3では次を主張しません。

- realization digest単独でruntime behaviorが完全に固定された
- standalone executable相当のSB-2を達成した
- Runtime Engine差替えに対するreplay safetyが成立した
- Web / Wasm engineとのportable bindingが成立した

E3が確認するのは、TEI-native Sourceをdeterministicなinterpreted realizationへ変換し、Build trace付きHost acceptanceを通して既存Runtime ABIから実行できることまでです。


### exact native builder

E3 runnerはTEI-native realizationをrunner内部のhelperから直接生成しません。`tools/reference-native-plugin-builder`を隔離したGo build stateで2回buildし、builder binaryのSHA-256 digest一致を確認します。

そのexact builder binary digestをnative Build Planの`BuilderIdentity.ArtifactDigest`へ入れます。Builder versionにはRepository revisionを使います。

その後、first builderとsecond builderからrealizationを各1回生成し、artifact bytesとdigestが一致することを要求します。

Go toolchainはnative PluginそのもののBuilder artifactとは扱わず、このReference builderを生成するstage0 seed側のEvidenceとして残します。これでもSB-3 convergenceが成立したことにはなりません。


### exact SB-1 package builder

SB-1 package materializationにもE3と同じBuilder identity原則を適用します。

runnerは`tools/reference-plugin-package-builder`を隔離Go stateで2回buildし、二つのbuilder binary digestが一致することを要求します。そのexact digestをSB-1 Build Planの`BuilderIdentity.ArtifactDigest`へ入れ、Builder versionにはRepository revisionを使います。

確定したBuild PlanをJSON invocation envelopeとして渡し、first / second builder binaryからpackageを各1回生成します。package bytesとdigestが一致した場合だけSB-1 Evidenceを成立させます。

これにより、Go toolchainをpackage builder artifactと呼ばず、GoはReference builderを生成するstage0 seed側へ後退します。builder executionをOS sandboxで隔離したという主張はしません。


### Runtime Engine Lock candidate

#973のpressureを受け、interpreted Builtin realizationについてRuntime Engine identityをPlugin realizationから分離しました。

Reference候補ではengine identityを次で表します。

- engine ID
- engine version
- interpreter / ABI contract version
- implementation digest
- Target

このidentityはPlugin Definition、Build Plan、Runtime Plugin Lockのいずれにも混ぜ込みません。Runtime EngineはPluginのlogical identityでもbuild-only dependencyでもないため、別の`RuntimeEngineLock`でEnvironment Planへbindします。

```text
Runtime Plugin Lock v3
  -> exact Plugin realization / Build Plan

Runtime Engine Lock
  -> exact expected interpreter identity

Environment Plan
  -> realization + engineの組合せ
  -> RuntimePolicyFingerprint
```

build-bound BuiltinはRuntime Engine Lockのbindingが無い場合、Invoke前にfail-closedです。Host-owned `BuiltinRegistration`はBuild Plan fingerprint / realization digestに加えRuntime Engine identityも保持しますが、registration側の値だけをAuthorityにはしません。Environment Planへ先にbindされた期待engineと一致する必要があります。

Invocation / Deployment provenanceには次を残します。

- realization digest
- Runtime Engine identity fingerprint
- Runtime Engine Lock fingerprint

Runtime Engine identityが変わればRuntime Engine Lock fingerprintとRuntimePolicyFingerprintも変わるため、同じrealization bytesを別engineで暗黙にreplayする経路をReference上では拒否できます。

Reference runnerではengine Source closureをRepository全体にはしません。現在は次の実装Sourceだけを固定します。

- `internal/plugin/implementation/model.go`
- `internal/plugin/implementation/builtin/adapter.go`
- `internal/plugin/runtime/model.go`
- `internal/plugin/runtime/adapters.go`

このSource closure digestとGo toolchain digestを組み合わせ、Reference engineの`implementationDigest`を作ります。これにより無関係なRepository変更でengine identityが不要に変わることを避けます。

ただし、このimplementation digestは**実際にHostへlinkされたmachine-code imageのdigestではありません**。Source closureとtoolchainを固定したReference identityです。したがって#973はまだ完了ではありません。

未達として残すもの:

- exact linked / loaded engine imageのattestation
- build flags / linker outputまで含むengine artifact identity
- Web / Wasm engineとのcross-target binding
- production Hostがaccepted engine artifactを独立検証する仕組み

この段階で成立させるのは、「engine identityを暗黙依存から明示Lockへ引き上げ、realizationとの組合せをRuntime PolicyとInvocation provenanceで追跡する」ことまでです。


### SB-3 convergence Evidence candidate

SB-3のactual self-hosting runnerに先立ち、stage chainを記録するread-only Evidence modelを追加しています。

```text
stage0 seed
  -> stage1 current builder
  -> stage2 rebuilt by stage1
  -> bit-for-bit convergence result
```

stage1とstage2は同じSource closureを使い、stage2を実行するbuilderはstage1 outputそのものでなければなりません。outputのlogical builder ID / version / contractも同一であることを要求します。

digestが違う場合はEvidenceを捨てず、`Converged=false`として両digestを保持します。

このmodelは実行機構ではありません。現在のReference package/native builderはまだGo seedから作られており、SB-3 stage executionを実施したとは扱いません。stage0 trustもconvergenceから消去しません。
