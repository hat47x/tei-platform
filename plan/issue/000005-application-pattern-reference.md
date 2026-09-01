# ISSUE-000005: `order_entry`でstandard Application Patternを実証する

- 状態: 着手可能
- GitHub Issue: #67
- 実装前設計: [`APPLICATION_PATTERN_V1_DESIGN.md`](../APPLICATION_PATTERN_V1_DESIGN.md)

## 背景

Product Requirement Gap Auditでは、RQ-014「標準的なApplication Patternを再利用できること」が、Driver、Presentation、Editor、AI支援へ進む前に埋めるべき大きなGapとして残っています。

v1では、既存の`reference/workspace/applications/order-entry`をReference Applicationとして育て、次の5 Patternを具体的な業務シナリオで検証します。

- browse
- record
- reference
- header-detail
- command

Patternの所有境界は[ADR-0017](../adr/0017-application-pattern-ownership.md)、正本の配置とfile pathからのidentity分離は[ADR-0018](../adr/0018-application-pattern-authoring-source.md)で定めます。

## 関連するProduct Requirement

- RQ-001 業務上の意味
- RQ-003 暗黙の複数正本を作らない
- RQ-005 Application ↔ Database N:M
- RQ-011 Projection
- RQ-012 Driver
- RQ-014 Application Pattern
- RQ-017 二重管理の最小化
- RQ-021 Reference / Conformance

## 目的

Application Patternを単なる画面テンプレートとして固定せず、**同じApplication上の意図をDriverや複数Targetから利用できる再利用単位として成立させられるか**を`order_entry`で確認します。

このIssueはv1 Reference Application PlanのStage 1です。完了しただけで直ちにDriverへ進むのではなく、`ISSUE-000003`の既存認証Compatibilityと合わせてGate A「Application上の意味をDriverへ渡せる」を評価します。

## 実装の進め方

5 Patternと新しい業務Semanticを一度に追加しません。既存資産だけで成立する`browse`から始め、Pattern基盤そのものを先に検証します。

### Step 1 — PatternのCanonicalモデル

- ApplicationがPatternを子として所有する
- Patternは明示的なCanonical IDを持つ
- Pattern KindをApplication層の小さな語彙として扱う
- file pathやsource positionをCanonicalモデルへ持ち込まない
- Pattern順序をCanonical IDで安定して正規化する

### Step 2 — 正本 / Loader / Source Index

- Application配下に`patterns/*.tei`を追加する
- 一ファイル一Patternから始める
- Workspace LoaderがPatternを所属Applicationへ統合する
- `application-pattern:<Canonical ID>`をSource Indexで追跡する
- ファイルを改名してもCanonical Worldが変わらないことを確認する

### Step 3 — `browse` Reference

既存の次の資産をそのまま利用します。

- Semantic Data `sales.order`
- Query `order_entry.recent_orders`
- logical datasource `sales`
- native SQL `queries/recent_orders.sql`

SQL本文やQuery定義をPattern側へコピーせず、Patternは既存のidentityを参照します。

この段階で、Canonicalモデル、Parser、Loader、Source Index、Diagnostic、Reference Worldまでを一続きに成立させます。

### Step 4 — `record` / `reference` / `command`

`browse`でPattern基盤が成立した後、既存のSemantic Data、Relation、Capabilityを使って段階的に追加します。

- `record`: `sales.order`
- `reference`: `sales.order.customer` → `customer.customer`
- `command`: `sales.order.submit`

この段階でもTarget固有の画面属性は追加しません。

### Step 5 — `header-detail`

`header-detail`は、新しい`sales.OrderLine`と対応RelationをSemantic側へ追加した後に検証します。

Pattern基盤の変更と新しい業務Semanticの追加を分けることで、問題が起きたときに原因となった境界を追いやすくします。

## `OrderLine`の追加

`header-detail`では、概念上、次の関係を扱います。

```text
Order 1 ── N OrderLine
```

`OrderLine`にはReferenceとして十分な最小属性だけを置きます。

- line identity
- item / product referenceに相当する識別子
- quantity
- unit price
- amount

最終的な属性名やtypeは、現行Semantic namingとの整合を確認して実装時に決めます。

Semantic RelationとDatabase Foreign Keyは別の意味単位として扱い、RelationからFKを暗黙生成しません。

## 5 Patternで確認すること

### browse

複数Recordを検索・列挙するApplication上の意図です。

最初のReferenceでは、既存のnative SQL Queryを利用して受注一覧を取得します。PatternはQueryを所有せず、「何を一覧として扱い、どのQueryを利用するか」を接続します。

Query自身がDatasourceを既に保持しているため、PatternでもDatasourceを重複指定すべきかは最初の実装で比較します。同じ意味を二重に手入力しないことを優先します。

### record

一つの業務RecordをApplication上の一件として扱う意図です。

`sales.order`を基準にし、Application上の意味と、表示可能・編集可能・read-only等のTarget側表現を分離します。

### reference

別の意味対象を参照するApplication上の意図です。

既存の`sales.order.customer` Relationを利用して`customer.customer`を参照し、Semantic RelationをPatternへ複製しません。同時に、Applicationがread-onlyな`master` Datasourceを利用できることを確認します。

### header-detail

一つのHeaderと複数DetailをApplication上の編集単位として扱う意図です。

`Order`と`OrderLine`を使い、次を確認します。

- 1:N関係をUI都合だけで定義しない
- Detailの追加・変更・削除をApplication上の意図として扱える
- persistence ownershipやtransaction boundaryを暗黙に決めない
- ValidationがHeader / Detailをまたげる

### command

状態を変化させる明示的な業務操作です。

既存の`sales.order.submit` Capabilityを利用し、受注確定を最小Referenceとします。単なるCRUD更新と明示的な業務Commandの違いを、Application上でどこまで表す必要があるかを確認します。

## 所有境界

ADR-0017に従い、次の三層を混同しません。

1. Semantic上の意味 — Data / Relation / Capability等
2. Application上の意図 — browse / record / reference / header-detail / command等
3. Projection / Target — Web等に固有の表現とInteraction

現在のPattern名や分類は永久的な標準語彙ではありません。Driver、第一Web Target、第二Targetから得られるEvidenceを使い、Semantic Kernelへ持ち上げる意味、Application層へ残す意味、Projectionへ下げる情報を見直します。

## 正本

ADR-0018に従い、Patternの正本はApplication配下の`patterns/*.tei`へ置きます。

Pattern identityは明示的なCanonical IDとし、file pathやファイル名から導出しません。

同じApplication上の意図について、DSLとEditor内部状態を暗黙の二重正本にはしません。当面のEditorはauthoring sourceのProjectionと編集支援を担います。

## Gate Aへ渡す情報

このIssueだけでGate Aを完了とはしませんが、Driverへ渡すApplication側の意味として少なくとも次を揃えます。

- 何をbrowseするか
- 一つのrecordとして何を扱うか
- 何を別Datasourceからreferenceするか
- HeaderとDetailの関係
- どのValidationを実行するか
- どのCapabilityをcommandとして呼ぶか
- どのDatasourceがread-onlyか

Authorizationは`ISSUE-000003`の既存認証Compatibilityから合流させます。

## Conformance

### Pattern基盤

- PatternをApplication modelから識別できる
- Patternがfile pathを意味上のidentityにしない
- PatternがSemantic Data、Relation、Query、Capability、Datasource等を型を保ったidentityで参照できる
- Pattern順序がsource fileの列挙順へ依存しない
- Source Index / DiagnosticからPattern定義へ戻れる
- ファイル改名だけではCanonical Worldが変わらない

### `browse`

- `order_entry.recent_orders` Queryを参照できる
- native SQLの正本をPattern側へ複製しない
- Queryが持つDatasource情報を不要に二重管理しない
- Reference WorldをRepositoryだけから再生成できる

### 後続Pattern

- `reference`が`master`のread-only Bindingを維持する
- `header-detail`のSemantic RelationとPersistence FKを混同しない
- `command`が既存Capabilityへ接続できる
- Driverが利用するためにWeb固有情報を必要としない

## 非目標

- Patternの網羅的なcatalog化
- Patternごとの完成UI
- responsive design等のTarget固有表現
- automatic UI design
- PatternをSemantic Kernelへ一括追加すること
- AIによるPattern推薦
- Pattern Catalogの共有方式を先に決めること
- Patternの継承・合成を先に実装すること
- このIssueだけでGate Aを完了させること

## 完了条件

`order_entry`の業務シナリオで5 Patternを一続きに表現でき、どこまでがApplication上の意図で、どこからがProjection / Target固有情報かを説明できることを完了条件とします。

さらに、Patternの正本、Canonical identity、Source Index、ファイル改名に対する不変性をRepository内のReference / Conformanceから再現できること。その同じApplication上の意味を後続のDriverが利用でき、`ISSUE-000003`のAuthorization境界と合わせてGate Aを評価できるEvidenceを残すことを求めます。

## 関連文書

- `plan/APPLICATION_PATTERN_V1_DESIGN.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0018-application-pattern-authoring-source.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
