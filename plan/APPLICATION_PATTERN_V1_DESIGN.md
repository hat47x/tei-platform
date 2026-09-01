# Application Pattern v1 設計・実装基線

## 目的

この文書は、`ISSUE-000005` / GitHub Issue #67で進めるApplication Patternについて、v1で守る設計境界、Repositoryで確認できたEvidence、次に閉じるべきGapを一つの基線として整理するものです。

Application Patternは、画面テンプレートの分類を先に作るための仕組みではありません。ApplicationがSemantic、Query、Capability、Datasourceなどをどのような意図で利用するかを、Target固有UIから切り離して表現するための層です。

v1では`order_entry`を実Referenceとして使い、候補となるPattern Kindを一つずつ実装・検証します。分類やfieldを先に一般化せず、実際の業務上の意味を既存資産から抽出し、必要だったものだけをCanonical contractへ残します。

Patternの所有境界は[ADR-0017](adr/0017-application-pattern-ownership.md)、正本の配置とfile pathからのidentity分離は[ADR-0018](adr/0018-application-pattern-authoring-source.md)に従います。

## 現在地

2026年9月時点で、次の3種類は実Referenceまで一続きに成立しています。

- `browse`
- `record`
- `reference`

次に実証するのは`command`です。その後、新しいSemantic Data `OrderLine`と対応Relationを追加して`header-detail`を検証します。

Stage 1全体、Gate A、Milestone 1の完了はまだ宣言しません。

## 現在有効なCanonical model

`application.Pattern`は、現時点で概ね次の情報を持ちます。

```text
Pattern
- ID
- Kind
- Data
- Relation
- Query
- Datasource
```

すべてのfieldをすべてのKindが使うわけではありません。Kindごとに必要な意味だけを利用します。

現時点で有効なKindは次です。

```text
browse
record
reference
```

`command`と`header-detail`は、実Referenceを通じて必要な意味を確認してから有効化します。

Semantic定義、Query本文、Database Contract本文、file path、Target固有PresentationはPatternへ複製しません。

## v1で守るCanonical境界

### PatternはApplicationが所有する

v1ではPatternをWorld直下の共有Catalogから始めません。ApplicationがPatternを子として所有します。

```text
Application
├─ ID
├─ SemanticUses
├─ Databases
└─ Patterns
   ├─ Pattern
   └─ ...
```

PatternはSemanticそのものでも、Web画面でもありません。Applicationが既存の意味資産をどのように利用するかを表します。

複数Applicationで本当に同じPatternを再利用するEvidenceが得られた場合にだけ、共有CatalogやPackageへ持ち上げる範囲を検討します。

### Pattern identityはfile pathから分離する

Patternは明示的なCanonical IDを持ちます。

```text
order_entry.recent_orders
order_entry.order_record
order_entry.customer_reference
order_entry.submit_order
order_entry.order_lines
```

Applicationへの所属もID文字列のprefixから推測せず、Canonical model上の所有関係として保持します。

Source Indexでは`application-pattern:<Canonical ID>`をSubjectとして扱い、実sourceのfile pathやrangeへ戻せます。file pathは来歴であってidentityではありません。

### 同じ意味を複数箇所へ手書きしない

Application Patternで最も重要な原則の一つです。

既存のSemantic Data、Relation、Query、Capability、Datasourceがすでに持つ意味は、Patternへコピーしません。PatternではCanonical IDやApplication内aliasで参照し、Application固有の利用意図だけを追加します。

この原則は、`browse`、`record`、`reference`の3種類すべてで実Referenceから確認できました。

### Kindごとの形を無理に共通化しない

`browse`でDataとQueryが必要だったからといって、`record`や`reference`へ同じfieldを持ち込みません。

v1では、共通の巨大なPattern schemaや任意の`role -> string` Mapを先に作りません。小さな型付きfieldを共有しつつ、各Kindが受理する組み合わせをParserとValidationで明確にします。

## 実証済みPattern

## `browse`

### 実Reference

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
```

### 現在の契約

- `data`: 必須
- `query`: 必須
- `datasource`: 任意
- `relation`: 受理しない

ReferenceではPattern側のDatasourceを省略しています。Query `order_entry.recent_orders`が`datasource sales`を正本として持つためです。

### 実証したこと

- PatternからSemantic Dataと既存QueryをCanonical IDで再利用できる
- native SQL本文をPatternへ複製しなくてよい
- Queryがすでに持つDatasourceを理由なく二重記述しなくてよい
- Queryが対象Applicationに属することをWorkspace Validationで確認できる
- Pattern側でDatasourceを明示した場合は、Query側Datasourceとの不一致を検出できる
- Pattern SubjectからSource Indexを通じてauthoring sourceへ戻れる
- Reference WorldをRepository自身から再生成できる
- 実Referenceのfile renameでもCanonical Worldを維持し、Source Artifact pathだけを追随させられる

`browse`の基盤から実Referenceまでの主なEvidenceはPR #87〜#103です。

## `record`

Issue #104、PR #105 / #106で実証しました。

### 実Reference

```tei
pattern order_entry.order_record record {
    data sales.order
}
```

### 現在の契約

- `data`: 必須
- `query`: 受理しない
- `datasource`: 受理しない
- `relation`: 受理しない

### 実証したこと

`record`は「ApplicationがどのSemantic Dataを一件として扱うか」という意味だけで成立しました。

既存Database Contractに`find_by_id`があることや、将来Driverが一件取得を行うことは、Patternへ取得経路を固定する理由にはなりませんでした。

同様に、`sales.order`とlogical datasource `sales`の関係もPatternへ重複記述していません。

この結果から、`record`をWebの詳細画面やForm定義として扱わず、取得方法や画面遷移と分離したまま保持できることを確認しました。

## `reference`

Issue #107、PR #108 / #109で実証しました。

### 実Reference

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

### 現在の契約

- `relation`: 必須
- `datasource`: 必須
- `data`: 受理しない
- `query`: 受理しない

### Relationに意味を寄せる

Semantic Relation `sales.order.customer`は、すでに次を保持しています。

```text
source:      sales.order
target:      customer.customer
cardinality: one
```

したがってPatternへ`data sales.order`や`target customer.customer`を重複記述しません。

### DatasourceはApplication固有の意味として保持する

同じSemantic Relationでも、Applicationによってどのlogical datasourceから解決するかは異なり得ます。

`order_entry`では`master`を選びます。

```text
master
- target: master_data
- access: read
- requires: master_data.customer ^2
```

Datasourceのaccess mode自体はPatternへ複製しません。ApplicationのDatabaseRequirementが正本です。

`reference`は読み取り意図なので、`write`専用Datasourceは`APPLICATION_PATTERN_DATASOURCE_NOT_READABLE`として拒否します。`read-write`は読み取り可能なので受理します。

### Database Contractとの接続

Reference Conformanceでは、次の接続まで追跡しています。

```text
reference Pattern
  relation sales.order.customer
      ↓ target
  customer.customer

reference Pattern
  datasource master
      ↓ requires
  master_data.customer ^2
      ↓ contract
  read customer.customer
```

既存共通Validationは、Relation targetの存在、ContractのSemantic参照、Applicationが要求するContractの存在、Contract提供Databaseとの一致、compatible majorをそれぞれ検査しています。

一方、「このreference Patternが選んだDatasourceのRequired Contractが、このRelation targetをread対象として公開している」という横断条件は、現時点ではReference Conformanceで明示的に観測しています。

この一例だけを理由にPattern専用の重複Contract体系は作りません。DriverやContract解決を実装する過程で同じ条件を共通利用するEvidenceが増えた場合に、再利用可能なValidationへ昇格します。

## Patternが参照する既存資産

v1で想定する主な参照先は次です。

| 参照先 | 型 | 現在の用途 |
| --- | --- | --- |
| Semantic Data | `core.CanonicalID` | `browse` / `record` |
| Semantic Relation | `core.CanonicalID` | `reference`、将来の`header-detail`候補 |
| Query | `core.CanonicalID` | `browse` |
| Datasource | `core.LocalName` | Application固有のlogical datasource |
| Capability | `core.CanonicalID` | 次の`command`で検証 |

Capability参照を`Pattern`へどのfield名・構造で追加するかは、`command`実証で決めます。

## 正本の配置

ADR-0018に従い、Patternの正本はApplication配下の`patterns/*.tei`へ置きます。

現在のReferenceは次の構成です。

```text
applications/order-entry/
├─ application.tei
├─ patterns/
│  ├─ recent-orders.tei
│  ├─ order-record.tei
│  └─ customer-reference.tei
├─ queries/
└─ realizations/
```

v1では一ファイル一Patternから始めています。

Editor内部状態を第二の正本にはしません。Editorはこの定義ソースのProjectionと編集支援を担います。

## ParserとWorkspace Validationの責務

### Parserで確認すること

Parserでは、Pattern単体から判断できる構文契約を確認します。

- Pattern IDが有効なCanonical IDである
- Kindが現時点で有効な値である
- 同じfieldが重複していない
- Kindに必須のfieldが存在する
- そのKindでは意味を持たないfieldを受理しない

### Workspace Validationで確認すること

Workspace全体を見なければ分からない参照整合性はValidation層で扱います。

現在は主に次を検査します。

- Pattern Canonical IDの重複
- Semantic Dataの存在
- Queryの存在とApplication所有関係
- Datasource aliasの存在
- PatternとQueryが両方Datasourceを明示した場合の一致
- Semantic Relationの存在
- `reference`が読み取り可能なDatasourceを選んでいること

不整合は`application-pattern:<Canonical ID>` Subject付きDiagnosticとして返し、Source Indexを通じてauthoring sourceへ戻せます。

## Source Indexと再現性

Patternは次の一続きの経路で追跡できます。

```text
patterns/*.tei
    ↓ parse
application.Pattern
    ↓ load
Application.Patterns
    ↓ normalize
reference/world.json
    ↓ source index
application-pattern:<Canonical ID>
    ↓
Source Artifact / Diagnostic
```

Canonical modelへline / columnやfile pathを埋め込みません。

`Application.Patterns`はCanonical ID順に正規化し、directory列挙順をCanonicalな意味へ持ち込みません。

`reference/world.json`は手編集せず、Repository自身の`tei world`から再生成します。

## 次の実証 — `command`

次は既存Semantic Capability `sales.order.submit`をApplication上の`command`として利用する最小契約を実証します。

既存Capabilityは次です。

```text
capability sales.order.submit {
    input order sales.order
    output sales.order
}
```

最初に確認する問いは次です。

- `command` Patternが保持すべき最小参照はCapabilityだけで足りるか
- 入力Data `sales.order`をPatternへ重複記述する必要があるか
- DatasourceをPatternへ持たせる必要があるか、それともCapability Binding / Driver側の責務か
- ValidationやAuthorizationをPatternが直接参照すべきか
- Capabilityの既存`output`をそのままBusiness Resultとして扱えるか

現時点では、次のような最小形を第一候補として考えます。

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

ただし、field名や構文を先に確定しません。既存Capability model、Binding、Driverへの受け渡しを確認したうえでIssue化します。

## `command`とCapability Outcome

Capability結果の共通境界は[ADR-0020](adr/0020-capability-business-outcome-boundary.md)と[`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md)に従います。

`command` Patternの導入だけを理由に、現行`sales.order.submit`の`output`をnamed Outcomeへ書き換えません。

Commandを正常に実行して初めて分かる業務上の分岐が実Referenceで必要になった場合に、Capability Outcomeの圧力試験として扱います。

Business ResultとExecution Errorの境界をPattern固有に作り直しません。

## `header-detail`と`OrderLine`

`header-detail`は、`command`の後に新しい`sales.OrderLine`と対応RelationをSemantic側へ追加して検証します。

概念上は次の関係です。

```text
Order 1 ── N OrderLine
```

Semantic Relation、Application上の`header-detail`、Database Foreign Keyは別の意味単位です。

1:N Relationがあるという理由だけでFKや画面構造を暗黙生成しません。Pattern基盤の変更と新しい業務Semanticの追加を分け、問題が生じたときにどの境界が原因だったかを追跡できるようにします。

## #67の実装単位と現在の状態

| 実装単位 | 状態 | Evidence / 残差 |
| --- | --- | --- |
| Pattern最小Canonical model | 完了 | PR #87、後続PRでData / Relation等を拡張 |
| `patterns/*.tei` Loader / Normalization | 完了 | PR #88 |
| Source Index / Pattern ID range | 完了 | PR #88〜#89 |
| Workspace Validation / Diagnostic統合 | 完了 | PR #95 / #99 / #103、後続Kindでも拡張 |
| `browse`実Reference / Golden | 完了 | PR #101 |
| 実Referenceのfile rename invariance | `browse`で完了 | PR #102。共通identity/path境界のEvidenceとして利用 |
| `record`最小契約 / 実Reference | 完了 | Issue #104、PR #105 / #106 |
| `reference`最小契約 / 実Reference | 完了 | Issue #107、PR #108 / #109 |
| `command` | 次の優先単位 | `sales.order.submit`から最小契約を抽出する |
| `header-detail` | 未着手 | `OrderLine`とRelation追加後に検証 |
| Gate A Evidence監査 | 未完了 | #68の既存認証Compatibilityと合流して評価 |

## まだ固定しないこと

実装Evidenceが増えるまで、次は固定しません。

- 5 Patternの最終的な標準分類
- `command` / `header-detail`の最終DSL
- Validation / AuthorizationをPatternが直接参照するか
- reference Relation targetとRequired Contract Readsの横断検査をCoreへ昇格するか
- Pattern Catalogの共有方式
- Patternの継承・合成
- Pattern versioning
- UI field grouping
- Editorでの最終編集方式

`browse`については、QueryがDatasourceを持つ場合にPattern側で同じDatasourceを重複記述しなくてよいことを実Referenceから確認済みです。

## Gate Aへ渡す条件

#67だけでGate Aを完了とはしません。

Application Pattern側では、少なくとも次をRepository上のEvidenceから説明できる状態にします。

- 何を`browse`するか
- 一つの`record`として何を扱うか
- どのRelationを、どのlogical datasourceから`reference`するか
- HeaderとDetailをどのようなApplication上の単位として扱うか
- どのValidationを実行するか
- どのCapabilityを`command`として呼ぶか
- どのDatasourceがread-only / read-writeか
- Patternの正本とCanonical identityをどのように追跡するか

Authorizationは`ISSUE-000003` / GitHub Issue #68の既存認証Compatibilityから合流させます。

## 完了条件

#67の完了条件は、`order_entry`の業務シナリオで候補となるPatternを一続きに表現でき、Application上の意図とProjection / Target固有情報の境界を説明できることです。

さらに、Patternの正本、Canonical identity、Source Index、Reference Worldの再現性をRepository内のReference / Conformanceから確認でき、その同じApplication上の意味を後続Driverが利用できる必要があります。

現在は`browse`、`record`、`reference`まで実Referenceを閉じました。次は`command`です。Stage 1やMilestone 1の完了はまだ宣言しません。

## 関連文書

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0018-application-pattern-authoring-source.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`

## 関連Issue / PR

- GitHub Issue #67 — v1 Application Pattern実証
- GitHub Issue #104 — `record`実証、完了
- GitHub Issue #107 — `reference`実証、完了
- PR #87〜#103 — `browse`基盤から実Referenceまで
- PR #105 / #106 — `record`
- PR #108 / #109 — `reference`
