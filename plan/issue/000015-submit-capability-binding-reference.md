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

これはApplication上の意図として適切です。一方、Driverが実際にCapabilityを呼び出すには、Capability identityからimplementationへ解決できるBindingが必要です。

現在の`order_entry` Referenceで明示されているRealizationは`order_entry.compose_order_summary with custom go`であり、`sales.order.submit`のBinding Referenceはまだありません。

Driverの中でfile pathやGo symbolを推測して補うのではなく、実行前に追跡可能なBinding identityを成立させます。

## 目的

Application Patternが保持するCapability identityと、そのCapabilityを実現するimplementationを分離したまま、Driverが明示的にBindingを解決できることを実証します。

```text
order_entry.submit_order
        │
        │ capability
        ▼
sales.order.submit
        │
        │ explicit Binding
        ▼
implementation
```

Pattern自身へ`custom go`等のimplementation方式を追加しません。

## 最初に確認すること

- 現行`CustomGoBinding`をReferenceとして利用できるか
- Application-scoped BindingとCapability-scoped Bindingのidentityをどのように追跡するか
- 同じCapabilityに複数implementationが存在する場合、選択責務をどの層に置くか
- Driverがfile pathやGo symbol名を直接推測せずBindingを解決できるか
- Bindingが変わっても`order_entry.submit_order`のCanonical meaningが変わらないか
- Binding authoring sourceへSource Index / Diagnosticから戻れるか

## 最小Reference

既存Realization境界を再利用し、`sales.order.submit`を小さいreference implementationへBindingすることを第一候補とします。

Custom Goは最初の実証手段として利用できますが、Custom Go自体をTEIの標準実装方式として固定することは目的ではありません。

最初のReferenceでは、**Capability identityとimplementation identityを別々に保持し、明示的に接続できること**を確認できれば十分です。

## 複数implementationの扱い

将来、同じCapabilityを次のような複数方式で実現できる可能性があります。

- builtin
- generated
- Custom Go
- process plugin
- 既存API

このIssueでは、完全なselection policyを先に設計しません。

ただし、複数Bindingが存在する場合にDriverが偶然の列挙順やfile pathから選ばないことは必要です。Referenceで複数候補が必要になった場合は、明示的な選択境界を追加します。

## Business Result / Execution Errorとの境界

Binding解決やimplementation起動に失敗した状態は、業務上のBusiness Resultではありません。

```text
Capability invocation
  ├─ Business Result
  │   └─ sales.order等
  └─ Execution Error
      ├─ Bindingを解決できない
      ├─ implementationを起動できない
      └─ adapter / transport障害
```

この境界はADR-0020 / `ISSUE-000009`と整合させます。

Binding failureをnamed Outcomeへ変換しません。

## Authorizationとの境界

Capability BindingとApplication利用者のAuthorizationを混同しません。

- Authorization: 誰がその業務操作を行ってよいか
- Binding: そのCapabilityをどのimplementationで実現するか

Plugin / Runtime Permissionも、Application Authorizationとは別の実行権限境界です。

## 守る境界

- `command` Patternへimplementation方式を追加しない
- Capability IDからGo関数名やfile pathをDriverが暗黙推測しない
- implementation交換でApplication Patternを変更しない
- Plugin / Runtime PermissionとApplication Authorizationを混同しない
- Binding failureを業務Outcomeへ変換しない
- implementation選択規則をWeb Targetへ置かない

## Gate Aへ渡す情報

- Capability identity `sales.order.submit`
- Binding identity
- implementation kind
- authoring source / provenance
- DriverがBindingを解決する入口
- Binding解決失敗をExecution Errorとして扱う境界

## Conformance

少なくとも次を確認します。

- `sales.order.submit`に対する明示的BindingをReferenceから読み込める
- Capability identityからBindingへ追跡できる
- Bindingからauthoring sourceへ戻れる
- Patternにはimplementation情報が増えない
- Bindingを変更してもPatternのCanonical meaningが変わらない
- Driver相当の利用側がfile pathやGo symbolを推測せずBindingを選べる
- Binding解決失敗をExecution Errorとして扱える

## 非目標

- すべてのCapability implementation方式を実装すること
- 完成したPlugin marketplace
- production-grade service discovery
- Web Target上のendpoint routing
- Application Authorization
- 汎用workflow engine

## 完了条件

1. `sales.order.submit`に対する明示的なBinding Referenceがある
2. DriverがPatternやfile pathからimplementationを推測せずBindingを解決できる
3. Bindingを交換してもPatternのCanonical meaningが変わらない
4. Binding authoring sourceを追跡できる
5. implementation failureとBusiness Resultを区別できる
6. Gate A監査からCapability Binding項目をReadyと判定できる

## 関連

- GitHub Issue #124 / #126 / #69 / #77
- `ISSUE-000013`: Gate A到達性監査
- `ISSUE-000006`: Application Driver
- `ISSUE-000009`: Capability Outcome / Execution Error
- ADR-0020
- RQ-003, RQ-008, RQ-012, RQ-017, RQ-018, RQ-021
