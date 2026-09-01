# ISSUE-000005: `order_entry`でstandard Application Patternを実証する

- 状態: 着手可能
- GitHub Issue: #67

## 背景

Product Requirement Gap Auditでは、RQ-014「標準的なApplication Patternを再利用できること」が未着手であり、Driver、Presentation、Editor、AI支援へ進む前に埋めるべき大きなGapであることが分かりました。

`ISSUE-000002`では、既存の`reference/workspace/applications/order-entry`をv1 Reference Applicationへ育て、次の5つのPatternを実証する方針を採用しています。

- browse
- record
- reference
- header-detail
- command

Patternの初期所有境界は[ADR-0017](../adr/0017-application-pattern-ownership.md)で定めます。v1ではSemantic Kernelへ早い段階から固定せず、Applicationが所有するCanonicalな意図として実証します。

## 関連するProduct Requirement

- RQ-001 業務上の意味
- RQ-005 Application ↔ Database N:M
- RQ-011 Projection
- RQ-012 Driver
- RQ-014 Application Pattern
- RQ-017 二重管理の最小化
- RQ-021 Reference / Conformance

## 目的

Application Patternを単なる画面テンプレートとして固定せず、**同じ業務上の意図をDriverや複数Targetから利用できる再利用単位として成立させられるか**を、`order_entry`の具体的な業務シナリオから確認します。

このIssueは、v1 Reference Application PlanのStage 1に当たります。完了後に直ちにDriver実装へ進むのではなく、既存認証Compatibilityと合わせてGate A「Application上の意味をDriverへ渡せる」を確認します。

## Reference拡張

### OrderLine

現在の`sales.Order`へ明細概念を追加します。

概念上は次を扱います。

```text
Order 1 ── N OrderLine
```

OrderLineには、Referenceとして十分な最小属性だけを置きます。

- line identity
- item / product referenceに相当する識別子
- quantity
- unit price
- amount

最終的な属性名やtypeは、現行Semantic namingとの整合を確認して実装時に確定します。

Semantic RelationとDatabase Foreign Keyは別の意味単位として扱い、RelationからFKを暗黙生成しません。

## 検証する5 Pattern

### 1. browse

複数Recordを検索・列挙するApplication上の意図です。

最初のReferenceでは、既存のnative SQL Queryを利用して受注一覧を取得します。

PatternがQueryそのものを所有するのではなく、「一覧として何を扱い、どのQueryやCapabilityを利用するか」を接続できる境界を検討します。

### 2. record

一つの業務Recordを表示・編集するApplication上の意図です。

`sales.Order`を基準に、表示可能、編集可能、read-onlyなどの差がどの層へ属するかを確認します。

### 3. reference

入力値やRecordから別の意味対象を参照するApplication上の意図です。

`order_entry`では`customerId`から`master_data`の顧客を参照します。

このPatternにより、Applicationがread-onlyな別Datasourceを自然に利用できることも同時に確認します。

### 4. header-detail

一つのHeaderと複数Detailを、一つのApplication上の編集単位として扱う意図です。

`Order`と`OrderLine`を使い、次を確認します。

- 1:N関係をUI都合だけで定義しない
- Detailの追加・変更・削除をApplication上の意図として表せる
- persistence ownershipやtransaction boundaryを暗黙に決めない
- ValidationがHeader / Detailをまたげる

### 5. command

状態を変化させる明示的な業務操作です。

既存の`sales.order.submit` Capabilityを利用し、受注確定を最小Referenceとします。

単なるCRUD更新との差を、どこまでApplication上の意図として表す必要があるかを確認します。

## Patternの所有境界

ADR-0017に従い、v1ではApplication PatternをApplicationが所有するCanonicalな意図として扱います。

次の三層を混同しません。

1. Semantic上の意味 — Data / Relation / Capabilityなど
2. Application上の意図 — browse / record / reference / header-detail / commandなど
3. Projection / Target — Webなどに固有の表現とInteraction

ただし、現在のPattern名や分類を永久的な標準語彙として固定するものではありません。

Driver、第一Web Target、第二Targetから得られるEvidenceによって、Semantic Kernelへ持ち上げるべき意味、Application層へ残す意味、Projectionへ下げる情報を見直します。

## 正本

Patternを導入しても、同じApplication上の意図についてDSLとEditor内部状態の二つを暗黙の正本にしません。

初期段階ではauthoring sourceを正本として明示し、EditorはそのProjectionとして扱います。

将来Editor-authored modeを導入する場合は、RQ-003の原則に沿って、どこを正本とするかを別途定めます。

## Gate Aへ渡す情報

このIssueの成果だけでGate Aを完了とはしませんが、Driverへ渡すApplication側の意味として少なくとも次を揃えます。

- 何をbrowseするか
- 一つのrecordとして何を扱うか
- 何を別Datasourceからreferenceするか
- HeaderとDetailの関係
- どのValidationを実行するか
- どのCapabilityをcommandとして呼ぶか
- どのDatasourceがread-onlyか

Authorizationは`ISSUE-000003`の既存認証Compatibilityから合流させます。

## Conformance

少なくとも次を確認します。

- 5 Patternを同じApplication modelから識別できる
- Patternがfile pathを意味上のidentityにしない
- PatternがSemantic Data、Query、Capability、DatasourceなどをCanonical IDで参照できる
- browseがnative SQLの正本を壊さない
- referenceが`master_data`のread-only Bindingを維持する
- header-detailのSemantic RelationとPersistence FKを混同しない
- commandが既存Capabilityへ接続できる
- Patternを追加しても既存の正規化済み意味表現を不要に作り直さない
- Source Index / DiagnosticからPattern定義へ戻れる
- Driverが利用するためにWeb固有情報を必要としない

## 非目標

- Patternの網羅的catalog化
- Patternごとの完成UI
- responsive designなどのTarget固有表現
- automatic UI design
- PatternをSemantic Kernelへ一括追加すること
- AIによるPattern推薦
- このIssueだけでGate Aを完了させること

## 完了条件

`order_entry`の具体的な業務シナリオで5 Patternを一続きに表現でき、どこまでがApplication上の意図で、どこからがProjection / Target固有情報かを説明できること。

その同じApplication上の意味を後続のDriverが利用できる状態にし、`ISSUE-000003`のAuthorization境界と合わせてGate Aを評価できるEvidenceを残します。

## 関連文書

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
