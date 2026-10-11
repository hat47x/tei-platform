# Actor Context R3C — Authorization Requirement Authority

- Status: ISSUE-000053 / Phase 3 authority-order refinement / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3B_AUTHORIZATION_PREREQUISITE_BOUNDARY_2026-09.md`
- Related: `docs/audits/LANE_B_AUTHORIZATION_TARGET_AUTHORITY_AUDIT_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 問い

R3A/R3Bでprincipal、target、policyを分離した結果、その前段にもう一つ独立した問いが残る。

```text
Does this Business Action require Authorization at all?
```

このrequirementを解決しないまま、

- principalが必要
- action/resourceが必要
- Authorization evaluationが必要

と扱うと、read-only Agentへ不要なsecurity semanticsを発明する可能性がある。

逆に、security referenceが見当たらないだけで `not-required` と扱うと、fail-openになる。

## 2. current Referenceの対照

### BP-06

```text
businessAction =
  determine whether the authenticated subject may execute the submit application action

realization = security-boundary
capabilities = [security.authorize]
```

このStep自身については、Business ProcedureがAuthorization concernを明示している。

test-only requirement projectionでは:

```text
status = required
source = business-procedure-security-boundary
supporting capability = security.authorize
```

とできる。

### BP-02

```text
businessAction =
  confirm the customer associated with the order

realization = application-use
applicationPatterns = [order_entry.customer_reference]
```

Authorization concernの明示はない。

しかしabsenceだけから `not-required` を作らない。

```text
status = unresolved
```

とする。

## 3. tri-state pressure

production enumとして固定しないが、Pressure上は少なくとも:

```text
required
not-required
unresolved
```

を区別する。

current Referenceから直接得られるのは:

- BP-06 = required
- BP-02 = unresolved

である。

current ReferenceにはBP-02をexplicit `not-required` とするaccepted sourceはない。

## 4. policy presence / absenceはrequirementをauthorしない

current Legacy policyには `browse / order_entry.recent_orders` がある。

しかしBP-00 Business Stepはsecurity boundaryを明示していない。

したがってpolicy presenceだけでBP-00を `required` にしない。

同様に、current policyに `order_entry.customer_reference` がないことからBP-02を `not-required` にしない。

test-onlyでcustomer_reference policy ruleを追加しても、Business Procedure requirement statusは変えない。

## 5. Application accessもrequirement authorityではない

BP-02のdatasource accessは `read`。

これはData access semanticsであり:

```text
read
  != authorization required
  != authorization not-required
```

である。

read/write privilegeとBusiness/Application Authorization applicabilityを混同しない。

## 6. principal / target / evaluationの順序を修正する

Authorization-specific prerequisiteはrequirementに条件づける。

```text
requirement = required
  -> authenticated subject / principal needed
  -> explicit action/resource target needed
  -> security.authorize evaluation needed

requirement = not-required
  -> Authorization-specific principal/target/evaluationは不要
  -> 他のruntime/security requirementは別途評価

requirement = unresolved
  -> fail closed before deciding either path
```

ここで「not-required」は将来explicit authorityが存在する場合の概念であり、absenceから生成しない。

## 7. R3A/R3Bへの補正

R3Aの `authorization-action-resource-unresolved` をcurrent BP-02の最初のgapとはみなさない。

より手前を:

```text
authorization-requirement-unresolved
```

とする。

R3Bでは:

- current blocker
- deferred runtime gaps

を分ける。

`authorization-not-evaluated` はruntime stateとして保持できるが、requirementがrequiredと判明する前はblockerへ昇格しない。

## 8. 次のauthority候補

BP-02等のapplication-use Actionでrequirementをどこにauthorするかは未決定。

比較候補:

1. Business Procedure側にrole-qualified security concern / requirement refを置く
2. Application Pattern側にsecurity requirement relationを置く
3. higher-level accepted security policy/profileがApplication Pattern群へ明示適用される
4. explicit not-requiredを表現する必要があるか、default ruleで十分かを別途検証する

この段階ではwinnerを選ばず、generic production relationも追加しない。

## 9. Evidence

Design / Source Evidence。

actual runtime Authorization、Query invocation、Human Outcome Evidenceではない。
