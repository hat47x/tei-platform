# Capability Invocation Context Reference

GitHub #1118の評価用Referenceです。

このReferenceは#951のCapability resolver、Provider selection、shape validation、read-only generic Invocationを実装し直しません。既存generic Invocationの手前に置けるcontext gateだけを単独でpressureします。実際のCapability呼出しは`NextInvocation` callbackとして扱います。

確認するのは次の境界です。

- Semantic RequestとHost注入のExecution Contextを別入力にする
- required context slotを明示し、欠落やversion不一致をfail closedする
- Hostが持つ未宣言slotをProviderへ渡さない
- callerのraw Go context valueをProviderへ渡さない
- cancellation / deadline等のcontrol signalは維持する
- raw context valueをInvocation receiptへ保存しない
- Execution Contextがある場合、Host-issued reuse partitionを必須にする
- Semantic Requestが同一でもreuse partitionを別identityとして保持する
- confidential slot値の直接反射を拒否する

## Reference shape

`ContextSlotContract`、`ContextSlotRequirement`、`ContextSlotBinding`はReference上の試験形状です。production schemaとして固定していません。

Hostはslot bindingへ非secretで記録可能なopaque `BindingReceiptRef`と`ReusePartitionRef`を与えます。Referenceは、これらの値が本当に非secretか、trusted sourceから発行されたかを検証しません。token、cookie、raw access ref、Principal payload等をreceipt refとして再利用してはなりません。current authorization / session / revocationの判断は既存のtrusted HostやSecurity境界の責務です。

Provider callbackへは、required slotだけを含む`ContextSlotView`とcontrol-only contextを渡します。control-only contextは元のDeadline / Done / Errを保ちますが、`Value`は公開しません。

これはGo realizationの一例です。TEIの製品契約をGo `context.Context`へ固定しません。

## Confidentiality boundary

`ForbidDirectResultReflection`を持つslotは、raw slot bytesがresultへそのまま含まれる場合にReferenceが拒否します。

この検査は直接反射だけを検出します。encoding、hash、派生情報、side channel等の一般的な情報漏えいを証明しません。そのためreceiptには`direct-reflection-check-does-not-prove-general-non-disclosure`を残します。

productionでは、Context Slot Contractのexposure policy、Provider Conformance、result schema、Security policy等との分担を別途判断します。

## Downstream invocation effect

このcontext gateは、下流Capabilityがread-onlyであることを検証しません。#951のgeneric Invocation Referenceはread-only effectだけを許可しており、本Referenceはその手前へ置く前提です。

direct reflection検査は下流呼出しのresultを受け取った後に行います。mutating Capabilityへ単独で使うと、検査失敗時にも外部副作用が残る可能性があります。

そのためreceiptには`context-gate-does-not-prove-downstream-invocation-is-read-only`を残します。mutating Capabilityのprepare / commit、補償、result validation失敗時の扱いは本Referenceの対象外です。

## Cache boundary

Execution Contextを要求する呼出しでは`ReusePartitionRef`を必須にします。

Reference自身はcacheを実装しません。同じSemantic Requestであっても、Hostが異なるreuse partitionを与えた場合に別partitionとして扱える情報を残すところまでです。

raw tokenやaccess refのhashからpartitionを自動生成しません。partitionの発行と失効はHost側の責務です。

## Authority limits

このReferenceが成功しても次は証明しません。

- objective Truth
- Canonical Acceptance
- authorization policyの正しさ
- session / Principalの真正性
- BindingReceiptRefの真正性
- ReusePartitionRefの安全な発行
- Provider Conformance
- production cacheの安全性
- transformed / indirect information leakの不存在

#1045と#1043をconsumer pressureとして利用し、必要性を確認してからADR化を判断します。
