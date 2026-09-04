# ISSUE-000019: reference PatternからDatabase Contract Queryへの明示的な呼び出し境界を定める

- 状態: Done
- GitHub Issue: #185（完了）
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 前提Issue: `ISSUE-000017` / #178（完了）、`ISSUE-000018` / #179（完了）
- runtime identity: `ISSUE-000020` / #222（完了）
- actual external Evidence: `ISSUE-000022` / #214
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 目的

`reference` PatternからDatabase Contract Queryへ進むときに、DriverがQuery名、input名、型一致、Contract数から呼び出し先を推測しないよう、**Application固有の明示invocation境界**を定めます。

責務を次の三層へ分けます。

1. **Semantic Relation** — どのData同士が、どのSemantic identityで対応するか
2. **Database Contract Query** — Database read契約が何を受け、何を返すか
3. **Reference invocation** — そのApplication Patternで、どのContract Queryを使い、source valueをどのinputへ渡すか

SQL、parameter、raw column等の物理実装はさらに下位のSQL realizationへ残します。

## 第一Reference

`order_entry.customer_reference`では次を明示的に接続します。

```text
Pattern
  order_entry.customer_reference
    relation: sales.order.customer
    datasource: master
        ↓
Application Datasource requirement
  master -> master_data
        ↓
Database Contract
  master_data.customer
        ↓
Contract Query
  find_by_id
        ↓
Relation source value
  sales.order.customer_id
        ↓ explicit invocation binding
Query input
  id : customer.customer_id
```

`find_by_id`という名前や`customer.customer_id`という型が偶然一致するために選ぶのではありません。

## 採用した所有境界

Reference invocationはPattern本体へ物理実現責務を戻さず、Application-owned realizationとして分離します。

Patternは引き続きApplication上の意図を保持します。

```text
reference Pattern
  relation
  datasource
```

invocation realizationが次を保持します。

- Application identity
- Pattern identity
- Relation identity
- logical Datasource
- Database Contract target
- Contract Query identity
- source Semantic attribute → Query input binding

この分離により、同じSemantic RelationやDatabase Contract Queryを別Applicationで異なる実現として利用できます。

## Validation

通常のWorkspace Validationから少なくとも次を確認します。

- Application / Patternが存在する
- Pattern kindがreferenceである
- Pattern Relationとinvocation Relationが一致する
- logical DatasourceがApplicationに定義されている
- Datasourceが要求するDatabase Contractの範囲内である
- Contract Queryが存在する
- Query result DataがRelation target Dataと整合する
- source Semantic attributeがRelation source Dataに属する
- binding先Query inputが存在する
- input typeとsource valueのSemantic typeが整合する
- required inputに不足がない
- 余分な暗黙bindingを行わない

Applicationが要求していないContractやQueryへDriverがfallbackしません。

## runtime

Driverは次の順に明示identityを辿ります。

```text
Application / Pattern
  ↓
Reference invocation
  ↓
logical Datasource
  ↓
Environment Database instance
  ↓
Contract Query executor Binding
  ↓
SQL realization
  ↓
provider Runner
```

Environment / Database instance / executor Capabilityは`ISSUE-000020`で保持するようにしました。

raw resultはSQL realizationの明示mappingからSemantic targetへ変換します。

## 0件とExecution Error

reference Queryが`zero-or-one`の場合、次を区別します。

- 0 row: 正常なnot-found
- 1 row: 正常なreference result
- 複数row: cardinality違反
- physical query失敗: Execution Error

Driverが0件をExecution Errorに変えたり、Execution Errorを「参照なし」として隠したりしません。

## Source traceability

Pattern、invocation、Database Contract、Contract Query、SQL realization、Environment Bindingをそれぞれのauthoring sourceへ追跡できます。

file pathをCanonical identityの代わりに利用しません。

## external-observed `master_data`との関係

このinvocation境界が成立しても、actual Oracle object / parameter / raw columns / SQL / privilegeが判明したことにはなりません。

actual external facts、provenance、read-only強制、実Oracle readは`ISSUE-000022` / #214で扱います。

PR #288 / #300 / #306でexternal Evidence intakeも成立しましたが、EvidenceからinvocationやSQL realizationを生成・補完しません。

actual Evidence未取得を、本Issueのinvocation設計未完了とは扱いません。

## 完了条件

- [x] reference Patternから利用するDatabase Contract Queryを名前推測なしで特定できる
- [x] Relation source valueとQuery inputの対応を型一致だけに依存せず明示できる
- [x] Application QueryとDatabase Contract Queryのidentityを混同しない
- [x] Applicationが要求していないContractを呼び出さない
- [x] Datasourceのread / read-write境界を維持する
- [x] Query result DataとRelation target DataをValidationできる
- [x] 0件とExecution Errorを区別したままDriverへ返せる
- [x] Pattern / invocation / Contract / Query / realization / Bindingのsourceを追跡できる
- [x] `Order -> Customer` Referenceを同じSemantic Order rowから実行経路へ接続できる
- [x] actual Oracle Evidenceを#214へ分離した

## 非目標

- 汎用ORM / auto join engine
- Contract Queryの自動選択
- GraphQL風resolver framework
- SQLやDatabase APIをApplication Patternへ埋め込むこと
- Target UI固有navigation
- actual Oracle物理情報の推測

## 関連

- `ISSUE-000006` / #69
- `ISSUE-000017` / #178
- `ISSUE-000018` / #179
- `ISSUE-000020` / #222
- `ISSUE-000022` / #214
- `ISSUE-000029` / #273
- `reference/workspace/applications/order-entry/`
- `internal/driver/reference_contract_query_environment.go`
