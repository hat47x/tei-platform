# ISSUE-000056: Issue番号の重複8組と状態語彙の逸脱を解消し、再発を防ぐ

- 状態: Proposed
- 提案日: 2026-09-19
- 性格: Planning governance / traceability
- 主担当要求: RQ-023
- 関連: AGENTS.md「文書ルール」 / ISSUE-000055 / ISSUE-000059

## 何に困っているのか

[`AGENTS.md`](../../AGENTS.md)は、Issueを6桁連番、状態を`Proposed / Ready / In Progress / Blocked / Done / Superseded`の6値と定めています。しかし現在の`plan/issue/`には、次の逸脱があります。

**番号の重複が8組（16file）あります。** 同じ番号が別の作業を指しているため、`ISSUE-000042`のような参照から作業を一意に特定できません。

| 番号 | file（パス参照数 / 追加日） |
| --- | --- |
| 000010 | `application-pattern-validation-integration`（1 / 09-02）、`command-application-pattern`（0 / 09-02） |
| 000011 | `order-entry-browse-reference`（1 / 09-02）、`order-line-semantic-reference`（3 / 09-02） |
| 000035 | `gate-b-actual-oracle-blocker-reaudit`（1 / 09-04）、`gate-b-external-provider-scope-audit`（5 / 09-04） |
| 000038 | `business-lifecycle-semantics`（9 / 09-05）、`gate-b-repository-source-observation-stability`（1 / 09-05） |
| 000039 | `business-change-rehearsal`（11 / 09-05）、`gate-b-final-go-environment-isolation`（1 / 09-05） |
| 000040 | `gate-b-child-go-environment-isolation`（1 / 09-05）、`semantic-component-composition`（11 / 09-05） |
| 000041 | `gate-b-postgres-fixture-ownership-marker`（1 / 09-05）、`partition-schema-drift-semantics`（4 / 09-05） |
| 000042 | `business-procedure-cross-view-reference`（19 / 09-06）、`gate-b-final-goenv-launch-guard`（1 / 09-05） |

パス参照数は、そのfile名（拡張子なし）を本文に含む他fileの数です。`ISSUE-NNNNNN`という書き方での参照は、重複のため片方に割り当てられず、ここには含まれません。

**状態語彙の逸脱が21件あります。** `完了` 4件、`Completed` 3件、`着手可能` 2件、`Ready to verify` 5件、`Resolved` 1件が6値の外にあり、`- 状態:`行そのものがないIssueが6件（000001 / 000002 / 000042 cross-view / 000048 / 000049 / 000050）あります。

## 防ぎたい失敗

- ADR・Runbook・他IssueからのIssue参照が、二つの作業のどちらを指すのか判断できなくなること。
- 状態語彙の揺れにより、機械的な集計（進行中の作業、未着手の作業）ができなくなること。
- 番号の割り当てが、複数の作業者（人・AI）が並行して新しいIssueを起票するたびに衝突すること。

## 現在分かっていること

- 2026-09-09にローカルbranch `docs/business-procedure-design-consolidation-20260909`（commit `88d074e0`）で、Gate B系の参照の少ない側を000048〜000053へ改番し、状態語彙も正規化する是正が作られました。これは未pushのまま、その後mainで000048〜000054が別の作業に使われたため、**改番先がそのままでは使えません**（`ISSUE-000059`で扱います）。
- 重複はすべて2026-09-02〜06に発生しており、その後の起票は連番を保てています（000043以降に新しい重複はありません）。

## 決めること

1. 各組でどちらが番号を保持するか。参照数の多い側（表のパス参照数）を残す、を出発点にできます。ただし、番号が文書内で`ISSUE-NNNNNN`形式でしか参照されていない場合は、参照が実際にどちらを指すかを文脈から確認する必要があります。
2. 改番する側の新しい番号。並行して起票が続くため、改番は起票と衝突しない手順（最新mainで最大番号を確認してから割り当てる）で行います。
3. 旧番号から新番号への対応表を、`plan/issue/README.md`へ残すか。
4. 状態語彙を6値へ揃える際の対応（`完了 / Completed` → `Done`、`着手可能` → `Ready`、`Ready to verify` → `In Progress`または`Ready`のどちらか、`Resolved` → `Done`）。`Ready to verify`は「実装は済み、実環境での確認待ち」という意味を持つため、単純な置換で情報を失わないか確認が必要です。
5. 再発防止として、Issue番号の一意性と状態語彙を機械的に検査する仕組み（`tools/repository-health-check.sh`への追加など）を置くか。

## 完了条件

- [ ] 8組の重複を解消し、旧番号→新番号の対応を残す
- [ ] 改番に伴い、`ISSUE-NNNNNN`形式とfile path形式の両方の参照を追従させる
- [ ] 状態語彙の逸脱21件を6値へ揃え、`- 状態:`行がない6件へ状態行を加える
- [ ] `plan/issue/README.md`の一覧・対応表を更新する
- [ ] 一意性と状態語彙の検査を自動化するかを決め、採用するなら`ISSUE-000055`のhealth checkへ組み込む
- [ ] Gate判定・Canonical採用・Evidence主張を変更していないことを確認する

## 非目標

- Issueの中身（背景・完了条件）の書き換え
- Gate / Evidence / Product主張の変更
- ADR番号の変更（ADRの番号は重複していません）

## 関連

- `plan/issue/README.md`
- `plan/issue/000059-unpushed-legacy-consolidation-branches.md`
