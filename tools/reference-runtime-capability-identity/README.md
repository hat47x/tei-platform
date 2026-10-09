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

`RuntimeEnvironmentPlan`から既存`resource.AcquireRequest`へ渡すときは、Runtime Capability Refのfingerprintから決定的なtransport IDを導出します。現在のResource Broker APIへ渡すため、`CapabilityKind`欄には固定値`generic-runtime-ref`を使います。

このtransport kind / IDはAuthorityではありません。任意文字列から作らず、検証済みRuntime Capability Refからだけ導出します。Invocation provenanceには元のRuntime Capability RefとRuntime Policy fingerprintを保持し、Contract、Provider、artifact、Execution Profileへ逆追跡できるようにします。

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

Resource Broker経路はfilesystem read一件のAcquire / Validate / Sandbox mount / Releaseを最初のpressureとします。renew / refresh、Supervisor ownership / fencing、Secret injection、Network endpointはproduction機構として既に存在しますが、このgeneric Runtime RefからのE2E Evidenceはまだありません。

また、`AcceptedRuntimeCandidate`やAuthority Evidenceのfingerprintが信頼できるissuerから来たことは、このReference単体では証明しません。上流のAcceptance / Conformance Authorityとの接続は別の検証点です。

codeを含むため、exact-headのfull `tools/repository-health-check.sh` PASSが得られるまではmainへ統合しません。
