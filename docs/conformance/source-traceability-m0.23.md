# M0.23 入力元から正本までの追跡可能性に関する準拠確認

この文書では、M0.23で検証する入力元から意味の正本までの追跡可能性について、守るべき条件とリポジトリ内の実装・テスト・コマンドラインとの対応を整理します。

現在のTEIは、本番利用できる完成済みの編集環境ではありません。ここでは、入力元と意味の正本を区別しつつ、利用者が意味上の対象から正確な入力元へ戻れることを確認します。

## 基本境界

```text
Authoring source
  .tei / .sql / ...
        │ parse / load
        ├──────────────────────┐
        ▼                      ▼
Canonical World          Source Index
path-independent         workspace-relative locator
semantic identity        content fingerprint
meaning / lifecycle      byte range / role
        │                      ▲
        │ validate             │ project
        ▼                      │
Diagnostic + Subject ──────────┘

native SQL
  ↓ Workspace resolves file
Query.Source
  ↓
Analyzer Request / Derived Query IR
```

意味の正本（Canonical World）と入力元索引（Source Index）は別の情報です。入力元索引は作業領域から再生成できる付随情報であり、意味上の識別子には含めません。

## 準拠確認の項目

### 1. ファイルの場所は意味上の識別子ではない

Canonical IDは入力ファイルのパスから生成しません。同じ内容・同じ意味の宣言を別のディレクトリへ移しても、正規化された全体モデルは変わりません。

変更されるのは、入力元索引に記録する作業領域からの相対パスだけです。

主な検証は次のとおりです。

- `internal/workspace/source_index_invariance_test.go`
- `internal/workspace/traceability_conformance_test.go`

### 2. 入力元の成果物は内容の照合値を持つ

入力元索引の成果物（Artifact）は、作業領域からの相対パス、形式、内容の照合値（`sha256:`）を持ちます。

ファイルの内容を変えずに移動しただけなら照合値は変わりません。内容を変更した場合は、照合値も変わります。

主な実装は次のとおりです。

- `internal/sourceindex/model.go`
- `internal/workspace/source_index.go`

### 3. 入力範囲はバイト位置で示す

範囲は半開区間`[start,end)`で表すバイト位置を基準とし、行番号・列番号は内容から検証できる表示用の情報として保持します。

構文解析器が示す構文上の位置と入力元索引の範囲は、業務モデルにファイル上の位置を埋め込まずに結び付けます。

主な実装は次のとおりです。

- `internal/syntax/anchor.go`
- `internal/syntax/semantic_anchor.go`
- `internal/sourceindex/model.go`

### 4. 意味上の診断から対象を介して入力元へ戻る

意味の検証は、入力元索引を必須の依存先とはしません。

```text
Validate(World)
  -> Diagnostic + path-independent Subject
  -> ProjectDiagnostics(Source Index)
  -> source reference
```

入力元が見つからない派生対象や合成対象については、推測でファイルパスを作らず、入力元への参照を持たない診断として保持します。

主な実装は次のとおりです。

- `internal/validation/validation.go`
- `internal/workspace/diagnostic_projection.go`

### 5. 構文解析の診断は入力元を直接示せる

構文解析や字句解析でエラーが起きた段階では、意味上の対象がまだ成立していない場合があります。その場合、架空の意味上の識別子は作りません。

読み込み機構は、発生位置を持つ構文エラーを入力元の成果物と範囲へ結び付け、`PARSE_ERROR`としてコマンドラインへ渡せます。

主な実装は次のとおりです。

- `internal/syntax/error.go`
- `internal/workspace/source_diagnostic.go`

### 6. SQLはSQLファイルのまま権威ある入力元になれる

問い合わせの宣言では、次の両形式を扱います。

```tei
sql postgres {
    SELECT ...
}
```

```tei
sql postgres file "recent_orders.sql"
```

外部のSQLファイルを使う形式では、`.sql`がSQL本文の正本となる入力元です。作業領域の読み込み機構が本文を読み取り、既存の`query.EmbeddedSQL.Source`へ渡します。

物理的なファイルのパスは`query.EmbeddedSQL`へ追加しません。

参照用の作業領域では、次の二つが同じ問い合わせの対象に結び付きます。

```text
query:order_entry.recent_orders
  definition -> applications/order-entry/queries/recent_orders.tei
  authoring  -> applications/order-entry/queries/recent_orders.sql
```

### 7. SQL解析器へ物理的な場所を渡さない

解析器との通信規約は、問い合わせのCanonical ID、SQLの方言、SQL本文などを受け取りますが、記述元ファイルのパスは要求しません。

そのため、SQL解析器や派生した問い合わせの中間表現には、作業領域のディレクトリ構成を業務上の意味として取り込みません。

主な実装は次のとおりです。

- `internal/query/analyzer/model.go`
- `internal/workspace/traceability_conformance_test.go`

### 8. 派生成果物から権威ある入力元へ暗黙に書き戻さない

M0.23では、入力元索引、診断結果の派生表示、解析結果などを使って`.tei`や`.sql`を自動的に書き換えるAPIは設けません。

将来、編集画面から修正できるようになっても、入力元索引が示すのは「情報がどこから来たか」です。派生した成果物を正本とみなして入力元を上書きする権限を与えるものではありません。

### 9. 権威ある入力元の競合を推測で統合しない

一つの対象に、役割の異なる複数の入力元を結び付けることはできます。たとえば問い合わせの宣言における`definition`と、SQL本文の`authoring`は別の役割です。

一方、同じ役割の正本が競合する場合、ファイル名や探索順から推測して一つに決めることはしません。必要であれば、明示的な統合規則を追加します。

## コマンドラインで確認する

Source Index全体:

```bash
go run ./cmd/tei source-index \
  -workspace reference/workspace
```

特定subject:

```bash
go run ./cmd/tei source-index \
  -workspace reference/workspace \
  -subject-kind query \
  -subject-id order_entry.recent_orders
```

このJSONは内容を調べるために生成した表示情報であり、保存済みの正本ではありません。

Diagnostic projection:

```bash
go run ./cmd/tei validate \
  -workspace reference/workspace
```

入力元への参照を解決できる診断結果は、`path:line:column`の形式で元の場所を示します。

## 参照実装と自動検査

入力元索引全体の基準ファイルは保存していません。現段階では、索引全体の差分を永続的な確認用成果物とするよりも、次の条件を固定する方が意味の境界を明確に保てると判断しています。

- generationがdeterministicであること
- Artifact fingerprint / rangeがcontentと整合すること
- 重要subjectのsource roleとlocatorが正しいこと
- rename / moveでWorldが変わらないこと
- Reference Queryがnative SQLから同じ`reference/world.json`を再生成できること

常設の継続的な検査では、`order_entry.recent_orders`の入力元索引を生成し、定義とSQL本文のパス、照合値を検査します。

## M0.23で保証する範囲外

この準拠確認によって、次の機能が成立したとはみなしません。

- production-ready GUI / LSP
- 全source formatへの対応
- Go source semantic analysis
- SQL formatter / refactoring engine
- AIによるauthoritative source自動修正
- 全SQL dialectの完全解析

M0.23では、ファイルの配置と意味上の識別を区別したまま、入力元から意味、診断、派生情報までをリポジトリ内で追跡できる境界を確認しました。
