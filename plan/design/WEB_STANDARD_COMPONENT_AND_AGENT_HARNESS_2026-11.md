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
