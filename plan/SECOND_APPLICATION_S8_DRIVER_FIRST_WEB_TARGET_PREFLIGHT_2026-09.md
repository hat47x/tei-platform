# Second Application Reference — S8 Driver / First Web Target Preflight

- Date: 2026-09-21
- Primary workstream: ISSUE-000040
- Stage: S8 preflight
- Entry: S7 accepted
- S7 accept commit: `0ca2016b900a3f66d96aa645bb0ea6d1b2447ad5`
- Current Workspace model revision after S7: `sha256:4c0ec01c83b1412454a4ab2ba4adf8cd8904ea7609f1326acb910ba5652f1f5c`
- Current status: **source-inspection preflight only; no Driver / Web Target code authored by this packet**
- Runtime authority: **not granted**
- Actual Oracle execution: **not claimed**

## 1. Purpose

S5〜S7で成立したSecond Application meaning / Environment bindingを、次の2境界で再現できるかを確認する。

1. Driver reference Contract Query execution
2. First Web Target projection / read-only interaction

ここで新しいBusiness / Semantic meaningをTarget側へ足さない。

```text
accepted Application
  repair_intake

accepted relation
  repair_intake.intake.customer
  repair_intake.intake.customer_id
    -> customer.customer.customer_id

accepted datasource
  repair_intake.customer_master
  -> master_data
  access read

accepted Environment binding
  development -> master_data.dev
  production  -> master_data.prod

shared Contract Query
  master_data.customer/find_by_id
```

## 2. Driver execution chain already present

current Driver path:

```text
repair_intake.customer_reference
  -> ResolveReferenceContractQuery
  -> Relation key
       repair_intake.intake.customer_id
       -> customer.customer.customer_id
  -> Query input
       customer.customer.customer_id -> id
  -> Application datasource
       customer_master -> master_data
  -> Environment binding
       repair_intake.customer_master -> master_data.<env>
  -> Contract Query executor binding
  -> instance-aware runtime registry
  -> ContractQueryExecutor
```

`PrepareReferenceContractQuery` はmapped rowのSemantic attribute identityだけを使い、列順・物理column名・型変換から値を推測しない。

Second Applicationのruntime inputは、Target-local inputから次のmapped rowを作れば既存Driverへ渡せる。

```text
column attribute = repair_intake.intake.customer_id
row value        = runtime Customer ID input
```

Relation keyによりこのsource valueが`customer.customer.customer_id`へ対応し、Contract Query input `id`へ渡る。

Targetが直接`master_data.customer/find_by_id`を呼ぶ設計にはしない。

## 3. Development Driver path

developmentには既に:

```text
environment binding:
  repair_intake.customer_master -> master_data.dev

Contract Query executor binding:
  environment development
  instance master_data.dev
  database master_data
  contract master_data.customer
  query find_by_id
  executor database.query.oracle
```

がある。

したがって、developmentでの**deterministic synthetic Driver scenario**はcurrent Canonical / Environment sourceだけで構成できる。

runtime registryにはexact key:

```text
Environment = development
Instance    = master_data.dev
Capability  = database.query.oracle
```

でsynthetic ContractQueryExecutorを登録する。

これはactual Oracle connectionではない。

Evidence labelを:

```text
synthetic-runtime-fixture
```

等で明示し、actual Oracle Evidenceへ格上げしない。

## 4. Production is not yet an equivalent execution path

production EnvironmentにはS7 binding:

```text
repair_intake.customer_master -> master_data.prod
```

は存在する。

しかしcurrent Repositoryにはproduction用の:

```text
contract-query-executor-binding
  production / master_data.prod / master_data.customer/find_by_id
```

が存在しない。

Driverはprovider名や同一Database定義からexecutorを推測しないため、production executionをdevelopment bindingから流用しない。

S8 initial Driver Evidenceはdevelopmentだけを対象にする。

production実行を必要とする時点で、production Environment-owned executor bindingを別の明示Proposal / Acceptanceとして扱う。

## 5. Runtime authority remains unresolved

S7 bindingはDatabase instance identityを解決しただけである。

現行EnvironmentのResource / Permissionは`sales_core` migration向けであり:

- database credential
- database endpoint
- migration workdir
- migration.atlas grants
- lease-policy

を`repair_intake`のruntime read authorityへ流用しない。

actual Oracleについては:

- principal
- role / grant
- credential delivery
- network connection authority
- read-only transaction policy
- actual DB handle assembly

が未観測 / 未付与である。

`internal/driver/queryruntime/oracle.Runner`も注入済み`*sql.DB`を要求するだけで、credential / endpoint / principalを推測しない。

したがってS8 initial Driver scenarioで解消できるのは:

```text
DRIVER_SCENARIO_NOT_RUN
```

のsynthetic部分だけであり:

```text
RUNTIME_AUTHORITY_NOT_GRANTED
actual Oracle external read
```

は残す。

## 6. First Web Target projection readiness

`buildWebTargetProjection` は`ApplicationID`を入力に取り、PatternをDriver Resolverで解決する。

したがって`repair_intake`についても:

```text
application
  repair_intake

pattern
  repair_intake.customer_reference
  kind reference

relation
  repair_intake.intake.customer
  repair_intake.intake -> customer.customer

datasource
  customer_master -> master_data
  access read

contract query
  master_data.customer v2.4 / find_by_id
  result customer.customer
  zero-or-one
```

をTarget-local projectionへ出せる構造である。

ProjectionはderivedでありCanonical authorityではない。

Successor Handover execution packet:

- `plan/SECOND_APPLICATION_S8_SUCCESSOR_HANDOVER_PREFLIGHT_2026-09.md`

## 6.1 Target provenance must use a successor Handover

The original First Web Target used:

```text
B / R
R = e5231839a5dcc62cce1181a2731fcb7c70d16736
```

That revision predates `repair_intake`.

Therefore a Second Application Target run must **not** emit a manifest that says:

```text
Application = repair_intake
baselineRepositoryRevision = e523...
```

because the Application does not exist at that revision.

The existing Handover contract already defines the correct rule:

```text
meaning changes / expands
  -> re-run required baseline Evidence at a new revision R2
  -> issue successor Handover B2 / R2
  -> keep Target build identity separate
```

For S8 the ordering is:

```text
S8a deterministic Driver scenario
  -> stabilize current Repository revision
  -> requalify Handover-required Evidence at R2
  -> add repair_intake Driver scenario / unresolved references
  -> issue successor Handover B2 / R2
  -> S8b First Web Target
       projection baseline = B2 / R2
       target build        = separate T2
```

Do not assign `R2` before the S8a Driver implementation/evidence is integrated, because that work may itself move the Repository revision.

Do not reuse old `B / R` merely because the same Target adapter is being reused.

The successor Handover remains Evidence/reference metadata, not a second Canonical source.

## 7. Current First Web Target mismatch

current executable serverには重要なapplication mismatchがある。

CLI:

```text
-application <Canonical Application ID>
```

はProjectionのApplicationを切り替えられる。

一方、`main.go`は常に:

```go
newReferenceReadScenarioRunner(loaded.World)
```

を渡し、このrunner内部は:

```text
environmentID = development
applicationID = order_entry
browse        = order_entry.recent_orders
record        = order_entry.order_record
reference     = order_entry.customer_reference
header-detail = order_entry.order_lines
```

に固定されている。

さらに`/scenario/read` inputは:

```text
from
row
```

で、Second Applicationのfirst target:

```text
Customer ID input -> lookup -> zero-or-one / not-found
```

と一致しない。

したがって現在:

```bash
go run ./tools/reference-first-web-target -application repair_intake ...
```

を実行すると、manifestは`repair_intake`なのに`/scenario/read`は`order_entry`を実行し得る。

これはS8 Evidenceとしてfail closedすべきTarget mismatchである。

## 8. Minimal Second Application read surface

S8で必要なのは、Target-localなread-only diagnostic surfaceだけである。

最小interaction:

```text
input:
  customerId

Target-local preparation:
  mapped row
    Semantic attribute = repair_intake.intake.customer_id
    value              = customerId

Driver:
  ExecuteReferenceContractQueryInEnvironment(
    development,
    repair_intake,
    repair_intake.customer_reference,
    mapped row,
    exact runtime registry
  )

result:
  0 row -> not-found
  1 row -> customer.customer
  >1 row -> Driver Contract cardinality error
```

Target側で:

- Relation key
- Contract Query target
- datasource target
- result cardinality

を再定義しない。

Customer ID inputのform / query parameter / HTTP route名はTarget-local interaction detailでありCanonicalへ昇格させない。

## 9. Web command surface is not inherited

`repair_intake`はcurrent accepted sourceでCommand Patternを持たない。

したがってexisting First Web Targetの:

```text
POST /scenario/submit
/custom-submit
```

をSecond Applicationへ流用しない。

Second Application server modeではcommand runnerを未設定にし、command endpointはavailableにしないか明示的にunsupportedとする。

order_entryのCommand / Validation / Authorization Scenarioをrepair_intakeへ暗黙copyしない。

## 10. Projection unresolved must become application-aware

current `buildWebTargetProjection` は全Applicationへ一律に:

```text
browse-environment-instance-binding-not-exposed-by-current-driver-api
actual-functional-evidence-not-yet-acquired
visual-evidence-not-yet-acquired
actual-external-command-observation-not-acquired
```

を付ける。

これはorder_entryのGate C Referenceには対応しているが、reference Pattern 1件だけの`repair_intake`へ:

- browse unresolved
- command observation unresolved

を付けるのは不正確である。

S8ではunresolved / authority limitsをApplication / Pattern inventoryから導くか、Second Application target runnerが自分の未解決だけを明示する。

Target固有の不足をCanonicalへ戻さない。

## 11. Required executable pressure

S8 codeへ進むとき、少なくとも次をpressureする。

### Driver

1. `repair_intake.intake.customer_id` mapped rowから`find_by_id.id`を組み立てられる。
2. development bindingが`master_data.dev`へ解決する。
3. existing development Contract Query executor bindingが`database.query.oracle`へ解決する。
4. exact runtime registry entryを使ってsynthetic customer rowを返せる。
5. registry未登録ならfail closedし、別instance / providerへfallbackしない。
6. productionはexecutor bindingが無い限りfail closedする。
7. result 0件は正常not-found、1件は正常、2件以上はzero-or-one違反として拒否する。

### Projection

1. `repair_intake` projectionがPattern 1件だけを持つ。
2. Relation source / targetが`repair_intake.intake -> customer.customer`。
3. datasourceが`customer_master -> master_data / read`。
4. Contract Queryが`master_data.customer v2.4 / find_by_id`。
5. input mapping / result cardinalityがaccepted sourceと一致する。
6. order_entry固有Pattern / command / browse meaningが混入しない。

### Server / interaction

1. manifest Applicationとread runner Applicationが必ず一致する。
2. `repair_intake` read interactionはCustomer IDを入力とする。
3. Target-local inputからSemantic mapped rowを作り、Driverへ渡す。
4. read scenario resultにactual Environment / instance / executor capabilityを明示する。
5. synthetic executor使用をEvidenceへ明示する。
6. command surfaceをrepair_intakeへ暗黙追加しない。
7. Web codeを外してもCanonical World / accepted sourceは変わらない。

## 12. Negative controls

少なくとも:

- projection = repair_intake / runner = order_entry mismatch
- missing `repair_intake.intake.customer_id`
- wrong Semantic attribute
- missing Environment binding
- missing development executor binding
- runtime registry missing exact instance
- registry only has another instance
- wrong provider runtime
- production binding without production executor binding
- duplicate / >1 Customer result
- command endpoint enabled for repair_intake without Command Pattern

をfail closedする。

## 13. S8 initial Evidence classification

S8 initial evidenceを2つへ分ける。

### E1 — deterministic Driver / Target evidence

Repositoryだけで再現可能。

```text
accepted Canonical source
accepted Environment binding
development Contract Query executor binding
synthetic runtime executor
First Web Target projection / HTTP interaction
```

これにより:

- Driver meaning
- Environment selection
- Contract Query routing
- zero-or-one behavior
- Target projection consistency

を確認する。

### E2 — actual external Oracle evidence

別途継続。

```text
actual principal / grants
credential / secret delivery
network endpoint authority
actual Oracle driver / Host assembly
actual SELECT
actual read-only Evidence
raw Oracle metadata
```

E1 PASSをE2 PASSへ読み替えない。

## 14. S8 entry / exit

Entry:

- S5 Application acceptance complete
- S6 reload / source independence complete
- S7 Environment binding acceptance complete
- current Workspace revision = `sha256:4c0ec01c83b1412454a4ab2ba4adf8cd8904ea7609f1326acb910ba5652f1f5c`

Initial exit for ISSUE-000040 requires:

- development synthetic Driver scenario PASS
- successor Handover `B2 / R2` issued from a revision that contains the accepted Second Application + S7 binding + S8a Driver Evidence
- First Web Target run references `B2 / R2`, not the historical first-Application `B / R`
- Target build identity is tracked separately as `T2`
- repair_intake Web projection PASS
- Customer ID input -> lookup Target interaction PASS
- manifest / runner Application identity一致
- no order_entry-only meaning leak
- no Canonical source changes caused by Web Target
- `DRIVER_SCENARIO_NOT_RUN` resolved for deterministic development scenario
- `TARGET_PROJECTION_NOT_EVALUATED` resolved for repair_intake First Web Target
- `RUNTIME_AUTHORITY_NOT_GRANTED` retained unless separate authority Evidence is accepted
- actual Oracle Evidence remains separately labeled unresolved if not acquired

production actual execution is not required for this initial S8 exit.

## 15. Handoff to S9

S8後、S9 change propagationへ渡すEvidence:

- shared Customer Semantic / Contract identity actually used by both Applications
- destination Pattern remains destination-owned
- origin Pattern remains provenance-only
- Environment alias / binding is destination-owned
- First Web Target projection derives from accepted repair_intake meaning
- actual external authority unresolved is not hidden

S9ではISSUE-000039のimpact analysisを再利用し、独自impact engineを作らない。


## 16. S8a branchless Driver pressure candidate — 2026-09-21

S8aのDriver pressureを、authoritative sourceを変更しないtest-only candidateとして作成した。

```text
candidate commit = decdb5c16d403e72e8ce12dcd62698d29b3ae40f
base main        = 50ce3bd6c334b0b3c6975f3a8e00981bc610f2bd
remote branch    = none
PR               = none
main connection  = none
```

変更は2 fileだけ。

```text
A tools/reference-business-procedure-check/second_application_s8_driver_pressure_test.go
M tools/run-knowledge-portability-reference-evidence.sh
```

Driver pressureは次の7点を固定する。

1. accepted `repair_intake.intake.customer_id`からdevelopmentの`master_data.customer/find_by_id`を実行できる。
2. zero-or-oneの0件を正常not-foundとして扱う。
3. 2件以上をcardinality violationとして拒否する。
4. Target側の別Semantic attributeからCustomer IDを推測しない。
5. exact runtime registry entryが無ければ別instance / providerへfallbackしない。
6. runtime Executor providerがOracle instanceと不一致なら拒否する。
7. productionはexplicit executor bindingが無い限り、development binding / providerから暗黙補完しない。

test fileはlocal `gofmt`後のGit blob:

```text
a1fc1da3562090bc22af156934798b290f5dc95f
```

とcandidate blobが一致した。

現時点のEvidence分類:

```text
gofmt/blob identity      = PASS
static API/error-path    = inspected
focused go test          = not obtained
reference evidence       = not obtained
repository health check  = not obtained
Executable Evidence      = absent
```

current execution environmentにはGo / GitはあるがRepository checkoutが無く、GitHub DNSも解決できないため、binding health checkを実行できていない。

したがってこのcandidateを`main`へ接続しない。

次のpromotion条件は:

```text
focused TestSecondApplicationS8* PASS
tools/run-knowledge-portability-reference-evidence.sh PASS
tools/repository-health-check.sh PASS
latest main drift recheck
direct-to-main integration
```

である。

このcandidateは旧 `d4fd397fe97291e6ca185c4a729de4a40e18202b` をsupersedeする。


Rebase note:

```text
source candidate = 015814af03e36859fbb4d5d896f26a9f4d4619b3
current candidate = decdb5c16d403e72e8ce12dcd62698d29b3ae40f
test blob         = a1fc1da3562090bc22af156934798b290f5dc95f (unchanged)
main drift        = docs-only / non-conflicting
```

The current candidate supersedes prior branchless S8a candidates. Health evidence remains absent, so it stays disconnected from `main`.
