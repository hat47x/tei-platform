# 既存認証の明示的な接続の準拠確認

## 目的

この準拠確認では、既存システム固有の認証仕様をTEI標準の利用者データ構造へ作り替えず、第1版の参照用アプリケーション`order_entry`から意味上の機能`security.authenticate`へ明示的に接続できることを確かめます。

認証処理そのものには、PR #145で追加した`internal/security/legacyauth`を使います。機能を正常に実行して得られた業務結果と実行エラーは、PR #141で定めた共通の呼び出しの境界に従って区別します。

## アプリケーションから見える意味

`order_entry`は、`security`という意味空間を明示的に利用します。

```tei
application order_entry {
    uses semantic sales
    uses semantic customer
    uses semantic security
    ...
}
```

認証処理を特定のデータベース名や既存のデータ構造名としてアプリケーションへ持ち込むことはしません。アプリケーションからは、意味上の機能である`security.authenticate`を利用します。

## 明示的な接続

`order_entry`では、既存の`sales.order.submit`と同じGoによる接続定義の形式を使います。

```tei
realize security.authenticate with custom go
```

対応付けを識別する値は、新たな任意の識別子を増やさず、既存の原則に従ってアプリケーションと機能の組合せで表します。

```text
order_entry/security.authenticate
```

入力元索引から、この対応付けを記述したファイルである

```text
applications/order-entry/realizations/authenticate.tei
```

へ戻れることを準拠確認で確かめます。

## 既存仕様の試験用データ

Gitで管理する参照用の試験データは、`reference/legacy-auth/users.json`に置きます。

この試験用データは、TEI標準の利用者データ構造ではありません。既存システムに含まれ得る項目名や状態を、そのまま参照用の値として保持します。

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

`FixtureStore`はこれらの情報を`legacyauth.User`へ渡します。ただし、TEI標準の利用者データ構造への変換を目的とした型ではありません。

## 秘密情報の境界

Gitで管理する試験用データには、パスワードの平文・ハッシュ値、認証情報のハッシュ値、トークン、秘密情報を保存しません。

既存の利用者記録が保持するのは`credential_ref`だけです。実際に提示された認証情報は、実行時に`CredentialVerifier`へ渡し、意味の正本、入力元索引、業務結果、通常の検証根拠へはコピーしません。

準拠確認では、実行時にだけ渡した値が`BusinessResult`へ混入していないことも確かめます。

## 実行経路

参照実装の準拠確認では、リポジトリに保存された既存システムを模した試験用データを読み、同じ`security.authenticate`を`capabilityinvoke.Invoke`へ渡します。

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

この経路で認証が成立すると、PR #141で定めた`authenticated`という業務結果を返します。

## 来歴

判定段階Aで必要なのは、同じ認証規則を意味の正本に別の形で複製することではなく、少なくとも次の情報を追跡できることです。

- 意味上の機能：`security.authenticate`
- アプリケーション内の対応付け：`order_entry/security.authenticate`
- 対応付けの記述元：`applications/order-entry/realizations/authenticate.tei`
- 既存システムを模した試験データの入力元：`reference/legacy-auth/users.json`
- 認証情報の実際の値を解決・検証する境界：`CredentialVerifier`

試験用データの入力元のパスは、準拠確認で`FixtureStore.SourcePath()`から取得できます。

## この段階で固定しないこと

- TEI標準の利用者データ構造
- `std.auth.*`への標準化
- 本番用のパスワードのハッシュ化方式
- LDAP、OIDC、SAMLなどの製品選択
- 認証専用の新しい対応付け形式
- Webのログイン画面
- 認可の規則

認可は、次の一連の実証で`security.authorize`として`order_entry`の参照・編集・確定の操作へ接続します。

## 判定段階Aとの関係

この準拠確認により、認証に関する次の事項をリポジトリから説明できます。

- アプリケーションが利用する機能の識別子
- どのアプリケーション内の対応付けを採用したか
- 対応付けを記述した入力元
- 既存の項目を保持した試験用データの入力元
- 秘密情報の実際の値を保存しない境界
- 既存の試験用データから機能の呼び出しまでの接続

この段階だけで判定段階A全体が完了したことにはなりません。認可と実現方式の切替えについて検証根拠を追加してから、Issue #68を再評価します。

関連: #68 / #69 / #124 / PR #141 / #145 / ADR-0019 / ADR-0020
