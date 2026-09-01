# ISSUE-000010: `sales.order.submit`をcommand Application Patternとして実証する

- 状態: Done
- GitHub Issue: #110（完了）
- 実装PR: #114 / #116
- 上位Issue: `ISSUE-000005` / GitHub Issue #67
- 関連設計: [`APPLICATION_PATTERN_V1_DESIGN.md`](../APPLICATION_PATTERN_V1_DESIGN.md)
- Capability結果境界: [ADR-0020](../adr/0020-capability-business-outcome-boundary.md) / [`CAPABILITY_OUTCOME_V1_DESIGN.md`](../CAPABILITY_OUTCOME_V1_DESIGN.md)

## 目的

`order_entry`の業務操作として、既存Semantic Capability `sales.order.submit`をApplication Patternから参照できることを実証しました。

確認したかったのは、ButtonやHTTP endpointの形ではありません。Applicationが「どの業務Capabilityを呼び出すか」というTarget非依存の意図を保持し、その意味を後続Driverへ渡せることです。

## 成立したCanonical契約

実Referenceは次の最小形で成立しました。

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

現時点の契約は次です。

- `capability`: 必須
- `data`: 受理しない
- `relation`: 受理しない
- `query`: 受理しない
- `datasource`: 受理しない

Capabilityがすでに持つinput / outputや、実装方式・DatasourceをPatternへ重複記述していません。

## PR #114 — Kind / Parser / Validation

PR #114で次を実装・検証しました。

- `PatternCommand`を有効化
- `Pattern`へ型付きの`Capability`参照を追加
- Capability-onlyの`command`をParserで受理
- Capabilityなしの`command`を拒否
- Data / Relation / Query / Datasourceを記述した`command`を拒否
- `browse` / `record` / `reference`へCapabilityを記述した場合も拒否
- Workspace ValidationでSemantic Capabilityの存在を確認
- 未知Capabilityを`application-pattern:<Canonical ID>` Subject付きDiagnosticとして報告
- 共通`validation.Validate`でも同じDiagnosticを一度だけ返すことを確認
- Capabilityのinput / outputをPatternが重複保持しないことをConformance化

CI #270は`verify` / PostgreSQL integrationとも成功しています。

## PR #116 — 実Reference

PR #116でReference Workspaceへ`order_entry.submit_order`を追加しました。

Repository内のConformanceでは、次を確認しています。

- `browse` / `record` / `reference` / `command`の4種類が同じ`order_entry`で共存する
- `command`から`sales.order.submit`をCanonical IDで追跡できる
- Capability input `order sales.order`をSemantic側の正本から確認できる
- Capability output `sales.order`をSemantic側の正本から確認できる
- PatternがData / Relation / Query / Datasourceを重複保持しない
- `order_entry`が既存定義で`uses semantic sales`を宣言している
- `application-pattern:order_entry.submit_order`から実authoring sourceへ戻れる
- `reference/world.json`をRepository自身の`tei world`から再生成できる

Golden差分は`command` Pattern一件に対応する追加だけでした。CI #275も`verify` / PostgreSQL integrationとも成功しています。

## 実証から得た判断

### Capability identityだけで成立する

`command`のCanonicalな意味は、現時点ではCapability identityだけで表現できます。

既存`sales.order.submit`はSemantic側で次を保持しています。

```tei
capability sales.order.submit {
    input order sales.order
    output sales.order
}
```

そのため、`data sales.order`やinput / outputをPatternへ再記述しません。

### Binding / implementationはPatternへ含めない

Patternは「どのCapabilityを業務操作として扱うか」を表します。

一方、どの実装でそのCapabilityを実現するかはBinding / Realization / Driver側の責務です。

Custom Go、Plugin、Database、外部APIなどへ実装方式が変わっても、Application上のcommand identityを変えない境界を維持します。

### DatasourceはPatternへ持ち込まない

Capability implementationが利用するresourceは実装境界の情報です。

実Referenceからも、`command` PatternへDatasourceを追加する必要性は確認されませんでした。

### Business Result / Execution Errorを混同しない

ADR-0020に従い、既存`output sales.order`を通常のBusiness Resultとして維持しています。

`command`を追加したという理由だけでnamed Outcomeへ書き換えていません。Database接続失敗やadapter異常などのExecution Error、外部世界から得るObservationもPatternへ混在させていません。

### `SemanticUses`は横断論点として残す

実Referenceでは`order_entry`が`uses semantic sales`を持つことを確認しています。

ただし、「Patternが参照するSemantic assetはApplicationの`SemanticUses`に属する必要があるか」という検査はData / Relation / Capabilityに共通する論点です。

`command`だけの規則として追加せず、Driverや後続Referenceで必要性が明確になった場合にPattern全体の共通境界として検討します。

## 完了条件の確認

1. `command` KindとCapability参照をCanonical model / Parser / Loader / Normalizationから扱える — 完了
2. `sales.order.submit`のinput / outputを複製せずcommand意図を表現できる — 完了
3. Capabilityの存在を共通Validationから検査できる — 完了
4. Pattern Diagnosticが既存Subject / Source Index経路を利用できる — 完了
5. Capability Binding / implementationをPatternから分離できる — 完了
6. 既存`output sales.order`をBusiness Resultとして維持し、Execution Errorと混同しない — 完了
7. `browse` / `record` / `reference` / `command`が`order_entry`で共存できる — 完了
8. Reference WorldをRepositoryだけから再生成できる — 完了
9. 後続DriverへCapability identityとADR-0020の結果境界をTarget非依存の情報として渡せる — Application Pattern側の準備は完了

## 後続

次は`header-detail`へ進みます。

ただしPatternを先に作らず、`ISSUE-000011` / GitHub Issue #117で`sales.OrderLine`と`sales.order.lines -> sales.order_line many`をSemantic Referenceとして成立させます。

その後、`ISSUE-000012` / GitHub Issue #118でRelationを利用した`header-detail` Application Patternを実証します。

## 関連

- GitHub Issue #67: v1 Application Pattern実証
- GitHub Issue #69: Application Driver
- GitHub Issue #77: Capabilityの業務Outcomeと実行エラー
- GitHub Issue #110: `command`実証（完了）
- GitHub PR #114 / #116
- `ISSUE-000011`: `OrderLine` Semantic Reference
- `ISSUE-000012`: `header-detail`実証
- ADR-0017: Application Pattern ownership
- ADR-0018: Application Pattern authoring source
- ADR-0020: Capability Business Outcome boundary
- RQ-001, RQ-003, RQ-008, RQ-011, RQ-014, RQ-017, RQ-018, RQ-021
