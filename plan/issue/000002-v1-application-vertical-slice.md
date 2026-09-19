# ISSUE-000002: v1へ向けたApplication vertical sliceを定義する

- 状態: Done

## 背景

TEIは、Semantic、Database、Deployment、Plugin / Runtimeの各境界を個別にかなり深く検証してきました。

Product Requirements Gap Auditでは、次の大きな不確実性が「一つの業務Applicationを設計し、Driverで検証し、第一TargetへProjectionする」という利用者視点の流れに集中していることが分かりました。

v1を機能一覧から逆算せず、実用的なvertical sliceから定義します。

## 関連するProduct Requirement

- RQ-004 Native asset
- RQ-005 ApplicationとDatabaseのN:M
- RQ-008 Capability単位の置換
- RQ-009 既存認証互換
- RQ-010 Editor
- RQ-011 Projection
- RQ-012 Driver
- RQ-013 Target-neutral Presentation
- RQ-014 standard Application Pattern
- RQ-017 二重管理の最小化
- RQ-018 Command / Observation
- RQ-021 Reference / Conformance

## Reference Application

新しいサンプルを別に作るのではなく、既存の`reference/workspace/applications/order-entry`をv1 Reference Applicationへ育てます。

現在の`order_entry`には、既に次の性質があります。

- `sales`と`customer`のSemantic Namespaceを利用する
- `sales_core`をread-writeで利用する
- `master_data`をread-onlyで利用する
- DatabaseごとにContract Requirementを持つ
- native SQL Queryを利用する
- `sales.order.submit` CapabilityがSemantic側に存在する

つまり、v1で新たに検証したいmulti-DB Application、reference lookup、native source、Capabilityを既存Referenceの延長で扱えます。

Referenceを増やしすぎず、過去のM0.xxで蓄積したEvidenceを同じ業務Applicationへ接続できることも利点です。

## 業務シナリオ

最初のv1 vertical sliceは「受注入力」を題材にします。

### 1. 受注一覧を探す

- browse / search Pattern
- `sales_core`から受注一覧を取得
- 既存native SQL Queryを利用
- 条件、並び順、ページング等は必要最小限から始める

### 2. 受注を開く

- record Pattern
- 受注Headerを表示・編集
- 顧客IDから`master_data`をreference lookup
- 顧客名称等は参照情報として表示するが、`sales_core`へ複製するかどうかは業務意味に基づき明示する

### 3. 明細を扱う

- header-detail Pattern
- `Order`と`OrderLine`の1:N関係を追加する
- 明細の追加・変更・削除をDriver上で再現する
- relationとpersistence FKを同一視せず、既存のSemantic / Persistence境界を維持する

### 4. 入力内容を検証する

- required / type / domain validation
- 合計値等、複数fieldにまたがるruleを少なくとも一つ扱う
- semantic rule、Application rule、provider constraintを必要以上に一つへ押し込めない

### 5. 受注を確定する

- command Pattern
- 既存の`sales.order.submit` Capabilityを利用する
- `draft → submitted`等の最小state transitionを扱う
- command acknowledgementとactual stateを必要に応じて分ける
- custom implementationを少なくとも一箇所接続する

### 6. 利用者を確認する

既存認証互換は`ISSUE-000003`で並行して実証します。

v1 Referenceへ統合するときは、TEI標準User schemaへ作り替えるのではなく、既存認証CapabilityをApplicationから利用する形にします。

## Database構成

既存の2 Database構成を維持します。

```text
order_entry
├─ sales  → sales_core  (read-write)
└─ master → master_data (read)
```

認証Referenceが独立Databaseを必要とする場合は、`ISSUE-000003`の実験結果を見て第三Datasourceとして追加するか、外部Capabilityとして接続するかを決めます。ここでは先に固定しません。

## Application Pattern

v1 Referenceで最初に検証するPatternは次の5つです。

1. `browse`
2. `record`
3. `reference`
4. `header-detail`
5. `command`

これらを最初からCore primitiveとして固定しません。

まず`order_entry`で必要な意味と差分を調べ、Semantic Kernel、Application-owned definition、Projection hintのどこへ属するかを比較します。

Patternは「画面テンプレート」だけにはしません。同じPatternがDriver、Web Projection、将来のDesktop等でも利用できる、上位のApplication intentとして成立するかを確かめます。

## Driver

Driverの最初の役割は、本番Runtimeを模倣することではありません。

`order_entry`について次を閉じた環境で実行できることを目標とします。

- Queryを実行して一覧を得る
- Recordを読み込む
- Referenceを解決する
- Header / Detailを編集する
- Validationを実行する
- Commandを呼び出す
- resulting stateを観測する

UIが未完成でも、この業務シナリオをprogrammaticに再現できればDriverの第一段階とみなします。

## 第一Target

第一Targetは**Web**を基本候補とします。

理由は、業務Applicationのbrowse / record / header-detailを最小構成で確認しやすく、custom UIとの接続も検証しやすいためです。

ただし、ここで特定のWeb frameworkをCore設計へ持ち込みません。

まず`order_entry`をWebへ出すために必要なPresentation / Interactionの意味を抽出し、その後第二Targetとの差からTarget-neutral境界を圧力試験します。

## Editor

最初のEditorは、完成した統合開発環境を目指しません。

v1 Reference上で、少なくとも次を同じCanonical meaningから確認できる最小Projectionを対象とします。

- ApplicationとDatasourceの関係
- Data / Relation
- Application Pattern
- Validation / Command
- Source / Diagnostic
- Driver実行結果

高密度な設計ビューを優先し、低密度なウィザードを多数並べる方向へは進めません。

## vertical sliceの全体像

```text
Authoring source
    ↓
Canonical Semantic / Application Meaning
    ↓
Application Pattern
    ↓
Validation / Driver
    ↓
Database / Capability Binding
    ↓
Presentation Projection
    ↓
Web Target
    ↓
Observation / Evidence

Editor / AI
    ↕
Canonical Meaning + Source / Decision History
```

## 実装順

大きな抽象を先に作らないため、次の順で進めます。

### Stage A — Application Pattern

`order_entry`へOrderLineと5種類のPatternを導入し、PatternのownershipとCanonical境界を検証します。

### Stage B — Driver

UIを作る前に、同じReferenceの業務シナリオをprogrammaticに実行します。

### Stage C — 第一Target

Driverで意味が確かめられたApplicationをWebへProjectionします。

### Stage D — Editor

同じvertical sliceを編集・検査できる最小高密度Projectionを作ります。

### Stage E — AI支援

PatternやDecisionが実資産として存在してから、知識探索と確認提案を接続します。

### Stage F — 第二Target

Webで得た設計を第二Targetへ接続し、Target-neutral境界を検証します。

既存認証Compatibility ReferenceはStage A〜Bと並行して進め、安定した時点でv1 Referenceへ統合します。

## 非目標

- 最初から全Targetを完成させること
- あらゆる業務機能を標準Pattern化すること
- production HAや大規模負荷を同時に完成させること
- Editorの見た目を先に作り込むこと
- Web frameworkの都合からPresentation Coreを先に決めること
- 認証をTEI標準schemaへ統一すること

## 後続Issue

この定義から、次の作業へ分解します。

- `ISSUE-000003`: 既存認証Compatibility Reference
- `ISSUE-000005`: `order_entry`でstandard Application Patternを実証する
- `ISSUE-000006`: Application Driverの最小vertical sliceを実装する
- `ISSUE-000007`: 第一Web TargetへのProjection境界を設計する

Editor / AI / 第二Targetは、Stage A〜Cから得られる意味を見てからIssue化します。現時点で先回りして詳細interfaceを固定しません。

## 完了確認

v1へ向けた最小Application vertical sliceを、既存`order_entry`を基準に、業務シナリオ、Application Pattern、Database、Capability、Driver、第一Target、Editorへの接続順まで具体化できました。

後続作業へ分解できる状態になったため、このIssueは完了とします。
