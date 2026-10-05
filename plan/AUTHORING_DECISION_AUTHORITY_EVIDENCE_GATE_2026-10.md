# Authoring Decision Authority Evidence Gate — 2026-10

Status: **ISSUE-000027 / PV-AI-01 authoring decision-authority boundary / Reference-only / no production schema adoption**

## 1. 目的

Shared Acceptanceは、candidate、Review Projection、Accept / Reject Decision、authoritative source更新を分離できるところまで進んだ。

現在はCLI `accept`でexact `reviewFingerprint`を要求し、accepted / rejected Decisionをcontent-addressed ledgerへ残せる。一方、次はまだ証明していない。

```text
who invoked Shared Acceptance
  != who proposed the candidate
  != authenticated decision actor
  != actor authorized to accept / reject this change
```

ここを自己申告の`actor`文字列や`approved=true`で埋めると、provenance metadataをAcceptance authorityへ昇格させてしまう。

本書は、authoring Shared Acceptanceでhuman Decision witnessを扱う前に必要な**Decision Authority Evidence Gate**を定義する。Runtime AuthorizationやBusiness Actorのprincipal modelは流用しない。

## 2. current boundary

### 2.1 Proposal origin

candidateの`origin`は、file / CLI / Studio / AI等の入口とprovenanceを表す。

```text
candidate.origin.actor
  = proposer-side provenance
  != decision actor
```

同じcandidateを別の人がReviewしてAcceptできるため、この二つを同一identityにしない。

### 2.2 Review identity

`reviewFingerprint`は、A1〜A8を含むReview Projectionのexact identityである。

```text
review fingerprint matched
  = the same Projection was bound
  != somebody read it
  != somebody was authorized to decide
```

### 2.3 Accepted / rejected Decision

Decision ledgerは、system上のAccept / Reject結果と理由、candidate / Evidence / Reviewのidentityを保持する。

content-addressや`reviewBindingStatus=verified`は記録内容の整合性を高めるが、判断者のidentityやauthorityを証明しない。

## 3. decision locusとdecision actorを分ける

authoring Decisionでは、少なくとも次を別概念として扱う。

```text
decision locus
  = どの境界がDecisionを受け取ったか
    CLI / Studio / API / future automation

decision actor
  = そのDecisionを明示した主体

decision authority
  = そのactorが対象scopeをAccept / Rejectできる根拠
```

CLI processを起動できたことだけからdecision actorやauthorityを導かない。OS user、environment variable、candidate metadata、AI model identityも自動的なauthority sourceにはしない。

## 4. Evidence Gate

human Decision witnessを`verified`として扱うには、少なくとも次が必要である。

### E1 — exact Decision target

- candidate identity
- exact Review Projection identity
- explicit decision（accept / reject）
- decision reason / provenance

decisionをcandidate内容やReview結果から推測しない。

### E2 — actor identity Evidence

decision actorは、accepted identity providerまたは同等の明示的なidentity Evidenceへ戻れる必要がある。

必要な情報の概念形:

```text
actor identity
issuer / provider identity
authentication or session evidence reference
freshness / expiry where applicable
```

具体的なproviderやwire schemaは本書では固定しない。

### E3 — authoring authority Evidence

authenticated actorであっても、それだけではauthoring Decision authorityにならない。

```text
authenticated actor
  != authorized reviewer
  != authorized source writer
```

authority Evidenceは、少なくとも対象workspace / project、変更scope、許可されたdecision semanticsへ結び付く必要がある。

例として将来検討できるsourceは:

- Studio / organization policy
- repository / SCM review policy
- organization-managed authoring policy
- 明示的に採用されたTEI authoring authority provider

ただし、候補を列挙しただけでいずれかをproduction標準にはしない。

### E4 — scope

authorityはglobal booleanにしない。

少なくとも概念上、次を区別できる必要がある。

- workspace / project
- source layerまたはsubject scope
- accept / reject等のdecision semantics
- 必要ならchange class

別workspaceや別scopeのauthority Evidenceを流用しない。

### E5 — freshness / revocation

過去に有効だったactor identityやauthority Evidenceをcurrent Decisionへ黙って再利用しない。

```text
previously authorized
  != currently authorized
```

session expiry、membership変更、policy revision、revocation等を扱う責務はproviderごとに異なり得るが、Decision時点でcurrentかを確認できなければ`unresolved`を保持する。

### E6 — exact witness binding

上記Evidenceが揃った場合でも、witnessは次へexactにbindingする。

```text
decision actor
+ authority evidence
+ candidate identity
+ review fingerprint
+ explicit decision
+ decision provenance
```

similar candidate、同じsource path、同じactor label等から補完しない。

## 5. gate state

production enumを採用するものではないが、Referenceで少なくとも次を区別する。

```text
authority-unresolved
  identityまたはauthority Evidenceが不足

identity-resolved-authority-unresolved
  actor identityは解決したがauthoring authorityが無い

authority-conflicted
  複数Evidenceが矛盾しwinnerを決められない

authority-qualified
  actor / scope / freshness / decision semanticsを満たす
```

`authority-qualified`も、それだけでsource mutationを完了したことは意味しない。

```text
authority qualified
  != Decision observed
  != Review matched
  != source committed
```

## 6. Runtime Actor Contextとの境界

R3AN / R3AOは、Runtime assertionについて:

```text
qualified acceptance authority evidence
  != accepted

explicit witness observed
  != Accepted Decision applied
```

を固定している。

authoring Shared Acceptanceもこの分離原則を利用する。ただし、R3AN / R3AOのAssertionIdentity、Runtime principal、Authorization targetをauthoring Decisionへ流用しない。

```text
Runtime assertion acceptance authority
  != authoring source Acceptance authority
```

共通化するのは「EvidenceとDecisionを分ける」「exact bindingする」「不明なら補完しない」という原則までである。

## 7. current Referenceの判定

current Repositoryには、Shared AcceptanceのDecision actorについてE2〜E5を満たすaccepted authoring authority sourceがまだない。

したがって現状は:

```text
candidate origin provenance
  -> available

exact Review binding
  -> available

accepted / rejected Decision provenance
  -> available

verified human decision actor
  -> unavailable

verified authoring decision authority
  -> unavailable
```

である。

この状態で次を実装しない。

- `origin.actor`をDecision actorへcopyする
- CLIの`--actor`文字列をverified actorとして保存する
- `approved=true`をhuman approval proofとする
- review fingerprint一致をhuman witnessへ昇格する
- Runtime principalをauthoring reviewerへ自動変換する

## 8. 次のReference pressure

次段では、抽象schemaを先に作らず、**一つの実在するauthoring identity / authority source**を選んでEvidenceを取得できるかをPressureする。

候補はStudioのauthenticated user contextまたはrepository / organization側のreview authorityである。選定時には次を確認する。

1. actor identityが自己申告ではない
2. authority scopeが明示できる
3. current freshness / revocationを確認できる
4. exact Review / candidate / decisionへbindingできる
5. Evidenceを消してもCanonical meaningは変わらない
6. providerを交換してもShared Acceptanceのsemantic candidate / Review / Decision contractを変えない

providerが選べない場合は`authority-unresolved`のまま止める。

## 9. 必須negative pressure

実装へ進む場合、少なくとも次をfail closedで確認する。

- candidate `origin.actor`しか無い
- actor identityだけありauthority Evidenceが無い
- authority Evidenceのscopeが別workspace
- authority Evidenceのdecision semanticsが対象decisionを許可しない
- authority Evidenceがexpired / revoked / stale
- actor identityとauthority Evidenceのsubjectが一致しない
- review fingerprintがDecision対象と一致しない
- accept / reject witnessが競合する
-複数authority Evidenceが競合しwinnerを決められない
- actor / authority Evidenceを削除した後に、過去Decisionをcurrent verified witnessとして読み替える

## 10. Product Valueとの接続

このGateの目的は承認手順を重くすることではない。

- AIやStudioが提案者になっても、判断者を混同しない
- 人がReviewしたという見かけのmetadataでauthorityを捏造しない
- 数か月後のAIがDecision historyへ再入場したとき、「何が決まったか」と「誰が決める権限を持っていたか」を別々に検証できる
- Runtime permission modelをauthoringへ誤流用しない
- 将来のidentity providerを交換してもCanonical meaningとShared Acceptanceの中核を保つ

ための境界である。

## 11. Current decision

```text
self-asserted actor
  -> provenance only

reviewFingerprint
  -> exact Projection identity only

current Shared Acceptance Decision
  -> system Decision provenance

verified human Decision witness
  -> Not Ready

authoring Decision Authority Evidence Gate
  -> required before verified witness

production actor / authority schema
  -> not adopted
```
