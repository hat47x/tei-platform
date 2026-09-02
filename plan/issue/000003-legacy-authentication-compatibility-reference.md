# ISSUE-000003: 既存認証仕様を保持したCompatibility Referenceを設計する

- 状態: Completed
- GitHub Issue: #68（完了）
- 実装・設計基線: [`LEGACY_AUTH_V1_DESIGN.md`](../LEGACY_AUTH_V1_DESIGN.md)
- 結果境界: ADR-0020 / `ISSUE-000009`
- Gate A: `ISSUE-000013`（Ready）

## 背景

業務システムの移行では、標準的なID / password認証であっても、ユーザー無効化、有効期限、ログイン失敗回数、lock、組織属性、role、個別ruleなど、システム固有の制御が長年積み重なっています。

TEIが既存システム移行に使えるためには、先に標準認証へ作り替えることを要求せず、既存仕様を保持したままApplicationから利用できるCapabilityへ接続する必要があります。

## 最終結果

既存認証仕様を変更せず、`security.authenticate` / `security.authorize`へ接続するReferenceが成立しました。

さらに、同じ`security.authenticate` Capabilityと`order_entry/security.authenticate` Binding identityを維持したまま、既存recordを読む方式と既存認証関数を呼ぶ方式を交換できることを確認しました。

ApplicationやDriverは、legacy schema、return code、role名を直接知る必要がありません。

## Authentication

Canonicalな認証契約は`security.authenticate`です。

```tei
capability security.authenticate {
    input login_id std.identifier
    input credential security.credential
    outcome authenticated security.authenticated_subject
    outcome denied security.authentication_denial
}
```

正常に認証判断を完了して得られる`authenticated` / `denied`はBusiness Resultです。

Database、adapter、既存関数等の障害はExecution Errorです。credential不一致やaccount lockをExecution Errorへ変換しません。

## 既存user recordを読む方式

`legacyauth.Adapter`では、既存システム側の次の意味を失わず利用できます。

- user ID
- subject identity
- credential reference
- disabled
- validity period
- login failure count
- lock状態
- organization
- role
- system-specific rule

これらをTEI標準User schemaへ物理コピーしてから利用する方式にはしていません。

## 既存認証関数を呼ぶ方式

PR #151では、既存関数、stored procedure、外部API等が認証判断まで完了する方式を`FunctionAdapter`として実証しました。

legacy固有のreturn codeはReference Mappingから`authenticated` / `denied`へ対応付けます。

return code自体をTEI標準Outcomeにはしません。未対応codeや既存関数の呼び出し障害はExecution Errorとして残します。

Mapping sourceの来歴を追跡できますが、secretは保存しません。

## 同一identityでの実装交換

Application側のBindingは変わりません。

```text
order_entry/security.authenticate
```

既存record方式と既存function方式の双方が、同じCanonical `security.authenticate`契約を満たします。

実装を交換しても、Application Pattern、Binding identity、Custom Go hook、Reference WorldのCanonicalな意味を作り直しません。

複数implementationを同時登録して動的選択する汎用機構は、このEvidenceだけを理由に追加していません。

## Authorization

`security.authorize`を`order_entry`へ接続しました。

最初のReferenceでは、Application Pattern identityを認可resourceとして利用します。

- `browse` / `order_entry.recent_orders`
- `edit` / `order_entry.order_record`
- `execute` / `order_entry.submit_order`

Authenticationで得たsubject identityから既存user recordを引き直し、legacy organization / role / ruleを実装側で評価します。

ApplicationやDriverへlegacy roleを公開して認可ruleを再実装させません。

`allowed` / `denied`はBusiness Resultで、Storeやrule評価の障害はExecution Errorです。

## Secretの扱い

password平文、password hash、token、credential hash等を次へ保存しません。

- `reference/world.json`
- Source Index
- Durable artifact
- 通常Diagnostic
- 通常log / Evidence
- Git管理するReference fixture

Canonical側にはCapability契約、Binding、secretではないMappingの来歴だけを残します。

提示credentialは実行時だけVerifierまたは既存認証関数へ渡します。

## Plugin / Runtime Permissionとの分離

Application利用者のAuthentication / Authorizationと、Plugin / Runtimeに与えるPermission Grant / Resource Leaseは別の責務です。

認証実装がprocess等で動く場合も、「誰が業務操作を実行できるか」と「実装コードがどの外部resourceへアクセスできるか」を同じ権限モデルへ畳みません。

## Gate Aへ渡した情報

- authenticated subject identity
- Authentication / Authorization Capability identity
- Application上のaction / resource
- Capability Binding / Mappingの来歴
- Business Result / Execution Error契約
- legacy条件を評価する実装境界
- secretを解決する境界
- 同一Capability identityで実現方式を交換できるEvidence

これらはGate A監査 `ISSUE-000013`でReadyと判定され、Driver #69の着手条件に利用されます。

## 完了条件の確認

- legacy field名をCanonical identityとして要求しない — 完了
- disabled / validity / failure count / lock等の既存意味を失わない — 完了
- system固有ruleを標準化のために削らない — 完了
- secret plaintextをCanonical / Durable artifact / Source Indexへ保存しない — 完了
- AuthenticationとAuthorizationを一つのbooleanへ畳まない — 完了
- Application user authorizationとPlugin runtime permissionを混同しない — 完了
- 同じCapability identityを保ったままimplementationを交換できる — 完了
- implementationを交換してもApplication側の意味を作り直さない — 完了
- `order_entry`のAuthorizationをTarget固有コードへ閉じ込めない — 完了
- Source / Binding / Mapping / implementationの来歴を追跡できる — 完了
- Driverがlegacy schemaの物理構造へ直接依存しない — 完了
- 業務上の拒否とExecution ErrorをADR-0020の境界で区別できる — 完了

## 非目標として維持するもの

- TEI標準User schemaの制定
- `std.auth.*`の即時標準化
- production IdPの内製
- OAuth / OIDC / SAML等の一括対応
- password hashing algorithmの独自実装
- Plugin Permission Grantとの統合
- GUI login page
- 複数implementationを同時選択する汎用policy

## 関連

- `plan/LEGACY_AUTH_V1_DESIGN.md`
- `plan/GATE_A_READINESS_AUDIT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/adr/0019-authentication-capability-boundary.md`
- `plan/adr/0020-capability-business-outcome-boundary.md`
- `plan/issue/000009-capability-outcome-and-execution-error.md`
- `plan/issue/000013-gate-a-readiness-audit.md`
- GitHub Issue #68 / #69 / #124
- GitHub PR #145 / #148 / #149 / #151
