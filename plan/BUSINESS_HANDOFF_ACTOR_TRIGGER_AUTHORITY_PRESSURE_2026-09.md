# Business Handoff Actor / Trigger Authority Pressure — 2026-09

Status: **Lane C / test-only structural pressure / second realistic Handoff is still unresolved / no production schema adoption**

## 1. 問い

Current H1/R3 candidateは:

```text
Handoff
  from { procedure, step }
  to   { procedure, step }
  condition
```

だけを保持する。

Pre-ADR Gate Cでsecond realistic Handoffを確認する前に、Watanabe external design evidenceから次の疑問が出る。

```text
actor / job context
Procedure trigger / event context
```

をHandoff自身へ追加しないと、responsibility continuationをlosslessにreviewできないか。

Current Referenceだけでrealistic sufficiencyを証明することはできないが、少なくとも**authority duplicationが必要か**をtest-onlyに反証する。

## 2. Actor context pressure

Current primary / follow-up Proceduresはともに`actor = sales operator`であるため、same-actorだけではHandoff endpoint sufficiencyを十分にPressureしていない。

`handoff_actor_trigger_independence_pressure_test.go`ではtarget Procedureだけを:

```text
order-entry.follow-up
  actor = sales follow-up coordinator
```

へ変更する。

Handoff sourceは変更しない。

Expected:

```text
from Procedure/Step
  -> exact resolve

to Procedure/Step
  -> exact resolve

Handoff normalized meaning
  -> unchanged
```

Resulting design pressure:

```text
Procedure actor context
  != Handoff endpoint identity

actor label
  need not be duplicated into Handoff
```

ActorはProcedure側Business contextとしてreview可能であり、Handoff relationへcopyしない。

これはactor taxonomy / Job Canonical kindが不要だという最終証明ではない。Second realistic caseでactor changeそのものがhandoff meaningになる具体例が出た場合だけreopenする。

## 3. Procedure trigger / Handoff condition pressure

Watanabe external evidenceはBusiness workがevent-drivenに開始され得ることをcorroborateする。

しかし:

```text
Handoff condition
  == target Procedure trigger
```

とはしない。

Test-only Pressureでは二方向を分ける。

### A. target Procedure trigger wordingだけ変更

```text
Handoff
  unchanged

Procedure.trigger
  changed
```

Handoff normalized meaningは変化しない。

### B. Handoff conditionだけ変更

```text
Handoff.condition
  changed

Procedure.trigger
  unchanged
```

Handoff semantic relation changeを観測し、target triggerへ逆書きしない。

したがってcurrent authority directionは:

```text
Procedure.trigger
  -> Procedure start/context authority

Handoff.condition
  -> responsibility continuation relation authority

neither authors the other
```

とする。

## 4. Second realistic acceptance seedへの更新

`BUSINESS_HANDOFF_WATANABE_SECOND_REALISTIC_EVIDENCE_PRESSURE_2026-09.md`で挙げたminimum questionsを現在Evidenceで分類する。

| Question | Current status |
| --- | --- |
| 1. `from {procedure, step}`でsource responsibilityを指せるか | current primary Handoffでtest-only |
| 2. `to {procedure, step}`でnext responsibilityを指せるか | current primary Handoffでtest-only |
| 3. natural-language `condition`でcontinuation meaningを保持できるか | one current caseのみ。second realistic unresolved |
| 4. target Procedure triggerとHandoff conditionを別authorityにできるか | **new test-only structural pressureあり** |
| 5. Business document / event identityがHandoff fieldに必須か | unresolved / realistic Evidence待ち |
| 6. actor / job change時にendpointだけでaddressできるか | **new test-only structural pressureあり** |
| 7. Business Flow / DFDなしでHandoff review可能か | current primary caseではyes / realistic generalization未証明 |

これによりsecond realistic Handoffの主な未解決は:

```text
condition sufficiency
business event/document identity necessity
broader realistic review context
```

へさらに絞られる。

## 5. External source quality note

Watanabe bookについて、書籍の存在・著者・出版年・「業務フローのパターン」等の章構成は日本実業出版社系書誌、国立国会図書館、版元ドットコム等で確認できる。

一方、次の細目:

```text
要求の認識と報告
作業指示の立案と発行
作業の実施と報告
実績の集計と分析
```

はcurrent web-accessible evidenceでは主に二次的な目次転載から得ている。

したがってこれらをTEI exact Procedure / Handoff authorityへ使用しない。

Current use is only:

```text
realistic external pressure candidate
not exact TEI source
```

である。

## 6. 今追加しないもの

- `Handoff.fromActor`
- `Handoff.toActor`
- `Handoff.actorRole`
- `Handoff.eventId`
- `Handoff.document`
- target Procedure triggerからのcondition自動生成
- Handoff conditionからのProcedure trigger自動生成
- actor labelからのAuthorization Role生成
- Business Flow / DFDのHandoffへの統合

## 7. Current decision

```text
Handoff endpoint
  = Procedure + Step local address

Procedure actor context
  = separate authority

Procedure trigger
  = separate authority

Handoff condition
  = separate authority

actor / trigger duplication into Handoff
  = current structural Evidenceでは不要

second realistic Handoff
  = still unresolved

H1/R3 expansion
  = not justified
```

Actual Repository runner green後にTEI-original second realistic caseを追加し、この構造Pressureがrealistic reviewでも成立するかを確認する。
