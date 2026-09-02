# Gate A 到達性監査

## 目的

この文書は、v1 Reference Application `order_entry`をApplication Driverへ渡す前に、Gate A「Application上の意味をDriverへ渡せる」の現在地を横断的に確認する監査結果です。

Application Patternが揃ったことだけでは、Driverを開始できるとは判断しません。DriverがWeb画面や専用Goコードから不足する意味を補わなくても、必要な情報をRepository上のReference / Conformanceから再現できるかを基準にします。

## 評価区分

| 区分 | 意味 |
| --- | --- |
| **Ready** | Gate Aに必要な範囲をReference / Conformanceから再現できる |
| **Partial** | 重要な境界は成立しているが、Driverへ渡す情報が不足している |
| **Blocked** | Driverへ進む前に独立した実証が必要である |
| **Deferred** | 必要な作業だがGate A以後でよい |

## 結論

**Gate A全体の現在判定はBlockedです。**

ただし、業務ValidationはPR #136、#139、#140を通じてGate Aに必要なEvidenceが揃い、`Ready`へ進みました。Issue #125も完了しています。

現在の独立した残条件は、次の2領域です。

1. #68 — 既存認証Compatibility
2. #77 — Capabilityのnamed OutcomeとExecution Errorを実行時に分離するEvidence

両者はAuthentication Referenceで大きく重なります。次の主線では、`security.authenticate`の実Referenceを用い、既存認証仕様を保持したまま`authenticated` / `denied`を正常なBusiness Resultとして扱い、Databaseやadapterの障害をExecution Errorとして分離できることを確認します。

```text
#68  既存認証Compatibility ───────┐
                                    ├─ Authentication Reference
#77  Capability Outcome ──────────┘
                                             ↓
                                         Gate A再監査
                                             ↓
                                         Driver #69
```

## Gate A項目別評価

| 項目 | 判定 | 現在のEvidence | 残る作業 |
| --- | --- | --- | --- |
| browse対象 | Ready | `order_entry.recent_orders` | なし |
| record対象 | Ready | `order_entry.order_record` | なし |
| reference対象 | Ready | `sales.order.customer` + datasource `master` | なし |
| header-detail関係 | Ready | `sales.order.lines -> sales.order_line many` | なし |
| command対象 | Ready | `sales.order.submit` | なし |
| Pattern identity / Source | Ready | Source Index / rename invariance / Golden | なし |
| Datasource read / write | Ready | `sales` read-write / `master` read | Driverでも同じ正本を利用する |
| Business Resultの単一output | Ready | `sales.order.submit -> sales.order` | named Outcomeとは別に維持する |
| named OutcomeのCanonical契約 | Partial | PR #132でmodel / Parser / Validation / Normalizationを実装 | #77で実ReferenceとInvocation境界へ接続する |
| Authentication | Blocked | ADR-0019 / #68の設計 | #68で実Referenceを成立させる |
| Authorization | Blocked | 意味境界は整理済みだが実Referenceなし | #68でApplication上のaction / resourceへ接続する |
| 業務Validation | Ready | PR #136 / #139 / #140。Canonical rule、実Reference、Evaluatorを実証 | Driverで同じEvaluator境界を利用する |
| Capability Binding | Ready | PR #135で`sales.order.submit`の明示Bindingと解決APIを実証 | Driverで同じ解決入口を利用する |
| Command / Observation分離 | Ready | 既存Runtime / DeploymentのEvidence | Driverで再利用する |
| Target固有Presentation | Deferred | Gate Aでは不要 | 第一Web Target #70 |

## Application Pattern — Ready

Issue #67は完了しています。

`order_entry`では、`browse`、`record`、`reference`、`command`、`header-detail`の5種類が同じApplicationで共存します。Pattern identity、Source Index、共通Validation、Golden再現性までRepository内で確認済みです。

Patternは既存資産の意味を書き直す場所ではなく、Applicationがどの既存identityを、どの意図で利用するかを保持する層として扱います。

## Datasource境界 — Ready

`order_entry`は、次の二つのlogical datasourceを利用します。

```text
sales  -> sales_core  -> read-write
master -> master_data -> read
```

Datasource accessをPatternへ複製せず、Application定義を正本とします。Driverも同じ`DatabaseRequirement`を利用し、独自設定へ書き写しません。

## Capability Binding — Ready

Issue #126 / PR #135で、`sales.order.submit`をApplication + Capabilityから明示的に解決できるReferenceが成立しました。

```text
order_entry / sales.order.submit
```

Driver相当の利用側は、file path、Go hook名、列挙順からimplementationを推測しません。

- 0件なら未定義としてerror
- 1件なら一意に解決
- 複数件ならambiguousとしてerror

Binding解決失敗はBusiness Resultではなく、実行準備またはInvocationを完了できない側のerrorとして扱います。

## 業務Validation — Ready

業務ValidationはIssue #125でGate Aに必要な範囲を完了しました。

### Canonical契約 — PR #136

`sales` Semantic Namespaceに、最初の実例から必要になった2種類のruleだけを導入しました。

```tei
validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
}

validation sales.order.lines.line_number_unique relation-unique {
    relation sales.order.lines
    attribute sales.order_line.line_number
}
```

汎用式言語やrule engineは導入していません。

### 実Reference — PR #139

二つのruleをReference Workspaceへ置き、Canonical ID、kind、参照先、Source Index、Golden Worldを再現できることを確認しました。

Application側へValidation ID一覧は追加していません。今回の二つは特定画面だけの入力都合ではなく、受注明細そのものの業務意味に属するため、`sales` Semantic Namespaceを正本としています。

### Evaluator — PR #140

同じCanonical ruleを実際の値へ適用する最小Evaluatorが成立しました。

- `quantity > 0`を`Satisfied / NotSatisfied`で評価できる
- 同一Order内の`lineNumber`一意性を`Satisfied / NotSatisfied`で評価できる
- 必要な入力が欠けるなど、評価そのものを完了できない状態は`NotSatisfied`へ畳まずerrorとして分離する
- EvaluatorはApplication、Database、Web固有のデータ構造を正本にせず、小さな`Source`境界から必要な値だけを受け取る

CI #337は`verify`とPostgreSQL integrationの双方が成功しました。

したがってDriverは、rule本文を専用Goコードへ複製せず、Canonical ValidationとEvaluator境界を利用できます。

## Capability結果境界 — Partial

ADR-0020により、意味上の境界は次のように整理されています。

```text
Business Result
  ├─ existing output
  └─ named Outcome

Execution Error
  └─ Invocation自体を完了できない状態
```

既存の`sales.order.submit`は`output sales.order`を維持します。

PR #132では、Semantic Capabilityが名前付きOutcomeを持つためのCanonical基盤まで成立しました。

- Outcome local name
- payload typeへのCanonical参照
- `outcome <name> <payload-type>` Parser
- `output`との排他Validation
- Outcome名の重複検査
- payload typeの参照整合性
- 宣言順に依存しないNormalization

Gate Aでまだ必要なのは実行時のEvidenceです。#77では、少なくとも次を確認します。

- `security.authenticate`等の実Referenceでnamed Outcomeを利用できる
- `authenticated` / `denied`を正常なBusiness Resultとして扱える
- expected denialをExecution Errorへ変換しない
- Binding / adapter / Database障害をnamed Outcomeへ押し込まない
- DriverへBusiness Result / Execution Errorの区別を渡せる

第一Web Targetへの表現はGate A以後へDeferredできます。

## Authentication / Authorization — Blocked

Reference Workspaceには、Gate Aで必要なAuthentication / Authorizationの実証がまだ揃っていません。

Issue #68では、既存認証仕様をTEI標準User schemaへ作り直すことを前提にせず、既存のfield、function、ruleをAuthentication / Authorization Capabilityへ接続します。

Gate Aに必要なのはproduction-readyなIdPやログイン画面ではありません。少なくとも次をRepositoryから追跡できる必要があります。

- authenticated subject identity
- Authentication Capability identity
- Authorization Capability identity
- Application上のaction / resource
- legacy field / functionからCapabilityへのMapping
- secret materialをCanonicalや通常Evidenceへ保存しない境界
- named Outcome / Execution Errorの結果契約

#68と#77はこの領域で合流します。Authentication一例のためだけの例外体系を作らず、Capability共通の結果境界として実証します。

## Driver開始判定

現時点ではDriver #69の主実装を開始しません。

Gate Aで残る必須項目は次の2つです。

1. #68 — Authentication / Authorization Reference
2. #77 — named Outcome / Execution Errorの実行時境界

業務Validation #125とCapability Binding #126はReadyです。5種類のPattern、Datasource境界、単一output、Source Index / identity、Command / Observation分離もReadyです。

#68と#77のGate A必須範囲がReference / Conformanceから再現できた時点で再監査し、Driver #69を`Ready / In Progress`へ移せるかを判断します。

## Gate A以後へDeferredするもの

次は重要ですが、Gate Aのblockerにはしません。

- named Outcomeの第一Web Target表現
- HTTP status / UI messageへのProjection
- production-ready IdP
- production HA
- Target-neutral Presentationの最終抽象
- 5 Patternの共有Catalog化
- `OrderLine`の物理Persistence / FK一般化
- production transaction coordinator
- 汎用Validation式言語 / rule engine
- Database constraintやWeb form validatorへのValidation Projection

## 次の主線

次は#68と#77を別々に大きく実装するのではなく、共通のAuthentication Referenceから始めます。

1. `security.authenticate`のSemantic Referenceを成立させる
2. `authenticated` / `denied`をnamed Outcomeとして読み込めることを確認する
3. 正常なBusiness ResultとExecution ErrorをInvocation境界で分離する
4. legacy認証fixtureとCapabilityを接続する
5. Authorizationを`order_entry`のaction / resourceへ接続する
6. Gate Aを再監査する

この順序により、認証専用の場当たり的な結果体系や、Driver専用の業務意味を作ることを避けます。

## 関連

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/LEGACY_AUTH_V1_DESIGN.md`
- `plan/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000009-capability-outcome-and-execution-error.md`
- `plan/issue/000013-gate-a-readiness-audit.md`
- `plan/issue/000014-business-validation-reference.md`
- `plan/issue/000015-submit-capability-binding-reference.md`
- GitHub Issue #67 / #68 / #69 / #77 / #124 / #125 / #126
- GitHub PR #132 / #135 / #136 / #139 / #140
- ADR-0019 / ADR-0020
