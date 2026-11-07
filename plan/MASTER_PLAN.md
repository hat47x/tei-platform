# TEIの開発全体計画

## 目的

この文書は、TEI（鼎／テイ）の製品価値と要求を開発上の作業領域へ整理し、優先する作業と並行できる作業を示します。

期限は設けません。優先順位は実装量ではなく、**TEIの中心価値に関する不確実性をどれだけ減らせるか**を基準に見直します。

過去の判断を本文に重ねて記すのではなく、現在の計画の基準を示します。以前の判断とその理由は、Gitの履歴、設計判断記録、課題からたどれます。

関連文書は次のとおりです。

- `product/`
- `requirements/`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/MILESTONES.md`
- `plan/research/TECH_RESEARCH_INTEGRATION_2026-10.md` ：直近の調査結果を製品価値や既存の課題に結び付ける統合計画
- `plan/REFERENCE_MODEL_TRANSFER_GATE_2026-10.md` ：成熟した外部の参照モデルをプラグイン中心で移し替え、TEIの表現力や暗黙の前提を横断的に検証する計画
- `plan/design/ARTIFACT_SET_SCOPE_AND_CONSUMER_READ_EVIDENCE_2026-11.md` ：導出成果物の対象集合・内部整合・公開世代・実Consumer読取を分けて検証する**提案中**の計画（#2482／#2604。製品Gateや受理条件の変更ではない）
- `plan/workstreams/TRIAD_MEANING_AND_CROSSLANE_ACCEPTANCE_HANDOFF_2026-11.md` ：Data・Function・Businessの意味を保ち、移植・DB移行・組織Relation・Plugin境界における既存Ownerの検収条件を照合する**提案中**の横断引渡し（#1125／#1126／#1129／#2619。新Gate・業務意味の採用ではない）

- `plan/design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md` ：原子的Traitの実装義務、Manual/Generated/Providerでの履行、合成・不適合診断を既存#951/#2495/#3271/#3267へ接合する**設計・実証候補**（[#3278](https://github.com/hat47x/tei-platform/issues/3278)）。新Resolver/新Hard Gateや実装完了を意味しない。
- `plan/design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md` ：Web/Desktop APIの例外・Client Reaction・Graph/Table/Traceの**提案中**の横断ドッグフーディング（[#3274](https://github.com/hat47x/tei-platform/issues/3274)）。P-1実コード照合で、#1043の**構成時Guard依存DAG**と実HTTP Decision Pathを区別し、Terminalイベント集合だけでは全経路Auditの証拠にならないGapを限定観測。HDG-21のV8 Characterization 2件PASS、実Node/Go HTTP/Graph Loader/E2Eは未実行。次はNode同一HEADと実ServerのBranch/Terminal証拠を取り、#1656/#1158/#1174/#2620/#1180/#3267のOwnerへ引き継ぐ。現行MainのGate B/Cを変更しない。

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

判定段階Cの後は、定義の記述・知識の活用と、二つ目の実行先の検証を並行して進めます。互いの完成を待つ必要はありません。最初の実行先で確認した違いと検証根拠を共有します。

## 現在の判定段階と引継ぎ

### 判定段階A：アプリケーションの意味を実行・検証環境へ渡せる

**Ready**です。

アプリケーションの構成パターン、データ接続先、検証規則、認証・認可、能力と実装の対応付け、業務上の結果と実行時エラーなどについて、画面固有の定義を必要とせず実行・検証環境へ渡せる意味の基準が成立しています。

### 判定段階B：画面なしで主要な業務シナリオを再現できる

**Ready**です（2026年9月19日の再監査。候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）。

必須の検証根拠は次の三つです。いずれも、変更のない同じ候補版`R`を対象とした一度の実行で成立しています。

1. #310 リポジトリの決定論的な再現性に関する検証根拠
2. #255 生成成果物の再現性に関する検証根拠
3. #308 実際のPostgreSQL提供実装と一時的な試験環境による検証根拠

`#387`での最終確認と、同じ実行で取得した子処理の成果物の独立監査を踏まえた`Ready`の判定記録は、`GATE_B_READINESS_AUDIT.md`を正本とします。

PostgreSQL以外の実際のデータベースによる検証根拠も重要です。最初の参照事例では、Oracleに関する#214・#371・#376が該当します。ただし、判定段階Bの第四の必須検証経路には含めません。未完了の範囲は`unresolved`として引き継ぎます。

### 引継ぎの基準：判定段階BからCへ

引継ぎは、新しい番号付きの判定段階ではありません。

判定段階Bで`Ready`となった候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`を、引継ぎ基準版`B`として固定しています。正式な記録は`GATE_B_TO_C_HANDOVER_BASELINE_B.md`にあります。

```text
Gate B Ready for R
        ↓
Handover Baseline B
  baselineRepositoryRevision = R
```

引継ぎ記録を、意味についての第二の正本にはしません。

### 判定段階C：最初の実行先で生じる違いを検証根拠で説明できる

**Ready**です（T = `0ba87a4e0a96f70a198b03ff7652fb00e3826abe`、2026年9月19日）。判定段階Bと引継ぎ基準版を起点に、最初のウェブ実行先の構築版`T1`について、意味・機能・表示の検証根拠を同じ来歴で揃えました（[判定段階Cの監査記録](gate/GATE_C_READINESS_AUDIT.md)）。この`Ready`は、本番運用が可能であることや、実際の外部環境で命令が成功したことを意味しません。実行先に依存しない表示の分類体系の完成、二つ目の実行先の不要化、表示の忠実度に対する人の承認も示しません。これらの限界は同じ監査記録に残しています。

最初のウェブ実行先では、引継ぎ基準版`B`と対象版`R`を意味の基準として使います。

ウェブ向け実装の版は`T1`として、基準版とは別に追跡します。

```text
Baseline meaning B at R
        ↓
First Web Target build T1
        ↓
Semantic / Functional / Visual Evidence
```

`T1`で基準版の対象の意味を変更する必要が生じた場合は、そのまま作業を続けず、新しい引継ぎ基準版を発行します。

判定段階Cでは、次の三種類の検証根拠を区別します。

1. 意味の準拠確認
2. 操作・機能の検証根拠
3. 表示の忠実度の検証根拠

画像を評価するAIの判断や画面画像の評価点だけで、意味や機能の検証を合格とは判定しません。

## 作業領域A：製品の基準と追跡可能性

### 目的

製品価値・要求・製品受入シナリオ・設計判断記録・計画上の課題・参照事例・検証根拠・判定段階を、一続きに追跡できる状態を維持します。

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

PASは単なる網羅率の指標ではありません。要求と実証を結び付ける追跡単位として扱います。

## 作業領域B：意味の定義・記述・知識

### 目的

定義元のソースから業務上の意味を再構成し、複数の表示形式や編集方法で同じ正本の意味を扱える基盤を整えます。

### 対象

- 意味を扱う中核
- ソース索引と診断
- 入力元から意味の正本への追跡可能性
- 分野固有の部品、意味の構成パターン、方針の候補
- 出所と対象の版
- 再利用できる意味情報の探索

第1版では、アプリケーションの構成パターンをアプリケーション側で管理します。最初の参照事例だけを根拠に、意味を扱う中核の標準的な概念一覧へ移すことはしません。

AI向けの知識・検索索引・埋め込み表現・処理履歴などは派生情報として扱い、意味の第二の正本にはしません。

判定段階Cの後に行う読取り専用の探索は、`ISSUE-000047`でPAS-06として実証します。

## 作業領域C：アプリケーションと既存資産の互換性

### 目的

新しい理想形だけを対象とせず、既存の業務資産を保持しながら段階的に移行できることを確認します。

### 対象

- アプリケーションの構成パターンと意図
- アプリケーションとデータベースの関係
- データベース契約と元形式の問い合わせ
- データ接続先の読取りと書込み
- 既存の認証と認可
- 既存のSQLと外部のAPI
- 機能と実装の対応付け
- 段階的な置き換え

実際の外部資産については、観測した事実と、正本に記述した契約を区別します。

実際に観測していない物理的な構成要素・列・SQL・データ型は、推測して補いません。

成熟した外部の参照モデルを移し替える作業は、`ISSUE-000078`と`REFERENCE_MODEL_TRANSFER_GATE_2026-10.md`で扱います。必要な業務・設計上の概念は、原則として意味の拡張やプラグインで表し、X-TEA固有のスキーマを中核へ直接追加しません。外部モデルの本体を公開の試験データへ複製せず、元資料の照合値・参照位置・対応付け・失われた意味・未解決事項・準拠確認の根拠から追跡できるようにします。

## 作業領域D：データベースと提供実装の変更管理

### 目的

期待する状態・実際の状態・差分・移行・提供実装の境界を、提供実装固有の知識を中核へ持ち込まずに成立させます。

PostgreSQLは詳細な検証に用いる提供実装として利用します。

異なる提供実装は、既存資産との互換性と、特定の提供実装に依存しない境界を検証するために利用します。Oracleは、最初の参照事例の`master_data`が利用する既存資産の一例であり、検証対象をOracleに限定するものではありません。MySQLやSQL Serverなどの追加は、具体的な資産と利用条件が明らかになってから判断します。検証では、物理的な事実の観測・読取り専用の制約・実行環境との接続・実際の読取りを区別します。

Oracleの実環境での検証が未完了であることだけを理由に、固定版での判定段階BのReadyを取り消しません。

### 汎用データソースの開発者向け管理範囲（2026-11-07、#3270）

SQL/NoSQLのデータ、ログファイル、クラウド情報源は別々の製品機能を増やすのでなく、既存Data Operations・Provider・標準データ構造と表示型から構成する。**スキーマ／テーブル／索引／制約の作成・変更・削除をTEI標準に含める**。これらはアプリケーション設計・データ管理の責務であり、DBA/インフラ運用（サーバ、クラスタ、バックアップ、レプリケーション、DBユーザー管理など）は標準外とする。実行時の対象・権限と破壊性は別途検査する。

[汎用データソース・ワークベンチの設計・縦断実証案](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)は#3270の計画として、①現状契約棚卸し・read-only構造観測、②実DBでのReverse/Schema Plan、③Disposable DBでのCREATE/ALTER/DROPと独立読取り、④ログ/NoSQLの部分取得・時系列・型揺れ、⑤CLI/MCP/Studioの共通投影、⑥組合せBOM/Cache資格の検証を段階化する。#1126・#1803・#2622・既存Schema Diff/Migration・#1641・#3267へ責務を戻し、第二のSchema/Query/Cache/Authority正本を作らない。**既存の判定段階B/Cと主線優先順位は変えず、設計・受入案と実装済みを区別する。**

**対応する製品要求は[RQ-035](../requirements/TEI_PRODUCT_REQUIREMENTS.md#rq-035-一般開発者が異種データソースのデータと構造を管理できること)**。2026-11-07に現行コードを静的確認した結果、`internal/database/provider/provider.go`のDB専用SPI、PostgreSQL `Introspect`、`internal/workbench/database/model.go`の派生投影、`cmd/tei/main.go`のSchema Diff/Migration入口は存在する。一方、異種Sourceを含む実Provider Read→完全性付きFunction Data Context→標準表現型の一往復は未確認であり、#2586と最初の不足一辺を検証する。Schema Historyのunknown→applied誤分類候補は既存#2892へ渡す。Trait・Decision Graph・C0/C1・独立期待値への接続とWB-15〜25は上記設計書の11〜15節に集約し、ここへ第二の契約正本を作らない。静的確認は実Provider/Go/GUIの実行成功を意味しない。**



### Core外PluginによるWorkbenchの同一契約実証（2026-11-07、#3270×#1251）

[#3277](https://github.com/hat47x/tei-platform/pull/3277)の[Plugin-first CLI/Studio/Runtime設計](design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md) D1/D2/D4/D5とWorkbench W0〜W5を**依存関係で接続**する。最初にCLI/Studio共通のInspector一操作を同一Contract/Providerで通し、次に**PostgreSQL実ReadとJSONLファイルRead→Function Data Context→Table/Tree/Timeline**を、同じContract選択と異質なProviderで比較する。成果物を読む`workbench-model`と実Provider Row Readは別資格。次に既存Schema Diff/Migrationの隔離DB適用、最後に同じRead/Validator契約を開発用Surfaceなしの生成Goバイナリへ組み込む。

[設計書の§16〜18](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)に実ソースの接続点、Selected/Linked/Invoked、異Alias競合、CLIの呼出別Effect表示、WB-26〜37の独立期待値と停止線を配置し、[合成SQL＋JSONLのFixture](../examples/data-source-workbench/README.md)を受入入力に固定する。Provider/Plugin/Contract Resolverの実装は#951/#1251、実Row→Data Contextは#2586、異DB Aliasは#2611、CLI Effect語彙は#3182、Migration Historyは#2892、検証済みBOMは#3267がOwner。新しいPlugin Registry、Host、Query/Cache/Authority正本を作らない。**この段階では静的コード確認＋Fixture作成のみで、実Plugin選択/Build/Provider接続/Studio/Goテストは未実施**。


## 作業領域E：設計の実行と検証

### 目的

画面を実装する前に設計内容を実行し、アプリケーションの意味を検証できる環境を維持します。

### 重点

- 一覧表示・記録・参照・見出しと明細・操作の構成パターン
- 検証
- 認証と認可
- 機能の呼出し
- 業務上の結果と実行時エラー
- 独立した状態の観測
- 再現可能で決定論的なシナリオ

設計用の実行・検証環境は、本番の実行基盤を単に簡略化したものではありません。設計の妥当性を検証するための基準です。

## 作業領域F：実行先に依存しない表示

### 目的

同じ正本上のアプリケーションを複数の実行先へ展開し、共通する意味と実行先固有の違いを検証根拠で区別します。

### 最初の実行先

最初のウェブ実行先は、PAS-04で初めて実証した対象です。

判定段階Cの検証では、引継ぎ基準版`R`とウェブ実行先の構築版`T1`を区別して追跡します。

### 二つ目の実行先

判定段階Cの後、`ISSUE-000026`で二つ目の実行先を選定します。

比較する観点は次のとおりです。

- アプリケーションの正本上の意味
- 実行先の能力
- 実行時の対応付け
- 実行条件と信頼の境界
- 実行先固有の実装
- 実行環境の管理機構
- 状態の観測
- 意味・機能・表示の検証根拠
- RQ-024に関する性能測定

最初の実行先だけの検証結果から、実行先に依存しない分類体系が完成したとは判断しません。

### 編集環境の実行基盤とアプリケーションの実行先を分ける

編集環境が開発者向けのデスクトップ基盤としてWailsを使っても、それを二つ目のアプリケーション実行先の検証根拠には数えません。

Wailsを使って生成したデスクトップアプリケーションを採用する場合は、それを独立した実行先とし、意味・機能・表示の検証根拠を取得します。

## 作業領域G：編集環境と開発体験

### 目的

厳格な意味モデルを、開発者が多くの情報を安全に扱える編集環境へつなぎます。

### 原則

編集環境そのものを、意味の正本を管理する主体とはしません。

```text
Studio / file-native / CLI / AI
        ↓ candidate
Shared Acceptance Boundary
        ↓
authoritative source
        ↓ reload
Canonical meaning
```

判定段階Cの後の最初の実証は、`ISSUE-000046`で行います。

最小限の操作手順は次のとおりです。

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

人が記述する安定識別子は必須とせず、編集環境専用の受理機構も導入しません。

### トポロジーと複合図表の評価順（未採用の追加検証候補）

[トポロジー・複合図表・開発パイプラインの共通表現](design/TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)を、既存契約を重複実装しないための横断的な設計・検証候補として参照します。現行のIssue優先順位、製品受入条件、実装済み機能の説明は変更しません。

最初は既存の型付き参照・Graph・Projection・Build/Compositionを棚卸しし、同じ構造からGraph・表・関係表を読み取り専用で投影します。次に、対象版とscopeを固定した実効パイプラインの解決完全性を評価し、完全性が証明できない場合に図表上の欠落を「不存在」と誤認しないことを検証します。編集・受理、認知負荷の比較、メタ生成の導入は、その後に具体的な不足と効果が観測された場合だけ進めます。

図表間リンクは、同一qualified要素の複数表示、別要素間の意味的な関係、表示用のナビゲーションを区別します。複合図表の関連グラフ・対応表も第二の意味正本としません。これらは採用済みの新Core SchemaやStudio編集APIを要求するものではありません。

### 三要素MethodのScript-first圧力試験（2026-11-06、評価待ち）

[三要素MethodのTEA-V0 / Self-build Fit/Gap](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)を、作業領域Gの**最小縦断実証の追加候補**として位置づける。Data/Function/Businessは同じauthoritative Script/標準データのRole別Projectionであり、専用図形を第二正本としない。既存#1124 M0〜M4と#1125/#1126/#951/#267がOwnerであり、新しい横断Gate・Resolver・Acceptanceは作らない。

最優先の不足は、①Source byteと意味が同一でもSource移動により旧定義ジャンプが失効すること、②可逆でないER/DFD線編集をSourceへ誤適用しないことを、**同一の実Consumerにおける正負の試験**で証明する点にある。並行するread-only評価では、**T06（三視点二世界の開示非干渉）・T07（同一Sourceへの複数Roleでのunknown保持）**を先に反証し、Policy/Role ResolverをTEA独自に実装しない。その後にバッチ/ファイル連携と業務図の役割区別、Method Pluginの実Build/Loadを扱う。Plugin self-buildのSB-1/2/3は既存Ownerの独立資格であり、TEAの初期read-only利用のブロッカーにはしない。 [T06/T07の限定Reference（Draft #2972）](https://github.com/hat47x/tei-platform/pull/2972)では、事前公開資格済みPublic Sliceの投影・unknown Roleの局所性に加え、Private metadata混入拒否をGitHub保存ファイルと同一BlobのNode 9件で確認。ただし実Source→公開SliceのPolicy非干渉／実Workbenchは未実証。#1124の実CLI→同一Consumer接合を直接優先とし、この安全性の二世界比較を#2516/#2620と並行Pressureに置く。現行の全体優先順・Gate B/C判定は変更せず、具体的な着手・採否は担当Issueの判断へ返す。

**Trait Intentと実装の齟齬の三視点照合（評価待ち、2026-11-06）**：横断Trait設計の[Draft #3027](https://github.com/hat47x/tei-platform/pull/3027)を唯一のTrait Definition/Intent/Observation/Assessment Ownerとし、三要素Methodは[設計§18](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)のT10〜T13を**読み取り専用の横断受入候補**として参照する。最小の`sales.order.status`では、Businessの閉値域Intent、DataのJava EnumとDB制約、Functionの書込みValidator/未確認Writerを**同じ意味対象・Scope・版・公開資格**へ接続し、制約保証の欠落を実データ違反と誤同一視しない。MECEの母集団/排他/網羅も別Claims。原典未確定の「強属性」を標準Facetとして採用せず、業務Identity・CandidateKey・不変性は独立して審査する。既存の#1124・#1126・#2516・#2620の受入を上書きせず、Studio M0→M1の直接P0や既存Gate B/C優先順を変更しない。T10〜T13の実Reader／Workbench／Policy試験は**NOT RUN**。

## 作業領域H：AIによる開発支援

### 目的

AIを大量の成果物を生成するだけの仕組みにはしません。既存知識の探索・変更案の作成・検査・意思決定の支援に利用します。

### 二段階

1. `ISSUE-000047`：人とAIが共通に利用できる、既存知識を優先した意味情報の探索（PAS-06）
2. `ISSUE-000027`：AIによる変更案と共通の受理の境界（PAS-01・PAS-06）

AIによる提案には、意味の正本を変更する権限はありません。

検索順位・類似度・モデルの確信度も、意味の正しさを示す検証根拠ではありません。

AIの自律的な作業と実行権限を区別します。AIがツールを直接呼び出せることを、能力や利用許可の仕組みの代わりにはしません。

## 作業領域I：周辺技術との連携と本番運用の境界

### 目的

中核と外部の実行基盤を分ける境界を、実際の利用に耐えられるよう段階的に整備します。

### 対象

- プラグインの配布方法と信頼性
- プラグインが定める機能契約（ISSUE-000080。中核に固定の列挙値を増やさず、記述時、構築時、実行時の範囲を分離する）
- 本番向け提供実装の接続部品
- 永続的な保存機構と高可用性
- 実行状態の観測
- 配備機構との連携
- ソフトウェアの供給経路に関する検証根拠
- 互換性の方針

外部のオーケストレーターやエージェント基盤にある汎用機能を、TEI独自の価値として重複実装することはしません。

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

すべての作業を順番に終える必要はありません。判定段階Cの後は、編集環境・AI・既存知識の探索と、二つ目の実行先の検証を並行して進められます。

## 2026年10月の技術調査を製品価値につなぐ

直近の調査結果を、新しいAI基盤や判定段階を増やす理由にはしません。「意味の正本を一つに保つ」「特定技術に依存しない」「人が変更を確認できる」「AIを使っても検証結果を再現できる」という製品価値に照らして、既存の作業領域へ割り当てます。

詳細は[TECH_RESEARCH_INTEGRATION_2026-10.md](research/TECH_RESEARCH_INTEGRATION_2026-10.md)を正本とします。

知識の探索とAIの利用については、次の順序で必要な手段を段階的に追加できるか検証します。

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

どの段階でも、候補の品質や確信度を、意味上の適用可能性・受理・書込み権限・実行権限と同一視しません。候補の適格性・検証・受理・権限は、同じ入力から同じ結果を得られる規則で最終確認します。

実行先と実行基盤については、Wails・WebView・ネイティブ実行・Wasmなどの違いを意味の正本へ持ち込みません。実行環境・接続部品・生成した実行物・実行物間の規約・実行設定の違いとして観測します。意味上は同じ操作を、異なる実行物や環境でも追跡できるかを、二つ目の実行先で検証します。

これらの調査結果は、RQ-010・013・015・016・021・024・027〜030に残る不足を減らすために活用します。研究を参照しただけで検証の達成段階を変更することはしません。


### 2026-10-20の国内ローコード調査の反映（追加検証案）

国内製品の公開文書には、設計情報の資産化、生成と手書き処理の分離、AIの編集範囲の限定が既に説明されています。TEIでは、それらを掲げるだけでなく、業務規則・標準部品・実現先の変更を通して案件の知識と処理を保ち、調査・修正・再試験を減らせるかを確かめます。調査資料の読解を、TEIや他製品の実動作の証拠とは扱いません。

| 依存の順序 | 確かめること | 作業領域・既存の入口 | 次へ渡すもの |
| --- | --- | --- | --- |
| 1 | 派生名の衝突、意味の未記入・曖昧な対応を読取り専用で示す | B・D、#2094・#2095・#1120・#1808 | 対象の識別子、根拠、規則の版、未評価の範囲 |
| 2 | 再生成で案件実装を保ち、標準定義更新時の案件修正を比較する | C・G、#2090・#2091・#2016・#1920 | 保全対象、互換性、衝突、取消し可能な範囲 |
| 3 | AI編集の対象と操作を検査し、開き直しても意味を保つ | G・H、#2092・#267・#1935 | 基準の版、差分全体、必要な検証、受理に残る条件 |
| 4 | 能力不足を配備前に示し、AI向けの参照経路を既存の認可と比較する | D・E・F・I、#2096・#2093・#1180・#1174 | 不足と未確認の区別、限定した公開範囲、同等性の比較結果 |
| 5 | 同じ受注変更を更新と引継ぎまで比較する | A・E、#2018 | 人の調査・再適用・再試験・維持の負担と不確かさ |

要求の具体化は[製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md)、担当境界と受入候補は[変更の継続性設計](design/CHANGE_CONTINUITY_DESIGN_2026-10.md)、測定は[生産性ベンチマーク](design/PRODUCTIVITY_BENCHMARK_PROTOCOL_2026-10.md)へ集約します。新しい汎用の受理口や上書き演算は先に作りません。

この表は追加検証の依存順です。現在の主線と優先順を変更せず、既存のOwner Action、Validationの依存、拒否の分類、Hostの権限が未決の場合は、その担当の判断を待つ条件として扱います。参照専用の試作、合成試験、実Hostでの成立は別々に記録し、調査を理由にGateや実装状況を格上げしません。

2026年10月21日には担当参照を#2090〜#2096へ更新しました。10月23日には、mainに統合された生成器契約の設計候補と適用範囲をそろえました。契約案との役割分担、失効時の証拠の扱い、再利用できる参照実装と追加検証は、[継続性設計の4.6節・5.1節](design/CHANGE_CONTINUITY_DESIGN_2026-10.md)で確認します。生産性記録ツールの同梱計画は初回4タスクで、計画の要素数も4件に固定されています。追加の更新タスクを同ツールの対応済みとして計画せず、実施時は対象と記録方式を別に固定します。

### 2026-10-09の追加検証候補（評価待ち）

直近の仕様確認の研究、画面操作の評価、WebAssemblyの接続規約の標準化は、**新しい判定段階や独立した基盤を増やすためではなく、既存の業務上の参照事例をより厳しく検証するため**に利用します。製品価値との関係と調査根拠は、[技術調査の統合方針第9節](research/TECH_RESEARCH_INTEGRATION_2026-10.md)に記します。

- **REF-SPEC-01：仕様の不足を実装前に発見する。** 既存の受注の参照事例を使い、人が欠落・曖昧さ・矛盾・古い前提を発見できるか比較します。関連課題はISSUE-000075・#982・#998・#915・#267です。最初に検証する価値は生成量ではなく、意味を誤って確定させないための変更の確認です。
- **REF-INTERACTION-01：画面操作を業務結果まで確認する。** 表示・操作・権限による拒否・業務上の結果・観測結果・画面の評価を分けて記録します。関連課題は#997・#1000・#1001・#259です。最初のウェブ実行先で得た判定段階Cの結果を遡って変更することはしません。
- **REF-PORTABLE-01：異なる実行物や環境で意味を保つ。** Goのネイティブ実行物やTinyGoのWasmなど、対応可能な方式を小規模な構成で比較します。WASI 0.2／0.3・バイナリ間の規約・実行環境から取り込む機能・依存関係・権限・資源・性能を分けて記録します。関連課題は#259・#943・#973・#976・#1016です。編集環境用のWailsと二つ目の業務用実行先を混同せず、HikeをGoコンパイラとして扱いません。

評価は、まず**業務変更時に役立つREF-SPEC-01**、次に**操作の成立を示すREF-INTERACTION-01**、続いて**可搬性を確かめるREF-PORTABLE-01**の順とします。ただし、これは研究候補の評価順であり、既存課題への着手承認・並行作業・保護された統合手順は変更しません。ISSUE-000075の候補Bでは業務規則の変更が受理されましたが、一連の処理を通した実証は未完了です。REF-SPEC-01はその受理を置き換えず、受理済みの意味と業務手順の記述の違いを読取り専用で示します。

**判定：** 文書への反映は実施済みです。三つの参照実験はいずれも設計案であり、実行・利用者評価・性能測定・要求充足を示す検証根拠の取得は未実施です。小規模な実証で中心価値に関する不確実性を減らせない場合は、方式や作業範囲を広げる前に見送ります。

## 外部の参照モデルを使った横断的な検証

小規模な参照事例だけでTEIの表現力を判断しないため、判定段階Cの後にISSUE-000078を横断的な検証として進めます。利用する外部参照モデルについては、公開の可否と利用条件を個別に確認します。外部モデルの本体や詳細な派生資料を、この公開リポジトリへ含めることはしません。

これは新たな製品判定段階Dではなく、既存の要求をまとめて厳しく検証する取組みです。X-TEAのスキーマや実行基盤に、TEIの意味を決める権限を与えるものではありません。データ・機能と操作・業務・元形式の既存資産・実行基盤と環境・表示形式を一式で移し替えても、意味を失わず説明できるか確認します。

移行時に見つかった不足は、業務手順・変更周期・変更の影響・実行先・意味の拡張などに関する既存課題で扱います。実行基盤や開発用ツールの機能を中核に固定された列挙値として増やさないための境界は、ISSUE-000080で扱います。外部の参照モデルに存在するという理由だけで、中核の基本要素を増やしません。利用条件を確認せずに原モデルを公開リポジトリへ転載することもありません。元資料の照合値・対応付け・失われた意味・未解決事項・TEI独自の試験データを検証根拠として保持します。

作業の目的と公開境界はISSUE-000078で管理します。

## 業務設計の実用性を基準にした範囲の再評価（2026-10-08、評価待ち）

[製品要求の整理#982](https://github.com/hat47x/tei-platform/issues/982)では、データ・機能と操作・業務手順の相互検証、早い段階で実行できる設計、人が理解できる表示を重視し、RQ-001〜033と検討中のRQ-034について第1版で着手する範囲を見直しています。**ここでは製品要求の削除や既存の優先順位の変更を確定しません。**

検討の入口は既存のISSUE-000075とします。実証Aでは、2026年10月5日に合成データを使った開発環境で実行記録を取得しました。候補Bでは2026年10月8日に管理主体が業務規則の変更を受理しましたが、ウェブ画面を含む一連の処理・すべての明細への規則適用・三つの視点からの意味の整合性確認は未完了です。実証Bの詳細と未完了範囲はISSUE-000075を参照します。

汎用的なプラグインの依存解決・構築器の自己構築・AIによる業務実行・複数の実行先など、長期的な能力を否定するものではありません。ただし、それらの完成を最小限の業務実証の前提にする場合は、**どの具体的な不足を解消するか**を明らかにしたうえで採否を判断します。現在の判定段階・各作業の適格性確認・保護された統合条件は変更しません。

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

判定段階BとCの`Ready`判定と過去の検証根拠は、それぞれの対象版に固定されています。現在は、判定段階Cの後に進める定義の記述・知識の活用・AI・二つ目の実行先の作業を重視します。以前の判定対象版で得た根拠を、現在の作業に対する検証根拠として無条件に引き継ぐことはしません。

## 2026-11-01 深掘り設計の位置づけ（評価待ち）

[Issueの深掘り対象の選定](workstreams/ISSUE_DEEPENING_2026-11.md)で選んだ5件と、その上流への接続（[深掘り設計の上流接続](workstreams/UPSTREAM_CONNECTION_2026-11.md)）を、既存の作業領域へ割り当てます。**優先順位と判定段階の条件は変更しません。**

| 設計 | 主な作業領域 | #982 の一往復での位置 |
| --- | --- | --- |
| [#1127](https://github.com/hat47x/tei-platform/issues/1127) 標準CRUD | E（設計の実行と検証）、F（実行先に依存しない表示） | 標準Function / UIの即時実行 |
| [#1129](https://github.com/hat47x/tei-platform/issues/1129) 効果規則 | B（意味の定義・記述）、E | 整合の確認（副作用と所有者） |
| [#1130](https://github.com/hat47x/tei-platform/issues/1130) 実行計画 | I（本番運用の境界）、E | 帳票・非画面業務 |
| [#2144](https://github.com/hat47x/tei-platform/issues/2144) 期間の状態 | B、I | 締めと訂正（追補候補） |
| [#1803](https://github.com/hat47x/tei-platform/issues/1803) 設計成果物 | B、G（編集環境と開発体験） | 不整合の発見と、意味の差分の確認 |

この割当は、5件の設計が提案段階であることを変えません。5件の実装が進んでも、判定段階B・Cの合格条件と、`docs/status.md`の実装状況は別に更新します。

上流に足りない決定は、[深掘り設計の上流接続](workstreams/UPSTREAM_CONNECTION_2026-11.md) 6章に置きました。マイルストーンの到達条件、#982 の一往復の担当、#1119 の契約枠への追加は、未決の判断として残ります。PoCの作業は、既定案を選んで進め、後から覆せる形で実装と検証を続けます。既定案を選んだ理由と覆す条件は、関連するIssue・PRへ記録します。

<!-- tei-data-form-read-handoff-20261106 -->
## 2026-11-06 データ様式と実Readの横断引渡し（採用前・既存Ownerに帰属）

データ様式の詳細な構造保存・変換・反例は[既存#2497の設計Draft #2496](https://github.com/hat47x/tei-platform/pull/2496)、組織Relationの業務意味は[#2389](https://github.com/hat47x/tei-platform/pull/2389)、参照モデル移植のRead-first検収は[#2783](https://github.com/hat47x/tei-platform/pull/2783)に帰属させる。本節は**最新main上の既存開発順へ渡す最小契約**のみであり、第二の構造正本、Query/Role Resolver、Host、Evidence Registry、Scenario/Gateを作らない。

**関係のIdentityと編集対象を構造ごとに区別する**：

| 元データの様式 | 保存すべき対象と不変条件 | 暗黙に推論してはいけないこと |
| --- | --- | --- |
| **Direct Relation**（#2681のNative `CaseRecord.handlingUnit`） | 所有Case ID、Relation Role、現在値、対象Source/Context。更新Intentには実Providerに束縛された対象行と必要なExpected Version | Graph描画線・JOIN行から独立Edge IDを発明しない。同NativeのCaseRecordには`rowVersion`属性がなく、合成Provider例の7→8は正式Field/CASの証拠ではない |
| **Materialized Relation**（`Membership`・`Occupancy`） | Membership IDなど独立した関係DataのIdentity、多重Occurrence、Relation向き・Field・粒度。後段の集計/編集が使うものを落とさない | 同じPerson/Unit/Labelの辺を勝手に一つへ統合しない。`validFrom`/`knownFrom`の属性宣言だけから二時点Query・Provider pushdown・時点Owner受理を推論しない |
| **Derived Relation/View**（Graph/Tree/SQL JOIN/集計/SHO） | 元SubjectのIdentifier/Occurrence（存在する場合）、Mapping、Scope/Grain、Source occurrence/Revision、Lossと用途相対Claim | 表示線・Row index・派生Count/0件だけから書戻しTarget、全件不存在、現在Grantを決めない。元への可逆写像が未成立ならread-only/明示Action Intentに留める |

**最短の製品受入順**：[#2586](https://github.com/hat47x/tei-platform/issues/2586)の既存同一Attemptを唯一の実Read統合Ownerとし、(R0)独立Expectedと負対照・Source/Provider Targetを固定 → **(R1) 1 Function・1 Query・1 Provider**の実Loaded/解析/Consumed Source、Function Data Contextの明示Relation path/Field/Grain、実Provider Rowset＋Field状態・Scope/Read Cohortを対照 → (R1v)同Queryへの二つの**明示的**SQL Realization切替・負の選択依存を対照 → (R1p)Page1/2継続の同一Realization/Cohort/完了根拠を検査 → Consumer開示やWrite/Effectは別資格、という順番で進める。

最初のR1は[#2783](https://github.com/hat47x/tei-platform/pull/2783)の合成受注（Tenant A/Order1001、子2件、他Tenantの同業務番号を負対照）を主線とする。**組織Case E2a**（Case C1→OrgUnit D1、C2→D2を負対照）は[#2681](https://github.com/hat47x/tei-platform/pull/2681)の別Subject/別Expected/別Attemptとして並行可能。**組織Membership E2b**のvalid/known・多重出現は後続。これらを一つの実行で成功したかのように合算しない。mainの`examples/xtea-lowcode`は`UNBOUND_DATASOURCE`を明示し、org NativeのCaseRecord Persistence/Provider Readも未接続であるため、R1実Readを成功済みとしない。`tei dbcheck`の接続とTEMP TABLE確認、Parser/OperationSurface/合成Scenario/旧Go CI PASSをR1物理Readの代替にしない。

**判定方法**：対象Claimが必要とする`Required（独立検証基準/業務意味）→ Declared → Resolved/Actually Consumed → Observed（Rowset/Field/Scope/Cohort）→ Current Consumer Qualification`を既存#1163/#2514/#1180の一試行証拠で対応付ける。Declaredが空でもRequiredが不明ならempty PASSとしない。部分/未取得/NULL/非開示/完全な0件、Entity Count/Relation occurrence Count/Join増幅、過去のEvidence/現在Grantはそれぞれ分ける。Product Readerの呼出し辺が見つからなければ`product-read-not-connected`、Case Providerが未接続なら`physical-read-unavailable`と**具体的に不足する辺を記録**し、合成テストの成功で補わない。

<!-- tei-data-form-evidence-absence-outcome-separation-20261107 -->
**未観測・不在・0件・失敗を混ぜない横断負例（既存#2586/#2514の判定票へ付加）**：

| 実際の観測 | 独立検証基準で許される結論 | 禁止推論／実装の所在 |
| --- | --- | --- |
| `*.tei.md`がManagedFilesに存在するだけ／#3031 Guardが`TEI_MD_PARSER_UNBOUND` | `not-loaded/unsupported`、必要Sourceの未充足。Raw Draftは保存対象になり得る | `validate ok`、Catalog空、`proposals=[]`からMeaning 0件・正常終了を推定しない（#3031 Draft、#2823） |
| `model catalog -namespace no.such.ns` | `NAMESPACE_NOT_FOUND`で`ok:false`、終了コード1。既存の宣言済み空Namespaceとは区別 | main統合[#3052](https://github.com/hat47x/tei-platform/pull/3052)の限定CLI変更。全Readerの必須Source完結や`v0`互換の最終判断までは示さない |
| 必要対象を一件も取得していない／一部のみ観測 | 集合の空・全件完了とも限らず`unverified` | [main統合済み #3051](https://github.com/hat47x/tei-platform/pull/3051) exact HEAD `581e699b62b126c86388c88c7c357674394ad252` の[限定Go/Health CI](https://github.com/hat47x/tei-platform/actions/runs/42256518693)はsuccess。ただしCallerの`CoverageComplete`自己申告を独立に裏付けるものではない |
| 対象内で`Holds:false`を一件観測 | 被覆が部分でも当該条件の`fails`証人になり得る | これを「全Targetの取得完了」や他Claimの普遍否定へ拡張しない（#3051の同一Go試験） |
| 実Rowsetが0行で返る | Selection/Scope/Cohort/同一Snapshot／Page終了条件が完全と実証された場合に限り対象の空を候補化 | 実Providerを呼べない`UNBOUND_DATASOURCE`、別Tenant/Replica・途中Page・JOIN増幅から完全な不存在を結論しない（#2586/#2611） |
| 参照Commandが`committed:true`を返す | 参照UnitOfWorkのCommit呼出しは完了し得る | main統合[#3054](https://github.com/hat47x/tei-platform/pull/3054)は`persisted:false`/`durability:none`を別表示。実DB永続化・独立再Readではない。main統合[#3053](https://github.com/hat47x/tei-platform/pull/3053)の`COMMAND_EFFECT_UNKNOWN`と再送拒否、[#3060](https://github.com/hat47x/tei-platform/pull/3060)の未解決`Differences`は**ReadのE2a合格とは別の効果境界** |

**受入の最小順**：#2586では①Requiredの実Target/Field/Scopeと独立期待値を先に固定、②宣言したPattern/Query/DB Aliasと実選択・実Provider消費の辺を対照、③行・列・集合・Page/Cohortの完全性を観測、④対象Claimだけを`holds/fails/unverified`へ判定する。Command/Receipt/Current Grantは同一Read成功から自動で継承せず、既存#1174/#2516へ分離する。上表は新Public Status Enum・新Test Runner・新Hostを提案せず、既存CLI/Completion/Source/Effectの機能差を同じ独立Expectedで比較する**計画**。実Function→Provider Product R1と`*.tei.md` Reader E2Eは今回**NOT RUN**。

<!-- tei-data-form-selected-alias-write-preflight-versus-runtime-effect-20261107 -->
### 2026-11-07 多重DB Aliasの書込前検査と実Provider対象を切り分ける（#3070／#2611、受入案）

main統合済み[#3070](https://github.com/hat47x/tei-platform/pull/3070)の`internal/command/datasource_access.go`は、DataEffectがcreate/update/deleteの場合にApplicationの`database <alias> {target ... access read}`とDBの`SemanticUses`を照合し、最初に一致したDB要件がread-onlyなら`COMMAND_DATASOURCE_READ_ONLY`で**Executorより前**に拒否する。単一Aliasの拒否／read-write／非該当宣言の局所Go試験はPRで報告済み。ただし**実DBへの接続、物理Provider権限、現在Host Grant、実Read/Write Effectの同一性は観測していない**。該当DB宣言を見つけられずreturn nilする経路は、書込許可の証拠でなく、宣言照合の**非拒否**にすぎない。

**新たなSource-staticの曖昧性**：`checkDatasourceAccess`は`app.Databases`を先頭から走査し、`databaseUsesNamespace`が真の**最初の一件**で拒否または`return nil`する。同じ`sales.order`を参照するRead Replica（`archive:access read`）とWrite Primary（`primary:access read-write`）が併存し、**既存World契約がこの構成を許容する場合**、宣言順を`[archive,primary]`から`[primary,archive]`に反転すると、実Selected Aliasを確認せずにPreflightの拒否/非拒否が逆転し得る。これはGoコードからの**反例予測**であり、現実に有効なFixture/実Go試験/実Providerへの越境を確認したものではない。複数AliasそのものがWorld Validatorで拒否されるなら、第一の独立Expectedはその明示診断となる。

**重複しない検収の最小追加**：既存[#2611](https://github.com/hat47x/tei-platform/issues/2611) PM-03/04（Read Replica vs Write Primary、Alias順反転）に`checkDatasourceAccess`を含む同一World固定Fixtureを追加し、①有効なWorldか、②実際に選択されたAlias/DB/Operation/Scopeはどれか、③read-onlyの宣言で拒否したか、④非拒否ならどのHost Grant/Provider実Targetが実際に許可・強制したかを別列にする。Alias順だけで実対象・認可・業務意味を推論しない。**新Provider Resolver/Permission Engineは設けない**。

またmain統合済み[#3068](https://github.com/hat47x/tei-platform/pull/3068)は、Command Executorが適用操作を列挙しなければ`OperationRecord.AppliedUnobserved=true`とし、CLI/Studioへ`difference-check: not observed`を伝える。従って`Differences=[]`から**「適用対象と宣言に差がない」**と判断できるのは、必要なOperation/Scopeが実際に列挙・観測された場合に限る。これは参照実装の証拠品質改善であり、実SQL・値の前後・DB Commit・独立再読取は未実証。必要な取得対象を実Readerが列挙できない場合の`CoverageUnestablished`と**同じ非推論原則**を共有するが、Read R1とCommand E4の実行資格を合算しない。

**配置／証拠上限**：多重Alias/Targetの選択判定は#2611/#2611 PM-03/04、Functionの実Provider Readは#2586、Host GrantとEffectは#1174/#2516、独立検証基準は#2514に保持する。今回の検証はmainの実Source静的照合と既存PRの限定試験記録であり、**多重Alias新Go試験、実Provider/DB、Owner受理はNOT RUN**。

**Owner／状態**：実ReadとFunction Contextは#2586/#2506、Provider/DB Aliasと明示SQLは#2622/#2611、構造の保存/Source編集は#2497/#2621、組織の意味とCaseは#2389/#1126、独立Expectedは#1131/#1180/#2514、Writer/Host Effect/Receiptは#1129/#1174。追加された本節は設計引渡しであり、実TEI Read/DB/Consumer/Owner意味受理の実験は**NOT RUN**。#2496は古いDraftであり、この節を同Draft全体のmainへの自動採用とみなさない。

<!-- tei-data-form-e1b-candidate-vs-e2-physical-read-20261106 -->
### 2026-11-06 Read候補と実取得の接続不足を一段具体化（#3024／#2586、採用前）

[#2681](https://github.com/hat47x/tei-platform/pull/2681) のNative `.tei`では、`org_view`の`org_read`（`org_store`、read）とMembershipのPersistenceだけがある。元のCaseRecordはEntity/直接Relation定義のみでPersistenceがない。現mainの`internal/operations/surface.go`にある`SurfaceWorkspace→workspace.Load→DeriveOperationSurface`は、**Readable Alias＋対応Persistence＋Data Key**からread/listの**候補**を導出するread-only経路であり、業務FunctionのQueryを発行したりProvider Rowsetを受領したりする処理ではない。

[積上げDraft #3024](https://github.com/hat47x/tei-platform/pull/3024) は#2681を親とし、実Native Workspaceの一時コピーへ`persist org_store.case_record`を加え、実Loaderと`SurfaceWorkspace`がCaseRecordのlist/read候補（Read Inputは`organization.case_record.case_id`）を導出できるかを試験する。基線でMembershipの2候補、Mapping追加後は計4候補となることを、[#3024 exact HEAD `cc10afc61f` のNative Case Mapping CI #42204092007](https://github.com/hat47x/tei-platform/actions/runs/42204092007) **completed/success**（job `native-case-mapping`のSource/整形確認、Native Go特性テスト、checkout清浄確認が成功）で検証済み。**この合格はE1bの候補導出に限定**し、Provider Binding/Row/Field/Scope/Cohort/Consumer/現在Grantを証明しない。親#2681も未統合であり、#3024はmain向け単独差分ではない。

このため組織E2aの不足は2段で表示する：（1）**E1b/Candidate**：CaseRecord Mapping→実Loader/OperationSurfaceの候補と読み取り入力Identityを限定検証（#3024）；（2）**E2a/Actual Read**：明示Function Data Context→選択DB Alias/Instance→Provider/Query/Args→実Case C1/D1のRow+Field/Scope/Cohort→独立Expected/Consumer（#2586/#2506/#2622）。途中の`Binding=""`・`Effects=[]`・`read-only`は実Read/Host Grantを示さない。Owner/HostのE4 WriteやRowVersion Bindingはさらに別である。

**横断の選択基準（既存Sourceの宣言辺を除外しない）**：受注RMTの合成受容例Eは、実Sourceの`pattern order_entry.recent_orders browse`→`query order_entry.recent_orders`、`application order_entry`の`database sales {target sales_core}`、Queryの`datasource sales`と`recent_orders.sql`が**宣言済み**（[#2783 §14](https://github.com/hat47x/tei-platform/pull/2783)、[#2586引継ぎ](https://github.com/hat47x/tei-platform/issues/2586#issuecomment-6690394573)）。「Pattern→QueryやAlias→論理DBが一切未接続」とは書かない。一方、`UNBOUND_DATASOURCE`は**実RuntimeでのProvider Instanceへの束縛・実Query呼出しが未証明**であることと整合する。最初の実行時不足辺は (1) ContextのQuery選択／実Consumed、(2) Alias→物理Instance/Provider、(3)選択SQL・引数、(4)実Rowset/Scope/Consumerの順で照合する。Org Caseは#3024のE1b Mapping/候補とE1c Query/SQL Source受容までは局所Go実証済み、Function→Query選択・実Providerは未接続。両者の**最初の不足辺は同じと仮定しない**。#2586の既存一Attemptへ最初の欠落Edgeのみを渡し、二SQL選択／Page継続は製品実Read成立後に検証する。第二Reader/Host/Query EngineやRelation一律IDは加えない。

**R1引継ぎへ加える最小の反例**：#3024が一時Workspaceへ追加した`persist org_store.case_record`は、`case_id`を単一Primary Keyとする**単一Mappingの局所特性化**である。導出されたRead Input `organization.case_record.case_id` は、そのMappingでの検索候補を示すにとどまり、同じ文字列`C1`が別Tenant/DB Instanceに存在する場合のグローバル一意性・可視性・業務所有権を証明しない。最初のC1→D1実Readに対して、**選択Alias/Instance/Provider＋Tenant/Scope＋Case ID＋Relation Role＋実Row Identity**の対応を同一Attemptで保存し、別Tenant C1→D2を混入させない。必要なTenant Scope/物理Keyは業務モデルとProviderごとに決め、全DomainへTenant列や独立Edge IDを一律強制しない。#2389 §71/#2586/#2611/#1656の既存負例・Ownerへ引き渡し、第二Fixture正本を作らない。

**証拠の版**：本PR #3023 旧HEAD `43775e2daf1341243aad639ab504cef3266698e8` の[Repository Health #42203726377](https://github.com/hat47x/tei-platform/actions/runs/42203726377)は **completed/success**。#3024のPASSは別HEAD（`cc10afc61f`）・別Scopeの局所Go検証であり、同一Product Readの合格へ加算しない。本追補は計画Markdownのみ。**追補後の新HEADのCIは別途確認**。実DB/Provider/Consumer E2E、実Host Grant、Owner意味受理は今回NOT RUN。

<!-- tei-data-form-e1c-and-tei-md-migration-20261106 -->
### 2026-11-06 E1c実Query読込と `*.tei.md` 統一方針の引継ぎ（決定と実装資格を分離）

**新しい局所実証**：[積上げDraft #3024](https://github.com/hat47x/tei-platform/pull/3024) のexact HEAD `705933667d5b081036314a9161280323e08a64bd`、[Actions #42208159977](https://github.com/hat47x/tei-platform/actions/runs/42208159977)は**completed/success**。実Native `workspace.Load`を用いて、コピー先へ追加した`persist org_store.case_record`のE1bに加え、`query org_view.case_read`（input `case_id`、datasource `org_read`、Case ID/handlingUnit IDのResult Mapping）・別`case_read.sql`のSourceContributionをE1cとして読み込んだ。SQL WHERE句の本文だけを改訂した場合、同Query ID/Result MappingのままWorld FingerprintとModel Revisionが変化する局所回帰を含む。旧試験では入力名`caseId`が文法に不適合だったため`case_id`へ修正して成功。**今回証明されたのは宣言とSource版変化のLoader経路であり、Function Context→Query Binding、SQL実Provider選択・実行、Rows/Fields/Scope/Cohort/Consumerは一度も製品として検証されていない**。#2586のE2aは次の一辺を保持する。

**Owner決定（Authoringの将来形式）**：`*.tei`の新規採用を原則停止し、TEI文書を**`*.tei.md`へ統一する方向**とする。`*.tei.md`は単なる汎用`.md`との区別をファイル名で維持し、Markdown編集・表・図・説明とTEIの構造化データを共存させる入口とする。これはユーザーの形式選択であって、現行Language Plugin/Parser、`formatVersion`、既存Workspaceの実装・正式Owner Meaningや実行Grantが移行済みという意味ではない。従来[#2621](https://github.com/hat47x/tei-platform/issues/2621)の「.teiを厳密Sourceとして残す／.tei.mdを標準化しない」**未採用比較案**より、この方針を優先し、既存の説明を新しい決定と混同しない。OKF frontmatter/tagsは文書の意図を示す追加情報であり、`tags:[tei]`やファイル名から意味受理・Host権限を自動発生させない。

**現行mainの具体的移行Gap**：`internal/workspace/loader.go`は`semantic`を`strings.HasSuffix(name, ".tei")`、`applications/.../queries`・`persistence`などを`*.tei`、`application.tei`/`database.tei`/`plugin.tei`等を**固定名**で列挙し、発見した原文を`syntax.NewParser`へ直接渡す。`scanManagedFiles`の`filepath.Ext("model.tei.md")`は`.md`となる。よって**現状の`*.tei.md`はNative意味Sourceとして発見・解釈できると主張しない**。Migrationで`*.tei`の一斉rename、旧ParserへのMarkdown本文直接投入、`.tei.md`を汎用mdとして無言skip、または旧新両方を同じSemantic IDで二重採用することを禁止する。

**移行計画の担当整合（最新文書で解消）**：移行専用[Draft #3030](https://github.com/hat47x/tei-platform/pull/3030) の`TEI_MD_UNIFIED_AUTHORING_MIGRATION_2026-11.md`最新冒頭は **M0〜M6/TM-01〜14を移行の主たる実装・受入候補**とし、[横断Draft #2667](https://github.com/hat47x/tei-platform/pull/2667) の`TEI_MARKDOWN_SOURCE_MIGRATION_2026-11.md`も **M0〜M5/TMD-01〜12を先行研究・履歴・対応表**として#3030へ委譲した。以前の「双方が相手を正本と呼ぶ」矛盾は**両ブランチの現行本文上で解消**。ただし両PRはDraft・main未統合であり、個別TM/AUの正式な実装・Owner業務意味受理は別である。#3031の暫定Loader停止ガード、#2823の実Loaded Coverage、#988のGrammar/Pin、#2497の構造保存、#2586の実Function Readは引き続き各担当に帰属させる。

<!-- tei-crosslane-migration-owner-conflict-and-terminology-20261107 -->
**実装された暫定停止線（別Draftの限定CI）**：[PR #3031](https://github.com/hat47x/tei-platform/pull/3031) は、選択されたWorkspace Source rootsを走査した`ManagedFiles`に `*.tei.md` があると、未選択Parserのまま意味が空になる誤成功を防ぎ`TEI_MD_PARSER_UNBOUND`＋Source pathで拒否する暫定Guardを追加。exact HEAD `63953ccb5c12ead26ec1268e7c0afdba4eff66ae` の[専用CI #42217379093](https://github.com/hat47x/tei-platform/actions/runs/42217379093) は**completed/success**（Focused 4件/7 Domain圧力、Workspace package試験、vetの限定結果）。これは**未マージDraft上でのFail-closed**であり、現行mainの`*.tei.md`解析・Parser選択・Meaning/Build資格が成立したことではない。Source roots外の文書はこのGuardが観測しない。Manifestの誤Root/optional未探索の完全性は引き続き#2823の別Claim。

<!-- tei-markdown-parse-reject-vs-validation-proposal-20261107 -->
**暫定Load拒否と診断修正案の非同一性**：main統合済み[#3041](https://github.com/hat47x/tei-platform/pull/3041)の`validation.Propose`は診断に**World Fingerprintを結び、`Accepted=false`で提案**する限定実装（`tei validate -json`の`proposals`）。一方、#3031の`TEI_MD_PARSER_UNBOUND`はWorldの正式Parse前にLoadを拒否するため、**`proposals`が空でも「修正不要」「全必須Source読込済み」ではない**。将来、暫定Guardを新Markdown Readerへ置換するときも、Required Sourceと実Loaded/Parsed、Diagnosticと未検査Scope、版付きProposalとOwner意味受理を区別し、#3030のTM-02/12/14と#2823の既存試験へ加える。#3041の`validation.Accept`は版一致検査を提供するが、修正Patchの適用・実Provider Read・業務Ownerの意味受理・Host Grantを証明したものではない。

**M1とM2/M3の順序補正（#3030への提案）**：`*.tei.md`を**下書き文書の新規作成既定**にすることは可能だが、Markdown Reader/選択Parser/Loaderが実Sourceとして消費できるまでは、選択済み実行Workspace rootへ「実行可能な既定Source」として投入しない。Raw Draft用の未選択領域とLoaded Worldの要求集合を混同しない。もし選択済みrootに旧`.tei`と新`.tei.md`が同居するなら、現#3031暫定Guardは**同内容でも拒否**する。将来、両方が解析可能になった段階では、同Subjectの二重正本を拒否またはOwnerが一方を明示選択し、先勝ち/後勝ちしない。

**R1との最小接続**：#3024の旧形式E1b/E1cは、固定されたLegacy Workspace/Source Revisionでの`CaseRecord Mapping→case_read Query→SQL SourceContribution`までの証拠。移行試験ではTM-02/03/05/09/12/14を再利用し、(a)宣言Sourceが選択・実消費されたか、(b)旧新SourceのSemantic/Query/Result Mappingが独立Expected上で整合するか、(c)同じSQL IDでもSQL-only変更後のRevision失効が生きるか、(d)raw byte/Markdown SpanとSQL相対出典が失われないかを、**移行後の別HEAD**で検収する。旧形式E1c PASSと新形式E1c PASSは別資格。#2586の最初の実Provider Readは検証可能な旧形式の固定Workspaceでも先行できるが、その成功を新形式の移行適格性として再利用しない。

**順序の衝突回避**：Markdown容器文法とLoader移行の未完成を、Read-first P0-R1全体の停止条件にしない。実製品Readerが未接続の間は既存`.tei`でSource/Query/実Readの不足一辺を確認してよく、その証拠にSource形式の旧版と適用範囲を明記する。新形式移行後には、旧E1b/E1cのSource/Query/SQL出典と実Readの適用範囲をそれぞれ再資格化する。Authoringの検収候補は#3030/#2621/#988、Source/構造保存は#2497、実Readは#2586、独立期待値・独立検証基準は#2514に帰属させる。新全域Parser/Host/Registry/第二Query Engineを増やさない。

<!-- tei-data-form-source-read-witness-and-cli-v0-20261107 -->
### 2026-11-07 Sourceの管理・実読込み・意味適格性を現行CLIで検収する（#2823／#3073／#3078／#3081）

**mainで新たに観測できる事実**：
- [#3073](https://github.com/hat47x/tei-platform/pull/3073)は`validate -json.sourceRootFindings`に、存在しない宣言root（`severity:error`、`ok:false`）、既知一覧にないが実在するroot（`severity:warning`）、`.tei.bak`/`Application.tei`等の非解釈ファイルを区別して返す。未知rootの警告は**正常なLoaded Sourceの証拠ではなく**、警告のみなら`ok:true`を維持し得る。Containment・symlink escapeの実拒否はこの欄への警告に格下げしない。
- [#3078](https://github.com/hat47x/tei-platform/pull/3078)は`ManagedFile.readByLoader`を、`readSourceForLoad`が実読取したバイトの`sourceSnapshot`への存在に基づいて立て、`workspace-files`へ`managedFiles[]`と`sourceRootFindings[]`を返す。**`readByLoader:true`は実Readバイトの証拠だけ**で、Parsed Occurrence・Role/Meaning採用・Operation/Build実消費の証拠ではない。逆に`false`はファイル名が管理対象でも現Loaderが解釈していない状態で、宣言無しの証明ではない。
- [#3081](https://github.com/hat47x/tei-platform/pull/3081)は宣言0件に`WORKSPACE_DECLARES_NO_MODEL_ELEMENTS` warning、`declarationCounts.total=0`を追加するが、`ok:true`・exit 0を維持する。警告があっても失敗しないCLI契約と、**実FunctionのRequired Sourceが揃ったか**の判定を別に行う。

**同じ公開合成Workspaceで比較する最小試験（#3030 TM-02/03/04/12/14と#2823の既存Fixtureを利用）**：

| 組合せ | 期待される観測と資格上限 |
| --- | --- |
| (A) 既存`semantic/order.tei`＋通常`notes.md` | `order.tei`は`managed=true/readByLoader=true`、`notes.md`は`managed=true/readByLoader=false`となる既存#3078正例。Reader実Consumeは認定できるが、業務Function/Provider到達とは別 |
| (B) `order.tei`を`order.tei.md`へ移すだけ | 現**main**では`managed`でもLoaderがまだ読まず、必要Sourceは未Loaded。`sourceRootFindings`警告/宣言0件warningがあっても`ok:true`はあり得る。別の未統合[#3031](https://github.com/hat47x/tei-platform/pull/3031)適用Checkoutなら`TEI_MD_PARSER_UNBOUND`で明示拒否。二つのブランチの結果を一つの製品状態と呼ばない |
| (C) 旧新が同じSource rootに併存 | 旧SourceだけでWorldが成立し、新Markdown側のbyte改訂がSource Indexには現れても**実Worldのモデル変化がない**反例を対象にする。旧新同一Subjectの意味衝突・Owner Selectionは新Readerの独立試験として未決。宣言順やFilenameで黙って採用しない |
| (D) manifestに存在しないroot／実在する未知rootを指定 | 存在しないrootはerror。実在する未知rootはwarningだが`readByLoader=false`とRequired不足を別に確認する。warningを必要Source集合が空である証拠へ変換しない |

**外部CLI Consumerの互換性に関する既知の未決**：#3078で`tei.cli.workspace-files/v0`の`result`が**配列→`{managedFiles,sourceRootFindings}`オブジェクト**へ変わった。[#3092/#3095](https://github.com/hat47x/tei-platform/pull/3095)のDiscoveryと実応答`schemaVersion`一致試験は、この**同じv0の内側JSON型の互換性**までは示さない。旧Readerが配列として解釈し空集合へ縮約すると、Loader対象の欠落が隠れる。旧Binary×旧Consumer、新Binary×旧Consumer、新Binary×新Consumer、旧Binary×新Consumerの**既存[#2824](https://github.com/hat47x/tei-platform/issues/2824)互換判定**をOwner判断で行う。`schemaVersion`一致やIndex Fingerprint変化だけをLoader/Consumer互換のPASSとしない。

**実Readとの最短接続**：#2586では`Required Source/Query/Field/Relation/Scope → Manifest Selected → Managed → readByLoader → Parsed Source/SQL Byte・args → Functionから実選択 → Provider Row/Field/Cohort → Consumer独立検証基準`の**初めて不足する一辺**を同一Attemptで示す。E1b/E1cと本節のSource観測はR1への入力であり、**実Provider取得の代替にならない**。正式`*.tei.md` Reader、TM新形式正例、外部Consumer旧新実実行、実Product R1/Owner受理は**NOT RUN**。新CLIレジストリ、Parser、独立Fixture群を本PRで実装しない。

**本節の証拠上限**：既存`internal/workspace/loader.go`と#3024 E1c Go/CIの実照合に加え、#3031暫定Guardの固定HEAD限定CIと、#3030移行設計の実Sourceを確認済み。実`*.tei.md` Markdown Reader/意味Parser・Builder・Writer、移行後Org E1c、実Provider/DB、SHO/Consumer、Owner意味受理は**NOT RUN／未成立**。この文書は設計引継ぎであり、#3030/#3031が既に担当する詳細契約・Goコードを二重定義しない。

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

参照事例や検証根拠によって必要性を確かめるまでは、これらの方式を固定しません。
