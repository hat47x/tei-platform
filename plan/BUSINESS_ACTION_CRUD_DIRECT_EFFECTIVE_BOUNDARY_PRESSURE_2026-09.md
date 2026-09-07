# Business Action × CRUD Direct / Effective Boundary Pressure — 2026-09

Status: **Lane C / predecessor-design input / future guardrail / no production contract adoption**

## 1. 目的

Current TEIではBusiness Action × CRUDのRを一部derived projectionできる一方、C/U/D attributionはaccepted Function -> mutation authority link不足により未解決である。

将来そのlinkを追加する場合、CRUDを単純な一枚のcellへ潰す前に、Action / Function自身のoperationと、Function callを通じて影響するoperationを区別する必要があるかを確認する。

本書は、過去のSemantic Platform / X-TEA vitality workで得たdirect/effective CUD設計を、current TEIへschemaとして移植するのではなく**失ってはいけないreview pressure**として保持する。

## 2. Prior design evidence

Prior Semantic Platform Reference Model workではFunction viewpointを:

```text
Action + View + CRUD/read/call cross-reference
```

として再構成し、Subsystem reviewでは:

```text
owner + direct/effective CUD + exposed Action boundary
```

を用いた。

MTO `sales_order` Reference pressureでは、local Data pageにrelevant Actions / Views / Proceduresを表示しつつ、direct CUDとeffective CUDを別表示した。

意味は:

```text
direct CUD
  = Action自身が行うmutation

effective CUD
  = same-Subsystem internal Action callを通じて到達するmutation
```

である。

Cross-Subsystem published Action boundaryではeffective CUD inheritanceを停止し、caller側へcallee内部mutationを直接所有させない。

## 3. Why this matters to current TEI

Current T-GAP-08では将来のoperation attribution authority候補として:

```text
Function/Data-I/O relation
Application + Capability realization -> mutation authority
Database mutation contract + explicit Function binding
```

を比較している。

どの案を採る場合でも、次を一つのCRUD cellへ無差別に畳むと設計review能力を失う。

```text
Function itself mutates Data
Function calls another Function that mutates Data
Function calls a published cross-boundary Function whose internal mutation is encapsulated
```

特にSubsystem / ownership reviewでは:

```text
who directly mutates owned data?
who merely depends on another subsystem's published action?
```

を区別できる必要がある。

## 4. Current TEI does not yet implement this

Current TEI Lane Cには:

- accepted C/U/D attribution relationがない
- qualifying TEI-original C/U/D Business Actionがまだない
- current Business Procedure ReferenceにはFunction call graphをoperation projectionへ合成するcontractがない

したがって今:

```text
DirectCRUD
EffectiveCRUD
```

というproduction field / type / matrix contractを追加しない。

## 5. Future semantic rule candidate

C/U/D attributionが採用される場合のguardrail候補:

```text
Direct operation
  authority = Function / realization自身のaccepted mutation relation

Effective operation
  authority = explicit Function call graph
              + callee direct/effective operation
              + propagation boundary policy
```

Names, Business Procedure adjacency, UI navigation, SQL textからcall/effectを推測しない。

### Same boundary

Same semantic ownership / subsystem boundaryでinternal callがacceptedされる場合:

```text
caller effective CUD
  may include callee direct/effective CUD
```

### Published cross-boundary Action

Cross-subsystem published Action / contract boundaryでは:

```text
caller direct CUD
  does not inherit callee mutation

caller effective dependency
  may record call / contract dependence

callee internal CUD
  remains callee-owned
```

とする方向がpredecessor designとX-TEA Subsystem pressureに整合する。

これはcurrent production ruleではなくfuture comparison criterionである。

## 6. X-TEA relation

E-007ではexternal-table C/U/DをSubsystem smellとして扱うHistorical pressureを記録している。

E-016では`TaskFunctionIO`にexplicit `OpC / OpR / OpU / OpD`があり、CRUD matrixがFunction/Table I/O informationからderiveされることを確認した。

Direct/effective distinctionを加えると、current translation pressureは:

```text
Function direct Data-I/O operation
        +
Function call dependency
        +
Subsystem / published-boundary semantics
        ->
review projection of direct / effective CRUD
```

となる。

Business Procedureはこのoperation graphのSOTにはしない。

## 7. Failure modes this guardrail prevents

### F1 — callerをdirect mutatorと誤表示

A calls B, B updates Data.

Flattened matrix:

```text
A = U Data
```

だけでは、A自身が直接Dataを更新するように見える。

### F2 — cross-subsystem encapsulation loss

Sales Action calls Inventory published Action.
Inventory internals update Inventory Data.

Callerへeffective Uを無条件伝播すると:

```text
Sales appears to mutate Inventory Data
```

となり、published contract boundaryが失われる。

### F3 — ownership diagnostic false positive

Effective CUDをdirect CUDとして扱うと、合法なpublished Action利用をcross-subsystem direct-write violationとして誤診断し得る。

### F4 — impact analysis false negative

逆にcall propagationを完全に捨てると、callee mutation changeがcaller Business Procedure / Function reviewへ与えるimpactを見落とす。

## 8. Future pressure requirements

T-GAP-08をreopenしたとき、operation model候補は少なくとも次をPressureする。

1. direct mutationをexactに識別できる。
2. explicit Function callを通じたeffective effectをderivedできる。
3. same-boundary propagationとcross-boundary encapsulationを分けられる。
4. published Action内部のData mutationをcaller ownershipへ移さない。
5. direct/effective distinctionをBusiness Procedureへ重複authorしない。
6. Function call changeだけでeffective projectionが追随する。
7. callee implementation changeでdirect semantic operationが変わらない場合、不要なBusiness meaning changeを出さない。
8. external API / physical work等のnon-CRUD effectを無理にCUD propagationへ含めない。

## 9. Current decision

```text
Direct/effective CUD distinction
  = future guardrailとして保持

Production schema
  = not adopted

Current partial R projection
  = unchanged

Current C/U/D
  = unresolved
```

Qualifying TEI-original C/U/D exampleとexplicit Function-call/effect authorityが揃うまではimplementationへ進めない。
