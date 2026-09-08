# Business Procedure Main Baseline Re-audit — 2026-09

Status: **Lane C / post-main-merge architecture re-audit / no semantic candidate change / no production Core adoption**

## 1. 目的

`main`で`docs/business-system-design.md`、`docs/naming.md`、X-TEA Evidence関連文書が再編されたため、Lane Cのcurrent Business Procedure candidateと意味上の矛盾がないかを再監査する。

Current Lane C candidate:

```text
D4 + singular narrow O1 + S4-A + H1/R3
```

Placement direction:

```text
Business authoring sources
  -> normalized Business semantic plane

TEI workspace sources
  -> ir.World

normalized Business model
  <-> exact cross-view Conformance <-> ir.World
```

本監査はmainのarchitecture baselineをLane C専用schemaへ書き換えるためのものではない。

## 2. Business Action / Function separation

Current main baseline:

```text
Business Action != Application Function
```

かつmanual / physical / conversation / external system / Capability / unresolved等を正常なBusiness meaningとして認める。

Lane C D4も:

```text
Body
  businessAction
  applicationPatterns[]
  validations[]
  capabilities[]
```

としてBusiness ActionをApplication realizationへ還元しない。

判定:

```text
main baseline
  == Lane C direction
```

矛盾なし。

## 3. Procedureのoptional Data / Relation reference

Main architecture vocabularyはProcedure stepについて:

```text
optional Data / Relation reference
```

を許容する一般形を示している。

一方、Lane C current minimumではorder-entry Referenceのexact Evidenceから:

```text
Pattern-owned Data / Relation / Query / Datasource meaning
  -> ir.Worldからderive
  -X-> Business sourceへ重複author
```

を採用している。

これは矛盾ではなく、一般architecture vocabularyをcurrent Reference Evidenceで狭めたspecializationである。

```text
main:
  Data / Relation cross-view referenceは必要に応じて可能

Lane C current candidate:
  current exact Data dependencyはWorld/Patternから導出できるため
  authored Procedure fieldには採用しない
```

将来、Business sourceだけがauthoritativeに保持すべきData relationがrealistic Referenceで現れた場合だけreopenする。

したがってmainの`optional Data / Relation reference`を根拠にcurrent D4へData fieldを追加しない。

## 4. Procedure identityとProcedure.id

Main baselineは:

```text
Procedureをdurable Canonical identityにするか
traceable derived artifactで足りるか
  = unresolved
```

としている。

Lane C S4-A / authoring candidateには:

```text
Procedure.id
Step.localKey
```

がある。

ここでLane C `Procedure.id`はまず:

- candidate source内logical identity
- duplicate source detection
- Handoff endpoint addressability
- path-independent normalization

のために必要なidentityである。

これは:

```text
Procedure.id exists
  -X-> Micro Kernel Canonical kind adoption
  -X-> ir.World.Business embedding
  -X-> platform-wide durable semantic identity contract
```

である。

`docs/naming.md`はworkspace identity / semantic identity / physical representationを分離する。Lane Cではproduction Canonical adoption前なので、Procedure.idを**Business candidate logical identity**として扱い、既存`semantic identity = Canonical ID`という用語へ早期昇格しない。

判定: 矛盾なし。ただし用語上のguardrailとして`logical identity candidate`を維持する。

## 5. Business FlowとHandoff

Main baselineは:

```text
Business Flow
Business Procedure
Application Interaction
Runtime Workflow Orchestration
```

を一つのsemantic objectへ畳まない。

Lane C current decisionも:

```text
H1/R3 thin Handoff
  != full Business Flow / DFD
  != Runtime Workflow
```

である。

mainへ再統合したX-TEA Evidence E-014も同じ境界をcorroborateする。

判定: strong alignment。

## 6. Actor / Authorization boundary

Main baseline:

```text
Business Actor / Job
  != Authenticated Principal
  != Authorization Role / Policy
```

Lane C:

```text
Procedure.actor
  != Handoff endpoint identity
  != Authorization authority
```

BP-06:

```text
supporting semantic authority = security.authorize
protected target              = order_entry.submit_order
direct realization            = none
```

であり、Authorization policyからBusiness protected targetをbackfillしない。

判定: strong alignment。

## 7. Shared Acceptance wording

Main baselineはEditor / Human / CLI / AIが同じShared Acceptance Boundaryへ接続するarchitecture directionを示す。

Lane C readiness splitでは:

```text
Contract ADR readiness
Production adoption readiness
Product Gate C後のStudio / AI shared Acceptance integration
```

を分離している。

したがってmainの記述を:

```text
shared Acceptance production substrate is already implemented
```

とは解釈しない。

Current absence of the production substrate remains implementation / roadmap sequencing gapであり、Business schema expansion Evidenceではない。

## 8. Matrix / derived projection

Main baselineはMatrixをderived Projectionとし、second source of truthへしない。

Lane C Data / CRUD projectionも:

```text
explicit Evidence pathだけをderive
unknownはunknownのまま
Data identityだけからCRUDを推測しない
```

ため整合する。

Current exact CRUD attribution:

```text
BP-00 -> R sales.order
BP-02 -> R customer.customer
BP-03 -> R sales.order_line
```

Qualifying C/U/D不足はcurrent Procedure schema blockerではない。

## 9. Current main re-audit result

```text
D4
  -> aligned

singular narrow O1
  -> aligned; main does not require generic ref-role taxonomy

S4-A
  -> aligned as Business candidate logical/address identity
  -> not yet Canonical kind adoption

H1/R3
  -> aligned; Business Flow / Workflow separation strengthened

separate normalized Business plane
  -> no contradiction found

Pattern-owned Data derivation
  -> current Lane C specialization of broader main vocabulary
```

No current main change requires:

- `ir.World.Business`
- generic Business relation bag
- generic ref-role enum
- Procedure-authored Data/Relation fields
- full Business Flow schema
- runtime Workflow fields
- durable Canonical Procedure kind

## 10. Readiness impact

Main re-auditによるsemantic candidate changeはない。

Contract ADR remains:

```text
Not Ready
```

material blockers:

1. actual Repository runner green未取得
2. independent Business requirement由来のsecond realistic TEI-original Handoff未取得

Main integration conflictはmerge commitで解消済みであり、current branchはmainに対してbehind 0へ復帰している。

## 11. Reopen triggers

次の具体的Evidenceが出た場合だけ再検討する。

- Business-owned Data relationをWorld/Patternからderiveできずlossが出る
- Procedure logical identityではreview/history/impact continuityが不足する
- H1/R3ではsecond realistic Handoff meaningをlosslessに保持できない
- separate Business planeではcross-view Conformanceをfail closedできない
- generic protected/support/concern roleが複数domainで独立に反復する

それまではmainの一般architecture vocabularyを理由にLane C minimum contractを膨張させない。
