# ISSUE-000023: header-detail Relation traversalの実現境界を定める

- 状態: Done
- GitHub Issue: #233
- 決定: ADR-0021
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 後続: `ISSUE-000024` / GitHub Issue #244
- 関連Issue: `ISSUE-000011` / #117、`ISSUE-000012` / #118、`ISSUE-000017` / #178、`ISSUE-000018` / #179、`ISSUE-000019` / #185
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 目的

`sales.order.lines`のようなkeylessな`many` Semantic Relationを、Driverやphysical Foreign Keyの都合でSemanticへ新しい属性を追加せず、Application Patternの責務も重くせずにruntime traversalへ接続する境界を定めます。

第一Referenceでは、次のSemanticが既に成立しています。

```tei
data Order {
    attribute orderId type sales.order_id
    ...
    relation lines to sales.order_line many
}

data OrderLine {
    attribute orderLineId type sales.order_line_id
    attribute lineNumber type std.integer
    attribute itemCode type std.code
    attribute quantity type std.decimal
    attribute unitPrice type std.money
}
```

`OrderLine`には親Order identityを表すSemantic attributeを置いていません。`ISSUE-000011`で、Driverや物理DB設計の都合だけを理由に`orderId`を追加しないことを確認しています。

header-detail Patternも、Application intentとしてはRelation-onlyです。

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

このIssueでは、この二つの境界を維持したまま`Order -> OrderLine[]`を実行できるかを実Referenceで検証しました。

## 比較した選択肢

### A. OrderLineへ親Order identityをSemantic attributeとして追加する

OrderLine自身の業務Semanticとして必要であることが確認できた場合には自然です。

しかし、runtime traversalやForeign Keyを実装しやすくするためだけに追加すると、物理実現の都合をSemanticへ逆流させます。このIssueの実証では必要ありませんでした。

### B. header-detail PatternへDatasourceと取得Queryを追加する

Pattern自身が、Relationに加えてlogical Datasource、Contract Query、source attributeからQuery inputへのbindingまで所有する案です。

実行は可能ですが、Relation-onlyで成立しているApplication intentへ取得方式を戻すことになります。Patternの責務を不必要に重くするため採用しませんでした。

### C. Pattern外のApplication固有realizationとして取得方法を置く

採用した案です。

```text
Application Pattern / Semantic Relation
        ↓
HeaderDetailContractQueryBinding
  logical Datasource
  Database / Contract / Query
  source Semantic attribute -> Query input
        ↓
Database Contract Query
  typed input / result Data / many
```

Application PatternはRelation-onlyのまま維持し、取得方法だけを独立realizationへ分離します。

### D. Order取得QueryでOrderLine collectionまで一括materializeする

APIや別Targetでは有効な場合がありますが、第一Referenceのためにflat result境界を先回りで広げる必要はありません。v1の標準境界にはしません。

### E. physical Foreign Key / Persistence Mappingから自動推論する

採用しません。物理構造をSemantic Relationの権威へ逆転させるためです。

## 採用した境界

ADR-0021として、C案を採用しました。

### HeaderDetailContractQueryBinding

v1では次の情報をApplication固有realizationとして保持します。

```text
HeaderDetailContractQueryBinding
  Application
  Pattern
  Datasource
  Target
    Database / Contract / Query
  Inputs[]
    SourceAttribute -> Query Input
```

Binding identityは`Application + Pattern`です。DatasourceやContract Queryは選択内容であり、identityには含めません。

Reference authoringは次です。

```tei
header-detail-contract-query-binding {
    application order_entry
    pattern order_entry.order_lines
    datasource sales
    database sales_core
    contract sales_core.order_line
    query find_by_order
    input sales.order.order_id to order_id
}
```

### RelationKeyを要求しない

Validationとruntime preparationでは、Bindingの`SourceAttribute`がRelation source Dataに属することを確認しますが、`sales.order.lines`にRelationKeyが存在することは要求しません。

`RelationKey`は、sourceとtargetの両側に業務上意味のあるSemantic attribute対応があるRelationで利用します。すべてのRelationに対する一般的なruntime join keyにはしません。

このため、`sales.order_line.order_id`を追加せずに、`sales.order.order_id`をcollection取得Queryへ渡せます。

### Database Contract Query

Referenceには次のtyped Queryを追加しました。

```tei
contract sales_core.order_line v1.0 {
    read sales.order_line
    query find_by_order {
        input order_id type sales.order_id
        result sales.order_line many
    }
}
```

ここにはphysical table、column、Foreign Key、SQL本文を含めません。

### 全件Queryへの退行を防ぐ

header-detail traversalには、少なくとも一つの`SourceAttribute -> Query input` bindingを必須としました。

Query inputが0件のContractを使って親Orderと無関係な全OrderLineを取得する形は、header-detail traversalとして認めません。

## 実証結果

### model / Validation

PR #235と#236で、Canonical modelと通常Validationを追加しました。

次を検査します。

- Application / Pattern identity
- Pattern kind = `header-detail`
- readable logical Datasource
- target Database
- required Contractとmajor compatibility
- typed Contract Query
- source attributeがRelation source Dataに属すること
- source attribute型とQuery input型の一致
- 全Query inputへの明示binding
- result Data = Relation target Data
- result cardinality = `many`
- Binding identityの一意性

RelationKeyは検査条件にしていません。

### authoring / Source Index

PR #237で、専用ParserとWorkspace Loaderを追加しました。

配置は次です。

```text
applications/<unit>/realizations/header-detail/*.tei
```

Application / Pattern / Datasource / Database / Contract / Query identityはsource本文から取得し、pathやfilenameから補完しません。

Bindingは次のSubjectでSource Indexへ登録されます。

```text
header-detail-contract-query-binding:<Application>/<Pattern>
```

Validation Diagnosticからrealization sourceへ戻れることも確認しました。

### 実Reference

PR #238で、`sales_core.order_line / find_by_order`と`order_entry.order_lines`のBindingを実Referenceへ追加しました。

この時点でも次は追加していません。

- `sales.order_line.order_id`
- `sales.order.lines`のRelationKey
- OrderLine Persistence Mapping
- physical table / column / Foreign Key
- SQL realization

### runtime parameter preparation

PR #239で、実Referenceのmapping済みOrder rowから次を準備できることを確認しました。

```text
sales.order.order_id = 1001
        ↓ explicit binding
order_id = 1001
```

mapped column順、raw column名、Query input名の類似、Persistence Mapping、physical Foreign Keyから値を推測しません。

### Contract Query execution

PR #240で、既存`ContractQueryExecutor`へそのまま接続しました。

`many` resultでは次を区別できます。

```text
0 row          -> 正常な空collection
複数row        -> 正常なsales.order_line[]
Executor error -> Execution Error
```

header-detail専用のDatabase Executorは追加していません。

### PostgreSQL runtime

PR #241で、共通`sqlquery.Executor`の下位にPostgreSQL SQL Runnerを追加しました。

既存の`:name -> $1, $2, ...` parameter binderを再利用し、runtime値をEvidenceへ保存しません。

PR #242ではReferenceの`tei.database.postgres`に`database.query.postgres` Capabilityを公開し、次のEnvironment compositionを明示しました。

```text
order_entry.sales
  -> development / sales_core.dev
  -> database.query.postgres
```

provider一致だけでCapabilityや別instanceのruntime Executorを選択しません。

### 物理realizationがない場合は停止する

PR #243では、上記compositionがすべて成立していても、`sales_core/sales_core.order_line/find_by_order`のPostgreSQL SQL realizationが存在しなければ、physical Runnerを呼ぶ前に停止する負のE2Eを追加しました。

Environment compositionの成立を、物理SQLを推測する根拠にはしません。

## 物理保存境界の分離

runtime traversalの責務境界は確定しましたが、managed-here Databaseで`sales.order.lines`をどのphysical column / Foreign Key / link structureへ保存するかは別問題です。

現行Persistence MappingはSemantic attributeとphysical columnの対応を基本としているため、target Dataに親Order attributeを追加しないkeyless Relationには、Database-ownedなrelationship storageを表す境界が不足しています。

この残件は`ISSUE-000024` / GitHub Issue #244へ分離しました。

`ISSUE-000024`が確定する前に、`order_lines.order_id`等のphysical columnやForeign Key、`find_by_order` native SQLをこのIssueから発明しません。

## 完了条件と結果

- [x] `OrderLine.orderId`をruntime都合だけでは追加しない境界を確認した
- [x] keylessな`many` Relationのruntime取得責務をPattern外realizationへ置いた
- [x] source instance → Query inputの対応を名前・型推測なしで表現した
- [x] result Data / cardinalityを明示的に検証した
- [x] physical Foreign KeyをSemantic Relation keyとして自動採用しない
- [x] 実Reference authoringを作り、Source Indexまで追跡した
- [x] Driverで`Order -> OrderLine[]`を推測なしで実行した
- [x] 0件を正常な空collectionとして扱った
- [x] 複数件を正常な`many` resultとして扱った
- [x] Execution Errorと空collectionを区別した
- [x] Environment / Database instance / executor Capabilityを明示compositionした
- [x] PostgreSQL runtime境界へ接続した
- [x] SQL realization欠落時に推測せず停止した
- [x] 物理保存方式の未決事項を`ISSUE-000024`へ分離した

通常GitHub Actionsは2026年9月3日時点でRepository上から明示的に停止されています。CI再開後の通常runnerによる再検証は、Gate B監査側の横断確認として扱います。このIssueの設計責務は完了しています。

## 非目標

- keyless Relationのphysical storage方式をこのIssueで決めること
- 汎用ORM association
- generic join engine
- UI Grid / Form layout
- physical Foreign Keyの自動発見
- 全種類のcollection Relationを一度に一般化すること

## 関連

- ADR-0021
- GitHub Issue #69 / #117 / #118 / #178 / #179 / #185 / #233 / #244
- GitHub PR #235〜#243
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000011-order-line-semantic-reference.md`
- `plan/issue/000012-header-detail-application-pattern.md`
- `plan/issue/000017-semantic-relation-traversal-key.md`
- `plan/issue/000018-database-contract-query-execution.md`
- `plan/issue/000024-keyless-relation-physical-storage.md`
- `reference/workspace/semantic/sales/order.tei`
- `reference/workspace/applications/order-entry/realizations/header-detail/order-lines.tei`
- `reference/workspace/databases/sales-core/contracts/order-line.tei`
