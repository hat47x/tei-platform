# Business Authoring Existing Acceptance Implementation Reuse Audit — 2026-09

Status: **Lane C / implementation reuse audit / no production Acceptance API adoption**

## 1. 目的

Lane Cのcurrent minimum candidateはsemantic / authoring / migration方向までかなり絞れている。

```text
D4 + singular narrow O1 + S4-A + H1/R3

migration adapter
  -> legacy-read preview
  -> candidate Procedure + standalone Handoff

Shared Acceptance
  -> one authority cutover

steady-state
  -> candidate-only load / reload / Conformance
```

残るPre-ADR blockerのうち大きいものは:

- production candidate loader / discovery
- complete stale detection
- atomic source apply / authority cutover
- fresh-process candidate-only reload

である。

ここでLane C専用のgeneric transaction / revision / Acceptance engineを新設する前に、current TEI Repositoryに既に存在するimplementation primitive / architectural precedentを監査し、**再利用可能なものと未実装gapを分離する**。

本書の監査範囲はcurrent branchで直接確認した:

- `internal/workspace`
- `internal/sourceindex`
- `internal/workbench/database`
- `internal/consumerimpact`
- `internal/releasegraph`
- `internal/deployment/durable`
- `cmd/tei`

である。

したがって「Repository全体に絶対存在しない」という証明ではない。Current audited implementation surfacesにgeneric Business authoring Acceptanceが確立されているかを判定する。

## 2. Reusable primitive A — Workspace source-root discovery

`internal/workspace/loader.go`は`tei.workspace.yaml`のexplicit `sources:`を読み、declared source rootをphysical discovery boundaryとして扱う。

Known rootは専用loaderへdispatchし、unknown rootはforward-compatible layoutとして受理されるが、その名前だけで`ir.World`へ新しいsemantic planeを埋め込まない。

その後`scanManagedFiles(root, roots)`によりdeclared root配下のmanaged filesを収集する。

Lane Cへの意味:

```text
workspace source-root discovery
  -> reusable physical source-set boundary

unknown future `business` root
  -X-> automatic ir.World.Business
```

Current Pressureでは、これをcandidate-only Business authority reachabilityの候補にできる。

ただしworkspace loader自体は:

- Business candidateをdecodeしない
- candidate authorityを決めない
- Preview / Accept / Rejectを行わない
- multi-file writeをatomicにしない

ため、Shared Acceptance engineではない。

## 3. Reusable primitive B — Source Index artifact identity / fingerprint

`internal/sourceindex/model.go`には:

```go
type Artifact struct {
    Path        string
    Format      string
    Fingerprint string
}
```

があり、fingerprintはsource contentから計算される。

`Index.Validate`はmanaged sourceをreadし直し、current content fingerprintがindex時点と異なればstaleとしてfailする。

既存`internal/workspace/source_index_invariance_test.go`では、same source bytesをphysical moveしても:

- normalized `ir.World`は同じ
- semantic subjectは同じ
- fingerprintは同じ
- locatorだけが変わる

ことをPressureしている。

Lane Cへの意味:

```text
physical locator
  != semantic identity

source content fingerprint
  -> reusable freshness primitive
```

である。

ただしSource Indexはaccepted candidate base全体を一つのShared Acceptance revisionとして定義しない。Business source setとWorld / Conformance baseを束ねるcomplete stale envelopeも自動では作らない。

## 4. Reusable precedent C — Workbench is projection, not authority

`internal/workbench/database/model.go`は明示的に:

> Model is a UI-oriented projection ... must never become a semantic or query source of truth.

という境界を持つ。

`Build(index.Derived)`は既存derived indexからreview-oriented modelを構築し、source / Canonicalをmutationしない。

Lane Cへの意味:

```text
Preview / review projection
  != authoritative source
```

というTEI既存architecture precedentである。

これはBusiness authoringでも:

```text
candidate
  -> Semantic Diff / impact / review projection
  -X-> second source of truth
```

を維持する根拠になる。

ただしWorkBenchは:

- authoring candidate store
- stale candidate check
- Acceptance Policy
- authoritative file write
- atomic cutover

を所有しない。

したがってそのままShared Acceptance implementationとして再利用しない。

## 5. Reusable precedent D — Consumer Impact / Release Graph are read-only decision projections

`internal/consumerimpact.AnalyzeTarget`はcurrent deployment snapshotとtarget database releaseからconsumer impact / safetyをderiveする。

`internal/releasegraph.Build`もimmutable Application releaseとindependently versioned database statesからdeployability graphをderiveし、commentにもglobal application/database versionやdistributed transaction boundaryを発明しないことが明示されている。

Lane Cへの意味:

```text
current accepted state + candidate/target
  -> deterministic impact projection
  -> review / decision support
```

という既存patternがある。

これは`ISSUE-000039`のread-only Semantic diff / impact ownershipと整合する。

一方、両packageともauthoritative Business sourceへのwrite / cutoverを所有しない。

## 6. Reusable precedent E — Deployment stale-reject / bound-state checks

`cmd/tei/main.go`にはdurable deployment commands向け:

```text
-expected-revision
```

があり、`loadCheckedDeploymentRun`で:

```go
if expectedRevision >= 0 && run.Revision != expectedRevision {
    return ... revision conflict
}
```

と明示的にstale runをrejectする。

さらに`internal/deployment/durable`は:

- `PlanFingerprint`
- `CheckPlan`
- `CheckPluginLock`
- `CheckRuntimePolicy`
- immutable action-structure check
- state meaningが変わった場合だけ`Run.Revision`をincrement

を持つ。

ここから借りられるarchitecture precedentは:

```text
expected revision mismatch
  -> fail closed

bound input fingerprint mismatch
  -> fail closed

accepted state transition
  -> revision changes
```

である。

### 重要な非流用境界

これはBusiness authoring Shared Acceptanceそのものではない。

Deployment durable stateは:

- deployment run artifact
- runtime / rollout state machine
- command / observation / evidence lifecycle

を扱う。

Business authoringは:

- authoritative source files
- normalized Business semantic meaning
- cross-view Conformance
- source-discovery authority cutover

を扱う。

したがって:

```text
Deployment Run.Revision
  -X-> Business semantic revision

Deployment state machine
  -X-> Business Workflow

Deployment expected-revision wrapper
  -X-> generic source transaction engine
```

とする。

再利用するならprinciple / small utility levelまでであり、Deployment packageへBusiness authoring責務を載せない。

## 7. Current audited surfacesで確立されていないもの

今回直接監査したsurfaceでは、次を一体で担うgeneric implementationは確立されていない。

### 7.1 Generic authoring candidate boundary

```text
current source state
+ candidate change-set
+ baseModelRevision
```

を共通形式で保持し、file-native / Studio / AI / CLIから同じAcceptanceへ送るproduction implementationは確認できていない。

Issue / ADR上のdesign principleは存在するが、実装済みと読み替えない。

### 7.2 Complete stale-envelope builder

Business source setとWorld / Conformance baseをまとめて:

```text
no false-negative stale
```

で検証するproduction implementationは確認できていない。

Workspace / Source Index fingerprintはprimitiveとして使えるが、complete envelope ownershipは別途必要。

### 7.3 Atomic multi-file authoritative source apply

```text
Procedure source changes
+ standalone Handoff changes
+ authority discovery switch
```

をpartialに残さないproduction transaction boundaryは、今回監査したsurfaceでは確認できていない。

### 7.4 Production Business candidate decoder / loader

Current inline O1 / standalone Handoff decoderはLane C test-only Pressureであり、production packageではない。

### 7.5 Fresh-process candidate-only Business reload

Current test-only disk reloadはsame Go test process内でcandidate filesをclose/reopenしてstrict decodeするところまでである。

```text
same-process disk reload
  != fresh OS process
  != production loader
  != accepted source cutover
```

である。

## 8. Implementation gapをschema Evidenceへ読み替えない

Generic Shared Acceptance implementationがcurrent auditで確立されていないことは、次の根拠ではない。

- `ir.World.Business`を追加する
- Business Procedureへrevision/hash fieldを追加する
- generic transaction engineをBusiness package内へ作る
- `businessAuthoringMode`を必須にする
- schemaVersionでauthority priorityを決める
- Business ProcedureへCRUD/effectを追加する
- HandoffへWorkflow fieldを追加する

Implementation gapはimplementation gapとして扱う。

## 9. Current strongest integration architecture

Production API名を決めず、責務だけを置くと:

```text
Business authoring adapter
  ├─ workspace source-root discovery
  ├─ Source Index / source fingerprint primitives
  ├─ strict Business source decode
  └─ normalize Business candidate
        ↓
Shared Acceptance boundary        // cross-lane ownership
  ├─ complete base stale check
  ├─ deterministic Validation / Conformance
  ├─ Semantic Diff / impact projection
  └─ Acceptance Policy
        ↓ Accept only
repository/source update boundary
  ├─ candidate source writes
  └─ authority-discovery cutover
        ↓ atomic
fresh process
  ├─ candidate-only Business loader
  ├─ normalize same Business meaning
  └─ current workspace WorldへConformance
```

ここでLane Cが新規所有するのはBusiness-specific adapter contractであり、Shared Acceptance substrateそのものではない。

## 10. Reuse / do-not-reuse table

| Existing implementation | Reuse direction | Do not reinterpret as |
| --- | --- | --- |
| workspace `sources:` / ManagedFiles | authoritative physical source-set discovery primitive | Business semantic authority itself |
| Source Index Artifact / fingerprint | source freshness / trace primitive | complete Shared Acceptance revision |
| Workbench projection | Preview / review separation precedent | second source of truth / Acceptance engine |
| Consumer Impact | deterministic impact analysis precedent | authoritative apply |
| Release Graph | cross-state dependency / deployability projection precedent | global version / transaction authority |
| Deployment `expected-revision` | stale-reject precedent | Business semantic revision |
| Deployment fingerprints | bound-input freshness precedent | Business source base model |
| Deployment durable state machine | no direct reuse | Business Procedure / Workflow engine |

## 11. Pre-ADR implication

Production blocker #2 / #3は次へ縮約できる。

### Available primitives

- source-root discovery
- managed file enumeration
- source path normalization
- source content fingerprint
- stale fingerprint validation
- workspace reload / normalization
- read-only projection / impact patterns
- explicit stale-revision rejection precedent

### Still required

1. production Business source decoder / loader boundary
2. Business source set + World / Conformance baseをcoverするcomplete stale envelopeへの接続
3. cross-lane Shared Acceptance implementationとのintegration
4. candidate source writes + authority discovery switchのatomic apply
5. fresh-process candidate-only reload / Conformance Evidence

したがって「何もないのでゼロからGeneric Acceptanceを作る」状態ではない。

一方「既存primitiveが揃ったのでproduction adoption Ready」でもない。

## 12. Current decision

```text
workspace / Source Index reuse
  = strong candidate

Workbench / impact packages
  = projection architecture precedent

Deployment expected-revision / fingerprint
  = stale-reject precedent only

existing generic source-authoring Shared Acceptance implementation
  = not established in current audited implementation surfaces

Lane C generic transaction engine
  = not adopted

Business-specific adapter contract
  = production prototype時のnext implementation boundary

Pre-ADR
  = Not Ready
```

この監査により、残るimplementation workはBusiness semantic schema expansionではなく、**既存source/fingerprint primitivesとcross-lane Shared Acceptanceを接続するadapter + atomic source update/reload Evidence**へさらに限定される。
