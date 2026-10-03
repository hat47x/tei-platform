# ISSUE-000077: Semantic ExtensionをCore変更なしで追加できる最小Referenceを実証する

- 状態: Ready
- 起点: RQ-031 / RQ-032 / RQ-033、PR #919
- 関連: ISSUE-000040 / ISSUE-000047 / ISSUE-000052
- Product value: 固定モデルではなく、後から意味体系を安全に追加・合成できること
- 主なReference pressure: SUI hierarchy adoption

## 目的

TEIのCoreへ新しい業務概念やValidation enumを追加しなくても、Plugin / DSL側から新しい意味論を宣言し、既存のCanonical meaningと関係付け、Tool / AIが発見・検証・再利用できることを最小Referenceで実証します。

このIssueでいうSemantic Extensionは、現在のprocess Pluginと同じものではありません。

現在のPluginは、`docs/plugins.md`が定めるとおり **Capability実装を配布する単位**です。Runtime、Permission、Artifact、Plugin Lock等の実行境界を持ちます。

一方、本Issueで追加したいものは、次のような**意味の宣言**です。

```text
Semantic Extension
  what concept / rule exists
  where it applies
  what parameters / dependencies it has
  what conflicts make it unresolved
  what deterministic validation is required
  how Tool / AI discovers it

Evaluator / Runtime Plugin
  how a declared rule is evaluated or realized
```

両者は配布上同梱できても、identityとAuthorityを同一にしません。

## 防ぎたい失敗

RQ-031〜033を実装するとき、次を防ぎます。

1. 新しい意味概念が必要になるたびにCoreのGo enum / parser / IR kindを増やす。
2. Runtime Pluginがコードを追加しただけで、その意味がCanonical authorityになったことにする。
3. Plugin package名やload順をsemantic identityや優先順位として使う。
4. 同じnamespaceへ互換性のない意味定義が来たとき、後勝ちで黙って上書きする。
5. 意味宣言が存在するだけで、Runtime execution / Database / Network等のAuthorityまで得る。
6. evaluatorが無い、versionが合わない、依存が欠けている状態を「Validation成功」とみなす。
7. AIが未知のextensionを無視して、Core内蔵語彙だけで近似実装する。
8. 一つのSUI固有gapだけを根拠に、`std.*`の標準語彙へ昇格する。

## 最初のReference pressure — SUI hierarchy

`ISSUE-000052`では、SUIの`IslandHierarchyContractV1`をTEIへread-only mappingした結果、現行Coreだけでは次をlosslessに表せないことが確認されています。

- nullable parent、すなわち0..1 presence
- parent targetの存在
- parent-child整合
- cycle禁止

現状はこれらをReference-local shadow validationとして扱っています。

このgapを、Coreに`acyclic`等の新しいValidationKindを直接追加せず、Semantic Extensionで表現できるかを最初のpressureにします。

## 最初に標準化しない

最初のextension identityは`std.*`にしません。

概念上はReference-localなidentityを使います。

例:

```text
reference.hierarchy.integrity
```

この名称自体を最終仕様として固定しません。

SUI以外の第二Referenceでも同じ意味が必要であり、同じ契約で再利用できることを確認した後で、標準namespaceへ昇格するかを判断します。

## Semantic Extensionの最小契約

最終syntaxを先に固定しません。まずmachine-readable contractとして、少なくとも次を表現できることを要求します。

### Identity

- extension logical identity
- namespace
- semantic version
- source / package provenance

### Applicability

- どの既存subject / relation / definitionへ適用できるか
- 適用に必要な既存semantic identity
- parameter schema
- 必須dependency extension
- compatible version range

### Meaning

- extensionが導入するsymbol / constraint / relationの意味
- parameterの意味
- 成立条件
- unknown / unsupportedの扱い
- 他extensionとのconflict key

### Validation / Conformance

- declarationを静的に検証する規則
- instance / definitionへ適用した結果を検証する規則
- evaluatorが必要な場合、その**Semantic evaluator requirement**
- evaluator不在時にfail-openしないこと

### Tool / AI discovery

- Source Index / Knowledge Contextからextension identityへ辿れる
- 適用対象から「利用可能なextension」を列挙できる
- extensionからSource / version / dependencies / Validationへ戻れる
- AIがcustom implementationより前にextensionを候補として発見できる

## 意味と実装を分ける

Semantic Extensionそのものへprocess実行Authorityを持たせません。

```text
semantic declaration
  reference.hierarchy.integrity
        ↓ requires evaluator capability if needed
semantic evaluator requirement
        ↓
Capability resolution
        ↓
Runtime Plugin
        ↓
existing Permission / Lock / Sandbox / Lease boundaries
```

extensionを読み込めたことは、evaluator processを実行してよいことを意味しません。

逆に、evaluator Pluginがinstallされていても、そのPluginがextensionの意味を勝手に変更できません。

同一Semantic Extensionを複数の実装Pluginが評価できる構造を許容し、Environmentが既存Capability negotiationで実装を選べる方向を優先します。

## 最初のReference declaration

SUI hierarchyの最初の宣言では、少なくとも次を分けます。

### H1 — optional parent presence

parent relationは各childについて0または1件である。

これは「single-valued relation」と「parentが無くてもよい」を分けて表現します。

### H2 — parent target existence

parent referenceがある場合、そのtarget identityが存在する。

### H3 — acyclic

parent relationを辿ってcycleを形成してはならない。

### H4 — parent / child consistency

source contractがparentとchildrenの両方向を持つ場合、両者が矛盾してはならない。

最初からこの四つを一つの`hierarchy`booleanへ畳みません。

第二Referenceで個別再利用できる可能性を残します。

## E0 — declaration / discoveryだけを先に実証する

最初のコードincrementは、SUI instance validationの実行まで一気に進めません。

まず次を実証します。

1. extension declarationをSourceからloadできる。
2. logical identity / version / dependencyがSource Indexへ入る。
3. SUI hierarchy definitionから適用extensionへexactに辿れる。
4. Knowledge Contextからextension Source / version / applicabilityへ戻れる。
5. extensionが無い状態をCore標準Validationへ暗黙変換しない。
6. incompatible duplicate definitionをload順で上書きせずfail closedまたはUnresolvedにする。
7. extensionを削除しても無関係なCanonical subjectsの意味は変わらない。

E0ではRuntime Pluginを起動しません。

## E1 — deterministic evaluator boundary

E0成立後、H1〜H4のうちCore declarative primitiveだけでは評価できないものについて、deterministic evaluator boundaryを追加します。

Evaluatorは次を守ります。

- inputはexplicit subject / extension identity / accepted definition snapshotへboundする
- outputはpass / fail / unknownとdiagnostic Evidenceを返す
- evaluator outputをCanonical meaningそのものへしない
- evaluator不在 / timeout / unsupported versionをpassへ変換しない
- Runtime Pluginの場合は既存Permission / Sandbox / Artifact / Lockを迂回しない

## E2 — SUI fixtureでpressureする

既存`reference/sui-adoption/` fixtureを再利用します。

最低限:

- M1 root — pass
- M2 3-level hierarchy — pass
- M3 missing parent — fail
- M4 cycle — fail

新しいextensionを評価するためにSUI source contractを書き換えません。

TEI側のmapping / extension declaration / evaluatorで結果を説明し、SUIのnative authorityを維持します。

## E3 — second consumerで標準化をpressureする

SUIだけで成立しても、Core標準語彙へ昇格しません。

第二Referenceでは、SUIとは異なるdomainで同じconstraintの少なくとも一部を利用します。

候補:

- 組織 / category tree
- BOM / dependency hierarchy
- nested configuration
- Business Procedure間の明示的な階層構造

第二Referenceは、単に同じJSON shapeを作るためのsynthetic fixtureではなく、独立した意味上の必要性を持つものを選びます。

確認すること:

- 同一extension identityを変更せず再利用できるか
- domain-specific parameterを追加しすぎていないか
- SUI固有語彙が漏れていないか
- Core primitiveへ昇格すべき共通性が本当にあるか
- extensionのまま保持した方が小さいCoreを保てるか

## Namespace / conflict rule

namespaceは意味Authorityの境界です。

最初のReferenceでは少なくとも次をfail closedにします。

- same extension identity + incompatible semantic version
- same symbolを互換性なしで複数extensionが定義
- required dependency version不一致
- dependency cycle
- unresolved implementation/evaluator requirement

競合をPlugin load orderで解決しません。

明示的なcompatibility / precedence contractが無い場合はUnresolvedとします。

## Versioning

Semantic versionはRuntime binary versionと独立させます。

理由:

- evaluator implementationだけ更新して意味契約が変わらない場合がある
- 同じSemantic Extensionを複数Runtime実装が提供できる
- 意味互換性と実装互換性を同じversionへ畳むと、不要なCanonical invalidationを起こす

Source / Decision / qualification Evidenceは、必要に応じてSemantic Extension identity + semantic version + declaration fingerprintへboundします。

## Shared Acceptanceとの関係

Semantic Extensionを追加・更新すること自体も意味変更です。

AI、CLI、Studio、Plugin installerのどこから来ても、採用済みextension declarationを黙って上書きしません。

```text
extension candidate
  ↓
Preview
  semantic symbols / applicability / conflicts / affected subjects
  ↓
Accept / Reject
  ↓ Accept only
authoritative extension source
  ↓
reload / Source Index
  ↓
Conformance
```

Runtime Pluginのinstall / activationは別Authorityです。

```text
accept semantic extension
  !=
allow process / network / database execution
```

## Knowledge-first / AI railとの関係

RQ-033のProduct valueを最初から検証します。

AI / Toolが新しい実装案を作る前に、対象subjectのKnowledge Contextから次を取得できることを目指します。

- applicable Semantic Extensions
- required parameters
- required dependencies
- deterministic Validation
- evaluator requirement
- unresolved conflict
- related Capability

AIがextensionを発見した後も、自動採用はしません。

`applicable`は「使える候補」であり、AcceptanceでもRuntime Authorityでもありません。ISSUE-000047と同じ境界を使います。

## Product Evidence

このReferenceで測る価値は、「新しい仕組みを実装できたか」だけではありません。

### Core growth

- 新しい意味論を追加するために変更したCore Go enum数
- parser / IR / validatorのCore固有分岐数
- 新しいApplicationごとに追加したCore code量

最初の目標は、SUI hierarchy固有の概念をCore enumへ追加しないことです。

### Reuse / change amplification

- extension一つの変更で影響するSource / Tool / Target数
- 同じconstraintの重複記述数
- 第二Referenceで再利用するための変更量
- extensionなしcustom implementationとのhand-written差分

### AI rail

- AIが既存extensionを発見できた割合
- custom implementationを作る前に既存extensionへ到達できたか
- required dependency / unresolvedを見落とした件数
- extensionを無視して近似実装した件数

### Safety

- incompatible extensionのsilent override = 0
- semantic declarationからRuntime Authorityへの暗黙昇格 = 0
- evaluator不在をpass扱い = 0
- unrelated Canonical meaningの変化 = 0

Safety違反を生産性指標と相殺しません。

## 最初のcode work unit候補

E0の最初のcode work unitは、一つのReference declarationとread-only loader / index pressureに限定します。

まだ最終syntaxは決めません。

候補構成:

```text
reference/workspace/extensions/<reference-extension>/<version>/...
internal/semanticextension/   # identity / declaration / validationのみ
internal/sourceindex/         # exact source references
internal/knowledgecontext/    # read-only discovery
```

ただし、このdirectory名やpackage名は実装前に既存syntax / Source Indexの拡張方法を再監査して決めます。

最初のincrementでは次をしません。

- Runtime process起動
- new Plugin CapabilityKind追加
- SUI dataのCanonical化
- std namespaceへの昇格
- AIによる自動選択

## Acceptance — E0

- [ ] CoreにSUI固有ValidationKindを追加せずSemantic Extensionを宣言できる
- [ ] extension logical identity / semantic version / Sourceへ戻れる
- [ ] applicabilityをexact existing subjectへ結び付けられる
- [ ] Source Indexからextension declarationを発見できる
- [ ] Knowledge Contextから適用候補をread-only取得できる
- [ ] duplicate / incompatible definitionをsilent overrideしない
- [ ] dependency不一致をUnresolvedまたはblocking diagnosticにする
- [ ] extension不在をCore標準意味へ推測変換しない
- [ ] Semantic ExtensionとRuntime Plugin identityを分ける
- [ ] extension declarationだけではRuntime Authorityを得ない
- [ ] accepted extension sourceをToolなしでreloadできる
- [ ] unrelated Canonical subjectsを変えない

## 次へ進む条件

E0がexact-head full`tools/repository-health-check.sh`を通ってmainへ入った後、E1 evaluator、E2 SUI fixture、E3 second consumerの順に進めます。

E2が成立しても第二consumer前に`std.*`へ昇格しません。

第二consumerで大きなdomain-specific変更が必要なら、共通extension化の仮説を棄却または分割します。

## 実装Gate

code変更はTEIの通常Gateに従います。

- latest main / open PR / active laneを再確認
- code PRはDraft
- exact-head full`tools/repository-health-check.sh` PASS
- clean pre/post state
- Gate EvidenceをPRへ記録
- PASSなしでmainへ統合しない

## 非目標

- Plugin system全体の作り直し
- Runtime PluginをSemantic authorityにすること
- 一つの万能Extension DSL
- arbitrary codeをmeaningとして登録すること
- SUI domain objectをTEI Core kindへ追加すること
- SUIのshadow validationをそのまま標準語彙へコピーすること
- extensionをAIが自動採用すること
- load orderでconflictを解決すること
- Semantic ExtensionとPermission / Runtime Authorityの統合
- 一つのReference成功からEmacs相当の汎用性を実証済みと主張すること

## 関連

- PR #919
- RQ-031 / RQ-032 / RQ-033
- `product/value/TEI_PRODUCT_VALUE.md`
- `product/principles/TEI_PRODUCT_PRINCIPLES.md`
- `docs/plugins.md`
- `plan/issue/000040-semantic-component-composition.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
- `plan/issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md`
