# Business Procedure O1 File-Native / Shared Acceptance Pressure — 2026-09

Status: **Lane C / O1 representation narrowing / test-only / no production schema adoption**

## 1. 目的

Current BP-06には、既存D4 bodyだけではlosslessに保持できないaccepted cross-view pressureがある。

```text
Business Action
  determine whether the authenticated subject may execute the submit application action

supporting Semantic authority
  security.authorize

protected Application target
  order_entry.submit_order

direct Application Function realization
  none
```

`applicationPatterns[]`はdirect realizationを意味するため、protected targetをそこへ入れると意味を誤変換する。

これまでO1 test-only candidateで:

```text
applicationPatternTargets[]
```

を別roleとして保持できることは確認した。

本Pressureでは次を追加確認する。

1. protected targetをfile-native sourceとしてdecode / reloadできるか。
2. stale Procedure / Step / targetをfail closedできるか。
3. protected-target-only changeがShared Acceptance approvalをinvalidにできるか。
4. sidecar source方式の追加コストをproduction representation判断へ反映できるか。

## 2. File-native test-only overlay

追加したReference pressure artifact:

```text
reference/business-system-design/order-entry-protected-target-overlay.json
```

内容は概念上:

```text
schemaVersion = business-procedure-pattern-target-overlay-v0
artifactRole  = reference-pressure-overlay
canonical     = false
procedure     = order-entry.order-handling

BP-06
  applicationPatternTargets = [order_entry.submit_order]
```

重要:

- primary `business-procedure-reference-v0` schemaは変更していない。
- production Canonical sourceではない。
- generic `role: protects`等は受け付けない。
- O1のsource mechanicsを検査するためだけのsidecarである。

## 3. File-native pressure result

`application_pattern_target_authoring_pressure_test.go`で次をPressureする。

### F1 — round-trip

JSON decode / encode / reload後も:

```text
BP-06 -> order_entry.submit_order
```

をlosslessに保持する。

### F2 — target != realization

File-native targetを読み込んでも:

```text
BP-06 ApplicationPatterns = []
BP-06 Capabilities        = [security.authorize]
BP-06 protected targets   = [order_entry.submit_order]
```

を維持し、BP-06をApplication Function realized stepへ変換しない。

### F3 — stale Procedure / Step fail closed

OverlayのProcedure identityまたはStep local addressがprimary Business sourceと一致しなければrejectする。

### F4 — duplicate source fail closed

同一Stepへ複数overlay entryを持たせて曖昧なtarget authorityを作らない。

### F5 — generic role vocabulary fail closed

Unknown field:

```text
role: protects
```

を受理しない。

O1を理由にgeneric role taxonomyをsidecarへ忍び込ませない。

## 4. Existing Procedure-only Acceptance blind spot

Current Business Procedure authoring pressureでは、Procedure source file自身のcontent fingerprintをAcceptance targetとしている。

`sourceindex.NewArtifact`のfingerprintはsource pathを含まず、content bytesだけのSHA-256である。

この性質は:

```text
pure file move
  -> same fingerprint
```

を可能にするため望ましい。

しかしO1を別sidecarへ置くと問題がある。

```text
Procedure source unchanged
O1 overlay target changed
```

の場合:

```text
Procedure fingerprint = unchanged
Business protected-target meaning = changed
```

となる。

`application_pattern_target_procedure_only_acceptance_negative_test.go`では、overlay targetを:

```text
order_entry.submit_order
  -> order_entry.order_record
```

へ変更した後も、current Procedure-only `acceptProcedureSource`が旧Procedure fingerprintで成功することを意図的に確認する。

これはcurrent Procedure acceptanceのbugではない。

> separate semantic sourceを追加するなら、Procedure-only fingerprintではそのsourceを保護できない

というsidecar designへのnegative Evidenceである。

## 5. Composite Shared Acceptance pressure

Sidecar方式を成立させる最小条件として、test-only composite fingerprintをPressureする。

```text
business-procedure-source=<procedure content fingerprint>
protected-target-overlay=<overlay content fingerprint>
  -> sourceindex.Fingerprint(...)
```

Pathは含めず、semantic source roleとcontent fingerprintだけを合成する。

### A1 — target-only change invalidates approval

Overlay targetだけを変更すると:

```text
Procedure fingerprint = unchanged
Overlay fingerprint   = changed
Composite fingerprint = changed
```

となり、旧composite fingerprintによるacceptanceをrejectする。

### A2 — pure overlay file move is no-op

Overlay contentを変更せずfile pathだけを移動すると:

```text
Overlay fingerprint   = unchanged
Composite fingerprint = unchanged
```

となる。

これはexisting Procedure source move semanticsと一致する。

### A3 — fresh fingerprint is not enough

Composite fingerprintを更新しても、target identityがnormalized Worldでmissingならacceptしない。

```text
fresh source revision
  != semantically valid source
```

を維持する。

### A4 — source roles are explicit

Procedure source fingerprintとoverlay source fingerprintは単なるunordered setとして混ぜず、役割付きで合成する。

```text
Business Procedure source
Protected-target source
```

の区別を失わない。

## 6. Representation option audit after file-native pressure

### O1-R0 — wording only / structured targetを持たない

Current Business wordingにはsubmit actionへの言及があるため、人間には意味を読める。

しかし:

- exact target resolution不可
- target rename / stale detection不可
- Authorization policyとのdeterministic corroboration不可
- target-only semantic diffを構造化できない

Current: **insufficient if protected target is accepted review meaning**。

### O1-R1 — production sidecar overlay

Pros:

- primary Procedure schemaを膨らませない
- role-specific sourceを独立管理できる

Cons:

- Procedure-only acceptanceでは意味変更を見逃す
- composite source revision / source-set identityが必要
- source move / stale / duplicate / registry管理が二source化
- Business Procedureの一つのaccepted meaningが複数fileへ分散する
- one-maintainer operational burdenが増える

Current: **test apparatusとして有用 / production candidateとして弱い**。

### O1-R2 — same Business source内のnarrow field

Conceptual pressure only:

```text
step
  businessAction
  applicationPatterns[]        // direct realization
  applicationPatternTargets[]  // protected/concern target, narrow role
  validations[]
  capabilities[]
```

Pros:

- one Business source revisionでapprovalをbindできる
- Procedure/Step renameと同じsource transactionで変更できる
- O1 meaningをD4 bodyと同じauthoring lifecycleへ載せられる
- generic role taxonomyを導入しなくてよい

Risks:

- one exact BP-06 caseからproduction fieldを増やすことになる
- field名/role semanticsを将来一般化しにくい可能性
- Application Pattern以外のfuture protected targetを別modelで扱う必要が出る可能性

Current: **preferred narrow production candidate if O1 is retained; not adopted**。

### O1-R3 — generic role-qualified semantic reference

```text
ExplicitSemanticRef {
  Kind
  Role
  Target
}
```

Current: **defer / effectively reject for current contract**。

O1一例から`protects / concerns / supports / checks / realizes...` taxonomyを作るEvidenceはない。

## 7. Source-of-truth judgment

Sidecar pressureから得られる重要な結論は:

> fileを分ければCore schemaを小さく保てる、とは限らない。

Semantic meaningが一体なら、別file化は:

```text
source set
composite fingerprint
stale coordination
move coordination
approval coordination
```

という別の複雑さを生む。

P3 `no implicit multiple SOT`、RQ-017 dual-maintenance minimization、one-maintainer maintainabilityの観点では、O1をaccepted Business meaningとしてproductionへ残す場合、**同じBusiness source内のnarrow explicit field**の方が現在は自然である。

ただしこれはproduction schema adoption decisionではない。

## 8. What is now proven vs still open

### Stronger Evidence now available

```text
protected target != direct realization
protected target exact identity is useful for deterministic cross-view review
file-native narrow target can round-trip
stale Procedure/Step can fail closed
separate sidecar needs composite Acceptance fingerprint
target-only change must invalidate approval
pure source move should remain semantic no-op
```

### Still open

```text
actual gofmt / go vet / go test green
Shared Acceptance production API shape
whether O1 should survive into the minimum production contract
whether a second realistic domain repeats target-vs-realization role
future non-Application protected target needs
generic role taxonomy
```

## 9. Pre-ADR implication

Pre-ADR checklist item:

```text
O1をproduction accepted meaningとして残す場合、
authoritative source / decode / reload / Shared Acceptance境界を閉じる
```

について、test-only構造上はかなり前進した。

Current classification:

```text
authoritative production source
  not yet

decode / reload pressure
  implemented test-only

stale Procedure / Step
  implemented test-only

Shared Acceptance target-only invalidation
  implemented test-only composite pressure

production representation
  sidecar disfavored
  narrow inline field preferred candidate
  generic role model deferred
```

Actual runner greenとPre-ADR全体の整合が取れるまでReadyへ変更しない。

## 10. Current decision

今はprimary fixture schemaへfieldを追加しない。

```text
O1 meaning pressure
  -> retain

production sidecar
  -> disfavor

production narrow inline field
  -> preferred candidate, not adopted

generic role-qualified ref
  -> defer

Procedure-only Acceptance for separate O1 source
  -> insufficient by construction

composite Acceptance
  -> test-only proof of requirement if sidecar were used
```

この結果により、O1の未決は「target roleが必要か」から、より狭い:

> accepted O1 meaningをminimum production Business sourceへ残す価値が、一つのnarrow field追加に見合うか

へ絞られた。
