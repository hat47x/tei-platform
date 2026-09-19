# Second Application Reference — Gate C Entry Record

- Status: Entry Evidence complete / S0 complete
- Date: 2026-09-19
- Primary workstream: ISSUE-000040
- Start packet: `plan/GATE_C_SECOND_APPLICATION_START_PACKET_2026-09.md`
- Gate C judgment: `plan/GATE_C_READINESS_AUDIT.md`
- Canonical second Application created: No
- Shared Acceptance attempted: No
- Environment binding created: No
- Target implementation started: No

## 1. Purpose

Gate C Ready後の第二Application Referenceを開始するため、Start Packet S0のentry Evidenceを一つに固定する。

このrecordは第二Applicationのsourceでも、Application identityのDecisionでもない。

```text
Gate C Entry Evidence complete
  != destination Application accepted
  != destination source authored
  != Environment binding created
```

## 2. Gate C decision

```text
gateC.status = ready
handoverBaseline = B
baselineRepositoryRevision R = e5231839a5dcc62cce1181a2731fcb7c70d16736
targetBuildRevision T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe
targetAdapter = reference-first-web-target
host = go-net-http
```

final Gate C readiness Evidence:

```text
gate-c-readiness-audit.json
SHA-256 = b796e38db9a249ef98f2cb600eb354afc70f9adf57d2c83d47116f9968a0ff6c
```

Gate Cの判定規則を本recordで再実装しない。`GATE_C_READINESS_AUDIT.md`のReady判定を外部authority inputとして利用する。

## 3. Independent Evidence classes

### Semantic Conformance

Semantic / Functional same-run summary:

```text
semantic-functional-summary.json
run = first-web-target-0ba87a4e0a96
SHA-256 = 78c4c3b1031a70eb62683233fb5f1794cf7f71fd8b420ffa741d5ebe3f649d6b
```

このsummaryでSemantic Conformanceはpassしている。

### Interaction / Functional

同じsummaryのFunctional Evidenceがpassしている。

successful pathだけでなく、

- read-only browse / record / reference / header-detail
- Authentication denied
- Authorization denied
- Canonical Validation not-satisfied
- Command Invocation Execution Error
- Command Business Result
- separate Observation
- standard / Custom UI shared Command boundary

を別Evidenceとして保持している。

### Visual Fidelity

```text
visual-audit.json
SHA-256 = 61858f3d89891381684b5f8a363fbbfa0471ae313286cf239e2903aa800e2180
visual run = visual-T
```

required captures / criteriaはpass済み。

Visual evaluatorは単一AI evaluatorで、人による確認ではない。この制約はEvidence attributionとして保持するが、final Gate C auditではblocking unresolvedではない。

## 4. Target Difference Handoff

First Web Targetで観測された差をsecond Application設計へ暗黙に混ぜない。

### Target Capability candidates

- `http-html-projection`
- `http-json-manifest`
- `http-read-scenario`
- `http-command-scenario`
- `http-custom-submit-form`

これらは第一Targetの観測候補であり、Target Capability taxonomyの正本ではない。

### Runtime Binding observations

- `order_entry.recent_orders`
  - datasource `sales`
  - Database `sales_core`
  - access `read-write`
  - current Driver APIではEnvironment instance selectionを露出しない
- `order_entry.customer_reference`
  - Environment `development`
  - instance `master_data.dev`
  - provider `oracle`
  - executor `database.query.oracle`
  - Contract Query `master_data/master_data.customer/find_by_id`
- `order_entry.order_lines`
  - Environment `development`
  - instance `sales_core.dev`
  - provider `postgres`
  - executor `database.query.postgres`
  - Contract Query `sales_core/sales_core.order_line/find_by_order`

### Execution / Host observation

```text
executionArtifact = go-run-native-process
ABI = linux/amd64
Host = go-net-http
```

actual Gate C environment:

- Ubuntu 22.04 / WSL2
- Go `go1.25.14 linux/amd64`
- loopback bind

### Web-specific, do not canonicalize

- HTTP route
- HTTP status
- HTML element
- CSS class
- Reference synthetic fixture value

second ApplicationのBusiness / Semantic meaningをこれらから逆設計しない。

## 5. Carry-forward unresolved

Gate C Readyを妨げなかったが、後続設計で成功へ読み替えない。

- `actual-external-command-observation-not-acquired`
- `browse-environment-instance-binding-not-exposed-by-current-driver-api`
- `general-target-neutral-presentation-taxonomy-not-yet-defined`
- `target-capability-taxonomy-remains-candidate-until-second-target`

これらは第二Application S1/S2のqualificationを自動失敗させない一方、destination sourceのDecision根拠にも利用しない。

## 6. Entry preflight

test-only source:

- `tools/reference-business-procedure-check/second_application_gate_c_actual_entry_pressure_test.go`

generic preflight:

- `tools/reference-business-procedure-check/gate_c_second_application_entry_preflight_pressure_test.go`

actual inputで、

```text
state = entry-evidence-complete
```

を要求する。

ただしこのstateから、

- Canonical Application ID
- Business Relation
- datasource alias
- Authorization requirement
- Environment binding
- UI surface

を生成しない。

## 7. S0 conclusion

S0 entry条件は成立した。

次に進めてよいのは、

```text
S1 candidate qualification
  -> S2 portability relation
```

までである。

S3 destination decision packetは、S1/S2のEvidenceからApplication identity等を自動採用せず、未決定事項を明示したまま別Decisionとして扱う。

## 8. Branch policy

以降の第二Application Referenceは、

```text
work/issue-000040-second-application-reference
```

の一つのprimary workstreamへ収束する。

exploration / portability / Pattern / Environment / Driverごとの長期branchを作らない。
