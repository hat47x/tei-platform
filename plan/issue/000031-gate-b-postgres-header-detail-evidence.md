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
clean repository revision
        +
Canonical Workspace
        ↓ Validation
real PostgreSQL engine
        +
tool-owned ephemeral fixture
        ↓
header-detail runtime Evidence
        ↓
same clean repository revisionを再確認
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

`TEI_TEST_POSTGRES_DSN`には、Evidence専用に作成した**空のPostgreSQL Database**を指定します。

```bash
go run ./tools/reference-postgres-header-detail-runtime
```

DSN、credential、runtime parameter value、接続先の実Database名をRepositoryや通常Evidenceへ保存しません。

## source revisionのfail-closed境界

Gate Bへ採用するProvider Evidenceは、後から実際に実行したsourceへ戻れる必要があります。

そのためDatabaseへ接続する前にrepository revisionとworking treeを確認し、dirtyなら停止します。

Provider実行とtool-owned fixture cleanupの後にも再確認し、次のいずれかが起きていれば成功Evidenceとして採用しません。

- 実行中にHEADが変化した
- 実行後にtracked / untracked差分が残った

成功JSONには少なくとも次を含めます。

```text
repositoryRevision
workingTreeClean = true
reproducibleRevision = true
postRunRepository.checked = true
postRunRepository.repositoryRevision = repositoryRevision
postRunRepository.revisionStable = true
postRunRepository.workingTreeClean = true
```

source状態の観測規則は`tools/internal/repositoryevidence`へ置き、#310 deterministic Repository Evidenceと#255 generated artifact Evidenceでも同じ規則を利用します。Provider固有のDB操作やSemantic意味はこの共有packageへ持ち込みません。

## Database側のfail-closed安全境界

PR #294 / #300 / #306で、Workspace Validation、既存fixture保護、tool-owned cleanupの基本境界を成立させました。

PR #324では、「`public.order_lines`さえ存在しなければ既存業務DBでもfixtureを作成できる」という残存リスクを閉じ、**専用空Databaseであることをfail closedに確認する境界**を追加しました。

現在の実行順は次です。

1. clean repository revisionを確認する
2. WorkspaceをLoadする
3. 通常`validation.Validate`を実行する
4. Validation ErrorがあればDB接続前に停止する
5. Validation成功後にのみPostgreSQLへ接続する
6. `current_database()`で接続先Database identityを取得する
7. `postgres` / `template0` / `template1`なら停止する
8. system / temporary領域を除くuser relation件数を確認する
9. user relationが1件でも存在すれば停止する
10. `public.order_lines`の既存有無を再確認し、存在すれば停止する
11. すべての安全条件を満たした場合だけtool-owned fixtureをCREATEする
12. tool自身がCREATEしたfixtureだけをcleanupする
13. repository HEAD / working treeを再確認する
14. 同じclean revisionへ固定できた場合だけJSON Evidenceを出力する

### 既定Databaseを拒否する

次のDatabaseは、内容が空に見えてもEvidence fixture実行先として利用しません。

```text
postgres
template0
template1
```

Database名からproduction / development等の用途を推測して許可する規則も持ちません。

### 既存user relationを拒否する

`pg_catalog` / `information_schema` / PostgreSQL内部・temporary領域を除き、次のrelationが一つでも存在すれば停止します。

- table / partitioned table
- view / materialized view
- foreign table
- sequence

既存relationの用途や安全性を推測せず、削除して空Databaseへ転用する処理も提供しません。

この検査は「Evidence専用Databaseである」という運用上の意図を完全に証明するものではありませんが、少なくとも既定Databaseや既存user assetを含むDatabaseへfixtureを書き込む経路をfail closedで閉じます。

### fixture objectを再確認する

空Database検査後も、native SQLが利用する`public.order_lines`の既存有無を明示確認します。

```sql
SELECT to_regclass('public.order_lines')::text
```

既存objectがあれば停止します。

**既存relationやtableを削除して続行するoverrideはありません。** 別の専用空Databaseを使用します。

## 安全判定をEvidenceへ残す

fixture作成前の安全guardは、実装されているだけでなく、そのrunで満たした非機密な観測結果を`fixtureDatabaseSafety`としてEvidence JSONへ残します。

期待値:

```json
{
  "guardPassed": true,
  "databaseIdentityVerified": true,
  "databaseIdentityPersisted": false,
  "nonDefaultDatabaseVerified": true,
  "userRelationsBeforeFixture": 0,
  "fixtureObjectAbsentBeforeCreate": true
}
```

これは次を意味します。

- `current_database()`によるidentity確認に成功した
- 実Database名自体はEvidenceへ保存していない
- PostgreSQL既定Databaseではなかった
- fixture作成前のuser relation数が0件だった
- `public.order_lines`がfixture作成前に存在しなかった
- それらをすべて確認した後だけfixture作成へ進んだ

`guardPassed = true`を「組織運用上、このDatabaseがEvidence専用であることの完全証明」へ格上げしません。toolがそのrunで観測できる安全条件を満たしたことだけを示します。

## 確認するEvidence

同じrunで少なくとも次を確認します。

- cleanかつ実行中に不変なrepository revisionへEvidenceを固定する
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
- `reproducibleRevision = true`を保持する
- `fixtureDatabaseSafety.guardPassed = true`を保持する
- `fixtureDatabaseSafety.nonDefaultDatabaseVerified = true`を保持する
- `fixtureDatabaseSafety.userRelationsBeforeFixture = 0`を保持する
- `fixtureDatabaseSafety.fixtureObjectAbsentBeforeCreate = true`を保持する
- `fixtureDatabaseSafety.databaseIdentityPersisted = false`を保持する
- 接続先Database名 / DSN / credentialを通常Evidenceへ保存しない

## 完了条件

- [x] Workspace Validation Error時にDB接続前に停止する
- [x] PostgreSQL既定Databaseをfixture先として拒否する（PR #324）
- [x] 既存user relationがあるDatabaseをfixture先として拒否する（PR #324）
- [x] `public.order_lines`既存時に停止する
- [x] tool-owned fixtureだけをcleanupする
- [x] 安全guardの非機密な判定結果をEvidence JSONへ残し、Database名 / DSNを保存しない
- [x] Provider Evidence開始前にclean repository revisionを要求する境界を実装した（PR #336）
- [x] Provider Evidence終了後にHEAD / working treeを再確認する境界を実装した（PR #336）
- [x] #310 / #255 / #308で共通利用するtool-only repository source観測packageを用意した（PR #336）
- [ ] Canonical Workspace Validationが成功するclean revisionで実行する
- [ ] Evidence専用の空PostgreSQL Databaseでtoolを実行する
- [ ] `reproducibleRevision = true`のJSON Evidenceを取得する
- [ ] `fixtureDatabaseSafety`が期待する安全条件を示すことを実runで確認する
- [ ] 2件取得を確認する
- [ ] 0件を正常な空collectionとして確認する
- [ ] physical table欠落をExecution Errorとして確認する
- [ ] Semantic mappingとsupport column非露出を確認する
- [ ] parameter value / credential / DSN / Database名が通常Evidenceへ残っていないことを確認する
- [ ] Evidence natureが`real-postgres-ephemeral-fixture`であることを確認する
- [ ] 結果を`ISSUE-000029` / #273のGate B監査へ対応付ける

## 非目標

- keyless Relation Storage境界の再設計
- header-detail PatternのUI layout設計
- production既存Databaseをfixtureで置換すること
- 既存relationを削除してEvidence用Databaseへ転用すること
- production PostgreSQL schemaの観測Evidence
- actual Oracle Evidenceの代替
- physical schemaからSemantic Relationを逆生成すること
- PostgreSQL固有仕様をSemantic modelへ逆流させること
- source revision規則をCanonical modelへ追加すること
- GitHub Actionsの再有効化

## 関連

- GitHub Issue #69 / #244 / #255 / #273 / #308 / #310
- PR #294 / #300 / #306 / #309 / #324 / #335 / #336
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/issue/000024-keyless-relation-physical-storage.md`
- `tools/internal/repositoryevidence/`
- `tools/reference-postgres-header-detail-runtime/README.md`
- `tools/reference-postgres-header-detail-runtime/main.go`
- `tools/reference-postgres-header-detail-runtime/safety.go`
