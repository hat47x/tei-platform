# Web標準UI・コンポーネント・AI開発ハーネスの設計

- **文書の性格**：実装前設計・技術比較と受入の引継ぎ。個別技術の製品採用、Ownerの業務意味受理、実OS試験合格を主張しない。
- **対象**：TEI StudioのWails v3 Workbench、Web/Desktop共通UI構成キット、TEIから生成するUI、生成AIによる開発と検証。
- **正本と担当**：全体の順序は `plan/MASTER_PLAN.md`、UI構成キットは #997/#1000、Studio実Windowは #1824、Stage1→Stage2は #2831/#4583、操作とHostの契約は #1158/#951、独立Expectedと変異試験は #1180/#2514。ここでは新たなHost、Parser、Registry、業務Meaningの正本、独立Gateを設けない。
- **最終照合**：2026-11-15。現行WorkbenchはNode >=24、TypeScript、esbuild、CodeMirrorを利用。Wails v3 beta.29から安定版への移行は #1824 S0-Uの別実証。テーマ試作PR #4628はclosed/unmergedであり、main採用済みとは扱わない。

## 1. 今回の判断

UIを「TEIの宣言的意味／操作契約」「Webにおける表示・操作」「OS/Hostの接続」に分割する。**HTML・CSS・Custom Elements・ES Modulesなど標準技術を優先**し、汎用的な表示部品の外部契約にはWeb Componentsを使う。ただし全画面をCustom Element化せず、SUIの高密度Canvasや既存React Starter、CodeMirrorの適切な利用を妨げない。

**Lit + TypeScriptは第一比較候補であり、採用決定ではない**。Vanilla TypeScript、Lit + JavaScript/JSDoc（`allowJs`/`checkJs`）を同一仕様で比較する。比較前にコンパイラ・ランタイム・CSS-in-JSの導入を前提化しない。人間がDOM構造と状態の流れを追えること、AIが変更範囲を絞れること、静的検査と実機テストが欠陥を検出することを優先する。

採否を判断する共通指標は、(a) UI仕様から実装と契約を追えるか、(b) 型・属性・イベント・CSS・アクセシビリティの検査結果、(c) AIによる変更と手修正の量／不具合検出率、(d) 実WebView上の動作、(e) Build/配布閉包の依存と再現性、(f) TEIによる生成割合の増加と定義体の簡素化、である。TypeScriptの行数だけで優劣を決めない。

## 2. Google Modern Web Guidanceの役割

2026-11-10に正式版1.0が公表されたModern Web Guidanceを、Web標準の最新パターン・ブラウザ互換性・アクセシビリティ・フォールバックを調べる**外部知識基盤**として位置付ける。出典：
- Google: https://developer.chrome.com/blog/modern-web-guidance-v1
- 公式Repository: https://github.com/GoogleChrome/modern-web-guidance
- Wails v3公式: https://v3.wails.io/faq/ 、 https://wails.io/status/

エージェントは新しいAPIや旧式実装の置換を提案する際に該当ガイドを検索・参照する。しかし、Skillの回答は仕様・実装・受入の証明ではない。**TypeScript Compiler、ESLint、Stylelint、コンポーネント契約、Playwright、Wails実機検証**で独立に確かめる。GoogleのスキルをTEI独自仕様に複製せず、TEI固有の適用・棄却条件だけを短い`AGENTS.md`から参照する。

CIで`@latest`を毎回実行しない。採用する場合、依存物の正確な版／取得元・Digest／ライセンスを固定し、更新は検証付きPRに限定する。プライベートなプロジェクト名や検索語の外部送信を避けるため、テレメトリーを初期設定で無効とする（公式の`DISABLE_TELEMETRY=1`）。Skill検索結果をそのまま実行許可や設計承認へ昇格させない。

## 3. Web標準とCSSを使う範囲

1. **構造とアクセシビリティ**：適切なHTML（button、label、dialog、popover等）とDOMの意味を先に選ぶ。ARIAはネイティブの意味を補う場合のみ使用。フォーカス、キーボード、IME、スクリーンリーダー、縮小画面を最初から受入対象とする。
2. **レイアウト**：通常はCSS Grid/Flexbox、必要時はSubgrid/Container Queriesを利用する。固定座標はGraph/Canvas等の正当な用途に限定。レイアウト測定のためのJS常時ループを原則避ける。
3. **デザイントークン**：色・タイポグラフィ・余白・密度・角丸・動きの意味別トークンを正本として共有し、CSS Custom Propertiesへ生成する。TEIの業務意味と表示トークンを混ぜない。公式Design Tokens形式は相互運用候補とし、特定の変換器を標準化する前に差分・再生成を比較する。
4. **優先順位**：`@layer reset, tokens, base, layout, components, utilities, overrides;`を候補基線とする。ただしLight DOMのアプリ全体とShadow DOM内部のカスケードは別。レイヤー外の通常宣言・`!important`・高詳細度セレクタによる無秩序な上書きをLint・レビューで検出。
5. **コンポーネント境界**：Shadow DOMは独立した入力部品等で選択的に使用する。外部のスタイル契約はトークンと`::part()`、構造拡張はslot、イベントはCustomEventで提供。すべてをShadow DOMへ閉じ込めない。キーボード・フォーム連携が必要ならその公開契約をテストする。
6. **新しいCSS API**：`@scope`、Anchor Positioning、View Transitions、`:has()`、CSS Nesting等は用途と対象エンジンごとの互換性を確認。未対応なら軽量なフォールバックか代替デザインを記録。最新Chromeのみの成功で採用しない。
7. **媒体と複雑描画**：Web/Desktop/狭幅・高DPI・拡大表示と、必要に応じて印刷・PDF生成を分ける。SUI/GraphのCanvas/SVG、仮想表のデータ量・スクロール、ドラッグ中のフレーム時間はUI別の専用性能試験を設ける。毎pointermoveでGo/WASM/Server往復を要求しない。

## 4. コンポーネントとOperationの公開契約

一つの部品について、Properties/Attributes、Events、Slots/Parts、アクセシビリティ、対応するTEI Action/Result Adapterを別々に記述する。Custom Elements Manifest（CEM）は**部品の公開構造を発見するための導出メタデータ**であり、TEIの業務契約や権限を記述する第二の正本ではない。

- 属性は文字列シリアライズとプロパティの型・欠損・既定値を区別する。
- イベントは公開名・payload・bubble/composed・キャンセル可能性を契約化。Shadow境界を越えない非公開イベントを業務操作へ誤接続しない。
- UIは「注文確定要求」等を通知し、上位Adapterが現在のTarget/Grant/Attempt/Resultと対応付ける。Wails Service呼出しを表示部品へ直書きしない。
- 非同期結果にはWorkspace/Subject/Selection/Pin/Request/Source Revisionの適切なスコープを持たせ、後着・A→B→A・失効した応答は現在のDOM/ARIA/操作可否に採用しない。Client側の二重クリック抑制は、ProviderのIdempotencyやEffectの不存在を証明しない。
- 参照実行、実Provider Effect、Receipt、独立Read、Ownerの業務完了を別表示。現在の開示資格のない内容をDOM/ARIA/テストログ・AI入力へ漏らさない。
- 生成物には選択した部品Contract/Provider版・Theme Token版・CSS/JS Asset Digest・Wails/Browser Target・Root/Sourceを記録する。Bundleのbyte一致だけで意味受理や実行権限が復活することはない。

## 5. テスト対象とAIの開発経路

既存の`AGENTS.md`とハーネス役割分担の下で、対象UIの契約→最小差分→静的検査→ブラウザ検査→実Wails検査→独立Consumer/Owner判定の順に進める。専用の新Harness・Gate・Evidence Storeを作らない。

| 検査 | 実装候補／観測 | 不合格にすべき典型例 |
| --- | --- | --- |
| TS/JS | 現行TypeScript Compilerのstrict検査、JSの場合`checkJs` | 契約と異なる属性・イベントpayload、null/undefined混同 |
| 構造・スタイル | ESLint、Stylelint、必要なら`lit-analyzer`、Baseline互換性検査 | 未公開のDOM内部依存、CSS優先順位の例外、WebView非対応API |
| 公開契約 | CEM出力とTEI Component Contractの差分検査 | 削除されたEvent、型のずれ、Target非互換 |
| 挙動・A11y | Playwright・axe-core・手動IME/読み上げを組合せ | フォーカス消失、キーボード操作不能、狭幅での重要操作の隠蔽 |
| 画面・性能 | ビューポート差分と実測、必要な場合はレイアウト負荷 | テーマ変更で入力消失、重い表／Canvasが操作不能 |
| 配布・実Host | 正規Source→Frontend build→Go embed→実Wails Windows WebView2、後続でmacOS/Linux | 参照Sourceだけ直し配布Assetが旧版、Browser PASSをDesktop PASSと表示 |
| TEI生成 | 宣言→出力HTML/CSS/JSと独立Expected・変更前後の手書き量 | Goへの単純delegateのみ、見かけ上の生成量増加、旧Meaning採用 |

同一評価課題を(a)通常の部品、(b)テーマ/密度、(c)インスペクタ、(d)ツリーまたは仮想表、(e)Canvas・専用Native、(f)TEI生成フォームの順で扱う。最初は**一つのインスペクタをVanilla TS／Lit TS／Lit JSで比較**し、その後に既存Workbenchの状態を消さないテーマ変更とデータ量の多い部品を評価する。React Starterは廃止対象ではなく、同じTEI Action契約を使う比較Consumerとする。

既存#1180/#2514の試験票へ、次の**独立Expected候補**を追加する（新しい正式Gateではない）。
- **WUI-01**：未対応CSS featureを検出し、適切な代替を選択したか。
- **WUI-02**：公開Event／属性変更を契約差分が検出するか。
- **WUI-03**：Workspace A→B→A／後着返信で旧結果を表示しないか。
- **WUI-04**：明暗・密度・狭幅・拡大・保存不能でも入力状態を保持するか。
- **WUI-05**：実Source/Node buildとWails埋込BundleのDigestが同じか。
- **WUI-06**：Browser／Windows WebView2／macOS WebKit／Linux WebKitGTKの観測を混ぜずに報告できるか。
- **WUI-07**：CEM取得やUI活性化を業務上のAuthorizationと誤認しないか。
- **WUI-08**：TEI定義の簡素化と生成・直接編集部分の比率を、実際の差分で比較できるか。

Known Mutant例は「旧返信を現在のDOMへ適用する」「非公開EventでOperationを起動する」「Mainと埋込Assetを別世代にする」「Reader errorを0件として表示する」。生成した実装と同じ生成AIにExpectedの正当性を自己採点させない。

## 6. WebViewの互換性・配布の閉包

Wails v3はWindows WebView2、macOS WebKit、Linux WebKitGTKを利用する。Baselineは候補を絞る指標にすぎず、配布対象OS/Runtimeの最小・実測版を別に記録する。Wails v3.1.0安定版と現行beta.29の移行は#1824 S0-R/S0-Uを通し、当設計の技術比較を移行完了扱いにしない。iOS/Androidは実験的検証に限定。

UI資産は現行のReference Source→Node build→Wails埋込資産という正規経路を尊重する。生成済み`cmd/tei-studio/assets`を独立編集して再現可能性を壊さない。スタイル／Component Pluginの選択を加える際も、Build専用の生成器と、生成製品に含めるRuntime部品を分け、#951/#1016のexact版Lock、Target互換、閉包、更新時の案件差分保持に接続する。Wails Updaterの採否・子Backend・Standalone複数Imageは#2044/#4635の既存計画へ委譲し、UI方式の決定と結び付けて先行採用しない。

## 7. 実装順と停止条件

| 順 | 最小成果 | Owner／止める場所 |
| --- | --- | --- |
| **P0-A 規約** | Modern Web Guidanceを外部参照とし、Web標準・CSS・CEM・静的検査・フォールバック規約を追加。検査が動く最小Fixtureと既存TS/esbuild経路を保持 | #997/#1824。規約だけで製品実装済みとしない |
| **P0-B 比較** | 一つのインスペクタをVanilla TS／Lit TS／Lit JSで独立Expectedに照合し、読みやすさ・CI・実Browserを比較 | #997/#1180。Lit/TS採用は結果確認後 |
| **P1-A 共通部品** | Tokens、Theme、CEM、公開EventとAction Adapterの契約、案件Layerの更新追従 | #997/#1000/#1016。旧#4628は未マージ |
| **P1-B 実Studio** | 正規生成Assetを実Windows WebView2で受領し、既存Workbench/Pin/IME/フォーカスを保つ | #1824/#4583。実Desktopの試験前は未受理 |
| **P2-A 生成** | TEI定義→UI構造／CSS／動作／テストと実生成物を照合。CLI先行の最小部品からStudioに再入 | #2831/#4583/#2621。単純delegateとStage2再構築を区別 |
| **P2-B 発展** | 高密度Canvas・表／複数Window・Mobile探索を対象能力に応じて別受入 | #993/#259/#2044。P0のHard dependencyにしない |

## 8. TEI Studioの実Sourceから実表示までを一つの変更で検収する

UIの品質はCSS・型の合格だけでは決まらない。Studioで必要なのは、**同じ実Sourceから、宣言的な表示構造、対象コンポーネントの契約、生成Asset、実OS WebView上のDOMまでが一貫していること**である。これは既存のStudio #1824、Desktop製品契約 #4498、Self-build #2831/#4583、UI構成キット #997、検証 #1180/#2514に分担し、UI専用の第二ParserやAuthorityを作らない。

### 8.1 五つの成果と四つの停止線

| 受領するもの | 同一性と観測範囲 | 異なるまま保持する条件 |
| --- | --- | --- |
| **S: Source** | Workspace Root、元byteとSpan、選択Frontend、Source/Model版、Saved/Dirty | 参照Source表示≠現在の未保存Buffer≠意味受理 |
| **C: Component Contract** | 公開Property/Event/Slot/Part、Target、exact Component/Plugin版、CEM導出 | CEM生成成功≠TEI Actionの意味・Grantの受理 |
| **B: Build Asset** | 生成されたHTML/CSS/JS、Token版、Lock、入力・出力Digest、Build Toolchain | CSSコンパイル成功≠生成物の配布・利用 |
| **R: Runtime Image** | Wails Binary/Go Embed/Target OS・WebView版、Release閉包 | Source build成功≠実Windows Window起動 |
| **D: Display/Interaction** | 現在のSubject/Selection/Pin/Request、表示されたDOM/ARIA、Consumer受領 | 表示成功≠実Provider Effect≠Owner Business Verified |

最初の実証では、既存`semantic-capability:sales.order.submit`の読み取り専用Inspectorを題材に、現在のCLI Inspect/Source Trace→Studio Presenter→正規Workbench build→Go埋込Asset→実Windows WebView2の**同一Subject/版の往復**を観測する。現在mainでSource Locationの純粋な投影が成立していても、S-OPEN（原Sourceの再読取）や実配布HTMLのPin対応まで合格したことにはしない。ここでは新しい実行Operationを追加しない。

**停止線**：SとCの接続不一致なら候補UI生成を停止、CとBの不一致なら古いComponent資産を配布しない、BとRが異世代ならDesktop受入不可、RとDで現在のWorkspace/Pinが一致しなければ結果の採用を止める。失敗を`not-run`や`unknown`へ丸めず、どこまで観測できたかを残す。

### 8.2 未保存状態とAI編集の境界

Studioの編集には、(1)保存済みCanonical Source、(2)未保存Buffer、(3)Source位置付きInspector/Graph、(4)実Builderが消費した入力、(5)起動中Windowの表示という別の寿命がある。AIがCSSやコンポーネントを変更しても、未保存Bufferを黙って保存しない。編集対象のWorkspace/Subject/OccurrenceとSource Revisionを固定してからDiffを示し、採用前に版を再照合する。Pin中の表示対象と作業中のFocusは別に管理し、古い非同期返信で現在の表示や入力値を上書きしない。

CEMやLayoutが導出された場合も、手動で変更した案件Layerの所有権を保持する。再生成は`provide/replace/decorate`の明示拡張点とexact parent版の差分に従う。Plugin／Token／Generatorの更新によるUI変更が、accepted Data/Function/Procedureや実行Authorityの変更と見なされないよう、変更の種類別に失効範囲を切り分ける。

### 8.3 比較試作の独立Expectedと段階判定

PoCを漫然と横展開しないため、まず一つのInspectorについて次の独立Expectedを固定する。

- **同じ仕様**：Properties/Events/Slots/Parts、アクセシビリティ、狭幅・拡大時の表示、Workspace切替中の非同期結果をVanilla TS／Lit TS／Lit JS/JSDocに同じ条件で要求する。
- **同じ検査**：実行可能なCompiler/Node試験・必要なStylelint/ESLintを比較し、未導入Lintは`NOT RUN`として明示する。CEMの公開情報と既存TEI Action/Adapter契約を別検証する。
- **同じ変更課題**：独立したエージェントに「項目追加」「画面幅の変化」「旧応答の到着」の三変更を与え、差分の範囲、手修正、検出できたKnown Mutant、依存増分とビルド再現性を記録する。
- **同じConsumer**：ブラウザ試験だけでなく、正規Source→Bundle→実Wails Windowに必要な範囲を確認し、実OS未実行なら当該列を`NOT RUN`とする。

選定の判断は「短い実装」のみでなく、元のTEIモデルと公開契約の可読性、変更後の安全性、生成コード量をどれだけ減らせたかに基づく。採用前は現行Reference Workbench/CodeMirrorを一括置換しない。既存UI構成キット#1000のReact Starterは同じ業務契約で比較する別Consumerであり、Web Componentの内部実装方式を強制しない。

### 8.4 P0への具体的な接続

最初の差分は**ドキュメントの整備を完了条件にせず**、現行Workbenchの一つのPresenter/Inspectorに限定する。対象の属性・イベント・Styleを契約として固定し、同一InputのVanilla/Lit比較Fixtureを作成する。コンパイラと既存`npm test/typecheck/build`で再現性を確認した後、必要なStylelint/ESLintとブラウザ試験を小さく足し、実OS Windows受領は既存#1824/#4583の縦断に接続する。

この作業をStudio Stage1→Stage2セルフビルドの新しい前提条件にはしない。並行P0は、正式`*.tei.md` Reader一件の実消費→TEI-native Go生成→通常CLI実Provider一往復である。**UIをTEIで定義する利点が、委譲したコードの増加ではなく定義の簡素化と生成範囲の拡大として観測されること**を、後続の採否条件とする。

### 確定事項・未確定事項

**設計上の基線**：標準HTML/CSSの優先、Web Componentsの公開契約、React/Nativeの共存、意味とUIの分離、実Hostの別検証、独立Expected、既存P0の維持。

**評価待ち**：LitとTypeScript/JSDocの最終選定、Design Tokens変換器、CSS最新機能のTarget別採否、CEMとTEI Modelerの往復編集、案件別スタイルPlugin配布の実装。

**未確認の実績**：現行mainでのLit実装、Modern Web Guidance導入済みCI、PR #4628の採用、Wails v3.1.0移行、実OS全対象のCSS合格、TEI定義からのWeb Component完全自動生成とStage2実Windows Consumer。これらを本書の存在だけでPASSにしない。

## 9. Studio実装に直結する縦断計画（2026-11-15追補）

### 9.1 先に決めるのはフレームワークではなく、何を共通化するか

Web Componentsを「すべてのTEI画面の内部表現」とはしない。TEIが公開・再利用する標準の境界は、**(a) Domain/Operation、(b) UI Projection、(c) Component Public Contract、(d) Framework/Target Adapter、(e) Generated Artifact**とする。この五層を分離できれば、同じComponent Contractの下でVanilla Custom Elements、Lit、既存React Starter、Canvas/SVGなどの固有実装を選べる。

| 既存の正本・資産 | 共通化する情報 | 共通化しない情報 |
| --- | --- | --- |
| Data/Function/Process/Rule/Graph | 対象ID、Relation、操作名、入出力型、検証の責務 | 業務意味をUI部品へ書き直した複製 |
| Studio Presenter／Inspector | 表示するSubject、Source Revision、現在資格、イベントの入出力 | Backend Grant、保存済みSourceの確定、Provider Effect |
| UI構成キット #997/#1016 | 公開点、版、Target互換、Token、提供・置換・装飾契約 | 必ずCustom Elementsで描画するという固定 |
| Native Canvas／Graph Editor | 選択・配置・ポインタの局所状態、描画用Edge | 描画Edge IDを正式Relation occurrence IDへ昇格 |
| Harness／Skill | 対象をPinした変更要求、適用差分、独立Expectedと実行ログ | Agentが受理したと称する業務意味／実行許可 |

UIを形式上「標準データ構造」に載せるだけでなく、TEI標準Data/Graph/Processから**どの要素を表示へ投影し、どのEventをOperationに接続するか**を、既存Meaning/Projection/Bindingとして追えることを受入条件とする。必要がなければUI専用Graph Schema、独自Process Runner、第二のComponent Registryを増やさない。

Node-RED型のノード編集やマッシュアップは今後の有力なConsumerだが、ノードの画面上の位置・ポートの見た目・表示ラベルをFunction/Relation/Operatorの意味と混同しない。型付きPort、Direction、Multiplicity、Capability、Effectの制約が既存TEI Graph/Process契約に乗ることを先に確認する。Node固有の処理を汎用Operator/Providerとして再利用できる場合は、ノードを**一つの表示・編集Projection**として扱う。これはUI Kitの比較試験を全ノードエディタ実装へ膨らませる理由にはならない。

### 9.2 着手順：まず現在の実配布を受領し、その後で表示方式を比較

全体のP0（正式Reader→生成→通常CLI実Provider、非Plugin Studio Stage1→Stage2 Windows）は変更しない。以下は、#1824の**現在の一つのStudio read-only Consumer**へ追加する小さな作業順であり、全体P0へ新しいHard dependencyを追加しない。

| 手順 | 最小の実差分と独立Expected | 次へ進む条件 |
| --- | --- | --- |
| **W0 実配布基線の回収** | 現行mainの`reference/tei-studio-workbench/main.ts`、`presentation.ts`、`source-locations.ts`、`build.mjs`を実Sourceとして固定し、`npm ci`→`npm test`→`npm run typecheck`→`npm run build`を実行。生成`index.html`／`main.js`／`style.css`／`THIRD_PARTY_NOTICES.txt`と依存MetafileをDigestで照合し、実Wails Go Embed・Windows Windowまで接続する | Reference BuildとEmbedded Assetを別実行で観測。生成元・Target・Toolchain不一致ならP1へ進めない。Node PASSとWindows PASSは別記 |
| **W1 同一Inspectorの比較** | 読み取り専用Subject Inspectorを**Vanilla TypeScript／Lit+TypeScript／Lit+JavaScript(JSDoc)**で独立Fixtureとして比較。既存`readInspection`・`readSourceLocations`の純粋投影を流用し、同じ入力を与える。元の画面／CodeMirror／React Starterは変更しない | 同一DOM意味・キー操作・Source状況表示・イベント公開・CSS縮小の独立Expectedに達した方式のみ導入候補。Lit未導入時は比較の隔離成果だけを残す |
| **W2 Component ContractとKit** | 実部品のAttribute/Property/Event/Slot/Part、CEM投影、Token版、UI Kit公開点、Target互換を**既存#997と並走**して照合 | CEMの生成結果だけでアプリケーションAction、業務Validation、Authorityを受理せず、React/Native経路でも意味と入力が一致 |
| **W3 明示PinとAgent差分** | #4515のSubject Pinを現在Scopeで固定して「項目追加・見た目変更」のAI差分を作成。Source Revision・Saved/Dirty・Pin/Focusの別寿命を保存し、旧応答採用やUI暗黙保存を拒否 | 現在のHost Observe・Source/Pin・Disclosureに不足があれば拒否／未確認。AIの実送信先・対象範囲・現在Grantを別に検収 |
| **W4 TEI生成と製品自己再入** | 正式に選択された`*.tei.md`から最小のInspector/View／CSS／型付きAdapterの一部を生成し、原SourceのModel/Artifact/ToolchainとBundle Digestを照合。Stage1→Stage2を**既存#2831/#4583**から受領 | GoやTSへの単純delegateではなく、ソース定義体の減少、生成が担う意味ある構造の増加、実Stage2 Consumer受領を別々に観測 |

**W0はデザイン刷新やWails安定版移行の開始を意味しない。** 現在の`cmd/tei-studio/go.mod`はWails `v3.0.0-beta.29`であり、安定版へのS0-Uは別の隔離差分・実OS資格で判断する。参照WorkbenchのCSSも現行は主に`style.css`で、試作PR #4628の`theme.css`はmainへ未統合。新しいLayer、Token、Lit依存を「既にある標準機構」として検査しない。

### 9.3 最小一部品の公開契約と負例

最初の対象は`semantic-capability:sales.order.submit`の**read-only Inspect**とする。以下は契約候補の比較列であり、製品Type/APIの新設宣言ではない。

| 列 | 提供元・観測 | 不正な短絡 |
| --- | --- | --- |
| `subject` | Hostから受け取った明示Kind/IDとWorkspace相対の選択 | 表示ラベルやGraph描画IDだけで対象を確定 |
| `source` | `readSourceLocations`のbounded metadata／Index fingerprint | Source Indexの一部を全Sourceの完全性・Raw Readと称する |
| `disclosure` | Hostの現在資格とReaderが返せる公開範囲 | CEM・DOM・CSS `hidden`だけを開示制御にする |
| `interaction` | read-only select/inspectを既存Adapterへ委譲 | 表示部品から直接Commit/Save/Approveへ昇格 |
| `theme` | Token/CSS変化とコンポーネントの状態寿命 | Theme変更でEditor入力を再生成・破棄 |
| `artifact` | Build入力、実Bundle、Go Embed、実Windowの各Digest | `dist`だけ合格して埋込Assetを同一と判定 |

最低限の対照は、正常一Subject、Workspace A→B→A、異Workspace同名Subject、後着旧Reply、失効Source、`references=null`と空Array、Index partial、Reader error、権限未確認、Shadow境界Event不達、狭幅・200%拡大、フォーカス・IME、CSS Theme変更の入力保持である。Known Mutantは「表示Elementから業務操作を直接呼ぶ」「古いIndexを現在のRootへ付け替える」「不正なEvent payloadを型検査から漏らす」「CSS対応をChromiumだけで保証する」「Buildの`dist`と埋込Assetの世代を取り違える」。

一つの対象について、実Node試験、ブラウザ、Wails Server、**Windows実WebView2**、Stage2を独立列にする。Ownerの受入前にAI自身が生成したExpectedを採点基準へ採用しない。#1180/#2514に新しい大きな試験台帳を作らず、既存Scenarioへ反証対象を紐付ける。

## 10. AIハーネス・Modern Web Guidanceの導入条件

公式の[2026-11-10発表](https://developer.chrome.com/blog/modern-web-guidance-v1)はModern Web Guidance **1.0の安定版**を明示している。一方、GitHubの説明には時点やキャッシュによってpreview表現が残るため、CLIの動作・配布版はインストール時の実出力で照合する。公式ドキュメントの`search`、`retrieve`、`legacy-scan`は**提案を得る入口**であり、コード自動変更・受理済み標準の根拠ではない。

- テレメトリーは`DISABLE_TELEMETRY=1`を適用し、外部送信の事実と範囲を確認。Agent Skillのインストールは明示した隔離対象で比較し、CIで無指定の`@latest`を毎回取得しない。CIに入れる場合は正確な版、取得元、依存閉包とライセンスを記録する。
- Modern Web Guidanceの静的な指摘は、**ブラウザ・Wailsの実動作観測**で必ず検証する。ガイドのBrowser BaselineはWindows WebView2/macOS WebKit/Linux WebKitGTKの正式対応版・配布OSの確認に代わらない。Chrome DevToolsによる自動確認が可能な場合も、外部WebKit系OSの検査を加算しない。
- 変更前後を同一課題・同一Context・同一独立Expectedで比較する。比較用のコード品質検査は、現在存在する`npm test/typecheck/build`を先に使い、`lint`・`axe`・CEM Analyzer等は**必要な最小Fixtureを導入したPRで初めて実行済みと記録**する。
- Agentは対象Source/Subject/VersionとRead/Write許可範囲を固定する。下位Skillや検索結果の記述を権限にしない。実装→独立レビュー→対象CI→実OS Consumerを区別し、成功した範囲だけ報告する。

## 11. 次に切り出す実装Packet

**最初のコード変更**は、#1824の現行Workbenchから一つのread-only Inspectorの**Public Event + 型付き入力／出力**を取り出すことに限定する。既存`presentation.ts`／`source-locations.ts`の純粋投影を使い、`main.ts`のWails呼出しとDOMの交換可能な接続点を明示する。通常のビルドスクリプト・Lock・CodeMirror・既存Source Readerは維持する。内部実装をLitへ置換することを初回の完了条件にしない。

このPacketが閉じた後、別の小PRで同じ入力・Event・DOM・CSSのFixtureをVanilla TS/Lit TS/Lit JSで比較する。採用判断まで実製品のLit依存・新Generator・Token規格を固定しない。次に公開ContractのCEM/Kit適合、テーマ・狭幅・IMEの負例、実Wails Consumerを順に接合する。

同時進行の#2621正式Reader、#4495通常CLI実Provider、#2831/#4583非Plugin Stage1→Stage2を**主線**とし、W1のUI方式比較をそれらのブロッカーにしない。#4515 Pin、#2516 Disclosure、#951/#1016 Plugin/Kit、#993特殊Canvas、#1180/#2514独立ExpectedのOwnerを変更しない。業務Actionの認可、実Provider Effect、Receipt、Ownerの意味受理は必ず元Ownerへ返す。

## 12. Web UIと型付きFlowの共同受入（2026-11-15追加）

### 12.1 共通にするのは意味と接続契約であり、編集画面ではない

[Rule／Plugin第一縦断](PLUGIN_RULE_PRODUCT_VERTICAL_SLICE_2026-11.md)のN0〜N4（型付きFlowからAPIへ進む道筋）と本書のW0〜W4（WorkbenchからUI生成へ進む道筋）は、独立したエンジンを作る計画ではない。どちらも**既存Data・Graph・Operator・Function・Process・OperationとContract/Providerを正本とし、投影先と検証経路だけを変える**。

| 共有する境界 | UI側 | Flow/API側 | 共通の停止条件 |
| --- | --- | --- | --- |
| Source・同一性 | **W0** 現行Inspector/Bundleを固定して読む | **N0** 既存Rule/Operatorをread-only Nodeに投影 | 未読取Sourceを受理済みと表示しない。描画IDをOccurrence IDに代用しない |
| 型・意味・接続 | **W1〜W2** 表示部品のProperty/EventとTarget Adapterを比較 | **N1** 型付きPort/Edgeと分岐を検証し、既存GeneratorでGo生成 | UI Event名はOperation/Port型の正本ではない。NULL・欠損・部分取得を混同しない |
| BuildとConsumer | **W0の実配布／W4** Source→Asset→WailsとTEI生成を検証 | **N2** Source→Build Intent→実Provider→通常CLIを検証 | Buildの成功を実Call・実OS・Stage2の成功へ昇格しない |
| 人間・AIによる編集 | **W3** Subject Pin、Source差分、未保存Buffer、Disclosure | **N3** StudioのFlow編集、Port Binding、再生成 | Layoutのみの変更は業務意味を変えない。意味の変更では依存生成物を失効させる |
| 外部公開 | **W4後続** Target別UI Kit/Assetを配布 | **N4** HTTP Trigger/OpenAPI/Connectorを接続 | API公開、認可、外部送信許可、Effect Receiptを別資格で検証する |

この表は**依存関係の指定ではない**。W0のNode/Source局所試験と実Windows WebView2受領は別の判定列である。N1〜N2はLit採用決定やStudio GUI完成を待たずに進める。逆にWebView成功をFlow生成・実Providerの資格へ転用しない。共通の失敗例は既存#1180/#2514の独立Expectedへ渡し、第二のテスト正本を設けない。

### 12.2 一つの部品をCLI・Node・Web・APIで再利用する条件

- **定義と参照**：Operator/Function/RuleのMeaning、Portが参照するData Type、Node/Edge occurrence、Role・向き・多重性は標準モデルが保持する。Node座標・表示順・Canvasズーム・CSS TokenはUIのProjectionに限定する。Data/Control/Event/State/Error/Referenceの接続を無型の一本の線にまとめない。
- **操作と結果**：UI Component Eventは利用者の意図の通知であり、Operation実行そのものではない。既存AdapterでWorkspace/Subject/Revision/Attemptを固定し、現在Grantを確認した実ProviderだけがEffectを担う。Web ComponentsのCEMは公開構造を表す導出物であり、MeaningやAuthorityの第二正本ではない。
- **現在資格**：明示PinのIntent、Readerが返したInspect、現在のDisclosure、Provider Effect/Receipt、業務Owner受理は別の寿命を持つ。`pin-state.ts`の世代一致や`source-locations.ts`の部分Indexから権限・全件取得を推測しない。
- **生成と更新**：Kit/Theme/Renderer変更では対応するUI AssetとConsumerの失効を検証する。Operator/Port Binding/Ruleの変更ではMeaning依存の生成物を再評価する。A→B→AやGenerator/Plugin更新をbyteの復帰だけで再利用許可へ変えない。

Wagby由来の入力・検索・承認や、SUIのKJ・マインドマップ・PERT・Canvasにも同じ整理を適用する。業務Data/Constraint/ProcedureをUIへ複製せず、Native特有の描画・操作はNativeに残す。Nodeの種類別Catalogは検索・表示上の分類であり、Coreへ排他的な新Role enumを追加する理由にしない。

### 12.3 実装に着手する二つのPacket

| PacketとOwner | 最初の具体的な変更 | 独立Expectedと反例 |
| --- | --- | --- |
| **UI-P1：#1824／#997** | 現行`reference/tei-studio-workbench`のread-only Inspectorを対象に、Presenterが受け取る検証済みSubject/Inspection/Source Statusと、利用者のSubject選択要求Eventを分離する。`main.ts`のWails接続とDOM表示を交換可能な境界に置く。新しいHostや保存・Effectを実装しない | WUI-02 Event契約破壊、WUI-03 A→B→A後着、WUI-07 Eventと実行許可の混同。`references=null`とpartial Indexを全件取得と扱わない |
| **FLOW-P1：#2621／#4544** | 既存`quantity_positive` RuleをN0のread-only Nodeとして投影し、型付きPort/Graph検証からN1の生成Go一件へつなぐ。成果物は既存Builderへ渡し、Canvasから直接実行しない | NG-01 同名別Occurrence、NG-02 多重Edge、NG-03 Port型不一致、NG-05 Layout変更で業務Goが変化、NG-06 Rule変更でも旧生成Goを利用 |

UI-P1が扱うのは表示とイベントの境界、FLOW-P1が扱うのは意味を持つFlowと生成であり、**同じComponentを別名で作る二重開発ではない**。両方の接点は既存のOperation/Adapter契約。`presentation.ts`や`source-locations.ts`の正規Reader検査を迂回せず、UI Kit、Graph Editor、CLI/Pluginのいずれも独自Runner・Registry・業務モデルを増やさない。

### 12.4 採否と停止の判断

UI-P1が現在の入力保持・キーボード・IME・Source表示を悪化させるなら、Lit比較や全面Custom Elements化を先行しない。Vanillaの公開契約分離で十分ならそれを基線とする。Flowが描けても正式Node/Edge occurrenceやPort型が確定しないなら実行へ昇格させない。逆にN1→N2がCLIで先に成立した場合、Studio GUIを待たずに実Consumerとして検収する。

両方の狭い実証が閉じてから、同じMeaning/Operation Contractに接続する二つ目の独立Consumerを検証し、Web/Node/APIの表現が増えても業務意味・Authority・Effectの正本が増殖せず、Project UI Layerを安全に更新できることを確認する。それまでは「UIとFlowを統合実装済み」と記載しない。

## 13. UI-P1の最小実装と次の検証段階（PR #4949）

2026-11-15の現行mainを照合すると、Pin配線の限定修正は[#4920](https://github.com/hat47x/tei-platform/pull/4920)で統合済み、Ruleから判断Goを生成するR1は[#4911](https://github.com/hat47x/tei-platform/pull/4911)で統合済みである。一方、既存BuilderへのR2の一試行[#4934](https://github.com/hat47x/tei-platform/pull/4934)は**closed／unmerged**であり、実Builder消費を製品受入済みとは扱わない。UI-P1はこの境界を壊さず、現在の読み取り専用表示へ接続する。

### 13.1 読み取り専用の選択要求を表示部品から切り離す

W0〜W4／N0〜N4の統合方針を実コードに接続する最初の小差分は、現行Workbenchの**モデル一覧とGraphノードの選択要求を一つのUIイベントに収束**させる。対象は[PR #4949](https://github.com/hat47x/tei-platform/pull/4949)の `reference/tei-studio-workbench/inspector-selection.ts`、`main.ts`、関連試験であり、新しいHost・業務Operation・Graph Engine・UI Frameworkは導入しない。

公開イベント `tei:inspector-selection-request` のdetailは、`workspaceId`、`workspaceEpoch`、`subject {kind,id}` の三要素だけとする。イベントはバブリングしShadow境界を越えられるが、**Hostへの実Callでも実行認可でもない**。受け側は現在のWorkspace IDとepochの一致と、実際の発信元がモデル一覧またはGraphであることを確認し、既存`inspect()`の`inspectEligibility`、`readInspection`、`responses.run("inspection")`を再利用する。イベントに混入した`effect`や`grant`等をHost資格として扱わない。

| 証拠・反例 | 検査する範囲 | この結果だけでは主張しないこと |
| --- | --- | --- |
| 同一Subjectの一覧／Graph選択 | 両UI入口が同一イベントに接続され、Inspect入口を共有 | HTML部品の正式CEM採用、全WebView受領 |
| Workspace A→B→A | 戻ったAの同名Subjectでも旧epochのEventを不採用 | Source/Policyの変化をすべて検出したこと |
| NULL/不完全なEvent detail | 必須Subject、Workspace、整数epochが揃わない場合は拒否 | Source Indexの完全性、正式Relationの一意性 |
| 偽Grant/Effect付きdetail | Eventからの型付き投影で権限・副作用の自己申告を捨てる | Host側Disclosure／AuthZや実Provider Receipt |
| 正規Workbench Build | 既存のNode24、tsc、esbuild、CodeMirror、Pin/Readerの回帰を維持 | 実Windows WebView2、正規配布Asset、Stage2完成 |

本差分で追加するNode標準試験は、公開イベントの形、失効、無効値、既存`inspect()`への静的接続を検査する。局所Node v22のType StrippingとTypeScript 5.8.3単体型検査のPASSは、**製品のNode24全テスト／正規Bundle／実Windows受入のPASSへ加算しない**。PRのCIが未終了なら`NOT RUN / PENDING`を保持する。

### 13.2 次の差分は表示結果のPresenter分離に限定する

イベント入口が一つになっても、`main.ts`には`readInspection`の結果をEditor、Source status、Authority limits、Noticeへ表示する処理が残る。次のUI-P1差分は、現在のReaderで検証した**読み取り専用の表示データ**と、DOMへの配置処理を分ける。選択のIntentと検証済みのInspect結果は別の型・寿命にし、Source Indexの`partial`／`null`、不明な現在Grant、Pin中のSubjectと現在Focusの違いを失わせない。現在の`main.ts`を全面コンポーネント化せず、失敗・保留・後着の出力を既存の入力・IME回帰へ接続する。

その後、同じ公開契約を使ってVanilla TS／Lit TS／Lit JSを比較する。UI Frameworkの優劣やCEM採用は独立Expectedの結果で判断し、Node EditorのFLOW-P1/N1〜N2はその選定を待たない。**既存の#2621/#4544/#4495と#2831/#4583の実Reader・実Provider・セルフビルドを、Web UI方式の追加Hard Gateで止めない。**

## 14. 実コードと二つの記述表現をInspectorに照合する（2026-11-15）

UI-P1の公開選択Eventは、利用者が何を見ようとしているかを伝えるための仕組みである。これに業務意味・出典証拠・実行資格を入れ直さず、[実装からモデルへのdogfooding](IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md)§9の**同一判断／別Source／同一生成結果／別資格**を、既存read-only Presenterに段階的に表示する。Node View・Markdown・Graphvizは同じ構造からの候補Projectionとし、画面ごとの第二正本を作らない。

最初の一Subjectは`sales.order_line.quantity_positive`とする。表示は以下の順に分離する。

| Inspectorの表示対象 | 受け取る根拠 | 別扱いすること |
| --- | --- | --- |
| 選択中のSubject | 既存`tei:inspector-selection-request`とWorkspace/epoch | EventはIntentでありHost Inspect成功ではない |
| 受理前のSource候補 | 既存Loader／Source Index、各Reader、path/byte spanと部分取得の状況 | 別表記で同じGoでも元Sourceの位置・現在性を合成しない |
| 型付きRuleと生成Go | 既存`semantic.Validation`の型、`GeneratePositive`と実生成byte | 実Builderが選択・消費したArtifactとは区別 |
| 図・DSL候補 | Go ASTのRead/Branch/Failureと元Sourceへの対応、欠損・Loss | 図が示さないEffect、逆編集不可範囲を隠さない |
| 実行・現在資格 | R2実Builder、R3通常CLI/Provider、Stage2 Windowsの各Attemptを取得できた場合だけ表示 | UIイベントやR1生成成功からGrant・Receipt・業務受理を補完しない |

既存の`presentation.ts`／`source-locations.ts`／`pin-state.ts`を利用し、対象がA→B→Aと切り替わった場合やPinとFocusが異なる場合、古いEvidenceを現在Subjectの証明として再利用しない。Sourceが二表記に分かれるときは別々の出典として開き、どちらが編集可能かを無条件に選ばない。編集機能や保存操作はまだ追加しない。

**最初の作業は表示前のread-only ViewModelのFit/Gapである。** 既存Inspectorの契約で不足する列（Sourceの完全性、対応する生成物、実Build資格）は「未取得／未接続」と表示できればよい。新しいHost APIやGo ASTを直接ブラウザへ公開することは受入条件ではない。正式Readerの全表現採用・Code-to-DSL自動変換・CEM生成・Litの製品採用をこの小差分の前提条件にせず、既存#1824/#1124/#2621/#1180/#2514へ検証を返す。

必要な負例は、(a)同じGo byteを持つ別Sourceを同一の出典として表示、(b)説明文変更後に旧Source Indexを保持、(c)古いepochの選択Eventで別Subjectを上書き、(d)部分Indexを完全な不存在と表示、(e)生成Goの実行成功をR2/R3/Stage2合格と表示、である。この五つを検出できない段階で、図やDSLを編集可能にすることはしない。

## 15. I2M-1の観測結果をInspectorに表示する前に（2026-11-15）

[実装→標準モデルのdogfooding§10](IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md)で整理した二つの限定Go回帰は、現行mainに存在する。ただし、テスト内で得たGo AST構造を製品Hostが返せるわけではない。**テストの観測能力と、公開されたProduct Inspectの能力を混同しない**ことを次のUI-P1受入に加える。

| 表示に必要な列 | 現在の接続点 | 次の実装判断 |
| --- | --- | --- |
| Workspace／選択Intent | `inspector-selection.ts`の公開EventとWorkspace epoch | 認可の代用にせず、今の選択に対するReaderを使う |
| Sourceと適格性 | `presentation.ts`、`source-locations.ts`、`pin-state.ts`の既存read-only投影 | bounded Indexを全件とみなさず、二つの異なる原Sourceを別々に追跡 |
| 元DSLと生成Goの結合 | mainの`positive_surface_identity_test.go`／`positive_ast_projection_test.go`は試験として存在 | Product APIでは未接続。出典／生成Go側Span／意味上の対応を想像で補完しない |
| Build／実行資格 | 既存R2／R3／W0／Stage2のOwner別Evidence | Go test、Browser、Windowsの結果はそれぞれの実Consumerで独立に確認 |

次の変更は、**現行Inspectorが実際に返すデータだけを使ったread-only PresenterのFit/Gap**に限定する。Subjectの選択、Hostが返した結果、Sourceの現在性、表示要素、Pin/Focus、開示範囲を別の寿命として扱い、未接続の生成結果には「未取得」と明示する。C0のGo AST回帰をそのままWebViewへ送るための新ServiceやHost Bridgeを追加しない。

その後、I2M-1が同一Source/Rule/生成Goの対応を独立Expectedで示せた段階で、既存Provider/Studioの適切な公開点へ投影する可否を判断する。Framework選定（Vanilla TS／Lit TS／Lit JS）、Component Manifest、編集可能なGraph／Node Viewは独立した後続判断であり、**正式Reader→実ProviderとStudio Stage1→Stage2の主線を止めない**。元Sourceへ戻る矢印が一意でない場合は編集操作を提供しない。


## 16. Webの開発方式を「公開形態・実行場所・UI表現」に分けて選ぶ（2026-11-16、設計・実装引渡し）

### 16.1 WebとDesktopで共有するもの、共有しないもの

[Desktop実装指針](TEI_DESKTOP_WEB_STANDARD_UI_IMPLEMENTATION_2026-11.md)、[共通Application APIのADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)、[標準Service Flow](SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)を前提に、Webアプリケーションを「Web専用の業務実行方式」とは定義しない。共有するのは、受理済みのData／Function／Rule／Process、Operationの入力・出力・Failure・Effect、UIへのProjection、型付きPort／Mapping、Source／Build／Providerの資格である。画面を描く方法と、どこで操作を実行・公開するかは独立に決める。

- **静的公開**：公開が許されたSource/Meaningからページ・Assetを生成する。説明文・言語別URL・Head等は静的成果物に向くが、Build時に公開可能だったことは公開時点のGrantや最新情報の保証ではない。非公開Field、Secret、実行可能な内部操作を静的Bundleへ含めない。
- **対話型Web**：ブラウザは入力、閲覧、局所状態と操作要求を担い、権限を要するRead・Command・Effectは現在のServer側Operation／Providerで扱う。ブラウザから呼べるAPIを列挙できることと、実Providerが選択・収録・実行されたことは別に確認する。
- **ServerでのHTML生成**：初期表示・公開検索・多言語ルートなどで必要な場合に選ぶ実現方式。画面の意味契約を二重化せず、同じData／UI Projectionから初期HTMLと後続の対話処理を構成する。SSR／SSG／Client描画の選択はRouteごとに可能とし、全画面への一律適用を前提にしない。
- **Desktop／組込みWebView**：Workbenchと共通の表示部品・Projectionを利用できるが、Wails Bridgeとローカル権限はWeb公開APIの代用品ではない。対象OS／WebViewでの実動作、Go埋込Asset、Releaseの版を別々に検収する。

この四つは排他的な製品種別ではなく、一つのApplication内でもRoute／Entry／Targetごとに組み合わせ得る**実現・配備の選択肢**である。選択結果は既存のTarget／Build Intent／Plugin Closure／公開Contributionと照合し、Web独自のMeaning正本、万能Router、SSR Engine、BFFを先に新設しない。選んだ実現方式がG（コード生成）、P（既存Operator等の限定解釈）、D（外部Providerへの委譲）のどれかもOperation／構成単位で記録し、同じ結果が返っただけで実Callの同値やEffectの安全性を推定しない。

### 16.2 API PluginをWailsから切り離し、Webの公開責任を明示する

採用済みの[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)と、審査中の[API Plugin／CLI直結設計 PR #5096](https://github.com/hat47x/tei-platform/pull/5096)で提案されている「Operation ProviderとAPI公開Contributionの分離」を、Webの受入候補として具体化する。通常のローカルCLIはApplication Operationへ直接到達し、HTTP listener、WebView、Wailsを必要としない。WebではHTTP Adapterが認証済みActor・Audience・Disclosure・入力の完全性・実Provider選択を評価し、Wails Serverは利用できるHostの一つとして扱う。Browserからの要求やHTTP Route名をOperationの意味ID・Grantの正本としない。

公開の適格性は **Operation契約 → Provider選択 → Targetへの収録 → Route公開 → 今回の認可・開示 → 実Call → Consumer受領** の順に確認する。型が一致しても内部用RecordのFieldをそのまま外部へ渡さず、全体Record／Field単位／明示AdapterのどのBindingで何が失われ、何が公開されるかを既存Mapping・Disclosureに示す。URLや表示上のアクセス制御だけで秘密を守らない。外部Browser向けには少なくともOrigin／CORS、CSRF、Cookie／Session、CSP、認証・認可、キャッシュの共有範囲をTarget別に検査し、Secretと内部CredentialをHTML／JS／Source Map／ログへ含めない。

Clientは入力中の値、Caret、Focus、Pin、編集中のBufferを局所的に保持する。一方、保存済みSource、現在のMeaning、Provider Read、Build入力、Command／Effect ReceiptはHostが保持する版と資格を使用する。Workspace A→B→A、Account切替、Route再利用、古い非同期Reply、再接続では表示用epochだけでなく、対象・Source／要求版・現在Disclosureを再照合する。通信途絶でEffectが不明な操作をClientが自動再送しない。オフライン編集やPWAは将来の比較候補とし、競合解決・資格の失効・再送の契約が整うまでは暗黙の機能としない。

### 16.3 B2C・ノード／マッシュアップ・SUIへの適用

[B2C対応方針 #3509](https://github.com/hat47x/tei-platform/issues/3509)の公開Headや多言語URLは、公開可能なFacetをRoute／Slotへ結び付けるProjectionとして扱う。静的SEOの説明情報と、価格・在庫・申込等の時点依存のCommerce情報は同じ静的更新周期に固定しない。既存TargetにHead Slotがなければ「未接続」と報告し、専用SEO Generatorを追加したことにしない。

Node View／Flowは標準Data型・Direction・Multiplicity・Presence・Coverage・Disclosureを保った接続の表示とし、Canvas上の配線がHTTP公開・実行可能性・副作用許可を成立させたことにはしない。SUIの高密度Canvas、StudioのCodeMirror、React Starter、Web Components、Litの使い分けは、**共通するOperationと公開Component Contractを維持できるか**で判断する。Native Canvasの描画性能を理由に、意味モデルや実行器までNative専用化しない。

## 17. Web方式の次の一巡を小さく検証する（2026-11-16、P1）

初回の実装対象は新しいWeb Serverではなく、mainに存在する `reference/tei-studio-workbench/inspector-selection.ts` の公開read-only Eventと、`presentation.ts` のReader／Presenter、`main.ts` のHost接続である。既存の `inspector-selection.test.mjs` はWorkspace epoch、A→B→A、偽Grant／Effectの除去を局所試験している。まずこの既存試験を退行させず、同じSubjectについて **Host非依存の表示入力 → 表示結果 → 選択Intent** を検査する。Wails Bridgeが返せる範囲以上のSource／Generation／GrantをViewModelへ捏造しない。

| Packet | 作業と実測 | 独立反証・止める場所 |
| --- | --- | --- |
| **WEB-P1：表示とHost境界**（#1824／#997／#1158） | Inspector一部品を現行Vanilla TSで分離し、同じ入力・EventでWeb向け薄いAdapterとDesktop Adapterを比較。Nodeの型検査・試験・正規Bundle・実Windowsを別々に記録 | A→B→A旧Reply、Pin/Focus干渉、Reader errorを空集合へ変換、未保存Buffer消失、偽Grant Event。ブラウザだけでDesktop PASSとしない |
| **WEB-P2：read-only API公開**（#1158／#951／#1251） | 既存`InspectSubject`でCLI direct／Studio Hostの結果を基線に、選択済み同一OperationへHTTP Adapterから実到達する最小Consumerを比較。Method／Route／Actor・Scopeを記録 | HTTP route登録のみでProvider呼出しと主張、CLIが暗黙にWailsを起動、異なるAudienceで内部Field漏出、古いSession／Grant再利用。実HTTP未実装ならNOT RUN |
| **WEB-P3：公開Route一件**（#3509／#997／#1016） | 公開可能な説明Facetを一つ選び、Source→Head／表示Slot→実Builder入力→生成Asset Member→外部閲覧を追う。静的説明と動的Offerを別途管理 | 未選択Slotへの暗黙配置、非公開Field混入、生成Assetの旧版利用、単なるHTML生成を実Consumer受領と誤認 |

WEB-P1〜P3は並列の**P1候補**であり、正式なStudio非Plugin BuildIntent→Stage1→Stage2実Windows（#2831／#4583）と、通常CLI→実Provider Read→独立Consumer（#2586／#4495）の二本のP0へ新しいHard Gateを追加しない。新しいWeb Framework、HTTP Runner、第二のOperation Registry、テスト正本は導入しない。反例は既存#1180／#2514の独立Expectedと、各OwnerのIssueへ引き渡す。現段階は設計・実装Packetの定義であり、WEB-P2/P3の製品E2E、Web向け認証・開示、SSR、オフライン編集、別OSでの実受領は未検収である。

## 18. 宣言的UIの共通契約を、表示技術から切り離す（2026-11-16、前回統合内容の復元・具体化）

### 18.1 画面を「意味・表示・配置・接続」の組合せとして扱う

Web／Desktop／将来のMobileで再利用するのは、フレームワークごとのコンポーネント木ではない。[標準データ構造・構文・UI投影の横断設計](SEMANTIC_STRUCTURE_MAPPING_AND_DECLARATIVE_SYNTAX_VIEW_2026-11.md)、[共通APIのADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)、[Desktop Workbench指針](TEI_DESKTOP_WEB_STANDARD_UI_IMPLEMENTATION_2026-11.md)にある要素を、利用者が読む順番で組み合わせる。新しい画面専用のMeaning正本や万能UI ASTを先に設けない。

| 共通にする関心事 | 定義・利用する情報 | 混ぜないもの |
| --- | --- | --- |
| 意味・入力 | qualified Subject、Field／Relation、粒度、型、NULL／欠損／未取得、Rule、Source occurrence | DOM ID、画面ラベル、表示順、描画用Node ID |
| 操作 | 共通Operation IDと入力／結果／Failure、事前条件、作用の種類 | CustomEvent名、HTTP Route名、Wails Service名 |
| 表示への投影 | 見せる項目と順序、表・Tree・Graph・Node等の表現、読み取り専用／編集候補、開示可能な状態 | 公開Componentの形だけから推定したGrantやMeaning受理 |
| 配置・装飾 | Grid／Flex、必要なCanvas座標、密度、Design Tokens、Target別の表示調整 | 業務上の制約、操作の実行順、実Providerの選択 |
| 実装・配信 | Vanilla TS／Lit／React／NativeのComponent、静的HTML／対話型Web／Server側HTML生成／Wails AdapterとAsset | 第二の業務実行器・Parser・Registry、全Targetで同じRuntimeを強制すること |

TEIの宣言から生成すべき第一候補は、Field／Group／Order、表示条件、型に対応する入力部品、Operationの型付きBinding、公開Component Contract、静的に検査できる制約である。認可、実データの保存、外部Effectは生成されたブラウザコードが単独で受理しない。内部で手書きのGo／TS関数を呼び出しただけでは、実装をTEIへ移管した成果と数えない。UIの表現方式は必要な場合だけTargetごとの実装で選ぶ。

Node FlowのPort全体接続／Field接続／明示Adapterや、Graphの向き・多重辺・Occurrenceは既存標準型とMappingに従う。単にNodeの描画を変更するだけなら業務意味を再生成せず、型・粒度・開示条件が変わればMappingから再検証する。SUIの高密度Canvasも、StudioのCodeMirrorも、この境界を保ったまま専用の描画技術を利用してよい。

### 18.2 同じ画面に存在する状態の寿命を分ける

UIで選択した対象は、まだHostで読んだ対象ではない。Hostが返したread-only Inspectionも、現在のSource原文、受理済みMeaning、現在のDisclosureや実行Grantの代わりではない。少なくとも **選択Intent → Catalog上の一意性 → Host Inspection → 表示用Projection → 利用者の明示操作** の順に扱う。画面側のWorkspace epoch、要求lane、Focus／Pinは応答の宛先を確かめるためのもので、権限を表さない。

未保存Buffer、Caret、IME、UndoはEditorが保持する。Pin Aを再確認する間にFocus Bが編集されても、Aの返信はBのEditor・Graph・Noticeを書き換えない。A→B→Aで表示名が同じに戻っても旧返信は再採用しない。実Sourceを保存した後も、Meaning受理・Build入力・Provider資格を自動更新しない。

SSR／SSGで初期HTMLを作る場合、初期Snapshotとその後のClient Readを区別する。公開時点のDisclosure、キャッシュのAudience／Scope、古いHTML・埋込JSON・Source Mapの情報漏れ、Hydration時の入力消失をTarget別に検査する。PWA／オフライン書込みは、競合とEffect unknownの扱いを確立するまで採用済みとはしない。

## 19. 現行Inspectorを使ったWeb開発方式の最小実装パケット（2026-11-16、P1の実装引渡し）

### 19.1 実コードで確認できる境界

現行の `reference/tei-studio-workbench` には、公開選択Event `inspector-selection.ts`、Catalog／Inspectionの純粋なReader `presentation.ts`、非同期結果のlaneを分ける `responses.ts`、Source位置の限定的な投影 `source-locations.ts`、それらを結ぶ `main.ts` がある。UI Frameworkの比較や新HTTP Serverを始める前に、ここで実際に表示できることを確かめる。

現在の経路は `emitInspectorSelection` → `inspectorSelectionFromEvent` → `inspectEligibility` → `responses.run("inspection")` → `client.inspectSubject` → `readInspection` → Editor／Source status／利用制限の表示である。Catalog上でSubjectが不在・曖昧ならHost Inspectへ進まず、応答のWorld／Source Index fingerprintがCatalogと合わなければ拒否する。Eventに混入したEffect／Grantは採用しない。**これらはNode試験と現行コードで確認できる読み取り専用の経路であり、公開Web／Stage2の完成ではない。**

`source-locations.ts` は複数の参照位置、Role、byte offset、Artifact fingerprintを最大20件まで表示用に検証し、観測範囲が全件であるとは主張しない。ただし現在のWorkbench表示からは未接続である。Source位置を得たことと、正しい原文を編集できることを結び付けず、S-LOCの表示とS-OPENの実Source再確認・編集を別に検収する。

### 19.2 WEB-P1を二つの小変更で閉じる

| 作業 | 差分の範囲 | 独立Expected／終了条件 |
| --- | --- | --- |
| **WEB-P1a：表示入力の分離** | `readInspection` が検証した結果から、Editor文書・版・Source status・利用制限・Noticeに必要なread-only ViewModelを純粋に導出する。最初は既存TypeScript・現在の `main.ts` とテストを小さく変更し、Host呼出し、Catalog資格、Response laneは動かさない | 同じSubject／Catalog／応答の表示値一致。異Kind、stale fingerprint、不正なAuthority limits、Host errorを正常な空表示へ落とさない。Presenterの型からGrant／Effectを生成できない |
| **WEB-P1b：表示先の非干渉** | 同じPresenter結果を一覧選択とGraph選択で使い、FocusとPinの既存laneに対して後着拒否を照合。S-LOCのDisplay接続は必要なHost範囲が確認できた場合の別差分とする | A→B→A、Pin Aの裏再確認中にBを編集、Source statusがpartial、旧Catalog、Reader error、狭幅／IMEでCaret・Editor・Graph・ARIAを保護。Node／Browser／実Windowsを別資格で報告 |

最初のPRは `presentation.ts` に既にあるReaderを再実装しない。**検証済みResultから画面に渡す値を整える純粋なPresenter** と、その境界の回帰試験だけを追加する。部分取得は「確認できた範囲」として表示し、「該当なし」「全件」「編集可能」を推定しない。Browserへの展開は同じPresenterの独立Consumerを使って確かめ、Wails Bridgeを共通Componentの必須依存にしない。

### 19.2.1 WEB-P1aの純粋Presenterと配布・生成Closureの受入を分ける（2026-11-16）

`reference/tei-studio-workbench/inspector-presentation.ts` に、既存 `readInspection` が検証した結果をEditor文書、Source状況、利用制限、Noticeへ整える純粋な `presentInspection` を追加する。対象IDの形式は既存 `subjectKey` に従い、Source状況が `partial` でもそのまま表示へ渡す。表示制限一覧はコピーする。Grant／Effect／Meaning受理や取得範囲の完全性を推定しない。`presentation.test.mjs` は既存Readerで検証した同一Subject／版の表示値、部分取得、制限の独立、旧Source Index指紋・非read-only応答・制限欠落の拒否を検査する。

**今回の差分は独立したPresenterファイルと局所試験であり、既存 `presentation.ts` と `main.ts` は変更しない。** 正規Workbenchの実表示はまだ旧処理のままで、新Presenterは実画面へ接続されていない。これは、[生成Workbenchの固定Closure](../../tools/reference-studio-static-composition/connect-workbench.mjs)に `presentation.ts` の過去のGit blobが明記されているためである。実体を改版したにもかかわらず過去の受領資格を維持するよう指紋を書き換えたり、検査を無効化したりしてはならない。今回の実Windows CIで現れた `workbench_closure_revision_mismatch` は、この境界を実際に検出した反例として扱う。

次のWEB-P1a-2では、`main.ts` から新Presenterを呼ぶ変更を**新しいSource／Closureとして資格化**する。Node24の `npm test`／`typecheck`／`build`、`reference/tei-studio-workbench/dist/main.js` とWails埋込 `cmd/tei-studio/assets/workbench/main.js` の実byte一致、HTML／CSS／第三者通知、既存Builderでの生成Workbench、最後に各Consumerの受領を順に確認する。歴史的に成功したPinを上書きせず、Source改版→生成→実Bundle→Consumerの新しい証拠を作る。手書きで埋込JSを似せただけでは合格としない。

WEB-P1a-1の局所試験、P1a-2の正規Bundle／生成Closure受領、WEB-P1bのFocus／Pin後着非干渉、S-LOCの実画面接続、実Windows／Stage2は別の資格である。既存#1180／#2514のExpectedと#1824／#997／#2831のOwnerに引き渡し、二本のP0への新しいHard Gateにはしない。

### 19.3 WEB-P2とWEB-P3へ進む条件

**WEB-P2（API公開）**では、既存 `operations.InspectSubject` を共通の意味基線とし、CLI direct、Wails非依存Studio Host、HTTP Adapterの三入口で対象、要求版、Failure分類、開示結果を照合する。HTTPのルート登録や同名のJSONを返すMockだけでは実Provider到達と数えない。HTTP側のActor／Audience／Disclosure、Current Grant、Rate／Size制限、Origin／CSRF／CORSを入口ごとに検査する。API Pluginの実ProviderはWailsに依存せず、Pluginの選択・Target収録・Host公開・実Call・Consumer受領を別々に観測する。

**WEB-P3（公開Route）**では、B2Cの公開可能な説明Facet一件に限り、Source→表示／Head Slot→選択済みBuilder入力→Asset Member→外部Consumer受領を追う。静的SEOと時点依存の価格・在庫、Experiment／計測同意を同じ再生成サイクルにまとめない。Slotが存在しないTargetは未接続とし、新SEO専用Engineを作ったことにしない。

WEB-P1／P2／P3はいずれも**全体P0の追加前提ではない**。P0は引き続き、非Plugin Studio BuildIntent→Stage1自身によるStage2構築→実Windows WebView2受領（#2831／#4583）と、通常製品CLI→実Provider Read→独立Consumer（#2586／#4495）。UIの実装方式は、同一Fixtureでの表示契約、コード生成が実際に担う構造、手書きの残差、差分失効、Build／実Consumerの根拠を比較してから採否を決める。

反証は既存#1180／#2514、UI実装は#997／#1824、共通Operationと公開は#1158／#951／#1251、SourceとMappingは#2497／#1124、B2Cは#3509のOwnerへ戻す。新しいUI専用Parser、HTTP Runner、永続State Store、独立Expected正本は設けない。本節は実装引渡しであり、新しいGo／Node／Browser／Windows／HTTP E2Eの合格を意味しない。

## 20. WEB-P1a-2のSource閉包と表示資格を、実Consumerから逆に確認する（2026-11-17、実装・検収引渡し）

### 20.1 共通Presenterは公開APIの開示判断ではない

WEB-P1a-1でmainへ入った `inspector-presentation.ts` は、`readInspection` が検証した局所read-onlyの表示結果からEditor文書・参照状況・利用制限・Noticeを純粋に導出する。ただし `readInspection` が認めるのは当該Workspace／Catalogと一致するHostの**表示用応答**であり、別Audienceへの開示、Source原文のOpen、Meaning受理、Effectの資格ではない。HTTPや公開HeadへこのViewModelの本文をそのまま送ってよいという契約は存在しない。WEB-P2ではServer側OperationとDisclosureの選択を確認した後、公開してよいFieldだけを別途投影する。

同名の `id` を持つ `semantic-data` と `semantic-rule` は異なるSubjectである。さらに同じqualified Subjectでも、Workspace AとBは別の表示先である。Editor文書IDは既存 `subjectKey` とWorkspaceを組み合わせ、表示版が同じでも対象の同一性を主張しない。Readerが返した `partial`、`unfetched`、明示NULLをPresenterで勝手に `complete`、不存在、ゼロへ縮約しない。選択の後着失効とPinの非干渉は、main統合済みの[PR #5168](https://github.com/hat47x/tei-platform/pull/5168)が扱う独立Expectedを再利用し、単なるPresenterの決定性から推定しない。

### 20.2 新しいWorkbenchを「別の版として」受け入れる最小手順

過去にWindowsで受領した `tools/reference-studio-static-composition/connect-workbench.mjs` の `WORKBENCH_NATIVE_CLOSURE` は**歴史的Snapshotの正確なGit blob一覧**である。新Presenterを組み込むために古いhashを更新すると、過去の検証事実が別のソースを指すことになる。WEB-P1a-2では旧Pinを保存したまま、今回のSourceと依存の組合せを**新たな資格**として検収する。既存の固定Snapshotに対する拒否試験もそのまま維持する。

| 段階 | 実際に確認するもの | 中止する条件 |
| --- | --- | --- |
| 1. Sourceと選択 | `main.ts` が `readInspection` 成功後に `presentInspection` を呼び、Workspace／qualified Subject／要求laneを変えずにEditorへ渡す。既存の `inspectEligibility` を維持 | 未資格Subjectで実Hostを呼ぶ、A→B→A旧返信を再表示する、Pin応答でFocusを上書きする |
| 2. 新しい依存Closure | `main.ts`、`presentation.ts`、`inspector-presentation.ts`、`responses.ts`、Editor／Graph／Command等の実import、lockfileの対象byteを一つのSource版で固定 | 古い `WORKBENCH_NATIVE_CLOSURE` を上書きする、対象外ファイルや版違いを黙って採用する |
| 3. 正規生成 | Node24・`npm ci`・`npm test`・`typecheck`・`build.mjs` を実行し、生成 `dist/main.js` とWailsの埋込 `main.js`、HTML／CSS／第三者通知をbyte比較 | TypeScriptだけ成功し生成物を未更新、手作業の差替え、生成器／lockfileの差異を無視する |
| 4. 配布先の実受領 | 通常Studioと既存生成Workbenchを**それぞれ**起動し、同一Subjectの本文・Source状況・制限・失敗表示、必要なら実Windows WebView2を検査 | Serverのhealthや静的HTMLだけでClient受領を代替、過去のWindows成功を新HEADへ流用 |
| 5. 独立Expected | #1180／#2514に、同名異Kind・Workspace分離、旧fingerprint、`partial`／明示NULL／未取得、後着、制限欠落の反例を返す | 現行の実装から期待値を自動生成、Source位置から書込み資格を作る |

この手順は新しいBuilder、生成専用Runner、永続Trace Storeの追加を求めない。現行の `build.mjs`、Workbench埋込Bundle CI、既存Static Composition／Wails CIを正規入口として利用する。UI Frameworkの全面変更、SSR／PWA実装、独立API Hostの追加は受入条件にしない。生成JSが同じbyteでもSource／Toolchain／Pluginの資格は別に確認し、生成器だけ変えたA→B→Aを無条件に再利用しない。

### 20.3 実測結果と未検収を同じ欄へ混ぜない

[PR #5173](https://github.com/hat47x/tei-platform/pull/5173)のマージ後に、同PRのHEADに対するGitHub Actionsを再確認した。**埋込Bundle一致のJobはsuccess**、通常・生成DesktopのWindows WebView2 smokeもsuccess。一方、repository-health、`studiohost` を含む一部のGo／生成Host Jobはfailureだった。ログの失敗例は `OPERATION_FAILED: external Evidence is unavailable`、Workspace／Web API／ThreeElementの別テストであり、WEB-P1a-1のPresenterが失敗原因とは断定しない。逆に他領域の不具合だから無視して製品全体をPASSとすることもしない。

WEB-P1a-1の新Presenterは実画面にはまだ未接続である。新WEB-P1a-2のSource・Closure・Bundle・実Consumerを検査していない限り、**過去のBundle一致・Windows成功は新Presenterの画面受領を証明しない**。新しいテストでは、同名異Kind・別Workspaceの表示ID、明示NULL、`partial`／`unfetched`、制限文の保持を、既存Readerと新Presenterの組合せで直接確かめる。WEB-P1bは既存#5168のFocus／Pin回帰へ接続し、WEB-P2/P3の公開・Target受領は別のOwnerに残す。

この追補によってP0の順序は変更しない。最優先は非Plugin StudioのStage1→Stage2実Windows（#2831／#4583）と通常CLIの実Provider Read→独立Consumer（#2586／#4495）。WEB-P1a-2はその生成・Consumer検収の共通手法を具体化するP1であり、独立の新Hard Gateではない。

## 21. Inspectorの検証と表示先を一件の契約にまとめる（2026-11-17、WEB-P1aの補強）

### 21.1 検証後にSubjectを取り違える余地をなくす

WEB-P1a-1の`presentInspection(view, workspaceId, selected)`は、Hostで検証した結果を安全な表示値に変換する。しかし`view`は元のqualified Subjectを保持せず、引数の`selected`と照合できない。個別に`readInspection`した後で別のSubjectを渡す配線は、表示用IDだけを誤った相手に付け替えられる。Source／意味／表示のIdentityを分離するTEIでは、この境界も明示的に扱う。

そこで既存Readerを変更せず、`inspector-presentation.ts`に`readInspectorPresentation(raw, { workspaceId, selected, catalog })`を追加する。最初にCatalogでSubjectの**存在と一意性**を確認し、同じ`selected`で`readInspection`のread-only／Host Authority Limits／World・Source Index fingerprintを検証してから、既存`presentInspection`で表示値を作る。途中で別Subjectを挿入する引数は持たない。同じ表示IDでも異Kindを混同せず、Catalogにない対象や曖昧な対象はHost返信が正しそうでも拒否する。

`workspaceId`は**画面の表示先**であり、Hostが認可したWorkspaceの証拠ではない。現行Inspect応答自体にWorkspace IDがないため、この純粋関数だけでは認可・現在のAudience・Disclosure・Host呼出先が正しいか確かめられない。Workspaceの切替・A→B→A・Pin AとFocus Bの応答が交錯する場合は、既存`responses.run`と[PR #5168](https://github.com/hat47x/tei-platform/pull/5168)の応答lane契約で現在要求かどうかを**先に**判断する。Presenterへ渡したという事実を、認可、元Sourceの存在、実Provider、公開の許可へ昇格させない。

### 21.2 受入を三つの境界に分ける

| 境界 | 合格とする最小事実 | まだ合格ではないこと |
| --- | --- | --- |
| **WEB-P1a-1b：型付き表示投影** | 同名異Kind・別WorkspaceのSubjectを区別し、Catalog不在／曖昧・異Subject応答・旧Source Index・非read-only・Authority Limits欠落を拒否する。明示NULL、`partial`、`unfetched`は表示上も保持する | 実ブラウザへの反映、元Sourceの全件性、Source Open、編集・業務意味受理 |
| **WEB-P1a-2：画面と配布** | 既存Readerと新Presenterを**現在の選択lane**に接続し、旧Pin／旧固定Closureを改版せずに新Source→正規生成→Wails埋込byte→通常／生成Consumerまで再資格化する | Source／Toolchainが変わった旧Evidenceの流用、正式Stage1→Stage2セルフビルドの代替 |
| **WEB-P2／SL-2：公開・実原文** | CLI direct／Studio Host／HTTPの共通Operationと、Audience別Disclosure、Source OwnerによるRoot・byte・Occurrence再照合をそれぞれ独立に確認する | Studioローカルの検証済みJSONを公開できるという推論 |

他チャットでmainへ統合された[Source位置の読み取り専用表示 PR #5212](https://github.com/hat47x/tei-platform/pull/5212)も同じ境界に属する。位置・Role・byte offsetを表示できることと、原文を開けることは別である。NodeのPort配線、Wails、SUI Canvas、B2C公開Headを横断しても、この境界を緩めない。

今回の新しいコードと回帰は`inspector-presentation.ts`と既存`presentation.test.mjs`のみ。歴史的`main.ts`、`presentation.ts`、固定Closure、Wails埋込Bundleを変更しない。次の画面接続では現行`main.ts`の成功時コールバックを新しい入口に置換してから、Node24／正規生成・埋込Bundle／実Consumerを**同一HEAD**で検収する。全体P0は非Plugin StudioのStage1→Stage2実Windowsと、通常CLI実Provider Readを引き続き優先する。

### 21.3 生成Inspector Readerの埋込不一致は別の実装不合格として閉じる

PR #5303の対象HEADでは、Node24の`npm test`／`typecheck`／`build`と埋込Bundle四ファイルのbyte一致を行う`build-and-match-embedded-bundle`が**success**だった。一方、`cmd/tei-studio`の`TestEmbeddedWorkbenchCarriesTheCurrentGeneratedInspectionReader`が複数のGo Jobで**failure**となった。これは「配布JSのbyteが現在のWorkbench buildと一致する」ことと、「必要な生成部品がそのJSに実際に組み込まれている」ことが異なるためである。失敗の直接原因は、埋込`main.js`に契約Digest付きの生成Readerが見つからないこと（`embedded Workbench bundle does not contain the generated Inspector reader`）。

今回のPRで変更していない`cmd/tei-studio/embedded_generated_part_test.go`、`assets/workbench/main.js`、`inspection-reader.generated.ts`、`main.ts`、`build.mjs`は、分岐元mainとGit blobがすべて一致している。したがってこの不合格を今回の純粋Presenterの変更に帰属させない。ただし、独立した製品障害として残し、**このチェックが失敗している限り生成Readerの埋込／実Consumerの資格は未受理**とする。

WEB-P1a-2では、契約Source→既存生成器→`inspection-reader.generated.ts`→実Workbench import／呼出し→正規esbuild→埋込JSを同一HEADで追う。生成Readerが**一つだけ**含まれ、契約Digestが現在のSourceと一致し、手書き`readInspection`との二重定義がなく、実利用記録を保持することを既存Goテストで確認する。生成物へ固定文字列だけを追記したり、旧Pinの指紋を置換したり、Goテストを緩めたりして通してはならない。次に通常・生成Workbenchと実WindowsのConsumerへ引き渡す。これは生成部品の組込み資格であり、Providerの実CallやStage1→Stage2セルフビルドの成功とは別である。
