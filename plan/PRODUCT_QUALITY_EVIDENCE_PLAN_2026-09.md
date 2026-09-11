# TEI Product Quality Evidence Plan — 2026-09

Status: **Planning baseline — incremental / no new Gate**

関連:

- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `docs/product-quality-architecture.md`
- `plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- GitHub Issue #565

## 1. 目的

Product Qualityを最後のproduction monitoringへ先送りせず、現在のReference / Gate / Target / Studio / AIへ段階的に接続します。

ただしquality concernごとに新しいRequirement、Gate、PAS、Evidence lane、toolを増殖させません。

```text
existing Product Requirement
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

Quality EnvelopeはEvidenceの読み方を揃えるderived conceptであり、固定schemaやCanonical kindではありません。

## 2. Product主線は変えない

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

Quality planningを理由にGate B Required Evidenceを増やしません。

Gate Bのcurrent blockerはactual executionです。planning / static reviewをactual Evidenceへ格上げしません。

## 3. Requirement mapping

新しい番号付きRequirementは追加せず、qualityを既存Requirementの横断Evidenceとして扱います。

| Quality concern | 主な既存Requirement |
| --- | --- |
| meaning stability | RQ-001 / 002 / 003 / 006 / 011 / 017 / 021 / 026 |
| change safety | RQ-002 / 006 / 017 / 020 / 021 / 022 |
| execution / diagnosability | RQ-018 / 019 / 020 / 021 / 022 |
| interaction / accessibility | RQ-010 / 011 / 013 / 021 / 024 |
| performance / resource | RQ-021 / 022 / 023 / 024 |
| security / authority | RQ-003 / 008 / 019 / 020 / 021 / 022 |
| maintainability | RQ-002 / 003 / 017 / 021 / 023 |
| business completeness | RQ-025 / 026 |

このmappingはcoverage scoreではありません。

## 4. Evidence原則

### 4.1 quality claimを単独値にしない

悪い例:

```text
p95 = 120ms
accessible = true
safe = true
```

最低限、必要な範囲で次へ戻れるようにします。

```text
subject
revision / build
Target / Host / Provider / Runtime
Scenario / workload / precondition
quality property
measure / probe
acceptance rationale
Evidence reference
status
```

### 4.2 measureとacceptanceを分ける

観測値があることと、合格条件を満たしたことは別です。

thresholdがまだない場合はbaselineとして保持できますが、勝手に`passed`へしません。

### 4.3 Evidence natureを混ぜない

- deterministic / synthetic
- generated artifact
- real-provider fixture
- actual external asset
- interactive Target
- production observation

を相互代替しません。

### 4.4 negative Pressureを持つ

正常系だけでquality boundaryを証明しません。

例:

- stale observationでdestructive applyを拒否する
- dependency unavailableをsuccessへ丸めない
- keyboardだけでは操作不能な箇所を検出する
- telemetry backend停止時もbusiness meaningが失われない
- AI candidateが外れてもFact / authorityへ昇格しない

## 5. Q0 — Architecture baseline

本Issue #565の範囲です。

完了条件:

- [x] Product Qualityをcross-cutting architectureとして定義
- [x] Quality Envelopeをderived Evidence conceptとして定義
- [x] failure / change / interaction / performance / AI / telemetryの境界を整理
- [x] 外部quality modelを第二正本にしない方針を定義
- [x] 既存Requirementへのmappingを定義
- [ ] Product Requirement Gap AuditをRQ-025 / RQ-026までcurrent化

Q0にactual execution Evidenceはありません。

## 6. Q1 — Gate B既存Evidenceのquality監査

Gate Bを拡張せず、既存Evidenceが何を示すかを明示します。

主な材料:

- Business Result / Execution Error / Observation分離
- Source / Validation Diagnostic
- Desired / Actual / Diff / Unresolved
- generated artifact check-only / repair分離
- real PostgreSQL fixture safety / cleanup
- repository source / revision observation
- Resource / secret boundary
- no-auto-replay / unknown保持

Q1では:

- failure nature
- safe logical identity
- source / runtime boundary
- stale / unresolved handling
- secret non-disclosure

を説明できるか監査します。

Q1はGate B Readyの追加条件ではありません。

## 7. Q2 — First Web Target / Gate C

Gate Cの既存Evidence分離を維持します。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence

これらを単一`qualityScore`へ統合しません。

First Targetでは追加で次を観測します。

- operability / recovery
- applicable accessibility checks
- performance / resource baseline
- failure / degraded / unknown behavior
- Quality Envelope provenance

### Reference Scenario matrix

| Scenario | Functional | Operability / accessibility | Performance | Failure / recovery |
| --- | --- | --- | --- | --- |
| browse | list usable | focus / navigation | query + render | provider unavailable |
| record | open usable | focus order | open latency | missing / error |
| reference | lookup usable | keyboard / state | lookup latency | 0件 / provider error |
| header-detail | relation usable | dense-table navigation | rows / render | partial failureを隠さない |
| validation | invalid action blocked | error association / recovery | validation latency | developer diagnosticと分離 |
| authorization | deny correctly | denial understandable | decision latency | system errorと区別 |
| command | invoke once | pending / result state | response latency | timeout / unknown |
| observation | actual state visible | state clarity | completion latency | unresolved保持 |

この表はplanning matrixであり固定artifact schemaではありません。

### Web accessibility

適用可能なWCAG 2.2 criteriaを外部test referenceとして利用できます。

まずReference Scenarioで価値が高い:

- keyboard operation
- focus visibility / order
- validation / error association
- status communication
- non-color-only communication
- zoom / reflow / pointer interaction where applicable

等からEvidence化します。

全適合claimは実Evidenceなしに宣言しません。

### performance baseline

RQ-024に従い、まず:

```text
B / R
T1
Target / Host
Runtime / Provider
Scenario / workload
warm / cold condition
measurement point
observed distribution
resource use where useful
```

を記録します。

全Scenarioへ一律`sub-second`等を要求しません。

## 8. Q3A — Studio / Authoring quality

重点Scenario:

```text
Select
 -> Propose
 -> Preview
 -> Accept / Reject
 -> reload / Conformance
```

Pressure:

- Preview / Rejectはnon-mutating
- stale baseを黙ってapplyしない
- partial write / failureをsuccess扱いしない
- high-densityでもselection / focus / diagnostic locationを失わない
- Acceptance failureからsource / subject / violated contractへ戻れる

主にRQ-002 / 010 / 017 / 021 / 023を圧力試験します。

## 9. Q3B — AI / derived cognition quality

次を分離します。

```text
candidate quality
   != applicability
   != accepted decision
   != execution authority
```

候補mechanismで測れるもの:

- recall / missed candidate
- irrelevant candidate
- no-candidate / unknown behavior
- determinism / stability where expected
- latency
- memory / index size
- rebuild cost
- external model / service cost

Failure Pressure:

- relevant candidateを返せない
- irrelevant candidateが上位
- index stale
- LLM / external service unavailable
- derived index全削除

いずれでもCanonical Worldを失わず、candidateをFactへ格上げしません。

## 10. Q3C — Second Target

Second Targetでは第一Targetで得たquality仮説を移植せず、同じmeaning baselineに対して再測定します。

比較軸:

- Target Capability
- Execution / Trust
- interaction modality
- Semantic / Functional / Visual Evidence
- operability / accessibility
- performance / resource
- failure / recovery

第一Targetのthresholdが第二Targetにも妥当かはEvidenceから判断します。

## 11. Q4 — Provider / production boundary

production maturityでは次を段階的にEvidence化します。

- operational telemetry correlation
- durable store / HA
- recovery / takeover
- compatibility / upgrade
- signed artifact / provenance
- dependency / supply-chain
- secret / credential operation

OpenTelemetry / NIST SSDF等を利用しても、それらへの形式準拠だけでTEI Product Requirement成立とは扱いません。

## 12. Quality Envelopeの最小評価表現

領域ごとに異なるartifactを使ってよいですが、review時には少なくとも次を説明できることを目指します。

```text
claim
  subject: ...
  context: ...
  property: ...
  observation: ...
  acceptance: ...
  evidence: ...
  status: observed-pass | observed-fail | unresolved | not-applicable | not-observed
```

この表現を今すぐCanonical schemaへしません。

## 13. 完了判定

Product Qualityは一度で`Done`になるworkstreamではありません。

#565の完了は:

- quality concernが既存Requirementへ整理される
- cross-cutting Architectureがある
-段階的なEvidence計画がある
- 新Gate / 新Requirement / 第二正本を不要に増やしていない

ことまでです。

actual quality claimは各Reference / Target / RuntimeのEvidence取得時に更新します。
