# Lane B — Authorization Target Authority Audit — 2026-09

Status: **Lane B / supporting architecture-pressure audit / no production schema adoption**

## 1. 目的

B-F23で確認したBusiness Procedure explicit-ref role gapを、TEIの既存Authentication / Authorization authority境界へ戻して精査する。

問いは次である。

```text
BP-06
  determine whether the authenticated subject may execute the submit application action
```

について、

```text
what is the authorization semantic authority?
what is the protected application target?
where is policy realization authoritative?
```

を一つのBusiness Procedure fieldへ混ぜずに説明できるか。

本書はPAS-07 Trace正本ではない。Product Requirement、E7、Gate、production kindを追加しない。

## 2. 既存TEI authority

### 2.1 Semantic authorization contract

Reference Workspaceの`security.authorize`はSemantic Capabilityとして次を入力に持つ。

```text
subject
action
resource
```

したがって「認可を判定する意味」のauthorityは既に存在する。

```text
security.authorize
  = semantic authorization contract
```

### 2.2 Legacy Authorization policy fixture

`reference/legacy-auth/authorization.json`には、たとえば:

```text
organization_code = SALES
role_code         = ORDER_EDITOR
action            = execute
resource          = order_entry.submit_order
effect            = allow
```

が存在する。

これはReference上で「submit Patternをexecuteするpolicy」が具体的に実現されているEvidenceである。

### 2.3 Application authoring

`reference/workspace/applications/order-entry/application.tei`は:

```text
uses semantic security
```

を持つが、個々のApplication Patternについて:

```text
order_entry.submit_order
  requires security.authorize(action=?, resource=?)
```

のようなpattern-specific authorization relationはauthoringしていない。

したがってcurrent accepted Application authoringから、BP-06のprotected target relationを直接deriveすることはできない。

## 3. B-F24 — authority層を分離する

現在の意味は少なくとも次の三層に分かれる。

```text
Business concern
  BP-06: submit application actionを実行可能か確認する

Semantic authorization contract
  security.authorize(subject, action, resource)

Policy realization / compatibility Reference
  execute + order_entry.submit_order -> allow/deny rule
```

これらを同じ正本へ畳まない。

### 禁止する逆流 1 — Legacy policyをBusiness sourceへする

Business Procedureから:

```text
reference/legacy-auth/authorization.json
```

を直接参照してprotected targetを決めない。

理由:

- Legacy AuthはCompatibility / realization Referenceである
- 将来RBAC / ReBAC / ABAC / provider差し替えがあり得る
- Business meaningが特定provider policy fileへ従属する
- RQ-003 No Multiple SoT / Authorization capability boundaryを崩す

### 禁止する逆流 2 — Authorization policyをApplication Patternへ複製する

`order_entry.submit_order`へLegacy policy ruleをコピーしてBusiness側の不足を埋めない。

Application Patternはsubmit commandのFunction-Interaction authorityであり、具体的role / organization policyの正本ではない。

### 禁止する逆流 3 — Business actorからpolicyをderiveする

```text
actor label = sales operator
```

から`ORDER_EDITOR`等を推測しない。

```text
Business Actor / Job
  != Authorization Role / Principal / Policy
```

を維持する。

## 4. protected targetは何の意味か

BP-06がBusiness viewpointとして言っているのは:

```text
this authorization concern applies to the submit application action
```

である。

これはAuthorization policyそのものではない。

また、`order_entry.submit_order`がBP-06をrealizeするという意味でもない。

したがって必要なら保持すべきなのは:

```text
Business Action
  -- concerns/protects --> Application Pattern
```

というcross-view target relationである。

これはB-F23の:

```text
reference target != realization
```

を具体化する。

## 5. X-TEA fallbackとの整合

XeadModeler public sourceでは`TaskAction`とApplication Functionが別objectであり、TaskAction配下の`TaskFunctionIO`が`FunctionID`と`IOID`を保持する構造が確認できる。

したがって歴史的にも:

```text
Business Action
  references Function / IO context
  != Business Action is Function
```

という分離が存在する。

TEIへ`TaskFunctionIO`や`IOID`をコピーする根拠にはしない。

このEvidenceから採るのは:

```text
Business-to-Function cross-reference can have its own meaning
```

という設計pressureだけである。

## 6. 最小候補の再評価

### A0 — Business wordingのみ

```text
BP-06 wording
  "submit application action"
security.authorize ref
```

長所:
- 最小
- 新conceptなし

弱点:
- submit Pattern rename / replacementのimpactをexactに追えない
- target一致をpolicy realizationとcross-checkできない

### A1 — target-refをBusiness側に保持

Test-only概念例:

```text
Business Step BP-06
  supportingSemanticRef = security.authorize
  targetApplicationRef  = order_entry.submit_order
```

重要なのはfield名ではなくrole差である。

```text
supporting semantic authority
  != protected application target
  != realization
```

### A2 — targetを隣接Stepからderive

BP-06の直後がBP-07で、BP-07が`order_entry.submit_order`をrealizeすることからtargetを推測する案。

原則reject寄りとする。

理由:

- step順序だけでは「認可対象」を意味しない
- 並び替えやmanual step挿入で意味が壊れる
- explicit exact reference原則に反する
- Business Action wordingからのlexical inferenceと本質的に同じ

### A3 — Legacy policyからderive

Reject。

Provider / Compatibility policyをBusiness authorityへ逆流させる。

## 7. Cross-view validationの方向

A1相当をtest-onlyでPressureする場合、cross-view checkは一方向のownershipではなく整合を見る。

```text
Business Procedure
  BP-06 target -> order_entry.submit_order

Semantic
  security.authorize exists

Authorization realization / Reference policy
  action/resource may map -> order_entry.submit_order
```

ここで:

```text
Business target ref
  does not own allow/deny policy

Authorization policy
  does not own Business Procedure purpose/order

Application Pattern
  does not own Business actor/job context
```

を維持する。

Policy fixtureが存在するReferenceでは、不一致をdiagnosticにできる可能性がある。

例:

```text
BP-06 target = order_entry.submit_order
policy resource = order_entry.order_record
```

ならcross-view tensionを表示できる。

ただしseverityやuniversal validation ruleは複数Reference Evidenceが出るまで固定しない。

## 8. PAS-07 mapping

新E7は作らない。

### E7-01

Business ActionがApplication Patternをtargetとして参照しても、そのPattern realizationへ同一化しない。

### E7-02

Business concern / Semantic Capability / Application Patternをexact identityでcross-viewする。

### E7-04

target changeをBusiness-side cross-view meaning changeとして必要なら観測する。

### E7-05 / S7-02

Business actor / Procedure contextをAuthorization Role / policyへ変換しない。

### E7-09

Legacy policyとの一例だけを理由にgeneric Authorization relation / universal SemanticRef / RBAC schemaをproduction Coreへ入れない。

## 9. RQ-002との関係

RQ-002が要求するsource ↔ Canonical traceを考えると、将来A1相当がaccepted Business meaningになる場合は:

```text
Business source target ref
  -> normalized exact Application identity
```

を追跡可能にする必要がある。

一方:

```text
Legacy policy file locator
provider mapping detail
role code
organization code
```

をBusiness Canonical meaningへ混ぜない。

これは:

```text
semantic trace
  != realization provenance duplication
```

という既存Architecture境界に一致する。

## 10. 現時点の判定

```text
B-F24 authorization target authority split
  Confirmed

security.authorize semantic authority
  Existing

submit protected target in Legacy policy Reference
  Existing as realization/policy Evidence

pattern-specific Authorization relation in Application authoring
  Not present

Business Procedure exact protected-target trace
  Not present

derive target from adjacency/text
  Reject

derive Business meaning from Legacy policy
  Reject

role-qualified Business target ref
  Test-only pressure candidate

production generic relation / Authorization schema change
  Not proven
```

## 11. 次

Lane Cへ次のtest-only pressureを返す。

1. BP-06にprotected Application targetをexactに持つ候補を作る。
2. targetを持っても`NoApplicationFunction = true`を維持する。
3. `security.authorize` supporting refとtarget Pattern refをmachine-readableに区別する。
4. target rename / missing / ambiguousをfail closedする。
5. Legacy policy fixtureが存在する場合だけ、action/resource targetとのcross-view mismatchを追加pressureとして試す。
6. Business ProcedureがLegacy role / organization / allow ruleを所有しないことをnegative pressureで固定する。
7. 複数Referenceで再発するまでproduction generic relationを作らない。

## 12. 境界

- Product Requirement追加なし
- PAS / E7追加なし
- Gate追加なし
- Authorization Role / policy ownership変更なし
- Legacy policyをBusiness authorityへ昇格しない
- Application Patternへpolicyを複製しない
- production generic SemanticRef / Core Edge追加なし
- Workflow Engine追加なし
- Gate B / Gate C Required Evidence変更なし
