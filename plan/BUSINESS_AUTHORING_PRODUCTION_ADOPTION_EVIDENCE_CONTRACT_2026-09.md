# Business Authoring Production Adoption Evidence Contract — 2026-09

Status: **Lane C / ADR-vs-production adoption Evidence contract / no production implementation adoption**

## 1. 目的

Current Lane Cではsemantic candidate自体はかなり絞れた。

```text
D4 + singular narrow O1 + S4-A + H1/R3

Procedure authoring
  -> inline singular O1 candidate

Handoff authoring
  -> standalone Handoff candidate

placement
  -> separate normalized Business semantic plane

migration
  -> legacy-read preview -> one authority cutover -> candidate-only load
```

一方、actual runner未実行のままproduction package / loader / schemaへ進むと、test-only vocabularyを実装都合で固定する危険がある。

本書では「Contract ADRへ進めるためのEvidence」と「決めたcontractをproduction authorityとして採用するためのEvidence」を分離する。

Readiness分離の正本補助は`BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md`とする。

## 2. Evidence natureを分離する

次を混同しない。

```text
source-level test-only Pressure
  != actual Repository execution Evidence
  != Contract ADR feasibility Evidence
  != production loader / transaction Evidence
  != PAS-01 Shared Acceptance end-to-end Evidence
  != Product completion claim
```

Contract ADR readinessは、contractを反証できるactual Evidenceとfeasibility Evidenceで判断する。

Production adoption readinessは、それに加えてproduction source authority / transaction / reload Evidenceを要求する。

PAS-01全体をLane Cだけでacceptedへ上げない。

また、本書の`Pre-ADR Gate C`はHandoff/addressのlocal audit gateであり、`GATE_C_EVIDENCE_CONTRACT.md`の第一Web Targetを扱う**Product Gate C**とは別物である。

## 3. E-A — Actual Repository mechanical green

同一candidate revision `R`で少なくとも:

```text
gofmt check
go vet ./...
go test ./...
reference Business Procedure checker baseline
intentional defect checker expected diagnostic
```

を実行する。

### 必須記録

- repository revision `R`
- Go version / relevant tool version
- command
- exit status
- failure時のlog / diagnostic

### 注意

`gofmt -w`して自動commitするworkflowは、read-only verification Evidenceと同じものではない。

Mechanical greenが得られるまでtest-only Pressureを`proven in repository`へ昇格しない。

## 4. E-B — Current minimum losslessness actual

Actual testsで少なくとも:

1. D4-onlyではBP-06 protected target meaningを失う。
2. D4 + singular O1でcurrent meaningを保持する。
3. S4-AでHandoff endpointをexactにresolveできる。
4. H1/R3でcondition / endpoint changeをsemantic diffとして保持する。
5. Business wording / O1 / Handoff-only changesで`ir.World`を偽変更しない。
6. missing Application targetはseparate Business planeでもfail closedする。

をgreenにする。

## 5. E-C — Authoring representation actual

### Procedure / O1

```text
inline singular protectedApplicationPattern?
```

について:

- strict decode
- round-trip
- zero-or-one
- multi-target fail closed
- generic role field fail closed
- legacy sidecarとsame normalized meaning

をactual確認する。

### Handoff

```text
standalone Handoff source
```

について:

- strict decode
- round-trip
- Reference Set Handoffとsame normalized meaning
- Procedure artifact locator非依存
- stale endpoint fail closed
- Workflow field fail closed

をactual確認する。

## 6. E-D — Multi-source semantic bundle actual

Candidate Procedure source群 + standalone Handoff sourceを:

```text
multiple files
  -> one normalized BusinessModel
```

としてactual loadする。

確認:

- duplicate Procedure logical identity fail closed
- source enumeration order semantic no-op
- path / locatorをsemantic identityへしない
- Procedure wording / O1 / Handoff conditionの各changeがsame Business semantic revisionへ反映

## 7. E-E — Freshness / Conformance-base actual

Preview baseについて:

```text
Business source freshness
  != Business semantic revision
  != cross-view Conformance base
```

をactual確認する。

Negative cases:

1. Procedure source changed after Preview -> stale.
2. O1 changed after Preview -> stale.
3. Handoff changed after Preview -> stale.
4. Business source unchanged / World target removed -> Conformance base stale.
5. World unchanged / Business source changed -> stale.

Production token / hash algorithm自体は、このEvidenceの前提ではない。

## 8. E-F — Coordinated semantic atomicity actual

Current reference case:

```text
BP-09 -> BP-09-renamed
Handoff.from.step -> BP-09-renamed
```

について:

```text
Procedure only
  -> fail closed

Handoff only
  -> fail closed

both candidate change-set
  -> normalize succeeds
  -> new semantic revision
```

をactual確認する。

これはrepository transaction implementationの完全証明ではないが、Contract ADRに必要なsemantic change-set atomicityの最低条件になる。

## 9. E-G — Migration preview / authority cutover

### Contract ADR Evidence

legacy sourcesをread-onlyに読みcandidateを生成できることをactual確認する。

- legacy inputs in-place mutationなし
- candidate same normalized meaning
- singular candidateで表現できないlegacy multi-targetはfail closed

さらにauthority semanticsとして:

```text
legacy + candidate simultaneous authority
  -> reject
```

をactual確認する。

- same Step O1
- same endpoint Handoff
- same meaningでもduplicate authorityとしてreject
- conflicting meaningもauto-mergeしない

ここまではContract ADR Evidenceである。

### Production adoption Evidence

実source authorityとして採用する前には、Shared Acceptance / repository transaction boundaryで:

```text
candidate source write
+ authoritative discovery switch
```

をpartialに残さないことを実証する。

**production multi-file writer / final repository transaction APIの完成はContract ADRの前提にしない。**

ただしfeasibility spikeでpartial authority switchを原理的に避けられないことが判明した場合はcontractをreopenする。

## 10. E-H — Candidate-only fresh-process reload

### Contract ADR feasibility Evidence

Contract ADR前には少なくとも:

```text
candidate Procedure sources
+ standalone Handoff source
  ↓
legacy migration inputを読まないfresh process / separate executable boundary
  ↓
strict decode / normalize
  ↓
same expected Business semantic revision
  ↓
current World Conformance
```

を再現できることを要求する。

これはproduction steady-state loader完成を要求するものではない。Test helper / focused executable / prototypeでも、**legacy fallbackなし**とprocess boundaryを実証できればContract feasibility Evidenceになり得る。

### Production adoption Evidence

Production adoption判断では:

```text
accepted cutover revision R2
  ↓
new production process
  ↓
legacy migration adapterを呼ばない
  ↓
production candidate-only loader
  ↓
same accepted Business semantic revision
  ↓
cross-view Conformance
```

を要求する。

旧Reference / migration filesがdisk / repositoryに残っていてもよい。

重要なのは:

```text
legacy file exists
  != legacy authority reachable
```

である。

## 11. E-I — Handoff subtraction / realistic pressure

Actual current structural testsで:

- actor context changeをHandoff endpointへ複製しない
- Procedure triggerとHandoff conditionを別authorityにする
- adjacency / Function sequence / ObservationからHandoffを生成しない

を確認する。

さらにContract ADR前には:

```text
second realistic TEI-original Handoff
```

が必要。

Current Repositoryにはqualifying caseがないため、ここは未充足として残す。

Second caseのためだけにsynthetic workflowを作らない。

## 12. E-J — Derived Data / CRUD guardrail actual

Business sourceへData / CRUD authorityを重複追加しない方針をactual testsで確認する。

- exact Data projection
- no trace != no dependency
- R operation projectionだけevidence-backed cellを埋める
- Pattern kindからCRUD推測しない
- Database writeからBusiness Action Uを推測しない
- Lifecycle transitionをgeneric Uへ短絡しない

Qualifying C/U/D caseがないこと自体はmechanical failureでもContract ADR blockerでもない。

```text
unknown / insufficient authority
```

を正しく保持することがEvidenceである。

Qualifying C/U/D caseは将来generic mutation/effect contractをreopenするtriggerとして残す。

## 13. 二段階readiness

### 13.1 Contract ADR Ready条件

最低限、同じbranch/revision系列で:

```text
E-A actual mechanical green
E-B current minimum actual
E-C authoring actual
E-D multi-source actual
E-E freshness actual
E-F semantic atomicity actual
E-G migration preview / dual-authority reject actual
E-H candidate-only fresh-process feasibility actual
E-I second realistic Handoff sufficiency
E-J derived projection guardrail actual
```

を確認する。

この段階では、production shared Acceptance service / production atomic writer / Studio / AI integration完成を要求しない。

### 13.2 Production implementation着手条件

Production `internal/business` / loader / source schema等へ着手する前に少なくともE-A〜E-Fをactual確認する。

Contract ADRが成立していれば、そのdecisionに従ってproduction adapter / loader prototypeへ進める。

### 13.3 Production adoption Ready条件

Contract ADR Readyに加えて:

- Business-specific production adapter
- production source discovery / strict decode
- Business + Worldをfalse-negativeなくcoverするcomplete stale envelope
- atomic candidate source apply
- atomic authority-discovery cutover
- production fresh-process candidate-only reload
- reload後Conformance
- failure時にpartial write / partial authority switchを残さない

をactualに実証する。

### 13.4 Product Gate C後のShared Acceptance integration

`ISSUE-000027` / `ISSUE-000046`が待つ`Gate C Ready`は第一Web Targetの**Product Gate C**であり、本書のHandoff Pre-ADR Gate Cとは別である。

Studio / AI end-to-end Shared AcceptanceはProduct Gate C後のworkであり、Business Procedure Contract ADRの前提にしない。

## 14. Production adoption後に初めて固定する候補

Evidenceが揃った後でも、次を一度に固定する必要はない。

第一段階候補:

- Business authoring source decoder/loader boundary
- normalized Business model internal boundary
- cross-view validator
- Source trace

後段:

- Studio / AI shared Acceptance integration
- Semantic diff / impact integration
- richer Business Flow / SubjectArea等

`ir.World.Business`はconcrete insufficiencyが出るまで採用しない。

`ir.World.Business`が必要だというEvidenceがないことはContract ADR blockerではなく、separate Business planeを維持するnegative Evidenceである。

## 15. No-go conditions

次のいずれかが出た場合はADRを起こす前にcandidateをreopenする。

- actual testでD4 + O1がcurrent meaningをlosslessに保持しない
- singular O1でreal current caseを縮退する
- Handoff `from/to/condition`がsecond realistic caseでlossy
- separate planeではConformanceをfail closedできない
- candidate-only fresh-process feasibilityがlegacy mergeなしでは成立しない
- multi-source source freshness / Conformance-base freshnessをfail closedできない
- authority cutover semanticsがold/new simultaneous authorityを避けられない

次はproduction adoption no-goであり、Contract ADRと分離する。

- production cutoverでpartial source write / partial authority switchを避けられない
- complete stale envelopeをproduction source planeに実装できない
- production candidate-only loaderがlegacy fallbackなしで成立しない

これらが単なる未実装ではなくcontract自体の不成立を示した場合はADRをreopenする。

## 16. Current readiness summary

```text
Semantic candidate
  -> strongly narrowed by test-only Pressure

Authoring candidate
  -> strongly narrowed

Placement
  -> separate Business plane strongest

Loader architecture
  -> migration adapter + candidate-only loader strongest

Actual Repository Evidence
  -> unresolved

Candidate-only fresh-process feasibility
  -> unresolved

Second realistic Handoff
  -> unresolved

Qualifying C/U/D case
  -> absent, but not Contract ADR blocker

ir.World.Business necessity Evidence
  -> absent, supports current separate-plane direction

Contract ADR
  -> Not Ready

Production adoption
  -> Not Ready; stronger post-ADR Evidence required

PAS-01 / Studio / AI end-to-end
  -> Product Gate C後の別work
```

本書はNot Readyを長引かせるためではなく、残るEvidenceを**decision前 / production採用前 / Product integration後**の実行可能な単位へ縮めるための契約である。
