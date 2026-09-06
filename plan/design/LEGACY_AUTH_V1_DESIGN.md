# 既存認証Compatibility v1 実装前設計

- 状態: 実装前設計

この文書は実装前に置いた設計を保持する文書です。実装・Evidenceの現在地は[ISSUE-000003](../issue/000003-legacy-authentication-compatibility-reference.md)と[`docs/status.md`](../../docs/status.md)を正とし、この文書の将来形表現をcurrent progressの主張として読み替えません。

## 目的

この文書は、`ISSUE-000003` / GitHub Issue #68で進める既存認証Compatibilityについて、v1 Referenceで何を実証し、どの境界をまだ固定しないかを整理するものです。

TEIが既存業務システムの移行に使えるためには、「標準的な認証方式へ作り替えてから取り込む」ことを前提にできません。実際のシステムでは、ユーザー無効化、有効期間、ログイン失敗回数、ロック、組織、役割、個別の例外条件などが長年積み重なっています。

v1では、こうした既存仕様を失わずにAuthentication / Authorization Capabilityへ接続し、ApplicationやDriverが物理的な認証schemaを直接知らなくても利用できる境界を実証します。

Application利用者の認証・認可と、Plugin / Runtimeに与えるPermissionやResource Leaseは別の責務です。この境界は[ADR-0019](../adr/0019-authentication-capability-boundary.md)で固定します。

Authenticationを含むCapabilityの業務Outcomeと実行エラーの境界は、[ADR-0020](../adr/0020-capability-business-outcome-boundary.md)で固定します。具体的なv1実装順は[`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md)に置きます。

## 現行実装から確認できる前提

### Semantic Capabilityは薄い契約として使える

現在のSemantic Capabilityは、Canonical ID、型付きinput、output型を中心とした小さな契約です。

そのため、認証製品名、Database schema、LDAP、IdP等の物理方式をCapabilityそのものへ埋め込まず、「Applicationから何を要求できるか」を表せます。

### RealizationはCapabilityから分離されている

現行のCustom Go Bindingは、ApplicationとCapabilityの対応をRealization側で持ちます。

この境界を使えば、同じCapability identityを保ったまま、既存Databaseを読むadapter、既存関数を呼ぶ実装、Custom Go、将来のprocess実装等を比較できます。

### Plugin Capabilityとは意味が異なる

`plugin.CapabilityKind`は、Deployment Executor、Database Provider、Observation Provider、Editor Intelligence等のHost側Capabilityを分類しています。

Application利用者を認証する`authenticate`や`authorize`は、このPlugin Capabilityとは別です。既存認証処理をprocessで実行したい場合でも、「processで動く」という実現方式を理由に、業務上のAuthentication CapabilityとPlugin Capabilityを同じ概念へ畳みません。

### `std`へはまだ固定しない

現時点の`std`は基本型を中心とした標準Semantic libraryです。

Authentication / Authorizationは将来`std.auth.*`等の標準Capabilityになる可能性がありますが、v1の一例だけで標準契約を固定しません。まずReference workspaceの`security`意味空間で実証し、複数Applicationや複数実現方式でも同じ契約として成立するEvidenceが揃ってから標準化を判断します。

## v1で扱う既存認証の姿

小さな既存業務システムを想定し、少なくとも次の状態を持つ認証資産をReferenceへ用意します。

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

さらに、単純なfield対応だけでは表しにくい既存ruleを一つ含めます。

たとえば「特定の組織では有効期間に加えて追加条件を満たす必要がある」といった、既存システムでは珍しくない個別条件です。最終的なruleは、Referenceを不自然に複雑にしない範囲で実装時に確定します。

重要なのは、このschemaをTEI標準のUser schemaへコピーしてから使うことではありません。既存の名前、状態、処理を保持したまま、Applicationから利用できる論理Capabilityへ接続します。

## v1のCapability境界

### `security.authenticate`

credentialをもとにApplication利用者のidentityを確認するCapabilityです。

概念上、次を扱える必要があります。

- login identifier
- credential inputまたはcredential handle
- 認証処理の結果
- 認証されたsubject
- 認証が成立しなかった場合の理由

v1ではADR-0020に従い、認証処理を正常に完了して得られる業務上の結末を名前付きOutcomeとして表します。最初のReferenceでは、概念上`authenticated`と`denied`を区別します。

Database接続失敗やadapter異常など、認証判断そのものを完了できなかった場合はOutcomeへ変換せず、Execution ErrorとしてInvocation境界へ残します。

`subject_id = ""`のようなsentinel値だけで「認証されなかった」を表す設計にはしません。

### `security.authorize`

authenticated subjectが、Application上のaction / resourceを実行できるかを判定するCapabilityです。

概念上、少なくとも次を入力として扱える必要があります。

- authenticated subject
- Application上のaction
- 対象resourceまたはresource category
- 必要な場合のcontext

既存のorganizationやrole、個別ruleは実装側から利用できますが、Web Targetに同じ認可条件を再実装しません。

Authorizationの具体的なOutcome名やpayloadは、AuthenticationのReferenceだけから先回りして固定しません。実装時にはADR-0020の境界を使い、業務上の不成立と実行エラーを分離します。

### AuthenticationとAuthorizationを分ける

ログインできることと、特定の業務操作を実行できることは別の意味です。

一つの`is_allowed` boolean関数へまとめず、Authenticationで得たsubjectをAuthorizationへ渡す構造から始めます。

## 既存schemaとのMapping

既存fieldをTEI標準fieldへ物理コピーすることを前提にしません。

v1では次の実現方式を比較対象とします。

1. Capability implementationが既存schemaを直接読む
2. Persistence Mapping等を通じて既存fieldを論理的な意味へ対応付ける
3. 既存関数、stored procedure、外部API等をadapterから呼ぶ

最初のReferenceでは、少なくとも一つの既存schema経路と、一つのCustom implementation経路を確認します。

どの方式でも、Applicationから見たCapability identityを変えないことを優先します。

## Secretの扱い

password平文、秘密鍵、token等のsecret materialをCanonical Worldへ保存しません。

Referenceでpassword verificationを扱う場合も、Canonical側には「どのCapabilityが何を要求するか」と、secretではない来歴だけを残します。secretの解決と検証はHostまたは既存implementation側の責務です。

少なくとも次へsecretを混入させません。

- `reference/world.json`
- Source Index
- Durable artifact
- 通常のDiagnostic
- 通常の実行log
- Git管理するReference fixture

テスト用credentialが必要な場合も、実秘密情報と混同しないfixtureとして扱います。

## 業務Outcomeと実行エラー

Authenticationでは、次の三つを意味の違いとして扱います。

```text
Authenticated(subject)
Denied(reason)
ExecutionError(error)
```

credential不一致、account disabled、lock等は、Authentication処理を正常に実行した結果としての拒否です。Database接続失敗やadapter異常とは意味が違います。

ADR-0020では、前者をSemantic Capabilityの名前付きOutcome、後者をInvocation境界のExecution Errorとして扱う方針を採用しました。

したがってv1のAuthentication実装では、`authenticated` / `denied`を正常なBusiness Resultとして扱い、`denied`をPlugin Runtimeの`ResponseError`やGo `error`へ変換しません。既存実装の固有return codeはBinding / adapterでSemantic Outcomeへ対応付けます。

この共通境界の実装・検証は`ISSUE-000009` / GitHub Issue #77と[`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md)で進めます。#68だけの場当たり的なerror規約は作りません。

## Application利用者の権限とPlugin権限を分ける

次の二つを混同しません。

```text
Application user authorization
  誰が、どの業務操作を実行できるか

Plugin / Runtime permission
  実装コードが、どの外部resourceへアクセスしてよいか
```

たとえば認証adapterがDatabaseへ接続するにはHost側のResource Authorityが必要になる場合があります。しかし、それは「この利用者が受注確定を実行できるか」という業務認可とは別です。

Driverや将来のWeb Targetから見ても、この二つを独立して追跡できることを条件とします。

## `order_entry`への接続

既存認証Compatibilityを単独fixtureだけで終わらせず、v1 Reference Applicationの`order_entry`へ接続します。

少なくとも次を確認します。

- 受注一覧を閲覧できる利用者
- 受注recordを編集できる利用者
- `sales.order.submit` commandを実行できる利用者

Authorization ruleをWeb Target内のif文として書き直さず、Application / Capability境界から利用します。

最初から複雑なRole Based Access Control製品を作ることは目的ではありません。既存仕様を失わず、Application上のactionへ接続できることを確認します。

## Databaseを第三Datasourceへ固定しない

既存認証情報は、別Database、同一Databaseの別schema、LDAP、IdP、独自API等のどこに存在するか分かりません。

そのため、Authentication Capabilityを「第三Datasourceが必ず存在する」契約にはしません。

Referenceでは実証しやすい既存Databaseを使って構いませんが、Applicationから見た契約はDatabase方式に依存させません。

## Source / Binding / Mappingの来歴

将来AIやEditorが「この認証条件は既に確認済みか」を判断できるよう、次を追跡可能にします。

- Capability identity
- authoritative source
- 採用しているBinding
- legacy field / functionとのMapping
- どの既存ruleを利用しているか
- secretをどの境界で解決するか

ただし、来歴を残すために同じ認証ruleをCanonicalへ複製しません。

## Gate Aへ渡す情報

#68の成果はApplication Pattern #67とGate Aで合流します。

Driverへ渡す情報は少なくとも次です。

- authenticated subjectを表すidentity
- Application上で認可対象となるaction / resource
- Authentication / Authorization Capabilityのidentity
- Capability BindingとMappingの来歴
- Capabilityの名前付きOutcomeとExecution Errorを区別する契約
- 既存仕様のどの条件が認証・認可判断に使われるか
- secret materialをどの境界で解決するか

Gate Aではproduction-readyなIdPやログイン画面を求めません。Webなしでも、既存仕様を保ったままApplication上の意味へ接続できることを確認します。

## 実装順

#68は次の順で小さく進めます。

1. `security`意味空間の最小Referenceを定める
2. Authentication / Authorization Capabilityのidentityと責務を定める
3. ADR-0020に基づく名前付きOutcomeの最小基盤を`ISSUE-000009`で実装・検証する
4. `security.authenticate`へ`authenticated` / `denied` Outcomeを接続する
5. legacy認証fixtureを用意する
6. existing schema / functionとCapabilityを接続する
7. Custom Go等、異なる実現方式を一つ追加する
8. denialとExecution Errorを別経路で再現し、secret非混入をConformance化する
9. `order_entry`のbrowse / edit / submitへAuthorizationを接続する
10. Source / Binding / Mappingの来歴を確認する
11. #67と合わせてGate Aを監査する

## まだ固定しないこと

- `std.auth.*`等の標準Capability名
- TEI標準User schema
- OAuth / OIDC / SAML等の網羅的model
- password hashing algorithm
- production IdPの構成
- Application user sessionの最終model
- refresh token等のTarget / protocol固有設計
- GUI login page
- 認証処理を実行するPlugin CapabilityKind
- Authorizationの最終Outcome語彙

これらはv1 Referenceで本当に必要な境界が見えてから決めます。

## Conformance

少なくとも次を確認します。

- legacy field名をTEIのCanonical identityとして要求しない
- disabled / validity / failure count / lock等の既存意味を失わない
- system固有ruleを標準化のために削らない
- secret plaintextをCanonical / Durable artifact / Source Indexへ保存しない
- AuthenticationとAuthorizationを一つのbooleanへ畳まない
- Application user authorizationとPlugin runtime permissionを混同しない
- 同じCapability identityを保ったままimplementationを交換できる
- implementationを交換してもApplication側の業務上の意味を作り直さない
- `order_entry`のAuthorizationをTarget固有コードへ閉じ込めない
- Source / Binding / Mappingの来歴を追跡できる
- Driverがlegacy schemaの物理構造へ直接依存しない
- `authenticated` / `denied`を名前付きOutcomeとして区別できる
- credential不一致やlockをExecution Errorへ変換しない
- Database / adapter障害を`denied` Outcomeへ変換しない
- ADR-0020のOutcome / Execution Error境界をDriverへ引き継げる

## 非目標

- TEI標準User schemaの制定
- production IdPの内製
- OAuth / OIDC / SAML等の一括対応
- password hashing algorithmの独自実装
- Plugin Permission Grantとの統合
- GUI login pageの完成
- このIssueだけでGate Aを完了させること

## 完了条件

既存認証仕様を変更せず、Authentication / Authorization Capabilityへ接続できるReferenceが成立することを完了条件とします。

さらに、同じCapability identityを保ったまま実現方式を交換でき、`order_entry`のbrowse / edit / submit Authorizationへ接続できること、secretがCanonicalへ漏れないこと、Driverがlegacy schemaへ直接依存しないことをEvidenceとして残します。

Authenticationでは、`authenticated` / `denied`の業務OutcomeとExecution ErrorをADR-0020の境界で分離し、Driverへその意味を保ったまま渡せることも確認します。

`ISSUE-000005`のApplication Patternと合わせて、Gate A「Application上の意味をDriverへ渡せる」を評価します。

## 関連文書

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0019-authentication-capability-boundary.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/design/CAPABILITY_OUTCOME_V1_DESIGN.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000009-capability-outcome-and-execution-error.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md` RQ-004、RQ-008、RQ-009、RQ-012、RQ-017、RQ-019、RQ-021
