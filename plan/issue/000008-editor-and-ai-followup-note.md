# ISSUE-000008: Editor / AI支援の着手条件を整理する

- 状態: 保留
- 解除条件: Gate C

## 背景

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

## 着手条件

少なくともGate Cを通過するまで、Editor / AIの詳細interfaceを固定しません。

Gate Cでは次が実資産として揃っていることを求めます。

1. Application Patternの所有境界がReferenceで確認されている
2. Driverが同じApplication上の意味をUIなしで実行できる
3. ValidationとAuthorizationをTarget固有コードへ閉じ込めていない
4. 第一Web Targetが同じApplication上の意味を利用している
5. 複数Targetで共有できそうなPresentation上の意味と、Web固有差分を追跡できる
6. Source / Application / Driver / Targetの来歴を辿れる

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

画面数を増やして情報を分断するのではなく、複数階層の関係を局所性を保ったまま確認できることを優先します。

Editor自身をCanonicalの正本にはしません。

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

## 2026-09-03 技術調査からの補足

直近のLLM × Model-Driven Engineering研究と、自然言語から再現可能なWorkflowへ変換する研究を確認した結果、TEIの現在の方針を変更するより、Gate C後に検証するAI支援の境界を次の設計仮説として明示しておく方が妥当と判断します。

### AIの非決定性をCanonicalより手前へ閉じ込める

AIが自然言語やRepository上のEvidenceを解釈しても、その出力をそのままCanonicalな正本や実行物にはしません。

AIが扱う最初の出力は、たとえば次のような候補です。

- 意図の構造化候補
- Canonical変更案
- Pattern / Capabilityの利用候補
- ValidationやPolicyの追加候補
- Presentation / InteractionのProjection候補

候補を採用する場合も、既存のParser、Schema、Validation、Conformance、Source-to-Canonical Traceability等の決定論的な境界を通した後にCanonicalへ反映します。

これは詳細なAI interfaceを今決めるという意味ではありません。Gate C後の実証で、どこまでをAIの提案として扱い、どこからをTEI自身が確定するかを比較できるようにするための仮説です。

### AI Knowledgeを第二の正本にしない

AIが参照する語彙、設計上のヒント、探索規則、利用例等を将来持つ場合も、それらをCanonical meaningの代替にはしません。

AI向けKnowledgeは、CanonicalなApplication / Semantic / Capability定義を理解・探索するための補助情報としてversion管理できても、同じ業務上の意味を別形式で人が二重管理する構造は避けます。

Knowledgeから導いた提案についても、最終的にどのCanonical subjectやSource、Evidenceを根拠としたかを辿れることを重視します。

### 生成より検証を重視する

2026年7月公開のLLM × MDE体系調査では、既存研究はModel Generationへ大きく偏り、DSL Engineering、Metamodeling、既存MDE toolchainとの深い統合はまだ限定的です。

TEIではこの偏りをそのまま追わず、AIが新しいモデルやコードを大量生成することよりも、既存知識の探索、変更候補の作成、構造検査、実行結果との照合を重視します。

特に、同じCanonical meaningをDriverとTarget Projectionが共有できるようになった後は、AI支援についても「生成できたか」だけでなく、次をEvidenceとして評価します。

- 提案が既存のCanonical identityを正しく参照したか
- 採用後のCanonicalがValidation / Conformanceを満たすか
- Driverで同じ意味を再現できるか
- Target固有コードへ意味を逃がしていないか
- AIを介さず再読み込みしても同じCanonical meaningを復元できるか

### 自然言語から直接Runtimeへ接続しない

自然言語から科学Workflowを構築する2026年の研究では、LLMによる意味解釈、構造化されたIntent、検証済みの決定論的Generator、専門Knowledgeを分離する構成が示されています。

TEIでも、自然言語から直接Runtime操作や生成コードへ飛ぶ経路を標準にはせず、少なくともCanonicalな意図または明示的な変更候補を経由する方式を優先して比較します。

概念上は次のような境界を検証対象とします。

```text
Natural Language / AI
        ↓
Proposal / Structured Intent
        ↓
Validation / Conformance
        ↓
Canonical TEI Meaning
        ↓
Deterministic Mapping / Driver / Projection
        ↓
Execution / Observation
```

実際の構文、API、承認方式はGate C後のEvidenceから決めます。

### 参考にした外部資料

- Weixing Zhang et al., "Large language models in model-driven engineering: a systematic mapping study", Empirical Software Engineering, published 2026-07-16, https://doi.org/10.1007/s10664-026-10921-4
- Bartosz Balis et al., "From Research Question to Scientific Workflow: Leveraging Agentic AI for Science Automation", arXiv:2604.21910, 2026, https://arxiv.org/abs/2604.21910

これらはTEIの仕様根拠そのものではなく、Gate C後の設計比較に使う外部Evidenceとして扱います。論文の方式をそのまま採用するのではなく、v1 Reference ApplicationでTEI自身の境界を実証してから判断します。

## EditorとAIを同時に固定しない

EditorとAI支援は強く関係しますが、一つの巨大な機能として同時に設計しません。

Gate C通過後、まず次を分離してIssue化します。

1. Editor Projectionとして必要なread / navigate / inspect境界
2. Editorから安全に正本を変更する境界
3. AIが既存知識を探索するread-only境界
4. AIが確認提案とDecision候補を作る境界
5. 採用済みDecisionと履歴を管理する境界

最初のread-only Projectionやknowledge searchを、正本の編集や自律実行と切り離して進められる構造を優先します。

## 第二Targetとの関係

Editor / AI支援を完成させてから第二Targetへ進む必要はありません。

第二Targetの主目的はTarget-neutral境界の圧力試験です。第一Web Targetで共通とみなした部分とWeb固有差分が追跡できれば、Editor / AI支援と並行して第二Targetを検討できます。

第二Targetから得られた差分は、Editorが表示すべきProjection差や、AIが判断時に参照すべきEvidenceとしてEditor / AI側へ戻します。

## 非目標

- Gate C成立前のUI framework選定
- chat-first IDEの固定
- AIによる正本の無確認自動編集
- Application上の意味とEditor内部状態の二重正本化
- EditorとAIを一つの巨大な実装Issueへまとめること
- 第二TargetをEditor完成まで待たせること

## 完了条件

Gate Cを通過し、Application / Driver / 第一TargetのEvidenceからEditor / AI支援に必要な意味境界を具体的に説明できるようになった時点で、本Issueの保留を解除します。

その後、Editor Projection、正本の編集、AI knowledge search、確認提案、Decision履歴などを、責務が混ざらない複数の実装Issueへ分解します。

## 関連文書

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
