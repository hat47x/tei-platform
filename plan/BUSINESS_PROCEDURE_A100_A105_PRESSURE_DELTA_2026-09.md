# Business Procedure A100–A105 Pressure Delta — 2026-09

Status: **Lane C / ISSUE-000042 inventory delta / not a readiness authority / actual runner unresolved**

## 1. 目的

A91–A99以後に追加した、workspace discovery / Source Index / candidate-only disk reloadのPressureを次回`ISSUE-000042` matrix統合用に整理する。

ADR readinessの正本は引き続き:

`BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`

である。

## 2. Delta inventory

| ID | Pressure | Current handling |
| --- | --- | --- |
| A100 | workspace manifestのdeclared source rootをBusiness physical discovery候補にしてもunknown rootは`ir.World` Business embeddingを自動発生させない | test-only implementation / existing loader corroboration |
| A101 | declared Business root外にlegacy Reference fileが存在しても`ManagedFiles` / Source Index managed artifactへ入れず、`file exists != authority reachable`を表現できる | test-only implementation |
| A102 | declared Business managed artifactはSource Index fingerprintを持て、same bytesのphysical moveではlocatorだけが変わり`ir.World`は変わらない | test-only implementation / Business semantic move invariance自体は未証明 |
| A103 | explicit `businessAuthoringMode`を追加せず、migration adapter + workspace/source-root discovery + candidate-only loaderでcutoverを表現できる可能性が高い | option pressure / mode marker further deferred |
| A104 | candidate Procedure + standalone Handoffをdiskへ書き、legacy migration inputを読まずstrict decoderだけでsame normalized BusinessModelを再構成しcurrent WorldへConformanceできる | test-only disk reload pressure |
| A105 | candidate-only disk reloadはunknown `legacyFallback` fieldをfail closedし、steady-state loaderへのlegacy fallback再侵入を許さない | test-only negative pressure |

## 3. A100–A103 — existing workspace mechanismの再利用圧力

Current `internal/workspace/loader.go`はmanifestの`source roots`を読み、known rootだけをsemantic loaderへdispatchする。

Unknown rootはforward-compatible layoutとして受理される一方、`scanManagedFiles`はdeclared root配下のphysical filesを追跡する。

したがって:

```text
source-root membership
  -> physical discovery

source-specific decoder/loader
  -> semantic interpretation
```

を分離できる。

Lane Cではtest-onlyに`business`というroot名を使うが、production名称採用ではない。

Relevant:

- `business_model_workspace_discovery_pressure_test.go`
- `BUSINESS_AUTHORING_WORKSPACE_DISCOVERY_BOUNDARY_PRESSURE_2026-09.md`

## 4. A101 — historical Evidenceとauthorityを物理削除で結ばない

Current pressure shape:

```text
business/... candidate
  -> declared root
  -> managed

reference/business-system-design/... legacy
  -> root外
  -> not managed by this workspace discovery
```

これにより:

```text
legacy file remains in repository
  != steady-state authority reachable
```

を表現できる。

Legacy Evidenceを削除するためだけのmigration cleanupをproduction contractにしない。

## 5. A102 — Source Index artifact stateとsemantic meaningを分ける

Current Source Indexはmanaged artifactに:

```text
Path
Format
Fingerprint
```

を持つ。

Test-only Business artifactをroot内でmoveすると:

```text
Path changes
Fingerprint unchanged
ir.World unchanged
```

をPressureする。

ただしunknown Business rootをcurrent workspace loaderがsemantic decodeしていないため、ここから:

```text
Business Procedure semantic identity move invariant
```

を証明したとは扱わない。

そのEvidenceはfuture Business decoder / normalized model側で別途必要。

## 6. A103 — mode marker further deferred

Previous candidate:

```text
businessAuthoringMode: legacy | candidate
```

は、current repository shapeでは独立正本を増やす可能性がある。

Current strongest physical direction:

```text
migration adapter
  -> legacy read only for preview

candidate source set
  -> non-authoritative staging

Shared Acceptance
  -> candidate files + authoritative discovery switch

steady-state loader
  -> candidate-only discovery
```

Discovery switchをworkspace/source-root membershipで表現できるならmode markerは不要。

Reopenはconcrete insufficiencyが出た場合だけとする。

## 7. A104 — candidate-only disk reload

Existing `business_authoring_authority_cutover_pressure_test.go`はcandidate-only reloadを主にin-memory bundleでPressureしていた。

追加test:

`business_authoring_candidate_disk_reload_pressure_test.go`

では:

```text
migration adapter
  -> candidate source objects生成

candidate source objects
  -> diskへserialize

fresh read path
  -> inline O1 strict decoder
  -> standalone Handoff strict decoder
  -> one BusinessModel normalize
  -> same normalized meaning
  -> current ir.World cross-view Conformance
```

をPressureする。

Reload段階ではlegacy overlay / Reference Set migration inputを読まない。

これはE-Hへ近づくが:

```text
disk reopen in same test process
  != actual fresh OS process
  != production steady-state loader
  != production repository cutover
```

である。

## 8. A105 — no legacy fallback field

Candidate disk sourceへ未知field:

```json
{"legacyFallback": true}
```

を追加したcaseをstrict decoderがrejectする。

目的はfield名そのものを予約することではなく:

```text
steady-state candidate source
  -X-> unknown compatibility shortcut silently accepted
```

を固定することにある。

Production schema採用時もunknown semantic fieldを黙って無視しない原則へ接続する。

## 9. E-H implication

Production adoption Evidence ContractのE-Hは:

```text
accepted cutover revision
  -> new process
  -> candidate-only loader
  -> same Business semantic revision
  -> cross-view Conformance
```

を要求する。

A104/A105でtest-onlyに確認できるのは:

```text
candidate source serialization
candidate-only strict decode
candidate-only normalize
same meaning
Conformance
unknown compatibility field fail closed
```

まで。

未充足:

- actual runner green
- actual fresh process
- production source discovery
- production decoder / loader
- atomic authority cutover
- revision-bound Evidence

したがってE-Hは**partially pressured / not satisfied**のままとする。

## 10. Current decision

```text
A100-A105
  = source-level / test-only delta documented

L2 + workspace discovery
  = strongest physical direction

L4 mode marker
  = concrete need still absent

E-H
  = file-native test pressure advanced
  = actual / production Evidence unresolved

Pre-ADR
  = Not Ready
```
