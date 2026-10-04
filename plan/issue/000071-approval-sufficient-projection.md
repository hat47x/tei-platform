# ISSUE-000071: 人が中身を確かめて承認できるProjectionの条件を定め、Shared Acceptance Previewを評価する

- 状態: Proposed — 評価待ち。条件案と、既存Previewに対する机上評価までを記録した段階で、人が実際にこのProjectionから承認した記録はまだない
- 提案日: 2026-09-30
- 性格: Requirement詳細化 / 既存Referenceの評価
- 主担当要求: RQ-028（主）/ RQ-010 / RQ-016 / RQ-021
- 関連Issue: [ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)（受理境界）、[ISSUE-000046](000046-studio-shared-acceptance-authoring-loop.md)（Studio）、[ISSUE-000053](000053-ai-native-development-and-agent-operational-knowledge.md) Phase 2（human review challenge）
- 価値との対応: [Product Value](../../product/value/TEI_PRODUCT_VALUE.md)「個人 3. 承認が、中身のある判断になる」「社会 3. 人による監督を、名目にしない」、原則P19
- Gate impact: None

## この文書の役割

RQ-028は「人間が実装詳細を全面的に読まずに変更を理解・承認できること」を求めていますが、**どういうProjectionなら承認に足りるのか**を判定する条件がありません。[Product Requirement Gap Audit](../PRODUCT_REQUIREMENT_GAP_AUDIT.md) §8の優先順位6も、RQ-028について「承認に十分なprojectionの条件を定義し、Shared Acceptance Previewがそれを満たすか評価する」ことを求めています。

このIssueは、その条件案を定め、既存の`tools/reference-shared-acceptance`のPreviewを条件に照らして評価します。条件案は評価待ちであり、Requirement本文を変更するものではありません。

## 背景

2026-09-21に、第二Application `repair_intake`のProposal（`c18ae0fd…`、741 bytes）が実際の`reference/workspace`へ受理されました（[ISSUE-000040](000040-semantic-component-composition.md)）。Proposalは生成AIが作り、別の生成AIが独立レビューし、Repository ownerが明示的に受理しています。

このときの承認は、Claude Code session上の回答「Acceptします。」として記録されています。ISSUE-000040のwitness記録の「直前に提示した内容」は、「独立レビュー完了（review-clear-with-conditions、必須修正なし）、残る停止条件はexplicit human Acceptance decisionだけであること」の2点です。Previewは機械可読なJSONとして全stage PASSしていました。PreviewのJSONや実装記録（S5 Implementation Packet §13）をownerがどの形で読んだか、AIが何をどう要約したかは記録されておらず、未確認です。確かなのは、TEIが出す人向けProjectionを介した承認の記録は無いことです。

つまり、承認の根拠を人がPreviewから直接確かめた記録は無く、承認が「PreviewのJSONが正しい」ことと「AIがそれを正しく要約した」ことの二つに依存していた可能性があります。後者はTEIの境界の外にあります。

## 防ぎたい失敗

- **形だけの承認。** 承認者が差分の意味・影響・未解決点を自分で確かめられず、AIの要約や「全stage PASS」という表示だけで承認する。責任だけが承認者に残る。
- **要約による欠落。** AIや人がPreviewを要約するとき、unresolvedや影響の一部を落とす。落ちたことを承認者が気づけない。
- **示されない種類の変更。** 権限の変更やnative codeへ残る箇所がPreviewの項目に無いため、「示されなかった」ことが「変わらなかった」と読まれる。
- **大きすぎる承認単位。** 一度に承認する対象が人の判断できる大きさを超え、実質的に全体を信じるしかなくなる。

## 守る原則

- 承認に足るかどうかは、Projectionの見た目ではなく、**承認者が判断に必要な項目へ、正本から欠落なく到達できるか**で決める（RQ-028、RQ-021）。
- 「示す項目が無い」ことと「変化が無い」ことを区別する。項目が無いものは、無いと明示する（原則P11）。
- 要約は許すが、要約から正本（Canonical ID / Source / Evidence）へ戻れなければならない（RQ-028）。
- 承認単位の大きさには上限を持つ。上限を超える変更は分割するか、分割できない理由を示す（原則P19）。

## 条件案 — 承認に足るProjectionのA1〜A8（評価待ち）

RQ-028が列挙する確認対象と、Product Valueの「形だけの承認」を防ぐ観点から、次の8条件を案とします。

| 条件 | 承認者が答えられるべき問い | 根拠 |
| --- | --- | --- |
| A1 意味差分 | 何の意味が、追加・変更・削除されたか | RQ-028、RQ-010 |
| A2 影響 | その変更が、どのData / Application / Pattern / Target / Environmentへ及ぶか。どこまで辿れていないか | RQ-028、RQ-026 |
| A3 例外・診断 | Validation diagnosticや業務上の例外で、判断を要するものは何か | RQ-028 |
| A4 未解決点 | 何がまだ分からず、どの判断を先送りしたか | RQ-028、原則P11 |
| A5 権限の変更 | 誰／何が、新たに何をできるようになるか。または権限が変わらないことが示されているか | RQ-028、RQ-019 |
| A6 native codeへ残る箇所 | 生成・宣言で表せず、custom implementationとして残る箇所はどこか | RQ-028、RQ-004 |
| A7 逆追跡 | 上の各項目から、Canonical ID・source・Evidenceへ戻れるか。要約が正本と一致することを確かめられるか | RQ-028、RQ-021 |
| A8 承認単位の大きさ | 一度に判断する対象が上限内か。上限を超える場合、分割できない理由が示されているか | 原則P19 |

「示す項目がそもそも無い」場合は、その条件を満たさないものとして扱います。項目が無いまま承認した場合、承認記録にその旨を残すことを想定します（記録の形は未決定）。

## 机上評価 — 既存のShared Acceptance Preview

評価対象は`tools/reference-shared-acceptance`の`preview`出力です。根拠はtoolのREADMEと、`plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_IMPLEMENTATION_PACKET_2026-09.md` §13の実Preview記録（`repair_intake`、11 stage PASS、変更subject 6件、impact 15本、unresolved 3件）です。コードは実行していません。

| 条件 | 評価 | 根拠と不足 |
| --- | --- | --- |
| A1 意味差分 | 満たす（機械可読） | `semanticDiff` / `applicationDiff` / `environmentDiff`。宣言したdeltaと実際の差分の一致も検査する |
| A2 影響 | 一部満たす | exact identityに基づく`impact`辺を返す。Driver scenarioとTarget Projectionへの影響は評価せず、`DRIVER_SCENARIO_NOT_RUN` / `TARGET_PROJECTION_NOT_EVALUATED`としてunresolvedへ明示している。辿れない範囲を隠していない点は条件に合う。ただし`Impact`は`removed`のsubjectを辿らず、辺もunresolvedも出さない（`impact.go:48-51`）。現在のoperationは追加だけなので顕在化していないが、削除・変更を扱う時点で再評価が要る。Environmentへ辿るのは`environment-binding`のsubjectだけである（`impact.go:93-99`） |
| A3 例外・診断 | 一部満たす | `validate` stageに出るのはerror severityのValidation diagnosticだけで、warningはPreviewへ出ない（`pipeline.go:202-219`）。errorが無ければstageはpassとなり、diagnosticは何も出力されない。業務上の例外（どの業務ケースで規則が効くか）を示す項目も無い |
| A4 未解決点 | 満たす（機械可読） | `unresolved`。例: `ENVIRONMENT_BINDING_NOT_AUTHORED`、`RUNTIME_AUTHORITY_NOT_GRANTED` |
| A5 権限の変更 | 満たさない（一部のoperationでは「変わらない」ことだけ文章で示す） | 権限の変化を独立に示す項目が無い（`pipeline.go:42-61`）。`add-application`ではunresolvedの`ENVIRONMENT_BINDING_NOT_AUTHORED`のmessageが「Environment / Runtime authorityはこの変更では変わらない」と述べ（`impact.go:123-129`）、`add-environment-binding`では`RUNTIME_AUTHORITY_NOT_GRANTED`が権限が付かないことを述べる（`impact.go:130-136`）。それ以外のoperationにはこうした言明が無く、`changedCollections` / `unchangedCollections`から間接的に読むしかない |
| A6 native codeへ残る箇所 | 満たさない | 項目が無い（`pipeline.go:42-61`）。現在の6 operation（`operation.go:14-21`）はnative codeを変えず、deltaから導いたcollection（semantics / applications / environments）以外が変われば`UNEXPECTED_COLLECTION_CHANGE`で拒否する（`pipeline.go:271-286`）。ただし、その「変えない」ことを承認者へ示す項目は無い |
| A7 逆追跡 | 一部満たす | `sourceEdits`（`path` / `layer` / `afterFingerprint` / `insertedText`と、既存file変更時だけの`beforeFingerprint`。`edit.go:23-29`）、subject ID、candidate fingerprint・Originがある。Evidenceは`resolve` stageで実在を検査するが、結果が出るのは失敗diagnosticとしてだけで（`resolve.go:188-194`）、Preview JSONにはIntentもEvidence本体も含まれない（`pipeline.go:42-61`）。IntentとEvidenceが読めるのは、受理後のDecision記録だけである（`decision.go:53-55`）。subject IDから該当source fileへ戻る対応も出力しない。ただし人向けの要約が無いため、「要約が正本と一致するか」を確かめる手段も無い |
| A8 承認単位の大きさ | 一部満たす | `maxSubjectChanges = 8`（`pipeline.go:18`）を、`acceptance-policy` stageが変更subject数（semantic / application / environmentの差分の合計、`pipeline.go:255`）に対して検査し、超えれば`DELTA_TOO_LARGE`で拒否する（`pipeline.go:254-262`）。ただし、超過は拒否するだけで、分割できない理由を示して承認を求める経路は無い（A8の後半を満たさない）。また、通ったPreviewには変更subject数も上限も出力されないため、人は差分を数えるしかない。whole-model再生成を標準経路にしない点は条件に合う |

**机上評価の結論:** Previewは、承認に必要な情報の多くを機械可読な形で正本から出しています（A1 / A4。A8は上限の検査のみ）。一方、人が読む面が無く、権限の変化とnative codeへ残る箇所を示す専用の項目もありません（A5 / A6）。このため、人がPreviewから直接確かめて承認する形にはなっていません（2026-09-21の受理で、ownerがPreviewをどの形で読んだかは記録にありません）。

## 机上評価の拡張 — 他のoperation

`repair_intake`のcomposition以外のPreviewでも、同じA1〜A8の観点で机上評価します。対象は`add-validation`（`boundary_test.go`の単価Validation）、`add-relation`（`relation_test.go`の`sales.order_line.order`）、`add-environment-binding`（`environment_test.go`の`repair_intake.customer_master`のbinding 2件）です。根拠は、`tools/reference-shared-acceptance/`のtestのassertionと、`tools/internal/changeanalysis/`の`impact.go`・`diff.go`・`analysis_test.go`・`persistence_test.go`、`tools/reference-shared-acceptance/pipeline.go`の読解です。コードは実行していません。

評価の語は上の表と同じです。「満たす」は承認者が答えを機械可読な項目から得られること、「一部満たす」は項目はあるが問いの一部にしか答えないこと、「満たさない」は答える項目が無いことを指します。

| 条件 | `repair_intake`（既存の評価） | add-validation | add-relation | add-environment-binding |
| --- | --- | --- | --- | --- |
| A1 意味差分 | 満たす | 満たす。`semanticDiff`はvalidation 1件（`boundary_test.go:70-72`） | 満たす。`semanticDiff`はrelation 1件、`applicationDiff`は空（`relation_test.go:50-55`） | 満たす。`semanticDiff` / `applicationDiff`は空で、決定は`environmentDiff`のbinding 2件に出る（`environment_test.go:62-69`）。意味が変わらないことと、何が決まったかが別の配列に分かれる |
| A2 影響 | 一部満たす | 一部満たす。辺5本（Application `order_entry`と`order_entry.order_lines` Patternへ届く）。未解決は`APPLICATION_VALIDATION_SELECTION_NOT_CANONICAL`、`DRIVER_SCENARIO_NOT_RUN`、`TARGET_PROJECTION_NOT_EVALUATED`（`boundary_test.go:94-123`） | 一部満たす。Persistence Mappingへの辺2本を含む（`relation_test.go:76-81`）。保存方法は`RELATION_STORAGE_NOT_AUTHORED`として残る（同:90-94） | 一部満たす。Environment / Application / database instance / Databaseへの辺（`environment_test.go:82-92`、`impact.go:448-470`）。Driver / Targetの未解決は無条件に付く（`impact.go:145-148`）。辺はこの4種だけで、Environmentが持つ既存の権限宣言へ向かう辺は`impact.go:448-470`に無い |
| A3 例外・診断 | 一部満たす | 一部満たす。診断はvalidate stageに出る（`boundary_test.go:301-303`）が、PASSしたPreviewでは「無い」としか分からず、業務上の例外は項目に無い | 一部満たす。拒否時にはnormalize / resolve / validate stageへ出る（`relation_test.go:201-223`）。PASS時は同上 | 一部満たす。unknown alias / unknown instance / target mismatchは共有Validationのdiagnosticとしてvalidate stageに出る（`environment_test.go:162-194`）。PASS時は同上 |
| A4 未解決点 | 満たす | 満たす。Application側が`uses validation`で宣言済みなら当該codeは消える（`tools/internal/changeanalysis/analysis_test.go:345-364`） | 満たす。保存方法が導けるRelationでは`RELATION_STORAGE_NOT_AUTHORED`は出ない（`tools/internal/changeanalysis/persistence_test.go:75-95`）。出る場合はMappingを名指しする（同:69-72） | 満たす。`RUNTIME_AUTHORITY_NOT_GRANTED`が、binding 1件ごとに出る（`impact.go:130-136`） |
| A5 権限の変更 | 満たさない（下記の補足あり） | 満たさない。権限を述べる項目は無く、`environments`が不変であることを不変collectionの一覧から読めるだけ（`boundary_test.go:76-80`） | 満たさない。同上（`relation_test.go:59-63`） | 満たさない。ただし境界例（下記）。`RUNTIME_AUTHORITY_NOT_GRANTED`が「Permission grant・Resource binding・lease・実行principalはこの変更に含まれない」と述べる（`environment_test.go:94-101`、`impact.go:130-136`）が、`unresolved`の自由文であり、権限の変化を示す独立項目ではない |
| A6 native codeへ残る箇所 | 満たさない（下記の補足あり） | 満たさない。専用の項目は無い | 満たさない。同上 | 満たさない。同上 |
| A7 逆追跡 | 一部満たす | 一部満たす。`sourceEdits`に挿入するsourceが出る（`boundary_test.go:82-88`）。EvidenceのsubjectとsourceはPreview時にresolveされる（同:261-262） | 一部満たす。挿入行まで確認できる（`relation_test.go:65-70`）。差分と辺のsubject IDから正本へ戻れる | 一部満たす。受理時に変わるのは2 fileのbind 1行ずつ（`environment_test.go:124-141`）。このcandidateのEvidenceはApplication sourceの指定1件だけで、「なぜこのinstanceか」を示す根拠は含まない（同:19-21）。Preview時の`sourceEdits`は同testでは検査されておらず未確認 |
| A8 承認単位の大きさ | 満たす（6 / 8） | 満たす（1 / 8） | 満たす（1 / 8） | 満たす（2 / 8）。`allChanges`が`environmentDiff`も数える（`pipeline.go:414-418`） |

読んだ範囲では、どのoperationでも人が読む要約は無く、「要約が正本と一致するか」を確かめる手段が無い点（A7）は`repair_intake`と同じです。

### 評価が変わるか

確定して変わる評価はありません。A1〜A4、A6〜A8は`repair_intake`と同じ水準で、A5だけが境界例です。

`add-validation`と`add-relation`では、Previewは権限について何も述べず、承認者は`unchangedCollections`に`environments`が入っていることから「権限を持つ層は動いていない」と推測するしかありません。

`add-environment-binding`では`RUNTIME_AUTHORITY_NOT_GRANTED`が出ます。これは「この変更は権限を付与しない」という否定の注記であり、誰が何を新たにできるようになるかには答えていません。しかも`unresolved`の自由文です。`repair_intake`の`ENVIRONMENT_BINDING_NOT_AUTHORED`も同じ種類の自由文で「Environment / Runtime authorityはこの変更では変わらない」と述べるので（`impact.go:123-128`）、前者だけを「一部満たす」とすると評価の物差しがそろいません。そのためここでは、どちらも「満たさない」のままとし、`unresolved`の自由文でA5を満たしたと数えるかは、条件の定義の問題としてowner reviewに残します。数える場合は、`repair_intake`とadd-environment-bindingの両方が「一部満たす」になります。

もう一つ、この読解で分かった点があります。権限の宣言とbindingは同じEnvironment fileに同居します（`environment_test.go:144`が、`grant deployment-executor`と`lease-policy deployment-executor`の両行が同じfileに残ることを検査している）。一方、`environmentDiff`のsubjectになるのはbindingだけです（`diff.go:203`）。このため、権限宣言が変わった場合に`environmentDiff`へ出るかは、読んだ範囲では確認できず、`changedCollections`が`environments`と示すだけでは、bindingの追加と権限宣言の変更を区別できません。collectionの一覧がA5の代わりにならないことが、この例から分かります。

また、bindingで結んだinstanceにEnvironmentの既存の権限宣言が及ぶかどうかをPreviewが評価しているかは、`impact.go:448-470`の辺に出ないことしか確認できていません（未確認）。「この変更は権限を付与しない」と「既存の権限が新しいbindingに及ぶか」は別の問いなので、A5の定義を決めるときに分けて扱うことを、owner reviewの論点として挙げます。

### 既存の`repair_intake`の評価への補足

- **A5。** 上のとおり、`repair_intake`のcompositionでは`ENVIRONMENT_BINDING_NOT_AUTHORED`が付き、その説明文は権限が変わらないと述べます（`impact.go:123-128`、`composition_test.go:350-358`）。「項目なし」ではなく、**独立した項目ではなく`unresolved`の自由文の中でだけ述べている**と書くのが正確です。
- **A6。** `unchangedCollections`は`customGoBindings`を含みます（`composition_test.go:173`）。この一覧は`CollectionDiff`がWorld直下の全collectionを比較して作るため（`diff.go:213-239`）、他のoperationにも同じ形で出ると読めます。ただしassertionで確かめたのは`repair_intake`のtestだけで、`customGoBindings`がnative codeへ残る箇所を過不足なく表すかも確認できていません（未確認）。A6を示す専用の項目が無いという評価は変わりませんが、「間接的な手がかりはある」と書き添える必要があります。

この拡張評価は、机上の読解に基づく評価待ちの記録です。

## 必要能力

1. Preview結果を、条件A1〜A8の順に人が読める形へ写すProjection。JSONの各項目へ戻るlinkまたはID参照を持つ。これはRQ-010のReview Surfaceの最小形であり、Studio UIの完成を待たない。
2. A5 / A6を示す項目。変わらない場合も「変わらない」と示す。
3. 承認記録（`.tei/decisions/`）に、承認時に満たしていなかった条件を残すこと。

## 2026-10-04 最小Review Projectionのcandidate実装

A1〜A8をRequirementとして採用する前に、既存Previewだけで人がどこまで判断できるかを実物で確認するため、`tools/reference-shared-acceptance`へread-onlyの`review` commandをcandidate実装した。実装は既存の`evaluate`を再利用し、Acceptanceの判定規則やCanonical modelは増やしていない。

`review`はA1〜A8を固定順で表示する。冒頭には`表示可能`・`一部`・`未評価`の件数と、確認が残るsectionをまとめる。A1〜A4は意味差分、影響、diagnostic、未解決点を既存Previewから取り出す。各sectionには導出元の`preview.*`または`candidate.*`項目を`sources`として残し、text出力でも`根拠:`として表示する。`candidateFingerprint`は意味、`evidenceFingerprint`は根拠、`reviewFingerprint`はA1〜A8を含む提示Projectionのidentityとして分ける。A7はEvidenceと変更予定sourceのpath・fingerprintも表示する。A8は変更subject数と既存上限を示す。

A5の権限変更とA6のnative code残存箇所には、現在のPreviewだけでは完全な専用評価がない。candidateのtyped operationから、Permission grant・Resource binding・lease・実行principalや`customGoBindings`を直接authorしないことは決定論的に確認できるため、この範囲は`一部`として表示する。一方、既存権限が変更後のApplicationやbindingへどう及ぶか、既存custom implementationのどこが変更subjectに依存するかは未評価として残す。前者を根拠に後者まで「変更なし」とは判定しない。

既定出力は人が読むtextで、`-format json`を指定すると同じProjectionを機械可読で取得できる。テストでは、A1〜A8の順序、A5/A6でcandidate自身のauthoringと未評価の実効影響を分けること、A7の逆追跡情報、A8の変更単位、`reviewFingerprint`の再現性、text / JSONの両出力、sourceを変更しないこと、途中stageで失敗した場合に後段の情報を推測で補わないことを対象にしている。

ReviewとAcceptの取り違えを防ぐため、`accept`には任意の`-expected-review-fingerprint`も追加した。Accept時にcurrent Reviewを再構築し、期待したfingerprintと違えばsource変更前にfail closedする。一致した場合は、`reviewBinding`としてfingerprintとcoverageをaccepted Decisionへ残し、`decisions`から会話履歴なしで再取得できる。

Decision Indexでは、Review bindingの有無と構造を`recorded / none / invalid`、coverageを`complete / incomplete / none / invalid`として別に示す。`incomplete`の場合は、受理時に`partial`または`not-evaluated`だったA1〜A8も列挙する。読み込み時は、file名に使ったSHA-256とrecord実内容を先に照合し、同じpathのまま書き換えられたrecordを拒否する。その後にJSON/schema、Source、Evidence、Review bindingを検査する。これは「どのReviewへAcceptを結び付け、何が未充足だったか」を再入場時に探しやすくするための索引であり、人が実際にReviewを読んだことの証明ではない。content-addressも署名ではなく、Decision recordだけから元Projectionや承認者の真正性を主張することはしない。

このcandidate実装は、A1〜A8の採用判断や「承認に十分」という結論を先取りしない。`reviewFingerprint`と`reviewBinding`は提示内容とのbindingを表すが、human approvalそのものを証明するtokenではない。`previewAcceptable`もShared Acceptanceの機械判定だけを表し、Review coverageの不足が解消されたことは意味しない。exact-headのfull Repository-health PASSもまだ取得していないため、下記の完了条件は現時点では更新しない。

## 具体例

`repair_intake`のPreviewをA1〜A8の順に写すと、たとえば次のようになります。

```text
A1 意味差分    semantic 4件（namespace / data / attribute / relation）、application 2件
A2 影響        15本。Driver scenario・Target Projectionは未評価（unresolved）
A3 例外・診断  validation error 0件（全stage PASS）。業務上の例外を示す項目は無い
A4 未解決点    ENVIRONMENT_BINDING_NOT_AUTHORED / DRIVER_SCENARIO_NOT_RUN / TARGET_PROJECTION_NOT_EVALUATED
A5 権限の変更  専用の項目なし。unresolvedのmessageが「権限は変わらない」と述べるのみ（満たさない）
A6 native code 項目なし（満たさない）
A7 逆追跡      作られるfile 3件と各fingerprint、candidate fingerprint 8a936cf9…
A8 大きさ      変更subject 6件 / 上限8
```

A6が「項目なし」と明示されていれば、承認者は「native codeは変わらないと示されたのではなく、示す仕組みが無い」と分かったうえで判断できます。A5は専用の項目こそ無いものの、`repair_intake`ではunresolvedのmessageが権限は変わらないと述べているため、「示す仕組みが全く無い」とは言えません。ただし、そのmessageはoperationによっては出ないため、承認者がmessageの有無に頼らず確かめられる専用の項目が要ります。

## 完了条件

- [ ] 条件A1〜A8を、owner reviewで採用・修正・却下のいずれかに判定する。採用した条件はRQ-028の解釈として、このIssueまたはADRへ残す
- [x] `repair_intake`以外のPreview（`add-validation`、`add-relation`、`add-environment-binding`）でも机上評価を行い、評価が変わる条件を記録した（2026-10-02、上の「机上評価の拡張」。コードは実行していない）
- [ ] Preview結果をA1〜A8の順に写す最小Projectionを、Reference toolとして作る（candidate実装あり。source変更を伴うため、`tools/repository-health-check.sh`のexact-head full PASSが統合条件）
- [ ] そのProjectionだけを見て、人が一件の変更を承認または却下した記録を残す（AIの要約を介さない）。承認者が気づいた不足を記録する
- [ ] A5 / A6を示す項目を追加するか、追加しない理由と、承認時にそれを明示する方法を決める（candidate実装では、typed operationから直接authorするgrant / native codeの有無までを`一部`として表示し、実効権限と既存native依存は未評価として残す）

## 完了後に判断できること

- RQ-028の「承認に足る」をGap AuditでPartialから引き上げてよいか
- Studio UI（ISSUE-000046）が最初に表示すべき項目と順序
- AIが承認の前段に入る場合、AIの要約をどこまで信用でき、どこで正本へ戻すべきか

## 非主張範囲

- 条件A1〜A8はRequirementではありません。採用までは評価待ちです。
- 机上評価はコードを実行しておらず、README・実装記録の読解に基づきます。
- 人の認知負荷が減ることは、このIssueでは測定しません。
- 2026-09-21の受理が不適切だったとは主張しません。受理はexplicit human Acceptance decisionと独立レビューを経ており、このIssueは次の承認をより確かめやすくするためのものです。
