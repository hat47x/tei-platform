# Business Procedure Main Baseline Re-audit — 2026-09

Status: **Lane C / refreshed-main architecture re-audit / no semantic candidate change / no production Core adoption**

## 1. 目的

`main`の業務システム設計文書・HTML仕様ガイド・外部参照文書が再整理されたため、Lane Cのcurrent Business Procedure candidateと意味上の矛盾がないかを再監査する。

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

本監査はmainのarchitecture baselineをLane C専用schemaへ書き換えるものではない。

## 2. 公開ArchitectureとLane C研究規則の責務分離

Refreshed mainでは`docs/business-system-design.md`をTEI-local Requirement / Reference / Evidence中心の公開architecture文書として簡潔化し、`UPSTREAM_XTEA_DESIGN_FALLBACK_PROTOCOL.md`と`XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md`も一般的なExternal Design Reference Noteへ縮約した。

一方、`requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`には引き続き、設計判断に迷う場合の参照順序として:

```text
渡辺幸三氏の明示的言説
  -> X-TEA / XEAD manual / model structure
  -> XeadModeler / Driver public source
  -> Reference Model usage
  -> current TEI Requirements / Architecture / Reference / Evidence
  -> small TEI Reference experiment
```

が残っている。

したがってcurrent responsibilityは:

```text
public architecture / specification
  -> TEI-local accepted meaningを説明
  -> third-party lineageをsemantic authorityとして前面化しない

Lane C design investigation
  -> Requirementに従い必要時にX-TEA / XEADへ戻る
  -> question-specific auditでprovenance / support / non-proofを保持
```

である。

詳細な外部Evidenceを横断Registerへ再集約しない。Handoff / Business Flow、O1 authority、CRUD等の具体的Evidenceは、それぞれのLane C audit文書へ近接配置する。

## 3. Business Action / Function separation

Main baselineとLane Cはいずれも:

```text
Business Action != Application Function
```

を維持する。

Lane C D4:

```text
Body
  businessAction
  applicationPatterns[]
  validations[]
  capabilities[]
```

はBusiness ActionをApplication realizationへ還元しないため整合する。

## 4. ProcedureのData / Relation意味

Architecture vocabularyとしてBusiness ProcedureとData / Relationのcross-viewは存在し得る。

しかしcurrent `order_entry` Evidenceでは:

```text
Pattern-owned Data / Relation / Query / Datasource meaning
  -> ir.Worldからderive
  -X-> Business sourceへ重複author
```

を維持する。

一般architecture vocabularyはcurrent minimumへのfield追加根拠ではない。Business sourceだけが保持できるData relationがrealistic Referenceで現れた場合だけreopenする。

## 5. Procedure identity

Lane C candidateには:

```text
Procedure.id
Step.localKey
```

があるが、current `Procedure.id`はまず:

- source内logical identity
- duplicate source detection
- Handoff endpoint addressability
- path-independent normalization

のためのidentityである。

```text
Procedure.id exists
  -X-> durable Canonical Procedure kind
  -X-> ir.World.Business
  -X-> platform-wide Canonical identity contract
```

を維持する。

## 6. Business Flow / Handoff

Main baselineの:

```text
Business Flow
Business Procedure
Application Interaction
Runtime Workflow Orchestration
```

を一つへ畳まない方針と、Lane Cの:

```text
H1/R3 thin Handoff
  != full Business Flow / DFD
  != Runtime Workflow
```

は整合する。

X-TEA側のDataflow node / line構造がこの分離を支持する詳細Evidenceは、`BUSINESS_PROCEDURE_HANDOFF_VS_BUSINESS_FLOW_PRESSURE_2026-09.md`等のquestion-specific auditに保持する。公開横断Registerへ戻さない。

## 7. Actor / Authorization / O1

Main/Lane Cとも:

```text
Business Actor / Job
  != Authenticated Principal
  != Authorization Role / Policy
```

を維持する。

BP-06:

```text
supporting semantic authority = security.authorize
protected target              = order_entry.submit_order
direct realization            = none
```

であり、Authorization policyからBusiness protected targetをauthor/backfillしない。

Task/Function分離がgeneric ref-role taxonomyを証明しないという詳細Evidenceは`BUSINESS_PROCEDURE_O1_AUTHORITY_DIRECTION_PRESSURE_2026-09.md`等へ残す。

## 8. Matrix / CRUD / derived projection

Matrix、CRUD表、graphは可能な限りexplicit accepted sourceからderiveし、第二正本にしない。

Current exact CRUD attribution:

```text
BP-00 -> R sales.order
BP-02 -> R customer.customer
BP-03 -> R sales.order_line
```

Qualifying C/U/D不足はcurrent Procedure schema blockerではない。

## 9. Current result

Refreshed mainはcurrent Lane C minimumをreopenしない。

```text
D4
  -> aligned

singular narrow O1
  -> aligned; generic ref-role taxonomy不要

S4-A
  -> logical/address identityとしてaligned
  -> Canonical kind adoptionではない

H1/R3
  -> aligned; full Business Flow / Workflowとは別

separate normalized Business plane
  -> contradictionなし
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

Current Contract ADR remains:

```text
Not Ready
```

Material blockers:

1. actual Repository runner green未取得
2. independent Business requirement由来のsecond realistic TEI-original Handoff未取得

Main documentation refreshはsemantic blockerではない。Branch同期後もactual runner Evidenceが無い限りPASSは主張しない。

## 11. Reopen triggers

次の具体的Evidenceが出た場合だけ再検討する。

- Business-owned Data relationをWorld/Patternからderiveできずlossが出る
- Procedure logical identityではreview/history/impact continuityが不足する
- H1/R3ではsecond realistic Handoff meaningをlosslessに保持できない
- separate Business planeではcross-view Conformanceをfail closedできない
- generic protected/support/concern roleが複数domainで独立に反復する

それまでは公開architectureの一般語彙、または外部参照の存在だけを理由にLane C minimum contractを膨張させない。
