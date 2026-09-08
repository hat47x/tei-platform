# Business Authoring Fresh-Process Reload Pressure — 2026-09

Status: **Lane C / test-only executable-boundary Pressure / actual execution unresolved**

## 1. 目的

既存の`business_authoring_candidate_disk_reload_pressure_test.go`はcandidate Procedure / standalone Handoffをdiskへ書き、legacy migration inputを読まずsame Go test process内でdecode / normalize / ConformanceできることをPressureした。

ただし:

```text
same-process disk reopen
  != fresh process
```

である。

Contract ADR readinessではproduction loader完成までは要求しないが、少なくともcandidate-only authorityが**別process境界でもlegacy fallbackなしに再構成可能**であるfeasibility Evidenceを要求する。

そのsource-level Pressureとして:

```text
tools/reference-business-procedure-check/
  business_authoring_candidate_fresh_process_pressure_test.go
```

を追加した。

## 2. Pressure構造

Parent test process:

```text
legacy Reference authority
  -> migration preview helper
  -> candidate Procedure sources
  -> standalone Handoff source
  -> expected normalized Business model
```

ここまではmigration setupである。

その後temporary diskへ:

```text
candidate Procedure A
candidate Procedure B
standalone Handoff
```

を書き出す。

同じtemporary rootにはlegacy-looking decoyも残す。

```text
legacy/order-entry-protected-target-overlay.json
```

しかしchild processへ渡すdescriptorにはlegacy locatorを含めない。

Child test process receives only:

```text
candidate Procedure paths
standalone Handoff path
current World workspace path   // cross-view Conformance only
result path
```

Child processではmigration helperを呼ばない。

```text
candidate source paths
  -> strict Procedure decode
  -> strict Handoff decode
  -> one BusinessAuthoringBundle
  -> normalized Business model
  -> current World Conformance
  -> normalized result
```

Parentはchild resultをcurrent expected Business semantic fingerprintと比較する。

## 3. なぜ`os.Args[0]`を再起動するか

Test-only feasibilityのため、production Business loader packageやCLI commandを新設しない。

Go test executable自身を:

```text
-test.run=^TestBusinessCandidateOnlyFreshProcessReloadHelper$
```

でchild processとして起動する。

これにより:

- OS process boundaryを持つ
- parent processのin-memory candidate objectを共有しない
- childはdisk sourceだけからBusiness meaningを再構成する
- production directory convention / manifest / registryを先取りしない

を同時に満たす。

Descriptorはtest apparatusでありproduction source-discovery contractではない。

## 4. `legacy file exists != authority reachable`

Legacy-looking decoyはdiskに存在する。

しかしchildへlocatorを渡さない。

したがってPressureしたい境界は:

```text
file exists
  != authoritative source reachable
```

である。

これはlegacy fileを物理削除しないとmigrationできない設計を避ける。

ただし「任意のfuture production discovery implementationが絶対にlegacyを読まない」ことをこのtestだけで証明するものではない。

Production adoption時にはactual source-root / authority discovery mechanismで同じ境界を再実証する。

## 5. Semantic比較

Child resultはJSON boundaryを通る。

そのため`reflect.DeepEqual`ではなく既存の:

```text
semanticFingerprintD4O1Probe
```

で比較する。

これにより`omitempty`等によるnon-semantic `nil` / empty representation差をBusiness semantic changeと誤認しない。

## 6. このPressureが証明しようとするもの

Actual greenが取得できた場合、Contract ADR feasibilityとして次を支持する。

1. candidate Procedure + standalone Handoffがprocess-local migration stateなしで再読込できる。
2. legacy migration locatorをchild processへ渡さなくてもcurrent normalized Business meaningを再構成できる。
3. candidate-only reloaded meaningをcurrent Worldへcross-view Conformanceできる。
4. production `businessAuthoringMode`やpermanent dual-readをContract requirementにしなくてよい可能性が高い。

## 7. このPressureが証明しないもの

```text
source-level test exists
  != actual PASS
```

である。

さらにactual PASS後でも次は別Evidence。

- production Business decoder / loader
- production workspace source-root discovery
- production complete stale envelope
- production multi-file atomic writer
- authority-discovery atomic cutover
- failure時partial write safety
- Studio / AI shared Acceptance end-to-end

したがって:

```text
fresh-process test PASS
  -> Contract ADR feasibility Evidence

production candidate-only reload PASS
  -> Production adoption Evidence
```

を分離する。

## 8. Current status

2026-09-08時点:

```text
source-level fresh-process Pressure
  = implemented

actual execution
  = unresolved

HAT47X
  = offline

Contract ADR blocker #2
  = remains unresolved until actual execution
```
