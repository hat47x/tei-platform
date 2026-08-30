# ADR-0007: actual database driverの組み立てをProvider Coreから分離する

- 状態: 採用
- 日付: 2026-08-31
- 関連Issue: #35
- 前提: M0.26 actual PostgreSQL integration

## 背景

TEIのDatabase lifecycleは、Persistence MappingからDesired Physical Schemaを導出し、外部DBから観測したActual Physical Schemaと比較することで、差分を明示する構造を持っています。

M0.26では、この境界を実際のPostgreSQLへ接続します。既存のPostgreSQL providerは`database/sql`を利用する`SQLCatalogReader`を持っており、catalog queryからPhysical Schema Snapshotを組み立てる責務はすでに分離されています。

ここでactual PostgreSQLへ接続するためには、`database/sql` compatible driverをどこかで登録し、接続文字列やcredentialを渡す必要があります。

実装上は、PostgreSQL provider package自身がpgxやlib/pq等のdriverをimportする方法もあります。しかし、その形にすると「PostgreSQLの意味を読み取るprovider」と「このHostでどのdriverを使って接続するか」という別の責務が同じpackageへ混ざります。将来、別driverへ交換したい場合や、Host側でTLS、connection pool、credential deliveryを制御したい場合にも、Core側へ実行環境固有の都合が入りやすくなります。

## 判断

actual database driverの選択と`database/sql`への登録、DSN、credential、接続poolの組み立ては、**Reference integrationまたはHost assembly側の責務**とします。

Database provider Coreは、`database/sql`等の安定したHost境界を受け取り、DB固有のmetadata解釈だけを担当します。

概念上は次の分離です。

```text
Host / Reference assembly
  - driver選択
  - driver登録
  - DSN
  - credential
  - TLS / pool等の接続設定
          ↓
      database/sql
          ↓
PostgreSQL Provider
  - catalog query
  - PostgreSQL metadataの意味解釈
  - Actual Physical Schema Snapshot生成
          ↓
Provider-neutral Database lifecycle
  - Desired
  - Actual
  - Diff
  - Migration Proposal
```

M0.26のReference integrationでは、`lib/pq`を`database/sql` driverの具体例として利用します。これはTEI Coreがlib/pqを標準driverとして固定するという意味ではありません。将来、Reference assemblyをpgx/stdlib等へ交換しても、Provider Coreの意味境界は変えないことを意図しています。

## credentialと接続情報

DSNやcredentialはCanonical World、Database Contract、Physical Schema Snapshot、Schema Diff、Durable Deployment artifactへ保存しません。

Reference toolではenvironment variable等のHost-side入力として受け取ります。

Physical Schema Snapshotに残すのは、DBから観測したschemaの意味情報です。どのpasswordで接続したか、どのsocketやtemporary service containerへ接続したかはSnapshotの意味ではありません。

## fixture setupとの境界

GitHub Actionsではactual PostgreSQL service containerを起動し、Reference integration用fixtureを準備します。

このfixture setupは、TEIのschema migration能力やdeployment executorとして扱いません。M0.26で検証するのは、既存schemaをmetadata-readし、DesiredとActualを比較する境界です。

したがって、CIが`CREATE TABLE`等を実行してfixtureを作ることと、TEIが本番DBへschema変更を自動適用できることを同一視しません。

## provider固有の観測差

actual DBでは、default expression、型名、暗黙index、schema qualificationなど、in-memory fixtureでは見えにくかったDB固有表現が現れます。

この差を扱う場合は、次の順で判断します。

1. PostgreSQL上で同じ意味なのかを確認する
2. providerのmetadata解釈でnormalizationすべきかを検討する
3. provider-neutral Physical Schemaへ持ち上げる意味があるかを検討する
4. testを通すためだけの文字列置換は行わない

つまり、actual integrationは外部DBの表現へCanonical modelを従属させるためではなく、TEIの意味境界に不足がないかを検証するために使います。

## 採用しなかった案

### PostgreSQL providerが具体driverを直接importする

実装は短くなりますが、driver選択とmetadata意味解釈が結合します。Hostごとの接続policyやdriver交換をCoreへ持ち込みやすいため採用しません。

### DSNをDatabase modelへ保存する

再現性が高く見えますが、credentialや環境固有locatorがCanonical / Durable artifactへ漏れる原因になります。Database identityと接続先locatorは別に扱うため採用しません。

### actual DB integrationをUnit testだけで代替する

catalog readerの局所的な動作は検証できますが、PostgreSQLが実際に返すmetadata表現やdriver境界を確認できません。M0.26の目的を満たさないため採用しません。

## 影響

- PostgreSQL providerは引き続き`database/sql`境界を中心に保てる
- Reference toolや将来のHost adapterがdriver、DSN、credentialを組み立てる
- lib/pqはM0.26 Reference integrationの具体実装であり、Canonical modelの一部ではない
- actual PostgreSQLを使うCIを追加しても、production接続policyを実装済みとは扱わない
- 将来MySQLや他DBをactual integrationする場合も、同じHost assembly原則を適用できる

このADRはschema変更の自動適用、migration executor、production-grade connection pooling、TLS policy、HA/failoverを決めるものではありません。それらは必要になった時点で別のIssue / ADRとして扱います。
