# Lane B — RQ-025 / RQ-026 Trace & Source-Boundary Audit — 2026-09

Status: **current Lane B audit / PAS-07 integrated into main trace / Reference implementation ownership corrected to Lane C ISSUE-000042**

## 1. 目的

RQ-025 / RQ-026採用後のRequirement → PAS → Reference → Evidenceの接続と、Business Procedureを扱うときのauthoritative source / Canonical meaning / Projection境界を監査します。

この監査はRQ-025 / RQ-026を再審議せず、Lane A / Cの実装責務を複製しません。

## 2. 確認済みの正式基線

mainでは次が成立しています。

- `TEI_PRODUCT_REQUIREMENTS.md` にRQ-025 / RQ-026が採用済み
- `docs/business-system-design.md` でBusiness Action / Application Function分離と三視点相互検証がArchitecture化済み
- `docs/architecture.md` でAuthoring DSL / Canonical IR / Runtime、actor / authority等の境界が明文化済み
- `UPSTREAM_XTEA_DESIGN_FALLBACK_PROTOCOL.md` と `XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md` にX-TEAへ立ち返る研究規則・Evidenceが保存済み
- `ORDER_ENTRY_BUSINESS_PROCEDURE_REFERENCE_EXPERIMENT.md` にReference experimentの問いが保存済み

本Lane B branchではさらに `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md` へPAS-07を統合済みです。

現在のReference / Evidence実装はLane C `plan/issue/000042-business-procedure-cross-view-reference.md`（PR #482）が所有します。

Gate B / Gate C scopeは変更されていません。

## 3. B-F02 — Business ProcedureのsourceとProjectionが混ざる危険

### 判定

Confirmed design ambiguity; Lane C Referenceで具体化中。

### 問題

Procedureが含み得る次の意味は、Data / Application Pattern / Capabilityから機械的には導出できません。

```text
purpose
trigger / event
actor / job context
manual action
external work
business judgment
handoff
business completion / continuation
```

一方、accepted identityからderivedできる意味もあります。

```text
Pattern-owned Data / Relation / Query / Capability
cross-reference
matrix
cross-view diagnostic
```

したがって:

```text
Business固有のauthoritative authoring meaning
        !=
derived Projection
```

を固定する必要があります。

### V0 / V1 / V2監査レンズ

Lane Bでは次をschemaではなく監査上の区別として使います。

```text
V0  pure derived Projection
V1  Business固有meaningを保持するaccepted authoring source
V2  durable Canonical Procedure identity
```

Lane C ISSUE-000042ではReference-local stable Procedure keyがreview / handoffに有用である一方、Canonical Procedure identityはまだEvidence上必須ではない、という圧力結果を扱っています。

Lane Bは別のV1 fixture / checkerを作らず、このEvidenceをPAS-07へ接続します。

## 4. B-F03 — RQ-025 / RQ-026のPAS trace不足

### 判定

Confirmed trace gap; corrected in this branch.

既存 `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md` はRQ-001〜024 / PAS-01〜06までの基線であり、RQ-025 / RQ-026のPrimary PASがありませんでした。

既存PASへ無理に吸収すると、Business ProcedureのProduct ValueがPAS-02のApplication Driver sequenceへ再び還元される危険があります。

### 対応

PAS-07を定義し、現在は主Traceへ統合済みです。

```text
PAS-07
手作業を失わずData / Function / Businessを相互検証して設計修正できる
```

Primary:

```text
RQ-025 -> PAS-07
RQ-026 -> PAS-07
```

Secondary:

```text
RQ-025 -> PAS-02 / PAS-04
RQ-026 -> PAS-01 / PAS-02 / PAS-04
```

`RQ025_RQ026_ACCEPTANCE_TRACE.md` はPASの第二正本ではなく、E7 mapping等を保持するsupporting Evidence contractへ縮退しました。

PAS-07は新Gateではありません。

## 5. B-F04 — 重複Requirement解釈文書の不整合

### 判定

Confirmed documentation defect; correction in this branch.

`requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md` がRQ-025 / RQ-026採用前の文言を保持していたため、Requirement第二正本ではなく解釈索引へ縮退します。

正本は一貫して `TEI_PRODUCT_REQUIREMENTS.md` とします。

## 6. B-F05 — Lane B / Lane C Reference ownershipの重複

### 判定

Confirmed system-of-artifacts defect; corrected in this branch.

### 発見

Lane B #475はPAS-07 traceを具体化する過程で、一時的に次まで実装範囲を拡張していました。

```text
Business Procedure fixtures
generator / checker
derived projections
Business Flow fixtures
identity / handoff pressure tests
専用workflow
```

その後、並行Lane C PR #482 / ISSUE-000042を横断監査したところ、Lane C側が明示的に:

```text
Reference / Evidence implementation = Lane C ownership
PAS-07 acceptance trace integration = Lane B ownership
```

を固定し、より成熟したprimary / follow-up Procedure、handoff、Source Index pressure、review continuity、approval snapshot bindingを所有していることを確認しました。

### 問題

Lane B側にも別fixture / checkerを残すと、同じRQ-025 / RQ-026に対して:

```text
Lane B Reference source
Lane C Reference source
```

が並行し、RQ-003が避ける暗黙の複数正本をsystem-of-artifactsレベルで再導入します。

これは実装内容の優劣ではなくownership defectです。

### 修正

Lane B #475から重複Reference実装を除去しました。

Lane B ownership:

```text
Requirement interpretation audit
RQ-025 / RQ-026 -> PAS-07 mapping
PAS-07 Acceptance pressure / Evidence contract
Reference / Evidence -> PAS claimのcross-lane監査
main trace / Evidence status maintenance
```

Lane C ISSUE-000042 ownership:

```text
Business Procedure authoring fixtures
Procedure Reference set / handoff
cross-view checker
intentional conflict probe
identity / Source Index pressure
review / approval pressure
file-native accepted-source pressure
actual execution Evidence
```

## 7. PAS-07 Evidenceのcross-lane contract

Lane BはLane C ISSUE-000042のA1〜A16を再実装せず、PAS-07 E7へ対応付けます。

主要対応:

```text
A2 / A8
  -> manual / external ActionとBusiness residual

A1 / A3 / A4
  -> exact reference / non-duplication

A5 / A6
  -> cross-view contract contradiction

A9
  -> Business-only change separation

A10〜A13
  -> reuse / authoring key / handoff / rename impact

A14 / A15
  -> review continuity / approval snapshot binding

A16
  -> no Canonical expansion by assumption
```

詳細mappingは `RQ025_RQ026_ACCEPTANCE_TRACE.md` をsupporting Evidence contractとして参照します。

## 8. B-F06 — Business-side Representation EvidenceとPAS-02 Runtime Evidenceの混同

### 判定

Confirmed evidence-nature distinction; not a Lane C defect.

### 発見

Lane C fixtureには次のBusiness-side boundaryが明示されています。

```text
BP-04 manual-unresolved business judgment
BP-05 validation-support
BP-06 security-boundary
BP-07 application command
BP-08 observation-boundary
```

これによりBusiness Procedure上で:

```text
judgment != Validation
Business actor / work context != runtime authority
Command != Observation
```

を同一stepへ畳まないRepresentation pressureがあります。

一方、Lane C checkerの中心はApplication Pattern / Data / Relation / Validation / Capability等のexact resolutionとcross-view diagnosticです。

次のruntime意味までLane C checkerが実行・証明するわけではありません。

```text
Validation evaluation result
Authentication / Authorization decision
Capability execution
Business Result
separate explicit read
actual Observation
```

これらは既存PAS-02のDriver scenarioが所有します。

### 修正したEvidence契約

```text
Lane C ISSUE-000042
  -> Business-side Representation / cross-view Evidence

PAS-02
  -> system-realizable Runtime semantic Evidence
```

片方をもう片方の代替にしません。

特に:

- `all Validation PASS != whole-business approval` はBusiness側の分離とruntime Validation evidenceを組み合わせて読む
- actor candidateが存在することをAuthorization evidenceへ昇格しない
- BP-07 / BP-08が別stepであることをactual Observation execution evidenceへ昇格しない

### Gateへの影響

この区別を理由にGate B Required Evidenceを追加しません。

PAS-02側は既存Gate B 3 laneのままです。

## 9. B-F07 — PAS-07「設計修正できる」のend-to-end Acceptance gap

### 判定

Confirmed acceptance coverage gap; Lane C Reference failureではなくcross-PAS依存。

### 発見

PAS-07の利用者価値は矛盾発見だけでなく:

```text
cross-view contradiction
  -> correction candidate
  -> authoritative source correction
  -> Shared Acceptance
  -> reload / regenerate
  -> intended resolutionを確認
```

まで含みます。

Lane C file-native source pressureはBusiness-only source変更について:

```text
same Procedure key
changed source fingerprint
old fingerprint -> stale reject
fresh fingerprint -> accepted
reload
Business-side Projection changed
Pattern-derived meaning unchanged
Canonical World unchanged
```

をexperiment helperでpressureしています。

これはE7-10の重要なpartial Evidenceです。

しかし`acceptProcedureSource`はproduction Shared Acceptance Boundaryではなく、Lane C文書自身もShared Acceptanceを最小再現する次pressureとして残しています。

### 結論

Lane C ISSUE-000042 A1〜A16がgreenになっても、それだけでPAS-07全体を`accepted`へ上げません。

```text
E7-01〜09
  -> Lane C Reference / Evidence

E7-10 correction / acceptance / reload
  -> Lane C accepted-source pressure
     + PAS-01 actual Shared Acceptance Evidence
```

が必要です。

PAS-07はRQ-025 / RQ-026のPrimary PASですが、end-to-end correctionはPAS-01との意図的なcross-PAS依存です。

## 10. B-F08 — actual greenをmain trace統合の前提にした過剰待機

### 判定

Confirmed trace-governance defect; corrected in this branch.

### 発見

旧Integration ruleは、Lane C ISSUE-000042のactual execution Evidenceを確認してからPAS-07を `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md` へ統合する、としていました。

しかし主Trace自身はverified-only台帳ではありません。

```text
Requirement
  -> PAS
  -> Design / Reference
  -> Evidence
  -> Gate / Product claim
```

を、`unresolved`を含めて追跡するplanning baselineです。

PAS-05 / PAS-06等もEvidenceが未完了のまま主Traceに存在します。

したがって:

```text
PAS semantics / ownership / Evidence contractが安定
  -> main traceへ統合可能

actual execution Evidence green
  -> Evidence statusを更新
```

と分けるのが整合的です。

### 問題

actual greenまで統合を待つと、RQ-025 / RQ-026のPrimary PASだけがsupporting extensionに残り、長期間:

```text
main trace
PAS-07 extension
```

という二つのTrace候補を並行維持することになります。

これはRQ-003 / RQ-023のsystem-of-artifacts原則にも不利です。

### 修正

- PAS-07を主Traceへ統合した
- current Evidence statusを`implementation exists / actual runner unresolved`として明記した
- `RQ025_RQ026_ACCEPTANCE_TRACE.md` を詳細Evidence contractへ縮退した
- actual runner greenは主Traceへの掲載条件ではなくEvidence status更新条件とした
- greenだけでPAS-07全体をacceptedへ上げない規則は維持した

## 11. B-F09 — RQ-002 trace targetとPAS-07 completionを混同する危険

### 判定

Confirmed unresolved architecture pressure; PAS-07 trace blockerではない。

### 発見

Lane Cのfile-native accepted-source pressureでは、Reference-local stable Procedure keyだけで次が成立しています。

```text
Preview / Reject
stale detection
Accept / reload
file move / logical rename separation
review / handoff target continuity
```

したがってShared Acceptanceの機械的安全性を理由にfull Canonical Procedureを要求する根拠は弱くなりました。

一方、RQ-002はauthoring sourceからCanonical subjectへのtraceabilityを要求し、現行Subject contractもTEI World内のsemantic / logical targetを前提としています。

残る論点は:

```text
accepted Business Procedure source
  -> ? Canonical trace target
```

です。

Lane Cでは少なくとも:

```text
A  minimal Canonical Procedure identity shell
B  Subject contract expansion
C  formal Authoring identity registry
```

を比較し、Aをexperiment-only第一候補としていますが、採用決定ではありません。

### Lane B判定

これはPAS-07を主Traceへ載せるかどうかの問題ではありません。

PAS-07では`unresolved`として保持し、Architecture側が具体Evidenceでidentity contractを決めた後にRQ-002 / Source Index / PAS-01 / PAS-07間のtraceを再監査します。

full Canonical Procedure modelへ飛躍しません。

## 12. X-TEAへ立ち返るときの監査原則

判断に迷った場合は次の順へ戻ります。

1. 渡辺幸三氏の三要素分析法 / 業務システム設計の言説
2. X-TEA Modeler / Driver manual・source
3. CONCEPTWARE / Reference Model
4. TEI固有Requirement / Evidence

X-TEAから受け取る中心はschemaの一対一コピーではなく:

```text
Data
Function-Interaction
Business Procedure / Flow
```

を異なる視点として保ち、cross-referenceで相互牽制する設計理由です。

特にX-TEAでBusiness definitionとBusiness Flowが分かれていることは、Business handoffをApplication call sequenceやRuntime Workflowへ還元しない監査根拠になります。

## 13. Evidence questions

Lane BがLane C Evidenceを受け取る際は、少なくとも次を確認します。

### Q1 — residual meaning

Business固有meaningをPattern-derived Projectionへ還元していないか。

### Q2 — explicit reference

Application / Semantic meaningを二重記述せずexact identityで参照しているか。

### Q3 — manual / external action

FunctionなしBusiness Actionをfake Capabilityへ変換していないか。

### Q4 — cross-view contradiction

Data / Function / Businessの矛盾がProjection convenienceで隠されていないか。

### Q5 — identity / handoff

Reference-local authoring keyの価値をCanonical identity採用理由へ飛躍させていないか。

### Q6 — review provenance

同じreview targetでもrevision / fingerprint変更後に旧approvalを暗黙継承していないか。

### Q7 — execution Evidence

fixture / checker / test sourceの存在とactual runnerによるgreen verificationを混同していないか。

### Q8 — correction completeness

矛盾を発見できることと、Shared Acceptanceを通してauthoritative sourceを修正・reloadできることを混同していないか。

### Q9 — runtime boundary nature

Business-side representationとPAS-02 runtime execution Evidenceを混同していないか。

### Q10 — V2 trigger

file-native / authoring-level identityでは保てない具体的意味損失が観測されたか。

観測されない限りCanonical `Procedure` / `Step`を追加しません。

### Q11 — trace integration nature

PASの主Trace掲載条件とPAS execution Evidence accepted条件を混同していないか。

### Q12 — RQ-002 trace target

Reference-local keyで実務上のidentityが足りることと、RQ-002のCanonical trace targetが解決済みであることを混同していないか。

## 14. ADR判定

現時点ではLane Bから追加ADRを作りません。

次のいずれかが実Evidenceで必要になった場合にArchitecture側へ判断を返します。

- durable Canonical Procedure identity
- portable Business Flow contract
- actor / event / decision / handoffの固定structured contract
- universal cross-view diagnostic severity
- durable `Subsystem` / `SubjectArea`

RQ-002 trace targetについてLane CのOption A pressureが小さいidentity-only contractを必要と実証した場合も、Lane BはPAS-07から自動採用せずArchitecture decisionへ返します。

## 15. Gate判定

Gate A / B / Cは変更しません。

PAS-07 Reference EvidenceはLane Cから供給されるparallel evidence workです。

Gate B Required Evidence 3 laneやGate C First Web Target Evidenceへ追加しません。

PAS-07の主Trace統合もGate追加・Gate scope変更を意味しません。

## 16. 次作業

1. Lane C ISSUE-000042のactual runner Evidence取得後、A1〜A16のaccepted / partial / unresolvedを確認する
2. `RQ025_RQ026_ACCEPTANCE_TRACE.md` のE7-01〜09へEvidence statusを正確に反映する
3. E7-10についてLane C accepted-source pressureとPAS-01 actual Shared Acceptance Evidenceを分離して監査する
4. S7-01〜03についてPAS-02 Runtime Evidenceとcross-checkする
5. Lane CのRQ-002 identity option pressureを受け取り、PAS-01 / PAS-07 / Source Indexへの影響だけをLane Bで再監査する
6. PAS-07の利用者価値・主要Scenario・Requirement mapping・Gate影響が変わる場合だけ主Traceを更新する
7. supporting Evidence contractを第二Trace正本として発展させない

Reference fixture / checkerをLane Bへ再導入しません。
