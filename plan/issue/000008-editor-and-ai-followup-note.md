# ISSUE-000008: Editor / AI支援の着手条件を整理する

- 状態: Done — Gate C Ready（T = `0ba87a4e0a96f70a198b03ff7652fb00e3826abe`、2026-09-19）で着手条件が成立し、担当をISSUE-000046 / 000047 / 000027へ引き渡し済み
- 解除条件: Gate C Ready（成立済み: [Gate C Readiness Audit](../GATE_C_READINESS_AUDIT.md)）
- 関連PAS: PAS-01 / PAS-06

## 統合後の担当境界

本IssueはGate C後のEditor / AI着手条件を記録します。最小Editor Projection（最初のauthoring Reference）は[ISSUE-000046](000046-studio-shared-acceptance-authoring-loop.md)が担当します。共通機能の実装は[ISSUE-000039](000039-business-change-rehearsal.md)の差分・依存解析、[ISSUE-000040](000040-semantic-component-composition.md)の探索・組み立て、[ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)のAcceptanceへ接続します。以下の構想に同じ機能が現れても別実装を起票せず、当該Issueの成果を利用します。

## 背景と着手条件の根拠

Product Requirements Gap Auditでは、EditorとAI支援が大きな未実証領域であることを確認しました。

一方で、EditorやAIを先に作ると、まだ実証されていないApplication PatternやDriverの意味をUI都合で固定する危険があります。

このIssueは実装Issueではなく、**いつEditor / AI支援へ着手してよいか**を明示するための待機条件を記録します。

v1 Reference Application実証計画では、Application Pattern、既存認証Compatibility、Driver、第一Web Targetを順に実証し、Gate C「第一Targetとの差を説明できる」を通過してからEditor / AI支援を具体化します。

## 関連するProduct Requirement

- RQ-010 Editor
- RQ-011 Projection
- RQ-015 AI knowledge-first workflow
- RQ-016 AI suggestion / decision UX
- RQ-017 二重管理の最小化
- RQ-019 Runtime permission / Resource
- RQ-021 Reference / Conformance
- RQ-022 検証段階の明示

## 着手条件

少なくともGate Cを通過するまで、Editor / AIの詳細interfaceを固定しません。

Gate Cでは次が実資産として揃っていることを求めます。

1. Application Patternの所有境界がReferenceで確認されている
2. Driverが同じApplication上の意味をUIなしで実行できる
3. ValidationとAuthorizationをTarget固有コードへ閉じ込めていない
4. 第一Web Targetが同じApplication上の意味を利用している
5. 複数Targetで共有できそうなPresentation上の意味と、Web固有差分を追跡できる
6. Source / Application / Driver / Targetの来歴を辿れる
7. Semantic ConformanceとInteraction / Functional Evidenceを分離して確認できる
8. Visual Fidelity Evidenceを上位の意味から分離して扱える
9. Handover Baseline revisionとTarget build revisionを混同せず追跡できる

単に`ISSUE-000005`〜`000007`がcloseされたことではなく、これらをEvidenceから説明できることを着手条件とします。

## Gate C後に最初に扱うEditor

条件が揃った後も、完成した統合開発環境から始めません。

最初のEditorでは、v1 `order_entry`について次を同じCanonical上の意味から高密度に確認できるProjectionを対象とします。

- Application / Datasource
- Data / Relation
- Application Pattern
- Validation / Command
- Authorization
- Source / Diagnostic
- Driver result
- 第一Web TargetとのProjection差
- Semantic / Functional / Visual Evidenceの違い

画面数を増やして情報を分断するのではなく、複数階層の関係を局所性を保ったまま確認できることを優先します。

Editor自身をCanonicalの正本にはしません。

Go + Wails/WebViewを利用する場合も、WailsはEditor / Hostの実装候補であり、Canonical ApplicationのPresentation意味にはしません。Editor shellの都合から業務Targetの意味を逆定義しないことを維持します。

最初の具体的なauthoring Referenceは[`ISSUE-000046`](000046-studio-shared-acceptance-authoring-loop.md)で扱います。StudioはProjection / Authoring Clientとして、file-native authoring、CLI、AI等と同じShared Acceptance Boundaryへ収束させます。

Studioのdeveloper-facing Wails Hostは、Generated Wails Desktop Application Targetとは別物です。Studio GUIの成功を第二Target Evidenceへ読み替えません。

## Gate C後に最初に扱うAI支援

最初から自律agentや大量のコード生成を中心にはしません。

実在するv1 Reference資産を使い、次の順で支援できるかを確かめます。

- 既存Application Patternの探索
- 既存Capabilityの探索
- 既回答事項の再質問抑止
- 不足している典型確認事項の提示
- AI提案と採用済みDecisionの分離
- Decisionの変更履歴
- 提案の無視・抑制・無効化

AIが何を知っているかではなく、Repository上のどのEvidenceを根拠に提案したかを追跡できることを重視します。

read-onlyなknowledge-first探索は[`ISSUE-000047`](000047-knowledge-first-semantic-exploration.md)、AI Proposal / Acceptanceは[`ISSUE-000027`](000027-ai-proposal-acceptance-boundary.md)で分離して扱います。

## 技術リサーチからの設計仮説

直近のLLM × Model-Driven Engineering、agentic workflow、AI-assisted Software Engineering、UI generation evaluationの研究を確認した結果、TEIでは研究方式をそのまま実装するのではなく、Gate C後に検証する境界を次のように具体化します。

詳細は[`TECH_RESEARCH_INTEGRATION_2026-09.md`](../TECH_RESEARCH_INTEGRATION_2026-09.md)を参照します。

### AIの非決定性をCanonicalより手前へ閉じ込める

AIが自然言語やRepository上のEvidenceを解釈しても、その出力をそのままCanonicalな正本や実行物にはしません。

AIが扱う最初の出力は、たとえば次のような候補です。

- 意図の構造化候補
- Canonical変更案
- Pattern / Capabilityの利用候補
- ValidationやPolicyの追加候補
- Presentation / InteractionのProjection候補

候補を採用する場合は、Editor / CLI / file-native authoringと共有するAcceptance Boundaryを通します。

概念上は次のように扱います。

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

Acceptance Boundaryでは、対象に応じてParse / Normalize / Validate / Architecture or Policy Check / Semantic Diff / Impact Analysis / Acceptance Policy / reload / Conformance等を利用します。

この境界は[ADR-0022](../adr/0022-ai-proposal-canonical-acceptance-boundary.md)で採用します。ただし、Proposal schema、API、provider、UIはGate C後まで固定しません。

### staleな提案を黙って適用しない

AIやEditorがProposalを作った後にsourceが変化する可能性があります。

必要な変更候補では`baseModelRevision`等を持ち、受理時に現在revisionと比較できる構造を検証します。

staleなProposalを検出した場合は、reject、rebase、regenerate、manual conflict resolution等へ明示的に分岐し、黙って最新版へ適用しません。

### AI Knowledgeを第二の正本にしない

AIが参照する語彙、設計上のヒント、探索規則、利用例等を将来持つ場合も、それらをCanonical meaningの代替にはしません。

AI向けKnowledgeは、CanonicalなApplication / Semantic / Capability定義を理解・探索するための補助情報としてversion管理できても、同じ業務上の意味を別形式で人が二重管理する構造は避けます。

Knowledgeから導いた提案についても、最終的にどのCanonical subjectやSource、Evidenceを根拠としたかを辿れることを重視します。

### 生成より検証を重視する

2026年公開のLLM × MDE体系調査では、既存研究はModel Generationへ大きく偏り、DSL Engineering、Metamodeling、既存MDE toolchainとの深い統合は限定的です。

TEIではこの偏りをそのまま追わず、AIが新しいモデルやコードを大量生成することよりも、既存知識の探索、変更候補の作成、構造検査、実行結果との照合を重視します。

同じCanonical meaningをDriverとTarget Projectionが共有できるようになった後は、AI支援についても「生成できたか」だけでなく、次をEvidenceとして評価します。

- 提案が既存のCanonical identityを正しく参照したか
- 採用後のauthoritative sourceをAIなしで再読込できるか
- reload後のCanonicalがValidation / Conformanceを満たすか
- Driverで同じ意味を再現できるか
- Target固有コードへ意味を逃がしていないか
- AI固有Evidenceを除去してもCanonical meaningが変わらないか

### AI支援の目的・対象・自律度を比較可能にする

AI-assisted Software Engineeringの研究では、AI支援を目的、対象活動、AI方式、自律度等の軸で分類する考え方が整理されています。

TEIではtaxonomyをそのままCanonical modelへ採用しませんが、AI機能を比較・監査するEvidenceとして、少なくとも次を識別できるようにします。

- purpose
- target activity / artifact
- AI mechanism / type
- autonomy level

さらにTEI固有の要求として、autonomyとauthorityを分けます。

自律度が高いAIでも、authoritative sourceへのwrite、Capability invocation、Resource / secret / network / filesystem access等は、RQ-019の権限境界から明示的に与えます。

### 過去trajectoryをDerived Experience Knowledgeとして再利用できる余地を残す

Workflow生成研究では、過去の成功trajectoryを直接再利用、部分修正、新規生成へ振り分ける方式が示されています。

TEIではこれをAI固有の第二の正本としては採用しません。一方、RQ-015の「既存知識を探索してから新規実装へ進む」を拡張し、過去の成功Proposal / Decision / Evidence /修正trajectoryをDerived Experience Knowledgeとして再利用する実験余地を残します。

再利用する場合も次を要求します。

- 元のCanonical ID / Source / Evidenceへ戻れる
- dependency / revision fingerprintを持てる
- stale経験を無条件に適用しない
- similarity / routingをauthoritative Decisionとみなさない
- direct reuseした候補もAcceptance Boundaryを通す

外部研究で報告されたtoken削減率やsuccess rateは、そのままTEIのKPIにはしません。

### Target評価の機能と視覚を混同しない

UI生成評価研究では、利用者workflowを実行できるかと、visual fidelityを分けて評価する方法が示されています。

第一Web Targetではこの考え方をTEI向けに拡張し、次を別Evidenceとして扱います。

- Semantic Conformance
- Interaction / Functional Evidence
- Visual Fidelity Evidence

VLMやscreenshot judge等を利用しても、visual scoreだけでSemantic / Functional Conformanceを成立させません。

このEvidenceは、AIがPresentation候補を作る場合にも同じように利用します。

### 自然言語から直接Runtimeへ接続しない

自然言語からWorkflowを構築する研究では、LLMによる意味解釈、構造化されたIntent、検証済みの決定論的Generator、専門Knowledgeを分離する構成が示されています。

TEIでも、自然言語から直接Runtime操作や生成コードへ飛ぶ経路を標準にはしません。

設計変更はProposal / Acceptanceを経由します。実行要求が必要な場合も、Semantic Capability、Authorization、Runtime Binding、Resource、Business Result / Execution Error / Observation境界へ接続します。

raw AI tool callをTEIのauthority modelの代わりにはしません。

## EditorとAIを同時に固定しない

EditorとAI支援は強く関係しますが、一つの巨大な機能として同時に設計しません。

Gate C後の最初の責務分離は、Repository上で次へ固定します。

1. **Studioの安全なread / inspect / minimal authoring loop** — `ISSUE-000046`
2. **Human / AI共通のread-only knowledge-first Semantic探索** — `ISSUE-000047`
3. **AI Proposal / Shared Acceptance Boundary** — `ISSUE-000027`

最初のread-only Projectionやknowledge searchを、正本の編集や自律実行と切り離して進められる構造を優先します。

採用済みDecision履歴やDerived Experience Knowledgeの具体実装は、上記Referenceから不足が確認された段階でさらに分けます。先に巨大な統合Issueを作りません。

## 第二Targetとの関係

Editor / AI支援を完成させてから第二Targetへ進む必要はありません。

第二Targetの主目的はTarget-neutral境界の圧力試験です。第一Web Targetで共通とみなした部分とWeb固有差分が追跡できれば、Editor / AI支援と並行して第二Targetを検討できます。

第二Targetから得られた差分は、Editorが表示すべきProjection差や、AIが判断時に参照すべきEvidenceとしてEditor / AI側へ戻します。

Target CapabilityとExecution / Trust Profileの境界は[`ISSUE-000026`](000026-target-capability-execution-profile-boundary.md) / GitHub Issue #259で検証します。

Studioのdeveloper-facing Wails Hostは第二Application Targetには数えません。Generated Wails Desktop Targetを採用する場合は、`ISSUE-000026`のApplication Targetとして別途Semantic / Functional / Visual Evidenceを取得します。

## 非目標

- Gate C成立前のUI framework選定
- chat-first IDEの固定
- AIによる正本の無確認自動編集
- AI tool callによるauthority境界の迂回
- Application上の意味とEditor内部状態の二重正本化
- AI向けKnowledge / trajectoryの第二正本化
- EditorとAIを一つの巨大な実装Issueへまとめること
- 第二TargetをEditor完成まで待たせること
- 外部研究のagent architectureや評価値をそのままTEI仕様にすること

## 完了条件

Gate Cを通過し、Application / Driver / 第一TargetのEvidenceからEditor / AI支援に必要な意味境界を具体的に説明できるようになった時点で、本IssueのBlockを解除します。

解除後は、まず次の既存plan issueをReadyへ移せるか確認します。

- `ISSUE-000046` Studio minimal authoring loop
- `ISSUE-000047` knowledge-first Semantic exploration
- `ISSUE-000027` AI Proposal / Acceptance Boundary

新しい実装Issueは、これらのReferenceで追加責務が確認された場合だけ分解します。

## 解除結果

2026-09-21: Gate Cは[Gate C Readiness Audit](../GATE_C_READINESS_AUDIT.md)でReady（T = `0ba87a4e0a96f70a198b03ff7652fb00e3826abe`、2026-09-19）と判定された。上記「着手条件」の1〜9は、Gate C判定条件のEvidence（Semantic / Functional / Visual Evidenceの分離、Handover Baseline `B / R / T`の追跡、Web固有実装のCanonicalへの逆流なし、Target差分のhandoff）から説明できる。よって本IssueのBlockを解除し、完了条件に従い次へ移した。

- `ISSUE-000027`: Ready
- `ISSUE-000047`: In Progress（Gate C Ready後のactual S1 qualification）
- `ISSUE-000046`: Ready
- `ISSUE-000051`: Ready（Product Gate C Ready後のReference evaluation。000047のqualificationを省略しない）

Gate C Readyの非主張（production-ready、第二Target不要、actual external Command成功、Visual Fidelityの人による承認）は[Gate C Readiness Audit](../GATE_C_READINESS_AUDIT.md)「Ready の意味と非主張」のとおりで、Editor / AI側でも読み替えない。最小Editor Projectionの実装責務は000046が持つ。本Issueは着手条件の記録としての役目を終えたためDoneとし、新しい実装Issueは、上記Referenceで追加責務が確認された場合だけ分解する。

## 関連文書

- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/TECH_RESEARCH_INTEGRATION_2026-09.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000026-target-capability-execution-profile-boundary.md`
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000046-studio-shared-acceptance-authoring-loop.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
