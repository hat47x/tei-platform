# TEI Product Quality Architecture

Status: **Design baseline — cross-cutting / Evidence-driven**

関連は次のとおりです。

- `requirements/TEI_PRODUCT_REQUIREMENTS.md` — Product Requirement正本
- `plan/PRODUCT_QUALITY_EVIDENCE_PLAN_2026-09.md` — 段階的な実証計画
- `plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md` — Requirement別の現在地
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md` — PASとの接続
- GitHub Issue #565

## 1. 位置づけ

TEIにとってProduct Qualityは、意味モデルとは別の「非機能Layer」ではありません。

TEIがすでに要求している次の境界を、Target / Provider / Runtime / Studio / AIまで一貫して守れるかを横断的に確認する設計軸です。

- business meaningとrealizationの分離
- authoritative sourceとderived artifactの分離
- explicit Binding / authority
- Desired / Actual / Diff / Unresolved
- Business Result / Execution Error / Observationの分離
- Reference / Conformance / Evidenceによる再現性
- Permission / Resource / secret境界
- Target-conditioned performance
- Data / Function / Businessのcross-check

このため、本書は新しい番号付きProduct Requirement、Canonical kind、Gate、Runtime subsystemを追加しません。

## 2. なぜ横断Architectureが必要か

Requirementが正しくても、quality concernを個別実装へ分散させると次の短絡が起こり得ます。

```text
Databaseでは安全
  != Targetでも安全

Semantic Conformance PASS
  != 利用者が操作・復旧できる

latency = 80ms
  != 条件を失っても「高速」と主張できる

logが多い
  != business meaningへ診断できる

AI retrieval accuracyが高い
  != semantic correctness
```

したがってqualityは、既存の意味・変更・実行・Interaction・Evidence境界へ同じ問いを掛けます。

## 3. Quality axes

### 3.1 Meaning quality

主な問いは次のとおりです。

- Projection / Target / Provider都合でCanonical meaningが変わっていないか。
- derived artifactが第二正本になっていないか。
- insufficient observationを`no diff`やsuccessへ変換していないか。
- identity解決がfirst-match等の偶然の順序へ依存していないか。

主なRequirement:

- RQ-001 / 002 / 003 / 006 / 011 / 017 / 018 / 021 / 026

### 3.2 Change quality

主な問いは次のとおりです。

- semantic no-op / compatible / incompatible / destructive / unresolvedを区別できるか。
- apply前にimpactとpreconditionを説明できるか。
- planning後にactual stateが変わった場合、古い観測でdestructive operationを実行しないか。
- retry / fallback / CASCADE等で作用範囲を暗黙拡大していないか。
- rollback不能ならrecovery制約を隠していないか。

主なRequirement:

- RQ-002 / 006 / 017 / 020 / 021 / 022

### 3.3 Execution quality

主な問いは次のとおりです。

- Business Result、named Outcome、Execution Error、Observationを区別できるか。
- dependency unavailable / degraded / timeout / unknownをsuccessへ丸めていないか。
- replay / takeover / retryで二重副作用を起こさないか。
- failureがどのApplication / Binding / Capability / Provider / Runtime境界で起きたかを追えるか。

主なRequirement:

- RQ-018 / 019 / 020 / 021 / 022

### 3.4 Interaction quality

主な問いは次のとおりです。

- high-density表示が操作不能・理解不能になっていないか。
- Target / Host / input modalityに応じて主要Scenarioを完遂できるか。
- validation / error / unavailable状態から利用者が復旧できるか。
- accessibilityをVisual Fidelityやscreenshot similarityへ還元していないか。

主なRequirement:

- RQ-010 / 011 / 013 / 021 / 024

### 3.5 Performance / resource quality

主な問いは次のとおりです。

- latency / throughput / memory / external cost等をTarget / Host / Provider / workload条件付きで観測しているか。
- averageだけでtail / cold-start / large-data等のrelevant caseを隠していないか。
- 一つのTargetで得たthresholdを別Targetへ無条件に転用していないか。

主なRequirement:

- RQ-021 / 022 / 023 / 024

### 3.6 Security / authority quality

主な問いは次のとおりです。

- actor、UI convenience、AI autonomyをexecution authorityへ読み替えていないか。
- secret / credential / DSN / unnecessary physical identityをdiagnosticやEvidenceへ複製していないか。
- current Permission / Resource boundaryを迂回する実行経路がないか。

主なRequirement:

- RQ-003 / 008 / 019 / 020 / 021 / 022

### 3.7 Maintainability / evolvability quality

主な問いは次のとおりです。

- 一つの意味変更に複数正本の人手同期が必要になっていないか。
- Provider / Target固有変更がCoreへ不要に波及していないか。
- Evidence tool / Plan / schemaを増やすこと自体が維持負債になっていないか。
- 一人のmaintainerでもRequirement → Decision → Reference → Evidenceを追えるか。

主なRequirement:

- RQ-002 / 003 / 017 / 021 / 023

### 3.8 Business completeness quality

主な問いは次のとおりです。

- Application Functionだけで業務全体を説明したことにしていないか。
- manual / external work、actor、handoff、judgmentを失っていないか。
- Data / Function / Businessの一視点だけで矛盾を隠していないか。

主なRequirement:

- RQ-025 / 026

## 4. Quality Envelope

### 4.1 定義

品質主張を単独値として保持すると、条件が失われた時点で意味が崩れます。

そこで、重要なquality claimを次へ結び付ける考え方を**Quality Envelope**と呼びます。

```text
subject
+ revision / build identity
+ Target / Host / Provider / Runtime context
+ Scenario / workload / precondition
+ quality property
+ measure / probe
+ acceptance rule or rationale
+ Evidence reference
+ status
```

Quality Envelopeは固定schemaではありません。必要な領域で必要な項目だけを持つderivedな監査単位です。

### 4.2 status

少なくとも概念上、次を区別します。

- observed / passed
- observed / failed
- unresolved
- not-applicable
- not-yet-observed

`not observed`を`passed`へ丸めません。

### 4.3 Canonicalではない

Quality Envelopeは:

- business meaningを再定義しない
- authoritative sourceへ暗黙に逆書きしない
- telemetryからBusiness Resultを逆推論しない
- benchmarkからProduct Requirementを自動生成しない
- Target固有qualityを全Targetの意味へ昇格しない

という境界を持ちます。

## 5. Failure topology

TEIでは「error」を一種類へ畳みません。

```text
Business Result
Named Outcome
Execution Error
Observation
Developer Diagnostic
Operational Telemetry
```

これらは相関してよいですが同一ではありません。

例は次のとおりです。

```text
HTTP 200
process exit 0
DB commit success
span.status = OK
```

はいずれも、それ単独ではbusiness successを意味しません。

Diagnostic / Telemetryには必要なlogical identityだけを持たせ、credential、password、DSN本文、secret、不要なraw business dataを通常記録へ複製しません。

## 6. Change safety

Change safetyはDatabase migrationだけに限定しません。

対象には少なくとも次があり得ます。

- Canonical source change
- Application / Binding change
- Database desired-state change
- generated artifact refresh
- Target / Runtime configuration change
- Studio / AI accepted edit

共通原則はは次のとおりです。

```text
candidate
  -> impact / compatibility
  -> current precondition where required
  -> explicit acceptance
  -> apply
  -> reload / observation / evidence
```

です。

外部状態に依存するdestructive operationでは、古いObservationだけを実行許可へ使いません。freshness、revision、identity、ownership等、当該領域で必要なpreconditionをapply直前に確認します。

一方、PostgreSQLの具体的な安全機構をそのまま全領域のUniversal Change Protocolへ昇格しません。

## 7. Interaction / accessibility

Target-neutralはlowest-common-denominator UIを意味しません。

同じApplication meaningに対し、Targetごとに自然なinteractionを提供しつつ、主要ScenarioのoperabilityをEvidence化します。

WebではWCAG 2.2等をtestable referenceとして利用できます。Desktop / Mobile / API / MCP / VUIへWeb固有のsemanticsを強制しません。

Visual Fidelity Evidenceだけでoperability / accessibilityを合格にしません。

## 8. Performance

RQ-024に従い、一律thresholdを先に固定しません。

最初にReferenceでbaselineを測り、少なくとも:

- Target / Host
- Provider / Runtime
- Scenario / workload
- warm / cold等の条件
- revision / build identity

と一緒に保持します。

利用目的から必要性が説明できた時点でacceptance thresholdを定めます。

## 9. AI / derived cognition

AI / associative retrieval / derived cognitionでは、candidate qualityとsemantic correctnessを分けます。

```text
candidate recall / ranking / latency
        !=
applicability
        !=
accepted decision
        !=
execution authority
```

index、embedding、FlyHash / HDC等のrepresentation、trajectoryはderivedです。削除・再生成してもCanonical Worldが変わらないことを守ります。

認知方式の評価値もQuality Envelopeへ結べますが、model confidenceやsimilarityをSemantic Evidenceへ格上げしません。

## 10. Operational Telemetry

Operational TelemetryはRuntimeを観測するProjectionです。

OpenTelemetry等のsemantic conventionsへ写像できることは相互運用上有用ですが、TEI Canonical modelをtelemetry data modelへ合わせません。

必要な相関をは次のとおりです。

```text
TEI logical identity
   -> operational projection
   -> trace / metric / log
```

の方向で行います。

## 11. 外部quality modelの扱い

外部標準は、TEIの第二Requirement正本ではなく抜け漏れ監査・test referenceとして利用します。

- ISO/IEC 25010:2023 — quality characteristicの監査
- WCAG 2.2 — Web interaction / accessibility
- OpenTelemetry semantic conventions — operational telemetry Projection
- NIST SSDF 1.1 — secure development / supply-chain process

明示的な採用判断とTEI自身のEvidenceなしに、準拠・認証・完成claimへ格上げしません。

## 12. Gateとの関係

Quality Architectureは新しいGateを追加しません。

- Gate B: 既存3 Required Evidenceを増やさない
- Gate C: Semantic / Functional / Visualの分離を維持する
- First Target: operability / accessibility / performance / failure behaviorを追加観測する
- Studio / AI / Second Target: それぞれ同じQuality Envelope原則で比較する
- production: observability / HA / supply-chain等を必要なEvidenceとともに拡張する

Quality planningやstatic reviewをactual Evidenceへ読み替えません。

## 13. 先に固定しないもの

- Quality Envelopeの固定schema
- universal quality score
- 全Target共通performance threshold
- 全Target共通accessibility checklist
- OpenTelemetryの必須Runtime dependency
- generic fallback engine
- generic retry policy
- generic migration / rollback protocol
- quality characteristicごとの新しいGate / PAS / Issue群

必要性はReference / Evidenceから決めます。
