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

## 2026-11-02 Everything is Data横断の証拠接合（提案・既存Gate内）

[#2619](https://github.com/hat47x/tei-platform/issues/2619)のEverything is Data方針は、業務以外の構造・定義体も含めた製品スコープの拡張候補として扱う。実装上は、既存の[#1119](https://github.com/hat47x/tei-platform/issues/1119)のG0〜G4、Source/Owner意味受理、Provider/Hostと独立実測を利用し、**新しい判定段階・第2のCanonical・TEI独自Hostを追加しない**。

最初の**同一試行**で接ぐ対象を次の順に確認する（段階名は検証手順、Gate追加ではない）。

| 接合順 | 明示して比較すること | 採用前の根拠／未確認 |
| --- | --- | --- |
| 1. Authoring→標準データ候補 | Markdown表・簡潔な式/DSL・Function形のSyntax差、Source位置、Role/Operator IDと版、候補≠意味受理 | [#2625](https://github.com/hat47x/tei-platform/pull/2625)隔離Python 23件、実TEI Language Plugin未実行 |
| 2. Context/Rule依存 | Display FootprintとRule Dependency Footprintを分離。利用Relation/Field/Grain/Scopeを明示し、要求した全件性と実取得の全件性を区別 | [#2624](https://github.com/hat47x/tei-platform/pull/2624)隔離Python 20件＋別18件、実TEI Rule/Context未接続 |
| 3. 複数Provider実現 | 同じSemantic Subjectの異Mapping/DB Alias、Replica/Primary、Snapshot/Cohortの実証責任を区別 | [#2627](https://github.com/hat47x/tei-platform/pull/2627)隔離Go 29件、実`DeriveOperationSurface`・実DB未実行 |
| 4. Action/実Result | View→Typed Intent→現在Rule/Grant→Request/Effect/Receipt→独立Read。先行Previewを実Effectに昇格させない | #2500/#2597/#2598/#1174、全経路接合未実施 |
| 5. Consumer展開 | 同じAccepted MeaningからStudio/AI CUI/SHO/外部Target。実読取・開示・Manifest一式公開を別観測 | #2497/#2516/#2604/#2630、実Consumer横断未実行 |

特に、**同じ表示から同じRule/Action結果を推定しない**。画面にOrder.amount=600のみ表示し、非表示の信用上限が700/500の二世界では、画面が同じでもRuleの結論が異なり得る。Ruleへの依存を理由にClient/LLMへの秘密Field取得を自動許可することもない。RDB/API、Primary/Replicaの同じID/数値Revisionが実共有Cohort・同じ更新対象を保証するわけでもない。根拠は[メタ概念設計§25](design/DATA_STRUCTURE_META_FACETS_AND_TOPOLOGY_2026-11.md)のIF-01〜10を参照する。

作業の分担は、Syntax/Meaning Role #988/#2620/#2621、Function/Rule #2506/#1129/#1729、Data Operations/Provider #2622/#2611、縦断 #2586、Effect #1174、Owner意味受理 #267 とする。**AIトークン供給増によって独立Pressureと非同期実証を拡充できても、別々の隔離ReferenceのPASSを製品の統合PASSへ合算しない**。現段階での正式Schema/DSL文法・新Executor/Policy/Host・外部Targetの採用やGate/優先順位の変更はない。

### 2026-11-02 観測閉包・実取得・開示・横断Claimの選択的再資格化

[メタ概念設計§26〜27](design/DATA_STRUCTURE_META_FACETS_AND_TOPOLOGY_2026-11.md)、[Pressure Corpus PC-157〜168](design/DATA_STRUCTURE_PRESSURE_CORPUS_2026-11.md)を、#2619のEverything is Data、#2616のContext起点Action Offer、#2516のConsumer Claim、#2582の増分観測、#2630の成果物一式の実読取と接続する。**同一Source/Activityと版を固定する実証単位の拡大**であり、既存[#1119](https://github.com/hat47x/tei-platform/issues/1119)のGate、Owner受理・merge資格を変更しない。

まず三つの異なる範囲を一枚の**受渡し比較票**で追跡する。(a) Function Display/Rule/Actionが論理的に必要とするRead Footprint、(b) Query/Providerが実際に取得したField・Row・RelationとCache/Logを含む観測可能なAccess Witness、(c) 各Consumerへ実際に送出・公開した値・Action/Tool名・診断・件数。必要な入力集合の和は物理Read最小性も外部への開示許可も示さない。さらにRowset完全性、FieldごとのValue/Observed NULL/Absent、Cohort/時点を別資格とする。

| 既存Ownerへの受渡し | 初回の反証・実測 | 明示する限界 |
| --- | --- | --- |
| #2506/#1129/#1729 | Display同じ、非公開Rule依存だけ違う二世界。Ruleの入力不足・部分取得・失効を分ける | [Draft #2624](https://github.com/hat47x/tei-platform/pull/2624)の20＋18隔離Python PASSは実TEI Rule/DBではない |
| #2622/#2611/#2387 | 同じ公開JSONでも秘密Fieldの過剰取得を含むProvider Access。異Alias・同Snapshot文字列の反例 | [Draft #2628](https://github.com/hat47x/tei-platform/pull/2628)の12隔離Python PASS、[Draft #2627](https://github.com/hat47x/tei-platform/pull/2627)の32隔離Go PASSは実OS/DBのアクセス抑制・共有Cohortを保証しない |
| #2507/#2516/#1178 | 公開Subjectが同じ二世界で秘密ActionやTool候補の存在だけ変える。低開示Consumerの候補・診断・結果を検査 | [Draft #2613](https://github.com/hat47x/tei-platform/pull/2613)の42隔離Python PASSは実Policy・Host Grantの検査ではない |
| #2582/#2600/#2516 | Rule、Provider Slice、開示Grant、Source Meaning、Artifact等の変更で**失効するClaimだけ**を再確認。未知の依存を無影響へ格上げしない | 新しい全体Claim Ledgerや資格化Engineを先行追加しない。実更新・再取得・再送出の因果証拠は別 |
| #2630/#2604/#2515 | 同じMeaning/ManifestでもConsumerが必要なArtifact一式を実読取できたか、どの公開Audience・外部Harnessへ送出したかを確認 | Manifest自己整合は期待Member充足・同時公開・実受信・機密出力不在を証明しない |
| #2619/#2620/#1124 | 未知Role／Pluginの標準データを保持し、read-onlyの説明・Source参照を維持。編集資格・Actionと区別 | [Draft #2631](https://github.com/hat47x/tei-platform/pull/2631)の16限定JS assertionsは実Studio/Owner受理ではない |

**横断Claimの選択的再資格化**は、TEI Query/Rule→SEI判断→TAI/外部HarnessのSkill/Tool→SHO文書利用→EKI配送などの「各部分のPASS」を足すこととは異なる。元のSubject/Activity、Source/Meaning/Rule/Provider/Skill/Consumer/Authority/Artifactの版と、必要な前提、現時点の用途・開示条件を、同じ実施活動へ明示的に結び付けて比較する。Skillの宣言・配布・Tool到達・実行資格・実結果は別。履歴として正しい過去のReceiptを、現在の公開許可へ自動変換しない。

最初の反証を**同じOrder/信用上限の受理済みRule候補**で作り、(1)データ1件の訂正、(2)Rule版変更、(3)Provider Alias/実Snapshot変更、(4)Consumerの開示Grant失効、(5)SHO用Artifactの部分公開、(6)未知Role追加に対して、それぞれどのClaim/表示/操作候補だけ再評価・保留すべきか説明する。既存#2586が実TEI縦断、#2582が依存影響、#2516がConsumer別の資格、#2600がBuild再利用、#1119が統合Gateを所有する。

**段階**：R0=read-only依存・版照合／R1=既存合成Referenceへ固定負例追加／R2=実TEI Function/Rule/Providerの一試行／R3=実Consumer・現行Policy/Tool/Host・Receipt／R4=フル再評価と選択的再評価の一致、誤維持、不要失効、再評価量を計測。Owner意味受理・実Provider・実Studio/SHO・外部Harness・本番Effectと全実行経路の証明は、各担当の観測が揃うまで**NOT RUN**。この追加計画はGateの合格・正式仕様採用・main反映を意味しない。


### 2026-11-02 非出力Query依存からIncremental・Disclosure・Current Grantまで接合

[メタ概念設計§28](design/DATA_STRUCTURE_META_FACETS_AND_TOPOLOGY_2026-11.md)で、同じ出力値を生成するExpressionでも、行の**成立/不在・JOIN多重度・選択順位/Limit・Ruleの評価条件**が変化するとResult Slice/Claimが変わり得ることを明示した。これを従来の#2582 Reactive Data Context、#2600 Incremental Build、#2387 Query Explain、#2514独立Oracleと接続する。SQLの`SELECT id WHERE secret_score > 50 ORDER BY secret_rank LIMIT 1`では、`id`の値Lineageが同一でも秘密Predicateや非表示順位が出力する行を変え得る。

| 接合順 | 既存Ownerと実際に必要な観測 | 不十分な根拠／判定保留 |
| --- | --- | --- |
| Q0 静的Query依存 | #2387／[Go Characterization #2634](https://github.com/hat47x/tei-platform/pull/2634)・[#2635](https://github.com/hat47x/tei-platform/pull/2635)：Output Value Inputs、WHERE/EXISTS、ORDER/LIMIT、Join/Bag、Source/Operator版とCoverageを別列 | OutputField ID/Value lineage一致だけ、`Lowering=full`の入力値だけでは説明の完全性を証明しない。重複Draftは同じOracleとして加算しない |
| Q1 実Analyzer連結 | #2387/#2622：**実SQLGlot Worker→Go DerivedIR**と、SQL Source bytes/Schema/Parameter/Query bindingを照合 | 手製`analyzer.Result`でのGo test PASSは実WorkerのColumnRefs網羅を保証しない |
| Q2 実Query/Context | #2506/#2624/#2611：実Provider Request/Read Witness、RowsetとField完全性、Snapshot issuer・Cohort・非表示Rule入力を固定 | 見えている値・列の一致からDBの不要Readや全件性、RLSの同Scopeを主張しない |
| Q3 送出・権限 | #2516/#1178/#1769と[限定D2修正 #2637](https://github.com/hat47x/tei-platform/pull/2637)：preflight/post-loadの実Read Grant、返却後のConsumer向け開示、現在Policyと非公開条件/Tool | 二地点検査は失効/復活ABAや返却後のEgressまで一括保証しない |
| Q4 再実行/Oracle/性能 | [F0/F1/S1/S2計画 #2636](https://github.com/hat47x/tei-platform/pull/2636)のBaseline/Changed Full/Selective/Negative対照、#2514の独立Expected、#2582/#2600の失効選択と待ち時間 | FullとSelectiveの偶然のByte一致や同一Generator由来のWantでは、一般的な安全skipや意味受理を保証しない |

**順序と資格**：先に#2387が二つのQX-27 Go Characterizationを同一Ownerの負例へ整理し、まだ型付きRow-membership edgeやPredicate/Order completenessの製品契約を採用しない。次にQ1〜Q3で**解析に必要な非出力Fieldを保持することと、それを外部へ開示できること**を分ける。#2637の現在Grant二地点再確認は既存D2拒否を維持した限定修正候補として扱い、実DB/Host/OSの継続的強制は別資格。Q4ではF1とS1の**結果同値・選択的再計算の正しさ・再利用Evidence・速度改善を別々に検査**する。

最初に追加する負例はLQ-01〜12（§28）。再実行の対象Source/Rule/Query/Provider/Skill/Artifact変更はそれぞれOwnerへ帰属させ、同じ業務Activity/版・Context・Consumerでの実観測が揃うまでは統合PASSを宣言しない。**新しいQuery Engine、グローバル依存Graph、独自Host、別のSemantic SSOT、Gateは追加しない**。Q1〜Q4の実一気通貫／外部Consumer／Owner意味受理は**NOT RUN**。

### 2026-11-02 実mainのCLIを起点とするSource→Operation→Artifact→Reader証拠の最小縦断

従来のEverything is Data横断計画（本節前半／[Draft #2629](https://github.com/hat47x/tei-platform/pull/2629)）は、Syntax→Role→Context→Provider→Consumerの理想的な受渡し条件を示す。一方、**2026-11-02に[#2587](https://github.com/hat47x/tei-platform/pull/2587)がmainへ統合**され、`cmd/tei`に`operations`、`artifacts`（`-out`あり）が実在するため、まず**存在する製品CLIだけを使った限定的な縦切り**から観測を開始できる。実装状況はmainの[`docs/status.md`](../docs/status.md)、操作の限定実証は[`plan/evidence/SURFACE_AND_ARTIFACT_VERIFICATION_2026-11.md`](evidence/SURFACE_AND_ARTIFACT_VERIFICATION_2026-11.md)を参照する。

**固定する初期対象**：main上の`reference/standard-crud/workspace`、Application `ledger`（`applications/ledger/application.tei`に宣言）、既存の合成データと`missing-semantics`／`missing-input`／`conflict`の負例。まずGit SHAと実buildされた`tei version -json`のBinary情報、Workspaceの実Bytes・`sourceIndexFingerprint`・`worldFingerprint`・`modelRevision`の各生成元を固定する。***名称やRevisionを同じ文字列に寄せて比較しない***。具体的な値を取れていない場合は空欄・未確立とし、後から値を創作しない。

実装上、`cmd/tei/main.go`の`operations`・`artifacts`経路はコマンド共通の`loadWorld`を通った後に、それぞれ`SurfaceWorkspace`・`RenderWorkspace`を呼ぶ。**同一コマンド内においてさえ実装が何回どのBytesを読むか／そのSourceが同じかは別途追跡が必要**。複数CLIを順に呼ぶだけで「同じLoad Snapshotから全出力が生成された」とは主張しない。

最小の実コマンド候補（**下記をこのPRで実行したという意味ではない**。実行時は同じ固定checkout・Builder/Go版・実Workspaceを証拠票へ記録する）：

```sh
go build -o /tmp/tei-eid-verify ./cmd/tei
/tmp/tei-eid-verify version -json
/tmp/tei-eid-verify validate -workspace reference/standard-crud/workspace -json
/tmp/tei-eid-verify operations -workspace reference/standard-crud/workspace -application ledger
/tmp/tei-eid-verify artifacts -workspace reference/standard-crud/workspace -application ledger
# -out は別試行：作業用の新しい空ディレクトリを明示し、出力前後のファイル名・digestを記録
```

| 試行段階 | 具体的に観測するもの | 成立しない主張／追加Owner |
| --- | --- | --- |
| C0 CLI/Source read | 実Binary・args・exit/Warning・対象Workspace bytes/Source Index・World版、Loadの回数と時点 | `version -json`は真正な署名・Owner受理でない。実ロード一致は#1163/#2605へ |
| C1 Operation Surface | `ledger`の導出`list/read/create/update`候補・Gaps、Data/Mapping/Target/Grant未解決を明示 | `operations`のJSON表示だけで実Provider Read/Write／SQL/Rule意味受理はしない（#1127/#2611/#2598） |
| C2 ArtifactSet | 表・機能・操作面・Manifest・Findings、Expected Memberと実Set、Generator/Templateの版、独立CheckSet | `Manifest.Complete`・同InputRevisionがConsumer期待集合／実ファイル読取済みを保証しない（#2482/#2630） |
| C3 ファイル公開/読取 | `-out`へ実出力、同一世代の各Path/bytes/digestとManifest、過去の残留ファイル、障害注入、独立Readerが実際に読んだMember | 現`writeArtifactFiles`の順次`os.WriteFile`は一式のAtomic Publish保証でない。#2604/#2630/#2633へ |
| C4 Source署名/受理 | [#2642](https://github.com/hat47x/tei-platform/pull/2642)統合後の`reference-host-acceptance sign`は`BuildSourceIndexFromLoad`で**Load時に捕捉したBytes**のみを用い、import宣言Workspaceを`WORKSPACE_IMPORTS_NOT_BOUND`で拒否 | これは`sign`内のSourceの限定保証。**別Invocationの`operations/artifacts`が同じSnapshot／Owner受理／実Target認可**とは限らない（#267/#1163） |
| C5 Meaning/Role/Client | [#2639](https://github.com/hat47x/tei-platform/pull/2639)既存Go Language Plugin Referenceと[#2638](https://github.com/hat47x/tei-platform/pull/2638)のRole bootstrap（いずれもDraft）からSource/候補/Pin/Meaningを照合し、Studio/SHO/外部Consumerで実Read | Go Referenceや合成RoleのPASSは正式TEI DSL、実Studio/SHO、Owner受理、現在の開示/Host EffectのPASSでない（#2619/#2620/#2516） |

**今回特に検証する負例**：①`operations`実行後、`artifacts`実行前にSourceが変更された場合、同じアプリ名でも出力世代が混在し得るか。②あるCLI内で`loadWorld`と`SurfaceWorkspace/RenderWorkspace`の間に管理対象Sourceが変わった場合、出力がどの捕捉版に属すると説明できるか。③`-out`で前の世代のファイルが残り、現Manifestは自己整合でもConsumerが残留を誤読するか。④Role依存PluginのEager Pinだけ改版してもSource bytes同じ場合、現在Meaning候補/Consumerの資格が変わるか。⑤実Read Grant/Consumer Disclosureだけ失効し、生成物は同じ場合、現在利用可能と誤表示しないか。**この5件は未実行の試験候補**。実データ・秘密・本番署名鍵は用いず、合成Workspaceと一時領域に限定する。C4において署名済みと扱う場合もOwner本人の判定を模擬Fixtureへ置換しない。

**優先度・重複解消**：#2629が横断Handoffパケット、#2587が現在のmain実装、#2605が実CLI Product Skill、#2639/#2638がLanguage/Role、#2604/#2630がArtifact公開・Consumer実読取、#1163/#267がSource・受理、#2514が独立Oracle、#2586が実TEI一気通貫を所有する。本計画は**既存CLIによる実在部分をP0として実測し、欠けるSource Snapshot/実Read/Role/Current Authority/Effectを各Ownerへ返すための計測手順**であり、新Host/統合Runner/中央Registryを増設する提案ではない。

**資格**：mainに操作面・生成CLIが存在することは確認したが、C0〜C5の固定checkout一連試験・同一Snapshotの証明・Consumer actual read・Owner意味受理は**NOT RUN**。PR #2638/#2639等の限定Reference CIと[#2642](https://github.com/hat47x/tei-platform/pull/2642)の統合成功を、C0〜C5の合格へ繰り上げない。

<!-- tei-masterplan-data-form-observation-integrity-20261106 -->
### 2026-11-06 データ様式：構造保存と実Read/継続の整合を同じ検収経路へ

構造分類・Projectionの一般契約は[#2497の設計§22](design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)へ集約する。#2495のFacet、#2497の保存・損失、#2619のEverything is Data、#2506のFunction Context、#2387/#2622のQuery/実Provider、#2516のDisclosureを別Ownerのまま接続し、新Core Schema/第二Query Engine/Host/Cursor Registry/受入Gateを設けない。

**最短の次工程**：既存#2586 HJ-SV-01〜05（同一Query IDでもPage間SQL Realization変更で混合するSQLite合成11/11）を固定Expectedとして、(1) #2611/#2622/#2387の実TEI Realization選択→解析実Input、(2) #2506/#2627/#2586の同一Function・実Provider Page1/2＋Cohort＋完了証拠、(3) #2516の実Consumer現在Grant/非干渉へ順番に接続する。初回は一Provider/二SQL、read-only。二DB全面互換、全Graph/Stream対応、Writer、Self-build、Studio/SHO全機能を初回Hard dependencyにしない。

**停止条件**：Query ID/SQL bytes/短い最終Page/Cache hit/可視0件を、選択・実Read・全体不存在・公開資格の共通PASSにしない。固定済み専用Asset欠落時の無音Fallback、異RealizationのCursor自動連結、実Readなしでのcomplete、非公開候補差の外部露出は禁止。依存変更はClaim相対で局所再資格化する。Graph/時系列/文書/Measureの適用例は設計候補で実Provider/外部Target未実施。実TEI Selector・RLS・Owner正式受理はNOT RUN。既存#1119 A/B/C/D Gate・Owner判断を変更せず、#2018には誤成功率/手順/性能/重複実装を比較対象として返す。

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
