# ADR-0019: Application利用者の認証・認可をSemantic Capabilityとして扱う

- Status: Accepted
- Date: 2026-09-01

## Context

TEIが既存業務システムの移行を扱うためには、既存の認証・認可仕様を失わずにApplicationへ接続できる必要があります。

一方、TEIにはすでにPlugin / RuntimeのPermissionやResource Authorityがあります。どちらも日常語では「権限」と呼べますが、意味は異なります。

```text
Application user authorization
  誰が、どの業務操作を実行できるか

Plugin / Runtime permission
  実装コードが、どの外部resourceへアクセスしてよいか
```

また、既存認証処理はDatabase、LDAP、IdP、独自API、既存関数等のさまざまな形で存在します。これらの物理方式をそのままCanonicalなAuthenticationの意味へ持ち込むと、Applicationが特定製品やschemaへ固定されます。

## Decision

Application利用者のAuthentication / Authorizationは、**Semantic Capabilityとして表します。**

v1 Referenceではworkspace側の`security`意味空間で検証します。

概念上、少なくとも次を分けます。

```text
security.authenticate
security.authorize
```

### AuthenticationとAuthorizationを分離する

Authenticationは利用者identityを確認する責務、Authorizationは確認済みsubjectがApplication上のaction / resourceを実行できるか判定する責務として分けます。

一つの`is_allowed` boolean関数へ統合しません。

### Plugin Capabilityとは別の概念として扱う

`plugin.CapabilityKind`へAuthentication / Authorizationを追加することを、このADRの基本方針にはしません。

認証実装がprocess Pluginとして動く場合でも、process PluginはSemantic Capabilityの**実現方式**です。Application利用者の認証という意味と、HostがPluginへ与えるPermissionを同じCapabilityへ畳みません。

### 既存schemaを標準User schemaへ強制変換しない

既存システムのuser table、状態field、組織情報、role、個別rule等を保持したままCapabilityへ接続します。

TEI標準のUser schemaへ物理コピーしてからでなければ利用できない設計にはしません。

### 物理方式はBinding / implementation側へ残す

Semantic CapabilityにはDatabase名、LDAP製品、IdP製品、DSN、credential等を持ち込みません。

既存Databaseを読むadapter、既存関数、Custom Go、将来のprocess実装等はRealization / Binding側で扱います。

同じCapability identityを保ったまま実現方式を交換できることをReferenceで確認します。

### SecretをCanonicalへ保存しない

password平文、秘密鍵、token等のsecret materialをCanonical World、Source Index、Durable artifact、通常Diagnosticへ保存しません。

Canonical側には、Capability契約、secretではないMapping / Bindingの来歴、どの境界でsecretを解決するかだけを残します。

### `std`への昇格は後で判断する

v1では`security.authenticate` / `security.authorize`をReference workspace側の意味として検証します。

`std.auth.*`等の標準Capability名や標準契約は、このADRでは固定しません。複数Applicationや複数実現方式でも同じ契約が成立するEvidenceを得てから標準化を判断します。

### 業務上の拒否と実行エラーは別課題として分離する

Authenticationでは、認証成功、credential不一致やlock等の正常な拒否、Database接続失敗等の実行エラーを区別する必要があります。

この問題はAuthentication固有ではないため、`ISSUE-000009` / GitHub Issue #77でCapability共通の結果境界として扱います。

このADRでは、Authentication専用の場当たり的なerror規約を定めません。

## Consequences

### 良くなる点

- Applicationが認証製品やlegacy schemaの物理構造へ直接依存しにくくなる
- 既存認証仕様を保持した移行が可能になる
- Authentication / AuthorizationとPlugin Permissionの意味を分離できる
- Web TargetやDriverで認可ruleを作り直さずに済む
- 実現方式をCapability identityから分離できる
- 将来AI / Editorが、既に決まっているBindingやMappingを追跡しやすくなる

### 負担になる点

- legacy schemaとCapabilityのMapping / Bindingを明示する必要がある
- secret解決境界を別に管理する必要がある
- Application利用者のsubject / action / resourceをDriverへ渡す契約が必要になる
- Capability outcomeと実行エラーの共通境界を別途設計する必要がある

これらは、既存認証仕様を失わず、Applicationと実現方式を分離するために受け入れます。

## 却下した選択肢

### TEI標準User schemaへ先に統一する

新規システムでは便利でも、既存システム移行では仕様変更を前提にしてしまいます。v1のCompatibility Referenceには採用しません。

### AuthenticationをPlugin Capabilityとして扱う

実行方式と業務上の意味を混同するため採用しません。process Pluginを使う場合でも、それはSemantic Capabilityの実現方式として扱います。

### Web Target内で認証・認可を実装する

Application上の意味がTarget固有コードへ閉じ、Driverや第二Targetで再利用できないため採用しません。

### `std.auth.*`を直ちに標準化する

一つのReferenceだけでは標準契約として十分なEvidenceがないため、v1では見送ります。

## Verification

`ISSUE-000003`では、少なくとも次を確認します。

- 既存schemaを標準User schemaへ変換せず利用できる
- Authentication / Authorizationを別Capabilityとして追跡できる
- Application user authorizationとPlugin / Runtime Permissionを混同しない
- 同じCapability identityを保ったまま実現方式を交換できる
- secretがCanonical / Source Index / Durable artifactへ混入しない
- Driverがlegacy schemaの物理構造へ直接依存しない
- Web Targetへ認可ruleを再実装しなくてよい
- Mapping / Binding / implementationの来歴を追跡できる
- `ISSUE-000009`の結果境界と組み合わせて業務上の拒否と実行エラーを分けられる

## 関連文書

- `plan/LEGACY_AUTH_V1_DESIGN.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000009-capability-outcome-and-execution-error.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
