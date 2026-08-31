# Conformance — M0.27 Foreign Key realization / drift

M0.27では、M0.26でactual PostgreSQLから観測できることまで確認したForeign Keyを、Desired Physical SchemaとSchema Diffへ統合します。

目的はPostgreSQL固有のForeign Key機能を増やすことではありません。Database所有のPersistence MappingでDesired FKを明示し、provider-neutralなPhysical Schema上でActualとの意味差を比較できるようにします。

設計判断は [ADR-0008](../adr/0008-persistence-foreign-key-identity.md) に記録しています。

## 検証する経路

```text
persist ... {
    foreign-key ...
}
        ↓
Persistence ForeignKeyMapping
        ↓
RealizeDesired
        ↓
Desired physical.ForeignKey

actual PostgreSQL
        ↓ introspection
Actual physical.ForeignKey

Desired + Actual
        ↓ schemadiff.Compare
AddForeignKey / DropForeignKey
```

Semantic RelationからForeign Keyを自動生成しません。FKを張るかどうかはDatabaseのPersistence realizationです。

## Authoring

Persistence Mappingでは次の形でFKを明示できます。

```tei
foreign-key orders_customer_fk {
    source sales.order.customer_id
    references sales_core.customer
    target sales.customer.customer_id
}
```

複合FKでは`source`と`target`を複数記述します。記述順を対応順として保持します。

```tei
foreign-key order_line_product_fk {
    source sales.order_line.tenant_id
    source sales.order_line.product_id
    references sales_core.product
    target sales.product.tenant_id
    target sales.product.product_id
}
```

Parserは少なくとも次を拒否します。

- sourceが0件
- referenced Mappingがない
- source / targetの件数不一致
- `references`の重複宣言

Mapping / Attributeの存在やDatabase境界はDesired realization時にも検査します。

## Desired realization

`persistence.ForeignKeyMapping`は物理table / column名を参照先として直接保持しません。

保持するのは次です。

- source Attribute IDs
- referenced Mapping ID
- referenced Attribute IDs
- optional constraint name

`RealizeDesired`はMapping IDからtarget tableを、Attribute IDからsource / target columnを解決します。

次の場合は失敗します。

- referenced Mappingが存在しない
- source Attributeにcolumn mappingがない
- referenced Attributeにcolumn mappingがない
- source / target attribute数が異なる
- source / target Mappingが異なるDatabase definitionに属する

M0.27ではcross-database FKを作りません。

## constraint nameと意味identity

`physical.ForeignKey.Name`は失いません。actual DBでどのconstraintを観測したか、人間へ説明する際、将来DDLを組み立てる際に有用だからです。

ただし、Schema Diffの意味identityにはNameを使いません。

比較keyは次の構造です。

```text
ordered source columns
+ referenced qualified table
+ ordered referenced columns
```

したがって、構造が同一でconstraint名だけ違うFKはdriftではありません。

## Diff表現

M0.27では次の2種類を追加します。

- `add-foreign-key`: DesiredにありActualにない
- `drop-foreign-key`: ActualにありDesiredにない

参照先やcolumn順序の変更は、既存制約の`drop-foreign-key`とDesired制約の`add-foreign-key`として表します。専用`alter-foreign-key`は設けません。

Diffは変更提案のための意味差であり、TEIがその場でDDLを実行する命令ではありません。

## 重複constraint

同じ構造を持つForeign Keyが複数存在する場合、mapへ1件だけ格納して黙って失いません。

構造keyごとの件数を比較します。Desiredが1件、Actualが同一構造2件なら、余分なActual constraintを1件の`drop-foreign-key`として表します。

この場合もNameは構造identityではありませんが、どのactual constraintを余分な1件として報告したかをChange内に保持します。

## Physical Snapshotのdeterminism

`SchemaSnapshot.Normalize()`はForeign KeyをNameだけで並べません。

まず構造順で並べ、同一構造のtie-breakとしてNameを使います。これにより、constraint名を意味identityへ昇格させずにdeterministicなSnapshotを維持します。

## actual PostgreSQL Conformance

`tools/reference-postgres-integration`のbaselineでは、Persistence Mappingから次のDesired FKを生成します。

```text
tei_reference.orders.customer_id
  → tei_reference.customers.id
```

actual PostgreSQLにも`orders_customer_fk`を作成し、FKを含むDesired / Actual全体で差分0件を要求します。

その後、test harnessがactual側だけで次を行います。

- `drift_note` columnを追加
- `orders_customer_fk` constraintを削除

再introspection後のDiffでは、少なくとも次を要求します。

- actual-only `drift_note`に対する`drop-column`
- Actualから欠落したDesired FKに対する`add-foreign-key`

ここでtest harnessがconstraintを削除することは、TEIのautomatic migration能力を意味しません。TEIは観測して差分を返すだけです。

## 常設CI

PostgreSQL serviceを使う常設jobで次を確認します。

- actual PostgreSQL introspection
- Foreign Key込みbaseline diff 0件
- actual FKを1件観測
- FK削除後に`foreignKeyDriftDetected=true`
- `add-foreign-key`を実DB由来Actualから検出
- M0.26から継続してconnection dataがSnapshot / Diffへ混入しない

通常のunit testでは次を固定します。

- ParserのForeign Key block
- Mapping / Attribute identityからのDesired realization
- cross-database拒否
- constraint名だけの差を無視
- structural changeをdrop + addで表現
- composite column順序を意味として保持
- 同一構造のmultiplicityを保持

## 非目標

M0.27では次を扱いません。

- ON DELETE / ON UPDATE
- deferrable / initially deferred
- match type
- PostgreSQL固有constraint option
- cross-database FK
- Semantic Relationからの自動FK生成
- migration SQL生成
- schema変更の自動適用

必要になった場合は、provider-neutralな意味があるかを確認してから別Issue / ADRで拡張します。
