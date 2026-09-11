# Concise Definition Body Reference Pressure — 2026-09

Status: **Planning / Architecture pressure — no new Gate / PAS / Product Requirement**

関連:

- `docs/definition-body-architecture.md`
- `docs/business-system-design.md`
- `plan/DERIVED_COGNITIVE_EXPLORATION_OVERVIEW_2026-09.md`
- `plan/PRODUCT_QUALITY_EVIDENCE_PLAN_2026-09.md`
- GitHub Issue #570

## 1. 目的

`docs/definition-body-architecture.md`の原則を、抽象的な「シンプルなDSLがよい」という好みで終わらせず、realistic Referenceで圧力試験できる形へ落とします。

検証したいのは次です。

> **人がauthorする意味記述を端的に保ちながら、業務上必要な区別を失わず、Function / Business / UI / impact / AI探索等の豊かなlensを再構成できるか。**

この計画はsourceを短くするbenchmarkではありません。

```text
minimize authored duplication
subject to semantic conservation
and deterministic reconstructability
```

を評価します。

## 2. 背景となるReference signal

過去のX-TEA/XEAD系Reference再評価では、toy fixtureではなく複数Subsystem・数十Data/Action規模のcontentを使うことで、Platform側の不足が見つかりました。

一つの過去pressure baselineでは概ね:

- 6 Subsystems
- 42 Data
- 57 explicit Actions
- 20 Procedures
- 147 Runtime Action + View items

を扱いながら、新しいtop-level surface conceptを増やさず圧力を解消できました。

特に重要だったのは、Functionを独立した第二正本へ戻さず、Action / View / read / CUD / call edgeからFunction lensを再構成できたことです。

また大量のderived inspection Viewを持ちながら、end-user application navigationに必要なViewだけをauthored intentとして残す分離も成立しました。

これらの数値自体をTEIのKPIや最低規模へ固定しません。重要なのは、**realistic contentで新しいsurface conceptの必要性を問うこと**です。

## 3. Pressureの評価単位

各Pressureは次を区別します。

```text
Authoritative input
Deterministic derivation contract
Derived output
Expected invariant
Observed gap
Decision
```

`Observed gap`があっても、すぐ新しいkeyword / kind / schemaを追加しません。

まず:

1. 既存sourceに意味が欠けているか
2. derived lensだけが不足しているか
3. realization / observation情報の不足か
4. cognition / retrievalの不足か
5. Reference fixture自体の不足か

を分類します。

## 4. CDB-P01 — Current source classification

### 目的

現在の`order_entry`等のsourceを、Definition Body ArchitectureのClass A〜Eへ分類し、authority leakageを探します。

### 対象例

- Semantic Data / Relation / type
- Application Pattern
- Capability Binding
- native Query / SQL
- Database Contract
- Environment Binding
- generated artifact
- Workbench / diagnostic index

### 確認

各情報について:

```text
A irreducible accepted meaning
B deterministic derived lens
C realization / environment
D actual observation / Evidence
E cognitive candidate
```

のどこに属するか説明できること。

### Failure signal

- B/D/Eの情報がauthoritative sourceへ手入力で複製されている
- CのProvider / Target detailがbusiness meaningへ混入している
- Aの意味がderived cacheにしか存在しない

## 5. CDB-P02 — Cold Canonical Reconstruction

### 目的

Canonical / accepted meaningがderived stateへ依存していないことを確認します。

### 操作

Conceptually次を削除または空から再生成します。

- generated indexes
- Workbench projections
- inspection views
- search caches
- embeddings
- FlyHash / HDC等のrepresentation
- AI summaries / trajectories

authoritative sourceとnative authoritative assetは残します。

### 期待

```text
source
  -> parse / normalize / resolve / validate
  -> same accepted meaning
```

へ戻れること。

### Evidence

将来actual runnerを作る場合は、before/afterのsemantic digestやnormalized projectionを比較できます。

ただしdigest値そのものを新しいsemantic authorityにはしません。

## 6. CDB-P03 — Function Lens Reconstruction

### 目的

Functionというreview視点を、二重管理のFunction正本なしに再構成できるか確認します。

### 入力

- Application Pattern / View
- Action / Capability
- Query
- read edge
- create / update / delete edge
- action / function call edge
- input / output

### derived lens例

```text
Function item
  intent / display label
  reads
  creates
  updates
  deletes
  calls
  called-by
  inputs
  outputs
  source trace
```

### Failure signal

- user-visible Function meaningが既存edgeから失われる
- two different business functionsが同じderived structureへcollapseする
- Function-specific accepted meaningを保持する場所がない

このFailureがrealistic Referenceで再現された場合のみ、新しいauthoring surface候補を検討します。

## 7. CDB-P04 — Derived inspection vs authored application intent

### 目的

Dataから便利なViewを大量生成しても、application navigation / user intentを勝手に作らないことを確認します。

### 期待

```text
Data
  -> derived inspection list/detail/reference

accepted user/task intent
  -> authored Pattern / View only when needed
```

### Pressure case

- Data数が増える
- derived inspection Viewが増える
- application画面は業務上必要なsubsetだけ

### Failure signal

- derived Viewが自動的にapplication menuへ露出する
- generated CRUDがBusiness Procedureとして扱われる
- application intentを得るため全Dataに同じView定義を手書きする必要がある

## 8. CDB-P05 — Business meaning preservation

### 目的

Definition Body簡潔化がData-only / Function-onlyへ退行しないことを確認します。

### Referenceに含める

- manual work
- external organization / system work
- actor / responsibility
- business judgment
- Handoff / continuation
- system resultとactualityの不一致
- unresolved realization

### 期待

これらをfake Capability / fake Application Functionへ変換せずにaccepted business meaningとして保持できること。

### Failure signal

「Runtimeで実行できない」ことを理由にbusiness meaningを捨てること。

## 9. CDB-P06 — Semantic Information Conservation

### 目的

shorthand / derivationでmaterial distinctionを失っていないか確認します。

### Pressure examples

- identifier vs text
- exact decimal vs generic number
- date vs instant
- optional vs nullable
- one vs many Relation
- business key vs technical record ID
- read-only vs read-write
- Business Result vs Observation
- actor vs authorization role

### 期待

sourceが短くても、Canonical / accepted meaning側では必要なdistinctionが保持されること。

### Failure signal

source短縮のために型や関係を曖昧化し、後段がheuristicに推測すること。

## 10. CDB-P07 — Local Change / Diff Amplification

### 目的

局所的なsemantic changeが、authoritative source上の局所変更として保てるか確認します。

### Pressure examples

- fieldのtype修正
- Relation cardinality変更
- explicit Viewの一項目変更
- Business Handoff condition修正
- datasource access変更

### 観測候補

- changed authoritative files
- changed authored statements
- unrelated reserialization
- generated artifact churn
- derived index rebuild範囲

### 原則

生成物が大きく変わること自体は失敗ではありません。

問題は、derived changeのためにauthoritative sourceのunrelated部分まで書き戻すことです。

## 11. CDB-P08 — Native asset non-duplication

### 目的

native SQL / Go / external configuration等の内容をTEI sourceへ二重転記せず利用できることを確認します。

### 期待

```text
accepted TEI contract / role
    -> native asset trace
    -> realization
```

### Failure signal

- SQL変更のたびに同じpredicate / projectionを別TEI DSLへ手修正
- Go implementationのI/O意味を複数正本へ手入力
- external configuration全文を「AIに見せるため」Canonicalへコピー

## 12. CDB-P09 — No-AI semantic reconstruction

### 目的

生成AIが利用不能でもTEIのaccepted designが成立することを確認します。

### 期待

- source reload
- validation
- deterministic derived lens
- Driver / Conformanceに必要なmeaning

がAI providerなしで成立すること。

AIは説明・候補生成・探索を強化できますが、semantic decoderにはしません。

## 13. CDB-P10 — No source inflation for cognition

### 目的

associative cognitionやretrievalを導入しても、Definition Bodyへ検索専用metadataを要求しないことを確認します。

### algorithm-neutral feature source

可能な限り次からderiveします。

- Canonical identity / kind
- type family
- Relation role / cardinality
- constraint / lifecycle signal
- Application Pattern / Capability relation
- Data read / CUD / call edges
- source / subsystem / ownership relation
- Reference / Evidence relation

### 禁止する短絡

- FlyHashのために手作業tagを追加する
- embedding精度のため全subjectへ長文descriptionを必須化する
- LLM prompt用annotationをbusiness meaningとして採用する

必要なdomain vocabularyがbusiness meaning自体ならauthorしてよいですが、検索engine convenienceとの違いを説明します。

## 14. CDB-P11 — Cognitive Layer deletion test

### 目的

brain-inspired / embedding / LLM knowledge layerが第二正本でないことを確認します。

### 操作

- cognition index削除
- model-specific cache削除
- trajectory削除
- rebuildまたは別algorithmへ切替

### 期待

Canonical Worldは変化しません。

retrieval orderやcandidate qualityは変化し得ます。その差はcognitive quality Evidenceとして扱います。

## 15. CDB-P12 — Candidate comparison, not answer authority

### 目的

Reference knowledgeをAIが利用するとき、一つの「最適解」をsemantic authorityとして返さないことを確認します。

### candidate representationの最低意味

```text
Candidate
  structure / referenced subjects
  provenance / Reference evidence
  applicability
  incompatibility / downside
  unresolved
  example Dataset / Scenario where useful
```

固定schemaではありません。

### 期待

Candidate A / B / Cを比較し、採用はProposal / Shared Acceptanceへ進むこと。

ranking top-1を自動acceptしません。

## 16. CDB-P13 — Pattern extraction without standard inflation

### 目的

Reference Modelの反復構造をknowledgeとして再利用しても、頻出だけでCore standardへ昇格しないことを確認します。

### 流れ

```text
Reference repetition
  -> derived Pattern candidate
  -> source / usage / variation analysis
  -> applicability / counterexample
  -> Product pressure
  -> explicit adoption only if justified
```

### Failure signal

- 一つのReferenceで頻出したPatternが自動的に`std.*`へ入る
- derived Pattern CardがCanonical source of truthになる

## 17. CDB-P14 — Complexity Rent

### 目的

新しいauthoring概念を追加する際、その複雑さが具体的価値を生むか確認します。

### Concept Rent Record候補

新concept候補ごとに:

```text
Reference defect
existing mechanism limitation
authored burden
new deterministic check enabled
projection / runtime benefit
cross-target/provider relevance
simpler alternative
reopen / removal condition
```

を短く説明します。

これを新しい恒久schemaへはしません。ADR / Issue / design noteで十分なら再利用します。

## 18. CDB-P15 — Definition Body readability

### 目的

機械的に端的でも、人間が業務意味を読めなければX-TEA型の実行可能仕様の生命を失います。

### Review question

- primary business nouns / relationsを追えるか
- explicit differenceが目に入るか
- derived defaultとexplicit overrideを区別できるか
- sourceを読むために内部IR knowledgeを要求していないか
- diagnosticから元sourceへ戻れるか

### 非目標

自然言語ですべてを重複説明することではありません。

## 19. Metric候補 — score化しない

設計変化の観測には次を使えます。

- authored subject / statement count
- derived projection count
- explicit override count
- manual duplicated mapping count
- source-to-derived explanation coverage
- one-change touched-source count
- derived cache rebuildability
- no-AI cold reconstruction result
- cognition index deletion invariance
- realistic Reference defect count

これらを一つの`concisenessScore`へ畳みません。

少ない行数・高いderived比率そのものをKPIにすると、意味を隠す圧力が生まれるためです。

## 20. Referenceの選び方

次の順で使います。

### R0 — current `order_entry`

現在のsmall vertical sliceでClass A〜Eとcold reconstructionの方法を確認します。

### R1 — existing realistic Business Procedure Reference

manual / external / Handoff / cross-viewを含むReferenceで、Data-only圧縮を防ぎます。

### R2 — broader realistic domain

販売・調達・在庫・製造・債権債務等の複数Subsystemにまたがる内容で、Function lens / ownership / cross-subsystem relationを圧迫します。

既存の第三者Reference contentをコピーするのではなく、利用条件に従い、必要ならTEI-original fixtureで構造的pressureだけを再現します。

### R3 — Second Application / Target

第一Applicationだけの偶然を標準にしないため、性質の異なるApplication / Targetで再評価します。

## 21. 段階

### Phase 0 — Architecture / inventory

- [x] Definition Body Architecture
- [x] 本Pressure Plan
- [ ] current sourceのA〜E inventory
- [ ] current derived artifacts inventory

### Phase 1 — deterministic reconstruction pressure

- [ ] CDB-P02 Cold Canonical Reconstruction
- [ ] CDB-P03 Function Lens
- [ ] CDB-P04 inspection vs authored intent
- [ ] CDB-P06 Semantic Information Conservation
- [ ] CDB-P07 Local Change

### Phase 2 — Business / native asset pressure

- [ ] CDB-P05 Business preservation
- [ ] CDB-P08 Native asset non-duplication
- [ ] realistic ReferenceでConcept Rent監査

### Phase 3 — AI / cognition pressure

Gate C後のknowledge-first Referenceと接続します。

- [ ] CDB-P09 No-AI reconstruction
- [ ] CDB-P10 no source inflation
- [ ] CDB-P11 cognition deletion invariance
- [ ] CDB-P12 candidate comparison
- [ ] CDB-P13 Pattern extraction

### Phase 4 — surface adoption decisions

Referenceでmaterial defectが残った概念だけ、Requirement / ADR / authoring syntax候補へ昇格します。

## 22. Gateとの関係

本Pressure PlanはGate B / Gate C Required Evidenceではありません。

現在の実行順は変えません。

```text
Gate B actual Evidence
  ↓
First Target / Gate C
  ↓
Studio / knowledge-first Reference
  ↓
CDB Phase 3 cognition pressure
```

Gate C前でもsource inventoryやstatic derivation reviewはできますが、それを後段actual Evidenceへ読み替えません。

## 23. Decision rule

新しいauthoring surfaceを採用するのは、次が揃った場合だけです。

1. realistic Referenceでmaterial meaning loss / unsafe ambiguity / duplicated managementが再現される
2. existing source / relation / projection / binding / evidence mechanismでは解決しない
3. derived-only approachでは不足する理由が明確
4. additional authored burdenを上回るtestable benefitがある
5. AI / Provider / Target固有の都合ではなくProduct meaningとして残る
6. cold reconstruction / source authorityを壊さない

満たさなければ、sourceを増やさずderived lens側を改善します。

## 24. 完了の意味

この計画の成功はDefinition Bodyが最小行数になることではありません。

成功は:

- accepted meaningが必要十分にauthorされる
- standard / inspectable knowledgeがdeterministically derivedされる
- rich Projectionが第二正本にならない
- Business manual / external meaningが失われない
- AI / cognitionを追加・削除してもauthorityが変わらない
- realistic Referenceから新概念の必要性を判断できる

状態です。
