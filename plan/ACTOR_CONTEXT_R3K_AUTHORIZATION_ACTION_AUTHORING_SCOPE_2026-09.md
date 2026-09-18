# Actor Context R3K — Authorization Action Authoring Scope Pressure

- Status: ISSUE-000053 / Phase 3 action-authoring scope pressure / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3J_AUTHORIZATION_ACTION_AUTHORITY_GATE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3F_REPOSITORY_BACKED_AUTHORIZATION_SCOPE_2026-09.md`
  - `reference/business-system-design/order-entry-business-procedure.json`
  - `reference/business-system-design/order-entry-follow-up-procedure.json`
  - `reference/workspace/applications/order-entry/patterns/recent-orders.tei`
  - `reference/legacy-auth/authorization.json`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3JではBP-06について、Business wording / Pattern kind / Legacy policyが `execute` を強く示唆しても、それらを合成してAuthorization action authorityを生成しないことを確認した。

次の問いは、将来explicit action authorityを導入する場合に、そのscopeをどこへ置くべきかである。

R3Kではproduction locusを選ばず、既存のcross-Procedure Pattern reuseを使って少なくとも次の三候補のreach差をPressureする。

```text
Business Step scoped action
Application Pattern scoped action
explicit cross-view mapping
```

## 2. second action caseはtest-only

current Repositoryでは:

```text
order-entry.order-handling/BP-00
  -> order_entry.recent_orders

order-entry.follow-up/FU-00
  -> order_entry.recent_orders
```

が存在する。

Pattern kindは `browse`、Legacy policyにも:

```text
browse / order_entry.recent_orders
```

がある。

しかしBP-00 / FU-00のAuthorization requirementはどちらもcurrent Business source上では `unresolved` である。

したがってR3Kで `browse` action candidateを置く場合、それはscope比較のためのtest-only candidateである。

```text
Pattern kind = browse
  + policy action = browse
  != accepted AuthorizationAction = browse
```

## 3. Business Step scoped action

test-onlyに:

```text
procedure = order-entry.order-handling
step = BP-00
action = browse
```

を置く場合、そのreachはBP-00だけである。

同じPatternを使うFU-00へ自動伝播しない。

長所候補:

- Business context固有のaction semanticsを保持しやすい
- same Pattern reuseとAuthorization scopeを分離できる

pressure:

- 複数Stepで同じactionを使う場合に重複authoringが生じ得る
- interaction固有のoperationをBusiness側へ複製する危険がある

## 4. Application Pattern scoped action

test-onlyに:

```text
pattern = order_entry.recent_orders
action = browse
```

を置く場合、reachはBP-00 / FU-00の双方になる。

長所候補:

- interaction固有のAuthorization operationをPatternに閉じ込められる
- cross-Procedure reuseで重複を避けられる

pressure:

- Business contextによってaction semanticsが異なる場合に過剰適用になる
- Pattern kind `browse` とAuthorizationAction `browse` を同一語彙だからという理由だけで同一authorityにしてはいけない

## 5. explicit cross-view mapping

第三候補として、Business Step / Application Patternの対応辺にactionを明示するcross-view mappingを考えられる。

test-onlyでは:

```text
order-entry.order-handling/BP-00
  + order_entry.recent_orders
  -> browse

order-entry.follow-up/FU-00
  + order_entry.recent_orders
  -> browse
```

のように、どの利用辺へactionを適用するかを明示できる。

これはscope表現力が高い一方:

- mapping artifactが第二の正本にならないか
- Procedure / Pattern変更時のfreshnessをどう追うか
-同一action反復を無意味に増やさないか

を別途Pressureする必要がある。

R3Kではcross-view mappingを採用しない。

## 6. reach差

同じtest-only `browse` candidateでも:

```text
Business Step scope
  -> order-entry.order-handling/BP-00

Application Pattern scope
  -> BP-00 + FU-00

Cross-view scope
  -> explicitly selected Procedure/Step + Pattern edges
```

となる。

この差はproduction locus選択前に保持すべき意味差である。

## 7. action scopeとrequirement scopeを結合しない

R3Kでは特に:

```text
Authorization requirement scope
  != Authorization action scope
```

を固定する。

Pattern-level requirementが将来成立してもactionまでPattern scopeとは限らない。

Business Step requirementが成立してもactionがBusiness Step scopeとは限らない。

両者を同じ場所へ置く設計は候補になり得るが、scope一致を暗黙規則にしない。

## 8. current Evidence Gate

current `recent_orders` caseは:

- repository-backed Pattern reuse
- repository-backed Pattern kind
- repository-backed Legacy policy action
- Authorization requirementはunresolved

である。

したがって:

```text
repository-backed reuse
  + corroborating action hints
  -> action-scope semantic pressure

but

no second repository-backed required Authorization interaction
  -> production action locus not qualified
```

とする。

## 9. test-only source

- `tools/reference-business-procedure-check/actor_context_r3k_authorization_action_scope_pressure_test.go`

Pressureする不変条件:

- Business Step actionは同Patternの別利用へ漏れない
- Pattern actionは全Pattern利用へreachする
- cross-view candidateは明示されたedgeだけへreachする
- Pattern kind / Legacy policyからaction authorityを生成しない
- requirement unresolvedをrequiredへ変換しない
- action scopeからrequirement scopeを逆算しない
- Winner / Preferred / Confidenceを作らない

## 10. current conclusion

R3Kで確定するのは次までである。

```text
Business Step action scope
  != Application Pattern action scope
  != cross-view action scope

Authorization requirement scope
  != Authorization action scope

current recent_orders case
  = useful semantic pressure
  != qualifying production-authority second case
```

production field / generic relation / mapping artifactは追加しない。

## 11. 次

次段では、action authority locusを決めるためのqualifying Evidence条件を明示する。

少なくとも:

- requirement = required がrepository-backedに成立
- protected resource authorityが成立
- actionがBusiness / Application authorityとして明示
- second interactionで異なるPattern kind / action語彙または異なるreuse shapeが存在
- Legacy policy providerを外してもmeaningを説明できる

ことを満たすsecond caseが必要である。
