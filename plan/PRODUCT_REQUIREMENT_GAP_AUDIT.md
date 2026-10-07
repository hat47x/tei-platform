# Product Requirement Gap Audit

- 監査日: 2026-09-21（再監査。前回は2026-09-11）
- 追記: 2026-10-03（RQ-031〜RQ-033の初期評価）
- 対象: RQ-001〜RQ-033
- 上流Trace: `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- Quality設計: `../docs/product-quality-architecture.md`
- Quality実証計画: `PRODUCT_QUALITY_EVIDENCE_PLAN_2026-09.md`

## 1. 目的

この文書は、`requirements/TEI_PRODUCT_REQUIREMENTS.md`の各Requirementについて、現在のRepositoryにどのEvidenceがあり、どの不確実性が残っているかを逆マッピングするcurrent auditです。

実装量、Issue close数、文書数ではなく、次を見ます。

- semantic boundaryが成立しているか
- authoritative sourceへ戻れるか
- Reference / Conformanceで再現できるか
- actual execution Evidenceが必要な場合に取得済みか
- synthetic / generated / real-provider / actual-external / Target Evidenceを混同していないか
- Provider / Target / Host / build条件を説明できるか
- failure / degraded / unknownを成功へ読み替えていないか
- unresolvedを成功へ丸めていないか

Product Qualityは新しいRequirement群として追加せず、既存RQを横断するEvidence観点として監査します。

## 2. 評価区分

| 区分 | 意味 |
| --- | --- |
| **Strong Evidence** | 実装・Reference・Conformance / Evidenceが一続きで存在し、主要意味境界を説明できる |
| **Partial Evidence** | 重要部分は成立しているが、Requirement全体またはactual / multi-context Evidenceが不足 |
| **Design Only** | Requirement / Architecture / Planは具体化されているが、対応するReference実証はまだない |
| **Not Started** | 対応する設計・Referenceがほぼない |

`Design Only`を実装済み、`Partial Evidence`をProduct claim成立済みとは読み替えません。

## 3. 現在のProduct主線

Gate BとGate Cは2026-09-19にReadyへ到達しました（Gate Cの判定記録: [`GATE_C_READINESS_AUDIT.md`](GATE_C_READINESS_AUDIT.md)。Visual Fidelityは単一のAI評価者による評価）。最大の直列blockerは、Gate C後の並行lane（Studio / knowledge-first・AI / 第二Application・第二Target）の着手判断です。

2026-09-21に、前回が別作業へ送っていたRQ別Evidence tierの再監査を行いました。同時に、2026-09-18のProduct再定義で追加されながらこの監査にも`UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`のPAS mappingにも載っていなかったRQ-027〜RQ-030を対象へ入れました。以後の欠落は`tools/repository-plan-check`の`TestEveryRequirementIsMappedToAnAcceptanceScenario`がfail-closedで検出します。

2026-10-03には、意味体系を後付けで拡張・合成できることをProduct Valueへ明示したことに伴い、RQ-031〜RQ-033を追加しました。この3件は追加時点ではDesign Onlyです。Requirement追加と同じ変更でPAS mappingと本監査へ登録し、前回のように価値定義だけが検証マップより先へ進む状態を作らないようにします。

```text
Gate A — Ready
   ↓
Gate B — Ready（R = e5231839a5dcc62cce1181a2731fcb7c70d16736）
   ↓
Handover Baseline B / revision R — 固定済み
   ↓
First Web Target build T1 — Evidence取得済み（T = 0ba87a4e0a96…）
   ↓
Gate C — PAS-04 — Ready
   ↓
   ├─ Studio minimal loop — PAS-01
   ├─ knowledge-first / derived cognition — PAS-06
   ├─ AI Proposal / Acceptance — PAS-01 / PAS-06
   └─ Second Target — PAS-04 / RQ-024
```

Cross-cutting Product Qualityは新しいGateを挿入しません。

actual Oracle #214 / #371 / #376は重要なPAS-03 / external asset Evidenceとして継続しますが、Gate Bの第四Required laneではありません。

## 4. Requirement別監査

| RQ | 評価 | 主PAS / scope | 現在成立していること | 主な残Gap |
| --- | --- | --- | --- | --- |
| RQ-001 | **Strong Evidence** | PAS-01 | Canonical ID / source path分離、Semantic Data / Relation / Capability、Provider / Target固有実装の隔離 | First / Second Targetを通したmeaning安定性、複数Application再利用 |
| RQ-002 | **Strong Evidence** | PAS-01 | Source Index、Source Diagnostic、native source trace、file move / source trace基盤、Shared Acceptance（acceptがauthoritative sourceを書き再読込して検算する、2026-09-21） | Studio UIからのE2E。accepted Decision / Evidenceは機械可読なindexを持たない（RQ-027と同じGap） |
| RQ-003 | **Strong Evidence** | PAS-01 | native source / derived IR、Evidence / Handover非Canonical、AI / Studio非authority | Studio / AI authoring E2E |
| RQ-004 | **Partial Evidence** | PAS-03 | native SQL、CustomGoBinding、Plugin、legacy auth、external-observed Contract Query | existing batch / API / config、actual external migration Evidence |
| RQ-005 | **Partial Evidence** | PAS-02 | Application DatabaseRequirement、logical Datasource、multi-DB Driver access | multi-DB transaction / failure、actual external E2E |
| RQ-006 | **Strong Evidence** | PAS-03 | Desired / Actual / Diff / Unresolved、PostgreSQL observation、ProviderExpression | Database外のchange / actual-state領域での実証 |
| RQ-007 | **Partial Evidence** | PAS-03 | Provider SPI、driver assembly分離、provider-specific normalization / Runner | actual second Database Provider圧力試験 |
| RQ-008 | **Partial Evidence** | PAS-02 | Semantic Capability、Binding、Environment、Plugin、legacy auth compatibility | builtin / generated / plugin / existingの同一Capability比較 |
| RQ-009 | **Partial Evidence** | PAS-02 | legacy Authentication / Authorization Reference、Gate A、Driver subject継承 | actual migration target、段階置換Evidence |
| RQ-010 | **Design Only** | PAS-01 | Studio非Canonical、high-density方針、`ISSUE-000046`、機械側projectionとしてのShared Acceptance Preview（意味差分 / impact / unresolved、2026-09-21） | 人向けsurfaceが無い（Studio UI未着手）。operability / accessibility。人が実際に理解・判断したEvidence |
| RQ-011 | **Partial Evidence** | PAS-04 | Source / Diagnostic / Workbench / Driver / Handover等の複数Projection、First Web Target `T1`（Gate C Ready、2026-09-19） | Studio authoring Projection、Second Target |
| RQ-012 | **Strong Evidence** | PAS-02 | Driver主要Scenario、Validation / Auth / Command / Observation、Gate B actual #310/#255/#308 same-R Evidence（2026-09-19成立）、First Web Target `T1`でのDriver meaning再現（Gate C Ready、2026-09-19） | production Runtime / 第二Targetでの再確認 |
| RQ-013 | **Partial Evidence** | PAS-04 | Gate C contract、baseline `R` / Target build `T`分離、Target Capability / Execution-Trust仮説、First Web Target `T1`のactual Evidence（Gate C Ready、`T = 0ba87a4`） | Second Targetとの比較が無く、Target-neutralityは1 Targetだけでは示せない |
| RQ-014 | **Partial Evidence** | PAS-02 | browse / record / reference / header-detail / commandをApplication intentとして実証 | 複数Application / Targetでの再利用Evidence |
| RQ-015 | **Design Only** | PAS-06 | knowledge-first原則、read-only exploration計画、derived cognition研究境界 | Gate C後のexploration actual Reference |
| RQ-016 | **Design Only** | PAS-06 | Proposal / Decision分離、suppression方針、autonomy / authority分離 | Proposal / Decision / suppressionの実UX / Evidence |
| RQ-017 | **Partial Evidence** | PAS-01 | deterministic generated artifact、native source authority、Shared Acceptance設計 | Studio / AI / Target間で二重管理を作らないE2E |
| RQ-018 | **Strong Evidence** | PAS-02 | Business Result / Execution Error / separate Observation read | Target Projectionで同境界を維持するEvidence |
| RQ-019 | **Strong Evidence** | PAS-05 | Permission Grant、Resource Broker / Lease、Sandbox Profile、Host material分離 | production-grade enforcementの拡張 |
| RQ-020 | **Strong Evidence** | PAS-05 | ownership / heartbeat / fencing / durable handle / no-auto-replay / unknown保持 | 複数production Runtimeでの圧力試験 |
| RQ-021 | **Strong Evidence** | cross | Reference Workspace、Conformance、golden、source-fixed runners、Provider Evidence tools、Gate B actual run（2026-09-19）、Gate C Target Evidence（`T = 0ba87a4`）、Lane C Gate A actual run（2026-09-20 / 2026-09-21） | Studio / AI / 第二Targetを含めた再現性 |
| RQ-022 | **Strong Evidence** | cross | architecture probe / actual Evidence / production claimを分離 | v1以後のsupport / compatibility level整理 |
| RQ-023 | **Partial Evidence** | cross | monorepo、Go中心、ADR / Plan / Reference、Provider / Target分離、doc guard（`repository-plan-check`）とhealth check | 保守量のGapが実測として現れた（§9）。plan文書108K行 / production Go 44.8K行。`reference-business-procedure-check`は171 file・50.8K行で、うち101 file・33.6K行がTEI packageをimportしない |
| RQ-024 | **Design Only** | PAS-04 | First / Second Targetで測る方針、Gate C測定点、R/T traceに加え、Plugin / Semantic Extensionのlogical境界とphysical artifact境界を分離し、cold / warm start、初期転送量、storage、memory、resolver / evaluator overhead、Extension追加時の増分costをTargetごとに観測する設計を追加。ISSUE-000080 E3ではgeneric Runtime Capabilityを既存Plugin Lockへ接続し、無関係なPlugin変更で過剰にstaleにしないbindingをReference化した。E4では同じlogical Provider identityを保ったTarget別artifact realizationを解決し、Target未指定・非対応Targetへのfallbackを拒否するcandidateを追加した | Plugin化前後のactual比較、unused implementation pruning、shared runtime deduplication、Web Targetでのprocess-free actual realization、context-specific thresholdが未実証。現行Runtime Plugin LockはPluginごとにartifact digestを一つしか持たず、同一Targetで1 Pluginを複数Capability artifactへ分割するAuthority表現も未成立。Wasm / JavaScript artifactのactual size / startup / memory Evidenceも無い |
| RQ-025 | **Partial Evidence** | PAS-07 | Business Action != Application Function、manual / external Action、actor / authority分離、Business Procedure Reference / Pressure、Lane C Gate A actual execution（2026-09-20）、独立second corpus（2026-09-21、作成者self-review） | 独立review未実施。Canonical identity最終判断。Contract ADRはNot Ready |
| RQ-026 | **Partial Evidence** | PAS-07 | Data / Function / Business cross-view Reference、Handoff identity Pressure、derived Projection原則、realistic second-case actual Evidence（`R = 82e4301`、Gate A A1〜A8） | 独立review未実施。Data-state / ownership cross-check（RRB-P2項目4）はData identityが無く未確認。diagnostic severityの確定 |
| RQ-027 | **Partial Evidence** | PAS-06 | `tei world` / `source-index`（artifact fingerprint + reference） / `validate` / `index` / `consumer-impact` / `workbench-model`、`changeanalysis`のdependency / impact、ACTOR_CONTEXT R0 source map・R1 builder challenge（production codeを検証）。**accepted Decisionの機械到達**（2026-09-21）: Shared Acceptance Boundaryを通った受理が`.tei/decisions/`へ残り、`decisions`が現在のworkspaceと照合して`current` / `superseded`と`intact` / `diverged`を返す。記録はauthorityではない（消去・偽造しても意味とrevisionは動かない） | 残る到達不能: **`Variation`は表現を持たない**（`internal/`に0件）。Gate Evidence / Release locatorは索引の無い散文からの参照のまま。決定記録はこの境界を通った変更に限られ、Coreには同等の仕組みが無い。再入場の十分性を判定するconformanceが無い。担当Issueが無い（ADR-0028のみ） |
| RQ-028 | **Partial Evidence** | PAS-01 | Shared Acceptance Previewが、意味差分・application差分・理由つきimpact辺・unresolved・changed collections・source edit fingerprintを、Canonical ID / source pathへ戻れる形で出す（2026-09-21）。ACTOR_CONTEXT R2 human review challenge、Source Diagnostic、Workbench model | 人向けsurfaceが無い（Studio UIはDesign Only）。人が実際に承認したEvidenceが無い。承認に十分なprojectionの定義が無い。native custom code / 権限変更のprojectionが未実証。担当IssueもADRも無い |
| RQ-029 | **Partial Evidence** | PAS-06 | 第二Applicationが共有`customer` semanticをorigin / provenanceを分けて再利用するS1〜S5 executable pressure、`run-knowledge-portability-reference-evidence.sh`、`KNOWLEDGE_PORTABILITY_R0_BOUNDARY`、Shared Acceptanceの型付きcomposition | `Variation` / destination-specific overrideの表現が無い（`internal/`に0件）。実際のAcceptanceは独立レビュー待ちで未実施。別Repository / 別組織への移植Evidenceが無い。担当IssueもADRも無い |
| RQ-030 | **Design Only** | PAS-07 | Business Procedure Referenceがmanual / external / observation-boundaryとHandoffを区別してauthorでき、baseline checkerとGate A A8で機械確認できる（agentが必要とする区別の素材）。Permission Grant / Resource Lease / authorization境界。ACTOR_CONTEXT R3 / R6系の設計圧力 | **agentがTEI成果物を読んで業務を遂行するpathが製品側に無い**。R3系圧力の大半がproduction codeへ接続していない（§9）。閉包条件が未定義 |
| RQ-031 | **Design Only** | PAS-06 | Product Value / Principle / Requirementで、Data Modelを含む特定のモデル体系をCoreの唯一の上位モデルにしない境界を定義。`tools/reference-semantic-extension-catalog` candidateでは、Core未定義の`service-level-policy` / `temporal-window`をopenなsubject kindとして既存Source Index subjectと同じcatalogへ追加。PR #927ではnamespace authorityをdescriptorから分離し、`std` / `std.*`をAccepted Authority Evidence無しでは拒否する。authorityがnamespaceとowner Extensionを明示した場合だけsemantic contributionを許可し、first claimantをnamespace authorityにしないReference境界も追加 | exact-headのfocused test / repository health未実行。namespace Authority Evidenceの発行Authority、subnamespace delegate、publisher trustのcontractは未実装。Reference専用descriptorをCore primitiveへ昇格すべきかも未判断 |
| RQ-032 | **Design Only** | PAS-06 | `reference-semantic-extension-catalog`はExtension identity / namespace / semantic version / package dependency / Semantic Interface / Constraint / relation / connection、complete-search、Semantic Lock、Validation PlanをReference化した。SUIとDatabase Migrationでは同じacyclic Interface / evaluatorを再利用する第二consumer pressureまでmainへ統合済み。ISSUE-000080のReferenceでは、Core固定enumを追加せずTool / Runtimeのcallable Capability Contractを解決し、Authority scope、Provider Conformance、明示wiring、exact Resolution Lockを分離した。既存`contract-query-executor`は旧`plugin.Definition`をAuthorityに保つ一方向projectionでgeneric contractへ写し、Application Runtime scopeでは既存Runtime Plugin Lockの対象entryへ派生Bindingで接続する。E4ではContract identityをTarget差から分離し、同じlogical ProviderをTarget別artifactへrealizeするcandidateも追加した | productionの`internal/plugin.CapabilityKind`は固定enumのままで、generic Capability ContractはReference専用。現行builtin Pluginはexact implementation artifact digestを持たず、そのままgeneric Conformance済みProviderへ昇格できない。1 Pluginを複数Capability artifactへ分割するRuntime Authorityも現行Plugin Lockでは表現できない。Contract / Conformance Evidenceの発行Authority・publisher trust、実Source Adapter consumer、Permission / Sandbox / signature境界、browser-compatible artifactのactual build / load / cost Evidenceは未確定。Semantic ExtensionのReference descriptorもproduction syntaxへ固定していない |
| RQ-033 | **Design Only** | PAS-06 | Subject Semantic Contextは対象subjectからExtension Source / version / dependency path / Constraint / Interface / evaluator / Connection / Unresolvedへread-onlyで再入場できる。Semantic Evaluation Evidenceはexact Semantic Set / Constraint / evaluator artifact / Conformance / input / statusへ結び付き、Shared Acceptanceではtyped artifactとして添付・fingerprint検証し、Review A3 / A7とDecision provenanceへ運べる。ISSUE-000080ではTool / Runtime Capabilityのfresh resolutionとexact Lock replayも分離し、Catalog成長だけで採用済みproviderを自動更新しないcandidateを追加した | Studio / AIの実consumerからSubject Context・Capability Contract探索・Lock replayまでを一続きに使うE2Eがない。`kind-applicable`候補とAccepted semanticsの区別、resolver budget / authority不足 / evaluator failureの統合提示、Capability ContractのSource / Authority discovery、current evaluator再実行を含むfreshness確認はproduction pathで未実証 |



2026-10-07のISSUE-000080 migration pressureでは、既存`CapabilityKind`をgeneric Capability Contractへ移す際の二重正本を追加監査した。旧`plugin.Definition`をAuthorityに残す一方向projectionへ、正規化したlegacy Capability宣言の`sourceDeclarationFingerprint`を追加し、Resolution identityとConformance Evidenceを投影元宣言へ結び付けた。artifact未確定のlegacy ProviderはCatalogへ残してもcallable候補にせず、別scopeの解決を阻害しない。一方、同じlogical providerを旧projectionと新宣言から重複登録する場合はartifact確定前でもfail closedとする。productionの`CapabilityKind`はまだ変更しておらず、exact-head Repository Gateも未取得のためEvidence tierは上げない。

同日のE3 Runtime Plugin Lock pressureでは、generic Capability Resolution Lockを新しいRuntime Authorityにせず、既存Plugin Lock内の対象Provider entryへ`RuntimeCapabilityBinding`で接続した。Environment Lock全体をfingerprintへ含めないため、無関係なPlugin追加だけではbindingをstaleにしない。一方、対象Pluginの削除、definition fingerprint、artifact digestの変化は拒否する。legacy migration pathではgeneric Providerがauthoritative `plugin.Definition`から一方向に導出されたことも再確認する。現行Plugin LockはPluginごとにartifact digestを一つしか持たないため、同一Targetで1 Pluginを複数Capability artifactへ分割する構成はfail closedのままとした。これはRQ-024 / RQ-032の未解決Gapであり、Evidence tierは変更しない。

同日のE4 Target realization pressureでは、Capability ContractをTarget別の意味へ分岐させず、同じPlugin / version / implementation IDに対してDesktop / Web等のTarget別artifactを持てるReference境界を追加した。ProviderとConformance EvidenceはTargetを明示し、Targetを指定した解決ではexact Targetだけを候補にする。Target未指定Providerを任意Target対応とみなさず、Target contextが無い場合もTarget固有Providerを推測しない。Resolution LockとRuntime Capability BindingもTargetを保持する。actual browser artifactと性能Evidenceは未取得のためDesign Onlyを維持する。
2026-10-06の追記: Semantic Evaluation Evidenceは`tools/internal/semanticevidence`のtyped artifactとしてShared Acceptanceへ接続され、受理時に見たexact artifactをSource Evidenceと分離してReview / Decisionへ残せるようになった。Evaluationの`pass / fail / unknown`だけからAccept / Rejectを自動導出せず、current evaluator / Conformanceを再実行したとも主張しない。

同日のISSUE-000080 candidateでは、Tool / Runtime CapabilityをCore固定`CapabilityKind`から分離するReferenceを進めた。Capability Resolution Lockはresolution policy、Authority scope、Requirement、明示wiring、exact Contract、Provider identity、artifact digestを固定し、fresh resolutionとreplayを分ける。Conformance Evidenceはexact artifactへ結び付け、同じlogical provider identityに複数artifactを重ねる状態を拒否する。既存`contract-query-executor`は旧Plugin定義をAuthorityにした一方向projectionでpressureする。現行Oracle / PostgreSQL builtin Pluginはartifact digestを持たないため、宣言の投影はできてもgeneric Conformance済みProviderとしては採用しない。実装artifact identityを信頼できるbuild / provenance経路で確定するmigration prerequisiteが残る。いずれもproduction contractへの昇格前であり、本branchのexecutable Gateも未取得なのでRQ-032 / RQ-033はDesign Onlyを維持する。


2026-10-07のISSUE-000080 Runtime Authority pressureでは、generic Capability Contractを既存Runtimeの第二selectorにしない境界を追加した。既存`contract-query-executor`はEnvironment / Database instance / Contract Query targetごとのBindingが先にexecutor Capabilityを選び、generic Contractは選択済みProviderだけをConformance検証する。Capability Resolution Lockはgeneric `Wiring`を持たず、target-specificな選択は`RuntimeCapabilitySelectionBinding`でEnvironment / instance / target / executor / Plugin / Capability Lockへ結び付ける。同一EnvironmentでOracleとPostgreSQLのtargetが同じgeneric Contractを別Providerで満たすpressureを追加したが、production pathへの統合とexact-head Repository Gateは未取得のためRQ-032 / RQ-033のtierは変更しない。

## 5. 評価集約

### Strong Evidence — 10

- RQ-001
- RQ-002
- RQ-003
- RQ-006
- RQ-012
- RQ-018
- RQ-019
- RQ-020
- RQ-021
- RQ-022

### Partial Evidence — 15

- RQ-004
- RQ-005
- RQ-007
- RQ-008
- RQ-009
- RQ-011
- RQ-013
- RQ-014
- RQ-017
- RQ-023
- RQ-025
- RQ-026
- RQ-027
- RQ-028
- RQ-029

### Design Only — 8

- RQ-010
- RQ-015
- RQ-016
- RQ-024
- RQ-030
- RQ-031
- RQ-032
- RQ-033

### Not Started — 0

2026-09-21の変更: RQ-012はFirst Web TargetでのDriver meaning再現が成立したためStrongへ、RQ-013は`T1` actual Evidenceが得られたためPartialへ移しました。RQ-027〜029をPartial、RQ-030をDesign Onlyとして新たに評価しました。tierを上げたのは、その行が挙げていたblocking Gapが記録されたEvidenceで閉じた場合だけです。

2026-10-03の追記: RQ-031〜RQ-033は追加時点でDesign Onlyとしました。既存のPlugin / Capability / cross-view / knowledge-firstの仕組みは関連する土台ですが、「意味体系そのものをCore変更なしで追加できる」「AI / Toolがその拡張を一般的に探索・合成できる」という新しいRequirementの成立Evidenceには数えません。

2026-10-04の追記: RQ-031〜RQ-033向けに`tools/reference-semantic-extension-catalog`のcandidate実装を追加した。Core未定義の意味kindをReference descriptorから既存Source Indexと同じcatalogへ載せ、compatible-major依存、subject conflict、namespace ownership conflictを決定論的に扱う。Core subject集合と正規化Extension意味をresolution identityへ結び、Source Index / descriptor provenanceはEvidence identityへ分離した。ただしfocused testとrepository healthを未実行のため、3件ともDesign Onlyを維持する。

2026-10-04の敵対的評価では、現行candidateとPre-ADRへ12個の破壊ケースを適用した。特に、highest-compatibleを先に固定する現行resolverは、有効なlower compatible versionが存在してもtransitive conflictを返せる。加えて、Semantic Interfaceの自己申告、`std.*` namespaceのfirst claimant、root / policy / canonicalizationを欠くlock、EnvironmentごとのEvaluator選択、optional diagnosticからAcceptanceへの逆流を設計Gapとして確認した。`SEMANTIC_EXTENSION_ADVERSARIAL_REVIEW_2026-10.md`とISSUE-000077 E0-B / E0-C / E1へ反映したが、いずれも設計または未実証の段階であるためEvidence tierは上げない。

2026-10-04の追記では、標準機能を広くPlugin / Semantic Extension化した場合の性能・storage・Target可搬性を追加監査した。logical Extension数をprocess数やWasm module数へ一対一対応させず、Target buildで実行artifactをprune / coalesce / deduplicate / lazy loadできる構造をRQ-024 / RQ-032、Pre-ADR、ISSUE-000077へ反映した。Web Targetではnative processを必須にせず、Target packaging差だけではSemantic Set fingerprintを変えない方針とした。actual size / startup / memory / browser Evidenceは未取得のためEvidence tierは変更しない。

2026-10-05のPR #927 candidateでは、AX-01のfalse conflictをReference codeへ反映した。compatible major内で高いversionを優先しつつ、推移依存またはversion固有cycleが全体制約と両立しない場合はlower compatible versionへbacktrackする。加えて、複数の完全解がある場合の選択規則を`resolutionPolicy`として明示し、そのidentityをSemantic resolution fingerprintへ含める。ただしrepository exact-head test / health gateを通していないため、Source Evidence候補以上には扱わず、RQ-032はDesign Onlyを維持する。

2026-10-05のE3 candidateでは、第二consumerとして既存Database Migrationの`Definition.Parents`を採用した。SUIとMigrationからacyclic判定に必要なnodes / directed edgesだけをdomain-neutral snapshotへ投影し、同じSemantic Interface contract、provider Extension、evaluator implementation / artifactを再利用する。SUI固有語彙が共通snapshotへ漏れないnegative controlも追加した。これはSource Evidence候補であり、exact-head test / health gate未取得のためRQ-032のtierは上げない。 現行CoreはMigration parentの存在を検査する一方でcycleを検査していないため、既存ruleをExtensionへ移すのではなく、未表現のacyclicだけをsecond-consumer pressureとして追加する。

同candidateでは、backtrackingの探索量をcandidate attempt数で制限できるoperational budget境界も追加した。budget超過はdependency conflictと別errorとして返し、十分なbudgetではunbounded解決と同じ結果になることをtestする。固定budget値をSemantic fingerprintへ含めず、意味仕様と運用上のresource policyを分離する。ただし実Repositoryでの性能測定とbudget値の妥当性検証は未実施である。

Semantic Lock candidateではsemantic replayとcurrent Evidence bindingも分離した。lockのexact graphはEvidence fingerprintを意味identityへ含めず再現し、必要な場合だけ`-verify-semantic-lock-evidence`でInterface contract / conformance profileとnamespace / ownerに対応するAccepted Evidenceの存在を確認する。Evidence発行Authorityやpublisher trustは未実証であり、exact-head test / health gateも未実施のためEvidence tierは変更しない。

2026-10-05のEvaluator scope追記では、A5のEnvironment依存をReference codeへ反映した。SUI Stage 0のhierarchy shadow validatorを共有packageへ切り出し、Semantic Extension側でも同じ実装を利用する。ConstraintはSemantic Interface evaluator requirementを明示し、Interface / provider / exact contract / conformance profileが解決できない場合はcatalogへ有効Constraintとして出さない。Validation implementationはEnvironmentのCapability negotiationではなく、exact Semantic Set fingerprintへboundした`SemanticValidationPlan`で全件明示し、implementation identity / artifact digestを固定する。M1 / M2 / M3 / M4をPlan経由でpressureし、Conformance欠落、別implementation差し替え、cancel / errorをfail openにしないcandidateを追加した。Plan / Evaluation EvidenceはCLI・JSONで再入場可能にしたが、exact-head test / health gateとRuntime Plugin実行境界は未実証なのでRQ-032 / 033はDesign Onlyを維持する。

2026-10-05のAI / Tool re-entry candidateでは、既存の`knowledgecontext.Build`を変更せず、対象subjectへresolved Semantic Catalogを重ねる`Subject Semantic Context`を追加した。ContextはSemantic Set / Evidence fingerprint、Extension Source / version / root dependency path、Constraint / Interface / evaluator provider、Connection、subject固有UnresolvedとSemantic Set全体のUnresolvedを保持する。Evaluator bindingを入力した場合だけValidation Planの対象bindingを含め、未指定時はimplementationを推測しない。Source Evidence候補であり、exact-head test / health gate未取得のためRQ-033はDesign Onlyを維持する。
## 6. Cross-cutting Product Quality監査

Quality concernを新しいRequirementへ分割せず、既存RequirementのEvidence completenessとして監査します。

| Quality concern | 現在の強み | 主な残Gap |
| --- | --- | --- |
| Meaning stability | Source / Canonical / Projection / Evidence非同一性が強い | multi-Target / Studio / AI E2E |
| Change safety | Desired/Actual/Diff、stale rejection、Provider preconditionがある | Database外を含むchange impact / recovery Evidence |
| Execution / diagnosability | Result / Error / Observation、unknown / no-auto-replayがある | Target / production横断diagnostic correlation |
| Interaction / accessibility | Target-neutral / high-density / Semantic-Functional-Visual分離がある | First Target actual operability / accessibility |
| Performance / resource | RQ-024とR/T traceがある | First / Second Target actual baseline / threshold rationale |
| Security / authority | Permission / Resource / secret境界が強い | effect-time current authority / production enforcement |
| Maintainability | monorepo、derived artifact非authority、deterministic generation | Studio / AI / quality docs追加後の管理負担 |
| Business completeness | RQ-025 / 026、manual / external Action、cross-view Pressure | actual execution / second corpus |

詳細は`docs/product-quality-architecture.md`を参照します。

## 7. Product Gapの再統合

### Gap A — Gate B actual Evidence（2026-09-19解消）

主にRQ-005 / 009 / 012 / 014 / 021。

```text
actual full checkout preflight
        ↓
#310 / #255 / #308
        ↓
same clean candidate R  ← e5231839a5dcc62cce1181a2731fcb7c70d16736
        ↓
#387 final convergence  ← 成立
        ↓
#273 Gate B audit  ← Ready（artifactAuditPassed = true）
```

Gate BはReadyへ到達し、Handover Baseline `B / R`も固定済みです（[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md) / [`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](GATE_B_TO_C_HANDOVER_BASELINE_B.md)）。RQ-005 / 009 / 014はGate B以外のPartial Evidence理由（multi-DB / actual migration / 複数Application再利用等）が残るため、引き続きPartial Evidenceのままです。

新しいquality laneを追加しません。

### Gap B — First Target

主にRQ-011 / 013 / 018 / 021 / 024。

Gate CではSemantic / Functional / Visual分離を維持しつつ、First TargetのReference Scenarioから:

- operability / accessibility
- failure / recovery
- performance / resource
- context-bound quality claim

を観測します。

### Gap C — safe authoring

主にRQ-002 / 003 / 010 / 011 / 017 / 021 / 023。

```text
Select
 -> Propose
 -> Preview Semantic Diff / impact
 -> Accept / Reject
 -> Shared Acceptance Boundary
 -> authoritative source
 -> reload / Conformance
```

Studioは第二正本になりません。

### Gap D — knowledge-first / AI / derived cognition

主にRQ-014 / 015 / 016 / 017 / 021 / 023。

```text
retrieval candidate
  != semantic applicability
  != accepted decision
  != execution authority
```

Derived Knowledge / index / trajectoryは削除・再生成可能に保ちます。

2026-10-02のresearch integrationでは、このGapを「より強いmodelを先に導入する」計画にはしません。[TECH_RESEARCH_INTEGRATION_2026-10.md](TECH_RESEARCH_INTEGRATION_2026-10.md)に従い、exact / deterministic → lightweight associative / statistical → bounded local cognition → local model → external modelの段階昇格をReferenceで比較します。どの方式もcandidate generation / ranking / abstentionの範囲に留め、`applicable / incompatible / unknown`、Acceptance、write / runtime authorityは既存の決定論的境界へ戻します。

この追記はplanning refinementであり、RQ-015 / 016等のEvidence tierを変更しません。

### Gap E — Business / cross-view

主にRQ-025 / 026。

Data / Function / Businessの異なる視点を同型のCanonical objectへ増やすのではなく、realistic contentで矛盾を相互検査します。

### Gap F — Provider / production breadth

主にRQ-004 / 005 / 007 / 008 / 009 / 019 / 020 / 024。

actual Oracle等のexternal Evidence、second Provider、production operational boundaryを必要な範囲で継続します。

## 8. 現在の優先順位

1. ~~Gate B actual final Evidenceを取得する~~ — 完了（2026-09-19、R = `e5231839a5dcc62cce1181a2731fcb7c70d16736`）
2. ~~Gate B ReadyならHandover Baseline `B / R`を固定する~~ — 完了
3. ~~First Web Target `T1`でGate C Evidenceとquality baselineを取得する~~ — 完了（2026-09-19、`T = 0ba87a4`）
4. ~~RQ-025 / 026のBusiness Procedure / cross-view actual Evidenceを取得する~~ — 取得（2026-09-20 / 2026-09-21）。独立reviewとPre-ADR再判定が残る
5. RQ-030の閉包条件を決める。ACTOR_CONTEXT圧力がどの不確実性を減らすかと、何が揃えば連続を終えられるかを先に定める（§9、`ISSUE-000053`）
6. RQ-027〜029の担当を決める。**3件とも担当Issueを持たず、RQ-028とRQ-029はADRも持ちません**（RQ-027はADR-0028のみ）。特にRQ-028は、承認に十分なprojectionの条件を定義し、Shared Acceptance Previewがそれを満たすか評価する（2026-09-30追記: RQ-028は[ISSUE-000071](issue/000071-approval-sufficient-projection.md)が担当し、条件案と既存Previewの机上評価を記録済み。RQ-029の担当案は[ISSUE-000040](issue/000040-semantic-component-composition.md)、RQ-027の担当案は[ISSUE-000053](issue/000053-ai-native-development-and-agent-operational-knowledge.md) Phase 1で、新規Issueは起票しない。いずれもowner判断待ち）
7. RQ-027のGapを閉じるか制約として残すかを分けて判断する（owner判断待ちの提案）。(a) `Variation`は、RQ-029と同じ「destination固有の差分に独立した表現が要るか」という問いなので、[ISSUE-000040](issue/000040-semantic-component-composition.md)で判断し、ここでは独立に閉じない。(b) 再入場に十分かどうかの確認は[ISSUE-000053](issue/000053-ai-native-development-and-agent-operational-knowledge.md) Phase 1の比較runで行う。(c) Gate Evidence / Release locatorの機械到達と、受入境界を通らない変更の記録は、当面閉じず次の制約として扱う。RQ-027は「accepted Decisionは、Shared Acceptance Boundaryを通った受理に限り機械到達できる」までを主張し、次は主張しない。Variationはまだ表現を持たず、再入場したAIがVariationを機械的に取得することは保証しない。Gate EvidenceのRelease locatorは索引化されておらず、散文からの参照が残る。受入境界を通らずに変更されたsourceは決定記録を残さないため、記録が無いことを「決定が無い」と読んではならない。
8. Gate C Ready後、Studio / knowledge-first / AI Proposal / Second Targetを並行する。knowledge / AIは`TECH_RESEARCH_INTEGRATION_2026-10.md`の段階昇格をbaselineとし、第二Targetではportable execution artifact / Host差もPressureする
9. Provider / production境界を必要なEvidenceとともに広げる

## 9. 維持コスト監査

RQ-023の観点から、上流整理自体が過剰な管理負担にならないようにします。

- current snapshot文書へ履歴を無制限追記しない
- 同じ意味のRequirementを重複追加しない
- 実行手順の正本を複数作らない
- Gate / PAS / Evidence laneを安易に増やさない
- Quality characteristicごとに専用schema / tool / Issueを作らない
- Evidence nature / failure modeが異なる場合だけartifactを分ける

### 実測 — 2026-09-21

RQ-023のGapをこれまで文章で述べていたので、`main b01aa01`で測りました。

```text
plan/**.md                     304 file    108,246 行
production Go（_test.go以外）    —          44,784 行
```

設計圧力が集中している`tools/reference-business-procedure-check`:

```text
package全体                    171 file     50,832 行   861 test関数
うちTEI packageをimportしない   101 file     33,576 行
actor_context_*                 80 file     31,992 行   494 test関数
うちTEI packageをimportしない    65 file     26,182 行（82%）
```

TEI packageをimportしないtestは、file内で定義したprobe型に対する**設計仮説の圧力**です。Canonical schemaを早期に固定しないための正当な手段であり、それ自体は誤りではありません。実測が示すのは次の三点です。

1. 製品codeに接続しないprobe code（33,576行）が、Repository全体のproduction Go（44,784行）の75%に達している。
2. `actor_context_*`の81 fileは2026-09-18〜19の2日間に作られ（20 file + 61 file）、`R3` → `R3A` → … → `R3BF`とアルファベットで伸びている。production codeを検証するのは`R0` / `R1` / `R2` / `R3` / `R3A` / `R3B` / `R3C`と`R3R` / `R3S` / `R3T`、R6系の一部で、長い後続はprobe型だけを対象にしている。
3. 494あるactor_context test関数のうち、Gate A critical manifest（39件）に載るのは1件である。対応するplan文書も`ACTOR_CONTEXT_*`が63 file・16,981行あり、どの文書も**この連続をどこで終えるか（closure条件）を定義していない**。

この連続が向かうRQ-030は、2026-09-21まで本監査にもPAS mappingにも載っていませんでした。つまり**閉じる対象が定義されていない状態で圧力だけが増えていた**ことになります。数を減らすこと自体は目的にしませんが、RQ-030のPAS（PAS-07）とEvidence tierが定まった以上、次のincrementでは「どの圧力がRQ-030のどの不確実性を減らすか」と「何が揃えば連続を終えられるか」を先に決めます。判断は`ISSUE-000053`が持ちます。

## 10. 結論

現在のTEIは、Application / Driverの主要meaningをactual Evidenceで閉じ、明示Handoverを経てFirst TargetへProjectionする段階です。

Product Qualityはこの主線と競合する別workstreamではなく、**既存RequirementがTarget / Provider / Runtime / Studio / AIへ渡っても壊れないことをEvidenceで確認する横断軸**として扱います。

2026-09-21の再監査で分かった横断的な事実は、**Productの価値定義がその検証マップより先に進んでいた**ことです。2026-10-03のRQ-031〜RQ-033追加では、Requirement、PAS mapping、本監査を同時に更新し、このずれを繰り返さない形にしました。2026-09-18にTEIを「AI-native継続開発とAgent業務遂行の共通知識基盤」へ再定義してRQ-027〜030を追加した一方、PAS mappingと本監査は翌日以降に更新されてもRQ-026で止まり、4件は優先順位にも現れませんでした。その間に、RQ-030へ向かう設計圧力だけが2日で81 file・32K行増えました（§9）。4件のうちRQ-027〜029は現在も担当Issueを持たず、RQ-028とRQ-029はADRも持ちません。作業項目を持つのは`ISSUE-000053`があるRQ-030だけで、圧力が偏った先もそこです。

さらにRQ-027の監査で、この構造がTEI自身へ跳ね返っていることが分かりました。RQ-027は「AIが会話履歴に頼らずApplicationへ再入場できる」ことを求め、そこにaccepted DecisionとEvidence / provenanceを含めています。しかしTEI Repository自身では、その二つだけが機械到達できません。Canonical meaning、source、validation、impactは`tei`コマンドで到達できるのに、判断は索引の無いADR 29件、Evidenceは散文から参照するRelease assetとして置かれています。**TEIは、自分が作ろうとしている共通知識面に、自分の判断とEvidenceをまだ載せていません。**
