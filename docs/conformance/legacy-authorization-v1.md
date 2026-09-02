# Legacy Authorization Conformance

## 目的

このConformanceは、既存システムにある組織・role・個別ruleをTEI共通のRBAC modelへ作り替えず、v1 Reference Application `order_entry`の認可判断へ接続できることを確認します。

AuthenticationとAuthorizationは分けます。Authenticationで得たsubject identityをAuthorizationへ渡しますが、ApplicationやDriverが既存のorganization codeやrole codeを直接読み取って判断する設計にはしません。

## Semantic Capability

v1 Referenceでは、`security.authorize`を次の最小契約として扱います。

```text
subject
+ action
+ resource
    ↓
security.authorize
    ├─ allowed
    └─ denied
```

`allowed`と`denied`はいずれも、認可判断を正常に完了できた場合のBusiness Resultです。

既存user storeへ到達できない場合や、既存ruleの評価処理そのものが失敗した場合はExecution Errorとし、`denied`へ変換しません。

## actionとresource

最初のReferenceでは、`order_entry`の実例から必要な三つの操作だけを扱います。

| action | resource | Application上の意味 |
| --- | --- | --- |
| `browse` | `order_entry.recent_orders` | 受注一覧を参照する |
| `edit` | `order_entry.order_record` | 受注recordを編集する |
| `execute` | `order_entry.submit_order` | 受注確定Commandを実行する |

resourceにはWeb画面名やURLを使いません。既存のApplication Pattern identityを使います。

`order_entry.submit_order`は既存のCommand Patternであり、`sales.order.submit` Capabilityへ接続されています。そのため、Authorization ruleへ同じCapability名を重ねて記述する必要はありません。

## legacy rule fixture

`reference/legacy-auth/authorization.json`は、既存システム固有の認可表を模したReferenceです。

現在の実例では、次の既存metadataをそのまま利用します。

- organization code
- role code
- action
- resource
- effect

これはTEI標準のrole modelではありません。既存仕様を失わずにSemantic Capabilityへ接続できることを確かめるためのfixtureです。

`ORDER_EDITOR`と`ORDER_VIEWER`も標準role名ではなく、このReference内のlegacy値です。

## subjectからlegacy userへの接続

Authenticationが返したsubject identityから、`FixtureStore.FindBySubject`を通して既存user recordを引き直します。

```text
Authenticated Subject
        ↓
security.authorize
        ↓
SubjectStore
        ↓
legacy user metadata
        ↓
legacy authorization rule
```

Application / Driverへorganizationやroleを公開して認可ruleを再実装させません。

## 明示Binding

`order_entry`ではAuthenticationと同じ既存のCustom Go Binding形式を使います。

```tei
realize security.authorize with custom go
```

Binding identityは次です。

```text
order_entry/security.authorize
```

Source Indexから、authoring sourceである

```text
applications/order-entry/realizations/authorize.tei
```

へ戻れることをConformanceで確認します。

Authorization専用のBinding形式は追加しません。

## Referenceで確認する判断

編集者では、次を`allowed`として確認します。

- 受注一覧の参照
- 受注recordの編集
- 受注確定Commandの実行

閲覧者では、次を確認します。

- 受注一覧の参照 → `allowed`
- 受注recordの編集 → `denied`
- 受注確定Commandの実行 → `denied`

未知のsubjectや、legacy ruleに一致しない操作は`denied / not_authorized`とします。

これはReference上の最小語彙です。Authentication一例とAuthorization一例だけから、標準的な拒否理由体系を先回りして固定しません。

## Execution Errorとの境界

次は`denied`ではありません。

- subject storeを実行できない
- legacy authorization ruleを評価できない
- `security.authorize`以外のCapabilityを誤ってadapterへ渡す
- subject / action / resourceが欠けている
- adapterの入力契約そのものが壊れている

これらは認可が拒否されたのではなく、認可判断を正常に完了できなかった状態です。PR #141で成立したCapability共通のExecution Error境界をそのまま利用します。

## Plugin / Runtime Permissionとの分離

このAuthorizationはApplication利用者が業務操作を実行できるかを表します。

process PluginがDatabaseや外部resourceへアクセスできるかを制御するPlugin / Runtime Permissionとは別の概念です。今回のReferenceでも両者を同じruleやCapabilityへ統合しません。

## この段階で固定しないこと

- TEI標準User / Role / Permission schema
- `std.auth.*`への標準化
- 汎用RBAC / ABAC rule language
- Webのroute guardやbutton表示
- HTTP statusへの変換
- production用Policy Engine
- Plugin / Runtime Permissionとの統合

これらは、複数Applicationや複数Targetから必要性が確認できた段階で検討します。

## Gate Aへの意味

このReferenceにより、Driverへ次を渡せる状態を目指します。

- authenticated subject identity
- Authorization Capability identity
- Application上のaction / resource
- legacy organization / role / ruleとの接続境界
- `allowed / denied`というBusiness Result
- Execution Errorとの区別
- Bindingとlegacy fixtureの来歴

このPRの後には、同じ`security.authenticate` Capability identityを維持したままlegacy実現方式を交換できることを確認します。そのEvidenceまで揃った段階でIssue #68とGate Aを再監査します。

関連: #68 / #69 / #124 / PR #141 / #145 / #148 / ADR-0019 / ADR-0020
