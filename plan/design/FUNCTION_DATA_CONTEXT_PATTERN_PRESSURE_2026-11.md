# Function Data Contextと機能パターンの横断設計・圧力試験（2026-11）

- **状態**：調査に基づく構成案・実証計画。契約採用、実装、合成試験、実Target適合、業務意味受理はいずれも今回の成果ではない。
- **目的**：異なるデータ構造と専門業務機能を、TEIの既存Semantic Data、Function/Action、Query、Plugin/Service Contract、外部Hostへ無理なく合成できるか検証し、汎用ローコードとしての意味保持と開発生産性を測定する。
- **基本方針**：#2492（開発順序のDraft）、#2493（要件ポートフォリオ）、#2495（データ構造のメタ概念）、#1127（機能パターン）、#1126（意味的関係）、#997（UI）、#951 / ADR-0032（サービス契約）、#2018（生産性比較）を参照する。第二Canonical、第二Runtime、第二Scenario、専用Host、新Gateを作ることを目的にしない。
- **出典の扱い**：調査元の製品・技術を模倣せず、必要な構造・操作・制約・反例を独立した要求へ変換する。個別製品の名称や画面表現を、本書の公開機能要件の名称に用いない。外部資料の権利と機密性は#941に従う。

## 1. なぜ必要か

受注Hの子集合D1が2行、独立した子集合D2が3行ある場合、二つを同じQueryの平坦な結果にするとJOIN由来の6行が発生し得る。検索結果の1行・表示行・業務Entityの1件・関係の1件・集計グループの1件は、同一の識別単位ではない。識別を取り違えると、集計重複、誤更新、旧応答の混入、部分取得を全件と見なす誤動作が生じる。

この問題は親子画面に限らない。部品構成の複数経路、組織の兼務・代理、集計・ドリルダウン、時系列観測、工程日程、地理座標、人の承認、長時間処理で形を変えて現れる。共通の要求は「値の配列を表示できること」よりも、**何を一件として扱い、どの文脈・時点・権限・意味で取得、操作、観測するか**を保存することである。

## 2. 複数の文脈を区別する

| 文脈 | 担当する意味・状態 | 主な所有先 |
| --- | --- | --- |
| Semantic Context | Entity、Relation、値域型、制約、関係の役割、生存期間・受理済み業務意味 | #1126/#2495/#1129 |
| **Function Data Context** | Functionが要求・保持する関係付きの結果、粒度、識別、取得・選択・編集状態 | #1127（本書） |
| Interaction / Presentation Context | UI上の表示・フォーカス・選択epoch・入力状態と変更Intent | #997/#2391 |
| Invocation / Effect Context | Service Contract、呼出し主体、権限、Provider、OperationID、Effect、世代 | #951/#1121/#1174 |
| Observation / Evidence Context | 出所、取得範囲・時点・一貫性、実効果、独立観測、開示可能性 | #1131/#1163/#1178 |

ADR-0032のComposition ContextはService/Effectの作用域を指し、Function Data Contextを意味の正本として兼ねさせない。両者を型付きContractで接続する。

### 2.1 最小の論理要素（案）

- **Context Definition**：Function-local ID、Semantic Subject/Relation path参照、期待される結果の粒度（grain）、識別方式、選択・取得・操作に必要な入出力の型、任意の集計・変換参照。
- **Context Link**：関係の役割・方向、多重度・任意性、親子従属／独立参照／関連実体／自己参照、条件と適用時点。JOIN条件を意味的Relationの代用としない。
- **Result Slice**：実際に取得した値・対象ID・行や経路の出現ID・ページ境界・順序・重複・フィルタ・権限scope・時点／版・取得完全性。
- **Interaction State**：親選択ID、selection epoch、編集パッチ、暫定行識別子、dirty state、UI送信・応答時点（秘密の複製を前提としない）。
- **Operation Binding**：固定されたEntity集合か、解決時点を指定した条件集合か、関係link/unlinkか、集計結果から明示Actionへ渡す操作かを区別。実効果・業務完了は既存Operation/Receipt/独立readで確認する。
- **Projection Binding**：表示側が必要とする属性の役割と型を宣言。表示形式を変えても識別・Action・業務制約の正本は共有する。

これらを新しいCore enumとして一括追加する提案ではない。既存Semantic/Query/Pattern/Operation/Plugin契約の再利用可能性を先に測り、欠ける不変条件のみを狭く追加する。

## 3. 型／構造のメタ空間とFunction Contextの接続

#2495の直交した「分類する問い」をFunctionの適用条件へ投影する。多重facetの所属は許容し、意味的subtype、構成演算子、数学的性質、物理表現、変換可能性を区別する。

| Facet | 適用条件の例 | 誤った推定の反例 |
| --- | --- | --- |
| Identity / Occurrence | Entity ID、Relation ID、Path occurrence、Event、Observation | 表示の行番号を更新対象IDにする |
| Construction | Product/Union、Set/Bag/Sequence/Map、Record、Graph/Hypergraph、Stream | 1:N×1:Nをひとつの業務Collectionとみなす |
| Relation semantics | 従属、参照、関連、役割、方向、自己参照、許容循環 | 外部キーだけで所有・削除連鎖を決める |
| Mathematical property | 順序、集約演算、距離、座標系、確率・測度、グラフの接続制約 | 数列を地理位置、比率を加算可能な量とみなす |
| Quantity / time | 単位・次元、通貨、絶対値／差分、時点／期間、有効時点／観測時点 | 同じ単位文字列で演算適合とする |
| Materialization / scope | 未取得／NULL／欠落、完全集合／ページ／フィルタ部分、権限scope、版 | ページ1を全件とする、Stream途中を空集合とする |
| Operation / effect | Read、編集Patch、Link/Unlink、移動、再計算、長時間Task | Chart上のdragをCommit成功とみなす |
| Realization | SQL、JSON、API、columnar、UI、Go/Java/.NET/Wasm | 物理表現が同じなら意味型も同じとする |

「トポロジー」という言葉は、グラフ構造の接続性、データ構造の構成形状、数学上の位相的性質を文脈で区別する。Facetの複数所属を言語の多重継承へ自動対応させない。

## 4. 機能パターンの合成契約（候補）

一つの巨大な機能タイプ列挙より、Patternの**適用に必要な契約**を明示する。

1. **要求するData Shape**：必要な粒度と識別性、Relation path、属性役割、順序・重複・時間・単位、完全集合要件を宣言する。
2. **必要なOperation**：Read/Select/Create/Update/Link/Unlink/Reparent/Calculate/Commit等を、意味上のActionと既存Service Contractへ接続する。抽象の列挙は概念説明であり新Core enumを意味しない。
3. **提供するInteraction**：選択、絞込み、階層展開、drag、ドリルダウン、承認・通知のユーザー意図を型付きに表現する。
4. **必要な実行保証**：取得済み範囲・Snapshot、Expected Version、対象集合確定時点、Authority、失敗・結果不明・再試行の契約。
5. **実現先の制約**：Framework/Target/Host/Providerの互換性、バージョン、Scope、オフライン／Stream対応、性能・可用性。
6. **診断**：適合、明示変換候補、必要な意味の不足、実現未対応、現在の実行権限不足を区別する。型や意味の暗黙変換を最適化の都合で受理しない。

合成案としては、Context Definition→要求Facet・Operationの充足判定→明示Binding／Adapter→Pattern＋Presentation＋Operation Service→CUIによるPreview・実証という流れを候補とする。既存#1127の標準list/read/create/updateと#67のPatternを最初に再利用する。

### 4.1 標準Patternの拡張方向

- Record / Collection / Multiple Collections（単票、一覧、複数独立1:N）。
- Reference Choice / Association Editing（参照選択、Link/Unlink、関係属性編集）。
- Tree / DAG / General Graph / Path Traverse（自己参照、正逆展開、Path occurrence）。
- Aggregate / Cube / Drill-through（Dimension/Measure/Grain、集計結果と元実体の区別）。
- Temporal / Schedule / Resource Allocation（時刻、期間、依存、制約、変更提案）。
- Stream / Event / Telemetry（event time、ingestion time、window、lateness、subscription）。
- Spatial / Geometry（座標系、Geometry、変換、空間計算、表示）。
- Document / Form / Annotation（文書構造、反復入力、引用・注記・版と出所）。
- Procedure / Approval / Durable Operation（人の介在、業務状態遷移、操作と確定・回復）。
- Domain-specific Services（組織の時点付き権限、BOM数量展開、在庫割当、最適化、医療観測等）。

固有領域の演算・規則は専門Plugin/Service Contractを利用し、共通Data Contextへ「結果を表現できること」と「その結果を正しく計算・実行できること」を混同しない。

## 5. 反証可能な横断試験群（初期候補・すべてNOT RUN）

| ID | 用途・Fixture | 満たすべき意味 | 誤成功として検出する反例 | 主担当 |
| --- | --- | --- | --- | --- |
| FC-01 | 親H／D1=2件／D2=3件 | 独立2集合、識別、表とChartの表示差 | JOIN fanout 6件を明細数と扱う | #1127/#1706 |
| FC-02 | 関係レコードのN:M | Entityの生成削除とLink/Unlink/関係属性編集を区別 | Unlinkで参照実体をdelete | #1126/#1127 |
| FC-03 | BOM DAG・同一部品の複数出現 | Entity/Path occurrence、経路別数量、逆展開 | ノード重複排除で数量を減算 | #1126/#2495 |
| FC-04 | 組織の所属・兼務・代理 | 有効期間、所属Role、委任scope、権限評価時点 | 表示上の所属を即時Host権限とみなす | #2386/#1129 |
| FC-05 | Task/Dependency/Resource/Calendar | ガント表示と制約計算、Action/Effectの分離 | drag表示成功で計画確定と扱う | #997/#951 |
| FC-06 | Dimension×Measureの集計 | grain、NULL、元Entityへの明示Drill-through | 集計行を黙ってUpdate、行増幅 | #2387/#1705 |
| FC-07 | イベント・時系列Stream | Event-time/processing-time、window、lateness、部分観測 | 流入途中を0件確定／同時点Snapshotと扱う | #2495/#951 |
| FC-08 | 空間Feature＋CRS＋単位 | geometry role、座標参照系変換、距離・尺度 | 経緯度を平面m座標として比較 | #2495/#997 |
| FC-09 | 異種Record Bundleと状態遷移 | resource/entry、版、長時間処理と部分成功 | 受付ACKを業務完了とする | #1130/#1174 |
| FC-10 | 型付きDocument／脚注・数式・注記 | 構造と表示、出典・意味と引用表現 | レンダリングの成功を計算意味と扱う | #1803、SHO/MMI |
| FC-11 | 複数Providerと権限付き部分集合 | scope、Snapshot/Completeness、公開可能な粒度 | 部分取得から全件操作、SecretをAI転送 | #1706/#1178 |
| FC-12 | 二つのTarget／外部Host | 同じContract/Meaningの再利用、独立Effect確認 | CLI/PASSをWeb・外部Hostの実証に流用 | #2390/#2077/#1180 |

「反証」はCore全体を一度の失敗で否定することを意味しない。個別のPattern前提・Adapter・Source仕様・Providerの限界と、共有概念そのものの不足を分類して設計を更新する。閉じたfixtureの0件と実Providerの不完全な返却集合を区別する。

## 6. 開発順序：AI-first・Headless-first（#2492のDraft方針と接続）

**研究対象と並列Issue数は広げる**。各PoCの検証単位は小さく保つ。トークン資源が豊富でも、Ownerによる意味受理、実Host、CI容量、統合競合は別に管理する。

- **S0 要件探索（並列）**：FC-01〜12に対して外部要求・観測可能な結果・失われる意味・代替案を記録。規格・公開資料に基づくSource factとTEI設計仮説を区別する。#2493でポートフォリオ管理。
- **S1 構造の机上適合**：#2495のメタ軸と既存Semantic/Query/Service Contractで各例が表現できるか比較。新Core型の追加はConsumer上の反例が必要な場合に検討する。
- **S2 Headless Reference**：まずFC-01（独立2集合）、FC-02（関係操作）、FC-06（集計grains）を、CUIでinspect→plan→preview→validateまで再現。Web/Canvas実装を最初の必須依存にしない。
- **S3 Pattern/Plugin Conformance**：同じデータ・操作の意味を二つの異なる表示・異なるProviderで確認。標準Patternの適用条件・失敗診断、Contract/Generationのexact bindingを測る。ADR-0032のProduction移行は別途実証。
- **S4 Driver/Host/独立観測**：読み取り／編集／関係変更／承認などのMeaning→Action→Provider→Effect→Observationを既存#1121/#2485/#1180で実測し、NOT RUN工程を残す。
- **S5 GUI／専門Consumer**：Table/Chart/Tree/Gantt/Map/Nativeの各Rendererを追加。最初はヘッドレス結果との整合とGUI固有のa11y・操作・表示の正しさを検証する。
- **S6 生産性比較**：#2018へ設定項目数、重複意味数、AI指示・変更・検証時間、Adapter工数、境界違反、Target切替の手間を戻す。

S0の探索やS1の比較は、各専門分野について並行してよい。既存のGate A/B/C、Accepted Source、Source Index、Driver Scenario、Owner受理の基準を置換しない。

## 7. 採否を決める測定

各ケースに以下を記録する。

- **表現力**：表現できる／明示Adapterが必要／意味喪失／未解決の分類。
- **汎用性**：新しいCore種類の追加数、既存契約を再利用できた数、専用Pluginで閉じた差分。
- **意味の一貫性**：同一Entity/Relation/Grain・値域・Action・権限に対する異なる表示／Targetの整合、誤対象更新件数。
- **状態と効果**：部分取得、stale、同版Evidence、実Provider独立読取、Unknown/Denied/Conflictの区別。
- **生産性**：初回構成／変更の作業量、AI反復回数、Adapter/Plugin変更量、Owner確認箇所、再生成影響範囲。
- **実装と運用**：Latency/P99、必要メモリ、プラグイン差替え、開示境界、エラー復旧。測定していない指標は未測定のまま保持。

**撤回条件の例**：複数の異質Consumerの共通化でかえって意味重複が増える、Contextの行を業務実体と誤認して誤更新が発生する、特定UIやDB前提のためのCore分岐が常態化する、専門アルゴリズムを汎用Data Shapeへ押し込む必要がある。撤回／変更時は既存契約へ戻し、根拠と未採用候補を保管する。

## 8. 課題の役割

- **#1127**：Function Data Contextを利用する標準機能と操作の導出。既存Operation Surfaceや#2485を変更せず、実証入力を追加する。
- **#2495**：構造・数学・単位・観測状態を分類するメタ概念。プログラミング型をそのまま上位の意味分類に使わない。
- **#2493**：異質な外部要求を収集し、並列PoCと担当Issueへ振り分ける。具体的な要件の出典と権利は既存規則を適用する。
- **#1126/#1705/#1706/#2387**：関係・型／単位・集合の完全性・Query由来の結果とLineageの担当。
- **#997/#2391**：標準/Native UI、外部制御、UI状態と操作の観測の担当。
- **#951/#1223/#2390**：Service Contract、Plugin/Host合成、外部Consumerとの連携の担当。
- **#2018**：比較評価の測定方法の正本。ここへ独立した競合ベンチマークを作らない。
- **#1121/#1131/#1163/#1180**：生成・実行・Dataset/Scenario・来歴とEvidenceを既存経路で確認する。

## 9. 依然として未決の点

1. Function Data ContextのDefinitionを正式Sourceへどの粒度で記述するか、既存Application/Query Bindingからどこまで導出できるか。
2. 取得完全性、Path occurrence、stream time、agg grainのうち、共通Contractへ残す最小項目と専門Pluginへ委ねる項目。
3. 複数Context間のSnapshot一貫性をProviderが実際に保証できる条件と、部分観測をまとめて提示する場合の表示方法。
4. Function Patternの適用条件をどの程度形式化するか。数学的facetに「自動変換」を導入する場合のLoss/権限・Owner判断。
5. 既存X-TEA等の参照資料に由来する機能型の知見と、新しい汎用Pattern候補との意味対応が実際の業務で一致する範囲。

本書は広い機能探索を促進するための設計案であり、機能一覧の完成・既存Issueの完了・Production Releaseの保証を示すものではない。

## 10. 読取り投影と編集可能な投影の分岐

**同じContextを表示できることと、その表示を編集して業務状態へ戻せることは別の能力**である。編集可能性は任意の表示コンポーネントの属性として先に保証せず、Source Dataの識別性・導出の情報損失・Actionの明示Bindingにより決定する。

- 読取り投影：`view = project(context, projectionSpec)`。必要な元Data・版・集計粒度・結果IDと、その表示の依存関係を記録する。
- 変更要求：`intent = interpret(viewSelection, proposedChange, sourceBinding)`。UIの差分をActionに対応づける。集計・JOIN・重複・ソート・正逆展開から更新先を一意に推定できない場合は、明示Actionまたは未対応とする。
- 業務状態への反映：`result = invoke(action, intent, authority)`。Validation、Transaction、Receipt、実効果と回復は既存実行契約が所有する。
- 独立再投影：独立した読取りで、正しい対象・時点・Scopeの結果が表示されるか確認する。

具体的には、編集可能な明細行と、同じ明細をまとめた取引先別売上チャートの集計棒は同じData Contextから派生できるが、チャートの金額を変更することから個々の受注行の更新先は一意に定まらない。対して、Task IDと期間変更Actionを明示束縛した時間軸のdragなら、変更Intentを生成できる可能性がある。

往復性の試験候補：同一Source/Query/Model revisionに対する読取り再現、変更なしの提案で意図せぬEffectが生じないこと、変更後に同じ対象・Scopeを独立読取りすること、二つのRenderer間でAction/ID/Validationの意味が一致すること。情報を失う投影に一般的な可逆性を要求せず、**可逆条件・明示的に非可逆な範囲を型付き診断**として扱う。

## 11. 複数Contextの被覆範囲・時点・権限の合成

「全件取得」は対象母集団が明示されて初めて意味を持つ。例えば `completeWithin(scope, predicate, snapshot)` のように、Scope・Filter・Snapshotに相対的な保証として扱うことを検討する（正式なAPI名ではない）。

必要な観測軸：

- **Coverage**：結果の出所、述語、返されたページ／継続トークン、取得完了が保証される範囲。権限により隠れた行の存在まで開示する要件ではない。
- **Temporal**：有効時点、取得時点、Event time、到着時点、選択epoch、Provider版の区別。複数Providerの読取を自動的に一つの整合したSnapshotとは扱わない。
- **Epistemic status**：観測済み値、テスト期待値、意味シミュレーション結果、未観測、未確認を区別。SEIの知識の確信度とは異なる軸として必要な接続のみ行う。
- **Disclosure**：AIへの渡し方は情報の可視性・列/行scope・機密区分・利用者の許可に従う。データの読取権限を外部LLMへの開示権限と同一視しない。

反証ケース：結果が0件でも、閉じたfixtureにおける確定0件と実際のProviderで返された部分集合0件では保証が異なる。親Contextの版と二つの子Contextの取得時点が異なる場合も、同一業務時点の一貫性を無条件には主張できない。UIで見える行集合と一括Actionが対象とする確定集合の区別は、既存#1127/#1708の固定／遅延Binding契約へ接続する。

## 12. 設計に利用できる外部標準（比較資料）

以下は各標準が扱う特定の意味境界に対する参照資料であり、そのままTEIの仕様採用・実装互換を意味しない。

- [SHACL（W3C）](https://www.w3.org/TR/shacl/)：Node/Property ShapeとProperty Pathの関係。Relation pathと型付き検証の参考。
- [RDF Data Cube（W3C）](https://www.w3.org/TR/vocab-data-cube/)：Dimension/Measure/Attribute/Observation。集計grainの参考。
- [PROV-O（W3C）](https://www.w3.org/TR/prov-o/)：Entity/Activity/Agentによる来歴・責任の区別。
- [Cursor Connections Specification](https://relay.dev/graphql/connections.htm)：Cursor/PageInfoによる結果集合とページ状態。
- [OGC SensorThings API](https://www.ogc.org/standards/sensorthings/)：Thing/Location/Datastream/Observationと観測・指示の区別。
- [QUDT](https://data.qudt.org/)：Quantity Kind、Unit、Dimension Vector。量の意味と数値表現の分離。
- [Apache Arrow Columnar Format](https://arrow.apache.org/docs/format/Columnar.html)：List/Struct/Map/Union等の物理表現とApplication意味の差。
