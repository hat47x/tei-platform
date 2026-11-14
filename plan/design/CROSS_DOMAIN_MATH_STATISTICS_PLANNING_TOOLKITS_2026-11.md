---
title: "数理・統計・計画・科学向けツールキットの設計と検証計画"
date: "2026-11-14"
status: "評価待ち（未採用）／実装前設計・受入計画"
issue: "https://github.com/hat47x/tei-platform/issues/4660"
---

# 数理・統計・計画・科学向けツールキットの設計・検証計画

## 1. 目的と結論

TEIの価値を、(a) 受注・管理・分析などのアプリケーションを高い生産性で設計・生成することと、(b) TEI Studio自身を知的な設計・編集・分析環境として利用し、同じモデルからStudio以外のアプリケーションへ展開できること、の両面から検証する。

Mathematica（Wolfram Language）やRに見られる数理・統計の体系、Window処理、Solver・実験計画法（DOE）、WBS・PERT・CPM、動的な系、生命科学の反応ネットワークや専門可視化は、有力な適用対象である。ただし、各製品の言語や独自形式をTEIの標準DSLへ移植することを目的にはしない。表現・数学的意味・実現手段・UIを分け、既存のTEI標準契約を先に使う。

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

**受入に必要な一往復**：Authoring Source → accepted Semantic → Operator Contract／Provider選択 → 生成された実Artifact → 実Call／結果 → Studioと非Studioそれぞれの実際の利用先 → 独立Expected。Studioの表示が通っただけでは、対象アプリの実行や生成完了を主張しない。

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

## 13. 外部資料（仕様調査の入口）

- [Wolfram Language：Equation Solving](https://reference.wolfram.com/language/guide/EquationSolving)
- [R CRAN Task View：Experimental Design](https://cran.r-project.org/web/views/ExperimentalDesign.html)
- [OR-Tools：Scheduling（CP-SAT）](https://github.com/google/or-tools/blob/stable/ortools/sat/docs/scheduling.md)
- [PostgreSQL：Window Functions](https://www.postgresql.org/docs/current/tutorial-window.html)
- [Modelica Language Specification](https://specification.modelica.org/)
- [FMI Standard](https://fmi-standard.org/)
- [SBML](https://sbml.org/) / [SBGN](https://sbgn.github.io/)

外部資料は参考資料であり、TEIの意味仕様や対応済み機能の証明ではない。各実装で必要なアルゴリズム・規格・版・ライセンスは、採用時に再照合する。

## 14. 確認できたことと未実証のこと

この文書は、2026年11月14日時点の既存Issue・設計資料・作業中のPRと、分野横断の検討を踏まえた**設計・検証計画**である。新しいGo・R・Python・Windows・SQL・科学計算の実行、独立した数値照合、専門家レビュー、市場調査は行っていない。既存の局所試験、生成器・Provider宣言、設計例から、製品としての科学計算・Solver・DOE・Toolkit完成を主張しない。採否は既存OwnerのIssue/ADR/実Evidenceに戻す。

