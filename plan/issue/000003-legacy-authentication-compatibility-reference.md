# ISSUE-000003: 既存認証仕様を保持したCompatibility Referenceを設計する

- 状態: 着手可能
- GitHub Issue: #68

## 背景

業務システムの移行では、標準的なID / password認証であっても、ユーザー無効化、有効期限、ログイン失敗回数、ロック、組織属性など、システム固有の制御が長年積み重なっています。

TEIが既存システム移行に使えるためには、最初に標準認証へ作り替えることを要求せず、既存仕様を保持したままCapabilityへ接続できる必要があります。

Product Requirements Gap Auditでも、RQ-009はv1へ向けた大きな未実証領域として確認されました。

このIssueは、v1 Reference Application実証計画ではApplication Patternと並行して進めます。両者の成果はGate A「Application上の意味をDriverへ渡せる」で合流します。

## 関連するProduct Requirement

- RQ-004 Native asset
- RQ-008 Capability単位の置換
- RQ-009 既存認証・認可仕様の保持
- RQ-015 AIによる既存知識の探索
- RQ-017 二重管理の最小化
- RQ-019 Resource authorityとの責務分離
- RQ-021 Reference / Conformance

## 目的

TEI標準のUser tableや認証schemaを先に定義するのではなく、**既存システム固有の認証状態と処理を、その意味を失わずAuthentication / Authorization Capabilityへ接続できること**を実証します。

ここで扱うApplication利用者の認証と、Plugin / RuntimeのPermission Grant / Resource Leaseは別の責務です。両者を同じ「権限」概念へ畳みません。

## Referenceの既存認証モデル

小さな既存システムを想定した認証資産を用意します。

最低限、次の既存fieldを持つものとします。

```text
LegacyUser
- user_id
- password_hash / credential reference
- disabled
- valid_from
- valid_until
- login_failure_count
- locked_until
- organization_code
- role_code
```

さらに、単純な標準化では消えやすいシステム固有ruleを一つ入れます。

例として、特定の状態や組織条件を含むログイン可否判定を用意します。最終的なruleは、Referenceを過度に人工的にしないものを選びます。

## Secretの扱い

passwordの平文やcredential secretをCanonical definitionへ保存しません。

Referenceでpassword verificationを扱う場合も、Canonical側には「どのCapabilityが何を要求するか」を置き、secret materialはHostまたは既存implementation側の責務として扱います。

## Capability Contract

最初から巨大なIAM modelを作らず、少なくとも次の二つを分けて検討します。

### authenticate

与えられたcredentialをもとに利用者identityを確認するCapabilityです。

概念上は次の情報を扱える必要があります。

- login identifier
- credential input / handle
- authentication result
- authenticated subject
- failure reason category

### authorize

authenticated subjectが、Application上のaction / resourceを実行できるか判定するCapabilityです。

組織やroleなどの既存情報を利用できる必要があります。

AuthenticationとAuthorizationを一つのboolean関数へ固定しません。

## Mapping

既存fieldをTEI標準fieldへ物理コピーすることを前提にしません。

次の方式を比較します。

1. Capability implementationが既存schemaを直接読む
2. 既存fieldを論理的な情報へMappingする
3. 既存関数、stored procedure、external APIをCapabilityとして呼ぶ

Referenceでは少なくとも一つの既存schema Mappingと、一つのCustom implementation経路を実証します。

## 実現方式の交換

RQ-008も同時に検証するため、同じCapability contractについて次のうち少なくとも二方式を比較します。

- existing implementation
- Custom Go Binding
- process Plugin
- 将来のbuiltin / generated implementation

最初から全方式を作る必要はありません。

重要なのは、Applicationから見たCapability identityを変えずにimplementationを差し替えられることです。

## `order_entry`との接続

Reference単体で境界を確認した後、v1 `order_entry`へ接続します。

Application上では、少なくとも次を確認します。

- browseを利用できる利用者
- recordを編集できる利用者
- `sales.order.submit` commandを実行できる利用者

Authorization ruleをWeb Target内のif文として再実装せず、Application / Capability境界から利用します。

## Database / external boundary

Legacy authentication dataを第三DatabaseとしてReference Workspaceへ追加するか、外部Capabilityとして扱うかは、実証上どちらが意味を明確にするかで決めます。

`order_entry`へ第三Datasourceを先に追加することを要件にはしません。

既存システムでは認証が別DB、LDAP、IdP、独自APIなどのいずれもあり得るため、Databaseに固定しないCapability契約を優先します。

## Gate Aへ渡す情報

このIssueでは、Driverが認証製品やlegacy schemaの内部構造を知らなくても、Application上のAuthorizationを評価できる境界を用意します。

Gate Aへ渡す情報は少なくとも次です。

- authenticated subjectを表すidentity
- Application上で認可対象となるaction / resource
- authorize Capabilityのidentity
- Capability BindingとMappingの来歴
- 既存仕様のうち、どの条件が認証・認可判断に使われるか
- secret materialをどの境界で解決するか

Gate Aではproduction-readyなIdPやログイン画面の完成は求めません。既存仕様を保ったままApplication上の意味へ接続できることを確認します。

## AI / Editorへの引き継ぎ

将来AIやEditorが認証要件を確認するとき、既にMapping済みのfieldやruleを再質問しないためには、採用済みのCapability BindingとMappingを機械的に確認できる必要があります。

このIssueではAI UI自体は実装しませんが、後から「何が決定済みか」を追跡できる構造を壊さないことを条件とします。

## Conformance

少なくとも次を確認します。

- legacy field名をCanonical identityとして要求しない
- disabled / validity / failure count / lockなどの既存意味を失わない
- secret plaintextをCanonical / Durable artifactへ保存しない
- AuthenticationとPlugin runtime permissionを混同しない
- 同じCapability identityへ複数implementationをBindingできる
- implementationを交換してもApplication側の業務上の意味を作り直さない
- `order_entry`のcommand authorizationをTarget固有コードへ閉じ込めない
- Source / Binding / implementationの来歴を追跡できる
- Driverがlegacy schemaの物理構造へ直接依存せずAuthorizationを評価できる

## 非目標

- TEI標準User schemaの制定
- production IdPの内製
- OAuth / OIDC / SAMLなどを一度に網羅すること
- password hashing algorithmの独自実装
- Plugin Permission Grantとの統合
- GUI login pageの完成
- このIssueだけでGate Aを完了させること

## 完了条件

既存認証仕様を変更せず、Authentication / Authorization Capabilityへ接続できるReferenceが成立すること。

さらに、同じApplication Capability identityのまま実現方式を交換でき、`order_entry`のbrowse / edit / command Authorizationへ接続できることを示します。

`ISSUE-000005`のApplication Patternと合わせて、Gate A「Application上の意味をDriverへ渡せる」を評価できるEvidenceを残します。

## 関連文書

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
