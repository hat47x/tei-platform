# RQ-025 / RQ-026 PAS-07 Evidence Contract

Status: **supporting Lane B Evidence contract / PAS baseline integrated into `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md` / Reference implementation owned by Lane C ISSUE-000042**

Canonical PAS summary: `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`

## 1. 目的

採用済みの次のProduct Requirementについて、主Traceに統合済みのPAS-07をReference Evidenceへ詳細接続します。

- `RQ-025` — 業務上の行為・手順をApplication Functionから独立して設計できること
- `RQ-026` — Data / Function-Interaction / Business Procedureの異なる視点から相互検証できること

PAS-07の利用者価値、代表Scenario、Requirement mappingの正本は `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md` です。

本書はAcceptance pressure、E7 mapping、Representation / Runtime Evidence境界を詳細化するsupporting contractであり、第二のPAS source of truthではありません。

Business Procedure Reference fixture / checker / handoff / identity pressureも所有しません。実装Evidenceの供給元はLane C `plan/issue/000042-business-procedure-cross-view-reference.md`（PR #482）です。

既存Gate B / Gate C scopeは変更しません。

## 2. PAS-07 — 手作業を失わずData / Function / Businessを相互検証して設計修正できる

### 利用者価値

開発者は、業務をApplication内部の実行手順へ還元せず、人・外部作業を含むBusiness Procedureを確認しながらData / Function-Interaction / Businessの三視点から矛盾・不足・過剰な仮定を見つけ、Projectionだけを補修するのではなくauthoritative sourceへ戻って設計を修正できます。

### 代表Scenario

```text
accepted Data / Relation / Application meaningを読む
        ↓
Business固有meaningをaccepted authoring sourceとして保持する
        ↓
manual / external Actionをfake Functionへ変換しない
        ↓
existing Application / Semantic identityをexplicitに参照する
        ↓
derived cross-view Projectionで相互検証する
        ↓
contradiction / omission / unknown / impactを発見する
        ↓
correction candidateをauthoritative source側へ返す
        ↓
Shared Acceptance Boundary
        ↓
reload / regenerate / reviewする
```

### 壊してはいけない境界

```text
Business Action != Application Function
Business Flow != Procedure != Runtime Workflow Orchestration
Business Actor / Job != Authentication Principal != Authorization Role / Policy
Validation PASS != whole-business approval
Application Business Result != external-world Observation
Projection != second source of truth
```

この節は主Traceの意味を詳細Evidenceとの対応のため再掲しています。意味の変更は主Traceへ先に反映します。

## 3. Lane ownership

### Lane B — 本文書が所有するもの

```text
PAS-07 main trace
      ↓
Acceptance pressure / Evidence contract
      ↓
Reference Evidence -> PAS claim audit
      ↓
Evidence status update
```

Lane BはReference実装を第二正本として持ちません。

### Lane C — Reference / Evidence供給元

Primary implementation issue:

`plan/issue/000042-business-procedure-cross-view-reference.md`

Lane Cは少なくとも次を所有します。

```text
primary / follow-up Business Procedure authoring fixture
Procedure Reference set / handoff
intentional access-conflict probe
cross-view checker
Pattern-owned meaning derivation
strict decoder
Source Index pressure
review continuity / handoff identity pressure
approval snapshot binding pressure
file-native accepted-source pressure
actual execution Evidence
```

Lane Bはこれらを再実装せず、PAS-07 claimへ対応付けます。

## 4. Representation pressure

PAS-07はCanonical `Procedure` kindを前提にしません。

Acceptance上は次の違いを監視します。

```text
V0  accepted meaningから再生成できるderived Projection
V1  Business固有meaningを保持するfile-native / authoring source
V2  durable Canonical Procedure identity
```

このV0 / V1 / V2はLane Bの監査レンズであり、独立したReference schemaではありません。

Lane C ISSUE-000042では現時点で:

```text
Reference-local stable Procedure key
  useful / required for review + handoff

Canonical Procedure identity
  not yet required by Evidence
```

という圧力結果を扱っています。

さらにfile-native accepted-source pressureでは、Preview / Reject / stale detection / Accept / reload / file move / logical renameの機械的安全性がReference-local stable keyで成立しています。

したがって、Canonical Procedure導入の根拠をShared Acceptanceの機械的必要性へ置きません。現在残る強いidentity論点はRQ-002上のCanonical trace targetです。

Lane Cのidentity option auditでは、少なくとも次を比較しています。

```text
A  minimal Canonical Procedure identity shell
B  Subject contract expansion
C  formal Authoring identity registry
```

これはArchitecture pressureであり、PAS-07 Evidence contractからOption Aやfull Canonical Procedureを採用決定しません。

## 5. RQ-025 acceptance pressure

### P25-1 — Functionを持たないBusiness Action

Application Pattern / Capabilityを直接持たないmanual / external Business Actionを正常な状態として扱えること。

### P25-2 — Businessとruntime authorityの分離

Business actor / jobからAuthentication Principal、Authorization Role、permissionを合成しないこと。

### P25-3 — Application利用はexplicit reference

Application / Semantic meaningをBusiness側へ複製せず、既存identityをexplicitに参照できること。

### P25-4 — handoffとRuntime Workflowの分離

Business Procedure間のhandoffをApplication call relation、scheduler、retry、saga等へ自動変換しないこと。

### P25-5 — Business固有meaningをderived扱いしない

purpose / trigger / manual action / judgment / continuation等、Application Patternから再生成できない意味をProjectionへ還元しないこと。

## 6. RQ-026 acceptance pressure

### P26-1 — exact cross-view reference

Data / Relation / Pattern / Validation等を名称類似で補完せず、accepted identityで相互参照できること。

### P26-2 — Function / Data contract contradiction

candidate Interactionがread-only datasource等のaccepted contractに反するとき、Target convenienceで隠さず矛盾として検出できること。

### P26-3 — Business-only change separation

Business Action文言等のBusiness-side変更をApplication-owned meaningの偽変更へ拡大しないこと。

### P26-4 — reuse / handoff identity

複数Procedureから同一Patternを利用する場合もBusiness文脈を畳まず、handoff endpointをlogical authoring keyで追跡できること。

### P26-5 — review / acceptance provenance

同じreview targetでもrevision / fingerprintが変われば旧approvalを暗黙carry-forwardしないこと。

### P26-6 — Canonical expansionをEvidenceなしで行わない

Reference-local identityが有用でも、それだけでCanonical `Procedure` / `Step`を追加しないこと。

## 7. PAS-07 Evidence contract

Lane C ISSUE-000042のAcceptanceをPAS-07へ次のように対応付けます。

| PAS-07 Evidence | Evidence source | Claim |
| --- | --- | --- |
| E7-01 | ISSUE-000042 A2 / A8 | manual / external Business ActionとBusiness authoring residualをFunctionへ還元しない |
| E7-02 | ISSUE-000042 A1 / A3 / A4 | exact reference resolutionとPattern-owned meaningの非重複 |
| E7-03 | ISSUE-000042 A5 / A6 | read-only contract contradictionをbaselineと区別して検出 |
| E7-04 | ISSUE-000042 A9 | Business-only changeをApplication meaningの偽変更へ拡大しない |
| E7-05 | ISSUE-000042 A10 / A11 | multiple Procedure -> same PatternとReference-local stable key |
| E7-06 | ISSUE-000042 A12 / A13 | source moveでhandoff維持、logical renameはfail closed |
| E7-07 | ISSUE-000042 A14 / A15 | review target continuityとapproval snapshot binding |
| E7-08 | ISSUE-000042 A7 | unknown field / schemaを黙って解釈しないfail-closed境界 |
| E7-09 | ISSUE-000042 A16 | Reference成功からCanonical expansionを自動導出しない |
| E7-10 | Lane C file-native accepted-source pressure + PAS-01 | authoritative Business sourceをstale baseから守り、fresh acceptance後にreload / re-deriveして設計修正を成立させる |

### E7-10のEvidence nature

Lane C `BUSINESS_PROCEDURE_FILE_NATIVE_SOURCE_PRESSURE_2026-09.md` / `accepted_source_pressure_test.go` はBusiness-only source変更について次をpressureしています。

```text
same Procedure key
changed source bytes / fingerprint
old fingerprint acceptance -> stale reject
fresh fingerprint acceptance -> accepted
reload
changed Business-side Projection
same Pattern-derived Application meaning
same Canonical World
```

これはPAS-07の「authoritative sourceへ戻して修正する」に近い重要なEvidenceです。

ただし現在の`acceptProcedureSource`はexperiment helperであり、productionのShared Acceptance Boundaryそのものではありません。

したがってE7-10は:

```text
Lane C accepted-source pressure
  -> partial / structural correction Evidence

PAS-01 actual Shared Acceptance
  -> end-to-end acceptance Evidence
```

の両方を必要とします。

Lane C側だけを根拠にPAS-07全体を`accepted`へ上げません。

## 8. Secondary semantic boundaries — Representation EvidenceとRuntime Evidenceを分ける

PAS-07のBusiness viewpointには、PAS-02が実行意味を所有する境界も現れます。

これらをLane C ISSUE-000042だけで「runtimeまでverified」と読み替えません。

### S7-01 — Validation / business judgment

Lane C fixtureでは少なくとも次が別stepです。

```text
BP-04
  surrounding business contextでsubmit可否を判断
  realization = manual-unresolved

BP-05
  accepted deterministic order constraintsを確認
  realization = validation-support
  explicit Validation refs
```

したがってBusiness judgmentとdeterministic Validationを同一Actionへ畳まないRepresentation Evidenceはあります。

一方、checkerが証明する中心はValidation identityのexact resolutionであり、実際のValidation実行結果や:

```text
all Validation PASS != whole-business approval
```

というruntime / decision意味の完全検証ではありません。

Validation executionはPAS-02側とcross-checkします。

### S7-02 — actor / authority

Lane C fixtureはBusiness actorをcandidate labelとして保持し、ISSUE-000042の固定境界も:

```text
Business Actor / Job
  != Authentication Principal
  != Authorization Role / Policy
```

としています。

ただしcross-view checkerはactor labelからAuthorization semanticsを実行・検証するものではありません。

したがってISSUE-000042から得るのは「Business actorをruntime authorityへ自動昇格しないauthoring boundary」であり、Authentication / Authorization execution EvidenceはPAS-02が所有します。

### S7-03 — Business Result / Observation

Lane C fixtureは:

```text
BP-07
  request accepted application command

BP-08
  verify relevant external state when continuation depends on actual state
```

を別stepとして保持しています。

これはCommandとObservationをBusiness Procedure上で同一視しないRepresentation Evidenceです。

一方、checkerはPattern-owned Capability等をresolveしますが、Command resultの後にactual explicit readを実行してObservationを確立するDriver Evidenceまでは所有しません。

PAS-02の代表Scenario:

```text
Validation
 -> Authorization
 -> Command
 -> Business Result
 -> separate explicit read
 -> Observation
```

と組み合わせて初めてruntime意味を評価します。

### 判定規則

```text
Lane C Procedure fixture / checker
  -> Business-side representation / cross-view Evidence

PAS-02 Driver Evidence
  -> system-realizable runtime semantic Evidence
```

片方を他方の代替にしません。

S7-01〜03のruntime未検証だけを理由にPAS-07 coreを失敗扱いにはしませんが、将来「Business Procedureからruntime境界まで一貫してverified」というProduct claimを置く場合には両方を要求します。

## 9. Evidence状態の読み方

Lane C ISSUE-000042にはfixture / checker / tests等が実装されていますが、GitHub-hosted Actionsは現時点でrunner割当て前に終了しており、actual Repository runnerによるgreen verificationは未取得です。

したがってLane Bも:

```text
implementation exists
  !=
PAS-07 execution Evidence accepted
```

として扱います。

GitHub Actionsの`failure`表示だけをcode failureまたはPAS-07 failureへ読み替えません。

さらにPAS-07の現在地は:

```text
E7-01〜09
  Lane C implementation exists
  actual runner Evidence unresolved

E7-10
  accepted-source pressure exists
  actual Shared Acceptance end-to-end unresolved
```

と分けます。

この`unresolved`状態はPAS-07を主Traceへ掲載しない理由にはしません。主TraceはRequirementからEvidence状態までを追うplanning baselineであり、未検証事項を明示して保持します。

## 10. Existing PASとの関係

### PAS-01

cross-viewで変更が必要になった後のauthoritative correction / review / approvalはPAS-01 Shared Acceptance Boundaryへ接続します。

RQ-026はPAS-07 primary / PAS-01 secondaryです。

E7-10の最終成立にはPAS-01のactual Shared Acceptance Evidenceが必要です。

### PAS-02

Validation / Authorization / Business Result / Observation等のsystem-realizable意味はPAS-02が所有します。PAS-07はBusiness側へ複製せずcross-checkします。

### PAS-04

accepted Business / Function / Data meaningをTarget固有UIが壊さないことへ接続します。ただしPAS-07をGate C Required Evidenceへ追加しません。

### PAS-06

Reference / Pattern Knowledge探索と接続できますが、knowledge-first探索だけでBusiness Procedure成立とはしません。

## 11. Requirement -> PAS mapping detail

| Requirement | Primary PAS | Secondary / cross-cutting |
| --- | --- | --- |
| RQ-025 | PAS-07 | PAS-02 / PAS-04 |
| RQ-026 | PAS-07 | PAS-01 / PAS-02 / PAS-04 |

RQ-021 / RQ-022もPAS-07にcross-cuttingします。

このmappingの正本は主Traceです。本表はE7との対応確認用です。

## 12. Gate / Stage

PAS-07を理由に現行Gateを変更しません。

```text
Gate A / B / C mainline
  existing definition unchanged

PAS-07 Reference Evidence
  Lane Cから供給
  Gate B Required Evidenceへ追加しない
  Gate C First Web Target Required Evidenceへ追加しない
```

将来「業務システム設計として成熟した」というProduct claimを置く場合には、PAS-07の実行Evidenceとrealistic multi-subsystem Referenceを別途要求し得ますが、新Gateは現時点で作りません。

## 13. Main trace integration / update rule

PAS-07は `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md` へ統合済みです。

統合条件をactual execution Evidenceのgreenにはしませんでした。主Traceはverified-only台帳ではなく、Requirement → PAS → Evidence → Gate / claimの計画・追跡基線だからです。

今後Lane Bは次を行います。

1. Lane C ISSUE-000042 A1〜A16についてactual execution Evidenceを確認する
2. A1〜A16をE7-01〜09へ再監査する
3. E7-10についてaccepted-source pressureとPAS-01 actual Shared Acceptance Evidenceを分けて確認する
4. S7-01〜03をPAS-02 Evidenceとcross-checkし、Representation / Runtimeを混同しない
5. unmet / partial / accepted / unresolvedを本書へ詳細反映する
6. PAS-07の利用者価値・主要Scenario・Requirement mapping・Gate影響に変更が必要な場合は主Traceへ先に反映する
7. 本書を第二のTrace source of truthとして並行発展させない

Reference fixture / checkerをLane B側へコピーしません。
