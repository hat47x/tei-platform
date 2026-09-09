# Business Action × CRUD Operation Gap Audit — 2026-09

Status: **Lane C / pre-contract gap audit / no production schema adoption**

## 1. 目的

`BUSINESS_ACTION_CRUD_PROJECTION_PRESSURE_2026-09.md`で、current TEI structured meaningからBusiness Action × CRUDをどこまで機械導出できるかをPressureした。

結果、current `order_entry`では一部のRをexactに導出できる一方、C/U/DはBusiness Actionへ安全に帰属できない。

本監査は、この不足を「CRUD fieldがない」と単純化せず、どの層のaccepted meaningが不足しているのかを分解する。

## 2. Current operation authority inventory

### 2.1 Application Pattern

Current Pattern kindは:

```text
browse
record
reference
command
header-detail
```

である。

Operationとして読める強さは同じではない。

```text
browse
  Query executionによるread intent

reference
  明示的read intent
  write-only datasourceはreject

header-detail
  Pattern単体ではoperationなし
  explicit Contract Query realizationがある場合にreadをderive可能

record
  one-record intentのみ
  read/updateを意味しない

command
  Capability invocation intentのみ
  create/update/deleteを意味しない
```

Pattern kindからgeneric CRUDを生成しない。

### 2.2 Database Contract

Current Database Contractは:

```text
Reads[]
Writes[]
Queries[]
```

を持つ。

Current Reference:

```text
sales_core.order
  read sales.order
  write sales.order.status

sales_core.order_line
  read sales.order_line

master_data.customer
  read customer.customer
```

`Reads[]`はread authorityとして使える。

`Writes[]`はwrite authorityを示すが:

```text
C
U
D
```

を区別しない。

さらに、Database Contract writeとBusiness Action / Capabilityを結ぶaccepted semantic relationはcurrent modelにない。

### 2.3 Semantic Capability

Capabilityは:

```text
input
output / named outcome
```

を持つが、persistence effectを宣言しない。

Current:

```text
sales.order.submit
  input  sales.order
  output sales.order
```

から:

```text
U sales.order
```

を導出しない。

Input/output Dataはpayload contractであり、persistence mutation contractではない。

### 2.4 Capability realization

Current `sales.order.submit` realizationは:

```text
realize sales.order.submit with custom go
```

である。

Custom Go realization identityは実装選択を表すだけで、Database Contract write / mutation targetを宣言しない。

したがってrealization kindからCRUDを推測しない。

## 3. Business Lifecycle proposalとの関係

Business Lifecycle v1設計（評価待ち）は:

```text
sales.order.status
  pending --[sales.order.submit]--> submitted
```

というBusiness transitionを検討している。

これはC/U/D不足への重要なinputだが、Lifecycle transitionそのものをCRUD Uへ変換してはならない。

理由:

1. LifecycleはBusiness state meaningを所有する。
2. physical state code / Persistence Mappingは別realizationである。
3. Command Business ResultとObservation readは分離される。
4. provider上の条件付き更新は別Evidenceとして扱われる。
5. Lifecycle設計自身が`on-entry` / `on-exit` side-effect hookを導入しない。

したがって:

```text
Business transition
  != persistence mutation
```

である。

将来Lifecycleが採用されても、それだけでCRUD Uを自動生成しない。

## 4. Current missing link

BP-07を例にすると、current accepted meaningは:

```text
Business Action BP-07
  -> Application Pattern order_entry.submit_order
     -> Capability sales.order.submit
        -> Custom Go realization
```

Database側には別に:

```text
sales_core.order
  write sales.order.status
```

がある。

しかし次のedgeがない。

```text
sales.order.submit
  -> mutates sales.order.status
```

または:

```text
order_entry.submit_order
  -> realizes Database Contract write sales_core.order/status
```

このedgeがない状態で、名称・Lifecycle候補・隣接Data・Persistence MappingからBP-07=Uを作ると、複数SOT / 推測補完になる。

## 5. Option audit

### M0 — partial CRUD projectionを維持

```text
RだけEvidence-backedに埋める
C/U/Dはunknownのまま
```

Pros:

- current authorityを壊さない
- P11 unresolved remains unresolvedに一致
- 新しいCore contractを先回りしない

Cons:

- X-TEA型CRUD matrixは部分的
- write impact reviewが弱い

Current: **採用中のPressure方針**。

### M1 — Capabilityへgeneric CRUD/effect field

例:

```text
capability sales.order.submit {
  effect update sales.order.status
}
```

Pros:

- Business Function / CapabilityからData mutationを明示できる

Risks:

- persistence-neutral CapabilityへDB風operation vocabularyを混ぜる
- external API / message / physical actionをCRUDへ押し込める
- Lifecycle / Business Result / provider effectとのownershipが曖昧になる

Current: **defer**。

### M2 — Capability / PatternとDatabase Contract writeのexplicit realization link

例:

```text
Capability
  -> Database Contract write target
```

Pros:

- Database write authorityとのcross-view linkを作れる
- operation projectionを構造化できる

Risks:

- Capabilityの実現方式をDatabaseへ過度に固定する
- Custom Go / Plugin / API realizationがDatabaseを使わない場合に不自然
- one Capabilityが複数resourceへeffectを持つ場合の意味が複雑

Current: **candidate pressure only / not adopted**。

### M3 — Lifecycle TransitionとPersistence Mutation Bindingを分離

```text
Business transition
  -> persistence realization binding
```

Pros:

- Business state meaningとphysical mutationを分けられる
- state transitionとconditional updateの対応を明示できる

Risks:

- Lifecycle未採用段階で先行設計になる
- CRUDすべてがLifecycleではない
- Create/Delete/non-state mutationを説明できない

Current: **future input / not adopted**。

### M4 — Business Procedure / Function側へCRUD letterをauthor

例:

```text
BP-07 U sales.order
```

Pros:

- matrixを直接埋められる

Risks:

- Database / Capability meaningの第二SOT
- implementation変更でdrift
- X-TEAの「structured definitionからmatrixをderiveする」方向に逆行

Current: **reject**。

## 6. X-TEAとの比較

X-TEAではFunction definitionがUI / Table I/O等を構造化して持ち、CRUD matrixをそこから機械導出する。

TEIでは意味を:

```text
Business Action
Application Pattern
Semantic Capability
Database Contract
Realization
```

へ分離している。

この分離自体は維持すべきであり、X-TEAのCRUD列を再現するためだけに一つのFunction objectへ戻さない。

必要なのは:

> TEIの分離されたauthority間に、operation attributionに十分なaccepted linkが本当に必要か

をrealistic Referenceで確認すること。

## 7. 次のEvidence pressure

C/U/D contractを新設する前に、次の順で確認する。

1. current `order_entry`でpartial CRUD R projectionをactual testでgreenにする。
2. `sales.order.submit`についてactual execution / Lifecycle / Database writeのどこまでがaccepted meaningとして既に接続されているか確認する。
3. next TEI-original realistic ReferenceでCreate / Delete / non-state Updateの少なくとも一例を探す。
4. 同じgapが複数domainで反復するか確認する。
5. 反復した場合にM1/M2/M3を比較する。

一つのsubmit例だけでgeneric mutation modelを決めない。

## 8. Pre-ADR判断

Current answer:

```text
Business Action × CRUD
  R: partial exact projection possible
  C/U/D: operation attribution contract insufficient
```

この不足はproduction Business Procedure schemaの不足とは限らない。

むしろ:

```text
Capability / realization / Database effect boundary
```

側のpressureである可能性が高い。

したがってLane CからBusiness ProcedureへCRUD fieldを追加しない。

## 9. Productionへ今入れない

- Business Procedure CRUD annotations
- Pattern generic CRUD field
- Capability generic Effect / CRUD field
- Database Contract write -> U自動変換
- Lifecycle transition -> U自動変換
- SQL AST / SQL本文解析によるCRUD推測
- generic mutation edge
- CRUD matrix Canonical authority

## 10. Verification status

`business_action_crud_projection_pressure_test.go`をRepositoryへ追加したが、actual Repository runnerでの:

```text
gofmt
go vet
go test
checker
```

は未取得である。

したがって本監査もpre-contract Evidenceであり、production contract採用根拠ではない。
