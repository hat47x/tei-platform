# TEI Master Plan

## 目的

この文書は、TEI（鼎／テイ）のProduct ValueとProduct Requirementsを、開発上のworkstreamへ分解し、どこを並行して進め、どこで合流させるかを示します。

期限は置きません。優先順位は、実装量ではなく「TEIの中心価値に関する不確実性をどれだけ減らせるか」で見直します。

## 現在地

Repositoryは、M0.22.1までにSemantic / Application / Database / Release / Deployment / Plugin / Runtimeを一本のvertical sliceとして接続し、その後M0.23でSource-to-Canonical Traceability、M0.24〜M0.25でactual process boundary、M0.26以降でactual PostgreSQLとPhysical SchemaのDesired / Actual / Diffを段階的に検証してきました。

M0.32時点では、ProviderExpressionの表現と比較未確定を含め、Database / Provider側の意味境界はかなり深く検証されています。

一方、Product Vision全体から見ると、Application Pattern、既存システム移行、Editor、AI支援、Target-neutral Presentation / Driver等は、Database領域ほど実証が進んでいません。

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
- standard Application Pattern
- provenance / change trace

### 重点

今後は、Source-to-Canonical Traceabilityの次に、**「業務システムで再利用可能な意味知識をどう持つか」**を実証する必要があります。

単にprimitiveを増やすのではなく、受注、参照、header-detail、履歴、状態遷移等の小さい具体例を通して、Micro Kernelと再利用可能Patternの境界を確認します。

## Workstream C: Application / Legacy Compatibility

### 目的

TEIが新規の理想形だけでなく、実際の業務システムを受け止められることを確かめます。

### 対象

- Application ↔ Database N:M
- Database Contract
- read / read-write Binding
- transaction / failure boundary
- existing authentication / authorization
- existing SQL / batch / external API
- Capability Mapping
- incremental replacement

### 重点

既存認証のように、標準化するとかえって移行できなくなる領域を重点的に検証します。

「TEI標準へ作り直してから移行」ではなく、既存仕様を保持したまま取り込み、後からCapability単位で置き換えられることをReferenceで確認します。

## Workstream D: Database / Provider Lifecycle

### 目的

DatabaseのDesired / Actual / Diff / Migration / Contract lifecycleを、Provider固有知識をCoreへ漏らさず成立させます。

### 現在の主な論点

- partition semantics
- provider default / identifier normalization
- offline provider policy
- advanced migration rendering
- additional providerでのConformance

### 進め方

PostgreSQLは深い実証Providerとして継続利用します。ただしPostgreSQL固有機能を網羅することを目的にはしません。

Provider-neutralな境界の妥当性を確認するため、一定の時点でOracle等の異なるProviderへ小さいReferenceを接続し、PostgreSQLで偶然成立している抽象がないか検査します。

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

1. 標準的な業務画面をWebでvertical sliceとして成立させる
2. 同じ上位設計を別TargetへProjectionする
3. その差分からTarget-neutralな境界を修正する
4. Coreを変更せず第三Targetを追加できることを確認する

この順に進めます。

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

## 次に優先して不確実性を減らす領域

M0.32以降は、次の順を基本とします。

1. Product BaselineをRepositoryへ定着させる
2. 現在の実装をProduct Requirementsへ逆マッピングし、空白を明示する
3. standard Application Patternと既存認証互換を小さいReferenceで実証する
4. DriverでApplication設計を動かす最小vertical sliceを作る
5. 同じ設計をWeb Projectionへ接続する
6. Editorの高密度ProjectionとAI確認UXを、そのvertical slice上で試す
7. 第二Targetを追加し、Target-neutral境界を圧力試験する
8. Database / Provider側は、上記を支える範囲とprovider-neutral境界の検証を中心に継続する

既存のDatabase関連Issueは捨てません。ただし、緊急な安全境界や他workstreamの前提でないものは、プロダクト全体の空白と比較しながら着手順を決めます。
