# TEI Milestones

## 方針

TEI（鼎／テイ）のMilestoneには期限を置きません。

Milestoneは「いつまでに終えるか」ではなく、**どのCapabilityが揃えば、次の段階へ進んだとみなせるか**を示します。

個別実装のM0.xxは設計検証上の細かなcheckpointとして継続利用できますが、この文書ではプロダクト全体を見渡す、より大きなMilestoneを扱います。

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
- standard Application Patternを少なくとも数種類、意味を保った再利用部品として表せる

### 意味

「モデルを読み込める」から一段進み、業務設計の知識を蓄積・再利用できる状態です。

M0.23までのSource-to-Canonical Traceabilityは、このMilestoneの基礎をかなり満たしています。今後は意味知識の再利用へ重点を移します。

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

## Milestone 3 — 設計をDriverで動かして確認できる

### 到達条件

- ApplicationのData / Rule / Validation / CapabilityをDriverで実行できる
- Database Bindingを含む小さい業務シナリオを閉じた環境で再現できる
- 設計変更をすぐにValidation / Simulationへ反映できる
- command、actual observation、evidenceを区別できる
- unknown / unresolvedを安全に保持できる
- Reference ScenarioをRepositoryだけから再現できる

### 意味

設計が、静的な文書や定義だけでなく「動かして確かめられる設計」になります。

## Milestone 4 — 一つの実用TargetでApplicationを完成形に近く通せる

### 到達条件

- target-neutral Presentation / Interaction境界がある
- 標準的な業務ApplicationをWeb等の第一TargetへProjectionできる
- browse / record / reference / header-detail / command等のPatternをUIへつなげられる
- Database、Capability、Validation、Authorizationを同じApplication vertical sliceで扱える
- 生成された画面だけでなく、custom UIを接続できる
- Driverと実Targetの意味差を説明できる

### 意味

TEIがarchitecture probeだけでなく、実際の業務Applicationを一つのTargetへ通せる段階です。

## Milestone 5 — 同じ意味を複数Targetへ展開できる

### 到達条件

- 第二TargetをCore変更なしで追加できる
- Target CapabilityとBindingの差を明示できる
- Web / Desktop / Mobile / API / MCP / VUIのうち複数で同じ上位設計を利用できる
- Target固有機能をCoreへ逆流させない
- TargetごとのConformanceと性能条件を定義できる

### 意味

Target-neutralが設計上の主張ではなく、複数Targetの実装で確認された状態です。

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

完成度の高い全Target対応より、ひとつの実用的なvertical sliceを優先します。

### v2 — Target / Provider Expansion

Milestone 5〜8を中心に、TargetとProviderの交換可能性、Editor、AI支援、外部Capabilityとの連携を広げます。

### v3以降 — Operational Maturity

production運用、互換性、供給網、組織規模での再利用等を深めます。

現時点でv3以降の機能を細かく固定しません。v1 / v2で得られた実証結果をもとに、Product Valueから改めて判断します。

## Milestoneの更新

Milestoneは、実装が進んだから自動的に「完了」へ変更しません。

Reference、Conformance、実利用に近い検証から、到達条件を説明できるEvidenceが揃ったときに更新します。

また、実験の結果として到達条件自体が不適切と分かった場合は、過去の計画へ固執せず、Product Valueとの整合を確認したうえでMilestoneを更新します。
