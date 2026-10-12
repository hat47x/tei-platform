# Gate A 到達性監査

## 目的

この文書は、第1版の参照アプリケーション`order_entry`をアプリケーションドライバーへ引き渡す前に、判定段階Aの「アプリケーション上の意味をドライバーへ渡せる」という条件を横断的に確認した監査記録です。

判定段階Aの合否は、関連するIssueの完了件数では決めません。ドライバーがウェブ画面や専用のGoコードで不足する意味を補わずとも、必要な情報をリポジトリの参照事例と準拠確認から再現できることを基準とします。

## 評価区分

| 区分 | 意味 |
| --- | --- |
| **Ready** | Gate Aに必要な範囲をReference / Conformanceから再現できる |
| **Partial** | 重要な境界は成立しているが、Driverへ渡す情報が不足している |
| **Blocked** | Driverへ進む前に独立した実証が必要である |
| **Deferred** | 必要な作業だがGate A以後でよい |

## 結論

**判定段階AはReadyです。アプリケーションドライバー#69の主な実装を開始できる状態です。**

最後まで残っていたIssue #68「既存認証の互換性」について、PR #148・#149・#151で判定段階Aに必要な検証根拠が揃いました。

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

この判定の根拠は、#68をクローズしたこと自体ではありません。認証と認可を含め、ドライバーに必要な意味をリポジトリから再現できると確認した結果です。

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

Issue #67では、`order_entry`の次の5種類のアプリケーションパターンが、実際の参照事例として成立しています。

- `browse`
- `record`
- `reference`
- `command`
- `header-detail`

パターンは既存資産の意味を書き換える場所ではありません。アプリケーションが正本上のどの識別情報を、どの目的で使うかを保持する層です。

パターンの識別情報はファイルのパスに依存せず、ソース索引から記述元のソースへたどれます。この5種類を恒久的な標準カタログとする判断は、ドライバーと複数の実行先で検証根拠が得られるまで保留します。

## Datasource境界 — Ready

`order_entry`は、次の二つの論理的なデータ接続先を利用します。

```text
sales  -> sales_core  -> read-write
master -> master_data -> read
```

データ接続先へのアクセス条件はパターンへ複製せず、アプリケーションの定義を正本とします。ドライバーも同じ`DatabaseRequirement`を使い、独自の設定には書き写しません。

認証情報の保存先を、この二つのデータ接続先に無理に組み込むことはしません。既存の認証機構がデータベース・LDAP・IdP・既存関数・外部APIなどのどこにあっても、ドライバーは認証・認可の能力を通して利用します。

## Capability Binding — Ready

Issue #126・PR #135により、アプリケーションと能力の組合せから`sales.order.submit`を明示的に解決できる参照事例が成立しました。

```text
order_entry / sales.order.submit
```

PR #148と#149では、同じ原則を認証・認可にも適用しています。

```text
order_entry / security.authenticate
order_entry / security.authorize
```

ドライバーは、ファイルパス・Goのフック名・列挙順から実装を推測しません。候補が0件または複数件で一意に特定できなければ、業務上の結果ではなく、実行準備が完了しないエラーとして扱います。

## 業務Validation — Ready

Issue #125・PR #136・#139・#140では、業務上の検証規則の正本、記述元のソース、最小限の評価器が成立しています。

```tei
validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
}

validation sales.order.lines.line_number_unique relation-unique {
    relation sales.order.lines
    attribute sales.order_line.line_number
}
```

評価結果には`Satisfied / NotSatisfied`を用います。必要な入力がないなど、評価そのものを完了できない場合は、別のエラーとして区別します。

ドライバーは`quantity > 0`などの規則の本文を専用のGoコードに複製せず、同じ正本上の検証規則と評価器の境界を利用します。

## Capability結果境界 — Ready

ADR-0020・Issue #77に従い、能力を正常に呼び出せた場合の業務上の結果と、呼出しそのものを完了できなかった状態を区別します。

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
- ISSUE-000003-legacy-authentication-compatibility-reference.md
- ISSUE-000006-application-driver-vertical-slice.md
- ISSUE-000009-capability-outcome-and-execution-error.md
- ISSUE-000013-gate-a-readiness-audit.md
- ISSUE-000014-business-validation-reference.md
- ISSUE-000015-submit-capability-binding-reference.md
- GitHub Issue #67 / #68 / #69 / #77 / #124 / #125 / #126
- GitHub PR #132 / #135 / #136 / #139 / #140 / #141 / #143 / #148 / #149 / #151
- ADR-0019 / ADR-0020
