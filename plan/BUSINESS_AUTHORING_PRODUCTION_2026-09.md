# Business Authoring Production Readiness — 2026-09

Status: **Lane C / Pre-ADR option audit + ADR-vs-production adoption Evidence contract / no production loader, schema, or implementation adoption**

## この文書について

本書は、Business Procedure/Handoff candidateをproduction authoring sourceとして採用するための残課題を扱った次の2文書を1本へ統合したものである。

1. `BUSINESS_AUTHORING_PRODUCTION_LOADER_VERSIONING_OPTION_AUDIT_2026-09.md`（元・loader ownership / schema versioningのL0〜L4比較、L2採用）
2. `BUSINESS_AUTHORING_PRODUCTION_ADOPTION_EVIDENCE_CONTRACT_2026-09.md`（元・Contract ADR Ready / Production adoption Readyを分けるEvidence契約E-A〜E-J、Part 1のL2決定を前提に含む）

両文書は同じLane C Pre-ADR production readinessの検討で、Part 1が絞ったloader architecture決定をPart 2のEvidence契約（E-G/E-H等）が前提として引用する関係にあるため統合した。個別ファイルの履歴はGit historyから追跡できる。

外部から旧ファイル名を参照していた文書（`plan/issue/000042-business-procedure-cross-view-reference.md`、`plan/issue/000049-business-authoring-production-adoption-readiness.md`）は本統合と同じPRで本ファイル名へ参照を更新した。

---

# Part 1 — Loader / Versioning Option Audit（L0〜L4比較、L2採用）

## 1. 目的

Current migration pressureでは:

```text
legacy Procedure + O1 overlay + Reference Set Handoff
  ↓ migration preview
inline O1 Procedure + standalone Handoff
  ↓ one authority cutover
candidate-only authority
```

まで絞れている。

残るproduction blockerの一つは:

> loader / schema versioningをどう設計すれば、migrationのためにlegacy formatを読める一方、cutover後にold/new permanent dual authorityを再導入しないか。

本書ではproduction APIやschema名を採用せず、loader ownershipとversioning directionだけを比較する。

## 2. 固定済み前提

### Semantic candidate

```text
Procedure source
  D4 + S4-A + singular narrow O1

standalone Handoff source
  H1/R3
```

### Authority cutover

```text
legacy source
  -> migration input / Evidence only

candidate source
  -> Preview時はnon-authoritative

Shared Acceptance
  -> one atomic authority switch

post-cutover
  -> candidate-only authoritative load
```

### 禁止

```text
legacy + candidate
  -> heuristic merge

legacy + candidate
  -> priority / last-writer-wins

legacy fallback
  -> silently enabled forever
```

## 3. Option L0 — current Reference fixture schemaをそのままproduction化

Current `business-procedure-reference-v0` / Reference Set / pressure overlayをproduction loaderへそのまま持ち込む。

### 問題

- `canonical: false` / `reference-*` artifactRole等、Reference pressure用の意味を含む。
- O1 sidecarを残すことでcomposite Acceptance complexityを恒久化する。
- Reference SetはProcedure artifact locatorとHandoff meaningを同じassembly artifactへ持つ。
- migration targetとmigration inputを区別できない。

**判定: reject.**

Reference fixtureはEvidenceとして残し、production contractへ名称ごと昇格しない。

## 4. Option L1 — 一つのproduction loaderがlegacy / candidateを常時dual-read

Conceptual:

```text
loadBusinessModel()
  accepts legacy Procedure
  accepts O1 overlay
  accepts Reference Set Handoff
  accepts inline O1 Procedure
  accepts standalone Handoff
  merges everything
```

### 問題

- migration完了後もold/new authority ambiguityが残る。
- source discovery順やmerge priorityが意味へ影響する。
- RQ-003の暗黙複数正本を再導入する。
- stale legacy fileが後から復活してmeaningを変える可能性がある。
- loaderがmigration policyとsteady-state semanticsを同時所有する。

**判定: reject.**

## 5. Option L2 — migration adapterとsteady-state loaderを分離

Conceptual:

```text
Migration adapter
  read legacy authoritative sources
  ↓
  produce candidate source objects
  ↓
  normalize / compare / Conformance
  ↓
  Shared Acceptance
  ↓
  authority cutover

Steady-state Business loader
  read candidate Procedure sources
  read standalone Handoff sources
  ↓
  normalize one BusinessModel
```

Migration adapterはlegacy sourceをread-onlyに扱い、production steady-state loaderのfallback pathではない。

### 利点

- authority directionが明瞭。
- post-cutover loaderにlegacy merge semanticsが残らない。
- migration codeを後から削除してもaccepted candidate sourcesをreloadできる。
- migration previewとnormal runtime authoring loadを別Evidenceとして検証できる。
- RQ-003 / RQ-017と整合する。

### Risk

- cutover時にsource discovery / configurationを一度切り替える必要がある。
- migration adapterとcandidate loaderが同じnormalized meaningへ収束するEvidenceが必要。
- production repository transaction ownershipは別途必要。

**判定: current primary option.**

ただしproduction package / APIは未採用。

## 6. Option L3 — schemaVersionだけでold/new authorityを切替

例:

```text
business-procedure-v0
business-procedure-v1
```

を一loaderが読み、version番号の大小でauthorityを決める。

### 問題

Schema versionはdecode contractのversionであり、authority priorityではない。

```text
newer schemaVersion
  != accepted authority
```

である。

Migration candidateがv1だからといって、Shared Acceptance前にlegacy v0 authorityを上書きしてはいけない。

**判定: reject as authority mechanism.**

Schema versionはformat compatibilityへ限定する。

## 7. Option L4 — repository revision / manifest flagだけでlegacy modeを永続切替

Repository全体に:

```text
businessAuthoringMode: legacy | candidate
```

のようなmode flagを持つ案。

### 評価

Explicit cutover markerとしては理解しやすいが、current Evidenceでは独立fieldが必要とは限らない。

一つのrepository transactionで:

- candidate files create
- old authoritative discovery remove / disable
- candidate discovery enable

をatomicに成立できるなら、別mode正本は二重管理になる。

**判定: defer.**

Concrete repository/discovery implementationがmode markerを必要とする場合だけreopenする。

## 8. Schema versioningの原則

Production schemaを将来採る場合も:

```text
schemaVersion
  -> decoder / compatibility contract

source authority
  -> Shared Acceptance / repository discovery contract

Business semantic revision
  -> normalized meaning comparison

source freshness
  -> accepted base consistency
```

を分ける。

一つのversion numberへ全役割を畳まない。

## 9. Unknown / future field handling

Current Reference pressureは`DisallowUnknownFields`でfail closedしている。

Productionでもminimum candidateの意味を守るため、少なくとも:

- unknown semantic fieldを黙って無視してlossy normalizeしない
- unsupported newer schemaを旧decoderでbest-effort mergeしない
- migration adapterが理解できないlegacy cardinalityをcandidateへ縮退しない

を要求する。

特にcurrent O1 candidateはsingularなので:

```text
legacy multi-target O1
  -> migration error / unresolved
  -X-> first target only
```

を維持する。

## 10. Candidate loader discovery boundary

Steady-state loaderはsemantic sourceをpath文字列の意味で識別しない。

必要なのは概念上:

```text
source discovery
  -> source kind / logical identityをresolve
  -> strict decode
  -> duplicate logical source reject
  -> normalize
```

である。

ただしcurrent Evidenceだけで:

- directory naming convention
- manifest file
- registry DB
- generated source index

のどれを採るかは決めない。

File pathをBusiness identityへしないことだけ固定する。

## 11. Accept後reload Evidence

Production adoption前に必要なEvidence:

```text
accepted cutover revision R
  ↓
start fresh process / loader
  ↓
legacy migration adapterを使わない
  ↓
load candidate Procedure + standalone Handoff
  ↓
normalize BusinessModel
  ↓
expected Business semantic revision
  ↓
cross-view Conformance current World
```

ここでlegacy filesがRepositoryにhistorical Evidenceとして残っていても、steady-state authoritative discoveryから到達しないことを確認する。

Physical deletionを必須にしない。

## 12. Historical Evidenceとauthorityの分離

旧Reference filesは:

- regression fixture
- migration Evidence
- historical comparison

としてRepositoryに残り得る。

したがって:

```text
file exists
  != authoritative source
```

である。

重要なのはdeleteではなくauthority discoveryから外れること。

これによりReference Evidenceを失わずRQ-003を守れる。

## 13. Shared Acceptanceとの接続

Lane Cはgeneric transaction APIを所有しない。

Production cutoverではexisting Shared Acceptance workへ次を要求するだけとする。

1. legacy source base freshnessを確認できる。
2. candidate normalize / diagnostics / cross-view ConformanceをPreviewできる。
3. Preview後のBusiness source / World staleを検出できる。
4. candidate writeとauthority discovery switchをpartialに残さない。
5. Accept後candidate-only loaderでreloadできる。

具体APIは`ISSUE-000027 / 000046`等のshared Acceptance設計と整合して決める。

## 14. Current option ranking

```text
L2 migration adapter + candidate-only steady-state loader
  -> primary

L4 explicit mode marker
  -> defer until repository discovery requires it

L0 Reference schema direct production adoption
  -> reject

L1 permanent dual-read loader
  -> reject

L3 schemaVersion as authority priority
  -> reject
```

## 15. Pre-ADR implication（Part 1）

Gate C / Dのproduction migration blockerはさらに:

```text
representation choice
  -> largely narrowed

loader ownership
  -> migration adapter vs candidate steady-state separation preferred

schemaVersion authority semantics
  -> rejected

permanent dual-read
  -> rejected

remaining
  -> actual loader/discovery implementation + atomic cutover + fresh-process reload Evidence
```

へ絞られる。

## Part 1 現在の決定

```text
production loader API
  = not adopted

production schema name/version
  = not adopted

migration adapter / steady-state loader separation
  = current primary architecture

legacy compatibility fallback after cutover
  = reject

historical legacy files must be deleted
  = no; existence != authority

Pre-ADR
  = Not Ready
```

---

# Part 2 — Production Adoption Evidence Contract（Contract ADR Ready vs Production adoption Ready、E-A〜E-J）

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

一方、actual runner未実行のままproduction package / loader / schemaへ進むと、test-only vocabularyを実装都合で固定する危険がある（2026-09-20にGate A actual executionはPASSしたが、Gate A greenはproduction adoptionを意味しないため、この分離は維持する）。

本書では「Contract ADRへ進めるためのEvidence」と「決めたcontractをproduction authorityとして採用するためのEvidence」を分離する。

Readiness分離の正本補助は`BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md`とする。

Loader architectureそのものの選択はPart 1（L0〜L4比較、L2採用）を前提とする。

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

Production token / hash算法自体は、このEvidenceの前提ではない。

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

## Part 2（全体の現時点）readiness summary

```text
Semantic candidate
  -> strongly narrowed by test-only Pressure

Authoring candidate
  -> strongly narrowed

Placement
  -> separate Business plane strongest

Loader architecture
  -> migration adapter + candidate-only loader strongest（Part 1）

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
