# Actor Context R3R — Projection Dependency Provenance / Freshness Planes

- Status: ISSUE-000053 / Phase 3 dependency-provenance pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3Q_PROJECTION_SELECTIVE_FRESHNESS_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_PROJECTIONS_2026-09.md`
  - `plan/ACTOR_CONTEXT_R0_SOURCE_MAP_2026-09.md`
  - `docs/adr/0004-source-to-canonical-traceability.md`
  - `internal/sourceindex/model.go`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Qではpurpose-bound projectionのfreshnessを:

```text
profile identity
  + selected semantic category dependencies
```

へ絞った。

次の問いは、そのselected categoryが何へ依存しているかを、actual TEI source / runtime Evidenceまでどう逆追跡するかである。

ここで全dependencyを一つのhashへ潰すと、少なくとも次が再び混同される。

```text
semantic meaning changed
source locator moved
Reference artifact changed
runtime Authorization Observation changed
projection profile changed
```

R3Rではこれらを**freshness plane**として分離する。

## 2. 四つのdependency plane

R3R test-onlyでは少なくとも次を別に保持する。

### A. Semantic subject dependency

Source Index管理下のaccepted / authored subjectについて:

```text
subject kind
subject id
content fingerprint
current locator
source role
```

を保持する。

freshness identityは:

```text
subject identity + content fingerprint
```

であり、locatorはdrill-down解決用である。

### B. Reference artifact dependency

Business Procedure Referenceやpressure overlayのように、current Source Index subjectへ正規化されていないReference inputは別planeへ置く。

```text
reference input class
content fingerprint
current locator
```

を保持する。

Source Index subject identityが存在しないものへ、架空のCanonical subjectを与えない。

### C. Runtime Observation dependency

Authorization evaluation等のruntime Evidenceはstatic source artifactではない。

少なくともtest-onlyに:

```text
Business address
principal / subject
action
resource
outcome
observation context
```

を保持する。

これをRepository source fingerprintへ変換しない。

### D. Projection profile dependency

R3Qの:

```text
audience
purpose
selected / omitted categories
```

から得るProfileFingerprintを独立に保持する。

## 3. semantic freshnessとlocator integrityを分ける

Source Indexはsemantic subject identityとartifact pathを分離している。

既存invariance Evidence:

```text
subject unchanged
content fingerprint unchanged
artifact path changed
  -> semantic meaning unchanged
  -> locator changed
```

R3Rではこの場合:

```text
semantic freshness = current
locator integrity = refresh-required
```

と表現できるようPressureする。

これはR3Qの `source-refs` categoryがselectedならbounded projectionを更新する必要があることと矛盾しない。

```text
projection bytes / drill-down locator need refresh
  != selected semantic meaning stale
```

を分離する。

## 4. content changeはsemantic stale

同じsubject identityでもcontent fingerprintが変われば:

```text
semantic freshness = stale
```

である。

pathが同じかどうかは関係しない。

```text
same locator
  != same meaning
```

を維持する。

## 5. Reference artifactはSource Index subjectへ偽装しない

current Business Procedure fixtureやprotected-target overlayはR3M〜R3Qのstop projectionへ実際に寄与している。

ただしcurrent source model上、それらを:

```text
subject = business-procedure:...
subject = protected-target:...
```

とproduction Source Index identityへ勝手に追加しない。

R3RではReference artifact dependencyとして:

- Business Procedure Reference bytes
- protected-target overlay bytes

のfingerprint / locatorを保持できることだけをPressureする。

将来Business Procedure production authorityが成立した場合のSource Index統合は別設計である。

## 6. runtime decision freshnessはstatic source freshnessではない

同じRepository revision / same Source Indexでも:

```text
Authorization Observation = allowed
Authorization Observation = denied
```

はruntime contextとして異なる。

したがって:

```text
static source current
  != runtime decision current
```

である。

逆にruntime Observationだけが変わっても、Application Pattern source contentが変わったことにはならない。

R3Rではstatic / runtime dependencyを一つのCombinedFingerprintへ畳まない。

## 7. current BP-06への適用

current action-authority stopでselected meaningへ関係する入力を分類すると、概念上:

```text
Reference artifact
  order-entry-business-procedure.json
    -> requirement = required

Reference artifact
  order-entry-protected-target-overlay.json
    -> protected resource = order_entry.submit_order

Source Index subject
  application-pattern:order_entry.submit_order
    -> current Application Pattern drill-down / corroborating source context

Runtime Observation
  none yet
    -> evaluation has not occurred

Projection profile
  Human orientation / Agent resolution-inspection
```

となる。

Application Pattern subjectはprotected resource authorityそのものへ昇格させない。
Source Index traceabilityを持つこととBusiness protected-target authorityは別である。

## 8. denied caseへの適用

test-only explicit target/actionとexact current Observationを使うdenied caseでは:

```text
static subject / Reference dependencies
  = unchanged

runtime Observation dependency
  outcome = denied
```

となる。

この変化はruntime freshness planeを変えるが、static semantic subject fingerprintを変えない。

## 9. test-only source

- `tools/reference-business-procedure-check/actor_context_r3r_projection_dependency_provenance_pressure_test.go`

Pressureするcase:

1. current Source Indexから `application-pattern:order_entry.submit_order` のsubject / artifact fingerprint / locatorを取得する
2. locator-only move candidateではsemantic dependencyはcurrent、locator refreshのみ必要とする
3. same subject + changed content fingerprintではsemantic staleとする
4. Business Procedure / overlayをReference artifact dependencyとして保持し、Source Index subjectを捏造しない
5. runtime Observation outcome changeはruntime planeだけを変える
6. static Source Index unchangedをruntime decision freshnessの証明にしない
7. ProfileFingerprintをstatic/runtime dependencyへ混ぜない
8. CombinedFingerprint / GlobalRevision / Current=true等のshortcutを作らない

## 10. current conclusion

R3Rで固定するのは:

```text
projection dependency provenance
  = profile plane
  + semantic subject plane
  + reference artifact plane
  + runtime observation plane
  + locator integrity
```

である。

```text
one global hash
  != sufficient explanation of freshness
```

を明示する。

production invalidation engine、runtime Observation store、Business Source Index extension、global revision protocolは追加しない。

## 11. 次

次段では、selected categoryからこれらdependency planeへの**dependency reason**を保持できるかをPressureする。

単にfingerprint集合を持つだけでなく:

```text
machine-reasons
  <- depends on requirement source + stop classifier

known-authority-facts
  <- depends on Business Reference / overlay

runtime-outcome
  <- depends on exact current Observation

source-refs
  <- depends on current locator resolution
```

のように、「なぜこのinputが必要か」を説明できることを目指す。

これはimpact analysis / incremental rebuild / agent explanationの共通基盤になり得るが、production dependency graph採用はまだ行わない。