# Business Procedure Canonical Meaning Pressure — 2026-09

Status: Test-only design pressure / no Core adoption

## 目的

Business Procedureのfile-native accepted sourceについて、`BusinessProcedureIdentity{id}`だけをCanonical側へ置けば十分かを再検証する。

前段のidentity shell pressureでは、次は成立した。

- path-independent Procedure identity
- Source Index target候補
- file move / logical rename分離
- duplicate ID fail-closed
- review / handoff / Shared Acceptance target

しかし、formal architectureと照合するとidentityだけでは不足する可能性がある。

## 1. 反証: Business-only accepted changeがCanonical Worldから見えない

現在のfile-native source pressureでは、BP-04のようなmanual Business Action文言をAcceptしても`ir.World`は変化しない。

```text
before
  Procedure ID = order-entry.order-handling
  BP-04 = decide whether the order is ready ...

Accept
  BP-04 = decide whether surrounding business circumstances permit submission

after
  Procedure ID = order-entry.order-handling
  ir.World = unchanged
```

identity shellを追加しても、

```text
BusinessProcedureIdentity{id}
```

はbefore / afterで同じである。

したがってidentity-onlyでは、Business-only accepted semantic changeをCanonical diff / impactから観測できない。

## 2. formal architectureとの緊張

`docs/architecture.md`は次を基線とする。

```text
Authoring source
   ↓ parse / resolve / accept
Canonical Semantic meaning / IR
```

また三視点は別々の正本ではなく、同じaccepted meaningを見る。

`docs/business-system-design.md`も、Business Procedureを

- purpose
- trigger / event
- actor / job context
- Business Action
- manual / external work
- system interaction reference
- judgment / continuation

を含むBusiness側の視点として扱う。

この意味を正式にAcceptした後もCanonical側がIDしか持たない場合、Canonical layerはBusiness viewpointのaccepted meaningを十分に保持していない。

## 3. RQ-002だけの問題ではない

前段では残件を主に

```text
RQ-002: authoring source -> Canonical subject
```

と捉えていた。

identity shellはこのtrace targetを作れる。

しかし新しいpressureでは、さらに次が残る。

- Business-only Semantic Diff
- Business change -> Data / Function impact
- Studio / AIによるaccepted Business meaningのread-only探索
- reload後のCanonical Conformance
- 000039 change graph上のmeaning change

よって「Canonical subjectがある」だけでは十分ではない。

## 4. Option D — thin normalized Canonical Business Procedure meaning

full Workflow modelではなく、accepted Business residual meaningだけを正規化する案を比較対象へ追加する。

test-only probe:

```text
BusinessProcedureMeaning
  id
  purpose
  trigger
  actor
  ordered steps

BusinessProcedureStep
  id
  businessAction
  realization
  explicit Application Pattern refs
  explicit Data refs
  explicit Relation refs
  explicit Validation refs
  explicit Capability refs
```

### 入れないもの

- source path
- fingerprint
- source range
- schemaVersion / artifactRole
- experimental identityStatus
- Web / Wails / concrete Target UI
- runtime branch
- timer
- retry
- queue
- saga
- lease
- scheduler
- runtime state
- Authorization Role

Business ProcedureをCanonical化することとWorkflow Engineを導入することを分離する。

## 5. Pattern-owned meaningをコピーしない

Business Procedure stepが

```text
order_entry.customer_reference
```

を参照する場合、そのPatternが所有する

- Data
- Relation
- Query
- Capability
- Datasource

をProcedureへ重複保存しない。

Canonical Business meaningに保持するのはexplicit referenceである。

```text
Business Action
  -> Application Pattern ID
```

Pattern-owned meaningは既存Canonical Application側から解決する。

一方、Business側がPatternからは導けないValidation等を明示した場合は、そのexplicit referenceを保持する。

## 6. manual workを残す

BP-04のようなmanual / unresolved stepは、Application FunctionやCapabilityなしでCanonical Business meaningへ残る必要がある。

```text
Business Action
  realization = manual-unresolved
  ApplicationPatterns = []
  Capabilities = []
```

fake Capabilityを生成しない。

## 7. source normalization

Canonical meaningはsource formattingと同一視しない。

- Procedure step順は意味があるため保持
- set-likeなexplicit reference配列はsortして正規化
- file pathは含めない
- pure file moveではmeaning不変
- Business wording changeではmeaning change
- logical Procedure renameではidentity change

## 8. test-only Evidence

`tools/reference-business-procedure-check/canonical_meaning_pressure_test.go`

主なPressure:

1. Procedure / Step field上限をreflectionで固定
2. Business-only wording changeがmeaning差になる
3. pure file moveではmeaning不変
4. manual BP-04をfake Functionなしで保持
5. Pattern-owned meaningをコピーしない
6. explicit Validation referenceを保持
7. set-like reference orderをnormalizationで吸収

actual Repository runnerがcheckout / Go testへ到達するまでは、これらはtest-only specificationでありgreen Evidenceとは数えない。

## 9. identity shellとの関係

Option Dはidentity shellを否定するものではない。

```text
identity shell
  = thin meaningのID部分
```

とみなせる。

ただしIDだけをproductionへ先行導入すると、Business-only semantic changeがCanonical layerから欠落した状態を制度化する危険がある。

したがってproduction integrationは、identity-onlyとthin meaningを別々に入れるのではなく、同じReference pressureで比較してから決める。

## 10. 現時点の判断

### 撤回する仮説

> RQ-002のCanonical trace targetだけが最後の未解決点であり、ID-only shellが最小十分である

これは強すぎた。

### 更新後の仮説

Business Procedureのproduction Canonical contractが必要なら、最小単位はIDだけではなく、**cross-view / diff / impactに必要なBusiness residual meaningを保持する薄いnormalized model**である可能性が高い。

ただし、まだproduction `ir.World`へ追加しない。

## 11. 次の判断条件

Option DをCore候補にする前に少なくとも次を確認する。

1. test-only probeがactual Repository testでgreen
2. Business-only changeをCanonical diffとして明確に説明できる
3. Pattern-owned meaningを重複しない
4. manual / external meaningを失わない
5. actorをAuthorization Roleへ変換しない
6. runtime Workflow semanticsが侵入しない
7. handoffをどうnormalized meaningへ置くかを別Pressureで確認する
8. current Referenceだけで過度に単一Application前提を固定しない

特にhandoffは現在`order-entry-procedure-reference-set.json`側にあるため、Procedure bodyだけをCanonical化して設計完了としない。
