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

## 既存TEI BuildPlan・BuildArtifactLockとの参照接続（#976）

`project_advice_build_lock_linux_test.go` は、独立して`go build`した案件Go実行体と、既存`plugin.BuildPlan`／`LockBuildOutput`／`VerifyRuntimePluginBuild`を接続します。実ファイル由来のArtifact SHA-256、Go実行コマンドのSHA-256、代表的な案件ソース・`go.mod`のSHA-256をBuildPlanに保持し、実行体のdigestとRuntime Plugin Lock、Generic Runtime Refが一致することをテストします。加えて、選択したソースのdigest、builder、build契約、Target、Artifact、Runtime Provider／Plugin Lockが変わった場合に旧BuildPlan／Lockの再利用が拒否されることを確認します。既存`PinLinuxProcess`による実ファイル検査・子プロセス実行も同じ試験で通します。

**このBuildPlanはテスト用観測記録であり、正式なBuilder発行結果ではありません。** 特に列挙したソースは、案件Goファイルと`go.mod`の代表例だけです。Go標準ライブラリ、間接依存、ビルドタグ、全環境変数、Goツールチェーン全体の**完全なSource Closureではなく**、Builderの署名・契約発行権限・再現性・信頼根拠も証明しません。また旧`plugin.Definition`はここでは試験用のProcess形だけで、案件Service Contract／ConformanceのAccepted証跡ではありません。従ってBuildPlan/Artifact Lockの整合だけを根拠に実行権限を付与してはなりません。

本番移行では、既存セルフビルドのSource Closureと正式Builder受理、現行HostのContract/Provider Acceptance Reader、既存Execution Authorityを同一exact artifactへ接続することが必要です。二重Builder・Resolver・Authorityは追加しません。

## 呼出し時のHost Environment Authority再照合（#976、限定参照）

`current_host_environment.go` の `invokeWithCurrentHostEnvironment` は、**Hostに固定されたEnvironment ID**と、Host所有の既存Authority正本を読む関数を受け取る薄い参照用アダプターです。前回のPlan生成時点のGrantをそのまま再利用せず、各呼出しで現在のEnvironment Authorityを取得し、既存 `verifyRuntimeEnvironmentPlan` で同一Generic Runtime Ref／権限世代を照合してから、既存のArtifact Verifier／Resource Broker／Host Sandbox経路へ渡します。ReaderはPlugin自身やブラウザー要求に任せません。

Linux限定の `project_advice_current_host_environment_linux_test.go` は、同じ実案件Goバイナリ／同一Generic Runtime Refを二つのEnvironmentで共有し、AのExecution Grant撤去後は古いA Planによる再起動を拒否しながら、Bの実プロセス実行を維持します。Authority Evidenceの世代変更、Reader障害、別Environment、重複Grant、Reader欠落、キャンセルは、**Artifact検査・Sandbox/Process起動前**に拒否します。これは新しいGrant Store/Resolver/Permission Keyの実装ではありません。

**重要な未達**：本参照Readerはテスト内の合成Sourceであり、実Hostの認証済みApplication／Environment正本やIssuer Trustへまだ接続していません。照合完了から実起動までのAuthority失効は競合可能であり、これだけではAtomic fencing／運用Leaseの証明になりません。実OS隔離・正式なContract/Conformance受理・署名Artifact・業務Action認可にも昇格しません。本番側では#1370など既存の実行開始Gateと正本の世代管理を流用し、追加の権限体系を作らず詰める必要があります。

## 署名済み独立案件Go実行体と現在のHost Authorityの交差検証（#976）

`project_advice_signed_current_host_linux_test.go` は、案件所有 `cmd/reference-project-advice-process` を独立ビルドし、既存の `PinSignedLinuxProcess`、`LinuxHostSignerTrust`、`invokeWithCurrentHostEnvironment`、`HostSandboxLauncher` を**改造せず**組み合わせるLinux限定試験です。Hostを模した試験コードが一時的にEd25519鍵を作り、案件Go実行体の正確なDigest／Plugin ID／Version／署名参照に対して署名します。公開鍵の信頼は、Plugin申告ではなく独立した試験Hostの信頼集合へ登録します。

同じ固定済み実行体で二つのEnvironmentを処理し、Environment AだけのGrant失効後もBの実行は継続する一方、**共通署名参照の失効はBの次回呼出しも拒否**することを検証します。署名参照へ同じ公開鍵を再登録しても、旧世代に固定した実行体は復権せず、新しい固定・検証が必要です。未知の署名者、署名改変、異なるPlugin IDによる署名使い回しも拒否します。異なる失効の単位を同じResolver／権限Storeへ混ぜません。

これは署名と現行Environment照合の**組合せ試験**であり、テスト鍵は現実の受理済み発行者ではありません。現在のEnvironment readerも試験用の合成値で、正式なService Contract、Provider Conformance、Build/Source ClosureのIssuer受理、運用時Principal／Action Grant、実Resource Lease／OS Sandboxは未成立です。署名者世代の起動Gateも一Host上の範囲であり、Environment Grantの失効と実行開始の原子的な順序付けや分散Host間の失効伝播を証明しません。#1370の並行PRを変更するものでもありません。

## Grantを読む時点とOS起動時点の競合の再現（#976、E39）

`project_advice_authority_start_gap_linux_test.go` は、**既知の未閉塞境界を成功する回帰診断として固定する**Linux専用試験です。既存の`invokeWithCurrentHostEnvironment` が現在のExecution Grantを読んだ後、既存の`PinnedLinuxProcess.Verify`へ進む直前に、テストHostでそのGrantを撤回します。単一スレッドの検証フックで順序を固定するため、スリープ時間や並行Map書込みに依存しません。

現行参照実装では、**撤回後でも最初の要求の実Go子プロセスが起動し、補助処理の成功応答を返します**。次の要求は、Readerが撤回を検出してArtifact Verifier／実行へ進まず拒否されます。このテストのPASSは、権限競合が*解消されたことではなく、実際に残っていること*の証拠です。Host signerの既存Start Gateによる署名者失効制御は、EnvironmentごとのGrantの失効制御とは別の対象です。

### 本番への最小接続条件

- Host管理の正本だけが、Environment・Runtime Capability Ref・実行世代に対する新規入場可否を決める。ブラウザー、案件Plugin、旧Runtime Planの提示を新規Authorityとみなさない。
- 「正本からの読込み → Verify → 実行開始」を一つの**既存Hostの入場／世代境界**で扱い、失効が完了した後に旧世代で新規Startを許さない。必要なら既存Process Launcherの開始Hook／Resource Leaseのstart admissionに結び付ける。署名者用のGateとは独立のGrant Storeを作らず、同じRuntime Capability Refを用いる。
- 新規入場拒否、先に入場済みの実行の扱い、失効完了時点、子プロセスの継続・中断、運用Auditを明示する。署名者失効とEnvironment利用拒否とArtifact全体の失効を混同しない。
- まず同一Linux Hostで、失効とStartをチャネル同期した正例・負例を確認する。別Host／Wasm／遠隔実行への保証を暗黙に拡張しない。

この節は本番用の新しい権限APIや実装の採用を意味しません。テスト用のEnvironment Grant、Contract/Provider/Conformance/Issuer、OS隔離等の未達は引き続き維持します。

## E40: Grant失効と実プロセス入場の順序づけを試験Hostで実証（#976）

`project_advice_guarded_host_admission_linux_test.go` は、E39（#1845）の既知競合に対する**保守的な比較用の正例**です。既存の `invokeWithCurrentHostEnvironment`、独立Go案件実行体、`PinnedLinuxProcess`、`HostSandboxLauncher` は変更しません。テスト専用のHost正本がEnvironmentごとに `sync.RWMutex` を持ち、読取り側が **Grantの現在確認から実プロセス終了まで** 入場を保持します。撤去側は書込みロックを取得してGrantを撤去します。

試験では、AのGrant読取り後／Artifact検証前に同期用Barrierで止め、同時にAのGrant失効を要求します。この時点では失効は完了できず、既に入場したAが実Goプロセスを終えた後で完了します。その後のAの再実行はArtifact検査前に拒否されます。Aの実行が停止している間でも別Environment BのGrant撤去は独立して完了し、Bの次回実行は拒否されます。

**採用上の留保**：これはテストHost内の順序づけを示すだけで、製品の新しいGrant Storeや権限APIではありません。読取りロックを実プロセス終了まで保持するため、長時間処理に対して**失効を長時間待たせる**方式です。本番の推奨実装とはせず、既存 `StdioProcessLauncher` のOS Start境界／Hostの現在のAuthority世代を用い、**失効完了後に旧世代Start不可**を保証しつつStart後すぐ入場ロックを解放する設計を別途検証します。入場済み子プロセスを止めるべきかは別のPolicyです。なお、E39が検出した**無防備な経路は引き続き存在**し、E40のPASSは製品修正の成立を意味しません。

正式Contract/Provider Conformance、Hostの認証済み実Authority Source、Resource Brokerの実Lease・fencing、署名Issuer、本物のOS Sandbox、複数Hostの失効同期には未接続です。
