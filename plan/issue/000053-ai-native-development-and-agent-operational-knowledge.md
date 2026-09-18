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

source上では、current Pattern / datasource access / Database Contract / Business Reference / intentional conflictを一つのbounded constraint projectionへ集める。R1 probeはこれらdecisive inputのcontent fingerprintを保持し、Business Procedure fingerprintだけをfreshness authorityにはしない。

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

R3 sourceはApplication / Pattern / Contract / Business Procedureというread-routeのactual decisive inputだけをfingerprintし、R1固有のcustomer-edit conflict Evidenceをfreshness dependencyへ持ち込まない。SourceRefsも同じfingerprint input集合からderiveする。

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
