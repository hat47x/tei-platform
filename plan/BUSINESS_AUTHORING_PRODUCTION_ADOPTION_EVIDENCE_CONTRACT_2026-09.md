# Business Authoring Production Adoption Evidence Contract — 2026-09

Status: **Lane C / Pre-ADR evidence contract / no production implementation adoption**

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

本書では「ADR / production adoptionへ進めるために何を実証すべきか」を一つのEvidence contractへまとめる。

## 2. Evidence natureを分離する

次を混同しない。

```text
source-level test-only Pressure
  != actual Repository execution Evidence
  != production loader Evidence
  != PAS-01 Shared Acceptance end-to-end Evidence
  != Product completion claim
```

Lane CのADR readinessは最初の三つを中心に判断し、PAS-01全体をLane Cだけでacceptedへ上げない。

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

## 8. E-F — Coordinated atomicity actual

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

これはrepository transaction implementationの完全証明ではないが、semantic change-set atomicityの最低条件になる。

## 9. E-G — Migration preview / authority cutover actual

### Preview

legacy sourcesをread-onlyに読みcandidateを生成する。

- legacy inputs in-place mutationなし
- candidate same normalized meaning
- singular candidateで表現できないlegacy multi-targetはfail closed

### Authority conflict

```text
legacy + candidate simultaneous authority
  -> reject
```

- same Step O1
- same endpoint Handoff
- same meaningでもduplicate authorityとしてreject
- conflicting meaningもauto-mergeしない

### Cutover

Shared Acceptance / testable repository transactionで:

```text
candidate source write
+ authoritative discovery switch
```

をpartialに残さない。

## 10. E-H — Fresh-process candidate-only reload

Production adoption判断で特に重要なEvidence。

```text
accepted cutover revision R2
  ↓
new process
  ↓
legacy migration adapterを呼ばない
  ↓
load candidate Procedure sources
  + standalone Handoff source
  ↓
normalize
  ↓
same accepted Business semantic revision
  ↓
cross-view Conformance
```

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

ただしproduction ADR前にはさらに:

```text
second realistic TEI-original Handoff
```

が必要。

Current Repositoryにはqualifying caseがないため、ここだけは未充足として残す。

Second caseのためだけにsynthetic workflowを作らない。

## 12. E-J — Derived Data / CRUD guardrail actual

Business sourceへData / CRUD authorityを重複追加しない方針をactual testsで確認する。

- exact Data projection
- no trace != no dependency
- R operation projectionだけevidence-backed cellを埋める
- Pattern kindからCRUD推測しない
- Database writeからBusiness Action Uを推測しない
- Lifecycle transitionをgeneric Uへ短絡しない

Qualifying C/U/D caseがないこと自体はmechanical failureではない。

```text
unknown / insufficient authority
```

を正しく保持することがEvidenceである。

## 13. Production implementation着手条件

Production `internal/business` / loader / source schema等へ着手する前に最低限:

```text
E-A actual mechanical green
E-B current minimum actual
E-C authoring actual
E-D multi-source actual
E-E freshness actual
E-F semantic atomicity actual
```

を同じbranch/revision系列で確認する。

E-G / E-Hの物理transaction部分はproduction prototypeと並走してもよいが、ADR Ready判定前にEvidenceを揃える。

E-I second realistic caseもPre-ADR Gate C blockerとして残す。

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

## 15. No-go conditions

次のいずれかが出た場合はADRを起こす前にcandidateをreopenする。

- actual testでD4 + O1がcurrent meaningをlosslessに保持しない
- singular O1でreal current caseを縮退する
- Handoff `from/to/condition`がsecond realistic caseでlossy
- separate planeではConformanceをfail closedできない
- candidate-only fresh reloadがlegacy mergeなしで成立しない
- multi-source source freshnessを安全に固定できない
- production cutoverでpartial authority switchを避けられない

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

Second realistic Handoff
  -> unresolved

Production ADR
  -> Not Ready
```

本書はNot Readyを長引かせるためではなく、残るEvidenceを**実行可能な単位へ縮める**ための契約である。
