# ADR-0018: Application Patternの正本をApplication配下の独立した定義ファイルに置く

- 状態: 採用
- 日付: 2026-09-01

## 背景

ADR-0017では、Application PatternをSemantic KernelのprimitiveやTarget固有の画面テンプレートとして早い段階から固定せず、Applicationが所有するCanonicalな意図として検証する方針を定めました。

次に決める必要があるのは、そのPatternをどこから読み込み、何を正本として扱うかです。

現在のApplicationは`application.tei`だけで完結していません。Application配下の`queries/`や`realizations/`に置かれた定義をWorkspace Loaderが読み込み、一つのCanonical Worldへ統合します。

また、Source-to-Canonical Traceabilityでは、Canonical identityをfile pathから分離し、正本となるソースの場所はSource Indexで追跡する境界をすでに持っています。

Application Patternを導入するときも、この二つの原則を崩さないことを優先します。

## 判断

v1では、Application Patternの正本を**各Applicationディレクトリ配下の`patterns/*.tei`**に置きます。

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

最初は、一つのPattern定義ファイルに一つのPatternを置きます。

### Pattern identityはfile pathから独立させる

Patternは明示的なCanonical IDを持ちます。

```text
order_entry.recent_orders
```

`patterns/recent_orders.tei`というpathは定義ファイルの場所を示す情報であり、Pattern identityではありません。

ファイル名や配置を変えてもCanonical IDが同じなら、Canonical World上の意味は変わらないことをConformanceで確認します。

### Applicationへの所属はCanonicalモデルで表す

Pattern定義ファイルはApplicationディレクトリ配下に置きますが、所属関係をfile pathだけには依存させません。

Workspace LoaderはPatternを格納先のApplicationへ統合し、Canonicalモデル上ではApplicationがPatternを所有する関係として保持します。

Pattern IDとApplication IDのprefixが整合しているかはValidationできますが、文字列規約だけを所属関係の正本にはしません。

### QueryやCapabilityの定義をPatternへ複製しない

Patternは、Semantic Data、Relation、Query、CapabilityなどをCanonical IDで参照します。

たとえば`browse` Patternがnative SQL Queryを利用しても、SQL本文をPattern側へコピーしません。Queryの正本はQuery側に残し、Patternは「ApplicationとしてこのQueryを利用する」という関係だけを持ちます。

### Source IndexではPatternを独立したSubjectとして扱う

Pattern定義は、次のSubjectとしてSource Indexへ登録します。

```text
Kind = application-pattern
ID   = <Pattern Canonical ID>
Role = definition
```

Source Indexの既存モデルは任意の論理Subjectを扱えるため、Pattern専用のSource Index基盤は追加しません。

### Editor内部状態を第二の正本にしない

当面のEditorは、Pattern定義ファイルのProjectionと編集支援を担います。

同じApplication上の意図について、`patterns/*.tei`とEditor内部状態の二つを暗黙の正本にはしません。将来Editor-authored modeを導入する場合は、RQ-003に従って正本の切り替え方を別途定めます。

### 表層構文はこのADRで固定しない

このADRが固定するのは、正本の配置、identityとpathの分離、Applicationへの所属、既存資産を参照する方針です。

`pattern`キーワードの詳細、field名、複数Patternを一ファイルへ書けるようにするかといった構文上の判断は、Parser実装とReferenceから得られるEvidenceを見て調整できます。

## 影響

### 良くなる点

- Patternの正本が明確になる
- Application本体を過度に大きくせずに済む
- Pattern単位で差分をレビューしやすい
- Source IndexとDiagnosticをPattern単位で接続しやすい
- ファイルの改名・移動とCanonical identityを分離できる
- Queryやnative SQLなどの既存正本を維持できる
- 将来Editorから特定Patternを開く単位が明確になる

### 増える負担

- Application配下のファイル数が増える
- Loaderに`patterns/*.tei`の読み込み経路が増える
- Pattern定義とApplicationの所属関係を検証する必要がある
- 将来、複数Patternを一ファイルにまとめる場合はParserとSource Indexの拡張が必要になる

この負担は、Application Patternとfile path、Editor状態、Target固有表現を混同しないために受け入れます。

## 採用しない案

### `application.tei`へすべて埋め込む

初期実装は単純になりますが、Application本体、Database Requirement、複数Patternの変更が一つのファイルへ集中します。Pattern単位のSource Indexや差分レビューを考えると、v1の基準にはしません。

### World共通Pattern Catalogから始める

将来有用になる可能性はありますが、現時点では複数Applicationで本当に共有できる意味を実証していません。`order_entry`一件から得た分類を全体へ早い段階で固定してしまうため、採用しません。

### Editor内部状態を正本にする

RQ-003に反します。Editor-authored modeが必要になった時点で、正本の扱いを改めて設計します。

## 検証

`ISSUE-000005`では、少なくとも次を確認します。

- `patterns/*.tei`からPatternを読み込める
- Patternが正しいApplicationに所属する
- Pattern IDがfile pathから独立している
- Pattern定義をSource Indexから追跡できる
- ファイル改名後もCanonical World上のPattern意味が変わらない
- QueryやCapabilityの定義をPattern側へ複製しない
- 構文エラーや参照エラーを元のPattern定義へ戻せる

## 関連文書

- `plan/design/APPLICATION_PATTERN_V1_DESIGN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
