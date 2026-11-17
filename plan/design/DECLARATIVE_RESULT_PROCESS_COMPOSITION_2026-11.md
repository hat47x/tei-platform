# 宣言的な成果物設計とプロセス合成：データ・計算・フローの横断モデル

- **状態**：2026-11-15の横断調査を踏まえた設計提案。正式DSL・最適化規則・実行方式・製品受入は未採用。
- **位置付け**：既存の[ETL・Batch構成設計](ETL_BATCH_FUNCTION_COMPOSITION_DESIGN_2026-11.md)および[ノードと軽量サービスの構成設計](SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)を拡張し、データマート、帳票、締め、反復計算、移行、簡易API、イベント駆動処理を同じモデル基盤に載せるための判断材料。ETL専用の第二DSL／第二モデル／第二実行Hostは作らない。
- **既存Owner**：#1130（Batch/帳票）、#2622（Data Operations）、#1124/#2497（標準Graph/表現）、#4497（Trigger Binding）、#1129/#1174（Effect/回復）、#951（Plugin選択）、#1180/#2514（独立Expected）、#2831/#4583（Studioセルフビルド）。既存の担当範囲を再実装しない。
- **優先順序**：現行P0（正式Source→生成Go→通常CLI実Provider、およびStudio Stage1→Stage2実Windows）を妨げず、まずread-only意味照合と純粋変換の検証から進む。

## 1. 結論：処理を描く前に「必要な結果」を記述する

一般的な処理フローは、入力と操作をノードでつなぎ、結果を得るための**実現方法**を表す。集計・分析・業務締めでは、先に「何を、どの粒度・時点・条件・品質で必要とするか」を示し、複数の成果物を横断して共通計算や適切な中間結果を導出する必要がある。

この二つは対立しない。TEIは次を区別して一つのSourceに接続する。

1. **成果物要求（Desired Result）**：必要なデータ、表、レポート、API応答、業務状態。値だけでなく粒度・時点・完全性・更新・品質・利用者を宣言。
2. **意味モデル（Accepted Meaning）**：Data、Attribute、Relation、分類、数量、単位、Rule、Function、Processの意味。業務判断を自動的に受理しない。
3. **論理計算・依存関係（Logical Composition）**：式、抽出、Join、集計、変換、状態遷移、複数出力間の依存。**何を計算するか**を記す。
4. **導出実行計画（Physical Plan）**：処理融合、SQL押込み、区分実行、中間結果の共有・再利用、並列化、TargetとProvider配置。**どう計算するか**の候補で、意味モデルの正本ではない。
5. **起動と実行観測（Invocation / Attempt / Result）**：手動・定期・イベント・API・外部委任、現在のGrant、実際のSource/Artifact、実Effect、ReceiptとConsumer受領。
6. **表示・編集投影（Representation）**：Markdown表、数式、データフロー、ノード、依存Graph、時間線、SQL、Trace。表示ノードID≠意味ID≠実行Step ID。

「一つの画面に全てを描く」ことは統合ではない。構成要素間の型付き関係と、用途に応じた可逆・非可逆な投影を保持することが統合である。

## 2. 標準データ構造・標準表現型・標準プロセスとの接続

新しい万能 `Flow` 型に全要素を押し込めない。既存Data/Relation/Function/RunPlan/Graph/Queryの意味を維持し、必要な情報を**役割／関係／制約**として加える。

| 層 | 標準化する対象 | 最低限保持すべき情報 | 既存の接続先 |
| --- | --- | --- | --- |
| 値と構造 | Scalar / Record / Relation / Table / Tree / Graph / Event / Stream / Window / N次元 | identity、型、単位、NULL・欠損、精度、出典 | `semantic.Data`/`Attribute`/`Relation`、標準Data |
| 計算対象 | DataSet、Relation Set、時間窓、集合の区分、成果物世代 | 対象範囲、業務キー、取得完全性、順序、Source/Revision | `RelationQuery`、Query IR、Period |
| 演算と接続 | Operator、Function、typed input/output port、dependency edge | 入出力型、粒度、必要時点、純粋性、決定性、Effect、失敗 | Capability、Validation、RunPlan、Plugin Contract |
| プロセス | 計算依存、制御依存、イベント因果、時間シフト、状態遷移 | Edge種類、Guard、繰返し境界、並列安全性 | Process/Graph、RunPlan、Trigger Binding |
| 実現 | Physical Plan、Target Artifact、Data Provider、Materialization | 版、配置、最適化の前提、失効条件、外部Effect | Query Translator、Generator、Build/Artifact、Provider |
| 観測 | Execution Trace、Lineage、品質・回復証跡 | Run/Attempt、入力世代、独立Expected、Receipt、Consumer | runlog、artifact、failure、Evidence |

**Port契約**は「型」だけでは足りない。少なくとも `shape / schema / grain / key / cardinality / time / completeness / ordering / effect / provenance` を照合する。全てを一律に必須にせず、Operatorが必要とする条件を明示する。未提供は無条件の既定値にせず、`unknown`/`unsupported`とする。

## 3. 特に重要な三つの独立性

### 3.1 処理管理単位・出力粒度・計算単位を分離

- **出力粒度（Grain）**：「一行が何を表すか」。例：伝票、伝票行、店舗×営業日。キー、時間、版、重複規則を持つ。
- **計算単位（Evaluation Scope）**：一つの式を評価する範囲。例：伝票行ごとの金額。
- **集約単位（Aggregation Scope）**：計算値をまとめる範囲。例：伝票の課税額を店舗へ合計。
- **実行区分（Partition / Work Unit）**：並列・再開・配布の単位。計算や集計の意味と独立。
- **業務プロセス境界（Process / Function）**：何を達成する処理か。出力行のキーと同一視しない。

伝票行から伝票へ集約した後に税額を丸め、伝票税額を店舗へ合計する処理を、行税額を店舗へ直接合計する処理へ変更してはならない。Group Keyと丸め位置は意味契約である。

### 3.2 分類階層と集計の意味を分離

分類は単純な親子ツリーに限らない。期間付き多対多、重複包含、複数の分類体系、同名異ID、変更履歴を持ち得る。分類対応表はRelationであり、表示階層はその一つの投影とする。

複数の分類に同じ明細が属する場合、各分類への配賦・重複計上・排他化のいずれかを業務規則として要求する。数値が合うよう自動的に配賦を決めない。多次元集計では必要なGroup Key集合を明示し、全組合せの無制限な自動生成を避ける。特に階層と多重所属を含む場合、加法性・重複・上位合計の定義を確認する。

### 3.3 データ依存・制御依存・時間依存を分離

- **Data edge**：値・レコード・集合の入力元。
- **Control edge**：前提条件、成功・失敗時の分岐、起動条件。
- **Event edge**：発生と配送、原因となったOperation。
- **Temporal edge**：前営業日の成果物を翌日に参照、前期残高を次期に繰越。
- **State edge**：状態遷移、復旧、補償。
- **Provenance edge**：結果に寄与した値・条件・定義・取得元。

同日の無条件な依存循環は原則拒否する。一方、`state(t+1) = f(state(t), input(t+1))` のような時間が進む循環は、初期状態・時間粒度・終了条件を持つ反復として扱う。収束反復はさらに収束基準・上限・丸め規則を要求する。図上の閉路を全て同じ実行上の循環とみなさない。

## 4. 成果物中心の要求と、ノードによるフローの棲み分け

| 問い | 第一の記述・表示 | 同じSourceからの別投影 |
| --- | --- | --- |
| 月別・部門別の売上表を作りたい | 出力粒度・指標・期間・品質を表で宣言 | 計算依存Graph、SQL、実行DAG |
| CSV→正規化→通知をつなぎたい | 型付きNode/PortとProcess Flow | Function、Data、Result、API構成 |
| 定時・イベント・画面から同じ業務を呼びたい | Function＋Trigger Binding | 各入口の起動図、実行Trace |
| 前日残高から当日残高を作りたい | 時点を伴う状態遷移・繰越規則 | 日次実行DAG、復旧票 |
| 複数の帳票を共通計算で生成したい | 複数のDesired Resultと共有する意味 | 合成計画、費用比較、中間成果物 |
| APIサービスを公開したい | Function＋入出力・Auth/Effect契約 | HTTP/Node Flow、Target別生成 |

ノード方式は**構成可能なOperatorの編集・観測UI**として重視する。ただし、ノードが自由なメッセージを接続できることだけで型・業務意味・許可・実Effectの条件が満たされたとみなさない。サブフローは既存Function/Processの再利用可能な投影、ノード部品はOperator/Plugin Contractの投影であり、独立した第二の計算モデルではない。

Studioは少なくとも「結果定義表」「データ・依存Graph」「手続き／ノードFlow」「出力プレビュー」「実行Trace」を同じ意味IDへ往復ナビゲートできるようにする。描画上の位置・色・ノードIDは業務識別子と分け、損失のある投影からの逆編集は確認を要求する。

## 5. 実行計画の導出と最適化の成立条件

最適化は変換の意味を変更しない範囲に限る。意味、許可、SourceとPluginの版、実際のTarget能力が明らかになって初めて物理計画候補を選ぶ。

| 最適化候補 | 成立する前提 | 棄却する例 |
| --- | --- | --- |
| 入力走査の共有 | 同じSnapshot、取得完全性、開示資格、評価時点 | 使用者ごとに異なる可視範囲 |
| Map/Filter/Project融合 | 純粋・決定的、例外とNULL意味を保存 | 条件評価に外部Effectや順序依存 |
| SQLへの押込み | Providerが演算の型・NULL・丸め・照合順序を保存 | TargetのDecimal/日付意味が異なる |
| 部分集計・並列Merge | 結合可能な正確な中間状態と完全被覆 | `AVG`の平均同士の平均、Distinctの無根拠合算 |
| 複数出力の共通化 | 入力の範囲・版・集計粒度、各Consumerの契約が一致 | 一方だけ異なる締め時点、非加法指標 |
| 中間結果のMaterialize | コスト・再利用・鮮度・保管期限・権限を明示 | 古い版や秘密データの不正再利用 |
| 差分・増分 | 変更・削除・遅延・順序・復旧可能性の証拠 | Watermarkだけで完全差分と推定 |
| 分散/並列配置 | Key分割、秩序、fencing、Source/Targetの特性 | 非冪等作用の並列重複 |

重要なのは「常に一つの物理プロセスへまとめる」ことではない。共通化にはメモリ、障害波及、権限分離、再実行範囲などの費用がある。実装するのは**候補計画と棄却理由の比較**であり、最適解を無条件に主張しない。

## 6. データマート・バッチ固有で残すべき契約

- **有効時点と観測時点**：業務日、会計期、データ発生時刻、取得時刻、訂正時刻、分類の有効期間を混同しない。
- **新規構築と再計算**：作成・バックフィル・差分更新・遡及訂正・期間締めを別Operationとして扱う。
- **成果物の公開**：生成済み、検証済み、公開済み、消費者読取済み、業務確定済みは別。
- **復旧**：Checkpointは位置であり、安全な再開の証拠ではない。Source/Meaning/Artifact/Provider、処理済み範囲とfenceを照合する。外部Commit不明時は照会優先で盲目的に再送しない。
- **影響分析**：値を直接参照する列だけでなく、Filter条件、Join条件、分類、集合の完全性、Triggerとカレンダー、出力形式も依存として追う。制御条件にのみ使う列をLineageから落とさない。
- **再現性**：World/Model/Artifact revision、Target/Operator版、入力Snapshot/範囲とデータ実値を別に記録する。Byte一致は業務意味や権限の再受理ではない。

## 7. 代表ケース：複数粒度の売上成果物

入力は伝票行。出力要求は「伝票ごとの税額」「店舗×日付の売上」「地域×月の売上」「部門別クロス集計」の四つを想定する。原則として四つの別バッチを最初から描かない。

1. Data/Relationで明細・伝票・店舗・地域と、必要なカテゴリ体系・期間を定義する。
2. 各出力のGrain、数量・金額・税額の式、丸め位置、期待完全性、使用する分類版を別々に宣言する。
3. Logical Compositionで、行金額→伝票金額→伝票税額という段階を保持する。店舗・地域に跨る多対多の分類は別検証する。
4. Plannerは共通入力の一度読み・複数出力、SQL押込み、段階的Materializeなどを比較し、`unsupported`を隠さない。
5. 固定Fixtureで別々に計算した独立Expectedと全出力を照合した後、物理計画の融合可否を判断する。

以下は**説明用の表であり、新しいDSL構文の提案ではない**。

| Result | 粒度 | 参照期間 | 計算上の必須境界 | 出力 |
| --- | --- | --- | --- | --- |
| 伝票税額 | 伝票 | 会計日 | 行→伝票、伝票で丸め | 表 |
| 店舗日次 | 店舗×営業日 | 営業日 | 伝票税額を合算 | 表/API |
| 地域月次 | 地域×月 | 会計月 | 有効期間付き所属を照合 | マート |
| 部門クロス | 部門×商品区分 | 指定期間 | 分類の多重所属と加法性 | 多次元表 |

## 8. 実装・検収の順序

| 段階 | 実施内容 | 独立Expected／停止条件 |
| --- | --- | --- |
| C0 | 現行Data/RunPlan/Query/Graphの読取専用Fit/Gap。既存ETL縦断と重複実装しない | 概念が存在するだけで製品接続済みと扱わない |
| C1 | **二つの出力粒度、一つの入力、共有候補**の小さな正式Source。表とGraph投影 | 同名異ID、NULL/空集合、Filter-only依存、複数出力の粒度不一致 |
| C2 | 既存Operator/Query IRを使う論理計算のTrace。一つの出力が二つの計算経路を持つ例 | 金額・税額の丸め位置を変えるとExpected不一致。未知関数はunsupported |
| C3 | 融合する計画／分離する計画の**二候補比較**。Target未依存の等価性試験 | AVG/Distinct、カテゴリ重複、部分取得、版違いで共有を拒否 |
| C4 | 実Query Generator／Go生成→実Builder Byte消費→通常CLI→実Provider／独立Consumer | Mock-only、Runner未接続、SQL意味差、成果物と実結果の混同で止める |
| C5 | 日付シフトの反復と再計算・Checkpoint/Receipt検査を既存ownerへ接続 | 当日ゼロ遅延閉路、古いSourceで再開、Commit unknown再送を拒否 |
| C6 | Studioの結果表・Graph・Node・Trace共通選択、Plugin部品再利用 | 表示ID≠意味ID、後着応答、Graphからの損失ある逆編集を検査 |

**最初の追加差分はC0→C1だけ**とする。C2以降をP0の新しい必須ゲートにはしない。検証用のプログラムは既存型で生成できるかを優先し、単にGo関数へ全処理を委任するだけの「セルフビルド」を成果と数えない。

## 9. Ownerへの引渡し

- #1130/#4410：一つの入力から二つの独立した出力を得る最小Corpusと、実行二入口の同値照合。
- #2622：Port契約のgrain・complete・time・orderingの不足／Providerで保存できない意味の一覧。
- #1124/#2497：意味Graph・表・ノード描画のID分離、制御条件依存、時間シフトEdge。
- #4497：同じFunctionにmanual/schedule/eventを結ぶだけのread-only Trigger候補。
- #1129/#1174：Materializeと業務Commit、再実行とEffect unknownの区別。
- #1180/#2514：C1〜C5の負例と独立Expected、製品経路の実Byte/実Consumer確認。
- #2831/#4583：C6は既存Studio Stage1→Stage2の後続。先行Hard Gateにしない。

本稿は設計案であり、正式語彙の追加や推奨Physical Planの製品採用を宣言しない。採否は代表Corpusの比較結果、Owner判断、ADRで確定する。


## 10. 成果物要求・ノード・運用の共通計算モデル（2026-11-15追補）

### 10.1 「何を得たいか」「何を計算するか」「どう実行するか」を別々に編集する

新たなデータ処理の要求が生まれたとき、最初に作業ノードを並べなくてもよいようにする。ただし、すべてのProcessを結果定義から自動で導けると考えるのも誤りである。人による承認、外部通知、例外対応などは、結果の値だけから実行順序や副作用を導けない。

そこで、同じ標準データとOperationを少なくとも次の四つの観点で扱う。

| 観点 | 利用者が決めること | 導出・編集できること | 正本と混同しないもの |
| --- | --- | --- | --- |
| **結果要求** | 必要な表・帳票・API応答・状態、列と粒度、対象期間、品質、利用範囲 | 達成に必要なData/Relation/Ruleの候補を引く | Resultを作る具体的な実行順序 |
| **計算の意味** | 参照関係、式、分類、計算単位、集約単位、NULL/精度、時間、業務規則 | 型付きOperatorの合成、影響分析、妥当な計算依存Graph | 特定DBや中間ファイルの配置 |
| **操作・制御** | 分岐、反復、時間差、確認、外部Effect、実行の前提 | Node/Port/Edgeの配線、Subprocess、起動・エラー経路 | データ依存だけから推測した副作用順序 |
| **物理実行** | 実Target、Provider、予算、保管期間、権限、回復可能性 | SQL/Go/外部Hostの生成、融合・分離・並列化・Materialize候補 | 受理済み業務意味と実行許可 |

これらを接続する基本要素は既存のData、Attribute、Relation、Function/Capability、Operator、RunPlan、Period、Plugin Contractとする。`Desired Result`、`Evaluation Scope`、`Materialization`などは当面**既存要素が担う意味上の役割**であり、新たなCore型や予約語を直ちに追加するものではない。

### 10.2 計算の粒度を実行区分から切り離す

同じ「集計」でも次の五つは異なる。

- **Source Grain**：入力の一件を何で特定するか。例：伝票行。
- **Evaluation Scope**：式をどの単位で評価するか。例：伝票行ごとの金額。
- **Aggregation Scope**：評価結果をどの単位にまとめるか。例：伝票ごとの課税対象金額。
- **Result Grain**：成果物の一件が何を表すか。例：店舗×営業日。
- **Execution Partition**：並列化・再実行・転送のための区分。例：日付範囲やハッシュ区間。

これらを独立させると、出力粒度だけでは復元できない計算の段階が保てる。たとえば、各伝票行が15円の二行に対して税率10%・四捨五入を適用すると、行ごとの丸め後合計は4円だが、伝票合計30円に対する丸めは3円となる。丸める単位を変えた計画は、最終的なGroup Keyが同じでも同値ではない。

分類・次元については、カテゴリ同士の関係を単なるTreeに固定せず、Sourceに束縛されたRelationとして扱う。同一商品が二つの分類に所属する場合、排他的分類、二重計上、配賦、未確定のどれかを選ぶのは業務側である。階層の表示形だけで上位合計を確定しない。金額・比率・在庫残高等は加法性が異なるため、集計可能な次元を演算・指標ごとに確認する。

複数キーの全組合せを自動生成することも既定とはしない。キーがn個なら非空部分集合は`2^n - 1`通りとなり、メモリ・出力数・実行時間が増える。必要な組合せだけをResult要求から選び、探索の予算上限を設ける。

### 10.3 依存Graphは値の経路だけでは完結しない

Data/Relation/OperatorをGraphへ投影するときは、**何のために参照されているか**を辺のRoleとして保持する。値を生み出す演算だけでなく、抽出条件、Join条件、業務日付、適用Rule、分類版、品質判定、利用者の可視範囲も成果物に影響する。

| 依存のRole候補 | 例 | 省略すると起きる問題 |
| --- | --- | --- |
| Value | 入力金額→算出金額 | 値の生成元が追えない |
| Condition | 取引状態→採用・除外の判定 | 条件列が変わっても再計算対象にならない |
| Relation / Join | 明細→伝票→店舗 | 多重度や欠損を保持できない |
| Classification | 商品→分類体系・適用期間 | 旧分類で作った出力を新分類へ取り違える |
| Temporal | 前営業日の残高→当日残高 | 有効な時間シフトと無条件循環を混同する |
| Control / Trigger | 締め済み→公表処理の起動 | Data接続だけから不正な起動を推測する |
| Effect / Recovery | 外部書込→Receipt確認 | 実行終了を業務確定に取り違える |
| Provenance / Authority | Input Snapshot・開示Scope→出力 | 別Tenant/旧版の中間結果を不正に再利用する |

Roleは新しいGraph種別の乱立ではなく、既存標準GraphのIdentity/方向/多重性/関係属性へ載せる候補とする。Node図の線ID、Relation ID、Operation出現ID、データ項目ID、実Step/Attempt IDを一つにまとめない。表示上は同じ矢印に見えても、意味上の依存と物理I/Oは別の投影である。

特に、`data(t) -> data(t+1)`という依存と`data(t) -> data(t)`の循環は区別する。前者は初期状態、基準日、業務カレンダー、期間締め、訂正を明示した反復として成立し得る。後者を無条件に実行可能なDAGへ変換してはならない。収束計算にはさらに停止条件、上限回数、誤差・丸め規則を必要とする。

### 10.4 Operatorが持つ性質を、最適化の根拠として検査する

同じ式やOperationでも、結合・交換・融合・分割に耐えるとは限らない。Operator/Plugin Contractに記載する候補の性質は次のとおり。

| 性質 | 使える最適化 | 反例／停止条件 |
| --- | --- | --- |
| pure / deterministic | 計算再利用、融合、並列化 | 現在時刻、乱数、外部書込、版未固定Lookup |
| associative / identity | 部分集計後の結合 | 途中丸め、浮動小数点の再結合、状態依存 |
| commutative / ordered | 処理順の変更・並列化 | first/last、累積計算、順序付きWindow |
| distributive / monotone | Filter・集計の押込み、増分更新 | 非加法指標、Delete、分類・条件の変更 |
| null/precision/collation semantics | DB側Query Pushdown | ProviderによるNULL、数値桁、時刻、照合順の差 |
| idempotent / compensatable | 安全な再入・復旧 | 外部Commit unknown、ACK消失、二重送信 |

性質は名前から推定してはならない。Provider/Targetによって同値範囲が変わるため、能力宣言・版・独立Expectedで検証できた組合せに限りRewriteする。未知の性質は`unsupported`または未確認と記し、最適化成功を宣言しない。専用の全自動Optimizerを先行実装せず、**融合法・分離法の候補と採用・棄却理由を比較するread-only Planner**から始める。

### 10.5 Node Flowは同じOperationの「配置図」であり、結果要求の代用品ではない

ノードは標準OperationをFlowの中で利用した**出現**である。Node/Port/Edgeは簡易API、Webhook、メッセージ処理、Plugin間の配線、データ処理の途中経路の編集に向く。一方、月次マート、帳票、複数粒度の集計のように結果の定義が先にある作業は、結果表・計算表・分類図から着手した方が余計な処理ノードを要求しない。

二つの編集経路を次のように整合させる。

1. **結果定義表→計算Graph→実行候補**：ユーザーが出力粒度、指標、期間を宣言し、既存Data/Ruleから必要な演算と複数の計画候補を導く。
2. **Node/Portの配線→Operation Graph→結果・サービス契約**：ユーザーが入出力・エラー・外部サービスを接続し、型・粒度・効果を検査してFunctionに結ぶ。

この二経路が共有するのは、型付きData、Operation、Relation、Process、Plugin、およびSource位置とIdentityである。結果定義表、Graphviz、ノード図、関数チェーン、SQL、Traceのいずれも独立した意味正本にしない。変換前後で保持できない情報がある投影からの逆編集は、Lossを表示し、明示確認またはread-onlyへ留める。

Node Flowの`Retry`や`Delay`などは描画ノードとして存在しても、意味上はTrigger/Process/OperationのPolicyであり得る。Node専用Runtimeに実体を閉じ込めず、CLI、Web/Desktop、外部Hostと同じ意味を用いる。

### 10.6 生成物の共有・公開・復旧は「結果が等しい」だけでは成立しない

同じ入力を複数のResultが使う場合でも、共通計算を共有してよいかはSnapshot、業務評価時点、現在のScope、分類・Rule版、完全性、精度・丸め、データ公開範囲とConsumerで異なる。物理実行時に各Consumerの資格を検査し、共通中間結果を不正な権限拡張に利用しない。

Materialization候補には少なくとも入力Revision、意味Revision、Operator/Target版、出力Grain、対象期間、保管期限、再計算条件、公開・可視範囲を紐づける。成果物の`generated`、独立に`verified`、正式に`published`、Consumerが`observed`した状態は別に持つ。Sourceの版が同一でも実データの値が更新され得るので、Source fingerprint、データSnapshot、実Run、出力Generationを混同しない。

データの事前保存に基づく復旧、Checkpointからの純粋変換再開、DBのCommit/Transaction、外部API送信の補償は異なる操作である。失敗直前の旧データがあっても外部Effectの取消しは保証できない。現在のGrant、元Operation/Attempt、Source/Artifact、実Provider Receiptを照合し、Commitの成否不明時は照会優先で無条件再送しない。

### 10.7 既存C0〜C6とN0〜N4に接続する反例

新たな第二の実装計画や独立Expected正本は作らず、下記を既存[#1130](https://github.com/hat47x/tei-platform/issues/1130)、[#2622](https://github.com/hat47x/tei-platform/issues/2622)、[#1180](https://github.com/hat47x/tei-platform/issues/1180)、[#2514](https://github.com/hat47x/tei-platform/issues/2514)の候補とする。

| 反例 | 比較する差 | 期待する診断・意味 |
| --- | --- | --- |
| CRP-01 | 同じ入力から異なる粒度の二出力 | 一物理Jobへの融合は任意。各Result Grainを維持 |
| CRP-02 | 伝票行丸めと伝票丸め | 同一出力Keyでも非同値を検出 |
| CRP-03 | 分類の多重所属・期間更新 | 配賦未定義なら結果を確定しない。旧結果を再利用しない |
| CRP-04 | 3キーの一部の組合せだけ要求 | 不要な集計群を生成せず、計画候補を比較 |
| CRP-05 | Filterだけが参照する列を変更 | Value Lineage以外のCondition依存で影響を検出 |
| CRP-06 | 同端点・同ラベル二Edge、二Union | Identity・多重性・二つの寄与経路を保持 |
| CRP-07 | 前日残高を翌日に利用／同日の無条件閉路 | 時間シフトありの反復のみ候補化 |
| CRP-08 | 同じ集計式を別Scope/Tenantで利用 | 不許可な中間結果の共用を禁止 |
| CRP-09 | SQL実行とGo実行でNULL/Decimalが異なる | Rewrite棄却、Provider能力未確認はunsupported |
| CRP-10 | 外部更新後にACKのみ消失 | Commit unknownのまま盲目的なRetryを拒否 |
| CRP-11 | ノード配置のみ変更／式・粒度変更 | View変更と意味変更を別に失効 |
| CRP-12 | Source→生成Go→Builder→CLI→別Consumer | Mock／説明／Artifact存在だけの成功を認めない |

**優先順位**：既存のC0→C1は一入力・二Resultの意味照合、C2→C3は粒度・分類・Scope・演算性質の負例と候補計画比較に限定する。C4は既存の実Reader/Go Generator/Builder/CLI/Provider/Consumerの経路に接続、C5で時間シフトと回復、C6でStudioの表・Graph・Node・Trace表示をつなぐ。N0→N4は同じOperation/Port契約のノード投影とサービス接続に用いる。**Studio Stage1→Stage2および現行CLI/PluginのP0を新Gateで妨げない。**

最初の差分で導入するのは正式な新DSLでも大規模Plannerでもなく、既存Source/標準Data/Operationを読んだ結果要求とNode投影の**照合票**および独立したExpectedである。業務意味の採用、Source編集資格、実Targetの適合、生成されたコードの実受領は引き続き別々に確認する。


## 11. 一つの意味モデルから複数の実現方式を選ぶ（2026-11-15追補）

### 11.1 結論：コード生成と直接解釈は対立する方式ではない

成果物中心の記述とノード中心の記述が同じData・Function・Operator・Processを参照するなら、次に必要なのは**処理の意味を変えず、実行先に応じて実現方法を選ぶ仕組み**である。

ある演算は、選択済みの型付き定義を既存の汎用関数がそのまま解釈すれば足りる。別の演算は、専用コードへ展開した方が扱いやすい。大量のデータを既に保持するDBへ計算を委譲する方が合理的な場合もある。外部作用を伴う操作は、実行権限・回復・隔離の契約を満たすHostやProviderでなければ動かせない。

いずれも**Accepted Meaningを実現する手段**であり、個別の業務ルールを四か所へ重複定義するものではない。定義の直接解釈を使うからといって、TEIに常駐する万能Runtimeや新しいScript Engineを必要とするわけでもない。既存のOperation、型、Evaluator、Query IR、Generator、Target／Provider Contractを組み合わせ、何を解釈し、何を生成し、何を外部に渡すかを段階的に決める。

| 実現経路 | 主に適する条件 | 実行で固定・確認するもの | 不適合とする例 |
| --- | --- | --- | --- |
| **既存Evaluator・汎用関数による直接解釈** | 純粋で範囲の定まった式、検証、簡易集計。既存の型・Operatorで意味を失わずに解釈可能 | 実Loadされた定義byte・選択Reader/Operator・Evaluator版・入力範囲、実呼出しと結果 | 未対応演算を黙って無視する／Target側の異なるNULL規則に合わせて解釈する |
| **Target向けコード生成・静的合成** | Go単一バイナリへの収録、反復利用する専門演算、静的依存・型検査、Target別の表現が有効 | 生成元閉包、選択Generator、生成byte、Builder実消費、Artifact、実Entry/Consumer | 既存の手書きGoへ委譲するだけで生成した判断がない／生成時と実行時のSource版が異なる |
| **DB Queryへの委譲** | Relational QueryをTargetが必要な精度・NULL・順序・時点で実現でき、データ移動を減らせる | Query IRとTarget SQL、Dialect/Provider能力、Snapshot/Scope、実DB結果 | SQL方言差を無視したPushdown／集計・丸め境界の変更／行スコープの漏れ |
| **外部Host・Process／Pluginへの委任** | 外部API・DB更新・分離実行・専門ライブラリ・非同期実行など、個別のAuthority/Recoveryが必要 | Contract、選択Provider、actual Image/Invocation、現在Grant、Effect/Receipt、実Consumer受領 | Providerの宣言があるだけで呼出し済みとする／Effect unknownで無条件再送する |

この分類はCoreの固定列挙型でも、ユーザーが必ず記述する実行モードでもない。一つのFunctionの一部をDBへ委譲し、別の純粋な計算をGoに生成するような**混成方式も候補**である。ただし各境界で、キー・粒度・時点・順序・完全性・情報公開範囲・効果の意味を保存できなければ分割しない。

### 11.2 何を固定し、何を選択可能にするか

業務の記述には、その結果を決める情報を残す。たとえば「伝票単位で丸める」は業務意味であり、物理実行のヒントへ移してはいけない。一方、「DBへ押し込む」「処理を融合する」「並列数を増やす」は、意味の保存が証明できる範囲で変更できる実現方法である。

| 記述の性格 | 例 | 保存先の考え方 |
| --- | --- | --- |
| **業務意味として必要** | 出力のGrain、算出式、分類版、丸め位置、業務時点、承認条件 | 元の表・DSL・GraphとAccepted Meaning。人が読める認知順を維持 |
| **必要な実行制約** | 最大遅延、実行地域、開示Scope、外部Effect可否、必要な再開安全性 | Function/Process/Operationの要求とTargetの適合検査 |
| **実装方式の希望** | 生成・直接解釈・Query委譲、計算共有、Materialization、並列度 | Build/Deployment/Provider選択の候補・Hintとして保持。意味の変更にはしない |
| **開発知識・パターン** | 特定の分類集計の展開手順、型変換の検証、よく使うOperator群 | Pattern/Skill/Generatorの再利用候補。業務DSLにGoの構文を持ち込まない |
| **図表の表示設定** | Node位置・線種、Markdownの表現、折り畳み | 表示Projection。意味・生成Artifactの無用な失効を起こさない |

図表に実装上の属性を追加すること自体は後退ではない。**どこまでを業務意味に取り込み、どこからを再利用可能な実装知識として分離できるかが明確になる**という設計の前進である。ただし、実行計画の都合だけで業務DSLを複雑にせず、必要な場合にInspectorで詳細を展開できる構成とする。具体的な構文や属性名は、実例・型付きSourceへの対応・Ownerの採否が確認されるまで固定しない。

### 11.3 実装経路を比較するときの証明対象

選択した経路が変わっても同じ意味を実現することは、結果の一件だけが一致したことからは証明できない。最小限、次の単位を揃えて比較する。

1. **意味の同一性**：対象Sourceの原byteとOccurrence、選択Reader、受理されたFunction/Ruleと必要なInput/Output・各Revision。
2. **入力の同一性**：業務対象とKey、Snapshotまたは取得範囲、完全性、NULL/欠損、分類の有効時点、開示Scope。異なるInputを同じ条件の比較にしない。
3. **変換の同値性**：Operatorの精度、丸め、三値論理、順序、Window、例外・評価不能、部分取得。表現できない場合は`unsupported`を残す。
4. **実現方法の同定**：生成なら生成byte・実Builder/Artifact、直接解釈なら実Load定義と実Evaluator/Operator、DBなら実Dialect/SQL/Provider、外部委任なら実Image/Entry。全経路に存在しない生成Sourceを架空に補わない。
5. **実行・受領の同定**：独立Expected、実Attempt/actual Provider、実Result bodyとConsumerの取得。外部EffectがあるときはReceiptを追加し、純粋変換の同値試験だけでは安全性を保証しない。

同一の評価関数を二度呼んで「二方式が一致した」とする試験は無効である。直接解釈と生成経路の独立性を確かめるには、少なくとも実行された二つの実装を区別し、さらに**どちらからも生成していない独立Expected**を用いる。実Targetによる丸め・Collation・日時の差は、同値になるよう暗黙変換せず、不適合と明示する。

### 11.4 Node View・Plugin選択・実装経路の接続

同じOperationのノードを複数のFlowやResultへ置けても、Nodeの接続は物理Providerの選択や実行許可を確定しない。

- **Logical Operation**：何を計算／実行するか。Function/Operatorと版、型、粒度、効果を参照する。
- **Node occurrence / Port**：一つのFlowでの配置と入力束縛。業務対象IDやPlugin package IDとは分ける。
- **Implementation candidate**：Interpreter、Generator、DB Adapter、外部Providerが公開する契約上の候補。同じFunctionへ複数候補があってよい。
- **Resolved／Built／Invoked**：選択済みProvider／Lock、実Buildへの収録、実Hostからの呼出しをそれぞれ異なる関係として示す。

共通Graph Canvasは、結果要求から導いた計算依存Graphと、ユーザーが作ったNode Flow、PluginのDeclared／Selected／Built／Observed関係を**表示上で切り替える**。描画に使うEdge、元の業務Relation、Plugin Contract Binding、実Attemptの因果関係は異なるIdentityとする。

PluginのPortを結び替える操作は、Source Revisionと唯一の編集対象を確定した型付きPatch案に変換する。Provider候補が複数、実Sourceの書戻し位置が曖昧、または権限が不足する場合はread-onlyに留める。既存[Node共通表現契約](SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md#9-ノードビューをプラグインの関係配線にも共用する2026-11-15追補)と[#951](https://github.com/hat47x/tei-platform/issues/951)を利用し、新しい配線Resolverを作らない。

### 11.5 二つの出力と二つの実装経路を、一つのFixtureで確認する

初回は、現行の内蔵Markdown Rule Reader、型付きValidation、生成Go、既存Evaluator、固定入力・期待値を再利用する。公開mainに統合された[PR #5009](https://github.com/hat47x/tei-platform/pull/5009)の「選択済みRuleの生成Goを通常CLIでビルドする限定経路」は、実生成・ビルドの基点となる。ただしその事実は、任意のBatch/Graphや実DB Providerへの接続、同一意味の全実現方式の受入を証明しない。

**第一段：既存Rule一件での限定比較。** 同じ原文Rule・型付き属性を、既存Evaluatorによる直接評価と、生成Goの別Process実行へそれぞれ渡す。正値／0／負値／NULL／欠損・参照Attribute変更を独立Expectedで比較する。両経路の選択実装・実際のInput・Source/Artifact Revisionを記録し、生成Goが判断を実装していることを確認する。

**第二段：一入力・二出力の純粋計算。** 既存[#4410](https://github.com/hat47x/tei-platform/issues/4410)の固定入力（取得→Filter→Convert→Aggregate）に、明細と集計など**粒度の異なる二つの結果要求**を結び付ける。新しい専用Batch DSLではなく既存Data/Query/Operatorへ落とす。最初は分離計画で独立Expectedと比較し、共有計画が同じMeaning、Snapshot、Scopeを満たすときだけ融合の候補にする。

**第三段：DB側とGo側の同値性比較。** 同じ型付き演算をDBへ委譲できる最小一例を選ぶ。実SQL生成・実Provider Read・独立Consumerを確認し、NULL、Decimal、NULL群への集約、丸め、Order/Collation、Join多重度の不一致を負例とする。通常製品CLI/ProviderのP0に接続し、比較のために第二Query Engineを作らない。

**第四段：増分・継続処理の適用条件。** 最初の二出力が正しく成立した後に、変更／削除・遅着・訂正・多重所属を追加する。時間窓からの暫定出力と、業務として確定した出力を同一視しない。Window、Watermark、再計算対象範囲、Source Revision、Consumerが求める完全性を保存できない方式は採用しない。外部Effectの再実行は既存のReceipt-first回復に委ねる。

### 11.6 具体的な独立Expected候補と段階

| ケース | 一つだけ変える条件 | 期待する区別／停止条件 |
| --- | --- | --- |
| **LC-01** | 同じ正値Ruleを既存Evaluator／生成Goへ渡す | 二実装の結果と第三の独立Expectedが一致。GeneratorのWrapperだけでは合格しない |
| **LC-02** | Ruleの参照Attributeを変更する | 両経路で現行Sourceを反映。旧Generated Artifact／旧解釈Cacheは失効 |
| **LC-03** | 一入力から明細と集計の二結果を作る | 二つのGrainを保持。計画を融合しなくても成立 |
| **LC-04** | 税額の丸め単位だけを変える | 融合やPushdownの結果が変われば非同値と判定 |
| **LC-05** | 実装経路に未対応の演算を含める | unsupportedと棄却理由。空結果や別意味へ黙って代替しない |
| **LC-06** | Providerだけを切り替え、SQLのNULL/精度意味を変える | 実Target差を検出し、無条件Pushdownを拒否 |
| **LC-07** | 同一Resultを異なるTenant/Scopeから要求する | 共通Materializationを無断再利用しない |
| **LC-08** | Source A→B→A／Generator・Evaluator版を更新する | World/Model/Artifactの失効を個別判定。古い承認・実装を復活させない |
| **LC-09** | Node Layoutだけ変更／Plugin接続先を変更する | View変更は意味を失効させず、接続変更はResolver/Build/Consumerで再検査 |
| **LC-10** | 外部Effect後に応答のみ消失する | pureな演算経路の再実行可能性を流用せず、Receipt照会・unknown維持 |
| **LC-11** | 生成コードが既存Goへ委譲するだけの構成に変更 | 意味あるコード生成・Plugin自己構築として計上しない |
| **LC-12** | 候補計画の処理費が低いがGrant／完全性が不足 | 安さを理由に不適合な計画を採用しない |

LC-01〜12の独立Expectedは既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)へ引き渡す。**優先する最小差分はLC-01/02**とする。実Evaluatorと生成Goを一つの比較票で照合した後、LC-03を既存C0→C1へ重ねる。NodeのPlugin配線は既存PV-N0/N1に、Studio実ConsumerはW0〜W4、外部サービスはN0〜N4にそれぞれ接続する。

### 11.7 セルフビルドの到達度をどう評価するか

モデルが実装をどれだけ担うようになったかを、単なるDSL行数や生成ファイル数だけでは測らない。最低限、次を継続比較する。

- **実装された判断の比率**：明示的な業務Rule・式・分岐・集約のうち、TEIのSourceに根拠を持ち生成実装または適格な汎用Operatorへ移せた割合。未モデル化領域を分母から隠さない。
- **Pluginの自己構築範囲**：Pluginの契約・構成・Data/Process・Generator入力のうち、`*.tei.md`と既存標準要素から実現できる部分。選択・収録・実呼出しを区別する。
- **重複と変更コスト**：同じ規則をGo／SQL／Graph／Nodeへ手書きで繰り返す箇所の減少、変更時の影響分析・再生成・再検証が必要な範囲。
- **実受領の到達範囲**：Source→生成／解釈→実BuilderまたはRuntime→実Entry/Provider→独立Consumerまで同じ対象・版・Attemptで確認できた範囲。

単純なGo delegateを何件増やしてもセルフビルドの進展とは数えない。一方、定義パラメータを解釈する**汎用Operator自体が正しくモデル化・再利用**され、個別の手書きコードや重複設定を減らすなら、それもTEIの有効な成果である。必ずしも毎回のコード生成を目標にしない。

## 12. 既存計画への引渡し（2026-11-15）

| Owner／既存計画 | 今回追加する最小の作業 | 追加しないもの |
| --- | --- | --- |
| #1130／#4410・C0〜C3 | 二Result・二粒度・同じOperatorの合成、融合／分離の比較 | 別Batch Engine、万能Optimizer、独立Scheduler |
| #2622／#988 | 実装経路が要求する型、粒度、精度、完全性、Operator性質・Unsupportedの照合 | 新しい基本型やOperator体系の無条件拡張 |
| #4544／#5009・R1→R2-P | 同じRuleのEvaluatorと生成Go比較、Builder消費byte・実Callの閉包 | Go関数呼出しだけの「生成達成」 |
| #951／#1016・PV-N0 | Operator／Pluginの候補、選択、Build収録、実呼出しを別Relationとして投影 | 図上の線からの自動Provider選択 |
| #1124／#2497・N0／W0 | 結果表・Flow・Plugin Graphを同じ意味Sourceへ読み取り専用で関連付け | 可逆性未確認のSource書戻し |
| #4497／#1129／#1174 | 起動契約と時間依存、ResultとEffect/Receiptの分離 | 無条件再送・汎用のexactly-once保証 |
| #1180／#2514 | LC-01〜12の独立Expected候補を既存Corpusへ統合 | 第二のExpected Registry |
| #2831／#4583 | 現行の非Plugin Desktop Stage1→Stage2と実Windows受領のP0を維持 | 上記LCの全面実装を先行Hard Gateにすること |

既存のCLI実生成・実ProviderとStudio Stage1→Stage2を優先する。ここで提案する実装経路の選択、計画同値条件、テスト候補は、正式DSL・実行契約・製品機能の完成を宣言するものではない。最初に一つの意味が**二つの実装で独立Expectedに合う**ことを示し、その成功範囲から設計と実装を広げる。


## 13. 成果物の変更影響を、再計算できる範囲まで追う（2026-11-15）

### 13.1 次に共通化すべきものは、処理そのものではなく「結果を維持する条件」

複数の成果物を同じData／Operationから構成できても、データや分類が変わるたびに全件再計算するだけでは、大規模バッチ・長期運用・データマートには十分でない。他方、入力の追加分だけを処理すれば済むとも限らない。業務ルール、分類体系、期間、利用者のScope、演算子の意味が変更された場合、以前の結果を安全に使えるかは入力レコードの差分だけで決まらない。

ここでは**Resultを正しく維持できる条件と、条件が崩れたときに必要な再計算範囲を説明する能力**を検討する。これは新しいジョブ実行エンジンではなく、既存のDependency／Data／Query／Source Revision／Artifact／Period／Pluginを照合する**読み取り専用の計画**から始める。

結果要求を評価する際に必要なのは少なくとも次の五つである。

| 観点 | 結果に結び付ける内容 | 「変わっていない」といえる条件 |
| --- | --- | --- |
| **Meaning** | 算出式、粒度・分類、丸め・NULL、必要なFunction/Rule版 | 結果に寄与する定義の意味が保たれ、変更された部分の非寄与を確認できる |
| **Input population** | 対象集合、Key、Snapshot、業務評価時点、取得範囲 | 対象集合の包含・欠落・重複とSource改版を区別して確認できる |
| **Dependencies** | 値・Filter条件・Join／分類・時間・状態・Effect／権限 | 変更された要素から当該Resultへの寄与経路がない、または再計算により解消した |
| **Realization** | 選択Operator／Generator／Evaluator／Provider・Target能力と実Build/Run | 変更後の実装も必要な意味契約を満たすことを独立に検証できる |
| **Publication** | 出力Generation、独立検証、公開先、利用者Scope、実Consumer | 現在の公開資格とConsumer要件を満たす。出力Digest一致だけでは足りない |

「式は変わっていない」と「集計対象のデータは変わっていない」を同一視しない。Sourceの原byteが同じでも、DBの実値やCurrent Grantは変化し得る。World／Model／Artifact Revision、入力Snapshot、Materialization Generation、実Attemptを別々に扱う。

### 13.2 増分計算の可否は、更新の種類と演算の性質で判断する

差分更新は、演算式を一度使い回せることではなく、**旧結果と入力差分から新結果を再構成できること**を要する。対象集合の完全性と業務意味を保存できなければ、局所再計算ではなく全件再構築、保留、または適用不可を選ぶ。

| 入力・意味の変化 | 局所更新が成立する可能性 | 決め手と停止条件 |
| --- | --- | --- |
| 完全性の確認された集合への追加 | 加法的なSUM／COUNTなどは候補 | 同じ粒度・Key・期間・精度で、重複なく追加部分を列挙できる。途中丸めを移動しない |
| 既存行の訂正・削除 | 旧寄与分を除去できる演算だけ候補 | 旧値・旧所属・旧時間窓を取得できるか。再計算に必要な履歴がなければ全件／対象範囲再構築 |
| AVG／DISTINCT／MIN／MAX | 合成可能な**中間状態**を持つ場合に限る | AVGは(sum,count)、COUNT DISTINCTは重複を追える正確な状態、最大値の削除時は次点の情報が必要 |
| Join・外部結合・存在否定条件 | 追加だけでも旧結果を取り消す場合がある | Anti-join、NULL補完、多重度が変わるか。単調性を仮定しない |
| 分類体系・対応関係の改版 | 影響する旧所属と新所属を列挙できれば候補 | 多重所属・有効期間・配賦の規則を確認。単なる現行コード表との差分だけでは過去Resultを更新できない |
| 抽出・検証・算出Ruleの変更 | 原則として変更対象の影響範囲を再評価 | **従来出力から除外された行**も採用候補になり得る。出力済み行のLineageだけでは不足 |
| 時間窓・業務締め・遅着 | 再計算範囲を閉じて列挙できる場合に限る | Windowの確定条件、前期残高の繰越、過去期間訂正と後続期間の因果関係を確認 |
| 取得権限／開示Scopeの変更 | 値が同じでも従前Materializationを使えないことがある | 現在のRead／Disclosure適格性を照合。権限緩和時も旧Scopeの結果は自動的に完全とはしない |

変化を`insert/update/delete`だけに絞らず、**Meaning変更、Reference変更、時間窓変更、実装選択変更、Authority変更**もそれぞれ異なる原因として残す。データの変更履歴が得られない場合、Sourceの改版だけからCDCの完全性を推測しない。

変更の伝播は、既存Graph／Relationの**型付き依存辺を前方へ辿る**候補として表現する。ただし、寄与経路があることは「値が必ず変わる」ことを意味しない。逆に、値の寄与が見えないだけでは非影響を証明できない。Filter／Join／Securityの条件として使われた項目を含め、影響を「要検証」として扱う。

### 13.3 再利用・局所再計算・全件再構築・保留を、実行計画ではなく判断の候補にする

各Result／区分について、既存の契約で表現できる範囲から次を比較する。

| 判断候補 | 前提 | 判断できない場合 |
| --- | --- | --- |
| **Reuse** | 入力・Meaning・実装・Scopeの必要な閉包が一致するか、変更の非寄与を根拠付きで示せる | 単なるArtifact Digest一致を根拠に使わない |
| **Scoped recompute** | 対象区分・旧新寄与・必要な中間状態を列挙でき、局所更新後の意味が保存される | 対象区分の完全性・更新順序・撤回可能性が不足すれば棄却 |
| **Full rebuild** | 対象集合の全体を、同じ評価時点・現在の権限で読み直せる | 全件取得が不可能な状態を「0件」や完全再構築成功と表示しない |
| **Hold / require review** | 計算根拠、現在Scope、外部Effect、時間上の確定条件が不明 | 不確かな部分を空集合として埋めたり、安全な再実行とみなしたりしない |

これは新しいRuntime State enumや一元Job Tableの提案ではない。各候補を**既存Source・Dependency・Artifact・Providerの証拠に基づく説明票**として生成する。選んだ計画に基づく実行・権限・公開は既存Host／Provider／Effect/Consumerで別に検収する。

**全体完全性と部分集合の完全性**を区別する。たとえば「特定伝票の明細はすべて取得した」ことと、「その営業日の全伝票を取得した」ことは異なる。部分取得から存在を確認できても、不存在、一意性、総額、期間を跨ぐ合計を確定できるとは限らない。Windowの早期出力も最終結果として公開できるとは限らない。

### 13.4 計算・制御・保存の三つを切り離し、必要な範囲だけ結び付ける

- **Calculation**：型、粒度、Expression、分類、時間窓、Operator Lawの意味。宣言的な結果定義表または明示的なNode/Operation合成から導く。
- **Coordination**：対象期間の列挙、Trigger、Process依存、実行条件、並列・再入・期限。業務日付と実際の起動時刻は独立。
- **Materialization**：Output Generation／Manifest／Snapshot／Partition／中間状態の保持、再計算・失効・検証・公開。共有したいからといって必ず同じ物理中間ファイルへ集約しない。

Result-firstとNode-firstは、この三者へ至る**異なる編集入口**である。Node表現はCalculationのOperation配置にもCoordinationの手順にも使えるが、Data EdgeとControl／Effect Edgeを同一視しない。結果表から「人が承認する」「送金を実行する」等の外部Effectを自動生成しない。実行制御のScheduler／Job管理は既存Host、OS、外部Providerで必要なものを選ぶ。

こう整理すると、データマート、月次締め、計算科学の反復、バックフィル、APIのキャッシュ更新、画面の再計算を**同じ変更影響と成果物維持の問い**で扱える。ただし、継続データのWindow、非可逆Effect、業務確定では必要な契約が異なるため、すべてを同じ実行保証へ変換しない。

### 13.5 最小Fixture：二つの結果と二種類の変更を用意する

既存の固定TSV／型付きRule・Operationを用い、**同じ入力から伝票税額表と店舗日次表**を生成する。税率10%・端数四捨五入（0.5切り上げ）を採用した人工例で、伝票Aの明細金額が15円と15円なら、各明細行で丸めてから合計すると4円、伝票合計30円に対して丸めると3円になる。これを変換の非同値を示す独立Expectedとする（実業務の税制を表すものではない）。

次に一因子ずつ変える。

1. **追加入力**：伝票Bの明細を増やす。Bの追加のみで済むか、全店日次の再計算範囲がどこかを、完全な元集合と重複規則を固定して比較する。
2. **従来除外されていた行の採用**：Filterの条件を変更する。旧出力には現れない明細の存在を示すCondition依存がなければ、再利用安全とは判定しない。
3. **分類改版**：店舗→地域の所属期間を変更する。旧期間・新期間のどちらに寄与するかを判別し、旧集計への再利用を説明可能にする。
4. **入力の一部欠落**：一伝票の一明細だけ未取得にする。伝票の確定税額も日次総額も確定結果へ昇格させない。
5. **実装方式だけを変更**：同じMeaningから直接Evaluator／生成Go／DB委譲のうち、実際に実装されている経路だけを比較する。未実装Providerを実測済みとしない。

このFixtureの原入力・出力・期待値・失敗条件は既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)のCorpusとして管理する。本稿に第二の試験正本を作らない。

## 14. 既存実装の到達点を踏まえた着手順（2026-11-15）

同じ型付きRuleの**既存Evaluatorと生成Go実Binaryを五入力・Read回数で比較する限定的な回帰**は、main統合済み[PR #5087](https://github.com/hat47x/tei-platform/pull/5087)に含まれる。Source／Rule／型を参照して生成Goを得る局所経路は[PR #5077](https://github.com/hat47x/tei-platform/pull/5077)にも含まれる。これらは新たなSource→正式Builder全閉包や通常CLI実Provider、汎用Batchの適合を実証したものではない。**新しい第一歩として同じ五入力試験を別実装しない。**

次に不足している実装の単位を以下へ絞る。

| 段階 | 最小の作業 | 合格の上限・Owner |
| --- | --- | --- |
| **R-0：read-only要求・影響票** | 一入力・二Resultを既存Data／Rule／Query／Operator参照へ写し、出力Grain、Value/Condition/Classification/Temporal依存と入力完全性を表示 | 構造と根拠の照合まで。#1130／#2497／#2622 |
| **R-1：再計算範囲の対照** | 上記FixtureでRebuild／Scoped recompute／Holdの候補と棄却理由を作り、独立Expectedで照合 | 純粋計算・固定入力に限る。#1130／#1180／#2514 |
| **R-2：二実装経路への接続** | 現行mainのRule・Data項目定義→生成Go／既存Evaluatorを、同一Source・入力閉包で二Resultの一部に適用 | 一Ruleの限定受入をバッチ全体へ拡張しない。#4544／#4410 |
| **R-3：実Query/Provider** | 元Query意味とTarget SQL、実DB Snapshot／Result Body／独立Consumerを照合し、追加・訂正・削除のうち表現できる差だけ採用 | 実Provider・Current Grant・Resultの証拠を別にする。#2622／#2586／#4495 |
| **R-4：Studio／Node／外部実行** | 同じSource・意味IDから結果表・Node View・依存Graph・再計算の理由を読み取り専用で表示 | Stage1→Stage2実Windowsと外部Effectは独立。#1124／#2831／#4583／#4497 |

R-0/R-1は既存C0→C3の狭い拡張として扱う。R-2/R-3はP0の正式Source→生成Go→Builder／通常CLI実Providerへ合流し、別Pipelineを並走させない。R-4は既存N0〜N4／W0〜W4を用いる。新しいGenerator本体・Batch Runner・Scheduler・常駐Host・統一Job Registry・Optimizer・第二Expected正本は導入しない。

**見直し条件**：実例で必要な粒度／時間／不在・部分取得／更新寄与が既存Data/Relation/Operatorに表現できない場合のみ、その**欠落した意味**を具体的な反例として既存Ownerへ戻す。新しい型を作る前にSource/Role/Facet／構成／Pluginのどこに置くべきか比較する。最適化や増分更新は、正しく再計算できることを検証してから採否を決める。

## 15. 複数成果物の世代整合：共通計算と共同公開を同一視しない（2026-11-17、設計候補）

### 15.1 新たな判断：計算の共有、成果物の世代、Consumerが見る組を分離する

§3・§10〜14の成果物要求／計算粒度／実現選択／変更影響を踏まえると、最後に残る境界は**「複数出力が計算できたこと」と「複数出力を整合した組として使えること」は異なる**という点である。たとえ二つのResultが同じ論理Operationから導出されても、別々のStore・DB・APIに公開された時点では、片方だけ新世代に切り替わることがある。Data/Relation/Ruleの正本、実Jobの完了、生成Artifactの完全性、各Consumerの実受領を一枚の成功にまとめない。

比較には最低限、次の区分を保つ。ここでの「組」は専用の新しい永続Core型ではなく、既存Result要求・Artifact Manifest・Source/Meaning revision・受理された業務期間の**read-onlyな照合視点**を指す。

| 比較単位 | 保持する意味 | それだけで証明できないこと |
| --- | --- | --- |
| Result要求 | 対象集合、出力Grain、集計条件、値・述語・分類・時間の依存、必要Scope | 実行・公開・Consumer受領 |
| Input Cut | Sourceごとの実Revision／Snapshot、対象範囲と欠落の有無、業務as-of条件 | 別Source間の業務整合時点があること |
| Shared Computation | 共有可能なOperator部分、利用した値、順序・丸め・NULL・分類・読取の契約 | 異なるGrainのResultが同時に公開されたこと |
| Result Generation | 出力ごとの不変内容／digest・実生成版、意味・入力依存、公開資格 | 他Result世代との組合せが有効なこと |
| Publication / Consumer Cut | 公開された組、参照方法、Consumerが実際に固定した世代と読取時点・開示Scope | 外部DB Effect確定、他Consumerの採用、業務完了 |

同じSource revision番号であっても、異なるReaderの実CutやScopeを同一と仮定しない。逆に、異なる粒度や異なる参照期間のResultには同一revision値を強要せず、**各Resultがどの入力CutとMeaningに従い、組として整合すると受理されたか**を説明する。分類の多重所属、集計前後の丸め、非出力Predicate依存は§3/§10/§13を再利用する。

### 15.2 合成反例：各Resultは正しくても組合せが誤る

不変の明細二行を持つ旧入力 S1（A=100、B=300、合計400）と、Bが500に訂正された新入力 S2（合計600）を用いる。明細Result D と集計Result T はそれぞれ独立に生成・検査可能とする。

- **正しい旧組**：D(S1)={100,300}、T(S1)=400。
- **正しい新組**：D(S2)={100,500}、T(S2)=600。
- **不整合な混在組**：D(S2)={100,500}、T(S1)=400。各Artifactのdigest・個別検算がPASSでも、同じ「現行結果の組」としては失格。逆の混在も同様。
- Consumerが「Dのactive」と「Tのactive」を別々の時刻に読み、途中で切替が起きた場合、表示時点に各activeが正しくても**同じ読取Cutを固定した証明にはならない**。

一つの信頼できるストアで**不変Artifact二つを指すManifestの参照先を一回の検証付き切替**として実現でき、ConsumerがそのManifestを一度固定して各Artifactのbytesを検証できるなら、そのストアとConsumerの能力の範囲で組の整合を示す候補になる。ただし、実Storeの原子的切替・保管寿命・権限・Consumer Pinを実測しない限り採用済みの保証としない。複数DB・外部APIへまたがる無条件の原子的公開を前提にしない。実現できないTargetは、**個別公開／組の未確認／保留**を区別し、部分的な成功を全組成功に丸めない。

### 15.3 再利用・更新・公開の順序

1. 受理されたResult要求と現在のSource／分類／時間／Scopeを照合し、各Resultの必要なInput PopulationとCutを確定する（§13のReuse／Scoped recompute／Rebuild／Holdを流用）。
2. Operatorの法則と実Source読取に基づき、**安全と証明できた共通計算だけ**共有する。Shared Execution自体を業務の原子境界にしない。
3. Resultごとに生成・検証して不変世代を識別し、旧世代参照を破壊しない。中途失敗や一部未生成があれば、組全体のReadyを宣言しない。
4. 依存する結果を同時に参照する利用者がいる場合、既存Manifest・Stage/Publish・Consumer契約で**一つの組の版をPinして読めるか**を検証する。実Targetにその能力がなければunsupported／unknownとして止め、第二のArtifact正本や強制Transaction Managerを作らない。
5. 下流にDB Write／Message／APIのEffectがある場合、Resultの公開版と**実Commit Receipt・Current Grant・Expected Version**を別の資格で照合する（#1174／#1129／#2499）。ACK不明時に再送・ロールバック可能と仮定しない。
6. 過去期間訂正、分類改版、外部Consumerの旧世代Pinの保持期限は、Result更新・旧世代保持・Reader終了・業務補償をそれぞれ別のClaimとする（バッチ設計§23・§25・§27・§29・§32）。

### 15.4 独立Expectedへ引き渡す反例（新しい試験正本を作らない）

| Case | 一因子の変更 | 期待する拒否・判定 |
| --- | --- | --- |
| MG-01 | S1の明細Dと集計Tを同じ旧Cutで読む | 二ResultのGrainと入力依存が一致し、合計400。純粋計算の限定正例 |
| MG-02 | DだけS2へ更新しTはS1のまま | 個別検証PASSでも同一世代の組として不適格。混在を検出 |
| MG-03 | T(S2)のStageが失敗しD(S2)だけ存在 | Run/Upload完了から共同公開を推論しない。旧組保持またはunknown |
| MG-04 | DとTのactiveポインタを二回に分けて取得 | Pinされた一組がなければ整合読取はunknown。時刻一致だけでPASSしない |
| MG-05 | 分類やFilter改版により旧Tの対象集合が変化 | 値のLineageしか追わない再利用を拒否。§13のCondition／Classification依存で再計算候補 |
| MG-06 | 同じ入力でもRead Scopeが異なり一部行が非開示 | 公開可能な限定結果と全域の結果を区別。非開示行の存在・件数を漏らさない |
| MG-07 | Tの下流DB更新後にACK消失 | Published／Applied／Commit unknownを区別し、Receipt照会なしの再送を拒否 |
| MG-08 | 値のdigestが偶然同じでもMeaning・Grain・期間が違う | byte一致だけで再利用／組の同値を認めない |

MG-01〜08は既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Expected候補として引き渡す。新しいRunner・Manifest型・中央Generation Store・Cross-Target Transaction Manager・Scheduler・第二DSLをこの文書から直接増設しない。

**Owner／順序**：#1130／#4410が先に固定TSVの一出力・CLI/Go二入口と不変Artifactを実証 → #1180/#2514がMG-01/02/03/04の独立Expectedを固定 → #2482／#2622が実Stage/Publish／実Readerに表現可能な範囲を検証 → #1129/#1174が下流EffectのReceiptを実証 → #1124/#2831/#4583が既存Studioへread-only投影する。既存§14のR-0〜R-4の中に追加し、P0の通常CLI実ProviderとDesktopセルフビルドを遅らせない。

**検証上限**：本節は調査・既存設計照合・有限算術による設計案。新規Go/CLI/DB/外部Consumer/Studio実行試験、実Providerの原子的切替、正式Owner意味受理は **NOT RUN / NOT ACCEPTED**。
