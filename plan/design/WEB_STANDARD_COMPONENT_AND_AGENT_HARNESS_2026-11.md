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
