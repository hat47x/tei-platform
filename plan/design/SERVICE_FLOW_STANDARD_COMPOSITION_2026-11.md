# 標準データ・プロセスからノードフローと軽量サービスを構成する

- **状態**：実装前の横断設計案。正式なDSL構文、標準型、公開API、Runtime、配備方式の採用決定ではない。
- **目的**：ノードフロー、APIマッシュアップ、Webhook、イベント連携、簡易サービスを、TEIの標準データ構造・汎用型・Function／Operation／Processの組合せで記述し、CLI・Web・Desktop・Batchから再利用できるようにする。
- **優先順位**：既存の製品P0（正式Source→実コード生成→通常CLIの実Provider、および非Plugin Studio Stage1→Stage2の実Windows受領）を維持する。サービス管理基盤の実装を先行必須条件にしない。
- **主なOwner**：標準Data／Facet [#2495](https://github.com/hat47x/tei-platform/issues/2495)、構造変換 [#2497](https://github.com/hat47x/tei-platform/issues/2497)、Operator [#988](https://github.com/hat47x/tei-platform/issues/988)、Process／Application [#1158](https://github.com/hat47x/tei-platform/issues/1158)、実Entry [#4495](https://github.com/hat47x/tei-platform/issues/4495)、Plugin解決 [#951](https://github.com/hat47x/tei-platform/issues/951)、実受領 [#2514](https://github.com/hat47x/tei-platform/issues/2514)。Ownerの判断を本書で代行しない。
- **参照する設計**：[起動・イベント・応答の意味](PROCESS_LIFECYCLE_TRIGGER_OPERATION_SEMANTICS_2026-11.md)、[実行モデルの投影](PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)、[Webの読込・書込API](WEB_READ_WRITE_API_ADAPTER_DESIGN.md)、[ETL・Batchの汎用化](ETL_BATCH_FUNCTION_COMPOSITION_DESIGN_2026-11.md)、[Ruleから実Builder・Providerへの縦断](PLUGIN_RULE_PRODUCT_VERTICAL_SLICE_2026-11.md)、[Web UIとフローの実装境界](WEB_STANDARD_COMPONENT_AND_AGENT_HARNESS_2026-11.md)。

## 1. 結論：フロー専用モデルではなく、標準要素の合成にする

ノードは処理の本体ではなく、**既存のOperationをあるフローに配置した出現**である。接続は汎用GraphのEdgeを用い、型、Role、Relation、順序、状態・副作用など必要な契約を付ける。画面の座標や線の曲がり方はProjectionの情報であり、業務上のRelationや実行許可ではない。

共通化するのは処理の意味と入出力契約である。ノード画面、DSLの関数チェーン、決定表、業務プロセス図、API定義、Batch定義は、それぞれ意味を保てる範囲で同じ標準要素を参照する。表示形式を切り替えたことだけで、Ownerが受理した意味を書き換えたり、新しい意味を補ったりしない。

専用のFlow Runtime／Event Loop／Graph Compiler／Provider Resolver／Registry／Expected Storeは最初から作らない。TEIの既存GeneratorとHost／Providerに渡せない具体的な反例が揃った場合にだけ、必要な抽象の追加をOwnerとADRで判断する。

## 2. 型・構造・意味・表示の境界

| 層 | 再利用する標準要素 | フローでの役割 | 混同してはならないもの |
| --- | --- | --- | --- |
| 値・データ型 | Scalar、Record、参照、結果等の既存の型・Facet | ポートで受け渡す値と制約 | 型の宣言と実際に読み取った値 |
| データ構造 | Collection、Relation／Table、Tree／Document、Graph、時系列・イベント等の構造候補 | データの形、粒度、向き、多重性、時間軸 | Graphという構造と実行できる処理 |
| 意味 | Role、Trait、Operator Law、Function／Capability、RunPlan、Validation、Effect | 変換、呼出し、状態遷移、制約 | 宣言されたCapabilityと現在の実行資格 |
| 構成 | Operation参照、Node出現、Port束縛、Edge出現、Process参照 | 処理の接続と再利用 | 同じOperator定義と複数の配置先 |
| 公開 | Trigger、Entry、Service、入出力契約、Policy、Target Binding | API・イベント・CLI等の受付形 | 公開予定と実際に動くEndpoint |
| 表示 | Canvas、Table、DSL、Trace、Inspector | 人間とAIの編集・理解 | 表示上のID・選択位置と更新対象のID |

`Stream<T>`、`Event<T>`、`Result<T,E>`、`Operation<I,O>`などは**説明用の型表現**である。新しいCore TypeKindとして追加する決定ではない。現在の製品構文で表せないFacetは「未対応」とし、標準Plugin／明示的な契約記述で検証してから採否を決める。NULL、欠損、未取得、非開示、部分取得を一種類の空値へ丸めない。

### 識別子の分離

- **Operation ID**：再利用する演算・操作の定義。版とProviderの要求を持つ。
- **Node occurrence ID**：特定のFlowでそのOperationを使う一回の配置。Subflow内外で別の出現になり得る。
- **Port ID／binding**：定義上の入力・出力と、今回の利用先で束縛された型・Context。
- **Edge occurrence ID**：接続の同一性。source／target、方向、Role、多重性、意味属性を保持する。同端点・同ラベルの複数辺を潰さない。
- **Business Relation ID／Process Action ID**：受理済み業務関係・工程のID。Node／Edge／描画線とは別に保持する。
- **View／Selection ID**：画面上の対象。ここから意味IDや編集権限を推定しない。

## 3. Process、Operation Graph、実行計画の対応

| 対象 | 表すこと | 保存する接続 |
| --- | --- | --- |
| Process Definition | 業務上の目的、順序、分岐、例外、状態遷移 | Action ID → 参照するOperation／Subgraph（0..N） |
| Operation Graph | 具体的な処理の合成、Port／Edgeの意味と制約 | Node occurrence → Operation ID、Port binding、Edge ID |
| Execution Plan | Target／実Provider／起動入口／依存関係／並列性／効果境界 | 生成元のOperation／Source／選択されたCapability |
| Trace／Receipt | 実行されたAttempt、実Provider、結果・効果・消費先 | 実Attempt → 生成Artifact → 独立Consumer |

Process上の一Actionを複数のNodeで実現してもよいし、一つのOperationを複数のProcessから参照してもよい。ただし、Process上の分岐と、Graph上のData依存が同じ意味であると自動判定しない。

接続は少なくとも **Data／Control／Event／State／Error／Effect** の意味を区別して扱う。これはCore enumの追加を要求するものではなく、Roleと契約を用いるための分類である。Data接続だけから、外部書込の順序や認可を導かない。循環は初期値、遅延、状態更新時点、収束または停止条件が宣言される場合だけ適格とし、無条件の閉路は診断する。

## 4. サービス定義と公開境界

ServiceはOperation Graphの別名ではない。**利用者から見た受付契約と、その内部処理への束縛**を表す。少なくとも次の情報が必要になる。

| 契約 | 内容と確認する境界 |
| --- | --- |
| Service identity | Service ID、Revision、Source occurrence、公開対象の範囲 |
| Trigger / Entry | HTTP、Webhook、Schedule、Message、CLIなどの受付形。宣言と実配備を分離 |
| Interface | 入出力の型、必須・欠損・NULL、スキーマ版、エラー、応答方式 |
| Operation binding | Process Action／Operation Graphへの参照、入力Mapping、結果Mapping |
| Data Context | Scope、粒度、キー、Snapshot、完全性、取得時刻、外部Source状態 |
| Execution policy | 同期／非同期、順序、並列数、Timeout、Retry、Cancellation |
| Authority / disclosure | 呼出し主体、現在Grant、操作・行・項目・テナント制約、公開可否 |
| Effect / recovery | 純粋Read、DB変更、外部送信、冪等性、Fence、Receipt／結果不明、補償 |
| Delivery / evidence | 実Target、選択Provider／Plugin Lock、Artifact、実Attempt、実Consumerの観測 |

**サービスの論理定義、実配備先、実際の稼働状態は別の資格**である。Go単一バイナリを第一経路とし、外部Hostにコード生成または契約付きAdapterを配置する。クラウド／エッジの監視・更新・共同管理は後続の管理機能として扱い、独自の常駐サービス基盤を前提にしない。

HTTPの公開では、既存のWeb Read/Write契約に従い、許可された行ScopeとTenantの制限をFilter／Aggregate／Pageより先に実現する。受信したURL、SQL、認証情報を直接別Providerへ流す汎用中継器にはしない。外部API仕様の取り込みは**契約候補の生成**であり、操作の意味受理や公開許可を自動的に与えない。OpenAPI等の出力も、実装したサブセットと版を正確に広告する。

## 5. 標準Operator・サービス部品の提供範囲

用途別に別々のOperator本体を重複実装せず、同じOperationをノード、関数チェーン、プロセス、API／CLIの各入口で参照できるようにする。

| 部品群 | 最初に用意する要素 | 後続で検証する要素 |
| --- | --- | --- |
| Data／変換 | Select、Map、Filter、Validate、Convert、Join、Aggregate | Window、Stream Join、差分・増分変換 |
| Process／判断 | Branch、Rule、Subprocess、明示的なJoin | 並列回復、補償、長期実行 |
| API／連携 | HTTP受信／応答、HTTP Client、入力／結果Mapping、明示的な認証Adapter | Webhook署名、イベント公開、GraphQL／OData等 |
| Data Provider | 既存の型付きQuery、固定SQL、read-only Lookup | Transaction／変更、Object Storage、Broker |
| 運用／検証 | Trace、Assert、Mock、Result分類、Timeout | 再試行、重複排除、Circuit Breaker、Replay |
| テンプレート | Read-only API Facade、Lookup Mashup、Schedule Read、軽量CRUDの設計雛形 | イベントブリッジ、状態付きWorkflow、クラウド配備 |

`Retry`などは画面上ノードとして表現できても、意味上はOperationの実行ポリシーになり得る。Node専用の実装を正本にしない。DB、メッセージ、外部SaaSなどへの接続はProvider Pluginが公開するCapabilityと正確な版を選択し、実行先のHostが現在の認可を再確認する。

## 6. 最初の縦断例：注文統合照会

受付は型付きの `GET /orders/{id}/summary`。内部では既存注文Queryと二つの外部Lookupを組み合わせ、項目ごとのSource・Scope・完全性を保ったまま、許可された統合結果だけを返す。どの情報も取得できない場合に「注文なし」と決めつけない。

1. **入口**：Routeと型付きInputを結び、実Targetが受付可能であることを確認する。
2. **受理前検査**：現在Principal／Tenant／行・項目の公開資格、入力の型、対象Scopeを検証する。
3. **Operation**：注文Queryと顧客・配送Lookupを接続し、取得済み・未取得・不在・権限拒否を区別する。並列にできるのは依存契約が明示された場合に限る。
4. **結果**：Result Mappingで取得完全性と部分成功を保存する。未許可項目やその推測材料をResponseへ出さない。
5. **実行証拠**：元Source／型／Graph／生成Go／Artifact／選択Provider／HTTP受領を別に記録する。Mockの成功を実Provider成功へ昇格させない。

最初の実装は外部通信を伴わない**固定Fixtureによるread-onlyの二〜三Operation**に絞る。同一Operationの直列関数経路とGraph経路について、定義した同値範囲で結果・欠損・型を比較する。実HTTP受付、外部API、認証、実Providerはそれぞれ独立の受入段階とする。

## 7. 反例と受入票（独立Expectedへの引渡し）

既存のScenario正本 [#1180](https://github.com/hat47x/tei-platform/issues/1180) と独立Expected [#2514](https://github.com/hat47x/tei-platform/issues/2514) に、次の**候補**を追加する。新しいTest Registryや第二のExpected正本は作らない。

| ID | 反例 | 期待する扱い |
| --- | --- | --- |
| SF-01 | 同じOperationを二つのNode occurrenceから参照 | 定義IDは同じでも出現IDとPort束縛を別に保持 |
| SF-02 | 同端点・同ラベルの二Edge | 各Edge IDと多重性を保持し、描画線IDと混同しない |
| SF-03 | Nodeの表示位置だけを変更 | 意味・生成結果の再受理を要求せず、View差分として扱う |
| SF-04 | Data接続だけで外部Updateの実行順を推定 | 明示Control／Effect契約がなければ拒否 |
| SF-05 | Delay／初期状態のない循環Graph | 実行計画を作らず診断 |
| SF-06 | 部分取得・未取得をNotFoundや空集合に変換 | 元状態と完全性を保持し、確定不在を主張しない |
| SF-07 | 非公開項目をFilter／Sort／Countで推測 | 実行前のScope検査で拒否し、事後の項目削除で済ませない |
| SF-08 | ProviderをBに選択したのに実際はAが応答 | Resultが同値でも実Provider不一致として失格 |
| SF-09 | 旧Source／旧Lock／A→B→Aの遅着結果を採用 | World／Model／Artifact・実Attemptごとに失効を照合 |
| SF-10 | 外部書込後にReceiptが取得できず自動Retry | EffectUnknownを保持し、無条件の再実行を拒否 |
| SF-11 | 外部API仕様だけから意味受理・公開許可を付与 | 契約候補に留め、Owner／現在Grantの判断を別にする |
| SF-12 | Mock HTTP成功を実Host・実Windows・実Consumer受領とみなす | 実Target／実Process／実Consumerを別資格として未受理 |

## 8. 実装順と既存計画への接続

既存のFlow N0〜N4とStudio W0〜W4に重ねて進め、新たな並行ロードマップを作らない。

| 接続点 | 最小の実装・検証 | 次へ進む条件 |
| --- | --- | --- |
| N0／構造契約 | 既存RuleとData／OperationからNode／Port／Edgeのread-only投影。Identity・型・FacetのFit/Gap表 | 生成元と投影先の対応が取れ、意味の自動補完がない |
| N1／型付きGraph | 既存 `quantity_positive` と純粋変換を二〜三Operationで構成。SF-01〜06の負例を独立Expectedで確認 | Port整合、欠損、循環、重複Edgeを区別できる |
| N2／実生成・Consumer | CLIから実Goを生成し、現行Builderの実消費byteとArtifact・別Process結果を照合。R2／R3へ接続 | 旧Go delegateだけでなく処理の実質を生成したと実証 |
| N3／Studio | 既存Inspectorからread-only Canvas、型とSource位置・Traceへジャンプ。W0〜W4の製品資格は別 | Viewのみの変更と意味変更を判別し、Pin／Dirtyを混同しない |
| N4／外部Service | API Contract→固定Mock→実HTTP Host→正確に選択されたProvider→独立Consumerへ段階的に接続。SF-07〜12を検証 | Scope／権限／部分取得／効果未知／実配備を別々に検収 |

最初の差分は**新規RuntimeでもGUIでもなく、標準Data／OperationとProcess Actionの参照を持つread-onlyのNode／Port／Edge投影＋CLI実生成**とする。作業結果を既存のRule縦断、Web API、Plugin／Provider、Studio UIのOwnerへ引き渡す。後続でサービスをクラウドへ配備できるようになっても、同じ標準意味を外部Hostへ写す方式を維持する。

### 未決事項

1. Portの型・時間・粒度・完全性をどこまで静的検証し、実Providerで何を再確認するか。
2. Message／StreamのEvent Time、Processing Time、再配送、順序・Exactly-onceをどの能力として契約化するか。
3. Flow編集から元のMarkdownへ戻せる範囲と、Lossを表示してread-onlyに留める境界。
4. 配備環境・Secrets・Tenant別のBindingを、既存Plugin Lock／Application Profile／Host設定のどこに置くか。

いずれもOwnerの実Consumer観測とADRによって決める。現時点では構想を正式な製品機能の完成、外部Hostへの適合、業務Meaning受理として扱わない。

## 9. ノードビューをプラグインの関係・配線にも共用する（2026-11-15追補）

### 9.1 表現型を共通化し、関係の意味は共通化し過ぎない

**Node Viewはプログラミング専用画面ではなく、型付きGraphを見せ、参照し、適格な接続だけを編集する標準表現型**とする。Operatorフロー、業務Process、Plugin合成、API／Service、UI構成、SUIの関係図、Build依存、実行Traceを同じ描画・選択・レイアウト部品で扱えるようにする。ただし、図に含まれる意味と編集操作は各モデルの契約に従う。SUIの自由な線を自動的に実行フローへ、Pluginの依存線を自動的にRuntime呼出しへ変換しない。

表示用のNodeやPortを新しい業務データ型・意味の正本にしない。標準GraphのSubject／Relation／Occurrenceと、そのPort・Edge Roleを投影する。原データの方向・多重性・同端点の複数Edge・識別子を保持し、グループ化や階層表示で省略したものは明示的な損失とする。Layout／折り畳み／ズーム／配色はView側の設定であり、これだけを変えてもPlugin Lockや生成コードの意味受理をやり直す理由にはならない。

| Node Viewの利用領域 | Nodeとして表示する対象 | Edge／Portで示す意味 | 正本・制御 |
| --- | --- | --- | --- |
| Operation Flow | Operator定義／Node occurrence／Subflow | Data・Control・Event・State・Effect | Function／Operation／Process、各契約 |
| Plugin Composition | Plugin package／貢献単位／Contract／Capability／Provider | requires・provides・implements・selects・resolves | 既存Plugin Contract、Application Profile、Resolver、Lock |
| Build／Delivery | Generator／Build recipe／Artifact member／Target | consumes・generates・includes・excludes | Build Intent、工程別BOM、Artifact／Toolchain証拠 |
| Runtime Wiring | Entry／Host／selected Provider／Operation／Hook | binds・invokes・subscribes・handles・observes | 実Host Binding、現在Grant、Trace／Receipt |
| Service／API | Trigger／Endpoint／Operation／Data Source | routes・maps・publishes・responds | Service／Application Operation、API Adapter |
| SUI／Studio | モデル要素／参照／表示用グループ | relation・reference・selection | 標準Data／Graph、Sourceと表示Projection |

同一のPluginが二つのTargetで異なるProvider選択を持ってよい。同じContractに複数の候補Providerが存在してもよい。**一枚の図で「要求している」「選択した」「成果物に含まれた」「実際に呼ばれた」を一本の確定線に畳まない。**

### 9.2 Plugin配線は、宣言・解決・実現・観測を切り替えて見る

Plugin Nodeは少なくともPackage／Contribution／Contract／Implementation／Bindingの識別を区別する。必要に応じてBuild専用Generator、実製品に含めるRuntime Provider、Language Plugin、Data Provider、Target Generator、UI部品、Hookを表示する。Pluginの名前が同じでも版・提供Contract・環境・Targetが違えば、同一実装として束ねない。

| 接続面 | 表示すること | その線だけでは主張できないこと |
| --- | --- | --- |
| Declared | Plugin AがContract Xを要求し、Plugin Bが実装候補を提供する | Bが選択済みである |
| Selected／Resolved | Application／Environmentの選択と既存Resolverの確定結果、exact版・Lock | Build成果物への同梱、実呼出し |
| Built／Delivered | Build専用とRuntime用の依存、収録されたImage／Asset／Artifact | 実Hostで呼出し可能、現在Grantを満たす |
| Invoked／Observed | 具体的なEntry／Provider Call／Hookの呼出し、Attempt／Trace／Receipt | 結果が業務意味として受理された、効果が確定した |

この四面は同じ図に色や線種で重ねることも、別々のレイヤーとして表示することもできる。ただし**重ね合わせに必要なSource Revision、Selection Scope、Plugin Lock、Artifact Revision、Attempt、観測完全性**を常に表示し、取得できない関係は「未確認」「部分取得」とする。Declared関係が無いことと、実際の呼出しが無いことも別である。

Providerの選択は既存[#951](https://github.com/hat47x/tei-platform/issues/951)・[#1016](https://github.com/hat47x/tei-platform/issues/1016)に委ねる。Node View上で矢印を引く操作が新しいResolverとなったり、単に線がつながったという理由でProvider／Permissionを有効化したりしない。工程別の依存集合は[#3267](https://github.com/hat47x/tei-platform/issues/3267)、実呼出しは[#1251](https://github.com/hat47x/tei-platform/issues/1251)、Hook／Lifecycleは[#4198](https://github.com/hat47x/tei-platform/issues/4198)、影響する部分だけの再検証は[#2600](https://github.com/hat47x/tei-platform/issues/2600)を利用する。

### 9.3 配線操作の契約

**閲覧**と**配線の変更**は別の権限で扱う。初期段階では、現在のContract／Plugin Lock／Build BOM／実Callをread-onlyで投影する。編集を許す場合は、線のドラッグ操作を「どのSourceのどの設定をどう変更するか」という型付きPatch候補に翻訳する。

1. 接続元と接続先の安定ID、Port Role、正確なContract版、環境・Target・工程、必要なTrait／Capabilityを確定する。型適合だけでなく、単一・複数の接続数、循環制約、実行位置、Trust境界、Side Effectを検査する。
2. 意図を既存Application Profile／Plugin選択／Operation／Processのうち該当する**唯一の正本**への差分として提示する。どこへ書き戻すか確定できない図、複数Sourceへまたがる曖昧な関係、read-onlyのTraceにはPatchを作らない。
3. 変更対象のSource Revision、OwnerのMeaning、現在Grant、既存選択・Lockを再検査する。表示上のドラッグやAIの接続提案から自動で実効化・インストール・通信・配備しない。
4. 保存後は既存Resolver／Buildで再評価し、Declared→Selected→Built→Invokedのどの面まで確認できたかを表示する。接続の変更・ロック版変更・外部Endpoint変更は影響先だけを失効させ、Layoutだけの変更は意味を失効させない。

必要なProviderが未選択ならPortを未接続として示す。複数候補があれば候補を並べて表示し、任意の一つへ自動配線しない。Build時にだけ必要なGeneratorがRuntime成果物へ存在しないことは正常になり得る。一方、Runtimeで必要なProviderが収録されない構成は、不足として診断する。異なるTransportやTargetをまたぐ接続はAdapter／Boundaryを明示する。

### 9.4 StudioとAI協業のUI

共通CanvasではNode／Edgeを選ぶと、対応するSource位置、契約、Lock、Build member、TraceをInspectorで横断表示する。複雑なPlugin構成では、Package単位→Capability単位→Provider実装／Port単位の順に展開し、環境・Target・工程・版で絞り込めるようにする。接続候補の検索や不適合箇所のハイライトは、GraphエディタだけでなくPlugin管理画面、サービス構築画面にも共通提供する。

人間とAIは同じ固定された対象（Workspace／Subject／Element／Occurrence／Selection／Source Revision）を参照する。AIが配線を提案した場合、**変更する契約・入力Source・想定する差分・再生成対象・必要な権限**をPreviewし、現在の選択と比較してから適用する。Pinは閲覧対象の固定であり、古いLockやGrantを復活させるものではない。

標準Node Viewの実装は既存Web UI／Canvasの比較Consumerとして扱う。表示用Web ComponentとGo側の契約・Rendererを一体化させず、実Windows WebView2での受領はStudio W0〜W4の別資格を使う。

### 9.5 既存計画へ渡す二つの最小検証

**PV-N0（CLI-first、read-only）**：合成Plugin AがContract Xを要求し、BとCがそのProvider候補であるFixtureを用意する。既存Plugin Inventory／Resolver／Lockから、A→X（要求）・B/C→X（提供候補）・X→B（環境ごとの選択結果）をそれぞれ別の型付きRelationとして投影する。Build-only Generator DとRuntime Provider Bを同じGraphに配置し、Dの非同梱とBの同梱を区別する。元のResolver結果と投影で選択結果・未選択・曖昧さが一致しなければ停止する。既存[#951](https://github.com/hat47x/tei-platform/issues/951)／[#1016](https://github.com/hat47x/tei-platform/issues/1016)／[#3267](https://github.com/hat47x/tei-platform/issues/3267)へ引き渡す。

**PV-N1（編集資格は後続）**：同じGraphをStudioでread-only表示し、契約のPortを選ぶと選択済み実装・Source・Lockへ移動できることを確認する。正式な配線編集は、Application Profile上の一設定を実際に更新できるSource Mappingと、変更前後のResolver／Artifact／Consumerの受領が揃うまで保留する。既存[#1124](https://github.com/hat47x/tei-platform/issues/1124)／[#1824](https://github.com/hat47x/tei-platform/issues/1824)へ引き渡す。

| Case | 独立Expected候補 | 失格とする誤り |
| --- | --- | --- |
| PV-01 | Plugin packageとCapability／Providerを別Node Identityにする | 同名で統合して必要な実装を隠す |
| PV-02 | 複数候補B/Cと、選択されたBを別のEdgeとして表す | 候補があるだけでCも選択済みとする |
| PV-03 | Environmentを変えるとBindingとLockが正しく切り替わる | 古い選択・古い版を現在表示する |
| PV-04 | Build-only DはRuntime非同梱、Runtime Bは同梱が必要 | 全Pluginを一括同梱、またはB欠落を合格とする |
| PV-05 | 同じResultでも、選択B／実Call Aを不一致とする | 結果一致から実Provider一致を推測する |
| PV-06 | Layout変更とContract／Provider変更の失効範囲を分ける | 座標変更で業務Meaningを再受理／実接続変更を無視 |
| PV-07 | 参照GraphのEdge ID・Role・多重性を保つ | 同端点の二Edgeを一本にまとめる |
| PV-08 | Portの型は適合してもGrant／Effect／Trust境界が未確認なら実行不可 | UIで接続できたことを実行許可とみなす |
| PV-09 | 必要なSource Mappingが曖昧ならread-onlyのままにする | 描画線から元Sourceを推測して書き換える |
| PV-10 | 部分取得・非開示・古いTraceを現在の実配線と区別する | Traceが無いことを未呼出しの証拠とする |

PV-01〜10は既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)に渡す反例候補であり、現時点で実行済みの検証ではない。最初の実装差分は**CLIでのPlugin関係read-only投影**に限定する。これとOperator FlowのN0/N1は共通のGraph表示契約を利用するが、互いを先行必須条件にせず並行して検証する。
