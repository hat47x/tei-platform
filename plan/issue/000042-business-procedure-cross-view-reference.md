# 000042 Business Procedure cross-view Reference

Status: In Progress

## 目的

`order_entry`を使い、Data / Function-Interaction / Businessの三視点cross-checkを実行可能なReferenceとして再現し、Business Procedureの最小Canonical contractをEvidenceから決める。

先にWorkflow Engineやfull Procedure schemaを固定しない。

正式基線:

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

## Lane境界

本IssueはレーンCのBusiness Procedure Reference / Evidenceを所有する。

RQ-025 / RQ-026はmainのProduct Requirementとして採用済みである。PAS-07 Acceptance trace正本はレーンB側で扱う。

本IssueではGate B / Gate C scopeを変更しない。

## 固定する境界

```text
Business Action != Application Function
Business Flow != Procedure != Runtime Workflow Orchestration
Business Actor / Job != Authentication Principal != Authorization Role / Policy
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

Reference fixtureは次を明示する。

```text
artifactRole = reference-authoring-fixture
canonical = false
procedure.identityStatus = reference-local-stable
```

`procedure.id`はReference corpus内でreview target / handoff endpoint / cross-referenceを安定して指すauthoring keyである。

一方、Application Patternが所有する次の意味はnormalized Worldからderivedする。

```text
Data
Relation
Query
Capability
Datasource
```

Business Procedureへコピーしない。

## Reference構成

### Primary Procedure

`reference/business-system-design/order-entry-business-procedure.json`

BP-00〜BP-09を持つ。BP-04 / BP-09はApplication Patternを持たないmanual / external Business Actionとして保持する。

### Follow-up Procedure

`reference/business-system-design/order-entry-follow-up-procedure.json`

別Business文脈から同じ`order_entry.order_record`を利用し、multiple Procedure -> same Patternを圧力試験する。

### Reference set / handoff

`reference/business-system-design/order-entry-procedure-reference-set.json`

```text
order-entry.order-handling / BP-09
  -> order-entry.follow-up / FU-00
```

Procedure keyとartifact locatorを分離する。

### Intentional defect

`reference/business-system-design/order-entry-customer-edit-conflict.json`

`order_entry.customer_reference`が利用する`master=read`に対しcandidate側だけ`read-write`を要求し、

```text
BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT
severity = architecture-policy
```

を期待する。

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

explicit referenceはexact identityでのみ解決し、名称類似から補完しない。

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

### A11 Reference-local stable key

review / handoff targetをfile pathから分離する。

### A12 file move invariance

pure file moveではProcedure key / handoff endpoint / thin Business meaningを変えない。

### A13 logical rename fail closed

Procedure key rename後のstale linkを名称類似で補修しない。

### A14 review continuity

同じProcedure keyを保つcontent changeではreview target continuityを保つ。

### A15 approval is snapshot-bound

approvalはrevision / fingerprintへ束縛し、新sourceへ暗黙carry-forwardしない。

### A16 file-native accepted source pressure

forward-compatible `procedures/` source rootをtemp Workspaceで読み、ManagedFilesへ載せながら既存`ir.World`を変えない経路を確認する。

### A17 Shared Acceptance target pressure

Preview / Reject / stale detection / Accept / reloadをReference-local key + fingerprintで圧力試験する。

### A18 identity shell pressure

ID-onlyのpath-independent targetをtest-onlyで確認する。

ただしID-onlyを最小十分contractとはみなさない。

### A19 thin Canonical Business meaning pressure

次のBusiness residual meaningをtest-onlyでnormalized modelへ投影する。

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

### A20 Pattern-owned meaning non-duplication in Canonical candidate

thin Business meaning側でもPattern-owned Data / Relation / Query / Capability / Datasourceをコピーしない。

### A21 manual action in Canonical candidate

BP-04をfake Function / Capabilityなしでthin Business meaningへ保持する。

### A22 thin handoff relation pressure

handoffをProcedure bodyのruntime control flowではなく、

```text
Procedure/Step -> Procedure/Step + condition
```

のBusiness semantic relationとしてtest-only正規化する。

### A23 handoff condition is semantic change

from/toが同じでもcondition変更をBusiness meaning changeとして観測する。

### A24 no runtime workflow creep

thin Procedure / Step / handoff probeへretry / timer / queue / saga / scheduler / runtime stateを入れないことをfield guardで固定する。

## Identity pressureの更新

前段では、review / handoffのために安定したauthoring-level keyが有用であり、Canonical Procedure identityはまだ不要と判断した。

さらにfile-native source / Acceptance / formal architectureを照合した結果、次へ更新する。

```text
Reference-local stable key
  useful / required for review + handoff

ID-only Canonical shell
  useful identity primitive candidate
  but insufficient as standalone accepted Business meaning

thin normalized Business meaning
  now the primary comparison target
```

理由は、同じProcedure IDのBusiness-only accepted changeをID-only shellではCanonical semantic diffとして観測できないためである。

## Canonical meaning pressure

formal architectureは

```text
Authoring
  -> accept
  -> Canonical Semantic meaning / IR
```

を基線とする。

三視点が同じaccepted meaningを検査するなら、Business-only accepted changeをCanonical側がIDしか保持しない状態は不十分な可能性が高い。

したがって現在の第一比較は次である。

```text
A: identity-only
vs
D: identity + thin Business residual meaning + thin handoff relation
```

production `ir.World`へはまだどちらも統合しない。

## 000039 / Shared Acceptanceとの境界

`ISSUE-000039`は共通revision-aware semantic diff / dependency propagationを将来所有する。

本Issueは共通change graphを先取り実装しない。

ただし000039がBusiness changeを見失わないために必要なmeaning粒度をReferenceで明らかにする。

Shared Acceptanceについてもproduction APIを完成させず、target resolution / stale rejection / reload invariantsだけをpressure testする。

## 非対象

- production Canonical Procedure schemaの採用確定
- Workflow Engine / retry / timer / queue / saga / scheduler
- actor -> Authorization Role自動mapping
- Business Lifecycle 000038のstate transition実装
- 000039共通change graph実装
- PAS-07 trace正本統合
- Gate B / Gate C完了条件変更

## 現在のVerification status

Repository上には次を実装済み。

- primary / follow-up Procedure fixture
- Reference set / handoff
- intentional access-conflict probe
- cross-view checker
- strict decoder
- Pattern-owned meaning derivation
- authoring residual / Business-only change tests
- review / handoff identity pressure
- file-native accepted source pressure
- Shared Acceptance target pressure
- ID-only identity shell pressure
- thin Canonical Business meaning pressure
- thin Canonical handoff relation pressure
- PR pathに連動したGitHub Actions workflow

GitHub-hosted Actionsは現時点まで`runner_id=0` / `steps=[]`でcheckout前に終了している。

Remote Desktop経由のlocal executionも試みたが、オンライン端末が接続されていなかった。

したがって新しいGo testsはRepositoryへ実装済みだが、actual `go test` greenとはまだ主張しない。

本Issueは`In Progress`、PRはDraftを維持する。

## 完了条件

1. actual Repository environmentで`gofmt / go vet / go test`がgreen
2. baseline checkerがclean
3. intentional access conflictだけをexpected diagnosticとして再現
4. Business-only changeをthin meaning差として再現
5. file move / logical rename境界を再現
6. manual workをfake Functionなしで保持
7. Pattern-owned meaningを重複しない
8. handoffをruntime Workflow化せずsemantic relationとして保持
9. identity-onlyとthin meaningのどちらがproduction Canonical contractとして最小十分かをReference Evidenceから説明できる

この完了後に初めてADR候補を作る。
