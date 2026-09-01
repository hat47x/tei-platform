# ISSUE-000012: `sales.order.lines`をheader-detail Application Patternとして実証する

- 状態: Done
- GitHub Issue: #118（完了）
- 実装PR: #121 / #123
- 前提Semantic: `ISSUE-000011` / GitHub Issue #117 / PR #119
- 上位Issue: `ISSUE-000005` / GitHub Issue #67

## 目的

成立済みSemantic Relation `sales.order.lines`を、Applicationが一つのHeaderと複数Detailとして扱う意図を、Target固有UIやPersistenceへ依存せず表現できることを実証する作業です。

## 実証した契約

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

契約は次です。

- `relation`: 必須
- 参照Relationのcardinality: `many`を必須とする
- `data`: 受理しない
- `query`: 受理しない
- `capability`: 受理しない
- `datasource`: 受理しない

Relationがsource / target / cardinalityを保持するため、Header Data、Detail Data、cardinalityをPatternへ再記述しません。

## PR #121 — Canonical model / Parser / Validation

PR #121で次を実装しました。

- `PatternHeaderDetail`を有効化
- Relation-onlyの`header-detail`をParserで受理
- Relationなしを拒否
- Data / Query / Capability / Datasourceを拒否
- Workspace ValidationでRelationの存在を確認
- Relationのcardinalityが`many`であることを確認
- `one` Relationには`APPLICATION_PATTERN_RELATION_NOT_MANY`を返す
- Pattern Diagnosticを既存`application-pattern:<Canonical ID>` Subjectへ接続
- 共通`validation.Validate`からDiagnosticを一度だけ返す

ParserはRelationの実体を知らないため、cardinality判定はWorkspace Validationへ置きました。

CI #279は`verify` / PostgreSQL integrationとも成功しました。

## PR #123 — 実Reference

Reference Workspaceへ次を追加しました。

```tei
pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

Conformanceで次を確認しています。

- 5種類のPatternが同じ`order_entry`で共存する
- `sales.order.lines`をCanonical IDで追跡できる
- Relation sourceが`sales.order`である
- targetが`sales.order_line`である
- cardinalityが`many`である
- PatternがData / Query / Capability / Datasourceを重複保持しない
- Source Indexからauthoring sourceへ戻れる
- `header-detail`追加だけで`OrderLine` Persistence Mappingを生成しない
- `header-detail`追加だけでDatabase Contractへ`OrderLine`を公開しない
- Reference WorldをRepository自身から再生成できる

Golden Worldの差分はPattern一件に対応するCanonicalな差分だけでした。

CI #282は`verify` / PostgreSQL integrationとも成功しました。

## 実証から得た判断

### Relationだけで成立する

Header Data、Detail Data、cardinalityを別fieldとして持つ必要はありませんでした。

Patternには「どの1:N RelationをHeader / Detailとして扱うか」というApplication固有の意図だけを残します。

### `many`は参照整合性として検査する

cardinality自体をPatternへコピーしませんが、`header-detail`の意味を満たすRelationであることはWorkspace Validationから確認します。

### `header-detail`はUI layoutではない

WebのHeader Form + GridをCanonicalな意味にはしません。

Web、Desktop、APIその他のTargetは、同じApplication上の意図をそれぞれ自然なPresentationへProjectionできます。

### Semantic RelationとDatabase Foreign Keyを分ける

Semantic RelationやApplication Patternの存在だけを理由に、Persistence Mapping、Foreign Key、Migration、Contractを生成しません。

保存・取得で必要になる物理構造は、後続Driver / PersistenceのEvidenceから設計します。

### Datasource / Query / Validation順序を先回りしない

Relation-onlyで実Referenceが成立したため、DatasourceやQueryをPatternへ追加するEvidenceはありませんでした。

HeaderとDetailをまたぐValidationは後続Scenarioで必要ですが、それをPattern自身のfieldにするかは別問題です。具体的な業務ruleから決めます。

## 完了条件の確認

1. `PatternHeaderDetail`をCanonical model / Parser / Loader / Normalizationから扱える — 完了
2. Relation-onlyでheader-detail意図を表現できる — 完了
3. Relationの存在と`many`を共通Validationから検査できる — 完了
4. Pattern Diagnosticからauthoring sourceへ戻れる — 完了
5. 不要なData / Query / Datasource / Capabilityを受理しない — 完了
6. 5種類のPatternが`order_entry`で共存できる — 完了
7. Reference WorldをRepositoryだけから再生成できる — 完了
8. Persistence / FK / UIを追加せずApplication上の意図を成立させられる — 完了
9. 後続DriverへHeader / DetailのTarget非依存情報を渡せる — Application Pattern側は完了

## 後続

5種類のApplication Patternは、このIssueの完了によって一通り実Referenceまで成立しました。

次は`ISSUE-000013` / GitHub Issue #124でGate A readinessを横断監査し、既存認証Compatibility、業務Validation、Capability Binding、Business Result / Execution Error境界をDriverへ渡せるかを確認します。

## 関連

- GitHub Issue #67 / #69 / #117 / #118 / #124
- GitHub PR #119 / #121 / #123
- `ISSUE-000005` / `ISSUE-000011` / `ISSUE-000013`
- ADR-0017 / ADR-0018
