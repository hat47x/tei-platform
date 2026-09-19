# ISSUE-000003: 既存認証仕様を保持したCompatibility Referenceを実証する

- 状態: Done
- GitHub Issue: #68
- 実装前設計: [`LEGACY_AUTH_V1_DESIGN.md`](../design/LEGACY_AUTH_V1_DESIGN.md)
- Gate A判定: Ready

## 背景

業務システムの移行では、標準的なID / password認証であっても、利用者の無効化、有効期間、ログイン失敗回数、lock、組織、role、個別ruleなど、システム固有の制御が長年積み重なっています。

TEIが既存システム移行に使えるためには、先に標準認証へ作り替えることを要求せず、既存仕様を保持したままApplicationから利用できるCapabilityへ接続できる必要があります。

このIssueでは、Application利用者のAuthentication / AuthorizationをSemantic Capabilityとして扱い、Plugin / RuntimeのPermission Grant / Resource Leaseとは別の責務として実証しました。

## 関連するProduct Requirement

- RQ-004 Native asset
- RQ-008 Capability単位の置換
- RQ-009 既存認証・認可仕様の保持
- RQ-012 Driver
- RQ-015 AIによる既存知識の探索
- RQ-017 二重管理の最小化
- RQ-019 Resource authorityとの責務分離
- RQ-021 Reference / Conformance

## 結論

**Gate Aで必要な既存認証CompatibilityはReadyです。**

PR #148、#149、#151により、次をRepository上のReference / Conformanceから確認できるようになりました。

- 既存の利用者情報をTEI標準User schemaへ作り替えずAuthenticationへ接続できる
- `security.authenticate`を`order_entry`から明示的にBindingできる
- authenticated subjectをApplication側へ渡せる
- `security.authorize`をApplication上のaction / resourceへ接続できる
- organization、role、既存ruleをadapter側へ保てる
- 業務上の拒否とExecution Errorを区別できる
- 実行時credentialやsecretをCanonical / Source Index / Business Result / 通常Evidenceへ保存しない
- 一つのApplication Bindingを保ったまま既存認証の実現方式を交換できる
- 実現方式を交換してもCanonical Worldを変更する必要がない

## Authentication

v1ではReference Workspaceの`security`意味空間にある`security.authenticate`を利用します。

```text
login identifier + credential
            ↓
    security.authenticate
       ├─ authenticated
       └─ denied
```

`authenticated` / `denied`は、認証処理を正常に完了して得られたBusiness Resultです。

Database接続失敗、adapter障害、既存関数の実行障害、未対応return code等は、認証判断を正常に完了できなかった状態としてExecution Errorに残します。

この境界はADR-0020 / Issue #77で成立したCapability共通のInvocation境界を利用します。

## 既存の利用者情報を保持するReference

PR #148では、既存システムを模した利用者情報を次のfieldを保ったまま読み込めるReferenceを追加しました。

```text
LegacyUser
- user_id
- subject_id
- credential_ref
- disabled
- valid_from
- valid_until
- login_failure_count
- locked_until
- organization_code
- role_code
```

これらをTEI標準のUser tableへコピーしてから利用する構造にはしていません。

`legacyauth.Adapter`は、既存の利用者情報、credential verification、システム固有ruleを`security.authenticate`へ接続します。

少なくとも次を区別できます。

- 正常なcredential → `authenticated`
- credential不一致 → `denied`
- disabled → `denied`
- 有効期間外 → `denied`
- lock中 → `denied`
- システム固有条件の不成立 → `denied`
- store / verifier / ruleの実行障害 → Execution Error

## Authentication Binding

`order_entry`では次のBindingを一意に解決できます。

```text
order_entry / security.authenticate
```

Bindingのauthoring sourceはSource Indexから追跡できます。

Authenticationだけの特別なBinding形式は作らず、`sales.order.submit`で成立したApplication + Capabilityの解決原則を再利用しています。

Driverはlegacy schema、fixture path、Go hook名等から認証実装を推測する必要がありません。

## Authorization

PR #149では、Authenticationで得たsubjectを利用する`security.authorize`を実証しました。

```text
subject + action + resource
          ↓
 security.authorize
     ├─ allowed
     └─ denied
```

最初のReferenceでは、認可対象のresourceにApplication Pattern identityを利用します。

```text
browse  / order_entry.recent_orders
edit    / order_entry.order_record
execute / order_entry.submit_order
```

Web画面名、URL、button名を認可の正本にはしません。

`order_entry.submit_order`はCommand Patternとして`sales.order.submit`へ接続済みなので、認可ruleへCapability実装の情報を重複記述しません。

既存のorganization、role、ruleはadapter側で評価します。ApplicationやDriverへこれらを公開し、同じ認可判断を再実装させない構造です。

## 既存認証の実現方式を交換する

PR #151では、次の二つの実現方式を比較しました。

### 既存の利用者情報を読む方式

`legacyauth.Adapter`が既存recordを読み、credential verificationと既存ruleを組み合わせます。

### 既存認証関数を呼ぶ方式

`legacyauth.FunctionAdapter`が、既存関数、stored procedure、外部API等に相当する認証処理を呼びます。

Referenceでは`legacy_security.authenticate_user`という既存関数を想定し、既存システム固有のreturn codeをadapterでSemantic Outcomeへ対応付けます。

```text
0  -> authenticated
10 -> denied / credential_mismatch
20 -> denied / account_disabled
30 -> denied / account_locked
40 -> denied / expired
50 -> denied / not_yet_valid
60 -> denied / organization_specific_condition
```

未対応return codeは正常な拒否とはみなさずExecution Errorにします。

## 実現方式交換で守った境界

実現方式を比較するために、複数のAuthentication Bindingを同時に登録して曖昧な選択状態を作ってはいません。

`order_entry/security.authenticate`という**一つのApplication Binding**を保ち、その背後のadapterを交換します。

両方式で次が不変です。

- Canonical Capability identity: `security.authenticate`
- Application Binding identity: `order_entry/security.authenticate`
- Binding hook
- Business Result契約
- authenticated subject identity

PR #151では`reference/workspace`と`reference/world.json`を変更していません。CI #374のReference World再現性検査もGolden更新なしで成功しました。

このEvidenceから、既存認証の実現方式を変えるためにApplication上の意味を作り直す必要がないことを確認できました。

## Secretの扱い

password平文、password hash、credential hash、token、secret等をCanonical Worldへ保存しません。

少なくとも次への非混入をReference / Conformanceで確認しています。

- `reference/world.json`
- Source Index
- Business Result
- Git管理するlegacy user fixture
- Git管理するfunction mapping fixture
- 通常Evidence

Canonical側の`security.credential`は入力の意味を表す型です。実際のcredential値は実行時に検証境界へ渡すだけで、Canonicalへコピーしません。

## Application利用者の権限とPlugin権限を分ける

次の二つは別の責務です。

```text
Application user authorization
  誰が、どの業務操作を実行できるか

Plugin / Runtime permission
  実装コードが、どの外部resourceへアクセスしてよいか
```

認証adapterがDatabaseや外部APIへ接続するためのHost側Permissionが必要になっても、それを「この利用者が受注確定を実行できるか」という業務認可と同じ概念にはしません。

## Gate Aへ渡す情報

#68からDriverへ、少なくとも次を渡せる状態になりました。

- authenticated subject identity
- Authentication Capability `security.authenticate`
- Authorization Capability `security.authorize`
- Application上のaction / resource
- Application-scoped Capability Binding
- Binding / legacy mappingの来歴
- named Outcome / Execution Errorの結果契約
- 既存仕様のうち認証・認可判断に利用する条件
- secretを実行時だけ解決する境界

organization、role、legacy return code、物理schema等をDriverのCanonicalな入力へ追加する必要はありません。

## Driverへ残す確認

Gate Aでは、Driverが利用できる意味境界が揃ったことを確認しました。

Driver #69では、その境界を実際に利用して次をConformance化します。

- legacy schemaへDriverが直接依存しない
- Authenticationの`denied`をDriver障害として扱わない
- Authorizationの`denied`ではcommandへ進まない
- Binding解決失敗やadapter障害はExecution Errorとして扱う
- secretをScenario Evidenceへ保存しない

これは#68の設計をやり直す作業ではなく、Gate Aで成立した境界をDriverから利用できることの確認です。

## 非目標

- TEI標準User schemaの制定
- `std.auth.*`の即時標準化
- production IdPの内製
- OAuth / OIDC / SAML等の一括対応
- password hashing algorithmの独自実装
- Plugin Permission Grantとの統合
- GUI login pageの完成
- Runtime上で複数implementationを動的に選択する汎用機構

## 完了条件の評価

- [x] 既存認証仕様を変更せずAuthentication Capabilityへ接続できる
- [x] AuthenticationとAuthorizationを分離できる
- [x] `order_entry`のbrowse / edit / submitへAuthorizationを接続できる
- [x] secretをCanonicalや通常Evidenceへ保存しない
- [x] 業務上の拒否とExecution Errorを区別できる
- [x] 同じCapability identityとApplication Bindingのまま実現方式を交換できる
- [x] 実現方式の交換でApplication上の意味を作り直さない
- [x] Gate AからDriverへ必要な意味を渡せる

## 関連文書

- `plan/design/LEGACY_AUTH_V1_DESIGN.md`
- `plan/GATE_A_READINESS_AUDIT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0019-authentication-capability-boundary.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/issue/000009-capability-outcome-and-execution-error.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- GitHub PR #148 / #149 / #151
