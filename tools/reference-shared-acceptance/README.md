# reference-shared-acceptance

`ISSUE-000027`（AI Proposal / Acceptance）と`ISSUE-000046`（Studio Shared Acceptance）が共有する、**Shared Acceptance Boundaryの決定論的なReference実装**です。

file edit・CLI・Studio・AIのどの入口から来た変更candidateも、同じ検査を同じ順序で通り、**acceptだけがauthoritative sourceを書き換えます**。AI専用・Studio専用の受入経路はありません。

## 位置づけ

- Reference専用のtoolです。Core primitive、platform共通のProposal schema、Patch DSLではありません。candidateの形式（`tei.reference.change-candidate/v0`）はこのReferenceの実測から決めた運搬形式で、固定を主張しません。
- 差分・依存影響の解析はread-onlyな[`tools/internal/changeanalysis`](../internal/changeanalysis)（`ISSUE-000039`の最初の解析境界）が担当し、このtoolはそれを呼びます。受理・source反映（`ISSUE-000027`）と解析を混ぜず、解析側はsourceを読み書きしません。
- AIによる候補生成、Studio UI、read-only探索そのものは含みません。これらはcandidateの生成側です。一方、`ISSUE-000047`で`applicable`になったqualification結果は、exactなsubject・source・fingerprint・requirementsをEvidenceとしてcandidateへ付与できます。requirementsが明示的なdependencyを含む場合は、そのdependencyのauthoritative sourceとfingerprintも必要な範囲だけ保持します。このtoolはPreview時に同じrequirementsで再qualificationし、現在も適用可能かを確認します。
- Gate B / Gate CのBaseline Evidenceは、その判定時のrevisionに固定されたhistorical Evidenceです。current `reference/workspace`を永久凍結する意味ではありませんが、baseline名のworkspaceへの誤書込みを避けるため、`accept`は`-allow-reference-baseline`が無い限り拒否します。通常はscratch copyで検証し、post-Gate-CのintentionalなReference更新だけをexplicit decision + flagで行います。Second Applicationの別生成AIレビューは2026-09-21に完了し、`review-clear-with-conditions`となりました。実際の`reference/workspace`へのacceptには、なおexact Proposalに対するexplicit human Acceptance decisionが必要です（`ISSUE-000040`）。

## 受入境界

```text
decode → normalize → base-revision → resolve → apply(overlay)
       → reload → validate → architecture-policy → semantic-diff
       → impact → acceptance-policy
```

| stage | 確認すること |
| --- | --- |
| decode | 未知fieldを許さず厳密に読む |
| normalize | 正規形へ揃え、意味を持つ部分（base revisionとdelta）だけの`candidateFingerprint`を作る。Origin / Intent / Evidenceは含めない。qualification付きEvidenceはrequirementsとrequired dependency Evidenceの対応を検証し、Evidence集合をdedup・sortして入口差を除く。正規化済みEvidenceには別の`evidenceFingerprint`を付け、意味identityと根拠identityを混ぜない |
| base-revision | candidateの`baseModelRevision`が現在のsourceと一致する。staleなら黙って適用せず、reject / 再読込して再生成 / 明示rebase / 手動conflict解消へ分岐する |
| resolve | 対象namespace・Attribute・Relationが存在し、追加するidが未使用で、Evidenceがcurrent Source Indexへ解決できることを確認する。qualification付きEvidenceはsubject kind・source・fingerprintを照合し、同じrequirementsで再qualificationする。明示required dependencyがある場合は、そのauthoritative sourceとfingerprintもcurrent qualificationと一致する必要がある。placementは意味を所有する層に限る |
| apply | 変更を**overlay（複製）**へ適用する。元のsourceはまだ変えない |
| reload | overlayを共有loaderで読み直す |
| validate | 共有Validationにerrorが一件も無い |
| architecture-policy | 変更されたfileがdeltaに対応するfileだけで、semantic層の中に閉じている |
| semantic-diff | Canonical Worldの実際の差分が、candidateの宣言したdeltaと一致し、semantic層以外のWorld collectionが変わっていない |
| impact | 追加されたsubjectから、exact identityに基づく依存（Attribute → Data → Relation → Application Pattern、namespace利用Application）と、判断できなかった事項を示す |
| acceptance-policy | 変更subject数の上限（`maxSubjectChanges`）を超えない。whole-model再生成を標準経路にしない |

後続stageは、前のstageが失敗すると`skipped`になります。

## 使い方

```bash
go run ./tools/reference-shared-acceptance attach-qualification -candidate FILE -qualification FILE [-out FILE]
go run ./tools/reference-shared-acceptance propose-qualified-reference-pattern -workspace DIR -candidate FILE -qualification FILE -reference-reuse FILE [-out FILE]
go run ./tools/reference-shared-acceptance revision -workspace DIR
go run ./tools/reference-shared-acceptance preview  -workspace DIR -candidate FILE [-out FILE]
go run ./tools/reference-shared-acceptance review   -workspace DIR -candidate FILE [-format text|json|html] [-out FILE]
go run ./tools/reference-shared-acceptance accept   -workspace DIR -candidate FILE -expected-review-fingerprint SHA256 [-derived-out FILE] [-out FILE]
go run ./tools/reference-shared-acceptance reject   -workspace DIR -candidate FILE [-reason TEXT] [-out FILE] [-allow-reference-baseline]
go run ./tools/reference-shared-acceptance decisions -workspace DIR [-out FILE]
```

- `attach-qualification`: `knowledge-qualify`が返した`applicable`な結果だけをcandidateのEvidenceへ付与します。candidateのdeltaやfingerprintは変えません。`incompatible`と`unknown`は付与できません。requirementsは添付時に正規化・検証し、`expectedKind`とEvidenceの`subjectKind`が一致しない場合もPreview前に拒否します。required dependencyごとのauthoritative Evidenceも必須で、requirementsに無いdependency Evidenceは拒否します。同値なEvidenceは正規化後にdedup・sortされるため、再添付は冪等です。
- `propose-qualified-reference-pattern`: `application-pattern`のapplicable qualificationから、既存candidateへ`add-reference-pattern`を1件追加します。Pattern ID、所有Application、Relation、datasource、placementは`reference-reuse`側で明示します。Contract / Query / input mappingはrequestから受け取らず、current Canonicalのqualified origin Patternから取得します。origin PatternのContract Queryがqualificationの`requiredDependencies`に明示されていない場合や、qualificationのSource / dependency Evidenceがcurrent workspaceと一致しない場合はProposalを生成しません。生成したcandidateにはqualification Evidenceをそのまま引き継ぎます。
- `revision`: workspaceの現在のmodel revision（管理対象source全fileの相対pathと内容fingerprintから導出）を出力します。candidateの`baseModelRevision`にはこれを使います。
- `preview`: sourceを一切書き換えずに、各stageの結果・Semantic Diff・impact・unresolvedを出力します。受理可能でなければ終了コード1です。
- `review`: `preview`と同じread-only評価を使い、承認時に確認する内容をA1〜A8の順で表示します。既定は人が読むtextで、`-format json`を指定すると同じ内容を機械可読で出力し、`-format html`を指定すると同じ内容を1枚のHTMLで出力します。HTMLは外部のscript・stylesheet・font・imageを使わず、JavaScriptなしで読めます。権限差分とnative codeの残存箇所は現時点で専用評価がないため、推測で「変更なし」とせず`未評価`と表示します。
- `accept`: CLIからauthoritative sourceを更新する場合は、先に`review`で得た`reviewFingerprint`を`-expected-review-fingerprint`へ必ず渡します。同じ検査とReview Projectionを最初から作り直し、fingerprintが一致して受入検査も通った場合だけsourceを更新します。更新後はtoolを介さない共有loaderで再読込し、Previewが示した差分どおりか・Validationが通るか・派生物を決定論的に再生成できるかを確認します。一つでも満たさなければ元のsourceへ戻します。`-derived-out`にaccepted sourceから再生成したCanonical World JSONを書き出せます。
- `reject`: authoritative sourceを変更せず、退けたcandidateをcontent-addressed Decisionとして`.tei/decisions/`へ記録します。staleなcandidateも退けられ、基準revisionとcurrent revisionの食い違いをDecisionへ残します。Decision ledgerはmanaged sourceではないためmodel revisionを変えません。Gate B / Gate Cのreference baselineへ記録する場合だけ`-allow-reference-baseline`が必要です。
- `decisions`: accepted / rejected Decisionの記録を、現在のworkspaceと照合して出力します。workspace全体のrevision、Decisionが書いたsourceの有無・現在性、判断時に使ったEvidenceの現在性を別々に確認します。

### 人が読むReview Projection

`review`は、AIの要約を新しい根拠にせず、Shared Acceptance Previewが持つ差分・影響・診断・未解決点・Evidence・source editをA1〜A8へ並べ直します。Semantic Evaluation artifactがEvidenceに添付されている場合は、A3に`pass / fail / unknown`とdiagnostic、A7にartifact / Semantic Set / evaluation fingerprintを表示します。冒頭は`表示可能`・`一部`・`未評価`の件数と、確認が残るsectionの要約です。各sectionは、導出元の`preview.*`または`candidate.*`項目を`sources`として持ち、text出力でも`根拠:`として表示します。`candidateFingerprint`は意味のidentity、`evidenceFingerprint`は正規化済み根拠のidentity、`reviewFingerprint`はA1〜A8を含む提示Projectionそのもののidentityです。A7ではEvidenceと変更予定sourceへ戻れる情報も出します。

`-format html`のページは、このProjectionを人が読むために描画したものであり、承認の記録ではありません。ページを開いたことや`reviewFingerprint`が一致したことは、人が内容を読んで承認したことを示しません。

A5の権限変更とA6のnative code残存箇所は、現在のPreviewだけでは完全に評価できません。そこで`review`は、現行のtyped operationがPermission grant・Resource binding・lease・実行principalや`customGoBindings`をauthorしないことをcandidateから確認し、その範囲だけを`一部`として表示します。一方、既存権限が変更後のApplicationやbindingへどう及ぶか、既存custom implementationのどこが変更subjectに依存するかは未評価のままです。candidateが権限やnative codeを直接書かないことを、実効権限やnative依存が変わらないことへ読み替えません。

CLIから`accept`する場合、`-expected-review-fingerprint`は必須です。Acceptはsourceを書き換える前にcurrent Reviewを作り直し、fingerprintが違えば`REVIEW_FINGERPRINT_MISMATCH`で拒否します。一致した場合は、fingerprint、coverage、A1〜A8を含むReview Projection snapshotを`reviewBinding`としてaccepted Decisionへ残します。Decision Indexはsnapshotからfingerprintを再計算し、coverageとも一致する場合だけ`verified`とします。

この必須化が閉じるのは「Review Projectionを一度も固定せず、CLIからsource mutationへ進む」迂回路です。package内部の`accept` primitiveまでhuman-review tokenを要求する設計にはしていません。内部primitiveはrollbackやconformance等の構成要素testでも使うためです。外部CLIのwrite boundaryだけをreview-binding必須とします。

このProjectionはread-onlyであり、承認そのものでもAcceptance authorityでもありません。`reviewFingerprint`と`reviewBinding`は「どのProjectionへ受理を結び付けたか」を識別するだけで、人が実際に読んで承認したことを証明するtokenではありません。出力の`previewAcceptable`はShared Acceptanceの機械判定だけを表し、Review coverageの不足が解消されたことは意味しません。A1〜A8が承認条件として十分かはISSUE-000071のowner reviewが必要で、Reference toolとしての統合にはexact-headのfull Repository-health PASSが必要です。

### Decisionの記録 — なぜ受け入れた / 退けたか

`accept`はauthoritative sourceを更新したあと、その受理を`.tei/decisions/<内容のSHA-256>.json`へ残します。`reject`もsourceを変更せず、退けたcandidateと理由を同じcontent-addressed Decision ledgerへ残します。数か月後や別sessionのAIが、会話履歴だけに頼らず「なぜこのsourceがあるのか」「なぜこのcandidateは採用されなかったのか」を機械的に辿るためです。

accepted Decisionには、fingerprintへ入らない`intent`と`evidence`、候補の同一性（`candidateFingerprint`）、基準と受理後のrevision、書いたfileとその前後のfingerprint、Semantic / Application差分、受理後Conformance、残ったunresolvedが入ります。rejected Decisionは同じcandidate / Evidence identityと判断理由を持ちますが、`sourceChanged=false`、`sourceEdits=none`です。stale candidateをrejectした場合は、candidateの`baseModelRevision`と判断時のcurrent revision、およびstale時の明示的な選択肢も保存します。

```bash
go run ./tools/reference-shared-acceptance decisions -workspace DIR
```

`decisions`は記録の内容を信用して意味を組み立て直さず、現在のworkspaceと照合した結果だけを付けます。

- `status`: `current`（記録時のrevisionが現在と一致）か`superseded`（workspaceが先へ進んだ）
- `sourceEdits`: `intact`（書いたfileが今も同じ内容）、`diverged`（変わった。`divergedPaths`に列挙）、`none`
- `evidenceStatus`: 全Evidenceの集約状態。`current`（Source Index型だけがありcurrent）、`recorded`（artifact型だけがあり受理時artifactの自己整合を確認）、`current-with-recorded-artifacts`（currentなSource Index型とrecordedなartifact型が併存）、`diverged`、`none`
- `sourceEvidenceStatus`: `current / diverged / none`。current Source Indexとqualificationへ再照合できるEvidenceだけを扱う
- `artifactEvidenceStatus`: `recorded / invalid / none`。受理時artifactの内部fingerprintと外側artifact fingerprintの整合だけを扱い、EvaluatorやConformanceを現在再実行したとは主張しない
- `artifactEvidenceDiagnostics`: artifact自己整合の失敗理由
- `evidenceDiagnostics`: `diverged`の理由。過去のAcceptanceを現在のapplicabilityとして読み替えないための照合結果
- `reviewBindingStatus`: `verified`（保存したReview Projection、fingerprint、coverageが一致）、`recorded`（Projection snapshotを持たない旧形式のbindingは構造だけ整合）、`none`（bindingなし）、`invalid`（fingerprint、coverage、Projectionのいずれかが不整合）
- `reviewCoverageStatus`: `complete`、`incomplete`、`none`、`invalid`。受理時に記録したA1〜A8のcoverageを分類した結果
- `reviewIncomplete`: 受理時に`partial`または`not-evaluated`だったsection。有効なReview bindingでだけ表示
- `reviewDiagnostics`: Review bindingの不整合。A1〜A8の件数、Partial / NotEvaluatedの内訳、section code、Projection schema、Projection再計算fingerprint、coverage一致を検査する
- `decisionActorStatus`: current Referenceでは`unresolved`。candidateの`origin.actor`やCLI実行者をDecision actorへ読み替えない
- `decisionAuthorityStatus`: current Referenceでは`unresolved`。Review bindingやDecision recordの存在をauthoring authorityの証明にしない
- 読み込み時は、まずfile名のSHA-256と実内容が一致するかを確認し、その後にJSON/schema、Source、Evidence、Review bindingを検査します
- 読めない記録があれば、黙って飛ばさずerrorにします

この記録はauthorityではありません。`.tei/`はmanaged sourceではないため、accepted / rejectedのどちらの記録を置いてもmodel revisionは変わらず、全部消してもaccepted meaningは変わりません。同じfile名のまま内容を書き換えたrecordはcontent-address不一致として拒否します。正しいcontent-addressを持つ偽recordでもmeaningは動かず、現在のworkspaceとの照合結果が食い違いとして出ます（`record_test.go`）。content-addressは記録内容の同一性を確認するもので、誰が承認・拒否したかを証明する署名ではありません。current Referenceでは`decisionActorStatus`と`decisionAuthorityStatus`をともに`unresolved`として返し、Proposal originやReview identityから判断者を補完しません。`reviewBindingStatus=verified`も、人がReviewを読んだことの証明ではありません。保存されたProjectionとfingerprintの一致を確認したことだけを示します。`recorded`はProjection snapshotを持たない旧形式のため、元のReview内容までは検証できません。`RQ-027`のaccepted Decision到達性に加え、PV-AI-01ではrejected / stale candidateの追跡可能性として同じledgerを再利用します。

### 確認の記録 — 一度答えた確認を繰り返さない

確認とその答えを`.tei/confirmations/`へ派生物として記録し、同じ確認を理由なく繰り返さないための入口です（ISSUE-000074、RQ-016）。決定記録と同じく、managed sourceではなくmodel revisionに入らず、消しても意味は変わりません。消すと確認がもう一度出るだけです。

```bash
go run ./tools/reference-shared-acceptance confirm-record -workspace DIR -input answer.json
go run ./tools/reference-shared-acceptance confirm-check  -workspace DIR -input query.json
go run ./tools/reference-shared-acceptance confirmations  -workspace DIR
```

`confirm-check`は`ask: true / false`と理由を返します。判定に使うのは確認の種類と対象、答えた時点の前提(subjectを宣言しているfileのfingerprint)が今も同じかだけで、確認の文面と記録の古さは使いません。記録が無い、前提が変わった、記録が読めない、履歴が分岐している、答えが人によるものでない場合は、確認を出す側へ倒します。答えを変えるときは`supersedes`で先頭の記録を指定します。詳細と暫定判断は[ISSUE-000074](../../plan/issue/000074-answered-confirmation-history.md)を参照します。

### knowledge qualificationからcandidateへの接続

`tei knowledge-qualify`で候補を`applicable`と判定した場合、結果にはauthoritativeなsubject kind、Source Index identity、source path、source fingerprintが含まれます。required dependencyを明示した場合は、そのdependencyについてもDefinition / Authoring sourceとfingerprintを`dependencyEvidence`へ含めます。`attach-qualification`は、そのEvidenceと判定時のrequirementsを既存candidateへ付与します。

AI Builderが`tei knowledge-review-handoff`を利用した場合は、そのHandoff JSONを`-qualification`へ直接渡してもよい。adapterはHandoffのschema / purpose / routing scope / subject / Review Plan identity / fingerprintを検証し、`qualified-for-disclosure`かつ内包Qualificationが`applicable`の場合だけ、既存のQualification Evidence経路へ変換する。Handoff自体やDisclosure本文を新しいProposal Evidence schemaとして保存しない。requirementsは同じ共有正規化を通すため、空の`expectedKind`、不正なdependency指定、重複dependency、`expectedKind`と`subjectKind`の不一致をgrounded candidateとして出力する前にfail closedで拒否します。required dependencyのEvidenceが欠ける場合や、requirementsに無いdependency Evidenceが混入した場合も拒否します。

```bash
go run ./cmd/tei knowledge-qualify \
  -workspace DIR \
  -subject-kind application-pattern \
  -subject-id order_entry.customer_reference \
  -qualification-requirements requirements.json \
  > qualification.json

go run ./tools/reference-shared-acceptance attach-qualification \
  -candidate candidate.json \
  -qualification qualification.json \
  > grounded-candidate.json

go run ./tools/reference-shared-acceptance preview \
  -workspace DIR \
  -candidate grounded-candidate.json
```

Builder Review経路では、bare Qualification Resultを別に作り直さず、Handoffを同じ`-qualification`へ渡せる。

```bash
go run ./cmd/tei knowledge-review-handoff \
  -workspace DIR \
  -subject-kind application-pattern \
  -subject-id order_entry.customer_reference \
  -qualification-requirements requirements.json \
  > review-handoff.json

go run ./tools/reference-shared-acceptance attach-qualification \
  -candidate candidate.json \
  -qualification review-handoff.json \
  > grounded-candidate.json

go run ./tools/reference-shared-acceptance preview \
  -workspace DIR \
  -candidate grounded-candidate.json
```

探索結果からdestination-owned reference PatternをProposalへ追加する場合は、PatternのContract Query dependencyまで明示したqualificationが前提です。たとえば`order_entry.customer_reference`を再利用候補にする場合、`master_data.customer/find_by_id`をrequired dependencyとして含めます。そのうえで、destination側で決める情報だけを次のrequestへ分けます。

```json
{
  "schema": "tei.reference.qualified-reference-reuse/v0",
  "patternId": "repair_intake.customer_reference",
  "application": "repair_intake",
  "relation": "repair_intake.intake.customer",
  "datasource": "customer_master",
  "placement": "applications/repair-intake/patterns/customer-reference.tei"
}
```

```bash
go run ./tools/reference-shared-acceptance propose-qualified-reference-pattern \
  -workspace DIR \
  -candidate base-candidate.json \
  -qualification qualification.json \
  -reference-reuse reference-reuse.json \
  > proposed-candidate.json
```

この経路では、originのPattern ID、Relation、datasource aliasをdestinationへコピーしません。Contract / Query / input mappingだけをcurrent Canonicalから読み、qualificationがそのContract Queryを明示的に評価している場合に限って使います。したがって、探索結果が`applicable`でも、未評価の依存関係を暗黙にProposalへ混ぜることはありません。Proposal生成後もAcceptanceは成立しておらず、通常どおり`preview` / `review` / `accept`を通します。

Previewでは、添付された結果をそのまま信用しません。current Source Indexに対してexact subjectとsourceを解決し、source fingerprintを照合したうえで、同じrequirementsを使ってqualificationを再実行します。required dependencyがある場合は、そのauthoritative sourceとfingerprintの集合もcurrent qualificationと照合します。したがって、dependencyのIDが同じままsource内容だけが変わった場合もstaleとして拒否できます。一方、requirementsが参照していないsourceの変更はqualification freshnessへ持ち込まず、repository-wideなglobal revisionは要求しません。

このEvidenceはcandidateの意味を変更しないため、`candidateFingerprint`には入りません。一方、Preview / Decisionは正規化済みEvidenceの`evidenceFingerprint`を別に持つため、「同じ意味だが根拠が違う」candidateを機械的に区別できます。Accepted DecisionにはEvidence本文とそのidentityがprovenanceとして残りますが、Acceptance、write authority、Runtime authorityを与えません。 Decision Indexは新しい記録について保存済み`evidenceFingerprint`とEvidence本文も照合し、記録だけを書き換えた場合を`EVIDENCE_FINGERPRINT_MISMATCH`として検出します。
### Semantic Evaluation artifactからcandidateへの接続

Semantic Extension Referenceが出力する`tei.reference.semantic-evaluation-evidence/v0`は、Source Index上のAuthoring sourceではありません。そのため、既存の`subject / source / sourceFingerprint`へ偽装して取り込みません。`attach-semantic-evaluation`は、Evaluation Evidenceをartifact-only Evidenceとしてcandidateへ付与します。

```bash
go run ./tools/reference-shared-acceptance attach-semantic-evaluation \
  -candidate candidate.json \
  -semantic-evaluation semantic-evaluation.json \
  > grounded-candidate.json

go run ./tools/reference-shared-acceptance review \
  -workspace DIR \
  -candidate grounded-candidate.json
```

handoffでは次を検査します。

- schemaが`tei.reference.semantic-evaluation-evidence/v0`である
- Semantic Set、Evaluator contract、Evaluator artifact、Conformance Evidence、input、Evaluation Evidenceの各fingerprintがexact SHA-256である
- Constraint / subject / validator / evaluator identityが空でない
- statusが`pass / fail / unknown`のいずれかである
- diagnosticがproducerと同じcanonical orderである
- Evaluation Evidence自身の`evidenceFingerprint`が内容から再計算できる
- さらにcanonical payload全体へ別の`artifactFingerprint`を付け、Shared Acceptance内のartifact bindingとして固定する

Source Index Evidenceとartifact Evidenceは一つのEvidence itemへ混在できません。必要なら別itemとして並べます。これにより「workspaceのauthoritative source」と「外部または派生Evaluation artifact」を同じauthorityとして扱いません。

Evaluation artifactはcandidateの意味を変えないため、`candidateFingerprint`には入りません。一方、`evidenceFingerprint`とReview Projectionには入るため、同じsemantic deltaでも「どのEvaluation Evidenceを見て判断したか」は区別できます。Accepted Decisionにもartifact本文とfingerprintが残ります。

重要なのは、EvaluationのstatusをAcceptance authorityへ昇格しないことです。`pass`だから自動Acceptせず、`fail`や`unknown`だから自動Rejectもしません。A3/A7へ状態とdiagnosticを表示し、最終的なAccept / RejectはShared Acceptanceの明示Decisionとして残します。これは失敗を無視するという意味ではなく、Evaluatorの判定とAcceptance Decisionを別の責任として保持するための境界です。

また、Decision Indexがartifactのfingerprint整合を確認できても、現在のEvaluator implementation、Conformance Evidence、Semantic Setを再実行してfreshnessを証明したことにはなりません。将来それを要求する場合は、Evaluation再実行またはcurrent Evidence bindingを別の検査として追加します。

### candidateの例

```json
{
  "schema": "tei.reference.change-candidate/v0",
  "id": "cand-unit-price-positive",
  "baseModelRevision": "sha256:<revisionの出力>",
  "origin": {"kind": "cli", "actor": "developer"},
  "intent": "order lineの単価が正であることを業務ruleとして明示する",
  "evidence": [
    {"subject": "sales.order_line.quantity_positive", "note": "同じ形のruleが数量には既にある"},
    {"source": "semantic/sales/validation.tei"}
  ],
  "delta": [
    {"op": "add-validation", "namespace": "sales", "id": "sales.order_line.unit_price_positive",
     "kind": "positive", "attribute": "sales.order_line.unit_price"}
  ]
}
```

- `origin.kind`は`file` / `cli` / `studio` / `ai`。provenanceだけで、fingerprint・判定・source・Canonical Worldへ影響しません（AI固有のmodel名・prompt digest等は`origin.notes`に置けますが、どこにも保存されません）。
- 対応するoperationは次の表のとおりです。任意のTEI textやfile patchは運ばず、sourceは意味から決定論的に生成します。Relation変更等は、このReferenceで必要になった時点で追加します。

| operation | 意味 | source |
| --- | --- | --- |
| `add-validation` | 既存namespaceへValidation（`positive` / `relation-unique`）を追加する | 既存fileの末尾へ追記（`placement`は省略可。省略時は同じnamespaceの既存Validationがあるfile） |
| `add-data` | **新しい**semantic namespaceへData（Attribute・Relation・key）を追加する | 新しいsemantic file（`placement`必須） |
| `add-relation` | **既存の**DataへRelation（key付き / 概念Relation）を一つ追加する | 対象Dataを宣言したfileのdata blockの、閉じ括弧の行の直前へ挿入（`placement`は持たない。閉じ括弧が行頭に無いlayoutは拒否） |
| `add-application` | Environment bindingを含まないApplication（semantic利用・Database requirement）を追加する | 新しい`applications/<dir>/application.tei`（`placement`必須） |
| `add-reference-pattern` | reference Patternを追加する | 新しい`applications/<dir>/patterns/<file>.tei`（`placement`必須） |
| `add-application-validation-use` | **既存の**Applicationが評価するValidationを`uses validation`として一件宣言する。宣言順が評価順になる（ADR-0029） | 既存のApplication source fileの、最後の`uses`宣言の後ろへ一行挿入（`placement`は必須で、そのApplicationを宣言しているfileに限る）。同じApplicationへの複数の宣言は、candidateが指定した順序を保つ（この順序はfingerprintにも入る） |

- 一つのcandidate内で`add-validation`と、そのValidationを使う`add-application-validation-use`を同時に記述できます。参照先のIDは既存のCanonical Worldまたは**同じcandidateで宣言された追加**から決定論的に解決し、未知のIDを黙って認めません。重複するoperation identityは既存の`normalizeCandidate`で拒否します。この経路は、業務規則だけを追加してApplicationが評価しない状態を避けるためのReferenceです。実装・テストが含まれるDraft PR #985の資格確認が完了するまでは、動作確認済みとは扱いません。
- 新しいfileを作るoperationのplacementは、意味を所有する層のprefix（semantic / applications）とloaderが読む形式に合い、まだ存在せず、他のoperationと重ならない必要があります。PatternはそのPatternを所有するApplicationのdirectoryに置きます（loaderはApplication manifestと同じdirectoryの`patterns/`をそのApplicationのものとして読みます）。
- `usesSemantic`や`databases`の順序は、authoredされたsourceの順序としてそのまま保ちます（入れ替えるとsourceとfingerprintが変わります）。
- 変わるべきWorld collectionは、deltaの種類から導きます（semantic層のoperationは`semantics`、Application層のoperationは`applications`）。実際に変わったcollectionの集合とexactに一致しなければ拒否します。
- 受理はfileを新しく作るため、失敗時は作ったfileと、そのために作ったdirectoryを削除して元へ戻します（既存fileへの挿入・追記は元のbyteへ戻します）。
- Relationの追加・変更では、Previewが端点（source / target Data）を実現するPersistence Mappingへの辺と、Relationの保存方法（relation storageが定義済み / keyのsource Attributeが列として実現済みで導ける / 未決定）を返します。未決定なら`RELATION_STORAGE_NOT_AUTHORED`として残し、この変更がPersistence Mapping・migrationを変えないことも示します。

## このtoolが示すこと / 示さないこと

示すこと（このtoolの`boundary_test.go` / `provenance_test.go` / `containment_test.go`と、`changeanalysis`の`analysis_test.go`）:

- Preview / pure Rejectはauthoritative sourceをbyte単位で変更しない。CLI RejectはDecision ledgerだけを増やし、model revisionとCanonical Worldを変更しない
- 受理した意味はtoolを介さず共有loaderで再読込でき、Validationを満たし、受理前後で評価結果が変わる
- staleなcandidateを黙って適用しない
- 不正なcandidate（未知field / 存在しない参照 / 別layerへのplacement / 数値ではないAttributeへの`positive` / 宣言外の副作用）はsourceを変えずに拒否される
- どの入口のcandidateも同じ`candidateFingerprint`・同じsource・同じCanonical Worldになり、AI固有のEvidenceがCanonicalへ入らない。同値なEvidenceは同じ`evidenceFingerprint`へ正規化され、Evidenceだけを変えてもsemantic fingerprintは変わらない
- accept途中の並行変更・受理後Conformance失敗では、書いたfileを元へ戻す
- 同じ意味を別fileへ移すだけならsemantic差分は出ず、論理的な改名はremoved / addedとして現れる
- 派生物（Canonical World JSON）を消しても、accepted sourceから同じものを再生成できる
- Environment binding: 既存のEnvironment fileへ`bind <application>.<alias> -> <instance>`を一行挿入します。Environment blockの閉じ括弧の直前へ入れ、内側のblock（`database-instance { … }`等）と取り違えません。挿入先は、そのEnvironmentを宣言しているfileに限ります。
- `ISSUE-000046`が最小loopとして名指しするRelation変更（既存Data `sales.order_line`へ`order`を追加）について、Preview（Semantic Diff・exact dependencyのimpact・Persistence Mappingへの辺と保存方法が未決定であること）→ Reject → Accept → toolなしの再読込までを通し、変わるのは一つのfileの挿入した行だけである（`relation_test.go`）
- Second Application `repair_intake`（改訂版S4 Proposal、SHA-256 `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`）を、3つの型付きoperationとして表し、source・payload・candidate identityをS4 Proposalへ結び付けられる（`composition_test.go`）
- そのcompositionをscratch copyへacceptすると、宣言した3 fileだけが作られ、Environmentは変わらず、受理後の再読込・Conformance・Previewとの差分一致が通る。並行変更・受理後Conformance失敗では、3 fileと作ったdirectoryを元へ戻す
- 同じcompositionを実際の`reference/workspace`へPreviewすると全stageがPASSし、`reference/workspace`は変更されない（Previewの結果は`plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_IMPLEMENTATION_PACKET_2026-09.md`「実装記録」）
- `ISSUE-000047`の`applicable`なqualificationをexact Evidenceとしてcandidateへ付与し、Preview時にcurrent Source Indexと同じrequirementsへ戻って再qualificationできる。required dependencyのauthoritative source / fingerprintもscoped Evidenceとして照合し、同じIDのdependency sourceが変わったcaseをfail closedにできる。requirements外のdependency Evidenceはhandoffしない（`qualification_handoff_test.go`）
- Semantic Evaluation Evidenceをartifact-only Evidenceとしてcandidateへ付与し、内部Evaluation fingerprintと外側artifact fingerprintを二重に固定できる。`pass / fail / unknown`はReview A3/A7へ露出するが、candidate meaningやAcceptance authorityへ昇格しない（`semantic_evaluation_handoff_test.go`）
- accepted / rejected Decisionが、会話履歴なしでworkspaceから読める。Decision Indexはworkspace全体の`current / superseded`、そのDecisionが書いたsourceの`intact / diverged / none`、判断理由に使ったEvidenceの集約statusとSource/artifact別statusを返す。stale rejectはrevision conflictも保持する。記録はauthorityではなく、消してもmodel revisionもaccepted meaningも変わらない（`record_test.go` / `semantic_evaluation_handoff_test.go`）
- Environment binding（`add-environment-binding`）をApplication意味の変更と分けて扱える。差分は`environmentDiff`へ出て意味の差分は空のまま、変わるcollectionは`environments`だけ、impactはEnvironment / Application / database instanceへexact identityで辿り、権限が付かないことを`RUNTIME_AUTHORITY_NOT_GRANTED`として残す（`environment_test.go`）
- そのbindingでも、層をまたぐ正しさ（unknown alias / unknown instance / target mismatch）は境界が再実装せず、共有Validationのdiagnosticとして`validate` stageに現れる

示さないこと:

- AIやStudioがcandidateのdeltaを生成できること、knowledge-firstのretrievalやranking自体
- Driver scenario / Web Target Projectionへの影響の実行（Previewは`DRIVER_SCENARIO_NOT_RUN` / `TARGET_PROJECTION_NOT_EVALUATED`をunresolvedとして明示する）
- Validationを追加しただけでは、実行時に評価されるとは限らないこと。Applicationが評価するValidationは`uses validation`の宣言から選ばれる（ADR-0029）ため、Previewは宣言していないApplicationを`APPLICATION_VALIDATION_SELECTION_NOT_DECLARED`として示す。宣言を足すのは別のoperation（`add-application-validation-use`）である
- Runtime操作のeffect-time authority、file move / renameの自動検出、Derived Experience Knowledge
- 決定記録が`RQ-027`を閉じること。この境界を通らずに変更したsourceは記録を残さず、`Variation`はまだ表現を持たず、Gate Evidence / Release locatorは索引化されていない。Coreにも同等の仕組みはない

## 検証

```bash
go test ./tools/reference-shared-acceptance -count=1
```

testはGate B / Gate Cのbaselineを変更せず、`reference/workspace`のscratch copyだけを使います。最初のtestは、scratch copyから得たCanonical Worldが`reference/world.json`とbyte単位で一致すること（実際のReferenceで試していること）を確認します。
