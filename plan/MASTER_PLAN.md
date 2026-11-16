# TEIの開発全体計画

## 目的

- [標準データ様式の取得状態・記録例・判断可能性](design/SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md#126-部分取得でも判断できる場合を規則ごとに見極める2026-11-16)：mainの`validationeval.Decide`と`tei.authoring/v1`の`example of`を、既存Data／Rule／Source Indexへ接続する。DPC-09〜13で、NULLと未取得、部分取得でも確定する重複、全件一行の一意性、例と定義の改版、暦日・期間のRule条件を区別する。read-only検収票から始め、独立Expectedは#1180/#2514を使う。実Provider・正式Builder・二Consumer・Stage2のP0は維持する。

- [標準データ様式の型付き接続と経路検収](design/SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md#12-型付き接続を経路として検証するデータ様式の保存条件2026-11-16)：データ型・構造Facet・表現型をPortの全体／Field／Adapter接続から再利用し、経路全体のIdentity・多重度・Presence・Coverage・時間・Source・公開条件を既存Mapping／Function Data Contextの契約で検査する。DPC-01〜08を#2495/#2497/#2506/#1180/#2514に引き渡し、[dogfooding§17](design/IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md#17-標準データ様式を実装の接続証拠へ折り返す2026-11-16)に最小Packetを記録する。設計・受入候補であり新Port Core／Runner／独立Expectedを作らず、正式Builder・通常CLI実Provider・Studio Stage1→Stage2のP0を維持する。

- [RDB・SQL・NoSQLの横断適合と製品実証計画](design/DATA_PROVIDER_CROSS_DB_CONFORMANCE_2026-11.md)：既存Data Operations／Query IR／Provider／Workbench／Effect Fenceへ責務を割り付け、PostgreSQL実Product Readを最優先に、SQLite比較・MongoDB階層・DynamoDB Index・Neo4j Graph・更新／Migrationを段階検証する。Native Query原文、NULL／missing／partial、Identity／粒度、実Grant／Receiptを保持し、別Query Engineや分散Txを先行導入しない。新Provider実接続・二DB同一Function・Studio E2Eは未実施。#2622/#3270/#2586/#4496/#2387/#1174/#2831へ接続する。


この文書は、TEI（鼎／テイ）の製品価値と要求を開発上の作業領域へ整理し、優先する作業と並行できる作業を示します。

期限は設けません。優先順位は実装量ではなく、**TEIの中心価値に関する不確実性をどれだけ減らせるか**を基準に見直します。

- [文書手法からアプリケーション抽象へ：意味を保った構成の横断設計](design/APPLICATION_ABSTRACTION_DOCUMENT_METHOD_WEAVING_2026-11.md)：見出し・表・図・引用・判断理由・変更履歴が持つスコープ、役割、写像、再利用、変更という抽象能力を、既存Data／Role・Facet／Function／Process／Projection／Pluginへ割り当てる。構造・意味・表現・実現・現在資格を別々に検査し、DW-01〜10の独立Expected候補を#1180/#2514と各Ownerへ接続する。**現行mainの実Product CLI inspect（#5058）、元Markdownの実Go Binary同梱・実Builder（#5051）、独立Shape基線（#5064）を足場**に、§8〜10では文書の記述Roleと演算のShape／Law／Bindingを分離した。新しい最小検証は同じ正値Ruleを既存汎用関数と生成Goで評価する五入力・Read回数比較と、DW-11〜16の一因子負例である。通常CLI実ProviderとStudio Stage1→Stage2の並行P0は変更しない。新しい文書Engine・Core Schema・Host・正本は先行採用しない（実装前設計）。

- [設計Projection・実ロード版・データ移行の横断受入](design/DERIVED_PROJECTIONS_RUNTIME_AND_MIGRATION_ACCEPTANCE_2026-11.md)：既存Data／Function／BusinessとRule／Graphから、設計書・Go・帳票・API・DB移行への投影を一つの受注機能で照合する。§7〜10ではLookupの取得時点、Draft／Commit、承認経路Snapshot／現在Grant、Builder・Runtime・Consumerの資格を分離。§11〜14ではmainの実Rule CLI Build #5009、Operator選択 #5007、Source/AST #5020、Studio Pin #5025の**異なる試験・Subject**を混同せず、PQ-A（原Rule／Reader／Generatorと実Buildの束縛）・PQ-B（実装A→B後の旧Scenario失効）に具体化する。既存#1180/#2514へ引き渡し、通常CLI実ProviderとStudio Stage1→Stage2を並行P0として維持。新Host・第二Canonical・新Gateは作らない（設計統合、追加E2E未実施）。

- [設計Projection・実ロード版・データ移行の横断受入](design/DERIVED_PROJECTIONS_RUNTIME_AND_MIGRATION_ACCEPTANCE_2026-11.md)：既存Data／Function／Businessと標準Graph／Ruleから、設計書・帳票・Go・API／MCP・DB移行へ意味を保って投影する。§7〜10で受注一機能を例に、Lookupの時点、Draft／Commit、承認経路Snapshot／現在Grant、生成byte／実ロード版、Migration／独立Readを一因子負例で接続。**最初は選択済みRule一件のSource位置→生成Go→実Builder入力、その後にCLI→実Provider→独立Consumer**。既存#1180/#2514と各Ownerへ引き渡し、新Host・第二Canonical・新Gateは作らない。Studio Stage1→Stage2の並行P0を維持する（設計統合・製品E2E未実施）。

過去の判断を本文に重ねて記すのではなく、現在の計画の基準を示します。以前の判断とその理由は、Gitの履歴、設計判断記録、課題からたどれます。


- [宣言的な成果物要求とデータ・プロセス合成](design/DECLARATIVE_RESULT_PROCESS_COMPOSITION_2026-11.md)：標準Data／Operation／Processから、結果要求・計算Graph・Node／Plugin配線と複数の実現方式（直接解釈／生成Go／DB委譲）を分けて扱う。**main統合済み#5087（既存Evaluatorと生成Go Binaryの五入力・Read回数の局所比較）・#5077（Data項目定義表→Rule→生成Go）を基線とし、同じ試験を作り直さない。§13〜14の次段R-0/R-1では「一入力・二Result」の粒度／条件／分類／時点／完全性を固定し、入力・定義・Scopeの変更からReuse／部分再計算／全件再構築／保留の根拠と棄却理由をread-onlyで示す。** R-2以降は正式Source→生成Go→実Builder・通常CLI Provider／独立Consumerへ接続し、既存#1130/#4410/#2622/#1180/#2514がOwnerを持つ。NodeとStudioの表示はN0〜N4／W0〜W4を利用する。新Optimizer・Runner・Scheduler・Host・第二試験正本は作らず、通常CLI実ProviderとStudio Stage1→Stage2の並行P0を維持する（計画、追加の実証はNOT RUN）。
関連文書は次のとおりです。

- [Wailsショーケース40件をTEIの能力検証へ接続する](research/WAILS_SHOWCASE_APPLICATION_CAPABILITY_CORPUS_2026-11.md)：公式掲載40アプリをData/Function/Process・Node View・UI・Plugin/Host・配布の能力検証事例として整理し、三つのReferenceとWC-01〜34を既存Ownerへ接続する。直接解釈とGo生成の同一Rule比較（#5073/#5118）、BuildPlan解決後の生成Go失効（#5109）、通常/生成Desktopの実Windows受領（#5059）を限定的な実物の足場にする。次は生成メソッド名に依存しない抽象経路、Focus/Pin/Dirtyの状態保存、実Providerと変更時の再資格化を小さな独立試験で確認する。既存の標準型・Mapping・Application API・Source/Build/Effect Evidenceを再利用し、第二のHost/Runner/Registryを増やさない。正式R2-P/R3および非Plugin Studio Stage1→Stage2 WindowsのP0を維持する（研究計画。追加検証の全件受理ではない）。


- [TEI実装から標準モデル・図・DSLへ折り返すdogfooding](design/IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md)：現行の二表記Rule→生成Go同値・出典分離と、生成Go ASTのRead／条件／byte位置を一つのI2M-1回帰に接続した。**コミット`68af0225`では`internal/semantic/validationgen`のGoテストが成功した一方、Repository Health全体は失敗**しており、局所成果と製品受入を分ける。**§12：原文・実Build入力の閉包**では、mainに入ったPR #5009の通常`tei self-build`による生成Goの限定実ビルドを踏まえ、次のR2-Pを「Rule原文・選択Reader・Generator版→生成Go→Build Intent→Builderの実入力→Artifact」を同一Attemptで照合する最小回帰に限定する。`definitionSource`は生成GoのEntrypointであり、元DSLへ置き換えない。公開Function Entry→actual Provider（R3）、現在Grant、Owner受理、Studio Stage1→Stage2 Windows（G6）は独立に検収する。独立Expectedは#1180/#2514、Ownerは#2621/#1121/#4544/#4495/#1824/#2831/#4583に維持し、新Schema・Runtime・Hard Gateは作らない。 **§13：生成元とビルド入力の二閉包**では、同じBuild IntentへSource名を並べるだけで出典連続性は成立しないと整理。現行`DirSourceResolver`が実byteを解決できることと、Rule Reader／Generatorが実際に採用した原Sourceの証明は異なる。PR #5056では生成Goの選択B／配置AをBuilder起動前に拒否する検査がmainへ入った。**§14.17の次の限定回帰**では、Plan解決後に`generated.go`だけを書き換えて旧Planを使う実CLI試行を拒否し、Artifact未発行を確認してから、復元したGoの正常な実Buildを確かめる（変更HEADでのCI合格は別判定）。元Rule／Reader／Generatorの資格と実消費byteの同一Attempt束縛は引き続き正式R2-Pの未受理部分とする。`definitionSource`のGo Entrypoint役割を保ち、新Source Kind・Receipt／Engineを先に増やさない。

- [実行モデルの因果図と独立証拠を照合する](design/PROCESS_EXECUTION_CAUSAL_PROJECTIONS_2026-11.md)：定義・Source・型付きRuleから、生成・直接評価・Provider・実Build・呼出し・Effect・Consumerまでを、証拠のある範囲だけ説明するread-only設計。§1〜39は因果関係・実行資格・時点・独立Evidenceの照合、§40〜47は実現方式とSource→Binary→Consumerの受渡し、§48〜56はメソッド名に依存しない抽象経路・不変条件・実Read IDの照合を扱う。**§57〜61ではmain済み#5142（値の状態とRelationの取得範囲を区別する`Decide`）、#5143（記録例と期待結果）、#5146（一行の日付比較）を反映。部分取得でも確定する違反と、未取得のため全体正常を証明できない場合を分け、Read ID・Presence・Coverage・Scopeを合わせて検査する。** 関係行間の期間重複は#5148の独立検収を待ち、業務上の`validAt`／`knownAt`や実Read Snapshotと混同しない。現行mainにはScenarioのOperator選択Pin、組織の型付き経路回帰、実CLI・直接Operation・Studio Hostの三入口回帰があるが、かつてのPR #5114／#5115／#5119が未マージで閉じた事実と、現行コードの存在・試験PASS・Owner受理は別に扱う。独立Expectedは既存#1180／#2514へ引き渡し、正式Source→Builder R2-P、通常CLIのactual Provider R3、非Plugin Studio Stage1→Stage2実WindowsのP0を優先する。新Runner／Host／Graph Store／第二Meaning・Expected正本や全般Ready Gateを設けない。


- [標準データ・Processからノードフローと軽量サービスを構成する](design/SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)：既存Data／Facet・Role・Operation・Processから、Node occurrence／Port／Edge／Service／Entry／Targetを分離してAPIマッシュアップ、Webhook、イベント連携に再利用する実装前設計。**第一差分は既存Ruleと純粋変換のread-only型付きFlow→CLIでの実生成・実Consumer照合**とし、外部HTTP／認証／配備は別受入。SF-01〜12の独立Expected候補を既存#1180/#2514へ渡し、N0〜N4／W0〜W4に接続する。新Runner・Registry・Hostは作らず、通常CLI実Providerと非Plugin Studio Stage1→Stage2のP0を変えない。 **§10はNode Viewの次段として、接続候補→意味適合→実装選択→実Consumerを同一視せず照合する。NW-Q0を既存Operation／Plugin証拠のread-only結合、NW-Q1を型付き純粋変換とGo生成／汎用Operator解釈の比較、NW-Q2をread-only API Facadeの段階検収へ割り当てる。NW-01〜10は既存#1180/#2514へ渡す未実行の独立Expected候補。**

- [Web標準UI・Web Components・AI開発ハーネスの横断設計](design/WEB_STANDARD_COMPONENT_AND_AGENT_HARNESS_2026-11.md)：Google Modern Web Guidanceを外部の知識基盤として利用し、HTML/CSS／Design Tokens／Web ComponentsをWeb/Desktop共通UIの基線に置く。Lit+TypeScript、Vanilla TypeScript、Lit+JavaScript/JSDocは同一インスペクタで比較し、静的検査・CEM・WebView実機・AI変更検証を通じて採否を判断する。既存React Starter・CodeMirror・SUI Canvasは排除せず、UI公開契約とTEI Action/Authority/Resultを分離する。#997/#1000/#1824/#1180/#2514へ受入を接続し、非Plugin Studio Stage1→Stage2 #2831/#4583、CLI実生成・実Providerの**P0を維持**する。**§9〜11では、現行Source→正規Bundle→Go Embed→実Windows WebView2のW0を先行実証し、同一read-only InspectorのVanilla TS／Lit TS／Lit JS比較W1、Component公開契約・Kit W2、PinとAI変更W3、TEI生成・Stage2 W4へ接合する。** Rendererを業務Meaningの第二正本とせず、Node/Graphの描画IDと正式Relationを分ける。UI方式の比較はStudio Stage1→Stage2／製品Reader・実Providerの新Hard Gateにしない。現時点は設計のみで、Google Skill CI導入／Lit製品採用／Theme PR #4628統合／実WebView検収は未実施。 **§12ではRule／Plugin側N0〜N4とUI側W0〜W4の対応を定め、共通Source・型付きOperation・失効・独立Expectedを共有する一方、表示部品とFlow生成のOwner、Node/CLIと実Windows受領を分ける。最初の二件はread-only Inspectorの公開Event分離（UI-P1）と、既存quantity_positiveの型付きNode投影／Go生成（FLOW-P1）とする。** **§13／PR #4949**ではread-only Inspectorの一覧・Graphからの選択要求を、Workspace ID/epochとSubjectだけを持つ一つの公開UIイベントへ接続する。従来のCatalog一意性検査・Reader・後着抑制を維持する狭いUI-P1実装であり、次は検証済みInspect結果のPresenter分離と独立Consumer試験へ進む。局所Node/型検査と実Node24・Windows WebView2・Stage2の検収を分ける。

- [Rule Authoringから実Plugin・製品CLIへ進む第一縦断](design/PLUGIN_RULE_PRODUCT_VERTICAL_SLICE_2026-11.md)：main統合済みの内蔵Rule Markdown Readerと製品回帰（#4890／#4891／#4900）を起点に、選択済みの `quantity_positive`（`std.decimal`）から**実判断Goを生成する最小実装**を[PR #4911](https://github.com/hat47x/tei-platform/pull/4911)でmainへ統合済み。R1は生成Goの比較・分岐、実Go Process、正値・ゼロ・負値・NULL・欠損、参照Attributeの変更追随を対象とし、Ownerの意味受理・実Builderによる生成byte消費・実Provider／通常CLI Call・Studio Stage2の成功には昇格させない。次は§14のR2で、生成Goの実byteと既存Builderが実際に読んだ入力を同じAttemptで照合し、ArtifactSetへ収録する。`tei/internal/semantic/validationeval` を参照する現在の生成Goは**TEI Goモジュール内での静的合成**を第一経路とし、外部Host用には公開API／Target-native Adapterを別検収する。**§15ではR2とR3を独立した並行経路**とし、R2は生成Rule→正式製品Builderへ、R3は受理済みpure Function一件の公開Entry→選択実Provider→共通Application API／通常製品CLIの二入口へ進める（#943/#2595/#951/#1251/#4495）。両経路の成功後に生成Ruleと実Providerを合流させ、Source改版・旧Binary・選択B／実Call Aを独立Expectedで拒否する。CLIの未宣言フラグ修復[PR #4940](https://github.com/hat47x/tei-platform/pull/4940)はmain済みだが、製品R2／R3や全体CIの合格とは別。Node-RED型Flow・簡易APIも既存Operator／標準データ構造／ProcessのNode投影として整理し、N0〜N4（read-only投影→型付きFlow→実Consumer→Studio GUI→外部API）の順に検証する。別Host／Runner／Rule Engineや第二Resolver／Expected正本は追加しない。PLV-01〜16／NG-01〜08は既存#1180/#2514へ引き渡す反証候補であり未実行のものを完了扱いしない。**Studio Desktop Stage1→Stage2 Windows実再入 #2831/#4583は引き続き並行する最上位P0**。


- [TEI Desktop製品契約（第18次統合・§0.18.8）](design/TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md)：**最優先は#2831/#4583のStage1実TEI→Stage2実Windows WebView2**、並行して正式`*.tei.md` Reader→実Go生成→通常CLI実Providerを進める。S-LOC-0の`source-locations.ts`／7件回帰はmainへ収録済みだが、Source位置表示・原文再取得S-OPENは別資格。§0.18.8では実`semantic-capability:sales.order.submit`（`semantic/sales/order.tei`、main #4853）を固定し、**P-01〜06**：実Go Inspect/Trace→S-LOC純粋投影→Pin/現在閲覧B非干渉→現行SourceからNode24で正規Bundle/Go embed同期→実Windows WebView2→Stage1実ProcessによるStage2再構築、を一つの受入経路に整理。Pin Fixtureはmain #4869でHostの四制限へ修正され局所Reader試験PASS、HTML ID修正は#4876で別検収。mainの現行埋込HTML/JSはPinをまだ配信せず、旧#4812の成功・旧生成Assetを再利用しない。現在Root/Source版、意味受理、開示許可、実Provider/Effectは別々に確認する。既存#1124/#3788/#1824/#4515/#2516/#1180/#2514を再利用し、S-OPEN・Updater・MobileをStage2の新Hard Gateにしない。

- [Function・Data Rule・集合処理・業務Stateの横断実証](design/DATA_DERIVATION_REFERENCE_CONTRACT.md)：**DataとFunctionの独立性**を保ち、Dataに普遍的な導出と、Function固有のQuery／集計・入力を分離する。項目の静的FacetとRuleの一次Sourceを二重管理せず、初期値・Draft・読取・保存前・確定Snapshotの評価時点／入力完全性を検証する（#1637/#1640/#1641/#2506）。[業務データ取込み](design/BUSINESS_DATA_IMPORT_DESIGN_2026-10.md)では検証のみ・全件原子確定・行別部分確定・再試行・採番同期を異なる保証とし、[Process／Invocation](design/PROCESS_LIFECYCLE_TRIGGER_OPERATION_SEMANTICS_2026-11.md)では申請時経路Snapshotと現在の担当資格・委任・人数条件を区別する。各案は既存#1129/#1130/#2622/#1174/#2386/#4198/#1180/#2514へ引き渡し、新Engine・Host・DSL・テスト正本を作らない。**検証例は未実行。正式Reader→実Go生成→通常CLI実ProviderとStudio Stage1→Stage2のP0を維持**する。 **宣言から実効への連鎖**：Rule／適用Phaseの宣言、Pluginの契約と実装入口、生成ArtifactとBuilder入力、通常CLIからの実Provider呼出し、DB Commit／Outbox／独立Readbackを別資格で照合する（RBE-01〜07）。Rule Catalogや説明文の存在を実実行・業務Effectの証拠にしない。既存#1637/#1129/#951/#988/#4495/#1174/#1180/#2514へ分担し、追加のRule Runtime／Hook Engineを設けない。 **単一参照の取得完全性（D9）**：[既存のRelation集合Reference](../reference/derivation-relationsets/README.md)では`scalar`／`optional-one`の`partial`／`unfetched`を`ErrIncomplete`として拒否し、完全取得済みの0件だけをNULLとする。Function Data Contextの読取ScopeとData Ruleの入力を混同しないための最小反証であり、#1706/#1637/#1640/#2622へ受入を引き継ぐ。合成Go Referenceと正式Reader／実Provider／Windows受領は別資格とする。

- [業務ルールの統合・決定表の検証](design/BRMS_DECISION_ANALYSIS_INTEGRATION_2026-11.md)：独立したBRMS Runtimeを設けず、既存のData／Function／Validation／Operatorとコード生成から判断・説明・検証を派生させる。**製品で確認できた限定的な前進**：内蔵`tei.rule-authoring/v0`の選択済みMarkdown→型付き`semantic.Validation`→既存Evaluator（#4890/#4900/#4901）、`std.decimal`正値判断のGo生成（#4911）、通常`tei self-build`での生成Go実Builder消費・実Binary実行（#5009）、原文byteコピーのDigest照合（#5051）。これらは**Ruleの元Locator／Reader・Generator実装資格を同一Build Attemptに正式束縛した証拠、公開Function→実Provider／独立Consumer、Studio Stage1→Stage2**ではない。§47〜49ではRule集合と入力Readの完全性を分け、有限Unique決定表を区分×開閉区間の交差・被覆から反例付きで検査する案を整理。まず#5056の選択Goと実Builder入力の取り違え拒否、#5054のOperator選択A→Bによる旧Scenario失効を既存Ownerへ引き渡し、その後R3／Stage2を独立検収する。生成元Markdownの実Binary同梱は公開可能Fixtureの限定実験であり、非公開Sourceを配布物へ自動埋込しない。#1180/#2514の独立Expectedを使い、新Parser／Rule Engine／Registry／Hard Gateを増やさない。

- [Studio／CLI／WebのBackend子プロセス・Plugin管理・配布更新の統合設計](design/STUDIO_CLI_BACKEND_PROCESS_DISTRIBUTION_UPDATE_2026-11.md)：現行Desktop P0は同一Process方式Aを基準とし、将来候補としてWails親からローカル子BackendへのIPC接続を比較する。CLIはWails非依存の共通Backendを直接／自己子起動／再利用Sessionで比較する。WebはServer配信Clientと認証済みAPIを分離し、遠隔CLIはServer側のPlugin構成を使用する。G0〜G6は既存#1158/#1824/#2044/#2589/#2831/#4583の実装Ownerに接続し、**CLI子起動の既定化・共有Daemon・Updater製品採用は実測とOwner判断待ち**とする。Studio Stage1→Stage2セルフビルドのP0を変更しない。

- [Studio Bootstrap／Release Closure／Plugin起動資格の実装契約](design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)：現行StudioのWails＋Go同一Process／CLI Directを基線とし、Bootstrapを起動検査・診断・復旧の最小経路に限定する。§25ではBOOTSTRAP_READY→CORE_READY→WORKSPACE_READY→OPERATION_READYを異なる資格とし、同一Releaseに含める実Image・UI Asset・標準Plugin Lock・既定設定SchemaをTarget別に検証する。Wails公式Updaterは署名検証と**一つの更新対象**の交換に利用し、D2単一Imageを第一候補、D3複数Imageの配布・活性化はInstaller/Deployment側で別受理する。利用者設定・秘密情報・DB／外部EffectはUpdaterで上書き・取消ししない。BRC-01〜09の独立Expectedを既存#1180/#2514へ渡し、#1824/#2044のWails安定版移行とD2実OS検証はP1、Wake／Mobileは実験扱いとする。**P0はCLI-firstの実生成・通常CLI実Provider・Studio Stage1→Stage2 Windows実受領**（#2621/#4495/#2831/#4583）に置き、子Backend比較#4635の結論を先取りしない。

- [Bootstrap／Plugin実体構成のP0接続](design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)：§26はPlugin管理S1の実在項目（declared／envSelected／resolved／lockPinned／artifactDigest）を用い、未実装のinstall／enable／init／invocation／effectを`notModelled`として維持する。既存#951 Resolver／#1016 Application Profile／#3267 RQ-037 BOMからContract→exact Provider→Build-only・Runtime・Delivery Member→実Image→Invocation／Current Grantへ至る**read-onlyな照合**を定義し、新Registryや万能Ready判定を作らない。まず正式選択された`*.tei.md`の同一Load byte読取S0-A（#2621/#1018）、通常CLI実Provider Read（#4495）をそれぞれ独立に閉じる。非Plugin Studio Stage1→Stage2（#2831/#4583）は並行P0のまま。FocusとPin、AI Harness、A→B→A失効、旧Reply・旧Consumerの扱いは既存#4515/#1180/#2514へ渡し、Release/Updater/D3/Mobileは前段の必須条件にしない。

- [Bootstrapから公開Plugin文書・Studio画面資産までの実検収](design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)：§27でmain統合済み#4800の`documentReadPreflight`（実Pluginの選択／exact版Lockのみ、Doc Source未束縛・Disclosure未確認・readEligibility未判定）と#4806の`InspectSubject.authorityLimits`保持・表示（参照Workbench）を接続する。#4812は最終HEADで9系統CI成功後も**closed／unmerged**のため、その検証差分を現行mainの配布画面へ自動算入しない。必要差分の後継PRへの抽出・再試験は#1824/#4583のOwnerで行う。main統合済みWindow 5例S0-P #4792は正式Reader S0-Aではなく、Plugin Manual Read #4714と業務`*.tei.md`のFrontend選択 #2621も別資格とする。P0の通常CLI実Provider #4495と非Plugin Studio Stage1→Stage2 #2831/#4583へ新Hard Gateを加えず、Release Closure／Updaterは後続P1で検収する。
- [Studio Bootstrapの現行実装回収・Release実証順](design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)：§28で**PRの旧HEAD CI成功≠現行main統合≠実配布製品受入**を明示し、閉鎖・未統合#4812のWorkspace/Bundle/Windows実装差分回収を#1824/#4583の小PRへ引き渡す。上位Pluginの直接依存について、実選択版・Lockを照合する#4826はmain統合済み。ただし#4800/#4826のDoc事前検査は閲覧・再公開の許可ではない。現行`cmd/tei-studio/go.mod`はWails `v3.0.0-beta.29`を利用。公式安定版v3.1.0と単一対象Updaterの採否・D2配布試験はP1でexact Pinを比較し、P0-S（製品`*.tei.md` Reader）、P0-R（通常CLI実Provider）、P0-B（非Plugin Stage1→Stage2 Windows）を優先・並行する。Source→Lock→Builder実消費→埋込UI→実Process→Invocation／Consumerの証拠は既存#1180/#2514へ分割して渡し、新Host／Registry／Ready APIを作らない。
- [Bootstrap起動前検査を実CLI・Studio・Releaseの受領証拠へつなぐ最小実装](design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)：§29ではmain統合済み#4826の直接上位Plugin実選択版・Lockを既存Resolver結果から投影する実Goと、main統合済み#4844のJSONL二重宣言修正を現行事実として扱う。新HEADで実CLI DiscoveryとLockを照合する#4845／#4854／#4857は元PRがclosed/unmergedでも、実CLI回帰・S-LOC投影と7試験・Language未接続の負例テストと同一Blobが現在mainに存在する。旧PRのCI成功は別資格。Plugin説明（L）、Studio Inspectと実埋込Asset（S）、TEI-native生成／Stage1→Stage2（B）の三経路をそれぞれ同一Subject/Revision/Attempt/実Image・Consumerで検収する。#4812の旧HEADの9系統CI成功は現行main収録ではなく、後継コードPRで再検証する。正式Reader S0-A #2621、実Provider R0 #4495、Studio非Plugin Stage2 #2831/#4583を並行P0とし、Plugin一Doc実Read #4714とWails安定版／Updater D2は独立後続。必要な新Assertだけを#1180/#2514へ引渡し、新Host／Registry／Evidence Store／万能Ready APIを作らない。
- [Bootstrapの現行Blob・配布Workbench資産・製品Language選択の断点](design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)：§29.6では旧#4845/#4854/#4857が未マージでも、実CLI／S-LOC／Reference Language負例の**同一Source Blobがmainにある**ことを確認。当該時点の旧PR CI／正式Doc Read／一般Language Readerの完成までは主張しない（後続の内蔵Rule実装は§29.7）。**実行時の停止線はWails配布資産**で、現行参照WorkbenchにはPin／Inspect制限DOMがあるのに`cmd/tei-studio/assets/workbench/index.html`には無い。#1824/#4583が現行Pin Fixture・HTML・Presenter→正規Bundle→Go Embedded Byte→実Windows WebView2を同一HEADで再検収し、旧#4812の生成Assetを無検証に流用しない。#2621/#1018のReaderはA0（製品受理済みContract/Lockから実Language Selectionが取れるか）を先に確認し、選択不可ならunsupported。A1〜A3は同一Load元byteとSpan・未選択拒否・旧`.tei`互換へ進む。P0-R実ProviderとP0-B非PluginStage2は従来どおり並行、Doc Read／S-OPEN／Updaterは新Hard Gateにしない。
- [内蔵Rule Markdown Readerの製品実装とBootstrap受入境界](design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)：§29.7に、main commit `7921b005`で実装された`tei.rule-authoring/v0`内蔵Readerの**Manifestによる一意選択→同一Load byte→`semantic.Validation`→Source Index上の正確な位置**を、製品S0-Aの限定的な達成として反映。main統合済み#4890の`internal/workspace/tei_md_rule_product_read_test.go`で選択あり／なし・不一致Contract版、二Rule・二定義領域の元byte範囲と説明文改版による旧Index失効を検査する。main統合と当該HEADのCI成功は別資格。外部Language PluginのResolver/Lock、Graph/Window、正式Meaning受理、生成Go／通常CLI実Provider、Studio Stage1→Stage2は未受理。WorkbenchのPin Host Fixtureはmain #4869、参照HTMLのDOM ID修正は#4876（Draft）、Go埋込資産とWindows実Consumerは別確認。既存#1180/#2514のIndependent Expectedへ委譲し、新RegistryやHard Gateは設けない。
- [Bootstrapから実Ruleの生成Go・CLI／Studio Consumerへの引渡し](design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)：§29.8ではmain統合済み#4890/#4891のRule Source・Inspector、#4900の旧`validation`との型付き意味一致・重複宣言拒否、#4901の**合成入力を用いる実Evaluator・Read要求8条件**を、到達済みの限定実装として分ける。次は既存[Plugin Rule縦断計画](design/PLUGIN_RULE_PRODUCT_VERTICAL_SLICE_2026-11.md)§11の`quantity_positive`一件から、型付き`semantic.Validation`→生成Goの判断→実Builder消費→Provider実選択→通常`tei` CLI／別ConsumerをOwner #1121/#4544/#2595/#1251/#4495へ段階接続する。Rule IDを同名Capability/Entryへ自動昇格せず、旧Source・説明文改版・条件改版・実入力差・生成器改版・Provider選択を別失効として検証。現mainのCLI `db-introspect`公開名／実Runnerと未接続Dispatchの不整合は競合する#4895（宣言除去）・#4896（入口接続）のどちらか一方を実Goビルド・別Process負例後に採用し、二重統合しない。Studio W0は#4869のHost Fixture→#4876参照HTML DOM ID（Draft）→正規Bundle/Go Embed→Windows実受領→Stage1自身によるStage2再Buildを独立に確認。外部Language Plugin／Graph・Window／Doc Read／Updaterを新Gateにしない。
- [起動契機と実行資格を分離したStudio／CLI／Web／Harnessの受入](design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)：§30ではCLI・Desktop・Web・定時処理・AI HarnessのTriggerをHost責務とし、共通Entry→Operation→実Provider→Effect／Consumerの意味と現在資格を分離する。標準Process／Node Viewはread-onlyの因果表示であり、新RuntimeやAI側のMeaning正本を増やさない。D1（Remote接続）と実行方式A（現行Native／Direct）の混同を訂正し、B0子WorkerとWails Updater補助起動のRole衝突を#4635/#1824で実OS比較する。main統合済み[実CLI同値試験 #5037](https://github.com/hat47x/tei-platform/pull/5037)はdirect Operationと別OS Process `tei inspect`のResult／Failureを同一Fixtureで照合した限定成果。子IPC・Updater・実Provider・Stage1→Stage2は未受理で、既存#1158/#4495/#2831/#4583／#1180/#2514への着手順は不変。
- [起動直前の実装選択と生成・解釈の受入](design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)：§31に#5051の実Binary原Markdown byte照合と、#5054のOperator選択変更後に旧Scenarioコマンドを拒否する未統合候補を別資格で接続。Generate／Interpretは既存の意味と独立Expectedを共有し、Builder実消費／標準Evaluator実Callの証拠を混同しない。#5042/#5052の生成PoC B実Windows成功も、正式Stage1→Stage2とは分ける。CLI B0一往復#4635と実Provider#4495を並行し、第二Runtimeや新Hard Gateは加えない。
- [起動の実証を実ReadとBuildPlanの現在性へ接続](design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)：§33は、main統合済み#5118の**G/P実Binaryの値＋実Read ID／回数**と#5109の**Plan確定後のGo byte改変を通常self-buildが拒否**する限定実績を、起動・実行・開示の適格性へ接続する。三入口Goテストは現行mainに存在（旧#5119/#5137は閉鎖）、HTTP API公開#5096は別資格。未統合のOperator選択Pin後継#5114は実装候補として扱い、旧#5054の未マージ試験をPASSへ加算しない。入口、選択実装、実Call、Audience、Effect Receiptを独立Expected（LA-01〜08）で分け、Generated Method名を不変条件の正本にせずSource／型付きRead／状態の抽象経路を使う。新Engine／Universal Readyを増やさず、通常CLI実Provider#4495と非Plugin Studio Stage1→Stage2#2831/#4583のP0を維持。
- [起動時点・実Build・実Caller・公開先の再確認](design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)：§34で**Source選択(S)→Builder実消費(B)→Provider呼出し直前(I)→Consumer受領(C)**の時点を分離し、同一Hashに戻るA→B→AでもOwner意味受理／現在Grantを自動復帰させない。main #5129では、Go生成物が変わらず原Markdown説明だけがBuildPlan確定後に変わる旧Planを通常`tei self-build`が拒否。main #5109の生成Go byte改版拒否と独立した負例（LA-09〜15）とする。三入口Goテストはmainへ収録済み（旧#5119/#5137は閉鎖）、API HTTP #5096・B0 #4635・Owner受理済み実Provider #4495・実Studio Stage2 #2831/#4583を別資格に保つ。新AdmissionTicket／Registry／Hard Gateは追加しない。
- [Builder・Toolchainの実体照合と未判定を起動契約へ統合](design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)：§35は現行mainの三入口Go実装、SB-C0 §4.56のBuilderファイルDigest照合／任意のGo Toolchain Closure Pin、RuleのNULL・未取得・partial判定（#5142）、Plugin Resolver診断被覆（#5147）を実装事実に合わせて整理する。L2-b局所ClaimとG4（現在UNQUALIFIED）、Process Image／Seed Trust、Hostの現在Grant、HTTP開示・実Consumerを独立資格とする。LA-16〜21を既存#1180/#2514と#2831/#4635へ割り当て、API公開#1158/#1824・通常CLI実Provider#4495・非Plugin Stage1→Stage2#2831/#4583の優先順位を維持する。第二Host／新Ready Gateは追加しない。


- [起動契機と実行資格を分離したStudio／CLI／Web／Harnessの受入](design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)：§30ではCLI・Desktop・Web・定時処理・AI HarnessのTriggerをHost責務とし、共通Entry→Operation→実Provider→Effect／Consumerの意味と現在資格を分離する。標準Process／Node Viewはread-onlyの因果表示であり、新RuntimeやAI側のMeaning正本を増やさない。D1（Remote接続）と実行方式A（現行Native／Direct）の混同を訂正し、B0子WorkerとWails Updater補助起動のRole衝突を#4635/#1824で実OS比較する。main統合済み[実CLI同値試験 #5037](https://github.com/hat47x/tei-platform/pull/5037)はdirect Operationと別OS Process `tei inspect`のResult／Failureを同一Fixtureで照合した限定成果。子IPC・Updater・実Provider・Stage1→Stage2は未受理で、既存#1158/#4495/#2831/#4583／#1180/#2514への着手順は不変。


<!-- rea-two-intakes-current-main-20261114 -->
- [REA外部候補から三要素Reviewへ進む最小検証](design/REA_THREE_ELEMENT_CANDIDATE_INTAKE_2026-11.md#10-main統合後の再整理観測と意味候補を二重に実装しない2026-11-14)：main統合済みの通常CLI external-candidates／R2 byte投影／ローカルStudio Host Reader（#4625/#4630/#4637）と、既存のthree-element rea-intake／Candidate・CarrierAudit・ReviewPacketを**二つのread-only入口**として役割分担する。§10では外部Identity／Source occurrenceと正式三要素Relation ID、modelref.Edge、描画Edge IDを区別し、同一Raw・複数Relation・問相対のCoverage・Native第二Product Consumer・Owner受理をJ-01〜05の独立Expectedに分ける。初回は既存Readerをもう一つ作らず、**同一Producer成果物の二入口・別Graph照合（semantic_graph／application graph）とRelation identityのLossのHeadless実証**を行う。Native Wails JSの実受領#4661、正式Source更新、旧新実測は後続の別資格。 Captured 6.8.0の実保存出力は既存TestReceiveRealSavedEvidenceがUTF-16LE/BOMと実Byte Digestを検証しており、J-01の二回のCI失敗でテキスト表示と元Byte列の取り違えが判明した。修正版では原本UTF-16LE/BOMをR2へ、別にUTF-8へ明示変換したJSONをrea-intakeへ渡し、二つの受領Byte/Digestを分離する。同じEvidence IDでも二Graphの関係IDや同一Attemptを自動同定しない。J-01a〜cの入力契約回帰を§10.6に固定する。**§10.7／J-02a〜dでは合成の同端点・同ラベル二Edgeを既存CLIでPreviewし、CandidateIDの個別保持とCarrierAuditの外部Edge ID Gap、重複Producer ID拒否を一つの局所Go試験で確認する計画**。Relation候補IDと正式Meaning Relation IDを同一視せず、Source逆編集・Native受領・Plugin実Codegenを別資格として#1124/#2497/#2514とmain統合済み#4703へ返す。**§10.8／J-02e〜gでは、CandidateSetの構造キーによる再採番候補と端点・向き変更の判別不能性を明記し、Comparatorに端点Mapping未確認の注意票と負例Goテストを追加**。外部Node IDの変更だけでOwner Meaningや旧Grantを復活させない。実Source A→B→A、Studio後着Pin、Codegen／Providerの実検査は引き続き別Owner受入。**§10.9／J-03では、合成Graphに加えて実保存REA 6.8.0のUTF-16LE原本から明示変換したUTF-8派生文書でも、Function/Businessがunsupportedのまま維持される局所CLI回帰**を追加し、未確認のRead/Writeや業務意味を不存在・未使用・削除可と断定しない。実Functionの問いはRequired(Q)／Consumed(Q)に照らし、正式Source・実Provider・Owner受理へ段階接続。**§10.10／J-02h〜jでは同一Source位置・同種の二Relationが再採番された場合、構造キーの決定論的仮対応と個々の関係の同一性を分離**する。再対応候補が複数ならOccurrence Mapping未確認を比較票へ追加し、単独再採番との正負Go試験を既存reacandidateに追加。媒体横断#4716やGraph→Go・Plugin自己構築#4709/#4722/#4720、Studio/Modeler#4694/#4696にはread-only受入条件だけを渡し、既存Host／Meaning正本／Gateを増やさない。**§10.11／J-02k〜mでは、同じCandidate ID・Source Digestでも解析器の`StructuralFacts`が変わる場合を区別し、`observed-facts-changed`というread-onlyの比較分類と再確認Noteを付ける**。これを正式なMeaning／Grantの継続やSource Patchの許可に使わず、Studio/SHOのConsumerでは未知分類をunchangedへ丸めない。J-02k/lの合成Go試験を追加し、J-02mの複合変更と実Consumer互換性は別Expected。新HEADのCIは別途確認。Studio Stage1→Stage2と通常tei実PG Function Readという製品P0は変えない。

<!-- rea-cli-evidence-body-qualification-20261116 -->
- **REA派生Requestの本文Byteと受領資格を一致させる**：[REA一Function縦断設計§32](design/REA_ONE_FUNCTION_EVIDENCE_TO_DESIGN_VERTICAL_SLICE_2026-11.md)で、`DeriveViewRequest`が`json.Marshal`で埋込Evidenceの字下げ・改行を失っていた問題を解消する。検査済み原JSONを派生RequestへByte維持で埋め、派生Request自体は独立Digestとして記録する。併せてCLIの旧Expectedを現行Viewのframing/control、未確認Parent Link、Tool不在時の保存済みRaw受領と整合させる。これらは既存Go CLI試験のR0独立検証であり、Producer Graphの業務意味受理・生成コード実消費・実Provider/Effect・Native Studio検収へ昇格させない。R1はRelation occurrence→Data/Function/Business→Resource/Event/Agent Roleの一問・Owner判断に絞る。独立P0の通常CLI実ReadとWindows Stage1→Stage2は維持する。

<!-- rea-jsonl-current-metadata-receipt-20261116 -->
- **REA 6.8実ViewのJSONL版差を現行Readerで検証**：[REA一Function縦断設計§31](design/REA_ONE_FUNCTION_EVIDENCE_TO_DESIGN_VERTICAL_SLICE_2026-11.md#31-rea-68のjsonl終端形式を実際の配送内容に合わせて受領する2026-11-16)で、以前の固定7行＋`$rea=...`終端と、現在の`{"$rea":"jsonl/v1","type":"metadata","records":...}`終端を同一意味へ無条件変換せず受領する最小修正を割り当てる。実data行数と`emitted`を照合し、`source`を全Source/Businessの完全性とみなさない。旧6.7/6.8保存Fixtureのbyteと検査を保ち、現行6.8の実Tool CIで`rea-evidence -view`と工具自身の負例まで合格して初めてR0-B完了と判定する。新Goテストと実Toolを別資格として扱い、Owner受理/正式Source/実Provider/Windows Stage2は独立検収のまま。

<!-- rea-real-tool-versioned-candidate-witness-20261115 -->
- **実REA実走の新たな受領事実と残った断点**：[REA一Function縦断設計§30](design/REA_ONE_FUNCTION_EVIDENCE_TO_DESIGN_VERTICAL_SLICE_2026-11.md#30-実rea-6768の同一ci実行から次の断点を確定する2026-11-15)／[PR #5038実CI](https://github.com/hat47x/tei-platform/actions/runs/43721894737)では、保存済み6.8.0の旧Receipt原本とは別のfresh`real-tool`を6.7/6.8で実行。両者とも実Producer Node/Relationの端点整合付きread-only候補選択とReceipt/Requestの成功が同一Attempt内で記録された。**6.7.0は全体completed・checks 4・notRun空**。**6.8.0はその2チェックの後、`tei rea-evidence -view`でexit 1**のため全体failed（負例二件未実行）。Python Verifierの端点反例10件PASSも、Go contract jobの旧`error-source-kind.jsonl`ファイル欠落による失敗と分離する。次のREA固有R0-Bは6.8のView Raw/JSONL FramingとTEI受領判定を診断し、同HEADの実Tool→Receipt→View→候補の全経路を閉じること。Graph取得成功を正式三要素/Owner意味受理・Source更新・実Function/Effect・Native Studioに昇格せず、P0の通常CLI実Provider／Studio Stage1→Stage2は並行維持する。

<!-- rea-pair-endpoint-consistency-20261115 -->
- **REA実受領の一組参照整合を強化**：[REA一Function縦断設計§29](design/REA_ONE_FUNCTION_EVIDENCE_TO_DESIGN_VERTICAL_SLICE_2026-11.md#29-producerの関係端点に基づく一組の選択を実rea入口で保証する2026-11-15)で、実Graph検証器が無関係な先頭Node IDとRelation IDを組み合わせることを防ぐ。Producer発行Relationのsource/target端点と観測Nodeが一致する場合だけ候補を選び、両端未観測はReceiptを維持して候補NOT RUN、重複Producer IDは拒否する。REA-PAIR-01〜04を既存Python transport mockで確認し、Graph付き実Producer受領、三要素意味受理、正式Source編集、Studio Native/実Providerは**別資格**として#3394/#1124/#4661/#2586/#1180/#2514へ引き渡す。Node Flow/Plugin配線/業務REAのRoleにも技術的端点整合と意味・時点・権限の区別を再利用するが、新しいGraph Core/Host/Parser/Expected正本を作らない。Desktop Stage1→Stage2と通常CLI実ProviderのP0は変更しない。

<!-- rea-two-lenses-promote-witness-20261115 -->
- **逆解析REAとResource–Event–Agentの接続を製品検収へ反映**：[一Function縦断設計§25〜28](design/REA_ONE_FUNCTION_EVIDENCE_TO_DESIGN_VERTICAL_SLICE_2026-11.md#25-逆解析reaと業務reaを三要素モデルへ接続する2026-11-15横断設計追補)で、外部REAを観測Candidateの取得経路、業務REAをResource／Economic Event／Agentの意味Roleとして**分離したまま**三要素・標準Graph/Rule/Processへ接続する。Functionの呼出し≠経済Event、Agent≠Principal/現在Grant、DB更新≠実業務Effect。外部Relation occurrence・意味Relation・図Edge・業務Keyを区別し、問相対P（実観測）/Q（必要根拠の閉包）/J（Owner意味対応）/A（生成・実実行）を単一Ready判定へ縮約しない。**現行mainの実REA 6.8.0保存原本はReceiptまででSemantic Graphなし**（#4962/#4990/#5003 main統合済み）。まず実Producer Graph/IDの有無を正しく受領・報告するR0 #3394と、合成一Functionの独立判別R1 #1124/#1180/#2514を並行。実Graphを合成Fixtureで置換せず、既存Decoder/Comparator/Reviewを再利用する。業務REA Roleの受理、正式Source保存、生成Go/Builder、実Provider/Effect、Native第二Consumer/Windows Stage2はそれぞれ別資格（#2621/#4544/#2586/#4661/#4583）。新REA Core、Reader、Host、Event Store、Expected正本を増やさない。Desktop Stage1→Stage2と通常CLI実Providerは引き続き独立P0。

- [REAの観測から一機能の業務設計・生成へ進む縦断実証](design/REA_ONE_FUNCTION_EVIDENCE_TO_DESIGN_VERTICAL_SLICE_2026-11.md)：現行R1/R2/R3-A/BのEvidence受領と、三要素Candidate/Reviewの二入口を区別したうえで、一Functionの問いをP（観測）→J（対応付け）→Q（根拠充足）→Owner受理→正式Source/Generator→旧L・期待D・新Tの独立検収に分解する。問別Required/Consumed・関係ID/多重性・再解析の曖昧さ・多媒体の矛盾・Current Grant/Disclosure・Consumer Attemptを保持し、最初はHeadlessの一問と独立Expectedから着手。Native実受領#4661と正式Source Applyは後続に分ける。TEI-native Graph/Rule→Go生成の局所Reference（#4733）を正式Plugin自己構築・製品CLI成功と取り違えない。Studio実セルフビルド#4583と通常tei実PG Function Context#2586のP0は維持し、新Host／Parser／意味正本／Gateは作らない。 **次のHeadless検証（§12）**：現行external-candidatesの一Observation＋一Relationと、three-element rea-intakeのReviewPacket／ThreeElementCoverageを別の読取証拠として確認し、実Function Read/Writeが未確認なら不足を答える。原Raw／派生byte、問い別Required/Consumed、関係の再対応とUnknownをRQ-01〜09として既存#1180/#2514へ引き渡し、新たなHost/Runner/意味正本は設けない。 **§16〜17の現行実装**：同じ構造キーの関係が2対2／1対2／2対1に一致する場合は、任意の個別対応を作らず`ambiguous-structural-match`で旧新を別々に保持する（main統合#4850）。既存Go `ReviewPacket.Summary()`にも比較のStatus別件数、未実行、未知状態、契約不整合を表示し、未認識Statusを`unchanged`へ丸めない（PR #4863）。この段階で確認できるのはread-only要約までで、Native JS受領・正式Relation/Source編集資格・Owner受理は未検収。 **§18のWire Projection補正**：`CandidateComparisonEntry`のGo値構造体が`omitempty`でも空の`beforeSource`／`afterSource`をJSON出力する問題を解消し、曖昧な旧新関係の未確認相手方はキーごと省略する。Go APIと比較状態は維持し、元Sourceの現在性・Owner受理・編集許可には昇格させない。 **§19の旧CLI出口補正**：`three-element rea-intake -before`が入力比較なしで固定Candidateを`unchanged`と返す旧経路は`REA_COMPARISON_UNAVAILABLE`としてfail-closed。旧`candidateSetFingerprint`の原本全体16進化は原本SHA256へ変更し、`fingerprintBasis`でRaw Byte Digestと候補同一性を区別する。実`reacandidate`比較を旧CLIへ接続するまで変更なし判定を再開しない。 **§20の型付き比較CLI**：既存`CandidateSet/v0`二件を`tei three-element rea-compare`で受領し、既存`CompareCandidateSets`→`BuildReviewPacket`と要約へ接続する。旧`rea-intake -before`は引き続きfail-closed、未知Schema・Fingerprint不一致・重複IDを拒否する。実外部Run・独立Native Consumer・Source編集資格・Owner受理は未証明。 **§21のSource-backed入口**：通常CLIに`rea-candidates -evidence [-output]`を追加し、既存REA Decode/Generate→型付きCandidateSetの保存→`rea-compare`までを別時点の読取専用E2Eとして検証する。受領Raw DigestとCandidate Fingerprintを分け、出力の上書きを拒否する。CLIテストの二重フィールドも是正するが、外部Tool実再実行とSource/Owner資格は未検収。 **§22の実保存原本再確認**：収録済み6.8.0の`analysis.json`はReceiptとして取得可能でも`normalized_result`にGraphがなく、実Observation/Relation候補は取り出せない。現行APIに存在しない選択フィールドを要求する旧テストを、Receipt受領＋`rea-candidates`の`GRAPH_MISSING`拒否の回帰へ置換する。合成Graph付きFixtureの比較成功を実原本のGraph取得成功と混同しない。 **§23の実REA CI境界**：旧検証スクリプトが存在しない`rea-evidence -observation-id/-relation-id`を呼んでいた誤りを是正。Receiptと派生Requestは常に検証し、実原本にProducer ID付きSemantic Graphがある場合だけ既存`external-candidates`を呼ぶ。Graph欠落は候補投影`notRun`として可視化し、実保存ByteやProvider成功資格を捏造しない。 捕捉済みAnalysis/ViewのDigestと失敗票をCI Artifactに残し、CI失敗の原因と実Toolの受領事実を分離する。 **§24の実原本と合成Graphの試験分離**：受領可能でも`normalized_result={}`の実保存6.8.0原本はReceipt検証とGraph欠落拒否に限定し、選択・重複・Derived Relationの試験は独立した明示的合成Fixtureへ移す。実CLI R3も合成候補成功と実原本候補拒否を正負両方で確認し、Go契約のpanicを架空の実Graph受領で埋めない。

- [Pluginの入出力を型付きで連結する：全体・Field・Adapterの三方式](design/PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md#19-入出力接続の粒度を選べるようにする値全体項目単位明示adapter2026-11-15横断設計)：同一契約の複合値は一括接続でき、必要なFieldだけを明示Mappingして入力Recordを構築することも、純粋Operator／選択済みProviderのAdapterを挿入することもできる。**受渡し粒度と変換方法を直交させ**、項目単位配線の煩雑さを標準Data／Semantic Type／型付きRef／Mappingへ委ねる。全体Portと展開Fieldは同一接続の表示粒度であり、二重Binding・非開示Field・NULL／unfetched・多重度・単位/精度・Effectを隠さない。最初はT0の全体直結／二Field Mappingをread-onlyで比較、T1一つの純粋Adapter、T2 Node View、T3実Provider／二Consumerへ展開する（PIO-01〜10を#1180/#2514へ引渡し）。新Port DSL／万能自動Adapter探索は先に作らず、CLI実Provider・Studio Stage1→Stage2のP0を維持する。
  **実Readまで接続する次段（Plugin I/O §20）**：main統合済み[#5118](https://github.com/hat47x/tei-platform/pull/5118)で、同一Markdown Data表／Ruleから直接評価（P）と生成Go（G）の実Binaryを比較し、値・結果が同じでもRead先Field IDが違えば拒否する限定回帰が成立。これを型付きMappingの受入に活用し、接続候補の型／Law適合、実Readの出典・取得完全性、Consumerへの実Delivery・Disclosureを別資格にする。最初はPIO-T0の全体Record／二Field Mappingと、T1の一段pure Adapterを既存Fixtureで検査する。PIO-11〜16は#1180/#2514へ引き渡す。#5109のBuildPlan解決後の差替え拒否、#5114のOperator実装選択Pinは別資格とし、正式R2-P／R3／Stage1→Stage2のP0を維持する（本追補に新Go・実Provider受領なし）。
  **状態と取得範囲の受入（Plugin I/O §21、2026-11-16）**：main統合済み[#5142](https://github.com/hat47x/tei-platform/pull/5142)の`validationeval.Decide`はNULL／未取得、集合のcomplete／partialを区別し、判断できない場合は理由付き`undecided`に留める。[#5143](https://github.com/hat47x/tei-platform/pull/5143)の記録例は意味検査であって実Provider Readではない。[#5146](https://github.com/hat47x/tei-platform/pull/5146)の同一記録の日付前後、[#5147](https://github.com/hat47x/tei-platform/pull/5147)の診断コード被覆も独立の資格。既存PIO-T0（Record全体／二Field）、T1（一段pure Adapter）へ取得完全性・実Read ID・Law・Disclosureを追加し、PIO-17〜23を#1180/#2514へ引き渡す。Draft #5148の期間非重複と実DB同時書込み保証、別PR #5150の指紋一本化は未受理のまま扱う。通常CLI実Providerと正式Stage2のP0は維持する。

- [型付き意味を直接評価するかコード生成するかの実現方式選定](design/PLUGIN_REALIZATION_AND_OPTIMIZATION_REQUIREMENTS_2026-11.md#10-型付き意味を解釈するかコードへ生成するかを機能ごとに選ぶ2026-11-15横断統合)：既存`validationeval.Evaluate`と`validationgen.GeneratePositive`の一Rule比較を出発点に、解釈・静的生成・混合・Target既存実装を**同じ意味要求に対する別実現候補**として選ぶ。先にShape／Law・Required Context・Target適合を検証し、選択実装・Model byte・生成Go／Builder／Artifact・実Caller・Consumerを別資格で照合する。コード生成を省略しても、モデル同梱・再配布・現在Grantの確認は省略しない。RZM-01〜08を既存#1180/#2514へ引き渡し、初回は一つのpositive Ruleを直接評価／生成評価で比較するテストだけを#1121/#4544へ渡す。新Interpreter/Resolver/Runnerは設けず、自己記述M2、正式R2-P／R3とStudio Stage1→Stage2のP0を維持。
- [横断Pluginの実Source・Provider・Consumerを一問で照合する](design/PLUGIN_CONNECTION_EVIDENCE_AND_COMPOSITION_SLICE_2026-11.md)：既存の`InspectSubject`／Source Index・Plugin Inventory・`composition.Selection`／`Service.Invoke`・Rule生成Go・Hookを**異なる資格の証拠**として扱い、同名同IDや結果値だけでは結ばない型付き接続検収票をHeadlessから実装する。最初のW0は一Workspace・一Ruleのread-only投影で、未確認のBuilder／Provider／Grant／Consumerを`unknown`として残す。W1はR2-P製品BuilderとR3公開pure Function実Providerを**独立に**実証、W2はStudio実Stage2受領、W3はGuard／Caller／Effectへ限定展開する。CPW-01〜10を#1180/#2514へ引き渡し、#951/#1251/#4495/#2595/#2600/#4198/#2831の既存Owner・P0を維持。新Resolver／Runner／Graph正本なし、今回の新実Go/Windows/Effect試験とOwner意味受理は未実施。
  **W0の実装着手点（§7）**：既存`InspectSubject`、Legacy `management.Report`、Service `Selection`／`Invoke`を照合しても、Rule→Function／Contractの正式な利用関係が無ければ結ばない。まずread-only W0-A/Bの誤接続拒否（CPW-01/04/03/05）を先行し、公開pure Entryの実Call R3と製品Builderの実byte消費R2-Pは独立検収とする。Graph表示、Builder、Grant、Effectを一枚の成功票に統合せず、既存Ownerのまま進める（設計・受入条件の更新であり新実行はNOT RUN）。
  **実装済み証拠からの次段（接続検収§8）**：main収録の#5007（Workspace Operator実装選択と局所Call）、#5009/#5021（Rule由来Go→通常`tei self-build`→実Binary）、#5020（Rule元Sourceと生成Go ASTの一試行照合）、#5025（Studio再読込時の旧Pin失効）を各Ownerの限定証拠として扱う。**次の最小Go修正は実装選択A→B後の旧生成Scenarioコマンド再利用拒否（CPW-11）**。その後、組込み実装選択とPlugin Lockの非同一、原Source→正式Builderの閉包、公開pure Entry→選択Providerの実Callを独立に結び、標準Node Viewへread-only投影する。CPW-11〜16は既存#1180/#2514へ引渡し、#5035のProjection／自己記述§49の試験を複製しない。Stage1→Stage2実Windowsを並行P0のまま維持する（設計追補・新しい製品Go／Windows試験は未実行）。
  **型付き貢献と接続証拠の次段（接続検収§9）**：mainへ入った#5051の元Markdown埋込みBinary照合をR2-Pの**原文コピー消費の限定証拠**として採用し、同一LoadのReader／Generator／元Sourceまでの資格束縛とは分ける。#5007のOperator組込み実CallはShapeとLaw、Workspace実装選択と正式Plugin Lock／Provider選択を分離して評価する。旧Scenarioの選択鮮度修正は既存Go PR #5054（審査中）へ委譲し、重複実装しない。次は一Operator Ruleの**定義→選択→局所実Callのread-only検収票**（CPW-17／20を先行）を既存CLI／Inspectorから作り、別RuleのBuild・正式Provider／Grant・Node View編集を未接続のまま残す。生成Desktop PoC B実Windows #5042/#5052と、正式Stage1→Stage2の製品受入も分離する。Go単一バイナリと既存Ownerを維持し、接続票を新Resolver／Graph Runtime／Expected正本にしない（§9は設計・引渡し、追加Go実行NOT RUN）。
  **接続条件の充足・配布・開示（接続検収§10）**：他レーンの[DB横断適合](design/DATA_PROVIDER_CROSS_DB_CONFORMANCE_2026-11.md)と自己記述§51を横断Pluginへ接続し、型付きPortの接続を**Required Data Contextの充足、Law適合、選択・実Call、現在Grant、Build／Runtime／DeliveryとDisclosure**に分けて検収する。mainの#5051で原Markdownの同一byteコピーが実Binaryに含まれても、元Source/Generatorの正式Build資格・意味解釈・配布先への原文公開は別。生成PoC B #5042とmain統合済みN2実CLI起動 #5059（同PRのWindows両Job成功）、Stage1→Stage2は別資格。組込みOperator選択の旧Command拒否は既存Go PR #5054の統合を待ち、二重実装しない。次は一Ruleのread-onlyな宣言→選択→局所Call票（W0）、D0実PostgreSQL一FunctionとR2-P／R3を独立して進める。CPW-23〜28を#1180/#2514へ引渡し、Node Viewは同じ証拠の投影とする。新Host／Resolver／Graph Engineや原文の無条件Runtime同梱は行わない（設計更新・新実試験NOT RUN）。
- [横断プラグインの役割別構成・実行境界・Studio自己再構築](design/PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md)：**Studio Desktopを正式な非Plugin Build対象としてStage1→Stage2→実Windows WebView2まで検証することをP0**とし、通常CLI／Studioから選択した実Providerへの到達を並行P0とする。Build時にだけ必要なGeneratorと生成製品に収録するProviderを分け、Lifecycle Graphを既存Operation／Hookに接続する（§1〜12）。**§13では構文・意味・演算・データ・表示・実行・検証の貢献別選択、Source／Anchor／Graphの往復編集、変更内容に応じた局所再検証、Studio／AIの対象固定と現在資格を整理**し、PCX-01〜10を既存#1180/#2514の独立Expected候補に引き渡す。**§14では必要Contract→exact Provider選択→Trait履行→工程・Target別の依存集合→実Builder／Provider→Consumer受領を、既存ResolverとBuildへ接続する。** Verified Composition BOM（RQ-037）、静的Go標準と外部Process／Wasm／Native配布の比較、DSL自己記述B0〜B4、PCC-01〜10の検証候補を加え、P0のセルフビルドを妨げない。新Grammar Engine／Resolver／Host／ABIの採用、実DB Effect、Owner意味受理は未実施。 **§15では依存先が契約で公開した図解・Relation・Obligationを、再公開許可・出典・版・現在開示の範囲で継承する。§16ではCLI/MCPからTEI定義→実コード生成→Plugin自身の生成割合拡大を先行実証し、正式Desktop Stage2の合格条件とは区別する**（PX-38〜47へ接続）。 **§17〜18ではApplication Graphに基づく横断拡張の対象選択と、Plugin説明・上位依存・短いAI Skillから現行Contractを辿る入口を整理する。§19では公開PluginのMarkdownマニュアル・型付きModel/Graph・Contractを同じDocumentation Surfaceから参照し、CLI/Studio/Harness間で固定された設計要素と上位依存を辿るD0〜D4実装計画と負例を定める（#4679）。** ただし、この説明経路を全体P0のStudio自己再構築や実Provider呼出しの必須前提にはしない。 **§20は二Node／二EdgeのGraph定義から出辺処理を実生成しCLIで受領する最小一機能のR0〜R5とPGF-01〜06を固定。§21はPlugin概要→実Contract→手順→上位公開文書→根拠を現在の選択版に結び、古いマニュアルとAI Skillの不整合をPHD-01〜08で検出する。** 両者とも#1180/#2514の独立Expectedへ接続し、全体P0を増やさない。 **§21.5では現行CLIの`plugin-lock`／`plugin-resolve`／`plugin-inventory`を最初の実入口に固定し、Plugin Doc索引の有無・実Contract版・上位公開参照・CLI手順を一件のread-only検収票にまとめる（H0〜H4）。** 未確認の`plugin docs`コマンドをSkillへ書かず、#4679/#2605とBackend設計PR #4690へ差分を引き渡す。 **§21.6では実`plugin-inventory`と`management.Row`に文書SourceRefが未接続であることをコードから確認し、main統合済み`testdata/tei-md-graph/`の固定Graph/Expectedを第20章の生成処理と第21章の公開文書検証で共用する。最初のGo差分は一Plugin・一文書の版／Source／公開範囲のread-only照合に限定する。** #4679／#4376／#1158に引き渡し、仮の`plugin docs` CLIや第二Registryは追加しない。 **§21.7はmainの6 Graph/Rule入力・実Go生成と同一BinaryへのNode別呼出し（#4729/#4733）を、公開Manualの実行例と同一Source／Contractの照合へ接続する。既存Source Indexで元byte位置と旧Artifact拒否を確かめた#4742を利用するが、Source登録は正式Reader/Role選択やDocument公開を意味しない。** S0のManifest管理Root・Pinned Source予備試験はmain統合済み#4751（旧#4745後継、**test-only**）。次の製品Go PRは、実`Workspace.Load`から選択済みFrontendへ同じPinned Source byteを渡すS0-Aの正負検証に限定し、#2621/#4704/#4730がOwnerとなる。文書の版・Disclosureは#4679が所有し、公開可能な通常Markdownの一Doc取得を業務`*.tei.md` Parser完成の新Hard Gateにはしない。 **§21.9ではPDC-0の最新main基点の合成A/B/C文書・Graph/Rule候補をGo監査するPR #4757の実CI成功（HEAD `d9219cf`、Repository Health・Cache Full/Incremental・脆弱性チェック）を記録した。これは候補Fixtureの整合性確認であり、Ownerの独立Expected採用や製品Doc Readの合格ではない。次のPDC-1（#4714）は実選択Plugin→明示Document Ref一件→適用Contract版／Source／現在Disclosure→read-only Resultを最小単位とし、業務`*.tei.md`正式Readerや独自Registryの実装を先行条件にしない。** **§21.8はmain #4758のGraph/Rule候補構造指紋を、Raw Source Fingerprint・生成Binary SHA・公開Manual SHA・Plugin Contract/Lock/Disclosureの別資格として整理する。** PDC-0の合成A/B/C・公開／非公開Manual・13ケース／14改変はmain統合済み#4757（対象HEADの3 CI成功、候補Expectedは未採用）が担当。実製品の最初の一Plugin・一Doc read-only取得はPDC-1 #4714が担当し、Fixture SHA一致をDoc開示・Provider受理へ昇格しない。 **§21.9のD1-0補強は**main統合済み#4800/#4826**が担い、実`plugin-inventory`から選択Pluginと直接上位の正確な版・Lockを既存Resolver結果で投影する。CompatibleMajorだけで厳密版を選ばず、上位文書のSource/Disclosure/閲覧可否は未受理のまま残す。** **[実CLI交差回帰 #4845](https://github.com/hat47x/tei-platform/pull/4845)は合成公開Manualの`tei commands`と実選択/Lock、未束縛Docの拒否側を同一Go Binaryで照合する。新HEAD CI待ちであり、D1-1（正式Doc帰属）～D1-3（元Source取得・現在Disclosure）は#4714/#4679の未実装範囲。**

  **§22では、横断Pluginの接続検収を一Operation・一Provider・一Consumerへ絞る。** Source/意味→Contract/Lock→工程別Build→実Invocation/Authority→Effect/Result→公開Doc/現在Disclosure→Consumerを、独立した既存Evidenceで照合する。最初の縦断は実CLI＋Studio二入口のProvider利用（#1251/#4495）と、一Plugin・一公開Markdownの製品read-only取得（#4714/#4679）を分離。Manual／Graph Layout／Rule／Provider／Grant変更の局所失効を#2600へ、重複しない負例を#1180/#2514へ引き渡す。**最優先P0のStudio Stage1→Stage2（#2831/#4583）を維持し、新Registry/Host/ABI/Scenario正本を加えない。** §22は設計・受入候補であり製品実行やOwner受理を意味しない。

  **§23は、Plugin Packageの導入ではなく貢献Role別の実消費と自己生成を検証する。** 正式Source Reader一件→Owner意味採用→Graph/Rule由来のGo生成とdelegate範囲の分離→通常CLIのselected Provider実Call→Studio/Workbenchの実Asset・Consumer→公開ManualとHarnessの現在開示をC0〜C5の異なるGateに分ける。次の製品Go PRはC0の一Graph/Rule Sourceに限定し、#2621/#1121/#1251/#2831/#4714のOwnerへ順に引き渡す。Window/REA/組織/BRMSは共通Source選択・出典の反証事例として扱い、専用Core/Hostを追加しない。反証PCG-01〜08は#1180/#2514の既存Scenarioへ差分候補として渡し、P0の正式Reader・実Provider・Studioセルフビルドを維持する。**Docs-only、実生成・実Consumer/Stage2・Owner受理は未実施。**

  **§24では、製品ReaderのA0〜A3を次の実装PRの最小受入に固定する。** A0は現行Contract/Lockから実Language Providerを選べること、A1は同じWorkspace.Load byte、A2はSource Index・元byte位置とInspector/Traceの対応、A3は未選択拒否・既存定義の互換性である。main済み#4853のread-only Inspector/Source Trace、#4850の曖昧なRelation対応拒否、#4826の上位Plugin exact版/Lockを局所実績として再利用し、これらを意味受理、生成Go、実Provider Call、Windows Stage2、公開文書Disclosureの完成へ昇格させない。#2621のS0-A一件と#1121/#1251/#4495の実生成・実CLIを二つの製品差分に分け、Stage2 #2831/#4583は並行P0のまま。既存#1163/#1180/#2514のEvidenceと独立Expectedを使い、新Registry/Runner/Hostは作らない（設計のみ、製品新受入なし）。
  **2026-11-15 実Ruleの製品実行証拠を更新**：内蔵`tei.rule-authoring/v0` Readerの実Workspace Load／Source Index／Inspector（#4890/#4891）、旧`.tei`との型付き意味同値（main b89209a2）、選択済みRuleを既存Evaluatorへ渡したRead Footprint回帰（#4901）がmainに入った。後者の入力はテスト用Sourceであり、実DB／実Providerの全件Readではない。第一実装対象は`sales.order_line.quantity_positive`の意味あるGo比較・分岐生成（#1121/#4544）、次に実Builder／Artifact、通常CLIの選択Provider actual call（#1251/#4495）とする。内蔵Readerの静的実装選択≠外部Language Plugin Resolver／Lock、Evaluator成功≠Owner意味受理／生成Plugin完成、CLI成功≠Studio Stage1→Stage2 Windows受領。既存縦断計画§12と横断Plugin設計§24.5〜24.6を参照する。新Host／Registry／Runner／Expectedは追加しない（計画のみ）。

  **§24.5〜25は、内蔵Rule Readerの製品実装を外部Language Pluginの受入から分離する。** mainの7921b005で、Manifest選択のtei.rule-authoring/v0→静的収録Implementation→同一Load byte→既存semantic.ValidationとSource anchorの経路が追加された。一方、外部PluginのResolver/Lock/実収録、正式Graph Reader、意味受理と生成Plugin実Callは未検収。次は#2621の内蔵Reader正負回帰と、#951/#1251の外部Selection・静的Build照合を分けて実装し、#1121/#4495の一Rule実生成→通常CLI実Providerへつなぐ。Rule集合の全件性と評価入力の全件性、HookのCurrent Grant/実Call/Receiptは別受入として#1180/#2514へ引き渡す。P0のStudio Stage1→Stage2 Windowsは維持し、文書公開・更新配布・実作用を新たな先行条件にしない。**

  **§24.5はRule Authoring Markdown Readerの限定的な製品実装を反映する。** mainのマージコミット7921b005で、Manifestのtei.rule-authoring/v0と内蔵Implementationを明示対応させ、同一Load原byteから既存semantic.ValidationとSource anchorへ投影する経路が追加された。未選択・実装欠落・曖昧実装を拒否するが、外部Language PluginのResolver/Lock、正式Graph Reader、生成Pluginの実呼出し、Owner意味受理、Studio Stage2の証明ではない。次の#2621はこの内蔵Rule Readerの独立Expected・旧tei回帰・Source失効の製品検収を先行し、#1121/#1251/#4495はRule由来の生成Go→実Provider/通常CLIの別PRとする（本追記はdocs-only）。


- [Script/Function起動・Generator・セルフビルドの接続計画](design/SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md)：**StudioのStage1→Stage2自己再構築を全体P0**とし、#4495の製品pure Function Entry→Host選択の実Go Provider→通常`tei` CLIと独立Consumerの縦断を並行P0とする。#4629/#4636でCapability宣言の実行前構造拒否はmain統合済み。追加の[§20 DSL/Workspace/Runtime四層診断](design/SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md#20-dsl-authoring診断と実行前宣言検査を一つの不正定義で照合する2026-11-14)は[#4652](https://github.com/hat47x/tei-platform/pull/4652)でmain統合済みだが、Workspace重複Input診断[#4655](https://github.com/hat47x/tei-platform/pull/4655)とSource→Adapterの回帰[#4654](https://github.com/hat47x/tei-platform/pull/4654)は別のGo Draftであり製品Entry受理ではない。実ReadはRA-0[#4626](https://github.com/hat47x/tei-platform/pull/4626)（17/18成功・Browser実APIの`sqlgen`依存除外FAIL）→RA-1/RA-2R統合検査[#4644](https://github.com/hat47x/tei-platform/pull/4644)（限定3/3成功・実PG16 Reference）→RA-2P製品Read→RA-3独立Consumerの順。取得範囲[#4647](https://github.com/hat47x/tei-platform/pull/4647)は観測された制限を`partial`にできてもNative SQLの完全性を保証せず、生成CLI[#4559](https://github.com/hat47x/tei-platform/pull/4559)は旧Sourceの未宣言入力を補うReferenceにとどまる。Studio[#4587](https://github.com/hat47x/tei-platform/pull/4587)は最新HEADで生成・通常Windows WebView2の両Jobが成功（B0、Draft）したがStage2ではない。Toolchain[#4648](https://github.com/hat47x/tei-platform/pull/4648)とState[#4537](https://github.com/hat47x/tei-platform/pull/4537)の別HEAD各8/8もStage2へ加算しない。**上位の完了条件は[実Consumer受領の横断設計](design/CONSUMER_QUALIFIED_CROSS_LANE_ACCEPTANCE_2026-11-14.md)に従い、Source・Authority・実Call・取得範囲・開示・Consumerを別Claimで検収する**。
- **2026-11-14 Script Launch実装接合**：[Script Launch計画§21〜25](design/SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md)で、Function/Entry/TriggerとExecution Bindingの分担、実行前の八検査、Provider不適合時の拒否、RDB/NoSQLの入力完備性とReceipt、実Go Provider/通常tei CLIの独立Witnessを明確化する。main統合済みの[Process Lifecycle設計](design/PROCESS_LIFECYCLE_TRIGGER_OPERATION_SEMANTICS_2026-11.md)と[実行モデルの図解](design/PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)を再利用し、新しいHost・Registry・Scheduler・Schemaは追加しない。**優先順位は、初回の意味実装・自己構築＝TEI CLI（MCPは任意）から*.tei.mdの図表・Rule→実Codegen→実CLI動作→Stage1 CLIによるStage2 CLI再構築とPlugin実Call、製品全体の到達目標＝Studio Stage1→Stage2 Windows実再構築、並行する起動P0＝#4495公開pure Entry→Host-selected actual Go Provider→型付きAPI／通常製品CLI、後続＝#2586実Read/Consumer・#4497 Trigger・#1174 Effect。Studio GUIをCLI初回実証の前提にしない。** 追加内容は検証計画であり、実ProviderやOwner業務意味の新たな受理ではない。 **§23では実コードに基づきAdapter呼出し前の拒否と呼出し後のResult契約違反を区別**し、後者を未実行・Rollback・安全なRetryとみなさない。入力値／Output Payloadの意味検証、Effect Receipt、実Consumer受領は既存Ownerに残し、#4495の最小実Call証拠と#1174の回復資格で別々に検収する。 **§24は#2831/#4675/#4676のCLI先行・プラグインTEI主導生成・定義の簡素化の計画と接続**し、Goへの委譲だけ、偽のStage1再入、上位Pluginの非公開図面の誤継承を合格にしない。上位契約から公開された図解・制約は参照可能だが、実行GrantとSource編集権限は継承しない。 **§25は現行製品CLIの実参照経路を棚卸しし、Product実Callとの区別を実装作業に固定**する。tei view -executeはReferenceExecutor/MemoryUnitOfWork、tei self-buildは子Process/Artifact観測であって、公開Entry→actual Go ProviderやTEI主導CLI Stage1→Stage2の合格に昇格させない。#4495の型付きAPI→実Providerを先に閉じ、通常製品tei別Processを次の差分にする。既存Failure Contractを保持し、Adapter後Result違反やEffect unknownを無条件Retryにしない。
- **2026-11-14 Script Launch実行配置・子Server接合**：[Script Launch計画§26](design/SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md)に、ADR-0048と[#4635](https://github.com/hat47x/tei-platform/issues/4635)のStudio子Server比較を接続する。**意味契約の共通化≠全入口の別Process化**。ローカルtei CLIはWails非依存・原則in-process、隔離が必要なOperationだけ明示的Child/Remoteへ委譲。**ローカルStudioはWails Client→薄いGo IPC Client→従属子Serverを設計上の基本経路**とし、同一exe Worker／別Server HTTP・Named Pipe/UDSの実装方式を同一Fixtureで実測し、Native一体を性能・障害の比較対照として残す。PluginのLock/選択規則は共有するが、Server内の実ロード、Host Grant、Artifact/Protocol Generation、実Provider Callは個別に照合。接続断・子Crash・更新中の版不一致ではBusiness Effect未知を安全Retryへ昇格させない。**#4495の実pure Entryと#2831/#4583のCLI先行・Desktop Stage2は引き続きP0**。子Serverの全面採用、Updater、Remote/MCP本番化は新Hard Gateにしない。新IPC/Plugin E2E・性能・Owner受理はNOT RUN。
- [標準データ構造・構文・図解・編集の意味保存](design/SEMANTIC_STRUCTURE_MAPPING_AND_DECLARATIVE_SYNTAX_VIEW_2026-11.md)：構造／評価／変更の同値性、Graph／Tree／Table・JSON Schema・合成Projectionの損失、Slot別編集資格を既存#2495/#2497/#4122へ接続する。Definition／Effective／Observedの型付きGraph、横断Hook、Plugin文書参照は#4198/#951/#1124へ渡す。**`*.tei.md`は通常Markdownとして可読**を基本条件とし、階層は入れ子リスト、小表はMarkdownテーブル、大表はCSV／TSVへの相対リンク、文書が大きければ別ファイルへ分割する。ER／シーケンス／FlowはMermaidを**直接Sourceとする場合**と、既存の受理済み意味から**導出図とする場合**を図ごとに区別し、意味の二重正本を作らない。**渡辺式ER図はModelerの用途別View**で、親を左・子を右、同じ配置段階は同列で上下に整列する。PK継承の識別関係とFKのみの非識別参照を異なる線で示し、複数親から子の左辺へ複数線を引く。Entity名の右側にカンマ区切り項目、PKは下線、二次識別子は`{...}`、導出・仮想項目は`(...)`とする。Mermaid ER内の**独立Entity宣言順を同列の上下配置に優先**し、未宣言Entityだけ初出順で補う。Relation宣言の並びを表示順や業務Keyに読み替えない。同一Source／Layout Profileで再現するが、GitHubの描画座標一致は要求しない。**Mermaidの`PK, FK`や識別線から親Key→子Keyの対応を推測確定しない**。#1126のCandidate Key／FD、論理Relationと物理PK/FK、#950のX-TEA Source construct（REFFER／FAMILY／SUBTYPE、ToKeyFields等）を出典付きで分け、未対応・未受理は原Sourceを保全する。第一縦断は①CUSTOMER→ORDER→ORDER_LINE、②二親PART／PROCESS→ALLOCATION、③不明なKey mapping／独立Entity宣言順／GUI編集の負例を、read-only →同一SourceのHeadless配置→一意Slot Patch→実Modeler Consumerの順に検証する。さらに、キーやRelation変更時には三要素分析法のデータ・機能・業務モデル、Function Data Context・CRUD・設計書への影響を#1124/#1143へ渡し、表示順だけの変更ならSource byte差分／Meaning差分／Build・Evidence再利用資格を別判定する。**ユーザー提供の業務フロー図・仕入れ／在庫のサブシステム別ER図を実図面Evidenceとして照合**。業務フローは担当者・契機・文書・現物・場所・台帳を併記するMethod Viewで、単なるMermaid flowchartや実行Workflowとみなさない。データモデル図は表ID／連結線／キー装飾／二次識別子／括弧付き項目と、同表が複数Subsystemに出現して`CRUD`／`R`が異なる利用区分を、共通Table IdentityとView相対表示に分離する。Mermaidは直接SourceまたはLossを明示したFallbackとし、渡辺式の箱内配置・線端・複数親・Subsystem表示は専用Projectionで評価する。画像の出典digestは保持するが図面そのものは再配布しない（#941）。図の色・端点・項目装飾の厳密な意味や未添付の業務／機能モデルは、**同版の`.xead`定義・Modeler helpを#950/#1031/#4615で確認するまで未受理**とする。既存X-TEA三視点読取りPlugin #4570の合成試験を、実図面での適格性へ昇格しない。V-0〜V-4の画像観測→原Source/help→read-only Headless→Studio実Consumer→限定Source編集を#1124/#1125/#1126/#3250へ引き渡す。#3030／#2621／#4370のMD-0〜MD-4、SM-01〜98は既存#1180/#2514へ引き渡し、第二Expectedや新Parser／Graph Compiler／Writerを作らない。**全体P0のStage1→Stage2 Studioセルフビルドと通常`tei` CLI→実Provider→独立Consumerを優先**する。
- [Markdown-first宣言的構文とAI支援の多経路モデル化](design/SEMANTIC_STRUCTURE_MAPPING_AND_DECLARATIVE_SYNTAX_VIEW_2026-11.md)：**既存#4359（DSL構文）／#2621（Markdown Authoring）／#1018（Language Plugin）を一本のAuthoring縦断として扱う**。通常の`*.tei.md`は見出し・本文・入れ子リスト・Markdown表・Mermaidを優先し、複雑な規則だけ低ノイズの`rule`等を選択済みLanguage Pluginで表す。大表はCSV／TSV、Native SQLは原文を正本として参照する。見出し／Fence／リンクの存在はSource選択や業務意味の受理・実行権限を意味しない。旧`validation`・短い`rule`・Markdown表・Operatorを、**同じRule IDと独立Expected**で比較する。現`main`の[`tei-md-graph-reader`](../reference/tei-md-graph-reader/README.md)は、固定6件のMarkdown表からNode／Edge／一行Rule候補と原byte位置を読み、実Go条件分岐を生成する。同じ生成Goを**一度だけBuildした同一Binary**に全件／Node別の入力を渡し、未知・明示空ID・余分な引数を拒否しつつ、独立Expected候補とBinary SHA-256不変を検査する参照実装まで進んだ。**これは正式Reader／Language Plugin／Meaning／製品CLI／実Provider／セルフビルドの成立ではない。** 次はA-1 Workspaceで選択されたMarkdown ReaderとSource Index→A-2受理済みGraph／Ruleから実Target生成→A-3選択Plugin・通常CLI・独立Consumer→A-4Stage1→Stage2とStudio実再構築の順に進む。**ホワイトボード写真、会議録・音声、Excel/CSV、帳票/PDF、既存画面、DB/SQL/コード**は、媒体ごとの観測位置（画像領域・時刻・Sheet/Cell・頁・画面要素・byte span）と元Source版／開示条件を保持したうえでAIが候補化し、同じData／Function／Business・Relation／Ruleへ対応付ける。観測／AI推測／業務上の採用を区別し、複数の資料の不一致は質問と独立Evidenceで解消する。素材とCandidateの入口は#958/#950/#3394、MMI/SHO・SUI/SEI/TAIと既存Pluginへ委譲し、TEIの意味・Source/Role・Acceptance・Projectionは#2497/#2620/#2516/#2514が担う。最初の実証は、**手描き図＋会議文字起こし＋CSVの非機密合成資料から発注―入荷のRelation一件を、根拠付きMarkdown表／Mermaid候補→未決質問→独立Expected→既存Source受理→別Consumerへ投影**する。情報の欠損・矛盾・Evidence水増し・隠れた実Effectは負例とし、実検証が終わるまで採用／実装済みと呼ばない。第二のIntake／Semantic／Evidence Store、万能Parser、TEI共通Host、P0 Hard Gateを作らず、**Stage1→Stage2実セルフビルドと通常製品CLI→実Provider→独立Consumerを優先**する。 **段階横断の受入資格**をRaw Source→Observation→Candidate→Source-bound Draft→Accepted Meaning→Consumed/Observedに分け、各段階で原資料版・媒体別Selector・Subject（未確定時は仮ID）・Scope・選択Parser/Plugin版・開示条件・不足を追跡する。既存[#4515](https://github.com/hat47x/tei-platform/issues/4515)のSubject固定選択、[#4503](https://github.com/hat47x/tei-platform/issues/4503)の限定Evidence取得、[#2516](https://github.com/hat47x/tei-platform/issues/2516)のAI用文脈と[#3419](https://github.com/hat47x/tei-platform/issues/3419)の共通操作を通じ、選択→資料照合→AI候補→対象差分Preview→限定Source変更→別Consumer再読取を一巡させる。画像の交差線、会議とCSVの多重度矛盾、標本での一意性、AI回答の重複Evidence、古いSource位置、描画用Edge ID、非開示資料、説明用Ruleの二重取込を独立Expectedの負例へ渡す。**生成コードへの単純委譲ではなく、TEI定義から実構造・制約・Pluginをどの程度生成できたかと、Stage1→Stage2／Windows実Consumerまでの根拠を別々に実測**する。未実装ReaderやAI媒体入力をP0の新Hard Gateにしない。 構文の正式化では**Sourceの選択**と**Markdown内の対象範囲・意味Roleの選択**を二段階に分け、固定日本語見出しを公開Grammarとして固定しない。変更資格は「Source変更と再生成」「同一Artifactへの実行時入力」「表示のみの差分」「Plugin/Generator版変更」で個別に判定し、A-1正式Reader→A-2既存意味／生成器→A-3製品Plugin・実Provider・独立Consumer→A-4Studio自己再構築へ渡す。検収票は意味同値／原文byteと編集資格／再生成と同一Binaryの入力差／実Consumer／TEI定義による構造・制約・Plugin生成率を比較する。並行[PR #4694](https://github.com/hat47x/tei-platform/pull/4694)の渡辺式・X-TEA原典整合を上書きせず、§31と§33〜42の差分を統合前に再照合する。 **規則・判断・Data Operationsの構文は同じ入口で扱い、意味は分離**する。Validation／Derivation／Decision／Lifecycle／Effect前提を新Rule Engineの一つの処理へ潰さず、Markdown決定表はUnique方式を初期比較候補とし、複数行一致は競合、未取得／NULLは判断不能として検証する。Mermaidが直接定義元になれるER／Sequenceの範囲と、Graph参照例の表正本＋派生Mermaidを図ごとに区別する。三要素分析法のData／Function／Businessの対応、Function Data ContextのField／Relation path／粒度・必要完全性は明示し、図に関係線があるだけで自動Fetch／業務Effectを増やさない（#2622／#1124／#2497）。 **§43の統合引渡し**では宣言的構文と三要素分析法のViewを同一Source／Role／Accepted Meaningでつなぎ、B0固定Bootstrap→B1選択Language→B2自己記述候補→B3独立受入→B4生成・Stage2を区別する。次の一辺はA-1/S0（選択Workspace Source byte→選択Reader／Role／元byte spanの製品接続）であり、Graph/Rule参照実装の同一Binary入力試験を製品CLI／実Provider達成と見なさない。SM-91〜98は既存#1180/#2514の独立Expected候補へ引き渡す。
- [宣言的構文の合成契約と共通ノード表現](design/DECLARATIVE_COMPOSITION_AND_NODE_SURFACE_CONTRACT_2026-11.md)：Data／Function／Business・Relation／Rule／Query／Operation／Pluginを、対象・関係・型／粒度／取得範囲／Effectの契約で接続する。Node／Port／配線は用途別のread-only投影から始め、データ線・実行因果・Plugin選択・表示用の線を混同しない。構造・評価・作用・Source編集・再利用の保存条件とDC-01〜12の一因子負例を既存#2514へ引き渡す。最初は内蔵Rule Reader→R1-M（型付き一箇所変更追随）→R2（実Builderによる生成Go消費）を検証し、CLI実Provider／Studio Stage1→Stage2のP0を維持する。新Graph Runtime／第二Semantic／万能文法は導入しない。 **2026-11-15追補（§8〜12）**：文書からの宣言選択・意味の合成・用途別投影を分離し、型／粒度／時点／Read範囲／Effectによる接続条件と、Lossを伴う図面の編集境界を整理。選択済み内蔵Markdown Rule→既存Validation／Source Index→参照Transform Generator→生成Goの別Process検証をR1-Mの新Focused Go試験として追加する。数量から閾値への一因子変更、旧入力へのFallback、NULL／欠落、非数値型拒否を固定期待値で試験する。ただし現実のProvider I/Oと製品Builderによる生成Source実消費（R2）、製品CLI／Windows／Stage1→Stage2は別資格であり、CIとexact HEADが確定するまで新試験は未受理。 **§18〜20（2026-11-15横断更新）**：Markdown表・Rule・成果要求・実装選択・実観測を、同じ表層でも異なるRoleとして既存Data／Validation／Query／Provider／Evidenceへ入場させる。設計時の接続適格（三判定）と実証結果（PASS／FAIL／NOT RUN／UNSUPPORTED／UNKNOWN／STALE）は別に表示。main #5007のOperator局所Call観測と、未マージ提案 #5010のData項目定義表を混同せず、DC-21〜26を#1180/#2514へ引き渡す。次の一件は実製品Workspaceから一RuleのSource・型・Read・不足証拠をread-onlyで照合すること。R2実Builder／R3通常CLI実ProviderとStudio Stage1→Stage2は独立P0を維持し、新Writer／Runtime／第二意味正本は採用しない。 **§21〜23（2026-11-15実装接続）**：現行mainのData項目定義表Reader・既存Data同値のGo回帰、#5007のOperator実選択・局所Call、#5009の生成Rule通常tei self-build、#5020の原Rule位置⇔生成Go AST試験をそれぞれ実証範囲内で評価する。Source形式／Role／受理Meaning、OperatorのLaw適合と選択変更時の旧Command失効、元Markdown→Generator→Build Intent→実Builder→Consumerの同一Attempt出典束縛を次の受入対象とする。DC-27〜32を既存独立Expectedへ渡し、CLI／Studio一Rule read-only診断はP1とする。正式R2-P・公開pure FunctionのR3・Stage1→Stage2実WindowsのP0は変えない。 **§24（2026-11-15製品検証）**：選択済みMarkdown Data項目定義表→既存Data／Attribute／Ruleの一Load→Product Inspector／Source Indexの同一原byte位置、説明改版とA→B→AのSource失効をFocused Go試験へ接続。これをRuleの既存Inspect回帰に対する異種表層の対照とし、公開InspectorにないReader情報をStudioが表示できると仮定しない。CI exact HEADとOwner受理は別に確認。正式R2-P、通常CLIの実Provider、Stage1→Stage2実Windowsの独立P0を変えない。 **§25〜26（2026-11-15実CLI接合）**：main #5037のRule CLI／Operation結果一致と、Data項目定義表の実CLI別Process照合を対にし、Data／Attribute／Ruleの厳密Kind・ID、元Source、説明改版、AuthorityLimitsを同一read-only契約で検証する。#5051のRule原文byte埋込みBuildと#5052の生成Workbench実Windows成功は別Consumerの限定成果で、正式R2-Pの元Source／Generator出典閉包とStudio Stage1→Stage2のP0を代替しない。DC-33〜38を既存#1180/#2514へ渡し、今回の新CLI回帰はexact HEAD CIで個別検収する。 **§28（2026-11-15実装接合）**：既存§27のG/P/D実現方式を、Data項目定義表と隣接Ruleの同一Workspace.Loadへ接続するFocused Go回帰で具体化。Pは既存validationevalの五入力・論理Read、Gは同じ型付きRuleから既存validationgenで生成したGoの参照先／条件を確認する。decimal→text型改版時のValidator・生成拒否もDC-39〜40で検証し、Gの実Binary比較は未実施として区別。R2-P正式Builder、R3実Provider、Stage1→Stage2実Windowsの独立P0を変更しない。 **§27.4〜27.7（2026-11-15、解釈と生成の二実現）**：同じMeaningを既存Evaluatorで解釈する方法と、選択済みRuleをGeneratorでGoへ変換する方法を、Target別の実現選択として区別する。実装向け図表属性は業務Meaning・標準データ構造と混ぜず、既存Provider／Target／Plugin契約へ割り当てる。新Focused試験は一Ruleについて直接Evaluatorと別Processの生成Goを独立Expected（正値・0・負値・負の0・NULL・欠落・型不適合）で比較し、診断コードと物理Readの同値までは主張しない。次の最小工程を実Builder入力取違え拒否（#5056）→原文・Generator資格の同一Attempt束縛→通常CLI実Provider→Stage1→Stage2とし、Node-RED型Flow／Batch／Plugin配線の共通Node Viewは意味契約のread-only投影とする。PR #5009／#5051／#5052／#5058の個別到達点を実Provider・全面自己構築へ合算せず、P0とOwner境界を維持する。 **§29（2026-11-15、実Read同値反証）**：印付きMarkdown Data表と隣接Ruleを同一Meaningで読み、既存P（Evaluator）とG（生成Goの実Binary）へ六入力を通して結果・Error・Number実Read ID／回数を独立Expectedで照合するGo回帰を追加。Gの読取先のみ変え、同値の結果を返す負例も実Binaryで観測する（DC-42／43）。これは正式R2-PのBuilder入力閉包、R3実Provider、Stage1→Stage2ではなく、Target・Scope・時点・開示・Grantの一般同値も未受理。DC-44／45は既存#1180／#2514に引き渡し、G／P／D方式分類とHook／混合配置を直交させる。 **§30（2026-11-15元宣言のPlan後変更）**：#5109で導入したPlan確定後のgenerated.go改版拒否と独立に、生成Goはそのままで同梱元Markdownの説明だけをPlan確定後に変える反例を通常tei self-buildへ追加。旧Plan／Intentで拒否し、L2-bを主張せず成果物を公開しないこと、復元後に記述的Plan Fingerprintが戻ることをDC-46で検証する。元Source・Go・実Builder入力は別々の現在性を保持し、G／Pの値一致をArtifact再利用や現在Grantへ昇格させない。正式R2-P、通常CLIの実Provider、Stage1→Stage2はP0の独立検証を維持する。 **§31（2026-11-16 Rule種別の実現適合）**：main #5142/#5143/#5146で値の状態を区別するDecide、記録例、暦日比較not-afterがMeaningとして増えたことを、P／G／Dの対応範囲と分けて検証。既存Data表の暦日版＋Ruleを製品Readerから同一Sourceとして読み、PはNULL・未取得・if presentと二日付Readを評価するが、GのGeneratePositiveはnot-afterをunsupportedとして拒否するFocused Go試験を追加。日時型への一因子変更もValidatorで拒否する（DC-50〜52）。新Rule種別の実現成功をR2-P／R3／Stage2へ合算せず、未対応G／未選択Dを無音Fallbackしない。
- [数理・統計・計画・科学向けツールキットの設計・検証計画](design/CROSS_DOMAIN_MATH_STATISTICS_PLANNING_TOOLKITS_2026-11.md)：ウィンドウ集計と帳票、WBS/PERT/CPM、Solver・実験計画法、数式・物理系・生命科学を題材に、既存のDSL・標準表現型・Operator・Providerの適用範囲を確かめる（[#4660](https://github.com/hat47x/tei-platform/issues/4660)）。独立した期待結果と、Mockから実際の計算Providerへ移る際の受入条件を先に定義する。TEI Studioと一般アプリケーションの双方で再利用できる設計を目指すが、Studio DesktopのStage1→Stage2セルフビルドを最優先とする方針は変えない。
- [汎用プログラム実行モデルの図解・Target間移植](design/PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)：Bootstrap／Trigger／Entry／Operation／Effectを型付き関係と実行規則から投影し、意味・実効計画・実Trace・図面を分離する設計案（#4198／#4502／#4193）。§16〜19では処理分割・言語移行の契約保存と局所再検証、§20〜23ではSource・静的解析・実Trace・AI推定の対応資格、Projectionの情報損失、Target同等性の観測範囲、SQL/NoSQLの粒度・取得条件、PX-30〜37を既存Ownerへ引き渡す。**Studio Stage1→Stage2と実Entry→実ProviderをP0**とし、新Graph Compiler・Host・Runtime・Hard Gateは先行導入しない。実Target検収と正式意味受理は未実行・未採用。 **§24〜27では上位Pluginの公開Facet・継承の情報境界、差分Graph合成とObligation維持、TEI-native Plugin自己生成のHeadless実証、PX-38〜47を既存Ownerへ引き渡す。** それぞれの実装・検証・正式採用は未完了。 **§28〜32でApp/Route/Entry/Operationの全体Graph、任意計測と必須監査の差、Target/Consentごとの適用、CLI・Studio・Harnessの対象固定、PX-48〜58を既存Issueへ接続する。** **§33〜36では定義・選択・実現・実観測を同一Subject・Target・Attemptで照合し、問いに応じたGraph完全性、CLI/Studio/HarnessのInstruction・Subject・Capability三閉包、Markdown優先の未実装Readerの扱い、最初の一枚のread-only検証票を既存Ownerへ引き渡す。** **§37〜39ではCLIセルフビルドの同名二Node・別ID多重Edgeを固定した変更対照表から、意味差・実Go生成・Builder実消費・実Provider・独立Expected・増分再検証を一試行で追跡する。Raw Source再読／意味再評価／Artifact再Build／Evidence再利用／現在Grantの五つは別に判定し、#2595/#2600とSB-0〜4の既存作業へ引き渡す。** **§40〜44ではHostのLifecycleと一件のInvocationを分けた図解、受注確定のRule・DB Commit・通知・Outcomeの別証拠、BRMS／組織／REA／B2C／Batch／Pluginへの適用境界、図→意味→Go→実Provider→Consumerの一Subject照合と既存P0への引渡しを追加する。** 最初の実証はread-only照合と既存Graph/Ruleの意味ある生成から始め、正式Reader/Meaning受理・実Effect・Stage2を局所Referenceの成功で代用しない。#4198/#4495/#2586/#2831/#4583/#1180/#2514が既存Ownerを維持する。 **§45〜50では意味・実行・観測の三面図、必須先行／並行／順序不明の因果関係、呼出し前後の失敗とEffect unknown、構造・評価・実行の三種類のTarget同等性を整理する。** Graph/Ruleの固定Fixtureと公開pure Entryを別Subjectのまま検証し、正式Reader S0-A、SL-1a〜1c、実BuilderとStage2 Consumerを各Ownerの既存検収へ接続する。説明・図・コード・試験は同一意味から派生させるが、独立ExpectedとOwner意味受理は別資格とする。 **§51〜56では定義／選択／実現／観測を一つの対象で比較するread-only実行図、肯定・否定・全件・Effectの証拠範囲、再起動後のReceipt照会と現在資格、CLI／Studio／AIの同一対象表示を既存Ownerへ引き渡す。** 最初はGraph/Rule固定Fixtureと公開pure Entryの別経路から始め、製品S0-A／実Provider／Stage2の資格は各実Consumerで独立検収する。 **§57〜60では現行mainのWindow五候補S0-P・Plugin exact Lock事前検査・REA観測事実変更・Workbench参照制限表示と、Script LaunchのTEI内評価／契約付きMock／実Providerの差を、問いに必要な証拠の結合条件へ整理する。** **§61〜63では現行`InspectSubject`の対応Kindにある一`semantic-capability`を起点に、Source参照位置の表示（S-LOC）と現在Root／原byte／権限を再検査する原文復帰（S-OPEN）を分け、Headless→Workbenchの実消費を既存Ownerの最小検証へ引き渡す。** §63.1では`reference/workspace`に実在する`semantic-capability:sales.order.submit`について、Canonical ID、Source byte span/Artifact Fingerprint、Trace一致、存在しないCanonical IDの拒否を確かめるGo回帰を追加する。§63.2ではCapabilityの一度のWorkspace読取を使って、末尾改行だけの変更とA→B→Aを検査し、Canonical同一でもSource/Index/Artifact版が失効し得ることを固定する。 §63.3ではmainに実装された内蔵`tei.rule-authoring/v0`を別の正例とし、**ManifestのLanguage選択→実Workspace.Load→二件の`semantic.Validation`→同一LoadのSource byte anchor／Inspector**をGo回帰へ接続する。説明用`text`領域の第三規則は読まれないことを負例で確認する。 続く§63.4では**実選択済みRuleのCanonicalを既存`validationeval.Evaluate`へ渡し、固定Expected・Read Footprintを8ケースで照合するGo回帰**を追加する。さらにRule IDと入力データを維持したまま、Markdown内で参照Attributeだけを変更し、実EvaluatorのRead先・結果・Source/World改訂が追随するかを一因子変更で検査する。これは既存Go Evaluatorへの委譲であり、生成Goと実Builder/Providerの使用証拠は別に取得する。数量0の業務上の不成立とNULL／未取得の評価エラー、明細番号重複と取得範囲不明を区別する。入力は厳密な合成`Source`であり、実Driver／ProviderのReadやRule集合の完全性は未検収である。Graph/Windowの製品Readerや外部Language Plugin・実Provider資格は引き続き別検収。**Headlessの局所確認と、未対応のGraph/Window Reader・実Provider・原文Open・実Windows受領は別の段階**として扱う。 Source Index位置、Inspectorの参照限界、Plugin選択、実Provider呼出し、Stage2の資格は連結したと推測せず個別に検収する。 Studio生成/埋込Assetの版一致は#4812、正式Readerは#2621、実Entryは#4495、Stage2は#2831/#4583で別検収し、新しいRuntime・Evidence Store・Hard Gateは増やさない。
- [理想の利用像を製品の受入へつなぐ計画](workstreams/PRODUCT_COMPLETION_PLAN.md)と[製品設計](design/PRODUCT_COMPLETION_DESIGN.md)は、設計書の編集、実接続、標準構成、帳票、複合表現、分野拡張、運用・費用をG01〜G71の作業へ結ぶ。最新の型・表現、DSL、ETL、プラグイン管理の段階計画へ接続し、申請等の一巡から添付、検索、通知、外部契約、継続イベント、デバッグ、利用者の管理を補う。評価は [#3796](https://github.com/hat47x/tei-platform/issues/3796)と各担当Issueで行い、既存のヘッドレス実証からGUIへ進む順序と、実Readを含む現在の主線を保つ。
- [Issue #2586](https://github.com/hat47x/tei-platform/issues/2586#issuecomment-6773141846) ：Plugin・Function実Read・三視点Model Map・実Effect・Build・Consumerを、**宣言／実選択／実入力／実呼出し／受領／現在資格**で区別する横断受入計画。§9はS2-reference（#3540）・限定L2-b（#3785/#3795）・限定L3・Hookの実績を反映。**S2-product／Function R0／全体G4は未資格**。各担当は#1251/#2586/#2831/#1124/#2600等を維持し、新Gate/第二正本を設けない。
- [Lifecycle Graph・AI Harness・三視点Modelerの統合資格とS0〜S2実装引渡し](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md#23-lifecycle-graphai-harness三視点modelerの統合資格2026-11-12) ：§23〜32.5で固定Selection Context、Graph Patchの構造/意味/実行資格、Source/Plugin/Artifact/EffectのEvidence Join、既存Owner別A〜Eバックログ、**テーブル列削除→モデル写像削除・UIは未マッピング警告（§27.7の六診断状態・§27.8のCRUDレビュー資格分離）**、**Function/Script/Entry/Trigger/実Artifactまでの失効伝播（§27.6、#4502/#4495）**、三要素分析法のMethod/Plugin化、**通常開発相当の意味レビューとAI候補の出典確認**、**観測参照0件≠未使用という問い相対の判定**、**Graph→Generatorの適格性をG1規約→G3独立Conformance→G2合成へ分割（§30、#4544/#4198）**、**#4539実PG Readのmain統合と製品`tei` CLI→Studioへの残差（§31）**、**Studio Desktopの生成部品実使用P0-A→TEI Stage1実ProcessによるStage2起動P0-B→X-TEA三要素Plugin化P0-C、DS-S0〜S4の受入（§32、#1824/#2831/#1124）**、**Source Indexからの高速参照移動とPin→AI Preview（§29）**、**§29.6の参照ナビゲーション・近傍直接テスト・実Entry起動・実Read・Effectの証拠を分離した統合受入（#4543/#4502）**を扱う。**#2586の横断受入、#1251の実Consumer、#1124/#1126のModeler/Schema、#3503のLifecycle、#1180/#2514のScenario、#267/#4143のAI/OKF契約を置き換えない**。S0〜S2、DMU-01〜15、NAV-01〜10、#4472 TE-CR-01〜06との交差条件、および#4502/#4495の既存起動負例との交差照合は検証用Reference Slice/独立Expected候補。局所合成実績とS2-product/Function R0/全体G4の未資格を区別し、汎用Coreや通常コーディングに三要素の必須制約を加えない。§29のP1a限定修正（#4392/#4524）はmainマージ済み。実Consumerは未実証。§29の実装順はP0実Provider/実Readを維持し、P1a Map/Locate/曖昧IDの整合（#4392/#4524）→P1b 一件のread-only参照往復（#4543の近傍直接テスト照合は並行可能）→P1c Pin送出Preview（#4479/#2516）→P2安全な逆編集→P3実Workspaceの性能・認知負担測定とする。**§32.5では新しいSource束縛Function生成Draft #4559の限定実績と残差を確認し、最優先D-W（実Wailsで生成部品を実利用）・独立F-B1（PG read-only Function生成/手書きReference比較）・並行B-S（Stage1自身によるStage2再入場）を分離。** 現行#4559はDraft/未統合、入力fromは未宣言、SQLはSource Index対象外、生成READ ONLYの実書込拒否と変異生成器実検出は未確認。D-Wの実証は**既存のPoC B生成Workbench→generated server/Chromium**と**通常Windows Desktop→WebView2**（さらにPoC A CIのgenerated Windows exe Buildのみ）という分かれた基線を、同じSource/Generator/実Build/実Windows WebView2 Consumerへ結ぶ**未接続の一辺**に限定する。既存compose-ui/connect-workbench等を再利用し、生成Assetの未import・手書き同値代替・serverのみの合格・旧Workspace後着を独立負例にする。#4559の実Rowset一致をEnvironment実BindingやStudio実証へ昇格しない。OwnerはD-W=#1824/#1121、F-B1=#1121/#2586、B-S=#2831の既存Issueのまま。文書のみ・通常CLI/Studio/Stage1→Stage2・X-TEA移植実接続、DS-S0〜S4/NAV/Script/Generator GG/D-W/性能の新試験NOT RUN。
- `product/`
- `requirements/`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/MILESTONES.md`
- `plan/research/TECH_RESEARCH_INTEGRATION_2026-10.md` ：直近の調査結果を製品価値や既存の課題に結び付ける統合計画
- [DeepSeek Harnessに学ぶ構成・AI-first連携・横断検証（研究案）](research/DEEPSEEK_HARNESS_TEI_CROSS_PRODUCT_ADOPTION_2026-11-01.md) ：外側のAgent HarnessとTEIのService Contract/Plugin/Host Authorityの境界を比較。正式要件・実実行の受理は既存担当へ
- [DSHのGoal・Schedule・通信境界と大規模AI並列開発（研究案）](research/DSH_SCALE_GOAL_SCHEDULE_EGRESS_RESEARCH_2026-11-01.md) ：軽量Profileの自律性、独立Workstream、既定Request拡張によるSession履歴送信、検証責務を明示。実証担当は[#2509](https://github.com/hat47x/tei-platform/issues/2509)／[TAI #234](https://github.com/hat47x/tai-pleroma/issues/234)。
- [DSH実験的Agent Teamsの並列開発・専門家連携・用途別Context（研究案）](research/DSH_AGENT_TEAMS_CONTEXT_EXPERT_HANDOFF_2026-11-02.md) ：Team Task DAG/CAS/共有Checkout・非強制writeScope、SEIの専門家照会、#2515実効隔離・#2516用途別Projectionを統合したT-01〜T-12反証案。既存Ownerと受理を維持する。
- [外部Harnessの結果継承と選択的再資格化（2026-11-02研究案）](research/HARNESS_EVIDENCE_CONTINUITY_AND_SELECTIVE_REQUALIFICATION_2026-11-02.md) ：DSH Headless NDJSONの投影/截断/Session再開と実Git/TEI/Oracle Database/Providerの証拠の差を識別。#2514独立検証基準、#2582 Reactive、#2595実Source、SEI #767の用途別根拠十分性に接続し、Claim単位で局所再評価する。
- `plan/REFERENCE_MODEL_TRANSFER_GATE_2026-10.md` ：成熟した外部の参照モデルをプラグイン中心で移し替え、TEIの表現力や暗黙の前提を横断的に検証する計画
- `plan/design/ARTIFACT_SET_SCOPE_AND_CONSUMER_READ_EVIDENCE_2026-11.md` ：導出成果物の対象集合・内部整合・公開世代・実Consumer読取を分けて検証する**提案中**の計画（#2482／#2604。製品Gateや受理条件の変更ではない）
- `plan/workstreams/TRIAD_MEANING_AND_CROSSLANE_ACCEPTANCE_HANDOFF_2026-11.md` ：Data・Function・Businessの意味を保ち、移植・DB移行・組織Relation・Plugin境界における既存Ownerの検収条件を照合する**提案中**の横断引渡し（#1125／#1126／#1129／#2619。新Gate・業務意味の採用ではない）

- `plan/design/AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md`：AI Tuning Hint/限定合成観測Reference、性能要求×Workload×実現×独立Evidence、Typed Lifecycle/Plugin/Cache/Query/Headless比較（§1〜19）。§20でCacheの速度・費用・正しさ、Commit/Hook/Outbox/CDCと失効の区別、AT-33〜36を定義。§21で一FunctionのNo Cache対Request-local/Memory（必要時Query/Batch）をW1〜W8の同一Workload/Expectedで比較。§22でH0（AIなし）/H1（DSH等外部Harness）/H2（別Harness/Provider）の任意接続とTool完全受領・開示・費用の別資格を定義。**§23（2026-11-12）**でStudio/CUIのFocus非依存の固定Selection（Qualified ID/Source/Revision/Scope/Grant）と、Product別権限を引き継がない共有Harness Session、stale時の明示再選択、Tool欠落/未承認Apply負例を接続。#2586の実Function→実PG→Rowset→ConsumerをP0とし、固定Selection/AI送出は後続。**§24（2026-11-12）**でGitに小さなManifest/Expected、Raw Trace等はLocal CAS/NAS/S3へ選択保管し、Identity/Storage Resolution/Qualification/Disclosure/Lifecycleを分離。EV-01〜08は既存Scenarioへの反例候補で、Local-onlyを正規経路とし、外部HarnessへのRaw Evidence自動送出を禁止。**§25（2026-11-13）**でDB Read Snapshot/Subject Pin/Row Edit Selection/Pending Changes/Commit Attempt/Receipt/独立Post-commit Read/Cache Projectionを分離。R0実Product PG Read→R1 Headless読取/Pin→R2単一行条件付き更新→R3独立Read/Cache失効→R4任意AI Harnessの順とし、RW-01〜08を既存Scenarioの未実行反例候補へ接続。Read-only A/Bは編集完了を一律前提とせず、更新鮮度のClaimにはR3相当の実証を要求。既存AT-01〜36、T0〜T4、#2586/#3262/#3271/#1180/#1452/#1824をOwnerとし、新Host/Registry/Hard Gateを追加しない。実Read E2E、Cache性能、実GUI/CUI固定選択、外部Harness完全受領、Owner受理は **NOT RUN / NOT ACCEPTED**。
- `plan/design/AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md` ：生成AI向け設計ヒント・19件の限定合成観測Reference（§9〜10）、性能要求×Workload×実現×独立Evidence、Typed Lifecycle、Plugin/Cache/Query/Headless比較（§11〜18）を維持。**§19（2026-11-09）**で#2586のM1b→R0（Function→Query/Relation→実Provider→Rowset→Consumer）を最優先の一Attemptに固定し、merged #3689/#3761のHook実位相/Outcome検査、merged #3757のBinding実呼出し範囲、merged #3717の実PostgreSQL試験用CI設定を各資格上限付きで統合。CacheのTTL/Hook/Build成功≠鮮度・実Read・性能改善を維持する。旧版に混入した重複見出し/旧T0〜T4表を正規化し§1〜19へ整理。**§20（2026-11-09）**でM1b→R0後の一Function A=No Cache/B=Request-localまたはMemoryの再現可能な性能実験（W1〜W8）、独立Expected・実Call・p95/p99/Origin負荷・測定費の二段階採否、必要時のC=Query/Batch改善対照を定義。E0〜E4は既存T0〜T4の内訳であり実測未実施。**§21**ではStudioのUI要素選択→用途限定Design Context→DSH等外部Harness→AI改善候補→TEI独立検証をH0（AIなし）/H1（外部Harness）/H2（別Harness/Provider）で比較。外部送出/Tool完全受領/OS隔離/契約上の接続可否/コストを別資格とし、AIを必須Runtime化しない。AT-01〜32と既存T0〜T4のOwnerを継続し、新Optimizer/Core/Host/Hard Gateを設けない。実Function→実DB、キャッシュ負荷計測、AI自動適用、正式Owner受理は **NOT RUN / NOT ACCEPTED**。
- [高度B2C向けHeadless・Conversion Engineering設計（Issue #3509）](https://github.com/hat47x/tei-platform/issues/3509) ：独自Frontend/標準表現型の任意選択、Journey/Experiment/Effectの別証拠、販促変更レーン、実Consumer受入と反例を扱う**提案中の設計・計画**。旧PR #3512の全文は[Issue内に保存](https://github.com/hat47x/tei-platform/issues/3509#issuecomment-6772845918)し、設計書を第二正本として復活させない。新Runtime/Commerce EngineやCVR改善の実証を意味しない。
- `plan/design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md` ：原子的Traitの実装義務、Manual/Generated/Providerでの履行、合成・不適合診断を既存#951/#2495/#3271/#3267へ接合する**設計・実証候補**（[#3278](https://github.com/hat47x/tei-platform/issues/3278)）。新Resolver/新Hard Gateや実装完了を意味しない。
- `plan/design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md` ：Web/Desktop APIの例外・Client Reaction・Graph/Table/Traceの**横断設計と段階的な実証計画**（[#3274](https://github.com/hat47x/tei-platform/issues/3274)）。**P-1a/b**：E1の構成時Guard DAGとRuntime HTTP Decision Pathを区別し、[実Node専用CI #42396614454](https://github.com/hat47x/tei-platform/actions/runs/42396614454)の限定HDG-21 CharacterizationはPASS。全体Healthは同PRでgofmt不一致（別Source）によりFAIL、Go vet/test/Workspace validateはPASS。**P-1c**：既存Go Reference Hostのdefer Auditと10経路Callback/Status/Stageを、[Node+Go専用CI #42461546535](https://github.com/hat47x/tei-platform/actions/runs/42461546535)で限定PASS。**P0-R（限定実証）**：E1・Goの独立宣言は両側が局所的に構成可能でも不一致になることを[Node/Go実CI #42462531490](https://github.com/hat47x/tei-platform/actions/runs/42462531490)で確認（Node 7/7＋Go package全PASS）。**P0-S（限定CI合格）**：実`reference/workspace`の`application-pattern:order_entry.submit_order`と`semantic-capability:sales.order.submit`・Source Index/Canonical参照を既存`operations.InspectSubject`で検査し、[Node/Go専用CI #42464774706](https://github.com/hat47x/tei-platform/actions/runs/42464774706)＋同HEAD全体Health #42464774715でPASS。**P0-B（限定CI合格）**：既存`driver.ResolveCommandBinding`を実Workspaceで通し、欠落・重複・有効だが意味的に違うCapabilityの負例を[専用CI #42470312187](https://github.com/hat47x/tei-platform/actions/runs/42470312187)でPASS（同HEAD全体HealthもPASS）。**P0-I（限定CI合格）**：実`NewHost`・`NewCanonicalActionAuthorizer`・実Driver`InvokeCapability`から合成`Capability Adapter.Invoke`までを一つのHTTP Attemptで通し、Session/CSRF/認可拒否・Adapter障害・Result契約違反を[専用CI #42474818698](https://github.com/hat47x/tei-platform/actions/runs/42474818698)でGo/Node PASS。**同HEADの全体Health #42474818899はfailure**（当時`plan/design/README.md`に並行追加AI性能設計書の目次登録なし。現main目次には登録済み）。**P0-E（限定CI合格）**：同一実Host＋実Driver＋合成Adapterで、202/503と合成状態の5つの反例・明示Observation readの失敗を[専用CI #42486408608](https://github.com/hat47x/tei-platform/actions/runs/42486408608)でPASS（Node 7/7＋Go全PASS）。**同HEADの全体Health #42486408590はfailure**：全体gofmt/vet/test/Workspace validateはPASSだが当時の別AI性能観測Workflowが未ピン留めaction・timeout欠落でCI契約違反。**現mainのWorkflowは修正済み**（別HEADであり過去CIをsuccessに変更しない）。HTTP status／Audit／正常Result≠永続業務効果、read natureはsynthetic、外部Receipt・Client完全受領は未実証。**上流Composition接続判断優先**：[判断資料 PR #3549](../plan/workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)のC-4（実行を外部Hostへ委任する現状維持）を含め、C-4→C-1/C-2→C-3の順で製品実行経路を定めるまでHTTP専用Registry/Routerを増設しない（[設計§20](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)）。**C-1参照実行の成熟差分（PR #3573経由）**：Environment Binding→既存`routing.Router.Dispatch`→Host登録の合成Executorという技術的成立を確認したが、製品CLI/実外部Providerへは未接続。新しい[設計§21](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)と[Composition判断資料§9](../plan/workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)では、Executor入場後のエラー／不正ACKを`refusedCount`（入場前拒否）に数えない**Reference v1補正**と合成負例BXE-01〜06を追加し、`unverifiedCount`（`ranCount`の内数）として保持する。[限定Go CI #42505232348](https://github.com/hat47x/tei-platform/actions/runs/42505232348)で実`Router.Dispatch`の6反例・既存Router回帰・Go vet/gofmtがPASS（v1 Reference限定）。**C-1/C-2は別のOperation役割**：配備Command（C-1）とService Request（C-2）は独立し得るため、二者択一を一律強制しない。同一Operationで実行Ownerを重複させず、C-4外部Host委譲も対象／Targetごとに未決。C-2参照はCarrier入場とProvider Process実呼出しを分離し、`Schema/Rule/RuleFingerprint`をv1として保存済みv0の版を流用しない。[C-1/C-2限定CI #42555954660](https://github.com/hat47x/tei-platform/actions/runs/42555954660)で両ReferenceのGo正負・vet/gofmtがPASS（[判断資料§11](../plan/workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)／[HTTP設計§22](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)）。**両案とも製品への採用判断は未決**であり、HTTP 202/503・Executor/Carrier ACK・実Effect Receiptを一つの成功/失敗へ畳まない。**横断S2の現況と次のGate**：[main統合PR #3540](https://github.com/hat47x/tei-platform/pull/3540)で通常CLI `inspect`と通常・生成Studio Hostから**静的Reference Inspector Provider**を既存Composition経由で選ぶS2-referenceが成立し、exact-head Health/Wails/WebView/CLIはPASS。一方、正式Profileで**Provider B選択→両実ConsumerがB実呼出し**するS2-product、Plugin実Artifact/Entry/Current Grant/2×2 BOMは未検証。HTTP側はこの残差を先行Gateとし、結果分類・Effect unknownをドッグフーディングする。**11/09成熟差分（[HTTP設計§23](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)）**：C-3 [#3692](https://github.com/hat47x/tei-platform/pull/3692)/[#3720](https://github.com/hat47x/tei-platform/pull/3720)はBinding由来Executor IDを既存Route/BuildPlan/Supportsで解決した**実行前Reference**（v0）であり、C-1/C-2 v1より遅れた未実行機能ではない。固定IDと実解決IDが異なる反例を保持する。Hook [#3721](https://github.com/hat47x/tei-platform/pull/3721)は`NotSatisfied`→Command拒否語彙を限定対応し、`not-satisfied`と技術ErrorをCommand拒否へ昇格しない。外部Plugin語彙検査、CLI/HTTP Clientへの実連動は未実証。実PostgreSQL [#3717](https://github.com/hat47x/tei-platform/pull/3717)の[専用CI #42570360880](https://github.com/hat47x/tei-platform/actions/runs/42570360880)は4系統でDSN設定・非SKIP・有限実試験PASSだが、別DB・別OperationでありHTTP Host→同一Operation Commit/Receipt/Client反応のE2Eではない。[Design Context継続Token #3723](https://github.com/hat47x/tei-platform/pull/3723)の`STALE_SNAPSHOT`も現在Grant/実Provider資格とは別。**HDG-X01〜06（NOT RUN）**を独立Expectedとして#1016/#1043/#1158/#1174/#3262へ引渡し、実Consumerを最優先にする。**11/09直近成果（[HTTP設計§24](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)）**：[main #3761](https://github.com/hat47x/tei-platform/pull/3761)の実`hook.Runner`が`ValidateOutcome`で未知名を`HOOK_OUTCOME_UNDECLARED`として拒否し、語彙内／空／`not-satisfied`とCommand拒否を分離（[全体CI #42606902787](https://github.com/hat47x/tei-platform/actions/runs/42606902787)成功）。外部Plugin Transport→HTTP/IPC/Clientへの受信・表示は未実証。[#3757](https://github.com/hat47x/tei-platform/pull/3757)はEnvironmentの6種Bindingを調査し、`ObservationProviders/EvidenceProviders`は通常CLI`-plan`で**解決済み**だが、製品`observation.Router/evidence.Router`の実組立てと`Observe/Collect`の実Invokeは未実証（[判断資料§12](../plan/workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)）。[#3760](https://github.com/hat47x/tei-platform/pull/3760)のself-build in-process Catalog+digestでPlan Fingerprint一致、`bound`は限定確認済みだが**`qualified=false`／L2 claimなし**。Plan導出を実Runtime Artifact/Current Grantに昇格しない。前段[#3736](https://github.com/hat47x/tei-platform/pull/3736)の全体Health #42576080511はsuccess、ただしHDG-X01〜X06はNOT RUN。次の[HDG-X02a/X03a/X05a/X05b](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)は独立Expected・未実行として#1251/#1016/#3421/#1174の各Ownerへ渡し、Product S2 Provider Bの実呼出しと独立Observation Consumerを優先する。**P0以降**：実CLI Design Context（#3262/#3354）の必須集合、標準Graphの多重Edge/出典・実Consumer（#1124/#2497）、現Command Unknown（#1174）、Trait/Provider/BOMの実資格（#3279/#3267）を接続する。E1 plan→Go HostのBinding、実TEI Graph Loader、Actual Provider/Client/Effectは未証明。既存Gate B/Cと各Ownerを維持する。

## TEI標準要件：モデル全容の図表化と、標準／カスタムの理解可能性（2026-11-08）

**基本要求（[RQ-036](../requirements/TEI_PRODUCT_REQUIREMENTS.md#rq-036-ドメインの全容を図表とナビゲーションで理解拡張できること)、利用者明示）**：TEIのすべての標準機能は、業務／システムの対象構造、イベント群、状態遷移、Function、Pattern、Data、Rule、Operation、Providerと外部連携が、**TEI自ら導出する一貫した図表体系**の中に位置づけられることを前提とする。各Patternだけを個別に説明しても標準機能として十分ではない。利用者が事前知識なしに「全体として何をするか・何が起きるか・どこを変えられるか」を理解・探索できる標準ガイド／ナビゲーションを備える。Web/Desktop/API/CLI/MCP/VUIなど媒体ごとに別の意味正本を作らず、異質なConsumerにも同じ型付き参照と適用資格を渡す。

**図表を全体のガイドとして保つ（2026-11-09、利用者明示）**：大きなFlow・状態遷移図・構成図は、個別要素の確認・編集中も全体の道案内として働く。詳細を開いても全体像の中の現在地が見え、概観の同じ位置へ戻れ、一枚で読めない図は意味上の区画で概観と内訳に分ける。生成AIには同じ概観・現在地・近傍をCLI/MCPから構造化テキストで渡す。図表は実装である以前に設計書であり、要素から実装・試験の所在と未実装の範囲へ辿れる実装のガイドとする。具体化と受入（MAP-17〜23）は[三要素Method §19.8](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)に置く。

**標準とカスタムを見分けられること**も機能要件とする。標準固定の意味・Guard・Authority/Effect境界、設定可能な既定値、明示Extension Slot/Hook、案件で置換可能な部分、観測のみの部分を、Flow・Graph・Matrix・CLI説明で同じように区別する。差し込みのある個所は**標準フロー／差分／合成後の実効フロー**を相互参照可能にし、Plugin/Generator/Targetの版と受理資格を隠さない。ドメイン別のEvent/State定義は必要に応じてカスタムできるが、型・Source・現在権限・実副作用の検査をPresentation変更として迂回しない。

**完全性の意味**：指定したWorld／Source Scope／Consumer公開資格に対し、要求される図表・関係・Pattern／Event／Stateの対象集合と、実際に解決／表示した集合、未解決・非公開・未対応の集合を区別して表示すること。必要集合自体が未確認なら「全体像はpartial」と説明し、表示したものだけで完全性を主張しない。図表・表・CLIは共通のAccepted Meaning/Sourceから導出し、手書きのガイド図やAI説明を第二正本にしない。

**実装／受入の標準接続**は[三要素Method設計§19](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)に集約する。具体的な初回受入案はMAP-01〜10（全容Catalog、Event→Lifecycle、Pattern位置づけ、固定/Custom/実効フロー、初心者の段階ナビゲーション、Scope・Coverage、複数Consumerの一致）を基本とし、**MAP-11〜16**でLifecycleを可変の型付きGraphとして組み立てる際の保証、Plugin競合、到達不能、編集中と実行中のRevision差、AIチャットのInteractionと業務Effectの分離を反証する。固定するのは**不変の保証**であり、全Domainに一律の手順列を強制しない。#1124をDiagram/Navigation、#1125をEvent/Business Flow、#1126をData、#1127/#2507をFunction/Pattern、#2497を投影意味の保持、#2619/#2620をEverything is Data/役割、#2514/#1180を独立Expectedの既存Ownerとする。最初は**read-onlyのModel Map→明示Gap→複数図表へのジャンプ**を検証し、その後にHook差分・Typed Edit・Target実効動作を扱う。現時点では標準要求を計画へ反映した段階で、全図表Generator／全媒体Navigation／全Patternの実装完了は**未確認**。既存Gate B/Cを遡及して変更しない。

<!-- tei-b2c-headless-plan-20261108 -->
## 高度B2Cの自由度とConversion Engineeringの独立検証（2026-11-08、提案段階）

高度B2Cにおいては、競争力を生む独自Frontend・SEO・体験実験・頻繁な販促更新のため、TEI標準Renderer/DOM/Framework/Runtimeを使わない完全Headlessも正規利用として認める。標準UIを使う省力化と、UIを全面的に独自開発して型付きData/Operation/Effect/Source/Scenarioだけを活用する方式を、同じContract上で比較する。B2CのためにTEIの第二Host・Resolver・Commerce/Analytics Engineを追加しない。

**検証を優先する価値**は、単なる汎用EC画面生成ではなく「施策変更→配信Variant→実表示→Intent→権威あるOperation Effect→計測・評価→採用/撤回」を既存のMeaning/Provenance/Plugin/Conformanceで安全に結べるかである。Experience IntentとBusiness Meaning、画面Rollbackと注文Effectの補償、通常の販促変更と価格/決済/同意を伴う変更を別責務とする。全体図はRQ-036の共通Model Mapへ位置づけ、固定保証と拡張Hookの実効Graphを区別する。生成AI/Harnessは提案/テスト/測定の薄いConsumerとし、現在権限や業務意味の受理を代行しない。

**順序**：B0 read-only Fit/Gap → **B1a** 独自Frontend＋既存の型付きRead/Operationの一往復（標準Rendererなし） → **B1b** Commerce SandboxのCart/Checkout・実Effect/独立Read（B1aと別の資格） → B2 部分標準UI/Experiment/旧Client/Consent/配信Cache・Browser回帰 → B3 AI性能・変更時間・計測品質の同条件比較 → B4 ECと予約/Subscriptionの異質Consumer比較を経て外部Adapterを製品化判断。#2586の実Function→Query→Provider→Data Contextと#4350の表現・写像Fit/Gap、既存#1119 E2や#951/#1251の実Provider検証を優先し、新規P0 Gateにしない。

**資格・正本**：[#3509](https://github.com/hat47x/tei-platform/issues/3509)が現行B2C受入Owner。旧#3512の設計本文は[保存コメント](https://github.com/hat47x/tei-platform/issues/3509#issuecomment-6772845918)へ全文移管され、#3529も横断計画への引渡し後に未マージでクローズされた。11月12日の敵対的検証・BC-25〜34はIssue本文で管理し、既存Scenario/Independent Expectedを再利用する。**実Native Browser、Commerce Provider、実Effect/別Read、Field Experiment、CVR改善、Owner受理はNOT RUN/NOT ACCEPTED**。


**2026-11-12の敵対的レビューで強化した契約**： (1) Headlessを「専用SDK/常駐JS必須」にすり替えず、Mandatory TEI Runtime/DOM/Router依存0のNative経路を実Artifactで確認する（TEI利用の総コスト0を意味しない）、(2) 変更レーンは文言/CSS等のファイル分類ではなく、実効のRule/Consent/Price/Effect/Disclosure依存で判定する、(3) 生成AIのWeb/Studio操作対象はFocus/DOM座標でなくSource Subject・Scope・Selection epoch・Revisionに固定し、古い選択からの書込みを拒否する、(4) Experiment assignment、実Exposure、Operation Effect、業務成果・因果推論を独立観測し、同意・欠測を「非購入」に変換しない、(5) Screenshot/実験ログ等のEvidenceは[#4307](https://github.com/hat47x/tei-platform/issues/4307)のlocal/NAS/S3とGitリンクの分離を使い、GitHubへの実体・存在情報の送出を必須にしない。[#4198](https://github.com/hat47x/tei-platform/issues/4198)のLifecycle Graph、[#3278](https://github.com/hat47x/tei-platform/issues/3278)のTrait/Hook、[#997](https://github.com/hat47x/tei-platform/issues/997)のUI Kitは各Ownerのまま。詳細と独立Expected BC-25〜34は[#3509](https://github.com/hat47x/tei-platform/issues/3509)に集約する。B2C独自の第二Workflow/Graph Compiler/Analytics Engineを追加しない。

## 基本開発方針（2026-11-01、Owner方針）

**外部要件による十分な机上評価 → 基盤構造の実装 → CUIと標準プラグインの並行開発・一気通貫検証 → 人間向けGUIの構築**を、新規機能開発の基本順序とします。これは過去のGate A/B/Cの判定を変更するものでも、独立した新しいGateを設けるものでもありません。実装済み・検証済みの範囲は[現行の実装状況](../docs/status.md)、個別の進行・受入は各Issueと既存Gateで確認します。

机上評価の観点、複合要件の反例、CUI・標準Pluginの一気通貫検証の具体案は[外部要件による基盤評価とAI-firstヘッドレス検証](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)に整理します（設計候補・未実行）。

基本データ構造の机上評価は、通常のプログラミング言語の型階層より上位にある**構造形式・数学的構造・数量／単位・トポロジー**の概念空間から検討します。具体的な構造には複数の親や適合契約を認めつつ、分類する問い・関係の意味・構成と実現方式を区別します。候補と未決点は[基本データ概念空間の横断設計](design/FOUNDATIONAL_DATA_CONCEPT_SPACE_2026-11.md)で管理します。これはCoreの新型や多重継承実装を確定するものではありません。

特に、同じ値集合が**異なる演算の束縛によって同種の数学的構造を複数持つ**場合（例：自然数の加法・乗法モノイド）、Facet名や多重親の列挙だけで演算・公理・適合を決定しません。数学的Theoryの要件、具体的OperatorへのBinding、成立するLaw、解釈Scope、根拠の種類を区別し、[多重Facetの整合と公理の検証 #2544](https://github.com/hat47x/tei-platform/issues/2544)に具体的反例を管理します。広域の構造探索は[#2501 Corpus](https://github.com/hat47x/tei-platform/issues/2501)、構造保存・編集逆写像は[#2497](https://github.com/hat47x/tei-platform/issues/2497)、分類そのものは[#2495](https://github.com/hat47x/tei-platform/issues/2495)を利用します。現段階はいずれも未採用の検証計画です。

**構造のMECE性・複数Projectionの整合**については、固定のData Type列挙より、名目ID／構成子／Facet／演算束縛／制約／解釈文脈／出典を区別し、各判断の適用範囲と具体的反例を残します。比較のための正規形が計算できても、それを業務意味の同一性・Owner受理へ昇格させません。異なる構造が同じ意味を投影する場合（例：[TEIの多値Record投影 #2576](https://github.com/hat47x/tei-platform/pull/2576)と[SUIの所属表現](https://github.com/hat47x/tai-pleroma/pull/253)）は、**どのRelation/Sourceを正本とするか、列挙順・重複・完全性・Version・更新Intentをどう保存するか**を別に判断します。既存[#2495](https://github.com/hat47x/tei-platform/issues/2495)/[#2544](https://github.com/hat47x/tei-platform/issues/2544)/[#2497](https://github.com/hat47x/tei-platform/issues/2497)と[Draft PR #2496](https://github.com/hat47x/tei-platform/pull/2496)が比較・反例の担当です。実Targetで意味が保存されたことや両表現の意味上の等価性は未実証です。

**Projectionの値忠実度と対象範囲**も、基本データ構造の机上評価からCUI/標準Plugin・実Target検証へ引き渡す項目です。Fieldの値型、必須/任意と未設定・明示NULL・空、単一/多値、重複・順序・完全集合性、数値精度・単位、Source/Consumer、編集時の変更Intentを区別し、特定の用途で許容する情報損失を明示します。既に実装された[Record Projectionの多値・数値・optional・boolean（#2576/#2578/#2580/#2581）](https://github.com/hat47x/tei-platform/pull/2578)を[#2585](https://github.com/hat47x/tei-platform/issues/2585)の局所Conformanceで検証し、[#2497](https://github.com/hat47x/tei-platform/issues/2497)の構造保存条件と[#2498](https://github.com/hat47x/tei-platform/issues/2498)の実Consumer適合へ接続します。SUIの読み取り専用artifactと既存SUI元データのような**部分Projection／複数Source合成**では、フィールドごとの正本・未投影項目・時点・版を説明できるかを確認します。厳密な金額計算の型契約は#1705、業務DBのmissing/nullは#2056の成果を再利用し、GUIやProviderの正式な意味適合は別途実証します。現在の追加項目は受入設計であり、実装・実証完了を宣言するものではありません。

**時間変化するProjectionの意味保存**は、静的なValue/Presence適合の次の検証対象です。同じQuery/Functionから生成するCUI・Studio・SHOの複数Consumerについて、Source意味版、Query/Result SliceのScope/Completeness、Snapshot/read-set、Event/Processing/Observation Time、Selection epoch、実Consumer読込版を分け、Claimの支持・暫定・撤回・置換・失効を説明できるかを確認します。先頭ページの0件を全体不存在へ昇格させないこと、JOINの重複・LEFT JOIN仮NULL・TOP-Nの変更で旧結論を適切に撤回すること、古い非同期応答を新しいContextへ混ぜないことを反例とします。[#2497の§12](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)と[#2501のPC-97〜108](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/DATA_STRUCTURE_PRESSURE_CORPUS_2026-11.md)に整理し、単一FunctionのHeadless実証は[#2586](https://github.com/hat47x/tei-platform/issues/2586)、Data Contextは#2506、Freshnessは#1708、版間対応は#2505、異質Consumerは#2498/#2516の既存Ownerに帰属させます。追加12件は反証候補であり、実試験PASSや業務意味受理ではありません。

**Measureの再集計適合とClaimの開示適合**を、値忠実度・時間変化するProjectionに続く検証軸として追加します。同じDecimal型・同じ表示値でも、平均・比率・重複排除人数・時点残高・分位数は集計軸と必要な中間状態が異なります。[#2593](https://github.com/hat47x/tei-platform/issues/2593)のMeasure再集計を[#2497の§13](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)に接続し、[#2501のPC-109〜120](https://github.com/hat47x/tei-platform/issues/2501)を独立反例として整理します。また、内部で正しく導出したClaim・根拠へのアクセス資格・外部Consumerへの開示資格・開示後の根拠充足・実際のConsumer読込を別々に検証します。Construction Context/AI CUI/SHOは[#2516](https://github.com/hat47x/tei-platform/issues/2516)、外部開示は#1178、SHO原本は#1803、実Consumerは#2498の既存Ownerに帰属させ、第二のMeasure/Authority/証拠Registryを作らず、受入・実送信・実Targetの観測を独立管理します。今回の反例は机上計画であり、実装/Owner受理は未実施です。

**Consumer間Claim比較・文書Occurrence・開示非干渉性**を、構造保存とConstruction Contextの追加検証軸として位置づけます。同じ表示値でも対象・演算・Grain・Scope・Snapshot・Source版が異なれば比較不能であり、異値でも部分取得と全件確定のように両方正当な場合があります。比較可能な同一Claimの不一致だけを独立検証基準へ返します。また、同じTEI意味要素がSHO文書に複数出現するとき、Semantic IDと文書内Occurrence ID、局所編集・再生成・レビュー現在性を区別します。内部で読める根拠が外部Consumerへ開示できるとは限らず、非公開情報の有無が比較結果・拒否理由・Traceから漏れないかを既存開示Ownerで検証します。[#2497の§14](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)と[#2501のPC-133〜144](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/DATA_STRUCTURE_PRESSURE_CORPUS_2026-11.md)へ整理し、実Consumer横断は#2516 W8、SHO文書はSHO #4/#6、数値忠実度は#2585、情報開示は#1178へ帰属させます。追加反例は設計段階で、実SHO/OS/AI/GUI試験やOwner受理は未実施です。

**Functionの依存閉包・物理Read・開示・Effect対象束縛**は別々の受入義務として扱います。表示用Data Mappingだけでは非表示Validation/Rule入力を閉じられず、同じ公開JSONでも実ProviderのOverfetch有無は判定できません。Rowset completeとField observed、宣言したallと実際の取得完了、表示対象と実Request/Effect対象、公開回答と秘密Action候補の開示可否を区別します。[#2497の§17](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)へDC-01〜08を整理し、既存Corpus PC-157〜168と重複起票しません。Rule/Mappingは#2624、Provider実Readは#2628/#2622、Contextは#2506、View→実Requestは#2598、外部開示は#2516、Authoringは#2621のOwnerを維持します。隔離合成試験のPASSを実Provider/Host/OS/AIのE2Eや業務意味受理へ昇格させません。

**SQL→Data Operatorの意味保存と複数Function編集の合成不変条件**を、別々の実証義務として追加します。SQLのJOIN/EXISTS、NOT IN/NOT EXISTS、LEFT JOIN、NULL三値論理、Bag多重度、DISTINCT、Filter/Sort/LIMIT順序、平均の再構成は、構文や表示値が一致しても一般に意味同等とは限りません。SQL正本とTarget条件を保持し、候補Data Operatorを独立検証基準・実Providerで検証します。また、複数Functionの個別保存成功は並行実行後の業務不変条件を保証しません。Editing Session、Read Predicate、実Transaction境界、Commit不明、Effect Lookup、独立read-backを区別します。[#2497 §18](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)と[#2501 PC-169〜180](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/DATA_STRUCTURE_PRESSURE_CORPUS_2026-11.md)へ反例を配置し、SQLは#2387/#2622、独立期待は#2514、編集は#2597、実Effectは#1174の既存Ownerへ戻します。隔離SQLite試験のPASSは実TEI/PG/Host/Owner受理を示しません。第二Query Runtime・共通Transaction Coordinator・TEI独自Hostは導入しません。

**空結果の否定依存・選択的再資格化・Product Skillの実利用資格**を分離して検証します。0件のQueryにもWHERE/NOT EXISTS/Relation/Scopeの依存があり、0→1→0の同値観測から途中の変更なしを推定できません。変更影響と現在のConsumer利用資格を別扱いとし、依存証拠のあるClaimのみ選択的再評価を候補とします。さらにSemantic Role、Operation、CLI、Product Skill読込、実Tool呼出し、外部Consumerへの開示、Host Effectを段階別に観測し、Skill説明やCLI存在を実行許可・完了へ昇格させません。[#2497 §19](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)、[#2501 PC-181〜192](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/DATA_STRUCTURE_PRESSURE_CORPUS_2026-11.md)へ反例を追加。Change/Queryは#2582/#2387、Consumer資格は#2516、再利用は#2018、Product Skillは#2605/#2619とTAI #271、実Effectは#1174のOwnerを維持します。実Provider/Harness/SHO/Host統合試験は未実施で、新しいNegative Cache/Reactive Engine/独自Hostは設けません。

**Artifact Envelopeの同一性・物理公開一式・実Consumer読取は三つの別資格**として受入を計画します。既存`Artifact.Digest`はContent raw hashではなくKind/InputRevision/Rule/Generator/Content/UnresolvedのEnvelope fingerprintであり、現行`tei artifacts -out`の個別保存byteとManifestだけではEnvelope検証入力を完全復元できません。Content同一でRule/Generator/Unresolvedが変化する負例、Manifest取得中の世代切替、ExpectedMemberの欠落、SHO/Skillの旧版実消費を[#2497 §20](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)のAE-01〜08へ整理しました。Artifact整合は#2482/#2630、公開一式は#2604、SHO/Consumer読取は#2516、Skill消費は#2605へ帰属します。#2633 main統合済みCharacterizationと#2659/#2607 Draftを区別し、別HEADのCI成功を現HEADの証拠にしません。実外部Consumer・障害注入・Owner受理は未実施で、第二Artifact正本や独自Hostは追加しません。

**Data Model正本とQuery二経路、生成アクセス性能の受入**：Table/Field/Key/Relationは直接宣言するData Modelを正本とし、SQL解析由来は候補に留めます。QueryはTEI QueryとNative SQLを第一級の別Authoring経路とし、AI変換は元SQLを保持した意味比較候補です。通常の取得はFunction Data ContextにTarget/Relation path/Field/Predicate/Scopeを明示し、複雑集計・方言固有処理ではNative SQLを維持します。生成アクセスのN+1、JOIN fanout、Read Scope、Query count、p50/p95を実Providerで測定し、静的に証明できる契約の重複ランタイム検証は前提・失効条件付きで省略候補とします。ただし実行時Grant、外部Input、Provider部分取得、Transaction/Effectの境界検証は省略しません。[#2497 §21](https://github.com/hat47x/tei-platform/blob/docs/data-structure-meta-facets-20261101/plan/design/STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)のDM-01〜10は設計票でNOT RUN。既存#1126/#1129/#2387/#2622/#2624/#2628/#2018/#1174へ帰属し、第二Data Model正本・新SQL Runtime・TEI独自Hostを設けません。

### 外部要件を先に受け止める（要求・構造の机上評価）

CUIやGUIの実装に入る前に、**データ構造、意味上の識別・型・値域・関係・制約、操作・状態・副作用、プラグインの契約・合成・ライフサイクル、Target/Hostとの境界**など、後戻りの費用が大きい基本構造を十分に検討します。

成熟した参照設計が示す豊かなデータ・機能・業務モデルと、一般的な業務製品・サービス・オープンソース実装等の多様な外部要件を、単なる機能名称の一覧ではなく、具体的な利用シナリオ・制約・組合せ・失敗条件として収集します。一製品の実装方式をそのままTEIの中核に固定せず、外部要件をTEIの共通概念、拡張可能な意味モデル、標準／固有プラグイン、外部実装のどこで受け止めるかを比較します。

対象となる外部要件集合について、**全件の対応先と残余を追跡可能にする**机上適合表を用意します。最低限、要件と出典、期待する振る舞い、TEIでの表現・実現経路、複合条件、失われる意味、未解決点、反例・検証方法、判断状態を対応付けます。「全要件を受け止められる」という判断は、選定・確認した集合と証拠範囲に限ります。資料の欠落や未知の製品仕様まで完全性を推測しません。

基本構造に関わる重要な未解決事項は、契約の固定や汎用化の判断より先に検討します。拡張で表現できること、実行できること、互換性を実証できたことは別々に記録します。既存の[参照モデル移植時の不足分類](design/REFERENCE_PORT_IMPLEMENTABILITY_GAP_TRIAGE_2026-10.md)、[技術検証用サンプル](design/TECH_VALIDATION_SAMPLES_2026-10.md)、[設計原則](../product/principles/TEI_PRODUCT_PRINCIPLES.md)を再利用し、要件や判定の第二正本を作りません。外部資料の権利・公開範囲も既存の規則を維持します。

### 基盤を先行実装する

机上評価で得た構造上の判断を踏まえ、Canonicalな意味、Source/ID/Trace、型・関係・制約、プラグインの発見・契約・合成・Binding、検証・生成・実行との接続など、**各種製品機能が共有する最小限の基盤**を優先します。個別製品の機能に合わせた中核要素の追加や、X-TEA固有の技術構造への固定は避けます。

机上評価は設計上の適合可能性の確認です。実装の成立、複合要件、性能、実Provider・実Hostでの振る舞いは、後続の実験で独立に確かめます。基盤全体の完成を待つのでなく、標準プラグインとCUIに接続できる最小の縦切りから実装を進めます。

### 生成AI向けCUIと標準プラグインを並行開発する

生成AIが通常のコーディングと同等以下の探索・判断負担で実装できることを、操作系の強い要件とします。**CLIを基本の開発入口**とし、通常のファイル／DSL／SQL直接参照・編集、Git、必要に応じたDB直接参照・独立検証を第一級の経路として維持します。生成AIが機能・定義位置・実装方法・必要な検証を自力で発見できる、版付きの機械可読な診断・操作契約を整備します。開発者向けMCPは必須とせず、利用者向けMCPやComputer Useは目的に応じたAdapterとして評価します。

基盤の最小接続部が成立した段階から、**CUIと標準プラグインを並行して開発**します。プラグインの契約だけを先に完成扱いにせず、CLIから定義・選択・合成・実行・観測まで接続し、標準機能の実用性と基盤の抽象化の妥当性を相互に検証します。標準プラグインが中核の特権的な別経路にならないことも確かめます。

### ヘッドレス一気通貫を基準にする

各縦切りでは、**要件／Source → モデルの作成・変更 → Validation → Plugin/Binding解決 → 生成または意味シミュレーション → 実行 → 独立した結果観測**を、CUIで追える形にします。正例だけでなく、不適合、権限拒否、未対応、競合、部分成功、結果不明などの負例を対象に含めます。

同じrevision・fixture・Contract・生成物・実行条件を追跡し、机上の表現可能性、合成試験のPASS、実DB効果、実Hostでの成立、Ownerの意味受理を混同しません。既存のScenario、Driver、Source Index、Evidence、Gateを再利用します。生成AIが試作・修正・検証を自律反復できるようにしながら、正式な意味受理、権限付与、本番配備などの判断は[AGENTS.md](../AGENTS.md)の責任境界に従います。

### 人間向けGUIはヘッドレスの実証後に構築する

Web／Desktopの編集画面や利用者向けGUIは、対応する機能的な振る舞いがCUIで再現・検証できる基盤を基準として追加します。GUIは同じ型付きOperation・意味・検証経路を利用し、表示・入力・対話・アクセシビリティ等の**GUI固有の差分**を独立して確認します。GUIで初めて現れる要件は、必要に応じて基盤へ還元して検証します。過去に実施したGUIの参照試験や並行する調査は保持しますが、新たな基盤機能の完了条件としてGUI実装を先行させません。
- `plan/design/AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md` ：生成AIによる性能改善のための設計ヒント、低負荷な統計収集、Source/Function/Query/Providerの観測接続、AI提案の独立Benchmark・適用・Rollbackを扱う**横断設計・実証候補**（[#3271](https://github.com/hat47x/tei-platform/issues/3271)、[#3262](https://github.com/hat47x/tei-platform/issues/3262)、[#2582](https://github.com/hat47x/tei-platform/issues/2582)）。新Optimizer/Core/Hard Gateや性能改善の実証完了を意味しない。
- `plan/design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md` ：原子的Traitの実装義務、Manual/Generated/Providerでの履行、合成・不適合診断を既存#951/#2495/#3271/#3267へ接合する**設計・実証候補**（[#3278](https://github.com/hat47x/tei-platform/issues/3278)）。新Resolver/新Hard Gateや実装完了を意味しない。
- [Issue #3503](https://github.com/hat47x/tei-platform/issues/3503#issuecomment-6772886371) ：Lifecycle Graphの型付きRole・不変Obligation・Plugin Patch競合/合成・Generator入力・L0〜L6縦断検証・LG-01〜16独立Expectedを、既存#3503/#1158/#951/#1121/#1180へ引渡すDraft。#3509のB2C Headless、AI Web Chat/VUIも異質Targetとして比較。既存P0を置換しない。
- [Issue #3503](https://github.com/hat47x/tei-platform/issues/3503#issuecomment-6772855707) ：API/Web/Desktop/CLI/MCP/VUI横断の共通Operation、型付きLifecycle Graph/Hook Binding、状態付きInteraction/AI Web Chat、実行Event/Effect意味論を、既存#1158/#2090/#951/#1121/#1180へ接続する**設計・実証候補**（#3503、Draft PR #3507）。標準/拡張/実効/観測の区別と、第二Host/Registry/新Gate禁止を維持する。
- [Issue #4198](https://github.com/hat47x/tei-platform/issues/4198) ：Statecharts/SCXML、Process Variants、Graph Transformation、Temporal、AG-UI/A2UI等から**Lifecycle Graph自身をPluginで拡張してGeneratorへ渡す**設計を検証する研究。図表編集からSource/Buildへの帰還、AI Web Chat/VUIの異質性を扱うが、正式Graph Compiler/実Target/Owner採用は未完了。
- `plan/design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md` ：Web/Desktop APIの例外・Client Reaction・Graph/Table/Traceの**横断設計と段階的な実証計画**（[#3274](https://github.com/hat47x/tei-platform/issues/3274)）。**P-1a/b**：E1の構成時Guard DAGとRuntime HTTP Decision Pathを区別し、[実Node専用CI #42396614454](https://github.com/hat47x/tei-platform/actions/runs/42396614454)の限定HDG-21 CharacterizationはPASS。全体Healthは同PRでgofmt不一致（別Source）によりFAIL、Go vet/test/Workspace validateはPASS。**P-1c**：既存Go Reference Hostのdefer Auditと10経路Callback/Status/Stageを、[Node+Go専用CI #42461546535](https://github.com/hat47x/tei-platform/actions/runs/42461546535)で限定PASS。**P0-R（限定実証）**：E1・Goの独立宣言は両側が局所的に構成可能でも不一致になることを[Node/Go実CI #42462531490](https://github.com/hat47x/tei-platform/actions/runs/42462531490)で確認（Node 7/7＋Go package全PASS）。**P0-S（限定CI合格）**：実`reference/workspace`の`application-pattern:order_entry.submit_order`と`semantic-capability:sales.order.submit`・Source Index/Canonical参照を既存`operations.InspectSubject`で検査し、[Node/Go専用CI #42464774706](https://github.com/hat47x/tei-platform/actions/runs/42464774706)＋同HEAD全体Health #42464774715でPASS。**P0-B（限定CI合格）**：既存`driver.ResolveCommandBinding`を実Workspaceで通し、欠落・重複・有効だが意味的に違うCapabilityの負例を[専用CI #42470312187](https://github.com/hat47x/tei-platform/actions/runs/42470312187)でPASS（同HEAD全体HealthもPASS）。**P0-I（実Host→Driver→合成AdapterのInvocation検証候補）**：実`NewHost`・`NewCanonicalActionAuthorizer`・実Driver`InvokeCapability`から合成`Capability Adapter.Invoke`までを一つのHTTP Attemptで通し、Session/CSRF/Canonical認可拒否・Adapter解決障害・実Invoke障害・Canonical Result契約違反の独立Expectedを[設計§18](design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)で試験する。実Adapter呼出しが確認できても本番Plugin Provider/Artifact/Grant・永続DB Effect/Client完全受領・Graph Required全閉包ではない。**P0以降**：実CLI Design Context（#3262/#3354）の必須集合、標準Graphの多重Edge/出典・実Consumer（#1124/#2497）、現Command Unknown（#1174）、Trait/Provider/BOMの実資格（#3279/#3267）を接続する。E1 plan→Go HostのBinding、実TEI Graph Loader、Actual Provider/Client/Effectは未証明。既存Gate B/Cと各Ownerを維持する。

## 現在の主線

第1版の参照アプリケーション`order_entry`に基づく判定段階と既存の検証レーンは次のとおりです。**新規開発の基本順序は上記のAI-first・要件先行方針**に従います。

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

### Studio Desktopセルフビルドを最優先にする現行の実証順（2026-11-14更新）


**Desktop Workbench・実行経路の横断統合（2026-11-16、設計・受入引渡し）**：[Desktop Web標準UI実装指針 §§14〜17](design/TEI_DESKTOP_WEB_STANDARD_UI_IMPLEMENTATION_2026-11.md)で、一つのSource-backed対象をEditor／表／Graph／Node／Process／Inspectorから参照する画面構成、Focus／Pinの非干渉、Buffer／実Source／Meaning／Build入力／実Consumerの版分離、保存時の実byte照合と復旧、CLIとWails Bridgeに共通なApplication Operation、標準型を使うPlugin接続を一本化する。第二Parser、意味正本、Plugin Loader、Flow Runtime、Desktop独自Host／Policy Engineは増やさない。

**次の実装と優先順位**：製品P0は引き続き (1) 非Plugin Studio BuildIntent→実選択Builder→N0／N2／N3同一Attempt→生成Stage1自身によるStage2 Build→そのStage2のWindows WebView2実受領（#2831／#4583／#1824）、(2) 通常CLI→受理済みpure Function→actual Provider Read→独立Consumer（#2586／#4495）の二本。UI側は#4515の後着／Pin、#1529／#1124の一Source Buffer→保存時競合→再Inspect、#1158のCLI／Wails read-only同値、#2497／#951の型付きPort／Adapter表示を並列の小PRとする。新§16〜17は**docs-only／未実行**であり、先行するPinや生成PoCの限定成功を正式Stage2／実Provider／Owner受理へ合算しない。

**Desktop読取から編集・生成へ進む現行実装順（2026-11-16）**：[Desktop Web標準UI指針 §19](design/TEI_DESKTOP_WEB_STANDARD_UI_IMPLEMENTATION_2026-11.md)に、実mainの三入口Inspect（CLI別Process／共通Operation／Wails非依存Studio Host）、PR #5132でmain統合済みの生成関数名変更・誤Attribute読取検査、L2-bのBuilder実ファイル照合／任意のGo Toolchain照合を、**独立した現在の限定成果**として反映した。次のDesktop実装は同じqualified SubjectをInspector→Source occurrence→Editorの実Caretへread-onlyで渡す一本に絞る。続いてPin後着とDirty Buffer非干渉、既存WriterのSource byte CAS→再Load→Meaning Review、型付きWhole／Field／Adapter配線へ進む。現行Reference HTMLと配布HTMLのPin操作は一致するが、元Sourceを開く操作・実保存・Windows Stage2の合格を推測しない。P0の非Plugin Studio Stage1→Stage2（#2831／#4583）と通常CLI実Provider（#2586／#4495）は並列のまま。新Host／Runner／Trace／UI意味正本なし、今回の追補は**docs-only、実E2E NOT RUN**。

**Desktopの抽象経路を検査する（2026-11-16、設計・試験引渡し）**：[Desktop Web標準UI指針 §18](design/TEI_DESKTOP_WEB_STANDARD_UI_IMPLEMENTATION_2026-11.md)で、表示名・生成メソッド名・DOM IDから離れ、qualified Subject／Source occurrence／Role・型付き義務・実装選択・実Invoker・実Consumerを既存Evidenceから追う観点を定める。最初は実Rule MarkdownのCLI Inspect／共通Operation／Wails非依存Studio Hostの同値と、異Kind／許可外Workspace／Source改版の拒否を#5108・#1180・#2514で確認する。続いてWails実UIのFocus/Pin・原文への往復を検収し、Generate／Interpret／Delegateの実現方式比較（#2620／#1121）は別の実使用証拠とする。#5115の抽象経路試験、#5101の型付きMappingは参考にするが、既存Ownerの結果をDesktop製品E2Eへ昇格させない。P0のStage1→Stage2と通常CLI実Providerを優先したまま進める。**本追補はdocs-only／新実装・実機試験NOT RUN**。

**Desktop統合仕様の引継ぎ（2026-11-14）**：[Desktop Host設計 §§11〜23](design/DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md)に、Wails/CLI、SourceとMeaning、Graph、Dirty Buffer、起動・復旧、生成Inspectorの変異試験、CLI先行の実質的セルフビルド、設計対象のAIへの引渡しを集約する。個別PRの古いHEADやCI成功は最新の製品合格へ加算しない。§14〜19のケースは#1180/#2514への引渡し候補であり、第二Scenario正本ではない。

**DesktopのWeb標準UI開発（2026-11-15追補）**：[実装・検収指針](design/TEI_DESKTOP_WEB_STANDARD_UI_IMPLEMENTATION_2026-11.md)で、現行TypeScript＋esbuild＋CodeMirrorを基線とし、S-LOCのread-only部品だけをVanilla Web Components／Lit＋TypeScript／Lit＋JSDocで比較する。Design Tokens・CSS Layer・Grid/Flex・Shadow DOMの選択、TypeScript/Lint/DOM/IME/ARIA・正規Bundleと実WebView2の検査を一つのFixtureで評価し、優位が確認できる範囲で採用する。現行配布HTMLにはPin UIが存在する。現在の静的HTML一致と実WebView2での検収、さらにStage2再入を分け、Framework選定やCSS刷新を#2831/#4583のStage1→Stage2 P0と#2586/#4495の実Provider P0に追加しない。

**Web Targetと宣言的UIの実装引渡し（2026-11-16更新）**：[Web標準UI横断設計 §§16〜19](design/WEB_STANDARD_COMPONENT_AND_AGENT_HARNESS_2026-11.md)に、静的公開／対話型Web／Server側HTML生成／Wails WebViewをTargetの配備・実現方式として位置付け、Meaning／Operation／UI Projection／Layout Token／Host Adapterの責務を整理する。現行mainの`inspector-selection.ts`→`inspectEligibility`→`responses.run`→`readInspection`→表示の実経路を基線とし、**WEB-P1a**は検証済みResultからread-only ViewModelを純粋に導出、**WEB-P1b**はFocus／Pinと後着応答の非干渉、**WEB-P2**はCLI direct／Studio Host／read-only HTTP公開、**WEB-P3**は公開Facet→Head Slot→Asset Member→独立Consumerをそれぞれ小さく検証する。`source-locations.ts`のSource位置読取は画面未接続であり、表示成功と原文の再読取・書戻しを混同しない。新UI Parser／Host／Registry／State Storeは追加せず、Lit・SSR・PWA・Mobileを先行採用しない。これは設計・実装引渡しであり、新しいWeb／Windows／HTTP E2EのPASSではない。独立Expectedは#1180／#2514、実装Ownerは#997／#1824／#1158／#951／#3509へ返し、二本の製品P0（非Plugin Stage1→Stage2実Windows、通常CLI実Provider Read）にHard Gateを追加しない。

**Desktop配布Sourceの回復と製品引渡し（2026-11-15）**：[Web標準UI指針 §§8〜11](design/TEI_DESKTOP_WEB_STANDARD_UI_IMPLEMENTATION_2026-11.md)に、現行Embed CIで照合するHTML／JS／CSS／第三者通知の四ファイル、Go埋込→実Windows WebView2→Stage1→Stage2の独立した受領、内蔵Rule Readerの限定成立、Pin／S-LOCとAI文脈の権限境界、D1現行・D2単一Image＋Updater・D3子Backendの導入判断をまとめた。次に閉じる実コード差分はW0の現行Source→正規Bundleと実Pin DOMの受領であり、Lit導入・Mobile・子Backend・S-OPENをP0の新Hard Gateにしない。既存#1824/#2831/#4583/#1180/#2514の受入を利用する（設計追補、当該変更の製品E2EはNOT RUN）。

**Desktop W0-A/Bの実装・検収済み範囲（2026-11-15）**：[Desktop実装指針 §12](design/TEI_DESKTOP_WEB_STANDARD_UI_IMPLEMENTATION_2026-11.md)を現行実績に更新。Pinの旧Controller／重複Handler／DOM ID・版表示の欠陥は[#4920](https://github.com/hat47x/tei-platform/pull/4920)で解消し、同一HEADのPin CI成功を確認。正規Node 24.21.0ビルドからWails埋込HTML/JS/CSSを更新した[#4945](https://github.com/hat47x/tei-platform/pull/4945)は、NOTICEを含む四ファイルのbyte一致／非配布dependencies.jsonをEmbed CIで検収済み。W0-C実Windows WebView2のDOM/Host受領とW0-D非Plugin Stage1→Stage2は引き続き独立P0であり、成功としていない。共通規約は`AGENTS.md`／横断Web標準UI設計を参照し、Desktop独自のHost・Framework・P0 Gateを増やさない。内蔵Rule Reader／Inspector／Evaluatorに加え、#4911のGo判断生成はR1の局所到達点だが、正式Builder・実Provider通常CLI・Desktop Consumerは未受理。ノード／フローUIは標準構造とProcess FlowからのProjectionとしてP1で扱う。

**Desktop W0-B再検収から実W0-Cへの接続（2026-11-15）**：#4949のInspector選択イベント追加で旧配布JSのByte一致が失効したため、[#4983](https://github.com/hat47x/tei-platform/pull/4983)で正規Node24 Bundleを再同期（Embed CI run 43701028956 success）。[#4955](https://github.com/hat47x/tei-platform/pull/4955)の通常Windows WebView2 job成功に続き、[#4995](https://github.com/hat47x/tei-platform/pull/4995)の同一HEAD [run 43703879023](https://github.com/hat47x/tei-platform/actions/runs/43703879023)では、**実Hostの版付きPin Aを固定したままGraphでBを閲覧し、旧観測の明示失効・A再資格化・解除後もB Editorを維持**する通常Desktop jobがsuccess。生成Desktop jobは固定Snapshot不一致でfailureのまま。今回の通常OS Consumer証拠をStage2自己再構築・実Provider・AI開示やDirty/Graph/ARIA完全不変に転用しない。次は#4515/#1824の後着/Root切替と#2831/#4583の実Stage2、#4495の通常CLI実Providerを並行する。詳細は[Desktop実装指針 §12](design/TEI_DESKTOP_WEB_STANDARD_UI_IMPLEMENTATION_2026-11.md)。

**W0-C実Windowsの後続検収（2026-11-15）**：[PR #5012](https://github.com/hat47x/tei-platform/pull/5012)（[run 43706400375](https://github.com/hat47x/tei-platform/actions/runs/43706400375)、通常Desktop job success）で、Pin A裏再確認・解除中も閲覧BのCaret・Focus・Graph SVG/ARIA・Editor・診断の不変を実測した。[PR #5025](https://github.com/hat47x/tei-platform/pull/5025)（[run 43706898173](https://github.com/hat47x/tei-platform/actions/runs/43706898173)、通常Desktop job success）では同Workspace再読込で古いPin読取版が自動復活せず、利用者の明示再確認後にだけ実Host Readerで再資格化されることを確認。**別Workspace A→B→A・意図的な旧Reply後着・編集可能Dirty Buffer・生成Desktop/Stage2は未受理**。Ruleから生成Goを通常`tei self-build`でBinaryへつなぐ[#5009](https://github.com/hat47x/tei-platform/pull/5009)もmain統合済みだが、非Plugin Studio自身のStage1→Stage2や正式Provider Entryとは別資格。詳細は[Desktop実装指針 §12](design/TEI_DESKTOP_WEB_STANDARD_UI_IMPLEMENTATION_2026-11.md)を正本とし、二重Host/Runnerや新しいP0ゲートを作らず既存Ownerへ引き渡す。

**W0-Cの生成PoC Bを現行Sourceで実機受領（2026-11-15）**：[PR #5042](https://github.com/hat47x/tei-platform/pull/5042) は、旧Workbench版のPinを残し、現行版とソース／依存Lock計11ファイルを別候補で固定して、初期の`workbench_revision_mismatch`を解消した。exact HEAD `00b92d3b6685834cab3a90595e449553e80d0d15`に対する[Windows CI run 43722212784](https://github.com/hat47x/tei-platform/actions/runs/43722212784)では**通常Desktop・生成Desktopの両jobがsuccess**。生成PoC BのUI/Go静的合成、`studio_generated` Go build、実WebView2配信JSのSHA-256、実Host/Pin/Keyboard/XEADと正常終了を同一jobで確認した。旧失敗runは過去版の履歴として保持し、**TEI自身がnon-Plugin Studio Build SubjectからStage2を生成したというセルフビルドP0の合格には算入しない**。設計・次の負例は[Desktop実装指針 §12](design/TEI_DESKTOP_WEB_STANDARD_UI_IMPLEMENTATION_2026-11.md)へ一本化。

**Graph/Rule→実Go生成の限定参照進捗（§23.5）**：mainに追加された`reference/tei-md-graph-reader`のParseRuleCandidate／GenerateGoは、固定六入力のMarkdown表からRule条件・Graph ID・Source byte位置を読み、Goの実行可能なNodeEdgeIDs関数を生成する。独立Expectedと別Go Processでの実行を照合するテストがあり、Edge反転／Rule条件変更では振る舞いが変わり、Mermaid配置だけの変更ではコードが変わらない。これは§18の「図表＋Ruleが実コードの振る舞いを担う」参照証拠に位置付ける。ただし正式`*.tei.md` Reader／Role／Workspace Source Index／製品Builder／実CLI／Studio Pinとの接続は未成立で、Graph表示IDを正式Relation IDへ昇格させない。#988/#2621へ正式Reader、#4544/#2831へ生成器・Builder・実Consumerの接続を分けて引き渡し、GUI Pinや新RuntimeをセルフビルドP0の追加前提にしない。

**非Plugin Studio Buildの型付き対象とStage2実再入（§24、2026-11-15優先判断）**：[Desktop統合設計 §24](design/DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md)に、現コードで確認した製品P0の直接Gapを固定する。`internal/buildintent/intent.go`は`go-command`／`go-package`を判別できるが、`BuildIntent.Validate()`が`Subject.Kind=plugin`のみを受理し、`resolve.go`も`plugin.BuildPlan.Plugin`と`PluginDefinitionFingerprint`へ導出する。よってRecipeがコマンド型でも**Studioの正式non-Plugin Build Subjectを受理・実行・追跡したことにはならない**。単に`command`文字列を許可したりStudio IDをPlugin欄へ入れたりせず、既存BuildExecution/ArtifactSetへtyped Subjectを接合する最小案を比較（A:共通BuildPlanのSubject一般化とPlugin互換、B:非Plugin用の薄い実行経路を既存BuildOperationとArtifactLockに接続。既存Plugin互換に大きな影響が出るならBを優先）。第一差分は一Studio SubjectのIntent→選択Builder／実Builder→Output digestを独立に照合すること。続くN0定義・生成Input、N1 Build Plan/実Child、N2 esbuild→Go埋込Byte、N3**Stage1 TEI実ProcessによるStage2 Wails再構築**、N4 Windows WebView2実Inspectorの一試行を別の資格で受入れる。NPS-01〜08を#4583/#2831/#1180/#2514の既存Expected候補へ渡す。Graph/Rule→Go ReferenceやPin A/Contextの限定合格をL3製品合格に足さず、外部Go Launcher二回やGoへの丸投げを自己構築とは数えない。通常製品tei→実Provider→独立Consumer Readも並行P0として維持。**実装済みの最小拒否基線**：`internal/buildintent/nonplugin_gap_test.go`へCommand Subject／go-package／go-commandの宣言が現行製品`ParseIntent`で明示拒否されることを確かめるGoテストを追加。正式な非Plugin受理ではなく、Plugin BuildPlanへ偽装しないための現在の失敗境界である。合格すれば次にN1のTyped Subject/Artifactへ進む。**非Plugin Build型や実Stage2の新実装・Windows・Owner受理はNOT RUN**。

**Studio PinとCLI必要情報を同一Subjectで照合する次段（§23、2026-11-15追補）**：[Desktop Host設計 §23](design/DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md)で、実CLI 13 ReplyからのD0.1/D0.2限定PASS（#4696）と、Pin担当Draft #4597がWorkbenchに追加したPin/Unpin/再確認・独立pinned-inspectionを**別Consumer／別Attempt**として結ぶ。次の最短検収は、WorkbenchでBを表示中にA（sales.order）をPinしてHost-owned Inspectで再確認してもBのEditor/Graph/Focus/Noticeを変えないPIN-R0の実DOM観測と、CLIのAに必要なData・Relation・Rule・Capability・Application五件の限定情報の参照条件を別票で比べること。Workspace表示IDやWorld/Source Index Fingerprintだけで同一Root／Source occurrence／現在Grantを証明しない。PDC-01〜08は旧Reply、A→B→A、Relationだけの欠落、未実装Markdown Reader、生成JS／埋込Bundle不一致を既存#4515/#3262/#1180/#2514へ渡す。#4597のSource TS画面接続はDraft実装だが、実DOM・Windows WebView2と埋込Bundle同一HEADの合格は未確認。別Pin GUIを重複実装せず、#4721のUX-PINを既存Ownerへ戻す。CLI・Studio二Consumer資格を製品P0の新Hard Gateにせず、正式non-Plugin Stage1→Stage2と初回実Provider Readの優先を維持する。

**D0.2の限定Required集合と不足箇所の特定（§22、2026-11-15追補）**：[Desktop統合設計 §22](design/DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md)で、既存#3262のread-only Context Consumerに、選択Data `sales.order`と独立指定したRelation `sales.order.lines`・Rule `sales.order_line.line_number_positive`・Capability `sales.order.submit`・Application `order_entry`の5必要項目を対応付ける`selectedTaskRequired`を追加した。各項目の既存Definition/Source/Rule/Function検査結果を再利用し、RelationのSource Indexにおける定義Refの一意性・Role・Targetを新たに照合する。故障注入はRelation target/role/重複、Rule種別、Capability出力、Applicationのusesの6ケースと、必要集合外の要素追加による非干渉1ケース。全5件が観測されても**固定Fixture内の宣言・参照の限定観測**であり、必要依存全件、実Provider、Source Parser/Meaning受理、外部Harness完全受領、現在Grant／Effect、テスト自動生成の資格にはならない。[read-only dogfood CI #43616678532](https://github.com/hat47x/tei-platform/actions/runs/43616678532)は、同期後の実コードHEAD `b4ed4b37`でcompleted/success。実CLI13 Replyと6つの対象別拒否＋1つのScope外非干渉、前段8件のSource Anchor負例を同一Jobで確認した。報告は引き続き`partial`で、自動Test Generation資格はwithheld。#3262/#4515/#1180/#2514へ引き渡し、Stage1→Stage2 Studio DesktopをP0に維持する。

**D0を既存CLI検査コードへ接続（§21、2026-11-15追補）**：[Desktop統合設計 §21](design/DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md)と既存#3262のtools/reference-issue-test-ability/context_gate.pyに、公開合成Fixtureの一Subject（semantic-data / sales.order）を、model catalog・inspect・全Source Indexの定義Reference/Artifactへ照合するselected_subject_source_sliceを追加。Subject ID・索引mode・role=definition・Source path・SHA-256付きArtifactの一致を限定判定し、独立した8変異（Catalog/Inspect status、mode、role、path、重複Reference、Artifact改変・欠落）を既存selftestへ接続。これはCLI応答内で報告された同一Anchorの照合であり、外部AI受領・実Host Grant・全Required Scope・Source原文実再取得・Meaning受理ではない。次は当該HEADの実CLIから13 Replyを取得したReference CIと、#4515のread-only Pinから同じSubjectを使う検証へ進む。既存Context Consumerを再利用し、第二Parser/Context Store/Runnerを作らない。**限定実証を追加確認**：[Issue-to-Design read-only dogfood CI #43612681535](https://github.com/hat47x/tei-platform/actions/runs/43612681535)は実コード変更HEAD `78dbf5e7`でcompleted/success。実CLI 13種類のReplyを用い、追加8件のselected-Subject Source Anchor負例を判別し、Context全体は引き続き`partial`、Test Generation資格はwithheld。別Source/外部Providerの全依存、実Windows、現在Grant／Sink Disclosure、外部Harness完全受領、意味Owner受理は未実証。Desktop Stage2 P0は維持**。

**三つの閉包と段階的なContext引渡し（§20、2026-11-15追補）**：[Desktop Host設計 §20](design/DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md)で、Subject/Information（Workspace・Source・意味ID・必要依存）、Instruction/Documentation（Plugin・Skill・依存マニュアルの実読取と版）、Capability/Authority（選択Provider・実呼出し・現在Grant・Sink Disclosure・実受領）を独立に検収する。C0の第一実装片D0は、既存CLIのmodel catalog／model locate／workspace-files／inspect／trace／source-index／knowledge-context／validate／requirements／plugin-lockの実JSON契約を照合し、Required/Loaded/Parsed/Bound/Returnedの欠損を一Subjectの公開Fixtureで示す。D1はPlugin文書と上位依存資料の読取・版照合、D2はCLI／Studio／外部Harnessの対象一致・開示・分割受領、D3はAI Patch/Effectの独立再読込みとReceiptを試す。Qualified Target RefはSource／Meaningの用途別投影であり、新正本／権限Token／永続Selection Storeではない。Prepared→Delivered→Produced→Recorded→Verified→Current eligibleを混同せず、8,192byte途中切断、旧SourceやGrantの流用、非冪等再送を拒否するSCH2-01〜10を既存OwnerのExpectedへ引渡す。#4515/#3262/#2516/#1158/#951/#2605/#1180/#2514に責務を分担。D0/D1はStudio GUI未完成でも並行可能、全工程をStage1→Stage2 Desktop製品P0のHard Gateにしない。**文書のみで新CLI/Plugin実験・外部送信・Owner受理はNOT RUN。**

**Subject選択→CLI/MCP/AIへ渡す経路（§19）**：C0は既存CLIで一SubjectのCatalog/Inspect/Source/Required Contextを実読取する。C1はUIローカルの明示PinとFocus・Dirty Buffer分離。C2は用途と開示先の現在資格を確認し、CLI/MCP/Harnessへread-only Contextを完全受領させる。C3はAI提案を差分ReviewとSource再検査を経て既存Acceptanceへ接続する。Pinは操作意図であり、現在Grant・Source受理・外部送信許可ではない。描画用Edgeや同名Nodeは正式Relation Identityにならず、Workspace/Revision/Source A→B→A、partial、Reader未束縛、8,192byteの途中切断、非公開情報を独立負例で拒否する。

**Pin実装の受入順序（2026-11-15）**：[Desktop Host設計 §20](design/DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md)へ、Pin Intent／Focus／Harness Attemptの寿命、Source・Required Context・Disclosure・Tool完全受領の別資格と、R0 CLI実Read→R1 Pin非干渉→R2実Windows配布→R3外部Harness→R4候補Reviewの検収順を統合する。YICの実行票はmain統合済み（[#648](https://github.com/hat47x/tai-pleroma/pull/648)、[#650](https://github.com/hat47x/tai-pleroma/pull/650)）。**Pin実装 #4597はDraft／main競合／実Node24・生成Bundle一致・実WebView2未検収**であり、文書をmainへ入れても製品合格とはしない。TEI #4583のセルフビルド／#2586の実Provider／TAI #168は引き続き独立P0とする。

**優先度とOwner**：C0はGUI未完成でも並行可能。#4515/#1824はPin/GUI、#3262はRequired Context、#2516/#1178はDisclosure、#1158は型付き操作、#951はPlugin文書、#1180/#2514はExpectedを担当する。新Host/Source Resolver/Harness/汎用Bridgeは作らない。正式non-Plugin BuildIntent→実Stage1→Stage2→Windows WebView2をP0に維持し、C1〜C3を追加Hard Gateにしない。現段階はdocs-onlyで、新MCP/AI送信、実Windows E2E、Owner受理はNOT RUN。


#### 2026-11-14 第13次更新：SaaSの複数Worktree・DB実行対象を段階分離しP0 Self-buildを保護

**最新の限定基線**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `0fcb7b6265d7dfc5b5e93017e5c99e296eb13134`のWindows N2→N3実Bundle/EmbedFiles前後byte照合・改変コピー拒否・実WebView2はCI **12/12 success**。[Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) HEAD `f1e219a`（Builder/Child事前pin）は**8/8 success**、[Draft #4624](https://github.com/hat47x/tei-platform/pull/4624) HEAD `2d403f49`（Resolver不正応答拒否）は**5/5 success**、ただし別HEAD/Draft・未merge。[Draft #4627](https://github.com/hat47x/tei-platform/pull/4627) HEAD `9491bf9e081bf98f39056d0a4142c2764c31289d`のWails Server `StudioBridge` exported 15 Method inventory/Service登録レビューtestは[Wails Host Run #43544229895](https://github.com/hat47x/tei-platform/actions/runs/43544229895)を含むCI **5/5 success**。**Method棚卸しの合格≠共有Web認証・実HTTP/WS公開遮断・DB/Workspace隔離**。現main WorkbenchはTypeScript/esbuild/CodeMirrorで、ReactもLitも依存していない。

**直列P0は変えない**：①#4623＋#4624を一HEADで宣言→Builder選択→実子起動前拒否まで統合、②N0 Inspector TS→N2選択Input（H0）のProducer/Consumer Role/digest/Attemptを結ぶ、③N1/N2 Node/esbuild/N3 Go/Wailsの**非Plugin Desktop**実Builder・Consumed Source・ArtifactSetを既存#1080/#1097 BuildExecutionへ段階接続、④生成Stage1 TEI自身の実Process→Stage2 Windows Desktop→当該Stage2実WebView2/BridgeをB2/B3として独立検収する。H1限定PASSをOwner受理/全Closure/完全セルフビルドへ拡大しない。SaaS・WASM・UI Framework変更は新Hard Gateにしない。

**並行P1：Desktop→複数対象・共有Webへの順序を追加**：まずローカルStudioでHost許可済み**2 Worktree×2 Database Bindingのread-only preview**を成立させ、Project/Repo/Worktreeの実Root・Git branch ref＋解決commit・Source revision・Environment/Profile→実Database instanceの束縛を**Viewer Selection・Host解決Context・Build/Operation Attempt**として別記録する。Git branch文字列/DB alias/Client入力だけでCurrent Grantとみなさず、別worktree同名SubjectやA→B→A、同alias別DB、Dirty Buffer/保存Source差を拒否。Provider/Environment#2586、Source/Root CAS#1529、Studio Host#1824の既存Ownerを使い、新Git/DB Manager/Global Transactionは導入しない。

次に[ADR-0048](https://github.com/hat47x/tei-platform/blob/main/plan/adr/0048-shared-application-api-and-transport-adapters.md)の共通Go OperationでDesktop IPCとWails Serverまたは薄いWeb Adapterの**選択read-onlyメソッドだけ**を比較。現`cmd/tei-studio/main.go`のloopback Wails Serverは全`StudioBridge`（Command/SHO含む）登録で、`studiohost.ReadOnly`はper-browser主体を持たない。#4627のMethod Inventoryを維持しつつ、次は実Binder/HTTP/WSで禁止Method・別Principal/Workspace/Browser Contextの拒否とイベントAudienceの境界を検証する。Server Buildができても即公開・多利用者資格にはならない。さらに将来SaaSのBuilder実Process/ArtifactはServer Hostの許可済み作業領域へ固定し、2 Principal×2 Workspace×2 Worktree×2 DB Bindingの隔離、Credential秘匿・Quota/取消/Recovery・Source Revision競合を独立検収する。**Wails ServerのローカルPoCをWeb SaaSの全セキュリティ基盤とみなさない**。

**公開前の依存版レビュー**：現nested `cmd/tei-studio/go.mod`が固定するWails v3.0.0-beta.29は、公式[v3.0.0セキュリティ修正案内](https://github.com/wailsapp/wails/releases)より旧い。公開Webや多利用者Serverを受け入れる前に、公式修正・残る制約・Upgrade後のBinder/Origin/IPC/WS/実Windows回帰を確認する。既存P0を無審査で依存Upgradeすることも、旧版のPoC successを公開安全とすることも避ける。これはWeb版のP1安全性評価であり、現P0セルフビルドへ新Hard Gateを増やさない。

**UI/Go-WASMの技術比較はP2候補**：現在のplain TypeScript＋CodeMirrorを基線とし、Web Components/Litの代表PaneとReact対照を、同一Operation/FixtureによるPin裏返信/Editor/Graph/IME/Keyboard/ARIA、JS Bundle/再Build、生成AI修正の局所性で比べる。React採用必須でもLit全面移行必須でもない。純粋Graph/Filter/Source projectionのみGo native vs TypeScript vs browser Go/WASMをベンチし、Source/Meaning/Current Grant/Build資格はHostに残す。実反復改善がなければ現行TSを維持し、採用前にP0生成・WebView2を壊さない回帰を要求。

**受入・Owner**：詳細な三Context、D0〜D3段階とSCX-01〜07の負例は[Desktop契約 Draft #4498 §0.13](design/TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md)へ置き、#1180/#2514の既存Expectedに割り当てる。最優先#4583/#2831、Studio/Web#1824/#3421、Worktree/Source#1529、DB#2586、Pin#4515に責務を戻す。全Targetで同じSource/Meaning/Authority正本を二重実装せず、Server/Clientの局所Context世代・Operation実Attemptを明示。**Docs-only**：新SaaS/DB/Git実動、React/Lit/WASM実比較、公開Web認可、Stage1→Stage2実行、Owner受理、main統合は行っていない。

#### 2026-11-14 第12次更新：H1実Windows PASS後のBuildIntentとDesktop／Web／WASM共通化

**最新実証（前回P0-3の限定完了）**：[Desktop統合Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `0fcb7b6265d7dfc5b5e93017e5c99e296eb13134`／[Windows CI #43542664470](https://github.com/hat47x/tei-platform/actions/runs/43542664470)でgenerated/ordinary Windows Wails WebView2両Job success、HEADのPR Workflowは**12/12 completed/success**。N2 esbuild実Bundle出力を`n2-bundle-producer.json`にRole/format/digest/HEAD/Attempt付きで固定し、N3の`go list -tags studio_generated`が選んだ`EmbedFiles`の実byteと**Go build開始前・後**に照合。`n2-n3-bundle-handoff.json`に同一Roleの受領・変造コピー拒否（元を変更せず別コピーに文字列追記）・`readTimeClosureProved=false`を記録。N2選択24件、N3 Go選択312件、生成exe SHA `e2158bc8e20a5f2fe153ca96d08aad1571d2bacc9c87672b288fbfcbe5566205`、実WebView2配信JS SHA `755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`を同HEADに紐付け。**H1（N2→N3 pre/post SHA/Role/Attemptの局所観測）PASS**であり、read-time完全閉包／正式BuildIntentによるN1/N2/N3駆動／Stage1→Stage2 B2／当該Stage2 B3は**NOT RUN**。PRはDraft未マージ。

**P0の未完了優先順へ更新**： (1) [Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) `f1e219a`の`-bind-resolved-builder`（CI 8/8 success）と[Draft #4624](https://github.com/hat47x/tei-platform/pull/4624) `2d403f49`の宣言/Resolver整合（CI 5/5 success）を、#4587の最新mainベースと**同一HEAD/実Attempt**で統合・正負検査。 (2) N0実Generator output `inspection-reader.generated.ts`→N2実esbuild selected input（H0）をRole/入力bytes/Source RevisionでN2実行前後に接続し、旧TS/誤TS・未選択inputを拒否。 (3) N1、N2 Node/esbuild、N3 nested Go/Wailsの実Builder/argv/ArtifactSetを**既存#1080/#1097の非Plugin Build Execution/ArtifactSetへ一工程ずつ**接続し、Workflow Shell直起動を正式BuildIntentへ置換。H1の証拠は失わず、Plugin主体BuildPlanにDesktop exeを偽装しない。 (4) 生成Stage1 TEI自身の実ProcessがStage2 native Windows Desktopを構築し、**そのStage2自身**のWebView2/Bridge/生成Inspector受領をB2/B3として独立検証する。Wails単体成功、Build Digestの一致、CI proxy呼出しを再入成功としない。

**並行するDesktop/Web共通化（P0の新Hard Gateにしない）**：[採用ADR-0048](https://github.com/hat47x/tei-platform/blob/main/plan/adr/0048-shared-application-api-and-transport-adapters.md)に従い、共通Go `internal/operations`＋StudioHost/Provider/Shared Acceptanceの型付きOperation/Receiptを**Desktop Wails Bridge・Web HTTP/stream Adapter・CLI/MCP**へ使う。既存`cmd/tei-studio/main.go`のWails Serverモードは`127.0.0.1`以外を拒否する**ローカルPoC**であり、これをWeb公開Host/Session/Origin/CSRF/CORS/HTTPSの検証済み実装と誤称しない。Workbench UI・read-only Build Preview・Pin/Focusの表示規則を共有し、WailsのDesktop OS WindowとブラウザーServerの実行・開示境界は分離。**既存`StudioBridge`には`ExecuteCommandWithPolicy`、`ExecuteCommandWithReferenceStore`、SHOのOpen/Resume等のexported methodが存在するため、Wails Serverの全Serviceをそのまま公開Webへ公開しない**。Web第一試作は明示したread-only代表OperationのMethod/Event公開リストを作り、ServerのActor/Workspace/Session/Grant・Origin/CSRF/CORS/HTTPS・表示先限定を独立検収する。全Browser向けbroadcast可能なEventへSourceやAI結果を流さない。**Go/WASMは非UIすべての移行先と決めず**、大量Graph/Filter/Projection等純粋関数のTS/Go Host/Go WASM対照（結果同値、cold/warm、memory、IPC bridge/copy cost）から局所採否を決める。WASM局所Pin/Context/CacheはViewer Snapshotであり、Canonical Source、Meaning受理、Current Grant、Build Builder選択、Artifact/Effectの正本・実行許可にしない。

**共通化の最初の検査**：AI OFFでDesktop/Webの同一Operationを実行し、型付きValue/Failure・Source Revision/Scopeを比較。Pin A裏応答がB Editor/Graph/Dirty Bufferに作用0、Web Session A/BのWorkspace/Principal非干渉、WASM旧SnapshotとHost R1の不一致失効、通信断後Effect unknownを独立Expectedへ引渡す。Web Host Security/Wails Server可用性は別環境実証、WASM最適化とB2/B3 Self-buildを直列Hard Gateにしない。詳細は[Desktop製品契約 Draft #4498 §0.12](design/TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md)。Ownerは#2831/#4583（Build）、#1824（Desktop）、#1158/#3421（共通Operation/Web）、#4515（Pin）、#1180/#2514（既存Expected）を維持。**本追補はDocs-only**。新Go/WASM/Web Browser/Stage2実行・Owner意味受理・main統合は行わない。

#### 2026-11-14 第11次更新：最新main追従DesktopとN2→N3 Artifact実受渡しを先行

**現状（exact HEADで独立検証）**：[Windows統合Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `efe1de03c23d7e8bd9271e465fc0b8436f9a993c`はmain merge `03679ccd`後の`mergeable=true`。実CI pathに`cmd/tei/**`と`internal/buildintent/**`を含め、CLI/Resolverが変化したらDesktop Windows CIが実行される。**再照合した新HEADの12 CIは11 success・Health 1 cancelled**。[Windows WebView2 Run #43541884048](https://github.com/hat47x/tei-platform/actions/runs/43541884048)のgenerated/ordinary両Jobがsuccess、2 Artifactあり。[Self-build Run #43541884152](https://github.com/hat47x/tei-platform/actions/runs/43541884152)もsuccess。[Health Run #43541884124](https://github.com/hat47x/tei-platform/actions/runs/43541884124)はcancelledで再検収対象。旧Windows CI #43450255223のPASSは新HEADへ流用せず、新HEAD自体の限定Windows成功として扱う。[共有CLI Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) HEAD `f1e219a`のBuildIntent→実子pre-start pinは**8/8 success**、[Resolver整合 Draft #4624](https://github.com/hat47x/tei-platform/pull/4624) HEAD `2d403f49`は**5/5 success**。**三つは別HEAD・Draft・未マージ**で、単一Desktop E2E資格ではない。

**新P0順序（既存Issue/Scenario/Build機構の範囲）**：
1. **P0-1 本統合HEADのWindows CI受入**：#4587のgenerated/ordinary Desktop、実WebView2/Bridge、Health/Self-buildの完了を同一HEADで判定。以前のmain競合は解消済みなので旧競合解消タスクは再要求しない。ただし#4587内N0はWorkflow Caller pin、N1〜N3はWorkflow直接実行であり、正式`BuildIntent`による全工程Buildではない。
2. **P0-2 Builder/Resolver同一HEAD**：#4623`-bind-resolved-builder`と#4624の宣言/Resolver応答整合拒否を同一HEADへまとめ、Intent→Resolved Plan→実子image/argv→Artifactを一Attemptで正負検証。正しいBuilder Bを選んだのに実子A、またはResolverが宣言と異なるAを返すときは子Call 0/Artifact 0。両PR個別Greenの合算は禁止。Catalog Authority/Meaning/現在Grantは独立。
3. **P0-3 最初の非Plugin Artifact辺はN2→N3**：実`n2-esbuild-metafile.json`の唯一の`cmd/tei-studio/assets/workbench/main.js`出力Role/digestを、実`go list -tags studio_generated`の`EmbedFiles`選択＋N3 Go build前後のbundle実byteと結ぶ。N2出力A/N3 embed B、同byte長の別bundle、PowerShell Step間変数消失、実行後に古いJSへ差替えを独立Expectedで拒否する。次にN0 Inspector TS→N2選択Source、N3 exe→実Windows配信JSへ拡張。現在の`b1-stage-input-observation.json`等の**既存Evidenceを拡張**し、新独自Manifest/Build Engineを作らない。`tools/reference-build-execution-binding/main.go`の`BuildExecutionPlan/ArtifactMember/ArtifactSet`は参照`package main`であるため製品APIへの直接importは不採用。非Plugin Desktop Targetは既存#1080/#1097担当との薄いAdapterとして設計。
4. **P0-4 Stage再入と実Consumer**：生成Stage1 TEIの実ProcessがStage2 Windows Desktopを構築した履歴（選択Intent/Builder/実invoker/Output exe）と、**そのStage2実exe**のWindow・WebView2配信JS/Bridgeの消費を別検収。Stage0/CI代理Go buildや旧Windowは不合格。OS Loader TOCTOU、Node/Go外部依存read-time全閉包、Runtime Provider Grantはunknownを維持。

**並行Studio/Web実装**：AI OFFのPin A裏再確認でFocus B/Editor/Graph/Dirty Buffer維持、保存済みCanonical Sourceの明示Build、実Build/Artifact/Consumer Receiptのread-only Preview、旧exeの暗黙置換禁止。Wails Desktopと将来のWeb Serverで同一Operation/Evidenceを使い、UI側のFocus/Pin/Viewer世代とHost Source/Build/Authorityを別寿命とする。X-TEA三要素・SHO/SUI・Plugin Profile/DB・AI Harnessは並行するが、P0セルフビルドへ新Hard Gateを増やさない。具体的File/Role/負例、H0 N0→N2・H1 N2→N3・H2 N3→実Consumerは[Desktop製品契約 Draft #4498 §0.11](design/TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md)へ一本化。**今回の改訂はDocs-only。新Go/Node/Windows実行・Stage1→Stage2再構築・Owner意味受理/本番統合は実施なし。**

#### 2026-11-14 最新差分：Resolver応答の宣言適合とResolved Builderの実行前束縛

- **実装更新A：選択Plan→実子**。[Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) **exact HEAD `f1e219a25976946bcc8a19ef57f5490333ec8a84`**は、`tei self-build -bind-resolved-builder`を明示したとき、同じTEIプロセスがCaller指定のBuild CatalogとSourceから解決した`ResolvedBuildReceipt.Plan.Builder.ArtifactDigest`を実Child画像の起動前SHA照合へ渡す。Selected B/Actual Aは`SELF_BUILD_BUILDER_SELECTION_MISMATCH`で拒否し、PlanReceiptを保全、ChildRunとArtifactを作らない。**当該HEADの8系統GitHub Actionsはすべてcompleted/success**。ただしCaller指定Catalogの意味/Authority、BuilderのOS Loader実read瞬間、Node/esbuild/Go/Wails多工程とStage1→Stage2は別資格。PRはDraft/未統合。
- **実装更新B：宣言→Resolver**。[Draft #4624](https://github.com/hat47x/tei-platform/pull/4624) exact HEAD `2d403f4996c205514dc57d5fc1237127c6bd994b` は`internal/buildintent.Resolve`で、可換Resolverの返すSource Kind/ID・Builder Contract/Major/指定Provider/Implementation/Pin/Pinned・依存Kind/ID/Majorが元Intentと違う場合、Resolved Receiptを返す前に拒否。PR記載のGo/race/CLI限定テストは成功。**当該HEADのGitHub Actionsは確認時点で実行中のJobがあり、全体CI成功・#4623との統合成功は未主張**。Recipe/Target適合、実Source raw byte、現在Grant/Meaningは別検査。
- **現行主線への影響**：前回の「Resolverが選択したBuilder digestを実子へ渡すコードがない」という不足は**#4623のOpt-in一工程に限って解消**。新たな残差は①#4623と#4624の別Draftを最新版mainで統合し、宣言→Resolver→Pin→子→Artifactを**一つのHEAD/Attempt**で検収、②N0生成TS→N2実esbuild→N3 Go embedの**非Plugin Desktop Artifact edge**を既存BuildExecution/ArtifactSetへ接続、③生成Stage1実TEI→Stage2 native Windows Desktop→そのStage2自身のWebView2/IPC受領をB2/B3として別検収すること。旧[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)はmain競合と共有CLI重複があり、新pin付きWindows統合HEADのCIが未観測。#4623 Greenと旧#4587 Greenを合算しない。
- **既存Ownerへの引渡し**：#2831/#4583（Build Intent/実Builder/Artifact）、#1824（Windows Window/IPC）、#4515/#3788（Focus/Pin/Dirty Buffer）、#1251/#4570（Stage2 Runtime Plugin実Consumer）、#1180/#2514（独立Expected）。最初のUIは同じOperation/Receiptのread-only Build PreviewとPin A裏確認中のB閲覧保持。Builder byte一致をOwner意味受理や現在Grant/Disclosureへ昇格せず、Plugin主体のBuildPlanにDesktop exeを偽装しない。段階別E1〜E5、Provider/AI/SHOの資格分離は[Desktop製品契約 Draft #4498 §0.10](design/TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md)に集約。**今回の追加は文書更新のみ**、新Go/Windows E2E・Owner受理・main統合は行っていない。

#### 2026-11-14 更新：選択Builder実行前Pinの限定CI成功とB1-B実結線

[共有CLI Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) exact HEAD `929fcf857cd0f6a792a69fb23cadfb5eccc05e90`では`tei self-build` primitiveの`-expected-child-digest`が、子Builder実exeのSHAを**起動前**に照合し、Selected B/Actual A・不正pin・境界Launcher迂回を拒否する。実CIは同HEADの**8系統すべてcompleted/success**（Self-buildとRepository Healthを含む）。ただし**Caller指定のPinであり、BuildIntent Resolver由来の選択や全Desktop経路は未成立**。最新mainからの共通CLI差分のみを#4623に置き、Desktop統合[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)の旧HEADと二重のCLI差分を合算・採用しない。#4587のpin統合Windows検収は新HEADで未確認、merge conflictの解消とCI再実行が必要。

**次の実装順**は (1) #4623の共通CLI pinをOwnerレビュー・回帰後に統合判断し、#4587の同じ2ファイルの重複を解消／最新mainへ追従、(2) 既存`internal/buildintent.Resolve`が導出する`Plan.Builder.ArtifactDigest`を**対応可能な一工程**の実Invoker Pinへ渡し、Plan FP・選択Builder ID/Contract・実子image/argv/Artifact roleを一Attemptに束縛、(3) N0実TS→N2 esbuild→N3 Go embedの成果物・後段実入力を最小非Plugin Target Adapterで結ぶ、(4) B2では**生成Stage1 TEI実プロセス**によるStage2 Windows Desktop Build、(5) B3では**そのStage2 exe自身**のWindows WebView2/IPC/生成Inspectorを検収する順。非Plugin DesktopをPlugin Subjectに偽装せず、既存`BuildExecution/ArtifactSet`・Build Intent/Builderを再利用する。前後SHA、Go/esbuild申告Inputsだけでread-time閉包やWindows OS強制を主張しない。対応不能工程は`unsupported`、未観測依存は`unknown`。

**既存Scenarioに渡す最小負例**：ResolverがBuilder Bを選択→actual A起動を事前拒否（子/Launcher/Artifact生成0）、Caller pinだけをL2へ昇格しない、#4623のGreen＋#4587旧Windows Greenの別HEAD合算を拒否、N2 output A→N3 embed Bを拒否、Stage2生成成功＋旧Windowを当該Stage2 Consumerと誤認しない。AI OFFのread-only Navigate/Pin（B閲覧中のPin A裏再確認でBのDirty Buffer不変）、Dirty/Saved/Meaning/実Builder/Runningの版分離は#4515/#1824/#3788の並行レーンで続行し、AI/SHO/X-TEA/実DB/Plugin全対応をP0の新Hard Gateにしない。判断とOwner別の反例は[Desktop契約 Draft #4498 §0.9](design/TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md)へ集約。**この追補はDocs-only・Draft未マージ**で新しい製品実装PASSやOwner受理を意味しない。

**2026-11-14の現行差分（Desktop第8次横断統合）**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) exact HEAD `24bfaed8fadcce5687fb7f02a121a21fd70f2861`／[Windows CI #43450255223](https://github.com/hat47x/tei-platform/actions/runs/43450255223)で、通常Go Buildで用意した**実TEI CLIプロセス**がprimitive `tei self-build -child`を通じてN0 Inspector Generatorを起動し、親/子の実image・argv・exit・生成TS SHA、別Process比較、no-clobber負例を検証した。同一CIではN2 esbuildの24選択入力、N3 Goの303選択入力、生成native Desktop/実WebView2も限定成功。**Build Catalog/Intentによる選択とN1〜N3のTEI自身による実Builder起動、生成Stage1→Stage2は未実証**。B1全工程の`qualifiedBuildIntent=false`、閉包`unknown`、B2/B3 `NOT RUN`を保持する。従前の「N0はTEI外部からしか動かない」という記述はこの固定HEADについて更新するが、L2/L3へは昇格しない。

**次のP0を一辺に限定**：#4583/#2831がN0実Inspector TS→N2 esbuild→N3 Go embedで、宣言Build Intent/選択Builder、実Process image/argv、実消費入力・出力Artifact role/digestを**同一Attempt**へ結線する。既存#1080/#1097のBuild Execution/ArtifactSet契約を参照し、現`BuildIntent`（一Subject/一Recipe、Plugin主体）が表せない非Plugin Desktop工程に必要な**最小Target Adapter**のみ設計する。CI Shellを正式Build Planへ偽装せず、N2 metafile・N3 go list・事後SHAからread-time全依存閉包／OS強制を推論しない。次にB2では**生成Stage1実TEI**がStage2 exeを構築し、B3では**当該Stage2実exe**のWindow/WebView2/IPC/配信JS/生成Inspector利用を独立受領する。異HEAD・旧Window・Stage0代理実行を合算しない。

**Desktop実用の並行レーン**：#1824/#4515/#3788はAI OFFでNavigate→明示Pin→裏Inspect→再Navigateを同じSource/Qualified Subjectで実Workbenchへ通し、B閲覧中のPin A再確認でもB Editor/Graph/Dirty Bufferを変えない。Build Previewは既存read-only Operation/Evidenceの投影から始め、Dirty Buffer・保存済みCanonical Source・Accepted Meaning・Builder実入力・生成Artifact・現在Windowを別版として示す。未保存Bufferの自動Save/Build/Acceptance、旧Replyの自動資格復活、Stage2生成直後の旧Studio上書きを認めない。SHO/X-TEA三要素/DB Read/外部Harnessは既存Ownerの並行レーンとし、P0 Self-buildの新Hard Gateにしない。UIはFocus/Pin/Reviewを区別し、Declared/Observed/Enforced、PASS/FAIL/NOT RUN/UNKNOWNを既存Receiptから表示する。詳細な製品契約と既存DT/ED/PIN/BDへのExpected引渡しは[Desktop契約 Draft #4498 §0.8](design/TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md)に集約する。**今回の反映は文書のみ**で、新たなGo/Node/Windows/実Provider試験、Owner正式受理、main統合は行っていない。

**最上位の製品目標は、TEI Studio Desktopを実OS上で構築・利用し、TEI自身が次のStudioを再構築すること**。この目的に対し、最初から新しいDSL/汎用Interpreter/独立Build Hostや全機能のX-TEA・B2C実装を要求しない。既存Wails、Workspace/Source Index、標準BuildIntent/Builder、型付きOperation/Providerを使い、以下の証拠を段階的に接続する。段階の記述は新しい全体Hard Gateではない。

**現実装基線と今回の新Draft**：
- [Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) exact HEAD `0fcb7b6265d7dfc5b5e93017e5c99e296eb13134`／[Windows CI #43542664470](https://github.com/hat47x/tei-platform/actions/runs/43542664470)ほか同HEAD **12/12 success**。N2 esbuildが実生成したJSをRole/`sha256`/Attempt付きで記録し、N3 `go list -tags studio_generated`が選んだEmbedFilesの実byteとGo Build前後で照合、独立改変コピーを拒否、生成版/通常版の実WebView2 smoke成功。**H1 N2→N3の局所実証済み**。N2/N3の実InvokerはまだWorkflow、compiler read-time完全閉包・B2/B3・Owner意味受理は未実証。
- [Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) HEAD `f1e219a`の`-bind-resolved-builder`／Child pin検査はCI **8/8 success**。[Draft #4624](https://github.com/hat47x/tei-platform/pull/4624) HEAD `2d403f49`の宣言/Resolver応答一致拒否はCI **5/5 success**。両者は別HEADで、Plugin主体BuildPlanの限定機能。
- **[統合Draft #4634](https://github.com/hat47x/tei-platform/pull/4634)**：#4623/#4624からGo本体3ファイルと正負試験3ファイルを同一main-based HEADに集約。[exact HEAD `653d0d9a1d0125631aec4379550d2bce20364207`](https://github.com/hat47x/tei-platform/commit/653d0d9a1d0125631aec4379550d2bce20364207)の[Self-build #43548771542](https://github.com/hat47x/tei-platform/actions/runs/43548771542)・[Repository Health #43548771555](https://github.com/hat47x/tei-platform/actions/runs/43548771555)ほか**8/8 completed/success**。これは同じCheckoutで宣言→Resolver整合と選択Builder→実Child起動前pinの両実装が共存してGo回帰を通過したという限定実証であり、Caller Catalogの意味資格/非Plugin Desktop Builder/OS Loaderのread-time byte/Stage1→Stage2を保証しない。以後のPlan更新commitは別HEADとして再CIする。

#### 2026-11-13 B0同一HEAD CI成功とB1の実コードFit/Gap（#4583）

- **B0の実証基線**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `b26e87781e00b19bf142b22a35cfc62a857e88a3` の [Windows CI #43368714294](https://github.com/hat47x/tei-platform/actions/runs/43368714294) は `generated-desktop-webview2-smoke` と元の `desktop-webview2-smoke` がともに成功。生成InspectorのContract digest/実受理回数と、実WebView2配信JSのSHA-256一致（682,327 bytes、`755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`）を**同じHEAD・生成版Job**で確認。Repository Health・Embed等の関連CIも同HEADでsuccess。#4572の追跡済みPython `.pyc` 2件は統合PR上で除去。ただし#4587はDraft・未merge、Stage2 Desktopビルドの証拠ではない。
- **B1-A／N0の実生成を同一Windows CIで追加検収**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)をHEAD `708ca27eb39ae31a039565429876f214a421ca52`へ更新し、[Windows Actions #43385876034](https://github.com/hat47x/tei-platform/actions/runs/43385876034)の生成版・通常版WebView2 Jobは両方completed/success。追加した生成版JobのN0で**既存Go Generator実行・独立Goテスト、契約/Transform raw digestと生成器報告値、再生成TSとWorkbenchが使うTSのバイト一致**を検査し、ログ上のN0出力 SHA-256 `6d3dfe806ef758dc4746ccc34b5ae646533eb056b78cbbc098630993d384586b`を確認。同じJobのWebView2実配信JS 682,327 bytes／SHA-256 `755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`も一致。**N0実生成→実Desktop Consumerの限定Claimは成立。ただしN0〜N3がBuildIntentを通ること、Stage2 Desktop buildは未実施**。PRはDraft・main未統合。
- **B1-Aの指定ファイル目録も実Windows CIで成功**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `d7ad13ceabc39b3917a129409630879564f8384c` の [Run #43388804188](https://github.com/hat47x/tei-platform/actions/runs/43388804188) で、generated/ordinary Windows WebView2両Jobがsuccess。生成JobはN0（契約・Transform・生成TS）、N1（Fixture・生成Go/TS）、N2（Inspector TS・main TS・Node lock・bundle）、N3（root/nested Go module・生成Go/JS・Windows exe）の**名前を固定した実ファイル**をハッシュ化し、`b1-stage-input-observation.json`を生成・Artifactにアップロード。ログ上のexe SHA-256 `9c86cd2b4e3894eb9dcca1f5c7fd5a7ed18d0c274af068a83406e9f25967f437`、WebView2配信JSは682,327 bytes／`755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`。**これは指定入力のWorkflow観測であって、依存閉包やTEI BuildIntent実駆動の証明ではない**。Evidence自体も`qualifiedBuildIntent=false`／`dependencyClosure=unknown`と記録し、Stage1によるStage2 Desktopは未実施。
- **B1-A／N0の生成物引渡しも実CIで成功**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `7babc6ad889960c5e75c1c3cbdd00a1518adeec5` の [Windows CI #43394244960](https://github.com/hat47x/tei-platform/actions/runs/43394244960)は生成版・通常版WebView2がともに成功。N0で実再生成したInspector TypeScriptをチェックイン済みInputとbyte一致検査したうえで、**実生成ファイルそのものをWorkbenchがimportするパスへ上書き配置**し、配置後SHA-256も照合した。後続のNode/esbuild→Wails native Desktop→実WebView2が同一生成版Jobで成功。実N0 part SHA-256 `6d3dfe806ef758dc4746ccc34b5ae646533eb056b78cbbc098630993d384586b`、Windows exe SHA-256 `ef545b98746a97dac27aff587b61dada206df959f034f97ad07d8397ca2b7cf0`。これで**N0→N2の限定的な実ファイル受け渡し**は成立。ただしN0〜N3がTEI BuildIntentで実駆動されたこと、esbuild/Go全Required input実読取とOS強制閉包、Stage1→Stage2 Desktop Buildは依然**NOT RUN／UNKNOWN**。工程外のCIが実行したという事実をL2/L3へ昇格しない。
- **参照Build Execution型を製品APIと混同しない**：`tools/reference-build-execution-binding/main.go`には`BuildExecutionPlan`／`ArtifactSet`等があるが、**`package main`のReference専用型**であり、`internal/buildintent`や`cmd/tei`から直接importして製品実行できない。`internal/plugin.BuildPlan`はPlugin主体の一成果物契約。B1で必要な汎用Build Execution接合はこの既存参照を評価基準とし、製品側の最小共有型／薄いAdapterへの昇格と実Invoker束縛を**具体的な差分**として判断する。Wails exeをPluginへ偽装しない（[Plugin Role設計Draft #4601](https://github.com/hat47x/tei-platform/pull/4601)も参照）。

- **B1は単一BuildIntentの増記だけでは成立しない**：現mainの`internal/buildintent/intent.go`は`BuildIntent`一件につきSubject一件、`plugin.BuildRecipe`は`kind/entrypoint`でNode/Edgeを記述しない。`Resolve`は現状`go-command`のpackage Sourceと、その他のPlugin Definition Sourceを別扱いする。`internal/buildintent/catalog.go`はBuilder候補/依存/Targetの選択を解決するが、複数工程を実行しない。`cmd/tei/cli_self_build_outer.go`は`-build-catalog`から自プロセスで一Planを導出できても、実行は指定された`-child`側であり、**Node/esbuild→Go embed→nested Wailsという全工程の実消費・順序・失敗を自動で証明しない**。
- **最小接合**：#4583 B1でN0（Inspector契約/Transform→生成TS）、N1（PoC B fixture→生成Go/TS）、N2（Node/esbuild→埋込JS）、N3（nested `cmd/tei-studio` Wails native exe）の**入力、出力Artifact role、実Invoker/Target、宣言できる既存BuildPlan/依存と未表現Gap**を実Workflowと比較。既存#1080/#1097のExecution/ArtifactSet/Subject Binding、#943のBuilder、#2831の実Build入口を優先再利用し、必要になったTarget Adapterだけを小さく追加する。Node工程を隠した単一shell child／自己申告の`required=[]`／Goの成功だけでL2とはしない。未表現は`unsupported`、未観測は`unknown`、未試行は`not-run`。
<!-- tei-master-b1b-observation-phase-20261113 -->
- **N2 esbuild実選択Inputと工程間Artifactの境界（#4587）**：[Windows CI #43410465181](https://github.com/hat47x/tei-platform/actions/runs/43410465181)、HEAD `7e77d341` の**生成版・通常版WebView2 Jobはともに成功**。生成版の実esbuildビルドで出力された`metafile`は24件の選択入力を申告し、`main.generated.ts`と`inspection-reader.generated.ts`を含むことを独立確認。選択ファイルの**ビルド後の**SHA-256は`n2-esbuild-selected-sources.json`に記録し、実配信JSとのdigest一致も維持。これにより「名指しファイルが存在する」から「実BuilderがそのSourceを選択した」へ進んだが、Builder自己申告＋事後Hashであるため**read-time byte、未宣言Inputなし、actual Provider選択、L2は未証明**。次は既存Build Execution/ArtifactSetに、実Invoker/工程間Output role/契約Targetと同じAttemptを結ぶ。
- **暗黙runspace共有を捨てる**：続く[Windows Run #43411158181](https://github.com/hat47x/tei-platform/actions/runs/43411158181)は、N2の選択出力をN3の観測結果へ表示しようとした際、PowerShellの別Stepに`$bundleOutput`が持ち越されないため**生成版JobがFAIL**（通常版はPASS）。修正は次のStepで前の変数を参照せず、**N2が検査済みの実Artifact pathと保存したmetafile**から読み直すこと。Stage/Processを跨ぐ受渡しは暗黙のスクリプト変数ではなく、明示的なSource/Artifact/Receiptで固定する。[修正HEAD `8480bdcd`のWindows CI #43411524719](https://github.com/hat47x/tei-platform/actions/runs/43411524719)で**生成版・通常版両Jobともsuccess**。N2選択入力24件、esbuild報告のmain.js出力先と実ファイルbyte長照合、N3生成exe SHA-256 `1d6722b79eafa4c0b74ab4c237cf393e1dcada27bd084d1fc9928558f7597b39`、実WebView配信JS 682,327 bytesの一致を同一HEADで検証した。失敗Attemptと修正後成功Attemptを混ぜず、B1 BuildIntent実駆動／Stage1→Stage2 Desktopは未実施とする。
- **N3 Goの実選択入力まで観測範囲を拡大（#4587）**：[Draft #4587 HEAD `001b40ce`](https://github.com/hat47x/tei-platform/pull/4587) の [Windows CI #43426489587](https://github.com/hat47x/tei-platform/actions/runs/43426489587)は生成版・通常版WebView2両Jobがsuccess。N3の**実Goビルドと同じnested module／`studio_generated` tags**で`go list -deps`が選択したリポジトリ内ソース・embed等**303ファイル**を記録し、実`go build`前後の各byte SHA-256を比較。Go選択に`main.go`とN2出力の`assets/workbench/main.js`を必須とし、`n3-go-selected-sources.json`とN2のesbuild選択24件を同一生成版Artifactに収録。生成版exe digest `e49fbbbe9ac7ccfdbefc0955a2844ffa71af8a36077e14c002ff33301769bbe0`、WebView2配信JSは682,327 bytes／`755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`。**到達はB1-Aの選択集合／前後照合**であり、実read-time byte、TOCTOU/ABA、Go SDK/Module Cache等外部依存、Node環境閉包、actual Builder Provider強制は未証明。特に現`BuildIntent.Validate`は`go-command`定義Source分岐があってもSubjectをPlugin限定とし、`plugin.BuildPlan`がPlugin Identityを要求するため、**Desktop exeをPluginに偽装してB1-Bを通すことはしない**。#1080/#1097のReference-only BuildExecution/ArtifactSetを製品側Adapterの検討材料とし、#2831の既存実Build/Invokerに**非Pluginの工程出力roleと実呼出し**を束縛する薄い接続から進める。B2 Stage1→Stage2 Windows Desktop／B3そのexeの実WebView2は未実施。
- **B1/N0に『TEI自身が実Generatorを起動』する細い実経路を追加（#4587）**：[Draft #4587 HEAD `24bfaed8`](https://github.com/hat47x/tei-platform/pull/4587)／[Windows CI #43450255223](https://github.com/hat47x/tei-platform/actions/runs/43450255223)は**生成版・通常版Windows WebView2 Jobともsuccess**。既存`cmd/tei`と`reference-studio-part-generator`をWindows実exeにビルドし、**TEI実Processの既存`self-build -child`**からN0のGenerator子を起動。TEI自身が報告した親・子実行ファイルbyte digest、子argv/exit、子生成TS digestとTEIによるディスク独立再読をWorkflowで実照合し、同一Outputへの再実行拒否・byte不変も検査。別プロセスでのInspector独立再生成Reportは実ProcessのReceiptとファイルを分けて保持し、**TEI子が生成した実TSを後続esbuildへ引き渡す**。同一生成版JobのN2 esbuild選択入力24件、N3 Go選択入力303件、生成exe SHA-256 `f7acb50feb74f9e93afe61ff60fc14718b5ffaf7cf6a88c38f837a6a8f757209`、実WebView2配信JS 682,327 bytes／`755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`を確認した。これは従来のCI直Generator起動から、**限定一工程だけTEI CLIが実InvokerになったB1の進展**。ただし`tei self-build`のprimitive childであり、`BuildIntent`／Provider CatalogからBuilderを解決していない。N1/N2/N3は依然Workflow直起動、TEI CLI exeもStage1生成物ではない。**全工程B1-B／B2生成Stage1→Stage2 Desktop／B3そのStage2 WebView2はNOT RUN**。#2831既存実行入口と#1080/#1097 BuildExecution/ArtifactSetの非Plugin Target接合を続ける。AI候補の採用やOwner意味受理には昇格しない。
- **B1/N0で『宣言された実Builder byte pin』の事前強制を追加（#4587、検証待ち）**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `4dcaa199d0de655e47ee7446da6cbd5160e6218f`。既存TEI CLI `self-build` primitiveへ`-expected-child-digest sha256:…`を**任意指定**で追加し、実子のファイル内容SHA-256と**exec開始前に比較**。不一致／不正形式は拒否し、成果物を書かない。一致したAttemptのReceiptへ、Caller指定のpinと`childPinChecked/childPinMatched`を実子digestと区別して記録する。Go単体負例`TestSelfBuildSelectedChildPinRefusesMismatchBeforeRunning`でSelected B／Actual A・不正pin・一致pinを検証対象にし、実Windows統合JobのN0でもGenerator実exeの事前SHAをCLIへ渡して、**誤pinのN0生成拒否・Artifact不作成**を試す。前HEAD `24bfaed8`／[Windows CI #43450255223](https://github.com/hat47x/tei-platform/actions/runs/43450255223)のN0実TEI起動成功とは別の変更であり、**本HEADのCIは確認できていないためPASSと記載しない**。このpinはWorkflowが明示した選択意図で、BuildIntent Resolver/Builder Catalogからの選択ではない。検査とOS loaderの間の差替えTOCTOU、Node/Go外部入力の閉包、再試行中の内容変化も未保証。次は既存`BuildIntent.Resolve`が得るBuilder `ArtifactDigest`を実子pinへ結合し、工程間Artifact role／Inputの実Consumerに連結する。Wails exeをPluginとして偽装せず、非PluginのBuildExecution/ArtifactSetへ薄く接続する（#1080/#1097/#2831）。
<!-- tei-b1b-builder-pin-mainline-extraction-20261114 -->
- **共通CLI側の実Builder事前pinを最新mainから独立検証（#4583 B1-Bの局所下地）**：旧[Desktop統合Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)はmainとの競合があり、旧ブランチの`cmd/tei/cli_self_build.go`がmainに追加された**OS境界Launcher／`driveSelfBuildIn`**を欠いていた。旧コードへのそのままの機能追加を採用せず、mainのLauncher経路を保全して選択子hashの**実行前照合だけ**を追加。共通CLIの修正を最新mainからの[独立Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) HEAD `929fcf857cd0f6a792a69fb23cadfb5eccc05e90`（2ファイル、mergeable）に切り出した。Mismatch／不正pin／一致pin＋虚偽Artifact／境界Launcher拒否のGo負例を追加。Caller指定`-expected-child-digest`は実子ファイルbyteと`exec`より前に照合し、Resultには選択pin／照合実施／一致を独立記録する。**BuildIntent Resolverから選択したpinではない**ためB1-B完了ではない。Child fileのcheck→OS LoaderにTOCTOUが残り、全依存閉包やOwner意味受理を保証しない。新PRのCIを独立に判定し、#4587の既存Windows N0→N3→WebView2実装にはCI未確認の統合変更として残す。#4623にStudio／Wailsの広い差分を持ち込まず、#4587の本線は競合解消後に別AttemptでWindows CI検収する。次は`BuildIntent.Resolve`のBuilder ArtifactDigestを実起動子と照合し、非Plugin BuildExecution/ArtifactSetへ接合する。新Host／Runner／Gateなし。
<!-- tei-b1b-buildintent-resolved-builder-binding-20261114 -->
- **B1-B／BuildIntent選択→実子Builderの起動前照合を初めて実接続（選択限定）**：[Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) HEAD `f1e219a25976946bcc8a19ef57f5490333ec8a84`では、既存`tei self-build`外側経路に任意`-bind-resolved-builder`を追加。`-build-catalog/-build-catalog-digest/-build-source-root`を指定した**製品内`buildintent.Resolve`**が作った`ResolvedBuildReceipt.Plan.Builder.ArtifactDigest`を、既存`driveSelfBuild`の`-expected-child-digest`相当の実子起動前照合へ渡す。**実Bundle/Builder新Registryは設けない**。選択SHAと実子ファイルbyte不一致時は、実子`exec`前に拒否し、受け取ったBuildIntent/Plan Receiptと`SELF_BUILD_BUILDER_SELECTION_MISMATCH`を同Attemptへ残す（ChildRunなし・Artifactなし）。一致時も`selectedChildDigest`、`childPinChecked`、`childPinMatched`を記録し、Builder一致をArtifact/業務意味/Runtime Authorityへ昇格しない。実Go試験には**正例／Selected B・Actual A負例／Catalogなし拒否**を追加し、旧CLIの未指定動作は維持。前コミット`929fcf85`の8系統CIは全成功だが、**本HEADのCIは別実行で判定**する。現在のBuildPlanはPlugin主体のため、この実例はPlugin buildの限定経路であり、**Windows Wails Desktop.exeをPluginに偽装した証拠ではない**。さらにBuilder実読取時のTOCTOU/ABA、Source/依存の完全閉包、N0～N3全工程実起動、生成Stage1→Stage2 Windows DesktopとそのWebView2は引き続き未証明。次は#4583のDesktop非Plugin工程を既存#1080/#1097 BuildExecution/ArtifactSetに束縛し、選択Target／Step Artifact role／実Invoker／Consumed bytesを一つのAttemptへ連結する。新Host/汎用Runner/第二Gateなし。
<!-- tei-desktop-selfbuild-E1-E5-crosschat-handoff-20261114 -->
- **横断チャットの成熟した境界をE1～E5の実工程で統一（2026-11-14）**：E1＝保存済みBuildIntent→`buildintent.Resolve`が導いたBuildPlan（Candidate/編集中BufferやOwner意味受理とは分離）、E2＝解決Planの選択Builder `ArtifactDigest`→起動前の実子Image比較、E3＝N0実TEI CLI生成TS→N2 esbuildが選択したInspector入力、E4＝N2の生成Workbench bundle→N3 Go `studio_generated`の実`EmbedFiles`選択、E5＝**TEI生成Stage1実Process自身**がStage2 Desktopを再構築し、その**Stage2**を実WebView2で検証する。E1/E2はPlugin主体の限定経路で[Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) HEAD `f1e219a`の**8/8 CI成功**（[Self-build](https://github.com/hat47x/tei-platform/actions/runs/43539856841)／[Health](https://github.com/hat47x/tei-platform/actions/runs/43539856799)を含む）。E3は[Desktop Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `efe1de03`で[Windows run #43541884048](https://github.com/hat47x/tei-platform/actions/runs/43541884048)の**生成版・通常版とも実WebView2 PASS**。ただし実読取時の全入力閉包やE5を示さない。E4は#4587新HEAD `0fcb7b6265d7dfc5b5e93017e5c99e296eb13134`で`n2-bundle-producer.json`へN2成果物の`role=workbench.bundle`・format・実SHA/byte長・HEAD・Attemptを**N2終了時点**に保存し、N3の`go list -tags studio_generated`が選んだEmbedFilesの**Go Build開始前digest／byte長／HEAD／Attempt**と照合、不一致ならN3実行前拒否する実装を追加。別コピーに1バイトを加えた負例を**同じ照合関数**で拒否し、N3実行後も再ハッシュを記録（`n2-n3-bundle-handoff.json`）。[Windows検証run #43542664470](https://github.com/hat47x/tei-platform/actions/runs/43542664470)は**生成版・通常版Windows WebView2 Jobともcompleted/success**。生成版の実ログ：`N2-producer-bundle-recorded`と`N2-N3-embed-handoff-verified-before-go-build`は双方`sha256:755e5e85f8487d4e5361fe4262c77595c3ebc048068f2647b0b27862de804299`、改変コピーの負例とGo build後digest照合もpass、実WebView2配信JSは682,327 bytes／同一SHA。N2 esbuild選択入力24件、N3`go list`リポジトリ内選択入力は今回**312件**（前HEADの303件を流用しない）、生成Windows exe SHA-256は`e2158bc8e20a5f2fe153ca96d08aad1571d2bacc9c87672b288fbfcbe5566205`。E4成功でも、N3 Go compilerのread-time捕捉・cache/external dependency完全性・実exeのbyteにembedされたことの独立証明は別票。E5は**NOT RUN**。**Build包含／実行利用／Authority/Effect／データ保持の8状態を混同しない**：BuildへJSを含めたこと、WebView2が実JSを使ったこと、Ownerが意味受理したことはそれぞれ別判定。StudioはCandidate/PreviewのUIであり意味正本でも独自Host/Approvalでもない。NonPlugin Desktop対象は#1080/#1097のReference `BuildExecutionPlan`/`ArtifactSet`を最小採用し、旧Plugin BuildPlanを偽Plugin IDで使わない。保存済みSource・Generator/Runtime Provider・行動Authority・Consumerを分離し、E4実検収→E5生成Stage1→Stage2へ最短接続する。新Build Engine/Generic Runner/第二Approval Gateは設けない。
<!-- tei-studio-main-merged-b1-n0-webview-ci-20261114 -->
- **2026-11-14 現行Desktop統合HEADへmainを実merge（#4583）**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)は、mainの現行HEAD `e888bb8f21367b721f0984953530da9627e2634d`を第二親とするmerge commit `03679ccd12b2780b4b6e4890792d49518f4461be`で**behind=0／mergeable=true**に復帰。22差分ファイルのうちmain側変更と交差したのは`cmd/tei/cli_self_build.go`と`plan/evidence/README.md`の2件のみ。前者は最新mainの境界Launcher／`driveSelfBuildIn`を残し、共通Builder pin [Draft #4623](https://github.com/hat47x/tei-platform/pull/4623)の実子SHA照合と同じGit blobを保持。後者はmain側の全索引更新を残してInspector生成文書への一行を追加。**単なるファイル上書き／旧mainへの復帰は行っていない**。続く実装HEAD `efe1de03c23d7e8bd9271e465fc0b8436f9a993c`ではWindows Desktop Workflowの`pull_request/push paths`へ、N0で実際に使う`cmd/tei/**`と`internal/buildintent/**`を追加。変更検知から実TEI CLI/BuildIntentが漏れた欠陥を解消した。[新HEAD CI（Windows含む）](https://github.com/hat47x/tei-platform/actions/runs/43541884048)は**main merge後の別Attempt**として試験中であり、古いWindows success（#43450255223）を新HEADの結果に流用しない。**共有CLI PR #4623 HEAD `f1e219a`の8系統CIは別HEADとして全success**（[#43539856841](https://github.com/hat47x/tei-platform/actions/runs/43539856841) self-build、[#43539856799](https://github.com/hat47x/tei-platform/actions/runs/43539856799) Healthを含む）。ただし#4623の正式BuildIntent `-bind-resolved-builder`は未mergeで#4587のN0 Windows Jobには含まれず、そこではprimitiveのCaller pinを使う。BuildIntent選択→非Plugin Desktop全工程の実Invoker連結、Stage1生成TEI→Stage2 Windows Wails構築→**そのStage2** WebView2は依然未受理。N0～N3のArtifact roleとRead-time消費を既存#1080/#1097 BuildExecution/ArtifactSetに薄く束ねるのが次のP0であり、Wailsを偽Plugin扱いせず新Runner/Gateを増やさない。
- **B1-Bで先に塞ぐ『事後Hash＝実読取』の誤認**：[#4587のB1-A実装レビュー](https://github.com/hat47x/tei-platform/pull/4587#issuecomment-6847421815)で、`b1-stage-input-observation.json`の`n2.inspectorInputSha256`・`n3.bundleInputSha256`・`n3.generatedGoServiceInputSha256`は、**N3 Windows Go Build後に名前付きファイルを再読取した値**であることを確認した。`invoker`も現状はWorkflow由来の説明文字列で、実Child image/argvの証跡ではない。B0/N0の実Windows Consumer成功は保持しつつ、この目録を実esbuild／Go Builderの入力閉包・現在資格へ昇格しない。B1-Bの最小作業は、各実Invokerの直前・直後の入力byte／選択Builder／出力を同一Attemptで照合し、差替え・未宣言transitive import・Selected B／Actual Aを既存[#4583の独立Expected](https://github.com/hat47x/tei-platform/issues/4583#issuecomment-6847431909)で拒否すること。**pre/post一致でもTOCTOU/ABAや実Readの全閉包は未証明**であり、#2595/#4486に従い不明を残す。`tools/reference-build-execution-binding/main.go`のExecution/ArtifactSetはReference-only（`package main`）なので、これを正式Product APIとして直接importせず、#1080/#1097の非Plugin Build実行／Subject束縛の境界を維持して小さなAdapterを検討する。
- **B2/B3の停止線**：Stage1のTEI実ProcessがStage2の**Windows Desktop**を起動・構築したこと（実Process image/argv、実Builder消費入力、Stage2 exe digest）と、**そのStage2 exe**が生成Inspectorを実WebView2で使ったことを別確認する。#4572のLinux局所部品再入や#4587のWindows外部Workflowビルドはこの代わりにならない。remote実行を行うなら別Target/Authority契約を明示し、暗黙のWindows fallbackをしない。
- **対象発見の独立性（main修正済み）**：#4577で観測された不存在Subjectに対する`source-index`の`ok:true`・空参照は、[merged #4584](https://github.com/hat47x/tei-platform/pull/4584)で修正された。Subject指定モードは既存`InspectionFromLoad`で実Canonical対象を確認し、不存在なら`SUBJECT_NOT_FOUND`／exit 1、unsupported kindならexit 2を返す（PR HEAD `dcfe1d69`の対象8 CIはsuccess）。**存在するSubjectの参照0件と、Subject自体の不存在を分離**してPin/AI/SHOへ渡す。ただし`internal/operations/trace.go`の参照列挙や`source-index`成功だけで、別Workspace Rootの同一性、現在Scope・Grant、必要な外部Sourceの取得完全性までは証明できない。固定Pin・生成Intent・Handoffでは各Productがそれらを再確認する。

**実装のOwner**：B0〜B3は[#4583](https://github.com/hat47x/tei-platform/issues/4583)、全体Builder/Stage再入は#2831、Windows Desktopは#1824、独立Expectedは#1180/#2514を維持する。X-TEA三要素のModelerプラグイン#4570、Function生成#4559、固定Subject Pin#4515は並行し、B1の新Hard Gateにしない。本追補は実ソースと既存CIの照合に基づく計画であり、**B1〜B3の新しい実BuildやSelfBuild資格・Owner受理は未実施**。

**検収票の分離**：生成したSource、実Builder入力、生成したArtifact、Stage1実Process、Stage2実呼出し、実Desktop WebViewでの使用、外部Provider/Consumerの受領を別欄に記録する。各段階に対応した実失敗・Source改版・未取得/partial・同名別Workspace・旧bundleの負例を独立Expectedで確認する（#1180/#2514）。**今回の計画追補は実装・GUI・CLI・Stage2の新たな製品E2Eを実行したものではない**。

**並行P1：Studio Desktop/Server/WebのService配置**：
- [採用ADR-0048](https://github.com/hat47x/tei-platform/blob/main/plan/adr/0048-shared-application-api-and-transport-adapters.md)に従い、`internal/operations`／StudioHost/Providerを唯一の意味処理として維持。現`cmd/tei-studio`はWails Bridgeから**同一Go ProcessのStudioHostを直接呼ぶ**。Server Buildは`-tags server`でnative Windowを作らず、`127.0.0.1`へ束縛する別Build成果物。従って「Wails Serverを従属ProcessにすればDesktopとServerが自動的に同一実行バイナリ・性能・障害特性になる」とは断定しない。
- **比較する構成は三つ（いずれもB/Cは未採用）**：A) 現行の同一Process Wails native Bridge→Go Operation（P0の実Windows実証基線）、B) **単一GUI exeをself-spawnした従属Workerモード**＋型付きローカルIPC（Wailsの`server` build tagとは別物）、C) **別のWails `-tags server` exeをGUI子Processとして起動監督**しloopback HTTP/WSで同じGo Operationへ接続。Bは配布1 binaryと実起動/終了分離を、CはDesktop＋Serverの2 Artifact/版・IPC公開面を検収する。Windows Named Pipe／Unix Domain Socket／keepalive HTTP/WSは計測・OS接続元制限・stream/backpressure・Serializationを比較し、**IPCの方がHTTPより常に速いとは仮定しない**。既存の[Owner #1824 LPC-01〜08](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6870456170)へ実Process・OS ACL/peer・子孫寿命・Unknown Effect・配布・性能の試験を引渡し、追加のWorker Manager/AuthorityをP0に作らない。
- **判定材料**：同一Fixture／同一実Go Operationで短いCatalog、長いGraph／大量Reader結果、連続入力更新、生成Build progressを測り、cold start・p50/p95/p99、メモリ、Payload copy/serialization、キャンセル、Crash/kill/restart、Window close、Build/Devホット更新、実Windows GUIの停止/応答を記録する。数値閾値は製品要求・基線実測を受けてから設定し、測定前に新子Process/IPCをP0へ強制しない。
- **故障境界**：Client/IPCでAck喪失・タイムアウトが起きてもBuild/DB/外部Effectを自動再送しない。HostによるWorkspace/Environment/Grant/Attempt ID再検査、現在Receipt lookupと`unknown-effect`、OS child終了・再接続時にOwner正本を失わないことを確認する。UIのDirty Buffer・Focus/Pin/Graph/Window世代はHostのSource・Meaning/Build/Effect正本と別寿命に保持する。
- [Wails Server公開面監査Draft #4627](https://github.com/hat47x/tei-platform/pull/4627) HEAD `b9cf5447bcd3881aec06d97e4f3060f7de80b331`は、`StudioBridge`の15 exported Methodと`application.Options.Services` ASTの登録変更・6合成改変例を固定し、[Wails Host #43549001431](https://github.com/hat47x/tei-platform/actions/runs/43549001431)と[Health #43549001517](https://github.com/hat47x/tei-platform/actions/runs/43549001517)を含む同HEAD **5/5 CI success**。旧HEADのgofmt driftはこのHEADで解消。**静的Inventory PASS≠公開WebのHTTP/WS許可/Principal/Workspace/Session/Current Grant**。#3421/#1824のWSP-02b/03実拒否試験を維持し、Command/SHO/Local Pathをpublic Webへ開放しない。

**並行P2（P0のHard Gateにはしない）**：
- 将来SaaSで複数Git Worktree×DB instanceを扱う際はViewer Selection／Host-resolved Source・Git commit・Environment/DB Binding／実Build・Command Attemptを分離。Git Branch名・DB alias・Renderer入力を現在Authorityにしない。まずローカルread-only二対象、次にWebの二Principal等へ拡張する。実Source/DB Current Grant・Git/DB Effectは既存Ownerを使用する。
- [Plugin理想構成Draft #4632](https://github.com/hat47x/tei-platform/pull/4632) HEAD `19dcdb1a`はStudio UI・Meaning・Generator・Target Runtime・TestのContributionとPackage/Contract/Variant/Artifact/Configuration/Resolutionを**将来の設計案**として定義し、docs CI 2/2 success。これを直ちに実製品API/Plugin Registryの完成やTarget包括適合とみなさない。初回実証は、既存X-TEA三視点read-only等で**Studio Contributionの実選択**と、生成アプリに必要な**Target別Runtime Artifact包含**を区別して同一Contract/Lockへ対応付け、DB/認証/組織/テストの実ConsumerとOwnerを独立検証する。非Plugin Desktop Buildはこれを前提にしない。
- 現Workbench plain TypeScript＋CodeMirror／esbuildを維持し、Web Components/Lit・React・Go/WASMの比較は代表Pane／純粋Projectionに限定する。実DOM/IME/Caret/Pin後着・Graph/ARIA・Bundle Build/OS Consumerと生成AI修正後の回帰を測定。未実装Web/共有SaaS・全WASMはP0阻害要因にしない。

**資格とOwner**：#4583/#2831はBuildおよびB2/B3、#1824はStudio/Windows/プロセス配置、#3421はWeb公開面、#1158は共通Operation、#2586は実DB束縛、#951/#2044はPlugin選択/管理、#1180/#2514は独立Expected。上記はPlan/Owner間の引継ぎであり、新Test SSOTや業務受理ではない。**本PRによって新しく実装したのは#4623/#4624由来の限定Go code同一HEAD集約だけ**で、非Plugin B1/B2/B3・Go/WASM・新IPC・SaaS実動は未実施。
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

PostgreSQL以外の実際のデータベースによる検証根拠も重要です。最初の参照事例では、Oracle Databaseに関する#214・#371・#376が該当します。ただし、判定段階Bの第四の必須検証経路には含めません。未完了の範囲は`unresolved`として引き継ぎます。

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

**CI・main健全性の横断運用（2026-11-12、実行計画案）**：製品の実受入と開発時CIの状態を混同せず、現mainのexact HEAD／run attempt／必要scope／環境・ツールチェーンを結ぶ。過去失敗・未実行・意図的停止・runなしを合格と誤認しない。PR→mainの再観測、差分に応じた検証、AI Harnessの実Tool/Consumer証拠、NAS/S3等への大容量Evidence退避と取得可否の扱いは、[main健全性とCI Evidenceの横断運用計画](workstreams/CI_MAIN_HEALTH_AND_EVIDENCE_CROSS_REPO_2026-11.md)にまとめる。既存AGENTS/品質設計・#2018/#2514/#2515/#2600の責務を維持し、新Gate・第二のRegistry・全リポジトリCI自動再有効化は行わない。

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

異なる提供実装は、既存資産との互換性と、特定の提供実装に依存しない境界を検証するために利用します。Oracle Databaseは、最初の参照事例の`master_data`が利用する既存資産の一例であり、検証対象をOracle Databaseに限定するものではありません。MySQLやSQL Serverなどの追加は、具体的な資産と利用条件が明らかになってから判断します。検証では、物理的な事実の観測・読取り専用の制約・実行環境との接続・実際の読取りを区別します。

Oracle Databaseの実環境での検証が未完了であることだけを理由に、固定版での判定段階BのReadyを取り消しません。

### 汎用データソースの開発者向け管理範囲（2026-11-07、#3270）

SQL/NoSQLのデータ、ログファイル、クラウド情報源は別々の製品機能を増やすのでなく、既存Data Operations・Provider・標準データ構造と表示型から構成する。**スキーマ／テーブル／索引／制約の作成・変更・削除をTEI標準に含める**。これらはアプリケーション設計・データ管理の責務であり、DBA/インフラ運用（サーバ、クラスタ、バックアップ、レプリケーション、DBユーザー管理など）は標準外とする。実行時の対象・権限と破壊性は別途検査する。

[汎用データソース・ワークベンチの設計・縦断実証案](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)は#3270の計画として、①現状契約棚卸し・read-only構造観測、②実DBでのReverse/Schema Plan、③Disposable DBでのCREATE/ALTER/DROPと独立読取り、④ログ/NoSQLの部分取得・時系列・型揺れ、⑤CLI/MCP/Studioの共通投影、⑥組合せBOM/Cache資格の検証を段階化する。#1126・#1803・#2622・既存Schema Diff/Migration・#1641・#3267へ責務を戻し、第二のSchema/Query/Cache/Authority正本を作らない。**既存の判定段階B/Cと主線優先順位は変えず、設計・受入案と実装済みを区別する。**

**対応する製品要求は[RQ-035](../requirements/TEI_PRODUCT_REQUIREMENTS.md#rq-035-一般開発者が異種データソースのデータと構造を管理できること)**。2026-11-07に現行コードを静的確認した結果、`internal/database/provider/provider.go`のDB専用SPI、PostgreSQL `Introspect`、`internal/workbench/database/model.go`の派生投影、`cmd/tei/main.go`のSchema Diff/Migration入口は存在する。一方、異種Sourceを含む実Provider Read→完全性付きFunction Data Context→標準表現型の一往復は未確認であり、#2586と最初の不足一辺を検証する。Schema Historyのunknown→applied誤分類候補は既存#2892へ渡す。Trait・Decision Graph・C0/C1・独立期待値への接続とWB-15〜25は上記設計書の11〜15節に集約し、ここへ第二の契約正本を作らない。静的確認は実Provider/Go/GUIの実行成功を意味しない。**

### Core外PluginによるWorkbenchの同一契約実証（2026-11-07、#3270×#1251）

[#3277](https://github.com/hat47x/tei-platform/pull/3277)の[Plugin-first CLI/Studio/Runtime設計](design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md) D1/D2/D4/D5とWorkbench W0〜W5を**依存関係で接続**する。最初にCLI/Studio共通のInspector一操作を同一Contract/Providerで通し、次に**PostgreSQL実ReadとJSONLファイルRead→Function Data Context→Table/Tree/Timeline**を、同じContract選択と異質なProviderで比較する。成果物を読む`workbench-model`と実Provider Row Readは別資格。次に既存Schema Diff/Migrationの隔離DB適用、最後に同じRead/Validator契約を開発用Surfaceなしの生成Goバイナリへ組み込む。

[設計書の§16〜18](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)に実ソースの接続点、Selected/Linked/Invoked、異Alias競合、CLIの呼出別Effect表示、WB-26〜37の独立期待値と停止線を配置し、[合成SQL＋JSONLのFixture](../examples/data-source-workbench/README.md)を受入入力に固定する。Provider/Plugin/Contract Resolverの実装は#951/#1251、実Row→Data Contextは#2586、異DB Aliasは#2611、CLI Effect語彙は#3182、Migration Historyは#2892、検証済みBOMは#3267がOwner。新しいPlugin Registry、Host、Query/Cache/Authority正本を作らない。**この段階では静的コード確認＋Fixture作成のみで、実Plugin選択/Build/Provider接続/Studio/Goテストは未実施**。
### 実Catalogの物理Relationから三視点候補への引渡し（2026-11-08、#3270）

SQL/NoSQL/ログ等のリバース結果は、**Physical Observation→Structural Candidate→Accepted Data/Function/Business→Diagram/CRUD/SHO**の各段階で資格を分ける。物理FK/UNIQUEやログ中の同一文字列だけから業務Relation、FunctionのCRU(D)、実行権限を生成しない。Catalogの0件や未取得を業務上の「関係なし」にしない。

[Workbench詳細設計§19〜20](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)へWB-42〜54と三経路（Schema/実Data Read/三視点）を登録。現行`postgres.Provider.Introspect`→`databaseworkbench.Build`の**合成Catalog read-only経路**でCapturedAt/Fingerprint、UNIQUE/FK・複合参照・IndexのNative式/部分索引/Readinessを落とさずTable Viewへ投影する狭いProduct Goコードはmainに存在するが、その検証は合成Catalogに限られ、実DB/Scope・Row Read・Schema Applyの資格とは異なる。実Provider/Studio/SHOの適格やOwnerによるMeaning受理は別。機能Readは#2586、Required Design Contextの完全性は#3262、三視点Diagram/CRUDのDraftは#3263/#3303、Plugin-first D1は#1251がOwner。実DBのSchema定義/変更操作は標準、DBA運用は非標準の境界を維持する。

**2026-11-08 横断調整**：[Workbench設計§21](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)にREAの外部原本→Evidence→正式TEI Source→実Read/SHO比較、SUI/SEI/KONの任意Consumer境界、共通Operation（ADR-0048）、Migration適用直前のInstance/History/Grant再照合を集約した。最短P0は実Provider Read→Function Data ContextとPlugin-first Inspector二Consumerであり、SUI・SEI・KON・SHOやOntology DSL製品完成を新たなHard dependencyにしない。**

**2026-11-12／REA外部候補の入場を横断計画へ反映（#3394、計画・受理境界）**：実REA 6.7.0の単独JSONと6.8.0の7 Record＋JSONL終端の版別検査は[main #4257](https://github.com/hat47x/tei-platform/pull/4257)で限定実証済み。一方、[PR #4322](https://github.com/hat47x/tei-platform/pull/4322)の`operations.ExternalCandidates`は保存実捕捉Fixtureからの外部Observation 1件＋方向・Edge ID付きRelation 1件の型付きread-only受領を局所実装・試験しているが、**2026-11-12確認時点でopen/unmerged**。元raw byteと派生入力のdigest、Producer/Attempt、Source位置・版、External Node/Edge ID、parallel relation、coverage/unknown/partial、選択部分集合を保持し、外部CandidateをCanonical Subject・業務FK/CRUD・Accepted Meaning・実DB観測へ自動昇格させない。CLI entryの局所成功をMCP/Studioの独立Consumer成功とは数えない。

**最小の検収順**：①#4322のexact HEADとmain差分を照合し、版違い・終端欠落・外部ID混同・向き逆転／平行辺縮約・unknownの観測化・1+1を全Graphとする偽PASSを拒否した上でmainへの受入を判断、②同一型付きOperation結果をCLIと独立したMCPまたはStudio Consumerで実受領し、送出範囲・Disclosure、selected/invoked/received、現在のSource改版・A→B→A失効を別に検収、③選択・Owner受理した候補だけを正式Authoringへ提案。正式`*.tei.md` Reader（#3346）、三視点Modeler（#1124）、SHOのSave/reopen、外部Provider実Rowset（#2586）・旧新DB Migration（#4125）の資格を足し合わせない。[TEI文書Draft #4317](https://github.com/hat47x/tei-platform/pull/4317)・[再投影方針 #4467](https://github.com/hat47x/tei-platform/pull/4467)・SHO #36/#40のmain統合は再利用するが、著者文の自動マージ、引用自動解決、Occurrence ID永続化、Meaning受理は行わない。Legacy Observed Actual／Owner Accepted Desired／Target Observed Actualの三状態は#4125の既存G0〜G5で別々に照合する。

**責務と停止線**：REAの詳細な受入票・独立Expected・最新状態の正本は[#3394の2026-11-12横断追補](https://github.com/hat47x/tei-platform/issues/3394#issuecomment-6813994847)。共通API=#1158/#3419、Provider実Call=#1251、Source/Context=#3262、実Function Read=#2586、DB移行=#4125に帰属。第二Parser/Host/Registry/Meaning Ledger/移行Gateは作らない。新規Consumer実E2E・正式Source Loader・実DB/Owner受理は**NOT RUN**。既存P0の一Function実ReadとPlugin-first実Consumerを、REA T1-B/SHO完成でブロックしない。


### REA受領Readerのmain統合とDesktop優先の実装順（2026-11-13、#3394）

**実装済み（本節の追補時点）**：[PR #4547](https://github.com/hat47x/tei-platform/pull/4547)をmainへ統合（squash `030936b3a48690b63150a8495efa4dc8634eb1c2`）。既存REA 6.7.0/6.8.0の保存済みバイト列からReceipt／構造候補を読む `internal/externalanalysis` と実捕捉Fixture・負例、既存共通JSONLの受領済みbyte読取を再利用可能にした。全体Health・セルフビルド・キャッシュ・依存検査・ドッグフードの**5系統CIが合格**した固定HEADに基づく統合である。ただし**Readerのmain入り≠製品Operation・通常tei CLI・Studio/MCP独立Consumer・Owner意味受理・実Provider/DB観測の完了**。

**R2のmain統合（2026-11-13）**：[PR #4548](https://github.com/hat47x/tei-platform/pull/4548)を独立した5ファイルの型付きread-only Operationとしてmainへsquash統合（`f843b3a894775f9f82049046d4566f37ded9d09a`）。Observation 1件＋有向Relation 1件の固定選択、外部IDの曖昧性拒否、受領Digest・Evidence ID・Producer申告Source Revisionの照合を実装した。PR HEAD `8b9ad73e5f8899edff83e2350ab52efb78ebfe5f`の12 Workflowはcompleted/success。ただしその成功とmain統合は**通常CLI/StudioによるR3実受領・元Source現在性・正式Meaning受理**を保証しない。次の実装対象は共通API二入口の実Consumer受領であり、旧PR #4322の42ファイルを一括移植しない。

**製品全体の順序**：最優先は**TEI Studio Desktopの実Wails Desktop接続とTEIによるセルフビルド**（#1824/#3421；S0 実build→起動→Catalog/Inspect→共通API、以後S1〜S4で再生成／再入場／実OS検収）。REA R1/R2はこれを阻害しない独立read-only線であり、REA/SHO/三要素分析法すべての完成をDesktopセルフビルドや実Provider Read→通常CLI（#2586）の前提にしない。main統合済みR2を使うR3は**通常CLIと独立した第二Consumerで、同じSelected/Invoked/Received・Digest/Attempt/Coverageを実受領**する最小縦断とし、独自のHost/Registry/Parser/Grant Engineを作らない。REA観測から汎用Graph・三要素分析法のData/Function/Business、SHO文書へ進む際は、Source根拠・外部Identity・Scope・意味の明示的採否を分離する。最新版・Ownerは[#3394](https://github.com/hat47x/tei-platform/issues/3394)を参照。

### REA受領から編集までの変更伝播規約（2026-11-12、#3394×#1124×#1126×#3270）

**対象を三つに分ける**。①REA／外部DB Reverseの`Observed Schema`、②TEIで編集・受理する`Table Definition`、③Table Definitionを写像した`Data Model`は、同じ名前でも別の出典・版・適格性を持つ。実DBとの差分は同期候補であり、**外部観測の欠落やREAのpartialをTEI定義の削除として扱わない**。正式な`Table Definition`の変更は、その同一版に束縛されたData Model投影と、機能モデルUIのBinding・CRUD・Query参照・SHO引用へ**変更影響**を通知する。外部REA Node/Edge IDと正式TEI要素ID、Canvas上のOccurrence IDを混用しない。

**GUI/CLI共通の操作意味を固定する**。画面上でUI項目を追加しても、テーブルや列は自動生成しない。必要なら「新しいテーブル項目を作成して結び付ける」を明示的に指示でき、Studioの操作とCLI/CUI・AIハーネスからの操作提案は**同じ型付きIntent・Preview・診断・更新前Revision検査**を通る。明示された編集によってTEI Table DefinitionのField IDを削除したときは、Data Modelの対応項目を同じ編集結果から消し、機能モデルのUI項目は**残したままBindingを解除し未マッピング警告**を出す。元のFieldと同名の別IDへ自動付替えしない。参照するQuery・Rule・CRUD・画面処理・文書・固定Selectionには影響／失効を表示し、削除された列への暗黙保存は拒否する。変更前に依存先をPreviewできるようにするが、Authoringでの削除が**実DB DDL Apply済み**であることは意味しない。

**実DB側への反映は別のEffect**。Workbench／Migrationの既存`Schema Diff→Plan→Apply→独立Read-back`を再利用し、破壊的DDL、対象Alias・Principal・Grant・読取時点と履歴は別資格で確認する。DB側で列の不在が観測されても、TEI定義・Function UI・Owner受理は自動変更せずDriftとして示す。逆にTEI定義を削除しても、実DBはApplyと再観測が済むまで`Target Observed Actual`として旧状態の可能性を保持する。Modelerの表示一致やREA EvidenceはDDL・DB Read・Meaning受理の代替ではない。これは過剰な承認プロセスの導入ではなく、通常開発の変更確認と生成AIの誤推定防止である。

**独立Expectedと偽PASS防止（仕様候補、未実装）**：

| Case | 操作・反例 | 必ず成り立つこと |
| --- | --- | --- |
| REA-MAP-01 | 機能モデルにUI項目だけを追加 | Data Model／Table Definition／実DB列は増えない。UIは未マッピングを表示 |
| REA-MAP-02 | 明示操作でTable DefinitionにFieldを追加してUIへ結合 | 事前Previewと版照合を経て、Field IDとUI Bindingが一致。物理DDLは未適用扱い |
| REA-MAP-03 | Table Definitionの参照中Fieldを削除 | Data Model対応項目を削除、UI自体は存続しBinding解除＋診断。旧FieldへのWrite拒否 |
| REA-MAP-04 | 削除済みFieldと同名の別IDを再作成 | UIの自動再結合なし。Pin Selectionも旧IDを指し、別対象へ黙示変更しない |
| REA-MAP-05 | REA／実DB Reverseのpartial・permission-denied・0件 | 不在を削除と断定せず、Observed差分・unknownを保持 |
| REA-MAP-06 | 同一Field IDのrenameと、別Fieldへのreplace | renameはID基準でBindingを保持、replaceは明示対応付けがなければ未マッピング |
| REA-MAP-07 | TEI定義を削除したが実DBのDROPは未実施・失敗・結果不明 | Definition/Projection更新とDBのActualを区別し、移行成功の偽PASSなし |
| REA-MAP-08 | 旧Revisionで作成したUI/AI編集案をSource変更後に適用 | staleを検出し再Preview／再指定を要求。旧選択の自動付替えなし |
| REA-MAP-09 | 同じ変更をGUIとCLIから提案 | 同じField ID・影響集合・診断意味・効果境界を返す。GUIだけの隠れた更新なし |
| REA-MAP-10 | SHO著者が説明文を編集した後にField削除／REA再解析 | 文書引用はstale表示、著者文の無断書換えはしない。正式Source再解決を別確認 |

**責務と実行順**：定義とData Modelの連動・Modeler投影は#1126/#1124、UI Binding／変更逆写像は機能モデルと#2500、外部観測とREA入場は#3394、実DBとの差分とDDLは#3270/#4125、Source Revision／失効とCLI・AIの固定選択は既存Operation契約、SHOは#4317/#4467の再投影に配賦する。最初の実装出口は`REA-MAP-01/03/04/05`を同じread-only Inspector・編集Preview上で独立Expectedに照らすこと。その後、明示Add/Edit・GUI/CLI一致・実DB Read-back・SHO再読込へ進む。**原典の三要素分析法に忠実な機能と、TEI独自の編集支援を区別する**。本節はPlanであり、実製品E2E・Owner意味受理・DDL適用は**NOT RUN**。

### モデル変更からScript起動までの連鎖を最小化する受入順（2026-11-12、#3394×#1124×#4515×#4502）

**既存二経路を混ぜない。**「外部REA／Workbenchによる観測→候補→TEI定義への明示的採用」は#3394の外部Evidence入場であり、「受理済みTEI Table Definition／Function／UI宣言の編集→参照切れ診断→必要時の再接続」は#1126/#1124の編集経路とする。どちらも`Preview / Change Intent / Source Revision check / Authoritative Source write / Reprojection / Diagnostics`の既存契約の範囲で検討するが、同一Operationで自動的に外部Sourceや実DBへ書き込まない。OKFは文書や根拠のIdentity・Version・Links・Lintを通常のGit／レビュー／試験と併用する。専用Owner意味承認Engine・第二Ledgerは設けない（#4143のK0〜K5は用途別Admissionと独立に検収）。

**実装と機能公開の連鎖を切る。** UIの新規追加・画面変更・REAのFunction候補発見は、テーブルFieldの新設・公開Entry・Trigger/Script起動の作成を**暗黙に実施しない**。Function／Entry／Fieldを削除・rename・移動・版変更した場合は、依存するFunction UI Binding、Script/Entry Input Mapping、Trigger Binding、Query/CRUD、SHO引用のうち各々の**未解決参照を残して診断**し、同名・同offset・同表示位置の別IDへ自動付替えしない。依存先の意味と実体を同定できるrenameはStable Identityと版の検査によって継続を判定し、同定できないreplace/split/mergeは候補提示へ戻す。Triggerは保存・発火しても現在のEntry公開/Host Grant/Artifact資格を示さず、Script Launch P0の実起動Ownerは#4495、manual/schedule/event/batchのbindingは#4497/#4502へ渡す。

**Pinは段階的に適格化する。** #4515の現行Studio Workbenchにおける最初の実装対象は`WorkspaceRef + Subject(kind,id)`の**read-only Pin Intent**。ユーザーの明示Pin/Unpinと一時選択・ハイライトは独立とし、Workspace変更、Source改版、同offset別Subject、A→B→A、後着返信では`needs-requalification/unresolved/unknown`を返す。現在のInspection事実を現在のGrant、実行権限、外部Harness送信許可へ昇格させない。Studioの`parent:<subjectKey>`のような**描画用Edge IDを正式RelationのOccurrence Identityとみなさない**。Relation/CRUDセル/SHO spanのPinは、別Ownerで出典裏付けのある正式Relation識別子を導入・再解決できた後の受入対象とする。Subject Pinの初期実装をGraph全機能の完成に依存させない。

**次の小さな独立Expected（上記REA-MAPの拡張、採用済み・試験済みではない）**：

| Case | 入力・操作 | 期待・拒否 |
| --- | --- | --- |
| REA-X-01 | UI項目を増やす／UI項目のみを削除 | Table Definition・実DB・公開Entryの増減なし。明示したUI変更のみ |
| REA-X-02 | Fieldを明示削除し、同Fieldを複数UI/Query/CRUD/Script Inputが参照 | Data Model写像からFieldを除き、UIは残して未Mapping警告。すべての依存先の参照切れを検出し暗黙Write/Launchを拒否 |
| REA-X-03 | Function/Entryの削除・rename・同名別IDでTriggerを再評価 | Bindingの一意な現在性が確かめられなければunresolved。自動再公開、再実行、別Functionへの付替えなし |
| REA-X-04 | REAの外部Graphに平行EdgeがあるがStudio描画EdgeにしかIDがない | Subject Pinは許容しうるが意味Relation Pinはunsupported。業務Relation受理と誤表示しない |
| REA-X-05 | Subject Pin後にWorkspace切替・Source改版・同offset A→B→A・旧Reply後着 | Pin Intentを維持しつつ現在資格を失効。silent retarget/旧返信の現在表示なし |
| REA-X-06 | GUI・CLI・AIから同じField編集案を要求 | 変更前・後の対象Identity、影響集合、診断意味、Source版、Action/Effect境界が同じ。AI案だけに権限追加なし |
| REA-X-07 | 外部REAのpartial/unknown、またはOKF broken linkを含む文書を読む | 形式の取得成功と「全項目不在」「意味受理」「Current Grant」を混同せず、用途に応じunknown/needs-review |
| REA-X-08 | Triggerが発火したがEntry未公開／Artifact版不一致／現在Grant欠落 | 起動しない。Trigger occurrence・Invocation/Attempt・Effect Receiptを独立観測し、ACK成功を業務成功としない |

**非重複の実装配賦・受入ゲート**：P0-A=#4322からREA Observation 1件＋有向Relation 1件の現main移植（製品API・受領bytesの独立照合）。P0-B=#4515で**Subject Pin**の初回GUI実Consumer（read-only）。両者は**並行**し、互いの未完了をHard dependencyにしない。P1=#1126/#1124で正式Table Definition更新のMapping解除とCLI/Studio診断の同値性（REA-X-01〜03/06）、#4495のpure Function二入口Invokeは別レーンで先行。P2=#3394の外部Graph候補から正式Relation/三要素分析法のモデル・CRUD/SHOへの**選択的**取込、#3270の実DB観測/Apply/Read-back、#4497のTrigger Effect・#4143のOKF K2以降を段階的に結合する。X-TEA Modeler原典に基づく三要素分析法の図面・機能階層・業務モデル・相互参照は#1124/#1125/#1031の忠実性検証を維持し、TEI独自のAI/Pin/Script拡張を原典機能だと記述しない。**本計画は仕様候補であり、実Go/Studio/REA Product E2E、実Provider Read、Owner受理、実DDL/実EffectはNOT RUN。**

### REA Evidence取得経路とStore契約の分離（2026-11-12、#3394×#4307×#4503）

[REA受領PR #4322](https://github.com/hat47x/tei-platform/pull/4322)の現実装は`ExternalCandidateRequest.EvidencePath`を受け`os.ReadFile`してから、`externalanalysis.Receive(raw, tool, at)`と同一Readerで解釈する。これは**保存済みローカル証拠を読む初期Adapter**として有用だが、ローカル／NAS／S3等の任意Evidence Storeの共通入力契約と同一視しない。Evidence実体・内部Locator・digest・件数をGitHubに必ず送る設計にしない（Storeの所有者=#4307、Local CAS実証=#4481、NAS/S3比較=#4494、用途別の取得・固定選択・送出=#4503）。

**最小の接続順**：P0：#4322のReceiver／型付きExternalCandidatesと、版別6.7.0/6.8.0実捕捉Fixtureを**現mainへ最小差分で切り出し**、ローカル保存byteを読み、Observation 1件＋有向Relation 1件を、Coverage/Unknown/Unselectedを保って共通read-only Operationへ投影する。41ファイルを一括マージせず、`externalanalysis`の受領・候補Readと`operations`のProjectionを先に再利用し、CLI・Document・専用CIは小さく後続へ分割する。P1：同じ受領・投影ロジックへ、既存Local CASの実Readerが読み出して**取得元と照合済みの限定byte列**を渡す内部経路を検証。Path専用APIを正本に固定せず、外部Producer形式の第二Parser／巨大な公開raw-bytes Envelopeを増やさない。P2：#4307/#4503の用途別Fetch Plan／選択Scope／送出許可が揃ってから、NAS/S3 Provider、CLI/Studio/Harness独立Consumerを比較。REA側で新しいCAS/Host/Grant Engineを作らない。

**判定に必要な分離**：①Source Revision（外部解析対象のproducer-declared byte版）、②受領Evidence raw digest、③Store objectのlocator/版・実Readback時のdigest、④REA version/protocol/attempt、⑤選択されたObservation/Relationの外部Identityとsubset coverage、⑥TEI Sourceへの候補対応、⑦現在の用途・開示資格。Digest一致は内容同一の証拠であって、Source位置、業務Meaning、Current Grant、全Graph取得、Git HEAD/Dirty入力との一致を保証しない。Pin Intentの維持は開示や取得の許可ではなく、Catalog/Source改版やA→B→Aは再資格化する。

**受入候補（新製品試験はNOT RUN）**：`REA-STORE-01`同一Evidence byteをローカルPathとLocal CAS Readerから受領し、選択Observation/有向Relation・raw digest・Coverageが一致／Locatorは別、`02`同digest別Source Revision／同Evidence ID別raw digestは別Attempt、`03`CAS member欠損・破損／metadataのみ存在をcompleteにしない、`04`Remote Access拒否・offline・partial・tool-absentの区別、`05`Pin対象の現在性／用途別Disclosureなしでは外部AI送出0、`06`1+1の部分投影を全Graphまたは正式TEI Relation/CRUDへ自動受理0。実Store読取り、外部送信、実REA→正式TEI Consumerの縦断試験はこの追記では実施していない。REA実装PR #4322は2026-11-12確認時点で**ahead 10 / behind 254、41ファイル、mergeable=false**。元PRの一括統合を完了条件とせず、現mainからの小分割PRで独立に検収する。

<!-- rea-ref-coverage-20261112 -->
### REA由来参照とTEI参照索引の区別・高速移動（2026-11-12、#3394×#3788×#1124）

**参照の出典と資格を保持する。** REAが発見した静的Relation／同名・近接の構造候補、TEIで受理したData・Function・Businessのモデル参照、実DBのFK／Query観測、実行時の依存は同じ「参照」ではない。既存Source Index／ModelRef／trace／三要素分析法の横断参照を再利用し、表示や検索では`origin=external-observation/candidate/accepted/runtime-observed`（分類候補）、`subject-kind+stable-id`、外部Edge IDと向き・多重性、Source span/revision、確認済み範囲・未解決・必要な開示範囲を区別する。REAのEdge IDや描画用Edge IDを正式TEI Relation IDへ暗黙変換しない。新しい意味正本・Graph Engine・LSPを作らない。

**件数と「未使用」は閉じた範囲でのみ確定する。** 参照数は`confirmed / candidate / unresolved`と検索範囲・Snapshot/Revision・Coverageを併記する。外部REAのpartial、権限不足、未取得Source、動的参照、未観測Pluginが残るときは「確認済み参照0件（未確認あり）」と表示し、「未使用確定」や安全な削除の根拠にしない。Candidateを正式参照数へ足し込まず、欠落をゼロに変換しない。AI向けCUIとStudioでは同じ型付き検索結果・診断・固定Selectionを扱い、引用元・参照先・参照元・SHO上の出典へキーボード主体で移動し、戻る／再移動の履歴を保持する。Pinは現在性・用途別Disclosure・改版で再資格化し、古い結果を最新の画面に上書きしない。

**修正済みの基底と、これから検収する製品経路を分ける。** [#4513](https://github.com/hat47x/tei-platform/pull/4513)（CRUDでのData索引）、[#4534](https://github.com/hat47x/tei-platform/pull/4534)（矛盾の全件報告と構造OK／要レビュー区別）、[#4551](https://github.com/hat47x/tei-platform/pull/4551)（明示CRUD-only組の表示と`UnusedData`）、[#4524](https://github.com/hat47x/tei-platform/pull/4524)（同Kind同ID衝突）、[#4507](https://github.com/hat47x/tei-platform/pull/4507)（Fingerprint入力非破壊）、[#4514](https://github.com/hat47x/tei-platform/pull/4514)（Sourceだけ改版した図面のstale扱い）は2026-11-13確認時点で**main統合済み**。いずれも実装した経路・固定HEADでの限定検証であり、三要素分析法の**全Consumer横断で参照数が正しいこと**や、CRUDのレビュー要否をStudio／SHO／AIが適用停止へつなぐことは未実証。特に`ok=true`は構造診断または投影成功の範囲に限り、`crudConflictStatus=conflicts-require-review`を意味受理・変更許可へ読み替えない。#3788を通常Editorの定義移動・逆引き・安全なrenameのOwner、#1124をModeler投影、#3394を外部REA由来の出典分類、#4503をEvidence Pin/DisclosureのOwnerとする。Source/Indexの増分失効とPluginの遅延読み込みを基本に、参照移動のp50/p95・Index更新範囲・起動時Plugin負荷を実測する。REA入場、#2586の実Function Read、#4515のSubject Pinを参照UI完成でブロックしない。

**独立Expected（基底の一部は修正・単体/CLIで試験済み。REA→参照一覧→Studio/SHO/AIの横断製品試験はNOT RUN）**：

| Case | 反例・操作 | 期待結果 |
| --- | --- | --- |
| REA-REF-01 | 外部REAにRelationだけ存在し、TEI Meaning未受理 | 外部候補として辿れるが、正式Modeler Relation／CRUDの確定参照数に混入しない |
| REA-REF-02 | 同名異ID、同端点の有向平行Edge、逆向きEdge | ID・向き・多重性・出典を保ち、単一Edgeへ縮約せず対象間違いをしない |
| REA-REF-03 | 調査範囲にpartial／permission-denied／動的参照／未観測Pluginがある | confirmed=0でも「未使用確定」を出さず、未解決／範囲不足を併記する |
| REA-REF-04 | 元Source A→B→A、改版、旧応答後着、旧Pinから参照へ移動 | 現在性を再確認し、旧結果のsilent retarget／最新表示への上書きを拒否 |
| REA-REF-05 | 明示CRUDのみ、Data/Business同一ID、同Kind同ID重複、Fingerprint計算前後 | #4472/#4475の負例で件数と名前の取り違えを拒否。Fingerprint計算は元モデル不変 |
| REA-REF-06 | CLI/CUIとStudioから同じ参照元一覧・参照数・依存変更Previewを取得 | 同一Scope/Revision/Grantで結果・unknown/diagnosticが一致。ショートカット操作は追加権限を持たない |
| REA-REF-07 | 大量参照・Plugin未ロード・1 Sourceのみ変更 | 変更影響のある範囲だけ失効・再索引し、不要Pluginの起動を要求しない。実測p50/p95と全件照合の一致を記録 |

**検収順**：P0はmain統合済み#4472/#4475の回帰を固定基線として、#3788の既存Source Indexから一Subjectの参照元・参照先一覧（Scope/unknown付き）を**通常CLIで実受領**し、列挙した個々のRelationと対応する元Source位置を突合する。main統合済み#4548の型付き`ExternalCandidates`は受領可能な候補Operationとして扱い、候補1 Observation＋有向Relation 1件のRead成功を参照索引への正式登録とみなさない。P1は#4515のSubject Pinを利用し、同じQualified Subject・出典・RevisionでStudioの定義移動／逆引き／戻るとCLIの結果を照合する。P2は外部候補と正式参照を**別の参照集計列**で並列表示し、SHOの引用・編集Preview・モデル統計・AIの用途別開示へ接続する。未受理外部Graphの部分取得を「全体把握・全件未使用確定」と呼ばない。

<!-- rea-question-bound-reference-20261113 -->
### 外部REAの参照・未使用判定を「問いに必要な証拠」で検収する（2026-11-13、#3394×#3788×#4472）

**存在の証拠と不存在の証拠は非対称。** REAの同一実Attemptで捕捉したExternal Relation 1件は、「指定Source/版/解析範囲に、指定向き・Edge IDの候補が観測された」という**存在の限定Witness**になる。一方「関連がない」「未使用」「直接Writerがいない」「移行後も同等」は、該当Subject・参照Kind・Source/Dependency・未ロードPlugin・動的到達・必要な業務意味・現在用途について、**その問いに必要な集合の取得と未解決数**が確認できる場合に限って判定する。検索結果の空配列、confirmed=0、Validation.OK、Delivery success、単独digest一致を全称／不存在／完全性の証拠にしない。必要集合を列挙できないときはunknown/not-examinedを明示し、判断不能を正常なゼロに変えない。これは#4527の問い相対の閉包と#4487の三要素横断Evidenceの議論を、REAの参照画面へ適用する提案であり、新たなVerifierやEvidence正本を作らない。

**修正済みの基底と未検収Consumerを区別する。** #4507（Fingerprint入力非破壊）、#4524（同Kind同ID）、#4513（Data索引）、#4534（CRUD矛盾・構造OK/要レビュー）、#4551（明示CRUD-onlyとUnusedData）はmain統合済み。基底の修正済み項目を未修正と記さない。一方、Studio／SHO／AIが明示CRUD-onlyと要レビュー診断を正しく受け取り、変更Previewの停止へ結び付ける製品E2Eは未検収。Function CRUD、業務Action、Subsystem Writer、実Effectを一つのWriter数に畳まない（#3250/#4520）。

**受入候補（独立Expected、実Product試験はNOT RUN）**：

| Case | 入力・反証 | 必須の観測結果 |
| --- | --- | --- |
| REA-QC-01 | REAで1件の外部有向Edgeを観測、TEI正式Relationは未受理 | 指定Attempt・Source版における候補の存在のみ成立。正式参照・CRUD・Current Grantへの昇格0 |
| REA-QC-02 | 一件も参照が見つからないがSource範囲がpartial、Plugin未観測、動的参照が残る | confirmed=0と未探索集合・unknownを別記し「未使用確定」/安全削除を拒否 |
| REA-QC-03 | 明示CRUD UのみでFunction Operations空、または矛盾CRUD U/D複数 | 派生表0行・単一行を「直接更新なし」「矛盾なし」と扱わず、脱落/要レビューを独立表示 |
| REA-QC-04 | 同じEvidence IDとraw digestだがSource現在版／Workspace／Disclosureが変化 | #4548の受領Receipt Pin一致だけでは現在のSource/閲覧・配信・編集資格PASSにしない。必要な再解決で失効またはunknown |

**実装順**：P0はmain統合済み#4547のReceiverと#4548のExternalCandidatesを基線に、#3788で一Subjectの参照／被参照とSource位置を通常CLIから実受領する。#4577/#4584のSubject不存在を参照0件と混同しない修正を別検収。P1は#4515のSubject PinとR3の独立Consumerで同一Candidateの方向・出典・unknownを比較し、#4551/#4534のCRUD/要レビューが実Consumerへ届くか確認。P2で選択された候補だけをModeler/SHOと旧新実Readへ渡す。REAの正式意味受理・実DB/Effect・SHO著者文更新は別資格。**Desktopセルフビルドと#2586の実ReadにR3完了を要求しない。**

### REA候補から三要素分析法・SHO・一機能移行への受渡し（2026-11-13、#3394）

**観測から設計への対応付けは、単なる型変換ではない。** REAの外部Observation／有向Relation（Evidence ID・元raw digest・実Attempt・Producer版・Source位置／申告Revision・Coverage／unknown）と、TEIのData／Function／BusinessのQualified Ref・Source Revisionを**異なるIdentity**として扱う。対応は1対1に限らず、複数Sourceが一機能を実装する場合、同一Sourceが複数機能へ関係する場合、手作業・判断がSourceに現れない場合を許容する。外部Node/Edgeの名称一致やSQL FKだけで業務Key・CRUD・Business Procedureを正式採用しない。採否と根拠・未対応を既存Modelerの候補Previewと通常のSourceレビューに残し、第二のMapping正本／専用承認Engineは設けない（#1124/#1125/#1126/#3394）。

**最初は一機能だけを縦断する。** R1（#4547）保存Evidence Readerのmain統合を基線とし、R2（#4548）はmain統合済み（squash `f843b3a894775f9f82049046d4566f37ded9d09a`）を基線とする。続いてR3で通常CLIと実際に独立したStudio／別Transport Consumerの同一Evidence・Attempt・選択1 Observation＋有向1 Relationを受領検査する。その後のT1-Bでは、外部候補→正式TEI Data／Function／Businessの対応表を**read-onlyのレビュー候補**として比較し、Ownerが受理したSourceだけを既存の編集・Preview・再投影へ渡す。T2では同一FunctionのQuery／Binding／Provider実Read（#2586/#2387）、SHOの引用元・文書Save／Reopen（#4317/#4467）、旧新のLegacy Observed Actual／Owner Accepted Desired／Target Observed Actual（#4125）を**別々のClaim**として照合する。REAの静的解析と旧DB/新DBの実観測、Function実行成功とBusiness完了、SHO著者文と自動生成文、Commit/Receiptを取り違えない。

**追加の独立Expected案（REA-ALIGN-01〜04、未実施）**：

| Case | 固定入力・反例 | 必要な判別 |
| --- | --- | --- |
| REA-ALIGN-01 | 旧ソースの二つの処理が一つの業務Functionを実装し、別の処理は複数Functionに使用される | Source Node数≠業務Function数。外部候補の多対多対応を保持し、重複Function・確定CRUDを自動生成しない |
| REA-ALIGN-02 | 同名別IDのData／Function候補、同端点の平行Edge、Source位置が曖昧な関係 | 対応候補と未対応を列挙するが、一件へ後勝ち確定せず、正式Sourceへの適用を停止 |
| REA-ALIGN-03 | Sourceに現れない手作業承認と、コード上のTask起動／Function成功が同時に観測される | Business Procedure完了や承認済みを推定しない。手作業・業務Ruleの欠落をOwnerレビューに残す |
| REA-ALIGN-04 | 同名の旧新Queryで件数は一致するが金額差／NULL差がある。別時点に元SourceがA→B→Aと変化 | 独立Expectedで意味差をFAIL／unknownとし、古いEvidence Pin・SHO出典・対応付けを現在資格へ自動復活させない |

**所有・優先順位**：外部受領とR3は#3394、三要素Modeler／原典忠実性は#1124/#1031/#1125、Relation/Keyは#1126、参照移動は#3788、Query/実Providerは#2586/#2387、SHOは#4317/#4467、旧新移行検収は#4125、独立ExpectedとAttemptは#2514/#1180/#1163、現在開示は#4503/#2516で引き受ける。新Parser・Graph Registry・Host・Owner承認エンジンは作らない。本節は**docs-onlyの設計・反証計画であり、REA-ALIGNの実試験、正式Meaning受理、製品E2E、旧新DB適用はNOT RUN**。Studio Desktop実Wails接続→TEIセルフビルド（#1824/#2831）と一Function実Provider Read（#2586）は、この縦断検証を前提条件とせず並行優先する。

<!-- rea-one-function-witness-20261113 -->
#### 一機能を端から端まで確かめるための検収単位（設計案・未実行）

**検証する問いを先に固定する。** 最初の問いは「この一つのFunctionが、どのDataを、どの条件・時点・権限で取得し、業務手順のどこから利用されるか」とする。検証票に `Question / Required Sources & Dependencies / Observed / Candidate / Accepted / Actual Read / Unknown & Unexplored` を記録する。REAが返したNode数やEdge数を必要証拠の母集団とはしない。問いに必要な業務手順・Function定義・Data/Key/Relation・実消費SQL/Binding・Plugin/外部依存をまず列挙し、Sourceが無い／読めない／部分的／権限で隠れる／動的経路が未知を分ける。全件取得を証明できない場合、「参照なし」「未使用」「Business手順なし」は確定しない（#4527/#3262/#3788）。

**一機能の小さな縦断**：①REAの実Attempt・元Source版・raw digest・Producer/Provider・Observation/有向Relation ID・Coverage/Unselectedを、#4547と#4548の既存Reader/Operationで固定選択する。②対応候補は外部IDとTEI Qualified Refを別列に置き、`0..N対0..N`、対応理由・反例・保留・参照位置を含む**read-onlyレビュー表**として#1124/#1125へ渡す。TaskActionの存在≠Function呼出し、ProcedureのManual Step≠欠けたFunction、同名のData≠同一業務Keyとする。③明示的に受理した項目のみ既存TEI Sourceへ編集Preview→Save→再Readし、差分・診断・逆引き参照を検収する。AIの選択、SHOへの引用挿入、REAの候補生成は正式Sourceの自動更新を許可しない。④Query/Provider実Readと旧新比較は同一のFunction・引数・読取Scope・業務Key/粒度・NULL/Decimal/順序・観測時点を固定し、**旧実測／受理済み期待値／新実測を別試行で比較**する。#2586のmain統合済み#4539は限定された実PG Read→Contextの基底であり、旧実行・通常製品CLI/Studioの同一受領・意味同値性を証明しない。⑤SHOはSourceに束縛した引用と著者文を分け、Save/別Open時の参照・未解決・Source改版を確認する。正式モデル受理、SHO保存、DB Read、実Effect/Receiptの合格を合算しない（#4317/#4467/#4125）。

**独立Expectedの追加候補（REA-HO-01〜05、すべてNOT RUN）**：

| Case | 固定入力と故障 | 期待する停止線・観測 |
| --- | --- | --- |
| REA-HO-01 | REAは外部Relation 1本を返すが、必要な別Source・Plugin・動的呼出しを未探索 | その1本の存在だけを観測済みにし、問いへの必要閉包と「参照なし／完了」をunknownに保持 |
| REA-HO-02 | X-TEAのTaskActionにFunction参照がない手作業、別Actionには呼出し候補だけある | 業務Stepは保持し、実Function起動・承認完了・業務完了を推定しない。原典とTEI独自拡張の差を明示 |
| REA-HO-03 | 同名別IDの外部Nodeが正式Dataと誤連結される、または候補RelationがSource編集後に古くなる | 既存Qualified Ref/Source revisionを再解決し、曖昧対応・古いPinはPreview/Saveへ昇格させない |
| REA-HO-04 | 新旧Functionは件数一致だがTenant別Key・NULL/金額・有効時点・ソート同値順位が異なる | 独立ExpectedでFAIL/unknown。Query/Provider選択や生成ログだけでは同値としない |
| REA-HO-05 | SHOを保存後にSourceが改版、または外部Evidenceは同digestでも現在の開示権限が変化 | 著者文・元引用・Source参照の資格を別に再確認。旧Receipt・候補・Pinから自動再受理／外部送出しない |

<!-- rea-product-locator-verdict-20261113 -->
#### 正式参照への接続前に防ぐ二つの偽PASS（2026-11-13、実証資格の追補）

**対象の存在と参照の有無は別の問い。** 修正前mainの`tei source-index -subject-kind ... -subject-id ...`では、改名後の不在Subjectにも`ok:true`・空`references`・exit 0を返す反例（#4577）が実測され、[修正PR #4584](https://github.com/hat47x/tei-platform/pull/4584)で既存`InspectionFromLoad`を使った正規Subject存在確認が提案・実装されている。REA候補からTEIのQualified Refへ対応付けるとき、**まず同一Workspace/Source版におけるCanonical Subjectの一意な存在を確認し、次にそのSubjectに紐づくSource引用・参照関係を検索する**。不在・Unsupported Kind・存在するが参照0・Index取得失敗・partialを別状態としてConsumerへ運び、空配列から存在・不存在・未使用を推論しない。#4584はgofmt修正後の固定HEAD `dcfe1d69`で8件のActionsがすべてcompleted/successとなり、mainへsquash統合（`f3b58e7b580f0ecbea1d2feb156f52c2147f63a7`）。**現在のmainではCanonical Subject不存在を`SUBJECT_NOT_FOUND`とするCLI経路が実装済み**。外部REA候補の正式対応・現在のSource・全参照閉包の検証とは独立に扱い、Consumer実受領は#3788/#3394で検収する。

**Studio三視点のserver成功は外部REAのDesktop入場ではない。** [Draft #4570](https://github.com/hat47x/tei-platform/pull/4570)は共通操作→X-TEA固有Plugin→実選択されたGo Provider→Studio Bridge／server画面までを合成Workspaceで局所実証し、Source位置・`occurrence`・固定Pin・CRUDを保持する。ただし`occurrence`はIdentityではなく、外部REA Node/Relation IDともTEI Qualified Refとも別。#4570の固定HEAD Actionsは11件中9件success・Repository Health/Wails Host 2件failureで、実Windows Desktop・実X-TEA原典対象の適合は未検証。#4548の12 Workflow成功や別HEADの生成Inspector/Desktop #4572/#4587の成功を加算し、「同一のREA Candidateが実Desktopで表示・再選択・再受領された」とは判定しない。R3は`operations.ExternalCandidates`の**実同一Attempt**を通常CLIと独立Consumerで受領し、それぞれのApplication API route／Selected・Invoked・Received／Consumer digest・固定対象・Coverage/Unknownを確認する。三視点Pluginはその後に正式モデルのread-only照合へ接続する。#4548のR2自体はmain統合済み（`f843b3a894775f9f82049046d4566f37ded9d09a`）。

**REAという名前の区別**：外部逆解析ツールREAと、会計・業務分析法としてのResource–Event–Agent（資源・経済事象・経済主体）は別概念である。外部解析ツールが返す`Agent`や`Relation`を会計上のAgent／Event、TEI業務Actor、実Principal/Grantへ型名一致で変換しない。Resource–Event–Agentを扱う場合も、業務上の意味を検討するための任意Projectionとして、三要素分析法のFunction/Business・Source観測との対応をレビューする。新しいCanonical型・標準DSL・共通承認機構をここで採用しない（#1124/#2386/#3394）。

| 追加反例（設計のみ） | 故障入力 | 必須の判別 |
| --- | --- | --- |
| REA-HO-06 | 旧Qualified RefのSource宣言は改名で消えたが`source-index`が空の参照配列で成功する。REA候補の名前は一致する | #4577/#4584の実CLI経路で`SUBJECT_NOT_FOUND`を確認。候補の一件採用・参照0＝未使用・Source更新を拒否。全件Index空とは分ける |
| REA-HO-07 | REA R2契約CI、三視点server、生成Inspector Desktopがそれぞれ別HEADで成功する一方、三視点Wails/Repository Healthは失敗する | ClaimごとにTested HEAD/Transport/Source/Consumer/受領byteを照合。異なるHEADのPASSを合算せず、外部REA→実Desktop Candidate受領はNOT RUNに維持 |

**採用と検証の優先順位**：main統合済み#4584のSource Subject存否修正を#3788の通常参照操作で利用・回帰検収すること、mainに入った#4548のR2を実Consumerへ接続すること、#3394でR3の実独立Consumerを測ることを**並列の小作業**とする。#4570の三視点Pluginは実Wails/原典Sourceの検証を別Owner #1124で進める。保存Evidenceの現在開示は#4503、実Query→Contextは#2586の既存経路で継続し、REA R3や全Modeler完成でDesktopセルフビルドを止めない。REA-HO-06/07の製品縦断試験はこの追補ではNOT RUN。

**実施順と既存Owner**：R2 #4548 main統合済みの型付きOperation→R3 #3394で通常CLI＋独立Consumer受領→#1124/#1125で一件のread-only対応Preview・Ownerによる通常Source編集→#2586/#2387と#2514/#1180で固定Functionの実Read・旧新独立Expected→SHO側の保存/再読取。保存Evidenceの実体と外部開示は#4307/#4503、実行・権限・Effectは#1174、Schema/移行切替は#4125へ委ねる。REA 6.9.0の選択読取・欠落リンク・ページングの版別検収は#3394/#4561に残し、#4257の6.7/6.8合格を継承しない。**ここでは追加のEngine/Registry/正式DSL/承認システムも、Desktopセルフビルドや#2586を止める新Gateも導入しない。**

<!-- rea-r3-delivery-qualification-consolidated-20261113 -->
#### R3の最小Product Exitと外部Evidence取得の停止線（設計、実E2E未実行）

**最初の製品入口はread-only一件に限定する。** R2に統合した`operations.ExternalCandidates`から同一保存EvidenceのObservation 1件・向き/Relation IDを保持したRelation 1件を選択し、通常`tei` CLIと独立したStudio（または共通APIを呼ぶ別Transport）の二経路で、`operation schema/version・Producer Tool/Version・Evidence ID・Raw digest・申告Source Revision・選択ID・Unselected・Coverage・Unknown`を実受信して照合する。二経路は`selected→invoked→received`をそれぞれ記録し、同一Go関数の出力を二度表示するだけの試験を独立Consumerとしない。必要な証拠の集合を満たさなければ、1+1の成功でも全Graph完全や未使用確定としない。既存REA-R3-01〜06/REA-REF/REA-QCを試験Owner #3394/#2514/#1180へ使い回し、新しいGateは作らない。

**配送境界を先に制約する。** 現R2は`EvidencePath`を受けて`os.ReadFile`する保存済みローカルファイル用のread-only Operationであり、任意の外部呼出し元へそのパスをそのまま公開するAPIではない。Studio/Remote/Harness側からは、Hostが現在のWorkspace Root/許可範囲で解決・検査した固定Evidenceの選択情報だけを受け取り、内部CAS/NAS/S3の実Locatorや秘密Sourceを出力しないことを検証する。保存byteを読み取れたこと、現在閲覧権限、用途別送出許可、実際の送達Receiptは別Claim（#4307/#4503/#2516）。`ExpectedEvidenceID/ExpectedReceivedDigest/ExpectedSourceRevision`は**受領されたEvidenceの照合**であり、元Source現在版・ABA・Workspace帰属・Disclosure/Grantの代わりにならない。API契約の不足は#3419へFit/Gapとして返し、R2の型に未審査の公開必須フィールドを追加しない。

**Tool版の資格は不連続とする。** #4257は実REA 6.7.0単独JSONと6.8.0のJSONL終端を対象にした限定実証。外部REA 6.9.0の選択読取／リンク探索／ページングは新しいT0-06の固定Tool＋Source＋Scopeで、終端・未取得ページ・Relation Identity・Coverage/Unknown・Consumer受領を別に検収する（#3394/#4561）。既存Evidenceを再利用できても、解析版更新や必要なSource閉包・Provider実消費・完全性を自動認定しない。

<!-- rea-r3-executable-slices-20261113 -->
**R3実装は3つの小分割で進める（Product検証は未実行）。**
1. **R3-A／ローカルCLI実入口**：通常`tei`の既存Command一覧・dispatch・read-only発見契約・flag grammar・JSON envelope・exit規則に`ExternalCandidates`を接続する。既存R2 Operationを直接呼び、別ParserやJSONL Readerを作らない。最初はローカルで明示選択した保存Evidenceだけを扱い、Source位置・ID/向き・原raw digest・Tool版・unknownを受領する。UI/remoteの汎用Path入力へこの入口を転用しない。必要なEvidence locator解決は#4503の既存Store/Scopeを使い、未実装のHostによる認可を合格扱いにしない。単一CLI実プロセスの`argv→selected→invoked→stdout/exit`と、存在しないEvidence／曖昧ID／不一致Pin／不完全JSONL／Scope未成立を独立Expectedで検証する。
2. **R3-B／独立第二Consumer**：既存の固定Workspace MCP read-only AdapterかStudio Bridgeのうち**既存Operationを安全に呼べる一経路だけ**を選ぶ。選択した外部IDはHost管理の許可済みEvidence参照として受け、任意ファイルPath・内部CAS/NAS/S3 locator・認可前の保存byteをTransportに露出させない。ReadOnly Scopeと用途別Disclosureの拒否は#4503/#2516の責務を再利用する。Studioに既存三視点の全実装や実Wailsを必須としない。
3. **R3-C／同一Attemptの二Consumer独立検収**：R3-A/Bそれぞれの実起動・実呼出し・受領byteを記録し、`EvidenceID / raw receivedDigest / producer.SourceRevision / selected Observation+Relation ID / direction / unselected / coverage / unknown`の一致と差分の理由を照合。正例の比較対象は同じ保存Evidenceでも**別Consumerの実受信結果**でなければならない。Source A→B→A、申告Source版と現在版の相違、未探索依存を検出できないときは現在資格を`unknown`に止める。Callerが二回`ExternalCandidates`を直接呼んだだけのGoテストやCLI stdout再表示はR3-CのPASSに数えない（#2514/#1180/#1163）。

**独立失敗ケース**：CLIで選べた保存ファイルのPathをRemote引数に転用する（拒否）、同一Evidence IDでraw digest違い（停止）、存在するCanonical Subjectで参照0件（未使用の全域確定を拒否）、存在しないSubjectを外部REAラベルで再束縛（#4584/4577の負例）、Producer 6.9.0の未取得ページ（partial/unknown）、成功した別HEADのCLIとStudio表示を合算（非PASS）。**この3分割はコードをまだ変更していない実装手順であり、R3 Product E2E/現在Grant/業務意味受理はNOT RUN**。Ownerは#3394、CLI API #3419、Store/Disclosure #4503/#2516、Studio #1124、Expected #2514/#1180。TEI Desktopセルフビルドと実Provider Readは別P0で維持する。

<!-- rea-r3-path-leak-and-atomic-read-20261113 -->
**R3-Bの安全境界を、ファイル名・エラー・取得競合まで検収する。** 現mainのR2 `operations.ExternalCandidates` は `EvidencePath` を `os.ReadFile` し、読取失敗を `EXTERNAL_CANDIDATE_READ_FAILED` として返す際、低位の`err.Error()`をメッセージへ含める。これはローカルCLIでの診断には役立つが、内部絶対PathやCAS/NAS/S3 Locatorを含む可能性があり、**そのError文字列をStudio/MCP/Remoteへそのまま公開しない**。R3-BのHost管理境界で既存の用途別Disclosure／エラー投影を再利用し、外部には安定Error Code・許可された診断だけを返す。入力`../`・絶対Path・シンボリックリンクによるRoot逸脱、存在しないPathを使った列挙、Root検査後の対象差替え、Scope変更を別々の負例として扱う。安全な解決と取得を単に`Clean`/文字列prefix比較で実現したと主張せず、許可済み保存Evidenceを固定して読み、**実際に読んだbyteのdigest**と選択したEvidence/Attemptを照合する。Readerの`ReadAt`は呼出し時刻であり、元Sourceの現在性やStore上の原子的スナップショットを証明しない。既存Store/Disclosure #4307/#4503が必要な固定取得能力を持たない場合は、R3-Bを無理に実装せずFit/GapをOwnerへ返す。

**追加の独立Expected（REA-R3-SEC-01〜04、未実行）**：①内部絶対Pathを含む`os.ReadFile`失敗がRemote診断に漏れない、②未許可Root／symlink／`../`が外部Evidenceへ到達しない、③Pathの検査とReadの間に実体が差し替わっても旧Pinで成功しない、④受領済みbyte digestが一致してもCurrent Grant／現在Sourceを勝手に認定しない。R3-AローカルCLIの詳細診断と、R3-B外部開示の診断は同一である必要はない。R3-Cでは**同じファイル名**ではなく、実Consumerが受領した同じAttempt/byteを比較する。新しい汎用ファイルサーバー・Grant Engineは設けない。

<!-- rea-r3-specific-ci-blockers-20261113 -->
**関連PRの具体的な失敗理由を追跡する。** [Source-index修正 #4584](https://github.com/hat47x/tei-platform/pull/4584)の旧HEAD `e1f589f` は8 Workflow中7 successで、Repository Healthの実ログが`gofmt drift in: cmd/tei/source_index_test.go`だった。後続HEAD `dcfe1d69`で整形を修正し、同HEADの8 Workflow全件successを確認したうえでmainへsquash統合（`f3b58e7b580f0ecbea1d2feb156f52c2147f63a7`）。**旧失敗は解消済み**であり、別Consumerでの正規Subject存在確認とREA Candidate対応付けは未検収。[三視点Plugin #4570](https://github.com/hat47x/tei-platform/pull/4570)の固定HEAD `4d89ad0` は、Repository Architectureの`tei/plugins/xtea-three-view -> tei/internal/xeadsource`という**正規Product Graphから旧Source Adapterへの直接依存**が拒否され、生成Hostの`TestGeneratedGoServicesRunThroughRealHost`は`unexpected Wails exports: 16`で失敗した。PluginをCoreへ埋め込む／禁止依存を許可リストへ追加するのではなく、既存Adapter境界で依存方向を修正し、公開Bridge契約と生成Hostの両経路を同じHEADで再検査する（#1124/#1824）。R3外部Candidate受領はこの三視点Pluginの修復をHard prerequisiteとせず、既存共通APIの最小別Consumerで先行可能とする。**失敗Jobの実ログは原因特定の証拠だが、対策の実装・再テストはNOT RUN。**

<!-- rea-r3-crosslane-three-aspect-20261114 -->
#### REA R3から三要素・Graph・SHO・旧新比較へ渡す統合契約（2026-11-14、設計候補・新Product E2E未実施）

**名称と採用境界。** ここでのREAは外部逆解析ツール **Reverse Engineer Anything** であり、会計・業務意味モデルのResource–Event–Agentとは別。REA Evidence中の`Agent`らしい名前を業務Actor、認証Principal、承認権限へ自動対応させない。REAのSource観測、AIによる構造候補、Ownerが受理したTEI Meaning、実Runtime／DBのObservationは別Claimであり、同じ表示名・行数・Graph形状を理由に昇格しない。主線はR3（共通Operationの実Consumer）→T1-B（一件の対応候補Previewと通常Source編集）→T2（一機能L/D/T）で、Studio Desktopのセルフビルド#1824/#2831および実Provider Read #2586とは**並列**に進める。

**既存モデルのまま保持する同一性。** REA由来`EvidenceID/Producer Attempt/raw receivedDigest/申告SourceRevision/GraphID/NodeID/RelationID/EvidenceContextID`と、TEI側`Workspace/Scope/Qualified Ref/Model Source Revision/Occurrence`、実DBの`Business Key/Physical PK`、実Consumerの`Invocation/Receipt`はそれぞれ別のIdentity。外部Relationの`from/to/direction/kind`と平行辺の個別IDを保持し、同端点・同labelの重複を集約しない。ProducerがRelation IDを持たないderived handleは衝突可能であり、現在のR2曖昧性拒否を維持する。Graph描画Edge ID・座標・行番号・外部Spanを正式Relation ID／TEI検証済みbyte rangeへ昇格させない（#2497/#1124/#1126/#4515）。対応は**多対多かつunresolvedを許すread-only候補**で、正式Source変更はOwnerの既存Shared Acceptance・Revision check・再投影に委譲する。

**三要素の解釈は「3種類のRoleを揃えれば原典同等」ではない。** [三要素・三層の設計候補 #4487](https://github.com/hat47x/tei-platform/pull/4487)と原典忠実性のOwner #1124/#1125/#1126/#1031 をつなぎ、Data/Function/Business ×「論理要件／業務・物理的制約／実装」を**問い相対の投影**として比較する（9種類の正本・Core enumは追加しない）。物理制約はDB SchemaやIT配置だけでなく、組織・人員・時間・社会的条件を含み得る。下表は設計例であり、REAが自動的に9枠を充足するという保証ではない。

| 三要素 | 論理要件（Owner受理待ち） | 業務・物理的制約（別根拠） | 実装・観測（REA等） |
|---|---|---|---|
| Data | 業務対象・同一性・粒度・有効時間 | 原記録・保存義務・取得可能範囲 | Table/Field/Query、外部Graph候補 |
| Function | 判定・計算・入力・結果 | 操作環境・時間制約・利用可能資源 | 画面・メソッド・SQL・呼出関係 |
| Business | 職務・手順・承認・例外 | 人員・委任・組織・社会的条件 | Task/Trigger/通知／一部実行Trace |

**CRUD/Effectの境界。** Function宣言のC/R/U/D、分析上の派生Potential、Business Action／手作業、実Provider/DBのActual Effect、業務上の完了／承認を別列で保持する。旧実装の2 method→1 Function、1 method→複数Function、Sourceに現れない人手Action、既存Businessの時間付き責任移譲を表現対象に含める。Event付きDFD・職務・Action Tree・Subsystem別ER/CRUDを原典に照らして検収し、Task起動・ACK・成功exit・通知だけで承認や業務完了を作らない。REＡの静的CUDは実更新Receiptではない（#1125/#1129/#3250/#3278）。

**必要Source閉包と量化。** 参照1件は当該Attempt・Scopeの存在Witnessだが、参照0件や「全Writer不存在」を主張するには、質問ごとにRequired Source集合、Dynamic call/未ロードPlugin、解析Graph Family・未取得ページ、権限不足、Current Source Revisionを明示し、必要閉包を満たす必要がある。形式上の配送complete、Graph coverage complete、問合せ相対のSource closure completeを別列にし、`Unknown/partial/not-found/permission-denied`を空集合へ縮約しない。TEA-Q09〜14（#4487）とREA-QC/REA-HO（#3394）のExpectedを重複新設せず参照する。

**R3の実装に返す具体的Fit/Gap。** 現mainの`operations.ExternalCandidates`は`EvidencePath`→`os.ReadFile`→`Receive/ReadCandidates`で、内部エラー文にPathが含まれ得る。R3-AローカルCLIは明示Path入口を既存CLI規約で限定し、R3-BはHostが許可した不透明Evidence selectorから安全に**実取得したbyte**を使い、同じ受領・投影ロジックへ渡す小さな内部責務分割を検討する（正式bytes入力APIを採用したことにはしない）。`ExpectedEvidenceID/ExpectedReceivedDigest/ExpectedSourceRevision`の一致は過去の受領byteの固定であり、現在Source/Grant/開示や原子的Source閉包を証明しない。Root逸脱・symlink・検査後差替え、内部Locator/診断漏洩は既存REA-R3-SEC-01〜04へ接続。Producerの解析AttemptとCLI/Studioの別Consumer Invocationは同一イベントではないので、Producer Evidenceを共通に比較し、ConsumerのSelected/Invoked/Receivedは**それぞれ独立して**記録する（#3419/#4503/#2516/#1163）。

**T2の比較単位。** 同一Functionに対しL=旧システムの実Observed Actual、D=Ownerが受理したDesired、T=TEI Targetの実Observed Actualを別Source/時点・実Provider・Consumerと結ぶ。最初はRead-only Queryの`Business Key/PK/複合Key・NULLと欠損・小数精度・並び順/同順位・権限/時間条件・Rowset`を独立Expectedで検証する。件数一致やJSON一致、TEI生成物同士の相互比較はDの独立証明にならない。後続のCommit/Receipt/RecoveryとSHO Save/Reopen・著者文保護は#4125/#1174/#4317/#4467へ分ける。既存`order_entry.recent_orders`の実Provider ReadをP0の先行証拠として尊重するが、旧`order_entry.submit_order`／TEI`sales.order.submit`の業務対応・更新移植が受理済みとは扱わない（#2586/#2387/#2514）。

**横断Expectedの差分（既存#2514/#1180/#1163の試験票候補、NOT RUN）**：
- REA-XLAN-01：外部同名別Node・同端点平行Edge・逆Edge。Producer/TEI双方のIdentityと方向を保持、Graph描画用IDへ置換しない。
- REA-XLAN-02：REAが実装Methodのみ確認、手作業承認と組織の時間条件はSource外。三要素の未観測セルをunresolvedに残し、業務完了を推定しない。
- REA-XLAN-03：Graph配送はcompleteだが対象Functionの動的依存／Plugin未ロード／未取得ページがある。参照0でも「未使用」「安全削除」を拒否。
- REA-XLAN-04：Function-level CRUD U宣言とBusiness Actionの更新なし／実DB Effect不明が並存。Declared/Potential/Actualを分離し、業務Actionへ偽のUを付けない。
- REA-XLAN-05：同じEvidence raw digest、旧Model A→B→A、Current Grant変更。受領Pinは一致しても旧Meaning・旧開示・旧Studio Pinを現在資格へ再昇格しない。
- REA-XLAN-06：CLIと独立Studio/MCPが同一Producer Evidenceを別Invocationで受領。選択・実呼出し・受領を両経路で別証明し、単一stdout二回表示は非PASS。
- REA-XLAN-07：旧新Rowset件数一致だが複合Key・NULL/欠損・金額精度・有効時点に差。L/D/Tで差を検出し、許容差はOwnerの独立Expectedに従う。
- REA-XLAN-08：Source改版後にSHO著者文・図面・参照が残る。引用のstaleを表示し、著者編集を自動上書きしない。SUI/AIへの共有は用途別Disclosureを再評価。
- REA-XLAN-09：Studio Stage1 Generatorが必要だがStage2実行物には不要。Pluginの役割とBuild Input/Artifact/Runtime Providerを区別し、REA連携の追加をStage2セルフビルドHard Gateにしない（#4601/#4583）。

**Owner・停止線。** R3のProduct出口と外部byte受領は#3394、共通操作/Transportは#3419、Store/開示は#4503/#2516、原典三要素/Graph/編集は#1124/#1125/#1126/#2497、SHOは#4317/#4467、試験は#2514/#1180、DB実Readは#2586、Studioセルフビルドは#1824/#2831。#4487/#4570/#4601等のDraft成果は候補として参照し、実Owner判断前に正式要件/合格へ昇格しない。この節は**docs-onlyの横断受入設計**。R3の実CLI・第二Consumer E2E、REA 6.9のT0-06、原典Source/三要素の正式受理、実Wailsセルフビルド、SHO/旧新DBは本変更ではNOT RUN／NOT ACCEPTED。

<!-- rea-question-bound-join-producer-support-20261114 -->
#### 問い相対のEvidence Joinと三要素の支持範囲：R3後段に追加すべき最小受入条件（2026-11-14、設計のみ）

**他レーンからの現行差分**：[三要素Diagram Draft #4487](https://github.com/hat47x/tei-platform/pull/4487) §38〜39は、局所Projectionのcompleteから「図中になければ存在しない」へ飛躍しない判定と、**TEA-Q41（REA Data支持からF/B不存在を推定しない）／Q43（Legacy Function宣言と新PG Read成功を同一Function実行としない）**の二つを優先的な独立Expectedとして定めた。本REA R3では既存#3394/REA-HO/REA-XLANの受領証拠を提供し、この二ケースを**既存三要素/実Read OwnerのConsumerで確認**する。新しいREA Parser、三要素専用Truth Store、9正本、共通Proof Engine、製品Gateは作らない。

**最小の問い相対Review票（既存Operation応答と既存Evidence/Source/Disclosureのread-only合成、正式APIではない）**：
| 確認列 | 例・検証点 | 欠落時の扱い |
| --- | --- | --- |
| Question/Decision use-site | 「外部の参照を1件表示」「正式モデルに同一Functionがある」「安全に未使用削除できる」 | 問いなしでglobal complete・一般的に安全とは言わない |
| Producer/Claim class | REA外部Observation、X-TEA `.xead`宣言、TEI Accepted Model、実PG Read、手作業Actualを区別 | 別ProducerのClaimへ黙示昇格しない |
| Subject identity/Scope | External Evidence+Node/Edge ID、TEI Qualified Ref、Workspace/Consumer/Audience | 同名・同bytes別Workspaceだけで結ばない |
| Attempt/Revision/Source closure | Producer raw digest、Source宣言版・Reader実Consumed Source集合、.xeaf/他Source・dynamic Plugin・欠落ページ | `unobserved/unsupported/partial`を「0件」にしない |
| Relation semantics/Grain | 向き/多重辺、Function-level CRUDとBusiness Action、表のBusiness Key/NULL・時間 | 描画Edge ID/CRUD集計数から実Writeや意味同値を主張しない |
| Disclosure/Delivery | 現Audienceで許可された公開Slice、CLI/Studioの別Invocationと受領証跡 | Path/Private件数・存在そのものを漏らさない |
| Assessment ceiling | observed candidate／unresolved／review-required／accepted meaning（既存判断記録を参照）／actual observed | 未資格の強い判断は`unknown/not-qualified`。新Core enum化しない |

**Producer支持範囲を行列で保存し、分母を混ぜない。** 現mainのR2 `operations.ExternalCandidateView`が運ぶ`GraphCoverage/Unknowns/Unselected`は外部解析の局所情報であり、**Data/Function/Business別の完成状態そのものを保証する型ではない**。#4487の`BuildReviewPacket.ThreeElement`等、別途Producerが持つ三要素評価をR2の実装済みフィールドと偽らない。REAがData候補1件、Graph配送completeであってもFunction/Businessの評価が`unsupported`なら「0件」「無い」「全て確認済み」と描かない。X-TEA `.xead`単一Sourceの構造read-only completeも別`.xead`/`.xeaf`/手作業/実Providerのabsenceには使わない。実PGのRowset成功はTarget実Readだけであり旧Function実行の証拠ではない。組織の担当候補・Business TaskAction宣言も、人間による実遂行・承認・Current Grantの証明ではない。

**三段階判定（P/J/Q）**：
1. **P: 局所Fact/Projection** — 一つのEvidenceからObservation/Edge/Source occurrence・Graph Familyのカバレッジと限界を正しく読めたか。対象の存在Witnessは限定的に得られる。
2. **J: 異種証拠のJoin** — 同じQuestion、対象Scope・版・出典、適用Mapping・異なるReader Attempt・業務有効時点と必要なCurrent Disclosureを明示的に再照合できるか。同一Digest／同一表示名・等しいJSON／同時刻だけではJoinを受理しない。
3. **Q: 問い相対の必要閉包** — 否定・全称・「安全に削除」等の主張には、Ownerが確定した`Required(Q)`と取得済み`Observed(Q)`の適格な一致、Relation family/未ロードPlugin/動的参照/Manual step/許可Scopeの十分性が要る。成立しなければ、**確認済み0件＋判断不能**であり「不存在確定」ではない。Private Sourceの存在の有無も公開回答から推測させない（#2516）。

**最短の縦切り：R3-A/B/Cの製品出口を増やさず、Consumerの二つの反例へ引渡す。**
- **TEA-Q41を先行**：固定の公開合成EvidenceからREA候補Node 1件＋Directed Relation 1件を既存R3 Operationで受領し、外部Graphの`complete/partial`と三要素別の`unsupported`を別出所として持つ**Consumer Review票**を作る。未対応のFunction/Businessを0件として表示したらFAIL。R2自体に存在しないFieldを追加済みと主張しない。#4487/#3394/#1124/#2516の既存Expected/Consumerを再利用。
- **TEA-Q43を次に**：X-TEA SourceのFunction宣言（R等）と、`order_entry.recent_orders`等の独立PG Target実Readを同じ表示名で提示しても、Legacy→TargetのQualified Mapping/Source版/実Provider・受領Rowset/業務Owner Expectedがない限り「旧Functionの実行成功」「同値」「移行済み」としない。実旧システム観測がなければL（Legacy Actual）を欠損/unknownのまま残す。#4570/#4615/#2586/#4191/#2514へ引渡す。
- **実Studioの表示へ**：#4570/`three-view-navigation/v0`の`coverage.status=partial`に`omittedRelations`だけがあるとき、画面で3つの0件内訳しか示さない既知Gapは#4613（CV-01〜04）に既に引渡し済み。表示上の`complete`を原典Source全要素、未読込Sourceや九観点の完全性に拡張しない。こちらでは重複実装せず、R3候補Consumerの出力でPartial理由を必ず保持する。
- **匿名化・不開示の反例**：Private側だけが違う二世界で、同じAudienceに返す公開候補・件数/unknown内訳・エラー理由/locator・公開Digest/JSONを比較し、非公開要素の存在を表示差から推測させない。Private取得後の単純redactは原則使わず、初めから許可済みSliceで問を構成する。#2516のTEA-Q45に再利用し、新しいAuthZ実装を作らない。

**受入の分担と優先度**：R3-A/Cの実Call/二Consumer受領は#3394/#3419、Remote Host固定取得と情報境界は#4503/#2516、原典/三要素支持範囲・UI表示は#4487/#1124/#4613/#4615、旧新Meaning/Target移行は#4191/#2586/#4125、独立Expectedは#2514/#1180。**最優先P0**のTEI Studioセルフビルド#1824/#2831と実Function Provider Read#2586をR3-Jや9観点の完成でブロックしない。#4487/#4570は未統合Draftであり現main実装済みとはしない。この節は**docs-only**で、新Go/Studio/Windows/PG、REA 6.9.0、二Consumer受領、現Grant/外部開示、旧新移行・Owner Meaningの新しい検証は**NOT RUN／NOT ACCEPTED**。

<!-- rea-r3a-product-branch-and-d0d3-20261114 -->
#### R3-A通常CLIの実装Draftと、データ様式D0〜D3への非阻害接合（2026-11-14）

**Product上の新しい事実はDraft実装に限定する。** [R3-A実CLI Draft #4625](https://github.com/hat47x/tei-platform/pull/4625) を現mainから独立ブランチとして起票。通常`tei external-candidates`へmain R2 `operations.ExternalCandidates`を結線し、`cmd/tei/cli_external_candidates.go`、実子プロセス検証`cli_external_candidates_test.go`、`main.go` dispatch、`cli_commands.go`、`cli_contract.go`、`cli_flag_contract.go`（受理フラグ/空Pin拒否）、`plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md`（契約一覧）の**計7ファイル**を変更した。旧HEAD `aa86e15e` のRepository Healthは`gofmt`差分・フラグ一覧・文書一覧不足でFAILし、最新Draft側で修正した。**この実装はPR #4625 Draft上だけ**であり、mainの実行可能コマンドではない。保存されたREA 6.8.0 capture fixtureのObservation 1件＋有向Relation 1件を、Evidence ID / raw digest / producer declared Source Revision / Graph coverage / Unknown / Unselected / AuthorityLimitsを落とさず受領する正負試験を用意した。新Go試験・actual Process/CI合格は確認されるまでNOT RUN／NOT PROVEN。

**R3-AのCLI契約候補（実装Draftのexact shape。第二Consumer用APIではない）**：
```text
tei external-candidates
  -evidence-path <saved-local-file>
  -tool-version <producer-declared-version>
  -observation-id <external-node-id>
  -relation-id <external-directed-edge-id>
  [-expected-evidence-id <id>]
  [-expected-received-digest <sha256>]
  [-expected-source-revision <producer-declared-revision>]
```
出力は`tei.cli.external-candidates/v0`の`schemaVersion/ok/result/error` Envelope（成功exit 0、引数/Local File Error exit 2、型付き受領・選択失敗exit 1）。ResultはR2の`tei.operations.external-candidates/v0`そのもの。既存Toolは実起動せず、`exec.LookPath("rea")`は読取環境での存在だけを観測する。CLIがローカルPathを許すことは**Remoteで任意Pathを受ける権限ではない**。R3-BではHost管理のEvidence selector→許可済み固定byteの取得→同じ投影ロジックの再利用という別の受入を維持する。実Source現在性、現在Grant/Disclosure、Producer解析AttemptとCLI Invocationの同一性はCLI Receiptからは導かない。

**同一Producer Evidence／別Consumer Invocationの評価（R3-C）**：受領raw digestが等しい二Consumerでも、受け取り時刻・Scope・Storage取得・開示・実Invocationは別。比較の構成要素はProducer ID/Tool/Source Revision、Storage read-back実byte、各ConsumerのSelected/Invoked/Received/版と返答、Audience別の許可済み公開Slice、Unknown/未選択。Digest一致・JSON見た目一致・片方のProcess stdoutを二回読むことはR3-C合格ではない。Private Sourceの件数・Hidden原因が公開応答/Errorを変えれば不適格（#2516）。R3-Aの公開コマンドはローカル用途、R3-Bの固定取得が成立するまでStudio/Remoteへ露出しない。

**他チャットの標準データ様式D0〜D3（[Draft #4619](https://github.com/hat47x/tei-platform/pull/4619)）との接続**：
| 段階 | REAと三要素への適用／独立Expected | 誤った昇格を拒否する境界 |
| --- | --- | --- |
| **D0 表現・構造の判別力** | Graph node/edge Identity・有向/平行辺・Source provenance・unknown、`Data/Function/Business`別support、同名異対象。Question Qの必要Slot（外部Relation ID/向き・TEI Qualified Ref）を保つ | 同一表示名・Source span・描画Edge IDや0件から正式Meaning、全域不存在、編集可能を推論しない |
| **D1 実Readと二Consumer** | 既存#2586の実PG `Function→Query→Provider→Rowset`と、Table/集計など同一Context/epochの二Consumerを独立受領。REA候補と旧X-TEA Function宣言は別Producerのまま比較 | REA R3成功やTable/集計同件数を、新旧意味同値・旧Actual・DB Update成功としない。JOIN fanout、NULL/欠損、小数は区別する |
| **D2 設計Source変更** | T1-BでREA多対多候補→TEI Sourceの明示Owner Review、Source版と逆写像Slot・参照切れを確認してPatch→再読取・再投影。Studio/CLI/AIのIntent同値を照合 | Graph全体がlossyでも局所Node editを全拒否する必要はないが、Relation ID不明・集計逆配分・未固定Slotはread-only。DB DDL/Effectは別 |
| **D3 実業務Operation** | T2でL旧実Observed／D受理済みDesired／T新実Observedと、実Host/Provider/Current Grant/Tx/Receipt/独立Readback・Recoveryを分けて検収 | D2 Source Patch成功や生成Artifact byte一致、Batch exit、ACKをDB Commit/承認/移行Cutover成功へ昇格しない |

**逆編集の必要条件を最小化する。** 形式的には、用途Qについて投影Pが同じ表示を返す二つの元Source状態x,yで、操作対象または必要な効果が異なるなら、`P(x)=P(y)`だけから対象を決める編集Intentは作れない。単一Slotへの原Source復帰には、固定Workspace/Source Revision/Qualified Ref/元Occurrenceと対象Slotの一意性・影響集合・Owner権限が必要。Graph全体の表示がlossyであっても、この局所条件を満たす箇所だけ編集候補を提示できる。ProjectionのLoss・可逆性の一般則は#2497、実Source Patchは#2500/#1126、実DB/Effectは#4125/#1174の既存Ownerに委ねる。

**実行順と報告**：#4625はR3-Aの局所PRとしてactual CI、コマンドcatalog/usage/繰り返しFlag/異常系を先に検査し、失敗があればそのDraft内で修正。R3-B/CとTEA-Q41/Q43は#3394/#4487/#1124の既存実Consumerへ引渡す。#4619 D1の実Provider/異質ConsumerとTEI Studio Desktop Stage1→Stage2セルフビルド#1824/#2831は、#4625のマージやR3-BをHard dependencyにしない。今回のこの節は**docs-only、main未統合**。PR #4625の実装結果を受理・動作済みと宣言せず、exact HEAD CIとOwner Reviewで別途判断する。新Core/Host/Parser/Registry/Policy/Validation Engineや第二Canonicalは増やさない。

<!-- rea-r3c-native-delivery-witness-20261114 -->
#### R3-Cの次の実装単位：製品CLIとNative Studioの独立受領（2026-11-14、仕様確定候補・製品試験未実施）

**技術検収の更新（PRごとに別資格）**：R3-A通常CLI [Draft #4625](https://github.com/hat47x/tei-platform/pull/4625) のHEAD `24863bd69a9139e1a75459a7c6607514dbc73ad8`は9/9 Workflow成功。R3-B byte-only R2入口 [Draft #4630](https://github.com/hat47x/tei-platform/pull/4630) のHEAD `1ad749fc0eaacc195dd28ea88a97bd44d9581050`は12/12成功。Host固定・bounded Reader [Draft #4637](https://github.com/hat47x/tei-platform/pull/4637) の**修正後の最新HEAD** `9b8f5ca9b50d34040cfa0682f026f4fc29090f01`も [Workflow実一覧](https://github.com/hat47x/tei-platform/actions/runs/43571893030) のRepository Healthを含む**12/12 completed/success**を確認した。初稿`0d961098`の2 Rootテストは誤ったExpectedにより失敗し、後続HEADの成功を旧HEADに遡及させない。主計画の本Draft #4603 HEAD `7b4de59`は別途2/2 Workflow成功。これらは**各Draftの技術合格でありmain未統合・R3-C未実施・Current Grant/Meaning未受理**である。GitHubの一時的な`mergeable`表示はOwnerレビューや競合再評価に代わらない。

**統合順序と差分所有**：#4630のbyte-only seamをR2既存仕様との互換・全Go/Health・Ownerレビューで単独検収してからmainへ取り込む。#4637は#4630を含む積層差分なので、取り込み後の最新mainに基準化して、Host専用4ファイルの差分と2 Root・実テスト子Process・絶対Path漏洩拒否を同一HEADで再検収する。#4625は独立にCLI既存JSON/exit互換・実子Processとmainとの差分を確認する。**この順序を満たさず、単に三PRのCIがgreenという理由で一括マージしない。**

**第二Product Consumerの第一候補はNative Studioの限定read-only受領**とする。現R3-B `LocalREAReader.Read(workspaceID,selector,observationID,relationID)`はGo Host内部にとどまり、Wails/UIから未到達。次の小PRでは固定WorkspaceとHost管理Selectorだけを入力とするnative専用の小さな読み出し境界を、既存#3419 Application Operationおよび#4503のHost選択に合わせて設計する。**一般`StudioBridge`へ無条件に公開メソッドを増やさない**。既存Wails Serverが同BridgeのCommand/外部プロセス機能まで登録していることは[#4627](https://github.com/hat47x/tei-platform/pull/4627)のSurface Inventoryで判明しているため、Native限定登録／Build対象とServer公開面の非増加を独立検査する。Web/HTTP/MCPへの送出は現在のActor/Grant/Disclosure資格が成立するまで非公開。Native版でもローカルHost範囲外のEvidenceを許可しない。別Host、別Parser、別AuthZ Engineは作らない。

**実際の二Consumer受領票（独立Expected #2514/#1180へ記録し、Operation v0へ新必須フィールドを追加しない）**：

| 観測する段階 | CLI側の証拠 | Native Studio側の証拠 | 合否の上限 |
| --- | --- | --- | --- |
| 対象の固定 | 実`tei external-candidates`実行のargv・ローカルEvidence実体／Pin | Host起動時固定`WorkspaceID+Selector`と許可Rootの実照合 | 同名・同Digestでも異Workspaceの現在許可は継承しない |
| 実読取 | プロセスが読んだEvidence byteのDigest、Evidence ID、申告Source版 | bounded Readerが同じ固定Evidenceを取得した実AttemptとDigest | Producer raw一致≠元Source現在性・必要閉包 |
| 操作呼出し | 本番CLI dispatch→R2 Operation→stdout/exitを子プロセスで受領 | 本番Studio native Service→R2 Operation→Wails境界を別実行で受領 | Go test子Processだけ、同一関数再呼出しだけではR3-Cに数えない |
| 受信結果 | `tei.cli.external-candidates/v0`の構造化結果 | UI側で受けた`tei.operations.external-candidates/v0`の許可済みread-only投影 | 比較はNode ID、Directed Edge IDと向き、Coverage、Unknown、Unselected、AuthorityLimits。別ReaderのReadAt/Tool在庫差は明示 |
| 許可と開示 | ローカル利用の範囲を記録。CLIのraw Path診断は別Scope | 対象外Selector/Root変更時は定型Error。Server登録やJS/DOM/ログ経由の非公開情報漏出なし | 同じ表示JSONでもConsumer/Audience/現Grantの同一性は主張しない |

**停止条件**：未知Selector、別Workspace、Root差替え、symlink脱出、4MiB超、保存raw改変と再Pin、同IDの平行Edge／曖昧選択、末尾欠落・partial/unknown、Private有無だけ異なる二世界、NativeからServerへの意図しないService露出を既存Expectedへ割り当てる。特に保存byteのA→B→A復元はDigestが再一致しても途中変更履歴や現在Grantを示さない。元Source/Worktree・Disclosure・Principal/Reader Epochを独立に照合できなければ、そのClaimを`unknown/not-qualified`にする。Nativeで1ユーザーの受領が成功しても、Webの2 Principal・2 Workspace・HTTP/WSのCurrent GrantをPASSとしない。

**R3-C後の最初の業務受入**は「旧ソフトのあるFunctionの参照Dataは何か」という問いに限定する。外部REA EvidenceのNode/有向Edge候補を、X-TEAのFunction宣言・Query SQL・TEI Data/Function/BusinessのQualified Refへ**0..N対0..Nのレビュー候補**として結ぶ。Sourceにない手作業、QueryのJOIN増幅、未確認のDB/Plugin、三要素別unsupportedを別に表示し、TEA-Q41/Q43とREA-ALIGN-01〜04で誤昇格を検査する。Ownerの明示承認後だけ通常`*.tei.md`をPreview→Save→再Readし、SHO出典再解決と旧実測L／Owner期待D／新実測Tをそれぞれ別試験にする。**今回の追補はdocs-onlyで、Native第二Consumer接続・実Delivery・Owner Meaning・新旧同値はNOT RUN**。Desktopセルフビルド#1824/#2831と実Provider Read#2586のP0をブロックしない。

<!-- rea-question-evidence-native-product-slice-20261114 -->
#### REA R3-Cを三要素・協業UX・実Providerへ接続する際の最小証拠契約（2026-11-14、横断成熟判断・Product E2Eは未実施）

**実装着手先**：具体的な第二Consumer実装と試験を [#4661](https://github.com/hat47x/tei-platform/issues/4661) に切り出し、R3全体のOwner [#3394](https://github.com/hat47x/tei-platform/issues/3394) に従属させる。別のREAランタイムや承認基盤を作るためではない。現mainの実コードでは `cmd/tei-studio/main.go` が `application.NewService(&StudioBridge{host: host})` を登録し、`assets/client.mjs` の `createReadOnlyClient` が `main.StudioBridge.*` を呼ぶ。`StudioBridge`にはCommand実行とSHO外部子Processの公開メソッドもある。[#4627](https://github.com/hat47x/tei-platform/pull/4627)の`-tags server` Inventory（Draft）は**静的method set・サービス登録の監査であって、実Wails route/現認可のPASSではない**。したがって、Native-only限定Serviceの登録範囲を実Build tagと生成JS、Windows実Wails受信で確定し、既存Bridgeのexported methodを安易に増やさない。現在の`LocalREAReader`はGo内部のHost Readerであって、まだ実Native Consumerではない。

**横断する主張をP/J/Qの3段階と製品受領に分離する**（既存三要素 [#4487](https://github.com/hat47x/tei-platform/pull/4487)／必要閉包 #3262／独立Expected #2514 の用語を再利用し、新たな正式Schemaや第三のProof正本は作らない）：

| 判断段階 | 具体的な問いと必要証拠 | 失敗時に主張してはならないこと |
| --- | --- | --- |
| **P：局所受領** | 固定したREA Producer EvidenceにObservation 1件＋`RelationID`・方向を持つEdge 1件が存在。受領raw digest、Producer Tool/版、Source宣言Revision、`Coverage/Unknowns/Unselected`と実Reader Attemptを保存 | 解析対象に含まれなかったFunction/Business/Manualの不存在、実DB Rowset、正式TEI Meaning |
| **J：異質な証拠の対応** | 外部Node/Edge ↔ X-TEA Source Occurrence ↔ TEI Qualified Ref／関係のSource Revisionを**多対多候補**として出典・粒度・未解決を保持。Target PGの実Readは別Provider/Instance/Attemptで示す | 名前・Digest・表示上のGraph形・同じタイムスタンプだけによる同一Function確定、旧新同値 |
| **Q：問いに必要な閉包** | 「使われているか／安全に削除できるか／業務手順が完了したか」に応じた`Required(Q)`と、実際に読んだ`Consumed(Q)`、未知のPlugin/動的呼出し/他Source/手作業/許可Scopeを比較 | 一件の陽性観測から全Graph完全、参照0から未使用確定、未探索から不存在確定 |
| **実Product受領** | 通常`tei` CLIの実プロセスと別のNative Wails WebView JSが、それぞれ固定証拠を`selected→invoked→received`した受領版／Result／Consumer Context | テスト用Go子Process、Reader関数2回、別HEADのgreen、同JSON表示から二製品受領・現Grantの成立 |

**否定命題に必要な証拠の非対称性**：適格な一件の陽性Witnessは「そのScope/Attemptで見つかった」を支持するが、「無い」「全て」「安全に削除可能」はその問いに必要なSource/Relation/手作業/動的依存の閉包を要する。一つの反例は全称を否定できても、少数の正常例では全称を証明しない。REA `GraphCoverage=complete`も**Producerが調べた集合の局所状態**であり、Data/Function/Businessそれぞれの独立支持範囲とは別。外部Graph completeかつData支持あり・Function/Business unsupportedの組合せで、画面に「F/B=0件」や「不存在確定」を描けばTEA-Q41失敗。Legacy側のFunction宣言と`order_entry.recent_orders`等のTarget PG実Read結果が似ていても、Legacy actual L、Owner Accepted Desired D、Target actual Tの独立証拠がないならTEA-Q43失敗。意図的なDの業務変更に対してL≠Tを一律FAILにも、一律同値にも扱わない。

**三軸は正本を増やさない**：Data／Function／BusinessはTEI三視点、論理要件／物理制約／実装は問い相対の評価観点、L／D／Tは旧実観測・正式受理・新実観測の資格区分。それぞれを直積にした27個のCanonicalモデルを設けず、既存Data/Relation/Function/Source/Receiptへの用途別Projectionとする。REA Node/Edge、物理FK/SQL、Function-level C/R/U/D、Business Action・手作業Effect、Host Principal/Grantは**型名が同じでも別の意味**として保持する。専門家に読めるTEI Business DSLから業務手順説明をほぼ直接生成する構想は、Ownerが受理した業務条件・手作業・例外・TODO外部部品と模擬契約を根拠にする。外部逆解析が出したフロー候補を正式な業務手順として自然文生成・保存しない（#1031/#1125/#1127/#4502）。

**StudioとAIの協業に渡す最小情報**（[#4605](https://github.com/hat47x/tei-platform/pull/4605)のUX-PIN/UX-DIAG、[#4650](https://github.com/hat47x/tei-platform/pull/4650)の「生成する一巡／使う一巡」、#4515）：
- **Locate→Observe→Pin→Review→Propose→Preview→明示Acceptance→Save/ReRead** のうち、R3-CはObserve/Pinのread-onlyまで。Focus、固定Pin、未保存Buffer、保存済Source、Accepted Meaning、生成Build、実行済み結果を別状態にする。Pin Aを裏再確認中に閲覧中BのEditor/Graph/Dirtyを変更しない。
- Native UIでの説明は「何を見たか／何が未探索か／何には使えないか」を先に示す。Source位置・Raw Evidence・非公開名前/件数は**受信者の許可Projectionへ選別してから**JS/DOM/ARIA/Loggerへ送る。AI外部Harnessへの転送権限をread-only Host Readerから推測しない。削除/Apply/正式TEI Source Patchは別のOwnerとSource slot一意性・現在版を要求する。
- 同じQuestionでもCLI/Native/SHO/AIは表示媒体が違う。Consumer間で照合するのはOutput文字列の同一ではなく、固定対象、Producer/Consumed Source、QualifierとVersion、Unknown/Disclosure、実受領内容。SHOへは既存Writerの引用OccurrenceとDocument Save/Reopenを別に検収する。

**最初の統合実例と非阻害順序**：①#4625/#4630/#4637の個別CI/Ownerレビュー・積層整理、②[#4661](https://github.com/hat47x/tei-platform/issues/4661)でCLI/Nativeの二実Consumer受領、③#1124/#1126/TEA-Q41のData/Function/Business多対多read-only Review、④一件の正式Source Owner受理・`*.tei.md` Preview→Save/ReRead、⑤#2586/#4638のTarget PG実Readと別のLegacy実測/Owner Expected照合（TEA-Q43）、⑥SHO保存文書の独立Readback。この順序は**上流の全機能完成を直列に待つ意味ではない**。Studio self-build #4583/#2831、通常製品Function実Read #2586は先行P0として並走する。REA 6.9.0のT0-06再資格・現在Grant/2 Principal・Query全件Coverage・SHO正式Writer・旧新Actual意味同値は別Expected、現時点ではNOT RUN。

<!-- rea-desktop-web-branch-session-evidence-handoff-20261114 -->
#### Studio Desktop／将来Web版の協業境界をREA Candidate Reviewへ接続（2026-11-14／構想整合・未実装）

**同一UI・異なるSource所有形態。** TEI Studio Desktopは開発者のローカルWorkspaceを編集しGitHub等で結果を共有、将来のStudio Webは単一（必要ならHA）開発サーバ上のチーム同時編集を想定する。共用対象はTypeScript Workbenchの表示・選択・レビューと既存型付きTEI Operation／Application Serviceの契約。Desktop=Wails/Local filesystem/ローカルGit、Web=Server Workspace/認証済みWeb Transport/共同編集Sessionとして分ける。Wails DesktopのBridgeそのもの、OS filesystem、Git repository stateや認可済みSessionをWebへコピーしない。**Wails/Go単一バイナリによるDesktopセルフビルド最優先**、Web共同編集は後続設計であり、R3実Consumer成功の前提にしない（#1824/#2831/#3419/#1251）。

**REAの表示を伴う状態を4層に分離する（実装Schema案ではなくreview観点）**：

| 層 | Desktop | Web共同編集 | 越境させないもの |
| --- | --- | --- | --- |
| **Evidence/Source版** | ローカル保存Evidenceと固定元Source/Workspace | Server保存EvidenceとBranchごとのSource/Manifest・許可Root | 同digest・同名・同pathを跨ぐ暗黙の資格継承 |
| **Working edit** | Dirty Bufferと保存済ファイル・Git差分 | 同Branchの共同編集Buffer（CRDT候補）と永続Source版 | CRDT収束、保存やGit commitだけでOwner Meaning受理 |
| **Selection/Review** | Focus（瞬間選択）と固定Pin、read-only Mapping候補 | Session別Focus/Pin、BranchごとのCandidate Review/競合 | 別Session操作による固定対象・未採用Candidateの暗黙置換 |
| **Acceptance/Delivery** | Owner DecisionとSource patchの明示適用、独立Readback | Ownerによる通常Source変更・Revision/権限再確認、用途別Disclosure | Git push/Server ACK/AI同意を承認・実DB Effectと同義化 |

**R3-Bを実装する場合のClient要求境界**：Browser/Sessionは任意`EvidencePath`や内部CAS/NAS/S3 locatorを渡さず、**Host管理Evidence selector＋外部Node/Relation ID**だけを提示する。Server側はRequestごとに信頼済みHost認証文脈から `Principal/Tenant/Workspace/Branch/Current Grant/Audience/Disclosure` を確定し、選択した `Evidence/SourceRevision/Reader Attempt` と照合して許可済みbyteを固定取得し、既存R2のReceive/投影ロジックへ渡す。**Clientが申告するPrincipal・Tenant・Branch・Plugin ID・AI提案はAuthorityではなく、Webの認証失敗をDesktopのローカルno-authへfallbackしない**。Branch名やSession IDだけではSecurity Principal/Owner Authorityにならない。Web未実装時は、固定Workspaceの既存MCP read-only ConsumerまたはDesktop Bridgeで第二Consumerを検収できる（#3394/#4503/#2516）。`ReadAt`、Producer申告Revision、受領Pin一致だけで現在Source・適用可能性を認定しない。

**同時編集・遅延応答におけるClaim Ceiling**：REAの解析Candidateは保存済みSnapshotの観測であって、共同編集中のDirty Bufferを自動解析した結果ではない。保存済Source R10に束縛したCandidateを表示中に、(a)Branch A→B切替、(b)R11への共同保存、(c)Session XのPin保持中にYが別Nodeを編集、(d)元Source R10のA→B→A、(e)Grant/Disclosure失効、(f)R10解析応答が後着した場合、**Pin intentを履歴として保持しても現在Qualificationは別に再確認し、旧Replyを新表示/Source Patchへ採用しない**。旧DOM/ARIA/キャッシュ/AI payloadに非公開Sourceが残る問題はPin単体の成功とは分け、#1124/#4515/#2516の失効条件へ戻す。共同BufferのCRDT/操作ログがあっても、TEI意味単位のPatch（Qualified Ref・Occurrence・Slot・Effect）には通常Revision precondition/Source再取得とOwnerレビューを要求する。

**最小の協業Expected（REA-WCOL-01〜05、既存#1180/#2514へ引渡す設計票、NOT RUN）**：
- **WCOL-01**: 同じraw Evidence digestを別Workspace/Branchで指す二Session。片方の許可済みSource RevisionとGrantだけでは、もう片方をQualifiedにせず、非公開存在を返答差に混入しない。
- **WCOL-02**: Session Aのread-only REA Pin中にSession Bが共同Bufferを変更、さらにSourceへ保存。Dirty Buffer／保存済Source／Producer申告Source版を別に示し、古いCandidateをSource Patchへ暗黙適用しない。
- **WCOL-03**: 同Branch内の二人が同名別IDまたは平行Relationを同時にReview。表示FocusやCRDTの変更順で最後の一件を正式意味へ後勝ち受理しない。多対多・unresolvedとOwner Decisionを保持。
- **WCOL-04**: Browserが`../`や内部Pathを送る／許可済みSelectorを別Branchで再利用。Hostが入力を拒否し、Error/DOM/ARIA/件数から非公開RootやEvidenceの存在を漏らさない。#3394 SEC-01〜04と#4503 EC-01〜05を再利用。
- **WCOL-05**: Desktopの同一EvidenceとWebの同一表示JSONが一致するが、ConsumerのSource Scope/Revision/Audience・Delivery Attemptが異なる。表示同値だけで同一Current Grant、旧新Source結合、Owner Meaning/実Effect/全閉包とは判断しない。

**既存試験との照合**：WCOL-01/04は#1824のWeb 2 Workspace×2 Principal×2 Provider／SWP-01/02/07と#2516のPrivate二世界、WCOL-02/03は#1824 SWP-03/05・#4515 PIN-S0・#1124のSource/Grant stale、WCOL-05は#3419異質Consumerと#1163 Evidence Packetへ対応する。**WCOL番号は横断追跡用の参照名であり、別Harnessや第二Expected正本を増設しない**。Web HostのPlugin Inventory≠Workspace Composition≠現在Grantは#1824/#951/#1016/#3421のOwnerが検収する。

**導入段階／Owner**：まずR3-A通常ローカルCLI #4625 のexact-head合否を確定→R3-Bは既存の固定Workspace MCPまたはDesktopで安全な第二Consumer一経路だけを成立→R3-C独立Producer Evidence/別Consumer Invocationを検収→T1-BでSourceに戻るread-only候補レビューを一本通す。その後、Web共同編集への`Branch/Session/Source Snapshot/Pin`資格確認を#1824/#1124/#4515のStudio Ownerで検討する。共同編集のCRDT・Git運用・マルチユーザAuthorityをR3-Bの新必須Provider/新Host/新Policyへ押し込まない。Source Patchは#2497/#2500、Owner Decisionは#267、Current Grantは#2516、Evidence取得は#4503。Web製品実装、共同編集負例、R3-B/C、実Grant/Owner受理は**NOT RUN／NOT ACCEPTED**。

<!-- rea-r3b-byte-only-product-seam-20261114 -->
#### R3-B準備：Hostが取得したbyteと、R2投影Operationの責務を切り分ける（2026-11-14）

**実装への最小投資**：[R3-B前段の独立Draft PR #4630](https://github.com/hat47x/tei-platform/pull/4630)（main起点、HEAD `1ad749fc0eaacc195dd28ea88a97bd44d9581050`、未統合）で、既存`internal/operations/external_candidates.go`に`ExternalCandidatesFromBytes(request, raw, at)`の**byte-only入口**を新設した。従来の`ExternalCandidates`はローカルPath/Version/IDの入力確認→同Pathを一度`os.ReadFile`→Pathを消したRequestと実byteを同一R2投影へ渡す薄い入口となる。R2の`externalanalysis.Receive`／`ReadCandidates`／Evidence ID/raw digest/申告SourceRevision／選択Node＋Directed Relation／Coverage/Unknowns/Unselected/AuthorityLimitsは**第二Parserを作らず共用**する。byte-only入口に`EvidencePath`が付いていれば`EXTERNAL_CANDIDATE_INVALID_REQUEST`で拒否する（FromBytesはファイルを取得せず、Policy/Grantも判定しない）。これを**Host管理の読取り結果を受け渡すための接続面**とし、R3-B完了と呼ばない。

**具体的な再利用先**：現main `internal/studiohost/observe_xead.go` の `ReadOnly.ObserveXead`には、(1)Host固定Workspace IDから許可Root取得、(2)相対Path正規化／用途の拡張子チェック、(3)`workspace.ReadSource`のbounded read、(4)Parser側にはファイル能力を渡さずbyteだけ渡す、(5)失敗文言をHost側で限定する既存先例がある。これはREAを今すぐ`ObserveXead`へ混在させる意味ではなく、R3-Bの**取得境界・負例設計の参照**である。`ExternalCandidatesFromBytes`にはCurrent Grant/Source現在性/Disclosure/Storage deliveryの資格がないので、R3-B実Consumerはその前後を既存Host/CAPI #3419/#4503/#2516で引き受ける。HTTP・Wails Serverへ`-evidence-path`やRoot実pathを渡すAPIを作らない。

**より具体的なStudio Web公開停止線**：[Wails Server Bridge inventory実装Draft #4627](https://github.com/hat47x/tei-platform/pull/4627)（最新HEAD `b9cf5447`でWails Host/Windows WebView/Repository Health等5/5 Workflow successful）では、現`application.NewService(&StudioBridge{host: host})`が現在15公開methodを持ち、Command実行やSHO子プロセス入口も含むことを実コードの反射/ASTで検知するテストが成功した。**5 Workflow PASSは15 methodのinventory成功であり、Public Webへ一括露出可能な安全性ではない**。R3-Bは既存のread-only Host入口または専用限定Facadeのみを使用し、Wails Server PoCのローカル`127.0.0.1`制限、将来の認証/Workspace/WS配送隔離を混同しない。Wails ServerでのR3新Operationの公開は#3421/#1043の別途レビューを要求する。

**Producer→Consumer byte受渡しの既存証拠との整合**：[Windows Desktop生成連鎖Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)は旧HEAD `0fcb7b62`の同一Windows AttemptでN2 bundleとN3 Go Embed対象の前後SHAを照合した。最新HEAD `357345d7`では生成版JobのN0→N2→N3とWebView2は成功したが、通常版JobのPowerShell OS版Probeが15秒Timeoutとなり、Workflow全体はfailure（11/12 Workflow success）である。この**受渡し確認の手順**はREAにも適用可能だが、別WorkflowのN2/N3成功をREAのStore→Consumer受領、Original Source closure/Current Grantへ移してはならない。

**R3-B→Cの最小検収境界（独立Expectedは#2514/#1180、別Harnessを作らない）**：

| 票 | 実装・検査入力 | 必須Expected／資格上限 |
| --- | --- | --- |
| **R3BY-01** | 同一Captured 6.8.0 REA byte、ローカルFile版と`FromBytes`版 | 型付きView/ReceivedDigest/GraphCoverage/Unknown/Unselectedが完全一致。これだけでは第二ConsumerのInvocationでない |
| **R3BY-02** | byte-only入口への任意Path、誤Pin、Producer未申告Version、不正byte | Pathは取得に使われず拒否、誤Pinはstale、不正byteはunusable。ローカルCLIだけがPathを受ける |
| **R3BY-03** | Hostの許可Workspace IDと固定Evidence Selector、Root外・symlink・改版・許可対象なし | Root/Revision/Grant不適格時に投影/外部送達0。内部absolute path・存在/件数を公開Error/DOM/ARIAへ漏らさない。**Host固定Root→限定byte Readは#4637で実装・旧HEAD CI合格。ただしCurrent Source Revision/Grant/外部Deliveryは未実装** |
| **R3BY-04** | 同一Producer受領byteをCLIとHost側の二つの実Invocationで受領 | 各Consumerの選択、実取得byte digest、受領時刻、許可Audience/Scope、完全Replyが独立に残る。同一JSONを二度読むだけは不合格。**R3-C未実装** |

**現実の到達順**：#4625 R3-Aの最新HEAD `24863bd6`は[Repository Health #43545393255](https://github.com/hat47x/tei-platform/actions/runs/43545393255)で`gofmt -l .`と`go test -count=1 -timeout 60m ./...`がPASSし、**exact HEAD 9/9 Workflow success**。R3-Aの局所技術CIは合格、PRはDraft/open・main未統合であり第二Consumer/Owner正式受理は未成立。[#4630](https://github.com/hat47x/tei-platform/pull/4630) `1ad749fc`はREA contract #43545780392とRepository Health #43545780222を含め、**同一HEADの12/12 Workflowが最終的にsuccess**。Windows WebView2 smoke #43545780481は第1 AttemptでPowerShell `OSVersion.VersionString`取得が15秒Timeout→FAIL、失敗したJobのみ再実行した**Attempt 2で実WebView2起動・Keyboard/ObserveXead smoke PASS**（`SMOKE_RESULT=PASS_NATIVE_WEBVIEW2_EDITOR_KEYBOARD_AND_OBSERVEXEAD`）。Windows再試行のsuccessは元Attemptの失敗が存在しなかったことを意味せず、REAのHost/第二Consumer実受領とも別である。**双方のexact HEAD検収と非連動の責務**を保持する。#4630自身はRoot/Host Consumerを実装せずWeb公開でもない。後続#4637で既存Host許可Rootからのbounded固定byte取得を実装した。次は実Studio Bridgeまたは固定Workspace MCPの許可されたReader一経路で、現在Grant/Disclosureを独立に確認してから実受領を観測し、R3-Cの別Consumerへ進む。TEA-Q41/Q43とT1-B Source差分レビューは並走可能だが、Desktop self-build #4583と実PG D1 #2586をブロックしない。新`ExternalCandidatesFromBytes`を第二Meaning/Source正本、Generic Reader Registry、Authority engineへ昇格させない。**この計画追補はdocs-only Draft。PR #4630のGo/REA契約・Repository Health・再試行後のWindows WebView2を含む12/12 Workflowは実CI PASS。ただしR3-B Host実Readの**Local-only実装は#4637旧HEADで技術CI合格**、REA製品第二Consumer、R3-C、正式Grant/Owner Meaningは未確認・未受理**。

<!-- rea-r3-ci-worktree-source-context-20261114 -->
#### R3 技術GateとSaaS多Worktree/DB Contextの非同一性（2026-11-14・検収結果＋後続設計）

**技術Gateの区別**：R3-A #4625 HEAD `24863bd69a9139e1a75459a7c6607514dbc73ad8`の9/9 Workflowは同HEADのLocal CLI/Go Scopeの成功であり、R3-B #4630のGo Operationを合格させるものではない。#4630 HEAD `1ad749fc0eaacc195dd28ea88a97bd44d9581050`はREA専用WorkflowとRepo Healthに加え、WebView2 smokeの**初回PowerShell OS版Probe Timeout（Attempt 1 FAIL）→失敗Jobのみ再実行（Attempt 2 PASS）**を受け、全12 Workflowがcompleted/success。**同HEAD／別run-attempt**で双方の証拠を持ち、初回失敗を消したり、古いWindows別HEADの成功を加算したりしない。R3 Product ExitはR3-BのHost ReaderとR3-Cの二Consumer実受領に依存するが、Stage1→Stage2 Desktop/実PG Readの独立P0をブロックしない。

**他チャットのStudio/SaaS ContextをREAへ限定的に取込む**：[Desktop Draft #4498 §0.13](https://github.com/hat47x/tei-platform/pull/4498)、[Studio統合Draft #4593](https://github.com/hat47x/tei-platform/pull/4593) の検討では、将来のSaaSが複数Git Worktree/Branchと複数DB instanceを扱う場合、①Viewerの選択（Branch表示・DB alias・Dirty Buffer・Pin）、②**Host実束縛**（trusted Workspace Root/Worktree実体＋Git commit＋Source Revision/Environment→実Provider Instance＋Principal/Current Grant）、③実Reader/Builder/Operation Attempt（固定入力・実Invoker・Delivery/Effect）を別Identityにする方針が明確になった。**Branch名/DB alias/Source digest/REA Evidence ID/同名Nodeの一致を、Worktree/DB Instance/Actorを横断する同一実対象の根拠にしない**。R2 `ExternalCandidatesFromBytes` は意図的にHost Contextを解決しないため、その結果単独から「どのWorktreeのSourceか」「現DB instanceへのReferenceか」「誰に開示可能か」を導けない。

**最小既存Expectedへの差分**：WCOL-01に「同一raw REA byte・同名Branch/DB alias・異なる実Worktree/Host Source Revision/Provider Instance」の変種を含め、WCOL-02にforce-push/Branch A→B→A/別Window後着の古いPin拒否を加える。WCOL-04ではClientが`workspaceID`・`branch`・`DB alias`・`EvidencePath`を自由に組み合わせても**Hostを迂回したOS Read／別Provider Invoke 0**を要求する。WCOL-05はCurrent Grant/Disclosure/Consumer Receiptが同一でない限り「同じ表示JSON」を同一受領資格としない。別のTest SSOTや認証/Builder/DB ResolverをREAへ増設せず、#1824/#3421/#2586/#4503/#2516/#2514へ配分する。実2Worktree×2DB×2PrincipalのEnd-to-Endは**NOT RUN**。Host固定Readerの単一Workspace Go実試験は#4637で実行済みだが、別Worktree・現在Principal/Grantへの適合は未検証。この設計だけでREAのR3-B合格とは呼ばない。

<!-- rea-r3b-local-host-reader-and-placement-gate-20261114 -->
#### R3-B ローカルHost固定Evidence Readerの実装候補／公開境界（2026-11-14）

**固定Root→byte受領のGo実装**：[Draft #4637](https://github.com/hat47x/tei-platform/pull/4637)（Host/実子Process技術CI合格HEAD `f38117dc69b722e14de36a69c3fae6478766d5d4`は12/12成功、最新の2 Root改変/復元試験HEAD `9b8f5ca9b50d34040cfa0682f026f4fc29090f01`は独立CI中、#4630依存、main比較6ファイル／Host固有4ファイル）に、`internal/studiohost/LocalREAReader`を加えた。**Host起動時だけ**選ぶ `WorkspaceID＋Selector→Workspace-relative File／Tool Version／EvidenceID／sha256 Raw Digest／任意SourceRevision` を固定し、閲覧側の`Read(workspaceID, selector, observationID, relationID)`にはRaw Path入力を設けない。以前の設計案からさらに修正し、**Evidence ReadAtを閲覧側に申告させずHost Clockで確定**する。テストでの固定ClockはHostパッケージ内の限定注入であり、外部APIには露出しない。

**既存機構の実利用**：既存 `ReadOnly.workspace()`の許可Root実体照合、`workspace.ReadSource`の正規化/Traversal-resistant rooted read＋通常ファイル/4MiB上限を使い、取得したbyteを#4630の `ExternalCandidatesFromBytes`（R2同一Parser/Receipt）へ渡す。不許可Selector・ファイル取得失敗・R2不完全/曖昧・誤Pinの外部応答は共通`OPERATION_FAILED`で内部Path/Producer診断を出さず、既存`WORKSPACE_NOT_ALLOWED`・`WORKSPACE_ROOT_CHANGED`を残す。Toolの実行ファイルPath/AbsentReasonは投影前に除く。新State Store/第二Parser/権限Resolver/Generic Evidence Registryは作らない。

**検証の最小独立反例**：実Captured REA6.8.0保存byteに対して、(1)Host Workspace固定Selector→一回Read→R2 view/Raw Digest/GraphCoverageと直接byte呼出しの全同値、(2)配列Binding変更による再ターゲット拒否、(3)他Workspace・任意Path文字列・不明Relation、(4)Evidence byte差替え（including同意味に見えるbyte変化）／4MiB超、(5)Root実体置換・symlink Root外脱出、(6)導入時の非正規化・Traversal・重複Selector・不正Pinを同PRのGoテストへ実装候補として固定。**新HEAD 170f9be5907cea4af981582ab4c11c5dafb91e87は12/12 Workflow completed/success**。[Repository Health #43549738182](https://github.com/hat47x/tei-platform/actions/runs/43549738182)の実ログでgofmt、全Goテスト、tei/internal/studiohostの成功を確認した。REA契約 #43549738160、Wails Host #43549738139、Windows WebView #43549738327、Self-build #43549738332も同HEAD success。これはR3BY-03の**Local Host byte取得だけ**の技術CI合格であり、実Wails/Desktop UIからのREA受領・Source最新版・Grant/Audience・独立Deliveryは未成立。

**他チャットのDesktop Worker方式の横断差分**：[Plugin横断 Draft #4621 §20](https://github.com/hat47x/tei-platform/pull/4621)で、Desktop＝(A)Native Bridge、(B)同一exeの従属Worker＋Windows Named Pipe／Unix UDS、(C)別Wails Serverバイナリ＋loopback HTTP/WSの三候補が性能/障害分離/ACL/Artifact多重性から比較対象となった。REA Readerは**共通型付きOperationを使うローカルHost内の取扱い**であり、A/B/CどのTransportを選んでも `Workspace/Worktree/Current Grant/Requested Audience/Reader Attempt` の相違を消さない。(B)/(C)を採用してもClientへRootや`EvidencePath`を渡す捷径を許さず、別Process上のRequest identity、OS ACL、Crash/timeout、Provider世代、ネットワーク/IPC Disclosureを#1824/#3421の既存Expectedへ渡す。**B/C方式選定とWeb/WASM導入はR3-BやStage1→Stage2 P0のHard Gateではない**。

**R3-B/Cのまだ欠ける辺**：このGo Readerは正式`StudioBridge`／Wails Service／MCP Serverへ未登録であり、ローカル同一Processの`Read`成功を第二Transport Consumerへ加算しない。Reader返却結果は外部Node・Source Range・Raw Digest/Unknownを含み得るため、`#2516`の用途別Current Disclosureが成立するまでWeb/Harnessに送信しない。保存EvidenceのDigest Pinは元Source現在性／A→B→A／Worktree Revision証明ではない。R3-CはローカルCLIと**異なる実Consumer経路**のSelected→Invoked→Received、固定Scope/Identity/独立Receiptを#2514のExpectedで検証し、その後T1-Bに進む。DraftのCI成功や新Host Readerの追加をOwner Meaning Acceptedへ昇格させない。

<!-- rea-r3b-independently-received-evidence-contract-20261114 -->
#### R3-BからR3-Cへの受領資格：Readerが動いたことと別Consumerに届いたことを分ける（2026-11-14）

**技術的に確認した範囲**：REA R3-A通常CLI [#4625](https://github.com/hat47x/tei-platform/pull/4625)、R2 byte-only Operation [#4630](https://github.com/hat47x/tei-platform/pull/4630)、Local Host bounded Reader [#4637](https://github.com/hat47x/tei-platform/pull/4637)は、それぞれ別HEADで技術CIが合格している。ReaderがGoプロセス内で実ファイルを取得し、R2 Viewを返すことは、別Process／Wails／MCP Consumerが実呼出し・実受信したこととは別のClaim。#4637のLocalREAReadResponseには別Consumer受信完了の独立ReceiptやPrincipal/Current Grantはない。**単一Functionの二度呼出しや同じJSONの再表示をR3-C成功とは数えない**。

**R3-Cへ引き渡す最小受領票（#2514/#1180の既存Expectedを使い、新しいReceipt正本を増設しない）**：

| 境界 | 観測すべき入力・結果 | 誤昇格させないもの |
| --- | --- | --- |
| 固定Selection | Host設定のWorkspace ID、Selector、Evidence ID、Raw Digest、Observation ID＋有向Relation ID。Client Path/内部Object LocatorはSelectorに使わない | Viewer Pin／同名SourceはHost Grantではない |
| 実Source・Scope | Root実体、Worktree/Git Commit・Source Revision・現在Grant・Principal・Audience、Read直前の版/可視範囲。#4637が現時点で実証するのは固定Rootと保存Evidence Pinのみ | Producer申告Revision／Raw Digestから元Source現在性を推定しない |
| Reader Attempt | 実Invocation、Host取得時刻、実byte長/digest、R2型付き結果、Unknown/Coverage、拒否/部分/失効。#4637のReadAtはHost Clock所有 | Reader実行と別Consumer受領は異なる。Tool PresentはHost宣言でありREA最新実行証明でない |
| Disclosure／配送 | 目的・Audience・許可Field/Subject/Public Slice、実送出byteと受信相手、拒否時OS/Network送達0、二Private World対照 | R2 ViewのLocation/Unknown/件数/digestを無審査でWeb/Agentへ送らない |
| 独立Consumer Receipt | CLIの実子Processと、別のHost/Studio/MCP実Invocationの両側で、Selected/Invoked/Received、Scope・時刻・実Response・欠落/Timeoutを独立記録 | 同一hash、同一JSON、内部関数呼出しだけで二Consumerとしない |

**独立負例**：他Principalへ同Selectorを渡す、別Worktreeの同名Branch/DB aliasへ切り替える、Source A→B→A・Grant失効・別Tab後着、Selected Provider A→Invoked B、Wails Server broadcastを無権限Audienceが受信、内部absolute Pathや非公開Evidenceの存在/Count/Digestが公開結果に現れる。正例の許可read-only Operationを壊さず、拒否時の実Provider/OS ReadとPrivate Deliveryが0であることを別検収する。現ReaderのWorkspace Root保護は、共有Webでの本人認証・現在Grant・Private Sliceの代用品でない。Ownerは#4503/#2516/#3421、実OS/外部送出は#2515、独立Expectedは#2514とし、REA専用Host／第二Parser／第二Authorityは作らない。

**並行案件との資格区別**：[Wails Server Method/Services Inventory Draft #4627](https://github.com/hat47x/tei-platform/pull/4627) HEAD b9cf5447は**5/5 Workflow successful**だが、実HTTP/WS認可を保証しない。[Desktop Builder Resolve＋Child起動前Pin Draft #4634](https://github.com/hat47x/tei-platform/pull/4634) HEAD 653d0d9aは**8/8 Workflow successful**だが、Stage1→Stage2実成果物やREA受領を保証しない。[標準データ様式 Draft #4619](https://github.com/hat47x/tei-platform/pull/4619)で検討する実Query入力・ProviderとRA-0/RA-1/RA-2は独立P0であり、REAの二Consumer完了を待たない。いずれもDraft・未merge。別HEADのPASSを合算してR3-C/Source Migration/Owner Meaningを完了扱いにしない。

<!-- rea-r3b-process-witness-cross-lane-receipt-20261114 -->
#### R3-B実Process Witnessと、Query/Builder/Registryに共通するConsumer証拠の切り分け（2026-11-14）

**R3-B技術試験をさらに一段進める**：既存[Host Reader Draft #4637](https://github.com/hat47x/tei-platform/pull/4637)へ、実Captured REA6.8.0を用いるテスト専用の**別OS子Process**を起動し、その子が`LocalREAReader.Read`から実取得したEvidence ID／Raw Digest／Node ID／有向Relation ID／Host ReadAtをプロセスID付きの最小JSON受領票として親Testへ返す`local_rea_reader_process_test.go`を新設（追加試験HEAD `f9e355d5ccdc96bf4a25ff939fca444a18de1942`は11/12 Workflow success・Repository Healthのみ`gofmt`失敗、整形修正HEAD `f38117dc69b722e14de36a69c3fae6478766d5d4`は[Repository Health #43558330965](https://github.com/hat47x/tei-platform/actions/runs/43558330965)を含め12/12 Workflow completed/success）。同じ実子入口で、未登録Selector・任意Path文字列を装ったSelector・他Workspace・Evidence改変では非公開Evidence ID/Digest/Node/Source PathをError/Replyへ載せない負例を加えた。**この子Processはテストハーネスであり、製品Studio/Wails/MCPの独立Consumerでも認証済み配送でもない**。親Testの別Process起動とChild Receivedを実証できても、R3-C ExitにはCLIと実Product第二Consumerの独立Selected→Invoked→Received／Audience/Current Grantが別途必要。

**exact HEAD評価に関する原則**：#4637の旧HEAD `170f9be5`は12/12 Workflow成功／Repository Healthで全GoテストPASS。Child実Process試験HEAD `f9e355d5`は[Repository Health #43556801954](https://github.com/hat47x/tei-platform/actions/runs/43556801954)の実ログで、**全Go testはPASSしたが、新規`local_rea_reader_process_test.go`の`gofmt`だけFAIL**。他11 Workflowは成功。整形修正HEAD `f38117dc`は、そのHEAD自身の12/12 Workflow completed/successで独立に検収した。**次の二Root隔離試験を加えたHEAD `0d961098ca61781d0db7ccfa2b10dc06b4dde0d4`は再度別のCI評価を要する**ため、旧HEADの成功を最新HEADの結果に転記しない。#4625 CLIと#4630 Byte Operationの既存別HEAD成功も、新Childが同一Delivery chainを検証した事実にはならない。

**他チャットから共通化できるのは合格数ではなく「証拠を渡す順序」**。Producer／Selector／実Input／実Invoker／実Result／Recipientの各段階を、対象の異なる案件でも分けて扱う。

| 他レーンの実根拠 | 既に検証済みの限定Claim | REAにも必要で、まだ別途検証するもの |
| --- | --- | --- |
| **実DB Browse RA-2 [#4638](https://github.com/hat47x/tei-platform/pull/4638)** | 使い捨てPG16の参照用Readerで7ケース実行、Run #43553849861成功。3/2/1/0行、Source変更/重複JOIN、RLS/NULL/権限等を独立Go/DB Expectedで照合 | 通常`tei`実Operation・独立Native/Studio二Consumer受領は未証明。REAのEvidence ID一致も実Source現在性の代替にならない |
| **Capability Input/Effect [merged #4636](https://github.com/hat47x/tei-platform/pull/4636)** | 不正Input/Effect宣言をAdapter/Provider実Call前に拒否する契約がmainへ統合された | REAについてはReader Selector/Source/Grant/Disclosureを別判定する。Input preflight合格を実Source・Current Grant・Meaning受理へ昇格させない |
| **Build Resolve＋実Child Pin [#4634](https://github.com/hat47x/tei-platform/pull/4634)** | 最新HEAD `7d6a1238`で8/8 Workflow success。選択されたBuilder契約と実Child起動前digestを照合 | 同一Attemptの非Plugin生成Stage1→Stage2実WebViewと、そのConsumer受領は未達。選抜/バイト一致だけでは実Product配置資格にならない |
| **Desktop H0/H1 [#4587](https://github.com/hat47x/tei-platform/pull/4587)** | HEAD `357345d7`のWindows生成版JobはN0→N2→N3と実WebView2成功 | 同じWorkflowの通常版はPowerShell OS probe 15秒timeoutで失敗。11/12成功を全件合格へ繰り上げず、非Plugin完全Self-buildとも混同しない |
| **Registry配布 [#4640](https://github.com/hat47x/tei-platform/pull/4640)・Generator [#4641](https://github.com/hat47x/tei-platform/pull/4641)** | 外部Release Assetの実readback・Redirect/署名/Trustと、Builder成果物の実Consumer資格を設計Candidateとして明確化（両者docs-only Draft） | Release外部実Asset/Current Trust、Generator実Artifact consumerとも未実証。REA同様に「Digest一致≠取得元/公開可否/実受領」を分けるが、新共通Evidence Engineは作らない |

**最新mainへの重複回避**：[Script Launch横断文書 #4642](https://github.com/hat47x/tei-platform/pull/4642)は2026-11-14に**main merged**（merge `1995f87e596ec77b2c05c4adeef288445cc91bb9`）。RA-2の実PG7ケース、生成Windows Desktopの部分PASS、署名付きReleaseのloopback Reference（merged #4617）を、既存Script Launch計画へ出典・適格性ごとに反映している。**REA専用の主計画では、この統合済み文書の本文を複製せず、異質Consumer受領の判断境界だけを接続する**。実GitHub Release外部取得／通常Product Query／非Plugin Stage2セルフビルドを、新たに成立したClaimと誤読しない。

**製品次段の最小API／受入順序**：Local Host Readerは現在`Read(workspaceID, selector, observationID, relationID)`のみで、`StudioBridge`へ未登録・Reader Attempt/Recipient/Grantを公開Resultへ追加していない。まず#4503のHost設定とSource/Worktree版・同一Evidence Byte、#2516のAudience相対Field開示、#3421の現在Principal/Grant／実HTTP/WS配送隔離を**現行Hostで**判定する。次に認可済みRead-only Operationだけを型付きWeb/Native Consumerへ渡し、受信側の実Bytes・識別子・時刻/Scopeを別に採取する。失敗時にOS Read/Child Process/Network送達0となる負例を検査し、最後に#4625の実CLI Processと独立Consumerの一致／相違を#2514へ送る。**この公開前Gateを省略して#4637の生R2 ExternalCandidateViewをHTTP/外部Harnessに流さない。** これは新Policy Engineや二番目のHostを採用する計画ではない。

**実装上の採用指針**：Product共有は既存Operation／Host Boundary／既存Receipt/Expectedの束縛規則に限る。REA Host Readerは`#4503`、Disclosureと匿名/他Principal二世界は`#2516/#3421`、Source/Workspace RevisionとOwner Meaningは`#1124`、独立Consumer/FailureのExpectedは`#2514`へ引き渡す。Current Grantも実OS/Network送出も未搭載のままR3-C/共有Web適格と認定しない。Desktop Self-build・実PG ReadのP0はREAの子Process試験でブロックしない。

<!-- rea-r3b-multiroot-identity-not-authorization-20261114 -->
#### R3-B 2つのWorkspace Rootのバイト列・局所失効・Producer契約の独立検証（2026-11-14）

**実装の到達点**：[Draft #4637](https://github.com/hat47x/tei-platform/pull/4637)に`internal/studiohost/local_rea_multiroot_test.go`を追加し、既存`ReadOnly`が許可した**2つの実Root**にCaptured REA 6.8.0の**同一Byte列**を保存する。両Rootは同じProducer Evidence ID・申告SourceRevision・Node ID・有向Relation ID・Raw Digestを持ち、`(WorkspaceID, Selector)`だけが別のHost固定対象を指す。両方のローカル正常Readを保ち、一方のRootだけが改変/置換されたときに**もう一方のRootへ選択・失敗を伝播させない**ことを受入対象とする。

**初回CIで試験の前提を訂正した**：最初の新HEAD `0d961098ca61781d0db7ccfa2b10dc06b4dde0d4`では、一方のJSONの末尾に改行を追加したうえでHost BindingのRaw Digestも更新し、「別byteでも正常Read可能」と試験で仮定していた。しかし[Wails Host #43571483812](https://github.com/hat47x/tei-platform/actions/runs/43571483812)の**実`studiohost` Go test**は、そのRootだけ`OPERATION_FAILED`となり、同試験がFAILした。これは**テスト用の改変されたProducer deliveryを正常Evidenceだと扱う誤ったExpected**であり、既存R2受領の契約を緩和する理由にならない。旧HEADのWorkflowは9件success・1件failure（別途完了2件）、全件成功ではない。

**修正版の2 Root正負例**：更新HEAD `9b8f5ca9b50d34040cfa0682f026f4fc29090f01`では(1)2 RootともCaptured原byteで正常Read、(2)Root Bのみ末尾改行を加えた際は**固定Host Pin不一致で拒否**、(3)改変後DigestへHost Bindingを再設定しても**R2の保存Producer deliveryを未検証byteとして拒否**、(4)Root Aは影響されず正常Read、(5)Root Bへ原byteを戻すとローカルread-only再取得は可能だがA→B→Aの中間Source変更・Grant失効は証明しない、(6)Root Aのディレクトリ実体を入れ替えたらAのみ`WORKSPACE_ROOT_CHANGED`、Bは正常、をGoテストにした。**最新HEADのCIは新Attemptで別に検証する**。この試験はSource/Grantの真正性や外部製品受領を検証するものではない。

| 保護対象 | 局所Go Expected | 別途Ownerが必要なEvidence |
| --- | --- | --- |
| 同じExternal ID/Raw byteの2 Host Root | `(WorkspaceID, Selector)`固定の正常Readを双方で保持 | 異なるGit Worktreeの実Commit・Model/Source Revision／Principal Current Grant |
| Root BだけのByte改変、別Raw DigestへのBinding再設定 | 元Pinで失効拒否。再Pinでも**不正Producer deliveryを正常へ昇格させない**。Root Aは継続 | REAの新しい正規Producer再実行、Target Source closure、Owner Meaning、時間的A→B→A判定 |
| Root A実体置換／Root B不変 | Aのみ拒否、Bの取得可能性とSelector対象を維持 | OS File Openの競合、Web/IPC Recipientごとの現在Disclosureと実送出 |

**他チャットの多Context設計との結合**：Studioの表示Branch・DB alias、Host実Worktree/Git Commit/Source Revision/DB Instance/Principal、実Operation/Consumer Attemptを同じIdentityにしない。特に**同一Evidence ID＋同一Raw Digestでも、2 Rootが同じSource/Grantであるとは限らない**。REAの今回のGo試験は許可Rootの実Readだけを扱い、2 Worktree×2 DB×2 Principalの実検証は未実施。#4503/#2516/#3421へ2 Root＋現在Grant／用途別送出の負例を渡し、#2514に独立Consumer Receivedを残す。新Host／Registry／Parser／Authority Engineは導入しない。

**P0を変更しない**：Studio Desktop Stage1→Stage2（#4583/#2831）と通常`tei`実PostgreSQL Read（#2586）を優先し、REAの局所同一性試験が追加Hard Gateとならないようにする。

<!-- rea-wails-server-wasm-delivery-isolation-20261114 -->
#### R3 Consumerの配置方式・配信隔離：Wails Server／Go WASM候補との整合（2026-11-14・設計のみ）

**他レーンの設計差分をR3へ過剰実装せず引き渡す**：[Plugin横断Draft #4621 §17](https://github.com/hat47x/tei-platform/pull/4621)では、Studio Webの実行形態としてWails v3 Server Buildを候補へ追加し、GoのBrowser WASMは純粋なread-only Projectionだけに限定して比較すると整理した。**R3-Bの最短経路は従来どおり既存固定Workspace Desktop BridgeまたはMCPの一方**であり、Server/WASM実装やWebでの2 Principal実HTTPを新しい受入Hard Gateへ加えない。Wails DesktopのBridge公開methodをそのままnetwork-exposedにせず、Host別Transportの入力・出力・Audienceの検査を#3421/#1043/#2516で再使用する。

**Wails Serverイベントの非開示境界**：Serverの通常Event broadcastが複数接続Browserへ送られ得ることと、Principal/Workspaceごとに閲覧許可されることは別。REA候補/Source span/Unknown/件数/Raw digest/内部Store locatorをEventに無条件同報しない。二Workspace×二Principal×二Tabで、公開側の受領bytes・DOM/ARIA・発行Event/下流Tool replyを確認し、他AudienceのCandidate/存在情報が一文字も混入しないことを、#3421のWSP-02/03、#2516二世界・#4503 EC-05へ引き渡す。Origin制限/localhost/Browser IDだけではGrant、Workspace認可、Recipient delivery証拠にならない。これらをR3の局所read-only CLI結果には帰属させない。

**Go WASM配置候補の制限**：R2の解析・Candidate Projectionを将来Browser WASMへ載せる選択は、同一固定・公開許可済みbytesからnativeと一致するread-only表示を作れる場合だけ比較可能。WASM側の計算結果はOwner Meaning、Source closure、Actual Host Provider/Grant、Source Patch、Raw Evidence取得権限を獲得しない。比較する場合、native↔WASMのNode/Parallel Relation/Unknown/NULL・欠落・Unicodeと出力順、toolchain/wasm_exec.jsとJS bridge版、初回load・メモリ・Workerの性能、明示Request前の通信とPrivate bytes受領0を独立観測する。利点がなければ不採用。**WASMを選んでもR2 Operation／Evidence／権限Resolverの第二正本は作らない**。

**既存検収への対応（新試験Harnessなし）**：WSP-02/03は#3421/#1824、Disclosureは#2516、Evidence固定選択は#4503、Native/WASM同値と性能は#4621/#1251、REA実Readerは#3394がOwner。Web実Event/実WASM/実HTTP/外部配信、Studio BrowserでのREA新受領、正式方式採用は**NOT RUN／NOT ACCEPTED**。R3-A #4625 のGo CLIでのlocal Evidence Read成功をWeb共有Server・WASMの資格へ昇格させない。

**到達条件と優先順位**：R2 main統合済み→R3の同一Evidence・独立二Consumer実受領→T1-Bの一機能対応Preview→T2の旧実測／受理済み意味／新実測・SHO Save/Reopen。修正済み#4584のSubject存否判別を各Consumerで利用できるか、#4570の三視点server経路と実Windows Desktopの差、生成Inspector別HEAD成功の非加算は別資格として保持する。TEI Studioセルフビルドと#2586通常Product Provider Readの完了をR3に依存させない。

<!-- rea-semantic-admission-process-witness-20261114 -->
#### REA観測をTEIの意味・実行へ結ぶ最小接続（2026-11-14／設計統合、製品受入は別）

**位置付け**：外部逆解析ツール **Reverse Engineer Anything（REA）** の成果は、TEIのData／Function／Businessを理解するための**根拠付き観測候補**である。業務・会計の **Resource–Event–Agent** モデルとは名称だけが共通で、同一機構ではない。将来つなぐ場合も、外部ソース中の「Agent」「Event」から業務上の担当者・責任・認証Principal・実行時Eventを自動確定しない。三要素固有の解釈はPlugin側、Identity・Source/Relation・型付きOperation・Projection・検査はTEI共通側に置く（#1124/#1125/#1126/#4198/#3394）。

**4つの区別を保ったまま取り込む。** ①ソースの記述・構文（取得集合、位置、Producer版）、②観測した構造（Node/Edge ID、方向・多重度、Graph coverage）、③業務上の意味の候補（Data/Function/Businessの対応、保留、反証）、④評価と実現（実Query/Provider、Business Operation、生成Artifact、Consumerでの動作）は別々の証拠である。標準データ様式の「型・構造・表現・Facet」も直交させ、Graphviz/Markdown/図面/説明文は同じ根拠付きCandidateの**用途別の見せ方**とする。見せ方を変えても外部Element IDを正式TEI Qualified Refへ上書きしない。DSL新構文やJSON Schema取込みは将来のLanguage Plugin/宣言的構文Adapterで検討し、R3のために第二Parser・第二Canonical・新DSL文法を必須化しない（#4619/#4359/#2497）。

| 接続段階 | 固定する根拠 | この段階で確定できないこと |
| --- | --- | --- |
| **観測の受領** | Producer Evidence ID、raw byte digest、Tool/版・Producer Attempt、申告Source Revision、Observed/Inference/Unavailable、Node ID、有向Relation ID、Unknown/Unselected/Coverage | 元Sourceの現在性、必要Source全量、正式TEI Subject/業務意味 |
| **問いに沿った照合** | Question、必要なSource/Graph/Function/Manual範囲、実Consumed集合、多対多候補と未決・反証 | 参照0による未使用確定、Graph completeからBusiness complete、同名による自動採用 |
| **人が受理する変更** | TEI Qualified Ref、元Source版/Occurrence/Slot、差分・影響、Owner決定、Save後の再Read・再Projection | UI選択、AI要約、SHO草稿やBuild成功による無承認Source変更 |
| **生成・実行・照合** | Provider選抜/実Invoke、生成Input/Artifact/使用Byte、実DB/Host Context、Result/Effect/独立Consumer受領 | byte一致からの実動作、Function exit0からのBusiness完了、応答成功からのDB commit |

**プロセス図との意味境界**：TEI共通の「Bootstrap → 待受け → Event/Request → 処理 → 必要時Response」という実行フローでは、REA Evidenceの**取得・閲覧要求**と、解析対象アプリケーションで推定した**イベント処理の流れ**を別レーンに描く。前者はHost管理Evidenceを読んでRecipientへ届けるRead-only Operationの実証、後者は旧実装に関する静的な観測候補である。推定されたHandler/TimerをTEIの実行可能なFunction/Actionへ暗黙登録しない。正式Function/Actionが業務意味、Script/Providerが実装を所有し、起動の適格性・現在Grant・Effect/Receiptを別途検収する。未実装部品をTODO＋Mockで全体シミュレーションする場合も、Mock成功をLegacy実測へ昇格させない（#1125/#1129/#4614）。

**R3-Cを縦断試験の最小断面とする。** 同じCaptured REA 6.8.0 Producer Evidenceから、通常の tei external-candidates 実子Processと、実Native Wails/WebView2 JSの**別Product Consumer**をそれぞれ起動して対照する。Host固定Selector→許可Rootからのbounded実Read→既存 ExternalCandidatesFromBytes →許可済みProjection→JS/DOM実受領の順に、Parent/Child Attemptの系譜・受領byteと版を記録する。Producer、CLI、Host Reader、Native ConsumerのAttemptを一つのIDへ潰さない。Selected/Invoked/ReceivedとNode ID・Relation ID/向き・Unknown/Unselectedを受信側で独立照合する。同じGo関数の再呼出しやテスト専用子ProcessはR3-C達成とはしない。Native専用Serviceが server build のNetwork公開面を増やしたら不合格とし、一般StudioBridgeに安易にREAメソッドを増やさない（#4661/#4627/#4503/#2516）。

**最初の業務上の問いは一つに絞る。** 「旧システムのFunction Fは、どのDataを読み書きするか」。REA構造証拠、X-TEA宣言、SQL/FKの観測、手作業、Target PGの実Readは**別Producer・別粒度**として併置し、対応は0..N対0..Nの候補とする。SQLに現れるテーブル名からFunctionの業務意味やC/R/U/Dを確定しない。GraphCoverage=completeでもDataのみ支持・Function/Business不明なら unknown/unsupported を保持（TEA-Q41）。Legacy Function名と新PG結果の一致も、旧実測L／Ownerが受理した期待D／新実測Tの同値証明ではない（TEA-Q43）。Previewから通常のTEI Sourceへ明示採用→Save/ReRead→SHO出典再解決は**R3-Cの後の独立受入**である（#1124/#3788/#2497/#4317/#4467）。

**既存Expectedへ加える検証観点（新しい試験正本は作らない）**：

| 項目 | 必須の正負対照 | 既存Owner |
| --- | --- | --- |
| **REA-XP-01 受領の系譜** | 同一Producer rawをCLIとNativeで別々に実Invoke/Receiveし、各Attempt/受領版を保持。共通関数を二度呼ぶだけなら不合格 | #4661/#3394/#2514 |
| **REA-XP-02 意味を自動昇格しない** | Graph completeかつData支持・F/B unsupported、同名別ID・平行Edgeでも、Meaningや不存在・未使用を確定しない | #1124/#4487/#3788 |
| **REA-XP-03 Source/Selection競合** | Source/Root/PinのA→B→A、別Tab後着、閲覧中Bに対するPin A再検証でもBのDirty Buffer/Graphを変えない | #4503/#4515/#1124 |
| **REA-XP-04 開示前の拒否** | 他Workspace/未知Selector、Raw改変・4MiB超、Current Grant不適格、Private二世界で、許可外Read・Delivery・DOM/ARIA/ログ漏洩を拒否 | #4503/#2516/#3421 |
| **REA-XP-05 業務上の採用** | Source版・Qualified Target・Owner決定を満たすときだけPreview→Save/ReRead。手作業・JOIN fanout・partial/NULL/未取得を残す | #1124/#1126/#2497/#2586 |
| **REA-XP-06 実行比較の上限** | L（旧実測）/D（Owner期待）/T（新実測）とProvider actual Invoke、Result/Effect/Recoveryを別評価。MockやBuild、REA受領だけなら未実施 | #2586/#2387/#1174/#1180 |

**順序と資格**：①#4625/#4630/#4637の各HEAD・積層差分・Ownerレビュー、②#4661のNative限定導線と二Product Consumer実受領、③一Functionのread-only対応Review・明示的なSource保存、④SHO実Save/Reopen、⑤L/D/Tの独立実測比較、と区切る。#4637 HEAD 9b8f5ca9b50d34040cfa0682f026f4fc29090f01 はRepository Healthを含め**12/12 Workflow completed/success**を再照合した。ただしDraft／main未統合で、R3-C、Current Principal/Grant、Owner Meaning、SHO Writerや旧新実測は未成立。Studio DesktopのStage1→Stage2セルフビルド（#4583/#2831）と通常CLIからの実PG Provider Read（#2586）はREAのHard Gateにしない。新Host/Authority/Parser/Receipt Engineを設けず、既存Operation・Host・Test Ownerを再利用する。


<!-- rea-meaning-preservation-question-lifecycle-20261114 -->
#### 逆解析から業務モデルへ進む際の意味保存と、処理の再構成（2026-11-14／横断計画の追加判断）

**今回確定する設計判断**：REAの役割は、旧システムにある構造・参照・処理候補を、出典と未解決事項を残してTEIの標準データ構造へ渡すところまでとする。正式な業務意味の受理、現在の実行権限、DB更新や通知の成功は、逆解析結果からは確定しない。すでにmainへ統合された[起動・イベント・処理・応答の意味設計](design/PROCESS_LIFECYCLE_TRIGGER_OPERATION_SEMANTICS_2026-11.md)（#4666）へは、まず**観測候補**として接続する。共通Event Loop、TEI独自Host、第二のParser/Registry/Meaning Storeを追加しない。

**関係を保ったまま意味を照合する。** 標準データ構造へのProjectionでは、少なくとも「元のIdentityと参照先／関係の型と向き／多重性／順序／有効時刻・観測時刻／NULL・欠損・未取得／粒度・単位／Source・導出規則・Coverage」を用途別に点検する。すべてを全型の必須Fieldにするのではなく、**元Sourceに存在する意味特性と、問いに必要な特性**を照合する。Propertyが不明なら不明のまま残し、変換で失われる特性はLoss/Diagnosticとして示す。表示専用Occurrence/Edge IDと外部REA Relation ID、TEI正式Qualified Refは別Identityであり、Graphや表の表示が一致しても意味上の同一性を推定しない。Source-backedな投影に新しい説明文や業務Actorを追加する場合は、元出典で確認できた範囲とAI推論部分を分ける。これは標準データ様式・自己記述DSLの共通Semantic Contract（#2497/#4619）を使う**検証対象**であり、REA専用の固定Schemaを採用する決定ではない。

**三要素分析の結び方**：X-TEAの Task→Action→Function→IOTable→Table のような参照連鎖は、元の要素ID・向き・関係種別・出典を保ったまま追跡する。画面上でTask→Dataへ短縮して表示しても、途中のFunction/IOと導出Ruleの出所を失わない。Function-levelのC/R/U/D、Action固有の作用、Business上の手作業・責任・完了条件は異なる。参照線があるだけで、Taskが当該Tableへ書き込む、Actionが成功した、Businessが完了した、という命題へ変換しない。GraphのSemantic Edge／Derived Edge／Presentation Edgeも区別し、編集時の逆写像は**対象Slotと元Source版の一意性**が証明できる範囲だけに許す。曖昧なEdgeや複数候補はread-onlyのReviewを続け、最初の一致先へ自動採用しない（#1124/#1125/#1126/#4487/#2497）。

**起動・実行の二つの異なる観測**：REAのEvidenceを取得するCLI/Studioの一件のReadは、実際に観測可能なTEI側Invocationである。一方、旧コードにあるHTTP Handler、Timer、Event Listener、TaskやState遷移は、まず**旧システムについて推定されたTrigger/Function/Effectの候補**であり、実RuntimeのInvocationではない。受理の際は、①契機と公開Entry、②入力・Guard・State、③Function/Rule、④Provider/外部部品、⑤EffectとOutcome、⑥Response/Recipient、の関係を順に確かめる。Process Lifecycle（ready等）と一件のInvocation成功は別。外部部品は契約/TODO/Mockでシミュレーションできても、実DB Commitや通知配送のReceiptとは区別する。文書や説明文は**受理された業務条件から**規則的に生成し、静的逆解析候補だけから確定した運用手順を作らない（#1158/#4495/#4666/#1125）。

**問いに必要な証拠の量を先に決める。** 「Function FがData Dを参照する」という局所の陽性主張には、当該Source/Relation/Attemptの実受領が支えになる。一方、「Fは他のDataを使わない」「Dは未使用だから消せる」「Businessが完了した」という否定・全称・効果に関する主張には、問い別にRequired SourceとConsumed Source、静的/動的参照、未ロードPlugin、SQL生成、RLS/LIMIT、手作業、外部Effect、部分取得や非開示を照合する必要がある。取得できなかった資料・非公開の参照・Coverageがpartialである範囲は、0件や不存在へ縮約しない。実Providerが返したRowsetの件数・値が一致しても、粒度・JOIN fanout・NULL/欠損・順序・時点・権限の差を見落とさない（#3262/#2586/#4487）。

**使う一巡と生成する一巡の連携**：REA Candidate Reviewは、Studioの「Workspaceを開く→固定Subject/Sourceを観測→Pin→根拠を確認→Preview→明示受理」の**使う一巡**に置く。構造候補の受理前後で、Dirty Buffer、保存済Source、Git/World Revision、Semantic Set、生成Artifact、実際に使われたArtifactを混ぜない。正式採用後に生成を要求する場合も、Builderの選択→実Input→Artifact→Stage1/Stage2→Windows実Consumerを別の資格として評価する。REAが読めたことをDesktopセルフビルドや生成UI実使用の代替証拠にしない（#4583/#2831/#4650/#4651）。

**追加するExpected観点（既存のExpected/Ownerに追記し、新しい試験正本を設けない）**：

| 検証 | 入力・反例 | 必須の判断・既存Owner |
| --- | --- | --- |
| **REA-XP-07 関係の意味保存** | 同名異ID、平行有向Edge、逆向き・複数端点、途中Function/IOを省いた短縮線 | 元の関係Chain/Rule/Sourceへ復帰可能。表示線からCUD・Effectを創作しない。#1124/#4487/#2497 |
| **REA-XP-08 不完全観測と否定命題** | Graph completeだがFunction/Business unsupported、未取得ページ、動的SQL、RLS/LIMIT、手作業 | Required(Q)の閉包を満たさなければ「不存在・未使用・削除可」にしない。#3262/#3394/#2586 |
| **REA-XP-09 旧Triggerと実Invocation** | 静的Handler発見、Timer未配備、Mock成功、Commit後通知ACK不明、受付済みで実処理未実行 | 観測候補・Trigger宣言・実Invoker・Effect・Outcome/Deliveryを別証拠とする。#1158/#4495/#4666 |
| **REA-XP-10 編集・生成の版境界** | 同一表示に別Source、A→B→A、Graph描画Edge→意味編集、Pin A再確認中の閲覧B、生成と実使用Artifact不一致 | Slot/Source版/Ownerが一意でない編集を拒否。別対象のDirty Bufferを変更せず、生成・実使用は別Receipt。#4515/#2497/#4583 |

**実装順序の再整理**：R3-A/BのDraft別HEAD技術合格と既存Parser/Hostの統合を先に整え、#4661でCLIとNative Studioの実Consumer受領を確かめる。REA-XP-07/08のHeadless意味保存・Source closure負例は**R3-Cと並行して**準備できるが、その成功を正式Meaning受理へ昇格しない。続いて「旧Function一件が参照するData」を問うReviewで、元Task/Action/Function/IO/TableのChainとTEI三要素対応を照合し、Ownerの明示選択後のみ通常Sourceへ反映する。その後にSHO実Save/ReopenおよびL（Legacy実測）/D（Owner期待）/T（Target実測）を独立に比較する。Studioの非Plugin Stage1→Stage2実Windowsセルフビルドと通常tei CLI→実Provider→Function Context→独立Consumerは引き続き**製品P0**であり、REA R3や三要素の全面整備をHard Gateへ加えない。

**この節の資格**：既存の各設計・実装・CIの確認から得た計画上の判断であり、REA-XP-07〜10、旧Targetの実Trigger、Native第二Consumer、Owner Meaning、旧新Function同値を新たに実行したものではない。主計画Draftはmain未統合のままレビューする。


<!-- rea-one-function-question-review-handoff-20261114 -->
#### 一機能の「根拠を確認して受理する」作業を最小の製品断面にする（2026-11-14／設計・実施引渡し）

**追加の判断**：REAが抽出したNode/Relationから、最初に作るのは新しい正式DSLや新しい業務モデルではない。**「旧Function Fは、どのDataを読み、どのDataを書き換えるか」への根拠付き回答**を、既存の型付きread-only Operationと三要素の参照・Source/Context契約を使って組み立てる。資料の位置、読めた範囲、対応が曖昧な部分まで表示し、利用者がSourceに戻って確認できることを優先する。一つの回答ができても「そのData以外は使わない」「旧機能を新Targetで再現できた」とは結論しない。

**回答を作るときの最小項目（既存Projectionを用途別に合成し、新しい保存Schemaは定義しない）**：

| 表示項目 | 含める内容・選択時の操作 | 合格の上限 |
| --- | --- | --- |
| 問いと対象 | Question、Workspace/Source版、旧Functionの候補ID、必要なData/Function/Businessの範囲 | 候補IDの一致は正式TEI Qualified Refとの同一性ではない |
| 確認できたこと | REA Evidence ID、Node/有向Relation IDと原Source位置、X-TEA宣言/SQL、観測・推論の区分 | 静的参照は実際のSQL実行、Business上の実行成功を証明しない |
| まだ確認していないこと | 未取得ファイル、動的参照、別Plugin/DB、手作業、partial/unknown、必要閉包 | 非表示・未取得・権限不足から「0件・不存在」を導かない |
| 対応候補と理由 | 外部Element↔旧Function/IO↔TEI Data/Function/Businessの0..N対応、支持・反証・曖昧な理由 | AIの説明、名前の一致、候補スコアはOwnerの意味受理ではない |
| 次に取れる操作 | 原Sourceへ移動、関係の経路を確認、候補の比較、必要な追加観測、変更Previewの作成 | 閲覧とPreviewでは実Source/DB/外部Hostへの書込みをしない |
| 独立した検証結果 | Consumer実受領、必要ならMock/実Provider/旧実測L・Owner期待D・新実測Tを出典別に表示 | Mock/Target結果をLegacy実測やOwner受理へ読み替えない |

**Studioの操作順**：対象を探す→**Source版と外部IDを固定**→候補と根拠を比較する→未解決・不完全を確かめる→必要なら採用候補を作る→変更先のQualified Ref/Source Slotと影響をPreviewする→Ownerが明示的に判断→Sourceを保存して独立再読取・再投影する。初期のREA R3-Cは**根拠を読むところまで**を実装する。Focusは一時的な閲覧位置、Pinは対象固定、Dirty Bufferは未保存内容であり、一つにまとめない。候補のReviewや見かけのSource一致だけで編集可能にしない。複数の元Occurrenceが同じ表示要素へ集約された場合、編集先が特定できない限りread-onlyのまま理由を示す。

**説明文の扱いを明確にする。** 説明の生成元を①旧ソースから直接確認できた事項、②解析やAIによる解釈の候補、③Ownerが受理した業務条件、④Mockで確かめた振る舞い、⑤実Host/DBで観測した結果に分ける。日本語の業務説明や仕様書へ正式な「する／しなければならない」を記載するのは、③の意味に根拠がある部分だけとし、①②は「コード上の参照がある」「候補である」、④⑤は実証した環境と範囲を明示する。専門家が読める業務DSLから説明文・テストパターン表・TODO外部部品のMockをほぼ直訳できる粒度を目指すが、逆解析の結果を自然な文章に整えたこと自体で承認済み仕様とはしない。出典がない手作業、Business上の責任、完了条件を補って確定文へしない（#2622/#1125/#1158）。

**三要素・DB移行との接続**：Function-level C/R/U/Dを認定するには、単にSQL/ERのRelationが見えることではなく、関係を辿った根拠と対応するFunction/Action/IOのSourceを示す。旧Source・新Query・DB Instance・Provider・入力パラメータ・取得範囲は別の束縛とし、特にJOINの行数増幅、NULL/欠損、RLS/制限付き取得を同一の意味へ潰さない。読み取りが成立しただけなら更新系のEffect/Recoveryを認定しない。通常利用者にはFunction/業務Dataと現在できる操作を先に示し、DB Alias/方言/実DSNは必要かつ開示が許される場面だけへ配置する。外部REAの対応候補を使ってDesired Schema/DB DDLを自動更新しない（#1126/#2586/#2622）。

**既存独立Expectedに引き渡す3つの小さなケース**：

| ケース | 正常・負例の入力 | 求める判断（実装結果ではない） |
| --- | --- | --- |
| **REA-XP-11 説明の出典** | 静的な参照経路はあるが業務条件の根拠がない。AIは完了条件を含む流暢な説明文を提案 | 出典のある構造と意味候補は表示可能。ただし未受理の完了条件を正式な説明・DSL・試験Oracleに採用しない（#1124/#1125/#2514） |
| **REA-XP-12 一問の回答の忠実度** | 同名Functionが二つ、平行Edge・未取得Pluginあり。短縮図でTask→Dataに見える | 元の連鎖・Sourceへ戻れ、対応候補と未解決が残る。単一Function/Writerの確定や「Dataは未使用」の誤断定をしない（#1124/#4487/#3262） |
| **REA-XP-13 Preview後の差替え** | 候補の採用Previewを開いたままSource版/Slotが変更。元byte A→B→Aや別Tab後着も起きる | 実際の変更は現在版・対象の一意性・Owner判断を再確認してから。古いPreviewを黙ってApplyせず、無関係な閲覧状態やDirty Bufferも変更しない（#2497/#4515/#1126） |

**実装・検証を分割する**：#3394/#4661はEvidenceを実CLI/Nativeへ届ける責務まで、#1124/#4487は一Functionの根拠付き三要素Reviewと文書/図面への投影、#1125/#1158/#4495は受理済み業務プロセスの説明とTODO/Mock、#1126/#2497は意味を落とさないSource編集、#2586/#2622は実Provider/Context、#4317/#4467はSHO保存・再読取、#2514/#1180は独立Expectedを扱う。**REA-XP-11〜13のHeadless FixtureはR3-Cと並行して設計可能**だが、正式Native受領・Owner受理・実移行の達成とは別に記録する。R3-A/Bを未統合のまま二Consumerを成功扱いせず、対象が未確定ならFit/Gapだけを既存Ownerへ返す。

**製品P0と現況**：非Plugin StudioのStage1→Stage2 Windows Desktop実セルフビルド（#4583/#2831）、通常tei CLIからの実PostgreSQL Provider Readと独立Consumer（#2586）を優先する。今回のREA-XP-11〜13は**設計と試験引渡しのみ**で、Headless/Native/旧Runtime/実DB/SHOの新試験やOwnerの意味受理は行っていない。新しいREA専用Host、承認Engine、Meaning Registry、共通Editorや第二Expected正本を導入しない。

### 2026-11-13／REA受領のmain統合後：実Consumer・参照完全性・Desktop優先の受入順（#3394）

**実装済みと未実証を区別する**。外部逆解析REAの版別Protocol検査（#4257）、外部EvidenceのReceiver／Candidate Reader（[R1 #4547](https://github.com/hat47x/tei-platform/pull/4547)）は**main統合済み**。後段の型付き`operations.ExternalCandidates`（[R2 #4548](https://github.com/hat47x/tei-platform/pull/4548)）は2026-11-13確認時点では**open、main未統合**であり、選択されたObservation 1件＋有向Relation 1件の実Go試験や固定Evidence ID／raw digest／producer-declared Source Revisionの一致を、CLI・Studio・独立Harnessが実受領した証拠と混同しない。旧#4322の大規模差分は丸ごとマージしない。R1のReaderが扱う外部REA Observation/Relationは、三要素分析法やResource–Event–Agentという別概念へ無条件に写像しない。

**三要素分析法の基底修正を再利用**：[#4513](https://github.com/hat47x/tei-platform/pull/4513)のData索引、[#4534](https://github.com/hat47x/tei-platform/pull/4534)のCRUD矛盾の列挙と`ok`の資格分離、[#4551](https://github.com/hat47x/tei-platform/pull/4551)の明示CRUD-only組の投影・`UnusedData()`補正、[#4507](https://github.com/hat47x/tei-platform/pull/4507)／[#4524](https://github.com/hat47x/tei-platform/pull/4524)のFingerprint純粋性／同種ID衝突、[#4514](https://github.com/hat47x/tei-platform/pull/4514)のDiagram Source-only drift検知はそれぞれ限定範囲でmain統合済み。従来の「#4472/#4475はすべて未修正」という読替えはしない。ただし実Studio／SHO／AI Consumerでの行・矛盾・unknown・参照件数の同値、未使用確定、編集資格までが実証されたわけではない。

**次に進めるR3実Consumerの条件**：#4548が最新HEAD・必要CI・Source範囲を固定してmainへ受理された後、既存共通read-only Operation（#1158/#3419）を通して、(A)実CLIと(B)別プロセス・別入口のStudioまたはMCP Consumerが**それぞれ実際に同じ限定候補を呼び出し・受領**したことを観測する。CLI JSONの再印字／同じGo単体試験を別名で実行したものは第二Consumerでない。元raw byte／受領digest／Evidence・Tool版／Attempt／Source位置・外部Node・Relation ID／edge方向／Coverage・Unknown・Unselectedを照合する。CLIでのローカル`EvidencePath`受領をStudio／Harnessへ任意Pathアクセスとして広げず、既存Workspace Root allowlist・限定Fetch／用途別Disclosure（#4503、#2516）に従う。**Pinは選択意図であり、元Sourceの現在性・Host Grant・外部送出許可・正式意味受理ではない**。Pin不一致、Source改版・同Root名別実体・A→B→A、同名別ID、平行辺・ID重複ではsilent retargetを拒否する。

| 既存受入ケース | R3で要求する独立Expected | 成立しない主張 |
| --- | --- | --- |
| XS-08／REA-STORE-01 | CLIと独立Consumerが同じ保存Evidenceから選択1+1を実Callし、同じraw digest・Attempt・外部ID・方向・unknown/Unselectedを受領。実受領の記録は入口ごとに別 | JSON表示一致だけで二Consumer実Call完了 |
| REA-REF-01/02/03 | 参照元/先と件数にSource Kind・Scope・Revision・Coverageを併記。0件の`unused`確定は問合せ対象のRequired Dependency Closureがcompleteのときに限る | REAのpartial/権限外/Plugin未解析を参照なしと断定 |
| REA-REF-04／REA-X-05 | 元Source変更・同一表示別Root・同名別ID・ABAでは古いPin/後着Replyをcurrentへ昇格しない | digest/モデル名一致だけで現在の編集・送出資格 |
| REA-REF-05／REA-X-04 | 修正済み三要素Productの明示CRUD-only・矛盾・同Kind ID重複を実Consumerへ流し、描画用`parent:`辺を正式Relation Pinとして採用しない | 1+1候補を業務CRUD・全Graph・Accepted Meaningと扱う |
| REA-MAP-03/05/10 | 外部観測にない列をTEI定義から削除せず、正式Field削除の影響PreviewとSHO著者編集を別々に保護する | External Candidate／DB DDL／文書再投影の自動成功 |

**製品優先順位**：最優先はTEI Studio Desktopの実Wails接続とTEI自身のセルフビルド（#1824/#3421）。#2586の実Function→Provider→Rowset ReadとScript起動のpure Entry実証も並行P0であり、REA R3／SHO／全外部Graphの完成をHard dependencyにしない。R3は既存Desktop経路のRead-only Consumerとして段階的に足し、新Host・第二Parser・専用意味受理Engine・Evidence正本を増やさない。これらのR3 Product E2E、Source現在再読取、実Storage送出・Grant、Owner意味受理は**本追記時点でNOT RUN**。実装担当・試験実施・結果更新は#3394、#3788、#4515、#4472、#4503の既存Ownerに戻す。

### REA受領後の実Consumerを最短で成立させる（2026-11-13、#3394×#4547×#4548×#1124）

**実装済みと未受理を更新する。** [REA Receiver / Candidate Reader PR #4547](https://github.com/hat47x/tei-platform/pull/4547)は**main統合済み**。REA 6.7.0/6.8.0の保存済み実捕捉Synthetic Evidenceを版別に読み、元raw byteのdigest・Source Revision・producer・Coverage/Unknown/Partial・候補Graphをread-onlyで保持する。[型付きOperation PR #4548](https://github.com/hat47x/tei-platform/pull/4548)はmain向け5ファイルへ縮小済みだが**open/unmerged**で、受領済みEvidence ID／raw digest／producer申告Source Revisionに対するExpected Pin不一致と重複Node/Relation/派生handle衝突をfail-closedにする限定実装。旧[PR #4322](https://github.com/hat47x/tei-platform/pull/4322)の42ファイルを一括統合しない。いずれの成功も元Sourceの**現在のbyte/Root/Revision/ABA**、外部Providerの実起動、現在Grant、正式TEI Meaning、全Graph取得、実DB Effectを証明しない。

**次の独立Product Exit（R3）**：#4548の最新HEADに対する独立Go／全体Healthを確認してmainへ受け入れた後、共通read-only Operationの**同一選択対象**を少なくともCLIと独立したDesktop/Studio Consumerの二入口で実受領する。利用者が固定した対象（Workspace・Source/Revision・外部Node/Relation ID・Occurrence・Scope）と、実際に受信した候補集合・digest・partial/unknown・診断を別観測にする。GUIで結果を再描画するだけ、同じCLIのJSONを再表示するだけでは独立Consumerに数えない。Studioの描画用Edge IDを正式Relation IDや外部Producer IDに読み替えず、Subject Pinは[Issue #4515](https://github.com/hat47x/tei-platform/issues/4515)のread-only範囲に止める。

| Case | 独立Expected（次の製品検収。未実行） |
| --- | --- |
| REA-R3-01 | 6.7.0単独JSON／6.8.0 JSONL終端のraw byteを別々に受領。各Version/Attempt/元byte digestと1 Observation＋有向Relation 1件・Unselected・unknownを保持し、相互に版を取り違えない |
| REA-R3-02 | 同じ固定SelectionをCLIと独立Studio Consumerに渡したとき、Qualified ID・向き・平行辺・Coverage・診断が一致。実Consumerが受け取ったbyte/Projectionと受領Outcomeを別に記録 |
| REA-R3-03 | 同ID重複、派生handle衝突、改版後の旧Expected Pin、同内容A→B→A、別Workspace同表示名では自動後勝ち・再束縛をしない。保存済みraw digest照合だけでは現在性のPASSを出さない |
| REA-R3-04 | partial／permission-denied／Source未取得／dynamic reference／非開示では、参照確定0と未使用確定を区別。Confirmed／Candidate／UnresolvedとScope/Required Closureを別列に表示し、正式CRUD／Meaningに加算しない |
| REA-R3-05 | Raw EvidenceのCAS/NAS/S3内部Locatorを外部Harnessへ暗黙送信しない。取得・現在Disclosure・元byteと伏字後のdigest・実Deliveryは#4307/#4503/#2516で独立検収 |
| REA-R3-06 | REA Candidateから正式三要素分析法のData/Function/Business・SHOへ提案しても、明示採用までは正式参照数・業務CRUD・Source編集・実DB DDL／実行権限が増えない |

**並行性と責務**：R3のProduct入口は#1158/#3419、正規化と外部Graphの出典は#3394、Editorの高速参照移動は#3788、正式三要素分析法のModelerは#1124、Evidence Storeと用途別開示は#4307/#4503が所有する。最優先の**Studio Desktop起動→TEI自身によるセルフビルド**（#1824/#2831）と実Function Provider Read（#2586）は、REA R3/SHO/三要素分析法の全件対応が未完でも進める。Go Receiver合格とCLI/Studio実E2E合格を混同せず、新Host/Registry/承認Engineは増やさない。本節は計画追補であり、R3 Product E2E、外部AIへの実送出、Owner受理は**NOT RUN**。

### REA製品入場の更新と6.9.0再資格化（2026-11-13、#3394）

**統合状況を更新する。** 横断計画[#4508](https://github.com/hat47x/tei-platform/pull/4508)はmain統合済み。外部EvidenceのReceiver、保存した実REA 6.7.0／6.8.0のbyte・digestによる読取り、Observation／Relation／Coverage／Unknownを扱う型付き候補Readerの最小移植[#4547](https://github.com/hat47x/tei-platform/pull/4547)もmainへ統合した。次段[#4548](https://github.com/hat47x/tei-platform/pull/4548)は、共通read-only `operations.ExternalCandidates`、同Kind重複ID・平行Relationの取り違え防止、および明示PinのExpectedEvidenceID／ExpectedReceivedDigest／ExpectedSourceRevisionを選択前に照合する**未統合の独立PR**である。Sourceの現在性、Workspace/Grant、独立CLI・Studio・MCP Consumerの実受領は別資格とする。元[#4322](https://github.com/hat47x/tei-platform/pull/4322)の大量差分を一括受け入れない。

**外部Providerの版差を継続して追う。** REAの[6.9.0公式リリース](https://github.com/morluto/rea/releases/tag/rea-agents-6.9.0)は保存Evidenceの部分取得、欠けたリンクの検出、ページング、解析対象の拡大等を公開した。これはTEIで6.9.0を実行・検証した証拠ではない。既存[#4257](https://github.com/hat47x/tei-platform/pull/4257)の6.7.0単独JSON／6.8.0 JSONL終端の合格を6.9.0の互換性へ流用せず、#3394のT0-06で固定実版のraw byte／元Source版／選択件数と未取得／metadata／終了状態を独立Expectedと照合する。選択取得・ページングは**取得済み部分集合の精度を高める可能性があるが、全Graphの完備性・現在の利用権限を保証しない**。6.9.0の実ツール・Product Consumer検証は未実施。

**既存の参照検索計画を前提に進める。** 直前のREA-REF-01〜07を新しい索引契約として重複実装せず、#4472のCRUD明示定義脱落と索引誤用、#4475の同Kind重複ID・Fingerprint純粋性の回帰をP0とする。その後、#4548の外部候補とTEI正式モデルの参照集合を**別種の結果として並列に観測**し、#3788の通常Editorと#4515のStudio Subject Pinから同Scope・同Revisionの一致／unknownを確認する。外部Relationが観測されたことは業務Relationの受理、参照件数0は全域未使用、選択Pin成功は現在のDisclosure/Grantを意味しない。未実装・未実行のProduct E2E、REA 6.9.0、Owner受理は**NOT RUN**。

**検討中の横断追補（DBアクセス・Query/2way SQL）**：対象DataSourceの既存Binding解決、DB別Native SQLの選択規約、SQL編集時の診断と分岐検証、異種DBでの意味同値性を、[DBアクセス・Query選択・2way SQLの最小適合計画](https://github.com/hat47x/tei-platform/issues/2622#issuecomment-6772875802)で小さな実証へ分解する。#2611・#2622・#2387・#2667に既存の責務を残し、新しいCore API・正式DSL構文・実DB対応の採用・完了を意味しない。
**2026-11-09／Product Queryの物理入力乖離を解消（独立Source Fixture段階）**：[Workbench設計§22](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)と[隔離PG Product Fixture](../examples/data-source-workbench/README.md)に、既存`postgres_seed.sql`の`sales_order`4列と、実`order_entry.recent_orders` SQLが要求する`public.orders`5列＋`:from`の**物理Schema不一致**を反映した。従来F-01〜10を壊さず、専用`recent_orders_product_seed.sql`を別Sourceとして追加し、3行／境界一致2行／後方0行の独立Source-static ExpectedをGoで判別する。**これは実PostgreSQL接続・Provider Invoke成功ではない**。最初の未成立実行辺は、固定Function/Queryの実Consumed SQL+args→Alias/物理Instance/Provider→実Rowset/Function Data Context。Ownerは#2586、実ProductのCLI/Studioは#1251、Schema適用は#2892。main #3540の静的Inspector S2-referenceは実CLI/Wails経路まで前進したが、外部Plugin Build/Grant/2×2実BOMを証明せず、異質JSONLは後続比較とする。新Gate/Host/Query正本なし。


### XTEA browse実Readのmain欠落を解消する（2026-11-12、#2586×#3270）

[Workbench設計§24](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)で、未merge終了の[#4076](https://github.com/hat47x/tei-platform/pull/4076)が「mainへ既存取込済み」と判断されたことと、現mainの**実際のSource存在**を分離した。実際には旧PRにある`internal/driver/query_execution_environment.go`、`internal/driver/browse_parameters.go`、`internal/functioncontext/functioncontext.go`、`cmd/tei/cli_view_browse_read.go`の経路がmainで見当たらない。旧Branchの隔離PG 30行・eq/ne・2ページ・通常CLIの**旧版限定実測**を、main製品のPASSへ流用しない。さらに旧Branchの`FROM sales.order`は**現mainの修正済み`FROM sales."order"`を退行**させるため、24ファイル一括上書きではなく必要辺を現在の既存型/APIへ小分けに戻す。

**最初のP0実装**：一方のWorkspace（`reference/workspace`の`:from/public.orders` 3/2/0、または`examples/xtea-lowcode`の`status/sales."order"` 30行と2ページ）を指定し、Environment/Instance→明示Query Input→実Executor/Read-only PostgreSQL→Rowset→Function Data Context→通常CLI出力・独立Expectedを同一Attemptで検収する。各WorkspaceのSQL/Fixture/Grain/Roleは別。0行/最終Pageの`declared-page-contract`をグローバルな完全性・不存在へ昇格させない。正式PluginのBuild/Entry/Grant、Studio/Harnessの**固定対象選択と誤解除防止**、NAS/S3等の外部Evidence格納・Source Revision、Schema DDL History/Applyは**別Consumer/資格**として並列に進める。#2586が実Read、#1251/#951が正式Plugin、#1124/#1824/#1158がContext選択、#2600/#1163がEvidence、#2892/#2887がDDL、#2514が独立Expectedを担当。既存WB-01〜54・HJ/QXとRQ-035 PASを再利用し、新Registry/Host/Runtime/Query正本/Gateを導入しない。

**現状**：物理FK/Index read-only投影とProduct Queryの静的Seedはmainに存在。XTEA実Read経路の旧PR成果は**mainで未確認／上記Path不在**。今回の記載だけでは新Go実装・実PG/CLI/Studio・Owner受理は成立しない。


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


### 2026-11-11 開発環境・言語ワークベンチ横断研究の実証線（設計候補）

Eclipse Xtext/Sirius/GLSP、JetBrains IntelliJ/MPS、VS Code/LSP、Vim/Emacs系および代表的ローコード製品の比較を、**既存 [#3788](https://github.com/hat47x/tei-platform/issues/3788) の未保存診断・参照・安全な改名**の検証に集約する。外部IDEは、[#3419](https://github.com/hat47x/tei-platform/issues/3419) / [ADR-0048](adr/0048-shared-application-api-and-transport-adapters.md) の共通Operationと、[#988](https://github.com/hat47x/tei-platform/issues/988)のLanguage Plugin/Source解決を使う**薄いAdapter候補**とする。Studioは[#1824](https://github.com/hat47x/tei-platform/issues/1824)のWails第一候補を維持し、LSP/GLSP/MPSを新しいSemantic正本・常駐Host・標準Editor Frameworkとして採用しない。

**順序と最小証拠**：①実際にReaderが読める一Source/一Ruleについて、CLIのSource位置・診断と未保存Bufferの型付き部分解析の成立範囲を確認（不成立なら#988へ戻す）→②実LSP/Editor Client一件で同じ対象の診断・定義参照を比較（LSP自体は採用判断前、`tei lsp`は構想）→③同名異ID・UTF-8/UTF-16位置・CRLF・旧応答・部分参照集合・外部変更を負例として改名Previewを比較→④可逆な場合だけ[#1124](https://github.com/hat47x/tei-platform/issues/1124)/[#2497](https://github.com/hat47x/tei-platform/issues/2497)/[#2500](https://github.com/hat47x/tei-platform/issues/2500)の既存Source編集・投影へ引渡し、Source再読込とConformanceを照合する。GUIやLSPの表示成功はAcceptance/Provider Effect/実受領ではない。

**通常コーディングとの関係**：Model-first/Code-first/Hybridを並存させ、手書きのGo/Java/.NET等は明示的な型/Effect/Source契約で接続する。生成コードの改変をモデルへ自動逆変換しない。外部IDEでの開発手数と、ドメイン専門家のStudio/CLI利用時の手数は[#2018](https://github.com/hat47x/tei-platform/issues/2018)/[#1168](https://github.com/hat47x/tei-platform/issues/1168)で同一の業務変更・独立Expectedから比較する。

**優先順位の停止線**：この研究をP0実Function→Query→Provider Read [#2586](https://github.com/hat47x/tei-platform/issues/2586)、CLI/Studio実選択PluginのProduct S2 [#1251](https://github.com/hat47x/tei-platform/issues/1251)、[#1119](https://github.com/hat47x/tei-platform/issues/1119)のGateの前提に追加しない。第二Parser/Resolver/Registry/Host/Policy/Scenarioを作らず、正式`*.tei.md` Reader未導入・未取得Sourceを「読めた」としない。現時点は**研究統合・設計/受入案**であり、LSP/実Editor/Studio逆編集/実DB/Owner意味受理の新規実証は **NOT RUN**。


### 2026-11-07 Plugin-first D1：実CLI／Studioの最初の不足辺を固定する（実装・受理前）

[main統合済みPR #3277](https://github.com/hat47x/tei-platform/pull/3277)の[Plugin-first移行計画](design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md) D0〜D6と、[#1016の構成→実使用Witness](https://github.com/hat47x/tei-platform/issues/1016#issuecomment-6704666016)、[#1251の実Go D1引継ぎ](https://github.com/hat47x/tei-platform/issues/1251#issuecomment-6704672969)を、**作業領域Gの一つの実装切り口**として接続する。main統合[#3300](https://github.com/hat47x/tei-platform/pull/3300)のD1 Fit/Gap（AS-01〜12）と[#3367](https://github.com/hat47x/tei-platform/pull/3367)の追加条件（AS-13〜20）を検収正本とし、[Desktop Draft #3352](https://github.com/hat47x/tei-platform/pull/3352)および[mainのPlugin-first設計§16](design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)（旧Draft #3369は閉鎖・未統合）と重複する仕様を増やさない。既存#1119のG0〜G4／B/C判定を変更しない。

**コード上の出発点（2026-11-07、GitHub mainの静的照合）**：
- `internal/operations/inspect.go` の `InspectSubject(root,subject)` はWorld/Source Indexのread-only投影で、`worldFingerprint`、`sourceIndexFingerprint`、`sourceStatus`を返す。ただし**Required Domain ClosureがすべてLoaded/Parsed/Boundされた証明**でも、expected Revisionでの実行前CASでもない。
- `cmd/tei/cli_commands.go` の `inspect`は手書きの受付一覧にあり、`cmd/tei-studio/main.go` の `StudioBridge.InspectSubject`は`internal/studiohost.ReadOnly`へ委譲する。現時点で**両方が一つのexact選択ProviderをComposition経由で実呼出しした**とは観測されていない。
- `internal/composition`の`Contract{ID,Version,Shape}`／`Provider{ID,Scope,Offers}`／`Selection{Revision,Epoch,...}`は宣言上の候補・選択に使えるが、現Host Grant、実Builder Source閉包、実Provider Callbackの資格そのものではない。
- [#3247](https://github.com/hat47x/tei-platform/pull/3247)はWorkspace名・manifest照合情報の限定CLI応答をmainへ統合済み、[#3284](https://github.com/hat47x/tei-platform/pull/3284)／[#3296](https://github.com/hat47x/tei-platform/pull/3296)／[#3354](https://github.com/hat47x/tei-platform/pull/3354)はDesign Contextの限定試験を段階的にmainへ統合済み。ただしWorkspace名は一意のRoot/Source identityではなく、Contextの実11 Replyと32判別ControlはRequired全域の同一原子的Snapshot／全業務Meaning／Plugin実Invokeを証明しない。

**P0の対象を一つに固定**：D1の候補をまず**既存`inspect`**とする（`model catalog`は比較対象であって同時に置換しない）。変更内容は次の不足一辺だけを解くこととし、汎用Plugin Manager／新Registry／新CLI構文／Studio用意味Service／第二Scenarioを先に作らない。

| 順 | 実変更・観測点 | 独立した合否／停止線 |
| --- | --- | --- |
| **P0-a: Source/Context基線** | 1 Workspace、1 Subject(kind＋qualified ID)、1 Source版。CLI実コマンドとStudio allowlist内のWorkspace ID→信頼済みRootを同じ対象へ束縛する | `worldFingerprint`値一致や`workspace.name`一致だけで同じWorkspace・同じSource集合・現在Grantとみなさない。Root実体/Manifest/Source occurrenceを保護された内部Evidenceで照合し、開示先には#2516で許可された情報だけ投影。Required Closure不足は#3262へ返す |
| **P0-b: exact ContractとProviderの限定導入** | 既存`InspectSubject`を一つのread-only Typed Service Adapterとして提供し、#951/#1016の既存解決・選択を経由させる | Providerなし／二候補曖昧／Contract shape・scope不一致／旧epochを拒否。Referenceの`Registry.Select`だけを実Call・Conformance・現在Authorityに昇格しない |
| **P0-c: 2つの実Consumer** | CLIの`inspect`とStudio `StudioBridge → studiohost`から、選択Providerの**実Invocation**へ到達。read-only旧経路とのshadow比較後、切替対象に暗黙fallbackを残さない | CLIがJSON化した結果をStudioが表示しただけでは「独立2 Consumer」にならない。実CLI Target／Studio Targetそれぞれのentry、Selection、Provider実Call、Input/Resultを別Attemptで照合。Wails Desktop E2Eは実起動を伴う別資格 |
| **P0-d: 不整合と実Build** | B selected→B invokedの正例、B selected→A invokedの負例、未選択AのBundle同梱と`init`混入、Root/Source/epoch変更、途中切断を既存テストへ接続 | 値が等しくても誤Providerなら不合格。Build成功・実Source members観測・Binary含有・Callback実行を別Claimとする。#3217の外部Module member観測（qualified=false）をG4のSource Closureへ転用しない |
| **P1: 同じConsumerを用いた展開** | #3262設計Context／#3270 Data Workbench／#3267 Verified BOM／#1124 Graphと三視点を**後から**同じOperation Contractへ適用 | D1 `inspect`のread-only PASSをDDL、外部Host、実DB Read、BOM組合せ、Owner意味受理へ拡大しない |

**特に混同しない2つの「2×2」**：D1でいうCLIとStudioは**二つのSurface/実Consumer経路**。#3267のBOM-1が検証する「Provider二版×Consumer二版」は**同じSurface二種類の試験ではない**。BOMの4 exact tupleには版・Target・Artifact・Independent Expectedとactual Invocationを独立に記録し、CLI/Studioが同じ結果を示しただけでは4組合せをVerifiedにしない。未実施／SKIP／途中切断は不合格・未知として残す。

**最初の実行成果物**：#1251／#1158／#1824で`inspect`への実Provider経由の入口を最小変更し、#1180の既存Scenario＋#2514の独立Expectedで上表P0-a〜dの正負を検査する。#3262はRequired/Loaded/Parsed/Bound/ReturnedとTool完全受領の照合、#951/#1016はexact契約・選択、#943/#1090/#2831は物理Build/Source資格、#2516/#2515は現在の公開／送出資格をそれぞれ所有する。**一つの列が欠けても他列の成功で補完しない。** 固定命令の削除やCore分割・Trait新Schema採用は、ここで実Consumerの不足と変更量を測ってから判断する。

**現時点の資格上限**：上記は実コード・既存PRを読んだP0実装順序の統合。新しいCLI/Studio共通Provider、実D1 Target、CI、実Wails、Source Closureの正式資格、Host Grant、業務Owner意味受理は**NOT RUN／NOT PROVEN**。自動的なmain移行・正式採用を示すものではない。

#### 2026-11-08 横断更新：Capability選択の実証と実Consumer資格の間を分ける

既存の実Referenceから観測できた前進を受入基線へ採用するが、**正本の検証規則・実Consumerへの接続は未完成**として区別する。

| 成熟した実証（main／別HEADの適用範囲） | これを根拠にしてよいこと | 追加で必要な反証 |
| --- | --- | --- |
| [#3337](https://github.com/hat47x/tei-platform/pull/3337)→[#3350](https://github.com/hat47x/tei-platform/pull/3350)：Capability Referenceで宣言12／未選択6→4（選択ルール改訂に伴う比較値）。DB instanceの技術名と能力IDを等号で比べず既存`SupportsDatabaseProvider`を使用 | **判定規則の変更で同じSourceでも結果が変わる**。選択されない能力は直ちに欠陥ではない。Catalog/判定器の版をEvidenceに明記 | 宣言Plugin → Capability Kind/ID → Environment binding → exact Contract/Provider Selection を同一視しない。[#3363](https://github.com/hat47x/tei-platform/pull/3363)はmain統合済みの限定参照であり、24 Binding中22 selected/2 missingと報告した。Workbench feature 2件は現Product Validatorと異なる探索範囲であり「製品Error 2件」ではない |
| [#3354](https://github.com/hat47x/tei-platform/pull/3354)：実CLI 11 read-only Reply／Consumer 32局所判別と限定独立3 Rule／Canonical／Source Index | 既存Context Gateが、正常JSON・同FPでも必須Rule/Relation欠落を有限範囲で拒否できる。CI上の限定合格を次のfixture設計へ使える | 11のCLI起動は**単一Snapshotの原子的取得ではない**。Required Universe、Loaded/Parsed/Bound、Source/Grantの時点差、受信Sink、外部Harness Tool Reply完全性は#3262/#2509で別途検証。qualifiedTestGeneration=falseを保持 |
| [#3333](https://github.com/hat47x/tei-platform/pull/3333)：`tei commands view`に`writeFlags`、実File/SQLite作成の正負実測 | command名の`read-only`広告と、引数Variantによる**ローカル出力作用**を区別できる | D1 `inspect`をread-onlyで開始しても、将来のTool/Plugin/CLI wrapperにおける出力・Session Log・外部Egressは別Effect/Disclosure。単なる`effect:read-only`を「書込み・送出ゼロ」に拡大しない |
| [#3336](https://github.com/hat47x/tei-platform/pull/3336)：子Build driverのinvoker/invoked実行ファイルdigestと子成果物再Read一致 | Build駆動主体と子実行物を分離して観測する実Reference。Stage2成果物の子報告だけを信用しない | これはPluginがLink/InvokeされたこともL3 TEI self-build成立も証明しない（l3Claimed=false）。選択Pluginの実Build closure・init・entrypointは#1251/#2831で別試験 |
| [#3327](https://github.com/hat47x/tei-platform/pull/3327)／[Desktop計画Draft #3352](https://github.com/hat47x/tei-platform/pull/3352)：Workbench/frontendの実UI受入 | Studioに`/`のPoC Aと`/workbench/`の別Frontend/RPC Consumerがあること、および別HEADでNode20/20等の局所試験を確認できる | 別HEADではWails埋込asset byte不一致が残る。D1で**実際に選んだFrontend ArtifactとWails起動時のRPC/Provider**を照合し、BridgeのGo単体試験だけで実Desktop二Consumer合格にしない |

**D1実配線の最小順序（main統合済み#3300/#3367のD1a→D1bを維持）**：①現`tei inspect`と`StudioBridge.InspectSubject`の入力RootをHost側で確定→②Required/Loaded/Parsedの用途相対不足を表示→③#951のexact Contract/Provider Scope選択を通す→④Host-owned Adapterから実Providerへ到達→⑤それぞれの実Target/Artifact/AttemptのInvocationを確認→⑥旧直接経路を切替対象から外して再試験、の順にする。[#3300](https://github.com/hat47x/tei-platform/pull/3300)で確認済みの旧Service Carrierが`application-runtime`以外を拒否する仕様を**文字列置換で迂回しない**。Authoring Host Adapterは既存`studiohost.RootServices/ServiceFactory`の信頼境界を利用し、Renderer/AIからTrusted RootやSelection Authorityを受け取らない。将来のMCP/生成Runtimeは同一Contractで別Consumer/Grant/Target資格を得る。

**追加の独立検収義務（既存Scenario/Expectedへの観点追加であり第二SSOTではない）**：
- **PGD-01：宣言とEnvironment選択**。一つのPluginが複数Capabilityを提供、ある一Kindのみが選択される／未選択でも正常、技術名だけのDB選択一致を拒否する。実Providerに到達したとは報告しない。
- **PGD-02：曖昧・Validator非対称**。同Bindingの適格Provider二件は`ambiguous`、0件は`missing`を候補として記録する。ただし参照集計の`missing`を既存Validatorの正式拒否に昇格させず、対象・種類・規則版を照合する。
- **PGD-03：選択と実呼出し**。Provider Bをexact選択→CLI・Studio双方でB実Callの正例。B選択・A実Callは結果一致でもFAIL。Registry選択成功、同一メソッド共用、Bundle同梱のみならunknown。
- **PGD-04：Source/受領と版**。Workspace名／WorldFP一致でもRoot・Required Relation欠落、異なるIndexモード、11返信の間のSource ABA、Tool Reply途中切断があればContextをcompleteにしない。公開先のSecret・件数は#2516に従う。
- **PGD-05：実Desktop**。CLI実Targetと`/workbench/`実Frontendの各call-site、Wails embedしたassetのbyte、UI ABA、RPCの実Provider Invocationを独立照合。Go Bridge単体や別HEADのNode Testは製品E2Eの代替ではない。
- **PGD-06：副作用とBuildの別資格**。read-only入力/argvと出力Sink、未選択PluginのGo import/init、子Build実行物と生成Artifactを分離。Plugin Build/Runtime資格、Verified BOM 2×2（#3267、[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)の旧RQ-036案（現行ではRQ-037へ再採番必須））、Host GrantとOwner Meaningはそれぞれ別欄のまま。
- **PGD-06：副作用とBuildの別資格**。read-only入力/argvと出力Sink、未選択PluginのGo import/init、子Build実行物と生成Artifactを分離。Plugin Build/Runtime資格、Verified BOM 2×2（#3267、[旧Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)のBOM採番旧RQ-036（現行RQ-037）候補）、Host GrantとOwner Meaningはそれぞれ別欄のまま。

**Owner／完了線**：#1251/#1158/#1824がD1コードと2実Consumer、#951/#1016は正式Selection、#3262はContext必要集合とReply受領、#2514/#1180はExpected/Scenario、#2515/#2516はSink・Disclosure、#943/#2831はBuild、#3267はBOMを担当する。[main統合#3368](https://github.com/hat47x/tei-platform/pull/3368)はPlugin選択≠Capability選択≠一意Provider解決≠Required Trait履行として3つのReference負例を追加したが、Go実行・Trait履行の完了は未証明。#3278/#3279はRequired/Default/manual/hintと合成の義務のOwnerとし、Go固定Kindや新Trait Runnerを作らない。Graph/三視点/Function表現型（#1124/#2497/#2507）は同じ出典Factの別投影であって、それを実Selection/実Effectの証拠にしない。


**今回の計画から外へ委譲した詳細**：AS-01〜20の逐次・Provider／Scope負例は[main #3300/#3367のEvidence](evidence/PLUGIN_FIRST_D1_AUTHORING_ROUTE_FIT_GAP_2026-11.md)に帰属し、本節のPGD-01〜06はそれらを独立に再定義するテスト群でなく**横断の完了票**とする。原子的Traitの組合せ詳細とTC-16〜20は[main #3368](https://github.com/hat47x/tei-platform/pull/3368)、Provider×Consumer版Matrixは[#3267](https://github.com/hat47x/tei-platform/issues/3267)／[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)、CLI/StudioのOfferとFunction内表現Bindingは[main統合#3338](https://github.com/hat47x/tei-platform/pull/3338)、Studio/Wailsの実Frontend資格は[Draft #3352](https://github.com/hat47x/tei-platform/pull/3352)が保有する。複数Issue/PRの件数やPASSを一つのContract/Functionの合格へ足し合わせない。

**P0開始時の不変条件**：既存`internal/plugin/runtime/service_carrier.go`は`application-runtime`以外のScopeを拒否する。Tooling/Authoring Inspectorの実装は当該制約を緩めず、既存`composition` SelectionとHost-owned`studiohost` Seamの不足辺だけを修正する。画面がSourceを読めたことと、Function Data ContextやOwner Accepted Meaningを満たすことは異なる。

**当時の計画変更の上限**：PGD-01〜06を同一HEAD・同一AttemptのCLI/Studio/Frontend/Hostで検証したわけではない。なお後続[main #3440](https://github.com/hat47x/tei-platform/pull/3440)で**cmd/tei自身の限定3段Self-build L3実測**は成立したが、Plugin Bの実Link/Call、Build Source全閉包、現在のGrant、2×2 BOM、Owner業務受理への資格移転はない。実B呼出のProduct二Consumerは引き続き **NOT PROVEN**。

#### 2026-11-08 D1の実装差分：局所Provider Callから実二Consumerへ（最新mainに基づく引継ぎ）

後続で[ADR-0048（main #3439）](adr/0048-shared-application-api-and-transport-adapters.md)が採用され、CLI/MCP/API/Web/Desktopは**共通Application Operationの意味**とTransport表現を分ける方針になった。[main #3458](https://github.com/hat47x/tei-platform/pull/3458)は`validate`のCLI/MCP間での限定投影同値に成功したが、`inspect`のPlugin経由実行・二Consumerの資格を代替しない。

| 段階と責務 | 現時点の実装候補・確認対象 | 合格としない状態／次の不足辺 |
| --- | --- | --- |
| **S0 Typed Port**（#1158/#1824） | [CLI Draft #3381](https://github.com/hat47x/tei-platform/pull/3381)の`inspectWorkspaceJSONWithService`、[Studio main統合 #3371](https://github.com/hat47x/tei-platform/pull/3371)の`RootServices.Inspection`。指定Service欠落・出力契約不一致・未許可Root時はfail-closedを検査 | **Studio Hostの型付きService seamのみmain統合済み**（squash `080d162c`、Health #42470378877 success）。CLI側はDraftで、通常CLI `main.go`はBuiltin経路。Service注入だけでは実Plugin選択の証拠にならない |
| **S1 selected Provider（局所）**（#951/#1016/#1251） | CLI Draft #3381の`internal/inspectoradapter`が既存Registry/Selection/SelectedCoordinator/Host guardを使ってB選択→B実Go Call／B選択→A登録拒否／epoch/Grant失効を**局所in-process**で検査する候補 | 合成ID/digest・登録Functionだけでは配布Binaryの実Member/Entry/Conformanceを証明しない。正常なReturnが得られてもTarget実Entry・Authorityは別 |
| **S2 CLI/Studio実Consumer**（#1251/#1824） | HostだけがTrust Rootとexact Selectionを保持し、**通常起動のCLI dispatchと実Wails Studio Bridge**を同じ契約・各自のTargetへ接続する。[main統合 #3378](https://github.com/hat47x/tei-platform/pull/3378)（squash `0f3455e132d2146834d65089c68cae40c31f3055`）の従来Direct経路を6 Subject＋Not Foundで比較するGo Testは、exact-head Healthが成功した**移行前Baseline**としてのみ利用 | UIがCLI JSONを表示しただけ／factory nilでBuiltinへフォールバック／別revisionから得た同値JSONを二Consumer成功としない。B selected→A Callは同値ResultでもFAIL |
| **S3 Source・Artifact・Effect実証**（#3262/#2600/#3267） | [main #3482](https://github.com/hat47x/tei-platform/pull/3482)の限定Function Required（13実CLI Reply/42局所Controls）と[main #3477](https://github.com/hat47x/tei-platform/pull/3477)のSource-only失効を、実S2 Attemptで得たSource/Provider/Artifact/Scope/独立Expectedへ照合 | 13 Replyは原子的SnapshotでなくFunction実行資格もない。[main #3479](https://github.com/hat47x/tei-platform/pull/3479)の外部Provider raw/Manifestのみ改訂、[main #3490](https://github.com/hat47x/tei-platform/pull/3490)の選択規則fingerprint変更は追加の失効反例であり、Consumer World FPの一致だけでfreshにしない |

**2026-11-08 D1 CI検収の前方移植（現在のS0〜S3と分離）**。[main統合済み #3499](https://github.com/hat47x/tei-platform/pull/3499)のS0〜S3／D1B-ERR・TRF・CALLを変更せず、並行PRの**各checkoutと実Actions Job Log**から「Go Test PASS・Health FAIL・統合済みReference」を別々に記録する。

| 対象 | 試験済みの具体的なcheckoutと合否 | 現在の判断／次の一手 |
| --- | --- | --- |
| [Studio #3371](https://github.com/hat47x/tei-platform/pull/3371) | 旧HEAD `c7acfd6b`の[Health #42464668701](https://github.com/hat47x/tei-platform/actions/runs/42464668701)：全Go Test／実Workspace validate **PASS**、`gofmt` 2ファイルで全Health **FAIL** | 書式修正HEAD `86f9261e`の[再CI #42470378877](https://github.com/hat47x/tei-platform/actions/runs/42470378877)が**success**、squash `080d162c`で**main統合**。S0 Typed Service／Error秘匿のGo検査であり、実Selected BのStudio Bridge呼出しはS2の別条件 |
| [CLI #3381](https://github.com/hat47x/tei-platform/pull/3381) | HEAD `5cdd3b9e`の[Health #42463722188](https://github.com/hat47x/tei-platform/actions/runs/42463722188)：全Go Test／validate **PASS**、`gofmt` 2テストファイルでHealth **FAIL** | 同PR内でGo1.26 `gofmt`修正後にexact HEADを再実行。S1の局所B Go関数実CallはGo Test PASS、通常`cmd/tei/main.go`でBを選ぶS2は未接続 |
| [Imported Source #3479](https://github.com/hat47x/tei-platform/pull/3479) | 旧HEAD `fff65f0a`の[Health #42464022323](https://github.com/hat47x/tei-platform/actions/runs/42464022323)：Source／Manifest各改訂を含む全Go Test／validate **PASS**、`gofmt` 1テストファイルでHealth **FAIL** | 書式修正HEAD `6f0a9735`の[再CI #42470247004](https://github.com/hat47x/tei-platform/actions/runs/42470247004)が**success**、squash `99635b87`で**main統合**。独立Source×Manifest 2×2は外部Importの同一Atomic Load閉包／実Provider Readを証明しない |
| [BOM政策 #3490](https://github.com/hat47x/tei-platform/pull/3490) | HEAD `6b2e0b18`の[Health #42464016653](https://github.com/hat47x/tei-platform/actions/runs/42464016653) **success**、PRはmain統合済み | 選択規則RuleFingerprintと旧Evidence再利用の限定Go Reference。BOM Provider×Consumer版4組合せ／同一Attemptの実Build・Invoke・現在Grantではない |

**判断と実装順**：上記CI状態の変化を`S1 Go Test PASS`、`Repository Health PASS`、`S2製品実Call PASS`、`S3実Source/Artifact/BOM/Grant qualified`へ単一の合否で昇格させない。Studio #3371と外部Source #3479は修正後exact-head Healthが成功・main統合済み。CLI #3381は新HEAD `602ddc30`のexact-head CI／main追随・衝突判定を残す。次に既存#1251/#1824の**通常CLI Dispatch→実Studio Bridgeにおける同一authoring Provider Bの実Call**を独立Expectedで検収する。部分Required・Imported Source／Manifest・Policy RuleはS3の用途別再資格化へ委譲。新Host／Resolver／Runner／Trait固定Enum／第二Scenarioを足さない。本追記はdocs-onlyで製品E2E、実PostgreSQL、実外部Harness、Owner意味受理はNOT RUN。
**次の一件をS2の実装と検収に固定**：同じ公開`reference/workspace`・同じ`semantic-data:sales.order`・exact`authoring-tool` Contractを用い、(a)CLI通常dispatch、(b)Studio Hostの実Bridgeから、それぞれ実選択BのGo関数へ入ったことを**別のCall-site記録**で確認する。BをAへ取り違える／Serviceなし／Source ABA／Host Guard失効／部分応答の各負例では、旧Builtinへ自動fallbackせず診断を返す。公開されるCall証拠は受信先の権限と#2516のDisclosureに従い、内部Root/Secretを誤開示しない。実バイナリとWebViewの適格性はS3で別に測る。

**予定の帰属**：S0はStudio main #3371／CLI main #3381、S1はmain #3381のComposition選択B→Host登録Bへの局所実Go呼出とCLI/Studio Host API二経路試験（製品S2ではない）、S2のProduct統合は#1251/#1824、S3のDesign Requiredは#3262、履歴Sourceと再利用判定は#2600、版Matrix/BOMは#3267、Trait実装義務はmain #3480/#3278、Scenario/独立Expectedは#1180/#2514。新しいResolver/Registry/Host/Scenario SSOT、HTTP常駐必須化、Legacy `application-runtime` CarrierのScope緩和は行わない。本節自体は計画の更新で、S2/S3の実試験成功・Owner Meaning受理を**主張しない**。
**予定の帰属**：S0はStudio main #3371／CLI Draft #3381、S1はCLI Draft #3381の局所実Go試験、S2のProduct統合は#1251/#1824、S3のDesign Requiredは#3262、履歴Sourceと再利用判定は#2600、版Matrix/BOMは#3267、Trait実装義務はmain #3480/#3278、Scenario/独立Expectedは#1180/#2514。新しいResolver/Registry/Host/Scenario SSOT、HTTP常駐必須化、Legacy `application-runtime` CarrierのScope緩和は行わない。本節自体は計画の更新で、S2/S3の実試験成功・Owner Meaning受理を**主張しない**。

**2026-11-09 C-1/C-2・Hook・公開Tool契約の横断追補（設計引継ぎ）**：[Trait設計§20](design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)で、新しい[入口比較実測](evidence/CONNECT_POINT_C1_C2_2026-11-09/README.md)と[Service Carrier限定実行](evidence/SERVICE_CARRIER_CONNECT_REFERENCE_2026-11-09/README.md)を踏まえ、**C-1配備動作とC-2世代固定Service要求は必ずしも代替ではない**と精密化した。採否・C-4外部実行は引き続き未決。main #3579に入った機能定義書のHook位相/reads/uses/produces表示は**宣言の投影**であり、実Hook/業務Effect・Trait履行は未実証。main #3563/#3582のCLI/MCP失敗整合と未統合#3590のTool Discovery差分を踏まえ、広告／Required Closure／実Provider／結果を別々に観測する。TCR-01〜06は#3278/#3279→#2110/#1129/#3419/#3420/#1180/#2514へ委譲し、新Core Trait/Host/Runnerを追加しない。

**2026-11-09 原子的Traitの実行主体・BOM資格の境界（受入計画）**：[Trait設計§19](design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)で、Required Traitの実装義務と、製品内実行／外部Host実行の選択（[構成の繋ぎ目判断材料](workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)のC-1〜C-4）を独立させた。C-4「TEIでは実行に接続しない」も正当な候補であり、本計画がその採否を決めない。#3540は通常CLI/StudioのS2へ向けた**未統合Draft**、#3538/#3541のRQ-037 BOM要求・Matrixも**未統合候補**。実ProviderのTrait履行、外部Receipt、同一Attempt/TargetのQualified BOM、現在Grantをそれぞれ別に検証し、TEB-01〜06は#3278/#3279→#1251/#3267/#1180/#2514への未実行受入観点として引き渡す。新Host/Resolver/Runner/Hard Gateなし。製品S2と外部Hostのどちらかを他方の前提にしない。

**2026-11-08 実装の新しい基線（main #3381／Plugin計画 #3525）**：[PR #3381](https://github.com/hat47x/tei-platform/pull/3381)がmain（統合 `f8896886c4528920a0d0cb3a2f67181971402987`）へ入り、`internal/inspectoradapter.Selected`で`authoring-tool`のexact Selection/Epoch/Host guard/Host登録Go実関数を接合、CLI注入入口とStudio Host Serviceで、Data・Rule・Relation・Capability・Application・Patternの6 Subjectを**二つの局所Go call-siteから同一Bへ合計12呼出し**する試験コードを収録した。B選択→A登録、Scope/Contract/Artifact不一致、Host Grant撤回、Epoch失効、Secret型付きError等の拒否もテストに含む。これらはS0と**限定S1**の実装であり、通常CLI `main.go`からのB自動選択、実Wails `StudioBridge`配線、実Plugin Binary/Source/Current Grantの真正性、外部Tool配信、2×2 Verified BOMはなお**S2/S3未成立**。PR旧HEADのGo package試験は一部索引漏れだけで全Health FAILし、索引を補正した新HEADを統合したが、**新HEAD全体CIの最終結論はこの追記時点では未確認**。この関係は[現行実装状況](../docs/status.md)と[Plugin計画 §19](design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)へ接続した。

**次に着手する唯一の本番経路**：既存#951/#1016のResolver/ProfileをHostが受け、CLIの通常`inspect`とStudioの実Wails Hostで**別々に**同じ選択済みProvider Bを注入する。その際「選択」が実Artifact/Provider Entryの真正性を意味しないため、Selection→actual Binding→実Call/Source/Effect/受領のReceiptを分割したまま#1180/#2514の独立Expectedで評価する。Lifecycle Graphの型付きPatch/Hook（#3503/#3510）、B2C Headless（#3509）、AIチューニング（#3520）は、最初のS2を阻害しない後続Consumer/圧力試験とする。新たなResolver、共通常駐Host、Inspector専用の意味受理機構、第二Runnerは作らない。
| **S0 Typed Port**（#1158/#1824） | [CLI Draft #3381](https://github.com/hat47x/tei-platform/pull/3381)の`inspectWorkspaceJSONWithService`、[Studio Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)の`RootServices.Inspection`。指定Service欠落・出力契約不一致・未許可Root時はfail-closedを検査 | 両方**未main統合**。通常CLI `main.go`はBuiltin経路。型付きService注入が可能でも実Plugin選択をしたことにはならない。#3371現HEADの全Go Test/Workspace validateは通過するが、`gofmt`不一致でRepository Health FAIL（下記実測） |
| **S1 selected Provider（局所）**（#951/#1016/#1251） | CLI Draft #3381の`internal/inspectoradapter`が既存Registry/Selection/SelectedCoordinator/Host guardを使ってB選択→B実Go Call／B選択→A登録拒否／epoch/Grant失効を**局所in-process**で検査する候補 | 局所B実Call・B選択/A登録拒否を含む現HEADの全Go Test/Workspace validateは通過、ただし`gofmt`不一致でHealth FAIL。合成ID/digest・登録Functionだけでは配布Binaryの実Member/Entry/Conformanceを証明しない。Target実Entry・Authorityは別 |
| **S2 CLI/Studio実Consumer**（#1251/#1824） | HostだけがTrust Rootとexact Selectionを保持し、**通常起動のCLI dispatchと実Wails Studio Bridge**を同じ契約・各自のTargetへ接続する。[main統合 #3378](https://github.com/hat47x/tei-platform/pull/3378)（squash `0f3455e132d2146834d65089c68cae40c31f3055`）の従来Direct経路を6 Subject＋Not Foundで比較するGo Testは、exact-head Healthが成功した**移行前Baseline**としてのみ利用 | UIがCLI JSONを表示しただけ／factory nilでBuiltinへフォールバック／別revisionから得た同値JSONを二Consumer成功としない。B selected→A Callは同値ResultでもFAIL |
| **S3 Source・Artifact・Effect実証**（#3262/#2600/#3267） | [main #3482](https://github.com/hat47x/tei-platform/pull/3482)の限定Function Required（13実CLI Reply/42局所Controls）と[main #3477](https://github.com/hat47x/tei-platform/pull/3477)のSource-only失効を、実S2 Attemptで得たSource/Provider/Artifact/Scope/独立Expectedへ照合 | 13 Replyは原子的SnapshotでなくFunction実行資格もない。[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)の外部Provider raw/Manifestのみ改訂は**現HEAD Go Test通過／gofmt FAIL**、[Draft #3490](https://github.com/hat47x/tei-platform/pull/3490)の選択規則fingerprint変更は**現HEAD Repository Health成功**の限定Go Reference。ともに未main統合であり、Consumer World FPの一致だけでfresh／Qualified BOMにしない |

**2026-11-08 exact HEAD CI再検収（S0/S1/S3の統合条件を訂正）**。旧計画の「CI未確認」を放置せず、GitHub Actionsの各**PR HEADに対応する実Job Log**を確認した。

| PR / 対象HEAD | exact HEADで実際に確認したこと | 現在の阻害・次の作業 |
| --- | --- | --- |
| [Studio #3371](https://github.com/hat47x/tei-platform/pull/3371) `c7acfd6b` | [Health #42464668701](https://github.com/hat47x/tei-platform/actions/runs/42464668701)：`go test ./...`／Workspace validate PASS | Health FAILの唯一の記録原因はGo1.26.10での`internal/studiohost/readonly.go`、`readonly_inspection_service_test.go`の**gofmt drift**。**当該PRで整形→再実行**。既存Wails smoke/Studio PoC Aなど別Workflowはsuccessだが実Selected BのS2 Callを証明しない |
| [CLI #3381](https://github.com/hat47x/tei-platform/pull/3381) `5cdd3b9e` | [Health #42463722188](https://github.com/hat47x/tei-platform/actions/runs/42463722188)：`go test ./...`／Workspace validate PASS | Health FAILは`cmd/tei/cli_inspect_service_test.go`、`internal/inspectoradapter/selected_test.go`の**gofmt drift**。**当該PRで整形→再実行**。S1局所B実Go CallのGo PASSと通常CLI S2未接続を区別 |
| [External Import #3479](https://github.com/hat47x/tei-platform/pull/3479) `fff65f0a` | [Health #42464022323](https://github.com/hat47x/tei-platform/actions/runs/42464022323)：`go test ./...`／Workspace validate PASS。Source／Manifest各改訂のGo testがそのcheckoutで実行された範囲 | Health FAILは`internal/workspace/import_source_context_revision_test.go`の**gofmt drift**。**当該PRで整形→再実行**。Go PASSでも複数Loadの原子的Closure／Current Grantなし |
| [BOM政策参照 #3490](https://github.com/hat47x/tei-platform/pull/3490) `6b2e0b18` | [Health #42464016653](https://github.com/hat47x/tei-platform/actions/runs/42464016653) **completed/success** | Scope／Policy RuleFingerprintのReference限定PASS。選択Ruleと実Provider Source／実Artifact／実Invoke／BOM 2×2の同一Attemptは未認定 |

**CIの実務順**：新しい抽象設計やProvider追加の前に、上記3 Draftを**各Ownerの既存PR内でGo1.26のgofmtだけ修正**し、exact HEAD Healthと各専用Workflowを再検収する。Go TestがPASSしたことをCI全体PASSとは呼ばず、Health PASS後もS2通常CLI／Studio実B呼出し・S3同一Input Closureと現在権限は別の欠落辺として残す。CI修復のために第二Test Runner／新Plugin Runtimeを設けない。
**次の一件をS2の実装と検収に固定**：同じ公開`reference/workspace`・同じ`semantic-data:sales.order`・exact`authoring-tool` Contractを用い、(a)CLI通常dispatch、(b)Studio Hostの実Bridgeから、それぞれ実選択BのGo関数へ入ったことを**別のCall-site記録**で確認する。BをAへ取り違える／Serviceなし／Source ABA／Host Guard失効／部分応答の各負例では、旧Builtinへ自動fallbackせず診断を返す。公開されるCall証拠は受信先の権限と#2516のDisclosureに従い、内部Root/Secretを誤開示しない。実バイナリとWebViewの適格性はS3で別に測る。

**予定の帰属**：S0はStudio main #3371／CLI Draft #3381、S1はCLI Draft #3381の局所実Go試験、S2のProduct統合は#1251/#1824、S3のDesign Requiredは#3262、履歴Sourceと再利用判定は#2600、版Matrix/BOMは#3267、Trait実装義務はmain #3480/#3278、Scenario/独立Expectedは#1180/#2514。新しいResolver/Registry/Host/Scenario SSOT、HTTP常駐必須化、Legacy `application-runtime` CarrierのScope緩和は行わない。本節自体は計画の更新で、S2/S3の実試験成功・Owner Meaning受理を**主張しない**。


**2026-11-08 実装の新しい基線（main #3381／Plugin計画 #3525）**：[PR #3381](https://github.com/hat47x/tei-platform/pull/3381)がmain（統合 `f8896886c4528920a0d0cb3a2f67181971402987`）へ入り、`internal/inspectoradapter.Selected`で`authoring-tool`のexact Selection/Epoch/Host guard/Host登録Go実関数を接合、CLI注入入口とStudio Host Serviceで、Data・Rule・Relation・Capability・Application・Patternの6 Subjectを**二つの局所Go call-siteから同一Bへ合計12呼出し**する試験コードを収録した。B選択→A登録、Scope/Contract/Artifact不一致、Host Grant撤回、Epoch失効、Secret型付きError等の拒否もテストに含む。これらはS0と**限定S1**の実装であり、通常CLI `main.go`からのB自動選択、実Wails `StudioBridge`配線、実Plugin Binary/Source/Current Grantの真正性、外部Tool配信、2×2 Verified BOMはなお**S2/S3未成立**。PR旧HEADのGo package試験は一部索引漏れだけで全Health FAILし、索引を補正した新HEADを統合したが、**新HEAD全体CIの最終結論はこの追記時点では未確認**。この関係は[現行実装状況](../docs/status.md)と[Plugin計画 §19](design/PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)へ接続した。

**次に着手する唯一の本番経路**：既存#951/#1016のResolver/ProfileをHostが受け、CLIの通常`inspect`とStudioの実Wails Hostで**別々に**同じ選択済みProvider Bを注入する。その際「選択」が実Artifact/Provider Entryの真正性を意味しないため、Selection→actual Binding→実Call/Source/Effect/受領のReceiptを分割したまま#1180/#2514の独立Expectedで評価する。Lifecycle Graphの型付きPatch/Hook（#3503/#3510）、B2C Headless（#3509）、AIチューニング（#3520）は、最初のS2を阻害しない後続Consumer/圧力試験とする。新たなResolver、共通常駐Host、Inspector専用の意味受理機構、第二Runnerは作らない。

**Atomic TraitとRQ-036全体マップの次の受入（既存S0〜S3の上で検証）**：[Trait設計§18](design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)に、main #3381／#3531の**局所S1到達**と通常CLI／実Wailsの**製品S2未接続**を明記した。次のS2一件は既存`inspectoradapter.Selected`を再利用し、通常CLI／実Wailsそれぞれのactual B invokeを確認する。Traitの`manual-required`欠落診断は#3279の既存Contractに薄く接続する。RQ-036（main #3505）の全体ナビゲーションへTraitを表示する場合は、既存Source／Required Closure／Provider／Conformanceの**read-only参照**に限定し、Model Map Draft #3518のpartial観測を「全Trait実装済み」にしない。TMP-01〜07は既存#1180/#2514への**受入候補・未実行**であり、Source/実Provider/Build/Host Grant/意味受理の新資格を発行しない。

**2026-11-08 S2追補：D1a既存基線とTyped Error秘匿・Atomic Trait二Consumerの判定**（既存S0〜S3の変更なし）

- **D1a基線を実績として固定する**：[main統合 #3378](https://github.com/hat47x/tei-platform/pull/3378)では既存CLI／Studioの直接Operation経路でData/Rule/Relation/Capability/Application/Patternの6 Subjectを読み取り、Canonical・Source位置・World/Source Index FP・EffectとNotFoundの同値を検証。exact HEAD [Health #42463508044](https://github.com/hat47x/tei-platform/actions/runs/42463508044)はcompleted/success。**D1a direct同値PASS**であって、D1b Provider Bの実Selected／actual Invokeではない。D1aを作り直さず、S2では同じSubject/Expectedで呼出ProviderのIDと実Call-siteを追加観測する。
- **Host-selected Inspectorの失敗時開示を独立P0にする**：[Studio main #3371](https://github.com/hat47x/tei-platform/pull/3371)のGo Sourceに、Providerが`operations.InspectionError`のCodeとMessageへ絶対Path/Token/内部Causeを埋め込める反例が追加された。非nil Factoryの選択Serviceだけに適用する`selectedInspectionFailure`は4つの許容Codeを固定Messageに変換し、それ以外を`OPERATION_FAILED`とする。通常CLIも別ConsumerとしてPublic Code＋秘密Message／Private Code＋秘密Message／Generic Errorの一因子負例でClient実受領を確認し、**Providerからの型付きErrorを既に安全な公開診断だと扱わない**。Source差分のexact-head CIはsuccessでmain統合済み。ただし実Wails BridgeでのProvider B実呼出しと公開結果は独立検証を要求する。
- **Traitは二種類の実Consumerで検収する**：[main統合 #3502](https://github.com/hat47x/tei-platform/pull/3502)により、P0-T=CLI/Studio read-only Inspector、P0-R=`pattern order_entry.recent_orders browse`→`query order_entry.recent_orders`→`recent_orders.sql (:from)`→`datasource sales`→`application order_entry.sales_core` Alias→実Provider Rowset→Function Contextを**別のRequired/Expected/Attempt**に固定。共通化するのはTraitのDefinition／Binding／Implementation／Invocation／Conformanceの証拠形だけ。Provider Instance/Source/Target/Current Grantは共通正本化しない。`sales.order.submit`のMutation/Transaction TraitをBrowseに暗黙継承させず、Manual Required不足とProvider未選択と実Readerの0件／partialを別に拒否する。P0-Rは#2586/#3270、Trait履行の正本は#3278/#3279。

**受入の次の最小追加条件（独立Expected #2514へ委譲）**：D1B-ERR=Typed Provider ErrorのCode/Message両方を非信頼として匿名化、D1B-TRF=TとRのTrait Required集合を別検査、D1B-CALL=同じ6 SubjectでB選択→B実Invokeを各通常CLI/Studio Call-siteで証明。**局所S1だけの成功、暫定Builtin Studio Host、Direct D1a同値、二Consumer間のTrait証拠形式同値をD1bやP0-R実Provider Readの合格へ昇格させない**。新Resolver/Runner/Host/二重Scenarioを追加せず、旧Carrierの`application-runtime`限定も緩めない。本追補は計画文書のみ、追加Go/実DB/実Wails/実Provider/Owner意味受理はNOT RUN。
**実装候補の進捗（2026-11-08）**：[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)では既存`studiohost.RootServices/ServiceFactory`へ型付き`InspectionService`を追加し、Factoryを設定した場合の旧`operations.InspectSubject`への無音fallback拒否、Subject/Effect/Schema/Fingerprint等の不一致拒否をGoコードと試験へ具体化している。ただし**未統合Draft／作成時点Go CI未検収**であり、実Compositionで選択したProviderのHost側供給、CLIの同じProviderへの接続、WebView/Build/actual Invocationの証拠は別。P0-b/cの候補として既存#1251/#1824が検証し、Studio seamの追加だけでD1bを完了扱いしない。

**今回の上限**：main統合済み参照結果の引用と、実Consumerに残る不足の計画反映。PGD-01〜06を同一HEAD・同一attemptで実CLI/Studio/Go/Frontend/Host検証したわけではない。新Contract Schema/ABI、実B呼出、G4 L3、BOM組合せの本番資格、Owner業務受理は **NOT RUN／NOT PROVEN**。
- **Host-selected Inspectorの失敗時開示を独立P0にする**：[Studio Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)のGo Sourceに、Providerが`operations.InspectionError`のCodeとMessageへ絶対Path/Token/内部Causeを埋め込める反例が追加された。非nil Factoryの選択Serviceだけに適用する`selectedInspectionFailure`は4つの許容Codeを固定Messageに変換し、それ以外を`OPERATION_FAILED`とする。通常CLIも別ConsumerとしてPublic Code＋秘密Message／Private Code＋秘密Message／Generic Errorの一因子負例でClient実受領を確認し、**Providerからの型付きErrorを既に安全な公開診断だと扱わない**。Draftの新HEADのCIとWailsの実公開結果は独立検証を要求する。
- **Traitは二種類の実Consumerで検収する**：[main統合 #3502](https://github.com/hat47x/tei-platform/pull/3502)により、P0-T=CLI/Studio read-only Inspector、P0-R=`pattern order_entry.recent_orders browse`→`query order_entry.recent_orders`→`recent_orders.sql (:from)`→`datasource sales`→`application order_entry.sales_core` Alias→実Provider Rowset→Function Contextを**別のRequired/Expected/Attempt**に固定。共通化するのはTraitのDefinition／Binding／Implementation／Invocation／Conformanceの証拠形だけ。Provider Instance/Source/Target/Current Grantは共通正本化しない。`sales.order.submit`のMutation/Transaction TraitをBrowseに暗黙継承させず、Manual Required不足とProvider未選択と実Readerの0件／partialを別に拒否する。P0-Rは#2586/#3270、Trait履行の正本は#3278/#3279。

**受入の次の最小追加条件（独立Expected #2514へ委譲）**：D1B-ERR=Typed Provider ErrorのCode/Message両方を非信頼として匿名化、D1B-TRF=TとRのTrait Required集合を別検査、D1B-CALL=同じ6 SubjectでB選択→B実Invokeを各通常CLI/Studio Call-siteで証明。**局所S1だけの成功、暫定Builtin Studio Host、Direct D1a同値、二Consumer間のTrait証拠形式同値をD1bやP0-R実Provider Readの合格へ昇格させない**。新Resolver/Runner/Host/二重Scenarioを追加せず、旧Carrierの`application-runtime`限定も緩めない。本追補は計画文書のみ、追加Go/実DB/実Wails/実Provider/Owner意味受理はNOT RUN。


### トポロジーと複合図表の評価順（未採用の追加検証候補）

[トポロジー・複合図表・開発パイプラインの共通表現](design/TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)を、既存契約を重複実装しないための横断的な設計・検証候補として参照します。現行のIssue優先順位、製品受入条件、実装済み機能の説明は変更しません。

最初は既存の型付き参照・Graph・Projection・Build/Compositionを棚卸しし、同じ構造からGraph・表・関係表を読み取り専用で投影します。次に、対象版とscopeを固定した実効パイプラインの解決完全性を評価し、完全性が証明できない場合に図表上の欠落を「不存在」と誤認しないことを検証します。編集・受理、認知負荷の比較、メタ生成の導入は、その後に具体的な不足と効果が観測された場合だけ進めます。

図表間リンクは、同一qualified要素の複数表示、別要素間の意味的な関係、表示用のナビゲーションを区別します。複合図表の関連グラフ・対応表も第二の意味正本としません。これらは採用済みの新Core SchemaやStudio編集APIを要求するものではありません。

### 三要素MethodのScript-first圧力試験（2026-11-06、評価待ち）

[三要素MethodのTEA-V0 / Self-build Fit/Gap](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)を、作業領域Gの**最小縦断実証の追加候補**として位置づける。Data/Function/Businessは同じauthoritative Script/標準データのRole別Projectionであり、専用図形を第二正本としない。既存#1124 M0〜M4と#1125/#1126/#951/#267がOwnerであり、新しい横断Gate・Resolver・Acceptanceは作らない。

最優先の不足は、①Source byteと意味が同一でもSource移動により旧定義ジャンプが失効すること、②可逆でないER/DFD線編集をSourceへ誤適用しないことを、**同一の実Consumerにおける正負の試験**で証明する点にある。並行するread-only評価では、**T06（三視点二世界の開示非干渉）・T07（同一Sourceへの複数Roleでのunknown保持）**を先に反証し、Policy/Role ResolverをTEA独自に実装しない。その後にバッチ/ファイル連携と業務図の役割区別、Method Pluginの実Build/Loadを扱う。Plugin self-buildのSB-1/2/3は既存Ownerの独立資格であり、TEAの初期read-only利用のブロッカーにはしない。 [T06/T07の限定Reference（Draft #2972）](https://github.com/hat47x/tei-platform/pull/2972)では、事前公開資格済みPublic Sliceの投影・unknown Roleの局所性に加え、Private metadata混入拒否をGitHub保存ファイルと同一BlobのNode 9件で確認。ただし実Source→公開SliceのPolicy非干渉／実Workbenchは未実証。#1124の実CLI→同一Consumer接合を直接優先とし、この安全性の二世界比較を#2516/#2620と並行Pressureに置く。現行の全体優先順・Gate B/C判定は変更せず、具体的な着手・採否は担当Issueの判断へ返す。

**Trait Intentと実装の齟齬の三視点照合（評価待ち、2026-11-06）**：横断Trait設計の[main統合済みPR #3027](https://github.com/hat47x/tei-platform/pull/3027)を唯一のTrait Definition/Intent/Observation/Assessment Ownerとし、三要素Methodは[設計§18〜19](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)のT10〜T13を**読み取り専用の横断受入候補**として参照する。最小の`sales.order.status`では、Businessの閉値域Intent、DataのJava EnumとDB制約、Functionの書込みValidator/未確認Writerを**同じ意味対象・Scope・版・公開資格**へ接続し、制約保証の欠落を実データ違反と誤同一視しない。MECEの母集団/排他/網羅も別Claims。原典未確定の「強属性」を標準Facetとして採用せず、業務Identity・CandidateKey・不変性は独立して審査する。既存の#1124・#1126・#2516・#2620の受入を上書きせず、Studio M0→M1の直接P0や既存Gate B/C優先順を変更しない。T10〜T13の[合成read-only Trait結合Reference（Draft #2972）](https://github.com/hat47x/tei-platform/pull/2972)では、既存の公開Slice投影から三視点Occurrenceを得て、外部資格済みIntent/Observation/Assessment参照の対応をNode 10件で局所検査済み。別Scope・旧Snapshot・非公開Claim混入の拒否を含むが、実Source Reader／Java・DB観測／Policyが正しい公開Sliceを生成すること／実Workbenchへの受領は**NOT RUN**。方法論の受入、セルフビルドや既存OwnerのGate合格へは昇格しない。

**三視点の世代結合とサブシステム分割候補（2026-11-08、並行評価）**：[三要素Method設計§19.4・§20](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)と[Reference Draft #2972](https://github.com/hat47x/tei-platform/pull/2972)を接続。Traitを貼る時は`Projection/Envelope/Claim`の呼出元Snapshot三者一致を要求し、同じView IDでも旧図への新Trait貼付を拒否する。GitHub exact sourceを用いたV8テスト互換ハーネス**22/22件PASS**（[exact HEADのNode 22専用CIはcompleted/success](https://github.com/hat47x/tei-platform/actions/runs/42385331823/job/128441389821)、Repository Health・Product E2Eは別資格）。このStampは実Source/現在Grantを証明しないため、実CLI→同一Consumer/Policy/旧DOM失効は#1124の直接P0に残す。並行する[#3250](https://github.com/hat47x/tei-platform/issues/3250)/[Draft #3255](https://github.com/hat47x/tei-platform/pull/3255)の論理サブシステム分割は、TEA側でDataの所有候補・Functionの**direct CRUDと委譲/Effective Effect**・Businessの手順/Actorを別Evidenceとして投影するT14〜T16（NOT RUN）だけを追加。最適化・Owner判断・実Providerを再実装せず、既存優先順位/Stage B・Cは変更しない。

**Workspaceの読取文脈と三視点資格の分離（2026-11-09、評価待ち）**：[設計§21](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)へ、[実CLIの異Workspace同Snapshot負例 #3321](https://github.com/hat47x/tei-platform/pull/3321)を受けて、`Workspace観測`と`Snapshot`の別照合を追加。合成TEA Reference [Draft #2972](https://github.com/hat47x/tei-platform/pull/2972)は公開用`workspaceObservation`をPublic Slice・Projection・Trait Envelope/Claimで照合し、同じSnapshot/表示IDでも異Workspaceなら拒否。GitHub実ソースのV8互換検査**27/27 PASS**に加え、[PR #2972のexact-head専用Node CI](https://github.com/hat47x/tei-platform/actions/runs/42591779028)も**completed / success（公開Slice12件＋Trait Bind15件）**。ただしRepository Health・実Policy・実Studioの製品受入は別資格。これは同じ物理Root・Read Attemptの証明ではない。[#3543](https://github.com/hat47x/tei-platform/pull/3543)の実CLI A/J→同一Workbench **Initial/List 0/0/1件**という限定証拠をM0→M1の既存進展として再利用し、残るGraph/Command・旧DOM・Source/Policy現在性・独立Readの実接合へ直接P0を絞る。[#3728](https://github.com/hat47x/tei-platform/pull/3728)の`Catalog read成功`と`Validation NG`、[#3518](https://github.com/hat47x/tei-platform/pull/3518)の部分Model Mapを別Claimとして示すT18〜T20は未実行。Studioに別Reader/Policy/Validator/新Ownerを作らず、既存Stage B/CとSelf-build/ETLの独立Gateは変更しない。

**Read・Validation・Source結合・開示の独立資格（2026-11-09夜）**：[merged #3728](https://github.com/hat47x/tei-platform/pull/3728)の実Catalog read成功＋`validation.ok=false`と、[merged #3492](https://github.com/hat47x/tei-platform/pull/3492)のSource Index構造PASSでも偽装Subjectへの実Inspectが拒否される反例を[TEA設計§22](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)へ接続。T21 ReadとValidateの別表示、T22 Locator Byte妥当性とSemantic Subject実在性の別検収、T23非公開診断件数の二世界非干渉を**新規Consumer受入候補（NOT RUN）**とした。既存[Reference Draft #2972](https://github.com/hat47x/tei-platform/pull/2972)にはSubject/Relation/Claim/ViewのArray accessor/iterator/holesを入力資格外として拒否する4負例を追加し、GitHub byteを用いたV8 shim **31/31 PASS**。さらに[合成T18 Catalog/Validation表示Reference（Draft #2972）](https://github.com/hat47x/tei-platform/pull/2972)へ6件を追加し、GitHub実ソースのV8互換試験**37/37件PASS**（Public14 / Trait17 / Catalog6）。Read成功とValidation NGを別表示し、未観測ReadからValidationを捏造しない。**新head Native Node CI、実Catalog JSON受領、現在Policy/GUIは別資格**。#1124のGraph/Command/旧DOM・現在Source/Policyの直接P0を最優先に維持し、ValidatorやSource Registry、第二PolicyをTEAへ複製しない。

**三要素Method T18：実CLI→独立Node Read/Validation受領（2026-11-11、限定実験）**：[設計§23](design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)に、main統合済み[三視点合成Reference #2972](https://github.com/hat47x/tei-platform/pull/2972)・[Method設計 #3257](https://github.com/hat47x/tei-platform/pull/3257)と実Catalog Validation [#3728](https://github.com/hat47x/tei-platform/pull/3728)を使う小さなProducer→Consumer試験を定義。Git HEADを固定して実Go CLIの正常/検証NG一時Workspaceを読み、別言語Nodeが`model catalog`と`validate -json`の資格を別々に観測したうえで、既存Read-only Binderへ**実Catalog Summary**を渡す。Role・Disclosure・Workspace witnessは試験用であり、Sourceの独立Read/実Studio Policy/Owner受理は未実証。既存[実CLI→同一Workbench #3543](https://github.com/hat47x/tei-platform/pull/3543)を重複実装しない。先行する#1124 Graph/Command・旧DOM・Current Grant/Source Closureと既存Gate/セルフビルド優先を変更せず、[実Go+Node exact-head専用CI #42999527960](https://github.com/hat47x/tei-platform/actions/runs/42999527960)は`af1f7bb5a60a965a8ea0fe0dc71a95aa9b6a6949`に対し**completed/success**。実CLIの正常／Validation NGの2公開Workspaceを独立Nodeが受領し、既存T18 Binderは実Catalog Summaryを保持できた。**実Role（Function/Business）・実Disclosure／Workspace Root/Atomic Attempt／Workbench GUI／Source編集・Owner受理は未実証**。Docs更新後の新HEAD全体Healthは別資格、Draftの受入判断は既存Ownerに返す。

## 作業領域H：AIによる開発支援

### 目的

AIを大量の成果物を生成するだけの仕組みにはしません。既存知識の探索・変更案の作成・検査・意思決定の支援に利用します。

### 二段階

1. `ISSUE-000047`：人とAIが共通に利用できる、既存知識を優先した意味情報の探索（PAS-06）
2. `ISSUE-000027`：AIによる変更案と共通の受理の境界（PAS-01・PAS-06）

AIによる提案には、意味の正本を変更する権限はありません。

検索順位・類似度・モデルの確信度も、意味の正しさを示す検証根拠ではありません。

AIの自律的な作業と実行権限を区別します。AIがツールを直接呼び出せることを、能力や利用許可の仕組みの代わりにはしません。

### AI-firstの現行受入：TT-56実CLI閉包→盲検T1-Bと実Providerを分ける（2026-11-08）

[AI-first資格・実測対応設計§10](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)を、作業領域Hの**受入順と担当への引渡し**として参照する。旧[PR #3241](https://github.com/hat47x/tei-platform/pull/3241)の複数段追記は主張の歴史として設計本文へ保持し、本計画へは重複掲載しない。現行mainのIssue/ADR・TAI Runbookが受入・実験の正本である。

**確認済みの最小成果**：[main統合 #3354](https://github.com/hat47x/tei-platform/pull/3354)は旧#3324/#3326を一つのContext Gateへ統合し、固定公開Sourceについて**11実CLI Reply＋32局所Consumer Controls**（既存16・Required三Rule 8・Rule/Relation/Source詳細8）を[専用exact-head CI #42391707962](https://github.com/hat47x/tei-platform/actions/runs/42391707962)で成功、Healthも別Runで成功。旧二PRはclosed/unmerged。これは一時コピーの欠落/改変を検出した実績であって、実Harnessの全Byte配送・全Domain/Function/BusinessのRequired Closure、AIのT1-B新規発見は**未証明**。`boundedReference=partial`／`qualifiedTestGeneration=false`を維持。

**P0-A/P0-B/P0-Cを分離**：(A) #3262の実Loader/Rule/Relation/Source/Index/Canonical取得と独立Requiredを一Subjectの版に束縛し、不足理由を返す。(B) TAI #168・TEI #2509/#2515で独立Expected/未公開Gap/旧回答への**実OS拒否**、許可`Command+Flag`/argv・Tool Reply全Byte/終端・Session Log/Egress Sink/Consumer受領を実測して初めて比較/盲検資格を与える。(C) #3273/#1180/#2514で**未知GapのT1-B Candidate一件**を独立Expected/Mutantで採点。既知行番号0のT1-A成功は未知Gapの発見でない。[Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)の部分Scenario目録TT-105/106はfixed-head Dogfood/Health/依存CIがいずれも成功したが、**部分受領＝notObservedであってuntestedConfirmedではない**、全Coverageはunknown、Draft未統合。

**別レーンの実資格**：[main統合 #3353](https://github.com/hat47x/tei-platform/pull/3353)は空Targetの`EmptyPolicyHandoff`に`HandoffConfirmed=false`なら`unverified`を返す局所修正を加えたが、実相手方Receipt・Target Set全Readではない。[main #3350/#3363](https://github.com/hat47x/tei-platform/pull/3363)の参照Capability/Bindingは12宣言・未選択4、24 Bindingの選択22/欠落2（`editor.database_workbench`）、ただしReferenceの`missing`≠現Validatorの違反≠実Plugin Loader。[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)はStudio InspectorをHostの選択InspectionServiceへ接続する候補だが、CLIは依然直接経路であり、**一Pluginのselected→actual loaded/invoked→CLI/Studio二Consumer**を実証したものではない。#1251/#951/#3292へ実Invoke試験を引き渡す。

**Inspector D1の実接続（2026-11-08追補）**：[AI-first資格・実測対応設計§11](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)へ、CLI/Studio双方の**既存直結Inspectorの応答基準**[Draft #3378](https://github.com/hat47x/tei-platform/pull/3378)、**CLIの型付き注入入口**[Draft #3381](https://github.com/hat47x/tei-platform/pull/3381)、**StudioHostのHost選択Service入口**[Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)を別資格で接続。両Surfaceの応答一致・Host Factoryからの委譲は、選択済みPlugin/Providerの実Build/Load/Invokeを証明しない。次は#951/#1251/#3292が所有する**一read-only Inspectorの`Provider B selected→B actually invoked`**をCLI/Studio二Consumerで実測し、B選択→旧Builtin A呼出し／明示Service未提供→暗黙Fallbackの負例を拒否する。既存`inspect_projection`の共有Result検証へ寄せ、第二Registry/Hostを作らない。#3378/#3381/#3371の旧exact-head Repository Healthは**それぞれ失敗**（前二者にはGo整形問題、Studioには既存Service注入契約の失敗もある）。他Workflow PASSと合算しない。[Draft #3370](https://github.com/hat47x/tei-platform/pull/3370)の`requirements.resultShape=array`は実Reply形状の改善候補で、外側の成功Envelope・Source完全性・権限とは独立。**P0のT1-B盲検/H2隔離と並列**とし、D1の完成を前提にしない。

**Tool公開とReaderの停止線**：[main #3333/#3340](https://github.com/hat47x/tei-platform/pull/3340)の`writeFlags`・Help/Discovery整合は安全な実argv/OS非書込みの証明ではない。[第2回CLIドッグフード #3356](https://github.com/hat47x/tei-platform/pull/3356)は一部改善とD-3/9/10残存を記録し、**現在の実入口はCLI、MCPはNOT RUN**と分類。Managed Markdown、ReaderでParsedされたSource、実Consumed Meaningを混同しない。旧`*.tei`の一Function→実Provider read-only（#2586）と新`*.tei.md` Reader移行（#3030/#3031）は並行し、T1-BのHard dependencyにしない。

**PoCのA3自律作業**は採用ADR-0035に従い可能だが、製品のHuman/Owner Shared Acceptance（#267）・実Host current Grant/Effect（#1174/#2598）・外部不可逆効果の人的確認は別。**今回の追加はDocs/Planのみ**。新H2/H3四Session・実T1-B・実DB/Plugin/Host・正式Meaning Acceptance・C1はNOT RUN。Scenario/Expected/Registry/Host/Runbook/Readerの第二正本は設けない。
### AI Proposalの入力・検証・受理・Host境界（2026-11-06横断追補）

[ADR-0022](adr/0022-ai-proposal-canonical-acceptance-boundary.md)と[#267](https://github.com/hat47x/tei-platform/issues/267)の**Shared Acceptance**を唯一の正式なSource変更経路として維持し、AI専用の書込み・受理・Provider直結経路を追加しません。[ADR-0030](adr/0030-context-scoped-epistemic-knowledge.md)／#958の由来・確認履歴・用途相対資格を、生成AI向け入力にも適用します。具体的な最小横断仕様・UI状態・AIB-01〜12の負例は[AI-first設計§5.4](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)へ一本化します。

- **Inputの現在性と外部送信**：Source ID/実消費byte/Revision/fingerprint/span、Semantic/Operator Lock、依存閉包、purpose/Principal、Allowed *proposal/read* capability、開示範囲を別々に固定します。宣言Input＝実Load/Parse済みではなく、内部read-only＝外部LLMへの送出許可でもありません。Source本文・Tool結果に埋め込まれた命令はAuthorityではありません。
- **Candidateと検証**：Candidate/Attempt/base revision、Typed Diff、根拠Source span/意味対象、前提/未解決を保持。構文/型/参照と、引用の意味妥当性、Source鮮度/完全性、Rule衝突を別判定します。欠落・部分取得・曖昧・矛盾は保留し、再生成・A→B→Aでも旧Reviewを自動継承しません。
- **受理とEffect**：人によるReview/Owner意味受理→Source反映→reload/Conformance→型付きAction Intent候補→外部Hostの現在Grant/Target確認→実Provider Effect/Receipt/独立readを段階別に検収します。受理済み意味は現在の実行許可を保証せず、Commit応答不明では無確認再試行を行いません。
- **Owner分担・優先順**：#988＝構文/Operator、#2619＝意味Role、#2622＝Data Operations、#267＝Shared Acceptance、#958＝認識状態、#2515/#2516＝Disclosure、#1180/#2514＝Scenario/独立検証基準、#2598/#1174＝実Target/Effect。P0固定CandidateのHeadless負例→P1独立検証基準/既知Mutant→P2実Consumer/外部送信・二世界非干渉→P3実Host/Receipt→P4実AI生成品質の順とし、低レベル試験のPASSを製品統合へ合算しません。

**状態**：設計と計画の接続のみ。AIB-01〜12、実AI入力/生成精度、OS・外部LLM送信、実Provider/Host、Studio/SHO、Owner正式受理の新しい横断E2Eは**NOT RUN**。新Schema・Core enum・Acceptance Engine・Policy Engine・TEI独自Hostの採用決定ではありません。

### `*.tei.md`統一後のAI-first Authoringと受理資格（2026-11-06追加決定）

**Ownerの現行方針**：TEI向けAuthoringを`*.tei.md`へ統一し、`*.tei`の新規使用を原則停止、既存資産は段階移行する。従来の「Markdownで検討し、別`.tei`へ切出し」「`.tei.md`は標準にしない」という#2621の旧比較案は現行方針ではない。ただし**拡張子の決定とParser/Build/Studio/SHOの実対応は別資格**であり、`*.tei.md`という名称、OKFタグ、AI候補をOwner Meaning/実Runtime Grantへ昇格しない。

**最小Authoring Slice**は、`*.tei.md`のSource発見→raw byte・論理Source ID・Revision固定→Seed/Manifest/Language Plugin実Selection/Pin→Section/Row/Source spanごとのRoleとRead-only解釈→根拠付きAI Candidate/独立Expected→#267 Shared Acceptance→Source保存と再Load→実Build/Consumer読取の照合とする。通常の`*.md`は説明/SHO/OKF用途として保持し、旧`*.tei`の移行は原内容の構文差・引用/Link再対応・Source SSOTを照合してから行う。単純renameや同ファイル内の一部受理をファイル全体の意味受理としない。Markdown中の指示文/コードフェンスや外部OKF metadataはAIへの命令・権限として扱わない。

**既存Owner/受入**：#2621はAuthoringと配置Lint、#988/#3028はLanguage/Bootstrap、#2620はRole、#267はShared Acceptance、#2514は独立Expected、#2515/#2516は用途別外部送出、#2386と#1174はTenant/Host Grant、#2598は実Request対象。新規差分は[AI-first設計§5.5のAMF-01〜07](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)に限定する。AIB-01〜12・#2621既設FM-01〜08・#988 BS-01〜15とCaseを二重実装しない。

**状態と停止条件**：命名統一はOwnerの決定。既存`*.tei`の一括変換・Parser実対応、同一Source/Revisionでの実CLI/Writer/Consumer、OS/LLM送出、Tenant別実Provider、Host Grant/Receipt、Owner意味受理、AMF-01〜07は**NOT RUN／NOT ACCEPTED**。対応不能は明示して旧Sourceの正本性を保持し、新しい独自Host/Resolver/Registry/Parser/受理Engineを前提としない。

### AI-first初回縦断の最小Done：Source実消費→候補→受理（2026-11-07）

過去のAIB-01〜12／AMF-01〜07を増殖させるより、[AI-first設計§5.6](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)を**最初に実行する一つの検収手順**として優先します。旧`*.tei`から新`*.tei.md`への移行は[#3030](https://github.com/hat47x/tei-platform/pull/3030)（docs-only Draft）のM0〜M6がOwner。現mainの`*.tei`固定Loader・`filepath.Ext`・Markdown非対応のTEI Parserという実際の障害を解消するまでは、新Source形式の実運用やAI定式化E2Eを合格としません。

**第一Slice**：注文一件の`*.tei.md` raw Sourceを固定し、実Manifest/Language Plugin/Source Indexが選択した断片の`actual bytes・Revision・span・Required Input`を観測。次にLLM非依存の固定Typed Candidate・誤引用・部分/未知の独立Expectedを#1180/#2514で検査し、#267のHuman/Owner Shared Acceptanceを通った場合に限り実Source更新→reload/Conformanceを検証。AI生成品質や外部LLM送信はこの初回Doneの前提にしない。Owner意味受理が未取得なら機械的な解析/検証合格までで停止する。既存TM-02/05・AMF-03/04・AIB-01/03/10を**一Fixtureへ再利用**し、新Parser/Runner/第二意味正本を設けない。

**Done以降を独立に判定**：#2515/#2516で外部AIへ実送出した内容とOS/Tool/CIの各Sink、#2598/#1174で現Principal/Target/Required-Supplied-Evaluated Guards、実Provider Effect/Receipt/独立Read、#1824で後着Resultの現Selection Epochを評価する。業務Meaning受理とHost current Grantは独立した二軸であり、その両方が成立してもCommit/業務完了の証拠にはならない。既存#2901 Green／#3017 Red（AI追加Scenarioの**一件の実Product Mutant kill**）、#2853の限定PostgreSQL unknown伝播は、いずれも当該対象・HEADのEvidenceであって本Sliceの実行PASSには数えない。

**現状**：本追補は設計・基本計画の更新のみ。新`*.tei.md` Loader/Parser、固定Candidateの実Shared Acceptance、AI/OS送信、実Host・Providerとの同一Attempt、Owner正式受理は**NOT RUN／NOT ACCEPTED**。P0の実行後に初めて対象Revision・Consumer・不足Guardと証拠を明記して次の作業へ進む。

### P0二レーン実証とCompletionの証拠資格（2026-11-07修正）

前節の「初回P0は`*.tei.md`実Reader/Parserが先決」という停止条件は、**新Authoring形式からのShared Acceptanceの合格条件**へ限定する。旧`*.tei`の実Provider read-only／独立Expected検証は、[Script起動 Draft #2884 §10](https://github.com/hat47x/tei-platform/pull/2884)に合わせた**P0-L（Legacy）**として並行着手できる。**P0-M（Migration）**は`*.tei.md`の正しいSource発見・Markdown解釈・実Parser/Build入力まで別試験とし、最後に**P0-J**で旧新それぞれのSource byte/Revision/実消費範囲と同じ業務Requirement/独立Expectedを照合する。LのPASSをMへ継承せず、MのParse PASSを実Provider成立へ拡大しない。#2667が移行横断TMDと合流・Legacy退役を管理し、#3030がAuthoring固有のTMと表層、#3031が未対応Sourceの暫定拒否を担当する。第三のMigration Case SSOTは設けない。

**必要対象の完全性を独立に検査**する。[main統合 #2926](https://github.com/hat47x/tei-platform/pull/2926)の`EvaluateCompletion`は、渡されたTargetSet/Coverage/EmptyPolicyの評価器であり、Callerが宣言した`CoverageComplete=true`・行数0・表面上の`holds`からProviderがOwner対象全件を読んだとは推定できない。`required→discovered→loaded→parsed→actual consumed`、Required/Observed TargetSet/Row/Field、必要Guard/実評価Guard、空集合の独立証拠、現Principal/Tenant/Scopeを同じClaimに照合する。未観測なら`unverified`、矛盾なら別扱い。#1180はRequired Obligation、#2514は独立Expected/判別力、#2586/#2628は実Provider Read、#1163は同一Attempt、#1174は実Effectを担当。新しいCompletion Engine/Proof Registryを作らない。

[AI-first設計§5.7](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)へ実証順・既存Fixture再利用・Evidence上限を集約。**限定既存成果**：#3031の固定HEAD `63953ccb`の未対応`*.tei.md` Guard Go/CI success（Draft/未merge）、#2926のCompletion evaluator main統合と局所Go PASS、#3028のBootstrap正負対照Draft。これらから新Reader/実TargetSet/Owner meaning・実Host合流は導かない。P0-L実Provider、P0-M実Reader、P0-J旧新Meaning比較、同一Source改訂のCandidate→Shared Acceptance／Consumer、実OS/AI送出・Host/Provider Effectは**NOT RUN**。Go単一バイナリと既存外部Host権限境界を維持する。

### 既存Proposal・Command境界を使った初回検収の修正（2026-11-07）

[AI-first設計§5.8](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)へ、mainに統合された**実APIと実証の上限**を接続した。新たなAI修正案APIを作る前に既存`validation.Propose(diagnostics, worldFingerprint)`・`validation.Accept(&proposal, acceptedRevision)`（[#3041](https://github.com/hat47x/tei-platform/pull/3041)）を限定利用する。前者は診断に版を付け、後者は**局所`Accepted`フラグを立てるに過ぎず、Human/Owner authority・正式Source更新・Shared Acceptance／Reloadとは別**。修正PatchやSource引用span/入力閉包/Allowed AI capabilityもこの実APIには無い。真正なDecisionは引き続き#267、Source/引用の現在資格は#958/#988がOwner。

[Commandの修正 #3042](https://github.com/hat47x/tei-platform/pull/3042)により、Authorizer未設定は`COMMAND_AUTHORIZER_UNAVAILABLE`、宣言PeriodのObserver未設定は`COMMAND_PERIOD_STATE_UNAVAILABLE`として拒否され、実Executor後の宣言外Effectは`EffectUnknown`で保全される。ただし参照用`ReferenceAuthorizer{}`は現在の実Host発行者・Policy・実権限の証明ではない。「拒否・unknownだからno Effect」は誤り。差の保持について[AC-8実証 #3060](https://github.com/hat47x/tei-platform/pull/3060)は参照Storeの`OperationRecord.Differences`／CLI/Studio投影まで。実SQL/DB Commit/Provider/Owner業務完了ではない。既存#2598/#1174のTarget/Grant/Receipt/独立Readで後段を別検証する。

**初回Doneの更新**：P0-L旧`*.tei`の実Provider read-onlyとP0-M新`*.tei.md` Readerを並行維持。新形式の移行・TM-01〜14／M0〜M6の**主担当は#3030**（#2667 TMDは横断追跡対応表）。#3031は未対応新形式の暫定Guardである。既存AIB-01/02/03/07/08/11/12とTM/AMFを同一Fixture・独立期待値で照合し、`diagnostic proposal created → local flag accepted → owner source accepted → host request admitted → provider effect observed`を**五つの別資格**として報告する。新AI API／第二Acceptance/Host/Registry/Runnerは設けない。

**用語**：[#3039](https://github.com/hat47x/tei-platform/issues/3039)と[docs/naming.md](../docs/naming.md)に従いテストの「Oracle」を「独立期待値」「独立検証基準」に置換。**Oracle Database**はDB製品名として区別し、歴史的コード名・API・URLは改名しない。設計/実証と命名更新を混同しない。#3041/#3042/#3060の既存限定Go・Reference証拠を、`*.tei.md`新Reader／人の正式受理／実外部Host Providerの同一Attempt成功へ自動継承しない。**当該統合はNOT RUN／NOT ACCEPTED**。

### AI Proposal・Product Skill・Attested Computationを三経路として接続する（2026-11-07）

[AI-first設計§5.9](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)に、外部Harness実測と[OKF v0.2 Attested Computation](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/main/SPEC.md)の利用契約を接続。**AIの自由な意味Proposal**（#267のHuman/Owner Shared Acceptance）と、**受理・Pin済みComputationへの型付きParameter Bind**（Executor実Receipt＋決定的Attester）と、**Provider/外部Actionへの型付きIntent候補**（#2598/#1174の現在Host Grant/Effect）を明確に分離する。計算をAttestできたことから業務Meaning正当性・Provider対象集合全件・現在の実行資格を導かず、`validation.Accept`の局所`Accepted=true`からComputation承認や正式Source受理を導かない。

**実績更新**：TEI #2605／[TAI実S1 CI #42101157330](https://github.com/hat47x/tai-pleroma/actions/runs/42101157330)では、TEI固定HEAD `50ebf8014340fd1906f6b35cfa30173e0ee2fc6c`の実Go CLIを外部リポジトリCIから起動し、公開`semantic-data:sales.order`のSource Index path/fingerprintと原byte照合を限定確認済み。これは**S1の実CLI read-only**であって、外部AI Harness自身の実Tool request/argv、Product Skill root/参照文書の実Loader読取、Source必須全集合、現在Disclosure、Owner受理ではない。

**優先する最短手順**：既存P0-L（旧`*.tei`実Provider Read）／P0-M（新`*.tei.md`のReader/Guard）／P0-J（旧新同一業務の比較）の各Ownerと並行して、AI Skillレーンは**TAI #168のH2 Tool-only最初の1実Session**を最優先で観測する。その後、固定Task/実Binary/Skill bytes/Tool profile/独立期待値を変えず、H2/H3各2 Sessionの計4回で実Tool使用・Skill root/Refsロード・誤答/Warning/Unknown/Costを検収する。割付した条件と実際に提供された条件を分け、失敗Sessionを採点から恣意的に除外しない。初回Tool/Skill比較の途中でOKF用Skill説明を書換えない。

**後続の任意read-only PoC**：[TAI #514](https://github.com/hat47x/tai-pleroma/issues/514)の既存AC-01〜12に従い、正規Computation/typed parameters/実Executor bytes・Job/Receipt/Attesterの実Identityと独立再Readを一件だけ照合する。OKFの`generated`/`verified`/`stale_after`は知識Conceptの来歴・現在性であり、実Actorの認証やTEI Owner Meaning/Host Grantではない。OKFはAttester ABI・wire format・sandboxを定めないため、第二Host/Receipt Registry/新共通CLIを本計画から追加しない。Product SkillのOwnerは**open #2605**、TEI #2614はclosed/duplicate。P0の実Harness・H3ロード・Attester/Job/Receipt E2Eは**NOT RUN**。Go単一バイナリ、既存Shared Acceptance、外部Hostの責務を維持する。

### 外部AI Harnessの実Tool接続済みと比較試験の未適格を区別（2026-11-07）

[AI-first設計§5.10](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)に、従来の「初回H2 Tool実起動は未実施」という状態の**明示的な訂正**を記録した。[TEI実測PR #3113](https://github.com/hat47x/tei-platform/pull/3113)／[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)のDSH 0.2.2-rc.1 Windows headless探索では、固定TEI `50ebf801…`と公開Source/固定Taskで、完了した`h2-08`のTool call 56／result 55、実TEI Binary起動32件とCLI/Source Indexの読み取りを確認した。**外部AI Harness→Tool request→実TEI CLI→Tool Resultの技術的接続は限定達成済み**。PR #3113はEvidenceのOpen/未マージであり、探索9試行中の短い接続確認2、長いTask完了1、Tool JSON連結エラー5、無効Config1は試行別の歴史的証拠として記録する。

**次のBlocking Exitを変更**：最初の「実Tool Callを一件取る」から、**比較資格を持つH2 Tool-onlyを一件取る**へ移す。探索Sessionはread-onlyでも作業Dir外のWorktreeを読めたため、Skill/独立期待値/旧回答の物理非露出・OS隔離が実証されていない。Session logの`enabled:false`は実通信送出0の証明ではなく、Windows `workspace-write` SandboxはWin32 5で初期化失敗。TAI #168の既存SK-02 RunbookをTrial正本として、(1)禁止Path/Canary実アクセス拒否、(2)許可された外部送出と未観測Sinkの分離、(3)固定Task/Source/CLI/Tool/Model/Skill／Reviewer-only期待値、(4)実Tool→回答→Consumer受領・独立採点を同一Sessionで確認、(5)その後H3実Skill root/References Loadと`H2→H3→H3→H2`四Sessionを開始する。探索9試行から成功例を事後選抜せず、**比較適格Trial 0/4**のままとする。#2509がDSH固有の実証、#2515/TAI #232が実OS・開示、#2605がTEI Skill、TAI #168が比較Owner。MALFORMED_RESPONSEはTool連結JSONという実失敗であり、`maxParallelToolCalls:1`でも回避できなかった。異なるProfile/Provider/モデルへの修正は新Pinned Trial Groupで行う。TEI独自Harness Runtime/Runner/Validatorは増設しない。

[main統合 #3051](https://github.com/hat47x/tei-platform/pull/3051)のCompletion評価は、未確立Coverageの空Targetを`unverified`、完全と確認した空Targetの`deny`を`fails`へ区別するよう改善済み。明白な反例が1件あれば部分取得でも`fails`。ただしCaller自身の`CoverageComplete`真偽や全Source/Target/Field/Tenant/Snapshot・実Provider Readを証明する改修ではない。[Draft #2935](https://github.com/hat47x/tei-platform/pull/2935)の`readByLoader`もLoader読取の限定目印であり、Parse/Build/実Consumer読取へ昇格させない。P0-L（既存`*.tei`実Provider）とP0-M（新`*.tei.md` Reader）を並行維持し、#3030は移行Owner、#2514/#2506は独立期待値・母集団検証を担当する。

**現在の証拠と未達**：DSH→TEI CLI技術的接続は探索1件で限定実測済み、#3051 Completionの局所修正はmain統合済み。一方、比較適格H2/H3、H3実Skillロード・効果/コスト、実OS拒否・Egress不在、`*.tei.md` Reader、全Provider対象集合の完全性、Owner意味受理・実Host Effectは**NOT RUN／NOT QUALIFIED**。前節§5.9の「H2実Tool呼出し未実施」は新Evidence以前の歴史状態であり、現在の適格性判定を上書きしない。

### AI-first実装境界の新しい到達点と次の一試行（2026-11-07）

[AI-first設計§5.11](design/AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)で最新mainの実装と実証を接続し、前節の「H2 Tool接続自体は限定達成、比較適格H2/H3は0/4」を維持。**新しい次のBlocking Exit**は、禁止Source/独立期待値/Skill/過去回答への実アクセス拒否と、用途別の実送出範囲が確認された環境で、**H2 Tool-onlyを比較適格な一Sessionとして実行すること**である。TAI #168の既存Trial/Task/割付を唯一の比較正本とし、探索9回の事後選別をしない。

**根拠を持つ実装進展**：[main #3120/#3122]のStudio版付き修正案（あくまで未受理）、[#3143](https://github.com/hat47x/tei-platform/pull/3143)の認可種別`reference-allow-all/reference-allow-list/external`、[#3124/#3127/#3131/#3155](https://github.com/hat47x/tei-platform/pull/3155)の永続化観測`not-applicable/not-observed/receipt-observed/resource-observed`（`persisted:true`は現在Resource一致の場合だけ）を、**個別の製品Evidence**として採用する。実PostgreSQLでのReceipt Lookup/Resource別接続観測の限定経路があるが、Business Completion/current Host Grantは未実証。ツール環境Grantの[main #3121](https://github.com/hat47x/tei-platform/pull/3121)は`PERMISSION_GRANT_CAPABILITY_NOT_RESOLVED`で停止しており、ToolをPluginと決めるかは未決。AI提案→Source変更は依然#267 Shared Acceptance、Host実行は#1174。PoCコード/Fixtureの開発を全部Owner待ちにしない。[採用ADR-0035/#3150](https://github.com/hat47x/tei-platform/pull/3150)に従いA3の機械検証・自律作業と、対外不可逆効果に必要なA4の人的確認を区別し、製品業務Meaning/実Grantの資格を機械PASSで偽装しない。

**実効隔離と送出の反証を先行**：[main #3152](https://github.com/hat47x/tei-platform/pull/3152)のTEI`access.FileSystem`は`os.OpenRoot`でリンク/ジャンクション経由のRoot逸脱を拒否できるが、DSHのShell・子Process・Socketまでは閉じ込めない。Access journalの改変検出も独立headの固定がなければ末尾削除や全再Hashを防げない。[main #3153/#3161](https://github.com/hat47x/tei-platform/pull/3161)のDSH 0.2.2-rc.1 **ローカルMock実測**は、既定Profileでの`dsh_session_log`約45KB送出、Tool markerのモデル入力とは別の2箇所への複写、off→on再開/503/stream切断後のbacklog再送を観測した。これは「`enabled:false`にしたから秘密送出が0」「read-onlyだから外送なし」の否定材料であり、実Providerでの漏えい確定やOS全出口の保護実証ではない。#2515/TAI #232で許可Audience・実測Sink・未測定を同Attemptに記録し、公開用EvidenceへMock rawを無制限に転送しない。

**実Source・Tool資格も別照合**：[main #3148/#3160](https://github.com/hat47x/tei-platform/pull/3160)のCLI discoveryは`modes`/`successShape`を広告するが、実argv・stdout/stderr/exit・Response Mode・実Workspaceを別検証。[Dogfooding Draft #3162](https://github.com/hat47x/tei-platform/pull/3162)は`view effect:read-only`でも`-execute -operation-log`や`-lifecycle-state`によるローカル書込みを観測した。初回H2 Tool allowlistは不要な`-execute`を公開せず、構文上のread-only・参照API・実OS/Host制約を分ける。[main #3163](https://github.com/hat47x/tei-platform/pull/3163)により`modelRevision`は**管理Source Indexの版であってWorldの版ではない**と明確化。空Workspaceで`ok:true`＋`std` entriesが出ても`declarationCounts.total=0`なら必要Subjectの実Loaded/Parsed/Consumed資格は未確立。旧`*.tei` Provider P0-Lと新`*.tei.md` P0-M/P0-Jは引き続き別レーンで進める。

**今回のDone/NOT DONE**：docs/planの契約追補のみ。実Tool探索とMock egress/限定PG・CLIは**既存mainの別HEAD**で観測された結果であり、このPRの同一End-to-End成功ではない。比較適格H2/H3、実OS隔離・全送信拒否、Studio Wails実通過、正式`*.tei.md` Reader、Owner Meaning、実Host・Business Effectは**NOT RUN／NOT QUALIFIED**。新AI Host/Policy Registry/Approval Engine/Runner/Fixture正本は増やさない。PR #2492はmainの[ADR-0035採用 #3150](https://github.com/hat47x/tei-platform/pull/3150)等より古いbaseから発展しているため、統合前にcurrent mainへの差分・旧A4記述と新受入境界の衝突を**別途rebase/レビューで照合**し、旧PRの既存CIを現main資格とみなさない。
### AI-first資格分離の初回実証（2026-11-07、現在mainへの成熟差分）

[AI-first資格・実測対応](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)を横断受入の現行基線として参照する。旧[PR #2492](https://github.com/hat47x/tei-platform/pull/2492)は現行mainから大きく乖離しているため、**全追補を機械移植しない**。本節は成熟したAI受入条件だけを移すもので、同PRに含まれる別件の要求・基礎データ・研究の自動採否ではない。既存Owner/ADR・TAI Runbookと競合する第二の正本・中央Agent Hostを作らない。

**三経路**：AIの自由なMeaning Proposalは#267のShared Acceptance、受理済みComputationの型付きParameter Bindは正規実行物・Receipt・独立Attester、Provider等のEffectful Intentは現在Host Grant/OperationID/Receipt・Resource再Readへ分ける。`validation.Accept`局所フラグ・Studio未受理案、PoCの機械検証によるADR-0035 A3、OKFの`verified`、本番のMeaning/Grant/Effectを相互に代替しない。

**現行実績と制限**：DSHからTEI実BinaryへのTool transportは[#3113](https://github.com/hat47x/tei-platform/pull/3113)で探索限定成功。ただしCatalog Tool Result 8,192 byteの途中終了、Tool call 56/Result 55、stdout/stderr付着、参加者の禁制資料へのアクセス拒否未確立により、TAI #168のH2/H3比較適格は**0/4**。[#3200](https://github.com/hat47x/tei-platform/pull/3200)のTool子Process環境実測、[#3153/#3161](https://github.com/hat47x/tei-platform/pull/3161)のDSH Session Log既定送出・再送は、モデルPrompt以外のSinkと実OS隔離を明示確認する根拠。これらは実Providerの全Egress非漏えいやSkill増分価値の証明ではない。

**Tool選択・Source完全性**：[main #3210](https://github.com/hat47x/tei-platform/pull/3210)ではCLI受理53／公開discovery19という実差があり、非公開34は意図的な選択。AI側がCLIで呼べることと公開Tool/Host資格は別。[Draft #3162](https://github.com/hat47x/tei-platform/pull/3162)の`read-only`表示でも`-execute`でローカル書込みがある反例を残す。SourceはRequired→Discovered→Loaded→Parsed→Actual Consumed、ProviderはRequired Target Set→Observed Result Slice→Predicate Evaluation/Scopeを照合。[現main向けDraft #3238](https://github.com/hat47x/tei-platform/pull/3238)は旧#3175合成Q05〜07を再配置し、隔離PostgreSQLでTenant A `{3,9}`の`$top=2`対`$top=1`を実Reader/独立Expectedで確認（[限定CI #42346176936](https://github.com/hat47x/tei-platform/actions/runs/42346176936)）。**有限Reference実Readを実Product Function→Query全件保証へ拡張しない**。[Draft #3156](https://github.com/hat47x/tei-platform/pull/3156)の実Build成功・Source閉包拒否も、artifactと資格を分ける実例として扱う。

**2026-11-07最新の資格差分**：[TEI #2509](https://github.com/hat47x/tei-platform/issues/2509)の4 Session原EvidenceでH3-A/BのProduct Skill本文各5,482 byte・参照文書の**実Tool Readは限定成立**。しかしH2-Aは旧Session再使用で事前隔離・割付が揃わず、Skill効果/独立回答の比較適格性は**0/4のまま**。新TrialではG0読取拒否→G1許可送出→G2 Tool Result全体性を満たし、Skill Source原byteとLoader wrapper/実消費本文を区別する。[再開の実測 #3237](https://github.com/hat47x/tei-platform/pull/3237)はSession Logの既ACK区間が非再送でも、再開`messages`へ旧Canaryが再出現。**Log非再送≠モデル入力非再開示**なので、再開時Audience/Disclosureを別に照合する。

**CLI/実行時のCurrentness**：[実装PR #3247](https://github.com/hat47x/tei-platform/pull/3247)はWorkspace名・manifest SHA・形式版を5つのCLI応答へ追加して限定実験済みだが未統合で、名称自体は一意IDではなく`source-index`の対象なし応答にも未対応。[Issue #3242](https://github.com/hat47x/tei-platform/issues/3242)は`tei commands`の引数型・enum・default未広告を担当。Hostへ渡すTyped Intentの元となるWorkspace/CLI/Tool Requestを**実解決・実消費した版**で束縛する。[main実測 #3254](https://github.com/hat47x/tei-platform/pull/3254)ではSource属性追加でWorld/Model版が変わってもOperation `basis`と件数は不変で、現実行面に期待Revision照合入口はない。#2598/#1125/#1174のOwnerへ、保存済み操作面の選択時点・実行直前World/Model・現Target/Grantの照合を渡す。**実装方式や既定の拒否判断は未決**、参照`basis`一致・CLI/Studio表示だけを実行資格としない。

**次の小さいDone**：(1) TAI #168／TEI #2509/#2515/#2605で、OSの禁止Source拒否・許可Egress/Tool Result完全性・Answer/Consumer受信を備えた**比較適格H2一Session**を新Trialで成立させる。(2) #2586/#2514/#2506で旧`*.tei`の一Function→実Provider read-onlyと独立期待値を照合する。(3) #3030/#3031で新`*.tei.md` Reader/旧新意味比較、#267で正式Source受理/再Loadへ接続する。(4) #1174/#2598で現在Host Grant・Receipt／Resource観測・業務Outcomeを段階別に確認する。(5) OKF Attested Computationは初回H2/H3比較条件を変更せず、TAI #514の後続read-only PoCへ置く。

**停止線**：隔離・独立採点を満たした新H2/H3の4Session（探索の限定4Sessionとは別）、実非公開Egress拒否、実Providerの全件閉包、`*.tei.md`実Parser、AI正式Meaning受理・本番Host一往復は**NOT RUN／NOT ACCEPTED**。PoC内A3の自律開発を停止させず、対外不可逆効果のA4と製品側資格を別に評価する。[Draft #3179](https://github.com/hat47x/tei-platform/pull/3179)の標準構造・機能内表現のMASTER_PLAN差分とは独立の節とし、merge前に最新mainとの差分を照合する。

**Design Context→T1新負例の最短経路（2026-11-07更新）**：[AI-first資格設計§6](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)へ、#3262の**用途相対Required Design Context read-only投影**を、AI Test Generatorより先の前提として配置。P-1a＝[main #3264](https://github.com/hat47x/tei-platform/pull/3264)の実CLI 9応答／Subject・Source Index一致／`readByLoader:true`／欠落・異版・未読込の3負例は限定実証済み、**全Data/Function/Business/Rule/Procedure閉包・新Context APIは未実証**。T0＝[main #3266](https://github.com/hat47x/tei-platform/pull/3266)の実Binary pinned Scenario 6 Step／実CLI4負例＋独立義務2負例／局所statement C0相当10.1%（C1未計測）。[main #3268](https://github.com/hat47x/tei-platform/pull/3268)でSource/CLI/Skill等変更時のCI再発火範囲を拡張したが、完全な依存Closureや全要求Coverageではない。

**次の新規検証はT1一件**：公開`sales.order`の「行番号は正」のRuleについて、#3262の実消費Source/Rule/Requirementと必要Context/未取得を読取り専用で照合し、**既存Scenarioが網羅していない行番号0の負例**を最小CLI/AI操作で候補生成する。独立Expected/Holdoutを生成Agentへ渡さず、#1180 Scenarioの実CLI実行と#2514の既知Mutant/独立判別で「提案が新しい欠落を検出した」と示す。P-1bの機械／人間両用の整形、T1の新Candidate実行、C1/NFR Coverageは未実装・未検収。案の競合[Draft #3258/#3259/#3260/#3261](https://github.com/hat47x/tei-platform/pull/3258)を第二のTest/Expected/Runner正本にせず、実装Owner #1131/#1126/#1125/#1180/#2514/#3223/#2811へ帰属させる。

**後続・並行レーン**：[三要素Diagram/CRUD Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)は同じSource/RoleからのProjectionにとどめ、Function文脈の表現型Instance、SHO図解・表示/編集を初回T1のBlockingにしない。[基本機能 #3267](https://github.com/hat47x/tei-platform/issues/3267)のVerified Composition BOM／[Reference Draft #3269](https://github.com/hat47x/tei-platform/pull/3269)のexact-lock／pairwise候補は`qualified:false`の計画・表示段階であり、実Plugin組合せPASSや現Host Grantではない。[Snapshot/Cache #3271](https://github.com/hat47x/tei-platform/issues/3271)も実消費・Input/Artifact・現在資格の後続再利用で、新P0 Hard Gateにしない。既存P0-L実ProviderとP0-M`*.tei.md` Reader、および比較適格H2/H3は相互に独立して並走する。**本追加はDocsのみ、P-1b/T1実Run・BOM実Build/Provider/Host・意味受理はNOT RUN**。

### T1-Aの実走とT1-B盲検・Plugin実Consumerの分岐（2026-11-08）

[AI-first資格・実測対応§7](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)へ、既存テストとPlugin-firstの新しい証拠を追加する。**新たな実装達成を意味するのではなく、実証した範囲だけの受入順補正**である。

- **Design Contextの準備**：[main #3296](https://github.com/hat47x/tei-platform/pull/3296)は実CLI9応答＋合成不足/Tool Reply改変16負例を限定CIで検出し、[main #3299](https://github.com/hat47x/tei-platform/pull/3299)はSource/Plugin Lock実byteの変化後に旧Scenarioを拒否した。しかしContextは`partial`、`qualifiedTestGeneration=false`。外部Harnessの切断Tool Result完全性、実Plugin Loader・Operator実消費、Current Grant、全Requirement/Source閉包は別。
- **T1の実績更新**：[main #3304](https://github.com/hat47x/tei-platform/pull/3304)の旧HEAD専用CIは、既存6 Stepと別の一時第7 Step「行番号=0」を実CLIで判別し、偽Expected/Fixture/旧Pinの3負例を検出。これは**既提案境界のT1-A実走**であり、AIが非公開Holdoutを見ずに未知条件を発見する**T1-BはNOT RUN**。#3262の一Subject・用途限定Design Context→#1180 Scenario→#2514独立Expected/既知Mutant→同Attempt受領の最初の盲検試行を次の実験とし、C0 10.1%局所・C1未計測・NFR未測定をそのまま表示する。外部H2/H3比較とは並列に進め、第二Generator/Runnerを作らない。
- **Plugin-firstの次のConsumer**：[main #3277](https://github.com/hat47x/tei-platform/pull/3277)の開発機能Plugin化要求に従い、[#3292](https://github.com/hat47x/tei-platform/pull/3292)のread-only Inspector一機能で**Selected Providerと実Invoked Provider、CLIとStudioの実Consumer**を照合する。2 SurfaceとProvider×Consumer 2×2は別の検証軸。Workbenchの[main #3287](https://github.com/hat47x/tei-platform/pull/3287)は合成Fixtureまで、実Provider/共通Invokeは未実証。標準業務Role/型・Dataの[main #3308/#3318](https://github.com/hat47x/tei-platform/pull/3318)を後続Domain負例に利用するが、DB実Rowとはみなさない。
- **既存Owner・未統合を尊重**：[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)の旧RQ-035 Workbench／RQ-036 BOM記述（現行要求はRQ-035 Workbench／RQ-036 Model Map／RQ-037 BOM）と2×2 Witnessは既存担当に保持。`qualified:false`を正式BOMへ変えない。[Draft #3302](https://github.com/hat47x/tei-platform/pull/3302)のread-only `view -execute`除外案や、closed/unmerged[#3315](https://github.com/hat47x/tei-platform/pull/3315)のWorkspace Identity欄を、現mainに導入済みと扱わない。#3241の設計PR、#3292/#3297の計画Draft、#3303三視点図面Draftは各Ownerの役割が異なり、二重の正本を作らず統合時に最新mainとの差分を比較する。
- **既存Owner・未統合を尊重**：[旧Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)のBOM旧RQ-036/PAS/Gap Audit（現行RQ-037へ再調整する履歴）と2×2 Witnessは既存担当に保持。`qualified:false`を正式BOMへ変えない。[Draft #3302](https://github.com/hat47x/tei-platform/pull/3302)のread-only `view -execute`除外案や、closed/unmerged[#3315](https://github.com/hat47x/tei-platform/pull/3315)のWorkspace Identity欄を、現mainに導入済みと扱わない。#3241の設計PR、#3292/#3297の計画Draft、#3303三視点図面Draftは各Ownerの役割が異なり、二重の正本を作らず統合時に最新mainとの差分を比較する。

**残るP0**：T1-Bの初回盲検Candidate、比較適格H2/H3の4Session、旧`*.tei`一Function→実Provider read-only、新`*.tei.md` Reader/実意味比較は**未成立**。既存局所CI Greenを正式Meaning/Authority/EffectやBuild G4適格に昇格させず、ADR-0035のPoC A3自律実装を不必要に止めない。

### 正常JSONでも必要Ruleが欠ける場合の停止条件とTool実効果（2026-11-08）

[AI-first資格・実測対応§8](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)で、**T1-Bへの入力の実資格と、AIに渡すToolの実効果**を二つの独立したP0に具体化した。先行する[main #3296](https://github.com/hat47x/tei-platform/pull/3296)の9件の実CLI応答＋合成欠落/破損16負例は、正常JSON/同Fingerprintのまま必要Rule・Relationだけ欠落する反例を検査しきらない。[Draft #3324](https://github.com/hat47x/tei-platform/pull/3324)は`inspect-validation`を加えた10応答で、`validation.tei` Loader実読取・Rule/Relation Identity/Canonical Attribute/Source Anchor/Definition Roleを比較し、Tool Reply一時コピー8負例を拒否する変更候補。ただし**Draft/CI未確認、Context`partial`/`qualifiedTestGeneration=false`**であり、実Tool配送・Function/Business全Requirement・新AIテストの資格へ昇格しない。

**実Toolの権限・副作用**：[main #3333](https://github.com/hat47x/tei-platform/pull/3333)は`tei.cli.discovery/v0`に`view.writeFlags=[-operation-log,-lifecycle-state]`を互換追加し、実ローカルFile書込みを限定Go試験で検証した。`effect:read-only`はCanonical Meaning非変更であり、File/Network無変更の証明ではない。H2/T1-Bでは許可Command/Flag/実argv・File/Session Log・Audienceを明示し、Holdout/Reviewer-only資料へ**実OS/Harnessで到達拒否できた**場合のみBlind資格を与える（#2509/#2515/TAI #168）。旧Draft #3302の別提案欄は現mainの実Contractではない。

**次の最小Done**：P0-A=#3262/#3324による一Ruleの独立RequiredとActual Source/Rule closure・不足理由の検査（まずDraft exact-head CI）、P0-B=#2509/#2515のHoldout物理隔離/Tool Result全体性・副作用/送出境界、P0-C=#3273/#1180/#2514の**未知Rule Gapからの盲検T1-B候補一件→独立Expected/Mutantの判別**。一件の狭いSourceが成立しても全Workspaceの`qualifiedTestGeneration`は別。#3304の既知行番号0のT1-A成功をT1-Bに転用しない。設計Contextが`partial`なら取得済みScopeの探索に限り、全域PASSを返さない。従来の適格H2/H3比較と、旧`*.tei`実Provider read-only、新`*.tei.md` Parser移行は**別の並行レーン**。

**後続Plugin/Consumer**：[main #3334](https://github.com/hat47x/tei-platform/pull/3334)の税・為替・採番・休日DataのApplication/Studio投影は実装済みだが、行取得/実DB Bind・Wails E2Eではない。[Draft #3292](https://github.com/hat47x/tei-platform/pull/3292)のInspector一機能で選択Plugin/Providerと実Loaded/Invoked Provider、CLI/Studio二Consumer、Current Scope/Independent Expectedを別々に確認。Studio ABAの[Draft #3327](https://github.com/hat47x/tei-platform/pull/3327)は旧UI応答を採用しない2件の局所成功だけで、Hostの旧Effect取消ではない。Verified BOM/Workbench/Cache/OKF/図面はP0-CのHard dependencyにしない。**今回の変更はDocs/Planのみで、新P0-A/B/C、実Provider・正式Owner受理・Host E2E・C1はNOT RUN**。追加Runner/第二Scenario/Registry/Hostなし。

### TT-56の二候補を一本化し、実Capability選択とBuild主体を分離する（2026-11-08）

[AI-first資格・実測対応§9](design/AI_FIRST_QUALIFIED_INTERFACE_CURRENT_MAIN_2026-11.md)に、**Draftの二つのTT-56実装候補を統合前に一本化する**採否手順を記録した。[#3324](https://github.com/hat47x/tei-platform/pull/3324)は必要な一Ruleの`inspect-validation`/Canonical Attribute/Relation/Source Anchor、[#3326](https://github.com/hat47x/tei-platform/pull/3326)は独立Required 3 Ruleと`source-index` **indexモード**の全件照合を実装候補とする。**両PRの最終固定HEADで、専用Dogfood／Repository Health／依存チェック3系統ともcompleted/success**（#3324 CI #42385606408/#42385606448/#42385606458、#3326 CI #42385726717/#42385726733/#42385726722）。ただし**Draft未マージ・合体HEAD未試験**。#3262を唯一のDesign Context Required/Observed Ownerとし、#3326の独立Required集合＋#3324の特定Rule/Relation/属性/Source Spanを重複なく一つの検証へ照合する。必須でないRuleの省略は正例に残し、重複Caseを二重計数しない。連続するCLI readは原子的Snapshotの証明ではない。

**P0の次の順序**：① #3262でRequired 3 Rule全件Indexと一Rule詳細を実CLI/Source版/返却byteで確認、合同HEAD・main pushは別CI。② #2509/#2515/TAI #168で独立Expectedへの物理アクセス拒否、Tool Reply終端・実argv/Session Log Egressを確認。③ #3273/#1180/#2514でAIが未予告Gapから独立に新Candidateを発見したか**T1-B盲検**を測る。#3304の既知行番号0 T1-A成功を再利用しない。Sourceが`partial`なら可観測Scopeの探索に限り、`qualifiedTestGeneration=false`のまま。不適格なRuleやTool全体を不可視のまま合格としない。

**選択・起動・Buildの実証差**：[main #3337](https://github.com/hat47x/tei-platform/pull/3337)の参照Capability Coverageでは12宣言中6件が**その参照方式では未選択**。後続[Open #3350](https://github.com/hat47x/tei-platform/pull/3350)は既存`SupportsDatabaseProvider`の技術名対応を使って未選択を4件とする改善候補で、現main結果とは異なる。**Declared→Selected Capability→実Plugin Loaded/Invoked→Provider/Consumer適合**は別資格。未選択は必ずしも欠陥でなく、選択済みも実装・権限・BOM合格ではない。[main #3336](https://github.com/hat47x/tei-platform/pull/3336)の子Build駆動記録はInvoking/Invoked executable digest・argv/exit・実Artifactを区別するが`l3Claimed=false`／Source Closure未資格。Plugin-first Inspectorの実Invoke（#3292）とSelf-Build L3（#2831）は後続別検証で、**T1-B開始条件にしない**。

**Tool/標準Dataの横断反例**：[main #3340](https://github.com/hat47x/tei-platform/pull/3340)の`-h` Value/Default とDiscoveryの同一変数化はTool説明の整合改善。実Flag許容・writeFlags #3333・OS/Network送出は別。標準Dataの[main #3343](https://github.com/hat47x/tei-platform/pull/3343)では複数Calendarが一Holiday Dataを共有できるが、Calendar省略の曖昧Readは拒否。**同じData ID／件数でも選択Contextが違えばProviderの実Sliceは別**とする。実DB Row、Studio実Provider、業務完了は未実証。

**新たな実装・試験の上限**：今回の変更は本計画とAI-first設計のDocs更新のみ。合同TT-56、T1-B、比較適格H2/H3、実CLI/Studio共通Provider、Go L3/外部Host/Owner AcceptanceはNOT RUN。新Runner/第二Test/Resolver/Hostを作らず、既存Draftの採否・main追随は個別Ownerが担う。

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

**2026-11-12／アウフヘーベン担当#3・保証境界の統合**：[横断検収記録](design/CROSS_CUTTING_ACCEPTANCE_BOUNDARIES_2026-11-12.md)で、Source原文→Accepted Meaning、Plugin選択→実Invoke、実Read→Consumer、Entry/Trigger→Effect、変更影響→選択的再検証を一つの反証表に接続する。これらは一括PASSにせず、既存#1119のP0-R（#2586）、P0-T（#1251）、P0-H（#2509/#2515）を独立維持する。次段の三要素分析法（データモデル・機能モデル・業務モデル）/CRUD/SHOの意味保持（#1124、Draft #3263/#4479）、Script Launch（#4495/#4497/#4502）、組織C1（#4501、Draft #4500）、変更後BOM再資格（#2600/#3267）を既存Ownerへつなぐ。**設計のみ・追加実験NOT RUN**。正式X-TEA整合、Owner意味受理、実DB/Host/外部Consumer合格、全体G4を主張しない。


**2026-11-11／実ReadとPlugin選択の現在の最短不足を再確定**：main [#3862](https://github.com/hat47x/tei-platform/pull/3862)で`reference/workspace`の`public.orders`用隔離Seedと3/2/0件の独立Expected、main [#3866](https://github.com/hat47x/tei-platform/pull/3866)で型付きM0 Model Mapの局所Source照合が成立した。一方[旧Draft #4076](https://github.com/hat47x/tei-platform/pull/4076)は**別Workspace** `examples/xtea-lowcode`でEnvironment→Database instance→実SQL Executor→Function Data Context→CLI `view -execute`を試みたが、現在は**closed/unmerged**であり、現mainの製品実行経路として扱えない。現在のXTEA SQLは`sales."order"`／`:status`の等価検索で、当該旧Draftの`operator/page-size/offset`等の入力追加はmainに存在しない。両方とも`order_entry.recent_orders`のIDだが**SQL/物理Relation/パラメータ/Fixture/Expectedが別**で、同じ成功票を共有しない。さらに`reference/workspace`側SQLの`:from`は現Query宣言に`input from`がなく、実Binder/型契約未成立。これを[Workbench §23](design/DATA_SOURCE_WORKBENCH_STANDARD_SCOPE_2026-11.md)と[独立Source-static回帰](../examples/data-source-workbench/query_route_separation_test.go)へ固定した。

**P0-Read（#2586）**：まず明示した一つのWorkspaceだけで、隔離実PG Schema/Seed→別Reader独立観測→該当QueryのInput宣言/Runtime Bind→実行Statement/params→Raw Rows/Presence/NULL/Scope→Mapping/Grain→Function Data Context→CLI受領の**一Attempt**を確認する。#4076のopt-in実PG試験は環境変数未設定でskipされた記録を実Read成功に足さない。`declared-page-contract`による`coverage=complete`・`emptyJustified`の推定を、RLS/Role/全母集団の独立完全性や業務不存在に拡張しない。`SourceRevision=unknown`はfreshではない。

**P0-Plugin（#1251/#951/#1016）**：[Draft #4202](https://github.com/hat47x/tei-platform/pull/4202)は既存CompositionでのA/B Entry選択、実Goシンボル照合、Host独立Manifest再観測の局所実装を加えたが、SP-04（一貫した偽Entry）・SP-05（同一Manifest/異Source）を検出できないと明示。正式Profile→対象Binaryに存在する実Provider Entry、同一Load Source Origin、現在Grant、CLI/Wails二Consumerの実Callを別列で検収する。選択ID・DB Provider種別・Host登録名・実Entry・返却IdentityをAdapterが上書きして偽一致を作らない。責務は既存[Composition判断 §13](workstreams/COMPOSITION_CONNECT_POINT_BRIEF_2026-11.md)、実Artifact/BOM #943/#3267、Source #3524、独立Expected #1180/#2514のOwnerに帰属し、第二Host/Registry/Runnerを作らない。M0・Hook・共有Harness/SEI/SUI/SHO/KONは有用な後続Consumerであり、最初の実Readや正式Provider選択の追加Hard Dependencyにはしない。


**2026-11-12／同名Queryの実Readと派生Consumerの検収を現在の実装へ補正（旧実績の取り消しではない）**：P0-Rは引き続き[#2586](https://github.com/hat47x/tei-platform/issues/2586)が唯一の一Function実Read統合Owner。mainの実Sourceは、**Reference** `reference/workspace/applications/order-entry/queries/recent_orders.tei`（Blob `a570bceddc7d61aae62ac315a886cc9aedcac082`）が`public.orders`・`:from`・5列・日時降順で、**XTEA** `examples/xtea-lowcode/applications/order-entry/queries/recent_orders.tei`（Blob `0bf3d516cc81c55493e45a4b525f6dd69bc6ab83`）が`sales."order"`・`:status`・2列という**同IDの別Source／別SQL／別引数／別Expected**である。Reference側は実SQLに`:from`があるがQuery宣言に`input from`がない。main [#4258](https://github.com/hat47x/tei-platform/pull/4258)のSource-static分離と[#4344](https://github.com/hat47x/tei-platform/pull/4344)の引用修正は実証済みの局所成果だが、旧[#4076](https://github.com/hat47x/tei-platform/pull/4076)のEnvironment実行入口・`internal/functioncontext`は**mainへ未統合**。新たな検証・実装の順は、①Workspace＋Query ID＋SQL原byteと入力宣言を固定、②型付き引数・Alias→実Instance/Providerを実Callerで追跡、③独立Seedと実QueryのRaw Rows/NULL/Scope/0件を確認、④Function Data Context→CLI Consumerの実受領を比較。**最初に欠けた辺だけを既存Driver/Bindingで補う**。現在[PR #4465](https://github.com/hat47x/tei-platform/pull/4465)のApplication DB要件未bind警告は未マージの修正候補で、`validate ok`やwarningの有無をRuntime Bind・Provider実Read成功と解釈しない。

**派生情報は「表示に成功」より「問いに必要な情報を保持したか」で検収する**：[main #4468](https://github.com/hat47x/tei-platform/pull/4468)は`overview`で読取可能／不能の種類を明示し、[main #4469](https://github.com/hat47x/tei-platform/pull/4469)は`trace`の2種類の未解決を`unresolvedScopes`で区別する。[実CLI調査 #3900](https://github.com/hat47x/tei-platform/pull/3900)では、`reference/workspace`の対象行の`unresolved=0`とGraph投影時の`projectionUnresolved=17`が**同時に成立**した。対象行に未解決がないことを、Graph／CRUD／仕様書／AI Contextの依存閉包・全Relation完結へ昇格しない。各Consumerは対象のOccurrence、必要なRelation/依存集合、Source/Model Revision、観測範囲・omitted/unknown/非開示、実受信を既存[#3262](https://github.com/hat47x/tei-platform/issues/3262)/[#2516](https://github.com/hat47x/tei-platform/issues/2516)で用途別に対照し、欠落を空集合や意味不存在としない。

**SHO/SUIへ渡す実装済み範囲を明記**：[merged #4317](https://github.com/hat47x/tei-platform/pull/4317)とSHO [#36](https://github.com/hat47x/sho-composer/pull/36)は、TEI受理Sourceから一要素・一関係を**一Workspace Load**で文書Draftへ投影し、SHOで編集・保存・再開、TEIで再照合する限定経路を実装。SHO側[相互実CLI CI #42995269509](https://github.com/hat47x/sho-composer/actions/runs/42995269509)は当該一往復に限りsuccess。一方、`three-element-diagram`文書種別は入力JSONを投影し**TEI Workspaceを読まない**。文書保存や`frozen`が実Modelの全要素・全Relationの自動受理やSource書戻しを証明するわけではない。[main #4445](https://github.com/hat47x/tei-platform/pull/4445)のSUI向けRecord/Disclosure/Scopeは本文を運ばない**Reference**で、SUIの業務意味をTEIへ直接変換した資格ではない。これらの成果は[#1124](https://github.com/hat47x/tei-platform/issues/1124)/[#2497](https://github.com/hat47x/tei-platform/issues/2497)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Expectedへ引き渡し、AI選択の固定・共有Harness Session・現在Grantを混同しない（進行中[Draft #4473](https://github.com/hat47x/tei-platform/pull/4473)/[#4477](https://github.com/hat47x/tei-platform/pull/4477)/[#4478](https://github.com/hat47x/tei-platform/pull/4478)は各別Owner）。**P0の実Read・正式Plugin実CallをSHO/SUI/共有Harnessの実装完了待ちにしない**。第二Model/Graph正本、Reader/Resolver/Host、Test Runner、横断Grant Engineを追加しない。今回の新Product実DB/GUI/外部Harness/Owner意味受理の横断試験はNOT RUN。

**2026-11-12／標準PreludeとImported Provider由来を実Readの必要閉包へ分解（#2620・#3262・#2586の所有範囲を維持）**：main `internal/stdlib/semantic.go`（Blob `b460998679c5e58cc3f637f063783636462bfc3a`）は12個の`std.*` primitive型を**Semantic Coreの種別ではなくGoプロセス内の標準Library Prelude**として構成する。XTEA側`recent_orders.tei`の`input status std.code`を検証するとき、`std.code`がSource Indexに存在しないことだけで「型／意味の不存在」「必須Source未取得」と判断しない。ただしBuiltinで解決できたことは**独立したOwner Meaning受理、Imported Sourceの実消費、SQL `:status`実Bind、Provider実Read**の証拠でもない。現PreludeがCompiled Standardである限り`standard-definition-origin=builtin-prelude`相当の**既存証拠票での分類**と実Build版を対照し、標準型の自己記述Sourceへの置換は[#2620](https://github.com/hat47x/tei-platform/issues/2620)のK0→K3受入へ委譲する。[Draft #4485](https://github.com/hat47x/tei-platform/pull/4485)の自己記述設計は**未統合・未実行**であり、`std.*`の現在の宣言経路が自動的にSource化したとは扱わない。

**外部Sourceの鮮度は、参照している必要閉包だけを判定する**：[merged #3479](https://github.com/hat47x/tei-platform/pull/3479)はImported ProviderのRaw／Manifestだけ変更してもConsumer World/Source Indexが不変となる局所負例を確認した。同一`order_entry.recent_orders`のProduct Readでも、**要求されたProvider/Language Plugin/Importが実選択・実消費された場合**だけ、そのSource Revision/Manifest・Artifact・実InstanceとSource/SQL引数を同じAttemptに照合する。Consumer側World一致をFreshnessや正しいProvider実Invokeへ昇格させない反面、**そのFunctionに不要な別Providerの改訂で無関係なRead候補まで一律失効させない**。必要性がSourceから未判別の場合は対象Claimだけ`unknown`とし、read-onlyのRaw探索・不変のQuery Sourceの事実は保持する。具体的な受入対照は①`std.code`が組込みPreludeで型解決しても`status`のSQL Bind未観測ならRead未資格、②必要ImportのManifest/Raw変更でWorld不変でもRead資格を再評価、③非選択Providerだけの変更では過剰失効させない、の3件（**未実行**）。#3262 Required Context、#1251/#951実選択、#2514独立Expectedへ引き渡し、新Prelude Registry／Resolver／Host／Role Engine／Runner／Hard Gateは追加しない。P0の実Function Provider ReadをこのK1〜K3自己記述移行で待たせない。

**2026-11-12／実Readの一Attempt証拠とGit外保管・CI再実行の境界（#2586の既存検証票を拡張）**：main統合済み[性能・Evidence保管 #4491](https://github.com/hat47x/tei-platform/pull/4491)では、Gitに最小Manifest・独立Expectedを置き、Raw Trace/Rowset等をLocal CAS/NAS/S3の許可された保管先に分離する。これを`order_entry.recent_orders`のP0-Rへ適用する場合、**保管場所と証拠の資格を分離**する。同一AttemptのWorkspace Root/Query Source byte・実Binary/Provider Artifact・選択DB Instance・型付き引数・実Read Scope/Cohort・Result/Field Presence・Consumer受領・独立Expected・Disclosureを既存台帳の参照先へ対応づけ、**必要なRawに実際にアクセス・再検証できたか**を別判定する。Content Digest一致だけで現在Grant、データの完全性、別DB・別Toolchainでの再利用、SourceのA→B→Aの中間履歴を証明しない。Local-onlyは正常な選択肢であり、公開Git・外部HarnessへのRaw SQL/Parameter/Rowset・NAS URLの無条件送出を要求しない。

**試験判定と保存Logを同一Attemptに固定する**：[Draft #4492](https://github.com/hat47x/tei-platform/pull/4492)はCache検証CIで判定用`go test`と証拠保存用`go test ... || true`の**二回実行**が混在し得ることを指摘し、一回の実行を`tee`で保存する修正候補（未統合）である。P0-Rの最初の実PostgreSQL/CLI検証にも同じ注意を適用し、**CI Verdict・保存されたstdout/stderr・Source/Toolchain/Provider版・実DB Target/Attempt**を一回の実行に束縛する。部分/失敗時にも診断Logを残すが、後続の再実行結果で元Attemptの成功・失敗を改ざんしない。最小負例は①同HEADで二回実行し片方だけ失敗、②同Source DigestでDB/Grantだけ変化、③保管Manifestは存在するがRaw Blob欠落／受信Reply不完全、の3つ（既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)のIndependent Expectedに委譲、**新実DB/CI横断試験NOT RUN**）。#4491の保管方式導入や#4492のCI修復を**P0実Readに先行する新Hard Gateにはしない**。新Evidence Registry・Host・Runner・Telemetry Storeは追加しない。

**2026-11-09／Traitの現main資格と受入統合（既存P0の更新）**：[原子的Trait設計§21](design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)で重複した旧§18を整理し、main #3540の**静的Inspector S2-reference成立**（通常CLI／Wails Host経路）と**正式ProviderでのS2-product／S3未成立**を一つの基線に統一した。main #3728/#3754の`model catalog.ok`と`validation.ok`の分離、#3761のHook結末語彙の境界拒否（実Plugin/DB効果は未証明）、#3757のEnvironment六Bindingの解決と実行到達の差、#3760の呼び手指定catalogによるself-build in-process計画導出（`qualified:false`／L2-b未認定）を、**Trait Required→選択→実Consumer→独立Expected→現在Grant**の別資格に接続した。次のP0-Aは#951/#1016の正式選択Provider Artifactを通常CLI／Studio双方で`selected B=actual B`へ接続する一辺、P0-Fは#3279の原子`manual-required`欠落検出。TQ-01〜08は#1180/#2514への未実行反証候補。RQ-036、#3267 BOM、#2831 Buildは既存Ownerを維持し、新Registry/Host/Runner/Hard Gateは作らない。

**2026-11-09／Trait実装義務の到達点更新（新規Hard Gateなし）**：[Trait設計§21.5](design/TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)へ、main #3828/#3843の**Hook `uses`→参照Command Caller実Call**と未実行`Uncalled/NotEvaluated`の差、#3802の`produces`候補表示、#2784のDispatch直前期間Guard、#2812の同一Provider Loadに束縛した非公開parsed Source（ただし`FullInputClosure=false`）、#3839の実CLI Build閉包、#3848の`attemptedCount/complete`（試行した分の全件成功のみ・Required集合完全ではない）を接続した。`CallOutcome.Called=true`は境界到達の申告であり、実Provider/Effect/Commitではない。RQ-036のAI向け「現在地→Trait Required→実装・試験所在」は既存#3847／#1124の導線を使用し、第二Registryを作らない。**TFX-01〜07は未実行の受入観点**で、P0-Aの正式Provider実Artifact→CLI/Wails実Invoke、P0-Fの`manual-required`欠落診断を引き続き先行する。実Plugin/DB・現在Grant・BOMを合成参照で代替しない。

**2026-11-10／Inspector S2-referenceのCanonical／Workspace境界をmainへ統合（#1251、#3181、#2586）**：[PR #3735](https://github.com/hat47x/tei-platform/pull/3735)（統合`e25278d59f5b9b6174d92191dcde4b266394a559`）で、`operations.ValidateInspectionProjection`が**要求Subject IDとCanonical top-levelの小文字・単一`id`を厳密一致**させ、異ID・欠落・大小文字別名・重複キー・JSON null/arrayを拒否する。 CLI／StudioのProvider注入境界にも反例を追加。同一Provider・同一Subject IDの二Workspace A/Bを実際にBind／Studio Bridgeで別々に読ませ、AのSourceファイルを移動するとAのWorld/Canonicalは不変だがSourceIndex FPとArtifactPathが変化、Bは不変という局所対照を追加。 PR exact-head `c70761c71e0d79d869a0f91c7a9c6726d6457541`の[Repository Health](https://github.com/hat47x/tei-platform/actions/runs/42578414117)、[Wails Host](https://github.com/hat47x/tei-platform/actions/runs/42578414227)、[Studio参照編集](https://github.com/hat47x/tei-platform/actions/runs/42578414161)、[Windows WebView](https://github.com/hat47x/tei-platform/actions/runs/42578414217)、[実CLI](https://github.com/hat47x/tei-platform/actions/runs/42578414183)はsuccess。 **これはS2-referenceの入力／投影整合とHostが束縛した二Rootの局所観測まで**。悪意あるProviderの異Root虚偽応答、Importのsame-Load raw Origin、正式ProfileからのProvider B→B実Entry、実Build Member・現在Grant・Disclosure・Owner意味受理・外部Tool受領は未検証であり、PR exact-head CIと統合後main CIの資格は別。

**二つの独立したP0とProvider identityの非洗浄原則**：Function一件の実Read（#2586）は`order_entry.recent_orders`について**Function→Query/Relation→`:from`付き実SQL→実PostgreSQL Provider→Rowset→Data Context→Consumer受領**を同一Attemptで実証する。現[Draft #3862](https://github.com/hat47x/tei-platform/pull/3862)はProduct SQLに合わせた隔離Seed／3・2・0行の独立Expectedを準備した**Source-static**成果で、実DB接続・パラメータ束縛・Rowset・Function Contextは未実行。 並行するPlugin S2-product（#1251/#951/#1016）は選択Providerと実Invoke Providerの同一性、別Artifact/Host Grantの資格を実証する。両者を同じ完了票へ合算しない。別系統のmain [#3936](https://github.com/hat47x/tei-platform/pull/3936)／[#3945](https://github.com/hat47x/tei-platform/pull/3945)が**観測Provider ID／配備Executor IDをAdapterが無条件に上書きして同一性拒否を無効化する不具合**を修正した事実を横断負例に採用し、`selected`／`registered`／`returned`／`actually-invoked`のIdentityを互いに上書きして整合を捏造しない。実Providerの真のプロセス／DB／Receiptは独立に追跡する。

**受入の上限**：現行の12種類のInspector Canonicalはtop-levelの小文字`id`を持つが、**その文字列が要求IDと一致することと、要求Kindに属する実Sourceから導出されたことは別**。同じIDはData・Relation・Query等の別Kindでも現れ得る。選択済みProviderのJSONを検査しても、実Source Root・Parser消費byte・Providerの実体・独立Grantを自動認証しない。対照の独立Expectedは既存#1180/#2514を利用する。

**次の一辺は変えない**：既存#951/#1016で選択したProviderとHost登録の**実Artifact Entry**を、通常CLIとStudioの二Consumerで同一Attemptに結び、B選択→A誤呼出しを拒否する。Workspace Source Revision・Manifest・Provider raw、Source Read Root、現在Host Authority、Function→Query→Provider Rowset（#2586のM1b/R0）を各別資格で観測する。C-1/C-2/C-4やKON/SHOを一律必須としない。Independent Expectedは#1180/#2514、Build/BOMは#3267が所有し、第二Resolver・Host・Runnerを増やさない。

**2026-11-09更新：Plugin-first InspectorのS2-referenceをmainへ統合。** [PR #3540](https://github.com/hat47x/tei-platform/pull/3540)（統合`2eeec81c36db6dfe8c4b6f5c464e0280f8facd87`）により、前回の限定S1（main #3381）から、通常`tei inspect` dispatch／通常・生成Studio Host／Wails Bridgeが**共通Contract・静的参照Provider・既存Composition**を使用する経路まで前進した。PR exact-headで全体Health、Wails Host、Windows Desktop WebView、Studio参照編集、Action Surface CLIが成功。実装・証拠の現在地は[docs/status.md](../docs/status.md)を正本とし、[D1 §15](evidence/PLUGIN_FIRST_D1_AUTHORING_ROUTE_FIT_GAP_2026-11.md)はその実装手順の履歴とする。下記に残る「通常CLI/Studio未配線」「Draft #3381」という文言は**記述当時の旧状態**であり、新しい現行状態へは読み替えない。

**次のP0はS2-product・S3の不足分だけに限定する。** まず#951/#1016の**既存Profile/Provider選択**から、固定された実Build Artifact/実Provider Entryへ束縛し、通常CLI・Studioの別Consumerで`selected B → actual B`と`selected B → actual A`（出力同一でもFAIL）を独立Expectedで再検証する。続いてProvider Source raw/Manifest・選択Ruleの変更、現在Host Grant、Required Source/Role/Disclosure、2×2 Verified BOMと実Build memberを各別資格として照合する。Host実行ファイルから読めたSHA-256とPlugin実Link/署名・本番認可を混同しない。Trait Required/Hookは実装義務と実経路でのみ資格化し、Graph描画やAI統計の自己申告を実呼出しに昇格しない。CarrierのC-1/C-2、外部Host C-4は用途別に選択可能なまま維持し、新Resolver・共通常駐Host・第二Authority・新Hard Gateを増やさない。実験のOwnerは#1251/#951/#1016/#1824、独立Expectedは#1180/#2514、BOMは#3267。KON/SHO/SUIを通常のTEI読込みに必須の実行依存にしない。

1. （完了、2026-09-19）判定段階Bの最終的な実環境での検証根拠を、同じ版（R = `e5231839`）についてそろえた
2. （完了、2026-09-19）引継ぎの基準`B`と対象版`R`を固定した
3. （完了、2026-09-19）最初のWeb向け実行先をビルド版`T1`（T = `0ba87a4e`）として実装し、判定段階Cの検証根拠を得て`Ready`と判定した
4. 判定段階Cの`Ready`後に継続する既存の検証レーン（現在地。新規実装は上記の外部要件机上評価→基盤→CUI・標準プラグイン並行→ヘッドレス一気通貫→GUIの順序を優先する。各課題の状態は、その課題の`状態`欄を正本とする）
   - `ISSUE-000046`：編集環境での最小限の記述・変更・受理の流れ
   - `ISSUE-000047`：既存知識を優先する意味情報の探索。厳密・決定的な方法を基本とし、必要に応じて軽量な認知機構からモデルへ段階的に進む
   - `ISSUE-000027`：AIによる提案と受理。手段によらず最小限の意味変更を共通の受理境界で扱う
   - `ISSUE-000026`：二つ目の実行先による検証。実行先の違いだけでなく、可搬な実行物と実行環境の違いも比較する
   - `ISSUE-000053`：AIが開発を再開し、業務知識を利用するための仕組み。契約、実装、設定、現在の権限、観測結果を再現可能にする
   - `ISSUE-000078`：外部の参照モデル全体を移し替え、表現力、失われる意味、暗黙の前提を横断的に検証する
5. 二つ目の実行先、定義の記述、AI、参照モデルの移行で得た検証根拠を相互に照合し、実行先からの独立性、記述方法、意味の拡張の境界を再評価する
6. 本番運用と外部技術との連携の境界を、必要な検証根拠とともに広げる

判定段階BとCの`Ready`判定と過去の検証根拠は、それぞれの対象版に固定されています。現在は、判定段階Cの後に進める定義の記述・知識の活用・AI・二つ目の実行先の作業を重視します。以前の判定対象版で得た根拠を、現在の作業に対する検証根拠として無条件に引き継ぐことはしません。

### 2026-11-08 一機能で検収する横断の最短経路（既存Ownerを横につなぐ実装順）

判定段階B/Cの確定記録とマイルストーン5〜7の並行方針は維持する。そのうえで、直近の設計・実装・他製品連携の議論を、**一つの機能を通して最初の未接続箇所を特定する順序**へ収束させる。詳細契約・試験の正本をここに複製せず、各Issueを担当とする。本節は優先する**検証計画**であり、新しい製品Gate、Runner、Parser、Plugin Resolver、Host、意味の承認機構を設ける判断ではない。

最初の対象候補は`order_entry.recent_orders`と関係する`sales.order`の一改訂とし、**同一Workspace名・Subject名だけで各出典や実Providerを接合しない**。Source/外部解析/設計/実Read/表示/Actionは、依存する実Revision・Scope・Attemptで比較する。Sourceを読み込めない場合でも、確認済みの局所情報は保持し、全体完了・不存在・実行許可は保留する。

| 順序・同時実行性 | 次に実証する「一つの辺」 | 判定・引渡し先 |
| --- | --- | --- |
| **P0-D**（他P0と並行） | 管理対象ファイルと、実Loader/Parserが読んだSourceを区別し、Data/Function/Business/Ruleの必要閉包とSource位置を返す。通常Markdown、正式`*.tei.md`候補、従来Source、外部逆解析Rawの資格を混同しない | [#2621](https://github.com/hat47x/tei-platform/issues/2621)／[#3262](https://github.com/hat47x/tei-platform/issues/3262)／[設計取込の後継Draft #3483](https://github.com/hat47x/tei-platform/pull/3483)（旧#3355は歴史的Draft）。main [#3354](https://github.com/hat47x/tei-platform/pull/3354)の11実CLI・32限定判別を基線に、[main #3482](https://github.com/hat47x/tei-platform/pull/3482)では`sales.order.submit`の3 Rule・Capability/Application/Sourceの限定閉包について13実CLI・42 Consumer対照が専用CIで成功。Function実Invoke/実Provider Read・全Requiredは未成立 |
| **P0-M** と **P0-R**（互いを待たない） | **M**：実Source→Data/Function/Business三視点とCRUD→元Relation/CellのSource locator。**R**：`order_entry.recent_orders`を初回対象に、**Pattern→Query Source／SQL raw→実Provider選択→実SQL+args→物理Rowset／Semantic Mapper→Function Data Context**の各辺を、独立Expectedと同一Attempt/Scopeで検収。`sales.order.submit`の宣言・Validation成功をBrowse実Readと合算しない。Graph表示・別DomainのExecutor成功・fake SQL Connectorを実PostgreSQL取得としない | M=[#3263 Draft](https://github.com/hat47x/tei-platform/pull/3263)／[#1124](https://github.com/hat47x/tei-platform/issues/1124)。全容Map要求は[main #3505（RQ-036）](https://github.com/hat47x/tei-platform/pull/3505)で採用済み、既存Worldからのread-only typed M0は[Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)で検証中（`partial-projection`であり実三視点/CRUD・State/Hook・Consumer全体の完成ではない）。R=[#2586 HJ-09](https://github.com/hat47x/tei-platform/issues/2586)／[#2506](https://github.com/hat47x/tei-platform/issues/2506)。[Draft #3500](https://github.com/hat47x/tei-platform/pull/3500)は**別Domain Native Case**の既存Executor/Mapper＋instrumented Connectorの局所Go Testが成功したが、隔離PG試験は**SKIP**。**P0-C**で同じSource cohort・Scope・Read時点・Consumerが揃うときだけ接合 |
| **P1-P**（P0-Dと並行試行可） | 開発ツールのread-only Inspector一操作を、CLIとStudioで **Contract→Provider選択→実ロード→実Handler Invoke→結果**まで共通契約でつなぐ。選択B／実呼出Aは結果が同じでも不適格。Commandのread-only表示と実argvのFilesystem効果は別に測る | [#3277 merged](https://github.com/hat47x/tei-platform/pull/3277)は設計・要求の基線。[#951](https://github.com/hat47x/tei-platform/issues/951)／[#1251](https://github.com/hat47x/tei-platform/issues/1251)が実装・適合のOwner。[main #3381](https://github.com/hat47x/tei-platform/pull/3381)は型付きInspector注入・Host選択Go呼出を限定実装したが、通常CLI Dispatcherと実Studio Bridgeが同じ選択済みProviderを使う製品S2は未実証（[main #3531](https://github.com/hat47x/tei-platform/pull/3531)）。認証方式の選択・実Principal・Identity Link・Session・業務認可を分ける試験は新規[#3412](https://github.com/hat47x/tei-platform/issues/3412)および既存#1043/#1174が担当し、業務Roleや現在Grantへの自動昇格は禁止 |
| **P1-T**（必要な設計が得られた範囲） | 設計入力の**D**（必要Source/Role/Ruleを実取得）・Tool/Consumer受領の**T**（Replyの完全受信）・生成候補の**G**（独立Expected/未知Rule/Mutant判別）を別評価。既知T1-A合格から未知T1-BやC1を認定しない | [#3262](https://github.com/hat47x/tei-platform/issues/3262)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)／[#1180](https://github.com/hat47x/tei-platform/issues/1180)。main [#3477](https://github.com/hat47x/tei-platform/pull/3477)の**同一Workspace** Source-only失効試験と、外部Import Provider原文・Manifestの改訂がConsumerの指紋に反映されない[main #3479](https://github.com/hat47x/tei-platform/pull/3479)の限定Go反例は別資格として扱う。同一Nested Provider Loadのprivate原文証拠は[Draft #3524](https://github.com/hat47x/tei-platform/pull/3524)で前方移植中であり、実Consumerの完全なSource引用資格を自動回復しない |
| **P0-D**（他P0と並行） | 管理対象ファイルと、実Loader/Parserが読んだSourceを区別し、Data/Function/Business/Ruleの必要閉包とSource位置を返す。通常Markdown、正式`*.tei.md`候補、従来Source、外部逆解析Rawの資格を混同しない | [#2621](https://github.com/hat47x/tei-platform/issues/2621)／[#3262](https://github.com/hat47x/tei-platform/issues/3262)／[設計取込後継Draft #3483](https://github.com/hat47x/tei-platform/pull/3483)。main [#3482](https://github.com/hat47x/tei-platform/pull/3482)は13実CLI応答・42限定判別でCapability／Application／Sourceの一部Requiredを実照合済み。ただし全Source閉包・Function実Provider呼出しではない |
| **P0-M** と **P0-R**（互いを待たない） | **M**：実Source→Data/Function/Business三視点とCRUD→元Relation/CellのSource locator。**R**：同一Product Function→実Query/args→実Provider Rowset→Function Context。Graph表示や別Reference SQLの成功を実Product Readとしない | M=[#3263 Draft](https://github.com/hat47x/tei-platform/pull/3263)／[#1124](https://github.com/hat47x/tei-platform/issues/1124)、R=[#2586](https://github.com/hat47x/tei-platform/issues/2586)。**P0-C**で同じSource cohort・Scope・Read時点・Consumerのときだけ接合する |
| **P1-P**（P0-Dと並行試行可） | 開発ツールのread-only Inspector一操作を、CLIとStudioで **Contract→Provider選択→実ロード→実Handler Invoke→結果**まで共通契約でつなぐ。選択B／実呼出Aは結果が同じでも不適格。Commandのread-only表示と実argvのFilesystem効果は別に測る | [#3277 merged](https://github.com/hat47x/tei-platform/pull/3277)は設計・要求の基線。[CLI入口Draft #3381](https://github.com/hat47x/tei-platform/pull/3381)／[Studio入口Draft #3371](https://github.com/hat47x/tei-platform/pull/3371)は型付きService Port・誤応答・欠落時fallback拒否の候補であり、**実Compositionのexact Provider Binding／両Consumer実Invokeは未接続**。[#951](https://github.com/hat47x/tei-platform/issues/951)／[#1251](https://github.com/hat47x/tei-platform/issues/1251)が実装・適合のOwner。認証方式・Principal・Identity Link・Session・業務認可は[#3412](https://github.com/hat47x/tei-platform/issues/3412)と#1043/#1174で別検証し、現在Grantへ自動昇格しない |
| **P1-T**（必要な設計が得られた範囲） | 設計入力の**D**（必要Source/Role/Ruleを実取得）・Tool/Consumer受領の**T**（Replyの完全受信）・生成候補の**G**（独立Expected/未知Rule/Mutant判別）を別評価。既知T1-A合格から未知T1-BやC1を認定しない | [#3262](https://github.com/hat47x/tei-platform/issues/3262)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)／[#1180](https://github.com/hat47x/tei-platform/issues/1180)。main [#3477](https://github.com/hat47x/tei-platform/pull/3477)の**Consumer管理Source-only**失効と、[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)の**別Import Provider Raw／Manifest-only**改訂（ConsumerのWorld/Index二指紋が不変）を区別。Provider実原文・ManifestのRevisionは用途相対に独立照合し、別Load時点を同一Snapshotと扱わない |
| **P1-C**（上記の結果を用途別に渡す） | 既存資産逆解析からの**観測事実→構造候補→Ownerが受理した意味→正式Authoring**を分離。SHOは出典付き文書のRead/候補Patch/正式Save・再Readを段階確認する。外部認知・意思決定へ渡すときも、局所即応・意思決定Intent・現在許可されたAction・実Effect/Receiptを別にする | [REA #3394](https://github.com/hat47x/tei-platform/issues/3394)／[TEI-EKI設計Draft #2744](https://github.com/hat47x/tei-platform/pull/2744)／[SHO計画 #2](https://github.com/hat47x/sho-composer/issues/2)。SHOの限定Reader/Plannerに加え、Working Copy保存・別Open [main #25](https://github.com/hat47x/sho-composer/pull/25)、公開TEI原文Reader [main #27](https://github.com/hat47x/sho-composer/pull/27)、Save As相対リンク差分preflight [main #28](https://github.com/hat47x/sho-composer/pull/28)、Editor UTF-16選択→Raw UTF-8位置写像 [main #29](https://github.com/hat47x/sho-composer/pull/29)が限定試験として統合済み。ただし正式Source Writer／実Editor Save/reopen／TEI Meaning受理・現在Disclosureは別資格 |
| **P2-B**（P1-Pの実呼出し成立後） | 実Plugin 2版×Consumer 2版の**4つのexact tuple**を、実Build・Invoke・独立Expectedでそれぞれ評価。PASS/FAIL/SKIP/NOT RUNを保持し、Current Host Grant・実DB Effect・Consumer開示は追加資格として検査する | [Verified BOM #3267](https://github.com/hat47x/tei-platform/issues/3267)／#951／#2514。[Draft #3379](https://github.com/hat47x/tei-platform/pull/3379)のCapability判定規則変更による旧Evidence失効もProvider版×Consumer版とは別の前提。候補一覧・静的Version一致・2件の試験・限定TEI Self-build L3を「互換な4件」としない |

**テスト生成のP0実証との接合（2026-11-08更新）**： [Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)のHEAD `e8745c6c`は[専用dogfood CI #42465435223](https://github.com/hat47x/tei-platform/actions/runs/42465435223)、全体Health、依存確認がsuccess。既存13件の実CLI read-only応答、42件の限定Context対照、6実Pinned Scenario Step、**9つのtest-only受領対照**を同一Runで検証した。独立したStep→Operation期待値が、ExpectedとReceiptを同時に誤変更するTT-154を拒否する。これは`test-ability`の**局所正証人**であり、正式Function実Invoke・Data Context/実DB Read・外部完全受領や盲検T1-Bの証拠ではない。詳細Expected/Caseは[#2514](https://github.com/hat47x/tei-platform/issues/2514)、Scenarioは[#1180](https://github.com/hat47x/tei-platform/issues/1180)、受入票は[既存テスト基本設計§35](https://github.com/hat47x/tei-platform/pull/3223)を参照し、このMasterには複製しない。[main #3502](https://github.com/hat47x/tei-platform/pull/3502)のTrait/Provider二Consumer設計や、[main #3505](https://github.com/hat47x/tei-platform/pull/3505)で採用済みの全体Map/RQ-036要求も、実Entry/Rowset・現在Grant・全Domainの不存在証明を代替しない。Mapを表示する場合は**観測済・未取得・部分取得・非公開・未実行**の状態を対象Scopeごとに明示し、表示カスタマイズで実Coverageや権限の判定を変更しない。Source-only/Manifest-only失効の[PR #3479](https://github.com/hat47x/tei-platform/pull/3479)は書式修正後HEADのCIがsuccessだが、外部Providerの原文をConsumerが実消費した同一Attempt/Source閉包は依然別資格である。

**次に通す実製品の最小3 Probe（2026-11-08 現行mainとの差分）**：詳細手順・Independent Expectedは既存Ownerの#1180/#2514へ置き、本節は順序と停止条件だけを示す。旧局所CIのPASSを集計して全体PASSを作らない。

| Probe | 最小の正常/負例と、最初に失敗した場合の停止箇所 | 次の実装を所有する場所 |
| --- | --- | --- |
| **D1-S2／実Inspector** | 同じScope/Contractに対するProvider Bの選択から通常`tei inspect`と実Studio Bridgeの**両Consumerのactual B Invoke**を記録し、B selected→A invoked、Host登録解除/版ずれを拒否。同じJSON・注入Helper Go Testだけでは不可 | #1251/#1824。main #3381/#3531のS0/S1を再実装せず、最初に途切れる**通常製品Dispatch→実Provider**を測る |
| **M0→M1／モデルの現在地** | 同じ`workspace.Load`とSource Indexに由来するData/Function/Business・Relationのtyped IDとSource位置を提示し、同名別ID・多重Edge・未知Event/Stateは欠落診断。M0成功を全容Map/CRUD・可逆編集・Generator成否としない | 採用済RQ-036・#1124とDraft #3518/#3263。まずread-onlyの`partial-projection`を実Consumerで受け、実Loader→Graph/CRUDの未接続辺を特定 |
| **HJ-09／Product実Read** | `order_entry.recent_orders`のPattern→Query raw/args→選択・実呼出Provider→実SQL/Rowset→Mapper→Function Contextを同じAttemptで照合。Rowset 0件/partial、Provider B選択→A実行、接続失敗を正常な不存在としない。物理PostgreSQLが未実行なら`NOT RUN` | #2586/#2506。別Domain Go ExecutorのPASSやInstrumented/Fake Adapterを本Functionの物理Readとしない |

この3 Probeは**D1-S2とHJ-09を並行優先、M0/M1をread-onlyで並行**させる。どれか一つが不適格でも確認済み局所Claimは残し、未取得・実行不能を空集合／不存在／権限ありへ変換しない。共通APIのTransport同値、Plugin Selection/actual Invoke、Graph Projection、DB Read、外部Reference currentnessは別の資格として照合する。Lifecycle/Hook Graph（#3503、[Draft #3507](https://github.com/hat47x/tei-platform/pull/3507)）、AI性能観測[main #3520](https://github.com/hat47x/tei-platform/pull/3520)、完全Headless B2C #3509、SHO/KON連携[Draft #3528](https://github.com/hat47x/tei-platform/pull/3528)は異質な**後続/並行Consumer試験**であり、この一Function実証に新しい常駐HostやCore依存を追加しない。

**要求IDの衝突を明示**：全体MapのRQ-036はmain [#3505](https://github.com/hat47x/tei-platform/pull/3505)で採用済み。[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)が過去に用いた「RQ-036=Verified BOM」は現mainの要求番号と衝突している。BOM要件の採用・再採番は#3267/#1119のOwnerへ委譲し、旧Draftの要件文で現RQ-036を置換しない。2×2適合の実証は引き続き未成立である。
**製品要求とBOMの受入同期（2026-11-08／[RQ-037](../requirements/TEI_PRODUCT_REQUIREMENTS.md#rq-037-plugincontractの検証済み組合せを構成bomとして再利用できること)）**：mainに存在するRQ-036は**全容Model Map／標準・カスタムFlow**の要求であり、Plugin構成BOMではない。BOMは独立した**RQ-037**として既存[#3267](https://github.com/hat47x/tei-platform/issues/3267)へ帰属させる。Primary PAS-06、PAS-02/04/01/05補助、Product GapはDesign Only。BOM Draft #3297は未統合で旧RQ-036を重複定義しており、mainに取り込む場合は要求番号・PAS・Gap Audit・Design Indexとgo build差分を現基線へ再調整することを**Hard stop**とする。図表RQ-036をBOM用へ改名・上書きしない。

**BOMのS2→S3へ渡す二つの未達**：main統合済み[#3381](https://github.com/hat47x/tei-platform/pull/3381)はCompositionを選択したProvider Bの局所Go実呼出とCLI Helper/Studio Host API二経路の限定証拠を持つが、通常`tei inspect` Dispatchと実Wails Bridgeの製品S2ではない。加えて、[#3490](https://github.com/hat47x/tei-platform/pull/3490)のCapability Rule鮮度、[#3477](https://github.com/hat47x/tei-platform/pull/3477)のConsumer Source-only、[#3479](https://github.com/hat47x/tei-platform/pull/3479)のImported Provider変更検査は、各Producerの歴史的証拠を現在へ再使用する際の必要条件に過ぎず、実Version 2×2の4 Attemptを検証したことにはならない。最初はS2一機能で**実選択B→通常CLI実Call B／実Studio Bridge実Call B**と誤Aへのfallback拒否を検証し、S3では実Build/Source closure/Toolchain/Target/Independent Expected/Attemptを対象ごとに照合する。主な受入負例は[既存PAS-06のRQ-037節](UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md)、詳細の唯一の実装Ownerは#3267/#1251/#1180/#2514。

**RQ-037 Verified Composition BOMの追跡修復（2026-11-09）**：製品RQ-036の正本は**全容Model Map／Navigation**である。PAS/Gap Auditに後から混入した「RQ-036=BOM」を修正し、BOMを**RQ-037、主PAS-06、Gap Tier=Design Only**へ区分する（[製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md)、[#3267](https://github.com/hat47x/tei-platform/issues/3267)）。この整理は旧BOM Draft #3297をそのままmainへ統合することではない。限定Go Referenceは[#3541](https://github.com/hat47x/tei-platform/pull/3541)でexact-head Repo Health/依存確認が成功したが、最新mainとの差分を再調整する必要があり`qualified:false`である。

**次の実CallとBuildの受入境界**：通常`tei inspect`と実Wails Studio Hostへの配線は[Draft #3540](https://github.com/hat47x/tei-platform/pull/3540)の**S2-reference**として実装候補があるが、固定コンパイル済みInspector ProviderのHost側Bindingや実行ファイルdigestは、別Providerへ切替可能な製品S2・実Link Closure・真正なConformanceの証拠ではない。[main #3583](https://github.com/hat47x/tei-platform/pull/3583)による既存実行入口の再照合ではC-1 `routing.Router`は配備Command、C-2 `servicehost`はService Requestと異なるため、両者を無条件に相互排他的なResolver候補としない。外部Hostへ委譲するC-4も許容し、実行Receiptの未取得は`delegated/not-observed`のまま留める。S2の実Provider選択／実Consumer受領をまず確定し、BOM 2×2の4ケースで各Target/Artifact/Independent Expected/Attemptを実照合する前にS3/Qualifiedへ上げない。

**テスト生成のP0実証との接合（2026-11-08更新）**： [Draft #3365](https://github.com/hat47x/tei-platform/pull/3365)のHEAD `e8745c6c`は[専用dogfood CI #42465435223](https://github.com/hat47x/tei-platform/actions/runs/42465435223)、全体Health、依存確認がsuccess。既存13件の実CLI read-only応答、42件の限定Context対照、6実Pinned Scenario Step、**9つのtest-only受領対照**を同一Runで検証した。独立したStep→Operation期待値が、ExpectedとReceiptを同時に誤変更するTT-154を拒否する。これは`test-ability`の**局所正証人**であり、正式Function実Invoke・Data Context/実DB Read・外部完全受領や盲検T1-Bの証拠ではない。詳細Expected/Caseは[#2514](https://github.com/hat47x/tei-platform/issues/2514)、Scenarioは[#1180](https://github.com/hat47x/tei-platform/issues/1180)、受入票は[既存テスト基本設計§35](https://github.com/hat47x/tei-platform/pull/3223)を参照し、このMasterには複製しない。[main #3502](https://github.com/hat47x/tei-platform/pull/3502)のTrait/Provider二Consumer設計や、[Draft #3505](https://github.com/hat47x/tei-platform/pull/3505)の全体Map/RQ-036候補も、実Entry/Rowset・現在Grant・全Domainの不存在証明を代替しない。Mapを表示する場合は**観測済・未取得・部分取得・非公開・未実行**の状態を対象Scopeごとに明示し、表示カスタマイズで実Coverageや権限の判定を変更しない。Source-only/Manifest-only失効の[PR #3479](https://github.com/hat47x/tei-platform/pull/3479)は書式修正後HEADのCIがsuccessだが、外部Providerの原文をConsumerが実消費した同一Attempt/Source閉包は依然別資格である。

**一改訂の引渡し票**には、少なくとも対象Subject・使用目的・Source/外部参照の実Revision・World/Source Indexの比較範囲・Requiredの取得/未取得・Meaning受理状態・選択したContract/Providerとactual Invoke・Target/Artifact・Attempt・Consumer受領・独立Expected・現時点の権限/Effectを、**観測できた列だけ**記録する。これは[#1163](https://github.com/hat47x/tei-platform/issues/1163)の証拠と[#3262](https://github.com/hat47x/tei-platform/issues/3262)の設計Context等を束ねる表示・検収の観点であり、第二の台帳や共通Schemaではない。
**一改訂の引渡し票**には、少なくとも対象Subject・使用目的・Consumer側Source/外部Import Providerの**RawとManifestそれぞれの観測Revision（未取得も明示）**・World/Source Indexの比較範囲・Capability選択の判定規則指紋・Requiredの取得/未取得・Meaning受理状態・選択したContract/Providerとactual Invoke・Target/Artifact・Attempt・Consumer受領・独立Expected・現時点の権限/Effectを、**観測できた列だけ**記録する。これは[#1163](https://github.com/hat47x/tei-platform/issues/1163)の証拠と[#3262](https://github.com/hat47x/tei-platform/issues/3262)の設計Context等を束ねる表示・検収の観点であり、第二の台帳や共通Schemaではない。

**2026-11-08の判定補正**：同じ`sales.order`に対し、①Consumer Source Indexだけ変わる改訂（main #3477）、②Import Provider Raw/Manifestだけ変わりConsumer World・Source Indexが変わらない改訂（Draft #3479）、③Capability選択ルールだけ変わる改訂（main #3350、Draft #3379の旧Evidence拒否）の三つは、**同一の失効イベントではない**。観測した入力閉包・判定規則・実Targetに応じて必要な再検査を選ぶ。Raw変更で全生成物を常に失効させる／World同一なら何でも再利用する、の両極は採らない。G4には[main #3440](https://github.com/hat47x/tei-platform/pull/3440)のTEI CLI自身の限定stage0→1→2実測があるが、供給信頼・独立二試行・既定Source Closure・Plugin/BOM適合は引き続き未資格。外部データ基盤KON Dataspaceはこの証拠・Meaning/Host Authorityの第二正本にも、D1の必須Runtime依存にも置かない。

最初に判定すべき停止条件は、(a)実Loaderに届かない設計、(b)同一Workspaceの指紋では見えない外部Provider改訂、(c)図・CRUDから実Readに到達しない辺、(d)選択Providerと実呼出し先の不一致、(e)Tool正常終了だがConsumer受領欠落、(f)認証済み主体を業務権限へ誤写像、(g)未実行のBOM組合せをPASSとする誤判定、の七つとする。新たな抽象化より先に各Ownerの既存Scenario/独立Expectedで**一対の正常・負例**を実行し、最初に失敗した辺と次の必要なProbeを示す。古い確定Gate B/C、合成テスト、Draftの資料を現在の製品E2Eへ転用しない。

#### 共通アプリケーションAPIの採用と既存Harness比較の入口（2026-11-08、ADR-0048／新Host不要）

**採用済みの設計判断**：[ADR-0048](adr/0048-shared-application-api-and-transport-adapters.md)により、CLI・MCP・HTTP/API・Web・Desktopから原則として**同じ型付きアプリケーション操作**を利用する。これはHTTPサーバーを常駐させる判断ではなく、localはプロセス内、remoteは明示した接続先で呼ぶ。開発者向けCLI、Editor/Studio、外部Harnessは同じ意味の操作を使うが、HostのWorkspace解決・開示・認可・副作用をこのAPIの内部に暗黙統合しない。現段階で全入口の同値性が実装・検証されたという意味ではない。

既存[#3419](https://github.com/hat47x/tei-platform/issues/3419)／[#3420](https://github.com/hat47x/tei-platform/issues/3420)のCAPI-1/2では、最初の4操作を **catalog / inspect / trace / validate** に限定する。特に`validate`で、呼出しの正常終了、意味Validationの合否、warning、Source root不在、missing/unknown、`failure.code`とsource範囲を型付きで区別し、CLI v0 wireは互換投影で保持する。かつて存在したCLI/MCPの`validate`成功・失敗の差は[main #3458](https://github.com/hat47x/tei-platform/pull/3458)で**正常・読込失敗・重複意味宣言の限定ケースについて実CLIと実MCP Adapterの投影一致**が確認された。残るcatalog/inspect/traceの完全同値、未知欄/null/上限、実Host認可・外部Harnessの配送は未資格のまま残す。CAPI-3[#3421](https://github.com/hat47x/tei-platform/issues/3421)のHTTP/StudioとCAPI-4[#3422](https://github.com/hat47x/tei-platform/issues/3422)のProposal／受理／実行はその後の別資格とし、read-onlyだけで現在Grant/Effectを許可しない。Plugin側[#951](https://github.com/hat47x/tei-platform/issues/951)／[#1251](https://github.com/hat47x/tei-platform/issues/1251)の選択→実Handler呼出しとも一つに潰さない。
既存[#3419](https://github.com/hat47x/tei-platform/issues/3419)／[#3420](https://github.com/hat47x/tei-platform/issues/3420)のCAPI-1/2では、最初の4操作を **catalog / inspect / trace / validate** に限定する。特に`validate`で、呼出しの正常終了、意味Validationの合否、warning、Source root不在、missing/unknown、`failure.code`とsource範囲を型付きで区別し、CLI v0 wireは互換投影で保持する。MCP参照Adapterはmain [#3403](https://github.com/hat47x/tei-platform/pull/3403)で4読取操作の限定値一致を観測し、その後main [#3458](https://github.com/hat47x/tei-platform/pull/3458)で**validateのCLI/MCP応答の限定的な形・診断・成否の差**を修正・試験済み。ただし全4操作の完全Body同値、外部Tool受領終端・実Host Grant・Plugin実Invokeまでは未資格。CAPI-3[#3421](https://github.com/hat47x/tei-platform/issues/3421)のHTTP/StudioとCAPI-4[#3422](https://github.com/hat47x/tei-platform/issues/3422)のProposal／受理／実行はその後の別資格とし、read-onlyだけで現在Grant/Effectを許可しない。Plugin側[#951](https://github.com/hat47x/tei-platform/issues/951)／[#1251](https://github.com/hat47x/tei-platform/issues/1251)の選択→実Handler呼出しとも一つに潰さない。

**独自AI Harnessは当面、実装決定ではなく追加価値の比較対象**とする。判別したい問いは「TEIの製品操作・Plugin・Skillで提供できない、再現可能な価値が独自Harnessの追加責務にあるか」である。固定Fixture/同一Source・Task・モデル/予算・Tool権限・開示条件を揃え、次を別々に評価する。

| 対照 | 実行する範囲 | 対象とする価値・比較不可の範囲 |
| --- | --- | --- |
| **B0** 決定的な共通API/CLI直接操作 | 4 read-only操作の取得、Source/診断/版/Requiredの確認 | データ取得・判別の基線。未知Ruleの自律生成は対象外であり、その非生成を「Harness品質不足」と採点しない |
| **B1** 既存外部Harness＋同じCLI/Tool（Skillなし／あり） | 独立Sessionで同じD（設計入力）・T（Tool受領）を検証し、G（未知Ruleのテスト候補生成）は別に盲検採点 | 既存Skillの純増分、Tool呼出回数、Token、待ち時間、コスト、誤肯定／誤不存在、不要操作を測定。単なるSkillロードやTool Exit0を増分成果にしない |
| **B2** 既存Harnessの薄いTool/Plugin Adapter（必要時だけ） | B1で反復して確認された一つの不足（例：型付き失敗/受領境界）が共通API＋薄いAdapterで解決するか対照 | Adapterで足りれば独自Agent Runtimeは作らない。Model/Providerを追加した改善と専用Harnessによる改善を混ぜない |

既存[DSH #2509](https://github.com/hat47x/tei-platform/issues/2509)のH2/H3では実Tool読取とSkill本文の限定一致を観測しているが、`model catalog`の各8,192-byte切断、56 Tool Request対55 Result、stderr付着、再利用Sessionの混入により**独立比較適格は0/4**。その過去4件を再採点して価値証明に転用せず、[#2515](https://github.com/hat47x/tei-platform/issues/2515)が持つ実OS/Network/Log送出と、[#2514](https://github.com/hat47x/tei-platform/issues/2514)が持つ独立Expected／Mutantを満たす**新しい独立4 Session**を優先する。Dを取得できてもTが途中切断なら、Gの成否と推定ROIを`not-qualified`にする。

最初の比較Taskは、P0-Dの一Function・3 Required RuleのContextと、T1-Bの**未提示Rule由来の新規負例を1件選択する能力**を別々の問いに分割する。Harnessに独立ExpectedやBlind Mutantを渡さず、旧Source改訂、欠落Relation、0件/partial Rowset、Selected Provider Bと実Invocation Aの取り違えを既存Scenarioへ委譲する。結果票には成否だけでなく、対象Run/Sourceのbyte・契約・actual Tool/Provider・応答終端・独立採点、実隔離と送出、使用Token/時間/費用、再試行を残し、任意の過大主張を拒否する。Domainを一つ追加しても差が残るか確認し、用途特化が必要なら汎用Harnessではなく**Domain Plugin/Skill/Policy Adapter**で足りる可能性を先に評価する。

**採否の停止線**：B1で既存機構の不足が独立に再現され、B2でも解消しない明確な機能差があり、少なくとも異なる2種類のTask/Domainで正確性・再現性・費用/運用負担・実効隔離の改善を確認するまでは、独自共通Harnessや新常駐Agent RuntimeをTEIの必須配備に追加しない。比較用の局所PoCは継続できるが、新しい証拠台帳・Owner・Test SSOTは作らず、#2509/#2514/#2515/#3262の既存経路へ結果を記録する。**今回の追加は計画のみ**。共通API実運用、CLI/MCP/Studioの完全同値、外部Harnessの新比較、独自Harnessの効果、OwnerのMeaning受理はNOT RUN/未検証。



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

<!-- tei-design-context-test-ability-t0-5-20261107 -->
### 2026-11-07 Issue→設計→テスト能力：Main実証とMVPへの統合（#3262／#2514）

**実装／実証済みの小さい縦切り**：[#3264](https://github.com/hat47x/tei-platform/pull/3264)は既存TEI CLI 9 read-only操作で一SubjectのSource/Inventory/Loader/Versionを限定検証。[#3266](https://github.com/hat47x/tei-platform/pull/3266)は既存#1180 Scenario SSOTから生成した**実別プロセス pinned CLI 6 Step**と、実CLI拒否4件／独立期待値検査2件をCIで確認。[#3268](https://github.com/hat47x/tei-platform/pull/3268)はSource/CLI/Scenario/Skill/Go依存変更を監視対象に追加してmain push再試験を実証。これらを全機能、正式意味、外部ProviderのE2Eとはしない。

**独立Requiredを満たしたかで次段の資格を決める**：[#3284](https://github.com/hat47x/tei-platform/pull/3284)は実9 CLI Replyを読む**テスト専用Consumer**をmainへ統合し、Source未読込、Subject欠落、Fingerprint版混合、Tool Reply欠落、Schema変更、Source root欠損を安全に拒否する6合成対照＋外部Provider/Harness/Grant未観測の資格保留1対照を検証。正常でも**`boundedReference=partial`／`testExecutionAdmission=exploration-only`／`qualifiedTestGeneration=false`**。Source Read≠Parsed/Accepted、同Fingerprint≠A→B→Aを含む時点・権限の完全保証、CLI受信≠H3外部Harnessへの完整なDelivery。局所`cmd/tei`1テストのStatement C0相当は10.1%、**C1未計測**。

| 段階 | Mainに取り込む/検証する成果 | 完了を禁じる不足条件と既存Owner |
| --- | --- | --- |
| **T0 実行・判別** | 固定公開Scenarioの実CLI pinned 6 Step、独立Propertyと意図的失敗の判別、正例・業務拒否の区別 | 実コードMutant・別Target・Owner/実DBの包括PASSにはしない。#1180/#2514 |
| **T0.5 Test Readiness** | 実CLIのSource/Subject/Version/必要Replyと独立Requiredを照合し、未Loaded/欠落/新旧混合を検出。必要な情報が未取得なら`blocked`又は`partial` | Product test-generation全資格、Plugin実Consumer Load、外部Harness受領／Grant／機密送出安全性は未実証。#3262/#2605/#2509 |
| **T1 AIによるCase発見** | 必須Data/Function/Business/Rule/Relationの出典・意味・未取得を同Scopeで回答し、最小指示で適用理由付きPattern／Candidate Scenarioを生成。独立Holdout非開示でMutant/false-qualified/未カバーを測定 | プロンプトからのExpected自己採点、Schema/件数・C0だけからのCoverage完了、AIがOwner Meaningを承認したふりをすることを拒否。#3223/#2514/#1180 |
| **T2 実製品横断** | Function→Query→実Provider Reader→Row/Field/Scope/Completeness、同ScenarioのCLI/Web/Studio分岐、API例外系のGraph＋表＋遷移検査。実Grant、Effect、Receipt、資源再読取を別証拠化 | 局所合成Query/HTTP静的Graph/旧Receiptや実PG別Reference成功から全経路PASSへ昇格しない。#2586/#3274/#1174/#2516 |
| **T3 自己記述・Self-Build** | TEI自体の要件/Rule/Graph/Scenario/Buildを標準Dataに記述・適切に投影、Stage1実TEI ProcessからStage2 Build/Artifactへの再入を検証 | `go build`、標準Plugin SB-3、GraphレンダリングだけでG4=L3合格としない。#2831/#1119 |

**順序と並列性**：T0.5はT1の**品質ゲート**だが、実Product ReaderのP0-R1（#2586）、Host Effect/Receipt、Plugin・Trait設計、DSL `*.tei.md`移行、Self-Build G4を全てT1の後に直列化する新Hard dependencyではない。別Ownerは並列に独立Fixtureを前進できるが、横断Claimを認定するときだけ`Required→Observed Source/実Consumer→Target Revision/Grant→Independent Expected→Claim Ceiling`の**同一試行**で照合する。T0.5のtest-only Python Readerは製品用Design Context Query・第2Parser/Verifier/権限正本に昇格させない。

**正本／次の引渡し**：[実装・証拠・詳細T0〜T3](design/ISSUE_TO_DESIGN_DOGFOOD_MVP_2026-11.md)を参照。T1では#3262のtyped read-only Contextの要件Closure／Unknownを先に実証し、#3223のPattern推薦→#1180採用Scenario→#2514の独立判別へ移す。外部H2/H3でのSkill効果はTAI #168／TEI #2509が固定Task・Skill実ロード・完全Reply・送出資格を揃えた**独立実験**を行う。Source/Rule/Plugin/Owner受理、C1・NFR、Function実Provider、外部H3 Test-Case生成は現時点で**NOT QUALIFIED**。
<!-- tei-data-shape-presentation-r1-20261107 -->
## 2026-11-07 標準データ構造・表現型・機能内Presentationの最初の受入（設計候補）

**目的と位置づけ**：X-TEAのテーブル間1:1／1:N関係、単票・明細やXF系機能の対応関係を、TEIでは**物理テーブルに依存しないData Shape**、**機能から独立して定義できるPresentation Type**、**特定Functionに束縛されたPresentation Instance**へ分解する。これは採用済みCore Schema／実装完了の宣言ではない。詳細型／Pattern適合の正本は[#2506](https://github.com/hat47x/tei-platform/issues/2506)・[#2507](https://github.com/hat47x/tei-platform/issues/2507)、標準CRUD／機能は[#1127](https://github.com/hat47x/tei-platform/issues/1127)、UI実現は[#997](https://github.com/hat47x/tei-platform/issues/997)に置き、本節は**製品横断の検収順序**だけを束ねる。既存のデータ様式設計・Read-first詳細は[旧横断Draft #3023](https://github.com/hat47x/tei-platform/pull/3023)を履歴・検討根拠として参照し、同じ大量の受入票を重複管理しない。

| 層 | 意味・責務 | これだけから推論しないこと |
| --- | --- | --- |
| **標準データ構造／Data Shape** | Scalar/Record/Collection、1:1・1:N等のRelation／Graph・Tree・時系列・空間／多次元・表、Identity・Cardinality・Grain・単位・NULL/欠損・出現ID・Temporal/ScopeのFacet。DB Table/列とのMappingは別 | テーブル・FK・ラベルや見た目からBusiness Meaning、正当な更新対象、必要全集合、現在Grantを決めない |
| **標準表現型／Presentation Type** | Panel/Form、Tabs、Text Input、単一/複数選択、On/Off、Slider、Table、Tree、Chart、Map等の型付き表示・入力方式。表示器／TargetやPluginへ実現を委譲 | UI種別からFunctionの業務責任、保存/承認Action、DB Bindingを生成しない。型の定義はFunctionに依存しないが、**実際の業務PresentationはFunctionの外に単独存在させない** |
| **機能内Presentation／Bound Instance** | Function Data Contextの対象／Relation path／Field／Grain／Selection／必要入力、Projection、Presentation Type Binding、Action Intent、状態/Revision・Consumerを**明示**して実際の表示を構成 | 表示可能≠変更可能、Operation候補≠現在Dispatch可、Save click≠Commit・Receipt・保存後対象データ確認 |
| **Provider/Host/Consumer** | Query/Target/Realizationの選択、実Provider Rowset/Field/Scope/Cohort、Host Grant/Effect/Receipt、実Consumerでの限定Projectionを各担当契約で連結 | Alias名やOperation ID、Source/Receiptの指紋一致から現在の実データ同一性や開示許可を認定しない |

**参照実装で観測された不足**：[main統合 #3145](https://github.com/hat47x/tei-platform/pull/3145)は固定Workspaceと見積候補を比較し、Model Revision・World Fingerprint・`function-definition`のPattern（候補側6件）は変わる一方、`tei operations`は**8件／gaps 0／conflicts 0／declared 0、全8件がderived**で不変と実測。つまり**既存OperationSurfaceに宣言Pattern/Business Actionを取り込むW3経路は、この限定観測では成立していない**。これを「全機能／全Actionを構成できる」証拠にしない。逆に、PatternをOperationへ無条件に複製する設計を決めたわけでもない。最初の関連は`function-definition`/宣言Sourceを通じて照合し、Action Purpose、KindとEffect、粒度/Context、Authorityの対応が揃うまでは`unresolved`として保持（#2507/#1127）。既存`surface.go`のderived Operationと実Service Contract/Commandの所有関係は別。

### 同一Fixtureでの最小受入順（試験IDの第二正本を作らない）

| 順位・Owner | 一つの正例と、先に固定する負例 | 成功の上限／停止線 |
| --- | --- | --- |
| **R0→R1／#2586・#2506・#2514** | 受注`Order1001`と明細2件（または組織`Case C1→Unit D1`）を独立Expectedに固定。Function Context→明示Query→Alias/実Provider→実Row/Field/Scope/Cohortの**同一Attempt**。別Tenant同番号、未取得、部分集合、NULL、完全な0件、Join増幅を対照 | まず**一Function・一Query・一実Provider**。宣言のみ、`readByLoader`、`UNBOUND_DATASOURCE`、型付きQuery候補、`tei dbcheck`、合成Go PASSではR1実Read合格でない。R1v二SQL・R1p Page継続は後続 |
| **P1／#1127・#2507・#997** | **同じFunction Context／Selection／業務ID**をPanel/FormとTableの二Presentationへ投影。列/Role/Unit/欠損/出現ID・Scopeを比較し、Renderer変更でもMeaning・Actionの正本は一つに保つ。Table↔Chartの集計／Tree↔GraphのPath多重性は既存FP-01/02へ | 2表示の局所投影が一致してもFunctionが実Read/書込可能とは言わない。集約の書戻しやTree node dragから更新対象を推論しない。Native Canvas等の例外はUI Kit/Targetが所有 |
| **P2／#2500・#2598・#1174** | Form編集／Tableセル編集→`ViewChange→型付きIntent→既存Action`。同一対象ID/Revision、必須Rule、現在Host Grant、選択Provider実Target、取消/再送・部分Effectの負例を対照 | read-only・適用候補・実Dispatch・実Effect/Receipt・Owner意味受理を分離。Widgetのeditable状態や`committed:true`を許可/保存の証拠にしない |
| **M／#3030・#2823・#988** | 旧`.tei`の固定R1/Presentation Fixtureと、新`*.tei.md`の同じ独立Expectedを比較。Required Root→Managed→`readByLoader`→Parsed Occurrence/Source Span→実Builder/Function消費を分離 | [#3073/#3078/#3081](https://github.com/hat47x/tei-platform/pull/3078)はRoot診断・Loader読込印・宣言0件Warningまで、[#3031](https://github.com/hat47x/tei-platform/pull/3031)は未統合の未束縛拒否Guard。新Markdown実Parserや移行後R1 PASSは未証明 |

**保存後の確認は最低二つに分ける**：[main #3124](https://github.com/hat47x/tei-platform/pull/3124)の`persistence`（`not-applicable/not-observed/observed`）と[main #3127](https://github.com/hat47x/tei-platform/pull/3127)の`effectcommit.Runner.Lookup`による独立**Receipt再照会**は、記録されたOperation ID・Payloadの照合であり、**業務対象Recordの現在値を独立取得した証拠ではない**。`persistence:observed`を出しても、Consumerへ「受注保存後の内容を確認済み」と説明するには**別のProvider Read／対象Identity/Scope/Version/Fieldの独立Expected**を要する。さらに`AppliedUnobserved`と`Differences=[]`の非同一性（main #3068/#3104）を維持する。

**優先順位・資格**：R0/R1は旧`.tei`で先行でき、Markdown ReaderやGUI完成はP0のブロッカーにしない。P1はR1と同一のMeaning/Context/Expectedを再利用し、表現を切替えても各Consumerの許容Scopeを変えない。P2とMはそれぞれ別Claimとして後続検証。構造・型・Role/Patternの提案と、機能内Presentationの業務意味受理はOwner判断。**今回の更新は既存Source/PR試験記録に基づく計画であり、P1/P2/Mの実製品E2E・実Provider R1・実DB値の独立Read・Owner正式受理はNOT RUN**。新Core UI種別、万能Presentation Engine、第二Parser/Host/Reader、影のAction/Authority正本を先に増やさない。

## 2026-11-07 標準データ構造・標準表現型と合成写像の統合検証（採用前）

X-TEAのDB 1:1／1:N→XF単票・明細という対応を、**物理テーブルと無関係な標準データ構造**、**Functionから独立した標準表現型の定義**、**Function Data Contextに所有される表現インスタンス**、**両側の構造間Mapping**へ一般化する検証を進めます。Container／Table／Tree／Chart／文書／音声等が同じ元Dataを異なる形に見せる際、表示能力・意味の保持・同期・編集候補・現在のHost Grant・Provider Effectを独立に評価します。

### 成果を一本の現行main起点の候補計画へまとめる

- [標準表現型と媒体時間軸](design/STANDARD_REPRESENTATION_TOPOLOGY_AND_MEDIA_AXES_2026-11.md)：Data/Representationの構造、Function-owned Instance、元データ時刻／取得時刻／Playback時間／文書位置、軸保持・変換・生成を分ける。旧[Draft #3192](https://github.com/hat47x/tei-platform/pull/3192)の成熟した設計記録を再利用。
- [複数表現の選択・状態・同期](design/REPRESENTATION_CROSSVIEW_COORDINATION_2026-11.md)：Subject/Result/Path/View occurrence、selection epoch、Presentation-only変更と業務Actionを分ける。旧[Draft #3222](https://github.com/hat47x/tei-platform/pull/3222)の成熟した設計記録を再利用。
- [合成写像の適格性・非単射・否定依存・開示](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)：Source→表示→他Consumerの各段でloss・Scope・unknown・Permissionを検収。局所表示成功から、合成後の完全性や逆編集を推論しない。SUI Tree/Island→SHO Outline、D1/D2 partial、空結果の否定依存、Research/Full Autoの操作モード、全出力面のDisclosureを実例とする。

**受入順（新Gateではない）**：P0 HeadlessでCX-01〜03/08の非単射・集計・空集合の判別 → P1 CX-04〜06/10のConsumer非干渉・操作Mode・epoch/版 → P2 SHO/SUI/Mediaを異質ConsumerとしてCX-07/09へ接続 → P3 #2586の実Function→Query→Provider Readを経て、編集が必要な場合のみ#2500→#2598→#1174のIntent/実Request/Receipt/独立Readへ接続。旧PRのRP/MVと新CXは[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)の既存独立Expected/Scenarioへ**差分引継ぎ**し、第二のテスト正本を設けません。

**担当を変更しない**：構造のFacet=#2495、Loss/Projection=#2497、Context/Result Slice=#2506、Pattern=#2507、変更影響=#2582、Renderer=#997、用途別開示=#2516、実Read=#2586、Writer/原文受理=SHO/TEI各Owner。今後のUI生成・Operation開発を早めるための設計であり、Core型列挙・共通Plugin Registry/Runtime/Host・第二Action/権限Engine・新Gateは追加しません。

**証拠の上限**：これは現行mainを起点とした設計資料と計画の整備であり、CX-01〜10の新試験、実Provider/Media/Renderer/Host接合、SHO/SUI実受領、Owner意味受理は**NOT RUN**。局所PRのGo/CI実績や見かけの同一表示を、実Product E2E／現在の適合資格へ自動昇格させません。並行変更がある場合は統合前に同一HEAD/内容差分を再照合します。

### 製品実Readから二表現へ渡す初回受入（2026-11-07追補）

上記の幅広いRP/MV/CX圧力ケースは維持するが、**最初にProduct資格を得る一本の実証**は[#2586 HR-01/02](https://github.com/hat47x/tei-platform/issues/2586)が所有する`order_entry.recent_orders` Headerの**Function→実選択Query/SQL・args→Runtime DataSource alias/物理Provider→actual Rowset/Scope/Field Presence→Function Data Context**に固定する。その後、同じContextからForm/Tableの**二つのread-only表現**を投影し、Identity・Grain・NULL/未取得・Page/部分集合・選択epoch・開示の一致と、情報損失を#2506/#2507/#2497で検収する。最初に欠ける接続辺はunresolvedとしてSource/Ownerとともに記録し、Headlessの参照・合成成功を実Product Read資格にしない。

現main統合[#3195](https://github.com/hat47x/tei-platform/pull/3195)/[#3204](https://github.com/hat47x/tei-platform/pull/3204)のPattern↔Operation突き合わせ参照実装は8 Operation／5 Patternの局所結果があるが、**正式なFunction/Action dispatchとSQL Readerではない**。参照用実PG Readの[Draft #3238](https://github.com/hat47x/tei-platform/pull/3238)もProduct HR-01/02とは別。反対に、[main統合#3155](https://github.com/hat47x/tei-platform/pull/3155)は資源独立Readによる`resource-observed`を既に持つため「永続化の観測機構が全TEIで未実装」とは記載しない。**Resource Readの局所成立≠受注FunctionのHeader/明細完全Read≠業務Outcome**とする。

順序はR1実Read→R2同一Contextの二read-only表現→R3必要な場合のみViewChange/typed Intent/現在Grant→R4 Effect/Receipt/独立Read→R5 SHO/SUI/Graph/Media異質Consumer。独立Expectedと否定依存／同件数異集合／LIMIT 1／Non-output Predicateの反例は既存#2514/#1180を再利用。詳細は[合成写像の適格性 §8](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)。新しい実行結果の主張ではなく、実Provider、Product二表現、Owner受理は**NOT RUN**。

### Workspace帰属・read-onlyの実効果・ER/CRUD図の資格（2026-11-07更新）

標準表現型の初回実証をR1 Product Header Read→R2同一ContextのForm/Table read-onlyとする順序は維持する。追加で三つの**適用資格**を検査する。

- **Workspace/Source帰属**：[#3149](https://github.com/hat47x/tei-platform/issues/3149)/[#3181](https://github.com/hat47x/tei-platform/issues/3181)のCLIドッグフーディングで、別Workspaceの同一 `sales.order` Subjectに対しmanifest/Workspace identityが応答に欠落する問題が観測された。Context Instance、Manifest identity、Source revision、実Provider Scope、Read epochを明示し、同名・同内容を同一実対象へ無音結合しない。CLI Contractの変更は#3181/#3242/#2824がOwner。第二Workspace Registryは作らない。
- **副作用の種類**：[#3182](https://github.com/hat47x/tei-platform/issues/3182)で、`view`のread-only広告と明示`-execute`＋ログ/状態ファイル出力が別であることが実測された。Business read-only、環境ファイル・送出効果、現在Host Grant/実Requestを独立評価する。Form/Tableを読み取り専用で投影できても`zero OS effects`、自動Action dispatch、業務完了を宣言しない。
- **異質な図表Consumer**：[#3250](https://github.com/hat47x/tei-platform/issues/3250)のSubsystem候補/ER/CRUD/依存図は、標準構造をGraph/Tableへ展開する追加圧力例として利用する。論理Subsystem/Owner候補・表示レイアウト・直接CRUD・間接Effect・実Provider Effect・物理配置を区別し、折畳み/図の移動をSource Rewriteとしない。未知・部分・非開示の越境Edgeを「越境0件」と推測しない。専用Optimizerと受理権限は#3250/#267のままとする。

**前進した実証**：[Draft #3238](https://github.com/hat47x/tei-platform/pull/3238)固定HEADの[#2586実行記録](https://github.com/hat47x/tei-platform/issues/2586)で、隔離PG 16.15→物理Read→局所Completion、Top 2/Top 1の区別が限定CI成功。これはReferenceの一部達成で、**Product Function→Query/Provider→Context→二表現のE2E合格ではない**。前節の「#3238 PGの結果は未確認」は作成時点の履歴とし、本節の限定成功を現在資格として優先する。

同じ[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)のIndependent Expected/Scenarioへ、[合成写像検収§9](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)のWX-01〜06を一因子差分として引き継ぐ。新Go/実Product DB→二表現・跨Workspace UI・ER Editorの試験とOwner Meaning Acceptanceは**NOT RUN**。複数並行PRのMaster Plan編集は統合前に内容比較し、同一受入を二度追記しない。

### 実SQL Sourceと隔離PG Readの対応証拠（2026-11-07：初回実証の停止線）

[合成写像検収§10](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)に、現mainの**実Sourceファイル比較**から得られた不足条件を追加した。Product `reference/workspace/applications/order-entry/queries/recent_orders.tei` は `datasource sales` と別ファイル `recent_orders.sql` を参照し、そのSQLは **`FROM public.orders`、`WHERE ordered_at >= :from`、`ORDER BY ordered_at DESC`**。一方[Draft #3238](https://github.com/hat47x/tei-platform/pull/3238)の実PG Referenceは専用の `public.tei_web_scoped_orders_reference` とTenant限定の実行により、`orderId`ソート・TOPを比較する。Logical Query IDや`sales_core`の一致、実PGのGreenだけで、**同じSQL byte/母集団/Time Window/Physical Provider/現在認可**とは断定しない。

初回HR-01/02では、**Function/Pattern実選択→Query Source + SQL fileの実消費bytes→`:from`のBinding/Time Zone→runtime datasource sales/physical Provider/実行statement→受領Rowset・Scope・Read epoch→Context/Result Slice**を一Attemptとして固定する。原Queryの`ordered_at`順とReferenceの`orderId`順が異なるなら、先頭ページの同数・同一IDだけで表示同等・全集合Coverageを宣言しない。QueryのSQL-only変更は別Source Contributionとして失効。Contextが同じでも複数Readerの**joint snapshot**は自動成立しない。

既存#2586（Product E2 Read）、#2387/#2862（SQL Source/Variant）、#2611（Provider binding）、#2506/#2507（Context→表現）、#2514/#1180（独立Expected/Scenario）の担当へ、QS-01〜06を追加の一因子差分として引き継ぐ。これは**出典に基づく設計更新のみ**であり、QSの新Go/実Provider/二View E2、Owner意味受理はNOT RUN。新SQL Executor/Reader/Registry/Gateは作らない。

**実測で更新した表示・操作資格**：[SUI Draft #3396](https://github.com/hat47x/sui-sensemaking/pull/3396)の固定HEAD CIでは実Product `buildContextProjection` の非単射（元Source cue／Island所属が異なっても公開投影が同じ）と正の変更対照を**3/3 PASS**で観測した。[main統合済みTEI #3253](https://github.com/hat47x/tei-platform/pull/3253)はこれを研究基線に帰属する。**公開投影が正しく同じことは元Source Identityの一致・逆編集の一意性を証明しない**。一方[main統合済み#3254](https://github.com/hat47x/tei-platform/pull/3254)の操作面drift実測ではモデル属性追加でWorld/Model fingerprintが変わっても5 Operationの`basis`が不変。現在の実Request資格を旧`basis`一致から導かず、必要依存・Source/Version/Selection・実Grantを確認する。影響外と証明できるread-only表示は局所維持を認め、すべての変更で無条件に再取得・拒否する第二Gateを設けない。

**開発上の接続**：[合成写像検収§11](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)にSUIの局所PASS、TEIの操作面照合未接続、Product E2未実証を別票で保存。#2497/#2507/#2582/#2598・独立Expected #2514/#1180へ引き渡す。並行しているドメイン意味駆動のテスト生成案[#3258](https://github.com/hat47x/tei-platform/pull/3258)〜[#3261](https://github.com/hat47x/tei-platform/pull/3261)はOwner #1131/#2514との採否・重複統合を待つ**別Draft**であり、#3236の第二データ生成器やScenario正本にはしない。TEI Product二View/実逆Patch/Hostの新試験・Owner受理はNOT RUN。

### 2026-11-07 三視点Diagram/CRUDと実Provider Readを並列P0で検収する（main成果との整合）

X-TEA Modeler由来の**Data/Function/Businessの3図とCRUD cross-reference**はTEIモデラーの基幹価値であり、実Providerが先に接続できることを前提にしない。[三視点受入 Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)（M-01〜10）を既存#1124/#1125/#1126/#2497/#1803へ振り分け、**P0-M**として受理済み限定Sourceから3図・CRUD MatrixをHeadless/read-onlyで導き、Source/Role/Relation/Occurrence/Revision・宣言/派生候補/実観測Effectの根拠を同時に返す。Graphの辺・多重性/1:Nの関係Roleを保存し、行列の空欄、未取得/非開示/partialを「CUDなし」の証拠にしない。図座標・Mermaid風文書の見た目を元Source更新としない。可逆な編集はTyped ProposalとOwner Acceptanceを経る。SHOの独立図解は既存SHO #5/#6/#19 Consumerで検収する。

従来の **P0-R**（#2586実Product Function→Query/SQL/Provider→Result Slice→同一ContextのForm/Table）も維持する。**P0-C**では三図/CRUDの`sales.order`と実Read結果を型付きSubject/Source Revision/Scope/Read Attemptで照合するが、Query実行がない`R`宣言を実Reader成功と呼ばず、同IDでもWorkspace/Providerが違えば接合しない。M/R両レーンは共通の出典・欠損診断を利用し、製品Mainの新Schemaや第二Reader/Runnerは立てない。

前段の設計完全性は[#3262](https://github.com/hat47x/tei-platform/issues/3262)が所有。**main統合済み[#3264](https://github.com/hat47x/tei-platform/pull/3264)の実CLI9呼出し・Skill静的14・Required Source Loader確認**と**[#3266](https://github.com/hat47x/tei-platform/pull/3266)/[#3268](https://github.com/hat47x/tei-platform/pull/3268)の既存Scenario 6 step／実CLI負例4＋独立義務2／対象pathのCI再起動**は限定のT0基線であり、Design Contextの全Source Closureや三図/CRUD実生成、MCP/外部Harness完全受領、AIの新Case生成を実証しない。Goの局所Statement Coverage C0相当をC1や機能/NFR Coverageへ昇格させない。未カバーRuleから新Caseを作るT1は[Draft #3273](https://github.com/hat47x/tei-platform/pull/3273)と#1131/#2514/#1180が担当し、第二Oracleを作らない。

[合成写像検収§12](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)のDM-01〜08を既存独立Expectedへ引き渡す。**今回の新たな実P0-M Graph/CRUD出力・P0-R Product Provider接続・P0-C Crossview・Studio/SHO・Owner意味受理はNOT RUN**。mainと並行Draftの更新内容を失わないよう、旧ブランチのMaster Planを最新mainに基づき統合し、PRの再比較とCIは別に確認する。

### 三視点Fixtureを現行CLI/Graph/Design Contextの実装入口へ接続（2026-11-07）

[三視点計画Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)にはData Entity 9／Relation 7（同端点の別Role二辺）／Function 4／Business Action 3、独立に記述した`expected-v0.json`、12種のKnown Mutant候補がある。しかし**合成JSONはTEI Parser入力でなく、Input/ExpectedともAI草案**。整合性15項目の静的検査は独立Expectedの正当性やTEI製品Graph/CRUDの実生成の証拠ではない。

実mainの `internal/modelref/model.go` はScoped Ref/Relation Evidence、`tools/reference-er-table-crossview/projection.go` はread-only ER/物理表比較、`tools/reference-business-dfd/diagram_context.go` はHandoff候補DOT、`reference/tei-studio-graph/graph.ts` はnode/edge IDと多重辺・Node選択を保持する。ただし**実三視点Source→Loader/Role→Graph/CRUD Projection→元Relation/CellのSource復帰→Studio/SHO実Consumer**はまだ未接続。GraphでEdge select/Field slotのSource復帰は現在のnode-only Referenceと別資格。既存CLI discoveryに三視点diagram/CRUD製品Commandは広告されていない。

**新しい検証前段**：[main統合#3284](https://github.com/hat47x/tei-platform/pull/3284)・[#3296](https://github.com/hat47x/tei-platform/pull/3296)の既存`context_gate.py`は、9 read-only CLI Replyを使用し、不正/欠落Root・古いSource指紋・途中切断・JSON重複Member等の**計16の合成負例を有限CIで判別済み**（[#3296固定HEAD CI](https://github.com/hat47x/tei-platform/actions/runs/42371310577)）。しかしRequiredの中心は`semantic-data:sales.order`で、Function/Business/Rule/Relationすべての閉包ではない。Diagram/CRUDの`complete`やテスト生成可へ昇格させず、現行Gateへ対象Requiredを拡張しても実Loader/Consumerが支持した範囲だけ資格化する。第二Packet/Runnerは作らない（重複PR #3293は未マージClose）。

最初は(1)合成FixtureのTEI Source/Loaderへの**表現可能部分**を特定、(2)型付きGraphの同端点多重辺・同名異IDの保持とSource復帰、(3)Function×Data CRUDで`declaredCrudComplete=false`を未知と表示、(4)Human/AI/Studio/SHOのConsumer別受領／編集の資格を検証。JOIN 2×3の6表示行≠実体6件。独立ExpectedとMutant評価は#2514/#1180を利用し、[#3236設計§13](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)のDGX-01〜04を順次検収する。**新製品Diagram/CRUD/GUI/Provider実行、Go/CLI/Owner受理は今回NOT RUN**。

## 組織意味と全容Model Mapの現行main接続（2026-11-09、設計・実行未受理）

<!-- org-crossview-currentmain-20261109 -->
[組織意味モデルの2026-11-09横断追補](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)で、組織所属／職位／案件担当／業務承認／Host認可／実引継ぎの**異なるRelation Identity**と時間・Owner・証拠を分離する。RQ-036が定めた全容Model Mapの要件は[main統合#3505](https://github.com/hat47x/tei-platform/pull/3505)、read-only型付きProjection候補は[Draft #3518](https://github.com/hat47x/tei-platform/pull/3518)、三視点/CRUDの実Relation照合は[Draft #3492](https://github.com/hat47x/tei-platform/pull/3492)に分担。現時点で組織Event/State/Hook、Graph GUI、SHO図の全経路実Productは未証明。Case C1のD1→D2担当変更は、Personの所属変更から暗黙に生成しない。古い承認候補／Grant、Replica Read、遅延・重複ACKを現行Effect/全件Handoffへ流用しない。

**実証優先の補正**：TEI全体の最初のP0-Rは、既存#2586 HJ-09の **order_entry.recent_orders**（Pattern→選択/実呼出Provider→SQL/Rows→Mapping→Function Context→Consumer）。組織Native C1→D1/D2（[Draft #3500](https://github.com/hat47x/tei-platform/pull/3500)のinstrumented Connector/E1f）は、その縦断契約の**次の分野別Pressure**にする。#3500の隔離PostgreSQLは既存確認でSKIP、Orgの実Provider/Trusted Scope/Consumer受領は未成立。P0-Mは同じSource/Scopeの型付きGraph・Function/Business・CRUDにread-only Source navigateと欠落/非開示を保持する比較を並行。P0-Cで二者の実受領と独立Expectedを同一Attemptへ統合。P1以降に#1174/#2499の現在Grant/Effect、#1125/#1444のHandoff、#1124のTyped編集・Acceptance/再Projectionを行う。

**所有先・重複回避**：組織Meaning=#2386、Pressure Corpus=#2510、Read=#2586/#2506、Model Map/三視点=#1124/#1125/#3492、Lifecycle/Trait=#3278/#3507、SubSystem分割は#3250、独立Expected/Scenario/Attempt=#2514/#1180/#1163。旧[Draft #3243](https://github.com/hat47x/tei-platform/pull/3243)は現mainより大幅に乖離しているため、旧MASTER_PLAN/README/組織Profileを丸ごと再投入しない。**本追補はdocs-only設計・作業順序**であり、新CI・実Provider/GUI/Host/Owner意味受理はNOT RUN。RQ-036全容MapとRQ-037 Verified Composition BOMの番号整合は別[Draft #3592](https://github.com/hat47x/tei-platform/pull/3592)へ委譲する。

### 組織責務切替の次段受入（2026-11-09、設計・独立Expected案）

<!-- org-cutover-tr-plan-20261109 -->
[組織意味設計のCutover追補](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)に、**所属更新≠Case handlingUnitの担当変更≠実行Grant切替≠相手方Handoff**を実効責務の四象限（旧側/新側がyes/no）として定義。単独責任を全組織へ強制せず、Ownerが共同責任・合議・移行中保留を選択可能にする。二重有効（単独責任時）／両者無効／現在性未照合／A→B→A旧Receipt復活を分離し、ORG-TR-01〜08を追加した。旧Draft #3243のORG-CF/MAPを第二Scenario化せず、#1180/#2514の既存独立Expectedへ差分接続する。

**他レーンの現在基線**：mainの #3540 は通常CLI/Studioの静的Inspector参照選択、#3635 はStudio参照Hookの executed/notEvaluated/skipped/refused を区別。どちらも組織実DB/現在Grantの資格ではない。旧#3496（closed/unmerged）のREA 6.7.0 T0実解析・Artifact監査はSource候補の限定証拠。T0-05は[Draft #4257](https://github.com/hat47x/tei-platform/pull/4257)で6.7.0の単独JSON／6.8.0の7行＋jsonl/v1終端を同じ公開合成JSの別2 Jobで実証（[Actions #42942865412](https://github.com/hat47x/tei-platform/actions/runs/42942865412)）。旧監査・候補投影コードはmainに無く、Owner意味受理/TEI Product Intakeは未成立。#3662のSubsystem固定Owner数理候補は現業務責任や物理Service分割でない。

**優先は据え置き**：全TEIのHJ-09一Function実Read（#2586）を先行し、組織C1→D1 read-onlyとRQ-036 Model Map／三視点の同一Source照合は次段、実Case切替・旧Grant/SoD・二件Handoffは#1174/#2499/#1125/#1444のP1。実行主体と認可Sourceの比較可能なrevisionが証明されない場合は保留し、HookのnotEvaluated/partialを許可やEffectに昇格しない。新Org Status Enum/Authority Engine/Workflow Host/Proof Store/Hard Gateを設けない。本追補はdocs-onlyでORG-TRの実Go/PG/Studio/外部Host/Owner受理は未実行。
## 2026-11-07 標準データ構造と標準表現型の対応（設計・検証候補）

X-TEAのテーブル1:1/1:N→XF単票/明細の対応を、**テーブル非依存の標準データ構造**と**機能から独立して定義する標準表現型**の二つのトポロジー間のMappingへ拡張して検証します。表現型の定義/Paletteは独立してよい一方、用途・対象を持つ具体的な表現インスタンスはFunctionに配置し、Function Data Context/Result Slice/Operationと明示的に結び付けます。データ側のEntity/Relation/Occurrenceと表現側のPanel/Tab/Table/Tree等の配置を同一視しません。

- **B・F・G領域の横断**：[#2495](https://github.com/hat47x/tei-platform/issues/2495)が構造Facet、[#2497](https://github.com/hat47x/tei-platform/issues/2497)が構造写像・情報損失・逆写像、[#2506](https://github.com/hat47x/tei-platform/issues/2506)/[#2507](https://github.com/hat47x/tei-platform/issues/2507)がFunction内Binding、[#997](https://github.com/hat47x/tei-platform/issues/997)がTarget別Presentationを担当します。第二のValidator/Renderer/Host/Scenario正本は作りません。
- **時間・媒体の追加検証**：元データの時刻、取得時刻、再生/生成時刻、文書の段落/ページ位置を分離。音声・動画・文章・ページ間では、軸を保持する写像、明示的な時刻変換、新たな再生時間軸の生成を区別します。朗読時間やreflowされたページ番号を元データの本質的な時間とみなしません。
- **順序と完了条件**：P0=分類比較、P1=Headless read-only mapping、P2=Function/Rendererへの配置、P3=逆編集/実Provider Effect、P4=SHO/SUI等の異質Consumerへの展開。先にRP-01〜08（JOIN fanout、DAG→Tree、数値→Slider、音声/動画同期、文書→TTS、Sonification、partial Tab、Function跨ぎ）の独立Expectedで誤実装を落とします。既存[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)へ接続し、新Gateは追加しません。

詳細は[設計・反証比較票](design/STANDARD_REPRESENTATION_TOPOLOGY_AND_MEDIA_AXES_2026-11.md)。**今回の追加は計画のみ**であり、RP-01〜08の実行、実Media生成/再生、Provider/Host接続、Owner意味受理、公開契約の採用を示しません。既存の判定段階A/B/Cの評価は変更しません。
## 2026-11-07 表現型間の選択・状態連携（検証候補）

標準データ構造と標準表現型を独立させ、Function内に表現Instanceを配置する[前段の設計Draft #3192](https://github.com/hat47x/tei-platform/pull/3192)に対し、**複数Viewの同期が業務対象・取得Scope・書込み資格を暗黙に共有する誤り**を検出する計画を追加します。前段PRは本節記載時点でDraft・未統合であり、正式採用済みではありません。

- **設計上の差分**：Data Subject／Relation occurrence、Query Result occurrence／grain、Representation occurrence（行・Graph/Tree経路・集計点）、Context Instance／selection epoch、操作Intentを区別。Panel/Tab/Canvasの配置・Focus・Zoom・SeekをDomain Mutationへ自動変換しない。異なるViewへの同期は用途・粒度・出現ID・Read Scope／Disclosure・現在の版に従い、exact／複数候補／aggregate関連／unknownを区別する。
- **既存責務を活用**：[#2506](https://github.com/hat47x/tei-platform/issues/2506) Context/Result Slice、[#2507](https://github.com/hat47x/tei-platform/issues/2507) Binding、[#2497](https://github.com/hat47x/tei-platform/issues/2497) Projection、[#2582](https://github.com/hat47x/tei-platform/issues/2582) Change/Refresh、[#2500](https://github.com/hat47x/tei-platform/issues/2500)/[#2598](https://github.com/hat47x/tei-platform/issues/2598) 更新Intent/Request、[#997](https://github.com/hat47x/tei-platform/issues/997) Rendererに分配し、第二のState Engine/Authority/Scenarioを作りません。
- **受入順**：C0 MV-01〜04（Table/Form、DAG→Tree、Aggregate、partial）Headless → C1 MV-05/06（Selection ABA、非開示の同期漏えい）→ C2 MV-07/08（SUI Canvas・SHO文書・朗読区間）異質Consumer → C3 既存Function→実Provider Read/Action/Receiptの縦切りへ接続。独立Expectedは[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)が所有します。

[表現型間連携・変更影響の設計比較票](design/REPRESENTATION_CROSSVIEW_COORDINATION_2026-11.md)を参照。**今回の成果は計画・比較票のみで、MV-01〜08、実Renderer/Provider、Owner受理はNOT RUN**。既存Gate A/B/CおよびProduct実装状況の判定には反映しません。
## ドメインモデル由来のテスト生成と開発者支援（設計・未実装）

TEIの三要素分析（Data / Function / Business）を、単なる入力データの生成にとどめず、値域・型・強属性の意味候補・業務Key/FD・Relation・業務Rule・手作業/承認を含むProcedure・広域のデータ整合性から**検証義務と正負テスト候補を導出する開発者支援**へ展開する。独立Expected、実Runner、実Provider、Source/Attempt Evidence、Owner受理の正本は増設しない。

最初は#1131の型付きDataset/Snapshot・候補生成、#1126/#2495の制約、#2514の独立Expected、#1180のScenario、#1163の実証Evidenceを利用し、**受理済みConstraintだけを決定論的に検査して正常例・境界値・一因子違反の反例を生成する**。次に#1125の業務手順・手作業/承認、#2506/#2586のContext/実Read、複数Tenant/Provider・変更/同時実行への横断検査を段階的に足す。AIは反例探索と変更影響分析を支援し、業務意味の採否や本番の副作用は実行しない。

**「強属性」の原典定義は未受理**であるため、業務Key・FD・必須属性と自動同一視しない。未取得、業務NULL、非選択SQL NULL、部分取得、異Snapshot、非公開条件の照合も区別する。詳細と16件の独立対照候補、P0〜P2の開発順、実行済み/未実行境界は [ドメインモデル由来のテスト生成・開発者支援計画](https://github.com/hat47x/tei-platform/issues/4134#issuecomment-6772836751) にまとめる（Draft、**生成機能は未実装・NOT RUN**）。

### 2026-11-08 標準表現型の残差：実Source入場と共通Consumerを最小検収へ接続

[合成写像の検収§20](design/COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)へ、mainで実証した限定Factと未接続Edgeを統合した。**第1〜13章と基本P0-M/R/Cは現mainに既に存在するため二重導入しない**。

- **Source**：main #3482は13実CLI Reply／42合成Consumer Controlsで`sales.order.submit`の宣言・Application・Rule/Source位置を限定確認したが、Product Function→SQL/物理Provider実Readを証明していない。main #3477は同Workspace Source-only改訂を検知。Import Provider原文だけの改訂は未マージ#3479が別の反例として扱う。Managed Markdown（#3346）は`readByLoader:false`で、物理FK/Index（後継Draft #3486、旧#3372は未マージClose）は業務意味の**Candidate**に留まる。
- **Consumer**：main #3403にread-only MCP参照Adapter4 Toolが存在し、ADR-0048 (#3439)はCLI/MCP/API/Web/Desktopの共通Operation方針を採用。#3458は`validate`のCLI/MCP正常・失敗の局所Parityを検証。残るTool、実Web/Desktop/外部H3/Hostの受領・効果・現在Grantまで同値とはしない。
- **最初の縦切り**：実TEI Source→read-only型付きGraph/Relation locator（P0-M）と、同一Product Function→実SQL/Provider→Data Context/Form/Table（P0-R）を並列実証し、同じSource/Scope/Revision/Read Cohortが観測できた部分だけP0-Cで結ぶ。Graph/CRUDの未取得はunknown、物理観測は未受理Candidate、参照CLI/MCPの値一致は現在の実Invoke/Effectではない。
- **担当と停止線**：#3263三視点、#2586実Read、#3262必要Source、#3479 Import鮮度、#3486 Workbench候補、#1158/#3420共通API、#2514/#1180独立Expectedを再利用。IA-01〜04は一因子の負例候補。新Source Registry/Parser/API/Host/Runnerを作らず、追加Product E2EとOwner意味受理は**NOT RUN**。
## 2026-11-07 組織表現と業務手順の非自動接続（#2386/#1125、受入提案）

[組織設計§84.6〜84.9](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)へ、組織の**Data/Relationトポロジー**、再利用可能な**標準表現型**、**Function内のPresentation/Contextインスタンス**、一階層の**業務DFD（業務間データ授受）**、内部**Business Procedure**、実**Workflow Instance/Host/Provider Effect**を分ける最小受入を接続した。画面Treeの部門後継線、案件担当Pickerの選択、DFDのデータ授受矢印、Procedureの引継ぎStep、実Case移管と人手の受領は、似た図形や同じC1/D1/D2でも同じ出来事ではない。

**具体的な検証は既存Ownerへ委譲**：#2506/#2507/#997は組織Tree/TableとCase Panel/PickerでFunction ContextとTarget・Scope/epochの保持を検査、#2500/#2598はViewChange→受理済みAction/実Requestの不一致を検査、#1125/#1444はDFD→Handoff/Procedure（人手/外部/機械）の**read-only対応**を検査、#1174/#1163は現在Grant/Target・Effect/Receipt/独立再読取を所有する。既存ORG-SU/SM・APP/TB・VE/RITの独立期待を再利用し、新Workflow/Policy Engine/専用Org Core型は設けない。

対象集合を使う業務完了判定では、#3238が扱う重複ID・TOP1・非出力Predicate、mainのCompletion APIへ渡すCaller提供Coverageの独立資格を§84.8で追加確認する。Receipt再照会だけで保存後資源の独立Readや人手引受を確定しない。

#3236の組合せ表現では、Data→Function View→SHO/AIの各写像が局所成功しても全体のRelation/Scope/開示を保証しない。#1129の**Data-attached Rule Attachment**はMembership発生やCaseの直接Relationへ明示Bindingし、read-time導出・validation・before-write・Tx内効果・post-commitを区別する。表示変化やSQL JOINの重複を発火・完了の証拠にしない（§84.7/84.9）。

Sourceの論理Workspace IDと物理root/realpathも別資格とし、Manifestの名前が同じでも実読込Sourceの位置・内容・現在許可範囲を検査する（組織設計§84.5.1、#2823/#2605/#2017）。

**製品E2の優先は不変**：main統合#2681のNative組織E1を起点に、[#2586](https://github.com/hat47x/tei-platform/issues/2586)で**一件のC1→D1を実Function→Query→Provider Read→Consumer**まで同じSource/Attemptで検証。旧[#3024](https://github.com/hat47x/tei-platform/pull/3024)と設計PR[#2389](https://github.com/hat47x/tei-platform/pull/2389)は**closed・unmerged**であり、その旧CI/未マージ草案から現main Product E2 PASSやOwner意味採用を主張しない。ここでの追補は**docs-only／独立期待・接続案**で、組織二Functionの実UI・実DB/Host・Procedure Instance、正式Owner受理はNOT RUN。

## 2026-11-07 組織トポロジー・単位系と標準表現型の横断受入（#2386/#2495/#2593、採用前）

[組織設計§85](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)と[組織トポロジー・数量・表現型合成Profile](https://github.com/hat47x/tei-platform/issues/2510#issuecomment-6772933215)に、**同じ組織の多軸Graph/多項Relation/時間・空間と、人数・FTE・予算・工数等のQuantity Kind／Unit/Basis／Rollup lawは直交するが互いに適合条件を持つ**という比較票を追加。組織を専用Tree Core型へ限定せず、#2495/#2544/#1705/#2593の一般Facetを組み合わせ、組織ProfileはRelation Role・用途別多重度・集計規範の不足分だけを与える。

独立Expectedの最小例は合成P1(A1 D1=0.6 FTE,A2 D2=0.4 FTE)、P2(A3 D1=1.0 FTE)。**Unique Person全体2人／Membership3件／FTE2.0**を混同しない。H→D1→CとH→D2→Cの二Pathは、Cの同一0.4 FTE Factを自動で0.8へ増やさない。現mainの`Membership.allocation std.decimal`はFTE Unit/分母・60と0.6の換算Ruleを宣言していないため、合成期待値を実Sourceの業務計算済み資格へ昇格しない。#2510/§16/39のTopologyケース、#2593 MC、#2497 CX、#2514の独立Expectedへ差分を引き渡し、第二テスト正本を作らない。

**表現型適合**はGraph、Tree、Matrix、Table、Chart、Timeline、Map、SHOの平易なMermaid風テキストを同じDataから構成し、用途/Scope/Unit/Path/時間/Disclosure・Lossと逆編集資格を別々に検証する。TEI表現インスタンスはFunction Data Contextに所属し、SHOは別Consumerとして再資格化。Readableな図の生成成功≠元Relationの再構成可能性／現在Grant／実編集Effect。標準の表現型と専門Profileを分離し、新Renderer/Topology Engineを組織専用で複製しない。

**三要素・図面・CRUD横断の受入差分**：[組織合成Profile§7](https://github.com/hat47x/tei-platform/issues/2510#issuecomment-6772933215)で、Data/Function/Businessの図面・SHO図解・CRUD等クロスリファレンスを同一Semantic Sourceから導出するときの意味、必要Source/Role/Quantity、Source逆編集資格を明示。合成例Org Explore／Case Handling／Workforce Report／人手Handoffでは、Rは**宣言された依存**、U?は**未受理の操作候補**であり、実Read/実Effect/Host Grantではない。Graphで二経路へ出現する同一C、D1のFTE集計セル、DFDの移管矢印、CRUDのセルは自動で同一更新対象にならない。ORG-XV-01〜07（軸だけ変更、FTEだけ変更、Path多重、DFDのみ変更、集計セル編集、非公開差、ABA）を既存#1124/#1125/#1803/#2497/#2593/#2514の検証へ接続する。モデル図面とCRUD表は派生表示であり、**一意なSource対象を持つ型付き変更候補だけ**既存Preview/Acceptance/reparseへ渡す。C0/C1の未カバー範囲は#1180/#2514の既存独立Expected/Scenarioの側で判定し、新Org専用Runner/Canonical/生成器は作らない。

**組織モデル由来の独立テスト義務**：[組織Profile§8](https://github.com/hat47x/tei-platform/issues/2510#issuecomment-6772933215)に、他チャットの未統合テスト生成Draft #3258〜#3261を組織の具体的反証へ接続した。ORG-TG-01〜08はHC≠Membership≠FTE、二Path配賦、軸だけの変更、欠落した集合、非単射図編集、DFD≠Effect、Public/Private差とABAを、**手で固定した独立Expected＋既知の誤実装＋常時unknownを防ぐ正例**で評価する候補。#1131のDataset候補→#1180のScenario正本→#2514の独立検証基準/Mutant→#2811のAIテスト入口→#1163の実Attempt Evidenceを再利用する。**コードC0/C1と、受理済み業務ObligationのCoverageと、Mutant判別力は別分母**。単なるAI生成ケース増加・全SKIP・未実行は充足に算入しない。現Native `allocation std.decimal`だけからFTE意味/単位系を自動採用せず、必要条件不明なら候補のまま停止する。新Org Test Generator/Runner/Expected Store/第二Scenarioを採用しない。

**優先順位**：#2586の実C1→D1 Function→Query→Provider Read→Consumerを先行し、その後P0合成のGraph/Headcount/FTE、P1二Function/SHO表示、P2原本への一意なIntent/Host Effectを段階検証。#2506/#2507/#2516/#2510/#2497/#2593が対応Owner。新Go/DB/GUI/SHO実行、Owner Quantity/Meaning受理、実Provider実証は**NOT RUN**。既存の組織計画§84・§85と並行採用可否を評価し、無根拠な大量Core新設やP0実Readの延期を避ける。

## 2026-11-08 組織Product E2の三ゲートと汎用Referenceからの証拠移転禁止

[組織Profile §9](https://github.com/hat47x/tei-platform/issues/2510#issuecomment-6772933215)へ、他チャットの#3263（三視点図/CRUDの実Source生成）、#3330（P0-M/P0-R/P0-Cの独立ゲート）、#3262（Design Context必要集合の取得完全性）、#2586（実Function→Query→Provider→Context）を、**単一Ownerを維持した実証依存**として接合した。P0-M（三視点Diagram/CRUD Source provenance）とP0-R（実行された物理ReadとRow/Field/Scope）は並行可能。P0-Cは同じMeaning ID／Source版／Snapshot／粒度／許可Scopeのもとで両証拠と独立Expectedを照合したときだけ成立する。図面表示PASS・Local Completion Holds・別DomainのPG PASSを組織E2へ流用しない。

**現Sourceで判明した具体的欠落**：main `reference/org-headless-workspace/semantic/organization/model.tei` には `CaseRecord.handlingUnitId→OrgUnit` の関係があるが、検査済み組織Source/現Operation SurfaceはCase用Function、Query、実Persistence Mapping、Provider Read／Host Grantを示していない。したがってC1→D1は、現時点では**意味を表す合成Expectedであって実DBから読み取った案件ではない**。#2586の既存受注Referenceでの汎用E2検査を先行可能とする一方、組織側に必要なFunction/Bindingを具体化し、P0-C接合前は `product-read-not-connected` を保持。何が実Workspace全体に存在しないかは#3262の範囲付き探索を経ずに断言しない。

**受入報告はM/R/Cの各状態と未解決を独立記録**：Model/Diagram Source、実SQL・args・Provider・row IDs/fields・read extent、必要母集団/非出力Predicate、Consumer Disclosure、exact HEAD/Attempt、Owner採否。#3238のTOP1・重複ID・隠れたPredicateはR→Cの負例として再利用するが、Caller申告 `CoverageComplete` を完全性のProducerとしない。実組織Provider/DB・三視点UI/SHO・全件業務完了・正式Owner受理は**NOT RUN**。

**実装回帰の追加（2026-11-08）**：[main統合済み #3353](https://github.com/hat47x/tei-platform/pull/3353)（squash `6cf510d6301a1e133ebbe69ad7bf004f03361b54`）で、`EvaluateCompletion`が`EmptyPolicyHandoff`・0件・`EmptyJustified=true`で、引継ぎ受領なしでも`holds`になり得た分岐を修正した。6ケースで`allow/requires-evidence/deny`と受領有無を対照し、確定`fails`を損なわない。**空集合の独立証拠≠相手方の受領証拠**を組織Profile§9.4のP0-C検収へ追加。ただし`HandoffConfirmed`/`CoverageComplete`はなおCaller申告なので、局所Go PASSだけを実Case引継ぎ・製品E2完了へ昇格させない。#3353のPR HEAD `cf3de9a61906167b3b5cc2fbd246e238a5018d5f` の[Repository Health #42391982442](https://github.com/hat47x/tei-platform/actions/runs/42391982442)はcompleted/success（gofmt、go vet、Go全体テスト、モデル検証）。ただし実案件Reader、相手方の受領、組織Product E2、Owner正式受理はNOT RUN。

**全件Handoffの残余条件（組織Profile§9.5）**：Local Evaluatorの`HandoffConfirmed`はClaim単位のboolなので、`L={C1,C2}`に対してC1だけのACK・同じACK二重計上・適用集合不明の一括ACK・旧版のACKでも、Callerが`true`と報告すれば局所的に`holds`を返し得る。#1125/#1444はOwnerが受理した対象別／正確な集合に束縛した一括Handoff証拠`H`を、#2586/#2622は実Read集合`R`を、#2514は必要集合`L`の独立Expectedを所有する。`L/R/H`のID・Scope・Revision・確認者／適用時点の適格性が揃うまでProducerは全件`HandoffConfirmed`をTrueにしない。#3353の6 GoケースとORG-HC-01〜07は別検証段階であり、Hの実読取や組織製品一気通貫は未実施。新専用Ledger/Completion Engineは作らない。

**外部Source・二時点・引継ぎ集合の再資格化（2026-11-08）**：[組織Profile§11](https://github.com/hat47x/tei-platform/issues/2510#issuecomment-6772933215)に、mainへ統合された[#3482](https://github.com/hat47x/tei-platform/pull/3482)の**限定Function Design Context（実CLI 13 Reply／42局所Control、宣言上の必要集合のみ）**と[ADR-0048](adr/0048-shared-application-api-and-transport-adapters.md)／main [#3403](https://github.com/hat47x/tei-platform/pull/3403)の**共通API／CLI-MCP Adapter**を、組織C1の次の未接続辺へ接合する。前者はCapability宣言を実Function→Query→ProviderへBindingした証拠でなく、後者の同値JSON・Tool広告は実Consumer受領／現在Authorityではない。

[#3380](https://github.com/hat47x/tei-platform/pull/3380)の二時点合成（現HEAD Node CI [#42462892409](https://github.com/hat47x/tei-platform/actions/runs/42462892409)成功、main未統合）、外部Provider Source-only driftの[Draft #3479](https://github.com/hat47x/tei-platform/pull/3479)、複合/並列FKの物理Catalog投影[Draft #3486](https://github.com/hat47x/tei-platform/pull/3486)は**別種の観測**。HRが申告する`knownFrom`≠TEI受信時刻、物理FK≠Owner受理Relation、Consumer World一致≠Imported Provider原Sourceの現在資格。Owner必要集合`L`／実Read`R`／対象に束縛された受領`H`と、必要Design Source`D`・Provider/Manifest観測を照合するORG-EC-01〜07を§11に配置した。正式Scenario/Runnerを複製せず#3262/#2586/#1125/#2514へ渡す。

**優先はP0-R一件Case実Readのまま**。組織実Case Function/Provider Binding・同一Snapshot観測・複数Consumerの実受領、全件Handoff/正式Owner受理は未成立。局所GoやSynthetic NodeのPASSをOrg E2/業務完了へ流用しない。

<!-- org-cf-plan-20261109 -->
**2026-11-09 組織改編・承認の時間整合差分（設計・未実行）**：組織Profile [§13](https://github.com/hat47x/tei-platform/issues/2510#issuecomment-6772933215) にORG-CF-01〜08を追加。C1→D1/D2の担当切替・旧Read・失効Grant・Replica遅延・比較不能revision・動的職務分掌・重複ACK・遡及Source訂正を、業務valid/known、TEI observed、実Provider Scope/Read Cohort、Business Rule、現在Host Grant、相手方Handoffの**別証拠**で判定する。異なるAuthority domainの時刻／版の大小を因果順序とみなさない。**#2586の一件P0-R実Readを先行**し、ORG-CF-01/04の読み取り観測を既存Attemptへ添付。その後、実行時再確認は#1174/#2499、受領と時間訂正は#1125/#1444/#3380、独立Expectedは#2514/#1180へ渡す。新Token、Global Clock、Authorization Engine、Provider、Gateを作らず、CF新テスト・実DB/Host・Owner受理はNOT RUN。
### 2026-11-08 TEI／SHO／KONの境界とHeadless／静的公開の最小受入（横断統合・KON実装は未着手）

TEIを静的サイト生成やブラウザ利用へ拡張する際に、**「TEI全体のWebアプリ化」「KON経由の必須Gateway」「標準UIを経由しないFrontend禁止」を採用しない**。これはWeb利用の否定ではなく、次の独立した仕事と責務を混ぜないための方針である。

| 役割 | 最初に所有する事実と境界 | 今回確約しないもの |
| --- | --- | --- |
| **TEI** | Source/Accepted Meaning、標準Data・Function/Operation・Trait/Provider Contract、生成Artifact、CLI/Studio等の型付きApplication API。既存ADR-0048に従いAdapterのTransportと意味Operationを分離 | TEIの全Authoring/Runtimeを新しい中央Web Hostで動かすこと、各WebページにTEI製Renderer/JSを同梱すること |
| **SHO Composer** | 自然言語文書・図表・引用を構成し、外部SourceのIdentity/Revision/受理済み部分を保持するDocument Consumer。分散・自由な執筆を妨げない | 文書が参照するデータのDB真値や現在Grant、TEIモデルの意味受理をSHO文面から自動確定すること |
| **KON Dataspace**（`hat47x/kon-dataspace`） | 足場のあるデータ保持・共有・版付き参照・公開を必要とする場合の**選択可能な**共通データ基盤。KONの実提供範囲は別リポジトリの実証で決める | TEI/SHO/SUIの必須サービス化、全Sourceの一括移管、別の意味正本・一元認可をTEIへ導入すること |
| **Web/Native Frontend・配信先** | Route/Brand固有のUX、DOM/Router、配信/SEO/SSR/Client・Edge/CDN、外部の実閲覧。B2Cでは完全Headless利用を正規の使い方とする | 標準表現型への強制マッピング、既存Commerce/CMS/決済の置換、TEIが配信状態を自己申告するだけで公開完了とすること |

**拡張契約の語彙を混同しない**：Hookはライフサイクル上の拡張点、Traitは満たすべき原子的な実装義務、Providerは選択・組込み・実呼出しが必要な実装、Pipelineはそれらの実行順・遷移である。静的サイトGenerator、Publish Adapter、Web編集UI、B2C Journeyはそれぞれ必要なHook/Operation/Providerを既存Contractへ載せる。図表で記述したLifecycleやWebチャット型Interactionも、Diagramだけで稼働と認めず、Generator入力→実Target/Hostの選択されたAction→Effect/Receipt→独立Readで実証する（#3503/#3278/#951）。Go単一バイナリのローカルTEIを基本経路として保持し、KON常駐やBrowser実行を導入前提にしない。

**利用形態は要求に相対的に選ぶ**。技術者はローカルTEI CLI→既存Git/CI→静的Artifact→外部配信を選べる。一方、Git運用を前提にできない執筆者向けには、**編集・Preview・Revision・公開申請・履歴参照等の限定操作面だけ**をブラウザへ出す案を検証する。その際、TEIの意味/Build処理を共通Operationに委譲し、保存が必要ならKONまたは既存CMS/ストレージをProviderとして接続する。BrowserがTEI製品の第二Modeler/Second Host/Authorityになることを前提にしない。B2Cの完全Native UIはTEI Core/Rendererをロードしなくても業務Action Contractを使えることを条件にする（既存[#3509](https://github.com/hat47x/tei-platform/issues/3509)／Draft[#3512](https://github.com/hat47x/tei-platform/pull/3512)のOwnerへ圧力を返す）。

**公開成功の判定は4段階**：①対象Source/Meaning/依存とDisclosure資格に対して生成・PreviewしたArtifact、②Host/Providerによる保存・Revision受領、③独立したPublish／Deploy ACK、④配信URL等から別Consumerが受け取った外部実Byte/版。Git push成功、SHO本文の保存、KONへの格納、Build Artifactの存在だけで④をPASSにしない。異なるSourceや外部Provider Raw/Manifestが変わった場合、World FP一致だけで公開物・引用・BOMの鮮度を継承しない（[#3477](https://github.com/hat47x/tei-platform/pull/3477)／[#3479](https://github.com/hat47x/tei-platform/pull/3479)／[#3490](https://github.com/hat47x/tei-platform/pull/3490)）。認可・公開範囲・未取得・partial・not-loadedを空集合または既存Grantと見なさない。

**既存Scenario/独立Expectedで検証する最小の横断反例（未実行）**：

| ID | 試行と独立に固定するExpected | 既存Owner |
| --- | --- | --- |
| **KW-01** | 同一受理Source→TEI CLI静的生成→Git/外部配信の実Byteを再取得。Build・Publish・公開後Readを別版・別証拠にし、外部未反映ならPublish完了を拒否 | #943/#1163/#1180、配備は対象Host |
| **KW-02** | Browser編集面で既存の型付き保存Operationを呼ぶ。KON/ストレージが未選択でも、別適格Providerがあれば機能を拒否しない。権限外/Version staleは拒否。UIからSource Meaningを再定義しない | #1158/#951/#1016、保存・公開Host |
| **KW-03** | 外部Providerの引用Rawだけ変更、TEI World FPは同一。SHO引用Revisionと公開物の再利用資格を差分として失効。必要Sourceがunavailableなら未取得を明示 | #2044/#3262/#3479、SHO Composer |
| **KW-04** | Native B2C Routeは標準Renderer/KON/TEI Browser Runtimeを搭載せず、外部Commerce ProviderのTyped Operationを利用。零の追加Runtime依存と実Effect/Consentを別に測り、CVR向上は実験なしで主張しない | #3509/#1158/#1174/#2514 |
| **KW-05** | 同一Function/SourceのCLI・Studio・限定Web操作で**選択Provider Bと実呼出B**を突合。B選択→A実呼出（結果同値）を失格、未受理/未Loadedはunknown。表現が同じでも逆編集の一意性を仮定しない | #1251/#951/#2497/#3267、既存D1受入 |

**高度B2Cの変更レーンと検証順（[#3509](https://github.com/hat47x/tei-platform/issues/3509)／[設計Draft #3512](https://github.com/hat47x/tei-platform/pull/3512)から受領）**：施策・Creative・Presentationだけの差分は、業務意味・Provider・契約・Host Grantに影響しないことを差分で証明できる場合、無関係なMeaning再受理を要求しない。一方、価格・在庫・決済・同意・注文Effectに触れる変更は通常の業務Operation／Effect／Disclosure／独立Expectedの再資格化を省略しない。実際のVariant配信、実表示・Intent、権威あるOrder Effect、計測対象・同意・帰属、採用・撤回は別の観測段階とする。画面のRollbackが実注文の補償になるとは扱わず、CVR向上は実験による独立測定の前には主張しない。

**B0〜B4の圧力試験**：B0=Native HeadlessのRead-only Fit/Gap、B1=独自FrontendからTyped Operationを実Provider経由で呼び別Consumerが実結果を再読取、B2=一部標準UIとの比較とExperiment/Publish/Browser・Cache負例、B3=AIチューニングの統計品質・独立Benchmark、B4=外部サービスAdapterとしてのPlugin採否判断。初手はECの一操作とし、予約・会員等は後続の異質Consumer候補。KW-04との独立Expectedを#2514/#1180へ、仕様・業務効果のOwnerを#3509へ返す。B0〜B4は全件未実行であり、#1251の通常CLI/Studio S2、#2586の実Function ReadのHard Gateを増やさない。

**実装された範囲と横断判断を更新する（2026-11-09）**：[main #3579](https://github.com/hat47x/tei-platform/pull/3579)はFunction定義書へHook宣言を既存Phase順で導出する実GeneratorコードとGo試験を統合した。ただし**宣言が出力されることと、Runnerが実Targetで実Hookを呼んだことは別**であり、Provider実呼出・実Effect・B2C配信を認定しない。さらに[main #3549](https://github.com/hat47x/tei-platform/pull/3549)／[#3588](https://github.com/hat47x/tei-platform/pull/3588)でC-1 Router／C-2 ServiceCarrier／C-3 Deployment Binding／C-4非接続を選択前の比較候補として保持し、C-2は試験外の限定Reference組立に進んだ。**TEI/KON/Headless全体のためにC-1/C-2を両方常設したり、中央Gatewayを必須化したりしない**。どの入口を製品化するかは#951/#1016/#1251と実二Consumer・Host現在権限の証拠で別判断する。公開CLIとStudioへ静的参照Inspectorを配線する[Draft #3540](https://github.com/hat47x/tei-platform/pull/3540)はS2-referenceの実装候補だが、正式Profileでの任意Provider切替と実Target/BOMのS2-productは未実証である。

**実行順は現行P0を変えない**：まず[#3378](https://github.com/hat47x/tei-platform/pull/3378)の既存CLI/Studio直結Baseline、main[#3371](https://github.com/hat47x/tei-platform/pull/3371)のHost選択Service、main[#3381](https://github.com/hat47x/tei-platform/pull/3381)のComposition選択Inspector（局所Go二Consumer実呼出まで）を、**通常CLI dispatchと実Studio Bridge双方の実Invoke**へ接続（#1251のS2）。並行して#2586の実Function→Query/Provider Read→Context/表現を検証する。KW-01〜04はこの結果から必要な小さなConsumer/Targetだけ実験し、KON全製品・B2C全機能・静的サイト配信サービスを新Hard Gateにしない。Performance/AIのHint→統計→独立測定はmain[#3520](https://github.com/hat47x/tei-platform/pull/3520)の限定Referenceを再利用し、計測資格なしの最適化やCVR効果を成功とは呼ばない。

**証拠状態**：本節は横断した責務・受入の計画整理。KONリポジトリ作成は実Web/保存/公開APIの実装証拠ではない。KW-01〜05、TEI Web限定面、SSG外部公開、実Native B2C、KON/SHO/TEI一貫実行・Owner受理は**NOT RUN**。新Core、第二Host／Registry／Resolver／Auth、独立Scenario正本は追加しない。

### 推移的Record Scope × 担当変更の接合（2026-11-09、設計のみ）

<!-- org-visibility-cutover-plan-20261109 -->
既存[組織意味設計§58](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)の**用途を指定した広域Org Policy・明示例外・Read-only**（合成[PR #2781](https://github.com/hat47x/tei-platform/pull/2781)のVP/VE）と、ORG-TRの**Case担当切替**を新ORG-VTR-01〜06で交差検証する。Membership異動≠CaseのR1→R2変更≠現Read/Update Grant≠Handoff。旧D1のCache/表示、D2の新関係、A→B→A復帰、適用除外/not-applicable、共同責任・多重Pathによる件数増加、Host/Provider行Scopeの未確認を同一Caseで判別。広域Policyの「既定適用」は**Ownerが対象指定したData/用途内**だけであり、任意Tenantへの既定許可ではない。

**既存工程**：HJ-09の最初のProduct Function/Provider read-first (#2586)は維持し、組織C1の次段P0-Rで#2506/#1656のRow Filter実施行点、SQL/実Rowset・Scope、List/Count/Page/Report/Export、Consumer/Cache現在性を独立照合。Current Grant/Effectは#1174/#2499、Model Map・UI/SHOからの開示は#1124/#2516、独立ScenarioとExpectedは#1180/#2514へ。Providerに行Scopeを強制できない保護Routeは拒否候補とし、UIで隠すだけで合格にしない。合成VP/VEのfixed-head PASS、mainのInspector/Hooks局所実装、実PostgreSQLの別Operationの成功は、Org Caseの実Policy Enforcement/Consumer/Owner受理を意味しない。第二Policy Engine/Host/Test Runnerを追加しない。**新VTRの実行・製品適合はNOT RUN**。

### 組織CaseのSource→実Read→Decision候補→Effect→Handoff：証拠を接合するが、権限と正本を増やさない（2026-11-09）

<!-- org-evidence-join-plan-20261109 -->
[組織意味設計の新追補](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)で、S（Source/Meaning）・Q（選択/実呼出Provider＋実Rowset/Cohort）・D（受理済Rule相対の判断候補）・E（現Principal/Grant・OperationID・Commit/Receipt）・H（必要L/実Read R/対象ACK H・実Consumer）を**別の同一性と資格**として整理。Case ID/World FP/同名・別Run成功の寄せ集めで、同一Snapshot/Authority/Attemptを捏造しない。遡及訂正は「訂正後の履歴解釈」を変え得るが、過去の実Effect/送信履歴を消さず、旧Receiptを現在有効化しない。SUI/外部REAのCandidate→SEI/人間のProposal→TEI Owner-accepted Rule→Host Effectの昇格はそれぞれ別決定で、長期Ontology DSL/Decision TwinはP0 Readの前提にしない。

**最初の製品優先**は引き続き#2586のorder_entry.recent_orders 1 Function/1 Provider/実Consumer。次段組織Case C1では、同WorkspaceのRelation occurrence、Function/Queryとselected/invoked Provider、SQL/args/Scope、Rowsetのpartial/0件、§58のRecord Policy（実Provider施行は別資格）、一Consumerの受信・Disclosureまでを**同じAttempt**で確認。判断・実Effect・L/R/H Handoffは後続P1で#1174/#1125/#1444/#2514/#1180/#1163の既存Ownerへ段階接続する。#3717のPostgreSQL4系統非SKIP成功、#2781の合成Org Policy成功、mainのInspector/Hook実装は各々の限定Witnessであり、Org全体の現在Grant/Provider Enforcement/Client E2Eではない。本追補はdocs-only、新規実Go/DB/Host/SEI/SHOテストはNOT RUN。

### 外部Observationから組織Relationへの候補写像を段階受入する（2026-11-09）

<!-- org-external-observation-t1a-plan-20261109 -->
[組織設計の外部Admission追補](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)で、[REA T1-A前段 #3496](https://github.com/hat47x/tei-platform/pull/3496)の**実REA 6.7.0・固定HEAD 4/4 Job・独立監査＋候補試験42/42 PASS**、valid **13/14**・partial **11/12**の二時点Source byte付きCandidateと各1 unbound Rootを、組織意味の**正式採用前の観測**として接続した。元データは公開JS合成Sourceであり、組織HR/実Caseの外部Producerが動いた証拠ではない。Source前後一致はA→B→AやProvider Actual Inputの原子的Snapshotを証明しない。旧#3496はclosed/unmergedで試験用Projectorはmain未搭載。Model Map #3866のmain統合と足しても外部Observation→正式Model Map/製品CLI実受領は未成立。TEICanonicalSubject=null／acceptedMeaning=false／sourceClosure=not-qualifiedを維持。

**次の欠落辺は1 Observation＋1 Edgeのみ**：既存共通Application API（ADR-0048/#1158）へProvider版・Evidence ID・Node/Edge個別ID・方向/多重性・Source path+raw digest+span・coverage/unknown・Scope/Workspaceを損失なく渡す限定read-only実Consumer。#3394が外部Producer/Consumer照合を所有し、#2386がPerson/Membership/Occupancy/CaseへのOwner意味写像、#3492/#3518/#1124がAccepted/候補を混同しない三視点Model Map、#2516が公開Slice/診断も含む開示、#2514/#1180が独立負例を所有する。ORG-EXT-01〜06で同名別ID、多重Edge、unbound端点、partial 0件、同byte別Workspace、A→B→Aを対照。既存InspectSubjectへ外部JSONを偽Canonicalとして流さず、独立Graph Store/Resolver/Parser/Hostを作らない。

全TEI最初の実Product Read #2586 HJ-09は**並行して最優先のまま**。External候補が得られても、実組織のCase C1 read、現在Principal Grant、Rule受理、Effect/Handoff L/R/H、SHO/SEIの正式受信まで昇格しない。上記外部Candidate→TEI Operation／組織Owner採用／Org Map実表示の新試験はNOT RUN。


### 共有Harnessで製品間を移る際の組織Scope／権限の再資格化（2026-11-10、設計のみ）

<!-- org-shared-harness-authority-plan-20261110 -->
[組織意味設計：ORG-HS-01〜08](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)に、他チャットの**製品共有Harness Session／Activity Contextの分離**を、既存§58 Record Policy・ORG-VTR Case切替・#2516 Construction Context/Consumer Disclosureへ接合した。SEI/SUI/TEI/CSW間で活動の目的、候補Selection、出典、未決、復帰位置は引き渡せるが、**Verified Principal/Issuer/Tenant、現行Grant、Policy epoch、Source実体/版、Recipientへの開示権、Operation Effect/Receiptは引き継がない**。Harness自体をTEIに内蔵せず、製品別UI/APIと既存Adapterを利用する。共有Session・選択済みCase・前Consumer成功は、実Provider Scope/Read/Count/ExportやCommitの許可ではない。

**追加する最小検収**：HS-01移動先のRead拒否／HS-02組織異動後の旧Cache／HS-03 Privateの件数・図・診断漏えい／HS-04目的・受信先変更／HS-05 partial 0件の不存在誤認／HS-06 Issuer/Tenant越境／HS-07 Selection ABA・後着AI応答／HS-08失効中のCommit uncertain再試行。既存#1131/#1180/#2514を独立Expectedの正本として使用し、#1163で同Attemptを照合。成果を新しい全製品共通Session Store、Org Grant Engine、全Route共通Policyの追加要件にはしない。

**順序とOwner**：#2586 order_entry.recent_orders の初回実Function→Provider→Consumer R0をP0最優先に維持。並行して公開合成C1をheadlessでCandidate→TEI read-only Operationへ引き渡す**試験準備**（#2386/#2516/#1124）。後続でCase C1実Provider/Scope、Product Consumer（#2506/#1656）、Current Grant・Effect（#1174/#2499）、ACK/Handoff（#1125/#1444）、Harness実Tool/送出・Log（#2509/#2515）をそれぞれ実証する。通常Org Data全体への秘密指定、全Session切替での一律再取得、旧送達済みCopyの消去保証はしない。

**資格**：ORG-HSは設計／受入候補のみ。共有Harnessの実製品間連携、実Grant、Row Scope Enforcement、Egress、Owner受理、HS-01〜08試験はNOT RUN。既存M0、合成Org Policy、DSHのTool Read、別Operation実PostgreSQL成功をこの縦断のPASSとして集計しない。


### SUI/CSWの探索的理解をTEI組織Meaning候補へ受け渡す（2026-11-11、受入設計のみ）

<!-- org-affinity-admission-plan-20261111 -->
[組織Meaning設計のORG-AG-01〜07](design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md)で、SUI Card/Island・CSWのKJ法的グループ／仮説と、TEI Person/OrgUnit/Membership/Case担当・承認Ruleの**正式Relationを非同一**とした。許可済みの理解パックは、元CardのSource/Review資格、反証・保留・孤立残差、Candidateと未解決、Purpose/Audienceを保って利用目的別に構成する。組織MeaningへのMappingは任意・非可逆で、Owner受理された部分だけがTEI正本を変更する。TEIの逆編集からSUIの元Islandを黙って書き換えず、SUI/CSWをTEIへ全面移行させない。

**単一の最小試行**：公開合成Card数件をSUIの実許可read-only Projectionから取得→実Consumerへの送達・受領・要約での反証残存を別々に観測→TEI既存read-only OperationにSource-backed Mapping Candidateを照合。SUI現context-projection.v1のSource資格はL0まで（#3368）。元Sourceを取得したかのようにL1/L2へ昇格しない。類似順位候補の成功（SUI #3157）やReviewは組織受理を意味しない。反例はAG-01同名島とOrgUnit／02 C1→D2未受理案／03孤立反証落ち／04L0根拠不足／05生成と実受信の差／06Private公開差／07新材料による再検討を、既存#1131/#1180/#2514で評価する。

**既存Owner・実施順**：探索/実投影はSUI #3368/#3157、用途別構成・開示はTEI #2516、TEI組織意味は#2386、Model Mapは#1124、実Tool/Consumerと不確実性は#1163/#2516。#2586 order_entry.recent_orders のFunction→実Provider→Consumer R0は依然P0最優先。組織C1実Provider/Scopeと現Grant/Effect/Handoffは後続#2506/#1656/#1174/#1444へ。本計画は新Affinity Runtime、共通Session Store、第二Meaning正本、別Gateを設けない。実SUI/CSW→TEI/SHO/SEI E2E、ORG-AG新試験、Ownerの正式採用、実Org DB/Hostは**NOT RUN**。


## 2026-11-12 高度組織モデリング：構造受入から業務責任・三視点・Consumer縦断へ（#2386/#4490）

<!-- org-crosslane-acceptance-20261112 -->

**段階計画の正本**：[高度組織モデリングの段階計画 §10](design/ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md#10-2026-11-12-横断統合組織責任実行三視点知識consumerを一案件で接合する採用前計画)。#2386の意味Ownerと既存Data/Relation/Rule/Operationを維持し、組織専用Core/Workflow Engine/Policy Engine/第二Expected正本は追加しない。

- **基線の訂正**：PR #4342の設計計画、#4399の構造テスト、#3741の責任Cutover設計、#4256のSUI/CSW候補境界、TEI→SHOの#4317/#4467はmain統合済み。ただし#4399の構造test PASSは二時点Query結果・承認候補・委任拒否等の製品上の振る舞いを保証しない。#4479の固定対象AI指示はDraft。
- **次の独立Evidence**：#4490でORG-BEH-01/02（valid/knownの結果差・WHERE/ORDER/EXISTSの非出力依存）を最初に実行。次にDirective/候補/代理/Case移管・ABA。WU-11〜14は子Issue #4501 に分離し、業務手順・Handoff ACK・loss-aware Projection・固定対象と再投影・保護Field例外を既存Ownerへ引き渡す。
- **Case C1の接合**：Membership変更≠Case責任移管≠Rule候補≠現行Host Grant≠Effect Receipt≠Handoff ACK。Source/Meaning、実Query/Read Cohort、Business Decision、Authority/Effect、Consumer受信のそれぞれを同一Attemptで明示的に照合。単なるCase ID/World hash一致は同一資格の証拠ではない。
- **優先順位**：全体の実Provider Read #2586 HJ-09を最優先に保ち、組織C1の合成Behavioral Expectedを並行作成。続いて実Org Rowset/Scope #2506/#2622、三視点/GUI/SHO #1124/#2497/#2516、Host Grant/Effect #1174/#2499、業務Handoff #1125/#1444へ接続。各段階は実測・未実行・Owner未受理を分ける。普通の組織Dataを組織管理モジュールというだけで特別な秘密情報にしない。

**三要素分析法・REAの補強（§10.7）**：X-TEA Modelerの三要素分析法（Data/Function/Business）を方法論名として正確に用い、TEI独自のSource/CRUD/Evidence接合を拡張として明示。Task契機≠Function起動、Function成功≠Business完了、REA Agent≠認証Principal、DB FK/SQL観測≠業務意味受理、Studio/SHO/AI Selection≠Source更新資格をCX-13〜18で判別する。実行計画は#4501、#1125/#1444、#2586/#2387、#1124/#2516の既存Ownerへ配分し、新Core/Runtimeを増やさない。

**組織FTE換算の命名衝突（2026-11-13、§10.8.1/§10.11.3）**：r7 `allocations[].basisPoints` は現行合成データ内でraw 100=1 FTEとする旧フィールド名であり、金融のbasis point（100 bp=1%=0.01）とは倍率が違う。Field名だけで汎用bpへ自動変換しない。#1705/#2495/#2593へUnit/Scale/Basis判別を接合し、既存JSON一括変更や新Unit/Query Engineは先行させない。同一Allocation ID重複は衝突、knownAt訂正によるD1の値変化と同条件D2値不変はClaim単位で分け、実Providerの同一Read cohortを推測しない。新たな合成試験はORG-CAP-UNIT-01/02、ORG-CAP-07b。元の§10.8 r7修正と7件のmain済み試験資格を維持し、追加の現行HEAD実行結果は別途確認する。

**組織の運営能力・数量・Scenario（§10.8、fixture整合修正）**：既存 `reference/org-semantic-views/source-fixture.json` の `org-synthetic-r7`（validAt=2027-06-15、knownAt=2027-07-01）に揃え、P1→D1(A1=1.0)、P2→D1(A2=0.6)・D2(A3=0.4)、P3→D2(A4=1.0)とした。全体Unique Person=3、Membership=4、Allocated FTE=3.0（D1 1.6、D2 1.4）。前稿の2人/3所属/2.0 FTEは既存fixtureにない別仮定だったため撤回。欠員はP1の利用可能配賦相当D1=0.6、仮配賦変更はP2 A2 0.6→0/A3 0.4→1.0でD1=1.0/D2=2.0とし、確定Membership/Case/Grant/Effectを改変しない。knownAtを2027-07-03へ進めた同一validAtではA2が外れ、Membership=3/FTE=2.4。実行可能な既存fixture合成テストを**main統合済みPR #4532**（Node v22.16.0、ローカル7件PASS、main squash `2e52ca47`）に切り出し、#2510/#4490/#4501とExpectedを重複させない。ORG-CAP-06/08、実Provider/Studio/SHO/Host、Owner意味受理は未成立。一方、合成Node Referenceは#4532 head `fa2d8ce98ff77536f51e32095748f63d935a2d07` の [Actions #43242282586](https://github.com/hat47x/tei-platform/actions/runs/43242282586) で313/313件PASS（新ORG-CAP 7件を実行ログで確認）。これは製品上のQuery/Provider/Grantを証明しない。#2495/#2593標準Quantity、#2586実Read先行、#2386 Rule/責任、#1124/#2516投影/開示、#1174/#2499実EffectのOwner境界を維持。

**組織C1のSchema/UI Mapping変更追補（§10.9/§10.9.4）**：実テーブル定義とDesired Schema、各版の**テーブル定義モデルFieldそのもの**、上位の業務意味/Computed Field、Function/UI要素とBindingを分離する。正本でFieldが削除されたDesired版、または完全観測で削除を確認したActual版では、**その版のテーブル定義モデルからFieldを除去**する。一方、UI要素は残して未マッピング警告、CRUD/派生Query/SHO/AI対象の依存を診断する。外部DBの変化を理由にDesired Sourceを無断更新しない。partial/unavailableを削除確定とせず、同名再作成・Schema ABAで旧Bindingを復活させない。画面追加だけでは列を作らず、明示的開発者操作→Preview→Migration/Grant→実DB再観測へ進める。GUI/CLI/実Targetは同じ操作意味。#4515の**Subject Pin先行**と描画Edgeの非意味ID・Relation Pin unsupportedを織り込み、#1143の横断変更資格、#2497の逆写像、#4125の旧新Client互換と照合する。ORG-MAP-01〜10は既存Expectedを改訂・再利用（実試験NOT RUN）。#1126/#1124/#3270/#1452/#4479/#4501へ配分し、#2586の実Read優先を維持。三要素分析法は業務向けの方法論/PluginでありTEI汎用Coreを専用固定しない。

**組織モデルの参照・未使用診断とStudio性能（§10.10）**：#4472の三要素CRUD索引/明示定義・矛盾表示と#4475の同Kind同ID/Fingerprint純粋性を先に検証し、その後#1177/#3788の既存Source Index/Traceで順方向・逆方向参照、Source位置への高速移動、Scope完全時だけの「参照数0・未使用候補」を実証。0件/partial/unknown/未解析Pluginを混同せず、静的Unusedから自動削除しない。#4515のSubject Pinを先行、意味Relation ID未確認の描画Edgeから編集しない。大規模性能はcold/warm p50/p95、CPU/メモリ、Index構築と実Query/描画を分離計測。必要なPanel/Pluginだけ遅延起動する案を#1824で比較し、第二Index/Parser/Hostは作らない。WU-12/13とORG-MAP/CXのExpectedを再利用し、#2586実Read優先を維持。実ナビゲーション/性能・Owner意味受理はNOT RUN。

**三要素・REA・組織参照の実証資格（§10.10.4）**：三要素Fingerprint入力破壊はmain統合 #4507 で限定修正済み、CRUDのData索引3箇所は#4513で修正・5 Workflow成功を確認後main統合（squash `7e36f85a`）。明示CRUD-only組の派生表脱落/Unused誤判定は#4551で修正しmain統合済み（squash `1a2271d8`）。矛盾の全件列挙と構造`ok`/CRUDレビュー資格分離は#4534（main merge `5f35d5dc`）で実装済み。ただしStudio/SHO/AI Consumerによる要レビュー診断の正しい表示・適用停止は未検証。CLIのtrace.unresolvedは返却行の未解決、projectionUnresolvedは全Graph投影の脱落を示す（#3906）。一方のみが0でも全体完全とは判定しない。外部REA 6.7/6.8のProducer Protocol検証 #4257 と、実TEI受領・Owner Accepted Meaning・正式CRUD参照は別（#3394）。ORG r7のNode 7件はPR #4532の固定HEAD fa2d8ce.../Actions #43242282586で合成CI成功、実Provider/Studio/Hostの代替ではない。組織参照の次のExitは「Accepted参照／外部候補／未解決・partial」の三群を混同せず、#4472/#4475→#3262/#1177→#3394実Receiver→#4515/#3788の順に既存Expectedで段階確認する。P0 #2586は維持。追加した§10.10.4は設計/現況の照合で、新製品試験はNOT RUN。

**UI起点のデータ文脈設計とClaim別再評価（§10.11）**：業務向けModelerではUI部品から既存Field／新しい永続Field候補／Derived／Transientの4経路を選択し、**新Fieldの明示設計操作**が行われた場合に限ってData Context→Desired Schema→Impact Preview→既存Migration/Grant→実DB独立再観測へ進む。UI配置だけではDDL/実DB変更なし。SQLの`SELECT *`/`alias.*`/`NATURAL JOIN`やWHERE/NOT EXISTS等の暗黙依存も追い、#2582/#4518/#4526のClaim別Required Closureでaffected/scoped-unaffected/unknownを判別。無関係なUnknownで全画面を過剰失効させず、未知の依存を無影響としない。SHO #50 QH-08〜10で同Markdown別Source・TEI実再束縛・実Recipient受領の資格を分離。#4524（同Kind同ID）、#4513（CRUD Data索引）、#4514（SHO図Source-only drift）、#4534（CRUD衝突診断）は範囲を限定した修正としてmain統合済み。ただしConsumer全面Gateではない。#4532（r7合成Node）も正確なPR HEADの4 Workflow成功を確認後main統合済み（squash `2e52ca47`）。ORG-MAP/ORG-CAP/ORG-BEH/CX/PIN既存Expectedの追加assertのみとし、別Engine/Accepted意味正本を作らない。#2586の実Read P0優先を維持。新しいProduct E2EはNOT RUN。

**REAの名称区別と組織/Subsystem責務（§10.7/§10.11.4）**：Resource–Event–Agent（業務・会計の分析モデル）と、#3394/#4322の外部解析ツールREAを別概念として扱う。解析ツールのObservation/Node/EdgeはSource-backed候補であり正式Membership/CRUD/Agentへ自動昇格しない。重複Node/Relation/並行Edge由来の候補ID衝突は任意の一件を採用せず拒否（#4542は親#4322へ統合、#4322はmain未統合）。同一Attemptの実TEI read-only API受領・Owner意味受理は別。組織の所属/Case取扱責任、論理SubsystemのDirect CUD・Owner、実DB配置、Host Grantを異なる軸として、組織改編・Subsystem表示名/IDの付替えだけで権限/責務移転を推測しない。#3250のSP-X23/24 Draftおよび#4501 CX/ORG-BEHの既存Expectedを引き継ぎ、Required/Supplied/Missing Guardと実Attempt/Effect/ACKを別検証（#1129/#1174）。変更影響は#2582/#4518のClaim別Closure、物理Field変更は§10.9/10.11に従う。外部REA/Product/Hostの新しい実接合はNOT RUN、新Core/第二意味正本を作らない。

**実PostgreSQL Readと組織Queryを接続する実装順（§10.10.4/§10.11）**：#2586のmain統合済み #4539（squash `ee1af44c`）は、受注Query `order_entry.recent_orders`を一つのEnvironment/Database Instanceと実PG16.15 Providerへ明示結合し、Raw Rows→Function Data Context→隔離CLI `tools/reference-browse-read` で実読取した限定実装。使い捨てPGで0件、NULL/欠落、RLS、権限拒否、Source SQL差を対照し、該当HEADの7 WorkflowはSUCCESS。ただし現在の製品`tei` CLI/Studio/SHO/AI Harnessには未結線、Env↔実Providerはhost-asserted、coverage/不存在/current Grantは未確立、参照実装のmain統合は確認済み。ただし通常CLI/Studio/組織固有Queryは未完。P0の次の一手は**通常CLIで選択と実Invokeと同一AttemptのQuery/params/Rowset/Context受領を一致させる**こと。最初の実例は受注Queryであり、組織C1のData/Case/二時点業務Queryへの実移植は別受入。#4534のCRUDレビュー資格をHost許可と混同しない。新Runtime/独立Expectedなし、GUI完成をCLI P0の前提にしない。

**二時点SQLの実装不足を具体的に修正（§10.11.5、#4490）**：既存org-staged-axisのas-of Queryは把握時点の比較方向が逆、終了時点の条件が欠落、物理Persistence表名も不一致だった。Native Goの旧WU-07テストは入力の宣言だけを検証し、SQL実行ではなかった。Queryの引数をvalid_at/known_atとし半開区間の2軸で絞り、Mappingの期間終了列と物理表を一致させる最小修正を別PRで実施する。ただし現Persistenceのmembership_id単独PKでは同一occurrenceの複数既知版を共存できず、二時点実保存・実PG Provider/行の整合・partial/Source観測時点は未検証。実装順は構造整合→版履歴の保持→実ProviderでknownAt変更の独立Expected→Studio/SHO/Case C1接続。#2586の受注Query P0実Readは妨げず、組織専用Temporal Engineや第二Runtimeを作らない。

**採否**：新規Source/Rule/公的契約や業務責任の解釈はOwnerの意味受理が必要。本追補はdocs-onlyであり、実Org Provider/Host・SHO/HarnessとのOrg C1 E2E・Owner意味受理は **NOT RUN**。

**Studio Desktopの実構築→TEIセルフビルド／組織C1受入の最短接続（組織計画§10.12）**：製品の最優先目標は**Wails/Go版TEI Studio Desktopの実OS起動**（#1824）と、既存TEI Typed Build Graphを使った**stage1 TEI実プロセスからstage2 Studioを再生成するセルフビルド**（#2831）。Go CLIのセルフビルドCI成功をDesktop起動・D2再生成成功とは見なさない。**生成Workbenchのserver/Chromiumでの成功と通常Wails Desktop/WebView2での成功も合算しない**。#4541のGD-0〜4に沿い、生成assetをserverなしのDesktopへ組み込み、実WebView2→IPC→Go Catalog/Inspect応答まで同一HEAD・実Process・Selectionで検証する。組織C1のread-only Data/三要素CRUDは#4551 main統合の派生結果と接続し、#4539の実PG受注Queryは通常tei CLI→Studioの別Consumer経路に結ぶ（同一業務Queryだと誤認しない）。X-TEA三要素は業務向けPluginとして移植し、汎用Coreを固定しない（#1124/#1125/#1251）。D0実Desktop、D1 Model/Provider受領、D2セルフビルド、D3 X-TEA Plugin、D4 Pin/SHO/AI適用の各資格を独立Expectedとexact HEADで判定する。#2586の通常CLI Provider実ReadはGUI完成待ちにせず並行P0。#4500/#4551のmain統合は確認済みだが、D0〜D4の実行証拠は未成立。新Build Engine/Host/Scenario正本を作らない。

<!-- org-six-scope-headless-master-20261113 -->
### 2026-11-13 組織意味とPlugin六つの適用範囲・B2C Headlessを接合する（受入設計）

[段階計画 §10.13](design/ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md)に、配布（Distribution）・設計編集（Authoring）・アプリ構成（Composition）・組織関係（Organization）・実行（Runtime）・用途別開示（Disclosure）を**別の適用範囲**として照合する方針を追加した。六つは比較・受入観点であり、固定Core enum、第二の認可基盤、TEI独自Hostではない。兼務Personや同一Plugin配布から別Tenant Grantを推定せず、B2Cの購買者をMembershipに押し込まない。専用Frontendが型付きApplication APIを呼ぶ完全Headlessを選べることを維持する。

最初に既存r7組織Sourceと追加の合成Tenant/Consumer/Host Scenarioを区別し、#4490 ORG-BEH、#4501 CX、#2903 SCOPE/JOIN、#3509 B0〜B4を**第二Expected正本なし**で横断照合する。#2586の通常 tei CLI→実Provider Read→一ConsumerをP0として先行させ、その後Org C1の実Queryと複数Consumer、Host現Grant/Effect/Receipt、Disclosure非干渉性を既存Ownerに引き渡す。組織Dataを無条件で秘密にせず、保護を要する項目と接続先に応じて制御する。UI選択・AI提案・Plugin配布・局所Node/Go PASSは実Readや現在Grantの代替としない。

**状態**：設計追補のみ。実Org C1 Provider、実Multi-Tenant/Headless B2C、Host/Disclosure、CI、Owner意味受理は未実施。TEI Desktopセルフビルドと初回実Readの優先順位を変更しない。

## 2026-11-14 組織C1の二時点保存・責任候補・Consumer接合を最短実証へ渡す（#2386/#4490/#4501）

**受入補遺**：[ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)（設計・未受理）。既存のORG-BEH/CX/org-synthetic-r7を再利用し、第二Expected/Scenario・組織専用Workflow/Policy/Temporal Runtimeを作らない。
- **11/14横断実装の限定成果（補遺§16.1）**：main統合済み#4629のCapability Result Preflight／Adapter call0は組織C1の宣言検査に再利用するが、動的承認者の適格性・現在Grant・実Effectではない。Draft #4627のWails Server公開Method InventoryもWebでのPrincipal/Workspace/DB/WS送出隔離の合格ではない。新GoテストPR #4633の固定HEAD CIは合格済みWorkflowとRepository Healthでの新テストの成否を分けて判定。#2611複数DB Alias、#4566/#4567時点SQL、#4490 BEH、#4501 CXの既存Owner/独立Expectedを再利用し、組織固有の新Host/Runner/Schemaを先行採用しない。P0 #2586実Read/#2831 Stage2を独立維持。
- **11/14 nullable Expand候補のCI失敗を再資格化（補遺§20.4）**：#4633のDDL差分追加HEAD `65b83487` は文字列`before`とSchema Snapshot変数の重複で実Goコンパイルエラー（[Studio Source Loop #43572108566](https://github.com/hat47x/tei-platform/actions/runs/43572108566) failure、[Health #43572108581](https://github.com/hat47x/tei-platform/actions/runs/43572108581) cancelled）。修正HEAD `85f65700` で`physicalBefore`と区別して再提出。[Health #43572492328](https://github.com/hat47x/tei-platform/actions/runs/43572492328)を新HEADとして合否判定し、旧7テスト＋5負例の合格は旧HEAD `2c155383`限定で保持。実PG Alter/Backfillは未実行。#4125移行段階と#4647被覆partialに沿い新Owner Gateは作らない。
- **11/14 組織C1のnullable Expand候補と取得被覆を接続（補遺§20）**：固定HEAD 2c155383の7 Goテスト＋5 DSL負例は[Health #43558325121](https://github.com/hat47x/tei-platform/actions/runs/43558325121)でGo/gofmt/vet success。新HEAD 65b83487の実DSL Loader→Desired→Source由来BaselineとのSchemaDiff→Postgres DDLは、変更がnullable `person_support_id`のADD1件でPK/FKを変えないことを**オフライン限定**で検査（新Health #43572108581は別資格）。実DB・過去人物Backfill・Owner受理は未実施。[#4647](https://github.com/hat47x/tei-platform/pull/4647) W1〜W3の`partial`とNative SQLの`unestablished`を使い、見えた組織件数/FTEを全件Claimへ昇格させない。#4644参照PG/RA-1も通常`tei`製品Readの証拠ではない。既存#4125 History移行、#4490 Person意味、#2586 Product Provider→Context/Consumerを分離。Studio Stage2 #2831と通常CLI #2586は独立P0維持。
- **11/14 組織C1を実DSLの正常／拒否へ接続（補遺§19、§19.4）**：#4633はorg-staged-axis一時複製上でPerson→Membership keyless Relation＋独立relation-storage .teiをWorkspace Loader/Source Index/World Validator/PostgreSQL offline Desiredへ通す。**7件版b124a5feの[Health #43556769631](https://github.com/hat47x/tei-platform/actions/runs/43556769631)はsuccess**。続く新HEAD 2c155383は同一7トップレベルテスト内へ、誤Source Attribute・別target Relation・unknown Relation・未解決Mapping・二重Sourceの拒否5サブケースを追加。[Health #43558325121](https://github.com/hat47x/tei-platform/actions/runs/43558325121)は新版単独で判定。正式Source/Schema/Owner Meaningは変更せず、二時点PK・実DB Person Read・Consumer/Hostは未実証。#4619 D1実Invocation/Recipientと#4638実PG参照7件成功は別領域の成果として引用し、C1 Productの完了と混同しない。#4599 DSL・#4125 History移行・#3421 Studioを既存Ownerへ委譲し、#2586 CLI実Read/#2831 Stage2を独立P0として維持。
- **11/14 Relation Storageの実装済み範囲を訂正し、C1実Validatorへ接続（補遺§18）**：main統合済みPR #246〜#252/#257では、RelationStorageの型だけでなくDSL Parser/Workspace Loader/Source Index・World Validation・Desired Schema・SQL realization・SchemaDiff/Migrationまで既存実装を確認。C1では新実装を複製せず、Person→Membership keyless候補の方向/SourceAttributeを採用前に比較する。[Go Draft #4633](https://github.com/hat47x/tei-platform/pull/4633) に第6テストとして実org-staged-axis World→既存Production Validator→Postgres offline Desired、誤source属性の拒否を追加。**5件版 509296692 の[Health #43549877695](https://github.com/hat47x/tei-platform/actions/runs/43549877695) はsuccess**、6件版 efa4a8fcは別HEADのCI判定を要する。既存DBへRequired support列をいきなりADDせず、#4125のG0〜G5/Owner過去Coverageを確認。A2版PK/実Rowset/Current Grantは独立未実証。P0は変更しない。
- **11/14追加：Person Relation物理表現の選択肢（補遺§17）**：既存`persistence.Mapping.RelationStorage` はSourceAttribute→target DB-owned support columnを表現するため、Membership.personId直接属性を全Profile必須とはしない。Draft #4633に既存mapping.RelationStorageの欠落検査を強化し、仮CandidateのRelationStorage MappingがIR JSON往復で保たれる第5 Go testを追加。新HEADの実Go/Healthは別に再検証し、旧4テスト成功を自動承継しない。業務Owner #2386/#1126がRelation向き・業務Keyを決め、実DSL/SQL/DB接続は#2622/#4490へ委譲。第二Core/Host/新Schemaは導入しない。
- **11/14 C1の数値一致を検収完了としない（補遺§21）**：r7では同一validAt=2027-06-15でもknownAtが07-01→07-03で、Person3は変わらずMembership4→3、FTE3.0→2.4となる。Source→実保存→実Provider/Query→Coverage/Context→独立Consumer受領を別々に検証する。main統合#4653のConsumer横断設計、Draft #4647のpartial、Draft #4646のMeaning Pin/段階別Contract、Draft #4650のStudio生成／利用二巡を既存Ownerへ接続。PR #4633 HEAD 85f657の[Health #43572492328](https://github.com/hat47x/tei-platform/actions/runs/43572492328)は全Go/gofmt/vet成功、同HEADの[Windows #43572492248](https://github.com/hat47x/tei-platform/actions/runs/43572492248)はPowerShell OSVersion15秒timeoutで失敗。#4587別HEADのWindows成功とは混ぜない。C1の実Person Link/二時点Read/通常tei CLI/二Consumerは未実証、P0 #2586/#2831を変更しない。
- **11/14固定HEADで組織C1の実Go検証を確認（補遺§16.2）**：[Draft #4633](https://github.com/hat47x/tei-platform/pull/4633) b529791f の [Health #43548199759](https://github.com/hat47x/tei-platform/actions/runs/43548199759) は completed/success。ジョブログで全Goテスト（tei/internal/operations PASS）、gofmt、vetが成功し、skip無しの4つの組織GAP特性テストが実行対象に含まれた。**製品のPerson Relation・A2二版の実DB保存・Query→Consumer・Host承認が実装されたという意味ではない**。#4622は計画、#4633はGAPコードの別Draftのまま維持。次は#2386/#1126 Meaning選択後の正例実装を別PRへ切り出す。P0維持。
- **11/14初回の実Go GAP特性テストを分離（補遺§16）**：[Draft PR #4633](https://github.com/hat47x/tei-platform/pull/4633)に、実Workspace.LoadのMembership→Person未接続、現Persistenceの単独PKとPerson列欠落、as-of Result 3列でPerson IDなし、独立r7のA2二版/P2兼務を当初4 Go testとして作成し、§17でRelationStorageの別候補検査を加えて5件へ発展。既存#2611 [Closed #2632](https://github.com/hat47x/tei-platform/pull/2632)の7 Alias特性は重複実装せず参照する。これは**未解決を実証するCharacterizationであり、Product Conformanceの合格ではない**。後続のMeaning/Relation/DB版採用時には不在assertを正例テストへ置換し、CI exact HEADのGo/Healthを別途確認。通常CLI実ProviderとStudio Stage2を独立P0に維持する。
- **11/14追加・検証単位を固定（補遺§15）**：組織C1の未証明を増やさず、#2514のTQC/独立Expected、#1180の0件/Skip偽PASS・Suite世代、#2600の旧辺を含む増分再資格化を#4490 BEH/#2611 PM/#4501 CXへ適用。9つの既存受入圧力をSource/Typed Meaning/実Provider/Function Consumer/実Host Effectの5段階に割当て、Case実行状況・Observed Verdict・Claim資格を分離。初回P0実ReadはS0〜S3でよく、Org C1 HostやStage2 GUIを新たなHard Gateにしない。直近はR0実Sourceと独立Expected/実行Coverageの棚卸し、R1 Owner選択したPerson Link＋履歴DBと別Alias競合の局所試験、R2通常CLI実Provider→Context→Consumer/並行Desktop Stage2の二本柱。文書のみで実試験は未実施。
- **11/14 組織ID解決と複数DB Aliasへの引渡し（補遺§14）**：mainではMembershipにpersonId/Person RelationがないがEmployment/Affiliation側にはPerson Relationがある。雇用所属からの黙示Person同定・同名統合を禁じ、#2386/#1126にM-P（直接Person）、M-E（Employment経由の明示Key）、M-X（非Person subject）のProfile案を返す。#2611で指摘の`surface.go`はAlias名sort→同じOperationIDの後続Mappingを無比較でdropし得るため、履歴移行中のCurrent/History二DBに同一Semanticがある際は同一IDだけでTarget/Write先を確定しない（PM-01〜06を再利用）。組織計画§10.12を両Draftで重複作成した#4555(D0〜D4)と#4585(六Scope/B2C)は、両内容を保持して前方移植する案（§10.12 Desktop、§10.13 Scope）を提示、Owner merge前の書換えはしない。通常CLI実ReadとStage2セルフビルドを引き続き独立P0とする。
- **11/14実Sourceで確定したSemantic→DB→Queryの接合GAP（補遺§13、§17）**：org-staged-axisのorganization.Membership自体にpersonId/Person Relationが無く、現Persistenceには人物Linkと記録版PKのMappingがなく、as_of_membershipのQuery Resultは3列のみ。直接ColumnMappingに加え、既存RelationStorageMappingでSource側Relation→DB support columnを扱う方法も候補として比較する（IR能力≠実Parser/SQL/Providerの合格）。r7の合成SourceにはP1/P2/P3があるため、既存#2510のPerson3・Membership4→3・FTE3.0→2.4はProduct実Readではまだ証明できない。#1126/#2386の受理するKey/Relation→#2622の物理Mapping・履歴→#2586/#2506の実Read/Context/Claimの順に確認する。r7 Allocation basisPoints=100が1.0 FTEという**fixture固有のスケール**、SQL datetime-tzとの変換、replica/primaryの同じs7でもknownAtが異なるRead Cohortは別要件。#4621 Studio Web二Workspace×二Principal×二ProviderのHost Scope/CASは組織編集へ後段接続し、#2586実Readと#2831 Stage2セルフビルドを独立P0に維持する。
- **11/14追加：組織改編の公開/承認/引継ぎ境界（補遺§12）**：既存GoのDirective/Role/Delegation試験は構造と非権限化に限定。Proposed→Validated→Accepted→Committed→Published→Consumer受領/Handoff ACKを**別資格**で確認。部分更新、DB跨ぎの非原子性、旧新責任4象限、Commit不明・lookup-first・Grant版の再確認、Source/Selection ABA、将来Web Studioの競合編集を#4490 BEH-03/04/06・#4501 CX・#1129/#1174/#1444・#4515へ渡す。第二Org StateMachine/Global Tx/新Session Storeは作らない。#4623の限定Builder-selected実子照合は別のPlugin Build証拠でありDesktop Stage2達成ではない。#2586通常CLI実Readと#2831 Studioセルフビルドを独立P0に維持し、組織専用一括Txを先行Hard Gateにしない。
- **11/14追加：組織履歴の実DB互換移行（補遺§11）**：#1126の業務Occurrence ID≠記録版ID≠物理PKを、#4125の未採用G0〜G5移行検収観点と#4234生成AppのSchema Migration不足へ接続。旧現行行しか残っていないDBから、既知でない過去のknown版を自動生成せず、履歴Coverageをunknownとして残す。新規隔離DBのA2二版共存と、旧Reader/Writer/直SQL/Batch/互換View・書込Fenceを伴う旧DBの移行は別受入。実Migration成功/自動Startup DDLは採用しない。#2586通常CLI実Readと#2831 Stage2セルフビルドを独立P0で維持し、組織本番CutoverをHard Gate化しない。
- **11/14の実Source再照合（補遺§10）**：Draft #4567の実PG試験で使用した9行はr7のA2二版共存とは別fixture。Native Queryの出力3列にperson_idがなく、distinct Personや履歴Record Versionの正しい同定をQuery単独では証明できない。#1126/#4490で物理版識別、#2586で必要Relation/Person Read Cohort、#4501でCase責任/候補の局所Claimを検証する。#4616のmain Source-root Validation drift、#4579 Windows生成Workbench B0、#4597 Pin局所、#4623 Builder事前SHA、#4602 Hook作用不明はいずれも個別資格とし、Stage2/Host実Effect完了に合算しない。

- **実Query vs 履歴保存**：Draft #4567の固定HEADでは `as_of_membership` valid/known半開区間・物理Mapping・使い捨てPG限定Readが成功したが、現行 `membership_id` 単独PKは同一Occurrenceの複数known版共存を妨げる。まず#4566/#4567の重複を照合してから、#1126/DB Ownerの物理版識別案で両版保存→同一validAt/異knownAt→実Rowset/Read Cohort/完全性→r7独立Expectedの縦断を行う。SQL WHERE成功だけをBEH-01全体合格としない。
- **判断候補の確定性**：0/1/複数/partialの観測結果と必要Readの完全性・Rule/Case rev・Current Host Grantを分離。異動≠Case責任移管≠Subsystem Owner、通知≠ACK、旧Receipt/GrantのABA復活禁止を維持する。
- **TriggerとCase引継ぎを分ける（補遺§21.5）**：main統合済み[#4666](https://github.com/hat47x/tei-platform/pull/4666)の共通Lifecycle／Trigger／Invocation／Outcome／DeliveryをC1へ適用。A2所属の遡及訂正や通知受信は、Case責任・現在Grant・移管Effectを自動的に変更しない。validAt/knownAt、Rule評価時点、Effect/Receipt、必要なACKを独立確認し、重複通知・Grant失効・Commit unknown・ACK欠落を既存ORG-BEH/CXの負例へ割り当てる。組織専用Event Loop/Host/Workflow Engineを増やさない。 **組織の表示と編集（§21.6）**も、main統合[#4668](https://github.com/hat47x/tei-platform/pull/4668)の意味定義／実効計画／実Trace／図面を分けて扱う。Draft [#4667](https://github.com/hat47x/tei-platform/pull/4667)の意味対応・投影契約を参考に、r7のP2兼務A2/A3をTree/Graph/Tableで表示しても人物／所属／記録版の同一性を失わず、集計や曖昧なTreeからの逆編集は許可しない。Source Pin／改版と独立Consumerの検収は既存Ownerへ委譲する。
- **他チャットの横断差分**：DSL自己記述/Source編集、三要素明示CRUD-onlyのmain修正 #4551、Studio/Stage2セルフビルド、Plugin六ScopeとB2C Headless、AI固定対象/SHO再投影、#2582の問い相対Required影響、#2516 Disclosure、#2600 Build/Evidence資格を同一C1の**別々のWitness**へ対応付ける。通常組織Dataへ一律ACLを追加しない。
- **優先・競合処理**：#2586 通常CLI実Provider Readと#1824/#2831 Stage2セルフビルドを独立P0で維持。組織段階計画§10.12へ並行追記するDraft #4555（Desktop）と#4585（六Scope）は、片方の全文上書きや暗黙優先で扱わず前方統合・節番号整理を検討する。#4566/#4567も実Queryの重複変更を無条件に両方採用しない。現補遺はdocs-onlyで新実試験/Owner受理はNOT RUN。



<!-- org-c1-crosslane-graph-plugin-integration-20261114 -->
### 2026-11-14 横断追補：組織C1を標準グラフ・実行モデル・Plugin公開契約の検証対象にする（補遺§24）

[組織C1横断受入補遺 §24](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)に、他チャットの成熟した設計を**既存Meaning/Graph/Query/Plugin/Hostへ差分統合**した。新しいCore/Compiler/第二Expected/専用Hostは設けない。

- **四つを分ける**：Person/Membership/Caseの業務関係Graph、Rule/Providerを選んだ実効Plan、actual SQL/Read Cohort/Receiptを含むObserved Trace、Studio/SHOのTree/Graph/Table/説明図。main統合済み[実行モデル #4668](https://github.com/hat47x/tei-platform/pull/4668)と[起動モデル #4666](https://github.com/hat47x/tei-platform/pull/4666)へ接続し、図面だけで実Actionを主張しない。
- **構造／評価／変更の意味保存を別検証**：P2のA2/A3兼務とA2旧新Record Version、同名Node・多重Edge・hidden parent、r7の人物3→3／Membership4→3／FTE3.0→2.4、NULL/partial/NOT EXISTS、Source Pin/ABA/非単射Viewの逆編集拒否を、既存#2497/#4490/#4501の独立Expectedへ割り当てる。Draft #4667の新文法・Semantic Engineは未採用。
- **Plugin横断拡張は公開契約だけを参照**：上位Pluginが公開した意味Graph/Hookを版・ID・Target・Scopeで選択し、衝突・順序・除外条件を検査する。Composition成功からLoad/Grant/Effectへ昇格しない。最初はC1 read-only候補のProvenance説明という非Effect Hook一箇所。B2Cの計測等も同じ設計を応用するが、同意前送出や一律Web Runtime強制は認めない（#951/#2516、Draft #4646）。
- **REAの二義を分離**：Resource–Event–Agentは業務分析レンズ、Reverse Engineer AnythingはSource付きの解析候補生成手段。Agent≠Person≠Principal、解析候補≠Owner受理。三要素分析法/SHO/SUI/AI固定対象は各OwnerのProjection/Disclosure/再資格化を使用し、Source/Rule変更時に旧提案を自動Applyしない。
- **次の実装単位はOG-0→OG-5**：C1の四グラフとSource対照→Headless候補0/1/複数/partialの実Rule評価→Person/履歴の実Provider Read→Loss-aware投影と公開Plugin Hook→Host責任移管/Receipt/ACK→定義・生成・Plugin自己記述の重複削減実測。OGは組織の補助順であり、DB移行のG0〜G5や全体P0の新Gateではない。

**今回の状態**：計画・補遺の文書更新のみ。組織C1の実Provider/Host/Studio/Plugin/AI E2E、Owner Meaning Acceptanceは未実行・未受理。通常tei CLIの初回実Provider Read **#2586** とStudio Stage1→Stage2セルフビルド **#2831** を独立した最優先課題として維持する。


<!-- org-c1-cli-first-plugin-docs-20261114 -->
### 2026-11-14 組織C1の次の実装を、候補Ruleの実評価とCLI-firstの可読性へ絞る（補遺§25）

**受入と実装の正本**：[組織C1横断受入補遺§25](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)。組織モデリングに新たな機構を増やさず、独立Relation Occurrence/Record Version/実Rowsetを共有する標準意味モデルの製品検収へ進む。main統合済みの[構造・評価・変更対応 #4667](https://github.com/hat47x/tei-platform/pull/4667)、[公開Facet #4677](https://github.com/hat47x/tei-platform/pull/4677)、[CLI-firstセルフビルド #4676](https://github.com/hat47x/tei-platform/pull/4676)、[dogfooding #4680](https://github.com/hat47x/tei-platform/pull/4680)をそれぞれの設計Ownerへ参照する。組織固有の第二Core/Resolver/Policy/Runnerは不要。

- **OI-0 受理前の識別を保全**：Person/Membership/Version/Physical Row/Case/Responsibility、Graph Edge、識別関係と参照関係を区別。渡辺式ER図の左親・右子、複合キー継承と通常FKの線の違いは、元の関係型・キーを保持した用途別投影にする。Mermaid/Markdown等は原文を保持する糖衣形式であり、図の配置が意味を決めない（#2386/#1126/#2497）。
- **OI-1 候補解決の次の小さな実装**：#4490の先行BEH-01/02（実時点差分・必要Read/非出力依存・取得完全性）を尊重し、WU-09/ORG-BEH-04の候補Ruleを0/1/複数/partialで評価する。同じ出力1件でもcompleteとpartialでは結論資格が異なることを独立Expectedで検査。現SourceのRoleAssignment（Person/Role）とResponsibility（Person/OrgUnit）は確認済みだが、Case C1の正式関係や承認Ruleは**未採用**。当初は合成条件に限定し、実Reader/Rule/Provider到達・候補の由来・実行Grantを別資格で扱う（#4490/#2386/#2510/#1180）。
- **OI-2/OI-3**：同一A2旧新履歴とPerson Linkの実DB/Query→Context→独立Consumerを§21〜22の未達のまま別資格で実証。Graph/Tree/TableのSlot別Source逆写像とEdit/Read-only判定を分け、hidden EdgeやAggregateへ書戻さない。内容がA→B→Aに戻ってもOwner Decision・Current Grant・実受領は復活しない（#2586/#2622/#2497/#4515）。
- **OI-4 文書を共通入口にする**：[Plugin文書探索Owner #4679](https://github.com/hat47x/tei-platform/issues/4679)の公開Contract/Graph/上位Manual/版/Source/開示Scopeを、組織Profileの公開文書で試す。短いTEI Product Skillから**実在するCLIのread-only Discovery**で辿り、必要なManualを遅延取得。未実装plugin docs等を製品コマンドとして書かない。Studioは同じ操作の別Projection、CLI/ローカルServer/遠隔ServerはHostの別Bindingであり、WailsをCLIの必須依存にしない（#4376/#2605/#951）。
- **OI-5 TEI-nativeの生成実績を別計測**：main統合済みCLI-firstのGraph最小fixture（同Label別Node、並行有向Edge、逆向きEdge）で生成Goの実処理→Plugin Actual Call→Consumerを確かめ、その後に組織の関係Reader/Ruleへ適用。Wrapper-onlyや単純Go委譲をTEI実装率に算入しない。意味実装割合、Build制御、次Stage再利用、手書き重複減少は別指標（#4676/#4680/#2831）。

**資格・優先順位**：§25と本項目はdocs-onlyで、OG/OIの製品試験は未実施。正本は既存#4490/#4501/#2510のExpectedと既存Source/Build/Plugin契約。全TEIの**通常tei CLIの最初の実Provider Read #2586とDesktop Stage1→Stage2 #2831は独立P0**のまま維持し、組織C1の高度機能・プラグイン文書・GUIを新Hard Gateとしない。


<!-- org-c1-oi1-actual-reader-acceptance-20261114 -->
### 2026-11-14 高度組織C1 OI-1の実装引渡し（補遺§26）

[横断受入補遺§26](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)に、mainの実SourceとGo構造テストを照合した候補解決の正負ケースCR-00〜08を追加した。Person/RoleのRoleAssignmentとPerson/OrgUnitのResponsibilityを**同一Person IDで突合する合成read-only Rule**は、業務Ownerが承認・Caseとの正式な意味を受理したわけではない。SourceにないCase→Responsibility・RuleやRoleAssignment/Responsibilityの二時点を捏造しない。

**順序の補正**：#4490が先に定めたBEH-01/02の二時点結果差分・非出力依存/取得完全性を保持する。OI-1は合成Referenceでの候補評価を並行できるが、製品のQuery/Provider/Rule→Consumer合格は実Read Cohortと必要Readの証拠が揃ってから判定する。重要な反例は、可視の人物1件が同じでも完全Readならcomplete-one、RLS/Limit/欠測で範囲不明ならincomplete-or-unknownとなること、候補があってもHost Grantは生まれないこと、二経路のPerson重複を根拠Pathごと保持することである。

**他レーンの再利用**：main統合済みApplication Graph #4681、Plugin共通文書Issue #4679とmain統合済み #4689、CLI in-process/Studio子Serverの配置候補Draft #4688/#4690、CLI-first自己生成 #4676/#4680へ接続する。製品入口や文書探索を模倣した第二Runtime/Doc Registry/Resolverは作らない。通常tei CLI実Provider Read #2586とDesktopセルフビルド #2831の独立P0は不変。

**資格**：§26・本段落はdocs-onlyの実装引渡しであり、今回の実Rule、Go/PG、CLI、Host、Plugin文書取得、Windows/Studio実試験、Owner採用はNOT RUN / NOT ACCEPTED。


<!-- org-c1-change-impact-multi-axis-20261115 -->
### 2026-11-15 高度組織モデル：異動・職位・役割・責任を別Relationとして解決する（補遺§27）

**実装契約**：[組織C1横断受入補遺§27](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)に、現行 `org-staged-axis` 実Sourceと[main統合済みの構造・評価・変更同値設計 #4667](https://github.com/hat47x/tei-platform/pull/4667)を対応付けた。**Membership・Occupancy・RoleAssignment・Responsibility・Directive・Caseを単一の所属・権限状態へ縮約しない**。所属・職位は二時点属性を持つが、現在のRoleAssignment/Responsibilityはその二時点列とCase C1関係を持たない。Membership→PersonやOccupancy→Personの正式なRelationも現Sourceでは欠けるため、未採用の人物解決経路を創作しない。

**既存Ownerと次の着手**：
- **#4490／#2386／#1126**：先行ORG-BEH-01/02の実時点・非出力依存・Read Completenessを保ち、次にOI-1/CR-00〜08の候補実Reader/Rule、さらに本補遺ORX-1〜3で「必要Relationの欠落」「職位占有人物の未解決」「時点を持たないRole/Responsibilityのas-of偽合格」を反証する。DomainごとのPerson/Position/Subject対応とDuty/Case意味はOwnerが採用したProfileだけに限定。
- **#4490／#4501／#1174／#1444**：異動・役職交替・責任変更・Directive発効・Case移管を別の変更Intent/Revision/Effectとし、旧新責任の4象限、Current Grant、実Receipt/ACKを独立検証。r7のFTE・A2訂正からCase C1や実権限を推論しない（ORX-4〜6）。
- **#2497／#1124／#2582／#2600**：複数親のOrgUnit Graph、渡辺式ER図の**親左・子右と識別／非識別線の区別**、Mermaid直接Sourceの対応可能範囲とLoss-aware Viewを再利用。非出力依存も含む問い相対の再資格化、無関係な変更の過剰失効防止、A→B→Aでの旧受理・旧Effect復活禁止を確認する（ORX-7〜8）。
- **#4681／#4679／#4689／#2831**：業務関係GraphとEntry/Function/Providerの実効計画・Observed Traceを分離。CLI/Studio/AIは同一Source/Meaning/Plugin版のread-only根拠を共有し、Plugin文書参照を権限や実Runの証拠にしない。#4689のPlugin生成・文書の設計はmain統合済みだが、実Help/生成Pluginの適合は別受入。

**導入順**：既存CRの純粋候補試験→ORX-1〜3の実Relation Path/時点の欠落→ORX-4〜6の責任・Grant非連動→ORX-7〜8の再資格化/図面保持。すべて既存ORG-BEH/CX/SMと#1180/#2514のIndependent Expectedへ割当て、組織専用新Runnerや第二Expectedを作らない。#2586の通常tei CLI初回実Provider Read、#2831のStudio Stage1→Stage2を独立P0で継続し、Org高度対応をHard Gateにしない。

**受入上限**：本項と§27は設計・担当引渡しのみ。正式Person/Occupancy/Case Link、actual PG二時点版、実候補Reader/Rule、Plugin/Studio/Host Product E2E、Owner Meaningは**NOT RUN / NOT ACCEPTED**。参照PRのmain統合は製品実証の成功を意味しない。


<!-- org-c1-crosschannel-three-aspect-candidate-20261115 -->
### 2026-11-15 組織モデルの実務入力を、三要素分析法とAI候補の受理経路へ接続する（補遺§28）

[組織C1横断受入補遺§28](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)に、**白板写真、会議の文字起こし、CSV/Excel、帳票、既存画面・DB、REA等の外部解析、Markdown/Mermaid直接Source**を、入力ごとにRaw/出典位置・取得Scope/版を保つ**Candidate入口**として整理した。組織専用OCR/Transcript Engine、別Meaning正本、第二Runnerを作らず、既存Source/三視点Data・Function・Business/候補Review/Shared Acceptanceに接続する。

- **意味の同定（#2386/#1120/#1126、#4490）**：例として白板の「D1の部門長→案件C1」、会議の「P2異動・引継ぎ協議」、CSVの「P2二行」は同じSource/Person/Relationを指すと証明されていない。候補としてSourceと対応付け、Person/Occupancy/Role/Duty/Caseの正式なKey/時点/Rule/Grantが不明なら`unresolved`として停止。歴史を語る発言の収録時刻を`knownAt`に流用しない。
- **Source表現（#2621/#4370/#2497、#4704/#4709）**：`*.tei.md`は通常Markdownでも読めることを要件とする。階層はインデントリスト、関係/規則はMarkdown表、直接表現可能な図はMermaid ER/Sequence、外部データは版付きCSV参照。Mermaid/渡辺式の多重性・識別/非識別・Source spanを保全。main統合#4709のGraph Fixtureは正式Reader/Go生成の完了ではなく、現在の`TEI_MD_PARSER_UNBOUND`を維持する。
- **外部候補（#3394、Draft #4693）**：REAのResource–Event–Agentという業務意味とReverse Engineer Anythingという解析チャネルを区別。R2の一Observation/Relation、reacandidateのAttempt全体、Semantic GraphとApplication Graphの異なるID、Raw BOM/Digest差、Candidate IDと正式Relation IDを混同しない。出典付きRelation同定だけを候補化する。
- **一往復の検収（#3262/#2516/#4515/#1124、#1180/#2514）**：公開合成Worldと一Raw資料→抽出Candidate→同名・多重・時点・partialの照合→CLI/Studio/SHO/AIの**read-only根拠付き表示**までを最小Consumerとする。Owner採否・Source Patch、実Query/生成Go/Plugin Call、Host Grant/Effectは後続の**独立した資格**。OCI-1〜8を既存ORG-BEH/CX、REA J、SM、TM/MDのExpectedへ割当て、新Scenario正本を増設しない。

**実施順・主線**：①#4490 BEH-01/02の実Read完全性→②§26 CR・§27 ORXの対象解決→③OCI-1/2/4の公開合成Candidate照合→④既存#4693/#4704/#4709による外部解析・自然Markdown Readerの適格性→⑤Owner採否/Source/実Consumer。通常`tei` CLI実Provider Read #2586とStudio Stage1→Stage2 #2831を独立P0として先行。**本追補はdocs-only**。画像・会議・CSVの実AI解析、正式`*.tei.md` Reader、実Go生成、実Provider/Host/Windows E2E、業務Ownerの意味受理は未実施。


<!-- org-c1-brms-actual-coverage-20261115 -->
### 2026-11-15 組織C1の候補判断を決定表・Rule生成・実Read完全性へ接続する（補遺§29）

[組織C1横断受入補遺§29](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)に、[main統合済みのBRMS設計PR #4765](https://github.com/hat47x/tei-platform/pull/4765)を、**現実の組織判断に適用する最初の小ケース**として受入仕様化した。BRMSは**設計文書がmainに統合済み、製品機能・意味受理は別検証**であり、組織専用Engineや正式Decision Table Schemaは採用しない。

- **モデルとRule**：合成D1/R1に対し、RoleAssignment(Person/Role)とResponsibility(Person/OrgUnit)が同じPerson IDを指す場合に限って候補とする。Role/Duty/Case/valid/knownの正式意味、Current Grantは別。Rule表OR-D01〜05は**説明用**であり、表示表の単純unique-hitや行順で人物・権限を決定しない。対象Person単位の結合・distinct Person・Relation Occurrence Path・No Matchと複数候補を区別する。
- **実Readerの限界と優先反例**：現`internal/functioncontext/functioncontext.go`は`Rowset.FullyRead=true`と`CoverageUnestablished`/`EmptyJustified=false`を同時に保持する。可視1件でも必要Readがpartial/unestablishedなら「唯一」を判定しない。非出力依存、RLS/Limit/Join/Scopeを必要集合へ含める。最重要反例はODR-3＝同じP1可視行でcomplete-oneとunknownの分離。
- **実Codegenへの引渡し**：main統合済み[限定Markdown Graph Reader #4722](https://github.com/hat47x/tei-platform/pull/4722)に加え、[Graph Rule→Go実分岐→別Go Process実行 #4729](https://github.com/hat47x/tei-platform/pull/4729)がReferenceで成立。ただし正式`*.tei.md` Loader/Meaning/Target Generator/Plugin/通常CLI、**組織の二Relation Join・候補の完全性**には未接続。採用済みRuleの狭い条件をSource→型付き式→実Go判断→独立Consumer/Expectedへ広げるS1〜S4の残差を分離し、Go Delegate-onlyをTEI実装へ計上しない。
- **Ownerと試験**：#4490のBEH-01/02（時点・非出力依存・取得完全性）を先に検収し、CR-00〜08とODR-1〜10の合成判断／競合／NULL/欠測／RoleとGrant分離を既存#1180/#2514へ追加する。BRMS一般分析の採否は#988/#1637/#1129/PR #4765（旧Draft #4723）、Source/Generator #2621/#1121/#951、実Provider/FunctionContext #2586/#4495。新Expected Storeや横断Gateは作らない。

**P0不変**：Studio Stage1→Stage2 #2831/#4583と通常製品CLI実Provider Read #2586/#1251を優先し、組織決定表の実装をBlockerにしない。**本項はDocs-only**。実Rowset/Rule、生成Go、CLI、Plugin、Host/Windows、Owner採用はNOT RUN/NOT ACCEPTED。


<!-- org-c1-candidate-proof-matrix-20261115 -->
### 2026-11-15 高度組織C1：候補の正の証拠と全件性を2軸で判定する（補遺§30）

[組織C1横断受入補遺§30](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)に、**「見えた人物の一致経路」と「必要なRelation全件を閉じて読めたか」**を分ける証明可能性マトリクスを追加。Current `functioncontext.From`のCursor `FullyRead=true`は依然`CoverageUnestablished`／`EmptyJustified=false`であり、完全性の証明ではない。

- **partialの0人/1人**：観測集合だけでは「該当なし」「唯一」を宣言できない。ただしP1の二Relation OccurrenceとPerson Key・時点・Scopeが実際に確認できれば「正の一致経路を観測した」は別に残せる。
- **partialの2人以上**：同定済みの異なるP1/P2の正の経路が揃えば、**唯一性は反証できる**。ただし完全な候補数・候補集合は確定できず、unresolved Personや未取得の否定条件がある場合は適格性を昇格しない。複数Pathで同じP1へ到達してもPersonは1人。
- **complete**：Scope、RoleAssignment・ResponsibilityのRequired Set、Person同定、RLS/Limit/Query非出力依存・時点・Read Cohortが閉じた場合のみ、0人/1人/複数の**全件集合として**結論を出せる。現行Native SQLだけでcompleteを表明するのは不可。
- **実装2段**：#4490 BEH-01/02を維持して、PR-AはCR/ODRの観測Path・Read閉包・独立Expectedを小さなread-only Ruleで評価。PR-Bは[#4729](https://github.com/hat47x/tei-platform/pull/4729)の一辺Graph Rule実Go生成を、正式に採用した型付きRelation/Decisionへ拡張できるか検証し、**二Relation Joinはまだ実装されていない**と区別する。既存#2621/#3030が正式Reader、#2586/#4495が実Provider、#1180/#2514がExpected、#4723/#988がDecision分析を担当。新Engine/Expected Storeなし。

**受入上限**：§30は設計と引渡し。main統合#4729でSource byte span付きGraph Ruleの実Go条件生成と別Go Processの限定実績はあるが、正式Loader・組織候補Join・実Org PG/Consumer/Host・Owner意味受理は本PRで**NOT RUN/NOT ACCEPTED**。通常製品CLI実Provider Read #2586とStudio Stage1→Stage2 #2831のP0は不変。


<!-- org-go-witness-qualification-20261115 -->
**2026-11-15 §30の初回コード実験へ着手**：[Go Test Draft #4737](https://github.com/hat47x/tei-platform/pull/4737)で既存`internal/operations`に実Workspace LoaderのRoleAssignment/Responsibility RelationKey、合成D1/R1の0/1/複数・partial・Read Cohort・Person Occurrenceの10独立リテラルExpectedを追加。さらに合成BrowseQueryの**RoleAssignmentとResponsibilityを別々の実`functioncontext.From`へ通し**、二つのContextからテスト用ConsumerがRelation ID／Person ID／Role・OrgUnit・NULLを取得して正の候補経路を照合する。両QueryのSynthetic ID／Source Digestは区別し、同一Read Cohortは**未実証**と明記する。[補遺§30.5](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)へ実ファイル/受入上限を記録した。実装は**テスト限定の正の証拠/全件性Probeと実Function Context投影**であり、正式C1承認Rule/Reader/実PG Provider・二Relationの実Context Join・Host GrantやTEI-native Go生成ではない。初期HEAD `99341c5d`の[Repository Health #43639067283](https://github.com/hat47x/tei-platform/actions/runs/43639067283)と後続HEAD `61513ae2`の[Health #43641147796](https://github.com/hat47x/tei-platform/actions/runs/43641147796)は**failure**。両者とも追加Goと当時のmainのgofmtドリフト、別領域のSQL/定数検査に問題があり、追加Goを整形・現行mainへ追随中。テスト単体の合否や最新HEADの資格には転用せず、再実行の正確なコミットとログで別途判定する。

**次の差分**：#4490 BEH-01/02の実二時点/非出力依存・Read cohort/Completenessを先行させ、今回のGo Probeの判別力を実Query→Function Context→実Consumerへ接続する。Graph/Rule→Go分岐のmain統合 #4729、同一Generated BinaryのCaller入力 #4733、管理Markdown Source byte/spanとIndexの整合・変更失効 #4751は有効な**それぞれ異なるReference成果**。次はそれらのSource/Rule/Artifact証拠を組織の二Query/Function Contextへ実際に束縛する。いずれも現段階で、組織の正式二Relation Join・実Provider Read cohort/Case Ruleを生成・検収済みとはしない。P0 #2586/#2831、Expected Owner #1180/#2514、Meaning Owner #2386の優先順位と境界を維持する。

<!-- org-cohort-proof-plan-20261115 -->
### 2026-11-15 組織C1：二Contextの同一性・必要Readを実Providerで証明する（受入補遺§31）

[組織C1補遺§31](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)で、実`functioncontext.From`を二回通す[Go Draft #4737](https://github.com/hat47x/tei-platform/pull/4737)から製品の候補判断へ進む際の**未証明の接続点**を固定する。二QueryのSource DigestやCursor読了は、同一validAt/knownAt、同一Read Snapshot、Scope/RLS/Limit、必要な否定依存のRead完了を意味しない。**正のPerson Pathが観測された事実、全件候補としての判断、Case責任・Host Current Grantは別資格**である。

- **最小実装順 OCP-0〜4**：#4737の最新HEADで合成Go回帰資格を確定→#4490のBEH-01/02（二時点・非出力依存・Coverage）→#2586/#4495の選択実Provider/SQL/Function Context/独立Consumerを一つのAttemptで照合→既存TEI-native Graph Rule Go生成 #4729/#4733の正式組織Relation/RuleへのFit/Gap→#4501/#1174のCase Revision・現在Grant・Effectと#2516/#1124の閲覧投影へ別ゲートで送達。単一Query方式と二Query同一Snapshot方式を比較し、根拠のないCohort同定は`unestablished`として残す。
- **OCC-01〜10の負例**：complete 0/1とpartial 0/1、partialでも異人物の確認済み正Path、同Person複数Occurrence、NULL Person Key、Snapshot/時点/Scopeのズレ、RLS/Limit/否定依存未取得、r7のknownAt差、Case移管/ABA/旧Grantを既存#4490/#4501/#1180/#2514の独立Expectedへ配置する。新しいScenario/Expected正本は作らない。
- **明示的な未達**：現在のRoleAssignment/Responsibilityの実Product Contextは**合成入力**、`test-fixture-cohort`は試験仮定。実組織Provider/同一Read Cohort、正式Person/Case意味受理、二時点DB共存、正式`*.tei.md` Reader→生成Go→通常CLI、Host Effect、Studio/SHO一巡は別実証。Source Index #4751を両Queryの受理済みSource Pinに自動昇格しない。
- **Owner/禁止事項**：組織意味の採否は#2386、Rule/Readは#4490、Product Providerは#2586/#4495、生成は#2621/#1121、Case/Effectは#4501/#1174、Expectedは#1180/#2514。組織専用Core/Resolver/Rule Runtime/Host/二重Evidence基盤は増設しない。

**P0維持**：通常製品`tei` CLIの実Provider Read #2586とStudio Stage1→Stage2セルフビルド #2831を優先。§31は**計画文書のみ**で、新Go/PG/Windows/Host/CLI/Studio E2E、Owner Meaning AcceptanceはNOT RUN/NOT ACCEPTED。


<!-- org-c1-brms-accepted-source-product-line-20261115 -->
### 2026-11-15 組織C1：BRMS・業務説明・Mock・生成コードを同じAccepted Sourceからつなぐ（補遺§32）

[組織C1補遺§32](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)へ、[main統合済みのBRMS設計 #4765](https://github.com/hat47x/tei-platform/pull/4765)の有限Unique／部分観測の判断、ほぼ直訳可能な業務説明、契約付きMockシミュレーションと実Go生成を接続。Person/RoleAssignment/Responsibility・時点・Rule・Required Readの意味を共有し、**同一Sourceから説明／テスト／コードを派生させる**ことを最小実験の目的とする。

- **組織固有の受入**：正のRelation一致と全候補の閉包、業務適格性、Case責任、Current Grant/Effectを別資格とする。二Context合成Go Draft #4737は実Provider/同一Snapshotの実証ではなく、#4490 BEH-01/02と§31 OCC-01〜10を先に接続する。r7のPerson Link/二時点DB共存、C1のCase意味採用は#2386 Ownerへ明示的に返す。
- **段階実証**：採用済みRule一件について正式 `*.tei.md` Reader（#2621/#988）→型付きMeaning／既存Generator（#1121/#4544）→生成Goの実分岐と独立Expected（#1180/#2514）→実Query/Context/独立Consumer（#2586/#4495）→説明とMockの同値を試す。Sourceの原byte/spanと候補構造指紋、生成Artifact、実Read Cohort、Consumer開示を別票に記録。Graph Reference #4729/#4733、Pinned Source予備試験 #4751の局所実績を製品資格へ流用しない。
- **編集・影響**：所属・Role・責任Occurrenceの多重性と二時点をTree/Graph/Tableで保存し、曖昧な投影から自動で逆編集しない。説明のみの変更／Rule意味変更／Generator版変更／Plugin Lock・現在Grant変更を別に再評価し、A→B→Aで旧Receiptを復活させない。Go delegateだけの自己生成は完了に数えない。
- **責務境界**：意味#2386、Read/候補#4490、BRMS設計#4765、生成#2621/#1121、Expected#1180/#2514、Case/Host#4501/#1174、Studio/SHO#1124/#2516。第二Rule/Simulation Engine、第二Source/Expected/Scenario正本、組織専用Hostは追加しない。

**実行状況**：組織C1の正式Reader→生成Go→実Provider/CLI→二Consumer/Host→Studio一巡、Owner Meaning Acceptanceは未達。今回の追記は**設計文書のみ**。優先P0の通常`tei` CLI実Read #2586とStage1→Stage2 Desktop #2831は独立維持する。

## 2026-11-15 宣言的構文の汎化を製品の受入へ接続する（設計案・実装未受理）

**方針**：TEIで汎化するのは一つの巨大な文法ではなく、Data／Function／Businessと標準データ構造、Relation／Rule／Query／Operationへつながる**意味の契約**である。Markdown、表、Mermaid、式、関数チェーン、Native SQLは用途別の記述形式として残す。文法の違いを理由に第二のSemantic Engine／Parser選択基盤／汎用Host／Rule Runtime／Expected正本を作らない。構文の正式採否と式の範囲は[#4359](https://github.com/hat47x/tei-platform/issues/4359)のS0比較・Owner判断に帰属し、MarkdownのSource選択・Role解釈は[#2621](https://github.com/hat47x/tei-platform/issues/2621)／[#1018](https://github.com/hat47x/tei-platform/issues/1018)へ戻す。

**区別する五つの資格**：

1. **Source**：選択された原文byte・Revision・Source span・Language/Profile／文書内Roleを固定する。単なる`*.tei.md`という拡張子、Manifestへの登録、Source Indexへの格納はReader成功や意味受理ではない。
2. **Meaning candidate / Accepted Meaning**：表／図／式／Ruleが既存の型・参照・粒度・単位・NULL／未取得・空集合／partial policyへ写せるか確認する。AI候補やParser成功をOwnerの受理へ昇格させない。同じ形のGraph／Table／Queryでも評価と変更の意味が一致するとは限らない。
3. **Generation**：Accepted Meaningと固定Operator／Generator／Targetの契約から、条件分岐・Data Operation・制約・UI・Plugin等の実装に寄与する生成物を作る。Goコードへ単純に処理を委譲しただけの場合、自己記述・セルフビルドの達成とはしない。
4. **Execution**：Definition／Rule／Edge／Generator版の変更と、同じ生成Binaryへ渡す実行時入力の変更を分ける。Providerは選択・組込み・実呼出し・認可・Effect観測を別資格で照合し、CLI／MCP／Studioや外部Hostの成功をReferenceの成功から推定しない。
5. **Provenance / Reuse**：意味の同値、Source byte位置、World Fingerprint、Model Revision、Artifact Revision、実行入力、現在のEvidence／Grant資格を混同しない。Mermaidの配置だけの変更とRuleの変更を区別し、誤再利用と不要な再生成の両方を検査する。

**直近の実装順（既存P0を置き換えない）**：

| 順序 | 接続する実装・検査 | 主なOwner・受入 |
| --- | --- | --- |
| 1 | [#4359](https://github.com/hat47x/tei-platform/issues/4359) S0の旧`validation`／低ノイズ`rule`／宣言Role付きMarkdown表／Operator pinを、同じ二規則・独立Expectedで比較する。S1以降の実装は正式構文をOwnerが選んでから進める | #4359、#988、#2514 |
| 2 | **内蔵Rule Markdown Readerの製品接続を基準に、Graph固有S0-Aを実装する**。mainの`tei.rule-authoring/v0`はManifestの明示Language選択→同一`Workspace.Load`原byte→既存`semantic.Validation`／Source Index Spanまで実装済み。GraphのNode/Edge ID・向き・多重性を内蔵Ruleへ読み替えず、正確なLanguage選抜A0→同一LoadのA1→出典A2→負例・互換A3で個別検収。外部Language Pluginの実Lockは別資格 | #2621、#988、#1018、[内蔵実装コミット](https://github.com/hat47x/tei-platform/commit/7921b005a44d325983d24ca2b8b58b0737dd778d)、[データ様式§39.10](design/SEMANTIC_STRUCTURE_MAPPING_AND_DECLARATIVE_SYNTAX_VIEW_2026-11.md) |
| 3 | 実Graph／RuleのSourceから、受理済み意味→既存Generator→生成Go／Pluginを接続し、二Consumerと独立Expectedで正負を照合する | #1121、#1251、#2586、#1180 |
| 4 | 生成された構造・制約・Pluginを次のBuilderが本当に利用したか追跡し、CLI先行→Studio Stage1→Stage2・Windows実Consumerへ進める | #2831、#4583 |
| 5 | 上の縦断で得たFit／Gapを、Graph／Tree／Table、BRMS、Query／Native SQL、Data Context、数理Toolkit、Studio／SHO／SUIの表示・限定Patchへ戻す。表示・逆編集・実Providerの資格はそれぞれ別に判定する | #2495、#2497、#1124、#4660 |

**判別すべき負例**：NULLと未取得の混同、空集合とpartialの混同、同名Node・多重Edgeの消失、曖昧参照の先勝ち、同一Rule IDの二重正本、説明用Code Fenceの誤実行、SQLのBag／NULL／Join差の消去、Mermaid配置変更による不要な再生成、Rule変更後の旧生成Binary再利用、AI候補の無受理実行、未選択Pluginの呼出し、生成物未使用の見かけ上のセルフビルド成功。独立ExpectedとScenarioは既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)の正本へ渡す。

<!-- data-forms-builtin-rule-reader-plan-20261115 -->
**製品実装と異種データの圧力試験を分ける**：内蔵Rule Sourceの`tei`定義Fence・Namespace・既存Validation・Source Spanは、現在の製品Loaderで成立する一限定経路である。Graph Node／Edgeと外部Language Plugin、実Generator/Builder、通常CLI/Provider/Stage2のOwner受入へ自動昇格しない。既存業務機能の比較で挙がった共有物理表・複合キー、組織C1の同一Membership訂正版を単独PKが`23505`で拒否する実PG負例を[データ様式§39.10](design/SEMANTIC_STRUCTURE_MAPPING_AND_DECLARATIVE_SYNTAX_VIEW_2026-11.md)へ引き渡す。物理キーが一致しても業務Occurrenceや履歴版を特定したとは扱わず、MVR-02の旧新版共存・二時点再Readは#2386/#1126で受理するまで未達。第一の製品Graph Source一件、通常CLI実Provider、非Plugin Stage1→Stage2のP0は変更しない。

<!-- data-forms-identity-asof-inspector-plan-20261115 -->
**実装済みの異質な反証への引渡し**：標準データ様式[§39.9〜39.10](design/SEMANTIC_STRUCTURE_MAPPING_AND_DECLARATIVE_SYNTAX_VIEW_2026-11.md)では、(1)実`InspectSubject`のCanonical/元Source byte位置の一致とS-OPEN未実施（main #4853）、(2)REAの構造キー一致が複数ある場合は個別Relationの対応を捏造しない（main #4850）、(3)組織二時点Queryの評価入力`valid_at/known_at`と物理区間列`valid_from/known_from`を混同せずBinderの旧名呼出しを拒否（main #4847）、(4)ReferenceのGraph読取とProduct Language未束縛拒否を別に観測、という四つの局所Go検査を次Ownerへ渡す。個別試験の存在・成功と、正式`*.tei.md` Reader／実Read Cohort／Source編集許可／Wails実Consumerの製品受入を分け、#2621/#988・#2497/#1124・#2586/#2506・#1824/#4583へ既存Expectedで引き継ぐ。これらを初回Graph S0-Aの新しいHard Gateにはしない。

<!-- data-forms-product-rule-meaning-evaluator-plan-20261115 -->
**内蔵RuleのR1局所生成からR2実Builder消費へ進む**：[データ様式§39.11〜39.12](design/SEMANTIC_STRUCTURE_MAPPING_AND_DECLARATIVE_SYNTAX_VIEW_2026-11.md)では、旧`validation`とMarkdown`rule`の型付き意味同値（main #4900）、既存Evaluatorの合成入力八例のRead Footprint（main #4901）、Rule原文の参照先変更によるEvaluatorの追随（main #4908/#4913）と、`validationgen.GeneratePositive`による**`std.decimal`正値判断のGo生成・別Process実行**（main #4911）を、それぞれ別の達成として記録する。次は、隔離Fixture内で比較前に二つの`std.decimal`属性を定義し、Ruleの参照先一箇所だけを変更した場合の再Load→型解決→生成Go／Read先→実結果の変更追随を確認する。現Fixtureの`line_number`は`std.integer`なので、これへの変更は**現Generatorでは拒否すべき負例**であり、生成成功の例に転用しない。R2では、実Builderが当該生成Goのbyteを消費した証拠と外部測定Artifact・独立Consumerを同一Attemptで照合する（#1121/#4544/#1251）。その後に通常CLI／実Provider・Input Coverageへ進み（#4495/#2586/#2506）、Graph Reader・Studio Stage1→Stage2の並行P0は維持する。

**REAのWireと比較結果を正しく拒否する**：main #4883の未観測Sourceキー省略に続き、main [#4912](https://github.com/hat47x/tei-platform/pull/4912)は旧`rea-intake -before`による無根拠な`unchanged`を廃止し、正式Comparator未接続では`REA_COMPARISON_UNAVAILABLE`で停止する。原Evidenceを可逆なhex列として返していたFingerprintも`sha256:`へ変更。byte digest、Candidate Identity、Owner意味受理、現在Grantを別資格とし、既存#1180/#2514のExpectedで検収する。

### 記述形式と意味の同値性を分けて検収する

**構文の採否を決める条件**：読みやすさ・短さだけでなく、同じ業務意味に対応するかを、(1)構造、(2)評価、(3)変更の三軸で独立に確認する。GraphをTreeやMarkdownへ投影して同じNodeが見えても、Edge ID・向き・多重辺を失えば構造は同値ではない。集計結果が一致しても、SQLのBag多重度・NULL三値論理・Windowの順序とFrame・JOIN粒度が違えば評価は同値とはいえない。派生Viewを表示できても、そのViewから元Sourceの共有属性や関係を一意に変更できるとは限らない。形式が異なる場合は、同値と保証できる**範囲**を明示し、損失した意味はunknown／unsupportedに止める。

**意味へ昇格できる条件**：ファイルの拡張子やMarkdown見出しではなく、Workspace／Language Profileの選択・exact pin・文書内のRole指定・元byte span・型付き参照・明示された評価条件から決める。説明文、引用、Code Fence、非選択Source、AIの候補が、近接しているRuleやDataと暗黙に同じAuthorityを得ることはない。同じRule IDが表とDSLに併記されている場合は、どちらが正本か明示するか、重複として拒否する。MermaidをSourceとして解釈する図と、型付きモデルから生成した表示図も別資格とする。

式を短く表す構文は[#4359](https://github.com/hat47x/tei-platform/issues/4359)のA3ハイブリッド候補（宣言骨格＋限定的な式位置）を比較対象に残すが、Goの`struct`／`func`／`package`を日常DSLへ持ち込まない。関数チェーンは既存Operator契約の読みやすい表層であり、新しい実行Engineではない。Native SQLは原文とDialectを保持し、解析不能なOpaqueや入れ子の未対応Queryを意味を落として再生成しない（[#4752](https://github.com/hat47x/tei-platform/pull/4752)）。数値の未指定・明示NULL・0・未取得を分け、整数精度、指数表記、単位、時点、集計範囲を勝手に補完しない。

### 同一Attemptの受渡しと自己記述への到達を検収する

製品の次の最小接続は、現在の`TEI_MD_PARSER_UNBOUND`を根拠なく解除することではない。**選択された一件の`*.tei.md`**について、Manifest／Pinned Root／Frontend選択・契約版／元byte／`readByLoader`／選択されたDocument Role／既存Source Index Artifactとbyte spanを、**同一Load Attempt**の結果として対応付ける。SourceIndexの役割（出典追跡）とMeaning Role（業務上の役割）を区別し、未実装のFrontend選択はunsupportedのままにする。これは[#4751](https://github.com/hat47x/tei-platform/pull/4751)でmainに入った**テスト専用S0-P**を、製品受入**S0-A**へ接続する差分であり、単なる予備検証の再実装ではない。

正例は、選択Root内のGraph／Rule一件を原文・Role・Spanとともに読み、読み込まれたSourceだけに由来する候補を確認する。負例は、Root外や未選択draft、引用・Code Fence内の偽見出し、未閉Fence・裸`|`行、重複ID、未知端点、Source改版による旧Index、旧`*.tei`互換、未選択／版不一致Frontendを、独立Expectedで検査する。[#4725](https://github.com/hat47x/tei-platform/pull/4725)のReference負例は採否と実装範囲を確認して移植し、Reference試験だけで製品の合格とはしない。

その後、意味の受理→Rule条件を反映する生成Go→生成Pluginの**実Build入力**→通常CLI/MCP・別Consumerの**実Call**→Studio Stage1自身によるStage2実再入へ進める。自己記述B0〜B4（[#2620](https://github.com/hat47x/tei-platform/issues/2620)／[PR #4744](https://github.com/hat47x/tei-platform/pull/4744)）と、実BuildのN0〜N3／B1-A〜C（[#4583](https://github.com/hat47x/tei-platform/issues/4583)）は担当別に照合する。Seed／固定Interpreterが新版の言語定義を読むこと、生成Parser・Operatorの機能、生成Pluginの次Builderでの実利用、Windows上のStage2 Desktop再構築を、互いに代替する証拠として扱わない。Pluginの選択・包含・実呼出し・公開・認可・Effect観測も別々に記録する。

### 差分に応じた再利用の判定

| 一因子だけを変える変更 | 必ず区別する結果 |
| --- | --- |
| 同一生成Binaryに渡すNode IDだけ変更 | 定義と生成物を保持したまま入力別結果を検査。実行結果や実呼出しEvidenceは新たに取得 |
| Mermaidの配置・Sourceの説明だけ変更 | Raw Source byteとSpanは変わり得る。業務上の候補構造が同じか確認し、実装再生成の要否を別判定 |
| GraphのEdge向き／多重性、Ruleの条件変更 | 候補構造と評価が変わるため、旧Meaning／Artifactを無条件に再利用しない |
| Operator／Generator／Pluginの版・指紋変更 | Sourceが同じでも生成器・意味契約・選択資格を再検証する |
| SourceのA→B→A | byteや局所的意味候補の一致はあり得るが、以前のOwner受理・Current Grant・外部Effect Evidenceは自動復活しない |
| 部分取得0行／明示NULL／不明・非開示 | 完全空集合や0と同一視しない。既存EmptyPolicy／Coverage／CompletionStateとSource資格で結果を分ける |

Graph／Ruleの候補構造指紋を生成Binaryから照会する[PR #4758](https://github.com/hat47x/tei-platform/pull/4758)は、上表の一部を検証する**main統合済みの限定Reference実装**である。Raw Source SHA-256、候補Structure Fingerprint、Accepted Meaning Revision、Artifact SHA-256、実行入力、現在のAuthority／Evidenceは別物であり、候補指紋の一致を製品Bundle・Provider・実行許可の資格へ昇格させない。

**main統合済み負例の受渡し（2026-11-15）**：[Graph Source再読込み #4759](https://github.com/hat47x/tei-platform/pull/4759)は、説明／Mermaid Layoutだけの変更では候補構造指紋・限定Reference生成Goが同じでも、Raw Source byteが違えば旧Source Indexは無効であることを実Go回帰で固定した。[実Binary照合 #4761](https://github.com/hat47x/tei-platform/pull/4761)は、候補指紋の定数を変えず生成Goの判定条件だけ改変すると、同じ指紋を申告する二BinaryのSHA・実行結果が異なる負例を実Go回帰へ追加した。いずれも限定Referenceの合格であり、正式Language/Role/Meaning・実Builder Source member・通常CLIの実ConsumerやHost認可の合格ではない。次の製品縦断では、旧Indexの拒否と生成物の実Build入力・実Binaryの外部照合を、同一Source／Subject／Target／Attemptで独立に検査する。新しい証明機構や第二Evidence Storeは設けない。

### 宣言的構文の品質とセルフビルドへの寄与を測る

最低限、同じFixtureを従来`validation`と低ノイズ`rule`で書き、専門家が読む順序、重複参照、曖昧性診断、byte位置、原文無変更保存を比較する。次にGraph／Rule・数値集計・判断表・Function/Operation各一件について、独立Expectedと負例から、生成された**条件分岐・型付き関係・制約・UI／Plugin構造**が実動作へ寄与したか確認する。

測定対象は、(a)業務仕様として記述する量、(b)手書きのTargetコードと重複定義の残量、(c)実際に生成・Buildへ包含・呼出された構造／処理の割合、(d)意味変更の際の手修正・再生成範囲、(e)初回から実Consumer受領までの失敗・修正・所要時間とする。数値目標を先に固定せず、同一機能の旧実装との比較で効果を評価する。Goに全処理をdelegateしただけの薄い`*.tei.md`、使われない生成ファイル、合成Mock成功はセルフビルド達成として数えない。

**Ownerと次の引渡し**：構文選定・read-only Validationは#4359、Source／Readerは#2621/#1018、Role／自己記述は#2620、構造と限定Patchは#2495/#2497/#1124、実ProviderとQueryは#2586/#2387、Pluginの選択・公開文書は#951/#4679、独立Expected／Scenarioは#2514/#1180、Stage1→Stage2は#2831/#4583の既存正本を維持する。各段階の未受理理由を次Ownerへ渡し、全体P0に新しいHard Gateを追加しない。

### 意味契約の適用範囲を固定し、RuleとQueryを同じ基準で評価する

一つの宣言を意味候補として比較する際には、**対象の識別、参照経路、評価範囲、時点・版、入力の完全性、演算の契約、結果とEffect**を確認する。これらは新しい共通SchemaやCore enumではなく、既存Data／Function／Business・Facet／Role／Data Context・Operator／Query／Rule／Lifecycleの契約を同じFixtureで突き合わせるための**確認観点**である。型が同じで結果の表示が似ていても、以下のいずれかが不明なら全面的な意味同値を主張しない。

| 観点 | Rule／Decisionで確認する意味 | Query／Graph／Timeで確認する意味 |
| --- | --- | --- |
| 対象とIdentity | 判断対象の業務キー・Scope、同名Ruleの正本 | Node／Edge ID・業務キー／複合キー／Source occurrence |
| 関係と粒度 | 条件が参照するRelation path、多重度、集約前後 | JOIN粒度・Bag・Edge向きと多重性・Window partition |
| 時点と適用期間 | ルール有効期間、申請・承認・観測の時点 | Snapshot／transaction time／計画時刻と実観測時刻 |
| 入力の資格 | NULL・未取得・非開示・partialを区別した評価可能性 | 0件でも完全取得か、結果範囲・欠測・単位・精度 |
| Operator／Provider | Rule版・Hit Policy・純粋判断とEffectの区別 | SQL方言・Native/Opaque・Window frame・能力とpin |
| OutcomeとAuthority | satisfied／not-satisfied／unverified／conflict、受理と実行Grant | Resultと実Effect・Receipt、現在のConsumer資格 |

**限定的な確定と不確定の境界**：Unique決定表で、同じScopeの二つの確認済み行が同時一致すると証明できれば、取得済み資料がpartialでも**競合が存在する**という局所結論は成立し得る。一方、一行だけの一致や零件から、必要なRule集合が完全である証拠なしに「唯一の決定」「該当なし」とは断定しない。未取得と明示NULL、Rule非一致とアクセス拒否、空集合と不完全な取得を混同しない（[BRMS統合設計](design/BRMS_DECISION_ANALYSIS_INTEGRATION_2026-11.md)／#1180／#2514）。これは全ケースを新Rule Engineへ集約する提案ではなく、既存Rule／Evaluation／Coverage／Source資格で判別する境界である。

**反映先を限定する**：REAのResource／Event／Agentや組織の責任移管は、時間条件・Identity／Relationの適格性を確認する専門圧力ケースにする。SUIのKJ配置とWBS依存DAG、Studio/SHOの図解と業務意味、テスト用Mockと実Provider Effectはそれぞれ区別する。構文汎化がこれらを単一のGraph／Rule形式へ強制する理由にはならない。未対応のNative SQLをTEI標準Queryとして再生成せず、入れ子Opaqueの明示拒否（mainへ統合された[#4752](https://github.com/hat47x/tei-platform/pull/4752)）を守る。

### 一つの小さな製品縦断で、宣言の有効性を確認する

次の一件を、既存[#4359](https://github.com/hat47x/tei-platform/issues/4359)のS0比較と[#2621](https://github.com/hat47x/tei-platform/issues/2621)の正式Readerで共有する。**同じGraph/Ruleを異なるSource表層で表し、同じ独立Expectedを使用するが、元byte位置・表示・保存は同じにしない**。

1. `*.tei.md`の宣言済みRoleから同名二Node・多重有向Edge・単一Ruleをread-onlyで取得し、Sourceと候補の一対一対応を照合する。未選択・引用・Fence内の偽規則は拒否／無視の規約を区別して検査する。
2. Ruleの条件で生成されたGoの分岐を、元のRuleと**独立したExpected**で照合する。既存Referenceの`-node`・`-candidate-shape`は製品CLIの仕様とはしない。明示Role、Meaning受理、選択Operator、Target Generatorの資格を次Ownerでつなぐ。
3. 生成Sourceを実Builderが読み、選択したGo/PluginとArtifact、実行したBinaryを同一Build Attemptへ束縛する。候補構造FingerprintとBinaryの自己申告だけでは実コードの正しさを保証しない。[#4758](https://github.com/hat47x/tei-platform/pull/4758)はmainの限定Reference、改変したGoが同じ候補指紋を申告する[#4761](https://github.com/hat47x/tei-platform/pull/4761)はmain統合済みの限定Reference負例として扱う。
4. 実行時Node変更、Edge方向変更、Rule条件変更、説明・Mermaid配置のみ、Generator／Plugin版変更を一因子ずつ試し、Source再読込・意味再検証・実装再生成・実呼出しEvidenceを必要な範囲だけ更新する。Artifactが生成されてもBuilderの入力にならなければ未達とする。
5. 次に同じ契約の`validation`正負二規則、Unique決定表のpartial／競合、SQL WindowのNULL・順序・Frameを**追加の比較対象**にする。Graph実証の成功をDecision／SQLの全面同値やDB実Consumerの合格へ転用しない。

**実行順**は、製品Source Reader S0-A → Accepted Meaningの限定一Rule → 通常Generatorと実Builder入力 → 通常CLIからの実呼出し・別Consumer → Stage1→Stage2 Desktopである。独立ExpectedとScenarioのIDは既存#1180／#2514で管理し、本文の比較表は仕様・検収手順の索引にとどめる。Grammarの追加、標準Data型の採否、Hit Policy、SQLの変換可否は各Ownerで決定する。全体P0を変更せず、正式Reader未対応時にはSource候補の段階でunsupportedとして停止する。

<!-- tei-declarative-contract-obligation-context-roundtrip-20261115 -->
### 宣言・実装義務・実現方法を混同しない

他のDSL自己記述・BRMS・標準データ様式の成熟成果を、既存の意味構造と製品受入へ渡す。**宣言できたことと実装できたことを分ける**ため、対象・判断・操作ごとに次を確認する。この表は確認票であり、新しい共通IR／Schema／Runtime分類ではない。

| 宣言の種類 | 意味として必要な条件 | 実装・受入で別に確認すること |
| --- | --- | --- |
| **Data／Relation** | Identity、Key／Relation path、Cardinality、粒度、Source／時点 | 取得済み範囲・集約・Joinの意味が保たれること。多重Graph Edgeと同一Personの複数Occurrenceを混同しない |
| **Validation／Constraint** | 対象、条件、NULL／欠損・精度・単位、評価対象の完全性 | `satisfied`／`not-satisfied`／error／unverifiedを独立Expectedで区別し、未解決を暗黙の真偽へ落とさない |
| **Derivation／Decision** | 導出元、適用範囲、Hit Policy、条件の有効期間、競合時の意味 | 生成された条件分岐と実際の出力が一致すること。partialの正証拠によるConflictと、全件取得が必要な唯一性／不在を分離する |
| **Process／Transition** | Trigger、状態遷移、先行条件、期限、イベントの因果・時間順序 | 判断の依存関係を実行順序と混同しない。再入・重複イベント・取消・失敗時の遷移は、契約で示した範囲だけ検査する |
| **Operation／Effect** | Input／Output、pure／外部Effect、認可主体と用途・Tenant Scope | 実Provider／Hostの現在Grant、呼出し、部分効果、Receiptを別に検収する。宣言された操作を自動実行しない |
| **Trait／Plugin能力** | 契約が要求する機能、版・依存・選択条件 | Traitの宣言だけで実装済みとはしない。実装義務の充足、生成物のBuilder実消費、通常CLI・別Consumerの実Callを確認する |

**自己記述の責務境界**：[#2620](https://github.com/hat47x/tei-platform/issues/2620)と[自己記述設計§45・§48〜54](design/ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)のD1記述可能→D2解釈可能→D3拡張可能→D4生成可能→D5再構築可能、および有限Seed B0〜B4を基準とする。Language／Role／Operator／Traitを`*.tei.md`に書けても、実Parser／Resolverが読んだことや生成したことにはならない。**選択・読取・解釈・採用・生成・利用・再構築**の実消費連鎖を満たす範囲だけ合格とし、手書きGoへのDelegateや使われないGenerator出力を自己実装に数えない。Studio Stage1→Stage2の実Windows再入はPlugin単体成功と独立に確認する（#2831/#4583）。 **§48のM0〜M4**は、既存Ruleの説明→契約による検査→手書き意味分岐の実置換→製品Builder/Consumer実使用→次段TEIの再構築を別々に検収する。最初の対象を`positive`一規則のRequired Shape／診断／Lowering適合条件に限定し、独立Expectedと実消費を満たさないDescriptor件数の増加を進捗として数えない。 **§49の次段**では、PR #5007で実装されたOperatorの宣言／manifest実装選択／Scenario評価関数での実Callを、自己記述の別Subjectとして検収する。選択だけ変えても型付きWorldは同一となり得るため、World Revision一致で実装の現行性を認定しない。最初の作業はSD-M1（正値Ruleの形・診断の検査候補）とSD-O1（同一Operator・入力0で実装選択を変え、実Callと結果および旧生成Scenarioコマンドの失効不足を確認）を並行する。M2の一箇所置換、製品R2-P、R3、Studio Stage2は別の受入に残す。

**第50節の実装優先**：製品で読める印付きData項目定義表（tei.authoring/v1）は既存semantic.Dataへの表層であり、独立したMeta InterpreterやOperator Lawの完成を意味しない。#5009でRule由来Goが通常tei self-buildに達し、#5020で原DSLのSource位置と生成Go ASTを照合できる一方、正式R2-Pの原文・Generator→Build Intent→実Builder入力の同一Attempt閉包は未完了。第一のM1はpositive一規則のRequired Shapeを既存Meaningからread-onlyで投影し、独立Expected後にGoの手書き検査を一箇所**実際に削減**した場合にだけM2と認める。別系統のOperator Law／manifest選択／実Callを混ぜず、旧Scenario Commandの鮮度と原Rule出典の不足を#2600／#4544に渡す。Studio Stage1→Stage2 Windowsと通常CLI実Providerを並行P0に維持する。

**第51節の検収基線**：Goのpositive形検査に固定の正常・負例テストを追加し、自己記述から検査実装を置換する際の独立Expectedの起点にする。これはM1/M2実現ではない。最新mainのRule原文をGo embedで完成Binaryへ収録するR2-P限定回帰を認めつつ、原文コピーの実消費・意味解釈・Generator選択資格を区別する。配布Binaryへの原文内包は公開許可を意味せず、非公開Sourceや業務記録の同梱は既存Disclosureの確認まで行わない。生成Windows Studio PoC Bの実Job成功も、再帰的Stage1→Stage2の成立とは分ける。Operator旧Scenario失効は並行実装PRで扱い、同じGo修正を複製しない。

**第52節の自己記述実験**：通常のtei.authoring/v1とsemantic.Dataを使い、TEIのValidation宣言が持つ項目をTEIの印付きData表として記述する製品Reader回帰を追加する。選択Reader、Canonical Data、項目別Source位置、同じIDの実Validation／Capabilityを自動生成しないことを検査する。required=noは「Relation指定禁止」ではないため、positiveのRequired Shape／Lawをこの表のLoadだけで受理しない。将来のM1は既存型・Roleから未表現の制約を補う根拠を比較し、一義務を生成・実使用してGo手書き分岐を削除した場合のみM2へ進める。詳細は[自己記述設計§52](design/ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)。

**第53節の抽象経路検収**：同一の型付きRule／Operationを、直接解釈P・Go生成G・Provider委譲Dのどの方式で実現するかは、Targetごとの選択であり意味IDではない。テストはメソッド名でなく、**Subject・実装義務・前提・要求経路・不変条件・独立Expected・実Attempt**で識別する。静的な配線適格／実際の通過／通過中の状態保存を別々に検査し、選択G／実Called Pのような結果一致だけでは発見できない不一致を拒否する。型付きPluginのWhole／Field／Adapter MappingとSource・Designer・Resource・Generator・Generated／Custom／Artifactの所有区分を同じ経路検査へ接続するが、新Path DSL・第二Runtime・専用拡張子は導入しない。最初は既存positive RuleのP/G比較と#5101の型付きMappingを再利用し、自己記述M2の一義務実置換、R2-P出典閉包、通常CLI実Provider、Studio Stage1→Stage2 Windowsを別受入に維持する。

**第54節の実回帰**：Data表と隣接Ruleから選択した同一意味Subjectを対象に、生成Goの関数名だけを変更する正例と、結果が同じでもRead先が異なる負例を**同じ実Binary試験**で比較する。新しい識別子やRuntimeではなく、既存TestMarkedDataTableRuleRealGoAndEvaluatorReadParityへ追加する。関数名変更後の結果・Read同値は意味上の対応を示すが、Go生成byteとBuildPlan／Artifactの再利用資格まで継承させない。#5109で実施したPlan解決後Go差替え拒否を別の証拠とし、#5097のData自己記述Reader回帰と#5114のScenario Operator選択Pinは未マージの別Owner作業として維持する。次のM1→M2はRequired Shapeの一条件をTEIから実装し、現行GeneratePositiveの重複Go分岐の削除と実使用を独立Expectedで確かめる。

### Data Contextと時点を、構文に埋め込まないまま明示的に束縛する

**「何を、いつ、どの範囲まで読んだか」**を、専門家にとって読める対象→条件→判断→結果の順で説明できるようにする。Data Contextの`grain`、Collectionのcompleteness、Contextをまたいだ同一Subject binding、型付きRelation pathとそのoccurrence、有効時点、取得元と版、Operator契約のpinを、既存Data／Function／Query／Role／Facetから辿れることが必要である。これらをすべて独自構文キーワードにする必要はなく、Markdown表／短いRule／型付き関係など、そのSourceで自然な形式を選ぶ。

同じPersonに複数の経路で到達しただけでは複数人ではない。異なる人物を二人確認できても、承認権限があると推定してはならない。逆に、部分取得の下で同じScopeに一致する二つのRule行が確実に存在すれば、Unique Conflictは確定できるが、一行だけの観測から唯一性は結論できない。**Contextの参照可否・値の既知性・業務上の適格性・現在の実行権限**を別の判断として維持する。

REAのResource／Event／Agent、組織の権限委譲／兼務、B2Cの非Membership主体、SUIのPERT依存、時間軸付きデータ、SQLのWindow／Bag／NULLを、この契約の圧力試験として使う。ただし初回製品Reader/Graph Ruleをこれら全領域の実装完了に依存させない。Source側の語彙は英語の短い業務表記を基本とし、表示上の日本語名称や自然な説明文は別のProfile／Viewから生成できるようにする。Goの`struct`／`func`をDSL表層へ導入せず、同一Ruleから「対象・条件・判断・不足時の扱い・結果」をほぼ直訳できるか比較する。

### 安全な編集を、構文汎化の必須の反証ケースにする

Studio／SHO／SUI／AIが同じ意味候補を提示しても、編集資格は同じではない。**Preview→Resolve→Admit→Commit→独立再Read／Observe**を既存のSource/Writer/Acceptance境界へ接続する。Parserが作成したSource spanは編集の候補位置であり、対象の一意性、Source Revision・World Fingerprint・依存閉包、利用者／HostのGrantが一致しないまま書き換えない。

| 編集・再生成の例 | 必要な期待結果 |
| --- | --- |
| Markdownの説明・Mermaid配置だけ変更 | 原byte／Source Indexは更新。業務Meaning／生成コードは同値の可能性があるが、旧受理資格やSource位置を流用しない |
| Graphの同名別Node・多重辺の一つだけ変更 | Edge IDと向きの指定を要求し、同名Nodeや別辺への誤適用を拒否する |
| 表・Mermaid・Ruleの二つが同じ意味の正本を主張 | 明示したAuthoritative Sourceの外は表示／候補とし、未決なら競合停止。先勝ち・後勝ちしない |
| 複合Key／REA関係／SQLのlossyな表示から逆編集 | 欠落しているField対応／Bag／NULL／粒度を補完せず、変更対象を確定できなければread-onlyとする |
| 同時編集でA→B→A、別Sessionから古いCandidateを適用 | byteが戻っても旧Intent／Grantを自動復活しない。保存前に現行Source／意図／権限を検証する |
| 未取得・非開示のFacetをAIが説明に補った | 推論をCandidateとして区別し、正式Source／公開文書／実行権限には自動昇格しない |

これは新しいDocument Writer／Patch Engineを作る指示ではなく、既存[#2497](https://github.com/hat47x/tei-platform/issues/2497)／[#2621](https://github.com/hat47x/tei-platform/issues/2621)／[#1124](https://github.com/hat47x/tei-platform/issues/1124)の限定Source編集と、[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Expectedへの引渡しである。

### 最初の実装順と成熟判断

1. **S0-A／Source**：既存[#4751](https://github.com/hat47x/tei-platform/pull/4751)の予備試験を重複実装せず、Manifest/Frontendのexact選択を受けた`*.tei.md`一件を実`Workspace.Load`のReaderで消費する。元byte・Role・Span／Indexの同一Attemptを検証する。未提供Profileは`TEI_MD_PARSER_UNBOUND`相当で停止。
2. **Meaning／独立Expected**：同名二Node・多重Edge＋一Ruleを、Source-bound Typed Candidateから既存の受理境界へ渡す。Ruleの評価適格性（Context・時点・partial）を明示し、旧`validation`二規則との構文比較は[#4359](https://github.com/hat47x/tei-platform/issues/4359)のS0／Owner判断のまま。
3. **Generation／Consumption**：一Ruleの**条件分岐自体**を既存Generatorから生成し、実Builder Source member／Toolchain／Artifact／実Binaryを束縛。限定Referenceの候補指紋と実Binary SHAの負例[#4759](https://github.com/hat47x/tei-platform/pull/4759)／[#4761](https://github.com/hat47x/tei-platform/pull/4761)を再利用するが、正式製品の合格へ読み替えない。
4. **Plugin／CLI／self-build**：同じ意味から生成した出辺判定Pluginを通常CLIの実Operationで呼び出す。Plugin自身のRequired Shapeと実装義務をTEIに移す範囲を段階計測し、Stage1→Stage2実Builder／Windows Desktopは別に受理する。
5. **一般化判断**：独立した第二のValidationと、限定Window／Unique決定表を**反証**に使う。異なる表層での診断／説明の自然さ、手書き判断の削減、生成部品の実使用、失われる意味と安全に編集できる範囲を同一Fixtureで測り、正式Grammar・Trait合成順序・Plugin ABI・標準Data型の全面採用を先回りしない。

本追補は新仕様・新Hard Gateの採用ではなく、既存Ownerへの実装引渡し案である。[#4771](https://github.com/hat47x/tei-platform/pull/4771)のデータ様式計画がmain統合済みであることと、自己記述§45の成熟結果を参照する。新しい製品Reader、Rule／Plugin生成、実Provider／CLI／StudioのE2E、Owner業務意味受理は本書の追補によって**実装・実証済みにならない**。

<!-- tei-declarative-lawful-composition-20261115 -->
### 宣言を合成しても、意味と作用が変わらない条件を明らかにする

次の実装では、短いRuleや関数チェーンを増やすことより、**その記述を別の表現やTargetへ移してよい条件**を先に確認する。宣言的構文の汎化は、見かけ上の式の一致を理由に、判断の適用範囲・結果の多重度・失敗・外部作用を変えてよいという意味ではない。

**最初に区別する三つの操作**：

- **読み取り・判定**：Graphの関係探索、Validation、計算、Queryなど。受理済みSourceとData Context、型・NULL／未取得、Operator契約の下で値と診断を評価する。出力値があっても、外部Effectの実行を意味しない。
- **変換・最適化**：Filter／Map／Aggregate／Join／Pipelineの組み替え、SQL方言へのLowering、GraphのView生成など。同じ入力で値が一致するだけでなく、Bag多重度、順序、精度・単位、Unknown、エラーの発生、Sourceへの追跡が保存される範囲だけ認める。未証明の変換は行わず、必要なら原表層／Opaqueを保持する。
- **状態変更・外部作用**：業務Operation、Transition、DB Commit、通知など。明示されたTrigger、事前条件、実Providerと現在のGrant、Effect／Failure／Receiptの契約に従う。Ruleが真と評価されたことや図に矢印が描かれたことだけで、自動実行や成功を認定しない。

**特に避ける変換**：`distinct`を暗黙に挿入して多重辺やSQLのBagを集合へ変えること、NULLや未取得をFalse／0にすること、順序が必要なWindow／LIMITを未規定の順序で処理すること、結果が部分取得の`count=0`を不存在とみなすこと、失敗・丸めがあるMapとFilterを無条件に入れ替えること、未対応のNative SQL部分式を落として残りだけを再生成すること。また、Mermaidの線・PERTの依存・Process図の縦位置を、受理済みの因果／先行条件なしに実行順へ変換しない（[実行モデル§45〜50](design/PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)）。

| 最初に比較する一因子差 | 独立Expectedで確かめること | 次の引渡し |
| --- | --- | --- |
| 同名二Node・多重Edgeから別の表示を生成 | ID・向き・多重性は意味として保存し、表示配置は実装の処理順としない | #2497／#4359 |
| 同じ一Ruleを表／短いDSL／Go生成で表す | 対象・条件・Unknown・診断と元byte位置を比較。Goへ委譲するだけの成果は分ける | #4359／#2621／#4544 |
| Filter／Map／Joinの並べ替え | NULL・重複行・順序・エラーを含む反例で、同値といえる範囲だけ採用 | #2387／#988 |
| 部分取得の決定表と集計 | 確認済みのConflictと、全件性が必要な唯一性・不存在・総数を区別 | #1180／#2514／#2506 |
| Rule判定後のDB／通知とACK欠落 | 判定・実Call・Effect・Receipt・Consumer受領を分け、結果不明なら自動再送しない | #1174／#1129／#2586 |
| Go／SQL／Java等への別Target生成 | 型・NULL／欠損・丸め・例外・時間条件の意味を、Target別の独立Expectedで照合 | #4544／#2387／#2514 |

**着手順**：[#4359](https://github.com/hat47x/tei-platform/issues/4359) S0では、この比較を正式構文の採否資料に限定する。S0-Aの最初の製品PRは引き続き選択`*.tei.md`の実Reader／Role／Source Index接続だけを対象とし、最適化器や新しいRule Runtimeを先行実装しない。その次に受理済み一Ruleの生成コードで、同一Contextの評価結果・失敗・実使用を照合する。Effectを含む実証は通常CLI→実Providerと独立Consumerで別に受け入れ、Studio Stage1→Stage2のP0を妨げない。

この節は構文・Operator・Targetの**合成可能性に対する実装引渡し**であり、新たな式文法・代数Optimizer・Process Engine・Target同等性の正式採用ではない。正負のケースは既存#1180／#2514へ渡す。ここで整理した反例をGo／DB／Windowsの実行試験として実施したことにはしない。

<!-- tei-declarative-evidence-projections-20261115 -->
### 宣言の解釈結果を、CLIとStudioで安全に確認できるようにする

宣言的構文を汎化した成果は、Parserが候補を返した時点ではなく、**利用者と実装担当が「確認できたこと／まだ確認していないこと」を同じSourceと版で追えること**で評価する。既存のSource Index、Plugin Inventory、Subject Inspect、Builder／Artifact／Operationの証拠を読み取り専用に投影し、新しいEvidence Store、権限エンジン、共通Status enumを増やさない。

実装の足場として、main統合済み[#4800](https://github.com/hat47x/tei-platform/pull/4800)の`plugin-inventory -plugin`には、実選択PluginのVersion／Lockだけを判定する`documentReadPreflight`がある。たとえ`selectionStatus=exact-version-pinned`でも、`sourceBinding=unbound`、`disclosure=not-checked`、`readEligibility=not-assessed`のままであり、文書本文の存在・読取・開示を認定しない。main統合済み[#4806](https://github.com/hat47x/tei-platform/pull/4806)では、参照用Workbenchが`InspectSubject`の`authorityLimits`を落とさずに扱う実装を追加したが、現時点で生成・埋込済みWails資産の利用まで実証したものではない。両者は**共通の受入原則を異なるConsumerへ投影する実装**であり、直接結合する新しい製品APIを要求しない。

| 見えた事実 | そこから認められること | 追加の照合がなければ認めないこと |
| --- | --- | --- |
| Manifest管理SourceをPinned Rootから読める | そのbyte列とSource Indexを照合できる。Graph／WindowのS0-P負例にも適用 | 正式Readerの解析・Role確定・Owner意味受理（S0-A） |
| Pluginのexact版とLockが一致 | その環境での既存Resolverの選択結果を確認できる | Document帰属、Source閲覧、現在Disclosure、Provider実Call |
| `InspectSubject`の対象とSource位置が返る | 現在Replyのread-only表示と根拠への参照 | 編集・公開・実行権限、Model Revisionの受理 |
| 候補Graph／Rule指紋や生成Goが一致 | 限定Reference上の構造・生成物を比較できる | 実Builderが使用したSource memberやArtifact／実Binaryの同一性 |
| 実Consumerが値や診断を返す | 実際に受け取ったResponseと独立Expectedを比較できる | 完全なRead Coverage、外部Effect確定、別Audienceへの開示 |

**最小の製品検収**は、同じWorkspace／Subject／Source Revisionを選んで、CLIのSource／Plugin選択結果とStudio Inspectorのread-only表示を比較する一件から始める。Studio側でWorkspace／選択／Sourceが変わった場合には旧Replyと旧`authorityLimits`を新しい対象へ持ち越さない。欠損した制限フィールドや不適格なResponseは表示側で拒否し、非公開のSource、件数、識別子、理由が別Audienceへ漏れないかを二世界の負例で確かめる。**未観測**を「存在しない」「利用可能」と表示せず、Source未読取、Plugin未選択、Coverage不完全、権限未照合を区別する。

**既存の引渡し先**：製品S0-AのSource／Frontend／Roleは#2621/#988/#1018、Plugin文書のSource-backed一件取得とDisclosureは#4714/#4679、WorkbenchのE-READ／E-PIN／E-TRACEと実資産照合は#1824/#4515/#4583、生成物の実消費は#4544/#2600、独立Expectedは#1180/#2514に戻す。WorkbenchのTypeScriptテスト成功から実`cmd/tei-studio/assets/workbench`／Wails埋込／Windows WebView2をPASSとせず、正規Build出力と埋込byte、実起動したConsumerを順に照合する。

この追補は**限定的な実装済みread-only機能を製品受入へ接続する設計案**である。正式`*.tei.md` Reader、現在Disclosureの実判定、Plugin Document読取、実Provider／Stage2 Desktop、業務Ownerの意味受理を実装済みに繰り上げない。Studio Stage1→Stage2と通常CLI実Provider ReadのP0を維持する。

<!-- tei-declarative-s0-evaluation-evidence-20261115 -->
### 構文S0の比較を、読みやすさと観測できる意味の両面から始める

[#4359](https://github.com/hat47x/tei-platform/issues/4359)のS0比較は、構文の採否を先取りせず**同じ二つの業務規則を何通りの表層で自然かつ厳密に表せるか**を確かめる最小試験とする。対象は既存`reference/workspace/semantic/sales/validation.tei`の数量の正値検証と明細番号の一意性検証。現行`validation ... positive/relation-unique`を基準とし、提案`rule ...: X is positive/unique per ...`、宣言済みRoleのMarkdown表、外部Pin付きOperator呼出しを**未採用候補として**並べる。実Frontendにない記法を受理済み・実行済みと表示しない。

| 比較軸 | 事前に固定する基準 | 記録する結果と担当 |
| --- | --- | --- |
| **業務文としての読み順** | 対象→条件→判断→結果、明示的なScope。Goの`struct/func/import`は要求しない | 初見の読者が対象と条件を読み取れるか、不要な参照反復・構文トークン・説明の追記を要した箇所を記録。#4359のOwnerが採否判断 |
| **意味と評価の区別** | 数量2は`satisfied`、0は`not-satisfied`、明示NULLはerror。明細番号1,1は`not-satisfied`。空の評価対象は現行Evaluatorで`satisfied` | 同じ独立Expectedを全候補で参照。空対象の局所評価結果だけから、実ProviderのRead全件性・不存在を主張しない。#2514/#2506 |
| **構文／Source忠実性** | 同一のValidation ID・Kind・Attribute・Relationを保持し、原byteとSource spanは候補ごとに区別 | BOM/CRLF、同名別Source、説明表、Role未選択、No-op保存、位置付き診断、旧`*.tei`互換。#2621/#988 |
| **評価時の観測** | 同一の戻り値に加え、評価した条件・必要Read・失敗・非開示を区別 | [#988](https://github.com/hat47x/tei-platform/issues/988)のEQ0〜EQ2とDTQ-01〜09を再利用。短絡で不要な非公開Fieldを読まないScriptを、先読みする決定表へ無条件に書き換えない。#2506/#2515 |
| **実装への到達** | 選択Frontendの同一`Workspace.Load`元byte→Source Index→意味候補をS0-Aで実観測 | `TEI_MD_PARSER_UNBOUND`を未実装時に維持し、Referenceの成功と正式Reader／受理／Generatorの資格を区別。#2621/#1018 |

**値一致への反証を実DBから取り込む**：main統合済み[SQLite Window負例 #4822](https://github.com/hat47x/tei-platform/pull/4822)では、同じ生成SQLを使い、B2が明示NULLの七行とB2行自体を削除した六行で、B3の累積平均がいずれも10になる。一方、B2を0に変更すると七行のままでB3の平均は20/3になる。したがって**一つの集計値が同じでも、InputのRow Identity・NULL／不存在・入力集合の閉包・Coverageは一致したとはいえない**。この実DBのC0負例を、Graph／RuleやDecision表の同値性に対する反証として再利用する。ただし、固定JSON→SQLiteの局所成功を正式Markdown Reader、標準Window Query、実製品Providerの受入へ昇格しない。

**実装に渡す順序**：まず#4359の旧記法と改善案の文章比較・独立Expected・観測可能なRead FootprintをS0で記録する。次に#2621/#988/#1018で**一つの選択済みGraph/Rule Source**の実`Workspace.Load`と候補位置へ進み、受理済み一Rule→実Generator／Builderの分岐変更を#4544/#2595で検証する。Window／Decision／異質TargetはS0-Aの前提ではなく後続の反証とする。専門家の可読性を重視しながら、Parserの実装容易性や結果値の一致だけで文法を選ばない。全体P0のStudio Stage1→Stage2と通常CLI実Provider呼出しも維持する。

**受入上限**：本追補は既存Reference・実SQLite回帰・DSL Ownerの独立Expectedを結び付ける設計手順。正式な新Rule文法採用、`*.tei.md` Product Reader、比較の実測、意味受理、実Provider・Stage2再構築は今回実施していない。

**現在の証拠と停止線**：mainの限定ReferenceにはGraph／Rule→条件分岐Go生成・同一Binaryへの実行時Node入力・Source Indexのbyte検査がある。一方、製品の正式Reader／選択Role／Accepted Meaning→通常Generator・Plugin・CLI実Call→Stage2実再構築は別の未達資格である。[PR #4744](https://github.com/hat47x/tei-platform/pull/4744)はMarkdown-firstと三要素図面の統合Draftであり、同PRや本節のdocs-only変更を製品機能の採用・Owner意味受理と読み替えない。


<!-- org-c1-first-product-read-cohort-20261115 -->
### 2026-11-15 組織C1：正の一致経路から実Provider Readへ進む最小契約（補遺§33）

[組織C1設計補遺§33](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)で、[コードDraft #4737](https://github.com/hat47x/tei-platform/pull/4737)の実Workspace Relation Key／合成二Context→テストConsumerによる候補Pathと、実DB Read Cohort・正式判断Rule・生成Go・現在Grantの**未接続部分**を整理した。既存Source、Function Context、Query/Provider、生成器、Expectedを再利用する。

- **実Read-1**：#2586/#4495で一つの選択済みOrg Queryを実Provider→実Rowset→Function Context→独立Consumerへ同一Attemptで接続する。Source/時点/Scope/Environment/Instance/SQL/args/NULL/拒否を照合。先行する受注Queryの成功とは別資格。
- **実Read-2**：#4490 BEH-01/02/04を先行条件として、二Relationの人物同定・validAt/knownAt・必要ReadとCoverage・実Snapshot/Scopeを比較する。一Query内の結合と、二Queryを同じRead範囲で実行する方式の両方を評価。RLS、Limit、否定例外、二時点Gapの反例が残れば唯一・全件・該当なしは保留する。partialでも確認済みの異人物2本の正経路は一意性の反証に利用できるが、承認適格の全件結論とはしない。
- **生成-1／表示-1**：#2621/#988/#1121/#4544の正式Reader→Accepted Meaning→Generator→実Builderで一Ruleを再生成し、説明・Mock・生成Go・独立Expectedを比較。#1124/#2516でCLI/Studio/SHOへSource・理由・Coverageを表示する際は開示資格を再判定。Case移管・Host Current Grant/Effectは#4501/#1174の独立検収へ委譲。
- **Owner・結果**：#2386がPerson/Role/責任・Case Linkの意味選択、#4490がRead/候補、#1180/#2514がExpected。新Rule Runtime、第二Source/Expected/Scenario正本、組織専用Host/DBは作らない。#4737はGo整形修正の新HEAD CIで再判定し、既存の合成Contextと実Providerの資格を混ぜない。

**優先順位**：最初の製品実Read #2586、Desktop Stage1→Stage2 #2831を独立P0として維持。今回の節は設計・作業引渡しであり、実組織Provider/CLI/Host/Windows/Studio E2EやOwner意味受理の新しい実行は**NOT RUN / NOT ACCEPTED**。


<!-- org-cross-view-rea-case-identity-20261115 -->
### 2026-11-15 組織・REA・案件をまたぐ主体同定と責任判断の安全な再利用（組織C1補遺§34）

[組織C1補遺§34](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)へ、REA・三要素分析・BRMS部分観測・Studioの判断理由Inspector・安全編集の成果を接合した。組織Person/Role/Responsibility、REA Economic Agent/Participation、Case Handling Unit/Handoff/Current Grantは**同名・同ID文字列・同じ画面表示だけでは業務上同一とみなさない**。Owner受理済みLink・Source/時点/Scope・Required Readがない限り、一つの承認者や業務担当へ短絡しない。

- **四つの異なる資格**：validAt、knownAt、実ProviderのRead Cohort/Attempt、現在Grantを別欄で照合。knownAtが同じでも同一DB Snapshotとは限らず、正の関係経路が見えても全件性・承認適格・実行権限を保証しない。Case移管やA→B→Aで旧Grant/Effectを復活させない。
- **反証・試験 OXV-01〜08**：未受理Person⇄REA Agent同定、二時点ずれ、Snapshot/Scopeずれ、同一人物の多重Occurrence、未取得否定条件、Case移管後の旧Grant、構造Dataだけに基づくFunction利用断定、Studio/SHOからの非公開理由・件数漏えい。これらは#1180/#2514の既存Expectedへ渡し、第二Result Schema/Engineは増やさない。
- **実装順**：§33の実Read-1/2（#2586/#4490）→組織意味Owner（#2386）とREA意味Owner（#3394）の最小Link受理→正式Reader/Rule/Generator（#2621/#988/#1121）→Case/Host実行の独立検収（#4501/#1174）→Studio/SHOの安全なread-only Projection（#1124/#2516）。Source→意味→生成物→実Read→開示→Effectは別Evidence。編集はPreview→Resolve→Admit→Commit→独立再Readへ委譲し、曖昧な逆編集を拒否する。

**現在の資格**：上記は採用前の受入契約であり、組織とREAの同一実DB世界・正式意味Link、組織の実Query→生成Go→CLI/Studio/Hostの一巡は未受理。#2586/#2831の独立P0を維持。


<!-- org-c1-person-revision-product-gap-20261115 -->
### 2026-11-15 組織C1：人物との関係・訂正版の保存・実Readを三段で検証する（補遺§35）

[組織C1補遺§35](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)で、合成r7の時点別人数/FTEと、実`org-staged-axis`のMembership/Person Link・物理保存・Readの間に残る二つの断線を具体化した。①現行MembershipにPersonへの正式Linkがなく実人数を確定できない、②物理`membership_id`単独PKのままでは同じ所属の訂正旧新版を同じテーブルに共存させられない。#4633のGAP試験と#4567の二時点Query/使い捨てPostgreSQL試験は**ともにmain統合済み**だが、GAPの検出と局所Query/実PG試験であり、履歴版共存・人物同定・通常CLIの製品完了を示さない。

- **先に意味Owner**：#2386/#1126がMembership→Personを直接またはEmployment/Affiliation経由で解決するか、業務IDと履歴版をどう識別するか、兼務・複合キー・有効区間を決める。履歴非必須アプリには二時点Schemaを強制しない。
- **次にProduct**：#4633の「欠落を確認する」GAP特性試験を採用契約の正例に置き換え、#4567のphysical Query/Mappingと同一Membership旧新版を隔離DBで保存・再読込する検査を接続。その後#2586/#4495/#4490で同一Attemptの実Provider→Function Context→独立ConsumerとvalidAt/knownAt・Required Read/Snapshot/Coverageを証明する。
- **MVR-01〜05**：人物Link・版共存・二時点の実Result・複合Key/NULL・RoleAssignment/Responsibility/Caseとの同一World連結を既存#1180/#2514の独立Expectedへ渡す。合成r7と別Case Fixtureを同じWorldとして便宜的に結合しない。
- **Go回帰の現在資格**：#4796はmain統合済みの合成二Context・正Path/partial/NULL回帰。旧HEADのgofmt/Health失敗を実DBやCase C1の達成に読み替えない。正式Reader→Rule意味受理→実Go生成／Builderは#2621/#988/#1121/#4544の別Product資格。

**P0**：通常`tei` CLI初回実Provider Read #2586およびStage1→Stage2 Desktop #2831は独立に継続。本文は採用前の設計・実装計画であり、新しい実組織DB/CLI/Host/Studio検証・Owner意味受理は未達。新Core/Temporal Engine/Host/Expected正本を追加しない。


<!-- org-c1-asof-sql-input-parity-20261115 -->
### 2026-11-15 組織C1：Query宣言とSQL入力の実整合をProductで検証する（補遺§36）

[組織C1補遺§36](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)で、main上の`as_of_membership.tei`に存在した具体的な不一致を解消する。**宣言は`valid_at/known_at`、SQLと旧物理テストは`valid_from/known_from`**であり、同じQueryの構造試験・実バインダ・隔離PG試験が異なる引数名を正と見なしていた。SQLの`valid_from/known_from`は行の区間開始列、`valid_at/known_at`は評価時点の入力として分離する。

- **修正と回帰**：既存Query SQLの時点パラメータを宣言済み`valid_at/known_at`へ統一。既存Goバインダ試験と使い捨てPG opt-in試験も同名入力に変更し、旧区間名を引数に渡す負例を追加。Datasource/Result Mapping/物理列/PK/半開区間条件は維持。
- **受入**：固定HEADの宣言・SQL構造、実PostgreSQLバインダ、隔離DBでの二時点結果を別々に証明。CIの`go test`だけではopt-inの実DB PASSを主張しない。Source/Query Inputを受理契約の正本とし、Controller・CLI・Studioが独自Aliasを正として実行しない。
- **未解決・Owner**：#2386/#1126のPerson Linkと版識別、同一Membershipの訂正旧新版保存、#2586/#4490の一Attempt実Cohort、正式`*.tei.md` Reader→生成Go、現在Grant/Caseは別資格。#1180/#2514の既存Expectedを使い、新Alias Registry/Host/Temporal Engine/第二正本は作らない。

**優先順位**：通常CLI実Provider初回Read #2586とDesktop Stage1→Stage2 #2831は変更しない。この修正自体は実組織人物数やCase承認の完成を示さない。


<!-- org-c1-duplicate-revision-real-pg-20261115 -->
### 2026-11-15 組織C1：訂正版共存不能を実PostgreSQLで検証し、MVR-02の正例へ引き渡す（補遺§37.4）

[組織C1設計§37.4](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)に、**同一Membership A2の旧版と訂正版を現在の単独PKでは並存保存できない**ことの実DB証拠を追加した。固定HEAD `8cf10feb` の[隔離PostgreSQL run 43681066038](https://github.com/hat47x/tei-platform/actions/runs/43681066038)はsuccess。SQLSTATE `23505`の一致・旧行が一件残る負例、従来の二時点境界4例が明示PASS。Queryの宣言InputとSQL Placeholderの集合一致もGo回帰に加え、現在main上の両Goファイルと設計§37のblobが固定HEADと一致することを確認した。

- **実証したこと**：現行`membership_id`単独PKによる同一Occurrence訂正版の挿入拒否。失敗後の旧版保持。隔離DBでの既存のas-of境界。これを「二時点履歴保存機能が完成した」と言い換えない。
- **次の受入条件**：#2386/#1126がMembership occurrence、訂正・版ID、Person Link/兼務の意味を選ぶ。#1180/#2514/#4490へMVR-02の**旧新2版を共存保存してknownAt別に再読込する正例**を渡す。複合キーや履歴表は比較候補であり、採用済みの製品仕様ではない。
- **証拠の扱い**：元[PR #4866](https://github.com/hat47x/tei-platform/pull/4866)はclosed/unmerged。コードと組織設計の現行mainに同じ内容があることと、PRがマージされたことは別の主張として記録する。一般Repository Health、通常tei CLI→実Provider→独立Consumer #2586、正式`*.tei.md` Reader/生成Go、Studio Stage2 #2831は独立P0。


<!-- org-revision-key-candidate-compare-20261115 -->
### 2026-11-15 組織C1：訂正履歴の物理キー候補を実測し、業務意味の採否へ返す（補遺§38）

[組織C1設計§38](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)で、現行`membership_id`単独PKの訂正版共存不能（隔離PG負例 [run 43681066038](https://github.com/hat47x/tei-platform/actions/runs/43681066038) 実PASS）から、**履歴版を保存できることと、knownAtで一つの版に解決できることを分ける**検証へ進む。

- **実験用の二版正例**：製品Persistence/Queryを変更せず、既存使い捨てPGに候補`(membership_id, known_from)`の一時テーブルを作る。A2旧版と新版を保持し、境界前後に別の配賦を一行ずつ読み出せるかを確認する。
- **区間重複の負例**：同じA2の第三版がknownAt上で旧新版と重なっても複合PKだけでは拒否されず、同一時点に異なる配賦の二版が該当することを確認。任意一行の選択や上書きを正解としない。隔離PGの実PASSまでは予想結果に留め、合成GoやSQL文字列検査から実証と宣言しない。
- **Owner採用判断**：#2386/#1126がMembership occurrence、版ID、訂正・取消・再所属、Person Link、兼務と区間非重複・同時更新の意味を受理する。複合PK、版ID分離、現在行＋履歴表の候補を比較し、選んだ方式だけを正式`*.tei` Persistence/Queryへ反映する。独立Expectedは#1180/#2514/#4490のMVR-01〜05を再利用。
- **実製品への橋渡し**：意味受理後、同じA2の二版を実DBへ保存・同一validAtでknownAtを切り替えて実Provider→Function Context→独立Consumerへ渡す。Read Cohort/Row Coverage/Current Grantと正式`*.tei.md` Reader・生成Goは別資格。CLI実Provider初回Read #2586とStudio Stage1→Stage2 #2831を独立P0として維持。

**実装状態**：今回の複合PKは検証用テーブルの候補であり、TEI標準Schema・新Temporal Engine・第二Expected管理・製品業務Ruleとして採用していない。


<!-- org-revision-key-pg-evidence-20261115 -->
### 2026-11-15 組織C1：履歴キー候補の実DB比較を受入結果へ確定（補遺§38.4）

[設計§38.4](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)と[PR #4880](https://github.com/hat47x/tei-platform/pull/4880)（main統合 `c6ebebf2`）で、隔離PostgreSQLの一時テーブルによる`(membership_id, known_from)`候補の比較を実施。固定HEAD `6fe73a1a`の[実PG run 43684287479](https://github.com/hat47x/tei-platform/actions/runs/43684287479)は**success**：同じA2の訂正旧新2版の共存、既知時点境界での読み分け、区間の重なる第三版による二つの異なる配賦の同時成立を検査した。CIの実行対象と必須PASS判定にも新試験を追加し、未実行の成功扱いを防いだ。

**採用しないまま残す条件**：複合主キーだけでは時点区間の非重複を保証しない。#2386/#1126が業務同一性・版識別・訂正／取消／再所属・Person Link・兼務・競合裁定を受理してから、正式Persistence・Query・実Provider→独立Consumerの正例へ接続する。今回の一時テーブルを製品Schemaに昇格させない。#2586の通常CLI実Read、#2831のDesktop Stage2を独立P0として維持し、全体Repository Health・Host Grant・Case/REAとのLinkは別資格とする。

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


### 2026-11-15 組織C1：二時点の重なりを直積で検証する（補遺§39）

[組織C1設計§39](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)に、複合主キーの版保存だけでは区間競合が防げない§38の実測を引き継ぎ、**valid期間とknown期間がともに重なる場合の版競合**と、**known期間が重なってもvalid期間が隣接する正当な変更**を区別するORG-BT-01〜05を追加した。単一Occurrenceに一版しか成立しないかは業務Ownerの選択であり、Personの兼務・別Membershipとの重なりを禁止しない。

隔離PostgreSQLの一時テーブルで二軸の正負例を比較する `TestOrgAxisAsOfBitemporalRectangleCandidateOptIn` を用意し、専用CIの必須PASS対象に加えた。**隔離PGでの対象試験は後続PR #5027の固定run 43707188012で明示PASSしたが、Owner意味受理・製品Persistenceへの採用を意味しない。** #2386/#1126の訂正と異動・版IDの意味判断後に、#1180/#2514/#4490のMVR-02/03を実Provider→二Consumerで検収する。#2586の通常CLI実Readと#2831のDesktop Stage1→Stage2は引き続き独立P0とする。


### 2026-11-15 組織C1：実PostgreSQLの確認結果を正式Sourceの受入順へ接続（補遺§40）

[組織C1補遺§40](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)と[段階計画§10.15](design/ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md)に、二時点の履歴版・人物との関係・正式Query・二Consumerの検収を具体化した。PR [#5027](https://github.com/hat47x/tei-platform/pull/5027)はmain統合済み（commit `a8d3c4f8`）。[隔離PostgreSQL run 43707188012](https://github.com/hat47x/tei-platform/actions/runs/43707188012)で、単独PKの版保存拒否、試験用複合PKの旧新版保存と区間重複、**有効期間×認識期間の二軸競合と別Membershipの同時共存**が明示PASS。一時テーブルでの物理比較であり、正式Persistenceの履歴保存・Person Link・実Cohort/Consumer・業務Ruleを受理したものではない。全体Repository Healthの失敗は別票。

**次の受入順**：#2386/#1126がOccurrence/Revision/Person Link・訂正と異動・単一値性を選択→受理した正式`*.tei` Persistence/Queryで同じA2の旧新二版を保存・再読込（MVR-02）→同じvalidAt/異なるknownAtとSource/Scope/Coverageを実Providerで照合（MVR-03）→Function Context/二Consumerの同版一致→BRMS/Case・Studio/SHO・Host Effectへ別資格で引渡し。外部REA/AIの候補、known-timeとrecorded-at/observed-at、通常Data閲覧と実Host Grantはそれぞれ独立のまま。**優先P0の通常CLI実Read #2586とDesktopセルフビルド #2831を妨げる新EngineやGateを追加しない**。

### 2026-11-15 組織C1：二時点競合の保存時拒否を隔離PGで比較する（補遺§41）

[組織C1補遺§41](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)と[段階計画§10.16](design/ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md)で、**読取時に曖昧さを発見する能力**と**保存時に競合版の混入を防ぐ能力**を分離した。既存の隔離PostgreSQL実測（PR #5027）では試験用複合キーだけでは二時点の競合を拒否できなかったため、同じMembershipのvalid/known両区間が重なったときだけ拒否するPostgreSQL GiST EXCLUDE候補を追加する。正当な認識訂正・有効期間切替・別Membershipの同時存在を保つこと、SQLSTATE 23P01と23505の識別、失敗後の再読込、同一トランザクションで旧known期間を閉じて新版を追加することを独立Expectedとする。

**合格上限**：新候補は一時DB内の物理制約の比較であって、製品の正式Persistence、業務Ownerが受理した単一値性、同時Writer/障害復旧、完全なRead Cohort、二Consumer、Host Grant・Effectの実証ではない。業務意味・三同一性（Membership occurrence／Assertion revision／Provider row）は#2386/#1126、正式Provider・Queryは#2586/#2622、Rule・Caseと現在実行は#4490/#4501/#1174が担当する。S3を選ばない簡潔モデルへTemporal Engineを強制せず、標準Graph/BRMS/REA/Studio/Pluginへ共通の意味・取得資格だけを引き渡す。**通常CLI実Read #2586とDesktopセルフビルド #2831を独立P0に保つ。**


<!-- org-bitemporal-two-writer-20261115 -->
### 2026-11-15 組織C1：二時点の保存時排他を同時Writerまで検証する（補遺§42）

[組織C1補遺§42](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)に、main統合済み[PR #5063](https://github.com/hat47x/tei-platform/pull/5063)の**隔離PostgreSQLでの明示PASS**（run #43724543096）を記録した。二つの時間区間の重なりを使うGiST EXCLUDE候補について、正当な期間交替、別Membership共存、23P01競合拒否、SAVEPOINT復旧は一トランザクション内で確認済み。一方、二つの接続が同時に書き込むときの待機・Commit後の競合判定は別資格である。

次の固定実測は、隔離DBの専用Schema上に共有テーブルを作り、A2の第一Writer未Commit中に第二WriterがDB Lock待機へ入ることを観測する。第一Writer確定後の第二Writerが23P01で拒否され、勝者の版だけが残り、境界で接する訂正版・別Membershipを引き続き保存できることを確認する。**実測更新**：[PR #5099](https://github.com/hat47x/tei-platform/pull/5099)はmain統合済み（`347d7d24`）。[隔離PostgreSQL run 43727567803](https://github.com/hat47x/tei-platform/actions/runs/43727567803)で、二Writerの実Lock待機、勝者Commit後の23P01拒否、隣接する訂正版・別Membershipの保存を明示PASSした。実験用Schemaに限定したObservedであり、正式Persistence・他Provider・Host Effectの受入とは別。同時Writer試験も正式Persistence・HostのEffect／Receipt、Ownerの単一値性受理にはならない。

業務意味は#2386/#1126、Product Persistence/Providerは#2622/#2586、業務Rule・Consumerは#4490/#2514、現在Grant/Effectは#1174/#2499で別検収する。PostgreSQL固有のExtension/IndexをTEI Core／DSLへ強制せず、標準Data/Relation/Queryから必要なProvider実装を選ぶ。通常CLI実Read #2586とStudio Desktopセルフビルド #2831を先行P0のまま維持する。


### 2026-11-16 組織C1：意味IDで識別する抽象経路と状態保存テスト（補遺§43）

[組織C1補遺§43](design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)と[段階計画§10.18](design/ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md)で、コード生成と図表解釈の両経路に共通する**型付き抽象経路の検収**を追加。識別基準は生成Goメソッド名ではなく、Source/World Revision、Semantic Data/Relation ID、Relation Key・方向・多重性、Query/Function ID、必要なSource/Rule/Read Cohort/Scopeと不変条件。結果として残すのは、経路の途中で保持すべきOccurrence/Assertion revision/validAt/knownAtとpartial/unknown、さらに各Consumerの受信と現行Host Grantを混同しなかった証拠。

第一段階のGo回帰は `TestOrgC1AbstractUnitReadPathUsesStableSemanticIDs` で実 `org-staged-axis` を読み、Membership→OrgUnitの宣言上の型付き辺とas-of Query結果の対応を検査する。表示Symbolの改名は意味不変、Target/Key/Query/Datasource/Resultのすり替えは拒否する。**IRテスト単体ではGenerator/実Provider/Relation traversal/二Consumerの実接続は未証明**。次は既存#1180/#2514/#4490のExpectedへ経路条件を加え、同じWorldの実Builder→実Provider→Function Context→二Consumerへ検査範囲を拡張する。

Pluginのport連結は基本型だけでなく複合Data・Relationの型も照合し、必要な変換は明示Adapterで表す。Studio/SHO/REA/AIのProjectionにおいて別Person・別Relation occurrence・別時点・別認可を同名表示から推定しない。新しいTrace Store/専用Test Engineを設けず、#2586通常CLI実Readと#2831 Desktopセルフビルドを独立P0に保つ。

**今回の検査を強化**：Relation Keyは文字列の包含判定ではなくSource/Targetの型付きCanonical IDを厳密照合する。QueryのInput名・型、Resultの生列名→Semantic属性も検査し、宣言順の変更は許容、方向逆転・非出力依存の欠落・誤対応は拒否する。Source/IRの静的到達可能性と実行時の通過、途中状態不変性は別資格。汎用評価と生成Goの同値検証の先例としてmain統合済み[PR #5077](https://github.com/hat47x/tei-platform/pull/5077)・[PR #5087](https://github.com/hat47x/tei-platform/pull/5087)を参照し、組織C1のGenerator/Provider/二Consumerは別途実測する。

<!-- org-ar1-realpg-review-20261116 -->
### 2026-11-16 組織C1：型付き抽象経路を同じ実PostgreSQL試行へ接続する

[PR #5115](https://github.com/hat47x/tei-platform/pull/5115)は未マージで閉じられたが、型付きRelation/Queryテストと計画§43/§10.18の内容は現在のmainで確認できた。一方、同PRの[隔離PostgreSQL CI run 43733717372](https://github.com/hat47x/tei-platform/actions/runs/43733717372)は、Go整形差分によりテスト実行前に失敗した。したがって旧runをAR-1/3の合格に使わない。

今回の修正では①該当Go試験の整形を修正し、②既存の`TestOrgAxisAsOfActualPostgresOptIn`で**同じWorkspaceのMembership→OrgUnitの型付きRelationと実Queryの名前付きInput/Result契約を確認した後、そのQueryを実PGで実行**する。新しいExpected・Provider・Runtimeを追加しない。これによりSource/IRと実Queryの同Attempt照合に進むが、OrgUnitの実JOIN、生成Go Artifact、正式な履歴版保存、Person Link、Function Context/二Consumerまでの状態保存は別試験である。

実PGの対象テストが固定HEADで明示PASSしSKIPがなければ、§43.5/段階計画§10.19のAR-1→AR-3接続を限定合格とする。AR-2（生成/解釈と実Builder）・AR-4（同一Read CohortをContextと二Consumerへ伝搬）・AR-5（Rule/現在Grant/Effect）を先回りして完了扱いにしない。CLI/MCP/DesktopやPluginの共通Meaningは維持し、選択・含有・実呼出し・Consumer受領を独立検収する。

通常CLI初回実Provider Read #2586とStudio Desktop Stage1→Stage2セルフビルド #2831を引き続きP0に保つ。

<!-- org-ar1-realpg-pass-20261116 -->
### 2026-11-16 組織C1：同一Attemptの型付きSource/Query→実PG Readを限定合格へ更新

[PR #5155](https://github.com/hat47x/tei-platform/pull/5155)をmainへ統合（`0f5b2390`）。[run 43859032163](https://github.com/hat47x/tei-platform/actions/runs/43859032163)で `TestOrgC1AbstractUnitReadPathUsesStableSemanticIDs` と `TestOrgAxisAsOfActualPostgresOptIn` が明示PASS。後者は同一Workspaceの型付きRelation/Query契約を検査してから**そのQueryで実PostgreSQLを読み**、既存の時点・配賦Expectedと照合した。Go整形と非SKIPも通過。

**合格範囲**：AR-1静的Source/IRとAR-3の限定実Query Readの接続。**未合格**：OrgUnit JOIN、Person Link、同一A2の正式旧新版保存、AR-2生成Go/Interpreter→実Builder、AR-4同一CohortのFunction Context・二Consumer、AR-5業務Rule・現在Host Grant・Effect。次は#2586通常CLI実Providerを同一AttemptのFunction/Query選択・入力/出力・Consumer受信まで接続する。#2831 Studio Desktopセルフビルドは別P0のまま。self-build専用CIの整形失敗も別管理とし、実PG成功を製品全体のPASSにしない。
