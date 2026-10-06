# TEI 業務システム設計アーキテクチャ

Status: current architecture baseline

## 1. 目的

この文書は、TEIの実装が変化しても保持すべき業務システム設計上の境界を整理します。

TEIの要求・Reference・Evidenceを正本とし、外部の公開知見は必要に応じて設計上の参考とします。特定の既存製品や手法との系譜・互換関係を前提にはしません。

対応する要求解釈は `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md` に置きます。Business Procedure authoring / reviewの具体的な品質観点は `plan/BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md` に置き、このArchitectureへ新しいfield一覧を逆輸入しません。

## 2. 一つの業務システムを複数視点から確認する

業務システムを少なくとも次の三視点から確認します。

```text
                Business
               /        \
              /          \
           Data -------- Function / Interaction
```

三視点は工程順でも、独立した三つのsource of truthでもありません。同じaccepted meaningを別角度から見て、欠落・矛盾・過剰な仮定を見つけるための視点です。

- **Data** — business fact、type、identity、Relation、cardinality、Constraint、lifecycle
- **Function / Interaction** — screen / form / report、input / output、Query / Capability、batch / API、Function間呼出し
- **Business** — trigger、actor / responsibility、business action、manual / external、judgment、handoff、exception、continuation

一つの視点だけで他を説明できない場合、設計はまだ閉じていません。一方、情報が不足している場合に他視点から推測で穴を埋めることも「閉じた設計」とはみなしません。

## 3. Data-first but not Data-only

TEIでは、変わりにくい業務DataとRelationを画面・物理DB・実装技術から分離します。

```text
business Data meaning
        ↓
Persistence Mapping
        ↓
physical schema / Provider
```

一方、正規化されたData Modelだけで業務システム設計を完了したことにはしません。Function / Business視点から、利用者が意味を確認できるか、業務行為に必要な情報が揃うか、UI操作とmutation contractが整合するかをcross-checkします。

Data traceが見つからないことを、その業務にData dependencyが存在しないというnegative factへ変換しません。

### Data meaningとrecord instanceを分ける

Data視点でいう`Customer`、`Order`、`Card`等のbusiness meaningと、Applicationが実際に保持する個々のrecordを同一視しません。

```text
Data definition / business meaning
  != persisted record instance
  != current runtime state
```

Relationについても、`Order refers to Customer`というrelation definitionと、ある`order-2026-001 -> customer-123`というinstance-level factは別です。

instance-level factはApplication data、Database、external system、Observation等で扱えます。Business design上必要ならEvidenceとして参照しますが、個々のinstanceをすべてCanonical Business designへ持ち込むことを要求しません。

ただしmaster/reference data、configuration、policy table等、instance集合自体が設計authorityを持つdomainでは明示的なauthoritative sourceとして扱う余地があります。その場合もdefinition / instance / Observationのどれがownerかを曖昧にしません。

### 業務上の識別・関数従属性・物理キーを分ける

Data設計を、各tableへ一律に単独の`id`を付けるところから始めません。まず、業務上の一つのoccurrenceを何が識別するのか、どのData項目が何によって定まるのかを確認します。識別と関数従属性は、画面やORM、Database Providerの都合より先に検討するData meaningです。

TEIでは、少なくとも次のidentityを同一視しません。

```text
Canonical subject identity
  != business occurrence identifier
  != physical primary key
```

Canonical identityは、設計上のsubjectをRepositoryやTargetをまたいで追跡するためのidentityです。business occurrence identifierは、対象業務で「何を同じ一件として扱うか」を表します。physical primary keyは、特定DatabaseやProviderでrecordを実装・参照するための仕組みです。

したがって、UUIDやsurrogate keyを物理実装で使うこと自体は否定しません。一方、それを追加したことで、業務上の複合識別や関数従属性、Relationの違いが見えなくなる設計は避けます。自然キーを物理主キーとして必須にすることも要求しません。

Provider、ORM、Target等の制約からsurrogate keyが必要になる場合も、本来のcandidate key、uniqueness、Relationの意味を失わないMapping / Constraintを保ちます。物理実装上の都合で正規化された構造から外れる場合は、その差を暗黙の標準形として扱わず、意図したRealization差として追跡します。

View、Application Pattern、formの入力単位、runtime objectのidentityから、business occurrence identifierを逆算しません。Presentationや実装objectはData meaningを利用できますが、その都合をData identityのauthorityにはしません。

たとえば、1:Nに見える構造でも意味は同じとは限りません。親から独立して存在し、参照先を変更し得るrecordと、特定の親occurrenceの内部で繰り返される明細と、独立した二つの識別対象の組合せを表すrecordでは、業務上の識別と従属性が異なります。UI上の見え方や「子table」という呼称だけから同じRelationへ畳みません。

Data設計時には、必要に応じて次を確認します。

- 何がそのoccurrenceを業務上識別するか
- determinantとなる項目または項目組と、そこから定まるdependentな項目は何か
- Relationが変更可能な参照なのか、親の内部にある反復なのか、独立した識別対象同士の組合せなのか
- 識別や従属性がどの時点・期間・状態で成立するか
- sourceから確定できることと、候補・推論・unresolvedのどれに留まるか

関数従属性そのものを、現時点から必ず新しいtop-level Canonical kindとして持つとは決めません。既存のData / Relation / Constraint / Source / Evidenceでlosslessに表現・追跡できる範囲を確認し、不足がReferenceで再現された場合に表現を追加します。

正規化はData設計を検査する有力な手段ですが、特定の正規形を全Persistenceへ機械的に強制しません。性能や外部schema互換性のために非正規化した実装を採る場合も、realization上の都合によって元のbusiness Data meaningやdeterminantを消さないことを優先します。

生成AIがData構造を提案する場合も、名称や一般的なschema patternだけから関数従属性を事実として補完しません。すべてのtableへ単独`id`を付けてRelationを作る案も、業務上の識別と従属性を確認した後の実現候補として扱います。根拠が不足している場合は、もっともらしいkey構造を確定せず、設計上の問いまたはunresolvedとして残します。

## 4. Function / InteractionはUIを含むが、UIはauthorityではない

Function / Interaction視点には、利用者が具体的に理解できるUI、帳票、入出力、query、command、batch、API、function call relationshipを含みます。

現行TEIの `browse` / `record` / `reference` / `header-detail` / `command` は、この視点をTarget非依存で扱うApplication Patternです。

```text
accepted Application meaning
        ↓
Presentation / Interaction projection
        ↓
Target Adapter
        ↓
concrete UI / API / MCP / VUI
```

Projectionは設計不備を見つけるために使えますが、Projection側の都合でCanonical meaningを黙って変更しません。

UI / Interaction Projectionは、利用者がDataやFunctionの意味を具体的に確認する**review surface**でもあります。Data Modelだけでは見落としやすい入力不足、意味の取り違え、利用不能な状態遷移等を、実際の利用像から発見できることを重視します。ただし、reviewしやすさのために具体化した表示内容や配置を、そのままCanonical authorityへ昇格させません。

## 5. Business ActionとApplication Functionを同一視しない

```text
Business Action != Application Function
```

業務行為には、manual、physical、conversation、external system、Application Pattern、Capability、unresolved等の実現があり得ます。したがって、Application Function参照を持たないBusiness Actionも正常な状態です。

一つのFunctionが複数の業務文脈から利用されたり、一つの業務行為が複数Functionを利用したりできるN:M性を保持します。

実現が未確定であることと、Business Action自体が未成立であることも区別します。Referenceを実行可能に見せるためだけにfake Function / Capabilityを作りません。

## 6. Business FlowとRuntime Workflowを分ける

Business Flowは単純なTask順序だけではなく、業務activityと情報・文書・business data・外部party等の間に生じる流れを含みます。

```text
Business Flow
Business Procedure
Application Interaction
Runtime Workflow Orchestration
```

これらは関連付けますが、一つのsemantic objectへ畳みません。将来、一つのgraph Projectionへ同時表示することは可能です。

## 7. Procedureは仕事の進め方を表す

Procedureは、ある業務を具体的にどう行うかを説明するBusiness側の視点です。

```text
Procedure
├─ purpose
├─ trigger / event
├─ actor / job context
├─ step / business action
│  ├─ manual instruction
│  ├─ optional View / Pattern / Capability reference
│  ├─ optional Data / Relation reference
│  └─ unresolved context
├─ decision / judgment
├─ exception
├─ handoff
└─ completion / continuation
```

これはarchitecture vocabularyであり、現時点の固定Canonical schemaではありません。特にoptional Data / Relation等の語彙が存在することは、current minimum authoring contractへdirect ref fieldを追加する根拠ではありません。

Procedureをdurable Canonical identityにするか、traceable derived artifactで足りるかはReference実験から判断します。

Procedure品質は「すべての要素を埋めること」ではなく、scope内でknown / accepted / unresolvedの境界を説明できることを重視します。

### Derived Procedural KnowledgeはBusiness Procedureではない

成功 / 失敗trajectory、agent exploration、過去の実行順、developer guidance等から、再利用可能なprocedural candidateを導出することはできます。

ただし、

```text
Derived Procedural Knowledge
  != Business Procedure
  != accepted Business Action sequence
  != Runtime Workflow authority
```

です。

「以前この順序で成功した」「agentがこの近傍をよく辿った」という経験は、Business Procedureを検討する候補やreview hintにはできますが、業務上の正しい手順・責任・Handoff・Authorization・Capability authorityを自動的には確定しません。

derived procedural candidateをBusiness meaningへ反映する場合は、元trajectory / Execution / Observation / Evidenceへ戻れるprovenanceを保ち、current accepted authority / Business contextで再確認したうえでProposalとしてShared Acceptance Boundaryへ送ります。

Derived Procedural Knowledgeやそのgraph / indexを削除・再生成しても、accepted Business meaningとCanonical Worldは変わってはいけません。Procedural Graphを理由にBusiness Procedure schema、Runtime Workflow Engine、top-level Canonical kindを先行追加しません。

## 8. Manual workをfirst-classな業務意味として残す

電話、対話、現物確認、組織判断、外部承認等が実際の業務に含まれるなら、それを直接表現できる必要があります。手作業をmodel上の欠陥や自動化待ちの穴とはみなしません。

また、`Validation PASS != Business Approval` です。業務判断をValidationへ置き換えるのは、accepted semantic contractが明示した場合だけです。

## 9. ActorとAuthorityとResponsibilityを分ける

Business視点のactor、job、department、responsibilityと、Security architectureのauthenticated principal、membership、scope、policy、authorization decisionを分けます。

AI Business ActorもBusiness側ではactor候補になり得ますが、「AIである」こと自体はAuthorization RoleでもCapability authorityでもありません。

```text
Business Responsibility / Actor / Job
        !=
Authenticated Principal
        !=
Authorization Role / Policy
        !=
Application owner
```

actor labelやUI convenienceからruntime authorityを推論しません。

同様に、

```text
AI understands Procedure
  != AI owns Procedure
  != AI is authorized to execute every Step
```

です。AI Actorが実行できないhuman judgment / physical / external Stepは、そのままhandoff / unresolved boundaryとして残します。AIへ実行可能に見せるためだけにBusiness meaningを書き換えません。

また、Business Handoffはactor changeそのものではありません。

```text
actor changed   != Handoff proven
actor unchanged != Handoff impossible
```

actor差はcorroborationになり得ますが、responsibility continuationはexplicit Business meaningで判断します。

## 10. Business設計ではEvidence natureを混ぜない

Business authoring / reviewでは、少なくとも次の違いを失わないようにします。

```text
source-backed fact
accepted TEI Business meaning
mechanically derived cross-view fact
proposal / inference
unresolved
```

これは現時点でCanonical enumを要求するものではありません。

重要なのはauthority directionです。

```text
external / public fact
  -> design pressure / constraintには使える
  -X-> exact TEI actor / Step / Handoffの自動authoring

derived cross-view fact
  -> review / contradiction discoveryに使える
  -X-> authoritative sourceへの暗黙backfill

proposal / inference
  -> Acceptance candidateになり得る
  -X-> Acceptance前のauthority

unresolved
  -> knowledge boundaryを示す
  -X-> convenient default
```

明示されたunknownは設計上の失敗とは限りません。unsupported detailで空白を埋めるfalse completenessの方を避けます。

## 11. Intention / system result / actuality / Business acceptanceを分ける

業務では「指示した」「systemが成功した」「現実に起きた」「業務上受け入れた」が一致しない場合があります。

```text
Business intention / instruction
        ↓ may request
Application / system action
        ↓ may report
system result
        ↓ may be corroborated by
Observation / external evidence
        ↓ may support
Business acceptance / completion / continuation
```

次を自動変換しません。

```text
instruction issued
  != physical work completed

command accepted
  != external fact observed

system record updated
  != physical-world fact necessarily true

physical fact observed
  != Business responsibility automatically transferred
```

domain contractが同値性を明示できる場合は統合してよいですが、実装順・画面遷移・状態名だけから同値とみなしません。

## 12. Handoffは責任継続のBusiness relationとする

Current architecture pressureでのthin Handoffは:

```text
Handoff
  from { procedure, step }
  to   { procedure, step }
  condition
```

です。

これは単なるchronological edgeではなく、**あるBusiness Stepにある仕事が、Business conditionの成立を境に、別のaddressable Business Stepで継続できる／継続すべきになること**を表します。

- `from` — responsibility transferの出発点としてreview可能なProcedure-scoped Step
- `to` — continuation側のBusiness responsibilityを担うaddressable Step
- `condition` — continuationがBusiness上成立するpredicate

`condition`はtarget Procedureのtriggerやruntime event IDそのものではありません。

Handoffをauthorしないもの。

```text
Step adjacency
Function sequence / call
actor difference
Lifecycle transition
Command success
physical Observation
queue / scheduler / transport event
```

これらがHandoffと併存する場合でも、Business relationのauthorityはexplicit accepted sourceへ戻します。

## 13. Handoff conditionと、その成立Evidenceを分ける

Handoff conditionは「何が成立すればBusiness responsibilityを継続できるか」という意味です。

Observation、document、approved record、external response等は、そのconditionが成立したと判断するEvidence候補です。

```text
Business condition
  != Observation identity
  != Evidence record
```

現在はHandoffへEvidence fieldを追加しません。second realistic Referenceでconditionだけではreview上lossyになる場合、Handoff schemaを拡張する前に既存Source / Provenance / Evidence mechanismでtrace可能かを確認します。

## 14. 三視点のcross-check

Architectureとして少なくとも次を確認します。

- **Business → Function:** この業務行為はどう支援されるか。支援されない／未確定でもBusiness意味は残るか。
- **Business → Data:** この仕事は何の情報に依存し、何を変えるか。trace不足をno dependencyへ丸めていないか。
- **Function → Data:** 見えている操作にaccepted read / write meaningがあるか。
- **Data → Function:** 重要な意味をどこで確認・維持できるか。
- **Business / Function → Security:** interactionがaccepted authorityを越えないか。
- **Function / Observation → Business:** system resultやobserved factをBusiness completion / Handoffへ過剰に昇格していないか。

Review結果は単純なPASS / FAILだけでなく、少なくとも意味上は:

```text
supported
contradiction
unresolved
```

を区別します。これはdiagnostic severityの固定enumではありません。

## 15. Matrixはcross-check用derived Projectionとする

業務行為と機能、Data、Procedure等の対応をMatrixとして可視化することは、cross-checkに有効です。ただしMatrix自体を第二のsource of truthにはしません。

候補例は次のとおりです。

```text
Business Action × Application Pattern
Business Action × Data / Relation
Procedure Step × View / Action
Business Activity × Data source
Application Pattern × Query / Capability
Data / Relation × impacted Procedure / View
```

Matrix上では `derived fact` / `human residual explanation` / `unknown` を区別し、名称の類似だけからunknownを自動補完しません。

Matrixの空欄も自動的な「なし」ではありません。exact traceが無いのか、not applicableなのか、unresolvedなのかを必要に応じて区別します。

## 16. Editorはconnected meaningの高密度Projectionとする

EditorはData、Relation、Rule、View、Application intent、Procedure、Source、Evidenceの関係を局所的に追える必要があります。

```text
Business Procedure
   └─ Step
       └─ Application Pattern
           ├─ Data / Relation
           ├─ Query / Capability
           ├─ Rule / Validation
           └─ Source / Evidence
```

EditorはAuthoring / Projection Clientであり、accepted file-native sourceとShared Acceptance Boundaryがauthorityを持ちます。

将来EditorでBusiness設計を扱う場合も、fact / accepted / derived / proposal / unresolvedの違いをUI上の見た目だけで失わないことを要求します。ただし現在から専用status fieldや色taxonomyを固定しません。

## 17. Driverは設計意味を実行・検査する環境とする

Driverは、TEIで採用された設計意味を実行可能な範囲で動かし、Semantic / Realization / Provider / Runtime / Evidenceの境界を検証する環境です。

少なくともAuthentication、Application Interaction、Validation、Authorization、Command、Business Result、Execution Error、explicit read、Observationの境界を確認できることを目指します。

Driverが実行するのはsystem-realizableな部分であり、Business Procedure全体を強制的にruntime Workflowへ変換しません。

Driver execution Evidenceが得られても、manual / physical / external Actionが自動的に実行済みになるわけではありません。

## 18. Business meaningとTarget-neutral Presentationを分ける

具体的なWeb component、Wails API、HTML template、mobile widgetをBusiness meaningへ直接埋め込みません。

```text
Business / Application meaning
        ↓
Target-neutral interaction intent
        ├─ Web
        ├─ Desktop
        ├─ Mobile
        ├─ API
        ├─ MCP
        └─ VUI
```

Target差はTarget CapabilityとExecution / Trust Profileへ置きます。

## 19. 時間軸はopt-inの業務dimensionとする

Event順序、validity、history、lead time、waiting time、bottleneck、temporal resource use等が必要な業務では時間軸を明示します。一方、通常の業務へEvent Sourcing / Bitemporal architectureやDistributed eventual consistencyを一律導入しません。

時系列上の前後関係だけでBusiness Handoffや責任境界を生成しません。

## 20. Referenceは学習・圧力コンテンツとする

Referenceはtechnical fixtureだけでなく、学習例、platform pressure test、design-review target、regression Evidenceを兼ねられることを目指します。

第三者の公開資料から設計上の着想を得る場合も、TEIの仕様はTEI自身のRequirements、Reference、ADR、Evidenceによって説明できる状態を保ちます。第三者のモデルや文書の内容は、利用条件を確認せずに複製・再配布しません。

Realistic Referenceは既存schemaを検証するためだけのfixtureではありません。Handoff等の対象fieldを取り除いても独立したBusiness needを説明できるseedから始め、そのneedから自然に出るpressureだけをcontract判断へ使います。

さらにReferenceは、TEIのArchitectureやRuntimeに都合のよい小例だけで成立させません。一定の業務複雑性を持つReferenceを通常のAuthoring / Validation / Binding / Driver / Target経路で扱えない場合、その事実をplatform側またはReference設計側のGapとして残します。Reference専用の例外実装で「対応済み」に見せません。

~~~text
Reference source / design
        ↓
author / validate / realize
        ↓
execute / observe / review
        ↓
platform gap or reference-design gap
        ↓
authoritative sourceへ戻って修正
~~~

実装を試みた結果、Reference自体の矛盾や不足が見つかることも正常なfeedbackです。したがって「モデルがある」「parseできる」「画面が出る」を同じ成熟度として扱わず、少なくとも設計表現、実行可能性、実装・観測による検証をEvidence上で区別します。これは新しい固定status enumを要求するものではありません。

一つのReferenceが動くことだけから汎用性を主張しません。異なるdomain、規模、Data構造、Business Procedure、Target / Provider条件を持つ複数Referenceによって、どの範囲までTEIの設計体系が成立するかを反証可能な形で広げます。

## 21. AIはcritic / proposerでありauthorityではない

AIはExisting knowledge探索、cross-reference候補、Procedure草案、Matrix / graph / UI Projection生成、不足検出、unresolved question提示を支援できます。

AI出力はAcceptanceされるまではProposal / derived artifactです。

```text
accepted fact / accepted meaning
inference
proposal
unresolved
```

Human / Studio / CLI / AIのいずれも同じShared Acceptance Boundaryへ接続します。

AIが空欄をもっともらしく補完することを品質向上とはみなしません。sourceに根拠がなければunresolvedを保持する提案も正当な結果です。

Data Modelについても同じです。AIがschemaを生成できたことや、生成結果がparse / type-checkを通ったことだけから、業務上のidentity、関数従属性、cardinality、Constraint、正規化上の妥当性が確認済みとはみなしません。source-backedなdeterminantや業務規則を確定できない場合は、もっともらしいkeyを補うよりunresolvedを残します。

決定論的なValidatorは、宣言済みのConstraintや整合性を検査できます。一方、業務の現実から未宣言の関数従属性や識別規則を自動的に「発見した」ことにはしません。AIによる候補発見と、accepted Data meaningの成立を分離します。

## 22. 外部参照の概念をCoreへ先回りして写さない

外部参照に存在する概念を、その存在だけを理由にMicro Kernel primitiveへ追加しません。Procedure、Event、Subject Area、Subsystem、View、Decision、Handoff等は、TEI Reference Evidenceが必要性を示した最小durable layerへ導入します。

Identity / Type / Relation / Expression / Constraint / Lifecycle / Provenance等の小さなmechanismで十分なら専用primitiveを増やしません。

外部Referenceは「その概念が存在する」という証明ではなく、「どの設計失敗を避けるためにその観点が必要か」を問い直す材料として使います。

## 23. 未解決Evidence question

- **O-01 Procedure identity:** traceable derived Procedureでaccepted business meaningを保持できるか。
- **O-02 Business Flow:** runtime Workflow Engine化せず表す最小構造は何か。
- **O-03 Actor / Responsibility:** actor / job / responsibilityのどこまでをstructuredにするか。
- **O-04 Decision / Rationale:** どの業務判断にdurable historyが必要か。
- **O-05 Subject Area / Subsystem:** durable meaningが必要か、derived viewで足りるか。
- **O-06 Target-neutral View:** Target Adapter前にどこまでinteraction structureをaccepted meaningとして持つか。
- **O-07 Reference scale:** 業務システム設計を十分にpressure-testできる規模・domain diversityはどの程度か。
- **O-08 Temporal semantics:** 何をbusiness meaningとして保持し、何をanalysis / persistence realizationへ残すか。
- **O-09 Evidence nature:** source-backed / accepted / derived / proposal / unresolvedの違いをどこまでdurable metadataとして保持する必要があるか。
- **O-10 Handoff Evidence:** Business condition成立を支えるEvidence traceを既存Provenanceで十分に説明できるか。
- **O-11 Data determinant:** business identity / candidate key / functional dependency / domain constraintのうち、どこまでをdurable semantic contractとして保持すべきか。
- **O-12 Reference falsifiability:** domain diversityと実装検証のどこまでが、TEIの汎用性を主張するための十分なReference Evidenceになるか。

これらはReference実験、ADR、Evidenceから決めます。

## 24. 現行TEI文書との関係

この文書は次と合わせて読みます。

- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`
- `docs/architecture.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`
- `plan/APPLICATION_PATTERN_V1_DESIGN.md`

業務システム設計の検討を理由に既存Gateを増やしたり、Studioのdeveloper-facing HostをApplication Target Evidenceへ読み替えたりしません。Procedure / Business Flow等の実験は、Evidenceが正式なRequirement / Architecture変更を要求するまではplanning / Reference側で進めます。
