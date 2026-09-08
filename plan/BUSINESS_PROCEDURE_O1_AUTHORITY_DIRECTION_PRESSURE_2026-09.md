# Business Procedure O1 Authority Direction Pressure — 2026-09

Status: **Lane C / test-only pressure / no production schema adoption**

## 1. 目的

O1の`protected Application target`について、Authorization側とBusiness Procedure側のどちらがaccepted meaningを所有するかを分離する。

Current case:

```text
BP-06 Business Action
  determine whether the authenticated subject may execute the submit application action

supporting Semantic authority
  security.authorize

protected Application target
  order_entry.submit_order

direct Application Function realization
  none
```

ここでAuthorization policyに`order_entry.submit_order`が存在することを理由に、Business protected targetをpolicyから逆算しない。

## 2. X-TEA fallback — E-015

### Evidence

XeadModelerでは`TaskAction`と`TaskFunctionIO`が分離され、`TaskFunctionIO`が`FunctionID` / `IOID`を保持する。

これは少なくとも次を支持する。

```text
Business Action
  != Application Function

Business Action
  may use / refer to Function or IO
```

### Evidenceが証明しないこと

このsource evidenceから、TEIのgenericなref role taxonomy:

```text
realizes
protects
supports
concerns
checks
...
```

を導いてはならない。

特にX-TEA sourceで確認したTask/Function利用関係だけでは、BP-06のような「Functionをprotected targetとして指すが、そのFunctionでBusiness Actionを実現しない」というroleを一般概念として証明しない。

したがってO1はX-TEA概念の移植ではなく、TEI current Authorization cross-view pressureとして限定する。

## 3. TEI current authorities

### 3.1 Semantic Authorization contract

`reference/workspace/semantic/security/authorization.tei`の`security.authorize`は:

```text
subject
action
resource
```

を入力とする汎用Authorization Capabilityである。

このcontract自身は:

```text
BP-06
order_entry.submit_order
Application Pattern protected-target role
```

を所有しない。

したがって`security.authorize` refだけからO1 targetをderiveできない。

### 3.2 Legacy Authorization policy

`reference/legacy-auth/authorization.json`にはcurrent Referenceとして:

```text
organization = SALES
role         = ORDER_EDITOR
action       = execute
resource     = order_entry.submit_order
effect       = allow
```

が存在する。

これはO1 targetと一致するためcross-view corroborationには使える。

しかしLegacy Authorization policyはAuthorization Role / Action / Resource / Effectのsecurity-side authorityであり、Business Procedure sourceではない。

## 4. Authority direction

Current pressureでは次を固定する。

```text
Business protected target
  if accepted/persisted in the future
  -> Business-side authoritative meaning

Authorization policy
  -> security-side authoritative meaning
  -> may corroborate or contradict Business target
  -> must not author/backfill Business target
```

禁止する方向:

```text
Legacy Authorization policy
  -> infer BP-06 exists
  -> infer BP-06 protected target
  -> populate O1 automatically
```

許可する方向:

```text
explicit O1 target
  + Authorization policy
  -> cross-view consistency diagnostic
```

Policy changeはBusiness targetそのものを書き換えず、不一致をdiagnostic / review pressureとして露出させる。

## 5. Test-only Pressure

`tools/reference-business-procedure-check/authorization_target_authority_direction_pressure_test.go`を追加した。

### TestO1ProtectedTargetCanBeCorroboratedByLegacyPolicyWithoutMakingPolicyAuthoritative

1. O1 targetを`order_entry.submit_order`として明示する。
2. Legacy policyに`execute / order_entry.submit_order / allow`があることを確認する。
3. test memory上でpolicy resourceだけを別Patternへ変更する。
4. corroborationは失われるが、O1 targetがpolicy変更で書き換わらないことを確認する。

### TestLegacyAuthorizationPolicyCannotBackfillMissingO1ProtectedTarget

1. O1 targetを明示しないcandidateを作る。
2. Legacy policyには`order_entry.submit_order` ruleが存在することを確認する。
3. policyの存在だけではO1 targetが補完されないことを確認する。

## 6. 設計上の含意

### 固定する

```text
Authorization policy may corroborate Business target
Authorization policy must not author Business target
protected target != direct Function realization
Business actor != Authorization Role
```

### まだ固定しない

```text
production applicationPatternTargets field
generic ExplicitSemanticRef { Kind, Role, Target }
generic role enum
Business Procedure -> Authorization policy binding
automatic policy generation from Business Procedure
```

## 7. Production source判断

本Pressureは「O1 roleを表現できる」「security-side policyと独立authorityにできる」ことだけを確認する。

まだ次は確認していない。

```text
file-native Business source persists O1 target
decode / reload preserves O1 target
Shared Acceptance preserves O1 target
stale detection includes O1 target
second realistic corpus also requires protected-target role
```

したがってPrimary `business-procedure-reference-v0`へtarget fieldを追加しない。

O1をproduction accepted Business meaningとして残す場合のみ、authoritative source pressureへ進む。

## 8. E-015 conclusion

E-015の結論を次とする。

```text
X-TEA evidence
  supports Business Action / Function separation
  does not prove generic protected-target role taxonomy

TEI local evidence
  proves current BP-06 needs target-vs-realization distinction for cross-view review
  proves Legacy Authorization policy can corroborate but must not author that target

therefore
  keep O1 narrow and test-only
  do not infer target from security policy
  do not generalize role taxonomy yet
```
