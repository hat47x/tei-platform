# TEI 業務システム設計アーキテクチャ

Status: current architecture baseline

## 1. 目的

この文書は、TEIの実装が変化しても保持すべき業務システム設計上の境界を整理します。

TEIの要求・Reference・Evidenceを正本とし、外部の公開知見は必要に応じて設計上の参考とします。特定の既存製品や手法との系譜・互換関係を前提にはしません。

対応する要求解釈は `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md` に置きます。

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

一つの視点だけで他を説明できない場合、設計はまだ閉じていません。

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

## 5. Business ActionとApplication Functionを同一視しない

```text
Business Action != Application Function
```

業務行為には、manual、physical、conversation、external system、Application Pattern、Capability、unresolved等の実現があり得ます。したがって、Application Function参照を持たないBusiness Actionも正常な状態です。

一つのFunctionが複数の業務文脈から利用されたり、一つの業務行為が複数Functionを利用したりできるN:M性を保持します。

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

これはarchitecture vocabularyであり、現時点の固定Canonical schemaではありません。Procedureをdurable Canonical identityにするか、traceable derived artifactで足りるかはReference実験から判断します。

## 8. Manual workをfirst-classな業務意味として残す

電話、対話、現物確認、組織判断、外部承認等が実際の業務に含まれるなら、それを直接表現できる必要があります。手作業をmodel上の欠陥や自動化待ちの穴とはみなしません。

また、`Validation PASS != Business Approval` です。業務判断をValidationへ置き換えるのは、accepted semantic contractが明示した場合だけです。

## 9. ActorとAuthorityを分ける

Business視点のactor、job、department、responsibilityと、Security architectureのauthenticated principal、membership、scope、policy、authorization decisionを分けます。

```text
Business Actor / Job
        !=
Authenticated Principal
        !=
Authorization Role / Policy
```

actor labelやUI convenienceからruntime authorityを推論しません。

## 10. 三視点のcross-check

Architectureとして少なくとも次を確認します。

- **Business → Function:** この業務行為はどう支援されるか。
- **Business → Data:** この仕事は何の情報に依存し、何を変えるか。
- **Function → Data:** 見えている操作にaccepted read / write meaningがあるか。
- **Data → Function:** 重要な意味をどこで確認・維持できるか。
- **Business / Function → Security:** interactionがaccepted authorityを越えないか。

## 11. Matrixはcross-check用derived Projectionとする

業務行為と機能、Data、Procedure等の対応をMatrixとして可視化することは、cross-checkに有効です。ただしMatrix自体を第二のsource of truthにはしません。

候補例:

```text
Business Action × Application Pattern
Business Action × Data / Relation
Procedure Step × View / Action
Business Activity × Data source
Application Pattern × Query / Capability
Data / Relation × impacted Procedure / View
```

Matrix上では `derived fact` / `human residual explanation` / `unknown` を区別し、名称の類似だけからunknownを自動補完しません。

## 12. Editorはconnected meaningの高密度Projectionとする

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

## 13. Driverは設計意味を実行・検査する環境とする

Driverは、TEIで採用された設計意味を実行可能な範囲で動かし、Semantic / Realization / Provider / Runtime / Evidenceの境界を検証する環境です。

少なくともAuthentication、Application Interaction、Validation、Authorization、Command、Business Result、Execution Error、explicit read、Observationの境界を確認できることを目指します。

Driverが実行するのはsystem-realizableな部分であり、Business Procedure全体を強制的にruntime Workflowへ変換しません。

## 14. Business meaningとTarget-neutral Presentationを分ける

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

## 15. 時間軸はopt-inの業務dimensionとする

Event順序、validity、history、lead time、waiting time、bottleneck、temporal resource use等が必要な業務では時間軸を明示します。一方、通常の業務へEvent Sourcing / Bitemporal architectureやDistributed eventual consistencyを一律導入しません。

## 16. Referenceは学習・圧力コンテンツとする

Referenceはtechnical fixtureだけでなく、学習例、platform pressure test、design-review target、regression Evidenceを兼ねられることを目指します。

第三者の公開資料から設計上の着想を得る場合も、TEIの仕様はTEI自身のRequirements、Reference、ADR、Evidenceによって説明できる状態を保ちます。第三者のモデルや文書の内容は、利用条件を確認せずに複製・再配布しません。

## 17. AIはcritic / proposerでありauthorityではない

AIはExisting knowledge探索、cross-reference候補、Procedure草案、Matrix / graph / UI Projection生成、不足検出、unresolved question提示を支援できます。

AI出力はAcceptanceされるまではProposal / derived artifactです。

```text
accepted fact
inference
proposal
unresolved
```

Human / Studio / CLI / AIのいずれも同じShared Acceptance Boundaryへ接続します。

## 18. 外部参照の概念をCoreへ先回りして写さない

外部参照に存在する概念を、その存在だけを理由にMicro Kernel primitiveへ追加しません。Procedure、Event、Subject Area、Subsystem、View、Decision、Handoff等は、TEI Reference Evidenceが必要性を示した最小durable layerへ導入します。

Identity / Type / Relation / Expression / Constraint / Lifecycle / Provenance等の小さなmechanismで十分なら専用primitiveを増やしません。

## 19. 未解決Evidence question

- **O-01 Procedure identity:** traceable derived Procedureでaccepted business meaningを保持できるか。
- **O-02 Business Flow:** runtime Workflow Engine化せず表す最小構造は何か。
- **O-03 Actor:** actor / jobのどこまでをstructuredにするか。
- **O-04 Decision / Rationale:** どの業務判断にdurable historyが必要か。
- **O-05 Subject Area / Subsystem:** durable meaningが必要か、derived viewで足りるか。
- **O-06 Target-neutral View:** Target Adapter前にどこまでinteraction structureをaccepted meaningとして持つか。
- **O-07 Reference scale:** 業務システム設計を十分にpressure-testできる規模・domain diversityはどの程度か。
- **O-08 Temporal semantics:** 何をbusiness meaningとして保持し、何をanalysis / persistence realizationへ残すか。

これらはReference実験、ADR、Evidenceから決めます。

## 20. 現行TEI文書との関係

この文書は次と合わせて読みます。

- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`
- `docs/architecture.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/APPLICATION_PATTERN_V1_DESIGN.md`

業務システム設計の検討を理由に既存Gateを増やしたり、Studioのdeveloper-facing HostをApplication Target Evidenceへ読み替えたりしません。Procedure / Business Flow等の実験は、Evidenceが正式なRequirement / Architecture変更を要求するまではplanning / Reference側で進めます。
