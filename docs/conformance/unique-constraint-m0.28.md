# Conformance — M0.28 Unique Constraint realization / drift

M0.28では、UNIQUE constraintをPersistence MappingからDesired Physical Schemaへ実現し、actual PostgreSQLから観測した制約とprovider-neutralなSchema Diffで比較します。

PostgreSQLはUNIQUE constraintを内部的にunique indexで支えますが、TEIでは両者を同じ意味として扱いません。UNIQUE constraintは「重複を許さない」というconstraint intent、明示的なunique indexは「一意なaccess pathを設ける」というphysical optimization intentです。

設計判断は [ADR-0009](../adr/0009-unique-constraint-and-unique-index.md) に記録しています。

## 検証する経路

```text
persist ... {
    unique ...
}
        ↓
Persistence UniqueConstraintMapping
        ↓
RealizeDesired
        ↓
Desired physical.UniqueConstraint

actual PostgreSQL
        ↓ pg_constraint / pg_index
Actual physical.UniqueConstraint
        ↓
constraint backing indexは通常Indexへ二重計上しない

Desired + Actual
        ↓ schemadiff.Compare
AddUniqueConstraint / DropUniqueConstraint
```

同時に、authoringで明示した`unique-index`は従来どおり`physical.Index{Unique:true}`として保持します。

## Authoring

Persistence Mappingでは次のようにUNIQUE constraintを明示します。

```tei
unique customers_identity_key {
    attribute sales.customer.tenant_id
    attribute sales.customer.external_id
}
```

複合constraintでは`attribute`を複数記述し、記述順を保持します。

Parserは空のconstraintを拒否します。Attributeの存在とphysical columnへの解決はDesired realizationでも検査するため、存在しないAttributeを制約へ黙って残しません。

## Desired realization

`persistence.UniqueConstraintMapping`が保持するのは次です。

- optional constraint name
- 順序付きAttribute IDs

`RealizeDesired`はAttribute IDを同じMapping内のphysical column名へ解決し、`physical.UniqueConstraint{Name, Columns}`を生成します。

constraint名は観測結果の説明や将来DDLを組み立てる際に有用なので保持します。ただしM0.28のSchema Drift identityには含めません。

## constraint nameとstructural identity

M0.28の比較identityは**順序付きcolumns**です。

したがって、次の二つは同じ意味として扱います。

```text
customers_email_key(email)
legacy_customer_email_unique(email)
```

名前だけが異なる場合はdriftにしません。

一方、columnやcolumn順序が異なれば別構造です。構造変更は専用`alter-unique-constraint`ではなく、Actual側の`drop-unique-constraint`とDesired側の`add-unique-constraint`として表します。

## multiplicityを失わない

同じcolumnsを持つUNIQUE constraintが複数存在する場合、mapへ1件だけ格納して消しません。

構造ごとの件数を比較し、DesiredとActualの件数差をadd/dropとして表します。名前は意味identityではありませんが、Actual側の余分なconstraintを説明するためChangeには保持します。

## PostgreSQL Actual normalization

PostgreSQL providerは`pg_catalog.pg_constraint`の`contype='u'`からUNIQUE constraintを観測します。

columnsは`conkey`の順序で復元します。M0.28が表現していない`NULLS NOT DISTINCT`を観測した場合は、通常のUNIQUE constraintへ黙って畳まず、unsupported semanticsとして明示的に失敗します。

### backing indexを二重計上しない

PostgreSQLではPRIMARY KEYやUNIQUE constraintにbacking indexがあります。

`Indexes()`では`pg_constraint.conindid`がPRIMARY KEYまたはUNIQUE constraintを指すindexを通常Indexから除外します。これにより、同じDatabase intentを次のように二重計上しません。

```text
UniqueConstraint(email)
+ Index(unique=true, email)  ← constraint内部実装を誤って別intent扱いしない
```

一方、利用者が明示的に作成したunique indexはconstraintではないため、通常の`physical.Index{Unique:true}`として残します。

## Diff表現

M0.28では次を追加します。

- `add-unique-constraint`: DesiredにありActualにない
- `drop-unique-constraint`: ActualにありDesiredにない

Diffは意味差の表現であり、その場でDDLを自動実行する命令ではありません。

## actual PostgreSQL Conformance

`tools/reference-postgres-integration`では、同じtableに次を同時に作ります。

- `external_id`に対するUNIQUE constraint
- `name`に対する明示的なunique index

Persistence Mapping側にも同じ二つのintentを明示し、baselineのDesired / Actual diffが0件であることを要求します。

さらにActual Snapshotについて次を確認します。

- `external_id`のUNIQUE constraintを観測できる
- constraint backing indexは通常Indexへ現れない
- 明示的なunique indexは`Unique=true`のIndexとして残る

その後、test harnessがactual DBからUNIQUE constraintだけを削除します。再introspection後のDiffでは`add-unique-constraint`が検出されなければ失敗します。

このtest harnessによるconstraint削除は、TEIがautomatic migrationや自動修復を行うことを意味しません。TEIはDesiredとActualを比較して意味差を返しています。

## 常設CI

PostgreSQL serviceを使うintegration jobで、少なくとも次を継続的に確認します。

- UNIQUE constraint込みbaseline diff 0件
- actual UNIQUE constraintの観測
- backing indexを通常Indexへ二重計上しない
- explicit unique indexをIndexとして保持
- actual UNIQUE constraint削除後の`add-unique-constraint`
- 既存のcolumn / Foreign Key drift検出
- connection dataをPhysical Snapshot / Diffへ保存しない

通常のunit testでは次を固定します。

- `unique { attribute ... }` authoring
- 空constraintの拒否
- Attribute IDからphysical columnへのDesired realization
- unknown Attributeの拒否
- constraint名だけの差を無視
- ordered columnsをstructural identityとして扱う
- 同一構造constraintのmultiplicityを保持
- explicit unique indexとのintent分離

## 非目標

M0.28では次を扱いません。

- `NULLS NOT DISTINCT`等の非default NULL semantics
- partial index
- expression index
- INCLUDE columns
- index validity / readiness / liveness
- partitioning
- migration SQL生成
- schema変更の自動適用

Advanced Index semanticsはIssue #42、Partition semanticsはIssue #43で別途設計します。provider固有のcatalog fieldを見つけたという理由だけで、provider-neutral modelへ無条件に持ち上げません。
