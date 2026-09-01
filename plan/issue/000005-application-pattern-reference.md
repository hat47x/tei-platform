# ISSUE-000005: `order_entry`でApplication Patternを実証する

- 状態: 進行中
- GitHub Issue: #67
- 設計・実装基線: [`APPLICATION_PATTERN_V1_DESIGN.md`](../APPLICATION_PATTERN_V1_DESIGN.md)

## 背景

Product Requirement Gap Auditでは、RQ-014「標準的なApplication Patternを再利用できること」が、Driver、Presentation、Editor、AI支援へ進む前に埋めるべき大きなGapとして残っています。

ただし、v1で最初から「標準Pattern」を確定するわけではありません。既存の`reference/workspace/applications/order-entry`をReference Applicationとして育て、次の5種類をApplicationが所有する意図の候補として一つずつ検証します。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

Patternの所有境界は[ADR-0017](../adr/0017-application-pattern-ownership.md)、正本の配置とfile pathからのidentity分離は[ADR-0018](../adr/0018-application-pattern-authoring-source.md)に従います。

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

Application Patternを単なる画面テンプレートとして固定せず、**同じApplication上の意図をDriverや複数Targetから利用できる単位として成立させられるか**を`order_entry`で確認します。

このIssueはv1 Reference Application実証計画のStage 1です。Pattern側が完了しただけで直ちにDriverへ進むのではなく、`ISSUE-000003`の既存認証Compatibilityなどと合わせてGate A「Application上の意味をDriverへ渡せる」を評価します。

## 現在地

`browse`、`record`、`reference`、`command`の4種類は実Referenceまで完了しています。

最後の候補である`header-detail`の前提となる受注明細Semanticも、`ISSUE-000011` / GitHub Issue #117 / PR #119で成立しました。

現在は`ISSUE-000012` / GitHub Issue #118で、成立済みRelation `sales.order.lines`を使った`header-detail`の最小Canonical契約を実証しています。

```text
browse                    完了
  ↓
record                    完了
  ↓
reference                 完了
  ↓
command                   完了
  ↓
OrderLine + 1:N Relation  完了
  ↓
header-detail             進行中
  ↓
5 Pattern Evidence監査
  ↓
既存認証Compatibility等とGate Aで合流
```

Stage 1、Gate A、Milestone 1の完了はまだ宣言しません。

## 共通基盤として成立しているもの

### Canonical model / Parser

PR #87以降、Patternの最小Canonical modelを導入し、実Referenceから必要性を確認できたKindとfieldだけを段階的に有効化してきました。

現在mainで有効なのは次の4種類です。

- `browse`
- `record`
- `reference`
- `command`

現行Patternは、ID、Kind、Data、Relation、Query、Capability、Datasourceを型付き参照として保持できます。ただし、すべてのfieldをすべてのKindで受理するわけではありません。

実Referenceから必要性が確認できた組み合わせだけを許可し、意味を持たないfieldは明示的に拒否します。

`header-detail`は現在、Relation-onlyの契約を実装・検証中です。

### Loader / Normalization

PR #88で次を実装しました。

- `applications/*/patterns/*.tei`の列挙
- Pattern sourceのParse
- 所属Applicationへの統合
- Canonical ID順のNormalization
- `application-pattern:<Canonical ID>`のsource mapping

### Source Index / rename invariance

PR #89でPattern ID tokenへのSource Index経路を実装しました。

PR #102では実ReferenceのPatternファイル名だけを変更し、Canonical WorldとPattern identityを変えず、Source Artifact pathだけが追随することを確認しています。

### 共通Validation / Diagnostic

PR #95、#99、#103以降、Pattern固有のWorkspace参照整合性は共通`validation.Validate`から一度だけ実行されます。

Diagnosticは`application-pattern:<Canonical ID>` Subjectを利用し、既存Source Indexを通じて元の定義sourceへ戻れます。

## `browse` — 完了

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
```

現時点の契約は次です。

- `data`: 必須
- `query`: 必須
- `datasource`: 任意
- `relation`: 受理しない
- `capability`: 受理しない

既存Query `order_entry.recent_orders`がDatasource `sales`を正本として保持するため、実ReferenceではPattern側へ同じDatasourceを重複記述していません。

## `record` — 完了

Issue #104、PR #105 / #106で実証しました。

```tei
pattern order_entry.order_record record {
    data sales.order
}
```

`record`は「ApplicationがどのSemantic Dataを一件として扱うか」という意味だけで成立しました。

既存Database Contractの`find_by_id`や将来Driverの一件取得を理由に、取得QueryやDatasourceをPatternへ持ち込んでいません。

## `reference` — 完了

Issue #107、PR #108 / #109で実証しました。

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

Relation `sales.order.customer`がsource `sales.order`とtarget `customer.customer`をすでに持つため、DataやtargetをPatternへ重複記述していません。

Datasource `master`はApplication固有の論理的な参照経路なのでPatternに保持します。ただし、`access read`そのものはApplicationの`DatabaseRequirement`を正本とします。

## `command` — 完了

Issue #110、PR #114 / #116で実証しました。

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

Capability `sales.order.submit`がinput `order sales.order`とoutput `sales.order`をすでに持つため、対象Dataや結果型をPatternへ重複記述していません。

DatasourceやCustom Go等の実装方式もPatternへ持ち込まず、Capability Binding / Realization / Driver側の責務として分離しています。

既存`output sales.order`はADR-0020に従うBusiness Resultとして維持し、Execution ErrorやObservationもPatternへ混在させていません。

## `OrderLine` Semantic Reference — 完了

`ISSUE-000011` / GitHub Issue #117 / PR #119で、`header-detail`の前提となるSemanticを独立に追加しました。

```tei
type OrderLineId from std.identifier

data OrderLine {
    attribute orderLineId type sales.order_line_id
    attribute lineNumber type std.integer
    attribute itemCode type std.code
    attribute quantity type std.decimal
    attribute unitPrice type std.money
}
```

既存`Order`には次のRelationを追加しています。

```tei
relation lines to sales.order_line many
```

ここではSemantic上の関係だけを追加し、Persistence Mapping、Database Foreign Key、Database Contract、Application Patternを同時には追加していません。

`OrderLine`へ`orderId`属性も先回りして追加せず、物理DBで関係を実現する方法をSemanticへ逆流させない境界を確認しました。

## `header-detail` — 進行中

`ISSUE-000012` / GitHub Issue #118では、成立済みRelation `sales.order.lines`をApplication上のHeader / Detail意図として利用できるかを実証します。

第一候補はRelation-onlyです。

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

Relationがsource `sales.order`、target `sales.order_line`、cardinality `many`をすでに持つため、Header Data、Detail Data、cardinalityをPatternへ重複記述しません。

`header-detail`は複数Detailを扱う意図なので、Relationが`many`であることはWorkspace Validationから確認します。ただしcardinalityそのものをPatternへコピーしません。

Datasource、Query、Persistence、Database Foreign Key、UI layoutも、実Referenceを成立させるために必要だというEvidenceが得られるまでPatternへ追加しません。

## 4種類のPatternと前提Semanticから得た共通原則

### 既存の意味をPatternへ複製しない

Patternは既存のSemantic Data、Relation、Query、CapabilityをCanonical IDで参照します。

参照先がすでに保持しているsource、target、input、output、Datasourceなどを、理由なくPatternへ書き写しません。

### Kindごとの意味を共通化しすぎない

Patternに共通するのは主にidentityとKindです。

必要な参照は種類ごとに異なるため、すべてのPatternへ同じfieldを持たせる設計にはしません。

一方、任意の`role -> string` Mapへ寄せて型を失うこともしません。

### 実現方法をApplication上の意図へ持ち込まない

native SQL、Datasource access、Capability implementation、Persistence Mapping、Database Foreign Key、Web Widgetなどは、それぞれ既存の責務境界へ残します。

Patternは、それらを利用するApplication上の意図だけを扱います。

### 新しい業務SemanticとPatternを分ける

`header-detail`では、Patternと`OrderLine`を同時に追加しませんでした。

まずSemantic Relationだけを独立して成立させ、その後Application上の利用意図を検証することで、Semantic不足とPattern設計の問題を切り分けています。

### 未検証の一般化を先に固定しない

4種類が成立し、5種類目の最小契約が見えたことは、5分類をTEI全体の永久的な標準語彙へ昇格できることを意味しません。

Driver、第一Web Target、第二Targetから得られるEvidenceを使い、Applicationに残す意味、共有知識へ持ち上げる意味、Projectionへ下げる情報を見直します。

## 所有境界

ADR-0017に従い、次の層を混同しません。

1. Semantic上の意味 — Data / Relation / Capabilityなど
2. Application上の意図 — browse / record / reference / command / header-detailなど
3. Realization / Persistence — Query、Binding、Mapping、Foreign Key、Migrationなど
4. Projection / Target — Webなどに固有の表現とInteraction

同じ業務シナリオを複数層へコピーして正本を増やすのではなく、各層が自分の意味だけを持ち、identityで接続する構造を維持します。

## Gate Aへ渡す情報

このIssueだけでGate Aを完了とはしませんが、Application Pattern側から少なくとも次を説明できるEvidenceを揃えます。

- 何を`browse`するか
- 一つの`record`として何を扱うか
- どのRelationを、どのDatasourceから`reference`するか
- どのCapabilityを`command`として呼ぶか
- HeaderとDetailをどのRelationで一まとまりとして扱うか
- どのValidationを実行するか
- どのDatasourceがread-onlyか
- Patternの正本とCanonical identityをどのように追跡するか

Authorizationは`ISSUE-000003`の既存認証Compatibilityから合流させます。

CapabilityのBusiness Result / Execution Error境界はADR-0020 / `ISSUE-000009`から受け取ります。

## Conformance

### 確認済み

- PatternをApplication modelで保持できる
- Pattern IDをfile pathから独立して保持できる
- Pattern sourceを`patterns/*.tei`からLoaderで読み込める
- Pattern順序をCanonical IDで正規化できる
- Source IndexからPattern ID tokenへ戻れる
- 共通ValidationからPattern Diagnosticを一度だけ返せる
- `browse`が既存Query / native SQLを重複せず利用できる
- `record`がSemantic Data一件の意味だけで成立する
- `reference`がSemantic Relationとlogical datasourceの組み合わせで成立する
- `reference`がwrite-only Datasourceを拒否する
- `command`がCapability identityだけで成立する
- `command`がCapability input / output / implementationを重複保持しない
- `browse` / `record` / `reference` / `command`が同じApplicationで共存する
- `OrderLine`と1:N RelationがPersistence / FK / Contractを追加せずSemanticだけで成立する
- Reference WorldをRepository自身から再生成できる

### 次に確認すること

- `header-detail`が成立済みRelationだけで自然に表現できる
- Relationの`many` cardinalityをPatternへ複製せずValidationできる
- 5種類のPatternが同じApplicationで共存する
- Driverが利用するためにWeb固有情報を必要としない

## 非目標

- Patternの網羅的なcatalog化
- 5分類を現時点でTEI標準語彙として確定すること
- Patternごとの完成UI
- responsive designなどのTarget固有表現
- automatic UI design
- PatternをSemantic Kernelへ一括追加すること
- AIによるPattern推薦
- Pattern Catalogの共有方式を先に決めること
- Patternの継承・合成を先に実装すること
- `OrderLine`追加と同時にPersistence / FK / Contractを作ること
- このIssueだけでGate Aを完了させること

## 完了条件

`order_entry`の業務シナリオで候補となる5 Patternを一続きに表現し、どこまでがApplication上の意図で、どこからがSemantic、Realization / Persistence、Projection / Target固有情報かを説明できることを完了条件とします。

さらに、Patternの正本、Canonical identity、Source Index、Reference Worldの再現性をRepository内のReference / Conformanceから確認できること。その同じApplication上の意味を後続Driverが利用でき、`ISSUE-000003`のAuthorization境界などと合わせてGate Aを評価できるEvidenceを残すことを求めます。

現在は`browse`、`record`、`reference`、`command`と、`header-detail`の前提Semanticまで完了しています。

## 関連文書

- `plan/APPLICATION_PATTERN_V1_DESIGN.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0018-application-pattern-authoring-source.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000010-command-application-pattern.md`
- `plan/issue/000011-order-line-semantic-reference.md`
- `plan/issue/000012-header-detail-application-pattern.md`
