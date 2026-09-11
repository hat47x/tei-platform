# TEI Product Quality Architecture

Status: **Design baseline — cross-cutting / Evidence-driven**

関連:

- `requirements/TEI_PRODUCT_REQUIREMENTS.md` — RQ-027〜RQ-030を含むProduct Requirement正本
- `plan/PRODUCT_QUALITY_EVIDENCE_PLAN_2026-09.md` — 段階的な実証計画
- `plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md` — Requirement別の現在地
- `plan/MASTER_PLAN.md` — Product全体のworkstream
- `docs/status.md` — 現在の実装状態
- GitHub Issue #565

## 1. 目的

TEIはこれまで、特に次の品質を強く設計してきました。

- 業務上のmeaningとrealizationの分離
- authoritative sourceとderived artifactの分離
- explicit Binding / authority
- Desired / Actual / Diff / Unresolved
- Business Result / Execution Error / Observationの分離
- Reference / Conformance / Evidenceによる再現性
- secret / Resource / Permissionの境界

これらは一般的な「非機能要件」と別世界の話ではありません。TEIにとっては、**意味を壊さず、変更を安全に行い、失敗を説明し、利用者が操作でき、主張をEvidenceへ戻せること**そのものがProduct Qualityです。

一方、quality concernを個々のProvider、Target、Runtime、Studio、AI Issueだけへ分散させると、次の失敗が起こり得ます。

1. Databaseでは安全でもTargetではsilent fallbackが起こる。
2. Semantic Conformanceは通るが利用者が操作・復旧できない。
3. 性能値はあるが、どのbuild / Host / workloadの値か分からない。
4. operational logは豊富だが、Application meaningへ戻れない。
5. destructive changeが個別Providerごとに別の安全基準になる。
6. AIやderived cognitionの成功率が、そのままProduct correctnessへ読み替えられる。
7. 「production-ready」「accessible」「secure」等の主張が、実証条件から離れて独り歩きする。

本書はこの散逸を防ぐため、qualityを新しい巨大Layerへするのではなく、**既存TEI architectureを横断する設計軸**として固定します。

## 2. Qualityは新しいCanonical Layerではない

品質を次のような独立worldへしません。

```text
Canonical World
      ↓
Quality World
      ↓
Runtime
```

Qualityは既存の各境界に対する問いです。

```text
Authoritative Source
       │
       ├─ meaningは再構成できるか
       ├─ change impactを説明できるか
       ↓
Canonical Meaning
       │
       ├─ Projectionで意味が変わらないか
       ├─ unresolvedを保持できるか
       ↓
Binding / Realization
       │
       ├─ authorityが曖昧でないか
       ├─ destructive preconditionを満たすか
       ↓
Runtime / Provider / Target
       │
       ├─ failureを分類できるか
       ├─ degraded / unknownを観測できるか
       ├─ performance / resource条件を測れるか
       ↓
Observation / Evidence
       │
       └─ quality claimを再確認できるか
```

したがってquality requirementを理由に、既存のSource / Canonical / Binding / Runtime / Evidence境界を複製しません。

## 3. TEI固有のQuality axes

ISO/IEC 25010:2023等の一般quality modelは抜け漏れ監査に有効ですが、TEIの設計判断はTEI固有のfailure modeへ翻訳します。

本書では主に次の軸を使います。

### 3.1 Meaning quality

問い:

- 同じCanonical subjectがProjectionやTarget都合で別meaningになっていないか。
- derived artifactが第二正本になっていないか。
- insufficient observationを「差分なし」へ変換していないか。
- identity解決がfirst-match等の偶然の順序へ依存していないか。

主な関連Requirement:

- RQ-001 / 002 / 003 / 006 / 011 / 017 / 018 / 021 / 026

代表Evidence:

- Source-to-Canonical trace
- Validation / Conformance
- deterministic normalization
- Handover baseline
- multi-Projection comparison

### 3.2 Change quality

問い:

- 変更の意味上の影響をapply前に説明できるか。
- planning後にactual stateが変わっていないか。
- destructive / incompatible / unresolvedを区別できるか。
- fallback / retry / CASCADE等で作用範囲を暗黙拡大していないか。
- rollbackできない変更を「安全」と表示していないか。

主な関連Requirement:

- RQ-006 / 017 / 020 / 021 / 028

代表Evidence:

- Semantic Diff / impact preview
- provider observation
- apply-time precondition
- stale revision rejection
- migration / compatibility Pressure
- recovery / manual intervention contract

### 3.3 Execution quality

問い:

- Business Result、named Outcome、Execution Error、Observationを区別できるか。
- dependency unavailable / degraded / timeout / unknownをsuccessへ丸めていないか。
- failureがどのmeaning / Binding / Runtime境界で起きたかを追えるか。
- replay、takeover、retryが二重副作用を起こさないか。

主な関連Requirement:

- RQ-018 / 019 / 020 / 027

代表Evidence:

- Driver Scenario
- Runtime / Provider failure Pressure
- durable handle / fencing / no-auto-replay
- explicit Observation
- diagnostic correlation

### 3.4 Interaction quality

問い:

- high-density表示が操作不能・理解不能になっていないか。
- Target固有input modalityで主要Scenarioを完遂できるか。
- validation / error状態から利用者が復旧できるか。
- accessibilityをvisual similarityへ還元していないか。

主な関連Requirement:

- RQ-010 / 011 / 013 / 021 / 029

代表Evidence:

- Scenario-based Interaction Evidence
- keyboard / focus / navigation checks
- validation / recovery case
- accessibility checks
- Visual Fidelity artifactとは別のoperability result

### 3.5 Performance / resource quality

問い:

- latency / throughput / resource useをTarget / Host / Provider条件と一緒に記録しているか。
- averageだけでtail latencyやworst relevant caseを隠していないか。
- AI / derived cognitionのlatency / memory / costを意味上のcorrectnessと混同していないか。

主な関連Requirement:

- RQ-023 / 024 / 030

代表Evidence:

- Target-specific measurement
- workload / fixture identity
- host / runtime conditions
- baseline / regression comparison

### 3.6 Security / authority quality

問い:

- authorityとconvenienceを混同していないか。
- secret / credential / physical identityを不要にEvidenceやdiagnosticへ複製していないか。
- AI / automationのautonomyからwrite / Resource authorityを暗黙付与していないか。
- supply-chainやartifact provenanceを確認できるか。

主な関連Requirement:

- RQ-003 / 008 / 019 / 020 / 027 / 030

### 3.7 Maintainability / evolvability quality

問い:

- 一つの変更が複数正本の同期を人へ要求していないか。
- Provider / Target固有変更がCoreへ波及しすぎないか。
- 新しいEvidence toolやplanを増やすこと自体が管理負債になっていないか。
- 一人のmaintainerでも「なぜこの境界があるか」へ戻れるか。

主な関連Requirement:

- RQ-002 / 003 / 017 / 021 / 023 / 028 / 030

## 4. Quality Envelope

### 4.1 なぜ必要か

品質値だけを保存しても、条件を失えば意味を持ちません。

```text
"latency = 80ms"
"accessible = true"
"reliable = 99.9%"
"migration safe = true"
```

これらはsubject、Host、workload、revision、判定方法が分からなければProduct claimとして再確認できません。

そこで、品質主張を次の情報へ束縛する考え方を**Quality Envelope**と呼びます。

```text
Quality Envelope
  subject
  context
  quality property
  measure / probe
  acceptance rule or rationale
  evidence reference
  status
```

### 4.2 最小概念

固定schemaではありません。領域に応じて必要な情報だけを持ちます。

#### subject

何についてのquality claimか。

例:

- Canonical subject
- Application / Pattern
- Capability Binding
- Database migration
- Driver Scenario
- Target build
- Studio operation
- AI / cognition retrieval mechanism
- production adapter

#### context

結果を解釈するための条件です。

例:

- repository revision
- Handover baseline `B / R`
- Target build `Tn`
- Host / OS / architecture
- Provider / Runtime version
- network / DB conditions
- Scenario / workload / fixture
- input modality
- feature flags / compatibility mode

#### quality property

何を評価しているか。

例:

- semantic stability
- destructive-change safety
- diagnosability
- accessibility / operability
- latency
- memory use
- availability behavior
- compatibility
- recovery

#### measure / probe

どのように観測したか。

数値に限りません。

- elapsed time
- memory high-water mark
- exact Conformance result
- keyboard-only Scenario result
- stale-precondition negative Pressure
- failure classification check
- structured audit

#### acceptance rule / rationale

なぜaccept / reject / unresolvedなのか。

閾値がある場合は、値だけでなく根拠へ戻れる必要があります。

#### evidence reference

再確認可能なartifact / test / run / sourceへ戻ります。

#### status

少なくとも概念上、次を区別します。

- observed / passed
- observed / failed
- unresolved
- not-applicable
- not-yet-observed

`not observed`を`passed`へ丸めません。

### 4.3 Quality EnvelopeはCanonicalではない

Quality Envelopeは次をしてはいけません。

- Canonical meaningを再定義する
- authoritative sourceへ暗黙に逆書きする
- Business Resultをtelemetryから逆推論して確定する
- Target固有qualityを全Targetの意味へ昇格する
- benchmarkをProduct Requirementへ自動変換する

Quality Envelopeを削除しても、authoritative sourceからCanonical Worldを再構成できなければなりません。

## 5. Failure topology

TEIでは「エラー」を一種類にしません。

```text
Business Result
  └─ business commandが意味上返した結果

Named Outcome
  └─ business上の明示的な分岐結果

Execution Error
  └─ 意図した実行を成立させられなかったruntime failure

Observation
  └─ external / actual stateを別操作で観測した結果

Developer Diagnostic
  └─ design / source / validation / bindingの問題を説明する情報

Operational Telemetry
  └─ runtime operationを観測・相関するderived signal
```

### 5.1 相関してよいが同一視しない

Operational TelemetryからExecution Errorの発生箇所を調べることはできます。

しかし、

```text
span.status = OK
```

だからBusiness Resultがsuccessだった、と逆定義してはいけません。

同様にHTTP 200、process exit 0、DB commit成功等をnamed Outcomeへ自動変換しません。

### 5.2 diagnostic identity

診断には、必要に応じて安全なlogical identityを利用します。

例:

- Application ID
- Pattern ID
- Capability ID
- Datasource alias
- Provider kind
- Target / Host identity
- Source Reference
- repository / build revision

credential、password、DSN本文、secret path、不要なphysical DB identity、raw business data等を「相関のため」という理由だけで通常diagnosticへ保存しません。

### 5.3 degraded / unknown

依存先障害時に、次を区別します。

```text
known success
known business outcome
known execution failure
known degraded mode
unknown completion state
```

`unknown`はfailureより弱いsuccessではありません。replay safetyやObservationが必要な独立状態として扱います。

## 6. Change safety architecture

変更安全性はDatabase migrationだけの問題ではありません。

```text
source edit
  ↓
Canonical change
  ↓
Binding / realization change
  ↓
Generated / Target change
  ↓
Runtime / external state change
```

各段階で「何が変わるか」を異なるEvidenceで確認します。

### 6.1 変更分類

すべての領域へ同じenumを強制しませんが、少なくとも設計上は次を区別します。

- semantic no-op
- compatible
- incompatible
- destructive
- unresolved

例えばfile moveはsemantic no-opになり得ますが、Canonical ID変更はmeaning changeです。

### 6.2 stale observation

actual state依存の変更では、planning時に安全だったことをapply時の安全へ自動延長しません。

```text
plan at O1
   ↓ time / external change
apply at O2
```

O1とO2の差がdestructive safetyへ影響する場合、fresh observationまたは同等のprecondition validationが必要です。

PostgreSQL destructive `DropTable`のfresh observation Pressureは、この一般原則の一つの実証です。Database固有実装をそのままCore protocolへ昇格しません。

### 6.3 recoveryを成功条件と混同しない

「rollbackできるから安全」とは限りません。

変更ごとに必要なら、次を分けます。

- prevention / precondition
- rollback
- restore from backup
- forward fix
- retry
- reconciliation
- manual intervention

存在しないrecovery pathを自動生成したことにしません。

## 7. Interaction / accessibility architecture

### 7.1 high-densityとaccessibilityを対立させない

TEI Studioは高密度な設計環境を志向します。

高密度とは情報量を減らさないことではなく、**関連情報の局所性を保ち、不要な画面遷移や分断を減らすこと**です。

したがって、文字を極端に小さくする、hoverだけに情報を置く、色だけで状態を伝える、keyboard focusを失う等はhigh-densityの要件から導かれません。

### 7.2 Targetごとのinteraction contract

同じbusiness intentでもTargetによりinteractionは異なります。

```text
same Application meaning
      ├─ Web: keyboard / pointer / responsive UI
      ├─ Desktop: keyboard / pointer / native dialog
      ├─ Mobile: touch / small viewport
      ├─ VUI: sequential speech interaction
      ├─ API: request / response contract
      └─ MCP: tool/resource interaction
```

WebのDOMやARIAをCanonicalへ持ち込みません。VUIやAPIへWeb accessibility semanticsを機械的に写しません。

### 7.3 Web TargetとWCAG 2.2

First Web Targetでは、適用可能なWCAG 2.2 success criteriaを外部のtestable referenceとして利用します。

これは次を意味しません。

- Gate C前に完全なWCAG適合を宣言する
- 全criteriaをTEI Canonical schemaへ写す
- visual evaluatorだけでaccessibilityを判定する

まずReference Scenarioに必要なinteractionから、keyboard、focus、label / name、status / error communication、contrast以外の非色依存伝達等をEvidence化します。

## 8. Performance / resource architecture

### 8.1 一律thresholdを置かない

RQ-024に従い、業務端末、B2C Web、Studio、API、AI/cognition等を同一latency targetへ押し込みません。

### 8.2 測定点をmeaningへ接続する

例えばFirst Web Targetでは、単なるpage-load scoreだけでなく次のScenario単位を候補にします。

- initial projection / load
- browse
- record open
- reference lookup
- validation
- authorization
- command response
- Business Result projection
- Observation completion

測定結果は`B / R / T1`等のbaseline / build identityへ結びます。

### 8.3 resource efficiency

latencyだけでなく必要に応じて次を扱います。

- memory
- CPU
- network round trips / bytes
- startup cost
- background activity
- AI token / external service cost
- cognitive index size / retrieval cost

ただしmetricの存在自体をProduct Valueにしません。利用Scenarioの制約を説明できるものだけを採用します。

## 9. Operational telemetryはProjectionとして扱う

production運用ではtrace / metric / log / eventを横断相関できることが有用です。

OpenTelemetry semantic conventions等を採用候補にできますが、TEI Canonical modelをtelemetry data modelへ合わせません。

概念上は次です。

```text
TEI logical identity / runtime context
        ↓ explicit mapping
Operational Telemetry Projection
        ↓
OpenTelemetry-compatible signals等
```

mappingは次を満たすべきです。

- Canonical identityを不要に失わない
- Target / Runtime / Provider contextを区別する
- sensitive materialを通常attributeへ入れない
- telemetry backendを変えてもbusiness meaningが変わらない

OpenTelemetryを利用しないHostでも、RQ-027 / RQ-030の診断・Evidence要件は失われません。

## 10. Surface別Quality responsibility

### 10.1 Authoring / Source

- parse / validation failureをsource locationへ戻す
- derived artifactからauthorityへ逆書きしない
- stale base revisionをrejectまたは明示conflict化する
- semantic no-opとmeaning changeを区別する

### 10.2 Canonical / Validation

- ambiguityをfirst-matchで解決しない
- unresolvedを保持する
- invalid cross-referenceを明示する
- deterministicなmeaning reconstructionを優先する

### 10.3 Driver

- UIなしでmeaningを再現する
- Business Result / Execution Error / Observationを分離する
- synthetic adapterのEvidence natureを明示する
- failure ScenarioをTarget実装前に再現できるようにする

### 10.4 Database / Provider

- provider固有観測をCore factへ無条件昇格しない
- destructive operationにpreconditionを置く
- actual provider failureをsemantic successへ変換しない
- cleanup / restorationを必要なEvidenceで確認する

### 10.5 Target

- Handover baselineからmeaningを再定義しない
- Semantic / Functional / Visual / Interaction qualityを分離する
- accessibility / operabilityをTarget条件付きで検証する
- performanceをbuild / Host / Scenario条件と共に測る

### 10.6 Studio

- Preview / Rejectはnon-mutating
- stale editを黙ってmergeしない
- high-densityとkeyboard / navigation / state visibilityを両立する
- accepted changeはStudioなしでreload / Conformanceできる

### 10.7 AI / derived cognition

- candidate qualityとsemantic correctnessを分ける
- false positive / false negative / unknownを観測する
- ranking scoreをapplicabilityへ変換しない
- latency / memory / external costをQuality Envelopeへ含められる
- AI / cognition停止時もCanonical authorityが失われない
- write / invoke authorityをretrieval qualityから導出しない

### 10.8 Runtime / production

- ownership / fencing / unknown completionを保持する
- operational telemetryからlogical subjectへ安全に相関できる
- secret / Resource authorityをdiagnostic便宜で広げない
- compatibility / upgrade / rollback制約を明示する

## 11. External quality modelsの位置づけ

### ISO/IEC 25010:2023

Product qualityの抜け漏れ監査用checklistとして参照します。

公式:

- https://www.iso.org/standard/78176.html
- https://webstore.iec.ch/en/publication/90024

TEI独自RequirementをISO characteristic名へ一対一に置換しません。特にMeaning / Authority / EvidenceはTEI固有のarchitecture concernとして維持します。

### WCAG 2.2

Web Targetで適用可能なaccessibility acceptance criteriaを検討する外部referenceです。

公式:

- https://www.w3.org/TR/WCAG22/

Evidenceなしに適合レベルを宣言しません。

### OpenTelemetry Semantic Conventions

Operational Telemetry Projectionのinteroperability候補です。

公式:

- https://opentelemetry.io/docs/specs/semconv/

OpenTelemetry data modelをCanonical schemaへしません。

### NIST Secure Software Development Framework

secure development / provenance / supply-chainのprocess checklistとして参照できます。

公式:

- https://csrc.nist.gov/pubs/sp/800/218/final

SSDF practiceをそのままTEI runtime semanticsへ移しません。

## 12. Quality debt / unresolved

quality上の不足を「将来やる」だけで隠さず、次のどれかへ分類します。

- Requirement未定義
- design未確定
- implementation未実装
- Reference不足
- actual Evidence未取得
- threshold未確定
- context不足
- external dependency / environment blocker

この分類はIssue labelを必須にするものではありません。少なくともplan / audit上で不足のnatureを混同しないための考え方です。

## 13. Gateとの関係

Quality Architectureは新しいGateを追加しません。

### Gate B

既存3 Required Evidenceを維持します。

RQ-027〜030を理由に第四laneを追加しません。

Gate Bの現在の目的はUI-less meaning / generated reproducibility / real PostgreSQL Providerをsame-Rで閉じることです。

### Gate C

First Targetでqualityを初めて利用者interactionへ広げる重要点です。

Gate CのSemantic / Functional / Visual Evidenceを壊さず、計画上は次を追加で観測できるようにします。

- Interaction / operability
- applicable accessibility checks
- failure / recovery path
- RQ-024 performance context
- Quality Envelope provenance

これらを一つの巨大scoreへ畳みません。

### Gate C後

Studio / AI / Second TargetのQuality Envelopeを比較し、Target-neutral / developer-experience / cognition設計を修正します。

## 14. 非目標

- qualityを新しいCanonical top-level modelへすること
- ISO/IEC 25010 taxonomyをTEI schemaへコピーすること
- 一つの総合quality scoreを作ること
- Gate BのRequired Evidenceを増やすこと
- 全Targetへ同じperformance thresholdを置くこと
- visual scoreをaccessibility / functional correctnessへ読み替えること
- telemetryをBusiness Resultのauthorityへすること
- OpenTelemetry SDKを全Runtimeへ必須化すること
- WCAG / ISO / NIST準拠を文書だけで宣言すること
- 全quality concernごとに専用tool / Issue / schemaを作ること

## 15. 設計判断の要約

```text
Quality is cross-cutting, not a new authority.

Meaning       -> silent driftを防ぐ
Change        -> impact / precondition / recoveryを明示する
Execution     -> failure / degraded / unknownを分ける
Interaction   -> usable / operable / accessibleをTarget条件で検証する
Performance   -> context付きで測る
Evidence      -> claimを再確認可能にする

Quality Envelope
  = subject + context + property + measure + acceptance + evidence + status
  != Canonical meaning
  != universal score
```

この境界を保つことで、TEIは厳格さを増やしながら、各Provider / Target / AI実装へ品質規則を重複実装することを避けます。