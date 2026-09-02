# Capability Business Result / Execution Error Conformance

## 目的

このConformanceは、Semantic Capabilityを正常に実行した結果と、実行そのものを完了できなかった状態を、同じ経路へ混在させないことを確認するものです。

対象はIssue #77およびADR-0020で定めたv1の最小境界です。

## 正本

業務上の結末はSemantic Capabilityの`output`または名前付き`outcome`を正本とします。

```text
Capability
├─ output
└─ outcome <name> <payload-type>
```

実行時には、正常に完了した結果を`BusinessResult`として返します。Database接続失敗、Binding解決失敗、adapter障害、契約にないOutcomeなどはBusiness Resultへ変換せず、Execution ErrorとしてGo `error`で返します。

```text
Invoke Capability
       │
       ├─ 正常に実行できた
       │      ↓
       │   BusinessResult
       │      ├─ output
       │      └─ named outcome
       │
       └─ 実行を完了できなかった
              ↓
            error
```

## Authentication実Reference

Reference Workspaceには`security.authenticate`を置きます。

```tei
capability security.authenticate {
    input loginId std.identifier
    input credential security.credential
    outcome authenticated security.authenticated_subject
    outcome denied security.authentication_denial
}
```

このCapabilityでは、credential不一致やaccount lockを`denied`として扱います。どちらも認証処理を正常に完了した結果であり、Execution Errorではありません。

一方、Databaseやadapterの障害は`denied`へ変換しません。

## 実行時Conformance

`internal/semantic/capabilityinvoke`のテストでは、Reference WorkspaceからCanonical Capabilityを直接読み込み、次を確認します。

| 条件 | 期待結果 |
| --- | --- |
| credentialが正しい | `authenticated` Business Result |
| credentialが一致しない | `denied` Business Result |
| accountがlockされている | `denied` Business Result |
| Database / adapter障害 | Execution Error |
| 未定義Outcomeを返す | Execution Error |

正常な拒否をGo `error`だけで表さず、逆に実行障害を`denied`へ押し込みません。

## 既存output Capabilityの互換性

Authenticationだけに合わせた抽象になっていないことを確認するため、既存の`sales.order.submit`も同じInvocation境界へ通します。

このCapabilityは従来どおり次の単一outputを持ちます。

```tei
capability sales.order.submit {
    input order sales.order
    output sales.order
}
```

同じ`Invoke`入口から`output` Business Resultを受け取れることを確認します。

また、作者が定義していない`submitted`等のOutcomeを暗黙に追加することは認めません。単一output Capabilityが名前付きOutcomeを返した場合はExecution Errorとして扱います。

## Payload境界

v1のInvocation境界は、Payloadの具体的な業務データObject modelを定義しません。

Canonical CapabilityはOutcome名とpayload型の意味を保持しますが、Go上の値表現、legacy値からの変換、外部API responseからの変換はAdapter側の責務です。

これはDriverより先に汎用的な値モデルを固定しないための意図的な制約です。型付きPayloadの共通表現が複数の実Referenceから必要になった場合に、別途設計します。

## 非目標

このConformanceでは、次を扱いません。

- HTTP statusやWeb画面へのProjection
- 汎用的な例外階層
- 汎用sum type
- legacy認証schemaへの接続
- Authorization
- secret解決
- Observationによる外部状態確認

legacy認証との接続はIssue #68で、この結果境界を前提として実証します。

関連: #68 / #69 / #77 / #124 / ADR-0020
