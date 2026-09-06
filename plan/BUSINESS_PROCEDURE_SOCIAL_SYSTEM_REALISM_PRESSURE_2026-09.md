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
BP-06  security-boundary
BP-07  application-use -> order_entry.submit_order
BP-08  observation-boundary
BP-09  manual-external-unresolved
```

`order_entry.submit_order`は`sales.order.submit` Capabilityへtraceする。

重要なのは:

```text
Command
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
6. submit Patternがaccepted Command Capabilityへtraceする
7. manual / security / observation / external stepsへfake Application Pattern / Capabilityを要求しない
8. Command -> Observation -> external continuationのBusiness順序を保持する

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
- security
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

Function-Interaction / Data
  remain independently authoritative where they own meaning

cross-view validation
  checks explicit references and contradictions
```

一方、historical X-TEAのBusiness Flow / DFD全体をHandoff modelへ流用しない。

Gate 15bはBusiness Flow schemaの必要性を証明するものではない。

## 6. D3 / H1 / R3 / S4-Aへの影響

15bから新しいproduction fieldは要求されない。

現時点の主候補:

```text
D3
  thin Business Procedure residual meaning
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

で、current social-system realismを説明できる。

特に次は追加しない。

- single Application owner
- Business Flow / DFD graph
- Workflow runtime state
- external organization runtime abstraction
- fake Capability for manual work
- generic Core Edge

## 7. Evidence level

現在確認できたのはRepository source / Reference構造とtest実装である。

actual `gofmt / go vet / go test`は、この作業時点でRemote Desktop実行端末が接続されていないため未実行。

したがって:

```text
Gate 15b source-level Pressure: Implemented
Gate 15b actual execution Evidence: Pending
Pre-ADR: Not Ready
PR: Draft維持
```

とする。

## 8. 次のPressure

1. actual Repository environmentで`gofmt / go vet / go test`を取得する
2. 15b testがgreenなら、D3がcurrent social-system Referenceで不足を出すか再監査する
3. 不足がなければrealistic second Applicationを概念追加の口実にしない
4. Procedure / Handoffだけでは保持できない具体的Business Flow meaningが出た場合のみBusiness Flow pressureを別途reopenする
