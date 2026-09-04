# Product Requirement Gap Audit

## 目的

この文書は、`requirements/TEI_PRODUCT_REQUIREMENTS.md`で定めたRQ-001〜RQ-024について、現在のRepositoryにどの程度のEvidenceがあり、どこにProduct上の不確実性が残っているかを逆マッピングする監査です。

実装量やコード行数ではなく、次を基準に評価します。

- 意味境界が実装されているか
- 同じReferenceで再現できるか
- Conformance / Evidenceから境界を説明できるか
- 一部だけ成立している場合、残っている不確実性を説明できるか
- 実装済みと実行Evidence取得済みを混同していないか

この監査は「未実装だから問題」という一覧ではありません。TEIのProduct Visionに対して、次にどこへ時間を使うと大きな不確実性を減らせるかを判断するための文書です。

## 評価区分

| 区分 | 意味 |
| --- | --- |
| **Strong Evidence** | 実装、Reference、Conformance / Evidenceが一続きで存在し、主要な意味境界を説明できる |
| **Partial Evidence** | 重要な部分は実証済みだが、Requirement全体や実環境Evidenceまでは届いていない |
| **Design Only** | 方針や契約は整理されているが、Referenceによる実証がまだない |
| **Not Started** | Product Requirementとしては存在するが、対応する実装・Referenceがほぼない |

## 2026-09-04時点の全体像

M0.32時点の旧監査から、Application側は大きく進みました。

特に次は、もはや「未着手」ではありません。

- Application Pattern Stage 1
- 既存認証・認可Compatibility
- Application Driverの主要意味境界
- Validation → Authorization → Command
- Business Result / Execution Error
- Command後の明示Observation read
- browse / record / reference / header-detail / commandをつないだ最終UI-less Scenario

一方、Product Vision全体から見ると、依然としてEvidenceが薄い領域があります。

- 第一TargetでのPresentation / Interaction実証
- 複数Projectionをまたぐ編集体験
- 開発者向けEditor
- Decision / Suggestion履歴を含むAI支援
- 再利用可能な業務知識をAIが探索する経路
- 第二TargetでのTarget-neutral境界の圧力試験
- Targetごとの性能契約

Database / Provider領域は、TEIの他領域より深く検証されています。今後も必要なProvider Evidenceは取得しますが、特定Providerの詳細だけがProduct全体の着手順を決めないようにします。

---

# Requirement別の現在地

## RQ-001 業務上の意味を実装技術から独立して保持できること

**評価: Strong Evidence**

主なEvidence:

- Canonical IDとsource pathの分離
- Semantic Data / Relation / Capability
- Application / Database Requirement
- Application PatternをApplication所有のCanonicalな意図として扱う境界
- Persistence MappingとPhysical Schemaの分離
- Target固有UIやProvider固有実装をCanonical meaningへ逆流させない設計

残るGap:

- Domain Component / Policy等、複数Applicationで再利用する業務知識の実証
- 第一Target・第二Targetを通したmeaningの安定性確認

---

## RQ-002 Authoring sourceとCanonical meaningを追跡できること

**評価: Strong Evidence**

主なEvidence:

- Source Index
- Source Artifact fingerprint / byte range / source role
- native SQLのauthoritative source
- rename / move invariance
- Source Diagnostic
- Application Pattern / Binding / native realizationからauthoring sourceへ戻る経路

残るGap:

- SQL以外のnative sourceに対する同水準のsemantic analysis
- Editor / AI等の新しいProjectionからauthoritative sourceへ戻る実証

---

## RQ-003 一つの関心事に暗黙の複数正本を作らないこと

**評価: Strong Evidence**

主なEvidence:

- native SQLとDerived Query IRの分離
- Source IndexをCanonical Worldから分離
- Editor / AIをCanonical authorityにしない方針
- external Evidenceを`ir.World`へ昇格させない境界
- Handover / Evidence artifactを第二の正本にしない契約

残るGap:

- Editorや自然言語Projectionから編集する場合のend-to-end Evidence

---

## RQ-004 Native assetをnativeなまま利用できること

**評価: Partial Evidence**

主なEvidence:

- native `.sql`
- `CustomGoBinding`
- actual process Plugin
- legacy Authentication / Authorization Compatibility Reference
- external-observed asset向けContract Query / Evidence intake

残るGap:

- 既存batch / external API / configurationを段階移行するReference
- Go source等のsemantic analysis / refactoring
- actual external assetを使った移行Evidence

---

## RQ-005 ApplicationとDatabaseをN:Mで扱えること

**評価: Partial Evidence**

主なEvidence:

- `Application.DatabaseRequirement[]`
- logical Datasource
- read / read-write Access
- Database Contract
- `order_entry`から`sales_core`と`master_data`を別Datasourceとして扱うReference
- browse / reference / header-detailをDatabase境界へ接続するDriver runtime

残るGap:

- 一つの業務処理が複数Databaseをまたぐ場合のtransaction / failure boundary
- external-observed Datasourceを含む実環境E2E

---

## RQ-006 Desired / Actual / Diff / Unresolvedを分離できること

**評価: Strong Evidence**

主なEvidence:

- Desired Physical Schema
- actual PostgreSQL observation
- Schema Diff
- Unresolvedの明示
- ProviderExpression / provenance
- 不確かな差分を破壊的migrationへ自動変換しない境界

残るGap:

- Database以外のTarget / external API等で同じ原則が必要になる場合の実証

---

## RQ-007 Provider固有知識をCoreへ無制限に持ち込まないこと

**評価: Partial Evidence**

主なEvidence:

- PostgreSQL Provider SPI
- ADR-0007によるactual DB driver assemblyの分離
- provider-specific normalization / rendering
- Contract Query SQL Executorとprovider-specific Runnerの分離
- Oracle向けRunnerでもCanonical Contractをdriverへ固定しない境界

残るGap:

- PostgreSQL以外のactual Providerによる抽象境界の圧力試験
- Provider固有policyが残る箇所の継続監査

**優先順位上の注意:** 第二Providerの深掘りは、Provider境界の検証に必要な範囲へ限定します。特定Providerの実環境詳細だけを理由に、Application / Driver / Target / EditorのProduct主線を長期間止めません。

---

## RQ-008 Capability単位で標準実装と固有実装を置き換えられること

**評価: Partial Evidence**

主なEvidence:

- Semantic Capability
- Capability Binding
- Environment Binding
- Plugin / Bundle / Capability negotiation
- legacy Authentication / Authorization Compatibility
- Business Result / Execution Error

残るGap:

- builtin / generated / plugin / existing implementationを同じ業務Capabilityで比較するReference
- 第一Targetまで含む置換Evidence

---

## RQ-009 既存システムの認証・認可仕様を保持した移行が可能であること

**評価: Partial Evidence**

旧監査の`Not Started`から更新します。

成立済み:

- legacy Authentication / Authorization Compatibility Reference
- Authentication / Authorization Capability境界
- Capability Binding
- Gate A Ready
- 最終UI-less ScenarioでAuthentication subjectをAuthorizationへ継承
- legacy schema / return codeをDriver Coreへ逆流させない境界

残るGap:

- 実移行対象を使ったCompatibility Evidence
- 標準実装と既存実装を段階的に置き換えるReference

---

## RQ-010 Editorを開発者向けの高密度な設計環境として提供できること

**評価: Design Only**

設計上の基線:

- EditorはCanonicalを所有しない
- high-densityなtable / graph / flow / UI Projection
- Source / Diagnostic navigation
- shared Acceptance Boundary
- Go + Wails/WebViewを有力Host候補とする

残るGap:

- Editor shell
- 実際の編集Projection
- 同じReferenceを複数視点で編集するEvidence

第一Web TargetでApplication / Presentation境界を観測する前にEditorの詳細UIを固定しません。

---

## RQ-011 同じCanonical meaningを複数のProjectionで扱えること

**評価: Partial Evidence**

主なEvidence:

- Source Index
- Diagnostic
- Database Workbench
- Query IR
- Driver result
- Handover Baselineというderivedな比較参照

残るGap:

- table / graph / flow / UI / natural languageを同じmeaningから切り替えるReference
- Projectionから編集した場合のauthoritative sourceへの反映

---

## RQ-012 Driverを設計検証用の実行・シミュレーション環境として利用できること

**評価: Partial Evidence**

旧監査の`Not Started`から更新します。

成立済み:

- Application Pattern解決
- Datasource / Contract Query runtime
- Capability Binding
- Authentication / Authorization
- Canonical Validation
- Business Result / Execution Error
- Command後の別OperationとしてのObservation read
- browse → record → reference → header-detail → Validation → Authorization → Command → Observationをつないだ最終UI-less Scenario
- deterministic Repository Evidence runner

残るGap:

- Gate Bの実run取得
- 実provider / generated artifact / external assetとのEvidence収束
- 第一Targetへ渡したときにWeb側の意味補完が不要であることの確認

Driverの主要な意味設計は成立しており、現在の中心は新しい抽象ではなくEvidence取得です。

---

## RQ-013 Target-neutralなPresentation / Interaction境界を持つこと

**評価: Design Only**

成立している契約:

- 第一Web Targetを先に実証する
- Target CapabilityとExecution / Trust Profileを分けて観測する
- Gate CでSemantic / Functional / Visual Evidenceを分離する
- 第一Targetだけで完全なPresentation IRを確定しない
- 第二Targetとの差から境界を修正する

残るGap:

- 第一Web Target実装
- 第二Targetとの比較

---

## RQ-014 標準的なApplication Patternを再利用できること

**評価: Partial Evidence**

旧監査の`Not Started`から更新します。

Stage 1で次の5種類をApplication所有の意図として実証しました。

- browse
- record
- reference
- header-detail
- command

成立済み:

- Canonical identity
- authoring source
- Semantic / Query / Relation / Capabilityとの分離
- Driverからの解決

残るGap:

- 複数Applicationでの再利用
- 第一Target / 第二Targetでの圧力試験
- 5分類のうち何を標準知識へ持ち上げるべきかのEvidence

現時点で5分類をTEI全体の永久的な標準Catalogにはしません。

---

## RQ-015 生成AIが既存知識を探索してから新規実装へ進めること

**評価: Design Only**

成立している方針:

- AIはProposal / authoring candidateを生成する
- Canonical authorityとは分離する
- shared Acceptance Boundaryを通す
- Derived Experience Knowledgeを第二の正本にしない

残るGap:

- AIが検索する価値のあるDomain Component / Pattern / Policy資産
- 知識探索protocol
- 実Reference

AI実装より先に、再利用可能な意味知識を増やします。

---

## RQ-016 AIによる確認提案は控えめで、履歴を持つこと

**評価: Design Only**

Requirement /設計上は次を重視しています。

- 既回答を再質問しない
- 提案を無視できる
- 同種提案を抑制できる
- autonomyとauthorityを分離する
- Decision / Proposal historyを第二のCanonicalにしない

残るGap:

- Decision / Suggestion / Resolution / Suppressionの最小モデル
- Editor上の実体験

---

## RQ-017 人の二重管理を最小化すること

**評価: Partial Evidence**

主なEvidence:

- Canonical WorldとSource Indexの分離
- generated artifactのdeterministic regeneration
- native SQLをauthoritative sourceとして扱う
- Desired / Actualの導出
- Gate B generated artifact check-only Evidence

残るGap:

- Editor / AIで一度決めた内容を複数Projectionへ再利用するE2E

---

## RQ-018 CommandとObservationを分離すること

**評価: Strong Evidence**

主なEvidence:

- ADR-0023
- Business Result / Execution Errorの分離
- Command後のObservationをseparate explicit readとするDriver境界
- 最終UI-less Scenario
- Deployment / Runtime側のObservation / Evidence境界

Command acknowledgementをObserved stateへ変換しません。

---

## RQ-019 実行時の権限とResourceを明示的に制約できること

**評価: Strong Evidence**

主なEvidence:

- explicit Permission Grant
- Resource Broker / Lease
- secret / network / filesystem logical binding
- Sandbox Profile
- artifact verification
- Host materialをDurable Recordへ漏らさない境界

production-grade sandboxは別課題ですが、Product Requirementの意味境界は強く検証されています。

---

## RQ-020 再実行・takeover・unknownを安全に扱えること

**評価: Strong Evidence**

主なEvidence:

- Supervisor Ownership / heartbeat
- fencing
- Durable Invocation Handle
- observe / reattach
- no-auto-replay
- unknownをunknownのまま保持

---

## RQ-021 ReferenceとConformanceで再現可能性を示すこと

**評価: Strong Evidence**

主なEvidence:

- Reference Workspace
- golden World / Plugin Lock / Durable Run
- Conformance tests
- source-fixed Evidence runner
- generated artifact check-only runner
- actual Provider向けEvidence tool

Gate Bの未実行Evidenceは残っていますが、Reference / Conformanceを基準に開発する方法自体は定着しています。

---

## RQ-022 完成製品と設計検証段階を明確に区別すること

**評価: Strong Evidence**

README / status /各Evidence文書で、architecture probe、実装済み境界、実行済みEvidence、production-readyを区別しています。

今後v1へ近づく段階では、Experimental / Preview / Supported等の互換性レベルが必要かを検討します。

---

## RQ-023 一人でも維持可能な開発構造を保つこと

**評価: Partial Evidence**

主なEvidence:

- 単一Repository
- package境界
- ADR / Issue / Referenceによる判断履歴
- Go中心の基本技術スタック
- Provider / Target / Host固有差の分離

残るGap:

- Editor / Target Adapter / AI支援が増えた後の保守負担
- Evidence toolや計画文書の増加による運用コスト

同じ意味の文書やEvidence toolを増やしすぎないこと自体を継続監査します。

---

## RQ-024 性能要求をTargetごとに検証可能にすること

**評価: Design Only**

方針:

- Targetごとに代表Scenarioと測定対象を定める
- 根拠のない共通数値を先に固定しない
- 第一Targetと第二Targetの差から性能contractの置き場所を決める

残るGap:

- 実Target
- 性能計測
- Target別contract

---

# 集約結果

## Strong Evidence

- RQ-001 Meaning / Canonical independence
- RQ-002 Source-to-Canonical Traceability
- RQ-003 Authoritative sourceの一意性
- RQ-006 Desired / Actual / Diff / Unresolved
- RQ-018 Command / Observation
- RQ-019 Permission / Resource authority
- RQ-020 Takeover / fencing / unknown
- RQ-021 Reference / Conformance
- RQ-022 Research / Product境界

## Partial Evidence

- RQ-004 Native asset
- RQ-005 Application ↔ Database N:M
- RQ-007 Provider boundary
- RQ-008 Capability substitution
- RQ-009 Legacy authentication / authorization compatibility
- RQ-011 Projection
- RQ-012 Driver
- RQ-014 Application Pattern
- RQ-017 二重管理の削減
- RQ-023 maintainability

## Design Only

- RQ-010 Editor
- RQ-013 Target-neutral Presentation / Interaction
- RQ-015 AI knowledge-first workflow
- RQ-016 AI suggestion / decision UX
- RQ-024 Target performance contract

## Not Started

現時点では、Requirement全体が完全な`Not Started`に留まる項目はありません。

ただしDesign Onlyの領域はReference実証がまだないため、実装済みと読み替えません。

# Product Gapの再統合

個別RQを24本の独立作業として並べるのではなく、Productとして意味の近い課題へ束ねます。

## A. Gate BをEvidenceで閉じる

対象:

- RQ-005
- RQ-009
- RQ-012
- RQ-014
- RQ-021

Application / Driverの意味設計は大きく進みました。ここで新しい抽象を増やすより、成立済み境界を通常Go runner、generated artifact、real PostgreSQL等のEvidenceで確認することを優先します。

外部Oracle等、実環境情報がなければ進められない個別Provider作業は、必要条件が揃うまで保留できます。特定Providerの詳細だけでProduct全体の学習を止めるかどうかは、Gateの目的に照らして別途監査します。

## B. Canonical meaningを第一Targetへ出す

対象:

- RQ-011
- RQ-013
- RQ-024

第一Web Targetでは、Application上の意味をWeb固有コードで補わずProjectionできるかを確認します。

ここで初めて、Target-neutral Presentation候補、Target Capability、Execution / Trust差、Visual / Functional Evidenceの実データが得られます。

## C. 厳格さを開発者の負担へ転嫁しない

対象:

- RQ-010
- RQ-011
- RQ-016
- RQ-017
- RQ-023

EditorはCanonicalを所有せず、同じReferenceを高密度に見て編集するProjectionとして実証します。

## D. 再利用可能な業務知識を育てる

対象:

- RQ-001の残差
- RQ-004
- RQ-008
- RQ-014
- RQ-015

一つのApplicationで便利だった分類をすぐ標準知識へ昇格させず、複数Application / TargetのEvidenceからDomain Component / Pattern / Policy等の再利用単位を抽出します。

AIはその後に載せます。

# 現在の優先順位

2026-09-04時点では、次の順でProduct上の不確実性を減らします。

1. **Gate Bのうち外部Oracleに依存しないEvidenceを先に取得する**
   - #310 deterministic Repository
   - #255 generated artifact reproducibility
   - #308 real PostgreSQL Provider
2. **actual external OracleをGate B必須blockerとする妥当性を、Gate B本来の目的から再監査する**
3. **Gate B Ready条件が満たされたらHandover Baselineを固定する**
4. **第一Web Targetへ同じApplication / Driver meaningをProjectionする**
5. **Gate CでSemantic / Functional / Visual Evidenceを分離して取得する**
6. **第一Targetで見えた実差分から最小Editor / Target-neutral候補を設計する**
7. **第二TargetでTarget-neutral境界を圧力試験する**
8. **複数Referenceから再利用可能な業務知識を抽出する**
9. **その知識を探索・提案するAI支援を実証する**
10. **Target別performance contractをEvidenceから定める**

Provider / Databaseの既存作業は必要な範囲で継続しますが、他workstreamよりEvidenceが深い領域の追加深掘りは、上記Product Gapと比較して着手順を判断します。

# 結論

TEIは、意味を失わずsource・実装・実行・観測へつなぐ下部構造だけを作っている段階から、すでに一歩進んでいます。

Application Pattern、既存認証Compatibility、Application Driverの主要Scenarioは実証が進み、旧Gap Auditの「Not Started」という評価は現状を表さなくなりました。

次に大きく不確実性を減らすのは、特定Database Providerの詳細をさらに深掘りすることではありません。

**成立したApplication / Driverの意味をEvidenceで閉じ、第一Targetへ出し、開発者が扱えるProjectionへ育て、その後に再利用知識とAIへ進むこと**が、現在のProduct主線です。
