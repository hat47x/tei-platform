# ISSUE-000018: Database Contract Queryのinput・result・実行境界を明示する

- 状態: Ready
- GitHub Issue: #179
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 前提Issue: `ISSUE-000017` / GitHub Issue #178
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

Application Driverで`reference` Patternから`master_data`のCustomerを取得する経路を検討したところ、Semantic Relationのkey対応とは別に、Database Contract Query自体の実行契約が不足していることが分かりました。

Referenceにはすでに次のContractがあります。

```tei
contract master_data.customer v2.4 {
    read customer.customer
    query find_by_id
}
```

ApplicationもこのContractを明示的に要求しています。

```tei
database master {
    target master_data
    access read
    requires {
        contract master_data.customer ^2
    }
}
```

しかし現行`database.QueryContract`が保持するのは`Name`だけです。

```text
QueryContract
  └─ name
```

このため`find_by_id`について、少なくとも次が機械可読には分かりません。

- どのinputを受けるか
- inputがどのSemantic identity / typeを表すか
- 何を返すか
- 0件、1件、複数件をどう扱うか
- 実際のDatabase API、SQL、stored procedure等へどう接続するか

Relationのkey対応を決めても、この不足をDriverが`find_by_id`という名前から推測して補うことはしません。

## ISSUE-000017との責務分離

`ISSUE-000017`は、たとえば次のSemanticな対応をどの層が正本として持つかを扱います。

```text
sales.order.customer_id
        ↓
customer.customer.customer_id
```

このIssueは、そのtarget keyを`master_data`へ渡してCustomer instanceを取得する**Database Contract Queryの契約と実現境界**を扱います。

```text
Relation traversal key
        ↓
Database Contract Query signature
        ↓
Database固有のquery realization
        ↓
external-observed Database
```

Semantic RelationへSQLやDatabase APIを持ち込まず、Database Contract QueryへRelationの業務意味を重複させません。

## Referenceの圧力条件

`master_data`はTEIが所有するDatabaseではありません。

```tei
database master_data {
    kind relational
    ownership external-observed
    uses semantic customer
}
```

Applicationからは`master` aliasでread-only利用します。

最初のReferenceでは、TEI管理tableを前提にせず、既存Databaseが公開するread契約を呼び出す形で境界を確かめます。

## このIssueで決めること

少なくとも次の三案を比較します。

### A. QueryContract自身へsignatureを追加する

概念例:

```text
query find_by_id
  input id type customer.customer_id
  result customer.customer zero-or-one
```

Database Contractが「何を受け、何を返すか」を所有し、SQL等の実装方式は別Realizationへ分離します。

### B. Semantic Capabilityへ寄せる

`customer.find_by_id`のようなCapabilityとして扱い、既存Capability Binding / Invocationを再利用します。

ただし、単なるDatabase read契約まで業務Capability化すると意味階層が混ざる可能性があるため、実Referenceの必要性なしには採用しません。

### C. Application Queryとして再定義する

Applicationが`datasource master`を持つQueryを定義します。

Application固有のProjectionであれば妥当ですが、external Database Contractが所有しているquery意味をApplicationへ複製する危険があります。

## 最小の実証対象

`find_by_id`で、まず次だけを成立させます。

- input identity / type
- result Semantic Data
- result cardinality
- query realizationの明示identity
- external-observed Databaseへのread-only実行
- 0件を正常な「見つからない」として扱う境界
- query実行障害を正常な0件へ変換しない境界

この段階では巨大なDatabase Service frameworkや汎用RPC modelへ広げません。

## 実証順

1. `find_by_id`に必要なinput / result / cardinalityをReferenceから確定する
2. Database ContractのauthoringとCanonical modelに最小表現を追加する
3. input / resultのSemantic参照整合性を共通Validationへ接続する
4. query realizationのidentityを明示する
5. `master_data`用Reference implementationをread-onlyで実行する
6. 0件と実行障害を分離する
7. `ISSUE-000017`で確定したRelation keyを暗黙変換なしでinputへ渡す
8. authoring sourceとrealization sourceの来歴を確認する
9. Driverから`reference` Patternを一続きに実証する

## 完了条件

- [ ] `find_by_id`のinputを名前から推測しない
- [ ] inputがどのSemantic identity / typeを表すか追跡できる
- [ ] resultが`customer.customer`であることを契約から追跡できる
- [ ] 必要なresult cardinalityを曖昧にしない
- [ ] Database Contract Queryと実現方式を分離できる
- [ ] `master_data`をread-onlyのまま利用する
- [ ] Database固有SQL / API / stored procedureをDriver Coreへ持ち込まない
- [ ] 0件とExecution Errorを区別する
- [ ] authoring source / realization sourceの来歴を追跡できる
- [ ] `ISSUE-000017`のRelation keyを暗黙変換なしでquery inputへ渡せる

## 非目標

- 汎用ORM
- 全Database Contract Queryを覆う完全な型体系
- production Connection Pool / HA
- external Database schemaのTEI管理化
- Database queryをすべてSemantic Capabilityへ統一すること
- Application Queryとの自動統合
- Target UI向けデータ取得API

## 関連

- GitHub Issue #69 / #167 / #178 / #179
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000017-semantic-relation-traversal-key.md`
- `reference/workspace/databases/master-data/database.tei`
- `reference/workspace/databases/master-data/contracts/customer.tei`
- `reference/workspace/applications/order-entry/application.tei`
- `reference/workspace/applications/order-entry/patterns/customer-reference.tei`
- `internal/database/model.go`
