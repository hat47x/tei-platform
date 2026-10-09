# Lane B — A80〜A83 Business Action × Data Projection Audit — 2026-09

Status: **Lane B / supporting Evidence-status audit / no new Requirement / PAS / E7 / Gate / production schema**

Related:

- `plan/RQ025_RQ026_ACCEPTANCE_TRACE.md`
- `docs/audits/LANE_B_A73_A79_PHYSICAL_OBSERVATION_HANDOFF_AUTHORITY_AUDIT_2026-09.md`
- Lane C PR #482 / ISSUE-000042
- `plan/BUSINESS_ACTION_DATA_PROJECTION_PRESSURE_2026-09.md`

## 1. 目的

Lane C ISSUE-000042のA80〜A83を、RQ-026 / PAS-07 / V0-V1-V2 / B-F27 / B-F19bの境界から再監査する。

中心は、Business ActionとData/Relationのcross-view reviewに必要な情報のかなりの部分を、Business sourceへData dependencyを重複authorせず既存accepted refsからV0 Projectionとして導けるかである。

さらにcurrent pressureは、projectionの**変更伝播方向**まで確認している。

## 2. A80 — exact accepted refsからData / Relation traceをderiveする

Lane CはBusiness Stepの:

```text
Application Pattern
Validation
explicit Capability
explicit Data / Relation（sourceに実在する場合のみ）
```

から、normalized World上のexact identityを辿ってData / Relation traceを構成するtest-only projectionを追加した。

使わないもの:

```text
Business Action wording
Step adjacency
Pattern ID prefix
datasource名
Database ownership
Query内部実装推測
O1 test helper target
```

### Lane B mapping

```text
A80
  -> E7-02 exact cross-view authority trace
  -> E7-04 derived cross-view review
  -> E7-09 ProjectionをCanonical / second authorityへ昇格しない
```

これはRQ-026のData / Function-Interaction / Business相互検証に直接寄与する。

## 3. A81 — Data/RelationをBusiness sourceへ重複authorせず有用なtrace rowsを得る

Current order_entryでは、少なくともBP-00/01/02/03/05/06/07について既存refsから有用なData traceをderiveできる。

```text
useful Data trace exists
  != Business source owns Data dependency
```

### B-F36 — useful projection != source duplication necessity

**Confirmed source-level boundary.**

Current D4 minimumにmandatory Data / Relation refsを戻す必要性は、A80/A81ではむしろ弱くなった。

```text
Pattern / Validation / Capability authority
  -> V0 Data trace projection

Business source
  -> Patternからderiveできないresidual Business meaning
```

を優先する。

A81はD4 Data slot追加の根拠ではない。

## 4. A82 — no current trace != no Data dependency

BP-04 / BP-08 / BP-09ではcurrent refsからData traceが得られない。

Lane Cはこれを:

```text
NoDataTraceFromCurrentRefs = true
```

として保持し:

```text
No Data dependency
```

とは扱わない。

### Lane B mapping

```text
A82
  -> E7-08 unresolved remains unresolved
  -> E7-04 unknown / omissionをcross-viewで可視化
  -> E7-09 completeness flagやauthoritative matrixを先行導入しない
```

### B-F37 — trace absence != dependency absence

**Confirmed boundary.**

Business Action × Data matrixは現時点では「trace matrix」であり、「complete dependency matrix」ではない。

```text
trace row
  = evidence-backed path found

no trace row
  = unresolved from current refs
  != no dependency
```

この区別をUI / report / matrixでも維持する。

## 5. A83 — BP-06 partial traceとO1 persistence

BP-06はauthoritative Business sourceに`security.authorize` explicit Capability refを持つためsecurity Data traceをderiveできる。

一方:

```text
protected Application target = order_entry.submit_order
```

はO1 test helper inputであり、authoritative Business sourceへpersistしていない。

したがってcurrent projectionはBP-06へ`sales.order`を補完しない。

```text
security trace exists
  != BP-06 Data dependency complete
```

を維持する。

### Lane B mapping

```text
A83
  -> E7-02 exact authority direction
  -> E7-08 partial coverageをcompleteへ丸めない
  -> E7-09 adjacency / O1 helperからDataをbackfillしない
  -> B-F27 protected-target authoritative-source persistence
```

### B-F38 — O1 persistence has a concrete downstream consequence

A83により、B-F27は単なるschema cleanlinessの議論ではなくなった。

もしprotected target relationをaccepted Business meaningとして将来保持するなら:

```text
O1 target persistence
  -> cross-view Data projection / impact analysis coverageへ効く
```

という具体的な利用価値がある。

しかし:

```text
projection coverage consequence exists
  != O1 production persistence is already required
```

である。

Current一例だけでproduction `applicationPatternTargets` fieldやgeneric ref-role taxonomyを導入しない。

B-F27は引き続きconditional unresolvedとするが、reopen triggerは具体化した。

## 6. Derived-only change direction

Lane Cは同じA80〜A83 pressure内で、V0 projectionが第二SOTではないことを変更伝播でもPressureしている。

### Business wording-only change

```text
Business wording changes
explicit refs unchanged
  -> Business accepted meaning may change
  -> Data projection unchanged
```

Business wordingからentityを抽出してData dependencyを推測しないため、これは正しい。

### referenced authority change

```text
Application Pattern / Validation / Capability accepted meaning changes
Business source unchanged
  -> Data projection follows referenced authority change
```

### missing / ambiguous identity

```text
referenced identity cannot be resolved exactly
  -> fail closed
  -X-> name-based補完
```

### B-F39 — derived projection dependency direction is one-way

**Confirmed source-level boundary.**

```text
Business source
  + normalized referenced authorities
  -> V0 Data projection

V0 Data projection
  -X-> author Business source
  -X-> author Application / Semantic authority
```

この一方向性はRQ-003 No Multiple SoT、RQ-017 dual-maintenance最小化、RQ-026 cross-view相互検証と整合する。

Projectionを豊かにするためだけにBusiness sourceへdirect Data refやO1 targetを追加してはならない。

## 7. V0 / V1 / V2への含意

A80〜A83はV0の有用性を強く示す。

```text
V0
  derived trace matrix
  evidence paths
  unresolved/no-current-trace cells
  referenced authority changeへ追随

V1
  Business固有authoring source
  current Data duplication不要

V2
  durable Canonical Business identity
  still not justified by A80〜A83
```

```text
matrix is useful
  != matrix must become authoritative
```

を維持する。

## 8. CRUD matrixとの境界

A80〜A83で安全に得られるのは:

```text
Business Action × Data / Relation trace
```

までである。

CRUDを付けるにはoperation semanticsをPattern / Query / Capability / datasource contractからさらにEvidence-backedに導く必要がある。

```text
Data identity visible
  != C/R/U/D known
```

現段階でCRUDを名称・Pattern種別・screen形状から推測しない。

## 9. PAS-07 Evidence contractへの反映

新E7を追加しない。

```text
A80 -> E7-02 / E7-04 / E7-09
A81 -> E7-02 / E7-09
A82 -> E7-04 / E7-08 / E7-09
A83 -> E7-02 / E7-08 / E7-09 + B-F27
```

Derived-only change directionはA80〜A83の既存claimを深めるものであり新A/E7を作らない。

## 10. Gate 15 / RRBとの関係

A80〜A83はcurrent `order_entry` Referenceを深掘りしたもので、second realistic sourceではない。

```text
RRB-P1 / P2 / P3
  unchanged / not achieved

Gate 15c
  not activated
```

Current ReferenceからかなりのData cross-view traceをderiveできるため、second corpusを急いで作る理由にはしない。

## 11. actual Evidence

Repository上にはtest-only projection / change-direction pressureが実装されているが:

```text
GitHub Actions on latest #482 head
  none

Remote Desktop
  no device available

actual gofmt / go vet / go test / checker green
  unresolved
```

source-level Evidenceをactual execution Evidenceへ読み替えない。

## 12. current decision

```text
A80〜A83
  existing E7へ吸収

B-F36 useful projection != source duplication necessity
  Confirmed

B-F37 no trace != no dependency
  Confirmed

B-F38 O1 persistence has concrete projection consequence
  Confirmed pressure
  but B-F27 remains conditional unresolved

B-F39 V0 projection dependency direction is one-way
  Confirmed source-level boundary

D4 mandatory Data / Relation refs
  still not justified

CRUD matrix
  deferred beyond trace matrix

production Data projection model
  not adopted

Pre-ADR
  Not Ready
```

## 13. 次

1. actual execution環境が復帰すればA80〜A83のchange-direction testsを含むcurrent pressure suiteを実行する。
2. actual green後もData trace projectionを第二SOTへ昇格しない。
3. current refsで失われるaccepted Business Data dependencyが具体的に出た場合だけD4 direct Data/Relation refをreopenする。
4. protected-target relationをaccepted meaningとして残す判断が出た場合、B-F27をData projection / impact analysis coverageも含めて再監査する。
5. CRUD semanticsはoperation authorityがEvidence-backedにderiveできるまで追加しない。
6. projectionを豊かにする目的だけでV1 authoring sourceを膨らませない。
