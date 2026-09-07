# Business Procedure A106–A110 Pressure Delta — 2026-09

Status: **Lane C / ISSUE-000042 inventory delta / not a readiness authority / no revision API adoption**

## 1. 目的

A100–A105以後に追加したphysical `baseModelRevision` / stale-envelope Pressureを、次回`ISSUE-000042` matrix統合用に整理する。

ADR readinessの正本は引き続き:

`BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`

である。

## 2. Delta inventory

| ID | Pressure | Current handling |
| --- | --- | --- |
| A106 | one physical `baseModelRevision`はBusiness source base + World / Conformance baseをfalse-negativeなくcoverするcomplete stale envelopeなら許容できる | option audit fixed / token schema未採用 |
| A107 | Git HEAD equalityだけではdirty / untracked authoritative source変更を検出できないためunconditional freshness proofにはならない | negative boundary fixed |
| A108 | unrelated repository changeによるfalse-positive staleは保守的tradeoffとして許容できるが、relevant Business / World changeを見逃すfalse-negative staleはrejectする | correctness rule fixed |
| A109 | physical freshnessは個別file contentだけでなくsource add/remove・authoritative source-root membership changeもcoverしなければならない | discovery-set boundary fixed |
| A110 | Business / World source planeが将来独立versioningされる場合、一つのrepository SHAに依存せず両stale conditionを独立検出またはcomposite tokenでcoverする必要がある | future reopen boundary fixed |

## 3. A106 — one tokenは概念分離を消さない

Current conceptual obligations:

```text
Business source freshness
  != Business semantic revision
  != cross-view Conformance base
```

Production実装が一つのopaque tokenにまとめても:

```text
Business source changed?
World / Conformance base changed?
```

という二つのfreshness correctness obligationは消えない。

Relevant:

- `BUSINESS_MODEL_PHYSICAL_BASE_REVISION_OPTION_AUDIT_2026-09.md`
- `BUSINESS_MODEL_PRODUCTION_ACCEPTANCE_BOUNDARY_AUDIT_2026-09.md`
- `ISSUE-000027`
- `ISSUE-000046`

## 4. A107 — Git HEAD-only negative pressure

Problem shape:

```text
HEAD = R
Preview against source bytes at R

then
  authoritative Business or World file edited in working tree
  HEAD still R

Accept compares HEAD only
  -> falsely fresh
```

したがって:

```text
Git HEAD alone
  -X-> complete freshness proof
```

である。

Current single-repository構成では:

```text
repository revision R
+ authoritative base is exactly represented by R
```

というclean/exact-base invariantが成立するなら、one-token conservative envelopeはviableである。

Lane CはGit SHA formatをplatform-wide contractへ固定しない。

## 5. A108 — stale判定の安全側

Correctness priority:

```text
false-negative stale
  -> reject

false-positive stale
  -> conservative but acceptable
```

例:

```text
unrelated plan document changed
  -> repository revision changes
  -> candidate stale
```

は効率上は過剰だが、古いBusiness/World baseを黙ってAcceptするより安全である。

後からscoped snapshotでprecisionを上げる余地はある。

## 6. A109 — source-set membershipもbase state

Workspace discovery pressureから、freshnessがcoverすべき物理状態はfile bytesだけではない。

少なくとも:

```text
source content change
source add
source remove
authoritative source-root membership change
```

を見逃してはいけない。

個別artifact fingerprintがすべて同じでも、新規authoritative sourceが追加された場合はsource set全体が同一とは限らない。

したがって:

```text
per-file fingerprints only
  without set-membership state
  -X-> complete source freshness
```

である。

## 7. A110 — independent source planes

将来:

```text
Business repository A
World repository B
```

またはconnected external source等へ分かれた場合:

```text
one Git SHA from A
  -X-> B freshness
```

である。

その場合は:

```text
Business base check
+ World / Conformance base check
```

を独立に成立させる。

物理tokenを二つにするかcomposite one-tokenにするかはshared Acceptance implementationが決める。

## 8. Gate E implication

Lane CのGate Eで未解決だった:

```text
semantic revision identity / source freshness / Conformance baseを
one physical tokenへ束ねるか分離するか
```

について、**tokenの個数はLane C pre-ADR design decisionではない**ところまで絞れた。

Gate Eが要求すべきなのは:

```text
complete stale detection
+ current Conformance or safe revalidation
+ atomic apply
+ candidate-only reload
```

である。

したがってtoken shapeそのものを理由にBusiness Procedure ADRを止め続けない。

ただしproduction shared Acceptance prototypeでcomplete stale envelopeが実証されるまではGate E全体はgreenにならない。

## 9. Non-adoptions

- Git SHAのplatform-wide `baseModelRevision`固定
- mandatory clean Git repository product rule
- Business専用revision type/store
- Business semantic revisionとGit revisionの同一化
- Business / World revision coupling type
- composite ETag schema
- source fingerprint algorithm固定

## 10. Current conclusion

```text
A106-A110
  = documented

one physical token
  = conditionally acceptable

Git HEAD-only
  = insufficient when authoritative base is not known exact/clean

false-positive stale
  = acceptable

false-negative stale
  = reject

Gate E token-shape decision
  = no Lane C-specific schema decision required

Pre-ADR
  = still Not Ready for execution / production apply-reload / second realistic Handoff
```
