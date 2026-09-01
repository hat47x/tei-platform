# Application Pattern v1 設計・実装基線

## 目的

この文書は、`ISSUE-000005` / GitHub Issue #67で進めるApplication Patternについて、v1で守る設計境界と、現在の実装Evidence、次に閉じるべきGapを一つの基線として整理するものです。

ここで固定するのは、`browse`、`record`、`reference`、`header-detail`、`command`という分類そのものではありません。既存の`order_entry`を使い、Application上の意図をTarget固有UIから切り離して保持し、Driverや将来の複数Targetから同じ意味を参照できる構造を段階的に実証します。

Patternの初期所有境界は[ADR-0017](adr/0017-application-pattern-ownership.md)、正本の配置とfile pathからのidentity分離は[ADR-0018](adr/0018-application-pattern-authoring-source.md)に従います。

## 現在地

Application Patternは未着手ではありません。一方、5種類のPatternを`order_entry`で実証し終えた段階でもありません。

PR #87〜#89によって、最初の`browse`を受け止める基盤の前半がRepositoryへ入りました。

### 実装済み

#### PR #87 — 最小Canonical modelと`browse` Parser

- `application.Pattern`を追加
- `Application.Patterns`を追加
- PatternがApplicationの子として保持される構造を追加
- Pattern Kindは`browse`だけを有効化
- Pattern ID、Data、Query、任意のDatasourceを型付きidentityとして保持
- `browse`でData / Queryを必須にするParserを追加
- 同じfieldの重複をParserで拒否

現行の`Pattern`は概ね次の情報だけを持ちます。

```text
Pattern
- ID
- Kind
- Data
- Query
- Datasource
```

Semantic定義、Query本文、file path、Target固有PresentationはPatternへ複製していません。

#### PR #88 — `patterns/*.tei` LoaderとNormalization

- `applications/*/patterns/*.tei`をWorkspace Loaderへ接続
- 読み込んだPatternを所属Applicationの`Application.Patterns`へ統合
- PatternをCanonical ID順に正規化
- `application-pattern:<Canonical ID>`のsource mappingを登録
- ファイル名の列挙順とCanonicalなPattern順を分離するConformanceを追加

この時点で、Application配下の独立したPattern定義をCanonical modelへ読み込む経路が成立しました。

#### PR #89 — Source IndexのPattern識別子range

- `application-pattern:<Canonical ID>`をPattern宣言中のCanonical ID tokenへ接続
- Pattern専用の新しいanchor機構を作らず、既存のDeclaration Identifier Anchorを再利用
- Source IndexがPattern ID全体のrangeを返すことを確認
- Patternファイル名を変更してもSubject identityを変えず、Source Artifact pathだけを更新できることを確認

Source Indexについては、Pattern identityとfile pathの分離を実コードで確認できる段階へ進みました。

### まだ実装済みとは言わないこと

現在の`reference/workspace/applications/order-entry`には、まだ`patterns/`がありません。

そのため、次は未完了です。

- `order_entry.recent_orders`の実Reference Pattern
- Patternが参照するData / Query / DatasourceのWorkspace全体での整合性Validation
- Pattern固有Diagnosticのend-to-end Conformance
- Patternを含む`reference/world.json`のgolden更新と再現性確認
- 実ReferenceでのCanonical World rename invariance
- `record` / `reference` / `command` / `header-detail`
- 5 PatternをDriverへ渡せる一続きのApplication上の意味

PR #89で確認したrenameは、Source IndexのSubject identityとArtifact pathの挙動です。これをもって、Patternを含むCanonical World全体のrename invarianceまで実証済みとはみなしません。

## v1で採るCanonical境界

### PatternはApplicationが所有する

v1では、PatternをWorld直下の共通Catalogから始めません。ApplicationがPatternを子として所有します。

```text
Application
├─ ID
├─ SemanticUses
├─ Databases
└─ Patterns
   ├─ Pattern
   └─ ...
```

Patternを「Semanticそのもの」でも「Web画面テンプレート」でもなく、ApplicationがSemanticをどのように利用するかという意図として扱うためです。

複数Applicationで本当に同じPatternを再利用できるEvidenceが得られた場合にだけ、共有CatalogやPackageへ持ち上げる範囲を検討します。

### Patternは明示的なCanonical IDを持つ

Patternのidentityをファイル名やfile pathから導出しません。

例:

```text
order_entry.recent_orders
order_entry.order_record
order_entry.customer_reference
order_entry.order_lines
order_entry.submit_order
```

Applicationへの所属も、ID文字列のprefixから推測せず、Canonical model上でApplicationがPatternを所有する関係として表します。

### Pattern KindはApplication層の小さな語彙として育てる

現行実装で有効なのは`browse`だけです。

将来のv1 Referenceでは次を候補として検証します。

- `browse`
- `record`
- `reference`
- `header-detail`
- `command`

これらを`core.Kind`へ入れたり、TEI全体の永久的な標準分類として先に固定したりしません。各Kindは、具体的なReferenceで必要な意味を確認してから一つずつ有効化します。

## Patternが保持する情報

Patternは、既存のSemantic Data、Relation、Query、Capability、Datasourceの内容を複製しません。既存のidentityを参照し、「Applicationとしてどのように使うか」だけを表します。

v1で想定する主な参照先は次です。

| 参照先 | 型の考え方 | 主な用途 |
| --- | --- | --- |
| Semantic Data | `core.CanonicalID` | record / browseの対象 |
| Semantic Relation | `core.CanonicalID` | reference / header-detailで使う関係 |
| Query | `core.CanonicalID` | browseなどの読み取り経路 |
| Capability | `core.CanonicalID` | commandの実行先 |
| Datasource | `core.LocalName` | Application内のlogical datasource alias |

柔軟性だけを優先した任意の`role -> string` Mapにはしません。一方、5種類のPatternごとに大きな専用構造を先に作ることもしません。

現行の`browse`は、ID、Kind、Data、Query、任意のDatasourceという小さな構造で始めています。後続Kindに必要なfieldは、それぞれのReferenceを実装するときに追加します。

Web固有のWidget、項目順、レスポンシブ指定などはPatternへ入れません。

## 正本の配置

ADR-0018に従い、Patternの正本はApplication配下の`patterns/*.tei`へ置きます。

```text
applications/order-entry/
├─ application.tei
├─ patterns/
│  ├─ recent_orders.tei
│  ├─ order_record.tei
│  ├─ customer_reference.tei
│  ├─ order_lines.tei
│  └─ submit_order.tei
├─ queries/
└─ realizations/
```

v1では一ファイル一Patternから始めます。

Loaderはこの配置をすでに扱えますが、実際の`order-entry/patterns/`はまだReferenceへ追加していません。

Editor内部状態を第二の正本にはしません。Editorは当面、この定義ソースのProjectionと編集支援を担います。

## Authoring構文

`browse`のParserは、現在次の形を扱えます。

```text
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
    datasource sales
}
```

重要なのは表層の綴りではなく、次の性質です。

- Pattern IDを明示できる
- Pattern Kindを明示できる
- Data、Query、Datasourceを型付きidentityで参照できる
- SQL本文をPatternへコピーしない
- file pathをPattern identityにしない

現行構文は最初の実装Evidenceとして採用していますが、後続Patternの実証から不自然さが見つかれば見直せます。

## 次の最小縦切り

次に優先するのは、新しいPattern Kindの追加ではありません。

`browse`基盤を実Referenceとして閉じるため、次の順で進めます。

### 1. Workspace全体の参照整合性Validation

Parser単体では分からない次を検査します。

- Pattern IDがApplication内で重複していない
- `Data`が存在する
- `Query`が存在し、対象Applicationから利用できる
- 明示した`Datasource`がApplicationに存在する
- Pattern Kindごとの必要参照が揃っている

不整合は、可能な限り元のPattern定義へ戻れるDiagnosticとして返します。

### 2. `order_entry.recent_orders`を実Referenceへ追加

既存の次の資産をそのまま利用します。

- Semantic Data `sales.order`
- Query `order_entry.recent_orders`
- logical datasource `sales`
- native SQL `queries/recent_orders.sql`

SQL本文やQuery定義をPattern側へコピーしません。

Query自身がDatasourceをすでに保持している場合、PatternでもDatasourceを明示する必要があるかはReferenceの二重管理を見て判断します。**同じ意味を複数箇所へ手書きしないことを上位原則とします。**

### 3. Reference Worldとend-to-end Conformance

PatternをReferenceへ追加したうえで、次を一続きに確認します。

```text
patterns/recent_orders.tei
        ↓ parse
application.Pattern
        ↓ load
Application.Patterns
        ↓ normalize
reference/world.json
        ↓ index
application-pattern:order_entry.recent_orders
        ↓
Source Index / Diagnostic
```

ここで初めて、実Referenceを含むCanonical Worldの再現性とrename invarianceまで閉じます。

## 後続Patternの実証順

`browse`がReference / Conformanceまで成立した後、既存資産だけで扱えるものを先に追加します。

1. `record` — `sales.order`をApplication上の一件として扱う
2. `reference` — `sales.order.customer` Relationから`customer.customer`を参照し、`master` Datasourceのread-only境界を維持する
3. `command` — `sales.order.submit` Capabilityへ接続する
4. `header-detail` — `OrderLine`と対応RelationをSemantic側へ追加した後に扱う

Pattern Kindは、この順番に合わせて必要なCanonical fieldとValidationを追加します。5種類を一度に有効化しません。

## `command`とCapability Outcome

`command` Patternを追加するときは、既存`sales.order.submit` Capabilityへ接続します。

Capability結果の共通境界は[ADR-0020](adr/0020-capability-business-outcome-boundary.md)と[`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md)に従います。

ただし、`command` Patternの導入だけを理由に、現行`sales.order.submit`の`output`をnamed Outcomeへ書き換えません。Commandを正常に実行して初めて分かる業務上の分岐が実例として必要な場合に、Capability Outcomeの圧力試験として扱います。

## `header-detail`と`OrderLine`

`header-detail`は、新しい`sales.OrderLine`と対応RelationをSemantic側へ追加した後に検証します。

概念上は次の関係です。

```text
Order 1 ── N OrderLine
```

Semantic Relation、Application上の`header-detail`、Database Foreign Keyは別の意味単位です。Relationがあるという理由だけでFKや画面構造を暗黙生成しません。

Pattern基盤の変更と新しい業務Semanticの追加を分け、問題が起きたときに原因となった境界を追えるようにします。

## Validationの責務

Parserでは、構文だけで確定できることを確認します。

- Pattern IDが有効なCanonical IDである
- Kindが現時点で認識している値である
- 同じfieldが重複していない
- Kindに必須のfieldが欠けていない

Workspace全体を見なければ分からない参照整合性は、Parserへ押し込みません。

後続Kindでは、たとえば次をWorkspace / Validation層で確認します。

- `reference`が参照するRelationが存在する
- Datasourceのread-only境界を維持する
- `command`が存在するCapabilityへ接続する
- `header-detail`がSemantic RelationをPersistence FKと混同しない

## Source IndexとDiagnostic

`application-pattern:<Canonical ID>`のSource Index登録とPattern ID tokenへのrange接続は実装済みです。

今後のConformanceでは、実Referenceを使って次まで閉じます。

- Pattern定義へSource Indexから戻れる
- Pattern固有の参照不整合をDiagnosticから元定義へ戻せる
- ファイルを改名してもnormalized Worldが変わらない
- 改名後はSource IndexのArtifact Pathだけが変わる
- fingerprintとrangeが新しいsourceと一致する
- 古いpathがCanonical Worldへ残らない

Canonical modelへline / column等のsource locatorは埋め込みません。

## Normalizationと再現性

`Application.Patterns`をCanonical IDで安定して正規化する処理は実装済みです。

ファイル列挙順やOSのdirectory orderをCanonicalなPattern順にはしません。

Patternの追加によって`reference/world.json`が変わること自体は意図した変更です。ただし、その差分にfile path、file name、Parser都合を混ぜません。

## #67の実装単位と現在の状態

| 実装単位 | 状態 | Evidence / 残差 |
| --- | --- | --- |
| 1. `application.Pattern`の最小Canonical model | 実装済み | PR #87 |
| 2. `browse` Parser | 実装済み | PR #87 |
| 3. `patterns/*.tei` Loader | 実装済み | PR #88 |
| 4. PatternのCanonical ID順Normalization | 実装済み | PR #88 |
| 5. Source Index接続 | 実装済み | PR #88〜#89。Pattern ID rangeまで確認済み |
| 6. Workspace参照整合性Validation / Pattern Diagnostic | 未完了 | 次の優先単位 |
| 7. `order_entry.recent_orders` browse Reference / golden World | 未完了 | `order-entry/patterns/`はまだ存在しない |
| 8. 実Referenceでの完全なrename invariance | 未完了 | Source Index単体のrename挙動はPR #89で確認済み |
| 9. `record` / `reference` / `command` | 未着手 | `browse`を閉じた後に順次追加 |
| 10. `OrderLine`追加後の`header-detail` | 未着手 | 新しいSemanticを後段で追加 |
| 11. Gate Aへ渡すEvidence監査 | 未着手 | #68の既存認証Compatibilityと合流して評価 |

途中でPattern分類や所有境界が不自然だと分かった場合は、後続を積み上げず、ADR-0017 / 0018とこの基線を見直します。

## まだ固定しないこと

実装Evidenceが増えるまで、次は固定しません。

- 5 Patternの最終的な標準分類
- 後続KindのGo structの細かなfield配置
- 後続KindのDSL最終構文
- QueryがDatasourceを持つ場合のPattern側Datasource記述の要否
- Validation / AuthorizationをPatternが直接参照するか
- Pattern Catalogの共有方式
- Patternの継承・合成
- Pattern versioning
- UI field grouping
- Editorでの編集方式

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

#67の完了条件は、`order_entry`の業務シナリオで5 Patternを一続きに表現でき、Application上の意図とProjection / Target固有情報の境界を説明できることです。

さらに、Patternの正本、Canonical identity、Source Index、実Referenceでのfile rename invarianceをReference / Conformanceから再現でき、その同じApplication上の意味を後続Driverが利用できる必要があります。

現在はその前半である`browse`基盤のmodel / Parser / Loader / Source Indexまでが実装済みです。Stage 1やMilestone 1の完了はまだ宣言しません。

## 関連文書

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0018-application-pattern-authoring-source.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
