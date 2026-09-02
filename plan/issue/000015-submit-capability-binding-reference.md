# ISSUE-000015: `sales.order.submit`のCapability Binding identityをReference化する

- 状態: 着手可能
- GitHub Issue: #126
- 上位Issue: `ISSUE-000013` / GitHub Issue #124
- 後段: `ISSUE-000006` / GitHub Issue #69（Application Driver）

## 背景

`command` Application Patternは、次の最小形で成立しました。

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

これはApplication上の意図として適切です。一方、Driverが実際にCapabilityを呼び出すには、Capabilityからimplementationへ明示的に解決できるRealization / Bindingが必要です。

現在の`order_entry` Referenceで明示されているCustom Go Realizationは`order_entry.compose_order_summary`だけで、`sales.order.submit`のBinding Referenceはまだありません。

Driverの中でfile pathやGo symbolを推測して補うのではなく、既存Realization境界をできるだけ再利用して`sales.order.submit`の最小Binding Referenceを成立させます。

## 現行モデルから分かったこと

現行`CustomGoBinding`は、独立したBinding IDを持たず、次の組でBindingを表しています。

```text
Application
Capability
```

Custom Goというimplementation kindはGoの型そのものが表しています。

adapter file nameとhook nameはCapability identityから導出されます。

そのため、このIssueでは**最初から第一級のBinding IDを新設しません**。

まず既存モデルの

```text
(Application, Capability, implementation kind)
```

をv1 Referenceの論理的なBinding keyとして十分に扱えるかを実証します。

このkeyで不足するEvidenceが得られた場合だけ、独立したBinding identityを追加します。

## 第一段階の仮説

`order_entry`に`sales.order.submit`のCustom Go Realizationを一つだけ追加し、次を成立させることを第一候補とします。

```text
order_entry
  + sales.order.submit
  + custom-go
        ↓
explicit Realization / Binding
        ↓
reference implementation
```

この段階では「同じApplication・Capability・implementation kindに複数の候補がある」状態を作りません。

0件または複数件で一意に解決できない場合は、Driverが偶然の列挙順で選ばずDiagnosticにできることを優先します。

## 第一級Binding IDを追加する条件

次のいずれかが実Referenceで必要になった場合に、独立したBinding IDを検討します。

- 同じApplication / Capability / implementation kindに複数implementationを併存させる必要がある
- environmentやreleaseごとにBinding選択を切り替える必要がある
- Binding自体へ独立したlifecycle / policy / versionを持たせる必要がある
- Source IndexやDiagnosticでCapabilityのimplementation referenceだけでは区別できない
- 外部AdapterやPluginを含む複数方式を同時に選択可能にする必要がある

将来必要になる可能性だけを理由に、今のCanonical modelへ新しいidentityを増やしません。

## Source Index / provenance

Source Indexはpath-independentなSubjectと、`definition` / `authoring` / `implementation`等のroleを分けて扱えます。

そのため第一段階では、Capabilityまたは既存Realizationのlogical Subjectに`implementation` roleのsourceを接続するだけで十分かを確認します。

専用のBinding Subjectが必要だと実Referenceから分かった場合だけ追加します。

file pathをBinding identityそのものにはしません。

## Driverが解決する入口

Driverは`command` Patternから次の順に解決します。

```text
order_entry.submit_order
        ↓ capability
sales.order.submit
        ↓ Application + CapabilityからBinding候補を解決
Custom Go Binding
        ↓
implementation
```

DriverがGo hook名やadapter filenameを自分で組み立て、存在すれば利用するという暗黙解決にはしません。

hook名やadapter filenameを導出する規則は、明示的に解決されたBindingを実行するためのimplementation detailとして扱います。

## 複数implementationの扱い

将来、同じCapabilityを次のような方式で実現できる可能性があります。

- builtin
- generated
- Custom Go
- process plugin
- 既存API

このIssueでは完全なselection policyを先に設計しません。

最初のReferenceは一つの明示Bindingで閉じます。複数候補が必要になった時点で、environment / release / explicit selection等のどの層が選択責務を持つべきかを別のEvidenceとして検討します。

## Business Result / Execution Errorとの境界

Binding解決やimplementation起動に失敗した状態は、業務上のBusiness Resultではありません。

```text
Capability invocation
  ├─ Business Result
  │   └─ sales.order 等
  └─ Execution Error
      ├─ Binding 0件 / 複数件で解決不能
      ├─ implementationを起動できない
      └─ adapter / transport障害
```

この境界はADR-0020 / `ISSUE-000009` / GitHub Issue #77と整合させます。

Binding failureをnamed Outcomeへ変換しません。

## Authorizationとの境界

Capability BindingとApplication利用者のAuthorizationを混同しません。

- Authorization: 誰がその業務操作を行ってよいか
- Binding: そのCapabilityをどのimplementationで実現するか

Plugin / Runtime PermissionもApplication Authorizationとは別の実行権限境界です。

## 守る境界

- `command` Patternへ`custom go`等のimplementation方式を追加しない
- file pathをBinding identityにしない
- DriverがCapability IDからGo hookやfile pathを暗黙推測してBindingを発見しない
- implementation交換でApplication Patternを変更しない
- 一例だけを理由に第一級Binding IDや汎用selection policyを追加しない
- 0件 / 複数件の曖昧なBinding候補を偶然の順序で選ばない
- Plugin / Runtime PermissionとApplication Authorizationを混同しない
- Binding failureを業務Outcomeへ変換しない
- implementation選択規則をWeb Targetへ置かない

## Gate Aへ渡す情報

第一段階では、少なくとも次をDriverから明示的に解決できる状態にします。

- Capability identity `sales.order.submit`
- Application scope `order_entry`
- implementation kind
- 解決されたBinding / Realization
- authoring source / provenance
- DriverがBindingを解決する入口
- Binding解決失敗をExecution Errorとして扱う境界

独立したBinding IDが必要かどうかは、このReferenceの結果から判断します。

## Conformance

少なくとも次を確認します。

- `sales.order.submit`に対する明示的なBinding / RealizationをReferenceから読み込める
- 現行`(Application, Capability, implementation kind)`で一意解決できる
- CapabilityからBindingへ追跡できる
- Bindingからauthoring sourceへ戻れる
- Patternにはimplementation情報が増えない
- Bindingを変更してもPatternのCanonical meaningが変わらない
- Driver相当の利用側がfile pathやGo hookをBinding discoveryに使わない
- 0件 / 複数件を暗黙に選ばない
- Binding解決失敗をExecution Errorとして扱える

## 非目標

- すべてのCapability implementation方式を実装すること
- 第一級Binding IDを将来要件だけから追加すること
- 完成したPlugin marketplace
- production-grade service discovery
- Web Target上のendpoint routing
- Application Authorization
- 汎用workflow engine

## 完了条件

1. `sales.order.submit`に対する明示的なBinding / Realization Referenceがある
2. 現行`(Application, Capability, implementation kind)`でv1の一意解決が可能かをEvidenceから判断できる
3. Driverがfile pathやGo hookをBinding discoveryに使わずimplementationを解決できる
4. Binding source / provenanceを追跡できる
5. Bindingを交換しても`order_entry.submit_order` PatternのCanonical meaningが変わらない
6. 0件 / 複数件で一意に解決できない場合を暗黙選択しない
7. implementation failureとBusiness Resultを区別できる
8. 第一級Binding IDが必要か不要かを実Referenceから判断できる
9. Gate A監査からCapability Binding項目を再評価できる

## 関連

- GitHub Issue #124 / #126 / #69 / #77
- `ISSUE-000013`: Gate A到達性監査
- `ISSUE-000006`: Application Driver
- `ISSUE-000009`: Capability Outcome / Execution Error
- ADR-0020
- RQ-003, RQ-008, RQ-012, RQ-017, RQ-018, RQ-021
