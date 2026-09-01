# ISSUE-000012: `sales.order.lines`をheader-detail Application Patternとして実証する

- 状態: In Progress
- GitHub Issue: #118
- 実装PR候補: #122
- 前提Issue: `ISSUE-000011` / GitHub Issue #117（完了）
- 上位Issue: `ISSUE-000005` / GitHub Issue #67
- 後続: `ISSUE-000006` / GitHub Issue #69（Application Driver）

## 目的

`sales.order.lines -> sales.order_line many`がSemantic Referenceとして成立したことを受け、そのRelationをApplicationがHeaderと複数Detailの一まとまりとして扱う意図を`header-detail` Patternで表現できることを実証します。

ここで確認するのは、Webの「ヘッダフォーム + 明細グリッド」という画面テンプレートではありません。Targetが変わっても残る可能性があるApplication上の意味だけを抽出し、UI layout、Persistence、Database Foreign Key、取得・保存手段は別の責務として保ちます。

## 前提として成立済みのSemantic

`ISSUE-000011` / GitHub Issue #117 / PR #119で、次がReferenceとして成立しています。

```text
sales.order
    1
    │
    └── many sales.order_line
```

Relation `sales.order.lines`は、Canonical World上で次を保持します。

- source: `sales.order`
- target: `sales.order_line`
- cardinality: `many`

Semantic追加だけではPersistence Mapping、Database Foreign Key、Database Contractが暗黙に増えないことも確認済みです。

## 最小Canonical契約の第一候補

Relationがsource、target、cardinalityをすでに保持するため、第一候補ではRelationだけをPatternに持たせます。

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

次をPatternへ重複記述しない方向から検証します。

- Header Data `sales.order`
- Detail Data `sales.order_line`
- cardinality `many`
- Datasource
- Query
- Persistence Mapping
- Database Foreign Key
- UI layout

実Referenceを成立させるために追加情報が本当に必要だと分かった場合だけ、そのEvidenceをもとに最小のfieldを追加します。

## `many`をValidationで確認する

`header-detail`は、一つのHeaderと複数のDetailを扱うApplication上の意図として実証します。

そのため、参照するRelationが`many`であることは意味上の前提になります。ただし、cardinalityをPatternへコピーしません。

ParserはRelationの実体を知らないため、`relation` fieldの存在だけを構文上確認します。Relationが実在し、cardinalityが`many`であることはWorkspace全体を見られるValidation層で検査します。

`one` Relationを指定した場合は、曖昧に受理せずPattern Subject付きDiagnosticを返す方針を採ります。

## DatasourceやQueryを先に持たせない理由

`header-detail`が「どの関係を一まとまりとして扱うか」だけで成立するなら、取得・保存経路までPatternへ固定する必要はありません。

DatasourceやQueryを先に追加すると、Application上の意図と実現方法が混ざる可能性があります。

後続Driver #69で実際の取得・保存を構成するときに、PatternへDatasourceやQueryが必要だというEvidenceが得られた場合は、その時点で見直します。

## Validationとの境界

HeaderとDetailをまたぐValidationはv1 Driver Scenarioで重要です。しかし、「Header / Detail関係を表すPattern」と「どのValidationを実行するか」は別の論点です。

このIssueでは、`header-detail`を成立させるために必要でないValidation identityをPatternへ先に追加しません。

Driverへ渡すApplication上の情報を監査するときに、Validationの所有位置を別途確認します。

## Persistence / Database Foreign Keyとの境界

Persistence MappingやDatabase Foreign Keyは、このPatternの前提ではありません。

Semantic Relation、Application Pattern、Persistence Mapping、Database Foreign Keyは、それぞれ異なる意味を持ちます。

`header-detail`が成立しただけでは、次を自動生成しません。

- `order_lines` table
- `OrderLine` Persistence Mapping
- `orders -> order_lines` Foreign Key
- Migration
- Database Contractのread/write拡張

物理的な保存・取得をDriverで実証するときに必要になった境界は、別のEvidenceとして追加します。

## 実装する範囲

### Canonical model

- `PatternHeaderDetail`を有効化する
- 既存の型付き`Relation`参照を利用する
- Header / Detail Dataやcardinalityを重複fieldとして追加しない

### Parser

- `header-detail`で`relation`を必須にする
- Data / Query / Capability / Datasourceは受理しない
- 既存Kindの契約を変更しない
- 同じfieldの重複を構文段階で拒否する

### Workspace Validation

- Relationが存在することを確認する
- Relationのcardinalityが`many`であることを確認する
- `one` Relationには`APPLICATION_PATTERN_RELATION_NOT_MANY`を返す
- Kind固有の禁止fieldが混入していないことを確認する
- Diagnosticが既存`application-pattern:<Canonical ID>` Subjectを利用する

### 実Reference

Canonical契約が成立した後、Reference Workspaceへ次を追加します。

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

### Conformance

- `browse` / `record` / `reference` / `command` / `header-detail`の5種類が同じ`order_entry`で共存する
- `header-detail`から`sales.order.lines`をCanonical IDで追跡できる
- Relation source `sales.order`をSemantic側から取得できる
- Relation target `sales.order_line`をSemantic側から取得できる
- Relation cardinality `many`をSemantic側から取得できる
- PatternがHeader / Detail Dataやcardinalityを重複保持しない
- Pattern固有Diagnosticからauthoring sourceへ戻れる
- Reference WorldをRepository自身から再生成できる
- Pattern追加だけではPersistence Mapping / FK / Contractを暗黙に生成しない

## Driverへ渡す意味

このIssueを完了すると、Application Pattern側では5種類の候補が一通りReferenceとして成立します。

Driverには少なくとも次のTarget非依存情報を渡せる状態を目指します。

- browse対象とQuery identity
- record対象
- reference対象RelationとDatasource
- command対象Capability
- header-detail対象Relation
- Relationから得られるHeader / Detail Dataとcardinality

ただし、PatternだけでGate Aを完了とはしません。既存認証Compatibility、Validation、Capability結果境界など、Driver Scenarioに必要な他のApplication上の意味と合わせて評価します。

## 設計上の境界

- `header-detail`をWeb画面テンプレートとは定義しない
- Header / Detail DataをRelationから導ける場合は重複記述しない
- cardinalityをPatternへコピーしない
- Semantic RelationとDatabase Foreign Keyを同一視しない
- Detailの並び順、ページング、Grid設定をCanonical Patternへ持ち込まない
- 一例だけでAggregate / Transaction一般モデルを確定しない
- Driverの実行順序をPattern自身へ押し込まない

## 非目標

- `sales.OrderLine` Semanticの再設計（`ISSUE-000011`で完了）
- `order_lines`物理table
- Database Foreign Key
- Database Contractの拡張
- Migration
- production transaction coordinator
- WebのHeader + Grid UI
- Detailの並び順・ページング仕様
- 汎用Form schema
- AIによるUI生成

## 完了条件

次をReference / Conformanceから再現できることを完了条件とします。

1. `PatternHeaderDetail`とRelation参照をCanonical model / Parser / Loader / Normalizationから扱える
2. `sales.order.lines`が持つsource / target / cardinalityをPatternへ複製せず、Application上のHeader / Detail意図を表現できる
3. Relationの存在と`many` cardinalityを共通Validationから検査できる
4. Pattern Diagnosticが既存Subject / Source Index経路を利用できる
5. UI layout、Datasource、Query、Persistence、FKをPatternから分離したまま維持できる
6. 5種類のPatternが`order_entry`で共存できる
7. Reference WorldをRepositoryだけから再生成できる
8. Application上の意味とTarget固有表現・Persistence実現の境界を説明できる
9. 後続DriverへHeader / Detailの意味をTarget非依存の情報として渡せる

## 関連

- GitHub Issue #67: v1 Application Pattern実証
- GitHub Issue #117: `OrderLine` Semantic Reference（完了）
- GitHub Issue #118: `header-detail`実証
- GitHub PR #119: Semantic Reference
- GitHub PR #122: 最小Canonical契約の実装候補
- `ISSUE-000005`: Application Pattern全体
- `ISSUE-000011`: Semantic substrate
- `ISSUE-000006`: Application Driver
- ADR-0017: Application Pattern ownership
- ADR-0018: Application Pattern authoring source
- RQ-001, RQ-003, RQ-005, RQ-011, RQ-012, RQ-014, RQ-017, RQ-021
