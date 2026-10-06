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

## E1 Resolution Lock candidate — 2026-10-06

E0のresolverだけでは、同じRequirementを後日もう一度解決したとき、Catalogへ追加された新しいContract versionやProviderへ自動的に切り替わります。Source Adapter等のTool Capabilityでは、同じSourceを扱っていても実装が変わり得るため、AIやToolが長期間にわたって同じ処理へ再入場する条件として不十分でした。

ReferenceにCapability Resolution Lock candidateを追加しました。Lockはresolution policy、Authority scope、root Requirement、明示wiring、exact Contract / fingerprint、Provider identity、artifact digestを固定します。Lock自身にもfingerprintを持たせます。

fresh resolutionとLock replayは別の操作として扱います。Catalogへcompatibleな新versionが追加された場合、fresh resolutionは新しい完全解を選べます。一方、既存Lockのreplayは記録済みのexact Contract / Providerが現在も同じ内容で存在することを確認し、新versionへ自動更新しません。

Conformance EvidenceもLockから分離しました。Evidence更新だけではresolution identityを変えません。current Evidenceが必要な処理では、Lock replayの後に別の検査を行います。これにより、exact Contract / Provider replay、current Conformance Evidence、Runtime / Tool execution Authorityを区別します。

敵対的な確認では、Conformance EvidenceがProvider artifactへ結び付いていない問題も見つかりました。同じPlugin / version / implementation IDのartifactだけを差し替えると、古いEvidenceを再利用できるためです。ReferenceではEvidenceへexact artifact digestを追加し、artifact差し替え時は新しいConformance Evidenceを要求します。

また、同じPlugin / version / implementation IDを異なるartifact digestで複数宣言すると、明示wiringだけではartifactを一意に指定できません。この状態はload順依存を生むため、logical provider identityの重複として拒否します。

現時点ではReference candidateであり、productionのPlugin Lockや`internal/plugin.CapabilityKind`は変更していません。focused testとfull Repository healthを通した後も、Reference結果からproduction schemaを直接固定せず、既存Capabilityの移行pressureと合わせてADR候補を判断します。
