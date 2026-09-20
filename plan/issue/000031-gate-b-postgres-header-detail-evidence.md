# ISSUE-000031: Gate Bでheader-detailのreal PostgreSQL engine Evidenceを取得する

- 状態: Done — Gate B Ready（R = `e5231839`）のactual #387 final runで確認済み（2026-09-19）
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
fixture cleanup / missing-table Error
        ↓
Database empty-state restoration check
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

## Repository sourceをEvidenceへ固定する

PR #336で、#310 deterministic Repository Evidence、#255 generated artifact Evidence、#308 real PostgreSQL Provider Evidenceのsource revision観測を`tools/internal/repositoryevidence`へ収束しました。

この共有packageが扱うのはtool実行時のRepository状態だけです。Canonical、Driver、Provider固有の意味やDB操作は持ち込みません。

#308では、Workspace LoadやPostgreSQL接続より前に次を確認します。

1. `git rev-parse HEAD`でrepository revisionを取得する
2. `git status --porcelain`でworking tree状態を取得する
3. dirty working treeならDB接続前に停止する

Provider runtime、tool-owned fixture cleanup、Database復元確認が完了した後にも次を再確認します。

1. `git rev-parse HEAD`を再取得する
2. 実行前HEADと完全一致することを確認する
3. `git status --porcelain`が空であることを確認する
4. 同じclean revisionへ固定できないrunは成功Evidenceとして拒否する

成功JSONには次を残します。

- `repositoryRevision`
- `workingTreeClean`
- `reproducibleRevision`
- `postRunRepository.checked`
- 実行後`repositoryRevision`
- `postRunRepository.revisionStable`
- 実行後`workingTreeClean`

`reproducibleRevision = true`には、開始時clean、post-run確認済み、実行前後のHEAD同一、終了時cleanをすべて要求します。

`RevisionStable = true`というbooleanだけを信用せず、post-runのrevision文字列自体も開始時revisionと一致する必要があります。

## fail-closed安全境界

PR #294 / #300 / #306で、Workspace Validation、既存fixture保護、tool-owned cleanupの基本境界を成立させました。

PR #324では、「`public.order_lines`さえ存在しなければ既存業務DBでもfixtureを作成できる」という残存リスクを閉じ、専用空Databaseとして扱える最小条件をfail closedに確認する境界を追加しました。

PR #365では、user relationが0件でもDDLへ介入し得るEvent TriggerがDatabaseに残る可能性を閉じました。fixtureの`CREATE TABLE` / `DROP TABLE`より前にEvent Trigger件数も確認し、1件でも存在すれば停止します。disabledなEvent Triggerも「存在しない」とは扱いません。既存Event Triggerを削除・無効化して実行を続ける処理は提供しません。

PR #414では、`TEI_TEST_POSTGRES_DSN`にcredentialが含まれ得ることを踏まえ、接続設定初期化と最初の接続確認でunderlying driver error本文をそのままpanic / stderrへ出さない境界を追加しました。malformed URL等でraw DSNがdriver errorへ含まれても、その本文をwrap・複製せず、非機密なstageだけを診断として残します。PR #415で同じ境界をtool READMEへ同期しました。

現在の実行順は次です。

1. repository revision / working tree cleanを確認する
2. dirtyならDB接続前に停止する
3. WorkspaceをLoadする
4. 通常`validation.Validate`を実行する
5. Validation ErrorがあればDB接続前に停止する
6. Validation成功後にのみPostgreSQLへ接続する。接続設定初期化または最初の接続確認が失敗した場合、driver error本文をstderrへ複製せず停止する
7. `current_database()`で接続先Database identityを取得する
8. `postgres` / `template0` / `template1`なら停止する
9. system / temporary領域を除くuser relation件数を確認する
10. user relationが1件でも存在すれば停止する
11. Event Trigger件数を確認する
12. Event Triggerが1件でも存在すれば停止する
13. `public.order_lines`の既存有無を再確認し、存在すれば停止する
14. すべての安全条件を満たした場合だけtool-owned fixtureをCREATEする
15. Provider runtime Evidenceを確認する
16. tool自身がCREATEしたfixtureをcleanupし、missing-table Execution Errorを確認する
17. system / temporary領域を除くuser relationが0件へ戻ったことを再確認する
18. Event Triggerが0件であることを再確認する
19. `public.order_lines`が引き続き存在しないことを再確認する
20. repository HEAD / working treeを再確認する
21. Databaseまたはsource状態が期待状態へ戻っていなければ成功JSONを出さず停止する

### 接続失敗時にcredentialをstderrへ露出しない

`TEI_TEST_POSTGRES_DSN`はRepositoryやEvidence JSONへ保存しないだけでなく、接続初期化失敗時のstderrにもそのまま露出させません。

次の段階ではunderlying driver errorをwrap・複製せず、非機密なstageだけを返します。

```text
PostgreSQL接続設定の初期化
PostgreSQL接続確認
```

この境界は接続初期化に限定します。fixture作成後のProvider runtime、専用空Database guard、cleanup、source-fixed Evidenceの診断まで一律に隠すものではありません。

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

### Event Triggerを拒否する

Event Triggerは通常のuser relationとは別にDatabase単位で存在でき、DDLへ介入できます。そのため、user relationが0件でもEvent Triggerが1件以上存在するDatabaseはfixture実行先にしません。

概念上は次を確認します。

```sql
SELECT count(*) FROM pg_catalog.pg_event_trigger
```

Event Triggerの用途や安全性を推測せず、disabledであっても既存objectとして扱います。削除・無効化してEvidence用Databaseへ転用するoverrideはありません。

### fixture objectを再確認する

空Database検査後も、native SQLが利用する`public.order_lines`の既存有無を明示確認します。

```sql
SELECT to_regclass('public.order_lines')::text
```

既存objectがあれば停止します。

**既存relation、Event Trigger、tableを削除・無効化して続行するoverrideはありません。** 別の専用空Databaseを使用します。

## fixture作成前の安全判定をEvidenceへ残す

PR #335以降、fixture作成前の安全guardは、実装されているだけでなく、そのrunで満たした非機密な観測結果を`fixtureDatabaseSafety`としてEvidence JSONへ残します。PR #365以降はEvent Trigger件数も同じ安全判定へ含めます。

期待値:

```json
{
  "guardPassed": true,
  "databaseIdentityVerified": true,
  "databaseIdentityPersisted": false,
  "nonDefaultDatabaseVerified": true,
  "userRelationsBeforeFixture": 0,
  "eventTriggersBeforeFixture": 0,
  "fixtureObjectAbsentBeforeCreate": true
}
```

これは次を意味します。

- `current_database()`によるidentity確認に成功した
- 実Database名自体はEvidenceへ保存していない
- PostgreSQL既定Databaseではなかった
- fixture作成前のuser relation数が0件だった
- fixture作成前のEvent Trigger数が0件だった
- `public.order_lines`がfixture作成前に存在しなかった
- それらをすべて確認した後だけfixture作成へ進んだ

`guardPassed = true`を「組織運用上、このDatabaseがEvidence専用であることの完全証明」へ格上げしません。toolがそのrunで観測できる安全条件を満たしたことだけを示します。

## cleanup後の空状態をEvidenceへ残す

Provider runtimeの2件 / 0件確認後、tool-owned fixtureを削除してmissing-table Execution Errorを確認します。その後、成功Evidenceを出す前にPostgreSQL catalogを再観測します。

期待する`fixtureCleanupSafety`:

```json
{
  "verified": true,
  "userRelationsAfterCleanup": 0,
  "eventTriggersAfterCleanup": 0,
  "fixtureObjectAbsentAfterCleanup": true
}
```

これは次を示します。

- system / temporary領域を除くuser relationが0件へ戻った
- Event Triggerが0件である
- `public.order_lines`が存在しない
- missing-table Execution Error確認後も上記状態が維持された

終了時にuser relationまたはEvent Triggerが1件でも存在する場合、それがtool由来か他主体の並行変更かを推測せずfail closedで停止します。

このcleanup EvidenceにもDatabase名、DSN、credential、Event Trigger名や定義本文は保存しません。

## 確認するEvidence

同じrunで少なくとも次を確認します。

- cleanなrepository revisionから開始したこと
- `repositoryRevision`を保持すること
- 実行前後のHEADが同一であること
- 実行後もworking treeがcleanであること
- `reproducibleRevision = true`であること
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
- `fixtureDatabaseSafety.guardPassed = true`
- `fixtureDatabaseSafety.nonDefaultDatabaseVerified = true`
- `fixtureDatabaseSafety.userRelationsBeforeFixture = 0`
- `fixtureDatabaseSafety.eventTriggersBeforeFixture = 0`
- `fixtureDatabaseSafety.fixtureObjectAbsentBeforeCreate = true`
- `fixtureDatabaseSafety.databaseIdentityPersisted = false`
- `fixtureCleanupSafety.verified = true`
- `fixtureCleanupSafety.userRelationsAfterCleanup = 0`
- `fixtureCleanupSafety.eventTriggersAfterCleanup = 0`
- `fixtureCleanupSafety.fixtureObjectAbsentAfterCleanup = true`
- 接続先Database名 / DSN / credentialを通常Evidenceへ保存しない
- 接続設定初期化 / 最初の接続確認のdriver error本文をstderrへ複製しない

## 完了条件

- [x] Workspace Validation Error時にDB接続前に停止する
- [x] PostgreSQL既定Databaseをfixture先として拒否する（PR #324）
- [x] 既存user relationがあるDatabaseをfixture先として拒否する（PR #324）
- [x] Event TriggerがあるDatabaseをfixture先として拒否する（PR #365）
- [x] `public.order_lines`既存時に停止する
- [x] tool-owned fixtureだけをcleanupする
- [x] fixture作成前guardの非機密な判定結果をEvidence JSONへ残し、Database名 / DSNを保存しない（PR #335 / #365）
- [x] dirty working treeではDB接続前に停止する（PR #336）
- [x] Provider runtime後にHEAD同一 / working tree cleanを再確認する（PR #336）
- [x] repository revision / `reproducibleRevision` / post-run状態をEvidence JSONへ含める（PR #336）
- [x] cleanup後にuser relation 0件 / Event Trigger 0件 / fixture object不在を再観測し、非機密な結果をEvidence JSONへ残す（PR #341 / #365）
- [x] 接続設定初期化 / initial Pingのdriver error本文をcredential-safeに隔離し、READMEへ同期する（PR #414 / #415、GitHub #411）
- [x] cleanな対象revisionでCanonical Workspace Validationを成功させる（`workspaceValidation.passed = true`）
- [x] Evidence専用の空PostgreSQL Databaseでtoolを実行する（`userRelationsBeforeFixture = 0` / `eventTriggersBeforeFixture = 0`）
- [x] `reproducibleRevision = true`を実runで確認する
- [x] `fixtureDatabaseSafety`が期待する安全条件を示すことを実runで確認する（`guardPassed = true`）
- [x] `fixtureCleanupSafety`が期待する終了状態を示すことを実runで確認する（`verified = true` / `userRelationsAfterCleanup = 0` / `eventTriggersAfterCleanup = 0` / `fixtureObjectAbsentAfterCleanup = true`）
- [x] 2件取得を確認する（`rows = 2`）
- [x] 0件を正常な空collectionとして確認する（`zeroRows = 0`）
- [x] physical table欠落をExecution Errorとして確認する（`executionErrorObserved = true`）
- [x] Semantic mappingとsupport column非露出を確認する（`semanticMappingApplied = true` / `semanticOrderLineParentAttributeAdded = false`）
- [x] parameter value / credential / DSN / Database名が通常Evidenceへ残っていないことを確認する（`parameterValuesPersisted = false` / `databaseIdentityPersisted = false`）
- [x] Evidence natureが`real-postgres-ephemeral-fixture`であることを確認する
- [x] 結果を`ISSUE-000029` / #273のGate B監査へ対応付ける（Gate B Readiness Audit「Required Evidence 3」）

## 非目標

- keyless Relation Storage境界の再設計
- header-detail PatternのUI layout設計
- production既存Databaseをfixtureで置換すること
- 既存relationやEvent Triggerを削除・無効化してEvidence用Databaseへ転用すること
- production PostgreSQL schemaの観測Evidence
- actual Oracle Evidenceの代替
- physical schemaからSemantic Relationを逆生成すること
- PostgreSQL固有仕様をSemantic modelへ逆流させること
- dirty / source変化中のrunをProvider Evidenceとして採用すること
- cleanup後にuser relationまたはEvent Triggerが残るrunを成功Evidenceとして採用すること
- GitHub Actionsの再有効化

## 関連

- GitHub Issue #69 / #244 / #255 / #273 / #308 / #310 / #411
- PR #294 / #300 / #306 / #309 / #324 / #335 / #336 / #341 / #365 / #368 / #414 / #415
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/issue/000024-keyless-relation-physical-storage.md`
- `tools/internal/repositoryevidence/`
- `tools/reference-postgres-header-detail-runtime/README.md`
- `tools/reference-postgres-header-detail-runtime/main.go`
- `tools/reference-postgres-header-detail-runtime/safety.go`
