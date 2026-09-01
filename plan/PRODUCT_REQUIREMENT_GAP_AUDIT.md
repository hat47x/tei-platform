# Product Requirement Gap Audit

## 目的

この文書は、`requirements/TEI_PRODUCT_REQUIREMENTS.md`で定めたRQ-001〜RQ-024について、M0.32時点のRepositoryにどの程度のEvidenceがあるかを逆マッピングした監査結果です。

ここで評価するのは、実装量やコード行数ではありません。

- 実装が存在するか
- Referenceで再現できるか
- ConformanceやCIで意味境界を説明できるか
- 要求の一部だけを満たしている場合、その残差を説明できるか

を基準とします。

この監査は「未実装だから問題」という評価ではありません。TEIのProduct Visionに対して、どの不確実性が既に減っており、どこがまだ大きく残っているかを把握するためのものです。

## 評価区分

| 区分 | 意味 |
| --- | --- |
| **Strong Evidence** | 実装、Reference、Conformance / CIが一続きで存在し、主要な意味境界を説明できる |
| **Partial Evidence** | 重要な一部は実証済みだが、Requirement全体には届いていない |
| **Design Only** | 設計思想や文書上の方針はあるが、Referenceによる実証がない |
| **Not Started** | Product Requirementとしては存在するが、対応する実装・Referenceがまだほぼない |

## 全体像

M0.32時点のTEIは、次の領域に強いEvidenceがあります。

- Canonical identityとsourceの分離
- native SQLを含むSource-to-Canonical Traceability
- ApplicationとDatabaseの基本的なN:M構造
- DatabaseのDesired / Actual / Diff
- commandとObservation / Evidenceの分離
- Plugin / Resource authority
- runtime takeover / fencing / unknown
- Reference / Conformanceによる再現性

一方、Product Vision上は重要でありながら、実証がほとんどない領域があります。

- 既存認証・認可を保持した移行
- standard Application Pattern
- Driverによる設計シミュレーション
- target-neutral Presentation / Interaction
- 開発者向けEditor
- 複数Projectionをまたぐ編集体験
- AIによる知識探索、確認提案、判断履歴

この非対称性が、現在もっとも重要な計画上の発見です。

---

## RQ-001 業務上の意味を実装技術から独立して保持できること

**評価: Strong Evidence**

### Evidence

- `internal/core`のCanonical ID
- `internal/semantic`のNamespace / Type / Data / Relation / Capability
- `internal/application`のApplicationとDatabaseRequirement
- Source IndexをCanonical Worldから分離
- file rename / moveでnormalized Worldを変えないM0.23 Conformance
- Persistence MappingとPhysical Schemaの分離

### 残るGap

Semantic KernelはType / Data / Relation / Capabilityを中心に成立していますが、Domain Component、Pattern、Policy等の上位再利用単位はまだReferenceで実証されていません。

RQ-001そのものの基礎は強い一方、Milestone 1で必要な「再利用可能な業務知識」へは次の段階があります。

---

## RQ-002 Authoring sourceとCanonical meaningを追跡できること

**評価: Strong Evidence**

### Evidence

M0.23で次をend-to-endに確認済みです。

- Canonical IDとpathの分離
- Source Artifact fingerprint
- byte range
- source role
- Semantic SubjectからSource ReferenceへのDiagnostic projection
- parse errorのsource-direct diagnostic
- native `.sql`のauthoritative source
- rename / move invariance
- `tei source-index` CLI

### 残るGap

SQL以外のnative sourceについて同じ水準のsemantic analysisはまだありません。ただし、RQ-002の基礎契約自体は成立しています。

---

## RQ-003 一つの関心事に暗黙の複数正本を作らないこと

**評価: Strong Evidence**

### Evidence

- SQL-authored Queryではnative SQLをauthoritative sourceとして扱う
- Query declarationとSQL bodyを異なるsource roleとして保持
- Derived Query IRへphysical pathを混ぜない
- derived artifactからauthoritative sourceへ暗黙に逆書きするAPIを設けない
- ambiguousなauthoritative sourceをheuristicにmergeしないM0.23方針

### 残るGap

Graph-authored、UI-authored等、将来の別authoring modeとの相互運用は未実証です。新しいProjectionを導入するとき、この原則を再度Conformanceで確認する必要があります。

---

## RQ-004 Native assetをnativeなまま利用できること

**評価: Partial Evidence**

### Evidence

- native `.sql`をauthoritative sourceとして利用できる
- Query analyzerへpathを持ち込まずSQL本文を流せる
- `CustomGoBinding`によりCapabilityとGo実装を接続する基本モデルがある
- actual process Pluginを外部実装として実行できる

### 残るGap

- Go sourceのsemantic analysis / refactoringは未実装
- 既存バッチや外部設定を意味上のsubjectへ接続する一般Referenceがない
- native assetをApplication移行でどのように段階的に取り込むかのReferenceがない

### 次の作業

`ISSUE-000002`と`ISSUE-000003`のApplication / Legacy vertical sliceで検証します。

---

## RQ-005 ApplicationとDatabaseをN:Mで扱えること

**評価: Partial Evidence**

### Evidence

- `Application.DatabaseRequirement[]`
- logical alias
- target Database Canonical ID
- read / read-write Access
- Required Database Contracts
- Database ReleaseとApplication Releaseの独立lifecycle
- multi-DB compatibilityを含むRelease Graph / Deployment validation

### 残るGap

N:M構造そのものは実装済みですが、Requirementに含めた次の利用者視点の意味はまだ十分にReference化されていません。

- Databaseごとのtransaction boundary
- Databaseごとのfailure boundary
- 一つの業務処理が複数Databaseをまたぐ場合の設計上の扱い
- Application Pattern / UI / Driverから複数Datasourceを自然に利用する経路

### 次の作業

v1 Application vertical sliceで、複数Datasourceを実際の業務シナリオから利用して確認します。

---

## RQ-006 Desired / Actual / Diff / Unresolvedを分離できること

**評価: Strong Evidence**

### Evidence

- Persistence MappingからDesired Physical Schemaを生成
- actual PostgreSQLからActual Physical Schemaを観測
- Schema Diff
- M0.31で確定Changeと比較未確定Unresolvedを分離
- M0.32でProviderExpressionのauthored / provider-normalized representationとprovenanceを分離
- Unresolvedを破壊的migrationへ自動変換しない

### 残るGap

現在の深い実証はDatabase Schemaを中心としています。将来、Target Capability、external API、Runtime state等でも「差分」と「比較未確定」を区別すべき場面が出た場合、概念を無理に共通frameworkへ一般化せず、同じ原則が必要か個別に検討します。

---

## RQ-007 Provider固有知識をCoreへ無制限に持ち込まないこと

**評価: Partial Evidence**

### Evidence

- PostgreSQL Provider SPI
- actual DB driver assemblyをCoreから分離したADR-0007
- ProviderExpressionにProvider / Dialect provenanceを保持
- PostgreSQL catalog normalizationを監査し、provider側normalizationとprovider-neutral model変更を分離

### 残るGap

- Desired Physical realizationのprovider固有policyがPersistence側へ残っている
- provider default / identifier normalizationの契約が未完
- PostgreSQL以外のactual Providerで同じ抽象を圧力試験していない

### 関連作業

- GitHub Issue #58
- GitHub Issue #59
- 将来の第二Database Provider Reference

---

## RQ-008 Capability単位で標準実装と固有実装を置き換えられること

**評価: Partial Evidence**

### Evidence

- Semantic Capability
- Plugin / Bundle / Capability negotiation
- Environment Binding
- Plugin Lock
- in-process / process boundary
- `CustomGoBinding`
- `std.*` prelude / namespace方針

### 残るGap

Requirementで想定する次の交換を、一つの業務Capabilityで比較するReferenceがありません。

- builtin
- generated implementation
- plugin
- existing implementation

また、Application側からCapabilityを利用し、Driver / Targetまで通す利用者視点の経路も未実証です。

### 次の作業

既存認証Compatibility Referenceで、同じAuthentication Capabilityに複数実現方式を接続する形を検討します。

---

## RQ-009 既存システムの認証・認可仕様を保持した移行が可能であること

**評価: Not Started**

### 現状

Permission / Resource authorityはPlugin実行権限のために存在しますが、業務Applicationのユーザー認証・認可とは別の問題です。

既存システムにある、たとえば次の仕様を保持したReferenceはありません。

- user ID / password
- disabled flag
- validity period
- login failure count
- lock state
- organization / role
- system-specific authorization rule

### 次の作業

`ISSUE-000003`でCompatibility Referenceを設計します。

これはv1へ向けた優先度の高いGapです。

---

## RQ-010 Editorを開発者向けの高密度な設計環境として提供できること

**評価: Not Started**

### 現状

`docs/status.md`でもproduction-ready GUI Editorは明示的な非目標です。

Source Index、Diagnostic、Database Workbench等、Editorが利用できる下位のderived modelはありますが、Editor shellや編集Projectionはまだありません。

### 次の作業

Editorを単独で先に作り込まず、v1 Application vertical sliceとDriverの意味が成立した後、その同じReferenceを編集する最小Editorを設計します。

---

## RQ-011 同じCanonical meaningを複数のProjectionで扱えること

**評価: Partial Evidence**

### Evidence

- Source IndexはCanonical Worldとは別のderived projection
- Database Workbench derived model
- Diagnostic projection
- Query IR / SQLのauthoritative / derived分離
- README / Architecture上でProjection概念を明示

### 残るGap

table / graph / flow / UI / natural language等を、同じmeaningから実際に切り替えるReferenceはありません。

特に「Projectionから編集したとき、どこがauthoritativeなのか」をRQ-003と矛盾なく成立させる必要があります。

---

## RQ-012 Driverを設計検証用の実行・シミュレーション環境として利用できること

**評価: Not Started**

### 現状

Deployment executor example、process Plugin example、runtime reattach example等の実行可能Referenceは存在します。

しかし、それらは「ApplicationのData / Rule / UI / Capabilityを、設計途中に閉じた環境で動かすDriver」というProduct Requirementを満たすものではありません。

### 次の作業

v1 Application vertical sliceを定義した後、最初の大きな実装GapとしてDriverの最小vertical sliceを起票します。

---

## RQ-013 Target-neutralなPresentation / Interaction境界を持つこと

**評価: Not Started**

### 現状

RepositoryにはWeb / Desktop / Mobile / API / MCP / VUIを共通上位設計から扱うPresentation IRやTarget Adapter境界はまだありません。

### 次の作業

第一Targetを一つ選び、Application vertical sliceを通して必要なPresentation意味を抽出します。

最初から抽象IRを大きく設計せず、第一Targetと第二Targetの差からTarget-neutral境界を確定する方針を維持します。

---

## RQ-014 標準的なApplication Patternを再利用できること

**評価: Not Started**

### 現状

Semantic Data / Relation、Query、Application、Database等のprimitiveはありますが、次のようなApplication Patternを意味上の再利用単位として表す実装はありません。

- browse
- record
- reference
- header-detail
- command

### 次の作業

Pattern単体を抽象設計するより、v1 Reference Application内で複数Patternを使い、共通部分が実際にどこに現れるかを確認してから固定します。

これはMilestone 1とMilestone 4の間をつなぐ優先度の高いGapです。

---

## RQ-015 生成AIが既存知識を探索してから新規実装へ進めること

**評価: Design Only**

### 現状

README / overviewでは、AIがDomain Component → Pattern → Policy → Composition → Capability → Custom Implementationの順に探索する方針を明示しています。

しかし、検索対象となるDomain Component / Pattern / Policy自体がまだ十分に実装されておらず、AIがRepository上の知識を実際に探索するprotocolもありません。

### 次の作業

先にRQ-014の再利用可能Patternを成立させます。その後、その資産をAIが探索する最小Referenceを作ります。

---

## RQ-016 AIによる確認提案は控えめで、履歴を持つこと

**評価: Not Started**

### 現状

次のUXはProduct Requirementとして定義されていますが、対応するdomain modelやReferenceはまだありません。

- 一度回答した質問を繰り返さない
- 選択と理由を追跡する
- 後から変更する
- 提案を無視する
- 提案カテゴリを抑制 / OFFする
- AI提案と採用済み判断を区別する

### 次の作業

AIの会話UIを先に作るのではなく、Decision / Suggestion / Resolution / Suppressionの最小意味モデルが必要かを、Editor vertical slice上で検討します。

---

## RQ-017 人の二重管理を最小化すること

**評価: Partial Evidence**

### Evidence

- Canonical WorldとSource Indexを分離しつつdeterministicに再生成
- World / Plugin Lock / Durable Run fixtureの再生成一致
- native SQLをsource of truthとしてQuery IRをderivedにする
- Desired Physical SchemaをPersistence Mappingから導出
- actual stateを観測から再生成

### 残るGap

設計者がEditorで選んだMappingやAIとの確認結果を、どのように一度だけauthoritativeに保持し、複数Projectionへ再利用するかは未実証です。

RQ-016のdecision historyと合わせて検討します。

---

## RQ-018 CommandとObservationを分離すること

**評価: Strong Evidence**

### Evidence

- Durable Deployment Run
- explicit Executor / Observation / Evidence routing
- evidence gate
- canary / traffic shift / drain / retire
- command acknowledgementだけで完了にしない
- Database actual introspection

このRequirementはTEIの現行実装で最も強く実証されている領域の一つです。

---

## RQ-019 実行時の権限とResourceを明示的に制約できること

**評価: Strong Evidence**

### Evidence

- explicit Permission Grant
- Resource Broker
- secret / network / filesystem logical binding
- Resource Lease
- Sandbox Profile
- artifact verification
- Host-side Lease Supervisor
- Durable RecordへHost materialを漏らさないConformance

production-grade sandbox自体は未実装ですが、Requirementが求める権限・Resource境界の基礎は強く検証されています。

---

## RQ-020 再実行・takeover・unknownを安全に扱えること

**評価: Strong Evidence**

### Evidence

- Supervisor Ownership / heartbeat
- monotonic fencing
- Durable Invocation Handle
- runtime observe / reattach
- missing時のno-auto-replay
- unknownをunresolvedのまま保持
- Supervisor generationとruntime-confirmed generationの分離

production-grade distributed ownership storeは未実装ですが、意味境界はConformanceで明確です。

---

## RQ-021 ReferenceとConformanceで再現可能性を示すこと

**評価: Strong Evidence**

### Evidence

- Reference Workspace
- golden World
- Plugin Lock
- observed state fixture
- Durable Run fixture
- M0.22.1〜M0.32のConformance文書
- actual process Reference
- actual PostgreSQL CI
- deterministic regeneration

TEIの開発方法そのものとして既に定着しているRequirementです。

---

## RQ-022 完成製品と設計検証段階を明確に区別すること

**評価: Strong Evidence**

### Evidence

READMEと`docs/status.md`は、現在がarchitecture probe / research implementationであり、完成Editor、production Runtime、HA等を提供していないことを明示しています。

個別Conformance文書も、確認できた範囲とproduction-readyを意味しない範囲を分けています。

### 残るGap

今後v1へ近づくにつれ、機能単位でExperimental / Preview / Supported等の互換性レベルを設ける必要が出る可能性があります。

---

## RQ-023 一人でも維持可能な開発構造を保つこと

**評価: Partial Evidence**

### Evidence

- 単一Repository内にCore、Provider、Reference、Docs、CIを保持
- package境界でApplication / Database / Plugin等を分離
- ReferenceをcloneしたRepositoryだけから再現可能
- ADR / Issue / Conformanceに判断を残す開発方法

### 残るGap

これは機能Conformanceだけでは測りにくい要求です。

今後Editor、Target Adapter、AI支援が増えたときに、package / repository分割がmaintainer負担を増やしていないかを定期的に監査する必要があります。

---

## RQ-024 性能要求をTargetごとに検証可能にすること

**評価: Not Started**

### 現状

現行CIは正しさと再現性を中心としており、Application / Targetごとの性能contractはありません。

process Pluginのbounded stdout等、安全上のlimitはありますが、これは利用者向け応答性能とは別です。

### 次の作業

Target-neutral Presentation / Driverが成立してから、Targetごとに測定対象と代表シナリオを定義します。

根拠のない共通数値を先に固定しません。

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
- RQ-011 Projection
- RQ-017 二重管理の削減
- RQ-023 maintainability

## Design Only

- RQ-015 AI knowledge-first workflow

## Not Started

- RQ-009 Legacy authentication / authorization compatibility
- RQ-010 Editor
- RQ-012 Driver
- RQ-013 Target-neutral Presentation
- RQ-014 Application Pattern
- RQ-016 AI suggestion / decision UX
- RQ-024 Target performance contract

# KJ法によるGapの再統合

個別RQの未実装をそのまま24本の作業へ分けると、Productとしてのまとまりを失います。

未実証領域を意味の近さから束ねると、次の4つの島になります。

## 島A: Applicationを現実の業務へ接続する

- RQ-004 Native asset
- RQ-005 multi-DB Application
- RQ-008 Capability substitution
- RQ-009 Legacy authentication
- RQ-014 Application Pattern

中心課題は、**現在のCore / Database / Capability資産を、現実の一つの業務Applicationとして成立させること**です。

## 島B: 設計を動かし、Targetへ出す

- RQ-011 Projection
- RQ-012 Driver
- RQ-013 Target-neutral Presentation
- RQ-024 Performance

中心課題は、**Canonical meaningを静的モデルのままにせず、設計中に動かし、実TargetへProjectionすること**です。

## 島C: 厳格さを人の負担へ転嫁しない

- RQ-010 Editor
- RQ-016 AI suggestion / decision UX
- RQ-017 二重管理の削減

中心課題は、**厳格な意味モデルを、人が日常的に扱える開発体験へ変換すること**です。

## 島D: 再利用知識をAIが探索する

- RQ-014 Application Pattern
- RQ-015 AI knowledge-first workflow
- RQ-016 AI suggestion UX

中心課題は、**AIが検索する価値のある業務知識を先に作り、その後AIを載せること**です。

# 優先順位

監査結果から、M0.32以降の優先順位を次のように更新します。

1. **Application vertical sliceの具体化**
2. **既存認証Compatibility Reference**
3. **standard Application Patternの抽出**
4. **Driverの最小vertical slice**
5. **第一TargetへのPresentation Projection**
6. **Editorの最小高密度Projection**
7. **Decision / Suggestion履歴を含むAI支援UX**
8. **第二TargetでTarget-neutral境界を圧力試験**
9. **Target別performance contract**

Database / Providerの既存Issueは継続しますが、他workstreamを塞いでいない深掘りについては、このProduct Gapと比較して着手順を判断します。

# 結論

M0.32までのTEIは、**意味を失わず、実装・実行・観測へ追跡可能につなぐための下部構造**について、想定以上に強い基礎を持っています。

次に必要なのは、DatabaseやRuntimeの境界をさらに細かくすることだけではありません。

その基礎を使って、

**現実の業務Applicationを組み、設計中に動かし、Targetへ出し、人とAIが無理なく編集できるか**

を確かめる段階へ移ることが、Product Valueから見て最も大きな不確実性を減らします。
