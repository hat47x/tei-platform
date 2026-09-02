# Gate A 到達性監査

## 目的

この文書は、v1 Reference Application `order_entry`をApplication Driverへ渡す前に、Gate A「Application上の意味をDriverへ渡せる」を横断的に確認した監査結果です。

Gate Aでは、個別のIssueが閉じたかどうかだけでは判定しません。DriverがWeb画面や専用Goコードから不足する意味を補わなくても、必要な情報をRepository上のReference / Conformanceから再現できるかを基準にします。

## 評価区分

| 区分 | 意味 |
| --- | --- |
| **Ready** | Gate Aに必要な範囲をReference / Conformanceから再現できる |
| **Partial** | 重要な境界は成立しているが、Driverへ渡す情報が不足している |
| **Blocked** | Driverへ進む前に独立した実証が必要である |
| **Deferred** | 重要な作業だがGate A以後でよい |

## 結論

**Gate AはReadyです。**

Application Driver #69の主実装へ進んでよいと判断します。

この判定は、Application Pattern、Datasource、Capability Binding、業務Validation、Capabilityの結果境界、既存認証Compatibility、Authorizationのすべてを、同じ`order_entry` Referenceから追跡できるようになったことに基づきます。

```text
Application Pattern / Datasource      Ready
Capability Binding                    Ready
Business Validation                   Ready
Business Result / Execution Error     Ready
Authentication                        Ready
Authorization                         Ready
Legacy authentication compatibility  Ready
                      ↓
                  Gate A Ready
                      ↓
                  Driver #69
```

## 項目別評価

| 項目 | 判定 | Repository上のEvidence |
| --- | --- | --- |
| browse対象 | Ready | `order_entry.recent_orders` |
| record対象 | Ready | `order_entry.order_record` |
| reference対象 | Ready | `sales.order.customer` + datasource `master` |
| header-detail関係 | Ready | `sales.order.lines -> sales.order_line many` |
| command対象 | Ready | `sales.order.submit` |
| Pattern identity / Source | Ready | Source Index / Diagnostic projection / rename invariance / Golden |
| Datasource read / write | Ready | `sales` read-write / `master` read |
| Business Resultの単一output | Ready | `sales.order.submit -> sales.order` |
| named Outcome / Execution Error | Ready | ADR-0020 / PR #132 / #141 |
| Authentication | Ready | `security.authenticate` / PR #145 / #148 / #151 |
| Authorization | Ready | `security.authorize` / PR #149 |
| 業務Validation | Ready | PR #136 / #139 / #140 |
| Capability Binding | Ready | PR #135 / #148 / #149 |
| Command / Observation分離 | Ready | 既存Runtime / DeploymentのConformance |
| Target固有Presentation | Deferred | 第一Web Target #70で実証する |

## Application Pattern — Ready

Issue #67は完了しています。

`order_entry`では、次の5種類のPatternが同じApplicationで共存します。

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

Patternは既存のSemantic、Query、Relation、Capabilityの意味を書き写す場所ではありません。Applicationが、既存identityをどの意図で利用するかを保持します。

5種類をTEI全体の永久的な標準一覧として固定したわけではありません。Driverと複数Targetから得られるEvidenceを見ながら、分類や所有境界は今後も見直せます。

## Datasource境界 — Ready

`order_entry`は次のlogical datasourceを利用します。

```text
sales  -> sales_core  -> read-write
master -> master_data -> read
```

Datasource accessはApplication定義を正本とします。PatternやDriver専用設定へ同じ情報を複製しません。

`reference` Patternでは`master`を明示しますが、read-onlyという属性自体はApplication側に残しています。

## Capability Binding — Ready

ApplicationとCapabilityを明示して、一つのCustom Go Bindingを解決できます。

```text
order_entry/sales.order.submit
order_entry/security.authenticate
order_entry/security.authorize
```

v1ではApplication + Capabilityの組をBinding identityとして扱います。

- 0件なら未定義としてerror
- 1件なら一意に解決
- 複数件ならambiguousとしてerror

Driverはfile path、Go symbol、列挙順からimplementationを推測しません。

またPR #151では、`order_entry/security.authenticate`という同じBinding identityとhook名のまま、既存record経路と既存function経路を交換できることを確認しました。実現方式を変えてもApplication上の意味は変わりません。

## 業務Validation — Ready

Referenceでは、実例から必要になった二つのCanonical ruleが成立しています。

```tei
validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
}

validation sales.order.lines.line_number_unique relation-unique {
    relation sales.order.lines
    attribute sales.order_line.line_number
}
```

最小Evaluatorから`Satisfied / NotSatisfied`を評価できます。

入力が不足するなど、評価そのものを完了できない状態は`NotSatisfied`へ畳まずerrorとして分離します。Driverはrule本文を専用Goコードへ複製せず、同じCanonical ValidationとEvaluator境界を利用します。

## Capability結果境界 — Ready

ADR-0020に従い、Capabilityを正常に実行して得られた業務上の結果と、Invocationそのものを完了できなかった状態を分離します。

```text
Business Result
  ├─ existing output
  └─ named Outcome

Execution Error
  └─ Invocation自体を完了できない状態
```

PR #141で、この区別を実行時の共通境界として確認しました。

`security.authenticate`では、credential不一致やaccount lockは`denied`という正常なBusiness Resultです。Databaseやadapterの障害、未定義Outcome、Capability identityの不一致はExecution Errorです。

一方、既存の`sales.order.submit`は`output sales.order`を維持しています。Authenticationのために既存Capabilityへ暗黙のOutcomeを追加していません。

## Authentication — Ready

Issue #68は完了しています。

ReferenceのCanonical契約は`security.authenticate`です。

```tei
capability security.authenticate {
    input login_id std.identifier
    input credential security.credential
    outcome authenticated security.authenticated_subject
    outcome denied security.authentication_denial
}
```

既存user recordを読む`legacyauth.Adapter`では、既存システム側の次の状態を失わず利用できます。

- user ID
- subject identity
- credential reference
- disabled
- validity period
- login failure count
- lock状態
- organization
- role
- system-specific rule

これらをTEI標準User schemaへ物理コピーしてから利用する構造にはしていません。

### Secretの境界

password、token、credential hash等のsecret materialをCanonical World、Source Index、Durable artifact、Reference fixture、通常Evidenceへ保存しません。

提示credentialは実行時だけVerifierまたは既存認証関数へ渡します。

### 第二実現方式

PR #151では、同じ`security.authenticate`と`order_entry/security.authenticate` Bindingを維持したまま、次の二方式を交換できることを確認しました。

1. 既存user recordを読み、Verifierとlegacy ruleを利用する方式
2. 既存関数・stored procedure・外部APIのreturn codeをadapterで対応付ける方式

legacy固有のreturn codeはTEI標準Outcomeにはしません。非secretなMapping sourceとして来歴を残し、`authenticated` / `denied`へadapterで対応付けます。未対応codeや関数呼び出し障害はExecution Errorです。

PR #151のCI #374では、Goldenを変更せずReference World再現性が成功しています。

## Authorization — Ready

PR #149で`security.authorize`を`order_entry`へ接続しました。

AuthorizationのresourceにはWeb URLや画面部品ではなく、既存Application Pattern identityを利用します。

- `browse` / `order_entry.recent_orders`
- `edit` / `order_entry.order_record`
- `execute` / `order_entry.submit_order`

Authenticationで得たsubject identityから既存user recordを引き直し、organization / role / legacy ruleを実装側で評価します。

ApplicationやDriverへ`ORDER_EDITOR`等のlegacy roleを公開してruleを再実装させません。

`allowed` / `denied`はBusiness Resultであり、Storeやruleの障害はExecution Errorです。

Application利用者のAuthorizationとPlugin / Runtime Permissionは引き続き別の責務として扱います。

## Driverへ渡せる情報

Gate Aの結果、Driver #69はRepositoryから少なくとも次を解決できます。

- browse / record / reference / header-detail / commandの対象
- logical datasourceとread / write境界
- 業務Validation identityとEvaluator境界
- `security.authenticate` / `security.authorize`
- authenticated subject identity
- Application上のaction / resource
- Capability Binding identityとauthoring source
- Business Result / Execution Errorの結果契約
- commandとして呼ぶ`sales.order.submit`
- CommandとObservationを分ける既存境界

これらをDriver専用設定やWeb固有コードから初めて知る必要はありません。

## Driver開始判定

**Application Driver #69は着手可能です。**

Driverでは、Gate Aで成立した意味を新しい正本へ書き直さず、既存のResolver、Evaluator、Capability Invocation境界を組み合わせます。

最初のDriverはproduction Runtimeの簡易版ではなく、Application設計をUIなしで実行し、意味の接続に不足がないかを確かめる設計検証環境として扱います。

## Gate A以後へDeferredするもの

次は重要ですが、Gate A通過の条件にはしません。

- named Outcomeの第一Web Target表現
- HTTP status / UI messageへのProjection
- production-ready IdP / Policy Engine
- production HA
- Target-neutral Presentationの最終抽象
- 5 Patternの共有Catalog化
- `OrderLine`の物理Persistence / FK一般化
- production transaction coordinator
- 汎用Validation式言語 / rule engine
- Database constraintやWeb form validatorへのValidation Projection
- Capability Payloadの汎用Object model
- 複数implementationを同時登録する汎用selection policy
- environment / release単位のimplementation切替

## 次の主線

Gate Aの監査は完了しました。

次は`ISSUE-000006` / GitHub Issue #69のApplication Driverへ進みます。

最初のDriver Scenarioでは、次を一続きに実行します。

1. browseで受注一覧を扱う
2. recordで一件を扱う
3. referenceで顧客を解決する
4. header-detailでOrderLineを扱う
5. 業務Validationを評価する
6. Authentication / Authorizationを評価する
7. commandで`sales.order.submit`を呼ぶ
8. Business ResultとExecution Errorを区別する
9. 必要な場合はObservationから外部状態を確認する

不足する意味が見つかった場合は、Driver内部へ閉じ込めず、所有すべき上位層へEvidenceとともに戻します。

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
- GitHub PR #135 / #136 / #139 / #140 / #141 / #145 / #148 / #149 / #151
- ADR-0019 / ADR-0020
