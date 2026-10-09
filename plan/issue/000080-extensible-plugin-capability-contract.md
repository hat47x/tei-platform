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
- logical contractとphysical artifact / processの境界を分離できる
- process非対応Targetでも同じcontractを別realizationへ載せられる構造を説明できる
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

Referenceでは次のnegative controlを置いた。

- Conformance Evidenceが無いproviderを拒否する
- `application-runtime` scopeのproviderを`authoring-tool`要求のfallbackに使わない
- 複数の適合providerがある場合、明示wiringなしで選ばない
- contract fingerprintが変わったproviderを拒否する
- Evidenceだけを更新してもcontract / provider resolution identityは変えず、Evidence fingerprintだけを変える

さらに敵対評価で、最高compatible contractを先に固定すると、そのversionにproviderが無いだけでlower compatible versionの有効解を失うことが分かった。Reference resolverはcontract versionとprovider / Conformanceの組を高いversionから探索し、完全な解があるlower versionへ退避できるcandidateへ修正した。

明示wiringも`ImplementationID`だけでは不足する。同名implementationを別Pluginが提供できるため、Plugin ID / Plugin version / Implementation IDの組で選択する。

現時点ではReference candidateであり、productionの`internal/plugin.CapabilityKind`は変更していない。exact-headのrepository health checkは未実施であり、PRはDraftとして扱う。

## E0 Reference candidate追補 — Authority逆方向とphysical artifact境界（2026-10-08）

完了条件の「Runtime scopeとのAuthority混同をnegative testで拒否できる」「logical contractとphysical artifact / processの境界を分離できる」について、`tools/reference-plugin-capability-contract/`に次を追加した。いずれもReference candidateの範囲で、production型（`internal/plugin.CapabilityKind`）は変更していない。

- **逆方向のnegative control。** `authoring-tool` scopeのproviderを、`application-runtime`要求（Contract Query Executor相当のcontract）のfallbackに使わない。先に置いた「runtime providerをauthoring-tool要求に使わない」testと対になり、どちらの向きの混同も拒否する。
- **Conformance Evidenceをphysical artifactに結び付ける。** 従来はplugin / version / implementationの組だけでEvidenceを照合していたため、同じ組で別のartifact（たとえばbuiltinとWasm）を出すと、片方のEvidenceをもう片方が引き継げた。Evidenceに`ArtifactDigest`を持たせ、providerのartifactと一致する場合だけ有効とした。
- **明示wiringのartifact曖昧性。** wiringがplugin / version / implementationまでしか指定しないとき、複数のartifactが一致すると、digestの並び順で先頭が選ばれていた。これは暗黙選択なので、一致が複数なら曖昧として拒否し、`ArtifactDigest`を指定したときだけ選ぶようにした。
- logical contractのfingerprintはartifactに依存せず、resolution identityは選ばれたartifactを記録する。同じcontractを別のrealizationへ載せても意味は変わらず、どのrealizationを選んだかは逆追跡できる。

まだ残る点: Xead / Xeaf Source Adapter本体のReference、既存`CapabilityKind`の移行ADR候補、process非対応Targetでの別realizationの説明。検証: この変更を含む作業tree（commit前、内容は同一）を非root userのコピー上で`tools/repository-health-check.sh`のfullを実行し、gofmt / vet / `go test` / validateがPASSした。root実行では`tools/reference-shared-acceptance`の`TestAcceptDoesNotDamageAnExistingDerivedArtifactWhenItCannotBeWritten`がFAILする。chmod 0555がroot権限では書込を止めないためで、この変更前の`main`でも同じく再現する環境起因であり、Repositoryの欠陥とは判断していない。ただし、これはexact-headのtrusted local PASS receiptではない。Gate authorityとしては使わない。


## E1 実Source Adapter consumer candidate — 2026-10-09

E0では`authoring.source-adapter.xead`をCore固定enumへ追加せず解決できることをpressureしたが、解決結果から実readerを呼ぶconsumerは存在しなかった。resolverだけが成立しても、実際のToolが別の特別経路でSource Adapterを起動するなら、Plugin-firstの価値は十分に確認できない。

このため、`tools/reference-plugin-capability-contract`からimport可能な`internal/xeadsource`を呼ぶReference consumer candidateを追加した。

consumerは外部から完成済みのResolutionを受け取らない。`ResolveInput`を毎回resolverへ通し、Conformance Evidence、Authority scope、明示wiringを検証した後、解決したexact Provider artifactをRegistryから取得する。解決は成功しても該当artifactがRegistryに無い場合は停止する。

Xead / Xeafのfixtureでは、次をpressureする。

- exact source bytesとinspection digestが一致する
- source bytesが変わればsource artifact identityも変わる
- 未知constructの属性・本文を落とさず、candidateをunresolvedのまま返す
- Conformance Evidenceが無ければreaderを実行しない
- `application-runtime` scopeからSource Adapterを実行しない
- 解決済みでも未登録artifactへfallbackしない
- Repository正本の転写契約から現在のXead `System / FormatVersion 1.2` profileと`Table`分類を適用できる

敵対的に確認したところ、import可能な`internal/xeadsource.InspectWithContract`は転写契約の`sourceProfiles`を確認していなかった。このままconsumerへ接続すると、RMT側で既に停止条件とした「未対応versionへ既知mappingを適用する」経路を再導入する。

candidateでは`InspectWithContract`へsource profile Gateを追加した。転写契約にProfileが無い場合、root kindが違う場合、`FormatVersion`が無い・複数・未対応の場合はclassificationへ進めない。reader側でも同じexpanded nameの属性重複と、root外の非空白textを拒否する。

この変更でRMT-2全体がCapability化されたとは扱わない。`internal/xeadsource`はimport可能なinspection APIであり、`tools/reference-xead-source-adapter`はPublication Receiptやpublic-safe redactionを含む別のReference surfaceである。今回確認するのはgeneric Capability resolutionから実readerへ到達する経路までで、RMT-2の公開Evidence contractやqualificationを代替しない。

長期的には二つのreaderが別々のXML解釈を持ち続けるとdriftの原因になる。共通化するsource observation kernelと、RMT固有のpublic Evidence projectionを分離できるかを後続pressureで確認する。

現headのfocused testとfull`tools/repository-health-check.sh`は未取得である。コード変更を含むためDraftのまま保持し、exact-head full Gate PASS前にmainへ統合しない。


### E1から見えた次のGap — 汎用的な実行呼出し

実際のSource Adapter利用経路を通したことで、Capabilityを汎用的に解決できることと、解決した実装を汎用的に呼び出せることは別の問題だと分かった。

現在のReferenceでは、Capability ContractとProviderの解決自体はCore固定enumへ依存しない。一方、実行側の`SourceAdapterRegistry`は`SourceAdapterRequest`と`xeadsource.Inspection`をGo型として知っている。新しいCapability Contractを追加するたびに、同じ方法でContract専用Registryやrequest / result型をHostへ書き足す必要があるなら、Core enumをなくしても拡張点が実行呼出しへ移るだけである。

したがって、ISSUE-000080の次段では、少なくとも次の責務を分けて検証する必要がある。

- Contractが要求するrequest / resultの機械可読な形
- exact Provider implementationを呼び出す共通境界
- in-process / process / Wasm等の物理的な実行方式
- request / resultのcodecやschema identity
- Permission / Resource / Target Authority

この段階でJSONやRPCをproduction標準として固定しない。Source Adapter以外のCapability consumerでも同じ呼出しprimitiveを再利用できるか確認したうえで、共通Contractへ昇格するかを判断する。

今回のPRでは、このGapを解消済みとは扱わない。実consumerを追加した目的は、resolverだけでは見えなかった次の固定点を具体化することである。


## E2 汎用Invocation candidate — 2026-10-09

E1で実Source Adapterを呼び出した結果、Capability Contractの解決を一般化するだけでは不十分だと分かった。実行側がCapabilityごとに専用Registryと専用dispatchを追加する構造では、Core固定enumを減らしても、新しい固定点がHostの呼出しcodeへ移る。

このため、Referenceへ`InvocationRegistry`を追加した。Registryはdomain型を扱わず、resolverが選んだexact Provider、request / result shape identity、shape fingerprint、payload codec identity、opaque payloadだけを扱う。request / result bytesのdigestも実行結果へ残す。

既存のSource Adapter consumerは、このRegistryを内部利用する形へ変更した。Provider選択、Conformance確認、exact artifact lookupをSource Adapter専用Registryへ重複実装しない。Source Adapter固有のsource digest確認やunknown保持はtyped wrapper側に残す。

第二consumerには既存の`driver.ContractQueryExecutor`を接続した。`authoring-tool`のSource Adapterと`application-runtime`のContract Queryを同じInvocation Registryへ登録し、それぞれ元のAuthority scopeを維持したまま呼び出す。共通Registryを使うことはAuthorityの統合を意味しない。

payload codecはContract scopedとした。ReferenceではSource AdapterとContract QueryにJSONを使うが、Registry自体はcodecを解釈しない。opaque binary payloadを往復するtestも追加し、JSONやRPCをproduction標準として固定しない。

敵対的に確認したところ、shape identityを文字列だけで照合すると、同じshape名のままwire schemaを変更できる。このためrequest / result shapeへexact fingerprintを追加し、ImplementationとContractで一致を要求した。ProviderのConformance EvidenceはContract fingerprintへ結び付くため、shape fingerprintの変更も再Conformanceが必要なContract変更として扱える。

ここで使うshape fingerprintは、exact schema artifactへ戻るためのidentity候補である。schema artifactの正本、発行Authority、schema validation engineは未実証であり、fingerprintだけを型検証の成立Evidenceにはしない。

現在のReference JSON adapterについては、未知fieldとtrailing JSON valueを拒否するexact decodeを追加した。shapeより緩いJSONをdomain実装へ流さないための局所的な検査であり、汎用schema validationが成立したとは扱わない。

このE2で、generic resolutionからgeneric invocationまでCore固定Capability enumへ依存しないReference経路ができた。ただしproduction Permission / Resource / Sandbox、Targetごとの実行方式、process / Wasm transport、schema artifact管理は未接続である。コード変更を含むため、exact-head focused testとfull Repository healthが得られるまでPRはDraftのまま維持する。


## E3 Invocation shape artifact candidate — 2026-10-09

E2ではrequest / result shapeにfingerprintを持たせたが、fingerprintが何を指すかは未定義だった。この状態では、同じshape名のschema driftを検出できても、schema本体の正本、発行者、検証方法へ戻れない。

Referenceでは、Source AdapterとContract Queryのrequest / result shapeを機械可読なJSON artifactとして追加した。artifactはshape ID、payload codec、shape format、issuer identity、構造定義を持つ。fingerprintは正規化したartifact全体から計算する。

Capability Contractには`shapeIssuer`を追加した。Implementationが提示するrequest / result artifactは、Contractのshape ID、fingerprint、payload codec、issuerと一致しなければ実行できない。Providerの自己申告だけではschemaを差し替えられない。

Invocation Registryはrequestを実装呼出し前に、resultを呼出し後に共通validatorで検査する。ReferenceのvalidatorはJSON object / array / map / scalarとopaque bytesを扱う。未知field、必須field欠落、型不一致、trailing JSON valueをfail closedにする。domain固有の値制約や業務意味は各adapterへ残す。

実shapeの正本は`tools/reference-plugin-capability-contract/shapes/`のJSONとした。Go側はこのJSONを埋め込んで読み込み、別のGo literalを正本にしない。shape artifactのissuerを変えた場合もfingerprintが変わる。

fingerprintの再計算規則もartifactへ`canonicalization`として明示した。Reference v0では`required`の順序を正規化し、source JSONの空白や記述順によってidentityが変わらないようにする。canonicalization revisionが変わった場合はshape fingerprintも変わり、未対応revisionは拒否する。

このincrementで、E2に残っていた「shape fingerprintが指すschema artifact」と「共通validation engine」はReference候補として具体化した。一方、issuer identityを信頼する根拠、署名、publisher trust、組織ごとの発行Authorityは未実証である。Referenceのshape languageもproduction標準へ固定しない。

また、Source Adapter result等の深い構造には明示的な`any`を残している。これはschemaが完全なdomain型を代替したという意味ではない。generic Invocation境界ではwire envelopeを検査し、domain adapterが深い意味と値制約を検査する二段構成としている。

コード変更を含むため、exact-headのfocused testとfull Repository healthが得られるまではDraftのままとし、Evidence tierも変更しない。


## E4 Side-effect boundary — 2026-10-09

E3でresult schema validationをInvocation Registryへ入れたことで、mutating Capabilityへそのまま一般化できない理由が具体化した。

requestは実装呼出し前に検査できる。一方、resultのschema違反は実装呼出し後にしか分からない。実装が外部へ書込みを行う場合、resultをrejectしても副作用だけが残り得る。Permission / Sandboxは実行可否の境界であり、この原子性問題を解決しない。

Referenceでは`Contract.invocationEffect`を追加し、generic Invocationが受け入れるのを`read-only`へ限定した。Source AdapterとContract Query、opaque copyのReferenceはread-onlyとして扱う。`mutating`を宣言したContractは、valid requestであってもdomain implementationを呼ぶ前に停止するnegative controlを置いた。

この制約はproductionでmutating Capabilityを禁止する決定ではない。書込みCapabilityを共通Invocationへ載せる前に、少なくともtransactionまたはprepare / commit、失敗時の補償、再試行、result validation失敗時のAuthorityを別pressureで定める必要があることを示す。

現時点ではReference candidateであり、production Permission / Resource / SandboxやDatabase transaction contractへは接続していない。Evidence tierは変更しない。
