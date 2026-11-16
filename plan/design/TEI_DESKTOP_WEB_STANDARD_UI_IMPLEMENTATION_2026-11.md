# TEI Desktop Web標準UI開発・検収指針（2026-11-15）

> **位置付け**：Desktop製品契約 [§0.13.5](TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md) と [§0.18.8](TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md)、[Desktop Host設計](DESKTOP_STUDIO_HOST_INTEGRATION_DESIGN.md)を実装へつなぐ指針。方式の比較・採用条件を整理したものであり、Litの正式採用や製品E2E合格を意味しない。技術選定によってStudioセルフビルドのP0を増やさない。

**共通規約とDesktop固有契約の分担（2026-11-15）**：実装エージェントの必読規約、ディレクトリの正本、実行する検査の選び方は[AGENTS.md](../../AGENTS.md)に従う。Web Components、CSS、Custom Elements Manifest、Lit比較、アクセシビリティ、AI開発ハーネスの分野横断設計は[Web標準UI・コンポーネント・AI開発ハーネス設計](WEB_STANDARD_COMPONENT_AND_AGENT_HARNESS_2026-11.md)を参照する。本書は**Studio Desktop固有のSource→配布Asset→Wails実Consumer、起動と再構築の停止線**を具体化する。共通規則を本書にも複製して独立更新しない。現在Workbenchの`package.json`には`lint`等のScriptがないため、採用候補の静的検査を実行済みの検査と混同しない。

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
| `cmd/tei-studio/assets/workbench/index.html` | 参照Workbenchと配布HTMLの両方にPin操作のDOM（pin-current／inspect-pinned／unpin-current）が存在し、確認したmainでは両HTMLのbyteが一致する | 現在の静的Source一致と、各変更HEADの正規Bundle・実WebView2受領・Stage2再入は独立に検収する |
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
| **U0 現行基線** | plain TSの現Component/Pinと配布Bundleの同期を実検査。現在の`typecheck/test/build`・生成byteを記録 | 現行確認版では参照HTMLと配布HTMLのPin操作が一致する。実JS・Go埋込・Windowsでの受領は同一HEADで別に確認する |
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

## 8. 開発用Sourceと実配布Consumerの同一性（2026-11-15追補）

他レーンのDesktop Bootstrap、非Plugin Self-build、Plugin公開文書、Pin／AI対象引渡しの議論を突き合わせると、UI方式の変更より前に**現在のSourceが実配布物に入っているか**を確定する必要がある。過去版では参照HTMLと埋込HTMLに差があった。現行確認版ではPin操作のDOMは一致しているが、S-LOCのSource Openと実編集・保存まで含む製品一巡は未検収である。これらは別の問題として扱う。

現在の[Workbench Embedded Bundle CI](../../.github/workflows/tei-studio-workbench-embed-ci.yml)は、Node **24.21.0**を使って`npm ci`、`npm test`、型検査、ビルドを実行したあと、`index.html`、`main.js`、`style.css`、`THIRD_PARTY_NOTICES.txt`の**四つのbyte一致**を調べる。ビルド時に生じる`dependencies.json`は監査用であり、埋込ディレクトリへ入れない条件も既にある。これは正しい出発点なので、もう一つのBundle整合検査や新たなManifest正本を作らない。

| 観測点 | 使用する既存Producer／Consumer | 停止線 |
| --- | --- | --- |
| **Source** | 現行`main.ts`・`index.html`・`style.css`、`package-lock.json`、部品の実Source | 前回生成した`dist`や旧Draft #4812のJS／HTMLをSourceの代用品にしない |
| **Generated Asset** | `build.mjs`が生成した`dist`の四ファイルと`dependencies.json` | 生成成功だけではWails収録済みとしない。通知・依存閉包の欠落を見逃さない |
| **Embedded Asset** | `cmd/tei-studio/assets/workbench/`の四ファイル、既存CIの`cmp` | 四つのうち一つでも不一致ならP-04 FAIL。生成した差分を理由なく手修正しない |
| **Actual Consumer** | `//go:embed assets`、通常／生成Desktopの実Binary、Windows WebView2、Wails Serviceの実応答 | byte一致、通常Desktop起動、生成Desktop起動、Inspector／PinのDOM実使用は**それぞれ別の観測** |
| **Recursive Builder** | #2831/#4583の既存BuildIntent／BuildExecution／ArtifactSetとStage1の実TEI Process | Stage0のスクリプトが二回Go Buildを呼んだだけではStage1→Stage2成立としない |

**W0の回復は旧版の課題であり、今後は同一性の継続検査を行う**。変更したHEADのSourceから正規ビルドし、Pin操作のDOMと`main.ts`のHandlerが一致する結果を収録する。参照UIに新Componentを追加する比較実験は、既存の正規Bundle検証を妨げないよう分離する。埋込四ファイルの差分に不審な箇所があれば、コピーして整えるのではなく、Source／Generator／Lockのどの違いによるものかを特定する。

W0の継続検証（対象PRの**同じHEAD**で実施）は以下を基線とする。現状は既存CIにこれらの工程が記述されているという意味であり、ここで実行済みと主張しない。

```sh
cd reference/tei-studio-workbench
npm ci --no-audit --no-fund
npm test
npm run typecheck
npm run build
cd ../..
for file in index.html main.js style.css THIRD_PARTY_NOTICES.txt; do
  cmp "reference/tei-studio-workbench/dist/$file" \
      "cmd/tei-studio/assets/workbench/$file"
done
test ! -e cmd/tei-studio/assets/workbench/dependencies.json
```

この後に、同じHEADの実Windows Wails Desktopを起動し、実Bridge経由のCatalog／InspectとPin表示をWebView2から確認して初めてP-05へ進む。**旧Sourceと現行main、Node専用の成功とWindowsの成功、通常DesktopとStage2 Desktopを加算しない。** 実Binary・配信JS・Workspace・Subject・Revision・試行IDを別々に記録する。成功させるためにPinや`authorityLimits`の検査を外さない。

## 9. 選択・固定・原文・Ruleを同じ画面で扱う際の境界

StudioはSource、意味モデル、Graph／Table、設計文書、生成Artifactを**一つの表示体系に投影**する。ただし、UIで見えること、Sourceを正確に参照できること、その意味を受理できること、変更できることは独立している。以下を共通Component契約とする。

| 表示対象・操作 | Presenterが保持・伝えるもの | UIが推測してはいけないこと |
| --- | --- | --- |
| Subject Focus／Pin | Workspace、qualified Kind/ID、対象世代、Catalog／Sourceの現在性、不足の理由 | Focus変更がPinを移動したこと、同名表示が同一Subjectであること |
| S-LOC／参照移動 | Artifact相対Path、Role、byte範囲・指紋、最大表示件数と観測件数、Coverage | pathが現在安全に開けること、表示0件が完全な参照0件であること |
| S-OPEN／Editor | Hostが現在Root・Scope・Revision・Raw byteを再確認した場合のみ原文を開く。未保存Bufferは別版 | DOM側のfile URL・位置変換・Editor表示がSource Write／意味受理を許可したこと |
| Graph／Table／三要素Modeler | Semantic Subject ID、Source occurrenceと表示用Node/Edge IDの対応状況・多重性・Loss | 同じラベル・端点や表示Edge IDが正式Relation identityであること |
| Rule・Function・Plugin | 正式に選択したReader／Contract、実取得された意味・Source anchor、Providerの使用状況と未確認範囲 | Readerの成功だけで生成Go・実Provider Call・Effect／Owner受理が成立したこと |
| AI Harnessへ渡す対象 | Pin Intentの再解決、Required Context、Disclosure、Sink、実Tool受領と元Taskの対応 | 見えている情報全部を渡せること、AI Proposalを自動Applyできること |

2026-11-15のmainには、**内蔵`tei.rule-authoring/v0` Reader**が`Workspace.Load`を通して選択され、定義領域内のRuleを`semantic.Validation`とSource anchorへ接続する限定実装がある（[Bootstrap設計§29.7](STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)）。前章までの「`*.tei.md`はすべて未Reader」という前提は更新する。一方、未選択Sourceの拒否、外部Language Plugin、Graph／Windowの一般Reader、生成Goと実Provider、通常CLIの実行資格、Owner受理は個別に確認する。StudioのUIはこの限定成立を表示できても、未検収部分を`complete`へ丸めない。

**Componentの外部EventはIntentのみ**とする。たとえばGraphで選択したIDやSourceの相対Pathは、現在のHostに再解決を依頼する入力であって、操作・送信の許可ではない。古いReplyの後着、Workspace A→B→A、未保存Bufferの改版、Reader未対応、権限による非公開のいずれも、Source表示・Pin Intent・AI文脈送出の現在性を自動復活させない。ARIA／DOM／Graph Tooltipに非公開のPath、ID、件数を埋め込まない。

## 10. Desktopの起動・配布・更新とUIの責務

[Bootstrap／更新設計](STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)と[Desktop製品契約](TEI_DESKTOP_PRODUCT_CONTRACT_2026-11.md)を、Frontend実装に対する依存順として読む。

- **D1：現行の比較基線**。Wails親Process内で既存Go Application OperationとFrontendを動かす。別Hostを設けない。現在のWails `v3.0.0-beta.29`は再現用固定版であり、正式配布の採用資格を意味しない。安定版への移行はService、Origin、IPC、外部Navigation、実OSを別PRで比較する。
- **D2：単一Imageを標準候補にしたRelease Closure**。WailsのUpdaterが扱う単位に沿い、StandaloneではFrontend・必要なGo Backend・選択済みPlugin／設定Schemaを**同じReleaseに束縛**する。標準Pluginは静的に収録可能なものを優先するが、利用者の設定や外部DB・NASまで原子的に更新できるとは主張しない。更新前の互換性・署名・旧版とのデータ移行・復旧経路を製品側で別検収する。
- **D3：子Backend／外部Processの必要性を実測で判断**。プロセス隔離、失敗回復、独立更新、資源制限など実際の理由がある場合だけ選択する。Child image、Protocol/Operation契約、IPC handshake、起動停止、更新整合性、部分的なEffect Unknownを独立検証する。単一ImageのUpdaterだけで複数Processの同時切替を保証しない。
- **Web／SaaS**は同じOperationを再利用する別の配備形態であり、DesktopのWindowに公開Hostの認証・セッション隔離を付け足すだけでは適格化できない。Wails Server Build・loopbackの成功をそのまま外部公開へ転用しない。
- **Android／iOSは実験レーン**。Wailsの対応状況、依存SDK、実機操作、配布・更新・権限境界の検証が揃うまで正式サポートに含めず、Desktop P0の依存にしない。

UIが保持する一時表示・Pin・Dirty Bufferと、Release後も必要なWorkspace／Draft／ユーザー設定／Secret handleの保存資格は別である。再起動・更新後に同じ名前のWorkspaceが見えても、以前のSource revision・Grant・未完了Effectがそのまま有効とはみなさない。実行結果不明時は既存Receipt／lookup-firstの回復契約に従い、FrontendのリトライでDB更新・Plugin実Effectを二重化しない。

## 11. 直近の差分を増やさない実装引渡し

| 順序 | 実装担当へ渡す変更 | 受け入れられる範囲／停止条件 |
| --- | --- | --- |
| **W0：配布資産を現在のSourceへ戻す** | #1824／#3421。既存Embed CIで四ファイルを照合し、実Pin DOM・Inspector制限をWindowsへ届ける | 既存Sourceの正規生成がPASSすること。見栄えだけ一致／古い生成JS／別HEADのPASSはFAIL |
| **W1：最初のWeb Component比較** | #1824／#4515。S-LOC表示だけを、同じPresenterと同じ独立Expectedでplain TS、Vanilla、Litの候補比較 | Host/Grant/Source処理を移植せず、IME/ARIA・静的検査・bundle差分・AI修正の再現性を比較。Lit全面採用は未判断 |
| **W2：実SourceとRuleを同じConsumerへ** | #1124／#2621／#3788。実`InspectSubject`のSource位置・内蔵Rule Readerの限定Resultを現在資格付きで表示 | S-OPEN、Source Write、Graph Reader、外部Providerを完了扱いしない |
| **W3：Pin→AI Proposalの受領** | #4515／#3262／#2516／TAI。Pinを現在Hostで再解決し、Recipient／Disclosure／Tool Reply完全性を確認してProposalへ戻す | Focus Bに旧AのProposalを自動Applyしない。実送出・Owner Reviewは別受入 |
| **主線：非Pluginセルフビルド** | #2831／#4583。既存BuildExecutionへ正式non-Plugin Subjectと実Builder・Stage1→Stage2のProcess/Artifactを接続する | W1〜W3の完成をHard Gateにしない。同じHEADのStage2自身のWebView2使用まで検証 |
| **並行：通常CLI実Provider** | #2586／#4495。内蔵Rule Reader・Plugin選択と実Provider Invocation／独立Consumerを区別して検収する | Studioの画面に出たことをProvider実行・業務Effectの証拠にしない |

すべて既存Issueと#1180／#2514の独立Expectedに引き渡す。新しいTest Runner、Source Store、Frontend Registry、万能Approval APIは追加しない。**次回はW0の実コードPRとそのexact HEADのEmbed CI／Windows CIを進捗指標にする。** この追補は実装順・責務・拒否条件を統合したものであり、ここで新たなNode／Windows／Updater／Mobile／Stage2試験を実行したわけではない。

## 12. W0の実装を小さく閉じる：現行Sourceと実Desktopの検収パケット（2026-11-15）

### 12.1 参照Pinと正規配布Bundleの実装到達点（2026-11-15現在）

同一性の検査対象は、[Workbenchの`main.ts`](../../reference/tei-studio-workbench/main.ts)、[参照`index.html`](../../reference/tei-studio-workbench/index.html)、[配布`index.html`](../../cmd/tei-studio/assets/workbench/index.html)、[Pin配線テスト](../../reference/tei-studio-workbench/pin-wiring.contract.test.mjs)と、[Embed CI](../../.github/workflows/tei-studio-workbench-embed-ci.yml)に固定する。以下は古い欠陥の再掲ではなく、**実際にmainへ入った修正と受入済みの範囲**である。

- **W0-A 参照Pin配線は修正済み**：[PR #4920](https://github.com/hat47x/tei-platform/pull/4920)（merge `dca4985f5b6e4d9e3fc08e9655b77a42eec79589`）で、二重に登録された`pin-current`、未定義の旧`pinReader`参照、旧`pin-check`／`pin-release`、欠落した`pin-version`の不一致を解消した。PR exact HEAD `159799bd82309666c2ad6b97da8b1017a8403219`に対する[Pin専用CI](https://github.com/hat47x/tei-platform/actions/runs/43688648884)は**completed/success**。SourceとそのNodeテストの成立までであり、実Windows Desktopの操作資格ではない。
- **W0-B 正規Bundleは配布Assetへ同期済み**：[PR #4945](https://github.com/hat47x/tei-platform/pull/4945)（merge `8961848bd0867cc69619e8d80a3e22f19a248065`）で、同一Sourceと依存Lockを用いてNode 24.21.0の`npm ci`→`npm test`→`npm run typecheck`→`npm run build`を実行した。HTML・JS・CSSの配布Blobを更新し、NOTICEは差分なし。exact HEAD `20350bd2423d220be88f961bec2e41039e93d59a`に対する[Embed CI](https://github.com/hat47x/tei-platform/actions/runs/43695051532)は**completed/success**（四ファイル`cmp`と`dependencies.json`非配布）。一時生成Workflowはmainへ持ち込まなかった。
- **W0-BはSourceの改版ごとに再検収する**：上記#4945の成功後、read-only Inspector選択イベントを公開する[PR #4949](https://github.com/hat47x/tei-platform/pull/4949)が`main.ts`を変更したため、埋込`main.js`の整合が一度失われた。そこで[PR #4983](https://github.com/hat47x/tei-platform/pull/4983)（merge `13289dcac0bd15947f2a9dd3e34881eea11eb036`）でNode 24.21.0による正規Bundle再生成を行い、`main.js`のみ再同期した。exact HEAD `2d21845fdc0bd57eef3bc7a2b136d183f2e118ca`の[Embed CI run 43701028956](https://github.com/hat47x/tei-platform/actions/runs/43701028956)は**completed/success**。四ファイルのbyte一致、`dependencies.json`非配布、Source／Lockの固定を再確認した。一時的な書込Workflowはmainに残さなかった。したがってW0-Bの合格は**当該Source/Toolchain HEADに対する資格**であり、今後のWorkbench変更時に自動継承しない。
- **W0-Cは通常Desktopの一部だけ実Windowsで検証済み**：[PR #4955](https://github.com/hat47x/tei-platform/pull/4955)（merge `054f3a4ac3cd1e442e68b906828e44e0b7b647a1`）は、通常Wailsに存在しない生成Inspectorマーカーの誤ったSmoke要求を除去した。exact HEAD `3238a8f1a9ddbe5f2f53e984a96e30c6cebe1dab`の[WebView2 run 43695617730](https://github.com/hat47x/tei-platform/actions/runs/43695617730)では、**通常Desktop jobはsuccess**。実WebView2でCatalog／Inspect／Graph選択、Validation、read-only Editorのキーボード操作による改変防止、XEAD観察・空集合・欠損Source拒否を通過し、Wails Processもexit 0だった。ただし**生成Desktop jobはfailure**：静的合成用`WORKBENCH_PR1_BLOB`と現行Workbenchの変更差分が`workbench_revision_mismatch`として正しく拒否された。固定Snapshotを自動的に現行Sourceへ読み替える、PIN回避オプションを設ける、同じSourceとして偽装する修正はしない。旧固定版の専用Snapshotで再現するか、新たな候補としてCommand／Responses／Graph依存も含めてexact pinを受理し直す。この後、[PR #4995](https://github.com/hat47x/tei-platform/pull/4995)（merge `35994c63cb96c30a233304dd1c61f5d7f073627f`）で**通常DesktopのPin実機操作が追加で成立**した。exact HEAD `2f7132503dc88229953c45db25b350a78802c5d2` の[Windows CI run 43703879023](https://github.com/hat47x/tei-platform/actions/runs/43703879023)では`desktop-webview2-smoke` jobが**success**で、ログに`native-webview2-pin-a-select-b-recheck-release-verified`、Wails Process exit 0を記録。実Hostから得たWorld/Source Index版とPin Aの版一致、Aの固定中にGraph選択でBをInspectしてもAのID・版／BのEditorが保たれること、明示再確認直後の旧版失効と再読取、解除後のBの表示維持を検査した。**生成Desktop jobは同runでもfailure**（固定Snapshotの版不一致）で、Stage2自己再構築、A→B→Aの別Workspace切替、後着旧応答の強制競合、BのDirty Buffer／Caret／Graph/ARIA全文不変、Source Open、AI開示・現在Grantは未検収。
埋込Assetのbyte一致、通常Wailsの実Host/Pin操作、生成PoC Bの固定版に対する適格性、Stage1→Stage2の実再構築、実Provider/Owner受理はそれぞれ独立した成果とする。後者の試験が未実施・不成立であっても、通常Windowsで観測した範囲を抹消せず、逆に通常Windowsの成功を生成Desktopの成功と合算しない。
- **W0-Cの非干渉と再読込時の失効も実機で検証済み**：[PR #5012](https://github.com/hat47x/tei-platform/pull/5012)（merge `99044c2e4e0930787a9d98681f489fb194858220`）の[exact HEAD Windows run 43706400375](https://github.com/hat47x/tei-platform/actions/runs/43706400375)では、Pin Aを裏で再確認し解除する前後で、閲覧中Bの**Editor本文・Caret/選択範囲・Focus・Graph SVG/ARIA・診断**が変化しないことを通常Desktop jobで確認した。続く[PR #5025](https://github.com/hat47x/tei-platform/pull/5025)（merge `a0761349df6896859051dcf01a3fb558f2616294`）の[exact HEAD Windows run 43706898173](https://github.com/hat47x/tei-platform/actions/runs/43706898173)では、**同じWorkspaceを再読込してもPin対象Aの意図だけが残り、前回の読取資格と版が失効し、明示的な再確認を経て初めて実Hostから再資格化**されることを確認した。どちらも通常Desktop jobはsuccess・Wails Process exit 0、生成Desktop jobは別途failure（既存固定Snapshot版不一致）。この成功を、意図的な旧Reply遅延、別Workspace間のA→B→A、編集可能Buffer/Undo/IME、現在の開示Grant、生成Desktop/Stage2の成立と混同しない。
- **W0-C生成PoC Bの同一HEAD実機受領も成立**：[PR #5042](https://github.com/hat47x/tei-platform/pull/5042)（merge `61dc653841f1c9830b2018fe74b4e86cb2870f35`）では、過去4つの固定Workbench版を保持しつつ、通常Windows WebView2で受領済みの現行`main.ts`を別候補として固定し、Command／Graph／ResponsesとEditor／Pin／Inspector Event等のソース9件、`package.json`／`package-lock.json`の**計11ファイル**をGit Blob指紋で検査した。混在改行を持つ旧`client.mjs`は原byteで照合し、欠落・余剰・各依存一因子変更を拒否するNode回帰を追加した。[exact HEAD `00b92d3b6685834cab3a90595e449553e80d0d15` のWindows run 43722212784](https://github.com/hat47x/tei-platform/actions/runs/43722212784)で**通常Desktop／生成Desktopの両jobがcompleted/success**。生成jobは、固定Source→静的UI/Go生成→Node/Go Build→`studio_generated` Windows Wails exe→実WebView2配信`/workbench/main.js`のSHA-256一致（`55371a6d0fe83af88c03ccfd58543d5d0cdefd111337e6a22063a587295a5767`）→実StudioBridge Catalog／Inspect・Pin A/B・同Workspace再読込・Keyboard／XEAD→正常終了まで同一試行で確認した。これは**PoC B静的合成の固定版に限る実Windows Consumer資格**であり、TEI Stage1が非Plugin Studio Build Subjectを解決・起動してStage2 Imageを生成した証拠ではない。過去runの`workbench_revision_mismatch`は当該旧HEADの履歴として保持する。
- **RuleはReaderより先へ一段進んだ**：main統合済み[製品Load #4890](https://github.com/hat47x/tei-platform/pull/4890)／[Inspect #4891](https://github.com/hat47x/tei-platform/pull/4891)／[型付き意味同値 #4900](https://github.com/hat47x/tei-platform/pull/4900)／[合成Read #4901](https://github.com/hat47x/tei-platform/pull/4901)に加え、[正値Ruleの条件付きGo生成 #4911](https://github.com/hat47x/tei-platform/pull/4911)がR1の局所実装としてmainに存在する。ただし参照Builder実消費、製品CLIの実Provider/Entry、Owner受理、Windows Stage2の成功へは昇格させない。

### 12.2 W0-A／B／C／Dの統合順

| 単位 | 実装範囲と既存の検査 | 合格とみなす境界 |
| --- | --- | --- |
| **W0-A 参照SourceのPin整合（#4920済）** | 旧Controllerを除去し、3操作・版表示・既存Pin Reader/Stateの契約を一致させた。`PIN-WIRING-03/08`とNode 24 Pin CIを同一HEADで実行 | **Pin CI success確認済み**。Source配線まで。Host権限や実Windowの受領は別資格 |
| **W0-B 正規Bundleの同期（#4945・#4983）** | 同一Node 24.21.0 BuildのHTML・JS・CSSを配布に反映（NOTICE差分なし）。[Embed CI](../../.github/workflows/tei-studio-workbench-embed-ci.yml)で4ファイルの`cmp`と`dependencies.json`非埋込を確認 | **Embed CI success確認済み**。Source→配布byteの一致まで。実Windows描画やStage2は未受理 |
| **W0-C 通常・生成Wailsの実操作（#4995・#5012・#5025・#5042）** | 通常DesktopでA固定／B閲覧中のEditor・Caret・Graph／ARIA非干渉とWorkspace再読込時のPin失効を検証。その後、固定依存11ファイルの現行WorkbenchをPoC Bへ静的合成し、生成JSを実WebView2でdigest照合、Host/Pin/XEADを実操作 | **#5042 exact HEADで両Windows job success**（run 43722212784）。固定PoC Bに限る。遅延旧Reply、異Workspace A→B→A、Dirty/Undo、非Plugin Stage1→Stage2は未検収 |
| **W0-D 実Stage2** | #2831/#4583が、選択済み非Plugin Build SubjectからStage1の実TEI Process→実Builder→Stage2 Imageへ接続し、Stage2自身からW0-Cの操作を再検証 | 既存BuildExecution／ArtifactSet／実起動主体／Go・Node・Wails入力とArtifactの一致。通常Desktopが成功しても自己再構築の合格へ転用しない |

**N2の実起動をTEIへ移した限定実証（2026-11-15）**：[PR #5059](https://github.com/hat47x/tei-platform/pull/5059)はmainへ統合済み（merge `5a6f3b887df5227133c39c8840f2f86723869c9a`）。そのexact HEAD `5fcd9e2e92afe8266cb7710fe56f54c27a54ca17`で[Windows WebView2 run 43724133016](https://github.com/hat47x/tei-platform/actions/runs/43724133016)の**通常Desktop・生成Desktop両jobがcompleted/success**。後者は、従来CIが直接呼び出していたN2 esbuildを、実`tei self-build` CLIからNode子Processとして起動する方式へ変更した。実Node実行物のSHA-256を起動前にPinし、TEI Invoker Image、Child Process・終了状態、子が報告した生成JS digestとCLIがDiskから再読したdigest、実Wails/WebView2で配信されたJS digestを段階別に照合した。不正引数・既存Artifact上書きは負例で拒否。CIの実行結果は合成された成果ではなく、同一HEADの実Windows Consumerまで接続されている。

**依然として未成立の辺**：この実行ではTEI実行物はCIがGoで構築したSeed側であり、`self-build`の子選択はNode実行物のCaller Pinであって、**正式非Plugin BuildIntentから解決した選択済みN2 Builder・Node/Module/Source Closureの資格ではない**。既存の11件のWorkbench Source Pin、Node実行物Pin、生成JS・実WindowsのReceiptが別々に成功しても、全工程を一つの製品BuildExecution／ArtifactSetへ束縛したことにはならない。次は#2831/#4583でN0生成入力とN2 Adapter＋esbuild・N3 Go/Wailsを正式Build Subjectの解決・実起動へ接続し、**生成したStage1 TEI自身がStage2 Desktopを構築し、そのStage2実Windowで受領したこと**を独立に検証する。CI上の別Processや旧Windowの利用をStage2へ昇格させない。

この区別は[#5051](https://github.com/hat47x/tei-platform/pull/5051)のRule生成Go／原Markdown byte同一コピー／実BuildPlan消費でも同様である。**選択元SourceがBuilderの実入力になった証拠**と**選択された実行者がその入力からArtifactを作った証拠**を別々に保持し、片方のdigest一致だけで古いBuilder選択や実行資格を復活させない。新しいBuilder／Registry／証拠正本を設けず、既存BuildExecutionと独立Expectedへ接続する。

**次の実装差分**：通常Desktopでのread-only Pin/Focus非干渉と同Workspace再読込時の失効は実機で成立した。残る#4515/#1824の負例は、**実Hostの旧Replyを意図的に遅延**させる競合、異なるWorkspaceのA→B→A、編集可能なDirty Buffer／Undo・IME、Source Open、現在Grantの更新である。生成PoC Bは現行Sourceの明示Pinと依存Closure付きで実Windowsまで検証したため、今後のSource／Toolchain更新時は資格を再取得する。並行するRule→生成Go→通常`tei self-build`・Binary実起動の限定成果[#5009](https://github.com/hat47x/tei-platform/pull/5009)は、Studioの非Plugin Stage1→Stage2や実Provider Entryを証明しない。正規Source→dist→Embedを維持し、#2831/#4583と#4495の製品P0へ接続する。

**実装の分割基準**：Aを単独PRにする場合、Bundle未同期に起因する既存CI失敗を消したことにはしない。BとCを一つのPRで検証する場合でも、各検査の結果を別に記録する。Bの正規生成は独立したJavaScript/CSS／Licenseファイル群の改版として扱い、手で整えた見かけの一致、pin検査無効化、Go Embedの期待値緩和でPASSを作らない。W0-Dは既存Self-build Ownerの作業に残し、Frontend担当へ新Builderを作らせない。

### 12.3 必要最小限の反証と報告

実装Ownerは#1180／#2514の独立Expectedに、現在のFixtureで確認できる負例を接続する。(1)参照HTMLのPin IDだけを旧値へ戻す、(2)配布HTMLのみ旧版へ戻す、(3)配布JSを旧版にする、(4)Pin Aの遅延ReplyをBのDOMへ反映する、(5)Source IndexのReferenceが0件のとき完全な不存在と表示する、(6)Stage2でStage0がBuilderを起動しただけなのにStage1による再構築と判定する、の**異なる失敗原因**を混同しない。入力Workspace・Subject ID、Root/Source版、Build Profile、UI Asset SHA、実Process/WindowとAttempt、テストのPASS/FAIL/SKIP/NOT RUNを必要な範囲で記録する。

特に**検査の非実行を合格にしない**。PR HEADのActionsがqueued/in_progressなら未判定のままとし、別HEADの緑色結果を流用しない。mainに存在するGo回帰やNodeのテスト記述は実行可能な資産であり、当該版の製品受入結果とは異なる。CI全体が別件で停止した場合は、到達した工程と止まった工程を区別して次の小PRへ引き渡す。

## 13. ノード／フロー編集をDesktopへ導入するときの共通モデル境界（P1の採否条件）

Node-RED型の視覚的プログラミング、業務プロセス、マッシュアップ／APIの接続フローをStudioに導入する場合も、**新しいノード専用の意味正本やDesktop固有のフロー実行機構は作らない**。標準データ構造・標準表現型（#2495/#2497/#4350）に、Node/Port/Edgeの一意Identity、型付き入出力、順序・多重性、Source occurrence、Rule、Function、Operation、Provider、Process Flowとの対応をそれぞれ表現し、Canvas／SVG／Tree／表はそのProjectionとする。

最初に検証するのは**同名の二Node、同じ端点を持つ異意味の二有向Edge、一つの型不適合Port**を含む固定Fixtureである。表示用Edge IDや座標を正式Relation IDに昇格させず、Edgeの向き・多重性・入出力型と接続先を表示から再確認できることを確かめる。Graphからプロセスフローへ変換する場合は、順序・条件・非同期・EffectのLossを明示し、表現できないものは`unsupported`とする。Runtime実行、外部API公開、Source逆編集の権限は、それぞれ既存Function/Entry・Provider/Host・Writer/Ownerの受入が必要である。

P1でのStudioの役割は、既存#1124の三要素Modelerと#4198のLifecycle Graphから**同じ標準要素を異なる図法で投影・操作できるか**を比較することとする。ノード単位の汎用部品はCanvas以外の表・API生成へも再利用できるように設計するが、独立した第二Rule Engine／フローRuntimeを前提にしない。最初の一Node/EdgeのSource→表示→選択→read-only Inspectまでを証明してから、ノード部品群・クラウドマッシュアップ・生成APIの用途を拡張する。

このP1は探索・設計の採否条件であり、W0-A〜Dや通常製品CLIの実ProviderというP0の新Hard Gateではない。実装・独立Expectedは既存#1124/#2497/#1180/#2514へ引き渡す。

## 14. Desktop Workbenchを共通の作業面として確定する（2026-11-16、設計・実装引渡し）

### 14.1 一つの対象を、複数の表現で扱う

Studioの中心は特定の図法やエディタではなく、**現在のSourceに裏付けられた対象を発見し、その構造を見て、変更の影響を確認できる作業面**とする。画面の構成は次の四領域を基準にし、表示・非表示や幅は利用者が調整できるようにする。

| 領域 | 最初に扱うもの | 守る境界 |
| --- | --- | --- |
| Project／対象一覧 | Workspace、Source（`*.tei.md`・通常Markdown・旧`*.tei`）、Data／Function／Process／Plugin等の分類、検索結果 | ファイル拡張子だけで意味読取済みとしない。WorkspaceとSourceの現在性が不明なら、その旨を示す |
| 中央の作業面 | Source Editor、表、Graph／Node、Process Flow、文書、生成Artifactのタブ。表示は同じ対象への別Projection | 図上の配置や一時的なNode IDを意味IDへ昇格しない。表現できない向き・多重辺・時点・NULL・順序を省略せず、Lossを示す |
| Inspector／固定対象 | Focus中の対象と、明示Pinした対象を別々に表示。Source位置、型、参照関係、診断、読取範囲を開く | Focus移動でPinを変えない。Pinを実行権限・AI送信許可・書込み資格として扱わない |
| 診断・実行結果 | 検証結果、差分Preview、テスト、Buildの選択と実消費、Artifact、実Consumerの観測 | `unknown`／`partial`／`unsupported`／`not run`を「問題なし」に変えない。CI上の生成成功とDesktopでの実使用を分ける |

タブやPaneは対象の種類ごとに別の意味管理機構を持たず、既存のCatalog／Inspect／Source Indexと標準データ構造を受け取る。Sourceに戻れない派生表示は、その理由と取得元を表示する。Source Editorの未保存Buffer、Diskに保存されたSource、受理済みMeaning、Build入力、生成Artifact、実行中のConsumerは**それぞれ別の版**として追跡する。切替や再読込で、未保存Bufferの内容を受理済みMeaningへ自動昇格させない。

### 14.2 共通の選択・移動・変更契約

画面間で共有するのは、既存Operationが解決できる**Workspace／qualified kind・ID／Source occurrenceへの参照意図**である。選択イベント自体にRead権限や編集資格を持たせず、必要な時点でHostが対象を再解決する。表示の更新は閲覧用のFocus lane、明示固定した対象の再読取はPin専用laneとする。片方の応答がもう片方のEditor・Caret・Graph・Noticeを変更してはならない。

原文・表・Graph・Node View・Process Viewの移動は、次の順序を共通にする。

`対象を選ぶ → Hostで現在のSubjectをInspect → Source occurrence／意味IDとの対応を表示 → 必要なら元Sourceを開く → 編集候補をPreview／Diffで比較 → 既存WriterとOwner受理へ明示的に渡す`

- 読取専用の選択と参照移動は早期実装できる。Sourceへ戻す位置はbyte offsetとSource Revisionを保ち、Editorの文字位置・IME・CRLFへの変換は別検査とする。
- 「変更可能」は表示Widgetの種類で決めない。Source変更、意味モデル変更、生成物の再Build、実Providerへの作用は、それぞれ既存のOwner・現在資格・条件付き更新経路が必要である。確定できないRelationやSource occurrenceはread-onlyにとどめる。
- Node配線は、基本型の個別Port同士、構造化型同士、型付きAdapterを介した接続を同じ標準型契約で表す。Portの向き・多重度・必須性・型変換時の情報損失は独立に検査する。線を描けることは、Operator選択・実行順・副作用許可が成立したことを意味しない。
- AI／CLI／SHOへの引渡しは同じ対象参照を出発点にできるが、必要Contextの完全性、Audience、Disclosure、実Tool Replyの完全受領、Owner Reviewは別々に確認する。AIを使わなくても閲覧・基本編集・検証を利用できるようにする。

### 14.3 入力遅延と非干渉を最初の品質指標にする

中央のEditorには、IME変換・選択範囲・Caret・未保存Buffer・Undo/Redoの即応性を優先して残す。長時間のIndex／Build／Plugin／外部Harnessからの後着結果は、操作開始時のWorkspace・Subject・Source／Buffer世代と現在表示を照合して採用する。背景のPin再確認でFocusを奪わず、特定Pluginが遅延・停止しても基本の編集と診断はできるだけ維持する。

性能は冷間起動、最初の操作可能時刻、入力遅延p95/p99、表示更新、Provider応答、Build待機、合計メモリを分けて測る。P0で架空の数値目標を固定せず、同じFixtureと実OS／WebViewで現行plain TypeScript基線と比較する。Web Components／Lit導入、子Backend分離、WASM化は、UIの応答・分離・保守に有意な利益が確認できた範囲で段階採用する。

## 15. 生成Desktopを「実際に使った一巡」で受け入れる（2026-11-16、P0の次段）

### 15.1 現在成立している辺と、次の断線

[main統合済み #5059](https://github.com/hat47x/tei-platform/pull/5059)のexact HEAD `5fcd9e2e92afe8266cb7710fe56f54c27a54ca17`、[Windows run 43724133016](https://github.com/hat47x/tei-platform/actions/runs/43724133016)では、**実TEI CLI→NodeのN2子Process→生成JS digest照合→通常／生成Wails WebView2**という限定した経路の両Job成功が確認されている。これはN2の呼出元と実配布Consumerを確かめた成果であり、正式な非Plugin BuildIntentの選択済みBuilder、入力閉包、生成Stage1自身がStage2を構築した資格を含まない。

次の実装は新しいBuild Engineを作るのではなく、既存`BuildIntent.Resolve`／Builder Catalog／BuildExecution／ArtifactSet／Subject Bindingと、今あるN0 Generator・N2 Adapter・N3 Go/Wails Buildを接続する。CI Workflowのコマンド列は**実行手順の観測元**にはなるが、そのまま製品の意味契約や正式Recipeとはしない。

| 段階 | 実装で結ぶ最小の辺 | その段階での停止線 |
| --- | --- | --- |
| **D1：選択** | 非Plugin Studio Build Subject→既存Resolverが返したBuilder／TargetとPlan Fingerprint | CLI引数の`-child`と起動前Digestだけでは「選択済みBuilder」としない。未表現の工程は`unsupported`を維持 |
| **D2：入力** | N0生成Source、N2のTS・Node／lock、N3のGo module・build tag・Wails資産を、実際に消費した集合へ対応付ける | Selected／Consumed／Observedを区別。外部module、OS／SDK、環境変数の欠落を全閉包PASSにしない |
| **D3：実行** | 選択Builderと実Invoker image／argv／Attempt、N0→N2→N3のArtifact依存を既存Executionへ束縛 | 宣言B／実行A、Plan外esbuild、旧JSの埋込み、上書き再Buildを拒否 |
| **D4：再入** | Windows上で実行した**生成Stage1 TEI自身**がStage2用Buildを選択・開始し、生成したStage2 Imageへ到達する | CI ShellやSeedの二度目のGo Build、別OSの局所再入はL3の代替にならない |
| **D5：実Consumer** | D4のStage2 Image／埋込JS digestに結び付けた実Windows WebView2からCatalog／Inspect／Pinを受領する | 先行PoC Bで起動したWindowや旧AttemptをStage2として数えない |

### 15.2 最初の実装・検収パケット

まず**D1→D3の一成果物だけ**を現在のBuild実行契約へ接続し、`Resolve`が選択したBuilder Bを実Invoker Aにすり替える負例、N2が報告したJSとN3に埋め込まれたJSが異なる負例、Build中のSource／lock改版、未知の依存を成功へ丸める負例を独立Expectedで固定する。実際に接続できない工程を一つの巨大な`child`内部へ隠して、見かけのBuildPlan完全性を作らない。

D1→D3が同じAttemptで成立した後、D4のStage1実ProcessとD5のStage2 Windowを接続する。受入票には、Sourceと実入力Byte、Model／World・Build／Artifact Revision、Builder選択、OS／Go／Node／Wails版、実Invoker／Child／Stage1／Stage2各Image digest、実WebViewが受け取ったJS、失敗・未実施項目を**独立列**で記録する。byte一致は意味受理ではなく、Stage2起動は外部Providerの実Effect受理ではない。

実装Ownerは[#2831](https://github.com/hat47x/tei-platform/issues/2831)（Build・再入）・[#4583](https://github.com/hat47x/tei-platform/issues/4583)（Studio一巡）・[#1824](https://github.com/hat47x/tei-platform/issues/1824)（Window／UI）、負例は[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)。§14のUI統合、実Provider Read、Node表示、AI協業、Wails子Backendの比較は並行可能だが、**D1〜D5と通常製品CLI→実Providerの二本のP0を新しいUI技術選定の完了待ちにしない**。

**今回の変更は仕様・受入条件の追補のみ。** 新Go／Node／Windows／Provider／Stage2の実行とOwnerの製品受理は行っていない。旧runの成功を本追補のHEADに移し替えない。

## 16. Desktopセッションの現在性と編集・保存の一巡（2026-11-16、設計・受入条件）

### 16.1 画面の状態と、業務上の状態を混同しない

Workbenchは一つのWorkspaceを大きな可変オブジェクトとして扱わない。UIの**閲覧位置**、編集中の**未保存Buffer**、Hostが読み直した**実Source**、Ownerが受理した**Meaning**、Build／Operationの**実Attempt**を分ける。一画面で複数の版を見られることと、それらが同じ版であることは別である。新しい中央State Storeや製品共通のStatus enumを設けず、既存のEditor・Presenter・Application Operation・Source／Build／Run Evidenceの所有範囲を保持する。

| 対象 | 所有と持続性 | 更新・失効の基準 |
| --- | --- | --- |
| Window／Tab／Pane／Focus | UIの局所表示状態。配置の復元はDesktop設定 | 閲覧先の切替だけではSourceやPinを更新しない |
| Pinと最後に確認したInspection | 明示的な閲覧意図とread-onlyのHost応答 | Root、Workspace、対象、Source／World、要求世代の不一致は現在の観測として採用しない |
| Editor Buffer／Caret／Undo | 各Editorが持つ未保存の作業状態 | Disk保存・Meaning受理・後着結果から独立。IME確定前の文字列と保存済みbyteを取り違えない |
| Disk上のSource／Source occurrence | Workspaceと既存Source Ownerが持つ原文 | 保存直前に実byte、Path／Root、対象Revisionと競合を再検査。Watcher通知は競合の証明ではない |
| 受理済みMeaning／Preview | 既存Reader・Acceptance Ownerと、根拠を持つ派生表示 | Buffer保存では自動受理しない。再読込・再検証後に投影 |
| Build／Operation／Provider結果 | 既存BuildExecution／Run／Receiptと実Consumer | UI再接続や完了通知を実行成功・再実行許可にしない |

**最小のEditor操作順**：Sourceを開く→Bufferで編集→差分を確認→既存Writerの保存入口で実Sourceのbyteと現在権限を照合→保存結果を再読込→既存Validation／ReviewでMeaning受理を別に判断→必要なら現在のBuild Intentから再構築する。保存と受理を同時に扱う既存Operationでも、結果・失敗は分けて表示する。保存失敗でBufferを捨てず、復元Draftは候補として扱い、自動受理・外部Effectの再送はしない。

### 16.2 非同期応答の宛先を、開始時と表示直前に照合する

非同期Readには既存のWorkspace／Root、qualified Subject、Source Revision、Focus／Pinのlane、操作世代を対応付ける。後着応答は**結果を受領した事実**として診断に残せても、表示先の現在性と一致しない限りEditor本文・Caret・Focus・Graph・現在の実行資格を書き換えない。A→B→Aで名前が戻っても、前のAの応答は自動復活しない。Pinの読取中にBを編集している場合、Pinの結果をBの未保存Buffer・Undo・IME・ARIA更新へ流さない。

異なるWindow・Worktree・Database Bindingも同様に、表示タブ名やDB aliasではなく、Hostが解決したRoot／Source版とEnvironment／Provider routeを操作ごとに照合する。Local Workspaceの認証不要という原則を守り、Connected／Hybridの期限切れGrantをローカル権限へ置き換えない。複数Windowの競合はUIタブの排他ではなくSource実byteと既存Acceptanceの条件付き更新で判断する。

### 16.3 既存の受入票へ渡す負例

既存#1824／#4515／#1529／#1124／#1180／#2514のFixtureを用い、①A→B→Aの後着Focus／Pin、②Pin A裏応答中のBのIME・Caret・Undo不変、③Buffer保存後も古いMeaningとBuild入力が自動更新されない、④外部編集／Symlink差替え／Source競合で保存拒否、⑤復元DraftからAccept／外部Effectが発火しない、⑥同名Subjectの2 Worktreeと同aliasの2 DB Bindingを取り違えない、を独立Expectedで確認する。現行Windows Pin閲覧の限定成功を、Dirty Buffer・複数Worktree・実Providerの合格に算入しない。

## 17. Wailsから独立した共通操作とPlugin接続（2026-11-16、設計・引渡し）

### 17.1 API PluginをDesktop Host内へ閉じ込めない

Studio Wails Bridgeは**共通Application OperationのDesktop輸送Adapter**とする。API向けPluginも同じOperation／Capability／標準型の契約を使い、**CLIから直接呼べる**ことを実装条件とする。Wails ServerはそのAdapterを利用できる実行形態の一つであり、API Pluginの意味正本、認可判断、Plugin Loader、業務状態の所有者ではない。DesktopだけのためにAPI実装がWails packageやWebView lifecycleをimportしない。

依存の向きは「UI／CLI／Web Adapter → 共通Operation／Capability → 選択Plugin／Provider → 実Target」。各入口は型付きInput・Result・Failure／Provenanceを共有するが、Caller、Scope、Resource Lease、Disclosure、現在Grantは入口と実呼出しごとに確認する。Wails Serverで登録された全公開Methodをブラウザー向けAPIへそのまま公開しない。Local loopbackの成功を外部公開・多利用者認可・配布の安全性に読み替えない。

### 17.2 Node ViewはPluginの配線を可視化する表現型

Pluginの入出力には標準型を用い、**構造全体の接続**、**Fieldごとの接続**、**明示Adapterによる変換**を選べるようにする。Node／Port／Edgeは一意Identity、向き、必須性、多重度、粒度、単位・精度、NULL／missing／unknown、秘密区分、順序・時間条件を必要な範囲で保持する。UIが隠したField、型変換で失う要素、未接続の必須PortはPreviewに表す。折り畳まれた線や単純な型一致だけで有効なMappingとしない。

Node Viewは標準データ構造、Process Flow、Operation／Plugin Bindingの**Projection**であり、線の描画・型一致・Preview合格は実Operator選択、実Provider実行、外部Effect、Source逆編集資格を与えない。新しいFlow Runtime／Node Registry／Desktop専用Plugin Managerを作らず、まず一つのSource-backed Node／EdgeをCLI InspectとStudioで同じ意味として読めることを確かめる。既存#2497／#951／#1223／#1180へ引き渡す。

### 17.3 実装の優先順と停止線

**P0は二本を維持する。** 第一は§15のD1→D3（非Plugin Build Subject／選択Builder／実Invoker／ArtifactSetの同一Attempt）→D4（生成Stage1自身がStage2を構築）→D5（そのStage2を実Windows WebView2が受領）。第二は受理済みpure Function→通常製品CLI→選択実ProviderのRead→独立Consumerである。いずれもUI Frameworkや子Backendの導入を先行条件にしない。

**並行するUI／APIの小PR**は、(a)既存Pin／Focusの非干渉、(b)一SourceのBuffer→保存時の実byte照合→再Inspect／Review、(c)CLIとWails Bridgeの同一read-only Operation／失敗分類、(d)一組の型付きPortとAdapterのread-only表示を対象とする。書込み・外部API公開・実Provider Effectは各Ownerの独立受入後に拡張する。

**性能と採否**：入力遅延p95／p99、初回操作可能時刻、メモリ、Host往復、Build待ちを現行TypeScriptと同一Fixtureで実測する。Web Components／Lit、子Backend、Go／WASMは改善を反証可能な場合に局所採用し、Go単一バイナリ配布と共通Operationの独立性を優先する。

**本§16〜17の状態**：設計と実装引渡しのみ。新Go／Node／Windows／実DB／実API／Stage2の検収、Owner意味受理、正式製品採用は行っていない。

## 18. Desktopで「意味の経路」を検査する（2026-11-16、既存テストへの実装引渡し）

### 18.1 見た目と実装名ではなく、要求された経路をたどる

StudioのEditor・表・Graph・Node・Inspectorは、同じ意味要素の異なる表現である。利用者が「この規則はどこで定義され、どのデータを読み、どの実装で評価され、何が確認できたか」を調べるとき、生成Goのメソッド名、UI DOM ID、描画座標、ノードの見かけのラベルを同一性の根拠にしない。

抽象経路は「対象Source／qualified kind・意味ID・Role → 型付き意味・実装義務 → 選択された実現方法／Operator・Plugin／Builder → 実際に呼ばれた入口と入力 → 結果／独立観測 → 現在のConsumer」の必要な区間を、**既存のSource Index・Operation・Build／Execution／Receipt・Consumer証拠から読み取る観点**とする。新しいTrace Engine、意味経路専用の永続Schema、第二Test DSLを設けない。画面がまだ実際に観測していない区間は、図で線をつないでも「未確認」とする。

| 確認する不変条件 | 変更してよいもの | 検査すべき違反 |
| --- | --- | --- |
| qualified Subject／Source occurrence／Roleと必要な参照の結び付き | 表示名・配置・タブ構成・生成メソッド名 | 同名別Subject、Relationの向き・Key・多重度・Source版の取り違え |
| 実装義務と型付きInput／Output／Read Footprint／Effectの対応 | Generate／Interpret／Delegateの実現方式 | 必須Field欠落、NULL・missing・unknownの縮約、別Operator／Provider選択 |
| 選択されたBuilder／Pluginと実Input・Artifact／Invokerの一致 | 実装先の言語、最適化、内部関数の分割 | 選択B・実A、旧Source／Toolchain／生成JS、未観測の依存を一致扱い |
| Hostで受理した現在のSource・Scope・GrantとConsumerへの開示 | CLI／Desktop／Webの表示方法 | A→B→A旧Reply復活、別Workspace／Worktreeへの漏洩、UI活性を実行資格へ昇格 |

**実現方式の比較**：Generateは選択した意味を実Sourceから読み、実Builderが生成物を消費したことまで確認する。Interpretは同じ意味から既存Evaluatorを実呼出しし、選択された読取属性・判断を照合する。Delegateは外部実装への委譲境界と実Providerを確かめる。三方式は同じ独立Expectedに照らして比較できるが、方式ごとの実使用・契約閉包は別に記録する。単に生成Goが起動したことや、Go関数が正しい値を返したことは、DSLがその判断を担った証拠ではない。

### 18.2 先にread-onlyの一経路を通す

最初の固定対象は、既存の選択済みRule Markdown Fixtureでよい。正規Workspace Load／Source Indexから読めたRuleを、通常製品CLIの実Process、共通operations.InspectSubject、Wails非依存studiohost.NewReadOnlyで比較し、次に既存Wails Bridgeから表示まで追う。現行mainの`cmd/tei/cli_inspect_rule_markdown_process_test.go`には、実CLI別Process・直接Operation・`studiohost.NewReadOnly`の三入口を比べるGo回帰が存在する。既存Fixtureを再利用し、実行したHEADの試験成績は別に確認する。CLIのInspectとHostの応答が同値でも、WebView／実Windows／Stage2を検査したことにはならない。

同じFixtureに「非TEI Fenceに書かれた偽Rule」「同名だが異Kind」「許可外Workspace」「実Sourceの一文字改版」「Ruleが読めても公開Function Entryにはならない」を置く。Source位置が一意でない場合はJump／Quick Fixを保留し、PreviewはSource byteを変更しない。実編集へ進む際は、§16のBuffer・Disk Source・Owner Acceptanceを区別し、保存時の実byte照合、再Inspect、診断への戻りを検証する。見かけのGraph選択やAI提案を自動修正・自動受理の根拠にしない。

### 18.3 受入結果を利用者に見せる粒度

ProblemsとInspectorは、少なくとも「Sourceを読んだ」「Meaningを解決した」「実装を選択した」「実Provider／Builderが呼ばれた」「結果をConsumerが受け取った」の**実際に裏付けのある段階だけ**を説明できるようにする。これらは既存証拠の表示観点であり、製品共通の新Status列挙や万能な合否点数ではない。Sourceのバイトが一致してもOwnerの意味受理を示さず、実行結果が同値でも実際に選択した実装が一致したことにはならない。部分取得、Read failure、制限付き情報は秘密側の存在・件数を公開DOM／ARIA／AI Contextへ漏らさない。

**最小検収**：CLI／studiohostのRule Inspectが一致し、誤Kind・別Workspaceの拒否分類が一致することを第一段とする。次にWails上の同じ対象について、Focus／Pinの後着失効、Sourceの実Open、read-only Diagramと原文の往復、診断から一意SourceへのJumpを段階的に追加する。Generate／Interpretの同じ意味・異方式比較は#2620／#1121、実Providerは#4495／#2586、実Stage2は#2831／#4583が担当し、Desktopの最初のread-only検証の追加Hard Gateとしない。独立Expectedは#1180／#2514の既存Packetへ接続する。

**現状**：上記は仕様と既存テストへの引渡しであり、#5115の組織C1抽象経路テスト、#5108のCLI／Studio Host同値、Plugin Mapping #5101の成果を参考にしたもの。新しいDesktop実IPC／Windows／編集保存／Provider実Call／Stage2の試験は本追補では実施していない。

## 19. 実装済みの読取契約から、Desktopの編集・生成へ進む（2026-11-16）

### 19.1 現在使える根拠と残る一巡

今回の作業基線は、実際のmainの製品コードにある次の三つの限定成果とする。第一に、実TEI CLIの別Process・直接Application Operation・Wails非依存のStudio Hostによる同一RuleのInspectを照合するGo回帰が存在し、未公開の異Kindと許可外Workspaceを拒否する。第二に、MarkdownのData表から選んだRuleについて、生成Goの関数名だけを変えた実行と、別Attributeを読んで結果が偶然一致する実行を区別する回帰がある（main統合済みPR #5132）。第三に、self-buildのL2-b判定には計画で選択したBuilderと起動直前に読んだファイルのdigest照合が組み込まれ、計画がGo toolchainを名指した場合は対応する照合も行う。ただしL2-bはOS Loaderが読んだ実Image、Builder内の再委譲、G4の意味受理、Desktop Stage1→Stage2を保証しない。

これらの存在は**現行mainに実装・試験コードがある**という確認であり、この文書変更のHEADでGo／Node／Windows実試験を実行したことではない。§15のD1〜D5と通常CLIの実Provider Readは、引き続き独立したP0である。

### 19.2 次に実装する四つの操作

| 作業単位 | 利用者の一操作と実装境界 | 最初の合格条件 |
| --- | --- | --- |
| **対象から原文へ** | Catalog／Inspectorで選んだqualified Subjectから、Hostが再解決したSource occurrenceをEditorへ渡す | 実byteのSource位置とBufferの文字位置を区別し、UTF-8／CRLF／IME条件で対象を誤らない。一意でない参照・未読取SourceではJumpを保留 |
| **FocusとPinの独立** | AをPinしたままBを操作し、AのInspectを明示的に再取得する | Aの後着結果からBのCaret・Undo・Dirty Buffer・Graph／ARIAを変えない。A→B→A、別Workspace同名Subjectで旧Replyを再利用しない |
| **候補編集から保存へ** | Source編集→Diff／Preview→既存Writerの保存条件照合→再読込→Meaning Reviewを別操作にする | 保存時の実Source byte／Root／Revision競合を拒否し、Bufferを保全する。保存成功や復元DraftだけでMeaningを自動受理しない |
| **同じ意味から実現先へ** | Ruleの型付き意味とSource位置から、Interpret／Generate／Delegateの選択と実読取を確認する | 表示Symbol変更は許容し、同じ結果でも別Attribute読取・別Provider／Builder実起動を検出。選択・実消費・実観測を分離する |

この表は新しいOperation API、UI専用StateMachine、Test DSL、Trace Storeの採用を求めない。最初の製品コード差分は**read-onlyの「対象から原文へ」**に絞り、既存のInspect／Source Index／Source Location Presenter／Editorを接続する。保存と逆編集は既存Source Ownerが実byteの条件付き更新を提供した範囲で段階的に追加する。

### 19.3 独立Expectedと製品画面の検収を分ける

最初の検証Packetは、実Rule Markdown一件と、同名異Kind・非TEIのFence・別Workspace、元Sourceの一文字変更、UTF-8文字を含む原文、CRLF原文を用意する。CLI・Operation・Studio Hostの照合結果を基準に、実Wails WebView2のInspector→Source位置表示→EditorのCaretへの受領を**同じSource Revisionと実Asset**で確認する。EditorにSource Openが未実装なら、その工程を未実施と記録し、単なるInspector表示を往復のPASSにしない。

追加でFocus／Pinの後着とBuffer未保存、RuleのGenerate／Interpretで**結果は同じだが実際の読取先が異なる**負例、Builder Bを選択して実ファイルAを起動する負例を、既存#1180／#2514の独立Expectedに接続する。試験で使うFunction名、Canvas座標、CSS classを意味IDへ昇格させない。情報の開示可否はHostが現在判断し、非公開のSource・件数・失敗詳細をDOM／ARIA／AIへ漏らさない。

### 19.4 実装の引渡しと停止線

最小の実装・検収順は次のとおりとする。

1. **read-only縦断**：#1158／#1824／#4515がCLI・Hostの既存同値を足場にInspector→一意Source位置→Editorの実受領を実装する。#1180／#2514で負例を固定する。
2. **保存・再資格化**：#1529／#1124／Source Ownerが一SourceのBuffer、CAS、再Load、Meaning受理を区別する。複数Window／Worktreeの競合とSource A→B→Aを含める。
3. **表現型の拡張**：#2497／#951／#1223で同じSubjectを表・Graph／Node・Plugin Portに投影し、Whole／Field／Adapterの型付きMappingと情報損失をread-onlyで確認する。
4. **独立した製品P0**：#2831／#4583が非Plugin Studio BuildのD1→D5を、#2586／#4495が通常CLI→実Provider→独立Consumerを進める。上記UI作業をどちらの新しい先行Hard Gateにも設定しない。

Web Components／Lit、Go／WASM、子Backend、Updaterは既存TypeScript／Wails基線の実測で明確な利益が出たときだけ別途採用する。本§19は既存契約と現行mainの実装到達点を結ぶ計画であり、新規製品E2E／Ownerの業務受理を実施したものではない。

## 20. Source位置の表示と原文への移動を分けて実証する（2026-11-16、実装引渡し）

### 20.1 位置を知ることと、原文を開けることは違う

§19の「対象から原文へ」の一巡を、**現在のWorkbenchで実装できる表示**と**Hostによる実Source再読取**に分ける。現行`reference/tei-studio-workbench/source-locations.ts`の`readSourceLocations`は、既存の`readInspection`で確かめた対象とCatalogに対し、返されたSource Indexの相対Path、Role、byte位置、Artifact Fingerprintを最大20件まで投影する。観測した参照件数と省略件数を返すが、`completenessEstablished`は常に`false`である。

この関数は原文のbyteを読まず、Workspace Rootの解決、ファイルを開く処理、Source Disclosureの判断を行わない。相対Pathを`file://` URLや保存先、権限として扱わない。索引上のbyte offsetをEditorのCaretへそのまま渡さない。UTF-8の複数byte文字、CRLF、IME確定前のBufferでは、原文と画面の文字位置が異なり得る。

| 段階 | 再利用する仕組みと最小変更 | 合格の範囲・停止条件 |
| --- | --- | --- |
| **SL-0：対象の確認** | Catalog／`inspectEligibility`／`readInspection`と既存のFocus・Pin別応答laneを使う | Workspace・qualified Subject・要求世代の合う観測だけを採用。別WorkspaceやA→B→Aの旧Replyを現在の結果にしない |
| **SL-1：位置の表示** | `readSourceLocations`の検証結果を安全なread-only ViewModelとしてInspectorへ渡す | 相対Path・Role・byte位置・Fingerprint・観測件数・省略件数・取得限界を表示。20件を全件としない。不正な索引を正常な0件にしない |
| **SL-2：実Sourceの再取得** | 利用者の明示Open要求を、既存Source Owner／Hostの適格なReadへ委ね、Root・実byte・Fingerprint・Occurrenceを再検証する | 適格な実Read経路がなければ「開く操作は未対応」と表示。元Source改版・候補の重複・Root不一致ではOpen／Jumpしない |
| **SL-3：Editorへの移動** | 実際に検証した原文byteからBuffer用の文字範囲を求め、既存Editorのread-only DocumentとCaretに渡す | 同じSource版・実Asset・実Consumerで本文とCaretを検査。IME・Undo・Pin／Focus・Graph／ARIAを旧応答で変えない |

**最初の製品コード差分はSL-1のみ**に絞る。横断Web設計のWEB-P1aと同じ、純粋なPresenterの仕事として実装する。新しいDesktop専用Parser／Locator／State Store／権限Engineを作らない。SL-2の実Read入口が既存Operationに不足するなら、Source Ownerへ必要条件を戻し、UIから任意Pathを直接読む近道を採らない。編集Bufferの保存／CAS／Meaning Reviewは§16・§19の後続検収である。

### 20.2 取得状態と実現方式を、そのまま診断へ運ぶ

main統合済み#5142の`validationeval.Decide`は、値の`present`・`null`・`not-fetched`、関係の`complete`・`partial`を区別し、成立・不成立・判定保留と読み手の矛盾を分ける。Desktopは**0件、NULL、未取得、部分取得、Readerエラー、権限上の非開示**を同じ「空」にしない。部分取得でも重複する行が観測されれば違反を確定できるが、重複が見つからないことだけで全体の成立は証明できない。

main統合済み#5166では、暦日`not-after` Ruleを直接解釈（P）できても、現行Go生成（G）は未対応である。Studioは「Sourceを読んだ」「意味を検証した」「実現方式を選択した」「実際に呼び出した」「Consumerに届いた」を別の事実として表示する。G未対応を暗黙のP成功へ置き換えず、結果が一致しても別Attributeの実Readや別Builderを見落とさない。

PluginのWhole／Field／明示Adapter（main済み#5162）は標準型に加えてPresence・Coverage・Read ID／Scopeを対応付け、Node Viewではまずread-onlyで表示する。組織C1の同一Attempt実PostgreSQL Read（main済み#5155／#5165）は実Providerまでの限定的な先例であり、一般Function Context・Desktop二Consumerの受領証明には転用しない。

### 20.3 独立ExpectedとOwnerへの引渡し

| 検収候補 | 期待する結果 |
| --- | --- |
| **SL-E01** 正しいRuleと一意なDefinition Ref | 位置・Role・Fingerprintを表示。実Source Openは未実施 |
| **SL-E02** 参照21件以上 | 表示20件・省略数・全件性未確定を別表示 |
| **SL-E03** 絶対Path／親Path／異Subject／Artifact不一致 | 既存位置Readerが拒否。正常な0件やFile URLへ変換しない |
| **SL-E04** Workspace切替／A→B→A／Pin A再読取中のB編集 | 古いAを新しいAと取り違えず、BのCaret・IME・Dirty Buffer・Graph／ARIAを保持 |
| **SL-E05** Source改版・別Worktreeの同名対象・複数Occurrence | HostでRootと実byteを再照合するまでOpen／Jumpしない |
| **SL-E06** 同じ値だが別Attributeを読んだ実行・G未対応Rule | 実Readの違いを示し、生成未対応を明示。無言のFallbackをしない |
| **SL-E07** NULL・未取得・部分取得・Reader失敗 | Coverageと判定理由を分け、非公開の値・件数をDOM／ARIAへ漏らさない |

これらは**既存#1180／#2514の独立Expectedへ渡す候補**であり、別のExpected正本は作らない。実装順はSL-1（WEB-P1aと共通）→Focus／Pin後着の独立試験（未統合PR #5168と実装を重複させない）→SL-2のHost実Read→SL-3のEditor／実Windows受領とする。`package.json`のテスト登録だけで生成Desktopの固定Source Closureを失効させる場合は、固定値を都合よく書き換えず、既存CIの受入経路に従い別資格で検査する。

SL-0/1/3は#1824／#4515／#997、SL-2と条件付きSource保存は#1158／#1529／#1124、Plugin投影は#2497／#951が担当する。正式な非Plugin Studio Stage1→Stage2（#2831／#4583）と通常CLI実Provider Read（#2586／#4495）は**独立P0を維持**し、この表示検収を新しい先行Hard Gateとしない。本節は設計統合のみで、新しいGo／TypeScript／Windows／DB試験やOwner意味受理を実施していない。

## 21. Source Indexの表示用Presenterを最小の製品差分として追加する（2026-11-17）

§20のSL-1を、さらに**SL-1a：検証済みSource位置から表示用の値を作ること**と**SL-1b：実WorkbenchのInspector DOMで受け取ること**に分ける。main統合済み#5173はInspect結果からEditor表示文書などを純粋に導出する`inspector-presentation.ts`を追加した。ただし本番`main.ts`に未接続である。main統合済み#5168はWorkspace A→B→AやPin・後着応答をNode試験で確認し、#5188は実CLI ProcessとStudio Hostの意味／Sourceを独立Expectedで比較した。いずれもWindowsの新しいSource位置表示や、ファイルを開く権限の成立を示さない。

今回の**SL-1a実装**として、既存`readSourceLocations`の検証結果だけを受け取る純粋な`source-locations-presentation.ts`を新設する。Source Status、観測件数／表示件数／省略件数、全件性未確定、相対Path・Role・行列および元Sourceのbyte offset・Artifact指紋を、**HTMLではなく表示用の文字列**として渡す。EditorのUTF-16位置を計算せず、`file://`／Openボタン／Grant／Effect／実Source byteを生成しない。これにより§20のSL-E01〜03、E07のうち**表示変換の局所部分のみ**を検査する。既存S-LOC Readerの内容を再解析せず、同じHost／Catalogと応答レーンで既に検査した結果を前提とする。

| 受入単位 | 実際に確認すること | 今回の資格 |
| --- | --- | --- |
| **SL-1a** | 一件・23件・0件、indexed／not-indexed、危険な見かけのPath、旧Source指紋・異Subject・write応答の拒否。元byte位置と全件性未確認の区別 | 新純粋PresenterとNode試験。Host／DOM／実ファイルは未使用 |
| **SL-1b** | 既存`main.ts`の選択／Pin laneが受け取った現行Inspect結果を、Source位置ViewModel→Inspectorの`textContent`で描画。旧応答は表示しない | **未実装**。既存Source Closureの改版と正規Bundle・実WebView2の再受入が必要 |
| **SL-2** | 明示OpenのHost Operationが現在のRoot／byte／Occurrenceを再確認 | **未実装**。索引上の相対Pathだけでは開かない |
| **SL-3** | 実原文のbyte位置→Editor文字位置の変換、Caret／IME／Dirty BufferとWindows受領 | **未実装**。Source位置一覧が表示できても合格としない |

**再現とCI**：既存Workbenchの`package.json`は、生成Desktopの過去のSource Closure固定に含まれるため、テストの登録だけを目的に改版しない。新規Node試験は既存`tei-studio-async-lanes-ci.yml`の独立Jobへ追加し、Node24で実行する。新Presenterを本番`main.ts`からまだimportしないため、現在の配布JS／HTML／CSS・第三者通知のbyteを変えない。将来のSL-1b接続で従来の固定成果物の指紋を意図的に変える場合は、旧資格を上書きするのではなく、新しいSource Closureと現行Toolchain／Embed／実Consumerを同一HEADで認定する。

**他レーンへの接続**：#5198の`-expected-fingerprint`を使うCLIの前提指紋・再読取は未統合候補であり、HostでのSL-2を代行するものではない。#5205の`*.tei.md` Reader診断は選択可能な言語を示すようになったが、Sourceを安全にOpenできる意味ではない。#5194の入れ子Go module被覆と#5202の状態境界は未統合候補として扱い、これらの未達をStudio P0の完了へ加算しない。原文未読取・部分取得・Rule未判定・生成方式未対応をUIで別状態として保持する。#2831／#4583と#2586／#4495の独立P0を維持し、新Host／新Parser／第二Test Engineは追加しない。
