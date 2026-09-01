# ISSUE-000005: `order_entry`でstandard Application Patternを実証する

- Status: Ready

## 背景

Product Requirements Gap Auditでは、RQ-014「標準的なApplication Patternを再利用できること」が未着手であり、Driver、Presentation、Editor、AI支援へ進む前に埋めるべき大きなGapであることが分かりました。

`ISSUE-000002`では、既存の`reference/workspace/applications/order-entry`をv1 Reference Applicationへ育て、次の5つのPatternを実証する方針を採用しました。

- browse
- record
- reference
- header-detail
- command

## 関連するProduct Requirement

- RQ-001 業務上の意味
- RQ-005 Application ↔ Database N:M
- RQ-011 Projection
- RQ-012 Driver
- RQ-014 Application Pattern
- RQ-017 二重管理の最小化
- RQ-021 Reference / Conformance

## 目的

Application Patternを単なる画面テンプレートとして固定せず、**同じ業務上のintentをDriverや複数Targetから利用できる再利用単位として成立させられるか**を、`order_entry`の具体的な業務シナリオから確認します。

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

最終的な属性名やtypeは、現行Semantic namingとの整合を見て実装時に確定します。

Semantic RelationとDatabase Foreign Keyは別の意味単位として扱い、RelationからFKを暗黙生成しません。

## 検証する5 Pattern

### 1. browse

複数Recordを検索・列挙するApplication intentです。

最初のReferenceでは、既存のnative SQL Queryを利用して受注一覧を取得します。

PatternがQueryそのものを所有するのではなく、「一覧として何を提示し、どのQuery / Capabilityを利用するか」を接続できる境界を検討します。

### 2. record

一つの業務Recordを表示・編集するintentです。

`sales.Order`を基準に、表示可能、編集可能、read-only等の差がどの層へ属するかを確認します。

### 3. reference

入力値やRecordから別の意味対象を参照するintentです。

`order_entry`では`customerId`から`master_data`の顧客を参照します。

このPatternにより、Applicationがread-onlyな別Datasourceを自然に利用できることも同時に確認します。

### 4. header-detail

一つのHeaderと複数Detailを一つの編集単位として扱うintentです。

`Order`と`OrderLine`を使い、次を確認します。

- 1:N関係をUI都合だけで定義しない
- Detail追加・変更・削除をApplication intentとして表せる
- persistence ownershipやtransaction boundaryを暗黙に決めない
- ValidationがHeader / Detailをまたげる

### 5. command

状態を変化させる明示的な操作intentです。

既存の`sales.order.submit` Capabilityを利用し、受注確定を最小Referenceとします。

単なるCRUD更新との差をどこまでPattern上で表現する必要があるかを確認します。

## Patternのownershipを比較する

実装前に、Patternの情報をどこへ置くかを比較します。

候補は少なくとも次の3つです。

1. Semantic Kernelのprimitive
2. Application-ownedなCanonical definition
3. Projection / Editor向けのhint

原則として、すべてをSemantic Kernelへ持ち上げません。

同じPatternがDriverと複数Targetで必要になる意味だけをApplication-ownedなintentとして残し、表示専用の差はProjection側へ置くことを第一候補とします。

ただし、これは実験前に決定済みとはしません。

## Authoritative source

Patternを導入しても、同じApplication intentについてDSLとEditor内部状態の二つを暗黙の正本にしません。

初期段階ではauthoring sourceを明示し、EditorはそのProjectionとして扱います。

将来Editor-authored modeを導入する場合は、RQ-003の原則に沿って別途source-of-truthを定めます。

## Conformance

少なくとも次を確認します。

- 5 Patternを同じApplication modelから識別できる
- Patternがfile pathをsemantic identityにしない
- browseがnative SQLのauthoritative sourceを壊さない
- referenceが`master_data` read-only Bindingを維持する
- header-detailのSemantic RelationとPersistence FKを混同しない
- commandが既存Capabilityへ接続できる
- Patternを追加してもexisting normalized meaningを不要に作り直さない
- Source Index / DiagnosticからPattern定義へ戻れる

## 非目標

- Patternの網羅的catalog化
- Patternごとの完成UI
- responsive design等のTarget固有表現
- automatic UI design
- PatternをSemantic Kernelへ一括追加すること
- AIによるPattern推薦

## 完了条件

`order_entry`の具体的な業務シナリオで5 Patternを一続きに表現でき、どこまでがApplication intentで、どこからがProjection / Target固有情報かを説明できること。

その同じモデルを後続のDriverと第一Targetが利用できる状態にします。
