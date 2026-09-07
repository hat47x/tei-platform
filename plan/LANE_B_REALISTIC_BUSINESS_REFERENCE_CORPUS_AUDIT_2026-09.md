# Lane B — Realistic Business Reference Corpus Audit — 2026-09

Status: **supporting Evidence-coverage audit / no new Requirement / no new Gate / no production Canonical adoption**

Related:

- `plan/LANE_B_RQ002_BUSINESS_PROCEDURE_CANONICAL_TRACE_AUDIT_2026-09.md`
- `plan/RQ025_RQ026_ACCEPTANCE_TRACE.md`
- `plan/XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md`
- Lane C PR #482 / `plan/issue/000042-business-procedure-cross-view-reference.md`

## 1. 問い

Lane CのBusiness Procedure pre-ADR pressureでは、current `order_entry`への過適合を避けるためrealistic Referenceが論点になっている。

ここで、既存TEI repository内に、業務事実を新たに創作せずrealism / breadthの素材として再利用できるprovenance-backed Business corpusが存在するかを監査する。

## 2. 直接確認したRepository資産

### Reference Workspace

`reference/workspace/applications/` のcurrent mainには `order-entry/` だけが存在する。

したがって、current accepted Reference Workspace内に第二の既存Business Applicationを見つけて接続する経路はない。

### Business-system-design Reference

`reference/business-system-design/` のcurrent mainは次の`order-entry`系artifactだけを持つ。

```text
order-entry-business-procedure.json
order-entry-follow-up-procedure.json
order-entry-procedure-reference-set.json
order-entry-customer-edit-conflict.json
```

これはcurrent Business Procedure pressureの有効な小型Referenceだが、独立した別subject-area / subsystemのBusiness corpusではない。

### `reference/observed/`

`production.deployment-state.json` / `production.canary.deployment-state.json`はDeployment / Validation / Durable Runを再現するための固定fixtureである。

Business actor、Business event、manual / physical work、cross-subject-area Business handoffのauthoritative sourceではない。

### `reference/order_rollout.durable-run.m0.21.start.json`

名前に`order`を含むが、内容はdeployment runtimeのdurable runである。

代表actionは:

```text
database-gate
application-start
traffic-shift
rollout-health-gate
```

であり、受注・在庫・製造・出荷等のBusiness Procedure corpusとして再解釈しない。

### `reference/legacy-auth/`

Authentication / Authorizationのlegacy Referenceであり、Business actor / job contextを構成する業務manual corpusではない。

### `examples/`

current mainのexampleは`process-plugin` / `runtime-reattach`等のtechnical/runtime exampleであり、Business corpusではない。

## 3. B-F19 — realistic Business corpus coverage gap

### 判定

**Confirmed Evidence coverage gap.**

current TEI repository内には、少なくとも監査したReference / example surface上、`provenance-backed realistic cross-subject-area Business Reference`として使える第二のBusiness corpusを確認できない。

この結論は次を意味しない。

```text
D3 / D4 / O1 / H1 / R3 / S4-A が失敗した
Business Procedure viewpointが不要
Business Flowを今追加すべき
synthetic structural pressureが無価値
```

意味するのは次である。

```text
current structural / social-system pressure
  !=
provenance-backed realistic Business corpus breadth
```

## 4. Lane C Gate 15a / 15b / 15c分解の再監査

Lane CはB-F19後、Application数とrealismを混同しないため次を導入した。

```text
15a multi-Application structural pressure
    synthetic second Application

15b social-system realism pressure
    current production order_entry Reference

15c broader realistic pressure
    15a + 15bでminimum contract sufficiencyを説明できない具体的不足が出た場合だけ開く
```

この分解の方向は妥当である。

### 15aが示すもの

- Business Procedure != single Application child
- Pattern owner Applicationのexact resolution
- fixture Application != semantic owner

を構造的にPressureする。

syntheticでよい。

### 15bが示すもの

current `order_entry`だけでも次の社会システム境界が共存する。

```text
sales_core
  managed-here / read-write

master_data
  external-observed / read-only

human judgment
existing security.authorize reference
Application Command
separate Observation wording
manual / external continuation
```

これは「Application内部処理だけではないBusiness Procedure」をPressureするうえで有用である。

ただし15bは、次を新たに供給したわけではない。

```text
independent second Business corpus
multiple proven business jobs / responsibilities
new provenance-backed business event chain
real-world cross-subject-area handoff corpus
```

したがって15bを`social-system structural realism`として扱うことと、`realistic corpus breadth`を得たと扱うことを分ける。

## 5. B-F20 — pre-ADR contract sufficiencyとReference maturityを同じGate statusにしない

### 判定

**Confirmed claim/status distinction.**

Canonical contractを最小化するためのpre-ADR pressureと、業務システム設計として広いReference maturityを示すEvidenceは同じclaimではない。

```text
Pre-ADR contract sufficiency
  D3 / D4 / O1 / H1 / R3 / S4-A / C1 が
  current social-system + synthetic structural pressureで崩れないか

Reference maturity / breadth
  provenance-backed realistic cross-subject-area corpusでも
  同じcontractが過適合せず成立するか
```

Lane Cが15a + 15bでscoped production ADR candidateのminimum contract pressureを説明できる可能性は残す。

ただしその場合も次を同時に明示する。

```text
minimum contract pressure sufficient for scoped ADR candidate
  !=
realistic Reference maturity established
```

後者はB-F19のcoverage gapが残る限りunresolvedである。

### current status authority

以前はLane C内でGate 15表現とreadiness surfaceのdriftがあったが、current sourceでは解消されている。

```text
Gate 15
  15a structural
  15b social-system
  15c conditional broader pressure

ISSUE-000042 A-set
  Evidence / Pressure inventory

ISSUE operational completion checklist
  execution / consistency checklist

Pre-ADR ADR作成前Gate
  production Canonical contract ADR Ready / Not Readyの唯一の判定正本
```

したがってB-F19 coverage gapを、15cという「未達Gate」へ読み替えない。

15cは免除・waiveするGateではなく、15a + 15bまたは次のrealistic pressureから**具体的なcontract insufficiencyが出た場合にだけ開く追加pressure**である。

B-F19はそれとは独立して、Reference maturity / breadth claimを制限するEvidence coverage statusとして保持する。

## 6. synthetic pressureの位置づけ

Lane Cがtemp Workspace等で:

```text
second Application
manual warehouse coordination
external delivery continuation
cross-subject-area-like handoff
```

を組み合わせることは、D3 / D4 / O1 / H1 / R3 / S4-Aの構造試験として有効である。

ただし、そのfixtureを新しい業務事実の根拠なしに:

- realistic business truth
- actual Reference Model
- CONCEPTWARE再現
- domain-correct business process

へ昇格しない。

fixtureへ入れる根拠不十分なactor / trigger / sequence / condition / responsibility / state changeは`candidate / unresolved`として保持する。

## 7. realistic corpus breadthを得るEvidence source候補

Reference maturity / breadth claimを上げるには、少なくとも次のいずれかを必要とする。

1. provenanceと再利用条件が明確な既存Business Reference content
2. 独立に作成したBusiness caseで、業務上の前提・由来・仮定を明示したcorpus
3. user / domain expertから提供されたBusiness source
4. 公開資料をdesign pressureとして用い、転載ではなくTEI独自Referenceとして再構成し、fact / candidate / unresolvedを分離したcorpus

X-TEA / CONCEPTWAREの公開説明は設計pressureの由来として使えるが、artifactのライセンス・provenanceを確認せず内容をコピーしない。

## 8. realistic Referenceの最低pressure

既存E-002 / E-007 / E-008 / E-010 / E-012に照らし、単なるApplication数ではなく少なくとも次をPressureする。

```text
multiple responsibility / job contexts
explicit trigger / event
cross-subject-area / subsystem handoff
manual / physical / external work
multiple Function / Application use where actually present
multiple Data-state facts
Business ↔ Function ↔ Data cross-check
relation-only Business meaning change
source / semantic / runtime separation
```

これらをsyntheticに再現できることはstructural Evidenceである。

provenance-backed breadthまで主張するには、業務内容の由来を別途説明する。

## 9. Lane B / Lane C boundary

Lane B:

- B-F19をEvidence coverage gapとして保持
- B-F20でcontract sufficiencyとReference maturity claimを分離
- PAS-07に新Evidence IDを追加しない
- Gate B / Gate Cを変更しない
- realistic Evidenceが得られた後にRequirement / PAS claimを再監査

Lane C:

- structural fixture / checker / test-only pressureを所有
- 15a / 15b / 15cのstatus正本を一本化
- synthetic / current social-system / provenance-backed breadthを混同しない
- actual runner greenとsource-level implementationを分離

## 10. 現時点の判定

```text
small order_entry Business Reference
  exists

15a synthetic multi-Application structural pressure
  implemented / useful

15b current order_entry social-system pressure
  source-level implemented
  actual runner pending

provenance-backed realistic cross-subject-area Business corpus in current repo
  not found

B-F19 Reference maturity / breadth claim
  unresolved

15c conditional broader pressure
  not automatically open
  open only if concrete contract insufficiency appears

production Canonical ADR
  Not Ready while actual execution Evidence is missing
  realistic breadth claim must not be implied
```

actual Repository runner greenの未取得とは別のEvidence coverage事項として扱う。
