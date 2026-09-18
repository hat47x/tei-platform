# Actor Context R3D — Authorization Requirement Authoring Scope

- Status: ISSUE-000053 / Phase 3 authority-scope pressure / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3C_AUTHORIZATION_REQUIREMENT_AUTHORITY_2026-09.md`
- Related:
  - `plan/BUSINESS_PROCEDURE_EXPLICIT_REF_AUTHORITY_PRESSURE_2026-09.md`
  - `plan/LANE_B_AUTHORIZATION_TARGET_AUTHORITY_AUDIT_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 問い

R3Cでは、BP-02についてAuthorization requirementが未解決であることを確認した。

次の問いは単に「requirementをどのファイルへ書くか」ではない。

```text
what does the requirement apply to?
```

を先に決める必要がある。

少なくとも次は意味が異なる。

```text
Business Step scoped requirement
  -> this Business Action in this Procedure context

Application Pattern scoped requirement
  -> every use of this Application interaction

Application-wide / profile scoped requirement
  -> a broader explicitly selected Application security scope
```

R3Dではproduction locusを選ばず、最初の二つのscope差をtest-onlyでPressureする。

## 2. current Application sourceが表すもの

`reference/workspace/applications/order-entry/application.tei`には:

```text
uses semantic security
```

がある。

これはApplicationがsecurity Semantic namespaceを利用可能であることを示す。

また:

```text
realize security.authorize with custom go
```

が存在する。

これはSemantic CapabilityにApplication realizationがあることを示す。

しかし、どちらも:

```text
BP-02 requires authorization
order_entry.customer_reference requires authorization
all order_entry patterns require authorization
```

をauthorしていない。

したがって:

```text
capability available
  != capability applicable

capability realized
  != capability required for every interaction
```

とする。

## 3. Business Step scope

Business Procedure側でrequirementを明示する場合、その自然なscopeは特定Business Actionである。

test-only例:

```text
procedure order-entry.order-handling
step BP-02
authorization requirement = required
supporting capability = security.authorize
```

この意味は、同じ `order_entry.customer_reference` Patternを別Business Actionが再利用したとしても、自動的にはその別Actionへ波及しない。

長所:

- Business concernとしての認可要否をそのBusiness contextに閉じ込められる
- Pattern reuseとBusiness policy contextを分離できる

pressure:

- 同じPatternを使う複数Actionすべてが同じsecurity requirementを持つ場合は重複authoringになり得る
- technical interaction invariantをBusiness Procedureへ複製する危険がある

## 4. Application Pattern scope

Application Pattern側でrequirementを明示する場合、その自然なscopeはPatternの全利用である。

test-only例:

```text
pattern order_entry.customer_reference
authorization requirement = required
supporting capability = security.authorize
```

この意味なら、同じPatternを別Business Actionから利用しても同じrequirementが適用される。

長所:

- Application interaction固有のsecurity invariantをPatternに閉じ込められる
- Business Procedure間の重複を避けられる

pressure:

- Business contextによって認可要否が異なるなら過剰適用になる
- PatternがBusiness policyを所有する形にならないよう注意が必要
- role / organization / allow-deny policyをPatternへコピーしてはいけない

## 5. current single-use Referenceではscopeを選べない

current fixtureでは `order_entry.customer_reference` を参照するBusiness StepはBP-02のみである。

したがってcurrentだけを見ると:

```text
Business Step scoped required
Application Pattern scoped required
```

のaffected setが偶然一致する。

これは二つのauthorityが同じことを意味する証明ではない。

R3D test-only pressureでは、同じPatternを使う第二Business Actionをfixture copyへだけ追加する。

その場合:

```text
Business Step scope
  -> BP-02 only

Application Pattern scope
  -> BP-02 + BP-02-ALT
```

となり、scope差が顕在化する。

## 6. Application-wide/profile scope

第三候補としてApplication-wide / security profile scopeがあり得る。

ただしcurrent:

```text
uses semantic security
```

をそのprofileとして再解釈しない。

namespace dependencyとsecurity applicabilityを混同するためである。

profile候補を採るなら、将来:

- explicit applicability
- inclusion / exclusion
- precedence
- provenance
- pattern / procedure override relation

などをEvidenceに基づきPressureする必要がある。

R3Dではproduction candidateを作らない。

## 7. requirementとpolicy realizationを分ける

どのscopeを採っても、requirementは:

```text
Authorization is applicable here
```

というauthorityである。

これは:

- authenticated subject
- action/resource target
- role
- organization
- allow/deny rule
- provider implementation

を所有しない。

```text
requirement scope
  != policy realization
  != principal
  != protected target
```

を維持する。

## 8. current conclusion

R3Dで確定するのは次までである。

```text
authoring locus must preserve applicability scope

Business Step scope
  != Application Pattern scope

current single-use BP-02 Reference
  insufficient to choose between them

uses semantic security
  != application-wide Authorization requirement

security.authorize realization exists
  != Authorization applicability
```

production field / generic relation / precedence ruleは追加しない。

## 9. 次

次段では、第二のrealistic use caseを用意し、

- 同一Patternを異なるBusiness contextで再利用する例
- 同一security requirementを複数Procedureで共有する例

のどちらがTEIに現実的に必要かをPressureする。

そこでscopeが再発してからauthoring locus候補を絞る。
