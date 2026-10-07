# ISSUE-000082: 文脈付きEpistemic AnnotationとProject Knowledge Healthを実証する

- 状態: In Progress — ADR-0030採用、TEI / SUI / SEIの横断Issue起票済み。First Reference candidateはDraft PR #961で実装中。exact-head full Repository Healthは#962で未完了
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

### 情報の働きと確認状態を分ける

事実でない情報が、すべて仮説になるわけではありません。設計上の要件、決定、制約は、世界についての事実とは異なりますが、後続作業の前提になります。

最初のReferenceではfact中心のpressureを置いていますが、次のincrementでは少なくともfact / observation / requirement / decision / constraint / hypothesisを区別します。ユーザーが現在の作業で直接与えた要件や決定は通常の作業前提にできますが、組織上の正式AuthorityやCanonical Acceptanceとは分けます。

### 入力者と内容の出所を分ける

ユーザーがAI出力や外部資料を貼り付けた場合、入力操作をした人と内容の本来の出所は異なります。入力主体だけをoriginとして使うと、貼り付けたAI文章をユーザー自身の主張へ誤変換できます。

Referenceでは、content provenanceとingestion actorを別に表し、出所不明のlegacy資料をユーザー由来へ推定しません。

### 検索上の関連度と現在文脈での適用可能性を分ける

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

### E0a: statement kind / provenance contract

E0のsidecar contractを、事実だけでなく設計作業に必要な情報の働きへ広げます。

少なくともfact、observation、requirement、decision、constraint、hypothesisを別に扱います。content provenanceとingestion actorも分けます。ユーザーがAI出力を貼り付けたcaseをnegative fixtureにし、取り込み操作だけでuser factual assertionへ変わらないことを確認します。

### E0b: context provenance

context bindingは、systemが直接観測したobserved、userやauthoritative sourceが明示したdeclared、AIやheuristicが推定したinferredを区別します。inferred contextだけで適用範囲を広げません。

### E0c: target anchoring / review history

paragraph、card、row、selected range等へのannotationは、編集後の再特定を考慮します。positionだけへ依存せず、利用可能なsubject identity、source revision、structure、quoted text等を組み合わせます。

同じ情報への複数回reviewは上書きせず、review eventとして保持します。現在の確認状態は、対象meaning、scope、time、review eventから導出する構造をpressureします。

### E1: confirmation history

現在の確認状態を単一fieldで上書きせず、review eventの履歴から導出します。

```text
AI generated
  -> no confirmation event
  -> unreviewed

human confirm
  -> confirmed
human reject
  -> rejected
human withdraw
  -> unreviewed

confirm
  -> reject
  -> confirm
  -> historyを残したままcurrent confirmation = confirmed
```

user-originかつuserが直接入力したworking-premise statementは、現在文脈では通常の前提として利用できます。ただし、確認eventがなければconfirmationはunreviewedのままです。「作業前提として使える」と「人間が確認済み」を同じ状態へしません。

review eventは対象meaning fingerprintへ結び付けます。semantic editで現在meaningとのbindingが切れた場合、旧eventは履歴として保持しつつ再確認対象に戻します。単純な文字編集だけで全reviewを失効させる実装にはせず、意味変更の判定は既存Semantic Diff等へ接続できるか確認します。

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

さらに、annotationされた情報だけでHealthを計算しません。重要な文書や判断領域が未分類、出所不明、対象不明であれば、epistemic coverage不足として別に観測します。coverageが不明な領域を「問題なし」と数えません。

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

### E6: legacy / export-import boundary

既存資料と外部連携ではmetadataが存在しない、または途中で失われる場合があります。

最初の境界として、provenance不明のlegacy資料をreview済みへ推定しないこと、metadataを保持できるexportでは本文と一緒に運ぶこと、保持できないexportでは喪失を明示すること、metadataなしで再importした本文へ過去のreviewを自動復元しないことを確認します。Evidenceやreview情報も元資料のaccess controlを迂回して露出させません。

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

Knowledge qualification / freshnessを再利用します。epistemic stateを既存qualificationへ無理に押し込まず、applicability、confirmation、lifecycleを分けたまま接続します。

### ISSUE-000077

新しい意味をCore enumへ追加せず、Semantic Extensionとしてpressureできるか確認します。

### ISSUE-000080 / #951

review、health、context evaluation等をPlugin-defined Capabilityとして提供できるか確認します。

## 2026-10-06 First Reference candidate — E0〜E4

このcandidateは、fact中心の初期版から、statement kind、content provenance / ingestion actor、observed / declared / inferred context、複数review eventまでpressureを広げました。2026-10-07の再点検で残ったtarget anchoring、epistemic coverage、legacy / export-import境界は、引き続き未実証です。

確認状態については、初期版のnative / reviewed / conditional / rejected / supersededという単一stateを廃止しました。review eventから導出するconfirmation、meaningへのreview binding、supersedes relationから導出するlifecycleを別軸として扱います。

ADR-0030採用後の最初のTEI側Reference candidateを、既存の `work/issue-000081-contextual-epistemic-knowledge` lane上に追加した。Issue改番前から存在するlaneを再利用しており、同目的のsibling branchは作らない。

実装位置:

- `tools/reference-epistemic-knowledge/`
- `testdata/epistemic-fixture-v0.json`
- `testdata/context-current.json`
- `testdata/context-other-project.json`

このcandidateはCanonical Worldへepistemic fieldを追加しない。入力sidecarからread-only Projectionを生成し、削除しても既存Canonical meaningへ影響しない構造を維持する。

### E0 sidecar contract candidate

一つのtruth / confidence fieldへ畳まず、少なくとも次を別fieldとして保持する。

- target
- content origin
- ingested by
- statement kind
- observed / declared / inferred context
- valid / checked time
- Evidence
- append-only review events
- supersedes / conflictsWith
- downstream dependency criticality

現在値としてのconfirmationは保存せず、review eventから導出する。supersessionもconfirmationへ混ぜず、relationからlifecycleを導出する。

schemaはReference限定の `tei.reference.epistemic-corpus/v0` とし、現時点で `std.epistemic` やCore enumへ昇格させない。

### E1 confirmation history candidate

user-originかつuserが直接入力したfact / observation / requirement / decision / constraintは、current scope内のworking premiseとして扱える。ただし、review eventがなければ `confirmationState=unreviewed` のままにする。AI-origin contentはユーザーが貼り付けてもdirect user premiseにはせず、保存や取り込みだけでconfirmedへ昇格させない。

review eventは `confirm / reject / withdraw` を持ち、positiveで一意なsequenceに従って履歴を保持する。v0では `method=human-review` のeventだけを受け入れる。sequenceが増えているのに `occurredAt` が過去へ戻る履歴は拒否する。

現在の `confirmationState` は、現在の `meaningFingerprint` に結び付く最新eventから導出する。過去meaningに対するeventしか存在しない場合は `reviewBinding=stale` とし、旧確認を現在値へ継承せず `review-required` とする。

このReference自身はarbitrary proseのsemantic diffを発明しない。meaning fingerprintは対象surface / transformation contractから受け取る入力であり、一致そのものをsemantic proofへ昇格させない。format差とsemantic差を判別できない場合に確認の有効性を推測しない。

### E1b supersession / conflict candidate

旧assertionのconfirmationを変更してsupersessionを表現せず、後続assertionの `supersedes` relationから `lifecycleState=superseded` を派生する。Projectionには `supersededBy` を残し、relation provenanceへ戻れるようにする。

Conflictもrelationとして保持し、current contextで両方が有効候補になる場合は一方を自動採用せず `review-required` とする。

### E2 context compatibility candidate

assertion scopeとcurrent contextをdimensionごとに照合する。

- 全必要dimension一致: `compatible`
- 値が異なる: `incompatible`
- 必要dimensionがcurrent contextに無い: `unknown`

`unknown` を「たぶん同じ」としてpremiseへ昇格させない。observed / declared / inferred contextを区別し、inferred contextを使ったassessmentにはsignalを残す。

### E3 Project Knowledge Health candidate

Healthはreview率を主判定にせず、`dependencies` のcriticalityと現在のuse stateを組み合わせる。

critical consumerがunreviewed / stale / Conflict / context不適合なassertionへ依存する場合は `critical` とする。低重要度unreviewed memoが多数存在しても、critical dependencyを件数比で希釈しないnegative testを置いた。

### E4 review request handoff candidate

review-required assertionと、high / critical dependencyがpremiseとして利用できないassertionをreview requestへ投影する。

Projectionは次へ戻れる。

- assertion identity / target
- derived confirmation state / review binding
- scope
- Evidence
- reason
- downstream impact
- reviewer candidate

reviewer candidateには `reviewer-candidate-is-not-review-authority` を明示し、正式Authorityへ昇格させない。

### fixture pressure

最初のfixtureには次を同時に含める。

- current-context user fact
- user decision / constraintのworking-premise test
- userが取り込んだAI-origin contentのprovenance保持test
- AI unreviewed assertion
- human-reviewed assertion
- narrow-scope conditional review
- stale reviewed assertion
- superseded assertion
- reviewed Conflict pair
- rejected AI assertion
- critical / high / low downstream dependency

同じfixtureをSUI #3184 / SEI #660へ渡せるtransport candidateとして設計し、TEI固有UIやruntime objectをfixtureの正本にしない。

### authority stop line

このReferenceのassessment / Health / review requestは、いずれも次を与えない。

```text
epistemic assessment
  != objective Truth
  != Canonical Acceptance
  != write authority
  != Runtime authority

reviewer candidate
  != review authority
```

### Evidence状態

現時点ではsource candidateとfocused testを追加した段階であり、exact-head full `tools/repository-health-check.sh` PASSは未取得である。

したがって、この追記だけで完了条件のcheckbox、Product Evidence tier、ADR以外のproduction authorityを更新しない。trusted full checkoutでpre/post clean stateを確認したfull Repository Health PASSを取得してから、Reference成立を判断する。

## 完了条件

- [ ] user factual assertionとAI unreviewed assertionを区別できる
- [ ] fact / observation / requirement / decision / constraint / hypothesisを区別できる
- [ ] content provenanceとingestion actorを分離できる
- [ ] observed / declared / inferred contextを区別できる
- [ ] 複数review eventから現在の確認状態を導出できる
- [ ] annotation targetを編集後に誤って別箇所へ付け替えない
- [ ] provenance不明のlegacy資料をreview済みへ推定しない
- [ ] metadata欠落export / reimportでconfirmation historyを自動復元しない
- [ ] epistemic coverage不足を観測できる
- [ ] AI生成主張が保存だけで確認済みへ昇格しない
- [ ] content provenance / ingestion actor / statement kind / confirmation history / lifecycle / scope / time / Evidenceを分離できる
- [ ] explicit contextとinferred contextを区別できる
- [ ] narrow scopeのfactをwide scopeへ黙って一般化しない
- [ ] semantic edit後に確認の有効性を再評価できる
- [ ] AI summaryへ確認状態を無条件継承しない
- [ ] contradictionを保持できる
- [ ] retrievalへrelevanceとcontext compatibilityを別signalとして渡せる
- [ ] Project Knowledge Healthがcritical dependencyを考慮できる
- [ ] stakeholderへの確認依頼へimpact / Evidence / scopeを渡せる
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
