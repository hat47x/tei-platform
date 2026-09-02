# ISSUE-000017: Semantic Relationをinstance間で辿るkey対応を明示する

- 状態: In Progress
- GitHub Issue: #178
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 関連RQ: RQ-005 / RQ-009 / RQ-012

## 背景

Application Driverで`reference` / `header-detail` Patternを実行へ進めると、Semantic Relationがtarget Dataとcardinalityだけでは、source instanceからtarget instanceを一意に辿れないことが分かりました。

Driverはattribute名、型の類似、物理Foreign Key、Persistence Mappingからこの対応を推測しません。Relation traversalに必要なSemantic attribute同士の対応は、Semantic Relation自身が明示的に所有する方針で実装を進めています。

## 第一Referenceで確定した境界

`order_entry.customer_reference`では、次をRelation keyとして明示しました。

```tei
relation customer to customer.customer one
    key sales.order.customer_id -> customer.customer.customer_id
```

意味は次です。

```text
sales.order.customer_id
        ↓
customer.customer.customer_id
```

これは物理Foreign KeyやDatabase queryを表すものではありません。source Data instanceからtarget Data instanceを辿るためのSemantic key対応です。

## ここまでに実装したこと

- `semantic.RelationKey` model
- Relation key authoring構文
- source / target Semantic attributeの存在確認
- source attributeがRelationのsource Dataへ属することの確認
- target attributeがRelation target Dataへ属することの確認
- source / targetのSemantic型identity一致確認
- 完全重複pairの拒否
- 同一target attributeへ複数source attributeを対応付ける曖昧性の拒否
- 第一Reference `sales.order.customer`へのkey適用
- Reference Contract Query inputへtarget keyを明示bindingする横断Validation
- `browse` mapped rowのsource key値を、Relation key経由でContract Query parameterへ渡すruntime境界

## 責務分離

Relation keyは次だけを所有します。

```text
source Semantic attribute
        ↓
target Semantic attribute
```

次は所有しません。

- SQL / Database API / stored procedure
- Database Contract Query名
- Application datasource alias
- physical table / column / Foreign Key
- runtime connection

Database Contract Queryのinput / result / cardinality / realizationは`ISSUE-000018`、Applicationのreference PatternからQueryを呼び出すbindingはGitHub Issue #185で扱います。

## 残る実証対象

第一Referenceの`customer` Relationは単一keyで実行経路まで接続できました。残る主要な実証対象は`header-detail`です。

```text
sales.order.lines
  source: sales.order
  target: sales.order_line
  cardinality: many
```

現行`OrderLine`には`order_id`相当のSemantic attributeがないため、Driver実行の都合だけで新しいattributeを追加してはいけません。Reference業務意味として`OrderLine`が親Order identityを保持することを確かめたうえで、必要なら次を追加します。

```text
sales.order.order_id
        ↓
sales.order_line.order_id
```

## 完了条件

- [x] Relation keyをCanonical modelで表現できる
- [x] Relation keyをauthoring sourceから読み込める
- [x] source / target attributeの所属とSemantic型identityを検査できる
- [x] target側からsource値を一意に逆引きできないkey定義を拒否できる
- [x] 第一Referenceのcustomer Relationで実値をQuery inputまで運べる
- [ ] header-detailの実EvidenceからOrder / OrderLine間のkeyを確定する
- [ ] 必要なら`OrderLine`側の親Order identityをSemantic Dataへ追加する
- [ ] `sales.order.lines`を実runtime traversalへ接続する

## 非目標

- physical Foreign KeyをSemantic Relationの正本にすること
- attribute名や型からRelation keyを自動生成すること
- 汎用ORMのassociation model
- UI上の親子表示仕様

## 関連

- GitHub Issue #69 / #178 / #179 / #185
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000018-database-contract-query-execution.md`
- `reference/workspace/semantic/sales/order.tei`
- `reference/workspace/applications/order-entry/patterns/customer-reference.tei`
- `internal/semantic/model.go`
- `internal/validation/semantic_relation_keys.go`
- `internal/driver/reference_contract_query_parameters.go`
