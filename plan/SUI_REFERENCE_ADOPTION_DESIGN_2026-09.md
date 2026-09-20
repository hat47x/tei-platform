# SUI Reference Adoption — TEI側の移管設計基線

- Status: Pre-design / Reference Adoption baseline
- Date: 2026-09-18
- Parent Issue: `plan/issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md`
- SUI counterpart: `sui-sensemaking/01_Plans/cross-repo/2026-09-18-sui-tei-reference-adoption-baseline.md`
- Gate impact: None
- Runtime impact: None
- Product Requirement impact: None in this document

## 1. 目的

SUI Sensemakingを、TEIが実際の複雑なApplicationを支えられるかをPressureするReference Adoptionとして扱う。

この文書では、

- 現行SUIのどの責務をそのまま残すか
- どの責務をTEIの既存境界へ寄せる候補とするか
- どこがPlugin / Adapter / Capability候補か
- 現時点では何をTEI Coreへ追加しないか

を整理する。

最初からSUIをTEI-native schemaへ全面変換することはしない。

> **encapsulation before translation**

を初期戦略とする。

## 2. 現在のTEI能力と制約

2026-09-18時点のcurrent mainでは、少なくとも次が存在する。

- Canonical Semantic Layer
- Application Pattern
- Validation
- Capability Binding
- Plugin / Bundle / Plugin Lock
- process Plugin runtime
- Permission / Resource境界
- Shared Acceptance BoundaryのArchitecture設計
- Target-neutral Presentation / Interaction要件
- Source Index / Provenance
- Reference / Conformance

一方、現在のProduct statusは、

- Gate A: Ready
- Gate B: Not Ready
- Gate C: Blocked
- First Web Target: 未実装 / Evidence未取得
- Studio / Shared Acceptance実装: Design Only

である。（2026-09-21注記: 起票時点の記述。Gate BはR = `e5231839`、Gate CはT = `0ba87a4e`で2026-09-19にReadyとなり、第一Web Targetも実装・Evidence取得済みである。判定は[Gate C Readiness Audit](GATE_C_READINESS_AUDIT.md)）以降の「当面」の順序は起票時点の判断であり、Gate C後の並行lane（`ISSUE-000046` / `000047` / `000027` / `000026`）に合わせて見直す。

したがって、SUI Reference Adoptionは当面、

- mapping
- Reference設計
- fixture / challenge準備
- Plugin boundary探索
- native asset境界の確認

を先に行う。

Gateを飛び越えてWeb TargetやStudioをproduction-readyとして実装したことにはしない。

## 3. 現行Plugin機構についての重要な観測

current `internal/plugin/model.go` の `CapabilityKind` は、現時点では主に次を扱う。

- deployment-executor
- observation-provider
- evidence-provider
- database-provider
- contract-query-executor
- migration-provider
- editor-intelligence

つまり、**「spatial canvas plugin」「presentation plugin」といった汎用UI拡張種別はまだ契約として存在しない。**

これは不足が確定したという意味ではない。

CanvasをTEIのPluginとして扱うには、まず次を比較する必要がある。

1. Target Adapter / native componentとして十分か
2. Application Pattern / Projection境界で扱えるか
3. Capability Pluginとして表現すべき実行能力があるか
4. presentation / interaction extensionという新しい共通境界が本当に必要か

SUI一件だけを根拠に新しい`CapabilityKind`を追加しない。

## 4. Canonical definitionとSUI instance dataを混同しない

SUI移管で最も重要な境界の一つは、TEI Canonical Worldと、SUI利用者が作る実データを分けることである。

たとえば、

```text
TEI Canonical definition
  sui.card
    id
    text
    x
    y

SUI application instance data
  card-001 = "..."
  card-002 = "..."
```

という関係になる。

**card-001やisland-003そのものをTEI Canonical objectにしない。**

同様に、将来SUIが扱うObservation / Hypothesis / Synthesis等も、SUIというApplicationのbusiness data instanceであり得る。TEIのSemantic Coreに`Hypothesis` kindを追加することとは別である。

```text
SUI domain vocabulary
  -> Application data schema / rule

TEI semantic vocabulary
  -> schemaやApplication intentを表現するためのplatform meaning
```

この分離により、SUIのdomain語彙をTEI Universal Coreへ流入させずに、TEIでSUIを実装できる可能性を評価する。

### 4.1 初回schema mapping仮説

最初のChallenge Aでは、次の最小schemaをPressureする。

#### Card

最低限:

- id
- text
- x
- y

`x / y`はSUI上ではinstance dataだが、sensemaking上の意味そのものではなくProjection / Interaction寄りのdataである。TEI Canonical semantic definitionに属性として見える場合でも、domain identityや業務Truthと同列には扱わない。

#### Island

最低限:

- id
- title
- parent island reference
- placard card reference

ここで、

- parent relation
- placard reference
- member cards

をどこまでSemantic Relationとして定義するかはReferenceで確認する。

特に、`placardCardId`が「SUI domain上の意味」なのか「現在のKJ Canvas Projection上の代表表示」なのかを先に分解する。Projection concernならTEI Canonical business meaningへ上げない。

#### Relation / Edge

初回Challenge Aでは全面対応しない。

現行SUI Edgeは、

- card / islandのheterogeneous endpoint
- relation type vocabulary
- directed / undirected semantics

を持つ。

TEI current `semantic.Relation` はData definition間のrelationであり、SUIの利用者が作る1本のEdge instanceとは役割が異なる。したがってSUI Edge instanceをTEI Semantic Relation objectへ一対一変換しない。

### 4.2 current TEI modelで早期に見えるPressure

current `semantic.Data / Attribute / Relation / Validation`を読む限り、初回SUI sliceは次を明らかにできる。

| SUI要求 | current TEIでの見込み | 初期分類 |
|---|---|---|
| Card / Islandというdata kind | Data definitionで表現候補 | A |
| scalar attribute | Attributeで表現候補 | A |
| parent relation | self Relationでsingle-valued shapeはPressure可能。rootのnullable parent（0..1）意味はcurrent `one | many`だけでは未表現 | A候補 / optionality unresolved |
| placard reference | RelationでPressure可能だが意味分類が先 | A / C |
| member cards | many RelationでPressure可能 | A候補 |
| optional / unionの豊富なfields | current minimal modelでは不足可能性 | C / D未確定 |
| polygon points | specialized instance payload | C |
| arbitrary Edge type | instance-level domain data | C / extension |
| acyclic hierarchy validation | current ValidationKindでは直接不足 | B / C / DをReferenceで判定 |
| placardがmemberである制約 | current ValidationKindでは直接不足 | B / C / DをReferenceで判定 |
| proposal / review state | Application data + Acceptance境界 | A / B |
| DocumentV1全体 | native application asset | Cから開始 |

この表のDは不足確定ではない。current modelで表しにくいという観測だけでProduct Requirementへ上げない。

### 4.3 Stage 0 constraint coverageを分離して記録する

Stage 0では、SUI正本のhierarchy constraintを一括して「TEIで表現できた」と判定しない。

report schema v2の`constraintCoverage`で、少なくとも次を分離する。

- `parent-single-valued`: current Relation `one`でsingle-valued shapeだけを観測する
- `parent-presence`: rootのnullable parentはcurrent Relationから独立に表現できず`unresolved`
- duplicate island / duplicate child
- parent / child existence
- parent-child consistency
- acyclic hierarchy

後者の整合性constraintはSUI正本をReference-local validatorがshadowしているだけであり、TEI native Validationとしては数えない。

この分類を次のReferenceにも持ち込み、同じconstraint familyが複数Applicationで反復して初めてreusable extension / Product gap候補として扱う。SUI一件だけを根拠にValidationKindやRelation optionality fieldを増やさない。

## 5. SUI責務の初期マッピング

| SUI concern | 現行の主な表現 | TEI初期分類 | 理由 |
|---|---|---|---|
| Card / Island / Relation | `DocumentV1`, domain types | A/Cの比較 | 安定意味はTEI Semanticへmap可能性があるが、現行sourceを即廃止しない |
| x/y / polygon geometry | Card座標、Island geometry | C — native候補 | 空間Projectionの実装詳細であり、Canonical business meaningへ直結させない |
| hierarchy / placard | Island parent / placard | A候補 | Data / Relation / Projectionの分離をPressureできる |
| overview / mid / detail | view state | A — Projection候補 | Canonicalとは分離すべきview concern |
| SafeMode | policy / UI / export gate | A/B比較 | Acceptance / policy / Target表示へどう分けるかをPressure |
| merge suggestion | proposal artifact | A候補 | Shared Acceptance Boundaryと親和性が高い |
| merge apply | explicit domain transaction | A/C比較 | generic AcceptanceとSUI固有merge semanticsを分ける |
| Evidence / provenance | EvidenceLink / attribution / export | A候補 | TEI Source / Evidence / Provenance境界をPressure |
| AI provider | LLM adapter / cognitive providers | B/C比較 | TEI editor-intelligenceだけへ押し込まずSEI連携も含める |
| import / export bundle | ZIP / bundle / metadata | B/C比較 | generic Artifact exchangeかSUI固有native assetか |
| Canvas rendering | React canvas components | C → B候補 | まずnative component、再利用性が出た場合のみPlugin化 |
| polygon editor | geometry engine | C | specialized native interactionとして局所化する方が自然 |
| review / approval | ReviewAttribution等 | A候補 | Shared Acceptanceとauthority分離をPressure |
| advanced semantic artifacts | Evidence→Synthesis contract | 保留 | SUI側L0 Planned。移管のためにTEI schemaを先行拡張しない |

A/B/C/DはSUI側の分類に合わせる。

- A: Existing TEI
- B: Reusable extension
- C: Native SUI
- D: TEI Product gap candidate

## 6. 初期Authority戦略

### 5.1 Stage 0

現行`DocumentV1`をauthoritative native assetとして残す。

```text
SUI DocumentV1
  = current application source

TEI projection / adapter
  = derived / reference
```

TEI側へ同一内容の第二正本を作らない。

### 5.2 Stage 1

TEIで安定意味を表現できることが確認できた場合、対象ごとにauthority migrationを検討する。

その際も一度に全Documentを切り替えず、

```text
source owner before cutover
source owner after cutover
migration transform
roundtrip / equivalence evidence
rollback
```

を明示する。

「TEIでも読める」ことと「TEIが正本になった」ことを同一視しない。

このStage 0 / Stage 1のauthority transitionは、TEI一般Architectureの[Native asset adoption / authority migration](../docs/architecture.md)境界に従う。SUI固有のcutover規則を第二のArchitecture正本にはしない。

## 7. Baseline Challenge mapping

### Challenge A — hierarchy / placard

Historical SUI PR #235:

- 11 files
- +328 / -10
- meaning / validation / view derivation / UIへ横断

TEIで確認すること:

- parent relationをSemantic Relationとして一度だけ表現できるか
- `parentIslandId: string | null`の0..1 presence semanticsを、current Relation modelでlosslessに表現できるか。現時点では`one | many`しかなくunresolvedとする
- placard referenceをdomain meaningとして表現する必要があるか、Projection metadataで十分か
- overview / mid / detailをProjectionだけで変えられるか
- validationをsource / UIごとに重複せず一か所へ寄せられるか

最初の目標は「SUIと同じUIを作ること」ではなく、**同じ意味変更の波及箇所を減らせるか**である。

### Challenge B — polygon editing

Historical SUI PR #286:

- 5 files
- +191 / -28

TEIで確認すること:

- polygon interactionをnative componentとしてHostできるか
- componentからTEIのValidation / Action boundaryへ接続できるか
- geometryをCanonical Semantic kindへ昇格させずに済むか
- 第二のspatial Applicationが現れたときだけPlugin surfaceへ一般化できるか

初期仮説:

> polygon editorはC（Native SUI）のままでもTEI Reference Adoptionは成功し得る。

### Challenge C — merge accept / apply

Historical SUI PR #2849:

- 3 files
- +465 / -0

TEIで確認すること:

- proposal decisionとmutationを分離できるか
- stale base / invalid state / authority不足をShared Acceptance側で共通処理できるか
- SUI固有コードを「何をmergeするか」というdomain transactionへ局所化できるか
- provenance / audit / acceptance envelopeをTEI側で共通化できるか

このchallengeは、TEIの共通Application基盤としての価値を最も直接Pressureする。

## 8. 既存SUI契約をStage 0 handoffに再利用する

SUIには既に、`DocumentV1`からIsland hierarchyだけを決定論的に切り出す契約がある。

- `03_Implement/frontend/src/domain/contracts/island_contracts.ts`
- `03_Implement/frontend/src/domain/contracts/island_hierarchy_handoff.ts`

`projectIslandHierarchyContractV1()` は、Documentから次だけを投影する。

```text
schemaVersion
islands[]
  id
  parentIslandId
  childIslandIds
```

このprojectionはCanvas座標、polygon、title、AI state等を含まない。

またSUI側validatorは少なくとも次を検査する。

- duplicate island id
- duplicate child reference
- missing parent
- missing child
- parent / child mismatch
- cycle

既存testのM1〜M4は、

- M1 root only — pass
- M2 3-level hierarchy — pass
- M3 missing parent — fail
- M4 cyclic hierarchy — fail

を固定している。

TEI側で同じcontractをshadow validationする場合、SUI正本validatorに存在しないdomain constraintを追加しない。local diagnostic codeやconsumer-side JSON compatibility checkは追加できるが、source contractのpass / fail意味とconsumer transport policyを分離する。

```text
source contract validity
  != consumer compatibility guard
```

Stage 0で両者を混ぜて「TEIの方が厳しいので正しい」としない。差が必要ならSUI source contract変更、TEI consumer policy、TEI Product gapのどれなのかを明示する。

### Stage 0 provenance / freshness

cross-repository Referenceでは、単に`SourceContract = IslandHierarchyContractV1`と書くだけでは、後からどのsource実装をshadowしたか再現できない。

Stage 0ではSUI repository revisionをnavigation provenanceとして記録し、freshness dependencyはactual contract / handoff / test blob SHAへ束縛する。

```text
repository revision
  = navigation provenance

contract / handoff / test blobs
  = actual Stage 0 source dependencies
```

したがってSUI repositoryの無関係なcommitだけでStage 0をstaleにしない。一方、依存blobが変化した場合は、fixture / shadow validator / mapping assumptionを再確認するまでcurrent-source conformanceを主張しない。

これはDesign Synthesis R6 / Derived Cognitionで採用したselective freshnessと同じ方向であり、cross-repositoryだからrepository-wide revision counterへ戻さない。

### Stage 0の入力をこの契約にする理由

移管専用の新しいJSON schemaを先に作ると、SUIとTEIの間に第三の意味表現を増やす。

そこで最初のprobeでは、SUIがすでに利用している`IslandHierarchyContractV1`を**handoff projection**として再利用する。

```text
DocumentV1
  ↓ SUI-owned deterministic projection
IslandHierarchyContractV1
  ↓ read-only
TEI Reference probe
```

この時点ではTEIがDocumentV1 parserを所有しない。

これにより、

- SUI source contractの解釈はSUI側に残る
- TEIはhandoffされた意味を受け取ることに集中できる
- full DocumentV1との密結合を避けられる
- Stage 0が失敗してもproduct runtimeへ影響しない

という性質を持たせる。

### placardはStage 0へ混ぜない

`IslandHierarchyContractV1`には`placardCardId`が含まれない。

Challenge A全体ではplacardも対象だが、最初のread-only probeではhierarchyだけを扱う。

hierarchyでTEIのData / Relation / Validation境界を確認した後に、

- existing SUI contractを拡張すべきか
- 別projectionにすべきか
- placardがdomain meaningかProjection metadataか

を決める。

移管都合だけでSUI既存contractへfieldを追加しない。

## 9. 最初のvertical slice

Gate C前にWeb UIの完成を要求しない。

最初のReference sliceは、次の論理sliceを推奨する。

```text
SUI native fixture
  Card
  Island(parent, placard)
  Relation
  review / proposal state
        ↓
SUI Source Adapter candidate
        ↓
TEI read-only semantic/application projection
        ↓
Validation / diagnostics
        ↓
derived Reference output
```

ここでは、

- sourceを変更しない
- Canvasを再実装しない
- Web Targetを実装しない
- Plugin kindを追加しない

ことを意図的な制約とする。

得たいEvidenceは、

1. SUI sourceから意味をlosslessに読めるか。Stage 0ではparent optionalityが未表現なため、現時点でlossless達成とはしない
2. TEI側で同じ意味を一度だけ表現できるか
3. hierarchy等のValidationを共通化できるか
4. file pathやCanvas座標をsemantic identityへ混ぜずに済むか
5. SUIの未対応概念をUnknownとして残せるか

である。

## 10. Plugin discoveryの判定

SUI固有componentをPlugin候補にする条件:

1. Application meaningとcomponent内部stateを分離できる
2. explicit capability / input / output / permission boundaryを持てる
3. TEI Core kindを増やさなくても接続できる
4. 第二Applicationで同種の需要を再現できる
5. PluginがなくてもCanonical meaningを読み直せる

### 初期候補

#### Spatial Canvas Host

候補責務:

- nodes / groups / typed relationsの描画
- pan / zoom
- drag / resize
- selection
- optional direct-manipulation geometry
- semantic IDとの明示binding

非責務:

- SUI Islandの意味
- KJ法
- SUI review authority
- AI grouping

#### Review / Diff Visualization

候補責務:

- proposal before / after
- semantic diff visualization
- objection / hold display
- accepted / rejectedの表示

TEI Shared AcceptanceとStudioで一般需要が確認できる場合に有力。

#### Offline Artifact Exchange

候補責務:

- bundle manifest
- deterministic packing
- import validation
- provenance-preserving exchange

ただしTEI自身のArtifact概念との重複を先に確認する。

## 11. TEI Coreへ追加しないもの

Reference開始時点では次を追加しない。

- `Island` Canonical kind
- `Card` Canonical kind
- polygon primitive
- KJ relation vocabulary
- SUI SafeMode専用primitive
- SUI review state専用primitive
- spatial-canvas CapabilityKind
- sensemaking Artifact kind一式

必要ならSUI source / metadata / extension / native assetとして保持する。

## 12. SUI以外への横展開

SUIでReference Adoptionが成立した後、次の順で知見を再利用する。

### SEI

TEI-first候補:

- Work / Interaction Surface
- Review / Resume Projection
- Evidence / Governance UI
- setting / policy editor

native維持:

- Semantic Core
- Cognitive Method
- model / classifier / associative runtime

### EKI

TEI-first候補:

- operator console
- Task / Result / UNKNOWN projection
- placement / budget policy editor
- Contribution / Evidence review

native維持:

- scheduler
- worker
- lease / retry
- sandbox
- protocol
- resource governor
- verification kernel

## 13. 次の実装開始条件

actual adapter / fixture codeへ進む前に、

- Gate B主線を妨げない作業単位である
- SUI baseline fixtureの範囲が固定されている
- source authorityがDocumentV1であることを明示できる
- read-onlyから始める
- Plugin / Coreの新contractを先取りしない
- 実装後に削除可能なReference probeとして隔離できる

ことを確認する。

この条件を満たせない場合は、mapping documentまでで止める。
