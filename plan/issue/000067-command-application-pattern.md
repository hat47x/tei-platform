# ISSUE-000067: `sales.order.submit`をcommand Application Patternとして実証する

- 状態: Done
- GitHub Issue: #110（完了）
- 実装PR: #114 / #116
- 上位Issue: `ISSUE-000005` / GitHub Issue #67
- 関連設計: [`APPLICATION_PATTERN_V1_DESIGN.md`](../design/APPLICATION_PATTERN_V1_DESIGN.md)
- Capability結果境界: [ADR-0020](../adr/0020-capability-business-outcome-boundary.md)

## 目的

`order_entry`の業務操作として、既存Semantic Capability `sales.order.submit`をApplication PatternからTarget非依存に参照できることを実証する作業です。

Button、HTTP endpoint、Custom Go等の実装方法をPatternの意味にせず、Applicationが「どの業務Capabilityを呼ぶか」だけを保持できるかを確認しました。

## 実証した契約

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

契約は次です。

- `capability`: 必須
- `data`: 受理しない
- `relation`: 受理しない
- `query`: 受理しない
- `datasource`: 受理しない

Capabilityのinput / outputはSemantic側を正本とします。

```tei
capability sales.order.submit {
    input order sales.order
    output sales.order
}
```

## PR #114 — Canonical model / Parser / Validation

PR #114で次を実装しました。

- `PatternCommand`を有効化
- Patternへ型付きCapability参照を追加
- Capability-onlyの`command`をParserで受理
- Capabilityなしの`command`を拒否
- Data / Relation / Query / Datasourceを拒否
- ほかのPattern KindへCapabilityが混入した場合も拒否
- Workspace ValidationでCapabilityの存在を確認
- Pattern Diagnosticを`application-pattern:<Canonical ID>` Subjectで返す
- 共通`validation.Validate`から同じDiagnosticを一度だけ返す

CI #270は`verify` / PostgreSQL integrationとも成功しました。

## PR #116 — 実Reference

Reference Workspaceへ次を追加しました。

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

Conformanceで次を確認しています。

- `browse` / `record` / `reference` / `command`が同じApplicationで共存する
- `sales.order.submit`をCanonical IDで追跡できる
- input `order sales.order`をSemantic側から取得できる
- output `sales.order`をSemantic側から取得できる
- Data / Relation / Query / DatasourceをPatternへ重複保持しない
- `order_entry`が`uses semantic sales`を保持する
- Source Indexからauthoring sourceへ戻れる
- Reference WorldをRepository自身から再生成できる

CI #275は`verify` / PostgreSQL integrationとも成功しました。

## 実証から得た判断

### Capability identityだけで成立する

対象Data、input、outputをPatternへ重複記述する必要はありませんでした。

### Capability Binding / implementationはPatternへ含めない

同じCapabilityがbuiltin、generated、Custom Go、process plugin、外部API等の異なる方式で実現されても、Application上のcommand identityを変えない構造を維持します。

### DatasourceはPatternへ持ち込まない

Capability implementationが利用するresourceはBinding / Realization / Driver側の情報です。

`command`のApplication上の意図を表すためにDatasourceを追加するEvidenceは得られませんでした。

### Business ResultとExecution Errorを混同しない

既存`output sales.order`はADR-0020に従うBusiness Resultとして維持しました。

`command`追加だけを理由にnamed Outcomeへ変更していません。

Database接続失敗、adapter異常、process起動失敗等はExecution Errorであり、業務Outcomeへ変換しません。

### CommandとObservationを分離する

Capability呼び出しのBusiness Resultと、その後に外部世界から観測するActual / Observationは別の意味です。

Observationを`command` Patternへ持ち込みません。

## 完了条件の確認

1. `command` KindとCapability参照をCanonical model / Parser / Loader / Normalizationから扱える — 完了
2. input / outputをPatternへ複製せずcommand意図を表現できる — 完了
3. Capabilityの存在を共通Validationから検査できる — 完了
4. Pattern Diagnosticが既存Subject / Source Index経路を利用できる — 完了
5. Capability Binding / implementationをPatternから分離できる — 完了
6. 既存outputをBusiness Resultとして維持し、Execution Errorと混同しない — 完了
7. 4種類のPatternが`order_entry`で共存できる — 完了
8. Reference WorldをRepositoryだけから再生成できる — 完了
9. 後続DriverへCapability identityと結果境界を渡せる — Application Pattern側は完了

## 関連

- GitHub Issue #67 / #69 / #77 / #110
- GitHub PR #114 / #116
- ADR-0017 / ADR-0018 / ADR-0020
- `ISSUE-000005` / `ISSUE-000009`
