# ISSUE-000053: AI-native継続開発とAgent Operational KnowledgeをReferenceで検証する

- 状態: Proposed / phased
- 提案日: 2026-09-18
- 性格: Product validation / AI-native development / operational agent reference
- 主担当要求: RQ-010 / RQ-015 / RQ-018〜RQ-021 / RQ-025〜RQ-030
- 関連ADR: ADR-0022 / ADR-0023 / ADR-0028
- 関連Issue: ISSUE-000027 / ISSUE-000042 / ISSUE-000047 / ISSUE-000052

## 目的

TEIを、

1. 生成AIが継続的にApplicationを構築・変更するための開発知識基盤
2. 人間がgenerated code全体を読まずに意味・影響・例外・Evidenceを理解する認知基盤
3. AI Business Actorが許可された範囲で業務を遂行するOperational Knowledge基盤
4. Domain / Infrastructure知識を別Applicationへ移植する再利用基盤

として利用できるかを、小さいReferenceで段階的に検証する。

このIssueはAI agent frameworkをTEI Coreへ追加するIssueではない。

## 減らす不確実性

### A. AI Builder re-entry

別session / 別agentが既存Applicationへ入るとき、Repository全体を自由探索して意味を再構成しなくても、

- change target
- dependency / impact
- accepted Decision
- Capability / Binding
- native asset
- unresolved
- Evidence

へ機械的に到達できるか。

### B. Human cognitive review

AIが行った変更について、人間がgenerated code全体を読む前に、

- semantic diff
- impact
- exception
- unresolved
- authority change
- Evidence

を高密度なProjectionで把握できるか。

### C. AI Business Actor

accepted Business Procedure / Action / Capabilityを読み、

- 自分が実行可能なAction
- Authorization不足
- human / external handoff
- Command後の必要Observation
- completion / failure / unknown

を判定できるか。

### D. Knowledge portability

一つのReference Applicationから抽出したDomain / Infrastructure knowledgeを別Applicationへ再利用したとき、

- origin
- assumption
- Variation
- dependency
- destination override
- Evidence

を失わず差分適用できるか。

## Phase 0 — 既存資産だけでMachine-readable contextを組む

新しいAI専用Knowledge Storeを作らない。

まずcurrent TEIにある、

- Canonical ID
- Source Index
- Semantic / Application dependency
- Business Procedure cross-view
- Capability
- Binding
- Validation
- Evidence / diagnostic
- Reference Workspace

から、AI Builder / AI Business Actorが必要とするcontextをどこまでderivedに作れるかを棚卸しする。

不足があっても、ただちにCanonical kindを増やさない。

Phase 0では、既存のDesign Synthesis `DesignContextPacket`をAI-native開発全体の万能contextへ拡張しない。

```text
Builder re-entry context
  != Design Synthesis Context Packet
  != Human Review Projection
  != Operational Projection
```

これらは同じCanonical / Source / Evidence / accepted Decisionをauthority rootとして共有しつつ、用途ごとに必要情報とdisclosure範囲を変えるderived projectionとして比較する。

最低限、各projectionについて、

- current task / audience
- selected subjects
- authority / Source / Evidence trace
- freshness / accepted snapshot dependency
- unresolved / omitted detail
- drill-down先
- authority limit

を説明できるかを確認する。ただしこれをproduction `ApplicationContext` schemaとして先に固定しない。

## Phase 0 findings — Actor Context Projection

Phase 0の設計基線は `plan/ACTOR_CONTEXT_PROJECTIONS_2026-09.md` とする。

current TEIの棚卸しから、次を確認した。

### F0-1 — 新しいAI専用Knowledge Storeは不要

AI Builder / Human Review / AI Business Actorは、同じAuthoritative Source / Canonical Meaning / Accepted Decisionをauthority rootとして利用できる。

必要なのはactor別のderived contextであり、第二の正本ではない。

### F0-2 — 一つの万能Context Packetへ統合しない

少なくとも次は用途が異なる。

```text
Builder Re-entry Context
Human Review Projection
Operational Projection
```

既存Design Synthesisの`DesignContextPacket`はBuilder系の一Projectionとして再利用できるが、Operational Agentまで含む万能schemaへ拡張しない。

### F0-3 — Operational Projectionはcross-viewである

Business Procedureだけでは、actual invocationに必要な情報が足りない。

Operational Projectionには必要に応じて、

- Business Action / Procedure
- Application Pattern / Semantic Capability
- Validation / Constraint
- Binding
- Permission / Authorization
- Resource
- Environment
- Observation / Evidence

をauthorityを混同せず束ねる必要がある。

### F0-4 — `canExecute`一値を導入しない

Agentに対しては少なくとも、

```text
defined
  -> resolvable
  -> bound
  -> authorized
  -> resource-ready
  -> invokable
  -> command acknowledged
  -> observed
  -> business continuation satisfied
```

を区別する必要がある。

この段階列をproduction enumとして固定はしないが、一つのboolean / confidenceへ畳まない。

### F0-5 — current order_entryはread-only Agent pressureへ利用可能

current Business Procedure Referenceのうち、

- BP-00〜BP-03
- BP-05
- BP-06
- BP-08

は、side effectを起こさず、Procedure理解、Pattern / Validation / Capability trace、Authorization boundary、Observation boundary、human / unresolved handoffをPressureできる。

BP-07 submit commandのside-effect Referenceは後段へdeferする。

### F0-6 — AI Business ActorをPresentation Targetへ自動分類しない

Operational Projectionを読むAI Agentは、Web / Desktop等と同じPresentation Targetとは限らない。

MCP / API等はmachine-readable interface候補になり得るが、

```text
transport choice
  != Presentation semantics
  != execution authority
```

である。

R0 / R3では、Application UI経由とCapability直結のmachine consumer経路を区別し、Target CoreをAgent都合で拡張しない。

## R0 — Static actor context derivation

Phase 1へ進む前に、同一shared accepted / reference baselineから次の3 Projectionを静的に組み、情報の重なりと差を確認する。

1. Builder Re-entry Context
2. Human Review Projection
3. read-only Operational Projection

R0で確認する項目:

- 同じCanonical / Source / Evidenceへ逆追跡できる
- actorごとに不要な情報を無制限dumpしない
- omitted / unresolvedを「存在しない」へ変換しない
- shared baselineだけでなく、actorごとのactual selected input fingerprintを追える
- unrelated changeでstale-allにせず、actual dependency changeを見逃さない
- Projection削除でaccepted meaningが変わらない
- Human向け要約をAgent authorityへ流用しない
- Agent向けruntime resolutionをCanonical truthへ逆流させない
- DesignContextPacketを万能schemaへ昇格させない

R0ではproduction schema、Agent framework、Runtime invocationを追加しない。

## R0 implementation source status

2026-09-18、test-onlyの最初のstatic pressure sourceを追加した。

- `tools/reference-business-procedure-check/actor_context_r0_pressure_test.go`

対象:

- Builder Re-entry Context — BP-02周辺のPattern / Data / Relation / dependency
- Human Review Projection — customer edit intentional conflict
- Operational Projection — BP-02 read-only Action
- unresolved preservation — BP-04 manual judgement / BP-08 Observation boundary

source上では次を固定している。

- 同じBusiness Procedure Reference fingerprintを三Projectionで保持
- そのfingerprintだけをprojection freshness authorityにせず、selected workspace source definition fingerprintを各Projectionへ保持
- Datasource / accepted accessを含むProjectionでは、そのauthorityを所有するApplication source fingerprintも保持
- Human Reviewではcustomer-edit conflict Evidence fingerprintも追加
- `canonical=false`を維持
- BP-02 datasource accessを`read`のまま保持
- read/write conflictをHuman Reviewへ露出
- BP-04をmachine executableへ昇格させない
- BP-08へfake Semantic Capabilityを追加しない
- `CanExecute` / `Executable` / `Authorized` / `Complete` / `Confidence`等の総合authority shortcut fieldをtest-only typeへ置かない

**この時点ではimplementation sourceであり、actual `go test` PASS Evidenceではない。**

actual runner / local executionが得られるまで、R0をExecutable Evidenceへ昇格させない。

## Phase 1 — AI Builder re-entry Reference

既存Reference Applicationへ小さいchange challengeを設定する。

例:

```text
change request
  -> relevant Canonical subjects
  -> current accepted Decision / constraint
  -> impacted Projection / Capability / native asset
  -> proposed change
  -> Shared Acceptance
  -> reload / conformance
```

比較対象:

- source / Repositoryをほぼ自由探索するbaseline
- TEI structured contextを先に利用する経路

観測候補:

- 読み込む必要があったartifact数
- irrelevant context量
- semantic sourceを取り違えた回数
- change amplification
- acceptance前に検出できたimpact
- re-entry時に人へ聞き直した事項

token量だけをProduct KPIにはしない。

## Phase 1 first challenge — customer_referenceを編集可能にする要求

最初のAI Builder re-entry challengeは `plan/ACTOR_CONTEXT_R1_AI_BUILDER_CHANGE_CHALLENGE_2026-09.md` を正本とする。

変更要求:

> 注文画面で参照している顧客情報を、その場で編集・更新できるようにする。

current accepted / Reference facts:

- `order_entry.customer_reference` は `reference` Pattern
- Relation = `sales.order.customer`
- datasource = `master`
- `order_entry.master access = read`
- Contract = `master_data.customer`
- Query = `find_by_id`
- Contractは `read customer.customer` を持ち、writeを持たない
- BP-02 Reference meaningは「associated customerをconfirmする」であり、customer maintenance authorityではない
- 既存intentional conflict fixtureはread-write要求に対し `BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT` を再現する

R1で確定してよいのは、

```text
current customer_reference path
  + direct customer mutation
  -> accepted designとconflict
```

までとする。

次は自動決定しない。

- read-only境界を維持して別ownerへhandoffする
- order_entryへ明示的write pathを追加する
- customer ownerのCapabilityを呼ぶ
- その他の設計へ変更する

これらはProposal / design alternativeであり、current conflictからwinnerを導かない。

比較はRepository free exploration / code-first baselineと、TEI structured context firstを同一change requestで行う。

観測の中心はcode generation速度ではなく、

- conflict発見前に実装へ進んだか
- exact authorityへ到達できたか
- ownership / write authority / Capabilityを発明したか
- contradictionとunresolvedを分けたか
- source / evidenceへ逆追跡できたか
- 一つのstructured constraintを複数Projectionへ再利用できたか

とする。

### R1 source status

2026-09-18、最初のchange challengeについてtest-only sourceを追加した。

- `tools/reference-business-procedure-check/actor_context_r1_builder_pressure_test.go`

source上では、current Pattern / datasource access / Database Contract / Business Reference / intentional conflictを一つのbounded constraint projectionへ集める。R1 probeはこれらdecisive inputのcontent fingerprintを保持し、Business Procedure fingerprintだけをfreshness authorityにはしない。Application / Pattern / ContractはCanonical subjectからSource Indexでcurrent locatorを解決し、固定file pathをsemantic identityへしない。

重要な停止線:

- Contract writesが空であることから架空のwrite Capabilityを作らない
- current path conflictをfeature-wide impossibilityへ一般化しない
- ownership / authority / Business responsibilityを要求文から推測しない
- resolution candidateをAccepted / Winnerへ昇格しない

actual `go test` PASSおよびfree-exploration vs structured-context比較runは未取得であり、Source Evidence以上へは昇格させない。

## Phase 2 — Human Review Projection

同じchange challengeから、

- semantic diff
- impacted subjects
- unresolved
- custom / native code touchpoint
- authority / permission change
- validation / conformance result
- Evidence

を一つのreviewable projectionへ出せるかを検証する。

人間がsourceを一切見なくてよいことを成功条件にはしない。

成功条件は、**sourceを見る前に、どこを見るべきかを正しく絞れること**である。

### R2 first challenge — human review projection

R1と同じcustomer edit要求を使い、生成AIの変更をコード精読前に人間が理解するためのchallengeは `plan/ACTOR_CONTEXT_R2_HUMAN_REVIEW_CHALLENGE_2026-09.md` を正本とする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r2_human_review_pressure_test.go`

R1 constraint projectionから、そのReference / required input fingerprintsを失わず継承し、

- intent
- current Business / Pattern / Relation / datasource / Contract meaning
- conflict
- unresolved dimensions
- potentially affected authority
- review questions
- exact drill-down refs（R1 fingerprint input集合からderive）
- authority limits

を組み立てる。

停止線:

- `potentially affected`を`must change`へ変換しない
- review questionをanswer / Decisionへ変換しない
- `Approved` / `Rejected` / `Safe` / `Score` / `Confidence`等をProjectionへ持ち込まない
- summaryをauthoritative sourceの代替にしない

source追加時点ではHuman Outcome Evidenceではない。actual human reviewでorientation / irrelevant reading / semantic misread等を比較するまではSource Evidenceに留める。

## Phase 3 — read-only AI Business Actor

Business Procedure Referenceが十分に安定した段階で、最初は副作用のないActionを選ぶ。

AI Business Actorが、

1. current Procedure / Actionを読む
2. relevant Data / Constraint / Capabilityを特定する
3. Authorization requirementを読む
4. read-only Capabilityをinvokeする
5. Observation / Evidenceを解釈する
6. next Actionまたはhandoffを返す

経路をReferenceで検証する。

GUI click imitationはbaseline比較に使っても、標準経路にはしない。

### R3 first challenge — read-only Business Actor

最初のread-only Agent challengeは `plan/ACTOR_CONTEXT_R3_READ_ONLY_BUSINESS_ACTOR_CHALLENGE_2026-09.md` を正本とする。

R3 sourceはApplication / Pattern / Contract / Business Procedureというread-routeのactual decisive inputだけをfingerprintし、R1固有のcustomer-edit conflict Evidenceをfreshness dependencyへ持ち込まない。Application / Pattern / Contractはsubject identityからSource Indexでcurrent locatorを解決し、SourceRefsも同じfingerprint input集合からderiveする。

対象はBP-02:

```text
confirm the customer associated with the order
```

current Referenceから、

```text
BP-02
  -> order_entry.customer_reference
  -> sales.order.customer
  -> master
  -> master_data.customer
  -> find_by_id
  -> customer.customer.customer_id -> id
  -> customer.customer zero-or-one
```

までをstaticに解決する。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3_read_only_agent_pressure_test.go`

保持するruntime gap:

- actor principal unresolved
- input instance value unresolved
- Authorization not evaluated
- Resource / Provider not evaluated

Source Evidenceでは、

- invocation attempted = false
- Observation received = false
- Business completion unresolved = true

を固定する。

これにより、

```text
read route understood
  != agent authorized
  != input value available
  != query succeeded
  != Business Action complete
```

をPressureする。

R3ではMCP / API等のtransport contract、Agent principal mapping、actual query invocationをまだ追加しない。

### R3A — principal / Authorization target boundary

R3 static read routeの次段は `plan/ACTOR_CONTEXT_R3A_PRINCIPAL_AUTHORIZATION_BOUNDARY_2026-09.md` を正本とする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3a_principal_authorization_pressure_test.go`

既存 `security.authenticate` / `security.authorize` を再利用し、新しいPrincipal / Role / Permission体系は作らない。

R3Aでは、

- Business Procedureのcandidate actor labelをruntime principalへ変換しない
- 明示的な `security.authenticate / authenticated` subjectだけでprincipal gapを解消する
- authenticated subjectだけではAuthorizationを成立させない
- `Application access=read` からAuthorization action `browse` を生成しない
- Legacy Authorization policyだけからmissing Business/Application targetをbackfillしない
- BP-02ではAuthorization requirement自体を未解決として保持し、requirement確定前にaction/resource不足を必須前提として扱わない

ことをPressureする。

```text
Business actor label
  != authenticated subject

authenticated subject
  != authorized

Application read access
  != Authorization browse action

policy rule
  != Business/Application target authority
```

Authentication Observationはruntime contextであり、R3 static source freshnessのcontent fingerprint集合へ混ぜない。

### R3B — Authorization prerequisite / fail-closed boundary

R3Aでauthenticated subjectを安全に解決できても、BP-02ではAuthorization requirement自体がcurrent accepted sourceから確定していない。

次段は `plan/ACTOR_CONTEXT_R3B_AUTHORIZATION_PREREQUISITE_BOUNDARY_2026-09.md` を正本とする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3b_authorization_prerequisite_pressure_test.go`

R3Bでは、

- BP-02の `Application access=read` から `browse` を生成しない
- Pattern identityだけからAuthorization resourceを生成しない
- BP-06に明示されたprotected targetをBP-02へ流用しない
- Legacy Authorization policyだけでmissing targetをbackfillしない
- action/resource authorityが未解決なら `security.authorize` をinvokeしない
- Authorization未評価のままContract Queryをinvokeしない
- stop reasonからhandoff ownerを推論しない

ことをPressureする。

```text
authenticated subject
  + read route understood
  + Authorization requirement unresolved
  -> authorization invocation attempted = false
  -> query invocation attempted = false
  -> completion unresolved
```

R3 sourceのSource Index移行に追随し、R3Aもsubject identity -> current locator解決を利用する。runtime Authentication Observationはstatic source freshnessへ混ぜない。

### R3C — Authorization requirement authority

R3Bをさらに精査すると、BP-02ではtarget/action以前に「このActionでAuthorizationが必要か」自体が未解決である。

正本:

- `plan/ACTOR_CONTEXT_R3C_AUTHORIZATION_REQUIREMENT_AUTHORITY_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3c_authorization_requirement_pressure_test.go`

current Referenceでは:

```text
BP-06 security-boundary + security.authorize
  -> Authorization requirement = required

BP-02 application-use
  -> Authorization requirement = unresolved
  != not-required
```

また、

```text
policy rule present
  != requirement required

policy rule absent
  != requirement not-required

Application access = read
  != requirement decision
```

をPressureする。

Operational prerequisiteはrequirementに条件づけ、requirementがrequiredの場合だけAuthorization用principal / target/action / evaluationを必須として扱う。requirementがunresolvedならそこでfail closedし、absenceから `not-required` を生成しない。

### R3D — Authorization requirement authoring scope

R3Cでrequirementを独立authorityにした後、authoring locusを選ぶ前にapplicability scopeを分離する。

正本:

- `plan/ACTOR_CONTEXT_R3D_AUTHORIZATION_REQUIREMENT_AUTHORING_SCOPE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3d_authorization_requirement_scope_pressure_test.go`

R3Dでは:

```text
Business Step scoped requirement
  != Application Pattern scoped requirement
```

をPressureする。

current BP-02ではcustomer_reference Patternの利用が1件しかないため、両候補のaffected setが偶然一致する。test-onlyに同Patternを使う第二Business Actionを追加すると、Business Step scopeは元Stepだけ、Pattern scopeは両Stepへ適用され、意味差が明示される。

また:

```text
uses semantic security
  != Authorization applicability

realize security.authorize
  != Authorization applicability
```

として、Semantic namespace availability / Capability realization / requirement applicabilityを分離する。

current ReferenceだけではBusiness Step側・Application Pattern側のどちらをproduction authorityにするか選ばない。


### R3E — second-use-case scope pressure

R3Dのscope差をsingle-use fixtureだけで終わらせず、同じApplication Patternを異なるBusiness contextから利用する第二use caseへ広げる。

正本:

- `plan/ACTOR_CONTEXT_R3E_AUTHORIZATION_REQUIREMENT_SECOND_USE_CASE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3e_authorization_second_use_case_pressure_test.go`

R3Eでは二つのcaseを分ける。

```text
Case A
  one Business context = explicit required
  second Business context = unresolved
  same Application Pattern reused

  -> Pattern reuseだけで第二contextをrequiredへしない

Case B
  two Business contexts = independently explicit required
  same Application Pattern reused

  -> duplication pressure
  != Pattern authority proven
```

また:

```text
unresolved
  != not-required
  != inherited required

Authorization applicable here
  != Authorization must be re-evaluated here
```

として、requirement applicabilityとevaluation cadenceを分離する。

この段階でもBusiness Step / Application Patternのproduction authoring locus、override / precedence rule、Application-wide profileは固定しない。

### R3F — repository-backed cross-Procedure scope pressure

R3Eのsynthetic second-use-caseを、current Repositoryに既に存在するPattern reuseでcorroborateする。

正本:

- `plan/ACTOR_CONTEXT_R3F_REPOSITORY_BACKED_AUTHORIZATION_SCOPE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3f_repository_backed_scope_pressure_test.go`

current Referenceには:

```text
order-entry.order-handling/BP-00
  -> order_entry.recent_orders

order-entry.follow-up/FU-00
  -> order_entry.recent_orders
```

が存在する。

二つは異なるBusiness Action文脈なので:

```text
Business Step scope
  -> one Procedure/Step address

Application Pattern scope
  -> both Procedure/Step addresses
```

というreach差をsynthetic Stepなしで観測できる。

さらにLegacy policyには `browse / order_entry.recent_orders / allow` が存在するが、BP-00 / FU-00のAuthorization requirementはcurrent Business source上ではどちらも `unresolved` のままとする。

```text
policy target exists
  != Authorization applicability authority
  != Pattern scope authority
```

なおR3Fは同一 `order_entry` Reference family内のcross-Procedure reuseであり、独立second corpusではない。ISSUE-000048 / ISSUE-000050のPre-ADR blocker Bを解消したとは扱わない。


### R3G — Authorization invariant / variation precedence

R3Fのrepository-backed cross-Procedure reuseを踏まえ、Pattern共通requirement候補とBusiness context固有variationを同じauthorityへ潰さないため、minimum invariant / additional requirement / exception intent / precedenceを分離する。

正本:

- `plan/ACTOR_CONTEXT_R3G_AUTHORIZATION_REQUIREMENT_INVARIANT_VARIATION_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3g_authorization_invariant_variation_pressure_test.go`

R3Gでは:

```text
Pattern minimum invariant
  != duplicated Business requirements

Business Step additional requirement
  != Pattern authority copy

explicit exception intent
  != accepted override

more specific scope
  != automatic precedence

applicability authority
  != precedence authority
```

をPressureする。

Pattern minimum `required` とBusiness Step `not-required` が衝突しても、より具体的なscopeだからStepが勝つとは扱わない。explicit exception intentがあってもprecedence authorityが未解決ならconflictのままfail closedする。

test-onlyには `pattern-minimum` / `step-exception` のprecedence候補を比較用に置くが、production ruleとして採用したとは扱わない。

この段階でもproduction Business Procedure schema、Application Pattern grammar、override relation、precedence ruleは固定しない。


### R3H — Authorization exception Evidence Gate

R3GでPattern minimum / Business Step variation / exception intent / precedenceを分離した後、production exception mechanismを必要とするrepository-backed caseがcurrent Referenceに存在するかを監査する。

正本:

- `plan/ACTOR_CONTEXT_R3H_AUTHORIZATION_EXCEPTION_EVIDENCE_GATE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3h_authorization_exception_evidence_gate_pressure_test.go`

current Referenceでは:

```text
BP-00 / FU-00
  same Pattern reuse
  -> requirement = unresolved

BP-06
  security-boundary + security.authorize
  -> requirement = required
```

までは存在するが、repository-backedなexplicit `not-required` requirementは観測されない。

Legacy policyのallow/deny ruleもrequirement / Pattern minimum / exception authorityへ昇格させない。

したがってcurrent exception Evidence Gateは未充足とし、`override` / `exclude` / `inherit` / priority / nearest-scope-wins等のproduction mechanismは追加しない。

synthetic required/not-required conflictはsemantic pressureとして保持できるが、repository-backed Business needなしにはproduction mechanism adoptionの根拠にしない。

qualifying caseが将来現れた場合でも:

```text
exception mechanism pressure qualified
  != precedence winner known
  != override syntax chosen
```

を維持する。



### R3I — Authorization target / evaluation cadence boundary

R3Hでexception mechanism lineをEvidence不足として保留し、read-only AI Business Actorの本線へ戻る。

正本:

- `plan/ACTOR_CONTEXT_R3I_AUTHORIZATION_TARGET_EVALUATION_CADENCE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3i_authorization_target_cadence_pressure_test.go`

R3Iでは:

```text
Authorization requirement
  != protected target/action authority
  != Authorization evaluation
  != evaluation reuse authority
```

をPressureする。

current BP-06は `security.authorize` によりrequirementは `required` で、protected-target overlayから `order_entry.submit_order` resource候補まで得られる。しかしoverlayはAuthorization actionをauthorしない。

Legacy policyに `execute / order_entry.submit_order` が存在してもmissing action authorityをbackfillしない。

したがって:

```text
requirement = required
resource resolved
action unresolved
  -> authorization-action-unresolved
```

で停止する。

またaction/resourceが明示され、prior Authorization evaluationが存在しても、別Step / Procedureへは暗黙reuseしない。

```text
evaluation observed once
  != evaluation reusable elsewhere
```

test-only reuse authorityは意味差比較にだけ利用し、production cache/session cadence、TTL、revocation modelは固定しない。


R3I refinementとして、evaluation ObservationはBusiness addressだけでは対応づけない。`subject / action / resource` tuple一致を要求し、unknown outcomeやtuple mismatchは `authorization-evaluation-observation-mismatch` として停止する。

またBP-06のBusiness Action本文に `execute` が現れ、Legacy policyにも `execute / order_entry.submit_order` が存在しても、その語彙一致からmissing Authorization action authorityを生成しない。current `denied` Observationは「評価済み」と「許可」を分離してoutcomeを保持する。

### R3J — Authorization action authority Evidence Gate

R3Iで `requirement=required` とprotected resourceまで解決してもAuthorization actionが独立authorityであることを確認したため、current Repositoryにaccepted action authorityが存在するかを監査する。

正本:

- `plan/ACTOR_CONTEXT_R3J_AUTHORIZATION_ACTION_AUTHORITY_GATE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3j_authorization_action_authority_gate_pressure_test.go`

current BP-06周辺には:

```text
Business wording contains "execute"
submit Pattern kind = command
Legacy policy action = execute
```

という相互に整合するhintがある。

しかし:

```text
multiple corroborating hints
  != accepted AuthorizationAction authority
```

として、`execute` を補完しない。current action authority Evidence Gateは未充足であり、`security.authorize` invocationは引き続きblockedとする。

test-only explicit action candidateにはprovenanceを要求し、それを与えてactionを解決しても次の状態は `authorization-not-evaluated` である。

production action field / generic relation / `command -> execute` convention / policy backfillは追加しない。


### R3K — Authorization action authoring scope pressure

R3Jでcurrent action authority Evidence Gateを未充足として保持したまま、production locusを選ぶ前にaction authorityのscope差をtest-onlyにPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3K_AUTHORIZATION_ACTION_AUTHORING_SCOPE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3k_authorization_action_scope_pressure_test.go`

current Repositoryの `order_entry.recent_orders` は:

```text
order-entry.order-handling/BP-00
order-entry.follow-up/FU-00
```

から再利用され、Pattern kindは `browse`、Legacy policy actionも `browse` である。

しかし両StepのAuthorization requirementは `unresolved` のため、`browse` はaccepted AuthorizationAction authorityではなくtest-only candidateとしてのみ扱う。

R3Kでは:

```text
Business Step scoped action
  -> selected Procedure/Step only

Application Pattern scoped action
  -> every use of the Pattern

explicit cross-view action
  -> explicitly selected Procedure/Step + Pattern edges
```

のreach差をPressureする。

また:

```text
Authorization requirement scope
  != Authorization action scope

Pattern kind = browse
  + policy action = browse
  != accepted AuthorizationAction = browse
```

を維持する。

current `recent_orders` caseはsemantic scope pressureには使えるが、repository-backedな第二のrequired Authorization interactionではないため、production action locusを選ぶEvidenceにはしない。


### R3L — Authorization action locus Evidence Gate

R3KでBusiness Step / Application Pattern / explicit cross-viewのaction scope差をPressureした後、production locusを選ぶ前提となるqualifying Evidence条件を明示する。

正本:

- `plan/ACTOR_CONTEXT_R3L_AUTHORIZATION_ACTION_LOCUS_EVIDENCE_GATE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3l_authorization_action_locus_evidence_gate_pressure_test.go`

qualifying protected interactionには少なくとも:

```text
Authorization requirement = required
protected resource = explicit authority
Authorization action = explicit Business/Application authority
repository-backed
provider-independent meaning
```

を要求する。

さらにproduction locus比較を開くには、二つ以上のqualifying interactionと、異なるPattern kind / action語彙 / reuse shape / Business-context variation等のlocus-discriminating variationが必要とする。

current Referenceでは:

```text
BP-06
  required = yes
  resource authority = yes
  action authority = unresolved

BP-00 / FU-00
  requirement = unresolved
```

なので、Action Locus Evidence Gateは `insufficient-evidence` のままである。

Business wording / Pattern kind / Legacy policyのhintを不足authorityの補完に使わず、一つのprotected interactionだけから再利用可能なproduction locus ruleを作らない。

```text
action-locus-pressure-qualified
  != locus winner selected
```

も維持する。




### R3M — Operational stop explanation / handoff boundary

R3Lでaction-locus Evidence不足のためproduction locus選択を停止した後、AI Business Actorが未解決authorityで止まった事実を、別authorityを発明せず外部へ説明するProjectionをPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3M_OPERATIONAL_STOP_EXPLANATION_HANDOFF_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3m_operational_stop_explanation_pressure_test.go`

current BP-06では:

```text
requirement = required
resource = order_entry.submit_order
action authority = unresolved
```

なので、exact blocker、known factsとprovenance、drill-down refs、downstream not-attemptedを保持する。

```text
resolution need class
  != missing value
  != resolution owner

resolution requested
  != handoff recipient known

downstream not attempted
  != downstream gap promoted to current blocker
```

を維持し、Business wording / Pattern kind / Legacy policyから `execute` を補完せず、security team / sales operator / application owner等をhandoff ownerとして推論しない。

R3M Projectionはderived / non-authoritativeであり、production ticket routing / human task assignment / notification transportは追加しない。


### R3N — Stage-aware operational stop taxonomy

R3Mでcurrent BP-06のaction-authority stopを正確に説明できることを確認した後、異なる停止状態を一つのgeneric Authorization errorへ潰さないtaxonomyをPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3N_STAGE_AWARE_OPERATIONAL_STOP_TAXONOMY_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3n_operational_stop_taxonomy_pressure_test.go`

R3Nでは少なくとも:

```text
requirement authority unresolved
principal unresolved
resource authority unresolved
action authority unresolved
evaluation not attempted
evaluation observation mismatch
evaluation denied
current-context evaluation unresolved
```

を別のstop meaningとして保持する。

特に:

```text
authority unresolved
  != evaluated denied

observation mismatch
  != denied

prior-context reuse unresolved
  != not evaluated
```

を固定する。

上流blockerが残る場合はearliest blockerだけをcurrent reasonとして保持し、後段は `downstream not attempted` として分離する。

resolution need classは示してもmissing valueやresolution ownerを推測せず、handoff owner / next action / Authorized / CanExecute等のshortcutを追加しない。


### R3O — Audience-neutral stop projection core

R3Mのprovenance付きstop explanationとR3Nのstage-aware taxonomyから、Human / Agent surfaceが共有できるtest-only semantic coreをPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3O_STOP_PROJECTION_CORE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3o_stop_projection_core_pressure_test.go`

共通candidateは:

```text
Business address
stop stage / class
machine reasons
known authority facts + provenance
resolution need classes
runtime outcome
downstream not attempted
source refs
resolution requested
handoff recipient status
authority limits
```

である。

Human表示文はderived renderer detailとし、文言が変わってもsemantic coreを変えない。Agent viewにはHuman display textやNextAction / HandoffOwner / CanExecute等をauthorityとして持ち込まない。

```text
shared semantic core
  != shared presentation
  != stable production API

human wording
  != machine reason
  != agent directive
```

exact current `denied` Observationはruntime outcomeとして保持し、resolution needを生成しない。R3M / R3NがBusiness address・reason・resolution needで矛盾する場合はsilent mergeせずfail closedする。

production `ActorContext` schema / UI message schema / Agent protocolは追加しない。


### R3P — Purpose-bound stop projection / explicit omission boundary

R3Oのaudience-neutral semantic coreから、Human / Agent向けにpurpose-boundなbounded projectionを作る際のomission boundaryをPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3P_PURPOSE_BOUND_STOP_PROJECTION_OMISSION_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3p_stop_projection_omission_pressure_test.go`

R3Pでは:

```text
omitted
  != absent

selected empty
  != omitted

unresolved
  != omitted
```

を固定する。

Human orientation viewはstop identity / machine reason / resolution need / runtime outcome / source refsを選択し、known facts / downstream detail / handoff status / authority limits等をinlineから省略できる。ただし `OmittedCategories` と `DrillDownRefs` を保持し、detailが存在しない意味へ変換しない。

Agent resolution-inspection viewはtest-onlyにfull machine coreを保持するが、Human `DisplaySummary` やNextAction / HandoffOwner / CanExecute等は持たない。

BP-06 action-authority stopでは `runtime-outcome` はselectedだが値は空であり、これはomittedではない。exact denied caseでは `resolution-needs` はselected + emptyであり、runtime denialをmissing authorityへ変換しない。

selected / omitted category partitionは完全かつ排他的であることを要求し、unknown / overlap / missing categoryをfail closedする。

production audience profile / disclosure policy / wire schema / UI layoutは追加しない。


### R3Q — Purpose-bound projection selective freshness

R3Pのselected / omitted category boundaryをbounded projectionのfreshness dependencyへ接続する。

正本:

- `plan/ACTOR_CONTEXT_R3Q_PROJECTION_SELECTIVE_FRESHNESS_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3q_projection_selective_freshness_pressure_test.go`

test-onlyではselected categoryごとにsemantic fingerprintを作り、RequiredCategoryFingerprintsへ含める。

```text
projection freshness
  = profile identity
  + selected semantic dependencies

  != all upstream artifact bytes
```

Human orientationでomitされた `known-authority-facts` / `authority-limits` だけが変化してもHuman viewをstaleにしない。一方、それらをselectedするAgent resolution viewではstaleになる。

Humanでもmachine reason / resolution need / runtime outcome / SourceRefs等のselected meaningが変わればstaleになる。SourceRefsはdrill-down integrityの一部としてselected dependencyに残す。

selected / omitted partition自体はProfileFingerprintへ含め、selected-but-empty categoryをomittedへ変えた場合も別profileとして扱う。

semantic category fingerprintはproduction cache key / Canonical versionではなく、production invalidation engineやwatcherは追加しない。


## Cross-cutting R6 — Selective Knowledge Routing

最新技術リサーチのR6を、Phase 3のAuthorization locus探索とは別の横断pressureとして扱う。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6_selective_knowledge_routing_pressure_test.go`

対象はBP-02のread-only knowledge pathとし、current Business Procedure Referenceに存在するApplication Pattern群をavailable knowledgeとして扱う。

最初の比較:

```text
exact deterministic router
  -> BP-02参照Patternだけをselect

broad test router
  -> available Patternを広くselect

both
  -> current task reference + Source Indexでdeterministic qualification
  -> 同じqualified identity
  -> qualified subsetだけをdisclose
```

これにより、

```text
Knowledge available
  != Knowledge selected
  != Knowledge qualified
  != Knowledge disclosed
```

をSource levelでPressureする。

routing方式はSemantic authorityを所有しない。routing score / confidence、類似度、model内部signalをSemantic applicability、Accepted Decision、Authorizationへ昇格させない。

このR6 sourceは、R3のprincipal / Authorization / action-locus Evidence Gateを迂回しない。qualified knowledge identityが得られても、それだけではAgent invocation authorityにならない。

R7 Replayable Behavior EvidenceはFirst Web Targetのactual Functional Evidenceへ、R8 Verified Experienceはactual Execution / Observationへ接続してから進める。static R6 sourceからR7 / R8の成功を捏造しない。

## Phase 4 — side-effect Action

read-only境界が成立した後だけ検討する。

必要条件:

- explicit Capability authority
- least privilege Resource
- stale / idempotency / retry boundary
- Command / Observation separation
- expected Evidence
- human escalation / stop
- audit / provenance

Product GateやBusiness Procedure production authorityを先取りしない。

## Phase 5 — Knowledge portability Reference

SUI Reference Adoption等で得た知識を含め、別Applicationへ次のいずれかを移植する。

- Domain pattern
- Validation / Constraint
- Procedure fragment
- Capability contract
- Infrastructure / Provider mapping knowledge

copy-pasteではなく、

```text
origin knowledge
  + origin dependency / assumption / Variation
  + destination context
  -> applicability / mismatch / override
  -> accepted reused knowledge
```

を確認する。

Phase 5では少なくとも二つのauthority形を比較対象にする。

```text
A. linked reuse
   destinationがorigin knowledgeへのexplicit dependencyを保持
   origin change時はsilent updateせずrequalification / stale判定

B. adopted local variation
   destinationがscoped meaningを自分のauthorityとしてAcceptance
   origin provenanceは保持
   origin changeはdestination authorityを自動変更しない
```

どちらの場合も、

- origin assumptionsをdestinationへ暗黙継承しない
- destination overrideをoriginへ暗黙back-propagateしない
- copy-pasteでorigin / Evidenceを失わない
- origin updateをdestinationへの無条件auto-applyにしない

ことをPressureする。

この比較前にKnowledge Package、inheritance graph、override schema、cross-repository auto-sync protocolをproduction contractとして固定しない。

### Phase 5 R0 — authority / provenance / variation boundary

Knowledge Portabilityの最初の設計基線は `plan/KNOWLEDGE_PORTABILITY_R0_BOUNDARY_2026-09.md` とする。

R0では、新しいCanonical Applicationを追加せず、current `order_entry`とtest-only destination profileから次を分離する。

```text
discovery provenance
  != authority owner
  != destination dependency

linked reuse
  != inherited origin Application

adopted local variation
  != origin Pattern shared authority

provider knowledge
  != Environment binding
```

current Referenceで最初にPressureする対象:

- `customer.customer` — linked reuse candidate
- `master_data.customer` — destination条件付きlinked reuse candidate
- `order_entry.customer_reference` — adopted local variation candidate
- `bind order_entry.master -> master_data.dev/prod` — rebind-required

test-only source:

- `tools/reference-business-procedure-check/knowledge_portability_r0_pressure_test.go`

synthetic destination labelは`repair_intake-test-only`とし、第二ApplicationのProduct採用やCanonical authoringを意味しない。

R0はISSUE-000047のcandidate qualificationやISSUE-000040のApplication compositionを重複実装しない。

責務は、再利用relationのauthority / provenance / freshnessを保持することである。

#### Freshness boundary

linked reuseでは、destinationが実際に依存するshared subjectが変わればrequalification対象になる。

一方、adopted local variationではorigin Pattern変更をdestination authorityのstaleへ自動変換しない。

```text
linked dependency changed
  -> dependency requalification

provenance-only origin changed
  -> provenance drift

provenance drift
  != destination invalid
  != auto patch
```

無関係なorigin Application変更でdestinationをstale-allにしない。

#### R0で追加しないもの

- production Knowledge Package
- inheritance graph
- override / precedence DSL
- cross-repository auto-sync
- second Canonical Application
- origin-to-destination auto patch
- destination-to-origin back propagation
- `portable=true`の一値判定

actual second-Application compositionはISSUE-000040のGate / Referenceへ接続してから行う。

## 不変条件

```text
AI understands != AI authorized
AI autonomous != AI authorized
AI reads Procedure != AI owns Procedure
Agent plan != accepted Business fact
Command accepted != Business completed
Operational Projection != Canonical source
AI memory != Application source of truth
knowledge reused != assumption automatically valid
```

## 非目標

- 特定agent frameworkの採用
- 全Business Procedureの自動実行
- GUI computer-useの排除
- 人間reviewの撤廃
- AI専用Canonical copyの新設
- prompt / trajectoryをBusiness authorityへすること
- token削減率を単独の成功KPIにすること

## 完了Evidence

- [ ] AI Builder re-entryでstructured contextの有効性を比較できた
- [ ] human review projectionがCanonical / Evidenceへ逆追跡できた
- [ ] AI Business Actorがread-only Actionとhandoffを正しく扱えた
- [ ] unauthorized Capability invocationをfail closedできた
- [ ] CommandとObservationを分離した
- [ ] side-effectを扱う場合はretry / unknown / Resource境界を満たした
- [ ] knowledge reuseでassumption / Variation差を明示できた
- [ ] derived AI / operational contextを削除してもCanonical Worldを再構成できた

## 着手順

Phase 0 / 1のread-only設計・fixture準備は先行可能。

Phase 3以降のBusiness Actor Referenceは、ISSUE-000042 / 000048 / 000049等のBusiness Procedure contract / production authority Evidenceを侵食しない範囲で進める。

side-effect Actionは、対応するProduct GateとAuthority境界が成立するまで実装しない。
