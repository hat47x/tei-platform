# M0.27 外部キーの生成と差分の準拠確認

M0.27では、M0.26で実際のPostgreSQLから取得できることを確認した外部キーを、期待する物理構造と構造差分の比較に組み込みます。

目的はPostgreSQL固有の外部キー機能を増やすことではありません。データベース側が管理する保存先への対応付けに、期待する外部キーを明示し、提供元に依存しない物理構造として実際の状態との差を比較できるようにします。

設計判断は[ADR-0008](../../plan/adr/0008-persistence-foreign-key-identity.md)に記録しています。

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

意味上の関係から外部キーを自動生成することはしません。外部キーを設けるかどうかは、データベースでの保存方法を決める際の判断です。

## 記述方法

保存先への対応付けでは、次のように外部キーを明示できます。

```tei
foreign-key orders_customer_fk {
    source sales.order.customer_id
    references sales_core.customer
    target sales.customer.customer_id
}
```

複合外部キーでは`source`と`target`を複数指定し、その記述順を列の対応順として保持します。

```tei
foreign-key order_line_product_fk {
    source sales.order_line.tenant_id
    source sales.order_line.product_id
    references sales_core.product
    target sales.product.tenant_id
    target sales.product.product_id
}
```

構文解析では、少なくとも次の記述を拒否します。

- sourceが0件
- referenced Mappingがない
- source / targetの件数不一致
- `references`の重複宣言

対応付けと属性の存在、データベースの境界については、期待する物理構造を生成する際にも検査します。

## 期待する構造の生成

`persistence.ForeignKeyMapping`には、参照先の物理テーブル名や列名を直接保存しません。

保持する情報は次のとおりです。

- source Attribute IDs
- referenced Mapping ID
- referenced Attribute IDs
- optional constraint name

`RealizeDesired`は対応付けの識別子から参照先のテーブルを、属性の識別子から参照元と参照先の列を特定します。

次の条件に該当する場合は失敗します。

- referenced Mappingが存在しない
- source Attributeにcolumn mappingがない
- referenced Attributeにcolumn mappingがない
- source / target attribute数が異なる
- source / target Mappingが異なるDatabase definitionに属する

M0.27では、異なるデータベースにまたがる外部キーは作りません。

## 制約名と意味上の同一性

`physical.ForeignKey.Name`は保持します。実際のデータベースでどの制約を観測したかを説明するときや、将来DDLを作成するときに必要なためです。

ただし、構造差分の比較では、名前を意味上の同一性の判定に使いません。

比較に使う情報は次の構造です。

```text
ordered source columns
+ referenced qualified table
+ ordered referenced columns
```

そのため、構造が同じで制約名だけが異なる外部キーは、差分として扱いません。

## 差分の表し方

M0.27では、次の2種類の差分を追加します。

- `add-foreign-key`: DesiredにありActualにない
- `drop-foreign-key`: ActualにありDesiredにない

参照先や列の順序が変わった場合は、既存の制約を表す`drop-foreign-key`と、期待する制約を表す`add-foreign-key`の組合せで示します。専用の`alter-foreign-key`は設けません。

差分は変更案を示すための情報であり、TEIがその場でDDLを実行する命令ではありません。

## 制約の重複

同じ構造の外部キーが複数ある場合も、一つだけを記録して残りを失うことはしません。

構造ごとの件数を比較します。期待する制約が1件で実際には同じ構造が2件ある場合、余分な制約を1件の`drop-foreign-key`として示します。

この場合も制約名は構造上の同一性を決めませんが、どの制約を余分な1件として報告したかを変更情報に記録します。

## 物理構造の記録の決定性

`SchemaSnapshot.Normalize()`は、外部キーを名前だけで並べることはしません。

まず構造で並べ、同じ構造の場合に限り名前で順序を決めます。これにより、制約名を意味上の識別情報として扱わずに、再現可能な観測記録を保ちます。

## PostgreSQL実環境での準拠確認

`tools/reference-postgres-integration`の基準状態では、保存先への対応付けから次の外部キーを生成します。

```text
tei_reference.orders.customer_id
  → tei_reference.customers.id
```

実際のPostgreSQLにも`orders_customer_fk`を作成し、外部キーを含めて期待状態と実際の状態の差分が0件であることを要求します。

その後、テスト用の仕組みが実際のデータベース側にだけ、次の変更を加えます。

- `drift_note` columnを追加
- `orders_customer_fk` constraintを削除

実際の構造を再取得した後の差分には、少なくとも次の項目が必要です。

- actual-only `drift_note`に対する`drop-column`
- Actualから欠落したDesired FKに対する`add-foreign-key`

ここでテスト用の仕組みが制約を削除しても、TEI自身が自動的な移行機能を持つことにはなりません。TEIは状態を観測し、差分を返すだけです。

## 継続的な自動検査

PostgreSQLのサービスを使う常設の自動検査では、次の点を確かめます。

- actual PostgreSQL introspection
- Foreign Key込みbaseline diff 0件
- actual FKを1件観測
- FK削除後に`foreignKeyDriftDetected=true`
- `add-foreign-key`を実DB由来Actualから検出
- M0.26から継続してconnection dataがSnapshot / Diffへ混入しない

通常の単体テストでは、次の条件を確認します。

- ParserのForeign Key block
- Mapping / Attribute identityからのDesired realization
- cross-database拒否
- constraint名だけの差を無視
- structural changeをdrop + addで表現
- composite column順序を意味として保持
- 同一構造のmultiplicityを保持

## 非目標

M0.27では、次の事項は扱いません。

- ON DELETE / ON UPDATE
- deferrable / initially deferred
- match type
- PostgreSQL固有constraint option
- cross-database FK
- Semantic Relationからの自動FK生成
- migration SQL生成
- schema変更の自動適用

必要になった場合は、提供元に依存しない意味があるかを確かめたうえで、別の課題と設計判断の記録で拡張を検討します。
