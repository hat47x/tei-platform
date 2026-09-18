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

である。

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

## 4. SUI責務の初期マッピング

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

## 5. 初期Authority戦略

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

## 6. Baseline Challenge mapping

### Challenge A — hierarchy / placard

Historical SUI PR #235:

- 11 files
- +328 / -10
- meaning / validation / view derivation / UIへ横断

TEIで確認すること:

- parent relationをSemantic Relationとして一度だけ表現できるか
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

## 7. 最初のvertical slice

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

1. SUI sourceから意味をlosslessに読めるか
2. TEI側で同じ意味を一度だけ表現できるか
3. hierarchy等のValidationを共通化できるか
4. file pathやCanvas座標をsemantic identityへ混ぜずに済むか
5. SUIの未対応概念をUnknownとして残せるか

である。

## 8. Plugin discoveryの判定

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

## 9. TEI Coreへ追加しないもの

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

## 10. SUI以外への横展開

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

## 11. 次の実装開始条件

actual adapter / fixture codeへ進む前に、

- Gate B主線を妨げない作業単位である
- SUI baseline fixtureの範囲が固定されている
- source authorityがDocumentV1であることを明示できる
- read-onlyから始める
- Plugin / Coreの新contractを先取りしない
- 実装後に削除可能なReference probeとして隔離できる

ことを確認する。

この条件を満たせない場合は、mapping documentまでで止める。
