# Context-scoped Epistemic Knowledge Reference

ADR-0030 / ISSUE-000082の最初のTEI Referenceです。

保存された情報を単一の「正しい / 未確認」へ畳まず、現在の作業文脈で利用できるかを、情報の出所、取り込み主体、statement kind、確認履歴、文脈、鮮度、ライフサイクル、Conflictに分けて評価します。

## 実行

```bash
go run ./tools/reference-epistemic-knowledge \
  -corpus tools/reference-epistemic-knowledge/testdata/epistemic-fixture-v0.json \
  -context tools/reference-epistemic-knowledge/testdata/context-current.json
```

出力は、意味契約とJSON表現を別IDで持ちます。

- semantic contract: `tei.epistemic-projection/v0`
- representation schema: `tei.reference.epistemic-assessment/v0`

consumerは両方を確認します。同じsemantic contractでも未知のrepresentation schemaを自動受理せず、同じschemaでもcontract IDが異なる場合は互換と推定しません。

## 分離する情報

一つのconfidenceやtruth scoreにはまとめません。

```text
content origin
  user / ai / system / imported / external

ingested by
  user / ai / system / connector / import

statement kind
  fact / observation / requirement / decision / constraint
  hypothesis / question / instruction / preference / other

review event
  confirm / reject / withdraw

derived confirmation state
  unreviewed / confirmed / rejected

review binding
  none / current / stale

context compatibility
  compatible / incompatible / unknown

freshness
  current / stale / not-yet-valid / unknown

lifecycle
  active / superseded

conflict
  none / inactive / present
```

現在のcontext assemblyでの利用状態だけを `premise / candidate-only / review-required / blocked` として派生します。この利用状態は客観的TruthやCanonical Authorityを表しません。

## ユーザー入力とAI生成情報

user originかつuserが直接取り込んだfact / observation / requirement / decision / constraintは、適用範囲が現在の文脈と一致すれば作業前提として利用できます。

この場合でも `confirmationState` は `unreviewed` のままです。「ユーザーが現在の作業で直接与えたため前提として使えること」と「人間の確認履歴がありconfirmedであること」を同じ状態へしません。また、組織上の正式承認やCanonical Acceptanceも意味しません。

入力者と内容の出所も分けます。ユーザーがAI出力を貼り付けた場合、content originは `ai`、ingested byは `user` のままです。貼り付けだけではuser assertionやconfirmedへ変わりません。

AI-origin情報にreview eventがなければ `unreviewed` であり、working-premise statementでも `candidate-only` です。

## 確認履歴

現在の確認状態をassertion上の単一fieldとして保存しません。`reviewEvents` に履歴を追記し、現在のmeaningに結び付く最新eventから確認状態を導出します。

v0のeventは次の3種類です。

- `confirm`: 現在のmeaningを人が確認した
- `reject`: 現在のmeaningを人が退けた
- `withdraw`: 以前の確認を撤回した

各eventはpositiveな `sequence`、一意なID、reviewer、`method=human-review`、対象meaning fingerprint、確認時点のstatement kind、context、target identity、valid from / valid until、時刻、任意のEvidenceとconditionsを持ちます。sequenceは一意で、時刻がsequenceに反して過去へ戻る履歴は拒否します。

review bindingは本文meaningだけでは成立しません。current assertionのstatement kind、context、target identity、validity windowのいずれかがevent snapshotと異なれば、そのeventは現在値のconfirmationへ使わずstale historyとして扱います。これにより、同じ本文を別scopeへ広げたり、factからhypothesisへ変更したり、別targetへ付け替えたりしただけで旧確認を継承しません。

適用条件を機械判定できる場合は、可能な限りassertionの構造化contextへ表します。review eventに自由文conditionsが残っている場合、このReferenceは条件を満たしたと推測せず、`REVIEW_CONDITIONS_UNRESOLVED`として`review-required`へ戻します。

たとえば、

```text
confirm
  -> reject
  -> confirm
```

という履歴は3eventを残したまま、現在値だけを `confirmed` として導出します。過去のrejectを上書きして消しません。

`withdraw` はassertionを削除したりrejectedへ変えたりする操作ではありません。確認だけを外します。外部由来の情報なら再び `candidate-only` になり、direct user working premiseなら、確認済みとは表示せず現在文脈の作業前提として利用できます。

## as-of時点のreview history

Projectionの `ContextQuery.asOf` より後に発生したreview eventは、現在の確認状態だけでなく、event件数、末尾sequence、reasonからも見えないようにします。過去時点のProjectionを、未来の確認履歴で書き換えません。

Projectionは `reviewEventCount` と `reviewLogSequence` をas-of時点で見えている履歴だけから返します。両者は同義ではありません。sequenceに欠番があっても、件数から次sequenceを推測しません。

Authority limitとして `review-history-is-bounded-by-context-as-of` を返します。

## 意味変更とreview binding

review eventはevent時点の `meaningFingerprint` を持ちます。

現在assertionのmeaning fingerprintが変わり、現在meaningへ結び付くreview eventがなくなった場合、過去のeventは履歴として残しますが、`reviewBinding=stale` として再確認を要求します。旧確認を現在値へ継承しません。

このReference自身は自然言語のsemantic diffを判定しません。meaning fingerprintは対象を所有するsurface / transformation contractが供給する入力です。fingerprint一致そのものをsemantic proofにはせず、Projectionにも `meaning-fingerprint-is-input-not-semantic-proof` を残します。

## context generalizationをしない

情報の適用範囲に必要なdimensionが現在contextに無い場合は `unknown` とし、前提へ自動昇格させません。値が異なる場合は `incompatible` です。

```text
usable in project A / production
  !=
usable in project B
  !=
usable when environment is unknown
```

context provenanceも `observed / declared / inferred` に分けます。repositoryやbranchなどをsystemが直接観測した場合はobserved、ユーザーや資料が明示した場合はdeclared、AIやheuristicの推定はinferredです。inferred contextを使ったassessmentは、その事実を出力へ残します。

## Supersession / Conflict

supersessionは確認状態へ混ぜません。後続assertionの `supersedes` relationから、旧assertionの `lifecycleState=superseded` を派生し、`supersededBy` にrelation provenanceを残します。

したがって、旧assertionに有効なconfirm eventが残っていても、後続relationを無視してpremiseへ戻しません。

Conflict relationも消しません。片側だけに `conflictsWith` が記録されていても、Projectionでは対称な関係として扱います。現在contextで両方が利用候補になる場合、AIが一方を選んで解消したことにはせず、双方を `review-required` として残します。

supersessionは方向を持つため対称化しません。ただしcycleは時間的・論理的な置換関係として解釈できないため、corpus validationで拒否します。

## annotation対象の再特定

paragraph、card、row、選択範囲へのannotationは、文字位置だけに依存しません。Referenceではtargetへ、元resourceのrevision fingerprint、構造path、TextQuote型のexact / prefix / suffixを持つanchorを付けられます。

現在resourceはcontext側のtarget surfaceとして渡します。元revisionと同じselectorが一意に見つかれば `exact`、revisionや構造位置が変わっても引用と前後文脈から一意に見つかれば `reanchored` とします。候補が複数なら `ambiguous`、見つからなければ `detached` です。

`ambiguous` / `detached` を別の文章へ自動的に付け替えず、利用状態を `review-required` へ戻します。考え方はW3C Web AnnotationのSelector / State、特にTextQuoteSelectorのexact / prefix / suffixを参考にしています。ただしW3C schema自体をTEIの正本にはしません。

anchor解決はsemantic identityの証明ではありません。Projectionには `target-anchor-resolution-does-not-prove-semantic-identity` を残します。

## Epistemic coverage

Healthはannotation済みassertionだけを数えません。Referenceでは、project側が「状態を把握しておくべき情報領域」をcoverage areaとして宣言し、その領域に対応するassertionをmappingできます。

mappingが空、またはmapping先のtargetがすべてambiguous / detachedならcoverage gapです。critical / highのgapはProject Knowledge Healthをそれぞれcritical / attentionへ引き上げます。

coverage mappingそのものは「必要情報が意味的に完全に揃った」ことを証明しません。そのため `coverage-mapping-is-input-not-semantic-completeness-proof` をAuthority limitとして明示します。

## Consumer freshness requirement

情報自体の `validUntil` だけでは、用途ごとに必要な新しさを表せません。Referenceではdependencyに任意の `maxAgeSeconds` を持たせ、consumerが必要とする最大経過時間を指定できます。

鮮度の基準時刻には、現在meaningに対するhuman confirm eventまたは `checkedAt` のうち、現在時刻以前で最も新しいものを使います。基準が無ければ `unknown`、最大経過時間を超えれば `stale` です。

consumer freshnessがstaleでも、assertion自体のconfirmationや通常のuse stateを書き換えません。同じ情報が低stakesな用途では利用可能でも、freshnessを要求するhigh / critical consumerではHealth issueやreview requestになる、という分離を保ちます。

## Project Knowledge Health

Healthは「確認済み件数 / 全件数」のpercentを主判定にしません。

`dependencies` は、どのassertionをどのconsumerがどのcriticalityで利用しているかを表します。criticalなconsumerが未確認、stale、Conflict、適用範囲不一致の情報へ依存する場合は `critical` を返します。

多数の低重要度memoが未確認でも、それだけでcritical dependencyを希釈しません。

## Human review mutation candidate

SUI等のhuman review UIからappend-only review historyへ戻すため、Referenceには副作用のないreview mutation candidateを追加しています。

command identity:

- semantic contract: `tei.epistemic-review-command/v0`
- representation schema: `tei.reference.epistemic-review-command/v0`

mutation result:

- semantic contract: `tei.epistemic-review-mutation/v0`
- representation schema: `tei.reference.epistemic-review-mutation/v0`

confirm / reject / withdraw commandは少なくとも次のpreconditionを持ちます。

- assertion ID
- expected meaning fingerprint
- expected review subject fingerprint
- expected target binding
- expected review log sequence
- reviewer / event ID / occurredAt

meaning、review subject、target binding、review sequenceのいずれかがUI表示後に変わっていれば古いcommandを拒否します。

`reviewSubjectFingerprint` は、review event、reviewer candidate、access referenceを除いた正規化assertionから作るopaqueな同時実行receiptです。target、statement kind、scope、temporal validity、Evidence、Conflict / supersession relation等の変更を検出します。これはTruthやsemantic identityを証明するfingerprintではありません。

これにより、別surfaceで意味編集、scope変更、classification変更、Evidence変更、relation変更、target移動、review追加が行われた後に、古い画面から確認を重ねるlost updateを避けます。

confirm / rejectではmetadataがcompleteであり、targetがambiguous / detachedではなく、現在contextがcompatibleであることも要求します。withdrawは既存確認を外す操作なので、contentの再確認とは分けます。

meaning fingerprintやtarget binding preconditionはsemantic identityの証明ではありません。Reviewer IDも組織Authorityを与えません。

CLIでは次のside-effect-free modeを使えます。

```bash
go run ./tools/reference-epistemic-knowledge \
  -corpus corpus.json \
  -context context.json \
  -review-command review-command.json
```

元corpusファイルを直接書き換えず、updated corpus、appended event、再評価Projectionを含むmutation resultを返します。storageへの永続化とmulti-surface commitはproduction adapterの責務です。

mutationへ渡すContextQueryは、UI表示時の古いProjection contextをそのまま再利用しません。review eventの `occurredAt` を包含するcurrent contextをcallerが再取得し、`context.asOf >= occurredAt` を満たす必要があります。Referenceはevent時刻がcontext as-ofより未来なら拒否します。

このfresh mutation contextでscope compatibilityとtarget bindingを再評価し、UI intentが持つexpected target binding等と照合します。Projection表示後にenvironment、target surface等が変わっていれば、古い画面の確認をそのまま適用しないためです。

## review request handoff

reviewが必要なassertion、またはhigh / critical consumerがpremiseとして利用できないassertionはreview requestへ投影できます。

requestは少なくとも次を保持します。

- assertion identity / target
- derived confirmation state / review binding
- context
- Evidence
- reason
- downstream impact
- reviewer candidate

reviewer candidateはAuthorityではありません。Projectionには `reviewer-candidate-is-not-review-authority` を明示します。

## fixture

`testdata/epistemic-fixture-v0.json` は、SUI / SEIへ同じ意味のfixtureを渡すためのtransport candidateです。

含むcaseは、current-context user fact、AI unreviewed assertion、human-confirmed assertion、構造化scopeによる限定確認、stale情報、relation-derived supersession、reviewed Conflict、reject event、critical / high / low downstream dependencyに加え、revisionをまたいで再特定するdocument-range anchorとcoverage areaです。自由文review condition、ambiguous / detached target、coverage gapをfail closedにするnegative testも持ちます。

TEI固有UIやruntime objectをfixtureの正本にはしません。

## Authority limits

このReferenceのassessment / health / review requestは、次を与えません。

- objective truth
- Canonical Acceptance
- write authority
- Runtime authority
- reviewer authority

Epistemic sidecarを削除しても既存TEI Canonical Worldが変わらないことが設計前提です。このtoolはCanonical loaderへ接続せず、入力sidecarからread-only Projectionを生成します。

## Legacy / metadata loss

既存文書やtext-only exportから再importした内容には、元のprovenanceや確認履歴が付いていない場合があります。Referenceでは `metadataState=absent / partial / complete` を区別し、出所を復元できない内容には `origin=unknown` を使えます。

同じ文章や同じmeaning fingerprintを持つ別assertionにconfirmedなreview historyが存在しても、それをmetadata-free reimportへ自動継承しません。review eventはassertion identityとそのmeaning bindingに属し、fingerprint一致だけでは移送しません。

fixtureでは、confirmedな `human-reviewed` と同じtext / meaning fingerprintを持つ `legacy-reimport-copy` を用意します。後者は `origin=unknown / ingestedBy=import / metadataState=absent` のまま `unreviewed / candidate-only` です。

Projectionには `content-fingerprint-does-not-restore-review-history` をAuthority limitとして残します。

metadata-preserving exporter / importerそのものは、このReferenceではまだ実装しません。保持できるtransportではsidecarを一緒に運び、text-only transportではmetadata lossを明示する契約を後続pressureとします。

## Metadata-preserving transportとaccess containment

Referenceには `tei.reference.epistemic-transport/v0` のtransport candidateを追加しています。

epistemic transport自身はRBACや組織Authorityを判定しません。既存のauthorization境界が与えるopaqueな `accessRefs` と、export時にcallerが利用可能なref集合だけを入力として扱います。refの意味を推測せず、完全一致だけでtransport対象を絞ります。

`accessRefs` は少なくともassertion、Evidence、review event、dependency、coverage areaへ付与できます。権限不足の場合は次のように扱います。

- assertion全体が不可視ならtransportから除外する
- Evidenceやreview eventだけが不可視なら、そのassertionを `metadataState=partial` にする
- 不可視assertionへのrelationは可視側からも除去し、可視assertionをpartialにする
- 不可視dependency / coverage areaは除外する
- 何かを除外したtransport全体を `metadataState=partial` とする

これにより、hidden reject / Conflict / supersession等を落とした後に、残ったconfirmだけから「完全な確認済み情報」を再構成しません。partial metadataはReference評価でfail closedし、通常のpremiseへそのまま昇格させません。

また、partial transportのProject Knowledge Healthはproject全体のHealthではありません。Projectionは `viewMetadataState` を返し、`partial-transport-health-is-not-project-wide-health` をAuthority limitとして残します。

redaction summaryは件数だけを返し、不可視assertion ID等をtransport metadataから漏らしません。

このcandidateが示すのはaccess containmentであり、authorization decisionそのものではありません。`accessRefs` の発行・解釈・組織Roleとの対応は既存TEI authorization境界の責務です。

## Semantic contractとrepresentation schemaの分離

SUI / SEIの第二consumerを作った結果、Reference内部のJSON schema IDへ直接依存すると、TEI側の表現変更がそのままconsumer破壊になることが分かりました。

そのためProjectionは次を分離します。

```text
contract
  tei.epistemic-projection/v0

schema
  tei.reference.epistemic-assessment/v0
```

`contract` はconsumerへ渡す意味境界、`schema` はこのReferenceの具体的JSON表現です。

互換性をschema名やfield presenceから推測しません。

- contractが未知: fail closed
- contractは既知だがschemaが未知: decoder未対応としてfail closed
- schemaが同じでもcontractが異なる: fail closed

将来、同じsemantic contractに対して別representation profileを追加する場合、consumerは対応decoderを明示的に追加します。意味変更と表現変更を同じversion軸へ押し込めません。

## v0をproduction modelへ固定しない

このReferenceは、confirmation historyを単一review stateから分離するpressureまで進めています。ただしevent kind、sequence方式、context scopeの表現、永続化方式をproduction contractとして固定したわけではありません。

特に、review eventを複数surfaceから永続化・mergeする仕組み、reviewerの正式Authority、production向けtarget resolver、coverage area / mappingの生成、legacy / export-import境界は別pressureが必要です。

## 今回まだ示さないこと

- SUIでのselection + 1操作やbulk review
- SEIでのrelevanceとcontext signalを使った実retrieval
- arbitrary proseからのsemantic meaning fingerprint生成
- reviewer / stakeholderの正式な組織Authority解決
- Semantic Extension / Pluginとしてのproduction registration
- review eventのdurable persistence / multi-surface commit・merge
- arbitrary documentからtarget surface / selector候補を生成するproduction resolver
- coverage area / mappingを自動抽出して意味的完全性を証明する仕組み
- transport candidateを実Provider / storage / connectorへ接続するadapter
- text-only transportでの明示的loss marker
- accessRefsを既存authorization providerへ接続するproduction contract
- Cross-repository fixtureの正式versioning

## 検証

focused test:

```bash
go test ./tools/reference-epistemic-knowledge -count=1
```

Repository integration authorityにはfocused testだけでなく、exact headで `bash tools/repository-health-check.sh` のfull PASSとclean pre/post stateが必要です。GitHub Actionsへは依存しません。
