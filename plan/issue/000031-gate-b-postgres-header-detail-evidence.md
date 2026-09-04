# ISSUE-000031: Gate Bでheader-detailのreal PostgreSQL engine Evidenceを取得する

- 状態: Ready
- GitHub Issue: #308
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 前提Issue: `ISSUE-000024` / GitHub Issue #244（Completed）
- Gate B監査: `ISSUE-000029` / GitHub Issue #273
- 関連RQ: RQ-009 / RQ-012 / RQ-014

## 目的

`Order -> OrderLine[]` header-detailについて、すでに成立したSemantic Relation、Database-owned Relation Storage、Contract Query realization、Driver runtimeが、**実PostgreSQL engine上でも同じ意味境界を保ったまま動作すること**をEvidenceとして確認します。

このIssueはRelation Storageやheader-detail traversalを再設計するためのものではありません。設計・実装境界は`ISSUE-000024` / #244でCompletedです。本Issueは、Gate Bに必要な**real-provider ephemeral fixture Evidence**だけを扱います。

## Evidenceの性質

```text
Canonical Workspace
        ↓ Validation
real PostgreSQL engine
        +
tool-owned ephemeral fixture
        ↓
header-detail runtime Evidence
```

このEvidenceは実PostgreSQL provider / driver / native SQL executionを通しますが、production既存Databaseの物理事実を観測するものではありません。

したがって、次の代替にはしません。

- `ISSUE-000022` / #214のactual external Oracle asset Evidence
- `ISSUE-000025` / #255のgenerated Reference artifact再現Evidence
- `ISSUE-000032` / #310のdeterministic Repository Evidence

## 正規実行入口

具体的なコマンド、順序、停止条件は`plan/GATE_B_EVIDENCE_RUNBOOK.md`を正とします。

実行tool:

```text
tools/reference-postgres-header-detail-runtime
```

専用の空PostgreSQL Databaseを`TEI_TEST_POSTGRES_DSN`で指定し、次を実行します。

```bash
go run ./tools/reference-postgres-header-detail-runtime
```

DSN、credential、runtime parameter valueをRepositoryや通常Evidenceへ保存しません。

## fail-closed安全境界

PR #294 / #300 / #306で、少なくとも次が成立しています。

1. WorkspaceをLoadする
2. 通常`validation.Validate`を実行する
3. Validation ErrorがあればDB接続前に停止する
4. Validation成功後にのみPostgreSQLへ接続する
5. `public.order_lines`が既に存在すれば停止する
6. 既存tableをDROPしてfixtureへ置換しない
7. tool自身がCREATEしたfixtureだけをcleanupする

誤ったDSNが既存資産を指していても、既存`public.order_lines`を削除して続行するoverrideは設けません。

## 確認するEvidence

同じrunで少なくとも次を確認します。

- `order_id=1001`でOrderLineを2件取得する
- `order_id=9999`で0件を正常な空collectionとして扱う
- tool-owned fixture table欠落をExecution Errorとして扱う
- Environment identityを保持する
- Database instance identityを保持する
- executor Capability identityを保持する
- Contract Query target identityを保持する
- raw resultからSemantic OrderLineへの明示mappingを利用する
- Relation Storage support column `order_id`をOrderLine Semantic attributeへ漏らさない
- parameter名だけをEvidenceへ残し、runtime parameter valueを保存しない
- `evidenceNature = real-postgres-ephemeral-fixture`を保持する

## 完了条件

- [ ] 対象repository revisionを記録する
- [ ] Canonical Workspace Validationが成功する
- [ ] Evidence専用の空PostgreSQL Databaseでtoolを実行する
- [ ] 2件取得を確認する
- [ ] 0件を正常な空collectionとして確認する
- [ ] physical table欠落をExecution Errorとして確認する
- [ ] Semantic mappingとsupport column非露出を確認する
- [ ] parameter value / credentialが通常Evidenceへ残っていないことを確認する
- [ ] Evidence natureが`real-postgres-ephemeral-fixture`であることを確認する
- [ ] 結果を`ISSUE-000029` / #273のGate B監査へ対応付ける

## 非目標

- keyless Relation Storage境界の再設計
- header-detail PatternのUI layout設計
- production既存Databaseをfixtureで置換すること
- production PostgreSQL schemaの観測Evidence
- actual Oracle Evidenceの代替
- physical schemaからSemantic Relationを逆生成すること
- PostgreSQL固有仕様をSemantic modelへ逆流させること
- GitHub Actionsの再有効化

## 関連

- GitHub Issue #69 / #244 / #255 / #273 / #308 / #310
- PR #294 / #300 / #306 / #309
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/issue/000024-keyless-relation-physical-storage.md`
- `tools/reference-postgres-header-detail-runtime/README.md`
- `tools/reference-postgres-header-detail-runtime/main.go`
