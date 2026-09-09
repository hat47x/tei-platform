# ADR-0027: Partition treeを伴うPostgreSQL DropTableをprovider execution preconditionでfail closedする

- 状態: 採用
- 関連Issue: #524
- 関連ADR: ADR-0026

## 背景

ADR-0026では、PostgreSQLのpartition childをprovider-owned realization observationとして保持し、known childをgeneric Schema Driftの独立したconvergence subjectから外す方針を採用しました。一方、partitioned parentはgeneric table identityを担うため、generic convergence subjectのまま残します。

その結果、desiredからparent tableが消えれば、generic `schemadiff.Compare`は正当に`drop-table` intentを導出し得ます。しかし、そのintentがPostgreSQL上でそのまま安全に実行可能であるとは限りません。

現行実装では、`migration.Plan()`がSchema Diffをprovider rendererへ渡し、PostgreSQL rendererは`DropTable`を次のSQLへ変換します。

```sql
DROP TABLE schema.table;
```

この経路にはpartition treeの有無を判断する入力がありません。

また、Migration Definitionには`BasedOnActualFingerprint`がありますが、ADR-0026でpartition relationをgeneric `physical.SchemaSnapshot`へ入れないと決めたため、generic snapshot fingerprintだけでは「このparentにpartition childが存在しない」ことを証明できません。

PostgreSQLのdeclarative partitioningでは、partitionはparentと結び付いたotherwise-ordinary tableであり、partition自身を`DROP TABLE`することは大量の行を一度に削除し得ます。`DROP TABLE`の既定動作は`RESTRICT`ですが、RESTRICTは一般的なdependency rejectionであり、TEIがpartition treeを理解した上でdestructive operationを承認した証拠ではありません。

このADRでは、generic `drop-table` intentとPostgreSQL execution readinessを分離し、partition treeを伴うdestructive DDLをfail closedにする境界を定めます。

## 判断

### 1. generic `DropTable`はintentであり、実行許可ではない

Schema Diffが生成する`DropTable`は、desired / actualのgeneric構造差を表します。

```text
generic DropTable
  = desiredからtable identityが消えた
  != PostgreSQLで安全にDROPできる
```

PostgreSQL providerは、`DropTable`をSQLへrenderする前にprovider-specific execution preconditionを評価します。

### 2. partition treeの有無をprovider-owned observationで評価する

PostgreSQLの`DropTable` readinessは、ADR-0026で定めたpartition relation observationを使って評価します。

最低限、対象relationについて次を区別します。

```text
A. targetにattached partition child / descendantがある
   -> blocked

B. target自身がknown partition childである
   -> blocked / invariant violation

C. partition observationが取得できない、または完全性を証明できない
   -> unresolved / not executable

D. provider observation上、targetがpartition childではなく、child / descendantも存在しない
   -> partition preconditionはpass
```

A〜Cを通常のSQLへ落としません。

sub-partitioningではdirect childだけでなくdescendant chain全体を対象にします。

### 3. generic snapshot fingerprintだけをpartition safety proofにしない

`BasedOnActualFingerprint`はmigrationがどのactual snapshotを基礎にしたかを追跡する重要な情報です。しかしpartition relationをgeneric snapshotへ含めないM0.xでは、それだけでpartition tree absenceを証明できません。

したがってimplementationは、次のどちらかを満たす必要があります。

- planningで使用したprovider-owned partition observationのlineage / digestを保持し、apply時にその前提を検証する
- apply直前にprovider observationを再取得し、同じpreconditionを再評価する

exactなartifact fieldやdigest形式はこのADRでは固定しません。

ただし、planning時に一度「childなし」と判定しただけで、その後のapplyを無条件に許可する設計にはしません。

### 4. partition treeが存在するparentのDROPを自動展開しない

対象parentにpartition treeが存在する場合、TEIはgeneric `DropTable`から次を自動生成しません。

- child partitionの個別`DROP TABLE`
- `ALTER TABLE ... DETACH PARTITION`
- descendantの再帰的drop
- retention / archive処理
- `DROP TABLE ... CASCADE`

これらはそれぞれ独立したデータ保持・lifecycle intentを必要とします。

partition childの削除はimplementation detailではなく、実データを大量に失い得るdestructive operationです。

### 5. `CASCADE`を自動付与しない

PostgreSQLの`CASCADE`は依存objectを再帰的に削除し得るため、generic Schema Diffの不足を補う安全策として使いません。

provider renderer / executorは、`DROP TABLE ... RESTRICT`相当の失敗を受けても、機械的に`CASCADE`へretryしません。

`CASCADE`が将来必要になる場合は、削除されるdependencyを明示した別のauthoritative operationとして設計します。

### 6. RESTRICTはdefense-in-depthであり、preconditionの代替ではない

plain `DROP TABLE`はPostgreSQLでは既定でRESTRICTです。このDatabase側安全停止は維持します。

ただしTEIは、RESTRICTが失敗するまでpartition treeを見ない設計にはしません。

```text
TEI provider precondition
  ↓ pass only
plain DROP TABLE (default RESTRICT)
  ↓
Database dependency checks
```

DatabaseがRESTRICTで拒否した場合はexecution errorとして停止し、その場でoperationを強めません。

### 7. planning / apply / Evidenceでblocked reasonを保持する

partition preconditionにより実行不可となった場合、単なるrenderer error文字列だけではなく、少なくとも次を追跡可能にします。

```text
target relation
reason
  partition-tree-present
  target-is-partition-child
  partition-observation-unavailable
  partition-observation-stale-or-unverified
observed child / descendant relation identities when available
```

relation identityはschema + relation nameを基本とし、catalog OIDをCanonical identityやEvidence正本にしません。

planは「generic diffにはDropTableがあるが、provider executionはblocked」であることを表現できる必要があります。

applyはblocked planを実行しません。

Evidenceは「実行しなかった理由」を残せますが、blocked Evidenceをmigration成功へ読み替えません。

### 8. offline planningでpartition observationが無い場合もfail closedする

PostgreSQL providerを指定したoffline migration planningがgeneric physical snapshotだけを受け取り、partition sidecarを受け取れない場合、その経路はpartition-sensitive `DropTable`のexecution readinessを証明できません。

この状態で従来どおりSQLをrenderするのではなく、provider precondition unresolvedとして停止します。

CLI / artifactへprovider observation sidecarを渡す契約はfollow-up実装で定めます。

## Pressure

follow-up実装では最低限次を固定します。

### parentにpartition treeがある

```text
diff:
  DropTable public.orders

observation:
  public.orders_2026 -> public.orders

=> provider execution blocked
=> DROP TABLE public.orders をrenderしない
=> child DROP / DETACH / CASCADEを生成しない
```

### sub-partition descendantがある

```text
orders_2026 -> orders
orders_2026_q1 -> orders_2026

DropTable orders
=> descendant chainを含めblocked
```

### target自身がpartition child

```text
diff:
  DropTable public.orders_2026

observation:
  orders_2026 -> orders

=> invariant violation / blocked
```

ADR-0026のprojection後は通常このdiffは生成されません。下流で現れた場合も黙って実行しないPressureにします。

### ordinary tableでpartition treeが無い

```text
diff:
  DropTable public.audit_log

complete provider observation:
  audit_log is not child
  no child / descendant edge points to audit_log

=> partition precondition pass
=> plain DROP TABLE "public"."audit_log";
=> CASCADEなし
```

### partition observationが無い

```text
PostgreSQL DropTable
+ generic snapshot only
+ partition observation unavailable

=> unresolved / not executable
```

### stale / unverified observation

```text
planning observationではchildなし
apply時にそのobservation lineageを検証できない

=> executeしない
=>再観測または明示的な再計画が必要
```

### Database RESTRICT failure

```text
precondition pass
DROP TABLE実行
PostgreSQLがdependencyで拒否

=> execution error
=> CASCADE retryなし
```

## 実装責務の配置

責務は次の順で置きます。

```text
Schema Diff
  ↓
generic DropTable intent
  ↓
PostgreSQL destructive readiness assessment
  + provider-owned partition observation
  ↓ pass only
PostgreSQL SQL rendering
  ↓
plain DROP TABLE
  ↓
execution-time precondition verification
  ↓
Database RESTRICT checks
```

`schemadiff.Compare`へPostgreSQL-specific safety fieldを追加しません。

`physical.Table`へpartition safety flagを追加しません。

provider-specific readinessをgeneric Schema Diffから分離することが、このADRの中心です。

## 検討した代替案

### A. plain DROP TABLEを常にrenderし、PostgreSQL RESTRICTに任せる

却下します。

Database dependency checkは重要ですが、TEIが既知のpartition realizationを見ないままdestructive intentをSQLへ昇格させることになります。またRESTRICT failureの有無は、TEIのauthoring intentやデータ保持判断を表しません。

### B. partition treeを見つけたらchildを先にDROPする

却下します。

partition childは大量データを保持し得ます。generic parent removalからchild data deletionを暗黙導出しません。

### C. partition treeを見つけたらCASCADEを付ける

却下します。

削除範囲を拡張して問題を隠すため、安全境界として逆方向です。

### D. provider-owned preconditionとしてfail closedする

採用します。

generic intentを保持したまま、provider realizationを知っている層だけがexecution readinessを判断できます。

## 非目標

- partition lifecycle authoring
- retention / archive policy
- DETACH PARTITIONの自動生成
- partition tree削除の承認UI
- generic Physical Schemaへのpartition field追加
- PostgreSQL以外のproviderのdestructive policy一般化
- transaction / lock strategyの詳細固定

最後のlock strategyは、apply-time revalidationを実装するときにactual PostgreSQL behaviorをPressureしながら具体化します。

## 影響

短期的には、PostgreSQL `DropTable`の一部が従来よりfail closedになります。特にgeneric snapshotだけを入力するoffline plannerは、partition observation contractが接続されるまでdestructive DropTableをrenderできなくなります。

これは機能後退ではなく、ADR-0026でgeneric modelから意図的に分離したprovider factを、destructive execution時だけ無視しないための整合化です。

一方、ordinary tableでpartition treeが存在しないことをprovider observationから確認できれば、plain `DROP TABLE`の既存renderingは維持できます。

このADRはexecution safety contractの採用であり、source実装済み、actual PostgreSQLでPressure済み、Gate B Readyを意味しません。

## 参考

- PostgreSQL 18 / current: DROP TABLE
  - https://www.postgresql.org/docs/current/sql-droptable.html
- PostgreSQL 18 / current: Table Partitioning
  - https://www.postgresql.org/docs/current/ddl-partitioning.html
- PostgreSQL: pg_depend
  - https://www.postgresql.org/docs/current/catalog-pg-depend.html
