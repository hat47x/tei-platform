# Gate A 到達性監査

## 目的

この文書は、v1 Reference Application `order_entry`をApplication Driverへ渡す前に、Gate A「Application上の意味をDriverへ渡せる」を横断的に確認した監査結果です。

Gate Aは、関連Issueの完了数では判定しません。DriverがWeb画面や専用Goコードから不足する意味を補わなくても、必要な情報をRepository上のReference / Conformanceから再現できるかを基準にします。

## 評価区分

| 区分 | 意味 |
| --- | --- |
| **Ready** | Gate Aに必要な範囲をReference / Conformanceから再現できる |
| **Partial** | 重要な境界は成立しているが、Driverへ渡す情報が不足している |
| **Blocked** | Driverへ進む前に独立した実証が必要である |
| **Deferred** | 必要な作業だがGate A以後でよい |

## 結論

**Gate AはReadyです。Application Driver #69の主実装を開始できます。**

最後まで残っていたIssue #68「既存認証Compatibility」について、PR #148、#149、#151でGate Aに必要なEvidenceが揃いました。

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

この判定は、#68を閉じること自体を根拠にはしていません。認証・認可を含め、Driverへ必要な意味をRepositoryから再現できることを確認した結果です。

## Gate A項目別評価

| 項目 | 判定 | Evidence | Driverへ渡す境界 |
| --- | --- | --- | --- |
| browse対象 | Ready | `order_entry.recent_orders` | Patternを正本として利用する |
| record対象 | Ready | `order_entry.order_record` | Patternを正本として利用する |
| reference対象 | Ready | `sales.order.customer` + datasource `master` | Relationとlogical datasourceを利用する |
| header-detail関係 | Ready | `sales.order.lines -> sales.order_line many` | Semantic Relationを利用する |
| command対象 | Ready | `order_entry.submit_order -> sales.order.submit` | Command PatternからCapabilityを解決する |
| Pattern identity / Source | Ready | Source Index / rename invariance / Golden | file pathをidentityにしない |
| Datasource read / write | Ready | `sales` read-write / `master` read | `DatabaseRequirement`を正本として利用する |
| Business Resultの単一output | Ready | `sales.order.submit -> sales.order` | 既存output契約を維持する |
| named Outcome / Execution Error | Ready | #77 / PR #132 / #141 | `capabilityinvoke`境界を利用する |
| Authentication | Ready | #68 / PR #148 / #151 | `security.authenticate`と明示Bindingを利用する |
| Authorization | Ready | #68 / PR #149 | `security.authorize`とApplication Pattern identityを利用する |
| legacy認証Compatibility | Ready | PR #148 / #149 / #151 | legacy schema / functionをadapterの内側へ保つ |
| 業務Validation | Ready | #125 / PR #136 / #139 / #140 | Canonical ruleとEvaluatorを利用する |
| Capability Binding | Ready | #126 / PR #135、認証はPR #148 / #149 | Application + Capabilityから明示的に解決する |
| Command / Observation分離 | Ready | 既存Runtime / Deployment Evidence | command応答と外部状態を別の問いとして扱う |
| Target固有Presentation | Deferred | Gate Aでは不要 | 第一Web Target #70で扱う |

## Application Pattern — Ready

Issue #67で、`order_entry`に次の5種類のApplication Patternが実Referenceとして成立しています。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

Patternは既存資産の意味を書き直す場所ではなく、ApplicationがどのCanonical identityを、どの意図で利用するかを保持する層として扱います。

Pattern identityはfile pathに依存せず、Source Indexからauthoring sourceへ戻れます。5種類を永久的な標準Catalogへ昇格する判断は、Driverと複数TargetからEvidenceを得るまで保留します。

## Datasource境界 — Ready

`order_entry`は、次の二つのlogical datasourceを利用します。

```text
sales  -> sales_core  -> read-write
master -> master_data -> read
```

Datasource accessをPatternへ複製せず、Application定義を正本とします。Driverも同じ`DatabaseRequirement`を利用し、独自設定へ書き写しません。

認証情報の保存先は、この二つへ無理に組み込みません。既存認証がDatabase、LDAP、IdP、既存関数、外部API等のどこに存在しても、DriverからはAuthentication / Authorization Capabilityを通して利用します。

## Capability Binding — Ready

Issue #126 / PR #135で、`sales.order.submit`をApplication + Capabilityから明示的に解決するReferenceが成立しました。

```text
order_entry / sales.order.submit
```

PR #148と#149では、同じ原則をAuthentication / Authorizationにも適用しています。

```text
order_entry / security.authenticate
order_entry / security.authorize
```

Driverはfile path、Go hook名、列挙順からimplementationを推測しません。0件または複数件で一意に解決できない状態はBusiness Resultではなく、実行準備を完了できないerrorとして扱います。

## 業務Validation — Ready

Issue #125 / PR #136 / #139 / #140で、業務ValidationのCanonical rule、authoring source、最小Evaluatorが成立しています。

```tei
validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
}

validation sales.order.lines.line_number_unique relation-unique {
    relation sales.order.lines
    attribute sales.order_line.line_number
}
```

評価結果は`Satisfied / NotSatisfied`として扱い、必要な入力がないなど評価そのものを完了できない状態はerrorとして分離します。

Driverは`quantity > 0`等のrule本文を専用Goコードへ複製せず、同じCanonical ValidationとEvaluator境界を利用します。

## Capability結果境界 — Ready

ADR-0020 / Issue #77に従い、Capabilityを正常に実行できた場合の業務上の結果と、Invocationそのものを完了できなかった状態を分けます。

```text
Business Result
  ├─ existing output
  └─ named Outcome

Execution Error
  └─ Invocation自体を完了できない状態
```

PR #141の実Referenceでは、少なくとも次を区別できます。

- 正しいcredential → `authenticated`
- credential不一致 → `denied`
- account lock → `denied`
- Database / adapter障害 → Execution Error
- 未定義Outcome → Execution Error

既存の`sales.order.submit`は従来どおり`output sales.order`を維持し、同じInvocation入口から扱えます。Authenticationのために既存Capabilityへ暗黙のOutcomeを追加していません。

## Authentication — Ready

PR #148で、`security.authenticate`をv1 Reference Application `order_entry`から既存認証へ明示的に接続しました。

Referenceでは、既存の利用者情報に含まれる次の意味を保持しています。

- user ID
- authenticated subject identity
- credential reference
- disabled
- validity period
- login failure count
- lock state
- organization
- role

これらをTEI標準User schemaへ作り替えてから利用する構造にはしていません。

`order_entry/security.authenticate`を一意に解決でき、Source IndexからBindingのauthoring sourceへ戻れます。実行時credentialは検証境界へだけ渡し、Canonical World、Source Index、Business Result、通常Evidenceへコピーしません。

## Authorization — Ready

PR #149で、`security.authorize`を次の最小契約として実証しました。

```text
subject + action + resource
          ↓
 security.authorize
     ├─ allowed
     └─ denied
```

認可対象のresourceにはWeb画面名やURLではなく、既存のApplication Pattern identityを利用します。

```text
browse  / order_entry.recent_orders
edit    / order_entry.order_record
execute / order_entry.submit_order
```

既存システム固有のorganization、role、ruleはadapter側で利用し、ApplicationやDriverへ公開して同じ認可条件を再実装させません。

`allowed / denied`は認可判断を正常に完了した業務上の結果です。rule store等の実行障害はExecution Errorとして分離します。

## 既存認証の実現方式交換 — Ready

PR #151では、`order_entry/security.authenticate`という一つのApplication Bindingを変えず、その背後の実現方式だけを交換できることを確認しました。

比較したのは次の二経路です。

1. 既存の利用者情報を読み、credential verificationと既存ruleを組み合わせる経路
2. 既存関数、stored procedure、外部API等に相当する既存認証関数を呼ぶ経路

第二経路では、既存システム固有のreturn codeをadapterで`authenticated / denied`へ対応付けます。未対応return codeや既存関数の実行障害はExecution Errorです。

両経路は、同じCanonical `security.authenticate`契約、同じBinding identity、同じhookを保ち、同じauthenticated subjectを返します。

さらにPR #151では`reference/workspace`と`reference/world.json`を変更していません。CI #374のReference World再現性検査もGolden更新なしで成功しました。つまり、実現方式の違いをCanonical WorldやApplication上の意味へ持ち込まずに交換できています。

## Secret境界 — Ready

Reference上の認証fixtureとfunction mappingには、password平文、password hash、credential hash、token、secretを保存しません。

Canonicalに存在する`security.credential`は入力の意味を表す型であり、実際のcredential値を保存する場所ではありません。

Gate Aで確認した境界は次です。

- Git管理するReferenceへ実secretを置かない
- Canonical Worldへ実secretを置かない
- Source Indexへ実secretを置かない
- Business Resultへ実行時credentialをコピーしない
- Driver Scenarioの再現性を理由にsecretを通常Evidenceへ記録しない

実際のsecret解決方式は、production IdP等の具体化とともにGate A以後で扱えます。

## Driverがlegacy schemaへ直接依存しないための条件 — Ready

Gate Aで必要なのは、Driver実装そのものではなく、Driverがlegacy schemaを知らずに認証・認可を実行できる入口が揃っていることです。

現在は次をRepositoryから解決できます。

```text
security.authenticate
        ↓
Application-scoped Binding
        ↓
legacy adapter / function adapter
        ↓
authenticated subject
        ↓
security.authorize
        ↓
Application Pattern identityを使ったaction / resource
```

organization、role、legacy return code、物理schema等をDriverのCanonicalな入力へ追加する必要はありません。

Driver #69では、この境界を実際に利用し、legacy schemaへ直接依存していないことをGate BのConformanceとして改めて確認します。

## Driver開始判定

**Driver #69は開始可能です。**

Gate Aの必須項目に独立したBlockerは残っていません。

ただし、Gate A通過はDriverが完成したことを意味しません。次は、同じApplication上の意味をUIなしで一続きに実行し、Gate Bへ進めるかを確認します。

最小Scenarioは次です。

1. Authenticationを実行してsubjectを得る
2. `browse`で受注一覧を扱う
3. `record`で一件を扱う
4. `reference`で顧客を`master_data`から解決する
5. `header-detail`でOrderLineを扱う
6. Canonical業務Validationを実行する
7. Authorizationを評価する
8. `command`で`sales.order.submit`を呼び出す
9. Business ResultとExecution Errorを区別する
10. 必要な場合はObservationから外部状態を確認する

## Gate A以後へDeferredするもの

次は重要ですが、Gate AのBlockerにはしません。

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
- Runtime上で複数implementationを動的に選択する汎用機構
- `std.auth.*`の標準化

## 関連

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/design/LEGACY_AUTH_V1_DESIGN.md`
- `plan/design/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000009-capability-outcome-and-execution-error.md`
- `plan/issue/000013-gate-a-readiness-audit.md`
- `plan/issue/000014-business-validation-reference.md`
- `plan/issue/000015-submit-capability-binding-reference.md`
- GitHub Issue #67 / #68 / #69 / #77 / #124 / #125 / #126
- GitHub PR #132 / #135 / #136 / #139 / #140 / #141 / #143 / #148 / #149 / #151
- ADR-0019 / ADR-0020
