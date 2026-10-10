# Business Authoring Authority Cutover Pressure — 2026-09

Status: **Lane C / migration authority pressure / test-only / no production source replacement**

## 1. 目的

Current Referenceでは、production candidateへ移行する前のpressure apparatusとして次が別sourceに存在する。

```text
legacy Procedure source
legacy protected-target O1 overlay
Reference Set
  - Procedure artifact assembly
  - Handoff meaning
```

一方、current strongest production authoring candidatesは:

```text
Procedure source
  + inline singular protectedApplicationPattern?

standalone Handoff source
```

である。

Migration previewでlegacy sourceを読むこと自体は必要だが、cutover時にlegacyとcandidateの両方をauthoritative sourceとして有効化すると、同じBusiness meaningに二つの正本が生じる。

本Pressureは、**legacy inputを読むこと**と**legacy authorityを維持すること**を分離する。

## 2. Authority cutover phases

Current conceptual sequence:

```text
Phase 0 — legacy authority
  legacy Procedure
  legacy O1 overlay
  legacy Reference Set Handoff

Phase 1 — migration preview
  read legacy authoritative sources
        ↓
  derive candidate
    inline O1 Procedure source
    standalone Handoff source
        ↓
  normalize / compare / cross-view validate

  authority remains legacy only
  candidate = proposal / preview

Phase 2 — Shared Acceptance
  source freshness
  Conformance base freshness
  migration losslessness
  authority conflict check
        ↓
  atomic cutover

Phase 3 — candidate authority
  inline O1 Procedure source
  standalone Handoff source

  legacy overlay / Reference Set Handoff
    -> no longer accepted Business authority
    -> may remain Reference / migration Evidence if clearly non-authoritative
```

新sourceを生成した時点をauthority切替とみなさない。

## 3. O1 dual-authority pressure

Current BP-06では:

```text
legacy O1 overlay
  order-entry.order-handling / BP-06
    -> order_entry.submit_order

candidate inline O1
  same Procedure / Step
    protectedApplicationPattern
      -> order_entry.submit_order
```

意味が同じ場合でも両方をauthorityにしない。

```text
same meaning in two sources
  != safe duplicate
```

理由:

- future target-only changeでどちらが正か不明になる
- freshness / approvalが二つへ分裂する
- source reload順で意味が変化し得る
- RQ-003のimplicit multiple SOTを再導入する

さらにlegacy側とcandidate側のtargetが異なる場合、migration merge / last-writer-wins / candidate preference等で自動解決しない。Authority conflictとしてfail closedする。

## 4. Handoff dual-authority pressure

Current Handoffはlegacy Reference Setとcandidate standalone sourceで同じendpoint relationを表せる。

```text
legacy Reference Set Handoff
  BP-09 -> FU-00 + condition

candidate standalone Handoff
  BP-09 -> FU-00 + condition
```

これも両方をauthorityにしない。

特にendpointが同じでconditionだけ異なる場合:

```text
legacy condition A
candidate condition B
```

を二本のBusiness Handoffとして増殖させず、migration authority conflictとしてfail closedする。

Current production candidateはhandoff-only relationであり、同じendpointへ複数conditionを持つ一般的Business Flow languageを導入していないためである。

## 5. Migration previewはdual authorityではない

Migration previewではlegacy sourcesを読み、candidateを生成してよい。

```text
legacy authority
  ↓ read
candidate preview
  ↓ normalize / compare
same Business meaning
```

この段階でcandidateはauthoritative sourceではない。

したがって:

```text
legacy input present
+ candidate object exists
  != dual authority
```

である。

問題になるのは:

```text
legacy accepted as authority
+ candidate accepted as authority
```

を同時に許す場合だけである。

## 6. Test-only Pressure

`business_authoring_authority_cutover_pressure_test.go`で次を追加する。

### C1 — preview may read legacy sources

Legacy O1 / Reference Set Handoffを読みcandidateを生成しても、candidate非authoritativeなら許可する。

Candidate自体はcurrent normalized Business modelとしてload可能であることを確認する。

### C2 — O1 same-meaning dual authority reject

Inline O1とlegacy overlayがsame Procedure / Stepをauthorする場合、target値が同じでもrejectする。

### C3 — O1 conflicting dual authority reject

同じProcedure / Stepでtargetが異なる場合も、自動mergeせずrejectする。

### C4 — Handoff same-meaning dual authority reject

Standalone Handoffとlegacy Reference Set Handoffがsame endpointをauthorする場合、conditionが同じでもrejectする。

### C5 — Handoff conflicting dual authority reject

Same endpoint / different conditionでも自動的に二本へ展開せずrejectする。

### C6 — candidate-only after cutover

Legacy authorityを無効化した後、candidate-only source bundleをreload / normalizeできることをPressureする。

Actual Repository runner greenまではexecution Evidenceではない。

## 7. Reference filesを今削除しない理由

Current legacy filesはReference / pressure Evidenceとして価値がある。

特に:

- sidecarがcomposite Acceptanceを要求すること
- Reference Set manifestとBusiness Handoff meaningの混在問題
- old -> candidate migration losslessness

を再現するために使っている。

したがってactual production cutover前にReference artifactを削除・置換しない。

重要なのはfileの存在ではなく、production loader / Shared Acceptanceが何をauthoritative sourceとして読むかである。

```text
file exists
  != accepted authority
```

## 8. Schema versionとの関係

本Pressureからproduction `schemaVersion`値やmigration manifestを決めない。

必要なのは意味上:

```text
old authority mode
  -> migration preview
  -> one accepted cutover
  -> new authority mode
```

が識別できることである。

Production implementationがrepository revision / source registry / loader configuration等のどれでauthority modeを決めるかはShared Acceptance実装へ委譲する。

Lane Cからdual-reader transition modeを長期production contractへしない。

## 9. Shared Acceptanceとの接続

Authority cutoverは`docs/audits/BUSINESS_MODEL_PRODUCTION_ACCEPTANCE_BOUNDARY_AUDIT_2026-09.md`の:

```text
Business source freshness
Business semantic revision
cross-view Conformance base
atomic change-set
reload / Conformance
```

に追加して、migration時だけ次を要求する。

```text
migration candidate lossless
+ no simultaneous legacy/candidate authority
```

Conceptual Accept:

```text
legacy base fresh
candidate derived / reviewed
current Conformance base fresh
candidate normalized / conformant
no dual-authority state
        ↓
atomic cutover
        ↓
candidate-only reload
        ↓
same normalized Business meaning
        ↓
current World Conformance
```

## 10. Gate C / Dへの帰結

### Gate D — O1

Production migration未確定の中心は、もはやsidecar vs inlineの意味比較ではない。

Current strongest path:

```text
legacy sidecar
  -> migration input / Evidence

inline singular O1
  -> candidate authority after cutover
```

残るのはactual greenとShared Acceptance implementation上のcutover方法である。

### Gate C — Handoff

Current strongest path:

```text
Reference Set Handoff
  -> fixture assembly / migration input / Evidence

standalone Handoff source
  -> candidate authority after cutover
```

Reference Set自体をproduction Handoff authorityとして並存させない。

Second realistic Handoffで`from / to / condition` sufficiencyを確認する条件は引き続き残る。

## 11. 採らない設計

- old/new authorityのpermanent dual-read
- last-writer-wins migration
- inline O1とsidecar O1のmerge priority
- standalone / Reference Set Handoffのmerge priority
- same endpoint / different conditionをmigration時に自動multi-edge化
- production migration schemaVersionの先行固定
- Reference artifactの即時削除
- Git commit = cutover transactionという固定規則

## 12. Current decision

```text
legacy source read during migration preview
  -> allowed

candidate source exists during preview
  -> allowed / non-authoritative

legacy + candidate simultaneous accepted authority
  -> reject

accepted cutover target
  O1      -> inline singular narrow field candidate
  Handoff -> standalone Handoff source candidate

post-cutover reload
  -> candidate-only source bundle

production cutover mechanism
  -> not adopted

Pre-ADR
  -> Not Ready
```
