# Lane B — Cross-View Reference Representation Option Audit — 2026-09

Status: **Lane B / supporting pre-ADR option audit / test-only recommendation / no production schema adoption**

## 1. 目的

B-F23 / B-F24で確認した次の不足を、最小のReference pressureで表現する候補を比較する。

```text
Business Action may reference an Application Pattern
  without being realized by that Pattern
```

Primary example:

```text
BP-06 Authorization concern
  supporting semantic authority = security.authorize
  protected application target  = order_entry.submit_order
  direct Application Function realization = none
```

本監査はproduction schemaを決定しない。

## 2. 既存表現の制約

current Lane C fixtureは概ね:

```text
applicationPatterns[]
data[]
relations[]
validations[]
capabilities[]
```

をStepへ持つ。

checkerは:

```text
NoApplicationFunction = len(applicationPatterns) == 0
```

とするため、`applicationPatterns`は事実上**direct Application Function realization refs**である。

この意味を壊さずにprotected targetを足す必要がある。

## 3. 比較軸

候補を次で比較する。

1. exact identity resolution
2. target rename / missingをfail closedできる
3. `Business Action != Application Function`を壊さない
4. Pattern-owned meaningをBusiness側へ複製しない
5. Authorization policyをBusiness側へ複製しない
6. normalized semantic diffでtarget changeを観測できる
7. generic relation taxonomyを一例から作らない
8. 将来Evidenceが増えたときに拡張可能

## 4. O0 — residual Business wordingだけ

```text
BP-06
  businessAction = "... submit application action"
  capabilities   = [security.authorize]
```

### 長所

- 既存schemaのまま
- 新しい概念なし
- Business wordingは保持される

### 弱点

- `order_entry.submit_order` rename / removalをexact traceできない
- target変更をmachine-readable cross-view diffとして観測できない
- wordingからtargetを推測するとfail-closed原則を破る
- Legacy policyとの差異をcross-viewで比較しにくい

### 判定

Baselineとして残すが、B-F23のpressureを十分には満たさない。

## 5. O1 — narrow field-specific target refs

Test-only shape例:

```text
step
  applicationPatterns[]        // direct realization only
  applicationPatternTargets[]  // referenced/concerned target only
```

BP-06:

```text
applicationPatterns = []
applicationPatternTargets = [order_entry.submit_order]
capabilities = [security.authorize]
```

### 長所

- current `applicationPatterns`の意味を変えない
- `NoApplicationFunction = true`を自然に維持できる
- protected targetをexact resolutionできる
- Pattern target changeをBusiness cross-view changeとして観測できる
- generic `role` enumやuniversal relationを導入しない
- implementation pressureが小さい

### 弱点

- 将来Data / Validation / Capabilityにもtarget/support roleが必要になるとfieldが増殖し得る
- `applicationPatternTargets`という名称がproduction語彙として適切とは限らない

### 判定

**current test-only primary candidate**。

理由は、この一例で必要な差だけを明示し、generic relationを先回りしないため。

## 6. O2 — generic role-qualified cross-view refs

Test-only conceptual shape:

```text
refs[]
  kind   = application-pattern | capability | validation | data | relation
  target = exact identity
  role   = realizes | supports | concerns | ...
```

### 長所

- role差を一貫して表現できる
- BP-03 / BP-05 Validation role差にも適用できる
- 将来のcross-view projectionに統一形を与えられる

### 弱点

- current one/two examplesから`role` taxonomyを設計することになる
- `supports / concerns / checks / protects`等の語彙境界が未証明
- generic Core Edge / relationへの拡張圧力を生みやすい
- existing typed fieldsの意味を再構成する範囲が大きい

### 判定

**defer**。

複数realistic Referenceで異なるref rolesが反復し、O1のfield-specific approachが破綻するEvidenceが出るまで採らない。

## 7. O3 — derive protected target from adjacent realization Step

```text
BP-06
  next Step BP-07
  BP-07 applicationPatterns = [order_entry.submit_order]
  therefore BP-06 target = order_entry.submit_order
```

### 判定

Reject。

理由:

- ordered adjacencyはauthorization-target semantic relationではない
- manual step挿入、順序変更、複数candidate actionで曖昧になる
- explicit exact reference原則に反する
- Business Action wordingからの推測と本質的に同じ

## 8. O4 — derive protected target from Legacy Auth policy

```text
authorization.json
  action=execute
  resource=order_entry.submit_order
```

からBP-06 targetを生成する。

### 判定

Reject。

理由:

- Compatibility / policy realizationをBusiness authorityへ逆流させる
- provider / RBAC / ReBAC / ABAC replaceabilityを損なう
- role / organization / allow-deny policyとBusiness concernを結合する
- No Multiple SoT境界を悪化させる

## 9. O1 test-only pressure contract

Lane CでO1を試す場合、少なくとも次をPressureする。

### T1 — target ref does not imply realization

```text
BP-06.applicationPatterns = []
BP-06.applicationPatternTargets = [order_entry.submit_order]

NoApplicationFunction = true
```

### T2 — exact resolution

`applicationPatternTargets`はnormalized Worldでexactに1件解決する。

missing / ambiguousはfail closed。

### T3 — target rename classification

`order_entry.submit_order` rename時:

- stale target only -> reject / semantic error
- Business sourceとApplication sourceのcoordinated accepted rename -> valid

名称類似で自動補修しない。

### T4 — target change is Business cross-view change

```text
order_entry.submit_order
  -> some_other_accepted_pattern
```

へtargetだけ変われば、BP-06のBusiness cross-view meaning差として観測する。

BP-06がApplication Function realizationへ変わったとは扱わない。

### T5 — Pattern-owned meaning remains derived

Target Patternの:

- Capability
- Data / Relation
- Datasource
- Query

をBP-06へ複製しない。

必要ならProjectionでderiveする。

### T6 — Authorization policy remains external authority

Legacy Auth Referenceにpolicyが存在しても、Business fixtureへ:

- organization code
- role code
- allow / deny
- provider binding

をコピーしない。

### T7 — no universal relation conclusion

O1がPASSしても:

```text
Generic SemanticRef required
Generic Core Edge required
Business graph complete
```

とは主張しない。

## 10. normalized D3への含意

D3の現在の説明:

```text
Step body
  businessAction
  explicitSemanticRefs
```

は、「ref identityの集合」だけでは意味が足りない可能性がB-F23で示された。

ただしproduction形を今:

```text
ExplicitSemanticRef { Role, Kind, Target }
```

へ固定しない。

O1 pressureから得たいのはより狭い原則である。

```text
If a cross-view reference changes Business meaning,
its semantic role must not be confused with realization.
```

production D3は後でこの原則を満たす最小形を再評価する。

## 11. BP-03 / BP-05 Validationは別pressureとして保留

BP-03とBP-05は同じValidationを参照するが、現在のEvidenceだけで:

```text
supports
checks
requires
```

等のrole taxonomyを固定しない。

先にBP-03側Validation refsをtest-onlyにsubtractし:

- impact discoverability
- cross-view review usefulness
- diagnostic loss

が実際に悪化するかを見る。

悪化しなければ、BP-03側explicit Validation refs自体が冗長な可能性もある。

## 12. X-TEA fallbackとの整合

XeadModelerはTaskActionとFunctionを別objectとして保持し、TaskAction配下の`TaskFunctionIO`が`FunctionID` / `IOID`を参照する。

ここから得るのは:

```text
Business-to-Function cross-reference can carry context
without collapsing Business Action into Function
```

というpressureである。

O1はX-TEAの`TaskFunctionIO` shapeをコピーせず、この境界だけをTEIのcurrent Referenceへ最小に適用する。

## 13. PAS-07 / RQ mapping

新E7は追加しない。

```text
E7-01
  target ref != Application Function realization

E7-02
  target ref exact resolution / no duplicated Pattern meaning

E7-04
  target changeをBusiness cross-view changeとして保持

E7-09
  generic ref relation / Core EdgeをEvidenceなしで固定しない
```

RQ-025:

```text
Business Action may use / reference Application meaning
without being identical to Application Function
```

RQ-026:

```text
cross-view trace must preserve why identities are related
when that distinction affects accepted meaning
```

までをPressureする。

## 14. 現時点の判断

```text
O0 wording only
  insufficient for exact protected-target trace

O1 narrow Application Pattern target refs
  primary test-only candidate

O2 generic role-qualified refs
  defer

O3 adjacency derivation
  reject

O4 policy derivation
  reject

production schema change
  not proven
```

## 15. 次

1. Lane CへO1をtest-only primary candidateとして返す。
2. O1でT1〜T7を実装可能かPressureする。
3. BP-03 Validation refs subtractionを別testで比較する。
4. O1だけでD3が十分ならgeneric role taxonomyを作らない。
5. 複数realistic Referenceでrole差が反復した場合のみO2をreopenする。
6. actual runner green前にproduction ADRへ進まない。

## 16. 境界

- Product Requirement追加なし
- PAS / E7追加なし
- Gate追加なし
- production `applicationPatternTargets` field採用なし
- production generic SemanticRef / role enum採用なし
- generic Core Edge追加なし
- Authorization policy ownership変更なし
- Workflow Engine追加なし
- Gate B / Gate C Required Evidence変更なし
