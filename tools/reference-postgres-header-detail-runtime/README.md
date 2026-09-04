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

`TEI_TEST_POSTGRES_DSN`へ、Evidence専用のPostgreSQL Databaseを指定します。

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
既存fixture object guard
  ↓
tool-owned fixture CREATE
  ↓
Provider runtime Evidence
```

したがって、Workspace authoringが不正な状態ではPostgreSQLへ接続せず、fixtureも作成しません。Warningだけで停止条件を増やさず、通常Validationの`Error`をEvidence実行不能条件として扱います。

## Database側の安全条件

このtoolは`reference/workspace`のnative SQLが明示参照する`public.order_lines`をfixtureとして利用します。

誤ったDSNによる既存資産破壊を避けるため、接続後、fixture作成前に次を確認します。

```sql
SELECT to_regclass('public.order_lines')::text
```

`public.order_lines`が既に存在する場合、用途や内容を推測せず**fail closedで実行を拒否**します。

既存tableをDROPしてfixtureへ置き換えるオプションは提供しません。専用の空Databaseを利用してください。

実行を継続できる場合だけtool自身が`public.order_lines`をCREATEし、以後そのfixtureだけをtool所有物として扱います。途中失敗時はbest-effort cleanupを行い、Execution Error確認時にはtool自身のfixtureを削除してmissing-table errorを発生させます。

## 出力

成功時はJSON Evidenceをstdoutへ出力します。

主な項目:

- Evidence nature
- fixture object / tool ownership / preexisting fixture protection
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

## 非目標

- production Databaseに対する破壊的fixture作成
- 既存`public.order_lines`の置換
- productionの実データ内容の検証
- #214 external-observed `master_data`の物理Evidence代替
- PostgreSQL既存Schema全体の受入れ検証
- runtime parameter値やcredentialのEvidence保存

Gate Bでは、本toolのEvidenceを「実PostgreSQLエンジン上のfixture runtime Evidence」として扱い、external-observed assetの実物理Evidenceやproduction data Evidenceとは区別します。
