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

Referenceは、呼出し開始時にSemantic Requestのbytesを複製し、下流へ渡す入力とreceiptのrequest digestを同じsnapshotに基づかせます。呼出し中にcallerが元のsliceを書き換えてもreceiptのidentityは変化しません。ただし、snapshot取得と同時に別goroutineが書き込むdata raceまで許容するものではありません。

呼出しのdispatch前にControl Contextがすでにcancelされていれば、下流を呼び出さずに終了します。dispatch後のcancelはControl Contextのsignalとして伝えます。下流の停止を強制できると保証するものではありません。

これはGo realizationの一例です。TEIの製品契約をGo `context.Context`へ固定しません。

## Confidentiality boundary

`ForbidDirectResultReflection`を持つslotは、raw slot bytesがresult payloadまたはdownstream error messageへそのまま含まれる場合にReferenceが拒否します。

この検査はresult / errorへの直接反射だけを検出します。encoding、hash、派生情報、Provider内部log、side channel等の一般的な情報漏えいを証明しません。そのためreceiptには`direct-reflection-check-does-not-prove-general-non-disclosure`を残します。

downstream errorの`Error()`文字列が安全に見えても、元のGo error objectをそのまま返すと、`errors.As`、`Unwrap`、custom formatting、構造体fieldから秘匿情報を回収できる。実際に、秘密をfieldへ保持し`Error()`は無害な文字列を返す型の漏えいを負例で再現した。

現Referenceでは、直接反射を検出した場合は従来どおり拒否する。それ以外でも`ForbidDirectResultReflection`のあるrequired slotが存在し、下流がerrorを返した場合は**原error objectを返さず**固定の汎用failureへ置き換える。非機密slotだけの呼出しでは元のerrorを保持する。

これはReference上の防御策であり、製品のerror codeやCapability固有のfailure semanticsを決定するものではない。productionではContractに従う公開可能なfailure表現を別途設計する。固定メッセージへの変換だけで間接漏えいやProvider内部logを排除したとは主張しない。

productionでは、Context Slot Contractのexposure policy、Provider Conformance、result schema、Security policy等との分担を別途判断します。

## Semantic Request shapeとの分担

このcontext gateはSemantic Requestのpayloadを解釈・書き換えません。Execution Contextらしいfieldを見つけて削除するSanitizerではありません。

たとえばrequest bytesに`grantedAccessRefs`というfieldが含まれていても、gateはそのbytesを下流へそのまま渡します。安全性は、#951側のrequest-shape validationやCapability固有のstrict decoderが、semantic contractに存在しないexecution-context fieldを拒否することに依存します。

#1045の`DetailQuery`はauthorization access refをschemaへ持たず、CLIも別の`-detail-access`入力へ分離しているため、この条件をpressureできる。

したがってreceiptには`context-gate-relies-on-capability-request-shape-to-exclude-execution-context-fields`を残します。context gate単体のPASSを、request schemaの安全性へ一般化しません。

また、`ContextSlotRequirement`の集合はconsumer payloadから作る値ではなく、resolved Capability Contractからtrusted Hostが取得した入力であることを前提とします。current ReferenceはそのContract Authorityやfingerprint binding自体を検証しないため、`context-requirements-are-trusted-contract-input-not-consumer-payload`も明示します。

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
