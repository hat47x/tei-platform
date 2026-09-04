# reference-postgres-header-detail-runtime

第一Referenceの`Order -> OrderLine[]` header-detailを、**実PostgreSQLエンジン上の一時fixture**で確認するEvidence toolです。

## Evidenceの性質

このtoolが確認するのは、実PostgreSQL provider / driver / SQL実行境界を通して次が成立することです。

- `development / sales_core.dev / database.query.postgres`の明示composition
- `find_by_order(order_id)`のnative PostgreSQL SQL実行
- physical support column `order_id`をSemantic OrderLine attributeへ漏らさないこと
- 複数rowを正常なcollectionとして扱うこと
- 0 rowを正常な空collectionとして扱うこと
- physical execution errorを0 rowへ変換しないこと
- runtime parameter値を通常Evidenceへ保存しないこと
- Evidenceを一つのclean repository revisionへ固定できること
- fixture実行前にuser relation 0件 / Event Trigger 0件 / fixture object不在を確認できること
- fixture実行後に接続先をuser relation 0件 / Event Trigger 0件 / fixture object不在へ戻せたこと

一方、tool自身が一時的にfixture table / rowを作ります。このため、**production既存Databaseの物理事実を観測したEvidenceではありません**。

Evidence JSONでは`evidenceNature = real-postgres-ephemeral-fixture`として区別します。

## 必須環境

`TEI_TEST_POSTGRES_DSN`へ、**Evidence専用に作成した空のPostgreSQL Database**を指定します。

```sh
TEI_TEST_POSTGRES_DSN='...' go run ./tools/reference-postgres-header-detail-runtime
```

DSNやcredentialはEvidence JSONへ保存しません。

### 接続失敗時のcredential保護

`TEI_TEST_POSTGRES_DSN`にはpassword等のcredentialが含まれ得ます。そのため、接続設定の解析または最初の接続確認でdriverが返すerror本文を、そのままpanic / stderrへ出しません。

次の二段階では、underlying driver errorをwrap・複製せず、非機密なstageだけを診断として残します。

```text
PostgreSQL接続設定の初期化
PostgreSQL接続確認
```

malformed URL等でdriver error本文にraw DSNが含まれる場合でも、その本文をstderrへ転記しないための境界です。

この抑制は接続初期化に限定します。fixture作成後のProvider runtime、専用空Database guard、cleanup、source-fixed Evidenceの診断まで一律に隠すものではありません。

## source revisionのfail-closed境界

Gate Bへ採用するProvider Evidenceは、後から実際に実行したsourceへ戻れる必要があります。

そのため、Databaseへ接続する前に次を確認します。

```bash
git rev-parse HEAD
git status --porcelain
```

working treeがdirtyなら、Workspace ValidationやPostgreSQL接続へ進まず停止します。

Provider Evidenceを取得し、tool-owned fixtureを片付けた後にも同じ二点を再確認します。次のいずれかが起きていれば成功Evidenceを出しません。

- 実行中にHEADが変化した
- 実行後にtracked / untracked差分が残った

成功JSONには次を含めます。

```text
repositoryRevision
workingTreeClean = true
reproducibleRevision = true
postRunRepository.checked = true
postRunRepository.repositoryRevision = <same revision>
postRunRepository.revisionStable = true
postRunRepository.workingTreeClean = true
```

source状態の観測規則は`tools/internal/repositoryevidence`へ置き、#310 deterministic Repository Evidenceと#255 generated artifact Evidenceでも同じ規則を利用します。Provider固有のDB操作やSemantic意味はこの共有packageへ持ち込みません。

## fail-closedの実行順

```text
Repository revision / working tree確認
  ↓ dirtyなら停止
Reference Workspace load
  ↓
Canonical Validation
  ↓ Errorがあれば停止
TEI_TEST_POSTGRES_DSNへ接続
  ↓
接続先Database identity確認
  ↓
既定postgres/template DBなら停止
  ↓
既存user relationが1件でもあれば停止
  ↓
Event Triggerが1件でもあれば停止
  ↓
public.order_lines既存確認
  ↓
tool-owned fixture CREATE
  ↓
Provider runtime Evidence
  ↓
tool-owned fixture cleanup / missing-table Execution Error確認
  ↓
user relation 0件 / Event Trigger 0件 / fixture object不在を再確認
  ↓ 残存があれば停止
Repository revision / working tree再確認
  ↓ 変化していれば停止
JSON Evidence
```

したがって、dirty source、Workspace authoringが不正な状態、既存資産やEvent Triggerを含むDatabase、cleanup後に観測条件へ戻っていないrunのいずれも成功Evidenceへ進みません。Warningだけで停止条件を増やさず、通常Validationの`Error`をEvidence実行不能条件として扱います。

## Database側の安全条件

このtoolは`reference/workspace`のnative SQLが明示参照する`public.order_lines`をfixtureとして利用します。

`public.order_lines`だけが未使用の既存業務DBを「空DB」と誤認しないため、fixture作成前に接続先全体を検査します。

### 1. 既定Databaseを拒否

次のPostgreSQL既定Databaseはfixture実行先にしません。

```text
postgres
template0
template1
```

特に`postgres`が偶然空であっても、管理・接続確認等に使われ得る既定Databaseなので拒否します。

### 2. 既存user relationを拒否

`pg_catalog` / `information_schema` / PostgreSQL内部・temporary領域を除き、次の種類のrelationが一つでも存在すれば停止します。

- table / partitioned table
- view / materialized view
- foreign table
- sequence

概念上は次のcatalog検査を行います。

```sql
SELECT count(*)
FROM pg_catalog.pg_class AS c
JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
  AND n.nspname NOT LIKE 'pg_toast%'
  AND n.nspname NOT LIKE 'pg_temp_%'
  AND c.relkind IN ('r', 'p', 'v', 'm', 'f', 'S');
```

1件以上なら、内容や用途を推測せず**fail closedで実行を拒否**します。既存relationを削除して「空」にする処理はありません。

### 3. Event Triggerを拒否

PR #365以降、user relationが0件でもEvent Triggerが残るDatabaseはfixture実行先にしません。

Event Triggerはrelationとは別にDatabase内へ存在でき、`CREATE TABLE` / `DROP TABLE`等のDDLへ介入できます。このtool自身がfixture tableをCREATE / DROPするため、relation 0件だけでは安全条件として不十分です。

```sql
SELECT count(*)
FROM pg_catalog.pg_event_trigger;
```

1件でも存在すれば、enabled / disabledや用途を推測せず停止します。既存Event Triggerを削除・無効化して続行する処理はありません。

この確認は「組織運用上、このDatabaseがEvidence専用である」という意図を完全に証明するものではありません。少なくとも、DDLへ介入し得る既存Event Triggerを持つDatabaseをfixture先として受け入れないためのfail-closed境界です。

### 4. fixture objectを再確認

空Database検査後も`public.order_lines`の既存有無を明示確認します。

```sql
SELECT to_regclass('public.order_lines')::text;
```

既存objectがあれば停止します。

これらは専用Databaseであることの運用上の意図を完全に証明するものではありませんが、少なくとも「他のuser relationがある既存DB」「Event TriggerがあるDB」「既定postgres DB」を誤ってfixture先にする経路を閉じます。

既存tableやEvent Triggerを削除してfixture用Databaseへ転用するオプションは提供しません。

実行を継続できる場合だけtool自身が`public.order_lines`をCREATEし、以後そのfixtureだけをtool所有物として扱います。途中失敗時はbest-effort cleanupを行い、Execution Error確認時にはtool自身のfixtureを削除してmissing-table errorを発生させます。

## 安全判定をEvidenceへ残す

安全guardを実装しているだけでなく、**そのrunがfixture作成前にどの条件を満たしたか**を、接続先identityを保存せずEvidence JSONへ残します。

`fixtureDatabaseSafety`には次を出力します。

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

意味は次です。

- `databaseIdentityVerified`: `current_database()`を取得できた
- `databaseIdentityPersisted`: 実Database名をEvidenceへ保存していない
- `nonDefaultDatabaseVerified`: `postgres` / `template0` / `template1`ではなかった
- `userRelationsBeforeFixture`: system / temporary領域を除くfixture作成前user relation数
- `eventTriggersBeforeFixture`: fixture作成前に存在したEvent Trigger数
- `fixtureObjectAbsentBeforeCreate`: `public.order_lines`がfixture作成前に存在しなかった
- `guardPassed`: 上記の安全条件をすべて通過してからfixture作成へ進んだ

`guardPassed = true`は「このDatabaseが組織運用上Evidence専用である」ことまで証明するものではありません。toolが観測できる安全条件を、そのrunで満たしたことを示します。

## cleanup後の空状態をEvidenceへ残す

missing-table Execution Errorを確認した後、成功JSONを出す前に接続先を再検査します。

`fixtureCleanupSafety`の期待値は次です。

```json
{
  "verified": true,
  "userRelationsAfterCleanup": 0,
  "eventTriggersAfterCleanup": 0,
  "fixtureObjectAbsentAfterCleanup": true
}
```

これは、tool-owned fixtureを削除したという制御フラグだけではなく、終了時のPostgreSQL catalogを再観測して次を確認したことを示します。

- system / temporary領域を除くuser relationが0件へ戻った
- Event Triggerが0件である
- `public.order_lines`が存在しない
- missing-table Execution Error確認後も上記状態が維持された

終了時にuser relationやEvent Triggerが残っている場合は、それがtool由来か他主体の並行変更かを推測せずfail closedで停止します。

このEvidenceにもDatabase名、DSN、credential、Event Trigger名や定義本文は保存しません。

## 出力

成功時はJSON Evidenceをstdoutへ出力します。

主な項目:

- Evidence nature
- repository revision / working tree / post-run source stability
- fixture object / tool ownership / preexisting fixture protection
- `fixtureDatabaseSafety`の非機密なfixture前安全判定
- `fixtureCleanupSafety`の非機密な終了時復元判定
- Pattern / Relation
- Environment / Database instance / executor Capability
- Contract Query target
- parameter names（値は保存しない）
- 2-row result / 0-row result
- Execution Errorを観測したこと
- Semantic attributes
- Relation Storage column
- parent Order identityをOrderLine Semantic attributeへ追加していないこと
- Semantic mapping適用

接続先の実Database名やDSNを通常Evidenceへ保存しません。

## 非目標

- production Databaseに対する破壊的fixture作成
- 既存relationを削除してEvidence用Databaseへ転用すること
- 既存Event Triggerを削除・無効化してEvidence用Databaseへ転用すること
- 既存`public.order_lines`の置換
- productionの実データ内容の検証
- #214 external-observed `master_data`の物理Evidence代替
- PostgreSQL既存Schema全体の受入れ検証
- runtime parameter値やcredentialのEvidence保存
- 接続先の実Database名やDSNを安全判定Evidenceへ残すこと
- 接続設定parse / initial connect失敗時にraw driver errorをcredential込みでstderrへ複製すること
- Event Trigger名や定義本文を通常Evidenceへ保存すること
- dirty working tree上の成功をGate B完了Evidenceへ格上げすること
- cleanup後にuser relationまたはEvent Triggerが残るrunを成功Evidenceへ格上げすること

Gate Bでは、本toolのEvidenceを「実PostgreSQLエンジン上のfixture runtime Evidence」として扱い、external-observed assetの実物理Evidenceやproduction data Evidenceとは区別します。
