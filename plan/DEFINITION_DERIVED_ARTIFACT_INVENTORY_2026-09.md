# Definition / Derived Artifact Inventory — 2026-09

Status: **P0-1 baseline audit / Reference-only / no schema change**

Baseline revision: `c4cfc387ff9b8c38386d3c8d7a9c398b391c3e5b`

Related:

- `docs/definition-centered-feedback-loop.md`
- `plan/DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`
- `reference/README.md`
- `docs/architecture.md`
- `docs/business-system-design.md`

## 1. 目的

Definition-centered feedback loopのP0-1として、現行`order_entry` Referenceを中心に、Repository上の定義・native asset・realization・golden・observed fixture・Business Procedure Referenceを分類する。

目的はfile数を減らすことではない。

次を区別する。

```text
人が意味として維持すべき情報
        !=
境界を越えるために明示すべきmapping
        !=
機械的に再生成できるprojection / golden
        !=
外部世界のobservation
        !=
未採用のproposal / Reference pressure
```

この区別なしに「端的な定義体」を追求すると、必要なauthorityやtime boundaryまで暗黙推論へ押し出すfalse compressionが起こる。

本inventoryでは新しいCanonical kind、DSL syntax、production schema、Gate、PASを追加しない。

## 2. 分類軸

P0計画のA〜Fを、実際のReferenceへ次のように適用する。

| Class | 意味 | 原則 |
| --- | --- | --- |
| A | human-authored irreducible meaning / intent / contract / promoted snapshot | 人が採用した意味・境界・時間的決定。単純に再生成物へ落とさない |
| B | mechanically derived meaning / projection / golden | authoritative inputから再生成可能。第二正本にしない |
| C | native external authoritative asset | SQL等をnativeなままauthoritative assetとして保持する |
| D | realization / target / environment-specific choice | Canonical meaningとは別だが、実装・配置・mapping上の明示的選択なのでauthorする |
| E | observed / runtime-only information | Actual / Observation。DesiredやSemanticへ自動逆書きしない |
| F | proposal / evidence / episodic / non-canonical Reference | 候補・検査・履歴。Acceptance前はauthorityにしない |

### 2.1 再生成可能性だけでauthoring要否を決めない

次の四問で判断する。

```text
Q1. 別のaccepted inputからexactに再構成できるか
Q2. 再構成すると、人が選んだ境界・実現・公開Contractが失われないか
Q3. concern境界を越えるmappingそのものが意味を持つか
Q4. current stateではなく、release等の過去時点を凍結したauthorityか
```

`Q1 = yes`でも、Q2〜Q4のいずれかがyesなら、人が保持すべき定義である可能性が高い。

したがって、`duplicate text == duplicate authority`ではない。

## 3. Current Referenceのauthority map

大きく次の流れとして読める。

```text
Semantic / Business meaning
        │
        ├─ Application intent / Pattern / Contract
        │
        ├─ native SQL
        │
        ├─ Persistence / Query / Capability realization
        │
        ├─ Environment / Resource / Permission choice
        │
        └─ Release-time promoted snapshot
                ↓
          normalized World
                ↓
       derived index / golden
                ↓
      runtime / observed state
```

Business Procedure Referenceは現時点ではこのauthoritative chainへ未採用であり、別のReference / Pressure planeにある。

## 4. A — Human-authored irreducible meaning

### 4.1 Semantic definitions

代表:

```text
reference/workspace/semantic/sales/order.tei
reference/workspace/semantic/sales/validation.tei
reference/workspace/semantic/customer/...
reference/workspace/semantic/security/...
```

`sales/order.tei`は少なくとも次を直接authorする。

- business type
- Data identity
- Attribute
- Relation
- cardinality / relation key
- Capability signature

これらをPersistence tableやUIから逆推論する設計には戻さない。

#### 判定

**Keep authored.**

理由:

- business identity / relation / capabilityは実装技術から独立した意味である
- physical schemaからのreverse inferenceではBusiness intentと偶然の実装を区別できない
- Reference / AI / cognitionの最終qualification先となる

### 4.2 Validation

`semantic/sales/validation.tei`等のConstraint / Validationは、Application formやDB constraintの存在から推論せず、accepted business / semantic ruleとして保持する。

#### 判定

**Keep authored.**

Runtime / UI / DBへ投影できても、Projection側を正本にしない。

### 4.3 Application definition

代表:

```text
reference/workspace/applications/order-entry/application.tei
```

ここでは:

- 利用するSemantic namespace
- logical datasource
- target Database definition
- access mode
- required Database Contract range

をauthorする。

#### 判定

**Keep authored.**

`uses semantic`やDatasource requirementは、単なるPattern一覧の集計ではなくApplicationのscope / dependency declarationとして扱う。

将来Patternから機械的に利用namespaceを列挙できても、それだけでexplicit scopeを削除する根拠にはならない。

### 4.4 Application Pattern

代表:

```text
patterns/order-record.tei
patterns/order-lines.tei
patterns/customer-reference.tei
patterns/recent-orders.tei
patterns/submit-order.tei
```

PatternはSemantic Dataそのものを複製するのではなく、ApplicationがどのData / Relation / Query / Capabilityをどのinteraction roleで利用するかをauthorする。

`customer-reference.tei`では、Relation、Datasource、Database Contract Query、input bindingを一つのApplication interactionとして結ぶ。

#### 判定

**Keep authored where the application intent is not a deterministic default.**

ただし、将来標準CRUD / standard Reference Selector等を安全にderiveできるcaseでは、Pattern全文を毎回authorしないprogressive disclosureを比較する余地がある。

その場合も、明示Patternが必要になるoverride / cross-datasource / non-standard query caseを失わないことが条件である。

### 4.5 Database Contract

代表:

```text
reference/workspace/databases/sales-core/contracts/order.tei
reference/workspace/databases/sales-core/contracts/order-line.tei
```

`contract sales_core.order v4.2`は、Data全体を公開しているのではなく、read対象、write可能field、query surfaceを明示している。

#### 判定

**Keep authored.**

ContractはSemantic Dataから自動的に全公開すべき情報ではなく、Database ownership boundaryの公開意思である。

```text
Data has attribute
  != Contract writes attribute
```

この差はderivedで消してはいけない。

### 4.6 Release definitions — promoted temporal authority

代表:

```text
reference/workspace/releases/applications/order-entry/3.7.0.tei
reference/workspace/releases/applications/order-entry/3.8.0.tei
```

ReleaseはApplication definitionとDatabase requirementを部分的に重ねるが、current definitionの単なるcopyではない。

実際、current `application.tei`は`sales_core.order_line ^1`もrequireしている一方、`order_entry 3.8.0`のrelease snapshotは`sales_core.order ^4`を中心とした当時のrequirementを固定している。

#### 判定

**Keep as promoted snapshot.**

```text
current Application definition
        !=
historical / released dependency snapshot
```

Release作成時にcurrent definitionから候補生成することはできても、release後にcurrent stateから再導出して書き換えてはいけない。

この種の情報は「導出可能だった」ことと「現在もderived artifactである」ことを区別する。

## 5. C — Native external authoritative asset

### 5.1 Application query SQL

代表:

```text
reference/workspace/applications/order-entry/queries/recent_orders.sql
```

SQL本文はnative query authoring sourceである。

対応する`.tei` declarationは:

- logical Query identity
- Datasource
- native SQL file
- result → Semantic mapping

を持つ。

#### 判定

**Keep SQL native. Keep declaration / mapping authored for now.**

TEI利用のためだけにSQLを独自query DSLへ全面移植しない。

一方、SQL analyzerがcolumn lineageをexactに証明できるcaseでresult mappingの一部を安全にderiveできる可能性はある。

ただし次をpressureするまで自動省略しない。

- alias
- same-name ambiguity
- computed expression
- rename
- join / CTE
- provider差
- unresolved lineage

`column name matches attribute key`だけで暗黙mappingへしない。

### 5.2 Database Contract Query SQL realization

代表:

```text
reference/workspace/databases/sales-core/query-realizations/order-line-find-by-order.sql
```

これもnative SQL assetであり、対応`.tei` realizationがContract Query / input / output Semantic mappingを明示する。

#### 判定

**Keep native SQL + explicit realization boundary.**

Application QueryとDatabase-owned Contract Queryはauthority ownerが異なるため、一つのquery abstractionへ無条件に統合しない。

## 6. D — Realization / Environment-specific detail

### 6.1 Persistence Mapping

代表:

```text
reference/workspace/databases/sales-core/persistence/order.tei
reference/workspace/databases/sales-core/persistence/order-line.tei
```

`sales.order`から`public.orders`、各Semantic Attributeからphysical column、storage type、requiredness、primary key、indexへmappingする。

#### 判定

**Keep authored as realization choice.**

一見Semantic definitionの繰り返しに見えても、ここで表しているのは別の意味である。

```text
sales.order.total : std.money
        !=
public.orders.total_amount : decimal
```

physical naming、storage type、index、legacy schemaとの対応はCanonical business meaningから一意に決まらない。

#### 将来のshorthand候補

次の条件を満たすReferenceが複数出た場合のみ、convention-based defaultを比較できる。

- newly-owned schema
- Semantic nameとphysical nameが安定して対応
- provider storage mappingがunambiguous
- explicit overrideを保持
- generated desired schemaとactual driftを区別可能

これは現時点の実装taskではない。

### 6.2 Query Realization declaration

`contract-query-sql-realization`は、Database Contract Queryをprovider-specific SQLへ結び、input / resultをSemanticへmappingする。

#### 判定

**Keep authored.**

native assetの存在だけでは、どのContract operationを満たすかは確定しない。

### 6.3 Capability / Application realization

代表:

```text
applications/order-entry/realizations/authenticate.tei
applications/order-entry/realizations/authorize.tei
applications/order-entry/realizations/submit-order.tei
applications/order-entry/realizations/header-detail/order-lines.tei
```

これらはBusiness / Semantic capabilityやApplication Patternを具体的実現へ結ぶ。

#### 判定

**Keep authored where multiple realizations or explicit selection exist.**

built-in defaultで安全に一意なcaseは将来derive可能でも、custom / existing / external implementationへ差し替えられる境界を失わない。

### 6.4 Environment / Binding / Resource / Permission

代表:

```text
reference/workspace/environments/development/environment.tei
```

ここでは:

- Database instance provider
- Application datasource binding
- deployment executor
- observation / evidence provider
- Resource handle
- Permission grant
- lease policy

を明示する。

#### 判定

**Keep authored. Do not compress into Semantic meaning.**

これらは環境ごとに変化するauthority / trust / resource choiceであり、AI autonomyやApplication actorから推論してはいけない。

secret materialそのものではなくResource referenceを置く現在の境界を維持する。

## 7. B — Mechanically derived projection / golden

### 7.1 `reference/world.json`

`reference/workspace`をnormalizeして生成するgolden World snapshot。

#### 判定

**Derived golden. Never author as a second meaning source.**

意味変更時に更新するが、差分は必ずauthoritative sourceの変更から説明する。

### 7.2 `.tei/plugin-lock.json`

Workspaceから解決したPlugin / Bundle selectionのgolden。

#### 判定

**Derived golden.**

直接「望むPlugin選択」をここへ手書きするのではなく、upstream definitionsから再生成する。

### 7.3 Source Index / dependency index / cross-view matrix

現時点でSource Index自体はRepository snapshotとして保存していない。

#### 判定

**Derived / regeneratable.**

将来保存する場合も:

- input revision
- generation path
- staleness
- deletion / rebuildability

を明示する。

Data / Function / Business matrixも同様にProjectionとして扱い、人が第二の意味正本として更新しない。

### 7.4 Durable Run start golden

`reference/order_rollout.durable-run.m0.21.start.json`はWorkspace、observed fixture、Plugin Lockから正規経路で生成されるruntime-start goldenである。

#### 判定

**B/E boundary artifact — generated runtime-state golden.**

手編集でfingerprintだけを合わせない。

これはaccepted Semantic definitionではない。

## 8. E — Observed / runtime-only information

### 8.1 Deployment state fixture

代表:

```text
reference/observed/production.deployment-state.json
reference/observed/production.canary.deployment-state.json
```

これらはcurrent productionを自動取得したものではなく、特定のactual-state scenarioを固定したReference fixtureである。

#### 判定

**Observed-state fixture. Not Semantic authority.**

Canonical authoring変更だけを理由に更新しない。

DesiredとActualを比較した結果として矛盾を検出しても、ActualからSemantic / Releaseへ自動backfillしない。

### 8.2 Runtime Observation / Evidence

実行時のBusiness Result、Execution Error、Observation、deployment health等もこの側に属する。

#### 判定

**Evidence / actuality.**

feedback sourceにはなるが、定義変更はownership routing → Proposal → Acceptanceを通す。

## 9. F — Proposal / Evidence / non-canonical Reference

### 9.1 Business Procedure Reference fixture

代表:

```text
reference/business-system-design/order-entry-business-procedure.json
reference/business-system-design/order-entry-follow-up-procedure.json
reference/business-system-design/order-entry-procedure-reference-set.json
reference/business-system-design/order-entry-protected-target-overlay.json
```

現行fixture自身が:

```text
artifactRole: reference-authoring-fixture
canonical: false
```

を明示している。

trigger / actorにもcandidate statusがあり、manual / external / observation boundaryを圧力試験している。

#### 判定

**F: Reference / Pressure candidate. Do not count as accepted Canonical authoring yet.**

Business Procedure Contract ADRがReadyになる前に、このfixtureのfile形状をdurable production schemaとして扱わない。

ただし、Data / Functionだけでは保持できないBusiness responsibility / manual work / handoff等が実在すると確認された場合、その意味を「定義を短くするため」に削除してはいけない。

### 9.2 AI Proposal / cognitive candidate / review trajectory

将来のProposal、Reference pattern retrieval、sparse activation、LLM output、accept/reject trajectory等もFに属する。

#### 判定

**Non-authoritative until accepted.**

Accepted後に残る意味はA / C / D等の正しいowner sourceへ移り、Fだけを唯一のdurable recordにしない。

## 10. Apparent duplication audit

### 10.1 Semantic Attribute vs Persistence Column

```text
Semantic Attribute
  -> Persistence mapping
  -> physical column
```

**Not duplicate authority.**

mapping edge自体がrealization choice。

### 10.2 Query SQL vs Query declaration result mapping

```text
native SQL result
  -> explicit Semantic result mapping
```

**Not duplicate authority today.**

SQLはC、mappingはD/Application boundary。Analyzerがlosslessに証明できる部分のみ将来derived候補。

### 10.3 Application Database requirement vs Application Release requirement

```text
current Application intent
        !=
released temporal snapshot
```

**Not duplicate authority.**

current definitionが変わってもpast releaseを再deriveしない。

### 10.4 Data vs Database Contract

```text
Data exists
  != published read/write/query surface
```

**Not duplicate authority.**

Contractはownership boundary。

### 10.5 Application Pattern vs Business Procedure

```text
Application interaction
  != Business responsibility / manual work / acceptance
```

**Not duplicate authority.**

現時点でBusiness ProcedureはFだが、将来acceptedになってもPatternから自動再構成できないBusiness meaningが残る可能性が高い。

### 10.6 Workspace vs `world.json`

**True derived duplication.**

Worldはgolden / executable-spec目的で保存するが、人はupstream sourceを直す。

### 10.7 Workspace / Plugin definitions vs plugin lock

**True derived duplication.**

Lockはresolved resultの再現性のため保持する。

## 11. Reduction candidates

このinventoryから直ちに削除すべきauthoringは見つからなかった。

一方、将来Referenceで比較する価値があるのは次。

### R1 — Standard Pattern progressive default

simple record / relation / reference selector等で、explicit Patternなしでも安全なderived Projectionを作れるcaseを比較する。

条件:

- explicit subsystem / datasource boundaryを越えない
- mutation authorityを勝手に増やさない
- overrideするときだけauthored Patternへ昇格できる

### R2 — Proven SQL result mapping

Analyzerがexact lineageを証明できるcolumnだけ、mapping候補を自動提示する。

標準は**proposal / Preview**であり、ambiguous / unresolvedを黙って採用しない。

### R3 — Convention-based Persistence proposal

newly-owned schemaで一意なconventionが成立するcaseのみ、Persistence Mapping候補を生成する。

legacy DB / rename / physical stability / provider-specific typeがある場合はexplicit mappingを維持する。

### R4 — Release snapshot creation assistance

current Application / Database definitionからRelease候補を生成することはできる。

ただしAccept後のReleaseはpromoted temporal authorityであり、derived cacheではない。

## 12. Authoring Necessity Test

今後新field / new source / Editor inputを追加する前に、次を問う。

```text
1. これは独立したBusiness / Semantic meaningか
2. これは公開Contract / responsibility / authority choiceか
3. これはconcern境界を越える明示mappingか
4. これはEnvironment / Provider / Targetの選択か
5. これはrelease等のtime-frozen Decisionか
```

どれかがyesなら、human-authored definitionである可能性が高い。

すべてnoなら次を問う。

```text
6. accepted sourceからdeterministicに再生成できるか
7. 再生成時にEvidence / unresolved / ambiguityを失わないか
8. 削除してもaccepted meaningを完全に再構築できるか
```

6〜8がすべてyesなら、derivedへ送る候補である。

さらに:

```text
Actual / Observationか
Proposal / candidateか
```

ならAへ混ぜず、それぞれE / Fへ置く。

## 13. Definition-centeredな最小核

現行Referenceで「短く保つべき核」は一つの巨大fileではない。

concern別に小さいauthoritative definitionがあり、明示的な関係で接続されている状態を指す。

```text
Semantic facts / rules
      +
Application intent
      +
Database Contract
      +
Native asset where native authority matters
      +
Explicit realization / environment choice
      +
Promoted release snapshot
```

そこから:

```text
World
Source Index
Dependency graph
Cross-view matrix
Plugin Lock
Generated target
Impact preview
Cognitive index
```

をderiveする。

「一つのfileに集約すること」や「全部をAIに補完させること」はsemantic densityではない。

## 14. X-TEA / XEADから見た示唆

Reference Modelに学ぶべき点は、definition fileの具体形式ではなく、少数の意味ある定義とcross-referenceから多くを検査できることにある。

TEIではその考え方を次へ翻訳する。

```text
small concern-owned definitions
        ↓
exact cross-reference
        ↓
multiple projections / executable reference
        ↓
contradiction discovery
        ↓
owner source correction
```

Function / CRUD matrix、Business flow、Data relation graph等を人が別々に同期する方向へ戻さない。

一方、Business responsibilityやmanual workまでData / Functionから推論して消すこともしない。

## 15. P0-1結論

現行Referenceは、Definition-centered architectureへ移行するために全面的なDSL再設計を必要としていない。

主な結論:

1. **Semantic / Application / Contract / Realization / Environmentのconcern分離は概ね健全。**
2. **目につく繰り返しの多くは重複正本ではなく、boundary mappingまたはtemporal snapshot。**
3. **`world.json`、Plugin Lock、Source Index、matrix等は明確にderivedとして扱える。**
4. **native SQLはnative authorityとして残す。**
5. **Observed stateはDefinitionへ自動backfillしない。**
6. **Business Procedure fixtureはまだCanonical authoring量へ数えない。**
7. **AI時代の削減余地は、definitionを削るより、safe default・proposal・Previewでboilerplate入力を減らす側にある。**
8. **Release snapshotのように、生成開始点がcurrent definitionでも、Accept後は独立authorityになるartifactがある。**

したがって次のP0作業では、新しいschemaを設計するより:

- feedback routing catalog
- three-memory separation audit
- Reference Pattern provenance handoff

を進める方が価値が高い。

## 16. Reopen conditions

次のEvidenceが出た場合だけauthoring削減をreopenする。

- 複数realistic Referenceで同じboilerplateが常に一意にderiveできる
- explicit mappingを削ってもrename / legacy / provider差 / ambiguityをlosslessに扱える
- Release等のtemporal authorityを損なわずcandidate生成とpromotionを分けられる
- generated defaultからexplicit overrideへ安全に昇格できる
- AIなしでも同じaccepted sourceを再構築できる

単に「AIなら推測できる」はreopen理由にしない。
