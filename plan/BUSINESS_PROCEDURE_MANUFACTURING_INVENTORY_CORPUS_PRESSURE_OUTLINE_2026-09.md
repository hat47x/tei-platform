# Business Procedure Manufacturing / Inventory Corpus Pressure Outline — 2026-09

Status: **Lane C / second-corpus integration pressure / no Reference fixture / Gate 15c not activated / no production schema adoption**

## 1. 役割

本書はLane B PR #500のprovenance / acceptance情報を複製しない。

Cross-lane authority:

```text
Lane B
  LANE_B_REALISTIC_BUSINESS_CORPUS_PUBLIC_SOURCE_AUDIT_2026-09.md
    -> external provenance / reuse boundary

  LANE_B_SECOND_REALISTIC_REFERENCE_PRESSURE_SEED_2026-09.md
    -> fact / candidate / unresolved pressure seed

  LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md
    -> B-F19b RRB-P1 / P2 / P3 claim boundary

Lane C (this file)
  -> seedをcurrent D3 / D4 / O1 / H1 / R3 / S4-A / C1へ
     どう当てるかだけを所有
```

Lane B filesがmainへ未mergeの間も、PR #500をcross-lane sourceとして扱い、本書へ同じprovenance inventoryを第二正本化しない。

## 2. 現在の入力pressure

Lane B seedから受け取る最小candidateは、広い生産管理systemの再現ではない。

概念上:

```text
manufacturing / inventory continuation

small action pressure
  material issue
  manufacturing completion
  stock continuation
```

程度から始める。

重要なのは具体的な外部画面・table・function inventoryではなく:

```text
manual / physical completion
state / ownership boundary
Command result != Observation
Business handoff != Runtime Workflow
multiple responsibility / subsystem context
```

でcurrent TEI candidateを壊せるかである。

Exact trigger / actor / action order / handoff condition / Data entity / Application owner / authorization rule等は、Lane B seedが`candidate / unresolved`としている限りLane Cでもaccepted factへ昇格しない。

## 3. D3 / D4へ当てるPressure

Second corpusで最初に問う。

```text
manual / physical Business Action wording
  -> current thin bodyで保持できるか

current D4 ref kinds
  Application Pattern
  Validation
  Capability
  -> minimumとして足りるか
```

Patternから独立したdirect Data / Relation refがaccepted Business meaningとして不可欠になった場合:

```text
D4
  -> fail closed
  -> ref-kind design reopen
```

とする。

Address / Handoff layerでその不足を隠さない。

## 4. O1へ当てるPressure

Current O1はBP-06の:

```text
protected target / concern
  != direct Application Function realization
```

から生じたfield-specific pressureである。

Second corpusでは、このrole差が自然に反復するかを見る。

```text
repeated independent role pressure exists
  -> O1 authoritative-source necessityを強め得る

not repeated
  -> generic Role taxonomyのEvidenceにしない
```

Second corpusを作るために`ExplicitSemanticRef{Kind,Role,Target}`を先行導入しない。

## 5. H1 / R3 / S4-Aへ当てるPressure

Procedure bodyとは独立したBusiness continuation meaningが必要かを見る。

Candidate example:

```text
physical completion / stock fact is available
  -> another Business responsibility may continue
```

必要なら:

```text
S4-A Procedure-scoped Step local address
H1/R3 handoff-only relation
```

でlosslessに保持できるか確認する。

次へ短絡しない。

```text
Handoff PASS
  != full Business Flow / DFD Conformance
  != Runtime Workflow transition
```

## 6. Command / Observation physical-world Pressure

Second corpusで特に強く確認したい境界:

```text
Application Command accepted
  != material physically received
  != material physically issued
  != manufacturing physically completed
  != stock physically present
  != shipment physically completed
```

具体的なread contractがsource / TEI authorityから確定しない限り、Observation identityやCapabilityを捏造しない。

Business Procedureはphysical completionを必要とするBusiness intentionを保持し、実際のObservationは既存Driver / Query / Provider boundaryへ残す。

## 7. Cross-view contradiction opportunities

Lane Cでsource/testへ昇格する価値があるのは、少なくとも次のような反証を作れる場合である。

```text
A. Business requires physical completion
   but Function only records Command acknowledgement

B. Business needs external-observed state
   but Application assumes managed-here write authority

C. Business continuation depends on actual stock/completion
   but representation derives relation only from adjacency

D. Business Action is manual / physical
   but projection invents Capability to make it executable

E. accepted Business meaning needs a direct authority ref
   but D4 would silently drop it
```

将来fixtureを作る場合、baselineとintentional defectを分ける。

## 8. Lane CでRRB-P1 / P2へ進む順序

Lane B Acceptance ContractのP1/P2/P3を正とする。

Lane Cでは:

```text
current #482 actual green
  ↓
second corpus workを続ける価値を再確認
  ↓
RRB-P1
  small TEI-original authoring source
  provenance link
  fact / candidate / unresolved preserved
  no copied external artifact
  ↓
RRB-P2
  checker / test-only pressure integration
  ↓
RRB-P3
  actual Repository execution green
```

とする。

本書自体はP1ではない。

## 9. Gate 15との関係

```text
B-F19b breadth work
  != Gate 15c automatically open
```

Second corpusのoutline / P1 / P2を進めても、それだけで15cを開かない。

D3 / D4 / O1 / H1 / R3等に**具体的なminimum-contract insufficiency**が見つかった場合だけ、15cまたは該当candidateをreopenする。

一方B-F19bはReference maturity / breadth claimとして、Lane B RRB-P3まで未解消のままである。

## 10. 現時点の判断

```text
external provenance source
  available (Lane B B-F19a)

Lane B pressure seed / acceptance contract
  available in PR #500

Lane C integration outline
  this document

TEI-native second Business authoring source
  not created

RRB-P1 / P2 / P3
  not achieved by this document

Gate 15c
  not activated

new ref kind / role taxonomy / Business Flow schema necessity
  not proven
```

## 11. 次

1. `HAT47X`等actual Repository execution経路が戻れば、まずcurrent #482 pressure testsをgreenにする。
2. actual green前にsecond corpusをfixture schemaへ昇格しない。
3. actual green後、B-F19bを進める価値がある場合だけRRB-P1の小型TEI-original sourceを作る。
4. RRB-P1から具体的不足が出たところだけD4 / O1 / H1-R3 / T-GAP-06をreopenする。

## 12. 非対象

- external Reference artifact import
- CONCEPTWARE / TALON reproduction
- external license inference
- production Canonical schema adoption
- Gate 15c activation
- generic ref-role taxonomy adoption
- full Business Flow / DFD adoption
- Gate B / Gate C Required Evidence change
