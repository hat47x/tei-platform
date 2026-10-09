# ADR-0028: TEI成果物を人間・AI Builder・AI Business Actorの共通知識面として扱う

- 状態: 採用
- 日付: 2026-09-18
- 関連: ADR-0022 / ADR-0023 / RQ-015 / RQ-018〜RQ-020 / RQ-025 / RQ-027〜RQ-030

## 背景

TEIはこれまで、業務上の意味をCanonicalに保持し、設計、Validation、Projection、Generation、Runtime Bindingへ接続する基盤として設計してきた。

生成AIの役割についても、既存知識を探索してから実装へ進むこと、AI ProposalとCanonical受入境界を分離することを定めている。

一方、今後のソフトウェア開発では、人間がコードを一行ずつ記述することよりも、生成AIが既存Applicationを読み、変更し、検証し、継続的に保守する形が中心になることをTEIの設計前提として扱う必要がある。

このときTEIへ期待される価値は、単にcode generationを容易にすることではない。

- 生成AIが後から開発へ再入場してもApplicationの意味を復元しやすい
- 人間がgenerated code全量を読まずに意味・差分・影響・例外・Evidenceを理解できる
- Domain / Infrastructure知識を別Applicationへ再利用・移植できる
- 場合によってはAIエージェントがTEI成果物を読み、Business ProcedureやCapabilityを理解して直接業務を遂行できる

ことが重要になる。

最後の点はADR-0022の「AIはProposalからCanonicalへ直接飛ばない」という設計と混同しやすい。

しかし、

1. **Applicationの定義を変更すること**
2. **既に受入済みのApplication / Business knowledgeに従って業務Actionを実行すること**

は別のAuthority境界である。

この区別を明示する必要がある。

## 判断

TEI成果物を、次の主体が同じ意味へアクセスするための**共通知識面**として扱う。

- 人間
- AI Builder / coding agent
- TEI Application / Runtime
- AI Business Actor
- 外部tool / integration

ただし、同じ知識を読めることは、同じ権限を持つことを意味しない。

### 1. Authoring PlaneとOperational Planeを分離する

#### Authoring Plane

AI BuilderがApplicationを構築・変更するときは、既存のADR-0022を維持する。

```text
Canonical / Source / Evidence
        ↓
AI Builder understanding
        ↓
Proposal / Structured Intent
        ↓
Validation / Impact / Policy
        ↓
Shared Acceptance Boundary
        ↓
authoritative source update
        ↓
reload / conformance
```

AI Builderが高い自律性を持っていても、Canonical authorityを暗黙には得ない。

AI Builderへ渡す理解用Contextは、Repository断片の単純な列挙に限定しない。対象subjectについて、現在のSource / Evidence、未解決事項、authority limit、設計判断に必要な確認順序を、用途別のderived Projectionとしてまとめてよい。

AI Builder向けには、共有Knowledge ContextそのものをBuilder専用schemaへ変えず、対象subjectから**Builder Review Plan**を別のderived Projectionとして生成してよい。この計画は`current-knowledge-context`をrouting scopeとし、Source / Evidence、定義、Data設計、依存・影響のうち何をどの順で読むかを示す。各Knowledgeが持つ設計原則や業務意味は複製しない。

```text
shared Knowledge Context
        ↓
Builder Review Plan
        ↓
purpose-bound Knowledge selection
        ↓
AI Builder understanding

Builder Review Plan
  != shared Context
  != semantic applicability
  != accepted Decision
  != write / runtime authority
```

Review Planのfingerprintはrouting identityを表すだけであり、選択したKnowledge自体のfreshness proofにはしない。また、推奨された区画をQualification済みKnowledgeやsemantic applicabilityへ昇格させず、必要なQualificationを別境界で通す。current Knowledge Contextに区画が無いことも、他のKnowledge surfaceを含む全体的な不存在へ一般化しない。Knowledge内容の再利用可否は、それぞれのProjection / DisclosureのfingerprintとEvidence境界で別に確認する。

Data設計では、`tei knowledge-context ... -disclose data-design` がこの境界を担う。識別子、候補キー、関数従属性等について、PersistenceやUIの構造から業務意味を推測せず、根拠が不足する場合は`unresolved`を保持する。投影が返すreview protocolや設計候補はAIへの作業指針であって、Canonical authorityやAcceptanceそのものではない。

```text
derived review protocol
  -> AI Builderの確認順序

derived candidate
  -> Proposal / review input

review protocol / candidate
  != accepted business meaning
  != write authority
```


#### Operational Plane

AI Business Actorが受入済みの業務知識に従って仕事をするときは、次の経路を基本とする。

```text
accepted Business / Application knowledge
        ↓
Business Procedure / Action
        ↓
Data / Constraint / Policy
        ↓
Semantic Capability
        ↓
Authorization / Resource / Binding
        ↓
Command / external action
        ↓
Observation / Evidence
        ↓
Business Result / unresolved
```

この経路では、Application定義を変更するProposalは必須ではない。

AI Business Actorは、既に定義・受入済みのAction / Capabilityを、与えられたAuthorityの範囲でinvokeできる。

### 2. TEIを読めることとAuthorityを分離する

次をそれぞれ独立して扱う。

- Canonical / Projectionをreadできるか
- Business Procedureを理解できるか
- Actionを選択できるか
- Capabilityをinvokeできるか
- Resource / secret / networkへaccessできるか
- write / external side effectを起こせるか
- retry / takeoverできるか
- authoritative sourceを変更できるか
- 人間承認済みDecisionとして扱われるか

AI actorであること、model性能、自律度、TEI知識を多く読めることをAuthorityの根拠にはしない。

### 3. UI操作をAI業務遂行の標準境界にしない

Semantic CapabilityやBusiness Actionとして明示的な境界が存在する場合、AIエージェントへ人間向けGUIのclick操作を模倣させることを標準経路にはしない。

```text
AI Agent
  -> Business Action / Capability

preferred over

AI Agent
  -> visual UI imitation
  -> hidden application behavior
```

ただし、既存システムにCapability境界がなくUIしか利用できない場合のadapter / computer-useを禁止しない。

その場合も、UI操作結果をそのまま業務完了とはみなさず、Observation / Evidenceへ戻す。

### 4. すべてのBusiness Procedureをmachine-executable workflowへ変換しない

Business Procedureは、

- 人間判断
- AI判断
- 物理作業
- 外部組織
- SaaS
- 既存システム
- 未自動化作業

を含み得る。

TEIは全手順をworkflow engineへ閉じ込めない。

AI Business Actorに必要なのは、少なくとも、

- 現在どのActionか
- 自分が実行可能か
- 必要なConstraint / Authorizationは何か
- どのCapabilityを利用できるか
- 実行後に何をObservation / Evidenceとして確認するか
- どこで人間・外部Actorへhandoffするか

を理解できることである。

### 5. Human Projectionを一級の成果物として扱う

生成AIが実装の多くを担う場合、人間へgenerated code全量を読ませることを主要なControlにはしない。

TEIは同じCanonical meaningから、

- high-density table
- graph / flow
- semantic diff
- impact
- unresolved / exception
- authority change
- Evidence / provenance
- natural-language explanation

等をProjectionし、人間が必要な判断を行える状態を目指す。

Human ProjectionもCanonical sourceではないが、人間の認知帯域を保護するための重要なProduct surfaceとする。

### 6. Knowledge portabilityをcode portabilityより広く扱う

別Applicationへ移植可能な対象には、source codeだけでなく次を含める。

- Domain meaning
- Relation / Constraint
- Procedure / Pattern
- Policy / Variation
- Validation
- Capability contract
- Authorization rule
- Database / Infrastructure mapping knowledge
- Provider / Target constraint
- accepted Decision / Evidence

再利用時は、origin、dependency、assumption、Variation、Evidence、destination overrideを追跡できることを優先する。

具体的なpackage形式やdistribution mechanismは本ADRでは固定しない。

## ADR-0022との関係

ADR-0022は変更しない。

「AIがCanonical変更を行う場合はProposal / Acceptanceを通す」という判断は維持する。

本ADRで追加するのは、

> AIエージェントが、受入済みのTEI knowledgeを読み、既に許可されたBusiness Action / Capabilityを実行することは、Canonical変更とは別のOperational Authorityである

という区別である。

したがって、ADR-0022の「自然言語からRuntimeへ直接飛ばない」という停止線も維持する。

AIエージェントの入力に自然言語が含まれても、実際の副作用はSemantic Capability / Authorization / Resource / Runtime Bindingを経由する。

## Command / Observationとの関係

ADR-0023およびRQ-018を維持する。

AI Business ActorがCommandを成功させても、

```text
command accepted
    !=
business completed
```

である。

必要な場合はObservation / Evidenceを取得し、Business Result、failure、unknownを判定する。

AIエージェントが自律的であることを理由に、unknownをsuccessへ丸めない。

## 影響

### 良くなる点

- TEIをcode generation用metadataより広い知識基盤として位置づけられる
- AI Builderが長期的にApplicationへ再入場しやすくなる
- 人間のreviewをgenerated code精読へ依存しにくくなる
- Agent-nativeな業務遂行を、GUI imitationだけに依存せず設計できる
- Business knowledgeをApplication / AI Agentの双方で再利用できる
- Domain / Infrastructure knowledgeの移植可能性をProduct Valueへ含められる
- autonomyとauthorityを混同せずAI利用範囲を広げられる

### 増える設計課題

- AI Business Actor向けOperational Projectionの最小要件
- Business Procedure / ActionとCapabilityのtraceability
- Action単位のAuthorization / Resource policy
- Agent handoffとhuman escalation
- Observation / Evidenceの取得方法
- Knowledge reuse時のassumption / Variation / invalidation
- Human Projectionの認知効率評価

これらは個別Reference / Issueから最小形を決める。

## 採用しない案

### AI AgentがTEIを読めれば全Capabilityをinvokeできる

read authorityとexecute authorityを混同するため採用しない。

### Agent向けにCanonical Worldを複製した専用Knowledge Baseを正本化する

二重正本化するため採用しない。

必要ならderived Operational Projection / indexを作るが、Canonicalへ戻れることを要求する。

### すべての業務をworkflow engineで実行可能にする

人間判断や外部作業を不当に削るため採用しない。

### Agentは常にGUIを操作する

Applicationのsemantic境界を利用できず、UI変更へ脆弱で、Business Result / Evidenceとの区別も難しくなるため標準経路にはしない。

### 人間reviewを不要にすることを成功条件にする

人間の役割はcodingから理解・判断へ変化し得るが、必要なAuthority / accountabilityを自動的に消すことは本ADRの目的ではない。

## 検証

後続Referenceでは少なくとも次を分離して検証する。

1. AI Builderが既存TEI knowledgeから局所変更contextを取得できる
2. 人間がsource全量を読まず意味差分・impact・Evidenceを理解できる
3. AI Business ActorがBusiness Procedureから実行可能Actionを識別できる
4. AgentがAuthorizationのないCapabilityをinvokeできない
5. read-only Actionとside-effect Actionを区別できる
6. Command成功をBusiness completionへ自動変換しない
7. human / external actorへのhandoffを保持できる
8. Agent向けderived knowledgeを削除してもCanonical meaningが失われない
9. 一つのDomain / Infrastructure knowledgeを別Applicationへ移植した際、assumption / Variation差を検出できる

## 関連文書

- `product/value/TEI_PRODUCT_VALUE.md`
- `product/vision/TEI_PRODUCT_VISION.md`
- `product/principles/TEI_PRODUCT_PRINCIPLES.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/adr/0023-command-observation-explicit-read-boundary.md`
- `plan/issue/000042-business-procedure-cross-view-reference.md`
