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

このとき受理の直前にownerへ示されたのは、TEIのProjectionそのものではなく、Claude Code session上の会話でした。ISSUE-000040のwitness記録には「直前に提示した内容: 独立レビュー完了（review-clear-with-conditions、必須修正なし）、残る停止条件はexplicit human Acceptance decisionだけであること」とあります。Previewは機械可読なJSONとして全stage PASSしていましたが、人が読む形へ写したのはAIの要約でした。

つまり現在の承認は、「PreviewのJSONが正しい」ことと「AIがそれを正しく要約した」ことの二つに依存しています。後者はTEIの境界の外にあります。

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
| A2 影響 | 一部満たす | exact identityに基づく`impact`辺を返す。Driver scenarioとTarget Projectionへの影響は評価せず、`DRIVER_SCENARIO_NOT_RUN` / `TARGET_PROJECTION_NOT_EVALUATED`としてunresolvedへ明示している。辿れない範囲を隠していない点は条件に合う |
| A3 例外・診断 | 一部満たす | Validation diagnosticは`validate` stageに出る。業務上の例外（どの業務ケースで規則が効くか）を示す項目は無い |
| A4 未解決点 | 満たす（機械可読） | `unresolved`。例: `ENVIRONMENT_BINDING_NOT_AUTHORED`、`RUNTIME_AUTHORITY_NOT_GRANTED` |
| A5 権限の変更 | 満たさない | 権限の変化を独立に示す項目が無い。Environment bindingでは権限が付かないことを`RUNTIME_AUTHORITY_NOT_GRANTED`で示すが、一般の変更では`changedCollections`から間接的に読むしかない |
| A6 native codeへ残る箇所 | 満たさない | 項目が無い。現在の5 operationはnative codeを変えないが、それを承認者へ示す項目も無い |
| A7 逆追跡 | 一部満たす | `sourceEdits`（前後のfingerprint）、subject ID、候補のEvidenceの解決結果がある。ただし人向けの要約が無いため、「要約が正本と一致するか」を確かめる手段も無い |
| A8 承認単位の大きさ | 満たす | `maxSubjectChanges`（上限8）を`acceptance-policy` stageで検査する。whole-model再生成を標準経路にしない |

**机上評価の結論:** Previewは、承認に必要な情報の多くを機械可読な形で正本から出しています（A1 / A4 / A8）。一方、人が読む面が無く、権限の変化とnative codeへ残る箇所を示す項目もありません（A5 / A6）。このため実際の受理では、AIの会話による要約が人とPreviewの間に入りました。

## 必要能力

1. Preview結果を、条件A1〜A8の順に人が読める形へ写すProjection。JSONの各項目へ戻るlinkまたはID参照を持つ。これはRQ-010のReview Surfaceの最小形であり、Studio UIの完成を待たない。
2. A5 / A6を示す項目。変わらない場合も「変わらない」と示す。
3. 承認記録（`.tei/decisions/`）に、承認時に満たしていなかった条件を残すこと。

## 具体例

`repair_intake`のPreviewをA1〜A8の順に写すと、たとえば次のようになります。

```text
A1 意味差分    semantic 4件（namespace / data / attribute / relation）、application 2件
A2 影響        15本。Driver scenario・Target Projectionは未評価（unresolved）
A3 例外・診断  validation error 0件（全stage PASS）。業務上の例外を示す項目は無い
A4 未解決点    ENVIRONMENT_BINDING_NOT_AUTHORED / DRIVER_SCENARIO_NOT_RUN / TARGET_PROJECTION_NOT_EVALUATED
A5 権限の変更  項目なし（満たさない）
A6 native code 項目なし（満たさない）
A7 逆追跡      作られるfile 3件と各fingerprint、candidate fingerprint 8a936cf9…
A8 大きさ      変更subject 6件 / 上限8
```

A5とA6が「項目なし」と明示されていれば、承認者は「権限は変わらないと示されたのではなく、示す仕組みが無い」と分かったうえで判断できます。

## 完了条件

- [ ] 条件A1〜A8を、owner reviewで採用・修正・却下のいずれかに判定する。採用した条件はRQ-028の解釈として、このIssueまたはADRへ残す
- [ ] `repair_intake`以外のPreview（例: `add-relation`、`add-environment-binding`）でも机上評価を行い、評価が変わる条件を記録する
- [ ] Preview結果をA1〜A8の順に写す最小Projectionを、Reference toolとして作る（source変更を伴うため、`tools/repository-health-check.sh`のfull PASSが統合条件）
- [ ] そのProjectionだけを見て、人が一件の変更を承認または却下した記録を残す（AIの要約を介さない）。承認者が気づいた不足を記録する
- [ ] A5 / A6を示す項目を追加するか、追加しない理由と、承認時にそれを明示する方法を決める

## 完了後に判断できること

- RQ-028の「承認に足る」をGap AuditでPartialから引き上げてよいか
- Studio UI（ISSUE-000046）が最初に表示すべき項目と順序
- AIが承認の前段に入る場合、AIの要約をどこまで信用でき、どこで正本へ戻すべきか

## 非主張範囲

- 条件A1〜A8はRequirementではありません。採用までは評価待ちです。
- 机上評価はコードを実行しておらず、README・実装記録の読解に基づきます。
- 人の認知負荷が減ることは、このIssueでは測定しません。
- 2026-09-21の受理が不適切だったとは主張しません。受理はexplicit human Acceptance decisionと独立レビューを経ており、このIssueは次の承認をより確かめやすくするためのものです。
