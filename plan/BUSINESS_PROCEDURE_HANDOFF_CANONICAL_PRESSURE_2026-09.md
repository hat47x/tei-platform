# Business Procedure Handoff Canonical Pressure — 2026-09

Status: Test-only design pressure / no Core adoption

## 目的

Business Procedure間handoffを、Procedure本文へ埋め込むruntime control flowではなく、Procedure / Step間のBusiness semantic relationとして正規化できるかを確認する。

前段では`order-entry-procedure-reference-set.json`により、Reference-local Procedure key + Step keyでhandoffをfile pathから分離できた。

次は、そのrelation自体をaccepted Business meaningとして扱う最小形をpressure testする。

## 1. 最小relation

```text
BusinessHandoff
  from
    procedure
    step
  to
    procedure
    step
  condition
```

### 入れないもの

- source file path
- fingerprint
- queue
- retry
- timer
- scheduler
- lease
- saga
- runtime state
- execution ownership
- transport
- Authorization Role

handoffを記述できることからruntime orchestrationを導入しない。

## 2. Procedure bodyと分離する理由

handoffは二つのProcedure / Stepを結ぶrelationである。

一方のProcedure本文へ相手側の業務意味をコピーすると、

- N:M relationを局所objectへ押し込む
- duplicate authorityを作る
- file move / rename impactを広げる
- Business FlowとProcedureを混同する

危険がある。

したがってtest-onlyでは、Procedure body meaningとhandoff relationを別normalized projectionとして扱う。

## 3. locator独立性

Canonical handoff endpointは

```text
Procedure key + Step key
```

である。

Reference setの

```text
Procedure key -> artifact locator
```

mappingはauthoring / Source側に残す。

pure file moveでlocator mappingが変わってもhandoff semantic endpointは変化しない。

## 4. conditionはBusiness meaning

handoff conditionの文言変更はBusiness semantic changeとして扱う。

```text
from/to = same
condition = changed
```

であればProcedure body identityは同じでもhandoff relation meaningは変わる。

identity-only shellではこの差も表現できないため、thin Business meaning pressureを補強する。

## 5. fail-closed endpoint resolution

handoffは存在するProcedure / Stepへexact referenceしなければならない。

- missing Procedure
- missing Step
- stale logical rename

を名称類似で補完しない。

Referenceでは既存`validateReferenceSetHandoffs`と新しいnormalized relation probeの両方でこの境界を確認する。

## 6. duplicate relation

同じ

```text
from Procedure/Step
-> to Procedure/Step
+ condition
```

を重複受理しない。

これはduplicate source authorityを防ぐための最小guardであり、将来複数の異なるhandoff conditionを同じendpoint間に持てるかまで固定しない。

## 7. test-only Evidence

`tools/reference-business-procedure-check/canonical_handoff_pressure_test.go`

確認項目:

1. relation fieldを`from / to / condition`へ限定
2. endpoint fieldを`procedure / step`へ限定
3. current Reference setからhandoffを導出
4. Procedure / Step endpointをexact解決
5. artifact locator変更でhandoff meaning不変
6. condition変更でhandoff meaning change
7. Procedure body meaningを不要に変更しない
8. missing endpointをfail closed
9. duplicate relationをfail closed

actual Repository runnerがGo testへ到達するまではgreen Evidenceとは数えない。

## 8. Option Dへの意味

`BUSINESS_PROCEDURE_CANONICAL_MEANING_PRESSURE_2026-09.md`のOption Dを次の二つへ分けられる可能性がある。

```text
Canonical Business Procedure body
  purpose / trigger / actor / ordered step meaning / explicit refs

Canonical Business relation
  handoff between Procedure / Step endpoints
```

これによりBusiness Flow / Procedure / Runtime Workflowを一つの巨大objectへ畳まずに済む。

## 9. まだ固定しないもの

- handoffを`ir.World`のどのfield / packageへ置くか
- handoff relationの専用Canonical kind名
- Business Activity / Eventの別model
- exception flow
- conditional branch syntax
- completion state machine
- runtime queue / retry
- multi-party organization model

current Referenceで必要になった最小semantic relationだけをpressure testする。

## 10. 次の判断

thin Procedure body + thin handoff relationで、

- Business-only diff
- cross-view impact
- file move / logical rename
- manual work
- multiple Procedure -> same Pattern
- Procedure -> Procedure handoff

を説明でき、actual Repository testがgreenなら、production Canonical contractのADR候補を作る価値が出る。

その場合もWorkflow Engine導入は別Issue / 別Evidenceとする。
