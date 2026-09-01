# TEI Milestones

## 方針

TEI（鼎／テイ）のMilestoneには期限を置きません。

Milestoneは「いつまでに終えるか」ではなく、**どのCapabilityとEvidenceが揃えば、次の段階へ進んだとみなせるか**を示します。

個別実装のM0.xxは設計検証上の細かなcheckpointとして継続利用できますが、この文書ではプロダクト全体を見渡す、より大きなMilestoneを扱います。

## v1 Reference Applicationとの対応

Milestone 1〜4は、既存の`order_entry`を育てるv1 Reference Applicationで一続きに検証します。

詳細な作業順は[`V1_REFERENCE_APPLICATION_PLAN.md`](V1_REFERENCE_APPLICATION_PLAN.md)に置き、次のEvidence Gateで段階を区切ります。

```text
Milestone 1 / 2
Application Pattern + 既存認証Compatibility
                ↓
             Gate A
                ↓
Milestone 3
Application Driver
                ↓
             Gate B
                ↓
Milestone 4
第一Web Target
                ↓
             Gate C
```

Gateは新しい機能一覧ではありません。

たとえばGate Aでは、WebやEditorを見なくてもApplication上の意味をDriverへ渡せること、Gate BではUIなしで主要な業務シナリオを再現できること、Gate Cでは第一Targetと共通意味の差を説明できることを確認します。

Milestoneの到達条件とGateのEvidenceが重なる場合は、同じReference / Conformanceを二重に作りません。一つのEvidenceを複数の観点から参照できるようにします。

## Milestone 0 — 上位基線がある

### 到達条件

- Product ValueがRepository上で定義されている
- VisionとPrinciplesがProduct Valueから自然につながっている
- Product Requirementsが固定IDで参照できる
- Master Planと期限なしMilestoneがある
- ADR / Issue / Architecture / Implementationの優先関係が明確である
- 文書を自然な日本語へ整える工程が開発ルールとして明記されている

### 意味

このMilestoneは機能完成を意味しません。

今後の実装が「何を確かめるための作業か」を判断できる地図がある状態です。

## Milestone 1 — Meaningを作り、追跡し、検査できる

### 到達条件

- authoring sourceからCanonical meaningへ追跡できる
- file pathとsemantic identityが分離されている
- native sourceをauthoritativeなまま接続できる
- Diagnosticをmeaningからsourceへ戻せる
- Domain Component / Semantic Pattern等の再利用単位を小さいReferenceで扱える
- Application Patternを少なくとも数種類、意味を保った再利用候補として表せ、どこまでを共有・標準化できるかをEvidenceから説明できる

### 意味

「モデルを読み込める」から一段進み、業務設計の知識を蓄積・再利用できる状態です。

M0.23までのSource-to-Canonical Traceabilityは、このMilestoneの基礎をかなり満たしています。今後は意味知識の再利用へ重点を移します。

v1では、browse、record、reference、header-detail、commandをまずApplicationが所有する意図の候補として実証し、DriverやTarget固有の表示定義と混同しないことを確認します。そのうえで、複数Applicationへ再利用できる部分や標準化できる範囲をEvidenceから判断します。

## Milestone 2 — 現実の既存業務をそのまま受け止められる

### 到達条件

- Application ↔ Database N:Mを実例で扱える
- Database Contract / failure / transaction boundaryを表現できる
- 既存SQLをnativeなまま利用できる
- システム固有の認証・認可仕様を変更せず接続できる
- Capability単位で標準実装と既存実装を切り替えられる
- 既存仕様を保った状態から、段階的に標準部品へ置き換える例を示せる

### 意味

新規の理想化されたサンプルだけでなく、実際のSIer案件で発生する移行要件を受け止められる状態です。

v1では、既存認証CompatibilityをApplication Patternと並行して実証し、認証・認可をWeb Targetへ後付けするのではなく、Gate AでDriverへ渡せるApplication上の意味として合流させます。

## Milestone 3 — 設計をDriverで動かして確認できる

### 到達条件

- ApplicationのData / Rule / Validation / CapabilityをDriverで実行できる
- Database Bindingを含む小さい業務シナリオを閉じた環境で再現できる
- Authentication / Authorization CapabilityをApplication上の意味から評価できる
- 設計変更をすぐにValidation / Simulationへ反映できる
- Capability invocationのBusiness ResultとExecution Errorを区別できる
- command acknowledgement、actual observation、evidenceを区別できる
- unknown / unresolvedを安全に保持できる
- Reference ScenarioをRepositoryだけから再現できる

### 意味

設計が、静的な文書や定義だけでなく「動かして確かめられる設計」になります。

v1のGate Bでは、browse → record → reference → header-detail → validation → authorization → commandをUIなしで再現し、Business Result / Execution Errorと、必要なObservationを意味の違いとして保持できることを確認します。

## Milestone 4 — 一つの実用TargetでApplicationを完成形に近く通せる

### 到達条件

- target-neutral Presentation / Interaction境界の候補を実例から抽出できる
- 標準的な業務ApplicationをWeb等の第一TargetへProjectionできる
- browse / record / reference / header-detail / command等のPatternをUIへつなげられる
- Database、Capability、Validation、Authorizationを同じApplication vertical sliceで扱える
- CapabilityのBusiness Result、Execution Error、Observationを意味の違いを保ったままTargetへProjectionできる
- 生成された画面だけでなく、Custom UIを接続できる
- Driverと実Targetの意味差を説明できる
- 第一Targetに固有の情報と、複数Targetで共有できそうな情報を分けて記録できる

### 意味

TEIがarchitecture probeだけでなく、実際の業務Applicationを一つのTargetへ通せる段階です。

ただし、第一TargetだけでTarget-neutral設計の完成を宣言しません。Gate Cで共通候補とWeb固有差分を明示し、第二Targetから検証し直せる状態を作ります。

## Milestone 5 — 同じ意味を複数Targetへ展開できる

### 到達条件

- 第二TargetをCoreへTarget固有実装を埋め込まず追加できる
- Target CapabilityとBindingの差を明示できる
- Web / Desktop / Mobile / API / MCP / VUIのうち複数で同じ上位設計を利用できる
- Target固有機能をCoreへ逆流させない
- 第一Targetで共通とみなした意味を第二Targetから検証し直せる
- TargetごとのConformanceと性能条件を定義できる

### 意味

Target-neutralが設計上の主張ではなく、複数Targetの実装で確認された状態です。

第二Targetで境界修正が必要になった場合も、それ自体を失敗とはみなしません。第一Targetの都合を共通意味へ持ち上げていなかったかを検証するEvidenceとして扱います。

## Milestone 6 — Editorが日常の開発環境として成立する

### 到達条件

- 開発者向けの高密度な設計ビューがある
- table / graph / flow / UI / source / diagnosticを同じmeaningから切り替えられる
- 複雑な関係を多数の小画面へ分断せず確認できる
- Database Workbench等のderived viewがCanonicalを汚さない
- 設計からDriverへすぐ移れる
- 変更履歴と判断履歴を追える

### 意味

CLIや手書きsource中心の研究実装から、TEI自身を使ってTEI型の開発を行える段階へ進みます。

Editorの詳細interfaceは第一Targetより先に固定せず、Gate CでApplication / Driver / Target差が実資産として揃ってから具体化します。

## Milestone 7 — AI支援が、厳格さを軽いUXへ変える

### 到達条件

- AIが既存Domain Component / Pattern / Policy / Capabilityを探索できる
- 新規コード生成より前に再利用候補を提示できる
- 1:N、関数従属性、履歴、監査、認証等の典型論点を検査できる
- 一度答えた質問を理由なく繰り返さない
- 提案の無視・抑制・OFFができる
- 過去の選択と理由を追跡し、後から変更できる
- AI提案と採用済みの正式判断が明確に分かれている

### 意味

AI機能が追加されたというだけではなく、TEIの厳格な内部構造が利用者の負担を減らす方向に働く状態です。

最初のAI支援は、実在するApplication Pattern、Capability、Driver Scenario、Projection差、Decision履歴を探索できるread-onlyな経路から始め、無確認のauthoritative編集とは分離します。

## Milestone 8 — Provider / Runtimeの交換可能性が実証される

### 到達条件

- PostgreSQL以外のDatabase Providerを小さい実例で接続できる
- Provider固有の正規化・実現知識がCoreへ漏れない
- 外部Runtime / orchestratorをCapabilityとして接続できる
- Plugin / Bundleの配布・trust boundaryが説明できる
- unknown、takeover、fencing、recoveryをProvider差を超えて扱える
- 外部製品を置き換えても上位meaningを作り直さずに済む

### 意味

「交換可能」という主張が、一つのReference実装だけに依存しなくなります。

## Milestone 9 — 実利用へ向けた運用境界が整う

### 到達条件

- production用の永続Store / HA境界が確立される
- credential / secret / resource authorityが運用可能な形で分離される
- signed artifact / registry / supply-chain evidenceを扱える
- compatibility policyとupgrade方針がある
- 観測・監査・障害解析に必要なEvidenceを取得できる
- 重要なCapabilityについてproduction adapterとConformanceがある

### 意味

研究実装を越え、限定された実利用へ責任を持って進められる段階です。

## Version Horizon

Milestoneとは別に、外部へ説明する大きなVersion Horizonを次のように考えます。

### v0 — Architecture / Product Validation

Product Valueと境界を実行可能なReferenceで検証する期間です。

現在のM0.xx系列はここに含まれます。機能数を揃えるより、後から壊しにくい意味境界を確かめることを優先します。

### v1 — Practical Developer Platform

少なくともMilestone 0〜4を満たし、実際の業務Applicationを、設計・検証・Database接続・第一Targetまで一続きに扱える状態を目安とします。

完成度の高い全Target対応より、一つの実用的なvertical sliceを優先します。

EditorとAI支援の完成、第二Target、production HAはv1の前提条件にはしません。これらをv1へ詰め込むより、v1で成立したApplication vertical sliceを基盤として後続段階で検証します。

### v2 — Target / Provider Expansion

Milestone 5〜8を中心に、TargetとProviderの交換可能性、Editor、AI支援、外部Capabilityとの連携を広げます。

### v3以降 — Operational Maturity

production運用、互換性、供給網、組織規模での再利用等を深めます。

現時点でv3以降の機能を細かく固定しません。v1 / v2で得られた実証結果をもとに、Product Valueから改めて判断します。

## Milestoneの更新

Milestoneは、実装が進んだから自動的に「完了」へ変更しません。

Reference、Conformance、実利用に近い検証から、到達条件を説明できるEvidenceが揃ったときに更新します。

また、実験の結果として到達条件自体が不適切と分かった場合は、過去の計画へ固執せず、Product Valueとの整合を確認したうえでMilestoneを更新します。
