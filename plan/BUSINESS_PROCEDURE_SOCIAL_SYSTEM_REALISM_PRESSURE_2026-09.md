# Business Procedure Social-System Realism Pressure — 2026-09

Status: **Lane C / Gate 15b source-level Pressure implemented / actual runner Evidence pending / no production Core adoption**

## 1. 問い

Business Procedureのproduction Canonical contractを判断する前に、syntheticなone Procedure -> multiple Applicationsだけでなく、current Referenceが現実の業務システムに特有の境界をどこまで保持できているか確認する。

ここでいうrealismはApplication数ではない。

```text
業務上の一つの仕事
  !=
一つのApplication内部の機械処理
```

として、少なくとも次が同じProcedure文脈に共存できることをPressureする。

- managed-here data
- external-observed data
- read/write policy差
- human judgment / unresolved work
- security boundary
- Application Command
- separate Observation
- manual / external continuation

## 2. current `order_entry`にすでにある境界

### 2.1 managed-here sales data

`order_entry`は`sales` datasourceを`read-write`で利用し、targetは`sales_core`である。

`sales_core`自身は:

```text
ownership managed-here
```

を持つ。

### 2.2 external-observed customer master

同じ`order_entry`は`master` datasourceを`read`で利用し、targetは`master_data`である。

`master_data`自身は:

```text
ownership external-observed
```

を持つ。

`order_entry.customer_reference`はこの`master` datasourceを明示し、`master_data.customer / find_by_id` Contract Queryを呼ぶ。

したがってcustomer参照は、Business Procedure fixtureへ外部masterの意味を複製せずにnormalized Worldからtraceできる。

### 2.3 human / security / command / observation / continuation

Primary Business Procedureは次を同じordered Procedureに保持する。

```text
BP-04  manual-unresolved
BP-06  security-boundary -> security.authorize
BP-07  application-use   -> order_entry.submit_order
BP-08  observation-boundary
BP-09  manual-external-unresolved
```

`order_entry.submit_order`は`sales.order.submit` CapabilityへPatternからtraceする。

一方BP-06にはApplication Patternがなく、Business Action自体がAuthorizationを明示している。Reference Workspaceには既存`security.authorize` Capabilityがあるため、BP-06からこれをexplicit semantic refとして保持するようReferenceを補強した。

これはBusiness ProcedureへSecurity定義を複製することではない。

```text
Business Procedure BP-06
  -> explicit Capability ref security.authorize
  -> existing Semantic authority
```

とするcross-view linkである。

重要なのは:

```text
Authorization
  -> Command
  -> separate Observation
  -> manual / external continuation
```

という順序であり、Command acknowledgementを外部現実の完了へ読み替えない。

## 3. 実装したPressure

`tools/reference-business-procedure-check/social_system_realism_pressure_test.go`

に、production Reference Workspaceだけを使うtestを追加した。

確認すること:

1. baseline Business Procedure cross-view evaluationにblocking diagnosticがない
2. synthetic `fulfillment_console`へ依存しない
3. `sales` -> `sales_core`がread-write + managed-here
4. `master` -> `master_data`がread-only + external-observed
5. customer reference Patternがexplicit datasource + Contract Query traceを持つ
6. BP-06がApplication Patternを捏造せず`security.authorize`へexplicitにtraceする
7. submit Patternがaccepted Command Capabilityへtraceする
8. manual / observation / external residual stepsへfake Application Pattern / Capabilityを要求しない
9. Authorization -> Command -> Observation -> external continuationのBusiness順序を保持する

このtestはtest-only Evidenceであり、`ir.World`へBusiness Procedureを追加しない。

## 4. Gate 15を分割する

従来の「realistic multi-Application」を一つの条件として扱うと、Application数と業務realismが混同される。

今後は次へ分離する。

### 15a — multi-Application structural pressure

```text
one Procedure
  -> Application A
  -> Application B
  -> manual
  -> external
```

synthetic second Applicationで構造上の非所有関係をPressureする。

目的は:

- Business Procedure != single Application child
- Pattern owner Applicationをexact resolutionする
- fixture Applicationをsemantic ownerにしない

である。

### 15b — social-system realism pressure

current production `order_entry` Referenceだけを使う。

目的は:

- managed-hereとexternal-observedの共存
- human judgment
- existing Authorization semantic authorityへのexplicit cross-view trace
- Command / Observation separation
- manual / external continuation

を一つの業務文脈で保持すること。

**source-level Pressureを実装済み。actual Repository runnerでgreenになるまでは実行Evidenceとは数えない。**

### 15c — realistic second-Application pressure

実在性を持つ第二Application / 第二業務領域を追加するのは、15a + 15bで説明できないaccepted meaningが出た場合だけとする。

Application数を成熟度KPIにしない。

## 5. Watanabe / X-TEA pressureとの関係

渡辺幸三氏の業務モデルを「業務マニュアル」「社会システム」として捉える観点では、業務手順はソフトウェア内部処理だけから構成されない。

人、外部組織、既存データ、画面利用、判断、次の仕事を含む。

このPressureからTEIへ持ち込むのは次の設計原則である。

```text
Business Procedure
  keeps residual social/business meaning

Function-Interaction / Data / Security
  remain independently authoritative where they own meaning

cross-view validation
  checks explicit references and contradictions
```

一方、historical X-TEAのBusiness Flow / DFD全体をHandoff modelへ流用しない。

Gate 15bはBusiness Flow schemaの必要性を証明するものではない。

## 6. D3 / H1 / R3 / S4-Aへの影響

15bから新しいproduction kindはまだ要求されない。

現時点の主候補:

```text
D3
  thin Business Procedure residual meaning
  + explicit semantic refs
  + Step body / addressability separation

H1
  thin normalized handoff meaning

R3
  handoff-only Business relation

S4-A
  Procedure-scoped Step local address

C1
  optional fixture Application context
```

でcurrent social-system meaningを保持できるPressureを追加した。

特にBP-06は、`security-boundary`というauthoring labelをproduction enumへ昇格させず、existing `security.authorize` identityへのexplicit refでcross-view意味を保持できる。

ただしBP-08 Observationには、current Business Procedure fixtureからexplicitに参照できる同等のSemantic identityをまだ置いていない。

したがってD3について現在主張できるのは:

```text
Business wording preservation        yes
explicit Authorization trace         yes
machine-readable Observation kind    not proven
```

である。

Observationを文字列解釈や`realization` labelでCanonical判定しない。

特に次は追加しない。

- single Application owner
- generic `StepKind` enum
- Business Flow / DFD graph
- Workflow runtime state
- external organization runtime abstraction
- fake Capability for manual work
- generic Core Edge

## 7. D3 social meaning Pressure

`tools/reference-business-procedure-check/d3_social_system_meaning_pressure_test.go`を追加した。

確認すること:

- `realization`をsubtractしたD3でもBP-04 / BP-06 / BP-08 / BP-09のBusiness wordingを保持
- BP-06の`security.authorize` explicit refを保持
- BP-07の`order_entry.submit_order` explicit Pattern refを保持
- manual / observation / external residual stepへfake Capabilityを足さない
- `security-boundary` / `observation-boundary`等のauthoring label変更だけではD3 meaningを変えない
- current D3がtyped `SecurityBoundary` / `ObservationBoundary` Conformanceを持つと過大主張しない

これはD3を完成扱いするtestではなく、**何を保持でき、何をまだtypedに検証できないか**を固定するPressureである。

## 8. Evidence level

現在確認できたのはRepository source / Reference構造とtest実装である。

actual `gofmt / go vet / go test`は、この作業時点でもRemote Desktop実行端末が接続されていないため未実行。

したがって:

```text
Gate 15b source-level Pressure: Implemented
D3 social meaning source-level Pressure: Implemented
Gate 15b actual execution Evidence: Pending
Pre-ADR: Not Ready
PR: Draft維持
```

とする。

## 9. 次のPressure

1. actual Repository environmentで`gofmt / go vet / go test`を取得する
2. `security.authorize` explicit refを含む15b / D3 testsをgreenにする
3. Observationについてtyped cross-view identityが本当に必要か、既存Driver / Observation authorityからderiveできるかを確認する
4. 不足がなければrealistic second Applicationやgeneric StepKindを概念追加の口実にしない
5. Procedure / Handoffだけでは保持できない具体的Business Flow meaningが出た場合のみBusiness Flow pressureを別途reopenする
