# ISSUE-000087: Mutating Capabilityのprepare / commit安全境界をReference化する

- 状態: In Progress
- 起票日: 2026-10-10
- GitHub Issue: #1157
- 親: ISSUE-000080 / GitHub #951
- 関連: RQ-019 / RQ-020 / RQ-021 / RQ-032

## 背景

ISSUE-000080 E4では、generic Invocationへ載せるCapabilityをread-onlyへ限定した。

requestは実装呼出し前に検査できる。一方、result shapeは実装呼出し後にしか検査できない。mutating implementationが外部へ書込みを行った後にresultをrejectすると、呼出し元からは失敗に見えても副作用だけが残る。

Permission、Resource Lease、Sandboxは実行可否を制御する。result validationと外部副作用の原子性は別問題である。

## 減らす不確実性

generic mutating Capabilityを、result validation、retry、crash recovery、unknown outcomeを壊さずに呼び出すための最小protocolをReferenceで確認する。

production transaction APIを先に固定しない。Referenceで必要になった境界だけを残す。

## Reference candidate

`tools/reference-capability-mutation-protocol/`で次の順序をpressureする。

```text
request
  -> prepare
  -> prepared result
  -> result validation
  -> prepared intent journal
  -> commit
  -> committed / not-committed / unknown
  -> reconcile
```

### Prepare

Prepareは業務上の副作用を確定しない。Providerはpreparation identity、Invocation Key、request digest、prepared result、result digest、concurrency token、preparation fingerprintを返す。transaction handle、lease、lock等の可逆な内部resourceは許容するが、commit前ならAbortまたはtimeoutで解放できることを前提とする。

Hostはprepared resultをcommit前に検査する。検査に失敗した場合はcommitしない。

### Journal

result validation後、commit前にprepared handleをJournalへ記録する。記録できない場合はcommitしない。

process停止後もInvocation Keyからprepared handleを復元し、Providerのstatusへ戻れる必要がある。

durable handleへprepared result payloadの生bytesは入れない。Journalはrequest / result digest、preparation identity、concurrency token等の回復情報を保持する。再起動後のreconcileはcommit outcomeを確定し、result payloadの再取得が必要な場合は別のObservation / Query境界へ委ねる。

Referenceのmemory Journalは永続性Evidenceではない。productionではdurable storage、暗号化、複製、retention等を別に決める必要がある。

### Commit / Reconcile

Commitはprepared identity、request digest、result digest、concurrency tokenを入力にする。

commit後にtransport errorが起きた場合、Hostは`not-committed`へ読み替えない。結果は`unknown`とし、同じ副作用を自動replayしない。

Provider receiptがprepared identityと一致しない場合も、副作用が既に発生した可能性を否定できないため`unknown`とする。

`unknown`になったmutationは、Journalからprepared handleを復元してProvider statusへ戻る。statusでも確定できなければ`unknown`を維持する。

Journalにcommitted / not-committed receiptが既にあれば、Provider commitを再実行せずJournalからoutcomeを再構築する。

## Idempotency

Invocation Keyはrequest digestと組にして扱う。同じInvocation Keyと同じrequestの再試行は同じ論理mutationとして扱う。別requestへ同じInvocation Keyを使う場合はcommit前に拒否する。

このReferenceはexactly-onceを保証したとは主張しない。安全なretryにはProvider側のidempotency、durable prepared state、status reconciliationが必要である。

## Concurrency

prepared resultはprepare時点のstateに基づく。ReferenceではConcurrency Tokenを使い、同じstateから複数mutationをprepareした後にstateが変化した場合、古いprepared resultをcommitしない。

実現方式としてoptimistic version、transaction handle、lease、fencing等を許容し、特定Database transaction APIをCoreへ固定しない。

## Outcome

最低限、`committed`、`not-committed`、`unknown`を区別する。

`unknown`は失敗の言い換えではない。外部状態を確認できるまで副作用の有無を断定しない。

commit後にoutcome receiptのJournal保存だけが失敗した場合も、副作用をrollback済みとは扱わない。current callerにはcommitted outcomeとJournal errorを同時に返し、同じInvocation Keyから再入場できる余地を保つ。

## Fingerprint / cleanup

request / result digestはraw bytesへSHA-256を適用する。prepared / Provider receipt / Mutation receiptのfingerprintはfield境界を保ったstructをJSONへmarshalしてからSHA-256を適用する。NUL区切り連結等の曖昧なfield結合は使わない。

prepare後にrequestがcancelされた場合も、Abort cleanupへ同じcancel済みContextを渡さない。Hostの値を保ちつつcancel / deadlineを外したcleanup Contextで、未commit stateの解放をbest-effort実行する。

このAbortはcommit済み効果を打ち消すcompensationではない。

## Compensation

compensationをgeneric Invocationの暗黙動作にしない。commit済み効果を取り消せるかはCapability固有の意味である。

`unknown`に対して自動compensationすると、実際には未commitだった場合に逆方向の副作用を作り得る。compensationが必要な場合は、独立したCapability / Procedure / Authorityとして扱う。

## Runtime Authority

このReferenceはPermission、Resource Lease、Sandbox、Plugin Lockを再実装しない。

Application Runtimeでmutating Capabilityを実行する場合はRQ-019のAuthority境界を別に通る必要がある。Runtime Authorityがあることも、prepare / validate / commit / reconcileの原子性を証明しない。

## 敵対的pressure

- invalid prepared resultでcommitしない
- prepared intentをJournalへ記録できなければcommitしない
- successful commitだけがprepared resultを公開する
- 同じInvocation Keyのretryで副作用を増やさない
- 同じInvocation Keyを別requestへ使わない
- commit response喪失を`unknown`にする
- malformed post-commit receiptを`not-committed`へ畳まない
- process再起動後にJournalからreconcileできる
- Journal済みoutcomeではProvider commitを再実行しない
- statusが`unknown`なら`unknown`を維持する
- stale concurrency tokenをcommitしない
- commit後のJournal failureをrollback扱いしない
- cancel済みrequestをprepare / commitしない

## 完了条件

- prepare / validate / journal / commitの順序をExecutable Referenceで確認できる
- idempotency keyとrequest identityを固定できる
- commit transport failureを`unknown`として保持できる
- payload-freeなJournal handleからprocess再起動後のoutcome reconcileへ戻れる
- stale prepared resultをcommitしない
- committed effectとJournal / receipt failureを混同しない
- compensationを暗黙recoveryとして扱わない
- Runtime Authorityとmutation protocolを別責務として説明できる

## 非目標

- production generic mutating Invocationの有効化
- exactly-once保証
- production durable Journal
- distributed transaction
- automatic compensation
- Database固有transaction API
- process / HTTP / Wasm transport
- Permission / Resource / Sandbox integration
- actual external serviceへのwrite Evidence

## Evidence境界

現段階はReference candidateである。

Repository exact-headのfocused testとfull health checkは未取得である。memory Journalはdurabilityを証明しない。in-memory ledgerもactual external mutationのEvidenceではない。

したがってRQ-019 / RQ-020のEvidence tierは変更しない。production contractへも昇格しない。
