# TEI Product Quality Evidence Plan — 2026-09

状態：**段階的な検証計画の基準版。新たな判定段階は設けない。**

関連:

- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `docs/product-quality-architecture.md`
- `plan/evidence/PRODUCT_REQUIREMENT_GAP_AUDIT.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/gate/GATE_B_READINESS_AUDIT.md`
- `plan/gate/GATE_C_EVIDENCE_CONTRACT.md`
- GitHub Issue #565

## 1. 目的

製品品質の検証を本番環境の監視まで先送りせず、参照事例・判定段階・実行先・Studio・AI支援へ段階的に組み込みます。

ただし、品質上の観点ごとに新たな製品要求・判定段階・PAS・必須検証経路・ツールを増やすことはしません。

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

Quality Envelopeは、検証根拠の読み方を揃えるための派生的な概念です。固定スキーマや正本の要素の種類ではありません。

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

品質の検証計画を理由に、判定段階Bの必須検証根拠を増やすことはしません。

本計画を策定した時点では、判定段階Bの阻害条件は実際の実行による検証でした。計画や静的レビューを、実際の実行の検証根拠として扱うことはしません。

## 3. Requirement mapping

番号付きの製品要求は新設せず、品質を既存の要求に共通する検証根拠の観点として扱います。

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

この対応関係は、要求の網羅率を表す点数ではありません。

## 4. Evidence原則

### 4.1 quality claimを単独値にしない

不十分な記述の例：

```text
p95 = 120ms
accessible = true
safe = true
```

少なくとも必要な範囲で、次の情報を追跡できるようにします。

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

値を観測したことと、合格条件を満たしたことは区別します。

合格の基準値が未確定でも、観測値を基準値設定のための記録として保持できます。ただし、根拠なく`passed`とはしません。

### 4.3 Evidence natureを混ぜない

- 決定論的な試験・合成データによる検証
- 生成成果物による検証
- 実際の提供実装と試験用データによる検証
- 実際の外部資産による検証
- 操作可能な実行先での検証
- 本番環境での観測

これらを互いの代わりに使うことはしません。

### 4.4 negative Pressureを持つ

正常系の成功だけでは、品質に関する境界が成立したとは証明できません。

例：

- 古い観測結果に基づく破壊的な変更を拒否する
- 依存先を利用できない状態を成功として扱わない
- キーボードだけでは操作できない箇所を検出する
- 監視データの保存先が停止しても業務上の意味が失われない
- AIの候補が誤っていても、事実や権限のある判断として確定させない

## 5. Q0 — Architecture baseline

この段階はIssue #565の対象範囲です。

完了条件：

- [x] 製品品質を横断的なアーキテクチャとして定義
- [x] Quality Envelopeを検証根拠の派生的な概念として定義
- [x] 失敗・変更・操作・性能・AI・監視の境界を整理
- [x] 外部の品質モデルを第二の正本にしない方針を定義
- [x] 既存の製品要求との対応関係を定義
- [ ] 製品要求のギャップ監査をRQ-025・RQ-026まで更新

Q0には、実際の実行による検証根拠はありません。

## 6. Q1 — Gate B既存Evidenceのquality監査

判定段階Bの条件は拡張せず、既存の検証根拠で何を確認できるかを明示します。

主な検証材料：

- 業務上の結果・実行時エラー・観測の区別
- ソースと検証規則の診断結果
- 期待する状態・実際の状態・差分・未解決事項
- 生成成果物の検査だけを行う処理と修復処理の区別
- 実際のPostgreSQLを用いた試験環境の安全性と後片付け
- リポジトリのソースと版の観測
- 資源と秘密情報の境界
- 自動再実行を避ける条件と不明状態の保持

Q1で確認する事項：

- 失敗の性質
- 安全に扱える論理識別情報
- ソースと実行環境の境界
- 古い情報と未解決事項の扱い
- 秘密情報を開示しないこと

これらを説明できるか監査します。

Q1は、判定段階BのReady判定に加える条件ではありません。

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
