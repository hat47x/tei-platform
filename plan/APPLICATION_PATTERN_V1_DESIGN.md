# Application Pattern v1 実装前設計

## 目的

この文書は、`ISSUE-000005` / GitHub Issue #67で進めるApplication Patternについて、最初の実装で必要となる境界を具体化するものです。

ここで固定するのは、`browse`、`record`、`reference`、`header-detail`、`command`という分類そのものではありません。まず既存の`order_entry`を使い、Application上の意図をTarget固有のUIから切り離して保持し、Driverや将来の複数Targetから同じ意味を参照できる最小構造を成立させます。その実証結果をもとに、Patternの分類や粒度は見直せる状態を保ちます。

Patternの初期所有境界は[ADR-0017](adr/0017-application-pattern-ownership.md)、正本の配置は[ADR-0018](adr/0018-application-pattern-authoring-source.md)に従います。

## 現行実装から確認できる前提

### ApplicationのCanonicalモデルはまだ薄い

現在の`internal/application.Application`が主に保持しているのは、ApplicationのCanonical ID、利用するSemantic Namespace、Database Requirementです。Application Patternに相当する定義はまだありません。

そのため、Patternを導入するときはApplicationモデルへ必要な意味だけを追加し、Semantic KernelやPresentation層まで一度に広げないことを優先します。

### Application配下には、すでに複数の正本ソースがある

Workspace Loaderは`application.tei`だけでなく、Application配下のQueryやRealizationも別のソースとして読み込み、一つのCanonical Worldへ統合しています。

```text
applications/order-entry/
├─ application.tei
├─ queries/
│  ├─ recent_orders.tei
│  └─ recent_orders.sql
└─ realizations/
   └─ ...
```

Patternもこの流儀に合わせ、Application配下の独立した定義ファイルとして扱います。

### Source Indexの基本構造はそのまま利用できる

Source Indexの`Subject`は、file pathを含まない`Kind + ID`で論理対象を識別します。`Kind`も閉じた列挙ではありません。

したがって、Pattern専用のSource Index基盤を新設する必要はありません。最初は次のSubjectで追跡します。

```text
Kind = application-pattern
ID   = <Pattern Canonical ID>
Role = definition
```

これにより、既存のfingerprint、source range、Diagnostic projection、ファイル改名に対する不変性を再利用できます。

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

これは、Patternを「Semanticそのもの」でも「Web画面テンプレート」でもなく、ApplicationがSemanticをどう利用するかという意図として扱うためです。

複数Applicationで本当に同じPatternを共有できるEvidenceが得られた場合にだけ、共有CatalogやPackageの必要性を検討します。

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

ファイルを移動・改名してもCanonical IDが同じなら、Canonical World上の意味は変わらないことをConformanceで確認します。

Applicationへの所属は、ID文字列だけから推測するのではなく、Canonicalモデル上でApplicationがPatternを所有する関係として表します。

### Pattern KindはApplication層の小さな語彙として始める

`browse`などを`core.Kind`へ追加しません。最初は`internal/application`の語彙として扱います。

- `browse`
- `record`
- `reference`
- `header-detail`
- `command`

第二Targetまで実証する前に、これらをTEI全体で永久に使う標準分類とはしません。

## Patternが保持する情報

Patternは、既存のSemantic Data、Relation、Query、Capability、Datasourceの内容を複製しません。既存のCanonical IDを参照し、「Applicationとしてどう使うか」だけを表します。

最初に必要な参照は次のとおりです。

| 参照先 | 型の考え方 | 主な用途 |
| --- | --- | --- |
| Semantic Data | `core.CanonicalID` | record / browseの対象 |
| Semantic Relation | `core.CanonicalID` | reference / header-detailで使う関係 |
| Query | `core.CanonicalID` | browseなどの読み取り経路 |
| Capability | `core.CanonicalID` | commandの実行先 |
| Datasource | `core.LocalName` | Application内のlogical datasource alias |

柔軟性だけを優先した任意の`role -> string` Mapにはしません。一方で、5種類のPatternごとに大きな専用構造を最初から作ることもしません。

v1では、共通のIDとKind、型を保った参照を明示し、Kindごとに必要な組合せをValidationで検査する中間案から始めます。

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

v1では、一つのファイルに一つのPattern定義を置くところから始めます。

この単位にすると、Source Indexとの対応、差分レビュー、Editorから特定Patternを開く経路、ファイル改名に対する不変性を単純に検証できます。

Editor内部状態を第二の正本にはしません。Editorは当面、この定義ソースのProjectionと編集支援を担います。

## 構文は実験用として扱う

最初のParserには構文が必要ですが、構文自体をProduct Valueとして固定しません。

概念上は、たとえば次のように書けます。

```text
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
    datasource sales
}
```

この例で重要なのはキーワードの綴りではなく、次の性質です。

- Pattern IDを明示できる
- Pattern Kindを明示できる
- Data、Query、Datasourceなどをidentityで参照できる
- SQL本文をPatternへコピーしない
- file pathをPattern identityにしない

Parserを実装するときは、既存DSLとの読みやすさや曖昧性を見て表層構文を調整できます。

## 最初の縦切りは`browse`一件にする

`order_entry`にはすでに次の資産があります。

- Semantic Data `sales.order`
- Query `order_entry.recent_orders`
- logical datasource `sales`
- `sales_core`へのread-write Binding
- native SQL `queries/recent_orders.sql`

そのため、新しい業務Semanticを追加しなくてもPattern基盤そのものを検証できます。

最初は次の一続きだけを成立させます。

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

この段階では`OrderLine`もWeb UIも不要です。

### `browse`で確認する意味

`order_entry.recent_orders` Patternは、受注一覧を扱うこと、対象が`sales.order`であること、読み取りに`order_entry.recent_orders` Queryを使うことを表します。

Query自身がDatasourceをすでに保持している場合、PatternでもDatasourceを重複して書くべきかは実装時に比較します。重複が不要なら、PatternからQueryを参照し、DatasourceはQueryから辿る方を優先します。

**同じ意味を複数箇所へ手書きしないことを上位原則とします。**

## 後続Patternの実証順

`browse`で基盤が成立した後、既存資産だけで扱えるものを先に追加します。

1. `record` — `sales.order`をApplication上の一件として扱う
2. `reference` — `sales.order.customer` Relationから`customer.customer`を参照し、`master` Datasourceのread-only境界を維持する
3. `command` — `sales.order.submit` Capabilityへ接続する
4. `header-detail` — `OrderLine`と対応RelationをSemantic側へ追加した後に扱う

この順序にすることで、Pattern基盤の変更と新しい業務Semanticの追加を同時に行わず、問題が起きたときに原因を切り分けやすくします。

## Validation

Parserでは、少なくとも次を確認します。

- Pattern IDが有効なCanonical IDである
- Kindがv1で認識している値である
- 同じfieldが矛盾する形で重複していない
- 必須値が欠けていない

Workspace全体を見なければ分からないことはParserへ押し込みません。Workspace / Validation層では、次を確認します。

- Pattern IDがApplication内で重複していない
- 参照するSemantic Data / Relation / Capabilityが存在する
- 参照するQueryが同じApplicationに属する
- Datasource aliasがApplicationに存在する
- `reference`がread-only Datasourceをwrite前提で扱わない
- `command`がCapabilityを参照している

Pattern Kindごとの不足は、元のPattern定義へ戻れるDiagnosticとして返します。

## Source IndexとDiagnostic

Pattern定義は`application-pattern:<Canonical ID>`としてSource Indexへ登録します。

Conformanceでは、少なくとも次を確認します。

- Pattern定義へSource Indexから戻れる
- ファイルを改名してもnormalized Worldが変わらない
- 改名後はSource IndexのArtifact Pathだけが変わる
- fingerprintとrangeが新しいソースと一致する
- 古いpathがCanonical Worldへ残らない

Canonicalモデルへline / columnなどのsource locatorは埋め込みません。

## Normalizationと再現性

`Application.Patterns`の順序は、ファイル列挙順やOSのdirectory orderに依存させません。Canonical IDで安定して正規化します。

Patternの追加によって`reference/world.json`が変わること自体は意図した変更です。ただし、その差分にfile pathやParser都合が混ざってはいけません。

## #67の実装単位

#67は次の順で小さく進めます。

1. `application.Pattern`の最小Canonicalモデル
2. Normalization / Validation
3. Pattern parser
4. `patterns/*.tei` Loader
5. Source Index / Diagnostic接続
6. `browse`一件のReference / Conformance
7. ファイル改名に対する不変性
8. `record` / `reference` / `command`
9. `OrderLine`追加後に`header-detail`
10. Gate Aへ渡すEvidenceの監査

途中でPattern分類や所有境界が不自然だと分かった場合は、後続を積み上げず、ADR-0017とこの文書を見直します。

## まだ固定しないこと

次は最初の実装Evidenceが出るまで固定しません。

- Go structの細かなfield配置
- Pattern DSLの最終構文
- QueryがDatasourceを持つ場合のPattern側Datasource記述の要否
- Validation / AuthorizationをPatternが直接参照するか
- Pattern Catalogの共有方式
- Patternの継承・合成
- Pattern versioning
- UI field grouping
- Editorでの編集方式

## 完了条件

この実装前設計は、#67の最初の実装PRで何を作り、何を作らないかをRepositoryだけから判断できれば完了とします。

具体的には、次を説明できることが条件です。

- Patternがどの層に属するか
- Pattern identityをどう保つか
- 正本をどこへ置くか
- 既存Semantic / Query / Capabilityをどう参照するか
- Source Indexへどう接続するか
- なぜ`browse`から始めるか
- 何をConformanceとして残すか
- どの判断をまだ保留しているか

## 関連文書

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0018-application-pattern-authoring-source.md`
- `plan/issue/000005-application-pattern-reference.md`
