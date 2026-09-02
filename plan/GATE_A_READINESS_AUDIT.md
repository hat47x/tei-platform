# Gate A 到達性監査

## 目的

この文書は、v1 Reference Application `order_entry`をApplication Driverへ渡す前に、Gate A「Application上の意味をDriverへ渡せる」の現在地を横断的に確認した監査結果です。

Application Pattern候補5種類は、すべて実Referenceまで成立しました。しかし、Patternが揃ったことと、Driverを開始できることは同じではありません。

DriverがWeb画面や専用Goコードから意味を補わなくても、必要なApplication上の情報をRepositoryから解決できるかを基準に評価します。

## 評価区分

| 区分 | 意味 |
| --- | --- |
| **Ready** | Gate Aに必要な範囲をReference / Conformanceから再現できる |
| **Partial** | 重要な境界は成立しているが、Driverへ渡す情報が不足している |
| **Blocked** | Driverへ進む前に独立した実証が必要である |
| **Deferred** | 必要な作業だがGate A以後でよい |

## 結論

**Gate A全体の現在判定はBlockedです。**

Application Pattern側はReadyですが、Authentication / Authorization、named Outcome、業務Validation、`sales.order.submit`のBinding identityがDriver開始条件を満たしていません。

現時点でDriver #69の主実装へ進むと、これらの意味をDriver専用コードへ暗黙に埋め込むおそれがあります。

したがって、次の4作業をGate Aの主線とします。

```text
#68   既存認証Compatibility ───────┐
#77   Capability Outcome ─────────┤
#125  業務Validation ─────────────┤
#126  Capability Binding ─────────┤
                                   ↓
                               Gate A再監査
                                   ↓
                               Driver #69
```

## Gate A項目別評価

| 項目 | 判定 | 現在のEvidence | 残る作業 |
| --- | --- | --- | --- |
| browse対象 | Ready | `order_entry.recent_orders` | なし |
| record対象 | Ready | `order_entry.order_record` | なし |
| reference対象 | Ready | `sales.order.customer` + datasource `master` | なし |
| header-detail関係 | Ready | `sales.order.lines -> sales.order_line many` | なし |
| command対象 | Ready | `sales.order.submit` | なし |
| Pattern identity / Source | Ready | Source Index / rename invariance / Golden | なし |
| Datasource read / write | Ready | `sales` read-write / `master` read、reference read検査 | Driverでも同じ正本を利用する |
| Business Resultの単一output | Ready | `sales.order.submit -> sales.order` | named Outcomeとは別に維持する |
| named Outcome | Blocked | ADR-0020の設計のみ | #77 |
| Authentication | Blocked | ADR-0019 / #68の設計のみ | #68 |
| Authorization | Blocked | 境界設計はあるがReferenceなし | #68 |
| 業務Validation | Blocked | Pattern定義のValidationとは別。業務ruleのReferenceなし | #125 |
| Capability Binding | Blocked | `sales.order.submit`の明示Bindingなし | #126 |
| Command / Observation分離 | Ready | 既存Runtime / DeploymentのEvidence。Patternにも混在させていない | Driverで再利用する |
| Target固有Presentation | Deferred | Gate Aでは不要 | 第一Web Target #70 |

## Application Pattern — Ready

Issue #67は完了しました。

`order_entry`では次の5種類が同じApplicationで共存します。

```tei
pattern order_entry.recent_orders browse {
    data sales.order
    query order_entry.recent_orders
}

pattern order_entry.order_record record {
    data sales.order
}

pattern order_entry.customer_reference reference {
    relation sales.order.customer
    datasource master
}

pattern order_entry.submit_order command {
    capability sales.order.submit
}

pattern order_entry.order_lines header-detail {
    relation sales.order.lines
}
```

Pattern identity、Source Index、共通Validation、Golden再現性までRepository内で確認済みです。

5種類の実証から、Patternは既存資産の意味を書き直す場所ではなく、Applicationがどの既存identityをどの意図で利用するかを保持する層として整理できました。

## Datasource read / write — Ready

`order_entry`は二つのlogical datasourceを利用します。

```text
sales  -> sales_core  -> read-write
master -> master_data -> read
```

`reference` Patternでは、Datasource accessをPatternへ複製せずApplication定義を正本としながら、読み取り不能なDatasourceをValidationで拒否できます。

Gate Aでは、この境界を新しいDriver設定へコピーせず、Applicationの`DatabaseRequirement`から解決できることを条件とします。

## Capability結果境界 — Partial / Blocked

ADR-0020により、意味上の境界は決まっています。

```text
Business Result
  ├─ existing output
  └─ named Outcome

Execution Error
  └─ invocation自体を完了できない状態
```

`sales.order.submit`については、既存`output sales.order`を維持するEvidenceがあります。

一方、現行Semantic modelにはnamed Outcomeがまだ実装されていません。Authenticationで必要な`authenticated` / `denied`をReference化するには、#77の実装Evidenceが必要です。

Gate Aでは#77全体の完了を機械的に要求しません。第一Web TargetへのProjectionなどはDeferredできますが、少なくとも次は必要です。

- named OutcomeをSemantic Capabilityで表現できる
- 既存`output`との排他境界を保持できる
- expected denialをExecution Errorへ変換しない
- DriverがBusiness Resultとして扱える結果契約がある

## Authentication / Authorization — Blocked

Reference WorkspaceのSemantic Namespaceには現在`sales`と`customer`があり、`security` Referenceはまだ成立していません。

Issue #68では、既存認証仕様をTEI標準User schemaへ作り直さず、Authentication / Authorization Capabilityへ接続することを求めています。

Gate Aに必要なのはproduction-readyなIdPやログイン画面ではありません。少なくとも次が必要です。

- authenticated subject identity
- Authentication Capability identity
- Authorization Capability identity
- Application上のaction / resourceとの接続
- legacy field / functionからCapabilityへのMapping
- secret materialをCanonicalへ保存しない境界
- named Outcome / Execution Errorの結果契約

このため#68はGate Aのblockerです。

## 業務Validation — Blocked

現在の`internal/validation`は、Canonical WorldやPatternの構文・参照整合性を検査する仕組みとして成立しています。

一方、Driver Scenarioが扱う「数量が業務上有効か」「HeaderとDetailの組み合わせが成立するか」といった業務Validationの正本は、Reference Workspaceにありません。

同じValidationという語を使っていても、この二つは役割が異なります。

#125では、少なくとも一項目型とHeader / Detailをまたぐruleを使い、次を実証します。

- ruleの正本をどこに置くか
- Driverが専用Goコードへruleをコピーしないこと
- Validation不成立を正常な業務結果として扱えること
- Web Targetが同じ結果を表示へProjectionできること

## Capability Binding — Blocked

`command` Patternは`sales.order.submit`というCapability identityだけで自然に成立しました。

しかし、現在の`order_entry/realizations/default.tei`で明示されているのは`order_entry.compose_order_summary with custom go`だけで、`sales.order.submit`にはDriverが利用できる明示的なBinding Referenceがありません。

#126では、Patternへimplementation方式を持ち込まず、Capability identityからBinding / implementationを追跡可能に解決できることを実証します。

Gate Aで必要なのは特定のCustom Go実装を標準化することではなく、**Driverがfile pathやGo symbolを暗黙に推測しなくてよいこと**です。

## Driver開始判定

現時点ではDriver #69の主実装を開始しません。

次の必須項目がReadyになった時点でGate Aを再監査します。

1. #68 — Authentication / Authorization Reference
2. #77 — Gate Aに必要なnamed Outcome / Execution Error境界
3. #125 — 業務Validation Reference
4. #126 — `sales.order.submit` Binding Reference

Datasource read / write、5 Pattern、単一output Business Result、Source Index / identityは既にReadyです。

## Gate A以後へDeferredするもの

次は重要ですが、Gate Aのblockerにはしません。

- named Outcomeの第一Web Target表現
- HTTP status / UI messageへのProjection
- production-ready IdP
- production HA
- Target-neutral Presentationの最終抽象
- 5 Patternの共有Catalog化
- `OrderLine`の物理Persistence / FKを一般化すること
- production transaction coordinator

## 次回監査

#68 / #77 / #125 / #126のEvidenceが増えたら、この表を更新します。

すべてを一度にReadyへ寄せず、項目ごとにEvidenceの強さを確認します。

Gate Aを通過した時点で、`ISSUE-000006` / GitHub Issue #69を主実装へ移し、最小Driver Scenarioを開始します。

## 関連

- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/APPLICATION_PATTERN_V1_DESIGN.md`
- `plan/issue/000003-legacy-authentication-compatibility-reference.md`
- `plan/issue/000005-application-pattern-reference.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000009-capability-outcome-and-execution-error.md`
- `plan/issue/000013-gate-a-readiness-audit.md`
- `plan/issue/000014-business-validation-reference.md`
- `plan/issue/000015-submit-capability-binding-reference.md`
- GitHub Issue #67 / #68 / #69 / #77 / #124 / #125 / #126
- ADR-0019 / ADR-0020
