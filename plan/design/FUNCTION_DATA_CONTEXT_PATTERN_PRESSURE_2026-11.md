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


## 13. 複数Data Contextの反応的な連動（2026-11-02：独立Issue #2582）

### 13.1 なぜ別の検証が必要か

Data Contextが複数のResult SliceとGrainを正しく保持しても、**変更が発生したとき、どのClaimを現在の値として扱えるか**は別途判断が必要である。親の選択変更、関係先の更新、Query/Rule改版、外部通知、業務操作のACK消失が同じ「再表示」イベントに見えるUIは、過剰な再計算と、誤った現在値・誤更新を同時に発生させ得る。

既存の#1706（集合同一性・入力完全性）と#1708（複数読取りの整合）は参照検証を扱い、#1641（導出値・キャッシュ・確定保存値）は鮮度の意味を所有する。#2387のRelationalIR/Query演算依存、#2506のResult Slice、#2500のViewChange→操作候補、#1174の実Effectと独立観測を参照し、**実行時の横断受渡しだけ**を新しい#2582で検証する。

### 13.2 Claim相対の依存と再評価条件

同じData Context内でもClaimごとに必要な入力・時点・意味・操作資格が異なるため、取得完了/有効/無効の一個の共有bitへ畳み込まない。

| 対象Claim | 必要な入力 | D2のみ変更時の判断例 |
| --- | --- | --- |
| 明細D1の単独合計 | D1の必要Scope・完全集合性、Rule版、演算単位 | D1の依存が確実に独立していれば維持可能 |
| D1×D2のJOIN由来集計 | D1とD2のRelation/Query演算、各Snapshot/Predicate/Scope | D2変更の影響を受け得るため再評価 |
| D2の未一致（LEFT JOIN） | D2全件の十分な観測、JOIN条件、時点 | 新しい一致行により旧未一致行が消え得る |
| D2が空であるというClaim | 対象Scopeに対する必要な否定条件・完全性 | 一件の取得や部分結果0件では確定不可 |
| 保存済み確定金額 | 業務確定Actionの歴史的Result・保存値と版 | 現在の元明細計算から自動上書きしない |

影響判定の入力となるのは、**Read Dependencies（対象と必要入力）**、**Change Trigger（何が変わったと実際に分かるか）**、**Current Binding（選択・版・取得Scope・時点）**である。結果は「根拠付き維持」「再確認が必要」「影響判定未確定」のような用途相対の診断候補とし、新しい固定Core enumにはしない。

### 13.3 更新通知は差分の意味に依存する

結果集合の更新は一般に追加だけではない。特に非単調な演算（LEFT JOINの未一致、NOT EXISTS、Top-N、集計、削除を伴う変更）では、後から届いた入力が以前の行を**撤回または置換**し得る。

- **単純再取得を初期の安全な基準**とする。既存Query/ProviderとCUIで比較し、正しい結果の同一性を独立Oracleで確認する。
- **局所再評価**は既存の演算依存が解決し、必要な入力が適切な版/範囲で揃う場合に限り比較する。
- **Provider差分購読・増分更新**は専門Pluginが実際に提供するEffect/Change Stream/Changelog契約へ接続する。全DB/外部APIがCDCや強い整合性を持つことは前提としない。
- **依存が不明な場合**は「無影響」と確定せず、再取得または未確認として扱う。無関係なD1を一律に捨てる過剰無効化と、依存するJOINの旧値を維持する過少無効化を対照する。
- **送信と確定の分離**：UIの楽観的な表示、Action送信、Provider到達、Commit Receipt、独立readと現在値は異なる観測。ACK消失でEffectが不明なら、表示の再取得で数値が一致しても当該Requestによる効果を証明したと主張しない。古い送信を無条件に再試行しない。

### 13.4 合成Headless受入の追加（#2582 RC-01〜09）

既存FC-01（H→D1 2行、D2 3行）を再利用し、以下の変更のみを投入する。

- **RC-01**：D2変更でD1単独集計は保持できる条件を示す一方、D1×D2のClaimは再評価する。
- **RC-02**：Query/IRのOpaque依存がある場合に、未知経路を「無影響」と誤診断しない。
- **RC-03**：同じFunction DefinitionのWindow A/Bは異なるInstance/epochを持ち、片方の親切替と遅延応答が他方へ誤作用しない。
- **RC-04/05**：LEFT JOIN仮NULLの撤回、NOT EXISTSの消失、集計の更新で、insert-only適用の誤りを検出する。
- **RC-06**：ACK消失・業務結果不明と、再取得された現在値の因果帰属を分離する。
- **RC-07**：Ruleの変更、説明・Rendererのみの変更、歴史的確定保存値を別々の再評価範囲とする。
- **RC-08/09**：別ProviderのSnapshot保証不在、stream後着訂正・Window結果変更を、許される範囲だけで表示する。

結果の診断は、**表示できる／選択できる／Operation Intent候補を作れる／実Hostで実行可能／業務完了を主張可能**という#2507の用途相対条件へ引き渡す。Renderer、AI CUI、SHOへの投影は同じ依存根拠を参照し、Consumerごとの第二正本を作らない。

### 13.5 外部実装との比較（参照資料）

- [Query keyと依存変数](https://tanstack.com/query/latest/docs/framework/react/guides/query-keys)、[変更後の無効化](https://tanstack.com/query/latest/docs/framework/react/guides/invalidations-from-mutations)：Queryパラメータの同一性、選択的な失効と再取得の分離を比較する。
- [増分化されたDataflowの更新](https://materialize.com/docs/fundamentals/concepts/arrangements/)：結果を(data,time,diff)の多重集合変更として捉え、追加・削除の両方を扱う設計の参考。
- [Dynamic Table Changelog](https://nightlies.apache.org/flink/flink-docs-release-2.3/docs/learn-flink/table_api/)：INSERT/DELETE/UPDATE_BEFORE/UPDATE_AFTERと集計の撤回・置換を扱う参考。

いずれも外部実装の公開資料に基づく比較であり、TEIが同等の変更ストリーム処理・Incremental View Maintenance Engine・実行時整合保証を備えることの証明ではない。比較から抽出したTEI側要件だけを採用判断候補とする。

## 14. 2026-11-02関連作業との接合と実証範囲

| 既存作業 | 今回再利用する成果 | 境界 |
| --- | --- | --- |
| #1706（closed） | 集合の粒度、重複、完全集合・NULLの参照試験 | 製品のFunction Context実装・実Providerを保証しない |
| #2500／Draft #2517 | 合成ViewChange→型付きOperation Intent候補の局所Go 12試験 | 正式Accepted Action、Go 1.26製品結合、実Commit未確認 |
| #2387／Draft #2577 | Query演算依存、実Go DeriveFromAnalysisへの合成入力テスト候補 | 実SQL Worker、同版Schema/SQL/Provider入力・業務対象の帰属未確認 |
| #2495/#2497/#2544 | 多重Facet・保存写像・演算と法則の適合 | 概念適合をTarget上の計算正しさへ昇格させない |
| #2493／Draft #2492 | 異質なDomain Pressure、AI-first→Headless→GUIの開発順序 | 既存Source/Acceptance/Gateを維持し、Domain別巨大Engineを前提にしない |
| #2516 | CUI/SHO/外部Consumerへの用途相対のProjection | すべてのConsumerに全Raw/機密情報を送出しない |

標準Plugin開発者が毎回メタ概念の分類やLawを手作業で列挙する必要がないことを目指し、既存Semantic Data/Relation/Ruleから利用可能な意味的性質を読み取る。組織管理などの通常DataへのアクセスはTEI内の一般的なData利用のままとし、特定保護情報の例外・外部送出・実Host Authorityの検証は既存の該当境界へ委譲する。


## 15. 用途相対の判定可能性と部分入力の意味（2026-11-02追補）

### 15.1 課題：完全性は必要十分条件の一部であり、常に一律ではない

Function Patternの利用資格は、全Data Contextに同じ`complete`や`stale`を付けるだけでは判断できない。必要なのは、**個別のClaim（どの結論を、誰が、どの時点・母集団について用いるか）に対する入力要件と観測の充足**である。

既存#2497の用途相対の保存条件を参照する。投影`P`からClaim`Q`を正しく決められるには、同じ`P`へ写る二つの許容入力が`Q`について異なる答えを持たないことが必要である。ただし、有限fixtureで反例が見つからないことは一般的な十分性証明ではない。#2507が所有するPattern適合、#2582が所有する変化後の再評価、#1063/#1706が扱う意味演算・集合の既存条件を利用し、この文書で別の論理推論器・第二の規則正本を設けない。

次の四つの質問を、CUIの用途別適合票で区別する。

1. **表示できるか**：手元にある結果の値・粒度・出典を、暫定性を含めて説明できるか。
2. **Claimを結論として扱えるか**：必要な母集団／入力版・演算規則が揃うか。条件付きの片側結論のみ成立するか。
3. **変更Intentを組み立てられるか**：業務対象ID・Action・変更対象・選択epochを明示できるか。集計行の逆更新は明示的なOperationが必要。
4. **実行・業務完了を説明できるか**：現行Hostの権限、Commit/Receipt、独立観測、Ownerが定義した完了条件を満たすか。上の三つから自動昇格させない。

### 15.2 片側結論と非単調な結論を判別する

初期の比較入力：親H、Line L1/L2は同額100、DeliveryはD1/D2/D3。Line/Deliveryは独立1:N。完全Line合計=200、両集合のJOIN後に単純SUMすると600、`SUM(DISTINCT amount)`=100。値の等しさとEntity IDの同一性は別。

| Claim／要求 | 部分入力から成立し得る説明 | 追加条件・未成立となる反例 |
| --- | --- | --- |
| `EXISTS matching Line` | 対象Scope/版で実際に一致する一件を観測すれば、そのSnapshotについて存在の証人になり得る | Delete/訂正/時点変更を越えた永続の存在証明ではない |
| `NOT EXISTS matching Line` | 観測した部分に見つからないことまで | 指定Scopeの検索が完了し、述語と時点が検証されるまでは不存在を確定しない |
| `COUNT>=1` | 同一対象と数える重複規則が明示された正しい証人があれば片側成立し得る | JOIN行や同一Entityの複数occurrenceを別実体として数えない |
| `SUM>=threshold` | 数量・金額が非負、重複/単位/粒度/時点が整合し、append-onlyに増える既知部分集合なら下界候補 | 負数、訂正/削除、JOIN増幅、異通貨換算・丸め差で下界保証を失う |
| `MAX>=threshold` | 順序が検証された観測値ひとつから条件付き成立し得る | 対象時点以降の削除／更新、数値比較の意味型不一致では再評価 |
| `TOP-N`の最終順位 | 観測済み部分の順位のみ | 未取得の上位候補が存在し得る。閉じた範囲や上界の根拠を要する |
| `LEFT JOIN unmatched` | 現在の取得部分で未一致と説明できる | 右集合の追加入力で仮NULL行が消える。SQL NULLと業務不存在を混同しない |
| `AVERAGE`／比率 | 観測した部分集団の値としてのみ | 母集団、分母・NULL・重複・時点・単位の条件が必要 |
| `exact SUM` | 必要な全Lineと計算規則を確認できた時点だけ | Deliveryの部分取得だけで独立Line合計まで失効させない |

**時間・履歴の条件**：append-onlyと仮定して得た「一度成立した片側結論」も、訂正、取り消し、データ削除、規則改訂、閲覧権限・母集団の変更を越えて自動保存しない。Event-time windowやProvider streamの閉鎖は、実際のwatermark／読取保証や専門Ruleで判断する。存在・不存在・集計は、観測時点に相対的なClaimとする。

**理論的な比較**：[CALM論文](https://arxiv.org/abs/1901.01930)、[The Coordination Criterion (2026)](https://arxiv.org/abs/2602.09435)では、単調性と調整・整合性の関係が研究されている。TEIでは当面「部分入力が増えた場合に、指定Claimが保持されるための条件」を**反証可能な評価項目**として利用する。TEIで任意のQueryの単調性を証明した、分散調整が不要になった、という製品主張は含まない。

### 15.3 判定が用途に相対的であることを示す例

同じD2の部分取得（1/3件）の下でも、次の区別を保つ。

- 「画面内にD2の観測済み1件を表示」：対象/時点/取得範囲を添えれば表示候補。
- 「現在のD2は業務上1件だけ」：残りの観測範囲が不明なら未確定。
- 「D1全2件の独立合計200」：D1の完全性と計算意味が適合すれば、D2部分取得とは独立して結論可能。
- 「D2全件を対象とする一括操作」：D2の対象集合・解決時点、Actionの要求に応じて判断。UIの表示件数から自動導出しない。
- 「D2の既知一件への明示ID編集候補」：Context epoch/ID/Actionが明示なら部分取得でも候補にはなり得る。Host現在権限や保存成功は別。
- 「外部HarnessにD2のRawデータを渡す」：画面の閲覧可能性とは別の開示条件で判断する。TEI内の通常組織Data利用を一律に特別保護へ移すものではない。

## 16. Data Context横断の単一ヘッドレス縦断シナリオ（統合計画）

- **統合実証の担当**：[Issue #2586](https://github.com/hat47x/tei-platform/issues/2586)。本書のHS-01〜12を、HJ-01〜07の受渡し票で段階的に検査する。個別Component/Driver/Scenarioの正本は各Issueに残す。

### 16.1 実証対象を固定する

複数担当の独立Referenceがあることと、それらが同一入力・同一意味で接続できることは別である。初回の共通合成Fixtureを**H・D1(2)・D2(3)**に固定し、同じSource/Schema/Query/Model/Rule/Context/Actionの版と来歴を保持する。別々のIssueが既に所有する解析・評価・更新・検証を再実装せず、**受渡し形式、対象Identity、必要な証拠の不足**だけを一枚の試験票で確認する。

構成上の縦断経路：

```text
Accepted Semantic Relation + Query Source/Schema/Dialect
    ↓ (#1126 / #2387: 解決可能な範囲と解析入力の版を提示)
RelationalIR / Output Grain / Source occurrence + Query Evidence
    ↓ (#2506: 関数ごとのContext参照とResult Sliceへ接合)
Function Context Definition → Instance + Selection epoch
    ↓ (#2507/#2497: Claim相対の可視化・操作適用性)
Pattern candidates: Table / Chart / Tree / Specialized
    ↓ (#2500: read-only ViewChangeをtyped Operation Intent候補へ)
Action candidate | ambiguous | unsupported | stale | unresolved
    ↓ (#1127/#2485/#1121/#1174: 別段のDriver/Host実実行・認可・Receipt)
Independent observation / Business outcome (#1131/#1180/#1125)
```

これは**既存担当をつなぐ検査経路**であり、第二Execution Runtime、第二Driver Scenario、独立Receipt/Authority、統合Context Registryを作る方針ではない。

### 16.2 共通受渡し票の最小欄（正式APIではない）

| 受渡し | 必須照合と失われる可能性がある情報 | 正本のOwner |
| --- | --- | --- |
| Semantic→Query | ID、Relation Role/Key、SQL Source/SHA・Dialect・Schema・Worker入力 | #1126/#2387 |
| Query→Context | Output Grain、Entity/Relation/Occurrenceとの対応根拠、lineage、Scope、Predicate、Page、Snapshot、NULLの由来 | #2387/#2506 |
| Context→Pattern | Context/Instance/epoch、Result Slice、Grain、Facet、Claim required inputs、適用/非適用理由 | #2506/#2507 |
| Pattern→Intent | 表示行occurrenceと業務対象IDの区別、Field/Edge Role、Action版、部分取得と全件Action要件 | #2497/#2500 |
| Intent→Effect | 実Target、Operation ID、現在認可、Rule/Expected version、Commit/不明な結果 | #1127/#1174/#2485 |
| Evidence→Consumer | 実行入力版、観測対象・時点・範囲、独立Oracle、残余、Consumerごとの開示条件 | #1131/#1180/#2514/#2516 |

既存`OutputField.Inputs`のField IDと`RelNode.Inputs`のOperation IDは別の名前空間であり、明示の根拠なしに機械的に結びつけない。実SQL Workerの応答が入力Source/Schema/Dialectと同じ試行に属するかは#2387の実装状況に従う。局所Go／Python Fixtureで構造が合っても、正式TEI Contextへの自動導出・実DB上の結果同等性は新たな実証が必要。

### 16.3 正例・負例を組にした試験票

| 対照 | 正しい結論 | 誤成功として検出するもの |
| --- | --- | --- |
| HS-01 独立集合 | D1=2、D2=3、JOIN表示=6を区別 | 6個の業務明細を新規作成したと扱う |
| HS-02 Claim別完全性 | D1完全ならLine合計は局所確定、D2部分はJOIN全体未確定 | Context全体を一律無効化／部分JOINを全件確定 |
| HS-03 NULLと未確定 | SQL SUMのNULL／LEFT JOIN仮NULL／未観測による結論無しを区別 | 全て同じJSON `null`に丸める |
| HS-04 版の入替え | Query/Schema/Rule/Actionの改版を対象ごとに検出 | 同じQuery IDだから旧解析結果を現在の根拠とする |
| HS-05 親選択A→B | 旧epochの応答をBへ混入させない | Aの表示行をBの業務対象へ逆結合 |
| HS-06 同一対象の複数出現 | L1のJOIN上3出現から明示IDつき単一Intent候補を作れる範囲を確認 | `rowIndex`や値一致だけで対象を確定 |
| HS-07 関係操作 | Relation occurrenceのunlinkとEntity deleteを区別 | 参照先の生存期間を変更してしまう |
| HS-08 非可逆投影 | 集約点の読取り・選択ができ、書戻しは明示Action待ち | 集約点の変更を任意の元行へ割り当てる |
| HS-09 Refresh/撤回 | LEFT JOIN仮NULL・NOT EXISTS・TOP-Nの旧出力を必要に応じ撤回 | append-only UI更新による残存行 |
| HS-10 ACK消失 | Result unknown／現在値の独立読取を別観測として表示 | 同値再読取を当該RequestのCommit証明へ昇格 |
| HS-11 Domain Plugin | unit・clock・operator lawを適合検査し専門計算へ委譲 | 数値型一致だけでtime step・solver差を無視 |
| HS-12 二Consumer | CUI JSONと異なるRendererの対象ID・Claim・欠落残余が一致 | 一方の表示仕様を第二Business定義とする |

**初回の縦切りはHS-01〜06のread-only出力**とし、従来の試験成功を再計測したかのように扱わない。HS-07〜12は独立して追加可能。将来のDriver/実DB/Host接続は#2485/#1180の既存Scenarioと組み合わせ、Ownerによる意味判断は既存受理経路で行う。

### 16.4 成功を測る単位

「Pattern名を認識した」ことより、次を測る。

- **情報保持**：SourceからClaimまでの必要Identity・粒度・時間・完全性・値域が保持された割合、LossとUnknownが説明された件数。
- **診断の判別力**：誤更新・誤集計・誤対象・不明効果を拒否できた件数と、正しい局所結果を過剰に拒否した件数。
- **統合負担**：既存Contractをそのまま利用できた接続、追加Adapter行数、重複定義、必要なCore変更、版の更新波及。
- **製品価値**：AIが一度のCUI往復で不足情報と次の検証を説明できるか、実装・設定・再試験の手間（生産性測定は#2018）。
- **実行証拠**：局所fixture、TEI正式Goテスト、実Worker入力、実Provider/DB、実Host、Owner受理を独立に記録。

## 17. Plugin Pattern拡張をData処理以外へ広げる条件

他領域の専門機能を、同一Data Contextに型を追加するだけで実現できると扱わない。#2493が検討する異種DomainのG0表現／G1解釈／G2評価／G3合成／G4実現の比較軸を参照し、ClockとEffectを含む契約を必要なConsumerだけに適用する。

- **組織・兼務/代理**：通常の組織DataはTEI内で一般のDataとして参照。多重Role/時点の意味を標準Tree/Graphへ投影し、表示の所属から現在Host Grantを自動作成しない。特定保護情報と外部開示は別条件。
- **BOM/工程**：同一Entityが複数Path occurrenceに現れること、数量展開とリソース時間計算の別演算、Gantt dragと実計画確定の区別。
- **IoT/監視と計算物理**：Sensor event-timeと取得時刻、一定tickと可変イベント、sample-and-hold、単位・座標、遅延、訂正。入力Producer/演算/専門HostのClock/Step/Failure契約を明示。可視化の一致をPhysics/実機効果とはみなさない。
- **長時間業務**：申請/承認・外部Handoff・Async Operationについて受付、進行、業務完了、再試行可能性・不明効果を別々のResult/Receiptとして表示。
- **文書・知識**：同一SubjectをCUI・SHO・図表へ投影し、出典・版・未決・開示可能範囲を用途相対で保持。資料に書かれた指示をHost実行権限へ引き上げない。

標準Plugin開発者に複雑なFacet/Law/Clock宣言を一律に要求することは目的ではない。**標準Semantic Data/Relation/型・Ruleに既存の意味がある場合、それを標準機能が活用できる**ことを優先し、専門Domainの要求が現れたときに明示的な演算・時制・Provider契約を追加する。

外部参照：[Flink Dynamic TablesのChangelog](https://docs.confluent.io/cloud/current/flink/concepts/dynamic-tables.html)、[Materialize data/time/diff](https://materialize.com/docs/fundamentals/concepts/arrangements/)、[Delta Lake Change Data Feed](https://docs.delta.io/delta-change-data-feed/)。これらは入力変更の表現に関する比較資料であり、TEIの実装・Snapshot保証・Changelog適合を示すものではない。

