# ISSUE-000080: Plugin Capability contractをCore固定enumから分離する

- 状態: In Progress
- 起票日: 2026-10-06
- GitHub Issue: #951
- 関連: RQ-019 / RQ-023 / RQ-031 / RQ-032 / RQ-033 / ISSUE-000077 / ISSUE-000079 / ISSUE-000078 / GitHub #950

## 背景

TEIでは、標準機能も第三者Extensionと同じ契約で追加できることをProduct Requirementとしている。一方、現在のRuntime Plugin実装では、提供可能なCapabilityの種類を`internal/plugin/model.go`の`CapabilityKind`へ固定列挙している。

現在の列挙にはDatabase Provider、Migration Provider、Deployment Executor、Observation Provider、Evidence Provider、Contract Query Executor、Editor Intelligence等がある。これは既存Referenceを成立させる実装としては機能しているが、新しい種類のCapabilityをPluginとして追加するたびにCoreのGo codeを変更する構造でもある。

Reference Model Transferでは、Xead / Xeafを読むSource Adapterが最初の具体的な反例になった。これを`editor-intelligence`等へ便宜的に押し込めばCore変更は避けられるが、契約の意味とAuthority scopeが不明瞭になる。反対に`source-adapter`を新しい`CapabilityKind`としてCoreへ足せば、Plugin-firstという要件を満たさない。

## 減らす不確実性

このIssueでは、次を判断できる状態を目指す。

> Pluginが新しい実行Capabilityの契約をCoreの固定enumへ追加せず定義でき、それでも契約の意味、互換性、Conformance、実行Authorityを厳格に検証できるか。

単に`CapabilityKind`を任意文字列へ変えることは解決ではない。名前を自由に増やせても、request / result / failure / unknownの意味やprovider適合性が検証できなければ、Plugin間の契約は実質的に非型付けになる。

## 分離するもの

### Capability Contract

Capability Contractは、Plugin packageや物理artifactとは独立したcallable contractとして扱う。

最小候補は次である。

- logical identity
- contract version
- normalized contract fingerprint
- request / result semantics
- failure / unknown semantics
- Conformance profile
- Target / execution条件へ接続する情報

最終schemaをこのIssueで先に固定しない。Referenceで必要になった差だけを追加する。

### Provider implementation

特定PluginがCapability Contractを実装する。

providerの自己申告だけで「互換」とみなさない。AcceptedなConformance Evidenceへ戻れることを要求する。複数providerが同じcontractを満たす場合、package IDやload順で一つを暗黙選択しない。

### Resolution / Authority scope

同じCapability Contractの共通機構を使っても、解決するAuthorityは分ける。

```text
semantic validation / acceptance
authoring tool
build tool
application runtime Environment
Target-local tooling / realization
```

たとえばReference ModelのSource Adapterはauthoring toolであり、Application RuntimeのEnvironment authorityを必要としない。Database ProviderはRuntime scopeで解決され、Source Adapterのfallbackにはならない。

scopeを単なるlabelとして追加するのではなく、どのLock、Permission、Resource、Acceptance境界が適用されるかを決める解決境界として扱う。

## Semantic Interfaceとの関係

ISSUE-000077では、Semantic Extensionが必要とするSemantic InterfaceをReference化している。Semantic InterfaceはAccepted meaningやSemantic Evaluatorの選択へ関わる。

本IssueのCapability Contractは、Tool / Runtimeで実行するサービス契約を対象とする。両者は同じAuthorityではない。

ただし、次の共通primitiveを再利用できる可能性がある。

- identity
- contract version
- normalized fingerprint
- provider conformance
- explicit wiring
- conflict / unresolved explanation

共通primitiveを使えることと、Semantic Lock、Tool Lock、Runtime Plugin Lockを一つへ統合することは別である。Environment差によってAccepted semantic validityが変わる設計にはしない。

## 最初のReference

最初のpressureには二つの異なるscopeを使う。

### A. Authoring Tool — Xead / Xeaf Source Adapter

GitHub #950のsource-faithful adapterを、Core固有の`source-adapter` enumを追加せず提供する。

最低限、次を確認する。

- exact input bytesへboundする
- read-onlyのsource factsを返す
- unknown constructをlossとして保持できる
- adapter installだけではCanonical Acceptanceを得ない
- adapter installだけではApplication Runtime Authorityを得ない

### B. Runtime — 既存Capability

既存の`contract-query-executor`またはDatabase Providerをnegative controlとして使う。

Authoring Tool scopeからRuntime providerを暗黙利用しないこと、Runtime EnvironmentがSource AdapterをCapability fallbackとして選ばないことを確認する。

この二つで不足する場合だけ、Build scopeやTarget-local scopeを追加する。ISSUE-000079のbuilderを、最初から同じReferenceへ巻き込まない。

## Resolverに要求する性質

- load順、Catalog列挙順、package名の辞書順でproviderを選ばない
- compatible providerが複数残る場合、明示wiringまたはAccepted policyなしに選択しない
- exact contract fingerprintとConformance profileを照合する
- Authority scopeが異なるproviderをfallbackに使わない
- unsupported Targetを別implementationへ切り替える場合もcontract semanticsを変えない
- selected contract / provider / artifact / Conformance Evidenceへ逆追跡できる
- providerが無い、互換contractが無い、Evidenceが足りない状態を成功へ畳まない

## Physical packaging

Capability ContractとPlugin package、artifact、processを一対一に固定しない。

同じlogical contractをDesktopではbuiltin、Webではlinked / Wasm等で実現できる余地を持つ。複数のlogical Pluginを一つのartifactへまとめても、Conformance、Evidence、Authorityから元のcontractとprovider identityへ戻れなければならない。

process非対応Targetで暗黙に別の意味へfallbackしない。

## 既存CapabilityKindの移行

既存`CapabilityKind`を一度に削除しない。

最初にgeneric contractのReferenceを成立させ、その後で既存kindを標準Capability Contractまたはtyped profileとして投影できるかを確認する。

移行期間に旧kindと新contractを二つの独立正本にしない。どちらがauthorityかを明示し、最終的に一方向へ収束できる設計にする。

## 完了条件

- Core enum追加なしで新しいCapability ContractをReferenceへ追加できる
- Xead / Xeaf Source Adapterをauthoring-tool scopeで解決できる
- Runtime scopeとのAuthority混同をnegative testで拒否できる
- provider Conformanceと明示wiringを検証できる
- callable ProviderのConformanceをexact implementation artifact identityへ結び付けられる
- fresh resolutionと採用済みresolution replayを分け、Catalog成長だけで採用済みProviderを自動更新しない
- Contract / Provider replayとcurrent Conformance Evidenceの検証を分けられる
- logical contractとphysical artifact / processの境界を分離できる
- process非対応Targetでも同じcontractを別realizationへ載せられる構造を説明できる
- existing `CapabilityKind`を二重Authorityにせず一方向に移行するpressureを示せる
- existing `CapabilityKind`をどう移行するかをADR候補として説明できる
- ISSUE-000077のSemantic InterfaceとAuthorityを混同しない

## 非目標

- 任意文字列Capabilityを無検証で許可すること
- Semantic InterfaceとRuntime / Tool Capabilityを同じAuthorityへ統合すること
- Source Adapter専用のCore enumを追加すること
- 任意のremote RPCを標準Plugin contractにすること
- Plugin Marketplaceの設計
- 既存CapabilityKindを一度に全面置換すること

## E0 Reference candidate — 2026-10-06

`tools/reference-plugin-capability-contract/`にproduction型を変更しないReference resolverを追加した。

最初のcontractは`authoring.source-adapter.xead`で、Authority scopeを`authoring-tool`として扱う。contract identity / version / fingerprintとprovider Plugin / implementation / artifactを分離し、providerの自己申告だけでは解決対象にしない。exact contract fingerprintとConformance profileへ結び付くEvidenceを要求する。

Contractにはrequest / result / failureに加えてunknown semanticsも明示する。unknown定義を省略したContractは拒否し、unknown semanticsだけが変わった場合もContract fingerprintを変える。Source Adapterで未解釈constructを保持することと、入力不正や実行失敗を成功扱いしないことを同じ状態へ畳まない。

Referenceでは次のnegative controlを置いた。

- Conformance Evidenceが無いproviderを拒否する
- `application-runtime` scopeのproviderを`authoring-tool`要求のfallbackに使わない
- 複数の適合providerがある場合、明示wiringなしで選ばない
- contract fingerprintが変わったproviderを拒否する
- Evidenceだけを更新してもcontract / provider resolution identityは変えず、Evidence fingerprintだけを変える

さらに敵対評価で、最高compatible contractを先に固定すると、そのversionにproviderが無いだけでlower compatible versionの有効解を失うことが分かった。Reference resolverはcontract versionとprovider / Conformanceの組を高いversionから探索し、完全な解があるlower versionへ退避できるcandidateへ修正した。

明示wiringも`ImplementationID`だけでは不足する。同名implementationを別Pluginが提供できるため、Plugin ID / Plugin version / Implementation IDの組で選択する。

現時点ではReference candidateであり、productionの`internal/plugin.CapabilityKind`は変更していない。exact-headのrepository health checkは未実施であり、PRはDraftとして扱う。

## E1 Resolution Lock candidate — 2026-10-06

E0のresolverだけでは、同じRequirementを後日もう一度解決したとき、Catalogへ追加された新しいContract versionやProviderへ自動的に切り替わります。Source Adapter等のTool Capabilityでは、同じSourceを扱っていても実装が変わり得るため、AIやToolが長期間にわたって同じ処理へ再入場する条件として不十分でした。

ReferenceにCapability Resolution Lock candidateを追加しました。Lockはresolution policy、Authority scope、root Requirement、明示wiring、exact Contract / fingerprint、Provider identity、artifact digestを固定します。Lock自身にもfingerprintを持たせます。

fresh resolutionとLock replayは別の操作として扱います。Catalogへcompatibleな新versionが追加された場合、fresh resolutionは新しい完全解を選べます。一方、既存Lockのreplayは記録済みのexact Contract / Providerが現在も同じ内容で存在することを確認し、新versionへ自動更新しません。

Conformance EvidenceもLockから分離しました。Evidence更新だけではresolution identityを変えません。current Evidenceが必要な処理では、Lock replayの後に別の検査を行います。これにより、exact Contract / Provider replay、current Conformance Evidence、Runtime / Tool execution Authorityを区別します。

敵対的な確認では、Conformance EvidenceがProvider artifactへ結び付いていない問題も見つかりました。同じPlugin / version / implementation IDのartifactだけを差し替えると、古いEvidenceを再利用できるためです。ReferenceではEvidenceへexact artifact digestを追加し、artifact差し替え時は新しいConformance Evidenceを要求します。

また、同じPlugin / version / implementation IDを異なるartifact digestで複数宣言すると、明示wiringだけではartifactを一意に指定できません。この状態はload順依存を生むため、logical provider identityの重複として拒否します。

現時点ではReference candidateであり、productionのPlugin Lockや`internal/plugin.CapabilityKind`は変更していません。focused testとfull Repository healthを通した後も、Reference結果からproduction schemaを直接固定せず、既存Capabilityの移行pressureと合わせてADR候補を判断します。

## E2 既存CapabilityKind migration pressure — 2026-10-06

既存`CapabilityKind`からgeneric contractへ移行する際、新旧のProvider declarationを独立してauthorすると二重正本になります。Referenceでは、現行`plugin.Definition`をAuthorityとして保持し、既存Capabilityからgeneric Provider候補を一方向に導出するcompatibility projection candidateを追加しました。

最初の対象は既存の`contract-query-executor`です。`reference/workspace`から`tei.database.oracle / database.query.oracle`と`tei.database.postgres / database.query.postgres`を実際に読み、`application-runtime.contract-query-executor`のProvider候補へ投影します。

projectionは旧宣言をConformanceへ自動昇格しません。generic ContractへのConformance Evidenceが無ければresolverへは採用されません。旧Plugin定義から対象Capabilityが消えた場合もprojectionはfail closedになります。

このpressureから、現行builtin Pluginにはもう一つmigration prerequisiteがあることも分かりました。Reference WorkspaceのOracle / PostgreSQL Pluginはartifact digestを持たないため、どの実装bitに対するConformance Evidenceかを固定できません。callable generic Providerを採用するにはexact implementation artifact digestを必須とし、legacy builtinは宣言の投影までに留めます。実装artifact identityが信頼できるbuild / provenance経路で確定した後にだけ、Conformance Evidenceとresolverへ進めます。

OracleとPostgresの両方がConformanceを満たしても、generic resolverは暗黙選択しません。既存`contract-query-executor`ではEnvironment / Database instance / Contract Query targetごとのBindingがすでに選択Authorityを持つため、generic Contract側へ第二のRuntime wiringを追加しません。既存Bindingで選択されたProviderだけをConformance検証へ渡します。投影したruntime providerをauthoring-tool scopeのSource Adapterへfallbackすることも拒否します。

projectionはread-onlyです。旧Plugin定義をsortやnormalizeの副作用で変更せず、移行bridgeがAuthority sourceを書き換えないことをtestします。

追加の敵対評価では、projectionがPlugin / version / implementation / artifactへだけ結び付いていると、投影元のlegacy Capability payloadが変わっても同じgeneric Providerとして見える問題が分かりました。Referenceではlegacy Capabilityを正規化した`sourceDeclarationFingerprint`をProviderへ持たせ、Resolution identityとConformance Evidenceの双方へ結び付けます。列挙順だけの差は吸収し、payload変更はdriftとして扱います。

また、artifact identityをまだ持たないlegacy ProviderをCatalog全体のinvalid inputにはしません。候補として存在してもConformance対象から外し、別scopeや別contractの正常な解決は継続します。一方、同じlogical provider identityを未確定projectionとexactな新Provider declarationで重複させた場合は、移行中の二重正本になるためfail closedにします。

この段階でもproductionの`CapabilityKind`は削除しません。次に判断すべき点は、generic contract / provider / scope / lockのReference契約が既存Runtime Plugin Lockとどこまで共有できるか、また旧`CapabilityKind`を標準contract profileへ投影する一方向の移行期間をどう終了するかです。


## E3 Runtime Plugin Lock binding pressure — 2026-10-07

generic Capability Resolution LockをApplication Runtimeで利用する場合も、新しいRuntime Authorityを作らず、既存Runtime Plugin Lockへ接続するReference candidateを追加した。

`RuntimeCapabilityBinding`は、Environment、Capability Lock fingerprint、ProviderのPlugin ID / version、Plugin definition fingerprint、artifact digestを結び付ける。Environment Lock全体のfingerprintはidentityへ含めない。同じEnvironmentに無関係なPluginが追加された場合まで、対象Capabilityのbindingを無効にしないためである。

対象Pluginの削除、definition fingerprintの変更、artifact digestの変更はfail closedとする。legacy migration pathでは、Capability Lock内のProviderが現在のauthoritative `plugin.Definition`から一方向に投影された結果と一致することも確認する。これにより、旧Capability宣言とgeneric Provider宣言を二つのAuthorityにしない。

Runtime Plugin Lock format 3の`buildPlanFingerprint`はCapability resolution identityへ含めない。同じartifactに対してBuild provenanceだけが更新された場合、Capability identityは変えない。Build provenanceは既存Plugin Lock / BuildArtifactLockから追跡する。

このpressureで、現行Runtime Plugin Lockのpackaging上の限界も明確になった。Plugin LockはPluginごとに一つの`artifactDigest`を持つため、一つのPluginを同じTargetで複数のCapability implementation artifactへ分割した構成を表現できない。ReferenceではProvider artifactとPlugin-level artifactが異なる場合を拒否し、対応済みとは扱わない。

この不足はRQ-024のWeb可搬性、lazy loading、Wasm分割、artifact coalescingと直接関係する。Plugin Lockへ新しいfieldを先に追加せず、ISSUE-000079のBuildArtifactLockとTarget packaging pressureを通して、provider-level artifact identityをRuntime Authorityへ固定する必要が実際に再現するかを確認する。


## E3 Runtime Authority integration — 2026-10-07

Application Runtime scopeでgeneric Capability Resolution Lockを新しいRuntime Authorityにしないため、既存Runtime Plugin Lockへ接続する`RuntimeCapabilityBinding` candidateを追加しました。対象EnvironmentのPlugin entryからPlugin definition fingerprintとartifact digestを照合し、Capability Lockが指すProviderと一致する場合だけ接続します。

Environment Lock全体のfingerprintはCapability Bindingへ固定しません。対象Providerと無関係なPlugin追加までstaleにしないためです。一方、対象Pluginの削除、definition drift、artifact差し替えは拒否します。legacy migration pathでは、Capability LockのProviderが現在のlegacy Plugin definitionから一方向に導出されたことも再確認します。

さらに、Contract Query runtimeには既存のtarget-specific Authorityがあります。

```text
Environment
  + Database instance
  + Contract Query target
      -> ContractQueryExecutorBinding
      -> executor Capability
```

この選択をgeneric Contractの`Wiring`で再実装すると二重Authorityになります。Referenceではproductionの`ResolvedContractQueryExecutor`を先に受け取り、選択済みPlugin / Capabilityだけをgeneric ContractのConformance検証へ渡します。この経路のCapability Resolution Lockはgeneric `Wiring`を持ちません。

target-specificな選択は`RuntimeCapabilitySelectionBinding`へ分離し、Environment、Database instance、Contract Query target、executor Capability、Plugin identity、Capability Lock、Runtime Capability Bindingをfingerprintで結びます。Oracle targetとPostgreSQL targetが同じgeneric Contractを要求しても、それぞれ既存Bindingに従って別Providerを利用できるpressureを追加しました。

この結果から、Capability Contractの共通resolverが常にProvider selection Authorityを持つとは限らないことが分かります。既存のdomain / runtime Bindingが選択Authorityを持つ場合、generic Contractは選択済みProviderの契約適合性を検証する層として使います。generic Contract自体が選択Authorityを持つscopeでは、曖昧なProviderを明示wiringまたはAccepted policyなしに選ばない規則を維持します。

Selectionのfreshnessもfingerprintの自己整合だけでは判定しません。legacy pathでは、current Runtime Plugin Lockとauthoritative `plugin.Definition`を照合し、`ResolveEnvironmentWithLock`でEnvironmentのPlugin setを再現したうえで、productionの`ResolveContractQueryExecutor`をcurrent Bindingに対して再実行します。Binding削除、重複、別executorへの変更、Plugin definition drift、artifact driftはいずれも停止条件です。

current Conformance Evidenceの検証はこのreplayへ混ぜません。Runtime Authorityのcurrent性と、Contract適合Evidenceのcurrent性は別々に判定します。


## E4 Target realization pressure — 2026-10-07

Web可搬性をCapability Contractへ反映するため、TargetごとのrealizationをReference resolverへ追加した。

Contract identityはTargetに依存させない。同じPlugin / version / implementation IDを保ったまま、DesktopとWeb Browserで別artifactを宣言できる。Provider declarationとConformance Evidenceはrealization対象Targetとexact artifactを保持し、resolution identity / Capability Resolution Lockは選択したTargetを含む。

Targetを指定した解決ではexact Targetだけを候補にする。Target未指定Providerを「任意Target対応」とみなさず、process前提Provider等をWebへ暗黙fallbackしない。反対にTarget context自体が無い場合も、Target固有Providerのどれかを推測して選ばない。

同じlogical ProviderのTarget違いは共存できるが、同じTarget内で同じlogical Provider identityを複数artifactへ重複宣言した場合はfail closedとする。明示wiringはlogical Providerを選び、物理realizationはTarget contextによって一意化する。

Capability Resolution LockはTarget driftも拒否する。Runtime Capability BindingにもTargetを露出し、Environment / Plugin Lockとの接続をTargetから追跡できるようにした。

このReferenceで確認したのはselectionとidentityの境界までである。Web Browser向けWasm / JavaScript implementationのactual build、load、sandbox、size / startup / memory測定は未実施であり、RQ-024のEvidence tierは上げない。


## E5 既存Runtime選択Authorityとの接続 — 2026-10-07

既存`contract-query-executor`には、Environment / Database instance / Contract Query targetごとの`ContractQueryExecutorBinding`がすでにProvider選択のAuthorityとして存在する。この選択へgeneric Capabilityの`Wiring`を重ねると、同じ実行先を二つの仕組みが選ぶことになる。

Referenceでは、productionの`ResolveContractQueryExecutor`が先に選んだPlugin / Capabilityだけをgeneric Contractへ投影し、Conformanceを確認する経路を追加した。この経路のCapability Resolution Lockはgeneric`Wiring`を持たない。generic ContractはProviderを選ぶのではなく、既存Authorityが選んだProviderが共通callable contractを満たすかを検証する。

`RuntimeCapabilitySelectionBinding`は、Environment、Database instance、Contract Query target、executor Capability、Plugin identity、Capability Resolution Lock、Runtime Capability Bindingを結び付けるderived artifactである。current Environment / Plugin set / Contract Query Bindingからproduction resolverを再実行し、同じ選択を再現できる場合だけcurrentと扱う。

Bindingが削除された場合、同じtargetにBindingが重複した場合、別executorへ変更された場合はreplayを拒否する。OracleとPostgreSQLが同じgeneric Contractを満たしても、それぞれの既存target-specific Bindingに従って別Providerを利用できる。

この結果は、Capability Contract共通化のために既存の細粒度なRuntime selection Authorityを捨てる必要がないことを示す。一方、generic Capability自体が選択Authorityを持つscopeでは、従来どおり複数Providerを暗黙選択せず明示`Wiring`を要求する。


## E6 Conformance issuer Authority pressure — 2026-10-07

Conformance Evidenceの存在だけをresolverの信頼根拠にすると、Provider自身が任意のEvidence fingerprintを作り、自己申告をConformance済みとして見せられる。このため、Evidenceの内容identityと発行Authorityを分離するReference candidateを追加した。

`CapabilityConformanceAuthorityGrant`は、exact Contract fingerprint、Conformance profile、issuer、Accepted Authority Evidence fingerprintを結び付ける。`CapabilityConformanceAttestation`は、current Conformance Evidence fingerprintをissuerとAuthority Evidenceへ結び付ける。

検証経路は次のように分ける。

```text
exact Contract / Provider replay
  -> current Conformance Evidence
  -> issuer attestation
  -> Accepted Conformance Authority Grant
```

Providerと同じIDをissuerにする場合も、Authority Grantが無ければ自己発行Evidenceは通らない。反対に、明示的なGrantがあればself-conformanceを表現できる。この差により「Providerだから信頼する」と「このissuerをこのContract/profileについて信頼する」を分離する。

Authority Evidenceの更新はCapability Resolution Lock identityへ含めない。Contract / Providerが同じならresolution identityは維持し、Authority binding fingerprintだけを更新する。

このcandidateはpublisher trustの最終実装ではない。Authority Evidence fingerprintそのものの真正性、署名、組織trust root、key rotation、失効は未実装である。production化では既存artifact signature / provenance境界と接続できるかを確認し、fingerprintの存在だけをtrustへ読み替えない。

また、実Source Adapter consumerについてはPR #968のSource Adapter contractがnamespaced lookalikeをUnresolvedに残す一方、現在mainのimportable `internal/xeadsource.Parse`はlocal nameだけでknown判定するため、そのままConformance済みProviderとして登録できないことを確認した。#963側で別readerを複製せず、#968のsource-faithful implementationがimportableなhost surfaceへ接続できる段階でE2E consumerを成立させる。

## E7 Build-backed physical realization pressure — 2026-10-07

E2では、現行builtin Pluginにexact artifact digestが無いため、legacy Capabilityをgeneric callable Providerへそのまま昇格できないことを確認しました。E7では、この不足をPlugin source definitionへ仮digestを追加して解消せず、ISSUE-000079で既に持つBuildArtifactLockへ接続します。

`ProviderBuildRealization` candidateは、logical Provider、exact BuildPlan、BuildArtifactLock内のruntime buildを結びます。Plugin / version / implementation ID、Source declaration fingerprint、Target、Build profile、BuildPlan fingerprint、artifact digestを保持します。

このbindingはBuild provenanceからphysical artifactを特定するためのderived artifactです。ConformanceやRuntime Authorityは付与しません。BuildArtifactLockからartifactを得たProviderでも、exact Target / artifact / Source declarationへ結び付いたConformance Evidenceが無ければgeneric resolverは採用しません。

同じartifact digestをOracleとPostgreSQL等の別logical Providerへ結び付けるcoalescing pressureも追加しました。physical artifactを共有しても、Plugin / implementation identityやProvider selectionは統合しません。複数Providerが同じContractを満たす場合は引き続き曖昧として扱います。

反対方向のpressureとして、一つのPluginを同じTargetで複数Capability artifactへ分割する構成もBuildArtifactLockでは表現できます。別Build profileから異なるartifact digestをProviderへrealizeできます。しかし、現行Runtime Plugin LockはPlugin entryごとにartifact digestを一つしか持ちません。片方のartifactをRuntime Authorityへ固定すると、もう片方のCapability artifactは同じPlugin entryからはauthorizeできません。

したがって、E7で成立したのはBuild provenanceとProvider artifact identityの接続、複数Providerのartifact coalescing候補、split packaging Gapの再現までです。Provider-levelまたはTarget packaging-levelのRuntime Authority表現はまだ未設計です。Plugin Lockへfieldを追加する判断は、このGapだけを理由に先行させません。

BuildPlanだけが変わりartifact bytesが同じ場合も区別します。ProviderBuildRealizationのbinding fingerprintは変わりますが、realized Providerのartifact identityは変えません。Build provenanceとcallable implementation identityを別の更新軸として扱うためです。

legacy移行では、BuildPlanのPlugin definition fingerprintをauthoritative `plugin.Definition`へ照合します。Plugin identityだけが一致する古いBuildPlanを、変更後のCapability宣言へ後付けする経路は拒否します。

## E8 Target packaging plan pressure — 2026-10-07

E7でlogical Providerとexact BuildPlan / BuildArtifactLockを結び付けたことで、Target内のphysical配置をProvider identityから独立して表現できるようになりました。E8では、その対応を`TargetCapabilityPackagingPlan`としてderivedにまとめます。

PlanはRuntime Authorityではありません。Capability Resolution LockとProviderBuildRealizationを入力にして再生成し、artifact digestごとにProvider参照をまとめます。複数のlogical Providerが同じartifactを共有するcoalescingと、一つのPluginが複数artifactへ分かれるsplit packagingを同じ表現で扱います。

logical Provider数はPlugin / version / implementation IDで数えます。同じimplementationが複数のCapability Contractを満たしても、Contract参照数をProvider数へ読み替えません。physical artifact数、coalesced artifact数、split Plugin数も内容から再計算し、自己申告のcountを信用しません。

同じlogical Providerを同じTargetで複数artifactへ割り当てる状態は拒否します。TargetやAuthority scopeが異なるrealizationも一つのPlanへ混ぜません。physical配置がProvider selection Authorityになることを避けるためです。

Planのfreshnessはfingerprintの自己整合だけでは判定しません。現在のCapability Resolution LockとProviderBuildRealizationからPlanを再構築します。artifact bytesが同じでもBuildPlanが変わればProviderBuildRealizationのfingerprintが変わるため、保存済みPlanはstaleになります。一方、Capability Resolution Lockは変わりません。Build provenanceとcallable Provider identityを別の更新軸として保つためです。

E8で成立したのは、Target packagingの機械可読なderived表現と、coalescing / splitの構造的な計測までです。actual Wasm / JavaScript artifactの生成、lazy load、byte deduplication、startup / memory / transfer測定、split artifactへのRuntime Authorityは未実証です。RQ-024のEvidence tierは上げません。

## E9 Execution Authority separation — 2026-10-07

Capability Resolution Lockが成立しても、そのProviderを実行してよいとは限りません。Referenceでは`CapabilityExecutionProfile`を別artifactとして追加し、Authority scope、realization Target、isolation policy、Permission集合を固定します。

Execution ProfileをProviderへbindするとresolution identityは変わります。一方、実行許可は別の`CapabilityExecutionGrant`で与え、`CapabilityExecutionAuthorityBinding`へ固定します。GrantはCapability Lock、Execution Profile、exact Permission集合、Authority Evidence fingerprintを参照します。

Permission不足だけでなく、Profileに無いPermissionの追加も拒否します。zero-Permission profileへambient権限を自動付与しません。host-trustedとhost-enforced-sandboxも別のisolation policyとして扱います。

この境界によって、Contract適合と実行権限を分離します。Conformance EvidenceがあってもExecution Authorityは得られず、Execution Authority Evidenceを更新してもCapability Resolution Lockのidentityは変えません。

E9はReference内のAuthority modelです。production hostのPermission Grant、Resource Broker、Lease、Sandbox enforcementへ接続したことまでは意味しません。

## E10 production Runtime Authority bridge — 2026-10-07

Application Runtimeのlegacy Capability移行では、generic Execution Profileを第二のPermission Authorityにしません。productionの`EnvironmentPlan`が既に選んだPlugin runtimeとCapability runtimeへ照合する`ProductionExecutionBridge` candidateを追加しました。

bridgeは、Capability Resolution Lock、RuntimeCapabilityBinding、Execution Profile、Execution Authority Binding、production EnvironmentPlan、legacy Capability identityを同時に照合します。

特に次をfail closedで確認します。

- production CapabilityのPlugin identityがlocked Providerと一致する
- legacy Capability declaration fingerprintがlocked Source declarationと一致する
- production側で実際にgrantされたPermission集合がExecution Profileとexactに一致する
- host-trusted profileはin-process / host-trusted runtimeへだけ接続する
- host-enforced-sandbox profileはprocess / external-process / sandbox-requiredへだけ接続する
- Runtime Capability Bindingのdescriptor / artifactとproduction Plugin runtimeが一致する

Resource BindingやLease Policyはgeneric Execution Profileへ移しません。productionの`CapabilityRuntime`に残し、そのfingerprintをbridge identityへ含めます。そのため、Resource / Lease policyが変われば保存済みbridgeはstaleになります。無関係なPlugin runtimeがEnvironmentPlanへ追加されても、対象Providerと無関係ならbridge identityは変えません。

`ProductionExecutionBridge`自身は`derived-verification-not-runtime-authority`です。Provider選択、Permission Grant、Resource取得、Host Sandbox起動のいずれも行いません。既存production Authorityとgeneric contractの整合を検証するだけです。

これにより、generic Capability Contractへ移行するために既存runtime Permission / Resource / Sandbox機構を捨てる必要がないことをReference上で示せます。ただしactual `HostSandboxLauncher`やResource Brokerをgeneric Capabilityから起動するE2Eは未実施です。production enforcement成立とは扱いません。

## E11 existing Host Sandbox invocation pressure — 2026-10-07

E10でgeneric Execution Profileとproduction Runtime Authorityの整合を確認した後、実行専用APIを追加せず、既存`Registry.InvokeWithProvenance`へlegacy Capability identityで委譲するReference orchestrationを追加しました。

orchestrationは保存済み`ProductionExecutionBridge`をcurrent Capability Lock / Runtime Binding / Execution Authority / EnvironmentPlanへreplayしてからだけproduction Registryを呼びます。bridgeがstaleならPlugin codeへ到達しません。

process / sandbox-requiredのzero-Permission fixtureでは、既存`HostSandboxLauncher`へ実際に到達し、artifact digest verification、deny-by-default Network policy、deny-by-default Filesystem policy、Sandboxed process runnerを通すpressureを追加しました。Network / Filesystem enforcerを供給しない場合は既存HostSandboxLauncherが実行を拒否します。

この結果は、generic Capability Contractの導入に合わせて別のprocess launcherやsandbox bypassを作る必要がないことを示します。Generic側は契約・identity・Authorityの整合を確認し、実行は既存production runtimeへ渡します。

E11で実行したpressureはzero-Permissionです。Resource Broker、Resource Binding、Lease、Secret injection、renew / refresh / fencingまでgeneric Capability経由で実行したEvidenceはまだありません。またRepository exact-head test / health gateも未取得です。
