# ADR-0021: keylessなheader-detail Relation traversalはPattern外のrealizationで実現する

- 状態: 採用
- 日付: 2026-09-03

## 背景

TEIの第一Referenceでは、`sales.order`と`sales.order_line`の間に次のSemantic Relationがあります。

```tei
data Order {
    attribute orderId type sales.order_id
    ...
    relation lines to sales.order_line many
}
```

一方、`sales.order_line`には親Order identityを表すSemantic attributeを置いていません。

これは実装漏れではありません。OrderLineの業務Semanticを成立させた時点で、Driverやphysical Foreign Keyの都合だけを理由に`OrderLine.orderId`を追加しない方針を採っています。

header-detail Patternも、Application intentとしては次のRelation-onlyな形で成立しています。

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

この状態から実行可能な`Order -> OrderLine[]` traversalへ進むには、具体的なOrder instanceの値を使って、どのDatabase Contract Queryを呼び出すかを表す必要があります。

ここで、reference Relationで用いている`RelationKey`をすべてのRelationへ一般化すると、target側に親Order IDのSemantic attributeを追加することが事実上必須になります。また、DatasourceやQueryをheader-detail Pattern本体へ追加すると、Relation-onlyで成立しているApplication intentへ取得方式の責務が逆流します。

`ISSUE-000023`では、これらを避けながら実行まで到達できる最小境界を実Referenceで比較・実証しました。

## 判断

keylessな`many` Relationをheader-detailとして取得する場合、**Application Pattern本体はRelation-onlyのまま維持し、取得方法はApplication固有の独立realizationとして表します。**

v1ではこのrealizationを`HeaderDetailContractQueryBinding`として扱います。

概念上の構造は次です。

```text
Application Pattern
  order_entry.order_lines
  relation sales.order.lines
        ↓
HeaderDetailContractQueryBinding
  application order_entry
  pattern order_entry.order_lines
  datasource sales
  target sales_core / sales_core.order_line / find_by_order
  input sales.order.order_id -> order_id
        ↓
Database Contract Query
  input order_id type sales.order_id
  result sales.order_line many
```

### BindingはRelation source側のSemantic attributeからQuery inputへ向ける

header-detail traversalでは、source instanceからcollection取得Queryへ値を渡します。

そのためBindingの向きは次です。

```text
Relation source Semantic attribute
        ↓
Database Contract Query input
```

第一Referenceでは次です。

```text
sales.order.order_id -> order_id
```

この対応を名前や型の類似から推測しません。両者のSemantic型identityが一致することをValidationとruntime準備の双方で確認します。

### keylessなRelationへ`RelationKey`を要求しない

`RelationKey`は、sourceとtargetの両側に業務上意味のあるSemantic attribute対応が存在するRelationで利用します。

たとえば`Order -> Customer`では次が自然に成立します。

```text
sales.order.customer_id
  -> customer.customer.customer_id
```

しかし、`Order -> OrderLine[]`ではtarget側に親Order ID attributeを置く必要性が確認されていません。

したがって、header-detail traversalを実行するためだけにRelationKeyを要求しません。`sales.order.lines`はkeylessなまま正当なSemantic Relationとして扱います。

### Pattern本体へDatasource / Contract Queryを戻さない

Patternは「何を扱うApplication intentか」を所有します。

取得に使うlogical Datasource、Database Contract、Query、source attributeからinputへのbindingは、Pattern本体ではなくrealization側が所有します。

これにより、同じRelation-only Patternを、将来別のDatabaseや別の実現方式へ写し替える余地を残します。

### Database Contract Queryはtyped result契約を所有する

collection取得の意味契約はDatabase Contract Query側に置きます。

第一Referenceでは次です。

```tei
contract sales_core.order_line v1.0 {
    read sales.order_line
    query find_by_order {
        input order_id type sales.order_id
        result sales.order_line many
    }
}
```

header-detail realizationはこのQueryを明示的に選択しますが、result Dataやcardinalityを独自に再定義しません。

### 0-input collection Queryをheader-detail traversalとして認めない

header-detail traversalは、具体的なsource instanceとの関係を辿る処理です。

そのため、少なくとも一つの`SourceAttribute -> Query input` bindingを要求します。親Orderと無関係な全OrderLine取得Queryを、header-detail traversalとして暗黙利用しません。

### Environmentとruntime implementationはさらに下位で解決する

`HeaderDetailContractQueryBinding`のDatasourceはlogical Datasourceです。

実行時には、EnvironmentのApplication→Database instance Binding、Contract Query executor Binding、Plugin Capability、instance-aware runtime Registryを順に解決します。

```text
order_entry.sales
  ↓
development / sales_core.dev
  ↓
database.query.postgres
  ↓
instance-aware runtime Executor
```

PostgreSQL providerが一致するという理由だけでCapabilityや別instanceのExecutorを選択しません。

### 物理保存方式はこのADRで決めない

この判断は、Semantic Relationをどのphysical column / Foreign Key / link tableへ保存するかを定めるものではありません。

managed-here Database上でkeyless Relationをどう保存するかは、`ISSUE-000024` / GitHub Issue #244で別に扱います。

したがって、このADRを根拠に次を自動生成しません。

- `sales.order_line.order_id`というSemantic attribute
- `order_lines.order_id`等のphysical column
- physical Foreign Key
- PostgreSQL native SQL

物理realizationが未定義なら、runtimeは未解決として停止します。

## 実証したReference

### authoring

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

### runtime parameter準備

mapping済みOrder rowに次の値がある場合、

```text
sales.order.order_id = 1001
```

明示Bindingからのみ、次を準備します。

```text
order_id = 1001
```

mapped columnの順序、raw column名、physical Foreign Key、Persistence Mappingから値を推測しません。

### collection result

既存`ContractQueryExecutor`をそのまま再利用し、`many` resultについて次を区別できることを確認しました。

```text
0 row        -> 正常な空collection
複数row      -> 正常なsales.order_line[]
Executor error -> Execution Error
```

### Environment composition

Referenceの`development`では、次を明示的に解決します。

```text
order_entry.sales
  -> sales_core.dev
  -> database.query.postgres
```

BindingやRegistry entryが欠けている場合は、providerやDatabase definitionから補完しません。

### physical realization欠落時

PostgreSQL executor Capabilityまで解決できても、`sales_core/sales_core.order_line/find_by_order`のSQL realizationが存在しなければ、physical Runnerを呼び出す前に明示的に停止します。

Environment compositionが成立していることを、物理SQLを発明する根拠にはしません。

## 影響

### 良くなる点

- Application PatternをRelation-onlyのintentとして維持できる
- keylessな`many` Relationを無理にtarget-side keyへ変換しなくてよい
- Driver都合でSemantic attributeを追加する必要がない
- Database Contract Queryのtyped契約を再利用できる
- logical DatasourceとEnvironment instanceを分離できる
- provider / physical SQLをDriver Coreへ固定しない
- Binding sourceとDiagnosticをSource Indexで追跡できる
- 別Database / provider / realizationへの差し替え余地を残せる

### 増える負担

- Patternとは別にApplication固有realizationをauthoringする必要がある
- source attribute、Datasource、Contract、Query、inputを横断Validationする必要がある
- Environment側にContract Query executor Bindingが必要になる
- managed-here Databaseでは、keyless Relationのphysical storageを別途設計する必要がある

これらは、Semantic・Application intent・Database契約・物理実現を混同しないために受け入れます。

## 採用しない案

### すべてのRelationへRelationKeyを必須化する

keylessな業務Relationまでtarget-side identity attributeを要求し、runtime都合をSemanticへ逆流させるため採用しません。

### `OrderLine.orderId`をDriver都合で追加する

OrderLine自身の業務Semanticとして必要だというEvidenceがないため採用しません。将来、独立した業務上の必要性が確認された場合は別判断として追加できます。

### header-detail PatternへDatasource / Queryを直接追加する

Application intentと取得realizationを再び一体化し、Patternの交換可能性を下げるため採用しません。

### physical Foreign Key / Persistence Mappingからtraversalを推測する

物理構造をSemantic Relationの権威へしてしまうため採用しません。

### 汎用ORM association / generic join engineへ一般化する

第一Referenceで必要な境界を越えて抽象化するため採用しません。

## 検証

PR #235〜#243で、少なくとも次を確認しました。

- Canonical modelとWorld storage
- Validation
- dedicated authoring parser / Workspace Loader
- Source Index / Diagnostic projection
- 実Referenceのtyped Contract QueryとBinding
- source Semantic attributeからruntime parameterを準備
- mapped column順へ依存しない
- RelationKeyなしで実行準備できる
- 0-input traversalを拒否する
- result Data / `many` cardinalityを検査する
- 0件と複数件を正常resultとして区別する
- Execution Errorを空collectionへ変換しない
- PostgreSQL Contract Query Runnerを共通SQL Executorへ接続する
- Environment / Database instance / executor Capabilityを明示compositionする
- providerや別instanceへのfallbackを拒否する
- SQL realization欠落時に物理Runnerを呼ばず停止する

通常GitHub Actionsは2026年9月3日時点でRepository上から明示的に停止されています。したがって、これらの成果をCI greenとは扱いません。CIが明示的に再開された後、関連Conformanceを通常runnerで再実行します。

## 関連文書

- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000011-order-line-semantic-reference.md`
- `plan/issue/000012-header-detail-application-pattern.md`
- `plan/issue/000017-semantic-relation-traversal-key.md`
- `plan/issue/000018-database-contract-query-execution.md`
- `plan/issue/000023-header-detail-relation-traversal-boundary.md`
- `plan/issue/000024-keyless-relation-physical-storage.md`
- GitHub Issue #69 / #117 / #118 / #178 / #179 / #233 / #244
- GitHub PR #235〜#243
- RQ-005 / RQ-009 / RQ-012 / RQ-014
