# ISSUE-000074: 一度答えた確認と退けた提案を記録し、理由なく繰り返さない

- 状態: Proposed — 評価待ち。現状の欠落と最小の記録形の案までを記録した段階で、実装は始めていない
- 提案日: 2026-09-30
- 性格: Requirementの最小Reference / 既存受入境界の拡張
- 主担当要求: RQ-016（主）/ RQ-017 / RQ-027
- 関連Issue: [ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)（受理境界と`reject`）、[ISSUE-000046](000046-studio-shared-acceptance-authoring-loop.md)（Studioで確認を出す面）、[ISSUE-000047](000047-knowledge-first-semantic-exploration.md) / [ISSUE-000051](000051-derived-cognitive-retrieval-evaluation.md)（提案を作る側）、[ISSUE-000008](000008-editor-and-ai-followup-note.md)（Done。再質問抑止を後続AI支援の一項目として挙げたが、記録の担当は割り当てていない）
- 設計メモ: [Derived Cognitive Requirement Draft](../DERIVED_COGNITIVE_REQUIREMENT_DRAFT_2026-09.md) DCR-18（「前にも見た」という慣れではなく、明示的なDecision / suppression historyで実装する）
- 価値との対応: [Product Value](../../product/value/TEI_PRODUCT_VALUE.md)「個人 5. 同じことを二度しなくてよい」、「価値の確かめ方 — 個人: 一度答えた確認が、理由なく繰り返されないこと」、原則P13
- Gate impact: None

## この文書の役割

RQ-016は、AI / Editorの確認提案について「一度回答した質問を理由なく繰り返さない」「不要な提案を無視できる」「提案カテゴリを抑制できる」「過去の選択と理由を後から追跡できる」「選択を後から変更できる」ことを求めています。Gap AuditではRQ-016は**Design Only**で、これを担う記録も、繰り返しを検査するtestもありません。

このIssueは、確認と退けた提案を記録する最小の形を決め、同じ確認が理由なく再び出ないことをReferenceで示します。

## 背景

2026-09-30時点で、受入境界（`tools/reference-shared-acceptance`）は受理した決定を`.tei/decisions/`へ残し、会話履歴なしで読めるようにしています（RQ-027）。一方、次の二つは記録されません。

- **退けた提案。** `reject`は「sourceを変えない」ことを確かめますが、何も記録しません。RQ-027が機械到達を求めるのはaccepted Decisionであるため、という理由がtoolのREADMEに書かれています。
- **答えた確認。** 「顧客マスタはread-onlyか」「履歴を残すか」のような確認とその答えを記録する形が、Repositoryのどこにもありません。

このため、AIや将来のStudioが同じ論点について確認を出したとき、「前に答えた」「前に退けた」ことを機械的に知る手段がありません。人は同じ質問に何度も答え、同じ提案を何度も退けることになります。

## 既存文書との所有の切り分け

確認と退けた提案の履歴を所有する文書は、このIssueの起票時点で他にありません。近い記述との関係を次のとおり分けます。

- [ISSUE-000008](000008-editor-and-ai-followup-note.md)は、既回答事項の再質問抑止と提案の抑制を後続AI支援の候補として挙げただけで、記録の担当を決めていません。このIssueが記録の形と再確認の規則を担当します。
- [ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)は受理境界と`reject`のsource不変を担当します。このIssueは、そこへ「退けた事実の記録」を足すかどうかだけを決めます。000027のRuntime authorityとhybrid routingの項（17〜23）は、確認履歴とは別の論点です。
- 000027と[ISSUE-000047](000047-knowledge-first-semantic-exploration.md)には、`known false-positive suppression`や、受理・却下したpatternの局所ranking hintが、将来の学習・探索順位の比較候補として書かれています。これらは利用者が見て解除できる明示履歴の代わりにはなりません（DCR-18）。両者の関係は、このIssueでは決めません。
- [ISSUE-000046](000046-studio-shared-acceptance-authoring-loop.md)と000047は、確認を出す面と提案を作る面の実装を持ちます。記録の形はこのIssueで決めます。

## 防ぎたい失敗

- **同じ質問の繰り返し。** 別のsessionや別のAIが、既に答えた確認を再び出す。人は確認を読まずに答えるようになり、確認そのものが形だけになる。
- **退けた提案の再来。** 退けた提案が、理由の変化なく再び提案される。
- **黙った抑制。** 逆に、前提が変わったのに「前に答えた」ことを理由に確認を出さない。答えた時点の前提と今の前提が違うことに誰も気づかない。
- **記録が正本になる。** 確認の記録を、業務上の意味やauthorityの根拠として扱う。

## 守る原則

- 確認の答えが業務上の意味を変える場合、その意味は正本へ受理する。確認の記録そのものはauthorityではなく、消しても意味は変わらない（`.tei/decisions/`と同じ扱い）。`.tei/`がmodel revisionに入らないのは、`tei.workspace.yaml`の`sources`へ宣言していないためで、`.tei/confirmations/`も`sources`へ宣言しない限りrevisionを動かさない。
- 再び確認するかどうかは、慣れではなく、**答えた時点の前提から何が変わったか**で決める（DCR-18）。
- 抑制は、利用者が見て、解除できる（RQ-016、原則P13）。

## 必要能力と最小の記録形（案）

確認記録（仮称 confirmation record）を、`.tei/confirmations/`に置く案です。`.tei/decisions/`と同じく内容のSHA-256でfile名を決め、後から書き換えません。現在の状態（有効な答え、抑制中のカテゴリ）は、同じidentityの記録を並べて導出し、解除や答えの変更は新しい記録として足します。identityをfile名にして上書きすると、過去の選択と理由を追えなくなるためです。

| 項目 | 意味 |
| --- | --- |
| identity | 確認の同一性。確認の種類（例: `master-data-write`、`history-retention`）と対象のCanonical IDから決まる。文面の言い回しからは決めない |
| basis | 答えた時点の前提。対象subjectと関連subjectのfingerprint、model revision |
| answer | 答え（例: `read-only`、`not-needed`）と理由 |
| outcome | `answered` / `rejected-proposal` / `category-suppressed` |
| decidedBy | 答えた主体（人 / AIの区別を含む）。provenanceであり、authorityではない |
| linkedDecision | 答えが正本の変更につながった場合、その`.tei/decisions/`の記録（`RecordPath`）。`rejected-proposal`の同一性は、退けたcandidateの`candidateFingerprint`と揃える案で、`reject`に記録を残すかどうかの判断（完了条件の1項目）に従う |

再確認の規則（案）:

- identityが同じで、basisのfingerprintが今も一致する → 確認を出さない。出さなかったことと、根拠にした記録を示す
- basisが変わった → 確認を出す。前回の答えと、何が変わったかを添える
- `category-suppressed` → そのカテゴリの確認を出さない。抑制の一覧を利用者が見て解除できる
- 記録が読めない・食い違う → 黙って抑制せず、確認を出す。`decisions`が読めない記録をerrorにするのと同じく、疑わしいときは確認を出す側へ倒す

## 具体例

`order_entry`で、AIが「顧客マスタ`master_data.customer`へ書き込む操作を足しますか」と確認し、人が「足さない。顧客マスタはread-only」と答えたとします。

```text
1回目  確認を出す → 答え: read-only（理由: 顧客マスタは別システムが所有）
       記録: identity = master-data-write / master_data.customer
             basis    = master_data.customerとDatabase Contractのfingerprint
2回目  別sessionのAIが同じ確認を作る → identityとbasisが一致 → 出さない
3回目  Database Contractが変わり、read-writeを提供するようになった
       → basisが不一致 → 確認を出す。「前回はread-onlyとして答えました。Contractが変わりました」
```

## 完了条件

- [ ] 記録の形と再確認の規則をowner reviewで確定する（`reject`に記録を残すかどうかの判断を含む。残す場合も、authorityにしない理由を明記する）
- [ ] `basis`のfingerprintをsubject単位で取れるか（現行の`sourceindex.Fingerprint`はfile内容のfingerprint）を確かめ、取れなければfile単位から始める
- [ ] 受入境界またはその隣のReference toolに確認記録の読み書きを実装する。`reference/workspace`のbaselineへは、決定記録と同じguard（`-allow-reference-baseline`なしでは書き込みを拒む）を適用する。`.tei/decisions/`の記録は現在Gitで追跡されている（`.gitignore`が除外するのは`.tei/cache`と`.tei/index`だけ）ため、`.tei/confirmations/`も同様に追跡するかを決める。次をtestで固定する: basis一致なら再確認しない / basis変化で再確認する / 抑制は解除できる / 記録を消しても意味が変わらない / 読めない記録で黙って抑制しない
- [ ] `tools/repository-health-check.sh`のfull PASSを、exact headで記録してから統合する
- [ ] 答えの変更や解除が新しい記録として足され、前の記録が残ることをtestで固定する（RQ-016「選択を後から変更できる」）
- [ ] Gap AuditでRQ-016の評価を見直す

## 完了後に判断できること

- Studio（ISSUE-000046）とAI（ISSUE-000047）が、確認を出す前に記録を参照する共通の入口を持てるか
- 確認の種類（identityの語彙）をどこまで標準化すべきか

## 非主張範囲

- AIが適切な確認を作れることは、このIssueでは示しません。
- 確認の答えを、業務上の意味やAuthorizationの根拠にしません。
- 確認の種類の網羅的な一覧は作りません。Referenceで必要になったものだけを置きます。
