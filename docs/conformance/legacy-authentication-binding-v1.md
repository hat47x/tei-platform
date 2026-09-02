# Legacy Authentication Binding Conformance

## 目的

このConformanceは、既存システム固有の認証仕様をTEI標準のUser schemaへ作り替えず、v1 Reference Application `order_entry`からSemantic Capability `security.authenticate`へ明示的に接続できることを確認します。

認証処理そのものの境界は、PR #145で追加した`internal/security/legacyauth`を利用します。Capabilityを正常に実行して得られる業務上の結果とExecution Errorの区別は、PR #141で成立した共通のInvocation境界をそのまま利用します。

## Applicationから見える意味

`order_entry`は`security`意味空間を明示的に利用します。

```tei
application order_entry {
    uses semantic sales
    uses semantic customer
    uses semantic security
    ...
}
```

Authenticationを特定のDatabase名やlegacy schema名としてApplicationへ持ち込みません。Applicationから見えるのは`security.authenticate`というSemantic Capabilityです。

## 明示Binding

`order_entry`では、既存の`sales.order.submit`と同じCustom Go Binding形式を使います。

```tei
realize security.authenticate with custom go
```

Binding identityは新しい任意IDを増やさず、既存原則どおりApplicationとCapabilityの組で表します。

```text
order_entry/security.authenticate
```

Source Indexから、このBindingのauthoring sourceである

```text
applications/order-entry/realizations/authenticate.tei
```

へ戻れることをConformanceで確認します。

## legacy fixture

Git管理するReference fixtureは`reference/legacy-auth/users.json`に置きます。

このfixtureはTEI標準User schemaではありません。既存システム側に存在し得るfield名と状態を、そのままReferenceとして保持します。

- `user_id`
- `subject_id`
- `credential_ref`
- `disabled`
- `valid_from`
- `valid_until`
- `login_failure_count`
- `locked_until`
- `organization_code`
- `role_code`

`FixtureStore`はこれらを`legacyauth.User`へ渡しますが、標準User schemaへ変換することを目的とした型ではありません。

## secretの境界

Git管理するfixtureには、password平文、password hash、credential hash、token、secretを保存しません。

legacy recordが保持するのは`credential_ref`だけです。実際に提示されたcredentialは実行時に`CredentialVerifier`へ渡し、Canonical World、Source Index、Business Result、通常のEvidenceへコピーしません。

Conformanceでは、実行時だけ渡した値が`BusinessResult`へ混入していないことも確認します。

## 実行経路

Reference Conformanceでは、Repositoryに保存されたlegacy fixtureを読み、同じ`security.authenticate` Capabilityを`capabilityinvoke.Invoke`へ渡します。

```text
order_entry
   │
   └─ security.authenticate
            │
            ├─ explicit Custom Go Binding
            │
            └─ Legacy Authentication Adapter
                    │
                    ├─ FixtureStore
                    └─ CredentialVerifier
```

この経路で認証が成立すると、PR #141で定めた`authenticated` Business Resultを返します。

## 来歴

Gate Aで必要な来歴は、同じ認証ruleを別のCanonical表現へ複製することではなく、少なくとも次を追跡できることです。

- Semantic Capability: `security.authenticate`
- Application-scoped Binding: `order_entry/security.authenticate`
- Binding authoring source: `applications/order-entry/realizations/authenticate.tei`
- legacy fixture source: `reference/legacy-auth/users.json`
- credentialの実値を解決・検証する境界: `CredentialVerifier`

fixtureのsource pathは`FixtureStore.SourcePath()`からConformance上で確認できます。

## この段階で固定しないこと

- TEI標準User schema
- `std.auth.*`への標準化
- production用password hash方式
- LDAP / OIDC / SAML等の製品選択
- Authentication専用の新しいBinding形式
- Webログイン画面
- Authorization rule

Authorizationは次の縦切りで`security.authorize`として`order_entry`のbrowse / edit / submitへ接続します。

## Gate Aへの意味

このConformanceにより、Authenticationについては次をRepositoryから説明できます。

- Applicationが利用するCapability identity
- どのApplication-scoped Bindingを採用したか
- Bindingのauthoring source
- legacy fieldを保持したfixtureのsource
- secret materialを保存しない境界
- legacy fixtureからCapability Invocationまでの接続

Gate A全体はまだ完了しません。次にAuthorizationと実現方式交換のEvidenceを追加したうえで、Issue #68を再評価します。

関連: #68 / #69 / #124 / PR #141 / #145 / ADR-0019 / ADR-0020
