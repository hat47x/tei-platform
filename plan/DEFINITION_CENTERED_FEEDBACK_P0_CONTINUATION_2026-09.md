# Definition-Centered Feedback Loop — P0 Continuation 2026-09

Status: **supporting static audit / P0-2〜P0-4 / no new Architecture / no actual Product Evidence**

Primary Architecture: `../docs/definition-centered-feedback-loop.md`  
Primary Plan: `DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`  
P0-1 Inventory: `DEFINITION_DERIVED_ARTIFACT_INVENTORY_2026-09.md`  
Issue: GitHub #570

## 1. 位置づけ

この文書は、mainへ統合済みのDefinition-centered feedback loopのPhase 0について、P0-1 inventoryの後続であるP0-2〜P0-4だけを具体化するsupporting auditです。

新しいArchitecture、Plan、Product Requirement、Gate、PAS、production schemaを追加しません。

P0-1のA〜F inventoryは`DEFINITION_DERIVED_ARTIFACT_INVENTORY_2026-09.md`を正とし、ここへ複製しません。

```text
P0-1 Definition / Derived Artifact Inventory
        ↓
P0-2 feedback routing
P0-3 memory separation
P0-4 Reference Pattern provenance handoff
```

すべてstatic source / artifact reviewです。actual executionやGate readinessを示すものではありません。

## 2. P0-2 — Feedback routing catalog

Feedbackを見つけたsurfaceへそのまま書き戻さず、まずownershipを判定します。

```text
visible symptom
  ↓
identify owner
  ├─ authoritative definition
  ├─ derived projection
  ├─ native asset / mapping
  ├─ realization / environment
  ├─ actual observation
  └─ proposal / cognition / reference
  ↓
change owner only
  ↓
regenerate / reobserve downstream
```

### 2.1 Review vocabulary

これはproduction enumではありません。Reference review用のvocabularyです。

| Feedback example | Nature | Primary owner | Avoid |
| --- | --- | --- | --- |
| Relation cardinalityが業務意味と違う | semantic contradiction | authoritative Semantic Definition | Target UIだけを補正 |
| Function lensにread targetが現れない | sourceは正しいがProjection欠落 | derived Function / dependency projection | Definitionへ重複field追加 |
| native SQL resultとSemantic mappingが一致しない | native / semantic boundary contradiction | native asset + explicit mapping boundary | 名前一致によるimplicit補完 |
| PostgreSQL storage mappingが不正 | Provider realization mismatch | Persistence / Provider realization | business typeをphysical都合へ変更 |
| Web focus orderが不自然 | Target interaction defect | Target realization | Canonical DataへUI field追加 |
| desired Databaseとactual stateが違う | actual-state drift | Observation → owner-specific Proposal | ActualからDesiredへautomatic backfill |
| dependency unavailable | runtime / Resource failure | execution / Resource boundary | Business Resultをsuccessへ丸める |
| relevant existing patternを取得できない | retrieval miss | derived cognition / retrieval | Canonicalへ検索専用tagを追加 |
| similar patternが業務要件に適用不能 | qualification failure | candidate qualification / unresolved | top-1をautomatic Accept |
| Business Procedure fixtureがsystem resultをactualityとみなす | false completeness | Business Reference / evidence boundary | missing Observationを捏造 |

### 2.2 Routing invariant

```text
where detected != where corrected
```

例:

- Studioで見つかったsemantic defectはStudio metadataへ閉じない。
- AIが見つけたRelation defectはprompt memoryへ閉じない。
- Runtimeが検出したactual driftをSemantic sourceへ自動反映しない。
- Cross-view matrixの欠落を、matrix手編集で修正しない。

最終的なaccepted changeは、そのmeaningを所有するsource / native asset / Bindingへ戻します。

## 3. P0-3 — Memory separation audit

Definition-centered loopでは、記憶を一つのKnowledge Storeへ畳みません。

### 3.1 Authoritative memory

accepted meaning / contract / realization choiceを、それぞれのowner sourceで保持します。

代表:

- Semantic Definition
- Application intent / Pattern
- Database Contract
- native SQL等のauthoritative asset
- Persistence / Realization mapping
- Environment / Resource / Permission choice
- promoted Release snapshot

重要なのは、これらを一つの巨大semantic fileへ集約することではなく、concern-owned authorityをexact linkで接続することです。

### 3.2 Derived structural / associative memory

代表:

- Source Index
- dependency graph
- Data / Function / Business matrix
- normalized lookup index
- Reference Pattern inventory
- FlyHash / HDC / embedding representation
- retrieval ranking cache

原則:

```text
delete derived memory
  -> authoritative meaning remains
  -> derived memory can be rebuilt from identified inputs
```

現行RepositoryではSource IndexをReference snapshotとして保存せず、必要時にCanonical sourceから導出します。この性質を将来のcognitive indexにも維持します。

### 3.3 Episodic / experience memory

代表:

- Proposal history
- accepted / rejected candidate trajectory
- Durable Run / execution episode
- quality / evaluation Evidence
- correction history

過去の成功trajectoryは候補rankや説明へ利用できますが、それだけでcurrent meaning / applicability / execution authorityにはなりません。

```text
past success != current applicability
past acceptance != current execution authority
trajectory != Business Procedure
```

### 3.4 Static audit result

現時点の`order_entry` authoritative definitionへ、FlyHash / embedding / LLM prompt等のcognition専用metadataがmandatoryに混入している状態は確認していません。

また、Source Index等のderived structureが第二semantic sourceとして保存されている状態もありません。

今後のpressure:

- derived / associative index削除後のcold rebuild
- AI provider無しでのsource reload / validation
- stale indexの検出
- episodic memory削除時のaccepted meaning invariance

これらのactual EvidenceはGate C前の本auditでは取得しません。

## 4. P0-4 — X-TEA / XEAD Reference Pattern handoff

X-TEA / XEAD系Reference Modelから得た反復構造は、TEIの自動ruleやCore primitiveへ直接変換しません。

利用目的は、**provenance付きの比較可能なcandidateを表面化すること**です。

### 4.1 Candidate corpusの最低情報

```text
pattern identity / label
provenance model / source
observed vs candidate nature
structural signature
representative examples
applicability question
known non-proof / counter-condition
```

具体的なproduction schemaはまだ固定しません。

### 4.2 Candidate presentation

自然言語intentやcurrent local contextに対して、単一の「正解」を返すより複数候補を比較可能にします。

```text
Candidate A / B / C
  ├─ structure
  ├─ provenance / Reference evidence
  ├─ applicability
  ├─ incompatibility / disadvantage
  ├─ unresolved
  └─ example Dataset / Scenario where useful
```

rankingはAcceptanceではありません。

### 4.3 Function viewの継承

X-TEA / XEADから継承するのは、独立`Function` nounを必ず持つことではありません。

現TEIでは、可能な範囲で:

```text
View / interaction
+ read / CUD
+ Action / Capability
+ call relation
+ input / output
  ↓
derived Function lens
```

を再構成します。

realistic ReferenceでFunction固有meaningがlosslessに保持できないことが示された場合だけauthoring surfaceをreopenします。

### 4.4 Business viewの継承

Task / Business ActionはApplication Functionと同一ではありません。

manual / external work、judgment、Business acceptance、responsibility Handoff等は、実在するなら「端的さ」のために削除しません。

一方、現在のBusiness Procedure Reference fixtureは`canonical=false`であり、RQ-025 / RQ-026 workstreamのauthority判断を先回りしてCanonicalへ昇格しません。

### 4.5 Standard adoption guard

```text
Reference frequency != Product Requirement
pattern match        != semantic applicability
similarity           != accepted relation
retrieval rank       != Decision
candidate            != authority
```

複数domainでdurable semantic needが再現され、既存mechanism / derived lensではlosslessに表せない場合だけ、Core / `std.*` / authoring syntax追加を別判断します。

## 5. P0-2〜P0-4 status

### P0-2 Feedback routing catalog

**Static review vocabulary complete.**

production enum / event schemaは未採用です。

### P0-3 Memory separation audit

**Static architecture audit complete.**

cold reconstruction / deletion invarianceのactual execution Evidenceは未取得です。

### P0-4 X-TEA / XEAD pattern evidence handoff

**Planning-level handoff boundary complete.**

pattern importer、vector index、Pattern Card production schema、特定cognitive algorithmは未実装・未採用です。

## 6. P0全体の収束

P0-1は既存inventory、P0-2〜P0-4は本auditでstatic baselineを持ちます。

```text
P0-1 current inventory
  -> complete at static baseline

P0-2 routing vocabulary
  -> complete at static baseline

P0-3 memory separation
  -> complete at static baseline

P0-4 Reference Pattern handoff
  -> complete at planning baseline
```

ここで「complete」はGate C後のactual closed-loop Evidenceが成立した意味ではありません。

## 7. 次のownership

このP0 completionを理由に、新しいDefinition / AI / cognition workstream branchを増やしません。

Gate C後の実作業は既存ownerへ戻します。

- Studio local authoring: `ISSUE-000046`
- knowledge-first exploration: `ISSUE-000047`
- AI Proposal / Shared Acceptance: `ISSUE-000027`
- derived cognition comparison: `ISSUE-000051`
- Business cross-view: `ISSUE-000042` / RQ-025 / RQ-026

次の順序はPrimary Planを正とします。

```text
Gate B / Gate C mainline
  ↓
Gate C Ready
  ↓
one-change closed loop
  ↓
Reference Pattern candidate source
  ↓
non-LLM cognition comparison
  ↓
actual Observation feedback
```

## 8. Branch convergence

本auditは既存#570 branch上で行い、新branchを作りません。

PR merge後は`CONTRIBUTING.md`のBranch / PR運用に従い、#570 branchを履歴保管場所として残さず、必要ならcurrent mainへ戻します。

Definition-centered Architecture / Primary Plan / P0-1 Inventoryを本書へ複製しないことも、document-sideのconvergence ruleとします。
