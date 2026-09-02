# Gate A 到達性監査

## 目的

この文書は、v1 Reference Application `order_entry`をApplication Driverへ渡す前に、Gate A「Application上の意味をDriverへ渡せる」の現在地を横断的に確認する監査結果です。

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

Application Pattern側はReadyです。また、PR #132によって名前付きOutcomeのCanonical基盤も成立し、Capability Outcome領域は`Blocked`から`Partial`へ進みました。

一方、Authentication / Authorization、業務Validation、`sales.order.submit`のBinding ReferenceはまだGate Aの必要条件を満たしていません。名前付きOutcomeについても、`security.authenticate`実ReferenceとInvocation結果境界が未成立です。

現時点でDriver #69の主実装へ進むと、これらの意味をDriver専用コードへ暗黙に埋め込むおそれがあります。

現在の主線は次です。

```text
#68   既存認証Compatibility ───────┐
#77   Capability Outcome ─────────┤  Canonical基盤は成立、Reference / Invocationが残る
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
| named OutcomeのCanonical契約 | Ready | PR #132。model / parser / validation / normalization | なし |
| named Outcomeの実Reference / Invocation | Partial | Outcome型とDSLは成立 | `security.authenticate` Reference、denial / Execution Error経路、Invocation結果契約 #77 |
| Authentication | Blocked | ADR-0019 / #68の設計のみ | #68 |
| Authorization | Blocked | 境界設計はあるがReferenceなし | #68 |
| 業務Validation | Blocked | Pattern定義のValidationとは別。業務ruleのReferenceなし | #125 |
| Capability Binding | Blocked | `sales.order.submit`の実Referenceはまだmain未反映 | #126 |
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

## Capability結果境界 — Partial

ADR-0020により、意味上の境界は次のように定めています。

```text
Business Result
  ├─ existing output
  └─ named Outcome

Execution Error
  └─ invocation自体を完了できない状態
```

`sales.order.submit`については、既存`output sales.order`を維持するEvidenceがあります。

PR #132では、名前付きOutcomeのCanonical基盤がmainへ入りました。

成立した範囲は次です。

- `semantic.Capability`が名前付きOutcomeを保持できる
- OutcomeはCapability内のlocal nameとpayload型を持つ
- `outcome <name> <payload-type>`をParserから扱える
- 既存`output`と名前付きOutcomeを同時に宣言できない
- Outcome名の重複を拒否する
- payload型の存在を検査する
- Outcomeの宣言順をCanonicalな意味にしない
- 既存の単一`output` Capabilityをそのまま維持する

このため「名前付きOutcomeをSemantic Capabilityで表現できる」というGate A条件はReadyへ進めます。

ただし、Gate Aで必要な結果境界全体はまだReadyではありません。次が残っています。

- `security.authenticate`を実Referenceとして定義する
- `authenticated` / `denied`を実際のBusiness Resultとして再現する
- expected denialをExecution Errorへ変換しない
- Database / adapter / Binding等の実行障害をdenialへ変換しない
- Invocation側で既存`output`とnamed Outcomeを一つのBusiness Result境界から扱える
- Driverへ渡せる結果契約をReference / Conformanceから説明できる

第一Web TargetでのHTTP statusや利用者向けmessageへのProjectionはGate A以後へDeferredします。

## Authentication / Authorization — Blocked

Reference Workspaceには、Gate Aで必要な`security` Authentication / Authorization Referenceがまだ成立していません。

Issue #68では、既存認証仕様をTEI標準User schemaへ作り直さず、Authentication / Authorization Capabilityへ接続することを求めています。

Gate Aに必要なのはproduction-readyなIdPやログイン画面ではありません。少なくとも次が必要です。

- authenticated subject identity
- Authentication Capability identity
- Authorization Capability identity
- Application上のaction / resourceとの接続
- legacy field / functionからCapabilityへのMapping
- secret materialをCanonicalへ保存しない境界
- named Outcome / Execution Errorの実結果契約

#77のCanonical基盤が一段進んだため、#68ではこの基盤を使う最小Referenceへ進めます。

## 業務Validation — Blocked

現在の`internal/validation`は、Canonical WorldやPatternの構文・参照整合性を検査する仕組みとして成立しています。

一方、Driver Scenarioが扱う業務Validationの正本はReference Workspaceにありません。

#125 / `ISSUE-000014`では、最初の実証例を次まで具体化しています。

```text
sales.order_line.quantity > 0
```

および

```text
sales.order.lines
  └─ lineNumber is unique within one order
```

前者は単一属性の業務rule、後者はRelationで結ばれたDetail集合のscopeを必要とするruleです。

この二例から、Semantic側の不変条件、Application-owned Validation、独立したCapability / Policy相当の境界のどこが自然かを比較します。汎用rule engineや式言語を先に導入しません。

## Capability Binding — Blocked

`command` Patternは`sales.order.submit`というCapability identityだけで自然に成立しました。

#126 / `ISSUE-000015`では、現行`CustomGoBinding`が持つApplication + Capabilityを第一候補のBinding identityとして使い切り、独立した任意IDを先に増やさない方針まで具体化しています。

並行PR #135では、この仮説に沿う実Referenceが実装されています。ただし、Gate A監査はmainへ反映されたEvidenceを正とするため、PR #135がmainへ入るまでは`Blocked`を維持します。

確認する主な条件は次です。

- `sales.order.submit`に対する明示的なBinding / Realization Referenceがある
- Driverがfile pathやGo hookをBinding discoveryに使わない
- 0件 / 複数件を暗黙選択しない
- Binding source / provenanceを追跡できる
- Binding交換で`command` PatternのCanonical meaningが変わらない
- Binding解決失敗をBusiness Outcomeへ変換しない

## Driver開始判定

現時点ではDriver #69の主実装を開始しません。

Gate Aを再監査する主な契機は次です。

1. #68 — Authentication / Authorization Reference
2. #77 — `security.authenticate` ReferenceとInvocation結果境界
3. #125 — 業務Validation Reference
4. #126 — `sales.order.submit` Binding Reference

Datasource read / write、5 Pattern、単一output Business Result、named OutcomeのCanonical契約、Source Index / identityは既にReadyです。

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
- GitHub PR #132 / #135
- ADR-0019 / ADR-0020
