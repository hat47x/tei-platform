# ISSUE-000064: PostgreSQL fixture削除をDB側ownership markerで保護する

- 状態: Done — Gate B Ready（R = `e5231839`）のactual #387 final runで確認済み（2026-09-19）
- GitHub Issue: #427
- 親: `ISSUE-000031` / #308
- Gate B最終収束: `ISSUE-000037` / #387
- Gate B監査: `ISSUE-000029` / #273

## 目的

#308 Evidence toolが自分で作成した`public.order_lines`だけを削除し、実行途中に同名objectが別主体のtableへ置き換わった疑いがある場合は、推測して削除せずfail closedで停止します。

Evidence専用空Databaseという前提は維持しつつ、cleanup可否をGo側の`fixtureOwned` booleanだけに依存させません。

## 現行境界

### fixture作成

cryptographically randomなownership markerを生成し、CREATEとmarker付与を一つのtransactionで行います。

```text
BEGIN
  CREATE TABLE public.order_lines (...)
  COMMENT ON TABLE public.order_lines IS '<tool ownership marker>'
COMMIT
```

markerなしのtool-created tableを正常作成結果として公開しません。

marker値自体は通常Evidence / stderr / Repositoryへ保存しません。

### cleanup

明示cleanupとdeferred cleanupは同じ保護経路を利用します。

```text
BEGIN
  LOCK TABLE public.order_lines IN ACCESS EXCLUSIVE MODE
  read DB-side ownership marker
  require exact match
  DROP TABLE public.order_lines
COMMIT
```

lockをmarker確認より先に取得し、照合からDROPまでの間に同名objectを置換できない状態にします。

次の場合はDROPしません。

- table不在
- lock取得不能
- marker観測不能
- markerなし
- marker mismatch
- transaction開始 / commit不能

cleanup不能でtool fixtureが残る可能性は、別主体objectを誤削除することより優先します。次回のempty-DB guardが残存objectをfail closedで検出します。

## process-local stateとの分離

```text
fixtureOwned boolean
  = cleanupを試みる必要があるか

DB-side ownership marker
  = その同名objectを削除してよいか
```

`fixtureOwned=true`だけではDROPを許可しません。

## #308の他の安全条件

ownership markerは次を置き換えません。

- clean Repository source
- Workspace Validation
- default Database拒否
- fixture前user relation 0件
- fixture前Event Trigger 0件
- fixture object事前不在
- credential-safe診断
- cleanup後user relation 0件
- cleanup後Event Trigger 0件
- cleanup後fixture object不在
- post-run同一clean revision

## 成立済み

- [x] DB-side ownership marker生成（PR #428）
- [x] CREATE + marker付与を同一transaction化
- [x] lock + marker exact match後だけDROP
- [x] deferred cleanupも同じ経路へ統一
- [x] name-only cleanup helperを排除
- [x] marker pure test
- [x] fake SQL driverでtransaction順序とmismatch時no-DROPを固定
- [x] 隔離Go module test
- [x] #308 / final runner tool READMEへ同期
- [x] `plan/GATE_B_EVIDENCE_RUNBOOK.md`へ同期
- [x] `plan/GATE_B_READINESS_AUDIT.md`へ同期

## 残作業

- [x] 実TEI checkoutの通常Go testで再確認する（2026-09-21にmain 9f4fcd5で`./tools/reference-postgres-header-detail-runtime`がok）
- [x] #308 actual PostgreSQL Evidenceで次を確認する（actual #308 childで`fixtureOwnedByTool = true` / `fixtureCleanupSafety.verified = true` / `userRelationsAfterCleanup = 0` / `eventTriggersAfterCleanup = 0` / `fixtureObjectAbsentAfterCleanup = true`。marker取得とDROPの内部手順は隔離testとfake SQL driver testが固定し、actual Evidenceは結果条件で確認した）
  - CREATE + COMMENT transaction成功
  - ACCESS EXCLUSIVE lock後のmarker取得
  - marker exact match後のDROP
  - cleanup後user relation 0 / Event Trigger 0 / fixture不在
- [x] #387 same-R final convergenceで同じ#308 Evidenceを取得する

actual PostgreSQL runは終了し、Gate B Ready時点で本Issueを完了としました。

## 非目標

- Evidence専用空Database guardの置換
- production Databaseでfixtureを許可すること
- ownership markerのCanonical化
- ownership marker値のEvidence保存
- 他主体へ協調lock運用を要求すること
- GitHub Actions再有効化
