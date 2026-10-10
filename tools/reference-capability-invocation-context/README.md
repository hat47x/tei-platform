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

Hostはslot bindingへ非secretで記録可能なopaque `BindingReceiptRef`と`ReusePartitionRef`を与えます。Referenceは、これらの値が本当に非secretか、trusted sourceから発行されたかを一般的に検証できません。token、cookie、raw access ref、Principal payload等をreceipt refとして再利用してはなりません。current authorization / session / revocationの判断は既存のtrusted HostやSecurity境界の責務です。

防御を一段追加し、`ForbidDirectResultReflection`のある必須slotの**raw bytesがそのまま**、公開されるReusePartitionRefやいずれかの必須BindingReceiptRefに含まれる場合は、Providerを起動する前に拒否します。他の必須slotのreceipt ref経由でも拒否し、未要求slotの壊れたrefには影響されません。エラー文にraw値は含めません。これは明白な誤設定を検出する局所対策であり、短い値の部分一致で正当な識別子を拒む可能性があります。変換・符号化・ハッシュ等による漏えいや、Host-issued identityの真正性まで保証しません。

Provider callbackへは、required slotだけを含む`ContextSlotView`とcontrol-only contextを渡します。control-only contextは元のDeadline / Done / Errを保ちますが、`Value`は公開しません。

ReferenceのGo callbackは同一packageから実行されるため、`ContextSlotView`のunexported map/byte sliceに直接書き込む攻撃的な実装も考えられます。実際に、Providerがslot bytesを書き換えてから元のsecretをresultへ返すと、以前の直接反射検査を回避できることを負例で再現しました。現在は**Host側の照合用snapshotとは別に、Provider専用のmapとbyte sliceをdeep copy**して渡します。Callbackが値やbinding refを変更・削除しても、Hostの照合用値とreceiptは変更されません。Go package構成への依存を避ける防御ですが、Providerに渡したsecretを一般的に隠す仕組みではなく、同時書込みによるdata raceや外部副作用を防ぐものでもありません。

Referenceは、呼出し開始時にSemantic Requestのbytesを複製し、下流へ渡す入力とreceiptのrequest digestを同じsnapshotに基づかせます。呼出し中にcallerが元のsliceを書き換えてもreceiptのidentityは変化しません。ただし、snapshot取得と同時に別goroutineが書き込むdata raceまで許容するものではありません。

呼出しのdispatch前にControl Contextがすでにcancelされていれば、下流を呼び出さずに終了します。dispatch後のcancelはControl Contextのsignalとして伝えます。また、下流が戻った時点でtrustedなControl Contextの終了状態を確認し、cancel / deadlineが成立していれば、**Providerが成功結果を返してもresultとreceiptを公開せず終了します**。下流の停止を強制できるわけではなく、検査後からcallerへの返却までに発生する競合を原理的に解消するものでもありません。外部副作用がすでに発生した場合の巻戻しも保証しません。

これはGo realizationの一例です。TEIの製品契約をGo `context.Context`へ固定しません。

## Confidentiality boundary

`ForbidDirectResultReflection`を持つ必須slotについて、成功したresult payloadにraw slot bytesが直接含まれていれば拒否します。**downstream errorについては`Error()`自体が任意のコードを実行し得るため、文字列を検査せず原error objectごと破棄して一律のopaque failureへ変換します。**

この検査は成功resultの直接反射だけを検出します。error経路は一律opaque failureで遮断します。encoding、hash、派生情報、Provider内部log、別goroutineのpanic、side channel等の一般的な情報漏えいを証明しません。そのためreceiptには`direct-reflection-check-does-not-prove-general-non-disclosure`を残します。

downstream errorの`Error()`文字列が安全に見えても、元のGo error objectをそのまま返すと、`errors.As`、`Unwrap`、custom formatting、構造体fieldから秘匿情報を回収できる。実際に、秘密をfieldへ保持し`Error()`は無害な文字列を返す型の漏えいを負例で再現した。

現Referenceでは、成功resultの直接反射を検出した場合に拒否し、機密required slotを持つ呼出しがdownstream errorを返した場合は**`Error()`や`Unwrap()`を一切呼び出さず、原error objectを破棄して固定failureへ置き換える**。この境界を越える同期的なProvider panicも、値を文字列化せず同じ固定failureへ変換する。非機密slotだけの呼出しでは元のerrorおよびGoのpanic伝播規則を維持する。panicの自動抑止を全Capabilityへ一般化しない。

ただし、単純にerrorを固定文言へ置き換えるだけでは、下流実行中に発生したControl Contextの`context.Canceled` / `context.DeadlineExceeded`まで失われる。実際に両者の区別が消えることを再現し、修正した。現在は、**trustedなControl Contextの`Err()`が返す制御状態のみ**を優先して返す。下流errorが`context.Canceled`をwrapしているだけでは信頼せず、Control Contextが生きている場合は固定failureへ変換する。元の下流error objectは、どちらの場合も返さない。

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
