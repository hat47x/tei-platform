# ISSUE-000010: Application Pattern Validationを共通Validateへ統合する

- 状態: Done — 実装済み（`validation.Validate`がApplication Pattern検証を呼ぶ。Application Pattern Stage 1 / Gate A Ready）
- GitHub Issue: #98
- 上位Issue: [`ISSUE-000005`](000005-application-pattern-reference.md)

## 背景

Application Patternの`browse`基盤は、Canonical model、Parser、Loader、Normalization、Source Indexに続き、Workspace全体を見た参照整合性Validationまで進んでいます。

PR #95では、Patternが参照するData / Query / Datasourceの存在、QueryのApplication所有関係、Datasourceの矛盾、Pattern IDの重複などを検査するPattern固有Validationを実装しました。

PR #99では、そのValidationを外部から合成できる`ValidateApplicationPatterns`入口と、Pattern固有Diagnosticを`application-pattern:<Canonical ID>` Subjectから既存Source Indexへ渡し、Pattern IDのauthoring sourceへ戻せることを確認しています。

ただし、通常の`validation.Validate`はまだPattern固有Validationを呼びません。このため、利用側がApplication Patternを特別扱いして追加のValidation入口を呼ばなければ、Pattern参照不整合を見落とす余地があります。

## 目的

Pattern固有Validationを通常のWorkspace validation契約へ統合し、CLI、Editor、Driverなどの利用側がApplication Patternだけを例外扱いしなくても済む状態を作ります。

この作業では新しい検査規則を増やすことより、すでに成立したPattern固有Validationを既存Validationの責務へ自然に組み込むことを重視します。

## 現在までに確認済みのこと

PR #95 / #99により、少なくとも次は確認済みです。

- Pattern Canonical IDの重複を検出できる
- 未知のSemantic Dataを検出できる
- 未知のQueryを検出できる
- 別Applicationが所有するQueryへの参照を検出できる
- 未知の明示Datasourceを検出できる
- Pattern側DatasourceとQuery側Datasourceの矛盾を検出できる
- Pattern側Datasourceを省略した場合、Query側Datasourceだけで成立できる
- Pattern固有Diagnosticが`application-pattern:<Canonical ID>` Subjectを保持する
- そのSubjectから既存Source Indexを通じてPattern ID tokenへ戻れる

これらをこのIssueで作り直しません。

## 実装方針

### 共通`Validate`へ一度だけ接続する

Pattern固有Validationを`validation.Validate(w)`から呼び、Diagnosticを通常のValidation結果へ統合します。

既存Validationの検査順序や責務を不必要に変えず、Pattern固有の参照整合性だけを追加できる位置を選びます。

### 集約経路をConformance化する

専用入口ではなく`validation.Validate(w)`を通し、次を確認します。

- 正常なPatternへ余計なDiagnosticを追加しない
- Pattern固有の参照不整合が通常のValidation結果へ現れる
- Pattern ID重複も通常のValidation結果へ現れる
- Diagnostic SubjectがPattern identityを保つ

### Diagnosticの重複を増やさない

既存ValidationとPattern固有Validationが同じ原因を別々に検出する場合は、利用者へ同義のDiagnosticを二重に提示しないかを確認します。

機械的な後処理で重複を隠すことを第一選択にはせず、どのValidationが何を判断するかという責務境界を優先して調整します。

## 責務境界

- Parserは、必須fieldや重複fieldなど、構文だけで判断できる事項を扱う
- Workspace Validationは、Data / Query / Datasource等の参照整合性を扱う
- Queryの返却Dataを、現行モデルに十分な意味情報がない状態で推測しない
- Query / Datasource側ですでに扱うread / write境界をPattern側へ重複実装しない
- Pattern専用のSource Indexを新設しない

## 非目標

- Diagnostic projection基盤の再実装
- `order_entry.recent_orders`の実Reference追加
- `reference/world.json`の更新
- 実Referenceでのfile rename invariance
- 後続Pattern Kindの追加

実Referenceは[`ISSUE-000068`](000068-order-entry-browse-reference.md)で扱います。

## 完了条件

次をRepository内のtest / Conformanceから再現できることを完了条件とします。

1. `validation.Validate`を通すだけでPattern参照不整合を検出できる
2. 正常なPatternへ不要なDiagnosticを追加しない
3. Pattern固有DiagnosticのSubjectを維持する
4. 同じ原因によるDiagnosticを不必要に重複させない
5. 既存Validationの意味や順序を不必要に変更しない

## 関連

- GitHub Issue #98
- GitHub Issue #67
- PR #95
- PR #99
- ADR-0017
- ADR-0018
- `plan/design/APPLICATION_PATTERN_V1_DESIGN.md`

関連RQ: RQ-001, RQ-002, RQ-003, RQ-014, RQ-021
