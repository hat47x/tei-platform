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
