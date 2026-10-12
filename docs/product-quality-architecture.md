# TEI Product Quality Architecture

状態：**横断的な設計基準。検証根拠をもとに評価する方針**

関連する文書と課題は次のとおりです。

- `requirements/TEI_PRODUCT_REQUIREMENTS.md` — 製品要求の正本
- `plan/evidence/PRODUCT_QUALITY_EVIDENCE_PLAN_2026-09.md` — 段階的な実証計画
- `plan/evidence/PRODUCT_REQUIREMENT_GAP_AUDIT.md` — 要求ごとの達成状況
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md` — 製品の受け入れシナリオとの対応
- GitHub Issue #565

## 1. 位置づけ

TEIの製品品質は、意味モデルから独立した「非機能の層」だけで扱うものではありません。

すでに定めた次の境界が、実行先、提供元、実行環境、設計・編集環境、AIの各領域でも一貫して守られているかを確かめるための設計上の観点です。

- 業務上の意味と、技術的な実現方法の分離
- 正本となる入力元と、そこから生成した成果物の分離
- 明示的な対応付けと権限
- 期待状態・実際の状態・差分・未解決状態の区別
- 業務結果・実行エラー・観測結果の分離
- 参照例・準拠確認・検証根拠による再現性
- 利用許可・資源・秘密情報の境界
- 実行先の条件を踏まえた性能
- データ・機能・業務の各視点からの相互確認

このため、本書では番号付きの製品要求、意味の正本の要素種別、判定段階、実行環境の部分機構を新たに追加しません。

## 2. なぜ横断Architectureが必要か

要求が適切に定められていても、品質を個別の実装ごとに評価すると、次のような早計な判断につながりかねません。

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

そのため品質は、意味・変更・実行・操作・検証根拠のそれぞれについて、共通の観点から確認します。

## 3. Quality axes

### 3.1 Meaning quality

主な問いは次のとおりです。

- 表示方法・実行先・提供元の都合で、正本の意味が変わっていないか。
- 生成した成果物が第二の正本になっていないか。
- 観測が不十分なのに、`no diff`や成功として扱っていないか。
- 対象の識別が、最初に一致したものを採用するなど、偶然の探索順序に依存していないか。

主なRequirement:

- RQ-001 / 002 / 003 / 006 / 011 / 017 / 018 / 021 / 026

### 3.2 Change quality

主な問いは次のとおりです。

- 意味に変化がない変更、互換性のある変更、互換性のない変更、破壊的な変更、未解決の変更を区別できるか。
- 変更を適用する前に、影響範囲と前提条件を説明できるか。
- 計画した後で実際の状態が変わった場合、古い観測結果を根拠に破壊的な操作を実行していないか。
- 再試行、代替処理、CASCADEなどによって、操作の対象範囲が暗黙に広がっていないか。
- 元に戻せない変更について、復旧上の制約を隠していないか。

主なRequirement:

- RQ-002 / 006 / 017 / 020 / 021 / 022

### 3.3 Execution quality

主な問いは次のとおりです。

- 業務結果、名前付きの結末、実行エラー、観測結果を区別できるか。
- 依存先の利用不能、機能低下、時間切れ、状態不明を、成功として扱っていないか。
- 再実行、監督役の交代、再試行によって、同じ外部変更を二重に起こさないか。
- 失敗が、アプリケーション・対応付け・機能・提供元・実行環境のどの境界で起きたか追跡できるか。

主なRequirement:

- RQ-018 / 019 / 020 / 021 / 022

### 3.4 Interaction quality

主な問いは次のとおりです。

- 情報量の多い表示が、操作しにくさや理解しにくさにつながっていないか。
- 実行先、ホスト、入力方法に合わせて、主要な利用場面の操作を完了できるか。
- 検証エラー、処理エラー、利用不能な状態から、利用者が操作を立て直せるか。
- 利用しやすさを、見た目の再現度や画面画像の類似度だけで評価していないか。

主なRequirement:

- RQ-010 / 011 / 013 / 021 / 024

### 3.5 Performance / resource quality

主な問いは次のとおりです。

- 応答時間・処理量・使用メモリ・外部費用などを、実行先・ホスト・提供元・処理負荷の条件とともに記録しているか。
- 平均値だけを示し、処理時間の長い例、初回起動、大量データなど重要な条件を隠していないか。
- 一つの実行先で決めた判定基準を、別の実行先へ無条件に適用していないか。

主なRequirement:

- RQ-021 / 022 / 023 / 024

### 3.6 Security / authority quality

主な問いは次のとおりです。

- 業務主体、画面の便利さ、AIの自律性を、実行権限があることの根拠にしていないか。
- 秘密情報、認証情報、接続文字列、不要な物理的識別情報を診断結果や検証根拠へ複製していないか。
- 現在の利用許可と資源管理の境界を迂回する実行経路がないか。

主なRequirement:

- RQ-003 / 008 / 019 / 020 / 021 / 022

### 3.7 Maintainability / evolvability quality

主な問いは次のとおりです。

- 一つの意味の変更に対して、複数の正本を人手で同期する必要が生じていないか。
- 提供元や実行先に固有の変更が、中核部分へ不要に波及していないか。
- 検証用ツール、計画、定義形式を増やすこと自体が、保守の負担になっていないか。
- 保守担当者が一人でも、要求から設計判断、参照例、検証根拠へとたどれるか。

主なRequirement:

- RQ-002 / 003 / 017 / 021 / 023

### 3.8 Business completeness quality

主な問いは次のとおりです。

- アプリケーションの機能だけで、業務全体を説明したことにしていないか。
- 手作業や外部で行われる仕事、業務主体、引き継ぎ、判断を漏らしていないか。
- データ・機能・業務の一つの視点だけを見て、矛盾を見落としていないか。

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

共通原則は次のとおりです。

```text
candidate
  -> impact / compatibility
  -> current precondition where required
  -> explicit acceptance
  -> apply
  -> reload / observation / evidence
```

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

必要な相関は次のように表します。

```text
TEI logical identity
   -> operational projection
   -> trace / metric / log
```

この方向で、論理的な識別子と運用上の記録を結び付けます。

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

品質の計画や静的レビューを、実環境で取得したEvidenceと同一視しません。

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
