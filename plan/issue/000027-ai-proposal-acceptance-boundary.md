# ISSUE-000027: AI Proposal / Acceptance Boundaryをv1 Referenceで実証する

- 状態: Blocked — Gate C Ready待ち
- GitHub Issue: #267
- 本格着手条件: Gate C Ready
- Product Acceptance Scenario: **PAS-01 primary / PAS-06 secondary**

## 背景

RQ-015では、生成AIが既存知識を探索してから新規実装へ進むことを要求しています。RQ-016では、AIの提案と採用済みDecisionを区別し、履歴・抑制・変更可能性を持つことを要求しています。

`ISSUE-000008`では、Application Pattern、Driver、第一Web TargetのEvidenceが揃うGate CまではAIの詳細interfaceを固定しない方針です。

AI支援をTEIへ接続する際に先に固定すべきなのはmodel providerやchat UIではなく、**AIの非決定的な候補とCanonical authorityの間に、Editor / CLI / file-native authoringと共有するAcceptance Boundaryを置くこと**です。

この判断は[ADR-0022](../adr/0022-ai-proposal-canonical-acceptance-boundary.md)で採用しています。

本IssueではGate C後、v1 `order_entry`を利用し、AI専用の正本変更経路を作らずにProposalを受理できることをReferenceから実証します。

read-onlyな既存Semantic資産探索そのものは`ISSUE-000047`で扱い、Studioの最小Authoring Loopは`ISSUE-000046`で扱います。本Issueはその探索結果・変更候補を**AI Proposalとしてshared Acceptance Boundaryへ接続する部分**に集中します。

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
- natural languageだけを入口にするか、structured editor actionも同じcandidateとするか
- AI provider
- retrieval方式
- embedding / vector store利用
- prompt format
- agent loop
- UI
- AI固有の永続store
- optional Patch形式

これらをCanonical meaningより上位の要求として固定しません。

## 前提: knowledge-first探索とProposalを分ける

AIが新しい変更候補を作る前に、`ISSUE-000047`のread-only explorationで既存Semantic資産を確認します。

```text
intent
  ↓
read-only semantic exploration
  ↓
applicable / incompatible / unknown
  ↓
explicit references
  ↓
AI Proposal candidate
```

検索順位、類似度、AI内部記憶だけをProposalのSemantic根拠にはしません。

既存資産を再利用できる場合は、qualified logical key / Canonical reference / Source / Evidenceを明示してProposalへ接続します。

## 最初のReference

最初のReferenceでは、同じ`order_entry`について小さいCanonical change candidateを扱います。

候補例はGate C後に実資産から決めますが、次のような範囲を想定します。

- Validation ruleの小さい追加・修正候補
- 既存Capability Bindingの明示化候補
- Relation等の小さいauthoring変更候補
- Presentation / Interaction Projection hint候補

既存のauthoritative sourceをAI固有形式へ移し替える作業は選びません。

AIが提案する変更と、`ISSUE-000046`のStudio操作、file-native edit、CLI / refactor等が、同じAcceptance原則へ収束できることを比較します。

## Proposalに必要な情報の候補

Proposal schemaは未確定ですが、Referenceでは少なくとも次を保持できる必要があります。

- proposal identity
- purpose
- target activity / artifact
- referenced qualified logical keys / Canonical IDs
- Source / Evidence references
- proposed change
- `baseModelRevision`
- AI mechanism / type
- autonomy level
- proposer / actor information
- audit information

model名、prompt、trajectory等をCanonical subjectのidentityへ含めません。

AIのautonomy levelとauthorityも分けます。Proposal metadata上の自律度から、write / invoke / Resource permissionを自動導出しません。

### mandatory authored Stable IDを要求しない

AI Proposalを識別・適用するためだけに、すべてのauthoring sourceへUUIDやmandatory authored Stable IDを追加しません。

対象の解決には、既存のqualified logical key / Canonical identity / Source Indexを利用します。

file pathそのものをSemantic identityにはしません。

## Shared Acceptance Boundary

AIだけの特別な受入APIを作りません。

```text
file edit -------┐
Studio ----------|
CLI / refactor --|--> candidate --> Shared Acceptance Boundary --> authoritative source
AI Proposal -----|
Matrix ----------|
optional Patch --┘
```

対象に応じて少なくとも次の決定論的検査を共有します。

```text
candidate
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

この順序を一つの固定API / processとして実装することまでは要求しません。

重要なのは、AI経路だけがValidation / Conformance / authorityを迂回しないことです。

## Preview / Reject / Accept

AI Proposalを生成した時点ではauthoritative sourceを変更しません。

### Preview

受理前に少なくとも次を確認できるようにします。

- Semantic Diff
- affected Canonical subjects
- source / dependency impact
- Driver / Target impact
- Validation / Architecture Diagnostic

### Reject

Rejectではauthoritative sourceを変更しません。

AI固有store、chat履歴、trajectory等を削除してもCanonical Worldが変わらないことを維持します。

### Accept

Accept時だけshared Acceptance Boundaryを通り、authoritative sourceを更新します。

受理後はAIなしでsourceをreloadし、同じCanonical meaning / Validation / Conformanceを再確認できることを要求します。

## revision conflict

Proposal作成時点の`baseModelRevision`と、受理時点のcurrent revisionを比較できるようにします。

staleなProposalを検出した場合、少なくとも次のいずれかへ明示分岐します。

- reject
- re-read / regenerate candidate
- explicit rebase candidate
- manual conflict resolution

黙って最新版へ適用しません。

Referenceでは、同じProposalを古いrevisionへ意図的に当てるnegative caseを含めます。

## file move / logical renameとの関係

ProposalがSourceへ参照を持つ場合も、file path移動だけをSemantic identity変更として扱いません。

- file move: qualified logical identityが同じならsemantic no-op候補
- logical rename: Semantic change

AI Proposalのstale判定やtarget解決をfile pathだけへ依存させません。

## Derived Experience Knowledge

過去の成功Proposal、Decision、Evidence、修正trajectory等をDerived Experience Knowledgeとして再利用する余地を残します。

初回Referenceで必須実装にはしませんが、利用する場合は少なくとも次を要求します。

- 元のCanonical ID / Source / Evidenceへ戻れる
- dependency / revision fingerprintを持てる
- stale経験を無条件に直接再生しない
- direct reuse / rewrite / new proposalの選択をheuristicとして扱う
- どの経路でもShared Acceptance Boundaryを通す

Derived Experience KnowledgeはCanonical Knowledgeの第二正本ではありません。

## Authority / Runtime境界

AIがProposalを生成できることと、RuntimeへCommandを送れることを同じauthorityにしません。

Runtime操作が必要なReferenceを将来追加する場合も、Semantic Capability、Authorization、Runtime Binding、Resource Permission、Business Result / Execution Error / Observation境界を利用します。

raw tool callをTEIのauthority modelの代わりにはしません。

## Evidence

完了時には少なくとも次を残します。

- Proposal input / normalized representation
- Proposalが参照したqualified logical key / Canonical ID / Source / Evidence
- `baseModelRevision`
- Preview Semantic Diff / impact
- Accept / Reject結果
- authoritative source変更の有無
- source / model revision after
- reload後のCanonical結果
- Validation / Conformance結果
- stale Proposal negative case
- AI経路を外した同一Acceptance Boundaryの利用例
- AI固有Evidence削除後もCanonical meaningが変わらないこと

AI provider固有のpromptやtrajectoryをEvidenceとして残す場合も、secretや不要なprivate dataを保存しません。

## Gate Cとの関係

本IssueはGate C前に本格実装しません。

Gate Cから少なくとも次を受け取ります。

- Handover Baseline / Application identity
- 第一Web Targetで実証したSource-to-Canonical trace
- Semantic / Functional / Visual Evidenceの分離
- Target固有差 / unresolved

Proposal schema、API、AI provider、Editor UI、承認UXは、その実資産から決めます。

## 完了条件

次をReferenceとConformanceから説明できることを完了条件とします。

1. `ISSUE-000047`のknowledge-first探索結果からexplicit-reference Proposalを作れる
2. ProposalそのものはCanonical authorityではない
3. Proposalがqualified logical key / Canonical ID / Source / Evidenceへtraceできる
4. AI ProposalとStudio / file-native / CLI候補が同じAcceptance原則へ収束できる
5. Preview / Rejectではauthoritative sourceを変更しない
6. stale `baseModelRevision`を検出できる
7. 受理後のauthoritative sourceをAIなしで再読込できる
8. reload後のCanonical meaningがValidation / Conformanceを満たす
9. file moveとlogical renameを区別できる
10. AI都合でmandatory authored Stable IDを導入しない
11. AIのautonomyとwrite / invoke / Resource authorityを分離できる
12. AI固有EvidenceやDerived Experience Knowledgeを削除してもCanonical Worldが変わらない
13. ProposalからRuntimeへ無権限に直接到達できない

## 非目標

- autonomous agent frameworkの完成
- chat-first IDEの完成
- AI providerの標準化
- vector databaseの標準化
- AIによる無確認自動変更
- AIによる無確認のside effect実行
- prompt / model metadataのCanonical化
- AI専用Acceptance engine
- mandatory UUID / Stable IDの導入
- Patch-only authoring modelの固定
- 外部研究のsuccess rateやtoken削減率をTEIの固定KPIにすること

## 関連文書

- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/TECH_RESEARCH_INTEGRATION_2026-09.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/issue/000008-editor-and-ai-followup-note.md`
- `plan/issue/000046-studio-shared-acceptance-authoring-loop.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `docs/architecture.md`
