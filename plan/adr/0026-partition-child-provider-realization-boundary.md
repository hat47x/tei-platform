# ADR-0026: Partition childをprovider-owned realizationとしてgeneric Schema Driftから分離する

- 状態: 採用
- 関連Issue: #43

## 背景

TEIのPhysical Schemaは、Databaseの物理構造をTarget非依存に観測・比較するためのモデルです。現在の`physical.SchemaSnapshot`はtable / view / materialized view、column、constraint、index等を保持しますが、partition hierarchyは持ちません。

一方、PostgreSQLのdeclarative partitioningでは、partitioned tableは論理的には一つのtableでありながら、自身はstorageを持たず、実データは関連付けられたpartitionに置かれます。partitionはotherwise-ordinary tableとしてcatalog上に現れ、partition自身がindex、constraint、default等を持つこともできます。また、parent-child関係はPostgreSQL catalogで明示的に観測できます。

現行Schema Diffはdesired / actualのtableをqualified nameで対応付け、actualにだけ存在するtableを`drop-table`として扱います。このため、PostgreSQL providerがpartition childを通常の`physical.Table`として並べるだけでは、TEIがpartition authoring semanticsを持っていないにもかかわらず、providerが所有するpartition realizationをgeneric driftとして削除対象に見せる危険があります。

ここでは、partitioningをgeneric Physical Schemaへ本格導入する前に、M0.xで安全に観測・比較する境界を決めます。

## 判断

### 1. M0.xではgeneric Physical Schemaにpartition hierarchyを追加しない

現時点では、PostgreSQLで得られた具体的なpartition semanticsだけを根拠に、provider-neutralな`physical.Table`へpartition parent / child、partition key、bound等を追加しません。

理由は次のとおりです。

- cross-providerで何を同じpartition intentとみなすかがまだ十分に検証されていない
- providerごとにpartitioningのDDL、lifecycle、constraint/index inheritance、運用単位が異なり得る
- PostgreSQL catalog fieldをそのままCoreへ持ち込むと、観測実装がgeneric contractを支配する
- partition authoring、repartition、attach/detach等のmutation semanticsを同時に設計しない限り、中途半端なgeneric fieldだけがauthorityになりやすい

将来generic partition intentを導入する場合は、複数providerで意味境界を比較した別ADRを必要とします。

### 2. partition factsはprovider-owned observationとして保持する

partition childをgeneric comparisonから外すために、catalog観測から事実そのものを捨ててはいけません。

PostgreSQL providerは少なくとも次をprovider-owned realization observationとして保持できる必要があります。

```text
relation identity
  = schema + relation name

partition relation
  child identity
  parent identity
  lifecycle / attachment state when safely observable

provider detail
  partition strategy / key / bound等
  = provider側で意味と安定性を確認できる範囲だけ
```

このobservationは`physical.Table`のgeneric fieldを拡張して保持するのではなく、PostgreSQL provider側の型またはprovider realization用のsidecarに置きます。

`relispartition`、`relpartbound`等のcatalog column名をそのままprovider-neutral contractにはしません。catalog固有表現からTEIが必要とするprovider-owned observationへnormalizationします。

### 3. generic convergence subjectとprovider realizationを分ける

M0.xのgeneric Schema Driftでは、PostgreSQL providerが「あるrelationはpartition childである」と確実に識別できる場合、そのchildを独立したgeneric convergence subjectとして扱いません。

概念的には次の境界です。

```text
PostgreSQL catalog
  ↓
provider observation
  ├─ generic physical relations
  └─ partition realization facts
          ↓
provider-owned convergence projection
          ↓
generic physical.SchemaSnapshot
          ↓
schemadiff.Compare
```

provider-owned convergence projectionは、partition childの存在を消去する処理ではありません。generic diffへ渡す対象を選ぶ処理であり、元のprovider observationにはchild relationとparent-child関係を残します。

### 4. known partition childをgeneric `drop-table`へ変換しない

actualにpartition childが存在し、desired generic modelに同名tableが存在しない場合、generic Schema Diffはその事実だけから`drop-table`を生成してはいけません。

これは「driftなし」と断定することとも異なります。

```text
known partition child
  != standalone unexpected table
  != generic drop candidate
```

partition childの追加・削除・attach / detach・bound変更等をTEIがauthoritativeに比較できるようになるまでは、それらはprovider realization observationとしてinspectableに残します。

### 5. desiredに同名standalone tableがある場合は黙って同一視しない

actual relationがpartition childである一方、desired generic Physical Schemaに同じqualified nameの通常tableが明示されている場合、TEIは両者を名前だけで同一tableとして比較してはいけません。

この状態は、generic intentとprovider realization roleの対応を現在のcontractだけでは確定できないため、provider-realization ambiguityとしてfail closedします。

実装ではstructured unresolved / diagnosticとして表現し、通常のtable equalityやmigration operationへ落とさないものとします。

### 6. parent tableはgeneric convergence subjectのままにする

partitioned parentは、logical database structureとして利用者が参照するtable identityを担うため、generic convergenceから除外しません。

ただし、desiredからparentが消えたことによるgeneric `drop-table` intentと、そのdropをPostgreSQL上で安全に実行できることは別問題です。

partition treeが存在するparentについて、provider migration renderer / plannerはgeneric `drop-table`だけを根拠にchildを個別削除したり`CASCADE`を自動付与してはいけません。PostgreSQLはdependencyを持つDROPを既定でRESTRICTし、partition maintenanceではchild tableのDROP自体が大量データ削除になり得ます。

したがってpartition treeを伴うdestructive DDLは、provider-specific safety policyが設計・実装されるまで自動実行可能へ格上げしません。

### 7. partition key / bound / authoringはこのADRではauthorityにしない

partition strategy、partition key、partition bound、sub-partitioning、attach / detach、repartition、partition lifecycle policy等は重要ですが、このADRの目的は「未設計のrealizationをgeneric driftとして誤削除しないこと」です。

それらのdesired authoring / mutation contractは別Issue・別ADRで扱います。

## identityの扱い

M0.xでは次を区別します。

```text
generic table identity
  = schema + table name

provider relation identity
  = schema + relation name

provider partition edge
  = child relation identity -> parent relation identity
```

同じ文字列identityを持つことは、generic standalone tableとprovider partition childのsemantic roleが同じことを意味しません。roleの差をprovider observationで保持します。

sub-partitioningでは一つのrelationが、上位parentから見ればchildであり、下位partitionから見ればparentになり得ます。このため単純な`isPartition bool`だけではなく、relation間のedgeとして表現できる設計を優先します。

## 検討した代替案

### A. generic `physical.Table`へpartition fieldsを直ちに追加する

却下します。

PostgreSQLの具体的なcatalog semanticsを早期にprovider-neutral contractへ固定することになり、他providerとの意味比較とdesired authoring contractが不足しています。

### B. provider-owned realization observationを保持し、generic convergence projectionを分離する

採用します。

観測事実を失わず、generic Schema Diffに未設計のpartition mutation authorityを与えません。将来generic partition intentを追加しても、provider observationをEvidenceとして再利用できます。

### C. catalog readerの段階でpartition childを完全に読み飛ばす

却下します。

generic `drop-table`誤判定は防げますが、実DBに存在する重要な物理事実が観測・Evidenceから消えます。問題を解決するのではなく不可視化します。

### D. partition childを通常tableとして比較し、actual-onlyなら`drop-table`にする

却下します。

TEIがpartition lifecycleをauthoritativeに設計していない状態でdestructive operationを導出するため、安全ではありません。

## 実装への要求

このADRを実装するfollow-upでは、少なくとも次をPressureします。

1. PostgreSQL catalog observationがparent-child relationを保持する
2. generic Physical Schemaへraw PostgreSQL catalog fieldを追加しない
3. actual-only known partition childから`DropTable`を生成しない
4. child observation自体はinspect可能なまま残る
5. desired standalone tableとactual partition childの同名衝突をfail closedする
6. sub-partitioningを単純booleanで潰さない
7. partition treeを伴うdestructive DDLをgeneric diffだけで自動安全とみなさない

## 非目標

- partitioned tableのauthoring syntax
- partition key / boundのprovider-neutral Canonical model
- partition作成・attach・detach・repartitionのmigration generation
- retention / archival policy
- provider間でのpartition feature parity
- PostgreSQL catalog queryそのものをこのADRで固定すること

## 影響

短期的には、generic Physical Schemaを小さく保ったままPostgreSQL partitionの誤ったdestructive driftを防ぐための実装境界が明確になります。

一方、partition realizationの完全なdesired/actual convergenceはまだ提供しません。TEIが知らないことを`drop-table`へ変換するより、provider observationとして保持して未解決境界を明示することを優先します。

この判断は設計境界の採用であり、runtime実装済み・actual PostgreSQL Evidence取得済み・Gate B Readyを意味しません。

## 参考

- PostgreSQL current documentation: Table Partitioning
  - https://www.postgresql.org/docs/current/ddl-partitioning.html
- PostgreSQL current documentation: `pg_inherits`
  - https://www.postgresql.org/docs/current/catalog-pg-inherits.html
- PostgreSQL current documentation: `DROP TABLE`
  - https://www.postgresql.org/docs/current/sql-droptable.html
