# Gate A 到達性監査

## 目的

この文書は、v1 Reference Application `order_entry`をApplication Driverへ渡す前に、Gate A「Application上の意味をDriverへ渡せる」の現在地を横断的に確認する監査結果です。

Application Pattern候補5種類はすべて実Referenceまで成立しました。しかし、Patternが揃ったことと、Driverを開始できることは同じではありません。

DriverがWeb画面や専用Goコードから意味を補わなくても、必要なApplication上の情報をRepositoryから解決できるかを基準に評価します。

## 評価区分

| 区分 | 意味 |
| --- | --- |
| **Ready** | Gate Aに必要な範囲をReference / Conformanceから再現できる |
| **Partial** | 重要な境界は成立しているが、Driverへ渡す情報が不足している |
| **Blocked** | Driverへ進む前に独立した実証が必要である |
| **Deferred** | 必要な作業だがGate A以後でよい |

## 結論

**Gate A全体の現在判定はBlockedです。**

Application PatternとCapability BindingはReadyです。

さらに、Capability OutcomeはPR #132で名前付きOutcomeのCanonical基盤まで、業務ValidationはPR #136でSemantic上の最小Canonical契約まで実装されました。このため、両者とも「正本の表現方法がまだない」という段階は通過しています。

一方、Authentication / Authorization、Capability Outcomeの実行時境界、業務Validationの実ReferenceとEvaluatorがDriver開始条件をまだ満たしていません。

現時点でDriver #69の主実装へ進むと、不足している意味や評価規則をDriver専用コードへ暗黙に埋め込むおそれがあります。

したがって、現在は次の3作業をGate Aの主線とします。

```text
#68   既存認証Compatibility ───────┐
#77   Capability Outcome ─────────┤
#125  業務Validation ─────────────┤
                                   ↓
                               Gate A再監査
                                   ↓
                               Driver #69
```

Capability Binding #126はPR #135で実証を終え、Gate AのReady項目へ移しました。

## Gate A項目別評価

| 項目 | 判定 | 現在のEvidence | 残る作業 |
| --- | --- | --- | --- |
| browse対象 | Ready | `order_entry.recent_orders` | なし |
| record対象 | Ready | `order_entry.order_record` | なし |
| reference対象 | Ready | `sales.order.customer` + datasource `master` | なし |
| header-detail関係 | Ready | `sales.order.lines -> sales.order_line many` | なし |
| command対象 | Ready | `sales.order.submit` | なし |
| Pattern identity / Source | Ready | Source Index / rename invariance / Golden | なし |
| Datasource read / write | Ready | `sales` read-write / `master` read、reference read検査 | Driverでも同じ正本を利用する |
| Business Resultの単一output | Ready | `sales.order.submit -> sales.order` | named Outcomeとは別に維持する |
| named OutcomeのCanonical契約 | Partial | PR #132でmodel / Parser / Validation / Normalizationを実装 | #77でReference / Invocation境界へ接続する |
| Authentication | Blocked | ADR-0019 / #68の設計のみ | #68 |
| Authorization | Blocked | 境界設計はあるがReferenceなし | #68 |
| 業務Validation | Partial | PR #136でSemantic ruleのCanonical model / Parser / Source Index /参照整合性を実装 | #125で実ReferenceとEvaluatorを成立させる |
| Capability Binding | Ready | PR #135で`sales.order.submit`の明示Bindingと解決APIを実証 | Driverで同じ解決入口を利用する |
| Command / Observation分離 | Ready | 既存Runtime / DeploymentのEvidence。Patternにも混在させていない | Driverで再利用する |
| Target固有Presentation | Deferred | Gate Aでは不要 | 第一Web Target #70 |

## Application Pattern — Ready

Issue #67は完了しました。

`order_entry`では次の5種類が同じApplicationで共存します。

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}

pattern order_entry.order_record record {
    data sales.order
}

pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}

pattern order_entry.submit_order command {
    capability sales.order.submit
}

pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

Pattern identity、Source Index、共通Validation、Golden再現性までRepository内で確認済みです。

5種類の実証から、Patternは既存資産の意味を書き直す場所ではなく、Applicationがどの既存identityをどの意図で利用するかを保持する層として整理できました。

## Datasource read / write — Ready

`order_entry`は二つのlogical datasourceを利用します。

```text
sales  -> sales_core  -> read-write
master -> master_data -> read
```

`reference` Patternでは、Datasource accessをPatternへ複製せずApplication定義を正本としながら、読み取り不能なDatasourceをValidationで拒否できます。

Gate Aでは、この境界を新しいDriver設定へコピーせず、Applicationの`DatabaseRequirement`から解決できることを条件とします。

## Capability結果境界 — Partial

ADR-0020により、意味上の境界は次の形で整理されています。

```text
Business Result
  ├─ existing output
  └─ named Outcome

Execution Error
  └─ invocation自体を完了できない状態
```

`sales.order.submit`については、既存`output sales.order`を維持するEvidenceがあります。

さらにPR #132で、Semantic Capabilityが名前付きOutcomeを持つための最小Canonical基盤を実装しました。

- Outcome local name
- payload typeへのCanonical参照
- `outcome <name> <payload-type>` Parser
- 既存`output`との排他Validation
- Outcome名の重複検査
- payload typeの参照整合性
- Outcome宣言順に依存しないNormalization

このため「named OutcomeをSemantic Capabilityで表現できない」というblockerは解消しました。

ただしGate Aに必要な実行時境界はまだ完了していません。#77では少なくとも次を確認する必要があります。

- `security.authenticate`等の実Referenceでnamed Outcomeを利用できる
- `authenticated` / `denied`を正常なBusiness Resultとして扱える
- expected denialをExecution Errorへ変換しない
- Binding / adapter / Database障害をnamed Outcomeへ押し込まない
- DriverへBusiness ResultとExecution Errorの区別を渡せる

第一Web TargetへのProjection等はGate A以後へDeferredできます。

## Authentication / Authorization — Blocked

Reference Workspaceには、Authentication / Authorizationを実証する`security` Referenceがまだ成立していません。

Issue #68では、既存認証仕様をTEI標準User schemaへ作り直さず、Authentication / Authorization Capabilityへ接続することを求めています。

Gate Aに必要なのはproduction-readyなIdPやログイン画面ではありません。少なくとも次が必要です。

- authenticated subject identity
- Authentication Capability identity
- Authorization Capability identity
- Application上のaction / resourceとの接続
- legacy field / functionからCapabilityへのMapping
- secret materialをCanonicalへ保存しない境界
- named Outcome / Execution Errorの結果契約

このため#68は引き続きGate Aのblockerです。

## 業務Validation — Partial

PR #136で、業務ValidationをSemantic上のCanonical ruleとして保持する最小契約が成立しました。

最初の二例は次です。

```tei
validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
}
```

```tei
validation sales.order.lines.line_number_unique relation-unique {
    relation sales.order.lines
    attribute sales.order_line.line_number
}
```

この二つは、特定のWeb画面だけの入力都合ではなく、受注明細という業務意味そのものに属すると判断し、`sales` Semantic Namespaceを正本としました。

PR #136では次まで成立しています。

- Semantic ValidationのCanonical model
- `positive` / `relation-unique`の型付きrule kind
- Parser / Loader
- Canonical ID順のNormalization
- Source Indexへの接続
- `positive`対象Attributeの数値型検査
- `relation-unique`対象Relationが`many`であることの検査
- Relation target Dataに属するAttributeだけを許可する検査

したがって、「業務ruleの正本とidentityが存在しない」というblockerは解消しました。

一方、Reference Workspace本体にはこの二つの実ruleがまだ置かれておらず、Canonical ruleを実際の入力に対して評価するEvaluatorもありません。

Gate AでReadyへ進むには、#125で少なくとも次を確認する必要があります。

- 二つのruleを実Referenceから読み込める
- Validation identityからauthoring sourceへ戻れる
- `quantity > 0`をSatisfied / NotSatisfiedの両方で評価できる
- 一つのOrderに属する明細集合で`lineNumber`一意性をSatisfied / NotSatisfiedの両方で評価できる
- 評価不能をNotSatisfiedへ畳まない
- Driverがrule本文を専用Goコードへ複製せず、Canonical Validationを利用できる

大きな式言語やrule engineを先に作ることは、引き続き目的にしません。

## Capability Binding — Ready

Issue #126 / PR #135で、`sales.order.submit`の明示的なCustom Go BindingをReference化しました。

v1では新しい任意のBinding IDを追加せず、ApplicationとCapabilityの組をBinding identityとして扱います。

```text
order_entry / sales.order.submit
```

Driver相当の利用側は、ApplicationとCapabilityを明示してBindingを解決できます。

- 0件: 未定義としてerror
- 1件: 一意に解決
- 複数件: ambiguousとしてerror

file path、Go hook名、列挙順から実装を推測しません。

Source Indexは`binding:<application>/<capability>`からauthoring sourceへ戻ることができ、file renameでもBinding identityが変わらないことを確認しました。

同じCapabilityを異なるApplicationがBindingしても衝突せず、同じApplication + Capabilityの重複Bindingは拒否します。

この実証では第一級Binding IDは必要になりませんでした。複数implementationの同時選択やenvironment / release単位の切替が実例として必要になった時点で再評価します。

Binding解決失敗はBusiness Resultではなく、実行準備またはCapability invocationを完了できない側のerrorとして扱います。

CI #309は`verify`とPostgreSQL integrationの双方が成功しました。

## Driver開始判定

現時点ではDriver #69の主実装を開始しません。

次の必須項目がReadyになった時点でGate Aを再監査します。

1. #68 — Authentication / Authorization Reference
2. #77 — Gate Aに必要なnamed Outcome / Execution Errorの実行境界
3. #125 — 業務Validationの実ReferenceとEvaluator

Capability Binding #126はReadyです。

Datasource read / write、5 Pattern、単一output Business Result、Source Index / identityも既にReadyです。

## Gate A以後へDeferredするもの

次は重要ですが、Gate Aのblockerにはしません。

- named Outcomeの第一Web Target表現
- HTTP status / UI messageへのProjection
- production-ready IdP
- production HA
- Target-neutral Presentationの最終抽象
- 5 Patternの共有Catalog化
- `OrderLine`の物理Persistence / FKを一般化すること
- production transaction coordinator
- 汎用Validation式言語 / rule engine
- Database constraintやWeb form validatorへのValidation Projection

## 次回監査

#68 / #77 / #125のEvidenceが増えたら、この表を更新します。

すべてを一度にReadyへ寄せず、項目ごとにEvidenceの強さを確認します。

Gate Aを通過した時点で、`ISSUE-000006` / GitHub Issue #69を主実装へ移し、最小Driver Scenarioを開始します。

## 関連

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/APPLICATION_PATTERN_V1_DESIGN.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000009-capability-outcome-and-execution-error.md`
- `plan/issue/000013-gate-a-readiness-audit.md`
- `plan/issue/000014-business-validation-reference.md`
- `plan/issue/000015-submit-capability-binding-reference.md`
- GitHub Issue #67 / #68 / #69 / #77 / #124 / #125 / #126
- GitHub PR #132 / #135 / #136
- ADR-0019 / ADR-0020
