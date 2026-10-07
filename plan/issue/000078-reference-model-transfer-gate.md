# ISSUE-000078: 成熟したReference ModelをTEIへ転写し、表現力と暗黙前提を実証する

- 状態: In Progress
- 提案日: 2026-10-06
- 性格: Product Value / Reference / Conformance / migration pressure
- 主担当要求: RQ-001 / 002 / 003 / 004 / 010 / 011 / 012 / 013 / 021 / 022 / 025 / 026 / 029 / 031 / 032 / 033
- Evidence contract: [Reference Model Transfer Evidence Gate](../REFERENCE_MODEL_TRANSFER_GATE_2026-10.md)

## 背景

TEIの現在のReferenceは、意味のSSOT、Application Driver、第一Web Target、Shared Acceptance、第二Application等の重要な境界を実行可能な形で確認してきた。一方、業務システムとしての機能量・関係性・実用性は、まだ小規模なReferenceの範囲にある。

TEIの価値を次の段階へ進めるには、既存の小さなReferenceを増やすだけでなく、長年の設計知識が入った成熟したReference Modelを一式で受け止められることを示す必要がある。

最初の対象として、渡辺幸三氏のX-TEA / XEAD系Reference Model「CONCEPTWARE／受注生産」を用いる。利用者が保持するmodel / manualと、渡辺氏本人のGitHubにあるXeadModeler / XeadEditor / XeadDriver sourceを照合する。

このIssueはX-TEA互換機能の実装を目的にしない。成熟した外部modelをTEIへ転写する過程で、TEIの表現力、Source traceability、native asset境界、Target independence、Business cross-view、Runtime authority、AI再入場性を同時にpressureする。

渡辺氏の設計思想をTEIへ翻訳する一般的なArchitecture整理は[Watanabe Business System Design Pressure](../design/watanabe-business-system-design-pressure.md)が担当する。本Issueは、その整理を実Reference Model全体でpressureし、転写できない意味や暗黙前提をEvidenceとして返す。

## 防ぎたい失敗

最も避けたいのは「主要な画面やTableが再現できたので移植できた」と判断し、sourceに含まれる意味や暗黙前提を落とすことである。

特に次を失敗とする。

- source constructを対応先不明のまま黙って捨てる
- X-TEA固有のIDやSortKeyをTEIのCanonical identityへそのまま昇格する
- Function / Table / Business Actionの名称から依存やCRUDを推測する
- native Scriptのactual effectを、宣言されたCRUDだけで説明済みとする
- Driver固有のtransaction、Session、credential、pathを業務意味として取り込む
- manual / external workをApplication Functionへ偽装する
- X-TEAのFunctionTypeやUI classをTEI Coreへ複製して互換性を作る
- model proseをAIが読めたことをAccepted Canonical meaningへ読み替える
- original Runtimeを実行していない挙動を「互換」と主張する
- Reference Modelの利用条件が未確認のままmodel contentsを公開Repositoryへ転載する

## このIssueで減らす不確実性

完了時には、少なくとも次をEvidenceから答えられるようにする。

> X-TEAで表現された成熟した業務システムを、TEIの意味境界を壊さず、どこまで転写・実行・投影できるか。転写できないもの、意味が曖昧なもの、旧Runtimeの暗黙前提に依存するものは何か。

さらに、X-TEAでは単一Desktop / Java Runtimeの前提で問題化しにくかった箇所を、TEIのmulti-Target / AI agent / Provider交換 / long-lived systemへ移した場合に何が不足するかを明らかにする。

## Source baseline

一次sourceは次とする。

~~~text
xead/XeadModeler
  68f1535b1c5dd80a327dfced37be401ecd3093c9

xead/XeadEditor
  ee19a27609b2749110ec39b62d4540693347fe09

xead/XeadDriver
  6c077773b74c6d7dbcc253835cf8d347e4ffb1ef

MfgOrder.xead
  sha256 31af1fe317c661f57c94c69285b7b3e91c7c2fcad940cd7639956aff781088a9

MfgOrder.xeaf
  sha256 f4b73b36abd8398aa25ad86ecd7570ae0334e7fd93c3ee204c5a05bddebdb754
~~~

model / manual本体は利用者Libraryのsource snapshotを利用し、初期EvidenceではRepositoryへコピーしない。公開範囲は[CONCEPTWARE Reference Model公開境界](../REFERENCE_MODEL_PUBLICATION_BOUNDARY_2026-10.md)に従う。

## 現在すでに分かっていること

### 三要素とcross-reference

Data、Function、Businessは同じ業務システムに対する別視点として実際に結ばれている。TaskActionからTaskFunctionIOを介してFunctionへ辿れ、FunctionのIOTableがC/R/U/Dを明示する。CRUD matrixはそこからderiveされる。

Reference ModelにはFunctionを持たないBusiness Actionも存在する。また、観測したmodel snapshotではSubsystem間のTable I/OにRがある一方、外部SubsystemへのC/U/Dは確認していない。

これはRQ-025 / RQ-026と強く対応するが、X-TEAのobject modelをそのままTEIへ採用する理由にはしない。

### design Functionとruntime Functionは1:1ではない

`MfgOrder.xead`には227 Function、`MfgOrder.xeaf`には203 Functionがある。観測したidentity集合では、`.xeaf`側だけに存在するFunctionが23件、`.xead`側だけに存在するFunctionが47件あり、一つの基本設計Functionが複数の実行Functionへ分解される例もある。基本設計FunctionとEditorで詳細化された実行Functionの間には補助Functionや基本設計only Functionがあり、単純なbijectionではない。

TEIでは、業務・interaction上のFunction meaningと、Target / Runtime固有Realizationを分けて扱う必要がある。


### Business ActionとUI操作

`TaskFunctionIO.Operations`には「更新ボタンを押す」「行を選んでエンターする」等の画面操作文が含まれる。これをBusiness Actionそのものとして転写すると、Target変更だけで業務意味が変わる。Business Action、利用Function / Capability、Target固有interaction instructionを分離するpressureを本Issueで所有する。

### 識別・関数従属性・定義域制約・選択キーを分ける

Xeadでは、TableのPK / FK / SK / XK、複数のTableKeyField、Relationshipが参照する両側のTableKeyが明示される。一方、XeadDriverではFunctionやDetailが独自の`KeyFields`を持つことができ、未指定の場合だけTableのPKへ戻る実装がある。

したがって、source上で「key」と呼ばれる情報を一種類のidentityへまとめない。

~~~text
legacy TableKey assertion
  != Relationship key mapping
  != Function / UI selection key
  != business occurrence identifier
  != Canonical identity
  != physical primary key
  != functional dependency
  != composite domain constraint
~~~

TableKeyの型、field membership、Relationshipのkey参照、Function / Detailの`KeyFields`はsource factとして保持できる。ただし、それだけを根拠にbusiness identifierや関数従属性へ昇格しない。複合項目についても、recordの識別だけでなく「許される値の組合せ」という定義域制約を持つ場合があるため、複合keyの形から意味を一意に決めない。

生成AIへ渡す場合も、source factとauthority limitを同時に提示する。根拠が足りないbusiness identifier、candidate key、functional dependency、domain constraintは、もっともらしい補完をせずUnresolvedのまま残す。

### 旧Runtimeに隠れやすい意味がある

Driver manual / sourceでは、Scriptからの任意SQL / TableOperator、caller contextに依存するcommit、return codeによるrollback、Login Script、共有Script、Session attribute、動的parameter map等を確認した。

これらは旧Driverでは実行規約として成立するが、TEIでTargetやagentを交換する場合は明示的なdependency / transaction / authority / effect境界が必要になる。

詳細なmodernization pressureはEvidence Gate `7を正本とし、本Issueへ重複列挙しない。

## 担当境界

このIssueが所有するのは、成熟したReference Modelを使った**転写の実証と、そこから見つかったTEI側のgapの切り分け**である。

既存Issueの責務を再実装しない。

| 見つかったgap | 引き渡し先 |
| --- | --- |
| Business Procedure / cross-view contract | 000042 / 000048 / 000049 |
| Lifecycle / state transition | 000038 |
| change impact / provenance | 000039 |
| reusable semantic component | 000040 |
| Studio / human projection | 000046 / 000071 |
| AI proposal / acceptance | 000027 |
| AI re-entry / operational knowledge | 000053 |
| Target差 / execution profile | 000026 |
| Semantic Extension / Plugin / DSL | 000077 |
| Command / Observation / retry / unknown | 000009 / RQ-018〜020の既存境界 |

Reference Modelで新しいgapが見つかっても、既存Issueが所有する責務なら本Issueの中に別実装を作らない。

## 作業順序

### Phase 1 — inventory / mapping ledger

source model全体のelement / attribute / relationをinventoryし、Evidence GateのClassificationへ割り当てる。

成果物候補:

~~~text
reference-model-transfer/
  source-baseline.json
  source-inventory.json
  transcription-contract-v0.json
  transcription-map.json
  loss-ledger.json
~~~

この段階ではTEI Core schemaを変えない。

### Phase 2 — cross-view transcription

Data / Function-Interaction / Businessについて、source locatorからTEI meaning / native asset / unresolvedへ戻れるmappingを作る。

Relational source factは、観測できたこととdeterministic loweringへ使えることを分ける。inspectionの`relationalResolutionStatus`が`unresolved`である間は、factをEvidenceとして保持してもSemantic loweringへは進めない。diagnosticを解消するか、後続の明示的なAcceptance境界で扱いを確定する。

既存`order_entry`で得たBusiness Procedure checkerやCRUD pressureを再利用し、Reference Modelの規模で壊れる箇所を探す。

### Phase 3 — modernization pressure

旧X-TEAでは暗黙でも成立した点を、TEIのRequirementへ照らして反証する。

優先するpressureは、identity、relationship mapping、declared/actual effect、transaction、Environment/secret、Script dependency、actor/auth、physical store、Instance/Scenario、external definition reuse、prose semantics、dynamic function contractである。

### Phase 4 — executable representative slice

source modelから、FunctionTypeや業務境界の異なる代表Scenarioを選び、TEI Driver / Targetへ転写する。

source business factを改変してTEIの既存Patternへ合わせない。既存Patternで表現できなければ、native preservation、Extension、Unresolvedの順で扱う。

### Phase 5 — full-reference accounting

部分sliceで固めたmappingをsource inventory全体へ適用し、silent dropと特定sliceへの過適合を監査する。

## 実装原則

Importerを作る場合も、最初から「Xead XML -> TEI Canonical」の一括変換器にはしない。

### Plugin-first

転写で新しく必要になった業務・設計概念は、原則としてSemantic Extension / Pluginで表す。X-TEAのFunctionType、Relationship、Business Procedure、lifecycle、CRUD lens、UI / report、Scenario等を、対応するsource要素があるという理由だけでCore enumへ追加しない。

標準機能として採用する概念にも、第三者Extensionと同じidentity、dependency、namespace authority、conflict、lock、discovery、Conformanceを適用する。X-TEA importerはcompatibility adapterであり、Canonical authorityやRuntime authorityを持たない。import結果はcandidateとしてShared Acceptanceへ渡す。

論理Extensionと物理artifactは分ける。declaration-only ExtensionにRuntime artifactを要求せず、Targetに応じて複数Extensionを一つのbuiltin / linked / portable artifactへまとめられることを要求する。Web Targetでnative processが使えないことを理由に意味を変えない。

~~~text
Xead / Xeaf source
      ↓ parse
Source-faithful intermediate inventory
      ↓ classify
mapping decision / unresolved
      ↓ accepted transcription
TEI authoritative source / native asset / binding
      ↓ reload
Canonical World
~~~

source-faithful intermediateはmigration Evidenceであり、TEIの第二Canonical modelにはしない。

source factの存在だけでlowering可能と判断しない。relational diagnosticが残る場合は`relationalResolutionStatus = unresolved`とし、deterministic semantic loweringを止める。fact countやcandidate classificationは、この停止条件を上書きしない。

変換不能項目をwarningだけで落とさない。mapping versionを変更した場合は同じsourceから得られるtranscription差分を比較できるようにする。

## Webチャット外担当

次はactual environmentまたは利用条件判断が必要であり、本Issueの文書研究だけで完了させない。

1. [GitHub #940](https://github.com/hat47x/tei-platform/issues/940)で、original X-TEA Modeler / Driverを起動し、代表Scenarioのactual behavior、native Scriptのtransaction / rollback / TableOperator effectを取得する。
2. [GitHub #941](https://github.com/hat47x/tei-platform/issues/941)では、Web調査で安全側のpublication boundaryを確定済み。原model / manualの公開範囲を広げるには、対象artifactへ適用される明示licenseまたは権利者による書面確認を取得する。
3. importer / runtime code変更後は、対象exact-headを固定してtrusted local cloneでfull `tools/repository-health-check.sh`を実行する。実装revisionが存在してからEvidence取得単位を確定する。
4. 業務設計としての転写妥当性を人が確認する場合、AI reviewとは別Evidenceとして保持する。

#940 / #941の結果は本IssueのEvidenceへ戻し、未取得のactual behaviorや利用条件を推測で補わない。#941の明示許諾が未取得でもRMT-1〜3はRepository外sourceを用いて継続し、public Evidenceへ原model内容を複製しない。

## 完了条件

- [ ] Evidence Gate RMT-0〜RMT-3のactual artifactが取得されている
- [ ] `MfgOrder.xead` / `MfgOrder.xeaf`のsource construct全体が転写Classificationへ割り当てられ、silent dropが無い
- [ ] source locatorからTEI側のmeaning / native asset / unresolvedへ逆追跡できる
- [ ] Data / Function-Interaction / Businessのcross-view diagnosticsをsource全体に適用できる
- [ ] relational diagnosticが残るsourceでfactをEvidenceとして保持しつつ、`relationalResolutionStatus = unresolved`によりdeterministic semantic loweringを止められる
- [ ] design Functionとruntime Realizationを1:1と仮定していない
- [ ] legacy TableKeyの型・field membership、Relationshipのkey参照、Function / Detailの明示`KeyFields`をsource factとして逆追跡でき、business identity / FD / domain constraintへ自動昇格しない
- [ ] declared CRUDとnative Script effectの差を検査またはUnresolvedとして保持できる
- [ ] identity / transaction / Environment / Secret / Session / actor / physical state等のmodernization pressureが、既存TEI境界または明示gapへ割り当てられている
- [ ] 主要実行型を代表するScenarioがTEI Driver / Targetでactual実行され、Semantic / Functional Evidenceが取得されている
- [ ] full-reference accountingで、特定sliceだけへの過適合がないことを確認している
- [ ] Reference転写で必要になった業務・設計概念を原則Semantic Extension / Pluginとして表し、Core固有enumの追加が必要な場合だけReference-local Extension / native preservationでは不足するEvidenceとADRがある
- [ ] original X-TEA actual runが必要なclaimは、actual比較または未取得Evidenceとして明示されている
- [ ] Reference Modelの利用条件とpublic Evidence境界が確認されている

## 完了後に可能になる判断

このIssueが完了すると、TEIについて次をEvidence付きで判断できる。

> 小規模な自作Referenceだけでなく、成熟した既存業務システムの設計資産を、意味・実装・環境・実行結果の境界を保ちながら移植可能な基盤として扱えるか。

不足が残った場合も価値がある。どの意味が失われるかをsource locatorとともに示せれば、次にCore、Extension、native asset boundary、Target、Runtimeのどこを改善すべきかを、Reference由来の具体的pressureとして決められる。

## 非目標

- X-TEA / XEADのdrop-in compatibility
- Java Swing UIの再実装
- Xead / Xeaf schemaをTEI標準schemaへすること
- Reference Model contentsの無条件な公開・再配布
- X-TEAの全仕様を一度にTEI Coreへ取り込むこと
- 旧Runtimeの暗黙挙動を、そのままTEIの推奨設計にすること
- source modelよりTEI都合を優先して「近い」fixtureへ作り替えること

## 関連

- [TEI Product Value](../../product/value/TEI_PRODUCT_VALUE.md)
- [TEI Product Requirements](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)
- [Business System Design Requirements](../../requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md)
- [ISSUE-000042 Business Procedure cross-view Reference](000042-business-procedure-cross-view-reference.md)
- [ISSUE-000077 Semantic Extension Reference](000077-semantic-extension-reference.md)
- [Reference Model Transfer Evidence Gate](../REFERENCE_MODEL_TRANSFER_GATE_2026-10.md)
