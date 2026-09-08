# Business Model Semantic Acceptance Boundary Pressure — 2026-09

Status: **Lane C / normalized semantic acceptance pressure / no production Acceptance API adoption**

## 1. 目的

Business Procedure minimum candidateが個別要素から次へ収束してきた。

```text
D4 body
+ narrow O1 protected target
+ S4-A Procedure-scoped Step local address
+ H1/R3 thin Business Handoff
```

本Pressureでは、これらが複数authoring artifactに分かれていても、review / acceptance上は一つのBusiness meaningとして安定に扱えるかを確認する。

Source file pathやraw artifact layoutをBusiness semantic identityへ混ぜない。

## 2. Current source layout

Current Referenceでは:

```text
order-entry-business-procedure.json
order-entry-follow-up-procedure.json
order-entry-procedure-reference-set.json
order-entry-protected-target-overlay.json  // O1 pressure apparatus only
```

が存在する。

Reference Setは二つのProcedureと一つのexplicit Handoffをauthorする。

```text
order-entry.order-handling / BP-09
  -> order-entry.follow-up / FU-00
  when follow-up is required after the earlier order interaction
```

このHandoffはProcedure bodyの順序や隣接性からderiveしたものではない。

## 3. Why O1 and Handoff have different source-placement pressure

### O1

Protected Application targetはBP-06という一つのBusiness Actionに属するmeaningである。

Separate sidecarへ置くとProcedure sourceとO1 sourceを一つのAcceptance unitへ束ねる追加mechanismが必要になる。

したがってproduction comparisonではsame Business sourceのnarrow optional fieldがstronger candidateになった。

### Handoff

Handoffは:

```text
Procedure A / Step X
  -> Procedure B / Step Y
```

という複数Procedure間relationである。

どちらか一方のProcedure bodyへ埋め込むと:

- relation ownershipが片側へ偏る
- cross-Procedure endpoint changeが片側sourceの内部meaningに見える
- Procedure bodyとrelation lifecycleが混ざる

したがってset-level / Business-model-level sourceは自然である。

```text
O1 sidecar complexity
  does not imply
Handoff set-level source is wrong
```

とする。

## 4. Normalized minimum Business meaning

Test-only normalized candidate:

```text
BusinessModel
  Procedures[]
    Procedure
      id
      purpose
      trigger
      actor
      Steps[]
        localKey
        D4 body
          businessAction
          applicationPatterns[]
          validations[]
          capabilities[]
        protectedApplicationPattern?  // zero-or-one current O1

  Handoffs[]
    from { procedure, step }
    to   { procedure, step }
    condition
```

このshapeはproduction schemaではない。

重要なのはmeaning boundaryである。

## 5. Semantic fingerprint pressure

`d4_o1_normalized_business_acceptance_pressure_test.go`ではnormalized Business modelをdeterministic JSONへし、test-only semantic fingerprintを作る。

### Meaning change must change fingerprint

現在Pressureする:

```text
protected target-only change
  -> fingerprint changes

Handoff condition-only change
  -> fingerprint changes
```

### Source infrastructure change must not change fingerprint

```text
Procedure artifact locator move
  -> normalized meaning unchanged
  -> semantic fingerprint unchanged

Reference Set procedure locator ordering
  -> semantic fingerprint unchanged
```

したがってAcceptance revisionをraw source bytesだけへ結び付けるより:

```text
source artifacts
  -> strict parse / reference resolution
  -> normalized Business meaning
  -> review / semantic revision
```

という方向が強い。

## 6. This is not a proposal to hash every Canonical model

Current semantic fingerprintはtest-only pressure vocabularyである。

Productionへ次を導入する決定ではない。

```text
global semantic hash
ContentAddressed Canonical objects
Merkle model
revision ID replacement
all source files normalized before acceptance
```

確認したいのは:

> Business review unitのstale判定がsource locator変更とaccepted meaning変更を区別できること。

である。

## 7. H1/R3 necessity refinement

Procedure bodiesだけではaccepted Handoff condition / endpoint relation changeを保持できない。

したがってcurrent multi-Procedure Reference meaningに対して:

```text
some thin normalized Business relation
```

は必要である。

これは以前の:

```text
H1 thin normalized handoff
R3 handoff-only Business relation boundary
```

pressureを強める。

一方、まだ次は証明しない。

```text
standalone top-level Canonical BusinessHandoff kind
generic Business Edge
full Business Flow / DFD
runtime handoff execution
workflow transition semantics
```

Current conclusion:

> Handoff meaning is required in the normalized Business model, but a standalone production Canonical kind is not yet required.

## 8. Handoff minimum body

Current accepted meaningから必要なのは:

```text
from Procedure/Step address
to Procedure/Step address
natural-language Business condition
```

である。

まだ入れない:

```text
payload
Data flow
Document type
participant / role
transport
queue
retry
timeout
sequence number
runtime status
execution ID
```

HandoffはBusiness responsibility / continuation relationとしてthinに保つ。

## 9. Local Step addressとの関係

Handoff endpointはS4-AのProcedure-scoped local Step addressを参照する。

```text
Procedure ID + Step local key
```

で十分であり、global Canonical Step IDを要求しない。

Logical Step renameは:

```text
address-only rename
  -> stale Handoff

coordinated address + Handoff endpoint rename
  -> accepted semantic change / migration
```

としてfail closedする。

## 10. Acceptance unit implication

Business meaningが複数sourceに分かれる場合でも、review unitはsource file単位に固定しない。

Conceptual direction:

```text
Business authoring sources
  - Procedure A
  - Procedure B
  - Handoff set relation
  - possibly inline narrow O1
        ↓
normalized Business model
        ↓
semantic review / acceptance target
```

Source Indexはlocator / provenanceを保持するが、normalized Business modelへPath/Fingerprintを埋め込まない。

## 11. Current candidate stack update

Current strongest minimum comparison becomes:

```text
D4 + singular narrow O1
  + S4-A local Step address
  + H1/R3 thin set-level Handoff relation
```

C1 optional Reference Application contextはreporting / Reference contextとして別に残す。

D3はsubtraction historyとして残す。

O1 production sidecarはpreferred candidateから外す。

## 12. What remains open

1. actual Repository runner green。
2. normalized semantic fingerprintをproduction Shared Acceptanceへどう接続するか。
3. Production sourceでO1をinline narrow fieldとして採用するか。
4. Handoff relationのproduction placementがBusiness source set / Canonical Business model / another thin boundaryのどれか。
5. second realistic cross-Procedure Handoff pressureでcurrent thin bodyが十分か。
6. BusinessModel自体をproduction `ir.World`へ持つ必要があるか。

## 13. Current decision

```text
D4 + narrow O1 + S4-A + H1/R3
  -> current minimum normalized Business meaning candidate

Handoff
  -> required as normalized meaning for current accepted multi-Procedure Reference
  -> keep set-level / relation-level
  -> do not embed into one Procedure body
  -> do not generalize to Business Flow

Acceptance
  -> pressure toward semantic review unit rather than raw source locator unit
  -> no production API adoption yet
```
