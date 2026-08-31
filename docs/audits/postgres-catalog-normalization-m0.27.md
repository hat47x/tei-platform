# PostgreSQL catalog normalization監査 — M0.27時点

この文書はIssue #38の監査結果です。目的は、新機能を増やすことではなく、actual PostgreSQLから得たcatalog情報をTEIのprovider-neutralなPhysical Schemaへ写す際に、PostgreSQL固有の実装詳細を二重計上したり、逆に意味のある差を落としたりしていないかを確認することです。

監査では、`internal/database/providers/postgres`、`internal/database/physical`、`internal/database/persistence`、`internal/database/schemadiff`を照合しました。PostgreSQL側の挙動については公式ドキュメントの`pg_class`、`pg_index`、`pg_constraint`、`information_schema.columns`を参照しています。

## 結論

M0.26で修正したPRIMARY KEY backing indexの二重計上以外にも、今後の実DB利用で意味上の問題になり得る境界が残っています。

大きく分けると、次の3種類です。

1. **PostgreSQL provider側の観測方法を直せばよいもの**
   - materialized viewの列観測
   - `information_schema.columns`の権限制約によるTable/Column観測の不整合

2. **provider-neutral Physical Schemaの表現力を先に決める必要があるもの**
   - UNIQUE constraintとexplicit unique indexの区別
   - partial index、expression index、INCLUDE列、indexの有効状態
   - partitioned tableとchild partitionの関係

3. **現時点では既知の制約として維持してよいもの**
   - Foreign KeyのON DELETE / ON UPDATE、deferrable等のPostgreSQL固有option
   - domain、collation、型modifier、default expressionの詳細なsemantic normalization

以下、それぞれを整理します。

## 1. Table / View / Materialized View

### 現行の観測

`Tables()`は`pg_class.relkind`の`r`、`p`、`v`、`m`を対象にしています。

- `r`: ordinary table
- `p`: partitioned table
- `v`: view
- `m`: materialized view

TEI側では`v`を`view`、`m`を`materialized-view`、それ以外を`table`へ写しています。

### 問題1: materialized viewの列

`Columns()`は`information_schema.columns`を使っています。PostgreSQL公式ドキュメントでは、このviewはtable columnまたはview columnを提供するものとして説明されています。materialized viewは`pg_class`では独立した`relkind='m'`です。

そのため現在の実装では、`Tables()`ではmaterialized viewを観測できる一方、`Columns()`ではその列を観測できず、**Tableだけ存在してColumnsが空になる可能性**があります。

これはprovider-neutral modelの不足ではなく、PostgreSQL providerの観測経路の不整合です。

### 問題2: 権限による列欠落

`information_schema.columns`は、現在のDBユーザーがアクセスできる列だけを返します。一方、`Tables()`は`pg_class`を直接参照しています。

このためmetadata read権限が限定された接続では、relation自体は`Tables()`に現れるのに、対応する列の一部または全部が`Columns()`に現れない可能性があります。結果として、実際には存在する列を`drop-column`相当のdriftとして誤認する危険があります。

### 推奨

PostgreSQLのrelation / column観測は、`pg_class`、`pg_namespace`、`pg_attribute`、`pg_type`を基底にした一貫したcatalog queryへ寄せることを推奨します。

この変更はPostgreSQL provider内で閉じられるため、ADRを必須とはしません。ただしactual PostgreSQLを使ったConformance testで、少なくともmaterialized viewを一件観測するべきです。

## 2. Physical type

### 現行の観測

`Columns()`は`udt_schema.udt_name`をPhysicalTypeへ入れています。Desired側もPostgreSQLでは`pg_catalog.int8`、`pg_catalog.varchar`、`pg_catalog.jsonb`等を生成するため、M0.26の対象範囲では整合しています。

### 残る制約

一方、現在のPhysicalTypeだけでは、次の情報を構造化していません。

- varchar等の長さ
- numericのprecision / scale
- domain型のidentity
- collation
- generated column等の属性

これらを文字列へ無理に埋め込んで比較するより、必要になった時点でprovider-neutral modelとして意味を定義した方が安全です。

現段階では「M0.26/M0.27で扱うStorageTypeの範囲に限定した既知の制約」として維持してよいと判断します。

## 3. nullable / default

`Nullable`は現行Physical Schemaで明示され、Desired / Actual比較に利用されています。

`Default`はactual PostgreSQLの`column_default`文字列をそのまま保持し、Schema Diffではtrim後の文字列一致で比較します。一方、Persistence Mapping側にはdefault authoringがありません。

したがって現時点では、defaultを持つactual DBを「Desiredと同じ意味」と表現できません。また、同じdefaultでもPostgreSQLが返すcastやsequence qualification等の文字列表現差が将来問題になる可能性があります。

ただし、default authoring自体が未設計なので、今すぐprovider側だけでcanonicalizeするのは早いと判断します。defaultをDesiredへ導入する際に、expression identityとnormalizationを別Issue / ADRで決めるべきです。

## 4. Primary Key

PRIMARY KEYは`pg_constraint.contype='p'`から列順を保持して観測し、Physical Schemaでは`PrimaryKey []string`へ写しています。

M0.26で、PRIMARY KEYが自動生成するbacking indexを通常Indexとして二重計上しないよう、`Indexes()`から`indisprimary`を除外しました。

この判断はprovider-neutral modelと整合しています。

## 5. UNIQUE constraintとunique index

### 現行の観測

PostgreSQLではUNIQUE constraintを作るとunique B-tree indexが自動生成されます。`pg_constraint`ではUNIQUE constraintは`contype='u'`として独立したconstraintですが、現行`Indexes()`は`indisprimary`だけを除外しているため、そのbacking indexを通常の`physical.Index{Unique:true}`として観測します。

### 問題

現在のPhysical Schemaには`UniqueConstraint`がなく、`Index`だけがあります。そのため次の二つが同じ形へ畳まれます。

- `UNIQUE (a, b)`というtable constraint
- `CREATE UNIQUE INDEX ... (a, b)`というexplicit index

PostgreSQLでは両者が近い実装を持ちますが、DDL上の意図、constraint identity、他DBMSへの可搬性は同じではありません。

これは単純に`indisunique`を除外すればよい問題ではありません。除外するとUNIQUE constraintの存在自体を失います。反対に現在のままではconstraintとindexを同一視します。

### 分類

**provider-neutral Physical Schemaの表現不足**です。

Unique Constraintを独立してmodel化するか、Indexを物理実装として扱い続けるかをADRで決めてから変更するべきです。

## 6. Foreign Key

M0.27でForeign KeyはDesired / Actual / Diffへ統合されました。source列、参照先qualified table、参照列の構造で比較し、constraint名だけの差はdriftとしません。

現行PostgreSQL観測は、このM0.27の範囲では整合しています。

ただし次は意図的に未表現です。

- ON DELETE / ON UPDATE
- MATCH type
- deferrable / initially deferred
- validation state

これらはM0.27の非目標として明示済みなので、今回の監査では不具合とは扱いません。

## 7. Index

Indexは今回の監査で最も多くの意味圧縮が残っている箇所です。

### 7.1 partial index

`pg_index.indpred`はpartial index predicateを保持しますが、現行`Indexes()`は取得していません。

そのため

```sql
CREATE INDEX ... ON orders(customer_id) WHERE deleted_at IS NULL;
```

と、同じ列のfull indexがTEI上では同じIndexに見え得ます。

これは意味のある差を落としています。

### 7.2 expression index

`pg_index.indkey`で0になっているkeyはexpressionです。現行実装は`pg_get_indexdef(indexrelid, column_no, true)`を`Columns`へ格納するため、expression自体の文字列はある程度保持できます。

ただしPhysical Schemaのfield名は`Columns`であり、Persistence MappingのIndexはAttribute IDだけを指定します。つまりactual側ではexpressionを観測できても、Desired側で同じ意味をauthoringできません。

### 7.3 INCLUDE列

現行queryは`indnkeyatts`までだけを取得しており、INCLUDE列は観測しません。

これは「検索keyとしてのindex identityには含めない」という解釈も可能ですが、covering indexとしての物理性能・DDL identityには影響します。TEIがIndexをschema convergence対象として扱うなら、無視してよいかは設計判断が必要です。

### 7.4 index lifecycle state

PostgreSQLの`pg_index`には`indisvalid`、`indisready`、`indislive`があります。現行queryはこれらを見ていません。

したがって、無効なindexやbuild / drop途中のindexを、完成した通常indexと同じようにActualへ入れる可能性があります。

単に無効indexをfilterすると、今度は「同名indexは存在するがdesired semanticsを満たさない」という運用上重要な状態を失います。これはprovider側filterだけでなく、Actual Schemaがoperational stateを持つべきかという設計論点です。

### 分類

partial predicate、INCLUDE、expression、lifecycle stateはいずれも、**provider-neutral Index modelの表現不足を含む**ため、まとめて設計Issueとして扱うことを推奨します。

## 8. Partitioned Table

`Tables()`は`relkind='p'`を通常の`TableKindTable`へ写します。またordinary child partitionは`relkind='r'`として別relationです。

現行Physical Schemaにはpartition hierarchyがありません。そのためpartitioned parentとchild partitionsが、互いに無関係な複数Tableとして見えます。

Schema DiffはActualにだけ存在するTableを`drop-table`として扱うため、Desiredがparentだけを表現する場合、child partitionを不要なactual-only Tableと判定する可能性があります。

これはPostgreSQL providerのfilterだけで決めるべきではありません。partitionをPhysical Schemaの一部として追跡するのか、provider implementation detailとして折り畳むのかで意味が変わります。

**provider-neutral model / drift semanticsの設計判断が必要**です。

## 9. orderingのdeterminism

現行SQLはrelation、column、constraint、indexを明示的にORDER BYし、`SchemaSnapshot.Normalize()`もTable / Column / Foreign Key / Indexの順序を安定化しています。

M0.27時点で、通常のTable / Column / PK / FK / Indexについてorderingの大きな不安定要因は見つかりませんでした。

## 推奨する次の作業

### Provider内で先に直せるもの

1. materialized viewを含むcolumn catalog queryへ統一する
2. metadata privilegeによってTableだけ見えてColumnが欠ける状態をConformanceで検出する

### ADRを先に置くもの

1. UNIQUE constraintとunique indexをどう区別するか
2. Index predicate / expression / INCLUDE / lifecycle stateをPhysical Schemaでどう表すか
3. partition hierarchyをSchema Diffでどう扱うか

### 将来の必要時に扱うもの

1. column default expressionのauthoring / normalization
2. domain、collation、typmod等の型詳細
3. FK action / deferrability等の追加意味

## 監査時点の判断

M0.26/M0.27のReference vertical slice自体は、限定した通常table / PK / explicit index / FKの範囲では成立しています。

一方、この範囲をそのまま「PostgreSQLの一般的なschema introspection」とみなすのは早いです。特にmaterialized view、partitioning、UNIQUE constraint、advanced indexを実DBで使う場合は、今回切り出した設計論点を先に閉じるべきです。
