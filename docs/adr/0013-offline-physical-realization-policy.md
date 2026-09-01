# ADR-0013: Desired Physical realizationのprovider policyをruntime Provider SPIから分離する

- 状態: 採用
- 日付: 2026-09-01
- 関連Issue: #58, #59
- 前提: ADR-0010〜ADR-0012 / M0.29〜M0.32

## 背景

Persistence MappingからDesired Physical Schemaを生成する処理は、もともと`internal/database/persistence/realize.go`の中でproviderごとの違いを解決していました。

M0.32時点では、少なくとも次の知識がPersistence packageに置かれています。

- provider-neutralなStorageTypeからPostgreSQL / Oracleのphysical typeへの変換
- provider既定のIndex access method
- unique indexの既定NULL semantics
- PostgreSQL B-treeのASC / DESC・NULL orderingのeffective default
- PostgreSQLのdefault collation表現の正規化

これらはPersistence Mappingそのものの意味ではなく、providerが所有すべきphysical realization policyです。

一方、既存の`database/provider.Provider`はActual databaseへ接続して`Capabilities`や`Introspect`を実行するruntime SPIです。Desired realizationのためだけにこのinterfaceへ機能を追加すると、Workspace validationやbuildへ不要な`context.Context`、`DatabaseInstance`、DB接続、runtime assemblyを持ち込む恐れがあります。

## 判断

Desired Physical realizationに必要な純粋なprovider policyを、runtime `Provider` SPIとは別の境界として扱います。

依存方向は次のようにします。

```text
Persistence Mapping / realize
          |
          | depends on interface only
          v
PhysicalRealizationPolicy
          ^
          |
          +-- providers/postgres.RealizationPolicy
          +-- providers/oracle.RealizationPolicy

composition / database/realization
          |
          +-- provider enum -> concrete policy
          +-- Persistence realizeを呼び出す
```

Persistence packageはPostgreSQLやOracle packageをimportしません。

具体的なprovider packageはPersistence側が定義する小さなpolicy interfaceを実装できます。外側のadapterが内側のportを実装する依存方向とします。

## PhysicalRealizationPolicy

M0.33では、現在すでに必要な責務だけをinterfaceへ置きます。

概念上は次です。

```text
PhysicalRealizationPolicy
  Provider()
  PhysicalType(StorageType)
  IndexDefaults()
  NormalizeIndexKey(IndexKey, method)
```

`IndexDefaults`は、少なくとも次を返します。

```text
IndexDefaults
  Method
  UniqueNullTreatment
```

将来使うかもしれない機能を先取りして、巨大なprovider interfaceにはしません。

## runtime Providerとは別interfaceにする

runtime `Provider`はActual databaseとの対話を担当します。

```text
Provider
  Capabilities(context, instance)
  Introspect(context, instance)
```

Physical realization policyは、DB接続を使わない純粋な変換を担当します。

```text
RealizationPolicy
  PhysicalType(...)
  defaults / deterministic normalization
```

同じGo packageが両者を実装しても構いませんが、interface自体は統合しません。

これにより、Desired realizationはruntime Provider objectやlive DatabaseInstanceを必要としません。

## provider enumからの選択はcomposition境界へ置く

利用側が`postgres.RealizationPolicy{}`のような具体型を毎回知る必要はありません。

`internal/database/realization`をcomposition境界として用意し、次を担当させます。

```text
Provider ID
  -> concrete PhysicalRealizationPolicy
  -> persistence.RealizeDesired(..., policy)
```

provider IDのswitchはここに集約します。

このswitchは「DBMSの意味をPersistence Coreへ埋め込む」ものではなく、抽象portへ具体adapterを組み立てるcompositionです。

unsupported providerはgeneric defaultへ縮退させず、明示的にエラーとします。

## PostgreSQL policy

PostgreSQL policyは、現在Persistence packageに存在する次の知識を引き取ります。

- `int64 -> pg_catalog.int8`
- `varchar -> pg_catalog.varchar`
- `text -> pg_catalog.text`
- `datetime-tz -> pg_catalog.timestamptz`
- `decimal -> pg_catalog.numeric`
- `json -> pg_catalog.jsonb`
- default index method = `btree`
- default unique NULL treatment = `distinct`
- B-tree `ASC NULLS LAST`をeffective defaultへ正規化
- B-tree `DESC NULLS FIRST`をeffective defaultへ正規化
- `pg_catalog.default` collationをprovider defaultとして空表現へ正規化

これらはlive DBを必要としません。

## Oracle policy

現行Persistence realizationがすでに持っているOracleのstorage type対応も、Oracle realization policyへ移します。

M0.33でOracle runtime Providerまで実装することは目的ではありません。Desired Physical Schemaを生成するためのoffline policyだけを独立して持てます。

## ProviderExpression normalizationは別段階にする

ADR-0012で、ProviderExpressionの`authored`と`provider-normalized`を区別しました。

しかしM0.33では、任意のauthored expressionをprovider-normalized expressionへ変換するAPIを`PhysicalRealizationPolicy`へ追加しません。

PostgreSQL parser/deparser、provider version、server round-trip等の条件がまだ必要だからです。

policy境界を作ったことを理由に、未確定なexpression normalizer interfaceを先取りしません。

## Issue #58との関係

Issue #58では、collation / operator class / access method等のprovider-native identifierとeffective defaultのnormalization contractをさらに整理します。

M0.33のpolicy interfaceは、その検討結果を実装できる所有場所を先に整えます。ただし、#58で必要になる全メソッドを予想して今から追加しません。

必要なら後続ADRで小さく拡張するか、別の狭いpolicy interfaceを追加します。

## Persistence RealizeDesiredの責務

`persistence.RealizeDesired`は次を担当します。

- Mapping / Attribute / Columnの対応解決
- Primary Key / Unique Constraint / Foreign Key / Index構造の組み立て
- provider expression provenanceと対象providerの整合確認
- provider policyを呼び出してphysical type / defaults / deterministic normalizationを適用

DBMS名ごとのswitchやphysical type tableは持ちません。

## Workspace loadとは分離する

Workspace loaderはPersistence authoringを読み、Canonical Worldへ格納します。

Desired Physical realizationは必要なタイミングで明示的に実行するprojectionです。M0.33によってWorkspace loadにDB接続やruntime Provider assemblyを追加しません。

## 採用しない案

### runtime Provider interfaceへrealization methodを追加する

offline処理とconnected runtime操作の責務が混ざるため採用しません。

### Persistence packageにprovider switchを残し続ける

DBMS知識がPersistence Coreへ蓄積するため採用しません。

### generic fallback policyを用意する

unsupported semanticsを暗黙に単純化する危険があるため採用しません。

### plugin discoveryを使って通常のDesired realizationを組み立てる

現段階では過剰です。まずbuilt-in provider policyを明示的なcompositionで組み立てます。将来external provider policyが必要になった場合に別途拡張します。

### ProviderExpression normalizerまで同時に抽象化する

#54でrepresentation contractを固定した直後であり、normalizerの実装条件はまだ十分に確定していないため採用しません。

## Conformance

M0.33では次を固定します。

1. Persistence realizationはprovider-specific packageをimportしない
2. PostgreSQLのphysical type/default/index normalizationはPostgreSQL policyが所有する
3. Oracleの既存physical type変換はOracle policyが所有する
4. provider enumからpolicyを選択するcompositionは`database/realization`へ集約する
5. unsupported providerは明示的に拒否する
6. PostgreSQL / Oracleの既存Desired出力を維持する
7. PostgreSQL Advanced Indexのdefault normalizationを維持する
8. ProviderExpressionのprovider整合検査を維持する
9. realizationに`context.Context`、DatabaseInstance、DB接続を要求しない
10. runtime `database/provider.Provider` interfaceを変更しない

## 影響

- Persistence packageからDBMS固有のtype/default tableを除去できる
- provider-specific knowledgeの所有場所がActual adapterと同じprovider領域へ寄る
- offline Desired realizationとconnected introspectionの責務が明確になる
- #58でidentifier/default normalizationを進める際の実装場所が明確になる
- 外部provider policyやplugin化は、必要になるまで先取りしない
