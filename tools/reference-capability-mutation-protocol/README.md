# Mutating Capability Protocol Reference

generic Capability Invocationをread-only以外へ広げる前に、副作用とresult validationの順序を検証するReferenceです。

ISSUE-000080 E4では、read-only Capabilityだけをgeneric Invocationへ通しています。実装呼出し後にresult shapeを検査する方式をmutating Capabilityへそのまま使うと、resultをrejectしても外部副作用だけが残るためです。

このReferenceでは、次の順序を候補として検証します。

```text
semantic request
  -> prepare
  -> prepared result
  -> result validation
  -> durable commit intent journal
  -> commit
  -> committed / not-committed / unknown
  -> reconcile when needed
```

## prepareとcommitの境界

`Prepare`は外部から観測できる副作用を確定してはいけません。Providerはprepared result、request digest、result digest、Concurrency Tokenを返します。

Hostはprepared resultをcommit前に検査します。resultがshapeやdomain ruleを満たさない場合はcommitせず、prepared stateをabortします。

result検査後はcommit対象のpayload-free handleをJournalへdurable commit intentとして記録します。Journalへの記録に失敗した場合もcommitしません。commit後にHost processが停止しても、Invocation Keyからprepared handleを復元し、Providerのstatusへ戻るためです。

durable handleへprepared resultの生bytesは入れません。保持するのはrequest digest、result digest、preparation identity、Concurrency Token等の回復情報です。result payloadに機密情報が含まれる場合、回復性のためだけにその内容をJournalへ永続化することを避けます。

再起動後のreconcileはcommit outcomeの確定までを責務とします。結果payloadが再度必要な場合は、別のObservation / Query境界から取得します。Commandの回復と結果データの再取得を同じdurable recordへ押し込みません。

ReferenceのJournalはmemory実装です。永続性、暗号化、複製、障害復旧を実証したものではありません。productionでは、commit前に回復可能な記録を残せる保存境界が別途必要です。

## idempotencyとretry

Invocation Keyはrequest digestと組にして扱います。

同じInvocation Keyと同じrequestを再試行した場合、Providerは同じ論理mutationとして扱う必要があります。同じInvocation Keyを別requestへ再利用した場合は、commit前に拒否します。

このReferenceはexactly-onceを保証したとは主張しません。安全なretryにはProvider側のidempotency、prepared state、commit status、durable Journalが必要です。

commit transportが失敗した場合、Hostは`not-committed`へ読み替えません。`unknown`を返し、自動で同じ副作用を再実行しません。

```text
commit request
  -> response lost
  -> unknown
  -> status / reconcile
       -> committed
       -> not-committed
       -> unknown
```

statusでも確定できない場合は`unknown`を維持します。

### point-of-no-return

result validationとrequest cancellationの確認を終えた後、Hostはcommit intentをJournalへ記録します。この記録が成功した時点をpoint-of-no-returnとします。

point-of-no-returnより前にrequestがcancelされた場合はprepared stateをAbortし、commitしません。commit intentを記録した後にcallerがcancelしても、Hostはそのcancelをcommit可否の判断へ使いません。commit / status reconciliationをcoordinator責務として完遂します。

そのためcommit呼出しにはcaller cancellation / deadlineを外したContextを使います。これはAuthorityを拡張するものではありません。どのProviderをどの権限で実行するかは、point-of-no-returnより前に確定したExecution Bindingへ固定します。

## concurrency

prepared resultはprepare時点の状態に基づきます。

Reference ProviderはConcurrency Tokenを返し、commit時に現在状態と照合します。同じstateから二つのmutationをprepareし、一方が先にcommitした場合、もう一方は古いprepared resultをcommitせず`not-committed`にします。

この仕組みはDatabase transaction方式をCore仕様へ固定するものではありません。optimistic version、transaction handle、lease、fencing token等で同じ境界を実現できる余地を残します。

## commit receipt

committed resultはprepare時に確定したresult bytesとdigestへ結び付けます。commit応答がprepared identityと一致しない場合、Hostは副作用が既に発生した可能性を否定できないため`unknown`として扱います。

Provider receiptには少なくとも次を結びます。

- Invocation Key
- request digest
- preparation identity
- result digest
- effect digest
- commit identity

Host側のMutation ReceiptはProvider receiptとprepared fingerprintへ戻れるようにします。Receiptもresult payload自体は保持せず、result digestだけを保持します。

## fingerprintとbyte rule

request / result digestは入力byte列そのものへSHA-256を適用します。文字列化、改行正規化、Unicode正規化等はこのReference内で暗黙に行いません。

prepared / Provider receipt / Mutation receiptのfingerprintは、field境界を持つstructをJSONへmarshalした後にSHA-256を適用します。NUL区切り連結等の曖昧なfield結合は使いません。canonicalization規則はReference candidateとして明示し、将来のproduction wire formatとは分離します。

## cleanup

request cancellationがprepare後に発生しても、未commit prepared stateのcleanupまで同じcancel済みContextで止めないようにします。Abortはrequest Contextの値を保ちつつcancellation / deadlineを外したcleanup Contextでbest-effort実行します。

これはcompensationではありません。commit前の一時resource解放だけを対象にします。

Abortに失敗した場合も、business outcomeは`not-committed`のままです。ただしresource leakの可能性を隠さず、cleanup errorを独立して返します。

## Journal integrity

同じInvocation Keyに同じcommit intentを再保存することはidempotentに許容します。一方、別bindingや別prepared identityを同じInvocation Keyへ上書きしてはいけません。Journal implementationはconflictとして拒否する必要があります。

Journalから読み戻したhandleやoutcome receiptも無条件には信用しません。binding、request / result digest、preparation fingerprint、receipt fingerprintを再検証し、改ざんまたは破損したrecordはProviderへ到達する前に停止します。

## Journal failure

commit intentの記録に失敗した場合はcommitしません。

一方、commit成功後にoutcome receiptの保存だけが失敗した場合、副作用を「失敗したから無かった」とは扱いません。Referenceは`committed` outcomeとJournal errorを同時に返します。同じInvocation Keyで再入場すればProviderのidempotencyによって副作用を増やさず回復できるpressureを置いています。

## compensation

compensationをgeneric Invocationの暗黙動作にはしません。

commit済みの業務効果を取り消せるかどうかはCapabilityごとの意味です。補償操作が存在しても、元のcommitが無かったことにはなりません。また、`unknown` outcomeに対して「たぶんcommitした」と判断してcompensationを自動実行すると、実際には未commitだった場合に逆方向の副作用を作れます。

compensationを導入する場合は、独立したCapability / Procedure / Authorityとして扱う必要があります。

## Runtime Authorityとの関係

このReferenceはPermission、Resource Lease、Sandbox、Plugin Lockを実装し直しません。

generic mutating protocolが成立しても、Application Runtimeで実行するにはRQ-019のAuthority境界を別に通る必要があります。逆にRuntime Authorityがあっても、prepare / result validation / commit / reconciliationの原子性問題は解決しません。

```text
Capability Contract / Provider / Conformance
  -> Runtime Authority
  -> mutation protocol
  -> external effect
```

各境界は別の責務です。

### Execution Binding

mutation protocolはCapability ResolutionやRuntime Permissionを再計算しません。上位で確定した実行経路をopaqueなexact SHA-256 fingerprintとして受け取ります。

Execution Binding fingerprintはPrepare、Commit、Status、durable handle、Provider receipt、Mutation Receiptへ伝播します。process再起動後のreconcileでも、current ExecutorのbindingとJournal内handleのbindingが一致しなければProviderへ到達する前に拒否します。

このfingerprint自体は新しいAuthorityではありません。将来productionへ接続する場合は、ISSUE-000080側のCapability Resolution identityとRuntime Invocation Provenanceから導出するpressureが必要です。

## Referenceで確認すること

Reference testsでは次をpressureします。

- prepared resultを検査する前にcommitしない
- commit intentをJournalへ記録できなければcommitしない
- commit成功時だけprepared resultを返す
- 同じInvocation Keyのretryで副作用を重複させない
- 同じInvocation Keyを別requestへ使わない
- commit response喪失後は`unknown`を保持する
- Journalからpayload-freeなprepared handleを復元し、process再起動後もoutcomeをreconcileできる
- malformedなcommit receiptを`not-committed`へ読み替えない
- statusが`unknown`なら`unknown`を維持する
- stale Concurrency Tokenでprepared resultをcommitしない
- commit後のJournal failureをrollback済みと誤認しない
- commit intent記録後のcaller cancellationでcommit decisionを破棄しない
- 同じInvocation Keyへ別bindingのcommit intentを上書きしない
- Journalから読み戻したhandle / receiptを再検証する
- Execution Binding driftをProvider.Status前に拒否する
- Abort cleanup failureをbusiness outcomeと分けて観測する

## 非目標

このReferenceだけでは次を成立済みと扱いません。

- production generic mutating Invocation
- exactly-once delivery / execution
- durable Journal implementation
- distributed transaction
- automatic compensation
- Database transaction contract
- process / HTTP / Wasm transport
- Permission / Resource / Sandbox integration
- crash-safe fsync / replication
- actual external serviceとのmutation Evidence

コード変更を含むReference candidateです。exact-head focused testとfull Repository healthが得られるまでproduction contractへ昇格しません。
