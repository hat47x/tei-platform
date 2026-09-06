# TEI 業務システム設計アーキテクチャ

Status: current architecture baseline

## 1. 目的

この文書は、TEIの実装が変化しても保持すべき業務システム設計上の境界を整理します。

「渡辺幸三の業務システム設計」シリーズで深めてきた検討を、現行TEI architectureへ接続するための正式基線です。

対応する要求解釈は `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md` に置きます。

この文書では、専用Canonical typeや `.tei` syntaxを先に固定するより、何を同一視してはいけないかを優先して定めます。

## 2. 設計系譜とfallback

TEIの設計出発点には、渡辺幸三氏の業務システム設計、三要素分析法、X-TEA Modeler / Driver、リファレンスモデルの実践があります。

判断に迷った場合は次へ立ち返ります。

```text
渡辺幸三氏の設計意図
        ↓
X-TEA公開マニュアル / ソース
        ↓
リファレンスモデルからのEvidence
        ↓
TEI Product Requirements / Reference / Evidence
```

ここで取り戻す対象は設計上の理由です。過去のclass構造、XML、UI、runtime実装を互換のために再現することは目的としません。

TEIは、元の意味境界を保つ限り、より小さいKernel、異なるauthoring form、異なるTarget、異なる実装技術を採用できます。

## 3. 一つの業務システムを三視点から相互牽制する

業務システムを少なくとも次の三視点から確認します。

```text
                Business
               /        \
              /          \
           Data -------- Function / Interaction
```

三視点は工程順でも、独立した三つのsource of truthでもありません。

同じaccepted meaningを別角度から見て、相互に欠落・矛盾・過剰な仮定を露出させます。

### Data

主に次を扱います。

- business fact
- type
- identity
- Relation
- cardinality
- semantic Constraint
- lifecycle

### Function / Interaction

主に次を扱います。

- 人・マシンが何を見て何を行うか
- screen / form / report
- input / output
- browse / record / reference / header-detail / command
- Query / Capability
- batch / API
- Function間呼出し

### Business

主に次を扱います。

- なぜ仕事が始まるか
- 誰がどの責任文脈で行うか
- どの業務行為を行うか
- 何がmanual / externalか
- どのsystem interactionを使うか
- judgment / handoff / exception / continuation

一つの視点だけで他を説明できない場合、設計はまだ閉じていません。

## 4. Data-first but not Data-only

TEIでは、変わりにくい業務DataとRelationを画面・物理DB・実装技術から分離します。

```text
business Data meaning
        ↓
Persistence Mapping
        ↓
physical schema / Provider
```

Data-firstの価値は維持します。

一方、正規化されたData Modelだけで業務システム設計を完了したことにはしません。

Data視点はFunction / Business視点から次のように批判される必要があります。

- 利用者が必要な意味を実際に確認・維持できるか
- 業務行為が必要とする情報が不足していないか
- UI上の操作がaccepted mutation contractと整合するか
- 業務の進め方から不足Relation / Constraintが見えてこないか

このfeedback loop自体を上流設計の一部とします。

## 5. Function modelはUIを含むが、UIはauthorityではない

Function視点には、利用者が具体的に理解できるUIを含みます。

```text
screen / form
report / document
input / output
query / browse
record interaction
reference interaction
header-detail interaction
command
batch
API
function call relationship
```

現行TEIの `browse` / `record` / `reference` / `header-detail` / `command` は、このFunction / Interaction視点をTarget非依存で扱うApplication Patternです。

具体的なWeb画面やDesktop UIはProjectionです。

```text
accepted Application meaning
        ↓
Presentation / Interaction projection
        ↓
Target Adapter
        ↓
concrete UI / API / MCP / VUI
```

Projectionは設計不備を露出させてよい一方、不備を隠すためにCanonical meaningを黙って変更してはいけません。

## 6. Business ActionとApplication Functionを同一視しない

中心的な境界は次です。

```text
Business Action != Application Function
```

業務行為は、TEIを使わなくても成立し得ます。

例えば次のrealizationがあり得ます。

- manual
- physical
- conversation
- external system
- Application Pattern
- Capability
- unresolved

したがって、

```text
Business Action -> no Application Function
```

は正常な状態です。

Functionが存在しないことを嫌って、契約のないfake Capabilityを作ってはいけません。

逆に、一つのFunctionが複数の業務文脈から利用されたり、一つの業務行為が複数Functionを利用したりできます。

業務モデルをmenu / screen hierarchyへ還元しないため、このN:M性を保持します。

## 7. Business Flowは単純なTask順序ではない

三要素分析法でいう業務フローは、単なる「業務Aの次に業務B」というcontrol-flow図だけではありません。

業務process / activityと、情報・データソース等のnodeの間をdata / informationが流れるDFD型の見方を含みます。

概念的には次です。

```text
Event / circumstance
        ↓
Business Activity A
        │
        ├── information / document / business data ──> Data source / external party
        │
        └── information / event ──────────────────────> Business Activity B
```

各業務はそれぞれのtrigger / eventを持ち、情報やEventを介して疎結合に扱えます。

これは業務システムを、一つの同期transactionではなく社会システムとして見るために重要です。

次を分離します。

```text
Business Flow
Business Procedure
Application Interaction
Runtime Workflow Orchestration
```

将来、一つのgraph Projectionへ同時表示することはあっても、一つのsemantic objectへ畳みません。

## 8. Procedureは一つの仕事の進め方を表す

Procedureは、ある業務を具体的にどう行うかを説明するBusiness側の詳細視点です。

概念形は次のようになります。

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

これはarchitecture vocabularyであり、現時点の固定Canonical schemaではありません。

Procedureをdurable Canonical identityにするか、traceable derived artifactで足りるかは未決です。

Reference実験で、regeneration、impact analysis、reuse、review時にaccepted business meaningが失われるかを見て判断します。

## 9. Manual workをfirst-classな業務意味として残す

手作業を、model上の欠陥や「いつか自動化すべき穴」とみなしません。

電話、対話、現物確認、組織判断、外部承認等が実際の業務に含まれるなら、それを直接表現できる必要があります。

したがって、Business StepにFunction参照がない状態を許容します。

また、

```text
Validation PASS != Business Approval
```

です。

業務上の判断をValidationへ置き換えるのは、accepted semantic contractが明示した場合だけです。

## 10. ActorとAuthorityを分ける

Business視点では、actor、job、department、responsibility等を扱えます。

Security architectureでは別に、authenticated principal、membership、scope、policy、authorization decisionを扱います。

```text
Business Actor / Job
        !=
Authenticated Principal
        !=
Authorization Role / Policy
```

Procedureに「受注担当」と書かれていても、その語からruntime authorityを推論してはいけません。

Authorizationは従来どおりCapability / Policy境界を通します。

この分離により、既存システム固有の組織・認証認可構造を一つのUniversal Role tableへ押し込まずに済みます。

## 11. 三視点のcross-check

Architectureとして次の批判経路を持ちます。

### Business -> Function

```text
この業務行為はどう支援されるか
```

Pattern / Capability / manual / external / unresolvedのいずれでも構いません。

### Business -> Data

```text
この仕事は何の情報に依存し、何を変えるのか
```

不足Data / Relationを露出させます。

### Function -> Data

```text
この見えている操作にはaccepted read / write meaningがあるか
```

read-only datasourceに対してUI convenienceだけで「編集」を生やしてはいけません。

### Data -> Function

```text
この重要な意味はどこで確認・維持できるか
```

到達不能なData meaningはFunction設計の不足を示す可能性があります。

### Business / Function -> Security

```text
このinteractionはaccepted authorityを越えないか
```

actor labelやUI convenienceはsecurity semanticsを迂回できません。

## 12. Matrixはcross-check用derived Projectionとする

X-TEA系で使われてきたMatrixを、TEIでも設計レビュー手段として重視します。

可能な限りexplicit relationから導出します。

例:

```text
Business Action × Application Pattern
Business Action × Data / Relation
Procedure Step × View / Action
Business Activity × Data source
Application Pattern × Query / Capability
Data / Relation × impacted Procedure / View
```

Matrix上では、少なくとも次を区別します。

```text
derived fact
human residual explanation
unknown
```

名称の類似だけからunknownを自動補完しません。

## 13. Editorはconnected meaningの高密度Projectionとする

Editorは、前述cross-checkを現実的に行える局所性を持つ必要があります。

Data、Relation、Rule、View、Application intent、Procedure、Source、Evidenceを、型ごとに孤立したwizard画面へ追い出しません。

例えば次のような3階層以上の関係を局所的に追えることを目指します。

```text
Business Procedure
   └─ Step
       └─ Application Pattern
           ├─ Data / Relation
           ├─ Query / Capability
           ├─ Rule / Validation
           └─ Source / Evidence
```

Dataから逆向きに辿ることもできるべきです。

EditorはAuthoring / Projection Clientであり、accepted file-native sourceとShared Acceptance Boundaryがauthorityを持ちます。

## 14. Driverは実行可能仕様の検証環境とする

Driverは、設計意味を最終Target実装前に実行・検査できる環境です。

少なくとも次の境界を実際に確認できることを目指します。

```text
Authentication
Application Interaction
Validation
Authorization
Command
Business Result
Execution Error
explicit read
Observation
```

これは、仕様を書けばDriverで動かして設計を検証できるX-TEA系の思想を継承します。

TEIではさらにProvider、authority、Target、Evidence境界を分離します。

Driverが実行するのはsystem-realizableな部分であり、Business Procedure全体を強制的にruntime Workflowへ変換しません。

## 15. Business meaningとTarget-neutral Presentationを分ける

ProcedureがView / Patternを参照することはあっても、具体的なWeb component、Wails API、HTML template、mobile widgetへ直接依存させません。

同じbusiness / Application meaningを複数TargetへProjectionできることを圧力試験します。

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

Target差はTarget CapabilityとExecution / Trust Profileへ置き、business meaningをTarget都合で変えません。

## 16. 時間軸はopt-inの業務dimensionとする

時間を見ることで、次のような解像度を得られます。

- Event順序
- validity
- history
- lead time
- waiting time
- bottleneck
- temporal resource use

一方で、通常の業務システムへEvent Sourcing / Bitemporal architectureを一律導入しません。

通常のACID transactionで十分な場合はそれを使います。

Distributed eventual consistencyも、必要性のない業務へ既定前提として持ち込みません。

## 17. Reference Modelは実行可能な学習・圧力コンテンツとする

Reference Modelはtechnical fixtureだけではありません。

次を兼ねられることを目指します。

- reproducible model
- 学習例
- platform pressure test
- design-review target
- regression Evidence

理想的なloopは次です。

```text
Reference Model
      ↓ exercises
TEI Platform / Editor / Driver / Target
      ↓ exposes weakness
Model or Platform correction
      ↓
Reference / Conformance re-run
```

これはCONCEPTWAREの役割に近いものですが、provenanceとlicenseは明確に分けます。

歴史的モデルの構造は研究できますが、内容は許諾確認なしにコピーしません。

将来のrealistic Referenceでは、toy `order_entry`だけでは出ないlocality / traceability問題を露出させるため、複数Subject Area / Subsystem、複数Data、複数Business Action / Function、十分なcross-relationを持たせます。

## 18. AIはcritic / proposerでありauthorityではない

AIは次を支援できます。

- Existing knowledge探索
- cross-reference候補
- Procedure草案
- Matrix / graph / UI Projection生成
- Data / Function / Business間の不足検出
- unresolved questionの提示

AI出力はAcceptanceされるまではProposal / derived artifactです。

少なくとも次を区別します。

```text
accepted fact
inference
proposal
unresolved
```

利用したCanonical ID / Source / Evidenceへ戻れる必要があります。

Human / Studio / CLI / AIのいずれも同じShared Acceptance Boundaryへ接続します。

## 19. 歴史的概念をCoreへ先回りして写さない

この設計系譜は、X-TEAに存在した全概念をMicro Kernel primitiveへ追加する根拠ではありません。

Procedure、Event、Subject Area、Subsystem、View、Decision、Handoff等は、TEI Reference Evidenceが必要性を示した最小durable layerへ導入します。

Identity / Type / Relation / Expression / Constraint / Lifecycle / Provenance等の小さなmechanismで十分なら、専用primitiveを増やしません。

設計上の圧力を継承しつつ、過去の実装構造そのものを再現しないための境界です。

## 20. 未解決Evidence question

### O-01 Procedure identity

traceable derived Procedureでaccepted business meaningを保持できるか、durable identityが必要か。

### O-02 Business Flow

DFD型のBusiness Activity、Data / Information Source、Event、Flowを、runtime Workflow Engine化せず表す最小構造は何か。

### O-03 Actor

actor / jobのどこまでをstructuredにし、どこからをdescriptiveにするか。

### O-04 Decision / Rationale

どの業務判断にQuestion / Decision / Rationaleのdurable historyが必要か。

### O-05 Subject Area / Subsystem

file organizationやApplication / Database境界を越えてdurable meaningが必要か。

### O-06 Target-neutral View

Universal widget DSLを作らず、Target Adapter前にどこまでinteraction structureをaccepted meaningとして持つか。

### O-07 Reference Model scale

業務システム設計の成熟を主張できるReference規模・domain diversityはどの程度か。

### O-08 Temporal semantics

何をbusiness meaningとして保持し、何をanalysis / persistence realizationに残すか。

これらはReference実験、ADR、Evidenceから決めます。用語の好みだけで固定しません。

## 21. 現行TEI文書との関係

この文書は次と合わせて読みます。

- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`
- `docs/architecture.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/APPLICATION_PATTERN_V1_DESIGN.md`

Gate B / Gate Cの既存順序は変更しません。

業務システム設計の検討を理由にGateを増やしたり、Studioのdeveloper-facing HostをApplication Target Evidenceへ読み替えたりしません。

Procedure / Business Flow等の実験は、Evidenceが正式なRequirement / Architecture変更を要求するまではplanning / Reference側で進めます。
