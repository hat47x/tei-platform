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

一方、tool自身が一時的にfixture table / rowを作ります。このため、**production既存Databaseの物理事実を観測したEvidenceではありません**。

Evidence JSONでは`evidenceNature = real-postgres-ephemeral-fixture`として区別します。

## 必須環境

`TEI_TEST_POSTGRES_DSN`へ、**Evidence専用に作成した空のPostgreSQL Database**を指定します。

```sh
TEI_TEST_POSTGRES_DSN='...' go run ./tools/reference-postgres-header-detail-runtime
```

DSNやcredentialはEvidence JSONへ保存しません。

## fail-closedの実行順

Databaseへ接続・変更する前に、まず`reference/workspace`を通常Loaderで読み込み、`validation.Validate`を実行します。

```text
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
public.order_lines既存確認
  ↓
tool-owned fixture CREATE
  ↓
Provider runtime Evidence
```

したがって、Workspace authoringが不正な状態ではPostgreSQLへ接続せず、fixtureも作成しません。Warningだけで停止条件を増やさず、通常Validationの`Error`をEvidence実行不能条件として扱います。

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

### 3. fixture objectを再確認

空DB検査後も`public.order_lines`の既存有無を明示確認します。

```sql
SELECT to_regclass('public.order_lines')::text;
```

既存objectがあれば停止します。

これらは専用Databaseであることの運用上の意図を完全に証明するものではありませんが、少なくとも「他のuser relationがある既存DB」や「既定postgres DB」を誤ってfixture先にする経路を閉じます。

既存tableをDROPしてfixtureへ置き換えるオプションは提供しません。

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
  "fixtureObjectAbsentBeforeCreate": true
}
```

意味は次です。

- `databaseIdentityVerified`: `current_database()`を取得できた
- `databaseIdentityPersisted`: 実Database名をEvidenceへ保存していない
- `nonDefaultDatabaseVerified`: `postgres` / `template0` / `template1`ではなかった
- `userRelationsBeforeFixture`: system / temporary領域を除くfixture作成前user relation数
- `fixtureObjectAbsentBeforeCreate`: `public.order_lines`がfixture作成前に存在しなかった
- `guardPassed`: 上記の安全条件をすべて通過してからfixture作成へ進んだ

`guardPassed = true`は「このDatabaseが組織運用上Evidence専用である」ことまで証明するものではありません。toolが観測できる安全条件を、そのrunで満たしたことを示します。

## 出力

成功時はJSON Evidenceをstdoutへ出力します。

主な項目:

- Evidence nature
- fixture object / tool ownership / preexisting fixture protection
- `fixtureDatabaseSafety`の非機密な安全判定結果
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

接続先の実Database名やDSNを通常Evidenceへ保存することは、このtoolの目的に含めません。

## 非目標

- production Databaseに対する破壊的fixture作成
- 既存relationを削除してEvidence用Databaseへ転用すること
- 既存`public.order_lines`の置換
- productionの実データ内容の検証
- #214 external-observed `master_data`の物理Evidence代替
- PostgreSQL既存Schema全体の受入れ検証
- runtime parameter値やcredentialのEvidence保存
- 接続先の実Database名やDSNを安全判定Evidenceへ残すこと

Gate Bでは、本toolのEvidenceを「実PostgreSQLエンジン上のfixture runtime Evidence」として扱い、external-observed assetの実物理Evidenceやproduction data Evidenceとは区別します。
