# ISSUE-000003: 既存認証仕様を保持したCompatibility Referenceを設計する

- 状態: In Progress
- GitHub Issue: #68
- Gate Aで唯一残っている独立blocker
- 実装前設計: [`LEGACY_AUTH_V1_DESIGN.md`](../LEGACY_AUTH_V1_DESIGN.md)
- 関連する意味境界: ADR-0019 / ADR-0020

## 背景

業務システムの移行では、標準的なID / password認証であっても、ユーザー無効化、有効期限、ログイン失敗回数、ロック、組織属性、role、個別ruleなど、システム固有の制御が長年積み重なっています。

TEIが既存システム移行に使えるためには、先に標準認証へ作り替えることを要求せず、既存仕様を保持したままApplicationから利用できるCapabilityへ接続する必要があります。

Application Pattern、Capability Binding、業務Validation、CapabilityのBusiness Result / Execution Error境界はGate Aに必要な範囲をすでに満たしました。したがって、現在のGate Aで独立して残っているのは、この既存認証Compatibilityだけです。

認証・認可の基本境界は[ADR-0019](../adr/0019-authentication-capability-boundary.md)、Capabilityの正常な業務結果とExecution Errorの境界は[ADR-0020](../adr/0020-capability-business-outcome-boundary.md)に従います。

## 関連するProduct Requirement

- RQ-004 Native asset
- RQ-008 Capability単位の置換
- RQ-009 既存認証・認可仕様の保持
- RQ-012 Driver
- RQ-015 AIによる既存知識の探索
- RQ-017 二重管理の最小化
- RQ-019 Resource authorityとの責務分離
- RQ-021 Reference / Conformance

## 目的

TEI標準のUser tableや認証schemaを先に定義するのではなく、**既存システム固有の認証状態と処理を、その意味を失わずAuthentication / Authorization Capabilityへ接続できること**を実証します。

Application利用者の認証・認可と、Plugin / RuntimeのPermission Grant / Resource Leaseは別の責務です。両者を同じ「権限」概念へ畳みません。

## すでに成立している前提

### `security.authenticate`のSemantic契約

Reference workspaceには、次のCapability契約があります。

```tei
capability security.authenticate {
    input login_id std.identifier
    input credential security.credential
    outcome authenticated security.authenticated_subject
    outcome denied security.authentication_denial
}
```

`authenticated`と`denied`は、Authentication処理を正常に完了して得られるBusiness Resultです。

### Business Result / Execution Error境界

Issue #77 / PR #141で、次を実行時に区別できることを確認済みです。

```text
valid credential       → authenticated
credential mismatch    → denied
account lock            → denied
Database / adapter障害 → Execution Error
```

したがって、#68では認証固有の別error体系を作りません。既存実装のreturn codeや例外を、ADR-0020で定めた共通境界へMappingします。

### Capability Binding

Issue #126 / PR #135で、Application + CapabilityからCustom Go Bindingを明示的に解決する入口が成立しています。

#68でも、file pathやGo symbolを暗黙に推測せず、この境界を再利用します。

## v1で扱う既存認証資産

小さな既存システムを想定し、最低限、次の状態を持つ認証資産をReferenceへ用意します。

```text
LegacyUser
- user_id
- credential_ref
- disabled
- valid_from
- valid_until
- login_failure_count
- locked_until
- organization_code
- role_code
```

さらに、単純なfield対応だけでは消えやすいシステム固有ruleを一つ含めます。

重要なのは、この既存schemaをTEI標準のUser schemaへコピーしてから使うことではありません。既存の名前、状態、処理を保持したまま、Applicationから利用できる論理Capabilityへ接続します。

## Authentication Capabilityへの接続

最初の実装では、既存User Storeとcredential verificationをadapter側へ残します。

```text
security.authenticate
        ↓
Capability Binding
        ↓
legacy authentication adapter
        ├─ existing user store / function
        └─ secret resolver / verifier
```

ApplicationやDriverはlegacy schemaのfield名や物理配置を知りません。

adapterは、既存のdisabled、有効期間、login failure count、lock、system-specific rule等を保持したまま、結果を`authenticated` / `denied`またはExecution Errorへ対応付けます。

## Authorization Capability

Authenticationで得たsubjectと、Application上のaction / resourceを使って認可を評価します。

v1ではReference workspace側の`security`意味空間で`security.authorize`を実証します。`std.auth.*`への標準化は、複数の実例から共通性が確認できるまで保留します。

少なくとも`order_entry`では、次の操作を区別できる必要があります。

- browse: 受注一覧を閲覧する
- edit: 受注recordと明細を編集する
- submit: `sales.order.submit`を実行する

Authorization ruleをWeb Targetのif文へ閉じ込めず、DriverとTargetが同じApplication上の意味を利用できる境界に置きます。

## Secretの扱い

password平文、秘密鍵、token、credential hash等のsecret materialをCanonical Worldへ保存しません。

少なくとも次へsecretを混入させません。

- `reference/world.json`
- Source Index
- Durable artifact
- 通常のDiagnostic
- 通常の実行log
- Git管理するReference fixture

Reference fixtureに置く`credential_ref`はsecretそのものではなく、実行時にsecret resolver / verifierへ渡す非秘密のhandleとして扱います。

テストでcredential入力が必要な場合も、実秘密情報ではないことが明らかなdeterministic fixture値を使い、通常Evidenceへ値を記録しません。

## 既存schemaとの接続方式

最終的なAuthentication CapabilityをDatabase方式へ固定しません。

比較対象は次です。

1. adapterが既存schemaを直接読む
2. 既存関数、stored procedure、外部API等をadapterから呼ぶ
3. 必要性が確認された場合に限り、Persistence Mapping等を介して論理意味へ対応付ける

v1では少なくとも二つの実現方式を同じ`security.authenticate` identityのまま扱い、implementationを交換してもApplication側の業務上の意味を作り直さないことを確認します。

## Plugin Capabilityとは分ける

認証実装がprocess Pluginとして動く場合でも、Application利用者のAuthenticationを`plugin.CapabilityKind`へ直接追加することを基本方針にはしません。

process PluginはSemantic Capabilityの実現方式であり、Plugin自身がどの外部resourceへアクセスしてよいかというPermissionとは別です。

```text
Application user authorization
  誰が、どの業務操作を実行できるか

Plugin / Runtime permission
  実装コードが、どの外部resourceへアクセスしてよいか
```

この二つをDriverや将来のWeb Targetでも混同しません。

## Gate Aへ渡す情報

Driverが認証製品やlegacy schemaの内部構造を知らなくても、Application上のAuthentication / Authorizationを評価できる境界を用意します。

Gate Aへ渡す情報は少なくとも次です。

- authenticated subject identity
- Authentication / Authorization Capability identity
- Application上のaction / resource
- Capability Bindingとlegacy Mappingの来歴
- 既存仕様のどの条件が認証・認可判断に使われるか
- secret materialをどの境界で解決するか
- `authenticated` / `denied`等のBusiness ResultとExecution Errorの区別

Gate Aではproduction-readyなIdPやログイン画面の完成を求めません。既存仕様を保ったままApplication上の意味へ接続できることを確認します。

## 実装順

現在の主線は次です。

1. **完了** — `security.authenticate`のSemantic契約をReference化する
2. **完了** — 名前付きOutcomeとExecution Errorの実行時境界を共通化する
3. legacy認証fixtureを、secretを含めずに用意する
4. legacy user store / functionを`security.authenticate`へ接続する最小adapterを実装する
5. `order_entry`から`security.authenticate`を明示的なBindingで利用できるようにする
6. denial / Execution Error / secret非混入をConformance化する
7. 同じCapability identityの第二実現方式を追加し、交換可能性を確認する
8. `security.authorize`の最小契約を実例から作る
9. `order_entry`のbrowse / edit / submitへaction / resourceを接続する
10. Source / Binding / Mappingの来歴を確認する
11. Gate Aを再監査する

## Conformance

少なくとも次を確認します。

- legacy field名をCanonical User schemaとして要求しない
- disabled / validity / failure count / lock等の既存意味を失わない
- system-specific ruleを標準化のために削らない
- secret plaintextやcredential hashをCanonical / Durable artifact / Source Index / 通常Evidenceへ保存しない
- AuthenticationとAuthorizationを一つのbooleanへ畳まない
- Application user authorizationとPlugin runtime permissionを混同しない
- 同じCapability identityを保ったままimplementationを交換できる
- implementationを交換してもApplication側の業務上の意味を作り直さない
- `order_entry`のAuthorizationをTarget固有コードへ閉じ込めない
- Source / Binding / implementationの来歴を追跡できる
- Driverがlegacy schemaの物理構造へ直接依存しない
- 業務上の拒否とExecution ErrorをADR-0020の境界で区別できる

## 非目標

- TEI標準User schemaの制定
- `std.auth.*`の即時標準化
- production IdPの内製
- OAuth / OIDC / SAML等の一括対応
- password hashing algorithmの独自実装
- Plugin Permission Grantとの統合
- GUI login pageの完成
- Authenticationを特定のDatabase方式へ固定すること

## 完了条件

既存認証仕様を変更せず、Authentication / Authorization Capabilityへ接続できるReferenceが成立することを完了条件とします。

さらに、同じCapability identityのまま実現方式を交換でき、`order_entry`のbrowse / edit / submit Authorizationへ接続できること、secretがCanonicalや通常Evidenceへ漏れないこと、Driverがlegacy schemaへ直接依存しないことをEvidenceとして残します。

この条件が揃った時点でGate Aを再監査し、Application Driver #69の開始可否を明示します。

## 関連文書

- `plan/LEGACY_AUTH_V1_DESIGN.md`
- `plan/GATE_A_READINESS_AUDIT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0019-authentication-capability-boundary.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000009-capability-outcome-and-execution-error.md`
