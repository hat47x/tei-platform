# TEI 業務システム設計 要求解釈基線

Status: current requirement interpretation baseline

## 1. この文書の位置づけ

この文書は、「渡辺幸三の業務システム設計」シリーズで深めてきた検討を、現行の `TEI_PRODUCT_REQUIREMENTS.md` に接続するための要求解釈基線です。

新しいProduct Requirement IDは追加しません。正式な要求IDは引き続き `RQ-001` 〜 `RQ-024` とし、この文書では、TEIを業務システム設計へ用いるときに既存Requirementをどう読むかを固定します。

判断に迷う場合は、次の順で設計原点へ立ち返ります。

1. 渡辺幸三氏の業務システム設計・三要素分析法に関する言説
2. 公開されているX-TEA Modeler / Driverのマニュアル・ソース
3. X-TEA / CONCEPTWARE系リファレンスモデルから得られるEvidence
4. TEI固有のProduct Requirements / Reference / Evidence

これはX-TEAの実装を再現するための互換規則ではありません。TEIの命名、authority境界、Target-neutral性、Provider分離、現行Evidenceを維持したまま、設計上の意図を取り戻すためのfallbackです。

## 2. 三要素は三つの正本ではなく、同じ業務システムを相互牽制する三視点とする

TEIでは、同じ業務システムを少なくとも次の三視点から確認できる必要があります。

```text
Data
Function / Interaction
Business
```

これは三要素分析法のデータモデル・機能モデル・業務モデルに対応する視点です。

ただし、三つの独立したsemantic source of truthを作って後からmergeすることはしません。同じaccepted meaningを複数Projectionから確認し、相互に不備を露出させます。

主に次のRequirementをこの意味で解釈します。

- RQ-001 業務上の意味を実装技術から独立して保持
- RQ-003 暗黙の複数正本を作らない
- RQ-010 Editor
- RQ-011 複数Projection
- RQ-017 人の二重管理を最小化
- RQ-021 Reference / Conformance

Matrix、業務マニュアル、UI、生成文書、AI要約等は、accepted meaningから導出できる部分について第二正本になってはいけません。

## 3. Data視点

Data視点では、物理DBとは独立して、業務上の事実、型、identity、Relation、cardinality、Constraint、lifecycleを扱います。

少なくとも次を確認できる必要があります。

- どの業務上の事実が存在するか
- 何が何を識別・参照するか
- Relationとcardinalityはどうなっているか
- semantic constraintとphysical realizationを区別できるか
- どのFunction / Procedureがその意味へ依存しているか

physical table名、FK、index、Provider catalog、Migration mechanicsは、それ自体が業務意味として受理されない限りrealization側に置きます。

Data-firstの考え方は維持しますが、Data-onlyにはしません。Data Modelだけでは、業務の進め方や人とシステムのinteractionを十分に設計したことにはなりません。

## 4. Function / Interaction視点

機能モデルはbackend function一覧だけではなく、人・マシンとのinteractionを含みます。

対象には必要に応じて次を含みます。

- 画面 / form
- 帳票 / document
- input / output
- browse / record / reference / header-detail / command
- Query / Capability
- batch / API
- Function間呼出し
- Target固有UI Projection

UIは、利用者が具体的な操作像からDataや業務の不備を指摘できる重要な設計レビューProjectionです。

一方で、Web widget、Wails API、HTML、特定frameworkのlayout等をCanonicalな業務意味へ昇格させません。

また、次を明確に分けます。

```text
Business Action != Application Function
```

業務行為の一部をApplication Functionが実現することはあっても、両者を同一視しません。

Application Pattern / Capabilityを直接持たない手作業・外部作業も、正当な業務意味として保持できなければなりません。

## 5. Business視点

Business視点では、システムの外側を含む社会的な仕事の進み方を扱います。

業務を十分に説明するため、必要に応じて次を区別できる必要があります。

- purpose
- trigger / event
- actor / job responsibility
- business action / step
- sequence（順序に意味がある場合）
- decision / judgment
- manual instruction
- Function / View / Pattern利用
- handoff
- exception
- completion / continuation
- unresolved

これらは設計上区別すべき概念ですが、この文書だけを根拠に各概念を専用Canonical primitiveや `.tei` syntaxへ固定しません。

次の分離は必須です。

```text
Business Actor / Job
    != Authentication Principal
    != Authorization Role / Policy
```

業務上の「担当者」というラベルだけで実行authorityを与えてはいけません。

同様に、

```text
Validation PASS != Business Judgment / Approval
```

です。accepted semantic contractが明示しない限り、機械判定可能なValidationの成功だけで業務上の判断全体を置き換えません。

## 6. Business Flow / Procedure / Runtime Workflowを分離する

業務モデルを、画面遷移や単純な処理順序へ還元してはいけません。

三要素分析法での業務フローは、業務ノードと情報・データソース等のノードの間をデータ／情報が流れるDFD型の見方を含みます。各業務はEvent / 契機を持ち、業務同士は情報やEventを介して疎結合に扱える必要があります。

一方、Procedureは一つの業務をどのように行うかを、手作業やFunction / UI利用を含めてより細かく説明する視点です。

したがって次を同一視しません。

```text
Business Flow != Procedure != Runtime Workflow Orchestration
```

ProcedureやBusiness Flowがあることから、retry、timer、queue、saga、lease等のruntime orchestrationを自動的に導入してはいけません。

## 7. 時間軸は必要な業務へ適用し、全体前提にはしない

時間軸は、履歴、Event順序、validity、lead time、待ち時間、bottleneck、resource利用等の業務解像度を高めます。

ただし、一般的な業務システムへ一律にEvent Sourcing、Bitemporal、append-only ledger、distributed eventual consistencyを要求しません。

通常のtransactional DBで正確に扱える業務は、その前提を維持できます。

時間に関する意味は、domainとReference Evidenceが要求するときに導入します。

## 8. 三視点の相互牽制をRequirement成立の一部とする

TEIは、三視点間で少なくとも次のような矛盾を発見できる構造を目指します。

```text
Business -> Function
  必要な業務行為に、意図した手作業経路もsystem interactionも存在しない

Business -> Data
  業務に必要な情報がaccepted Data meaningに存在しない

Function -> Data
  UI上は編集可能だが、accepted write meaningが存在しない

Data -> Function
  重要なData / Relationを業務上必要な場所で確認・維持できない

Business / Function -> Authorization
  actor labelや便利なUIがaccepted authorityを越える
```

矛盾が見つかった場合は、Projection側だけを修正して見えなくするのではなく、authoritative sourceとShared Acceptance Boundaryへ戻ります。

## 9. MatrixとManualは原則derivedとする

X-TEAで重視されてきたBusiness × Function、Business × Table、Table × Function等のMatrixは、TEIでも重要なレビュー手段とします。

可能な限りexplicit referenceから導出します。

例:

- Business Action × Application Pattern
- Business Action × Data / Relation
- Procedure Step × View / Action
- Application Pattern × Query / Capability
- Data / Relation × impacted Procedure / View

不明な関係は、名称から推測して埋めずunknownのまま残します。

生成可能なfactと、人が補う説明は区別します。

## 10. RQ-010 Editorの解釈

Editorは単純なform wizardではなく、複雑な業務システムを高密度に確認する設計環境として解釈します。

Data、Relation、Rule、View、Application intent、Procedure、Source、Evidence等を、必要に応じて3階層以上の関連を保ちながら確認できることを目指します。

EditorはProjection / Authoring Clientであり、第二のCanonical authorityにはしません。

また、自動生成で矛盾を隠すのではなく、三視点間の緊張を見つけやすくすることを重視します。

## 11. RQ-012 Driverの解釈

Driverは「実行可能な仕様」の系譜にある設計検証環境として解釈します。

最終Targetのコード生成・手実装を待たず、accepted Application meaningを実行・観測して整合性を確認できる必要があります。

少なくとも次を区別します。

- Semantic meaning
- Validation
- Authorization
- Command request
- Business Result
- Execution Error
- explicit read
- Observation

Driverが実行するのはsystem-realizableな部分です。周辺の手作業まで無理に実行可能なWorkflowへ変換することは要求しません。

## 12. RQ-014 Application Patternの解釈

`browse`、`record`、`reference`、`header-detail`、`command`等のApplication Patternは、Target非依存のApplication intentであり、Function / Interaction視点の再利用単位です。

Pattern集合だけでは、周辺のBusiness Procedureは定義されません。

```text
Procedure -> multiple Patterns
multiple Procedures -> same Pattern
Business Action -> no Pattern
```

はいずれも許容します。

Patternの並びを、accepted Business Modelなしに業務手順として解釈してはいけません。

## 13. Reference Modelはfixtureだけでなく実行可能な設計コンテンツとする

RQ-021は技術fixtureの再現性だけで終わらせません。

Reference Modelは将来的に次の役割を兼ねられることを目指します。

- reproducible Conformance input
- 業務システムモデリングの学習例
- Editor / Driver / Targetへのpressure test
- modelとplatformを相互改善する設計対象
- regression Evidence

CONCEPTWARE系モデルは歴史的な設計Evidenceとして構造を調査しますが、ライセンスとprovenanceが確認できるまでは内容をTEIへコピーしません。

業務システム設計の成熟を主張する前には、toy exampleを越える複数Subject Area / Subsystem・複数Data・複数Business Action / Functionを持つReferenceで圧力試験する必要があります。

具体的な規模やGateは、この文書では固定しません。

## 14. AI支援も同じauthority境界に従う

AIは既存知識の探索、Procedure候補、Matrix、UI Projection、三視点の不足指摘等を支援できます。

ただしAI固有のsemantic authorityを作りません。

AI Proposalは、利用したCanonical ID / Source / Evidenceを追跡可能にし、accepted fact / inference / proposal / unresolvedを区別し、Human / Studio / CLIと同じShared Acceptance Boundaryを通します。

これはRQ-015 / RQ-016 / RQ-017 / RQ-021の解釈です。

## 15. X-TEA名称はprovenanceとして扱う

X-TEA、XEAD、XeadModeler、XeadDriver等の名称は、歴史的・設計系譜上のprovenanceとして扱います。

新しいTEIの実装symbol、file format、namespace、generated artifact、public product namingはTEI名を使います。

目的は過去製品のsurfaceを再現することではなく、設計知識を継承・発展させることです。

## 16. Evidence待ちとして残す論点

次は、名称の好みではなくReference Evidenceから判断します。

1. Business Procedureをdurable Canonical identityにするか、traceable derived artifactで足りるか
2. actor / event / handoff / decision / manual instructionのどこまでをstructured semanticsにするか
3. DFD型Business FlowをProcedure / Data / Pattern / Eventへどう接続するか
4. Subject Area / Subsystemにdurable identityが必要か
5. Target Adapter前にどの程度のTarget-neutral View / Interaction構造が必要か
6. realistic Reference Modelをどうpackage / version / licenseするか
7. temporal semanticsのどこまでをdomain meaningとし、どこからをanalysis / realizationとするか

Evidenceが揃うまでは、X-TEAに同名概念があることだけを理由にCore schemaやProduct Requirement IDを追加しません。
