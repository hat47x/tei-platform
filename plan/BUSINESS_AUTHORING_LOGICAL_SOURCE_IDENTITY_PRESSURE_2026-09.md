# Business Authoring Logical Source Identity Pressure — 2026-09

Status: **Lane C / test-only discovery pressure / no production loader adoption**

## 1. 目的

Candidate-only steady-state loader directionでは、physical discoveryとBusiness semantic identityを分離する必要がある。

Current boundary:

```text
workspace source root / ManagedFiles / Source Index
  -> physical managed-source discovery

Business candidate decoder / normalizer
  -> Business semantic identity / meaning
```

したがってsource path、file name、Source Index locatorをBusiness Procedure identityとして扱わない。

## 2. Pressure

`business_authoring_duplicate_logical_source_pressure_test.go`を追加した。

### P1 — different paths / same logical Procedure / same meaning

```text
business/procedures/order-entry.json
business/procedures/archive/same-logical-procedure.json
```

の双方が同じ:

```text
Procedure.id = order-entry.order-handling
```

をauthorする場合、candidate discovery / bundle constructionはfail closedしなければならない。

```text
different path
  != different Business source identity
```

同じlogical identityを二つのphysical sourceがauthorする状態を、たとえsource bytes / Business meaningが同じでもduplicate authorityとしてrejectする。

### P2 — same logical Procedure / conflicting meaning

さらに同じ`Procedure.id`を持つ二つのsourceでBusiness Action meaningを変えたcaseをPressureする。

```text
same Procedure.id
+ different Business meaning
+ different physical paths
```

でも:

```text
merge
first writer wins
last writer wins
lexical path precedence
```

のいずれにも解決しない。

Testは入力path順を反転してもrejectを要求する。

したがって:

```text
path order
  != authority arbitration
```

である。

### P3 — arbitrary path / stable logical Procedure

同じcandidate sourceを任意のphysical file nameへ置いても:

```text
Business Procedure identity
  = source content内のProcedure.id
```

であり、pathから再生成しない。

このPressureはexisting source-move invarianceと整合する。

## 3. Authority implication

```text
file path
  -> locator / physical discovery metadata

Procedure.id
  -> Business logical identity candidate

source authority
  -> accepted discovery/source-set boundary
```

三者を混同しない。

特に次を禁止する。

- pathをProcedure IDとして採用
- duplicate logical sourceをpath順でmerge
- duplicate logical sourceをfirst / last writer winsで解決
- conflicting duplicateをlexical path precedenceで解決
- schemaVersionの大小でduplicate authorityを解決
- legacy directory / candidate directoryというpath名だけでsemantic priorityを決める

Migration previewではlegacy sourceを読むこと自体は許容されるが、candidateとlegacyをaccepted dual authorityへしないという既存cutover contractは変えない。

## 4. Freshnessとの関係

A109のsource-set freshness pressureとは補完関係にある。

```text
source-set membership
  -> 何がauthoritative/reachableか

logical source identity
  -> そのsourceが何をauthorしているか
```

したがってcomplete stale envelopeは少なくとも:

```text
relevant source bytes
+ source add/remove
+ source-root membership
+ logical source uniqueness
+ World / Conformance base
```

をfalse-negativeなく扱える必要がある。

ただし本Pressureはproduction revision token/hash schemaを決めない。

## 5. Production loaderへの含意

Current strongest loader directionは引き続き:

```text
migration adapter
  -> legacy read / candidate preview only

workspace-level physical discovery
  -> candidate source set

strict candidate decoder
  -> logical identity / meaning

candidate bundle construction
  -> duplicate logical identity fail closed

normalize
  -> separate Business semantic plane
```

である。

これはproduction package名、root名、manifest schema、Source Index subject kindを採用する決定ではない。

## 6. Source-level mechanical audit

Current branch上の次を直接読み合わせた。

- `business_authoring_candidate_fresh_process_pressure_test.go`
- `business_authoring_duplicate_logical_source_pressure_test.go`
- candidate disk reload / migration helpers

Source-level確認では:

```text
fresh child process
  -> candidate source paths + World workspaceだけを受領
  -> migration / legacy locatorを受領しない

duplicate discovery
  -> strict candidate decode後のProcedure.idで衝突判定
```

という責務境界に明らかな不整合は見つかっていない。

ただしこれはGo compiler / test runner Evidenceではない。実際のbuild correctnessはactual runnerでのみ確定する。

## 7. Evidence level

現在はsource-level / test-only Pressureでありactual execution Evidenceではない。

HAT47X runnerまたは同等のactual Repository runnerで:

```text
test -z "$(gofmt -l cmd internal examples tools)"
go vet ./...
go test ./...
go test -count=1 ./tools/reference-business-procedure-check
checker baseline
intentional defect checker expected diagnostic
```

が同一revisionでgreenになるまでPASSとは主張しない。

## 8. Current decision

```text
path == Business identity
  -> reject

duplicate logical Procedure across paths
  -> fail closed

same logical identity + conflicting meaning
  -> fail closed

path/input-order precedence
  -> reject

schemaVersion-based authority precedence
  -> reject

production loader/type adoption
  -> deferred until actual Evidence
```
