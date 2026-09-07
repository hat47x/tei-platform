# Business Authoring Workspace Discovery Boundary Pressure — 2026-09

Status: **Lane C / existing-repository discovery pressure / test-only / no production Business loader or source-root adoption**

## 1. 目的

Production loader option auditでは、current primary optionを:

```text
migration adapter
  !=
steady-state candidate loader
```

とし、独立したrepository mode marker `businessAuthoringMode: legacy | candidate` はdeferしていた。

本書では、現行TEI workspace loader / Source Indexの実装から、**mode markerを追加せずにauthority discoveryを表現できる既存mechanismがあるか**をPressureする。

新しいproduction source root名、Business package、`ir.World.Business`、manifest schema変更を採用する文書ではない。

## 2. Existing workspace discovery contract

Current `workspace.Load`はworkspace manifestの:

```yaml
sources:
  - plugins
  - bundles
  - semantic
  - applications
  - databases
  - environments
  - releases
```

を読み、source rootごとにloaderをdispatchする。

重要なのは、未知のsource rootについて:

```go
default:
    // Other source roots are accepted for forward-compatible workspace layout.
```

としており、unknown rootを即`ir.World` semantic kindへ解釈しない点である。

一方 `scanManagedFiles` はmanifestに列挙されたroot配下のphysical filesを `ManagedFiles` として収集する。

したがって既存構造には概念上すでに:

```text
workspace manifest source-root membership
  -> physical managed-source discovery

source-specific loader
  -> semantic interpretation
```

の二段階がある。

## 3. Source Index boundary

`workspace.BuildSourceIndex`は`ManagedFiles`をArtifactへ投影し、source bytesからfingerprintを作る。

Current `sourceindex.Artifact`:

```text
Path
Format
Fingerprint
```

であり、physical metadataは`ir.World`の外側にある。

既存source-move testでも:

```text
same source bytes
+ physical file move
  -> normalized World unchanged
  -> semantic subject unchanged
  -> Artifact locator changes
  -> fingerprint unchanged
```

を確認している。

これはLane Cの:

```text
file path
  != Business semantic identity

source freshness
  != Business semantic revision
```

と整合する。

ただしcurrent Source IndexにBusiness Procedure / Handoff subject kindを追加することまでは本Pressureから導出しない。

## 4. New test-only pressure

追加:

`tools/reference-business-procedure-check/business_model_workspace_discovery_pressure_test.go`

Test-only root名として`business`を用いるが、production名称採用ではない。

### Case 1 — declared Business root

Temporary workspace:

```text
tei.workspace.yaml
  sources:
    - business

business/order-entry-procedure.json

reference/business-system-design/legacy-order-entry-procedure.json
```

を作る。

Expected pressure:

```text
business/... candidate artifact
  -> ManagedFilesに入る
  -> Source Index Artifact / fingerprintを持てる
  -> unknown rootなのでir.World Application / Database meaningへ自動埋め込みされない

reference/business-system-design/... legacy artifact
  -> declared source root外
  -> ManagedFilesへ入らない
  -> Source Index managed artifactへ入らない
```

このcaseは:

```text
legacy file exists
  != legacy authority reachable
```

を既存workspace mechanismで表現できる可能性を補強する。

### Case 2 — physical move inside Business root

同じbytesのcandidate artifactをBusiness root内でmoveし:

```text
Artifact.Path changes
Artifact.Fingerprint stays same
ir.World stays same
```

をPressureする。

これはBusiness semantic identityそのもののmove invarianceを証明しない。
Business semantic identityはfuture Business decoder / normalized modelで別途確認する。

## 5. L4 mode markerへのimplication

Previous L4:

```text
businessAuthoringMode: legacy | candidate
```

はcurrent Evidenceではさらに弱くなる。

Repository / workspaceのauthoritative discoveryが:

```text
manifest source-root membership
+ source-specific strict loader
```

で表現できるなら、別mode fieldは同じ切替を二重管理する。

したがってcurrent rankingを実質:

```text
L2 migration adapter + candidate-only loader
  -> primary

L2 + existing workspace discovery boundary
  -> strongest current physical discovery direction

L4 explicit mode marker
  -> not currently needed; reopen only on concrete discovery insufficiency
```

と読む。

Production Business root名、manifest syntax、loader APIはまだ採用しない。

## 6. Candidate cutover shape

Current repository shapeから考えられる最小physical flow:

```text
PRE-CUTOVER
  legacy Reference / migration Evidence
    -> authoritative only through migration adapter context

  candidate Business source files
    -> staged outside steady-state authoritative discovery

PREVIEW
  migration adapter reads legacy
  candidate strict decode
  normalize
  compare
  cross-view Conformance

ACCEPT
  one repository transaction
    - candidate source set becomes discoverable
    - old migration-only discovery is not part of steady-state loader

POST-CUTOVER / FRESH PROCESS
  workspace-level discovery
    -> candidate Business source set only
    -> Business decoder / loader
    -> normalized BusinessModel
    -> cross-view Conformance with current ir.World
```

Physical implementation may use manifest source-root membership as the discovery switch.

ただし:

```text
Git commit
  != Business semantic revision

manifest revision
  != Conformance result
```

である。

Shared Acceptanceのrevision / stale contractは別途必要。

## 7. Candidate filesをどこにstageするか

Current Evidenceから固定しない。

Options:

1. declared authoritative root外へtemporary/staging write
2. repository transaction内でcandidate files + source-root discoveryを同時create
3. tool-local temporary directoryからPreviewし、Accept時だけauthoritative rootへwrite

重要なのは:

```text
Preview candidate exists
  != authoritative discovery reachable
```

である。

`businessAuthoringMode`を追加してcandidate/legacyのpriorityを管理することは第一候補にしない。

## 8. Source freshnessとの接続

Existing Source Index artifact fingerprintはBusiness source freshnessのphysical primitiveとして再利用可能性がある。

しかし本Pressureから:

- SHA-256をproduction Acceptance tokenへ固定
- Artifact fingerprintをBusiness semantic revisionへ流用
- file pathをlogical source keyへする
- one Artifact = one Business semantic objectと仮定

しない。

Conceptual separationは維持する。

```text
physical Artifact state
  -> freshness evidence candidate

normalized Business meaning
  -> Business semantic revision

current World state
  -> cross-view Conformance base
```

## 9. Shared Acceptance `baseModelRevision`との関係

`ISSUE-000027` / `ISSUE-000046`はcandidateが必要に応じて`baseModelRevision`を持ち、staleなら黙ってlatestへ適用しない原則を既に持つ。

Lane Cからは、そのphysical implementationに対して少なくとも:

```text
Business source base changed
  -> stale detect

World / Conformance base changed
  -> stale or revalidate
```

を要求する。

一つのrepository revision tokenが両方を物理的にcoverできる場合でも、概念上の二条件を消さない。

Lane Cはplatform-wide `baseModelRevision` schemaを定義しない。

## 10. Reopen conditions for explicit mode marker

次の具体的問題が出た場合だけL4-like markerをreopenする。

1. same declared source root内でlegacy / candidate formatが共存し、loader discoveryだけではauthorityを一意にできない。
2. migration期間にcandidate filesをauthoritative root内へ置く必要があり、Shared Acceptance前の非authority状態を他手段で表せない。
3. multiple Business source setsをworkspace levelで切替える必要があり、root membershipだけではlosslessでない。
4. repository transaction境界上、manifest/source discovery switchをcandidate writesとatomicに扱えない。

その場合もmode markerをsemantic identityやschemaVersionと混ぜない。

## 11. Non-adoptions

本Pressureから次を採用しない。

- production root名 `business`
- `ir.World.Business`
- production `internal/business`
- `workspace.Result.Business`
- Business Procedure / Handoff Source Index subject kind
- `businessAuthoringMode`
- manifest schema version変更
- SHA-256をShared Acceptance revision contractへ固定
- legacy filesのphysical deletion要求
- path-based Business identity

## 12. Current conclusion

```text
Existing workspace manifest
  -> explicit physical source-root discovery boundary exists

Unknown source root
  -> managed-file discovery possible
  -> ir.World semantic embedding does not happen automatically

Source Index
  -> managed artifact path / fingerprint can remain outside ir.World

Legacy Reference outside declared root
  -> may remain physically present without becoming managed steady-state source

L4 explicit mode marker
  -> current concrete need not found

L2 + workspace discovery boundary
  -> strongest current physical-loader direction

Production Business loader / source root
  -> not adopted

Actual test execution
  -> still unresolved
```
