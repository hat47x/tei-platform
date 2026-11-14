---
title: "数理・統計・計画・科学向けツールキットの設計と検証計画"
date: "2026-11-14"
status: "評価待ち（未採用）／実装前設計・受入計画"
issue: "https://github.com/hat47x/tei-platform/issues/4660"
---

# 数理・統計・計画・科学向けツールキットの設計・検証計画

## 1. 目的と結論

本計画は、業務アプリケーションの設計・生成を効率化することと、TEI Studio自身を設計・編集・分析のために活用することを、共通の設計基盤の二つの用途として扱う。Studioの設計文書を、業務システムや研究支援アプリケーションの開発にも自然に展開できるかを検証する。

Mathematica（Wolfram Language）やRが扱う数理・統計の体系、ウィンドウ集計、Solver・実験計画法（DOE）、WBS・PERT・CPM、物理システムの時間発展、生命科学の反応ネットワークや可視化を、具体的な検証対象とする。ただし、各製品の言語や独自形式をTEIの標準DSLへ移植することを目的にはしない。表現・数学的意味・実現手段・UIを分け、既存のTEI標準契約を先に使う。

**提案の核**：汎用基盤の拡張基準と、専門ツールキットの製品価値の基準を区別する。Coreには意味の共通性・長期安定性を求める。一方、専門ツールキットは汎用性を無理に要求せず、特定領域の実務を一巡できる完成度で評価する。独立した専門市場の需要検証も許容し、Studioセルフビルドの全体P0を損なわない並行レーンとして扱う。

本書は構想を受入可能な設計課題へ分解したもので、Core/DSLの新仕様、プラグインABI、数理エンジン、新しい承認機構、外部ターゲットの採用を確定しない。実装・性能・市場適合は未実証。

## 2. 既存設計との接合

| 既存の正本・Owner | 今回接合する内容 | 今回は行わないこと |
| --- | --- | --- |
| [#2619](https://github.com/hat47x/tei-platform/issues/2619)、[#988](https://github.com/hat47x/tei-platform/issues/988)、[#4359](https://github.com/hat47x/tei-platform/issues/4359) | Everything is Data、DSL／Operatorの意味契約と必要最小限の構文 | 分野別の専用DSL、別のParserや意味定義の正本 |
| [#2495](https://github.com/hat47x/tei-platform/issues/2495)、[#2544](https://github.com/hat47x/tei-platform/issues/2544)、[#2501](https://github.com/hat47x/tei-platform/issues/2501) | 数学的Carrier、演算・法則、量・単位、配列／表／グラフ／時系列のFacetと圧力事例 | 排他的な型階層や一律のCore enum追加 |
| [#2497](https://github.com/hat47x/tei-platform/issues/2497)、[#3827](https://github.com/hat47x/tei-platform/issues/3827)、[#4350](https://github.com/hat47x/tei-platform/issues/4350) | 表現型／Mapping／Projection、科学図と帳票の編集・逆写像 | 可視化配置から業務意味を推定 |
| [#951](https://github.com/hat47x/tei-platform/issues/951)、[#1016](https://github.com/hat47x/tei-platform/issues/1016)、[#1121](https://github.com/hat47x/tei-platform/issues/1121) | Operator Contract、Provider、Target Generator、Toolkit構成の接続 | 第二Plugin Manager／汎用Runtime／共通ABIの先行確定 |
| [#2387](https://github.com/hat47x/tei-platform/issues/2387)、[#2586](https://github.com/hat47x/tei-platform/issues/2586) | SQL／LINQ風Query、ウィンドウ・集計、実Data Context、Consumerへの受渡し | SQLの外部実行結果を自動的に同じ意味と認める |
| [#1824](https://github.com/hat47x/tei-platform/issues/1824)、[#2831](https://github.com/hat47x/tei-platform/issues/2831)、[#4583](https://github.com/hat47x/tei-platform/issues/4583) | Studio共通Workbenchと、設計からの実Stage2セルフビルド | 専門ツールキットをStage2達成の先行条件にする |
| [#1180](https://github.com/hat47x/tei-platform/issues/1180)、[#2514](https://github.com/hat47x/tei-platform/issues/2514)、[#2600](https://github.com/hat47x/tei-platform/issues/2600) | 独立Expected、数値比較、実性能計測、ビルド再利用資格 | 生成物同士の一致を独立検証扱いする |
| [#3791](https://github.com/hat47x/tei-platform/issues/3791)、[#3796](https://github.com/hat47x/tei-platform/issues/3796) | 異分野のPressure Corpusと製品一巡の実証 | 本書を全領域の新たな要求正本にする |

Go単一バイナリをTEI本体の標準とし、StudioではWails Native/Server＋共通Go Serviceの現行候補を維持する。R/Python、数式処理、Solver、Modelica/FMIなどは、必要な場合に既存Provider／Targetの接続先とする。生成する一般アプリケーションへWailsや特定の言語・フレームワークを強制しない。

## 3. 標準機能への採用を判断する手順

適用対象ごとに、次の順序で検討する。順序は「すべてを通過しなければ次へ進めない」という意味ではなく、各段階で表現可能性と意味喪失の有無を記録する検証手順である。

1. **既存の汎用DSLとSource**：既存のData/Function/Relation/Rule、関数チェーン、Query、Markdown・表・Graphviz・Native SQLなどで記述可能か。原文を保持する。
2. **既存の意味概念の汎化**：変数、条件、制約、評価、目的、時刻、値域などが、業務と研究で同一の意味で使えるか。名称だけの類似は採用根拠としない。
3. **標準データ構造・表現型**：数式・単位・多次元配列・グラフ・時系列等を既存Facet/Mappingで表せるか。表現できても意味、丸め、順序、多重度、境界条件を失っていないか。
4. **Operator／契約**：データを受けて結果を返す共通の計算・変換・検証契約が必要か。Evaluate、Solve、Sample、Windowなどは能力分類の候補であり、新しい予約語として確定しない。
5. **Provider／Target／専門Plugin**：R、Python、SQL、OR-Tools等の外部実行、独自描画、分野別メタデータで実現する。Contractは共通化できても、アルゴリズムの完全同義性を仮定しない。
6. **Core採用の再判断**：既存Contractでは無理な最小のギャップ、複数異分野の実例、意味保存の証拠、後方互換性をもってOwnerが判断する。

専門ツールキットはこの手順を使って構築するが、分野固有の専門操作が業務システム一般にも役立つことを採用条件にはしない。

## 4. 主な能力と役割の分担

| 題材 | 標準データ／意味の候補 | Operator／Providerの候補 | 固有性と注意点 |
| --- | --- | --- | --- |
| Wolfram Language型の数式処理 | 式木、Symbol、定義域、論理式、仮定、量・単位 | 簡約、微分、解法、数値評価 | 同値変形の前提条件、複数解、数値解と証明の区別 |
| Rの統計計算 | 表、群、分布、観測値、欠測、標本設計 | 推定、検定、回帰、再標本化、DOE | 検定の適用前提、欠測処理、自由度・精度、乱数と多重比較 |
| Window／分析帳票 | 型付き行集合、順序、時刻、Partition、Frame | SQL Window、移動計算、分位点、ランキング | ROWS/RANGE/GROUPS、NULL、ties、時間帯、隠れた取得範囲の差 |
| WBS／PERT／CPM | 階層WBS、依存有向グラフ、Calendar、Resource、Baseline/Actual | トポロジカルソート、CPM、確率的見積、資源制約最適化 | WBS親子≠先行関係、循環不可、稼働日・期間・遅延の定義 |
| Solver・最適化 | 変数、値域、制約、目的、単位、初期条件 | LP/MIP/CP/非線形/多目的Provider | feasible/optimal/infeasible/unbounded/unknown、証明境界、許容差 |
| 物理系・熱力学 | 系、境界、状態量、流量、接続、方程式、保存則 | DAE/ODE解法、イベント処理、FMI連携 | 温度と温度差、単位・次元、連続／離散の整合、収束 |
| 生命科学 | Species、Compartment、Reaction、Stoichiometry、時系列、Graph | SBML系モデル連携、数値Solver、Pathway表示 | 化学量論、量と濃度、反応速度、SBGN図の意味差 |
| 科学可視化 | 連続値、軸・単位・座標、解像度、Tile、Version、Mapping | 専門Renderer、LOD、統計図／反応図生成 | 色を原値の正本にしない、再描画と編集逆写像の適格性 |

数学・統計・科学の「意味」はProviderによる実装ではなく、既存モデルの定義と必要な分野別契約に帰属する。数学的な公理・定理・アルゴリズムの導入を新たなKnowledge Storeの正本とせず、Source・Contract・Provider版と根拠を既存の仕組みで追跡する。

## 5. 設計文書をStudioから業務・研究アプリへ自然に展開する

編集するMarkdown、表、図、ネイティブ資産は、原形式を保持できるSourceである。説明文から無条件に実行意味を推定せず、機械が検証できる設計要素を既存Semantic/Function/Scenarioへ対応付け、Source ID・位置・Revisionを追跡する。見た目の構成や自動整形は、意味の変更として扱わない。

同じ設計要素を、Studioと一般のアプリケーションで使えるようにする。Studio用の第二DSLや第二Generator正本は作らない。

| 同じ設計上の意味 | StudioでのConsumer | 一般アプリでのConsumer |
| --- | --- | --- |
| 変数・条件・制約・目標 | Inspector、制約入力、診断、試行比較 | 業務ルール編集、配車・人員配置、最適化フォーム |
| 集合・Window・統計式 | DB Workbench、性能計測結果 | BI、会計帳票、観測系列分析 |
| 階層・依存グラフ・期間 | Studio自身の実装WBS、開発進捗 | 案件計画、生産・研究スケジュール |
| 数式・系・反応ネットワーク | 専門ツールキットを載せたモデル編集 | 工学・生命科学の生成アプリ |

**受入に必要な一往復**：Authoring Source → accepted Semantic → Operator Contract／Provider選択 → 生成された実Artifact → 実Call／結果 → Studioと一般アプリケーションそれぞれの実際の利用先 → 独立Expected。Studioの表示が通っただけでは、対象アプリの実行や生成完了を主張しない。

## 6. 設計文書を生成アプリケーションへつなぐ契約

研究・業務・Studio自身の設計を別々の専用言語に分割しない。Markdown、表、数式、Graphviz、ネイティブSQLなどは元の形式を保ち、機械が解釈できる部分を既存の意味モデルとOperatorの契約に結び付ける。

設計文書の役割は次の三つに分ける。

| 設計文書にあるもの | TEI側の扱い | 生成先での扱い |
| --- | --- | --- |
| 利用目的、判断理由、説明、注記 | 原文を保持し出典の位置に戻れる | 説明文・操作ガイド・仕様資料に利用 |
| 型付きデータ、条件、演算、依存、操作 | 確定した意味モデルとSourceを関連付ける | DB/API/Batch/UI/計算Targetを生成 |
| 例、テストパターン、期待値、TODO、外部部品契約 | Scenario、独立Expected、Mockと実Providerを区別する | Mock段階の全体シミュレーションから実結合テストへ移行 |

外部Solverや科学計算部品の実装が未完成でも、設計上の入力・出力・計算前提・エラー・状態を契約として定義し、**TODOとMockを明示した全体シミュレーション**を可能にする。Mockの成功は実Solverの成功には昇格しない。業務プロセスの文章は、意味を保持した図表・DSLから規則的に生成できる密度を目指し、単純な集計手順まで無理に図形化しない。

Studioでは、Schemaを読み取って得た構造や意味の候補を、ツリー、図、Inspector、入力支援などへ投影する構成も検証する。JSON Schemaなどの既存形式とTEIの設計文書を、専用の図面モデルへコピーして二重管理しない。まず原文の参照・読み取り専用表示から始め、固定した対象と版を確認できるようにする。続いて、診断と一項目の編集案を既存の差分・影響確認へ渡し、保存後に原文を再読取して整合性を確かめる。構文解析・意味解釈・表示・逆編集・実行は、それぞれ独立して検証する。これを新たなGrammarエンジンや第二のEditor Runtimeの実装義務とはしない。

業務プロセスや研究の実験手順は、起動条件、処理、状態変化、結果の応答を共通の操作・イベント構造として記述できるか検討する。WBSの作業依存、タイマーによる起動、DOEの実験順序は、似た図面になっても同じ時間・実行意味とは限らない。共通化するのは構造や参照であり、スケジューラや実行Hostまで統一するわけではない。

UI生成では、Studioの編集・比較操作と、生成アプリの運用・承認操作で異なる配置・権限・Targetを認める。共通なのはSource/Meaning/Function/Operator/Expectedであり、固定のUI実装ライブラリではない。Studioの選択・Pin・改訂確認と生成アプリの入力・更新許可も混同しない。

## 7. 計算の意味と実装を分ける契約

計算の要求を意味上のProblem/Planとして記述し、実行するProvider/Target、実行結果の解釈・受理と区別する。必須の共通フィールドを増やしすぎないため、次は契約の**確認項目**であって、新しいCore構造の確定案ではない。

| 確認項目 | 記録すべき情報 |
| --- | --- |
| Problem/Plan | 入力、値域、制約、評価基準、アルゴリズムの選択条件 |
| Implementation | Provider/Toolchain/Algorithm・ライブラリ版、Lock、数学的前提 |
| Run | 実験計画版、データの版、乱数種、計算環境、Attempt、実消費入力 |
| Result | 結果の定義、単位・精度・誤差、計算状態、時間・資源費用、収束とGap |
| Evidence | 参照期待値・実結果、失敗原因、実Artifact、観測範囲、利用したConsumer |
| Acceptance | 独立Expectedへの一致、分野専門家/Ownerによる結果の用途別受理 |

World Fingerprint／Model Revision／Artifact Revisionと、Experiment Plan／Run／Result identityは互いに代用しない。Byte一致で数学的正しさを保証しない。乱数・浮動小数点・並列計算の再現性は、同一入力再実行、許容差内一致、独立検証を別の主張に分ける。

### Solverの結果を扱う際の区別

- 実行失敗と数学的な不可能（infeasible）を分ける。
- 解の存在（feasible）と最適性の証明（optimal）、許容差内の近似（gap）、未解決（unknown/timeout）を分ける。
- 単目的、優先順位付き、多目的/Paretoを勝手に一つの加重合計へ変換しない。
- 物理単位や定義域の違反は、求解へ渡す前に分かる範囲で検査する。
- 解を業務上の確定や外部効果の許可へ自動昇格させない。

### 実験計画法で保持する情報

- 水準の全組合せ列挙と、実験単位、ランダム化、ブロック化、反復、交互作用を区別する。
- 結果を見てから選んだ仮説と、事前に固定した仮説を区別する。
- 設計上の条件が一致しても、観測欠落、順序効果、交絡、母集団の違いは別途評価する。

## 8. 数値計算の品質と製品化の判断基準

### 8.1 数値・科学処理に共通する確認事項

- **QuantityとDimension**：物理量の値、単位、次元、絶対温度と温度差、換算時の精度・オフセットを区別する。
- **ExpressionとEquality**：プログラム代入、数学的等式、制約、常微分・代数方程式、論理命題を表示記号だけで同一視しない。
- **DataとObservation**：観測時刻、取得時刻、改訂、欠測、外れ値、標本集団、取得範囲・権限を区別する。
- **ExperimentとProduction**：実験計画、Run Attempt、評価、業務確定、外部効果の責務と承認を分離する。
- **UncertaintyとGuarantee**：計算誤差、統計的信頼区間、予測不確実性、SolverのGap、実行結果の資格は異なる意味である。
- **Domain Predicate**：分析上の相関・予測と因果的主張、数値的近似と論理的証明など、分野ごとの強い主張をOperatorの成功で自動受理しない。

### 8.2 ツールキットを独立した製品候補として比較する

ツールキットの技術構成は既存[#951](https://github.com/hat47x/tei-platform/issues/951)/[#1016](https://github.com/hat47x/tei-platform/issues/1016)などのPackage・Contract・Contribution・Version・Scopeに従う。新たなPlugin階層やRegistryを必須としない。**ツールキットは実用上の構成・配布・学習の単位**であり、技術的な単一Pluginとは限らない。

| 比較する軸 | 測定・確認するもの |
| --- | --- |
| 使い始められるか | 代表データから最初の正しい成果を得るまでの操作、習得負荷 |
| 時間を減らせるか | 既存R/Python/Mathematica/PMツールでの同一作業との比較 |
| 意味と品質 | 計算契約の同義性、専門標準、Independent Expected、専門家による検証 |
| 構成と運用 | Provider調達、ライセンス、更新、インストール、異常時の扱い |
| 生成資産の再利用 | Studioから非Studioアプリへ戻せるSource/Meaningの比率と欠落 |
| 市場適合性 | 現利用者への課題調査、代替ツール、継続利用・支払意思等の検証 |

特定専門市場に深く入る判断は、上表の証拠で行い、「汎用性が低いからCoreに入れない」という判断を専門ツールキットの不採用理由に転用しない。外部の数理・生命科学コードやデータを移植・再配布する場合は、公開手法の独立実装と第三者成果物の利用条件を区別して判断する。

### 8.3 作業順序と受入の引渡し

- **P0を変更しない**：Studio Desktop実セルフビルド（[#2831](https://github.com/hat47x/tei-platform/issues/2831)/[#4583](https://github.com/hat47x/tei-platform/issues/4583)）。数理Pluginを依存に追加しない。
- **並行P0設計**：TA-01a/02aのFixture、Fit/Gap、標準表現型とSourceの結合の検討。Owner #2495/#2497/#2387/#3791へ戻す。
- **P1実際の利用先**：TA-01/02をHeadless・Studio・非Studioの別Consumerで実証。独立Expectedと現行資格を別Claimに保持。
- **P2専門計算**：TA-03a/04aを小さな外部Provider・Mock→実Providerへ段階接続。初期から巨大な汎用科学実行機構を設けない。
- **P3専門製品化**：数式・物理系・生物系の専門ツールキットは、具体的な利用者とワークフロー・専門規格・実計算の受入を得て個別に判定する。

**未受理事項**：新たな汎用数式型・文法、Solver/DOE Provider、数値品質、Windows実際の利用先、専門業務システムの生成、市場適合。資料・Issue・Mockの整備と、実装済み・製品受理を混同しない。

## 9. 専門ツールキットの構成と製品化

**Plugin**は技術契約に基づく拡張単位、**Toolkit**は利用者が一連の課題を解くための構成物である。ツールキットはモデル雛形、Operator、Provider、Studio編集面、生成用UI、検証Case、レポート、説明を束ねる。ただし全要素を必須とはしない。

| ツールキット候補 | 最初の製品的な一巡 | 共通機構に還元する部分 | 専門部分 |
| --- | --- | --- | --- |
| Analytics & Reporting | SQL/時系列の集計→表／図→レポート | Query、Window、Snapshot、Mapping | 分野別指標・表示規則 |
| Project Planning | WBS＋依存DAG→CPM→ガント／変更比較 | Graph、Calendar、制約、評価 | 工程・資源・計画手法 |
| Statistics & DOE | 因子設計→試行→分析→検証報告 | Sample、Evaluate、Result/Evidence | 推定・検定・実験方法 |
| Solver & Optimization | 制約入力→実Solver→結果比較→承認 | Problem、Constraint、Provider Contract | 求解方式、目的の意味 |
| Mathematical Modeling | 式→変形／求解→検証・表示 | 式・論理・単位・誤差 | 数式処理の専門実装 |
| Physical Systems | 系／保存則→外部計算→時系列図 | Graph、Quantity、状態・時間 | 熱・流体・動力学の専門意味 |
| Systems Biology | SBMLモデル→反応計算→SBGN等の図 | Graph、MultiDim、Mapping | Species/Reaction等の専門意味 |

市場優先順位は「Coreへ貢献するか」だけで決めない。専門家の作業時間削減、現行ツールの不足、共有・配布・ライセンス、専門標準適合、保守可能性、得られる製品価値を別票で評価し、仮説・実測・利用者評価を混同しない。

第三者製品の仕様・公開された手法を参照することと、著作権で保護されたコードや素材を移植・再配布することは別判断とする。Rパッケージ・数式処理エンジン・生物学用データには実装単位のライセンスと配布条件を確認する。

## 10. 異分野での検証事例と独立した期待結果

| ID | 入力と実行対象 | 独立Expected／負例 | 製品上のConsumer |
| --- | --- | --- | --- |
| TA-01 Window | 同値の取引行と時刻付き観測データ、NULL・同順位・時刻境界を含む | frameの差で結果が変わること、Native SQLの取得範囲不明を完全集計と称さないこと | Studio DB Workbench＋一般分析アプリ |
| TA-02 Planning | WBS 6項目、依存DAG、非稼働日、2資源 | 最長経路の独立手計算、依存循環拒否、CPMと資源制約付き解の差 | Studio開発計画＋業務案件WBS |
| TA-03 Solver | 小さな制約充足と最適化問題 | feasible/optimal/infeasible/timeout/unknownを区別、単位不一致拒否 | 配置/割当アプリ、比較Inspector |
| TA-04 DOE | 2因子・2水準、ブロックと乱数種を固定した小事例 | 単なる直積列挙≠適切なDOE、欠測・交絡・実験順を保持 | TEIビルド性能測定＋品質分析アプリ |
| TA-05 Mathematics | 前提付き式変形・複数解・数値近似 | 前提のない平方根・除算の変形拒否、近似値≠証明 | 数式・業務計算の双方 |
| TA-06 Dynamics | 単純な熱収支モデル、単位・初期条件・時間刻み | 次元の不一致拒否、保存則の差分、数値刻みの影響を検出 | シミュレーションToolkit |
| TA-07 Biology | 2つのSpeciesと反応、Compartment、Rate | 量/濃度の混同拒否、反応方向・係数・図の意味喪失を検出 | 生化学Pathway Toolkit |
| TA-08 Visualization | 同一データの表/図/帳票、解像度違い | 色・レイアウト変更による原データ同一性維持、編集逆写像不能を明示 | Studioと生成アプリ |
| TA-09 Provenance | 同じモデル＋違うProvider版／乱数／Compiler | 再利用可否を別Claim、旧実行の結果で現行Providerを保証しない | 実験比較・Build・報告書 |

最初から全ケースを実装しない。TA-01/02を汎用構造の早期実証、TA-03/04をOperator実行と計算契約の検証、TA-05〜08を専門ツールキットの圧力ケースとする。実装済みの能力やCI成功として記述しない。

## 11. 実装時に照合する独立した期待結果

「説明ができること」と「生成して正しく動くこと」を分けるため、まず手計算や別系統の実装から期待値を定め、既存のScenario/Driverへ引き渡す。以下の値は**設計上の固定Fixture**であり、TEI実行による計測結果ではない。Test Pattern表や必要時のLean等による形式検証は、既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Expectedの一表現として扱い、第二のTest Runnerを導入しない。

### TA-01a：Window計算と行順序

対象系列を順序が一意な4行（値10、20、30、40）とし、行フレーム「現在行と直前2行」の平均を求める。

| 行 | 値 | 参照する値 | Expected |
| --- | ---: | --- | ---: |
| 1 | 10 | 10 | 10 |
| 2 | 20 | 10、20 | 15 |
| 3 | 30 | 10、20、30 | 20 |
| 4 | 40 | 20、30、40 | 30 |

- 順序を削除したQueryを同値扱いしない。時刻列に同順位がある場合、追加の安定順序キーまたは結果の未確定状態が必要となる。
- NULLを追加した別Fixtureでは、SQLの`AVG`がNULLを集計対象から除外することと、専門時系列Operatorが欠測値をどう扱うかを別Contractで検証する。NULL→0の黙示置換を認めない。
- 同じ4行でも`ROWS`と`RANGE`、カレンダーの3日間と直前3行は同じ処理とは限らない。
- Native SQLのRLS/Limitや非観測の条件により全行を取得したか不明な場合、`complete`な集計結果として受理しない。

### TA-02a：WBSと依存関係、資源制約の差

作業A=2、B=3、C=4、D=2（いずれも同一の時間単位、資源と休日は一旦無視）。依存関係はA→B、A→C、B→D、C→Dとする。WBS親子関係はこの依存関係とは別に保持する。

- CPMの最早時刻：Aは0–2、Bは2–5、Cは2–6、Dは6–8。完了時点は**8**、クリティカルパスは**A→C→D**、Bの余裕は**1**。
- BとCが同時に使えない専有資源を1単位だけ必要とする別Fixtureでは、両作業を直列化する必要があり、同じ依存条件で最短完了時点は**11**。この差を単なるCPM誤差ではなく、問題の制約追加として区別する。
- 依存DAGを循環させた場合はスケジュール計算前に拒否する。稼働日・休日・実績を追加した結果は、上記の単位時間Fixtureと混同しない。

### TA-03a：Solverの結果と状態

整数変数`x,y ∈ {0,1,2,3}`、制約`x+2y ≥ 4`、目的`min(x+y)`とする。手計算で最適解は**x=0、y=2、目的値2**。ただし実Providerがこの値を返しただけでは、最適性保証が得られたとは扱わず、実装の証明・Gap条件も確認する。

矛盾する別Fixture `x ≥ 2` かつ `x ≤ 1` の場合は、通常の解なし（infeasible）である。Timeout、実行エラー、外部Provider不達、unknownをinfeasibleへ読み替えない。

### TA-04a：DOEと「組合せ」以上の意味

要因A、Bがそれぞれ2水準で、同一条件を2回反復する計画を用いる。全組合せは**4通り**、実験単位は**8件**。各組合せの反復数が2であることと、乱数種・ランダム化法・ブロック・実行順序が記録されることを別々に期待する。

設計を通しただけでは、実際の試行が行われたこと、交絡がないこと、因果的な結論が成立することを主張しない。観測結果のない未実行Runや脱落を欠落のまま扱う。

### TA-05a：数学的意味の保存

実数`x`に対し、`√(x²) = |x|`であり、無条件に`x`へ簡約できない。反例として`x=-3`なら結果は3である。仮定`x≥0`が確認できる場合にのみ`x`へ変形可能となる。記号変形・数値近似・形式的な証明は別の能力として扱う。

これらのFixtureでは、TEIモデル、生成した実行コード、Studio表示、非Studioアプリ表示、説明・報告文に**同じ条件と期待値**が維持されることを確認する。Provider AとBの出力一致だけを独立Expectedとみなさない。

## 12. 実装の順序と完了条件

| 段階 | 作業 | 退出条件 |
| --- | --- | --- |
| P0 並行設計 | #2495/#2544/#2497/#4350などと照合したFit/Gap票、TA-01/02独立Expected、ツールキット候補を登録 | DSL/Core追加なしで表現可能な範囲、不可な範囲、Owner判断箇所が分かる |
| P1 Headless縦断 | Window＋Planningの各1件を既存Source→Operation/Provider→結果→独立Expectedで動作させる | 実入力・選択・実Call・出力に同一Source/Revision・Attemptの帰属がある |
| P2 実際の利用先 | Studioと非StudioのStudioと一般アプリの双方へ同一意味を投影し実使用する | 単なる図・合成結果ではなく、Target artifactと受領を別々に確認 |
| P3 専門能力 | Solver/DOE、次いで数式・物理・生物系のProvider/Rendererを適用する | 専門規格、数値的意味、ライセンス、負例、ドメイン専門家による受理 |
| P4 製品選定 | ツールキットごとに利用者の試行・生産性・保守費・差別化を計測する | Core汎用性と専門市場への適合性を別尺度で判断 |

全体P0は引き続き[#2831](https://github.com/hat47x/tei-platform/issues/2831)/[#4583](https://github.com/hat47x/tei-platform/issues/4583)の**非Plugin Studio Desktop Stage1→Stage2実再構築・生成したWindows WebView2 Consumerでの実使用**である。Toolkit機能をそのGateに加えない。計画・実測の小規模Dogfoodは並行して進められる。

反証の例：Source A→B→Aで旧計算結果を現在のものとして再利用する、同じ値域の別Facet演算を自動選択する、WBS親子を先行関係と扱う、NULLをゼロに変える、Window順序を省く、近似解を最適性証明とみなす、グラフ表示から元データへの可逆編集を偽装する、試験用Mockの成功を実Providerと実際の利用先の成功として扱う。いずれも不合格とする。

## 13. 計算の共通性を判断する基準

数理・統計・計画・科学分野を一つの基盤へ収める際に、共通化すべきものは「同じ見た目の構文」ではない。利用者が定義する値、条件、関係、計算の要求を、意味を失わずに保持し、どの実装を選んでも契約の違いが分かることである。TEIの標準DSLは、専門家が理解しやすい記述を優先する。数式や関数名を大量に予約語として追加する方針は採らない。

計算には、結果を一つの値として求めるものだけでなく、式の変形、制約を満たす候補の探索、実験条件の設計、時間に伴う状態変化などがある。それぞれの計算目的を区別したまま、既存のData・Function・Rule・Relation・Operator契約で受け止められるかを検証する。

| 処理の種類 | 利用者が求めるもの | 共通化する候補 | 同じものとして扱わない条件 |
| --- | --- | --- | --- |
| 集計・Window | 集合や順序に基づく派生値 | 型付き集合、粒度、順序、範囲、計算規則 | NULL、取得範囲、同順位、フレームの違い |
| 記号計算 | 前提を保つ数式の変形・求解 | 式、記号、定義域、仮定、変換関係 | 無条件の同値、近似値と証明の混同 |
| Solver | 制約を満たす解や目的値 | 値域、制約、目的、評価条件 | 実行成功と可解性、実行結果と最適性の混同 |
| 実験計画法 | 比較・検証のための試行条件 | 要因、条件、実験単位、観測・評価 | 組合せ列挙と実験設計、相関と因果の混同 |
| スケジューリング | 作業・資源・暦を考慮した配置 | 階層、依存、期間、制約、実績 | WBS親子と先行関係、CPMと資源制約解の混同 |
| 動的システム | 状態の時間変化と相互作用 | 状態、量・単位、関係、時刻、イベント | 微分・代数・離散イベントの評価規則の違い |

ここで挙げた概念は、正式なCore型や新しいDSL予約語の一覧ではない。まず既存の型・Facet・Role・Operator Contractへ対応付け、足りない意味を具体的な反例で示す。例えば、SQLの`PARTITION BY`と実験計画法のブロックには「集団を分ける」という共通点があるが、その分割の統計的な意味と演算規則は異なる。同じ`Group`という名称にまとめることだけでは汎用化したことにならない。

### 横断Fit/Gap票の最小記録

検証対象ごとに、(1) 原文と意味の定義元、(2) 必要なデータ構造・Facet、(3) Operatorが要求する計算意味、(4) Provider/Targetが実際に満たす範囲、(5) Studioと一般アプリケーションの操作、(6) 意味を失う変換・編集・計算の反例、(7) 独立した期待結果、の七項目を記録する。

変換が一方向にしか成立しない場合は、その範囲を明示する。既存の数学・科学言語をTEIへ無条件に完全変換したとみなさず、原文のまま保持する経路も正規の選択肢とする。標準表現型に収まらないことと、計算ができないことは別の問題として扱う。

## 14. 設計・計算・実行・受理を分ける

実験計画、最適化、性能比較、帳票の集計結果には、計算前の設計、選択された実装、実際に渡した入力、得られた結果、用途に応じた採否判断がある。これらの対応を、既存のSource、Operation、Build、実行Attempt、Evidenceと結び付ける。

| 段階 | 保持・確認するもの | 確認できたとはみなさないもの |
| --- | --- | --- |
| 設計 | 対象、条件、計算規則、想定する結果、Source位置 | 元の文章が自動的に正式な意味として受理されたこと |
| 構成 | Operator契約、必要な能力、Providerとその版、実行先 | Providerの宣言だけで実呼出しが成立したこと |
| 試行 | 入力データの版、取得範囲、乱数種、反復、呼出しAttempt | 設計したすべての試行が実行されたこと |
| 観測 | 実際の返答、解の状態、精度、欠測、警告、費用 | 数値が返っただけで、最適性や科学的妥当性が保証されたこと |
| 検証 | 独立した期待結果、誤差条件、負例、再実行の条件 | 異なるProviderの結果一致だけで独立検証が済んだこと |
| 受理・利用 | 確認した主体、用途、実Consumerが受け取った結果 | 計算結果が業務確定、外部効果の許可や製品完成へ自動昇格したこと |

既存のWorld Fingerprint、Model Revision、Artifact Revisionは計算Runの同一性を代用しない。入力値が同じでも、取得時点、Providerの版、乱数種、実行環境、計算精度が異なる場合がある。再利用してよいのは、対象となるClaimに必要な条件が改めて満たされるときだけである。

数値結果を比較する際は、ビット単位の同一性、許容差内の数値的一致、数学的な同値、統計的な妥当性、業務上の受理を別の判断として扱う。近似的な結果に必要な誤差の種類と範囲は、分野・演算・用途ごとに定める。共通の浮動小数点許容差を全Operatorへ強制しない。

この扱いは、TEI Studioのビルド性能比較にも共通する。生成された成果物が同じでも、測定対象のWorkload、Cache状態、実際の実行経路が違えば、改善の証拠としては不足する。逆に、実測が改善していてもSourceの意味が変わった場合は同等な最適化として受理しない。

## 15. Studioと生成アプリケーションが共有する編集の仕組み

TEI Studioは、設計文書の構造を読むだけの専用IDEにしない。Schema、Markdown、表、グラフ、数式、SQLなどのSourceを解釈できる範囲でモデルに対応付け、ツリー、図、Inspector、入力支援、診断、差分へ投影する。この仕組みを、専門ツールキットを組み込んだアプリケーションにも利用できるようにする。

最初の題材は、外部形式として広く使われるJSON Schemaの**固定した小さな例**とする。SchemaをすべてTEI DSLへ変換するのではなく、プロパティ、参照、必須条件、値の制約を読み取り、原文の位置に戻れる表示を作る。見た目を変えただけで意味が変化しないこと、構造を一意に戻せない表示からは無条件に書き戻さないことを確かめる。

編集は次の順序で検証する。

1. 固定したSourceとRevisionを読み取り、構造と原文位置を表示する。
2. 対象を選び、どのSource occurrenceに対応するかを固定する。表示上の一時的なフォーカスだけでは編集対象とみなさない。
3. 一項目だけの変更案を作り、型、参照、影響する範囲、変更しない部分を示す。
4. 元のRevisionが変わっていたり、逆写像が一意でなかったりする場合は、書込みを止めて再選択を求める。
5. 承認された変更を元の文書へ局所適用し、再読取と再表示によって、対象外の内容を保てたことを確認する。

この一連の流れは、StudioのSchema編集に限らず、WBSの作業、グラフの辺、分析式、科学モデルのパラメータを編集する際にも応用できる。ただし、専門形式ごとの編集可能範囲はそれぞれ明示する。Coreに新しいParser、汎用の文法実行エンジン、独立した編集用Runtimeを加えることを、初期実証の条件にはしない。

**優先順**は、既存[#2831](https://github.com/hat47x/tei-platform/issues/2831)/[#4583](https://github.com/hat47x/tei-platform/issues/4583)のStudio Desktop実セルフビルドを全体P0、Schemaの読み取り専用表示・原文への移動を並行する小規模な設計検証、編集・再投影およびDSL自己記述を後続の検証とする。現行のBuilder、通常CLI、Wails実Consumerの完了条件を変更しない。

## 16. 最初の三つの薄い縦断実証

全分野の機能を先に実装せず、現在の契約で表せる範囲を最小の事例で確かめる。ここでは、既存のTA-01〜09を**実装可能な順序と差分の条件**へ落とす。新しいScenario体系は作らず、既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)へ引き渡す。

### 16.1 ウィンドウ集計：まず小さなQueryから始める

四つの値10、20、30、40を一意な順序で扱い、現在行と直前二行の平均10、15、20、30を独立した期待結果とする。次にNULL、同順位、Partition、空集合、取得範囲の部分性を加え、結果の意味が変わる条件を確認する。

最初に確認するのは、既存のQueryとSQL方言の契約が、必要な`WINDOW`構成要素や実行条件を運べるかである。`WINDOW`句を生成できることは、任意のウィンドウ関数やフレーム・データ型に対応した証拠ではない。実際のDBを読む段階では、既存[#2387](https://github.com/hat47x/tei-platform/issues/2387)/[#2586](https://github.com/hat47x/tei-platform/issues/2586)の実Read、取得範囲、独立Consumer受領の条件を使う。Studioと一般分析アプリで同じ値を表示するだけでなく、どのSource・入力・Providerの結果かをたどれるようにする。

### 16.2 工程計画：階層と依存を混同しない

WBSの親子関係と作業の先行・後続関係を別の構造として保持する。A=2、B=3、C=4、D=2という小さな作業群で、A→B、A→C、B→D、C→Dを設定した場合、資源制約のない最短完了時点は8であり、A→C→Dがクリティカルパスとなる。同じBとCに排他的な一つの資源を割り当てれば最短11になる。

実装では、データ構造を表示すること、CPMを計算すること、資源を考慮して予定を決めること、実績を保存することを別々に検証する。暦の休日や時刻帯の意味は、時間単位だけの例を通した後に追加する。TEIの開発計画と一般案件のWBSで同じ編集・計算契約を共有できるかを確かめるが、Studio用のProject Managerを先に必須としない。

SUIのカードやキャンバス上の島は、検討中の情報を整理するための表示・分類であり、作業の実行順序を確定したものではない。SUIから工程管理へ展開する場合も、元カードのIDと出典・受理状態、WBSの作業ID、依存DAGの辺ID、工程計算の結果を別々に追跡する。自動配置や階層化を理由に依存関係を推測してはならない。

### 16.3 Solverと実験計画法：計算契約の差を確認する

Solverでは、値域・制約・目的を先に定め、実Providerの`feasible`、`optimal`、`infeasible`、`unknown`などの結果を区別する。最適解が返る場合と、最適性が証明された場合は別の主張である。実験計画法では、二要因・各二水準・二反復で四条件・八試行を定義し、実行順、乱数種、脱落、観測値を保持する。

両者は同じ「条件と結果」を扱うが、Solverは目的に沿って候補を探索し、実験計画法は条件の影響を調べるための実験を構成する。通常の条件列挙を実験計画法として受理しない。最初は外部部品の入出力契約とMockによるシミュレーションを認め、実Providerでの計算、独立検証、専門家による解釈を後段で確認する。

## 17. 専門ツールキットを製品として判断する条件

専門ツールキットの価値は、その分野の利用者が課題をどこまで解決できるかで判断する。数式処理、生命科学、物理シミュレーションなどは、汎用性が高いものだけを残すのではなく、専門的な能力を深く提供する選択肢を保持する。

ただし、数式やアルゴリズムを一つ呼び出せるだけでは、実用的なツールキットの完成とはしない。入力や既存資産の利用、モデルの編集、計算、結果の可視化、誤りの診断、出典の確認、レポートと成果物の受渡しまでを、対象分野で必要な範囲に限定して評価する。技術面の評価と、実際の利用者の生産性・継続利用・市場性の評価を別に記録する。

Toolkitの構成は、既存のPackage、Contract、Provider、Generator、表示用Contributionと構成Lockの仕組みで表せるかを先に確認する。専門分野ごとのソフトウェア、ライブラリ、データ、モデルの利用条件は、導入先・生成物への同梱・再配布のそれぞれで確認する。外部エンジンの採用がTEI本体の単一Goバイナリ配布を崩す場合は、本体に埋め込むのではなく、必要な生成Targetや外部実行環境との連携を優先する。

**採用の判断は二つ**に分ける。汎用基盤へ取り込むかは既存Core/DSL/標準データ・OperatorのOwnerが、専門ツールキットとして提供するかは対象分野の用途・利用者・必要な機能を根拠に検討する。後者の判断を前者の採用と取り違えない。

## 18. 既存計画への具体的な引渡し

| 優先順 | 次に作るもの | 主な既存Owner／接続先 | 受入の条件 |
| --- | --- | --- | --- |
| 全体P0（変更なし） | TEI Studio Desktopの実Stage1→Stage2セルフビルド | [#2831](https://github.com/hat47x/tei-platform/issues/2831)、[#4583](https://github.com/hat47x/tei-platform/issues/4583) | 生成した実Windowsアプリの起動・利用と、そのSource/Artifactの追跡 |
| 並行設計 | Window・WBS・Solver・DOEのFit/Gap票 | [#2495](https://github.com/hat47x/tei-platform/issues/2495)、[#2497](https://github.com/hat47x/tei-platform/issues/2497)、[#988](https://github.com/hat47x/tei-platform/issues/988) | 既存の意味・構造で表現できる範囲と、失われる情報が明確 |
| 最初の計算実証 | TA-01のWindow計算と独立した期待結果 | [#2387](https://github.com/hat47x/tei-platform/issues/2387)、[#2586](https://github.com/hat47x/tei-platform/issues/2586)、[#1180](https://github.com/hat47x/tei-platform/issues/1180) | Source→実Provider→結果→独立した期待値を区別して検証 |
| 次の構造実証 | TA-02のWBS/依存DAG/CPM | [#2495](https://github.com/hat47x/tei-platform/issues/2495)、[#2501](https://github.com/hat47x/tei-platform/issues/2501)、[#2514](https://github.com/hat47x/tei-platform/issues/2514) | 階層・依存・資源の意味を保ち、期待結果と負例を照合 |
| 並行する編集検証 | 固定JSON Schemaの読み取り専用表示とSource位置への移動 | [#2497](https://github.com/hat47x/tei-platform/issues/2497)、[#2620](https://github.com/hat47x/tei-platform/issues/2620)、[#1124](https://github.com/hat47x/tei-platform/issues/1124) | 原文の保持、安定した対象選択、編集不能な写像を明示 |
| 後続の専門実証 | Solver/DOE、物理系・生命科学の連携 | [#951](https://github.com/hat47x/tei-platform/issues/951)、[#1016](https://github.com/hat47x/tei-platform/issues/1016)、[#3791](https://github.com/hat47x/tei-platform/issues/3791) | 専門的な意味、計算品質、外部Provider、配布条件を確認 |

この表は、既存Ownerへの**受入候補の引渡し**であり、各Issueの優先順位や完了条件を直接変更するものではない。新たな横断Gate、第二の設計正本、共通計算Runtime、中央Evidence Storeを作らない。現行の実装・CI・Consumer受領や専門家による意味の採否は、各担当の記録によって別途確認する。

**この追補の状態**：設計と実証条件を具体化した段階。Window・Solver・実験計画法・専門ツールキットの実装、生成アプリケーションでの利用、利用者による製品評価は未実施である。

## 19. RDB・NoSQL・ログから計算するときに保持する意味

データベースの種類を広げることと、同じ計算がどの情報源でも成立することは別の課題である。TEIのQueryとOperatorが保持すべき情報を明確にし、各Providerが表現・実行できる範囲を検証する。既存の[Data Source Workbench設計](DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)、[SQL方言の実装上の制限](SQL_DIALECT_LIMITATIONS_2026-11.md)、[#2387](https://github.com/hat47x/tei-platform/issues/2387)/[#2586](https://github.com/hat47x/tei-platform/issues/2586)を担当の正本とし、本稿では数理・統計用途から必要な条件を引き渡す。

| 情報源と利用形態 | 最初に確かめること | 計算結果へ持ち込んではいけない推測 |
| --- | --- | --- |
| RDB／SQL | Relationと業務上の粒度、複合キー、結合条件、明示的な行順序、NULL、取得範囲 | SQLが実行されたので、意図した全件と業務上の一意な実体が得られたという判断 |
| Document／NoSQL | 文書のidentity、可変な構造、fieldのmissing／null、配列の多重度、検索・ソート能力 | 観測した文書のサンプルからCollection全体のSchemaが確定したという判断 |
| Graph／階層 | ノード・辺のidentity、向き、多重辺、探索範囲、訪問条件 | 同じlabelのノードを同一実体とする判断や、部分的な探索を完全な関係とする判断 |
| 時系列／ログ／Stream | イベント発生時刻、受領時刻、順序キー、遅着、重複、Window境界、Watermark／観測範囲 | 到着順が発生順と一致し、現在見える範囲が確定済みの全期間であるという判断 |
| 外部API／ファイル | ページング、Token、取得上限、権限・公開範囲、Source版、途中失敗 | HTTP成功やファイル読込み完了だけで取得対象全体の完全性を保証する判断 |

共通の計算計画では、少なくとも**対象のidentity、計算単位（grain）、順序、範囲、NULL／missing／未取得、評価時点、結果の完全性、SourceとProviderの版**を、必要な用途ごとに説明できなければならない。これらは既存のQuery、Data Context、Facet、Provider、EvidenceへのFit/Gap確認項目であり、新しいCore Schemaの必須列を宣言するものではない。

例えば、明細2件の金額がそれぞれ100で、別の関係にある配送行3件と単純に結合すると、明細を6行として集計するおそれがある。業務上の明細金額は200でも、JOIN後に何も考えず合計すれば600となる。同額明細が2件あるため、`SUM(DISTINCT amount)`でも100となり、正しい解決策ではない。**何の単位を数えているか**をモデル・Query・報告書の間で保持する必要がある。

Documentの配列も同様である。文書の件数と、配列要素を展開した後の行数は同じではない。配列要素のidentityを失ったまま表へ変換したり、missingをnullへ統一したりした場合には、変換の情報損失を明示する。表示上の行番号やGraphの位置だけで元のDocumentへの更新先を確定してはならない。

## 20. 計算Providerの能力と結果の同等性を分けて検証する

既存のSQL方言実装では、`SelectOptions`が名前付き`WINDOW`句を運べること、SQLiteの限定的な実行試験があることは確認されている。しかし、SQLのWindow frameや全集計関数、任意の方言への意味保存まで実証されたわけではない。**構文を出力できること、実行に成功すること、結果が独立した期待値に一致すること、利用先がその結果を受け取ったこと**を別々に確認する。

最初の検証では、同じ固定した四つの観測値をRDBとDocumentへ用意する。値は10、20、30、40、各観測には一意なIDと順序キーを付ける。現在行と直前二行の平均が10、15、20、30になることを、実装に先立つ期待値として保持する。SQLはDB側でWindow計算できる範囲を、Document側はProviderが順序付き取得や集計を保証できる範囲を確認する。**両方を無条件に同じ方法へ変換することは求めない。**

| 状況 | 期待する判定 |
| --- | --- |
| 対象Providerが必要なWindow計算と順序・範囲を保証 | その能力を明示して実行し、実入力と結果を独立Expectedへ照合する |
| 計算能力はないが、権限・取得上限・費用・対象範囲が明示され、有限の全入力を取得できる | 別の計算経路は明示的なPlanとして提示可能。自動Fallbackはしない |
| 先頭・末尾の切取りやRLS、APIページング等で対象全体が観測できない | 部分的な観測として表示し、全件に対する結果を確定値としない |
| missing／null／NaN、Decimal精度、時間帯、同時刻の順序がProvider間で異なる | 非同等、要変換、比較不能のいずれかを理由付きで示す |
| 全件取得・ローカル計算が秘密情報の開示や予算を超える | 計算を拒否または再計画する。黙った転送や実行は行わない |

### 最小試験パケット：七つの観測値から始める

最初の試験には、順序が一意なA系列の四行と、同順位・NULLを含むB系列の三行を用いる。数値は小さく固定し、丸め誤差の検討は次の段階に分ける。以下は**SQLite向けの参照SQL**であり、TEIのQuery生成器が現時点でこのSQLを生成できるという主張ではない。

```sql
WITH obs(grp, seq, obs_id, value) AS (
  VALUES
    ('A',1,'A1',10), ('A',2,'A2',20),
    ('A',3,'A3',30), ('A',4,'A4',40),
    ('B',1,'B1',5),  ('B',2,'B2',NULL),
    ('B',2,'B3',15)
)
SELECT grp, obs_id, seq, value,
  AVG(value) OVER (
    PARTITION BY grp ORDER BY seq, obs_id
    ROWS BETWEEN 2 PRECEDING AND CURRENT ROW
  ) AS avg3,
  AVG(value) OVER (
    PARTITION BY grp ORDER BY seq, obs_id
    ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
  ) AS cumulative_rows,
  AVG(value) OVER (
    PARTITION BY grp ORDER BY seq
    RANGE BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
  ) AS cumulative_range
FROM obs
ORDER BY grp, seq, obs_id;
```

| 系列・行 | avg3 | 累積ROWS | 累積RANGE | 何を確かめるか |
| --- | ---: | ---: | ---: | --- |
| A1 | 10 | 10 | 10 | 先頭行 |
| A2 | 15 | 15 | 15 | 直前一行 |
| A3 | 20 | 20 | 20 | 三行に達した時点 |
| A4 | 30 | 25 | 25 | 直前三行と累積範囲は別 |
| B1 | 5 | 5 | 5 | 系列の分割 |
| B2（NULL） | 5 | 5 | 10 | NULLの除外と、RANGEが同順位B3を含むこと |
| B3 | 10 | 10 | 10 | 同順位でもobs_idによってROWSの順序を確定 |

上のSQLは**SQLite 3.46.1で単独実行して値を照合した参考例**である。これはSQLの参照結果に関する限定的な確認であり、TEIのSource読込、SQL生成、通常CLI、実Provider選択、Studio表示、生成アプリケーションの受入試験ではない。

比較すべき追加の負例は、A1を計算入力から削除したときA3のavg3が20ではなく25になること、B2とB3で順序キー`obs_id`を落とすと`ROWS`の結果を一意に決められないこと、Documentの値が未記載（missing）である状態をSQLのNULLに黙って変換しないこと、の三つとする。結果表示のページングだけを変更した場合は、計算前の入力を変更していない限り同じWindow結果でなければならない。

実行に移すときは、独立した期待値、固定FixtureのRevision、実Provider/DBとSQL、読み取った範囲・権限・順序、実際の出力、Consumerの受領を別々に記録する。DB接続、閲覧、テスト結果の表示を一つの「PASS」にまとめない。これらは既存のScenario/Expectedに引き渡す入力であり、第二のTest Runnerを作るものではない。

### Query変換の適合を説明するための判定語彙

SQLとTEI Query、NoSQLのNative Pipeline等を相互に扱うときは、次の区別をレビュー票に残す。この分類は**設計上の記録語彙の候補**であって、新しいCore enumや正式なSQL AST/IRの採用を宣言するものではない。

| 判定 | 意味 |
| --- | --- |
| Equivalent | 対象、粒度、順序、NULL、Window範囲、取得条件が保たれることを検証済み |
| Conditional | 明記した入力型、Provider能力、範囲・版・権限等の条件下だけで意味を保てる |
| Native | 専門記法を原形のまま保持する。汎用Queryへの変換や逆変換は保証しない |
| Unsupported | 必要な意味や実行保証を現在の経路で表現・実行できないため拒否する |
| Unverified | 対応の可能性はあるが、必要な実装や検証の証拠がない |

`Equivalent`は構文の形だけで判定しない。`Conditional`は条件が満たされなければ`Unsupported`または`Unverified`に戻す。`Native`を選んでも、元のSQL・Pipelineを実行してよいかは対象Provider、入力、Hostの許可・開示条件に従う。

時系列のWindowは、取得した表示範囲だけで計算すると境界近くの値が変わることがある。例えば第3行の平均は、全系列では10・20・30の平均20だが、第2行以降しか取得できなければ20・30の平均25になる。**表示する行の範囲と、計算に必要な入力の範囲は異なる**。入力の必要範囲を満たせない場合は、欠けたデータを埋めたことにせず、未確定とする。

異なる情報源を結合する場合には、スナップショットの時点、データの鮮度、読み取り権限が一致するかを確認する。共通時点を保証できない複数Providerを、ひとつの整合した分散トランザクションの結果として扱わない。結果が偶然一致しても、両Providerの実行保証が同じとは限らない。

Providerの選択は「SQL対応」「NoSQL対応」という種類だけでは足りない。**対象Instance、実際に読むデータ、環境、利用主体とそのScope、要求するOperation**の組合せに対し、必要な能力が利用できるかを確認する。現在の権限や開示条件、入力の取得範囲が確定できない場合は、分析結果の完全性を高く見積もらない。

## 21. 工程計画・イベント・科学計算の時間を区別する

WBS・PERT・CPMで扱う予定は、作業の開始・終了を計画するための情報であり、実行Hostがその時刻に処理を起動した証拠ではない。実験計画法の試行順序も、実験室での実際の実施時刻とは異なる。ログやStreamの到着時刻、計算対象の発生時刻、Studioが結果を受け取った時刻も区別する。

したがって、同じ時間軸を画面へ表示しても、**計画時刻・対象の有効時刻・観測時刻・実行Attempt・通知／受領**を必要に応じて分ける。関係をGraphとして表示することはできても、どの規則で時間が進み、遅延や重複を扱うかはModel of ComputationとProvider/Hostの契約に属する。既存の[実行モデル設計](PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)と[#4198](https://github.com/hat47x/tei-platform/issues/4198)/[#4495](https://github.com/hat47x/tei-platform/issues/4495)に従い、新しい汎用SchedulerやEvent LoopをCoreに追加しない。

例として、工程計画に表示した「9時開始」を起点に実ジョブが起動した場合でも、起動要求、実際の開始、処理完了、結果の保存、通知の受領は別の出来事である。計算が失敗したり、通知だけ再送されたりすることがある。表示上の予定をそのまま実行・確定の証拠にしない。

科学計算では、モデルの時間刻みと実計算にかかった時間も区別する。物理シミュレーションの時刻`t=10s`は壁時計で10秒後を意味しない。刻みの誤差、単位、状態遷移条件、Providerの収束結果を保持し、時系列表示が正しくても計算の妥当性まで確認済みとはしない。

## 22. 最小のFit/Gapと検証票を既存担当へ渡す

ここまでの方針を、実際の設計・実装判断に使える最小事例へ落とし込む。以下は既存TA-01〜09および[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)へ渡す**追加の反例候補**であり、新たなテスト体系や合否の正本ではない。

| 例 | 独立Expected・拒否条件 | 既存担当 |
| --- | --- | --- |
| CTK-01 | 同じ四観測をRDB／Documentで扱う。完全に観測した入力のWindow平均のみ10、15、20、30と判断する | #2387／#2586／#3270 |
| CTK-02 | 時刻同順位、欠損、null、NaN、数値精度の違いを黙って同一視しない | #2495／#2514 |
| CTK-03 | 画面表示の先頭行が失われたことでWindow境界が変わる場合、完全集計として表示しない | #2506／#2586 |
| CTK-04 | 明細2件×配送3件のJOINで、金額200を600や100へ誤変換しない | #2387／#1126 |
| CTK-05 | Document配列を展開しても文書・要素のidentityを追跡。missingをnullへ無断変換しない | #2495／#2497／#3270 |
| CTK-06 | ProviderにWindow機能がなく全件取得も許可されない場合、黙ったローカル代替計算を行わない | #951／#2622／#3270 |
| CTK-07 | 工程の予定時刻があっても、実処理の起動／保存／通知が未観測なら実行済みと表示しない | #4198／#4495 |
| CTK-08 | 二つのProviderで数値が一致しても、Snapshot時点・実入力・Scope・許可が異なるなら同一条件の再現と扱わない | #2582／#2600／#2514 |
| CTK-09 | 分析・Window・科学計算の読取り操作が、DB未初期化を理由に暗黙のDDLやMigrationを実行しない | #4234／#4657／#3270 |
| CTK-10 | B2のNULL、同順位のB3を含む参照SQLで、ROWS 2 PRECEDING／累積ROWS／累積RANGEの値を取り違えない | #2387／#2514 |
| CTK-11 | SQL/Query/Native演算の変換を、意味一致／条件付き／Native保持／非対応／未検証として説明し、未確認を同義としない | #2387／#951／#2497 |
| CTK-12 | SUI上のカード・島・配置をWBSの先行関係へ自動昇格させず、独立した作業IDとDAGを保持する | #2495／#2497／#3791 |

分析や科学計算を目的とする読取り操作から、DBの作成・Schema変更・Migration適用を暗黙に起動しない。未初期化のDBを必要とする場合は、既存のDB設計・Migrationの計画、明示的な実行許可、実DB再観測へ別の操作として引き渡す。Solverが提示した業務案も、承認された業務操作や外部Effectとは別に扱う。この境界は、既存[#4234](https://github.com/hat47x/tei-platform/issues/4234)/[#4657](https://github.com/hat47x/tei-platform/issues/4657)の実装・受入に従う。

実装順は、既存のStudio Desktopセルフビルドを最優先としたうえで、まず**既存SQL方言で表せる限定Window計算**のSource・SQL・実結果・独立Expectedを照合し、次に**固定Documentの読取りと表への投影**を試す。その後、RDBとDocumentで意味上同じ計算が可能な範囲、計算できないときの理由、実Consumerでの表示を確認する。Document側に存在しない集計能力を、無条件のローカル全件取得で補わない。

Solver・実験計画法は、同じ計算契約・観測・受理の分離を使う後続事例とする。Rや科学計算の外部Providerについても、型、精度、実行方法、ライセンスと成果物の配布条件を、対応Targetごとに確かめる。StudioのDB Workbenchや図表示の完成を計算の第一歩の前提にはせず、Headlessの限定実証から必要なUIとTargetへ広げる。

**現在の状態**：本節は既存のRDB・NoSQL・Query・実行モデル設計を、数学・統計ツールキットの条件として読み替えた計画である。RDBとDocumentの計算同等性、外部データソースの完全性、専門Provider、実Consumerの受領を新たに検証したものではない。

## 23. 外部資料（仕様調査の入口）

- [Wolfram Language：Equation Solving](https://reference.wolfram.com/language/guide/EquationSolving)
- [R CRAN Task View：Experimental Design](https://cran.r-project.org/web/views/ExperimentalDesign.html)
- [OR-Tools：Scheduling（CP-SAT）](https://github.com/google/or-tools/blob/stable/ortools/sat/docs/scheduling.md)
- [PostgreSQL：Window Functions](https://www.postgresql.org/docs/current/tutorial-window.html)
- [Modelica Language Specification](https://specification.modelica.org/)
- [FMI Standard](https://fmi-standard.org/)
- [SBML](https://sbml.org/) / [SBGN](https://sbgn.github.io/)

外部資料は参考資料であり、TEIの意味仕様や対応済み機能の証明ではない。各実装で必要なアルゴリズム・規格・版・ライセンスは、採用時に再照合する。

## 24. 確認できたことと未実証のこと

この文書は、2026年11月14日時点の既存Issue・設計資料・作業中のPRと、分野横断の検討を踏まえた**設計・検証計画**である。新しいGo・R・Python・Windows・SQL・科学計算の実行、独立した数値照合、専門家レビュー、市場調査は行っていない。既存の限定的な試験や設計例だけを根拠に、数値計算やSolver、実験計画法、専門ツールキットが製品として完成したとは判断しない。採否は既存OwnerのIssue/ADR/実Evidenceに戻す。
