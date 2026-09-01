# M0.33 Offline Physical Realization Policy Conformance

M0.33では、Persistence MappingからDesired Physical Schemaを生成するときに必要となるprovider固有の知識を、Persistence Coreおよびconnected runtime Provider SPIから分離しました。

この文書では、その境界が実装上どのように保たれているかを確認します。

## 対象となる境界

Desired realizationは次の三層に分かれます。

```text
Persistence Mapping / persistence.RealizeDesired
          |
          | PhysicalRealizationPolicyだけを見る
          v
provider-owned offline policy
          ^
          |
          +-- postgres.RealizationPolicy
          +-- oracle.RealizationPolicy

外側のcomposition
          |
          +-- database/realization
              provider ID -> policy -> persistence.RealizeDesired
```

Persistence CoreはPostgreSQLやOracleのpackageをimportしません。DBMSごとのphysical type、Indexの既定値、接続なしで確定できる正規化規則はprovider packageが所有します。

## offlineで完結すること

`PhysicalRealizationPolicy`には、次の情報を渡しません。

- `context.Context`
- `DatabaseInstance`
- DB接続
- 認証情報
- runtime Provider object

したがって、Desired Physical Schemaの生成だけを目的としてlive databaseへ接続する必要はありません。

Workspace load / validationについても、M0.33によってDB接続は追加されません。Persistence authoringを読み込む処理と、必要なprovider向けにDesired Physical Schemaを投影する処理は別のままです。

## providerが所有する責務

M0.33で`PhysicalRealizationPolicy`へ移した責務は、現在必要な次の範囲に限定しています。

- provider-neutralな`StorageType`からphysical typeへの変換
- Index access methodの既定値
- unique indexにおけるNULL semanticsの既定値
- 接続なしで確定できるIndex keyのdeterministic normalization

PostgreSQLでは、従来Persistence Coreが持っていた次の規則をpolicyへ移しました。

- `int64 -> pg_catalog.int8`
- `varchar -> pg_catalog.varchar`
- `text -> pg_catalog.text`
- `datetime-tz -> pg_catalog.timestamptz`
- `decimal -> pg_catalog.numeric`
- `json -> pg_catalog.jsonb`
- Index methodの既定値を`btree`とする
- unique indexの既定NULL treatmentを`distinct`とする
- B-treeの`ASC NULLS LAST`をeffective defaultへ正規化する
- B-treeの`DESC NULLS FIRST`をeffective defaultへ正規化する
- `pg_catalog.default` collationを既定表現へ正規化する

Oracleについても、既存のstorage type変換をOracle側のoffline policyへ移しています。

## compositionの責務

`internal/database/realization`は、provider IDから具体的な`PhysicalRealizationPolicy`を選び、Persistence realizationへ渡すだけのcomposition境界です。

ここにprovider switchを集約することで、CLIやintegration toolは具体policy型を知る必要がありません。一方、Persistence Coreへprovider switchを戻すこともありません。

未対応providerはgenericな既定値へ縮退させず、明示的なエラーとします。意味を確定できないproviderを、それらしい共通値へ変換することはしません。

## Unit Conformance

`internal/database/realization/realize_test.go`で、少なくとも次を確認します。

1. PostgreSQL IDからPostgreSQL offline policyを取得できる
2. Oracle IDからOracle offline policyを取得できる
3. generic providerにはbuilt-in policyを割り当てない
4. 同じPersistence intentでもPostgreSQLでは`pg_catalog.int8`、Oracleでは`NUMBER(19)`へ投影される
5. PostgreSQLのIndex method / unique NULL treatmentの既定値が維持される
6. PostgreSQL B-treeのASC / NULL ordering / default collationの正規化がcomposition経由でも維持される

Persistence package自身のテストでは、具体provider policyを直接注入します。これにより、Persistence realizationがprovider IDのswitchではなくportへの依存として動作していることも確認します。

## Actual PostgreSQL Conformance

既存の`reference-postgres-integration`は、M0.33から`database/realization`を通じてDesired Physical Schemaを生成します。

そのDesiredと、PostgreSQL 17の実DBをcatalog introspectionして得たActual Physical Schemaを比較し、baseline driftが0であることを引き続き確認します。

このため、policy分離後も少なくとも次の既存意味が崩れていないことを実DBで確認できます。

- physical type
- Primary Key / UNIQUE constraint / Foreign Key
- simple / unique index
- Advanced Indexの観測
- PostgreSQLのeffective default normalization

## runtime Provider SPIとの分離

`database/provider.Provider`はActual databaseとの対話を担当するruntime SPIであり、M0.33では変更しません。

Desired realizationを行うために`Capabilities`や`Introspect`を呼び出すこともありません。offline policyとconnected runtime providerを同じ万能interfaceへまとめないことをConformance上の境界とします。

## ProviderExpressionとの関係

ADR-0012 / M0.32で、ProviderExpressionは`authored`と`provider-normalized`を区別し、比較を三値化しました。

M0.33では、任意のauthored expressionをprovider-normalized expressionへ変換するparser / deparserは追加しません。provider versionやserver round-tripを含む条件が未確定なためです。

`PhysicalRealizationPolicy`は、現時点で確定している純粋なphysical realizationだけを扱います。expression以外のprovider-native identifierやeffective defaultの追加契約はIssue #58で継続します。

## 非目標

M0.33では、次を実装しません。

- runtime Provider SPIの統合・拡張
- Workspace load時のDB接続
- arbitrary SQL expressionのcanonicalization
- production databaseを使ったnormalization用DDL
- external provider policyのplugin discovery
- Advanced Index DDL rendering

これらを同時に取り込まず、Desired Physical realizationの所有境界だけを先に安定させます。
