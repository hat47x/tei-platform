# TEIの開発全体計画

## 目的

この文書では、TEI（鼎／テイ）の製品価値と要求を開発上の作業領域に分け、優先して進める作業と並行して進められる作業を示します。

期限は設けません。優先順位は実装量ではなく、**TEIの中心価値に関する不確実性をどれだけ減らせるか**で見直します。

この文書には過去の判断を積み重ねず、現在の計画の基準を記します。以前の判断とその理由は、Gitの履歴、設計判断記録、課題からたどります。

関連文書は次のとおりです。

- `product/`
- `requirements/`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/MILESTONES.md`
- `plan/research/TECH_RESEARCH_INTEGRATION_2026-10.md` ：直近の調査結果を製品価値や既存の課題に結び付ける統合計画
- `plan/REFERENCE_MODEL_TRANSFER_GATE_2026-10.md` ：成熟した外部の参照モデルをプラグイン中心で移し替え、TEIの表現力や暗黙の前提を横断的に検証する計画

## 現在の主線

第1版の参照アプリケーション`order_entry`を中心に、現在は次の順で進めます。

```text
Product Value / Requirements
        ↓
Application Pattern
        ↓
Gate A — Ready
        ↓
Application Driver
        ↓
Gate B — 3 Required Evidence / Ready（R = e5231839a5dcc62cce1181a2731fcb7c70d16736）
        ↓
Handover Baseline B at revision R — 固定済み
        ↓
First Web Target build T1 — Evidence取得済み（T = 0ba87a4e0a96…）
        ↓
Gate C — PAS-04 Evidence — Ready
        ↓
        ├──────── Authoring / Knowledge lane
        │           ├─ Studio minimal loop — PAS-01
        │           ├─ knowledge-first exploration — PAS-06
        │           └─ AI Proposal / Acceptance — PAS-01 / PAS-06
        │
        └──────── Second Target pressure-test lane — PAS-04 / RQ-024
```

判定段階Cの後は、定義の記述と知識の活用、第二の実行先の検証を、互いの完成を待たずに進めます。最初の実行先で確認した違いと検証根拠を共有しながら、並行して作業できます。

## 現在の判定段階と引継ぎ

### 判定段階A：アプリケーションの意味を実行・検証環境へ渡せる

**Ready**です。

アプリケーションの構成パターン、データソース、検証、認証・認可、機能と実装の対応付け、業務上の結果と実行エラーなどを、画面固有の定義なしで実行・検証環境へ渡す意味の基準が成立しています。

### 判定段階B：画面なしで主要な業務シナリオを再現できる

**Ready**です（2026-09-19の再監査、候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）。

必要な検証根拠は次の三つです。いずれも、変更のない同一の候補版`R`を使った一度の実行で成立しています。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

`#387`での最終確認と、同じ実行で行った子成果物の監査を含む`Ready`判定の正本は`GATE_B_READINESS_AUDIT.md`です。

PostgreSQL以外の実際のデータベースで得る検証根拠も重要です。最初の参照事例ではOracleに関する#214、#371、#376が該当します。ただし、これらは判定段階Bにおける四つ目の必須条件には含めません。未完了の範囲は、`unresolved`として引き継ぎます。

### 引継ぎの基準：判定段階BからCへ

引継ぎは、新しい番号付きの判定段階ではありません。

判定段階Bで`Ready`となった候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`は、引継ぎの基準`B`として固定済みです。正式な記録は`GATE_B_TO_C_HANDOVER_BASELINE_B.md`にあります。

```text
Gate B Ready for R
        ↓
Handover Baseline B
  baselineRepositoryRevision = R
```

引継ぎの記録は、意味の第二の正本ではありません。

### 判定段階C：最初の実行先で生じる違いを検証根拠で説明できる

**Ready**です（T = `0ba87a4e0a96f70a198b03ff7652fb00e3826abe`、2026-09-19）。Gate B / Handoverを起点に、第一Web Target build `T1`のSemantic / Functional / Visual Evidenceが同一provenanceでそろいました（[GATE_C_READINESS_AUDIT.md](gate/GATE_C_READINESS_AUDIT.md)）。Readyの非主張（production-ready、actual external Commandの成功、Target-neutral Presentation taxonomyの完成、第二Targetが不要になったこと、Visual Fidelityの人による承認）は同auditのとおりです。

最初のWeb向け実行先では、引継ぎの基準`B`と対象版`R`を意味の基準として使います。

Web向け実装の版は`T1`として別に追跡します。

```text
Baseline meaning B at R
        ↓
First Web Target build T1
        ↓
Semantic / Functional / Visual Evidence
```

`T1`で基準となる対象の意味を変更した場合は、そのまま作業を続けず、新しい引継ぎの基準を発行します。

判定段階Cでは次の三つを分けます。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence

画像を評価するAIやスクリーンショットの評価点だけを根拠に、意味や機能の検証に合格したとは判断しません。

## 作業領域A：製品の基準と追跡可能性

### 目的

製品価値、要求、製品受入シナリオ、設計判断記録、計画上の課題、参照事例、検証根拠、判定段階を一続きに追跡できる状態を維持します。

### 重点

```text
Product Requirement
        ↓
Product Acceptance Scenario
        ↓
ADR / Plan Issue
        ↓
Reference / Fixture
        ↓
Evidence
        ↓
Gate / Product claim
```

PASは単なる網羅率の指標ではなく、要求と実証を結び付ける追跡単位として扱います。

## 作業領域B：意味の定義・記述・知識

### 目的

定義元のソースから業務上の意味を再構成し、複数の表示・編集形式で同じ正本の意味を扱える基盤を整えます。

### 対象

- Semantic Kernel
- Source Index / Diagnostic
- Source-to-Canonical Traceability
- Domain Component / Semantic Pattern / Policy候補
- provenance / revision
- reusable semantic knowledge exploration

第1版では、アプリケーションの構成パターンはアプリケーション側で管理します。最初の参照事例だけを根拠に、意味を扱う中核の標準的な概念一覧へ移しません。

AI向けの知識、検索索引、埋め込み表現、処理の履歴などは派生情報として扱い、意味の第二の正本にはしません。

判定段階Cの後に行う読み取り専用の探索は、`ISSUE-000047`でPAS-06として実証します。

## 作業領域C：アプリケーションと既存資産の互換性

### 目的

新規理想形だけでなく、既存業務資産を保持しながら段階移行できることを確認します。

### 対象

- Application Pattern / intent
- Application ↔ Database
- Database Contract / native Query
- Datasource read / write
- existing authentication / authorization
- existing SQL / external API
- Capability Mapping
- incremental replacement

実際の外部資産については、観測した事実と正本に定めた契約を区別します。

実際には観測していない物理オブジェクト、列、SQL、データ型を推測で補いません。

成熟したReference Modelの転写は`ISSUE-000078` / `REFERENCE_MODEL_TRANSFER_GATE_2026-10.md`で扱います。転写で必要になった業務・設計概念は原則Semantic Extension / Pluginとして表し、X-TEA固有schemaをCoreへ直接追加しません。外部model本体をpublic fixtureへコピーせず、exact input digest、Source locator、mapping、loss / unresolved、Conformance Evidenceから再現します。

## 作業領域D：データベースと提供実装の変更管理

### 目的

期待する状態、実際の状態、差分、移行、提供実装の各境界を、提供実装に固有の知識を中核へ持ち込まずに成立させます。

PostgreSQLは、詳細な検証に用いる提供実装として利用します。

異なるProviderは、既存資産との互換性と、特定Providerに依存しない境界を厳しく検証するために利用します。Oracleは第一Referenceの`master_data`が持つ既存資産としての最初の具体例であり、要件がOracleに限られるわけではありません。MySQL、SQL Server等を加えるかは、具体的な既存資産や利用条件が現れた時点で決め、Oracleと同じEvidence契約（物理事実の観測、read-only境界、driver assembly、actual read）で検証します。

Oracleの実環境での検証が未完了であることを理由に、判定段階Bを未達成の状態へ戻しません。

## 作業領域E：設計の実行と検証

### 目的

画面を実装する前に設計内容を動かし、アプリケーションの意味を検証できる実行・検証環境を維持します。

### 重点

- browse / record / reference / header-detail / command
- Validation
- Authentication / Authorization
- Capability invocation
- Business Result / Execution Error
- separate Observation read
- deterministic Scenario

設計用の実行・検証環境は、本番の実行基盤を簡略化したものではありません。設計を検証するための基準です。

## 作業領域F：実行先に依存しない表示

### 目的

正本となる同じアプリケーションを複数の実行先へ展開し、共通する意味と実行先ごとの違いを検証根拠で区別します。

### 最初の実行先

最初のWeb向け実行先は、PAS-04の初めての実証対象です。

Gate C EvidenceではHandover baseline `R`とWeb build `T1`を分けて追跡します。

### 二つ目の実行先

判定段階Cの後に、`ISSUE-000026`で二つ目の実行先を選びます。

比較軸:

- Canonical Application meaning
- Target Capability
- Runtime Binding
- Execution / Trust Profile
- Target固有実装
- Host control plane
- Observation
- Semantic / Functional / Visual Evidence
- RQ-024 performance measurement

最初の実行先だけを根拠に、実行先に依存しない分類体系が完成したとは判断しません。

### 編集環境の実行基盤とアプリケーションの実行先を分ける

編集環境が開発者用デスクトップ基盤としてWailsを利用しても、それを二つ目のアプリケーション実行先の検証根拠には数えません。

Wailsを使って生成したデスクトップアプリケーションを採用する場合は、別の実行先として、意味・機能・表示に関する検証根拠を取得します。

## 作業領域G：編集環境と開発体験

### 目的

厳格な意味モデルを、開発者が多くの情報を安全に扱える編集環境へ接続します。

### 原則

編集環境は意味の正本を管理する主体ではありません。

```text
Studio / file-native / CLI / AI
        ↓ candidate
Shared Acceptance Boundary
        ↓
authoritative source
        ↓ reload
Canonical meaning
```

判定段階Cの後の最初の実証は`ISSUE-000046`で行います。

最小限の操作の流れは次のとおりです。

```text
Select
  ↓
Propose
  ↓
Preview Semantic Diff / impact
  ↓
Accept / Reject
  ↓ Accept only
Shared Acceptance Boundary
  ↓
reload / Conformance
```

人が記述する安定識別子を必須にはせず、編集環境専用の受理機構も導入しません。

## 作業領域H：AIによる開発支援

### 目的

AIを大量の成果物を生成するだけの仕組みとせず、既存知識の探索、変更候補の作成、検査、意思決定の支援に利用します。

### 二段階

1. `ISSUE-000047`：人とAIが共通に使える、既存知識を優先する意味情報の探索（PAS-06）
2. `ISSUE-000027`：AIによる変更提案と共通の受理境界（PAS-01、PAS-06）

AIによる提案は、意味の正本を変更する権限そのものではありません。

検索順位、類似度、モデルの確信度も、意味の正しさを示す検証根拠ではありません。

AIの自律性と実行権限を分け、AIが道具を直接呼び出したことを機能や利用許可の仕組みの代わりにはしません。

## 作業領域I：周辺技術との連携と本番運用の境界

### 目的

中核と外部の実行基盤を分ける境界を、実際の利用に耐えるよう段階的に拡充します。

### 対象

- Plugin distribution / trust
- Plugin-defined Capability Contract（ISSUE-000080。Core固定enumを増やさずauthoring / build / runtime scopeを分離）
- production Provider adapters
- durable store / HA
- observability
- deployment integrations
- supply-chain Evidence
- compatibility policy

外部のオーケストレーターやエージェント基盤が提供する汎用機能を、TEI固有の価値として再実装しません。

## 作業領域間の依存関係

```text
A Product Baseline
      ↓
B Semantic / Knowledge ───┬── C Application / Legacy
                          ├── D Database / Provider
                          └── E Driver
                               ↓
                    F First Target / Gate C
                       ┌───────┴────────┐
                       ↓                ↓
                 G Studio          F Second Target
                       ↕                │
                 H AI / Knowledge ──────┘
                       ↓
                 I Production Boundary
```

作業をすべて順番に終える必要はありません。判定段階Cの後は、編集環境、AI、既存知識の探索と、二つ目の実行先の検証を並行して進められます。

## 2026年10月の技術調査を製品価値につなぐ

直近の調査結果を、新しいAI基盤や判定段階を増やす理由にはしません。「意味の正本を一つに保つ」「特定技術から独立する」「人が変更を確認できる」「AIを使っても検証結果を再現できる」という製品価値に照らし、既存の作業領域へ割り当てます。

詳細は[TECH_RESEARCH_INTEGRATION_2026-10.md](research/TECH_RESEARCH_INTEGRATION_2026-10.md)を正本とします。

知識の探索とAIの利用については、次の順序で必要な手段を段階的に増やせるか検証します。

~~~text
exact / explicit
   ↓ unresolved only
deterministic inference
   ↓ unresolved only
lightweight associative / statistical candidate generation
   ↓ unresolved only
bounded local cognition
   ↓ unresolved only
local SLM / LLM
   ↓ unresolved only
external LLM
~~~

どの段階でも、候補の品質や確信度を、意味上の適用可能性、受理、書込み権限、実行時の権限と同一視しません。候補の適格性、検証、受理、権限は、同じ入力なら同じ結果になる規則で最終的に確認します。

実行先と実行基盤については、Wails、WebView、ネイティブ実行、Wasmなどの違いを意味の正本へ持ち込みません。実行環境、接続部品、生成した実行物、実行物間の規約、実行設定の違いとして観測します。意味上は同じ操作を、異なる実行物や環境でも追跡できるか、二つ目の実行先で検証します。

これらの調査結果は、RQ-010、013、015、016、021、024、027〜030に残る不足を減らすために活用します。研究を参照しただけでは、検証の達成段階を変更しません。


### 2026-10-09の追加検証候補（評価待ち）

直近の仕様確認に関する研究、画面操作の評価、WebAssemblyの接続規約の標準化は、**新しい判定段階や独立した基盤を増やすためではなく、既存の業務上の参照事例をより厳しく検証するため**に利用する。製品価値との関係と調査根拠は、[技術調査の統合方針第9節](research/TECH_RESEARCH_INTEGRATION_2026-10.md)に記す。

- **REF-SPEC-01：仕様の不足を実装前に発見する。** 既存の受注の参照事例を使い、人が欠落、曖昧さ、矛盾、古い前提を発見できるか比較する。関連課題はISSUE-000075、#982、#998、#915、#267。最初に確かめたい価値は、生成量ではなく、意味を誤って確定させない変更の確認である。
- **REF-INTERACTION-01：画面操作を業務結果まで確認する。** 表示、操作、権限による拒否、業務上の結果、観測結果、画面の評価を分けて記録する。関連課題は#997、#1000、#1001、#259。最初のWeb向け実行先で得た判定段階Cの結果を遡って変更しない。
- **REF-PORTABLE-01：異なる実行物や環境で意味を保つ。** Goのネイティブ実行物やTinyGoのWasmなど、対応可能な方式を小さな構成で比較する。WASI 0.2／0.3、バイナリ間の規約、実行環境から取り込む機能、依存関係、権限、資源、性能を分けて記録する。関連課題は#259、#943、#973、#976、#1016。編集環境用のWailsと二つ目の業務用実行先を混同せず、HikeをGoコンパイラとして扱わない。

評価は、まず**業務変更時に役立つREF-SPEC-01**、次に**操作の成立を示すREF-INTERACTION-01**、続いて**可搬性を確かめるREF-PORTABLE-01**の順とする。ただし、これは研究候補の評価順であり、既存課題の着手承認、並行作業、保護された統合手順を変更しない。ISSUE-000075の候補Bでは業務規則の変更を受理したが、一連の処理を通した実証は未完了である。REF-SPEC-01はその受理を置き換えず、受理済みの意味と業務手順の記述の違いを読み取り専用で示す。

**判定：**文書への反映は実施済み。三つの参照実験はいずれも設計案であり、実行、利用者評価、性能計測、要求を満たしたことを示す検証根拠の取得は未実施。小さな実証で中心価値に関する不確実性を減らせない場合は、方式や作業範囲を広げる前に見送る。

## 外部の参照モデルを使った横断的な検証

小規模な参照事例だけでTEIの表現力を判断しないため、判定段階Cの後にISSUE-000078を横断的な検証として進めます。利用する外部の参照モデルについては、公開の可否と利用条件を個別に確認します。外部モデルの本体や詳細な派生資料は、この公開リポジトリには含めません。

これは新しい製品判定段階Dではなく、既存の要求をまとめて厳しく検証するためのものです。X-TEAのスキーマや実行基盤をTEIの意味を決める権限の根拠にはしません。データ、機能・操作、業務、元形式の既存資産、実行基盤と環境、表示形式を一式で移し替えても、意味を失わず説明できるか確認します。

移し替える際に見つかった不足は、業務手順、変更周期、変更の影響、実行先、意味の拡張などに関する既存課題で扱います。実行基盤や開発用の道具の機能を、中核に固定された列挙値として増やさないための境界はISSUE-000080で扱います。外部の参照モデルに存在するという理由だけで中核の基本要素を増やしません。原モデルの内容は利用条件を確かめずに公開リポジトリへ転載しません。元資料の照合値、対応付け、失われた意味や未解決の事項、TEI独自の試験データを検証根拠として保持します。

作業の目的と公開境界はISSUE-000078で管理します。

## 業務設計の実用性を基準にした範囲の再評価（2026-10-08、評価待ち）

[製品要求の整理#982](https://github.com/hat47x/tei-platform/issues/982)では、データ、機能・操作、業務手順の相互検証、早期に動かせる設計、人が理解できる表示を優先する観点から、RQ-001〜033と検討中のRQ-034について第1版で着手する範囲を見直している。**ここでは製品要求の削除や既存の優先順位の変更を確定しない。**

検討の入口は既存のISSUE-000075とする。実証Aは2026-10-05に合成データを用いた開発環境で実行記録を取得した。候補Bでは2026-10-08に管理主体が業務規則の変更を受理したが、Web画面を含む一連の処理、すべての明細への規則適用、三つの視点での意味の整合性確認は完了していない。実証Bの詳細と未完了範囲はISSUE-000075を参照する。

汎用的なプラグインの依存解決、構築器の自己構築、AIによる業務実行、複数の実行先などの長期的な能力は否定しない。ただし、それらの完成を最小限の業務実証の前提とする場合は、**どの具体的な不足を解消するか**を明らかにしてから採否を判断する。現在の判定段階、各作業の適格性確認、保護された統合条件は変更しない。

## 現在の優先順

1. （完了、2026-09-19）判定段階Bの最終的な実環境での検証根拠を、同じ版（R = `e5231839`）についてそろえた
2. （完了、2026-09-19）引継ぎの基準`B`と対象版`R`を固定した
3. （完了、2026-09-19）最初のWeb向け実行先をビルド版`T1`（T = `0ba87a4e`）として実装し、判定段階Cの検証根拠を得て`Ready`と判定した
4. 判定段階Cの`Ready`後に並行する作業（現在地。各課題の状態は、その課題の`状態`欄を正本とする）
   - `ISSUE-000046`：編集環境での最小限の記述・変更・受理の流れ
   - `ISSUE-000047`：既存知識を優先する意味情報の探索。厳密・決定的な方法を基本とし、必要に応じて軽量な認知機構からモデルへ段階的に進む
   - `ISSUE-000027`：AIによる提案と受理。手段によらず最小限の意味変更を共通の受理境界で扱う
   - `ISSUE-000026`：二つ目の実行先による検証。実行先の違いだけでなく、可搬な実行物と実行環境の違いも比較する
   - `ISSUE-000053`：AIが開発を再開し、業務知識を利用するための仕組み。契約、実装、設定、現在の権限、観測結果を再現可能にする
   - `ISSUE-000078`：外部の参照モデル全体を移し替え、表現力、失われる意味、暗黙の前提を横断的に検証する
5. 二つ目の実行先、定義の記述、AI、参照モデルの移行で得た検証根拠を相互に照合し、実行先からの独立性、記述方法、意味の拡張の境界を再評価する
6. 本番運用と外部技術との連携の境界を、必要な検証根拠とともに広げる

判定段階BとCの`Ready`判定や過去の検証根拠は、それぞれの対象版に固定済みです。現在は判定段階Cの後に行う定義の記述、知識の活用、AI、二つ目の実行先の作業を重視します。以前の判定対象版で得た根拠を、現在の作業領域の検証根拠として引き継ぐことはしません。

## 先に固定しないもの

- 最初のWeb向け実行先で用いる開発基盤
- 表示に関する完全な中間表現
- 実行先の機能の完全な分類体系
- 編集環境の最終的な画面構成
- AIの提供元
- ベクトルデータベース
- 変更提案の最終的なスキーマ
- 人が記述する安定識別子やUUIDの必須化
- 差分の適用だけを認める記述方式
- WailsをTEIの標準実行基盤とすること
- 二つ目の実行先の種類
- 一律の性能基準値
- 汎用の業務処理の実行機構
- 本番運用の高可用性を共通機構として扱うこと

参照事例や検証根拠によって必要性を確認する前に、これらの方式を固定しません。
