# BP-01 Prepare Operation Ambiguity Pressure — 2026-09

Status: **Lane C / wording-to-authority pressure / unresolved / no production contract adoption**

## 1. 目的

Current Business Procedure `order-entry.order-handling` のBP-01は:

```text
inspect or prepare the selected order
```

と記述され、Procedure purposeにも:

```text
inspect and prepare an order
```

が含まれる。

一方、BP-01が利用するApplication Pattern `order_entry.order_record`はcurrent contract上:

```text
record
  data sales.order
```

だけを表し、read / update / create / delete operationを所有しない。

本監査は、`prepare`というBusiness wordingをPersistence Updateへ暗黙変換せず、この語が実際に何を意味するのかをEvidence待ちとして分離する。

## 2. Current accepted facts

### Business side

```text
BP-01
  Business Action = inspect or prepare the selected order
  realization     = application-use
  Pattern         = order_entry.order_record
```

### Application Pattern side

`record`は「ApplicationがどのSemantic Dataを一件として扱うか」というintentである。

Current Pattern contractは:

```text
Data required
Relation not allowed
Query not allowed
Capability not allowed
Datasource not allowed
```

であり、persistence operationは宣言しない。

### Database side

`sales_core.order`には:

```text
read sales.order
write sales.order.status
```

がある。

しかしBP-01 / `order_entry.order_record`とこのwriteを結ぶaccepted relationはない。

## 3. 禁止する推論

次を行わない。

```text
"prepare"
  -> U sales.order

"edit" / "update"というBusiness wording
  -> U

record Pattern
  -> R/U

sales_core.order has Writes[]
  -> BP-01 U
```

Natural-language verbはBusiness meaningの重要なreview inputだが、CRUD operation authorityではない。

`business_action_crud_wording_independence_pressure_test.go`では、BP-01 wordingを意図的に:

```text
edit, correct, and update the selected order before submission
```

へ変更しても、refsが同じならCRUD projectionがbaselineから変化しないことをPressureする。

さらにpurpose / actionへ`create` / `delete`という語を入れてもC/D cellを生成しない。

## 4. 三つの解釈候補

### P0 — inspect-only meaning

実際のcurrent Application Functionが閲覧だけなら、Business Action wordingの`prepare`が強すぎる可能性がある。

候補:

```text
inspect the selected order
```

Pros:
- current `record` intentとの距離が小さい
- mutation authority gapを作らない

Risk:
- original Business intentに存在した準備作業を失う可能性

Current: **not selected**。

### P1 — preparation is manual / external Business work

`prepare`が人間による判断、別システム、電話・メール、資料整理等を含むなら、Application Functionへ押し込むべきではない。

候補:

```text
BP-01a inspect selected order using order_record
BP-01b prepare surrounding order information / work manually or externally
```

Pros:
- RQ-025のBusiness Action / Function separationと一致
- fake mutation Functionを作らない

Risk:
- 現時点のReferenceにない具体的Business sequenceを発明しやすい

Current: **plausible but evidence insufficient**。

### P2 — preparation includes actual Application mutation

もしsales operatorが`order_entry`上でorder内容を編集・保存することがaccepted Business requirementなら、BP-01はrealistic non-state Update pressureになり得る。

ただし必要なのはBusiness wordingからUを付けることではない。

少なくとも:

```text
Business Action
  -> explicit Application Function / Capability
  -> accepted mutation/effect authority
  -> exact Semantic Data / Attribute relation
```

のどこかを設計として閉じる必要がある。

Current `order_record` Patternだけでは不足する。

Current: **strong future pressure if independently confirmed; not yet accepted**。

## 5. X-TEA fallbackとの関係

XeadModelerではCRUD markはBusiness wordingから推測されず、Function/Table I/Oの`OpC / OpR / OpU / OpD`というstructured informationを基にmatrixへ集約される。

したがってBP-01でP2が成立しても、解決策は:

```text
Business Procedureに "U" と書く
```

ではない。

将来の候補は:

```text
Function / Data-I/O operation relation
or
Capability realization -> mutation authority
or
Database mutation contract + explicit Function binding
```

等のFunction-side / realization-side boundaryで比較する。

## 6. Current decision

BP-01について現時点では:

```text
Business wording suggests preparation relevance
Application Pattern proves one-record intent
CRUD Update authority is not proven
```

とする。

したがって:

- BP-01をU cellへ昇格しない
- wordingをinspect-onlyへ勝手に変更しない
- manual stepを勝手に追加しない
- update CapabilityをCRUD pressure目的で作らない
- `order_record` Patternへgeneric update fieldを追加しない

## 7. Reopen conditions

次のいずれかが得られた場合に再評価する。

1. prior/authoritative business-system-design sourceで、order preparationがApplication内のedit/saveを含むと明示される。
2. independent product/business requirementとしてorder editingが採用される。
3. Application Function / Capability / Database mutationのaccepted linkが別作業から導入される。
4. realistic TEI-original Referenceで同じnon-state Update attribution gapが反復する。

## 8. Design implication

BP-01は、current TEI-original domain内で最も近いnon-state Update **pressure candidate**ではある。

しかしcandidateとaccepted exampleを分離する。

```text
wording pressure
  != operation authority

possible Update
  != accepted U cell
```

この分離により、CRUD matrix completenessのためにdomain meaningを発明せず、将来P2が本当に成立した時だけ適切なFunction/effect boundaryを設計できる。
