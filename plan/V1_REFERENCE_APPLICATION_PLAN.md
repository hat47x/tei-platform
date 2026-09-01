# TEI v1 Reference Application 実証計画

## 目的

この文書は、TEI（鼎／テイ）がv1へ進むために必要なApplication側の実証を、既存の`order_entry`を中心に一つの流れとして整理するものです。

Application Pattern、既存認証Compatibility、Driver、Web Target、Editor、AI支援を別々に実装するだけでは、「一つの業務ApplicationがTEIの各層をどのように通るのか」を確認しにくくなります。

そこで同じReference Applicationを段階的に育て、前の段階で確認した意味を次の段階がそのまま利用する進め方を採ります。

期限は置きません。次へ進む条件は日付ではなく、ReferenceとConformanceから確認できるEvidenceで判断します。

## v1 Reference Application

v1では、理想化した新しいサンプルを別に作らず、既存の`reference/workspace/applications/order-entry`を育てます。

現在の`order_entry`には、少なくとも次の資産があります。

- `sales`と`customer`のSemantic Namespace
- `sales.order`と`sales.order_line`
- `sales.order.customer` Relation
- `sales.order.lines -> sales.order_line many` Relation
- `sales_core`へのread-write接続
- `master_data`へのread-only接続
- Database Contract
- native SQL Query
- `sales.order.submit` Capability
- 5種類のApplication Pattern
- Source-to-Canonical Traceability
- Repository自身から再生成できるCanonical World

M0.xxから積み上げた資産を捨てず、そのままApplication側のv1実証へ接続します。

## 現在地

Stage 1「Application Pattern」は完了しました。

候補としていた5種類は、すべて同じReference Workspace上で実Referenceまで成立しています。

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}

pattern order_entry.order_record record {
    data sales.order
}

pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}

pattern order_entry.submit_order command {
    capability sales.order.submit
}

pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

この結果は、5分類をTEI全体の永久的な標準語彙として確定するものではありません。具体的なReference ApplicationでApplication上の意図として成立することを確認した段階です。

現在の主線は、既存認証Compatibility #68とValidationのApplication上の構成を揃え、Gate Aを評価することです。Gate Aを通過してからApplication Driver #69の主実装へ進みます。

## v1で確認する業務シナリオ

最初の業務シナリオは、「受注を探し、開き、顧客を参照し、明細を編集し、内容と利用権限を確認して確定する」という一続きの操作です。

```text
受注一覧を探す
    ↓
受注を開く
    ↓
顧客を参照する
    ↓
受注明細を編集する
    ↓
入力内容を検証する
    ↓
利用権限を確認する
    ↓
受注を確定する
    ↓
結果を観測する
```

このシナリオをApplication Pattern、Driver、Web Targetで別々に作り直しません。同じApplication上の意味を共有し、それぞれの層では必要なBindingやProjectionだけを追加します。

## Stage 1 — Application Pattern: 完了

v1では次の5種類を実証候補として扱いました。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

5種類に共通する重要な結果は、Patternが「既存資産の意味を書き直す場所」ではなく、「Applicationがどの既存identityをどの意図で利用するか」を表す場所として成立したことです。

### `browse`

既存Queryをidentityで参照し、native SQLとDatasourceをPatternへ複製せず成立しました。

### `record`

`sales.order`をApplication上の一件として扱う意味だけで成立し、一件取得用QueryやDatasourceは必要ありませんでした。

### `reference`

Semantic Relation `sales.order.customer`とlogical datasource `master`の組み合わせで成立しました。

Relationのsource / targetはSemantic側、Datasource accessはApplicationのDatabaseRequirementを正本とします。

### `command`

`sales.order.submit` Capability identityだけで成立しました。

Capabilityのinput / output、implementation、DatasourceをPatternへ複製していません。

### `header-detail`

`header-detail`の前提として、まず`sales.OrderLine`と`Order 1:N OrderLine`をSemantic側へ独立して追加しました。

```tei
relation lines to sales.order_line many
```

その後、Application PatternはRelationだけを参照する形で成立しました。

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

Header Data、Detail Data、cardinalityはRelationを正本とします。

この実証では、Semantic RelationからPersistence Mapping、Database Foreign Key、Database Contract、Migrationを暗黙生成していません。`header-detail`をWebのHeader Form + Gridという画面テンプレートにも固定していません。

詳細なEvidenceと設計境界は[`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)を参照します。

## Gate A — Application上の意味をDriverへ渡せる

Stage 1が完了しただけではGate A通過とはしません。

Gate Aでは、少なくとも次をRepository上のEvidenceから一続きに説明できる必要があります。

- browse対象
- record対象
- reference対象とlogical datasource
- header-detail Relation
- Validation
- commandとして呼ぶCapability
- Authentication / Authorization Capabilityと対象action / resource
- read / read-write Datasource境界
- Capability Bindingのidentity
- Business Result / Execution Error境界
- PatternのCanonical identityとauthoring source

Application Pattern側からは、5 PatternのTarget非依存情報が揃いました。

残る主な作業は、既存認証Compatibility #68とValidationのApplication上の構成です。Capability結果境界はADR-0020 / [`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md)を利用します。

Gate Aでは完成したログイン画面やproduction-grade IdPを求めません。DriverがApplication上の意味を、UI固有設定やlegacy schemaの物理構造から初めて推測しなくてよい状態を求めます。

## Stage 2 — Application Driver

Gate A通過後、Issue #69 / `ISSUE-000006`で設計検証用Driverの最小vertical sliceを実装します。

最初のDriverはproduction Runtimeの簡易版ではありません。UIなしでApplication設計を動かし、意味の欠落や境界の誤りを早く見つけるための実証環境です。

最小Scenarioは次です。

1. browseで受注一覧を取得する
2. recordで一件を開く
3. referenceで顧客を解決する
4. header-detailでOrderLineを扱う
5. Validationを実行する
6. Authorizationを評価する
7. commandで`sales.order.submit`を呼び出す
8. Business ResultとExecution Errorを区別する
9. 必要に応じてObservationを確認する

ここで初めて実データの保存・取得に`OrderLine` Persistenceが必要になる場合は、そのEvidenceに基づいてPersistence Mapping、Contract、FK等を設計します。

## Gate B — UIなしで主要シナリオを再現できる

Driverを実装しただけではGate B通過とはしません。

Repositoryだけから主要Scenarioを再現し、Application上の意味、Business Result、Execution Error、Observationを区別したまま第一Web Targetへ渡せることを確認します。

## Stage 3 — 第一Web Target

Gate B通過後、Issue #70 / `ISSUE-000007`で最初のWeb Projectionを実証します。

最初から巨大なTarget-neutral Presentation IRを作りません。Driverで成立したApplication上の意味をWebへProjectionし、その過程から次を分けます。

- 複数Targetでも残りそうなPresentation上の意味
- Web固有の表現とInteraction

HTTP status、DOM、Widget、CSS、Web frameworkをCanonicalなApplicationの意味へ逆流させません。

## Gate C — 共通意味とWeb固有差を分離できる

第一Web Targetが表示できただけではGate C通過とはしません。

DriverとWebが同じApplication上の意味を共有し、どこまでが共通候補で、どこからがWeb固有差かをEvidenceから説明できることを求めます。

Gate Cを通過してからEditor / AI支援の保留を解除するか判断します。

## Editor / AI支援

EditorやAI支援は、意味モデルが固まる前に巨大な機能として先行させません。

ただし、保留は「後回しにして忘れる」という意味ではありません。各Stageで得られたSource Index、Canonical identity、Diagnostic、Pattern、Driver Evidenceを、将来EditorやAIが利用しやすい形で残します。

## 第二Target

第一Web TargetだけでTarget-neutral設計が完成したとはみなしません。

第二Targetで同じApplication上の意味を利用し、Webで共通だと考えた要素が本当にTarget-neutralかを圧力試験します。

第二Targetで境界修正が必要になること自体は失敗ではありません。むしろ、Web固有の都合を共通モデルへ持ち上げすぎていないかを検出するための実証です。

## Evidence Gateの考え方

各Gateは進捗率の区切りではありません。

次の問いにRepository上のEvidenceで答えられるかを確認します。

### Gate A

Application上の意味をDriverが推測せず受け取れるか。

### Gate B

UIなしで主要業務Scenarioを再現できるか。

### Gate C

第一Targetから共通意味とTarget固有差を分離できるか。

このGate構造によって、WebやEditorの実装都合から上位モデルを先回りして固定することを避けます。

## 現在の優先順位

1. 既存認証Compatibility #68を進める
2. ValidationのApplication上の構成を実例から確定する
3. Gate Aを監査する
4. Gate A通過後にDriver #69へ進む
5. Driverで必要になったPersistence / Bindingの不足をEvidenceから補う
6. Gate B通過後に第一Web Target #70へ進む

Application Patternの5分類をさらに抽象化したり、共有Catalogを先に設計したりすることは、現在の優先事項ではありません。

## 関連文書

- [`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)
- [`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md)
- [`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md)
- [`issue/000005-application-pattern-reference.md`](issue/000005-application-pattern-reference.md)
- [`issue/000006-application-driver-vertical-slice.md`](issue/000006-application-driver-vertical-slice.md)
- [`issue/000011-order-line-semantic-reference.md`](issue/000011-order-line-semantic-reference.md)
- [`issue/000012-header-detail-application-pattern.md`](issue/000012-header-detail-application-pattern.md)
- ADR-0017 / ADR-0018 / ADR-0019 / ADR-0020
