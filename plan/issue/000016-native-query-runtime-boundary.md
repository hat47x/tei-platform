# ISSUE-000016: native Queryの実行境界と結果mappingの正本を確立する

- 状態: In Progress
- GitHub Issue: #167
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- 関連PR: #166
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

Application Driverの第五縦切りで、`browse` PatternからCanonical Queryとlogical Datasourceを解決し、その意味をDriverで作り直さず`BrowseQueryExecutor`へ渡す境界まで成立しました。

Referenceでは次の経路を追跡できます。

```text
order_entry.recent_orders
        ↓ browse Pattern
order_entry.recent_orders Query
        ↓ datasource sales
Application database sales
        ↓
sales_core / read-write
        ↓
BrowseQueryExecutor
```

ここから実PostgreSQLでQueryを動かす段階へ進むと、Driverの便宜だけでは決めてはいけない二つの境界が現れます。

1. Query authoring上のnamed parameterを、Provider固有のplaceholderへ安全に束縛する方法
2. native SQLが返す物理列を、Semantic Dataへ対応付ける正本

このIssueでは、この二つを「native Queryを実行して、その結果を次のApplication上の意味へ渡すための境界」として扱います。

## 現在成立していること

PR #166で次を確認済みです。

- `order_entry.recent_orders`をCanonical IDから解決できる
- Queryが`datasource sales`を正本として保持する
- Application側で`sales -> sales_core / read-write`を解決できる
- native SQLをDriverへ複製しない
- SQL本文から物理Databaseやtableを推測しない
- Workspace LoaderがCanonicalに正規化したQuery Sourceを、Driverがさらに加工しない
- `:from` parameterをExecutorへ渡せる
- 実行時parameterをDriverの実行結果へ保持しない
- Executor障害を正常な空結果へ丸めない

## Query parameter binding

Reference SQLは次のようなnamed parameterを持ちます。

```sql
WHERE o.ordered_at >= :from
```

一方、PostgreSQLの実行時placeholderはProvider固有です。

`:from`を単純な文字列置換で`$1`へ変えるだけでは、SQLの意味を壊す可能性があります。少なくとも次を誤認しない必要があります。

- 文字列リテラル中の`:name`
- line comment / block comment
- PostgreSQL固有構文に現れる`:`
- 同じparameterの複数出現
- `:from`と`:from_date`のようなprefix関係
- quoted identifierやdollar-quoted string

したがって、Driver Coreへ簡易置換処理を置きません。

### 求める境界

Provider / Query Executor側で、次を満たす最小方式を確立します。

- Query authoring上のnamed parameterを保持できる
- 実行時にはProviderが必要とするplaceholderへ安全に変換できる
- parameter名と実行引数の対応が再現可能である
- 同じparameterの複数出現を一貫して扱える
- 値そのものを通常Evidenceへ無条件に残さない
- PostgreSQL固有処理をCanonical Queryの意味へ逆流させない

## Query result mapping

`recent_orders.sql`は、たとえば次の物理列を返します。

```text
order_id
customer_id
ordered_at
status
total_amount
```

一方、Semantic Data `sales.order`には業務上のattributeがあります。

現時点では「このQueryのどの列が、どのSemantic attributeへ対応するか」を表すCanonicalな正本がありません。

この状態でDriverが列名の類似や命名規則から自動対応すると、意味がDriver固有実装へ漏れます。

### まず観測すること

最初の実PostgreSQL Referenceでは、結果をすぐSemantic Dataへ変換しません。

1. Query identityを保持する
2. 実行したlogical Datasourceを保持する
3. native SQLの来歴を保持する
4. 実際に返った列名と値のshapeをraw resultとして観測する
5. `record / reference / header-detail`へ何を渡す必要があるかを確認する
6. そのEvidenceから、mappingの所有層を決める

候補としてQuery、Projection、Application側のBinding等が考えられますが、このIssueの開始時点では固定しません。

## Reference PostgreSQL Executor

最初に必要なのはproduction用Database Runtimeではありません。

`order_entry.recent_orders`を一つだけ実DBで動かし、Driver CoreとProvider固有処理の境界を確認できる小さなReference Executorを作ります。

接続情報はDriver Coreへ埋め込みません。Reference composition側から、Applicationが解決した`target sales_core`に対応する実接続をExecutorへ与えます。

```text
Driver
  ↓ Canonical Query + logical Datasource
Reference composition
  ↓ target databaseに対応する接続
PostgreSQL Query Executor
  ↓ Provider固有parameter binding
actual PostgreSQL
```

## 作業順

1. PostgreSQL named parameter bindingに必要な構文範囲を整理する
2. 既存依存関係で安全に実装できるか確認する
3. 必要なら小さなProvider固有lexer / binderを作る
4. binder単体で文字列・comment・cast・重複parameter等を検証する
5. Reference PostgreSQL Executorを作る
6. `recent_orders`用の実DB fixtureを用意する
7. `:from`を実引数へ束縛してQueryを実行する
8. raw resultの列shapeをEvidenceとして確認する
9. 次の`record`へ渡すために不足する意味を抽出する
10. result mappingの所有層を決め、必要ならADRへ昇格する

## 完了条件

- [ ] Driver CoreにDB固有placeholder処理が入っていない
- [ ] named parameterを安全かつ再現可能にPostgreSQLへ束縛できる
- [ ] 文字列・comment等をparameterと誤認しない
- [ ] `order_entry.recent_orders`を実PostgreSQLで実行できる
- [ ] `sales -> sales_core`の明示関係から実行先へ到達している
- [ ] Query identity / Datasource / native SQLの来歴を追跡できる
- [ ] parameter値を通常Evidenceへ無条件に保存しない
- [ ] raw resultとSemanticへ対応付けた結果を区別できる
- [ ] result mappingの所有層をEvidenceに基づいて決定している
- [ ] mappingが未定義の間、Driverが推測で補っていない
- [ ] PostgreSQL固有実装をCoreの意味として固定していない

## 非目標

- 汎用ORM
- SQL全方言を扱う完全parserの内製
- production用Connection Pool / HA
- Driver固有の業務Data model
- 列名一致だけによるSemantic mapping
- Target UI向けViewModel
- Query実行結果をCanonical Worldの正本へ昇格すること

## 判断をADRへ上げる条件

parameter bindingのProvider所有は既存の境界から自然に導ける範囲では、すぐADRを増やしません。

一方、Query result mappingについて、Query / Projection / Application等の所有境界を長期的に固定する必要が生じた場合は、実PostgreSQL ReferenceのEvidenceを添えてADR化します。

## 関連

- `plan/issue/000006-application-driver-vertical-slice.md`
- GitHub Issue #69 / #167
- PR #166
- `reference/workspace/applications/order-entry/queries/recent_orders.tei`
- `reference/workspace/applications/order-entry/queries/recent_orders.sql`
