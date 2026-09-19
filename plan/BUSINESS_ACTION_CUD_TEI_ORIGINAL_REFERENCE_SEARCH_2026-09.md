# Business Action × C/U/D TEI-original Reference Search — 2026-09

Status: **Lane C / negative Evidence / no new production contract**

## 1. 目的

`BUSINESS_ACTION_CRUD_2026-09.md`（Part 2、旧`CRUD_OPERATION_GAP_AUDIT`）では、current `order_entry`からBusiness Action × CRUDをderived projectionした結果、Rの一部はexactに導出できる一方、C/U/Dはaccepted meaning不足のため安全に帰属できなかった。

本監査は、その次のEvidence pressureとして、Repository内の**TEI-original Reference**にCreate / Delete / non-state Updateのqualifying exampleがすでに存在するかを確認する。

ここでqualifying exampleが見つからない場合、それを埋めるためだけにsynthetic Capability / Data / Procedureを追加しない。

## 2. Reference規律

`reference/README.md`の方針に従い、Reference artifactは「生成できるから」「テストに便利だから」追加するのではなく、設計境界を長期的に再現する共有基準として必要なものだけを置く。

したがって:

```text
CRUD matrixを埋めたい
  -> Create/Delete Capabilityを作る
```

という順序は禁止する。

正しい順序は:

```text
realistic Business / Application / Data requirementが独立に存在
  -> accepted Function / Capability / Database authorityが成立
  -> operation attribution gapをPressure
  -> 必要ならcontractを再検討
```

である。

## 3. Current TEI-original corpus inventory

### 3.1 Semantic namespaces

Current Reference Workspaceのdomain Semantic namespaceは:

```text
customer
sales
security
```

である。

`sales`のBusiness Capabilityはcurrent primary Referenceでは:

```text
sales.order.submit
```

が中心であり、Create / Delete / non-state Updateをexplicitに表す別Capabilityは存在しない。

`security`のauthenticate / authorizeはsecurity-side authorityであり、Business Data CRUDへ読み替えない。

### 3.2 Business-system-design Reference

Current `reference/business-system-design/`はorder-entry系のProcedure / follow-up / Handoff / intentional access-conflict probeで構成される。

Qualifyingな第二Business domain Procedureはまだ存在しない。

`order-entry-customer-edit-conflict.json`はread-only masterに対するwrite要求を意図的に作る**defect probe**である。

これは:

```text
accepted customer update Business Action
```

ではない。

このprobeをUpdate Referenceへ昇格すると、検出すべきarchitecture violationをaccepted meaningに変えてしまうためrejectする。

### 3.3 Database Contracts

Current relevant contracts:

```text
sales_core.order
  read sales.order
  write sales.order.status

sales_core.order_line
  read sales.order_line

master_data.customer
  read customer.customer
```

`write sales.order.status`はDatabase write authorityだが、C/U/D classificationやBusiness Action attributionを所有しない。

したがって既存Contract inventoryから新しいC/U/D Business Action exampleを作らない。

### 3.4 Deployment / migration actionsはBusiness CRUDではない

Reference Worldにはdeployment executor actionとして:

```text
application-start
application-retire
stop-accepting-work
traffic-shift
```

等がある。またDatabase migrationはschema lifecycle上CREATE / ALTER / DROP相当のphysical operationを持ち得る。

しかしこれらは:

```text
Deployment / Infrastructure / Database schema lifecycle
```

のoperationであり、Business Action × Business Data CRUDではない。

名前にcreate / retire / delete相当の語があってもBusiness CRUDへ流用しない。

## 4. Current search result

2026-09時点のcurrent Repository / Reference corpusでは、次のqualifying exampleは見つかっていない。

```text
Create Business Action        : 0
Delete Business Action        : 0
non-state Update Business Action : 0
```

ここでの`0`は「TEIにCreate/Delete/Updateが不要」という意味ではない。

意味は:

> current accepted Reference authorityから、Business ActionへC/U/Dを安全に帰属できる独立exampleがまだない。

である。

## 5. Qualifying exampleの採用条件

次のReferenceをC/U/D pressureへ使うには、少なくとも以下を満たす。

### Q1 — CRUD検証以外の独立したBusiness reason

そのFunction / Procedure / Dataが、CRUD matrixを埋めるためではなくBusiness scenarioとして自然に必要であること。

### Q2 — Business ActionがFunctionから独立して説明できる

目的、actor/job context、trigger/event、manual/external workとの関係を説明でき、単なる「DB INSERTを実行するFunction」ではないこと。

### Q3 — exact Semantic Data / resource identity

対象Data / Relation / Attributeがaccepted Semantic identityとして存在すること。

### Q4 — exact Application Function / Capability identity

Business Actionが利用するFunction / Capabilityを名称推測なしに特定できること。

### Q5 — effect / mutation attributionのaccepted source

少なくとも「このFunction / Capabilityがこのresource mutationに関係する」とaccepted authorityから説明できること。

SQL本文、function名、output payload、adjacent lifecycleだけからeffectを推測しない。

### Q6 — C/U/Dの区別がdomain meaningとして妥当

Create / Update / Deleteが本当にBusiness review上意味を持つこと。

例えばphysical-world action、message publication、external SaaS requestを無理にCRUDへ押し込まない。

### Q7 — non-state UpdateはLifecycle transitionと分離

non-state Update pressureでは、`status pending -> submitted`のようなLifecycle transitionとは別のmutationを選ぶ。

Lifecycle transitionとgeneric Updateを一例で混ぜない。

### Q8 — Deployment / schema lifecycleを混入させない

Application start/retire、database migration、schema create/dropはBusiness CRUD exampleとして採用しない。

## 6. Candidate audit

### Candidate A — customer maintenance

一見するとcustomer masterの登録・変更・削除はC/U/D pressureに適する。

しかしcurrent Referenceでは:

```text
master_data
  external-observed

order_entry master datasource
  read
```

という境界が重要なaccepted architectureである。

そのためorder-entry内にcustomer maintenanceを追加すると、current external-master/read-only pressureを壊す。

別のowner systemとしてcustomer maintenance Applicationを設計するBusiness reasonが独立に生じた場合のみ再検討する。

Current: **do not add for CRUD pressure**。

### Candidate B — order line create/update/delete

受注明細はdomain上C/U/Dを想像しやすい。

しかしcurrent Referenceでは`OrderLine`はSemantic Relation / header-detail read pressureとして導入され、Create/Delete Business ActionやCapabilityはacceptedされていない。

CRUD検証のためだけに`add_line` / `remove_line`等を作るとtest-driven domain inventionになる。

Current: **defer until independent business requirement exists**。

### Candidate C — manufacturing / inventory second corpus

physical receipt / production / stock / shipmentはoperation pressureが強い。

しかしcurrent second-corpusはprovenance-backed outlineであり、exact TEI-original Data / Pattern / Database authorityをまだ持たない。

これをCreate/Update exampleへsynthetic化しない。

Current: **not a qualifying TEI-original fixture yet**。

## 7. Current answer

```text
TEI-original C/U/D qualifying Reference
  = none yet

Therefore
  no generic CRUD/effect contract adoption
  no synthetic C/U/D fixture
  no Business Procedure CRUD annotation
```

Current partial R projectionを維持する。

C/U/D gapは次のrealistic Referenceが自然に生じた時点で再評価する。

## 8. Reopen condition

次のいずれかが起きた場合だけ本監査をreopenする。

1. New TEI-original Business ReferenceがCreate / Delete / non-state Updateを独立要件として必要とする。
2. Existing `order_entry`に、CRUD pressureとは無関係なaccepted business requirementとして新mutation Functionが追加される。
3. Provider / Capability / Database effect境界でexplicit mutation attributionが別作業から導入される。
4. X-TEA fallback / realistic Reference comparisonから、現行TEIの分離構造ではoperation attributionを失う具体的失敗が追加で見つかる。

## 9. Productionへ今入れない

- Create / Update / Delete専用Pattern kind
- generic Capability Effect
- Business Procedure CRUD letter
- synthetic customer-maintenance Application
- synthetic order-line mutation Capability
- manufacturing/inventory synthetic CRUD fixture
- deployment action -> Business CRUD変換
- schema migration -> Business CRUD変換

## 10. Pre-ADR implication

Current negative EvidenceはD4を弱めない。

むしろ:

```text
Business Procedureを小さく保つ
  +
structured authorityからderived matrixを作る
  +
不足operationはunknownのまま残す
```

方針を支持する。

C/U/D gapが今後反復した場合も、最初にBusiness Procedureを太らせるのではなく、`Capability / realization / Database effect`のどこにaccepted linkが必要かを比較する。
