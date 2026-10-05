# ISSUE-000027: AI Proposal / Acceptance Boundaryをv1 Referenceで実証する

- 状態: In Progress — Gate C Ready済み。決定論的なShared Acceptance Boundaryの最初のincrementを実装済み（2026-09-21、`tools/reference-shared-acceptance`）。AI Proposalの生成・`ISSUE-000047`の探索結果との接続・Runtime authority系は未着手
- GitHub Issue: #267
- 本格着手条件: Gate C Ready
- Product Acceptance Scenario: **PAS-01 primary / PAS-06 secondary**

## このIssueを最初に読むときの要点

このIssueは、AIに設計を任せるための仕組みを作るものではありません。

中心となる問いは、**AIや他の認知方式が有用な変更候補を作っても、その候補をCanonical authorityへ直接昇格させず、file edit・Studio・CLIと同じ受理境界へ通せるか**です。

```text
既存資産探索 / 設計候補
        ↓
minimum semantic deltaを持つProposal
        ↓
Preview / deterministic checks
        ↓
Reject / Accept
        ↓ Accept時だけ
Shared Acceptance Boundary
        ↓
authoritative source
```

AIが賢いことと、AIにauthorityがあることは別です。

```text
proposal quality
      !=
write authority
      !=
runtime authority
```

### このIssueが減らす不確実性

主に次の問いへ答えます。

1. AIの非決定的な提案を、第二の正本を作らず既存authoring経路へ接続できるか。
2. AIがwhole file / whole modelを再生成するのではなく、人が意味上の差分をreviewできる最小変更として提示できるか。
3. Proposal作成後にsourceが変わった場合、staleな提案を黙って最新版へ適用せず止められるか。
4. Reject / Preview段階ではauthoritative sourceを一切変更せず、Accept後はAIなしでも同じCanonical meaningを再構成できるか。
5. AIのautonomy、Proposal生成能力、write authority、Runtime実行権限を別々に保てるか。

### 完了すると何を判断できるか

このIssueが完了すると、**AIをTEIのauthoring支援へ接続しても、AIだけに特別な正本変更経路やauthorityを与えず、他のauthoring手段と同じAcceptance原則で扱える**と判断できます。

一方、次はこのIssueの完了だけでは言えません。

- AIが自動で変更を採用してよい
- AIがRuntime Commandを直接実行してよい
- 特定のLLM / prompt / vector storeが標準である
- Studio UIが完成している
- Derived Experience KnowledgeがSemantic authorityを持つ

read-only探索と適用可能性判定は[ISSUE-000047](000047-knowledge-first-semantic-exploration.md)、Studioの最小Authoring Loopは[ISSUE-000046](000046-studio-shared-acceptance-authoring-loop.md)が担当します。

## 背景

RQ-015では、生成AIが既存知識を探索してから新規実装へ進むことを要求しています。RQ-016では、AIの提案と採用済みDecisionを区別し、履歴・抑制・変更可能性を持つことを要求しています。

`ISSUE-000008`では、Application Pattern、Driver、第一Web TargetのEvidenceが揃うGate CまではAIの詳細interfaceを固定しない方針です。

AI支援をTEIへ接続する際に先に固定すべきなのはmodel providerやchat UIではなく、**AIの非決定的な候補とCanonical authorityの間に、Editor / CLI / file-native authoringと共有するAcceptance Boundaryを置くこと**です。

この判断は[ADR-0022](../adr/0022-ai-proposal-canonical-acceptance-boundary.md)で採用しています。

本IssueではGate C後、v1 `order_entry`を利用し、AI専用の正本変更経路を作らずにProposalを受理できることをReferenceから実証します。

read-onlyな既存Semantic資産探索そのものは`ISSUE-000047`で扱い、Studioの最小Authoring Loopは`ISSUE-000046`で扱います。本Issueはその探索結果・変更候補を**AI Proposalとしてshared Acceptance Boundaryへ接続する部分**に集中します。

より広い設計では、[`../../docs/definition-centered-feedback-loop.md`](../../docs/definition-centered-feedback-loop.md)に従い、AIを「definition generator」ではなく、**intent解釈・候補比較・説明・minimum semantic delta提案**を担う一つの認知方式として位置づけます。

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
intent / selected local context
  ↓
read-only semantic exploration
  ↓
exact assets + Reference pattern candidates + optional cognition
  ↓
applicable / incompatible / unknown
  ↓
explicit references
  ↓
AI Proposal candidate
```

検索順位、類似度、pattern frequency、AI内部記憶だけをProposalのSemantic根拠にはしません。

既存資産を再利用できる場合は、qualified logical key / Canonical reference / Source / Evidenceを明示してProposalへ接続します。

## AI ProposalはMinimum Semantic Deltaを基本にする

生成AIはwhole fileやwhole modelを容易に生成できますが、それをTEIの標準authoring単位にはしません。

Referenceで優先するのは、current accepted revisionに対する**review可能な最小meaning change**です。

```text
base revision
  + purpose / problem
  + exact target identities
  + minimum proposed meaning change
  + Source / Evidence / candidate rationale
        ↓
Proposal
```

### なぜwhole regenerationを標準にしないか

- 本質的な差分が大量のgenerated wordingに埋もれる
- AIが未要求の意味を補完しやすい
- stale conflict範囲が広がる
- file formatting / comment等の非Semantic差分を増やしやすい
- 人間reviewが「何を決めたか」より「生成物全体の査読」になる

新規artifactや大規模変換など、whole-artifact generationが合理的な場合は否定しません。その場合でもSemantic Diff / accepted meaningを局所的に説明できることを要求します。

### Minimum Semantic DeltaはPatch DSLではない

file edit、Studio action、CLI refactor、AI structured proposalが同じsemantic deltaへ収束できればよく、このIssueで一つのPatch languageを固定しません。

## feedbackはmeaning-owning layerへrouteする

AIが問題を見つけても、すべてをCanonical Semantic sourceへ提案しません。

```text
Business relation / constraint missing
  -> semantic / business authoring candidate

Target-only UI friction
  -> target / projection candidate

Provider mismatch
  -> binding / realization candidate

actual DB drift
  -> observation / mapping / migration candidate

retrieval miss
  -> derived cognition / index candidate
```

AIの役割はvisible symptomをもっともらしく直すことではなく、**どのconcernがmeaningを所有するかを明示し、候補を正しいAcceptance経路へ送ること**です。

Routing自体が不明なら`unknown`として人へ返し、最も書きやすいfileへ変更案を作りません。

## 最初のReference

最初のReferenceでは、[`../DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`](../DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md)に従い、同じ`order_entry`について小さいCanonical change candidateを扱います。

候補例はGate C後に実資産から決めますが、次のような範囲を想定します。

- Validation ruleの小さい追加・修正候補
- 既存Capability Bindingの明示化候補
- Relation等の小さいauthoring変更候補
- Business Procedure / Application cross-view contradictionの修正候補
- Presentation / Interaction Projection hint候補。ただしTarget-onlyならSemantic sourceへ押し込まない

既存のauthoritative sourceをAI固有形式へ移し替える作業は選びません。

AIが提案する変更と、`ISSUE-000046`のStudio操作、file-native edit、CLI / refactor等が、同じAcceptance原則へ収束できることを比較します。

## Proposalに必要な情報の候補

Proposal schemaは未確定ですが、Referenceでは少なくとも次を保持できる必要があります。

- proposal identity
- purpose / observed problem
- target activity / artifact
- referenced qualified logical keys / Canonical IDs
- Source / Evidence references
- proposal generation source: exact / Reference pattern / cognition / LLM / human
- proposed minimum semantic change
- unchanged assumptions / unresolved where relevant
- feedback routing target concern
- `baseModelRevision`
- AI mechanism / type
- autonomy level
- proposer / actor information
- audit information

model名、prompt、trajectory等をCanonical subjectのidentityへ含めません。

AIのautonomy levelとauthorityも分けます。Proposal metadata上の自律度から、write / invoke / Resource permissionを自動導出しません。

## Replaceable Decision PolicyのPressure

本IssueでAIを使う場合も、AI専用のRuntime action semanticsを作りません。

```text
Rule policy ----┐
                ├-> same validated semantic action
AI policy ------┘
                       ↓
              same Validation / Authorization
                       ↓
              same Capability / Result / Observation
```

Referenceでは、少なくとも一つの既知operationについてRule / deterministic pathとAI-assisted pathが同じsemantic action boundaryへ収束できるかを比較します。

次を禁止方向とします。

- AI policy専用Capability
- AI policy専用Permission semantics
- raw tool callをApplication Action identityにすること
- policy固有のBusiness Outcome / Observation model
- policy selection scoreをexecution authorityへ昇格すること

Rule / AIのどちらがactionを選んでも、effect時点のcurrent authority checkを通ります。

これは汎用Planner / Policy Engine / Action Catalog schemaを先行実装する要求ではありません。Gate C後のsmall Referenceで二重化が発生するかをPressureし、必要なformal conceptはEvidence後に判断します。

### Hybrid Intent FulfillmentのPressure

同じknown operationを毎回AIへ送ることも標準にはしません。

Referenceでは、少なくとも一つのintentについて次を比較できる余地を残します。

```text
A: exact / deterministic path
B: AI-assisted path
C: hybrid routing
```

期待する境界は次です。

- exact identity / known rule /既知operationで解ける場合はdeterministic pathを利用できる
- ambiguous / compositional intentだけをAI reasoning候補へ送れる
- A / B / Cのどの経路でも同じsemantic action / Validation / Authorization / Capability / Result境界へ戻る
- routing choiceやrouting scoreをSemantic applicability / Acceptance / execution authorityへ昇格しない
- AI call count削減だけを目的化せず、success / semantic correctness / latency / cost / retry / authority rejection等を分けて観測する
- 外部研究の固定比率やthresholdをTEIのKPIへ輸入しない

この比較のためにproduction routerや固定heuristicを先行実装しません。

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
- persistence / realization impact where relevant
- Validation / Architecture Diagnostic
- unresolved / contradiction
- why each derived effect follows from the proposed definition change

PreviewではAIによる自然言語説明を補助的に使えますが、impact setそのものは可能な限りexact dependency / deterministic projectionから得ます。

### Reject

Rejectではauthoritative sourceを変更しません。

AI固有store、chat履歴、trajectory、cognitive index等を削除してもCanonical Worldが変わらないことを維持します。

Reject理由をexperience memoryとして保持する場合も、次回のcandidate rankingを助けるhintであってsemantic ruleではありません。

### Accept

Accept時だけshared Acceptance Boundaryを通り、authoritative sourceを更新します。

受理後はAIなしでsourceをreloadし、同じCanonical meaning / Validation / Conformanceを再確認できることを要求します。

受理後のUI / index / matrix / generated artifactはaccepted sourceから再生成可能でなければなりません。

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

- 元のCanonical ID / Source / Evidenceへのprovenance
- dependency / revision fingerprint
- invalidationまたは再評価可能性
- similarity / routing結果とauthoritative Decisionの分離
- 再利用候補もAcceptance Boundaryを通ること

Derived Experience KnowledgeはCanonical Knowledgeの第二正本ではありません。

### local learningを先に比較する

将来学習を導入する場合、いきなりglobal policyをlearnしません。

先に比較するのは:

- task-local candidate association
- known false-positive suppression
- accepted / rejected patternの局所ranking hint
- outcome-local retrieval weight

です。

これらを削除してもaccepted semanticsが変わらないことを要求します。

## Authority / Runtime境界

AIがProposalを生成できることと、RuntimeへCommandを送れることを同じauthorityにしません。

さらに、ProposalがShared Acceptance Boundaryを通ってaccepted meaningになったことも、その後のeffectful operationに対する継続的なRuntime authorityではありません。

```text
t0: Proposal accepted
t1: Permission / Resource authority changes
t2: effectful operation attempts execution
```

ReferenceでRuntime操作まで扱う場合は、t2でcurrent Authorization / Permission / Resource Lease / ownership / fencing等を決定論的に再評価します。t0のapprovalやProposal metadataをcurrent authorityの代わりにしません。revoked / stale / missing / ambiguousなら副作用前にfail closedし、reasonとcurrent authority referenceをEvidenceへ残します。

このpressureはRQ-019 refinement候補ですが、本Issueだけを根拠にRequirement正本や新しいapproval token / permission schemaを追加しません。

Runtime操作が必要なReferenceを将来追加する場合も、Semantic Capability、Authorization、Runtime Binding、Resource Permission、Business Result / Execution Error / Observation境界を利用します。

raw tool callをTEIのauthority modelの代わりにはしません。

Observationから変更候補を作る場合も、`Observation -> semantic write`へ直結せず、feedback routing / Proposal / Acceptanceを通します。

## Evidence

完了時には少なくとも次を残します。

- Proposal input / normalized representation
- selected definition / local context
- Proposalが参照したqualified logical key / Canonical ID / Source / Evidence
- decision policy / selection path（Rule / deterministic / AI / hybrid等、比較する場合）
- routing reason / candidate mechanism（hybrid比較を行う場合）
- selected semantic action / Capability reference（Runtime actionを扱う場合）
- candidate mechanism / Reference pattern provenance where used
- feedback routing target
- `baseModelRevision`
- minimum semantic delta
- Preview Semantic Diff / impact / derivation reason
- Accept / Reject結果
- authoritative source変更の有無
- source / model revision after
- reload後のCanonical結果
- Validation / Conformance結果
- stale Proposal negative case
- Runtime操作を扱う場合のeffect-time authority reference / rejection reason
- Proposal acceptance後にauthorityを変化させるnegative case（扱う場合）
- AI経路を外した同一Acceptance Boundaryの利用例
- AI / derived memory削除後もCanonical meaningが変わらないこと

AI provider固有のpromptやtrajectoryをEvidenceとして残す場合も、secretや不要なprivate dataを保存しません。

## 実証状況

### 最初のincrement — 決定論的なShared Acceptance Boundary（2026-09-21）

[`tools/reference-shared-acceptance`](../../tools/reference-shared-acceptance/README.md)が、AIを含まない決定論的な受入境界をReferenceとして実装した。candidateの入口（file / CLI / Studio / AI）は`origin`で区別するだけで、同じ検査・同じ順序・同じ書込経路を通る。差分・依存影響の解析は[`tools/internal/changeanalysis`](../../tools/internal/changeanalysis)（`ISSUE-000039`の最初の解析境界）が担当し、このtoolは呼ぶだけである。Gate B / Gate Cが固定した`reference/workspace`は変更せず、scratch copyに対して実行する。

最初のcandidateは、`order_entry`が使うsemantic namespace `sales`へ`positive` Validation（order lineの単価）を一つ追加する最小deltaである。最初は`add-validation`だけを扱った。同じ日の次のincrementで、Second Application（`ISSUE-000040`）が必要とする型付きoperation `add-data`（新しいnamespaceへのData）/ `add-application` / `add-reference-pattern`を、別の受入経路を作らずに同じ境界へ追加した（新しいfileを作るため、失敗時は作ったfileとdirectoryを戻す。変わるべきWorld collectionはdeltaから導き、exactに一致しなければ拒否する）。さらに`ISSUE-000046`が最小loopとして名指しする`add-relation`（既存のDataへRelationを一つ追加）を加えた（Persistence Mappingへの影響と保存方法の未決定を、Previewの辺と`unresolved`で示す）。Relationの変更・削除等は必要になった時点で追加する。

| 完了条件 | 状態 | 根拠・残り |
| --- | --- | --- |
| 1 `ISSUE-000047`の探索結果からexplicit-reference Proposalを作れる | 未着手 | candidateは人 / CLIが書く。探索結果からの生成は`ISSUE-000047`との接続後 |
| 2 ProposalそのものはCanonical authorityではない | 実証済み | Preview / Rejectはauthoritative sourceをbyte単位で変更しない |
| 3 qualified logical key / Canonical ID / Source / Evidenceへtraceできる | 一部 | candidateのevidenceが存在するsubject / sourceへ解決できなければ拒否する。根拠を探索結果から自動で得る部分は未着手 |
| 4 whole model regenerationを標準にせずminimum semantic deltaを扱う | 実証済み（operation kind 5種） | 宣言外のsemantic変更・World collection変更は拒否、変更subject数に上限（8件） |
| 5 AI Proposalと他の入口が同じAcceptance原則へ収束 | 一部 | 入口が違っても同じfingerprint・同じsource・同じCanonical Worldになり、AI固有Evidenceは入らない。実際のStudio / AI producerは未実装 |
| 6 feedbackをmeaning-owning concernへroute | 一部 | Validationはsemantic層でだけauthorでき、別layerへのplacementは拒否する。一般的なrouting catalogの実行は未着手 |
| 7 Preview / Rejectでauthoritative source不変 | 実証済み | |
| 8 Preview impactがexact dependency / deterministic derivationへtrace | 実証済み（Validation追加） | Attribute → Data → Relation → Pattern、namespace利用Application。Driver / Target影響は`unresolved`として明示 |
| 9 stale `baseModelRevision`を検出 | 実証済み | 4つの分岐先を示し、黙って適用しない |
| 10 受理後のsourceをAIなしで再読込 | 実証済み | 共有loader / Validationだけで再読込する |
| 11 reload後のCanonicalがValidation / Conformanceを満たす | 実証済み | 受理前後で評価結果が変わることも確認 |
| 12 derived projection / indexをaccepted sourceから再生成 | 一部 | Canonical World JSONを再生成し、決定論的であること・削除後に再生成できることを確認。Source Index等の他派生物・Target Projectionは未確認 |
| 13 file moveとlogical renameを区別 | 実証済み | file移動はsemantic差分なし、改名はremoved / added |
| 14 mandatory authored Stable IDを導入しない | 実証済み | 追記されるsourceはqualified logical keyだけ |
| 15 autonomyとwrite / invoke / Resource authorityを分離 | 未着手 | |
| 16 AI固有Evidence / Derived Experience Knowledgeを削除してもCanonical不変 | 一部 | AI固有のnotesはWorld / sourceへ入らない。Derived Experience Knowledge自体は未実装 |
| 17〜23 Runtime authority・hybrid routing | 未着手 | |

Previewが判断できなかった事項は成功へ読み替えず`unresolved`に残す。特に、`order_entry`は`uses validation`を宣言しておらず、現行のDriver / Canonical modelはApplicationが評価するValidationの選択を宣言から導出していない（`APPLICATION_VALIDATION_SELECTION_NOT_CANONICAL`）。Validationを追加しただけでは実行時に評価されるとは言えない、というこのReferenceの発見は、次のincrementで扱う。

## qualification Evidenceとの接続候補 — 2026-10-03

`ISSUE-000047`のdeterministic qualificationから既存のShared Acceptance Boundaryへexact Evidenceを渡すcandidate実装を、primary workstreamの`work/issue-000040-second-application-reference`へ統合している。

接続では新しいProposal schemaを作らず、既存candidateのEvidenceを後方互換で精密化する。`applicable`なqualificationからsubject kind、Source Index identity、source path、source fingerprint、評価したrequirementsを付与する。required dependencyを明示した場合は、そのdependencyのauthoritative sourceとfingerprintもscoped Evidenceとして保持する。

Previewではcurrent Source Indexへ戻り、同じrequirementsでqualificationを再実行する。candidate自身のsourceだけでなくrequired dependencyのEvidenceもcurrent qualificationと比較するため、dependencyのIDが同じままsource内容が変わった場合をfail closedにできる。requirements外のKnowledgeはこのhandoffへ含めず、無関係なsource変更をqualification freshnessへ持ち込まない。

Accepted Decisionへ保存したEvidenceもDecision Indexの読込時にcurrent Sourceへ再照合する。Decisionの`current / superseded`、そのDecisionが書いたsourceの`intact / diverged`、受理理由に使ったEvidenceの`current / diverged`を分け、historical Acceptanceとcurrent applicabilityを混同しない。Evidence本文は正規化後にdedup・sortし、保存時の`evidenceFingerprint`とも照合するため、Decision記録だけを書き換えた場合も別の不整合として検出できる。

この方式でもqualification結果をapprovalとして扱わない。`candidateFingerprint`はbase revisionとminimum semantic deltaだけを表し、Evidenceは含めない。一方、正規化済みEvidenceには別の`evidenceFingerprint`を与え、PreviewとDecisionの間で同じ根拠identityを保持する。これにより「意味は同じだが、承認時に提示した根拠が違う」状態を、意味identityを壊さず機械的に区別する。

現時点ではexact-head full Repository-healthを通したProduct Evidenceではないため、完了条件1や3の状態はまだ更新しない。

## Semantic Evaluation Evidenceとの接続候補 — 2026-10-05

Semantic Extension Referenceで得たEvaluation結果をShared Acceptanceへ接続するcandidateを追加した。ただし、Evaluationの`pass / fail / unknown`をAcceptance authorityへ昇格しない。

接続では新しいAI専用ProposalやAcceptance engineを作らず、既存candidateの`Evidence`へartifact-only itemを追加する。Semantic Evaluation EvidenceはWorkspaceのAuthoring sourceではないため、既存の`subject / source / sourceFingerprint`へ偽装しない。Source Index Evidenceとartifact Evidenceを同一itemへ混在させず、別のauthority shapeとして保持する。

producerであるSemantic ExtensionとconsumerであるShared Acceptanceは、`tools/internal/semanticevidence`の同じschema / normalization / fingerprint contractを使う。これによりEvaluation Evidenceの型とfingerprint規則を二重実装しない。

artifact handoffでは、Semantic Set、Constraint、subject、validator、Evaluator Interface / contract、provider Extension、implementation / artifact digest、Conformance Evidence、input、status、diagnostic、Evaluation Evidence fingerprintを検査する。その正規化済みpayload全体へShared Acceptance側の`artifactFingerprint`も与え、受理時に見たartifactをexactに固定する。

このartifactは`candidateFingerprint`には入らず、minimum semantic deltaのidentityを変えない。一方、`evidenceFingerprint`とReview Projectionには入るため、「意味差分は同じだが、判断時に見たEvaluation Evidenceが違う」状態を区別できる。Review A3にはstatus / diagnostic、A7にはartifact / Semantic Set / evaluation fingerprintを表示し、Accepted DecisionにもEvidence本文を残す。

重要な停止線は次である。

```text
semantic evaluation pass
  != Acceptance

semantic evaluation fail / unknown
  != automatic Reject

recorded evaluation artifact
  != current evaluator rerun
  != current Conformance revalidation
  != Runtime authority
```

Decision IndexもSource Evidenceとartifact Evidenceを別statusで返す。Source Indexへ再照合できる根拠は`sourceEvidenceStatus = current / diverged / none`、Evaluation artifactは`artifactEvidenceStatus = recorded / invalid / none`とし、artifactだけを「current」と呼ばない。両方が健全に併存する場合、集約`evidenceStatus`は`current-with-recorded-artifacts`とする。

このcandidateは「Evaluation結果をレビュー可能な根拠として同じShared Acceptance Boundaryへ運べるか」をpressureするものであり、Semantic Evaluation Evidenceのcurrent freshnessを再実行する仕組みではない。将来current validityを要求する場合は、Semantic Set / Validation Plan / Conformance Evidenceを再取得してEvaluatorを再実行する別検査を追加する。

exact-headのfocused test / full Repository healthは現在の実行環境では未取得であるため、本追記だけで完了条件やProduct Evidence tierを上げない。

## Authoring Decision Authority Evidence Gate — 2026-10-05

PV-AI-01の次段としてhuman Decision witnessを検討した結果、自己申告のactor fieldを追加する前にauthoring側のauthority Evidenceを確定する必要がある。

正本は[Authoring Decision Authority Evidence Gate](../AUTHORING_DECISION_AUTHORITY_EVIDENCE_GATE_2026-10.md)へ分離した。

current Shared Acceptanceで既に確定しているのは、candidate / Evidence / Review Projection / Accept・Reject Decisionのidentityとprovenanceである。一方、次はまだ未解決である。

```text
candidate origin actor
  != decision actor

decision actor identity
  != authoring decision authority

reviewFingerprint matched
  != human witness verified
```

authoring側でverified witnessを扱うには、少なくともactor identity Evidence、workspace / projectとdecision semanticsへ束縛されたauthority Evidence、Decision時点のfreshness / revocation、exact candidate / Review bindingが必要である。Runtime Actor Contextのprincipal / AuthorizationやR3AN / R3AOのRuntime assertion authorityを、そのままauthoring Decisionへ流用しない。

current RepositoryにはこのEvidence Gateを満たすaccepted authoring authority sourceがない。したがって`origin.actor`、CLI引数、`approved=true`、review fingerprint一致をhuman authorityへ昇格する実装は行わない。

次のReferenceではproduction schemaを先に作らず、Studioのauthenticated user contextまたはrepository / organization側のreview authority等から、実在するidentity / authority Evidenceを一つ取得できるかをPressureする。providerを選べなければ`authority-unresolved`のまま止める。

## Gate Cとの関係

本IssueはGate C前に本格実装しません。

Gate Cから少なくとも次を受け取ります。

- Handover Baseline / Application identity
- 第一Web Targetで実証したSource-to-Canonical trace
- Semantic / Functional / Visual Evidenceの分離
- Target固有差 / unresolved

Proposal schema、API、AI provider、Editor UI、承認UXは、その実資産から決めます。

Gate C後は`ISSUE-000046` / `ISSUE-000047`と同じone-change Referenceへ合流し、AIだけ別scenarioで検証しません。

## 完了条件

次をReferenceとConformanceから説明できることを完了条件とします。

1. `ISSUE-000047`のknowledge-first探索結果からexplicit-reference Proposalを作れる
2. ProposalそのものはCanonical authorityではない
3. Proposalがqualified logical key / Canonical ID / Source / Evidenceへtraceできる
4. AIがwhole model regenerationを標準にせずminimum semantic deltaを提案できる
5. AI ProposalとStudio / file-native / CLI候補が同じAcceptance原則へ収束できる
6. feedbackをmeaning-owning concernへrouteし、Target / Provider問題をSemanticへ誤投入しない
7. Preview / Rejectではauthoritative sourceを変更しない
8. Preview impactが可能な範囲でexact dependency / deterministic derivationへtraceできる
9. stale `baseModelRevision`を検出できる
10. 受理後のauthoritative sourceをAIなしで再読込できる
11. reload後のCanonical meaningがValidation / Conformanceを満たす
12. derived projection / indexをaccepted sourceから再生成できる
13. file moveとlogical renameを区別できる
14. AI都合でmandatory authored Stable IDを導入しない
15. AIのautonomyとwrite / invoke / Resource authorityを分離できる
16. AI固有EvidenceやDerived Experience Knowledgeを削除してもCanonical Worldが変わらない
17. ProposalからRuntimeへ無権限に直接到達できない
18. Runtime操作を扱う場合、Proposal acceptance /過去approvalではなくeffect直前のcurrent authorityで許可可否を判定できる
19. acceptance後にauthorityがrevoked / stale / missing / ambiguousになったnegative caseで、副作用前にfail closedできる
20. Rule / deterministic pathとAI-assisted pathを比較する場合、同じvalidated semantic action / Capability / Result / Observation境界へ収束できる
21. policyごとに別Capability / Permission semantics / Business Outcome / Observation modelを作らない
22. known operationをdeterministic pathで処理できるcaseとAI-assisted pathを必要とするcaseを区別できる
23. hybrid routingを比較する場合、routing choice / scoreがSemantic applicability・Acceptance・execution authorityを上書きしない

## 非目標

- autonomous agent frameworkの完成
- chat-first IDEの完成
- AI providerの標準化
- vector databaseの標準化
- AIによる無確認自動変更
- AIによる無確認のside effect実行
- whole-model regenerationを標準authoring経路にすること
- prompt / model metadataのCanonical化
- AI専用Acceptance engine
- mandatory UUID / Stable IDの導入
- Patch-only authoring modelの固定
- global online learning
- 外部研究のsuccess rateやtoken削減率をTEIの固定KPIにすること

## 関連文書

- `docs/definition-centered-feedback-loop.md`
- `plan/DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/TECH_RESEARCH_INTEGRATION_2026-09.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/issue/000008-editor-and-ai-followup-note.md`
- `plan/issue/000046-studio-shared-acceptance-authoring-loop.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `docs/architecture.md`