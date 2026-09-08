# Business Model Production Acceptance Boundary Audit — 2026-09

Status: **Lane C / Pre-ADR production-boundary pressure / no production transaction API adoption**

## 1. 目的

Business Procedure / Handoffのcurrent minimum candidateは複数authoring sourceを一つのnormalized BusinessModelへ束ねる。

```text
Procedure source A ─┐
Procedure source B ─┼─> normalize -> BusinessModel
...                ─┤
Handoff source     ─┘
```

Test-only PressureではBusiness source freshness、Business semantic revision、coordinated cross-source changeまで成立している。

本書では、これをproduction Shared Acceptanceへ接続する際にLane Cが要求すべき**意味上の前提条件**だけを絞る。Git transaction、filesystem locking、Studio API、AI Proposal API等の物理実装は採用しない。

## 2. Cross-lane ownership

既存のshared Acceptance設計をLane Cへ複製しない。

```text
ISSUE-000027 AI Proposal / Acceptance
  -> candidate + baseModelRevision + shared Acceptance principle

ISSUE-000039 Business change rehearsal
  -> read-only Semantic diff / impact analysis
  -> authoritative write / Acceptance Policyは所有しない

ISSUE-000046 Studio authoring loop
  -> candidate + baseModelRevision
  -> stale candidateを黙って適用しない
  -> shared Acceptance Boundaryへ接続

Lane C
  -> Business multi-source normalization固有のfreshness / conformance obligations
  -> Business semantic meaningのlossless reload条件
```

Lane Cから000027 / 000039 / 000046へtransaction engineやgeneric revision modelを逆輸出しない。

## 3. 三つのidentityを混同しない

Production Shared Acceptanceで少なくとも次の三つは意味が異なる。

### A. Business source freshness

```text
logical Business source key
+ source content state
```

Preview対象のProcedure / Handoff sourceがAccept直前まで同じかを見る。

Current test-only representationはlogical key + content fingerprintだが、hash / ETag algorithmはproduction contractではない。

### B. Business semantic revision

```text
all Business sources
  -> normalize
  -> BusinessModel meaning
```

Procedure wording、protected target、Handoff condition等のaccepted Business meaningを比較する。

source pathやsource enumeration orderをsemantic identityへ含めない。

### C. Cross-view Conformance base

```text
BusinessModel
  <-> current ir.World / reloadable workspace semantic base
```

Preview時にBusinessModelをcross-view validateした相手側のmeaningである。

Business sourceが一切変わっていなくても、Preview後にApplication Pattern / Validation / Capability等のWorld側meaningが変われば、以前のConformance判定はそのままAccept根拠にできない。

したがって:

```text
Business source fresh
  != Conformance base fresh

Business semantic revision unchanged
  != current WorldとのConformanceが再確認済み

Business semantic revision
  != ir.World revision
```

である。

## 4. Preview境界

Conceptual Previewは次の意味を持つ。

```text
current Business sources
+ candidate change-set
        ↓
source freshness snapshot
        ↓
normalize all affected Business sources
        ↓
Business semantic candidate / revision
        ↓
validate Business referential integrity
        ↓
cross-view validate against current World
        ↓
Preview result
  - Business source base
  - Business semantic candidate
  - Conformance base
  - diagnostics / projection / impact
```

Previewはauthoritative Business sourceを変更しない。

`Business source base`、`Business semantic candidate`、`Conformance base`の具体型・hash形式はまだ固定しない。

## 5. Accept前に必要なfail-closed条件

Accept時には少なくとも次を満たす必要がある。

1. Preview対象Business sourceがstaleでない。
2. Preview時のConformance baseがstaleでない、またはcurrent Worldへcross-view validationをAccept直前に再実行する。
3. candidate全体をnormalizeし、Procedure / Step / Handoff referential integrityを再確認する。
4. protected Application target等をcurrent Worldへexact resolutionする。
5. cross-view Conformance失敗時にBusiness sourceのpartial writeを残さない。
6. coordinated Step rename + Handoff endpoint rename等を一つのsemantic change-setとして扱う。
7. Accept後に全Business sourceをreloadし、accepted Business semantic meaningを再現する。
8. reload後のBusinessModelをcurrent Worldへ再度Conformance確認できる。

重要なのは「どのDB transaction APIを使うか」ではなく、**古いBusiness sourceまたは古いWorldを根拠にacceptしないこと**である。

## 6. World変更negative case

Current ReferenceにはBP-06 protected target:

```text
order_entry.submit_order
```

がある。

Preview後にWorldからこのApplication Patternが削除された場合:

```text
Business sources
  unchanged / fresh

Business semantic revision
  unchanged

World / Conformance base
  changed

cross-view result
  stale target -> fail closed
```

となる。

`business_model_conformance_base_freshness_pressure_test.go`はこの境界をtest-onlyにPressureする。

このcaseによりBusiness source fingerprintだけではShared Acceptance freshnessとして不足することが分かる。

## 7. baseModelRevisionとの関係

ISSUE-000027 / 000046ではcandidateに`baseModelRevision`を持ち、stale candidateを黙って最新版へ適用しない。

Lane Cはこれを否定しない。ただしBusiness multi-source caseでは一つのopaque revision tokenへ意味を潰す前に、少なくとも概念上:

```text
candidate base
  includes enough information to reject
    - stale Business authoring sources
    - stale cross-view Conformance base
```

であることを要求する。

Production implementationが一つのrepository revisionで両者を安全に固定できるなら、tokenが物理的に一つでもよい。逆にBusiness sourcesとWorld sourcesが独立更新可能なら、それぞれを検証できる必要がある。

Lane Cからplatform-wide `baseModelRevision` schemaを決めない。

## 8. Business semantic revisionの役割

Business semantic revisionはfreshness lockではない。

例えばwhitespace / formattingだけが変わった場合:

```text
source freshness
  changed

normalized Business semantic meaning
  unchanged
```

があり得る。

一方Business wording / O1 target / Handoff conditionが変われば:

```text
Business semantic meaning
  changed
```

として扱う。

したがってsemantic revisionはreview / diff / evidence bindingには有用だが、Accept直前のsource freshness確認を代替しない。

## 9. Migration境界

Current Reference authoringからproduction candidateへのtest-only migrationでは:

```text
legacy Procedure source
+ legacy O1 overlay
+ Reference Set Handoff
        ↓ preview migration
inline singular O1 Procedure source
+ standalone Handoff source
```

へ変換してもnormalized Business meaningを保持できることをPressure済みである。

Production migrationを行う場合の条件:

- legacy authoritative inputをin-place mutationするpreviewにしない
- O1 multi-targetが現れたらsingular candidateへ黙って縮退しない
- path / artifact locatorをBusiness semantic fieldへコピーしない
- Procedure + Handoff coordinated migrationを一つのaccepted change-setとして扱う
- migration後reloadでsame normalized meaningを再現する

Actual runner greenとproduction writer ownershipが確定するまで、current Reference filesをproduction formatへ置換しない。

## 10. 採らない設計

現時点では次を採用しない。

- `BusinessRevision` platform-wide production type
- Business専用transaction engine
- Business専用Approval store
- Git commit = Business semantic revisionという固定規則
- hash / ETag algorithmの固定
- `ir.World.Business`
- Business semantic revisionと`ir.World` revisionのcoupling
- source pathをfreshness identity / Business identityへすること
- stale WorldでもBusiness sourceがfreshならAcceptすること
- Preview diagnosticsだけを保存してAccept時revalidationを省略すること

## 11. Pre-ADR Gate Eへの帰結

Conceptual Gate Eは次まで絞れる。

```text
multi-source normalize
  -> test-only Evidenceあり

Business source freshness
  -> test-only Evidenceあり

coordinated referential atomicity
  -> test-only Evidenceあり

World / Conformance base freshness
  -> test-only Pressure追加

shared Acceptance principle
  -> ISSUE-000027 / 000046と整合

read-only semantic diff / impact
  -> ISSUE-000039へ委譲
```

Production ADR前に残る問いは:

1. existing shared Acceptance implementationがBusiness source baseとConformance baseをどう固定・再確認するか。
2. candidate applyをどのrepository transaction境界でatomicにするか。
3. Accept後reloadでsame Business semantic revisionをどうEvidence化するか。
4. actual Repository runnerでmulti-source / freshness / conformance-base testsがgreenになること。

この4点は重要だが、Lane Cがgeneric Acceptance APIを先行設計する根拠にはしない。

## 12. Current decision

```text
Business source freshness
  != Business semantic revision
  != cross-view Conformance base

Accept
  requires fresh Business source base
  + fresh/revalidated Conformance base
  + atomic Business change-set
  + reload / Conformance

Shared Acceptance implementation ownership
  -> existing cross-lane acceptance workへ接続

Lane C generic transaction API
  -> not adopted

Pre-ADR
  -> Not Ready
```
