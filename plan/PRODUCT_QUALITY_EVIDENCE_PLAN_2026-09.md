# TEI Product Quality Evidence Plan — 2026-09

Status: **Planning baseline — incremental / no new Gate**

関連:

- `requirements/TEI_PRODUCT_REQUIREMENTS.md` — RQ-027〜RQ-030
- `docs/product-quality-architecture.md`
- `plan/MASTER_PLAN.md`
- `plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- GitHub Issue #565

## 1. 目的

Product Qualityを「最後にproduction monitoringを足す作業」へ追いやらず、現在のReference / Gate / Target / Studio / AI workstreamへ段階的に接続します。

一方、品質項目を増やすたびにGate、Evidence lane、tool、Issueを増殖させることも避けます。

本計画の原則は次です。

```text
Requirement
    ↓
existing design boundary
    ↓
small Reference / Pressure
    ↓
context-bound Evidence
    ↓
Quality Envelope
    ↓
Product claim / unresolved
```

Quality EnvelopeはEvidenceの読み方を揃えるためのderivedな概念であり、固定schemaやCanonical kindの先行実装を要求しません。

## 2. Product Qualityを直列blockerにしない

現在のProduct主線は変えません。

```text
Gate B actual Evidence
   ↓ Ready only
Handover B / R
   ↓
First Target T1
   ↓
Gate C
   ↓
Studio / AI / Second Target
```

RQ-027〜RQ-030を追加したことを理由に、Gate Bへ第四Required Evidence laneを追加しません。

Gate Bのcurrent blockerは引き続きactual executionです。quality planningはその環境待ちと並行して進められますが、planningをactual Evidenceへ格上げしません。

## 3. Evidence設計の原則

### 3.1 quality claimを単独値にしない

悪い例:

```text
p95 = 120ms
accessible = true
reliable = true
safe migration = true
```

採用する形:

```text
subject
context
scenario / precondition
property
measure / probe
acceptance rationale
artifact / source
status
```

### 3.2 measureとacceptanceを分ける

観測値と合否規則を同一視しません。

```text
measurement = 140 ms
acceptance threshold = 200 ms
rationale = touch-terminal reference scenario
```

thresholdがまだない場合、観測値を保存してbaselineにできますが、勝手にpassへしません。

### 3.3 actual / synthetic / generatedを混ぜない

既存Evidence原則をqualityにも適用します。

- synthetic Scenario
- generated artifact
- real-provider fixture
- actual external asset
- interactive Target
- production observation

を相互代替しません。

### 3.4 positiveだけでなくnegative Pressureを持つ

品質境界は正常系だけでは分かりません。

例:

- stale observationでdestructive applyを拒否する
- dependency unavailableをsuccessへ丸めない
- keyboardだけで操作不能な箇所を検出する
- telemetry backend停止時もbusiness meaningが失われない
- AI candidateが外れてもauthorityへ昇格しない

## 4. Phase Q0 — Requirements / Architecture baseline

現在のincrementです。

### 完了対象

- [x] RQ-027 Failure / degraded / unknown diagnosability
- [x] RQ-028 Change impact / compatibility / destructiveness
- [x] RQ-029 Target-conditioned operability / accessibility
- [x] RQ-030 Context / Measure / Evidence-bound quality claim
- [x] `docs/product-quality-architecture.md`
- [x] 本Evidence Plan
- [ ] Product Requirement Gap AuditへRQ-025〜030を統合
- [ ] Master Planへcross-cutting qualityを統合
- [ ] PAS traceからRQ-027〜030へ到達できることを確認

このPhaseに実行Evidenceはありません。

## 5. Phase Q1 — Gate Bから再利用できるquality Evidence

Gate Bを拡張するのではなく、既存Evidenceがすでに何を示すかを明示します。

### RQ-027 diagnosability

既存の主な材料:

- Business Result / Execution Error / Observation分離
- source / validation Diagnostic
- repository source observation failure
- secret-safe PostgreSQL error boundary
- runtime unknown / no-auto-replay設計

Q1で必要なのは、これらを一つのgeneric logging subsystemへ統合することではありません。

まず各Evidenceから:

- failure nature
- safe logical identity
- source / runtime boundary
- secret non-disclosure

を説明できるか監査します。

### RQ-028 change safety

既存の主な材料:

- Desired / Actual / Diff / Unresolved
- generated artifact check-only / repair分離
- PostgreSQL migration readiness
- destructive `DropTable` precondition / fresh observation
- Shared Acceptanceのstale base設計

Q1ではDatabase実装を全change typeのgeneric protocolへ昇格しません。

### Q1完了条件

- [ ] Gate B既存EvidenceのどこがRQ-027 / RQ-028を部分実証するかtraceできる
- [ ] secret-safe diagnosticを壊すquality instrumentationを追加していない
- [ ] actual Gate B成功前にqualityをStrong Evidence扱いしていない

Q1はGate B Readyの追加条件ではありません。

## 6. Phase Q2 — First Web Target / Gate C

First Targetは、qualityを利用者interactionへ初めて接続する重要なReferenceです。

### 6.1 Evidenceを一つのscoreへまとめない

Gate C既存の:

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence

を維持します。

quality追加で、これらを置き換える単一`qualityScore`を作りません。

計画上は少なくとも次を追加観測できるようにします。

4. Interaction / Operability profile
5. Accessibility checks
6. Performance / resource measurement
7. Failure / recovery observation
8. Quality Envelope provenance

Gate C contractへどこまで直接入れるかは、First Target着手時にEvidence artifactの重複を監査してから決めます。

### 6.2 Reference Scenario単位

候補:

| Scenario | Semantic | Functional | Operability / accessibility | Performance | Failure / recovery |
| --- | --- | --- | --- | --- | --- |
| browse | same meaning | list usable | focus / navigation | query+render | provider unavailable |
| record open | same data | open usable | focus order | open latency | missing / error |
| reference lookup | same relation | lookup usable | keyboard / state | lookup latency | 0件 / provider error |
| header-detail | same relation | detail usable | dense table navigation | render / rows | partial failureなし |
| validation | same rule | prevent invalid action | error association / recovery | validation latency | diagnostic separation |
| authorization | same policy | deny correctly | denial understandable | auth latency | system errorと区別 |
| command | same capability | invoke once | pending / result state | response latency | timeout / unknown |
| observation | same observed subject | completion visible | state clarity | completion latency | unresolved保持 |

この表は固定artifact schemaではなく、Evidence抜け漏れを発見するplanning matrixです。

### 6.3 Web accessibility

First Web Targetでは適用可能なWCAG 2.2 criteriaを外部referenceとして使います。

最初から全適合claimを目標にせず、Reference Scenarioで価値が高い項目からEvidence化します。

優先候補:

- keyboard operation
- focus visibility / focus order
- form / validation association
- status / error communication
- non-color-only communication
- target size / pointer interaction where applicable
- reflow / zoom where applicable

具体criterion番号は実UIと採用technologyが決まった後、official WCAG 2.2本文へ戻って確定します。記憶や二次資料だけでcriterionを固定しません。

### 6.4 performance baseline

RQ-024 / RQ-030に従い、まずbaselineを測ります。

候補の記録:

```text
B / R
T1
Target / Host
Runtime / Provider
Scenario
measurement point
warm / cold condition
observed distribution
resource usage where useful
```

一律`sub-second`を全Scenarioへ要求しません。

業務端末等で短応答がProduct Valueに直結する場合は、Reference実測と利用条件からthresholdを後で決めます。

### Q2完了条件

- [ ] `B / R / T1`へ戻れる
- [ ] Semantic / Functional / Visual / Operabilityを混同しない
- [ ] applicable accessibility Evidenceがある
- [ ] failure / degraded / unknownの少なくとも一つをTargetで再現する
- [ ] performance baselineがTarget / Host / Scenario条件付きで取得される
- [ ] quality claimがEvidence artifactへ戻れる

## 7. Phase Q3A — Studio / Authoring quality

Studioでは「機能がある」より、変更を安全に扱えることを重視します。

### 重点Scenario

- Select
- Propose
- Preview Semantic Diff / impact
- Accept
- Reject
- stale base conflict
- reload / Conformance

### quality Pressure

#### non-mutating preview

Preview / Rejectでauthoritative sourceを変えない。

#### stale change

base revisionが古いProposalを最新版へ黙ってapplyしない。

#### recovery

write途中失敗時に、成功したことにしない。partial stateがあり得る場合は明示する。

#### interaction

high-density UIでもkeyboard navigation、focus、selection context、diagnostic locationを失わない。

#### diagnosability

Acceptance failureからsource / Canonical subject / violated contractへ戻れる。

### Q3A完了条件

- [ ] Preview / Reject non-mutating Evidence
- [ ] stale conflict Evidence
- [ ] accepted sourceをStudioなしでreloadできる
- [ ] Studio操作のoperability / accessibility profile
- [ ] failure時にauthority状態を説明できる

## 8. Phase Q3B — AI / derived cognition quality

AIとderived cognitionでは「正答率」をSemantic correctnessへ直結させません。

### qualityの分離

```text
retrieval / candidate quality
        !=
semantic applicability
        !=
accepted decision
```

### candidate mechanismで測る候補

- candidate recall
- irrelevant candidate rate
- unknown / no-candidate behavior
- ranking stability / determinism where expected
- latency
- memory / index size
- rebuild cost
- external model / service cost where applicable

### failure Pressure

- relevant candidateを返せない
- irrelevant candidateが上位に来る
- indexがstale
- LLM / external service unavailable
- derived index削除

いずれの場合もCanonical Worldを失わず、candidateをFactへ格上げしません。

`DERIVED_COGNITIVE_EXPLORATION_OVERVIEW_2026-09.md`等の研究計画は、このQuality Envelopeを評価時に利用できますが、研究結果をRQへ自動昇格しません。

### Q3B完了条件

- [ ] benchmark corpus / revisionへ戻れる
- [ ] candidate qualityとsemantic acceptanceを分離できる
- [ ] latency / resource / cost contextを記録できる
- [ ] derived mechanism停止時もCanonical operationが成立する範囲を説明できる

## 9. Phase Q3C — Second Target comparison

Second Targetでは機能差だけでなくQuality Envelope差を比較します。

```text
Handover B / R
      ├─ T1 Quality Envelopes
      └─ T2 Quality Envelopes
```

比較候補:

- interaction modality
- latency / startup
- resource use
- failure presentation
- accessibility / platform integration
- trust / session / Resource境界
- observation method

T1とT2で同じ数値を要求することが目的ではありません。

同じApplication meaningについて、Targetごとにどのquality conditionが必要かを説明できることが目的です。

## 10. Phase Q4 — Production / operational maturity

Milestone 9へ近づく段階で扱います。

### operational telemetry

必要に応じてOpenTelemetry semantic conventions等へmappingします。

取得候補:

- traces
- metrics
- logs
- events
- resource / deployment identity

ただし、Operational Telemetry ProjectionはCanonical / Business Resultのauthorityではありません。

### compatibility / upgrade

- version compatibility policy
- migration path
- deprecated contract
- upgrade / downgrade制約
- rollback不可条件

### security / supply chain

NIST SSDF 1.1等をprocess checklistとして利用し、次を段階的にEvidence化します。

- build / artifact provenance
- dependency / vulnerability handling
- secret handling
- signed artifact / registry where required
- release integrity

### production claim

production-readyを単一booleanへしません。

最低でも対象範囲、Host / Provider条件、既知limit、operational Evidenceへ戻れる必要があります。

## 11. Quality Evidence Matrix

| Requirement | 現在の主Evidence | 次の主Evidence |
| --- | --- | --- |
| RQ-027 | Result/Error/Observation分離、source diagnostic、runtime recovery設計 | First Target failure/recovery、operational correlation |
| RQ-028 | Desired/Actual/Diff、migration safety、stale-base設計 | Studio impact preview、cross-domain compatibility |
| RQ-029 | high-density方針、Gate C Interaction/Visual分離 | First Web Target accessibility/operability |
| RQ-030 | source-fixed Evidence、B/R/T identity、RQ-024 context方針 | actual Quality Envelope records |

関連する既存RQも同じEvidenceから圧力試験されます。新RQ専用toolを必須にしません。

## 12. Metric discipline

### 12.1 metric theaterを避ける

測りやすい値をProduct Valueの代理にしません。

例:

- test count
- line coverage
- number of spans
- Lighthouse score
- AI benchmark top-k accuracy
- issue close数

これらは有用な局所指標になり得ますが、単独でProduct Requirement成立を証明しません。

### 12.2 baseline before threshold

根拠がない領域では:

```text
measure
  ↓
understand distribution / failure mode
  ↓
relate to user / operational need
  ↓
set threshold if needed
  ↓
re-measure / regression
```

とします。

### 12.3 averageだけを使わない

性能等では必要に応じてdistribution、tail、cold start等を見ます。

ただしsample数や統計手法もContextの一部であり、小さいReferenceで過剰な統計精度を装いません。

## 13. Issue / tool増殖を防ぐ規則

RQ-023のmaintainabilityを守るため、quality concernごとに自動的にIssue / toolを作りません。

新Issueを分離するのは、少なくとも次のいずれかがある場合です。

- 独立したfailure mode
- 別のauthority / safety boundary
- 別のactual environmentが必要
- 別のacceptance decisionが必要
- 実装差分を独立reviewした方が安全

同じScenarioの別metricだけなら、既存Evidence artifactへ追加できないか先に検討します。

## 14. Stop conditions

品質改善の名目でも、次の場合はそのincrementを停止して上位設計へ戻ります。

- Canonicalとquality telemetryのauthorityが曖昧になる
- quality schemaのためにTarget / Provider固有概念をCoreへ持ち込む
- arbitrary thresholdを満たすためにScenarioを歪める
- instrumentationがsecret / raw business dataを広げる
- quality tool自体が再現性のない第二正本になる
- 一つの総合scoreでunresolvedを隠す
- Gate B / Gate CのEvidence natureを曖昧にする

## 15. 外部reference

外部モデルはchecklist / evaluation referenceとして利用し、準拠claimは別判断とEvidenceを必要とします。

- ISO/IEC 25010:2023  
  https://www.iso.org/standard/78176.html
- WCAG 2.2  
  https://www.w3.org/TR/WCAG22/
- OpenTelemetry Semantic Conventions  
  https://opentelemetry.io/docs/specs/semconv/
- NIST SP 800-218 Secure Software Development Framework 1.1  
  https://csrc.nist.gov/pubs/sp/800/218/final

## 16. 現在の次作業

Product mainlineの優先順位は変えません。

1. Gate B actual Evidenceを閉じる
2. Handover B / Rを固定する
3. First Web Target T1着手時にQ2 matrixを具体化する
4. Gate C Evidence contractと重複しない最小Quality Envelopeを決める
5. First Target実測をもとにRQ-024 / RQ-029 / RQ-030のacceptanceを具体化する
6. Gate C後、Studio / AI / Second Targetへ同じ考え方を適用する
7. production段階でoperational telemetry / compatibility / supply-chainへ広げる

環境blocker中にFirst Target実装を前倒ししません。一方、Quality Requirement / Architecture / Evidence設計は上流契約として先に整備し、実装開始時の後付け品質を避けます。