# ISSUE-000015: `sales.order.submit`のCapability Binding identityをReference化する

- 状態: Done
- GitHub Issue: #126（完了）
- 実装PR: #135
- 上位Issue: `ISSUE-000013` / GitHub Issue #124
- 後段: `ISSUE-000006` / GitHub Issue #69（Application Driver）

## 背景

`command` Application Patternは、次の最小形で成立しています。

```tei
pattern order_entry.submit_order command {
    capability sales.order.submit
}
```

Patternが保持するのはApplication上の意図であり、implementation方式ではありません。一方、Driverが実際にCapabilityを呼び出すには、`sales.order.submit`から実装へ明示的に解決できるBindingが必要です。

このIssueでは、Driverがfile pathやGo hookを暗黙に推測する構造を作らず、既存の`CustomGoBinding`を利用して最小のBinding Referenceを実証しました。

## 実証したBinding identity

現行`CustomGoBinding`は、独立した任意IDを追加せず、ApplicationとCapabilityの組でBindingを識別します。

```text
order_entry / sales.order.submit
```

実装上は次のidentityを利用します。

```text
BindingIdentity(application, capability)
```

Custom Goというimplementation kindはBinding型そのものが表します。

このReferenceでは、第一級のBinding IDを新設しなくても、v1 Driverが必要とする一意なBinding解決とSource追跡を成立させられました。

## 実Reference

PR #135で、`order_entry`に`sales.order.submit`のCustom Go Realizationを追加しました。

Pattern側にはimplementation情報を追加していません。

```text
order_entry.submit_order
        ↓ capability
sales.order.submit
        ↓ Application + Capability
Custom Go Binding
        ↓
reference implementation
```

Application上の`command` identityと、Capabilityをどの方式で実現するかを分離したまま接続しています。

## Driverが利用する解決入口

`ResolveCustomGoBinding`を追加し、ApplicationとCapabilityを明示してBindingを解決できるようにしました。

解決結果は次のように扱います。

- 0件: Binding未定義としてerror
- 1件: 一意なBindingとして解決
- 複数件: ambiguousとしてerror

Driver相当の利用側が、列挙順、file path、Go hook名から実装を選ぶことはありません。

0件または複数件は業務上のBusiness Resultではなく、Capabilityを実行する準備を整えられなかった状態です。後続DriverではADR-0020のExecution Error境界へ接続します。

## Source Index / provenance

Bindingのsource keyは次の形へ整理しました。

```text
binding:<application>/<capability>
```

`realize`宣言中のCapability ID tokenをSource Indexから正確に追跡できます。

file pathをBinding identityにはしていません。Reference fileを改名してもBinding identityは変化せず、authoring sourceだけが新しいpathへ追随することをConformanceで確認しています。

## 複数Applicationと重複Binding

同じCapabilityを異なるApplicationがBindingしても衝突しません。

```text
first_app  / sales.order.submit
second_app / sales.order.submit
```

一方、同じApplicationとCapabilityのBindingを複数定義した場合は、`DUPLICATE_CUSTOM_GO_BINDING`として拒否します。

この実証から、v1時点ではApplication + Capabilityの組をBinding identityとして扱えば十分であることが確認できました。

## 第一級Binding IDを追加しなかった理由

次の要件は、今回のReferenceでは必要になりませんでした。

- 同じApplication / Capability / implementation kindに複数implementationを併存させる
- environmentやreleaseごとにBinding選択を切り替える
- Binding自体へ独立したlifecycle / policy / versionを持たせる

将来これらが実例として必要になった場合は、第一級Binding IDやselection policyを改めて検討します。

将来必要になる可能性だけを理由に、現時点のCanonical modelへ新しいidentityを増やしません。

## Business Result / Execution Errorとの境界

Binding解決やimplementation起動に失敗した状態は、業務上のBusiness Resultではありません。

```text
Capability invocation
  ├─ Business Result
  │   └─ sales.order 等
  └─ Execution Error
      ├─ Binding未定義
      ├─ Bindingが複数あり一意に解決できない
      ├─ implementationを起動できない
      └─ adapter / transport障害
```

Binding failureをnamed Outcomeへ変換しません。

この境界はADR-0020 / `ISSUE-000009` / GitHub Issue #77と整合させます。

## Authorizationとの境界

Capability BindingとApplication利用者のAuthorizationも別の責務です。

- Authorization: 誰がその業務操作を行ってよいか
- Binding: そのCapabilityをどのimplementationで実現するか

Plugin / Runtime PermissionもApplication Authorizationとは別の実行権限境界として維持します。

## Conformance

PR #135で次を確認しました。

- `sales.order.submit`に対する明示的なCustom Go BindingをReferenceから読み込める
- Application + Capabilityから一意なBindingを解決できる
- Binding未定義をerrorとして扱う
- 同じApplication + Capabilityの重複Bindingを拒否する
- 同じCapabilityを異なるApplicationが利用してもidentityが衝突しない
- Binding authoring sourceをSource Indexから追跡できる
- file renameでBinding identityが変わらない
- `command` Patternへimplementation情報を追加しない
- Reference WorldをRepository自身から再生成できる

CI #309は`verify`とPostgreSQL integrationの双方が成功しました。

## 完了条件の確認

1. `sales.order.submit`に対する明示的なBinding / Realization Referenceがある — 完了
2. Application + Capabilityでv1の一意解決が可能かをEvidenceから判断できる — 完了
3. Driverがfile pathやGo hookをBinding discoveryに使わずimplementationを解決できる — 完了
4. Binding source / provenanceを追跡できる — 完了
5. Bindingを追加しても`order_entry.submit_order` PatternのCanonical meaningを変えない — 完了
6. 0件 / 複数件を暗黙選択しない — 完了
7. Binding failureとBusiness Resultを区別する境界を維持できる — 完了
8. 第一級Binding IDを今すぐ追加する必要がないと判断できる — 完了
9. Gate A監査からCapability Binding項目を再評価できる — 完了

## Gate Aへの結論

Capability Binding項目はReadyです。

Driver #69は、`sales.order.submit`から明示的なBindingを解決するためにfile pathやGo symbolを推測する必要がなくなりました。

Gate A全体は、既存認証Compatibility #68、Capability Outcome / Execution Error #77、業務Validation #125が未完了のため、引き続きBlockedです。

## 関連

- GitHub Issue #124 / #126 / #69 / #77
- GitHub PR #135
- `ISSUE-000013`: Gate A到達性監査
- `ISSUE-000006`: Application Driver
- `ISSUE-000009`: Capability Outcome / Execution Error
- ADR-0020
- RQ-003, RQ-008, RQ-012, RQ-017, RQ-018, RQ-021
