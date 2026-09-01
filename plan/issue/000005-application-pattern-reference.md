# ISSUE-000005: `order_entry`でApplication Patternを実証する

- 状態: In Progress
- GitHub Issue: #67
- 設計・実装基線: [`APPLICATION_PATTERN_V1_DESIGN.md`](../APPLICATION_PATTERN_V1_DESIGN.md)
- v1全体計画: [`V1_REFERENCE_APPLICATION_PLAN.md`](../V1_REFERENCE_APPLICATION_PLAN.md)

## 目的

既存のReference Application `order_entry`を使い、Applicationが持つ業務上の意図をTarget固有UIから切り離して表現できることを実証します。

最初の候補は次の5種類です。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

この5分類をTEI全体の永久的な標準語彙として先に固定しません。各Kindは、実際のReferenceで必要な意味を確認してから一つずつ有効化します。

## 現在地

### `browse` — 完了

`order_entry.recent_orders`を実Referenceとして追加し、次を確認しました。

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}
```

- Canonical model / Parser
- Loader / Normalization
- Source Index / Diagnostic projection
- Data / Query / Datasourceの参照整合性Validation
- 共通`validation.Validate`への統合
- Golden World再生成
- 実Referenceでのfile rename invariance

Queryが`datasource sales`を正本として持つため、Patternへ同じDatasourceを重複記述していません。

### `record` — 完了

GitHub Issue #104、PR #105 / #106で次を実証しました。

```tei
pattern order_entry.order_record record {
    data sales.order
}
```

`record`は「ApplicationがどのSemantic Dataを一件として扱うか」という意味だけで成立しました。

Query、Datasource、特定`browse`との遷移関係は、現時点ではCanonical fieldへ追加していません。

### `reference` — 完了

GitHub Issue #107、PR #108 / #109で次を実証しました。

```tei
pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}
```

Relationがすでにsource / target / cardinalityを持つため、それらをPatternへ重複記述していません。

Datasource accessはApplicationのDatabaseRequirementを正本とし、`reference`ではread可能であることをValidationします。read-writeも読み取り可能なので受理し、read-onlyだけを要求する一般則にはしていません。

### `command` — 次の実証対象

Repository内の作業定義は[`ISSUE-000010`](000010-command-application-pattern.md)、GitHub Issueは#110です。

最小構文の第一候補は次です。

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

Capabilityがすでに保持しているinput / output、Datasource、implementationをPatternへ複製しない方針から検証します。

CapabilityのBusiness Result / Execution Error境界はADR-0020 / GitHub Issue #77を利用し、`command`追加だけを理由に既存`output sales.order`を名前付きOutcomeへ変更しません。

### `header-detail` — `command`後に実証

`header-detail`では、既存Referenceにない`OrderLine`と対応Semantic Relationを先に追加します。

```text
Order 1 ── N OrderLine
```

Semantic Relation、Application Pattern、Database Foreign Key、Target固有UIを同一視しません。

## ここまでに確認できた設計判断

- PatternはApplicationが所有する
- Pattern identityはfile pathから導出しない
- Patternの正本は`applications/*/patterns/*.tei`に置く
- Semantic Data / Relation / Query / Capabilityの定義内容をPatternへ複製しない
- 同じ意味を複数箇所へ手書きしない
- Pattern Kindごとに必要なfieldは異なり得る
- `browse`のfield構造を共通テンプレートとして一般化しない
- Workspace全体でしか判断できない参照整合性は共通Validationで扱う
- Diagnosticは`application-pattern:<Canonical ID>` Subjectからauthoring sourceへ戻せる
- PatternをWeb画面やWidgetの分類として固定しない

## 現在の実証順

```text
browse                完了
  ↓
record                完了
  ↓
reference             完了
  ↓
command               ISSUE-000010 / #110
  ↓
OrderLine + Relation
  ↓
header-detail
  ↓
Stage 1 Evidence監査
  ↓
既存認証CompatibilityとGate Aで合流
```

既存認証Compatibilityは`ISSUE-000003` / GitHub Issue #68として並行して進めます。

## Gate Aへ渡す情報

Application Pattern側から少なくとも次をTarget非依存の情報として説明できる状態を目指します。

- 何を`browse`するか
- 何を一つの`record`として扱うか
- どのRelationをどのDatasourceから`reference`するか
- Header / DetailをどのApplication上の関係として扱うか
- どのCapabilityを`command`として呼ぶか
- Application Validation
- read / read-write Datasource境界
- PatternのCanonical identityとauthoring source

Authentication / Authorizationは#68から合流します。

CapabilityのBusiness Result / Execution Error境界はADR-0020 / #77から受け取ります。

## 完了条件

次をRepository内のReference / Conformanceから再現できることを完了条件とします。

1. v1業務シナリオに必要なPattern候補を同じ`order_entry`から識別できる
2. Patternのidentityがfile pathに依存しない
3. Semantic / Query / Relation / Capabilityの正本をPatternへ複製していない
4. native SQLを別DSLへ書き直していない
5. read-only Datasourceの境界を破っていない
6. `command`がCapability identityへ接続されている
7. Semantic Relation、Application Pattern、Persistence Mappingの境界を説明できる
8. Source Index / DiagnosticからPatternのauthoring sourceへ戻れる
9. Reference WorldをRepositoryだけから再生成できる
10. 同じApplication上の意味を後続Driver #69へ渡せる

現時点では`browse`、`record`、`reference`まで完了していますが、`command`と`header-detail`が残っています。そのため、Stage 1、Gate A、Milestone 1の完了は宣言しません。

## 関連

- GitHub Issue #67: v1 Application Pattern
- GitHub Issue #68: 既存認証Compatibility
- GitHub Issue #69: Application Driver
- GitHub Issue #77: Capability Outcome / Execution Error
- GitHub Issue #104: `record`実証
- GitHub Issue #107: `reference`実証
- GitHub Issue #110: `command`実証
- ADR-0017: Application Pattern ownership
- ADR-0018: Application Pattern authoring source
- ADR-0020: Capability Business Outcome boundary
- RQ-001, RQ-003, RQ-005, RQ-008, RQ-011, RQ-012, RQ-014, RQ-017, RQ-018, RQ-021
