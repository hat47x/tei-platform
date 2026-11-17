# 標準データ・プロセスからノードフローと軽量サービスを構成する

- **状態**：実装前の横断設計案。正式なDSL構文、標準型、公開API、Runtime、配備方式の採用決定ではない。
- **目的**：ノードフロー、APIマッシュアップ、Webhook、イベント連携、簡易サービスを、TEIの標準データ構造・汎用型・Function／Operation／Processの組合せで記述し、CLI・Web・Desktop・Batchから再利用できるようにする。
- **優先順位**：既存の製品P0（正式Source→実コード生成→通常CLIの実Provider、および非Plugin Studio Stage1→Stage2の実Windows受領）を維持する。サービス管理基盤の実装を先行必須条件にしない。
- **主なOwner**：標準Data／Facet [#2495](https://github.com/hat47x/tei-platform/issues/2495)、構造変換 [#2497](https://github.com/hat47x/tei-platform/issues/2497)、Operator [#988](https://github.com/hat47x/tei-platform/issues/988)、Process／Application [#1158](https://github.com/hat47x/tei-platform/issues/1158)、実Entry [#4495](https://github.com/hat47x/tei-platform/issues/4495)、Plugin解決 [#951](https://github.com/hat47x/tei-platform/issues/951)、実受領 [#2514](https://github.com/hat47x/tei-platform/issues/2514)。Ownerの判断を本書で代行しない。
- **成果物中心の設計との接続**：[成果物要求と実現方式の選択](DECLARATIVE_RESULT_PROCESS_COMPOSITION_2026-11.md#11-一つの意味モデルから複数の実現方式を選ぶ2026-11-15追補) §11。NodeのOperator出現と、直接解釈・Go生成・DB委譲・外部Providerの選択は別の関係とする。最初は既存Rule一件のEvaluator／生成Goを独立Expectedで照合し、Plugin配線のPV-N0とは共通Graph表示契約を使いつつ別々に検収する。図上の接続から実行方法・現在Grant・実Callを推測しない（実装前設計）。
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

## 10. 接続できることの意味を、構成と実行の両側から検証する（2026-11-15横断統合）

### 10.1 次の目標は配線そのものではなく、配線の成立条件を説明すること

§9でノードビューの共通化を整理した後、宣言的な構文・標準データ構造とFacet、成果物要求からの計算合成、Pluginの実接続証拠、Processの因果表示が他の設計で進んだ。これらを別のGraphや契約へ複製せず、**「この接続は何を保証でき、何が未確認で、どこまで実現されたか」**に、同じ対象について答えられるようにする。

| 段階 | 確認すること | 成立しても主張できないこと |
| --- | --- | --- |
| 候補の発見 | 必要なOperation／Capabilityに提供候補があるか | 契約適合、選択済み、実行資格 |
| 意味の適合 | 出力が次の入力の要求を満たすか。型・Role・粒度・時間・完全性・演算則を照合 | 成果物への収録、現在Grant |
| 構成の実現 | 既存Profile／Resolver／Lock／Generator／Builder／Targetで何を選び実装したか | 実際のEntry／Provider呼出し |
| 実行の受領 | 実Call、Attempt、Result、Receipt、独立Consumerの受領が一致するか | 別の時点・Audience・Targetの現在資格 |

これらは新しい永続Schemaや検証Engineではなく、既存Ownerの結果を問相対に照合するための検収の切り方である。例えばPortの型が一致しても、月次集計を日次明細として使うこと、部分取得を全件として使うこと、Pluginの提供候補を選択済みとして描くことは認めない。正の不一致はmismatch／rejected、必要な証拠が不足すればunknown／unqualifiedとして区別する。

### 10.2 接続には供給側の保証と利用側の要求がある

接続候補A.out → B.inについて、Aが提供できる保証とBの要求を既存の型・Facet・Role・Operation契約から取り出し、以下の観点を必要な範囲で比較する。これはPortの新Core型や万能制約Solverの採用宣言ではない。

| 観点 | 比較する内容 | 不適合の例 |
| --- | --- | --- |
| 型・構造 | Record／Collection、単位、必要Field、値域、数量の尺度 | 同じDecimalでも単位や丸めが異なる |
| 同一性・粒度 | Subject、業務キー、Occurrence、Cardinality、Bag | 注文明細を一注文として扱い、重複を消す |
| 時間・版 | 有効時点、取得時点、Event Time、Snapshot、Source Revision | 過去Snapshotで現在の在庫を判断する |
| Presence・Coverage | NULL、欠損、未取得、非開示、partial、否定依存 | 一部しか取得していないのに不存在を確定する |
| 評価 | 必要Read、短絡、失敗、評価順序、決定性 | 同じ結果でも不要な外部Readを追加する |
| Effect・Trust | 読取／書込、公開Scope、通信境界、Secret、再試行 | HTTP入力を無許可でDB更新へ送る |

接続検査は**問相対**である。純粋な数値比較に不要な全時系列情報を要求すると再利用性が下がる。一方、存在しないことを確定するOperationには取得範囲と完全性が必要である。未宣言のFacetに暗黙の既定値を補わず、その問いに必要なら不足とする。

**A→BとB→Cが局所的に適合しても、A→Cの意味保存を無条件には主張しない。** Bが粒度・順序・失敗・開示範囲を変え得るためである。Subgraphを一部品として公開するときも、内部を隠すことと、外部向けの保証・Effect・Failureを省略することは異なる。既存のMapping契約（#2497）で保存・損失・逆編集資格を検査する。

### 10.3 Adapterは自動配線の抜け道にしない

JSONから型付きRecord、RESTからOperation、RecordからAPI Responseなどの明示的なAdapterを候補として示せるようにする。ただし、形を変えられることと、情報・意味・権限を保てることは異なる。

最初は既存のOperatorまたはProviderが公開する**一段の明示Adapter**に絞る。複数段の最短経路探索、任意Providerの生成・インストール、外部Endpointへの自動公開は後続判断とする。候補を見せるだけで実ProfileやSourceを変更しない。Adapterを挿入した場合は、それ自体のOperation／Provider ID、Mapping、追加Read、変換による損失、Effect、Target境界を追跡し、直結と同等だったと扱わない。

### 10.4 同じ読み取りサービスを三つの面から見て一つの縦断例にする

§6の注文統合照会を利用し、業務上の新しい意味受理や実HTTPサービスの完成は主張せず、同じ対象に対する複数の問いを検証する。

| 表示面 | ノード／接続 | 最初に説明すること |
| --- | --- | --- |
| 処理 | Order Lookup→顧客・配送Lookup→Join／Result Mapping | 型、粒度、欠損、partial、必要Read、Error |
| Plugin構成 | LookupのCapability要求→候補Provider→選択Provider | exact版、Scope、Profile、選択の曖昧さ、工程別同梱 |
| 公開サービス | HTTP Trigger／Entry→Operation→Response | 公開契約候補、認可・開示、Target／Consumerの未確認箇所 |

三つの独立した意味正本を作るのではなく、標準Data／Function／Operation／Plugin／Processの別投影とする。ノードを選ぶと契約、Source位置、Resolver結果、Artifact、Traceに進めるが、型適合からProvider選択やHTTP公開資格を推定しない。

同じLookupをCLIとHTTPで再利用しても、呼出主体、行・項目Scope、応答・Secretの開示は入口ごとに異なる。CLI成功をHTTP公開の受理に、HTTPのMock成功を実Providerへの到達に変換しない。API仕様へ投影する際は実対応範囲だけを明示する。

### 10.5 コード生成と汎用Operatorの解釈を選び分ける

すべてのノードを個別にコード生成する必要はない。小さな純粋FilterやMapping、診断用のGraph走査は、型付きパラメータを**既存の汎用Operator**へ渡して実行できる。一方、静的結合、性能、外部Target、必要なProvider集合の固定が必要な部分は、既存GeneratorでTarget-nativeコードにする価値がある。

| 実現 | 向く条件 | 確かめること |
| --- | --- | --- |
| 汎用Operatorを解釈 | 既存の受理済みOperatorで足り、実Hostがその能力を提供する | 実装版、入力・出力・失敗、実Provider呼出し |
| Target-native生成 | 静的Closure、性能、Target固有API／配布が重要 | 意味・Generator→生成byte→実Builder入力→Artifact→Consumer |
| 明示的な混合 | 一部は生成、一部は汎用Operator、外部接続はProvider | 境界ごとの型・Effect、生成と委譲の区別 |

**汎用解釈は新しいTEI Hostを増やす提案ではない。** 既存Operationと選択されたTargetの実現方法を選ぶ話である。解釈方式ではコード生成工程を省けても、Version、Grant、結果、独立Consumerの検証は省けない。生成方式では手書きGoへ単純委譲した箇所をTEI自身の生成実績と数えない。両方式の比較では値だけでなく、Read、Failure、Effect、観測Scopeを独立Expectedで照合する。

### 10.6 最初の実装Packetと採否

**NW-Q0（CLI-first・read-only）**：一つの既存Operationと一つのPlugin接続候補について、同じScopeの根拠が揃うかを見る。InspectSubject／Rule Reader／management.Report／composition.Selectionを既存の責務のまま参照する。Legacy InventoryとComposition Selectionを名前やResolvedフラグだけで結ばず、不足する関係はunknownにする。専用Registry、Graph Compiler、CLI Flagは作らない。

**NW-Q1（純粋変換の合成）**：既存のquantity_positiveと実在する純粋Operatorを対象に、型・Read Footprint・結果と、Go生成または汎用評価の検証可能な範囲を照合する。表示だけの変更、参照Attributeの変更、実装A→Bの選択変更で失効すべき資格を分ける。旧生成CLIの選択失効は既存[#5054](https://github.com/hat47x/tei-platform/pull/5054)のOwnerに任せる。

**NW-Q2（サービスへ接続）**：read-only API Facadeの契約候補→固定Fixture／Mock→実HTTP Host→選択Provider→独立HTTP Consumerの順に証拠を増やす。Secrets、外部認証、Webhook、非冪等Effectを初期の必須条件としない。部分取得、未選択Provider、Scope不足では黙って実行を続けない。

| Case | 独立Expected候補 | 不合格とする誤実装 |
| --- | --- | --- |
| NW-01 | 型一致でも粒度違いの接続を拒否 | Record型だけで適合と判定 |
| NW-02 | partialな結果からNotFoundを確定しない | 空集合から不存在と判断 |
| NW-03 | 必要Facetが未提供なら不足と表示 | 時点・順序を無断補完 |
| NW-04 | Adapterの前後でRead／Loss／Effectを比較 | 形式変換を無損失とみなす |
| NW-05 | B/C候補・選択B・実Call Aを区別 | 結果一致でProviderも一致とする |
| NW-06 | 同一OperationでもCLI／HTTPのAudience／Scopeを別にする | CLIで読めるからHTTPでも公開 |
| NW-07 | LayoutとPort／契約変更で失効先を変える | 座標変更で全Build、意味変更は無視 |
| NW-08 | 実装選択A→Bの旧実行資格を拒否 | World FPが同じなので旧Commandを許可 |
| NW-09 | 汎用演算と生成Goの比較でRead／Failureを保持 | 値が同じなら全面的な同値とする |
| NW-10 | Graph→Tree／折畳みのLossと編集対象を区別 | 描画線から業務Relationを書換え |

NW-01〜10は既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)へ渡す**未実行の反証候補**である。[Plugin接続の証拠](PLUGIN_CONNECTION_EVIDENCE_AND_COMPOSITION_SLICE_2026-11.md)、[宣言とNode表現](DECLARATIVE_COMPOSITION_AND_NODE_SURFACE_CONTRACT_2026-11.md)、[成果物要求と計算合成](DECLARATIVE_RESULT_PROCESS_COMPOSITION_2026-11.md)の既存契約を用い、新しいValidator／Expected正本を増やさない。

正式なCore Port型、万能Adapter探索、自動配線編集の採用判断は、少なくとも二つの異なる実Consumerで再利用と誤接続検出を確認してから行う。CLI実Provider・Studio実Windows Stage1→Stage2の既存P0を、この追加検討の先行条件として止めない。
---

## 11. Portの全体接続と項目接続を切り替える：型付き配線の具体化（2026-11-15）

[プラグイン入出力Mapping設計§19](PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md)では、「複合型全体を一度に渡す」「必要な基本型・Fieldだけつなぐ」「型変換用Adapterを間に置く」を、**受渡し粒度と変換の有無の組合せ**として整理した。

Node Viewの初期表示は、Record／Sequence／Graphなどの**型付きの値全体**を一つのPortとして扱ってよい。細部を編集する場合だけField／Relation Pathを展開し、入力側の必須項目と利用できる出力Fieldを照合する。全体型が合う場合は必ずしも個別Field配線を求めず、個別Fieldを組み合わせられる場合は必ずしも新しい複合Typeを定義させない。途中にAdapterを入れても、元のProducer／ConsumerやSourceの意味を新たに複写しない。

| UIでの操作 | 型付き構成における意味 | Preview／診断 |
| --- | --- | --- |
| 全体Port同士をつなぐ | 同一／受理済み互換のContract全体の受渡し候補 | 必須保証、非開示Field、Scope、Collectionの完全性を確認 |
| 出力Portを展開してFieldをつなぐ | 一つのSource Field／Relation Pathから入力Itemへの明示Mapping | 同名推測をしない。未接続の必須項目、重複入力、欠損と多重度を表示 |
| 複数Fieldを一つの入力Recordにまとめる | 構築するRecordの項目集合とMapper／Operatorの入力閉包 | 不足Field、Parent＋Child二重指定、競合時の優先順不明を拒否 |
| Adapterを挿入する | 明示的な型付き変換（純粋Operatorまたは別Provider） | Source→Adapter→Sinkの型・Law・Loss・追加Read／Effect・実選択を個別に検査 |
| Portを折畳む・展開する | 同一Mappingの表示粒度のみ変更 | Mapping ID、生成物、実行回数は変化しない |

**型一致だけで接続済みとはしない**。例えば`Decimal`を円／米ドル双方に流用するには単位・意味Lawが足りず、`OrderSummary`全体が型適合でも非公開の`internalNote`をHTTP応答へ転送してよいとは限らない。配列の先頭だけを選ぶこと、Graph辺の多重度を潰すこと、丸め・通貨変換は明示Operator／Adapterの意味を要する。

最初のUIはread-onlyの**全体Port⇄Field表示**から始め、既存の同一Contract直結と一つの項目Mapping、純粋Adapter一つの3例をCLIの検収票に対応付ける。接続をドラッグしてもProposalを作るだけとし、Source Apply・Builder再生成・Runtime Provider選択・Grant・Effectは既存Ownerで独立に確認する。Producerの一値を複数Consumerへ配ることとProducerを複数回呼ぶことも区別する。

実装順は§10のNW-Q0／Q1を流用し、PIO-01〜10を#1223／#951／#2497／#1180／#2514へ渡す。新しいPort Core型、万能型変換検索、第二Graph Runtimeは不要と判断できる限り導入しない。Studio Stage1→Stage2実WindowsとCLI実Providerの独立P0を変更しない。**本節は設計・実装引渡しであり、新しいNode View／Adapter実装の受入ではない**。
---

<!-- data-form-path-conformance-20261116 -->
## 12. 型付き接続を経路として検証する：データ様式の保存条件（2026-11-16）

### 12.1 単一Portの適合から、経路全体の適合へ

§11で定めた「型全体を渡す／Fieldを結ぶ／明示Adapterを挟む」は、同じ標準Dataを異なる粒度で受け渡すための三つの操作である。**一辺ごとに型が合っても、複数の辺を通った結果が正しいとは限らない。** 例えば、Record全体の型が一致しても途中で未開示項目をHTTP向けのDataに加えることはできない。GraphのNodeをTableへ投影するとき、同じ端点を持つ複数のEdgeを一行へ畳めば、元のRelation occurrenceは復元できなくなる。

ここで共通化するのは新しい「Port値」や実行Engineではなく、**既存のMapping／Projection契約を各辺に適用し、その主張が最終Consumerまで成り立つかを問う検収方法**である。PortはData／Function Data Contextに束縛された入出力の接点、NodeはOperation／Process／Pluginの配置上の出現、Edgeは宣言された接続とする。表示上のEdge IDから意味上のRelation IDや実呼出し済みの証拠を作らない。

| 問い | 各辺で確認すること | 経路で改めて確認すること |
| --- | --- | --- |
| 型・意味 | 標準型と構造Facet、OperatorのShape／Law／Binding | 最終Consumerの要求を全段で充足し、途中の変換・変名が意味を変えていないか |
| 識別・粒度 | Subject／Occurrence／Version、Field／Relation Path、入出力grain | 集約・結合・fanoutの後に元個体を再特定できると誤認していないか |
| 順序・多重度 | ordered／bag／set、Relationの向きとEdge ID | 重複除去・並べ替え・投影で失われたものを後段が必要としていないか |
| 値の状態 | present／explicit NULL／missing／unfetched／withheld／empty | 途中で「ゼロ」「空」「存在しない」へ無言で置き換えていないか |
| 取得範囲 | Context、Scope、Snapshot、coverageと否定依存 | 部分取得から「全件」「該当なし」「唯一」を結論していないか |
| 時間・数量 | valid／knownなどの時点、domain／observation、単位・尺度・丸め | 時点・単位の異なる変換を、名称・数値一致だけでつなげていないか |
| 出典・資格 | Source版、Mapping／Operator／Provider選択、根拠のbyte位置 | 現在のSourceから実Builder／実Call／Consumerへ到達した証拠があるか |
| 権限・公開 | 機密項目・Audience・Disclosure、現在Grant、Effect境界 | 中間段の権限や「読める」という事実を最終公開・更新資格へ流用していないか |

判定には既存#2497／#4350の語彙 **preserved／lossy／rejected／unqualified／unknown** を使い、新しい適合Enumを増やさない。型の候補一致は接続の提案には使えるが、Meaning受理・実現選択・現在Grantを含む「実行可能」の宣言には使えない。

### 12.2 経路合成の原則

A→B→Cを検討する際、次の順序を守る。

1. **要求から逆に調べる。** Cが必要とするField・粒度・時点・Read完全性・Lawを列挙し、Bが保証できる範囲を確認する。Bの出力型がCの入力型と同名であるだけでは十分でない。
2. **各Mappingの保存・損失を接続する。** A→Bで失ったOccurrenceやFieldは、B→Cで型が一致しても自動復元しない。変換で新しい事実を取得した場合は追加Read・Source・時点・権限が別途必要である。
3. **非単射の逆写像を禁止する。** 集計一行、Treeへ折り畳んだ多重Graph、先頭要素抽出、欠落Fieldを含むRecordから、元の一意な更新対象を推定しない。明示Actionは逆写像の代用品ではなく、別に対象確定・Expected State・Grantを要する。
4. **出典と実行を別に証明する。** Source Indexの現在性、Readerで解決した型、Operator／Plugin Lock、生成GoとBuilderの実消費、Provider実Call、Consumer受領、Effect receiptは別の観測である。結果の数値が同じことだけで間を埋めない。
5. **表示操作は経路を変更しない。** Node ViewのPort折畳み／展開・配線のハイライトはProjectionの変更であり、Mapping ID、Read回数、実行回数、モデル版を変化させない。配線の編集はまずProposalとし、Source変更・再検証・再実現を別に行う。

特に、**Adapterを挿入するだけでは権限は増えない**。通貨・温度の点と差・CRS・日時の変換には、必要な変換Law、時点、精度、Loss、選択Providerを明示する。AdapterがIOやEffectを伴うなら純粋変換と同一の扱いにしない。型適合、業務的な意味適合、実装の選択、呼出し資格は互いに独立である。

### 12.3 最初の検証対象と独立Expected

最初の検証は**read-onlyの経路検収票**で行い、既存のPIO-01〜10／XMAP-01〜10／NW-Q0〜Q1のExpectedを参照する。下記はそれらを経路として組み合わせる追加の判別例であり、第二のExpected正本ではない。

| ID | 固定する差分 | 検収で拒否または区別するもの |
| --- | --- | --- |
| DPC-01 | 型付きRecord全体の接続と、同じRecordの必要Fieldだけの接続 | 必須Field不足、任意Fieldの未取得を無条件に同値扱いすること |
| DPC-02 | Decimal項目の純粋な正値検査と、単位・尺度の異なる同じDecimal | 基本型の一致だけで正値Lawや単位変換を認めること |
| DPC-03 | Graphの同一端点を持つ多重EdgeからTable／Treeへ投影 | Edge occurrenceの消失を隠し、表示行から更新対象を逆算すること |
| DPC-04 | 七行の部分取得を集計し、後段で該当なし／唯一を判定 | Read閉包を欠いた否定・一意性の断定 |
| DPC-05 | 明示NULL・missing・unfetched・withheldをField接続で通す | 0、空文字、false、非存在へ黙って正規化すること |
| DPC-06 | Adapterを含む三辺接続でSourceだけ改版、または実OperatorをA→B→Aへ変更 | 出力byte一致を旧Source Index・旧実現資格の復活へ読み替えること |
| DPC-07 | 内部Recordの項目をread-only CLIと公開HTTPへ投影 | 型一致・CLI可読性から非開示Fieldの公開Grantを推定すること |
| DPC-08 | Node Viewを全体PortからField表示へ切り替える | 表示だけでSource変更・Mapping変更・追加実Callが起きること |

**検証順**：まず既存Source／Data／Rule／Mapping・Node投影の情報だけでDPC-01／02／05／08のread-only票を作り、保存・拒否・未検証を表示する。次にDPC-03／04／06／07を、既存のGraph／Query／Plugin／APIのOwnerへ反例として渡す。製品Parser、Core Port型、Registry、実行Runnerを追加する前に、既存の表現できない条件を具体的に記録する。

実装上の入口は#2495（Facet）、#2497／#2500（Mapping・逆写像）、#2506／#2586（Data Context・Read）、#951／#1251（Plugin選択）、#1121／#4544（実現）、#1180／#2514（独立Expected）、#1124（Node View）とする。DPCの表は受入候補であり、新しいGoテスト・Provider実Call・Owner意味受理を実施したことを意味しない。

**優先順位は変えない。** 製品P0はSource→生成Go→実Builderの閉包、通常CLIから選択Providerを呼ぶ実Read、非Plugin Studio Stage1→Stage2の実Windows受領である。経路検収票はそれらを説明しやすくするために追加するのであって、新たな先行Gateではない。

### 12.4 Producerの保証とConsumerの要求を経路全体で照合する

一つのPortが提供する型と、利用側が必要とする情報は同じではない。**Producerの保証、Consumerの要求、Adapterによる変換の前提・結果・損失**を個別に取り出し、A→B→Cの経路で必要条件が充足するかを確認する。各辺で型が一致しても、途中で失った情報は後段の型一致によって復元しない。

例えば内部Operationの`OrderSummary`をHTTPで公開するとき、Record全体の型が一致していても`internalNote`の公開が認められるとは限らない。公開に必要なFieldだけを明示的に射影し、Audience・Disclosure・Scopeと現在Grantを別途検証する。多重GraphのEdgeをTreeへ畳んで表示する場合も、表示が成功したことと元Edge occurrenceを一意に更新できることは異なる。部分取得した7行から求めた集計値が正しくても、全件の該当なし・唯一性を主張するには取得閉包の証拠を要する。

| 段階 | 判定対象 | 不適格・未立証の扱い |
| --- | --- | --- |
| 構成可能性 | Field・型・構造Facet・Multiplicity・Shape／Law／Bindingが利用側の要求を満たすか | 不整合は`rejected`、必要な宣言や根拠が足りなければ`unqualified`／`unknown` |
| 情報の保存 | Identity・Occurrence・Presence・Coverage・grain・時点が最終Consumerの要求範囲で残るか | 明示的な損失のあるread-only Projectionは`lossy`。失った情報を使う否定・逆編集は拒否 |
| 実現・利用 | 現在のSource・実装選択・Builder入力・Provider実Call・実Consumer受領がそれぞれ確かめられたか | 表示結果や型の一致から、実呼出し・公開・Effectの資格を作らない |

`preserved`は**当該Consumerが要求する観測範囲での保存**であり、元Dataの全制約が無損失という意味ではない。Adapterが変換のため追加ReadやEffectを必要とする場合は、純粋な型変換と同等には扱わず、依存・選択・失敗・Grantを別票にする。

### 12.5 最小のread-only接続検収票と反証

最初のコード差分では新しいFlow EngineやCore Port型を作らず、既存の`Workspace.Load`で選択したData表・Attribute・Ruleの**一つのSource版**から、読み取り専用の検収票を得る。票に記録するのは、対象ID、原Source byte位置、Producer保証、Consumer要求、Mapping／Adapter、保存・損失、根拠の不足、判定と理由である。Readerが返した項目定義を、実Occurrence・実値・全件取得の証拠として使わない。

最初の独立ExpectedはDPC-01／02／05を対象とし、全体RecordとFieldの接続、Decimal型と意味Lawの違い、`required`と明示NULL／missing／unfetched／withheldの違いを確認する。次にDPC-08でNode Viewを展開・折畳みしてもMapping Identity・生成物・Read回数を変更しないことを確認する。Graph多重辺・取得閉包・HTTP公開を扱うDPC-03／04／07は各Ownerの独立試験へ接続し、DPC-06はSource改版と実装選択改版を**別の失効条件**として扱う。

誤成功を検出するMutantとして、`missing→NULL`、`partial→complete`、多重Edgeの合併、非公開FieldのHTTP転送、古い実装選択Pinの受理を固定する。判定結果だけでなく**どの保証が不足したか**を独立Expectedへ記録し、実装から期待値を生成しない。並行PR #5097（自己記述）と#5114（Scenarioの選択Pin）は、そのexact HEADが受理されるまではmainの実績へ算入しない。

### 12.6 部分取得でも判断できる場合を、規則ごとに見極める（2026-11-16）

データ様式の取得状態と、規則について**結論を出せるかどうか**は別である。「部分取得なら必ず判定不能」とすると、すでに見えている違反まで見落とす。一方、見えている行だけに違反がないことから、全体でも違反がないとは結論できない。

現行mainの `internal/semantic/validationeval/reading.go` には、読み手が状態を伝える `StatefulSource` と `Decide` がある。数値項目は `present`／`null`／`not-fetched` を区別し、Relationは `complete`／`partial` を区別する。戻り値は、規則が成立した `satisfied`、不成立だった `not-satisfied`、理由を伴う `undecided` に分かれる。読取状態の矛盾や未対応の入力は、判定不能という業務上の結果ではなくエラーとして扱う。

この整理の中心は、**未取得部分にどのような値があっても結論が変わらないときだけ確定する**ことである。取得範囲が完全かどうかは重要な入力だが、全ての規則に一律の停止条件を設けない。

| 読み取れたこと | 規則 | この範囲で言えること | 言えないこと |
| --- | --- | --- | --- |
| `quantity=2.5` が存在する | 正値 | `satisfied` | 別項目やレコード全体の完全性 |
| `quantity=0` が存在する | 正値 | `not-satisfied` | 未取得の他項目の状態 |
| 数値項目に明示的なNULLがある | 正値 | `undecided(value-null)` | NULLを0と見なした不成立 |
| 数値項目をまだ読んでいない | 正値 | `undecided(value-not-fetched)` | 値が存在しないという判断 |
| 同じキーが2行で実際に観測された。残りは部分取得 | Relationの一意性 | `not-satisfied`。未取得行は既存の重複を取り消せない | ほかの重複数や取得全件数 |
| Relationを全件取得し、行が1件だけ。キーが未取得またはNULL | Relationの一意性 | `satisfied`。一行だけでは重複しえない | そのキーの具体的な値 |
| 重複は見つかっていないが、Relationは部分取得 | Relationの一意性 | 通常は`undecided` | 全体が一意であるという断定 |

この表で説明しているのは既存の正値・一意性判断の範囲である。すべてのRule種別へ自動的に一般化しない。日時の前後・期間重複を扱う規則についても、**どの時点の値、どのRelation行、どのグループを比較するか**をそのRuleの条件で検査する。現在mainには暦日の前後と期間非重複の実装があるが、それだけで二時点履歴の全整合性やProviderのSnapshot閉包が成立したことにはしない。

現行の `ValueState` が直接表すのは上記3状態であり、従来のDPC-05に列挙した `withheld`（非開示）や「対象レコード自体が存在しない」はこの3状態へ無理に押し込まない。非開示の可否はFunction Data Context・Audience／Disclosureと現在の権限判定に保持し、未取得と同じ業務事実だとは扱わない。

### 12.7 定義表・記録例・実Providerデータの三つの役割を区別する

mainに導入された `tei.authoring/v1` の `example of <data>` は、項目定義表とは別に、**記録の例と、規則に期待する結果**を文書に置くための明示的な印である。選択済みReaderは例を `RuleExamples` に保持し、受理済みの規則に対して `tei validate` が期待と実際の判断を照合できる。印のない説明表を自動的な実データとしない。例を追加・編集しても `World` の意味定義は変わらない。

ここで区別するものは次の三つである。

| 役割 | 情報源 | 保存・照合するもの | 自動的に付与しない資格 |
| --- | --- | --- | --- |
| **定義** | `data`／`rule` の選択済みSource | Data・Attribute・Ruleの型付き意味、Source位置・版 | 実レコードの存在と値 |
| **記録の例** | 印付き `example of` と期待結果 | 例のセル、NULL相当の空欄、未記載列、期待結果のbyte位置、規則との照合 | Providerの実Read、業務Ownerの独立受理、全件性 |
| **実際の記録** | DB／API／Fileの実Provider | Subject・Occurrence・ValueState・Read範囲・時点・実Called | サンプル例からの実在推定、現在GrantやEffect |

例の空セルは、その例の記録に値がないことを表す。例に**列そのものがない**場合は読んでいない項目として扱う。実データの未取得・非開示・不存在まで一律に空欄へ変換しない。例の期待値は記述者が記した確認条件であり、#1180／#2514の独立Expectedを置き換えるものではない。規則自体の型・参照が不正なら例の判定を進めず、Operatorの実装が選択されていないときも「例が一致した」と報告しない。

意味定義が不変のまま例だけを直した場合、`World`のFingerprintが同じでも、**例のSource byte・期待結果・その検収証拠は改版**される。Source Indexの現在性、Model／Artifact再利用、検証結果の再利用は分けて判断する。例の期待結果が変わっただけで業務プログラムの再生成を強制せず、かといって古い検収票を現在の結果として使わない。

### 12.8 既存の独立Expectedに渡す最小反例

新しい試験正本や汎用Validatorは作らず、既存の `validationeval/reading_test.go`、`workspace/tei_md_rule_example_test.go`、`operations/rule_example_check_test.go` と対応するSourceを再利用する。最初の追加差分は、選択した一Sourceに対するread-onlyな「**定義／取得状態／規則要求／判断理由／元Source位置**」の照合票とする。

| ID | 検証対象 | 独立Expectedで区別すること |
| --- | --- | --- |
| DPC-09 | 同じ正値Ruleへのpresent／NULL／not-fetched | 成立・不成立・判定不能の違いと、それぞれの理由 |
| DPC-10 | Relation一意性の部分取得と、観測済み重複 | 部分取得でも確定する不成立と、未取得部分で結論が変わる判定不能 |
| DPC-11 | 全件取得した一行だけのRelationでキーが未取得 | キーの値を知らなくても一意性は成立するという確定条件 |
| DPC-12 | 定義表・印付き例・印のない説明表、例の期待値改版 | Worldの不変性と、検収Source／期待結果の失効を別に扱う |
| DPC-13 | 暦日の順序と期間非重複、部分Read、開いた終了日 | Ruleに明示された期間・グループ・終端の意味だけで判定し、二時点・実Providerの完全性を推定しない |

DPC-09〜13は**新しい実装の合格宣言ではなく、既存の実装から次に検証すべき接続条件**である。生成Goと既存Evaluatorの結果・Read先ID一致（main #5118）や、Build Plan後の元Markdown改版拒否（main #5129）は、すでにある狭い回帰として参照し、重複実装しない。結果が合っても読み取った対象が異なる場合、同じConsumer結果として認定しない。

最初の製品への接続は `Workspace.Load`→Data／Rule／ExampleとSource位置→`Decide`の判断・理由→既存CLI／Studioのread-only Inspector とする。Provider実Read、生成Go・Builderの同一Attempt、独立Consumer、Windows Stage1→Stage2は各Ownerの別検収であり、この追加設計の先行Gateにはしない。

### 12.9 第一縦断：選択した定義と実際に問われた項目を同一Loadで照合する

DPC-09の最初の実装候補を、既存の`internal/operations`の読み取り専用経路で具体化する。新しいInspection APIは導入せず、`Workspace.Load`、`BuildSourceIndexFromLoad`、`InspectionFromLoad`、`validationeval.Decide`を一つの試験で連結する。

一度だけ選択した`TransferRequest`のMarkdownから、`allocation`属性（`std.decimal`、`required`）と`allocation_positive`規則を同じWorld／Source Indexで照合する。規則が読むべきAttribute IDと、状態を明示したSourceが実際に受けたRead IDを比較し、原文のAttribute行が示すbyte位置を確認する。判定は次の四例を独立Expectedとして固定する。

| 入力状態 | 確認する結末 | 誤合格を拒否する観点 |
| --- | --- | --- |
| 数値2.5を取得 | `satisfied`、正しいRead ID | 別Fieldを読んでも結果が同じなら合格、としない |
| 数値0を取得 | `not-satisfied`、正しいRead ID | 真偽を単なる成功／失敗APIへ畳まない |
| 明示NULL | `undecided(value-null)`、Resultなし | Required定義だけからNULLを0や違反に変換しない |
| 値を未取得 | `undecided(value-not-fetched)`、Resultなし | 未取得をNULL・不存在・非開示と同一扱いしない |

この第一縦断は、**記述元の型・判断の対象・理由・出典位置が連結していること**に限る。評価用Sourceはテストが明示的に渡す値であり、正式Providerの実Read、CLI／Studio独立Consumer、生成Go／実Builder、Audience／Grant、Collection・Graphの多重性やRelationの取得閉包を検証しない。既存のP/G実Binary同値回帰（#5118）やPlugin純粋Transformの別PR（#5156）を、同じAttemptの成果に合算しない。

次に必要なのは実データを返すProviderについて同じRead ID・Presence・Scope・Snapshot・Coverageを観測することである。**Read IDだけが一致しても、値がどのOccurrenceや時点から来たかは証明されない。** 同じ図表を使うNode View／Studioの可視化は、この出典と取得条件に対するProjectionとして扱う。

### 12.10 出典の位置を表示することと、原文を再確認することを分離する

現行mainのDesktop設計（#5178）では、`Source Index`の参照先を一覧表示できても、それだけで原文ファイルを現在の状態で開けるとは主張しない。Data様式の最初のInspector／Rule縦断もこの区別を引き継ぐ。

| 資格 | 読み取り専用で提示できること | 次の検収に必要なこと |
| --- | --- | --- |
| 宣言の特定 | 同じLoadのData／Attribute／Rule ID、Source Indexとbyte位置 | 対象Sourceの現在性。意味受理や編集権限は付与しない |
| 位置の表示（SL-1） | 相対Path・範囲・省略数を含む参照一覧 | 一覧が全参照を含むとは推定せず、原文の取得済みとも言わない |
| 原文の再確認（SL-2） | HostがRoot内のSourceを再読取した結果 | byte・選択版・対象Occurrenceの再照合。古い位置を現在の原文へ適用しない |
| 編集・実利用（SL-3） | 本文や対象に対する操作の提案 | 編集対象の一意性、現在Grant、実Consumer受領。表示からWrite資格を作らない |

DPC-14（Data／Rule／Source／判断の一Load照合）とDPC-15（表示参照からの実Source再確認）を別の独立Expectedへ分ける。最初のGo回帰が試すのはDPC-14のみであり、Wails／WindowsでのSL-2／SL-3を一緒に合格させない。

また、Plugin側で並行するWhole Record／Field純粋Projection（#5156）は、出力形状の検証である。ここで得られるAttributeの出典やRuleの判断理由を、Transform Valueの出力だけから生成しない。公開対象や取得範囲の適格性は引き続き型付きMapping・Function Data Context・Ownerの判断に残す。

### 12.11 値の状態を正しく渡すだけでは、取得範囲の正しさを保証できない（2026-11-17）

mainの`internal/driver/validationread`は、Contract Queryの列値を`present`／`null`／`not-fetched`へ写し、Relationの`complete`／`partial`を受け取って既存`validationeval.Decide`へ渡せる。SQLite実ファイルを使う`sqlite_path_test.go`では、実SQL Executor→列Mapping→`FromHeaderDetail`→Rule判定の限定経路が確認されている。これは、これまでの手渡し`StatefulSource`より一段具体的な証拠である。

ただし、**`CoverageComplete`は結果行から導かれる事実ではなく、現状では呼出し側の申告である。** 同じ試験では、実際には4行あるRelationから`LIMIT 2`で2行だけ取得した際、正しく`partial`と申告すれば座席番号の一意性を判定保留にするが、誤って`complete`と申告すると成立と判定されることが示されている。Rule評価器が与えられた条件に従って正しく計算しても、前段の申告が間違っていれば業務上の結論は間違う。

そこで次の検収を、**取得範囲の主張を作る段階**と、**その主張の下でRuleが判断する段階**に分ける。Queryの結果件数、`LIMIT`の有無、ページ終端、Filter、RelationとHeaderの対応、Scope、Snapshotを確認できない場合、単に「2行返った」ことや呼出し側の希望から`complete`を認定しない。

| 受理対象 | 何を証明するか | 証明できない場合 |
| --- | --- | --- |
| 取得要求 | Function Data Contextに束縛された対象Relation・Header・Field・時点・Scope | 結果の型や行数だけから対象を確定しない |
| 実行されたRead | 選択Provider／Query／Parameters、実Call、返却行と列、使用された実装版 | Query定義があるだけで実Read済みとしない |
| 取得閉包 | 必要なRelation範囲について全ページを受領したか、Filter／Limit／欠落列があるか、どの時点の観測か | `partial`または未立証として扱い、`complete`を自動付与しない |
| Rule判断 | 定義されたAttribute／Relation IDを実際に読み、上記の状態・範囲で結論が不変か | `undecided`と`not-satisfied`、入力矛盾によるerrorを分ける |
| 最終Consumer | 判断の結果・理由・参照した範囲・出典と版を受領したか | Inspector表示だけで外部配信・Owner受理を認定しない |

現行`Detail`を未証明の`complete`が受け取れない仕様へ変更した、という意味ではない。**完全性を保証する責任は、実Query／Providerの実行計画と結果を束縛する呼出し側に残る**。この境界を通常製品CLIで通すのが次の実装課題である。独立したCoverage Engineや第二のData Contextは作らない。

### 12.12 同じ値の見え方と、同じ情報の保存を分ける

型付きPlugin PortのWhole Record／Field／明示Adapterと、Studioのread-only Presenterも、上の取得条件を変えてはならない。main統合済み#5156の純粋Transform試験では、全体Recordなら余分な`internalNote`を保持する一方、必要な二Fieldへの射影では追加項目を出力しない。出力形状の正しさは確認できるが、公開Grantや業務上のField IDの正しさは別である。

審査中#5214では、`select-field(optional=true)`が欠損Fieldと明示NULLを同じ値へ畳み得ることを負例として扱っている。**同じJSONになったことは、同じ取得状態であった証拠にならない。** その結果を再びRuleへ入力する際、元の`null`／`not-fetched`を暗黙に復元できると扱わず、元のReader／Contextで状態を保持するか、情報を失った`lossy`な投影として用途を限定する。

DesktopのSource位置Presenter（main #5212）も同様に、場所・Fingerprint・省略数などを表示する純粋Projectionである。表示できたという事実から、元ファイルを実際に再読取した、全参照がそろった、編集できる、という資格を作らない。型付きMapping、Rule判断、実Provider、表示、公開は**同じSourceを参照できても異なる受理段階**である。

### 12.13 次の独立Expectedは「申告の偽り」と「投影での損失」を検出する

DPC-01〜15を置き換えず、次の一因子反証を追加する。いずれも既存`validationread/sqlite_path_test.go`、`validationeval`、Plugin Transformの小回帰、Source Index／Presenterを足場にし、期待値を実装から生成しない。

| ID | 固定する反証 | 受入で区別するもの |
| --- | --- | --- |
| DPC-16 | 実Relationは4行、SQLの`LIMIT 2`が返す2行を誤って`complete`と申告 | 結果2行と完全性の証明は別。通常製品Callerは証拠のない全件申告を拒否／保留 |
| DPC-17 | 同じ行の`NULL`列と、狭いSELECTで取得していない列 | `value-null`と`value-not-fetched`の理由とSourceの違い |
| DPC-18 | `std.decimal`にSQL REAL（`float64`）を渡す | 値がたまたま0.5でも、精度の根拠を欠く変換を拒否 |
| DPC-19 | Whole Record・明示Field射影・任意Fieldを通過 | 非選択Fieldの除外、欠損とNULLの損失、公開Grantと出力形状の分離 |
| DPC-20 | 部分取得の重複／期間重複と、違反未観測の一覧 | 見えた違反は確定できるが、見えない違反を全件不存在と断定しない |

**最初に実装するのはDPC-16**。既存SQL Executorを通った結果から、Callerが`CoverageComplete`を申告できる条件を既存Query／Function Data Contextの選択情報と突き合わせる。現状のSQLite試験は、誤申告が起き得ることを示す基線であって、その拒否が製品で成立した証拠ではない。正式な実PostgreSQL→通常CLI→独立Consumerでの受理は#2586／#4496の別P0に維持する。

### 12.14 全件性は「問われた範囲」についてのみ証明する：DPC-16実装契約（2026-11-17）

データ様式が保存すべきものを、**入力の構造**、**実際に観測した内容**、**判断に必要な閉包**に分ける。「SQLの返却行を読み終えた」だけでは、Functionが問うRelation・Scope・時点の全件性は保証されない。逆に、データベース全体を走査しなくても、定義した対象範囲を読み切れれば、その範囲での判断は成立し得る。

最小の読取専用検収票では、既存のQuery／Function Data Context／Evidenceを参照し、次の欄を**別々の事実**として扱う。これは既存の受理票に加える照合項目であり、新しいCore型や第二のデータ正本を設けない。

| 欄 | 記録・照合する内容 | 誤って推論してはならないこと |
| --- | --- | --- |
| Requested | Function／Ruleが要求したRelation、Header・業務Identity、Scope、時点、Field、粒度 | SQLの返却列だけで要求範囲を決める |
| Executed | 選択Query Contract、実RealizationとSource Revision、Provider、Binding、実Callの識別 | 宣言があるだけで実行済みとする |
| Observed | 実行が返したRow／Attribute、明示NULL、未選択列、ページ進行、Read終了状態 | 返却件数や値の一致から全件とみなす |
| Claimed | Callerが「全件」と申告した範囲と根拠 | CoverageCompleteを証拠そのものとみなす |
| Qualified | Requestedに対して実Readが証明した範囲、未立証の理由、現在の利用資格 | 「未立証」と「欠落が実際に確認済み」を混同する |

**ReaderとCallerの境界**：現行のvalidationread.FromHeaderDetailは、Callerが渡したCoverageをRule評価へ引き継ぐReaderであって、完全性の証明者ではない。製品Callerはその前でClaimを照合し、根拠のないcompleteを渡さない。既存validationeval.Coverageはcomplete／partialの二値なので、証拠不足は検収票ではunknownとして保持し、Ruleへの入力では安全側のpartialにする。そうすると、観測済みの重複違反は確定し得る一方、未観測の重複が存在しないとは断定できない。未立証を、物理的な欠落が確定した証拠として表示しない。

#### 初回の限定合格と拒否条件

既存のinternal/driver/validationread/sqlite_path_test.go（D1の4行、members／members_first_page、seatUnique／personUnique）を再利用する。Provider、Relation、Scope、Ruleを固定し、次の一因子比較から始める。

| ID | 入力差分 | Callerが許してよい結果 |
| --- | --- | --- |
| DPC-16a | first_pageの2/4行にcompleteを申告 | 根拠不足としてcompleteを拒否／保留し、Ruleへ偽の全件を渡さない |
| DPC-16b | 同じ2/4行を未立証として扱う | seatUniqueは判定保留、観測済みpersonUnique違反は不成立と確定 |
| DPC-16c | 異なるScope・Query Bindingを全件の根拠として提示 | 返却行が一致しても、別Scopeの資格を流用しない |
| DPC-16d | 0行の応答だけを全件の根拠とする | 空集合・不存在・一意性成立の認定を保留 |

最初は**一つのContract Queryについて、実行したQuery／Binding／ScopeとCallerの申告を同じAttemptで突き合わせる**。SQLにLIMITが見えないこと、ReadがEOFまで進んだこと、件数が一致したことは一般的な完全性の証明にはならない。WHEREの対象縮小、JOIN・DISTINCT・集計の粒度変化、ページング未完了、時点や可視範囲の不一致は別の反証とし、静的に認定できないQueryは未立証とする。複数ページを結合するときは、継続位置・一意な順序・読み取りSnapshotの一貫性も確認する。

**合格の上限**：初回DPC-16a〜dが実装されても、最初に認定できるのは限定Queryに対するCaller境界の拒否能力のみ。全SQL静的解析、全Provider、公開Grant、実Builder、Studio、Owner業務意味受理は含まない。実PostgreSQL→通常CLI→独立Consumerは既存#2586／#4496、独立Expectedは#1180／#2514に引き渡す。この文書の追加自体は新規Go／DB／Studio試験を実施した証拠ではない。

### 12.15 Operatorの署名・図面Projection・Sourceを一つのデータ取得へ結び付ける（2026-11-17横断統合）

[規則の正規形とOperator署名](RULE_NORMAL_FORM_AND_OPERATOR_SIGNATURE_DECISION_2026-11.md)の採用判断、[三要素Method](THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)と[自己記述の第一縦断](TEI_SELF_DESCRIPTION_REVIEW_SLICE_2026-11.md)の設計、および既存の型付きPort設計§11を照合する。ここでは新しい意味モデルを増やさず、**一つのDataと実Readに、必要なRule入力・図面・Consumerを対応付ける条件**を整理する。

#### 12.15.1 Ruleが要求する範囲と、Queryが供給を宣言する範囲を突き合わせる

採用済みの規則設計は、規則を「宣言済みOperator一つと名前付き入力の割当て」に正規化し、Operator署名に型・入力単位・値か有無か・一部観測で反証できるかを記述する方針である。**これは設計判断であり、現行製品の評価器全体が正規形へ移行済みという意味ではない。** 現行の六種のValidationは移行中も互換性を保つ。

- Operator署名が「Relationの全行」を要求する場合、まずRuleのRelation ID・親業務Identity・Scopeと、選択されたContract Query束縛の「対象Relationをすべて返す」という宣言を照合する。束縛が未宣言、別Relation、別Scopeなら、全件性を要求するRuleに無条件で供給しない。
- Query束縛の宣言は**供給者の保証の主張**であって、実SQLが正しく全件を返したことの自動証明ではない。実行したRealization、入力Binding、ページ終端、Read Cohort・時点、取得列、必要な独立回帰を確認する。現行製品にこれらをすべて証明する能力がなければ、未立証のまま扱う。
- 実行時のCallerができるのは、束縛が述べた範囲をより狭く扱うことまでとし、**partial／未立証からcompleteへの格上げを認めない**。既存二値Coverageへ渡すときは未立証を保守的にpartialへ写す。ただし、検収票上の「未立証」と「実際に一部しか読んでいない」は分けて残す。
- 見えた重複で不成立と判断できるかは、Operator署名の「一部の行で反証できる」条件に依存する。観測できなかった行の値を推測して、成立／不存在／総件数を確定しない。

§12.14の「根拠付き全件性」と矛盾させないため、**静的な契約適合**、**実行時に守られた取得範囲**、**独立試験で確認した保証範囲**を別々の資格として記録する。Operator署名とQuery宣言を照合できても、Provider実Readや第三者Consumerの成功に昇格させない。

#### 12.15.2 同じ読取結果を表と図へ渡す際に保つもの

標準データ構造の意味は、表・Graph・Tree・三要素の図面・Node Viewといった表現型に従属しない。Meta／Model／InstanceとProjectionを分け、同じQualified Subjectを複数画面で表示する場合は、少なくとも**意味Identity、個々のRelation occurrence、順序と多重性、値の状態、単位・精度、対象範囲・時点、Source版**のうち当該判断に必要なものを追跡する。

例えば同じ所属Relationを表とGraphに投影するとき、表が一部の列を省略しても、元Relationが完全であったと再認定してはならない。同名の二Node・並列Edgeを一つへまとめるProjectionは、表示用には許せても、元のRelation IDやEdge IDを復元して編集する根拠にはならない。Whole Record／Field／明示Adapterの型が一致しても、粒度・NULLと未取得・Disclosure・現在Grantが異なれば、経路全体の適格性は別検査とする（§11〜12.13）。

図面・Studio・SHO・SUIが共用するのは**選択したSubjectとSource／Projectionの関係**であり、表示を正本として複製することではない。現在の原本は選択済みMarkdown等のSourceとし、read-onlyレビュー→Source位置の再確認→対象Occurrence／Revision・被参照の検証→編集Preview→明示Save→再Read→再Projectionの順に進める。Viewからの直接書戻しや、古いPinに基づく変更は別資格とする。

#### 12.15.3 一つのFixtureで各境界を個別に検収する

最初の実装Packetは既存D1所属の実SQLite4行と一意性Ruleを使い、複雑なGraphや新DSLを先行実装しない。同じSource版とQuery選択を使って、①Relationの要求Scope、②実SQLの4行／2行、③Operatorの必要な全件性と部分観測の反証、④Field投影でのNULL／not-fetched、⑤表と簡単な関係図のread-only Projection、⑥独立Expectedと受領Consumer、を**別の検収欄**としてつなぐ。

組織C1の実PostgreSQL試験で観測された「同じMembershipの投影4件から異なる仮想Person対応（3人／4人）が成立する」反例は、表面の値が一致しても人物数のClaimを決定できないことを示す。これをGraphや型付きPortへも再利用する際、**未観測のPerson Linkを推定で補わない**。実際にSourceへ保存するIdentity／Relationと、図の候補Edgeは区別する。

| 検収する一辺 | 成立条件 | 不成立・保留の例 |
| --- | --- | --- |
| Rule要求→Query束縛 | 同一Relation／Header／Scopeに対する必要範囲の一致 | 別Scopeのcomplete宣言を流用 |
| 契約束縛→実Read | 選択Realization・Binding・実結果の同一Attempt照合 | 2/4行のページを全件と自己申告 |
| 実Read→Rule判断 | Operator入力の値状態・必要閉包・反証条件が一致 | 観測なしを「違反なし」と確定 |
| Data→Projection／Port | Identity・粒度・状態・Source版・開示条件を明示 | 同じ表示値から欠けたPerson Linkを推定 |
| Projection→元Source | Subject／Occurrence／現在Revisionと変更対象が一意 | 同名Nodeや古いPinから直接書戻し |

この照合は既存DPC-16〜20・三要素SDM-01〜12・独立Expected #1180/#2514へ接続し、新しい試験正本は増やさない。**次の実コード差分はDPC-16aのCaller拒否を優先**し、Operator正規形の全移行や図面の双方向編集をその前提にしない。実Provider→通常CLI→独立Consumer #2586/#4496、およびStudio Stage1→Stage2 #2831/#4583は並行の最上位P0を維持する。今回の設計追補自体は新規の実行・Owner受理の証拠ではない。

### 12.16 全件性の根拠を「宣言・実行・観測・検証」に分けて受け渡す（2026-11-17）

§12.14〜12.15の実装を進める際、Callerが単に`CoverageComplete`を引数へ渡せることと、その申告が正しいことを混同しない。現行`validationread.FromHeaderDetail`は取得結果とCoverageを受け取るReaderであり、契約QueryのSQLや実際の母集合を検証する責務は持たない。初回はReaderやRuleを変更せず、**Readerを呼ぶ直前の製品Callerに限って、申告の根拠を点検する**。

| 資格 | 記録するもの | それだけでは認めないこと |
| --- | --- | --- |
| Requested | Rule ID、対象Relation、親Identity、Scope、必要な列・全件性 | 実Readが行われたこと |
| Declared | 選択Contract Query、Realization、Query入力、供給範囲の宣言 | 実SQLが宣言を満たしたこと |
| Executed | 実行したQuery／Realization、Binding、Provider、Attempt | 母集合の完全な観測 |
| Observed | 取得行、取得列、NULL／not-fetched、ページ継続・終了の観測 | 全件性、存在しない行の不存在 |
| Qualified | RequestedとDeclaredの一致、実行根拠、保証済み範囲、未立証理由 | Ownerによる業務意味受理、他Providerへの一般化 |

**判定規則**：同じRelation・Scope・Binding・Attemptで必要な全件性が裏付けられる場合だけ`complete`を渡す。証拠が欠ける、食い違う、またはページ途中である場合は、理由を記録して既存Ruleには`partial`を渡す。これは安全側の縮退であり、未立証を実際の部分取得と同一視するものではない。反証可能な重複が観測済みなら`not-satisfied`は維持できるが、未観測の行がある限り`satisfied`への格上げは認めない。

初回DPC-16aの独立Expectedは「D1に4行を挿入し、`members_first_page`が2行を返す。同じ2行についてCallerが`complete`を要求しても、`seatUnique`を`satisfied`と返さない」とする。続くDPC-16bは`personUnique=not-satisfied`と`seatUnique=undecided: relation-partially-read`を同時に確認する。DPC-16cは異なるScope／Bindingの根拠を流用しないこと、DPC-16dは0行だけで全件性を証明しないことを確認する。**比較対象は同じ実SQLの行結果とし、Coverage申告・資格だけを変える**。

全件保証を静的に判定できない一般SQL、JOIN／DISTINCT／集計による粒度変更、複数ページのSnapshot不整合、認可による可視範囲の縮小は初回の合格対象にしない。限定Queryで成立した証拠を一般SQLや他Providerへ流用せず、必要ならQuery契約側に不足する宣言とその検証方法を別Issueへ切り出す。独立Expectedは#1180／#2514、実PostgreSQL・通常CLI・独立Consumerは#2586／#4496、StudioのSource往復は別検収のまま維持する。

## 13. ノードモデル導入の最小契約：定義・出現・配線・表示を分ける（2026-11-17）

### 13.1 ノードは業務要素の別名ではなく、ある構成内での出現である

同じOperationを別のProcessから使う場合や、一つのProcessで二度使う場合、**Operation定義は一つでもNode出現は複数**になる。逆に、Graph上の一つの箱が複数の定義を束ねて見せる場合、その箱を一つの意味Subjectへ縮約しない。これらを区別できることを、ノードモデル導入の最初の受入条件とする。

| 識別対象 | 役割 | 識別・版の最小要件 | してはならないこと |
| --- | --- | --- | --- |
| Definition／Semantic Subject | Data・Function・Rule・Operation・Pluginなど、参照先の意味定義 | 既存の安定ID、選択済みSource／Meaning版 | 表示名をIDとする。Flowごとに定義をコピーする |
| Node occurrence | Graph／Processのどこに、その定義をどう配置したか | 所属Graph、出現ID、参照先Definition、出現固有の束縛 | 同じDefinitionの二出現を一つに潰す |
| Port occurrence／Binding | どの出現の、どの入出力・Field・構造Pathを使うか | Node出現、契約Port／Role、型付きPath、必要なSnapshot／Scope | Port名やField名だけで同一とみなす |
| Semantic edge occurrence | どの二つのPort／出現を、何の関係で結ぶか | **独立したEdge ID**、向き、Role、多重性、Source／Revision | 同端点・同labelの二辺を重複として消す |
| View glyph／segment | Canvas上の箱、線分、折り畳み、グループ | View内の表示ID、Layout、Projection版、損失情報 | 座標や表示IDをMeaning ID・Writer Targetに転用する |

上表は**既存の標準Graph・Subject・Source Index・契約情報の投影を検収するための項目**である。新しいCoreの\`Node\`型、共通ID採番器、Graph Registry、独自Compilerの採用案ではない。Source側に安定出現ID／Edge IDが存在しない場合は「未確定」として表示専用に留める。描画時の一時IDを恒久IDへ偽装しない。

### 13.2 同じグラフを三つの問いに使う

**構造の問い**では、Node／EdgeのID、向き、多重性、親Graph、入出力Pathと型が保存されているかを検査する。**意味の問い**では、接続先が要求するRole、grain、key、cardinality、time、NULL／not-fetched、complete、順序、Effectを、供給元の保証と照合する。**現在の利用資格の問い**では、Source・選択Plugin／Operator・Snapshot・Grant・実Builder／Provider・Attemptが実際に一致するかを検査する。

「型が合う」「図上でつながる」「Generatorが生成できる」「実Callが成功した」を同じBooleanにしない。Data Edge、Control Edge、Event Edge、Effect Edge、Source／Evidence Link、Plugin bindingは表示部品を共有しても、成立条件とOwnerは共有しない。Edge Roleが不明なら、実行順序・権限・更新操作を導かない。

Whole Recordをつなぐ場合、Field単位につなぐ場合、明示Adapterを挟む場合は、同じ構造Pathと要求を比較できるようにする。Adapterの存在は型の変換候補を示すに過ぎず、情報損失のない変換、未取得の補完、Provider実装の選択、外部への開示許可を証明しない。AggregationやGraph→Treeのような非単射投影はread-onlyを既定とする。

### 13.3 最初のread-only投影が返すべき内容

新しい永続モデルを作る前に、既存CLI／Inspectorから得られるSubject／Source Index／Operation／Plugin情報を使い、**対象を限定した照合結果**をJSONとStudio Node Viewへ渡す。必要な項目は次のとおり。

| 出力項目 | 何を示すか | 不足した場合 |
| --- | --- | --- |
| \`graphRef\`・\`subjectRef\`・\`selection\` | 読み取ったGraph／SubjectとSource／Reader／Selection版 | 別版への合流や現在性を推測しない |
| \`nodes\`（出現ID・定義参照・Source locator） | 表示する個々の出現とその出典 | 同名ノードを併合しない。位置不明なら逆編集不可 |
| \`ports\`（出現・契約・Path・要求／保証） | 接続点と型付き関係、明示された必要条件 | 未指定を\`any\`や\`complete\`に補わない |
| \`edges\`（独立ID・両端・向き・Role） | 多重Edgeを含む実際の関係 | ID重複は診断、ID未確定は編集対象外 |
| \`qualifications\`・\`losses\`・\`diagnostics\` | Source読取、意味候補、構造適合、実利用、開示・編集可否と失われた情報 | 不明をPASSと表示しない |

上記のJSON項目名は**read-only応答の検討用ラベル**であり、公開API schemaとして固定しない。既存の\`InspectSubject\`、Source Index、Data Context、Plugin Lock、RunPlan／Traceのどこで情報が確認できるかを先に採取し、不足項目だけを元Ownerへ返す。一般的な情報が得られない場合でも、部分的なNode Viewは作れる。ただし当該Graph全体を読み終えたと宣言しない。

### 13.4 四つの最小反例と編集の停止線

| ID | 入力・変更 | 独立Expectedとして必要な判別 | 既存Owner |
| --- | --- | --- | --- |
| **NM-01** | 同じlabelのNode二個、同じ両端・同じlabelの有向Edge二本。別のEdgeが同じIDを再利用する負例も用意 | Node出現二個とEdge二本を独立に保持。重複Edge IDは曖昧なSourceとして拒否し、先勝ち・後勝ちで選ばない | #1124／#2497／#2514。REA外部取込 #3394 |
| **NM-02** | 一つのOperation定義を二つのProcess出現から参照し、一方だけField束縛を変える | Definition IDは一つ、出現とPort Pathは二つ。変更影響は当該束縛へ限定し、他方の出現を書き換えない | #2507／#2620／#4198 |
| **NM-03** | Whole Record→二Field投影→Adapter→Consumer。未取得FieldとNULL、実Readが2/4行の差を含む | 値・型の一致と、必要なgrain／complete／Source／開示の一致を別判定。失った情報を推定で補完しない | #2497／#2506／#951／DPC-16a |
| **NM-04** | 意味Graphは不変でLayoutだけ変更。続いてSourceをA→B→A、PluginまたはScopeを変更 | LayoutだけならMeaning／実Artifact再生成を要求しない。A→B→Aで過去のSelection／Review／実行資格を自動復活させない | #1124／#2600／#4198／#4583 |

NM-01は既存のREA候補取込にあるEdge identityの実例を参照できる。ただし外部観測のEdgeと受理済みSemantic Edgeは別物であり、外部取込の成功をGraph編集・Owner受理の証拠にしない。各Expectedは既存#1180／#2514のScenarioに置き、ここに第二のExpected Storeを作らない。

**編集の停止線**は一意な対象出現、元Sourceの現在Revision、意味を失わない局所逆写像、既存Writerの操作対応、現在の権限がそろう地点である。線のドラッグ・Port接続変更・AI提案は、最初は差分候補とSourceへのnavigationに留める。既存AcceptanceでPreview→確認→適用→独立再Readまで検収できた操作だけを編集可能にする。

### 13.5 最初の実装Packetと採否

1. **N0／NM-01**：既存Graph／REA／Sourceの固定Fixtureを使い、Definition／Occurrence／Port／Edge／Viewの識別対応と多重Edge保持をread-onlyで検査。まず実Sourceの読取・同一版の照合を記録する。
2. **N1／NM-02・03**：現行Operator／Data Context／Pluginの要求と保証を二出現・Field単位で照合し、合格・不適合・材料不足を区別する。未知の意味契約を新型で埋めない。
3. **N2／限定実現**：既存Source→選択Operator／Generator→実Builder入力→Artifact→通常CLI／独立Consumerに、同一Meaningの一つのNode経路を接続する。実Provider／Effectの資格は別票とし、新しいRunnerを作らない。
4. **N3／Studio**：同一SubjectをProperty／Table／Graph／Node View／Source間で選択し、Pin／Dirty Buffer／保存済みSource／Accepted Meaningを別の状態として検査。既存Stage1→Stage2実WindowsのP0を維持する。逆編集は前段の一意Slotに限定する。

**採否判断**：NM-01〜04で既存のData／Graph／Role／Operation／Source Indexだけでは保持できない事実が特定されたときに限り、最小の追加契約を担当OwnerとADRへ提案する。新Node DSL、新汎用Graph Runtime、第二Resolver／Host／Evidence Engineを導入しない。ここまでの記述は計画であり、NM試験の実行・正式契約の採用・Owner意味受理を意味しない。
