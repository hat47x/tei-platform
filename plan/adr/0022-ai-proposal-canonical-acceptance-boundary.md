# ADR-0022: AI ProposalとCanonical受入境界を分離する

- 状態: 採用
- 日付: 2026-09-04

## 背景

TEIは、Authoring sourceとCanonical meaningを分離し、Source-to-Canonical Traceability、Validation、Conformanceを通して意味を確定する設計を採用しています。

AI支援についてもRQ-015 / RQ-016で、既存知識を探索してから新規実装へ進むこと、AI提案と採用済みDecisionを分離することを要求しています。

一方、生成AIをAuthoringへ接続すると、自然言語やRepository上のEvidenceから得た非決定的な出力を、どの時点でTEIの正式な変更として扱うかが新しい境界になります。

直近のLLM × Model-Driven Engineering研究ではModel Generationへの偏りが大きく、自然言語から再現可能なWorkflowへ変換する研究では、LLMによるIntent抽出と、検証された構造からの決定論的生成を分ける構成が示されています。

これらはTEIの仕様根拠ではありませんが、AIの非決定性をCanonicalより手前へ閉じ込め、既存の受入・検証機構へ接続する方向を支持する外部Evidenceとして利用できます。

また、これまでのEditor設計では、file-native candidate、Editor、CLI、AI等がそれぞれ別の正本変更経路を持つのではなく、共通のAcceptance Boundaryへ収束させる方針を検討してきました。

## 判断

AIが生成した内容を、そのままCanonical Worldやauthoritative sourceへ書き込みません。

AIの最初の出力は、**Proposal / Structured Intent candidate**として扱います。

```text
Natural Language / Repository Evidence / AI
                    ↓
       Proposal / Structured Intent
                    ↓
         Shared Acceptance Boundary
                    ↓
           authoritative source
                    ↓ reload
          Canonical TEI Meaning
                    ↓
   Mapping / Driver / Projection / Runtime
```

### ProposalはCanonical authorityではない

Proposalは、たとえば次を含められる候補です。

- Canonical subjectへの変更候補
- Pattern / Capability利用候補
- Validation / Policy追加候補
- RelationやMappingの候補
- Presentation / Interaction Projection候補
- Custom implementation候補

Proposal schemaやAPIをこのADRでは固定しません。

重要なのは、Proposalが受理前の候補であり、Canonical meaningの別表現や第二の正本ではないことです。

### AI・Editor・CLIを共通Acceptance Boundaryへ収束させる

AI専用の書込み経路を作りません。

file-native authoring、Editor、CLI、AI等から生じた変更候補は、最終的に同じ受入原則を通ります。

Acceptance Boundaryでは、対象に応じて少なくとも次の決定論的検査を組み合わせます。

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

この順序を一つの固定APIや単一processとして実装することまでは要求しません。対象となるauthoring sourceや変更種類に必要な検査を省略せず、AI経路だけが近道にならないことを要求します。

### staleなProposalを黙って適用しない

interactiveなEditor / AI支援では、Proposal作成後にauthoritative sourceやCanonical Worldが変更される可能性があります。

Proposalには、必要に応じて`baseModelRevision`等の基準revisionを持たせ、受理時に現在revisionと比較できる構造を採用します。

基準が古い場合は、自動的に最新状態へ上書き適用せず、再生成、再base、明示的なconflict解消等へ進みます。

revisionの具体形式はGate C後のReferenceで決めます。

### autonomyとauthorityを分離する

AI機能の自律度が高いことは、権限が強いことを意味しません。

次を別々に扱います。

- AIが何を自律的に判断・提案できるか
- authoritative sourceを変更できるか
- Capabilityをinvokeできるか
- Resource / secret / network / filesystemへアクセスできるか
- side effectを伴うCommandを実行できるか

AI actorであることや、特定modelを利用していることをtrustの根拠にはしません。

AI / Editor / generated codeもRQ-019のCapability / Permission / Resource境界を通ります。

### AI Knowledge / trajectoryはderivedとして扱う

AIが参照するSkill、探索規則、利用例、過去の成功Proposal、修正trajectory等は、既存知識を効率よく探索・再利用するために保持できます。

ただし、これらをCanonical meaningの複製正本にはしません。

Derived Experience Knowledgeを再利用する場合も、少なくとも次を保ちます。

- 元のCanonical ID / Source / Evidenceへのprovenance
- dependency / revision fingerprint
- invalidationまたは再評価可能性
- similarity / routing結果とauthoritative Decisionの分離
- 再利用候補もAcceptance Boundaryを通ること

### AI固有EvidenceをCanonicalへ混ぜない

model名、prompt、tool call、trajectory、token使用量、judge score等は、再現性や評価のためのEvidenceになり得ます。

しかし、業務上の意味を定義しない限りCanonical Semantic Layerへ持ち込みません。

secret、credential、不要なprivate dataをPrompt / Evidence / Durable artifactへ保存しない原則も維持します。

### 自然言語からRuntimeへ直接飛ばない

TEIの設計・Authoring支援では、自然言語から生成した内容をそのままRuntime操作や生成コードの実行へ接続することを標準経路にはしません。

設計変更はProposal / Acceptanceを経由します。

実行要求が必要な場合も、既存のSemantic Capability、Authorization、Runtime Binding、Resource、Business Result / Execution Error / Observation境界を利用し、AI tool callをauthority modelの代替にはしません。

## Gate Cとの関係

このADRは、Gate C前にAI interfaceを実装・固定する判断ではありません。

Gate Cまでは、Application、Driver、第一Web Targetから、AIが将来参照・変更すべき意味境界をEvidenceとして揃えます。

Gate C後、`ISSUE-000027` / GitHub Issue #267で、v1 `order_entry`を使ってProposal / Acceptance Boundaryを小さく実証します。

Proposal schema、API、AI provider、Editor UI、承認UXは、そのEvidenceから決めます。

## 影響

### 良くなる点

- AIの非決定性をCanonical authorityから分離できる
- Editor / CLI / AIで別々の正本変更経路を作らずに済む
- AIを交換・停止してもauthoritative sourceとCanonical meaningを復元できる
- stale proposalの黙示的な上書きを避けられる
- AIの自律度とsecurity authorityを混同しなくなる
- 過去trajectoryの再利用を、二重正本化せずに導入できる
- AI生成量ではなくValidation / Conformance / Evidenceを品質基準にできる

### 増える負担

- Proposalと採用済みDecisionを区別して管理する必要がある
- revision / conflict handlingが必要になる
- AI経路でも通常のValidation / Conformanceを通す必要がある
- provenanceやEvidenceを保存する範囲を設計する必要がある

これらは、AI支援をTEIの既存authority modelへ安全に統合するために受け入れます。

## 採用しない案

### AIがCanonicalを直接変更する

非決定的な出力がauthoritative changeと同一化し、Validation、Traceability、Decision履歴を迂回できるため採用しません。

### AI専用のCanonical Knowledge Storeを作る

同じ業務上の意味を人が二重管理する可能性が高く、RQ-003 / RQ-017に反するため採用しません。

### AIのautonomy levelから権限を自動決定する

autonomyとauthorityは別の関心事であり、security境界を曖昧にするため採用しません。

### 研究論文のagent architectureをそのままTEIへ移植する

研究上のworkflowや評価条件をTEIの業務Applicationへ一般化するEvidenceがないため採用しません。

### Gate C前にProposal schemaやAI providerを固定する

実在するApplication / Driver / Target差が揃う前にinterface都合から意味を固定する危険があるため採用しません。

## 検証

`ISSUE-000027`では少なくとも次を確認します。

- read-only knowledge searchとCanonical変更候補を分離できる
- ProposalがCanonical ID / Source / Evidenceを参照できる
- AI Proposalをそのままauthoritative sourceへ書かない
- Editor / CLI等の候補も同じAcceptance原則を利用できる
- stale base revisionを検出できる
- 採用後のauthoritative sourceをAIなしで再読込できる
- reload後のCanonicalがValidation / Conformanceを満たす
- AIのautonomyとwrite / invoke / Resource authorityを別に設定できる
- Derived Experience Knowledgeを無効化してもCanonical Worldが変わらない
- AI固有Evidenceを除去しても業務上の意味が失われない

## 関連文書

- `plan/TECH_RESEARCH_INTEGRATION_2026-09.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `plan/issue/000008-editor-and-ai-followup-note.md`
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `docs/architecture.md`
