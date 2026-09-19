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


### R3R — Projection dependency provenance / freshness planes

R3Qでpurpose-bound projectionのfreshnessをselected semantic categoryへ絞った後、そのcategoryが何へ依存したかをactual source / runtime Evidenceまで逆追跡する。

正本:

- `plan/ACTOR_CONTEXT_R3R_PROJECTION_DEPENDENCY_PROVENANCE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3r_projection_dependency_provenance_pressure_test.go`

R3Rではfreshnessを一つのhashへ潰さず、少なくとも:

```text
projection profile
semantic Source Index subject
Reference artifact input
runtime Observation
drill-down locator integrity
```

を別planeとして扱う。

Source Index管理下では:

```text
subject identity + content fingerprint
  = semantic freshness dependency

artifact path
  = current drill-down locator
```

とし、locator-only moveではsemantic meaningをstaleにせずlocator refreshだけを要求する。

一方、same subjectでもcontent fingerprintが変わればsemantic staleとする。

Business Procedure Reference / protected-target overlayのようなcurrent Source Index外のReference inputはReference artifact dependencyとして保持し、架空のCanonical / Source Index subjectを発明しない。

また:

```text
static source current
  != runtime Authorization Observation current
```

として、allowed / denied等のruntime outcomeをRepository fingerprintへ混ぜない。

production invalidation engine / runtime Observation store / Business Source Index extension / global revision protocolは追加しない。



### R3S — Projection dependency reason boundary

R3Rでprojection dependencyをprofile / Source Index subject / Reference artifact / runtime Observation / locator integrityへ分離した後、selected categoryが**なぜそのdependencyを必要とするか**をreason付きでPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3S_PROJECTION_DEPENDENCY_REASON_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3s_projection_dependency_reason_pressure_test.go`

R3Sでは:

```text
selected projection category
  -> dependency key
  -> reason code
```

をderived relationとして保持する。

current BP-06では、たとえば:

```text
machine-reasons
  <- Business Procedure Reference
     reason = authorization-requirement-source

machine-reasons
  <- protected-target overlay
     reason = protected-resource-source

source-refs
  <- current locator
     reason = drill-down-locator
```

のように、同じartifactでもProjection上の役割を区別する。

Human bounded viewでomittedされたcategoryのreason edgeはrequired dependencyへ混ぜず、Agent full viewではselected categoryとして保持できる。

runtime outcomeがnon-emptyならexact current Observation reasonを要求し、static sourceだけでruntime outcomeを説明しない。

```text
dependency reason
  != authority edge
  != next action
  != handoff owner
```

を維持し、production dependency graph / impact engine / incremental build engineは追加しない。


### R3T — Projection re-evaluation reason / impact explanation

R3Sでselected categoryからdependency / reasonへ戻れるようにした後、actual input changeが起きたときに「どのcategoryが、なぜ再評価対象になったか」を説明するtest-only pressureへ進む。

正本:

- `plan/ACTOR_CONTEXT_R3T_PROJECTION_REEVALUATION_REASON_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3t_projection_reevaluation_reason_pressure_test.go`

R3Tでは少なくとも:

```text
semantic-source-changed
reference-input-changed
locator-refresh-required
runtime-observation-changed
profile-changed
```

を別のchange classとして扱う。

```text
stale = true
```

だけへ畳まない。

R3S reason edgeを使い、changeがselected categoryへ実際に接続される場合だけre-evaluation reasonを生成する。

Human bounded viewでomittedされたdependency changeは:

```text
no selected-category re-evaluation reason
```

とし、「globally irrelevant」とは扱わない。

locator-only moveでは `source-refs / locator-refresh-required` だけを説明し、semantic machine reasonをstaleへしない。

runtime Observation変化は `runtime-outcome / runtime-observation-changed` へ接続し、static source changeへ読み替えない。

selected / omitted partition変更は `profile-changed` として独立に説明する。

```text
affected
  != automatically rebuilt
```

を維持し、ISSUE-000039のproduction impact engine / invalidation graph / event routingは前倒ししない。



### R3U — Multi-reason re-evaluation aggregation boundary

R3Tでbounded projectionのre-evaluation reasonをchange class / dependency / reason付きで説明できるようにした後、複数changeが同時に発生した場合のaggregationをPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3U_MULTI_REASON_AGGREGATION_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3u_multi_reason_aggregation_pressure_test.go`

R3Uでは:

```text
multiple changes
  -> multiple independent reasons
  -> exact duplicate normalization
  != one winning reason
```

を固定する。

同じcategoryに複数の独立reasonが存在することを許し、semantic source / locator / runtime Observation / profile changeを一つのpriorityへ畳まない。

exact duplicateは:

```text
category
+ change class
+ dependency key
+ dependency reason
```

が完全一致する場合だけ統合する。

また、selected reasonとして説明できたchangeをUnselectedChangeへ二重計上しない。一方、bounded projectionで選択されていないchangeは保持し、「globally irrelevant」へ変換しない。

stable sortはrepresentationの決定性だけに用いる。

```text
stable order
  != priority
  != precedence
```

とし、Winner / PrimaryReason / Severity / FirstCause / AutoRebuild等は導入しない。


### R3V — Cross-audience re-evaluation reason identity

R3Uで一つのbounded projection内のmulti-reason aggregationをlosslessに扱えるようにした後、Human orientation / Agent resolution-inspection間で共有されるreason identityをPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3V_CROSS_AUDIENCE_REASON_IDENTITY_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3v_cross_audience_reason_identity_pressure_test.go`

R3Vではreason identityを:

```text
category
+ change class
+ dependency key
+ dependency reason
```

で構成し、audience / purpose / Human display wording / rendering orderを含めない。

同じBusiness Procedure changeについて:

```text
Human:
  machine-reasons -> affected

Agent:
  machine-reasons -> affected
  known-authority-facts -> affected
```

となる場合、共有されたmachine-reasonsは同じdependency provenance identityへ戻る。一方Agent-only reasonをHumanへ強制開示しない。

```text
shared reason identity
  -> same dependency provenance

different disclosure set
  != inconsistent reason

cross-audience agreement
  != stronger authority
```

を維持する。

dependency keyが異なるreasonは、category / change class / wordingが同じでも同一化しない。

UniversalPacket / PreferredAudience / Winner / StrongerAuthority / NextAction等は導入しない。


### R3W — Re-evaluation reason / category value change boundary

R3T / R3U / R3Vのreason集合を、actual selected category value changeと分離する。

正本:

- `plan/ACTOR_CONTEXT_R3W_CATEGORY_VALUE_CHANGE_BOUNDARY_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3w_category_value_change_pressure_test.go`

```text
re-evaluation reason
  -> what to recompute

R3Q fingerprint comparison
  -> whether selected meaning actually changed
```

とする。

reasonが存在しても再構成後のcategory fingerprintが同一なら `unchanged-after-reevaluation`、異なれば `changed-after-reevaluation` とする。one categoryへ複数reasonがある場合も一つのcategory resultへreason集合を保持する。

selected category fingerprintが変わったのにtracked reasonがない場合はfail closedし、missing dependency / incomplete observation / derivation-rule freshness gapを露出させる。

Human viewでomittedされたcategory変更はselected resultへ昇格させず、profile-changedはsame-profile比較へ混ぜない。

```text
reason exists
  != value changed

value changed
  != automatic rebuild
  != runtime Authorization reevaluation authority
```

を維持し、production rebuild engine / invalidation schedulerは追加しない。


### R3X — Authorization Observation context identity / freshness authority boundary

R3Wでactual selected category changeを確定した後、そのchangeとR3I Authorization Observationのcontext identityを分離する。

正本:

- `plan/ACTOR_CONTEXT_R3X_AUTHORIZATION_CONTEXT_IDENTITY_BOUNDARY_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3x_authorization_context_identity_pressure_test.go`

R3Iのcurrent Observation tuple:

```text
Business address
subject id
action
resource
```

のいずれかが変われば、prior Observationはcurrent-context evidenceではない。

一方、tupleが同じでもprior Observationのfreshness / reuseは自動承認しない。requirement applicability transitionもtuple mismatchとは分離する。

```text
projection category changed
  != Authorization tuple component changed

tuple mismatch
  -> prior Observation is not current-context evidence

tuple match
  != freshness proven
  != reuse authorized

runtime outcome changed
  != context identity changed
```

を維持する。

static semantic change / SourceRefs changeでtupleが同じ場合は `tuple-compatible-freshness-authority-unresolved` とし、prior decision invalidation / reuse / automatic reauthorizationのいずれも選ばない。

production TTL / Authorization cache / revocation watcher / reevaluation schedulerは追加しない。



### R3Y — Authorization re-evaluation / prior Observation reuse authority

R3XでAuthorization Observationのtuple identityとfreshness authorityを分離した後、tuple match時に残る二つのauthorityを別軸としてPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3y_authorization_reevaluation_reuse_authority_pressure_test.go`

R3Yでは:

```text
reevaluation authority
  != prior Observation reuse authority
```

を固定する。

test-only axisは:

```text
reevaluation:
  unresolved
  explicit-reevaluation-required

reuse:
  unresolved
  explicit-reuse-permitted
```

とし、negative/default ruleは作らない。

```text
no reevaluation directive
  != reevaluation not required

no reuse directive
  != reuse prohibited
```

である。

same-tuple static semantic change、target authority revision、provider policy revisionはfreshness pressureとして保持できるが、それだけではreevaluation / reuseのどちらもauthorしない。

requirement applicability transitionはfreshness authorityで解決せず、locator / presentation-only changeをAuthorization freshnessへ逆流させない。

explicit reevaluation directiveとexplicit reuse permissionが同時に存在する場合も、双方のprovenanceを保持してprecedenceをunresolvedとする。

```text
both explicit
  != last-writer-wins
  != more-specific-wins
```

production Authorization cache / TTL / revocation watcher / automatic reevaluation schedulerは追加しない。


### R3Z — Authorization freshness precedence Evidence Gate

R3Yでreevaluation authorityとprior Observation reuse authorityを分離した後、production precedence mechanismを必要とするqualifying conflict Evidenceが実際に存在するかをGateにする。

正本:

- `plan/ACTOR_CONTEXT_R3Z_AUTHORIZATION_FRESHNESS_PRECEDENCE_EVIDENCE_GATE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3z_freshness_precedence_evidence_gate_pressure_test.go`

重要な境界は:

```text
prior-observation-reuse-permitted
  != prior-observation-reuse-sufficient-without-reevaluation
```

である。

したがって:

```text
reevaluation-required
  + generic reuse-permitted
  != proven contradiction
```

とする。

precedence mechanism比較を開くには少なくとも:

- same Authorization tuple
- explicit reevaluation directive
- explicit reuse-without-reevaluation directive
- explicit provenance
- same applicability scope
- repository-backed Business need

を要求する。

Gate outcome:

```text
insufficient-evidence
same-source-compatibility-unresolved
synthetic-conflict-pressure
precedence-mechanism-pressure-qualified
```

same tuple / same scopeで両directiveが同一sourceから出る場合は、cross-authority precedence conflictへ昇格させず、source内compatibility semanticsをunresolvedとして保持する。

ただしqualifiedでもWinner / precedence rule / syntaxは選ばない。

current Referenceにはaccepted production freshness directiveがなく、R3Y explicit directiveはtest-only comparison inputであるため、current Gateは `insufficient-evidence` のままとする。

provider policy revisionもexplicit freshness directiveへ昇格させず、priority / override / nearest-scope-wins / last-writer-wins / TTL precedenceは追加しない。




### R3AA — Authorization freshness authority operational stop / handoff

R3Zのfreshness precedence Gateがcurrent Repositoryでは `insufficient-evidence` のままであるため、same-tuple prior Observationのfreshness authority unresolvedをhidden cache decisionにせずOperational stop / handoffへ戻す。

正本:

- `plan/ACTOR_CONTEXT_R3AA_AUTHORIZATION_FRESHNESS_STOP_HANDOFF_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3aa_authorization_freshness_stop_handoff_pressure_test.go`

test-only taxonomy:

```text
stage = authorization-evaluation-freshness-authority
class = evaluation-freshness-authority-unresolved
reason = authorization-evaluation-freshness-authority-unresolved
resolution need = authorization-evaluation-freshness-authority
```

prior Observationが `allowed` / `denied` でも、current decision evidenceとして利用できるfreshness authorityが未解決ならcurrent runtime outcomeへ昇格させない。

```text
freshness authority unresolved
  -> explainable stop
  -> explicit resolution need
  -> unresolved handoff recipient

  != denied
  != retry-required
  != human-approval-required
  != hidden cache decision
```

R3M explanation / R3N taxonomy / R3O common core / R3P Human・Agent bounded projectionを再利用し、別packetは作らない。

requirement transition unresolvedやObservation tuple mismatch等のearlier / more specific blockerがある場合はfreshness stopへ昇格させない。

production Agent protocol、notification transport、ticket routing、automatic reevaluation schedulerは追加しない。


### R3AB — Cross-audience freshness drill-down provenance

R3AAでfreshness authority unresolvedをOperational stopへ接続した後、Human / Agentが同じshared reasonをdrill downしたときに、同じderived analysis chainとprovenance root identityへ戻れることをPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3AB_CROSS_AUDIENCE_FRESHNESS_DRILLDOWN_PROVENANCE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3ab_cross_audience_freshness_drilldown_provenance_pressure_test.go`

R3X / R3Y / R3Z / R3AAのplan/test artifact自体はauthority rootではなく、derived analysis nodeとして扱う。

```text
freshness stop reason
  -> R3AA Operational stop projection
     -> R3X context identity analysis
     -> R3Y freshness authority analysis
     -> R3Z precedence Evidence Gate
        -> provenance roots
```

provenance rootはtest-onlyに少なくとも次へ分離する。

```text
Reference Evidence
runtime Observation
test-only Authorization context
Legacy policy corroboration
```

Business Procedure Reference / protected-target overlayはReference pressure inputでありproduction Canonical authorityへ昇格させない。prior Authorization Observationはruntime Evidenceであり、freshness authority unresolved中のcurrent RuntimeOutcomeへ昇格させない。R3Xのexplicit action contextはtest-only inputのままにし、Legacy policyはcorroborationに留める。

```text
shared reason identity
  -> same derived chain
  -> same provenance root identity

audience-specific disclosure
  != audience-specific truth

derived analysis
  != authority root

corroboration
  != authority
```

Human compact viewとAgent resolution viewはinline disclosure量が異なってよいが、shared root setを別rootへ差し替えない。

root identityとcurrent locatorも分離し、locator-only moveをdifferent truth rootへ変換しない。

TrustedRoot / CanonicalTruth / PreferredSource / Winner / NextAction / HandoffOwner等は追加しない。


### R3AC — Freshness drill-down root change classification

R3ABでfreshness stopのderived analysisとprovenance rootを分離した後、rootの変更自体も一つのgeneric `source changed` へ潰さずrole-awareに分類する。

正本:

- `plan/ACTOR_CONTEXT_R3AC_FRESHNESS_DRILLDOWN_ROOT_CHANGE_CLASSIFICATION_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3ac_freshness_drilldown_root_change_pressure_test.go`

R3ACでは少なくとも:

```text
reference-evidence-content-changed
runtime-observation-content-changed
test-only-context-changed
corroboration-changed
locator-refresh-required
provenance-root-set-changed
```

を別change classとして保持する。

```text
Reference delta
  != runtime Observation delta
  != test-only context delta
  != corroboration delta
  != locator delta
```

である。

R3ACのsame-root比較addressは `Identity + Role` とし、ClaimKind / fingerprintをrevision側へ置く。same Authorization tupleでObservation claimが `allowed -> denied` へ変わってもroot-set changeへせずruntime Observation changeとして扱う。

同じroot addressでcontent revisionとlocatorが同時に変わった場合は両reasonを保持し、一方へ丸めない。root追加 / 削除もauthority migrationへ推測変換せず `provenance-root-set-changed` とする。

Human / AgentはR3ABのshared root setを使うため、同一root deltaは同じclassificationへ戻る。inline disclosure差をfreshness semanticsへ逆流させない。

change classは観測分類に留め、ShouldReevaluate / ReusePermitted / RetryRequired / CacheValid / AutoRefresh / HandoffOwner等を生成しない。



### R3AD — Provenance root delta / existing projection dependency binding

R3ACのrole-aware root deltaをR3Tへ直接流さず、R3R/R3Sで既に成立しているcurrent projection dependencyとのexplicit bindingを一段挟む。

正本:

- `plan/ACTOR_CONTEXT_R3AD_ROOT_CHANGE_PROJECTION_IMPACT_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3ad_root_change_projection_impact_pressure_test.go`

```text
root delta
  != projection impact

root delta
  + explicit root-to-existing-dependency binding
  + selected R3S dependency reason
  -> R3T bounded re-evaluation reason
```

とする。

R3ADはparallel `provenance-root:*` dependency keyを作らず、Business Procedure Reference / protected-target overlay rootを既存R3R Reference dependency keyへ明示bindingする。

R3ACからR3Tへlosslessに写せる `reference-evidence-content-changed` / `locator-refresh-required` / exact runtime dependencyがある場合の `runtime-observation-content-changed` だけを変換する。

bindingがないdeltaは `projection-dependency-binding-unresolved`、bindingはあってもexact R3T classがないdeltaは `r3t-change-class-unmapped` としてresidual保持する。

prior Authorization Observationはfreshness authority unresolved中のcurrent RuntimeOutcomeではないためcurrent runtime dependencyへ自動bindingしない。test-only Authorization contextとLegacy policy corroborationもroleだけでdependencyへ昇格させない。

bindingがあってもcurrent R3S selected reason edgeがなければR3T `UnselectedChanges` に残す。

```text
unselected
  != unmapped
  != ignored

residual
  != globally irrelevant
```

を維持する。

production dependency registry / impact graph / watcher / scheduler / cache invalidation / auto reauthorizationは追加しない。



### R3AE — Root change / category re-evaluation outcome

R3ADでexplicit existing-dependency bindingを通してselected re-evaluation reasonを得た後、そのreasonをR3Uで正規化し、R3Wのsame-profile category fingerprint比較へ渡す。

正本:

- `plan/ACTOR_CONTEXT_R3AE_ROOT_CHANGE_CATEGORY_VALUE_OUTCOME_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3ae_root_change_category_value_outcome_pressure_test.go`

```text
root delta
  -> explicit existing-dependency binding
  -> selected re-evaluation reason
  -> category recomputation
  -> changed / unchanged after re-evaluation
```

とする。

```text
reason exists
  != category meaning changed

root changed
  != all bound categories changed

category changed
  != response action selected
```

を維持する。

requirement Reference root content changeでfreshness-stop coreのmachine-reasonsが同じままなら `unchanged-after-reevaluation`、実際にcategory fingerprintが変われば `changed-after-reevaluation` とする。

prior Authorization Observationはrefined R3ADでcurrent runtime dependencyへbindingされないため、allowed -> denied revisionも `projection-dependency-binding-unresolved` residualのまま保持し、selected machine-reasons / runtime-outcome resultを生成しない。

locator-only deltaは既存 `source-refs` reasonへbindingされた場合のみR3W比較へ進み、reasonがあってもactual SourceRefs値が不変ならunchanged、actual locatorがprojectionへ反映されればchangedとする。

HumanでomittedされたAgent-only `known-authority-facts` はHuman selected resultへ昇格させない。Agentではselected reasonとactual fingerprint changeがあればchanged resultを持てる。

R3ADのresidual deltaはselected category change authorityではない。ResidualRootDeltaしかないのにselected category fingerprintが変わる場合はR3Wのunexplained category changeとしてfail closedする。

Stale / Fresh / Safe / Reusable / ShouldReevaluate / AutoRebuild / RetryRequired / CacheValid / NextAction等は追加しない。



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

### R6B — heterogeneous knowledge candidate routing

R6の次段では、Application Patternだけでなく異なるauthority / evidence kindを候補poolへ共存させる。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6b_heterogeneous_knowledge_routing_pressure_test.go`

current sourceから次のtask scopeを使う。

```text
BP-02
  Procedure Reference
  + order_entry.customer_reference

BP-05
  Procedure Reference
  + exact Semantic Validation subjects

BP-06
  Procedure Reference
  + security.authorize Semantic Capability

R1 customer-edit review
  Procedure Reference
  + order_entry.customer_reference
  + order-entry-customer-edit-conflict probe Evidence
```

exact routerとbroad routerは異なるcandidate集合を返してよいが、current task relation / Source Index / exact Evidence scopeでqualificationした後の集合は一致しなければならない。

特に、R1のcustomer-edit conflict probeはBP-02のread-only Operational contextへ自動注入しない。逆にR1 reviewではexplicit Evidenceとしてtrace可能に保持する。

```text
available kind diversity
  != authority flattening

Evidence relevant to review
  != Evidence relevant to operation

Capability qualified
  != Authorization evaluated
  != Capability invocation authorized
```

R6BはR3N以降のstop / disclosure / freshness semanticsを置き換えず、どのknowledgeをそのProjectionへ渡す候補にするかという一段手前のrouting boundaryだけをPressureする。

### R6C — qualified knowledge / projection dependency handoff

R6Bのqualified knowledgeと、R3R以降のprojection dependency provenanceを直接同一視しないためのhandoffをPressureする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6c_projection_dependency_handoff_pressure_test.go`

責務分離:

```text
R6 / R6B
  available / selected / qualified knowledge

R6C
  qualified knowledge
    -> current consumerが実際に利用するR3 dependencyへのlink

R3R〜R3X
  dependency provenance
  dependency reason
  re-evaluation reason
  category value change
  Authorization context / freshness authority
```

current BP-06では、Business Procedure Referenceと`security.authorize`がどちらもtask knowledgeとしてqualifiedになり得る。

しかしcurrent Human orientation / Agent resolution-inspection stop projectionがR3Sで実際に説明しているdependencyへだけlinkする。

Business Procedure Referenceは既存のR3 dependency key / reasonを再利用する。

```text
dependency key
  = reference:business-procedure-reference:<locator>

reason
  = authorization-requirement-source
```

Human / Agentでselected categoryが異なる場合もdependency provenanceを作り直さない。Agentだけが`known-authority-facts`を選べても、Humanへ強制開示しない。

一方、`security.authorize`がqualifiedであるという理由だけでR3 semantic dependencyを追加しない。

```text
qualified knowledge
  != selected projection dependency

qualified Capability
  != Capability source required by this projection
  != Authorization evaluated
  != invocation authorized
```

projection reasonが、どのqualified knowledge bindingからも説明できない場合はfail closedする。

R6C自身にはFresh / Stale / Changed / ChangeClass / Authorized / Rebuild等を持たせず、R3のfreshness / impact / authority semanticsを重複所有しない。

### R6D — routed / consumer-local dependency origin

R6Cのhandoffをfull projection reason集合へ適用するには、全dependencyをrouted knowledgeへ要求しない。

current BP-06 Human orientationでは、Business Procedure Reference由来のdependencyはR6B qualified knowledgeへlinkできる。一方、protected-target overlayはcurrent consumerが直接使うReference inputであり、R6 candidate poolへ無理に追加しない。

runtime denied caseのexact Authorization Observationも同様にconsumer-local runtime dependencyとして扱う。

```text
routed-qualified-knowledge
consumer-reference-input
consumer-runtime-observation
consumer-semantic-subject
```

をdependency origin classとしてPressureする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6d_dependency_origin_classification_pressure_test.go`

重要な境界:

```text
qualified knowledge
  != all projection dependencies

consumer-local dependency
  != missing routing knowledge

origin classification
  != authority ownership
  != freshness evaluation
```

一つのdependency keyをroutingとconsumer-localの双方がclaimする場合はambiguousとしてfail closedする。どちらにも分類できないdependencyもfail closedする。

R6DはR3 dependency key / reasonを変更せず、そのinputがroutingから来たかconsumer側の直接入力かだけを説明する。

### R6E — router substitution invariance through dependency origin

R6/R6Bで示したrouter交換可能性を、R6C/R6Dの下流までPressureする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6e_router_substitution_origin_invariance_pressure_test.go`

current BP-06ではexact routerと、意図的に候補を広く返すbroad routerを同じtaskへ適用する。

```text
exact
  -> selected narrow
  -> qualified set Q

broad
  -> selected broad
  -> rejected unrelated candidates
  -> qualified set Q

Q
  -> R6C / R6D
  -> same R3 dependency provenance / origin classification
```

を要求する。

これにより:

```text
selected set may differ

qualified set
dependency reason
dependency origin
  must not differ because router implementation changed
```

を固定する。

protected-target overlayはどちらのrouterでもconsumer-reference-inputのまま、runtime Authorization Observationはconsumer-runtime-observationのまま保持する。

`security.authorize`がqualifiedでも、current R3 reason edgeにbindingされていなければprojection dependencyへ昇格しない。この性質もrouter choiceから独立である。

R6Eではrouting performance / ranking qualityを比較せず、PreferredRouterやproduction router selection policyを決めない。

### R6F — dependency-driven materialization

R6Eまででrouter交換後のqualified / dependency originを安定させた上で、AI / Human consumerへ実際にmaterializeするrouted knowledgeを最小化する。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6f_dependency_driven_materialization_pressure_test.go`

materialization rule:

```text
qualified knowledge
  -> R6D origin classification
  -> routed-qualified-knowledgeとしてprojectionが実際に消費
  -> materialize

qualified but no routed dependency consumption
  -> keep as audit identity
  -> do not materialize
```

current BP-06ではBusiness Procedure Referenceだけがmaterialized対象となり、`security.authorize` Capabilityはqualifiedのままでもcurrent projection payloadへ自動投入しない。

protected-target overlay / runtime Observation等のconsumer-local dependencyは別集合として保持し、router candidate poolを引き込む理由にしない。

```text
available
  != selected
  != qualified
  != materialized

consumer-local dependency
  != routed context payload
```

R6F manifestはUniversal Context Packetではなく、現在のbounded consumerに必要なrouted source identityを再構成するためのderived manifestである。

Prompt本文、token budget、all-skill metadata、routing diagnostics、selected/rejected candidate全量をproduction payload schemaとして固定しない。

### R6G — explainable non-materialization

R6Fでcontext materializationを最小化した結果、qualifiedだがmaterializeされないknowledgeが生じる。R6Gではその状態をsilent omissionにせず、derived witnessとして説明可能にする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6g_materialization_explanation_pressure_test.go`

witnessは:

- knowledge identity
- knowledge kind
- materialized / qualified-not-materialized disposition
- reason code
- materialized時だけexact dependency keys
- SourceRefs

を保持する。

```text
qualified-not-materialized
  reason = no-consumed-routed-projection-dependency
```

は:

```text
missing source
qualification failure
globally irrelevant
```

を意味しない。

current BP-06では`security.authorize` CapabilityはqualifiedのままSourceRefsを保持するが、current bounded projectionがrouted dependencyとして消費していないためknowledge本文をpayloadへ materializeしない。

説明用witnessへContent / Body / Text / Prompt / Payloadを持ち込まず、explainabilityがcontext膨張を逆流させないようにする。

runtime Observationのみをconsumer-local inputとして扱うsliceでは、Business Procedure ReferenceもCapabilityも`qualified-not-materialized`として説明できる。

exact / broad routerが同じqualificationとdependency originへ収束すれば、materialization explanationも一致しなければならない。

R6GはPreferredRouter / Score / Confidence / Fresh / Authorized / NextActionを追加しない。

### R6H — materialization revision provenance

R6Gでmaterialization / non-materializationを説明可能にした後、その判断時に参照したknowledge revisionもtraceできるようにする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6h_materialization_revision_provenance_pressure_test.go`

各knowledge witnessへR6B qualification時の`RequiredInputFingerprints`を接続し、identity / locatorだけでなくcurrent content revisionへ戻れるようにする。

```text
materialization explanation
  + input fingerprints
  -> revision provenance
```

ただしfingerprint差分をR6側で:

- Fresh / Stale
- Changed category
- ShouldReevaluate
- AutoRefresh
- rematerialize / dematerialize decision

へ変換しない。

current BP-06では`security.authorize` Capabilityのfingerprintが変化しても、R3 dependency consumptionが追加されない限り`qualified-not-materialized`のままである。

Business Procedure Referenceのfingerprintが変化した場合も、R6Hはrevision差分を露出するだけで、current routed dependencyが存在するというmaterialization理由自体を自動削除しない。

```text
source revision changed
  != routing qualification automatically revoked
  != materialization decision automatically changed
  != freshness authority resolved
```

actual freshness / category re-evaluation / Authorization freshness authorityはR3R〜R3AF側の責務を利用する。

R6Hはknowledge revisionを追跡可能にするが、global revision token、single current flag、automatic invalidation engineは追加しない。

### R6I — shared knowledge materialization / bounded use edges

R6Hまででqualified knowledgeのmaterialization dispositionとrevision provenanceを分離した。R6Iでは、同じknowledgeが複数のprojection categoryから利用される場合に、knowledge本文を用途ごとに重複materializeしない。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6i_shared_materialization_use_edges_pressure_test.go`

構造:

```text
shared knowledge body
  knowledge identity
  knowledge kind
  input fingerprints
  source refs

bounded use edge
  knowledge identity
  projection category
  dependency key
  dependency reason
```

current BP-06ではHuman / Agentが同じBusiness Procedure Referenceを共有する。

```text
Human orientation
  Procedure body x1
  use: machine-reasons

Agent resolution-inspection
  same Procedure body x1
  use: machine-reasons
  use: known-authority-facts
```

Agent側で利用categoryが増えてもProcedure bodyやsource revision provenanceを複製しない。

同時に`security.authorize` Capabilityは両consumerで`qualified-not-materialized`を維持し、Agentがより詳細なviewを持つことを理由に自動投入しない。

```text
more consumer uses
  != more knowledge authority
  != more source copies
  != execution permission
```

shared knowledge identityからAudience / Purpose / ProjectionCategoryを除外し、consumer-specific useはedgeに閉じ込める。

R6IはR3AG〜R3AIのoutcome materialization / provenance drill-down / read-only Agent consumptionを重複実装しない。R6Iの責務はknowledge input側の重複排除とbounded use provenanceである。

### R6J — bounded input plane assembly

R6Dのorigin分類、R6F〜R6Hのmaterialization / omission / revision provenance、R6Iのshared body / bounded use edgeを一つのderived assemblyへ接続する。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6j_bounded_input_plane_assembly_pressure_test.go`

R6J assemblyは:

1. routed knowledge body
2. routed use edge
3. consumer-local use edge
4. qualified-not-materialized identity

を別planeとして保持する。

```text
routed knowledge body
  != consumer-local input

consumer-local dependency
  != router miss

qualified-not-materialized
  != missing source
```

current BP-06 Human orientationではBusiness Procedure Reference bodyを1回だけmaterializeし、そのProcedureを説明する3つのrouted use edgeを保持する。

protected-target overlayは3つのconsumer-local Reference use edgeとして保持し、KnowledgeKeyを与えない。

runtime Authorization Observationだけのsliceではrouted knowledge bodyを0件にでき、Procedure / Capabilityはqualified-not-materializedのまま、runtime Observationだけをconsumer-local runtime useとして保持する。

consumer-local planeにはrouted knowledge identity / content provenanceを載せないため、assembly段階でorigin boundaryを再び平坦化しない。

R6JはUniversal Context PacketやPrompt payloadを定義しない。assemblyはbounded consumer inputを検証するderived Reference shapeに留め、freshness / retry / authorization / invocation policyを所有しない。

### R6K — explicit bounded input scope

R6Jのassemblyでplaneが空の場合、それをmissing / irrelevant / intentionally excludedのどれかへ暗黙変換しない。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6k_explicit_input_scope_expectation_pressure_test.go`

bounded consumerはknown input planeをすべて:

```text
required
out-of-scope
```

のどちらかとして宣言する。

current classes:

1. routed knowledge body
2. routed use edge
3. consumer-local Reference use
4. consumer-local Runtime use
5. consumer-local Semantic use

validation boundary:

```text
required plane
  + zero observed inputs
  -> fail closed

out-of-scope plane
  + observed input
  -> fail closed
```

一方:

```text
out-of-scope
  + zero observed
  -> intentional bounded absence
  != missing input
```

である。

BP-06 Human orientationはProcedure routingとprotected-target Reference inputをrequiredとし、runtime / semantic consumer-local planeをout-of-scopeとする。

runtime Observationだけのsliceではconsumer-runtime planeだけをrequiredとし、routed planeが0件でもrouting failureとは扱わない。この場合でもProcedure / Capabilityの`qualified-not-materialized` identityは別に保持する。

scope witnessにはglobal `Complete` / `Valid` / `Ready` / `Missing`を追加しない。plane-local expectationとobserved countだけをderived witnessとして保持する。

R6KはR3AJ〜R3ALのauthority-information request / response / evidence qualificationへ踏み込まず、knowledge input boundaryのscope declarationだけを担当する。

### R6L — scope identity / cardinality separation

R6Kでrequired / out-of-scopeを明示した後、そのscope declarationの意味とobserved input countを別軸にする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6l_scope_identity_cardinality_pressure_test.go`

semantic scope identityは各planeの:

- plane class
- scope disposition

だけから導出する。

Audience / Purposeはscope bindingとして別に保持し、ObservedCountはR6K observationとして別に保持する。

current BP-06のshared Procedure useではHumanが1 use edge、Agentが2 use edgeを持つが:

```text
required:
  routed knowledge body
  routed use edge

out-of-scope:
  consumer Reference
  consumer Runtime
  consumer Semantic
```

というdeclarationは同じなのでscope identityも同じである。

逆にconsumer Runtimeをout-of-scopeからrequiredへ変更した場合、observed countに関係なくscope identityは変化する。

```text
cardinality change
  != scope meaning change

scope declaration change
  = scope meaning change
```

scope declarationの入力順序はidentityへ影響させない。

R6L scope identityは`Complete` / `Ready` / `Fresh` / `Authorized`ではなく、bounded consumer input expectationのderived semantic handleに留める。

R3AG〜R3ALのoutput materialization / authority-information lifecycleは再実装しない。

### R6M — same-scope observation delta

R6Lのsemantic scope identityを前提に、同一scope内でobserved plane cardinalityだけが変化した場合の比較をPressureする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6m_same_scope_observation_delta_pressure_test.go`

R6M comparisonはbefore / afterの`ScopeIdentity`一致を必須とする。

一致する場合だけ、各planeについて:

- ScopeDisposition
- BeforeObservedCount
- AfterObservedCount
- CountDelta

をderived observationとして保持する。

current BP-06のHuman / Agent shared Procedure useでは:

```text
same scope identity

routed knowledge:
  1 -> 1
  delta 0

routed use:
  1 -> 2
  delta +1
```

となる。

```text
same scope + cardinality delta
  != scope meaning changed
  != source meaning changed
  != authority strengthened
```

ScopeIdentityが異なる場合はR6M comparisonを拒否し、異なるscope間のcount差を同一意味上のdeltaとして扱わない。

また同一ScopeIdentityにもかかわらず同じplaneのrequired / out-of-scope dispositionが変化していればfail closedする。

R6M observation deltaは`ScopeChanged` / `SemanticChanged` / `Fresh` / `ShouldReevaluate` / `Authorized`等を持たない。

R3AM以降のqualified assertion / Accepted Decision境界とは独立し、knowledge input observationのみを対象とする。

### R6N — same-scope membership delta

R6Mのcardinality deltaだけでは、countが同じままmember identityが置換されたケースを検出できない。R6Nでは同じScopeIdentity内のmember set差分をPressureする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6n_same_scope_membership_delta_pressure_test.go`

member identity:

```text
routed knowledge
  -> KnowledgeKey

routed use
  -> KnowledgeKey + Category + DependencyKey + DependencyReason

consumer-local use
  -> OriginClass + Category + DependencyKey + DependencyReason
```

Human / Agent shared Procedure caseではknowledge memberは同じで、Agent側に追加されたuse edgeだけが`Added`へ現れる。

同数置換も:

```text
count delta = 0
added       = 1
removed     = 1
```

として保持するため、cardinalityだけでは見えないbounded input composition差を失わない。

ただしKnowledgeKeyが同じままfingerprintだけが変化してもmembership deltaにはしない。

```text
revision provenance
  != membership identity
```

revision差はR6H、scope declaration差はR6L、cardinality差はR6M、member set差はR6Nとして責務を分離する。

R6Nは`SemanticChanged` / `ContentChanged` / `Fresh` / `ShouldReevaluate` / `Authorized`を導入せず、member differenceをauthority/freshness判断へ昇格させない。

R3AM以降のqualified authority assertion / acceptanceとは独立する。

### R6O — retained-member provenance delta

R6Nまででscope / cardinality / membershipを分離した後、同一memberが残ったままsource provenanceだけが変化するケースを独立して観測する。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6o_retained_member_provenance_delta_pressure_test.go`

R6Oは同じScopeIdentity内でbefore / after双方に存在するKnowledgeKeyだけを対象とする。

各retained knowledgeについて:

- KnowledgeKey
- KnowledgeKind
- RequiredInputFingerprints
- SourceRefs

のbefore / after snapshotを保持する。

```text
same member
  + fingerprint differs
  -> provenance delta

same member
  + SourceRefs differ
  -> provenance delta
```

ただし:

```text
provenance delta
  != semantic changed
  != stale
  != reevaluation required
```

である。

SourceRefs差をlocator-only change / semantic change等へ分類するのはR6Oの責務ではない。

knowledge memberの追加・削除はR6Nに残し、R6Oはadded / removed memberをrevisionとして扱わない。

routed use memberだけが変化し、knowledge provenanceが同じ場合もR6O deltaは空である。

これにより入力観測の責務は:

```text
R6L  scope identity
R6M  cardinality delta
R6N  membership delta
R6O  retained-member provenance delta
```

と分離される。

R6OはR3AO / R3APのAccepted Decision、authority write、current authority applicationを扱わない。

### R6P — orthogonal observation decomposition

R6M / R6N / R6Oのsame-scope observationを一つのderived comparisonへ接続する。ただしそれぞれの意味を一つのgeneric changeへ丸めない。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6p_orthogonal_observation_decomposition_pressure_test.go`

R6P resultは:

1. CardinalityDeltas
2. MembershipDeltas
3. RetainedKnowledgeProvenance

を別々に保持する。

cardinality / membershipには次の整合条件を要求する。

```text
CountDelta
  == Added count
   - Removed count
```

同じmemberをduplicate観測しただけでcountが増えるケースはこの整合条件に反するためfail closedする。

代表的な分解:

```text
fingerprint-only
  count       0
  membership  0
  provenance  1

same-count substitution
  count       0
  added       1
  removed     1
  provenance  0

use expansion
  count      +1
  added       1
  removed     0
  provenance  0
```

またadded / removed knowledge memberをretained provenanceとして扱わない。

これによりR6のinput observationは:

```text
scope meaning
cardinality
membership
retained provenance
```

を別軸のまま比較できる。

R6Pは`Changed` / `SemanticChanged` / `ScopeChanged` / `Fresh` / `Authorized`等を追加せず、R3のfreshness / acceptance / action authorityへ踏み込まない。

### R6Q — router-substitution observation invariance

R6Eで固定したrouter substitution boundaryを、現在のR6P observation decompositionまで再確認する。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6q_router_substitution_observation_invariance_pressure_test.go`

exact / broad routerはpre-qualification selectionが異なってよい。

```text
selected candidates
  may differ

qualified knowledge
  must converge for same task

downstream bounded observation
  must not retain router choice
```

current BP-06ではbroad routerが実際にover-selectし、rejected candidateを持つことを確認したうえで、exact / broad双方のqualified knowledgeから同一bounded input assemblyを得る。

router substitution自体をR6P comparisonへ渡すと:

```text
cardinality delta = 0
membership delta  = 0
provenance delta  = 0
```

である。

さらに同一Procedure fingerprint revisionを双方へ与えた場合:

```text
exact decomposition
  == broad decomposition

count       0
membership  0
provenance  1
```

を要求する。

runtime-only consumer-local Observationもrouter independentであり、broad selectionを理由にrouted knowledgeへ昇格させない。

R6Q downstream observationはRoutingMechanism / PreferredRouter / Winner / Score / Confidence / rejected candidate情報を含まない。

```text
router substitution
  != semantic input change
  != freshness change
  != authority change
```

R3AQ以降のAccepted Decision commit / write authority境界とは独立する。

### R6R — candidate-pool augmentation invariance

R6 selective routingのrich repository / thin bounded context原則を、candidate pool growthに対してPressureする。

test-only source:

- `tools/reference-business-procedure-check/actor_context_r6r_candidate_pool_augmentation_invariance_pressure_test.go`

current BP-06 poolへ、BP-02向けのtest-only unrelated Application Pattern candidateを1件追加する。

exact router:

```text
available +1
selected unchanged
qualified unchanged
disclosed unchanged
```

broad router:

```text
available +1
selected +1
rejected + unrelated candidate
qualified unchanged
disclosed unchanged
```

とする。

broad routerはrepository全体を広く見てもよいが、current deterministic task scopeにないcandidateをbounded contextまで運ばない。

downstreamではbaseline / augmentedについて:

- bounded assembly
- ScopeIdentity
- QualifiedNotMaterialized
- R6P cardinality delta
- R6P membership delta
- R6P retained provenance delta

が不変でなければならない。

```text
repository candidate added
  != bounded knowledge added
  != scope changed
  != source provenance changed
```

CandidatePoolSize / RepositoryKnowledgeCount / OverSelectionCount等はrouting diagnosticsに留め、bounded consumer shapeへ持ち込まない。

このboundaryは将来repository knowledgeが増えることを許容しつつ、working contextの無条件肥大化を防ぐ。

R6RはPhase 5 Knowledge Portabilityの採否、R3 Accepted Decision、Gate C readinessを決定しない。






### R6 Source-pressure exit assessment

R6A〜R6RのSelective Knowledge Routingは、ReferenceのSource / Design Evidenceとして次の責務分離を一巡した。R6Rでは無関係なcandidate pool growthまでdownstream meaningから隔離した。

```text
available != selected != qualified != materialized
router mechanism != qualified meaning != bounded observation meaning
routed knowledge != consumer-local input
knowledge body != use edge
scope identity != cardinality
cardinality != membership != retained provenance
```

R6Qでexact / broad router substitution、R6Rで無関係なcandidate pool augmentationがR6P downstream observationへ漏れないことまで確認したため、具体的defectなしにR6S以降を機械的に増やさない。

未達・非主張:

- actual runtime latency / token / retrieval quality Evidence
- AI Builder / Agent task success Evidence
- embedding / model-native / Jev等のrouter比較
- production Context Packet / prompt serialization
- freshness / acceptance / execution authority
- R7 replayable behavior actual Evidence
- 新Requirementへの昇格

R6再開は、actual consumerで境界違反が観測された場合、production router候補をR6 invariantsへPressureする場合、またはGate C actual EvidenceからR7へ接続する場合に限定する。

現時点ではR6成果を既存原則のReference Evidenceとして保持し、R3 authority chainとFirst Web Target / Gate Cの並行レーンを重複実装しない。


### R7A — Gate C actual Evidence entry boundary

R7 Replayable Behavior EvidenceはFirst Web Targetのactual Gate C Evidenceから開始し、static sourceだけからreplay successを作らない。

test-only source:

- `tools/reference-business-procedure-check/replayable_behavior_r7a_gate_c_entry_pressure_test.go`

R7Aではactual Gate C readiness / Semantic-Functional Evidenceを受けるための入口だけをPressureする。

必要条件:

```text
GateCReady = true
all readiness checks passed
no blocking unresolved
same Handover Baseline B
same baseline revision R
same Target build T
same Target Adapter / Host
same Semantic / Functional run identity
exact raw artifact SHA-256
```

第一Web Targetではread / success / Authentication denial / Authorization denial / Validation not-satisfied / Execution Errorを別behavior candidateとして保持し、各candidateを元raw artifactへ戻せるようにする。

```text
semantic operation
  != target realization

target realization
  != Canonical selector / coordinate / click sequence

Gate C Ready
  != replay executed
  != replay passed
  != whole workflow passed
```

R7Aのtest-only candidateは常に `not-executed` であり、Source Evidenceをactual Replayable Behavior Evidenceへ格上げしない。

actual Gate C readiness artifactが存在するまでR7B replay executionへ進めない。

#### R7A operational handoff hardening

First Web Target並行レーンでR7A handoff sourceが具体化されたことを受け、R7A側のtest-only entry helperも同じfail-closed条件へ揃える。

追加Pressure:

- readiness checksが空ならreject
- unnamed / failed readiness checkをreject
- blocking unresolvedだけでなくunnamed unresolvedもreject
- Target Difference Handoffが `order_entry` とexact Target Adapter / Hostを指さない場合reject
- Semantic / Functional summaryが `GateCReady=true` を自己申告する場合reject
- B / R / T / Target / run provenanceが空のまま相互一致しているだけの入力をreject
- same-run artifact digestの不完全recordを黙ってskipしない

これにより、syntheticな「両方空だから一致」「checkが無いから全check pass」「使わないartifactだからdigest不完全でも無視」というshortcutをR7入口から除く。

この変更はR7Aの入口整合性だけであり、R7B actual replay executionを前倒ししない。

#### R7A final-readiness provenance closure

R7A entryをfinal Gate C readiness artifactへexactに結び付ける。

追加Pressure:

- current final readiness auditのrequired 10 check identityを完全集合として要求
- missing / duplicate / unknown readiness checkをreject
- `VisualRun` をnon-empty provenanceとして要求
- final readiness artifact exact-bytes SHA-256を要求
- Semantic / Functional summary exact-bytes SHA-256を要求
- replay candidateへ上記provenanceを保持

```text
GateCReady=true
  != readiness Evidence identity established

VisualRun present
  != Visual result is Functional replay result

readiness artifact digest
  + Semantic / Functional summary digest
  + raw artifact digest
  -> replay candidate provenance chain
```

R7A側はfinal readiness判定を再実装せず、第一Web Target final auditが出すcheck contractとartifact identityをfail closedで受領する。actual handoff sourceの統合・actual artifact取得まではR7Bへ進めない。

#### R7A actual handoff source integration

First Web Target Gate C Readyが`T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`で正式記録されたため、actual Evidence JSONをR7へ渡すsourceをR7A final contractへ揃える。

source:

- `tools/reference-first-web-target-r7a-handoff/main.go`
- `tools/reference-first-web-target-r7a-handoff/main_test.go`

handoff CLIはfinal readiness / Semantic-Functional summaryのexact input bytes SHA-256を計算し、candidate provenanceへ保持する。

同時にfinal readinessのrequired 10 checksをexact setとして検証し、missing / duplicate / unknown / failed checkをrejectする。

必要provenance:

```text
B / R / T
Target Adapter / Host
Semantic-Functional run
VisualRun
readiness artifact SHA-256
Semantic-Functional summary SHA-256
raw behavior artifact SHA-256
```

candidateは`not-executed` / `behavior-unit`のままである。

```text
handoff source integrated
  != handoff artifact acquired
  != replay execution
  != replay success
```

したがってR7Bは、Repository外に保持されたactual Gate C EvidenceをCLIへ入力し、actual handoff artifactを生成・取得してから開始する。

#### R7B — actual handoff entry gate

PR #817でFirst Web Targetのactual Gate C Readyを取得したため、R7B入口をSource pressureする。

固定済みactual Gate C identity:

- `B = B`
- `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`
- `T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`
- Semantic / Functional run `first-web-target-0ba87a4e0a96`
- Semantic / Functional summary SHA-256 `78c4c3b1031a70eb62683233fb5f1794cf7f71fd8b420ffa741d5ebe3f649d6b`
- final Gate C readiness SHA-256 `b796e38db9a249ef98f2cb600eb354afc70f9adf57d2c83d47116f9968a0ff6c`
- judgment record repository revision `d34edb9273f5f138abcbc23f4dcb000dcb5cdf20`

actual R7A handoff artifactはまだRepository上で取得・fingerprintされていないため、current stateはR7B execution開始前である。

R7B entryは:

- actual handoff artifact exact SHA-256
- exact current B / R / T / Target / Host / run
- exact final readiness / Semantic Functional summary SHA-256
- six First Web behaviorの完全集合
- 各behaviorのexact raw artifact SHA-256
- `ReplayStatus=not-executed`
- `CoverageClass=behavior-unit`

を要求する。

missing / duplicate / unknown behavior、wrong artifact mapping、provenance mismatch、actual handoff digest欠落をrejectする。

R7B execution intakeはactual executionを意味せず、初期値を:

```text
ReplayExecutionStatus = not-executed
RepeatabilityStatus = not-observed
```

とする。

current Repository stateからactual raw Functional artifact digestを推測・再構成しない。Repository外Evidenceへアクセスしてactual R7A handoff artifactを生成・fingerprintできた時点でのみ、次のactual replay executionへ進む。



### R3AF — Category re-evaluation outcome / Operational explanation

R3AEのcategory fingerprint outcomeとR3YのAuthorization freshness authorityを別軸のままOperational explanationへ戻す。

正本:

- `plan/ACTOR_CONTEXT_R3AF_REEVALUATION_OUTCOME_EXPLANATION_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3af_reevaluation_outcome_explanation_pressure_test.go`

```text
category outcome
  = changed / unchanged-after-reevaluation
  + before / after fingerprint
  + exact reason identity

freshness authority status
  = separate Authorization authority question

unselected / residual
  = separately preserved
```

とする。

`unchanged-after-reevaluation` はtracked reasonでcategoryを再計算しmeaningが同じだったことだけを示し、Fresh / Safe / Reusable / prior Observation reuse authorityを意味しない。

`changed-after-reevaluation` もmeaning changeの説明であり、retry / rebuild / reauthorize / handoff / next actionを選ばない。

Human / Agentの共通selected category outcome identityは一致させ、Agent-only `known-authority-facts` の追加outcomeは許す。

R3T UnselectedChangesとR3AD ResidualRootDeltasはcategory outcomeへ吸収せず、global stale/freshへ丸めない。

production UI schema / notification / routing / scheduler / cache policyは追加しない。



### R3AG — Purpose-bound outcome explanation materialization

R3AFのunderlying Operational explanationを再計算せず、Human / Agentのpurposeに応じてmaterializeする。

正本:

- `plan/ACTOR_CONTEXT_R3AG_PURPOSE_BOUND_OUTCOME_EXPLANATION_MATERIALIZATION_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3ag_purpose_bound_outcome_explanation_materialization_pressure_test.go`

```text
materialization
  != machine outcome recomputation
  != authority recomputation

inline disclosure
  != underlying semantic identity
```

とする。

Human / orientationではcategory / status、freshness stop reason、resolution need、reevaluation / reuse authority statusをinlineに保ち、fingerprint / exact reason / unselected detail / residual detail / provenance detailをdrill-downへ送れる。

ただしcompact表示でも:

```text
UnselectedChangeCount
ResidualRootDeltaCount
```

相当の存在情報をinlineに残し、detailを隠したことを「存在しない」へ変換しない。

Agent / resolution-inspectionではexact fingerprint / reason / unselected / residual detailをinlineにmaterializeできるが:

```text
more detail
  != stronger authority
  != remediation authority
  != execution authority
```

である。

Human / Agent双方に存在するcategoryではcategory / statusとunderlying R3AF identityを共有する。Agent-only `known-authority-facts` をHumanへ強制開示しない。

freshness authority statusはHuman / Agent双方でinline保持し、`unchanged-after-reevaluation` をFresh / Safe / Reusableへ読み替えない。

Humanがexact detailをinline省略する場合、existing dependency keyまたはresidual root identityを使うdrill-down traceを要求し、traceがない場合はfail closedする。

production UI component / Agent wire protocol / notification transport / routing engineは追加しない。


### R3AH — Cross-audience outcome drill-down chain

R3AGのpurpose-bound presentationから、shared outcomeのprovenanceを同じchainで辿れることをPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3AH_CROSS_AUDIENCE_OUTCOME_DRILLDOWN_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3ah_cross_audience_outcome_drilldown_pressure_test.go`

```text
summary outcome identity
  -> exact R3AF category outcome
  -> exact R3T reason
  -> existing R3R/R3S dependency key
  -> explicit R3AD root binding
  -> R3AB provenance root
  -> current locator
```

とし、Human / Agentのpresentation差で別provenance chainを作らない。

summary handleはderived identityでありauthorityではない。reasonのdependency keyからrootへ戻る際はR3AD explicit bindingだけを使い、role / locator類似から推測しない。

prior Authorization Observation等のResidualRootDeltaはselected dependency chainへ捏造せず、residual reason -> root address / role -> current locatorという別chainで保持する。

summary handle mismatch、missing dependency key、missing explicit root binding、missing current root / locatorはfail closedする。

```text
traceability
  != authority
  != remediation
  != execution
```

を維持し、production provenance service / global graph / UI routing / Agent invocation protocolは追加しない。



### R3AI — Read-only Agent consumption / remediation non-selection

R3AGのAgent resolution-inspection materializationとR3AH provenance traceをmachine-readableに消費しても、read-only understandingからremediation / invocationを選ばない境界をPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3AI_READ_ONLY_AGENT_CONSUMPTION_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3ai_read_only_agent_consumption_pressure_test.go`

```text
purpose-bound materialization
  + exact provenance trace
  -> read-only Agent understanding

understand unresolved state
  != select remediation

resolution need
  != next action
```

とする。

Agentはfreshness stop reason、resolution need class、reevaluation / reuse authority status、exact category outcome、unselected change、ResidualRootDelta、dependency / provenance traceを保持できる。

ただし:

```text
explicit reevaluation-required
  != this Agent may invoke reevaluation

explicit reuse-permitted
  != protected operation execution authorized

unchanged-after-reevaluation
  != reuse / continue / fresh

changed-after-reevaluation
  != retry / rebuild / escalate
```

である。

prior Authorization Observation no-binding residualをcurrent runtime decisionへ昇格させず、同じmaterialization / traceの反復消費をauthority strengthやconfidence overrideへ変換しない。

R3AI read-only viewはToolCall / InvocationPlan / Command / NextAction / Remediation / Retry / Reauthorize / ApprovalRequest / HandoffOwner / CanExecute / Authorized / Allowed等を持たない。

production Agent protocol / planner / tool router / approval workflow / ticket routing / Command executionは追加しない。



### R3AJ — Authority-information request / recipient non-inference boundary

R3AIのread-only reasoningから外部情報が必要になっても、問い合わせをremediation / action / owner selectionへ変換しない境界をPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3AJ_AUTHORITY_INFORMATION_REQUEST_BOUNDARY_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3aj_authority_information_request_boundary_pressure_test.go`

R3AJは:

```text
R3AI read-only reasoning
  + R3P Agent resolution-inspection view
  -> scoped authority-information request
```

として構成し、両者のResolutionNeedClassesをcross-checkする。

current freshness stopでは:

```text
information need
  = authorization-evaluation-freshness-authority

evidence need
  = authorization-evaluation-freshness-authority

resolution requested
  = true

handoff recipient status
  = unresolved
```

を保持する。

ただし:

```text
request for missing authority information
  != request to execute remediation

recipient status unresolved
  != guessed owner

evidence need class known
  != missing authority value known
```

である。

BusinessAddress / StopStage / StopClass / MachineReasons / known SourceRefs / DrillDownRefsまでをscopeとして保持できるが、scopeをrouting destinationやworkflow stepへ変換しない。

exact denied Observationではresolution need / resolution requestがないためauthority-information requestを生成しない。

R3AI / R3Pのneed不一致、missing recipient status、missing BusinessAddress、missing source/drill-down contextはfail closedする。

Recipient / HandoffOwner / CandidateValue / ToolCall / InvocationPlan / Command / NextAction / Remediation / Retry / Reauthorize / ApprovalRequest / CanExecute / Authorized / Allowed等は追加しない。

production messaging protocol / ticket creation / notification transport / recipient directory / workflow engine / tool invocationは追加しない。





### R3AK — Authority information request identity / response correlation

R3AJのauthority-information requestにsemantic identityとcurrent context revisionを分けて付与し、responseをcurrent requestへcorrelateできるようにする。

正本:

- `plan/ACTOR_CONTEXT_R3AK_REQUEST_CORRELATION_BOUNDARY_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3ak_request_correlation_boundary_pressure_test.go`

```text
RequestIdentity
  = semantic question identity

RequestRevisionIdentity
  = RequestIdentity
    + current SourceRefs / DrillDownRefs
    + HandoffRecipientStatus
```

とする。

routing destination / recipient identity / delivery metadata / lifecycle stateはRequestIdentityへ含めない。

SourceRefs / DrillDownRefsだけが変わる場合は:

```text
RequestIdentity = same
RequestRevisionIdentity = changed
```

となり、旧revisionへのresponseをcurrent requestへsilent applyしない。

test-only lifecycleは `prepared` と `response-correlated` だけを扱う。

```text
prepared
  != sent / delivered / routed / assigned

response-correlated
  != accepted
  != approved
  != sufficient
  != resolved
  != authority accepted
```

response witnessはRequestIdentity / RequestRevisionIdentity / ResponseRef / ProvidedInformationClasses / EvidenceRefsだけを持ち、identityとrevisionがcurrent requestと一致した場合だけcorrelateする。

ProvidedInformationClassesがneedと一致してもEvidence sufficient / authority acceptedとはしない。逆にinformation classが異なってもidentity / revisionが一致していればresponse自体のcorrelationは可能とする。

recipient statusがunresolvedでもcorrelationでき、Sender / RoutingKey / Channel等を推測しない。

production messaging / delivery tracking / recipient directory / ticketing / authority acceptance workflowは追加しない。

次段ではcorrelated responseについて `response evidence observed != authority evidence qualified != Accepted Decision` を維持し、response evidence qualificationの入口だけをPressureする。




### R3AL — Correlated response evidence qualification

R3AKでcurrent request revisionへcorrelateしたresponseのEvidenceRefをqualification入口へ渡す。

正本:

- `plan/ACTOR_CONTEXT_R3AL_RESPONSE_EVIDENCE_QUALIFICATION_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3al_response_evidence_qualification_pressure_test.go`

```text
response evidence observed
  != authority evidence qualified
  != Accepted Decision
```

R3AJ requestのEvidenceNeedClassesとresponseのProvidedInformationClassesのoverlapは記録できるが、class一致だけではevidenceをqualifiedへ昇格させない。

no witnessでは `qualification-unresolved`、explicit test-only qualification witnessがexact EvidenceRefへ存在する場合だけ `explicit-evidence-qualified` とする。

```text
explicit-evidence-qualified
  != authority accepted
  != asserted authority value accepted
  != stop resolved
```

qualification witnessはexact EvidenceRefへだけ適用し、SourceRef欠落、current responseに存在しないEvidenceRef、request identity / revision mismatchはfail closedする。

複数EvidenceRefは独立assessmentとして保持し、一つのqualified evidenceを他へ伝播させない。response correlationが成立してもEvidenceRefsが空ならassessmentは空でよく、generic success / resolvedへ丸めない。

AuthorityValue / AcceptedDecision / Approved / Sufficient / Resolved / Apply / ToolCall / Command / NextAction等は追加しない。

production evidence registry / trust store / signature validation / authority acceptance workflowは追加しない。

次段ではqualified evidenceがauthority valueをassertする場合でも、`asserted authority value != accepted authority value` を維持し、assertion provenanceとAccepted Decisionを分離する。




### R3AM — Qualified evidence authority assertion / Acceptance boundary

R3ALでexplicitにqualifiedされたEvidenceがauthority valueをassertする場合でも、assertionとAcceptanceを分離する。

正本:

- `plan/ACTOR_CONTEXT_R3AM_QUALIFIED_EVIDENCE_AUTHORITY_ASSERTION_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3am_qualified_evidence_authority_assertion_pressure_test.go`

```text
qualified evidence
  -> provenance-bound authority assertion

asserted authority value
  != accepted authority value
  != Accepted Decision
```

test-only assertion witnessはEvidenceRef / AssertionClass / AssertedValue / AssertionSourceRefを持つ。

R3AMでauthority assertionとしてmaterializeできるのはR3AL `QualificationStatus=explicit-evidence-qualified` のexact EvidenceRefだけとし、`qualification-unresolved + assertion witness` はfail closedする。

asserted valueがR3Yの `explicit-reevaluation-required` / `explicit-reuse-permitted` と同じ語彙でも、R3Y current authority statusへ書き戻さない。

複数のqualified Evidenceが同じAssertionClassについて競合する値をassertしてもwinner / last-writer / quorumを作らず独立保持する。exact duplicate assertionだけnormalizeする。

```text
assertion provenance available
  != acceptance authority available
```

AcceptedAuthorityValue / AcceptedDecision / AcceptanceStatus / CurrentAuthority / ToolCall / Command / NextAction / CanExecute等は追加しない。

次段ではassertionをAccepted authorityへ昇格させるためのRepository-backed acceptance authority / decision locusをEvidence GateとしてPressureし、scope specificity / corroboration count / source locationによる暗黙Acceptanceを禁止する。




### R3AN — Assertion acceptance authority / decision locus Evidence Gate

R3AMのqualified authority assertionをAccepted Decision候補へ進める前に、Repository-backedなacceptance authority / decision locusが存在するかをEvidence GateでPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3AN_ACCEPTANCE_AUTHORITY_EVIDENCE_GATE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3an_acceptance_authority_evidence_gate_pressure_test.go`

```text
assertion provenance available
  != acceptance authority available

qualified assertion
  + repository-backed acceptance authority evidence
  -> acceptance authority evidence qualified

acceptance authority evidence qualified
  != assertion accepted
  != Accepted Decision
```

qualifying evidenceには少なくとも:

- repository-backed evidence
- explicit `assertion-acceptance-authority` declaration
- exact AssertionClass coverage
- explicit BusinessAddress scope coverage
- explicit DecisionLocusRef
- explicit accept / reject semantics

を要求する。

AssertionSourceRefやqualification provenance、Business Procedureのoperational need、一般的Shared Acceptance概念だけからacceptance authorityを推測しない。

current inspected corpusではこれらを同時に満たすsourceを確認できないため:

```text
acceptance-authority-unresolved
```

を維持する。これはassertion rejected / incorrectを意味しない。

exact-business-addressとany-business-addressの双方がqualifyしても:

```text
acceptance-authority-multiple-qualified-precedence-unresolved
```

とし、scope specificityをwinner / precedence authorityへ変換しない。

AcceptedAuthorityValue / AcceptedDecision / AcceptanceDecision / SelectedAuthority / Winner / CurrentAuthority / ToolCall / Command / NextAction / CanExecute等は追加しない。

次段ではacceptance authority evidenceがqualifyしたという仮定の下で、explicit acceptance witnessをexact AssertionIdentity + exact acceptance authority evidence + acceptance provenanceへbindingし、accept / rejectをassertion内容やscope specificityから推測しない境界をPressureする。



### R3AO — Explicit acceptance witness / exact binding boundary

R3ANでacceptance authority evidenceがqualifyしても、それだけでassertionをaccept / rejectしない。

正本:

- `plan/ACTOR_CONTEXT_R3AO_EXPLICIT_ACCEPTANCE_WITNESS_BOUNDARY_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3ao_explicit_acceptance_witness_boundary_pressure_test.go`

```text
single-qualified acceptance authority evidence
  + exact AssertionIdentity
  + exact AcceptanceAuthorityEvidenceID
  + explicit accept / reject
  + AcceptanceProvenanceRef
  -> provenance-bound acceptance witness observation
```

とする。

current Repository-backed R3AN stateは `acceptance-authority-unresolved` なので、current corpusだけからacceptance witnessを生成しない。accept/reject caseはsynthetic complete acceptance-authority evidenceによるPressureである。

witnessが一つのcandidateを名指ししてもR3ANの `multiple-qualified-precedence-unresolved` を迂回させない。no witnessではsingle-qualified authorityがあってもdecisionを推測しない。

accept / reject witnessが競合しても両方をattributed observationとして保持し、winner / last-writer / quorumを作らない。exact duplicateだけnormalizeする。

```text
observed accept witness
  != Accepted Decision applied
  != current authority

observed reject witness
  != Authorization denied
  != RuntimeOutcome = denied
```

AcceptedAuthorityValue / AcceptedDecision / CurrentAuthority / R3Y authority rewrite / ToolCall / Command / NextAction / CanExecute等は追加しない。

次段ではwitness observationをAccepted Decisionとしてauthoritative sourceへcommitするためのseparate write authority / provenance Evidence GateをPressureする。



### R3AP — Accepted Decision authoritative write / provenance Evidence Gate

R3AOのprovenance-bound acceptance witness observationをauthoritative Accepted Decisionとしてcommitできるwrite authority / sink / revision boundaryがRepository上に明示されているかをEvidence GateでPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3AP_ACCEPTED_DECISION_WRITE_AUTHORITY_EVIDENCE_GATE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3ap_accepted_decision_write_authority_evidence_gate_pressure_test.go`

```text
acceptance witness observed
  != Accepted Decision committed

acceptance witness observation
  + repository-backed write authority evidence
  -> Accepted Decision write authority Gate

Gate qualified
  != witness committed
  != Accepted Decision created
  != current authority applied
```

一般的なADR-0022 / Shared AcceptanceやBusiness authoringのstale-check precedentを、Operational Authorization assertionのAccepted Decision write authorityへ暗黙転用しない。

qualifying evidenceには少なくとも:

- repository-backed evidence
- explicit `operational-accepted-decision-write-authority` declaration
- exact AssertionClass / witness Decision / DecisionLocus applicability
- explicit BusinessAddress scope
- explicit authoritative AcceptedDecisionSinkRef
- explicit RevisionAuthorityRef + CurrentDecisionRevision
- explicit WriteProvenanceRef

を要求する。

current inspected Repository evidenceではこれらを同時に満たすOperational write locusを確認できないため:

```text
accepted-decision-write-authority-unresolved
```

を維持する。これはR3AO witness observationがinvalid / rejectedという意味ではない。

complete synthetic candidateでGateをpressure-qualifiedにできるが、複数candidateがqualifyする場合はprecedence unresolvedとし、scope specificity / source location / revision値 / corroboration countからwinnerを選ばない。

revision evidenceを保持してもcompare-and-write / locking / transactionは実行しない。

AcceptedDecision / CommittedDecision / CurrentAuthority / R3Y authority status / RuntimeOutcome / ToolCall / Command / NextAction等は追加しない。

次段ではsingle-qualified write authorityをsyntheticに与え、explicit commit witness + exact R3AO WitnessIdentity + exact write-authority evidence + expected decision revision + write provenanceからtest-only commit candidateを作る境界をPressureする。

```text
commit candidate
  != committed Accepted Decision
  != current authority applied
```



### R3AQ — Accepted Decision explicit commit witness / candidate boundary

R3APでAccepted Decision write-authority Evidence Gateがsingle-qualifiedとなるsynthetic pressureでも、それだけではauthoritative writeを実行しない。

正本:

- `plan/ACTOR_CONTEXT_R3AQ_ACCEPTED_DECISION_COMMIT_CANDIDATE_BOUNDARY_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3aq_accepted_decision_commit_candidate_pressure_test.go`

```text
explicit commit witness
  + exact R3AO WitnessIdentity
  + exact single R3AP write authority
  + exact AcceptedDecisionSinkRef
  + matching ExpectedDecisionRevision
  + CommitProvenanceRef
  -> test-only commit candidate

commit candidate
  != committed Accepted Decision
  != authoritative sink mutation
  != current authority applied
```

current Repository-backed R3AP stateは `accepted-decision-write-authority-unresolved` のため、current corpusだけからcommit candidateを生成しない。

commit witnessはWitnessIdentity / WriteAuthorityEvidenceID / AcceptedDecisionSinkRef / ExpectedDecisionRevision / CommitProvenanceRefを明示し、R3AO AcceptanceProvenanceRefやR3AP WriteProvenanceRefからcommit intentを補完しない。

write authorityがmultiple-qualifiedならcandidate指定でprecedenceを迂回しない。ExpectedDecisionRevisionがcurrent decision revisionと一致しなければstaleとしてfail closedする。

accept / rejectはR3AO witnessのDecisionをそのまま保持するが:

```text
accept commit candidate
  != authority value applied

reject commit candidate
  != Authorization denied
```

とする。

exact duplicate commit witnessだけnormalizeし、異なるCommitProvenanceRefやExpectedDecisionRevisionをlast-writer / latest revision / majorityで解決しない。

Committed / CommitSucceeded / AcceptedDecision / NewDecisionRevision / CurrentAuthority / RuntimeOutcome / ToolCall / Command / NextAction / CanExecute / Authorized等は追加しない。

production decision store / transaction / CAS / locking / authority application / Command executionは追加しない。



### R3AR — Accepted Decision application authority / current revision Evidence Gate

R3AQ commit candidateをauthoritative sinkへ適用する前に、application authority / current sink revision / idempotency contract / write outcome contractがexplicitに揃っているかをEvidence GateでPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3AR_APPLICATION_AUTHORITY_REVISION_GATE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3ar_application_authority_revision_gate_pressure_test.go`

```text
commit candidate prepared
  != write authorized now

R3AQ commit candidate
  + explicit repository-backed application authority
  + exact current sink revision observation
  + idempotency contract locus
  + write outcome contract locus
  -> application-authority Evidence Gate

Gate qualified
  != write attempt
  != Accepted Decision committed
```

R3AP write-authority evidenceやR3AQ CommitProvenanceRefをapplication authorityへ暗黙転用しない。

current inspected corpusではcomplete production application-authority locusを確認していないため:

```text
accepted-decision-application-authority-unresolved
```

を維持する。

qualifying evidenceはexact CommitCandidateIdentity / WriteAuthorityEvidenceID / AcceptedDecisionSinkRef / RevisionAuthorityRefへbindingし、BusinessAddress scope、IdempotencyContractRef、WriteOutcomeContractRef、ApplicationProvenanceRefを要求する。

R3AQ ExpectedDecisionRevisionとattributed current sink revision observationのObservedDecisionRevisionが一致しなければfail closedする。

```text
idempotency contract exists
  != idempotency key assigned
  != duplicate write prevented

write outcome contract exists
  != write outcome observed
```

とする。

multiple-qualified application authorityはprecedence unresolvedとし、scope specificity / revision / source location / corroboration countからwinnerを選ばない。

WriteAttempt / CommitCommand / IdempotencyKey / WriteOutcome / CommitSucceeded / NewDecisionRevision / AcceptedDecision / CurrentAuthority / ToolCall / Command / NextAction等は追加しない。

production transaction / CAS / locking / idempotency implementation / write executor / decision store mutationは追加しない。




### R3AS — Explicit write-attempt witness / idempotency identity boundary

R3AR application-authority Gateがsingle-qualifiedでも自動write attemptへ進まず、explicit write-attempt witnessが存在する場合だけtest-only write-attempt candidateを作る。

正本:

- `plan/ACTOR_CONTEXT_R3AS_WRITE_ATTEMPT_CANDIDATE_BOUNDARY_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3as_write_attempt_candidate_pressure_test.go`

```text
application-authority Gate qualified
  != write attempt created

explicit write-attempt witness
  + exact R3AQ commit candidate
  + exact single R3AR application authority
  + exact revision observation
  + explicit idempotency identity
  + exact outcome contract
  + attempt provenance
  -> test-only write-attempt candidate

write-attempt candidate
  != write execution
  != write outcome
  != Accepted Decision committed
```

current Repository-backed R3AR stateはapplication authority unresolvedなのでcurrent corpusだけからattempt candidateを生成しない。

witnessはCommitCandidateIdentity / ApplicationAuthorityEvidenceID / AcceptedDecisionSinkRef / RevisionObservationRef / ObservedDecisionRevision / IdempotencyContractRef / IdempotencyIdentity / WriteOutcomeContractRef / AttemptProvenanceRefをexplicitに持つ。

```text
idempotency identity present
  != duplicate suppression executed
  != exactly-once delivery

outcome contract bound
  != outcome observed
```

とする。

multiple-qualified R3AR Gateはwitnessで迂回しない。exact duplicate witnessだけnormalizeし、同じcandidate/revisionで異なるIdempotencyIdentityまたはAttemptProvenanceRefが現れた場合はretry / precedence authority未解決としてfail closedする。

WriteExecuted / WriteOutcome / Applied / Conflict / Committed / NewDecisionRevision / AcceptedDecision / ToolCall / Command / Executor / Retry / NextAction等は追加しない。

production writer / transaction / CAS / transport / retry scheduler / idempotency store / decision store mutationは追加しない。



### R3AT — Write-attempt observation / accepted outcome separation

R3AS write-attempt candidateへwriterから返ったと仮定するtest-only observationをbindingしても、raw observationをaccepted write outcomeへ昇格させない。

正本:

- `plan/ACTOR_CONTEXT_R3AT_WRITE_ATTEMPT_OBSERVATION_BOUNDARY_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3at_write_attempt_observation_pressure_test.go`

```text
exact write-attempt candidate
  + attributed writer observation
  + idempotency identity echo
  + attributed post-write revision observation
  -> test-only raw attempt observation

raw attempt observation
  != accepted write outcome
  != Accepted Decision committed
  != current authority applied
```

raw ReportedOutcomeはtest-onlyに:

- writer-reported-applied
- writer-reported-not-applied
- writer-reported-conflict
- writer-reported-unknown

を区別するが、Accepted Decision write outcomeのauthority enumではない。

```text
writer-reported-applied
  != Accepted Decision committed

writer-reported-conflict
  != retry authorized
```

とする。

PostWriteObservedDecisionRevisionはReportedOutcomeと独立axisで保持し、revision changed / unchangedからoutcome acceptanceを推論しない。

WriterObservationRef / PostWriteRevisionObservationRefを必須とし、WriteAttemptCandidateIdentity / sink / IdempotencyIdentityへexact bindingする。

exact duplicate observationだけnormalizeし、ReportedOutcome / post-write revision / observation provenanceが競合する場合はaggregation / acceptance authority未解決としてfail closedする。

AcceptedWriteOutcome / OutcomeAccepted / Committed / AcceptedDecision / NewDecisionRevision / CurrentAuthority / RuntimeOutcome / Retry / ToolCall / Command / NextAction等は追加しない。

production writer / transport / signature validation / outcome acceptance engine / decision store mutation / retry schedulerは追加しない。




### R3AU — Write outcome acceptance authority Evidence Gate

R3AT raw writer observationをauthoritative write outcomeへ進める前に、outcome acceptance authorityがRepository-backedに存在するかをEvidence GateでPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3AU_WRITE_OUTCOME_ACCEPTANCE_AUTHORITY_GATE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3au_write_outcome_acceptance_authority_gate_pressure_test.go`

```text
raw writer observation
  + repository-backed outcome-acceptance authority evidence
  -> outcome-acceptance Evidence Gate

Gate qualified
  != raw outcome accepted
  != Accepted Decision committed
  != current authority applied
```

R3AR WriteOutcomeContractRefやR3AT raw ObservationIdentity自体をacceptance authorityへ昇格させない。

qualifying evidenceはexact WriteAttemptCandidateIdentity / ObservationIdentity / WriteOutcomeContractRef / ReportedOutcome / AcceptedDecisionSinkRef / BusinessAddress scopeへbindingし、OutcomeDecisionLocusRef / OutcomeAcceptanceProvenanceRefを要求する。

writer-reported-applied / not-applied / conflict / unknownはraw classのまま保持し、post-write revision changed / unchangedからacceptanceを推論しない。

multiple-qualified evidenceはprecedence unresolvedとし、scope specificity / source location / corroboration count / revision / raw outcome内容からwinnerを選ばない。

AcceptedWriteOutcome / OutcomeAccepted / AcceptedDecision / CurrentAuthority / RetryEligible / ToolCall / Command / NextAction等は追加しない。

次段ではsingle-qualified outcome-acceptance authorityのsynthetic pressure下でも、explicit outcome acceptance witnessをexact ObservationIdentity + authority + decision + provenanceへbindingし、accepted-outcome candidateまでに留める。




### R3AV — Explicit write outcome acceptance witness / candidate boundary

R3AU outcome-acceptance Gateがsingle-qualifiedでも自動accept/rejectせず、explicit witnessが存在する場合だけtest-only candidateを作る。

正本:

- `plan/ACTOR_CONTEXT_R3AV_OUTCOME_ACCEPTANCE_WITNESS_CANDIDATE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3av_outcome_acceptance_witness_candidate_pressure_test.go`

```text
single-qualified R3AU Gate
  + exact raw ObservationIdentity
  + exact outcome-acceptance authority
  + explicit accept/reject decision
  + AcceptanceProvenanceRef
  -> test-only outcome-acceptance candidate

candidate
  != authoritative outcome applied
  != Accepted Decision committed
  != retry authorized
```

Decisionは `accept-observed-outcome` / `reject-observed-outcome` をtest-onlyにPressureし、raw writer-reported-applied/conflict/unknownやpost-write revisionから推測しない。

multiple-qualified R3AU Gateはwitness指定で迂回しない。exact duplicate witnessだけnormalizeし、decision / acceptance provenanceの競合はfail closedする。

accept candidateからCurrentAuthority適用へ進まず、reject candidateからwrite failed / RetryEligibleへ進まない。

AcceptedWriteOutcome / OutcomeAccepted / AcceptedDecision / CurrentAuthority / RetryEligible / ToolCall / Command / NextAction等は追加しない。

次段ではaccept-observed-outcome candidateが存在しても、post-write authoritative-state reconciliation / revision acceptance / authority applicationを別Evidence GateとしてPressureする。



### R3AW — Post-write authoritative-state reconciliation / revision acceptance Evidence Gate

R3AVの `accept-observed-outcome` candidateが存在しても、outcome evidenceのacceptanceをAccepted Decision state synchronizationへ自動昇格させない。

正本:

- `plan/ACTOR_CONTEXT_R3AW_POST_WRITE_RECONCILIATION_AUTHORITY_GATE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3aw_post_write_reconciliation_authority_gate_pressure_test.go`

```text
accept-observed-outcome candidate
  + repository-backed reconciliation authority evidence
  + exact pre-write revision
  + exact post-write revision observation
  -> reconciliation Evidence Gate

Gate qualified
  != state synchronized
  != revision applied
  != current authority applied
```

とする。

qualifying evidenceはR3AV OutcomeAcceptanceCandidateIdentityだけでなく、R3AT ObservationIdentity、R3AS WriteAttemptCandidateIdentity、R3AQ CommitCandidateIdentity、AcceptedDecisionSinkRef、RevisionAuthorityRef、pre-write revision、PostWriteRevisionObservationRef、post-write revision、BusinessAddressへexact bindingする。

ReconciliationLocusRef / RevisionAcceptanceLocusRef / AuthorityApplicationLocusRefは別々に保持し、一つのauthority shortcutへ畳まない。

current inspected corpusではcomplete reconciliation authority locusを確認していないため:

```text
post-write-authoritative-state-reconciliation-authority-unresolved
```

を維持する。

`reject-observed-outcome` candidateはR3AW reconciliation入口にせず、write failed / rollback / retryへ変換しない。

pre→post revisionがchanged / unchangedのどちらでも、そのrevision関係だけからwrite acceptance / state synchronizationを推論しない。

multiple-qualified evidenceはprecedence unresolvedとし、scope specificity / source path / revision値 / raw outcome / corroboration count / latest evidenceからwinnerを選ばない。

StateSynchronized / RevisionAccepted / AcceptedWriteOutcome / AcceptedDecision / CurrentAuthority / Retry / Rollback / ToolCall / Command / NextAction等は追加しない。

次段ではsingle-qualified R3AW Gateでも自動reconciliationせず、explicit reconciliation witnessからtest-only reconciliation candidateまでに留める。




### R3AX — Explicit post-write reconciliation witness / candidate boundary

R3AW reconciliation authority Gateがsingle-qualifiedでも自動reconciliationせず、explicit witnessが存在する場合だけtest-only reconciliation candidateを作る。

正本:

- `plan/ACTOR_CONTEXT_R3AX_RECONCILIATION_WITNESS_CANDIDATE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3ax_reconciliation_witness_candidate_pressure_test.go`

```text
single-qualified R3AW Gate
  + exact R3AV outcome-acceptance candidate
  + exact reconciliation authority
  + exact post-write revision
  + explicit reconciliation decision
  + provenance
  -> test-only reconciliation candidate

candidate
  != authoritative state synchronized
  != revision applied
  != current authority applied
```

witnessはOutcomeAcceptanceCandidateIdentity / ReconciliationAuthorityEvidenceID / AcceptedDecisionSinkRef / RevisionAuthorityRef / pre-write revision / PostWriteRevisionObservationRef / post-write revisionへexact bindingする。

Decisionはtest-onlyに:

- `accept-post-write-state-reconciliation`
- `reject-post-write-state-reconciliation`

だけをPressureし、revision changed / unchangedやraw writer outcomeから推測しない。

`reject-post-write-state-reconciliation` はrollback / compensation / retry / write failedを意味しない。

multiple-qualified R3AW GateはwitnessでEvidenceIDを指定してもprecedenceを迂回しない。

exact duplicate witnessだけnormalizeし、Decision / ReconciliationProvenanceRef / revision bindingの競合はfail closedする。

StateSynchronized / RevisionAccepted / AcceptedDecision / CurrentAuthority / Retry / Rollback / ToolCall / Command / NextAction等は追加しない。

次段ではreconciliation candidateが存在しても、reconciliation execution observation / post-reconciliation current-state evidenceを別boundaryとしてPressureする。



### R3AY — Reconciliation execution observation / current-state Evidence boundary

R3AX reconciliation candidateの後段でも、reconciliation execution observationとpost-reconciliation current-state observationを別Evidence planeとして扱う。

正本:

- `plan/ACTOR_CONTEXT_R3AY_RECONCILIATION_EXECUTION_OBSERVATION_BOUNDARY_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3ay_reconciliation_execution_observation_pressure_test.go`

```text
reconciliation intent prepared
  != reconciliation execution observed
  != post-reconciliation current-state observed
  != authoritative state current
```

とする。

execution observationはraw class:

- reconciler-reported-applied
- reconciler-reported-not-applied
- reconciler-reported-conflict
- reconciler-reported-unknown

だけを保持し、reported-appliedをStateSynchronizedへ昇格させない。

current-state observationはCurrentStateObservationRef / ObservedDecisionRevision / ObservedStateFingerprintを保持できるが:

```text
revision matches candidate context
  != reconciliation execution proven
  != RevisionAccepted
  != Accepted Decision current
```

である。

execution/current-state observationは互いに独立で、一方から他方を補完しない。

R3AX accept candidateだけをexecution-observation入口にし、reject candidateをrollback / compensation / retryへ変換しない。

両観測はReconciliationCandidateIdentity / AcceptedDecisionSinkRef / RevisionAuthorityRefへexact bindingし、wrong bindingはfail closedする。

exact duplicateはnormalizeできるが、same ObservationRefでoutcome / revision / fingerprintが競合する場合はwinnerを選ばずfail closedする。

current Repositoryではproduction reconciliation executor / current-state observer authorityを確認していないため、R3AY observation caseはsynthetic pressureである。Gate C ReadyやFirst Web Target EvidenceをOperational reconciliation authorityへ転用しない。

StateSynchronized / RevisionAccepted / AcceptedDecision / CurrentAuthority / Retry / Rollback / ToolCall / Command / NextAction等は追加しない。



### R3AZ — Post-reconciliation Evidence acceptance authority Gate

R3AYで分離したexecution observation / current-state observationについて、Evidence acceptance authorityを別GateとしてPressureする。

正本:

- `plan/ACTOR_CONTEXT_R3AZ_RECONCILIATION_EVIDENCE_ACCEPTANCE_AUTHORITY_GATE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3az_reconciliation_evidence_acceptance_authority_gate_pressure_test.go`

```text
raw reconciliation execution Evidence
  != execution Evidence accepted

raw post-reconciliation current-state Evidence
  != current-state Evidence accepted

acceptance authority qualified
  != Evidence accepted
  != authoritative state current
```

とする。

test-only Gateは少なくとも:

```text
ExecutionEvidenceAcceptanceAuthorityStatus
CurrentStateEvidenceAcceptanceAuthorityStatus
```

を別々に持ち、片方のqualificationを他方へ伝播させない。

execution authority evidenceはexact ReconciliationCandidateIdentity / execution ObservationRef / raw execution outcome / sink / BusinessAddress scopeへbindingする。

current-state authority evidenceはexact ReconciliationCandidateIdentity / current-state ObservationRef / observed revision / observed state fingerprint / sink / RevisionAuthorityRef / BusinessAddress scopeへbindingする。

current Repositoryではproduction post-reconciliation Evidence acceptance authority locusを確認していないため、両Gateはunresolvedを維持する。Gate C Ready / First Web Target EvidenceをOperational acceptance authorityへ転用しない。

```text
reconciler-reported-applied
  != accepted execution outcome

current revision == candidate revision
  != accepted current state

current revision != candidate revision
  != retry / rollback authority
```

である。

各planeで複数authority evidenceがqualifyする場合はmultiple-qualified-precedence-unresolvedとし、他planeのsingle-qualified状態やscope specificity / revision / fingerprint / corroboration countでwinnerを選ばない。

ExecutionEvidenceAccepted / CurrentStateEvidenceAccepted / StateSynchronized / RevisionAccepted / AcceptedDecision / CurrentAuthority / Retry / Rollback / ToolCall / Command / NextAction等は追加しない。




### R3BA — Explicit post-reconciliation Evidence acceptance witness / candidate boundary

R3AZのexecution/current-state Evidence acceptance authority Gateがsingle-qualifiedでも、自動accept/rejectせずexplicit witnessが存在する場合だけplane-localなtest-only candidateを作る。

正本:

- `plan/ACTOR_CONTEXT_R3BA_RECONCILIATION_EVIDENCE_ACCEPTANCE_WITNESS_CANDIDATE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3ba_reconciliation_evidence_acceptance_witness_candidate_pressure_test.go`

witness / candidateはEvidencePlaneを明示し:

- `execution`
- `current-state`

を別々に扱う。

```text
single-qualified plane Gate
  + exact R3AY raw Evidence
  + explicit accept / reject witness
  + acceptance provenance
  -> plane-local Evidence acceptance candidate

candidate
  != Evidence accepted authoritatively
  != StateSynchronized
  != CurrentAuthority applied
```

とする。

execution witnessはexact execution authority EvidenceID / execution ObservationRef / raw outcomeへbindingする。

current-state witnessはexact current-state authority EvidenceID / current-state ObservationRef / observed revision / observed fingerprint / RevisionAuthorityRefへbindingする。

wrong plane / authority / observation / raw value / revision / fingerprintはfail closedし、qualified Gateだけではcandidateを自動生成しない。

```text
reconciler-reported-applied
  != accept Evidence

matching current revision
  != accept current-state Evidence
```

Decisionはexplicit witnessだけから取得する。

execution / current-state両方にaccept candidateが存在してもStateSynchronized / RevisionAccepted / AcceptedDecision / CurrentAuthorityへ進めない。

reject Evidence candidateをretry / rollback / compensation / failureへ変換しない。

current RepositoryのR3AZ production acceptance authorityはunresolvedのため、candidate caseはsynthetic single-qualified GateによるPressureである。

EvidenceAccepted / StateSynchronized / RevisionAccepted / AcceptedDecision / CurrentAuthority / Retry / Rollback / ToolCall / Command / NextAction等は追加しない。



### R3BB — Reconciliation result / current-state join authority Evidence Gate

R3BAのexecution / current-state plane-local Evidence accept candidateが両方存在しても、authoritative reconciliation resultへ自動joinしない。

正本:

- `plan/ACTOR_CONTEXT_R3BB_RECONCILIATION_RESULT_ACCEPTANCE_AUTHORITY_GATE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3bb_reconciliation_result_acceptance_authority_gate_pressure_test.go`

entryでは同じR3AX ReconciliationCandidateIdentityに属する:

- execution accept candidate
- current-state accept candidate

がexactly oneずつ必要で、両方とも `accept-reconciliation-evidence` でなければならない。

```text
execution Evidence accept candidate
  + current-state Evidence accept candidate
  != authoritative reconciliation result
  != StateSynchronized
  != CurrentAuthority applied
```

とする。

qualifying authority evidenceはexact ReconciliationCandidateIdentity / execution candidate identity / current-state candidate identity / sink / RevisionAuthorityRef / BusinessAddress scopeへbindingし、さらに:

- ReconciliationResultDecisionLocusRef
- CurrentStateAcceptanceLocusRef
- ResultAcceptanceProvenanceRef

を明示する。

result decision locusとcurrent-state acceptance locusをgeneric authority locusへ畳まない。

current Repositoryではproduction authoritative reconciliation-result/current-state acceptance locusを確認していないため、derived R3BA candidatesだけではGateはunresolvedを維持する。R3AW / R3AZ / Gate C / First Web Target EvidenceをR3BB authorityへ暗黙転用しない。

```text
matching revision
  != result acceptance authority

reported-applied
  != result acceptance authority

both plane candidates accepted
  != result acceptance authority
```

である。

multiple-qualifiedではprecedence unresolvedを維持し、ReconciliationResultAccepted / StateSynchronized / RevisionAccepted / AcceptedDecision / CurrentAuthority / Retry / Rollback / ToolCall / NextAction等は追加しない。



### R3BC — Explicit reconciliation-result acceptance witness / candidate boundary

R3BB Gateがsingle-qualifiedでもreconciliation resultを自動accept/rejectせず、explicit witnessが存在する場合だけtest-only result acceptance candidateを作る。

正本:

- `plan/ACTOR_CONTEXT_R3BC_RECONCILIATION_RESULT_ACCEPTANCE_WITNESS_CANDIDATE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/actor_context_r3bc_reconciliation_result_acceptance_witness_candidate_pressure_test.go`

```text
single-qualified R3BB Gate
  + exact plane-local candidate identities
  + exact qualified result authority
  + explicit accept / reject witness
  + acceptance provenance
  -> test-only reconciliation-result acceptance candidate

candidate
  != authoritative result applied
  != StateSynchronized
  != CurrentAuthority applied
```

とする。

witnessはexact:

- ReconciliationCandidateIdentity
- ResultAcceptanceAuthorityEvidenceID
- execution EvidenceAcceptanceCandidateIdentity
- current-state EvidenceAcceptanceCandidateIdentity
- ReconciliationResultDecisionLocusRef
- CurrentStateAcceptanceLocusRef

へbindingする。

Decisionは `accept-reconciliation-result` / `reject-reconciliation-result` のexplicit witnessだけから取得し、両planeがaccept candidateであること、matching revision、reported-applied等から推測しない。

reject result candidateをretry / rollback / compensation / failureへ変換しない。

multiple-qualified R3BB Gateはwitnessで迂回できず、exact duplicateはnormalizeできる一方、decision / locus / provenance conflictはfail closedする。

current RepositoryのR3BB production authorityはunresolvedであるため、R3BC candidate caseはsynthetic single-qualified pressureである。Gate C / R7A / R7B EvidenceをOperational result acceptance witnessへ流用しない。

ReconciliationResultAccepted / StateSynchronized / RevisionAccepted / AcceptedDecision / CurrentAuthority / Retry / Rollback / ToolCall / Command / NextAction等は追加しない。



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

### Phase 5 Pre-Gate — second Application Reference contract

Knowledge Portability R0 / handoffの次に、Gate C後の第二Application着手条件だけを先に固定する。

正本:

- `plan/SECOND_APPLICATION_REFERENCE_PRE_GATE_CONTRACT_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/second_application_reference_pre_gate_pressure_test.go`

このPressureではcurrent Repositoryから、

- shared Customer Semantic
- compatible Customer Contract
- origin Application Pattern provenance
- origin Environment Binding Evidence

を読むが、第二Canonical Applicationを作らない。

```text
qualified reusable knowledge
  != destination source authored
  != Shared Acceptance
  != Environment binding created
```

を固定する。

Gate C後の着手順は、

```text
ISSUE-000047 qualification
  -> Phase 5 portability relation
  -> ISSUE-000040 composition candidate
  -> Shared Acceptance
  -> destination source
  -> Environment Binding Proposal
  -> Driver / First Web Target
  -> change propagation Evidence
```

とする。

Pre-GateではISSUE-000039のimpact engine、第一Web Target、second Target、write Capabilityを前倒ししない。

### Phase 5 Pre-Gate — schema sufficiency

second Application Pre-Gate contractに対し、新しいPortability schemaを追加せず既存TEI grammarで表現可能かをPressureする。

正本:

- `plan/SECOND_APPLICATION_SCHEMA_SUFFICIENCY_PRESSURE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/second_application_schema_sufficiency_pressure_test.go`

current hypothesis:

```text
Portability planning context
  -> existing Canonical Application / Pattern / Environment source

Portability planning context
  != new Canonical declaration kind
```

AI Builder向けのorigin provenance / reuse relation / unresolvedはderived contextとして保持し、生成されるauthoritative sourceは通常のTEI sourceとする。

actual second Applicationで不足が観測されるまで、Knowledge Package / inherit / override / sync grammarをproduction化しない。

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
