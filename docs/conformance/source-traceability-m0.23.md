# Conformance — M0.23 Source-to-Canonical Traceability

この文書は、M0.23で検証しているSource-to-Canonical Traceabilityの不変条件と、Repository上の実装・テスト・CLIの対応をまとめたものです。

TEIはまだproduction-readyなEditorではありません。ここで確認するのは、Authoring sourceとCanonical meaningを混同せず、それでも利用者が意味から正確なsourceへ戻れる境界です。

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

Canonical WorldとSource Indexは別の情報です。Source IndexはWorkspaceから再生成できるderived sidecarであり、Canonical identityの一部ではありません。

## Conformance項目

### 1. File pathはsemantic identityではない

Canonical IDはsource pathから導出しません。同一内容・同一Canonical declarationを別directoryへ移しても、normalized Worldは変わりません。

Source Indexのlocatorだけが新しいWorkspace-relative pathへ更新されます。

主な検証は次のとおりです。

- `internal/workspace/source_index_invariance_test.go`
- `internal/workspace/traceability_conformance_test.go`

### 2. Source Artifactはcontent fingerprintを持つ

Source IndexのArtifactはWorkspace-relative path、format、`sha256:` content fingerprintを持ちます。

同じcontentをmoveしただけならfingerprintは不変です。contentを変えればfingerprintは変化します。

主な実装は次のとおりです。

- `internal/sourceindex/model.go`
- `internal/workspace/source_index.go`

### 3. Source Rangeはbyte offsetが基準

Rangeは半開区間`[start,end)`のbyte offsetを基準とし、line / columnはそのcontentから検証可能な表示情報として保持します。

Parserのsyntax anchorとSource Index rangeはdomain modelへphysical locationを埋め込まずに接続します。

主な実装は次のとおりです。

- `internal/syntax/anchor.go`
- `internal/syntax/semantic_anchor.go`
- `internal/sourceindex/model.go`

### 4. Semantic diagnosticはSubject経由でsourceへ戻る

ValidationはSource Indexを必須依存にしません。

```text
Validate(World)
  -> Diagnostic + path-independent Subject
  -> ProjectDiagnostics(Source Index)
  -> source reference
```

Sourceを見つけられないderived / synthetic subjectは、推測でpathを作らずsourceなしのdiagnosticとして保持します。

主な実装は次のとおりです。

- `internal/validation/validation.go`
- `internal/workspace/diagnostic_projection.go`

### 5. Parse diagnosticはsource-directでよい

Parse / lexer errorではCanonical subjectがまだ成立していない場合があります。その段階で擬似的なsemantic identityを作りません。

Loaderは位置付きsyntax errorをsource artifact + rangeへ結び、`PARSE_ERROR`としてCLIへ渡せます。

主な実装は次のとおりです。

- `internal/syntax/error.go`
- `internal/workspace/source_diagnostic.go`

### 6. Native SQLはnative fileのままauthoritative sourceになれる

Query declarationは次の両方を扱います。

```tei
sql postgres {
    SELECT ...
}
```

```tei
sql postgres file "recent_orders.sql"
```

native formでは`.sql`がauthoritative authoring sourceです。Workspace loaderがSQL本文を読み、既存`query.EmbeddedSQL.Source`へ渡します。

Physical file pathは`query.EmbeddedSQL`へ追加しません。

Reference Workspaceでは次の二つが同じQuery Subjectへ結び付きます。

```text
query:order_entry.recent_orders
  definition -> applications/order-entry/queries/recent_orders.tei
  authoring  -> applications/order-entry/queries/recent_orders.sql
```

### 7. Native SQLからAnalyzerへphysical pathを渡さない

Analyzer protocolはCanonical Query ID、dialect、SQL source等を受け取りますが、authoring file pathを要求しません。

このためSQL analyzer / Derived Query IRは、Workspace layoutをsemantic meaningとして取り込みません。

主な実装は次のとおりです。

- `internal/query/analyzer/model.go`
- `internal/workspace/traceability_conformance_test.go`

### 8. Derived artifactからauthoritative sourceへ暗黙に逆書きしない

M0.23ではSource Index、Diagnostic projection、Analyzer result等から`.tei` / `.sql`を自動書き換えるAPIを設けません。

将来Editorが修正操作を提供する場合も、Source Indexは「どこから来たか」を示す情報であって、「derived artifactを正としてauthoritative sourceを上書きしてよい」という権限ではありません。

### 9. Authoritative sourceの競合をheuristicにmergeしない

一つのsubjectへ複数source roleを持たせること自体は可能です。たとえばQuery declarationの`definition`とSQL本文の`authoring`は役割が異なります。

一方、同一roleで競合するauthoritative sourceを、filenameや探索順序から推測して一つへ決める設計にはしません。必要なら明示的なcomposition ruleを追加します。

## CLIで確認する

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

このJSONはinspection用のderived projectionです。保存済みCanonical artifactではありません。

Diagnostic projection:

```bash
go run ./cmd/tei validate \
  -workspace reference/workspace
```

source referenceを解決できるfindingは`path:line:column`へ戻ります。

## Reference / CI

Source Index全体のgolden fileは保存していません。現段階では、Index全体の差分を恒久的なreview artifactにするより、次を固定する方が意味境界を明確に保てると判断しています。

- generationがdeterministicであること
- Artifact fingerprint / rangeがcontentと整合すること
- 重要subjectのsource roleとlocatorが正しいこと
- rename / moveでWorldが変わらないこと
- Reference Queryがnative SQLから同じ`reference/world.json`を再生成できること

常設CIでは`order_entry.recent_orders`のSource Index projectionを実行し、definition / authoring pathとfingerprintを検査します。

## M0.23で主張しないこと

このConformanceは次を意味しません。

- production-ready GUI / LSP
- 全source formatへの対応
- Go source semantic analysis
- SQL formatter / refactoring engine
- AIによるauthoritative source自動修正
- 全SQL dialectの完全解析

M0.23の到達点は、source layoutとsemantic identityを分離したまま、source → meaning → diagnostic / derived boundaryをRepository内で追跡可能にしたことです。
