# TEI 技術リサーチ統合方針 — 2026-09

## 位置づけ

この文書は、`TEI 技術リサーチ短報`で確認した外部研究・技術動向を、TEIの既存Product Requirements、Architecture、Evidence Gateへどのように取り込むかを整理します。

外部研究やframeworkはTEIの仕様根拠そのものではありません。研究で得た構造や失敗しやすい境界を仮説として取り込み、`order_entry` Reference Application、Driver、第一Web Target、第二Targetから得るEvidenceで検証します。

新しい技術名をCanonical Semantic Layerへ追加することより、技術が変わっても残る境界を要求として固定することを優先します。

## 今回取り込む主要な知見

### LLM × Model-Driven Engineering

2026年のsystematic mapping studyでは、LLMをMDEへ適用した研究はModel Generationへ大きく偏り、DSL Engineering、Metamodeling、既存toolchainとの深い統合は相対的に少ないことが報告されています。

TEIではこの偏りを追わず、AIの主要価値を「大量生成」に固定しません。既存知識の探索、変更候補の作成、構造検査、Validation / Conformance、Source-to-Canonical Traceabilityを中心に置きます。

### Structured Intentと決定論的Generator

自然言語から科学Workflowを構築する研究では、LLMによるIntent抽出と、検証可能な構造化Intent、決定論的Generatorを分けるarchitectureが示されています。

TEIではこの考え方を、特定のWorkflow frameworkとして採用するのではなく、**AIの非決定性をCanonicalより手前へ閉じ込める**設計原則として取り込みます。

```text
Natural Language / AI / Editor
            ↓
Proposal / Structured Intent candidate
            ↓
Shared Acceptance Boundary
            ↓
authoritative source
            ↓ reload
Canonical TEI Meaning
            ↓
Deterministic Mapping / Driver / Projection
            ↓
Execution / Observation
```

ProposalやStructured IntentはCanonicalの別名ではなく、受理前の候補です。

### Workflow experience reuse

WorkflowGenでは、過去の成功trajectoryを再利用し、類似度に応じて直接再利用、部分的な書き換え、新規生成を切り替える方式が提示されています。

TEIでは、これをAI向けの新しい正本としては採用しません。一方、RQ-015のknowledge-first workflowを一段具体化する材料として、過去の成功した提案・修正・実行trajectoryを**Derived Experience Knowledge**として再利用する余地を残します。

再利用時も次を要求します。

- 元のCanonical ID / Source / Evidenceへ戻れる
- dependencyやbase revisionが変わった場合に古い経験を無条件に適用しない
- similarityやrouting結果をauthoritative decisionとみなさない
- 直接再利用した候補も共通Acceptance Boundaryを通す
- token削減率等の外部研究値をTEIのProduct KPIとして固定しない

### AI-assisted Software Engineeringの分類軸

AI4SEのtaxonomyでは、AI支援を目的、対象となるSoftware Engineering活動、AI方式、自律度等の軸から整理しています。

TEIではtaxonomyそのものをCanonical modelへ採用しませんが、AI支援機能の挙動を比較・監査するため、少なくとも次をEvidenceとして識別可能にします。

- purpose
- target activity / artifact
- AI mechanism / type
- autonomy level

さらにTEIでは、**autonomyとauthorityを別軸**とします。自律度が高いことは、Canonical書込み権限、Capability実行権限、Resource access、secret accessを意味しません。

### Vision2Webの評価分離

Vision2Webでは、生成されたWeb UIを、操作workflowを確認するfunctional evaluationと、見た目の近さを評価するvisual evaluationに分けています。

TEIではこの考え方をTarget Conformanceへ取り込み、少なくとも次の三層を混同しません。

1. **Semantic Conformance** — 同じCanonical Application上の意味を保持しているか
2. **Interaction / Functional Evidence** — 利用者が必要なScenarioを実行できるか
3. **Visual Fidelity Evidence** — Target上の見た目・配置・デザイン品質が意図に沿うか

Semantic Conformanceは必須です。GUI agent、VLM、image similarity等をVisual Evidenceへ利用しても、Semantic / Functional Conformanceの代替にはしません。

### Wails v3

Wails v3は、Go backendとWebView frontendをDesktop中心に扱いながら、Mobile、Server、MCP等へ機能を広げているため、TEIのTarget / Host境界を圧力試験する材料として有用です。

ただし、WailsのAPIやplatform modelをTEIのCanonical Presentationとして採用しません。

特に次を分離します。

- **Target Capability**: window、dialog、tray、touch、haptics等、そのTargetで何ができるか
- **Execution / Trust Profile**: local / network、transport、origin、session、authorization、Resource境界等、どの信頼・実行条件で動くか

また、Wails等が提供するMCP / control interfaceは、それ自体をApplicationのMCP Targetとはみなしません。Hostのinspection / control面と、業務ApplicationがMCPをPresentation / Interaction Targetとして提供する場合を区別します。

Wails v3のMobile等には実験的な要素も残るため、Go + Wails/WebViewはEditor / Hostの有力候補かつarchitecture probeとして扱い、Target-neutral設計の前提にはしません。

## Product Requirementsへの具体化

今回、新しいProduct Requirement番号を増やすことはせず、既存要求を次のように具体化します。

### RQ-013 Target-neutral Presentation / Interaction

- Target CapabilityとExecution / Trust Profileを同じ差分へ畳まない
- Target固有Capability不足をCanonical Application意味の欠落と混同しない
- Hostのinspection / control protocolとApplication Targetを区別する
- 第二Targetで第一Targetの抽象を圧力試験する

### RQ-015 AI knowledge-first workflow

- AIは既存Domain Component / Pattern / Policy / Capabilityを探索してから新規生成へ進む
- 過去の成功trajectory等はDerived Experience Knowledgeとして再利用可能にする
- AI出力を直接CanonicalへせずProposal / Structured Intent candidateとして扱う
- 構造化された候補から決定論的に生成・変換できる部分は、自由生成より優先する

### RQ-016 AI suggestion / decision UX

- AI提案と採用済みDecisionを分離する
- Proposalの根拠となるCanonical ID / Source / Evidenceを追跡できる
- purpose / target activity / AI mechanism / autonomy levelを比較可能にする
- autonomyとauthorityを分離する
- staleなbase revisionに対する候補を黙って適用しない
- 提案、Decision、変更理由、抑制状態を追跡できる

### RQ-019 Runtime permission / Resource

- AI / Editor / generated codeも通常のCapability / Permission / Resource境界を迂回しない
- AI actorであること、自律度が高いことから権限を暗黙付与しない

### RQ-021 Reference / Conformance

- AI候補は受理後にAIなしで再読込・再現できることを確認する
- interactive TargetではSemantic / Functional / Visual Evidenceを分離する
- Visual evaluatorの結果だけでSemantic / Functional Conformanceを合格にしない

### RQ-022 検証段階の明示

- prerelease / experimental frameworkを利用するときはarchitecture probeとproduction-readyを区別する
- 外部研究の評価値をTEI自身のProduct KPIやConformance基準へそのまま転記しない

## 設計判断として固定するもの

今回のうち、実装方式が変わっても残るAI境界はADR-0022へ切り出します。

- AIの出力はProposal / candidateでありCanonical authorityではない
- file-native / Editor / CLI / AIを共通Acceptance Boundaryへ収束させる
- stale proposalをbase revision等から検出する
- AIのautonomyとauthorityを分離する
- AI Knowledge / trajectoryはderivedであり第二の正本ではない

詳細なProposal schema、AI API、UI、model providerはGate C前に固定しません。

## Gateへの反映

### Gate Bまで

AI interfaceは実装しません。DriverでCanonical Application上の意味とScenarioを再現し、後からAI提案を評価できる基準線を作ります。

### 第一Web Target / Gate C

`ISSUE-000007`では、Semantic Conformanceと利用者ScenarioのFunctional Evidenceを取得します。Visual Fidelityは別Evidenceとして記録できるようにします。

Gate C通過時には、Web固有差分、Target Capability候補、Runtime / Trust差、Visual固有差を区別できることを求めます。

### Gate C後

`ISSUE-000008`の保留を解除し、次を小さく分けて実証します。

- Editor read / navigate / inspect
- safe canonical edit
- AI knowledge search
- Proposal / Structured Intent
- shared Acceptance Boundary
- Decision history
- Derived Experience Knowledgeの再利用

AI Proposal / Acceptance BoundaryのReferenceは`ISSUE-000027` / GitHub Issue #267で扱います。

### 第二Target / Gate D

Target CapabilityとExecution / Trust Profileの分離は`ISSUE-000026` / GitHub Issue #259で圧力試験します。

第二TargetはWailsを採用するために選ぶのではなく、RQ-013の不確実性を最も減らせるTarget / Hostを選びます。

## 採用しない短絡

- AIが自然言語からCanonicalやRuntimeへ直接書き込むこと
- autonomous agentをTEIのauthority modelの代わりにすること
- AI向けKnowledgeをCanonical meaningの複製正本にすること
- Wails APIをTEIのTarget-neutral Presentation語彙へ写すこと
- WailsのMCP control面を自動的に業務MCP Targetとみなすこと
- VLM / screenshot scoreをSemantic Conformanceとみなすこと
- 第一Web TargetだけでTarget-neutral設計を確定すること
- 外部研究で得られたtoken削減率、success rate等をTEIの固定KPIとすること

## 参考にした外部資料

- Weixing Zhang et al., "Large language models in model-driven engineering: a systematic mapping study", Empirical Software Engineering, 2026
- Bartosz Balis et al., "From Research Question to Scientific Workflow: Leveraging Agentic AI for Science Automation", 2026
- WorkflowGen: experience-driven adaptive workflow generation, 2026
- Vision2Web: functional workflow verification and visual fidelity evaluation for web generation, 2026
- "Augmenting software engineering with AI" / AI4SE taxonomy, 2026
- Wails v3 documentation / changelog / releases, 2026-09時点

外部資料の更新によって実装候補は変わり得ますが、この文書で抽出したTEI固有の境界はReference / Conformanceから再評価します。
