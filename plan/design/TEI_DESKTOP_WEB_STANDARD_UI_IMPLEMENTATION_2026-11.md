# TEI Desktop Web標準UI開発・検収指針（2026-11-15）

> **位置付け**：Desktop製品契約 [§0.13.5](TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md) と [§0.18.8](TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md)、[Desktop Host設計](DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md)を実装へつなぐ指針。方式の比較・採用条件を整理したものであり、Litの正式採用や製品E2E合格を意味しない。技術選定によってStudioセルフビルドのP0を増やさない。

## 1. 採用する原則

TEI Studio DesktopのUIは、HTML、DOM、CSS、Web Componentsを軸に構成する。**現行plain TypeScript＋esbuild＋CodeMirrorを比較の基線**とし、独立した部品に限ってWeb Componentsを試す。状態・再描画・テンプレートの複雑さを実測で削減できる場合、**Lit＋TypeScriptを第一候補**とする。Litそのものは業務意味の正本でも、Application Operationの入口でもない。

比較にはplain TypeScript、Lit＋TypeScript、Lit＋JavaScript/JSDocを含める。Reactは既存エコシステムや状態制御に明確な優位が認められる場合だけ対照に加える。設計上の「標準寄り」は、フレームワーク不使用という意味ではなく、**部品の公開契約を標準DOMで利用でき、技術の追加が正当化できること**を指す。方式の確定前にWorkbench全体を移植しない。

Goの共通Application OperationとSource/Meaning/Grantの所有権は従来どおりHost側にある。Wails Bindingは輸送路、UI Eventは利用者の操作意図であり、DOMの属性・選択・Custom Eventが操作許可や正式Meaningを発行することはない。生成Desktop TargetとStudioのFrontendも、意味正本は共有するが配布物・Capability/Authority・実Consumerの検収は別である。

## 2. 実装の出発点と現行の断点

2026-11-15時点のmainを照合した現行の実装入口は次のとおり。

| 確認箇所 | 実装上の事実 | 今回の扱い |
| --- | --- | --- |
| `reference/tei-studio-workbench/package.json` | Node 24以上、TypeScript 5.8.3、esbuild 0.25.12、CodeMirror。正式なLit/React依存はない | 既存の`npm test`、`npm run typecheck`、`npm run build`を壊さず比較する |
| `reference/tei-studio-workbench/tsconfig.json` | `strict: true`、`noEmit: true`、DOM型を有効化 | 新UI部品も同等以上の静的検査を維持。JavaScriptを試すときはJSDocと型検査の実効範囲を比較 |
| `reference/tei-studio-workbench/main.ts` | Host返信のPresenter、Focus/Pin、応答世代を管理 | 新コンポーネントが独自にHost/Grant/Sourceを解決しない |
| `reference/tei-studio-workbench/source-locations.ts` | S-LOCのread-only位置情報を純粋投影する | 第一候補の小部品に安全な表示モデルとして渡す |
| `cmd/tei-studio/assets/workbench/index.html` | 参照WorkbenchにあるPin操作用DOMが、配布用HTMLにはまだない | 参照画面の成功と配布Desktopの成功を分離。古いAssetコピーは使わない |
| `cmd/tei-studio/go.mod` | Go 1.26.11／Wails v3.0.0-beta.29の既存再現基線 | Wails v3.1.0への移行・正式配布資格を別の実OS比較とする |

この表はSourceの静的確認であり、すべての記載が当該mainの全CI成功・実Wails画面合格であるという意味ではない。実装担当はPRの**exact HEAD**で状況を更新する。

## 3. コンポーネントの分担と入出力

最初の実験対象は、**InspectorにSource参照位置を表示するread-only部品**とする。S-LOCから受け取った許可済みの表示モデルを描画するだけで、元SourceのOpen、Pinの更新、AI送出、Source Write、Provider実行には手を出さない。仮称`tei-source-locations`は製品APIとして確定しない。

| 層 | 役割 | 禁止する近道 |
| --- | --- | --- |
| Application Operation / Go Host | Root・Scope・Subject・現在資格を解決。実Sourceの読取、操作、Disclosure判定は既存Ownerへ委譲 | Web Componentへ許可済みパス、Secret、実行資格の発行を委任しない |
| Transport Adapter | Wails/Server等の結果と失敗分類を受け取る | Wails専用の第二Meaning/Grantを組み立てない |
| Presenter/State（plain TS） | `readInspection`／`source-locations`の検証済み出力、PinとFocusの別世代、unknown/partialを管理 | DOMをStore、CSSクラスを正本、旧Replyを現在のModel Revisionへ昇格させない |
| Web Component | 型付き・read-onlyの表示モデルを受け取り、文字列を安全に描画。操作が必要ならIntentのみ通知 | HTML文字列の無検証挿入、file URL生成、Wails API直接呼出し、Pin更新を行わない |
| CSS / Layout | Theme Token・表示密度・画面サイズ・文字拡大・焦点表示を制御 | CSSの状態表示で受理・実行の真偽を決めない |

公開プロパティは細粒度の独自状態を並べず、**Presenterが確定した表示用View Model一つ**を基本とする。View Modelは元byte offset、Role、参照件数とCoverage、表示件数の上限、未確認状態を欠落させない。表示の20件制限は全件完備を示さない。UIイベントが必要になっても、`CustomEvent`は`kind/id`と要求世代等の操作意図を伝えるだけとし、Hostで再解決する。発火だけでSourceを開かない。

**Shadow DOMは部品ごとに選択**する。デザイン部品など閉じた表示には有効だが、CodeMirror、IME、選択範囲、支援技術、外部Overlayと連携する部分では、Light DOMとの対照試験が先である。Shadow DOMを採る場合、外部からのスタイル変更はCSS Custom Properties、必要な表示部位だけ`::part`で契約化する。Closed Shadow Rootは初期既定にしない。

## 4. CSSの標準規則

1. **Design Tokensを公開契約の中心に置く**。色、余白、文字、線、角丸、可動時間、表示密度はセマンティックなCSS Custom Propertiesで表し、コンポーネントの局所ルールに業務状態の色・意味を埋め込まない。OSのライト/ダークと強制配色、文字拡大への対応はToken側で解決する。
2. **責務別のCascade Layer**（例：`@layer reset, tokens, base, layout, components, utilities`）を使い、特異性競争と`!important`の増殖を防ぐ。Layerの順序は一本に定める。Shadow DOMの内部Styleと外部Layerは同じCascadeではないため、Tokenを通じて受け渡す。
3. **全体配置はCSS Grid、局所の横並びはFlex**を基本とする。コンテナ幅に応じたPaneの変形はContainer Queriesを候補とし、Subgridは対応実WebViewの確認後に採否を決める。未対応環境で操作対象が消える構成にはしない。
4. **可読性とアクセシビリティ**：日本語と英数字の混在、長いQualified ID、狭いPane、125〜200%の文字拡大、キーボードFocus、スクリーンリーダー、`prefers-reduced-motion`、`forced-colors`を確認。判定は色だけに依存させず、`unknown`/stale/partialを文字と意味的な状態で伝える。
5. **DOM上の意味を優先**：既存の`label`、`button`、`details`、`role=status`を活かし、表示のための非対話要素へ安易にクリック処理を付けない。IME変換確定前のEnterやEscape、Focus移動を業務操作と取り違えない。

これはUIの責務と品質基準であり、CSS差分が業務意味に無関係だと**ファイル種別だけで**断定する規則ではない。Actionの可視/不可視、Consent、価格表示、開示の変更を伴えば既存のEffect/Disclosure/Owner確認に戻す。

## 5. AIエージェントが実装できる静的検査・作業手順

**現時点で実行できる基線**はNode 24、`npm ci`、`npm test`、`npm run typecheck`、`npm run build`。既存テストを置換・黙ってSkipしない。追加ツールは本指針だけでインストール済み・CI必須と扱わず、代表部品の実装PRで比較して採用する。

| 検査 | 導入候補と狙い | 合格証拠 |
| --- | --- | --- |
| TS/JSDocの型 | `tsc --strict`相当。JavaScript候補は`checkJs`・JSDocの診断範囲を別測定 | 不正View Modelと公開API破壊をcompile時に拒否 |
| コード／CSS | ESLint、Stylelint。明示Scopeで定義したDOM安全性、Layer/Token、unused rulesを検査 | 同じ違反例の検出と誤検出率。設定だけの導入はPASSにしない |
| Web Component公開面 | Custom Elements Manifest（CEM）とLit採用時のlit-analyzer等 | 実装のproperty/event/partと公開説明の一致、不要なHost API露出0 |
| DOM・アクセシビリティ | Playwright＋axe-core、キーボード/IME、Focus/ARIAの実操作 | 実RendererでAをPin中にBを編集/閲覧し、Bの状態非干渉と主要アクセシビリティ退行0 |
| 視覚回帰 | 同一Source/Viewport/フォント/OS条件でのスクリーンショット差分 | デザイン差分をレビューでき、意味状態や実DOM受領を画像一致だけで推論しない |
| 配布資産 | Node build成果→同期Asset→Go embed→配信実Byte→Windows WebView2 | 同一HEAD/Toolchain/Consumerで現行JS・HTML・CSSのbyteを確認。単なる`dist`成功は不十分 |

AI実装の一変更は、**対象Component → UI Intent/表示モデル → 許可されたSource → 変更差分 → 静的検査 → DOM操作 → 正規Bundle → 実Consumer**の順に根拠を残す。AIが作った画面キャプチャや説明を、Hostが受け取ったReplyやAuthorityの証拠の代わりにしない。Google等のWeb開発ガイダンスはLint/Reviewの参照に利用できるが、外部Skillの記述そのものをTEIのMeaning/Policyに取り込まない。

## 6. 比較実験と受入の停止線

最初の一つのFixtureは、**A=`sales.order`を明示PinしたままBを閲覧し、AのInspect/S-LOC結果が遅れて届く**操作とする。正常・後着・Workspace切替・A→B→A、同名異ID、元Sourceのpartial、20件超、UTF-8/CRLFの位置、Authority Limit欠落を、同じExpectedで比較する。

| 段階 | 実験で行うこと | 現時点の状態・停止線 |
| --- | --- | --- |
| **U0 現行基線** | plain TSの現Component/Pinと配布Bundleの未同期を実検査。現在の`typecheck/test/build`・生成byteを記録 | 参照TSと埋込HTMLが違う。最新mainの実DOM・Windows PASSとはしない |
| **U1 小部品の比較** | 同じS-LOC View Modelでplain TS・Vanilla Custom Element・Lit/TS（必要ならLit/JSDoc）を実装。Host/Presenterは共有 | 規模、差分局所性、Type/Lint検出、メモリ、DOM/IME/ARIA、保守変更の再現性で採否を判断 |
| **U2 配布同一性** | Node24正規生成、収録HTML/JS/CSS、Go Embedと実配信byteを同一HEADで照合 | 旧bundle・参照画面だけ成功・Pin DOM欠落はFAIL。採用前でも既存P-04の回復を優先 |
| **U3 実Windows** | Wails実Window/IPCで正常・拒否・後着を実DOM受領。必要時WebKit系を追加比較 | WebView2成功とWebKitGTK/WKWebView対応を混同しない。正式版Wails移行は別のSecurity/版検収 |
| **U4 生成Desktop** | #2831/#4583のStage1→Stage2を同一Build Intent・Artifact/実Consumerで証明 | UI部品が同じ見た目で動いてもTEI自身が実再構築した証拠にはならない |

比較時は単純なコード行数だけで優劣を決めない。**新機能の変更箇所、AIが修正した後の回帰検出、入力遅延p95/p99、初期Bundle byte、cold/warm起動、メモリ、Render更新量**を同じ条件で採取する。数値SLOは実測前に固定しない。Lite/React等の採否に伴うFramework移行をStudio P0の必須工程に加えない。

## 7. Ownerと優先順位

- **#1824／#4515／#3788**：Workbench Component、Pin/Focus/Dirty Buffer、IME/後着拒否、Windows実DOMの受領。S-LOCは§0.18.7/0.18.8の既存計画で実装し、別のSource管理を作らない。
- **#3421／#4583／#2831**：Node・esbuild・正規収録・Wails Bridge・実Windows・非Plugin Stage1→Stage2。Wailsの現beta.29再現と安定版評価、Release/Updaterの受入は別。
- **#1180／#2514／#2516**：独立Expected、開示・安全な表示、実Source/ConsumerのEvidence。ARIA/DOMやCLI JSONにも非公開Metadataを流さない。
- **全体P0を維持**：①非Plugin StudioのStage1→Stage2実再構築とStage2自身のWindows WebView2受領、②通常製品`tei`から実Providerへ到達し独立Consumerで結果を検証。UI技術選定、Web/SaaS、子Backend、Wails Updater、Mobileは新Hard Gateにしない。

**最初に切る実装PR**は、S-LOCの表示モデルを受け取る一つのread-only UI部品（選定比較）と、現Workbenchの正規HTML/JS/CSS収録・現Pin UIの実Consumer回復を分離する。方式変更を見た目の改善や既存UIの単純再記述で終わらせず、静的検査・契約の局所化・AI修正の回帰検出という実益が出た範囲で採用する。

参考：[Wails v3のフロントエンド規約](https://v3.wails.io/ja/guides/dev/frontend-frameworks/)、[LitのWeb Components](https://lit.dev/docs/)、[LitのShadow DOM・Style](https://lit.dev/docs/components/shadow-dom/)。
