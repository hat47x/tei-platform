# 000042 Business Procedure cross-view Reference

Status: In Progress

## 目的

`order_entry`を起点に、Data / Function-Interaction / Businessの三視点cross-checkを実行可能なReferenceとして再現し、Business Procedureの最小production Canonical contractをEvidenceから決める。

先にWorkflow Engine、full Procedure schema、global Step identityを固定しない。

## 正式基線

- `requirements/TEI_PRODUCT_REQUIREMENTS.md` — RQ-025 / RQ-026
- `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`
- `docs/architecture.md`
- `docs/business-system-design.md`
- `plan/BUSINESS_PROCEDURE_IDENTITY_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_REVIEW_HANDOFF_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_FILE_NATIVE_SOURCE_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_ACCEPTANCE_TARGET_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_IDENTITY_OPTION_AUDIT_2026-09.md`
- `plan/BUSINESS_PROCEDURE_CANONICAL_MEANING_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_HANDOFF_CANONICAL_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_MULTI_APPLICATION_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_REALISTIC_MULTI_APPLICATION_SCENARIO_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_UNCERTAINTY_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_REALIZATION_SUBTRACTION_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_STEP_IDENTITY_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_STEP_ADDRESS_OPTION_AUDIT_2026-09.md`
- `plan/BUSINESS_PROCEDURE_STEP_ADDRESS_LAYER_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_APPLICATION_CONTEXT_PRESSURE_2026-09.md`
- `plan/BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`

## Lane境界

本IssueはレーンCのBusiness Procedure Reference / Evidenceを所有する。

RQ-025 / RQ-026はmainのProduct Requirementとして採用済み。PAS-07 Acceptance trace正本はレーンB側で扱う。

Gate B / Gate C scope、Business Lifecycle 000038、共通change graph 000039を本Issueから拡張しない。

## 固定する境界

```text
Business Action != Application Function
Business Flow != Procedure != Runtime Workflow Orchestration
Business Actor / Job != Authentication Principal != Authorization Role / Policy
Business Procedure != single Application child
Data / Function-Interaction / Business = 同じaccepted meaningを相互牽制する視点
```

Business ProcedureにはApplication / Semantic Worldから機械的に導出できないresidual meaningがある。

```text
purpose
trigger / event
actor / job context
manual / external / unresolved Business Action
judgment / continuation
handoff
```

したがってBusiness Procedure source全体をderived artifactとは扱わない。

## Reference authoring / derived boundary

current Reference fixture:

```text
artifactRole = reference-authoring-fixture
canonical = false
procedure.identityStatus = reference-local-stable
```

`procedure.id`はReference corpus内でreview target / handoff endpoint / cross-referenceを安定して指すauthoring keyである。

Pattern-owned meaningはnormalized `ir.World`からderiveし、Business Procedureへコピーしない。

```text
Data
Relation
Query
Capability
Datasource
Application ownership / datasource policy
```

## Reference構成

### Primary Procedure

`reference/business-system-design/order-entry-business-procedure.json`

BP-00〜BP-09。BP-04 / BP-09はApplication Patternを持たないmanual / external Business Actionとして保持する。

### Follow-up Procedure

`reference/business-system-design/order-entry-follow-up-procedure.json`

別Business文脈から同じ`order_entry.order_record`を利用し、multiple Procedure -> same PatternをPressureする。

### Reference set / handoff

`reference/business-system-design/order-entry-procedure-reference-set.json`

```text
order-entry.order-handling / BP-09
  -> order-entry.follow-up / FU-00
```

Procedure keyとartifact locatorを分離する。

### Intentional defect

`reference/business-system-design/order-entry-customer-edit-conflict.json`

`order_entry.customer_reference`が利用する`master=read`に対しcandidate側だけ`read-write`を要求する。

期待diagnostic:

```text
BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT
severity = architecture-policy
```

### Intermediate realistic multi-Application Pressure

temp Workspaceへsynthetic `fulfillment_console`を追加し、同一Procedure内で次を共存させる。

```text
order_entry interaction
  -> fulfillment_console interaction
  -> manual warehouse coordination
  -> external delivery continuation
```

これはproduction業務仕様ではなく、Business Procedureがsocial-system / business-manual viewpointを保てるかを見る中間Pressureである。

## Cross-view checker

`tools/reference-business-procedure-check`は独自に`.tei`をparseせず、既存`workspace.Load()`のnormalized `ir.World`を利用する。

```text
Business authoring source
        +
normalized ir.World
        ↓
experiment checker
        ↓
Matrix / diagnostics / semantic pressure
```

explicit referenceはexact identityでのみ解決し、名称類似やID prefixからauthorityを推測しない。

Application Patternを使うpolicy traceでは、Patternとowner Applicationをnormalized Worldから同時にexact解決する。

## Acceptance / Evidence pressure

### A1 exact reference resolution
Application Pattern / direct semantic referenceをnormalized Worldでexact解決する。

### A2 no fake Function
manual / external Business ActionはApplication Pattern / Capabilityなしでも正常状態とする。

### A3 Pattern-owned meaning is derived
Data / Relation / Query / Capability / DatasourceをBusiness Procedureへ二重記述しない。

### A4 Procedure-side explicit refs
Patternから導けないValidation等だけをexplicit refとして保持する。

### A5 intentional access conflict
read-only datasourceへのcandidate write要求をarchitecture-policy diagnosticとして検出する。

### A6 baseline clean
accepted baselineでは同diagnosticを出さない。

### A7 fail-closed decode
未知field / unsupported schemaVersion / stale targetを黙って受理しない。

### A8 Business authoring residual
purpose / trigger / actor / manual actionをPattern-derived projectionから再生成可能とみなさない。

### A9 Business-only change separation
Business wording changeはBusiness-side diffとして現れ、Pattern-owned Application meaningを偽って変えない。

### A10 multiple Procedure -> same Pattern
同じPatternを異なるBusiness文脈から利用できる。

### A11 Reference-local stable Procedure key
review / handoff targetをfile pathから分離する。

### A12 file move invariance
pure file moveではProcedure key / handoff endpoint / thin Business meaningを変えない。

### A13 logical Procedure rename fail closed
stale linkを名称類似で補修しない。

### A14 review continuity
同じProcedure keyを保つcontent changeではreview target continuityを保つ。

### A15 approval is snapshot-bound
approvalはrevision / fingerprintへ束縛し、新sourceへ暗黙carry-forwardしない。

### A16 file-native accepted source pressure
forward-compatible `procedures/` source rootをtemp Workspaceで読み、ManagedFilesへ載せる経路をPressureする。

### A17 Shared Acceptance target pressure
Preview / Reject / stale detection / Accept / reloadをReference-local key + fingerprintでPressureする。

### A18 identity shell pressure
ID-onlyのpath-independent targetを確認する。ただしID-onlyを最小十分contractとはみなさない。

### A19 thin Canonical Business meaning pressure

```text
Procedure
  id
  purpose
  trigger
  actor
  ordered steps

Step
  id
  businessAction
  realization
  explicit semantic refs
```

Business-only accepted changeがidentity不変でもmeaning changeになることを確認する。

### A20 Pattern-owned meaning non-duplication
thin Business meaning側でもPattern-owned Data / Relation / Query / Capability / Datasourceをコピーしない。

### A21 manual action in thin candidate
BP-04をfake Function / Capabilityなしで保持する。

### A22 thin handoff relation pressure

```text
Procedure/Step -> Procedure/Step + condition
```

handoffをruntime control flowではなくBusiness semantic relationとしてPressureする。

### A23 handoff condition is semantic change
from/toが同じでもcondition変更をBusiness meaning changeとして観測する。

### A24 no runtime workflow creep
retry / timer / queue / saga / scheduler / runtime stateをthin Procedure / handoffへ入れない。

### A25 one Procedure can cross Applications
qualified Application Pattern identityを使って2つ以上のApplicationを参照できる。mandatoryな単一Application owner fieldを置かない。

### A26 multi-Application pressure is realization-independent
第二Applicationは必要最小限とし、無関係なGate B Provider readinessへ依存させない。

### A27 preserve epistemic status
trigger / actorの`status`をlosslessに保持し、`candidate`をaccepted factへ丸めない。

### A28 uncertainty change is Business semantic change
`candidate -> unresolved`等をProcedure identityやPattern refsが同一でもBusiness meaning changeとして観測可能にする。

### A29 actor uncertainty != Authorization change
actor label / uncertainty変更をAuthorization Role / Principal / Permission / Credential / Authority policy変更へ暗黙変換しない。

### A30 do not freeze uncertainty vocabulary prematurely
current Referenceだけを根拠にstatus vocabularyやmissing / unknown severityをproduction Coreへ固定しない。

### A31 realization subtraction pressure
thin candidateから`step.realization`を除いたprojectionを比較する。

### A32 realization-only label change need not be Canonical change
Business Action / refsが同一なら`realization` review labelだけの変更をCore meaning変更へ自動昇格しない。

### A33 manual action survives realization subtraction
BP-04は`realization` fieldなしでもBusiness Actionとして残り、fake Pattern / Capabilityを要求しない。

### A34 Step body meaning can be ID-free
Procedure body projectionからStep IDを除いてもordered Business Actionとexplicit refsを保持できるか確認する。

### A35 handoff still requires exact Step addressability
body meaningがStep IDなしで成立しても、handoff relationのstale Step keyはfail closedする。

### A36 Step addressability != runtime workflow identity
stable endpointが必要でもRuntime Execution / Instance / Token / State identityを導入しない。

### A37 realistic multi-Application / manual / external coexistence
一つのProcedureでApplication A、Application B、manual work、external / physical workを共存させてもthin candidateを崩さない。

### A38 Application association is resolved, not guessed
Procedureが関係するApplication集合はqualified Pattern refsをnormalized Worldで解決して導出する。Pattern ID文字列の`.`区切り等からownerを推測しない。

### A39 access policy follows Pattern owner Application
cross-Application access probeはfixture-level primary Applicationではなく、Patternを所有するApplicationのdatasource requirementを使用する。

owner Applicationのaccessを`read-write -> read`へ変えたとき、同じwrite probeがexpected conflictへ変わることをPressureする。

### A40 fixture Application context != Procedure owner
fixture-level `application`変更でthin Canonical meaningを変えない。reporting contextが存在してもmandatory Procedure ownerへ昇格しない。

### A41 Step address layer respects Source Index boundary
Canonical handoffを採用候補に残す場合、handoff endpoint解決をSource Indexへ依存させない。

normalized Procedure側にProcedure-scoped local keyをaddress metadataとして置くS4-Aを優先Pressureする。

### A42 Step local key rename separates body from address change
local key renameだけでBusiness Action body changeを捏造しない。一方、handoff / reviewのreferential address changeとしては観測する。

### A43 no global Canonical Step ID by default
handoff / review addressabilityが必要なことだけを根拠にglobal UUID / runtime Step identityを導入しない。

### A44 primary Application context optionality — open pressure
explicit Pattern refsだけでbaseline cross-view evaluationが成立するなら、fixture-level `application`をblocking semantic requirementにしない方向を比較する。

current `evaluate()`にはまだApplication存在必須checkが残るため、これは未解決Pressureである。

## 現在のCanonical候補比較

```text
A  = identity-only
D1 = identity + thin Business residual meaning + handoff
D2 = D1 - realization review label
D3 = D2 + Step body identity / Step addressability separation
```

D3が現在の主比較対象。

production `ir.World`へはいずれもまだ統合しない。

## Step addressの現在判断

```text
S4-A
  normalized Procedure
    ordered Step
      procedure-scoped local address metadata
      body meaning

S4-B
  Canonical bodyはID-free
  Step local keyはSource Indexだけ
```

Canonical handoffをsemantic relationとして採用するなら、Semantic validationをSource Index依存にしないためS4-Aが有力。

handoffをauthoring-onlyに留める結論ならS4-Bへ戻れる。

Step local keyはglobal identityでもruntime identityでもない。

## Application contextの現在判断

```text
single Application owner
  -> production Business Procedureへ入れない

Pattern owner Application
  -> realization / policy traceのauthority

fixture Application
  -> Reference reporting / primary context候補
  -> semantic ownerではない
```

current `evaluate()`に残るfixture Application存在必須checkは、次のsubtraction pressureで再評価する。

## 000039 / Shared Acceptanceとの境界

`ISSUE-000039`は共通revision-aware semantic diff / dependency propagationを将来所有する。

本Issueは共通change graphを先取り実装せず、Business changeを見失わないmeaning粒度とaddress境界だけをReferenceで明らかにする。

Shared Acceptanceもproduction APIを完成させず、target resolution / stale rejection / reload invariantsをPressureする。

## 非対象

- production Canonical Procedure / Handoff schemaの採用確定
- Workflow Engine / retry / timer / queue / saga / scheduler
- actor -> Authorization Role自動mapping
- `realization` production enum
- global / durable Canonical Step ID
- generic Uncertainty framework / certainty lattice / approval state machine
- Business Lifecycle 000038のstate transition実装
- 000039共通change graph実装
- PAS-07 trace正本統合
- Gate B / Gate C完了条件変更

## 現在のVerification status

Repository上には次を実装済み。

- primary / follow-up Procedure fixture
- Reference set / handoff
- intentional access-conflict probe
- cross-view checker / strict decoder
- Pattern-owned meaning derivation
- authoring residual / Business-only change pressure
- review / handoff identity pressure
- file-native accepted source pressure
- Shared Acceptance target pressure
- ID-only identity shell pressure
- thin Canonical Business meaning / handoff relation pressure
- multi-Application structural pressure
- realistic multi-Application + manual + external intermediate pressure
- uncertainty preservation pressure
- realization subtraction pressure
- Step identity / address option / address layer pressure
- cross-Application Pattern-owner access policy resolution
- fixture Application context subtraction pressure
- pre-ADR Canonical contract audit

一時的なGitHub Actions workflowは、runnerが`runner_id=0` / `steps=[]`でcheckout前に終了し続け、merge対象として価値を持たないためbranchから除去した。

Remote Desktop経由のlocal executionもオンライン端末不在で実行できなかった。

したがってGo testsはRepositoryへ実装済みだが、actual `go test` greenとはまだ主張しない。

本Issueは`In Progress`、PR #482はDraftを維持する。

## 完了条件

1. actual Repository environmentで`gofmt / go vet / go test`がgreen
2. baseline checkerがclean
3. intentional access conflictだけをexpected diagnosticとして再現
4. Business-only wording / uncertainty changeをthin meaning差として再現
5. pure file move / logical rename / local Step address renameを分類して再現
6. manual / external workをfake Functionなしで保持
7. Pattern-owned meaningを重複しない
8. handoffをruntime Workflow化せずsemantic relationとして保持できる
9. one Procedure -> multiple Applicationsを単一owner fieldなしで保持
10. Pattern owner Applicationからcross-view policyをexactにtraceできる
11. trigger / actor uncertaintyをlosslessに保持しAuthorizationへ混ぜない
12. `realization`をsubtractしても必要Business meaningを保持できる
13. Step body meaningとhandoff / review addressabilityを分離できる
14. global Step IDなしで必要なaddressabilityを説明できる
15. S4-A / S4-Bの選択をhandoffのCanonical必要性から説明できる
16. fixture-level primary Application contextをsemantic ownerから分離し、optional化 / 除去の扱いを決める
17. current `order_entry`より一段現実的なmulti-Application / manual / external Pressureをactual Evidenceで確認する
18. A / D1 / D2 / D3のどれが最小十分かReference Evidenceから説明できる

この完了後に初めてproduction Canonical contract ADR候補を作る。
