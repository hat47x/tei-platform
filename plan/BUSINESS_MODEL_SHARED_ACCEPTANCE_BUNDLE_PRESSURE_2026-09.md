# Business Model Shared Acceptance Bundle Pressure — 2026-09

Status: **Lane C / test-only Acceptance pressure / no production revision API adoption**

## 1. 問い

Current authoring candidatesは一つのfileに閉じない。

```text
Procedure source A
Procedure source B
...
standalone Handoff source
```

Procedure source内にはD4 + S4-A + singular narrow O1があり、Handoffはcross-Procedure relationとして別sourceへ置く方が自然である。

このときShared Acceptanceをfile単位へ分解すると、Procedure wording / protected target / Handoff conditionが別々のapproval / revisionへ分裂し、cross-view Business meaningの一貫したreview単位を失う可能性がある。

本Pressureでは、複数source kindを一つの**normalized Business semantic revision**へ束ねられるかを確認する。

## 2. Test-only bundle

`business_model_shared_acceptance_bundle_pressure_test.go`では概念的に:

```text
BusinessAuthoringBundle
  Procedures[]
    inline Procedure source
      D4 + S4-A + singular narrow O1

  Handoffs
    standalone Handoff source
```

を入力とする。

Bundleはsource path / artifact locator / runtime stateをsemantic fieldsとして持たない。

## 3. Normalization target

```text
Procedure source A ─┐
Procedure source B ─┼─> normalize -> BusinessModel
...                ─┤
Handoff source     ─┘

BusinessModel
  Procedures[]
  Handoffs[]
```

Current test-only bundleから得るmodelは、既存Reference Set + sidecar pressureから得ていたcurrent normalized Business modelと同じmeaningになることをPressureする。

つまりsource decompositionを変えてもBusiness semantic meaningを変えない。

## 4. One semantic revision across source kinds

次のaccepted Business source changeはいずれも、同じBusiness semantic revisionの差として観測する。

```text
BP-04 businessAction change
  -> revision changes

BP-06 protectedApplicationPattern change
  -> same revision changes

Handoff condition change
  -> same revision changes
```

これらをProcedure / O1 / Handoffの独立した意味正本へ分割しない。

Source fileは分かれても、accepted Business meaningは一つのnormalized review unitとして扱う。

## 5. Source ordering / locatorはmeaningではない

Procedure source配列の記述順だけを変更してもnormalized Business semantic revisionを変えない。

既存Pressureでもpure Procedure artifact locator moveでnormalized meaningが変化しない。

したがってproduction Acceptanceでも:

```text
source enumeration order
file path
manifest locator
```

をBusiness semantic identityへしてはいけない。

## 6. Source freshnessはsemantic revisionと分ける

`business_model_source_freshness_pressure_test.go`では、Preview時のauthoring source snapshotを:

```text
logical source key
+ content fingerprint
```

としてtest-onlyに保持する。

Current logical keys:

```text
procedure:order-entry.order-handling
procedure:order-entry.follow-up
handoffs:default
```

Pathはkeyに含めない。

Pressure:

- Procedure wording source change -> stale
- inline O1 source change -> stale
- standalone Handoff source change -> stale
- Procedure source enumeration orderのみ -> same freshness snapshot

意味比較とfreshness比較は別である。

```text
semantic revision
  = normalized Business meaningを比較

source freshness
  = Preview対象sourceがAccept直前まで同一かを比較
```

Whitespace等でsource contentが変わればfreshnessはstaleになり得る一方、normalize後にsemantic no-opかどうかは別判定である。

Production hash / ETag algorithmはまだ採用しない。

## 7. Cross-view Conformance base freshnessも分ける

Business source freshnessだけではShared AcceptanceのPreview前提を十分に表せない。

Business sourcesが変化していなくても、Preview時にcross-view validateした`ir.World`側のmeaningがAccept前に変化する可能性がある。

Current negative case:

```text
Preview
  Business sources = unchanged
  BP-06 protected target = order_entry.submit_order
  World resolves target

After Preview
  Business sources = unchanged
  World removes order_entry.submit_order
```

この場合:

```text
Business source freshness
  -> fresh

Business semantic revision
  -> unchanged

cross-view Conformance base
  -> stale
```

であり、以前のConformance PASSをそのままAccept根拠にしてはいけない。

`business_model_conformance_base_freshness_pressure_test.go`では:

- Business source / Worldともunchanged -> Preview base accepted
- Business source unchanged / World changed -> stale reject
- World unchanged / Business source changed -> stale reject

をtest-onlyにPressureする。

したがって概念上:

```text
Business source freshness
  != Business semantic revision
  != cross-view Conformance base
```

を維持する。

Production implementationが一つのrepository-level `baseModelRevision`で両source群を安全に固定できるなら物理tokenは一つでもよい。ただしLane Cからplatform-wide revision schemaを決めない。Business sourceとWorld sourceが独立更新可能な実装なら、それぞれのstaleを検出する必要がある。

Detailed production-boundary auditは`BUSINESS_MODEL_PRODUCTION_ACCEPTANCE_BOUNDARY_AUDIT_2026-09.md`へ委譲する。

## 8. Relation to O1 sidecar pressure

O1 sidecar単独ではProcedure-only fingerprintがtarget-only changeを見逃す問題があった。

Current candidateではO1をProcedure sourceへinlineする方向へ進んだため、O1についてはこの複雑さを減らせる。

一方Handoffはcross-Procedure relationなのでstandalone sourceが自然であり、複数source自体は残る。

Shared Acceptance bundleは、この残る複数sourceをsemantic unitとして統合する。

## 9. Relation to separate Business plane

Current strongest placement candidate:

```text
Business authoring sources
  -> normalize
  -> Business semantic revision

TEI workspace sources
  -> ir.World
  -> runtime/application revision

Business model
  <-> cross-view Conformance <-> ir.World
```

Business bundleのどのsourceが変わってもBusiness semantic revisionは変わり得るが、それだけで`ir.World` revisionを変更する必要はない。

```text
first-class Business meaning
  != same runtime/compiler revision
```

というplacement pressureを補強する。

同時にseparate planeであるからこそ、Accept時にはBusiness source baseだけでなくcross-view Conformance baseのstalenessも確認する必要がある。

## 10. Atomicity pressure

複数sourceを一つのsemantic revisionとして扱う場合、協調changeを片方だけacceptしてはいけない。

Current test-only case:

```text
Procedure BP-09 -> BP-09-renamed
Handoff from.step BP-09 -> BP-09-renamed
```

`business_model_shared_acceptance_bundle_pressure_test.go`では:

```text
Procedure Stepだけrename
  -> stale Handoff endpoint
  -> fail closed

Handoff endpointだけrename
  -> missing Procedure Step
  -> fail closed

both in same candidate bundle
  -> normalize succeeds
  -> new Business semantic revision
```

をPressureする。

したがってShared Acceptanceは少なくとも:

```text
candidate change-set
  -> check Business source freshness
  -> normalize all affected Business sources
  -> validate Business referential integrity
  -> establish / recheck current Conformance base
  -> cross-view validate against current ir.World
  -> accept/reject as one semantic change-set
  -> reload / Conformance
```

という境界を持つ必要がある。

Git transaction、filesystem locking、repository commit方式等の物理実装は未決とする。

## 11. Cross-lane shared Acceptanceとの関係

Lane C専用のgeneric Acceptance engineは作らない。

```text
ISSUE-000027
  candidate + baseModelRevision + shared Acceptance principle

ISSUE-000039
  read-only Semantic diff / impact analysis

ISSUE-000046
  Studio candidate + baseModelRevision
  stale candidate reject
  shared Acceptance Boundary

Lane C
  Business multi-source固有の
    source freshness
    normalized Business revision
    cross-view Conformance base
    reload obligation
```

を分担する。

## 12. Production APIとしてまだ決めないもの

- `BusinessRevision` production type
- semantic hash algorithm
- freshness ETag format
- multi-file transaction API
- Git commit = Business revisionという固定規則
- `ir.World` revisionとのcoupled version
- independent O1 / Handoff approval stores
- production `internal/business` package
- platform-wide `baseModelRevision` schema
- Business専用transaction / locking mechanism

Testで使うsemantic / source / World fingerprintsは比較語彙でありproduction contractではない。

## 13. Production Shared Acceptanceが満たすべき性質

1. 複数Procedure source + Handoff sourceを一つのnormalized Business modelへloadできる。
2. same Procedure identityのduplicate sourceをfail closedする。
3. stale Handoff endpointをfail closedする。
4. protected Application targetを`ir.World`へexact resolutionする。
5. source order / pathだけのchangeをsemantic no-opとして扱える。
6. Procedure wording / O1 target / Handoff conditionのchangeをsemantic changeとして扱う。
7. Accept直前にlogical Business sourceごとのfreshnessを再確認できる。
8. Preview後にWorld / Conformance baseが変化した場合、staleとしてrejectまたはcurrent Worldへrevalidateできる。
9. coordinated Step/address + Handoff endpoint changesを一つのchange-setとしてvalidateできる。
10. Accept後に全sourceをreloadし同じnormalized Business modelを再現できる。
11. reload後にcurrent Worldとのcross-view Conformanceを再確認できる。
12. cross-view Conformance失敗時に部分acceptしない。
13. Projection / Matrix / diagnosticをauthoritative sourceへ逆書きしない。

## 14. Current result

Current test-only Evidence / Pressureは:

```text
multi-source authoring
  -> one normalized Business semantic meaning

Business source snapshot
  -> independent freshness check

World / Conformance base
  -> independent stale check pressure

coordinated cross-source change
  -> one atomic semantic change-set pressure
```

まで進んだ。

したがってPre-ADR Gate Eの問いは、概念的bundling / Business-source freshness / Conformance-base freshness / referential atomicityの可否から、**existing shared AcceptanceへBusiness固有obligationをどう実装接続するか**へ絞られた。

## 15. Current decision

```text
Conceptual multi-source semantic bundle
  -> test-onlyで有力

Logical-key Business source freshness snapshot
  -> test-onlyで有力

Cross-view Conformance base freshness
  -> test-only Pressure追加

Coordinated atomic semantic change-set
  -> test-onlyで有力

Raw file layout as revision authority
  -> reject

One normalized Business semantic review unit
  -> current strongest Acceptance candidate

Lane C generic Acceptance engine
  -> reject

Production revision/hash/transaction API
  -> not adopted

Pre-ADR
  -> Not Ready
```

Actual Repository runner greenとproduction adoption/migration設計が得られるまではADRへ進めない。
