# ISSUE-000082: 文脈付きEpistemic AnnotationとProject Knowledge Healthを実証する

- 状態: In Progress — ADR-0030採用、TEI / SUI / SEIの横断Issue起票済み。Reference実装は未着手
- GitHub Issue: #958
- Cross-repository Issues: SUI #3184 / SEI #660
- 関連ADR: ADR-0030

## このIssueを最初に読むときの要点

このIssueは、生成AIに世界の真偽を判定させる機能を作るものではありません。

中心となる問いは、**人間が与えた前提、AIが生成した未確認情報、人間が確認した情報、仮説、誤りを、適用文脈を失わず保存し、後続のAIが同じ強さの事実として扱わないようにできるか**です。

特に次の二つを同時に満たします。

1. ユーザーが事実として入力した情報は、その作業文脈では通常の前提として利用する。
2. AIが生成した情報は、保存されたことだけではreview済みの事実へ昇格しない。

さらに、どちらも別のproject、version、environmentへ無条件に一般化しません。

## Product value

生成AIを継続的に設計へ使うと、誤ったAI出力が文書へ残り、その文書を別のAIが事実として読む連鎖が起こり得ます。

本Issueでは、この連鎖をPrompt上の注意書きではなく、保存、検索、review、context assemblyの契約で止めます。

利用者側では、大量出力へ細かいmetadataを手入力させません。AI出力は自動的にunreviewedとし、利用者は確認できた範囲だけを高速にreviewします。

## 設計上の境界

### Truth Engineにはしない

TEIは世界の客観的真偽を判定するAuthorityになりません。

扱うのは、誰が、どの文脈で、いつ、どの状態としてそのassertionを利用できるとしたかです。

### originとreview stateを分ける

user由来であることは客観的検証済みを意味しません。ただし、現在の作業文脈ではユーザーが与えた前提として扱います。

AI由来であることは誤りを意味しません。ただし、人間review等のEvidenceなしにreview済みへ昇格しません。

### review stateとscopeを分ける

review済みのassertionも、scope外では自動的に利用しません。

たとえば次の二つは、同じ意味ではありません。

```text
reviewed in project A / production
  !=
reviewed in project B / external API
```

### relevanceとapplicabilityを分ける

検索結果が意味的に近いことと、現在の文脈で前提にできることは別です。

## 最初のReference fixture

一つのsubjectについて、少なくとも次の状態を同時に用意します。

1. current projectでのuser factual assertion
2. AI unreviewed assertion
3. human-reviewed assertion
4. human-reviewedだがscopeが狭いassertion
5. stale assertion
6. superseded assertion
7. reviewed同士のConflict
8. rejected assertion

同じfixtureをTEI、SUI、SEIで共有できる形へ寄せます。ただし、Repository固有のUIやruntime modelをfixtureの正本にしません。

## TEI側の最初の作業

### E0: sidecar contract

paragraph、card、row、selected range等の対象へ、Canonical meaningを複製せずにepistemic metadataを結び付けられる最小modelを定義します。

最低限必要な情報は次です。

- target / subject reference
- origin
- assertion mode
- review state
- explicit / inferred context
- valid / checked time
- Source / Evidence / review record
- supersession / contradiction relation

実装前に、既存Source Index、Evidence、Decision Index、Semantic Extensionのどれを再利用できるか確認します。

### E1: state transition

少なくとも次を表現します。

```text
AI generated
  -> unreviewed
  -> reviewed / rejected / conditional

user factual assertion
  -> native assertion in current context
  -> reviewed when explicit review is needed

reviewed assertion
  -> semantic edit
  -> review validity re-evaluation
```

単純な文字編集だけで全reviewを失効させる実装にはしません。意味変更の判定は既存Semantic Diff等へ接続できるか確認します。

### E2: context compatibility

同じassertionを別project、branch、version、environmentへ持ち出し、scope mismatchを検出します。

inferred contextを利用する場合、その推定結果と根拠を保持し、後から訂正できるようにします。

### E3: Project Knowledge Health

最初のHealthは単純なpercentだけにしません。

少なくとも次のnegative fixtureを持ちます。

- 多数の低重要度unreviewed memo
- 少数のcritical unreviewed dependency
- reviewedだがstaleなcritical dependency
- reviewed同士のConflict
- scope mismatchしたcritical dependency

低重要度memoが多いだけのcaseより、critical dependencyが未確認のcaseを強くalertできることを確認します。

### E4: review request handoff

SUI #3184へ渡すreview request projectionを定義します。

少なくとも次を含みます。

- assertion
- current state / scope
- reason
- downstream impact
- Source / Evidence
- reviewer candidate
- unresolved / conditional state

reviewer candidateをAuthorityにはしません。

### E5: SEI context assembly handoff

SEI #660へ、semantic relevanceとepistemic / context signalsを別々に渡せることを確認します。

TEI側で一つの最終scoreへ固定しません。

## Plugin / Extension方針

最初からCore enumを増やしません。

ISSUE-000077 Semantic ExtensionとISSUE-000080 / #951 Plugin Capability contractを利用し、次を分けます。

```text
Epistemic meaning / state contract
  -> Semantic Extension候補

review / health / context evaluation
  -> Tool Capability / Plugin候補

SUI projection
  -> consumer UI

SEI routing / assembly
  -> consumer cognition
```

`std.epistemic`は候補名に留め、Referenceで第二consumerまで成立してから標準namespaceへの昇格を判断します。

## SUI #3184へのhandoff

SUIでは次を実証します。

- AI出力へunreviewedを自動付与
- user factual assertionへ通常の再承認操作を要求しない
- selection + 1操作
- bulk review
- conditional review / scope refinement
- semantic edit後の再確認表示
- Knowledge Health projection
- stakeholder review queue

SUI固有のTruth modelは作りません。

## SEI #660へのhandoff

SEIでは次を実証します。

- context-compatible retrieval
- status-aware context assembly
- Conflict / stale / supersededの保持
- narrow scopeからwide scopeへのgeneralization抑制
- review escalation候補
- RecommendationとAuthorityの分離

similarityやmodel confidenceをTruthへ昇格させません。

## 既存設計との接続

### #267 / ADR-0022

AI ProposalとCanonical Acceptanceを分離する既存境界は維持します。本Issueのreview済み情報が存在しても、それだけでCanonical changeをAcceptedにはしません。

### ISSUE-000047

Knowledge qualification / freshnessを再利用します。epistemic stateを既存qualificationへ無理に押し込まず、applicabilityとreview stateを分けたまま接続します。

### ISSUE-000077

新しい意味をCore enumへ追加せず、Semantic Extensionとしてpressureできるか確認します。

### ISSUE-000080 / #951

review、health、context evaluation等をPlugin-defined Capabilityとして提供できるか確認します。

## 完了条件

- [ ] user factual assertionとAI unreviewed assertionを区別できる
- [ ] AI生成主張が保存だけでreviewedへ昇格しない
- [ ] origin / assertion mode / review state / scope / time / Evidenceを分離できる
- [ ] explicit contextとinferred contextを区別できる
- [ ] narrow scopeのfactをwide scopeへ黙って一般化しない
- [ ] semantic edit後にreview validityを再評価できる
- [ ] AI summaryへreview stateを無条件継承しない
- [ ] contradictionを保持できる
- [ ] retrievalへrelevanceとcontext compatibilityを別signalとして渡せる
- [ ] Project Knowledge Healthがcritical dependencyを考慮できる
- [ ] stakeholder review requestへimpact / Evidence / scopeを渡せる
- [ ] conditional reviewでscope refinementを保持できる
- [ ] SUI #3184とSEI #660が同じcontractを利用できる
- [ ] Core固定enum追加なしで最初のReferenceを成立させられる
- [ ] Extension / Pluginを外しても既存Canonical meaningが変化しない

## 非目標

- 汎用Truth Engine
- user inputの一律外部検証
- 全AI出力の人手確認必須化
- review率100%の要求
- reviewer推薦による自動承認
- 初回から全文をClaim graph化すること
- epistemic metadataをCanonical業務意味の第二正本にすること

## 関連

- GitHub Issue #958
- ADR-0030
- SUI Sensemaking #3184
- SEI Cognition #660
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
- `plan/issue/000077-semantic-extension-reference.md`
- ISSUE-000080 / GitHub Issue #951
