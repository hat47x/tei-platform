# Generic Runtime Capability Identity Reference

GitHub Issue #976「Generic Runtime Capability Identityをproduction Authorityへ接続する」のReferenceです。

このReferenceは、Coreの`CapabilityKind`へ新しい列挙値を追加せず、新しいApplication Runtime CapabilityをEnvironmentの実行Authorityへ結び付けられるかを検証します。production schemaの採用案ではありません。

主にRQ-032「Plugin / DSLで意味論を宣言・合成できること」のうち、Runtime CapabilityをCore固定enumへ戻さずに既存Authority境界へ接続できるかという不確実性を減らします。

## 防ぎたい失敗

現在のproduction Runtimeでは、Permission Grant、Resource Binding、Lease Policy、Resource Brokerへの要求、Invocation provenanceが`CapabilityKind + CapabilityID`を前提にしています。新しいCapabilityを追加するたびにCore enumを増やすと、標準機能もPluginで提供する方針と衝突します。

一方、`CapabilityKind`を単なる任意文字列へ置き換えるだけでは、どのContractとProviderを実行するのかを検証できません。Permissionだけをgeneric化しても、Resourceや実行経路に旧identityが残ればAuthorityが二重になります。

このReferenceでは、実行対象の意味と実装をexactに固定したRuntime Capability Refを作り、Environment固有のExecution Grant、Permission、Resource、Leaseを同じRefへ結び付けます。

## Runtime Capability Ref

`RuntimeCapabilityRef`は、Acceptedな上流結果から次を固定します。

- Contract ID、version、fingerprint
- Provider Plugin ID、version、implementation ID
- exact artifact digest
- realization Target
- Execution Profile fingerprint
- Capability Resolution Lock fingerprint

Ref自身にはResource ID、mount path、secret target、Lease Policyを入れません。これらはEnvironment固有のAuthorityだからです。

Runtime Capability Refが存在するだけでは実行できません。Environmentにexact RefとExecution Profileへ結び付いた`RuntimeExecutionGrant`が一件だけ必要です。Authority Evidenceが更新された場合、Runtime Capability Refのidentityは変えず、Runtime Environment Planだけをstaleとして扱います。

## Environment Authority

`RuntimeEnvironmentAuthority`はRuntime Capability Refのfingerprintをkeyにして、次を結び付けます。

- Execution Grant
- Permission Grant
- Resource Binding
- Lease Policy

Execution Profileが要求するPermissionとEnvironmentのGrantはexactに一致しなければなりません。必要なPermissionが足りない場合だけでなく、Profileに無いPermissionをEnvironment側が追加した場合も拒否します。

同じRuntime Capability Refにlegacy `CapabilityKind + CapabilityID`のAuthority claimとgeneric Authorityを同時に持たせる状態も拒否します。移行期間に二つの独立した権限系を作らないためです。

別のRuntime Capability Ref向けにAuthorityが追加されても、既存Planのidentityは変えません。

## legacy Capabilityからの一方向migration

既存の`CapabilityKind + CapabilityID`は、このReferenceで直ちに削除しません。ただし、generic Runtime Capability Refと並ぶ第二のAuthorityとして残し続けることもしません。

`LegacyRuntimeProjection`は、authoritativeな旧`plugin.Definition`から対象Capabilityを一件だけ取り出し、Accepted Runtime CandidateとExecution Profileへ結び付けます。旧PluginのID / versionとexact artifactがAccepted Providerと一致し、旧Capabilityが宣言するPermissionとExecution Profileが一致する場合だけRuntime Capability Refへ投影します。

projectionは旧Capability declarationのfingerprintを保持します。旧宣言が変わった場合は、保存済みprojectionをcurrentとして扱いません。

Environment側は`LegacyEnvironmentAuthorityProjection`として分けます。旧Environmentの対象Capabilityに結び付いたPermission Grant、Resource Binding、Lease Policyだけをgeneric Runtime Capability Refへ投影します。別Capability向けの設定はprojection identityへ含めません。

legacy Environmentの存在だけからExecution Authorityを作ることはありません。migrationには、別途AcceptedなExecution Authority Evidenceが必要です。

projection結果のgeneric Authorityにはlegacy claimを残しません。旧Authorityとgeneric Authorityを同時に有効化する状態は`RuntimeEnvironmentPlan`が拒否します。migrationは旧設定を読み取って新Authority候補を作る一方向の操作であり、新旧の設定を二つの独立正本として同期する仕組みではありません。

保存済みmigrationを再利用するときは、Plugin declarationとEnvironment Authorityの両projectionをcurrent sourceへreplayします。

## 既存Resource BrokerとHost Sandboxの再利用

最初の実行pressureは`host-enforced-sandbox`のprocess realizationだけを対象にします。

Resource BrokerのAuthority subjectとPlugin processのdispatch identityは分けます。

Resource Brokerへ渡すsubject IDはfull Runtime Capability Refのfingerprintから導出します。Execution ProfileやResolution Lockが変わればRuntime Authorityも変わるため、Resource Authority subjectも変わります。現行`resource.AcquireRequest`の`CapabilityKind`欄には固定値`generic-runtime-authority`を使います。

process dispatch IDはContract ID / version / fingerprintとimplementation IDから導出します。Execution ProfileやEnvironment Authorityが変わっても、同じContract semanticsを同じimplementationで処理する限りdispatch IDは変えません。現行`plugin/runtime.Request`の`CapabilityKind`欄には固定値`generic-runtime-dispatch`を使います。

どちらのkind / IDもAuthorityそのものではありません。任意文字列から作らず、検証済みRuntime Capability Refから決定的に導出するcompatibility envelopeです。Invocation provenanceには元のRuntime Capability Ref、Runtime Policy fingerprint、Resource Authority subject、process dispatch IDを保持し、Contract、Provider、artifact、Execution Profileへ逆追跡できるようにします。

実行前には既存`plugin/runtime.ArtifactVerifier`でexact artifactを検証します。digest、必要なsignature、provenanceの検証を通過するまでResource Authorityを取得しません。

Permissionがある場合は既存Resource Brokerでleaseを取得し、`resource.ValidateLease`を通した後、既存`HostSandboxLauncher`へ渡します。filesystem resourceは既存Sandbox mountへ変換されます。呼出し後は既存Brokerでleaseをreleaseします。

このReference専用のPermission Store、Resource Broker、process launcher、sandbox実装は追加しません。

## Negative control

Reference testでは少なくとも次を拒否します。

- stringだけで作ったRuntime Capability identity
- Execution Profileと異なるTarget
- Contract、artifact、Target、Execution Profileのdriftを同じRuntime Capability identityとして扱うこと
- Runtime Capability Refの存在だけでExecution Authorityを得ること
- Environment Permissionの不足と過剰付与
- legacy Authorityとgeneric Authorityの二重化
- process realizationのartifact drift後にResource Authorityを取得すること
- Artifact Verifierが無い、またはdigestを検証できない状態でResource Authorityを取得すること
- Resource Brokerが無いPermission付きCapabilityの実行
- `host-trusted` Profileをsandbox専用経路へ暗黙転送すること

## 現在のEvidenceと非主張範囲

このReferenceで確認するのは、generic Runtime identity、Environment Authority、既存Resource Broker / Host Sandboxの接続境界です。

まだproductionの`plugin.Definition`、`EnvironmentPlan`、`permission.Grant`、`Registry.Invoke`を変更していません。Host-trusted builtinのgeneric実行経路も未実装です。

generic実行は既存Artifact Verifier、Resource Broker、HostSandboxLauncherへ直接委譲するReference orchestrationです。production `Registry.Invoke`へgeneric identityを組み込んだことはまだ意味しません。

現行process protocolへはderived dispatch IDを渡します。このIDはContract semantics + implementationから静的に再現できます。

Reference testには、同じprocess artifact内の二つのgeneric Contractを別々のdispatch IDへ写し、既存`StdioProcessLauncher`で実OS processを起動するpressureも追加しています。子processはEnvironment固有のRuntime Refを知らなくても、Contract fingerprintとimplementation IDから事前計算できるdispatch IDで処理を分けられます。Responseが別dispatch IDを返した場合はHost側で拒否します。

このtestはまだexact-head Gateを通していません。したがって、現行`Request / Response`へ追加metadataが不要だと確定したわけではありません。Gate結果と、より複雑なProvider構成での再利用性を見て判断します。

Resource Broker経路はfilesystem read一件のAcquire / Validate / Sandbox mount / Releaseを最初のpressureとします。renew / refresh、Supervisor ownership / fencing、Secret injection、Network endpointはproduction機構として既に存在しますが、このgeneric Runtime RefからのE2E Evidenceはまだありません。

また、`AcceptedRuntimeCandidate`やAuthority Evidenceのfingerprintが信頼できるissuerから来たことは、このReference単体では証明しません。上流のAcceptance / Conformance Authorityとの接続は別の検証点です。

codeを含むため、exact-headのfull `tools/repository-health-check.sh` PASSが得られるまではmainへ統合しません。

## 案件専用Go助言処理のConsumer pressure（#1567 → #976）

[統合済みの案件Go/Lit参照実装](https://github.com/hat47x/tei-platform/pull/1567)を、Generic Runtime Capability Refの移行条件に当てた**独立した否定試験**です。Goの案件補助提案は正式な受注ActionやAccepted Validationではありません。

現状の`tools/reference-project-advice-lit/contract_bridge.go`にある`ProjectAdvisoryContract`は、`Version=v0`および説明的な`Shape=customer-code-note-to-advisory-v0`を持ちます。Compositionの選択を示すだけで、#976のRuntime Capability Candidateに必要な受理済みsemantic version・契約fingerprint・Provider Plugin/実artifact/Resolution Lock・Execution Grantにはなりません。実行Hostも合成`ArtifactDigest`と簡易Guardを使う診断用です。

`project_advice_consumer_test.go`では、*将来正式に受理されると仮定した合成Candidate*を、既存`deriveRuntimeCapabilityRef`と`buildRuntimeEnvironmentPlan`に渡します。検証内容は、旧`v0`・説明的Shape・欠落Lock／Artifactの拒否、Runtime Refの存在だけでの実行不可、明示Grantの失効・重複拒否、legacy/generic二重Authorityの拒否、Provider・Contract fingerprint・Artifact・Resolution Lock・Permissionの差し替え時の拒否です。新Resolver・Permission Store・Grant発行機構は作りません。

**注意**：合成Candidateの`1.0.0`、fingerprint、artifact digest、Authority Evidenceはテスト材料であり、実案件プラグインが`v1`として受理された証拠でも実成果物の署名・照合結果でもありません。テストでは本番の`Registry.Invoke`、Resource Brokerの現行leaseやHost Sandboxによる案件Go実行、業務`order_entry` Actionの委譲を実行しません。今後は既存#951の契約受理・Conformanceと#976のRuntime Authorityへ、同一のexact Candidate/Lockを接続する必要があります。

局所試験は `go test -count=1 ./tools/reference-runtime-capability-identity ./plugins/reference-project-customer-advice` を実行します。専用Workflowも同じ2パッケージをPR正確HEADで検証します。

## 案件Goロジックの実子プロセスpressure（#976、後続）

`project_advice_real_process_test.go` は、統合済み `plugins/reference-project-customer-advice` の実際の `Evaluate` をGoテスト実行ファイルへリンクし、**別OSプロセス**で実行します。既存 `StdioProcessLauncher`、`HostSandboxLauncher`、`invokeWithExistingHostBoundary` の経路を用い、案件補助提案とエラーを通信します。テストでは `os.Executable()` の実bytesからSHA-256を取り、Host側の検査者が実ファイルと宣言Digestを照合します。Digest不一致時にポリシー適用器・子プロセス呼出しへ到達しないことも確認します。

**受理範囲は限定的**です。このバイナリはテストプロセス自身であり、実際に配布・署名された案件Plugin成果物ではありません。Contract `1.0.0`、Conformance／Resolution Lock、Environment Execution Grantは依然として**合成候補**です。また使用するネットワーク／ファイルシステムPolicy Enforcerは既存のテスト用記録器であり、OS隔離は行っていません。検査したファイルを同じFDから起動する実行ファイル固定、署名／発行者の信頼、Permission Brokerの実Lease・fencing、運用時のPrincipal/Action認可、Accepted業務処理の委譲も証明しません。従って、実子プロセスの成功をProduction Host Authority成立の証拠にしてはなりません。

現在の主目的は、**案件Goロジックの呼出し経路を既存generic Runtime機構へ接続できること**と、**未検証artifactを実行前に拒否する順序**を試験することです。正式化は#976/#951等の既存Authority／受理機構から進めます。

## 独立Go実行体のBuild・Linux固定実行（#976、追加pressure）

`project_advice_built_pinned_linux_test.go` は、案件の既存Goパッケージから`cmd/reference-project-advice-process`を**独立したOS実行ファイルとして`go build`**し、`os.Executable()`を再利用した前段より実配備に近い境界を試します。既存`PinLinuxProcess`を**ArtifactVerifier兼実行器**に使い、実行バイナリの内容から計算したSHA-256と宣言Digestの一致をホスト側で確認し、同一保持FDを介して子プロセスを起動します。実行前のDigest不一致は固定処理で拒否し、固定後に元のパスを別ファイルへ置き換えても、保持した内容で案件の実Go補助ロジックを実行できることを検証します。

プロセスの入口は既存のProcess Request/Responseの必要項目だけを解釈する**案件所有の薄いアダプター**であり、Runtimeの権限系・Contract Resolver・Business Actionを複製しません。ホストが導出したdispatch IDは、ホスト設定である起動引数から照合する限定的な仕組みです。引数の存在は**Contract Acceptanceを証明しません**。

**未成立**：Contract / Conformance / Resolution Lock / Execution Grantはテスト用合成Evidenceのまま。Pluginの正式Builder・Plugin Lock・署名・発行者、Runtimeのproduction`Registry.Invoke`、実Permission Broker／Lease・fencing、OSのネットワーク／ファイルシステム隔離、業務Action認可は結合していません。Network/Filesystem Enforcerはテスト用記録器です。Linuxの保持FDはTOCTOU低減のみを示し、署名・実OS隔離・完全な本番Admissionを意味しません。
