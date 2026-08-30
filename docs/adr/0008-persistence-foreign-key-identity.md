# ADR-0008: Desired Foreign KeyをPersistence Mappingで明示し、構造で差分比較する

- 状態: 採用
- 日付: 2026-08-31
- 関連Issue: #37
- 前提: M0.26 actual PostgreSQL integration

## 背景

M0.26では、Persistence Mappingから生成したDesired Physical Schemaと、actual PostgreSQLから観測したActual Physical Schemaを`schemadiff.Compare`へ接続しました。Actual SnapshotにはForeign Keyも含まれています。

しかし、現行のPersistence Mappingはcolumn、primary key、indexまでは表現できる一方、Desired Foreign Keyを表現できません。また、`schemadiff.Compare`もForeign Keyを比較していません。

この状態でDiffだけへForeign Key比較を足しても、Desired側に「このFKを存在させたい」という根拠がありません。反対に、Semantic Relationを自動的にFKへ変換すると、業務上の関係と物理DBの参照制約を同一視することになります。Relationは存在してもFKを張らない設計、同じ意味を別DBへ分散する設計、外部システムとの関係などを表現しにくくなります。

そのため、Foreign Keyをどの層の意思として扱うかを先に固定します。

## 判断

Desired Foreign Keyは、**Database所有のPersistence Mappingで明示します**。

Semantic Relationから自動生成しません。Semantic Layerは「何が関係しているか」を表し、Persistence Mappingは「このDatabaseで、その関係をどの物理制約として実現するか」を決めます。

概念上は次の構造です。

```text
Semantic relation
  └─ domain meaning

Persistence Mapping
  ├─ table / columns
  ├─ primary key
  ├─ indexes
  └─ foreign keys  ← Database-owned realization decision
        ↓
Desired Physical Schema
        ↓
Actual Physical Schema
        ↓
Schema Diff
```

## Persistence Mappingの参照方法

Foreign Keyの参照先を、authoring時点から`public.customers(id)`のような生の物理名だけで指定しません。

代わりに、次を保持します。

- source側Attribute IDの順序付き列
- referenced Mapping ID
- referenced側Attribute IDの順序付き列
- constraint name（任意のtrace / DDL metadata）

概念上のGo modelは次です。

```go
type ForeignKeyMapping struct {
    Name                 string
    Attributes           []core.CanonicalID
    ReferencedMapping    core.CanonicalID
    ReferencedAttributes []core.CanonicalID
}
```

`RealizeDesired`が、source Mappingとreferenced Mappingのcolumn定義を使ってphysical table / columnへ解決します。

これにより、column名を変更してもPersistence Mapping内のAttribute identityを基準に整合を検査でき、参照先table名の組み立ても一か所へ集約できます。

## Authoring構文

現行`persist` blockへ、複合Foreign Keyにも対応できる小さなblockを追加します。

```tei
foreign-key orders_customer_fk {
    source sales.order.customer_id
    references sales_core.customer
    target sales.customer.customer_id
}
```

複合Foreign Keyでは`source`と`target`を複数記述します。

```tei
foreign-key order_line_product_fk {
    source sales.order_line.tenant_id
    source sales.order_line.product_id
    references sales_core.product
    target sales.product.tenant_id
    target sales.product.product_id
}
```

`source`と`target`の記述順を対応順として保持します。件数が一致しなければ無効です。

## cross-database Foreign Key

M0.27では、source Mappingとreferenced Mappingが同じDatabase definitionに属することを要求します。

多くのRDBMSの通常Foreign Keyは同一Database内の物理制約です。異なるDatabase definitionをまたぐ関係を同じForeign Keyとして扱うと、DBMS固有機能や分散整合性の意味を暗黙に混ぜることになります。

cross-database relationshipが必要な場合はSemantic RelationやApplication logic等、別の境界で表現します。将来、特定DBMSのcross-database参照を扱う必要が出た場合は別Issue / ADRで検討します。

## Physical Foreign Keyの名前

`physical.ForeignKey.Name`は保持します。

これはactual DBで観測したconstraintを識別したり、将来DDLを生成したり、人間へ説明したりする際に有用です。ただし、**constraint nameそのものをSchema Driftの意味identityにはしません**。

DBMSやmigration toolはconstraint名を自動生成することがあり、同じ参照制約でも名前だけが異なる場合があります。名前差をそのままdriftとすると、意味上同じschemaを不要に異なるものとして扱います。

## Schema Diffの意味identity

Foreign Keyの意味比較は、次の構造を基準にします。

```text
source table
source columns (ordered)
referenced table
referenced columns (ordered)
```

同じTable内を比較する際はsource tableは既知なので、Foreign Key keyとしては概念上次を使えます。

```text
ordered source columns
+ referenced qualified table
+ ordered referenced columns
```

constraint nameだけが異なる場合は差分なしとします。

source columns、referenced table、referenced columnsのいずれかが変わった場合は、M0.27では`DropForeignKey` + `AddForeignKey`として表します。専用`AlterForeignKey`は増やしません。既存Index diffと同じく、物理制約の置換として明示する方が単純だからです。

## 重複Foreign Key

構造が同じForeign Keyをmapへ1件だけ格納して黙って潰しません。

Actual DBに同一構造のconstraintが複数存在することは冗長ですが、観測事実としては区別すべきです。比較では構造keyごとのmultiplicityを保持し、Desired / Actualの件数差を検出できる形にします。

## 名前を完全に捨てない理由

意味比較からNameを外す一方で、Changeには実際に追加・削除対象となる`physical.ForeignKey`を保持します。

これにより、Diffの意味判定はprovider-neutralな構造で行いながら、Actual側でどのconstraint名を観測したかという情報を失いません。

## 採用しなかった案

### Semantic RelationからForeign Keyを自動生成する

便利に見えますが、業務上の関係と物理参照制約を同一視します。Databaseごとに異なる実現を選べなくなるため採用しません。

### referenced table / columnを物理文字列で直接authoringする

記述は短くなりますが、Attribute identityと物理column mappingの整合検査が弱くなり、renameにも脆くなります。既存Persistence MappingがAttribute→columnを明示する設計とも揃わないため採用しません。

### constraint nameをForeign Key identityとする

DBMSやmigration toolの命名規則へ意味identityが依存するため採用しません。Nameはtrace / DDL metadataとして保持します。

### Foreign Keyの構造変更を`AlterForeignKey`一件で表す

多くのDBMSでは参照制約の本質的変更はdrop + addとして実現されます。現段階で専用ChangeKindを増やす利点が小さいため採用しません。

## 影響

- Persistence MappingにForeign Key realizationを明示できる
- Semantic Relationとphysical FKの責務を分離したまま保てる
- `RealizeDesired`がMapping / Attribute identityからphysical FKを決定的に生成できる
- Schema DiffがFK driftをprovider-neutralに検出できる
- constraint名だけの差で不要なdriftを作らない
- actual PostgreSQLで観測済みのFKをM0.27からbaseline comparisonへ含められる

このADRはON DELETE / ON UPDATE、deferrable constraint、DBMS固有FK option、migration DDL生成、自動schema修復を決めるものではありません。必要になった段階で別のIssue / ADRとして扱います。
