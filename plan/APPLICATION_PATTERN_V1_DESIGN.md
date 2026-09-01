# Application Pattern v1 設計・実装基線

## 目的

この文書は、`ISSUE-000005` / GitHub Issue #67で進めるApplication Patternについて、v1で守る設計境界、現在までに得られた実装Evidence、次に閉じるべきGapを一つの基線として整理します。

`browse`、`record`、`reference`、`command`、`header-detail`という5分類を、現時点でTEI全体の永久的な標準語彙として固定するものではありません。既存の`order_entry`を使い、Application上の意図をTarget固有UIから切り離して保持し、Driverや将来の複数Targetから同じ意味を利用できるかを段階的に実証します。

Patternの所有境界は[ADR-0017](adr/0017-application-pattern-ownership.md)、正本の配置とfile pathからのidentity分離は[ADR-0018](adr/0018-application-pattern-authoring-source.md)に従います。

## 現在地

Application Patternはすでに実装へ入っています。`browse`については実ReferenceとGolden Worldまで成立しましたが、通常のWorkspace validationへの統合と、実Referenceを使ったfile rename invarianceが残っています。

### PR #87 — 最小Canonical modelとParser

- `application.Pattern`と`Application.Patterns`
- ApplicationがPatternを子として所有する構造
- `browse`だけを有効化した`PatternKind`
- Pattern ID、Data、Query、任意のDatasource
- Data / Queryを必須にする`browse` Parser
- 同じfieldの重複を拒否するParser validation

現行の`Pattern`は小さく保っています。

```text
Pattern
- ID
- Kind
- Data
- Query
- Datasource（任意）
```

Semantic定義、Query本文、file path、Target固有PresentationはPatternへ複製しません。

### PR #88 — LoaderとNormalization

- `applications/*/patterns/*.tei`をWorkspace Loaderへ接続
- Patternを所属Applicationへ統合
- Canonical ID順にNormalization
- `application-pattern:<Canonical ID>`のsource mapping
- file列挙順とCanonicalなPattern順を分離

### PR #89 — Source Indexとidentity

- Pattern ID tokenへのSource Index range
- 既存のDeclaration Identifier Anchorを再利用
- 小さいfixtureで、file rename後もSubject identityを維持し、Artifact pathだけを追随させられることを確認

このrename EvidenceはSource Index境界を確認したものです。実`order_entry` Referenceで同じ不変条件が成立するかは、別途確認します。

### PR #95 — Workspace参照整合性Validation

Parser単体では判断できない、Workspace全体の参照整合性をPattern固有Validationとして実装しました。

- Pattern Canonical IDの重複
- 未知のSemantic Data
- 未知のQuery
- 別Applicationが所有するQuery
- 未知の明示Datasource
- Pattern側DatasourceとQuery側Datasourceの矛盾
- Pattern側Datasourceを省略した場合、Query側Datasourceだけで成立すること

Queryが返すDataを、現行モデルに十分な意味情報がない状態で推測して照合することや、Query / Datasource側のread / write検査をPattern側へ重複実装することは避けています。

### PR #99 — Diagnostic projection

Pattern固有Validationを外部から合成できる`ValidateApplicationPatterns`入口を追加し、DiagnosticのSubjectから既存Source Indexを通じてPattern定義へ戻れることを確認しました。

- Diagnosticが`application-pattern:<Canonical ID>` Subjectを保持する
- SubjectからPattern sourceへ解決できる
- 投影範囲がPattern行全体ではなくCanonical ID tokenそのものになる
- source lookupがDiagnostic messageの文字列解析に依存しない

### PR #101 — `order_entry.recent_orders`実Reference

最初の実Reference Patternを`order_entry`へ追加しました。

```text
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
```

既存のSemantic Data `sales.order`とnative SQL Query `order_entry.recent_orders`をidentityで参照しています。

Query側にはすでにDatasource `sales`があるため、Pattern側ではDatasourceを重複記述しません。Reference Conformanceでも、Pattern側Datasourceが空のままでQuery側の`sales`を利用できることを確認しています。

Pattern追加後のCanonical Worldを`reference/world.json`へ反映し、実Reference PatternのSubjectから実際のauthoring sourceへ戻れることも確認しました。

## 現在残っているGap

`browse`について、主な残差は次の二つです。

1. Pattern固有Validationを共通`validation.Validate`へ統合する
2. 実Referenceを使ってfile rename invarianceを確認する

共通Validation統合は[`ISSUE-000010`](issue/000010-application-pattern-validation-integration.md) / GitHub Issue #98、実Reference側の残差は[`ISSUE-000011`](issue/000011-order-entry-browse-reference.md) / GitHub Issue #100で扱います。

この二つが揃えば、`browse`を通常のWorkspace validationから実Reference、Golden World、Source Indexまで一続きに扱える状態になります。

その後は、次を段階的に進めます。

- `record`
- `reference`
- `command`
- `header-detail`
- 5種類をDriverへ渡せるApplication上の意味への統合

## v1で採るCanonical境界

### PatternはApplicationが所有する

v1では、PatternをWorld直下の共通Catalogから始めません。

```text
Application
├─ ID
├─ SemanticUses
├─ Databases
└─ Patterns
   ├─ Pattern
   └─ ...
```

PatternはSemanticそのものでも、Web画面テンプレートでもありません。Applicationが既存のSemantic、Query、Capability等をどのように利用するかという意図として扱います。

複数Applicationで本当に再利用できる意味が見つかった場合に、共有Catalog、Domain Component、Semantic Pattern等へ持ち上げる範囲をEvidenceから判断します。

### Patternは明示的なCanonical IDを持つ

Pattern identityをfile nameやfile pathから導出しません。

例:

```text
order_entry.recent_orders
order_entry.order_record
order_entry.customer_reference
order_entry.submit_order
order_entry.order_lines
```

Applicationへの所属もID文字列のprefixから推測せず、Canonical model上の所有関係として表します。

### Pattern KindはApplication層の小さい語彙として育てる

現行実装で有効なのは`browse`だけです。

v1では次を候補として順番に実証します。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

これらを`core.Kind`へ入れたり、十分なEvidenceなしに標準Catalogへ固定したりしません。各Kindは、具体的なReferenceで必要な意味が見えてから有効化します。

## Patternが保持する情報

Patternは既存資産の内容を複製せず、identityで参照します。

| 参照先 | 型の考え方 | 主な用途 |
| --- | --- | --- |
| Semantic Data | `core.CanonicalID` | browse / recordの対象 |
| Semantic Relation | `core.CanonicalID` | reference / header-detailで使う関係 |
| Query | `core.CanonicalID` | browse等の読み取り経路 |
| Capability | `core.CanonicalID` | commandの実行先 |
| Datasource | `core.LocalName` | Application内のlogical datasource alias |

柔軟性だけを優先した任意の`role -> string` Mapにはしません。一方、5種類それぞれに大きな専用構造を先に作ることもしません。

Web固有Widget、項目順、レスポンシブ指定などはPatternへ入れません。

## 正本とAuthoring

Patternの正本はApplication配下の`patterns/*.tei`です。

```text
applications/order-entry/
├─ application.tei
├─ patterns/
│  └─ recent-orders.tei
├─ queries/
└─ realizations/
```

v1では一ファイル一Patternから始めます。Editor内部状態を第二の正本にはせず、EditorはこのsourceのProjectionと編集支援を担います。

`browse`ではDatasourceを明示できますが、必須ではありません。

```text
pattern sample.orders browse {
    data sales.order
    query sample.orders
    datasource sales
}
```

`order_entry.recent_orders`ではQuery自身がDatasource `sales`を持つため、実ReferenceではPattern側へ重複記述していません。

**同じ意味を複数箇所へ手書きしないことを上位原則とします。**

## 次に閉じる二つのEvidence

### 1. 共通Validation契約

[`ISSUE-000010`](issue/000010-application-pattern-validation-integration.md)では、Pattern固有Validationを通常の`validation.Validate`へ統合します。

目的は新しい規則を増やすことではなく、CLI、Editor、Driver等の利用側がApplication Patternだけを特別扱いせず、同じWorkspace validation契約からDiagnosticを得られるようにすることです。

同じ原因によるDiagnosticを不必要に重複させないことも確認します。

### 2. 実Referenceでのrename invariance

[`ISSUE-000011`](issue/000011-order-entry-browse-reference.md)では、PR #101で追加した実Referenceを使い、file rename前後の意味を比較します。

確認対象は次です。

- Pattern Canonical ID
- normalized Canonical World
- Source IndexのSubject identity
- Source Artifact path
- 古いpathの残存有無

Canonical WorldとSubject identityは変わらず、authoring sourceのpathだけが追随することを確認します。

`ISSUE-000010`と`ISSUE-000011`の実装順は固定しません。

## 後続Patternの実証順

`browse`を閉じた後、既存資産だけで扱えるものを先に進めます。

1. `record` — `sales.order`をApplication上の一件として扱う
2. `reference` — `sales.order.customer` Relationから`customer.customer`を参照し、`master` Datasourceのread-only境界を維持する
3. `command` — `sales.order.submit` Capabilityへ接続する
4. `header-detail` — `OrderLine`と対応RelationをSemantic側へ追加した後に扱う

Pattern基盤の変更と新しい業務Semanticの追加をできるだけ分け、問題が起きたときにどの境界が原因かを追えるようにします。

## `command`とCapability結果

`command` Patternは既存の`sales.order.submit` Capabilityへ接続します。

結果境界は[ADR-0020](adr/0020-capability-business-outcome-boundary.md)と[`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md)に従います。

```text
Business Result
├─ 既存output value
└─ Named Outcome

Execution Error
└─ Capability呼び出しそのものを完了できなかった状態
```

`command` Patternを追加するという理由だけで、現行`sales.order.submit`の`output`をnamed Outcomeへ書き換えません。正常実行後に初めて判明する業務上の分岐が本当に必要な場合に圧力試験します。

## `header-detail`と`OrderLine`

`header-detail`は、新しい`sales.OrderLine`と対応RelationをSemantic側へ追加した後に検証します。

```text
Order 1 ── N OrderLine
```

Semantic Relation、Application上の`header-detail`、Database Foreign Keyは別の意味単位です。Relationがあるという理由だけでFKや画面構造を暗黙生成しません。

## Validationの責務

Parserは、構文だけで確定できることを扱います。

- Pattern IDの形式
- 認識できるKind
- fieldの重複
- Kindごとの必須field

Workspace Validationは、Workspace全体を見なければ分からない参照整合性を扱います。

- Data / Query / Datasourceの存在
- QueryのApplication所有関係
- Datasourceの矛盾
- 後続Kindが参照するRelation / Capability等の存在

不足や不整合は、可能な限りPattern自身のSubjectを保持したDiagnosticとして返します。

## Source IndexとDiagnostic

Pattern専用の別Source Indexは作りません。

`application-pattern:<Canonical ID>`という既存Subjectモデルを使い、authoring artifact、fingerprint、source rangeへ接続します。

PR #99までにPattern固有DiagnosticからPattern ID tokenへ戻れることを確認し、PR #101では実Reference PatternのSubjectから実sourceへ戻れることも確認しています。

## Gate Aへ渡す条件

#67だけでGate Aを完了とはしません。

Application Pattern側では、少なくとも次をRepository上のEvidenceから説明できる状態にします。

- 何を`browse`するか
- 一つの`record`として何を扱うか
- 何を別Datasourceから`reference`するか
- HeaderとDetailをどのようなApplication上の単位として扱うか
- どのValidationを実行するか
- どのCapabilityを`command`として呼ぶか
- どのDatasourceがread-onlyか
- Patternの正本とCanonical identityをどのように追跡するか

Authorizationは`ISSUE-000003`の既存認証Compatibilityから合流させます。

## 完了条件

`ISSUE-000005` / #67の完了条件は、`order_entry`の具体的な業務シナリオで5種類のPatternを一続きに表現し、Application上の意図とProjection / Target固有情報の境界を説明できることです。

さらに、Patternの正本、Canonical identity、Source Index、実Referenceでのfile rename invarianceをReference / Conformanceから再現でき、その同じApplication上の意味を後続Driverが利用できる必要があります。

現時点ではStage 1、Gate A、Milestone 1の完了は宣言しません。

## 関連文書

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0018-application-pattern-authoring-source.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000010-application-pattern-validation-integration.md`
- `plan/issue/000011-order-entry-browse-reference.md`
