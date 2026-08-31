# ADR-0009: UNIQUE constraintとunique indexを別のphysical intentとして扱う

- 状態: 採用
- 日付: 2026-09-01
- 関連Issue: #41
- 前提: M0.27 Foreign Key realization / drift、Issue #38 PostgreSQL catalog normalization監査

## 背景

M0.26でactual PostgreSQLのschema introspectionを常設CIへ入れた際、PRIMARY KEYが内部的に作るbacking indexを通常Indexとして二重計上する問題が見つかりました。そのため、PostgreSQL providerでは`indisprimary`を通常Index一覧から除外し、Primary Keyは`physical.Table.PrimaryKey`、明示的なIndexは`physical.Table.Indexes`へ分けています。

Issue #38の監査では、同じ問題がUNIQUE constraintにも潜んでいることが分かりました。

PostgreSQLでは、次の二つはどちらもunique B-tree indexを利用できます。

```sql
ALTER TABLE customers
    ADD CONSTRAINT customers_email_key UNIQUE (email);
```

```sql
CREATE UNIQUE INDEX customers_email_idx
    ON customers(email);
```

しかし、これらはTEIが保持すべき意思として同一ではありません。

- UNIQUE constraintは「この列の組は重複してはならない」という表制約です。
- unique indexは「この索引を作り、その索引自体にもunique性を持たせる」という物理access pathです。

PostgreSQLではUNIQUE constraintの実装としてindexが作られますが、その実装詳細をそのままTEIの意味へ持ち上げると、constraintとindexが二重に現れたり、反対にconstraintという意思そのものが失われたりします。

## 判断

TEIでは、**UNIQUE constraintとunique indexを別のDatabase-owned physical intentとして扱います**。

概念上は次です。

```text
Persistence Mapping
  ├─ Primary Key
  ├─ Unique Constraints   ← relational constraint intent
  ├─ Foreign Keys
  └─ Indexes              ← access-path intent
        ↓
Desired Physical Schema
        ↓
Actual Physical Schema
        ↓
Schema Diff
```

`physical.Table`には、既存の`PrimaryKey`、`ForeignKeys`、`Indexes`とは別に`UniqueConstraints`を持たせます。

概念上のmodelは次の形です。

```go
type UniqueConstraint struct {
    Name    string
    Columns []string
}
```

Persistence Mapping側でも、unique constraintをIndexMappingの`Unique=true`で代用せず、専用のauthoring intentとして保持します。

概念上は次です。

```go
type UniqueConstraintMapping struct {
    Name       string
    Attributes []core.CanonicalID
}
```

`RealizeDesired`がAttribute IDからphysical column名へ解決します。

## explicit unique indexはIndexのまま維持する

既存の`IndexMapping.Unique`は削除しません。

たとえば次のように、unique indexそのものを物理access pathとして明示したい場合があります。

```text
Index
  name: customers_external_id_idx
  columns: [external_id]
  unique: true
```

この場合、TEIは「UNIQUE constraintを作りたい」とは解釈しません。

同じ列にUNIQUE constraintと明示的なunique indexの両方がauthoringされている場合も、勝手に一つへ統合しません。冗長に見えても、二つの異なるphysical intentが記述されているためです。

## PostgreSQL backing indexの扱い

PostgreSQL providerでは、UNIQUE constraintに紐づくbacking indexを通常の`physical.Indexes`へ二重計上しません。

Actualの組み立ては概念上、次のように分けます。

```text
pg_constraint contype = 'u'
    → physical.UniqueConstraint

pg_index
    ├─ PRIMARY KEY backing index      → Indexesへ入れない
    ├─ UNIQUE constraint backing index → Indexesへ入れない
    └─ explicit index                  → physical.Index
```

constraintとindexの対応は、名前の推測ではなくPostgreSQL catalogのidentityを使います。PostgreSQLではconstraintが利用するindex OIDを`pg_constraint.conindid`で追跡できるため、providerはこれを基準にbacking indexを除外します。

この判断はPostgreSQLに限ったものではありません。他DB providerでも、DBMS内部の補助indexをTEIの独立したDesired intentとして二重計上しない、という一般原則に従います。

## constraint nameは意味identityにしない

Foreign Keyと同様に、UNIQUE constraintの`Name`は保持します。

名前は次に役立ちます。

- actual DBで観測したconstraintを人間へ説明する
- 将来DDLを生成する
- migration / evidenceと対応付ける

一方、名前だけが異なるconstraintを別のunique semanticsとはみなしません。

Schema Diffの意味比較は、少なくともM0.28の最小vertical sliceでは**ordered columns**を基準にします。

```text
UNIQUE (tenant_id, external_id)
```

と

```text
UNIQUE (external_id, tenant_id)
```

は、論理的な重複禁止条件だけを見れば近い意味を持ちますが、DBMSが生成するbacking indexの列順や将来のDDL identityに影響し得ます。TEIのPhysical Schemaは単なる論理制約一覧ではなく、再現可能な物理意図を扱うため、M0.28では順序を保持します。

将来、constraint semanticsとbacking access pathをさらに分離し、列集合として比較する必要が生じた場合は別ADRで再評価します。

## NULLの扱い

UNIQUE constraintにおけるNULLの扱いはDBMS間で差があります。PostgreSQLにも`NULLS NOT DISTINCT`があります。

このADRでは、NULL semanticsを無視して同一扱いすることも、PostgreSQL固有のflagをそのままprovider-neutral modelへ持ち上げることも決めません。

M0.28の最小実装では、**各providerの通常のUNIQUE constraint semanticsだけをauthoring対象**とし、非defaultなNULL semanticsを観測した場合に黙って通常constraintへ畳まないことを要求します。

非default NULL semanticsをDesiredへauthoringする必要が生じた段階で、provider-neutralな意味表現を別Issue / ADRで定義します。

## Schema Diff

Unique Constraintの差分には、Foreign Keyと同様に専用ChangeKindを使います。

概念上は次です。

```text
add-unique-constraint
drop-unique-constraint
```

名前だけの差はdriftにしません。

列構造が異なる場合は、専用Alterを増やさずdrop + addで表します。

同じ列構造のconstraintが複数存在する場合は、mapへ1件だけ格納して黙って潰しません。Actualで観測したmultiplicityを保持し、Desiredとの差を検出します。

## Foreign Keyとの関係

このADRは、Foreign KeyがUNIQUE constraintを参照できるか、どのcandidate keyを参照先として許可するかというvalidationを新たに決めるものではありません。

M0.28ではまずUNIQUE constraint自体のDesired / Actual / Diffを成立させます。FKの参照先validationを強化する必要が出た場合は、別Issueで扱います。

## 採用しなかった案

### UNIQUE constraintを`Index{Unique:true}`へ統一する

PostgreSQLでは実装上近いですが、constraint intentとaccess-path intentを失います。他DB providerへの写像も不明瞭になるため採用しません。

### UNIQUE constraint backing indexを単純に除外し、constraint自体も保持しない

Actual DBが持つ重複禁止制約を観測結果から失うため採用しません。

### UNIQUE constraintとunique indexを両方観測して二重に保持する

DBMS内部で一つのconstraintを実現するために作られたbacking indexまで独立Desired intentとして見えるため採用しません。

### constraint名をidentityにする

DBMSやmigration toolの命名へ意味identityが依存し、名前だけの変更で不要なdriftが発生するため採用しません。

## 影響

- provider-neutral Physical SchemaにUnique Constraintという独立した意味単位が加わる
- Persistence MappingでUNIQUE constraintをIndexとは別にauthoringできる
- PostgreSQL providerは`pg_constraint`からUNIQUE constraintを観測し、そのbacking indexを通常Indexから除外する
- explicit unique indexは従来どおりIndexとして保持する
- Schema DiffでUnique Constraintのmissing / extra / structural driftを検出できる
- M0.27で確立したPrimary Key / Foreign Key / Indexの責務分離を維持できる

このADRは、advanced index semantics、partitioning、default expression、automatic migration SQL生成・適用を決めるものではありません。それらは既存の別Issueで扱います。
