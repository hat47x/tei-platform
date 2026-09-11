# Product Requirement Gap Audit

- 監査日: 2026-09-11
- 対象: RQ-001〜RQ-026
- 上流Trace: `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- Quality設計: `../docs/product-quality-architecture.md`
- Quality実証計画: `PRODUCT_QUALITY_EVIDENCE_PLAN_2026-09.md`

## 1. 目的

この文書は、`requirements/TEI_PRODUCT_REQUIREMENTS.md`の各Requirementについて、現在のRepositoryにどのEvidenceがあり、どの不確実性が残っているかを逆マッピングするcurrent auditです。

実装量、Issue close数、文書数ではなく、次を見ます。

- semantic boundaryが成立しているか
- authoritative sourceへ戻れるか
- Reference / Conformanceで再現できるか
- actual execution Evidenceが必要な場合に取得済みか
- synthetic / generated / real-provider / actual-external / Target Evidenceを混同していないか
- Provider / Target / Host / build条件を説明できるか
- failure / degraded / unknownを成功へ読み替えていないか
- unresolvedを成功へ丸めていないか

Product Qualityは新しいRequirement群として追加せず、既存RQを横断するEvidence観点として監査します。

## 2. 評価区分

| 区分 | 意味 |
| --- | --- |
| **Strong Evidence** | 実装・Reference・Conformance / Evidenceが一続きで存在し、主要意味境界を説明できる |
| **Partial Evidence** | 重要部分は成立しているが、Requirement全体またはactual / multi-context Evidenceが不足 |
| **Design Only** | Requirement / Architecture / Planは具体化されているが、対応するReference実証はまだない |
| **Not Started** | 対応する設計・Referenceがほぼない |

`Design Only`を実装済み、`Partial Evidence`をProduct claim成立済みとは読み替えません。

## 3. 現在のProduct主線

最大の直列blockerは引き続き**Gate B actual Evidence**です。

```text
Gate A — Ready
   ↓
Gate B — Not Ready
  #310 / #255 / #308 same-R actual Evidence待ち
   ↓
Handover Baseline B / revision R
   ↓
First Web Target build T1
   ↓
Gate C — PAS-04
   ↓
   ├─ Studio minimal loop — PAS-01
   ├─ knowledge-first / derived cognition — PAS-06
   ├─ AI Proposal / Acceptance — PAS-01 / PAS-06
   └─ Second Target — PAS-04 / RQ-024
```

Cross-cutting Product Qualityは新しいGateを挿入しません。

actual Oracle #214 / #371 / #376は重要なPAS-03 / external asset Evidenceとして継続しますが、Gate Bの第四Required laneではありません。

## 4. Requirement別監査

| RQ | 評価 | 主PAS / scope | 現在成立していること | 主な残Gap |
| --- | --- | --- | --- | --- |
| RQ-001 | **Strong Evidence** | PAS-01 | Canonical ID / source path分離、Semantic Data / Relation / Capability、Provider / Target固有実装の隔離 | First / Second Targetを通したmeaning安定性、複数Application再利用 |
| RQ-002 | **Strong Evidence** | PAS-01 | Source Index、Source Diagnostic、native source trace、file move / source trace基盤 | Studio / AIからauthoritative sourceへ戻るE2E |
| RQ-003 | **Strong Evidence** | PAS-01 | native source / derived IR、Evidence / Handover非Canonical、AI / Studio非authority | Studio / AI authoring E2E |
| RQ-004 | **Partial Evidence** | PAS-03 | native SQL、CustomGoBinding、Plugin、legacy auth、external-observed Contract Query | existing batch / API / config、actual external migration Evidence |
| RQ-005 | **Partial Evidence** | PAS-02 | Application DatabaseRequirement、logical Datasource、multi-DB Driver access | multi-DB transaction / failure、actual external E2E |
| RQ-006 | **Strong Evidence** | PAS-03 | Desired / Actual / Diff / Unresolved、PostgreSQL observation、ProviderExpression | Database外のchange / actual-state領域での実証 |
| RQ-007 | **Partial Evidence** | PAS-03 | Provider SPI、driver assembly分離、provider-specific normalization / Runner | actual second Database Provider圧力試験 |
| RQ-008 | **Partial Evidence** | PAS-02 | Semantic Capability、Binding、Environment、Plugin、legacy auth compatibility | builtin / generated / plugin / existingの同一Capability比較 |
| RQ-009 | **Partial Evidence** | PAS-02 | legacy Authentication / Authorization Reference、Gate A、Driver subject継承 | actual migration target、段階置換Evidence |
| RQ-010 | **Design Only** | PAS-01 | Studio非Canonical、high-density方針、Shared Acceptance、`ISSUE-000046` | Gate C後のStudio actual Reference、operability / accessibility |
| RQ-011 | **Partial Evidence** | PAS-04 | Source / Diagnostic / Workbench / Driver / Handover等の複数Projection | First Web Target、Studio authoring、Second Target |
| RQ-012 | **Partial Evidence** | PAS-02 | Driver主要Scenario、Validation / Auth / Command / Observation、#310 runner | Gate B actual #310/#255/#308 same-R Evidence |
| RQ-013 | **Design Only** | PAS-04 | Gate C contract、baseline `R` / Target build `T`分離、Target Capability / Execution-Trust仮説 | First Web Target、Second Target比較 |
| RQ-014 | **Partial Evidence** | PAS-02 | browse / record / reference / header-detail / commandをApplication intentとして実証 | 複数Application / Targetでの再利用Evidence |
| RQ-015 | **Design Only** | PAS-06 | knowledge-first原則、read-only exploration計画、derived cognition研究境界 | Gate C後のexploration actual Reference |
| RQ-016 | **Design Only** | PAS-06 | Proposal / Decision分離、suppression方針、autonomy / authority分離 | Proposal / Decision / suppressionの実UX / Evidence |
| RQ-017 | **Partial Evidence** | PAS-01 | deterministic generated artifact、native source authority、Shared Acceptance設計 | Studio / AI / Target間で二重管理を作らないE2E |
| RQ-018 | **Strong Evidence** | PAS-02 | Business Result / Execution Error / separate Observation read | Target Projectionで同境界を維持するEvidence |
| RQ-019 | **Strong Evidence** | PAS-05 | Permission Grant、Resource Broker / Lease、Sandbox Profile、Host material分離 | production-grade enforcementの拡張 |
| RQ-020 | **Strong Evidence** | PAS-05 | ownership / heartbeat / fencing / durable handle / no-auto-replay / unknown保持 | 複数production Runtimeでの圧力試験 |
| RQ-021 | **Strong Evidence** | cross | Reference Workspace、Conformance、golden、source-fixed runners、Provider Evidence tools | Gate B actual run、Gate C Target Evidence |
| RQ-022 | **Strong Evidence** | cross | architecture probe / actual Evidence / production claimを分離 | v1以後のsupport / compatibility level整理 |
| RQ-023 | **Partial Evidence** | cross | monorepo、Go中心、ADR / Plan / Reference、Provider / Target分離 | Studio / Target / AI追加後の保守量、文書 / Evidence tool増加 |
| RQ-024 | **Design Only** | PAS-04 | First / Second Targetで測る方針、Gate C測定点、R/T trace | actual Target measurement、context-specific threshold |
| RQ-025 | **Partial Evidence** | PAS-07 | Business Action != Application Function、manual / external Action、actor / authority分離、Business Procedure Reference / Pressure | Lane C actual execution、独立second corpus、Canonical identity最終判断 |
| RQ-026 | **Partial Evidence** | PAS-07 | Data / Function / Business cross-view Reference、Handoff identity Pressure、derived Projection原則 | realistic second-case actual Evidence、diagnostic severityの確定 |

## 5. 評価集約

### Strong Evidence — 9

- RQ-001
- RQ-002
- RQ-003
- RQ-006
- RQ-018
- RQ-019
- RQ-020
- RQ-021
- RQ-022

### Partial Evidence — 12

- RQ-004
- RQ-005
- RQ-007
- RQ-008
- RQ-009
- RQ-011
- RQ-012
- RQ-014
- RQ-017
- RQ-023
- RQ-025
- RQ-026

### Design Only — 5

- RQ-010
- RQ-013
- RQ-015
- RQ-016
- RQ-024

### Not Started — 0

## 6. Cross-cutting Product Quality監査

Quality concernを新しいRequirementへ分割せず、既存RequirementのEvidence completenessとして監査します。

| Quality concern | 現在の強み | 主な残Gap |
| --- | --- | --- |
| Meaning stability | Source / Canonical / Projection / Evidence非同一性が強い | multi-Target / Studio / AI E2E |
| Change safety | Desired/Actual/Diff、stale rejection、Provider preconditionがある | Database外を含むchange impact / recovery Evidence |
| Execution / diagnosability | Result / Error / Observation、unknown / no-auto-replayがある | Target / production横断diagnostic correlation |
| Interaction / accessibility | Target-neutral / high-density / Semantic-Functional-Visual分離がある | First Target actual operability / accessibility |
| Performance / resource | RQ-024とR/T traceがある | First / Second Target actual baseline / threshold rationale |
| Security / authority | Permission / Resource / secret境界が強い | effect-time current authority / production enforcement |
| Maintainability | monorepo、derived artifact非authority、deterministic generation | Studio / AI / quality docs追加後の管理負担 |
| Business completeness | RQ-025 / 026、manual / external Action、cross-view Pressure | actual execution / second corpus |

詳細は`docs/product-quality-architecture.md`を参照します。

## 7. Product Gapの再統合

### Gap A — Gate B actual Evidence

主にRQ-005 / 009 / 012 / 014 / 021。

```text
actual full checkout preflight
        ↓
#310 / #255 / #308
        ↓
same clean candidate R
        ↓
#387 final convergence
        ↓
#273 Gate B audit
```

新しいquality laneを追加しません。

### Gap B — First Target

主にRQ-011 / 013 / 018 / 021 / 024。

Gate CではSemantic / Functional / Visual分離を維持しつつ、First TargetのReference Scenarioから:

- operability / accessibility
- failure / recovery
- performance / resource
- context-bound quality claim

を観測します。

### Gap C — safe authoring

主にRQ-002 / 003 / 010 / 011 / 017 / 021 / 023。

```text
Select
 -> Propose
 -> Preview Semantic Diff / impact
 -> Accept / Reject
 -> Shared Acceptance Boundary
 -> authoritative source
 -> reload / Conformance
```

Studioは第二正本になりません。

### Gap D — knowledge-first / AI / derived cognition

主にRQ-014 / 015 / 016 / 017 / 021 / 023。

```text
retrieval candidate
  != semantic applicability
  != accepted decision
  != execution authority
```

Derived Knowledge / index / trajectoryは削除・再生成可能に保ちます。

### Gap E — Business / cross-view

主にRQ-025 / 026。

Data / Function / Businessの異なる視点を同型のCanonical objectへ増やすのではなく、realistic contentで矛盾を相互検査します。

### Gap F — Provider / production breadth

主にRQ-004 / 005 / 007 / 008 / 009 / 019 / 020 / 024。

actual Oracle等のexternal Evidence、second Provider、production operational boundaryを必要な範囲で継続します。

## 8. 現在の優先順位

1. Gate B actual final Evidenceを取得する
2. Gate B ReadyならHandover Baseline `B / R`を固定する
3. First Web Target `T1`でGate C Evidenceとquality baselineを取得する
4. Gate C Ready後、Studio / knowledge-first / AI Proposal / Second Targetを並行する
5. RQ-025 / 026のBusiness Procedure / cross-view actual Evidenceを継続する
6. Provider / production境界を必要なEvidenceとともに広げる

## 9. 維持コスト監査

RQ-023の観点から、上流整理自体が過剰な管理負担にならないようにします。

- current snapshot文書へ履歴を無制限追記しない
- 同じ意味のRequirementを重複追加しない
- 実行手順の正本を複数作らない
- Gate / PAS / Evidence laneを安易に増やさない
- Quality characteristicごとに専用schema / tool / Issueを作らない
- Evidence nature / failure modeが異なる場合だけartifactを分ける

## 10. 結論

現在のTEIは、Application / Driverの主要meaningをactual Evidenceで閉じ、明示Handoverを経てFirst TargetへProjectionする段階です。

Product Qualityはこの主線と競合する別workstreamではなく、**既存RequirementがTarget / Provider / Runtime / Studio / AIへ渡っても壊れないことをEvidenceで確認する横断軸**として扱います。
