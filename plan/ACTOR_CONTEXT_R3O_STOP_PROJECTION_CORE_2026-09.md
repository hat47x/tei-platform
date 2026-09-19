# Actor Context R3O — Audience-neutral Stop Projection Core / Rendering Boundary

- Status: ISSUE-000053 / Phase 3 projection-shape pressure / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3N_STAGE_AWARE_OPERATIONAL_STOP_TAXONOMY_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_PROJECTIONS_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3M_OPERATIONAL_STOP_EXPLANATION_HANDOFF_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Mではstop stateをprovenance付きexplanationへ変換し、R3Nでは異なる停止状態をstage-aware taxonomyへ分離した。

次の問いは:

> Human向け表示とAgent向けmachine consumptionへ、どこまで同じ意味を共有させるべきか。

である。

R3Oでは、両者の間にtest-onlyの**audience-neutral semantic core**を置く。

```text
R3M provenance / drill-down
  + R3N stage / class / machine reason / runtime outcome
  -> common stop core
       ├── human rendering
       └── agent projection
```

ただしこのcoreをproduction `ActorContext` / API schemaへ昇格させない。

## 2. 共通coreに残すもの

最低限のcandidateは:

```text
Business address
stop stage
stop class
machine-readable reasons
known authority facts + provenance
resolution need classes
runtime outcome
downstream not attempted
source refs
resolution requested
handoff recipient status
authority limits
```

である。

これは「全audienceへ同じJSONを配る」という意味ではない。

意味の核を同じauthority rootから再構成できることをPressureする。

## 3. Human renderingはsemantic authorityではない

Human向けには短い説明文が有用である。

例:

```text
Authorization action authority is unresolved.
```

しかし:

```text
display summary
  != machine-readable reason
  != authority value
  != stable semantic identifier
```

である。

文言を言い換えてもcommon coreが変化してはならない。

またdisplay summaryだけを保存してmachine reason / provenanceを捨てない。

## 4. Agent projectionへ表示文をdirectiveとして渡さない

Agent向けprojectionでは、exact machine reason / source / resolution needを使える。

一方:

```text
human explanation
  != agent instruction
```

である。

R3OのAgent viewには:

- NextAction
- SuggestedAction
- HandoffOwner
- CanExecute
- Authorized
- Confidence

等を追加しない。

Human表示の「確認してください」のような文言からAgent行動を生成するauthorityを作らない。

## 5. deniedはresolution needへ変換しない

R3Nのexact current `denied` Observationはruntime outcomeである。

R3Oでも:

```text
stop class = evaluated-denied
runtime outcome = denied
resolution need classes = []
```

を保持する。

Human向けには「Authorization was denied」と表示できても:

```text
denied
  != missing authority
  != ask security team
  != retry authorized
```

である。

## 6. R3M / R3Nの整合を検証する

common coreを作る際、同じstopについてR3MとR3Nが矛盾していたらfail closedする。

少なくとも:

- Business address
- current blocking / machine reason
- resolution need classes

を照合する。

```text
projection sources disagree
  -> projection build error
  != choose one silently
```

とする。

## 7. test-only source

- `tools/reference-business-procedure-check/actor_context_r3o_stop_projection_core_pressure_test.go`

Pressureするcase:

1. current BP-06 action-authority stop
2. Human renderer wordingを変えてもcoreが不変
3. Agent viewがHuman display textをauthorityとして持たない
4. exact denied Observationがruntime outcomeとして保持される
5. R3M / R3N reason不一致をfail closed
6. projection shapeにrouting / authorization shortcutを追加しない

## 8. current conclusion

R3Oで固定するのは:

```text
shared semantic core
  != shared presentation
  != stable production schema

human wording
  != machine reason
  != agent directive

machine projection
  != authorization
  != next-action authority
```

である。

共通coreはderived / regenerableであり、Canonical sourceやruntime source of truthにはしない。

## 9. 次

次段では、common coreをどこまでpurpose-boundに削れるかをPressureする。

特に:

- Human UIに不要なruntime detailをomitできるか
- Agentに不要なpresentation textをomitできるか
- omissionを「情報が存在しない」と誤認させないか
- selected / omitted / unresolved categoryを説明できるか

を確認する。

これは `ACTOR_CONTEXT_PROJECTIONS_2026-09.md` の bounded projection / omission boundaryへ戻る。
