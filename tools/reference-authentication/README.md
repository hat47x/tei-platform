# reference-authentication（#2020）

SaaS・SIer向けの提供形態で使う、認証境界の最小Referenceです。認証情報の発行・保存・OIDC連携は行わず、提供pluginが返したprincipalをアプリケーション境界で検証します。

## 境界

```go
principal, err := authn.Authenticate(ctx, verifier, credential, tenantID)
```

- `Verifier`は、credentialを検証して`Principal`を返すplugin側の実装です。
- `Authenticate`は、credentialまたはtenantが空なら`AUTHENTICATION_REQUIRED`を返します。
- verifierが失敗した場合も`AUTHENTICATION_REQUIRED`とし、verifierの詳細やcredentialをエラーへ含めません。
- principalのtenantまたはsubjectが空なら`INVALID_PRINCIPAL`を返します。
- principalのtenantが要求tenantと異なる場合は`TENANT_MISMATCH`を返します。
- 匿名のprincipalへフォールバックしません。

## 提供しないもの

- credentialの発行、保存、更新、失効。
- OIDC・SAML・LDAPなどの具体的なProvider接続。
- MFA、セッション管理、権限（authorization）。
- 本番用のfixture verifier。

このReferenceは、Provider pluginが守るべき境界と、その境界のconformance testを提供します。

## plugin記述

`plugin.json`は、能力`authentication`、保証、保証しないことを宣言します。Delivery Profileの`verify`は、`authentication`を選んだ構成に対して次のconformanceを実行します。

- 有効なfixture credentialでprincipalを取得できる。
- credentialがない場合に拒否する。
- 別tenantのprincipalを拒否する。
- tenantまたはsubjectが欠けたprincipalを拒否する。

## 未実装

- 実Providerとの接続、credentialの安全な受渡し。
- 署名付きtokenの検証、鍵管理、失効。
- 権限判定、監査との接続。
