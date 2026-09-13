# Definition-Centered Feedback Loop — P0 Audit 2026-09

Status: **supporting static audit / no new Architecture / no new Gate / no actual Product Evidence**

Primary plan: `DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`  
Architecture: `../docs/definition-centered-feedback-loop.md`  
Issue: GitHub #570

## 1. 位置づけ

この文書は、`DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`のPhase 0を現在のRepositoryへ具体適用したsupporting auditです。

新しいArchitectureや第二の計画正本ではありません。

特に、過去#570で検討した「Definition Body」独立Architectureを並立させず、mainへ統合済みの`definition-centered-feedback-loop`を上位正本として使います。

このauditが扱うのは次だけです。

1. P0-1 current definition / derived artifact inventory
2. P0-2 feedback routing examples
3. P0-3 memory separation audit
4. P0-4 X-TEA / XEAD pattern evidence handoff boundary

すべてstatic source / artifact reviewです。実行したこと、GateがReadyになったこと、actual Evidenceを取得したことを意味しません。

## 2. A〜F分類

Primary planの分類をそのまま利用します。

```text
A. human-authored irreducible meaning
B. mechanically derived meaning / projection
C. native external authoritative asset
D. realization / target / environment-specific detail
E. observed / runtime-only information
F. proposal / evidence / episodic information
```

分類の目的はfile extensionごとに一律の箱へ入れることではありません。

同じRepository内に保存される情報でも、authority natureが異なることを明確にし、同じmeaningを複数classへauthoritativeに複製しないことが目的です。

## 3. P0-1 — `order_entry` inventory

### 3.1 A — human-authored irreducible meaning

現時点でAとして扱うのが自然なsourceは次です。

#### Semantic definition

`reference/workspace/semantic/sales/order.tei`

- `OrderId` / `OrderLineId` / `OrderStatus`のtype family
- `Order` / `OrderLine`のData meaning
- attribute type
- `customer` Relation
- `lines` Relationとcardinality
- customer key mapping
- `sales.order.submit` Capabilityのinput / output

これらを消して名前やSQLから推測すると、accepted meaningの区別が失われます。

`reference/workspace/semantic/sales/validation.tei`

- quantityがpositiveであるというaccepted validation meaning
- subject / expression

Validation messageはhuman-facing説明も含みますが、少なくともvalidation contract自体はAです。

#### Application definition

`reference/workspace/applications/order-entry/application.tei`

- Applicationが利用するSemantic namespace
- logical datasource `sales` / `master`
- target Database contract
- `read-write` / `read` access intent
- required contract / version range

ここでProvider instanceやcredentialを書かず、Application-level requirementだけに留めている点は現設計の良い境界です。

#### Application Pattern / interaction intent

次のPattern sourceはUI component treeではなく、利用者向けinteraction intentの差を短くauthorしています。

- `customer-reference.tei` — customer reference + master datasource
- `order-lines.tei` — header/detail + accepted Relation + sales datasource
- `order-record.tei` — order record + sales datasource
- `recent-orders.tei` — browse + query reference
- `submit-order.tei` — command + Capability + realization selection

これらは現在のReferenceではAです。

ただし「Pattern syntaxに書かれているから永久にA」という意味ではありません。将来、ある項目が別のaccepted sourceからexactly derivableだとReferenceで示せた場合はBへ移せます。

#### Query semantic boundary

`queries/recent_orders.tei`のうち:

- datasource selection
- native SQL assetの参照
- SQL result columnとSemantic fieldの対応

は現時点ではauthoritative mappingとして扱います。

特にresult mappingは、SQL textとDatabase schemaだけから安全に意味を推測できるというEvidenceがまだありません。

したがって**「名前が同じのでderiveできそう」という理由だけで削除しません**。

このmappingの一部を将来Bへ移せるかはP1以後のpressure questionとして残します。

### 3.2 B — mechanically derived meaning / projection

現在Bに置くべきもの、またはBとして扱う方針がすでに明確なもの:

- normalized Canonical World projection
- `reference/world.json` — Workspaceから生成するgolden snapshot
- `.tei/plugin-lock.json` — Workspaceから解決したPlugin Lock golden
- Source Index — current policyではRepositoryへsnapshot保存せず、Canonical sourceから必要時にderive
- Data / Function / Business cross-view matrix
- Function lensのread / create / update / delete / call関係
- dependency / impact graph
- standard inspection projection
- derived cognition用feature / graph / index

重要な点:

```text
B may be durable
  != B is semantic authority
```

`world.json`やPlugin LockをRepositoryに保持していても、それらは生成経路と比較用途を持つgoldenです。手で意味を追加する場所にはしません。

### 3.3 C — native external authoritative asset

`queries/recent_orders.sql`はCの代表例です。

```sql
SELECT ... FROM sales_order ORDER BY ordered_at DESC
```

SQLはSQLとしてauthoritative native assetの役割を保ち、同じstatement bodyをTEI DSLへ再記述していません。

TEI source側には、そのassetをどのQuery / datasource / Semantic resultへ接続するかを置きます。

この分離はRQ-004とdefinition-centered方針の両方に整合します。

将来のGo callback、外部configuration、legacy assetについても、native contentをAI向けに丸ごと別Definitionへ複製しないことを基準にします。

### 3.4 D — realization / target / environment-specific detail

代表:

- `applications/order-entry/realizations/default.tei`
- `applications/order-entry/realizations/submit-order.tei`
- `applications/order-entry/realizations/authenticate.tei`
- `databases/sales-core/database.tei`のProvider selection
- `environments/development/environment.tei`

Environmentには:

- Database instance
- Application datasource binding
- deployment / observation executor
- execution timeout
- lease store
- Resource
- Permission
- permission grant
- sandbox / network / filesystem scope

等があります。

これらは「Definitionを端的にする」ために削除すべきboilerplateではありません。

```text
A business/application meaning
        !=
D execution authority / environment realization
```

という別concernのauthorityだからです。

端的さはDをAへ隠すことではなく、AとDを重複authorしないことで実現します。

### 3.5 E — observed / runtime-only information

代表:

- actual Provider observation
- runtime execution state / actual completion observation

Repository内の:

- `reference/observed/production.deployment-state.json`
- `reference/observed/production.canary.deployment-state.json`

はEそのもののactual production observationではなく、**Eのshapeを再現する固定Reference fixture**です。

したがって:

```text
actual Observation        -> E
observed-state fixture     -> F (E semanticsを再現するReference Evidence/fixture)
```

と区別します。

`environment.tei`にobservation executorが定義されていることもactual observationそのものではありません。

```text
observation capability / wiring (D)
  != observed actuality (E)
```

を維持します。

### 3.6 F — proposal / evidence / episodic information

代表:

- Business Procedure Reference authoring fixture
- observed-state fixture
- Proposal / Preview / Semantic Diff
- AI / human correction candidate
- Evidence artifact
- Durable Run / execution episode
- future trajectory / experience knowledge
- search / cognition candidate and ranking

`reference/business-system-design/order-entry-business-procedure.json`は明示的に:

```text
artifactRole = reference-authoring-fixture
canonical = false
```

であり、trigger / actorにも`candidate` statusがあります。

したがってBusiness意味を検討する価値はありますが、current Canonical Application Definitionと同じAとして扱いません。

同様に`reference/order_rollout.durable-run.m0.21.start.json`は再現可能な重要goldenであっても、accepted business Definitionの正本ではありません。

### 3.7 A〜Fを跨ぐ場合

一つのfileを必ず一classへ固定しません。

たとえばQuery concernは:

```text
TEI-side semantic mapping -> A
native SQL              -> C
actual runtime result   -> E
query execution evidence -> F
```

へ分かれます。

分類単位は必要に応じてfileではなくresponsibilityです。

## 4. P0-1で見つかった良い圧縮

現`order_entry`には、すでにX-TEA/XEAD系Reference Modelの「端的な定義体」に近い性質があります。

### Data / Relation

Data、type、Relation、cardinality、key mappingが短いsourceへ集中しています。

### Function / Interaction

独立した巨大Function定義を持たず:

```text
Pattern / View intent
+ Query
+ Capability
+ Data read/write relation
+ realization reference
```

からFunction lensを再構成できます。

これは「Function視点を捨てた」のではなく、Functionを別の二重管理正本へしない設計です。

### UI

Patternはfull UI treeではなくinteraction intentだけを持ち、Target component / DOM / pixel realizationをsourceへ混ぜていません。

### Native Query

SQL bodyをnative SQLに残し、TEI側では意味との接点を保持します。

### Runtime / Environment

Application sourceへinstance / secret / executor / permissionを埋めず、Environment authorityへ分離しています。

## 5. P0-1で残ったpressure question

### Q1 — Query result mappingのどこまでをderiveできるか

現在は安全側としてAに残します。

将来、次がexactに成立する場合に限り一部derived化を検討できます。

- SQL projectionをparseできる
- Database Contractとcolumn identityをexactに解決できる
- semantic mappingが一意
- alias / expression / join / cast等でも曖昧にならない
- native source revisionをtraceできる

一つでも曖昧ならexplicit mappingを維持します。

### Q2 — Pattern datasourceの重複

Applicationにはlogical datasource requirementがあり、Patternにもdatasourceがあります。

これは直ちに重複ではありません。

Application側は「利用可能なdatasource contract」、Pattern側は「このinteractionがどのlogical datasourceを使うか」を表しています。

ただし、単一候補しかないcaseで毎回明示が必要かはrealistic multi-datasource Referenceで再評価できます。

### Q3 — `submit_order`のrealization参照

PatternがCapabilityとrealizationを両方参照しています。

これは「business/application intent」と「default execution selection」が同じsurfaceへ近接している箇所です。

現段階で削除しませんが、将来:

```text
Pattern -> Capability
Environment / Binding -> Realization
```

だけで必要意味が保てるかをReference pressureとして確認する価値があります。

これはsyntax cleanupではなく、A/D ownershipの確認です。

### Q4 — Business Procedureのauthoritative destination

current Business Procedure ReferenceはFのcandidate/reference fixtureです。

RQ-025 / RQ-026 workstreamでProcedure identity / authorityが最終判断されるまでは、このDefinition feedback loop側からAへ先回りして昇格しません。

## 6. P0-2 — Feedback routing catalog

production enumではなく、current Reference review vocabularyとして次を採用します。

| Feedback例 | 観測 | Primary routing | 誤った修正先 |
| --- | --- | --- | --- |
| wrong Relation cardinality | semantic contradiction | A Semantic Definition | Target UI |
| Function lensにread targetが出ない | sourceは正しいがProjection欠落 | B derived projection | Aへ重複field追加 |
| SQL result semantic mapping不一致 | native/result boundary contradiction | A/C boundary review | AI prompt |
| PostgreSQL datatype実現が不正 | Provider-specific mismatch | D Provider / Realization | business typeをDB都合へ変更 |
| Web focus orderが不正 | Target interaction problem | D Target realization | Canonical Data |
| desired DBとobserved DBが異なる | actual-state drift | E Observation -> correct owner proposal | ActualをCanonicalへ自動backfill |
| dependency unavailable | runtime/resource failure | D/E execution boundary | Business Resultをsuccessへ丸める |
| relevant patternをretrievalできない | cognition miss | F candidate mechanism | Canonicalへ検索tag追加 |
| nearest patternが業務要件と不一致 | unsupported inference | F qualification / unknown | top-1をAccept |
| Business Procedureがsystem resultからactualityを推測 | false completeness | F Business Reference correction | Observationの捏造 |

Routing原則:

```text
visible symptom
  -> identify owning class
  -> change owner only
  -> regenerate / reobserve downstream
```

UIやAIで見つけた問題を、そのsurfaceの都合でAへ書き戻さないことが重要です。

## 7. P0-3 — Memory separation audit

### 7.1 authoritative semantic memory

Authorityを持つのは、accepted source / native authoritative asset / explicit Binding等、それぞれのresponsibilityの正本です。

Semantic meaningについてはAが中心です。

C / Dも各concernではauthorityを持ち得ますが、Aの意味を複製する第二semantic memoryにはしません。

### 7.2 derived structural / associative memory

例:

- Source Index
- dependency graph
- CRUD / cross-view matrix
- normalized lookup index
- Reference Pattern inventory
- FlyHash / HDC / embedding representation
- ranking cache

現在Source IndexはRepository snapshotとして保存せず、Canonical sourceから必要時に導出する方針です。

この性質をcognitive indexにも維持します。

削除後にA/C/D sourceからrebuildできることを要求します。

### 7.3 episodic / experience memory

例:

- Proposal history
- Accepted / rejected candidate trajectory
- execution / Durable Run episode
- Evaluation Evidence
- actual correction history

Episodic memoryから「過去うまくいった」を取り出せても、それだけでcurrent Definitionやauthorityにはなりません。

### 7.4 static audit result

現時点で、AI/cognition専用metadataが`order_entry` A-sourceへmandatoryに埋め込まれている兆候は確認していません。

また、Source Indexが保存snapshotとして第二正本になっている状態もありません。

今後守るinvariant:

```text
delete derived / associative / episodic memory
  -> accepted Definition remains reconstructable
```

このinvariantのactual cold-reconstruction EvidenceはP0 static auditでは未取得です。

## 8. P0-4 — X-TEA / XEAD pattern evidence handoff

過去Reference Model調査から得た反復構造は、TEI standard ruleへ直結させません。

最低限、derived candidate corpusでは次を保持します。

```text
pattern identity / label
provenance model / source
observed or candidate nature
structural signature
representative examples
applicability question
known non-proof / counter-condition
```

### 8.1 Candidate presentation

AI / cognitionが利用するときは、top-1 answerより複数候補を基本にします。

```text
Candidate A / B / C
  structure
  provenance
  applicability
  incompatibility / downside
  unresolved
  example Dataset / Scenario where useful
```

### 8.2 Function pattern

X-TEA/XEADのFunction viewから継承するのは`Function` nounそのものではなく:

```text
View / interaction
read / CUD
call relation
input / output
```

をcross-checkできる性質です。

現TEIで既存情報からlensを再構成できる限り、新しいFunction正本を追加しません。

### 8.3 Business pattern

Task Action / Business ActionをApplication Functionへ畳みません。

manual / external / judgment / Handoffを含むpatternは、Business Procedure workstreamのauthority判断を尊重し、candidate knowledgeからCanonicalへ自動昇格させません。

### 8.4 Standard adoption

Referenceで頻出することはstandard採用の必要条件でも十分条件でもありません。

```text
Reference frequency != requirement
pattern match        != applicability
retrieval rank       != acceptance
```

複数domainのReferenceでdurable semantic needが再現され、既存mechanismでlosslessに表せない場合だけCore / `std.*`採用を別判断します。

## 9. P0 status

### P0-1 Definition inventory

**Static review complete for current `order_entry` baseline.**

残るQ1〜Q4は削除候補ではなく、後続Reference pressureです。

### P0-2 Feedback routing catalog

**Static vocabulary complete.**

production enum / schemaは未採用です。

### P0-3 Memory separation audit

**Static architecture audit complete.**

Cold reconstruction / deletion invarianceのactual executionは未実施です。

### P0-4 X-TEA / XEAD pattern evidence handoff

**Handoff contract complete at planning level.**

Importer、vector index、Pattern Card production schemaは未実装・未採用です。

## 10. 次の順序

このauditを理由にGate C前のAI / Studio実装へ進みません。

```text
current Gate B / Gate C mainline
  ↓
Gate C Ready
  ↓
P1 one-change closed loop
  ↓
Reference Pattern candidate source
  ↓
non-LLM cognition comparison
  ↓
actual Observation feedback
```

Gate C前に追加作業をする場合は、Q1〜Q4について既存sourceから静的に確認できる範囲に限定します。

新しいDefinition syntax、Cognitive Layer、Function kind、Pattern schemaを先行採用しません。

## 11. Branch / document convergence rule

このP0 auditは#570の既存branchで作成しました。

- 同テーマで新branchを作らない
- #571でmainへ入ったArchitecture / Planを正本とする
- #570ではそのP0具体化だけを補完する
- merge後にこのbranchを新しい長期作業laneとして残さない
- P1以後は既存owner Issue（#46 / #47 / #27 / #51等）の責務へ戻す

これにより、Definition-centered designを複数branch /複数Architectureへ分岐させません。
