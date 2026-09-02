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

ただし、独立した残条件は**Issue #68「既存認証Compatibility」だけ**になりました。

Issue #77「Capability Outcome / Execution Error」はPR #141でGate Aに必要な実行時Evidenceが成立し、完了しています。`security.authenticate`の`authenticated` / `denied`を正常なBusiness Resultとして扱い、Databaseやadapterの障害、未定義Outcome等をExecution Errorとして分離できます。

業務Validation #125とCapability Binding #126もすでにReadyです。

現在のGate Aは、次の形です。

```text
Application Pattern / Datasource      Ready
Capability Binding                    Ready
Business Validation                   Ready
Business Result / Execution Error     Ready
Authentication contract               Partial
Legacy authentication compatibility  Blocked
Authorization                         Blocked
                      ↓
                    #68
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
| named Outcome / Execution Error | Ready | PR #132 / #141。Canonical契約、実Reference、Invocation境界を実証 | Driverで同じInvocation境界を利用する |
| Authentication契約 | Partial | `security.authenticate`と`authenticated` / `denied`の実Reference | #68でlegacy実装へ接続する |
| Authorization | Blocked | ADR-0019で意味境界を整理済み | #68でApplication上のaction / resourceへ接続する |
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

Issue #125はPR #136、#139、#140でGate Aに必要な範囲を完了しました。

`sales` Semantic Namespaceに、実例から必要になった二つのruleを保持しています。

```tei
validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
}

validation sales.order.lines.line_number_unique relation-unique {
    relation sales.order.lines
    attribute sales.order_line.line_number
}
```

同じCanonical ruleを最小Evaluatorへ渡し、`Satisfied / NotSatisfied`を評価できます。必要な入力が欠けるなど、評価そのものを完了できない状態は`NotSatisfied`へ畳まずerrorとして分離します。

Driverはrule本文を専用Goコードへ複製せず、このCanonical ValidationとEvaluator境界を利用します。

## Capability結果境界 — Ready

ADR-0020に従い、Capabilityの正常な業務結果と、Invocationそのものを完了できなかった状態を分離します。

```text
Business Result
  ├─ existing output
  └─ named Outcome

Execution Error
  └─ Invocation自体を完了できない状態
```

PR #132で名前付きOutcomeのCanonical契約を実装し、PR #141で実行時境界まで実証しました。

Authentication Referenceでは次を区別できます。

- 正しいcredential → `authenticated` Business Result
- credential不一致 → `denied` Business Result
- account lock → `denied` Business Result
- Database / adapter障害 → Execution Error
- 未定義Outcome → Execution Error

また、既存の`sales.order.submit`は従来どおり`output sales.order`を維持し、同じInvocation入口から単一outputのBusiness Resultを扱えます。Authenticationのために、既存Capabilityへ暗黙のOutcomeを追加していません。

Payloadの共通Go表現はまだ固定していません。Driverより先に汎用的な業務データObject modelを作らず、複数の実Referenceから必要性が確認された時点で設計します。

第一Web TargetへのHTTP statusやUI messageへのProjectionはGate A以後へDeferredします。

## Authentication / Authorization — 残るBlocker

`security.authenticate`のSemantic契約と結果境界は成立しましたが、Issue #68の本題である**既存認証仕様を保持したまま実装へ接続するEvidence**はまだ不足しています。

Gate Aで残るのは、少なくとも次です。

- 既存のuser ID、disabled、validity period、failure count、lock、organization、role、system-specific ruleを保持したfixture
- 既存schemaや既存関数をTEI標準User schemaへ変換せずAuthentication Capabilityへ接続するadapter / Binding
- credentialやtoken等のsecret materialをCanonical World、Source Index、通常Evidenceへ保存しない境界
- authenticated subject identity
- `security.authorize`等のAuthorization Capability
- `order_entry`上のaction / resourceとの接続
- Driverがlegacy schemaの物理構造へ直接依存しないこと

`security.credential`の型定義はsecret値そのものをCanonicalへ保存するものではありません。実行時に渡されるcredential valueは機微情報として扱い、Scenario logや通常Evidenceへ記録しないことを#68のReferenceで明示的に確認します。

## Driver開始判定

現時点ではDriver #69の主実装を開始しません。

Gate Aで残る独立blockerは#68だけです。#68のGate A必須範囲がReference / Conformanceから再現できた時点で再監査し、Driver #69を`Ready / In Progress`へ移せるかを判断します。

Issue全体のclose状態だけでは判定しません。Driverへ必要な意味をRepositoryから再現できることを条件とします。

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
- Capability Payloadの汎用Object model

## 次の主線

次はIssue #68を、AuthenticationとAuthorizationに分けて小さく進めます。

1. legacy認証fixtureを、既存仕様を崩さない形で用意する
2. `security.authenticate`をlegacy fixtureへ接続する最小adapter / Bindingを実証する
3. `authenticated` / `denied` / Execution Errorの既存境界をそのまま利用する
4. secret materialがCanonical / Source Index / 通常Evidenceへ流れないことを確認する
5. `security.authorize`の最小契約を実例から作る
6. `order_entry`のbrowse / edit / submitへaction / resourceを接続する
7. Gate Aを再監査する

Authentication一例から巨大な認証基盤や標準User schemaを設計せず、既存システムの意味を保持できる最小Evidenceを積み上げます。

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
- GitHub Issue #68 / #69 / #77 / #124 / #125 / #126
- GitHub PR #132 / #135 / #136 / #139 / #140 / #141 / #143
- ADR-0019 / ADR-0020
