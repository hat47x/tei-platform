# ISSUE-000027: AI Proposal / Acceptance Boundaryをv1 Referenceで実証する

- 状態: 保留
- GitHub Issue: #267
- 本格着手条件: Gate C

## 背景

RQ-015では、生成AIが既存知識を探索してから新規実装へ進むことを要求しています。RQ-016では、AIの提案と採用済みDecisionを区別し、履歴・抑制・変更可能性を持つことを要求しています。

`ISSUE-000008`では、Application Pattern、Driver、第一Web TargetのEvidenceが揃うGate CまではAIの詳細interfaceを固定しない方針です。

一方、技術リサーチから、AI支援をTEIへ接続する際に先に固定すべきなのはmodel providerやchat UIではなく、**AIの非決定的な候補とCanonical authorityの間に共通Acceptance Boundaryを置くこと**だと整理しました。

この境界は[ADR-0022](../adr/0022-ai-proposal-canonical-acceptance-boundary.md)で設計判断として採用します。

本IssueではGate C後、v1 `order_entry`を利用し、AI専用の正本変更経路を作らずにProposalを受理できることをReferenceから実証します。

## 関連するProduct Requirement

- RQ-002 Authoring sourceとCanonical meaningを追跡できること
- RQ-003 暗黙の複数正本を作らないこと
- RQ-015 AI knowledge-first workflow
- RQ-016 AI suggestion / decision UX
- RQ-017 人の二重管理を最小化すること
- RQ-019 実行時の権限とResourceを明示的に制約できること
- RQ-021 Reference / Conformance
- RQ-022 検証段階の明示

## 本Issueで固定しないもの

Gate C後のReferenceを作るIssueですが、最初から完成形のAI subsystemを設計しません。

次は比較対象として残します。

- Proposalの具体schema
- natural languageだけを入口にするか、structured editor actionも同じProposalとするか
- `baseModelRevision`の具体形式
- model provider
- retrieval方式
- embedding / vector store利用
- prompt format
- agent loop
- UI
- AI固有の永続store

これらをCanonical meaningより上位の要求として固定しません。

## 最初のReference

最初のReferenceは、同じ`order_entry`について次の二種類を扱います。

### 1. read-only proposal

既存Application Pattern / Capability / Validation / Relation / Sourceを探索し、変更を伴わない確認提案を作ります。

例:

- submit Commandが参照するValidationを説明する
- 既存Capabilityから利用可能なものを列挙する
- 第一Web Targetとの差を指摘する

この段階ではCanonicalを変更しません。

### 2. canonical change candidate

小さく、結果を明確に検証できる変更候補を作ります。

候補例はGate C後に実資産から決めますが、次のような範囲を想定します。

- Validation ruleの追加・修正候補
- Presentation Projection hintの追加候補
- 既存Capability Bindingの明示化候補
- authoring source上の小さい宣言変更

既存のauthoritative sourceをAI固有形式へ移し替える作業は選びません。

## Proposalに必要な情報の候補

Proposal schemaは未確定ですが、Referenceでは少なくとも次を保持できる必要があります。

- proposal identity
- purpose
- target activity / artifact
- referenced Canonical IDs
- Source / Evidence references
- proposed change
- base revision
- AI mechanism / type
- autonomy level
- proposer / actor information
- created time等のaudit情報

ただし、model名、prompt、trajectory等をCanonical subjectのidentityへ含めません。

AIのautonomy levelとauthorityも分けます。Proposal metadata上の自律度から、write / invoke / Resource permissionを自動導出しません。

## Acceptance Boundary

Referenceでは、AIだけの特別な受入APIではなく、Editor / CLI等でも共有できる受入原則を確認します。

概念上は次を扱います。

```text
proposal / candidate
        ↓
parse / decode
        ↓
normalize
        ↓
validate
        ↓
architecture / policy check
        ↓
semantic diff / impact analysis
        ↓
acceptance policy
        ↓
authoritative source update
        ↓
reload / conformance
```

変更種類によって不要なstepはあり得ますが、AI経路だけがValidationやConformanceを迂回してはいけません。

## revision conflict

Proposal作成時点のbase revisionと、受理時点のcurrent revisionを比較できるようにします。

staleなProposalを検出した場合、少なくとも次のいずれかを明示します。

- reject
- rebase candidate
- regenerate
- manual conflict resolution

黙って最新版へ適用しません。

Referenceでは、同じProposalを古いrevisionへ意図的に当てるnegative caseを含めます。

## Derived Experience Knowledge

技術リサーチでは、過去の成功trajectoryを再利用する方式が、AI workflowの効率改善に有効な例として報告されています。

TEIではこれをCanonical Knowledgeへせず、実験的なDerived Experience Knowledgeとして扱います。

Gate C後の初回Referenceで必須実装にはしませんが、少なくとも次を満たす設計余地を残します。

- 過去Proposal / Decision / Evidenceへ戻れる
- dependency / revision fingerprintを持てる
- stale経験を無条件に直接再生しない
- direct reuse / rewrite / new proposalの選択をheuristicとして扱える
- どの経路でもAcceptance Boundaryを通す

## Authority / Runtime境界

AIがProposalを生成できることと、RuntimeへCommandを送れることを同じauthorityにしません。

Runtime操作が必要なReferenceを将来追加する場合も、Semantic Capability、Authorization、Runtime Binding、Resource Permission、Business Result / Execution Error / Observation境界を利用します。

raw tool callをTEIのauthority modelの代わりにはしません。

## Evidence

完了時には少なくとも次を残します。

- Proposal input / normalized representation
- Proposalが参照したCanonical ID / Source / Evidence
- base revision
- Acceptance結果
- Semantic Diff
- Validation / Conformance結果
- authoritative sourceの変更
- reload後のCanonical結果
- stale proposalのnegative case
- AI経路を外した同一Acceptance Boundaryの利用例

AI provider固有のpromptやtrajectoryをEvidenceとして残す場合も、secretや不要なprivate dataを保存しません。

## 完了条件

次をReferenceとConformanceから説明できることを完了条件とします。

1. AIが既存Knowledge / Evidenceを探索してProposalを作れる
2. ProposalそのものはCanonical authorityではない
3. ProposalがCanonical ID / Source / Evidenceへtraceできる
4. AI ProposalとEditor / CLI候補が同じAcceptance原則へ収束できる
5. stale base revisionを検出できる
6. 受理後のauthoritative sourceをAIなしで再読込できる
7. reload後のCanonical meaningがValidation / Conformanceを満たす
8. AIのautonomyとwrite / invoke / Resource authorityを分離できる
9. AI固有EvidenceやDerived Experience Knowledgeを削除してもCanonical Worldが変わらない
10. ProposalからRuntimeへ無権限に直接到達できない

## 非目標

- autonomous agent frameworkの完成
- chat-first IDEの完成
- AI providerの標準化
- vector databaseの標準化
- AIによる無確認自動変更
- AIによる無確認のside effect実行
- prompt / model metadataのCanonical化
- 外部研究のsuccess rateやtoken削減率をTEIの固定KPIにすること

## 関連文書

- `plan/TECH_RESEARCH_INTEGRATION_2026-09.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/issue/000008-editor-and-ai-followup-note.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `docs/architecture.md`
