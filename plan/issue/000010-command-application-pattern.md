# ISSUE-000010: `sales.order.submit`をcommand Application Patternとして実証する

- 状態: Ready
- GitHub Issue: #110
- 上位Issue: `ISSUE-000005` / GitHub Issue #67
- 関連設計: [`APPLICATION_PATTERN_V1_DESIGN.md`](../APPLICATION_PATTERN_V1_DESIGN.md)
- Capability結果境界: [ADR-0020](../adr/0020-capability-business-outcome-boundary.md) / [`CAPABILITY_OUTCOME_V1_DESIGN.md`](../CAPABILITY_OUTCOME_V1_DESIGN.md)

## 目的

`order_entry`の業務操作として、既存Semantic Capability `sales.order.submit`をApplication Patternから参照できることを実証します。

ここで確認したいのは、ButtonやHTTP endpointの形ではありません。Applicationが「どの業務Capabilityを呼び出すか」というTarget非依存の意図を保持し、その意味を後続Driverへ渡せることです。

既存Capabilityがすでに持つ入力型、出力型、Datasource、実装方式をPatternへ複製せず、Canonical IDによる参照だけで成立するかを確かめます。

## 前提

`browse`、`record`、`reference`の最小実証は完了しています。

既存Capabilityは次の定義です。

```tei
capability sales.order.submit {
    input order sales.order
    output sales.order
}
```

Application `order_entry`は`uses semantic sales`を宣言済みです。

Capabilityの業務上の結果と実行エラーの境界はADR-0020で定めています。既存の単一`output`はBusiness Resultとして利用でき、名前付きOutcomeは複数の業務上の結末が実例として必要な場合にだけ導入します。

## 最小構文の第一候補

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

初期契約は次の形から検証します。

- `capability`: 必須
- `data`: 受理しない
- `relation`: 受理しない
- `query`: 受理しない
- `datasource`: 受理しない

Capabilityのinput / outputはSemantic側を正本とし、Patternでは再記述しません。

## 実装する範囲

### Canonical model

- `PatternCommand`を有効化する
- `Pattern`へ型付きの`Capability`参照を追加する
- 既存Kindの意味を変更しない

### Parser

- `command`で`capability`を必須にする
- `command`にData / Relation / Query / Datasourceが記述された場合は拒否する
- `browse` / `record` / `reference`にCapabilityが記述された場合も拒否する
- 同じfieldの重複は既存方針どおり構文段階で拒否する

### Workspace Validation

少なくとも次を確認します。

- CapabilityがCanonical Worldに存在する
- `command`にCapabilityが設定されている
- Kind固有の禁止fieldが混入していない
- Diagnosticが`application-pattern:<Canonical ID>` Subjectを利用する
- CapabilityのSemantic NamespaceをApplicationから利用できることについて、既存`SemanticUses`とValidationで保証済みかを先に確認する

Semantic Namespace利用境界が既存Validationで十分に守られている場合、Pattern専用の同等検査を重ねません。不足がReferenceから確認できた場合だけ、より共通的な境界として最小の検査を追加します。

Capability IDの文字列prefixだけからNamespace所属を推測しません。

### 実Reference

Reference Workspaceへ次を追加します。

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

Patternへ次を複製しません。

- `input order sales.order`
- `output sales.order`
- logical datasource
- Database Contract
- Custom Go等の実装方式
- Button / HTTP等のTarget固有表現

### Conformance

次をRepository内のテストとReferenceから確認します。

- `browse` / `record` / `reference` / `command`が同じ`order_entry`で共存する
- `command`から`sales.order.submit`をCanonical IDで追跡できる
- input / outputはSemantic Capabilityの定義から取得できる
- PatternがData / Query / Datasource / implementationを重複保持しない
- Pattern固有Diagnosticからauthoring sourceへ戻れる
- Reference WorldをRepository自身の`tei world`から再生成できる
- 既存`output sales.order`を不用意に名前付きOutcomeへ変更しない

## Capability Bindingとの境界

Application Patternは「どのCapabilityを呼ぶか」を表します。

一方、Capabilityをどの実装で実現するかはBinding / Realization / Driver側の責務です。

同じCapabilityが将来、builtin、generated、custom Go、process plugin、既存API等の異なる方式で実現されても、Application上のcommand identityを変えない構造を維持します。

このIssueではBindingをPatternのfieldとして持たせません。

## Business Result / Execution Errorとの境界

`command` Pattern自身は結果型を再定義しません。

現行`sales.order.submit`の`output sales.order`は、ADR-0020に従ってBusiness Resultとして扱います。

Commandを実行して初めて判明する複数の業務上の結末が本当に必要だと分かった場合は、GitHub Issue #77の圧力試験として名前付きOutcomeを検討します。このIssueだけを理由に、既存CapabilityをOutcome形式へ書き換えません。

Database接続失敗、adapter異常、process起動失敗等はExecution Errorであり、業務Outcomeへ変換しません。

## CommandとObservationの境界

Capability呼び出しによって返されたBusiness Resultと、その後に外部世界から観測したActual / Observationは別の意味です。

このIssueではApplication上のcommand意図までを扱い、ObservationをPatternのfieldにはしません。

両者の接続は後続Driverで扱います。

## 非目標

このIssueでは次を行いません。

- `header-detail` / `OrderLine`の実装
- Driverからの実Capability invocation
- production-grade transaction coordinator
- 名前付きOutcomeの強制導入
- Web Button、HTTP status、画面messageの設計
- Custom Go Bindingの具体的な実装方式の確定
- Workflow全般の抽象化

## 完了条件

次をReference / Conformanceから再現できることを完了条件とします。

1. `command` KindとCapability参照をCanonical model / Parser / Loader / Normalizationから扱える
2. `sales.order.submit`のinput / outputをPatternへ複製せずcommand意図を表現できる
3. Capabilityの存在とApplicationからの利用可能性を、既存境界を尊重して検査できる
4. Pattern Diagnosticが既存Subject / Source Index経路を利用できる
5. Capability Binding / implementationをPatternから分離したまま維持できる
6. 既存`output sales.order`をBusiness Resultとして保ち、Execution Errorと混同しない
7. `browse` / `record` / `reference` / `command`が`order_entry`で共存できる
8. Reference WorldをRepositoryだけから再生成できる
9. 後続DriverへCapability identityと結果境界をTarget非依存の情報として渡せる

## 関連

- GitHub Issue #67: v1 Application Pattern実証
- GitHub Issue #69: Application Driver
- GitHub Issue #77: Capabilityの業務Outcomeと実行エラー
- GitHub Issue #104: `record`実証
- GitHub Issue #107: `reference`実証
- ADR-0017: Application Pattern ownership
- ADR-0018: Application Pattern authoring source
- ADR-0020: Capability Business Outcome boundary
- RQ-001, RQ-003, RQ-008, RQ-011, RQ-014, RQ-017, RQ-018, RQ-021
