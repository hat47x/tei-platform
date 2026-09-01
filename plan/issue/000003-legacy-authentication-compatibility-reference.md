# ISSUE-000003: 既存認証仕様を保持したCompatibility Referenceを設計する

- 状態: 着手可能
- GitHub Issue: #68
- 実装前設計: [`LEGACY_AUTH_V1_DESIGN.md`](../LEGACY_AUTH_V1_DESIGN.md)

## 背景

業務システムの移行では、標準的なID / password認証であっても、ユーザー無効化、有効期限、ログイン失敗回数、ロック、組織属性、role、個別ruleなど、システム固有の制御が長年積み重なっています。

TEIが既存システム移行に使えるためには、先に標準認証へ作り替えることを要求せず、既存仕様を保持したままApplicationから利用できるCapabilityへ接続する必要があります。

Product Requirements Gap Auditでも、RQ-009はv1へ向けた大きな未実証領域として残っています。

このIssueはApplication Pattern #67と並行して進め、両者の成果をGate A「Application上の意味をDriverへ渡せる」で合流させます。

認証・認可の基本境界は[ADR-0019](../adr/0019-authentication-capability-boundary.md)に従います。

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

## v1で扱う既存認証資産

小さな既存システムを想定し、最低限、次の状態を持つ認証資産を用意します。

```text
LegacyUser
- user_id
- credential reference / password hash
- disabled
- valid_from
- valid_until
- login_failure_count
- locked_until
- organization_code
- role_code
```

さらに、単純なfield対応だけでは消えやすいシステム固有ruleを一つ含めます。

最終的なruleはReferenceを不自然に複雑にしない範囲で実装時に決めます。重要なのは、こうした既存仕様を「TEI標準に合わせるため」という理由で削らないことです。

## Capability境界

v1ではReference workspace側の`security`意味空間で、少なくとも次を分けて検証します。

```text
security.authenticate
security.authorize
```

### `security.authenticate`

credentialをもとにApplication利用者のidentityを確認します。

概念上、次を扱える必要があります。

- login identifier
- credential inputまたはcredential handle
- authentication outcome
- authenticated subject
- 認証が成立しなかった場合の理由

成功、正常な拒否、実行エラーの表し方は`ISSUE-000009` / GitHub Issue #77で決めます。

### `security.authorize`

authenticated subjectがApplication上のaction / resourceを実行できるかを判定します。

組織、role、既存の個別rule等をimplementation側から利用できる必要があります。

AuthenticationとAuthorizationを一つのboolean関数へ固定しません。

## `std`への標準化は保留する

将来`std.auth.*`等の標準Capabilityが有用になる可能性はあります。

ただし、一つのReferenceだけで標準契約を固定しません。v1ではworkspace側の`security`意味空間で実証し、複数Applicationや複数実現方式でも同じ契約として成立するEvidenceを得てから標準化を判断します。

## 既存schemaとの接続

既存fieldをTEI標準fieldへ物理コピーすることを前提にしません。

次の方式を比較します。

1. Capability implementationが既存schemaを直接読む
2. Persistence Mapping等を通じて既存fieldを論理的な意味へ対応付ける
3. 既存関数、stored procedure、外部API等をadapterから呼ぶ

Referenceでは、少なくとも一つの既存schema経路と、一つのCustom implementation経路を確認します。

重要なのは、Applicationから見たCapability identityを変えずに実現方式を交換できることです。

## Plugin Capabilityとは分ける

認証実装がprocess Pluginとして動く場合でも、Application利用者のAuthenticationを`plugin.CapabilityKind`へ直接追加することを基本方針にはしません。

process PluginはSemantic Capabilityの実現方式であり、Plugin自身がどの外部resourceへアクセスしてよいかというPermissionとは別です。

## Secretの扱い

password平文、秘密鍵、token等のsecret materialをCanonical Worldへ保存しません。

少なくとも次へsecretを混入させません。

- `reference/world.json`
- Source Index
- Durable artifact
- 通常のDiagnostic
- 通常の実行log
- Git管理するReference fixture

Canonical側には、Capability契約、secretではないMapping / Bindingの来歴、どの境界でsecretを解決するかだけを残します。

## 業務結果と実行エラー

Authenticationでは、少なくとも次を分ける必要があります。

```text
Authenticated(subject)
Denied(reason)
ExecutionError(error)
```

credential不一致やaccount lockは、認証処理を正常に実行した結果としての拒否です。Database接続失敗やadapter異常とは意味が異なります。

この問題はAuthentication固有ではないため、`ISSUE-000009` / GitHub Issue #77へ分離します。#68だけの場当たり的なerror規約は作りません。

## `order_entry`との接続

Reference単体で境界を確認した後、v1 `order_entry`へ接続します。

少なくとも次を確認します。

- 受注一覧を閲覧できる利用者
- 受注recordを編集できる利用者
- `sales.order.submit` commandを実行できる利用者

Authorization ruleをWeb Target内のif文として再実装せず、Application / Capability境界から利用します。

## Database方式へ固定しない

既存認証情報は、別Database、同一Databaseの別schema、LDAP、IdP、独自API等のどこに存在するか分かりません。

Referenceで既存Databaseを利用しても、Authentication Capability自体を「第三Datasourceが必ず存在する」契約にはしません。

## Gate Aへ渡す情報

Driverが認証製品やlegacy schemaの内部構造を知らなくても、Application上のAuthorizationを評価できる境界を用意します。

Gate Aへ渡す情報は少なくとも次です。

- authenticated subjectを表すidentity
- Application上で認可対象となるaction / resource
- Authentication / Authorization Capabilityのidentity
- Capability BindingとMappingの来歴
- 既存仕様のどの条件が認証・認可判断に使われるか
- secret materialをどの境界で解決するか

Gate Aではproduction-readyなIdPやログイン画面の完成を求めません。既存仕様を保ったままApplication上の意味へ接続できることを確認します。

## AI / Editorへの引き継ぎ

将来AIやEditorが認証要件を確認するとき、既にMapping済みのfieldやruleを再質問しないためには、採用済みのCapability BindingとMappingを機械的に確認できる必要があります。

このIssueではAI UI自体は実装しませんが、後から「何が決定済みか」を追跡できる構造を壊さないことを条件とします。

## 実装順

1. `security`意味空間の最小Referenceを定める
2. Authentication / Authorization Capabilityのidentityと責務を定める
3. #77でCapability outcomeと実行エラーの境界を確定する
4. legacy認証fixtureを用意する
5. existing schema / functionとCapabilityを接続する
6. Custom Go等の異なる実現方式を一つ追加する
7. secret非混入をConformance化する
8. `order_entry`のbrowse / edit / submitへAuthorizationを接続する
9. Source / Binding / Mappingの来歴を確認する
10. #67と合わせてGate Aを監査する

## Conformance

少なくとも次を確認します。

- legacy field名をCanonical identityとして要求しない
- disabled / validity / failure count / lock等の既存意味を失わない
- system固有ruleを標準化のために削らない
- secret plaintextをCanonical / Durable artifact / Source Indexへ保存しない
- AuthenticationとAuthorizationを一つのbooleanへ畳まない
- Application user authorizationとPlugin runtime permissionを混同しない
- 同じCapability identityを保ったままimplementationを交換できる
- implementationを交換してもApplication側の業務上の意味を作り直さない
- `order_entry`のAuthorizationをTarget固有コードへ閉じ込めない
- Source / Binding / implementationの来歴を追跡できる
- Driverがlegacy schemaの物理構造へ直接依存しない
- 業務上の拒否と実行エラーを#77の境界で区別できる

## 非目標

- TEI標準User schemaの制定
- `std.auth.*`の即時標準化
- production IdPの内製
- OAuth / OIDC / SAML等の一括対応
- password hashing algorithmの独自実装
- Plugin Permission Grantとの統合
- GUI login pageの完成
- このIssueだけでGate Aを完了させること

## 完了条件

既存認証仕様を変更せず、Authentication / Authorization Capabilityへ接続できるReferenceが成立することを完了条件とします。

さらに、同じCapability identityのまま実現方式を交換でき、`order_entry`のbrowse / edit / submit Authorizationへ接続できること、secretがCanonicalへ漏れないこと、Driverがlegacy schemaへ直接依存しないことをEvidenceとして残します。

`ISSUE-000005`のApplication Patternと合わせて、Gate A「Application上の意味をDriverへ渡せる」を評価します。

## 関連文書

- `plan/LEGACY_AUTH_V1_DESIGN.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0019-authentication-capability-boundary.md`
- `plan/issue/000009-capability-outcome-and-execution-error.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
