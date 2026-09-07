# Subsystem Canonical Reopen Gate — 2026-09

Status: **Lane C / T-GAP-05 gate definition / explicit Subsystem still deferred**

## 1. 目的

Current TEIでは`Subsystem` / `SubjectArea`をproduction Canonical nounとして導入していない。

一方、X-TEA / predecessor Semantic PlatformではSubsystemが:

```text
comprehension / cohesion boundary
Table / Function grouping
external-table C/U/D review boundary
published Action boundary
direct / effective CUD propagation boundary
```

として重要な役割を持っていた。

本Gateは、current TEIがderived lensだけで十分な間はexplicit Subsystemを追加せず、**どのEvidenceが揃ったら再検討を開始するか**を先に固定する。

「X-TEAにあったから戻す」でも「current Referenceで不要だから永久に不要」でもない。

## 2. Current derived alternatives

Current Lane CではSubsystem nounなしに二つのreview lensをPressureしている。

### 2.1 Operation ownership boundary lens

```text
Business Action
  -> evidence-backed CRUD operation
     -> Application datasource requirement
        -> Database ownership / access
```

Current `order_entry`では:

```text
BP-00 R sales.order
  -> sales_core / managed-here / read-write

BP-02 R customer.customer
  -> master_data / external-observed / read

BP-03 R sales.order_line
  -> sales_core / managed-here / read-write
```

をderiveできる。

### 2.2 Semantic Namespace lens

```text
Business Action
  -> exact Data trace
     -> owning Semantic Namespace
```

Currentでは:

```text
BP-02 -> sales + customer
BP-06 -> security
```

等のcross-domain dependencyを表示できる。

これらは有用だが:

```text
Database ownership
!= Semantic Namespace
!= Application
!= Business SubjectArea
!= Subsystem cohesion boundary
```

を維持する。

## 3. Historical pressure retained

Predecessor Reference Model Gateではrealistic manufacturing/order domainで:

```text
Subsystems: 6
Data: 42
Explicit Actions: 57
Procedures: 20
Action calls: 4
Effective CUD entries: 54
```

を扱い、Function viewを:

```text
Action + View + CRUD/read/call cross-reference
```

として再構成していた。

またdirect / effective CUDを分け:

```text
Action's own mutation
!= same-Subsystem call propagation
```

を表示し、cross-Subsystem published Action boundaryではeffective CUD inheritanceを止めていた。

Cross-subsystem Procedureも存在し、Procedure自体はSubsystem境界の写しではなかった。

このEvidenceはexplicit boundaryがrealistic scaleで役立つことを示すが、current TEIに同じ`Subsystem` shapeを採用すべきことは証明しない。

## 4. Reopen prerequisites

`Subsystem` production conceptの比較を開始する前に、少なくとも次の前提が必要である。

### R1 — qualifying C/U/D attribution exists

T-GAP-08で少なくとも一つではなく、複数のrealistic TEI-original domain / actionで:

```text
Function / Application meaning
  -> accepted direct mutation authority
```

が成立している。

CRUD matrix completeness目的のsynthetic mutationは数えない。

### R2 — explicit Function call dependency exists

Names / sequence / implementation source解析ではなく、accepted Function / Capability call relationが存在する。

これがなければeffective CUDを設計する対象自体がない。

### R3 — direct / effective distinction becomes review-relevant

少なくとも次が実際に起きる。

```text
caller does not directly mutate Data
callee does mutate Data
```

かつ、両者を区別しないとownership / impact reviewを誤る。

### R4 — cross-boundary encapsulation is exercised

Published / exposed call boundaryのcallee内部mutationをcallerへ直接帰属すると誤解が生じるrealistic caseがある。

### R5 — current derived boundaries become insufficient

Database ownership / Semantic Namespace / Application lensを重ねても、designerが安定して次を説明できない具体的Evidenceがある。

```text
which Data / Functions form one cohesive review unit
where direct mutation responsibility stops
which calls are internal implementation vs published boundary
what should be inspected together after a change
```

R1-R5が揃う前にSubsystem Canonical kindを設計しない。

## 5. Reopen signals

R5の具体的signalとして、次を記録する。

### S1 — one cohesive business design unit spans several current boundaries

例:

```text
one review unit
  spans multiple Applications
  spans multiple Databases
  spans multiple Semantic Namespaces
```

それでも一つの凝集したFunction/Data設計として扱う必要がある。

### S2 — one current boundary contains several independent cohesion groups

例:

```text
one Database / Namespace
  contains several groups that should not inherit internal effective CUD from each other
```

Database / NamespaceをSubsystem代替にすると粗すぎる。

### S3 — repeated manual grouping appears in Editor / review artifacts

Designerが毎回同じData / Function集合を手作業で絞り込まないとreviewできず、そのgrouping自体が安定した意味を持ち始める。

### S4 — impact analysis needs a stable boundary identity

Change impactで:

```text
inside this design boundary
outside through published contract
```

を繰り返し区別する必要があり、単なるderived clusteringではreview continuityを保てない。

### S5 — external direct CUD diagnostic cannot be stated correctly

Database ownershipだけでは:

```text
external to which semantic design boundary?
```

を決められず、合法なinternal mutationと禁止すべきcross-boundary direct mutationを区別できない。

### S6 — call topology becomes unreadable without boundary collapse

realistic ReferenceでFunction call graphが増え、全call graphでは理解困難だが、stable cohesion boundaryごとにcollapseすると自然なreview単位になる。

## 6. Reopen does not mean automatic adoption

R1-R5が成立した場合でも、まず次のoptionsを比較する。

### B0 — derived multi-lens only

```text
Namespace lens
+ Database ownership lens
+ Application lens
+ direct/effective operation projection
+ call graph projection
```

でまだ十分か確認する。

### B1 — derived review grouping without durable Canonical identity

Editor / analysis側でstable source authorityを持たないderived cluster / saved review filterだけで足りるか確認する。

この場合、groupingをBusiness meaningのSOTにしない。

### B2 — lightweight authored review boundary

Core semantic nounではなく、設計レビュー用の明示groupingが必要か検討する。

ただしData ownership / deployment / auth policyを自動所有させない。

### B3 — durable Subsystem semantic boundary

次の意味が本当に独立して必要な場合のみ候補にする。

```text
stable identity
membership / cohesion intent
published Function boundary
direct mutation ownership rule
impact-review continuity
```

### B4 — reuse Application / Database / Namespace

原則として慎重に扱う。

偶然current Referenceで一致していても:

```text
Application == Subsystem
Database == Subsystem
Namespace == SubjectArea / Subsystem
```

を一般化しない。

## 7. Acceptance pressure if B3 is considered

Durable Subsystem candidateを作る場合、少なくとも次を満たす必要がある。

1. Subsystem identityがrepository path / deployment unitから独立する。
2. ProcedureをSubsystem childへ閉じ込めない。
3. Business Action / Function / Dataのmembership意味を説明できる。
4. cross-boundary Rを合法に表現できる。
5. cross-boundary direct C/U/Dをdeterministicにreviewできる。
6. same-boundary internal callのeffective CUDをderiveできる。
7. published boundaryでcallee internal mutationをcaller ownershipへ漏らさない。
8. direct/effective CUDをSubsystem sourceへ重複authorしない。
9. Namespace / Application / Databaseとの違いがReferenceで観察可能である。
10. realistic scaleでEditor comprehensionが改善する。
11. grouping変更のsemantic impactを説明できる。
12. top-level noun追加によるone-maintainer負担に見合うProduct Valueがある。

## 8. Current T-GAP-05 answer

Current answer:

```text
Do effective CUD and call dependencies remain understandable without an explicit
Subsystem semantic contract at realistic scale?

-> current TEI cannot answer yet
```

理由:

```text
accepted C/U/D attribution: not yet available
accepted Function call authority: not yet available
current order_entry operation coverage: R only
current realistic second domain with exact TEI identities: insufficient
```

したがってT-GAP-05をfixtureで人工的に埋めない。

## 9. Current decision

```text
Subsystem Canonical concept
  -> deferred

SubjectArea Canonical concept
  -> deferred

Current derived ownership / Namespace lenses
  -> continue as test-only pressure

T-GAP-05
  -> reopen only after R1-R5
```

このGateにより、将来realistic CUD / Function call pressureが現れた時にはX-TEAのSubsystem vitalityへ戻れる一方、現在の情報不足を新しいCore nounで埋めない。
