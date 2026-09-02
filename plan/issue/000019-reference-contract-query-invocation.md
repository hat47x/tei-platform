# ISSUE-000019: reference PatternからDatabase Contract Queryへの明示的な呼び出し境界を定める

- 状態: In Progress
- GitHub Issue: #185
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 前提Issue: `ISSUE-000017` / GitHub Issue #178、`ISSUE-000018` / GitHub Issue #179
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

Application Driverで`reference` Patternから`master_data`のCustomerを取得する経路を具体化したところ、Semantic Relationのkey対応とDatabase Contract Queryのsignatureをそれぞれ定義しても、その二つを安全につなぐには、もう一つ明示的な境界が必要であることが分かりました。

現行Referenceは次のように、Relationとlogical Datasourceを示しています。

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

また、`master_data`にはCustomer用Contractがあります。

```tei
contract master_data.customer v2.4 {
    read customer.customer
    query find_by_id
}
```

Applicationも、このContractのmajor version 2と互換であることを要求しています。

```tei
database master {
    target master_data
    access read
    requires {
        contract master_data.customer ^2
    }
}
```

しかし、RelationとDatasourceだけでは、次の二点を一意に決められません。

1. `master_data.customer`に複数のQueryがある場合、どのQueryをこのreferenceで利用するか
2. Relation target側のどのSemantic attributeを、Contract Queryのどのinputへ渡すか

DriverがQuery名、input名、Semantic型、Contract内のQuery数などから呼び出し先を推測する方針は採りません。

## 三つの責務を分離する

このIssueでは、次の三層を混同しないことを前提とします。

### Semantic Relation

`ISSUE-000017`が扱う層です。

```text
sales.order.customer_id
        ↓
customer.customer.customer_id
```

ここでは、source instanceとtarget instanceがどのSemantic attributeで対応するかを表します。

### Database Contract Query

`ISSUE-000018`が扱う層です。

```text
find_by_id
  input id type customer.customer_id
  result customer.customer zero-or-one
```

ここでは、Database read契約が何を受け、何を返すかを表します。

### Reference invocation

このIssueが扱う層です。

```text
reference Pattern
  relation sales.order.customer
  datasource master
  contract master_data.customer
  contract-query find_by_id
  bind customer.customer.customer_id -> id
```

ここでは、そのApplication PatternでどのDatabase Contract Queryを利用し、Relation target keyの値をどのQuery inputへ渡すかを表します。

## なぜ型一致による自動接続を採らないか

最初のReferenceだけを見ると、Relation target側の`customer.customer.customer_id`とQuery inputの型`customer.customer_id`が対応するため、型一致から接続できるように見えます。

しかし、同じSemantic型を持つinputが複数ある場合、その型だけではどのinputへ渡すべきか決まりません。また、将来Query signatureへalias、scope、tenant key等が加われば、同型のparameterが複数存在することは十分にあり得ます。

そのため、型は整合性を検査する材料には使えても、bindingを選ぶ正本にはしません。

## Application Queryとのidentityを分ける

`application.Pattern`には既に`Query`がありますが、これは現在のDriverでApplication-ownedな`query.EmbeddedSQL`のCanonical IDとして利用されています。

Database-ownedなContract Queryを同じfieldへ載せると、次の異なるidentity体系が混ざります。

```text
Application Query
  order_entry.recent_orders

Database Contract Query
  Contract: master_data.customer
  Query: find_by_id
```

このため、Reference Contract Query呼び出しは既存`Pattern.Query`とは別のmodel / authoring境界で表現します。

## 現在の実装到達点

このIssueから、次の小さな実装単位を分離しています。

### PR #186 — Canonical model

`reference` Patternが次を保持できるmodelを追加しました。

```text
ContractQueryInvocation
  ├─ Contract
  ├─ Query
  └─ Inputs[]
       ├─ TargetAttribute
       └─ Input
```

既存PatternのCanonical JSON shapeを変えないよう、未定義時は省略します。

### PR #187 — Application内で閉じる基本Validation

Relation keyやtyped Query signatureとまだ結合せず、次を検査します。

- `reference` Pattern以外では利用しない
- Contract / Query identityを明示する
- Datasourceを明示する
- ApplicationがそのDatasourceで要求しているContractだけを利用する
- target attribute / Query inputの重複bindingを拒否する

### PR #188 — authoring構文

Application Patternへ独立した`contract-query`ブロックを追加しました。

```tei
contract-query {
    contract master_data.customer
    query find_by_id
    input customer.customer.customer_id -> id
}
```

Application Query用の`query`構文とは分離しています。

### PR #189 — Driverによる明示解決

Driverが次の経路だけを辿ってContract Queryを解決する境界を追加しました。

```text
Patternのcontract-query
        ↓
ApplicationのDatasource requirement
        ↓
target Database
        ↓
要求されたContract / compatible major
        ↓
明示されたContract Query
```

別Databaseの同名Contractや、存在しないQueryへの暗黙フォールバックは行いません。

## 実証順

1. Reference Contract Query呼び出しのCanonical modelを追加する
2. Application自身の情報だけで確認できる基本Validationを追加する
3. `reference` Patternのauthoring構文へ接続する
4. DriverがContract / Query identityをDatasource境界内で明示解決できるようにする
5. `ISSUE-000017`のRelation keyとReference invocationのtarget attributeを横断Validationする
6. `ISSUE-000018`のtyped Query signatureとinput bindingを横断Validationする
7. Query result Semantic DataとRelation target Dataの整合をValidationする
8. Reference sourceへ実authoringを追加し、Canonical World / Source Indexを同期する
9. Relation source instanceの実値からContract Query parameterを組み立てる
10. Database Contract Query realizationへ接続し、read-onlyで実行する
11. 0件とExecution Errorを分離したままDriverへ返す
12. `customer_reference`を一続きのReference Scenarioとして実証する

## 横断Validationで確認すること

三つのmodelが共通baseへ合流した後、少なくとも次を確認します。

- bindingの`TargetAttribute`がRelation keyのtarget側に存在する
- 必要なRelation target keyをbindingが過不足なく扱う
- binding先のQuery inputがsignatureに存在する
- Relation target attributeのSemantic型とQuery input型が整合する
- Contract Queryのresult DataがRelation target Dataと一致する
- Query result cardinalityがreference Patternの意味と矛盾しない

型一致は、明示bindingを検証するために使います。bindingを自動生成するためには使いません。

## 完了条件

- [ ] reference Patternから利用するDatabase Contract Queryを名前推測なしで特定できる
- [ ] Application QueryとDatabase Contract Queryのidentityが混同されない
- [ ] Relation target keyとQuery inputの対応を明示できる
- [ ] 同じSemantic型のinputが複数あってもbindingが曖昧にならない
- [ ] Applicationが要求していないContractを呼び出せない
- [ ] Datasourceのread / read-write境界を維持する
- [ ] Contract compatible majorを無視して呼び出さない
- [ ] Query resultがRelation target Dataと整合することをValidationできる
- [ ] Source IndexからPattern、Contract、Query、bindingのauthoring sourceを追跡できる
- [ ] Relation source instanceの値を明示bindingに従ってQuery parameterへ渡せる
- [ ] 0件とExecution Errorを区別したままDriverへ返せる
- [ ] `Order -> Customer` Referenceを一続きに実証できる

## 非目標

- 汎用ORM
- Query名や型からのContract Query自動選択
- generic join engine
- GraphQL風resolver framework
- SQLやDatabase APIをApplication Patternへ埋め込むこと
- Database Contract QueryをすべてSemantic Capabilityへ置き換えること
- Target UI固有のnavigation設計
- production向けcache / batching / Connection Pool設計

## CI上の制約

2026-09-03時点では、GitHub Actionsが`runner_id=0`かつjob step未生成のまま終了する事象が継続しています。

このため、各PRは通常runnerで`verify`とPostgreSQL integrationが実行・成功するまでmainへ統合しません。runner割当前のfailureをコード失敗とは扱いませんが、CI成功とも扱いません。

## 関連

- GitHub Issue #69 / #178 / #179 / #185
- PR #181 / #182 / #183 / #184 / #186 / #187 / #188 / #189
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000017-semantic-relation-traversal-key.md`
- `plan/issue/000018-database-contract-query-execution.md`
- `reference/workspace/applications/order-entry/application.tei`
- `reference/workspace/applications/order-entry/patterns/customer-reference.tei`
- `reference/workspace/databases/master-data/contracts/customer.tei`
- `internal/application/model.go`
- `internal/driver/reference_contract_query.go`
