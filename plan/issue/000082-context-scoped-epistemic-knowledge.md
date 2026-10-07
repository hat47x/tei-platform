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
2. AIが生成した情報は、保存されたことだけでは確認済み情報へ昇格しない。

さらに、どちらも別のproject、version、environmentへ無条件に一般化しません。

## Product value

生成AIを継続的に設計へ使うと、誤ったAI出力が文書へ残り、その文書を別のAIが事実として読む連鎖が起こり得ます。

本Issueでは、この連鎖をPrompt上の注意書きではなく、保存、検索、review、context assemblyの契約で止めます。

利用者側では、大量出力へ細かなmetadataを手入力させません。AI出力は自動的に未確認として扱い、利用者は確認できた範囲だけを高速に処理します。

## 設計上の境界

### Truth Engineにはしない

TEIは世界の客観的真偽を判定するAuthorityになりません。

扱うのは、誰が、どの文脈で、いつ、どの状態としてそのassertionを利用できるとしたかです。

### 情報の出所、取り込み主体、働き、確認状態を分ける

ユーザー由来であることは、客観的に検証済みであることを意味しません。一方、ユーザーが現在の作業で直接与えた事実、観測、要件、決定、制約は、その適用範囲では通常の作業前提として利用できます。

AI由来であることも誤りを意味しません。ただし、人間による確認等のEvidenceがなければ確認済みへ昇格しません。また、ユーザーがAI文章を貼り付けても、内容の出所はAIのまま保持します。

### 確認状態と適用範囲を分ける

確認済みの情報も、確認が成立した適用範囲の外では自動的に利用しません。

たとえば「project Aのproductionで確認済み」であることと、「project Bやexternal APIでも利用可能」であることは別です。

### 情報の働きと確認状態を分ける

事実でない情報が、すべて仮説になるわけではありません。設計上の要件、決定、制約は、世界についての事実とは異なりますが、後続作業の前提になります。

Reference candidateでは、fact / observation / requirement / decision / constraint / hypothesisを区別しています。ユーザーが現在の作業で直接与えた要件や決定は通常の作業前提にできますが、組織上の正式AuthorityやCanonical Acceptanceとは分けます。

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
- content provenance
- ingestion actor
- statement kind
- append-only confirmation history
- observed / declared / inferred context
- valid / checked time
- Source / Evidence
- supersession / contradiction relation
- downstream dependency criticality

実装前に、既存Source Index、Evidence、Decision Index、Semantic Extensionのどれを再利用できるか確認します。

### E0a: statement kind / provenance contract

E0のsidecar contractを、事実だけでなく設計作業に必要な情報の働きへ広げます。

少なくともfact、observation、requirement、decision、constraint、hypothesisを別に扱います。content provenanceとingestion actorも分けます。ユーザーがAI出力を貼り付けたcaseをnegative fixtureにし、取り込み操作だけでuser factual assertionへ変わらないことを確認します。

### E0b: context provenance

context bindingは、systemが直接観測したobserved、userやauthoritative sourceが明示したdeclared、AIやheuristicが推定したinferredを区別します。inferred contextだけで適用範囲を広げません。

### E0c: target anchoring / review history

paragraph、card、row、selected range等へのannotationは、編集後の再特定を考慮します。positionだけへ依存せず、利用可能なsubject identity、source revision、structure、quoted text等を組み合わせます。

同じ情報への複数回の確認は上書きせず、review eventとして保持します。現在の確認状態は、対象meaning、適用範囲、時点、review eventから導出する構造をpressureします。

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

同じassertionを別project、branch、version、environmentへ持ち出し、適用範囲の不一致を検出します。

inferred contextを利用する場合、その推定結果と根拠を保持し、後から訂正できるようにします。

### E3: Project Knowledge Health

最初のHealthは単純なpercentだけにしません。

少なくとも次のnegative fixtureを持ちます。

- 多数の低重要度unreviewed memo
- 少数のcritical unreviewed dependency
- reviewedだがstaleなcritical dependency
- reviewed同士のConflict
- 適用範囲の不一致したcritical dependency

低重要度memoが多いだけのcaseより、critical dependencyが未確認のcaseを強くalertできることを確認します。

さらに、annotationされた情報だけでHealthを計算しません。重要な文書や判断領域が未分類、出所不明、対象不明であれば、epistemic coverage不足として別に観測します。coverageが不明な領域を「問題なし」と数えません。

### E4: review request handoff

SUI #3184へ渡すreview request projectionを定義します。

少なくとも次を含みます。

- assertion
- current state / 適用範囲
- reason
- downstream impact
- Source / Evidence
- reviewer candidate
- unresolved / conditional state

reviewer candidateをAuthorityにはしません。

### E5: SEI context assembly handoff

SEI #660へ、semantic relevanceとepistemic / context signalsを別々に渡せることを確認します。

TEI側で一つの最終scoreへ固定しません。

### E6: legacy / export-import / access containment boundary

既存資料と外部連携ではmetadataが存在しない、または途中で失われる場合があります。

import側の最小candidateとして、`metadataState=complete / partial / absent` と `origin=unknown` を追加した。confirmedな別assertionとtext / meaning fingerprintが一致しても、metadata-free reimportへreview historyを自動移送しないnegative testを置く。

metadata-preserving transportのReference candidateとして `tei.reference.epistemic-transport/v0` を追加した。assertion / Evidence / review event / dependency / coverage areaはopaqueな `accessRefs` を持てる。transport自身はauthorizationを判断せず、callerが渡すgranted refとの完全一致だけを使う。

権限不足のassertionは除外し、Evidence / review event / relationだけが欠ける場合は残るassertionを `metadataState=partial` にする。hidden assertionが片側だけからConflict / supersessionを指している場合も、可視側をpartialにして関係を忘れたままpremiseへ戻さない。DependencyやCoverage Areaを除外したtransport全体もpartial viewとして保持し、Project Knowledge Healthをproject全体の完全評価へ格上げしない。

transport adapterを実storage / connectorへ接続すること、text-only exportでloss markerを運ぶこと、accessRefsを既存authorization providerへ接続すること、transport署名は未実証として残す。

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

- AI出力を自動的に未確認として扱う
- userが直接与えたfact / observation / requirement / decision / constraintへ通常の再承認操作を要求しない
- selection + 1操作
- bulk review
- 条件付き確認 / 適用範囲の精密化
- 意味変更後の再確認表示
- ambiguous / detached targetの再接続UI
- epistemic coverage gapの表示
- Knowledge Health projection
- stakeholder review queue

SUI固有のTruth modelは作りません。

## SEI #660へのhandoff

SEIでは次を実証します。

- context-compatible retrieval
- status-aware context assembly
- Conflict / stale / supersededの保持
- 狭い適用範囲から広い適用範囲へのgeneralization抑制
- ambiguous / detached targetをpremiseへ投入しない
- consumer freshness requirementをquery / task文脈から適用する
- review escalation候補
- RecommendationとAuthorityの分離

similarityやmodel confidenceをTruthへ昇格させません。

## 既存設計との接続

### #267 / ADR-0022

AI ProposalとCanonical Acceptanceを分離する既存境界は維持します。本Issueで確認済みとなった情報が存在しても、それだけでCanonical changeをAcceptedにはしません。

### ISSUE-000047

Knowledge qualification / freshnessを再利用します。epistemic stateを既存qualificationへ無理に押し込まず、applicability、confirmation、lifecycleを分けたまま接続します。

### ISSUE-000077

新しい意味をCore enumへ追加せず、Semantic Extensionとしてpressureできるか確認します。

### ISSUE-000080 / #951

review、health、context evaluation等をPlugin-defined Capabilityとして提供できるか確認します。

## 2026-10-06 First Reference candidate — E0〜E4

このcandidateは、fact中心の初期版から、statement kind、content provenance / ingestion actor、observed / declared / inferred context、append-only review history、target anchoring、epistemic coverage、metadata-free legacy reimport、consumer freshnessまでpressureを広げました。未実証なのは、transport candidateを実storage / connectorへ接続するadapter、text-only transportのloss marker、既存authorization providerとの接続、production向けtarget resolver / coverage生成、consumer側での実retrieval統合です。

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

適用条件を構造化できる場合はassertion contextへ置く。review eventに自由文 `conditions` が残る場合、このReferenceは条件成立を推測せず `REVIEW_CONDITIONS_UNRESOLVED` として `review-required` にする。

このReference自身はarbitrary proseのsemantic diffを発明しない。meaning fingerprintは対象surface / transformation contractから受け取る入力であり、一致そのものをsemantic proofへ昇格させない。format差とsemantic差を判別できない場合に確認の有効性を推測しない。

### E1b supersession / conflict candidate

旧assertionのconfirmationを変更してsupersessionを表現せず、後続assertionの `supersedes` relationから `lifecycleState=superseded` を派生する。Projectionには `supersededBy` を残し、relation provenanceへ戻れるようにする。

supersession cycleは置換関係として解釈できないためvalidationで拒否する。

Conflictもrelationとして保持する。入力が片側だけでもProjectionでは対称なConflictとして扱い、current contextで両方が有効候補になる場合は一方を自動採用せず `review-required` とする。

### E2 context compatibility / target anchoring candidate

assertion scopeとcurrent contextをdimensionごとに照合する。

- 全必要dimension一致: `compatible`
- 値が異なる: `incompatible`
- 必要dimensionがcurrent contextに無い: `unknown`

`unknown` を「たぶん同じ」としてpremiseへ昇格させない。observed / declared / inferred contextを区別し、inferred contextを使ったassessmentにはsignalを残す。


target annotationについては、任意targetへ元resourceのrevision fingerprint、structural path、text quoteのexact / prefix / suffixを持つanchorを付けられるcandidateを追加した。current context側にはresourceの現在revisionとsegment群を渡す。

- 同revisionのselectorが一意に一致: `exact`
- revision / path変更後もquote + 周辺文脈で一意に一致: `reanchored`
- 複数候補: `ambiguous`
- 候補なし: `detached`

`ambiguous / detached` は別textへ自動転送せず `review-required` にする。anchor解決はsemantic identityの証明ではなく、その限界をProjectionへ明示する。

### E2b: consumer freshness requirement

情報側のvalidUntilとは別に、dependency consumerが必要とする最大経過時間を `maxAgeSeconds` として指定するcandidateを追加した。基準時刻にはcurrent meaningのhuman confirm eventまたは `checkedAt` のうち最新のものを使う。

consumer requirementを満たさなくてもassertion本体のconfirmation / base use stateは書き換えない。同じ情報を別用途で利用できる余地を残したまま、high / critical consumerについてHealth issue / review requestへ反映する。

### E3 Project Knowledge Health / coverage candidate

Healthはreview率を主判定にせず、`dependencies` のcriticalityと現在のuse stateを組み合わせる。

critical consumerがunreviewed / stale / Conflict / context不適合なassertionへ依存する場合は `critical` とする。低重要度unreviewed memoが多数存在しても、critical dependencyを件数比で希釈しないnegative testを置いた。

さらに、project側が状態を把握すべき情報領域をcoverage areaとして入力できるcandidateを追加した。areaへ対応するassertion mappingが無い、またはmapping先targetを再特定できない場合はcoverage gapとする。critical / high gapはHealthへ反映する。

coverage mappingは入力であり、意味的完全性の証明ではない。自動抽出や「必要情報がすべて揃った」という判定は本ReferenceのAuthority外とする。

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
- human-confirmed assertion
- 構造化された適用範囲を持つconditional confirmation
- staleなconfirmed assertion
- superseded assertion
- confirmed Conflict pair
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

- [ ] userが直接与えた作業前提とAI未確認情報を区別できる
- [ ] fact / observation / requirement / decision / constraint / hypothesisを区別できる
- [ ] content provenanceとingestion actorを分離できる
- [ ] observed / declared / inferred contextを区別できる
- [ ] 複数review eventから現在の確認状態を導出できる
- [ ] target bindingをexact / reanchored / ambiguous / detachedとして観測できる
- [ ] coverage mappingを意味的完全性の証明へ昇格させない
- [ ] annotation targetを編集後に誤って別箇所へ付け替えない（Reference candidate追加済み。full qualification未完了）
- [ ] provenance不明のlegacy資料をreview済みへ推定しない（import-side Reference candidate追加済み。full qualification未完了）
- [ ] metadata欠落export / reimportでconfirmation historyを自動復元しない（reimport negative candidate追加済み）
- [ ] metadata-preserving transportでEvidence / review event / relationのaccess containmentを保持できる（Reference candidate追加済み。full qualification未完了）
- [ ] partial transportをproject全体の完全なHealth viewへ格上げしない（Reference candidate追加済み。full qualification未完了）
- [ ] epistemic coverage不足を観測できる（Reference candidate追加済み。full qualification未完了）
- [ ] AI生成主張が保存だけで確認済みへ昇格しない
- [ ] content provenance / ingestion actor / statement kind / confirmation history / lifecycle / scope / time / Evidenceを分離できる
- [ ] explicit contextとinferred contextを区別できる
- [ ] 狭い適用範囲の情報を広い適用範囲へ黙って一般化しない
- [ ] semantic edit後に確認の有効性を再評価できる
- [ ] AI summaryへ確認状態を無条件継承しない
- [ ] contradictionを保持できる
- [ ] one-sided conflictsWithを対称なConflictとして投影できる
- [ ] supersession cycleを拒否できる
- [ ] retrievalへrelevanceとcontext compatibilityを別signalとして渡せる
- [ ] Project Knowledge Healthがcritical dependencyを考慮できる
- [ ] consumer側のfreshness requirementをassertion本体の確認状態と分離して評価できる
- [ ] stakeholderへの確認依頼へimpact / Evidence / 適用範囲を渡せる
- [ ] 条件付き確認で適用範囲の精密化を保持できる
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
