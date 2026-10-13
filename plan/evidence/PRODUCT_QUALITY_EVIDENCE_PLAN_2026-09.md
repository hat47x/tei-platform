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

判定段階Cで定めた検証根拠の区別を維持します。

1. 意味の準拠確認
2. 操作・機能の検証根拠
3. 表示の忠実度に関する検証根拠

これらを単一の`qualityScore`にまとめることはしません。

最初の実行先では、さらに次の事項を観測します。

- 操作可能性と回復
- 適用可能なアクセシビリティの確認
- 性能と資源使用量の基準値
- 失敗・機能低下・不明状態での動作
- Quality Envelopeの来歴

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

この表は検証計画を整理するためのもので、固定された成果物のスキーマではありません。

### Web accessibility

適用可能なWCAG 2.2の達成基準を、外部の試験基準として参照できます。

まず、参照シナリオで重要度の高い次の事項を検証します。

- キーボードによる操作
- フォーカスの視認性と移動順序
- 検証規則とエラー表示の対応付け
- 状態の伝達
- 色だけに依存しない情報伝達
- 適用可能な場合の拡大表示・リフロー・ポインター操作

これらから順に検証根拠を取得します。

実際の検証根拠なしに、全基準への適合を主張することはしません。

### performance baseline

RQ-024に従い、まず次の条件を記録します。

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

各項目を記録します。

すべてのシナリオに、`sub-second`などの一律の性能基準を適用することはしません。

## 8. Q3A — Studio / Authoring quality

重点的に検証するシナリオ：

```text
Select
 -> Propose
 -> Preview
 -> Accept / Reject
 -> reload / Conformance
```

厳密に検証する条件：

- 事前確認と却下では正本の内容を変更しない
- 基準版が古い場合に、確認せずに変更を適用しない
- 書込みが一部だけ完了した場合や失敗を成功として扱わない
- 情報量の多い画面でも、選択位置・フォーカス・診断箇所を失わない
- 受理に失敗した場合、ソース・対象・違反した契約まで追跡できる

主にRQ-002・010・017・021・023の境界を厳密に検証します。

## 9. Q3B — AI / derived cognition quality

次の事項を区別します。

```text
candidate quality
   != applicability
   != accepted decision
   != execution authority
```

候補を生成する仕組みについて測定できる項目：

- 再現率と見逃した候補
- 関係のない候補
- 候補がない場合と不明状態での動作
- 必要な場合の決定論的な動作と安定性
- 応答時間
- メモリ使用量と索引のサイズ
- 再構築に必要な費用や処理量
- 外部モデルやサービスの利用費用

失敗時に検証する条件：

- 関連する候補を返せない
- 関係のない候補が上位に表示される
- 索引が古くなっている
- LLMや外部サービスを利用できない
- 派生した索引がすべて削除された

いずれの場合も正本のモデルを失わず、候補を事実として確定させることはしません。

## 10. Q3C — Second Target

二つ目の実行先では、最初の実行先で得た品質の仮説をそのまま当てはめず、同じ意味の基準版を対象に改めて測定します。

比較する観点：

- 実行先の能力
- 実行と信頼の条件
- 操作方法
- 意味・機能・表示の検証根拠
- 操作可能性とアクセシビリティ
- 性能と資源使用量
- 失敗と回復

最初の実行先の合格基準値が二つ目の実行先でも妥当かどうかは、検証根拠に基づいて判断します。

## 11. Q4 — Provider / production boundary

本番運用の成熟度については、次の事項を段階的に検証します。

- 運用上の監視記録の対応付け
- 永続的な保存機構と高可用性
- 回復と実行担当の引継ぎ
- 互換性と更新
- 署名付き成果物と来歴
- 依存関係とソフトウェアの供給経路
- 秘密情報と認証情報の運用

OpenTelemetryやNIST SSDFなどを参照しても、形式上の準拠だけでTEIの製品要求を満たしたとは判断しません。

## 12. Quality Envelopeの最小評価表現

領域ごとに異なる成果物を使えます。ただし、レビューでは少なくとも次の内容を説明できることを目指します。

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

この表現を現時点で正本のスキーマとして固定することはしません。

## 13. 完了判定

製品品質は、一度の作業で`Done`になるものではありません。

#565で完了と判断する範囲は次のとおりです。

- 品質上の観点が既存の製品要求との対応関係に整理されている
- 横断的なアーキテクチャが定義されている
- 段階的な検証根拠の取得計画がある
- 新たな判定段階・製品要求・第二の正本を不要に増やしていない

ここまでを完了範囲とします。

品質についての具体的な主張は、参照事例・実行先・実行環境の検証根拠を取得した時点で更新します。
