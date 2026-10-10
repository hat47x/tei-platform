# Capability Invocation Context設計

- 状態: 評価待ち（未採用）
- 対応Issue: GitHub #1118
- 関連Issue: #951、#1045、#1043、#958
- 役割: Capability呼出しでsemantic inputと実行時の文脈を分離する設計候補
- 採用判断: 将来ADRを作成した場合はADRを正とする。現時点では#1118が作業状態を管理する

## この文書の位置づけ

この文書は、Capabilityへ渡す業務上・意味上の入力と、Hostが実行時に与える認可情報などを分けるための設計候補をまとめる。TEI全体の層構造やPlugin Architectureの第二正本にはせず、この機能内部の境界だけを扱う。

#951では、Capability Contract、Provider、Conformance、Authority scope、read-only Invocationを分離するReferenceが作られている。#1118ではその仕組みを作り直さず、generic Invocationへ実行時の文脈を安全に渡すために不足している境界だけを扱う。

最初のconsumerは#1045のepistemic detail resolverとする。第二consumerには#1043のWeb要求経路を使う。ただし、どちらの既存Security contractやepistemic contractもこの設計へ移さない。

## 現在の結論

Capabilityのsemantic requestとexecution contextは別に扱う。

Contractは、呼出しに必要なexecution context slotの契約identityを宣言できる。一方、token、access ref、session、現在のPrincipalなどの実値はsemantic requestへ保存しない。実値はtrusted Hostが呼出し時に解決し、Contractが宣言したslotだけをProviderへ渡す。

同じsemantic Capability Contractをauthoring-toolとapplication-runtimeの双方で実装できても、ProviderのAuthority scopeは既存#951のまま分離する。scopeが異なるProviderへ暗黙にfallbackしない。

execution contextが結果の可視範囲へ影響する場合、semantic request identityだけで結果をcacheしてはならない。cross-context reuseを禁止するか、Hostが発行する非secretなreuse partitionで結果を分離する。

この結論はまだ製品契約ではない。#1045と#1043の二つのconsumerへ当て、情報漏えい、Authority混同、暗黙依存を防げるかを評価してから採用を判断する。

## 三種類の入力を分ける

### Semantic Request

Semantic Requestは、Capabilityが何を問い合わせるか、何を処理するかを表す入力である。consumerが構築でき、必要に応じて保存やreplayの対象になる。

#1045では`DetailQuery`がこれに当たる。対象assertionと、表示時点に得たmeaning fingerprint、review subject fingerprint、target binding、review log sequenceを持つ。

### Semantic Context

Semantic Contextは、問い合わせの適用範囲や結果の意味に関わる文脈である。単なる実行権限ではない。

#1045の`ContextQuery`は、project、environment、as-ofなどを使ってassertionの適用可能性を再評価するため、semantic inputとして扱う。Hostが観測した値を含む場合でも、結果意味に使うことを理由にExecution Contextへ移さない。

### Execution Context

Execution Contextは、現在の呼出しをどの条件で実行してよいかをHostが決めるための入力である。consumerが自己申告する値として扱わない。

#1045では、authorization境界が与えるopaqueなaccess ref集合が該当する。#1043では、検証済みsessionやPrincipal、認可判断に必要なHost側情報が候補になる。

「tenant」「environment」のようなfield名だけでSemantic ContextかExecution Contextかを固定しない。同じ名称でも、業務上の問い合わせ対象なのか、現在callerの実行境界なのかで役割が変わり得る。

### Semantic Request shapeはcontext gateで書き換えない

Execution Contextを別入力へ移しても、context gateがSemantic Request payloadから「認可らしいfield」を推測して削除してはならない。

どのfieldがSemantic Requestに属するかはCapability Contractのrequest shapeが決める。#951のshape validationとCapability固有decoderが、Contract外fieldをfail closedする。

```text
request shape
  -> Semantic Requestに何を含められるか

Context Slot requirement
  -> Hostが別経路で何を注入するか
```

両方に同じAuthority inputを持たせない。request schemaにExecution Context値を追加してcontext gateとの二経路を作らない。

Context Slot requirement集合もconsumerがrequest内で指定しない。resolved Capability Contractからtrusted Hostが取得する。最初の#1118 Referenceはこのrequirement集合をtrusted inputとして受け取り、Contract Authorityやfingerprint binding自体は後続境界として残す。

## Context Slot Contract

generic Invocationへ自由形式のcontext mapを追加しない。Providerが任意のkeyへ暗黙依存すると、Contractから実行条件を確認できなくなるためである。

Capability Contractは、必要なExecution Contextのslot contractを宣言する候補とする。

概念上は次の情報を分ける。

```text
ContextSlotContract
  identity
  version
  value shape identity
  validation / currentness semantics
  confidentiality / exposure policy

CapabilityContract
  required execution context slot requirements
```

actual token、access ref、session IDなどはContext Slot Contractへ含めない。Capability Contract fingerprintへ含めるのも、必要なslot contractのidentityと互換条件までとする。

Provider Conformanceでは、必要slotを受け取った状態でCapability Contractのrequest、result、failure semanticsを守れることを検証する。

## Requirement declarationをCapability Contractへ束縛する

current Referenceは`ContextSlotRequirement`の集合をtrusted Host inputとして受け取る。これはcontext gateの挙動を試すためのReference上の簡略化であり、production Authorityではない。

required slot一覧をCapability Contractから独立したoptional sidecarにすると、sidecar欠落を「required slotなし」と解釈できるdowngrade経路が生じる。

```text
Accepted Capability Contract
  + optional requirement sidecar

sidecar missing
  -> no required slots
  -> execution context Gateを迂回
```

そのためproductionでは、required execution context declarationをAccepted Capability Contract identityへ必ず束縛する。

actual session、Principal、access ref等のruntime値をContract fingerprintへ含めるのではない。Contractへ結び付けるのは「どのContext Slot Contractを必要とするか」という宣言だけである。

候補は二つある。

1. Capability ContractがexactなExecution Context Profile artifact identity / fingerprintを参照する。
2. 他にも独立したContract拡張artifactが必要になる場合、genericなContract Extension referenceを導入し、そのfingerprintをCapability Contract fingerprintへ含める。

#1118一件だけを理由に汎用Extension機構を先回りして作らない。

### ConformanceProfileへ押し込まない

#951の`ConformanceProfile`はCapability Contract fingerprintの一部だが、現在はprofile identityであり、required slot一覧をmachine-readableに保持するartifactではない。

ここへslot名をopaque stringとして詰めても、Hostがrequired slotを列挙できず、profile本文のdriftやProviderのslot適合性をexactに検査できない。

したがってContext Slot requirementの運搬先として`ConformanceProfile`を流用しない。

productionへ進む場合、少なくとも次のchainへ戻れる必要がある。

```text
Accepted Capability Contract fingerprint
  -> exact context requirement declaration fingerprint
  -> Provider conformance
  -> Host binding receipt
  -> Invocation receipt
```

current Referenceは最初の二つをまだ実証していない。receiptの`context-requirements-are-trusted-contract-input-not-consumer-payload`は、この未実証境界を明示するstop lineである。

## Hostによる注入

Execution Contextの実値はtrusted Hostまたは既存のSecurity / authorization adapterが解決する。

```text
consumer
  -> Semantic Request
  -> Semantic Context

trusted Host
  -> current Execution Contextを解決
  -> 必要slotだけを抽出

generic Invocation
  -> request shapeを検証
  -> required slotを検証
  -> Providerを呼び出す
  -> result shapeを検証
```

Hostが持つambient state全体をProviderへ渡さない。Contractが宣言したslotだけを渡す。

consumer payload内に同名の値が存在しても、Hostが注入するExecution Contextを上書きするAuthorityを与えない。Providerも、宣言されていないambient contextへ依存しない。

## Control ContextとExecution Contextを分ける

#951のgeneric Invocationは現在、Provider implementationを`func(context.Context, []byte) ([]byte, error)`として保持し、callerから受けたGo `context.Context`をそのままProviderへ渡している。

cancellationやdeadlineを伝えるだけなら自然だが、#1043のようにverified Principalを`context.WithValue`へ入れる実装と組み合わせると注意が必要になる。ContractがExecution Context slotを宣言しても、Providerがraw `context.Value`から別の値を読めれば、slot境界を迂回できるためである。

そのため、generic Invocationでは実行制御と実行データも分ける。

```text
Control Context
  cancellation
  deadline
  実行中断に必要なcontrol signal

Execution Context
  Contractが宣言したslot
  Hostが検証したcurrent binding
```

Providerへcallerのambient context全体を見せることを標準にはしない。

Go Referenceを作る場合、元contextのDeadline、Done、Err等のcontrol signalだけを引き継ぎ、Valueを公開しないwrapperを一つの実現候補としてpressureできる。ただし、このwrapper自体を製品Contractにはしない。Wasm、remote Provider、Desktop等はそれぞれの方法で同じ意味境界を実装できるようにする。

Execution Contextを分けても、tracing等のcontrol metadataをどこまでControl Contextへ含めるかはまだ固定しない。初回はauthorization等のAuthorityに関わる値がambient valueとしてProviderへ漏れないことを優先する。

### Invocation receipt

#951の現在のInvocationExecutionはrequest digestとresult digestを記録する。同じSemantic RequestでもExecution Contextによって可視結果が変わる場合、request digestだけでは安全な再利用単位にならない。

一方、raw Principal、token、access ref集合をreceiptへ保存すべきではない。

そのため#1118では、必要に応じて次へ戻れる非secretなreceiptを候補とする。

```text
execution context contract set
Host-issued binding receipt
execution reuse partition
```

これらはsemantic request digest、Capability Lock、Authorization decisionのいずれとも別のidentityである。

## Authority scopeとの関係

Execution Context slotとProviderのAuthority scopeは別の軸である。

#951にはauthoring-tool、build-tool、application-runtime、target-localのscopeがある。Context Slot Contractを追加しても、このscopeを統合しない。

同じCapability Contractにauthoring-tool Providerとapplication-runtime Providerが存在する場合、resolverは要求されたscopeに一致するProviderだけを候補にする。必要なExecution Contextが似ていても、別scopeのProviderをfallbackとして使わない。

Invocationに成功しても、Execution ContextはCanonical Acceptance、objective Truth、reviewer authority、source変更権を与えない。

## Generic Runtime Capability Authorityとの関係

#976は、AcceptedなCapability Contract / Provider / Execution Profileから導出したRuntime Capability identityへ、Permission、Resource、Lease、provenanceを一貫して結び付ける課題を扱う。

#1118のExecution Contextとは責務が異なる。

```text
#976
  どのRuntime Capability realizationを
  どのAuthorityで実行できるか

#1118
  その1回のInvocationへ
  trusted Hostがどのcurrent execution inputを渡すか
```

Execution Context slotの実値をRuntime Capability identity、Permission Grant、Resource Binding、Capability Resolution Lockへ含めない。sessionやauthorizationが変わるたびにCapability identityを作り直す構造にしない。

逆に、Context Slot bindingが存在してもRuntime CapabilityのPermission、Resource、Lease、Sandbox Authorityを自動取得しない。#976側のAuthority成立をContext Slotで代替しない。

同じRuntime Capability identityでも、Invocationごとにverified Principalやaccess contextが異なることはあり得る。その差は#1118のcurrent bindingとreuse partitionで扱い、#976のstable identityとは分ける。

## currentnessとLock

Plugin LockやCapability resolutionはContractとProviderのidentityを固定する。一方、session、authorization、revocation、leaseなどのcurrentnessは呼出し時に確認する。

Lock fingerprintが一致していても、現在のsessionやaccessが有効であるとは扱わない。Execution Contextの実値をLockへ埋め込んで再現性を作ろうとしない。

必要な監査情報は、Security subsystemが発行する非secretで記録可能なopaque receipt reference等で別に扱う候補とする。raw credentialやaccess ref集合を通常のInvocation receiptへ保存しない。receipt reference自体へtoken、cookie、Principal payload等を流用しない。

## cacheとreuse partition

Semantic RequestからExecution Contextを分離すると、cache設計に新しい制約が生じる。

高い権限で取得した結果をSemantic Request identityだけでcacheし、低い権限のcallerへ再利用すると情報が漏れる。そのため、Execution Contextへ依存するresultをcontextをまたいで再利用してはならない。

最初の候補は次の二つである。

1. Execution Context依存resultはcross-context cacheを行わない。
2. trusted Hostが発行する非secretなreuse partition identityでcacheを分離する。

raw tokenやaccess refの単純hashを標準partition identityにはしない。Secretの推測、caller間の相関、rotationや失効の意味をcache layerへ持ち込むためである。

```text
semantic request identity
  != execution reuse partition
  != provider resolution identity
```

Provider resolutionが同じでも、reuse partitionが違えば結果を共有しない。session失効やauthorization変更後に旧partitionを利用できるかどうかもHost側のcurrentness policyで判断する。

## redaction transparency

#1045は、可視assertionのEvidenceやreview metadataの一部がaccess不可なら、hidden identityや件数を返さず`metadataState=partial`を返す。

これは「現在のviewが不完全である」ことをcallerへ知らせる設計である。不完全さを隠すと、hidden reject、Conflict、Evidence等を存在しなかったものとして扱う危険がある。

ただし、追加metadataの存在自体が機密となる環境もあり得る。その場合は、partialを返すこと自体がside channelになる。

そのためproduction policyでは、少なくとも次の差を表現できる余地を残す。

```text
transparent-incomplete
  visible contentは返す
  不完全であることは開示する

existence-sensitive
  不完全さの存在も開示できない
  対象detail全体をunavailableとして扱う
```

このpolicyを最初からCore enumとして固定しない。#1045はtransparent-incompleteのReferenceとして残し、existence-sensitiveは#1118のHost policy pressureで評価する。

## failure semantics

generic Invocation Context層がaccess deniedやnot foundの外部表現を一律に決めない。外部へ何を返すかはCapability Contractのfailure semanticsに属する。

#1045ではexistence oracleを避けるため、unknown assertionとaccess不可assertionを同じ`unavailable`として扱う。

別Capabilityで監査要件や利用者向けエラーが異なる場合、そのContractが必要なfailure semanticsを定義する。Execution Context層は、Contractが要求する失敗表現に必要な情報だけをProviderまたはHostへ渡す。

## Consumer pressure

### Epistemic detail resolver

#1045 / Draft PR #1055では、semantic `DetailQuery`とauthorization `TransportAccess`を分離している。

このconsumerでは次を確認する。

- Semantic Requestにaccess refを含めない
- `ContextQuery`は適用可能性を再評価するSemantic Contextとして維持する
- access contextはHost側から別入力で渡す
- access不可とunknown assertionを同じunavailableとして扱う
- hidden metadataを結果へ漏らさない
- resultをaccess contextをまたいでcacheしない

#1057のqualificationが終わるまでは、#1045をExecutable Evidenceとして扱わない。

### Web要求経路

#1043では、HTTP入口、session、CSRF、Principal対応付け、`security.authorize`、Application Actionの境界を扱っている。

第二pressureでは、検証済みPrincipalやsession由来情報をApplication Actionのsemantic payloadへ混ぜず、Host側から必要な実行文脈として渡せるかを確認する。

#1043の認証・認可機構をContext Slot Contractへ置き換えない。既存Security contractが作った結果をgeneric Invocationへ安全に受け渡せるかだけを評価する。

## Web要求経路へ当てた結果

#1043 / Draft PR #1114へこの設計を当てると、Execution Contextとsemantic requestの境界をさらに絞れる。

現在のWeb Referenceでは、Hostがsession、issuer、CSRFを確認した後、検証済みPrincipalをprivateなGo contextへ格納して後段へ渡している。browserのheader、query、JSONから同じ値を設定する経路はない。

```text
browser request
  -> Host VerifySession / issuer / CSRF
  -> verified Principal
  -> trusted SubjectLink mapping
  -> business SubjectID
  -> security.authorize
       SubjectID / Action / Resource
  -> allowed / denied
```

このverified PrincipalはExecution Contextの候補になる。現在の呼出しでHostが検証したidentityであり、Application Actionのsemantic payloadとしてbrowserに自己申告させる値ではない。

一方、trusted SubjectLinkで得たbusiness SubjectIDまでExecution Contextへ移すべきではない。既存`security.authorize`に渡すSubjectID、Action、Resourceは、Authorization Capabilityが何を判定するかを表すsemantic requestである。

したがって、認証・認可に関わる値を一つのsecurity contextへまとめない。

```text
verified Principal
  -> Execution Context候補

SubjectLink
  -> trusted identity mapping / currentness境界

SubjectID + Action + Resource
  -> security.authorizeのSemantic Request

allowed / denied
  -> Authorization Capabilityのresult
```

authorization resultをContext Slotへ保存して別のCapabilityへ配る構造も、初期候補にはしない。結果には独自の意味、対象、鮮度があるため、単なるambient execution stateへ変えるとAuthorityの由来を失う。

現在の実装で使うGoの`context.WithValue`とprivate keyは、一つのlocal realizationである。Context Slot ContractをGo contextへ固定しない。Desktop、Web、Wasm、remote Provider等でも同じ意味境界を保てる必要がある。

Web Referenceの`SubjectLinkLookup`は、current linkとaccount lifecycleを確認し、unknown / revokedをfail closedする。この構造は、Execution ContextのcurrentnessをPlugin LockやProvider resolutionで代替できないことも示している。

また、Web HostのAuditRecordはcredential、token、cookie、request bodyを保存しない。この方針は、Invocation auditへraw Execution Contextを複製しない設計と整合する。

#1045と#1043を比較すると、共通化すべきものは認証・認可の業務意味ではなく、次の小さい境界である。

```text
semantic input
  != trusted Hostが注入するcurrent execution input
  != Capabilityが返すAuthority / decision result
```

この結果だけでContext Slot Contractのproduction shapeは固定しない。最初のReferenceでは、slot requirement、Host binding、missing / undeclared contextのfail-close、result / cacheへの漏えい防止に絞る。

## 必須の反例

Referenceでは少なくとも次を拒否する必要がある。

- callerがSemantic Requestへaccess refやsession情報を埋め込み、HostのExecution Contextを上書きする
- ProviderがContract未宣言のambient contextへ依存する
- 必須context slotが無い状態で空値を補って実行する
- authoring-tool呼出しがapplication-runtime Providerへfallbackする
- context内のSecretやpolicy identifierをresultへ反射する
- high-privilege resultをlow-privilege callerの同一Semantic Requestへcache再利用する
- revoked sessionの旧reuse partitionを無条件に再利用する
- raw tokenやaccess ref hashを外部へ見えるcache / audit identityとして使う
- Lock一致だけで現在のauthorizationを有効と判断する
- Execution Contextの存在をCanonical Acceptanceやobjective Truthの根拠にする

## 実装方針

#951の既存Referenceへ機能を追加し続けない。#951はReference saturationによりfeature追加をfreezeしている。

#1118でReferenceを作る場合も、Capability resolver、Provider selection、shape validation、read-only Invocationを再実装しない。既存generic Invocationへ渡す最小のcontext requirement / bindingをpressureする形に限定する。

最初からproductionのcontext registry、Secret store、tenant model、DI containerを作らない。

## 評価と採用判断

この設計は、少なくとも#1045と#1043の二つのconsumerへ当てる。

評価では次を確認する。

- semantic requestからExecution Contextを除いても必要な意味情報を失わない
- required slotをContractから検査できる
- Host以外がExecution Contextを自己申告できない
- undeclared / missing slotをfail closedできる
- Providerへ必要slotだけを渡せる
- cross-context cache leakageを防げる
- scope違いのProviderを混同しない
- auditへSecretを複製しない

二つのconsumerで共通化の必要性と最小境界が確認できた時点で、ADR化とproduction最小deltaを判断する。

## この設計が主張しないこと

この文書は、次を実装済み・検証済みとは扱わない。

- production authorization provider
- production Context Slot Registry
- session / tenant / Secret管理
- cross-process context propagation
- distributed tracing
- universal deadline / locale contract
- production cache partition実装
- publisher trustや署名検証
- #951 Referenceのproduction採用
- #1045、#1043のqualification完了
