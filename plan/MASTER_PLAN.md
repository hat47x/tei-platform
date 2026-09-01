# TEI Master Plan

## 目的

この文書は、TEI（鼎／テイ）のProduct ValueとProduct Requirementsを開発上のworkstreamへ分解し、どこを並行して進め、どこで合流させるかを示します。

期限は置きません。優先順位は、実装量ではなく「TEIの中心価値に関する不確実性をどれだけ減らせるか」で見直します。

## 現在地

Repositoryは、M0.22.1までにSemantic / Application / Database / Release / Deployment / Plugin / Runtimeを一本のvertical sliceとして接続しました。その後、M0.23でSource-to-Canonical Traceability、M0.24〜M0.25でactual process boundary、M0.26以降でactual PostgreSQLとPhysical SchemaのDesired / Actual / Diffを段階的に検証してきました。

M0.35までにDatabase / Provider側では、比較未確定の分離、ProviderExpression、offline provider policy、Advanced Indexの識別子比較、PostgreSQL向けマイグレーションSQLの生成と実DBでの再観測まで、一連の意味境界を深く検証しています。

一方、Product Vision全体から見ると、Application Pattern、既存システム移行、Editor、AI支援、Target-neutral Presentation、Driverなどは、Database領域ほど実証が進んでいません。

今後はDatabase検証を止めるのではなく、**一つの技術領域だけがプロダクト全体の進路を決めないよう、上位のworkstreamを並走させます。**

## Workstream A: Product Baseline

### 目的

Product Value、Vision、Principles、Requirements、Milestone、ADR / Issue運用を、Repository上の基準として成立させます。

### 主な成果物

- `product/`
- `requirements/`
- `plan/`
- 上位文書からREADME / Architecture / Issueへ辿れる導線
- 実装と要求のGap Audit

### 完了の考え方

上位文書が「理念だけの文書」ではなく、個別Issueがどの要求を確かめるものか判断できる状態になったら、基線として扱います。

## Workstream B: Semantic / Authoring / Knowledge

### 目的

業務上の意味を表し、sourceから意味へ戻り、複数のProjectionから同じ内容を扱える基盤を育てます。

### 対象

- Semantic Kernel
- Source Index / Diagnostic
- native source integration
- Domain Component
- Semantic Pattern / Facet
- Policy / Variation
- provenance / change trace
- Applicationから再利用できる意味知識の探索・参照基盤

### 重点

Source-to-Canonical Traceabilityの次は、**「業務システムで再利用可能な意味知識をどう持つか」**を実証する必要があります。

単にprimitiveを増やすのではなく、受注、参照、履歴、状態遷移などの小さい具体例を通して、Micro Kernelと再利用可能な意味知識の境界を確認します。

Application Patternそのものは、v1ではこのworkstreamの所有物にはしません。ADR-0017に従い、Applicationが所有するCanonicalな意図としてWorkstream Cで実証します。その結果から、複数Applicationで再利用すべき部分が見つかった場合に、Workstream BのDomain Component、Semantic Pattern、Policy等とどのように接続するかを判断します。

この順序により、一つのApplicationで便利だった分類を、十分なEvidenceなしにSemantic Kernelや標準知識へ持ち上げることを避けます。

## Workstream C: Application / Legacy Compatibility

### 目的

TEIが新規の理想形だけでなく、実際の業務システムを受け止められることを確かめます。

### 対象

- Application PatternとApplication上の利用意図
- Application ↔ Database N:M
- Database Contract
- read / read-write Binding
- transaction / failure boundary
- existing authentication / authorization
- existing SQL / batch / external API
- Capability Mapping
- incremental replacement

### 重点

v1では、Application PatternをApplicationが所有するCanonicalな意図として扱います。`browse`、`record`、`reference`、`header-detail`、`command`を最初のReferenceとして実証しますが、この分類を直ちにTEI全体の標準Catalogへ固定しません。

Patternの所有境界は[ADR-0017](adr/0017-application-pattern-ownership.md)、正本の配置は[ADR-0018](adr/0018-application-pattern-authoring-source.md)、具体的な実証順は[`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)を参照します。

既存認証のように、標準化するとかえって移行できなくなる領域も並行して検証します。「TEI標準へ作り直してから移行」ではなく、既存仕様を保持したまま取り込み、後からCapability単位で置き換えられることをReferenceで確認します。

Authentication / Authorizationの意味境界は[ADR-0019](adr/0019-authentication-capability-boundary.md)、実装前設計は[`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md)を参照します。

## Workstream D: Database / Provider Lifecycle

### 目的

DatabaseのDesired / Actual / Diff / Migration / Contract lifecycleを、Provider固有知識をCoreへ漏らさず成立させます。

### 現在の主な論点

M0.35までに、provider default / identifier comparison、offline provider policy、Advanced Index migration renderingの主要な安全境界は実装とConformanceまで進みました。

今後の主な論点は次です。

- partition semantics
- PostgreSQL以外のProviderによる抽象境界の圧力試験
- production環境でのonline index build / recovery policy
- provider versionや拡張catalogを含むnormalization authorityの高度化

### 進め方

PostgreSQLは深い実証Providerとして継続利用します。ただし、PostgreSQL固有機能を網羅すること自体を目的にはしません。

Provider-neutralな境界の妥当性を確認するため、一定の時点でOracleなど異なるProviderへ小さいReferenceを接続し、PostgreSQLで偶然成立している抽象がないか検査します。

## Workstream E: Driver / Execution

### 目的

設計内容を、実装後まで待たずに動かして検証できるDriverを成立させます。

### 対象

- closed simulation
- data / rule / validation execution
- capability invocation
- database binding
- deterministic test scenario
- runtime observation
- production runtimeとの境界

### 重点

Driverを本番Runtimeの簡易版だけにはしません。

開発中に「この設計は整合しているか」を確かめる設計検証環境として成立させ、Editorと往復できることを重視します。

UIより先にDriverを成立させることで、Application上の不足をWeb固有コードで補ってしまうことを避けます。

## Workstream F: Target-neutral Presentation

### 目的

ApplicationのPresentation / Interactionを、一つのWeb frameworkへ固定せず複数TargetへProjectionできることを実証します。

### 対象

- target-neutral Presentation IR
- Target Capability
- Runtime Binding
- Web Adapter
- Desktop Adapter
- Mobile Adapter
- API / MCP / VUI Adapter
- Target Conformance

### 着手順

最初から全Targetを同じ深さで作りません。

1. Application PatternとDriverでTarget非依存の意味を確かめる
2. 標準的な業務Applicationを第一Web TargetへProjectionする
3. Webで必要になった共通候補とWeb固有差分を分けて記録する
4. 同じ上位設計を第二TargetへProjectionする
5. 第二Targetとの差からTarget-neutralな境界を修正する
6. Coreを変更せず第三Targetを追加できるか確認する

第一Targetだけで「Target-neutral設計が完成した」とは判断しません。

## Workstream G: Editor / Developer Experience

### 目的

TEIの厳格な意味モデルを、開発者が実用上軽く扱えるEditorへつなぎます。

### 対象

- high-density design view
- table / graph / flow / UI projection
- 3階層以上の関係を保った局所表示
- Database Workbench
- Source / Diagnostic navigation
- decision history
- reversible choices
- Wails / WebView based desktop shell

### 重点

EditorはCanonicalを所有しません。Projectionと編集支援を担います。

また、単純なno-code UIを目指すのではなく、複雑な業務システムを扱う開発者が高い情報密度で全体を追えることを優先します。

Application Pattern、Driver、第一Targetの実証から意味境界が見えるまでは、Editorの詳細interfaceを先に固定しません。

## Workstream H: AI-assisted Engineering

### 目的

生成AIを、コード生成器だけでなく、既存知識の探索、設計検査、実装候補作成へ使います。

### 対象

- component / pattern search
- requirement suggestion
- relation / functional dependency check
- history / audit / lifecycle suggestion
- implementation mapping generation
- natural-language projection
- choice history
- suggestion suppression

### UX原則

AIは、使えるからといって常時話しかけません。

- 既回答を再質問しない
- 重要性の高い論点を優先する
- 提案を無視できる
- 同種提案を抑制できる
- 採用済み判断を明示する
- 後から判断理由と履歴へ戻れる

このUXはAI機能の付加価値ではなく、TEIの「厳格だが軽い」を成立させる主要要件として扱います。

AI支援の詳細も、実在するPattern、Driver Scenario、Projection差、Decision履歴を入力として扱える段階になってから具体化します。

## Workstream I: Ecosystem / Production Boundary

### 目的

Coreと外部世界の境界を、実際の利用に耐える形へ段階的に広げます。

### 対象

- Plugin distribution / trust
- signed artifact
- remote registry
- production Provider adapters
- durable store / HA
- observability
- deployment integrations
- supply-chain evidence
- compatibility policy

### 進め方

外部orchestratorやworkflow engineが既に提供している機能は、TEI固有の価値でない限り積極的に再実装しません。

TEIは、意味、Binding、Evidence、Conformanceを保持し、実行手段は外部Capabilityへ委譲できることを基本とします。

## Workstream間の依存

大まかな関係は次のとおりです。

```text
A Product Baseline
      ↓
B Semantic / Knowledge ───┬── C Application / Legacy
                          │
                          ├── D Database / Provider
                          │
                          └── E Driver
                               ↓
                    F Target-neutral Presentation
                               ↓
                    G Editor / Developer Experience
                               ↕
                    H AI-assisted Engineering

D / E / F / G ───────────────→ I Ecosystem / Production
```

実際には完全な直列ではありません。B〜Hは小さいvertical sliceを共有しながら並走します。

## v1 Reference Applicationでworkstreamを合流させる

上記のworkstreamを抽象的な並行作業のままにせず、v1では既存の`order_entry`へ合流させます。

詳細な実証順とEvidence Gateは[`V1_REFERENCE_APPLICATION_PLAN.md`](V1_REFERENCE_APPLICATION_PLAN.md)にまとめています。

主線は次のとおりです。

```text
Application Pattern ────────┐
      ↓                     │
Application Driver     既存認証Compatibility
      ↓                     │
第一Web Target ─────────────┘
      ↓
Editor / AI支援
      ↓
第二Target
```

この流れでは、日付ではなく次のEvidence Gateで進行を判断します。

### Gate A — Application上の意味をDriverへ渡せる

WebやEditorを見なくても、何をbrowseし、何をreferenceし、どのValidationとCommandを使い、誰が操作でき、どのDatasourceへwriteできるかを説明できることを確認します。

### Gate B — UIなしで主要シナリオを再現できる

Driverだけでbrowse → record → reference → header-detail → validation → authorization → command → observationを再現できることを確認します。

### Gate C — 第一Targetとの差を説明できる

WebとDriverが同じApplication上の意味を共有し、共通Presentation候補とWeb固有差分を追跡できることを確認します。

### Gate D — 第二Targetで圧力試験できる

第一Targetで共通とみなした部分とTarget固有とみなした部分が追跡可能で、第二Targetから検証し直せる状態を確認します。

Gateを設ける目的は手続きを増やすことではありません。実装しやすい層の都合で、未実証の意味を上位モデルへ固定することを避けるためです。

## 次に優先して不確実性を減らす領域

M0.35以降は、次の順を基本とします。

1. Product BaselineをRepositoryへ定着させる
2. 現在の実装をProduct Requirementsへ逆マッピングし、空白を明示する
3. Application Patternと既存認証Compatibilityを`order_entry`周辺の小さいReferenceで実証する
4. Gate Aを確認する
5. DriverでApplication設計を動かす最小vertical sliceを作る
6. Gate Bを確認する
7. 同じ設計を第一Web TargetへProjectionする
8. Gate Cを確認してEditor / AI支援の具体化へ進む
9. 第二Targetを追加し、Target-neutral境界を圧力試験する
10. Database / Provider側は、上記を支える範囲とprovider-neutral境界の検証を中心に継続する

Application Patternについては、最初から「標準Pattern Catalogの完成」を目標にしません。v1 ReferenceでApplication上の意図として実証し、複数Applicationや複数Targetから再利用Evidenceが得られた段階で、どこまでを標準知識として共有するかを判断します。

既存のDatabase関連Issueは捨てません。ただし、緊急な安全境界や他workstreamの前提でないものは、プロダクト全体の空白と比較しながら着手順を決めます。
